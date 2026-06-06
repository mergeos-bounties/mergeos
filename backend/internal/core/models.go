package core

import "time"

type PaymentMethod string

const (
	PaymentPayPal PaymentMethod = "paypal"
	PaymentCrypto PaymentMethod = "crypto"
	PaymentUSDT   PaymentMethod = "usdt"
	PaymentStripe PaymentMethod = "stripe"
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
	TaskOpen      TaskStatus = "open"
	TaskClaimed   TaskStatus = "claimed"
	TaskSubmitted TaskStatus = "submitted"
	TaskAccepted  TaskStatus = "accepted"
)

type User struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	CompanyName     string     `json:"company_name"`
	Email           string     `json:"email"`
	Role            UserRole   `json:"role"`
	PasswordSalt    string     `json:"-"`
	PasswordHash    string     `json:"-"`
	WalletAddress   string     `json:"wallet_address,omitempty"`
	GitHubID        string     `json:"github_id,omitempty"`
	GitHubUsername  string     `json:"github_username,omitempty"`
	GitHubAvatarURL string     `json:"github_avatar_url,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	LastLoginAt     *time.Time `json:"last_login_at,omitempty"`
}

type PublicUser struct {
	ID              string     `json:"id"`
	Name            string     `json:"name"`
	CompanyName     string     `json:"company_name"`
	Email           string     `json:"email"`
	Role            UserRole   `json:"role"`
	WalletAddress   string     `json:"wallet_address,omitempty"`
	GitHubUsername  string     `json:"github_username,omitempty"`
	GitHubAvatarURL string     `json:"github_avatar_url,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	LastLoginAt     *time.Time `json:"last_login_at,omitempty"`
}

type Wallet struct {
	Address        string     `json:"address"`
	Chain          string     `json:"chain,omitempty"`
	LegacyAddress  string     `json:"legacy_address,omitempty"`
	OwnerUserID    string     `json:"owner_user_id,omitempty"`
	GitHubID       string     `json:"github_id,omitempty"`
	GitHubUsername string     `json:"github_username,omitempty"`
	RecoverySalt   string     `json:"-"`
	RecoveryHash   string     `json:"-"`
	CreatedAt      time.Time  `json:"created_at"`
	LinkedAt       *time.Time `json:"linked_at,omitempty"`
}

type Session struct {
	Token     string    `json:"-"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type Notification struct {
	ID        string     `json:"id"`
	UserID    string     `json:"user_id"`
	ProjectID string     `json:"project_id,omitempty"`
	Channel   string     `json:"channel"`
	Subject   string     `json:"subject"`
	Body      string     `json:"body"`
	Status    string     `json:"status"`
	CreatedAt time.Time  `json:"created_at"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
}

type CreateDisputeRequest struct {
	ProjectID string `json:"project_id,omitempty"`
	TaskID    string `json:"task_id,omitempty"`
	Subject   string `json:"subject"`
	Body      string `json:"body"`
	Severity  string `json:"severity,omitempty"`
}

type CreateDisputeResponse struct {
	ProtocolVersion string       `json:"protocol_version"`
	Kind            string       `json:"kind"`
	DisputeID       string       `json:"dispute_id"`
	ProjectID       string       `json:"project_id"`
	TaskID          string       `json:"task_id,omitempty"`
	UserID          string       `json:"user_id"`
	Severity        string       `json:"severity"`
	Status          string       `json:"status"`
	Subject         string       `json:"subject"`
	Body            string       `json:"body"`
	Notification    Notification `json:"notification"`
	CreatedAt       time.Time    `json:"created_at"`
}

type CreateProposalRequest struct {
	TaskID         string  `json:"task_id"`
	CoverLetter    string  `json:"cover_letter"`
	BidCents       int64   `json:"bid_cents,omitempty"`
	EstimatedHours float64 `json:"estimated_hours,omitempty"`
	Availability   string  `json:"availability,omitempty"`
}

type CreateProposalResponse struct {
	ProtocolVersion      string                  `json:"protocol_version"`
	Kind                 string                  `json:"kind"`
	Proposal             WorkerSubmittedProposal `json:"proposal"`
	WorkerNotification   Notification            `json:"worker_notification"`
	CustomerNotification Notification            `json:"customer_notification"`
}

type ProposalDecisionRequest struct {
	Decision string `json:"decision"`
}

type Attachment struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id,omitempty"`
	ProjectID    string    `json:"project_id,omitempty"`
	OriginalName string    `json:"original_name"`
	StoredName   string    `json:"stored_name"`
	ContentType  string    `json:"content_type"`
	SizeBytes    int64     `json:"size_bytes"`
	URL          string    `json:"url"`
	StoredPath   string    `json:"-"`
	IsImage      bool      `json:"is_image"`
	CreatedAt    time.Time `json:"created_at"`
}

type Project struct {
	ID               string        `json:"id"`
	ClientUserID     string        `json:"client_user_id"`
	Title            string        `json:"title"`
	ClientName       string        `json:"client_name"`
	CompanyName      string        `json:"company_name"`
	ClientEmail      string        `json:"client_email"`
	Phone            string        `json:"phone"`
	SiteType         string        `json:"site_type"`
	PackageTier      string        `json:"package_tier"`
	Timeline         string        `json:"timeline"`
	Brief            string        `json:"brief"`
	PaymentMethod    PaymentMethod `json:"payment_method"`
	PaymentStatus    string        `json:"payment_status"`
	PaymentProvider  string        `json:"payment_provider"`
	PaymentReference string        `json:"payment_reference"`
	BountyRepoName   string        `json:"bounty_repo_name"`
	RepoVisibility   string        `json:"repo_visibility"`
	RepoProvider     string        `json:"repo_provider"`
	RepoURL          string        `json:"repo_url"`
	RepoLocalPath    string        `json:"repo_local_path,omitempty"`
	AllowAgents      *bool         `json:"allow_agents,omitempty"`
	BudgetCents      int64         `json:"budget_cents"`
	FeeCents         int64         `json:"fee_cents"`
	WorkPoolCents    int64         `json:"work_pool_cents"`
	Status           ProjectStatus `json:"status"`
	CreatedAt        time.Time     `json:"created_at"`
	Tasks            []*Task       `json:"tasks"`
	Attachments      []*Attachment `json:"attachments"`
}

type Task struct {
	ID                 string     `json:"id"`
	ProjectID          string     `json:"project_id"`
	IssueNumber        int        `json:"issue_number"`
	Title              string     `json:"title"`
	Acceptance         string     `json:"acceptance"`
	RewardCents        int64      `json:"reward_cents"`
	RequiredWorkerKind WorkerKind `json:"required_worker_kind"`
	SuggestedAgentType string     `json:"suggested_agent_type"`
	BountyType         string     `json:"bounty_type,omitempty"`
	Status             TaskStatus `json:"status"`
	WorkerKind         WorkerKind `json:"worker_kind,omitempty"`
	WorkerID           string     `json:"worker_id,omitempty"`
	AgentType          string     `json:"agent_type,omitempty"`
	ProofHash          string     `json:"proof_hash,omitempty"`
	IssueURL           string     `json:"issue_url,omitempty"`
	IssueState         string     `json:"issue_state,omitempty"`
	PullRequestURL     string     `json:"pull_request_url,omitempty"`
	ReviewEvidenceURL  string     `json:"review_evidence_url,omitempty"`
	ReviewNotes        string     `json:"review_notes,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	AcceptedAt         *time.Time `json:"accepted_at,omitempty"`
	SubmittedAt        *time.Time `json:"submitted_at,omitempty"`
}

type LedgerEntry struct {
	Sequence     int       `json:"sequence"`
	Type         string    `json:"type"`
	FromAccount  string    `json:"from_account,omitempty"`
	ToAccount    string    `json:"to_account,omitempty"`
	AmountCents  int64     `json:"amount_cents"`
	Reference    string    `json:"reference"`
	PreviousHash string    `json:"previous_hash"`
	EntryHash    string    `json:"entry_hash"`
	CreatedAt    time.Time `json:"created_at"`
}

type LedgerVerificationResponse struct {
	Valid          bool       `json:"valid"`
	EntryCount     int        `json:"entry_count"`
	LastSequence   int        `json:"last_sequence"`
	LastHash       string     `json:"last_hash"`
	BrokenSequence int        `json:"broken_sequence,omitempty"`
	Error          string     `json:"error,omitempty"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty"`
}

type LedgerProtocolResponse struct {
	ProtocolVersion string                     `json:"protocol_version"`
	Kind            string                     `json:"kind"`
	TokenSymbol     string                     `json:"token_symbol"`
	Verification    LedgerVerificationResponse `json:"verification"`
	Entries         []LedgerEntry              `json:"entries"`
}

type PublicLedgerProofResponse struct {
	ProtocolVersion   string                 `json:"protocol_version"`
	Kind              string                 `json:"kind"`
	TokenSymbol       string                 `json:"token_symbol"`
	Valid             bool                   `json:"valid"`
	EntryCount        int                    `json:"entry_count"`
	VerifiedCount     int                    `json:"verified_count"`
	BrokenCount       int                    `json:"broken_count"`
	RootHash          string                 `json:"root_hash"`
	PublicRootHash    string                 `json:"public_root_hash"`
	ContractReference string                 `json:"contract_reference"`
	GeneratedAt       time.Time              `json:"generated_at"`
	Entries           []PublicLedgerProofRow `json:"entries"`
}

type PublicLedgerProofRow struct {
	Sequence           int       `json:"sequence"`
	Type               string    `json:"type"`
	AmountCents        int64     `json:"amount_cents"`
	Reference          string    `json:"reference"`
	EntryHash          string    `json:"entry_hash"`
	PublicHash         string    `json:"public_hash"`
	PreviousHash       string    `json:"previous_hash"`
	PublicPreviousHash string    `json:"public_previous_hash"`
	Valid              bool      `json:"valid"`
	CreatedAt          time.Time `json:"created_at"`
}

type PublicTokenEconomyResponse struct {
	ProtocolVersion string                   `json:"protocol_version"`
	Kind            string                   `json:"kind"`
	TokenSymbol     string                   `json:"token_symbol"`
	Stats           PublicTokenEconomyStats  `json:"stats"`
	Totals          PublicTokenEconomyTotals `json:"totals"`
	Balances        []PublicTokenBalance     `json:"balances"`
	Flows           []PublicTokenFlow        `json:"flows"`
	RecentEntries   []LedgerEntry            `json:"recent_entries"`
}

type PublicTokenEconomyStats struct {
	LedgerEntryCount int        `json:"ledger_entry_count"`
	TokenEventCount  int        `json:"token_event_count"`
	EscrowEventCount int        `json:"escrow_event_count"`
	PayoutCount      int        `json:"payout_count"`
	AirdropCount     int        `json:"airdrop_count"`
	PresaleCount     int        `json:"presale_count"`
	BalanceCount     int        `json:"balance_count"`
	FlowCount        int        `json:"flow_count"`
	UpdatedAt        *time.Time `json:"updated_at,omitempty"`
}

type PublicTokenEconomyTotals struct {
	VerifiedFundingCents  int64 `json:"verified_funding_cents"`
	MintedCents           int64 `json:"minted_cents"`
	PlatformFeeCents      int64 `json:"platform_fee_cents"`
	TreasuryBalanceCents  int64 `json:"treasury_balance_cents"`
	ProjectReserveCents   int64 `json:"project_reserve_cents"`
	TaskReserveCents      int64 `json:"task_reserve_cents"`
	ReleasedCents         int64 `json:"released_cents"`
	ManualCreditCents     int64 `json:"manual_credit_cents"`
	AirdropClaimCents     int64 `json:"airdrop_claim_cents"`
	PresaleReserveCents   int64 `json:"presale_reserve_cents"`
	RemainingReserveCents int64 `json:"remaining_reserve_cents"`
	TokenSupplyCents      int64 `json:"token_supply_cents"`
}

type PublicTokenBalance struct {
	ID          string     `json:"id"`
	Label       string     `json:"label"`
	Role        string     `json:"role"`
	AmountCents int64      `json:"amount_cents"`
	EntryCount  int        `json:"entry_count"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

type PublicTokenFlow struct {
	Type           string     `json:"type"`
	Label          string     `json:"label"`
	AmountCents    int64      `json:"amount_cents"`
	Count          int        `json:"count"`
	LatestSequence int        `json:"latest_sequence"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty"`
}

type RegisterRequest struct {
	Name        string `json:"name"`
	CompanyName string `json:"company_name"`
	Email       string `json:"email"`
	Password    string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type PasswordResetRequest struct {
	Email string `json:"email"`
}

type PasswordResetResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type GitHubAuthRequest struct {
	Code          string `json:"code"`
	RedirectURI   string `json:"redirect_uri"`
	WalletAddress string `json:"wallet_address,omitempty"`
	RecoveryCode  string `json:"recovery_code,omitempty"`
}

type GitHubAuthProfile struct {
	ID        string
	Username  string
	Name      string
	Email     string
	AvatarURL string
}

type CreateWalletRequest struct {
	Label string `json:"label,omitempty"`
}

type CreateWalletResponse struct {
	Address      string        `json:"address"`
	RecoveryCode string        `json:"recovery_code"`
	Wallet       WalletSummary `json:"wallet"`
}

type LinkWalletRequest struct {
	Address      string `json:"address"`
	RecoveryCode string `json:"recovery_code,omitempty"`
}

type WalletSummary struct {
	Address          string     `json:"address"`
	Account          string     `json:"account"`
	Chain            string     `json:"chain,omitempty"`
	LegacyAddress    string     `json:"legacy_address,omitempty"`
	BalanceCents     int64      `json:"balance_cents"`
	ReceivedCents    int64      `json:"received_cents"`
	SentCents        int64      `json:"sent_cents"`
	TransactionCount int        `json:"transaction_count"`
	LinkedAccounts   []string   `json:"linked_accounts"`
	GitHubUsername   string     `json:"github_username,omitempty"`
	OwnerLinked      bool       `json:"owner_linked"`
	CreatedAt        time.Time  `json:"created_at"`
	LinkedAt         *time.Time `json:"linked_at,omitempty"`
}

type CreateWalletMigrationRequest struct {
	LegacyChain   string `json:"legacy_chain"`
	LegacyAddress string `json:"legacy_address"`
	SolanaWallet  string `json:"solana_wallet,omitempty"`
}

type WalletMigrationContractArgs struct {
	LegacyChain       string `json:"legacy_chain"`
	LegacyAddressHash string `json:"legacy_address_hash"`
	SolanaWallet      string `json:"solana_wallet"`
}

type WalletMigrationContract struct {
	Network          string                      `json:"network"`
	ProgramID        string                      `json:"program_id"`
	ProgramReady     bool                        `json:"program_ready"`
	Instruction      string                      `json:"instruction"`
	PDASeeds         []string                    `json:"pda_seeds"`
	PDASeedFormats   []string                    `json:"pda_seed_formats"`
	Args             WalletMigrationContractArgs `json:"args"`
	TokenMint        string                      `json:"token_mint,omitempty"`
	TreasuryReceiver string                      `json:"treasury_receiver,omitempty"`
}

type WalletMigrationResponse struct {
	ProtocolVersion   string                  `json:"protocol_version"`
	Kind              string                  `json:"kind"`
	MigrationID       string                  `json:"migration_id"`
	Status            string                  `json:"status"`
	LegacyChain       string                  `json:"legacy_chain"`
	LegacyAddress     string                  `json:"legacy_address"`
	LegacyAddressHash string                  `json:"legacy_address_hash"`
	TargetChain       string                  `json:"target_chain"`
	TargetAddress     string                  `json:"target_address"`
	TargetAccount     string                  `json:"target_account"`
	TokenSymbol       string                  `json:"token_symbol"`
	RequiredProofs    []string                `json:"required_proofs"`
	Contract          WalletMigrationContract `json:"contract"`
	Wallet            WalletSummary           `json:"wallet"`
	CreatedAt         time.Time               `json:"created_at"`
}

type AdminUpdateUserRequest struct {
	Name        string   `json:"name"`
	CompanyName string   `json:"company_name"`
	Email       string   `json:"email"`
	Role        UserRole `json:"role"`
	Password    string   `json:"password,omitempty"`
}

type AuthResponse struct {
	Token string     `json:"token"`
	User  PublicUser `json:"user"`
}

type CreateProjectRequest struct {
	Title            string        `json:"title"`
	ClientName       string        `json:"client_name"`
	CompanyName      string        `json:"company_name"`
	ClientEmail      string        `json:"client_email"`
	Phone            string        `json:"phone"`
	SiteType         string        `json:"site_type"`
	PackageTier      string        `json:"package_tier"`
	Timeline         string        `json:"timeline"`
	Brief            string        `json:"brief"`
	BudgetCents      int64         `json:"budget_cents"`
	PaymentMethod    PaymentMethod `json:"payment_method"`
	PaymentReference string        `json:"payment_reference"`
	AttachmentIDs    []string      `json:"attachment_ids"`
	SourceRepoURL    string        `json:"source_repo_url,omitempty"`
	AllowAgents      *bool         `json:"allow_agents,omitempty"`
}

type ProjectPriceEvaluationRequest struct {
	Title                string   `json:"title"`
	Description          string   `json:"description"`
	ProjectType          string   `json:"project_type"`
	Requirements         string   `json:"requirements"`
	Deliverables         []string `json:"deliverables"`
	Timeline             string   `json:"timeline"`
	TechStack            string   `json:"tech_stack"`
	Complexity           string   `json:"complexity"`
	Constraints          string   `json:"constraints"`
	ReferenceBudgetCents int64    `json:"reference_budget_cents"`
}

type ProjectPriceEvaluationResponse struct {
	ProtocolVersion     string               `json:"protocol_version"`
	Kind                string               `json:"kind"`
	SuggestedPriceCents int64                `json:"suggested_price_cents"`
	SuggestedRange      PriceRange           `json:"suggested_range"`
	Confidence          string               `json:"confidence"`
	Breakdown           []PriceBreakdownItem `json:"breakdown"`
	Assumptions         []string             `json:"assumptions"`
	Risks               []string             `json:"risks"`
	Editable            bool                 `json:"editable"`
}

type PriceRange struct {
	LowCents  int64 `json:"low_cents"`
	HighCents int64 `json:"high_cents"`
}

type PriceBreakdownItem struct {
	Category    string `json:"category"`
	AmountCents int64  `json:"amount_cents"`
	Reason      string `json:"reason"`
}

type AcceptTaskRequest struct {
	WorkerKind WorkerKind `json:"worker_kind"`
	WorkerID   string     `json:"worker_id"`
	AgentType  string     `json:"agent_type"`
}

type TaskClaimResponse struct {
	ProtocolVersion string     `json:"protocol_version"`
	Kind            string     `json:"kind"`
	ID              string     `json:"id"`
	ClaimID         string     `json:"claim_id"`
	TaskID          string     `json:"task_id"`
	ProjectID       string     `json:"project_id"`
	IssueNumber     int        `json:"issue_number,omitempty"`
	Title           string     `json:"title"`
	Status          TaskStatus `json:"status"`
	WorkerKind      WorkerKind `json:"worker_kind"`
	WorkerID        string     `json:"worker_id"`
	AgentType       string     `json:"agent_type,omitempty"`
	RewardCents     int64      `json:"reward_cents"`
	ProofHash       string     `json:"proof_hash,omitempty"`
	AcceptedAt      *time.Time `json:"accepted_at,omitempty"`
	Task            Task       `json:"task"`
}

type TaskSubmissionRequest struct {
	PullRequestURL    string `json:"pull_request_url"`
	EvidenceURL       string `json:"evidence_url"`
	ReviewEvidenceURL string `json:"review_evidence_url"`
	Notes             string `json:"notes"`
	ReviewNotes       string `json:"review_notes"`
}

type TaskSubmissionResponse struct {
	ProtocolVersion   string     `json:"protocol_version"`
	Kind              string     `json:"kind"`
	ID                string     `json:"id"`
	ClaimID           string     `json:"claim_id"`
	TaskID            string     `json:"task_id"`
	ProjectID         string     `json:"project_id"`
	IssueNumber       int        `json:"issue_number,omitempty"`
	Title             string     `json:"title"`
	Status            string     `json:"status"`
	WorkerKind        WorkerKind `json:"worker_kind"`
	WorkerID          string     `json:"worker_id"`
	AgentType         string     `json:"agent_type,omitempty"`
	PullRequestURL    string     `json:"pull_request_url,omitempty"`
	ReviewEvidenceURL string     `json:"review_evidence_url,omitempty"`
	ReviewNotes       string     `json:"review_notes,omitempty"`
	SubmittedAt       time.Time  `json:"submitted_at"`
	Task              Task       `json:"task"`
}

type TaskReviewRequest struct {
	Notes       string `json:"notes"`
	ReviewNotes string `json:"review_notes"`
	Reason      string `json:"reason"`
}

type TaskReviewResponse struct {
	ProtocolVersion string     `json:"protocol_version"`
	Kind            string     `json:"kind"`
	ID              string     `json:"id"`
	ClaimID         string     `json:"claim_id"`
	TaskID          string     `json:"task_id"`
	ProjectID       string     `json:"project_id"`
	IssueNumber     int        `json:"issue_number,omitempty"`
	Title           string     `json:"title"`
	Decision        string     `json:"decision"`
	Status          TaskStatus `json:"status"`
	WorkerKind      WorkerKind `json:"worker_kind"`
	WorkerID        string     `json:"worker_id"`
	AgentType       string     `json:"agent_type,omitempty"`
	ReviewNotes     string     `json:"review_notes"`
	RequestedAt     time.Time  `json:"requested_at"`
	Task            Task       `json:"task"`
}

type AdminTaskPullRequestsResponse struct {
	TaskID       string                 `json:"task_id"`
	IssueNumber  int                    `json:"issue_number"`
	IssueURL     string                 `json:"issue_url,omitempty"`
	Repository   string                 `json:"repository"`
	PullRequests []AdminTaskPullRequest `json:"pull_requests"`
}

type AdminTaskPullRequest struct {
	Number         int                       `json:"number"`
	Title          string                    `json:"title"`
	Body           string                    `json:"-"`
	State          string                    `json:"state"`
	HTMLURL        string                    `json:"html_url"`
	MergeURL       string                    `json:"merge_url,omitempty"`
	Author         string                    `json:"author"`
	Draft          bool                      `json:"draft"`
	Merged         bool                      `json:"merged"`
	MergeableState string                    `json:"mergeable_state,omitempty"`
	BaseRef        string                    `json:"base_ref,omitempty"`
	HeadRef        string                    `json:"head_ref,omitempty"`
	Labels         []string                  `json:"labels,omitempty"`
	ChangedFiles   []AdminPullRequestFile    `json:"changed_files,omitempty"`
	Readiness      AdminPullRequestReadiness `json:"readiness"`
	CreatedAt      time.Time                 `json:"created_at"`
	UpdatedAt      time.Time                 `json:"updated_at"`
	MergedAt       *time.Time                `json:"merged_at,omitempty"`
}

type AdminPullRequestFile struct {
	Path      string `json:"path"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

type AdminPullRequestReadiness struct {
	Status    string   `json:"status"`
	CanMerge  bool     `json:"can_merge"`
	RiskLevel string   `json:"risk_level"`
	Blockers  []string `json:"blockers,omitempty"`
	Warnings  []string `json:"warnings,omitempty"`
	Signals   []string `json:"signals,omitempty"`
}

type ProjectPullRequestsResponse struct {
	ProtocolVersion string                    `json:"protocol_version"`
	Kind            string                    `json:"kind"`
	ProjectID       string                    `json:"project_id"`
	ProjectTitle    string                    `json:"project_title"`
	Stats           ProjectPullRequestStats   `json:"stats"`
	Tasks           []ProjectTaskPullRequests `json:"tasks"`
	UpdatedAt       time.Time                 `json:"updated_at"`
}

type ProjectPullRequestStats struct {
	TaskCount              int `json:"task_count"`
	LinkedTaskCount        int `json:"linked_task_count"`
	PullRequestCount       int `json:"pull_request_count"`
	OpenPullRequestCount   int `json:"open_pull_request_count"`
	MergedPullRequestCount int `json:"merged_pull_request_count"`
	ReadyCount             int `json:"ready_count"`
	NeedsReviewCount       int `json:"needs_review_count"`
	BlockedCount           int `json:"blocked_count"`
	ErrorCount             int `json:"error_count"`
	AutoReleaseReadyCount  int `json:"auto_release_ready_count"`
}

type ProjectTaskPullRequests struct {
	TaskID            string                      `json:"task_id,omitempty"`
	IssueNumber       int                         `json:"issue_number"`
	Title             string                      `json:"title"`
	Status            string                      `json:"status"`
	RewardCents       int64                       `json:"reward_cents,omitempty"`
	WorkerKind        WorkerKind                  `json:"worker_kind,omitempty"`
	WorkerID          string                      `json:"worker_id,omitempty"`
	AgentType         string                      `json:"agent_type,omitempty"`
	IssueURL          string                      `json:"issue_url,omitempty"`
	Repository        string                      `json:"repository,omitempty"`
	MonitorStatus     string                      `json:"monitor_status"`
	MonitorError      string                      `json:"monitor_error,omitempty"`
	ReviewPacket      map[string]any              `json:"review_packet,omitempty"`
	ReleasePacket     map[string]any              `json:"release_packet,omitempty"`
	AutoReleasePacket map[string]any              `json:"auto_release_packet,omitempty"`
	PullRequests      []ProjectPullRequestSummary `json:"pull_requests"`
	UpdatedAt         time.Time                   `json:"updated_at"`
}

type ProjectPullRequestSummary struct {
	Number         int                       `json:"number"`
	Title          string                    `json:"title"`
	State          string                    `json:"state"`
	HTMLURL        string                    `json:"html_url"`
	MergeURL       string                    `json:"merge_url,omitempty"`
	Author         string                    `json:"author"`
	Draft          bool                      `json:"draft"`
	Merged         bool                      `json:"merged"`
	MergeableState string                    `json:"mergeable_state,omitempty"`
	BaseRef        string                    `json:"base_ref,omitempty"`
	HeadRef        string                    `json:"head_ref,omitempty"`
	Labels         []string                  `json:"labels,omitempty"`
	Readiness      AdminPullRequestReadiness `json:"readiness"`
	CreatedAt      time.Time                 `json:"created_at"`
	UpdatedAt      time.Time                 `json:"updated_at"`
	MergedAt       *time.Time                `json:"merged_at,omitempty"`
}

type AdminMergeTaskPullRequestRequest struct {
	RewardMRG   int64  `json:"reward_mrg"`
	RewardCents int64  `json:"reward_cents,omitempty"`
	BountyType  string `json:"bounty_type"`
}

type AdminMergeTaskPullRequestResponse struct {
	Task         *Task                `json:"task"`
	PullRequest  AdminTaskPullRequest `json:"pull_request"`
	WorkerID     string               `json:"worker_id"`
	RewardMRG    int64                `json:"reward_mrg"`
	BountyType   string               `json:"bounty_type"`
	AdminURL     string               `json:"admin_url"`
	CreditURL    string               `json:"credit_url,omitempty"`
	CommentURL   string               `json:"comment_url,omitempty"`
	CommentError string               `json:"comment_error,omitempty"`
}

type AdminManualCreditRequest struct {
	WorkerID    string `json:"worker_id"`
	RewardMRG   int64  `json:"reward_mrg"`
	AmountMRG   int64  `json:"amount_mrg,omitempty"`
	RewardCents int64  `json:"reward_cents,omitempty"`
	BountyType  string `json:"bounty_type"`
	TaskID      string `json:"task_id,omitempty"`
	PRURL       string `json:"pr_url,omitempty"`
	PRTitle     string `json:"pr_title,omitempty"`
	Reference   string `json:"reference,omitempty"`
	Note        string `json:"note,omitempty"`
}

type AdminManualCreditResponse struct {
	LedgerEntry LedgerEntry `json:"ledger_entry"`
	WorkerID    string      `json:"worker_id"`
	RewardMRG   int64       `json:"reward_mrg"`
	BountyType  string      `json:"bounty_type"`
	CreditURL   string      `json:"credit_url,omitempty"`
}

type StatusResponse struct {
	Service      string `json:"service"`
	Version      string `json:"version"`
	Environment  string `json:"environment"`
	TokenSymbol  string `json:"token_symbol"`
	PaymentMode  string `json:"payment_mode"`
	RepoProvider string `json:"repo_provider"`
}

type RuntimeConfigResponse struct {
	Environment       string              `json:"environment"`
	TokenSymbol       string              `json:"token_symbol"`
	PaymentMode       string              `json:"payment_mode"`
	RepoProvider      string              `json:"repo_provider"`
	GitHubOAuthReady  bool                `json:"github_oauth_ready"`
	GitHubOAuthClient string              `json:"github_oauth_client_id,omitempty"`
	GoogleOAuthReady  bool                `json:"google_oauth_ready"`
	PayPalReady       bool                `json:"paypal_ready"`
	CryptoReady       bool                `json:"crypto_ready"`
	StripeReady       bool                `json:"stripe_ready"`
	CardReady         bool                `json:"card_ready"`
	StripePublicKey   string              `json:"stripe_publishable_key,omitempty"`
	CardPublicKey     string              `json:"card_public_key,omitempty"`
	PaymentRails      []PaymentRailOption `json:"payment_rails"`
	GitHubReady       bool                `json:"github_ready"`
	SMTPReady         bool                `json:"smtp_ready"`
	DevPaymentEnabled bool                `json:"dev_payment_enabled"`
	DevPaymentCode    string              `json:"dev_payment_code,omitempty"`
	CryptoReceiver    string              `json:"crypto_receiver,omitempty"`
	CryptoAsset       string              `json:"crypto_asset,omitempty"`
	CryptoToken       string              `json:"crypto_token,omitempty"`
	BountyRoot        string              `json:"bounty_root,omitempty"`
	UploadRoot        string              `json:"upload_root,omitempty"`
	AdminBootstrap    bool                `json:"admin_bootstrap"`
	PrimaryDomain     string              `json:"primary_domain,omitempty"`
	AdminDomain       string              `json:"admin_domain,omitempty"`
	ScanDomain        string              `json:"scan_domain,omitempty"`
	SSLReviewDomains  []string            `json:"ssl_review_domains,omitempty"`
}

type PaymentRailOption struct {
	ID                string `json:"id"`
	Label             string `json:"label"`
	Method            string `json:"method"`
	Caption           string `json:"caption"`
	Enabled           bool   `json:"enabled"`
	Ready             bool   `json:"ready"`
	DisabledReason    string `json:"disabled_reason,omitempty"`
	RequiresReference bool   `json:"requires_reference"`
	PublicKey         string `json:"public_key,omitempty"`
	Asset             string `json:"asset,omitempty"`
	Receiver          string `json:"receiver,omitempty"`
	TokenContract     string `json:"token_contract,omitempty"`
}

type AdminSettings struct {
	LLMProvider       string    `json:"llm_provider"`
	LLMModel          string    `json:"llm_model"`
	GeminiReviewModel string    `json:"gemini_review_model"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type LLMProviderOption struct {
	ID     string   `json:"id"`
	Label  string   `json:"label"`
	Models []string `json:"models"`
}

type AdminSettingsResponse struct {
	LLMProvider              string              `json:"llm_provider"`
	LLMModel                 string              `json:"llm_model"`
	LLMProviderOptions       []LLMProviderOption `json:"llm_provider_options"`
	GeminiReviewModel        string              `json:"gemini_review_model"`
	GeminiReviewModelOptions []string            `json:"gemini_review_model_options"`
	UpdatedAt                time.Time           `json:"updated_at"`
}

type UpdateAdminSettingsRequest struct {
	LLMProvider       string `json:"llm_provider"`
	LLMModel          string `json:"llm_model"`
	GeminiReviewModel string `json:"gemini_review_model"`
}

type CreatePayPalOrderRequest struct {
	AmountCents     int64  `json:"amount_cents"`
	Description     string `json:"description"`
	Flow            string `json:"flow,omitempty"`
	ProjectID       string `json:"project_id,omitempty"`
	SuggestedTaskID string `json:"suggested_task_id,omitempty"`
	ReturnURL       string `json:"return_url"`
	CancelURL       string `json:"cancel_url"`
}

type CreatePayPalOrderResponse struct {
	OrderID          string `json:"order_id"`
	PaymentReference string `json:"payment_reference"`
	ApprovalURL      string `json:"approval_url"`
	Status           string `json:"status"`
	Provider         string `json:"provider"`
	Flow             string `json:"flow,omitempty"`
	AmountCents      int64  `json:"amount_cents,omitempty"`
	Currency         string `json:"currency,omitempty"`
}

type CreateCardPaymentIntentRequest struct {
	AmountCents int64  `json:"amount_cents"`
	Description string `json:"description"`
	Flow        string `json:"flow,omitempty"`
}

const (
	PaymentOrderFlowProjectFunding        = "project_funding"
	PaymentOrderFlowRepositoryTaskFunding = "repo_task_funding"
)

type PaymentOrderIntent struct {
	OrderID         string     `json:"order_id"`
	Provider        string     `json:"provider"`
	Flow            string     `json:"flow"`
	UserID          string     `json:"user_id"`
	ProjectID       string     `json:"project_id,omitempty"`
	SuggestedTaskID string     `json:"suggested_task_id,omitempty"`
	AmountCents     int64      `json:"amount_cents"`
	Currency        string     `json:"currency"`
	Description     string     `json:"description,omitempty"`
	Status          string     `json:"status"`
	ApprovalURL     string     `json:"approval_url,omitempty"`
	ReturnURL       string     `json:"return_url,omitempty"`
	CancelURL       string     `json:"cancel_url,omitempty"`
	CaptureID       string     `json:"capture_id,omitempty"`
	WebhookEventID  string     `json:"webhook_event_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	CapturedAt      *time.Time `json:"captured_at,omitempty"`
}

type CreateCardPaymentIntentResponse struct {
	PaymentReference string `json:"payment_reference"`
	PaymentIntentID  string `json:"payment_intent_id,omitempty"`
	ClientSecret     string `json:"client_secret,omitempty"`
	Status           string `json:"status"`
	Provider         string `json:"provider"`
	Mode             string `json:"mode"`
	PublicKey        string `json:"public_key,omitempty"`
	Brand            string `json:"brand,omitempty"`
	Last4            string `json:"last4,omitempty"`
}

type ImportRepoIssuesRequest struct {
	RepoURL string `json:"repo_url"`
}

type ImportRepoIssuesResponse struct {
	ProtocolVersion     string               `json:"protocol_version"`
	Kind                string               `json:"kind"`
	Owner               string               `json:"owner"`
	Name                string               `json:"name"`
	RepoURL             string               `json:"repo_url"`
	IssueCount          int                  `json:"issue_count"`
	TotalEstimatedCents int64                `json:"total_estimated_cents"`
	TotalEstimatedHours float64              `json:"total_estimated_hours"`
	Issues              []*ImportedRepoIssue `json:"issues"`
}

type ProjectIssueSyncResponse struct {
	ProtocolVersion    string                    `json:"protocol_version"`
	Kind               string                    `json:"kind"`
	ProjectID          string                    `json:"project_id"`
	ProjectTitle       string                    `json:"project_title"`
	SourceRepoURL      string                    `json:"source_repo_url"`
	ImportedIssueCount int                       `json:"imported_issue_count"`
	AddedTaskCount     int                       `json:"added_task_count"`
	UpdatedTaskCount   int                       `json:"updated_task_count"`
	OpenIssueCount     int                       `json:"open_issue_count"`
	ClosedIssueCount   int                       `json:"closed_issue_count"`
	IssueMappings      []ProjectIssueSyncMapping `json:"issue_mappings"`
	SyncedAt           time.Time                 `json:"synced_at"`
}

type ProjectIssueSyncMapping struct {
	IssueNumber        int                 `json:"issue_number"`
	IssueTitle         string              `json:"issue_title"`
	IssueState         string              `json:"issue_state"`
	IssueURL           string              `json:"issue_url,omitempty"`
	SyncStatus         string              `json:"sync_status"`
	TaskID             string              `json:"task_id"`
	TaskTitle          string              `json:"task_title"`
	TaskStatus         TaskStatus          `json:"task_status"`
	ClaimID            string              `json:"claim_id"`
	ClaimEndpoint      string              `json:"claim_endpoint"`
	TaskProtocolURL    string              `json:"task_protocol_url"`
	ActionEndpoint     string              `json:"action_endpoint"`
	RewardCents        int64               `json:"reward_cents"`
	RewardMRG          float64             `json:"reward_mrg"`
	EstimatedHours     float64             `json:"estimated_hours"`
	RequiredWorkerKind WorkerKind          `json:"required_worker_kind"`
	SuggestedAgentType string              `json:"suggested_agent_type,omitempty"`
	Routing            ProjectRoutingRoute `json:"routing"`
}

type ImportedRepoIssue struct {
	Number             int        `json:"number"`
	Title              string     `json:"title"`
	State              string     `json:"state"`
	URL                string     `json:"url"`
	Labels             []string   `json:"labels"`
	Comments           int        `json:"comments"`
	Score              int        `json:"score"`
	Complexity         string     `json:"complexity"`
	EstimatedCents     int64      `json:"estimated_cents"`
	EstimatedHours     float64    `json:"estimated_hours"`
	RequiredWorkerKind WorkerKind `json:"required_worker_kind"`
	SuggestedAgentType string     `json:"suggested_agent_type"`
	Reasons            []string   `json:"reasons"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type MarketplaceResponse struct {
	ProtocolVersion string                    `json:"protocol_version"`
	Kind            string                    `json:"kind"`
	Stats           MarketplaceStats          `json:"stats"`
	Projects        []*MarketplaceProject     `json:"projects"`
	Bounties        []*MarketplaceBounty      `json:"bounties"`
	Contributors    []*MarketplaceContributor `json:"contributors"`
	Agents          []*MarketplaceAgent       `json:"agents"`
}

type MarketplaceStats struct {
	ProjectCount      int        `json:"project_count"`
	OpenTaskCount     int        `json:"open_task_count"`
	AcceptedTaskCount int        `json:"accepted_task_count"`
	LedgerEntryCount  int        `json:"ledger_entry_count"`
	TotalBudgetCents  int64      `json:"total_budget_cents"`
	WorkPoolCents     int64      `json:"work_pool_cents"`
	TokenSymbol       string     `json:"token_symbol"`
	UpdatedAt         *time.Time `json:"updated_at,omitempty"`
}

type MarketplaceProject struct {
	ID                string        `json:"id"`
	Title             string        `json:"title"`
	Brief             string        `json:"brief"`
	SiteType          string        `json:"site_type,omitempty"`
	PackageTier       string        `json:"package_tier,omitempty"`
	Timeline          string        `json:"timeline,omitempty"`
	Status            ProjectStatus `json:"status"`
	ClientDisplayName string        `json:"client_display_name"`
	BountyRepoName    string        `json:"bounty_repo_name,omitempty"`
	RepoProvider      string        `json:"repo_provider,omitempty"`
	RepoURL           string        `json:"repo_url,omitempty"`
	BudgetCents       int64         `json:"budget_cents"`
	WorkPoolCents     int64         `json:"work_pool_cents"`
	TaskCount         int           `json:"task_count"`
	OpenTaskCount     int           `json:"open_task_count"`
	AcceptedTaskCount int           `json:"accepted_task_count"`
	Tags              []string      `json:"tags"`
	CreatedAt         time.Time     `json:"created_at"`
}

type MarketplaceBounty struct {
	ID                 string          `json:"id"`
	ClaimID            string          `json:"claim_id,omitempty"`
	ProjectID          string          `json:"project_id"`
	ProjectTitle       string          `json:"project_title"`
	IssueNumber        int             `json:"issue_number"`
	Title              string          `json:"title"`
	Acceptance         string          `json:"acceptance"`
	RewardCents        int64           `json:"reward_cents"`
	EstimatedHours     float64         `json:"estimated_hours,omitempty"`
	RequiredWorkerKind WorkerKind      `json:"required_worker_kind"`
	SuggestedAgentType string          `json:"suggested_agent_type,omitempty"`
	BountyType         string          `json:"bounty_type,omitempty"`
	EvidenceRequired   []string        `json:"evidence_required,omitempty"`
	SourceRepository   string          `json:"source_repository,omitempty"`
	IssueURL           string          `json:"issue_url,omitempty"`
	ProposalEndpoint   string          `json:"proposal_endpoint,omitempty"`
	ProposalPacket     *ProposalPacket `json:"proposal_packet,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
}

type ProposalPacket struct {
	CanClaim          bool                  `json:"can_claim"`
	Status            string                `json:"status"`
	ProposalEndpoint  string                `json:"proposal_endpoint"`
	ContextURLs       map[string]string     `json:"context_urls,omitempty"`
	Runbook           []AgentRunbookStep    `json:"runbook,omitempty"`
	Payload           CreateProposalRequest `json:"payload"`
	EvidenceChecklist []string              `json:"evidence_checklist,omitempty"`
	Warnings          []string              `json:"warnings,omitempty"`
}

type MarketplaceContributor struct {
	WorkerID        string     `json:"worker_id"`
	Name            string     `json:"name"`
	Kind            WorkerKind `json:"kind"`
	AgentType       string     `json:"agent_type,omitempty"`
	TaskCount       int        `json:"task_count"`
	EarnedCents     int64      `json:"earned_cents"`
	LastPaidAt      time.Time  `json:"last_paid_at"`
	ReputationScore int        `json:"reputation_score"`
	ReputationLevel string     `json:"reputation_level"`
	RiskLevel       string     `json:"risk_level"`
	Flags           []string   `json:"flags,omitempty"`
}

type MarketplaceAgent struct {
	Type               string     `json:"type"`
	Title              string     `json:"title"`
	WorkerKind         WorkerKind `json:"worker_kind"`
	Role               string     `json:"role,omitempty"`
	ParentAgentType    string     `json:"parent_agent_type,omitempty"`
	SubagentTypes      []string   `json:"subagent_types,omitempty"`
	DelegationEndpoint string     `json:"delegation_endpoint,omitempty"`
	Focus              []string   `json:"focus,omitempty"`
	TaskCount          int        `json:"task_count"`
	OpenTaskCount      int        `json:"open_task_count"`
	BudgetCents        int64      `json:"budget_cents"`
}

type PublicLiveFeedResponse struct {
	ProtocolVersion string               `json:"protocol_version"`
	Kind            string               `json:"kind"`
	Stats           PublicLiveFeedStats  `json:"stats"`
	Items           []PublicLiveFeedItem `json:"items"`
	Cursor          string               `json:"cursor,omitempty"`
	AfterID         string               `json:"after_id,omitempty"`
	Since           *time.Time           `json:"since,omitempty"`
	Replay          bool                 `json:"replay,omitempty"`
	CursorFound     bool                 `json:"cursor_found,omitempty"`
	HasMore         bool                 `json:"has_more,omitempty"`
	TotalItemCount  int                  `json:"total_item_count,omitempty"`
}

type PublicLiveFeedStats struct {
	ProjectCount           int        `json:"project_count"`
	OpenTaskCount          int        `json:"open_task_count"`
	AcceptedTaskCount      int        `json:"accepted_task_count"`
	ProposalCount          int        `json:"proposal_count,omitempty"`
	ActiveContributorCount int        `json:"active_contributor_count"`
	ActiveAgentCount       int        `json:"active_agent_count"`
	LedgerEntryCount       int        `json:"ledger_entry_count"`
	AIActionCount          int        `json:"ai_action_count"`
	TotalBudgetCents       int64      `json:"total_budget_cents"`
	TokenSymbol            string     `json:"token_symbol"`
	UpdatedAt              *time.Time `json:"updated_at,omitempty"`
}

type PublicLiveFeedItem struct {
	ID               string             `json:"id"`
	Type             string             `json:"type"`
	Title            string             `json:"title"`
	Body             string             `json:"body"`
	ProjectID        string             `json:"project_id,omitempty"`
	ProjectTitle     string             `json:"project_title,omitempty"`
	TaskID           string             `json:"task_id,omitempty"`
	Actor            string             `json:"actor,omitempty"`
	Action           string             `json:"action,omitempty"`
	AmountCents      int64              `json:"amount_cents,omitempty"`
	LedgerSequence   int                `json:"ledger_sequence,omitempty"`
	EntryHash        string             `json:"entry_hash,omitempty"`
	Reference        string             `json:"reference,omitempty"`
	EvidenceRequired []string           `json:"evidence_required,omitempty"`
	ContextURLs      []string           `json:"context_urls,omitempty"`
	Evidence         []string           `json:"evidence,omitempty"`
	Runbook          []string           `json:"runbook,omitempty"`
	Checks           []AgentActionCheck `json:"checks,omitempty"`
	SourceFindingID  string             `json:"source_finding_id,omitempty"`
	Signal           string             `json:"signal,omitempty"`
	Path             string             `json:"path,omitempty"`
	DelegatedBy      string             `json:"delegated_by,omitempty"`
	DesignAgent      string             `json:"design_agent,omitempty"`
	SubagentType     string             `json:"subagent_type,omitempty"`
	DelegationChain  []string           `json:"delegation_chain,omitempty"`
	URL              string             `json:"url,omitempty"`
	Status           string             `json:"status"`
	CreatedAt        time.Time          `json:"created_at"`
}

type PublicEventProtocolResponse struct {
	Stats          PublicLiveFeedStats     `json:"stats"`
	Events         []EventProtocolDocument `json:"events"`
	Cursor         string                  `json:"cursor,omitempty"`
	AfterID        string                  `json:"after_id,omitempty"`
	Since          *time.Time              `json:"since,omitempty"`
	Replay         bool                    `json:"replay,omitempty"`
	CursorFound    bool                    `json:"cursor_found,omitempty"`
	HasMore        bool                    `json:"has_more,omitempty"`
	TotalItemCount int                     `json:"total_item_count,omitempty"`
}

type PublicTaskProtocolResponse struct {
	Stats MarketplaceStats       `json:"stats"`
	Tasks []TaskProtocolDocument `json:"tasks"`
}

type PublicAgentProtocolResponse struct {
	Stats  MarketplaceStats        `json:"stats"`
	Agents []AgentProtocolDocument `json:"agents"`
}

type AgentQueueResponse struct {
	ProtocolVersion string            `json:"protocol_version"`
	Kind            string            `json:"kind"`
	Stats           AgentQueueStats   `json:"stats"`
	Agents          []AgentQueueAgent `json:"agents"`
	Tasks           []AgentQueueTask  `json:"tasks"`
}

type AgentQueueStats struct {
	TotalCount  int        `json:"total_count"`
	AgentCount  int        `json:"agent_count"`
	ReadyCount  int        `json:"ready_count"`
	RewardCents int64      `json:"reward_cents"`
	TokenSymbol string     `json:"token_symbol"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

type AgentQueueAgent struct {
	Type               string     `json:"type"`
	Title              string     `json:"title"`
	WorkerKind         WorkerKind `json:"worker_kind"`
	Role               string     `json:"role,omitempty"`
	ParentAgentType    string     `json:"parent_agent_type,omitempty"`
	SubagentTypes      []string   `json:"subagent_types,omitempty"`
	DelegationEndpoint string     `json:"delegation_endpoint,omitempty"`
	Focus              []string   `json:"focus,omitempty"`
	TaskCount          int        `json:"task_count"`
	OpenTaskCount      int        `json:"open_task_count"`
	BudgetCents        int64      `json:"budget_cents"`
	Status             string     `json:"status"`
	SupportedActions   []string   `json:"supported_actions"`
	QueueDepth         int        `json:"queue_depth"`
}

type AgentQueueTask struct {
	ID               string          `json:"id"`
	BountyID         string          `json:"bounty_id"`
	ProjectID        string          `json:"project_id"`
	ProjectTitle     string          `json:"project_title"`
	IssueNumber      int             `json:"issue_number"`
	Title            string          `json:"title"`
	Summary          string          `json:"summary"`
	RewardCents      int64           `json:"reward_cents"`
	WorkerKind       WorkerKind      `json:"worker_kind"`
	AgentType        string          `json:"agent_type,omitempty"`
	Readiness        string          `json:"readiness"`
	EvidenceRequired []string        `json:"evidence_required"`
	ClaimEndpoint    string          `json:"claim_endpoint"`
	ActionEndpoint   string          `json:"action_endpoint"`
	ProtocolURL      string          `json:"protocol_url"`
	WorkPacket       AgentWorkPacket `json:"work_packet"`
}

type AgentWorkPacket struct {
	ClaimEndpoint       string               `json:"claim_endpoint"`
	ActionEndpoint      string               `json:"action_endpoint"`
	SubmitEndpoint      string               `json:"submit_endpoint"`
	SupervisorAgentType string               `json:"supervisor_agent_type,omitempty"`
	SubagentType        string               `json:"subagent_type,omitempty"`
	DesignReviewAgent   string               `json:"design_review_agent,omitempty"`
	DelegationChain     []string             `json:"delegation_chain,omitempty"`
	ContextURLs         map[string]string    `json:"context_urls"`
	Runbook             []AgentRunbookStep   `json:"runbook"`
	ActionPayloads      []AgentActionPayload `json:"action_payloads"`
}

type AgentRunbookStep struct {
	Step     int    `json:"step"`
	Action   string `json:"action"`
	Label    string `json:"label"`
	Method   string `json:"method"`
	Endpoint string `json:"endpoint"`
}

type AgentActionPayload struct {
	Action   string         `json:"action"`
	Label    string         `json:"label"`
	Method   string         `json:"method"`
	Endpoint string         `json:"endpoint"`
	Body     map[string]any `json:"body"`
}

type PublicContributorProtocolResponse struct {
	Stats        MarketplaceStats              `json:"stats"`
	Contributors []ContributorProtocolDocument `json:"contributors"`
}

type ProtocolManifestResponse struct {
	ProtocolVersion string                     `json:"protocol_version"`
	Kind            string                     `json:"kind"`
	Schemas         []ProtocolManifestSchema   `json:"schemas"`
	Endpoints       []ProtocolManifestEndpoint `json:"endpoints"`
}

type ProtocolManifestSchema struct {
	Version     string `json:"version"`
	Kind        string `json:"kind"`
	SchemaURL   string `json:"schema_url"`
	Description string `json:"description"`
}

type ProtocolManifestEndpoint struct {
	Method      string `json:"method"`
	Path        string `json:"path"`
	Protocol    string `json:"protocol,omitempty"`
	Auth        string `json:"auth"`
	Description string `json:"description"`
}

type TaskProtocolDocument struct {
	ProtocolVersion    string         `json:"protocol_version"`
	Kind               string         `json:"kind"`
	ID                 string         `json:"id"`
	ProjectID          string         `json:"project_id,omitempty"`
	Title              string         `json:"title"`
	Summary            string         `json:"summary,omitempty"`
	SourceRepository   string         `json:"source_repository,omitempty"`
	IssueURL           string         `json:"issue_url,omitempty"`
	RewardMRG          float64        `json:"reward_mrg"`
	EstimatedHours     float64        `json:"estimated_hours,omitempty"`
	Complexity         string         `json:"complexity,omitempty"`
	RiskLevel          string         `json:"risk_level,omitempty"`
	BountyType         string         `json:"bounty_type,omitempty"`
	WorkerKind         WorkerKind     `json:"worker_kind"`
	AgentType          string         `json:"agent_type,omitempty"`
	AcceptanceCriteria []string       `json:"acceptance_criteria"`
	Dependencies       []string       `json:"dependencies,omitempty"`
	EvidenceRequired   []string       `json:"evidence_required,omitempty"`
	Tags               []string       `json:"tags,omitempty"`
	Metadata           map[string]any `json:"metadata,omitempty"`
}

type AgentProtocolDocument struct {
	ProtocolVersion    string         `json:"protocol_version"`
	Kind               string         `json:"kind"`
	ID                 string         `json:"id"`
	Type               string         `json:"type"`
	Title              string         `json:"title"`
	WorkerKind         WorkerKind     `json:"worker_kind"`
	Role               string         `json:"role,omitempty"`
	ParentAgentType    string         `json:"parent_agent_type,omitempty"`
	SubagentTypes      []string       `json:"subagent_types,omitempty"`
	DelegationEndpoint string         `json:"delegation_endpoint,omitempty"`
	Focus              []string       `json:"focus,omitempty"`
	SupportedActions   []string       `json:"supported_actions"`
	Capabilities       []string       `json:"capabilities"`
	TaskCount          int            `json:"task_count"`
	OpenTaskCount      int            `json:"open_task_count"`
	BudgetMRG          float64        `json:"budget_mrg"`
	Status             string         `json:"status"`
	OpenTaskIDs        []string       `json:"open_task_ids,omitempty"`
	Tags               []string       `json:"tags,omitempty"`
	Metadata           map[string]any `json:"metadata,omitempty"`
}

type ContributorProtocolDocument struct {
	ProtocolVersion    string         `json:"protocol_version"`
	Kind               string         `json:"kind"`
	ID                 string         `json:"id"`
	WorkerID           string         `json:"worker_id"`
	DisplayName        string         `json:"display_name"`
	WorkerKind         WorkerKind     `json:"worker_kind"`
	AgentType          string         `json:"agent_type,omitempty"`
	CompletedTaskCount int            `json:"completed_task_count"`
	EarnedMRG          float64        `json:"earned_mrg"`
	ReputationScore    int            `json:"reputation_score"`
	ReputationLevel    string         `json:"reputation_level"`
	RiskLevel          string         `json:"risk_level"`
	LastPaidAt         time.Time      `json:"last_paid_at"`
	MatchedTaskIDs     []string       `json:"matched_task_ids,omitempty"`
	Capabilities       []string       `json:"capabilities"`
	Flags              []string       `json:"flags,omitempty"`
	Tags               []string       `json:"tags,omitempty"`
	Metadata           map[string]any `json:"metadata,omitempty"`
}

type EventProtocolDocument struct {
	ProtocolVersion string         `json:"protocol_version"`
	Kind            string         `json:"kind"`
	ID              string         `json:"id"`
	Type            string         `json:"type"`
	OccurredAt      time.Time      `json:"occurred_at"`
	Actor           string         `json:"actor"`
	ProjectID       string         `json:"project_id,omitempty"`
	TaskID          string         `json:"task_id,omitempty"`
	Reference       string         `json:"reference,omitempty"`
	AmountMRG       *float64       `json:"amount_mrg,omitempty"`
	Payload         map[string]any `json:"payload,omitempty"`
}

type ProjectDeploymentResponse struct {
	ProtocolVersion string             `json:"protocol_version"`
	Kind            string             `json:"kind"`
	ProjectID       string             `json:"project_id"`
	ProjectTitle    string             `json:"project_title"`
	Status          string             `json:"status"`
	Progress        int                `json:"progress"`
	UpdatedAt       time.Time          `json:"updated_at"`
	Stages          []DeploymentStage  `json:"stages"`
	Signals         []DeploymentSignal `json:"signals"`
}

type ProjectEscrowResponse struct {
	ProtocolVersion     string              `json:"protocol_version"`
	Kind                string              `json:"kind"`
	ProjectID           string              `json:"project_id"`
	ProjectTitle        string              `json:"project_title"`
	TokenSymbol         string              `json:"token_symbol"`
	ReleaseStatus       string              `json:"release_status"`
	BudgetCents         int64               `json:"budget_cents"`
	FeeCents            int64               `json:"fee_cents"`
	WorkPoolCents       int64               `json:"work_pool_cents"`
	ProjectReserveCents int64               `json:"project_reserve_cents"`
	TaskReserveCents    int64               `json:"task_reserve_cents"`
	TaskPaymentCents    int64               `json:"task_payment_cents"`
	ManualCreditCents   int64               `json:"manual_credit_cents"`
	ReleasedCents       int64               `json:"released_cents"`
	RemainingCents      int64               `json:"remaining_cents"`
	OverdrawnCents      int64               `json:"overdrawn_cents"`
	UnallocatedCents    int64               `json:"unallocated_cents"`
	PaidTaskCount       int                 `json:"paid_task_count"`
	OpenTaskCount       int                 `json:"open_task_count"`
	UpdatedAt           time.Time           `json:"updated_at"`
	Tasks               []ProjectEscrowTask `json:"tasks"`
}

type ProjectEscrowTask struct {
	TaskID         string    `json:"task_id"`
	IssueNumber    int       `json:"issue_number"`
	Title          string    `json:"title"`
	Status         string    `json:"status"`
	ReleaseStatus  string    `json:"release_status"`
	RewardCents    int64     `json:"reward_cents"`
	PaidCents      int64     `json:"paid_cents"`
	RemainingCents int64     `json:"remaining_cents"`
	OverpaidCents  int64     `json:"overpaid_cents"`
	WorkerID       string    `json:"worker_id,omitempty"`
	ProofHash      string    `json:"proof_hash,omitempty"`
	IssueURL       string    `json:"issue_url,omitempty"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ProjectPayoutsResponse struct {
	ProtocolVersion string             `json:"protocol_version"`
	Kind            string             `json:"kind"`
	ProjectID       string             `json:"project_id"`
	ProjectTitle    string             `json:"project_title"`
	TokenSymbol     string             `json:"token_symbol"`
	ReleaseStatus   string             `json:"release_status"`
	WorkPoolCents   int64              `json:"work_pool_cents"`
	ReleasedCents   int64              `json:"released_cents"`
	RemainingCents  int64              `json:"remaining_cents"`
	OverdrawnCents  int64              `json:"overdrawn_cents"`
	TaskCount       int                `json:"task_count"`
	PaidTaskCount   int                `json:"paid_task_count"`
	OpenTaskCount   int                `json:"open_task_count"`
	ReleaseCount    int                `json:"release_count"`
	UpdatedAt       time.Time          `json:"updated_at"`
	Payouts         []ProjectPayoutRow `json:"payouts"`
}

type ProjectPayoutRow struct {
	TaskID           string     `json:"task_id"`
	IssueNumber      int        `json:"issue_number"`
	Title            string     `json:"title"`
	Type             string     `json:"type"`
	Status           string     `json:"status"`
	ReleaseStatus    string     `json:"release_status"`
	WorkerID         string     `json:"worker_id,omitempty"`
	PayoutAccount    string     `json:"payout_account,omitempty"`
	RewardCents      int64      `json:"reward_cents"`
	PaidCents        int64      `json:"paid_cents"`
	RemainingCents   int64      `json:"remaining_cents"`
	OverpaidCents    int64      `json:"overpaid_cents"`
	LedgerSequence   int        `json:"ledger_sequence,omitempty"`
	LedgerEntryCount int        `json:"ledger_entry_count"`
	EntryHash        string     `json:"entry_hash,omitempty"`
	ProofHash        string     `json:"proof_hash,omitempty"`
	Reference        string     `json:"reference,omitempty"`
	URL              string     `json:"url,omitempty"`
	ReleasedAt       *time.Time `json:"released_at,omitempty"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type ProjectAutoReleaseRequest struct {
	TaskIDs    []string                      `json:"task_ids"`
	Policy     string                        `json:"policy,omitempty"`
	Candidates []ProjectAutoReleaseCandidate `json:"candidates,omitempty"`
}

type ProjectAutoReleaseCandidate struct {
	TaskID            string     `json:"task_id"`
	WorkerKind        WorkerKind `json:"worker_kind"`
	WorkerID          string     `json:"worker_id"`
	AgentType         string     `json:"agent_type,omitempty"`
	RewardCents       int64      `json:"reward_cents"`
	Repository        string     `json:"repository,omitempty"`
	PullRequestNumber int        `json:"pull_request_number"`
	PullRequestURL    string     `json:"pull_request_url,omitempty"`
	PullRequestTitle  string     `json:"pull_request_title,omitempty"`
	ReadinessStatus   string     `json:"readiness_status"`
	CanMerge          bool       `json:"can_merge"`
	RiskLevel         string     `json:"risk_level"`
	DeploymentStatus  string     `json:"deployment_status,omitempty"`
	ValidationSignals []string   `json:"validation_signals,omitempty"`
	Draft             bool       `json:"draft"`
	CanRelease        bool       `json:"can_release"`
}

type ProjectAutoReleaseResponse struct {
	ProtocolVersion string                    `json:"protocol_version"`
	Kind            string                    `json:"kind"`
	ProjectID       string                    `json:"project_id"`
	Policy          string                    `json:"policy"`
	ReleasedCount   int                       `json:"released_count"`
	SkippedCount    int                       `json:"skipped_count"`
	Released        []TaskClaimResponse       `json:"released"`
	Skipped         []ProjectAutoReleaseSkip  `json:"skipped"`
	ReleaseProofs   []ProjectAutoReleaseProof `json:"release_proofs"`
	Payouts         ProjectPayoutsResponse    `json:"payouts"`
}

type ProjectAutoReleaseSkip struct {
	TaskID string `json:"task_id"`
	Reason string `json:"reason"`
}

type ProjectAutoReleaseProof struct {
	TaskID            string     `json:"task_id"`
	ClaimID           string     `json:"claim_id"`
	IssueNumber       int        `json:"issue_number"`
	WorkerKind        WorkerKind `json:"worker_kind"`
	WorkerID          string     `json:"worker_id"`
	AgentType         string     `json:"agent_type,omitempty"`
	PullRequestNumber int        `json:"pull_request_number"`
	PullRequestURL    string     `json:"pull_request_url"`
	ReadinessStatus   string     `json:"readiness_status"`
	RiskLevel         string     `json:"risk_level"`
	DeploymentStatus  string     `json:"deployment_status"`
	ValidationSignals []string   `json:"validation_signals,omitempty"`
	Policy            string     `json:"policy"`
	LedgerReference   string     `json:"ledger_reference"`
	ReleasedAt        time.Time  `json:"released_at"`
}

type DeploymentStage struct {
	ID                    string    `json:"id"`
	Title                 string    `json:"title"`
	Body                  string    `json:"body"`
	Status                string    `json:"status"`
	Tone                  string    `json:"tone"`
	SourceTaskIssueNumber int       `json:"source_task_issue_number,omitempty"`
	Reference             string    `json:"reference,omitempty"`
	URL                   string    `json:"url,omitempty"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type DeploymentSignal struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Status    string    `json:"status"`
	Reference string    `json:"reference,omitempty"`
	URL       string    `json:"url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type ProjectDashboardResponse struct {
	ProtocolVersion  string                        `json:"protocol_version"`
	Kind             string                        `json:"kind"`
	Project          ProjectDashboardOverview      `json:"project"`
	Escrow           ProjectEscrowResponse         `json:"escrow"`
	Payouts          ProjectPayoutsResponse        `json:"payouts"`
	Deployment       ProjectDeploymentResponse     `json:"deployment"`
	AIWorkflow       ProjectAIWorkflowResponse     `json:"ai_workflow"`
	TaskGraph        ProjectTaskGraphResponse      `json:"task_graph"`
	RepositoryScan   ProjectRepositoryScanResponse `json:"repository_scan"`
	PullRequests     ProjectPullRequestsResponse   `json:"pull_requests"`
	Proposals        []WorkerSubmittedProposal     `json:"proposals"`
	PullRequestError string                        `json:"pull_request_error,omitempty"`
	UpdatedAt        time.Time                     `json:"updated_at"`
}

type ProjectDashboardOverview struct {
	ProjectID         string        `json:"project_id"`
	Title             string        `json:"title"`
	Brief             string        `json:"brief"`
	SiteType          string        `json:"site_type,omitempty"`
	PackageTier       string        `json:"package_tier,omitempty"`
	Timeline          string        `json:"timeline,omitempty"`
	Status            ProjectStatus `json:"status"`
	RepoProvider      string        `json:"repo_provider,omitempty"`
	RepoURL           string        `json:"repo_url,omitempty"`
	BountyRepoName    string        `json:"bounty_repo_name,omitempty"`
	BudgetCents       int64         `json:"budget_cents"`
	FeeCents          int64         `json:"fee_cents"`
	WorkPoolCents     int64         `json:"work_pool_cents"`
	TaskCount         int           `json:"task_count"`
	OpenTaskCount     int           `json:"open_task_count"`
	AcceptedTaskCount int           `json:"accepted_task_count"`
	AgentTaskCount    int           `json:"agent_task_count"`
	HumanTaskCount    int           `json:"human_task_count"`
	HybridTaskCount   int           `json:"hybrid_task_count"`
	CreatedAt         time.Time     `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
}

type ProjectAIWorkflowResponse struct {
	ProtocolVersion string             `json:"protocol_version"`
	Kind            string             `json:"kind"`
	ProjectID       string             `json:"project_id"`
	ProjectTitle    string             `json:"project_title"`
	Status          string             `json:"status"`
	Progress        int                `json:"progress"`
	CurrentStep     string             `json:"current_step,omitempty"`
	TaskCount       int                `json:"task_count"`
	AgentTaskCount  int                `json:"agent_task_count"`
	HumanTaskCount  int                `json:"human_task_count"`
	HybridTaskCount int                `json:"hybrid_task_count"`
	AIActionCount   int                `json:"ai_action_count"`
	UpdatedAt       time.Time          `json:"updated_at"`
	Stages          []AIWorkflowStage  `json:"stages"`
	Signals         []AIWorkflowSignal `json:"signals"`
}

type ProjectTaskGraphResponse struct {
	ProjectID    string          `json:"project_id"`
	ProjectTitle string          `json:"project_title"`
	Status       string          `json:"status"`
	Progress     int             `json:"progress"`
	Stats        TaskGraphStats  `json:"stats"`
	Nodes        []TaskGraphNode `json:"nodes"`
	Edges        []TaskGraphEdge `json:"edges"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type ProjectRoutingResponse struct {
	ProtocolVersion string                `json:"protocol_version"`
	Kind            string                `json:"kind"`
	ProjectID       string                `json:"project_id"`
	ProjectTitle    string                `json:"project_title"`
	Status          string                `json:"status"`
	Summary         string                `json:"summary"`
	Stats           ProjectRoutingStats   `json:"stats"`
	Lanes           []ProjectRoutingLane  `json:"lanes"`
	Routes          []ProjectRoutingRoute `json:"routes"`
	UpdatedAt       time.Time             `json:"updated_at"`
}

type ProjectRoutingStats struct {
	TaskCount                 int `json:"task_count"`
	ReadyCount                int `json:"ready_count"`
	BlockedCount              int `json:"blocked_count"`
	ContributorCandidateCount int `json:"contributor_candidate_count"`
	AgentCandidateCount       int `json:"agent_candidate_count"`
	HumanLaneCount            int `json:"human_lane_count"`
	AgentLaneCount            int `json:"agent_lane_count"`
	HybridLaneCount           int `json:"hybrid_lane_count"`
}

type ProjectRoutingLane struct {
	ID             string     `json:"id"`
	Title          string     `json:"title"`
	WorkerKind     WorkerKind `json:"worker_kind"`
	AgentType      string     `json:"agent_type,omitempty"`
	RecommendedFor string     `json:"recommended_for"`
	TaskCount      int        `json:"task_count"`
	ReadyCount     int        `json:"ready_count"`
	BlockedCount   int        `json:"blocked_count"`
	RewardCents    int64      `json:"reward_cents"`
	Status         string     `json:"status"`
}

type ProjectRoutingRoute struct {
	ID                    string                     `json:"id"`
	TaskID                string                     `json:"task_id"`
	IssueNumber           int                        `json:"issue_number"`
	Title                 string                     `json:"title"`
	Lane                  string                     `json:"lane"`
	Status                string                     `json:"status"`
	Ready                 bool                       `json:"ready"`
	BlockedBy             []string                   `json:"blocked_by,omitempty"`
	RewardCents           int64                      `json:"reward_cents"`
	RequiredWorkerKind    WorkerKind                 `json:"required_worker_kind"`
	SuggestedAgentType    string                     `json:"suggested_agent_type,omitempty"`
	RecommendedNextAction string                     `json:"recommended_next_action"`
	MatchScore            int                        `json:"match_score"`
	RoutingReason         []string                   `json:"routing_reason,omitempty"`
	RecommendedAgent      *ProjectRoutingAgent       `json:"recommended_agent,omitempty"`
	RecommendedWorker     *ProjectRoutingContributor `json:"recommended_worker,omitempty"`
}

type ProjectRoutingAgent struct {
	Type       string `json:"type"`
	Title      string `json:"title"`
	Status     string `json:"status"`
	QueueDepth int    `json:"queue_depth"`
}

type ProjectRoutingContributor struct {
	WorkerID        string     `json:"worker_id"`
	Name            string     `json:"name"`
	Kind            WorkerKind `json:"kind"`
	ReputationScore int        `json:"reputation_score"`
	RiskLevel       string     `json:"risk_level"`
}

type WorkflowProtocolDocument struct {
	ProtocolVersion string                     `json:"protocol_version"`
	Kind            string                     `json:"kind"`
	ID              string                     `json:"id"`
	ProjectID       string                     `json:"project_id"`
	Status          string                     `json:"status,omitempty"`
	Progress        int                        `json:"progress,omitempty"`
	CurrentStep     string                     `json:"current_step,omitempty"`
	Nodes           []WorkflowProtocolNode     `json:"nodes"`
	Edges           []WorkflowProtocolEdge     `json:"edges"`
	Stages          []WorkflowProtocolStage    `json:"stages,omitempty"`
	Checks          []WorkflowProtocolCheck    `json:"checks,omitempty"`
	NextActions     []WorkflowProtocolAction   `json:"next_actions,omitempty"`
	Evidence        []WorkflowProtocolEvidence `json:"evidence,omitempty"`
	Metadata        map[string]any             `json:"metadata,omitempty"`
}

type WorkflowProtocolNode struct {
	ID                 string     `json:"id"`
	TaskID             string     `json:"task_id"`
	IssueNumber        int        `json:"issue_number,omitempty"`
	Title              string     `json:"title"`
	Lane               string     `json:"lane"`
	Status             string     `json:"status"`
	RewardMRG          float64    `json:"reward_mrg,omitempty"`
	EstimatedHours     float64    `json:"estimated_hours,omitempty"`
	RequiredWorkerKind WorkerKind `json:"worker_kind,omitempty"`
	SuggestedAgentType string     `json:"agent_type,omitempty"`
	IssueURL           string     `json:"issue_url,omitempty"`
	Dependencies       []string   `json:"dependencies,omitempty"`
}

type WorkflowProtocolEdge struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Relation string `json:"relation"`
}

type WorkflowProtocolStage struct {
	ID                string            `json:"id"`
	Title             string            `json:"title"`
	Summary           string            `json:"summary"`
	Status            string            `json:"status"`
	Tone              string            `json:"tone,omitempty"`
	ArtifactKind      string            `json:"artifact_kind"`
	InputEndpoint     string            `json:"input_endpoint,omitempty"`
	OutputEndpoint    string            `json:"output_endpoint"`
	OutputProtocol    string            `json:"output_protocol"`
	OutputProtocolURL string            `json:"output_protocol_url"`
	ActionEndpoint    string            `json:"action_endpoint,omitempty"`
	ContextURLs       map[string]string `json:"context_urls"`
	OutputIDs         []string          `json:"output_ids,omitempty"`
	ProducedCount     int               `json:"produced_count"`
	Reference         string            `json:"reference,omitempty"`
	URL               string            `json:"url,omitempty"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

type WorkflowProtocolCheck struct {
	ID       string `json:"id"`
	StageID  string `json:"stage_id"`
	Title    string `json:"title"`
	Status   string `json:"status"`
	Required bool   `json:"required"`
	Summary  string `json:"summary,omitempty"`
}

type WorkflowProtocolAction struct {
	ID           string     `json:"id"`
	Type         string     `json:"type"`
	Label        string     `json:"label"`
	TargetStep   string     `json:"target_step"`
	TargetNodeID string     `json:"target_node_id,omitempty"`
	TaskID       string     `json:"task_id,omitempty"`
	WorkerKind   WorkerKind `json:"worker_kind,omitempty"`
	Method       string     `json:"method,omitempty"`
	Endpoint     string     `json:"endpoint,omitempty"`
}

type WorkflowProtocolEvidence struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	Reference string    `json:"reference,omitempty"`
	URL       string    `json:"url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type ProjectRepositoryScanResponse struct {
	ProjectID      string                     `json:"project_id"`
	ProjectTitle   string                     `json:"project_title"`
	Status         string                     `json:"status"`
	Summary        string                     `json:"summary"`
	Stats          RepositoryScanStats        `json:"stats"`
	Languages      []RepositoryLanguage       `json:"languages"`
	Dependencies   []RepositoryDependencyFile `json:"dependencies"`
	Findings       []RepositoryScanFinding    `json:"findings"`
	SuggestedTasks []RepositorySuggestedTask  `json:"suggested_tasks"`
	UpdatedAt      time.Time                  `json:"updated_at"`
}

type RepositoryScanProtocolDocument struct {
	ProtocolVersion string                     `json:"protocol_version"`
	Kind            string                     `json:"kind"`
	ID              string                     `json:"id"`
	ProjectID       string                     `json:"project_id"`
	ProjectTitle    string                     `json:"project_title,omitempty"`
	Status          string                     `json:"status"`
	Summary         string                     `json:"summary,omitempty"`
	SourceRepo      string                     `json:"source_repository,omitempty"`
	UpdatedAt       time.Time                  `json:"updated_at"`
	Stats           RepositoryScanStats        `json:"stats"`
	Languages       []RepositoryLanguage       `json:"languages,omitempty"`
	Dependencies    []RepositoryDependencyFile `json:"dependencies,omitempty"`
	Findings        []RepositoryScanFinding    `json:"findings"`
	SuggestedTasks  []RepositorySuggestedTask  `json:"suggested_tasks,omitempty"`
	Metadata        map[string]any             `json:"metadata,omitempty"`
}

type RepositoryScanStats struct {
	FileCount          int `json:"file_count"`
	ScannedFiles       int `json:"scanned_files"`
	SkippedFiles       int `json:"skipped_files"`
	DependencyFiles    int `json:"dependency_files"`
	FindingCount       int `json:"finding_count"`
	SuggestedTaskCount int `json:"suggested_task_count"`
}

type RepositoryLanguage struct {
	Language  string `json:"language"`
	Extension string `json:"extension"`
	FileCount int    `json:"file_count"`
}

type RepositoryDependencyFile struct {
	Path         string `json:"path"`
	Ecosystem    string `json:"ecosystem"`
	PackageCount int    `json:"package_count"`
	HasLockfile  bool   `json:"has_lockfile"`
}

type RepositoryScanFinding struct {
	ID       string `json:"id"`
	Severity string `json:"severity"`
	Category string `json:"category"`
	Title    string `json:"title"`
	Body     string `json:"body"`
	Path     string `json:"path,omitempty"`
	Line     int    `json:"line,omitempty"`
	Signal   string `json:"signal,omitempty"`
}

type RepositorySuggestedTask struct {
	ID                   string                  `json:"id"`
	SourceFindingID      string                  `json:"source_finding_id"`
	Signal               string                  `json:"signal"`
	Title                string                  `json:"title"`
	Body                 string                  `json:"body,omitempty"`
	Severity             string                  `json:"severity"`
	Lane                 string                  `json:"lane"`
	Path                 string                  `json:"path,omitempty"`
	EstimatedRewardCents int64                   `json:"estimated_reward_cents"`
	EstimatedHours       float64                 `json:"estimated_hours,omitempty"`
	WorkerKind           WorkerKind              `json:"worker_kind"`
	SuggestedAgentType   string                  `json:"suggested_agent_type,omitempty"`
	ReadyForBounty       bool                    `json:"ready_for_bounty"`
	AcceptanceCriteria   []string                `json:"acceptance_criteria,omitempty"`
	EvidenceRequired     []string                `json:"evidence_required,omitempty"`
	FundingPacket        RepositoryFundingPacket `json:"funding_packet"`
}

type RepositoryFundingPacket struct {
	Status                  string         `json:"status"`
	CanFund                 bool           `json:"can_fund"`
	RecommendedRewardCents  int64          `json:"recommended_reward_cents"`
	RecommendedFundingCents int64          `json:"recommended_funding_cents"`
	FundEndpoint            string         `json:"fund_endpoint"`
	PayPalOrderEndpoint     string         `json:"paypal_order_endpoint"`
	FundPayload             map[string]any `json:"fund_payload"`
	PayPalOrderPayload      map[string]any `json:"paypal_order_payload"`
	EvidenceChecklist       []string       `json:"evidence_checklist"`
}

type FundRepositorySuggestedTaskRequest struct {
	SuggestedTaskID  string        `json:"suggested_task_id,omitempty"`
	RewardCents      int64         `json:"reward_cents"`
	BudgetCents      int64         `json:"budget_cents"`
	PaymentMethod    PaymentMethod `json:"payment_method"`
	PaymentReference string        `json:"payment_reference"`
}

type FundRepositorySuggestedTaskResponse struct {
	ProtocolVersion     string          `json:"protocol_version"`
	Kind                string          `json:"kind"`
	ProjectID           string          `json:"project_id"`
	SuggestedTaskID     string          `json:"suggested_task_id"`
	Task                *Task           `json:"task"`
	LedgerEntries       []LedgerEntry   `json:"ledger_entries"`
	FundingReference    string          `json:"funding_reference,omitempty"`
	EvidenceChecklist   []string        `json:"evidence_checklist,omitempty"`
	TaskProtocolURL     string          `json:"task_protocol_url,omitempty"`
	WorkflowProtocolURL string          `json:"workflow_protocol_url,omitempty"`
	ScanProtocolURL     string          `json:"scan_protocol_url,omitempty"`
	WorkPacket          AgentWorkPacket `json:"work_packet"`
}

type RepositorySuggestedTaskPayPalOrderRequest struct {
	SuggestedTaskID string `json:"suggested_task_id,omitempty"`
	RewardCents     int64  `json:"reward_cents"`
	BudgetCents     int64  `json:"budget_cents"`
	ReturnURL       string `json:"return_url"`
	CancelURL       string `json:"cancel_url"`
}

type TaskGraphStats struct {
	NodeCount     int `json:"node_count"`
	EdgeCount     int `json:"edge_count"`
	ReadyCount    int `json:"ready_count"`
	BlockedCount  int `json:"blocked_count"`
	CompleteCount int `json:"complete_count"`
	OpenCount     int `json:"open_count"`
}

type TaskGraphNode struct {
	ID                 string     `json:"id"`
	TaskID             string     `json:"task_id"`
	IssueNumber        int        `json:"issue_number"`
	Title              string     `json:"title"`
	Lane               string     `json:"lane"`
	Status             string     `json:"status"`
	Ready              bool       `json:"ready"`
	BlockedBy          []string   `json:"blocked_by,omitempty"`
	RewardCents        int64      `json:"reward_cents"`
	EstimatedHours     float64    `json:"estimated_hours,omitempty"`
	RequiredWorkerKind WorkerKind `json:"required_worker_kind"`
	SuggestedAgentType string     `json:"suggested_agent_type,omitempty"`
	IssueURL           string     `json:"issue_url,omitempty"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type TaskGraphEdge struct {
	ID       string `json:"id"`
	From     string `json:"from"`
	To       string `json:"to"`
	Relation string `json:"relation"`
}

type AIWorkflowStage struct {
	ID                string            `json:"id"`
	Title             string            `json:"title"`
	Body              string            `json:"body"`
	Status            string            `json:"status"`
	Tone              string            `json:"tone"`
	ArtifactKind      string            `json:"artifact_kind"`
	InputEndpoint     string            `json:"input_endpoint,omitempty"`
	OutputEndpoint    string            `json:"output_endpoint"`
	OutputProtocol    string            `json:"output_protocol"`
	OutputProtocolURL string            `json:"output_protocol_url"`
	ActionEndpoint    string            `json:"action_endpoint,omitempty"`
	ContextURLs       map[string]string `json:"context_urls"`
	OutputIDs         []string          `json:"output_ids,omitempty"`
	ProducedCount     int               `json:"produced_count"`
	Reference         string            `json:"reference,omitempty"`
	URL               string            `json:"url,omitempty"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

type AIWorkflowSignal struct {
	ID              string    `json:"id"`
	Type            string    `json:"type"`
	Title           string    `json:"title"`
	Body            string    `json:"body"`
	Status          string    `json:"status"`
	Reference       string    `json:"reference,omitempty"`
	URL             string    `json:"url,omitempty"`
	DelegatedBy     string    `json:"delegated_by,omitempty"`
	DesignAgent     string    `json:"design_agent,omitempty"`
	SubagentType    string    `json:"subagent_type,omitempty"`
	DelegationChain []string  `json:"delegation_chain,omitempty"`
	SourceFindingID string    `json:"source_finding_id,omitempty"`
	Signal          string    `json:"signal,omitempty"`
	Path            string    `json:"path,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type WorkerDashboardResponse struct {
	ProtocolVersion    string                    `json:"protocol_version"`
	Kind               string                    `json:"kind"`
	Profile            WorkerProfile             `json:"profile"`
	Stats              WorkerStats               `json:"stats"`
	ClaimedTasks       []WorkerClaimedTask       `json:"claimed_tasks"`
	Rewards            []WorkerRewardEntry       `json:"rewards"`
	Reputation         []WorkerReputation        `json:"reputation"`
	ReputationAudit    WorkerReputationAudit     `json:"reputation_audit"`
	Proposals          []WorkerProposal          `json:"proposals"`
	SubmittedProposals []WorkerSubmittedProposal `json:"submitted_proposals"`
	IdentityStatus     []WorkerIdentityHint      `json:"identity_status"`
}

type WorkerProfile struct {
	UserID          string `json:"user_id"`
	Name            string `json:"name"`
	Email           string `json:"email"`
	WalletAddress   string `json:"wallet_address,omitempty"`
	GitHubUsername  string `json:"github_username,omitempty"`
	GitHubAvatarURL string `json:"github_avatar_url,omitempty"`
}

type WorkerStats struct {
	ClaimedTaskCount       int        `json:"claimed_task_count"`
	OpenProposalCount      int        `json:"open_proposal_count"`
	SubmittedProposalCount int        `json:"submitted_proposal_count"`
	RewardCents            int64      `json:"reward_cents"`
	ReputationScore        int        `json:"reputation_score"`
	RiskLevel              string     `json:"risk_level"`
	LastPaidAt             *time.Time `json:"last_paid_at,omitempty"`
}

type WorkerClaimedTask struct {
	ID                string     `json:"id"`
	ProjectID         string     `json:"project_id"`
	ProjectTitle      string     `json:"project_title"`
	IssueNumber       int        `json:"issue_number"`
	Title             string     `json:"title"`
	Acceptance        string     `json:"acceptance"`
	RewardCents       int64      `json:"reward_cents"`
	WorkerKind        WorkerKind `json:"worker_kind"`
	AgentType         string     `json:"agent_type,omitempty"`
	Status            string     `json:"status"`
	ProofHash         string     `json:"proof_hash,omitempty"`
	IssueURL          string     `json:"issue_url,omitempty"`
	PullRequestURL    string     `json:"pull_request_url,omitempty"`
	ReviewEvidenceURL string     `json:"review_evidence_url,omitempty"`
	ReviewNotes       string     `json:"review_notes,omitempty"`
	AcceptedAt        *time.Time `json:"accepted_at,omitempty"`
	SubmittedAt       *time.Time `json:"submitted_at,omitempty"`
}

type WorkerRewardEntry struct {
	Sequence    int       `json:"sequence"`
	Type        string    `json:"type"`
	AmountCents int64     `json:"amount_cents"`
	Reference   string    `json:"reference"`
	EntryHash   string    `json:"entry_hash"`
	CreatedAt   time.Time `json:"created_at"`
}

type WorkerReputation struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Tone  string `json:"tone"`
}

type WorkerReputationAudit struct {
	WorkerID               string     `json:"worker_id"`
	Name                   string     `json:"name,omitempty"`
	Kind                   WorkerKind `json:"kind,omitempty"`
	AgentType              string     `json:"agent_type,omitempty"`
	Score                  int        `json:"score"`
	Level                  string     `json:"level"`
	RiskLevel              string     `json:"risk_level"`
	CompletedTaskCount     int        `json:"completed_task_count"`
	RewardCents            int64      `json:"reward_cents"`
	RewardRowCount         int        `json:"reward_row_count"`
	HasGitHub              bool       `json:"has_github"`
	HasWallet              bool       `json:"has_wallet"`
	DuplicateIdentityCount int        `json:"duplicate_identity_count"`
	Flags                  []string   `json:"flags,omitempty"`
	LastPaidAt             *time.Time `json:"last_paid_at,omitempty"`
}

type WorkerProposal struct {
	ID                 string          `json:"id"`
	ClaimID            string          `json:"claim_id,omitempty"`
	ProjectID          string          `json:"project_id"`
	ProjectTitle       string          `json:"project_title"`
	IssueNumber        int             `json:"issue_number"`
	Title              string          `json:"title"`
	Acceptance         string          `json:"acceptance"`
	RewardCents        int64           `json:"reward_cents"`
	EstimatedHours     float64         `json:"estimated_hours,omitempty"`
	RequiredWorkerKind WorkerKind      `json:"required_worker_kind"`
	SuggestedAgentType string          `json:"suggested_agent_type,omitempty"`
	MatchScore         int             `json:"match_score"`
	MatchReasons       []string        `json:"match_reasons,omitempty"`
	EvidenceRequired   []string        `json:"evidence_required,omitempty"`
	IssueURL           string          `json:"issue_url,omitempty"`
	ProposalEndpoint   string          `json:"proposal_endpoint,omitempty"`
	ClaimPacket        *ProposalPacket `json:"claim_packet,omitempty"`
	CreatedAt          time.Time       `json:"created_at"`
}

type WorkerSubmittedProposal struct {
	ID             string    `json:"id"`
	ProjectID      string    `json:"project_id"`
	ProjectTitle   string    `json:"project_title"`
	TaskID         string    `json:"task_id"`
	ClaimID        string    `json:"claim_id,omitempty"`
	WorkerID       string    `json:"worker_id"`
	IssueNumber    int       `json:"issue_number"`
	Title          string    `json:"title"`
	CoverLetter    string    `json:"cover_letter"`
	BidCents       int64     `json:"bid_cents"`
	EstimatedHours float64   `json:"estimated_hours,omitempty"`
	Availability   string    `json:"availability,omitempty"`
	Status         string    `json:"status"`
	Reference      string    `json:"reference"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type WorkerIdentityHint struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Ready bool   `json:"ready"`
}

type AdminSummary struct {
	UserCount         int                `json:"user_count"`
	AdminCount        int                `json:"admin_count"`
	ClientCount       int                `json:"client_count"`
	WalletCount       int                `json:"wallet_count"`
	ProjectCount      int                `json:"project_count"`
	OpenTaskCount     int                `json:"open_task_count"`
	AcceptedTaskCount int                `json:"accepted_task_count"`
	NotificationCount int                `json:"notification_count"`
	AttachmentCount   int                `json:"attachment_count"`
	TotalBudgetCents  int64              `json:"total_budget_cents"`
	WorkPoolCents     int64              `json:"work_pool_cents"`
	PlatformFeeCents  int64              `json:"platform_fee_cents"`
	PaidTaskCents     int64              `json:"paid_task_cents"`
	TokenSymbol       string             `json:"token_symbol"`
	PaymentMode       string             `json:"payment_mode"`
	RepoProvider      string             `json:"repo_provider"`
	PayPalReady       bool               `json:"paypal_ready"`
	CryptoReady       bool               `json:"crypto_ready"`
	GitHubReady       bool               `json:"github_ready"`
	SMTPReady         bool               `json:"smtp_ready"`
	DevPaymentEnabled bool               `json:"dev_payment_enabled"`
	BountyRoot        string             `json:"bounty_root,omitempty"`
	UploadRoot        string             `json:"upload_root,omitempty"`
	SSLReviews        []*SSLReviewStatus `json:"ssl_reviews,omitempty"`
}

type AdminReputationResponse struct {
	Stats   AdminReputationStats    `json:"stats"`
	Workers []WorkerReputationAudit `json:"workers"`
}

type AdminReputationStats struct {
	WorkerCount        int `json:"worker_count"`
	HighRiskCount      int `json:"high_risk_count"`
	MediumRiskCount    int `json:"medium_risk_count"`
	LowRiskCount       int `json:"low_risk_count"`
	TrustedCount       int `json:"trusted_count"`
	NewWorkerCount     int `json:"new_worker_count"`
	CompletedTaskCount int `json:"completed_task_count"`
}

type AdminOpsQueueResponse struct {
	ProtocolVersion string              `json:"protocol_version"`
	Kind            string              `json:"kind"`
	Stats           AdminOpsQueueStats  `json:"stats"`
	Items           []AdminOpsQueueItem `json:"items"`
}

type AdminOpsQueueStats struct {
	TotalCount        int        `json:"total_count"`
	DisputeCount      int        `json:"dispute_count"`
	ModerationCount   int        `json:"moderation_count"`
	ProposalCount     int        `json:"proposal_count"`
	PayoutReviewCount int        `json:"payout_review_count"`
	FraudCount        int        `json:"fraud_count"`
	SecurityCount     int        `json:"security_count"`
	CriticalCount     int        `json:"critical_count"`
	UpdatedAt         *time.Time `json:"updated_at,omitempty"`
}

type AdminOpsQueueItem struct {
	ID           string                `json:"id"`
	Type         string                `json:"type"`
	Severity     string                `json:"severity"`
	Title        string                `json:"title"`
	Body         string                `json:"body"`
	ProjectID    string                `json:"project_id,omitempty"`
	ProjectTitle string                `json:"project_title,omitempty"`
	TaskID       string                `json:"task_id,omitempty"`
	IssueNumber  int                   `json:"issue_number,omitempty"`
	UserID       string                `json:"user_id,omitempty"`
	Reference    string                `json:"reference,omitempty"`
	URL          string                `json:"url,omitempty"`
	Status       string                `json:"status"`
	Actions      []AdminOpsQueueAction `json:"actions,omitempty"`
	CreatedAt    time.Time             `json:"created_at"`
}

type AdminOpsQueueAction struct {
	ID       string         `json:"id"`
	Label    string         `json:"label"`
	Type     string         `json:"type"`
	URL      string         `json:"url,omitempty"`
	Method   string         `json:"method,omitempty"`
	Endpoint string         `json:"endpoint,omitempty"`
	Payload  map[string]any `json:"payload,omitempty"`
}

type AdminUser struct {
	PublicUser
	ProjectCount     int                    `json:"project_count"`
	TotalBudgetCents int64                  `json:"total_budget_cents"`
	LastProjectAt    *time.Time             `json:"last_project_at,omitempty"`
	WorkerAudit      *WorkerReputationAudit `json:"worker_audit,omitempty"`
}

type SSLReviewStatus struct {
	Domain        string     `json:"domain"`
	Port          string     `json:"port"`
	Status        string     `json:"status"`
	Issuer        string     `json:"issuer,omitempty"`
	Subject       string     `json:"subject,omitempty"`
	SerialNumber  string     `json:"serial_number,omitempty"`
	DNSNames      []string   `json:"dns_names,omitempty"`
	NotBefore     *time.Time `json:"not_before,omitempty"`
	NotAfter      *time.Time `json:"not_after,omitempty"`
	DaysRemaining int        `json:"days_remaining"`
	LastCheckedAt *time.Time `json:"last_checked_at,omitempty"`
	NextCheckAt   *time.Time `json:"next_check_at,omitempty"`
	Error         string     `json:"error,omitempty"`
	CheckedBy     string     `json:"checked_by,omitempty"`
}

type GeminiAPIKey struct {
	ID              string     `json:"id"`
	Provider        string     `json:"provider"`
	Model           string     `json:"model,omitempty"`
	KeyValue        string     `json:"key_value"`
	KeyHint         string     `json:"key_hint"`
	Status          string     `json:"status"`
	RequestCount    int64      `json:"request_count"`
	SuccessCount    int64      `json:"success_count"`
	QuotaErrorCount int64      `json:"quota_error_count"`
	LastStatusCode  int        `json:"last_status_code"`
	LastError       string     `json:"last_error,omitempty"`
	LastUsedAt      *time.Time `json:"last_used_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type GeminiWebhookLog struct {
	ID              string             `json:"id"`
	DeliveryID      string             `json:"delivery_id,omitempty"`
	EventName       string             `json:"event_name"`
	Action          string             `json:"action,omitempty"`
	Repository      string             `json:"repository,omitempty"`
	PullNumber      int                `json:"pull_number,omitempty"`
	Sender          string             `json:"sender,omitempty"`
	Status          string             `json:"status"`
	StatusCode      int                `json:"status_code"`
	Error           string             `json:"error,omitempty"`
	CommentURL      string             `json:"comment_url,omitempty"`
	KeyID           string             `json:"key_id,omitempty"`
	Labels          []string           `json:"labels,omitempty"`
	ContextURLs     []string           `json:"context_urls,omitempty"`
	Evidence        []string           `json:"evidence,omitempty"`
	Runbook         []string           `json:"runbook,omitempty"`
	Checks          []AgentActionCheck `json:"checks,omitempty"`
	SourceFindingID string             `json:"source_finding_id,omitempty"`
	Signal          string             `json:"signal,omitempty"`
	Path            string             `json:"path,omitempty"`
	DelegatedBy     string             `json:"delegated_by,omitempty"`
	DesignAgent     string             `json:"design_agent,omitempty"`
	SubagentType    string             `json:"subagent_type,omitempty"`
	DelegationChain []string           `json:"delegation_chain,omitempty"`
	DurationMillis  int64              `json:"duration_millis"`
	ReceivedAt      time.Time          `json:"received_at"`
	CompletedAt     *time.Time         `json:"completed_at,omitempty"`
}

type AgentActionCheck struct {
	Name         string `json:"name"`
	Status       string `json:"status"`
	Summary      string `json:"summary,omitempty"`
	ReferenceURL string `json:"reference_url,omitempty"`
}

type AgentActionRequest struct {
	Action          string             `json:"action"`
	ClaimID         string             `json:"claim_id,omitempty"`
	BountyID        string             `json:"bounty_id,omitempty"`
	AgentType       string             `json:"agent_type,omitempty"`
	DelegatedBy     string             `json:"delegated_by,omitempty"`
	DesignAgent     string             `json:"design_agent,omitempty"`
	SubagentType    string             `json:"subagent_type,omitempty"`
	DelegationChain []string           `json:"delegation_chain,omitempty"`
	Status          string             `json:"status,omitempty"`
	PullNumber      int                `json:"pull_number,omitempty"`
	ReferenceURL    string             `json:"reference_url,omitempty"`
	Labels          []string           `json:"labels,omitempty"`
	ContextURLs     []string           `json:"context_urls,omitempty"`
	Evidence        []string           `json:"evidence,omitempty"`
	Runbook         []string           `json:"runbook,omitempty"`
	Checks          []AgentActionCheck `json:"checks,omitempty"`
	SourceFindingID string             `json:"source_finding_id,omitempty"`
	Signal          string             `json:"signal,omitempty"`
	Path            string             `json:"path,omitempty"`
	DurationMillis  int64              `json:"duration_millis,omitempty"`
}

type AgentActionResponse struct {
	ProtocolVersion string             `json:"protocol_version"`
	Kind            string             `json:"kind"`
	ActionID        string             `json:"action_id"`
	ProjectID       string             `json:"project_id"`
	ClaimID         string             `json:"claim_id,omitempty"`
	BountyID        string             `json:"bounty_id,omitempty"`
	Action          string             `json:"action"`
	AgentType       string             `json:"agent_type"`
	Status          string             `json:"status"`
	Repository      string             `json:"repository,omitempty"`
	PullNumber      int                `json:"pull_number,omitempty"`
	ReferenceURL    string             `json:"reference_url,omitempty"`
	Labels          []string           `json:"labels,omitempty"`
	ContextURLs     []string           `json:"context_urls,omitempty"`
	Evidence        []string           `json:"evidence,omitempty"`
	Runbook         []string           `json:"runbook,omitempty"`
	Checks          []AgentActionCheck `json:"checks,omitempty"`
	SourceFindingID string             `json:"source_finding_id,omitempty"`
	Signal          string             `json:"signal,omitempty"`
	Path            string             `json:"path,omitempty"`
	DelegatedBy     string             `json:"delegated_by,omitempty"`
	DesignAgent     string             `json:"design_agent,omitempty"`
	SubagentType    string             `json:"subagent_type,omitempty"`
	DelegationChain []string           `json:"delegation_chain,omitempty"`
	DurationMillis  int64              `json:"duration_millis"`
	ReceivedAt      time.Time          `json:"received_at"`
	CompletedAt     *time.Time         `json:"completed_at,omitempty"`
	Log             GeminiWebhookLog   `json:"log"`
}

type EvaluateProjectRequest struct {
	Description     string   `json:"description"`
	Requirements    []string `json:"requirements"`
	Deliverables    []string `json:"deliverables"`
	Timeline        string   `json:"timeline"`
	TechStack       string   `json:"tech_stack"`
	Complexity      string   `json:"complexity"`
	Constraints     string   `json:"constraints"`
	ReferenceBudget int64    `json:"reference_budget,omitempty"` // in USD
}

type EvaluateProjectResponse struct {
	SuggestedLow    int64            `json:"suggested_low"`
	SuggestedHigh   int64            `json:"suggested_high"`
	ConfidenceLevel float64          `json:"confidence_level"`
	TaskBreakdown   map[string]int64 `json:"task_breakdown"`
	Assumptions     []string         `json:"assumptions"`
	Risks           []string         `json:"risks"`
	Rationale       string           `json:"rationale"`
}
