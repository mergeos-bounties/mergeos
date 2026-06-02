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
	CreatedAt          time.Time  `json:"created_at"`
	AcceptedAt         *time.Time `json:"accepted_at,omitempty"`
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
	Environment       string   `json:"environment"`
	TokenSymbol       string   `json:"token_symbol"`
	PaymentMode       string   `json:"payment_mode"`
	RepoProvider      string   `json:"repo_provider"`
	GitHubOAuthReady  bool     `json:"github_oauth_ready"`
	GitHubOAuthClient string   `json:"github_oauth_client_id,omitempty"`
	PayPalReady       bool     `json:"paypal_ready"`
	CryptoReady       bool     `json:"crypto_ready"`
	GitHubReady       bool     `json:"github_ready"`
	SMTPReady         bool     `json:"smtp_ready"`
	DevPaymentEnabled bool     `json:"dev_payment_enabled"`
	DevPaymentCode    string   `json:"dev_payment_code,omitempty"`
	CryptoReceiver    string   `json:"crypto_receiver,omitempty"`
	CryptoAsset       string   `json:"crypto_asset,omitempty"`
	CryptoToken       string   `json:"crypto_token,omitempty"`
	BountyRoot        string   `json:"bounty_root,omitempty"`
	UploadRoot        string   `json:"upload_root,omitempty"`
	AdminBootstrap    bool     `json:"admin_bootstrap"`
	PrimaryDomain     string   `json:"primary_domain,omitempty"`
	AdminDomain       string   `json:"admin_domain,omitempty"`
	ScanDomain        string   `json:"scan_domain,omitempty"`
	SSLReviewDomains  []string `json:"ssl_review_domains,omitempty"`
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
	AmountCents int64  `json:"amount_cents"`
	Description string `json:"description"`
	ReturnURL   string `json:"return_url"`
	CancelURL   string `json:"cancel_url"`
}

type CreatePayPalOrderResponse struct {
	OrderID     string `json:"order_id"`
	ApprovalURL string `json:"approval_url"`
	Status      string `json:"status"`
}

type ImportRepoIssuesRequest struct {
	RepoURL string `json:"repo_url"`
}

type ImportRepoIssuesResponse struct {
	Owner               string               `json:"owner"`
	Name                string               `json:"name"`
	RepoURL             string               `json:"repo_url"`
	IssueCount          int                  `json:"issue_count"`
	TotalEstimatedCents int64                `json:"total_estimated_cents"`
	Issues              []*ImportedRepoIssue `json:"issues"`
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
	RequiredWorkerKind WorkerKind `json:"required_worker_kind"`
	SuggestedAgentType string     `json:"suggested_agent_type"`
	Reasons            []string   `json:"reasons"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type MarketplaceResponse struct {
	Stats        MarketplaceStats          `json:"stats"`
	Projects     []*MarketplaceProject     `json:"projects"`
	Bounties     []*MarketplaceBounty      `json:"bounties"`
	Contributors []*MarketplaceContributor `json:"contributors"`
	Agents       []*MarketplaceAgent       `json:"agents"`
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
	ID                 string     `json:"id"`
	ProjectID          string     `json:"project_id"`
	ProjectTitle       string     `json:"project_title"`
	IssueNumber        int        `json:"issue_number"`
	Title              string     `json:"title"`
	Acceptance         string     `json:"acceptance"`
	RewardCents        int64      `json:"reward_cents"`
	RequiredWorkerKind WorkerKind `json:"required_worker_kind"`
	SuggestedAgentType string     `json:"suggested_agent_type,omitempty"`
	BountyType         string     `json:"bounty_type,omitempty"`
	IssueURL           string     `json:"issue_url,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
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
	Type          string     `json:"type"`
	Title         string     `json:"title"`
	WorkerKind    WorkerKind `json:"worker_kind"`
	TaskCount     int        `json:"task_count"`
	OpenTaskCount int        `json:"open_task_count"`
	BudgetCents   int64      `json:"budget_cents"`
}

type PublicLiveFeedResponse struct {
	Stats PublicLiveFeedStats  `json:"stats"`
	Items []PublicLiveFeedItem `json:"items"`
}

type PublicLiveFeedStats struct {
	ProjectCount      int        `json:"project_count"`
	OpenTaskCount     int        `json:"open_task_count"`
	AcceptedTaskCount int        `json:"accepted_task_count"`
	LedgerEntryCount  int        `json:"ledger_entry_count"`
	AIActionCount     int        `json:"ai_action_count"`
	TotalBudgetCents  int64      `json:"total_budget_cents"`
	TokenSymbol       string     `json:"token_symbol"`
	UpdatedAt         *time.Time `json:"updated_at,omitempty"`
}

type PublicLiveFeedItem struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	Title        string    `json:"title"`
	Body         string    `json:"body"`
	ProjectID    string    `json:"project_id,omitempty"`
	ProjectTitle string    `json:"project_title,omitempty"`
	Actor        string    `json:"actor,omitempty"`
	AmountCents  int64     `json:"amount_cents,omitempty"`
	Reference    string    `json:"reference,omitempty"`
	URL          string    `json:"url,omitempty"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

type ProjectDeploymentResponse struct {
	ProjectID    string             `json:"project_id"`
	ProjectTitle string             `json:"project_title"`
	Status       string             `json:"status"`
	Progress     int                `json:"progress"`
	UpdatedAt    time.Time          `json:"updated_at"`
	Stages       []DeploymentStage  `json:"stages"`
	Signals      []DeploymentSignal `json:"signals"`
}

type ProjectEscrowResponse struct {
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

type ProjectAIWorkflowResponse struct {
	ProjectID       string             `json:"project_id"`
	ProjectTitle    string             `json:"project_title"`
	Status          string             `json:"status"`
	Progress        int                `json:"progress"`
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

type ProjectRepositoryScanResponse struct {
	ProjectID    string                     `json:"project_id"`
	ProjectTitle string                     `json:"project_title"`
	Status       string                     `json:"status"`
	Summary      string                     `json:"summary"`
	Stats        RepositoryScanStats        `json:"stats"`
	Languages    []RepositoryLanguage       `json:"languages"`
	Dependencies []RepositoryDependencyFile `json:"dependencies"`
	Findings     []RepositoryScanFinding    `json:"findings"`
	UpdatedAt    time.Time                  `json:"updated_at"`
}

type RepositoryScanStats struct {
	FileCount       int `json:"file_count"`
	ScannedFiles    int `json:"scanned_files"`
	SkippedFiles    int `json:"skipped_files"`
	DependencyFiles int `json:"dependency_files"`
	FindingCount    int `json:"finding_count"`
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
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Status    string    `json:"status"`
	Tone      string    `json:"tone"`
	Reference string    `json:"reference,omitempty"`
	URL       string    `json:"url,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AIWorkflowSignal struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Status    string    `json:"status"`
	Reference string    `json:"reference,omitempty"`
	URL       string    `json:"url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type WorkerDashboardResponse struct {
	Profile         WorkerProfile         `json:"profile"`
	Stats           WorkerStats           `json:"stats"`
	ClaimedTasks    []WorkerClaimedTask   `json:"claimed_tasks"`
	Rewards         []WorkerRewardEntry   `json:"rewards"`
	Reputation      []WorkerReputation    `json:"reputation"`
	ReputationAudit WorkerReputationAudit `json:"reputation_audit"`
	Proposals       []WorkerProposal      `json:"proposals"`
	IdentityStatus  []WorkerIdentityHint  `json:"identity_status"`
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
	ClaimedTaskCount  int        `json:"claimed_task_count"`
	OpenProposalCount int        `json:"open_proposal_count"`
	RewardCents       int64      `json:"reward_cents"`
	ReputationScore   int        `json:"reputation_score"`
	RiskLevel         string     `json:"risk_level"`
	LastPaidAt        *time.Time `json:"last_paid_at,omitempty"`
}

type WorkerClaimedTask struct {
	ID           string     `json:"id"`
	ProjectID    string     `json:"project_id"`
	ProjectTitle string     `json:"project_title"`
	IssueNumber  int        `json:"issue_number"`
	Title        string     `json:"title"`
	Acceptance   string     `json:"acceptance"`
	RewardCents  int64      `json:"reward_cents"`
	WorkerKind   WorkerKind `json:"worker_kind"`
	AgentType    string     `json:"agent_type,omitempty"`
	ProofHash    string     `json:"proof_hash,omitempty"`
	IssueURL     string     `json:"issue_url,omitempty"`
	AcceptedAt   *time.Time `json:"accepted_at,omitempty"`
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
	ID                 string     `json:"id"`
	ProjectID          string     `json:"project_id"`
	ProjectTitle       string     `json:"project_title"`
	IssueNumber        int        `json:"issue_number"`
	Title              string     `json:"title"`
	Acceptance         string     `json:"acceptance"`
	RewardCents        int64      `json:"reward_cents"`
	RequiredWorkerKind WorkerKind `json:"required_worker_kind"`
	SuggestedAgentType string     `json:"suggested_agent_type,omitempty"`
	MatchScore         int        `json:"match_score"`
	IssueURL           string     `json:"issue_url,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
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
	Stats AdminOpsQueueStats  `json:"stats"`
	Items []AdminOpsQueueItem `json:"items"`
}

type AdminOpsQueueStats struct {
	TotalCount        int        `json:"total_count"`
	DisputeCount      int        `json:"dispute_count"`
	ModerationCount   int        `json:"moderation_count"`
	PayoutReviewCount int        `json:"payout_review_count"`
	FraudCount        int        `json:"fraud_count"`
	SecurityCount     int        `json:"security_count"`
	CriticalCount     int        `json:"critical_count"`
	UpdatedAt         *time.Time `json:"updated_at,omitempty"`
}

type AdminOpsQueueItem struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	Severity     string    `json:"severity"`
	Title        string    `json:"title"`
	Body         string    `json:"body"`
	ProjectID    string    `json:"project_id,omitempty"`
	ProjectTitle string    `json:"project_title,omitempty"`
	TaskID       string    `json:"task_id,omitempty"`
	IssueNumber  int       `json:"issue_number,omitempty"`
	UserID       string    `json:"user_id,omitempty"`
	Reference    string    `json:"reference,omitempty"`
	URL          string    `json:"url,omitempty"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
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
	ID             string     `json:"id"`
	DeliveryID     string     `json:"delivery_id,omitempty"`
	EventName      string     `json:"event_name"`
	Action         string     `json:"action,omitempty"`
	Repository     string     `json:"repository,omitempty"`
	PullNumber     int        `json:"pull_number,omitempty"`
	Sender         string     `json:"sender,omitempty"`
	Status         string     `json:"status"`
	StatusCode     int        `json:"status_code"`
	Error          string     `json:"error,omitempty"`
	CommentURL     string     `json:"comment_url,omitempty"`
	KeyID          string     `json:"key_id,omitempty"`
	Labels         []string   `json:"labels,omitempty"`
	DurationMillis int64      `json:"duration_millis"`
	ReceivedAt     time.Time  `json:"received_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
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
