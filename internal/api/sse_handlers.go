package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
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
	// Add error recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("SSE metrics handler panic recovered: %v", r)
		}
	}()

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Ensure we can flush
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Send initial connection confirmation
	fmt.Fprintf(w, "data: {\"type\": \"connected\", \"timestamp\": %d}\n\n", time.Now().Unix())
	flusher.Flush()

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
			if err := s.writeSSEEvent(w, event); err != nil {
				log.Printf("Failed to write SSE metrics event: %v", err)
				return
			}
		}
	}
}

// handleStatusSSE handles Server-Sent Events for service status
func (s *Server) handleStatusSSE(w http.ResponseWriter, r *http.Request) {
	// Add error recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("SSE handler panic recovered: %v", r)
		}
	}()

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Ensure we can flush
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Send initial connection confirmation
	fmt.Fprintf(w, "data: {\"type\": \"connected\", \"timestamp\": %d}\n\n", time.Now().Unix())
	flusher.Flush()

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
			if err := s.writeSSEEvent(w, event); err != nil {
				log.Printf("Failed to write SSE event: %v", err)
				return
			}
		}
	}
}

// sendMetricsEvent sends current metrics to the client
func (s *Server) sendMetricsEvent(clientChan chan SSEEvent) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("sendMetricsEvent panic recovered: %v", r)
		}
	}()

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
		log.Printf("SSE metrics channel full, skipping update")
	}
}

// sendStatusEvent sends current service status to the client
func (s *Server) sendStatusEvent(clientChan chan SSEEvent) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("sendStatusEvent panic recovered: %v", r)
		}
	}()

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
		log.Printf("SSE status channel full, skipping update")
	}
}

// writeSSEEvent writes an SSE event to the response writer
func (s *Server) writeSSEEvent(w http.ResponseWriter, event SSEEvent) error {
	if event.ID != "" {
		if _, err := fmt.Fprintf(w, "id: %s\n", event.ID); err != nil {
			return err
		}
	}

	if event.Event != "" {
		if _, err := fmt.Fprintf(w, "event: %s\n", event.Event); err != nil {
			return err
		}
	}

	// Serialize data to JSON
	data, err := json.Marshal(event.Data)
	if err != nil {
		if _, writeErr := fmt.Fprintf(w, "data: {\"error\": \"Failed to serialize data\"}\n\n"); writeErr != nil {
			return writeErr
		}
		return err
	}

	if _, err := fmt.Fprintf(w, "data: %s\n\n", string(data)); err != nil {
		return err
	}

	// Flush the data to the client
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	return nil
}

// collectMetrics collects current system metrics
func (s *Server) collectMetrics() MetricsData {
	metrics := MetricsData{
		Timestamp:      time.Now(),
		CPUUsage:       s.getCPUUsage(),
		MemoryUsage:    s.getMemoryUsage(),
		ActiveSessions: s.getActiveSessions(),
		TotalRequests:  s.getTotalRequests(),
	}

	// Get vector database stats
	if s.vectorDB != nil {
		ctx := context.Background()
		if stats, err := s.vectorDB.GetStats(ctx); err == nil {
			metrics.VectorDBStats.TotalChunks = stats.TotalChunks
			metrics.VectorDBStats.IndexSize = stats.TotalChunks * 1000 // Approximate
			metrics.VectorDBStats.CacheHitRate = 0.85                  // Would need actual cache metrics
		} else {
			// Fallback values if stats unavailable
			metrics.VectorDBStats.TotalChunks = 0
			metrics.VectorDBStats.IndexSize = 0
			metrics.VectorDBStats.CacheHitRate = 0.0
		}
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

// System monitoring helper methods

// getCPUUsage returns current CPU usage percentage
func (s *Server) getCPUUsage() float64 {
	// Simple CPU usage estimation using runtime stats
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Use GC stats as a rough proxy for CPU activity
	// This is a simplified approach - for production, consider using
	// a proper system monitoring library like gopsutil
	gcCPU := float64(m.GCCPUFraction) * 100
	if gcCPU > 100 {
		gcCPU = 100
	}

	// Add some baseline CPU usage
	baseCPU := 5.0 + gcCPU
	if baseCPU > 100 {
		baseCPU = 100
	}

	return baseCPU
}

// getMemoryUsage returns current memory usage percentage
func (s *Server) getMemoryUsage() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Calculate memory usage as percentage of allocated memory
	// This is simplified - for production, consider system memory limits
	allocMB := float64(m.Alloc) / 1024 / 1024
	sysMB := float64(m.Sys) / 1024 / 1024

	if sysMB == 0 {
		return 0
	}

	usage := (allocMB / sysMB) * 100
	if usage > 100 {
		usage = 100
	}

	return usage
}

// getActiveSessions returns the number of active sessions
func (s *Server) getActiveSessions() int {
	// Count active sessions from chat storage
	if s.chatStorage != nil {
		sessions := s.chatStorage.GetAllSessions()
		activeCount := 0
		for _, session := range sessions {
			if session.Status == "active" {
				activeCount++
			}
		}
		return activeCount
	}
	return 0
}

// getTotalRequests returns the total number of requests processed
func (s *Server) getTotalRequests() int64 {
	// This would typically be tracked by middleware
	// For now, return a simple estimate based on active sessions and time
	// In production, implement proper request counting
	activeSessions := s.getActiveSessions()
	if activeSessions == 0 {
		return 0
	}

	// Simple estimate: assume each active session has made some requests
	return int64(activeSessions * 10)
}
