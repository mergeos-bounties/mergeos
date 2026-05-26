package core

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"sort"
	"time"
)

const defaultSSLPort = "443"

func (s *Store) StartSSLReviewJob(ctx context.Context) {
	if !s.cfg.SSLReviewEnabled || len(s.cfg.SSLReviewDomains) == 0 {
		return
	}
	interval := sslReviewInterval(s.cfg)
	go func() {
		_, _ = s.ReviewSSLNow(ctx, "startup")
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_, _ = s.ReviewSSLNow(ctx, "scheduled")
			}
		}
	}()
}

func (s *Store) ReviewSSLNow(ctx context.Context, checkedBy string) ([]*SSLReviewStatus, error) {
	if !s.cfg.SSLReviewEnabled {
		return s.ListSSLReviews(), nil
	}
	rows := make([]*SSLReviewStatus, 0, len(s.cfg.SSLReviewDomains))
	for _, domain := range s.cfg.SSLReviewDomains {
		rows = append(rows, reviewSSLDomain(ctx, s.cfg, domain, checkedBy))
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, row := range rows {
		s.sslReviews[row.Domain] = cloneSSLReview(row)
	}
	if err := s.saveLocked(); err != nil {
		return rows, err
	}
	return rows, nil
}

func (s *Store) ListSSLReviews() []*SSLReviewStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sslReviewRowsLocked()
}

func (s *Store) sslReviewRowsLocked() []*SSLReviewStatus {
	seen := map[string]bool{}
	rows := []*SSLReviewStatus{}
	for _, domain := range s.cfg.SSLReviewDomains {
		if domain == "" || seen[domain] {
			continue
		}
		seen[domain] = true
		if review, ok := s.sslReviews[domain]; ok {
			rows = append(rows, cloneSSLReview(review))
			continue
		}
		rows = append(rows, pendingSSLReview(s.cfg, domain))
	}

	extras := []string{}
	for domain := range s.sslReviews {
		if !seen[domain] {
			extras = append(extras, domain)
		}
	}
	sort.Strings(extras)
	for _, domain := range extras {
		rows = append(rows, cloneSSLReview(s.sslReviews[domain]))
	}
	return rows
}

func reviewSSLDomain(ctx context.Context, cfg Config, domain, checkedBy string) *SSLReviewStatus {
	domain = cleanDomain(domain)
	now := time.Now().UTC()
	next := now.Add(sslReviewInterval(cfg))
	row := &SSLReviewStatus{
		Domain:        domain,
		Port:          defaultSSLPort,
		Status:        "pending",
		LastCheckedAt: &now,
		NextCheckAt:   &next,
		CheckedBy:     checkedBy,
	}
	if domain == "" {
		row.Status = "error"
		row.Error = "domain is empty"
		return row
	}

	checkCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	dialer := tls.Dialer{
		NetDialer: &net.Dialer{Timeout: 10 * time.Second},
		Config: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			ServerName:         domain,
			InsecureSkipVerify: true, //nolint:gosec -- the monitor verifies below after inspecting the cert.
		},
	}
	conn, err := dialer.DialContext(checkCtx, "tcp", net.JoinHostPort(domain, defaultSSLPort))
	if err != nil {
		row.Status = "error"
		row.Error = err.Error()
		return row
	}
	defer conn.Close()

	tlsConn, ok := conn.(*tls.Conn)
	if !ok {
		row.Status = "error"
		row.Error = "connection did not negotiate TLS"
		return row
	}
	state := tlsConn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		row.Status = "error"
		row.Error = "server returned no certificate"
		return row
	}
	cert := state.PeerCertificates[0]
	intermediates := x509.NewCertPool()
	for _, intermediate := range state.PeerCertificates[1:] {
		intermediates.AddCert(intermediate)
	}
	_, verifyErr := cert.Verify(x509.VerifyOptions{
		DNSName:       domain,
		CurrentTime:   now,
		Intermediates: intermediates,
	})
	notBefore := cert.NotBefore.UTC()
	notAfter := cert.NotAfter.UTC()
	row.Subject = cert.Subject.CommonName
	row.Issuer = cert.Issuer.CommonName
	if row.Issuer == "" {
		row.Issuer = cert.Issuer.String()
	}
	row.SerialNumber = cert.SerialNumber.String()
	row.DNSNames = append([]string{}, cert.DNSNames...)
	row.NotBefore = &notBefore
	row.NotAfter = &notAfter
	row.DaysRemaining = int(notAfter.Sub(now).Hours() / 24)
	row.Status = "ok"
	if now.After(notAfter) {
		row.Status = "expired"
	} else if now.Before(notBefore) {
		row.Status = "error"
		row.Error = "certificate is not valid yet"
	} else if verifyErr != nil {
		row.Status = "error"
		row.Error = verifyErr.Error()
	} else if row.DaysRemaining <= int(cfg.SSLExpiryWarnDays) {
		row.Status = "warning"
	}
	return row
}

func pendingSSLReview(cfg Config, domain string) *SSLReviewStatus {
	now := time.Now().UTC()
	next := now.Add(sslReviewInterval(cfg))
	return &SSLReviewStatus{
		Domain:      domain,
		Port:        defaultSSLPort,
		Status:      "pending",
		NextCheckAt: &next,
	}
}

func sslReviewInterval(cfg Config) time.Duration {
	minutes := cfg.SSLReviewIntervalMinutes
	if minutes <= 0 {
		minutes = 360
	}
	return time.Duration(minutes) * time.Minute
}

func cloneSSLReview(review *SSLReviewStatus) *SSLReviewStatus {
	if review == nil {
		return nil
	}
	copyReview := *review
	copyReview.DNSNames = append([]string{}, review.DNSNames...)
	return &copyReview
}
