package dialogs

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/themes"
)

// FileSelectedMsg is sent when files are selected
type FileSelectedMsg struct {
	Paths []string
}

// FileDialog is a file picker dialog
type FileDialog struct {
	theme         themes.Theme
	width         int
	height        int
	currentPath   string
	entries       []fs.DirEntry
	selectedIndex int
	selectedFiles map[string]bool
	multiSelect   bool
	showHidden    bool
}

// NewFileDialog creates a new file picker dialog
func NewFileDialog(theme themes.Theme) tea.Model {
	cwd, _ := os.Getwd()
	return &FileDialog{
		theme:         theme,
		currentPath:   cwd,
		selectedFiles: make(map[string]bool),
		multiSelect:   true,
	}
}

func (f *FileDialog) Init() tea.Cmd {
	return f.loadDirectory()
}

func (f *FileDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		f.width = msg.Width
		f.height = msg.Height
		
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, fileKeys.Cancel):
			return f, func() tea.Msg { return DialogCloseMsg{} }
			
		case key.Matches(msg, fileKeys.Select):
			return f, f.handleSelection()
			
		case key.Matches(msg, fileKeys.Up):
			f.moveUp()
			
		case key.Matches(msg, fileKeys.Down):
			f.moveDown()
			
		case key.Matches(msg, fileKeys.Enter):
			return f, f.handleEnter()
			
		case key.Matches(msg, fileKeys.Back):
			return f, f.navigateUp()
			
		case key.Matches(msg, fileKeys.ToggleFile):
			f.toggleCurrentFile()
			
		case key.Matches(msg, fileKeys.ToggleHidden):
			f.showHidden = !f.showHidden
			return f, f.loadDirectory()
			
		case key.Matches(msg, fileKeys.Home):
			return f, f.navigateHome()
		}
		
	case directoryLoadedMsg:
		f.entries = msg.entries
		f.selectedIndex = 0
	}
	
	return f, nil
}

func (f *FileDialog) View() string {
	if f.width == 0 || f.height == 0 {
		return ""
	}
	
	// Calculate dialog dimensions
	dialogWidth := min(f.width-4, 70)
	dialogHeight := min(f.height-4, 30)
	
	// Build content
	var content strings.Builder
	
	// Title
	title := "Select Files"
	if len(f.selectedFiles) > 0 {
		title = fmt.Sprintf("Select Files (%d selected)", len(f.selectedFiles))
	}
	titleStyle := f.theme.DialogTitleStyle().Width(dialogWidth - 4).Align(lipgloss.Center)
	content.WriteString(titleStyle.Render(title))
	content.WriteString("\n")
	
	// Current path
	pathStyle := f.theme.MutedText().Width(dialogWidth - 4)
	content.WriteString(pathStyle.Render(truncatePath(f.currentPath, dialogWidth-4)))
	content.WriteString("\n\n")
	
	// File list
	listHeight := dialogHeight - 8
	content.WriteString(f.renderFileList(dialogWidth-4, listHeight))
	content.WriteString("\n\n")
	
	// Help text
	helpText := f.renderHelp()
	helpStyle := f.theme.MutedText().Width(dialogWidth - 4).Align(lipgloss.Center)
	content.WriteString(helpStyle.Render(helpText))
	
	// Apply dialog style
	dialogStyle := f.theme.DialogStyle().
		Width(dialogWidth).
		Height(dialogHeight).
		MaxWidth(dialogWidth).
		MaxHeight(dialogHeight)
		
	return dialogStyle.Render(content.String())
}

func (f *FileDialog) renderFileList(width, height int) string {
	if len(f.entries) == 0 {
		return f.theme.MutedText().Render("Empty directory")
	}
	
	// Calculate visible range
	visibleItems := height
	startIdx := 0
	if f.selectedIndex >= visibleItems {
		startIdx = f.selectedIndex - visibleItems + 1
	}
	endIdx := min(startIdx+visibleItems, len(f.entries))
	
	var lines []string
	
	// Render entries
	for i := startIdx; i < endIdx; i++ {
		entry := f.entries[i]
		line := f.renderEntry(entry, i == f.selectedIndex, width)
		lines = append(lines, line)
	}
	
	return strings.Join(lines, "\n")
}

func (f *FileDialog) renderEntry(entry fs.DirEntry, selected bool, width int) string {
	name := entry.Name()
	fullPath := filepath.Join(f.currentPath, name)
	
	var parts []string
	
	// Selection indicator
	if selected {
		parts = append(parts, ">")
	} else {
		parts = append(parts, " ")
	}
	
	// Checkbox for files
	if !entry.IsDir() {
		if f.selectedFiles[fullPath] {
			parts = append(parts, "[✓]")
		} else {
			parts = append(parts, "[ ]")
		}
	} else {
		parts = append(parts, "   ")
	}
	
	// Icon
	if entry.IsDir() {
		parts = append(parts, "📁")
	} else {
		parts = append(parts, getFileIcon(name))
	}
	
	// Name
	if entry.IsDir() {
		name += "/"
	}
	parts = append(parts, name)
	
	// File info
	if info, err := entry.Info(); err == nil && !entry.IsDir() {
		size := formatFileSize(info.Size())
		parts = append(parts, fmt.Sprintf("(%s)", size))
	}
	
	// Join parts
	line := strings.Join(parts, " ")
	
	// Apply style
	style := f.theme.ListItem()
	if selected {
		style = f.theme.ListItemActive()
	}
	if f.selectedFiles[fullPath] {
		style = style.Foreground(f.theme.Success())
	}
	
	// Truncate if needed
	if lipgloss.Width(line) > width {
		line = truncate(line, width-3) + "..."
	}
	
	return style.Width(width).Render(line)
}

func (f *FileDialog) renderHelp() string {
	helps := []string{
		"↑/↓: navigate",
		"space: select",
		"enter: open/confirm",
		"backspace: up",
		"h: hidden",
		"esc: cancel",
	}
	return strings.Join(helps, " • ")
}

// Directory navigation
func (f *FileDialog) loadDirectory() tea.Cmd {
	return func() tea.Msg {
		entries, err := os.ReadDir(f.currentPath)
		if err != nil {
			return err
		}
		
		// Filter hidden files if needed
		if !f.showHidden {
			var filtered []fs.DirEntry
			for _, entry := range entries {
				if !strings.HasPrefix(entry.Name(), ".") {
					filtered = append(filtered, entry)
				}
			}
			entries = filtered
		}
		
		// Sort: directories first, then alphabetically
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].IsDir() != entries[j].IsDir() {
				return entries[i].IsDir()
			}
			return entries[i].Name() < entries[j].Name()
		})
		
		return directoryLoadedMsg{entries: entries}
	}
}

func (f *FileDialog) handleEnter() tea.Cmd {
	if f.selectedIndex >= 0 && f.selectedIndex < len(f.entries) {
		entry := f.entries[f.selectedIndex]
		if entry.IsDir() {
			f.currentPath = filepath.Join(f.currentPath, entry.Name())
			return f.loadDirectory()
		} else {
			// Toggle file selection
			f.toggleCurrentFile()
		}
	}
	return nil
}

func (f *FileDialog) handleSelection() tea.Cmd {
	if len(f.selectedFiles) > 0 {
		var paths []string
		for path := range f.selectedFiles {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		
		return func() tea.Msg {
			return FileSelectedMsg{Paths: paths}
		}
	}
	return nil
}

func (f *FileDialog) toggleCurrentFile() {
	if f.selectedIndex >= 0 && f.selectedIndex < len(f.entries) {
		entry := f.entries[f.selectedIndex]
		if !entry.IsDir() {
			fullPath := filepath.Join(f.currentPath, entry.Name())
			if f.selectedFiles[fullPath] {
				delete(f.selectedFiles, fullPath)
			} else {
				f.selectedFiles[fullPath] = true
			}
		}
	}
}

func (f *FileDialog) navigateUp() tea.Cmd {
	parent := filepath.Dir(f.currentPath)
	if parent != f.currentPath {
		f.currentPath = parent
		return f.loadDirectory()
	}
	return nil
}

func (f *FileDialog) navigateHome() tea.Cmd {
	home, err := os.UserHomeDir()
	if err == nil {
		f.currentPath = home
		return f.loadDirectory()
	}
	return nil
}

func (f *FileDialog) moveUp() {
	if f.selectedIndex > 0 {
		f.selectedIndex--
	}
}

func (f *FileDialog) moveDown() {
	if f.selectedIndex < len(f.entries)-1 {
		f.selectedIndex++
	}
}

// Helper functions
func getFileIcon(name string) string {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".go":
		return "🐹"
	case ".rs":
		return "🦀"
	case ".py":
		return "🐍"
	case ".js", ".ts", ".jsx", ".tsx":
		return "📜"
	case ".json", ".yaml", ".yml", ".toml":
		return "⚙️"
	case ".md", ".txt", ".doc":
		return "📄"
	case ".png", ".jpg", ".jpeg", ".gif", ".svg":
		return "🖼️"
	case ".zip", ".tar", ".gz":
		return "📦"
	default:
		return "📄"
	}
}

func formatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func truncatePath(path string, maxWidth int) string {
	if lipgloss.Width(path) <= maxWidth {
		return path
	}
	
	// Try to show the end of the path
	parts := strings.Split(path, string(os.PathSeparator))
	for i := 0; i < len(parts)-1; i++ {
		truncated := ".../" + strings.Join(parts[i+1:], "/")
		if lipgloss.Width(truncated) <= maxWidth {
			return truncated
		}
	}
	
	return "..." + path[len(path)-maxWidth+3:]
}

// Messages
type directoryLoadedMsg struct {
	entries []fs.DirEntry
}

// Key bindings
type fileKeyMap struct {
	Cancel       key.Binding
	Select       key.Binding
	Up           key.Binding
	Down         key.Binding
	Enter        key.Binding
	Back         key.Binding
	ToggleFile   key.Binding
	ToggleHidden key.Binding
	Home         key.Binding
}

var fileKeys = fileKeyMap{
	Cancel: key.NewBinding(
		key.WithKeys("esc", "ctrl+c"),
		key.WithHelp("esc", "cancel"),
	),
	Select: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "confirm selection"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "open/select"),
	),
	Back: key.NewBinding(
		key.WithKeys("backspace", "left", "h"),
		key.WithHelp("backspace", "parent directory"),
	),
	ToggleFile: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "toggle selection"),
	),
	ToggleHidden: key.NewBinding(
		key.WithKeys("H"),
		key.WithHelp("H", "toggle hidden files"),
	),
	Home: key.NewBinding(
		key.WithKeys("~"),
		key.WithHelp("~", "go home"),
	),
}