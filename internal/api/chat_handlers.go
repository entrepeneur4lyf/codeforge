package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/chat"
	"github.com/entrepeneur4lyf/codeforge/internal/markdown"
	"github.com/gorilla/mux"
)

// ChatSession represents a chat session
type ChatSession struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Model     string    `json:"model"`
	Provider  string    `json:"provider"`
}

// ChatMessage represents a chat message
type ChatMessage struct {
	ID        string                 `json:"id"`
	SessionID string                 `json:"session_id"`
	Role      string                 `json:"role"` // "user", "assistant", "system"
	Content   string                 `json:"content"`
	Timestamp time.Time              `json:"timestamp"`
	Model     string                 `json:"model,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// EnhancedChatMessage represents a chat message with multiple format support
type EnhancedChatMessage struct {
	ID        string                 `json:"id"`
	SessionID string                 `json:"session_id"`
	Role      string                 `json:"role"`
	Content   map[string]string      `json:"content"` // Format -> rendered content
	Timestamp time.Time              `json:"timestamp"`
	Model     string                 `json:"model,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ChatRequest represents a new chat message request
type ChatRequest struct {
	Message  string                 `json:"message"`
	Model    string                 `json:"model,omitempty"`
	Provider string                 `json:"provider,omitempty"`
	Context  map[string]interface{} `json:"context,omitempty"`
}

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type    string      `json:"type"`
	Data    interface{} `json:"data"`
	Error   string      `json:"error,omitempty"`
	EventID string      `json:"event_id,omitempty"`
}

// ChatStorage manages chat sessions and messages in memory
type ChatStorage struct {
	sessions map[string]*ChatSession
	messages map[string][]ChatMessage
	mu       sync.RWMutex
}

// NewChatStorage creates a new chat storage instance
func NewChatStorage() *ChatStorage {
	return &ChatStorage{
		sessions: make(map[string]*ChatSession),
		messages: make(map[string][]ChatMessage),
	}
}

// CreateSession creates a new chat session
func (cs *ChatStorage) CreateSession(title string) *ChatSession {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	sessionID := fmt.Sprintf("session-%d", time.Now().Unix())
	session := &ChatSession{
		ID:        sessionID,
		Title:     title,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    "active",
	}

	cs.sessions[sessionID] = session
	cs.messages[sessionID] = []ChatMessage{}
	return session
}

// GetSession retrieves a session by ID
func (cs *ChatStorage) GetSession(sessionID string) (*ChatSession, bool) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	session, exists := cs.sessions[sessionID]
	return session, exists
}

// GetAllSessions returns all sessions
func (cs *ChatStorage) GetAllSessions() []*ChatSession {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	sessions := make([]*ChatSession, 0, len(cs.sessions))
	for _, session := range cs.sessions {
		sessions = append(sessions, session)
	}
	return sessions
}

// DeleteSession removes a session and its messages
func (cs *ChatStorage) DeleteSession(sessionID string) bool {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if _, exists := cs.sessions[sessionID]; !exists {
		return false
	}

	delete(cs.sessions, sessionID)
	delete(cs.messages, sessionID)
	return true
}

// AddMessage adds a message to a session
func (cs *ChatStorage) AddMessage(sessionID string, message ChatMessage) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if _, exists := cs.sessions[sessionID]; !exists {
		return fmt.Errorf("session not found")
	}

	message.ID = fmt.Sprintf("msg-%d", time.Now().UnixNano())
	message.Timestamp = time.Now()

	cs.messages[sessionID] = append(cs.messages[sessionID], message)

	// Update session timestamp
	cs.sessions[sessionID].UpdatedAt = time.Now()

	return nil
}

// GetMessages retrieves all messages for a session
func (cs *ChatStorage) GetMessages(sessionID string) ([]ChatMessage, error) {
	cs.mu.RLock()
	defer cs.mu.RUnlock()

	if _, exists := cs.sessions[sessionID]; !exists {
		return nil, fmt.Errorf("session not found")
	}

	messages := cs.messages[sessionID]
	result := make([]ChatMessage, len(messages))
	copy(result, messages)
	return result, nil
}

// createLLMChatSession creates a real LLM chat session with proper API key integration
func (s *Server) createLLMChatSession(model string) (*chat.ChatSession, error) {
	// Get API key for the model using the chat module's logic
	apiKey := chat.GetAPIKeyForModel(model)
	if apiKey == "" {
		return nil, fmt.Errorf("no API key found for model: %s", model)
	}

	// Create chat session using the proper chat module
	session, err := chat.NewChatSession(model, apiKey, "", true, "text")
	if err != nil {
		return nil, fmt.Errorf("failed to create chat session: %w", err)
	}

	return session, nil
}

// handleChatSessions handles GET /chat/sessions and POST /chat/sessions
func (s *Server) handleChatSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.getChatSessions(w, r)
	case "POST":
		s.createChatSession(w, r)
	}
}

// getChatSessions returns all chat sessions
func (s *Server) getChatSessions(w http.ResponseWriter, r *http.Request) {
	sessions := s.chatStorage.GetAllSessions()

	s.writeJSON(w, map[string]interface{}{
		"sessions": sessions,
		"total":    len(sessions),
	})
}

// createChatSession creates a new chat session
func (s *Server) createChatSession(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title    string `json:"title"`
		Model    string `json:"model,omitempty"`
		Provider string `json:"provider,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Title == "" {
		req.Title = "New Chat Session"
	}

	session := s.chatStorage.CreateSession(req.Title)
	session.Model = req.Model
	session.Provider = req.Provider

	w.WriteHeader(http.StatusCreated)
	s.writeJSON(w, session)
}

// handleChatSession handles GET /chat/sessions/{id} and DELETE /chat/sessions/{id}
func (s *Server) handleChatSession(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	switch r.Method {
	case "GET":
		s.getChatSession(w, r, sessionID)
	case "DELETE":
		s.deleteChatSession(w, r, sessionID)
	}
}

// getChatSession returns a specific chat session
func (s *Server) getChatSession(w http.ResponseWriter, r *http.Request, sessionID string) {
	session, exists := s.chatStorage.GetSession(sessionID)
	if !exists {
		s.writeError(w, "Session not found", http.StatusNotFound)
		return
	}

	s.writeJSON(w, session)
}

// deleteChatSession deletes a chat session
func (s *Server) deleteChatSession(w http.ResponseWriter, r *http.Request, sessionID string) {
	// Implement actual session deletion using ChatStorage
	if !s.chatStorage.DeleteSession(sessionID) {
		s.writeError(w, "Session not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleChatMessages handles GET /chat/sessions/{id}/messages and POST /chat/sessions/{id}/messages
func (s *Server) handleChatMessages(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	sessionID := vars["id"]

	switch r.Method {
	case "GET":
		s.getChatMessages(w, r, sessionID)
	case "POST":
		s.sendChatMessage(w, r, sessionID)
	}
}

// getChatMessages returns messages for a session
func (s *Server) getChatMessages(w http.ResponseWriter, r *http.Request, sessionID string) {
	// Implement actual message retrieval using ChatStorage
	messages, err := s.chatStorage.GetMessages(sessionID)
	if err != nil {
		s.writeError(w, err.Error(), http.StatusNotFound)
		return
	}

	s.writeJSON(w, map[string]interface{}{
		"messages": messages,
		"total":    len(messages),
	})
}

// sendChatMessage sends a new message in a session
func (s *Server) sendChatMessage(w http.ResponseWriter, r *http.Request, sessionID string) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create user message
	userMessage := ChatMessage{
		ID:        generateMessageID(),
		SessionID: sessionID,
		Role:      "user",
		Content:   req.Message,
		Timestamp: time.Now(),
		Metadata:  req.Context,
	}

	// Store user message
	s.chatStorage.AddMessage(sessionID, userMessage)

	// Get session to determine model
	session, exists := s.chatStorage.GetSession(sessionID)
	if !exists {
		s.writeError(w, "Session not found", http.StatusNotFound)
		return
	}

	// Use default model if none specified
	model := session.Model
	if model == "" {
		model = chat.GetDefaultModel()
	}

	// Create LLM chat session
	llmSession, err := s.createLLMChatSession(model)
	if err != nil {
		s.writeError(w, fmt.Sprintf("Failed to create LLM session: %v", err), http.StatusInternalServerError)
		return
	}

	// Process message with integrated CodeForge app if available
	var response string
	if s.app != nil {
		ctx := r.Context()
		appResponse, err := s.app.ProcessChatMessage(ctx, sessionID, req.Message, model)
		if err != nil {
			log.Printf("Error processing chat message with app: %v", err)
			// Fallback to LLM session
			response, err = llmSession.ProcessMessage(req.Message)
			if err != nil {
				s.writeError(w, fmt.Sprintf("Failed to process message: %v", err), http.StatusInternalServerError)
				return
			}
		} else {
			response = appResponse
		}
	} else {
		// Use LLM session directly
		response, err = llmSession.ProcessMessage(req.Message)
		if err != nil {
			s.writeError(w, fmt.Sprintf("Failed to process message: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Create assistant message
	assistantMessage := ChatMessage{
		ID:        generateMessageID(),
		SessionID: sessionID,
		Role:      "assistant",
		Content:   response,
		Timestamp: time.Now(),
		Model:     model,
	}

	// Store assistant message
	s.chatStorage.AddMessage(sessionID, assistantMessage)

	// Return assistant response
	s.writeJSON(w, assistantMessage)
}

// sendChatMessageEnhanced handles enhanced chat messages with markdown support
func (s *Server) sendChatMessageEnhanced(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		s.writeError(w, "Message is required", http.StatusBadRequest)
		return
	}

	// Get session ID from URL
	vars := mux.Vars(r)
	sessionID := vars["sessionID"]

	// Get or create session
	session, exists := s.chatStorage.GetSession(sessionID)
	if !exists {
		// Create new session using CreateSession method
		session = s.chatStorage.CreateSession("Enhanced Chat Session")
		session.ID = sessionID // Override the generated ID
		session.Model = req.Model
		session.Provider = req.Provider
	}

	// Store user message
	userMessage := ChatMessage{
		ID:        generateMessageID(),
		SessionID: sessionID,
		Role:      "user",
		Content:   req.Message,
		Timestamp: time.Now(),
		Model:     req.Model,
		Metadata:  req.Context,
	}
	s.chatStorage.AddMessage(sessionID, userMessage)

	// Process with AI (using CodeForge app integration)
	var responseContent string
	if s.app != nil {
		ctx := r.Context()
		modelID := req.Model
		if modelID == "" {
			modelID = session.Model
		}
		if modelID == "" {
			modelID = "default"
		}

		response, err := s.app.ProcessChatMessage(ctx, sessionID, req.Message, modelID)
		if err != nil {
			log.Printf("Chat processing error: %v", err)
			s.writeError(w, "Failed to process message", http.StatusInternalServerError)
			return
		}
		responseContent = response
	} else {
		responseContent = "Chat processing is not available - app not initialized"
	}

	// Process response with markdown support
	enhancedResponse, err := s.processMessageWithMarkdown(responseContent, sessionID, req.Model)
	if err != nil {
		log.Printf("Markdown processing error: %v", err)
		// Fallback to regular response
		assistantMessage := ChatMessage{
			ID:        generateMessageID(),
			SessionID: sessionID,
			Role:      "assistant",
			Content:   responseContent,
			Timestamp: time.Now(),
			Model:     req.Model,
			Metadata: map[string]interface{}{
				"markdown": false,
				"via":      "rest_api",
			},
		}
		s.chatStorage.AddMessage(sessionID, assistantMessage)
		s.writeJSON(w, assistantMessage)
		return
	}

	// Convert enhanced message to regular message for storage
	assistantMessage := ChatMessage{
		ID:        enhancedResponse.ID,
		SessionID: enhancedResponse.SessionID,
		Role:      enhancedResponse.Role,
		Content:   enhancedResponse.Content["plain"], // Store plain text version
		Timestamp: enhancedResponse.Timestamp,
		Model:     enhancedResponse.Model,
		Metadata:  enhancedResponse.Metadata,
	}
	s.chatStorage.AddMessage(sessionID, assistantMessage)

	// Return enhanced response with all formats
	s.writeJSON(w, enhancedResponse)
}

// generateSessionID generates a unique session ID
func generateSessionID() string {
	return "session-" + time.Now().Format("20060102-150405")
}

// processMessageWithMarkdown processes a chat message with markdown support
func (s *Server) processMessageWithMarkdown(content string, sessionID string, model string) (*EnhancedChatMessage, error) {
	// Create markdown processor
	processor, err := markdown.NewMessageProcessor()
	if err != nil {
		return nil, fmt.Errorf("failed to create markdown processor: %w", err)
	}

	// Process the content into multiple formats
	processedContent, err := processor.ProcessMessage(content)
	if err != nil {
		return nil, fmt.Errorf("failed to process markdown: %w", err)
	}

	// Create enhanced chat message
	response := &EnhancedChatMessage{
		ID:        generateMessageID(),
		SessionID: sessionID,
		Role:      "assistant",
		Content:   processedContent,
		Timestamp: time.Now(),
		Model:     model,
		Metadata: map[string]interface{}{
			"markdown":          true,
			"available_formats": []string{"plain", "markdown", "terminal", "html"},
			"via":               "rest_api",
		},
	}

	return response, nil
}

// generateMessageID generates a unique message ID
func generateMessageID() string {
	return "msg-" + time.Now().Format("20060102-150405-000")
}
