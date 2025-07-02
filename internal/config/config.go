package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/models"
	"github.com/spf13/viper"
)

// AgentName represents different agent types
type AgentName string

const (
	AgentCoder      AgentName = "coder"
	AgentSummarizer AgentName = "summarizer"
	AgentTask       AgentName = "task"
	AgentTitle      AgentName = "title"
)

// Agent defines configuration for different LLM models and their token limits
type Agent struct {
	Model           models.ModelID `json:"model"`
	MaxTokens       int64          `json:"maxTokens"`
	ReasoningEffort string         `json:"reasoningEffort"`
}

// ModelConfig defines model-specific configuration including context limits
type ModelConfig struct {
	ContextWindow      int     `json:"contextWindow"`      // Maximum context window size
	MaxOutputTokens    int     `json:"maxOutputTokens"`    // Maximum output tokens
	CostPer1KInput     float64 `json:"costPer1KInput"`     // Cost per 1K input tokens
	CostPer1KOutput    float64 `json:"costPer1KOutput"`    // Cost per 1K output tokens
	SupportsTools      bool    `json:"supportsTools"`      // Whether model supports tool calling
	SupportsReasoning  bool    `json:"supportsReasoning"`  // Whether model supports reasoning
	SummarizeThreshold float64 `json:"summarizeThreshold"` // Threshold (0.0-1.0) to trigger summarization
}

// ContextConfig defines context management configuration
type ContextConfig struct {
	AutoSummarize      bool    `json:"autoSummarize"`      // Enable automatic summarization
	SlidingWindow      bool    `json:"slidingWindow"`      // Enable sliding window
	WindowOverlap      int     `json:"windowOverlap"`      // Overlap size for sliding window
	CacheEnabled       bool    `json:"cacheEnabled"`       // Enable context caching
	CacheTTL           int     `json:"cacheTTL"`           // Cache TTL in seconds
	MaxCacheSize       int     `json:"maxCacheSize"`       // Maximum cache entries
	CompressionLevel   int     `json:"compressionLevel"`   // Context compression level (0-9)
	RelevanceThreshold float64 `json:"relevanceThreshold"` // Minimum relevance score for inclusion
}

// Provider defines configuration for an LLM provider
type Provider struct {
	APIKey   string `json:"apiKey"`
	Disabled bool   `json:"disabled"`
}

// Data defines storage configuration
type Data struct {
	Directory string `json:"directory,omitempty"`
}

// TUIConfig defines terminal UI configuration
type TUIConfig struct {
	Theme string `json:"theme"`
}

// ShellConfig defines shell configuration
type ShellConfig struct {
	Path string   `json:"path"`
	Args []string `json:"args"`
}

// EmbeddingConfig defines embedding service configuration
type EmbeddingConfig struct {
	Provider string `json:"provider"` // "ollama", "openai", "auto"
	Model    string `json:"model"`    // e.g., "nomic-embed-text"
	BaseURL  string `json:"baseURL"`  // for custom Ollama instances
}

// MCPServer defines MCP server configuration
type MCPServer struct {
	Command []string          `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// LSPConfig defines Language Server Protocol configuration
type LSPConfig struct {
	Command []string          `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// MCPConfig defines Model Context Protocol configuration
type MCPConfig struct {
	Command     []string          `json:"command"`
	Args        []string          `json:"args,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	Description string            `json:"description,omitempty"`
	Enabled     bool              `json:"enabled"`
}

// PermissionConfig defines permission system configuration
type PermissionConfig struct {
	Enabled              bool   `json:"enabled"`              // Enable permission system
	RequireApproval      bool   `json:"requireApproval"`      // Require approval for operations
	AutoApproveThreshold int    `json:"autoApproveThreshold"` // Trust level threshold for auto-approval
	MaxPerSession        int    `json:"maxPerSession"`        // Maximum permissions per session
	AuditEnabled         bool   `json:"auditEnabled"`         // Enable audit logging
	DefaultExpiration    string `json:"defaultExpiration"`    // Default permission expiration (e.g., "24h")
	CleanupInterval      string `json:"cleanupInterval"`      // Cleanup interval (e.g., "1h")
	DatabasePath         string `json:"databasePath"`         // Path to permission database
}

// Config is the main configuration structure for the application
type Config struct {
	Data         Data                              `json:"data"`
	WorkingDir   string                            `json:"wd,omitempty"`
	MCPServers   map[string]MCPServer              `json:"mcpServers,omitempty"`
	Providers    map[models.ModelProvider]Provider `json:"providers,omitempty"`
	LSP          map[string]LSPConfig              `json:"lsp,omitempty"`
	MCP          map[string]MCPConfig              `json:"mcp,omitempty"`
	Agents       map[AgentName]Agent               `json:"agents,omitempty"`
	Debug        bool                              `json:"debug,omitempty"`
	DebugLSP     bool                              `json:"debugLSP,omitempty"`
	ContextPaths []string                          `json:"contextPaths,omitempty"`
	TUI          TUIConfig                         `json:"tui"`
	Shell        ShellConfig                       `json:"shell,omitempty"`
	Embedding    EmbeddingConfig                   `json:"embedding,omitempty"`
	AutoCompact  bool                              `json:"autoCompact,omitempty"`
	Models       map[string]ModelConfig            `json:"models,omitempty"` // Model-specific configurations
	Context      ContextConfig                     `json:"context"`          // Context management configuration
	Permissions  PermissionConfig                  `json:"permissions"`      // Permission system configuration

	// Enhanced configuration managers (Phase 4)
	ModelConfigManager *ModelConfigManager `json:"-"` // Enhanced model configuration manager
	ProviderManager    *ProviderManager    `json:"-"` // Provider health and load balancing manager
	CostTracker        *CostTracker        `json:"-"` // Cost tracking and optimization
	ToolConfigManager  *ToolConfigManager  `json:"-"` // Tool configuration and security manager
}

// Application constants
const (
	defaultDataDirectory     = ".codeforge"
	defaultLogLevel          = "info"
	appName                  = "codeforge"
	MaxTokensFallbackDefault = 4096
)

var defaultContextPaths = []string{
	".github/copilot-instructions.md",
	".cursorrules",
	".clinerules",
	".windsurfrules",
	".cursor/rules/",
	"CLAUDE.md",
	"CLAUDE.local.md",
	"codeforge.md",
	"codeforge.local.md",
	"CodeForge.md",
	"CodeForge.local.md",
	"CODEFORGE.md",
	"CODEFORGE.local.md",
	"AGENTS.md",
}

// Global configuration instance
var cfg *Config

// Load initializes the configuration from environment variables and config files
func Load(workingDir string, debug bool) (*Config, error) {
	if cfg != nil {
		return cfg, nil
	}

	cfg = &Config{
		WorkingDir: workingDir,
		MCPServers: make(map[string]MCPServer),
		Providers:  make(map[models.ModelProvider]Provider),
		LSP:        make(map[string]LSPConfig),
		Agents:     make(map[AgentName]Agent),
	}

	configureViper()
	setDefaults(debug)

	// Read global config
	if err := readConfig(viper.ReadInConfig()); err != nil {
		return cfg, err
	}

	// Load providers from environment variables
	loadProvidersFromEnv()

	// Set default agents based on available providers
	setDefaultAgents()

	// Initialize enhanced configuration managers (Phase 4)
	initializeEnhancedManagers()

	// Ensure Data.Directory is set if empty
	if cfg.Data.Directory == "" {
		cfg.Data.Directory = defaultDataDirectory
	}

	return cfg, nil
}

// configureViper sets up viper's configuration paths and environment variables
func configureViper() {
	viper.SetConfigName(fmt.Sprintf(".%s", appName))
	viper.SetConfigType("json")
	viper.AddConfigPath("$HOME")
	viper.AddConfigPath(fmt.Sprintf("$XDG_CONFIG_HOME/%s", appName))
	viper.AddConfigPath(fmt.Sprintf("$HOME/.config/%s", appName))
	viper.SetEnvPrefix(strings.ToUpper(appName))
	viper.AutomaticEnv()
}

// setDefaults configures default values for configuration options
func setDefaults(debug bool) {
	viper.SetDefault("data.directory", defaultDataDirectory)
	viper.SetDefault("contextPaths", defaultContextPaths)
	viper.SetDefault("tui.theme", "codeforge")
	viper.SetDefault("autoCompact", true)

	// Context management defaults
	viper.SetDefault("context.autoSummarize", true)
	viper.SetDefault("context.slidingWindow", true)

	// Permission system defaults
	viper.SetDefault("permissions.enabled", true)
	viper.SetDefault("permissions.requireApproval", true)
	viper.SetDefault("permissions.autoApproveThreshold", 80)
	viper.SetDefault("permissions.maxPerSession", 100)
	viper.SetDefault("permissions.auditEnabled", true)
	viper.SetDefault("permissions.defaultExpiration", "24h")
	viper.SetDefault("permissions.cleanupInterval", "1h")
	viper.SetDefault("permissions.databasePath", "")
	viper.SetDefault("context.windowOverlap", 200)
	viper.SetDefault("context.cacheEnabled", true)
	viper.SetDefault("context.cacheTTL", 3600) // 1 hour
	viper.SetDefault("context.maxCacheSize", 1000)
	viper.SetDefault("context.compressionLevel", 3)
	viper.SetDefault("context.relevanceThreshold", 0.1)

	// Set default shell from environment or fallback to /bin/bash
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		shellPath = "/bin/bash"
	}
	viper.SetDefault("shell.path", shellPath)
	viper.SetDefault("shell.args", []string{"-l"})

	if debug {
		viper.SetDefault("debug", true)
		viper.Set("log.level", "debug")
	} else {
		viper.SetDefault("debug", false)
		viper.SetDefault("log.level", defaultLogLevel)
	}
}

// readConfig reads configuration from file and environment
func readConfig(err error) error {
	if err != nil {
		// Config file not found; ignore error if desired
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error
			return nil
		} else {
			// Config file was found but another error was produced
			return fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Unmarshal config into struct
	if err := viper.Unmarshal(cfg); err != nil {
		return fmt.Errorf("unable to decode config: %w", err)
	}

	return nil
}

// loadProvidersFromEnv loads provider configurations from environment variables
func loadProvidersFromEnv() {
	providers := map[models.ModelProvider]string{
		models.ProviderCopilot:    "COPILOT_API_KEY",
		models.ProviderAnthropic:  "ANTHROPIC_API_KEY",
		models.ProviderOpenAI:     "OPENAI_API_KEY",
		models.ProviderGemini:     "GEMINI_API_KEY",
		models.ProviderGROQ:       "GROQ_API_KEY",
		models.ProviderOpenRouter: "OPENROUTER_API_KEY",
		models.ProviderBedrock:    "AWS_ACCESS_KEY_ID", // Special case for AWS
		models.ProviderAzure:      "AZURE_OPENAI_API_KEY",
		models.ProviderVertexAI:   "GOOGLE_APPLICATION_CREDENTIALS", // Special case for GCP
		models.ProviderXAI:        "XAI_API_KEY",
		models.ProviderLocal:      "LOCAL_ENDPOINT", // Special case for local
	}

	for provider, envVar := range providers {
		if apiKey := os.Getenv(envVar); apiKey != "" {
			if cfg.Providers == nil {
				cfg.Providers = make(map[models.ModelProvider]Provider)
			}
			cfg.Providers[provider] = Provider{
				APIKey:   apiKey,
				Disabled: false,
			}
		}
	}
}

// setDefaultAgents sets default agent configurations based on available providers
func setDefaultAgents() {
	// Provider priority order (same as OpenCode)
	providerOrder := []models.ModelProvider{
		models.ProviderCopilot,
		models.ProviderAnthropic,
		models.ProviderOpenAI,
		models.ProviderGemini,
		models.ProviderGROQ,
		models.ProviderOpenRouter,
		models.ProviderBedrock,
		models.ProviderAzure,
		models.ProviderVertexAI,
	}

	// Find the first available provider
	var selectedProvider models.ModelProvider
	for _, provider := range providerOrder {
		if providerCfg, exists := cfg.Providers[provider]; exists && !providerCfg.Disabled {
			selectedProvider = provider
			break
		}
	}

	// Set default agents if we have a provider
	if selectedProvider != "" {
		// Get the default model for this provider
		defaultModel, exists := models.GetDefaultModelForProvider(selectedProvider)
		if !exists {
			return
		}

		cfg.Agents[AgentCoder] = Agent{
			Model:     defaultModel.ID,
			MaxTokens: defaultModel.DefaultMaxTokens,
		}
		cfg.Agents[AgentSummarizer] = Agent{
			Model:     defaultModel.ID,
			MaxTokens: defaultModel.DefaultMaxTokens,
		}
		cfg.Agents[AgentTask] = Agent{
			Model:     defaultModel.ID,
			MaxTokens: defaultModel.DefaultMaxTokens,
		}
		cfg.Agents[AgentTitle] = Agent{
			Model:     defaultModel.ID,
			MaxTokens: 80,
		}
	}
}

// Get returns the global configuration instance
func Get() *Config {
	return cfg
}

// WorkingDirectory returns the current working directory from the configuration
func WorkingDirectory() string {
	if cfg == nil {
		panic("config not loaded")
	}
	return cfg.WorkingDir
}

// GetModelConfig returns configuration for a specific model
func (c *Config) GetModelConfig(modelID string) ModelConfig {
	if config, exists := c.Models[modelID]; exists {
		return config
	}

	// Return default configuration based on model type
	return getDefaultModelConfig(modelID)
}

// getDefaultModelConfig returns default configuration for common models
func getDefaultModelConfig(modelID string) ModelConfig {
	modelLower := strings.ToLower(modelID)

	switch {
	case strings.Contains(modelLower, "gpt-4o"):
		return ModelConfig{
			ContextWindow:      128000,
			MaxOutputTokens:    4096,
			CostPer1KInput:     0.0025,
			CostPer1KOutput:    0.01,
			SupportsTools:      true,
			SupportsReasoning:  false,
			SummarizeThreshold: 0.9,
		}
	case strings.Contains(modelLower, "gpt-4"):
		return ModelConfig{
			ContextWindow:      8192,
			MaxOutputTokens:    4096,
			CostPer1KInput:     0.03,
			CostPer1KOutput:    0.06,
			SupportsTools:      true,
			SupportsReasoning:  false,
			SummarizeThreshold: 0.9,
		}
	case strings.Contains(modelLower, "gpt-3.5"):
		return ModelConfig{
			ContextWindow:      16385,
			MaxOutputTokens:    4096,
			CostPer1KInput:     0.0015,
			CostPer1KOutput:    0.002,
			SupportsTools:      true,
			SupportsReasoning:  false,
			SummarizeThreshold: 0.9,
		}
	case strings.Contains(modelLower, "claude-3.5-sonnet"):
		return ModelConfig{
			ContextWindow:      200000,
			MaxOutputTokens:    8192,
			CostPer1KInput:     0.003,
			CostPer1KOutput:    0.015,
			SupportsTools:      true,
			SupportsReasoning:  false,
			SummarizeThreshold: 0.9,
		}
	case strings.Contains(modelLower, "claude-3"):
		return ModelConfig{
			ContextWindow:      200000,
			MaxOutputTokens:    4096,
			CostPer1KInput:     0.003,
			CostPer1KOutput:    0.015,
			SupportsTools:      true,
			SupportsReasoning:  false,
			SummarizeThreshold: 0.9,
		}
	case strings.Contains(modelLower, "gemini-1.5-pro"):
		return ModelConfig{
			ContextWindow:      2097152, // 2M tokens
			MaxOutputTokens:    8192,
			CostPer1KInput:     0.00125,
			CostPer1KOutput:    0.005,
			SupportsTools:      true,
			SupportsReasoning:  false,
			SummarizeThreshold: 0.95, // Higher threshold due to large context
		}
	case strings.Contains(modelLower, "gemini"):
		return ModelConfig{
			ContextWindow:      32768,
			MaxOutputTokens:    8192,
			CostPer1KInput:     0.00075,
			CostPer1KOutput:    0.003,
			SupportsTools:      true,
			SupportsReasoning:  false,
			SummarizeThreshold: 0.9,
		}
	default:
		// Generic default
		return ModelConfig{
			ContextWindow:      8192,
			MaxOutputTokens:    2048,
			CostPer1KInput:     0.001,
			CostPer1KOutput:    0.002,
			SupportsTools:      false,
			SupportsReasoning:  false,
			SummarizeThreshold: 0.9,
		}
	}
}

// ShouldSummarize determines if conversation should be summarized based on token usage
func (c *Config) ShouldSummarize(modelID string, currentTokens int) bool {
	modelConfig := c.GetModelConfig(modelID)
	threshold := int(float64(modelConfig.ContextWindow) * modelConfig.SummarizeThreshold)
	return currentTokens >= threshold
}

// GetContextConfig returns the context management configuration
func (c *Config) GetContextConfig() ContextConfig {
	return c.Context
}

// GetProvider returns the configuration for a specific provider
func GetProvider(provider models.ModelProvider) (Provider, bool) {
	if cfg == nil {
		return Provider{}, false
	}
	providerCfg, exists := cfg.Providers[provider]
	return providerCfg, exists
}

// GetAgent returns the configuration for a specific agent
func GetAgent(agent AgentName) (Agent, bool) {
	if cfg == nil {
		return Agent{}, false
	}
	agentCfg, exists := cfg.Agents[agent]
	return agentCfg, exists
}

// Helper methods for accessing configuration values with defaults

// GetString returns a string configuration value with a default
func (c *Config) GetString(key, defaultValue string) string {
	if viper.IsSet(key) {
		return viper.GetString(key)
	}
	return defaultValue
}

// GetInt returns an integer configuration value with a default
func (c *Config) GetInt(key string, defaultValue int) int {
	if viper.IsSet(key) {
		return viper.GetInt(key)
	}
	return defaultValue
}

// GetBool returns a boolean configuration value with a default
func (c *Config) GetBool(key string, defaultValue bool) bool {
	if viper.IsSet(key) {
		return viper.GetBool(key)
	}
	return defaultValue
}

// GetDuration returns a duration configuration value with a default
func (c *Config) GetDuration(key, defaultValue string) time.Duration {
	if viper.IsSet(key) {
		return viper.GetDuration(key)
	}
	duration, _ := time.ParseDuration(defaultValue)
	return duration
}

// GetPermissionConfig returns the permission system configuration
func (c *Config) GetPermissionConfig() PermissionConfig {
	return c.Permissions
}

// initializeEnhancedManagers initializes the enhanced configuration managers for Phase 4
func initializeEnhancedManagers() {
	if cfg == nil {
		return
	}

	// Initialize enhanced model configuration manager
	cfg.ModelConfigManager = NewModelConfigManager()

	// Initialize provider manager with health monitoring
	cfg.ProviderManager = NewProviderManager()

	// Initialize cost tracker for optimization
	cfg.CostTracker = NewCostTracker()

	// Initialize tool configuration manager
	cfg.ToolConfigManager = NewToolConfigManager()

	// Load existing model configurations into enhanced manager
	for modelID, modelConfig := range cfg.Models {
		enhancedConfig := &EnhancedModelConfig{
			ContextWindow:      modelConfig.ContextWindow,
			MaxOutputTokens:    modelConfig.MaxOutputTokens,
			CostPer1KInput:     modelConfig.CostPer1KInput,
			CostPer1KOutput:    modelConfig.CostPer1KOutput,
			SupportsTools:      modelConfig.SupportsTools,
			SupportsReasoning:  modelConfig.SupportsReasoning,
			SummarizeThreshold: modelConfig.SummarizeThreshold,
		}
		cfg.ModelConfigManager.SetModelConfig(models.CanonicalModelID(modelID), enhancedConfig)
	}

	// Initialize provider configurations
	for providerID := range cfg.Providers {
		providerConfig := cfg.ProviderManager.GetProviderConfig(providerID)
		cfg.ProviderManager.SetProviderConfig(providerID, providerConfig)
	}

	// Set up default budgets for cost tracking
	cfg.CostTracker.SetBudget(PeriodHourly, 10.0)    // $10/hour default
	cfg.CostTracker.SetBudget(PeriodDaily, 100.0)    // $100/day default
	cfg.CostTracker.SetBudget(PeriodMonthly, 1000.0) // $1000/month default

	// Create default tool configurations
	defaultTools := []string{
		"file_reader", "file_writer", "code_executor", "web_search",
		"terminal", "git", "linter", "formatter", "package_manager",
	}

	for _, toolID := range defaultTools {
		toolConfig := cfg.ToolConfigManager.GetToolConfig(toolID)
		cfg.ToolConfigManager.SetToolConfig(toolID, toolConfig)
	}
}

// GetEnhancedModelConfig returns enhanced model configuration
func (c *Config) GetEnhancedModelConfig(modelID models.ModelID) *EnhancedModelConfig {
	if c.ModelConfigManager == nil {
		return nil
	}
	return c.ModelConfigManager.GetModelConfig(models.CanonicalModelID(modelID))
}

// GetProviderConfig returns provider configuration with health status
func (c *Config) GetProviderConfig(providerID models.ModelProvider) *EnhancedProviderConfig {
	if c.ProviderManager == nil {
		return nil
	}
	return c.ProviderManager.GetProviderConfig(providerID)
}

// GetToolConfig returns tool configuration with security settings
func (c *Config) GetToolConfig(toolID string) *EnhancedToolConfig {
	if c.ToolConfigManager == nil {
		return nil
	}
	return c.ToolConfigManager.GetToolConfig(toolID)
}

// RecordTokenUsage records token usage for cost tracking and optimization
func (c *Config) RecordTokenUsage(record TokenUsageRecord) error {
	if c.CostTracker == nil {
		return fmt.Errorf("cost tracker not initialized")
	}
	return c.CostTracker.RecordUsage(record)
}

// GetCostSummary returns cost summary for a specific period
func (c *Config) GetCostSummary(period CostPeriod, startTime, endTime time.Time) *CostSummary {
	if c.CostTracker == nil {
		return nil
	}
	return c.CostTracker.GetCostSummary(period, startTime, endTime)
}

// IsWithinBudget checks if a cost would exceed the budget
func (c *Config) IsWithinBudget(period CostPeriod, additionalCost float64) bool {
	if c.CostTracker == nil {
		return true // Allow if tracker not available
	}
	return c.CostTracker.IsWithinBudget(period, additionalCost)
}

// GetOptimizationRecommendations returns cost optimization recommendations
func (c *Config) GetOptimizationRecommendations() []CostOptimizationRecommendation {
	if c.CostTracker == nil {
		return nil
	}
	return c.CostTracker.GetOptimizationRecommendations()
}

// SelectOptimalProvider selects the best provider based on load balancing strategy
func (c *Config) SelectOptimalProvider(providers []models.ModelProvider, strategy LoadBalancingStrategy) (models.ModelProvider, error) {
	if c.ProviderManager == nil {
		if len(providers) > 0 {
			return providers[0], nil // Fallback to first provider
		}
		return "", fmt.Errorf("no providers available")
	}
	return c.ProviderManager.SelectProvider(providers, strategy)
}

// ValidateToolExecution validates if a tool execution is allowed
func (c *Config) ValidateToolExecution(toolID string, params map[string]any) error {
	if c.ToolConfigManager == nil {
		return nil // Allow if manager not available
	}
	return c.ToolConfigManager.ValidateToolExecution(toolID, params)
}

// RecordToolUsage records tool usage for analytics and security monitoring
func (c *Config) RecordToolUsage(record ToolUsageRecord) {
	if c.ToolConfigManager != nil {
		c.ToolConfigManager.RecordToolUsage(record)
	}
}

// updateCfgFile updates the configuration file with the provided update function
func updateCfgFile(updateCfg func(config *Config)) error {
	if cfg == nil {
		return fmt.Errorf("config not loaded")
	}

	// Get the config file path
	configFile := viper.ConfigFileUsed()
	var configData []byte
	if configFile == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		configFile = filepath.Join(homeDir, fmt.Sprintf(".%s.json", appName))
		configData = []byte(`{}`)
	} else {
		// Read the existing config file
		data, err := os.ReadFile(configFile)
		if err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		configData = data
	}

	// Parse the JSON
	var userCfg *Config
	if err := json.Unmarshal(configData, &userCfg); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	updateCfg(userCfg)

	// Write the updated config back to file
	updatedData, err := json.MarshalIndent(userCfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configFile, updatedData, 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// UpdateTheme updates the theme in the configuration and writes it to the config file
func UpdateTheme(themeName string) error {
	if cfg == nil {
		return fmt.Errorf("config not loaded")
	}

	// Update the in-memory config
	cfg.TUI.Theme = themeName

	// Update the file config
	return updateCfgFile(func(config *Config) {
		config.TUI.Theme = themeName
	})
}
