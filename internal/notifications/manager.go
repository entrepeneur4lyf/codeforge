package notifications

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/events"
	"github.com/google/uuid"
)

// NotificationLevel represents the severity level of a notification
type NotificationLevel string

const (
	LevelInfo    NotificationLevel = "info"
	LevelSuccess NotificationLevel = "success"
	LevelWarning NotificationLevel = "warning"
	LevelError   NotificationLevel = "error"
)

// NotificationStatus represents the current status of a notification
type NotificationStatus string

const (
	StatusPending   NotificationStatus = "pending"
	StatusDisplayed NotificationStatus = "displayed"
	StatusDismissed NotificationStatus = "dismissed"
	StatusExpired   NotificationStatus = "expired"
)

// Notification represents a single notification
type Notification struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Level       NotificationLevel      `json:"level"`
	Status      NotificationStatus     `json:"status"`
	Duration    time.Duration          `json:"duration"`
	Dismissible bool                   `json:"dismissible"`
	Actions     []NotificationAction   `json:"actions,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
	UserID      string                 `json:"user_id,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	DisplayedAt *time.Time             `json:"displayed_at,omitempty"`
	DismissedAt *time.Time             `json:"dismissed_at,omitempty"`
	ExpiresAt   *time.Time             `json:"expires_at,omitempty"`
}

// NotificationAction represents an action that can be taken on a notification
type NotificationAction struct {
	ID       string                 `json:"id"`
	Label    string                 `json:"label"`
	Style    string                 `json:"style"` // primary, secondary, danger
	Handler  string                 `json:"handler,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NotificationOptions contains options for creating notifications
type NotificationOptions struct {
	SessionID   string
	UserID      string
	Duration    time.Duration
	Dismissible bool
	Actions     []NotificationAction
	Metadata    map[string]interface{}
}

// NotificationOption is a function that modifies NotificationOptions
type NotificationOption func(*NotificationOptions)

// WithSessionID sets the session ID for the notification
func WithSessionID(sessionID string) NotificationOption {
	return func(opts *NotificationOptions) {
		opts.SessionID = sessionID
	}
}

// WithUserID sets the user ID for the notification
func WithUserID(userID string) NotificationOption {
	return func(opts *NotificationOptions) {
		opts.UserID = userID
	}
}

// WithDuration sets the duration for the notification
func WithDuration(duration time.Duration) NotificationOption {
	return func(opts *NotificationOptions) {
		opts.Duration = duration
	}
}

// WithDismissible sets whether the notification can be dismissed
func WithDismissible(dismissible bool) NotificationOption {
	return func(opts *NotificationOptions) {
		opts.Dismissible = dismissible
	}
}

// WithActions sets the actions for the notification
func WithActions(actions []NotificationAction) NotificationOption {
	return func(opts *NotificationOptions) {
		opts.Actions = actions
	}
}

// WithMetadata sets metadata for the notification
func WithMetadata(metadata map[string]interface{}) NotificationOption {
	return func(opts *NotificationOptions) {
		opts.Metadata = metadata
	}
}

// Manager manages notifications and integrates with the event system
type Manager struct {
	eventManager     *events.Manager
	notifications    map[string]*Notification
	sessionQueues    map[string][]*Notification
	mu               sync.RWMutex
	maxNotifications int
	defaultDurations map[NotificationLevel]time.Duration
	cleanupInterval  time.Duration
	stopCleanup      chan struct{}
}

// NewManager creates a new notification manager
func NewManager(eventManager *events.Manager) *Manager {
	return &Manager{
		eventManager:     eventManager,
		notifications:    make(map[string]*Notification),
		sessionQueues:    make(map[string][]*Notification),
		maxNotifications: 1000,
		defaultDurations: map[NotificationLevel]time.Duration{
			LevelInfo:    5 * time.Second,
			LevelSuccess: 3 * time.Second,
			LevelWarning: 10 * time.Second,
			LevelError:   0, // Don't auto-dismiss errors
		},
		cleanupInterval: 1 * time.Minute,
		stopCleanup:     make(chan struct{}),
	}
}

// Start initializes the notification manager
func (m *Manager) Start() error {
	log.Printf("🔔 Notification manager started")

	// Start cleanup goroutine
	go m.cleanupLoop()

	return nil
}

// Stop shuts down the notification manager
func (m *Manager) Stop() {
	close(m.stopCleanup)
	log.Printf("🔔 Notification manager stopped")
}

// CreateNotification creates a new notification
func (m *Manager) CreateNotification(title, message string, level NotificationLevel, opts ...NotificationOption) *Notification {
	options := &NotificationOptions{
		Duration:    m.defaultDurations[level],
		Dismissible: true,
	}

	for _, opt := range opts {
		opt(options)
	}

	notification := &Notification{
		ID:          uuid.New().String(),
		Title:       title,
		Message:     message,
		Level:       level,
		Status:      StatusPending,
		Duration:    options.Duration,
		Dismissible: options.Dismissible,
		Actions:     options.Actions,
		Metadata:    options.Metadata,
		SessionID:   options.SessionID,
		UserID:      options.UserID,
		CreatedAt:   time.Now(),
	}

	// Set expiration time if duration is specified
	if notification.Duration > 0 {
		expiresAt := notification.CreatedAt.Add(notification.Duration)
		notification.ExpiresAt = &expiresAt
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Store notification
	m.notifications[notification.ID] = notification

	// Add to session queue if session ID is provided
	if notification.SessionID != "" {
		m.sessionQueues[notification.SessionID] = append(m.sessionQueues[notification.SessionID], notification)
	}

	// Trim notifications if we exceed max
	m.trimNotifications()

	// Publish notification event
	if m.eventManager != nil {
		eventType := events.NotificationInfo
		switch level {
		case LevelSuccess:
			eventType = events.NotificationSuccess
		case LevelWarning:
			eventType = events.NotificationWarning
		case LevelError:
			eventType = events.NotificationError
		}

		payload := events.NotificationEventPayload{
			Title:       title,
			Message:     message,
			Level:       string(level),
			Duration:    notification.Duration,
			Dismissible: notification.Dismissible,
			Actions:     convertActionsToEventActions(options.Actions),
			Metadata:    options.Metadata,
		}

		publishOpts := []events.PublishOption{}
		if options.SessionID != "" {
			publishOpts = append(publishOpts, events.WithSessionID(options.SessionID))
		}
		if options.UserID != "" {
			publishOpts = append(publishOpts, events.WithUserID(options.UserID))
		}

		m.eventManager.PublishNotification(eventType, payload, publishOpts...)
	}

	return notification
}

// convertActionsToEventActions converts notification actions to event actions
func convertActionsToEventActions(actions []NotificationAction) []events.NotificationAction {
	eventActions := make([]events.NotificationAction, len(actions))
	for i, action := range actions {
		eventActions[i] = events.NotificationAction{
			ID:    action.ID,
			Label: action.Label,
			Style: action.Style,
		}
	}
	return eventActions
}

// GetNotification retrieves a notification by ID
func (m *Manager) GetNotification(id string) (*Notification, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	notification, exists := m.notifications[id]
	return notification, exists
}

// GetSessionNotifications retrieves all notifications for a session
func (m *Manager) GetSessionNotifications(sessionID string) []*Notification {
	m.mu.RLock()
	defer m.mu.RUnlock()

	notifications := m.sessionQueues[sessionID]
	result := make([]*Notification, len(notifications))
	copy(result, notifications)
	return result
}

// GetActiveNotifications retrieves all active (non-dismissed, non-expired) notifications
func (m *Manager) GetActiveNotifications(sessionID string) []*Notification {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var active []*Notification
	now := time.Now()

	notifications := m.sessionQueues[sessionID]
	for _, notification := range notifications {
		if notification.Status == StatusPending || notification.Status == StatusDisplayed {
			// Check if not expired
			if notification.ExpiresAt == nil || notification.ExpiresAt.After(now) {
				active = append(active, notification)
			}
		}
	}

	return active
}

// MarkDisplayed marks a notification as displayed
func (m *Manager) MarkDisplayed(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	notification, exists := m.notifications[id]
	if !exists {
		return fmt.Errorf("notification not found: %s", id)
	}

	if notification.Status == StatusPending {
		notification.Status = StatusDisplayed
		now := time.Now()
		notification.DisplayedAt = &now
	}

	return nil
}

// DismissNotification dismisses a notification
func (m *Manager) DismissNotification(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	notification, exists := m.notifications[id]
	if !exists {
		return fmt.Errorf("notification not found: %s", id)
	}

	if !notification.Dismissible {
		return fmt.Errorf("notification is not dismissible: %s", id)
	}

	notification.Status = StatusDismissed
	now := time.Now()
	notification.DismissedAt = &now

	return nil
}

// trimNotifications removes old notifications to stay within limits
func (m *Manager) trimNotifications() {
	if len(m.notifications) <= m.maxNotifications {
		return
	}

	// Find oldest notifications to remove
	var toRemove []string
	count := len(m.notifications) - m.maxNotifications

	for id, notification := range m.notifications {
		if notification.Status == StatusDismissed || notification.Status == StatusExpired {
			toRemove = append(toRemove, id)
			if len(toRemove) >= count {
				break
			}
		}
	}

	// Remove notifications
	for _, id := range toRemove {
		delete(m.notifications, id)
	}
}

// cleanupLoop periodically cleans up expired notifications
func (m *Manager) cleanupLoop() {
	ticker := time.NewTicker(m.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cleanupExpiredNotifications()
		case <-m.stopCleanup:
			return
		}
	}
}

// cleanupExpiredNotifications removes expired notifications
func (m *Manager) cleanupExpiredNotifications() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	var expired []string

	for id, notification := range m.notifications {
		if notification.ExpiresAt != nil && notification.ExpiresAt.Before(now) {
			if notification.Status != StatusExpired {
				notification.Status = StatusExpired
			}
			expired = append(expired, id)
		}
	}

	log.Printf("🔔 Cleaned up %d expired notifications", len(expired))
}

// GetStats returns notification statistics
func (m *Manager) GetStats() NotificationStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := NotificationStats{
		Total:    len(m.notifications),
		Sessions: len(m.sessionQueues),
		ByLevel:  make(map[NotificationLevel]int),
		ByStatus: make(map[NotificationStatus]int),
	}

	for _, notification := range m.notifications {
		stats.ByLevel[notification.Level]++
		stats.ByStatus[notification.Status]++
	}

	return stats
}

// NotificationStats contains notification statistics
type NotificationStats struct {
	Total    int                        `json:"total"`
	Sessions int                        `json:"sessions"`
	ByLevel  map[NotificationLevel]int  `json:"by_level"`
	ByStatus map[NotificationStatus]int `json:"by_status"`
}

// Convenience methods for common notification types

// Info creates an info notification
func (m *Manager) Info(title, message string, opts ...NotificationOption) *Notification {
	return m.CreateNotification(title, message, LevelInfo, opts...)
}

// Success creates a success notification
func (m *Manager) Success(title, message string, opts ...NotificationOption) *Notification {
	return m.CreateNotification(title, message, LevelSuccess, opts...)
}

// Warning creates a warning notification
func (m *Manager) Warning(title, message string, opts ...NotificationOption) *Notification {
	return m.CreateNotification(title, message, LevelWarning, opts...)
}

// Error creates an error notification
func (m *Manager) Error(title, message string, opts ...NotificationOption) *Notification {
	return m.CreateNotification(title, message, LevelError, opts...)
}
