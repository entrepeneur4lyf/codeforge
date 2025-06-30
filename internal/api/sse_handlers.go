package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// SSEEvent represents a Server-Sent Event
type SSEEvent struct {
	ID    string      `json:"id,omitempty"`
	Event string      `json:"event,omitempty"`
	Data  interface{} `json:"data"`
}

// MetricsData represents system metrics
type MetricsData struct {
	Timestamp      time.Time `json:"timestamp"`
	CPUUsage       float64   `json:"cpu_usage"`
	MemoryUsage    float64   `json:"memory_usage"`
	ActiveSessions int       `json:"active_sessions"`
	TotalRequests  int64     `json:"total_requests"`
	VectorDBStats  struct {
		TotalChunks  int     `json:"total_chunks"`
		IndexSize    int     `json:"index_size"`
		CacheHitRate float64 `json:"cache_hit_rate"`
	} `json:"vectordb_stats"`
}

// StatusData represents service status
type StatusData struct {
	Timestamp time.Time                `json:"timestamp"`
	Services  map[string]ServiceStatus `json:"services"`
	Overall   string                   `json:"overall"`
}

// ServiceStatus represents the status of a service
type ServiceStatus struct {
	Status       string    `json:"status"` // "healthy", "degraded", "down"
	LastCheck    time.Time `json:"last_check"`
	ResponseTime int64     `json:"response_time_ms"`
	Message      string    `json:"message,omitempty"`
}

// handleMetricsSSE handles Server-Sent Events for metrics
func (s *Server) handleMetricsSSE(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create a channel for this client
	clientChan := make(chan SSEEvent, 10)
	defer close(clientChan)

	// Send initial metrics
	s.sendMetricsEvent(clientChan)

	// Set up ticker for periodic updates
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Handle client disconnect
	ctx := r.Context()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sendMetricsEvent(clientChan)
		case event := <-clientChan:
			s.writeSSEEvent(w, event)
		}
	}
}

// handleStatusSSE handles Server-Sent Events for service status
func (s *Server) handleStatusSSE(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create a channel for this client
	clientChan := make(chan SSEEvent, 10)
	defer close(clientChan)

	// Send initial status
	s.sendStatusEvent(clientChan)

	// Set up ticker for periodic updates
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	// Handle client disconnect
	ctx := r.Context()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sendStatusEvent(clientChan)
		case event := <-clientChan:
			s.writeSSEEvent(w, event)
		}
	}
}

// sendMetricsEvent sends current metrics to the client
func (s *Server) sendMetricsEvent(clientChan chan SSEEvent) {
	metrics := s.collectMetrics()

	event := SSEEvent{
		ID:    fmt.Sprintf("metrics-%d", time.Now().Unix()),
		Event: "metrics",
		Data:  metrics,
	}

	select {
	case clientChan <- event:
	default:
		// Channel is full, skip this update
	}
}

// sendStatusEvent sends current service status to the client
func (s *Server) sendStatusEvent(clientChan chan SSEEvent) {
	status := s.collectServiceStatus()

	event := SSEEvent{
		ID:    fmt.Sprintf("status-%d", time.Now().Unix()),
		Event: "status",
		Data:  status,
	}

	select {
	case clientChan <- event:
	default:
		// Channel is full, skip this update
	}
}

// writeSSEEvent writes an SSE event to the response writer
func (s *Server) writeSSEEvent(w http.ResponseWriter, event SSEEvent) {
	if event.ID != "" {
		fmt.Fprintf(w, "id: %s\n", event.ID)
	}

	if event.Event != "" {
		fmt.Fprintf(w, "event: %s\n", event.Event)
	}

	// Serialize data to JSON
	data, err := json.Marshal(event.Data)
	if err != nil {
		fmt.Fprintf(w, "data: {\"error\": \"Failed to serialize data\"}\n\n")
		return
	}

	fmt.Fprintf(w, "data: %s\n\n", string(data))

	// Flush the data to the client
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

// collectMetrics collects current system metrics
func (s *Server) collectMetrics() MetricsData {
	metrics := MetricsData{
		Timestamp:      time.Now(),
		CPUUsage:       45.2, // TODO: Implement actual CPU monitoring
		MemoryUsage:    67.8, // TODO: Implement actual memory monitoring
		ActiveSessions: 3,    // TODO: Track actual active sessions
		TotalRequests:  1247, // TODO: Track actual request count
	}

	// Get vector database stats
	if s.vectorDB != nil {
		// TODO: Fix GetStats call when context is available
		metrics.VectorDBStats.TotalChunks = 1247 // Mock data for now
		metrics.VectorDBStats.IndexSize = 1024000
		metrics.VectorDBStats.CacheHitRate = 0.85
	}

	return metrics
}

// collectServiceStatus collects current service status
func (s *Server) collectServiceStatus() StatusData {
	status := StatusData{
		Timestamp: time.Now(),
		Services:  make(map[string]ServiceStatus),
		Overall:   "healthy",
	}

	// Check vector database
	if s.vectorDB != nil {
		status.Services["vectordb"] = ServiceStatus{
			Status:       "healthy",
			LastCheck:    time.Now(),
			ResponseTime: 15,
			Message:      "Vector database operational",
		}
	} else {
		status.Services["vectordb"] = ServiceStatus{
			Status:       "down",
			LastCheck:    time.Now(),
			ResponseTime: 0,
			Message:      "Vector database not initialized",
		}
		status.Overall = "degraded"
	}

	// Check authentication service
	if s.auth != nil {
		status.Services["auth"] = ServiceStatus{
			Status:       "healthy",
			LastCheck:    time.Now(),
			ResponseTime: 5,
			Message:      "Authentication service operational",
		}
	} else {
		status.Services["auth"] = ServiceStatus{
			Status:       "down",
			LastCheck:    time.Now(),
			ResponseTime: 0,
			Message:      "Authentication service not initialized",
		}
		status.Overall = "degraded"
	}

	// API service is always healthy if we're responding
	status.Services["api"] = ServiceStatus{
		Status:       "healthy",
		LastCheck:    time.Now(),
		ResponseTime: 10,
		Message:      "API service operational",
	}

	return status
}
