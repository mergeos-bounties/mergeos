package core

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

const walletAddressBytes = 20

func (s *Store) CreateGuestWallet(_ CreateWalletRequest) (*CreateWalletResponse, error) {
	recoveryCode, err := newWalletRecoveryCode()
	if err != nil {
		return nil, err
	}
	recoverySalt, recoveryHash, err := hashPassword(recoveryCode)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	wallet, err := s.createWalletLocked("", recoverySalt, recoveryHash)
	if err != nil {
		return nil, err
	}
	summary := s.walletSummaryLocked(wallet)
	if err := s.saveLocked(); err != nil {
		return nil, err
	}

	return &CreateWalletResponse{
		Address:      wallet.Address,
		RecoveryCode: recoveryCode,
		Wallet:       summary,
	}, nil
}

func (s *Store) WalletSummary(address string) (WalletSummary, bool) {
	address = normalizeWalletAddress(address)
	if !validWalletAddress(address) {
		return WalletSummary{}, false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	wallet, ok := s.wallets[address]
	if !ok {
		return WalletSummary{}, false
	}
	return s.walletSummaryLocked(wallet), true
}

func (s *Store) LinkWalletToUser(userID string, req LinkWalletRequest) (PublicUser, error) {
	address := normalizeWalletAddress(req.Address)
	if !validWalletAddress(address) {
		return PublicUser{}, errors.New("wallet address is invalid")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[strings.TrimSpace(userID)]
	if !ok {
		return PublicUser{}, errors.New("user not found")
	}
	wallet, ok := s.wallets[address]
	if !ok {
		return PublicUser{}, errors.New("wallet not found")
	}
	if wallet.OwnerUserID != "" && wallet.OwnerUserID != user.ID {
		return PublicUser{}, errors.New("wallet is already linked to another account")
	}
	if wallet.OwnerUserID == "" && !verifyPassword(req.RecoveryCode, wallet.RecoverySalt, wallet.RecoveryHash) {
		return PublicUser{}, errors.New("wallet recovery code is invalid")
	}

	now := time.Now().UTC()
	wallet.OwnerUserID = user.ID
	wallet.LinkedAt = &now
	user.WalletAddress = wallet.Address
	if user.GitHubID != "" || user.GitHubUsername != "" {
		wallet.GitHubID = strings.TrimSpace(user.GitHubID)
		wallet.GitHubUsername = normalizeGitHubUsername(user.GitHubUsername)
	}
	if err := s.saveLocked(); err != nil {
		return PublicUser{}, err
	}
	return publicUser(user), nil
}

func (s *Store) AuthenticateGitHub(profile GitHubAuthProfile, walletAddress, walletRecoveryCode string) (*AuthResponse, error) {
	githubID := strings.TrimSpace(profile.ID)
	githubUsername := normalizeGitHubUsername(profile.Username)
	if githubID == "" || githubUsername == "" {
		return nil, errors.New("github profile is missing id or username")
	}

	email := strings.TrimSpace(profile.Email)
	if email != "" {
		normalized, err := normalizeEmail(email)
		if err != nil {
			return nil, err
		}
		email = normalized
	} else {
		email = fmt.Sprintf("github-%s@users.mergeos.local", githubID)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	user := s.userByGitHubLocked(githubID, githubUsername)
	if user == nil {
		user = s.userByEmailLocked(email)
	}

	now := time.Now().UTC()
	created := false
	if user == nil {
		name := strings.TrimSpace(profile.Name)
		if name == "" {
			name = githubUsername
		}
		user = &User{
			ID:              s.newID("usr"),
			Name:            name,
			CompanyName:     "",
			Email:           email,
			Role:            RoleClient,
			GitHubID:        githubID,
			GitHubUsername:  githubUsername,
			GitHubAvatarURL: strings.TrimSpace(profile.AvatarURL),
			CreatedAt:       now,
			LastLoginAt:     &now,
		}
		if s.cfg.AdminAutoPromote && !s.hasAdminLocked() && len(s.users) == 0 {
			user.Role = RoleAdmin
		}
		created = true
	} else {
		user.GitHubID = githubID
		user.GitHubUsername = githubUsername
		if strings.TrimSpace(profile.Name) != "" && strings.TrimSpace(user.Name) == "" {
			user.Name = strings.TrimSpace(profile.Name)
		}
		if strings.TrimSpace(profile.AvatarURL) != "" {
			user.GitHubAvatarURL = strings.TrimSpace(profile.AvatarURL)
		}
		user.LastLoginAt = &now
	}

	wallet, err := s.ensureWalletForUserLocked(user, walletAddress, walletRecoveryCode)
	if err != nil {
		return nil, err
	}
	wallet.GitHubID = githubID
	wallet.GitHubUsername = githubUsername
	if wallet.LinkedAt == nil {
		linkedAt := now
		wallet.LinkedAt = &linkedAt
	}
	if created {
		s.users[user.ID] = user
		s.addNotificationLocked(user.ID, "", "email", "GitHub connected", "Your GitHub account is linked to an MRG wallet for future rewards.", "logged:github-wallet")
	}

	token, err := newToken()
	if err != nil {
		return nil, err
	}
	s.sessions[token] = &Session{
		Token:     token,
		UserID:    user.ID,
		CreatedAt: now,
		ExpiresAt: now.Add(30 * 24 * time.Hour),
	}
	if err := s.saveLocked(); err != nil {
		return nil, err
	}
	return &AuthResponse{Token: token, User: publicUser(user)}, nil
}

func (s *Store) ensureWalletForUserLocked(user *User, requestedAddress, recoveryCode string) (*Wallet, error) {
	if user == nil {
		return nil, errors.New("user is required")
	}
	requestedAddress = normalizeWalletAddress(requestedAddress)
	if requestedAddress != "" && !validWalletAddress(requestedAddress) {
		return nil, errors.New("wallet address is invalid")
	}

	address := normalizeWalletAddress(user.WalletAddress)
	if address != "" && !validWalletAddress(address) {
		address = ""
	}
	if requestedAddress != "" {
		if existingUserWallet := normalizeWalletAddress(user.WalletAddress); existingUserWallet != "" && existingUserWallet != requestedAddress {
			if _, ok := s.wallets[existingUserWallet]; ok {
				return nil, errors.New("account already has an MRG wallet")
			}
		}
		address = requestedAddress
	}

	if address != "" {
		wallet, ok := s.wallets[address]
		if !ok {
			wallet = &Wallet{
				Address:   address,
				CreatedAt: time.Now().UTC(),
			}
			s.wallets[address] = wallet
		}
		if wallet.OwnerUserID != "" && wallet.OwnerUserID != user.ID {
			return nil, errors.New("wallet is already linked to another account")
		}
		if wallet.OwnerUserID == "" && wallet.RecoveryHash != "" && !verifyPassword(recoveryCode, wallet.RecoverySalt, wallet.RecoveryHash) {
			return nil, errors.New("wallet recovery code is invalid")
		}
		now := time.Now().UTC()
		wallet.OwnerUserID = user.ID
		if wallet.LinkedAt == nil {
			wallet.LinkedAt = &now
		}
		user.WalletAddress = wallet.Address
		return wallet, nil
	}

	wallet, err := s.createWalletLocked(user.ID, "", "")
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	wallet.LinkedAt = &now
	user.WalletAddress = wallet.Address
	return wallet, nil
}

func (s *Store) createWalletLocked(ownerUserID, recoverySalt, recoveryHash string) (*Wallet, error) {
	for attempts := 0; attempts < 8; attempts++ {
		address, err := newWalletAddress()
		if err != nil {
			return nil, err
		}
		if _, exists := s.wallets[address]; exists {
			continue
		}
		wallet := &Wallet{
			Address:      address,
			OwnerUserID:  strings.TrimSpace(ownerUserID),
			RecoverySalt: recoverySalt,
			RecoveryHash: recoveryHash,
			CreatedAt:    time.Now().UTC(),
		}
		s.wallets[address] = wallet
		return wallet, nil
	}
	return nil, errors.New("could not generate a unique wallet address")
}

func (s *Store) walletSummaryLocked(wallet *Wallet) WalletSummary {
	if wallet == nil {
		return WalletSummary{}
	}
	address := normalizeWalletAddress(wallet.Address)
	accounts := []string{walletAccount(address)}
	if username := normalizeGitHubUsername(wallet.GitHubUsername); username != "" {
		accounts = append(accounts, githubWorkerAccount(username))
	}
	accountSet := map[string]bool{
		legacyWalletAccount(address): true,
	}
	for _, account := range accounts {
		accountSet[account] = true
	}

	summary := WalletSummary{
		Address:        address,
		Account:        walletAccount(address),
		LinkedAccounts: accounts,
		GitHubUsername: normalizeGitHubUsername(wallet.GitHubUsername),
		OwnerLinked:    wallet.OwnerUserID != "",
		CreatedAt:      wallet.CreatedAt,
		LinkedAt:       wallet.LinkedAt,
	}
	for _, entry := range s.ledger {
		matched := false
		if accountSet[entry.ToAccount] {
			summary.ReceivedCents += entry.AmountCents
			matched = true
		}
		if accountSet[entry.FromAccount] {
			summary.SentCents += entry.AmountCents
			matched = true
		}
		if matched {
			summary.TransactionCount++
		}
	}
	summary.BalanceCents = summary.ReceivedCents - summary.SentCents
	return summary
}

func (s *Store) payoutAccountForWorkerLocked(workerID string) string {
	workerID = normalizeWorkerID(workerID)
	if workerID == "" {
		return ""
	}
	if address := normalizeWalletAddress(workerID); validWalletAddress(address) {
		return walletAccount(address)
	}
	if strings.HasPrefix(strings.ToLower(workerID), "wallet:") {
		address := normalizeWalletAddress(workerID[len("wallet:"):])
		if validWalletAddress(address) {
			return walletAccount(address)
		}
	}
	if username, ok := strings.CutPrefix(strings.ToLower(workerID), "github:"); ok {
		username = normalizeGitHubUsername(username)
		if wallet := s.walletByGitHubLocked(username); wallet != nil {
			return walletAccount(wallet.Address)
		}
		return githubWorkerAccount(username)
	}
	return "worker:" + workerID
}

func normalizeWorkerID(value string) string {
	value = strings.TrimSpace(value)
	if address := normalizeWalletAddress(value); validWalletAddress(address) {
		return walletAccount(address)
	}
	return value
}

func (s *Store) walletByGitHubLocked(username string) *Wallet {
	username = normalizeGitHubUsername(username)
	if username == "" {
		return nil
	}
	for _, wallet := range s.wallets {
		if normalizeGitHubUsername(wallet.GitHubUsername) == username {
			return wallet
		}
	}
	return nil
}

func (s *Store) userByGitHubLocked(githubID, username string) *User {
	githubID = strings.TrimSpace(githubID)
	username = normalizeGitHubUsername(username)
	for _, user := range s.users {
		if githubID != "" && strings.TrimSpace(user.GitHubID) == githubID {
			return user
		}
		if username != "" && normalizeGitHubUsername(user.GitHubUsername) == username {
			return user
		}
	}
	return nil
}

func newWalletAddress() (string, error) {
	bytes := make([]byte, walletAddressBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "0x" + hex.EncodeToString(bytes), nil
}

func newWalletRecoveryCode() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "mrg-" + hex.EncodeToString(bytes), nil
}

func normalizeWalletAddress(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, "wallet:")
	return value
}

func validWalletAddress(value string) bool {
	value = normalizeWalletAddress(value)
	if len(value) != 42 || !strings.HasPrefix(value, "0x") {
		return false
	}
	_, err := hex.DecodeString(value[2:])
	return err == nil
}

func normalizeGitHubUsername(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, "github:")
	value = strings.Trim(value, "/")
	return value
}

func walletAccount(address string) string {
	return normalizeWalletAddress(address)
}

func legacyWalletAccount(address string) string {
	return "wallet:" + normalizeWalletAddress(address)
}

func githubWorkerAccount(username string) string {
	return "github:" + normalizeGitHubUsername(username)
}
