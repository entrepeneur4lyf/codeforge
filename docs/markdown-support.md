# Markdown Support in CodeForge API Chat

CodeForge now includes comprehensive markdown support for the API chat system, providing rich text formatting and enhanced readability for chat messages.

## Features

### Multi-Format Message Processing
- **Plain Text**: Original message content
- **Markdown**: Raw markdown format
- **Terminal**: Terminal-optimized rendering with colors and formatting
- **HTML**: Web-ready HTML output with styling

### Automatic Format Detection
The system automatically detects markdown content and processes it accordingly:
- Headers (`#`, `##`, `###`)
- Bold (`**text**`) and italic (`*text*`)
- Code blocks (` ```language `)
- Inline code (`` `code` ``)
- Lists (`-`, `*`, `1.`)
- Links (`[text](url)`)
- Blockquotes (`>`)

### Smart Format Selection
The system can automatically select the best format based on:
- Client type (web, terminal, api)
- Accept headers (text/html, text/plain, etc.)
- User preferences

## API Endpoints

### Enhanced Chat Endpoint
```
POST /api/v1/chat/sessions/{sessionID}/messages/enhanced
```

**Request Body:**
```json
{
  "message": "# Hello\nThis is **markdown** content with `code`",
  "model": "gpt-4",
  "provider": "openai",
  "context": {
    "format_preference": "html"
  }
}
```

**Response:**
```json
{
  "id": "msg_123",
  "session_id": "session_456",
  "role": "assistant",
  "content": {
    "plain": "Response in plain text",
    "markdown": "Response in **markdown**",
    "terminal": "Response with terminal formatting",
    "html": "<div>Response in HTML</div>"
  },
  "timestamp": "2024-01-01T12:00:00Z",
  "model": "gpt-4",
  "metadata": {
    "markdown": true,
    "available_formats": ["plain", "markdown", "terminal", "html"],
    "formats": {
      "plain": "...",
      "markdown": "...",
      "terminal": "...",
      "html": "..."
    }
  }
}
```

### WebSocket Support
WebSocket messages now include enhanced markdown processing:

```javascript
// Send message
ws.send(JSON.stringify({
  type: "chat_message",
  data: {
    message: "# Hello\nThis is **markdown** content"
  }
}));

// Receive enhanced response
{
  "type": "chat_response",
  "data": {
    "content": {
      "plain": "...",
      "markdown": "...",
      "terminal": "...",
      "html": "..."
    },
    "metadata": {
      "markdown": true,
      "available_formats": ["plain", "markdown", "terminal", "html"]
    }
  }
}
```

## Code Examples

### Basic Usage
```go
import "github.com/entrepeneur4lyf/codeforge/internal/markdown"

// Create processor
processor, err := markdown.NewMessageProcessor()
if err != nil {
    log.Fatal(err)
}

// Process message
result, err := processor.ProcessMessage("# Hello\nThis is **markdown**")
if err != nil {
    log.Fatal(err)
}

// Access different formats
plainText := result["plain"]
htmlOutput := result["html"]
terminalOutput := result["terminal"]
```

### Format Selection
```go
selector := markdown.FormatSelector{
    ClientType: "web",
    AcceptTypes: []string{"text/html"},
}

availableFormats := []string{"plain", "markdown", "terminal", "html"}
bestFormat := selector.SelectBestFormat(availableFormats)
// Returns: "html"
```

### Code Block Extraction
```go
codeBlocks := processor.ExtractCodeBlocks(markdownContent)
for _, block := range codeBlocks {
    fmt.Printf("Language: %s\nCode: %s\n", block.Language, block.Code)
}
```

## Configuration

### Renderer Configuration
```go
config := &markdown.RendererConfig{
    Width: 80,
    Theme: "auto", // "dark", "light", or "auto"
}

renderer, err := markdown.NewRenderer(config)
```

### Built-in Configurations
- `markdown.ChatConfig()`: Optimized for chat display
- `markdown.WebConfig()`: Optimized for web rendering
- `markdown.DefaultConfig()`: General purpose configuration

## Integration

### Backward Compatibility
The enhanced markdown system maintains full backward compatibility:
- Existing `/chat/sessions/{id}/messages` endpoint unchanged
- Plain text messages work exactly as before
- WebSocket clients receive enhanced data but can ignore it

### Performance
- Lazy rendering: Formats are only generated when requested
- Caching: Rendered content is cached for repeated access
- Efficient detection: Fast markdown pattern matching

## Testing

Run the markdown tests:
```bash
go test ./internal/markdown -v
```

Run the demo:
```bash
go run examples/markdown_demo.go
```

## Dependencies

- `github.com/charmbracelet/glamour`: Terminal markdown rendering
- Built-in Go regex: Markdown pattern detection
- Standard library: HTML escaping and processing

The markdown support enhances CodeForge's chat capabilities while maintaining simplicity and performance.
