package chat

import (
	"fmt"
	"regexp"
	"strings"
)

// MessageParser parses message content into parts
type MessageParser struct {
	// Regex patterns for different content types
	codeBlockPattern *regexp.Regexp
	fileBlockPattern *regexp.Regexp
    toolPattern      *regexp.Regexp
    toolResultPattern *regexp.Regexp
	diagPattern      *regexp.Regexp
}

// NewMessageParser creates a new message parser
func NewMessageParser() *MessageParser {
	return &MessageParser{
		// Match code blocks with optional language
		codeBlockPattern: regexp.MustCompile("(?s)```(?:([a-zA-Z0-9_+-]+)\n)?(.+?)```"),
		// Match file blocks with path and optional line range
		fileBlockPattern: regexp.MustCompile("(?s)```file:([^\n]+)\n(.+?)```"),
        // Match tool invocations
        toolPattern: regexp.MustCompile("(?s)(?:<tool(?:\\s+[^>]*)?>(.*?)</tool>)|(?:<tool_use>(.*?)</tool_use>)"),
        // Match tool results
        toolResultPattern: regexp.MustCompile("(?s)<tool_result>(.*?)</tool_result>"),
		// Match diagnostics
		diagPattern: regexp.MustCompile("(?s)<diagnostic[^>]*>(.*?)</diagnostic>"),
	}
}

// ParseMessage parses a message into parts
func (p *MessageParser) ParseMessage(msg Message) []MessagePart {
	content := msg.Content
	var parts []MessagePart
	lastIndex := 0
	
	// Find all special blocks and sort by position
	type match struct {
		start   int
		end     int
		partType MessagePartType
		content string
		metadata map[string]interface{}
	}
	
	var matches []match
	
	// Find file blocks
	for _, m := range p.fileBlockPattern.FindAllStringSubmatchIndex(content, -1) {
		if len(m) >= 6 {
			pathInfo := content[m[2]:m[3]]
			fileContent := content[m[4]:m[5]]
			
			matches = append(matches, match{
				start:    m[0],
				end:      m[1],
				partType: PartTypeFile,
				content:  fileContent,
				metadata: map[string]interface{}{"path": pathInfo},
			})
		}
	}
	
    // Find tool invocations
    for _, m := range p.toolPattern.FindAllStringSubmatchIndex(content, -1) {
        if len(m) >= 6 {
            startIdx, endIdx := m[2], m[3]
            if startIdx == -1 || endIdx == -1 {
                startIdx, endIdx = m[4], m[5]
            }
            if startIdx != -1 && endIdx != -1 {
                // Include the full block so renderers can access attributes like name
                toolBlock := content[m[0]:m[1]]
                matches = append(matches, match{
                    start:    m[0],
                    end:      m[1],
                    partType: PartTypeTool,
                    content:  toolBlock,
                    metadata: nil,
                })
            }
        }
    }

    // Find tool results
    for _, m := range p.toolResultPattern.FindAllStringSubmatchIndex(content, -1) {
        if len(m) >= 4 {
            resContent := content[m[2]:m[3]]
            matches = append(matches, match{
                start:    m[0],
                end:      m[1],
                partType: PartTypeToolResult,
                content:  resContent,
                metadata: nil,
            })
        }
    }
	
    // Find diagnostics (include full tag so attributes are available)
    for _, m := range p.diagPattern.FindAllStringSubmatchIndex(content, -1) {
        if len(m) >= 2 {
            fullBlock := content[m[0]:m[1]]
            matches = append(matches, match{
                start:    m[0],
                end:      m[1],
                partType: PartTypeDiagnostic,
                content:  fullBlock,
                metadata: nil,
            })
        }
    }
	
	// Find code blocks that aren't file blocks
	for _, m := range p.codeBlockPattern.FindAllStringSubmatchIndex(content, -1) {
		if len(m) >= 6 {
			// Check if this is a file block
			isFileBlock := false
			for _, fm := range matches {
				if fm.partType == PartTypeFile && fm.start == m[0] && fm.end == m[1] {
					isFileBlock = true
					break
				}
			}
			
            if !isFileBlock {
				lang := ""
				if m[2] != -1 && m[3] != -1 {
					lang = content[m[2]:m[3]]
				}
				codeContent := content[m[4]:m[5]]
                
                partType := PartTypeCode
                if strings.ToLower(lang) == "diff" {
                    partType = PartTypeDiff
                }
                
                matches = append(matches, match{
                    start:    m[0],
                    end:      m[1],
                    partType: partType,
                    content:  codeContent,
                    metadata: map[string]interface{}{"language": lang},
                })
			}
		}
	}
	
	// Sort matches by start position
	for i := 0; i < len(matches); i++ {
		for j := i + 1; j < len(matches); j++ {
			if matches[j].start < matches[i].start {
				matches[i], matches[j] = matches[j], matches[i]
			}
		}
	}
	
	// Build parts list
	partIndex := 0
	for _, m := range matches {
		// Add text before this match
		if m.start > lastIndex {
			textContent := strings.TrimSpace(content[lastIndex:m.start])
			if textContent != "" {
				parts = append(parts, MessagePart{
					Type:      PartTypeText,
					Content:   textContent,
					MessageID: msg.ID,
					PartIndex: partIndex,
					Metadata:  nil,
				})
				partIndex++
			}
		}
		
		// Add the matched part
		parts = append(parts, MessagePart{
			Type:      m.partType,
			Content:   m.content,
			MessageID: msg.ID,
			PartIndex: partIndex,
			Metadata:  m.metadata,
		})
		partIndex++
		
		lastIndex = m.end
	}
	
	// Add any remaining text
	if lastIndex < len(content) {
		textContent := strings.TrimSpace(content[lastIndex:])
		if textContent != "" {
			parts = append(parts, MessagePart{
				Type:      PartTypeText,
				Content:   textContent,
				MessageID: msg.ID,
				PartIndex: partIndex,
				Metadata:  nil,
			})
		}
	}
	
	// If no parts were found, treat the entire content as text
	if len(parts) == 0 && strings.TrimSpace(content) != "" {
		parts = append(parts, MessagePart{
			Type:      PartTypeText,
			Content:   content,
			MessageID: msg.ID,
			PartIndex: 0,
			Metadata:  nil,
		})
	}
	
	return parts
}

// ParseToolInvocation parses tool invocation content
func ParseToolInvocation(content string) (*ToolInvocation, error) {
	// This is a simplified parser - in production, use proper XML/JSON parsing
	tool := &ToolInvocation{}
	
	// Extract tool name
	if idx := strings.Index(content, "<name>"); idx != -1 {
		start := idx + 6
		if end := strings.Index(content[start:], "</name>"); end != -1 {
			tool.Name = content[start : start+end]
		}
	}
	
	// Extract parameters as JSON string
	if idx := strings.Index(content, "<parameters>"); idx != -1 {
		start := idx + 12
		if end := strings.Index(content[start:], "</parameters>"); end != -1 {
			paramStr := content[start : start+end]
			// For now, store as a map with "json" key
			tool.Parameters = map[string]interface{}{"json": paramStr}
		}
	}
	
	if tool.Name == "" {
		return nil, fmt.Errorf("no tool name found")
	}
	
	return tool, nil
}

// CodeBlock represents a code block with language and content
type CodeBlock struct {
	Language string
	Code     string
}

// ExtractCodeBlocks extracts code blocks from content
func (p *MessageParser) ExtractCodeBlocks(content string) []CodeBlock {
	var blocks []CodeBlock
	matches := p.codeBlockPattern.FindAllStringSubmatch(content, -1)
	for _, m := range matches {
		if len(m) > 2 {
			block := CodeBlock{
				Code: m[2],
			}
			if len(m) > 1 && m[1] != "" {
				block.Language = m[1]
			}
			blocks = append(blocks, block)
		}
	}
	return blocks
}