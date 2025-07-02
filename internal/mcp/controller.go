package mcp

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// MCPClientInterface defines the interface for MCP clients
type MCPClientInterface interface {
	Initialize(ctx context.Context, request mcp.InitializeRequest) (*mcp.InitializeResult, error)
	ListTools(ctx context.Context, request mcp.ListToolsRequest) (*mcp.ListToolsResult, error)
	ListResources(ctx context.Context, request mcp.ListResourcesRequest) (*mcp.ListResourcesResult, error)
	ListPrompts(ctx context.Context, request mcp.ListPromptsRequest) (*mcp.ListPromptsResult, error)
	CallTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error)
	ReadResource(ctx context.Context, request mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error)
	GetPrompt(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error)
	Close() error
}

// MCPController manages the lifecycle of MCP servers
type MCPController struct {
	config       *MCPConfig
	clients      map[string]*MCPClient
	clientsMux   sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	healthTicker *time.Ticker
}

// MCPClient represents a connected MCP client
type MCPClient struct {
	Name      string
	Config    *MCPServerConfig
	Client    MCPClientInterface // MCP client implementation
	Connected bool
	LastSeen  time.Time
	Tools     []mcp.Tool
	Resources []mcp.Resource
	Prompts   []mcp.Prompt
	mutex     sync.RWMutex
}

// NewMCPController creates a new MCP controller
func NewMCPController(config *MCPConfig) *MCPController {
	ctx, cancel := context.WithCancel(context.Background())

	controller := &MCPController{
		config:  config,
		clients: make(map[string]*MCPClient),
		ctx:     ctx,
		cancel:  cancel,
	}

	// Start health check routine
	controller.startHealthCheck()

	return controller
}

// Start initializes and starts all enabled MCP servers
func (mc *MCPController) Start() error {
	log.Printf("Starting MCP controller...")

	enabledServers := mc.config.ListEnabledServers()
	if len(enabledServers) == 0 {
		log.Printf("No enabled MCP servers found")
		return nil
	}

	var wg sync.WaitGroup
	errors := make(chan error, len(enabledServers))

	for name, serverConfig := range enabledServers {
		wg.Add(1)
		go func(name string, config *MCPServerConfig) {
			defer wg.Done()
			if err := mc.startServer(name, config); err != nil {
				errors <- fmt.Errorf("failed to start server %s: %w", name, err)
			}
		}(name, serverConfig)
	}

	wg.Wait()
	close(errors)

	// Collect any errors
	var startupErrors []error
	for err := range errors {
		startupErrors = append(startupErrors, err)
		log.Printf("MCP server startup error: %v", err)
	}

	if len(startupErrors) > 0 {
		log.Printf("Started MCP controller with %d errors", len(startupErrors))
	} else {
		log.Printf("MCP controller started successfully with %d servers", len(enabledServers))
	}

	return nil
}

// Stop gracefully shuts down all MCP servers
func (mc *MCPController) Stop() error {
	log.Printf("Stopping MCP controller...")

	// Cancel context to stop health checks
	mc.cancel()

	// Stop health check ticker
	if mc.healthTicker != nil {
		mc.healthTicker.Stop()
	}

	mc.clientsMux.Lock()
	defer mc.clientsMux.Unlock()

	var wg sync.WaitGroup
	for name, client := range mc.clients {
		wg.Add(1)
		go func(name string, client *MCPClient) {
			defer wg.Done()
			if err := mc.stopClient(name, client); err != nil {
				log.Printf("Error stopping MCP client %s: %v", name, err)
			}
		}(name, client)
	}

	wg.Wait()

	// Clear clients map
	mc.clients = make(map[string]*MCPClient)

	log.Printf("MCP controller stopped")
	return nil
}

// startServer starts a single MCP server
func (mc *MCPController) startServer(name string, config *MCPServerConfig) error {
	log.Printf("Starting MCP server: %s (%s)", name, config.Type)

	client := &MCPClient{
		Name:      name,
		Config:    config,
		Connected: false,
		LastSeen:  time.Now(),
		Tools:     []mcp.Tool{},
		Resources: []mcp.Resource{},
		Prompts:   []mcp.Prompt{},
	}

	// Connect based on server type
	switch config.Type {
	case MCPServerTypeLocal:
		return mc.connectLocalServer(client)
	case MCPServerTypeRemote, MCPServerTypeHTTP:
		return mc.connectRemoteServer(client)
	case MCPServerTypeSSE:
		return mc.connectSSEServer(client)
	default:
		return fmt.Errorf("unsupported server type: %s", config.Type)
	}
}

// connectLocalServer connects to a local MCP server
func (mc *MCPController) connectLocalServer(mcpClient *MCPClient) error {
	log.Printf("Connecting to local MCP server: %s", mcpClient.Name)

	// Convert environment map to slice
	var envSlice []string
	for k, v := range mcpClient.Config.Environment {
		envSlice = append(envSlice, k+"="+v)
	}

	// Create MCP client using the mark3labs/mcp-go library
	stdioClient, err := client.NewStdioMCPClient(
		mcpClient.Config.Command[0],
		envSlice,
		mcpClient.Config.Command[1:]...,
	)
	if err != nil {
		return fmt.Errorf("failed to create MCP client: %w", err)
	}

	// Initialize the client connection
	ctx, cancel := context.WithTimeout(mc.ctx, mcpClient.Config.Timeout)
	defer cancel()

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "CodeForge",
		Version: "1.0.0",
	}

	if _, err := stdioClient.Initialize(ctx, initRequest); err != nil {
		return fmt.Errorf("failed to initialize MCP client: %w", err)
	}

	// Get server capabilities
	toolsResult, err := stdioClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		log.Printf("Warning: failed to get tools from %s: %v", mcpClient.Name, err)
		mcpClient.Tools = []mcp.Tool{}
	} else {
		mcpClient.Tools = toolsResult.Tools
	}

	resourcesResult, err := stdioClient.ListResources(ctx, mcp.ListResourcesRequest{})
	if err != nil {
		log.Printf("Warning: failed to get resources from %s: %v", mcpClient.Name, err)
		mcpClient.Resources = []mcp.Resource{}
	} else {
		mcpClient.Resources = resourcesResult.Resources
	}

	promptsResult, err := stdioClient.ListPrompts(ctx, mcp.ListPromptsRequest{})
	if err != nil {
		log.Printf("Warning: failed to get prompts from %s: %v", mcpClient.Name, err)
		mcpClient.Prompts = []mcp.Prompt{}
	} else {
		mcpClient.Prompts = promptsResult.Prompts
	}

	mcpClient.Client = stdioClient
	mcpClient.Connected = true
	mcpClient.LastSeen = time.Now()

	mc.clientsMux.Lock()
	mc.clients[mcpClient.Name] = mcpClient
	mc.clientsMux.Unlock()

	log.Printf("Local MCP server %s connected with %d tools, %d resources, %d prompts",
		mcpClient.Name, len(mcpClient.Tools), len(mcpClient.Resources), len(mcpClient.Prompts))
	return nil
}

// connectRemoteServer connects to a remote MCP server
func (mc *MCPController) connectRemoteServer(mcpClient *MCPClient) error {
	log.Printf("Connecting to remote MCP server: %s at %s", mcpClient.Name, mcpClient.Config.URL)

	// Create HTTP MCP client using mark3labs/mcp-go library
	httpClient, err := client.NewStreamableHttpClient(mcpClient.Config.URL)
	if err != nil {
		return fmt.Errorf("failed to create HTTP MCP client: %w", err)
	}

	// Initialize the client connection
	ctx, cancel := context.WithTimeout(mc.ctx, mcpClient.Config.Timeout)
	defer cancel()

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "CodeForge",
		Version: "1.0.0",
	}

	if _, err := httpClient.Initialize(ctx, initRequest); err != nil {
		return fmt.Errorf("failed to initialize HTTP MCP client: %w", err)
	}

	// Get server capabilities
	toolsResult, err := httpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		log.Printf("Warning: failed to get tools from %s: %v", mcpClient.Name, err)
		mcpClient.Tools = []mcp.Tool{}
	} else {
		mcpClient.Tools = toolsResult.Tools
	}

	resourcesResult, err := httpClient.ListResources(ctx, mcp.ListResourcesRequest{})
	if err != nil {
		log.Printf("Warning: failed to get resources from %s: %v", mcpClient.Name, err)
		mcpClient.Resources = []mcp.Resource{}
	} else {
		mcpClient.Resources = resourcesResult.Resources
	}

	promptsResult, err := httpClient.ListPrompts(ctx, mcp.ListPromptsRequest{})
	if err != nil {
		log.Printf("Warning: failed to get prompts from %s: %v", mcpClient.Name, err)
		mcpClient.Prompts = []mcp.Prompt{}
	} else {
		mcpClient.Prompts = promptsResult.Prompts
	}

	mcpClient.Client = httpClient
	mcpClient.Connected = true
	mcpClient.LastSeen = time.Now()

	mc.clientsMux.Lock()
	mc.clients[mcpClient.Name] = mcpClient
	mc.clientsMux.Unlock()

	log.Printf("Remote MCP server %s connected with %d tools, %d resources, %d prompts",
		mcpClient.Name, len(mcpClient.Tools), len(mcpClient.Resources), len(mcpClient.Prompts))
	return nil
}

// connectSSEServer connects to an SSE MCP server
func (mc *MCPController) connectSSEServer(mcpClient *MCPClient) error {
	log.Printf("Connecting to SSE MCP server: %s at %s", mcpClient.Name, mcpClient.Config.URL)

	// Create SSE MCP client using mark3labs/mcp-go library
	sseClient, err := client.NewSSEMCPClient(mcpClient.Config.URL)
	if err != nil {
		return fmt.Errorf("failed to create SSE MCP client: %w", err)
	}

	// Initialize the client connection
	ctx, cancel := context.WithTimeout(mc.ctx, mcpClient.Config.Timeout)
	defer cancel()

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "CodeForge",
		Version: "1.0.0",
	}

	if _, err := sseClient.Initialize(ctx, initRequest); err != nil {
		return fmt.Errorf("failed to initialize SSE MCP client: %w", err)
	}

	// Get server capabilities
	toolsResult, err := sseClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		log.Printf("Warning: failed to get tools from %s: %v", mcpClient.Name, err)
		mcpClient.Tools = []mcp.Tool{}
	} else {
		mcpClient.Tools = toolsResult.Tools
	}

	resourcesResult, err := sseClient.ListResources(ctx, mcp.ListResourcesRequest{})
	if err != nil {
		log.Printf("Warning: failed to get resources from %s: %v", mcpClient.Name, err)
		mcpClient.Resources = []mcp.Resource{}
	} else {
		mcpClient.Resources = resourcesResult.Resources
	}

	promptsResult, err := sseClient.ListPrompts(ctx, mcp.ListPromptsRequest{})
	if err != nil {
		log.Printf("Warning: failed to get prompts from %s: %v", mcpClient.Name, err)
		mcpClient.Prompts = []mcp.Prompt{}
	} else {
		mcpClient.Prompts = promptsResult.Prompts
	}

	mcpClient.Client = sseClient
	mcpClient.Connected = true
	mcpClient.LastSeen = time.Now()

	mc.clientsMux.Lock()
	mc.clients[mcpClient.Name] = mcpClient
	mc.clientsMux.Unlock()

	log.Printf("SSE MCP server %s connected with %d tools, %d resources, %d prompts",
		mcpClient.Name, len(mcpClient.Tools), len(mcpClient.Resources), len(mcpClient.Prompts))
	return nil
}

// stopClient stops a single MCP client
func (mc *MCPController) stopClient(name string, client *MCPClient) error {
	log.Printf("Stopping MCP client: %s", name)

	client.mutex.Lock()
	defer client.mutex.Unlock()

	// Implement proper client shutdown based on type
	if client.Client != nil && client.Connected {
		// Close the underlying client connection
		if err := client.Client.Close(); err != nil {
			log.Printf("Error closing MCP client %s: %v", name, err)
		}
	}

	// Reset client state
	client.Connected = false
	client.Client = nil
	client.Tools = []mcp.Tool{}
	client.Resources = []mcp.Resource{}
	client.Prompts = []mcp.Prompt{}

	log.Printf("MCP client %s stopped", name)
	return nil
}

// GetClient returns an MCP client by name
func (mc *MCPController) GetClient(name string) (*MCPClient, error) {
	mc.clientsMux.RLock()
	defer mc.clientsMux.RUnlock()

	client, exists := mc.clients[name]
	if !exists {
		return nil, fmt.Errorf("MCP client %s not found", name)
	}

	return client, nil
}

// ListClients returns all connected MCP clients
func (mc *MCPController) ListClients() map[string]*MCPClient {
	mc.clientsMux.RLock()
	defer mc.clientsMux.RUnlock()

	clients := make(map[string]*MCPClient)
	for name, client := range mc.clients {
		clients[name] = client
	}

	return clients
}

// GetAllTools returns all tools from all connected MCP servers
func (mc *MCPController) GetAllTools() map[string][]mcp.Tool {
	mc.clientsMux.RLock()
	defer mc.clientsMux.RUnlock()

	tools := make(map[string][]mcp.Tool)
	for name, client := range mc.clients {
		if client.Connected {
			client.mutex.RLock()
			tools[name] = client.Tools
			client.mutex.RUnlock()
		}
	}

	return tools
}

// GetAllResources returns all resources from all connected MCP servers
func (mc *MCPController) GetAllResources() map[string][]mcp.Resource {
	mc.clientsMux.RLock()
	defer mc.clientsMux.RUnlock()

	resources := make(map[string][]mcp.Resource)
	for name, client := range mc.clients {
		if client.Connected {
			client.mutex.RLock()
			resources[name] = client.Resources
			client.mutex.RUnlock()
		}
	}

	return resources
}

// startHealthCheck starts the health check routine
func (mc *MCPController) startHealthCheck() {
	if mc.config.GlobalSettings.HealthCheckInterval <= 0 {
		return
	}

	mc.healthTicker = time.NewTicker(mc.config.GlobalSettings.HealthCheckInterval)

	go func() {
		for {
			select {
			case <-mc.ctx.Done():
				return
			case <-mc.healthTicker.C:
				mc.performHealthCheck()
			}
		}
	}()
}

// performHealthCheck checks the health of all connected clients
func (mc *MCPController) performHealthCheck() {
	mc.clientsMux.RLock()
	clients := make([]*MCPClient, 0, len(mc.clients))
	for _, client := range mc.clients {
		clients = append(clients, client)
	}
	mc.clientsMux.RUnlock()

	for _, client := range clients {
		go mc.checkClientHealth(client)
	}
}

// checkClientHealth checks the health of a single client
func (mc *MCPController) checkClientHealth(client *MCPClient) {
	client.mutex.Lock()
	defer client.mutex.Unlock()

	// Implement actual health check based on client type
	if !client.Connected || client.Client == nil {
		return
	}

	// Create a timeout context for the health check
	ctx, cancel := context.WithTimeout(mc.ctx, 5*time.Second)
	defer cancel()

	// Perform health check by attempting to list tools
	// This is a lightweight operation that verifies the connection is still active
	_, err := client.Client.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		log.Printf("Health check failed for MCP client %s: %v", client.Name, err)

		// Mark client as disconnected and attempt reconnection
		client.Connected = false

		// Schedule reconnection attempt
		go func() {
			time.Sleep(time.Second * 10) // Wait before reconnecting
			if err := mc.reconnectClient(client); err != nil {
				log.Printf("Failed to reconnect MCP client %s: %v", client.Name, err)
			}
		}()
	} else {
		// Health check passed, update last seen time
		client.LastSeen = time.Now()
	}
}

// reconnectClient attempts to reconnect a disconnected client
func (mc *MCPController) reconnectClient(client *MCPClient) error {
	log.Printf("Attempting to reconnect MCP client: %s", client.Name)

	// Stop the existing client cleanly
	if err := mc.stopClient(client.Name, client); err != nil {
		log.Printf("Error stopping client during reconnection: %v", err)
	}

	// Remove from clients map temporarily
	mc.clientsMux.Lock()
	delete(mc.clients, client.Name)
	mc.clientsMux.Unlock()

	// Attempt to restart the server
	if err := mc.startServer(client.Name, client.Config); err != nil {
		return fmt.Errorf("failed to restart server: %w", err)
	}

	log.Printf("Successfully reconnected MCP client: %s", client.Name)
	return nil
}
