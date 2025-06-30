package chat

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/config"
	"github.com/entrepeneur4lyf/codeforge/internal/events"
	"github.com/entrepeneur4lyf/codeforge/internal/llm"
	"github.com/entrepeneur4lyf/codeforge/internal/llm/agent"
	"github.com/entrepeneur4lyf/codeforge/internal/llm/providers"
)

// GetAPIKeyForModel returns the appropriate API key for the given model
func GetAPIKeyForModel(model string) string {
	// Determine provider from model and get corresponding API key

	// OpenRouter models FIRST (most important - supports 300+ models with provider/ format)
	if strings.Contains(model, "/") {
		// OpenRouter uses provider/model format, so any model with "/" is likely OpenRouter
		if key := os.Getenv("OPENROUTER_API_KEY"); key != "" {
			return key
		}
		// If no OpenRouter key, try to extract provider and use direct provider key
		parts := strings.Split(model, "/")
		if len(parts) == 2 {
			provider := parts[0]
			switch provider {
			case "anthropic":
				return os.Getenv("ANTHROPIC_API_KEY")
			case "openai":
				return os.Getenv("OPENAI_API_KEY")
			case "google":
				return os.Getenv("GEMINI_API_KEY")
			case "groq":
				return os.Getenv("GROQ_API_KEY")
			case "mistralai", "mistral":
				return os.Getenv("MISTRAL_API_KEY")
			case "deepseek":
				return os.Getenv("DEEPSEEK_API_KEY")
			case "cohere":
				return os.Getenv("COHERE_API_KEY")
			}
		}
	}

	// Direct provider models (without provider/ prefix)
	// Anthropic models
	if strings.HasPrefix(model, "claude-") || strings.HasPrefix(model, "anthropic.") {
		return os.Getenv("ANTHROPIC_API_KEY")
	}

	// OpenAI models
	if strings.HasPrefix(model, "gpt-") || strings.HasPrefix(model, "o1-") || strings.HasPrefix(model, "chatgpt-") {
		return os.Getenv("OPENAI_API_KEY")
	}

	// Google models
	if strings.HasPrefix(model, "gemini-") {
		return os.Getenv("GEMINI_API_KEY")
	}

	// Groq models
	if strings.HasPrefix(model, "llama") && strings.Contains(model, "versatile") {
		return os.Getenv("GROQ_API_KEY")
	}

	// Additional provider-specific keys
	if strings.Contains(model, "together/") || strings.Contains(model, "togetherai/") {
		return os.Getenv("TOGETHER_API_KEY")
	}
	if strings.Contains(model, "fireworks/") {
		return os.Getenv("FIREWORKS_API_KEY")
	}
	if strings.Contains(model, "deepseek/") {
		return os.Getenv("DEEPSEEK_API_KEY")
	}
	if strings.Contains(model, "cohere/") {
		return os.Getenv("COHERE_API_KEY")
	}
	if strings.Contains(model, "mistral/") {
		return os.Getenv("MISTRAL_API_KEY")
	}
	if strings.Contains(model, "perplexity/") {
		return os.Getenv("PERPLEXITY_API_KEY")
	}
	if strings.Contains(model, "cerebras/") {
		return os.Getenv("CEREBRAS_API_KEY")
	}
	if strings.Contains(model, "sambanova/") {
		return os.Getenv("SAMBANOVA_API_KEY")
	}

	// Priority fallback order (most versatile first)
	if key := os.Getenv("OPENROUTER_API_KEY"); key != "" {
		return key
	}
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		return key
	}
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		return key
	}
	if key := os.Getenv("GEMINI_API_KEY"); key != "" {
		return key
	}

	return ""
}

// detectProviderFromModel detects the provider from the model ID
func detectProviderFromModel(model string) string {
	// OpenRouter models have provider/model format
	if strings.Contains(model, "/") {
		return "openrouter"
	}

	// Direct provider models
	if strings.HasPrefix(model, "claude-") || strings.HasPrefix(model, "anthropic.") {
		return "anthropic"
	}
	if strings.HasPrefix(model, "gpt-") || strings.HasPrefix(model, "o1-") || strings.HasPrefix(model, "chatgpt-") {
		return "openai"
	}
	if strings.HasPrefix(model, "gemini-") {
		return "gemini"
	}
	if strings.Contains(model, "llama") && strings.Contains(model, "versatile") {
		return "groq"
	}
	if strings.HasPrefix(model, "mistral-") {
		return "mistral"
	}
	if strings.HasPrefix(model, "deepseek-") {
		return "deepseek"
	}
	if strings.HasPrefix(model, "command-") {
		return "cohere"
	}

	return ""
}

// GetDefaultModel returns a default model if none specified
func GetDefaultModel() string {
	// Try to find a model based on available API keys (priority order)

	// OpenRouter is most versatile (300+ models)
	if os.Getenv("OPENROUTER_API_KEY") != "" {
		return "anthropic/claude-3.5-sonnet"
	}

	// Direct provider keys
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		return "claude-3-5-sonnet-20241022"
	}
	if os.Getenv("OPENAI_API_KEY") != "" {
		return "gpt-4o"
	}
	if os.Getenv("GEMINI_API_KEY") != "" {
		return "gemini-1.5-pro"
	}
	if os.Getenv("GROQ_API_KEY") != "" {
		return "groq/llama-3.1-70b-versatile"
	}

	// Additional providers
	if os.Getenv("TOGETHER_API_KEY") != "" {
		return "together/meta-llama/Llama-3-70b-chat-hf"
	}
	if os.Getenv("FIREWORKS_API_KEY") != "" {
		return "fireworks/llama-v3p1-70b-instruct"
	}
	if os.Getenv("DEEPSEEK_API_KEY") != "" {
		return "deepseek/deepseek-chat"
	}
	if os.Getenv("COHERE_API_KEY") != "" {
		return "cohere/command-r-plus"
	}

	// Default to Claude (user will get error if no API key)
	return "claude-3-5-sonnet-20241022"
}

// ChatSession represents an interactive chat session
type ChatSession struct {
	handler         llm.ApiHandler
	messages        []llm.Message
	systemPrompt    string
	quiet           bool
	model           string
	format          string
	commandRouter   *CommandRouter
	favorites       *Favorites
	contextGathered bool   // Track if context has been gathered for this session
	sessionContext  string // Store the gathered context for the session

	// Agent integration
	agentService agent.Service
	eventManager *events.Manager
	sessionID    string
	currentAgent config.AgentName
}

// NewChatSession creates a new chat session with the specified configuration
func NewChatSession(model, apiKey, provider string, quiet bool, format string) (*ChatSession, error) {
	// Create handler options
	options := llm.ApiHandlerOptions{
		APIKey:  apiKey,
		ModelID: model,
	}

	// Detect provider from model if not explicitly specified
	if provider == "" {
		provider = detectProviderFromModel(model)
	}

	// Set provider-specific options
	if provider != "" {
		setProviderSpecificOptions(&options, provider, apiKey)
	}

	// Build the appropriate handler
	handler, err := providers.BuildApiHandler(options)
	if err != nil {
		return nil, fmt.Errorf("failed to create LLM handler: %w", err)
	}

	// Set up system prompt for CodeForge
	systemPrompt := `You are CodeForge, an AI coding assistant. You help developers with:

- Building and fixing projects
- Analyzing and debugging code
- Searching for code patterns and solutions
- Explaining code and providing documentation
- Code reviews and optimization suggestions

When users ask you to:
- "build" or "compile" - help with build systems and compilation
- "search" or "find" - help locate code patterns or solutions
- "fix" or "debug" - analyze and fix code issues
- "explain" or "document" - provide clear explanations

Be concise, practical, and focus on actionable solutions. Provide code examples when helpful.`

	// Get current working directory for command router
	workingDir, err := os.Getwd()
	if err != nil {
		workingDir = "."
	}

	// Initialize favorites
	favorites, err := NewFavorites()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize favorites: %w", err)
	}

	return &ChatSession{
		handler:       handler,
		messages:      []llm.Message{},
		systemPrompt:  systemPrompt,
		quiet:         quiet,
		model:         model,
		format:        format,
		commandRouter: NewCommandRouter(workingDir),
		favorites:     favorites,
	}, nil
}

// StartInteractive starts an interactive chat session
func (cs *ChatSession) StartInteractive() error {
	if !cs.quiet {
		fmt.Println("🏗️ CodeForge Interactive Chat")
		fmt.Printf("Model: %s\n", cs.model)
		fmt.Println("Type 'exit', 'quit', or press Ctrl+C to end the session")
		fmt.Println("Type '/help' for available commands")
		fmt.Println()
	}

	scanner := bufio.NewScanner(os.Stdin)

	for {
		// Show prompt
		if !cs.quiet {
			fmt.Print("> ")
		}

		// Read user input
		if !scanner.Scan() {
			break // EOF or error
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		// Handle special commands
		if strings.HasPrefix(input, "/") {
			if cs.handleCommand(input) {
				break // Exit command
			}
			continue
		}

		// Handle exit commands
		if input == "exit" || input == "quit" {
			if !cs.quiet {
				fmt.Println("Goodbye!")
			}
			break
		}

		// Process the message
		response, err := cs.ProcessMessage(input)
		if err != nil {
			if cs.quiet {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Printf("❌ Error: %v\n", err)
			}
			continue
		}

		// Display response
		cs.displayResponse(response)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	return nil
}

// SetAgentService sets the agent service for this chat session
func (cs *ChatSession) SetAgentService(agentService agent.Service, eventManager *events.Manager, sessionID string) {
	cs.agentService = agentService
	cs.eventManager = eventManager
	cs.sessionID = sessionID
	cs.currentAgent = config.AgentCoder // Default to coder agent
}

// ProcessMessageWithAgent processes a message using the agent system
func (cs *ChatSession) ProcessMessageWithAgent(userInput string) (string, error) {
	// First, check if this is a direct command (build, file operations)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if commandResponse, handled := cs.commandRouter.RouteDirectCommand(ctx, userInput); handled {
		// Direct command was handled, return the response
		return commandResponse, nil
	}

	// Use agent service if available
	if cs.agentService != nil {
		return cs.processWithAgent(ctx, userInput)
	}

	// Fallback to original processing
	return cs.ProcessMessage(userInput)
}

// processWithAgent processes the message using the agent service
func (cs *ChatSession) processWithAgent(ctx context.Context, userInput string) (string, error) {
	// Gather context only once per session
	if !cs.contextGathered {
		cs.sessionContext = cs.commandRouter.GatherContext(ctx, userInput)
		cs.contextGathered = true
	}

	// Prepare content blocks
	var attachments []llm.ContentBlock
	if cs.sessionContext != "" {
		attachments = append(attachments, llm.TextBlock{
			Text: "\n\n**Relevant Context:**\n" + cs.sessionContext,
		})
	}

	// Run agent
	eventChan, err := cs.agentService.Run(ctx, cs.sessionID, cs.currentAgent, userInput, attachments...)
	if err != nil {
		return "", fmt.Errorf("failed to run agent: %w", err)
	}

	// Process agent events
	var fullResponse strings.Builder
	var usage *llm.Usage

	for event := range eventChan {
		switch event.Type {
		case agent.AgentEventTextChunk:
			if text, ok := event.Data["text"].(string); ok {
				fullResponse.WriteString(text)
				if !cs.quiet {
					fmt.Print(text)
				}
			}

		case agent.AgentEventUsage:
			if inputTokens, ok := event.Data["input_tokens"].(int); ok {
				if outputTokens, ok := event.Data["output_tokens"].(int); ok {
					if totalCost, ok := event.Data["total_cost"].(float64); ok {
						usage = &llm.Usage{
							PromptTokens:     inputTokens,
							CompletionTokens: outputTokens,
							TotalTokens:      inputTokens + outputTokens,
							TotalCost:        totalCost,
						}
					}
				}
			}

		case agent.AgentEventError:
			if errorMsg, ok := event.Data["error"].(string); ok {
				return "", fmt.Errorf("agent error: %s", errorMsg)
			}

		case agent.AgentEventCompleted:
			// Agent completed successfully
			break
		}
	}

	response := fullResponse.String()

	// Add messages to conversation history
	userMessage := llm.Message{
		Role: "user",
		Content: []llm.ContentBlock{
			llm.TextBlock{Text: userInput},
		},
	}
	cs.messages = append(cs.messages, userMessage)

	assistantMessage := llm.Message{
		Role: "assistant",
		Content: []llm.ContentBlock{
			llm.TextBlock{Text: response},
		},
	}
	cs.messages = append(cs.messages, assistantMessage)

	// Show usage info in non-quiet mode
	if !cs.quiet && usage != nil {
		fmt.Printf("\n\n💡 Tokens: %d input, %d output", usage.PromptTokens, usage.CompletionTokens)
		if usage.TotalCost > 0 {
			fmt.Printf(" | Cost: $%.4f", usage.TotalCost)
		}
		fmt.Println()
	}

	return response, nil
}

// SetCurrentAgent sets the current agent for this session
func (cs *ChatSession) SetCurrentAgent(agentName config.AgentName) {
	cs.currentAgent = agentName
}

// ProcessMessage processes a single message and returns the AI response
func (cs *ChatSession) ProcessMessage(userInput string) (string, error) {
	// First, check if this is a direct command (build, file operations)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if commandResponse, handled := cs.commandRouter.RouteDirectCommand(ctx, userInput); handled {
		// Direct command was handled, return the response
		return commandResponse, nil
	}

	// Gather context only once per session
	if !cs.contextGathered {
		cs.sessionContext = cs.commandRouter.GatherContext(ctx, userInput)
		cs.contextGathered = true
	}

	// Enhance the user input with session context
	enhancedPrompt := userInput
	if cs.sessionContext != "" {
		enhancedPrompt = userInput + "\n\n**Relevant Context:**\n" + cs.sessionContext
	}

	// Add user message to conversation
	userMessage := llm.Message{
		Role: "user",
		Content: []llm.ContentBlock{
			llm.TextBlock{Text: enhancedPrompt},
		},
	}
	cs.messages = append(cs.messages, userMessage)

	// Update context timeout for LLM call
	ctx, cancel = context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Send message to LLM
	stream, err := cs.handler.CreateMessage(ctx, cs.systemPrompt, cs.messages)
	if err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}

	// Collect response
	var responseText strings.Builder
	var usage *llm.Usage

	for chunk := range stream {
		switch c := chunk.(type) {
		case llm.ApiStreamTextChunk:
			responseText.WriteString(c.Text)
			// Show streaming response in real-time for interactive mode
			if !cs.quiet {
				fmt.Print(c.Text)
			}
		case llm.ApiStreamUsageChunk:
			usage = &llm.Usage{
				PromptTokens:     c.InputTokens,
				CompletionTokens: c.OutputTokens,
				TotalTokens:      c.InputTokens + c.OutputTokens,
			}
			if c.TotalCost != nil {
				usage.TotalCost = *c.TotalCost
			}
		case llm.ApiStreamReasoningChunk:
			// Handle reasoning/thinking chunks (for models that support it)
			if !cs.quiet {
				fmt.Printf("\n[Thinking: %s]\n", c.Reasoning)
			}
		}
	}

	response := responseText.String()

	// Add assistant response to conversation
	assistantMessage := llm.Message{
		Role: "assistant",
		Content: []llm.ContentBlock{
			llm.TextBlock{Text: response},
		},
	}
	cs.messages = append(cs.messages, assistantMessage)

	// Show usage info in non-quiet mode
	if !cs.quiet && usage != nil {
		fmt.Printf("\n\n💡 Tokens: %d input, %d output", usage.PromptTokens, usage.CompletionTokens)
		if usage.TotalCost > 0 {
			fmt.Printf(" | Cost: $%.4f", usage.TotalCost)
		}
		fmt.Println()
	}

	return response, nil
}

// handleCommand processes special chat commands
func (cs *ChatSession) handleCommand(command string) bool {
	switch command {
	case "/help":
		cs.showHelp()
	case "/clear":
		cs.clearHistory()
	case "/model":
		cs.selectModel()
	case "/favorites":
		cs.showFavorites()
	case "/embedding":
		cs.selectEmbedding()
	case "/history":
		cs.showHistory()
	case "/exit", "/quit":
		if !cs.quiet {
			fmt.Println("Goodbye!")
		}
		return true
	default:
		if !cs.quiet {
			fmt.Printf("Unknown command: %s\nType '/help' for available commands.\n", command)
		}
	}
	return false
}

// showHelp displays available commands
func (cs *ChatSession) showHelp() {
	if cs.quiet {
		return
	}

	fmt.Println("Available commands:")
	fmt.Println("  /help      - Show this help message")
	fmt.Println("  /clear     - Clear conversation history")
	fmt.Println("  /model     - Interactive model selector")
	fmt.Println("  /embedding - Select embedding provider")
	fmt.Println("  /favorites - Show favorite providers and models")
	fmt.Println("  /history   - Show conversation history")
	fmt.Println("  /exit      - Exit the chat session")
	fmt.Println("  exit       - Exit the chat session")
	fmt.Println("  quit       - Exit the chat session")
	fmt.Println()
	fmt.Println("Natural language commands:")
	fmt.Println("  'build' or 'compile' - Build the project")
	fmt.Println("  'search for X' - Semantic code search")
	fmt.Println("  'find definition of X' - LSP-powered symbol lookup")
	fmt.Println("  'find references to X' - Find all symbol references")
	fmt.Println("  'commit' or 'git commit' - AI-powered commit with generated message")
	fmt.Println("  'generate commit message' - Generate commit message without committing")
	fmt.Println("  'commit staged' - Commit only staged changes with AI message")
	fmt.Println()
}

// selectEmbedding allows user to select embedding provider
func (cs *ChatSession) selectEmbedding() {
	if cs.quiet {
		return
	}

	fmt.Println("\n🔍 Embedding Provider Selection")
	fmt.Println("Current: Fallback (simple hash-based)")
	fmt.Println()

	// Check available providers
	ollamaAvailable := isOllamaAvailable()
	openaiAvailable := isOpenAIAvailable()

	fmt.Println("Available providers:")
	fmt.Println("  1. Fallback (current) - Simple hash-based, always works")

	if ollamaAvailable {
		fmt.Println("  2. Ollama - High-quality local embeddings (nomic-embed-text)")
	} else {
		fmt.Println("  2. Ollama - Not available (install with: curl -fsSL https://ollama.ai/install.sh | sh)")
	}

	if openaiAvailable {
		fmt.Println("  3. OpenAI - Premium cloud embeddings (uses API key)")
	} else {
		fmt.Println("  3. OpenAI - Not available (set OPENAI_API_KEY)")
	}

	fmt.Print("\nSelect provider (1-3): ")

	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		choice := strings.TrimSpace(scanner.Text())

		switch choice {
		case "1":
			cs.setEmbeddingProvider("fallback")
		case "2":
			if ollamaAvailable {
				cs.setEmbeddingProvider("ollama")
			} else {
				fmt.Println("❌ Ollama not available. Install with:")
				fmt.Println("   curl -fsSL https://ollama.ai/install.sh | sh")
				fmt.Println("   ollama pull nomic-embed-text")
			}
		case "3":
			if openaiAvailable {
				cs.setEmbeddingProvider("openai")
			} else {
				fmt.Println("❌ OpenAI not available. Set OPENAI_API_KEY environment variable.")
			}
		default:
			fmt.Println("❌ Invalid choice")
		}
	}
}

// setEmbeddingProvider updates the embedding provider configuration
func (cs *ChatSession) setEmbeddingProvider(provider string) {
	// Update config file
	configPath := filepath.Join(os.Getenv("HOME"), ".codeforge")

	// Read existing config
	var config map[string]interface{}
	if data, err := os.ReadFile(configPath); err == nil {
		json.Unmarshal(data, &config)
	} else {
		config = make(map[string]interface{})
	}

	// Update embedding config
	embeddingConfig := map[string]interface{}{
		"provider": provider,
	}

	if provider == "ollama" {
		embeddingConfig["model"] = "nomic-embed-text"
		embeddingConfig["baseURL"] = "http://localhost:11434"
	}

	config["embedding"] = embeddingConfig

	// Write config back
	if data, err := json.MarshalIndent(config, "", "  "); err == nil {
		if err := os.WriteFile(configPath, data, 0644); err == nil {
			fmt.Printf("✅ Embedding provider set to: %s\n", provider)
			fmt.Println("💡 Restart CodeForge for changes to take effect")
		} else {
			fmt.Printf("❌ Failed to save config: %v\n", err)
		}
	} else {
		fmt.Printf("❌ Failed to encode config: %v\n", err)
	}
}

// Helper functions for embedding provider detection
func isOllamaAvailable() bool {
	resp, err := http.Get("http://localhost:11434/api/tags")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	var tags struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return false
	}

	// Check for embedding models
	embeddingModels := []string{"nomic-embed-text", "all-minilm", "mxbai-embed-large"}
	for _, model := range tags.Models {
		for _, embModel := range embeddingModels {
			if strings.Contains(model.Name, embModel) {
				return true
			}
		}
	}

	return false
}

func isOpenAIAvailable() bool {
	return os.Getenv("OPENAI_API_KEY") != ""
}

// setProviderSpecificOptions sets provider-specific fields in ApiHandlerOptions
func setProviderSpecificOptions(options *llm.ApiHandlerOptions, provider, apiKey string) {
	switch strings.ToLower(provider) {
	case "openrouter":
		options.OpenRouterAPIKey = apiKey
	case "anthropic":
		// APIKey is already set, no additional fields needed
	case "openai":
		// APIKey is already set, no additional fields needed
	case "gemini":
		// APIKey is already set, no additional fields needed
	case "github":
		// Set GitHub-specific options if needed
		options.GitHubOrg = "github" // Default org
	case "vertex":
		// Set Vertex-specific options if needed
		// options.VertexProjectID would need to be set by user
	case "xai", "mistral", "deepseek", "groq", "ollama":
		// These providers use the generic APIKey field
		// No additional provider-specific fields needed
	}
}

// clearHistory clears the conversation history
func (cs *ChatSession) clearHistory() {
	cs.messages = []llm.Message{}
	if !cs.quiet {
		fmt.Println("✅ Conversation history cleared")
	}
}

// showModelInfo displays current model information
func (cs *ChatSession) showModelInfo() {
	if cs.quiet {
		return
	}

	modelInfo := cs.handler.GetModel()
	fmt.Printf("Current model: %s\n", modelInfo.ID)
	fmt.Printf("Context window: %d tokens\n", modelInfo.Info.ContextWindow)
	fmt.Printf("Max tokens: %d\n", modelInfo.Info.MaxTokens)
	fmt.Printf("Supports images: %v\n", modelInfo.Info.SupportsImages)
	if modelInfo.Info.InputPrice > 0 {
		fmt.Printf("Input price: $%.2f per million tokens\n", modelInfo.Info.InputPrice)
		fmt.Printf("Output price: $%.2f per million tokens\n", modelInfo.Info.OutputPrice)
	}
}

// showHistory displays the conversation history
func (cs *ChatSession) showHistory() {
	if cs.quiet {
		return
	}

	if len(cs.messages) == 0 {
		fmt.Println("No conversation history")
		return
	}

	fmt.Println("Conversation history:")
	for i, msg := range cs.messages {
		role := strings.ToUpper(msg.Role)
		if len(msg.Content) > 0 {
			if textBlock, ok := msg.Content[0].(llm.TextBlock); ok {
				preview := textBlock.Text
				if len(preview) > 100 {
					preview = preview[:100] + "..."
				}
				fmt.Printf("%d. %s: %s\n", i+1, role, preview)
			}
		}
	}
}

// displayResponse formats and displays the AI response
func (cs *ChatSession) displayResponse(response string) {
	if cs.quiet {
		// In quiet mode, just output the response
		fmt.Println(response)
		return
	}

	// In interactive mode, add formatting
	fmt.Println() // New line after streaming text
}

// selectModel shows the interactive model selector
func (cs *ChatSession) selectModel() {
	if cs.quiet {
		return
	}

	fmt.Println("🤖 Opening model selector...")

	selector := NewModelSelector(cs.favorites)
	provider, model, err := selector.SelectModel()

	if err != nil {
		fmt.Printf("❌ Model selection failed: %v\n", err)
		return
	}

	// Update the current model
	cs.model = model

	// Create new handler with the selected model
	apiKey := GetAPIKeyForModel(model)
	if apiKey == "" {
		fmt.Printf("❌ No API key found for provider: %s\n", provider)
		return
	}

	// Create handler options
	options := llm.ApiHandlerOptions{
		APIKey:  apiKey,
		ModelID: model,
	}

	// Build the new handler
	handler, err := providers.BuildApiHandler(options)
	if err != nil {
		fmt.Printf("❌ Failed to create handler for %s: %v\n", model, err)
		return
	}

	cs.handler = handler
	fmt.Printf("✅ Switched to model: %s\n", model)
}

// showFavorites displays favorite providers and models
func (cs *ChatSession) showFavorites() {
	if cs.quiet {
		return
	}

	favorites := cs.favorites.GetAllFavorites()
	if len(favorites) == 0 {
		fmt.Println("📝 No favorites yet!")
		fmt.Println("💡 Use the /model command and press spacebar to add favorites")
		return
	}

	fmt.Println("⭐ Your Favorites:")
	fmt.Println()

	// Group by type
	providers := []FavoriteItem{}
	models := []FavoriteItem{}

	for _, fav := range favorites {
		if fav.Type == "provider" {
			providers = append(providers, fav)
		} else {
			models = append(models, fav)
		}
	}

	// Show favorite providers
	if len(providers) > 0 {
		fmt.Println("🔌 Favorite Providers:")
		for _, provider := range providers {
			fmt.Printf("  • %s\n", provider.Name)
		}
		fmt.Println()
	}

	// Show favorite models
	if len(models) > 0 {
		fmt.Println("🎯 Favorite Models:")
		for _, model := range models {
			fmt.Printf("  • %s\n", model.Name)
		}
		fmt.Println()
	}

	providerCount, modelCount := cs.favorites.GetStats()
	fmt.Printf("📊 Total: %d providers, %d models\n", providerCount, modelCount)
}
