package api

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// ChatWebSocketClient represents a WebSocket client
type ChatWebSocketClient struct {
	conn      *websocket.Conn
	sessionID string
	send      chan WebSocketMessage
	server    *Server
}

// handleChatWebSocket handles WebSocket connections for real-time chat
func (s *Server) handleChatWebSocket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionId"]

	// Upgrade HTTP connection to WebSocket
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Create client
	client := &ChatWebSocketClient{
		conn:      conn,
		sessionID: sessionID,
		send:      make(chan WebSocketMessage, 256),
		server:    s,
	}

	log.Printf("🔌 WebSocket client connected for session: %s", sessionID)

	// Start client goroutines
	go client.writePump()
	go client.readPump()
}

// readPump handles incoming WebSocket messages
func (c *ChatWebSocketClient) readPump() {
	defer func() {
		c.conn.Close()
		log.Printf("🔌 WebSocket client disconnected for session: %s", c.sessionID)
	}()

	// Set read deadline and pong handler
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg WebSocketMessage
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle different message types
		c.handleMessage(msg)
	}
}

// writePump handles outgoing WebSocket messages
func (c *ChatWebSocketClient) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteJSON(message); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming WebSocket messages
func (c *ChatWebSocketClient) handleMessage(msg WebSocketMessage) {
	switch msg.Type {
	case "chat_message":
		c.handleChatMessage(msg)
	case "typing_start":
		c.handleTypingStart(msg)
	case "typing_stop":
		c.handleTypingStop(msg)
	case "ping":
		c.sendMessage(WebSocketMessage{Type: "pong", EventID: msg.EventID})
	default:
		c.sendError("Unknown message type", msg.EventID)
	}
}

// handleChatMessage processes chat messages
func (c *ChatWebSocketClient) handleChatMessage(msg WebSocketMessage) {
	// Parse chat request
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		c.sendError("Invalid message data", msg.EventID)
		return
	}

	message, ok := data["message"].(string)
	if !ok {
		c.sendError("Missing message content", msg.EventID)
		return
	}

	// Send acknowledgment
	c.sendMessage(WebSocketMessage{
		Type:    "message_received",
		EventID: msg.EventID,
		Data: map[string]interface{}{
			"message_id": generateMessageID(),
			"timestamp":  time.Now(),
		},
	})

	// Process message asynchronously
	go c.processMessage(message, msg.EventID)
}

// processMessage handles AI response generation
func (c *ChatWebSocketClient) processMessage(message string, eventID string) {
	// Send typing indicator
	c.sendMessage(WebSocketMessage{
		Type: "assistant_typing",
		Data: map[string]interface{}{
			"typing": true,
		},
	})

	// TODO: Integrate with actual chat engine
	// For now, simulate AI response
	time.Sleep(2 * time.Second)

	// Send AI response
	response := ChatMessage{
		ID:        generateMessageID(),
		SessionID: c.sessionID,
		Role:      "assistant",
		Content:   "I received your message: " + message,
		Timestamp: time.Now(),
	}

	c.sendMessage(WebSocketMessage{
		Type:    "chat_response",
		EventID: eventID,
		Data:    response,
	})

	// Stop typing indicator
	c.sendMessage(WebSocketMessage{
		Type: "assistant_typing",
		Data: map[string]interface{}{
			"typing": false,
		},
	})
}

// handleTypingStart handles typing start events
func (c *ChatWebSocketClient) handleTypingStart(msg WebSocketMessage) {
	// Broadcast typing indicator to other clients in the same session
	// TODO: Implement session-based broadcasting
	log.Printf("User started typing in session: %s", c.sessionID)
}

// handleTypingStop handles typing stop events
func (c *ChatWebSocketClient) handleTypingStop(msg WebSocketMessage) {
	// Broadcast typing stop to other clients in the same session
	// TODO: Implement session-based broadcasting
	log.Printf("User stopped typing in session: %s", c.sessionID)
}

// sendMessage sends a message to the WebSocket client
func (c *ChatWebSocketClient) sendMessage(msg WebSocketMessage) {
	select {
	case c.send <- msg:
	default:
		close(c.send)
	}
}

// sendError sends an error message to the WebSocket client
func (c *ChatWebSocketClient) sendError(error string, eventID string) {
	c.sendMessage(WebSocketMessage{
		Type:    "error",
		Error:   error,
		EventID: eventID,
	})
}
