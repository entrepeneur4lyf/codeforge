package config

import (
	"fmt"
	"os"
	"strings"

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
