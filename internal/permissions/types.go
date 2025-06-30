package permissions

import (
	"strings"
	"time"
)

// PermissionType defines the type of permission being requested
type PermissionType string

const (
	// File operations
	PermissionFileRead   PermissionType = "file:read"
	PermissionFileWrite  PermissionType = "file:write"
	PermissionFileDelete PermissionType = "file:delete"
	PermissionFileCreate PermissionType = "file:create"
	PermissionFileList   PermissionType = "file:list"

	// Directory operations
	PermissionDirRead   PermissionType = "dir:read"
	PermissionDirWrite  PermissionType = "dir:write"
	PermissionDirCreate PermissionType = "dir:create"
	PermissionDirDelete PermissionType = "dir:delete"

	// Code execution
	PermissionCodeExecute PermissionType = "code:execute"
	PermissionShellAccess PermissionType = "shell:access"
	PermissionProcessRun  PermissionType = "process:run"

	// Network operations
	PermissionNetworkAccess PermissionType = "network:access"
	PermissionHTTPRequest   PermissionType = "http:request"

	// System operations
	PermissionSystemInfo   PermissionType = "system:info"
	PermissionSystemConfig PermissionType = "system:config"

	// Tool operations
	PermissionToolUse PermissionType = "tool:use"
	PermissionMCPCall PermissionType = "mcp:call"

	// Git operations
	PermissionGitRead  PermissionType = "git:read"
	PermissionGitWrite PermissionType = "git:write"

	// Database operations
	PermissionDBRead  PermissionType = "db:read"
	PermissionDBWrite PermissionType = "db:write"
)

// PermissionScope defines the scope of a permission
type PermissionScope string

const (
	ScopeSession    PermissionScope = "session"    // Valid for current session only
	ScopePersistent PermissionScope = "persistent" // Persists across sessions
	ScopeTemporary  PermissionScope = "temporary"  // Valid for limited time
	ScopeOneTime    PermissionScope = "one_time"   // Valid for single use
)

// PermissionStatus defines the status of a permission request
type PermissionStatus string

const (
	StatusPending  PermissionStatus = "pending"
	StatusApproved PermissionStatus = "approved"
	StatusDenied   PermissionStatus = "denied"
	StatusExpired  PermissionStatus = "expired"
	StatusRevoked  PermissionStatus = "revoked"
)

// PermissionRequest represents a request for permission
type PermissionRequest struct {
	ID          string                 `json:"id"`
	SessionID   string                 `json:"session_id"`
	Type        PermissionType         `json:"type"`
	Resource    string                 `json:"resource"` // File path, URL, tool name, etc.
	Scope       PermissionScope        `json:"scope"`
	Reason      string                 `json:"reason"`  // Human-readable reason
	Context     map[string]interface{} `json:"context"` // Additional context
	RequestedAt time.Time              `json:"requested_at"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// PermissionResponse represents a response to a permission request
type PermissionResponse struct {
	RequestID   string           `json:"request_id"`
	Status      PermissionStatus `json:"status"`
	Reason      string           `json:"reason,omitempty"`
	Conditions  []string         `json:"conditions,omitempty"` // Additional conditions
	ExpiresAt   *time.Time       `json:"expires_at,omitempty"`
	RespondedAt time.Time        `json:"responded_at"`
	RespondedBy string           `json:"responded_by"` // User ID or "auto"
}

// Permission represents a granted permission
type Permission struct {
	ID         string                 `json:"id"`
	SessionID  string                 `json:"session_id"`
	Type       PermissionType         `json:"type"`
	Resource   string                 `json:"resource"`
	Scope      PermissionScope        `json:"scope"`
	Status     PermissionStatus       `json:"status"`
	Conditions []string               `json:"conditions,omitempty"`
	GrantedAt  time.Time              `json:"granted_at"`
	ExpiresAt  *time.Time             `json:"expires_at,omitempty"`
	UsageCount int                    `json:"usage_count"`
	MaxUsage   int                    `json:"max_usage,omitempty"` // 0 = unlimited
	LastUsedAt *time.Time             `json:"last_used_at,omitempty"`
	GrantedBy  string                 `json:"granted_by"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// PermissionCheck represents a permission check request
type PermissionCheck struct {
	SessionID string                 `json:"session_id"`
	Type      PermissionType         `json:"type"`
	Resource  string                 `json:"resource"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

// PermissionCheckResult represents the result of a permission check
type PermissionCheckResult struct {
	Allowed          bool        `json:"allowed"`
	Permission       *Permission `json:"permission,omitempty"`
	Reason           string      `json:"reason,omitempty"`
	RequiresApproval bool        `json:"requires_approval"`
	AutoApproved     bool        `json:"auto_approved"`
}

// PermissionAuditEntry represents an audit log entry
type PermissionAuditEntry struct {
	ID        string                 `json:"id"`
	SessionID string                 `json:"session_id"`
	Action    string                 `json:"action"` // "request", "approve", "deny", "use", "revoke"
	Type      PermissionType         `json:"type"`
	Resource  string                 `json:"resource"`
	Status    PermissionStatus       `json:"status"`
	UserID    string                 `json:"user_id,omitempty"`
	Reason    string                 `json:"reason,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	IPAddress string                 `json:"ip_address,omitempty"`
	UserAgent string                 `json:"user_agent,omitempty"`
}

// SessionTrust represents trust level for a session
type SessionTrust struct {
	SessionID      string           `json:"session_id"`
	TrustLevel     int              `json:"trust_level"`  // 0-100 scale
	AutoApprove    []PermissionType `json:"auto_approve"` // Types to auto-approve
	CreatedAt      time.Time        `json:"created_at"`
	LastActivity   time.Time        `json:"last_activity"`
	SuccessCount   int              `json:"success_count"`   // Successful operations
	ViolationCount int              `json:"violation_count"` // Permission violations
}

// PermissionPolicy represents a permission policy
type PermissionPolicy struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Rules       []PermissionRule `json:"rules"`
	Priority    int              `json:"priority"`
	Enabled     bool             `json:"enabled"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

// PermissionRule represents a rule within a policy
type PermissionRule struct {
	ID         string          `json:"id"`
	Type       PermissionType  `json:"type"`
	Resource   string          `json:"resource"` // Can use wildcards
	Action     string          `json:"action"`   // "allow", "deny", "require_approval"
	Conditions []RuleCondition `json:"conditions,omitempty"`
	Priority   int             `json:"priority"`
	Enabled    bool            `json:"enabled"`
}

// RuleCondition represents a condition for a permission rule
type RuleCondition struct {
	Field    string      `json:"field"`    // "session_id", "trust_level", "time", etc.
	Operator string      `json:"operator"` // "eq", "gt", "lt", "contains", "matches"
	Value    interface{} `json:"value"`
}

// PermissionConfig represents permission system configuration
type PermissionConfig struct {
	DefaultScope             PermissionScope    `json:"default_scope"`
	DefaultExpiration        time.Duration      `json:"default_expiration"`
	RequireApproval          bool               `json:"require_approval"`
	AutoApproveThreshold     int                `json:"auto_approve_threshold"` // Trust level threshold
	MaxPermissionsPerSession int                `json:"max_permissions_per_session"`
	AuditEnabled             bool               `json:"audit_enabled"`
	CleanupInterval          time.Duration      `json:"cleanup_interval"`
	Policies                 []PermissionPolicy `json:"policies"`
}

// PermissionError represents a permission-related error
type PermissionError struct {
	Code     string         `json:"code"`
	Message  string         `json:"message"`
	Type     PermissionType `json:"type,omitempty"`
	Resource string         `json:"resource,omitempty"`
}

func (e *PermissionError) Error() string {
	return e.Message
}

// Common permission errors
var (
	ErrPermissionDenied = &PermissionError{
		Code:    "PERMISSION_DENIED",
		Message: "Permission denied",
	}
	ErrPermissionExpired = &PermissionError{
		Code:    "PERMISSION_EXPIRED",
		Message: "Permission has expired",
	}
	ErrPermissionNotFound = &PermissionError{
		Code:    "PERMISSION_NOT_FOUND",
		Message: "Permission not found",
	}
	ErrInvalidPermissionType = &PermissionError{
		Code:    "INVALID_PERMISSION_TYPE",
		Message: "Invalid permission type",
	}
	ErrPermissionLimitExceeded = &PermissionError{
		Code:    "PERMISSION_LIMIT_EXCEEDED",
		Message: "Permission usage limit exceeded",
	}
)

// Helper functions

// IsExpired checks if a permission has expired
func (p *Permission) IsExpired() bool {
	if p.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*p.ExpiresAt)
}

// IsUsageExceeded checks if permission usage limit is exceeded
func (p *Permission) IsUsageExceeded() bool {
	if p.MaxUsage == 0 {
		return false // Unlimited usage
	}
	return p.UsageCount >= p.MaxUsage
}

// CanUse checks if permission can be used
func (p *Permission) CanUse() bool {
	return p.Status == StatusApproved && !p.IsExpired() && !p.IsUsageExceeded()
}

// IncrementUsage increments the usage count
func (p *Permission) IncrementUsage() {
	p.UsageCount++
	now := time.Now()
	p.LastUsedAt = &now
}

// IsExpired checks if a permission request has expired
func (pr *PermissionRequest) IsExpired() bool {
	if pr.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*pr.ExpiresAt)
}

// ResourceMatches checks if a resource matches a pattern (supports wildcards)
func ResourceMatches(pattern, resource string) bool {
	if pattern == "*" {
		return true
	}
	if pattern == resource {
		return true
	}

	// Simple wildcard matching
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(resource, prefix)
	}

	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(resource, suffix)
	}

	return false
}

// GetPermissionCategory returns the category of a permission type
func GetPermissionCategory(permType PermissionType) string {
	switch {
	case strings.HasPrefix(string(permType), "file:"):
		return "file"
	case strings.HasPrefix(string(permType), "dir:"):
		return "directory"
	case strings.HasPrefix(string(permType), "code:"), strings.HasPrefix(string(permType), "shell:"), strings.HasPrefix(string(permType), "process:"):
		return "execution"
	case strings.HasPrefix(string(permType), "network:"), strings.HasPrefix(string(permType), "http:"):
		return "network"
	case strings.HasPrefix(string(permType), "system:"):
		return "system"
	case strings.HasPrefix(string(permType), "tool:"), strings.HasPrefix(string(permType), "mcp:"):
		return "tool"
	case strings.HasPrefix(string(permType), "git:"):
		return "git"
	case strings.HasPrefix(string(permType), "db:"):
		return "database"
	default:
		return "other"
	}
}

// GetRiskLevel returns the risk level of a permission type (1-10 scale)
func GetRiskLevel(permType PermissionType) int {
	switch permType {
	case PermissionFileRead, PermissionDirRead, PermissionFileList, PermissionSystemInfo, PermissionGitRead, PermissionDBRead:
		return 2 // Low risk - read operations
	case PermissionFileWrite, PermissionFileCreate, PermissionDirCreate:
		return 5 // Medium risk - write operations
	case PermissionFileDelete, PermissionDirDelete, PermissionGitWrite, PermissionDBWrite:
		return 7 // High risk - destructive operations
	case PermissionCodeExecute, PermissionShellAccess, PermissionProcessRun:
		return 9 // Very high risk - code execution
	case PermissionNetworkAccess, PermissionHTTPRequest:
		return 6 // Medium-high risk - network access
	case PermissionSystemConfig:
		return 8 // High risk - system configuration
	case PermissionToolUse, PermissionMCPCall:
		return 4 // Medium-low risk - tool usage
	default:
		return 5 // Default medium risk
	}
}
