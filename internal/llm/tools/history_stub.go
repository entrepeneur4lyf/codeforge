package tools

import (
	"context"
	"errors"
	"sync"
	"time"
)

// FileHistory represents a file's history entry
type FileHistory struct {
	ID        string
	SessionID string
	Path      string
	Content   string
	Version   string
	CreatedAt time.Time
}

// HistoryService provides a simple in-memory file history service
type HistoryService struct {
	mutex   sync.RWMutex
	history map[string][]FileHistory // path -> versions
}

// NewHistoryService creates a new history service
func NewHistoryService() *HistoryService {
	return &HistoryService{
		history: make(map[string][]FileHistory),
	}
}

// Create creates a new file history entry
func (h *HistoryService) Create(ctx context.Context, sessionID, path, content string) (FileHistory, error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	entry := FileHistory{
		ID:        generateID(),
		SessionID: sessionID,
		Path:      path,
		Content:   content,
		Version:   "initial",
		CreatedAt: time.Now(),
	}

	h.history[path] = append(h.history[path], entry)
	return entry, nil
}

// CreateVersion creates a new version of a file
func (h *HistoryService) CreateVersion(ctx context.Context, sessionID, path, content string) (FileHistory, error) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	versions := h.history[path]
	version := "v1"
	if len(versions) > 0 {
		version = "v" + string(rune(len(versions)+1))
	}

	entry := FileHistory{
		ID:        generateID(),
		SessionID: sessionID,
		Path:      path,
		Content:   content,
		Version:   version,
		CreatedAt: time.Now(),
	}

	h.history[path] = append(h.history[path], entry)
	return entry, nil
}

// GetByPathAndSession gets the latest version of a file for a session
func (h *HistoryService) GetByPathAndSession(ctx context.Context, path, sessionID string) (FileHistory, error) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	versions := h.history[path]
	for i := len(versions) - 1; i >= 0; i-- {
		if versions[i].SessionID == sessionID {
			return versions[i], nil
		}
	}

	return FileHistory{}, errors.New("file not found")
}

func generateID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// SaveFileVersion is a convenience method that creates or updates a file version
func (h *HistoryService) SaveFileVersion(path, content string) {
	ctx := context.Background()
	sessionID := "default"
	
	h.mutex.Lock()
	versions := h.history[path]
	h.mutex.Unlock()
	
	if len(versions) == 0 {
		_, _ = h.Create(ctx, sessionID, path, content)
	} else {
		_, _ = h.CreateVersion(ctx, sessionID, path, content)
	}
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}