package context

import (
	"log"

	"github.com/entrepeneur4lyf/codeforge/internal/config"
)

// SlidingWindow manages conversation context using a sliding window approach
type SlidingWindow struct {
	tokenCounter *TokenCounter
	config       *config.Config
}

// NewSlidingWindow creates a new sliding window manager
func NewSlidingWindow(cfg *config.Config) *SlidingWindow {
	return &SlidingWindow{
		tokenCounter: NewTokenCounter(),
		config:       cfg,
	}
}

// WindowResult represents the result of applying sliding window
type WindowResult struct {
	Messages         []ConversationMessage `json:"messages"`
	OriginalCount    int                   `json:"original_count"`
	FinalCount       int                   `json:"final_count"`
	TokensRemoved    int                   `json:"tokens_removed"`
	TokensRetained   int                   `json:"tokens_retained"`
	WindowsApplied   int                   `json:"windows_applied"`
	OverlapTokens    int                   `json:"overlap_tokens"`
	CompressionRatio float64               `json:"compression_ratio"`
}

// ApplyWindow applies sliding window to conversation messages
func (sw *SlidingWindow) ApplyWindow(messages []ConversationMessage, modelID string) (*WindowResult, error) {
	if len(messages) == 0 {
		return &WindowResult{
			Messages:      messages,
			OriginalCount: 0,
			FinalCount:    0,
		}, nil
	}

	contextConfig := sw.config.GetContextConfig()
	if !contextConfig.SlidingWindow {
		// Sliding window disabled, return original messages
		return &WindowResult{
			Messages:      messages,
			OriginalCount: len(messages),
			FinalCount:    len(messages),
		}, nil
	}

	modelConfig := sw.config.GetModelConfig(modelID)
	originalUsage := sw.tokenCounter.CountConversationTokens(messages, modelID)

	// Check if we need to apply sliding window
	if originalUsage.TotalTokens <= modelConfig.ContextWindow {
		return &WindowResult{
			Messages:       messages,
			OriginalCount:  len(messages),
			FinalCount:     len(messages),
			TokensRetained: originalUsage.TotalTokens,
		}, nil
	}

	log.Printf("Applying sliding window for model %s, original tokens: %d, limit: %d",
		modelID, originalUsage.TotalTokens, modelConfig.ContextWindow)

	// Find the last summary to preserve context continuity
	lastSummaryIndex := sw.findLastSummaryIndex(messages)

	// Always keep system messages and summaries
	preservedMessages := sw.getPreservedMessages(messages, lastSummaryIndex)
	preservedTokens := sw.tokenCounter.CountConversationTokens(preservedMessages, modelID).TotalTokens

	// Calculate available tokens for recent messages
	availableTokens := modelConfig.ContextWindow - preservedTokens
	overlapTokens := contextConfig.WindowOverlap

	// Get recent messages that fit in the window
	recentMessages := sw.getRecentMessages(messages, lastSummaryIndex+1, availableTokens, overlapTokens, modelID)

	// Combine preserved and recent messages
	finalMessages := append(preservedMessages, recentMessages...)
	finalUsage := sw.tokenCounter.CountConversationTokens(finalMessages, modelID)

	result := &WindowResult{
		Messages:         finalMessages,
		OriginalCount:    len(messages),
		FinalCount:       len(finalMessages),
		TokensRemoved:    originalUsage.TotalTokens - finalUsage.TotalTokens,
		TokensRetained:   finalUsage.TotalTokens,
		WindowsApplied:   1,
		OverlapTokens:    overlapTokens,
		CompressionRatio: float64(finalUsage.TotalTokens) / float64(originalUsage.TotalTokens),
	}

	log.Printf("Sliding window applied: %d -> %d messages, %d -> %d tokens (%.2f%% retained)",
		result.OriginalCount, result.FinalCount,
		originalUsage.TotalTokens, finalUsage.TotalTokens,
		result.CompressionRatio*100)

	return result, nil
}

// findLastSummaryIndex finds the most recent summary message
func (sw *SlidingWindow) findLastSummaryIndex(messages []ConversationMessage) int {
	for i := len(messages) - 1; i >= 0; i-- {
		if metadata, ok := messages[i].Metadata["summary"]; ok {
			if isSummary, ok := metadata.(bool); ok && isSummary {
				return i
			}
		}
	}
	return -1
}

// getPreservedMessages returns messages that should always be preserved
func (sw *SlidingWindow) getPreservedMessages(messages []ConversationMessage, lastSummaryIndex int) []ConversationMessage {
	var preserved []ConversationMessage

	// Preserve system messages and summaries
	for i := 0; i <= lastSummaryIndex && i < len(messages); i++ {
		msg := messages[i]
		if msg.Role == "system" || sw.isSummaryMessage(msg) {
			preserved = append(preserved, msg)
		}
	}

	return preserved
}

// getRecentMessages returns the most recent messages that fit in available tokens
func (sw *SlidingWindow) getRecentMessages(messages []ConversationMessage, startIndex, availableTokens, overlapTokens int, modelID string) []ConversationMessage {
	if startIndex >= len(messages) {
		return []ConversationMessage{}
	}

	recentMessages := messages[startIndex:]
	if len(recentMessages) == 0 {
		return []ConversationMessage{}
	}

	// Start from the end and work backwards to fit in available tokens
	var selected []ConversationMessage
	currentTokens := 0

	for i := len(recentMessages) - 1; i >= 0; i-- {
		msg := recentMessages[i]
		msgTokens := sw.tokenCounter.CountMessageTokens(msg, modelID).TotalTokens

		if currentTokens+msgTokens <= availableTokens {
			selected = append([]ConversationMessage{msg}, selected...)
			currentTokens += msgTokens
		} else {
			break
		}
	}

	// Apply overlap if we have more messages available
	if len(selected) < len(recentMessages) {
		selected = sw.applyOverlap(recentMessages, selected, overlapTokens, modelID)
	}

	return selected
}

// applyOverlap adds overlap messages to maintain context continuity
func (sw *SlidingWindow) applyOverlap(allMessages, selectedMessages []ConversationMessage, overlapTokens int, modelID string) []ConversationMessage {
	if overlapTokens <= 0 || len(selectedMessages) == 0 {
		return selectedMessages
	}

	// Find the index of the first selected message in allMessages
	firstSelectedIndex := -1
	for i, msg := range allMessages {
		if len(selectedMessages) > 0 && msg.Timestamp == selectedMessages[0].Timestamp {
			firstSelectedIndex = i
			break
		}
	}

	if firstSelectedIndex <= 0 {
		return selectedMessages
	}

	// Add overlap messages before the selected range
	var overlapMsgs []ConversationMessage
	currentOverlapTokens := 0

	for i := firstSelectedIndex - 1; i >= 0 && currentOverlapTokens < overlapTokens; i-- {
		msg := allMessages[i]
		msgTokens := sw.tokenCounter.CountMessageTokens(msg, modelID).TotalTokens

		if currentOverlapTokens+msgTokens <= overlapTokens {
			overlapMsgs = append([]ConversationMessage{msg}, overlapMsgs...)
			currentOverlapTokens += msgTokens
		} else {
			break
		}
	}

	return append(overlapMsgs, selectedMessages...)
}

// isSummaryMessage checks if a message is a summary
func (sw *SlidingWindow) isSummaryMessage(msg ConversationMessage) bool {
	if metadata, ok := msg.Metadata["summary"]; ok {
		if isSummary, ok := metadata.(bool); ok {
			return isSummary
		}
	}
	return false
}

// OptimizeWindow optimizes the sliding window for better context retention
func (sw *SlidingWindow) OptimizeWindow(messages []ConversationMessage, modelID string) (*WindowResult, error) {
	// First apply basic sliding window
	result, err := sw.ApplyWindow(messages, modelID)
	if err != nil {
		return nil, err
	}

	// Apply additional optimizations
	optimizedMessages := sw.optimizeMessageSelection(result.Messages, modelID)

	if len(optimizedMessages) != len(result.Messages) {
		// Recalculate metrics
		originalUsage := sw.tokenCounter.CountConversationTokens(messages, modelID)
		optimizedUsage := sw.tokenCounter.CountConversationTokens(optimizedMessages, modelID)

		result.Messages = optimizedMessages
		result.FinalCount = len(optimizedMessages)
		result.TokensRetained = optimizedUsage.TotalTokens
		result.TokensRemoved = originalUsage.TotalTokens - optimizedUsage.TotalTokens
		result.CompressionRatio = float64(optimizedUsage.TotalTokens) / float64(originalUsage.TotalTokens)
	}

	return result, nil
}

// optimizeMessageSelection applies intelligent message selection
func (sw *SlidingWindow) optimizeMessageSelection(messages []ConversationMessage, _ string) []ConversationMessage {
	contextConfig := sw.config.GetContextConfig()
	if contextConfig.RelevanceThreshold <= 0 {
		return messages
	}

	// For now, return messages as-is
	// In production, this would apply relevance scoring and filtering
	return messages
}

// GetWindowStats returns statistics about the sliding window
func (sw *SlidingWindow) GetWindowStats(messages []ConversationMessage, modelID string) map[string]interface{} {
	usage := sw.tokenCounter.CountConversationTokens(messages, modelID)
	modelConfig := sw.config.GetModelConfig(modelID)
	contextConfig := sw.config.GetContextConfig()

	return map[string]interface{}{
		"total_messages":  len(messages),
		"total_tokens":    usage.TotalTokens,
		"context_window":  modelConfig.ContextWindow,
		"utilization":     float64(usage.TotalTokens) / float64(modelConfig.ContextWindow),
		"sliding_enabled": contextConfig.SlidingWindow,
		"overlap_tokens":  contextConfig.WindowOverlap,
		"needs_window":    usage.TotalTokens > modelConfig.ContextWindow,
		"summary_count":   sw.countSummaryMessages(messages),
	}
}

// countSummaryMessages counts the number of summary messages
func (sw *SlidingWindow) countSummaryMessages(messages []ConversationMessage) int {
	count := 0
	for _, msg := range messages {
		if sw.isSummaryMessage(msg) {
			count++
		}
	}
	return count
}
