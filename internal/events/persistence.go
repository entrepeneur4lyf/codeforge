package events

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	_ "github.com/tursodatabase/go-libsql"
)

// MemoryPersistenceStore implements in-memory event persistence
type MemoryPersistenceStore struct {
	events  []Event[any]
	mu      sync.RWMutex
	maxSize int
}

// NewMemoryPersistenceStore creates a new in-memory persistence store
func NewMemoryPersistenceStore(maxSize int) *MemoryPersistenceStore {
	if maxSize <= 0 {
		maxSize = 10000 // Default max size
	}

	return &MemoryPersistenceStore{
		events:  make([]Event[any], 0, maxSize),
		maxSize: maxSize,
	}
}

// Store stores an event in memory
func (m *MemoryPersistenceStore) Store(event Event[any]) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.events = append(m.events, event)

	// Trim if exceeding max size
	if len(m.events) > m.maxSize {
		// Remove oldest events (keep most recent maxSize events)
		copy(m.events, m.events[len(m.events)-m.maxSize:])
		m.events = m.events[:m.maxSize]
	}

	return nil
}

// Retrieve retrieves events matching the filter
func (m *MemoryPersistenceStore) Retrieve(filter EventFilter, limit int) ([]Event[any], error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []Event[any]

	for _, event := range m.events {
		if filter == nil || filter(event) {
			result = append(result, event)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}

	return result, nil
}

// Delete removes an event by ID
func (m *MemoryPersistenceStore) Delete(eventID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, event := range m.events {
		if event.ID == eventID {
			// Remove event by swapping with last element and truncating
			m.events[i] = m.events[len(m.events)-1]
			m.events = m.events[:len(m.events)-1]
			return nil
		}
	}

	return nil // Event not found, but not an error
}

// Cleanup removes events older than the specified time
func (m *MemoryPersistenceStore) Cleanup(olderThan time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var filtered []Event[any]
	for _, event := range m.events {
		if event.Timestamp.After(olderThan) {
			filtered = append(filtered, event)
		}
	}

	m.events = filtered
	return nil
}

// GetStats returns statistics about the persistence store
func (m *MemoryPersistenceStore) GetStats() PersistenceStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var oldestTime, newestTime time.Time
	if len(m.events) > 0 {
		oldestTime = m.events[0].Timestamp
		newestTime = m.events[0].Timestamp

		for _, event := range m.events {
			if event.Timestamp.Before(oldestTime) {
				oldestTime = event.Timestamp
			}
			if event.Timestamp.After(newestTime) {
				newestTime = event.Timestamp
			}
		}
	}

	return PersistenceStats{
		TotalEvents: len(m.events),
		MaxSize:     m.maxSize,
		OldestEvent: oldestTime,
		NewestEvent: newestTime,
	}
}

// PersistenceStats contains statistics about the persistence store
type PersistenceStats struct {
	TotalEvents int       `json:"total_events"`
	MaxSize     int       `json:"max_size"`
	OldestEvent time.Time `json:"oldest_event"`
	NewestEvent time.Time `json:"newest_event"`
}

// GetEventsForSession retrieves events for a specific session since a timestamp
func (m *MemoryPersistenceStore) GetEventsForSession(sessionID string, since time.Time) ([]Event[any], error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []Event[any]
	for _, event := range m.events {
		if event.SessionID == sessionID && event.Timestamp.After(since) {
			result = append(result, event)
		}
	}

	return result, nil
}

// GetEventsForUser retrieves events for a specific user since a timestamp
func (m *MemoryPersistenceStore) GetEventsForUser(userID string, since time.Time) ([]Event[any], error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []Event[any]
	for _, event := range m.events {
		if event.UserID == userID && event.Timestamp.After(since) {
			result = append(result, event)
		}
	}

	return result, nil
}

// GetEventsByType retrieves events of a specific type since a timestamp
func (m *MemoryPersistenceStore) GetEventsByType(eventType EventType, since time.Time) ([]Event[any], error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []Event[any]
	for _, event := range m.events {
		if event.Type == eventType && event.Timestamp.After(since) {
			result = append(result, event)
		}
	}

	return result, nil
}

// GetEventsSince retrieves events since a specific timestamp
func (m *MemoryPersistenceStore) GetEventsSince(since time.Time) ([]Event[any], error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []Event[any]
	for _, event := range m.events {
		if event.Timestamp.After(since) {
			result = append(result, event)
		}
	}

	return result, nil
}

// DatabasePersistenceStore implements database-backed event persistence using libsql
type DatabasePersistenceStore struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewDatabasePersistenceStore creates a new database persistence store
func NewDatabasePersistenceStore(dbPath string) (*DatabasePersistenceStore, error) {
	// Use libsql like other parts of the system
	db, err := sql.Open("libsql", "file:"+dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	store := &DatabasePersistenceStore{
		db: db,
	}

	// Initialize tables
	if err := store.initTables(); err != nil {
		return nil, fmt.Errorf("failed to initialize tables: %w", err)
	}

	return store, nil
}

// initTables creates the events table if it doesn't exist
func (d *DatabasePersistenceStore) initTables() error {
	query := `
	CREATE TABLE IF NOT EXISTS events (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		payload TEXT NOT NULL,
		timestamp INTEGER NOT NULL,
		session_id TEXT,
		user_id TEXT,
		metadata TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_events_type ON events(type);
	CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);
	CREATE INDEX IF NOT EXISTS idx_events_session_id ON events(session_id);
	CREATE INDEX IF NOT EXISTS idx_events_user_id ON events(user_id);
	`

	_, err := d.db.Exec(query)
	return err
}

// Store stores an event in the database
func (d *DatabasePersistenceStore) Store(event Event[any]) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Serialize payload to JSON
	payloadBytes, err := json.Marshal(event.Payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Serialize metadata to JSON
	var metadataBytes []byte
	if event.Metadata != nil {
		metadataBytes, err = json.Marshal(event.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
	}

	query := `
	INSERT INTO events (id, type, payload, timestamp, session_id, user_id, metadata)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err = d.db.Exec(query,
		event.ID,
		string(event.Type),
		string(payloadBytes),
		event.Timestamp.Unix(),
		event.SessionID,
		event.UserID,
		string(metadataBytes),
	)

	return err
}

// Retrieve retrieves events matching the filter
func (d *DatabasePersistenceStore) Retrieve(filter EventFilter, limit int) ([]Event[any], error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
	SELECT id, type, payload, timestamp, session_id, user_id, metadata
	FROM events
	ORDER BY timestamp DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := d.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []Event[any]
	for rows.Next() {
		var event Event[any]
		var payloadStr, metadataStr string
		var timestampUnix int64
		var sessionID, userID sql.NullString

		err := rows.Scan(
			&event.ID,
			&event.Type,
			&payloadStr,
			&timestampUnix,
			&sessionID,
			&userID,
			&metadataStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		// Parse timestamp
		event.Timestamp = time.Unix(timestampUnix, 0)

		// Parse session and user IDs
		if sessionID.Valid {
			event.SessionID = sessionID.String
		}
		if userID.Valid {
			event.UserID = userID.String
		}

		// Parse payload
		if err := json.Unmarshal([]byte(payloadStr), &event.Payload); err != nil {
			return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
		}

		// Parse metadata
		if metadataStr != "" {
			if err := json.Unmarshal([]byte(metadataStr), &event.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		// Apply filter if provided
		if filter == nil || filter(event) {
			events = append(events, event)
		}
	}

	return events, nil
}

// Delete removes an event from the database
func (d *DatabasePersistenceStore) Delete(eventID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `DELETE FROM events WHERE id = ?`
	_, err := d.db.Exec(query, eventID)
	return err
}

// Cleanup removes old events from the database
func (d *DatabasePersistenceStore) Cleanup(olderThan time.Time) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	query := `DELETE FROM events WHERE timestamp < ?`
	_, err := d.db.Exec(query, olderThan.Unix())
	return err
}

// GetEventsForSession retrieves events for a specific session since a timestamp
func (d *DatabasePersistenceStore) GetEventsForSession(sessionID string, since time.Time) ([]Event[any], error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
	SELECT id, type, payload, timestamp, session_id, user_id, metadata
	FROM events
	WHERE session_id = ? AND timestamp > ?
	ORDER BY timestamp ASC
	`

	return d.queryEvents(query, sessionID, since.Unix())
}

// GetEventsForUser retrieves events for a specific user since a timestamp
func (d *DatabasePersistenceStore) GetEventsForUser(userID string, since time.Time) ([]Event[any], error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
	SELECT id, type, payload, timestamp, session_id, user_id, metadata
	FROM events
	WHERE user_id = ? AND timestamp > ?
	ORDER BY timestamp ASC
	`

	return d.queryEvents(query, userID, since.Unix())
}

// GetEventsByType retrieves events of a specific type since a timestamp
func (d *DatabasePersistenceStore) GetEventsByType(eventType EventType, since time.Time) ([]Event[any], error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
	SELECT id, type, payload, timestamp, session_id, user_id, metadata
	FROM events
	WHERE type = ? AND timestamp > ?
	ORDER BY timestamp ASC
	`

	return d.queryEvents(query, string(eventType), since.Unix())
}

// GetEventsSince retrieves events since a specific timestamp
func (d *DatabasePersistenceStore) GetEventsSince(since time.Time) ([]Event[any], error) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	query := `
	SELECT id, type, payload, timestamp, session_id, user_id, metadata
	FROM events
	WHERE timestamp > ?
	ORDER BY timestamp ASC
	`

	return d.queryEvents(query, since.Unix())
}

// queryEvents is a helper method to execute queries and parse results
func (d *DatabasePersistenceStore) queryEvents(query string, args ...any) ([]Event[any], error) {
	rows, err := d.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []Event[any]
	for rows.Next() {
		var event Event[any]
		var payloadStr, metadataStr string
		var timestampUnix int64
		var sessionID, userID sql.NullString

		err := rows.Scan(
			&event.ID,
			&event.Type,
			&payloadStr,
			&timestampUnix,
			&sessionID,
			&userID,
			&metadataStr,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}

		// Parse timestamp
		event.Timestamp = time.Unix(timestampUnix, 0)

		// Parse session and user IDs
		if sessionID.Valid {
			event.SessionID = sessionID.String
		}
		if userID.Valid {
			event.UserID = userID.String
		}

		// Parse payload
		if err := json.Unmarshal([]byte(payloadStr), &event.Payload); err != nil {
			return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
		}

		// Parse metadata
		if metadataStr != "" {
			if err := json.Unmarshal([]byte(metadataStr), &event.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		events = append(events, event)
	}

	return events, nil
}

// GetStats returns statistics about the database persistence store
func (d *DatabasePersistenceStore) GetStats() PersistenceStats {
	d.mu.RLock()
	defer d.mu.RUnlock()

	var stats PersistenceStats

	// Get total event count
	row := d.db.QueryRow("SELECT COUNT(*) FROM events")
	if err := row.Scan(&stats.TotalEvents); err != nil {
		stats.TotalEvents = 0
	}

	// Get oldest and newest event timestamps
	row = d.db.QueryRow("SELECT MIN(timestamp), MAX(timestamp) FROM events")
	var oldestUnix, newestUnix sql.NullInt64
	if err := row.Scan(&oldestUnix, &newestUnix); err == nil {
		if oldestUnix.Valid {
			stats.OldestEvent = time.Unix(oldestUnix.Int64, 0)
		}
		if newestUnix.Valid {
			stats.NewestEvent = time.Unix(newestUnix.Int64, 0)
		}
	}

	// Database doesn't have a max size limit like memory store
	stats.MaxSize = -1

	return stats
}
