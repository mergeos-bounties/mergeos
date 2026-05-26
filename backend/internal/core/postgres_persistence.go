
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

func (p *postgresPersistence) Close() error {
	return p.db.Close()
}

func (p *postgresPersistence) migrate(ctx context.Context) error {
	if _, err := p.db.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS schema_migrations (version text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())"); err != nil {
		return fmt.Errorf("ensure schema migrations table: %w", err)
	}
	entries, err := fs.ReadDir(postgresMigrations, "migrations")
	if err != nil { return fmt.Errorf("read postgres migrations: %w", err) }
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil { return fmt.Errorf("begin postgres migration: %w", err) }
	defer tx.Rollback()

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") { continue }
		var version string
		if parts := strings.SplitN(entry.Name(), "_", 2); len(parts) > 0 { version = parts[0] }
		var count int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE version = $1", version).Scan(&count); err != nil { return fmt.Errorf("check migration %s: %w", version, err) }
		if count > 0 { continue }
		sqlBytes, err := fs.ReadFile(postgresMigrations, "migrations/"+entry.Name())
		if err != nil { return fmt.Errorf("read migration %s: %w", entry.Name(), err) }
		if _, err := tx.ExecContext(ctx, string(sqlBytes)); err != nil { return fmt.Errorf("apply migration %s: %w", entry.Name(), err) }
		if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil { return fmt.Errorf("record migration %s: %w", version, err) }
	}
	return tx.Commit()
}

func (p *postgresPersistence) Load(ctx context.Context) (*persistedState, error) {
	state := &persistedState{}
	if err := p.loadStoreMeta(ctx, state); err != nil { return nil, err }
	if err := p.loadUsers(ctx, state); err != nil { return nil, err }
	if err := p.loadWallets(ctx, state); err != nil { return nil, err }
	projects, err := p.loadProjects(ctx, state)
	if err != nil { return nil, err }
	if err := p.loadTasks(ctx, state, projects); err != nil { return nil, err }
	if err := p.loadSessions(ctx, state); err != nil { return nil, err }
	if err := p.loadNotifications(ctx, state); err != nil { return nil, err }
	if err := p.loadAttachments(ctx, state, projects); err != nil { return nil, err }
	if err := p.loadSSLReviews(ctx, state); err != nil { return nil, err }
	if err := p.loadLedger(ctx, state); err != nil { return nil, err }
	return state, nil
}

func (p *postgresPersistence) loadStoreMeta(ctx context.Context, state *persistedState) error {
	var key, value string
	var updatedAt time.Time
	err := p.db.QueryRowContext(ctx, "SELECT key, value, updated_at FROM store_meta WHERE key = 'next_id'").Scan(&key, &value, &updatedAt)
	if err == sql.ErrNoRows { state.NextID = 1; return nil }
	if err != nil { return fmt.Errorf("load store meta: %w", err) }
	state.NextID, _ = strconv.Atoi(value)
	if state.NextID < 1 { state.NextID = 1 }
	return nil
}

func (p *postgresPersistence) loadUsers(ctx context.Context, state *persistedState) error {
	rows, err := p.db.QueryContext(ctx, "SELECT id, name, company_name, email, role, password_salt, password_hash, wallet_address, github_id, github_username, github_avatar_url, COALESCE(identity_providers, '{}'::jsonb), created_at, last_login_at FROM users ORDER BY created_at, id")
	if err != nil { return fmt.Errorf("load users: %w", err) }
	defer rows.Close()
	for rows.Next() {
		u := &User{}
		var ghID sql.NullInt64; var ghUser, ghAvatar sql.NullString
		var idProviders []byte; var lastLoginAt sql.NullTime
		if err := rows.Scan(&u.ID, &u.Name, &u.CompanyName, &u.Email, &u.Role, &u.PasswordSalt, &u.PasswordHash, &u.WalletAddress, &ghID, &ghUser, &ghAvatar, &idProviders, &u.CreatedAt, &lastLoginAt); err != nil {
			return fmt.Errorf("scan user: %w", err)
		}
		u.GitHubID = int(ghID.Int64); u.GitHubUsername = ghUser.String; u.GitHubAvatarURL = ghAvatar.String
		if len(idProviders) > 0 { json.Unmarshal(idProviders, &u.IdentityProviders) }
		u.LastLoginAt = timePtr(lastLoginAt)
		state.Users = append(state.Users, u)
	}
	return rows.Err()
}

func saveUsers(ctx context.Context, tx *sql.Tx, users []*User) error {
	for _, user := range users {
		if user == nil { continue }
		providersJSON := []byte("{}")
		if len(user.IdentityProviders) > 0 { providersJSON, _ = json.Marshal(user.IdentityProviders) }
		if _, err := tx.ExecContext(ctx, "INSERT INTO users (id,name,company_name,email,role,password_salt,password_hash,wallet_address,github_id,github_username,github_avatar_url,identity_providers,created_at,last_login_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12::jsonb,$13,$14)",
			user.ID, user.Name, user.CompanyName, user.Email, user.Role, user.PasswordSalt, user.PasswordHash,
			normalizeWalletAddress(user.WalletAddress), int64(user.GitHubID), normalizeGitHubUsername(user.GitHubUsername), user.GitHubAvatarURL,
			string(providersJSON), user.CreatedAt, user.LastLoginAt,
		); err != nil { return fmt.Errorf("save user %s: %w", user.ID, err) }
	}
	return nil
}

func timePtr(t sql.NullTime) *time.Time {
	if !t.Valid { return nil }
	val := t.Time; return &val
}
