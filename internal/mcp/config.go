package mcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// MCPServerType defines the type of MCP server connection
type MCPServerType string

const (
	MCPServerTypeLocal  MCPServerType = "local"
	MCPServerTypeRemote MCPServerType = "remote"
	MCPServerTypeSSE    MCPServerType = "sse"
	MCPServerTypeHTTP   MCPServerType = "http"
)

// MCPServerConfig defines configuration for an MCP server
type MCPServerConfig struct {
	Name        string        `json:"name"`
	Type        MCPServerType `json:"type"`
	Description string        `json:"description,omitempty"`
	Enabled     bool          `json:"enabled"`

	// Local server configuration
	Command     []string          `json:"command,omitempty"`
	Args        []string          `json:"args,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	WorkingDir  string            `json:"workingDir,omitempty"`

	// Remote server configuration
	URL string `json:"url,omitempty"`

	// Connection settings
	Timeout time.Duration `json:"timeout,omitempty"`
	Retries int           `json:"retries,omitempty"`

	// Capabilities
	Tools     bool `json:"tools,omitempty"`
	Resources bool `json:"resources,omitempty"`
	Prompts   bool `json:"prompts,omitempty"`

	// Security settings
	AllowedPaths []string `json:"allowedPaths,omitempty"`
	DeniedPaths  []string `json:"deniedPaths,omitempty"`

	// Metadata
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// MCPConfig manages all MCP server configurations
type MCPConfig struct {
	Servers        map[string]*MCPServerConfig `json:"servers"`
	GlobalSettings MCPGlobalSettings           `json:"globalSettings"`
	configPath     string
}

// MCPGlobalSettings defines global MCP configuration
type MCPGlobalSettings struct {
	DefaultTimeout      time.Duration `json:"defaultTimeout"`
	DefaultRetries      int           `json:"defaultRetries"`
	EnableLogging       bool          `json:"enableLogging"`
	LogLevel            string        `json:"logLevel"`
	MaxConcurrent       int           `json:"maxConcurrent"`
	HealthCheckInterval time.Duration `json:"healthCheckInterval"`
}

// NewMCPConfig creates a new MCP configuration manager
func NewMCPConfig(configDir string) *MCPConfig {
	configPath := filepath.Join(configDir, "mcp-config.json")

	config := &MCPConfig{
		Servers:    make(map[string]*MCPServerConfig),
		configPath: configPath,
		GlobalSettings: MCPGlobalSettings{
			DefaultTimeout:      30 * time.Second,
			DefaultRetries:      3,
			EnableLogging:       true,
			LogLevel:            "info",
			MaxConcurrent:       10,
			HealthCheckInterval: 5 * time.Minute,
		},
	}

	// Load existing configuration if it exists
	config.Load()

	return config
}

// Load loads MCP configuration from file
func (mc *MCPConfig) Load() error {
	if _, err := os.Stat(mc.configPath); os.IsNotExist(err) {
		// Create default configuration
		return mc.Save()
	}

	data, err := os.ReadFile(mc.configPath)
	if err != nil {
		return fmt.Errorf("failed to read MCP config: %w", err)
	}

	if err := json.Unmarshal(data, mc); err != nil {
		return fmt.Errorf("failed to parse MCP config: %w", err)
	}

	return nil
}

// Save saves MCP configuration to file
func (mc *MCPConfig) Save() error {
	// Ensure config directory exists
	if err := os.MkdirAll(filepath.Dir(mc.configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(mc, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal MCP config: %w", err)
	}

	if err := os.WriteFile(mc.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write MCP config: %w", err)
	}

	return nil
}

// AddServer adds a new MCP server configuration
func (mc *MCPConfig) AddServer(config *MCPServerConfig) error {
	if config.Name == "" {
		return fmt.Errorf("server name cannot be empty")
	}

	if _, exists := mc.Servers[config.Name]; exists {
		return fmt.Errorf("server %s already exists", config.Name)
	}

	// Set defaults
	if config.Timeout == 0 {
		config.Timeout = mc.GlobalSettings.DefaultTimeout
	}
	if config.Retries == 0 {
		config.Retries = mc.GlobalSettings.DefaultRetries
	}

	config.CreatedAt = time.Now()
	config.UpdatedAt = time.Now()

	mc.Servers[config.Name] = config
	return mc.Save()
}

// UpdateServer updates an existing MCP server configuration
func (mc *MCPConfig) UpdateServer(name string, config *MCPServerConfig) error {
	if _, exists := mc.Servers[name]; !exists {
		return fmt.Errorf("server %s does not exist", name)
	}

	config.Name = name
	config.UpdatedAt = time.Now()

	// Preserve creation time
	if existing := mc.Servers[name]; existing != nil {
		config.CreatedAt = existing.CreatedAt
	}

	mc.Servers[name] = config
	return mc.Save()
}

// RemoveServer removes an MCP server configuration
func (mc *MCPConfig) RemoveServer(name string) error {
	if _, exists := mc.Servers[name]; !exists {
		return fmt.Errorf("server %s does not exist", name)
	}

	delete(mc.Servers, name)
	return mc.Save()
}

// GetServer retrieves an MCP server configuration
func (mc *MCPConfig) GetServer(name string) (*MCPServerConfig, error) {
	server, exists := mc.Servers[name]
	if !exists {
		return nil, fmt.Errorf("server %s not found", name)
	}

	return server, nil
}

// GetAllServers returns all configured MCP servers
func (mc *MCPConfig) GetAllServers() []*MCPServerConfig {
	servers := make([]*MCPServerConfig, 0, len(mc.Servers))
	for _, server := range mc.Servers {
		servers = append(servers, server)
	}
	return servers
}

// ListServers returns all MCP server configurations
func (mc *MCPConfig) ListServers() map[string]*MCPServerConfig {
	return mc.Servers
}

// ListEnabledServers returns only enabled MCP server configurations
func (mc *MCPConfig) ListEnabledServers() map[string]*MCPServerConfig {
	enabled := make(map[string]*MCPServerConfig)
	for name, server := range mc.Servers {
		if server.Enabled {
			enabled[name] = server
		}
	}
	return enabled
}

// EnableServer enables an MCP server
func (mc *MCPConfig) EnableServer(name string) error {
	server, exists := mc.Servers[name]
	if !exists {
		return fmt.Errorf("server %s not found", name)
	}

	server.Enabled = true
	server.UpdatedAt = time.Now()
	return mc.Save()
}

// DisableServer disables an MCP server
func (mc *MCPConfig) DisableServer(name string) error {
	server, exists := mc.Servers[name]
	if !exists {
		return fmt.Errorf("server %s not found", name)
	}

	server.Enabled = false
	server.UpdatedAt = time.Now()
	return mc.Save()
}

// UpdateGlobalSettings updates global MCP settings
func (mc *MCPConfig) UpdateGlobalSettings(settings MCPGlobalSettings) error {
	mc.GlobalSettings = settings
	return mc.Save()
}

// Validate validates the MCP configuration
func (mc *MCPConfig) Validate() error {
	for name, server := range mc.Servers {
		if err := mc.validateServer(name, server); err != nil {
			return fmt.Errorf("server %s: %w", name, err)
		}
	}
	return nil
}

// validateServer validates a single server configuration
func (mc *MCPConfig) validateServer(name string, server *MCPServerConfig) error {
	if server.Name != name {
		return fmt.Errorf("server name mismatch: config name %s, server name %s", name, server.Name)
	}

	switch server.Type {
	case MCPServerTypeLocal:
		if len(server.Command) == 0 {
			return fmt.Errorf("local server must have command")
		}
	case MCPServerTypeRemote, MCPServerTypeSSE, MCPServerTypeHTTP:
		if server.URL == "" {
			return fmt.Errorf("remote server must have URL")
		}
	default:
		return fmt.Errorf("invalid server type: %s", server.Type)
	}

	return nil
}
