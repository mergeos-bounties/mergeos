package core

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
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
	if err := p.loadLedger(ctx, &state); err != nil {
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
   OR EXISTS (SELECT 1 FROM ledger_entries)`).Scan(&found)
	if err != nil {
		return false, fmt.Errorf("check postgres state: %w", err)
	}
	return found, nil
}

func (p *postgresPersistence) loadMeta(ctx context.Context, state *persistedState) error {
	var raw string
	err := p.db.QueryRowContext(ctx, `SELECT value FROM store_meta WHERE key = 'next_id'`).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("load store meta: %w", err)
	}
	nextID, err := strconv.Atoi(raw)
	if err != nil {
		return fmt.Errorf("parse postgres next_id %q: %w", raw, err)
	}
	if nextID > 0 {
		state.NextID = nextID
	}
	return nil
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
       status, worker_kind, worker_id, agent_type, proof_hash, issue_url, created_at, accepted_at
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
			&task.AgentType, &task.ProofHash, &task.IssueURL, &task.CreatedAt, &acceptedAt,
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

func (p *postgresPersistence) Save(ctx context.Context, state persistedState) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin postgres save: %w", err)
	}
	defer tx.Rollback()

	for _, table := range []string{
		"ledger_entries",
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
	if err := saveLedger(ctx, tx, state.Ledger); err != nil {
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
  status, worker_kind, worker_id, agent_type, proof_hash, issue_url, created_at, accepted_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8,
  $9, $10, $11, $12, $13, $14, $15, $16, $17
)`,
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
