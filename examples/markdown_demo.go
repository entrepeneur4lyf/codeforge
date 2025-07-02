package main

import (
	"fmt"
	"log"

	"github.com/entrepeneur4lyf/codeforge/internal/markdown"
)

func main() {
	// Create a message processor
	processor, err := markdown.NewMessageProcessor()
	if err != nil {
		log.Fatalf("Failed to create message processor: %v", err)
	}

	// Example markdown content
	markdownContent := `# CodeForge Markdown Demo

Welcome to **CodeForge**! This is a demonstration of our markdown processing capabilities.

## Features

- **Bold** and *italic* text support
- Code blocks with syntax highlighting
- Lists and headers
- Links and more!

### Code Example

Here's some Go code:

` + "```go" + `
func main() {
    fmt.Println("Hello, CodeForge!")
}
` + "```" + `

### JavaScript Example

And some JavaScript:

` + "```javascript" + `
function greet(name) {
    console.log(` + "`Hello, ${name}!`" + `);
}
greet("CodeForge");
` + "```" + `

## API Usage

You can use inline code like ` + "`processor.ProcessMessage()`" + ` in your text.

> This is a blockquote showing how markdown enhances readability.

### Links

Check out [CodeForge](https://github.com/entrepeneur4lyf/codeforge) on GitHub!

---

*This demo shows how CodeForge processes markdown in real-time chat.*`

	// Process the markdown content
	fmt.Println("🔄 Processing markdown content...")
	result, err := processor.ProcessMessage(markdownContent)
	if err != nil {
		log.Fatalf("Failed to process message: %v", err)
	}

	// Display results
	fmt.Println("\n📋 Available Formats:")
	for format := range result {
		fmt.Printf("  - %s\n", format)
	}

	fmt.Println("\n📝 Plain Text Format:")
	fmt.Println("=" + string(make([]rune, 50)) + "=")
	fmt.Println(result["plain"])

	fmt.Println("\n🎨 Terminal Rendered Format:")
	fmt.Println("=" + string(make([]rune, 50)) + "=")
	fmt.Println(result["terminal"])

	fmt.Println("\n🌐 HTML Format (first 200 chars):")
	fmt.Println("=" + string(make([]rune, 50)) + "=")
	htmlContent := result["html"]
	if len(htmlContent) > 200 {
		fmt.Printf("%s...\n", htmlContent[:200])
	} else {
		fmt.Println(htmlContent)
	}

	// Demonstrate code block extraction
	fmt.Println("\n🔍 Extracted Code Blocks:")
	fmt.Println("=" + string(make([]rune, 50)) + "=")
	codeBlocks := processor.ExtractCodeBlocks(markdownContent)
	for i, block := range codeBlocks {
		fmt.Printf("Block %d (%s):\n%s\n\n", i+1, block.Language, block.Code)
	}

	// Demonstrate format selection
	fmt.Println("🎯 Format Selection Examples:")
	fmt.Println("=" + string(make([]rune, 50)) + "=")

	selectors := []markdown.FormatSelector{
		{ClientType: "web"},
		{ClientType: "terminal"},
		{ClientType: "api"},
		{AcceptTypes: []string{"text/html"}},
		{AcceptTypes: []string{"text/plain"}},
	}

	availableFormats := []string{"plain", "markdown", "terminal", "html"}

	for _, selector := range selectors {
		bestFormat := selector.SelectBestFormat(availableFormats)
		fmt.Printf("Client: %s, Accept: %v -> Best Format: %s\n",
			selector.ClientType, selector.AcceptTypes, bestFormat)
	}

	fmt.Println("\n✅ Markdown processing demo completed successfully!")
}
