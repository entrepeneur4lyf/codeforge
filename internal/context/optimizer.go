package context

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
)

// ContextOptimizer handles context optimization and redundancy removal
type ContextOptimizer struct {
	config *OptimizationConfig
}

// OptimizationConfig configures context optimization behavior
type OptimizationConfig struct {
	RemoveDuplicates       bool    `json:"remove_duplicates"`
	MinSimilarityThreshold float64 `json:"min_similarity_threshold"`
	RemoveBoilerplate      bool    `json:"remove_boilerplate"`
	CompressWhitespace     bool    `json:"compress_whitespace"`
	RemoveComments         bool    `json:"remove_comments"`
	DeduplicateCode        bool    `json:"deduplicate_code"`
	MaxLineLength          int     `json:"max_line_length"`
	PreserveStructure      bool    `json:"preserve_structure"`
	AggressiveMode         bool    `json:"aggressive_mode"`
}

// DefaultOptimizationConfig returns default optimization settings
func DefaultOptimizationConfig() *OptimizationConfig {
	return &OptimizationConfig{
		RemoveDuplicates:       true,
		MinSimilarityThreshold: 0.8,
		RemoveBoilerplate:      true,
		CompressWhitespace:     true,
		RemoveComments:         false, // Preserve comments by default
		DeduplicateCode:        true,
		MaxLineLength:          120,
		PreserveStructure:      true,
		AggressiveMode:         false,
	}
}

// NewContextOptimizer creates a new context optimizer
func NewContextOptimizer(config *OptimizationConfig) *ContextOptimizer {
	if config == nil {
		config = DefaultOptimizationConfig()
	}

	return &ContextOptimizer{
		config: config,
	}
}

// OptimizationResult represents the result of context optimization
type OptimizationResult struct {
	OriginalMessages     []ConversationMessage  `json:"original_messages"`
	OptimizedMessages    []ConversationMessage  `json:"optimized_messages"`
	OriginalTokens       int                    `json:"original_tokens"`
	OptimizedTokens      int                    `json:"optimized_tokens"`
	CompressionRatio     float64                `json:"compression_ratio"`
	OptimizationsApplied []string               `json:"optimizations_applied"`
	ProcessingTime       time.Duration          `json:"processing_time"`
	Statistics           map[string]interface{} `json:"statistics"`
}

// OptimizeContext optimizes a conversation context
func (co *ContextOptimizer) OptimizeContext(messages []ConversationMessage, modelID string) (*OptimizationResult, error) {
	startTime := time.Now()

	log.Printf("Starting context optimization for %d messages", len(messages))

	// Create a copy to avoid modifying the original
	optimized := make([]ConversationMessage, len(messages))
	copy(optimized, messages)

	var optimizationsApplied []string
	statistics := make(map[string]interface{})

	// Count original tokens
	tokenCounter := NewTokenCounter()
	originalTokens := tokenCounter.CountConversationTokens(messages, modelID).TotalTokens

	// Apply optimizations in order
	if co.config.RemoveDuplicates {
		before := len(optimized)
		optimized = co.removeDuplicateMessages(optimized)
		after := len(optimized)
		if before != after {
			optimizationsApplied = append(optimizationsApplied, "remove_duplicates")
			statistics["duplicates_removed"] = before - after
		}
	}

	if co.config.RemoveBoilerplate {
		before := co.calculateTotalLength(optimized)
		optimized = co.removeBoilerplate(optimized)
		after := co.calculateTotalLength(optimized)
		if before != after {
			optimizationsApplied = append(optimizationsApplied, "remove_boilerplate")
			statistics["boilerplate_chars_removed"] = before - after
		}
	}

	if co.config.CompressWhitespace {
		before := co.calculateTotalLength(optimized)
		optimized = co.compressWhitespace(optimized)
		after := co.calculateTotalLength(optimized)
		if before != after {
			optimizationsApplied = append(optimizationsApplied, "compress_whitespace")
			statistics["whitespace_chars_removed"] = before - after
		}
	}

	if co.config.RemoveComments {
		before := co.calculateTotalLength(optimized)
		optimized = co.removeComments(optimized)
		after := co.calculateTotalLength(optimized)
		if before != after {
			optimizationsApplied = append(optimizationsApplied, "remove_comments")
			statistics["comment_chars_removed"] = before - after
		}
	}

	if co.config.DeduplicateCode {
		before := co.calculateTotalLength(optimized)
		optimized = co.deduplicateCodeBlocks(optimized)
		after := co.calculateTotalLength(optimized)
		if before != after {
			optimizationsApplied = append(optimizationsApplied, "deduplicate_code")
			statistics["code_chars_removed"] = before - after
		}
	}

	// Apply content-level optimizations
	optimized = co.optimizeMessageContent(optimized)

	// Count optimized tokens
	optimizedTokens := tokenCounter.CountConversationTokens(optimized, modelID).TotalTokens

	// Calculate compression ratio
	compressionRatio := 1.0
	if originalTokens > 0 {
		compressionRatio = float64(optimizedTokens) / float64(originalTokens)
	}

	processingTime := time.Since(startTime)

	log.Printf("Context optimization completed: %d -> %d tokens (%.2f%% reduction) in %v",
		originalTokens, optimizedTokens, (1-compressionRatio)*100, processingTime)

	return &OptimizationResult{
		OriginalMessages:     messages,
		OptimizedMessages:    optimized,
		OriginalTokens:       originalTokens,
		OptimizedTokens:      optimizedTokens,
		CompressionRatio:     compressionRatio,
		OptimizationsApplied: optimizationsApplied,
		ProcessingTime:       processingTime,
		Statistics:           statistics,
	}, nil
}

// removeDuplicateMessages removes duplicate or highly similar messages
func (co *ContextOptimizer) removeDuplicateMessages(messages []ConversationMessage) []ConversationMessage {
	var result []ConversationMessage
	seen := make(map[string]bool)

	for _, msg := range messages {
		// Create a normalized key for comparison
		key := co.normalizeForComparison(msg.Content)

		if !seen[key] {
			seen[key] = true
			result = append(result, msg)
		} else {
			log.Printf("Removed duplicate message: %.50s...", msg.Content)
		}
	}

	return result
}

// removeBoilerplate removes common boilerplate text
func (co *ContextOptimizer) removeBoilerplate(messages []ConversationMessage) []ConversationMessage {
	boilerplatePatterns := []string{
		`(?i)^(sure|okay|alright|yes|no problem|of course)[,!.]?\s*`,
		`(?i)\s*(let me|i'll|i will|i can)\s+`,
		`(?i)\s*(here's|here is|this is)\s+`,
		`(?i)\s*(as you can see|as shown|as mentioned)\s*`,
		`(?i)\s*(please note|note that|keep in mind)\s*`,
	}

	var result []ConversationMessage
	for _, msg := range messages {
		content := msg.Content

		// Apply boilerplate removal patterns
		for _, pattern := range boilerplatePatterns {
			re := regexp.MustCompile(pattern)
			content = re.ReplaceAllString(content, "")
		}

		// Only include if there's meaningful content left
		if strings.TrimSpace(content) != "" {
			msg.Content = strings.TrimSpace(content)
			result = append(result, msg)
		}
	}

	return result
}

// compressWhitespace compresses excessive whitespace
func (co *ContextOptimizer) compressWhitespace(messages []ConversationMessage) []ConversationMessage {
	var result []ConversationMessage

	for _, msg := range messages {
		content := msg.Content

		// Replace multiple spaces with single space
		content = regexp.MustCompile(`\s+`).ReplaceAllString(content, " ")

		// Replace multiple newlines with double newline
		content = regexp.MustCompile(`\n\s*\n\s*\n+`).ReplaceAllString(content, "\n\n")

		// Trim leading/trailing whitespace
		content = strings.TrimSpace(content)

		if content != "" {
			msg.Content = content
			result = append(result, msg)
		}
	}

	return result
}

// removeComments removes code comments (when enabled)
func (co *ContextOptimizer) removeComments(messages []ConversationMessage) []ConversationMessage {
	var result []ConversationMessage

	commentPatterns := []string{
		`//.*$`,           // Single-line comments
		`/\*[\s\S]*?\*/`,  // Multi-line comments
		`#.*$`,            // Python/shell comments
		`<!--[\s\S]*?-->`, // HTML comments
	}

	for _, msg := range messages {
		content := msg.Content

		// Only remove comments from code blocks
		if strings.Contains(content, "```") {
			for _, pattern := range commentPatterns {
				re := regexp.MustCompile(pattern)
				content = re.ReplaceAllString(content, "")
			}
		}

		msg.Content = content
		result = append(result, msg)
	}

	return result
}

// deduplicateCodeBlocks removes duplicate code blocks
func (co *ContextOptimizer) deduplicateCodeBlocks(messages []ConversationMessage) []ConversationMessage {
	var result []ConversationMessage
	seenCodeBlocks := make(map[string]bool)

	for _, msg := range messages {
		content := msg.Content

		// Extract code blocks
		codeBlockPattern := regexp.MustCompile("```[\\s\\S]*?```")
		codeBlocks := codeBlockPattern.FindAllString(content, -1)

		// Check for duplicates and replace if found
		for _, block := range codeBlocks {
			normalized := co.normalizeCodeBlock(block)
			if seenCodeBlocks[normalized] {
				// Replace with reference
				content = strings.Replace(content, block, "[Code block omitted - duplicate]", 1)
			} else {
				seenCodeBlocks[normalized] = true
			}
		}

		msg.Content = content
		result = append(result, msg)
	}

	return result
}

// optimizeMessageContent applies content-level optimizations
func (co *ContextOptimizer) optimizeMessageContent(messages []ConversationMessage) []ConversationMessage {
	var result []ConversationMessage

	for _, msg := range messages {
		content := msg.Content

		// Truncate very long lines if configured
		if co.config.MaxLineLength > 0 {
			content = co.truncateLongLines(content, co.config.MaxLineLength)
		}

		// Remove excessive repetition
		content = co.removeRepetition(content)

		// Compress verbose explanations in aggressive mode
		if co.config.AggressiveMode {
			content = co.compressVerboseContent(content)
		}

		msg.Content = content
		result = append(result, msg)
	}

	return result
}

// normalizeForComparison normalizes text for duplicate detection
func (co *ContextOptimizer) normalizeForComparison(text string) string {
	// Convert to lowercase
	normalized := strings.ToLower(text)

	// Remove extra whitespace
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")

	// Remove punctuation for comparison
	normalized = regexp.MustCompile(`[^\w\s]`).ReplaceAllString(normalized, "")

	return strings.TrimSpace(normalized)
}

// normalizeCodeBlock normalizes code blocks for duplicate detection
func (co *ContextOptimizer) normalizeCodeBlock(codeBlock string) string {
	// Remove language specifier
	normalized := regexp.MustCompile("```\\w*\\n").ReplaceAllString(codeBlock, "```\n")

	// Remove trailing ```
	normalized = strings.TrimSuffix(normalized, "```")
	normalized = strings.TrimPrefix(normalized, "```")

	// Normalize whitespace
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")

	return strings.TrimSpace(normalized)
}

// truncateLongLines truncates lines that exceed the maximum length
func (co *ContextOptimizer) truncateLongLines(content string, maxLength int) string {
	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		if len(line) > maxLength {
			truncated := line[:maxLength-3] + "..."
			result = append(result, truncated)
		} else {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// removeRepetition removes repetitive content using advanced pattern detection
func (co *ContextOptimizer) removeRepetition(content string) string {
	words := strings.Fields(content)
	if len(words) < 4 {
		return content
	}

	// Multi-pass repetition removal with different pattern sizes
	result := co.removeRepeatedPhrases(words, 5) // 5-word phrases
	result = co.removeRepeatedPhrases(result, 4) // 4-word phrases
	result = co.removeRepeatedPhrases(result, 3) // 3-word phrases
	result = co.removeRepeatedPhrases(result, 2) // 2-word phrases

	// Remove excessive word repetition
	result = co.removeExcessiveWordRepetition(result)

	// Remove redundant sentences
	sentences := co.splitIntoSentences(strings.Join(result, " "))
	uniqueSentences := co.removeDuplicateSentences(sentences)

	return strings.Join(uniqueSentences, " ")
}

// removeRepeatedPhrases removes repeated phrases of specified length
func (co *ContextOptimizer) removeRepeatedPhrases(words []string, phraseLength int) []string {
	if len(words) < phraseLength*2 {
		return words
	}

	var result []string
	seenPhrases := make(map[string]int)

	for i := 0; i < len(words); i++ {
		// Check if we can form a phrase of the desired length
		if i+phraseLength <= len(words) {
			phrase := strings.Join(words[i:i+phraseLength], " ")

			// Track phrase occurrences
			seenPhrases[phrase]++

			// If this phrase appears too frequently, skip subsequent occurrences
			if seenPhrases[phrase] > 2 {
				// Skip this phrase but continue with next word
				continue
			}

			// Check for immediate repetition
			if i+phraseLength*2 <= len(words) {
				nextPhrase := strings.Join(words[i+phraseLength:i+phraseLength*2], " ")
				if phrase == nextPhrase {
					// Skip the repeated phrase
					i += phraseLength - 1
					continue
				}
			}
		}

		result = append(result, words[i])
	}

	return result
}

// removeExcessiveWordRepetition removes words that repeat too frequently in close proximity
func (co *ContextOptimizer) removeExcessiveWordRepetition(words []string) []string {
	if len(words) < 10 {
		return words
	}

	var result []string
	windowSize := 10 // Look at 10-word windows

	for i, word := range words {
		// Count occurrences of this word in the current window
		count := 0
		start := i - windowSize/2
		end := i + windowSize/2

		if start < 0 {
			start = 0
		}
		if end >= len(words) {
			end = len(words) - 1
		}

		for j := start; j <= end; j++ {
			if j != i && strings.ToLower(words[j]) == strings.ToLower(word) {
				count++
			}
		}

		// Skip if word appears too frequently in window (more than 3 times)
		if count <= 3 {
			result = append(result, word)
		}
	}

	return result
}

// splitIntoSentences splits text into sentences for deduplication
func (co *ContextOptimizer) splitIntoSentences(text string) []string {
	// Split on sentence boundaries
	sentences := regexp.MustCompile(`[.!?]+\s+`).Split(text, -1)

	var result []string
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) > 10 { // Only keep meaningful sentences
			result = append(result, sentence)
		}
	}

	return result
}

// removeDuplicateSentences removes duplicate or very similar sentences
func (co *ContextOptimizer) removeDuplicateSentences(sentences []string) []string {
	var result []string
	seen := make(map[string]bool)

	for _, sentence := range sentences {
		// Normalize sentence for comparison
		normalized := strings.ToLower(strings.TrimSpace(sentence))
		normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")

		// Check for exact duplicates
		if seen[normalized] {
			continue
		}

		// Check for high similarity with existing sentences
		tooSimilar := false
		for existing := range seen {
			if co.calculateSentenceSimilarity(normalized, existing) > 0.85 {
				tooSimilar = true
				break
			}
		}

		if !tooSimilar {
			seen[normalized] = true
			result = append(result, sentence)
		}
	}

	return result
}

// calculateSentenceSimilarity calculates similarity between two sentences
func (co *ContextOptimizer) calculateSentenceSimilarity(s1, s2 string) float64 {
	words1 := strings.Fields(s1)
	words2 := strings.Fields(s2)

	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}

	// Count common words
	wordSet1 := make(map[string]bool)
	for _, word := range words1 {
		wordSet1[word] = true
	}

	commonWords := 0
	for _, word := range words2 {
		if wordSet1[word] {
			commonWords++
		}
	}

	// Calculate Jaccard similarity
	totalUniqueWords := len(wordSet1)
	for _, word := range words2 {
		if !wordSet1[word] {
			totalUniqueWords++
		}
	}

	if totalUniqueWords == 0 {
		return 0.0
	}

	return float64(commonWords) / float64(totalUniqueWords)
}

// compressVerboseContent compresses verbose explanations
func (co *ContextOptimizer) compressVerboseContent(content string) string {
	// Replace verbose phrases with concise alternatives
	replacements := map[string]string{
		"in order to":                  "to",
		"due to the fact that":         "because",
		"at this point in time":        "now",
		"it is important to note that": "note:",
		"please be aware that":         "note:",
		"it should be mentioned that":  "",
	}

	result := content
	for verbose, concise := range replacements {
		pattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(verbose) + `\b`)
		result = pattern.ReplaceAllString(result, concise)
	}

	return result
}

// calculateTotalLength calculates total character length of all messages
func (co *ContextOptimizer) calculateTotalLength(messages []ConversationMessage) int {
	total := 0
	for _, msg := range messages {
		total += len(msg.Content)
	}
	return total
}

// GetOptimizationStats returns optimization statistics
func (co *ContextOptimizer) GetOptimizationStats(result *OptimizationResult) map[string]interface{} {
	stats := make(map[string]interface{})

	stats["original_message_count"] = len(result.OriginalMessages)
	stats["optimized_message_count"] = len(result.OptimizedMessages)
	stats["messages_removed"] = len(result.OriginalMessages) - len(result.OptimizedMessages)
	stats["token_reduction"] = result.OriginalTokens - result.OptimizedTokens
	stats["compression_percentage"] = (1 - result.CompressionRatio) * 100
	stats["processing_time_ms"] = result.ProcessingTime.Milliseconds()
	stats["optimizations_applied"] = result.OptimizationsApplied

	// Merge in specific statistics
	for k, v := range result.Statistics {
		stats[k] = v
	}

	return stats
}

// OptimizeForModel optimizes context specifically for a model's characteristics
func (co *ContextOptimizer) OptimizeForModel(messages []ConversationMessage, modelID string) (*OptimizationResult, error) {
	// Adjust optimization strategy based on model
	originalConfig := co.config

	// Create model-specific config
	modelConfig := *originalConfig

	// Adjust for different model types
	if strings.Contains(strings.ToLower(modelID), "gpt-4") {
		// GPT-4 can handle more context, be less aggressive
		modelConfig.AggressiveMode = false
		modelConfig.RemoveComments = false
	} else if strings.Contains(strings.ToLower(modelID), "gpt-3.5") {
		// GPT-3.5 has smaller context, be more aggressive
		modelConfig.AggressiveMode = true
		modelConfig.RemoveComments = true
	}

	// Temporarily use model-specific config
	co.config = &modelConfig
	defer func() { co.config = originalConfig }()

	return co.OptimizeContext(messages, modelID)
}

// BatchOptimize optimizes multiple conversations
func (co *ContextOptimizer) BatchOptimize(conversations [][]ConversationMessage, modelID string) ([]*OptimizationResult, error) {
	var results []*OptimizationResult

	for i, messages := range conversations {
		log.Printf("Optimizing conversation %d/%d", i+1, len(conversations))

		result, err := co.OptimizeContext(messages, modelID)
		if err != nil {
			return nil, fmt.Errorf("failed to optimize conversation %d: %w", i, err)
		}

		results = append(results, result)
	}

	return results, nil
}

// UpdateConfig updates the optimization configuration
func (co *ContextOptimizer) UpdateConfig(config *OptimizationConfig) {
	co.config = config
}

// GetConfig returns the current optimization configuration
func (co *ContextOptimizer) GetConfig() *OptimizationConfig {
	return co.config
}
