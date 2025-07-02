package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/config"
	"go.lsp.dev/protocol"
)

// ServerState represents the state of an LSP server
type ServerState int

const (
	StateStarting ServerState = iota
	StateReady
	StateError
	StateStopped
)

// OpenFileInfo tracks information about files opened in the LSP server
type OpenFileInfo struct {
	URI        string
	LanguageID string
	Version    int32
	Content    string
}

// NotificationHandler handles server notifications
type NotificationHandler func(params json.RawMessage)

// ServerRequestHandler handles server requests
type ServerRequestHandler func(params json.RawMessage) (interface{}, error)

// Client represents an LSP client connection
type Client struct {
	Cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	stderr io.ReadCloser

	// Request ID counter
	nextID atomic.Int32

	// Response handlers
	handlers   map[int32]chan *Message
	handlersMu sync.RWMutex

	// Server request handlers
	serverRequestHandlers map[string]ServerRequestHandler
	serverHandlersMu      sync.RWMutex

	// Notification handlers
	notificationHandlers map[string]NotificationHandler
	notificationMu       sync.RWMutex

	// Diagnostic cache
	diagnostics   map[string][]protocol.Diagnostic
	diagnosticsMu sync.RWMutex

	// Files currently opened by the LSP
	openFiles   map[string]*OpenFileInfo
	openFilesMu sync.RWMutex

	// Server state
	serverState atomic.Value
}

// Message represents a JSON-RPC 2.0 message
type Message struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int32           `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *ResponseError  `json:"error,omitempty"`
}

// ResponseError represents a JSON-RPC 2.0 error
type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Manager manages multiple LSP clients
type Manager struct {
	clients   map[string]*Client
	clientsMu sync.RWMutex
	config    *config.Config
}

// Global manager instance
var manager *Manager

// Initialize sets up the LSP manager with configuration
func Initialize(cfg *config.Config) error {
	manager = &Manager{
		clients: make(map[string]*Client),
		config:  cfg,
	}

	// Initialize LSP clients based on configuration
	ctx := context.Background()
	for name, lspConfig := range cfg.LSP {
		if len(lspConfig.Command) > 0 {
			command := lspConfig.Command[0]
			args := append(lspConfig.Command[1:], lspConfig.Args...)
			go manager.createAndStartLSPClient(ctx, name, command, args...)
		}
	}

	return nil
}

// GetManager returns the global LSP manager instance
func GetManager() *Manager {
	return manager
}

// GetClientForLanguage returns an LSP client for the specified language
func (m *Manager) GetClientForLanguage(language string) *Client {
	if language == "" {
		// Return any available client if no language specified
		m.clientsMu.RLock()
		defer m.clientsMu.RUnlock()
		for _, client := range m.clients {
			if client.GetState() == StateReady {
				return client
			}
		}
		return nil
	}

	m.clientsMu.RLock()
	defer m.clientsMu.RUnlock()

	// Try exact language match first
	if client, exists := m.clients[language]; exists && client.GetState() == StateReady {
		return client
	}

	// Try language aliases (e.g., "golang" -> "go")
	aliases := map[string]string{
		"golang":     "go",
		"javascript": "js",
		"typescript": "ts",
		"python3":    "python",
		"py":         "python",
	}

	if alias, exists := aliases[language]; exists {
		if client, exists := m.clients[alias]; exists && client.GetState() == StateReady {
			return client
		}
	}

	return nil
}

// GetAllClients returns all active LSP clients
func (m *Manager) GetAllClients() map[string]*Client {
	m.clientsMu.RLock()
	defer m.clientsMu.RUnlock()

	// Return a copy to avoid concurrent access issues
	clients := make(map[string]*Client)
	for lang, client := range m.clients {
		if client.GetState() == StateReady {
			clients[lang] = client
		}
	}

	return clients
}

// GetActiveClients returns a slice of all active LSP clients
func (m *Manager) GetActiveClients() []*Client {
	m.clientsMu.RLock()
	defer m.clientsMu.RUnlock()

	var clients []*Client
	for _, client := range m.clients {
		if client.GetState() == StateReady {
			clients = append(clients, client)
		}
	}

	return clients
}

// GetClientForFile returns an LSP client for the specified file based on its extension
func (m *Manager) GetClientForFile(filepath string) *Client {
	// Detect language from file extension
	language := detectLanguageFromFile(filepath)
	return m.GetClientForLanguage(language)
}

// detectLanguageFromFile detects programming language from file extension
func detectLanguageFromFile(filepath string) string {
	ext := strings.ToLower(filepath[strings.LastIndex(filepath, ".")+1:])

	languageMap := map[string]string{
		"go":   "go",
		"rs":   "rust",
		"py":   "python",
		"js":   "javascript",
		"ts":   "typescript",
		"java": "java",
		"cpp":  "cpp",
		"cc":   "cpp",
		"cxx":  "cpp",
		"c":    "c",
		"h":    "c",
		"hpp":  "cpp",
		"lua":  "lua",
		"ex":   "elixir",
		"exs":  "elixir",
	}

	if language, exists := languageMap[ext]; exists {
		return language
	}

	return ""
}

// GetState returns the current state of the client
func (c *Client) GetState() ServerState {
	if state := c.serverState.Load(); state != nil {
		return state.(ServerState)
	}
	return StateStopped
}

// IsFileOpen checks if a file is currently open in the LSP server
func (c *Client) IsFileOpen(filepath string) bool {
	uri := fmt.Sprintf("file://%s", filepath)

	c.openFilesMu.RLock()
	defer c.openFilesMu.RUnlock()

	_, exists := c.openFiles[uri]
	return exists
}

// NewClient creates a new LSP client
func NewClient(ctx context.Context, command string, args ...string) (*Client, error) {
	cmd := exec.CommandContext(ctx, command, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	client := &Client{
		Cmd:                   cmd,
		stdin:                 stdin,
		stdout:                bufio.NewReader(stdout),
		stderr:                stderr,
		handlers:              make(map[int32]chan *Message),
		notificationHandlers:  make(map[string]NotificationHandler),
		serverRequestHandlers: make(map[string]ServerRequestHandler),
		diagnostics:           make(map[string][]protocol.Diagnostic),
		openFiles:             make(map[string]*OpenFileInfo),
	}

	// Initialize server state
	client.serverState.Store(StateStarting)

	// Start the LSP server process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start LSP server: %w", err)
	}

	// Handle stderr in a separate goroutine
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Fprintf(os.Stderr, "LSP Server: %s\n", scanner.Text())
		}
	}()

	// Start message handling loop
	go client.handleMessages()

	return client, nil
}

// GetClient returns an LSP client by name
func GetClient(name string) (*Client, bool) {
	if manager == nil {
		return nil, false
	}

	manager.clientsMu.RLock()
	defer manager.clientsMu.RUnlock()
	client, exists := manager.clients[name]
	return client, exists
}

// GetAvailableClients returns all available LSP clients
func GetAvailableClients() map[string]*Client {
	if manager == nil {
		return nil
	}

	manager.clientsMu.RLock()
	defer manager.clientsMu.RUnlock()

	clients := make(map[string]*Client)
	for name, client := range manager.clients {
		clients[name] = client
	}
	return clients
}

// createAndStartLSPClient creates and initializes an LSP client
func (m *Manager) createAndStartLSPClient(ctx context.Context, name string, command string, args ...string) {
	// Create the LSP client
	lspClient, err := NewClient(ctx, command, args...)
	if err != nil {
		fmt.Printf("Failed to create LSP client for %s: %v\n", name, err)
		return
	}

	// Initialize with timeout
	initCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Initialize the LSP client
	_, err = lspClient.InitializeLSPClient(initCtx, m.config.WorkingDir)
	if err != nil {
		fmt.Printf("Initialize failed for %s: %v\n", name, err)
		lspClient.Close()
		return
	}

	// Wait for the server to be ready
	if err := lspClient.WaitForServerReady(initCtx); err != nil {
		fmt.Printf("Server failed to become ready for %s: %v\n", name, err)
		lspClient.SetServerState(StateError)
	} else {
		fmt.Printf("LSP server is ready: %s\n", name)
		lspClient.SetServerState(StateReady)
	}

	// Add to clients map
	m.clientsMu.Lock()
	m.clients[name] = lspClient
	m.clientsMu.Unlock()
}

// handleMessages processes incoming messages from the LSP server
func (c *Client) handleMessages() {
	for {
		msg, err := c.readMessage()
		if err != nil {
			if err == io.EOF {
				break
			}
			fmt.Printf("Error reading message: %v\n", err)
			continue
		}

		if msg.ID != 0 {
			// This is a response to a request
			c.handlersMu.RLock()
			ch, exists := c.handlers[msg.ID]
			c.handlersMu.RUnlock()

			if exists {
				select {
				case ch <- msg:
				default:
					// Channel is full or closed
				}
			}
		} else if msg.Method != "" {
			// This is a notification or server request
			if strings.HasPrefix(msg.Method, "$/") {
				// Server notification
				c.handleNotification(msg.Method, msg.Params)
			} else {
				// Server request
				c.handleServerRequest(msg.Method, msg.Params)
			}
		}
	}
}

// SetServerState sets the server state
func (c *Client) SetServerState(state ServerState) {
	c.serverState.Store(state)
}

// GetServerState returns the current server state
func (c *Client) GetServerState() ServerState {
	return c.serverState.Load().(ServerState)
}

// WaitForServerReady waits for the server to become ready
func (c *Client) WaitForServerReady(ctx context.Context) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			state := c.GetServerState()
			if state == StateReady {
				return nil
			}
			if state == StateError || state == StateStopped {
				return fmt.Errorf("server is in error or stopped state")
			}
		}
	}
}

// Close closes the LSP client
func (c *Client) Close() error {
	// Close stdin to signal the server
	if err := c.stdin.Close(); err != nil {
		return fmt.Errorf("failed to close stdin: %w", err)
	}

	// Wait for the process to exit with timeout
	done := make(chan error, 1)
	go func() {
		done <- c.Cmd.Wait()
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		// Force kill if it doesn't exit gracefully
		if err := c.Cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
		return <-done
	}
}
