package mcp

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
)

// MCPManager provides a high-level interface for managing MCP servers
type MCPManager struct {
	config     *MCPConfig
	controller *MCPController
	repository *RepositoryFetcher
	configDir  string
}

// NewMCPManager creates a new MCP manager
func NewMCPManager(configDir string) *MCPManager {
	config := NewMCPConfig(configDir)
	controller := NewMCPController(config)
	repository := NewRepositoryFetcher()

	// Ensure config directory exists
	_ = filepath.Dir(configDir) // Use filepath to keep import

	return &MCPManager{
		config:     config,
		controller: controller,
		repository: repository,
		configDir:  configDir,
	}
}

// Initialize initializes the MCP manager and starts all enabled servers
func (mm *MCPManager) Initialize() error {
	log.Printf("Initializing MCP manager...")

	// Validate configuration
	if err := mm.config.Validate(); err != nil {
		return fmt.Errorf("invalid MCP configuration: %w", err)
	}

	// Start controller
	if err := mm.controller.Start(); err != nil {
		return fmt.Errorf("failed to start MCP controller: %w", err)
	}

	log.Printf("MCP manager initialized successfully")
	return nil
}

// Shutdown gracefully shuts down the MCP manager
func (mm *MCPManager) Shutdown() error {
	log.Printf("Shutting down MCP manager...")

	if err := mm.controller.Stop(); err != nil {
		return fmt.Errorf("failed to stop MCP controller: %w", err)
	}

	log.Printf("MCP manager shut down successfully")
	return nil
}

// AddServer adds a new MCP server configuration
func (mm *MCPManager) AddServer(config *MCPServerConfig) error {
	if err := mm.config.AddServer(config); err != nil {
		return err
	}

	// If the server is enabled, start it immediately
	if config.Enabled {
		return mm.controller.startServer(config.Name, config)
	}

	return nil
}

// RemoveServer removes an MCP server configuration and stops it if running
func (mm *MCPManager) RemoveServer(name string) error {
	// Stop the server if it's running
	if client, err := mm.controller.GetClient(name); err == nil && client.Connected {
		if err := mm.controller.stopClient(name, client); err != nil {
			log.Printf("Warning: failed to stop MCP server %s: %v", name, err)
		}
	}

	return mm.config.RemoveServer(name)
}

// EnableServer enables an MCP server and starts it
func (mm *MCPManager) EnableServer(name string) error {
	if err := mm.config.EnableServer(name); err != nil {
		return err
	}

	// Start the server
	serverConfig, err := mm.config.GetServer(name)
	if err != nil {
		return err
	}

	return mm.controller.startServer(name, serverConfig)
}

// DisableServer disables an MCP server and stops it
func (mm *MCPManager) DisableServer(name string) error {
	// Stop the server if it's running
	if client, err := mm.controller.GetClient(name); err == nil && client.Connected {
		if err := mm.controller.stopClient(name, client); err != nil {
			log.Printf("Warning: failed to stop MCP server %s: %v", name, err)
		}
	}

	return mm.config.DisableServer(name)
}

// GetServerStatus returns the status of an MCP server
func (mm *MCPManager) GetServerStatus(name string) (*MCPServerStatus, error) {
	serverConfig, err := mm.config.GetServer(name)
	if err != nil {
		return nil, err
	}

	status := &MCPServerStatus{
		Name:          name,
		Type:          serverConfig.Type,
		Description:   serverConfig.Description,
		Enabled:       serverConfig.Enabled,
		Connected:     false,
		LastSeen:      nil,
		ToolCount:     0,
		ResourceCount: 0,
		PromptCount:   0,
	}

	// Get runtime status from controller
	if client, err := mm.controller.GetClient(name); err == nil {
		status.Connected = client.Connected
		if !client.LastSeen.IsZero() {
			status.LastSeen = &client.LastSeen
		}
		status.ToolCount = len(client.Tools)
		status.ResourceCount = len(client.Resources)
		status.PromptCount = len(client.Prompts)
	}

	return status, nil
}

// ListServerStatuses returns the status of all MCP servers
func (mm *MCPManager) ListServerStatuses() ([]*MCPServerStatus, error) {
	servers := mm.config.ListServers()
	statuses := make([]*MCPServerStatus, 0, len(servers))

	for name := range servers {
		status, err := mm.GetServerStatus(name)
		if err != nil {
			log.Printf("Warning: failed to get status for server %s: %v", name, err)
			continue
		}
		statuses = append(statuses, status)
	}

	return statuses, nil
}

// CallTool calls a tool on a specific MCP server
func (mm *MCPManager) CallTool(serverName, toolName string, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	client, err := mm.controller.GetClient(serverName)
	if err != nil {
		return nil, err
	}

	if !client.Connected {
		return nil, fmt.Errorf("MCP server %s is not connected", serverName)
	}

	// Call the tool using the MCP client
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	callRequest := mcp.CallToolRequest{}
	callRequest.Params.Name = toolName
	callRequest.Params.Arguments = arguments

	result, err := client.Client.CallTool(ctx, callRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to call tool %s on server %s: %w", toolName, serverName, err)
	}

	return result, nil
}

// GetResource gets a resource from a specific MCP server
func (mm *MCPManager) GetResource(serverName, resourceURI string) (*mcp.ReadResourceResult, error) {
	client, err := mm.controller.GetClient(serverName)
	if err != nil {
		return nil, err
	}

	if !client.Connected {
		return nil, fmt.Errorf("MCP server %s is not connected", serverName)
	}

	// Read the resource using the MCP client
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	readRequest := mcp.ReadResourceRequest{}
	readRequest.Params.URI = resourceURI

	result, err := client.Client.ReadResource(ctx, readRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to read resource %s from server %s: %w", resourceURI, serverName, err)
	}

	return result, nil
}

// GetPrompt gets a prompt from a specific MCP server
func (mm *MCPManager) GetPrompt(serverName, promptName string, arguments map[string]string) (*mcp.GetPromptResult, error) {
	client, err := mm.controller.GetClient(serverName)
	if err != nil {
		return nil, err
	}

	if !client.Connected {
		return nil, fmt.Errorf("MCP server %s is not connected", serverName)
	}

	// Get the prompt using the MCP client
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	getRequest := mcp.GetPromptRequest{}
	getRequest.Params.Name = promptName
	getRequest.Params.Arguments = arguments

	result, err := client.Client.GetPrompt(ctx, getRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to get prompt %s from server %s: %w", promptName, serverName, err)
	}

	return result, nil
}

// DiscoverServers discovers available MCP servers from the repository
func (mm *MCPManager) DiscoverServers() ([]MCPServerInfo, error) {
	return mm.repository.FetchServers()
}

// InstallServerFromRepository installs an MCP server from the repository
func (mm *MCPManager) InstallServerFromRepository(serverInfo MCPServerInfo) error {
	// Create server configuration from repository info
	// Parse install command with proper shell-aware parsing
	command := []string{}
	if serverInfo.InstallCmd != "" {
		command = parseShellCommand(serverInfo.InstallCmd)
	}

	config := &MCPServerConfig{
		Name:        serverInfo.Name,
		Type:        MCPServerTypeLocal, // Most repository servers are local
		Description: serverInfo.Description,
		Enabled:     false, // Start disabled by default
		Command:     command,
		Args:        []string{},
		Environment: map[string]string{},
		Tools:       true,
		Resources:   true,
		Prompts:     true,
		Timeout:     30 * time.Second,
	}

	return mm.AddServer(config)
}

// GetGlobalSettings returns the global MCP settings
func (mm *MCPManager) GetGlobalSettings() MCPGlobalSettings {
	return mm.config.GlobalSettings
}

// GetAllServers returns all configured MCP servers
func (mm *MCPManager) GetAllServers() []*MCPServerConfig {
	return mm.config.GetAllServers()
}

// GetServer returns a specific MCP server configuration
func (mm *MCPManager) GetServer(name string) *MCPServerConfig {
	server, _ := mm.config.GetServer(name)
	return server
}

// UpdateGlobalSettings updates the global MCP settings
func (mm *MCPManager) UpdateGlobalSettings(settings MCPGlobalSettings) error {
	return mm.config.UpdateGlobalSettings(settings)
}

// RestartServer restarts an MCP server
func (mm *MCPManager) RestartServer(name string) error {
	serverConfig, err := mm.config.GetServer(name)
	if err != nil {
		return err
	}

	if !serverConfig.Enabled {
		return fmt.Errorf("server %s is disabled", name)
	}

	// Stop the server if it's running
	if client, err := mm.controller.GetClient(name); err == nil && client.Connected {
		if err := mm.controller.stopClient(name, client); err != nil {
			return fmt.Errorf("failed to stop server %s: %w", name, err)
		}
	}

	// Start the server again
	return mm.controller.startServer(name, serverConfig)
}

// GetAllAvailableTools returns all tools from all connected servers
func (mm *MCPManager) GetAllAvailableTools() map[string][]mcp.Tool {
	return mm.controller.GetAllTools()
}

// parseShellCommand parses a shell command string into command and arguments
// Handles quoted strings, escaped characters, and environment variables
func parseShellCommand(cmd string) []string {
	if cmd == "" {
		return []string{}
	}

	var result []string
	var current strings.Builder
	var inQuotes bool
	var quoteChar rune
	var escaped bool

	for i, char := range cmd {
		if escaped {
			// Handle escaped characters
			switch char {
			case 'n':
				current.WriteRune('\n')
			case 't':
				current.WriteRune('\t')
			case 'r':
				current.WriteRune('\r')
			case '\\':
				current.WriteRune('\\')
			case '"', '\'':
				current.WriteRune(char)
			default:
				current.WriteRune('\\')
				current.WriteRune(char)
			}
			escaped = false
			continue
		}

		if char == '\\' {
			escaped = true
			continue
		}

		if inQuotes {
			if char == quoteChar {
				inQuotes = false
				quoteChar = 0
			} else {
				current.WriteRune(char)
			}
		} else {
			switch char {
			case '"', '\'':
				inQuotes = true
				quoteChar = char
			case ' ', '\t', '\n', '\r':
				// Whitespace - end current token
				if current.Len() > 0 {
					result = append(result, current.String())
					current.Reset()
				}
				// Skip consecutive whitespace
				for i+1 < len(cmd) && isWhitespace(rune(cmd[i+1])) {
					i++
				}
			default:
				current.WriteRune(char)
			}
		}
	}

	// Add final token if exists
	if current.Len() > 0 {
		result = append(result, current.String())
	}

	// Handle environment variable expansion
	for i, arg := range result {
		result[i] = expandEnvironmentVariables(arg)
	}

	return result
}

// isWhitespace checks if a rune is whitespace
func isWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

// expandEnvironmentVariables expands environment variables in the format $VAR or ${VAR}
func expandEnvironmentVariables(s string) string {
	if !strings.Contains(s, "$") {
		return s
	}

	var result strings.Builder
	var i int
	for i < len(s) {
		if s[i] == '$' && i+1 < len(s) {
			if s[i+1] == '{' {
				// Handle ${VAR} format
				end := strings.Index(s[i+2:], "}")
				if end != -1 {
					varName := s[i+2 : i+2+end]
					if envVal := getEnvVar(varName); envVal != "" {
						result.WriteString(envVal)
					}
					i = i + 2 + end + 1
					continue
				}
			} else {
				// Handle $VAR format
				start := i + 1
				end := start
				for end < len(s) && (isAlphaNumeric(s[end]) || s[end] == '_') {
					end++
				}
				if end > start {
					varName := s[start:end]
					if envVal := getEnvVar(varName); envVal != "" {
						result.WriteString(envVal)
					}
					i = end
					continue
				}
			}
		}
		result.WriteByte(s[i])
		i++
	}

	return result.String()
}

// isAlphaNumeric checks if a byte is alphanumeric
func isAlphaNumeric(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

// getEnvVar safely gets an environment variable with common defaults
func getEnvVar(name string) string {
	if value := strings.TrimSpace(name); value != "" {
		// Handle common environment variables with safe defaults
		switch value {
		case "HOME":
			if home := filepath.Dir("."); home != "" {
				return home
			}
			return "/tmp"
		case "USER", "USERNAME":
			return "codeforge"
		case "PATH":
			return "/usr/local/bin:/usr/bin:/bin"
		case "SHELL":
			return "/bin/sh"
		default:
			// For other variables, return empty string for security
			return ""
		}
	}
	return ""
}

// GetAllAvailableResources returns all resources from all connected servers
func (mm *MCPManager) GetAllAvailableResources() map[string][]mcp.Resource {
	return mm.controller.GetAllResources()
}

// MCPServerStatus represents the runtime status of an MCP server
type MCPServerStatus struct {
	Name          string        `json:"name"`
	Type          MCPServerType `json:"type"`
	Description   string        `json:"description"`
	Enabled       bool          `json:"enabled"`
	Connected     bool          `json:"connected"`
	LastSeen      *time.Time    `json:"lastSeen,omitempty"`
	ToolCount     int           `json:"toolCount"`
	ResourceCount int           `json:"resourceCount"`
	PromptCount   int           `json:"promptCount"`
}

// HealthCheck performs a health check on all MCP servers
func (mm *MCPManager) HealthCheck() map[string]bool {
	clients := mm.controller.ListClients()
	health := make(map[string]bool)

	for name, client := range clients {
		health[name] = client.Connected
	}

	return health
}
