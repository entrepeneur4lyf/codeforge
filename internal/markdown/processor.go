package markdown

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// MessageFormat represents different output formats for chat messages
type MessageFormat string

const (
	FormatPlain    MessageFormat = "plain"
	FormatMarkdown MessageFormat = "markdown"
	FormatHTML     MessageFormat = "html"
	FormatTerminal MessageFormat = "terminal"
)

// ProcessedMessage represents a chat message with multiple format options
type ProcessedMessage struct {
	ID        string                 `json:"id"`
	SessionID string                 `json:"session_id"`
	Role      string                 `json:"role"`
	Content   map[string]string      `json:"content"` // Format -> rendered content
	Timestamp int64                  `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// MessageProcessor handles markdown processing for chat messages
type MessageProcessor struct {
	chatRenderer *Renderer
	webRenderer  *Renderer
}

// NewMessageProcessor creates a new message processor with optimized renderers
func NewMessageProcessor() (*MessageProcessor, error) {
	chatRenderer, err := NewChatRenderer()
	if err != nil {
		return nil, fmt.Errorf("failed to create chat renderer: %w", err)
	}

	webRenderer, err := NewWebRenderer()
	if err != nil {
		return nil, fmt.Errorf("failed to create web renderer: %w", err)
	}

	return &MessageProcessor{
		chatRenderer: chatRenderer,
		webRenderer:  webRenderer,
	}, nil
}

// ProcessMessage processes a raw message content into multiple formats
func (mp *MessageProcessor) ProcessMessage(content string) (map[string]string, error) {
	formats := make(map[string]string)

	// Always include the original plain text
	formats[string(FormatPlain)] = content

	// Check if content contains markdown
	if mp.containsMarkdown(content) {
		// Store original markdown
		formats[string(FormatMarkdown)] = content

		// Render for terminal/chat display
		terminalRendered, err := mp.chatRenderer.Render(content)
		if err != nil {
			// If rendering fails, fall back to plain text
			formats[string(FormatTerminal)] = content
		} else {
			formats[string(FormatTerminal)] = terminalRendered
		}

		// Render for web/HTML display
		htmlRendered, err := mp.webRenderer.RenderToHTML(content)
		if err != nil {
			// If HTML rendering fails, create basic HTML
			formats[string(FormatHTML)] = mp.createBasicHTML(content)
		} else {
			formats[string(FormatHTML)] = htmlRendered
		}
	} else {
		// No markdown detected, use plain text for all formats
		formats[string(FormatMarkdown)] = content
		formats[string(FormatTerminal)] = content
		formats[string(FormatHTML)] = mp.createBasicHTML(content)
	}

	return formats, nil
}

// ProcessChatMessage creates a ProcessedMessage from basic message data
func (mp *MessageProcessor) ProcessChatMessage(id, sessionID, role, content string, timestamp int64, metadata map[string]interface{}) (*ProcessedMessage, error) {
	processedContent, err := mp.ProcessMessage(content)
	if err != nil {
		return nil, fmt.Errorf("failed to process message content: %w", err)
	}

	// Add format metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["has_markdown"] = mp.containsMarkdown(content)
	metadata["available_formats"] = []string{
		string(FormatPlain),
		string(FormatMarkdown),
		string(FormatTerminal),
		string(FormatHTML),
	}

	return &ProcessedMessage{
		ID:        id,
		SessionID: sessionID,
		Role:      role,
		Content:   processedContent,
		Timestamp: timestamp,
		Metadata:  metadata,
	}, nil
}

// GetContentForFormat returns content in the specified format
func (pm *ProcessedMessage) GetContentForFormat(format MessageFormat) string {
	if content, exists := pm.Content[string(format)]; exists {
		return content
	}
	// Fallback to plain text if format not available
	return pm.Content[string(FormatPlain)]
}

// ToJSON converts the processed message to JSON
func (pm *ProcessedMessage) ToJSON() ([]byte, error) {
	return json.Marshal(pm)
}

// containsMarkdown checks if content contains markdown syntax
func (mp *MessageProcessor) containsMarkdown(content string) bool {
	// Common markdown patterns
	patterns := []string{
		`^#{1,6}\s+`,           // Headers
		`\*\*.*\*\*`,           // Bold
		`\*.*\*`,               // Italic
		`__.*__`,               // Bold (underscore)
		`_.*_`,                 // Italic (underscore)
		"`.*`",                 // Inline code
		"```",                  // Code blocks
		`^\s*[-*+]\s+`,         // Unordered lists
		`^\s*\d+\.\s+`,         // Ordered lists
		`^\s*>\s+`,             // Blockquotes
		`\[.*\]\(.*\)`,         // Links
		`!\[.*\]\(.*\)`,        // Images
		`^\s*\|.*\|\s*$`,       // Tables
		`^\s*[-=]{3,}\s*$`,     // Horizontal rules
	}

	for _, pattern := range patterns {
		matched, err := regexp.MatchString(pattern, content)
		if err == nil && matched {
			return true
		}
	}

	// Check for multi-line patterns
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		// Check for setext headers (underlined with = or -)
		if i > 0 && len(strings.TrimSpace(line)) > 0 {
			if matched, _ := regexp.MatchString(`^[=-]+$`, strings.TrimSpace(line)); matched {
				return true
			}
		}
	}

	return false
}

// createBasicHTML creates basic HTML from plain text
func (mp *MessageProcessor) createBasicHTML(content string) string {
	// Escape HTML entities
	content = strings.ReplaceAll(content, "&", "&amp;")
	content = strings.ReplaceAll(content, "<", "&lt;")
	content = strings.ReplaceAll(content, ">", "&gt;")
	content = strings.ReplaceAll(content, "\"", "&quot;")
	content = strings.ReplaceAll(content, "'", "&#39;")

	// Convert newlines to <br> tags
	content = strings.ReplaceAll(content, "\n", "<br>")

	// Wrap in a div
	return fmt.Sprintf(`<div class="plain-text-content">%s</div>`, content)
}

// ExtractCodeBlocks extracts code blocks from markdown content
func (mp *MessageProcessor) ExtractCodeBlocks(content string) []CodeBlock {
	var blocks []CodeBlock
	
	// Regex to match code blocks with optional language
	codeBlockRegex := regexp.MustCompile("```(\\w*)\\n([\\s\\S]*?)```")
	matches := codeBlockRegex.FindAllStringSubmatch(content, -1)
	
	for i, match := range matches {
		language := match[1]
		code := match[2]
		
		if language == "" {
			language = "text"
		}
		
		blocks = append(blocks, CodeBlock{
			ID:       fmt.Sprintf("block_%d", i),
			Language: language,
			Code:     strings.TrimSpace(code),
		})
	}
	
	return blocks
}

// CodeBlock represents an extracted code block
type CodeBlock struct {
	ID       string `json:"id"`
	Language string `json:"language"`
	Code     string `json:"code"`
}

// FormatSelector helps clients choose the appropriate format
type FormatSelector struct {
	UserAgent   string
	AcceptTypes []string
	ClientType  string
}

// SelectBestFormat selects the most appropriate format based on client capabilities
func (fs *FormatSelector) SelectBestFormat(availableFormats []string) MessageFormat {
	// Default to plain text
	bestFormat := FormatPlain

	// Check client type preferences
	switch strings.ToLower(fs.ClientType) {
	case "web", "browser":
		if contains(availableFormats, string(FormatHTML)) {
			return FormatHTML
		}
	case "terminal", "cli":
		if contains(availableFormats, string(FormatTerminal)) {
			return FormatTerminal
		}
	case "api", "json":
		if contains(availableFormats, string(FormatMarkdown)) {
			return FormatMarkdown
		}
	}

	// Check accept types
	for _, acceptType := range fs.AcceptTypes {
		switch acceptType {
		case "text/html":
			if contains(availableFormats, string(FormatHTML)) {
				return FormatHTML
			}
		case "text/markdown":
			if contains(availableFormats, string(FormatMarkdown)) {
				return FormatMarkdown
			}
		case "text/plain":
			return FormatPlain
		}
	}

	return bestFormat
}

// Helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
