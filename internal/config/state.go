package config

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// State represents the application state like OpenCode
type State struct {
	Theme    string `toml:"theme"`
	Provider string `toml:"provider"`
	Model    string `toml:"model"`
	
	// Additional CodeForge-specific settings
	WorkingDirectory string `toml:"working_directory"`
	LSPEnabled       bool   `toml:"lsp_enabled"`
	VectorDBPath     string `toml:"vector_db_path"`
}

// NewState creates a new state with default values
func NewState() *State {
	return &State{
		Theme:       "opencode",
		Provider:    "anthropic",
		Model:       "claude-3-5-sonnet-20241022",
		LSPEnabled:  true,
	}
}

// GetModelDisplay returns a formatted model name for display in the status bar
func (s *State) GetModelDisplay() string {
	if s.Provider == "" || s.Model == "" {
		return "🤖 No Model Selected"
	}
	
	// Format like OpenCode: provider/model
	switch s.Provider {
	case "anthropic":
		switch s.Model {
		case "claude-3-5-sonnet-20241022":
			return "🤖 Claude 3.5 Sonnet"
		case "claude-3-haiku-20240307":
			return "🤖 Claude 3 Haiku"
		default:
			return "🤖 Claude"
		}
	case "openai":
		switch s.Model {
		case "gpt-4":
			return "🤖 GPT-4"
		case "gpt-3.5-turbo":
			return "🤖 GPT-3.5 Turbo"
		default:
			return "🤖 OpenAI"
		}
	case "google":
		return "🤖 Gemini"
	case "groq":
		return "🤖 Groq"
	default:
		return fmt.Sprintf("🤖 %s", s.Provider)
	}
}

// SaveState writes the state to a TOML file like OpenCode
func SaveState(filePath string, state *State) error {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", dir, err)
	}
	
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create/open config file %s: %w", filePath, err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	encoder := toml.NewEncoder(writer)
	if err := encoder.Encode(state); err != nil {
		return fmt.Errorf("failed to encode state to TOML file %s: %w", filePath, err)
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush writer for state file %s: %w", filePath, err)
	}

	slog.Debug("State saved to file", "file", filePath)
	return nil
}

// LoadState loads the state from a TOML file like OpenCode
func LoadState(filePath string) (*State, error) {
	var state State
	if _, err := toml.DecodeFile(filePath, &state); err != nil {
		if _, statErr := os.Stat(filePath); os.IsNotExist(statErr) {
			// Return default state if file doesn't exist
			return NewState(), nil
		}
		return nil, fmt.Errorf("failed to decode TOML from file %s: %w", filePath, err)
	}
	return &state, nil
}

// GetConfigPath returns the default config file path
func GetConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./codeforge-config.toml"
	}
	return filepath.Join(homeDir, ".config", "codeforge", "config.toml")
}

// UpdateModel updates the current model and saves the state
func (s *State) UpdateModel(provider, model string) error {
	s.Provider = provider
	s.Model = model
	
	configPath := GetConfigPath()
	return SaveState(configPath, s)
}
