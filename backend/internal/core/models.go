package core

import "time"

type PaymentMethod string

const (
	PaymentPayPal PaymentMethod = "paypal"
	PaymentCrypto PaymentMethod = "crypto"
)

type WorkerKind string

const (
	WorkerHuman  WorkerKind = "human"
	WorkerAgent  WorkerKind = "agent"
	WorkerHybrid WorkerKind = "hybrid"
)

type UserRole string

const (
	RoleClient UserRole = "client"
	RoleAdmin  UserRole = "admin"
)

type ProjectStatus string

const (
	ProjectFunded ProjectStatus = "funded"
)

type TaskStatus string

const (
	TaskOpen     TaskStatus = "open"
	TaskAccepted TaskStatus = "accepted"
)

type User struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	CompanyName       string            `json:"company_name"`
	Email             string            `json:"email"`
	Role              UserRole          `json:"role"`
	PasswordSalt      string            `json:"password_salt,omitempty"`
	PasswordHash      string            `json:"password_hash,omitempty"`
	WalletAddress     string            `json:"wallet_address"`
	GitHubID          int               `json:"github_id,omitempty"`
	GitHubUsername    string            `json:"github_username"`
	GitHubAvatarURL   string            `json:"github_avatar_url"`
	IdentityProviders map[string]string `json:"identity_providers,omitempty"`
	CreatedAt         time.Time         `json:"created_at"`
	LastLoginAt       *time.Time        `json:"last_login_at"`
}

type PublicUser struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	CompanyName     string    `json:"company_name"`
	Email           string    `json:"email"`
	Role            UserRole  `json:"role"`
	WalletAddress   string    `json:"wallet_address"`
	GitHubUsername  string    `json:"github_username"`
	GitHubAvatarURL string    `json:"github_avatar_url"`
	CreatedAt       time.Time `json:"created_at"`
	LastLoginAt     *time.Time `json:"last_login_at"`
}

type Wallet struct {
	Address       string    `json:"address"`
	OwnerUserID   string    `json:"owner_user_id"`
	GitHubID      int       `json:"github_id"`
	GitHubUsername string   `json:"github_username"`
	RecoverySalt  string    `json:"recovery_salt,omitempty"`
	RecoveryHash  string    `json:"recovery_hash,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	LinkedAt      time.Time `json:"linked_at"`
}

type Session struct {
	Token     string    `json:"token"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type Project struct {
	ID              string     `json:"id"`
	ClientUserID    string     `json:"client_user_id"`
	Title           string     `json:"title"`
	ClientName      string     `json:"client_name"`
	CompanyName     string     `json:"company_name"`
	ClientEmail     string     `json:"client_email"`
	Phone           string     `json:"phone,omitempty"`
	SiteType        string     `json:"site_type"`
	PackageTier     string     `json:"package_tier"`
	Timeline        string     `json:"timeline"`
	Brief           string     `json:"brief"`
	PaymentMethod   string     `json:"payment_method"`
	PaymentStatus   string     `json:"payment_status"`
	PaymentProvider string     `json:"payment_provider"`
	PaymentReference string    `json:"payment_reference"`
	BountyRepoName  string     `json:"bounty_repo_name"`
	RepoVisibility  string     `json:"repo_visibility"`
	RepoProvider    string     `json:"repo_provider"`
	RepoURL         string     `json:"repo_url"`
	RepoLocalPath   string     `json:"repo_local_path"`
	BudgetCents     int64      `json:"budget_cents"`
	FeeCents        int64      `json:"fee_cents"`
	WorkPoolCents   int64      `json:"work_pool_cents"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	Tasks           []*Task    `json:"tasks,omitempty"`
	Attachments     []*Attachment `json:"attachments,omitempty"`
}

type Task struct {
	ID                  string     `json:"id"`
	ProjectID           string     `json:"project_id"`
	IssueNumber         int        `json:"issue_number"`
	Title               string     `json:"title"`
	Acceptance          string     `json:"acceptance"`
	RewardCents         int64      `json:"reward_cents"`
	RequiredWorkerKind  string     `json:"required_worker_kind"`
	SuggestedAgentType  string     `json:"suggested_agent_type"`
	BountyType          string     `json:"bounty_type"`
	Status              string     `json:"status"`
	WorkerKind          string     `json:"worker_kind"`
	WorkerID            string     `json:"worker_id"`
	AgentType           string     `json:"agent_type"`
	ProofHash           string     `json:"proof_hash"`
	IssueURL            string     `json:"issue_url"`
	CreatedAt           time.Time  `json:"created_at"`
	AcceptedAt          *time.Time `json:"accepted_at"`
}

type Attachment struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id,omitempty"`
	ProjectID    string    `json:"project_id"`
	OriginalName string    `json:"original_name"`
	StoredName   string    `json:"stored_name"`
	ContentType  string    `json:"content_type"`
	SizeBytes    int64     `json:"size_bytes"`
	URL          string    `json:"url"`
	StoredPath   string    `json:"stored_path,omitempty"`
	IsImage      bool      `json:"is_image"`
	CreatedAt    time.Time `json:"created_at"`
}

type Notification struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	ProjectID string    `json:"project_id"`
	Channel   string    `json:"channel"`
	Subject   string    `json:"subject"`
	Body      string    `json:"body"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type SSLReviewStatus struct {
	Domain              string    `json:"domain"`
	Port                int       `json:"port"`
	Status              string    `json:"status"`
	Issuer              string    `json:"issuer"`
	Subject             string    `json:"subject"`
	SerialNumber        string    `json:"serial_number"`
	DNSNames            []string  `json:"dns_names"`
	NotBefore           *time.Time `json:"not_before"`
	NotAfter            *time.Time `json:"not_after"`
	DaysRemaining       int       `json:"days_remaining"`
	LastCheckedAt       *time.Time `json:"last_checked_at"`
	NextCheckAt         *time.Time `json:"next_check_at"`
	Error               string    `json:"error,omitempty"`
	CheckedBy           string    `json:"checked_by"`
}

type LedgerEntry struct {
	Sequence     int       `json:"sequence"`
	Type         string    `json:"type"`
	FromAccount  string    `json:"from_account"`
	ToAccount    string    `json:"to_account"`
	AmountCents  int64     `json:"amount_cents"`
	Reference    string    `json:"reference"`
	PreviousHash string    `json:"previous_hash"`
	EntryHash    string    `json:"entry_hash"`
	CreatedAt    time.Time `json:"created_at"`
}