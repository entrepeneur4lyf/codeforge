package chat

import (
	"encoding/json"
	"fmt"
    "regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/theme"
)

// ToolRenderer handles rendering of tool invocations and results
type ToolRenderer struct {
	theme           theme.Theme
	expandedTools   map[string]bool // Track which tools are expanded
	cache           *MessageCache
	codeRenderer    *BlockRenderer
	resultRenderer  *BlockRenderer
	headerRenderer  *BlockRenderer
	compactRenderer *BlockRenderer
}

// NewToolRenderer creates a new tool renderer
func NewToolRenderer(theme theme.Theme) *ToolRenderer {
	return &ToolRenderer{
		theme:         theme,
		expandedTools: make(map[string]bool),
		cache:         NewMessageCache(50),
		codeRenderer: NewBlockRenderer(theme,
			WithBackgroundColor(theme.BackgroundSecondary()),
			WithTextColor(theme.Text()),
			WithPadding(1),
			WithNoBorder(),
		),
		resultRenderer: NewBlockRenderer(theme,
			WithBackgroundColor(theme.BackgroundSecondary()),
			WithBorder(lipgloss.NormalBorder(), theme.Success()),
			WithPadding(1),
		),
		headerRenderer: NewBlockRenderer(theme,
			WithTextColor(theme.Info()),
			WithBold(),
			WithNoBorder(),
		),
		compactRenderer: NewBlockRenderer(theme,
			WithTextColor(theme.TextMuted()),
			WithItalic(),
			WithNoBorder(),
		),
	}
}

// ToolInvocation represents a tool call
type ToolInvocation struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Parameters map[string]interface{} `json:"parameters"`
	Timestamp  string                 `json:"timestamp,omitempty"`
}

// ToolResult represents the result of a tool invocation
type ToolResult struct {
	ID        string      `json:"id"`
	ToolID    string      `json:"tool_id"`
	Output    interface{} `json:"output"`
	Error     string      `json:"error,omitempty"`
	Duration  string      `json:"duration,omitempty"`
	Timestamp string      `json:"timestamp,omitempty"`
}

// RenderToolInvocation renders a tool invocation with expandable details
func (r *ToolRenderer) RenderToolInvocation(tool ToolInvocation, expanded bool) string {
	cacheKey := r.cache.GenerateKey(tool.ID, tool.Name, fmt.Sprintf("%v", tool.Parameters), expanded)
	
	if cached, found := r.cache.Get(cacheKey); found {
		return cached
	}
	
	var result strings.Builder
	
	// Header with tool name and expand/collapse indicator
	expandIcon := "▶" // Collapsed
	if expanded {
		expandIcon = "▼" // Expanded
	}
	
	header := r.headerRenderer.Render(fmt.Sprintf("%s Tool: %s", expandIcon, tool.Name))
	result.WriteString(header)
	
	if expanded {
		// Show full parameters
		result.WriteString("\n")
		
		// Format parameters
		if len(tool.Parameters) > 0 {
			paramStr := r.formatParameters(tool.Parameters)
			paramBlock := r.codeRenderer.Render(paramStr)
			result.WriteString(paramBlock)
		}
		
		// Show metadata if available
		if tool.Timestamp != "" {
			meta := r.compactRenderer.Render(fmt.Sprintf("Called at: %s", tool.Timestamp))
			result.WriteString("\n" + meta)
		}
	} else {
		// Show compact view
		paramCount := len(tool.Parameters)
		compactInfo := fmt.Sprintf(" (%d parameter", paramCount)
		if paramCount != 1 {
			compactInfo += "s"
		}
		compactInfo += ")"
		
		compact := r.compactRenderer.Render(compactInfo)
		result.WriteString(compact)
	}
	
	rendered := result.String()
	r.cache.Set(cacheKey, rendered)
	return rendered
}

// RenderToolResult renders a tool result with expandable details
func (r *ToolRenderer) RenderToolResult(result ToolResult, expanded bool) string {
	cacheKey := r.cache.GenerateKey(result.ID, result.ToolID, fmt.Sprintf("%v", result.Output), expanded)
	
	if cached, found := r.cache.Get(cacheKey); found {
		return cached
	}
	
	var output strings.Builder
	
	// Header
	expandIcon := "▶"
	if expanded {
		expandIcon = "▼"
	}
	
	status := "Success"
	headerColor := r.theme.Success()
	if result.Error != "" {
		status = "Error"
		headerColor = r.theme.Error()
	}
	
	header := NewBlockRenderer(r.theme,
		WithTextColor(headerColor),
		WithBold(),
		WithNoBorder(),
	).Render(fmt.Sprintf("%s Tool Result: %s", expandIcon, status))
	
	output.WriteString(header)
	
	if expanded {
		output.WriteString("\n")
		
		// Show result or error
		if result.Error != "" {
			errorBlock := ErrorMessageRenderer(r.theme).Render(result.Error)
			output.WriteString(errorBlock)
		} else {
			resultStr := r.formatOutput(result.Output)
			resultBlock := r.resultRenderer.Render(resultStr)
			output.WriteString(resultBlock)
		}
		
		// Show metadata
		var meta []string
		if result.Duration != "" {
			meta = append(meta, fmt.Sprintf("Duration: %s", result.Duration))
		}
		if result.Timestamp != "" {
			meta = append(meta, fmt.Sprintf("Completed: %s", result.Timestamp))
		}
		
		if len(meta) > 0 {
			metaStr := r.compactRenderer.Render(strings.Join(meta, " | "))
			output.WriteString("\n" + metaStr)
		}
	} else {
		// Compact view
		if result.Error != "" {
			compact := r.compactRenderer.Render(" (error occurred)")
			output.WriteString(compact)
		} else {
			// Show brief output preview
			preview := r.getOutputPreview(result.Output)
			compact := r.compactRenderer.Render(fmt.Sprintf(" → %s", preview))
			output.WriteString(compact)
		}
	}
	
	rendered := output.String()
	r.cache.Set(cacheKey, rendered)
	return rendered
}

// ToggleExpanded toggles the expanded state of a tool
func (r *ToolRenderer) ToggleExpanded(toolID string) {
	r.expandedTools[toolID] = !r.expandedTools[toolID]
	// Clear cache when toggling to force re-render
	r.cache.InvalidateMatching(func(key string) bool {
		return strings.Contains(key, toolID)
	})
}

// IsExpanded checks if a tool is expanded
func (r *ToolRenderer) IsExpanded(toolID string) bool {
	return r.expandedTools[toolID]
}

// SetExpanded sets the expanded state of a tool
func (r *ToolRenderer) SetExpanded(toolID string, expanded bool) {
	if r.expandedTools[toolID] != expanded {
		r.expandedTools[toolID] = expanded
		r.cache.InvalidateMatching(func(key string) bool {
			return strings.Contains(key, toolID)
		})
	}
}

// ExpandAll expands all tools
func (r *ToolRenderer) ExpandAll() {
	// This would need to know all tool IDs, so it's mainly for UI controls
	r.cache.Clear()
}

// CollapseAll collapses all tools
func (r *ToolRenderer) CollapseAll() {
	r.expandedTools = make(map[string]bool)
	r.cache.Clear()
}

// formatParameters formats tool parameters for display
func (r *ToolRenderer) formatParameters(params map[string]interface{}) string {
	// Try to format as JSON first
	jsonBytes, err := json.MarshalIndent(params, "", "  ")
	if err == nil {
		return string(jsonBytes)
	}
	
	// Fallback to simple key-value format
	var lines []string
	for key, value := range params {
		lines = append(lines, fmt.Sprintf("%s: %v", key, value))
	}
	return strings.Join(lines, "\n")
}

// formatOutput formats tool output for display
func (r *ToolRenderer) formatOutput(output interface{}) string {
	switch v := output.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case error:
		return v.Error()
	default:
		// Try JSON formatting
		jsonBytes, err := json.MarshalIndent(output, "", "  ")
		if err == nil {
			return string(jsonBytes)
		}
		// Fallback to fmt
		return fmt.Sprintf("%v", output)
	}
}

// getOutputPreview gets a brief preview of the output
func (r *ToolRenderer) getOutputPreview(output interface{}) string {
	str := r.formatOutput(output)
	
	// Remove newlines for compact view
	str = strings.ReplaceAll(str, "\n", " ")
	str = strings.TrimSpace(str)
	
	// Truncate if too long
	const maxLen = 50
	if len(str) > maxLen {
		return str[:maxLen-3] + "..."
	}
	
	return str
}

// ParseToolContent parses tool content from various formats
func ParseToolContent(content string) (*ToolInvocation, error) {
	// Try to parse as JSON first
	var tool ToolInvocation
	if err := json.Unmarshal([]byte(content), &tool); err == nil {
		return &tool, nil
	}
	
    // Try to parse XML-style format with regex for attributes and input content
    // <tool name="calculator"><input>2+2</input></tool>
    toolTag := regexp.MustCompile(`(?s)^<tool\s+([^>]*)>(.*)</tool>$`)
    if m := toolTag.FindStringSubmatch(content); len(m) == 3 {
        attrs, body := m[1], m[2]
        nameRe := regexp.MustCompile(`name\s*=\s*"([^"]+)"`)
        if nm := nameRe.FindStringSubmatch(attrs); len(nm) == 2 {
            tool.Name = nm[1]
        }
        // Extract <input>...</input> if present; otherwise use full body
        inputRe := regexp.MustCompile(`(?s)<input>(.*)</input>`)
        if im := inputRe.FindStringSubmatch(body); len(im) == 2 {
            trimmed := strings.TrimSpace(im[1])
            var params map[string]interface{}
            if (strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}")) && json.Unmarshal([]byte(trimmed), &params) == nil {
                tool.Parameters = params
            } else {
                tool.Parameters = map[string]interface{}{"input": trimmed}
            }
        } else {
            tool.Parameters = map[string]interface{}{"input": strings.TrimSpace(body)}
        }
        if tool.Name != "" {
            return &tool, nil
        }
    }
	
	return nil, fmt.Errorf("unable to parse tool content")
}

// ParseToolResult parses tool result content
func ParseToolResult(content string) (*ToolResult, error) {
	var result ToolResult
	
	// Try XML-style first to extract inner content
	innerContent := content
	if strings.HasPrefix(content, "<tool_result>") && strings.HasSuffix(content, "</tool_result>") {
		start := len("<tool_result>")
		end := strings.LastIndex(content, "</tool_result>")
		if end > start {
			innerContent = strings.TrimSpace(content[start:end])
		}
	}
	
	// Try JSON on the inner content
	if err := json.Unmarshal([]byte(innerContent), &result); err == nil {
		return &result, nil
	}
	
	// Fallback: treat inner content as output
	result.Output = innerContent
	return &result, nil
}