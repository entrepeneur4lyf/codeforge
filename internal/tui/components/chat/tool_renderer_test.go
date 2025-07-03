package chat

import (
	"strings"
	"testing"

	"github.com/entrepeneur4lyf/codeforge/internal/tui/theme"
)

func TestToolRenderer(t *testing.T) {
	th := theme.GetTheme("default")

	t.Run("creation", func(t *testing.T) {
		renderer := NewToolRenderer(th)
		
		if renderer.theme == nil {
			t.Error("Expected theme to be set")
		}
		if renderer.expandedTools == nil {
			t.Error("Expected expandedTools map to be initialized")
		}
		if renderer.cache == nil {
			t.Error("Expected cache to be initialized")
		}
		if renderer.codeRenderer == nil {
			t.Error("Expected codeRenderer to be initialized")
		}
		if renderer.resultRenderer == nil {
			t.Error("Expected resultRenderer to be initialized")
		}
	})

	t.Run("render tool invocation collapsed", func(t *testing.T) {
		renderer := NewToolRenderer(th)
		
		tool := ToolInvocation{
			ID:   "tool-1",
			Name: "calculator",
			Parameters: map[string]interface{}{
				"expression": "2 + 2",
				"precision":  2,
			},
		}
		
		rendered := renderer.RenderToolInvocation(tool, false)
		stripped := stripANSI(rendered)
		
		// Should show collapsed indicator
		if !strings.Contains(stripped, "▶") {
			t.Error("Expected collapsed indicator ▶")
		}
		
		// Should show tool name
		if !strings.Contains(stripped, "calculator") {
			t.Error("Expected tool name in output")
		}
		
		// Should show parameter count
		if !strings.Contains(stripped, "2 parameter") {
			t.Error("Expected parameter count in collapsed view")
		}
		
		// Should NOT show parameter details
		if strings.Contains(stripped, "expression") {
			t.Error("Should not show parameter details when collapsed")
		}
	})

	t.Run("render tool invocation expanded", func(t *testing.T) {
		renderer := NewToolRenderer(th)
		
		tool := ToolInvocation{
			ID:   "tool-2",
			Name: "file_reader",
			Parameters: map[string]interface{}{
				"path": "/home/user/file.txt",
				"mode": "read",
			},
			Timestamp: "2024-01-01 12:00:00",
		}
		
		rendered := renderer.RenderToolInvocation(tool, true)
		stripped := stripANSI(rendered)
		
		// Should show expanded indicator
		if !strings.Contains(stripped, "▼") {
			t.Error("Expected expanded indicator ▼")
		}
		
		// Should show parameter details
		if !strings.Contains(stripped, "path") {
			t.Error("Expected parameter name 'path' in expanded view")
		}
		if !strings.Contains(stripped, "/home/user/file.txt") {
			t.Error("Expected parameter value in expanded view")
		}
		
		// Should show timestamp
		if !strings.Contains(stripped, "Called at:") && !strings.Contains(stripped, "2024-01-01") {
			t.Error("Expected timestamp in expanded view")
		}
	})

	t.Run("render tool result success collapsed", func(t *testing.T) {
		renderer := NewToolRenderer(th)
		
		result := ToolResult{
			ID:     "result-1",
			ToolID: "tool-1",
			Output: "4",
		}
		
		rendered := renderer.RenderToolResult(result, false)
		stripped := stripANSI(rendered)
		
		// Should show success status
		if !strings.Contains(stripped, "Success") {
			t.Error("Expected 'Success' status")
		}
		
		// Should show output preview
		if !strings.Contains(stripped, "→ 4") {
			t.Error("Expected output preview in collapsed view")
		}
		
		// Should have collapsed indicator
		if !strings.Contains(stripped, "▶") {
			t.Error("Expected collapsed indicator")
		}
	})

	t.Run("render tool result error expanded", func(t *testing.T) {
		renderer := NewToolRenderer(th)
		
		result := ToolResult{
			ID:       "result-2",
			ToolID:   "tool-2",
			Error:    "File not found: /invalid/path",
			Duration: "125ms",
		}
		
		rendered := renderer.RenderToolResult(result, true)
		stripped := stripANSI(rendered)
		
		// Should show error status
		if !strings.Contains(stripped, "Error") {
			t.Error("Expected 'Error' status")
		}
		
		// Should show expanded indicator
		if !strings.Contains(stripped, "▼") {
			t.Error("Expected expanded indicator")
		}
		
		// Should show error message
		if !strings.Contains(stripped, "File not found") {
			t.Error("Expected error message in expanded view")
		}
		
		// Should show duration
		if !strings.Contains(stripped, "125ms") {
			t.Error("Expected duration in expanded view")
		}
	})

	t.Run("toggle expanded state", func(t *testing.T) {
		renderer := NewToolRenderer(th)
		
		toolID := "tool-test"
		
		// Initially collapsed
		if renderer.IsExpanded(toolID) {
			t.Error("Expected tool to be collapsed initially")
		}
		
		// Toggle to expand
		renderer.ToggleExpanded(toolID)
		if !renderer.IsExpanded(toolID) {
			t.Error("Expected tool to be expanded after toggle")
		}
		
		// Toggle back to collapse
		renderer.ToggleExpanded(toolID)
		if renderer.IsExpanded(toolID) {
			t.Error("Expected tool to be collapsed after second toggle")
		}
	})

	t.Run("set expanded state", func(t *testing.T) {
		renderer := NewToolRenderer(th)
		
		toolID := "tool-set"
		
		// Set to expanded
		renderer.SetExpanded(toolID, true)
		if !renderer.IsExpanded(toolID) {
			t.Error("Expected tool to be expanded")
		}
		
		// Set to same state (should not clear cache)
		renderer.SetExpanded(toolID, true)
		if !renderer.IsExpanded(toolID) {
			t.Error("Expected tool to remain expanded")
		}
		
		// Set to collapsed
		renderer.SetExpanded(toolID, false)
		if renderer.IsExpanded(toolID) {
			t.Error("Expected tool to be collapsed")
		}
	})

	t.Run("expand and collapse all", func(t *testing.T) {
		renderer := NewToolRenderer(th)
		
		// Set some tools as expanded
		renderer.SetExpanded("tool1", true)
		renderer.SetExpanded("tool2", false)
		renderer.SetExpanded("tool3", true)
		
		// Collapse all
		renderer.CollapseAll()
		
		// All should be collapsed
		if renderer.IsExpanded("tool1") || renderer.IsExpanded("tool2") || renderer.IsExpanded("tool3") {
			t.Error("Expected all tools to be collapsed after CollapseAll")
		}
	})

	t.Run("format parameters", func(t *testing.T) {
		renderer := NewToolRenderer(th)
		
		params := map[string]interface{}{
			"string":  "hello",
			"number":  42,
			"boolean": true,
			"array":   []int{1, 2, 3},
		}
		
		formatted := renderer.formatParameters(params)
		
		// Should contain all keys
		if !strings.Contains(formatted, "string") {
			t.Error("Expected 'string' key in formatted output")
		}
		if !strings.Contains(formatted, "number") {
			t.Error("Expected 'number' key in formatted output")
		}
		if !strings.Contains(formatted, "boolean") {
			t.Error("Expected 'boolean' key in formatted output")
		}
		if !strings.Contains(formatted, "array") {
			t.Error("Expected 'array' key in formatted output")
		}
	})

	t.Run("format output", func(t *testing.T) {
		renderer := NewToolRenderer(th)
		
		// String output
		strOutput := renderer.formatOutput("Hello, world!")
		if strOutput != "Hello, world!" {
			t.Errorf("Expected string output to be unchanged, got %s", strOutput)
		}
		
		// Byte slice output
		byteOutput := renderer.formatOutput([]byte("byte content"))
		if byteOutput != "byte content" {
			t.Errorf("Expected byte output to be converted to string, got %s", byteOutput)
		}
		
		// Structured output
		structOutput := renderer.formatOutput(map[string]interface{}{
			"result": "success",
			"count":  10,
		})
		if !strings.Contains(structOutput, "result") || !strings.Contains(structOutput, "success") {
			t.Error("Expected structured output to be formatted")
		}
	})

	t.Run("output preview", func(t *testing.T) {
		renderer := NewToolRenderer(th)
		
		// Short output
		preview := renderer.getOutputPreview("Short output")
		if preview != "Short output" {
			t.Errorf("Expected short output unchanged, got %s", preview)
		}
		
		// Long output
		longOutput := strings.Repeat("Very long output ", 10)
		preview = renderer.getOutputPreview(longOutput)
		if len(preview) > 50 {
			t.Errorf("Expected preview to be truncated, got length %d", len(preview))
		}
		if !strings.HasSuffix(preview, "...") {
			t.Error("Expected truncated preview to end with ...")
		}
		
		// Multiline output
		multiline := "Line 1\nLine 2\nLine 3"
		preview = renderer.getOutputPreview(multiline)
		if strings.Contains(preview, "\n") {
			t.Error("Expected newlines to be removed in preview")
		}
	})

	t.Run("caching", func(t *testing.T) {
		renderer := NewToolRenderer(th)
		
		tool := ToolInvocation{
			ID:   "cache-test",
			Name: "test_tool",
			Parameters: map[string]interface{}{
				"param": "value",
			},
		}
		
		// First render (cache miss)
		result1 := renderer.RenderToolInvocation(tool, false)
		
		// Second render (cache hit)
		result2 := renderer.RenderToolInvocation(tool, false)
		
		if result1 != result2 {
			t.Error("Expected cached result to be identical")
		}
		
		// Toggle expansion clears cache for this tool
		renderer.ToggleExpanded(tool.ID)
		
		// Should get different result when expanded
		result3 := renderer.RenderToolInvocation(tool, true)
		if result3 == result1 {
			t.Error("Expected different result when expanded")
		}
	})
}

func TestParseToolContent(t *testing.T) {
	t.Run("parse JSON format", func(t *testing.T) {
		content := `{"id": "123", "name": "calculator", "parameters": {"expression": "2+2"}}`
		
		tool, err := ParseToolContent(content)
		if err != nil {
			t.Fatalf("Failed to parse JSON tool content: %v", err)
		}
		
		if tool.Name != "calculator" {
			t.Errorf("Expected name 'calculator', got %s", tool.Name)
		}
		
		if expr, ok := tool.Parameters["expression"].(string); !ok || expr != "2+2" {
			t.Error("Expected expression parameter '2+2'")
		}
	})

	t.Run("parse XML format", func(t *testing.T) {
		content := `<tool name="file_reader"><input>/path/to/file.txt</input></tool>`
		
		tool, err := ParseToolContent(content)
		if err != nil {
			t.Fatalf("Failed to parse XML tool content: %v", err)
		}
		
		if tool.Name != "file_reader" {
			t.Errorf("Expected name 'file_reader', got %s", tool.Name)
		}
		
		if input, ok := tool.Parameters["input"].(string); !ok || input != "/path/to/file.txt" {
			t.Error("Expected input parameter '/path/to/file.txt'")
		}
	})

	t.Run("parse XML with spaces", func(t *testing.T) {
		content := `<tool name="writer">
			<input>
				Some content with
				multiple lines
			</input>
		</tool>`
		
		tool, err := ParseToolContent(content)
		if err != nil {
			t.Fatalf("Failed to parse XML with spaces: %v", err)
		}
		
		if tool.Name != "writer" {
			t.Errorf("Expected name 'writer', got %s", tool.Name)
		}
		
		input, ok := tool.Parameters["input"].(string)
		if !ok {
			t.Fatal("Expected input parameter")
		}
		
		// Should preserve the content but trim outer whitespace
		if !strings.Contains(input, "Some content with") {
			t.Error("Expected content to be preserved")
		}
	})

	t.Run("parse invalid content", func(t *testing.T) {
		content := "This is not a valid tool format"
		
		_, err := ParseToolContent(content)
		if err == nil {
			t.Error("Expected error for invalid content")
		}
	})
}

func TestParseToolResult(t *testing.T) {
	t.Run("parse JSON format", func(t *testing.T) {
		content := `{"id": "r1", "tool_id": "t1", "output": {"result": 42}, "duration": "50ms"}`
		
		result, err := ParseToolResult(content)
		if err != nil {
			t.Fatalf("Failed to parse JSON result: %v", err)
		}
		
		if result.ToolID != "t1" {
			t.Errorf("Expected tool_id 't1', got %s", result.ToolID)
		}
		
		if result.Duration != "50ms" {
			t.Errorf("Expected duration '50ms', got %s", result.Duration)
		}
	})

	t.Run("parse XML format", func(t *testing.T) {
		content := `<tool_result>The operation completed successfully</tool_result>`
		
		result, err := ParseToolResult(content)
		if err != nil {
			t.Fatalf("Failed to parse XML result: %v", err)
		}
		
		output, ok := result.Output.(string)
		if !ok {
			t.Fatal("Expected string output")
		}
		
		if output != "The operation completed successfully" {
			t.Errorf("Expected output message, got %s", output)
		}
	})

	t.Run("parse plain text fallback", func(t *testing.T) {
		content := "Just a plain text result"
		
		result, err := ParseToolResult(content)
		if err != nil {
			t.Fatalf("Failed to parse plain text: %v", err)
		}
		
		if result.Output != content {
			t.Errorf("Expected output to be '%s', got %v", content, result.Output)
		}
	})
}

func TestToolRendererWithComplexData(t *testing.T) {
	th := theme.GetTheme("default")
	renderer := NewToolRenderer(th)

	t.Run("complex nested parameters", func(t *testing.T) {
		tool := ToolInvocation{
			ID:   "complex-1",
			Name: "api_caller",
			Parameters: map[string]interface{}{
				"endpoint": "/api/v1/users",
				"method":   "POST",
				"headers": map[string]interface{}{
					"Content-Type":  "application/json",
					"Authorization": "Bearer token123",
				},
				"body": map[string]interface{}{
					"name":  "John Doe",
					"email": "john@example.com",
					"roles": []string{"user", "admin"},
				},
			},
		}
		
		rendered := renderer.RenderToolInvocation(tool, true)
		stripped := stripANSI(rendered)
		
		// Should contain nested data
		if !strings.Contains(stripped, "endpoint") {
			t.Error("Expected endpoint in output")
		}
		if !strings.Contains(stripped, "Content-Type") {
			t.Error("Expected headers in output")
		}
		if !strings.Contains(stripped, "roles") {
			t.Error("Expected body data in output")
		}
	})

	t.Run("large output truncation", func(t *testing.T) {
		largeOutput := make(map[string]interface{})
		for i := 0; i < 100; i++ {
			largeOutput[strings.Repeat("key", 10)] = strings.Repeat("value", 20)
		}
		
		result := ToolResult{
			ID:     "large-1",
			ToolID: "tool-1",
			Output: largeOutput,
		}
		
		// Collapsed view should have truncated preview
		rendered := renderer.RenderToolResult(result, false)
		stripped := stripANSI(rendered)
		
		// Check that preview is truncated
		lines := strings.Split(stripped, "\n")
		if len(lines) > 2 { // Header + preview
			t.Error("Expected collapsed view to be compact")
		}
	})
}