package dialog

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrepeneur4lyf/codeforge/internal/app"
	"github.com/entrepeneur4lyf/codeforge/internal/llm"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/theme"
)

const (
	dialogMaxWidth  = 80
	dialogMaxHeight = 30
)

// ModelSelectedMsg is sent when a model is selected
type ModelSelectedMsg struct {
	Provider string
	Model    string
}


// ModelDialog represents the model selection dialog
type ModelDialog struct {
	app               *app.App
	theme             theme.Theme
	providers         []string
	providerModels    map[string][]llm.ModelResponse
	currentProvider   int
	currentModel      int
	width             int
	height            int
	favorites         map[string]bool
	showOnlyFavorites bool
}

// NewModelDialog creates a new model selection dialog
func NewModelDialog(app *app.App, theme theme.Theme) tea.Model {
	return &ModelDialog{
		app:            app,
		theme:          theme,
		providerModels: make(map[string][]llm.ModelResponse),
		favorites:      make(map[string]bool),
	}
}

func (m *ModelDialog) Init() tea.Cmd {
	// Load models from the app
	m.loadModels()
	return nil
}

func (m *ModelDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Cancel):
			return m, func() tea.Msg { return DialogCloseMsg{} }

		case key.Matches(msg, keys.Select):
			// Select current model
			if provider := m.getCurrentProvider(); provider != "" {
				if model := m.getCurrentModel(); model != nil {
					return m, func() tea.Msg {
						return ModelSelectedMsg{
							Provider: provider,
							Model:    model.Name,
						}
					}
				}
			}

		case key.Matches(msg, keys.NextProvider):
			m.nextProvider()

		case key.Matches(msg, keys.PrevProvider):
			m.prevProvider()

		case key.Matches(msg, keys.NextModel):
			m.nextModel()

		case key.Matches(msg, keys.PrevModel):
			m.prevModel()

		case key.Matches(msg, keys.ToggleFavorite):
			m.toggleFavorite()

		case key.Matches(msg, keys.ShowFavorites):
			m.showOnlyFavorites = !m.showOnlyFavorites
			m.loadModels() // Reload to apply filter
		}
	}

	return m, nil
}

func (m *ModelDialog) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	// Calculate dialog dimensions
	dialogWidth := min(m.width-4, dialogMaxWidth)
	dialogHeight := min(m.height-4, dialogMaxHeight)

	// Build content
	var content strings.Builder

	// Title
	title := "Select Model"
	if m.showOnlyFavorites {
		title += " (Favorites)"
	}
	titleStyle := lipgloss.NewStyle().
		Foreground(m.theme.TextEmphasized()).
		Width(dialogWidth - 4).
		Align(lipgloss.Center)
	content.WriteString(titleStyle.Render(title))
	content.WriteString("\n\n")

	// Provider tabs
	content.WriteString(m.renderProviderTabs(dialogWidth - 4))
	content.WriteString("\n\n")

	// Model list
	content.WriteString(m.renderModelList(dialogWidth-4, dialogHeight-10))
	content.WriteString("\n\n")

	// Help text
	helpText := m.renderHelp()
	helpStyle := lipgloss.NewStyle().
		Foreground(m.theme.TextMuted()).
		Width(dialogWidth - 4).
		Align(lipgloss.Center)
	content.WriteString(helpStyle.Render(helpText))

	// Apply dialog style
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.theme.BorderNormal()).
		Width(dialogWidth).
		Height(dialogHeight).
		MaxWidth(dialogWidth).
		MaxHeight(dialogHeight)

	return dialogStyle.Render(content.String())
}

func (m *ModelDialog) loadModels() {
	// Get all available models from the app
	allModels := m.app.GetAvailableModels()

	// Clear current data
	m.providers = []string{}
	m.providerModels = make(map[string][]llm.ModelResponse)

	// Group by provider
	providerMap := make(map[string][]llm.ModelResponse)
	for _, model := range allModels {
		// Apply favorite filter if enabled
		if m.showOnlyFavorites {
			key := fmt.Sprintf("%s:%s", model.Provider, model.Name)
			if !m.favorites[key] {
				continue
			}
		}
		providerMap[model.Provider] = append(providerMap[model.Provider], model)
	}

	// Sort providers by popularity (you can customize this ordering)
	for provider := range providerMap {
		m.providers = append(m.providers, provider)
	}
	sort.Slice(m.providers, func(i, j int) bool {
		// Custom provider ordering
		order := map[string]int{
			"anthropic":  0,
			"openai":     1,
			"gemini":     2,
			"openrouter": 3,
			"groq":       4,
			"local":      5,
		}

		iOrder, iOk := order[m.providers[i]]
		jOrder, jOk := order[m.providers[j]]

		if iOk && jOk {
			return iOrder < jOrder
		}
		if iOk {
			return true
		}
		if jOk {
			return false
		}
		return m.providers[i] < m.providers[j]
	})

	// Store models
	m.providerModels = providerMap

	// Reset selection
	if m.currentProvider >= len(m.providers) {
		m.currentProvider = 0
	}
	m.currentModel = 0
}

func (m *ModelDialog) renderProviderTabs(width int) string {
	if len(m.providers) == 0 {
		return lipgloss.NewStyle().
			Foreground(m.theme.TextMuted()).
			Render("No providers available")
	}

	var tabs []string
	for i, provider := range m.providers {
		style := lipgloss.NewStyle().
			Padding(0, 2).
			Background(m.theme.BackgroundSecondary()).
			Foreground(m.theme.Text())
		if i == m.currentProvider {
			style = lipgloss.NewStyle().
				Padding(0, 2).
				Background(m.theme.Primary()).
				Foreground(m.theme.Background())
		}

		// Add model count
		count := len(m.providerModels[provider])
		label := fmt.Sprintf("%s (%d)", provider, count)

		tabs = append(tabs, style.Render(label))
	}

	// Add scroll indicators
	prefix := ""
	suffix := ""
	if m.currentProvider > 0 {
		prefix = lipgloss.NewStyle().
			Foreground(m.theme.TextMuted()).
			Render("← ")
	}
	if m.currentProvider < len(m.providers)-1 {
		suffix = lipgloss.NewStyle().
			Foreground(m.theme.TextMuted()).
			Render(" →")
	}

	tabLine := lipgloss.JoinHorizontal(lipgloss.Top, tabs...)

	// Center and add indicators
	centeredWidth := lipgloss.Width(tabLine)
	if centeredWidth < width {
		padding := (width - centeredWidth) / 2
		tabLine = strings.Repeat(" ", padding) + tabLine
	}

	return prefix + tabLine + suffix
}

func (m *ModelDialog) renderModelList(width, height int) string {
	provider := m.getCurrentProvider()
	if provider == "" {
		return lipgloss.NewStyle().
			Foreground(m.theme.TextMuted()).
			Render("No provider selected")
	}

	models := m.providerModels[provider]
	if len(models) == 0 {
		return lipgloss.NewStyle().
			Foreground(m.theme.TextMuted()).
			Render("No models available for this provider")
	}

	// Calculate visible range
	visibleItems := height - 2 // Account for scroll indicators
	startIdx := 0
	if m.currentModel >= visibleItems {
		startIdx = m.currentModel - visibleItems + 1
	}
	endIdx := min(startIdx+visibleItems, len(models))

	var lines []string

	// Top scroll indicator
	if startIdx > 0 {
		lines = append(lines, lipgloss.NewStyle().
			Foreground(m.theme.TextMuted()).
			Render("↑ more models above"))
	}

	// Model items
	for i := startIdx; i < endIdx; i++ {
		model := models[i]
		line := m.renderModelItem(model, i == m.currentModel, width)
		lines = append(lines, line)
	}

	// Bottom scroll indicator
	if endIdx < len(models) {
		lines = append(lines, lipgloss.NewStyle().
			Foreground(m.theme.TextMuted()).
			Render("↓ more models below"))
	}

	return strings.Join(lines, "\n")
}

func (m *ModelDialog) renderModelItem(model llm.ModelResponse, selected bool, width int) string {
	// Build model info
	var parts []string

	// Selection indicator
	if selected {
		parts = append(parts, ">")
	} else {
		parts = append(parts, " ")
	}

	// Favorite indicator
	key := fmt.Sprintf("%s:%s", model.Provider, model.Name)
	if m.favorites[key] {
		parts = append(parts, "★")
	} else {
		parts = append(parts, " ")
	}

	// Model name
	parts = append(parts, model.Name)

	// Context window
	if model.Info.ContextWindow > 0 {
		ctx := formatContextWindow(model.Info.ContextWindow)
		parts = append(parts, fmt.Sprintf("[%s]", ctx))
	}

	// Pricing
	if model.Info.InputPrice > 0 || model.Info.OutputPrice > 0 {
		price := fmt.Sprintf("$%.2f/$%.2f", model.Info.InputPrice, model.Info.OutputPrice)
		parts = append(parts, price)
	}

	// Join parts
	line := strings.Join(parts, " ")

	// Apply style
	style := lipgloss.NewStyle().
		Background(m.theme.Background())
	if selected {
		style = lipgloss.NewStyle().
			Background(m.theme.Primary()).
			Foreground(m.theme.Background())
	}

	// Truncate if needed
	maxWidth := width - 4
	if lipgloss.Width(line) > maxWidth {
		line = truncate(line, maxWidth-3) + "..."
	}

	return style.Width(width).Render(line)
}

func (m *ModelDialog) renderHelp() string {
	helps := []string{
		"←/→: switch provider",
		"↑/↓: select model",
		"enter: confirm",
		"space: favorite",
		"f: show favorites",
		"esc: cancel",
	}
	return strings.Join(helps, " • ")
}

// Navigation methods
func (m *ModelDialog) getCurrentProvider() string {
	if m.currentProvider >= 0 && m.currentProvider < len(m.providers) {
		return m.providers[m.currentProvider]
	}
	return ""
}

func (m *ModelDialog) getCurrentModel() *llm.ModelResponse {
	provider := m.getCurrentProvider()
	if provider == "" {
		return nil
	}

	models := m.providerModels[provider]
	if m.currentModel >= 0 && m.currentModel < len(models) {
		return &models[m.currentModel]
	}
	return nil
}

func (m *ModelDialog) nextProvider() {
	if m.currentProvider < len(m.providers)-1 {
		m.currentProvider++
		m.currentModel = 0
	}
}

func (m *ModelDialog) prevProvider() {
	if m.currentProvider > 0 {
		m.currentProvider--
		m.currentModel = 0
	}
}

func (m *ModelDialog) nextModel() {
	provider := m.getCurrentProvider()
	if provider != "" {
		models := m.providerModels[provider]
		if m.currentModel < len(models)-1 {
			m.currentModel++
		}
	}
}

func (m *ModelDialog) prevModel() {
	if m.currentModel > 0 {
		m.currentModel--
	}
}

func (m *ModelDialog) toggleFavorite() {
	if model := m.getCurrentModel(); model != nil {
		key := fmt.Sprintf("%s:%s", model.Provider, model.Name)
		m.favorites[key] = !m.favorites[key]

		// TODO: Persist favorites to app configuration
		// m.app.SetModelFavorite(model.Provider, model.Name, m.favorites[key])
	}
}


// Key bindings
type keyMap struct {
	Cancel         key.Binding
	Select         key.Binding
	NextProvider   key.Binding
	PrevProvider   key.Binding
	NextModel      key.Binding
	PrevModel      key.Binding
	ToggleFavorite key.Binding
	ShowFavorites  key.Binding
}

var keys = keyMap{
	Cancel: key.NewBinding(
		key.WithKeys("esc", "ctrl+c"),
		key.WithHelp("esc", "cancel"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	NextProvider: key.NewBinding(
		key.WithKeys("right", "l", "tab"),
		key.WithHelp("→/tab", "next provider"),
	),
	PrevProvider: key.NewBinding(
		key.WithKeys("left", "h", "shift+tab"),
		key.WithHelp("←/shift+tab", "prev provider"),
	),
	NextModel: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "next model"),
	),
	PrevModel: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "prev model"),
	),
	ToggleFavorite: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "toggle favorite"),
	),
	ShowFavorites: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "show favorites"),
	),
}
