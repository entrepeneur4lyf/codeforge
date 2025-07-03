package chat

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/theme"
)

// FileContent represents file content to be displayed
type FileContent struct {
	Path      string
	Content   string
	Language  string
	StartLine int
	EndLine   int
	Highlight []int // Lines to highlight
}

// FileRenderer renders file contents with syntax highlighting
type FileRenderer struct {
	theme        theme.Theme
	cache        *MessageCache
	codeRenderer *BlockRenderer
	
	// Chroma configuration
	style     *chroma.Style
	formatter chroma.Formatter
	
	// Display options
	showLineNumbers bool
	maxWidth        int
}

// NewFileRenderer creates a new file renderer
func NewFileRenderer(theme theme.Theme) *FileRenderer {
	// Use a terminal-friendly style
	chromaStyle := styles.Get("monokai")
	if chromaStyle == nil {
		chromaStyle = styles.Fallback
	}
	
	return &FileRenderer{
		theme:           theme,
		cache:           NewMessageCache(50),
		codeRenderer:    CodeBlockRenderer(theme),
		style:           chromaStyle,
		formatter:       formatters.Get("terminal256"),
		showLineNumbers: true,
		maxWidth:        120,
	}
}

// RenderFile renders a file with syntax highlighting
func (r *FileRenderer) RenderFile(file FileContent) string {
	cacheKey := r.cache.GenerateKey(
		file.Path,
		file.Content,
		fmt.Sprintf("%d-%d", file.StartLine, file.EndLine),
		fmt.Sprintf("%v", file.Highlight),
		fmt.Sprintf("ln:%v", r.showLineNumbers),
	)
	
	if cached, found := r.cache.Get(cacheKey); found {
		return cached
	}
	
	var result strings.Builder
	
	// Render file header
	header := r.renderFileHeader(file)
	result.WriteString(header)
	result.WriteString("\n\n")
	
	// Render content with syntax highlighting
	content := r.renderContent(file)
	result.WriteString(content)
	
	rendered := result.String()
	r.cache.Set(cacheKey, rendered)
	return rendered
}

// renderFileHeader renders the file path and metadata
func (r *FileRenderer) renderFileHeader(file FileContent) string {
	// Create header with file icon and path
	icon := r.getFileIcon(file.Path)
	
	headerStyle := lipgloss.NewStyle().
		Foreground(r.theme.Primary()).
		Bold(true).
		Border(lipgloss.Border{
			Bottom: "─",
		}, false, false, true, false).
		BorderForeground(r.theme.Primary()).
		Width(r.maxWidth)
	
	// Build header content
	var header strings.Builder
	header.WriteString(icon)
	header.WriteString(" ")
	header.WriteString(file.Path)
	
	// Add line range if specified
	if file.StartLine > 0 || file.EndLine > 0 {
		rangeStyle := lipgloss.NewStyle().
			Foreground(r.theme.TextMuted()).
			Italic(true)
		
		if file.StartLine > 0 && file.EndLine > 0 {
			header.WriteString(" ")
			header.WriteString(rangeStyle.Render(fmt.Sprintf("(lines %d-%d)", file.StartLine, file.EndLine)))
		} else if file.StartLine > 0 {
			header.WriteString(" ")
			header.WriteString(rangeStyle.Render(fmt.Sprintf("(from line %d)", file.StartLine)))
		}
	}
	
	return headerStyle.Render(header.String())
}

// renderContent renders the file content with syntax highlighting
func (r *FileRenderer) renderContent(file FileContent) string {
	// Detect language if not specified
	language := file.Language
	if language == "" {
		language = r.detectLanguage(file.Path, file.Content)
	}
	
	// Get lexer for the language
	lexer := lexers.Get(language)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)
	
	// Split content into lines for line number handling
	lines := strings.Split(file.Content, "\n")
	
	// Determine line range
	startLine := 1
	endLine := len(lines)
	if file.StartLine > 0 {
		startLine = file.StartLine
	}
	if file.EndLine > 0 && file.EndLine < endLine {
		endLine = file.EndLine
	}
	
	// Extract relevant lines
	if startLine > 1 || endLine < len(lines) {
		// If the requested range is beyond the file, use the whole file
		if startLine > len(lines) {
			startLine = 1
			endLine = len(lines)
		} else {
			if endLine > len(lines) {
				endLine = len(lines)
			}
			lines = lines[startLine-1 : endLine]
		}
	}
	
	// Apply syntax highlighting
	highlighted, err := r.highlightCode(strings.Join(lines, "\n"), lexer)
	if err != nil {
		// Fallback to plain rendering
		highlighted = strings.Join(lines, "\n")
	}
	
	// Add line numbers if enabled
	if r.showLineNumbers {
		highlighted = r.addLineNumbers(highlighted, startLine, file.Highlight)
	}
	
	// Wrap in code block style
	codeStyle := lipgloss.NewStyle().
		Background(r.theme.BackgroundSecondary()).
		Padding(1).
		Border(lipgloss.NormalBorder()).
		BorderForeground(r.theme.TextMuted())
	
	return codeStyle.Render(highlighted)
}

// highlightCode applies syntax highlighting to code
func (r *FileRenderer) highlightCode(code string, lexer chroma.Lexer) (string, error) {
	// Tokenize
	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return code, err
	}
	
	// Format
	var buf strings.Builder
	err = r.formatter.Format(&buf, r.style, iterator)
	if err != nil {
		return code, err
	}
	
	return buf.String(), nil
}

// addLineNumbers adds line numbers to highlighted code
func (r *FileRenderer) addLineNumbers(code string, startLine int, highlightLines []int) string {
	lines := strings.Split(code, "\n")
	
	// Calculate line number width
	maxLineNum := startLine + len(lines) - 1
	lineNumWidth := len(fmt.Sprintf("%d", maxLineNum))
	
	// Create highlight set for quick lookup
	highlightSet := make(map[int]bool)
	for _, line := range highlightLines {
		highlightSet[line] = true
	}
	
	var result strings.Builder
	for i, line := range lines {
		lineNum := startLine + i
		
		// Line number style
		lineNumStyle := lipgloss.NewStyle().
			Foreground(r.theme.TextMuted()).
			Width(lineNumWidth).
			Align(lipgloss.Right)
		
		// Highlight style for highlighted lines
		if highlightSet[lineNum] {
			lineNumStyle = lineNumStyle.
				Foreground(r.theme.Warning()).
				Bold(true)
			
			// Add highlight indicator
			highlightStyle := lipgloss.NewStyle().
				Foreground(r.theme.Warning()).
				Bold(true)
			result.WriteString(highlightStyle.Render("→ "))
		} else {
			result.WriteString("  ")
		}
		
		// Add line number
		result.WriteString(lineNumStyle.Render(fmt.Sprintf("%d", lineNum)))
		result.WriteString(" │ ")
		
		// Add line content
		if highlightSet[lineNum] {
			// Highlight the entire line
			highlightStyle := lipgloss.NewStyle().
				Background(r.theme.Warning())
			result.WriteString(highlightStyle.Render(line))
		} else {
			result.WriteString(line)
		}
		
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}
	
	return result.String()
}

// detectLanguage attempts to detect the language from file extension or content
func (r *FileRenderer) detectLanguage(path, content string) string {
	// Try to detect from file extension
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "go"
	case ".js", ".mjs":
		return "javascript"
	case ".ts", ".tsx":
		return "typescript"
	case ".jsx":
		return "jsx"
	case ".py":
		return "python"
	case ".rb":
		return "ruby"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".c":
		return "c"
	case ".cpp", ".cc", ".cxx":
		return "cpp"
	case ".cs":
		return "csharp"
	case ".php":
		return "php"
	case ".swift":
		return "swift"
	case ".kt":
		return "kotlin"
	case ".scala":
		return "scala"
	case ".sh", ".bash":
		return "bash"
	case ".yml", ".yaml":
		return "yaml"
	case ".json":
		return "json"
	case ".xml":
		return "xml"
	case ".html", ".htm":
		return "html"
	case ".css":
		return "css"
	case ".scss", ".sass":
		return "scss"
	case ".sql":
		return "sql"
	case ".md", ".markdown":
		return "markdown"
	case ".r":
		return "r"
	case ".lua":
		return "lua"
	case ".vim":
		return "vim"
	}
	
	// Try to detect from content patterns
	if strings.HasPrefix(strings.TrimSpace(content), "#!/") {
		firstLine := strings.Split(content, "\n")[0]
		if strings.Contains(firstLine, "python") {
			return "python"
		}
		if strings.Contains(firstLine, "bash") || strings.Contains(firstLine, "sh") {
			return "bash"
		}
		if strings.Contains(firstLine, "ruby") {
			return "ruby"
		}
		if strings.Contains(firstLine, "node") {
			return "javascript"
		}
	}
	
	// Default to text
	return "text"
}

// getFileIcon returns an appropriate icon for the file type
func (r *FileRenderer) getFileIcon(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "🐹"
	case ".js", ".mjs", ".jsx":
		return "📜"
	case ".ts", ".tsx":
		return "📘"
	case ".py":
		return "🐍"
	case ".rb":
		return "💎"
	case ".rs":
		return "🦀"
	case ".java":
		return "☕"
	case ".c", ".cpp", ".cc", ".h":
		return "⚙️"
	case ".cs":
		return "🔷"
	case ".php":
		return "🐘"
	case ".swift":
		return "🦉"
	case ".kt":
		return "🟣"
	case ".sh", ".bash":
		return "🖥️"
	case ".yml", ".yaml":
		return "⚙️"
	case ".json":
		return "📋"
	case ".xml":
		return "📄"
	case ".html", ".htm":
		return "🌐"
	case ".css", ".scss", ".sass":
		return "🎨"
	case ".sql":
		return "🗃️"
	case ".md", ".markdown":
		return "📝"
	case ".txt":
		return "📄"
	case ".pdf":
		return "📕"
	case ".doc", ".docx":
		return "📘"
	case ".xls", ".xlsx":
		return "📊"
	case ".png", ".jpg", ".jpeg", ".gif", ".svg":
		return "🖼️"
	case ".mp4", ".avi", ".mov":
		return "🎬"
	case ".mp3", ".wav", ".flac":
		return "🎵"
	case ".zip", ".tar", ".gz":
		return "📦"
	default:
		return "📄"
	}
}

// SetShowLineNumbers enables or disables line numbers
func (r *FileRenderer) SetShowLineNumbers(show bool) {
	r.showLineNumbers = show
	r.cache.Clear() // Clear cache when setting changes
}

// SetMaxWidth sets the maximum width for rendering
func (r *FileRenderer) SetMaxWidth(width int) {
	r.maxWidth = width
	r.cache.Clear() // Clear cache when width changes
}

// RenderInline renders a compact file reference
func (r *FileRenderer) RenderInline(path string, line int) string {
	icon := r.getFileIcon(path)
	
	style := lipgloss.NewStyle().
		Foreground(r.theme.Primary()).
		Bold(true)
	
	reference := fmt.Sprintf("%s %s", icon, path)
	if line > 0 {
		reference += fmt.Sprintf(":%d", line)
	}
	
	return style.Render(reference)
}