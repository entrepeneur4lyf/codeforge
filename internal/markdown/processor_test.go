package markdown

import (
	"strings"
	"testing"
)

func TestMessageProcessor_ProcessMessage(t *testing.T) {
	processor, err := NewMessageProcessor()
	if err != nil {
		t.Fatalf("Failed to create message processor: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected map[string]bool // format -> should exist
	}{
		{
			name:  "Plain text",
			input: "Hello, this is a simple message.",
			expected: map[string]bool{
				"plain":    true,
				"markdown": true,
				"terminal": true,
				"html":     true,
			},
		},
		{
			name:  "Markdown with code",
			input: "Here's some code: `console.log('hello')`",
			expected: map[string]bool{
				"plain":    true,
				"markdown": true,
				"terminal": true,
				"html":     true,
			},
		},
		{
			name: "Markdown with code block",
			input: `Here's a code block:
` + "```javascript" + `
function hello() {
    console.log('Hello, world!');
}
` + "```",
			expected: map[string]bool{
				"plain":    true,
				"markdown": true,
				"terminal": true,
				"html":     true,
			},
		},
		{
			name:  "Markdown with headers",
			input: "# Main Title\n## Subtitle\nSome content here.",
			expected: map[string]bool{
				"plain":    true,
				"markdown": true,
				"terminal": true,
				"html":     true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.ProcessMessage(tt.input)
			if err != nil {
				t.Errorf("ProcessMessage() error = %v", err)
				return
			}

			// Check that all expected formats are present
			for format, shouldExist := range tt.expected {
				if shouldExist {
					if _, exists := result[format]; !exists {
						t.Errorf("Expected format %s to exist in result", format)
					}
					if result[format] == "" {
						t.Errorf("Expected format %s to have non-empty content", format)
					}
				}
			}

			// Verify plain text is always the original input
			if result["plain"] != tt.input {
				t.Errorf("Plain format should match original input")
			}
		})
	}
}

func TestMessageProcessor_ContainsMarkdown(t *testing.T) {
	processor, err := NewMessageProcessor()
	if err != nil {
		t.Fatalf("Failed to create message processor: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Plain text",
			input:    "Hello, this is a simple message.",
			expected: false,
		},
		{
			name:     "Inline code",
			input:    "Here's some code: `console.log('hello')`",
			expected: true,
		},
		{
			name:     "Bold text",
			input:    "This is **bold** text.",
			expected: true,
		},
		{
			name:     "Italic text",
			input:    "This is *italic* text.",
			expected: true,
		},
		{
			name:     "Header",
			input:    "# This is a header",
			expected: true,
		},
		{
			name:     "Code block",
			input:    "```\ncode here\n```",
			expected: true,
		},
		{
			name:     "Link",
			input:    "Check out [this link](https://example.com)",
			expected: true,
		},
		{
			name:     "List",
			input:    "- Item 1\n- Item 2",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.containsMarkdown(tt.input)
			if result != tt.expected {
				t.Errorf("containsMarkdown() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestMessageProcessor_ExtractCodeBlocks(t *testing.T) {
	processor, err := NewMessageProcessor()
	if err != nil {
		t.Fatalf("Failed to create message processor: %v", err)
	}

	input := `Here's some JavaScript:
` + "```javascript" + `
function hello() {
    console.log('Hello, world!');
}
` + "```" + `

And some Python:
` + "```python" + `
def hello():
    print("Hello, world!")
` + "```"

	blocks := processor.ExtractCodeBlocks(input)

	if len(blocks) != 2 {
		t.Errorf("Expected 2 code blocks, got %d", len(blocks))
	}

	if blocks[0].Language != "javascript" {
		t.Errorf("Expected first block language to be 'javascript', got '%s'", blocks[0].Language)
	}

	if blocks[1].Language != "python" {
		t.Errorf("Expected second block language to be 'python', got '%s'", blocks[1].Language)
	}

	if !strings.Contains(blocks[0].Code, "console.log") {
		t.Errorf("Expected first block to contain 'console.log'")
	}

	if !strings.Contains(blocks[1].Code, "print") {
		t.Errorf("Expected second block to contain 'print'")
	}
}

func TestFormatSelector_SelectBestFormat(t *testing.T) {
	tests := []struct {
		name            string
		selector        FormatSelector
		availableFormats []string
		expected        MessageFormat
	}{
		{
			name: "Web client prefers HTML",
			selector: FormatSelector{
				ClientType: "web",
			},
			availableFormats: []string{"plain", "markdown", "terminal", "html"},
			expected:         FormatHTML,
		},
		{
			name: "Terminal client prefers terminal",
			selector: FormatSelector{
				ClientType: "terminal",
			},
			availableFormats: []string{"plain", "markdown", "terminal", "html"},
			expected:         FormatTerminal,
		},
		{
			name: "API client prefers markdown",
			selector: FormatSelector{
				ClientType: "api",
			},
			availableFormats: []string{"plain", "markdown", "terminal", "html"},
			expected:         FormatMarkdown,
		},
		{
			name: "HTML accept type",
			selector: FormatSelector{
				AcceptTypes: []string{"text/html"},
			},
			availableFormats: []string{"plain", "markdown", "terminal", "html"},
			expected:         FormatHTML,
		},
		{
			name: "Fallback to plain",
			selector: FormatSelector{
				ClientType: "unknown",
			},
			availableFormats: []string{"plain"},
			expected:         FormatPlain,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.selector.SelectBestFormat(tt.availableFormats)
			if result != tt.expected {
				t.Errorf("SelectBestFormat() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
