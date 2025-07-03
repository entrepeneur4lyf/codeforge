package chat

import (
	"strings"
	"testing"
)

func TestMessageParser(t *testing.T) {
	parser := NewMessageParser()

	t.Run("empty message", func(t *testing.T) {
		msg := Message{ID: "test-1", Content: ""}
		parts := parser.ParseMessage(msg)
		if len(parts) != 0 {
			t.Errorf("Expected 0 parts for empty message, got %d", len(parts))
		}
	})

	t.Run("plain text message", func(t *testing.T) {
		msg := Message{
			ID:      "test-2",
			Content: "This is a plain text message with no special formatting.",
		}
		parts := parser.ParseMessage(msg)

		if len(parts) != 1 {
			t.Fatalf("Expected 1 part, got %d", len(parts))
		}

		part := parts[0]
		if part.Type != PartTypeText {
			t.Errorf("Expected text type, got %s", part.Type)
		}
		if part.MessageID != "test-2" {
			t.Errorf("Expected message ID test-2, got %s", part.MessageID)
		}
		if part.PartIndex != 0 {
			t.Errorf("Expected part index 0, got %d", part.PartIndex)
		}
		if strings.TrimSpace(part.Content) != strings.TrimSpace(msg.Content) {
			t.Errorf("Content mismatch:\nExpected: %q\nGot: %q", msg.Content, part.Content)
		}
	})

	t.Run("message with code block", func(t *testing.T) {
		msg := Message{
			ID: "test-3",
			Content: `Here's some code:

` + "```go" + `
func main() {
    fmt.Println("Hello, World!")
}
` + "```" + `

And some text after.`,
		}

		parts := parser.ParseMessage(msg)

		// Should have 3 parts: text, code, text
		if len(parts) != 3 {
			t.Fatalf("Expected 3 parts, got %d", len(parts))
		}

		// First part: text before code
		if parts[0].Type != PartTypeText {
			t.Errorf("Expected first part to be text, got %s", parts[0].Type)
		}
		if !strings.Contains(parts[0].Content, "Here's some code:") {
			t.Errorf("First text part missing expected content")
		}

		// Second part: code block
		if parts[1].Type != PartTypeCode {
			t.Errorf("Expected second part to be code, got %s", parts[1].Type)
		}
		if !strings.Contains(parts[1].Content, "func main()") {
			t.Errorf("Code part missing expected content")
		}
		if parts[1].Metadata["language"] != "go" {
			t.Errorf("Expected language 'go', got %v", parts[1].Metadata["language"])
		}

		// Third part: text after code
		if parts[2].Type != PartTypeText {
			t.Errorf("Expected third part to be text, got %s", parts[2].Type)
		}
		if !strings.Contains(parts[2].Content, "And some text after") {
			t.Errorf("Last text part missing expected content")
		}
	})

	t.Run("message with multiple code blocks", func(t *testing.T) {
		msg := Message{
			ID: "test-4",
			Content: `First code:

` + "```python" + `
print("Python")
` + "```" + `

Middle text.

` + "```javascript" + `
console.log("JavaScript")
` + "```" + `

End text.`,
		}

		parts := parser.ParseMessage(msg)

		// Should have 5 parts: text, code, text, code, text
		if len(parts) != 5 {
			t.Fatalf("Expected 5 parts, got %d", len(parts))
		}

		// Check code blocks
		if parts[1].Type != PartTypeCode || parts[1].Metadata["language"] != "python" {
			t.Errorf("First code block incorrect")
		}
		if parts[3].Type != PartTypeCode || parts[3].Metadata["language"] != "javascript" {
			t.Errorf("Second code block incorrect")
		}
	})

	t.Run("message with tool invocation", func(t *testing.T) {
		msg := Message{
			ID: "test-5",
			Content: `I'll help you with that.

<tool name="calculator">
  <input>2 + 2</input>
</tool>

The result is 4.`,
		}

		parts := parser.ParseMessage(msg)

		// Should have 3 parts: text, tool, text
		if len(parts) < 3 {
			t.Fatalf("Expected at least 3 parts, got %d", len(parts))
		}

		// Find the tool part
		var toolPart *MessagePart
		for i, part := range parts {
			if part.Type == PartTypeTool {
				toolPart = &parts[i]
				break
			}
		}

		if toolPart == nil {
			t.Error("No tool part found")
		} else if !strings.Contains(toolPart.Content, "calculator") {
			t.Errorf("Tool part missing expected content")
		}
	})

	t.Run("message with tool result", func(t *testing.T) {
		msg := Message{
			ID: "test-6",
			Content: `Here's the calculation:

<tool_result>
Result: 42
</tool_result>

That's the answer.`,
		}

		parts := parser.ParseMessage(msg)

		// Find the tool result part
		var resultPart *MessagePart
		for i, part := range parts {
			if part.Type == PartTypeToolResult {
				resultPart = &parts[i]
				break
			}
		}

		if resultPart == nil {
			t.Error("No tool result part found")
		} else if !strings.Contains(resultPart.Content, "Result: 42") {
			t.Errorf("Tool result part missing expected content")
		}
	})

	t.Run("message with diff", func(t *testing.T) {
		msg := Message{
			ID: "test-7",
			Content: `Here's the change:

` + "```diff" + `
- old line
+ new line
` + "```" + `

Applied successfully.`,
		}

		parts := parser.ParseMessage(msg)

		// Find the diff part
		var diffPart *MessagePart
		for i, part := range parts {
			if part.Type == PartTypeDiff {
				diffPart = &parts[i]
				break
			}
		}

		if diffPart == nil {
			t.Error("No diff part found")
		} else {
			if !strings.Contains(diffPart.Content, "- old line") {
				t.Errorf("Diff part missing old line")
			}
			if !strings.Contains(diffPart.Content, "+ new line") {
				t.Errorf("Diff part missing new line")
			}
		}
	})

	t.Run("complex message with mixed content", func(t *testing.T) {
		msg := Message{
			ID: "test-8",
			Content: `Let me analyze this code:

` + "```go" + `
func add(a, b int) int {
    return a + b
}
` + "```" + `

<tool name="analyzer">
  <code>func add</code>
</tool>

<tool_result>
Function is correct
</tool_result>

Here's an improved version:

` + "```diff" + `
- func add(a, b int) int {
+ func add(a, b int) (int, error) {
+     if a > 1000000 || b > 1000000 {
+         return 0, errors.New("number too large")
+     }
      return a + b
  }
` + "```" + `

The changes add error handling.`,
		}

		parts := parser.ParseMessage(msg)

		// Count part types
		typeCounts := make(map[MessagePartType]int)
		for _, part := range parts {
			typeCounts[part.Type]++
		}

		// Should have various part types
		if typeCounts[PartTypeCode] < 1 {
			t.Error("Expected at least 1 code part")
		}
		if typeCounts[PartTypeTool] != 1 {
			t.Errorf("Expected 1 tool part, got %d", typeCounts[PartTypeTool])
		}
		if typeCounts[PartTypeToolResult] != 1 {
			t.Errorf("Expected 1 tool result part, got %d", typeCounts[PartTypeToolResult])
		}
		if typeCounts[PartTypeDiff] != 1 {
			t.Errorf("Expected 1 diff part, got %d", typeCounts[PartTypeDiff])
		}
		if typeCounts[PartTypeText] < 1 {
			t.Error("Expected at least 1 text part")
		}
	})

	t.Run("part indices are sequential", func(t *testing.T) {
		msg := Message{
			ID: "test-9",
			Content: `Text 1

` + "```code" + `
Code block
` + "```" + `

Text 2

<tool>Tool</tool>

Text 3`,
		}

		parts := parser.ParseMessage(msg)

		// Check that part indices are sequential
		for i, part := range parts {
			if part.PartIndex != i {
				t.Errorf("Expected part index %d, got %d", i, part.PartIndex)
			}
			if part.MessageID != "test-9" {
				t.Errorf("Expected message ID test-9, got %s", part.MessageID)
			}
		}
	})

	t.Run("line numbers are correct", func(t *testing.T) {
		msg := Message{
			ID: "test-10",
			Content: `Line 1
Line 2

` + "```" + `
Code line 1
Code line 2
Code line 3
` + "```" + `

Line after code`,
		}

		parts := parser.ParseMessage(msg)

		// First text part should start at line 0
		if len(parts) > 0 && parts[0].StartLine != 0 {
			t.Errorf("First part should start at line 0, got %d", parts[0].StartLine)
		}

		// Check that line numbers progress correctly
		for i := 1; i < len(parts); i++ {
			// Each part should start after the previous one ends
			if parts[i].StartLine < parts[i-1].EndLine {
				t.Errorf("Part %d starts at line %d but previous part ends at line %d",
					i, parts[i].StartLine, parts[i-1].EndLine)
			}
		}
	})
}

func TestExtractCodeBlocks(t *testing.T) {
	parser := NewMessageParser()

	t.Run("single code block", func(t *testing.T) {
		content := "Text before\n\n```go\nfunc main() {}\n```\n\nText after"
		blocks := parser.ExtractCodeBlocks(content)

		if len(blocks) != 1 {
			t.Fatalf("Expected 1 block, got %d", len(blocks))
		}

		if blocks[0].Language != "go" {
			t.Errorf("Expected language 'go', got %s", blocks[0].Language)
		}
		if strings.TrimSpace(blocks[0].Code) != "func main() {}" {
			t.Errorf("Expected 'func main() {}', got %s", blocks[0].Code)
		}
	})

	t.Run("multiple code blocks", func(t *testing.T) {
		content := `
` + "```python" + `
print("Hello")
` + "```" + `

Some text

` + "```javascript" + `
console.log("World")
` + "```"

		blocks := parser.ExtractCodeBlocks(content)

		if len(blocks) != 2 {
			t.Fatalf("Expected 2 blocks, got %d", len(blocks))
		}

		// First block
		if blocks[0].Language != "python" {
			t.Errorf("First block: expected language 'python', got %s", blocks[0].Language)
		}
		if !strings.Contains(blocks[0].Code, "print") {
			t.Errorf("First block missing print statement")
		}

		// Second block
		if blocks[1].Language != "javascript" {
			t.Errorf("Second block: expected language 'javascript', got %s", blocks[1].Language)
		}
		if !strings.Contains(blocks[1].Code, "console.log") {
			t.Errorf("Second block missing console.log")
		}
	})

	t.Run("code block without language", func(t *testing.T) {
		content := "```\nplain code\n```"
		blocks := parser.ExtractCodeBlocks(content)

		if len(blocks) != 1 {
			t.Fatalf("Expected 1 block, got %d", len(blocks))
		}

		if blocks[0].Language != "" {
			t.Errorf("Expected empty language, got %s", blocks[0].Language)
		}
		if strings.TrimSpace(blocks[0].Code) != "plain code" {
			t.Errorf("Code content mismatch")
		}
	})

	t.Run("no code blocks", func(t *testing.T) {
		content := "Just plain text with no code blocks"
		blocks := parser.ExtractCodeBlocks(content)

		if len(blocks) != 0 {
			t.Errorf("Expected 0 blocks, got %d", len(blocks))
		}
	})
}

// func TestSortBlocks(t *testing.T) {
// 	blocks := []block{
// 		{start: 30, end: 40, partType: PartTypeCode},
// 		{start: 10, end: 20, partType: PartTypeText},
// 		{start: 50, end: 60, partType: PartTypeTool},
// 		{start: 5, end: 8, partType: PartTypeText},
// 	}

// 	sortBlocks(blocks)

// 	// Check that blocks are sorted by start position
// 	for i := 1; i < len(blocks); i++ {
// 		if blocks[i].start < blocks[i-1].start {
// 			t.Errorf("Blocks not sorted: block %d starts at %d, block %d starts at %d",
// 				i-1, blocks[i-1].start, i, blocks[i].start)
// 		}
// 	}

// 	// Verify exact order
// 	expectedStarts := []int{5, 10, 30, 50}
// 	for i, block := range blocks {
// 		if block.start != expectedStarts[i] {
// 			t.Errorf("Block %d: expected start %d, got %d", i, expectedStarts[i], block.start)
// 		}
// 	}
// }

func TestOverlappingBlocks(t *testing.T) {
	parser := NewMessageParser()

	t.Run("diff inside code block", func(t *testing.T) {
		// When a diff pattern is inside a code block, it should be treated as code, not diff
		msg := Message{
			ID: "test-overlap",
			Content: "Here's a code example:\n\n```\n```diff\n- old\n+ new\n```\n```\n\nDone.",
		}

		parts := parser.ParseMessage(msg)

		// Should not have a separate diff part
		hasDiff := false
		for _, part := range parts {
			if part.Type == PartTypeDiff {
				hasDiff = true
			}
		}

		if hasDiff {
			t.Error("Should not detect diff when it's inside a code block")
		}
	})
}

// Helper function for debugging - not a test
func debugPrintParts(parts []MessagePart) {
	for i, part := range parts {
		contentPreview := part.Content
		if len(contentPreview) > 50 {
			contentPreview = contentPreview[:47] + "..."
		}
		contentPreview = strings.ReplaceAll(contentPreview, "\n", "\\n")
		
		println("Part", i, ":")
		println("  Type:", string(part.Type))
		println("  Lines:", part.StartLine, "-", part.EndLine)
		println("  Content:", contentPreview)
		if part.Metadata != nil && len(part.Metadata) > 0 {
			println("  Metadata:", part.Metadata)
		}
	}
}

// TestMessagePartTypes verifies all part type constants
func TestMessagePartTypes(t *testing.T) {
	expectedTypes := map[MessagePartType]string{
		PartTypeText:       "text",
		PartTypeCode:       "code",
		PartTypeTool:       "tool",
		PartTypeToolResult: "tool_result",
		PartTypeDiff:       "diff",
		PartTypeFile:       "file",
		PartTypeError:      "error",
	}

	for partType, expected := range expectedTypes {
		if string(partType) != expected {
			t.Errorf("Part type %s has unexpected value: %s", expected, string(partType))
		}
	}
}