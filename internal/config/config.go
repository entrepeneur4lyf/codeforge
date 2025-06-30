package config

import (
	"fmt"
	"os"
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
	".cursor/rules/",
	"CLAUDE.md",
	"CLAUDE.local.md",
	"codeforge.md",
	"codeforge.local.md",
	"CodeForge.md",
	"CodeForge.local.md",
	"CODEFORGE.md",
	"CODEFORGE.local.md",
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
