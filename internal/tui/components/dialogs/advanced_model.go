package dialog

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrepeneur4lyf/codeforge/internal/app"
	chatfav "github.com/entrepeneur4lyf/codeforge/internal/chat"
	"github.com/entrepeneur4lyf/codeforge/internal/models"
	"github.com/entrepeneur4lyf/codeforge/internal/tui/theme"
)

// AdvancedModelDialog provides intelligent model selection with recommendations
type AdvancedModelDialog struct {
	app              *app.App
	theme            theme.Theme
	models           []*models.CanonicalModel
	filteredModels   []*models.CanonicalModel
	currentModel     int
	width            int
	height           int
	viewMode         ViewMode
	filterCriteria   models.ModelSelectionCriteria
	recommendations  []models.ModelRecommendation
	showingFavorites bool
	favoritesMgr     *chatfav.Favorites
}

// ViewMode defines different viewing modes
type ViewMode int

const (
	ViewAll ViewMode = iota
	ViewRecommended
	ViewFavorites
	ViewByProvider
	ViewByCapability
)

var viewModeNames = map[ViewMode]string{
	ViewAll:          "All Models",
	ViewRecommended:  "Recommended",
	ViewFavorites:    "Favorites",
	ViewByProvider:   "By Provider",
	ViewByCapability: "By Capability",
}

// (unified with ModelSelectedMsg in shared.go)

// NewAdvancedModelDialog creates a new advanced model selection dialog
func NewAdvancedModelDialog(app *app.App, theme theme.Theme) *AdvancedModelDialog {
	return &AdvancedModelDialog{
		app:   app,
		theme: theme,
		filterCriteria: models.ModelSelectionCriteria{
			TaskType:         "code",
			RequiredFeatures: []string{"code", "tools"},
			MaxCost:          50.0, // $50 per million tokens
			PreferredSpeed:   "balanced",
		},
	}
}

func (m *AdvancedModelDialog) Init() tea.Cmd {
	// Load models and get recommendations
	// Initialize favorites manager for persistence
	if fav, err := chatfav.NewFavorites(); err == nil {
		m.favoritesMgr = fav
		// Hydrate app favorites from persisted ids if possible
		if m.app != nil {
			for _, id := range fav.GetFavoriteModels() {
				// Try to map string to canonical ID
				m.app.AddFavoriteModel(models.CanonicalModelID(id))
			}
		}
	}
	m.loadModels()
	return m.getRecommendations()
}

func (m *AdvancedModelDialog) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, advancedKeys.Cancel):
			return m, func() tea.Msg { return DialogCloseMsg{} }

		case key.Matches(msg, advancedKeys.Select):
			if model := m.getCurrentModel(); model != nil {
				provider := m.getBestProvider(model)
				selProvider := string(provider)
				selModel := string(model.ID)
				return m, func() tea.Msg {
					return ModelSelectedMsg{Provider: selProvider, Model: selModel}
				}
			}

		case key.Matches(msg, advancedKeys.NextModel):
			m.nextModel()

		case key.Matches(msg, advancedKeys.PrevModel):
			m.prevModel()

		case key.Matches(msg, advancedKeys.ToggleFavorite):
			return m, m.toggleFavorite()

		case key.Matches(msg, advancedKeys.SwitchView):
			m.switchViewMode()
			m.applyCurrentView()

		case key.Matches(msg, advancedKeys.ShowFavorites):
			m.viewMode = ViewFavorites
			m.applyCurrentView()

		case key.Matches(msg, advancedKeys.ShowRecommended):
			m.viewMode = ViewRecommended
			m.applyCurrentView()
			return m, m.getRecommendations()

		}
	}

	return m, nil
}

func (m *AdvancedModelDialog) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	// Calculate dialog dimensions
	dialogWidth := min(m.width-4, 100)
	dialogHeight := min(m.height-4, 40)

	var content strings.Builder

	// Title with view mode
	viewName := viewModeNames[m.viewMode]
	title := fmt.Sprintf("Select Model - %s", viewName)
	titleStyle := lipgloss.NewStyle().
		Foreground(m.theme.TextEmphasized()).
		Width(dialogWidth - 4).
		Align(lipgloss.Center).
		Bold(true)
	content.WriteString(titleStyle.Render(title))
	content.WriteString("\n\n")

	// Criteria summary (if in recommended mode)
	if m.viewMode == ViewRecommended {
		criteriaText := m.renderCriteria()
		criteriaStyle := lipgloss.NewStyle().
			Foreground(m.theme.TextMuted()).
			Width(dialogWidth - 4).
			Align(lipgloss.Center)
		content.WriteString(criteriaStyle.Render(criteriaText))
		content.WriteString("\n\n")
	}

	// Model list
	content.WriteString(m.renderModelList(dialogWidth-4, dialogHeight-12))
	content.WriteString("\n\n")

	// Model details
	if model := m.getCurrentModel(); model != nil {
		content.WriteString(m.renderModelDetails(model, dialogWidth-4))
		content.WriteString("\n\n")
	}

	// Help text
	helpText := m.renderAdvancedHelp()
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
		Padding(1)

	return dialogStyle.Render(content.String())
}

func (m *AdvancedModelDialog) loadModels() {
	m.models = m.app.GetCanonicalModels()
	m.applyCurrentView()
}

func (m *AdvancedModelDialog) applyCurrentView() {
	switch m.viewMode {
	case ViewAll:
		m.filteredModels = m.models

	case ViewRecommended:
		// Show recommended models based on current recommendations
		var recommended []*models.CanonicalModel
		for _, rec := range m.recommendations {
			recommended = append(recommended, rec.Model)
		}
		m.filteredModels = recommended

	case ViewFavorites:
		m.filteredModels = m.app.GetFavoriteModels()

	case ViewByProvider:
		// Sort by provider
		m.filteredModels = make([]*models.CanonicalModel, len(m.models))
		copy(m.filteredModels, m.models)
		sort.Slice(m.filteredModels, func(i, j int) bool {
			// Get primary provider for each model
			providerI := m.getPrimaryProvider(m.filteredModels[i])
			providerJ := m.getPrimaryProvider(m.filteredModels[j])
			return providerI < providerJ
		})

	case ViewByCapability:
		// Sort by capabilities (reasoning models first, then vision, etc.)
		m.filteredModels = make([]*models.CanonicalModel, len(m.models))
		copy(m.filteredModels, m.models)
		sort.Slice(m.filteredModels, func(i, j int) bool {
			scoreI := m.getCapabilityScore(m.filteredModels[i])
			scoreJ := m.getCapabilityScore(m.filteredModels[j])
			return scoreI > scoreJ
		})
	}

	// Reset selection
	if m.currentModel >= len(m.filteredModels) {
		m.currentModel = 0
	}
}

func (m *AdvancedModelDialog) renderModelList(width, height int) string {
	if len(m.filteredModels) == 0 {
		return lipgloss.NewStyle().
			Foreground(m.theme.TextMuted()).
			Render("No models available")
	}

	// Calculate visible range
	visibleItems := height
	startIdx := 0
	if m.currentModel >= visibleItems {
		startIdx = m.currentModel - visibleItems + 1
	}
	endIdx := min(startIdx+visibleItems, len(m.filteredModels))

	var lines []string

	// Model items
	for i := startIdx; i < endIdx; i++ {
		model := m.filteredModels[i]
		line := m.renderAdvancedModelItem(model, i == m.currentModel, width)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func (m *AdvancedModelDialog) renderAdvancedModelItem(model *models.CanonicalModel, selected bool, width int) string {
	var parts []string

	// Selection indicator
	if selected {
		parts = append(parts, "▶")
	} else {
		parts = append(parts, " ")
	}

	// Favorite indicator
	favorites := m.app.GetFavoriteModels()
	isFavorite := false
	for _, fav := range favorites {
		if fav.ID == model.ID {
			isFavorite = true
			break
		}
	}
	if isFavorite {
		parts = append(parts, "★")
	} else {
		parts = append(parts, " ")
	}

	// Model name with family
	name := fmt.Sprintf("%s (%s)", model.Name, model.Family)
	parts = append(parts, name)

	// Capabilities badges
	var badges []string
	if model.Capabilities.SupportsReasoning {
		badges = append(badges, "🧠")
	}
	if model.Capabilities.SupportsVision {
		badges = append(badges, "👁")
	}
	if model.Capabilities.SupportsTools {
		badges = append(badges, "🔧")
	}
	if len(badges) > 0 {
		parts = append(parts, strings.Join(badges, ""))
	}

	// Context size
	ctx := formatContextWindow(model.Limits.ContextWindow)
	parts = append(parts, fmt.Sprintf("[%s]", ctx))

	// Pricing tier
	tier := getPricingTier(model.Pricing.OutputPrice)
	parts = append(parts, tier)

	// Recommendation score (if in recommended view)
	if m.viewMode == ViewRecommended {
		for _, rec := range m.recommendations {
			if rec.Model.ID == model.ID {
				parts = append(parts, fmt.Sprintf("%.0f%%", rec.Score))
				break
			}
		}
	}

	// Join parts
	line := strings.Join(parts, " ")

	// Apply style
	style := lipgloss.NewStyle().
		Background(m.theme.Background()).
		Padding(0, 1)
	if selected {
		style = lipgloss.NewStyle().
			Background(m.theme.Primary()).
			Foreground(m.theme.Background()).
			Padding(0, 1)
	}

	// Truncate if needed
	maxWidth := width - 2
	if lipgloss.Width(line) > maxWidth {
		line = truncate(line, maxWidth-3) + "..."
	}

	return style.Width(width).Render(line)
}

func (m *AdvancedModelDialog) renderModelDetails(model *models.CanonicalModel, width int) string {
	var details []string

	// Basic info
	provider := m.getPrimaryProvider(model)
	details = append(details, fmt.Sprintf("Provider: %s", provider))
	details = append(details, fmt.Sprintf("Context: %s tokens", formatNumber(model.Limits.ContextWindow)))
	details = append(details, fmt.Sprintf("Max Output: %s tokens", formatNumber(model.Limits.MaxOutputTokens)))

	// Pricing
	details = append(details, fmt.Sprintf("Input: $%.2f/M", model.Pricing.InputPrice))
	details = append(details, fmt.Sprintf("Output: $%.2f/M", model.Pricing.OutputPrice))

	// Capabilities
	var caps []string
	if model.Capabilities.SupportsReasoning {
		caps = append(caps, "Reasoning")
	}
	if model.Capabilities.SupportsVision {
		caps = append(caps, "Vision")
	}
	if model.Capabilities.SupportsTools {
		caps = append(caps, "Tools")
	}
	if model.Capabilities.SupportsStreaming {
		caps = append(caps, "Streaming")
	}
	if len(caps) > 0 {
		details = append(details, fmt.Sprintf("Capabilities: %s", strings.Join(caps, ", ")))
	}

	// Show recommendation reasoning if in recommended view
	if m.viewMode == ViewRecommended {
		for _, rec := range m.recommendations {
			if rec.Model.ID == model.ID {
				if len(rec.Reasoning) > 0 {
					details = append(details, fmt.Sprintf("Why: %s", strings.Join(rec.Reasoning, ", ")))
				}
				break
			}
		}
	}

	// Style details
	detailsStyle := lipgloss.NewStyle().
		Foreground(m.theme.TextMuted()).
		Width(width).
		Border(lipgloss.NormalBorder()).
		BorderForeground(m.theme.BorderNormal()).
		Padding(0, 1)

	return detailsStyle.Render(strings.Join(details, "\n"))
}

func (m *AdvancedModelDialog) renderCriteria() string {
	var parts []string
	parts = append(parts, fmt.Sprintf("Task: %s", m.filterCriteria.TaskType))
	if len(m.filterCriteria.RequiredFeatures) > 0 {
		parts = append(parts, fmt.Sprintf("Features: %s", strings.Join(m.filterCriteria.RequiredFeatures, ", ")))
	}
	if m.filterCriteria.MaxCost > 0 {
		parts = append(parts, fmt.Sprintf("Max Cost: $%.0f/M", m.filterCriteria.MaxCost))
	}
	return strings.Join(parts, " • ")
}

func (m *AdvancedModelDialog) renderAdvancedHelp() string {
	helps := []string{
		"↑/↓: navigate",
		"enter: select",
		"space: favorite",
		"v: view mode",
		"r: recommended",
		"f: favorites",
		"esc: cancel",
	}
	return strings.Join(helps, " • ")
}

func (m *AdvancedModelDialog) getRecommendations() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		recommendations, err := m.app.GetModelRecommendation(ctx, m.filterCriteria)
		if err != nil {
			return nil
		}

		// Store recommendations
		if recommendations != nil {
			m.recommendations = []models.ModelRecommendation{*recommendations}
			// Add fallbacks if available
			m.recommendations = append(m.recommendations, recommendations.Fallbacks...)
		}

		return nil
	}
}

func (m *AdvancedModelDialog) toggleFavorite() tea.Cmd {
	return func() tea.Msg {
		if model := m.getCurrentModel(); model != nil {
			// Check if it's already a favorite
			favorites := m.app.GetFavoriteModels()
			isFavorite := false
			for _, fav := range favorites {
				if fav.ID == model.ID {
					isFavorite = true
					break
				}
			}

			if isFavorite {
				m.app.RemoveFavoriteModel(model.ID)
				if m.favoritesMgr != nil {
					_ = m.favoritesMgr.RemoveModelFavorite(string(model.ID))
				}
			} else {
				m.app.AddFavoriteModel(model.ID)
				if m.favoritesMgr != nil {
					_ = m.favoritesMgr.AddModelFavorite(string(model.ID))
				}
			}
		}
		return nil
	}
}

func (m *AdvancedModelDialog) showTaskSelector() tea.Cmd {
	// Cycle through common task types for simplicity
	cycle := []string{"code", "chat", "analysis", "creative", "reasoning", "vision"}
	current := m.filterCriteria.TaskType
	idx := 0
	for i, v := range cycle {
		if v == current {
			idx = i
			break
		}
	}
	next := cycle[(idx+1)%len(cycle)]
	m.filterCriteria.TaskType = next
	m.applyCurrentView()
	return m.getRecommendations()
}

// Helper methods
func (m *AdvancedModelDialog) getCurrentModel() *models.CanonicalModel {
	if m.currentModel >= 0 && m.currentModel < len(m.filteredModels) {
		return m.filteredModels[m.currentModel]
	}
	return nil
}

func (m *AdvancedModelDialog) nextModel() {
	if m.currentModel < len(m.filteredModels)-1 {
		m.currentModel++
	}
}

func (m *AdvancedModelDialog) prevModel() {
	if m.currentModel > 0 {
		m.currentModel--
	}
}

func (m *AdvancedModelDialog) switchViewMode() {
	m.viewMode = (m.viewMode + 1) % 5 // Cycle through view modes
}

func (m *AdvancedModelDialog) getBestProvider(model *models.CanonicalModel) models.ProviderID {
	// Get user preferences
	prefs := m.app.GetModelPreferences()

	// Check user's preferred providers first
	for _, prefProvider := range prefs.PreferredProviders {
		if _, exists := model.Providers[prefProvider]; exists {
			return prefProvider
		}
	}

	// Fallback to first available provider
	for providerID := range model.Providers {
		return providerID
	}

	return ""
}

func (m *AdvancedModelDialog) getPrimaryProvider(model *models.CanonicalModel) string {
	// Return the first provider alphabetically for consistent sorting
	var providers []string
	for providerID := range model.Providers {
		providers = append(providers, string(providerID))
	}
	if len(providers) > 0 {
		sort.Strings(providers)
		return providers[0]
	}
	return ""
}

func (m *AdvancedModelDialog) getCapabilityScore(model *models.CanonicalModel) int {
	score := 0
	if model.Capabilities.SupportsReasoning {
		score += 4
	}
	if model.Capabilities.SupportsVision {
		score += 3
	}
	if model.Capabilities.SupportsTools {
		score += 2
	}
	if model.Capabilities.SupportsStreaming {
		score += 1
	}
	return score
}

// Helper functions
func getPricingTier(outputPrice float64) string {
	if outputPrice <= 1.0 {
		return "💚 Low"
	} else if outputPrice <= 10.0 {
		return "🟡 Med"
	} else {
		return "🔴 High"
	}
}

func formatNumber(n int) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%dk", n/1000)
	}
	return fmt.Sprintf("%d", n)
}

// Advanced key bindings
type advancedKeyMap struct {
	Cancel          key.Binding
	Select          key.Binding
	NextModel       key.Binding
	PrevModel       key.Binding
	ToggleFavorite  key.Binding
	SwitchView      key.Binding
	ShowFavorites   key.Binding
	ShowRecommended key.Binding
	FilterByTask    key.Binding
}

var advancedKeys = advancedKeyMap{
	Cancel: key.NewBinding(
		key.WithKeys("esc", "ctrl+c"),
		key.WithHelp("esc", "cancel"),
	),
	Select: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
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
	SwitchView: key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "view mode"),
	),
	ShowFavorites: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "favorites"),
	),
	ShowRecommended: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "recommended"),
	),
	FilterByTask: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "filter by task"),
	),
}
