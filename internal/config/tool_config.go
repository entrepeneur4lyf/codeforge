package config

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// ToolType represents different types of tools
type ToolType string

const (
	ToolTypeFileSystem     ToolType = "filesystem"
	ToolTypeCodeExecution  ToolType = "code_execution"
	ToolTypeWebSearch      ToolType = "web_search"
	ToolTypeAPICall        ToolType = "api_call"
	ToolTypeDatabase       ToolType = "database"
	ToolTypeGit            ToolType = "git"
	ToolTypeTerminal       ToolType = "terminal"
	ToolTypeEditor         ToolType = "editor"
	ToolTypeDebugger       ToolType = "debugger"
	ToolTypeLinter         ToolType = "linter"
	ToolTypeFormatter      ToolType = "formatter"
	ToolTypePackageManager ToolType = "package_manager"
)

// ToolSecurityLevel defines security levels for tools
type ToolSecurityLevel string

const (
	SecurityLevelSafe      ToolSecurityLevel = "safe"      // Read-only operations
	SecurityLevelModerate  ToolSecurityLevel = "moderate"  // Limited write operations
	SecurityLevelElevated  ToolSecurityLevel = "elevated"  // System modifications
	SecurityLevelDangerous ToolSecurityLevel = "dangerous" // Potentially harmful operations
)

// ToolExecutionLimits defines execution limits for tools
type ToolExecutionLimits struct {
	MaxExecutionTime   time.Duration `json:"max_execution_time"`
	MaxMemoryUsage     int64         `json:"max_memory_usage"` // bytes
	MaxCPUUsage        float64       `json:"max_cpu_usage"`    // percentage
	MaxFileSize        int64         `json:"max_file_size"`    // bytes
	MaxNetworkRequests int           `json:"max_network_requests"`
	MaxConcurrentOps   int           `json:"max_concurrent_ops"`
	MaxOutputSize      int64         `json:"max_output_size"` // bytes
	RateLimitPerMinute int           `json:"rate_limit_per_minute"`
	RateLimitPerHour   int           `json:"rate_limit_per_hour"`
	CooldownPeriod     time.Duration `json:"cooldown_period"`
}

// ToolPermissions defines what a tool is allowed to do
type ToolPermissions struct {
	// File system permissions
	AllowedPaths   []string `json:"allowed_paths"`
	ForbiddenPaths []string `json:"forbidden_paths"`
	CanRead        bool     `json:"can_read"`
	CanWrite       bool     `json:"can_write"`
	CanExecute     bool     `json:"can_execute"`
	CanDelete      bool     `json:"can_delete"`
	CanCreateFiles bool     `json:"can_create_files"`
	CanCreateDirs  bool     `json:"can_create_dirs"`

	// Network permissions
	CanMakeNetworkCalls bool     `json:"can_make_network_calls"`
	AllowedDomains      []string `json:"allowed_domains"`
	ForbiddenDomains    []string `json:"forbidden_domains"`
	AllowedPorts        []int    `json:"allowed_ports"`

	// System permissions
	CanAccessEnvVars  bool     `json:"can_access_env_vars"`
	CanModifyEnvVars  bool     `json:"can_modify_env_vars"`
	CanRunCommands    bool     `json:"can_run_commands"`
	AllowedCommands   []string `json:"allowed_commands"`
	ForbiddenCommands []string `json:"forbidden_commands"`

	// Database permissions
	CanAccessDatabase bool     `json:"can_access_database"`
	CanModifyDatabase bool     `json:"can_modify_database"`
	AllowedTables     []string `json:"allowed_tables"`

	// Git permissions
	CanReadGit        bool `json:"can_read_git"`
	CanModifyGit      bool `json:"can_modify_git"`
	CanPush           bool `json:"can_push"`
	CanCreateBranches bool `json:"can_create_branches"`
}

// ToolUsageMetrics tracks tool usage statistics
type ToolUsageMetrics struct {
	TotalExecutions      int64         `json:"total_executions"`
	SuccessfulExecutions int64         `json:"successful_executions"`
	FailedExecutions     int64         `json:"failed_executions"`
	AverageExecutionTime time.Duration `json:"average_execution_time"`
	TotalExecutionTime   time.Duration `json:"total_execution_time"`
	LastUsed             time.Time     `json:"last_used"`
	ErrorRate            float64       `json:"error_rate"`

	// Resource usage
	PeakMemoryUsage int64   `json:"peak_memory_usage"`
	TotalMemoryUsed int64   `json:"total_memory_used"`
	PeakCPUUsage    float64 `json:"peak_cpu_usage"`

	// Security events
	SecurityViolations int64 `json:"security_violations"`
	PermissionDenials  int64 `json:"permission_denials"`
	RateLimitHits      int64 `json:"rate_limit_hits"`
}

// ToolDependency represents a dependency between tools
type ToolDependency struct {
	ToolID     string            `json:"tool_id"`
	Version    string            `json:"version,omitempty"`
	Required   bool              `json:"required"`
	Conditions map[string]string `json:"conditions,omitempty"`
}

// EnhancedToolConfig represents comprehensive tool configuration
type EnhancedToolConfig struct {
	// Basic tool information
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Type        ToolType `json:"type"`
	Enabled     bool     `json:"enabled"`

	// Security configuration
	SecurityLevel   ToolSecurityLevel `json:"security_level"`
	Permissions     ToolPermissions   `json:"permissions"`
	RequiresAuth    bool              `json:"requires_auth"`
	RequiresConfirm bool              `json:"requires_confirm"`

	// Execution limits
	Limits ToolExecutionLimits `json:"limits"`

	// Dependencies
	Dependencies []ToolDependency `json:"dependencies"`

	// Configuration parameters
	Parameters map[string]any `json:"parameters,omitempty"`

	// Usage tracking
	Metrics ToolUsageMetrics `json:"metrics"`

	// Scheduling and availability
	AvailableHours  []string `json:"available_hours,omitempty"` // e.g., ["09:00-17:00"]
	MaintenanceMode bool     `json:"maintenance_mode"`

	// Tool-specific settings
	CustomSettings map[string]any `json:"custom_settings,omitempty"`

	// Audit and compliance
	AuditEnabled    bool   `json:"audit_enabled"`
	ComplianceLevel string `json:"compliance_level,omitempty"`

	// Performance optimization
	CacheEnabled bool          `json:"cache_enabled"`
	CacheTTL     time.Duration `json:"cache_ttl"`
	PreloadData  bool          `json:"preload_data"`

	// Error handling
	RetryPolicy   RetryPolicy `json:"retry_policy"`
	FallbackTools []string    `json:"fallback_tools,omitempty"`
}

// RetryPolicy defines retry behavior for tool execution
type RetryPolicy struct {
	Enabled           bool          `json:"enabled"`
	MaxRetries        int           `json:"max_retries"`
	InitialDelay      time.Duration `json:"initial_delay"`
	BackoffMultiplier float64       `json:"backoff_multiplier"`
	MaxDelay          time.Duration `json:"max_delay"`
	RetryableErrors   []string      `json:"retryable_errors"`
}

// ToolConfigManager manages tool configurations
type ToolConfigManager struct {
	configs   map[string]*EnhancedToolConfig
	templates map[string]*EnhancedToolConfig
	mu        sync.RWMutex

	// Global settings
	globalLimits   ToolExecutionLimits
	globalSecurity ToolSecurityLevel
	auditEnabled   bool

	// Usage tracking
	usageHistory   []ToolUsageRecord
	maxHistorySize int
}

// ToolUsageRecord represents a single tool usage event
type ToolUsageRecord struct {
	ID             string          `json:"id"`
	ToolID         string          `json:"tool_id"`
	SessionID      string          `json:"session_id"`
	UserID         string          `json:"user_id,omitempty"`
	Timestamp      time.Time       `json:"timestamp"`
	ExecutionTime  time.Duration   `json:"execution_time"`
	Success        bool            `json:"success"`
	ErrorMessage   string          `json:"error_message,omitempty"`
	ResourceUsage  ResourceUsage   `json:"resource_usage"`
	Parameters     map[string]any  `json:"parameters,omitempty"`
	Output         string          `json:"output,omitempty"`
	SecurityEvents []SecurityEvent `json:"security_events,omitempty"`
}

// ResourceUsage tracks resource consumption during tool execution
type ResourceUsage struct {
	MemoryUsed      int64   `json:"memory_used"`
	CPUUsed         float64 `json:"cpu_used"`
	NetworkRequests int     `json:"network_requests"`
	FilesAccessed   int     `json:"files_accessed"`
	OutputSize      int64   `json:"output_size"`
}

// SecurityEvent represents a security-related event during tool execution
type SecurityEvent struct {
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Severity    string    `json:"severity"`
	Timestamp   time.Time `json:"timestamp"`
	Action      string    `json:"action"` // allowed, denied, logged
}

// NewToolConfigManager creates a new tool configuration manager
func NewToolConfigManager() *ToolConfigManager {
	return &ToolConfigManager{
		configs:        make(map[string]*EnhancedToolConfig),
		templates:      make(map[string]*EnhancedToolConfig),
		maxHistorySize: 10000,
		globalLimits: ToolExecutionLimits{
			MaxExecutionTime:   30 * time.Second,
			MaxMemoryUsage:     100 * 1024 * 1024, // 100MB
			MaxCPUUsage:        50.0,              // 50%
			MaxFileSize:        10 * 1024 * 1024,  // 10MB
			MaxNetworkRequests: 10,
			MaxConcurrentOps:   5,
			MaxOutputSize:      1 * 1024 * 1024, // 1MB
			RateLimitPerMinute: 60,
			RateLimitPerHour:   1000,
			CooldownPeriod:     1 * time.Second,
		},
		globalSecurity: SecurityLevelModerate,
		auditEnabled:   true,
	}
}

// GetToolConfig returns the configuration for a tool
func (tcm *ToolConfigManager) GetToolConfig(toolID string) *EnhancedToolConfig {
	tcm.mu.RLock()
	defer tcm.mu.RUnlock()

	if config, exists := tcm.configs[toolID]; exists {
		return config
	}

	// Return default configuration
	return tcm.getDefaultToolConfig(toolID)
}

// SetToolConfig sets the configuration for a tool
func (tcm *ToolConfigManager) SetToolConfig(toolID string, config *EnhancedToolConfig) {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	config.ID = toolID
	tcm.configs[toolID] = config
}

// ValidateToolExecution validates if a tool execution is allowed
func (tcm *ToolConfigManager) ValidateToolExecution(toolID string, params map[string]any) error {
	config := tcm.GetToolConfig(toolID)

	if !config.Enabled {
		return fmt.Errorf("tool %s is disabled", toolID)
	}

	if config.MaintenanceMode {
		return fmt.Errorf("tool %s is in maintenance mode", toolID)
	}

	// Check rate limits
	if !tcm.checkRateLimit(toolID) {
		return fmt.Errorf("rate limit exceeded for tool %s", toolID)
	}

	// Check dependencies
	if err := tcm.checkDependencies(config); err != nil {
		return fmt.Errorf("dependency check failed for tool %s: %v", toolID, err)
	}

	// Check security permissions
	if err := tcm.validatePermissions(config, params); err != nil {
		return fmt.Errorf("permission denied for tool %s: %v", toolID, err)
	}

	return nil
}

// RecordToolUsage records a tool usage event
func (tcm *ToolConfigManager) RecordToolUsage(record ToolUsageRecord) {
	tcm.mu.Lock()
	defer tcm.mu.Unlock()

	// Set timestamp if not provided
	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now()
	}

	// Generate ID if not provided
	if record.ID == "" {
		record.ID = fmt.Sprintf("%s-%d", record.ToolID, record.Timestamp.Unix())
	}

	// Add to history
	tcm.usageHistory = append(tcm.usageHistory, record)

	// Trim history if necessary
	if len(tcm.usageHistory) > tcm.maxHistorySize {
		tcm.usageHistory = tcm.usageHistory[len(tcm.usageHistory)-tcm.maxHistorySize:]
	}

	// Update tool metrics
	if config, exists := tcm.configs[record.ToolID]; exists {
		tcm.updateToolMetrics(config, record)
	}
}

// GetToolUsageStats returns usage statistics for a tool
func (tcm *ToolConfigManager) GetToolUsageStats(toolID string, period time.Duration) *ToolUsageMetrics {
	tcm.mu.RLock()
	defer tcm.mu.RUnlock()

	cutoff := time.Now().Add(-period)
	metrics := &ToolUsageMetrics{}

	var totalExecutionTime time.Duration

	for _, record := range tcm.usageHistory {
		if record.ToolID == toolID && record.Timestamp.After(cutoff) {
			metrics.TotalExecutions++
			totalExecutionTime += record.ExecutionTime

			if record.Success {
				metrics.SuccessfulExecutions++
			} else {
				metrics.FailedExecutions++
			}

			if record.Timestamp.After(metrics.LastUsed) {
				metrics.LastUsed = record.Timestamp
			}

			// Track resource usage
			if record.ResourceUsage.MemoryUsed > metrics.PeakMemoryUsage {
				metrics.PeakMemoryUsage = record.ResourceUsage.MemoryUsed
			}
			metrics.TotalMemoryUsed += record.ResourceUsage.MemoryUsed

			if record.ResourceUsage.CPUUsed > metrics.PeakCPUUsage {
				metrics.PeakCPUUsage = record.ResourceUsage.CPUUsed
			}
		}
	}

	// Calculate averages
	if metrics.TotalExecutions > 0 {
		metrics.AverageExecutionTime = totalExecutionTime / time.Duration(metrics.TotalExecutions)
		metrics.ErrorRate = float64(metrics.FailedExecutions) / float64(metrics.TotalExecutions)
	}

	metrics.TotalExecutionTime = totalExecutionTime

	return metrics
}

// CreateToolTemplate creates a configuration template for a tool type
func (tcm *ToolConfigManager) CreateToolTemplate(toolType ToolType) *EnhancedToolConfig {
	template := &EnhancedToolConfig{
		Type:          toolType,
		Enabled:       true,
		Limits:        tcm.globalLimits,
		SecurityLevel: tcm.globalSecurity,
		RetryPolicy: RetryPolicy{
			Enabled:           true,
			MaxRetries:        3,
			InitialDelay:      1 * time.Second,
			BackoffMultiplier: 2.0,
			MaxDelay:          30 * time.Second,
		},
		AuditEnabled: tcm.auditEnabled,
		CacheEnabled: true,
		CacheTTL:     5 * time.Minute,
	}

	// Set type-specific defaults
	switch toolType {
	case ToolTypeFileSystem:
		template.SecurityLevel = SecurityLevelModerate
		template.Permissions = ToolPermissions{
			CanRead:        true,
			CanWrite:       true,
			CanCreateFiles: true,
			CanCreateDirs:  true,
			AllowedPaths:   []string{"./"},
			ForbiddenPaths: []string{"/etc", "/sys", "/proc"},
		}

	case ToolTypeCodeExecution:
		template.SecurityLevel = SecurityLevelElevated
		template.RequiresConfirm = true
		template.Limits.MaxExecutionTime = 60 * time.Second
		template.Permissions = ToolPermissions{
			CanRunCommands:  true,
			AllowedCommands: []string{"go", "python", "node", "cargo"},
		}

	case ToolTypeWebSearch:
		template.SecurityLevel = SecurityLevelSafe
		template.Permissions = ToolPermissions{
			CanMakeNetworkCalls: true,
			AllowedDomains:      []string{"google.com", "bing.com", "duckduckgo.com"},
		}

	case ToolTypeTerminal:
		template.SecurityLevel = SecurityLevelDangerous
		template.RequiresAuth = true
		template.RequiresConfirm = true
		template.Permissions = ToolPermissions{
			CanRunCommands: true,
		}
	}

	return template
}

// getDefaultToolConfig returns a default configuration for a tool
func (tcm *ToolConfigManager) getDefaultToolConfig(toolID string) *EnhancedToolConfig {
	// Try to infer tool type from ID
	var toolType ToolType
	switch {
	case toolID == "file_reader" || toolID == "file_writer":
		toolType = ToolTypeFileSystem
	case toolID == "code_executor":
		toolType = ToolTypeCodeExecution
	case toolID == "web_search":
		toolType = ToolTypeWebSearch
	case toolID == "terminal":
		toolType = ToolTypeTerminal
	default:
		toolType = ToolTypeFileSystem // Default
	}

	config := tcm.CreateToolTemplate(toolType)
	config.ID = toolID
	config.Name = toolID

	return config
}

// checkRateLimit checks if a tool is within rate limits
func (tcm *ToolConfigManager) checkRateLimit(toolID string) bool {
	tcm.mu.RLock()
	defer tcm.mu.RUnlock()

	config, exists := tcm.configs[toolID]
	if !exists {
		return true // Allow if no config exists
	}

	now := time.Now()

	// Check requests per minute limit
	if config.Limits.RateLimitPerMinute > 0 {
		minuteAgo := now.Add(-time.Minute)
		recentRequests := 0

		for _, record := range tcm.usageHistory {
			if record.ToolID == toolID && record.Timestamp.After(minuteAgo) {
				recentRequests++
			}
		}

		if recentRequests >= config.Limits.RateLimitPerMinute {
			return false
		}
	}

	// Check requests per hour limit
	if config.Limits.RateLimitPerHour > 0 {
		hourAgo := now.Add(-time.Hour)
		recentRequests := 0

		for _, record := range tcm.usageHistory {
			if record.ToolID == toolID && record.Timestamp.After(hourAgo) {
				recentRequests++
			}
		}

		if recentRequests >= config.Limits.RateLimitPerHour {
			return false
		}
	}

	return true
}

// checkDependencies checks if tool dependencies are satisfied
func (tcm *ToolConfigManager) checkDependencies(config *EnhancedToolConfig) error {
	for _, dep := range config.Dependencies {
		depConfig := tcm.GetToolConfig(dep.ToolID)
		if dep.Required && !depConfig.Enabled {
			return fmt.Errorf("required dependency %s is not enabled", dep.ToolID)
		}
	}
	return nil
}

// validatePermissions validates tool permissions for an operation
func (tcm *ToolConfigManager) validatePermissions(config *EnhancedToolConfig, params map[string]any) error {
	if !config.Enabled {
		return fmt.Errorf("tool is disabled")
	}

	// Check security level requirements
	if config.SecurityLevel == SecurityLevelDangerous {
		// Require explicit confirmation for dangerous operations
		if confirmed, ok := params["confirm_dangerous"]; !ok || confirmed != true {
			return fmt.Errorf("dangerous operation requires explicit confirmation")
		}
	}

	// Validate file system permissions
	if config.Permissions.CanRead || config.Permissions.CanWrite {
		if path, ok := params["path"].(string); ok {
			// Check if path is within allowed directories
			if len(config.Permissions.AllowedPaths) > 0 {
				allowed := false
				for _, allowedPath := range config.Permissions.AllowedPaths {
					if strings.HasPrefix(path, allowedPath) {
						allowed = true
						break
					}
				}
				if !allowed {
					return fmt.Errorf("path %s not in allowed paths", path)
				}
			}

			// Check for forbidden paths
			for _, forbiddenPath := range config.Permissions.ForbiddenPaths {
				if strings.HasPrefix(path, forbiddenPath) {
					return fmt.Errorf("path %s is forbidden", path)
				}
			}
		}
	}

	// Validate network permissions
	if config.Permissions.CanMakeNetworkCalls {
		if url, ok := params["url"].(string); ok {
			// Check allowed domains
			if len(config.Permissions.AllowedDomains) > 0 {
				allowed := false
				for _, domain := range config.Permissions.AllowedDomains {
					if strings.Contains(url, domain) {
						allowed = true
						break
					}
				}
				if !allowed {
					return fmt.Errorf("domain not in allowed list")
				}
			}

			// Check forbidden domains
			for _, forbiddenDomain := range config.Permissions.ForbiddenDomains {
				if strings.Contains(url, forbiddenDomain) {
					return fmt.Errorf("domain %s is forbidden", forbiddenDomain)
				}
			}
		}
	} else if _, hasURL := params["url"]; hasURL {
		return fmt.Errorf("network access not permitted for this tool")
	}

	// Validate system command permissions
	if config.Permissions.CanRunCommands {
		if command, ok := params["command"].(string); ok {
			// Check allowed commands
			if len(config.Permissions.AllowedCommands) > 0 {
				allowed := false
				for _, allowedCmd := range config.Permissions.AllowedCommands {
					if strings.HasPrefix(command, allowedCmd) {
						allowed = true
						break
					}
				}
				if !allowed {
					return fmt.Errorf("command %s not in allowed list", command)
				}
			}

			// Check forbidden commands
			for _, forbiddenCmd := range config.Permissions.ForbiddenCommands {
				if strings.Contains(command, forbiddenCmd) {
					return fmt.Errorf("command contains forbidden term: %s", forbiddenCmd)
				}
			}
		}
	} else if _, hasCommand := params["command"]; hasCommand {
		return fmt.Errorf("system command execution not permitted for this tool")
	}

	return nil
}

// updateToolMetrics updates metrics for a tool based on usage record
func (tcm *ToolConfigManager) updateToolMetrics(config *EnhancedToolConfig, record ToolUsageRecord) {
	config.Metrics.TotalExecutions++
	config.Metrics.LastUsed = record.Timestamp

	if record.Success {
		config.Metrics.SuccessfulExecutions++
	} else {
		config.Metrics.FailedExecutions++
	}

	// Update averages
	if config.Metrics.TotalExecutions > 0 {
		config.Metrics.AverageExecutionTime = time.Duration(
			(int64(config.Metrics.AverageExecutionTime)*config.Metrics.TotalExecutions + int64(record.ExecutionTime)) /
				(config.Metrics.TotalExecutions + 1),
		)

		config.Metrics.ErrorRate = float64(config.Metrics.FailedExecutions) / float64(config.Metrics.TotalExecutions)
	}

	config.Metrics.TotalExecutionTime += record.ExecutionTime

	// Update resource usage
	if record.ResourceUsage.MemoryUsed > config.Metrics.PeakMemoryUsage {
		config.Metrics.PeakMemoryUsage = record.ResourceUsage.MemoryUsed
	}

	if record.ResourceUsage.CPUUsed > config.Metrics.PeakCPUUsage {
		config.Metrics.PeakCPUUsage = record.ResourceUsage.CPUUsed
	}
}
