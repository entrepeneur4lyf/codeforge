package chat

import (
	"regexp"
	"strings"
)

// MessageParser parses messages into selectable parts
type MessageParser struct {
	// Patterns for different content types
	codeBlockPattern   *regexp.Regexp
	toolPattern        *regexp.Regexp
	toolResultPattern  *regexp.Regexp
	fileRefPattern     *regexp.Regexp
	diffPattern        *regexp.Regexp
}

// NewMessageParser creates a new message parser
func NewMessageParser() *MessageParser {
	return &MessageParser{
		// Match code blocks with optional language
		codeBlockPattern: regexp.MustCompile("(?s)```([a-zA-Z0-9_+-]*)\n(.*?)```"),
		
		// Match tool invocations (common patterns)
		toolPattern: regexp.MustCompile("(?s)<tool(?:\\s+[^>]*)?>.*?</tool>"),
		
		// Match tool results
		toolResultPattern: regexp.MustCompile("(?s)<tool_result>.*?</tool_result>"),
		
		// Match file references (e.g., path/to/file.go:123)
		fileRefPattern: regexp.MustCompile(`(?m)^[\w/\-_.]+\.\w+(?::\d+)?$`),
		
		// Match diff blocks
		diffPattern: regexp.MustCompile("(?s)```diff\n(.*?)```"),
	}
}

// ParseMessage parses a message into selectable parts
func (p *MessageParser) ParseMessage(msg Message) []MessagePart {
	if msg.Content == "" {
		return nil
	}
	
	// Track what content has been processed
	content := msg.Content
	var parts []MessagePart
	partIndex := 0
	
	// Keep track of positions for accurate line numbers
	processedUntil := 0
	
	// First, find all special blocks and their positions
	type block struct {
		start    int
		end      int
		partType MessagePartType
		content  string
		metadata map[string]interface{}
	}
	
	var blocks []block
	
	// Find code blocks
	for _, match := range p.codeBlockPattern.FindAllStringSubmatchIndex(content, -1) {
		language := content[match[2]:match[3]]
		code := content[match[4]:match[5]]
		
		blocks = append(blocks, block{
			start:    match[0],
			end:      match[1],
			partType: PartTypeCode,
			content:  code,
			metadata: map[string]interface{}{
				"language": language,
			},
		})
	}
	
	// Find tool invocations
	for _, match := range p.toolPattern.FindAllStringIndex(content, -1) {
		toolContent := content[match[0]:match[1]]
		blocks = append(blocks, block{
			start:    match[0],
			end:      match[1],
			partType: PartTypeTool,
			content:  toolContent,
		})
	}
	
	// Find tool results
	for _, match := range p.toolResultPattern.FindAllStringIndex(content, -1) {
		resultContent := content[match[0]:match[1]]
		blocks = append(blocks, block{
			start:    match[0],
			end:      match[1],
			partType: PartTypeToolResult,
			content:  resultContent,
		})
	}
	
	// Find diffs (if not already captured as code blocks)
	for _, match := range p.diffPattern.FindAllStringSubmatchIndex(content, -1) {
		// Check if this overlaps with a code block
		overlaps := false
		for _, b := range blocks {
			if b.partType == PartTypeCode && match[0] >= b.start && match[1] <= b.end {
				overlaps = true
				break
			}
		}
		
		if !overlaps {
			diffContent := content[match[2]:match[3]]
			blocks = append(blocks, block{
				start:    match[0],
				end:      match[1],
				partType: PartTypeDiff,
				content:  diffContent,
			})
		}
	}
	
	// Sort blocks by start position
	sortBlocks(blocks)
	
	// Now create parts, including text between blocks
	currentPos := 0
	currentLine := 0
	
	for _, b := range blocks {
		// Add text before this block (if any)
		if b.start > currentPos {
			textContent := content[currentPos:b.start]
			textContent = strings.TrimSpace(textContent)
			
			if textContent != "" {
				startLine := currentLine
				endLine := currentLine + strings.Count(textContent, "\n")
				
				parts = append(parts, MessagePart{
					MessageID: msg.ID,
					PartIndex: partIndex,
					Type:      PartTypeText,
					Content:   textContent,
					StartLine: startLine,
					EndLine:   endLine,
				})
				partIndex++
				currentLine = endLine + 1
			}
		}
		
		// Add the special block
		startLine := currentLine
		endLine := currentLine + strings.Count(b.content, "\n")
		
		part := MessagePart{
			MessageID: msg.ID,
			PartIndex: partIndex,
			Type:      b.partType,
			Content:   b.content,
			StartLine: startLine,
			EndLine:   endLine,
			Metadata:  b.metadata,
		}
		
		parts = append(parts, part)
		partIndex++
		
		currentPos = b.end
		currentLine = endLine + 1 + strings.Count(content[b.start:b.end], "\n") - strings.Count(b.content, "\n")
	}
	
	// Add any remaining text
	if currentPos < len(content) {
		textContent := content[currentPos:]
		textContent = strings.TrimSpace(textContent)
		
		if textContent != "" {
			startLine := currentLine
			endLine := currentLine + strings.Count(textContent, "\n")
			
			parts = append(parts, MessagePart{
				MessageID: msg.ID,
				PartIndex: partIndex,
				Type:      PartTypeText,
				Content:   textContent,
				StartLine: startLine,
				EndLine:   endLine,
			})
		}
	}
	
	// If no parts were created, treat entire message as one text part
	if len(parts) == 0 && msg.Content != "" {
		parts = append(parts, MessagePart{
			MessageID: msg.ID,
			PartIndex: 0,
			Type:      PartTypeText,
			Content:   msg.Content,
			StartLine: 0,
			EndLine:   strings.Count(msg.Content, "\n"),
		})
	}
	
	return parts
}

// sortBlocks sorts blocks by start position
func sortBlocks(blocks []block) {
	// Simple bubble sort for small arrays
	for i := 0; i < len(blocks); i++ {
		for j := i + 1; j < len(blocks); j++ {
			if blocks[j].start < blocks[i].start {
				blocks[i], blocks[j] = blocks[j], blocks[i]
			}
		}
	}
}

// ExtractCodeBlocks extracts all code blocks from content
func (p *MessageParser) ExtractCodeBlocks(content string) []struct {
	Language string
	Code     string
} {
	var blocks []struct {
		Language string
		Code     string
	}
	
	matches := p.codeBlockPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		blocks = append(blocks, struct {
			Language string
			Code     string
		}{
			Language: match[1],
			Code:     match[2],
		})
	}
	
	return blocks
}