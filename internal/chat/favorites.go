package chat

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Favorites manages favorite providers and models
type Favorites struct {
	Providers []string `json:"providers"`
	Models    []string `json:"models"`
	filePath  string
}

// FavoriteItem represents a favorite provider or model for display
type FavoriteItem struct {
	Type string // "provider" or "model"
	Name string
	ID   string
}

// NewFavorites creates a new favorites manager
func NewFavorites() (*Favorites, error) {
	// Get config directory
	configDir, err := getConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	filePath := filepath.Join(configDir, "favorites.json")
	
	favorites := &Favorites{
		Providers: []string{},
		Models:    []string{},
		filePath:  filePath,
	}

	// Load existing favorites
	if err := favorites.load(); err != nil {
		// If file doesn't exist, that's okay - we'll create it on first save
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load favorites: %w", err)
		}
	}

	return favorites, nil
}

// getConfigDir returns the configuration directory for CodeForge
func getConfigDir() (string, error) {
	// Try XDG_CONFIG_HOME first
	if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
		return filepath.Join(configHome, "codeforge"), nil
	}

	// Fall back to ~/.config/codeforge
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(homeDir, ".config", "codeforge"), nil
}

// load reads favorites from the JSON file
func (f *Favorites) load() error {
	data, err := os.ReadFile(f.filePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, f)
}

// save writes favorites to the JSON file
func (f *Favorites) save() error {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(f.filePath, data, 0644)
}

// IsProviderFavorite checks if a provider is marked as favorite
func (f *Favorites) IsProviderFavorite(providerID string) bool {
	for _, id := range f.Providers {
		if id == providerID {
			return true
		}
	}
	return false
}

// IsModelFavorite checks if a model is marked as favorite
func (f *Favorites) IsModelFavorite(modelID string) bool {
	for _, id := range f.Models {
		if id == modelID {
			return true
		}
	}
	return false
}

// AddProviderFavorite adds a provider to favorites
func (f *Favorites) AddProviderFavorite(providerID string) error {
	if !f.IsProviderFavorite(providerID) {
		f.Providers = append(f.Providers, providerID)
		sort.Strings(f.Providers)
		return f.save()
	}
	return nil
}

// RemoveProviderFavorite removes a provider from favorites
func (f *Favorites) RemoveProviderFavorite(providerID string) error {
	for i, id := range f.Providers {
		if id == providerID {
			f.Providers = append(f.Providers[:i], f.Providers[i+1:]...)
			return f.save()
		}
	}
	return nil
}

// AddModelFavorite adds a model to favorites
func (f *Favorites) AddModelFavorite(modelID string) error {
	if !f.IsModelFavorite(modelID) {
		f.Models = append(f.Models, modelID)
		sort.Strings(f.Models)
		return f.save()
	}
	return nil
}

// RemoveModelFavorite removes a model from favorites
func (f *Favorites) RemoveModelFavorite(modelID string) error {
	for i, id := range f.Models {
		if id == modelID {
			f.Models = append(f.Models[:i], f.Models[i+1:]...)
			return f.save()
		}
	}
	return nil
}

// GetAllFavorites returns all favorites for display
func (f *Favorites) GetAllFavorites() []FavoriteItem {
	var items []FavoriteItem

	// Add provider favorites
	for _, providerID := range f.Providers {
		items = append(items, FavoriteItem{
			Type: "provider",
			Name: strings.Title(providerID),
			ID:   providerID,
		})
	}

	// Add model favorites
	for _, modelID := range f.Models {
		// Extract a readable name from the model ID
		name := f.getModelDisplayName(modelID)
		items = append(items, FavoriteItem{
			Type: "model",
			Name: name,
			ID:   modelID,
		})
	}

	return items
}

// getModelDisplayName converts a model ID to a display name
func (f *Favorites) getModelDisplayName(modelID string) string {
	// Handle common model ID patterns
	switch {
	case strings.HasPrefix(modelID, "claude-3-5-sonnet"):
		return "Claude 3.5 Sonnet"
	case strings.HasPrefix(modelID, "claude-3-5-haiku"):
		return "Claude 3.5 Haiku"
	case strings.HasPrefix(modelID, "claude-3-opus"):
		return "Claude 3 Opus"
	case strings.HasPrefix(modelID, "gpt-4o-mini"):
		return "GPT-4o Mini"
	case strings.HasPrefix(modelID, "gpt-4o"):
		return "GPT-4o"
	case strings.HasPrefix(modelID, "gpt-4-turbo"):
		return "GPT-4 Turbo"
	case strings.HasPrefix(modelID, "gemini-1.5-pro"):
		return "Gemini 1.5 Pro"
	case strings.HasPrefix(modelID, "gemini-1.5-flash"):
		return "Gemini 1.5 Flash"
	case strings.Contains(modelID, "llama-3.1-70b"):
		return "Llama 3.1 70B"
	case strings.Contains(modelID, "llama-3.1-8b"):
		return "Llama 3.1 8B"
	default:
		// Clean up the model ID for display
		name := modelID
		// Remove provider prefixes
		if idx := strings.LastIndex(name, "/"); idx != -1 {
			name = name[idx+1:]
		}
		// Replace dashes with spaces and title case
		name = strings.ReplaceAll(name, "-", " ")
		name = strings.Title(name)
		return name
	}
}

// GetFavoriteProviders returns just the favorite providers
func (f *Favorites) GetFavoriteProviders() []string {
	return append([]string{}, f.Providers...)
}

// GetFavoriteModels returns just the favorite models
func (f *Favorites) GetFavoriteModels() []string {
	return append([]string{}, f.Models...)
}

// Clear removes all favorites
func (f *Favorites) Clear() error {
	f.Providers = []string{}
	f.Models = []string{}
	return f.save()
}

// GetStats returns statistics about favorites
func (f *Favorites) GetStats() (int, int) {
	return len(f.Providers), len(f.Models)
}

// Export returns favorites as a JSON string for backup
func (f *Favorites) Export() (string, error) {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Import loads favorites from a JSON string
func (f *Favorites) Import(jsonData string) error {
	var imported Favorites
	if err := json.Unmarshal([]byte(jsonData), &imported); err != nil {
		return err
	}

	f.Providers = imported.Providers
	f.Models = imported.Models
	
	// Sort and save
	sort.Strings(f.Providers)
	sort.Strings(f.Models)
	
	return f.save()
}
