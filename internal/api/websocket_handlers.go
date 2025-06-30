package api

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/events"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// ChatWebSocketClient represents a WebSocket client
type ChatWebSocketClient struct {
	conn      *websocket.Conn
	sessionID string
	send      chan WebSocketMessage
	server    *Server
	cancel    context.CancelFunc
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

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Create client
	client := &ChatWebSocketClient{
		conn:      conn,
		sessionID: sessionID,
		send:      make(chan WebSocketMessage, 256),
		server:    s,
		cancel:    cancel,
	}

	log.Printf("🔌 WebSocket client connected for session: %s", sessionID)

	// Register connection with connection manager
	s.connectionManager.AddChatConnection(sessionID, client)

	// Replay missed events for offline users
	go client.replayMissedEvents()

	// Start client goroutines
	go client.writePump()
	go client.readPump()
	go client.subscribeToEvents(ctx)
}

// readPump handles incoming WebSocket messages
func (c *ChatWebSocketClient) readPump() {
	defer func() {
		c.cancel()
		c.server.connectionManager.RemoveChatConnection(c.sessionID, c)
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

	// Integrate with actual chat engine
	// Get session to determine model and provider
	session, exists := c.server.chatStorage.GetSession(c.sessionID)
	if !exists {
		c.sendMessage(WebSocketMessage{
			Type:  "error",
			Error: "Session not found",
		})
		return
	}

	// Use the integrated CodeForge app for real chat processing
	var responseContent string
	if c.server.app != nil {
		ctx := context.Background()
		modelID := session.Model
		if modelID == "" {
			modelID = "default"
		}

		response, err := c.server.app.ProcessChatMessage(ctx, c.sessionID, message, modelID)
		if err != nil {
			log.Printf("Chat processing error: %v", err)
			responseContent = "I apologize, but I encountered an error processing your message. Please try again."
		} else {
			responseContent = response
		}
	} else {
		responseContent = "Chat processing is not available - app not initialized"
	}

	// Create assistant response
	response := ChatMessage{
		ID:        generateMessageID(),
		SessionID: c.sessionID,
		Role:      "assistant",
		Content:   responseContent,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"model":    session.Model,
			"provider": session.Provider,
			"via":      "websocket",
		},
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
	// For now, just log the event - in production this would broadcast to other connected clients
	log.Printf("User started typing in session: %s", c.sessionID)

	// Send acknowledgment back to the client
	c.sendMessage(WebSocketMessage{
		Type: "typing_ack",
		Data: map[string]interface{}{
			"session_id": c.sessionID,
			"status":     "typing_started",
		},
	})
}

// handleTypingStop handles typing stop events
func (c *ChatWebSocketClient) handleTypingStop(msg WebSocketMessage) {
	// Broadcast typing stop to other clients in the same session
	// For now, just log the event - in production this would broadcast to other connected clients
	log.Printf("User stopped typing in session: %s", c.sessionID)

	// Send acknowledgment back to the client
	c.sendMessage(WebSocketMessage{
		Type: "typing_ack",
		Data: map[string]interface{}{
			"session_id": c.sessionID,
			"status":     "typing_stopped",
		},
	})
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

// subscribeToEvents subscribes to chat and system events for this session
func (c *ChatWebSocketClient) subscribeToEvents(ctx context.Context) {
	if c.server.app == nil || c.server.app.EventManager == nil {
		log.Printf("Event manager not available for chat event subscription")
		return
	}

	// Subscribe to chat events for this session
	chatCh := c.server.app.EventManager.SubscribeChat(ctx,
		events.FilterBySessionID(c.sessionID))

	// Subscribe to system events (no session filter for system-wide events)
	systemCh := c.server.app.EventManager.SubscribeSystem(ctx)

	for {
		select {
		case chatEvent, ok := <-chatCh:
			if !ok {
				return
			}

			// Convert chat event to WebSocket message
			wsMsg := WebSocketMessage{
				Type:    "chat_event",
				EventID: chatEvent.ID,
				Data: map[string]interface{}{
					"event_type": chatEvent.Type,
					"payload":    chatEvent.Payload,
					"timestamp":  chatEvent.Timestamp.Unix(),
					"session_id": chatEvent.SessionID,
				},
			}

			// Send to WebSocket client
			select {
			case c.send <- wsMsg:
			default:
				log.Printf("Chat event channel full for session %s", c.sessionID)
			}

		case systemEvent, ok := <-systemCh:
			if !ok {
				return
			}

			// Convert system event to WebSocket message
			wsMsg := WebSocketMessage{
				Type:    "system_event",
				EventID: systemEvent.ID,
				Data: map[string]interface{}{
					"event_type": systemEvent.Type,
					"payload":    systemEvent.Payload,
					"timestamp":  systemEvent.Timestamp.Unix(),
					"user_id":    systemEvent.UserID,
				},
			}

			// Send to WebSocket client
			select {
			case c.send <- wsMsg:
			default:
				log.Printf("System event channel full for session %s", c.sessionID)
			}

		case <-ctx.Done():
			return
		}
	}
}

// replayMissedEvents replays events that occurred while the client was offline
func (c *ChatWebSocketClient) replayMissedEvents() {
	if c.server.app == nil || c.server.app.EventManager == nil {
		return
	}

	// Get the last 24 hours of events for this session
	since := time.Now().Add(-24 * time.Hour)

	// Get missed chat events
	if chatEvents, err := c.server.app.EventManager.GetEventsForSession(c.sessionID, since); err == nil {
		for _, event := range chatEvents {
			// Only replay certain event types to avoid spam
			if event.Type == events.ChatMessageSent || event.Type == events.ChatMessageReceived {
				wsMsg := WebSocketMessage{
					Type:    "replay_event",
					EventID: event.ID,
					Data: map[string]interface{}{
						"event_type": event.Type,
						"payload":    event.Payload,
						"timestamp":  event.Timestamp.Unix(),
						"session_id": event.SessionID,
						"replayed":   true,
					},
				}

				// Send replayed event
				select {
				case c.send <- wsMsg:
				default:
					// Channel full, skip this event
					log.Printf("Skipping replay event for session %s - channel full", c.sessionID)
					return
				}
			}
		}
	}

	// Send replay complete notification
	c.sendMessage(WebSocketMessage{
		Type: "replay_complete",
		Data: map[string]interface{}{
			"session_id": c.sessionID,
			"timestamp":  time.Now().Unix(),
		},
	})
}

// NotificationWebSocketClient represents a WebSocket client for notifications
type NotificationWebSocketClient struct {
	conn      *websocket.Conn
	sessionID string
	send      chan interface{}
	server    *Server
	cancel    context.CancelFunc
}

// handleNotificationWebSocket handles WebSocket connections for real-time notifications
func (s *Server) handleNotificationWebSocket(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["sessionId"]

	// Upgrade HTTP connection to WebSocket
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Notification WebSocket upgrade failed: %v", err)
		return
	}

	// Create context for cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Create client
	client := &NotificationWebSocketClient{
		conn:      conn,
		sessionID: sessionID,
		send:      make(chan interface{}, 256),
		server:    s,
		cancel:    cancel,
	}

	log.Printf("🔔 Notification WebSocket client connected for session: %s", sessionID)

	// Register connection with connection manager
	s.connectionManager.AddNotificationConnection(sessionID, client)

	// Start client goroutines
	go client.writePump()
	go client.readPump()
	go client.subscribeToNotifications(ctx)
}

// subscribeToNotifications subscribes to notification events for this session
func (c *NotificationWebSocketClient) subscribeToNotifications(ctx context.Context) {
	if c.server.app == nil || c.server.app.EventManager == nil {
		log.Printf("Event manager not available for notification subscription")
		return
	}

	// Subscribe to notification events for this session
	notificationCh := c.server.app.EventManager.SubscribeNotification(ctx,
		events.FilterBySessionID(c.sessionID))

	for {
		select {
		case notification, ok := <-notificationCh:
			if !ok {
				return
			}

			// Send notification to WebSocket client
			select {
			case c.send <- notification:
			default:
				log.Printf("Notification channel full for session %s", c.sessionID)
			}

		case <-ctx.Done():
			return
		}
	}
}

// readPump handles incoming WebSocket messages for notifications
func (c *NotificationWebSocketClient) readPump() {
	defer func() {
		c.cancel()
		c.server.connectionManager.RemoveNotificationConnection(c.sessionID, c)
		c.conn.Close()
		log.Printf("🔔 Notification WebSocket client disconnected for session: %s", c.sessionID)
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Notification WebSocket error: %v", err)
			}
			break
		}
	}
}

// writePump handles outgoing WebSocket messages for notifications
func (c *NotificationWebSocketClient) writePump() {
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
				log.Printf("Notification WebSocket write error: %v", err)
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
