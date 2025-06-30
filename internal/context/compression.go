package context

import (
	"regexp"
	"strings"
)

// ContextCompressor handles context compression
type ContextCompressor struct {
	level int
}

// NewContextCompressor creates a new context compressor
func NewContextCompressor(level int) *ContextCompressor {
	return &ContextCompressor{level: level}
}

// CompressionResult represents the result of context compression
type CompressionResult struct {
	Messages         []ConversationMessage `json:"messages"`
	OriginalTokens   int                   `json:"original_tokens"`
	CompressedTokens int                   `json:"compressed_tokens"`
	CompressionRatio float64               `json:"compression_ratio"`
	Method           string                `json:"method"`
	BytesSaved       int                   `json:"bytes_saved"`
}

// CompressMessages compresses conversation messages
func (cc *ContextCompressor) CompressMessages(messages []ConversationMessage, modelID string) (*CompressionResult, error) {
	if cc.level <= 0 {
		return &CompressionResult{
			Messages:         messages,
			OriginalTokens:   0,
			CompressedTokens: 0,
			CompressionRatio: 1.0,
			Method:           "none",
		}, nil
	}

	compressedMessages := make([]ConversationMessage, len(messages))
	originalSize := 0
	compressedSize := 0

	for i, msg := range messages {
		originalContent := msg.Content
		compressedContent := cc.compressText(originalContent, cc.level)
		
		originalSize += len(originalContent)
		compressedSize += len(compressedContent)
		
		compressedMsg := msg
		compressedMsg.Content = compressedContent
		compressedMessages[i] = compressedMsg
	}

	compressionRatio := float64(compressedSize) / float64(originalSize)
	if originalSize == 0 {
		compressionRatio = 1.0
	}

	return &CompressionResult{
		Messages:         compressedMessages,
		OriginalTokens:   originalSize,
		CompressedTokens: compressedSize,
		CompressionRatio: compressionRatio,
		Method:           cc.getCompressionMethod(),
		BytesSaved:       originalSize - compressedSize,
	}, nil
}

// compressText applies compression to text based on level
func (cc *ContextCompressor) compressText(text string, level int) string {
	if level <= 0 {
		return text
	}

	compressed := text

	// Level 1: Basic whitespace compression
	if level >= 1 {
		compressed = cc.compressWhitespace(compressed)
	}

	// Level 2: Remove redundant punctuation
	if level >= 2 {
		compressed = cc.compressRedundantPunctuation(compressed)
	}

	// Level 3: Compress common phrases
	if level >= 3 {
		compressed = cc.compressCommonPhrases(compressed)
	}

	// Level 4: Remove filler words
	if level >= 4 {
		compressed = cc.removeFillerWords(compressed)
	}

	// Level 5: Aggressive compression
	if level >= 5 {
		compressed = cc.aggressiveCompression(compressed)
	}

	return compressed
}

// compressWhitespace removes extra whitespace
func (cc *ContextCompressor) compressWhitespace(text string) string {
	// Replace multiple spaces with single space
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	
	// Remove leading/trailing whitespace from lines
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	
	// Remove empty lines
	var nonEmptyLines []string
	for _, line := range lines {
		if line != "" {
			nonEmptyLines = append(nonEmptyLines, line)
		}
	}
	
	return strings.Join(nonEmptyLines, "\n")
}

// compressRedundantPunctuation removes redundant punctuation
func (cc *ContextCompressor) compressRedundantPunctuation(text string) string {
	// Replace multiple punctuation marks with single ones
	text = regexp.MustCompile(`\.{2,}`).ReplaceAllString(text, ".")
	text = regexp.MustCompile(`!{2,}`).ReplaceAllString(text, "!")
	text = regexp.MustCompile(`\?{2,}`).ReplaceAllString(text, "?")
	text = regexp.MustCompile(`,{2,}`).ReplaceAllString(text, ",")
	text = regexp.MustCompile(`;{2,}`).ReplaceAllString(text, ";")
	
	return text
}

// compressCommonPhrases replaces common phrases with shorter versions
func (cc *ContextCompressor) compressCommonPhrases(text string) string {
	replacements := map[string]string{
		"you can":        "you can",
		"I would like":   "I'd like",
		"I will":         "I'll",
		"you will":       "you'll",
		"it is":          "it's",
		"that is":        "that's",
		"there is":       "there's",
		"there are":      "there're",
		"I have":         "I've",
		"you have":       "you've",
		"we have":        "we've",
		"they have":      "they've",
		"I am":           "I'm",
		"you are":        "you're",
		"we are":         "we're",
		"they are":       "they're",
		"cannot":         "can't",
		"do not":         "don't",
		"does not":       "doesn't",
		"did not":        "didn't",
		"will not":       "won't",
		"would not":      "wouldn't",
		"could not":      "couldn't",
		"should not":     "shouldn't",
		"must not":       "mustn't",
	}

	for long, short := range replacements {
		// Case insensitive replacement
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(long) + `\b`)
		text = re.ReplaceAllString(text, short)
	}

	return text
}

// removeFillerWords removes common filler words
func (cc *ContextCompressor) removeFillerWords(text string) string {
	fillerWords := []string{
		"actually", "basically", "essentially", "literally", "obviously",
		"definitely", "certainly", "absolutely", "totally", "completely",
		"really", "very", "quite", "rather", "pretty", "fairly",
		"just", "only", "simply", "merely", "exactly", "precisely",
		"um", "uh", "er", "ah", "well", "so", "like", "you know",
	}

	for _, filler := range fillerWords {
		// Remove filler words (case insensitive, word boundaries)
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(filler) + `\b\s*`)
		text = re.ReplaceAllString(text, "")
	}

	// Clean up extra spaces created by removal
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	return text
}

// aggressiveCompression applies aggressive compression techniques
func (cc *ContextCompressor) aggressiveCompression(text string) string {
	// Remove articles in non-critical contexts
	text = regexp.MustCompile(`\b(a|an|the)\s+`).ReplaceAllString(text, "")
	
	// Remove some prepositions
	text = regexp.MustCompile(`\b(of|in|on|at|by|for|with|from)\s+`).ReplaceAllString(text, "")
	
	// Compress common programming terms
	programmingReplacements := map[string]string{
		"function":    "func",
		"variable":    "var",
		"parameter":   "param",
		"argument":    "arg",
		"return":      "ret",
		"initialize":  "init",
		"configuration": "config",
		"implementation": "impl",
		"interface":   "iface",
		"structure":   "struct",
		"object":      "obj",
		"method":      "meth",
		"property":    "prop",
		"attribute":   "attr",
	}

	for long, short := range programmingReplacements {
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(long) + `\b`)
		text = re.ReplaceAllString(text, short)
	}

	// Clean up extra spaces
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	return text
}

// getCompressionMethod returns the compression method name based on level
func (cc *ContextCompressor) getCompressionMethod() string {
	switch cc.level {
	case 0:
		return "none"
	case 1:
		return "whitespace"
	case 2:
		return "punctuation"
	case 3:
		return "phrases"
	case 4:
		return "filler_words"
	case 5:
		return "aggressive"
	default:
		return "custom"
	}
}

// DecompressText attempts to reverse compression (limited functionality)
func (cc *ContextCompressor) DecompressText(text string) string {
	// Basic decompression - expand common contractions
	expansions := map[string]string{
		"I'm":       "I am",
		"you're":    "you are",
		"we're":     "we are",
		"they're":   "they are",
		"it's":      "it is",
		"that's":    "that is",
		"there's":   "there is",
		"I'll":      "I will",
		"you'll":    "you will",
		"can't":     "cannot",
		"don't":     "do not",
		"doesn't":   "does not",
		"didn't":    "did not",
		"won't":     "will not",
		"wouldn't":  "would not",
		"couldn't":  "could not",
		"shouldn't": "should not",
		"mustn't":   "must not",
	}

	for short, long := range expansions {
		re := regexp.MustCompile(`\b` + regexp.QuoteMeta(short) + `\b`)
		text = re.ReplaceAllString(text, long)
	}

	return text
}

// GetCompressionStats returns statistics about compression effectiveness
func (cc *ContextCompressor) GetCompressionStats(original, compressed string) map[string]interface{} {
	originalSize := len(original)
	compressedSize := len(compressed)
	
	compressionRatio := float64(compressedSize) / float64(originalSize)
	if originalSize == 0 {
		compressionRatio = 1.0
	}
	
	return map[string]interface{}{
		"original_size":     originalSize,
		"compressed_size":   compressedSize,
		"bytes_saved":       originalSize - compressedSize,
		"compression_ratio": compressionRatio,
		"compression_percent": (1.0 - compressionRatio) * 100,
		"level":            cc.level,
		"method":           cc.getCompressionMethod(),
	}
}
