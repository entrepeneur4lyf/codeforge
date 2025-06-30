package agent

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/config"
	"github.com/entrepeneur4lyf/codeforge/internal/events"
	"github.com/entrepeneur4lyf/codeforge/internal/llm"
	"github.com/entrepeneur4lyf/codeforge/internal/llm/prompt"
	"github.com/entrepeneur4lyf/codeforge/internal/llm/providers"
	"github.com/entrepeneur4lyf/codeforge/internal/models"
	"github.com/google/uuid"
)

// AgentEvent represents events emitted by the agent service
type AgentEvent struct {
	ID        string                 `json:"id"`
	Type      AgentEventType         `json:"type"`
	SessionID string                 `json:"session_id"`
	AgentName config.AgentName       `json:"agent_name"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// AgentEventType represents different types of agent events
type AgentEventType string

const (
	AgentEventStarted    AgentEventType = "agent_started"
	AgentEventCompleted  AgentEventType = "agent_completed"
	AgentEventError      AgentEventType = "agent_error"
	AgentEventCancelled  AgentEventType = "agent_cancelled"
	AgentEventTextChunk  AgentEventType = "agent_text_chunk"
	AgentEventToolCall   AgentEventType = "agent_tool_call"
	AgentEventToolResult AgentEventType = "agent_tool_result"
	AgentEventUsage      AgentEventType = "agent_usage"
)

// Service defines the agent service interface
type Service interface {
	// Run executes an agent with the given configuration
	Run(ctx context.Context, sessionID string, agentName config.AgentName, content string, attachments ...llm.ContentBlock) (<-chan AgentEvent, error)

	// Cancel cancels a running agent session
	Cancel(sessionID string)

	// IsSessionBusy checks if a session is currently running an agent
	IsSessionBusy(sessionID string) bool

	// IsBusy checks if any agent is currently running
	IsBusy() bool

	// UpdateAgent updates the configuration for a specific agent
	UpdateAgent(agentName config.AgentName, modelID models.ModelID) (models.Model, error)

	// GetModel returns the current model for an agent
	GetModel(agentName config.AgentName) (models.Model, error)

	// Subscribe subscribes to agent events
	Subscribe(ctx context.Context, sessionID string) <-chan AgentEvent
}

// AgentService implements the Service interface
type AgentService struct {
	config       *config.Config
	eventManager *events.Manager

	// Session management
	activeSessions map[string]*AgentSession
	sessionMutex   sync.RWMutex

	// Event broadcasting
	eventBroker *events.Broker[AgentEvent]

	// Agent configurations
	agentConfigs map[config.AgentName]AgentConfig
	configMutex  sync.RWMutex
}

// AgentConfig holds configuration for a specific agent
type AgentConfig struct {
	Agent    config.Agent
	Model    models.Model
	Handler  llm.ApiHandler
	Provider models.ModelProvider
}

// AgentSession represents an active agent session
type AgentSession struct {
	ID        string
	AgentName config.AgentName
	StartTime time.Time
	Cancel    context.CancelFunc
	EventChan chan AgentEvent
}

// NewAgentService creates a new agent service
func NewAgentService(cfg *config.Config, eventManager *events.Manager) (*AgentService, error) {
	service := &AgentService{
		config:         cfg,
		eventManager:   eventManager,
		activeSessions: make(map[string]*AgentSession),
		eventBroker:    events.NewBroker[AgentEvent](),
		agentConfigs:   make(map[config.AgentName]AgentConfig),
	}

	// Initialize agent configurations
	if err := service.initializeAgents(); err != nil {
		return nil, fmt.Errorf("failed to initialize agents: %w", err)
	}

	return service, nil
}

// initializeAgents initializes all configured agents
func (s *AgentService) initializeAgents() error {
	s.configMutex.Lock()
	defer s.configMutex.Unlock()

	for agentName, agentCfg := range s.config.Agents {
		model, exists := models.GetModel(agentCfg.Model)
		if !exists {
			log.Printf("Warning: Model %s not found for agent %s", agentCfg.Model, agentName)
			continue
		}

		// Create handler for this agent
		handler, err := s.createHandlerForModel(model)
		if err != nil {
			log.Printf("Warning: Failed to create handler for agent %s: %v", agentName, err)
			continue
		}

		s.agentConfigs[agentName] = AgentConfig{
			Agent:    agentCfg,
			Model:    model,
			Handler:  handler,
			Provider: model.Provider,
		}

		log.Printf("✅ Initialized agent %s with model %s", agentName, model.Name)
	}

	return nil
}

// createHandlerForModel creates an API handler for the given model
func (s *AgentService) createHandlerForModel(model models.Model) (llm.ApiHandler, error) {
	// Get provider configuration
	providerCfg, exists := s.config.Providers[model.Provider]
	if !exists {
		return nil, fmt.Errorf("provider %s not configured", model.Provider)
	}

	if providerCfg.Disabled {
		return nil, fmt.Errorf("provider %s is disabled", model.Provider)
	}

	// Create handler options
	options := llm.ApiHandlerOptions{
		APIKey:  providerCfg.APIKey,
		ModelID: string(model.ID),
	}

	// Build handler using the providers factory
	return providers.BuildApiHandler(options)
}

// Run executes an agent with the given configuration
func (s *AgentService) Run(ctx context.Context, sessionID string, agentName config.AgentName, content string, attachments ...llm.ContentBlock) (<-chan AgentEvent, error) {
	// Check if session is already busy
	if s.IsSessionBusy(sessionID) {
		return nil, fmt.Errorf("session %s is already running an agent", sessionID)
	}

	// Get agent configuration
	s.configMutex.RLock()
	agentConfig, exists := s.agentConfigs[agentName]
	s.configMutex.RUnlock()

	if !exists {
		return nil, fmt.Errorf("agent %s not configured", agentName)
	}

	// Create session context with cancellation
	sessionCtx, cancel := context.WithCancel(ctx)

	// Create event channel
	eventChan := make(chan AgentEvent, 100)

	// Create session
	session := &AgentSession{
		ID:        sessionID,
		AgentName: agentName,
		StartTime: time.Now(),
		Cancel:    cancel,
		EventChan: eventChan,
	}

	// Register session
	s.sessionMutex.Lock()
	s.activeSessions[sessionID] = session
	s.sessionMutex.Unlock()

	// Start agent execution in goroutine
	go s.executeAgent(sessionCtx, session, agentConfig, content, attachments...)

	return eventChan, nil
}

// executeAgent executes the agent logic
func (s *AgentService) executeAgent(ctx context.Context, session *AgentSession, config AgentConfig, content string, attachments ...llm.ContentBlock) {
	defer func() {
		// Clean up session
		s.sessionMutex.Lock()
		delete(s.activeSessions, session.ID)
		s.sessionMutex.Unlock()

		close(session.EventChan)
	}()

	// Emit started event
	s.emitEvent(session, AgentEventStarted, map[string]interface{}{
		"agent_name": session.AgentName,
		"model":      config.Model.Name,
		"start_time": session.StartTime,
	})

	// Get system prompt for this agent
	systemPrompt := prompt.GetAgentPrompt(session.AgentName, config.Provider)

	// Create message with content and attachments
	messageContent := []llm.ContentBlock{llm.TextBlock{Text: content}}
	messageContent = append(messageContent, attachments...)

	messages := []llm.Message{
		{
			Role:    "user",
			Content: messageContent,
		},
	}

	// Execute LLM request
	stream, err := config.Handler.CreateMessage(ctx, systemPrompt, messages)
	if err != nil {
		s.emitEvent(session, AgentEventError, map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	// Process stream
	var fullResponse string
	for chunk := range stream {
		select {
		case <-ctx.Done():
			s.emitEvent(session, AgentEventCancelled, map[string]interface{}{
				"reason": "context_cancelled",
			})
			return
		default:
		}

		switch c := chunk.(type) {
		case llm.ApiStreamTextChunk:
			fullResponse += c.Text
			s.emitEvent(session, AgentEventTextChunk, map[string]interface{}{
				"text": c.Text,
			})

		case llm.ApiStreamUsageChunk:
			s.emitEvent(session, AgentEventUsage, map[string]interface{}{
				"input_tokens":  c.InputTokens,
				"output_tokens": c.OutputTokens,
				"total_cost":    c.TotalCost,
			})
		}
	}

	// Emit completion event
	s.emitEvent(session, AgentEventCompleted, map[string]interface{}{
		"response":    fullResponse,
		"duration_ms": time.Since(session.StartTime).Milliseconds(),
		"end_time":    time.Now(),
	})
}

// emitEvent emits an agent event
func (s *AgentService) emitEvent(session *AgentSession, eventType AgentEventType, data map[string]interface{}) {
	event := AgentEvent{
		ID:        uuid.New().String(),
		Type:      eventType,
		SessionID: session.ID,
		AgentName: session.AgentName,
		Data:      data,
		Timestamp: time.Now(),
	}

	// Send to session channel
	select {
	case session.EventChan <- event:
	default:
		log.Printf("Warning: Agent event channel full for session %s", session.ID)
	}

	// Broadcast to event broker
	s.eventBroker.Publish(events.EventType(eventType), event)
}

// Cancel cancels a running agent session
func (s *AgentService) Cancel(sessionID string) {
	s.sessionMutex.RLock()
	session, exists := s.activeSessions[sessionID]
	s.sessionMutex.RUnlock()

	if exists && session.Cancel != nil {
		session.Cancel()
	}
}

// IsSessionBusy checks if a session is currently running an agent
func (s *AgentService) IsSessionBusy(sessionID string) bool {
	s.sessionMutex.RLock()
	defer s.sessionMutex.RUnlock()

	_, exists := s.activeSessions[sessionID]
	return exists
}

// IsBusy checks if any agent is currently running
func (s *AgentService) IsBusy() bool {
	s.sessionMutex.RLock()
	defer s.sessionMutex.RUnlock()

	return len(s.activeSessions) > 0
}

// UpdateAgent updates the configuration for a specific agent
func (s *AgentService) UpdateAgent(agentName config.AgentName, modelID models.ModelID) (models.Model, error) {
	model, exists := models.GetModel(modelID)
	if !exists {
		return models.Model{}, fmt.Errorf("model %s not found", modelID)
	}

	// Create new handler
	handler, err := s.createHandlerForModel(model)
	if err != nil {
		return models.Model{}, fmt.Errorf("failed to create handler: %w", err)
	}

	// Update configuration
	s.configMutex.Lock()
	s.agentConfigs[agentName] = AgentConfig{
		Agent: config.Agent{
			Model:     modelID,
			MaxTokens: model.DefaultMaxTokens,
		},
		Model:    model,
		Handler:  handler,
		Provider: model.Provider,
	}
	s.configMutex.Unlock()

	// Update config
	s.config.Agents[agentName] = config.Agent{
		Model:     modelID,
		MaxTokens: model.DefaultMaxTokens,
	}

	return model, nil
}

// GetModel returns the current model for an agent
func (s *AgentService) GetModel(agentName config.AgentName) (models.Model, error) {
	s.configMutex.RLock()
	defer s.configMutex.RUnlock()

	agentConfig, exists := s.agentConfigs[agentName]
	if !exists {
		return models.Model{}, fmt.Errorf("agent %s not configured", agentName)
	}

	return agentConfig.Model, nil
}

// Subscribe subscribes to agent events
func (s *AgentService) Subscribe(ctx context.Context, sessionID string) <-chan AgentEvent {
	eventCh := s.eventBroker.Subscribe(ctx, events.FilterBySessionID(sessionID))
	agentCh := make(chan AgentEvent, 100)

	go func() {
		defer close(agentCh)
		for event := range eventCh {
			agentCh <- event.Payload
		}
	}()

	return agentCh
}
