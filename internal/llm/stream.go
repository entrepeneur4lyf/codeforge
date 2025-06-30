package llm

import (
	"context"
	"time"
)

// ApiStream represents a stream of API response chunks
// Based on Cline's ApiStream design from transform/stream.ts
type ApiStream <-chan ApiStreamChunk

// ApiStreamChunk represents different types of streaming responses
// Translated from Cline's TypeScript types
type ApiStreamChunk interface {
	Type() string
}

// ApiStreamTextChunk represents text content in the stream
type ApiStreamTextChunk struct {
	Text string `json:"text"`
}

func (c ApiStreamTextChunk) Type() string { return "text" }

// ApiStreamReasoningChunk represents reasoning/thinking content
type ApiStreamReasoningChunk struct {
	Reasoning string `json:"reasoning"`
}

func (c ApiStreamReasoningChunk) Type() string { return "reasoning" }

// ApiStreamUsageChunk represents token usage and cost information
type ApiStreamUsageChunk struct {
	InputTokens        int     `json:"inputTokens"`
	OutputTokens       int     `json:"outputTokens"`
	CacheWriteTokens   *int    `json:"cacheWriteTokens,omitempty"`
	CacheReadTokens    *int    `json:"cacheReadTokens,omitempty"`
	ThoughtsTokenCount *int    `json:"thoughtsTokenCount,omitempty"` // OpenRouter
	TotalCost          *float64 `json:"totalCost,omitempty"`          // OpenRouter
}

func (c ApiStreamUsageChunk) Type() string { return "usage" }

// StreamCollector helps collect and aggregate stream chunks
type StreamCollector struct {
	TextChunks     []string
	ReasoningChunks []string
	Usage          *ApiStreamUsageChunk
	StartTime      time.Time
	EndTime        time.Time
}

// NewStreamCollector creates a new stream collector
func NewStreamCollector() *StreamCollector {
	return &StreamCollector{
		TextChunks:      make([]string, 0),
		ReasoningChunks: make([]string, 0),
		StartTime:       time.Now(),
	}
}

// Collect processes a stream chunk and adds it to the collector
func (sc *StreamCollector) Collect(chunk ApiStreamChunk) {
	switch c := chunk.(type) {
	case ApiStreamTextChunk:
		sc.TextChunks = append(sc.TextChunks, c.Text)
	case ApiStreamReasoningChunk:
		sc.ReasoningChunks = append(sc.ReasoningChunks, c.Reasoning)
	case ApiStreamUsageChunk:
		sc.Usage = &c
		sc.EndTime = time.Now()
	}
}

// GetFullText returns the complete text from all text chunks
func (sc *StreamCollector) GetFullText() string {
	result := ""
	for _, chunk := range sc.TextChunks {
		result += chunk
	}
	return result
}

// GetFullReasoning returns the complete reasoning from all reasoning chunks
func (sc *StreamCollector) GetFullReasoning() string {
	result := ""
	for _, chunk := range sc.ReasoningChunks {
		result += chunk
	}
	return result
}

// GetDuration returns the total duration of the stream
func (sc *StreamCollector) GetDuration() time.Duration {
	if sc.EndTime.IsZero() {
		return time.Since(sc.StartTime)
	}
	return sc.EndTime.Sub(sc.StartTime)
}

// StreamProcessor provides utilities for processing streams
type StreamProcessor struct {
	ctx context.Context
}

// NewStreamProcessor creates a new stream processor
func NewStreamProcessor(ctx context.Context) *StreamProcessor {
	return &StreamProcessor{ctx: ctx}
}

// ProcessStream processes an entire stream and returns the collected result
func (sp *StreamProcessor) ProcessStream(stream ApiStream) (*StreamCollector, error) {
	collector := NewStreamCollector()
	
	for {
		select {
		case chunk, ok := <-stream:
			if !ok {
				// Stream closed
				return collector, nil
			}
			collector.Collect(chunk)
		case <-sp.ctx.Done():
			return collector, sp.ctx.Err()
		}
	}
}

// ProcessStreamWithCallback processes a stream and calls a callback for each chunk
func (sp *StreamProcessor) ProcessStreamWithCallback(
	stream ApiStream,
	callback func(ApiStreamChunk) error,
) (*StreamCollector, error) {
	collector := NewStreamCollector()
	
	for {
		select {
		case chunk, ok := <-stream:
			if !ok {
				// Stream closed
				return collector, nil
			}
			collector.Collect(chunk)
			if err := callback(chunk); err != nil {
				return collector, err
			}
		case <-sp.ctx.Done():
			return collector, sp.ctx.Err()
		}
	}
}

// StreamBuffer provides buffering capabilities for streams
type StreamBuffer struct {
	chunks []ApiStreamChunk
	closed bool
}

// NewStreamBuffer creates a new stream buffer
func NewStreamBuffer() *StreamBuffer {
	return &StreamBuffer{
		chunks: make([]ApiStreamChunk, 0),
		closed: false,
	}
}

// Add adds a chunk to the buffer
func (sb *StreamBuffer) Add(chunk ApiStreamChunk) {
	if !sb.closed {
		sb.chunks = append(sb.chunks, chunk)
	}
}

// Close marks the buffer as closed
func (sb *StreamBuffer) Close() {
	sb.closed = true
}

// ToChannel converts the buffer to a channel
func (sb *StreamBuffer) ToChannel() ApiStream {
	ch := make(chan ApiStreamChunk, len(sb.chunks))
	
	go func() {
		defer close(ch)
		for _, chunk := range sb.chunks {
			ch <- chunk
		}
	}()
	
	return ch
}

// GetChunks returns all chunks in the buffer
func (sb *StreamBuffer) GetChunks() []ApiStreamChunk {
	return sb.chunks
}

// IsClosed returns whether the buffer is closed
func (sb *StreamBuffer) IsClosed() bool {
	return sb.closed
}
