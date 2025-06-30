package context

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

// MetadataManager handles context metadata and tagging
type MetadataManager struct {
	tags     map[string][]string // message ID -> tags
	metadata map[string]map[string]interface{} // message ID -> metadata
}

// NewMetadataManager creates a new metadata manager
func NewMetadataManager() *MetadataManager {
	return &MetadataManager{
		tags:     make(map[string][]string),
		metadata: make(map[string]map[string]interface{}),
	}
}

// ContextMetadata represents metadata for a conversation message
type ContextMetadata struct {
	MessageID    string                 `json:"message_id"`
	Tags         []string               `json:"tags"`
	Category     string                 `json:"category"`
	Priority     int                    `json:"priority"`     // 1-10 scale
	Complexity   int                    `json:"complexity"`   // 1-10 scale
	CodeBlocks   []CodeBlock            `json:"code_blocks"`
	Entities     []Entity               `json:"entities"`
	Topics       []string               `json:"topics"`
	Language     string                 `json:"language"`     // Programming language if applicable
	Intent       string                 `json:"intent"`       // question, answer, explanation, etc.
	Quality      float64                `json:"quality"`      // 0.0-1.0 quality score
	Relationships []Relationship        `json:"relationships"`
	Custom       map[string]interface{} `json:"custom"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// CodeBlock represents a code block within a message
type CodeBlock struct {
	Language string `json:"language"`
	Content  string `json:"content"`
	StartPos int    `json:"start_pos"`
	EndPos   int    `json:"end_pos"`
	Type     string `json:"type"` // function, class, snippet, etc.
}

// Entity represents a named entity in the message
type Entity struct {
	Name     string  `json:"name"`
	Type     string  `json:"type"`     // function, class, variable, file, etc.
	Context  string  `json:"context"`  // surrounding context
	Confidence float64 `json:"confidence"` // 0.0-1.0
}

// Relationship represents a relationship between messages
type Relationship struct {
	Type      string  `json:"type"`      // dependency, reference, continuation, etc.
	TargetID  string  `json:"target_id"` // related message ID
	Strength  float64 `json:"strength"`  // 0.0-1.0
	Direction string  `json:"direction"` // incoming, outgoing, bidirectional
}

// TaggedMessage represents a message with its metadata
type TaggedMessage struct {
	Message  ConversationMessage `json:"message"`
	Metadata ContextMetadata     `json:"metadata"`
}

// AnalyzeAndTagMessages analyzes messages and generates metadata
func (mm *MetadataManager) AnalyzeAndTagMessages(messages []ConversationMessage) []TaggedMessage {
	var taggedMessages []TaggedMessage

	for i, msg := range messages {
		msgID := fmt.Sprintf("msg_%d", i)
		metadata := mm.analyzeMessage(msg, msgID, i, len(messages))
		
		taggedMessages = append(taggedMessages, TaggedMessage{
			Message:  msg,
			Metadata: metadata,
		})
		
		// Store in internal maps
		mm.tags[msgID] = metadata.Tags
		mm.metadata[msgID] = map[string]interface{}{
			"category":   metadata.Category,
			"priority":   metadata.Priority,
			"complexity": metadata.Complexity,
			"intent":     metadata.Intent,
			"quality":    metadata.Quality,
			"language":   metadata.Language,
			"topics":     metadata.Topics,
		}
	}

	return taggedMessages
}

// analyzeMessage analyzes a single message and generates metadata
func (mm *MetadataManager) analyzeMessage(msg ConversationMessage, msgID string, index, totalMessages int) ContextMetadata {
	now := time.Now()
	
	metadata := ContextMetadata{
		MessageID: msgID,
		CreatedAt: now,
		UpdatedAt: now,
		Custom:    make(map[string]interface{}),
	}

	// Extract code blocks
	metadata.CodeBlocks = mm.extractCodeBlocks(msg.Content)
	
	// Extract entities
	metadata.Entities = mm.extractEntities(msg.Content)
	
	// Determine category
	metadata.Category = mm.categorizeMessage(msg)
	
	// Generate tags
	metadata.Tags = mm.generateTags(msg, metadata.CodeBlocks, metadata.Entities)
	
	// Extract topics
	metadata.Topics = mm.extractTopics(msg.Content)
	
	// Determine programming language
	metadata.Language = mm.detectProgrammingLanguage(msg.Content, metadata.CodeBlocks)
	
	// Determine intent
	metadata.Intent = mm.determineIntent(msg)
	
	// Calculate priority
	metadata.Priority = mm.calculatePriority(msg, index, totalMessages)
	
	// Calculate complexity
	metadata.Complexity = mm.calculateComplexity(msg.Content, metadata.CodeBlocks)
	
	// Calculate quality score
	metadata.Quality = mm.calculateQuality(msg)

	return metadata
}

// extractCodeBlocks extracts code blocks from message content
func (mm *MetadataManager) extractCodeBlocks(content string) []CodeBlock {
	var blocks []CodeBlock
	
	// Extract fenced code blocks (```)
	fencedPattern := regexp.MustCompile("```(\\w+)?\\n([\\s\\S]*?)```")
	matches := fencedPattern.FindAllStringSubmatch(content, -1)
	
	for _, match := range matches {
		if len(match) >= 3 {
			language := match[1]
			if language == "" {
				language = "unknown"
			}
			
			block := CodeBlock{
				Language: language,
				Content:  strings.TrimSpace(match[2]),
				Type:     mm.determineCodeBlockType(match[2]),
			}
			blocks = append(blocks, block)
		}
	}
	
	// Extract inline code (`)
	inlinePattern := regexp.MustCompile("`([^`]+)`")
	inlineMatches := inlinePattern.FindAllStringSubmatch(content, -1)
	
	for _, match := range inlineMatches {
		if len(match) >= 2 && len(match[1]) > 3 { // Only longer inline code
			block := CodeBlock{
				Language: "inline",
				Content:  match[1],
				Type:     "snippet",
			}
			blocks = append(blocks, block)
		}
	}
	
	return blocks
}

// extractEntities extracts named entities from message content
func (mm *MetadataManager) extractEntities(content string) []Entity {
	var entities []Entity
	
	// Extract function names
	funcPattern := regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)\s*\(`)
	funcMatches := funcPattern.FindAllStringSubmatch(content, -1)
	for _, match := range funcMatches {
		if len(match) > 1 {
			entities = append(entities, Entity{
				Name:       match[1],
				Type:       "function",
				Context:    mm.extractContext(content, match[0]),
				Confidence: 0.8,
			})
		}
	}
	
	// Extract class names (capitalized words)
	classPattern := regexp.MustCompile(`\b([A-Z][a-zA-Z0-9_]*)\b`)
	classMatches := classPattern.FindAllStringSubmatch(content, -1)
	for _, match := range classMatches {
		if len(match) > 1 && len(match[1]) > 2 {
			entities = append(entities, Entity{
				Name:       match[1],
				Type:       "class",
				Context:    mm.extractContext(content, match[0]),
				Confidence: 0.6,
			})
		}
	}
	
	// Extract file names
	filePattern := regexp.MustCompile(`\b([a-zA-Z0-9_-]+\.[a-zA-Z0-9]+)\b`)
	fileMatches := filePattern.FindAllStringSubmatch(content, -1)
	for _, match := range fileMatches {
		if len(match) > 1 {
			entities = append(entities, Entity{
				Name:       match[1],
				Type:       "file",
				Context:    mm.extractContext(content, match[0]),
				Confidence: 0.9,
			})
		}
	}
	
	return mm.deduplicateEntities(entities)
}

// categorizeMessage categorizes the message
func (mm *MetadataManager) categorizeMessage(msg ConversationMessage) string {
	content := strings.ToLower(msg.Content)
	
	// Code-related categories
	if strings.Contains(content, "```") || strings.Contains(content, "function") || strings.Contains(content, "class") {
		return "code"
	}
	
	// Question category
	if strings.Contains(msg.Content, "?") || strings.Contains(content, "how") || strings.Contains(content, "what") {
		return "question"
	}
	
	// Error/debugging category
	if strings.Contains(content, "error") || strings.Contains(content, "bug") || strings.Contains(content, "debug") {
		return "debugging"
	}
	
	// Explanation category
	if msg.Role == "assistant" && len(msg.Content) > 200 {
		return "explanation"
	}
	
	// Configuration category
	if strings.Contains(content, "config") || strings.Contains(content, "setting") || strings.Contains(content, "setup") {
		return "configuration"
	}
	
	return "general"
}

// generateTags generates tags for the message
func (mm *MetadataManager) generateTags(msg ConversationMessage, codeBlocks []CodeBlock, entities []Entity) []string {
	var tags []string
	
	// Role-based tags
	tags = append(tags, msg.Role)
	
	// Content-based tags
	content := strings.ToLower(msg.Content)
	
	if len(codeBlocks) > 0 {
		tags = append(tags, "code")
		
		// Language-specific tags
		for _, block := range codeBlocks {
			if block.Language != "unknown" && block.Language != "inline" {
				tags = append(tags, "lang:"+block.Language)
			}
		}
	}
	
	// Entity-based tags
	for _, entity := range entities {
		if entity.Confidence > 0.7 {
			tags = append(tags, entity.Type+":"+entity.Name)
		}
	}
	
	// Content pattern tags
	if strings.Contains(content, "error") || strings.Contains(content, "exception") {
		tags = append(tags, "error")
	}
	
	if strings.Contains(content, "test") {
		tags = append(tags, "testing")
	}
	
	if strings.Contains(content, "performance") || strings.Contains(content, "optimize") {
		tags = append(tags, "performance")
	}
	
	if strings.Contains(content, "security") {
		tags = append(tags, "security")
	}
	
	// Length-based tags
	if len(msg.Content) > 1000 {
		tags = append(tags, "long")
	} else if len(msg.Content) < 100 {
		tags = append(tags, "short")
	}
	
	return mm.deduplicateTags(tags)
}

// extractTopics extracts topics from message content
func (mm *MetadataManager) extractTopics(content string) []string {
	var topics []string
	contentLower := strings.ToLower(content)
	
	topicKeywords := map[string]string{
		"database":    "Database",
		"api":         "API",
		"frontend":    "Frontend",
		"backend":     "Backend",
		"testing":     "Testing",
		"deployment":  "Deployment",
		"security":    "Security",
		"performance": "Performance",
		"algorithm":   "Algorithms",
		"data structure": "Data Structures",
		"machine learning": "Machine Learning",
		"ai":          "Artificial Intelligence",
		"docker":      "Docker",
		"kubernetes":  "Kubernetes",
		"git":         "Version Control",
		"ci/cd":       "CI/CD",
	}
	
	for keyword, topic := range topicKeywords {
		if strings.Contains(contentLower, keyword) {
			topics = append(topics, topic)
		}
	}
	
	return topics
}

// detectProgrammingLanguage detects the primary programming language
func (mm *MetadataManager) detectProgrammingLanguage(content string, codeBlocks []CodeBlock) string {
	// Check explicit language declarations in code blocks
	languageCounts := make(map[string]int)
	
	for _, block := range codeBlocks {
		if block.Language != "unknown" && block.Language != "inline" {
			languageCounts[block.Language]++
		}
	}
	
	// Find most common language
	maxCount := 0
	primaryLanguage := ""
	for lang, count := range languageCounts {
		if count > maxCount {
			maxCount = count
			primaryLanguage = lang
		}
	}
	
	if primaryLanguage != "" {
		return primaryLanguage
	}
	
	// Fallback: detect from content patterns
	contentLower := strings.ToLower(content)
	
	languagePatterns := map[string][]string{
		"go":         {"func ", "package ", "import ", "go.mod"},
		"python":     {"def ", "import ", "from ", "__init__", "pip install"},
		"javascript": {"function", "const ", "let ", "var ", "npm install"},
		"typescript": {"interface", "type ", "export ", "import "},
		"java":       {"public class", "private ", "public ", "import java"},
		"rust":       {"fn ", "let mut", "cargo ", "use "},
		"cpp":        {"#include", "std::", "namespace", "class "},
		"c":          {"#include", "int main", "printf", "malloc"},
	}
	
	for language, patterns := range languagePatterns {
		for _, pattern := range patterns {
			if strings.Contains(contentLower, pattern) {
				return language
			}
		}
	}
	
	return "unknown"
}

// determineIntent determines the intent of the message
func (mm *MetadataManager) determineIntent(msg ConversationMessage) string {
	content := strings.ToLower(msg.Content)
	
	if msg.Role == "user" {
		if strings.Contains(msg.Content, "?") {
			return "question"
		}
		if strings.Contains(content, "please") || strings.Contains(content, "can you") {
			return "request"
		}
		if strings.Contains(content, "error") || strings.Contains(content, "problem") {
			return "problem_report"
		}
		return "statement"
	} else if msg.Role == "assistant" {
		if strings.Contains(content, "here's") || strings.Contains(content, "you can") {
			return "solution"
		}
		if len(msg.Content) > 300 {
			return "explanation"
		}
		return "response"
	}
	
	return "unknown"
}

// calculatePriority calculates message priority (1-10)
func (mm *MetadataManager) calculatePriority(msg ConversationMessage, index, totalMessages int) int {
	priority := 5 // Base priority
	
	// Recent messages get higher priority
	recencyBonus := int(float64(index) / float64(totalMessages) * 3)
	priority += recencyBonus
	
	// Questions get higher priority
	if strings.Contains(msg.Content, "?") {
		priority += 2
	}
	
	// Error messages get higher priority
	if strings.Contains(strings.ToLower(msg.Content), "error") {
		priority += 2
	}
	
	// Code messages get moderate priority boost
	if strings.Contains(msg.Content, "```") {
		priority += 1
	}
	
	// Clamp to 1-10 range
	if priority > 10 {
		priority = 10
	}
	if priority < 1 {
		priority = 1
	}
	
	return priority
}

// calculateComplexity calculates message complexity (1-10)
func (mm *MetadataManager) calculateComplexity(content string, codeBlocks []CodeBlock) int {
	complexity := 1
	
	// Length-based complexity
	if len(content) > 500 {
		complexity += 2
	}
	if len(content) > 1000 {
		complexity += 2
	}
	
	// Code complexity
	complexity += len(codeBlocks)
	
	// Technical terms increase complexity
	technicalTerms := []string{
		"algorithm", "complexity", "optimization", "architecture",
		"design pattern", "concurrency", "async", "thread",
	}
	
	contentLower := strings.ToLower(content)
	for _, term := range technicalTerms {
		if strings.Contains(contentLower, term) {
			complexity++
		}
	}
	
	// Clamp to 1-10 range
	if complexity > 10 {
		complexity = 10
	}
	
	return complexity
}

// calculateQuality calculates message quality score (0.0-1.0)
func (mm *MetadataManager) calculateQuality(msg ConversationMessage) float64 {
	quality := 0.5 // Base quality
	
	// Length considerations
	length := len(msg.Content)
	if length > 50 && length < 2000 {
		quality += 0.2 // Good length
	}
	
	// Grammar and structure (simple heuristics)
	if strings.Contains(msg.Content, ".") || strings.Contains(msg.Content, "!") {
		quality += 0.1 // Has sentence endings
	}
	
	// Code formatting
	if strings.Contains(msg.Content, "```") {
		quality += 0.2 // Well-formatted code
	}
	
	// Avoid very short or very long messages
	if length < 10 {
		quality -= 0.3
	}
	if length > 3000 {
		quality -= 0.2
	}
	
	// Clamp to 0.0-1.0 range
	if quality > 1.0 {
		quality = 1.0
	}
	if quality < 0.0 {
		quality = 0.0
	}
	
	return quality
}

// Helper functions

func (mm *MetadataManager) extractContext(content, match string) string {
	index := strings.Index(content, match)
	if index == -1 {
		return ""
	}
	
	start := index - 20
	if start < 0 {
		start = 0
	}
	
	end := index + len(match) + 20
	if end > len(content) {
		end = len(content)
	}
	
	return strings.TrimSpace(content[start:end])
}

func (mm *MetadataManager) determineCodeBlockType(code string) string {
	codeLower := strings.ToLower(code)
	
	if strings.Contains(codeLower, "function") || strings.Contains(codeLower, "def ") {
		return "function"
	}
	if strings.Contains(codeLower, "class ") {
		return "class"
	}
	if strings.Contains(codeLower, "test") {
		return "test"
	}
	if strings.Contains(codeLower, "config") {
		return "configuration"
	}
	
	return "snippet"
}

func (mm *MetadataManager) deduplicateEntities(entities []Entity) []Entity {
	seen := make(map[string]bool)
	var result []Entity
	
	for _, entity := range entities {
		key := entity.Type + ":" + entity.Name
		if !seen[key] {
			seen[key] = true
			result = append(result, entity)
		}
	}
	
	return result
}

func (mm *MetadataManager) deduplicateTags(tags []string) []string {
	seen := make(map[string]bool)
	var result []string
	
	for _, tag := range tags {
		if !seen[tag] {
			seen[tag] = true
			result = append(result, tag)
		}
	}
	
	return result
}

// Query and filtering methods

// FindMessagesByTag finds messages with specific tags
func (mm *MetadataManager) FindMessagesByTag(tag string) []string {
	var messageIDs []string
	
	for msgID, tags := range mm.tags {
		for _, t := range tags {
			if t == tag {
				messageIDs = append(messageIDs, msgID)
				break
			}
		}
	}
	
	return messageIDs
}

// FindMessagesByCategory finds messages in a specific category
func (mm *MetadataManager) FindMessagesByCategory(category string) []string {
	var messageIDs []string
	
	for msgID, metadata := range mm.metadata {
		if cat, ok := metadata["category"].(string); ok && cat == category {
			messageIDs = append(messageIDs, msgID)
		}
	}
	
	return messageIDs
}

// GetMessageTags returns tags for a specific message
func (mm *MetadataManager) GetMessageTags(messageID string) []string {
	if tags, exists := mm.tags[messageID]; exists {
		return tags
	}
	return []string{}
}

// GetMessageMetadata returns metadata for a specific message
func (mm *MetadataManager) GetMessageMetadata(messageID string) map[string]interface{} {
	if metadata, exists := mm.metadata[messageID]; exists {
		return metadata
	}
	return make(map[string]interface{})
}

// AddTag adds a tag to a message
func (mm *MetadataManager) AddTag(messageID, tag string) {
	if tags, exists := mm.tags[messageID]; exists {
		// Check if tag already exists
		for _, t := range tags {
			if t == tag {
				return
			}
		}
		mm.tags[messageID] = append(tags, tag)
	} else {
		mm.tags[messageID] = []string{tag}
	}
}

// RemoveTag removes a tag from a message
func (mm *MetadataManager) RemoveTag(messageID, tag string) {
	if tags, exists := mm.tags[messageID]; exists {
		var newTags []string
		for _, t := range tags {
			if t != tag {
				newTags = append(newTags, t)
			}
		}
		mm.tags[messageID] = newTags
	}
}

// GetAllTags returns all unique tags
func (mm *MetadataManager) GetAllTags() []string {
	tagSet := make(map[string]bool)
	
	for _, tags := range mm.tags {
		for _, tag := range tags {
			tagSet[tag] = true
		}
	}
	
	var allTags []string
	for tag := range tagSet {
		allTags = append(allTags, tag)
	}
	
	sort.Strings(allTags)
	return allTags
}

// ExportMetadata exports all metadata as JSON
func (mm *MetadataManager) ExportMetadata() (string, error) {
	export := map[string]interface{}{
		"tags":     mm.tags,
		"metadata": mm.metadata,
	}
	
	data, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return "", err
	}
	
	return string(data), nil
}

// ImportMetadata imports metadata from JSON
func (mm *MetadataManager) ImportMetadata(jsonData string) error {
	var imported map[string]interface{}
	
	if err := json.Unmarshal([]byte(jsonData), &imported); err != nil {
		return err
	}
	
	if tags, ok := imported["tags"].(map[string]interface{}); ok {
		mm.tags = make(map[string][]string)
		for msgID, tagList := range tags {
			if tags, ok := tagList.([]interface{}); ok {
				var stringTags []string
				for _, tag := range tags {
					if tagStr, ok := tag.(string); ok {
						stringTags = append(stringTags, tagStr)
					}
				}
				mm.tags[msgID] = stringTags
			}
		}
	}
	
	if metadata, ok := imported["metadata"].(map[string]interface{}); ok {
		mm.metadata = make(map[string]map[string]interface{})
		for msgID, meta := range metadata {
			if metaMap, ok := meta.(map[string]interface{}); ok {
				mm.metadata[msgID] = metaMap
			}
		}
	}
	
	return nil
}
