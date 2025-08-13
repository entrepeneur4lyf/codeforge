package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"go.lsp.dev/protocol"
)

// NewRequest creates a new JSON-RPC request message
func NewRequest(id int32, method string, params any) (*Message, error) {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	return &Message{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  paramsJSON,
	}, nil
}

// NewNotification creates a new JSON-RPC notification message
func NewNotification(method string, params any) (*Message, error) {
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	return &Message{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsJSON,
	}, nil
}

// WriteMessage writes a JSON-RPC message to the writer
func WriteMessage(w io.Writer, msg *Message) error {
    if w == nil {
        return fmt.Errorf("writer is nil")
    }
    data, err := json.Marshal(msg)
    if err != nil {
        return err
    }

    content := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(data), data)
    _, err = w.Write([]byte(content))
    return err
}

// ReadMessage reads a JSON-RPC message from a reader (following OpenCode's pattern)
func ReadMessage(r io.Reader) (*Message, error) {
	scanner := bufio.NewScanner(r)

	// Read headers
	headers := make(map[string]string)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break // End of headers
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read headers: %w", err)
	}

	// Get content length
	contentLengthStr, exists := headers["Content-Length"]
	if !exists {
		return nil, fmt.Errorf("missing Content-Length header")
	}

	contentLength, err := strconv.Atoi(contentLengthStr)
	if err != nil {
		return nil, fmt.Errorf("invalid Content-Length: %w", err)
	}

	// Read content
	content := make([]byte, contentLength)
	_, err = io.ReadFull(r, content)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	// Parse message
	var msg Message
	if err := json.Unmarshal(content, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	return &msg, nil
}

// readMessage reads a JSON-RPC message from the LSP server
func (c *Client) readMessage() (*Message, error) {
	return ReadMessage(c.stdout)
}

// Call makes a request and waits for the response
func (c *Client) Call(ctx context.Context, method string, params any, result any) error {
	id := c.nextID.Add(1)

	msg, err := NewRequest(id, method, params)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Create response channel
	ch := make(chan *Message, 1)
	c.handlersMu.Lock()
	c.handlers[id] = ch
	c.handlersMu.Unlock()

	defer func() {
		c.handlersMu.Lock()
		delete(c.handlers, id)
		c.handlersMu.Unlock()
	}()

    // Send request
    if c.stdin == nil {
        return fmt.Errorf("LSP server not initialized")
    }
    if err := WriteMessage(c.stdin, msg); err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	// Wait for response
	select {
	case response := <-ch:
		if response.Error != nil {
			return fmt.Errorf("LSP error: %s", response.Error.Message)
		}

		if result != nil && response.Result != nil {
			if err := json.Unmarshal(response.Result, result); err != nil {
				return fmt.Errorf("failed to unmarshal result: %w", err)
			}
		}

		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Notify sends a notification (no response expected)
func (c *Client) Notify(ctx context.Context, method string, params any) error {
	msg, err := NewNotification(method, params)
	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}

    if c.stdin == nil {
        return fmt.Errorf("LSP server not initialized")
    }
    return WriteMessage(c.stdin, msg)
}

// InitializeLSPClient initializes the LSP client with the server
func (c *Client) InitializeLSPClient(ctx context.Context, workspaceDir string) (*protocol.InitializeResult, error) {
	initParams := &protocol.InitializeParams{
		ProcessID: int32(os.Getpid()),
		ClientInfo: &protocol.ClientInfo{
			Name:    "codeforge",
			Version: "0.1.0",
		},
		RootURI: protocol.DocumentURI("file://" + workspaceDir),
		Capabilities: protocol.ClientCapabilities{
			TextDocument: &protocol.TextDocumentClientCapabilities{
				PublishDiagnostics: &protocol.PublishDiagnosticsClientCapabilities{
					RelatedInformation: true,
				},
			},
			Workspace: &protocol.WorkspaceClientCapabilities{
				ApplyEdit: true,
			},
		},
	}

	var result protocol.InitializeResult
	if err := c.Call(ctx, "initialize", initParams, &result); err != nil {
		return nil, fmt.Errorf("initialize failed: %w", err)
	}

	if err := c.Notify(ctx, "initialized", struct{}{}); err != nil {
		return nil, fmt.Errorf("initialized notification failed: %w", err)
	}

	// Register default handlers
	c.RegisterNotificationHandler("textDocument/publishDiagnostics", c.handleDiagnostics)
	c.RegisterNotificationHandler("window/showMessage", c.handleShowMessage)

	return &result, nil
}

// handleNotification handles server notifications
func (c *Client) handleNotification(method string, params json.RawMessage) {
	c.notificationMu.RLock()
	handler, exists := c.notificationHandlers[method]
	c.notificationMu.RUnlock()

	if exists {
		handler(params)
	}
}

// handleServerRequest handles server requests
func (c *Client) handleServerRequest(method string, params json.RawMessage) {
	c.serverHandlersMu.RLock()
	handler, exists := c.serverRequestHandlers[method]
	c.serverHandlersMu.RUnlock()

	if exists {
		_, _ = handler(params)
	}
}

// RegisterNotificationHandler registers a handler for server notifications
func (c *Client) RegisterNotificationHandler(method string, handler NotificationHandler) {
	c.notificationMu.Lock()
	defer c.notificationMu.Unlock()
	c.notificationHandlers[method] = handler
}

// RegisterServerRequestHandler registers a handler for server requests
func (c *Client) RegisterServerRequestHandler(method string, handler ServerRequestHandler) {
	c.serverHandlersMu.Lock()
	defer c.serverHandlersMu.Unlock()
	c.serverRequestHandlers[method] = handler
}
