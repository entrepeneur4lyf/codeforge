package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

// EnvironmentVariable represents an environment variable
type EnvironmentVariable struct {
	Name        string `json:"name"`
	Value       string `json:"value"`
	Masked      bool   `json:"masked"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Category    string `json:"category"`
}

// EnvironmentUpdateRequest represents an environment variable update
type EnvironmentUpdateRequest struct {
	Variables map[string]string `json:"variables"`
}

// handleEnvironment handles GET /environment
func (s *Server) handleEnvironment(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s.getEnvironmentVariables(w, r)
	case "PUT":
		s.updateEnvironmentVariables(w, r)
	default:
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getEnvironmentVariables returns all relevant environment variables
func (s *Server) getEnvironmentVariables(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category") // "llm", "embedding", "database", "all"

	variables := []EnvironmentVariable{}

	// LLM Provider API Keys
	if category == "" || category == "all" || category == "llm" {
		llmVars := []EnvironmentVariable{
			{
				Name:        "ANTHROPIC_API_KEY",
				Value:       maskAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
				Masked:      true,
				Description: "API key for Anthropic Claude models",
				Required:    false,
				Category:    "llm",
			},
			{
				Name:        "OPENAI_API_KEY",
				Value:       maskAPIKey(os.Getenv("OPENAI_API_KEY")),
				Masked:      true,
				Description: "API key for OpenAI GPT models",
				Required:    false,
				Category:    "llm",
			},
			{
				Name:        "OPENROUTER_API_KEY",
				Value:       maskAPIKey(os.Getenv("OPENROUTER_API_KEY")),
				Masked:      true,
				Description: "API key for OpenRouter (300+ models)",
				Required:    false,
				Category:    "llm",
			},
			{
				Name:        "GEMINI_API_KEY",
				Value:       maskAPIKey(os.Getenv("GEMINI_API_KEY")),
				Masked:      true,
				Description: "API key for Google Gemini models",
				Required:    false,
				Category:    "llm",
			},
			{
				Name:        "GROQ_API_KEY",
				Value:       maskAPIKey(os.Getenv("GROQ_API_KEY")),
				Masked:      true,
				Description: "API key for Groq ultra-fast inference",
				Required:    false,
				Category:    "llm",
			},
		}
		variables = append(variables, llmVars...)
	}

	// Additional LLM Provider Keys
	if category == "" || category == "all" || category == "llm" {
		additionalLLMVars := []EnvironmentVariable{
			{
				Name:        "TOGETHER_API_KEY",
				Value:       maskAPIKey(os.Getenv("TOGETHER_API_KEY")),
				Masked:      true,
				Description: "API key for Together AI",
				Required:    false,
				Category:    "llm",
			},
			{
				Name:        "FIREWORKS_API_KEY",
				Value:       maskAPIKey(os.Getenv("FIREWORKS_API_KEY")),
				Masked:      true,
				Description: "API key for Fireworks AI",
				Required:    false,
				Category:    "llm",
			},
			{
				Name:        "DEEPSEEK_API_KEY",
				Value:       maskAPIKey(os.Getenv("DEEPSEEK_API_KEY")),
				Masked:      true,
				Description: "API key for DeepSeek",
				Required:    false,
				Category:    "llm",
			},
			{
				Name:        "COHERE_API_KEY",
				Value:       maskAPIKey(os.Getenv("COHERE_API_KEY")),
				Masked:      true,
				Description: "API key for Cohere",
				Required:    false,
				Category:    "llm",
			},
			{
				Name:        "MISTRAL_API_KEY",
				Value:       maskAPIKey(os.Getenv("MISTRAL_API_KEY")),
				Masked:      true,
				Description: "API key for Mistral AI",
				Required:    false,
				Category:    "llm",
			},
		}
		variables = append(variables, additionalLLMVars...)
	}

	// Database Configuration
	if category == "" || category == "all" || category == "database" {
		dbVars := []EnvironmentVariable{
			{
				Name:        "CODEFORGE_DB_PATH",
				Value:       os.Getenv("CODEFORGE_DB_PATH"),
				Masked:      false,
				Description: "Path to CodeForge database file",
				Required:    false,
				Category:    "database",
			},
			{
				Name:        "CODEFORGE_DATA_DIR",
				Value:       os.Getenv("CODEFORGE_DATA_DIR"),
				Masked:      false,
				Description: "Directory for CodeForge data files",
				Required:    false,
				Category:    "database",
			},
		}
		variables = append(variables, dbVars...)
	}

	// Embedding Configuration
	if category == "" || category == "all" || category == "embedding" {
		embeddingVars := []EnvironmentVariable{
			{
				Name:        "CODEFORGE_EMBEDDING_PROVIDER",
				Value:       os.Getenv("CODEFORGE_EMBEDDING_PROVIDER"),
				Masked:      false,
				Description: "Default embedding provider (ollama, openai, fallback)",
				Required:    false,
				Category:    "embedding",
			},
			{
				Name:        "OLLAMA_ENDPOINT",
				Value:       getEnvWithDefault("OLLAMA_ENDPOINT", "http://localhost:11434"),
				Masked:      false,
				Description: "Ollama server endpoint",
				Required:    false,
				Category:    "embedding",
			},
		}
		variables = append(variables, embeddingVars...)
	}

	// Development Configuration
	if category == "" || category == "all" || category == "development" {
		devVars := []EnvironmentVariable{
			{
				Name:        "CODEFORGE_DEBUG",
				Value:       os.Getenv("CODEFORGE_DEBUG"),
				Masked:      false,
				Description: "Enable debug mode (true/false)",
				Required:    false,
				Category:    "development",
			},
			{
				Name:        "CODEFORGE_LOG_LEVEL",
				Value:       getEnvWithDefault("CODEFORGE_LOG_LEVEL", "info"),
				Masked:      false,
				Description: "Log level (debug, info, warn, error)",
				Required:    false,
				Category:    "development",
			},
		}
		variables = append(variables, devVars...)
	}

	response := map[string]interface{}{
		"variables": variables,
		"total":     len(variables),
		"category":  category,
		"note":      "Masked values show only first/last 4 characters for security",
	}

	s.writeJSON(w, response)
}

// updateEnvironmentVariables updates environment variables
func (s *Server) updateEnvironmentVariables(w http.ResponseWriter, r *http.Request) {
	var req EnvironmentUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	updated := []string{}
	errors := []string{}

	for name, value := range req.Variables {
		// Validate environment variable name
		if !isValidEnvVarName(name) {
			errors = append(errors, "Invalid environment variable name: "+name)
			continue
		}

		// Set environment variable
		if err := os.Setenv(name, value); err != nil {
			errors = append(errors, "Failed to set "+name+": "+err.Error())
			continue
		}

		updated = append(updated, name)
	}

	response := map[string]interface{}{
		"success": len(errors) == 0,
		"updated": updated,
		"errors":  errors,
		"message": "Environment variables updated",
		"note":    "Changes apply to current session only. For persistence, update your shell profile.",
	}

	if len(errors) > 0 {
		w.WriteHeader(http.StatusPartialContent)
	}

	s.writeJSON(w, response)
}

// handleEnvironmentVariable handles GET/PUT/DELETE /environment/{name}
func (s *Server) handleEnvironmentVariable(w http.ResponseWriter, r *http.Request) {
	// Extract variable name from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/environment/")
	varName := strings.Split(path, "/")[0]

	// Validate variable name for security
	if !isValidEnvVarName(varName) {
		s.writeError(w, "Invalid environment variable name", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		s.getEnvironmentVariable(w, r, varName)
	case "PUT":
		s.setEnvironmentVariable(w, r, varName)
	case "DELETE":
		s.deleteEnvironmentVariable(w, r, varName)
	default:
		s.writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// getEnvironmentVariable returns a specific environment variable
func (s *Server) getEnvironmentVariable(w http.ResponseWriter, r *http.Request, varName string) {
	value := os.Getenv(varName)
	masked := shouldMaskVariable(varName)

	variable := EnvironmentVariable{
		Name:        varName,
		Value:       conditionalMask(value, masked),
		Masked:      masked,
		Description: getVariableDescription(varName),
		Required:    isRequiredVariable(varName),
		Category:    getVariableCategory(varName),
	}

	s.writeJSON(w, variable)
}

// setEnvironmentVariable sets a specific environment variable
func (s *Server) setEnvironmentVariable(w http.ResponseWriter, r *http.Request, varName string) {
	var req struct {
		Value string `json:"value"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Store previous value for rollback info
	previousValue := os.Getenv(varName)
	wasSet := previousValue != ""

	// Set the new value
	if err := os.Setenv(varName, req.Value); err != nil {
		s.writeError(w, "Failed to set environment variable: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Determine action type
	action := "updated"
	if !wasSet {
		action = "created"
	}

	response := map[string]interface{}{
		"success":        true,
		"variable":       varName,
		"action":         action,
		"previous_value": conditionalMask(previousValue, shouldMaskVariable(varName)),
		"new_value":      conditionalMask(req.Value, shouldMaskVariable(varName)),
		"message":        fmt.Sprintf("Environment variable %s successfully", action),
		"category":       getVariableCategory(varName),
	}

	s.writeJSON(w, response)
}

// deleteEnvironmentVariable unsets a specific environment variable
func (s *Server) deleteEnvironmentVariable(w http.ResponseWriter, r *http.Request, varName string) {
	// Check if variable exists before deletion
	previousValue := os.Getenv(varName)
	wasSet := previousValue != ""

	if !wasSet {
		s.writeError(w, "Environment variable not found", http.StatusNotFound)
		return
	}

	// Prevent deletion of critical system variables
	if isCriticalVariable(varName) {
		s.writeError(w, "Cannot delete critical system variable", http.StatusForbidden)
		return
	}

	// Unset the variable
	if err := os.Unsetenv(varName); err != nil {
		s.writeError(w, "Failed to unset environment variable: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"success":        true,
		"variable":       varName,
		"action":         "deleted",
		"previous_value": conditionalMask(previousValue, shouldMaskVariable(varName)),
		"message":        "Environment variable removed successfully",
		"category":       getVariableCategory(varName),
		"note":           "Variable removed from current session only. Update shell profile for persistence.",
	}

	s.writeJSON(w, response)
}

// Helper functions
func getEnvWithDefault(name, defaultValue string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return defaultValue
}

func isValidEnvVarName(name string) bool {
	if name == "" {
		return false
	}

	// Check if it's a CodeForge-related variable or known API key
	validPrefixes := []string{
		"CODEFORGE_",
		"ANTHROPIC_",
		"OPENAI_",
		"OPENROUTER_",
		"GEMINI_",
		"GROQ_",
		"TOGETHER_",
		"FIREWORKS_",
		"DEEPSEEK_",
		"COHERE_",
		"MISTRAL_",
		"OLLAMA_",
	}

	for _, prefix := range validPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}

	return false
}

func shouldMaskVariable(name string) bool {
	return strings.Contains(strings.ToUpper(name), "API_KEY") ||
		strings.Contains(strings.ToUpper(name), "SECRET") ||
		strings.Contains(strings.ToUpper(name), "TOKEN")
}

func conditionalMask(value string, shouldMask bool) string {
	if shouldMask {
		return maskAPIKey(value)
	}
	return value
}

func getVariableDescription(name string) string {
	descriptions := map[string]string{
		"ANTHROPIC_API_KEY":            "API key for Anthropic Claude models",
		"OPENAI_API_KEY":               "API key for OpenAI GPT models",
		"OPENROUTER_API_KEY":           "API key for OpenRouter (300+ models)",
		"GEMINI_API_KEY":               "API key for Google Gemini models",
		"GROQ_API_KEY":                 "API key for Groq ultra-fast inference",
		"CODEFORGE_EMBEDDING_PROVIDER": "Default embedding provider",
		"CODEFORGE_DEBUG":              "Enable debug mode",
		"OLLAMA_ENDPOINT":              "Ollama server endpoint",
	}

	if desc, exists := descriptions[name]; exists {
		return desc
	}
	return "CodeForge configuration variable"
}

func isRequiredVariable(name string) bool {
	// No variables are strictly required as CodeForge has fallbacks
	return false
}

func getVariableCategory(name string) string {
	if strings.Contains(name, "API_KEY") {
		return "llm"
	}
	if strings.HasPrefix(name, "CODEFORGE_DB") {
		return "database"
	}
	if strings.Contains(name, "EMBEDDING") || strings.Contains(name, "OLLAMA") {
		return "embedding"
	}
	if strings.Contains(name, "DEBUG") || strings.Contains(name, "LOG") {
		return "development"
	}
	return "general"
}

func isCriticalVariable(name string) bool {
	// Prevent deletion of critical system variables
	criticalVars := []string{
		"PATH", "HOME", "USER", "SHELL", "TERM",
		"PWD", "OLDPWD", "LANG", "LC_ALL",
		"GOPATH", "GOROOT", "GOPROXY",
	}

	for _, critical := range criticalVars {
		if name == critical {
			return true
		}
	}

	return false
}
