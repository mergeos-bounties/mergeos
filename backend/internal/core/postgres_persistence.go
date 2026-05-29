package core

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

//go:embed migrations/*.sql
var postgresMigrations embed.FS

type postgresPersistence struct {
	db *sql.DB
}

func newPostgresPersistence(ctx context.Context, cfg Config) (*postgresPersistence, error) {
	db, err := sql.Open("pgx", strings.TrimSpace(cfg.DatabaseURL))
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	persistence := &postgresPersistence{db: db}
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	if err := persistence.migrate(pingCtx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return persistence, nil
}

func (p *postgresPersistence) Close() error {
	return p.db.Close()
}

func (p *postgresPersistence) migrate(ctx context.Context) error {
	if _, err := p.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
  version text PRIMARY KEY,
  applied_at timestamptz NOT NULL DEFAULT now()
)`); err != nil {
		return fmt.Errorf("ensure schema migrations table: %w", err)
	}

	entries, err := fs.ReadDir(postgresMigrations, "migrations")
	if err != nil {
		return fmt.Errorf("read postgres migrations: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin postgres migration: %w", err)
	}
	defer tx.Rollback()

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		version := strings.TrimSuffix(entry.Name(), ".sql")
		var applied bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, version).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %s: %w", version, err)
		}
		if applied {
			continue
		}
		statement, err := postgresMigrations.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", version, err)
		}
		if _, err := tx.ExecContext(ctx, string(statement)); err != nil {
			return fmt.Errorf("apply migration %s: %w", version, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, version); err != nil {
			return fmt.Errorf("record migration %s: %w", version, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit postgres migrations: %w", err)
	}
	return nil
}

func (p *postgresPersistence) Load(ctx context.Context) (persistedState, bool, error) {
	found, err := p.hasState(ctx)
	if err != nil {
		return persistedState{}, false, err
	}
	if !found {
		return persistedState{}, false, nil
	}

	state := persistedState{NextID: 1}
	if err := p.loadMeta(ctx, &state); err != nil {
		return persistedState{}, false, err
	}
	if err := p.loadUsers(ctx, &state); err != nil {
		return persistedState{}, false, err
	}
	if err := p.loadWallets(ctx, &state); err != nil {
		return persistedState{}, false, err
	}
	projectsByID, err := p.loadProjects(ctx, &state)
	if err != nil {
		return persistedState{}, false, err
	}
	if err := p.loadTasks(ctx, &state, projectsByID); err != nil {
		return persistedState{}, false, err
	}
	if err := p.loadSessions(ctx, &state); err != nil {
		return persistedState{}, false, err
	}
	if err := p.loadNotifications(ctx, &state); err != nil {
		return persistedState{}, false, err
	}
	if err := p.loadAttachments(ctx, &state, projectsByID); err != nil {
		return persistedState{}, false, err
	}
	if err := p.loadSSLReviews(ctx, &state); err != nil {
		return persistedState{}, false, err
	}
	if err := p.loadGeminiAPIKeys(ctx, &state); err != nil {
		return persistedState{}, false, err
	}
	if err := p.loadGeminiWebhookLogs(ctx, &state); err != nil {
		return persistedState{}, false, err
	}
	if err := p.loadLedger(ctx, &state); err != nil {
		return persistedState{}, false, err
	}
	if err := p.loadTestSettingsConfig(ctx, &state); err != nil {
		return persistedState{}, false, err
	}
	if err := p.loadTestSettingsEntries(ctx, &state); err != nil {
		return persistedState{}, false, err
	}
	return state, true, nil
}

func (p *postgresPersistence) hasState(ctx context.Context) (bool, error) {
	var found bool
	err := p.db.QueryRowContext(ctx, `
SELECT EXISTS (SELECT 1 FROM store_meta WHERE key = 'next_id')
   OR EXISTS (SELECT 1 FROM users)
   OR EXISTS (SELECT 1 FROM wallets)
   OR EXISTS (SELECT 1 FROM projects)
   OR EXISTS (SELECT 1 FROM gemini_api_keys)
   OR EXISTS (SELECT 1 FROM gemini_webhook_logs)
   OR EXISTS (SELECT 1 FROM ledger_entries)
   OR EXISTS (SELECT 1 FROM test_settings_config)
   OR EXISTS (SELECT 1 FROM test_settings_entries)`).Scan(&found)
	if err != nil {
		return false, fmt.Errorf("check postgres state: %w", err)
	}
	return found, nil
}

func (p *postgresPersistence) loadMeta(ctx context.Context, state *persistedState) error {
	rows, err := p.db.QueryContext(ctx, `SELECT key, value, updated_at FROM store_meta WHERE key IN ('next_id', 'gemini_review_model', 'llm_provider', 'llm_model')`)
	if err != nil {
		return fmt.Errorf("load store meta: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var key string
		var raw string
		var updatedAt sql.NullTime
		if err := rows.Scan(&key, &raw, &updatedAt); err != nil {
			return fmt.Errorf("scan store meta: %w", err)
		}
		switch key {
		case "next_id":
			nextID, err := strconv.Atoi(raw)
			if err != nil {
				return fmt.Errorf("parse postgres next_id %q: %w", raw, err)
			}
			if nextID > 0 {
				state.NextID = nextID
			}
		case "gemini_review_model":
			if state.AdminSettings == nil {
				settings := defaultAdminSettings(Config{})
				state.AdminSettings = &settings
			}
			state.AdminSettings.GeminiReviewModel = raw
			if strings.TrimSpace(state.AdminSettings.LLMProvider) == "" {
				state.AdminSettings.LLMProvider = "gemini"
			}
			if strings.TrimSpace(state.AdminSettings.LLMModel) == "" {
				state.AdminSettings.LLMModel = raw
			}
			if updatedAt.Valid {
				state.AdminSettings.UpdatedAt = updatedAt.Time
			}
		case "llm_provider":
			if state.AdminSettings == nil {
				settings := defaultAdminSettings(Config{})
				state.AdminSettings = &settings
			}
			state.AdminSettings.LLMProvider = raw
			if updatedAt.Valid {
				state.AdminSettings.UpdatedAt = updatedAt.Time
			}
		case "llm_model":
			if state.AdminSettings == nil {
				settings := defaultAdminSettings(Config{})
				state.AdminSettings = &settings
			}
			state.AdminSettings.LLMModel = raw
			if updatedAt.Valid {
				state.AdminSettings.UpdatedAt = updatedAt.Time
			}
		}
	}
	return rows.Err()
}

func (p *postgresPersistence) loadUsers(ctx context.Context, state *persistedState) error {
	rows, err := p.db.QueryContext(ctx, `
SELECT id, name, company_name, email, role, password_salt, password_hash, wallet_address, github_id, github_username, github_avatar_url, created_at, last_login_at
FROM users
ORDER BY created_at, id`)
	if err != nil {
		return fmt.Errorf("load users: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var user User
		var lastLogin sql.NullTime
		if err := rows.Scan(
			&user.ID, &user.Name, &user.CompanyName, &user.Email, &user.Role, &user.PasswordSalt,
			&user.PasswordHash, &user.WalletAddress, &user.GitHubID, &user.GitHubUsername,
			&user.GitHubAvatarURL, &user.CreatedAt, &lastLogin,
		); err != nil {
			return fmt.Errorf("scan user: %w", err)
		}
		user.LastLoginAt = timePtr(lastLogin)
		state.Users = append(state.Users, &user)
	}
	return rows.Err()
}

func (p *postgresPersistence) loadWallets(ctx context.Context, state *persistedState) error {
	rows, err := p.db.QueryContext(ctx, `
SELECT address, owner_user_id, github_id, github_username, recovery_salt, recovery_hash, created_at, linked_at
FROM wallets
ORDER BY created_at, address`)
	if err != nil {
		return fmt.Errorf("load wallets: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		wallet := &Wallet{}
		var linkedAt sql.NullTime
		if err := rows.Scan(
			&wallet.Address, &wallet.OwnerUserID, &wallet.GitHubID, &wallet.GitHubUsername,
			&wallet.RecoverySalt, &wallet.RecoveryHash, &wallet.CreatedAt, &linkedAt,
		); err != nil {
			return fmt.Errorf("scan wallet: %w", err)
		}
		wallet.LinkedAt = timePtr(linkedAt)
		state.Wallets = append(state.Wallets, wallet)
	}
	return rows.Err()
}

func (p *postgresPersistence) loadProjects(ctx context.Context, state *persistedState) (map[string]*Project, error) {
	rows, err := p.db.QueryContext(ctx, `
SELECT id, client_user_id, title, client_name, company_name, client_email, phone, site_type, package_tier, timeline,
       brief, payment_method, payment_status, payment_provider, payment_reference, bounty_repo_name, repo_visibility,
       repo_provider, repo_url, repo_local_path, budget_cents, fee_cents, work_pool_cents, status, created_at
FROM projects
ORDER BY created_at, id`)
	if err != nil {
		return nil, fmt.Errorf("load projects: %w", err)
	}
	defer rows.Close()

	projects := map[string]*Project{}
	for rows.Next() {
		project := &Project{}
		if err := rows.Scan(
			&project.ID, &project.ClientUserID, &project.Title, &project.ClientName, &project.CompanyName,
			&project.ClientEmail, &project.Phone, &project.SiteType, &project.PackageTier, &project.Timeline,
			&project.Brief, &project.PaymentMethod, &project.PaymentStatus, &project.PaymentProvider,
			&project.PaymentReference, &project.BountyRepoName, &project.RepoVisibility, &project.RepoProvider,
			&project.RepoURL, &project.RepoLocalPath, &project.BudgetCents, &project.FeeCents,
			&project.WorkPoolCents, &project.Status, &project.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		project.Tasks = []*Task{}
		project.Attachments = []*Attachment{}
		projects[project.ID] = project
		state.Projects = append(state.Projects, project)
	}
	return projects, rows.Err()
}

func (p *postgresPersistence) loadTasks(ctx context.Context, state *persistedState, projects map[string]*Project) error {
	rows, err := p.db.QueryContext(ctx, `
SELECT id, project_id, issue_number, title, acceptance, reward_cents, required_worker_kind, suggested_agent_type, bounty_type,
       status, worker_kind, worker_id, agent_type, proof_hash, issue_url, issue_state, created_at, accepted_at
FROM tasks
ORDER BY project_id, issue_number, created_at, id`)
	if err != nil {
		return fmt.Errorf("load tasks: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		task := &Task{}
		var acceptedAt sql.NullTime
		if err := rows.Scan(
			&task.ID, &task.ProjectID, &task.IssueNumber, &task.Title, &task.Acceptance, &task.RewardCents,
			&task.RequiredWorkerKind, &task.SuggestedAgentType, &task.BountyType, &task.Status, &task.WorkerKind, &task.WorkerID,
			&task.AgentType, &task.ProofHash, &task.IssueURL, &task.IssueState, &task.CreatedAt, &acceptedAt,
		); err != nil {
			return fmt.Errorf("scan task: %w", err)
		}
		task.AcceptedAt = timePtr(acceptedAt)
		state.Tasks = append(state.Tasks, task)
		if project, ok := projects[task.ProjectID]; ok {
			taskCopy := *task
			project.Tasks = append(project.Tasks, &taskCopy)
		}
	}
	return rows.Err()
}

func (p *postgresPersistence) loadSessions(ctx context.Context, state *persistedState) error {
	rows, err := p.db.QueryContext(ctx, `
SELECT token, user_id, created_at, expires_at
FROM sessions
WHERE expires_at > now()
ORDER BY created_at, token`)
	if err != nil {
		return fmt.Errorf("load sessions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		session := &Session{}
		if err := rows.Scan(&session.Token, &session.UserID, &session.CreatedAt, &session.ExpiresAt); err != nil {
			return fmt.Errorf("scan session: %w", err)
		}
		state.Sessions = append(state.Sessions, session)
	}
	return rows.Err()
}

func (p *postgresPersistence) loadNotifications(ctx context.Context, state *persistedState) error {
	rows, err := p.db.QueryContext(ctx, `
SELECT id, user_id, project_id, channel, subject, body, status, created_at
FROM notifications
ORDER BY created_at, id`)
	if err != nil {
		return fmt.Errorf("load notifications: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		notification := &Notification{}
		if err := rows.Scan(&notification.ID, &notification.UserID, &notification.ProjectID, &notification.Channel, &notification.Subject, &notification.Body, &notification.Status, &notification.CreatedAt); err != nil {
			return fmt.Errorf("scan notification: %w", err)
		}
		state.Notifications = append(state.Notifications, notification)
	}
	return rows.Err()
}

func (p *postgresPersistence) loadAttachments(ctx context.Context, state *persistedState, projects map[string]*Project) error {
	rows, err := p.db.QueryContext(ctx, `
SELECT id, user_id, project_id, original_name, stored_name, content_type, size_bytes, url, stored_path, is_image, created_at
FROM attachments
ORDER BY created_at, id`)
	if err != nil {
		return fmt.Errorf("load attachments: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		attachment := &Attachment{}
		if err := rows.Scan(
			&attachment.ID, &attachment.UserID, &attachment.ProjectID, &attachment.OriginalName,
			&attachment.StoredName, &attachment.ContentType, &attachment.SizeBytes, &attachment.URL,
			&attachment.StoredPath, &attachment.IsImage, &attachment.CreatedAt,
		); err != nil {
			return fmt.Errorf("scan attachment: %w", err)
		}
		if attachment.URL == "" {
			attachment.URL = "/api/uploads/" + attachment.ID + "/download"
		}
		state.Attachments = append(state.Attachments, attachment)
		if project, ok := projects[attachment.ProjectID]; ok {
			project.Attachments = append(project.Attachments, cloneAttachment(attachment))
		}
	}
	return rows.Err()
}

func (p *postgresPersistence) loadSSLReviews(ctx context.Context, state *persistedState) error {
	rows, err := p.db.QueryContext(ctx, `
SELECT domain, port, status, issuer, subject, serial_number, dns_names, not_before, not_after, days_remaining,
       last_checked_at, next_check_at, error, checked_by
FROM ssl_reviews
ORDER BY domain`)
	if err != nil {
		return fmt.Errorf("load ssl reviews: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		review := &SSLReviewStatus{}
		var dnsRaw []byte
		var notBefore, notAfter, lastCheckedAt, nextCheckAt sql.NullTime
		if err := rows.Scan(
			&review.Domain, &review.Port, &review.Status, &review.Issuer, &review.Subject, &review.SerialNumber,
			&dnsRaw, &notBefore, &notAfter, &review.DaysRemaining, &lastCheckedAt, &nextCheckAt, &review.Error, &review.CheckedBy,
		); err != nil {
			return fmt.Errorf("scan ssl review: %w", err)
		}
		if len(dnsRaw) > 0 {
			if err := json.Unmarshal(dnsRaw, &review.DNSNames); err != nil {
				return fmt.Errorf("decode ssl dns names for %s: %w", review.Domain, err)
			}
		}
		review.NotBefore = timePtr(notBefore)
		review.NotAfter = timePtr(notAfter)
		review.LastCheckedAt = timePtr(lastCheckedAt)
		review.NextCheckAt = timePtr(nextCheckAt)
		state.SSLReviews = append(state.SSLReviews, review)
	}
	return rows.Err()
}

func (p *postgresPersistence) loadLedger(ctx context.Context, state *persistedState) error {
	rows, err := p.db.QueryContext(ctx, `
SELECT sequence, type, from_account, to_account, amount_cents, reference, previous_hash, entry_hash, created_at
FROM ledger_entries
ORDER BY sequence`)
	if err != nil {
		return fmt.Errorf("load ledger: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var entry LedgerEntry
		if err := rows.Scan(&entry.Sequence, &entry.Type, &entry.FromAccount, &entry.ToAccount, &entry.AmountCents, &entry.Reference, &entry.PreviousHash, &entry.EntryHash, &entry.CreatedAt); err != nil {
			return fmt.Errorf("scan ledger entry: %w", err)
		}
		state.Ledger = append(state.Ledger, entry)
	}
	return rows.Err()
}

func (p *postgresPersistence) loadGeminiAPIKeys(ctx context.Context, state *persistedState) error {
	rows, err := p.db.QueryContext(ctx, `
SELECT id, key_value, key_hint, status, request_count, success_count, quota_error_count,
       last_status_code, last_error, last_used_at, created_at, updated_at, provider, model
FROM gemini_api_keys
ORDER BY request_count, last_used_at NULLS FIRST, id`)
	if err != nil {
		return fmt.Errorf("load gemini api keys: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var key GeminiAPIKey
		var lastUsedAt sql.NullTime
		if err := rows.Scan(
			&key.ID, &key.KeyValue, &key.KeyHint, &key.Status, &key.RequestCount, &key.SuccessCount,
			&key.QuotaErrorCount, &key.LastStatusCode, &key.LastError, &lastUsedAt, &key.CreatedAt, &key.UpdatedAt,
			&key.Provider, &key.Model,
		); err != nil {
			return fmt.Errorf("scan gemini api key: %w", err)
		}
		key.LastUsedAt = timePtr(lastUsedAt)
		state.GeminiAPIKeys = append(state.GeminiAPIKeys, &key)
	}
	return rows.Err()
}

func (p *postgresPersistence) loadGeminiWebhookLogs(ctx context.Context, state *persistedState) error {
	rows, err := p.db.QueryContext(ctx, `
SELECT id, delivery_id, event_name, action, repository, pull_number, sender, status, status_code,
       error, comment_url, key_id, labels, duration_millis, received_at, completed_at
FROM gemini_webhook_logs
ORDER BY received_at DESC
LIMIT 200`)
	if err != nil {
		return fmt.Errorf("load gemini webhook logs: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var log GeminiWebhookLog
		var labelsRaw string
		var completedAt sql.NullTime
		if err := rows.Scan(
			&log.ID, &log.DeliveryID, &log.EventName, &log.Action, &log.Repository, &log.PullNumber, &log.Sender,
			&log.Status, &log.StatusCode, &log.Error, &log.CommentURL, &log.KeyID, &labelsRaw,
			&log.DurationMillis, &log.ReceivedAt, &completedAt,
		); err != nil {
			return fmt.Errorf("scan gemini webhook log: %w", err)
		}
		if labelsRaw != "" {
			_ = json.Unmarshal([]byte(labelsRaw), &log.Labels)
		}
		log.CompletedAt = timePtr(completedAt)
		state.GeminiWebhookLogs = append(state.GeminiWebhookLogs, &log)
	}
	return rows.Err()
}

func (p *postgresPersistence) Save(ctx context.Context, state persistedState) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin postgres save: %w", err)
	}
	defer tx.Rollback()

	for _, table := range []string{
		"test_settings_entries",
		"test_settings_config",
		"ledger_entries",
		"gemini_webhook_logs",
		"gemini_api_keys",
		"ssl_reviews",
		"attachments",
		"notifications",
		"sessions",
		"tasks",
		"projects",
		"wallets",
		"users",
		"store_meta",
	} {
		if _, err := tx.ExecContext(ctx, "DELETE FROM "+table); err != nil {
			return fmt.Errorf("clear %s: %w", table, err)
		}
	}

	if _, err := tx.ExecContext(ctx, `
INSERT INTO store_meta (key, value, updated_at)
VALUES ('next_id', $1, now())`, strconv.Itoa(state.NextID)); err != nil {
		return fmt.Errorf("save store meta: %w", err)
	}
	settings := state.AdminSettings
	if settings == nil {
		settings = cloneAdminSettings(defaultAdminSettings(Config{}))
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO store_meta (key, value, updated_at)
VALUES ('gemini_review_model', $1, $2)`, normalizedGeminiReviewModelOrDefault(settings.GeminiReviewModel), settings.UpdatedAt); err != nil {
		return fmt.Errorf("save admin settings: %w", err)
	}
	provider := normalizedLLMProviderOrDefault(settings.LLMProvider)
	model := normalizedLLMModelOrDefault(provider, settings.LLMModel)
	if _, err := tx.ExecContext(ctx, `
INSERT INTO store_meta (key, value, updated_at)
VALUES ('llm_provider', $1, $2), ('llm_model', $3, $2)`, provider, settings.UpdatedAt, model); err != nil {
		return fmt.Errorf("save LLM settings: %w", err)
	}
	if err := saveUsers(ctx, tx, state.Users); err != nil {
		return err
	}
	if err := saveWallets(ctx, tx, state.Wallets); err != nil {
		return err
	}
	if err := saveProjects(ctx, tx, state.Projects); err != nil {
		return err
	}
	if err := saveTasks(ctx, tx, state.Tasks); err != nil {
		return err
	}
	if err := saveSessions(ctx, tx, state.Sessions); err != nil {
		return err
	}
	if err := saveNotifications(ctx, tx, state.Notifications); err != nil {
		return err
	}
	if err := saveAttachments(ctx, tx, state.Attachments); err != nil {
		return err
	}
	if err := saveSSLReviews(ctx, tx, state.SSLReviews); err != nil {
		return err
	}
	if err := saveGeminiAPIKeys(ctx, tx, state.GeminiAPIKeys); err != nil {
		return err
	}
	if err := saveGeminiWebhookLogs(ctx, tx, state.GeminiWebhookLogs); err != nil {
		return err
	}
	if err := saveLedger(ctx, tx, state.Ledger); err != nil {
		return err
	}
	if err := saveTestSettingsConfig(ctx, tx, state.TestSettingsConfig); err != nil {
		return err
	}
	if err := saveTestSettingsEntries(ctx, tx, state.TestSettingsEntries); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit postgres save: %w", err)
	}
	return nil
}

func saveUsers(ctx context.Context, tx *sql.Tx, users []*User) error {
	for _, user := range users {
		if user == nil {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO users (
  id, name, company_name, email, role, password_salt, password_hash, wallet_address,
  github_id, github_username, github_avatar_url, created_at, last_login_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8,
  $9, $10, $11, $12, $13
)`,
			user.ID, user.Name, user.CompanyName, user.Email, user.Role, user.PasswordSalt, user.PasswordHash,
			normalizeWalletAddress(user.WalletAddress), user.GitHubID, normalizeGitHubUsername(user.GitHubUsername),
			user.GitHubAvatarURL, user.CreatedAt, user.LastLoginAt,
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
		if _, err := tx.ExecContext(ctx, `
INSERT INTO wallets (address, owner_user_id, github_id, github_username, recovery_salt, recovery_hash, created_at, linked_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			address, wallet.OwnerUserID, wallet.GitHubID, normalizeGitHubUsername(wallet.GitHubUsername),
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
		if _, err := tx.ExecContext(ctx, `
INSERT INTO projects (
  id, client_user_id, title, client_name, company_name, client_email, phone, site_type, package_tier, timeline,
  brief, payment_method, payment_status, payment_provider, payment_reference, bounty_repo_name, repo_visibility,
  repo_provider, repo_url, repo_local_path, budget_cents, fee_cents, work_pool_cents, status, created_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
  $11, $12, $13, $14, $15, $16, $17,
  $18, $19, $20, $21, $22, $23, $24, $25
)`,
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
		if _, err := tx.ExecContext(ctx, `
INSERT INTO tasks (
  id, project_id, issue_number, title, acceptance, reward_cents, required_worker_kind, suggested_agent_type, bounty_type,
  status, worker_kind, worker_id, agent_type, proof_hash, issue_url, issue_state, created_at, accepted_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8,
  $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
)`,
			task.ID, task.ProjectID, task.IssueNumber, task.Title, task.Acceptance, task.RewardCents, task.RequiredWorkerKind,
			task.SuggestedAgentType, task.BountyType, task.Status, task.WorkerKind, task.WorkerID, task.AgentType, task.ProofHash,
			task.IssueURL, normalizeIssueState(task.IssueState), task.CreatedAt, task.AcceptedAt,
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
		if _, err := tx.ExecContext(ctx, `
INSERT INTO sessions (token, user_id, created_at, expires_at)
VALUES ($1, $2, $3, $4)`,
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
		if _, err := tx.ExecContext(ctx, `
INSERT INTO notifications (id, user_id, project_id, channel, subject, body, status, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
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
		if _, err := tx.ExecContext(ctx, `
INSERT INTO attachments (id, user_id, project_id, original_name, stored_name, content_type, size_bytes, url, stored_path, is_image, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
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
		if _, err := tx.ExecContext(ctx, `
INSERT INTO ssl_reviews (
  domain, port, status, issuer, subject, serial_number, dns_names, not_before, not_after, days_remaining,
  last_checked_at, next_check_at, error, checked_by
) VALUES (
  $1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10,
  $11, $12, $13, $14
)`,
			review.Domain, review.Port, review.Status, review.Issuer, review.Subject, review.SerialNumber,
			string(dnsNames), review.NotBefore, review.NotAfter, review.DaysRemaining, review.LastCheckedAt,
			review.NextCheckAt, review.Error, review.CheckedBy,
		); err != nil {
			return fmt.Errorf("save ssl review %s: %w", review.Domain, err)
		}
	}
	return nil
}

func saveGeminiAPIKeys(ctx context.Context, tx *sql.Tx, keys []*GeminiAPIKey) error {
	for _, key := range keys {
		if key == nil || strings.TrimSpace(key.KeyValue) == "" {
			continue
		}
		if key.ID == "" {
			key.ID = geminiAPIKeyID(key.KeyValue)
		}
		key.Provider = normalizedLLMProviderOrDefault(key.Provider)
		key.Model = normalizedLLMModelOrDefault(key.Provider, key.Model)
		if key.KeyHint == "" {
			key.KeyHint = geminiAPIKeyHint(key.KeyValue)
		}
		if key.Status == "" {
			key.Status = GeminiAPIKeyStatusActive
		}
		if key.CreatedAt.IsZero() {
			key.CreatedAt = time.Now().UTC()
		}
		if key.UpdatedAt.IsZero() {
			key.UpdatedAt = key.CreatedAt
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO gemini_api_keys (
  id, provider, model, key_value, key_hint, status, request_count, success_count, quota_error_count,
  last_status_code, last_error, last_used_at, created_at, updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
)`,
			key.ID, key.Provider, key.Model, key.KeyValue, key.KeyHint, key.Status, key.RequestCount,
			key.SuccessCount, key.QuotaErrorCount, key.LastStatusCode, key.LastError, key.LastUsedAt,
			key.CreatedAt, key.UpdatedAt,
		); err != nil {
			return fmt.Errorf("save gemini api key %s: %w", key.ID, err)
		}
	}
	return nil
}

func saveGeminiWebhookLogs(ctx context.Context, tx *sql.Tx, logs []*GeminiWebhookLog) error {
	if len(logs) > maxGeminiWebhookLogs {
		sort.Slice(logs, func(i, j int) bool {
			return logs[i].ReceivedAt.After(logs[j].ReceivedAt)
		})
		logs = logs[:maxGeminiWebhookLogs]
	}
	for _, log := range logs {
		if log == nil {
			continue
		}
		if log.ID == "" {
			log.ID = geminiWebhookLogID()
		}
		if log.ReceivedAt.IsZero() {
			log.ReceivedAt = time.Now().UTC()
		}
		labels, err := json.Marshal(log.Labels)
		if err != nil {
			return fmt.Errorf("encode gemini webhook labels %s: %w", log.ID, err)
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO gemini_webhook_logs (
  id, delivery_id, event_name, action, repository, pull_number, sender, status, status_code,
  error, comment_url, key_id, labels, duration_millis, received_at, completed_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8,
  $9, $10, $11, $12, $13::jsonb, $14, $15, $16
)`,
			log.ID, log.DeliveryID, log.EventName, log.Action, log.Repository, log.PullNumber, log.Sender,
			log.Status, log.StatusCode, log.Error, log.CommentURL, log.KeyID, string(labels),
			log.DurationMillis, log.ReceivedAt, log.CompletedAt,
		); err != nil {
			return fmt.Errorf("save gemini webhook log %s: %w", log.ID, err)
		}
	}
	return nil
}

func saveLedger(ctx context.Context, tx *sql.Tx, ledger []LedgerEntry) error {
	for _, entry := range ledger {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO ledger_entries (sequence, type, from_account, to_account, amount_cents, reference, previous_hash, entry_hash, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
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
func (p *postgresPersistence) loadTestSettingsConfig(ctx context.Context, state *persistedState) error {
	var raw string
	err := p.db.QueryRowContext(ctx, `SELECT value FROM store_meta WHERE key = 'test_settings_config'`).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("load test_settings_config: %w", err)
	}
	var config TestSettingsConfig
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		return fmt.Errorf("parse test_settings_config: %w", err)
	}
	state.TestSettingsConfig = &config
	return nil
}

func (p *postgresPersistence) loadTestSettingsEntries(ctx context.Context, state *persistedState) error {
	rows, err := p.db.QueryContext(ctx, `SELECT id, integration_type, display_name, setting_key, setting_value, key_value_map, status, last_used_at, created_at, updated_at FROM test_settings_entries ORDER BY created_at, id`)
	if err != nil {
		return fmt.Errorf("load test_settings_entries: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var entry TestSettingsEntry
		var kvRaw []byte
		var lastUsedAt sql.NullTime
		if err := rows.Scan(
			&entry.ID, &entry.IntegrationType, &entry.DisplayName, &entry.SettingKey,
			&entry.SettingValue, &kvRaw, &entry.Status, &lastUsedAt,
			&entry.CreatedAt, &entry.UpdatedAt,
		); err != nil {
			return fmt.Errorf("scan test_settings_entry: %w", err)
		}
		entry.LastUsedAt = timePtr(lastUsedAt)
		if len(kvRaw) > 0 {
			if err := json.Unmarshal(kvRaw, &entry.KeyValueMap); err != nil {
				return fmt.Errorf("decode key_value_map for entry %s: %w", entry.ID, err)
			}
		}
		if entry.KeyValueMap == nil {
			entry.KeyValueMap = map[string]string{}
		}
		state.TestSettingsEntries = append(state.TestSettingsEntries, &entry)
	}
	return rows.Err()
}

func saveTestSettingsConfig(ctx context.Context, tx *sql.Tx, config *TestSettingsConfig) error {
	if config == nil {
		return nil
	}
	raw, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal test_settings_config: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO store_meta (key, value, updated_at) VALUES ('test_settings_config', $1, now())`, string(raw)); err != nil {
		return fmt.Errorf("save test_settings_config: %w", err)
	}
	return nil
}

func saveTestSettingsEntries(ctx context.Context, tx *sql.Tx, entries []*TestSettingsEntry) error {
	for _, entry := range entries {
		if entry == nil || entry.ID == "" {
			continue
		}
		if entry.CreatedAt.IsZero() {
			entry.CreatedAt = time.Now().UTC()
		}
		if entry.UpdatedAt.IsZero() {
			entry.UpdatedAt = entry.CreatedAt
		}
		kvRaw, err := json.Marshal(entry.KeyValueMap)
		if err != nil {
			return fmt.Errorf("encode key_value_map for entry %s: %w", entry.ID, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO test_settings_entries (id, integration_type, display_name, setting_key, setting_value, key_value_map, status, last_used_at, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9, $10) ON CONFLICT (id) DO UPDATE SET integration_type = EXCLUDED.integration_type, display_name = EXCLUDED.display_name, setting_key = EXCLUDED.setting_key, setting_value = EXCLUDED.setting_value, key_value_map = EXCLUDED.key_value_map, status = EXCLUDED.status, last_used_at = EXCLUDED.last_used_at, updated_at = EXCLUDED.updated_at`,
			entry.ID, entry.IntegrationType, entry.DisplayName, entry.SettingKey,
			entry.SettingValue, string(kvRaw), entry.Status, entry.LastUsedAt,
			entry.CreatedAt, entry.UpdatedAt,
		); err != nil {
			return fmt.Errorf("save test_settings_entry %s: %w", entry.ID, err)
		}
	}
	return nil
}
