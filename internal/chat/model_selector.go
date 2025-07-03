package chat

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrepeneur4lyf/codeforge/internal/llm/providers"
	"github.com/entrepeneur4lyf/codeforge/internal/vectordb"
)

// ModelSelector handles interactive model selection
type ModelSelector struct {
	providers         []ProviderInfo
	models            []ModelInfo
	openRouterFilters []OpenRouterFilter
	selectedIndex     int
	mode              SelectionMode
	selectedProvider  string
	selectedFilter    string
	favorites         *Favorites
	result            chan SelectionResult
	loading           bool
	loadingMessage    string
}

type SelectionMode int

const (
	SelectingProvider SelectionMode = iota
	SelectingOpenRouterFilter
	SelectingModel
)

// Message types for async model loading
type modelsLoadedMsg []ModelInfo
type loadingMsg string

type ProviderInfo struct {
	Name      string
	ID        string
	Available bool
	Favorite  bool
}

// OpenRouterFilter represents a provider filter for OpenRouter
type OpenRouterFilter struct {
	Name        string
	ProviderKey string
	Description string
}

type ModelInfo struct {
	Name          string
	ID            string
	Provider      string
	Favorite      bool
	Description   string
	ContextLength int
	InputPrice    float64
	OutputPrice   float64
	Capabilities  []string
}

type SelectionResult struct {
	Provider string
	Model    string
	Canceled bool
}

// Styles for the TUI
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("57")).
			Foreground(lipgloss.Color("230")).
			Bold(true)

	favoriteStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("226")).
			Bold(true)

	availableStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46"))

	unavailableStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				Strikethrough(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)
)

// NewModelSelector creates a new model selector
func NewModelSelector(favorites *Favorites) *ModelSelector {
	ms := &ModelSelector{
		favorites: favorites,
		mode:      SelectingProvider,
		result:    make(chan SelectionResult, 1),
	}
	ms.loadOpenRouterFilters()
	return ms
}

// SelectModel shows the interactive model selector and returns the selected provider/model
func (ms *ModelSelector) SelectModel() (string, string, error) {
	// Load providers
	ms.loadProviders()

	// Start the TUI
	p := tea.NewProgram(ms, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return "", "", fmt.Errorf("failed to run model selector: %w", err)
	}

	// Get result
	result := <-ms.result
	if result.Canceled {
		return "", "", fmt.Errorf("selection canceled")
	}

	return result.Provider, result.Model, nil
}

// loadProviders loads available providers and checks their availability
func (ms *ModelSelector) loadProviders() {
	providerNames := []string{"anthropic", "openai", "gemini", "groq", "github", "openrouter"}

	for _, name := range providerNames {
		available := ms.isProviderAvailable(name)
		favorite := ms.favorites.IsProviderFavorite(name)

		ms.providers = append(ms.providers, ProviderInfo{
			Name:      titleCase(name),
			ID:        name,
			Available: available,
			Favorite:  favorite,
		})
	}

	// Sort providers: favorites first, then available, then unavailable
	sort.Slice(ms.providers, func(i, j int) bool {
		if ms.providers[i].Favorite != ms.providers[j].Favorite {
			return ms.providers[i].Favorite
		}
		if ms.providers[i].Available != ms.providers[j].Available {
			return ms.providers[i].Available
		}
		return ms.providers[i].Name < ms.providers[j].Name
	})
}

// loadOpenRouterFilters loads the available provider filters for OpenRouter from database
func (ms *ModelSelector) loadOpenRouterFilters() {
	// Load filters dynamically from database
	filters, err := ms.loadOpenRouterFiltersFromDB()
	if err != nil {
		// Fallback to basic filters if database fails
		ms.openRouterFilters = []OpenRouterFilter{
			{Name: "All Providers", ProviderKey: "", Description: "Show models from all providers"},
		}
		return
	}

	ms.openRouterFilters = filters
}

// loadOpenRouterFiltersFromDB loads provider filters from database with actual provider counts
func (ms *ModelSelector) loadOpenRouterFiltersFromDB() ([]OpenRouterFilter, error) {
	// Get database instance
	vdb := vectordb.GetInstance()
	if vdb == nil {
		return nil, fmt.Errorf("vector database not initialized")
	}

	// Query distinct providers with counts from database
	query := `
		SELECT provider, COUNT(*) as count
		FROM openrouter_models
		WHERE last_seen > datetime('now', '-24 hours')
		AND provider IS NOT NULL
		AND provider != ''
		GROUP BY provider
		ORDER BY count DESC
	`

	ctx := context.Background()
	rows, err := vdb.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query providers: %w", err)
	}
	defer rows.Close()

	// Start with "All Providers" filter
	filters := []OpenRouterFilter{
		{Name: "All Providers", ProviderKey: "", Description: "Show models from all providers"},
	}

	// Add provider-specific filters
	for rows.Next() {
		var provider string
		var count int
		if err := rows.Scan(&provider, &count); err != nil {
			continue
		}

		// Transform provider name to display name
		displayName := ms.getProviderDisplayName(strings.ToLower(provider))
		description := fmt.Sprintf("%s models via OpenRouter (%d models)", displayName, count)

		filters = append(filters, OpenRouterFilter{
			Name:        displayName,
			ProviderKey: strings.ToLower(provider),
			Description: description,
		})
	}

	return filters, nil
}

// getProviderDisplayName converts database provider name to display name
func (ms *ModelSelector) getProviderDisplayName(provider string) string {
	displayNames := map[string]string{
		"anthropic":  "Anthropic",
		"openai":     "OpenAI",
		"google":     "Google",
		"meta":       "Meta/Llama",
		"mistral ai": "Mistral AI",
		"deepseek":   "DeepSeek",
		"x ai":       "xAI",
		"qwen":       "Qwen",
		"cohere":     "Cohere",
		"nvidia":     "NVIDIA",
		"microsoft":  "Microsoft",
	}

	if display, exists := displayNames[provider]; exists {
		return display
	}

	// Capitalize first letter for unknown providers
	if len(provider) > 0 {
		return strings.ToUpper(provider[:1]) + provider[1:]
	}
	return provider
}


// titleCase converts a string to title case without using deprecated strings.Title
func titleCase(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	runes[0] = unicode.ToUpper(runes[0])
	for i := 1; i < len(runes); i++ {
		if runes[i-1] == ' ' || runes[i-1] == '-' || runes[i-1] == '_' {
			runes[i] = unicode.ToUpper(runes[i])
		}
	}
	return string(runes)
}

// isProviderAvailable checks if a provider has an API key
func (ms *ModelSelector) isProviderAvailable(provider string) bool {
	switch provider {
	case "anthropic":
		return os.Getenv("ANTHROPIC_API_KEY") != ""
	case "openai":
		return os.Getenv("OPENAI_API_KEY") != ""
	case "gemini":
		return os.Getenv("GEMINI_API_KEY") != ""
	case "groq":
		return os.Getenv("GROQ_API_KEY") != ""
	case "github":
		return os.Getenv("GITHUB_TOKEN") != ""
	case "openrouter":
		return os.Getenv("OPENROUTER_API_KEY") != ""
	default:
		return false
	}
}

// loadModels loads models for the selected provider asynchronously
func (ms *ModelSelector) loadModels(providerID string) tea.Cmd {
	ms.loading = true
	ms.loadingMessage = "Loading models..."

	return func() tea.Msg {
		var models []ModelInfo

		// Load models based on provider
		switch providerID {
		case "openrouter":
			models = ms.loadOpenRouterModels()
		case "anthropic":
			models = ms.loadAnthropicModels(providerID)
		case "openai":
			models = ms.loadOpenAIModels(providerID)
		default:
			models = ms.loadDefaultModels(providerID)
		}

		// Sort models: favorites first, then alphabetically
		sort.Slice(models, func(i, j int) bool {
			if models[i].Favorite != models[j].Favorite {
				return models[i].Favorite
			}
			return models[i].Name < models[j].Name
		})

		return modelsLoadedMsg(models)
	}
}


// Bubble Tea interface implementation
func (ms *ModelSelector) Init() tea.Cmd {
	return nil
}

func (ms *ModelSelector) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case modelsLoadedMsg:
		ms.loading = false
		ms.models = []ModelInfo(msg)
		ms.selectedIndex = 0
		return ms, nil

	case loadingMsg:
		ms.loadingMessage = string(msg)
		return ms, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			ms.result <- SelectionResult{Canceled: true}
			return ms, tea.Quit

		case "up", "k":
			if ms.selectedIndex > 0 {
				ms.selectedIndex--
			}

		case "down", "j":
			var maxIndex int
			switch ms.mode {
			case SelectingProvider:
				maxIndex = len(ms.providers) - 1
			case SelectingOpenRouterFilter:
				maxIndex = len(ms.openRouterFilters) - 1
			default:
				maxIndex = len(ms.models) - 1
			}
			if ms.selectedIndex < maxIndex {
				ms.selectedIndex++
			}

		case "enter":
			switch ms.mode {
			case SelectingProvider:
				// Select provider and load models
				if ms.selectedIndex < len(ms.providers) {
					provider := ms.providers[ms.selectedIndex]
					if provider.Available {
						if provider.ID == "openrouter" {
							// Show OpenRouter filter menu
							ms.mode = SelectingOpenRouterFilter
							ms.selectedProvider = provider.ID
							ms.selectedIndex = 0
							ms.loadOpenRouterFilters()
						} else {
							// Load models directly for other providers
							ms.mode = SelectingModel
							ms.selectedIndex = 0
							return ms, ms.loadModels(provider.ID)
						}
					}
				}
			case SelectingOpenRouterFilter:
				// Select OpenRouter filter and load filtered models
				if ms.selectedIndex < len(ms.openRouterFilters) {
					filter := ms.openRouterFilters[ms.selectedIndex]
					ms.selectedFilter = filter.ProviderKey
					ms.mode = SelectingModel
					ms.selectedIndex = 0
					return ms, ms.loadModels("openrouter")
				}
			default:
				// Select model and finish (SelectingModel mode)
				if !ms.loading && ms.selectedIndex < len(ms.models) {
					model := ms.models[ms.selectedIndex]
					ms.result <- SelectionResult{
						Provider: model.Provider,
						Model:    model.ID,
						Canceled: false,
					}
					return ms, tea.Quit
				}
			}

		case " ":
			// Toggle favorite
			if ms.mode == SelectingProvider {
				if ms.selectedIndex < len(ms.providers) {
					provider := &ms.providers[ms.selectedIndex]
					provider.Favorite = !provider.Favorite
					if provider.Favorite {
						ms.favorites.AddProviderFavorite(provider.ID)
					} else {
						ms.favorites.RemoveProviderFavorite(provider.ID)
					}
				}
			} else {
				if ms.selectedIndex < len(ms.models) {
					model := &ms.models[ms.selectedIndex]
					model.Favorite = !model.Favorite
					if model.Favorite {
						ms.favorites.AddModelFavorite(model.ID)
					} else {
						ms.favorites.RemoveModelFavorite(model.ID)
					}
				}
			}

		case "backspace":
			// Go back to previous selection level
			switch ms.mode {
			case SelectingModel:
				if ms.selectedProvider == "openrouter" {
					ms.mode = SelectingOpenRouterFilter
				} else {
					ms.mode = SelectingProvider
				}
				ms.selectedIndex = 0
			case SelectingOpenRouterFilter:
				ms.mode = SelectingProvider
				ms.selectedIndex = 0
			}
		}
	}

	return ms, nil
}

func (ms *ModelSelector) View() string {
	var b strings.Builder

	switch ms.mode {
	case SelectingProvider:
		b.WriteString(titleStyle.Render("Select AI Provider"))
		b.WriteString("\n\n")

		for i, provider := range ms.providers {
			line := ""

			// Add favorite indicator
			if provider.Favorite {
				line += favoriteStyle.Render("★ ")
			} else {
				line += "  "
			}

			// Add provider name with availability styling
			if provider.Available {
				line += availableStyle.Render(provider.Name)
			} else {
				line += unavailableStyle.Render(provider.Name + " (no API key)")
			}

			// Highlight selected item
			if i == ms.selectedIndex {
				line = selectedStyle.Render(" " + line + " ")
			}

			b.WriteString(line + "\n")
		}

		b.WriteString("\n")
		b.WriteString(helpStyle.Render("↑/↓: navigate • enter: select • space: favorite • q: quit"))

	case SelectingOpenRouterFilter:
		b.WriteString(titleStyle.Render("🌐 OpenRouter - Select Provider Filter"))
		b.WriteString("\n\n")

		for i, filter := range ms.openRouterFilters {
			line := "  " + filter.Name

			// Highlight selected item
			if i == ms.selectedIndex {
				line = selectedStyle.Render(" " + line + " ")
			}

			b.WriteString(line + "\n")
		}

		b.WriteString("\n")
		b.WriteString(helpStyle.Render("↑/↓: navigate • enter: select • backspace: back • q: quit"))

	default:
		b.WriteString(titleStyle.Render("Select Model"))
		b.WriteString("\n\n")

		if ms.loading {
			// Show loading state
			b.WriteString("  " + ms.loadingMessage + "\n")
			b.WriteString("  Please wait...\n\n")
			b.WriteString(helpStyle.Render("q: quit"))
		} else if len(ms.models) == 0 {
			// Show empty state
			b.WriteString("  No models available\n\n")
			b.WriteString(helpStyle.Render("backspace: back • q: quit"))
		} else {
			// Show models
			for i, model := range ms.models {
				line := ""

				// Add favorite indicator
				if model.Favorite {
					line += favoriteStyle.Render("★ ")
				} else {
					line += "  "
				}

				line += model.Name

				// Highlight selected item
				if i == ms.selectedIndex {
					line = selectedStyle.Render(" " + line + " ")
				}

				b.WriteString(line + "\n")
			}

			b.WriteString("\n")
			b.WriteString(helpStyle.Render("↑/↓: navigate • enter: select • space: favorite • backspace: back • q: quit"))
		}
	}

	return b.String()
}

// addOpenRouterModels fetches and adds OpenRouter models dynamically
func (ms *ModelSelector) addOpenRouterModels() {
	// Try to get OpenRouter API key
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		// Fallback to hardcoded models if no API key
		ms.addOpenRouterFallbackModels()
		return
	}

	// Fetch all models categorized by provider (sorted by date DESC)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	providerModels, err := providers.GetOpenRouterModelsByProvider(ctx, apiKey)
	if err != nil {
		// Fallback to hardcoded models on error
		ms.addOpenRouterFallbackModels()
		return
	}

	// Convert OpenRouter models to ModelInfo, showing top models from each provider
	// Limit to prevent UI performance issues
	totalModelsAdded := 0
	maxTotalModels := 50 // Limit total OpenRouter models to prevent UI spinning

	for providerName, models := range providerModels {
		if totalModelsAdded >= maxTotalModels {
			break
		}

		// Take top 2 models from each provider (already sorted by date DESC)
		maxModels := 2
		if len(models) < maxModels {
			maxModels = len(models)
		}

		// Don't exceed total limit
		if totalModelsAdded+maxModels > maxTotalModels {
			maxModels = maxTotalModels - totalModelsAdded
		}

		for i := 0; i < maxModels; i++ {
			model := models[i]

			// Filter out non-coding models
			if !ms.isCodeGenerationModel(model.ID, []string{}) {
				continue
			}

			// Extract capabilities from architecture
			capabilities := []string{"text"}
			if model.Architecture.Modality != "" {
				if strings.Contains(model.Architecture.Modality, "image") {
					capabilities = append(capabilities, "vision")
				}
				if strings.Contains(model.Architecture.Modality, "audio") {
					capabilities = append(capabilities, "audio")
				}
			}

			// Use actual OpenRouter metadata
			modelInfo := ModelInfo{
				Name:          fmt.Sprintf("[%s] %s", providerName, model.Name),
				ID:            model.ID,
				Provider:      "openrouter",
				Favorite:      ms.favorites.IsModelFavorite(model.ID),
				Description:   model.Description,
				ContextLength: model.ContextLength,
				InputPrice:    parsePrice(model.Pricing.Prompt),
				OutputPrice:   parsePrice(model.Pricing.Completion),
				Capabilities:  capabilities,
			}
			ms.models = append(ms.models, modelInfo)
			totalModelsAdded++
		}
	}
}

// addOpenRouterFallbackModels adds hardcoded OpenRouter models as fallback (June 2025)
func (ms *ModelSelector) addOpenRouterFallbackModels() {
	fallbackModels := []ModelInfo{
		{Name: "Claude 3.5 Sonnet (Latest)", ID: "anthropic/claude-3.5-sonnet-20241022", Provider: "openrouter"},
		{Name: "GPT-4o (Latest)", ID: "openai/gpt-4o-2024-08-06", Provider: "openrouter"},
		{Name: "GPT-4o Mini (Latest)", ID: "openai/gpt-4o-mini-2024-07-18", Provider: "openrouter"},
		{Name: "o1 Preview", ID: "openai/o1-preview-2024-09-12", Provider: "openrouter"},
		{Name: "Gemini 2.5 Pro (Latest)", ID: "google/gemini-2.5-pro", Provider: "openrouter"},
		{Name: "Gemini 2.5 Flash (Latest)", ID: "google/gemini-2.5-flash", Provider: "openrouter"},
		{Name: "Llama 3.3 70B (Latest)", ID: "meta-llama/llama-3.3-70b-instruct", Provider: "openrouter"},
		{Name: "Mistral Large 2407", ID: "mistralai/mistral-large-2407", Provider: "openrouter"},
		{Name: "DeepSeek R1 (Latest)", ID: "deepseek/deepseek-r1-0528", Provider: "openrouter"},
		{Name: "Grok 3 (Latest)", ID: "x-ai/grok-3", Provider: "openrouter"},
		{Name: "Command R+ (Latest)", ID: "cohere/command-r-plus-08-2024", Provider: "openrouter"},
		{Name: "MiniMax M1", ID: "minimax/minimax-m1", Provider: "openrouter"},
		{Name: "Inception Mercury", ID: "inception/mercury", Provider: "openrouter"},
	}

	for _, model := range fallbackModels {
		model.Favorite = ms.favorites.IsModelFavorite(model.ID)
		ms.models = append(ms.models, model)
	}
}

// parsePrice converts OpenRouter price string to float64
func parsePrice(priceStr string) float64 {
	if priceStr == "" {
		return 0.0
	}

	// OpenRouter prices are typically in scientific notation like "3e-06"
	if price, err := strconv.ParseFloat(priceStr, 64); err == nil {
		// Convert to price per million tokens (multiply by 1,000,000)
		return price * 1000000
	}

	return 0.0
}

// loadOpenRouterModels loads OpenRouter models and returns them
func (ms *ModelSelector) loadOpenRouterModels() []ModelInfo {
	// Get OpenRouter API key
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		// Return fallback models if no API key
		return ms.loadDefaultModels("openrouter")
	}

	// Get models from database
	ctx := context.Background()

	// If a specific provider filter is selected, get models for that provider
	if ms.selectedFilter != "" {
		models, err := providers.GetOpenRouterModelsBySpecificProvider(ctx, apiKey, ms.selectedFilter)
		if err != nil {
			// Return fallback models on error
			return ms.loadDefaultModels("openrouter")
		}

		// Convert OpenRouter models to ModelInfo
		var modelInfos []ModelInfo
		maxModels := 50 // Limit total models to prevent UI issues

		for i, model := range models {
			if i >= maxModels {
				break
			}

			// Filter out non-coding models
			if !ms.isCodeGenerationModel(model.ID, []string{}) {
				continue
			}

			// Use basic model information from database
			modelInfo := ModelInfo{
				Name:          model.Name,
				ID:            model.ID,
				Provider:      "openrouter",
				Favorite:      ms.favorites.IsModelFavorite(model.ID),
				Description:   model.Description,
				ContextLength: model.ContextLength,
				Capabilities:  []string{"text"}, // Default capability
			}
			modelInfos = append(modelInfos, modelInfo)
		}

		return modelInfos
	}

	// If no filter selected, get all models
	allModels, err := providers.GetOpenRouterModels(ctx, apiKey)
	if err != nil {
		// Return fallback models on error
		return ms.loadDefaultModels("openrouter")
	}

	// Convert OpenRouter models to ModelInfo
	var modelInfos []ModelInfo
	maxModels := 50 // Limit total models to prevent UI issues

	for i, model := range allModels {
		if i >= maxModels {
			break
		}

		// Filter out non-coding models
		if !ms.isCodeGenerationModel(model.ID, []string{}) {
			continue
		}

		// Use basic model information from database
		modelInfo := ModelInfo{
			Name:          model.Name,
			ID:            model.ID,
			Provider:      "openrouter",
			Favorite:      ms.favorites.IsModelFavorite(model.ID),
			Description:   model.Description,
			ContextLength: model.ContextLength,
			Capabilities:  []string{"text"}, // Default capability
		}
		modelInfos = append(modelInfos, modelInfo)
	}

	return modelInfos
}

// loadDefaultModels loads default models for a provider and returns them
func (ms *ModelSelector) loadDefaultModels(providerID string) []ModelInfo {
	var models []ModelInfo

	defaults := map[string][]ModelInfo{
		"anthropic": ms.loadAnthropicModels(providerID),
		"openai":    ms.loadOpenAIModels(providerID),
		"gemini":    ms.loadGeminiModels(providerID),
		"openrouter": {
			{
				Name: "Claude 3.5 Sonnet", ID: "anthropic/claude-3.5-sonnet", Provider: providerID,
				Description: "Claude 3.5 Sonnet via OpenRouter", ContextLength: 200000,
				InputPrice: 3.0, OutputPrice: 15.0, Capabilities: []string{"text", "code", "reasoning"},
			},
			{
				Name: "GPT-4o", ID: "openai/gpt-4o", Provider: providerID,
				Description: "GPT-4o via OpenRouter", ContextLength: 128000,
				InputPrice: 5.0, OutputPrice: 15.0, Capabilities: []string{"text", "code", "vision"},
			},
		},
	}

	if providerModels, exists := defaults[providerID]; exists {
		for _, model := range providerModels {
			model.Favorite = ms.favorites.IsModelFavorite(model.ID)
			models = append(models, model)
		}
	}

	return models
}

// loadOpenAIModels loads OpenAI models from cache (populated by background fetcher)
func (ms *ModelSelector) loadOpenAIModels(providerID string) []ModelInfo {
	// Try to get OpenAI API key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		// Return fallback models if no API key
		return ms.getOpenAIFallbackModels(providerID)
	}

	// Get models from cache (no timeout needed since it's just reading from cache)
	ctx := context.Background()
	openaiModels, err := providers.GetOpenAIModels(ctx, apiKey)
	if err != nil {
		// Return fallback models on error
		return ms.getOpenAIFallbackModels(providerID)
	}

	// Convert OpenAI models to ModelInfo, filtering out non-coding models
	var models []ModelInfo
	for _, model := range openaiModels {
		// Filter out audio, video, image, and other non-coding models
		if !ms.isCodeGenerationModel(model.ID, []string{}) {
			continue
		}

		modelInfo := ms.convertOpenAIModelToModelInfo(model, providerID)
		models = append(models, modelInfo)
	}

	// If no models found, return fallback
	if len(models) == 0 {
		return ms.getOpenAIFallbackModels(providerID)
	}

	return models
}

// getOpenAIFallbackModels returns fallback OpenAI models
func (ms *ModelSelector) getOpenAIFallbackModels(providerID string) []ModelInfo {
	return []ModelInfo{
		{
			Name: "GPT-4o (Latest)", ID: "gpt-4o", Provider: providerID,
			Description: "Multimodal flagship model", ContextLength: 128000,
			InputPrice: 5.0, OutputPrice: 15.0, Capabilities: []string{"text", "code", "vision", "audio"},
			Favorite: ms.favorites.IsModelFavorite("gpt-4o"),
		},
		{
			Name: "GPT-4o Mini (Latest)", ID: "gpt-4o-mini", Provider: providerID,
			Description: "Affordable and intelligent small model", ContextLength: 128000,
			InputPrice: 0.15, OutputPrice: 0.6, Capabilities: []string{"text", "code", "speed"},
			Favorite: ms.favorites.IsModelFavorite("gpt-4o-mini"),
		},
	}
}

// convertOpenAIModelToModelInfo converts an OpenAI model to ModelInfo
func (ms *ModelSelector) convertOpenAIModelToModelInfo(model providers.OpenAIModelInfo, providerID string) ModelInfo {
	// Create user-friendly name
	name := strings.ReplaceAll(model.ID, "-", " ")
	// Convert to title case manually since strings.Title is deprecated
	words := strings.Fields(name)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	name = strings.Join(words, " ")

	// Set default pricing and capabilities based on model type
	var inputPrice, outputPrice float64
	var capabilities []string
	var contextLength int
	var description string

	switch {
	case strings.Contains(model.ID, "gpt-4o"):
		inputPrice, outputPrice = 5.0, 15.0
		capabilities = []string{"text", "code", "vision", "audio"}
		contextLength = 128000
		description = "Multimodal flagship model"
	case strings.Contains(model.ID, "gpt-4"):
		inputPrice, outputPrice = 30.0, 60.0
		capabilities = []string{"text", "code", "reasoning"}
		contextLength = 128000
		description = "Advanced reasoning model"
	case strings.Contains(model.ID, "o1"):
		inputPrice, outputPrice = 15.0, 60.0
		capabilities = []string{"reasoning", "complex-tasks"}
		contextLength = 128000
		description = "Advanced reasoning model"
	case strings.Contains(model.ID, "gpt-3.5"):
		inputPrice, outputPrice = 0.5, 1.5
		capabilities = []string{"text", "code", "speed"}
		contextLength = 16000
		description = "Fast and efficient model"
	default:
		inputPrice, outputPrice = 2.5, 10.0
		capabilities = []string{"text", "code"}
		contextLength = 128000
		description = "OpenAI model"
	}

	return ModelInfo{
		Name:          name,
		ID:            model.ID,
		Provider:      providerID,
		Description:   description,
		ContextLength: contextLength,
		InputPrice:    inputPrice,
		OutputPrice:   outputPrice,
		Capabilities:  capabilities,
		Favorite:      ms.favorites.IsModelFavorite(model.ID),
	}
}

// loadAnthropicModels loads Anthropic models from cache (populated by background fetcher)
func (ms *ModelSelector) loadAnthropicModels(providerID string) []ModelInfo {
	// Try to get Anthropic API key
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		// Return fallback models if no API key
		return ms.getAnthropicFallbackModels(providerID)
	}

	// Get models from cache (no timeout needed since it's just reading from cache)
	ctx := context.Background()
	anthropicModels, err := providers.GetAnthropicModels(ctx, apiKey)
	if err != nil {
		// Return fallback models on error
		return ms.getAnthropicFallbackModels(providerID)
	}

	// Convert Anthropic models to ModelInfo, filtering out non-coding models
	var models []ModelInfo
	for _, model := range anthropicModels {
		// Filter out non-coding models
		if !ms.isCodeGenerationModel(model.ID, []string{}) {
			continue
		}

		modelInfo := ms.convertAnthropicModelToModelInfo(model, providerID)
		models = append(models, modelInfo)
	}

	// If no models found, return fallback
	if len(models) == 0 {
		return ms.getAnthropicFallbackModels(providerID)
	}

	return models
}

// getAnthropicFallbackModels returns fallback Anthropic models
func (ms *ModelSelector) getAnthropicFallbackModels(providerID string) []ModelInfo {
	return []ModelInfo{
		{
			Name: "Claude 3.5 Sonnet", ID: "claude-3-5-sonnet-20241022", Provider: providerID,
			Description: "Most intelligent model for complex reasoning", ContextLength: 200000,
			InputPrice: 3.0, OutputPrice: 15.0, Capabilities: []string{"text", "code", "vision", "reasoning"},
			Favorite: ms.favorites.IsModelFavorite("claude-3-5-sonnet-20241022"),
		},
		{
			Name: "Claude 3.5 Haiku", ID: "claude-3-5-haiku-20241022", Provider: providerID,
			Description: "Fastest model for simple tasks", ContextLength: 200000,
			InputPrice: 0.8, OutputPrice: 4.0, Capabilities: []string{"text", "code", "speed"},
			Favorite: ms.favorites.IsModelFavorite("claude-3-5-haiku-20241022"),
		},
	}
}

// convertAnthropicModelToModelInfo converts an Anthropic model to ModelInfo
func (ms *ModelSelector) convertAnthropicModelToModelInfo(model providers.AnthropicModelInfo, providerID string) ModelInfo {
	return ModelInfo{
		Name:          model.DisplayName,
		ID:            model.ID,
		Provider:      providerID,
		Description:   fmt.Sprintf("Anthropic %s model", model.Type),
		ContextLength: 200000, // All Claude models support 200k context
		InputPrice:    model.InputPrice,
		OutputPrice:   model.OutputPrice,
		Capabilities:  []string{"text", "code", "vision", "reasoning"},
		Favorite:      ms.favorites.IsModelFavorite(model.ID),
	}
}

// loadGeminiModels loads Gemini models from cache (populated by background fetcher)
func (ms *ModelSelector) loadGeminiModels(providerID string) []ModelInfo {
	// Try to get Gemini API key
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		// Return fallback models if no API key
		return ms.getGeminiFallbackModels(providerID)
	}

	// Get models from cache (no timeout needed since it's just reading from cache)
	ctx := context.Background()
	geminiModels, err := providers.GetGeminiModels(ctx, apiKey)
	if err != nil {
		// Return fallback models on error
		return ms.getGeminiFallbackModels(providerID)
	}

	// Convert Gemini models to ModelInfo, filtering out non-coding models
	var models []ModelInfo
	for _, model := range geminiModels {
		// Filter out non-coding models
		if !ms.isCodeGenerationModel(model.ID, []string{}) {
			continue
		}

		modelInfo := ms.convertGeminiModelToModelInfo(model, providerID)
		models = append(models, modelInfo)
	}

	// If no models found, return fallback
	if len(models) == 0 {
		return ms.getGeminiFallbackModels(providerID)
	}

	return models
}

// getGeminiFallbackModels returns fallback Gemini models
func (ms *ModelSelector) getGeminiFallbackModels(providerID string) []ModelInfo {
	return []ModelInfo{
		{
			Name: "Gemini 2.0 Flash (Experimental)", ID: "gemini-2.0-flash-exp", Provider: providerID,
			Description: "Latest experimental Gemini model", ContextLength: 1000000,
			InputPrice: 0.075, OutputPrice: 0.3, Capabilities: []string{"text", "code", "vision", "reasoning"},
			Favorite: ms.favorites.IsModelFavorite("gemini-2.0-flash-exp"),
		},
		{
			Name: "Gemini 1.5 Pro", ID: "gemini-1.5-pro", Provider: providerID,
			Description: "Most capable Gemini model", ContextLength: 2000000,
			InputPrice: 1.25, OutputPrice: 5.0, Capabilities: []string{"text", "code", "vision", "reasoning"},
			Favorite: ms.favorites.IsModelFavorite("gemini-1.5-pro"),
		},
		{
			Name: "Gemini 1.5 Flash", ID: "gemini-1.5-flash", Provider: providerID,
			Description: "Fast and efficient Gemini model", ContextLength: 1000000,
			InputPrice: 0.075, OutputPrice: 0.3, Capabilities: []string{"text", "code", "vision", "speed"},
			Favorite: ms.favorites.IsModelFavorite("gemini-1.5-flash"),
		},
	}
}

// convertGeminiModelToModelInfo converts a Gemini model to ModelInfo
func (ms *ModelSelector) convertGeminiModelToModelInfo(model providers.GeminiModelInfo, providerID string) ModelInfo {
	// Determine context length based on model
	contextLength := 128000 // Default
	if strings.Contains(model.ID, "1.5-pro") {
		contextLength = 2000000 // 2M tokens
	} else if strings.Contains(model.ID, "1.5-flash") || strings.Contains(model.ID, "2.0-flash") {
		contextLength = 1000000 // 1M tokens
	}

	return ModelInfo{
		Name:          model.DisplayName,
		ID:            model.ID,
		Provider:      providerID,
		Description:   model.Description,
		ContextLength: contextLength,
		InputPrice:    model.InputPrice,
		OutputPrice:   model.OutputPrice,
		Capabilities:  []string{"text", "code", "vision", "reasoning"},
		Favorite:      ms.favorites.IsModelFavorite(model.ID),
	}
}

// isCodeGenerationModel filters out non-coding models (audio, video, image, etc.)
func (ms *ModelSelector) isCodeGenerationModel(modelName string, _ []string) bool {
	modelLower := strings.ToLower(modelName)

	// Exclude audio/video/image models
	excludePatterns := []string{
		"audio", "video", "realtime", "transcribe", "tts", "image", "vision",
		"whisper", "dall-e", "tts-1", "embedding", "moderation",
	}

	for _, pattern := range excludePatterns {
		if strings.Contains(modelLower, pattern) {
			return false
		}
	}

	// TEMPORARILY ALLOW ALL OTHER MODELS TO DEBUG
	return true

	// Include only text-based coding models
	// includePatterns := []string{
	// 	"gpt-4o", "gpt-4", "o1", "gpt-3.5", "claude", "gemini", "llama", "mistral",
	// 	"deepseek", "qwen", "coder", "code", "instruct",
	// }

	// for _, pattern := range includePatterns {
	// 	if strings.Contains(modelLower, pattern) {
	// 		return true
	// 	}
	// }

	// Default to false for unknown models
	// return false
}
