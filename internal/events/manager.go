package events

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Manager coordinates multiple event brokers and provides a unified interface
type Manager struct {
	// Typed brokers for different event categories
	chatBroker         *Broker[ChatEventPayload]
	contextBroker      *Broker[ContextEventPayload]
	permissionBroker   *Broker[PermissionEventPayload]
	notificationBroker *Broker[NotificationEventPayload]
	systemBroker       *Broker[SystemEventPayload]
	fileBroker         *Broker[FileEventPayload]
	vectorBroker       *Broker[VectorEventPayload]
	mcpBroker          *Broker[MCPEventPayload]

	// Generic broker for any event type
	genericBroker *Broker[any]

	mu          sync.RWMutex
	persistence PersistenceStore
	started     bool
	shutdown    chan struct{}
}

// NewManager creates a new event manager
func NewManager() *Manager {
	return &Manager{
		chatBroker:         NewBroker[ChatEventPayload](),
		contextBroker:      NewBroker[ContextEventPayload](),
		permissionBroker:   NewBroker[PermissionEventPayload](),
		notificationBroker: NewBroker[NotificationEventPayload](),
		systemBroker:       NewBroker[SystemEventPayload](),
		fileBroker:         NewBroker[FileEventPayload](),
		vectorBroker:       NewBroker[VectorEventPayload](),
		mcpBroker:          NewBroker[MCPEventPayload](),
		genericBroker:      NewBroker[any](),
		shutdown:           make(chan struct{}),
	}
}

// SetPersistence sets the persistence store for all brokers
func (m *Manager) SetPersistence(store PersistenceStore) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.persistence = store
	m.chatBroker.SetPersistence(store)
	m.contextBroker.SetPersistence(store)
	m.permissionBroker.SetPersistence(store)
	m.notificationBroker.SetPersistence(store)
	m.systemBroker.SetPersistence(store)
	m.fileBroker.SetPersistence(store)
	m.vectorBroker.SetPersistence(store)
	m.mcpBroker.SetPersistence(store)
	m.genericBroker.SetPersistence(store)
}

// Start initializes the event manager
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return fmt.Errorf("event manager already started")
	}

	m.started = true
	log.Printf("🎯 Event manager started")

	// Publish system started event
	go func() {
		m.PublishSystem(SystemStarted, SystemEventPayload{
			Component: "event_manager",
			Status:    "started",
			Message:   "Event management system initialized",
		})
	}()

	return nil
}

// Shutdown gracefully shuts down all brokers
func (m *Manager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return
	}

	select {
	case <-m.shutdown:
		return // Already shut down
	default:
		close(m.shutdown)
	}

	// Publish shutdown event
	m.systemBroker.Publish(SystemShutdown, SystemEventPayload{
		Component: "event_manager",
		Status:    "shutting_down",
		Message:   "Event management system shutting down",
	})

	// Shutdown all brokers
	m.chatBroker.Shutdown()
	m.contextBroker.Shutdown()
	m.permissionBroker.Shutdown()
	m.notificationBroker.Shutdown()
	m.systemBroker.Shutdown()
	m.fileBroker.Shutdown()
	m.vectorBroker.Shutdown()
	m.mcpBroker.Shutdown()
	m.genericBroker.Shutdown()

	m.started = false
	log.Printf("🎯 Event manager shut down")
}

// Chat event methods
func (m *Manager) PublishChat(eventType EventType, payload ChatEventPayload, opts ...PublishOption) {
	m.chatBroker.Publish(eventType, payload, opts...)
	m.publishToGeneric(eventType, payload, opts...)
}

func (m *Manager) SubscribeChat(ctx context.Context, filters ...EventFilter) <-chan Event[ChatEventPayload] {
	return m.chatBroker.Subscribe(ctx, filters...)
}

// Context event methods
func (m *Manager) PublishContext(eventType EventType, payload ContextEventPayload, opts ...PublishOption) {
	m.contextBroker.Publish(eventType, payload, opts...)
	m.publishToGeneric(eventType, payload, opts...)
}

func (m *Manager) SubscribeContext(ctx context.Context, filters ...EventFilter) <-chan Event[ContextEventPayload] {
	return m.contextBroker.Subscribe(ctx, filters...)
}

// Permission event methods
func (m *Manager) PublishPermission(eventType EventType, payload PermissionEventPayload, opts ...PublishOption) {
	m.permissionBroker.Publish(eventType, payload, opts...)
	m.publishToGeneric(eventType, payload, opts...)
}

func (m *Manager) SubscribePermission(ctx context.Context, filters ...EventFilter) <-chan Event[PermissionEventPayload] {
	return m.permissionBroker.Subscribe(ctx, filters...)
}

// Notification event methods
func (m *Manager) PublishNotification(eventType EventType, payload NotificationEventPayload, opts ...PublishOption) {
	m.notificationBroker.Publish(eventType, payload, opts...)
	m.publishToGeneric(eventType, payload, opts...)
}

func (m *Manager) SubscribeNotification(ctx context.Context, filters ...EventFilter) <-chan Event[NotificationEventPayload] {
	return m.notificationBroker.Subscribe(ctx, filters...)
}

// System event methods
func (m *Manager) PublishSystem(eventType EventType, payload SystemEventPayload, opts ...PublishOption) {
	m.systemBroker.Publish(eventType, payload, opts...)
	m.publishToGeneric(eventType, payload, opts...)
}

func (m *Manager) SubscribeSystem(ctx context.Context, filters ...EventFilter) <-chan Event[SystemEventPayload] {
	return m.systemBroker.Subscribe(ctx, filters...)
}

// File event methods
func (m *Manager) PublishFile(eventType EventType, payload FileEventPayload, opts ...PublishOption) {
	m.fileBroker.Publish(eventType, payload, opts...)
	m.publishToGeneric(eventType, payload, opts...)
}

func (m *Manager) SubscribeFile(ctx context.Context, filters ...EventFilter) <-chan Event[FileEventPayload] {
	return m.fileBroker.Subscribe(ctx, filters...)
}

// Vector event methods
func (m *Manager) PublishVector(eventType EventType, payload VectorEventPayload, opts ...PublishOption) {
	m.vectorBroker.Publish(eventType, payload, opts...)
	m.publishToGeneric(eventType, payload, opts...)
}

func (m *Manager) SubscribeVector(ctx context.Context, filters ...EventFilter) <-chan Event[VectorEventPayload] {
	return m.vectorBroker.Subscribe(ctx, filters...)
}

// MCP event methods
func (m *Manager) PublishMCP(eventType EventType, payload MCPEventPayload, opts ...PublishOption) {
	m.mcpBroker.Publish(eventType, payload, opts...)
	m.publishToGeneric(eventType, payload, opts...)
}

func (m *Manager) SubscribeMCP(ctx context.Context, filters ...EventFilter) <-chan Event[MCPEventPayload] {
	return m.mcpBroker.Subscribe(ctx, filters...)
}

// Generic event methods
func (m *Manager) PublishGeneric(eventType EventType, payload any, opts ...PublishOption) {
	m.genericBroker.Publish(eventType, payload, opts...)
}

func (m *Manager) SubscribeGeneric(ctx context.Context, filters ...EventFilter) <-chan Event[any] {
	return m.genericBroker.Subscribe(ctx, filters...)
}

// publishToGeneric publishes to the generic broker for cross-cutting concerns
func (m *Manager) publishToGeneric(eventType EventType, payload any, opts ...PublishOption) {
	m.genericBroker.Publish(eventType, payload, opts...)
}

// GetStats returns statistics for all brokers
func (m *Manager) GetStats() ManagerStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return ManagerStats{
		Started:      m.started,
		Chat:         m.chatBroker.GetStats(),
		Context:      m.contextBroker.GetStats(),
		Permission:   m.permissionBroker.GetStats(),
		Notification: m.notificationBroker.GetStats(),
		System:       m.systemBroker.GetStats(),
		File:         m.fileBroker.GetStats(),
		Vector:       m.vectorBroker.GetStats(),
		MCP:          m.mcpBroker.GetStats(),
		Generic:      m.genericBroker.GetStats(),
	}
}

// ManagerStats contains statistics for all brokers
type ManagerStats struct {
	Started      bool        `json:"started"`
	Chat         BrokerStats `json:"chat"`
	Context      BrokerStats `json:"context"`
	Permission   BrokerStats `json:"permission"`
	Notification BrokerStats `json:"notification"`
	System       BrokerStats `json:"system"`
	File         BrokerStats `json:"file"`
	Vector       BrokerStats `json:"vector"`
	MCP          BrokerStats `json:"mcp"`
	Generic      BrokerStats `json:"generic"`
}

// Convenience methods for common notifications

// NotifyInfo publishes an info notification
func (m *Manager) NotifyInfo(title, message string, opts ...PublishOption) {
	m.PublishNotification(NotificationInfo, NotificationEventPayload{
		Title:       title,
		Message:     message,
		Level:       "info",
		Duration:    5 * time.Second,
		Dismissible: true,
	}, opts...)
}

// NotifySuccess publishes a success notification
func (m *Manager) NotifySuccess(title, message string, opts ...PublishOption) {
	m.PublishNotification(NotificationSuccess, NotificationEventPayload{
		Title:       title,
		Message:     message,
		Level:       "success",
		Duration:    3 * time.Second,
		Dismissible: true,
	}, opts...)
}

// NotifyWarning publishes a warning notification
func (m *Manager) NotifyWarning(title, message string, opts ...PublishOption) {
	m.PublishNotification(NotificationWarning, NotificationEventPayload{
		Title:       title,
		Message:     message,
		Level:       "warning",
		Duration:    10 * time.Second,
		Dismissible: true,
	}, opts...)
}

// NotifyError publishes an error notification
func (m *Manager) NotifyError(title, message string, opts ...PublishOption) {
	m.PublishNotification(NotificationError, NotificationEventPayload{
		Title:       title,
		Message:     message,
		Level:       "error",
		Duration:    0, // Don't auto-dismiss errors
		Dismissible: true,
	}, opts...)
}

// GetEventsForSession retrieves events for a specific session since a timestamp
func (m *Manager) GetEventsForSession(sessionID string, since time.Time) ([]Event[any], error) {
	if m.persistence == nil {
		return nil, fmt.Errorf("persistence store not available")
	}

	if memStore, ok := m.persistence.(*MemoryPersistenceStore); ok {
		return memStore.GetEventsForSession(sessionID, since)
	}

	if dbStore, ok := m.persistence.(*DatabasePersistenceStore); ok {
		return dbStore.GetEventsForSession(sessionID, since)
	}

	return nil, fmt.Errorf("GetEventsForSession not supported for this persistence store type")
}

// GetEventsForUser retrieves events for a specific user since a timestamp
func (m *Manager) GetEventsForUser(userID string, since time.Time) ([]Event[any], error) {
	if m.persistence == nil {
		return nil, fmt.Errorf("persistence store not available")
	}

	if memStore, ok := m.persistence.(*MemoryPersistenceStore); ok {
		return memStore.GetEventsForUser(userID, since)
	}

	if dbStore, ok := m.persistence.(*DatabasePersistenceStore); ok {
		return dbStore.GetEventsForUser(userID, since)
	}

	return nil, fmt.Errorf("GetEventsForUser not supported for this persistence store type")
}

// GetEventsByType retrieves events of a specific type since a timestamp
func (m *Manager) GetEventsByType(eventType EventType, since time.Time) ([]Event[any], error) {
	if m.persistence == nil {
		return nil, fmt.Errorf("persistence store not available")
	}

	if memStore, ok := m.persistence.(*MemoryPersistenceStore); ok {
		return memStore.GetEventsByType(eventType, since)
	}

	if dbStore, ok := m.persistence.(*DatabasePersistenceStore); ok {
		return dbStore.GetEventsByType(eventType, since)
	}

	return nil, fmt.Errorf("GetEventsByType not supported for this persistence store type")
}
