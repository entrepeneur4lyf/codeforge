package events

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	defaultBufferSize = 64
	defaultMaxEvents  = 1000
)

// Broker implements a generic publish-subscribe broker with type safety
type Broker[T any] struct {
	subs         map[chan Event[T]]SubscriberInfo
	mu           sync.RWMutex
	done         chan struct{}
	subCount     int
	maxEvents    int
	bufferSize   int
	eventHistory []Event[T]
	historyMu    sync.RWMutex
	persistence  PersistenceStore
}

// SubscriberInfo contains metadata about a subscriber
type SubscriberInfo struct {
	ID      string
	Filters []EventFilter
	Created time.Time
}

// PersistenceStore defines the interface for event persistence
type PersistenceStore interface {
	Store(event Event[any]) error
	Retrieve(filter EventFilter, limit int) ([]Event[any], error)
	Delete(eventID string) error
	Cleanup(olderThan time.Time) error
}

// NewBroker creates a new broker with default settings
func NewBroker[T any]() *Broker[T] {
	return NewBrokerWithOptions[T](defaultBufferSize, defaultMaxEvents)
}

// NewBrokerWithOptions creates a new broker with custom settings
func NewBrokerWithOptions[T any](channelBufferSize, maxEvents int) *Broker[T] {
	return &Broker[T]{
		subs:         make(map[chan Event[T]]SubscriberInfo),
		done:         make(chan struct{}),
		subCount:     0,
		maxEvents:    maxEvents,
		bufferSize:   channelBufferSize,
		eventHistory: make([]Event[T], 0, maxEvents),
	}
}

// SetPersistence sets the persistence store for the broker
func (b *Broker[T]) SetPersistence(store PersistenceStore) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.persistence = store
}

// Publish publishes an event to all subscribers
func (b *Broker[T]) Publish(eventType EventType, payload T, opts ...PublishOption) {
	select {
	case <-b.done:
		return // Broker is shut down
	default:
	}

	// Apply options
	options := &PublishOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// Create event
	event := Event[T]{
		ID:        uuid.New().String(),
		Type:      eventType,
		Payload:   payload,
		Timestamp: time.Now(),
		SessionID: options.SessionID,
		UserID:    options.UserID,
		Metadata:  options.Metadata,
	}

	// Store in history
	b.addToHistory(event)

	// Persist if requested
	if options.Persist && b.persistence != nil {
		if err := b.persistence.Store(Event[any]{
			ID:        event.ID,
			Type:      event.Type,
			Payload:   payload,
			Timestamp: event.Timestamp,
			SessionID: event.SessionID,
			UserID:    event.UserID,
			Metadata:  event.Metadata,
		}); err != nil {
			log.Printf("Failed to persist event %s: %v", event.ID, err)
		}
	}

	// Send to subscribers
	b.mu.RLock()
	defer b.mu.RUnlock()

	for ch, info := range b.subs {
		// Apply filters
		if b.shouldSendToSubscriber(event, info.Filters) {
			select {
			case ch <- event:
			default:
				// Channel is full, log warning but don't block
				log.Printf("Warning: Event channel full for subscriber %s, dropping event %s", info.ID, event.ID)
			}
		}
	}
}

// Subscribe creates a new subscription with optional filters
func (b *Broker[T]) Subscribe(ctx context.Context, filters ...EventFilter) <-chan Event[T] {
	b.mu.Lock()
	defer b.mu.Unlock()

	ch := make(chan Event[T], b.bufferSize)
	info := SubscriberInfo{
		ID:      uuid.New().String(),
		Filters: filters,
		Created: time.Now(),
	}

	b.subs[ch] = info
	b.subCount++

	// Start goroutine to handle context cancellation
	go func() {
		<-ctx.Done()
		b.unsubscribe(ch)
	}()

	return ch
}

// unsubscribe removes a subscriber
func (b *Broker[T]) unsubscribe(ch chan Event[T]) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, exists := b.subs[ch]; exists {
		delete(b.subs, ch)
		close(ch)
		b.subCount--
	}
}

// shouldSendToSubscriber checks if an event should be sent to a subscriber based on filters
func (b *Broker[T]) shouldSendToSubscriber(event Event[T], filters []EventFilter) bool {
	if len(filters) == 0 {
		return true // No filters means accept all events
	}

	// Convert to Event[any] for filter compatibility
	anyEvent := Event[any]{
		ID:        event.ID,
		Type:      event.Type,
		Payload:   event.Payload,
		Timestamp: event.Timestamp,
		SessionID: event.SessionID,
		UserID:    event.UserID,
		Metadata:  event.Metadata,
	}

	for _, filter := range filters {
		if !filter(anyEvent) {
			return false
		}
	}
	return true
}

// addToHistory adds an event to the in-memory history
func (b *Broker[T]) addToHistory(event Event[T]) {
	b.historyMu.Lock()
	defer b.historyMu.Unlock()

	b.eventHistory = append(b.eventHistory, event)

	// Trim history if it exceeds max events
	if len(b.eventHistory) > b.maxEvents {
		// Remove oldest events
		copy(b.eventHistory, b.eventHistory[len(b.eventHistory)-b.maxEvents:])
		b.eventHistory = b.eventHistory[:b.maxEvents]
	}
}

// GetHistory returns recent events matching the given filters
func (b *Broker[T]) GetHistory(filters ...EventFilter) []Event[T] {
	b.historyMu.RLock()
	defer b.historyMu.RUnlock()

	if len(filters) == 0 {
		// Return copy of all history
		result := make([]Event[T], len(b.eventHistory))
		copy(result, b.eventHistory)
		return result
	}

	var result []Event[T]
	for _, event := range b.eventHistory {
		if b.shouldSendToSubscriber(event, filters) {
			result = append(result, event)
		}
	}

	return result
}

// GetStats returns broker statistics
func (b *Broker[T]) GetStats() BrokerStats {
	b.mu.RLock()
	defer b.mu.RUnlock()

	b.historyMu.RLock()
	historyCount := len(b.eventHistory)
	b.historyMu.RUnlock()

	return BrokerStats{
		SubscriberCount: b.subCount,
		EventHistory:    historyCount,
		MaxEvents:       b.maxEvents,
		BufferSize:      b.bufferSize,
		IsShutdown:      b.isShutdown(),
	}
}

// BrokerStats contains broker statistics
type BrokerStats struct {
	SubscriberCount int  `json:"subscriber_count"`
	EventHistory    int  `json:"event_history"`
	MaxEvents       int  `json:"max_events"`
	BufferSize      int  `json:"buffer_size"`
	IsShutdown      bool `json:"is_shutdown"`
}

// isShutdown checks if the broker is shut down
func (b *Broker[T]) isShutdown() bool {
	select {
	case <-b.done:
		return true
	default:
		return false
	}
}

// Shutdown gracefully shuts down the broker
func (b *Broker[T]) Shutdown() {
	select {
	case <-b.done: // Already closed
		return
	default:
		close(b.done)
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// Close all subscriber channels
	for ch := range b.subs {
		delete(b.subs, ch)
		close(ch)
	}

	b.subCount = 0
	log.Printf("Event broker shut down with %d events in history", len(b.eventHistory))
}

// ReplayEvents replays historical events to a new subscriber
func (b *Broker[T]) ReplayEvents(ctx context.Context, since time.Time, filters ...EventFilter) <-chan Event[T] {
	ch := make(chan Event[T], b.bufferSize)

	go func() {
		defer close(ch)

		// Get historical events
		b.historyMu.RLock()
		var eventsToReplay []Event[T]
		for _, event := range b.eventHistory {
			if event.Timestamp.After(since) && b.shouldSendToSubscriber(event, filters) {
				eventsToReplay = append(eventsToReplay, event)
			}
		}
		b.historyMu.RUnlock()

		// Send historical events
		for _, event := range eventsToReplay {
			select {
			case ch <- event:
			case <-ctx.Done():
				return
			}
		}

		// Subscribe to new events
		newEventsCh := b.Subscribe(ctx, filters...)
		for {
			select {
			case event, ok := <-newEventsCh:
				if !ok {
					return
				}
				select {
				case ch <- event:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return ch
}

// String returns a string representation of the broker
func (b *Broker[T]) String() string {
	stats := b.GetStats()
	return fmt.Sprintf("Broker[subscribers=%d, history=%d, shutdown=%v]",
		stats.SubscriberCount, stats.EventHistory, stats.IsShutdown)
}
