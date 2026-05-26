package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

type postgresPersistence struct {
	db *sql.DB
}

func (p *postgresPersistence) Load(ctx context.Context) (*persistedState, error) {
	state := &persistedState{Users: []*User{}, Wallets: []*Wallet{}, Projects: []*Project{}, Tasks: []*Task{}, Sessions: []*Session{}, Notifications: []*Notification{}, Attachments: []*Attachment{}, SSLReviews: []*SSLReviewStatus{}, Ledger: []LedgerEntry{}}

	if err := p.loadStoreMeta(ctx, state); err != nil {
		return nil, err
	}
	if err := p.loadUsers(ctx, state); err != nil {
		return nil, err
	}
	if err := p.loadWallets(ctx, state); err != nil {
		return nil, err
	}
	projects, err := p.loadProjects(ctx, state)
	if err != nil {
		return nil, err
	}
	if err := p.loadTasks(ctx, state, projects); err != nil {
		return nil, err
	}
	if err := p.loadSessions(ctx, state); err != nil {
		return nil, err
	}
	if err := p.loadNotifications(ctx, state); err != nil {
		return nil, err
	}
	if err := p.loadAttachments(ctx, state, projects); err != nil {
		return nil, err
	}
	if err := p.loadSSLReviews(ctx, state); err != nil {
		return nil, err
	}
	if err := p.loadLedger(ctx, state); err != nil {
		return nil, err
	}
	return state, nil
}

func (p *postgresPersistence) loadStoreMeta(ctx context.Context, state *persistedState) error {
	row := p.db.QueryRowContext(ctx, "SELECT key, value, updated_at FROM store_meta WHERE key = 'next_id'")
	var key, value string, updatedAt time.Time
	if err := row.Scan(&key, &value, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			state.NextID = 1
			return nil
		}
		return err
	}
	state.NextID, _ = strconv.Atoi(value)
	if state.NextID < 1 {
		state.NextID = 1
	}
	return nil
}

func (p *postgresPersistence) loadUsers(ctx context.Context, state *persistedState) error {
	rows, err := p.db.QueryContext(ctx, "SELECT id, name, company_name, email, role, password_salt, password_hash, wallet_address, github_id, github_username, github_avatar_url, identity_providers, created_at, last_login_at FROM users ORDER BY created_at, id")
	if err != nil {
		return fmt.Errorf("load users: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		u := &User{}
		var ghID sql.NullInt64
		var ghUser, ghAvatar, identityProviders sql.NullString
		var lastLoginAt sql.NullTime
		if err := rows.Scan(&u.ID, &u.Name, &u.CompanyName, &u.Email, &u.Role, &u.PasswordSalt, &u.PasswordHash, &u.WalletAddress, &ghID, &ghUser, &ghAvatar, &identityProviders, &u.CreatedAt, &lastLoginAt); err != nil {
			return fmt.Errorf("scan user: %w", err)
		}
		u.GitHubID = int(ghID.Int64)
		u.GitHubUsername = ghUser.String
		u.GitHubAvatarURL = ghAvatar.String
		if identityProviders.Valid && identityProviders.String != "" {
			var providers map[string]string
			if err := json.Unmarshal([]byte(identityProviders.String), &providers); err == nil {
				u.IdentityProviders = providers
			}
		}
		u.LastLoginAt = timePtr(lastLoginAt)
		state.Users = append(state.Users, u)
	}
	return rows.Err()
}

func (p *postgresPersistence) loadWallets(ctx context.Context, state *persistedState) error {
	rows, err := p.db.QueryContext(ctx, "SELECT address, owner_user_id, github_id, github_username, recovery_salt, recovery_hash, created_at, linked_at FROM wallets ORDER BY created_at")
	if err != nil {
		return fmt.Errorf("load wallets: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		w := &Wallet{}
		var ghID sql.NullInt64
		var ghUser sql.NullString
		if err := rows.Scan(&w.Address, &w.OwnerUserID, &ghID, &ghUser, &w.RecoverySalt, &w.RecoveryHash, &w.CreatedAt, &w.LinkedAt); err != nil {
			return fmt.Errorf("scan wallet: %w", err)
		}
		w.GitHubID = int(ghID.Int64)
		w.GitHubUsername = ghUser.String
		state.Wallets = append(state.Wallets, w)
	}
	return rows.Err()
}

func (p *postgresPersistence) loadProjects(ctx context.Context, state *persistedState) (map[string]*Project, error) {
	projects := map[string]*Project{}
	rows, err := p.db.QueryContext(ctx, "SELECT id, client_user_id, title, client_name, company_name, client_email, phone, site_type, package_tier, timeline, brief, payment_method, payment_status, payment_provider, payment_reference, bounty_repo_name, repo_visibility, repo_provider, repo_url, repo_local_path, budget_cents, fee_cents, work_pool_cents, status, created_at FROM projects ORDER BY created_at, id")
	if err != nil {
		return nil, fmt.Errorf("load projects: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		p := &Project{}
		if err := rows.Scan(&p.ID, &p.ClientUserID, &p.Title, &p.ClientName, &p.CompanyName, &p.ClientEmail, &p.Phone, &p.SiteType, &p.PackageTier, &p.Timeline, &p.Brief, &p.PaymentMethod, &p.PaymentStatus, &p.PaymentProvider, &p.PaymentReference, &p.BountyRepoName, &p.RepoVisibility, &p.RepoProvider, &p.RepoURL, &p.RepoLocalPath, &p.BudgetCents, &p.FeeCents, &p.WorkPoolCents, &p.Status, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		projects[p.ID] = p
		state.Projects = append(state.Projects, p)
	}
	return projects, rows.Err()
}

func saveUsers(ctx context.Context, tx *sql.Tx, users []*User) error {
	for _, user := range users {
		if user == nil {
			continue
		}
		providersJSON := "{}"
		if len(user.IdentityProviders) > 0 {
			b, err := json.Marshal(user.IdentityProviders)
			if err == nil {
				providersJSON = string(b)
			}
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO users (id, name, company_name, email, role, password_salt, password_hash, wallet_address, github_id, github_username, github_avatar_url, identity_providers, created_at, last_login_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)",
			user.ID, user.Name, user.CompanyName, user.Email, user.Role, user.PasswordSalt, user.PasswordHash,
			normalizeWalletAddress(user.WalletAddress),
			int64(user.GitHubID), normalizeGitHubUsername(user.GitHubUsername), user.GitHubAvatarURL,
			providersJSON, user.CreatedAt, user.LastLoginAt,
		); err != nil {
			return fmt.Errorf("save user %s: %w", user.ID, err)
		}
	}
	return nil
}

func saveWallets(ctx context.Context, tx *sql.Tx, wallets []*Wallet) error {
	for _, wallet := range wallets {
		if wallet == nil {
			continue
		}
		address := normalizeWalletAddress(wallet.Address)
		if !validWalletAddress(address) {
			continue
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO wallets (address, owner_user_id, github_id, github_username, recovery_salt, recovery_hash, created_at, linked_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)",
			address, wallet.OwnerUserID, int64(wallet.GitHubID), normalizeGitHubUsername(wallet.GitHubUsername),
			wallet.RecoverySalt, wallet.RecoveryHash, wallet.CreatedAt, wallet.LinkedAt,
		); err != nil {
			return fmt.Errorf("save wallet %s: %w", wallet.Address, err)
		}
	}
	return nil
}

func saveProjects(ctx context.Context, tx *sql.Tx, projects []*Project) error {
	for _, project := range projects {
		if project == nil {
			continue
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO projects (id, client_user_id, title, client_name, company_name, client_email, phone, site_type, package_tier, timeline, brief, payment_method, payment_status, payment_provider, payment_reference, bounty_repo_name, repo_visibility, repo_provider, repo_url, repo_local_path, budget_cents, fee_cents, work_pool_cents, status, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25)",
			project.ID, project.ClientUserID, project.Title, project.ClientName, project.CompanyName, project.ClientEmail,
			project.Phone, project.SiteType, project.PackageTier, project.Timeline, project.Brief, project.PaymentMethod,
			project.PaymentStatus, project.PaymentProvider, project.PaymentReference, project.BountyRepoName,
			project.RepoVisibility, project.RepoProvider, project.RepoURL, project.RepoLocalPath, project.BudgetCents,
			project.FeeCents, project.WorkPoolCents, project.Status, project.CreatedAt,
		); err != nil {
			return fmt.Errorf("save project %s: %w", project.ID, err)
		}
	}
	return nil
}

func saveTasks(ctx context.Context, tx *sql.Tx, tasks []*Task) error {
	for _, task := range tasks {
		if task == nil {
			continue
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO tasks (id, project_id, issue_number, title, acceptance, reward_cents, required_worker_kind, suggested_agent_type, bounty_type, status, worker_kind, worker_id, agent_type, proof_hash, issue_url, created_at, accepted_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)",
			task.ID, task.ProjectID, task.IssueNumber, task.Title, task.Acceptance, task.RewardCents, task.RequiredWorkerKind,
			task.SuggestedAgentType, task.BountyType, task.Status, task.WorkerKind, task.WorkerID, task.AgentType, task.ProofHash,
			task.IssueURL, task.CreatedAt, task.AcceptedAt,
		); err != nil {
			return fmt.Errorf("save task %s: %w", task.ID, err)
		}
	}
	return nil
}

func saveSessions(ctx context.Context, tx *sql.Tx, sessions []*Session) error {
	now := time.Now().UTC()
	for _, session := range sessions {
		if session == nil || !now.Before(session.ExpiresAt) {
			continue
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO sessions (token, user_id, created_at, expires_at) VALUES ($1,$2,$3,$4)",
			session.Token, session.UserID, session.CreatedAt, session.ExpiresAt,
		); err != nil {
			return fmt.Errorf("save session for user %s: %w", session.UserID, err)
		}
	}
	return nil
}

func saveNotifications(ctx context.Context, tx *sql.Tx, notifications []*Notification) error {
	for _, notification := range notifications {
		if notification == nil {
			continue
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO notifications (id, user_id, project_id, channel, subject, body, status, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)",
			notification.ID, notification.UserID, notification.ProjectID, notification.Channel,
			notification.Subject, notification.Body, notification.Status, notification.CreatedAt,
		); err != nil {
			return fmt.Errorf("save notification %s: %w", notification.ID, err)
		}
	}
	return nil
}

func saveAttachments(ctx context.Context, tx *sql.Tx, attachments []*Attachment) error {
	for _, attachment := range attachments {
		if attachment == nil {
			continue
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO attachments (id, user_id, project_id, original_name, stored_name, content_type, size_bytes, url, stored_path, is_image, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)",
			attachment.ID, attachment.UserID, attachment.ProjectID, attachment.OriginalName, attachment.StoredName,
			attachment.ContentType, attachment.SizeBytes, attachment.URL, attachment.StoredPath, attachment.IsImage,
			attachment.CreatedAt,
		); err != nil {
			return fmt.Errorf("save attachment %s: %w", attachment.ID, err)
		}
	}
	return nil
}

func saveSSLReviews(ctx context.Context, tx *sql.Tx, reviews []*SSLReviewStatus) error {
	for _, review := range reviews {
		if review == nil || review.Domain == "" {
			continue
		}
		dnsNames, err := json.Marshal(review.DNSNames)
		if err != nil {
			return fmt.Errorf("encode ssl dns names for %s: %w", review.Domain, err)
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO ssl_reviews (domain, port, status, issuer, subject, serial_number, dns_names, not_before, not_after, days_remaining, last_checked_at, next_check_at, error, checked_by) VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8,$9,$10,$11,$12,$13,$14)",
			review.Domain, review.Port, review.Status, review.Issuer, review.Subject, review.SerialNumber,
			string(dnsNames), review.NotBefore, review.NotAfter, review.DaysRemaining, review.LastCheckedAt,
			review.NextCheckAt, review.Error, review.CheckedBy,
		); err != nil {
			return fmt.Errorf("save ssl review %s: %w", review.Domain, err)
		}
	}
	return nil
}

func saveLedger(ctx context.Context, tx *sql.Tx, ledger []LedgerEntry) error {
	for _, entry := range ledger {
		if _, err := tx.ExecContext(ctx, "INSERT INTO ledger_entries (sequence, type, from_account, to_account, amount_cents, reference, previous_hash, entry_hash, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)",
			entry.Sequence, entry.Type, entry.FromAccount, entry.ToAccount, entry.AmountCents, entry.Reference,
			entry.PreviousHash, entry.EntryHash, entry.CreatedAt,
		); err != nil {
			return fmt.Errorf("save ledger entry %d: %w", entry.Sequence, err)
		}
	}
	return nil
}

func timePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time
	return &t
}