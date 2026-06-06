package core

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"
)

const (
	walletAddressBytes       = 32
	walletChainSolana        = "solana"
	legacyWalletHashPrefix   = "mergeos:solana-wallet-migration:"
	legacyWalletHashV1Prefix = "mergeos:legacy-wallet:v1:"
	walletMigrationPDASeed   = "wallet-migration"
	solanaBase58Alphabet     = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	solanaAddressPrefix      = "solana:"
	solanaShortAddressPrefix = "sol:"
)

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

func (s *Store) CreateUserWallet(userID string, _ CreateWalletRequest) (*CreateWalletResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[strings.TrimSpace(userID)]
	if !ok {
		return nil, errors.New("user not found")
	}
	wallet, err := s.ensureWalletForUserLocked(user, "", "")
	if err != nil {
		return nil, err
	}
	summary := s.walletSummaryLocked(wallet)
	if err := s.saveLocked(); err != nil {
		return nil, err
	}

	return &CreateWalletResponse{
		Address: wallet.Address,
		Wallet:  summary,
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

func (s *Store) CreateWalletMigration(userID string, req CreateWalletMigrationRequest, cfg Config) (WalletMigrationResponse, error) {
	legacyChain, err := normalizeLegacyChain(req.LegacyChain)
	if err != nil {
		return WalletMigrationResponse{}, err
	}
	legacyAddress := normalizeLegacyWalletAddress(req.LegacyAddress)
	if !validLegacyWalletAddressForChain(legacyChain, legacyAddress) {
		return WalletMigrationResponse{}, fmt.Errorf("legacy_address must be a valid %s wallet address", legacyChain)
	}
	targetAddress := normalizeWalletAddress(req.SolanaWallet)
	if targetAddress != "" && !validWalletAddress(targetAddress) {
		return WalletMigrationResponse{}, errors.New("solana_wallet is invalid")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[strings.TrimSpace(userID)]
	if !ok {
		return WalletMigrationResponse{}, errors.New("user not found")
	}
	if targetAddress == "" {
		targetAddress = normalizeWalletAddress(user.WalletAddress)
	}
	if targetAddress != "" && !validWalletAddress(targetAddress) {
		targetAddress = ""
	}

	now := time.Now().UTC()
	var wallet *Wallet
	if targetAddress == "" {
		wallet, err = s.createWalletLocked(user.ID, "", "")
		if err != nil {
			return WalletMigrationResponse{}, err
		}
	} else {
		var ok bool
		wallet, ok = s.wallets[targetAddress]
		if !ok {
			wallet = &Wallet{
				Address:     targetAddress,
				Chain:       walletChainSolana,
				OwnerUserID: user.ID,
				CreatedAt:   now,
			}
			s.wallets[targetAddress] = wallet
		}
		if wallet.OwnerUserID != "" && wallet.OwnerUserID != user.ID {
			return WalletMigrationResponse{}, errors.New("solana_wallet is already linked to another account")
		}
		wallet.OwnerUserID = user.ID
	}
	wallet.Chain = walletChainSolana
	wallet.LegacyAddress = legacyAddress
	if wallet.CreatedAt.IsZero() {
		wallet.CreatedAt = now
	}
	if wallet.LinkedAt == nil {
		wallet.LinkedAt = &now
	}
	if user.GitHubID != "" || user.GitHubUsername != "" {
		wallet.GitHubID = strings.TrimSpace(user.GitHubID)
		wallet.GitHubUsername = normalizeGitHubUsername(user.GitHubUsername)
	}
	user.WalletAddress = wallet.Address

	summary := s.walletSummaryLocked(wallet)
	response := walletMigrationResponse(legacyChain, legacyAddress, summary, cfg, now)
	ledgerReference := walletMigrationLedgerReference(response)
	s.addLedger("wallet_migration", "legacy:"+legacyChain+":"+response.LegacyAddressHash, walletAccount(response.TargetAddress), 0, ledgerReference)
	s.addNotificationLocked(
		user.ID,
		"",
		"wallet",
		"Solana wallet migration staged",
		"Your legacy "+strings.ToUpper(legacyChain)+" wallet is linked to a Solana MRG wallet. Complete the Anchor registration proof before distribution.",
		"pending_contract_registration",
	)
	if err := s.saveLocked(); err != nil {
		return WalletMigrationResponse{}, err
	}
	return response, nil
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
		s.addNotificationLocked(user.ID, "", "email", "GitHub connected", "Your GitHub account is linked to a Solana MRG wallet for future rewards.", "logged:github-wallet")
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
				return nil, errors.New("account already has a Solana MRG wallet")
			}
		}
		address = requestedAddress
	}

	if address != "" {
		wallet, ok := s.wallets[address]
		if !ok {
			wallet = &Wallet{
				Address:   address,
				Chain:     walletChainSolana,
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
			Chain:        walletChainSolana,
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
	if legacyAddress := normalizeLegacyWalletAddress(wallet.LegacyAddress); legacyAddress != "" {
		accountSet[legacyAddress] = true
		accountSet[legacyWalletAccount(legacyAddress)] = true
	}
	for _, account := range accounts {
		accountSet[account] = true
	}

	summary := WalletSummary{
		Address:        address,
		Account:        walletAccount(address),
		Chain:          normalizedWalletChain(wallet.Chain),
		LegacyAddress:  normalizeLegacyWalletAddress(wallet.LegacyAddress),
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
	return base58Encode(bytes), nil
}

func newWalletRecoveryCode() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "mrg-" + hex.EncodeToString(bytes), nil
}

func normalizeWalletAddress(value string) string {
	value = strings.TrimSpace(value)
	value = trimAddressPrefix(value, "wallet:")
	value = trimAddressPrefix(value, solanaAddressPrefix)
	value = trimAddressPrefix(value, solanaShortAddressPrefix)
	return strings.TrimSpace(value)
}

func validWalletAddress(value string) bool {
	value = normalizeWalletAddress(value)
	if value == "" {
		return false
	}
	decoded, ok := base58Decode(value)
	return ok && len(decoded) == walletAddressBytes
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

func normalizedWalletChain(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return walletChainSolana
	}
	return value
}

func normalizeLegacyWalletAddress(value string) string {
	value = strings.TrimSpace(value)
	value = trimAddressPrefix(value, "wallet:")
	value = trimAddressPrefix(value, "tron:")
	value = trimAddressPrefix(value, "trc20:")
	value = trimAddressPrefix(value, "eip155:")
	if validLegacyEVMWalletAddress(value) {
		return strings.ToLower(value)
	}
	return strings.TrimSpace(value)
}

func normalizeLegacyChain(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "trc20", "tron":
		return "trc20", nil
	case "evm", "ethereum":
		return "evm", nil
	default:
		return "", errors.New("legacy_chain must be trc20 or evm")
	}
}

func validLegacyWalletAddressForChain(chain, address string) bool {
	switch chain {
	case "trc20":
		return validLegacyTronWalletAddress(address)
	case "evm":
		return validLegacyEVMWalletAddress(address)
	default:
		return false
	}
}

func validLegacyWalletAddress(value string) bool {
	value = normalizeLegacyWalletAddress(value)
	return validLegacyEVMWalletAddress(value) || validLegacyTronWalletAddress(value)
}

func validLegacyEVMWalletAddress(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != 42 || !strings.HasPrefix(strings.ToLower(value), "0x") {
		return false
	}
	_, err := hex.DecodeString(value[2:])
	return err == nil
}

func validLegacyTronWalletAddress(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != 34 || !strings.HasPrefix(value, "T") {
		return false
	}
	decoded, ok := base58Decode(value)
	return ok && len(decoded) == 25
}

func solanaWalletFromLegacy(value string) string {
	legacyAddress := normalizeLegacyWalletAddress(value)
	if !validLegacyWalletAddress(legacyAddress) {
		return ""
	}
	sum := sha256.Sum256([]byte(legacyWalletHashPrefix + legacyAddress))
	return base58Encode(sum[:])
}

func legacyWalletAddressHashHex(chain, address string) string {
	normalizedChain, err := normalizeLegacyChain(chain)
	if err != nil {
		return ""
	}
	normalizedAddress := strings.ToLower(normalizeLegacyWalletAddress(address))
	if normalizedAddress == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(legacyWalletHashV1Prefix + normalizedChain + ":" + normalizedAddress))
	return hex.EncodeToString(sum[:])
}

func walletMigrationResponse(chain, legacyAddress string, wallet WalletSummary, cfg Config, now time.Time) WalletMigrationResponse {
	legacyHash := legacyWalletAddressHashHex(chain, legacyAddress)
	programID := normalizeWalletAddress(cfg.SolanaProgramID)
	programReady := programID != "" && validWalletAddress(programID)
	tokenSymbol := strings.TrimSpace(cfg.TokenSymbol)
	if tokenSymbol == "" {
		tokenSymbol = defaultTokenSymbol
	}
	return WalletMigrationResponse{
		ProtocolVersion:   "mergeos.wallet-migration.v1",
		Kind:              "wallet_migration",
		MigrationID:       "wmg_" + legacyHash[:16],
		Status:            "pending_contract_registration",
		LegacyChain:       chain,
		LegacyAddress:     legacyAddress,
		LegacyAddressHash: legacyHash,
		TargetChain:       walletChainSolana,
		TargetAddress:     wallet.Address,
		TargetAccount:     wallet.Account,
		TokenSymbol:       tokenSymbol,
		RequiredProofs: []string{
			"legacy_wallet_ownership_signature",
			"anchor_register_legacy_wallet_transaction",
		},
		Contract: WalletMigrationContract{
			Network:      solanaNetworkFromRPCURL(cfg.CryptoRPCURL),
			ProgramID:    programID,
			ProgramReady: programReady,
			Instruction:  "register_legacy_wallet",
			PDASeeds:     []string{walletMigrationPDASeed, chain, "legacy_address_hash_bytes"},
			PDASeedFormats: []string{
				"utf8",
				"utf8",
				"bytes32:hex_decode(contract.args.legacy_address_hash)",
			},
			Args: WalletMigrationContractArgs{
				LegacyChain:       chain,
				LegacyAddressHash: legacyHash,
				SolanaWallet:      wallet.Address,
			},
			TokenMint:        normalizeWalletAddress(cfg.CryptoTokenContract),
			TreasuryReceiver: normalizeWalletAddress(cfg.CryptoReceiver),
		},
		Wallet:    wallet,
		CreatedAt: now,
	}
}

func walletMigrationLedgerReference(response WalletMigrationResponse) string {
	fields := []string{
		"wallet_migration:" + sanitizeLedgerReferenceValue(response.MigrationID),
		"legacy_chain:" + sanitizeLedgerReferenceValue(response.LegacyChain),
		"legacy_hash:" + sanitizeLedgerReferenceValue(response.LegacyAddressHash),
		"target:" + sanitizeLedgerReferenceValue(response.TargetAddress),
		"instruction:" + sanitizeLedgerReferenceValue(response.Contract.Instruction),
	}
	if response.Contract.ProgramID != "" {
		fields = append(fields, "program:"+sanitizeLedgerReferenceValue(response.Contract.ProgramID))
	}
	return strings.Join(fields, ";")
}

func solanaNetworkFromRPCURL(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch {
	case strings.Contains(normalized, "mainnet"):
		return "mainnet-beta"
	case strings.Contains(normalized, "devnet"):
		return "devnet"
	case strings.Contains(normalized, "testnet"):
		return "testnet"
	default:
		return "localnet"
	}
}

func trimAddressPrefix(value, prefix string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(strings.ToLower(value), prefix) {
		return strings.TrimSpace(value[len(prefix):])
	}
	return value
}

func base58Encode(input []byte) string {
	if len(input) == 0 {
		return ""
	}
	value := new(big.Int).SetBytes(input)
	base := big.NewInt(58)
	zero := big.NewInt(0)
	mod := new(big.Int)
	encoded := []byte{}
	for value.Cmp(zero) > 0 {
		value.DivMod(value, base, mod)
		encoded = append(encoded, solanaBase58Alphabet[mod.Int64()])
	}
	for _, b := range input {
		if b != 0 {
			break
		}
		encoded = append(encoded, solanaBase58Alphabet[0])
	}
	if len(encoded) == 0 {
		encoded = append(encoded, solanaBase58Alphabet[0])
	}
	for left, right := 0, len(encoded)-1; left < right; left, right = left+1, right-1 {
		encoded[left], encoded[right] = encoded[right], encoded[left]
	}
	return string(encoded)
}

func base58Decode(value string) ([]byte, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, false
	}
	result := big.NewInt(0)
	base := big.NewInt(58)
	for _, char := range value {
		index := strings.IndexRune(solanaBase58Alphabet, char)
		if index < 0 {
			return nil, false
		}
		result.Mul(result, base)
		result.Add(result, big.NewInt(int64(index)))
	}
	decoded := result.Bytes()
	leadingZeros := 0
	for _, char := range value {
		if char != rune(solanaBase58Alphabet[0]) {
			break
		}
		leadingZeros++
	}
	if leadingZeros > 0 {
		decoded = append(make([]byte, leadingZeros), decoded...)
	}
	return decoded, true
}
