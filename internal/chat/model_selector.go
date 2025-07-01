package chat

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/entrepeneur4lyf/codeforge/internal/llm/providers"
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
			Name:      strings.Title(name),
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

// loadOpenRouterFilters loads the available provider filters for OpenRouter
func (ms *ModelSelector) loadOpenRouterFilters() {
	ms.openRouterFilters = []OpenRouterFilter{
		{Name: "All Providers", ProviderKey: "", Description: "Show models from all providers"},
		{Name: "Anthropic", ProviderKey: "anthropic", Description: "Claude models via OpenRouter"},
		{Name: "OpenAI", ProviderKey: "openai", Description: "GPT models via OpenRouter"},
		{Name: "💎 Google", ProviderKey: "google", Description: "Gemini models via OpenRouter"},
		{Name: "🦙 Meta/Llama", ProviderKey: "meta-llama", Description: "Llama models via OpenRouter"},
		{Name: "🌊 Mistral", ProviderKey: "mistralai", Description: "Mistral models via OpenRouter"},
		{Name: "🧠 DeepSeek", ProviderKey: "deepseek", Description: "DeepSeek models via OpenRouter"},
		{Name: "⚡ xAI", ProviderKey: "x-ai", Description: "Grok models via OpenRouter"},
		{Name: "🔮 Cohere", ProviderKey: "cohere", Description: "Command models via OpenRouter"},
		{Name: "Others", ProviderKey: "others", Description: "Other providers via OpenRouter"},
	}
}

// getProviderNameFromFilter converts filter key to provider name
func getProviderNameFromFilter(filter string) string {
	switch filter {
	case "anthropic":
		return "Anthropic"
	case "openai":
		return "OpenAI"
	case "google":
		return "Google"
	case "meta-llama":
		return "Meta"
	case "mistralai":
		return "Mistral AI"
	case "deepseek":
		return "DeepSeek"
	case "x-ai":
		return "01.AI"
	case "cohere":
		return "Cohere"
	default:
		return strings.Title(filter)
	}
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
		if providerID == "openrouter" {
			models = ms.loadOpenRouterModels()
		} else {
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

// addDefaultModels adds default models when registry is empty
func (ms *ModelSelector) addDefaultModels(providerID string) {
	// Special handling for OpenRouter - fetch dynamic models
	if providerID == "openrouter" {
		ms.addOpenRouterModels()
		return
	}
	defaults := map[string][]ModelInfo{
		"anthropic": {
			{
				Name: "Claude 3.5 Sonnet", ID: "claude-3-5-sonnet-20241022", Provider: providerID,
				Description: "Most intelligent model for complex reasoning", ContextLength: 200000,
				InputPrice: 3.0, OutputPrice: 15.0, Capabilities: []string{"text", "code", "vision", "reasoning"},
			},
			{
				Name: "Claude 3.5 Haiku", ID: "claude-3-5-haiku-20241022", Provider: providerID,
				Description: "Fastest model for simple tasks", ContextLength: 200000,
				InputPrice: 0.25, OutputPrice: 1.25, Capabilities: []string{"text", "code", "speed"},
			},
			{
				Name: "Claude 3 Opus", ID: "claude-3-opus-20240229", Provider: providerID,
				Description: "Most powerful model for complex tasks", ContextLength: 200000,
				InputPrice: 15.0, OutputPrice: 75.0, Capabilities: []string{"text", "code", "vision", "reasoning"},
			},
		},
		"openai": {
			{
				Name: "GPT-4o (Latest)", ID: "gpt-4o-2024-08-06", Provider: providerID,
				Description: "Multimodal flagship model", ContextLength: 128000,
				InputPrice: 5.0, OutputPrice: 15.0, Capabilities: []string{"text", "code", "vision", "audio"},
			},
			{
				Name: "GPT-4o Mini (Latest)", ID: "gpt-4o-mini-2024-07-18", Provider: providerID,
				Description: "Affordable and intelligent small model", ContextLength: 128000,
				InputPrice: 0.15, OutputPrice: 0.6, Capabilities: []string{"text", "code", "speed"},
			},
			{
				Name: "o1 Preview", ID: "o1-preview-2024-09-12", Provider: providerID,
				Description: "Advanced reasoning model", ContextLength: 128000,
				InputPrice: 15.0, OutputPrice: 60.0, Capabilities: []string{"reasoning", "math", "code"},
			},
			{
				Name: "o1 Mini", ID: "o1-mini-2024-09-12", Provider: providerID,
				Description: "Smaller reasoning model", ContextLength: 128000,
				InputPrice: 3.0, OutputPrice: 12.0, Capabilities: []string{"reasoning", "math", "speed"},
			},
			{
				Name: "ChatGPT-4o Latest", ID: "chatgpt-4o-latest", Provider: providerID,
				Description: "Latest ChatGPT model", ContextLength: 128000,
				InputPrice: 5.0, OutputPrice: 15.0, Capabilities: []string{"text", "code", "chat"},
			},
		},
		"gemini": {
			{Name: "Gemini 2.5 Pro (Latest)", ID: "gemini-2.5-pro", Provider: providerID},
			{Name: "Gemini 2.5 Flash (Latest)", ID: "gemini-2.5-flash", Provider: providerID},
			{Name: "Gemini 1.5 Pro", ID: "gemini-1.5-pro-latest", Provider: providerID},
		},
		"groq": {
			{Name: "Llama 3.3 70B (Latest)", ID: "llama-3.3-70b-versatile", Provider: providerID},
			{Name: "Llama 3.1 70B", ID: "llama-3.1-70b-versatile", Provider: providerID},
			{Name: "Llama 3.1 8B", ID: "llama-3.1-8b-instant", Provider: providerID},
		},
		"github": {
			{Name: "GPT-4o (Latest)", ID: "gpt-4o-2024-08-06", Provider: providerID},
			{Name: "GPT-4o Mini (Latest)", ID: "gpt-4o-mini-2024-07-18", Provider: providerID},
			{Name: "o1 Preview", ID: "o1-preview-2024-09-12", Provider: providerID},
		},
		"xai": {
			{Name: "Grok 3 (Latest)", ID: "grok-3", Provider: providerID},
			{Name: "Grok 3 Mini", ID: "grok-3-mini", Provider: providerID},
		},
		"mistral": {
			{Name: "Mistral Large 2407 (Latest)", ID: "mistral-large-2407", Provider: providerID},
			{Name: "Mistral Small 3.2 24B", ID: "mistral-small-3.2-24b-instruct", Provider: providerID},
			{Name: "Magistral Medium", ID: "magistral-medium-2506", Provider: providerID},
		},
		"deepseek": {
			{Name: "DeepSeek R1 (Latest)", ID: "deepseek-r1-0528", Provider: providerID},
			{Name: "DeepSeek R1 Distill", ID: "deepseek-r1-distill-qwen-7b", Provider: providerID},
		},
		"ollama": {
			{Name: "Llama 3.1 8B", ID: "llama3.1:8b", Provider: providerID},
			{Name: "Llama 3.1 70B", ID: "llama3.1:70b", Provider: providerID},
			{Name: "Code Llama", ID: "codellama:13b", Provider: providerID},
			{Name: "Mistral 7B", ID: "mistral:7b", Provider: providerID},
			{Name: "DeepSeek Coder", ID: "deepseek-coder:6.7b", Provider: providerID},
		},
	}

	if models, exists := defaults[providerID]; exists {
		for _, model := range models {
			model.Favorite = ms.favorites.IsModelFavorite(model.ID)
			ms.models = append(ms.models, model)
		}
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
			maxIndex := 0
			if ms.mode == SelectingProvider {
				maxIndex = len(ms.providers) - 1
			} else if ms.mode == SelectingOpenRouterFilter {
				maxIndex = len(ms.openRouterFilters) - 1
			} else {
				maxIndex = len(ms.models) - 1
			}
			if ms.selectedIndex < maxIndex {
				ms.selectedIndex++
			}

		case "enter":
			if ms.mode == SelectingProvider {
				// Select provider and load models
				if ms.selectedIndex < len(ms.providers) {
					provider := ms.providers[ms.selectedIndex]
					if provider.Available {
						if provider.ID == "openrouter" {
							// Show OpenRouter filter menu
							ms.mode = SelectingOpenRouterFilter
							ms.selectedProvider = provider.ID
							ms.selectedIndex = 0
						} else {
							// Load models directly for other providers
							ms.mode = SelectingModel
							ms.selectedIndex = 0
							return ms, ms.loadModels(provider.ID)
						}
					}
				}
			} else if ms.mode == SelectingOpenRouterFilter {
				// Select OpenRouter filter and load filtered models
				if ms.selectedIndex < len(ms.openRouterFilters) {
					filter := ms.openRouterFilters[ms.selectedIndex]
					ms.selectedFilter = filter.ProviderKey
					ms.mode = SelectingModel
					ms.selectedIndex = 0
					return ms, ms.loadModels("openrouter")
				}
			} else {
				// Select model and finish
				if ms.selectedIndex < len(ms.models) {
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
			if ms.mode == SelectingModel {
				if ms.selectedProvider == "openrouter" {
					ms.mode = SelectingOpenRouterFilter
				} else {
					ms.mode = SelectingProvider
				}
				ms.selectedIndex = 0
			} else if ms.mode == SelectingOpenRouterFilter {
				ms.mode = SelectingProvider
				ms.selectedIndex = 0
			}
		}
	}

	return ms, nil
}

func (ms *ModelSelector) View() string {
	var b strings.Builder

	if ms.mode == SelectingProvider {
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

	} else if ms.mode == SelectingOpenRouterFilter {
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

	} else {
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
	var models []ModelInfo

	// Get OpenRouter API key
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		// Return fallback models if no API key
		return ms.loadDefaultModels("openrouter")
	}

	// Fetch models from OpenRouter API in background
	ctx := context.Background()
	providerModels, err := providers.GetOpenRouterModelsByProvider(ctx, apiKey)
	if err != nil {
		// Return fallback models on error
		return ms.loadDefaultModels("openrouter")
	}

	// Convert OpenRouter models to ModelInfo, limiting to prevent UI issues
	totalModelsAdded := 0
	maxTotalModels := 50 // Limit total OpenRouter models to prevent UI spinning

	for providerName, modelList := range providerModels {
		if totalModelsAdded >= maxTotalModels {
			break
		}

		// Take top 2 models from each provider (already sorted by date DESC)
		maxModels := 2
		if len(modelList) < maxModels {
			maxModels = len(modelList)
		}

		// Don't exceed total limit
		if totalModelsAdded+maxModels > maxTotalModels {
			maxModels = maxTotalModels - totalModelsAdded
		}

		for i := 0; i < maxModels; i++ {
			model := modelList[i]

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
			models = append(models, modelInfo)
			totalModelsAdded++
		}
	}

	return models
}

// loadDefaultModels loads default models for a provider and returns them
func (ms *ModelSelector) loadDefaultModels(providerID string) []ModelInfo {
	var models []ModelInfo

	defaults := map[string][]ModelInfo{
		"anthropic": {
			{
				Name: "Claude 3.5 Sonnet", ID: "claude-3-5-sonnet-20241022", Provider: providerID,
				Description: "Most intelligent model for complex reasoning", ContextLength: 200000,
				InputPrice: 3.0, OutputPrice: 15.0, Capabilities: []string{"text", "code", "vision", "reasoning"},
			},
			{
				Name: "Claude 3.5 Haiku", ID: "claude-3-5-haiku-20241022", Provider: providerID,
				Description: "Fastest model for simple tasks", ContextLength: 200000,
				InputPrice: 0.25, OutputPrice: 1.25, Capabilities: []string{"text", "code", "speed"},
			},
			{
				Name: "Claude 3 Opus", ID: "claude-3-opus-20240229", Provider: providerID,
				Description: "Most powerful model for complex tasks", ContextLength: 200000,
				InputPrice: 15.0, OutputPrice: 75.0, Capabilities: []string{"text", "code", "vision", "reasoning"},
			},
		},
		"openai": {
			{
				Name: "GPT-4o (Latest)", ID: "gpt-4o-2024-08-06", Provider: providerID,
				Description: "Multimodal flagship model", ContextLength: 128000,
				InputPrice: 5.0, OutputPrice: 15.0, Capabilities: []string{"text", "code", "vision", "audio"},
			},
			{
				Name: "GPT-4o Mini (Latest)", ID: "gpt-4o-mini-2024-07-18", Provider: providerID,
				Description: "Affordable and intelligent small model", ContextLength: 128000,
				InputPrice: 0.15, OutputPrice: 0.6, Capabilities: []string{"text", "code", "speed"},
			},
		},
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
