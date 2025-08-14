package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/app"
	"github.com/entrepeneur4lyf/codeforge/internal/builder"
	"github.com/entrepeneur4lyf/codeforge/internal/config"
	"github.com/entrepeneur4lyf/codeforge/internal/embeddings"
	"github.com/entrepeneur4lyf/codeforge/internal/llm"
	"github.com/entrepeneur4lyf/codeforge/internal/lsp"
	"github.com/entrepeneur4lyf/codeforge/internal/mcp"
	"github.com/entrepeneur4lyf/codeforge/internal/permissions"
	"github.com/entrepeneur4lyf/codeforge/internal/vectordb"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// Server represents the web server
type Server struct {
	router   *mux.Router
	upgrader websocket.Upgrader
	config   *config.Config
	clients  map[*websocket.Conn]bool
	app      *app.App // Integrated CodeForge application
}

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// SearchRequest represents a search request
type SearchRequest struct {
	Query    string `json:"query"`
	Language string `json:"language,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

// LSPRequest represents an LSP operation request
type LSPRequest struct {
	Operation string                 `json:"operation"`
	FilePath  string                 `json:"filePath"`
	Line      int                    `json:"line,omitempty"`
	Character int                    `json:"character,omitempty"`
	Query     string                 `json:"query,omitempty"`
	Args      map[string]interface{} `json:"args,omitempty"`
}

// MCPRequest represents an MCP operation request
type MCPRequest struct {
	Operation string                 `json:"operation"`
	ToolName  string                 `json:"toolName,omitempty"`
	Server    string                 `json:"server,omitempty"`
	URI       string                 `json:"uri,omitempty"`
	Args      map[string]interface{} `json:"args,omitempty"`
}

// ChatRequest represents an AI chat request
type ChatRequest struct {
	Message string `json:"message"`
	Model   string `json:"model,omitempty"`
}

// FileRequest represents a file operation request
type FileRequest struct {
	Operation string `json:"operation"`
	Path      string `json:"path"`
	Content   string `json:"content,omitempty"`
}

// BuildRequest represents a build operation request
type BuildRequest struct {
	Language string `json:"language"`
	Command  string `json:"command,omitempty"`
}

// SettingsRequest represents a settings update request
type SettingsRequest struct {
	Category string                 `json:"category"`
	Settings map[string]interface{} `json:"settings"`
}

// CommandRequest represents a command palette request
type CommandRequest struct {
	Command string                 `json:"command"`
	Args    map[string]interface{} `json:"args,omitempty"`
}

// NewServer creates a new web server
func NewServer(cfg *config.Config) *Server {
	router := mux.NewRouter()

    upgrader := websocket.Upgrader{
        CheckOrigin: func(r *http.Request) bool {
            // Only allow same-host or localhost origins by default
            origin := r.Header.Get("Origin")
            if origin == "" {
                return true
            }
            // Simple allowlist for localhost development
            // Avoid full URL parsing to keep dependencies minimal
            allowed := []string{
                "http://localhost:",
                "http://127.0.0.1:",
                "http://[::1]:",
            }
            for _, a := range allowed {
                if strings.HasPrefix(origin, a) {
                    return true
                }
            }
            return false
        },
    }

	server := &Server{
		router:   router,
		upgrader: upgrader,
		config:   cfg,
		clients:  make(map[*websocket.Conn]bool),
	}

	server.setupRoutes()
	return server
}

// NewServerWithApp creates a new web server with integrated CodeForge app
func NewServerWithApp(cfg *config.Config, codeforgeApp *app.App) *Server {
	router := mux.NewRouter()

    upgrader := websocket.Upgrader{
        CheckOrigin: func(r *http.Request) bool {
            origin := r.Header.Get("Origin")
            if origin == "" {
                return true
            }
            allowed := []string{
                "http://localhost:",
                "http://127.0.0.1:",
                "http://[::1]:",
            }
            for _, a := range allowed {
                if strings.HasPrefix(origin, a) {
                    return true
                }
            }
            return false
        },
    }

	server := &Server{
		router:   router,
		upgrader: upgrader,
		config:   cfg,
		clients:  make(map[*websocket.Conn]bool),
		app:      codeforgeApp,
	}

	server.setupRoutes()
	return server
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	// Static files
	s.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("web/static/"))))

	// API routes
	api := s.router.PathPrefix("/api").Subrouter()
	api.HandleFunc("/chat", s.handleChat).Methods("POST")
	api.HandleFunc("/files", s.handleFiles).Methods("GET", "POST")
	api.HandleFunc("/build", s.handleBuild).Methods("POST")
	api.HandleFunc("/settings", s.handleSettings).Methods("GET", "POST")
	api.HandleFunc("/commands", s.handleCommands).Methods("POST")

	// MCP management routes
	api.HandleFunc("/mcp/servers", s.handleMCPServers).Methods("GET", "POST")
	api.HandleFunc("/mcp/servers/{name}", s.handleMCPServer).Methods("GET", "PUT", "DELETE")
	api.HandleFunc("/mcp/servers/{name}/enable", s.handleMCPServerEnable).Methods("POST")
	api.HandleFunc("/mcp/servers/{name}/disable", s.handleMCPServerDisable).Methods("POST")
	api.HandleFunc("/mcp/discover", s.handleMCPDiscover).Methods("GET")
	api.HandleFunc("/providers", s.handleProviders).Methods("GET")
	api.HandleFunc("/lsp", s.handleLSP).Methods("POST")
	api.HandleFunc("/mcp", s.handleMCP).Methods("POST")
	api.HandleFunc("/mcp/tools", s.handleMCPTools).Methods("GET")
	api.HandleFunc("/mcp/resources", s.handleMCPResources).Methods("GET")
	api.HandleFunc("/status", s.handleStatus).Methods("GET")

	// WebSocket endpoint
	s.router.HandleFunc("/ws", s.handleWebSocket)

	// Main page
	s.router.HandleFunc("/", s.handleIndex).Methods("GET")
}

// handleIndex serves the main web interface (TUI-style)
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>CodeForge - AI-Powered Code Assistant</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: 'JetBrains Mono', 'Fira Code', 'Consolas', monospace;
            background: #0d1117;
            color: #c9d1d9;
            height: 100vh;
            overflow: hidden;
        }

        /* TUI-style layout */
        .main-container {
            display: grid;
            grid-template-columns: 250px 1fr 300px;
            grid-template-rows: 40px 1fr 200px;
            height: 100vh;
            gap: 1px;
            background: #21262d;
        }

        /* Header bar */
        .header-bar {
            grid-column: 1 / -1;
            background: #161b22;
            display: flex;
            align-items: center;
            padding: 0 16px;
            border-bottom: 1px solid #30363d;
        }

        .header-bar h1 {
            color: #58a6ff;
            font-size: 16px;
            font-weight: 600;
        }

        .header-status {
            margin-left: auto;
            display: flex;
            gap: 12px;
            font-size: 12px;
        }

        .status-indicator {
            display: flex;
            align-items: center;
            gap: 4px;
        }

        .status-dot {
            width: 8px;
            height: 8px;
            border-radius: 50%;
            background: #f85149;
        }

        .status-dot.active { background: #3fb950; }
        .status-dot.warning { background: #d29922; }

        /* File browser pane */
        .file-browser {
            background: #0d1117;
            border-right: 1px solid #30363d;
            overflow-y: auto;
        }

        .pane-header {
            background: #161b22;
            padding: 8px 12px;
            border-bottom: 1px solid #30363d;
            font-size: 12px;
            font-weight: 600;
            color: #7d8590;
        }

        .file-tree {
            padding: 8px;
        }

        .file-item {
            padding: 4px 8px;
            cursor: pointer;
            border-radius: 4px;
            font-size: 13px;
            display: flex;
            align-items: center;
            gap: 6px;
        }

        .file-item:hover {
            background: #21262d;
        }

        .file-item.selected {
            background: #1f6feb;
            color: white;
        }

        .file-icon {
            width: 16px;
            text-align: center;
        }

        /* Code editor pane */
        .code-editor {
            background: #0d1117;
            display: flex;
            flex-direction: column;
        }

        .editor-tabs {
            background: #161b22;
            border-bottom: 1px solid #30363d;
            display: flex;
            overflow-x: auto;
        }

        .editor-tab {
            padding: 8px 16px;
            border-right: 1px solid #30363d;
            cursor: pointer;
            font-size: 13px;
            white-space: nowrap;
            display: flex;
            align-items: center;
            gap: 6px;
        }

        .editor-tab.active {
            background: #0d1117;
            color: #58a6ff;
        }

        .editor-tab:hover:not(.active) {
            background: #21262d;
        }

        .tab-close {
            margin-left: 4px;
            opacity: 0.6;
            cursor: pointer;
        }

        .tab-close:hover {
            opacity: 1;
            color: #f85149;
        }

        .editor-content {
            flex: 1;
            padding: 16px;
            overflow: auto;
            font-family: 'JetBrains Mono', monospace;
            font-size: 14px;
            line-height: 1.5;
        }

        .code-textarea {
            width: 100%;
            height: 100%;
            background: transparent;
            border: none;
            color: #c9d1d9;
            font-family: inherit;
            font-size: inherit;
            line-height: inherit;
            resize: none;
            outline: none;
        }

        /* AI Chat pane */
        .ai-chat {
            background: #0d1117;
            border-left: 1px solid #30363d;
            display: flex;
            flex-direction: column;
        }

        .chat-messages {
            flex: 1;
            overflow-y: auto;
            padding: 12px;
        }

        .message {
            margin-bottom: 16px;
            padding: 8px 12px;
            border-radius: 6px;
            font-size: 13px;
            line-height: 1.4;
        }

        .message.user {
            background: #1f6feb;
            color: white;
            margin-left: 20px;
        }

        .message.ai {
            background: #21262d;
            margin-right: 20px;
        }

        .message.system {
            background: #2d1b00;
            color: #d29922;
            font-style: italic;
        }

        .chat-input-container {
            border-top: 1px solid #30363d;
            padding: 12px;
        }

        .chat-input {
            width: 100%;
            background: #21262d;
            border: 1px solid #30363d;
            border-radius: 6px;
            padding: 8px 12px;
            color: #c9d1d9;
            font-size: 13px;
            resize: none;
            outline: none;
        }

        .chat-input:focus {
            border-color: #58a6ff;
        }

        /* Output/Terminal pane */
        .output-terminal {
            grid-column: 1 / -1;
            background: #0d1117;
            border-top: 1px solid #30363d;
            display: flex;
            flex-direction: column;
        }

        .terminal-tabs {
            background: #161b22;
            border-bottom: 1px solid #30363d;
            display: flex;
        }

        .terminal-tab {
            padding: 6px 12px;
            border-right: 1px solid #30363d;
            cursor: pointer;
            font-size: 12px;
            color: #7d8590;
        }

        .terminal-tab.active {
            background: #0d1117;
            color: #c9d1d9;
        }

        .terminal-content {
            flex: 1;
            padding: 12px;
            overflow: auto;
            font-family: 'JetBrains Mono', monospace;
            font-size: 12px;
            line-height: 1.4;
        }

        .terminal-output {
            white-space: pre-wrap;
            color: #8b949e;
        }

        /* Responsive design */
        @media (max-width: 768px) {
            .main-container {
                grid-template-columns: 1fr;
                grid-template-rows: 40px 200px 1fr 150px;
            }

            .file-browser {
                border-right: none;
                border-bottom: 1px solid #30363d;
            }

            .ai-chat {
                border-left: none;
                border-top: 1px solid #30363d;
            }
        }

        /* Syntax highlighting */
        .keyword { color: #ff7b72; }
        .string { color: #a5d6ff; }
        .comment { color: #8b949e; font-style: italic; }
        .function { color: #d2a8ff; }
        .variable { color: #ffa657; }

        /* Modal customizations - WebTUI will handle most styling */
        .modal-overlay {
            position: fixed;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background: #1e3a8a; /* Solid blue background */
            display: none;
            z-index: 1000;
            align-items: center;
            justify-content: center;
        }

        /* Command palette styles */
        .command-palette {
            width: 600px;
            max-height: 400px;
        }

        .command-input {
            width: 100%;
            background: #0d1117;
            border: 1px solid #30363d;
            border-radius: 6px;
            padding: 12px 16px;
            color: #c9d1d9;
            font-size: 14px;
            margin-bottom: 12px;
        }

        .command-input:focus {
            outline: none;
            border-color: #58a6ff;
        }

        .command-list {
            max-height: 300px;
            overflow-y: auto;
        }

        .command-item {
            padding: 8px 12px;
            cursor: pointer;
            border-radius: 4px;
            display: flex;
            align-items: center;
            gap: 12px;
        }

        .command-item:hover,
        .command-item.selected {
            background: #21262d;
        }

        .command-icon {
            width: 16px;
            text-align: center;
        }

        .command-details {
            flex: 1;
        }

        .command-name {
            font-weight: 500;
            color: #c9d1d9;
        }

        .command-description {
            font-size: 12px;
            color: #8b949e;
        }

        .command-shortcut {
            font-size: 11px;
            color: #8b949e;
            background: #21262d;
            padding: 2px 6px;
            border-radius: 3px;
        }

        /* Settings panel styles */
        .settings-nav {
            display: flex;
            border-bottom: 1px solid #30363d;
            margin-bottom: 20px;
        }

        .settings-tab {
            padding: 8px 16px;
            cursor: pointer;
            border-bottom: 2px solid transparent;
            color: #8b949e;
        }

        .settings-tab.active {
            color: #58a6ff;
            border-bottom-color: #58a6ff;
        }

        .settings-section {
            display: none;
        }

        .settings-section.active {
            display: block;
        }

        .setting-group {
            margin-bottom: 24px;
        }

        .setting-label {
            display: block;
            margin-bottom: 8px;
            font-weight: 500;
            color: #c9d1d9;
        }

        .setting-input,
        .setting-select {
            width: 100%;
            background: #0d1117;
            border: 1px solid #30363d;
            border-radius: 6px;
            padding: 8px 12px;
            color: #c9d1d9;
            font-size: 14px;
        }

        .setting-input:focus,
        .setting-select:focus {
            outline: none;
            border-color: #58a6ff;
        }

        .setting-checkbox {
            margin-right: 8px;
        }

        .provider-card {
            background: #0d1117;
            border: 1px solid #30363d;
            border-radius: 6px;
            padding: 16px;
            margin-bottom: 12px;
        }

        .provider-header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            margin-bottom: 12px;
        }

        .provider-name {
            font-weight: 600;
            color: #c9d1d9;
        }

        .provider-status {
            font-size: 12px;
            padding: 2px 8px;
            border-radius: 12px;
        }

        .provider-status.available {
            background: #1a7f37;
            color: #ffffff;
        }

        .provider-status.unavailable {
            background: #da3633;
            color: #ffffff;
        }

        .model-list {
            display: grid;
            gap: 8px;
        }

        .model-item {
            display: flex;
            align-items: center;
            gap: 8px;
            padding: 8px;
            background: #21262d;
            border-radius: 4px;
        }

        .model-radio {
            margin: 0;
        }

        .model-name {
            flex: 1;
            color: #c9d1d9;
        }

        .model-tokens {
            font-size: 12px;
            color: #8b949e;
        }

        /* Responsive design */
        @media (max-width: 768px) {
            .main-container {
                grid-template-columns: 1fr;
                grid-template-rows: 40px 200px 1fr 150px;
            }

            .file-browser {
                border-right: none;
                border-bottom: 1px solid #30363d;
            }

            .ai-chat {
                border-left: none;
                border-top: 1px solid #30363d;
            }

            .modal {
                width: 95%;
                max-height: 90%;
            }

            .command-palette {
                width: 100%;
            }
        }

        /* Syntax highlighting */
        .keyword { color: #ff7b72; }
        .string { color: #a5d6ff; }
        .comment { color: #8b949e; font-style: italic; }
        .function { color: #d2a8ff; }
        .variable { color: #ffa657; }
    </style>
</head>
<body>
    <div class="main-container">
        <!-- Header Bar -->
        <div class="header-bar">
            <h1>🔧 CodeForge</h1>
            <div class="header-status">
                <button onclick="openCommandPalette()" style="background: none; border: none; color: #8b949e; cursor: pointer; margin-right: 12px;" title="Command Palette (Ctrl+Shift+P)">⌘</button>
                <button onclick="openSettings()" style="background: none; border: none; color: #8b949e; cursor: pointer; margin-right: 12px;" title="Settings">⚙️</button>
                <div class="status-indicator">
                    <div class="status-dot active" id="embeddingDot"></div>
                    <span>Embedding</span>
                </div>
                <div class="status-indicator">
                    <div class="status-dot warning" id="lspDot"></div>
                    <span>LSP</span>
                </div>
                <div class="status-indicator">
                    <div class="status-dot warning" id="mcpDot"></div>
                    <span>MCP</span>
                </div>
            </div>
        </div>

        <!-- File Browser -->
        <div class="file-browser">
            <div class="pane-header">FILES</div>
            <div class="file-tree" id="fileTree">
                <div class="file-item" onclick="openFile('README.md')">
                    <span class="file-icon"></span>
                    <span>README.md</span>
                </div>
                <div class="file-item" onclick="openFile('main.go')">
                    <span class="file-icon">🔧</span>
                    <span>main.go</span>
                </div>
                <div class="file-item" onclick="openFile('config.yaml')">
                    <span class="file-icon">⚙️</span>
                    <span>config.yaml</span>
                </div>
            </div>
        </div>

        <!-- Code Editor -->
        <div class="code-editor">
            <div class="editor-tabs">
                <div class="editor-tab active" id="welcomeTab">
                    <span>Welcome</span>
                    <span class="tab-close" onclick="closeTab('welcome')">×</span>
                </div>
            </div>
            <div class="editor-content">
                <textarea class="code-textarea" id="codeEditor" placeholder="// Welcome to CodeForge!
// Open a file from the file browser or start a new conversation with AI.

package main

import (
    &quot;fmt&quot;
)

func main() {
    fmt.Println(&quot;Hello, CodeForge!&quot;)
}"></textarea>
            </div>
        </div>

        <!-- AI Chat -->
        <div class="ai-chat">
            <div class="pane-header">AI ASSISTANT</div>
            <div class="chat-messages" id="chatMessages">
                <div class="message system">
                    CodeForge AI Assistant ready! Ask me about your code, request explanations, or get help with debugging.
                </div>
            </div>
            <div class="chat-input-container">
                <textarea class="chat-input" id="chatInput" placeholder="Ask me anything about your code..." rows="3"></textarea>
            </div>
        </div>

        <!-- Output/Terminal -->
        <div class="output-terminal">
            <div class="terminal-tabs">
                <div class="terminal-tab active" onclick="switchTerminalTab('output')">Output</div>
                <div class="terminal-tab" onclick="switchTerminalTab('terminal')">Terminal</div>
                <div class="terminal-tab" onclick="switchTerminalTab('problems')">Problems</div>
            </div>
            <div class="terminal-content">
                <div class="terminal-output" id="terminalOutput">
CodeForge initialized successfully.
Ready for development.

Use Ctrl+backtick to focus terminal, Ctrl+1 for files, Ctrl+2 for editor, Ctrl+3 for AI chat.
                </div>
            </div>
        </div>
    </div>

    <!-- Command Palette Modal using WebTUI -->
    <div class="modal-overlay" id="commandPaletteModal">
        <div class="wt-modal wt-modal-lg command-palette">
            <div class="wt-modal-header">
                <h4 class="wt-modal-title">Command Palette</h4>
                <button class="wt-btn wt-btn-ghost wt-btn-sm" onclick="closeModal('commandPaletteModal')">×</button>
            </div>
            <div class="wt-modal-body">
                <input type="text" class="wt-input command-input" id="commandInput" placeholder="Type a command..." />
                <div class="wt-list command-list" id="commandList">
                    <div class="wt-list-item command-item" onclick="executeCommand('file.new')">
                        <div class="command-icon"></div>
                        <div class="command-details">
                            <div class="command-name">New File</div>
                            <div class="wt-text-muted command-description">Create a new file</div>
                        </div>
                        <div class="wt-badge command-shortcut">Ctrl+N</div>
                    </div>
                    <div class="wt-list-item command-item" onclick="executeCommand('file.save')">
                        <div class="command-icon">💾</div>
                        <div class="command-details">
                            <div class="command-name">Save File</div>
                            <div class="wt-text-muted command-description">Save the current file</div>
                        </div>
                        <div class="wt-badge command-shortcut">Ctrl+S</div>
                    </div>
                    <div class="wt-list-item command-item" onclick="executeCommand('build.run')">
                        <div class="command-icon">🔨</div>
                        <div class="command-details">
                            <div class="command-name">Run Build</div>
                            <div class="wt-text-muted command-description">Build the current project</div>
                        </div>
                        <div class="wt-badge command-shortcut">Ctrl+B</div>
                    </div>
                    <div class="wt-list-item command-item" onclick="executeCommand('ai.chat')">
                        <div class="command-icon"></div>
                        <div class="command-details">
                            <div class="command-name">Focus AI Chat</div>
                            <div class="wt-text-muted command-description">Focus the AI chat input</div>
                        </div>
                        <div class="wt-badge command-shortcut">Ctrl+3</div>
                    </div>
                    <div class="wt-list-item command-item" onclick="executeCommand('settings.open')">
                        <div class="command-icon">⚙️</div>
                        <div class="command-details">
                            <div class="command-name">Open Settings</div>
                            <div class="wt-text-muted command-description">Open the settings panel</div>
                        </div>
                        <div class="wt-badge command-shortcut">Ctrl+,</div>
                    </div>
                </div>
            </div>
        </div>
    </div>

    <!-- Settings Modal using WebTUI -->
    <div class="modal-overlay" id="settingsModal">
        <div class="wt-modal wt-modal-xl">
            <div class="wt-modal-header">
                <h4 class="wt-modal-title">Settings</h4>
                <button class="wt-btn wt-btn-ghost wt-btn-sm" onclick="closeModal('settingsModal')">×</button>
            </div>
            <div class="wt-modal-body">
                <div class="wt-tabs settings-nav">
                    <div class="wt-tab wt-tab-active settings-tab" onclick="switchSettingsTab('llm')">LLM Providers</div>
                    <div class="wt-tab settings-tab" onclick="switchSettingsTab('editor')">Editor</div>
                    <div class="wt-tab settings-tab" onclick="switchSettingsTab('mcp')">MCP Tools</div>
                    <div class="wt-tab settings-tab" onclick="switchSettingsTab('terminal')">Terminal</div>
                </div>

                <!-- LLM Providers Settings -->
                <div class="settings-section active" id="llmSettings">
                    <div id="providersContainer">
                        <!-- Providers will be loaded dynamically -->
                    </div>
                </div>

                <!-- Editor Settings -->
                <div class="settings-section" id="editorSettings">
                    <div class="wt-form-group setting-group">
                        <label class="wt-label">Theme</label>
                        <select class="wt-select" id="editorTheme">
                            <option value="dark">Dark</option>
                            <option value="light">Light</option>
                        </select>
                    </div>
                    <div class="wt-form-group setting-group">
                        <label class="wt-label">Font Size</label>
                        <input type="number" class="wt-input" id="editorFontSize" value="14" min="10" max="24" />
                    </div>
                    <div class="wt-form-group setting-group">
                        <label class="wt-label">Tab Size</label>
                        <input type="number" class="wt-input" id="editorTabSize" value="4" min="2" max="8" />
                    </div>
                    <div class="wt-form-group setting-group">
                        <label class="wt-label wt-checkbox">
                            <input type="checkbox" id="editorWordWrap" checked />
                            <span class="wt-checkmark"></span>
                            Word Wrap
                        </label>
                    </div>
                    <div class="wt-form-group setting-group">
                        <label class="wt-label wt-checkbox">
                            <input type="checkbox" id="editorLineNumbers" checked />
                            <span class="wt-checkmark"></span>
                            Line Numbers
                        </label>
                    </div>
                </div>

                <!-- MCP Tools Settings -->
                <div class="settings-section" id="mcpSettings">
                    <div class="setting-group">
                        <label class="setting-label">
                            <input type="checkbox" class="setting-checkbox" id="mcpAutoStart" checked />
                            Auto-start MCP servers
                        </label>
                    </div>
                    <div id="mcpServersContainer">
                        <!-- MCP servers will be loaded dynamically -->
                    </div>
                </div>

                <!-- Terminal Settings -->
                <div class="settings-section" id="terminalSettings">
                    <div class="wt-form-group setting-group">
                        <label class="wt-label">Shell</label>
                        <input type="text" class="wt-input" id="terminalShell" value="/bin/bash" />
                    </div>
                    <div class="wt-form-group setting-group">
                        <label class="wt-label">Font Size</label>
                        <input type="number" class="wt-input" id="terminalFontSize" value="12" min="8" max="20" />
                    </div>
                    <div class="wt-form-group setting-group">
                        <label class="wt-label">Scrollback Lines</label>
                        <input type="number" class="wt-input" id="terminalScrollback" value="1000" min="100" max="10000" />
                    </div>
                </div>
            </div>
        </div>
    </div>

    <script>
        // Global state
        let currentFile = null;
        let openTabs = ['welcome'];
        let activeTab = 'welcome';

        // Initialize WebSocket connection
        const ws = new WebSocket('ws://localhost:8080/ws');

        ws.onopen = function() {
            addSystemMessage('Connected to CodeForge server');
        };

        ws.onmessage = function(event) {
            const data = JSON.parse(event.data);
            if (data.type === 'status') {
                updateStatus(data.data);
            }
        };

        // File operations
        function openFile(filename) {
            // Remove selection from other files
            document.querySelectorAll('.file-item').forEach(item => {
                item.classList.remove('selected');
            });

            // Select current file
            event.target.closest('.file-item').classList.add('selected');

            // Add tab if not already open
            if (!openTabs.includes(filename)) {
                openTabs.push(filename);
                addTab(filename);
            }

            // Switch to tab
            switchTab(filename);
            currentFile = filename;

            // Load file content from API
            loadFileContent(filename);
        }

        function addTab(filename) {
            const tabsContainer = document.querySelector('.editor-tabs');
            const tab = document.createElement('div');
            tab.className = 'editor-tab';
            tab.id = filename + 'Tab';
            tab.innerHTML = '<span>' + filename + '</span><span class="tab-close" onclick="closeTab(\'' + filename + '\')">×</span>';
            tab.onclick = () => switchTab(filename);
            tabsContainer.appendChild(tab);
        }

        function switchTab(filename) {
            // Update tab appearance
            document.querySelectorAll('.editor-tab').forEach(tab => {
                tab.classList.remove('active');
            });
            document.getElementById(filename + 'Tab').classList.add('active');

            activeTab = filename;
            loadFileContent(filename);
        }

        function closeTab(filename) {
            openTabs = openTabs.filter(tab => tab !== filename);
            document.getElementById(filename + 'Tab').remove();

            if (activeTab === filename) {
                if (openTabs.length > 0) {
                    switchTab(openTabs[openTabs.length - 1]);
                } else {
                    // No tabs left, show empty state
                    activeTab = null;
                    document.getElementById('codeEditor').value = '// No files open\n// Open a file from the file browser to start editing';
                }
            }
        }

        function loadFileContent(filename) {
            const editor = document.getElementById('codeEditor');

            if (filename === 'welcome') {
                editor.value = '// Welcome to CodeForge!\\n// Open a file from the file browser or start a new conversation with AI.\\n\\npackage main\\n\\nimport (\\n    "fmt"\\n)\\n\\nfunc main() {\\n    fmt.Println("Hello, CodeForge!")\\n}';
                return;
            }

            // Load file content from API
            fetch('/api/files', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ operation: 'read', path: filename })
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    editor.value = data.data.content;
                } else {
                    editor.value = '// Error loading file: ' + data.error;
                }
            })
            .catch(error => {
                editor.value = '// Error loading file: ' + error.message;
            });
        }

        // Chat functionality
        function sendMessage() {
            const input = document.getElementById('chatInput');
            const message = input.value.trim();

            if (!message) return;

            addUserMessage(message);
            input.value = '';

            // Send to AI
            fetch('/api/chat', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ message: message })
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    addAIMessage(data.data.message);
                } else {
                    addAIMessage('Error: ' + data.error);
                }
            })
            .catch(error => {
                addAIMessage('Error: ' + error.message);
            });
        }

        function addUserMessage(message) {
            const messagesContainer = document.getElementById('chatMessages');
            const messageDiv = document.createElement('div');
            messageDiv.className = 'message user';
            messageDiv.textContent = message;
            messagesContainer.appendChild(messageDiv);
            messagesContainer.scrollTop = messagesContainer.scrollHeight;
        }

        function addAIMessage(message) {
            const messagesContainer = document.getElementById('chatMessages');
            const messageDiv = document.createElement('div');
            messageDiv.className = 'message ai';
            messageDiv.textContent = message;
            messagesContainer.appendChild(messageDiv);
            messagesContainer.scrollTop = messagesContainer.scrollHeight;
        }

        function addSystemMessage(message) {
            const messagesContainer = document.getElementById('chatMessages');
            const messageDiv = document.createElement('div');
            messageDiv.className = 'message system';
            messageDiv.textContent = message;
            messagesContainer.appendChild(messageDiv);
            messagesContainer.scrollTop = messagesContainer.scrollHeight;
        }

        // Terminal functionality
        function switchTerminalTab(tab) {
            document.querySelectorAll('.terminal-tab').forEach(t => {
                t.classList.remove('active');
            });
            event.target.classList.add('active');

            const output = document.getElementById('terminalOutput');
            const content = {
                'output': 'CodeForge initialized successfully.\\nReady for development.\\n\\nUse Ctrl+backtick to focus terminal, Ctrl+1 for files, Ctrl+2 for editor, Ctrl+3 for AI chat.',
                'terminal': '$ echo "Welcome to CodeForge terminal"\\nWelcome to CodeForge terminal\\n$ ',
                'problems': 'No problems detected.\\n\\nLSP diagnostics will appear here when available.'
            };

            output.textContent = content[tab] || '';
        }

        // Status updates
        function updateStatus(status) {
            document.getElementById('embeddingDot').className = 'status-dot ' + (status.embedding ? 'active' : '');
            document.getElementById('lspDot').className = 'status-dot ' + (status.lsp ? 'active' : 'warning');
            document.getElementById('mcpDot').className = 'status-dot ' + (status.mcp ? 'active' : 'warning');
        }

        // Modal functions
        function openCommandPalette() {
            document.getElementById('commandPaletteModal').style.display = 'flex';
            document.getElementById('commandInput').focus();
        }

        function openSettings() {
            document.getElementById('settingsModal').style.display = 'flex';
            loadProviders();
            loadMCPServers();
        }

        function closeModal(modalId) {
            document.getElementById(modalId).style.display = 'none';
        }

        function executeCommand(command) {
            fetch('/api/commands', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ command: command })
            })
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    handleCommandAction(data.data.action);
                }
                closeModal('commandPaletteModal');
            })
            .catch(error => {
                console.error('Command execution failed:', error);
            });
        }

        function handleCommandAction(action) {
            switch(action) {
                case 'new_file':
                    // Create new file logic
                    break;
                case 'save_file':
                    // Save current file logic
                    break;
                case 'focus_chat':
                    document.getElementById('chatInput').focus();
                    break;
                case 'open_settings':
                    openSettings();
                    break;
            }
        }

        function switchSettingsTab(tab) {
            // Update tab appearance
            document.querySelectorAll('.settings-tab').forEach(t => t.classList.remove('active'));
            document.querySelector('.settings-tab[onclick*="' + tab + '"]').classList.add('active');

            // Update section visibility
            document.querySelectorAll('.settings-section').forEach(s => s.classList.remove('active'));
            document.getElementById(tab + 'Settings').classList.add('active');
        }

        function loadProviders() {
            fetch('/api/providers')
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    renderProviders(data.data);
                }
            })
            .catch(error => {
                console.error('Failed to load providers:', error);
            });
        }

        function renderProviders(providers) {
            const container = document.getElementById('providersContainer');
            container.innerHTML = '';

            Object.entries(providers).forEach(([key, provider]) => {
                const card = document.createElement('div');
                card.className = 'provider-card';
                card.innerHTML =
                    '<div class="provider-header">' +
                        '<div class="provider-name">' + provider.name + '</div>' +
                        '<div class="provider-status ' + (provider.available ? 'available' : 'unavailable') + '">' +
                            (provider.available ? 'Available' : 'Unavailable') +
                        '</div>' +
                    '</div>' +
                    '<div class="model-list">' +
                        provider.models.map(model =>
                            '<div class="model-item">' +
                                '<input type="radio" name="selectedModel" value="' + model.id + '" class="model-radio" />' +
                                '<div class="model-name">' + model.name + '</div>' +
                                '<div class="model-tokens">' + model.maxTokens.toLocaleString() + ' tokens</div>' +
                            '</div>'
                        ).join('') +
                    '</div>';
                container.appendChild(card);
            });
        }

        function loadMCPServers() {
            fetch('/api/mcp/tools')
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    renderMCPServers(data.data);
                }
            })
            .catch(error => {
                console.error('Failed to load MCP servers:', error);
            });
        }

        function renderMCPServers(servers) {
            const container = document.getElementById('mcpServersContainer');
            container.innerHTML = '';

            Object.entries(servers).forEach(([serverName, tools]) => {
                const serverDiv = document.createElement('div');
                serverDiv.className = 'setting-group';
                serverDiv.innerHTML =
                    '<label class="setting-label">' +
                        '<input type="checkbox" class="setting-checkbox" checked />' +
                        serverName + ' (' + tools.length + ' tools)' +
                    '</label>';
                container.appendChild(serverDiv);
            });
        }

        // Keyboard shortcuts - F-key standardized action bar
        document.addEventListener('keydown', function(e) {
            // F-key shortcuts (no modifiers needed)
            switch(e.key) {
                case 'F1':
                    e.preventDefault();
                    document.querySelector('.file-browser').focus();
                    break;
                case 'F2':
                    e.preventDefault();
                    document.getElementById('codeEditor').focus();
                    break;
                case 'F3':
                    e.preventDefault();
                    document.getElementById('chatInput').focus();
                    break;
                case 'F4':
                    e.preventDefault();
                    document.querySelector('.terminal-content').focus();
                    break;
                case 'F5':
                    e.preventDefault();
                    openSettings();
                    break;
                case 'F6':
                    e.preventDefault();
                    openCommandPalette();
                    break;
            }

            // Legacy Ctrl shortcuts for compatibility
            if (e.ctrlKey || e.metaKey) {
                switch(e.key) {
                    case '1':
                        e.preventDefault();
                        document.querySelector('.file-browser').focus();
                        break;
                    case '2':
                        e.preventDefault();
                        document.getElementById('codeEditor').focus();
                        break;
                    case '3':
                        e.preventDefault();
                        document.getElementById('chatInput').focus();
                        break;
                    case String.fromCharCode(96):
                        e.preventDefault();
                        document.querySelector('.terminal-content').focus();
                        break;
                    case ',':
                        e.preventDefault();
                        openSettings();
                        break;
                }
            }

            if (e.ctrlKey && e.shiftKey && e.key === 'P') {
                e.preventDefault();
                openCommandPalette();
            }

            if (e.key === 'Escape') {
                closeModal('commandPaletteModal');
                closeModal('settingsModal');
            }
        });

        // Chat input handling
        document.getElementById('chatInput').addEventListener('keydown', function(e) {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                sendMessage();
            }
        });

        // Load initial status
        fetch('/api/status')
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    updateStatus(data.data);
                }
            })
            .catch(error => {
                console.error('Failed to load status:', error);
            });
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

// handleChat handles AI chat requests
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use integrated app if available
	if s.app != nil {
		response, err := s.app.ProcessChatMessage(ctx, "web-session", req.Message, req.Model)
		if err != nil {
			s.sendError(w, fmt.Sprintf("AI completion failed: %v", err), http.StatusInternalServerError)
			return
		}

		s.sendSuccess(w, map[string]interface{}{
			"message": response,
			"model":   req.Model,
		})
		return
	}

	// Fallback to direct LLM integration
	// Get the default model
	defaultModel := llm.GetDefaultModel()

	// Create completion request
	temp := 0.7
	completionReq := llm.CompletionRequest{
		Model: defaultModel.ID,
		Messages: []llm.Message{
			{
				Role: "user",
				Content: []llm.ContentBlock{
					llm.TextBlock{Text: req.Message},
				},
			},
		},
		MaxTokens:   defaultModel.Info.MaxTokens,
		Temperature: &temp,
	}

	// Get completion
	resp, err := llm.GetCompletion(ctx, completionReq)
	if err != nil {
		s.sendError(w, fmt.Sprintf("AI completion failed: %v", err), http.StatusInternalServerError)
		return
	}

	s.sendSuccess(w, map[string]interface{}{
		"message": resp.Content,
		"model":   defaultModel.ID,
	})
}

// handleFiles handles file operations
func (s *Server) handleFiles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// List files in working directory
		files, err := s.listFiles(".")
		if err != nil {
			s.sendError(w, fmt.Sprintf("Failed to list files: %v", err), http.StatusInternalServerError)
			return
		}
		s.sendSuccess(w, files)

	case "POST":
		var req FileRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.sendError(w, "Invalid request format", http.StatusBadRequest)
			return
		}

		switch req.Operation {
		case "read":
			// Use integrated file operations if app is available
			if s.app != nil && s.app.FileOperationManager != nil {
				ctx := r.Context()
				fileReq := &permissions.FileOperationRequest{
					SessionID: "web-session",
					Operation: "read",
					Path:      req.Path,
					Context:   map[string]interface{}{"source": "web"},
				}

				result, err := s.app.FileOperationManager.ReadFile(ctx, fileReq)
				if err != nil {
					s.sendError(w, fmt.Sprintf("Failed to read file: %v", err), http.StatusInternalServerError)
					return
				}

				if !result.Success {
					s.sendError(w, result.Error, http.StatusForbidden)
					return
				}

				s.sendSuccess(w, map[string]string{"content": string(result.Content)})
			} else {
				// Fallback to direct file reading
				content, err := s.readFile(req.Path)
				if err != nil {
					s.sendError(w, fmt.Sprintf("Failed to read file: %v", err), http.StatusInternalServerError)
					return
				}
				s.sendSuccess(w, map[string]string{"content": content})
			}

		case "write":
			// Use integrated file operations if app is available
			if s.app != nil && s.app.FileOperationManager != nil {
				ctx := r.Context()
				fileReq := &permissions.FileOperationRequest{
					SessionID: "web-session",
					Operation: "write",
					Path:      req.Path,
					Content:   []byte(req.Content),
					Context:   map[string]interface{}{"source": "web"},
				}

				result, err := s.app.FileOperationManager.WriteFile(ctx, fileReq)
				if err != nil {
					s.sendError(w, fmt.Sprintf("Failed to write file: %v", err), http.StatusInternalServerError)
					return
				}

				if !result.Success {
					s.sendError(w, result.Error, http.StatusForbidden)
					return
				}

				s.sendSuccess(w, map[string]string{"status": "saved"})
			} else {
				// Fallback to direct file writing
				if err := s.writeFile(req.Path, req.Content); err != nil {
					s.sendError(w, fmt.Sprintf("Failed to write file: %v", err), http.StatusInternalServerError)
					return
				}
				s.sendSuccess(w, map[string]string{"status": "saved"})
			}

		default:
			s.sendError(w, "Unknown file operation", http.StatusBadRequest)
		}
	}
}

// handleBuild handles build operations
func (s *Server) handleBuild(w http.ResponseWriter, r *http.Request) {
	var req BuildRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Execute real build using the multi-language build system
	var buildOutput []byte
	var buildErr error

	if req.Language != "" {
		// Use language-specific build
		if lang, exists := builder.SupportedLanguages[req.Language]; exists {
			buildOutput, buildErr = builder.BuildWithLanguage(".", lang)
		} else {
			s.sendError(w, fmt.Sprintf("Unsupported language: %s", req.Language), http.StatusBadRequest)
			return
		}
	} else {
		// Auto-detect language and build
		buildOutput, buildErr = builder.Build(".")
	}

	status := "success"
	output := string(buildOutput)
	if buildErr != nil {
		status = "error"
		output = fmt.Sprintf("Build failed: %v\n%s", buildErr, output)
	}

	result := map[string]interface{}{
		"status":   status,
		"language": req.Language,
		"output":   output,
		"command":  req.Command,
	}

	s.sendSuccess(w, result)
}

// handleSettings handles settings operations
func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// Load settings from config file
		settings, err := s.loadSettings()
		if err != nil {
			// Return default settings if loading fails
			settings = s.getDefaultSettings()
		}
		s.sendSuccess(w, settings)

	case "POST":
		var req SettingsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.sendError(w, "Invalid request format", http.StatusBadRequest)
			return
		}

		// Save settings to config file
		err := s.saveSettings(req.Category, req.Settings)
		if err != nil {
			s.sendError(w, fmt.Sprintf("Failed to save settings: %v", err), http.StatusInternalServerError)
			return
		}

		s.sendSuccess(w, map[string]string{"status": "saved", "category": req.Category})
	}
}

// handleCommands handles command palette operations
func (s *Server) handleCommands(w http.ResponseWriter, r *http.Request) {
	var req CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// Execute command based on type
	switch req.Command {
	case "file.new":
		s.sendSuccess(w, map[string]string{"action": "new_file"})
	case "file.save":
		s.sendSuccess(w, map[string]string{"action": "save_file"})
	case "file.open":
		s.sendSuccess(w, map[string]string{"action": "open_file"})
	case "build.run":
		s.sendSuccess(w, map[string]string{"action": "run_build"})
	case "ai.chat":
		s.sendSuccess(w, map[string]string{"action": "focus_chat"})
	case "settings.open":
		s.sendSuccess(w, map[string]string{"action": "open_settings"})
	default:
		s.sendError(w, "Unknown command", http.StatusBadRequest)
	}
}

// handleProviders returns available LLM providers and models
func (s *Server) handleProviders(w http.ResponseWriter, r *http.Request) {
	providers := map[string]interface{}{
		"anthropic": map[string]interface{}{
			"name":      "Anthropic",
			"available": true,
			"models": []map[string]interface{}{
				{"id": "claude-3-5-sonnet-20241022", "name": "Claude 3.5 Sonnet", "maxTokens": 200000},
				{"id": "claude-3-5-haiku-20241022", "name": "Claude 3.5 Haiku", "maxTokens": 200000},
			},
		},
		"openai": map[string]interface{}{
			"name":      "OpenAI",
			"available": true,
			"models": []map[string]interface{}{
				{"id": "gpt-4o", "name": "GPT-4o", "maxTokens": 128000},
				{"id": "gpt-4o-mini", "name": "GPT-4o Mini", "maxTokens": 128000},
			},
		},
		"groq": map[string]interface{}{
			"name":      "Groq",
			"available": false,
			"models": []map[string]interface{}{
				{"id": "llama-3.1-70b-versatile", "name": "Llama 3.1 70B", "maxTokens": 32768},
			},
		},
	}

	s.sendSuccess(w, providers)
}

// saveSettings saves settings to the configuration file
func (s *Server) saveSettings(category string, settings map[string]interface{}) error {
	// Get the current config
	cfg := config.Get()
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	// Get the config file path
	configPath, err := s.getConfigFilePath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	// Load existing config from file
	existingConfig := make(map[string]interface{})
	if configData, err := os.ReadFile(configPath); err == nil {
		json.Unmarshal(configData, &existingConfig)
	}

	// Update the specific category
	existingConfig[category] = settings

	// Write back to file
	configData, err := json.MarshalIndent(existingConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure config directory exists
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// getConfigFilePath returns the path to the configuration file
func (s *Server) getConfigFilePath() (string, error) {
	// Try XDG config directory first
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		return filepath.Join(xdgConfig, "codeforge", ".codeforge"), nil
	}

	// Fall back to home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	return filepath.Join(homeDir, ".codeforge"), nil
}

// loadSettings loads settings from the configuration file
func (s *Server) loadSettings() (map[string]interface{}, error) {
	configPath, err := s.getConfigFilePath()
	if err != nil {
		return nil, fmt.Errorf("failed to get config path: %w", err)
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return s.getDefaultSettings(), nil
	}

	// Read config file
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	var settings map[string]interface{}
	if err := json.Unmarshal(configData, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return settings, nil
}

// getDefaultSettings returns the default settings configuration
func (s *Server) getDefaultSettings() map[string]interface{} {
	return map[string]interface{}{
		"llm": map[string]interface{}{
			"defaultProvider": "anthropic",
			"defaultModel":    "claude-3-5-sonnet-20241022",
			"temperature":     0.7,
			"maxTokens":       4096,
		},
		"editor": map[string]interface{}{
			"theme":       "dark",
			"fontSize":    14,
			"tabSize":     4,
			"wordWrap":    true,
			"lineNumbers": true,
		},
		"terminal": map[string]interface{}{
			"shell":      "/bin/bash",
			"fontSize":   12,
			"scrollback": 1000,
		},
		"mcp": map[string]interface{}{
			"autoStart":      true,
			"enabledServers": []string{"filesystem"},
		},
	}
}

// Helper functions for file operations
func (s *Server) listFiles(dir string) ([]map[string]interface{}, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []map[string]interface{}
	for _, entry := range entries {
		// Skip hidden files
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		var icon string
		fileType := "file"

		if entry.IsDir() {
			icon = ""
			fileType = "directory"
		} else {
			switch {
			case strings.HasSuffix(entry.Name(), ".go"):
				icon = "🔧"
			case strings.HasSuffix(entry.Name(), ".rs"):
				icon = "🦀"
			case strings.HasSuffix(entry.Name(), ".py"):
				icon = "🐍"
			case strings.HasSuffix(entry.Name(), ".js"), strings.HasSuffix(entry.Name(), ".ts"):
				icon = "📜"
			case strings.HasSuffix(entry.Name(), ".md"):
				icon = ""
			case strings.HasSuffix(entry.Name(), ".yaml"), strings.HasSuffix(entry.Name(), ".yml"):
				icon = "⚙️"
			default:
				icon = ""
			}
		}

		files = append(files, map[string]interface{}{
			"name": entry.Name(),
			"type": fileType,
			"icon": icon,
		})
	}

	return files, nil
}

func (s *Server) readFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (s *Server) writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// handleSearch handles semantic code search requests
func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	if req.Limit == 0 {
		req.Limit = 10
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Generate embedding for query
	embedding, err := embeddings.GetEmbedding(ctx, req.Query)
	if err != nil {
		s.sendError(w, fmt.Sprintf("Failed to generate embedding: %v", err), http.StatusInternalServerError)
		return
	}

	// Search vector database
	vdb := vectordb.Get()
	if vdb == nil {
		s.sendError(w, "Vector database not available", http.StatusServiceUnavailable)
		return
	}

	results, err := vdb.SearchSimilarCode(ctx, embedding, req.Language, req.Limit)
	if err != nil {
		s.sendError(w, fmt.Sprintf("Search failed: %v", err), http.StatusInternalServerError)
		return
	}

	s.sendSuccess(w, results)
}

// handleLSP handles LSP operation requests
func (s *Server) handleLSP(w http.ResponseWriter, r *http.Request) {
	var req LSPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	manager := lsp.GetManager()
	if manager == nil {
		s.sendError(w, "LSP manager not available", http.StatusServiceUnavailable)
		return
	}

	switch req.Operation {
	case "symbols":
		client := manager.GetClientForLanguage("go") // Default to Go for demo
		if client == nil {
			s.sendError(w, "No LSP client available", http.StatusServiceUnavailable)
			return
		}

		symbols, err := client.GetWorkspaceSymbols(ctx, req.Query)
		if err != nil {
			s.sendError(w, fmt.Sprintf("Symbol search failed: %v", err), http.StatusInternalServerError)
			return
		}
		s.sendSuccess(w, symbols)

	case "definition":
		client := manager.GetClientForFile(req.FilePath)
		if client == nil {
			s.sendError(w, "No LSP client available for file", http.StatusServiceUnavailable)
			return
		}

		locations, err := client.GetDefinition(ctx, req.FilePath, req.Line, req.Character)
		if err != nil {
			s.sendError(w, fmt.Sprintf("Definition search failed: %v", err), http.StatusInternalServerError)
			return
		}
		s.sendSuccess(w, locations)

	default:
		s.sendError(w, "Unknown LSP operation", http.StatusBadRequest)
	}
}

// handleMCP handles MCP operation requests
func (s *Server) handleMCP(w http.ResponseWriter, r *http.Request) {
	var req MCPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, "Invalid request format", http.StatusBadRequest)
		return
	}

	// MCP is now a standalone server - provide information about capabilities
	response := map[string]interface{}{
		"message": "MCP is now a standalone server. Use 'codeforge mcp server' to start it.",
		"capabilities": map[string]interface{}{
			"tools":     []string{"semantic_search", "read_file", "write_file", "analyze_code", "get_project_structure"},
			"resources": []string{"codeforge://project/metadata", "codeforge://files/{path}", "codeforge://git/status"},
			"prompts":   []string{"code_review", "debug_help", "refactoring_guide", "documentation_help", "testing_help"},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleMCPTools returns available MCP tools
func (s *Server) handleMCPTools(w http.ResponseWriter, r *http.Request) {
	tools := []map[string]interface{}{
		{"name": "semantic_search", "description": "Search for code using semantic similarity"},
		{"name": "read_file", "description": "Read file contents from the workspace"},
		{"name": "write_file", "description": "Write content to files in the workspace"},
		{"name": "analyze_code", "description": "Analyze code structure and extract symbols"},
		{"name": "get_project_structure", "description": "Get directory structure of the project"},
	}
	s.sendSuccess(w, tools)
}

// handleMCPResources returns available MCP resources
func (s *Server) handleMCPResources(w http.ResponseWriter, r *http.Request) {
	resources := []map[string]interface{}{
		{"uri": "codeforge://project/metadata", "description": "Project information"},
		{"uri": "codeforge://files/{path}", "description": "File content access"},
		{"uri": "codeforge://git/status", "description": "Git repository status"},
	}
	s.sendSuccess(w, resources)
}

// handleMCPServers handles MCP server management
func (s *Server) handleMCPServers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		// Return list of configured MCP servers
		if s.app != nil && s.app.MCPManager != nil {
			servers := s.app.MCPManager.GetAllServers()
			s.sendSuccess(w, servers)
		} else {
			s.sendSuccess(w, []interface{}{})
		}
	case "POST":
		// Add new MCP server
		var config map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			s.sendError(w, "Invalid request format", http.StatusBadRequest)
			return
		}

		// Implement server addition
		if s.app != nil && s.app.MCPManager != nil {
			// Convert map to MCPServerConfig
			serverConfig, err := s.convertToMCPServerConfig(config)
			if err != nil {
				s.sendError(w, fmt.Sprintf("Invalid server configuration: %v", err), http.StatusBadRequest)
				return
			}

			// Add the server
			if err := s.app.MCPManager.AddServer(serverConfig); err != nil {
				s.sendError(w, fmt.Sprintf("Failed to add server: %v", err), http.StatusInternalServerError)
				return
			}

			s.sendSuccess(w, map[string]interface{}{
				"status": "server added",
				"server": serverConfig,
			})
		} else {
			s.sendError(w, "MCP manager not available", http.StatusServiceUnavailable)
		}
	}
}

// handleMCPServer handles individual MCP server operations
func (s *Server) handleMCPServer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverName := vars["name"]

	switch r.Method {
	case "GET":
		// Get server details
		if s.app != nil && s.app.MCPManager != nil {
			server := s.app.MCPManager.GetServer(serverName)
			if server != nil {
				s.sendSuccess(w, server)
			} else {
				s.sendError(w, "Server not found", http.StatusNotFound)
			}
		} else {
			s.sendError(w, "MCP manager not available", http.StatusServiceUnavailable)
		}
	case "PUT":
		// Update server configuration
		var config map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			s.sendError(w, "Invalid request format", http.StatusBadRequest)
			return
		}

		// Implement server update
		if s.app != nil && s.app.MCPManager != nil {
			// Get existing server configuration
			existingServer := s.app.MCPManager.GetServer(serverName)
			if existingServer == nil {
				s.sendError(w, "Server not found", http.StatusNotFound)
				return
			}

			// Convert map to MCPServerConfig and merge with existing
			updatedConfig, err := s.convertToMCPServerConfig(config)
			if err != nil {
				s.sendError(w, fmt.Sprintf("Invalid server configuration: %v", err), http.StatusBadRequest)
				return
			}

			// Preserve the original name
			updatedConfig.Name = serverName

			// Remove the old server and add the updated one
			if err := s.app.MCPManager.RemoveServer(serverName); err != nil {
				s.sendError(w, fmt.Sprintf("Failed to remove old server: %v", err), http.StatusInternalServerError)
				return
			}

			if err := s.app.MCPManager.AddServer(updatedConfig); err != nil {
				s.sendError(w, fmt.Sprintf("Failed to add updated server: %v", err), http.StatusInternalServerError)
				return
			}

			s.sendSuccess(w, map[string]interface{}{
				"status": "server updated",
				"server": updatedConfig,
			})
		} else {
			s.sendError(w, "MCP manager not available", http.StatusServiceUnavailable)
		}
	case "DELETE":
		// Remove server
		if s.app != nil && s.app.MCPManager != nil {
			err := s.app.MCPManager.RemoveServer(serverName)
			if err != nil {
				s.sendError(w, fmt.Sprintf("Failed to remove server: %v", err), http.StatusInternalServerError)
			} else {
				s.sendSuccess(w, map[string]string{"status": "server removed"})
			}
		} else {
			s.sendError(w, "MCP manager not available", http.StatusServiceUnavailable)
		}
	}
}

// handleMCPServerEnable enables an MCP server
func (s *Server) handleMCPServerEnable(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverName := vars["name"]

	if s.app != nil && s.app.MCPManager != nil {
		err := s.app.MCPManager.EnableServer(serverName)
		if err != nil {
			s.sendError(w, fmt.Sprintf("Failed to enable server: %v", err), http.StatusInternalServerError)
		} else {
			s.sendSuccess(w, map[string]string{"status": "server enabled"})
		}
	} else {
		s.sendError(w, "MCP manager not available", http.StatusServiceUnavailable)
	}
}

// handleMCPServerDisable disables an MCP server
func (s *Server) handleMCPServerDisable(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverName := vars["name"]

	if s.app != nil && s.app.MCPManager != nil {
		err := s.app.MCPManager.DisableServer(serverName)
		if err != nil {
			s.sendError(w, fmt.Sprintf("Failed to disable server: %v", err), http.StatusInternalServerError)
		} else {
			s.sendSuccess(w, map[string]string{"status": "server disabled"})
		}
	} else {
		s.sendError(w, "MCP manager not available", http.StatusServiceUnavailable)
	}
}

// handleMCPDiscover discovers available MCP servers
func (s *Server) handleMCPDiscover(w http.ResponseWriter, r *http.Request) {
	if s.app != nil && s.app.MCPManager != nil {
		servers, err := s.app.MCPManager.DiscoverServers()
		if err != nil {
			s.sendError(w, fmt.Sprintf("Failed to discover servers: %v", err), http.StatusInternalServerError)
		} else {
			s.sendSuccess(w, servers)
		}
	} else {
		s.sendError(w, "MCP manager not available", http.StatusServiceUnavailable)
	}
}

// handleStatus returns system status
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"embedding": embeddings.Get() != nil,
		"vectordb":  vectordb.Get() != nil,
		"lsp":       lsp.GetManager() != nil,
		"mcp":       true, // MCP server is available as standalone
		"timestamp": time.Now().Unix(),
	}

	s.sendSuccess(w, status)
}

// handleWebSocket handles WebSocket connections for real-time updates
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("WebSocket upgrade failed: %v\n", err)
		return
	}
	defer conn.Close()

	s.clients[conn] = true
	defer delete(s.clients, conn)

	// Send initial status
	status := map[string]interface{}{
		"type": "status",
		"data": map[string]bool{
			"embedding": embeddings.Get() != nil,
			"vectordb":  vectordb.Get() != nil,
			"lsp":       lsp.GetManager() != nil,
			"mcp":       true, // MCP server is available as standalone
		},
	}
	conn.WriteJSON(status)

	// Keep connection alive and handle messages
	for {
		var msg map[string]interface{}
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}

		// Echo back for now (can be extended for real-time features)
		conn.WriteJSON(map[string]interface{}{
			"type": "echo",
			"data": msg,
		})
	}
}

// sendSuccess sends a successful API response
func (s *Server) sendSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    data,
	})
}

// sendError sends an error API response
func (s *Server) sendError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(APIResponse{
		Success: false,
		Error:   message,
	})
}

// Start starts the web server
func (s *Server) Start(port int) error {
	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("🌐 Web interface starting on http://localhost%s\n", addr)
	return http.ListenAndServe(addr, s.router)
}

// convertToMCPServerConfig converts a map to MCPServerConfig
func (s *Server) convertToMCPServerConfig(config map[string]interface{}) (*mcp.MCPServerConfig, error) {
	// Extract required fields
	name, ok := config["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("name is required and must be a string")
	}

	serverType, ok := config["type"].(string)
	if !ok || serverType == "" {
		return nil, fmt.Errorf("type is required and must be a string")
	}

	// Validate server type
	var mcpType mcp.MCPServerType
	switch serverType {
	case "local":
		mcpType = mcp.MCPServerTypeLocal
	case "remote", "http":
		mcpType = mcp.MCPServerTypeRemote
	case "sse":
		mcpType = mcp.MCPServerTypeSSE
	default:
		return nil, fmt.Errorf("invalid server type: %s", serverType)
	}

	// Extract optional fields with defaults
	description, _ := config["description"].(string)
	enabled, _ := config["enabled"].(bool)

	// Extract command for local servers
	var command []string
	if cmdInterface, exists := config["command"]; exists {
		if cmdSlice, ok := cmdInterface.([]interface{}); ok {
			for _, cmd := range cmdSlice {
				if cmdStr, ok := cmd.(string); ok {
					command = append(command, cmdStr)
				}
			}
		} else if cmdStr, ok := cmdInterface.(string); ok {
			command = []string{cmdStr}
		}
	}

	// Extract URL for remote servers
	url, _ := config["url"].(string)

	// Extract environment variables
	environment := make(map[string]string)
	if envInterface, exists := config["environment"]; exists {
		if envMap, ok := envInterface.(map[string]interface{}); ok {
			for k, v := range envMap {
				if vStr, ok := v.(string); ok {
					environment[k] = vStr
				}
			}
		}
	}

	// Extract capability flags with defaults
	tools := true
	if toolsInterface, exists := config["tools"]; exists {
		if toolsBool, ok := toolsInterface.(bool); ok {
			tools = toolsBool
		}
	}

	resources := true
	if resourcesInterface, exists := config["resources"]; exists {
		if resourcesBool, ok := resourcesInterface.(bool); ok {
			resources = resourcesBool
		}
	}

	prompts := true
	if promptsInterface, exists := config["prompts"]; exists {
		if promptsBool, ok := promptsInterface.(bool); ok {
			prompts = promptsBool
		}
	}

	// Extract timeout with default
	timeout := 30 * time.Second
	if timeoutInterface, exists := config["timeout"]; exists {
		if timeoutFloat, ok := timeoutInterface.(float64); ok {
			timeout = time.Duration(timeoutFloat) * time.Second
		} else if timeoutInt, ok := timeoutInterface.(int); ok {
			timeout = time.Duration(timeoutInt) * time.Second
		}
	}

	return &mcp.MCPServerConfig{
		Name:        name,
		Type:        mcpType,
		Description: description,
		Enabled:     enabled,
		Command:     command,
		Args:        []string{}, // Could be extracted from config if needed
		URL:         url,
		Environment: environment,
		Tools:       tools,
		Resources:   resources,
		Prompts:     prompts,
		Timeout:     timeout,
	}, nil
}
