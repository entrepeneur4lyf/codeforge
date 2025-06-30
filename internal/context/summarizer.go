package context

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/config"
)

// Summarizer handles automatic conversation summarization
type Summarizer struct {
	tokenCounter *TokenCounter
	config       *config.Config
}

// NewSummarizer creates a new conversation summarizer
func NewSummarizer(cfg *config.Config) *Summarizer {
	return &Summarizer{
		tokenCounter: NewTokenCounter(),
		config:       cfg,
	}
}

// SummaryResult represents the result of summarization
type SummaryResult struct {
	Summary          string                 `json:"summary"`
	OriginalTokens   int                    `json:"original_tokens"`
	SummaryTokens    int                    `json:"summary_tokens"`
	CompressionRatio float64                `json:"compression_ratio"`
	MessagesKept     int                    `json:"messages_kept"`
	MessagesRemoved  int                    `json:"messages_removed"`
	Timestamp        int64                  `json:"timestamp"`
	Method           string                 `json:"method"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// SummarizeConversation summarizes a conversation when it approaches token limits
func (s *Summarizer) SummarizeConversation(ctx context.Context, messages []ConversationMessage, modelID string) (*SummaryResult, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("no messages to summarize")
	}

	// Count current tokens
	originalUsage := s.tokenCounter.CountConversationTokens(messages, modelID)

	// Check if summarization is needed
	if !s.config.ShouldSummarize(modelID, originalUsage.TotalTokens) {
		return nil, fmt.Errorf("summarization not needed, current tokens: %d", originalUsage.TotalTokens)
	}

	log.Printf("Starting conversation summarization for model %s, current tokens: %d", modelID, originalUsage.TotalTokens)

	// Find the last summary message to avoid re-summarizing
	lastSummaryIndex := s.findLastSummaryIndex(messages)

	// Get messages to summarize (everything after last summary)
	messagesToSummarize := messages
	if lastSummaryIndex >= 0 {
		messagesToSummarize = messages[lastSummaryIndex+1:]
	}

	if len(messagesToSummarize) < 2 {
		return nil, fmt.Errorf("not enough messages to summarize")
	}

	// Create summary
	summary, err := s.createSummary(ctx, messagesToSummarize, modelID)
	if err != nil {
		return nil, fmt.Errorf("failed to create summary: %w", err)
	}

	// Count summary tokens
	summaryTokens := s.tokenCounter.CountTokens(summary, modelID)

	// Calculate compression ratio
	originalTokensToSummarize := s.tokenCounter.CountConversationTokens(messagesToSummarize, modelID).TotalTokens
	compressionRatio := float64(summaryTokens) / float64(originalTokensToSummarize)

	result := &SummaryResult{
		Summary:          summary,
		OriginalTokens:   originalTokensToSummarize,
		SummaryTokens:    summaryTokens,
		CompressionRatio: compressionRatio,
		MessagesKept:     len(messages) - len(messagesToSummarize) + 1, // +1 for the summary message
		MessagesRemoved:  len(messagesToSummarize),
		Timestamp:        time.Now().Unix(),
		Method:           "auto_summarize",
		Metadata: map[string]interface{}{
			"model_id":           modelID,
			"last_summary_index": lastSummaryIndex,
			"trigger_threshold":  s.config.GetModelConfig(modelID).SummarizeThreshold,
		},
	}

	log.Printf("Summarization complete: %d tokens -> %d tokens (%.2f%% compression)",
		originalTokensToSummarize, summaryTokens, compressionRatio*100)

	return result, nil
}

// createSummary generates a summary of the conversation messages
func (s *Summarizer) createSummary(_ context.Context, messages []ConversationMessage, _ string) (string, error) {
	// For now, implement a simple extractive summarization
	// In production, this would call an LLM to generate a proper summary

	var summary strings.Builder
	summary.WriteString("## Conversation Summary\n\n")

	// Extract key information from messages
	userQuestions := []string{}
	assistantResponses := []string{}
	codeBlocks := []string{}

	for _, msg := range messages {
		switch msg.Role {
		case "user":
			// Extract questions and requests
			if len(msg.Content) > 0 {
				userQuestions = append(userQuestions, s.extractKeyPoints(msg.Content, 100))
			}
		case "assistant":
			// Extract key responses and solutions
			if len(msg.Content) > 0 {
				assistantResponses = append(assistantResponses, s.extractKeyPoints(msg.Content, 150))
				// Extract code blocks
				codeBlocks = append(codeBlocks, s.extractCodeBlocks(msg.Content)...)
			}
		}
	}

	// Build summary
	if len(userQuestions) > 0 {
		summary.WriteString("### User Requests:\n")
		for i, question := range userQuestions {
			if i >= 5 { // Limit to 5 most recent
				break
			}
			summary.WriteString(fmt.Sprintf("- %s\n", question))
		}
		summary.WriteString("\n")
	}

	if len(assistantResponses) > 0 {
		summary.WriteString("### Key Responses:\n")
		for i, response := range assistantResponses {
			if i >= 5 { // Limit to 5 most recent
				break
			}
			summary.WriteString(fmt.Sprintf("- %s\n", response))
		}
		summary.WriteString("\n")
	}

	if len(codeBlocks) > 0 {
		summary.WriteString("### Code Examples:\n")
		for i, code := range codeBlocks {
			if i >= 3 { // Limit to 3 most recent
				break
			}
			summary.WriteString(fmt.Sprintf("```\n%s\n```\n\n", s.truncateText(code, 200)))
		}
	}

	return summary.String(), nil
}

// findLastSummaryIndex finds the index of the last summary message
func (s *Summarizer) findLastSummaryIndex(messages []ConversationMessage) int {
	for i := len(messages) - 1; i >= 0; i-- {
		if metadata, ok := messages[i].Metadata["summary"]; ok {
			if isSummary, ok := metadata.(bool); ok && isSummary {
				return i
			}
		}
	}
	return -1
}

// extractKeyPoints extracts key points from text, limiting to maxLength
func (s *Summarizer) extractKeyPoints(text string, maxLength int) string {
	// Simple extraction - take first sentence or up to maxLength
	text = strings.TrimSpace(text)

	// Find first sentence
	sentences := strings.Split(text, ". ")
	if len(sentences) > 0 && len(sentences[0]) <= maxLength {
		return sentences[0]
	}

	// Truncate to maxLength
	return s.truncateText(text, maxLength)
}

// extractCodeBlocks extracts code blocks from text
func (s *Summarizer) extractCodeBlocks(text string) []string {
	var blocks []string

	// Find code blocks marked with ```
	lines := strings.Split(text, "\n")
	var currentBlock strings.Builder
	inBlock := false

	for _, line := range lines {
		if strings.HasPrefix(line, "```") {
			if inBlock {
				// End of block
				if currentBlock.Len() > 0 {
					blocks = append(blocks, currentBlock.String())
					currentBlock.Reset()
				}
				inBlock = false
			} else {
				// Start of block
				inBlock = true
			}
		} else if inBlock {
			currentBlock.WriteString(line + "\n")
		}
	}

	return blocks
}

// truncateText truncates text to maxLength with ellipsis
func (s *Summarizer) truncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}

	if maxLength <= 3 {
		return text[:maxLength]
	}

	return text[:maxLength-3] + "..."
}

// CreateSummaryMessage creates a conversation message containing the summary
func (s *Summarizer) CreateSummaryMessage(summary string, sessionID string) ConversationMessage {
	return ConversationMessage{
		Role:    "assistant",
		Content: summary,
		Metadata: map[string]interface{}{
			"summary":    true,
			"session_id": sessionID,
			"type":       "auto_summary",
		},
		Timestamp: time.Now().Unix(),
	}
}

// ShouldSummarize checks if conversation should be summarized
func (s *Summarizer) ShouldSummarize(messages []ConversationMessage, modelID string) bool {
	if !s.config.GetContextConfig().AutoSummarize {
		return false
	}

	usage := s.tokenCounter.CountConversationTokens(messages, modelID)
	return s.config.ShouldSummarize(modelID, usage.TotalTokens)
}
