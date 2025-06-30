package context

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// TokenCounter provides token counting functionality for different models
type TokenCounter struct {
	// Cache for token counts to avoid recalculation
	cache map[string]int
}

// NewTokenCounter creates a new token counter
func NewTokenCounter() *TokenCounter {
	return &TokenCounter{
		cache: make(map[string]int),
	}
}

// TokenUsage represents token usage for a conversation or message
type TokenUsage struct {
	InputTokens      int `json:"input_tokens"`
	OutputTokens     int `json:"output_tokens"`
	CacheReadTokens  int `json:"cache_read_tokens"`
	CacheWriteTokens int `json:"cache_write_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// Add combines two token usages
func (tu *TokenUsage) Add(other TokenUsage) {
	tu.InputTokens += other.InputTokens
	tu.OutputTokens += other.OutputTokens
	tu.CacheReadTokens += other.CacheReadTokens
	tu.CacheWriteTokens += other.CacheWriteTokens
	tu.TotalTokens += other.TotalTokens
}

// CountTokens estimates token count for text using multiple methods
func (tc *TokenCounter) CountTokens(text string, model string) int {
	// Create cache key
	cacheKey := fmt.Sprintf("%s:%s", model, hashString(text))

	// Check cache first
	if count, exists := tc.cache[cacheKey]; exists {
		return count
	}

	var count int

	// Use model-specific counting if available
	switch {
	case strings.Contains(strings.ToLower(model), "gpt"):
		count = tc.countGPTTokens(text)
	case strings.Contains(strings.ToLower(model), "claude"):
		count = tc.countClaudeTokens(text)
	case strings.Contains(strings.ToLower(model), "gemini"):
		count = tc.countGeminiTokens(text)
	default:
		count = tc.countGenericTokens(text)
	}

	// Cache the result
	tc.cache[cacheKey] = count
	return count
}

// countGPTTokens estimates tokens for GPT models
func (tc *TokenCounter) countGPTTokens(text string) int {
	// GPT tokenization approximation: ~4 characters per token
	// More accurate for English text

	// Remove extra whitespace
	text = strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(text, " "))

	// Count characters
	charCount := utf8.RuneCountInString(text)

	// Estimate tokens (GPT models average ~4 chars per token)
	tokenCount := charCount / 4

	// Add some tokens for special characters and formatting
	specialChars := len(regexp.MustCompile(`[{}[\]().,;:!?'"<>]`).FindAllString(text, -1))
	tokenCount += specialChars / 2

	// Minimum 1 token for non-empty text
	if charCount > 0 && tokenCount == 0 {
		tokenCount = 1
	}

	return tokenCount
}

// countClaudeTokens estimates tokens for Claude models
func (tc *TokenCounter) countClaudeTokens(text string) int {
	// Claude tokenization is similar to GPT but slightly different
	// Approximately 3.5-4 characters per token

	text = strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(text, " "))
	charCount := utf8.RuneCountInString(text)

	// Claude tends to be slightly more efficient
	tokenCount := charCount * 10 / 35 // ~3.5 chars per token

	// Add tokens for code blocks and special formatting
	codeBlocks := len(regexp.MustCompile("```").FindAllString(text, -1))
	tokenCount += codeBlocks * 2

	if charCount > 0 && tokenCount == 0 {
		tokenCount = 1
	}

	return tokenCount
}

// countGeminiTokens estimates tokens for Gemini models
func (tc *TokenCounter) countGeminiTokens(text string) int {
	// Gemini tokenization approximation
	text = strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(text, " "))
	charCount := utf8.RuneCountInString(text)

	// Gemini averages around 4 characters per token
	tokenCount := charCount / 4

	if charCount > 0 && tokenCount == 0 {
		tokenCount = 1
	}

	return tokenCount
}

// countGenericTokens provides a generic token counting method
func (tc *TokenCounter) countGenericTokens(text string) int {
	// Generic approximation: split by whitespace and punctuation
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}

	// Split by whitespace
	words := strings.Fields(text)
	tokenCount := len(words)

	// Add tokens for punctuation and special characters
	punctuation := len(regexp.MustCompile(`[.,;:!?(){}[\]"']`).FindAllString(text, -1))
	tokenCount += punctuation / 3

	return tokenCount
}

// CountConversationTokens counts tokens for an entire conversation
func (tc *TokenCounter) CountConversationTokens(messages []ConversationMessage, model string) TokenUsage {
	usage := TokenUsage{}

	for _, msg := range messages {
		tokens := tc.CountMessageTokens(msg, model)
		usage.Add(tokens)
	}

	return usage
}

// CountMessageTokens counts tokens for a single message
func (tc *TokenCounter) CountMessageTokens(message ConversationMessage, model string) TokenUsage {
	usage := TokenUsage{}

	// Count content tokens
	contentTokens := tc.CountTokens(message.Content, model)

	// Add role and metadata overhead (typically 3-5 tokens per message)
	overhead := 4

	switch message.Role {
	case "user":
		usage.InputTokens = contentTokens + overhead
	case "assistant":
		usage.OutputTokens = contentTokens + overhead
	default:
		usage.InputTokens = contentTokens + overhead
	}

	usage.TotalTokens = usage.InputTokens + usage.OutputTokens

	return usage
}

// ConversationMessage represents a message in a conversation
type ConversationMessage struct {
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp int64                  `json:"timestamp"`
}

// EstimateContextSize estimates the total context size including system prompts
func (tc *TokenCounter) EstimateContextSize(messages []ConversationMessage, systemPrompt string, model string) int {
	conversationTokens := tc.CountConversationTokens(messages, model)
	systemTokens := tc.CountTokens(systemPrompt, model)

	// Add some overhead for message formatting and API structure
	overhead := len(messages) * 3

	return conversationTokens.TotalTokens + systemTokens + overhead
}

// ClearCache clears the token counting cache
func (tc *TokenCounter) ClearCache() {
	tc.cache = make(map[string]int)
}

// GetCacheSize returns the number of cached token counts
func (tc *TokenCounter) GetCacheSize() int {
	return len(tc.cache)
}

// hashString creates a simple hash of a string for caching
func hashString(s string) string {
	// Simple hash for cache keys - not cryptographic
	hash := 0
	for _, char := range s {
		hash = hash*31 + int(char)
	}
	return fmt.Sprintf("%x", hash)
}

// TokenCountResult represents the result of token counting with metadata
type TokenCountResult struct {
	Count     int    `json:"count"`
	Model     string `json:"model"`
	Method    string `json:"method"`
	Cached    bool   `json:"cached"`
	TextHash  string `json:"text_hash"`
	Timestamp int64  `json:"timestamp"`
}

// CountWithMetadata returns detailed token counting information
func (tc *TokenCounter) CountWithMetadata(text string, model string) TokenCountResult {
	textHash := hashString(text)
	cacheKey := fmt.Sprintf("%s:%s", model, textHash)

	cached := false
	if _, exists := tc.cache[cacheKey]; exists {
		cached = true
	}

	count := tc.CountTokens(text, model)

	method := "generic"
	switch {
	case strings.Contains(strings.ToLower(model), "gpt"):
		method = "gpt"
	case strings.Contains(strings.ToLower(model), "claude"):
		method = "claude"
	case strings.Contains(strings.ToLower(model), "gemini"):
		method = "gemini"
	}

	return TokenCountResult{
		Count:     count,
		Model:     model,
		Method:    method,
		Cached:    cached,
		TextHash:  textHash,
		Timestamp: getCurrentTimestamp(),
	}
}

// getCurrentTimestamp returns current Unix timestamp
func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}
