package cmd

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/app"
	"github.com/entrepeneur4lyf/codeforge/internal/chat"
	"github.com/entrepeneur4lyf/codeforge/internal/embeddings"
	"github.com/entrepeneur4lyf/codeforge/internal/llm"
	"github.com/entrepeneur4lyf/codeforge/internal/llm/providers"
	"github.com/entrepeneur4lyf/codeforge/internal/lsp"
	"github.com/entrepeneur4lyf/codeforge/internal/ml"
	"github.com/entrepeneur4lyf/codeforge/internal/project"
	"github.com/entrepeneur4lyf/codeforge/internal/tui"
	"github.com/spf13/cobra"
)

var (
	debug      bool
	workingDir string
)

var (
	quiet    bool
	model    string
	provider string
	format   string
	tuiMode  bool
	logFile  *os.File // For cleanup
)

// Global app instance for integrated systems
var codeforgeApp *app.App

// setupLogging configures logging to redirect verbose output to a file
func setupLogging(workingDir string, debug bool) error {
	if debug {
		// In debug mode, keep logging to stderr
		return nil
	}

	// Create .codeforge directory if it doesn't exist
	logDir := filepath.Join(workingDir, ".codeforge")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create log file
	logPath := filepath.Join(logDir, "codeforge.log")
	var err error
	logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}

	// Redirect log output to file
	log.SetOutput(logFile)

	return nil
}

// cleanupLogging closes the log file if it was opened
func cleanupLogging() {
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
}

// autoGenerateProjectOverview automatically analyzes existing projects
func autoGenerateProjectOverview() {
	if codeforgeApp == nil {
		return
	}

	agentMdPath := filepath.Join(workingDir, "AGENTS.md")

	// Create project service directly instead of using CLI command infrastructure
	projectService := project.NewService(codeforgeApp.Config, workingDir, codeforgeApp.FileOperationManager)

	if _, err := os.Stat(agentMdPath); os.IsNotExist(err) {
		// AGENTS.md doesn't exist - analyze project and create overview
		analyzeProjectDirectly(projectService)
	} else {
		// AGENTS.md exists - read it, analyze current project, and update
		updateProjectDirectly(projectService)
	}
}

// animatedSpinner shows an animated spinner with message and dots
func animatedSpinner(message string, done chan bool) {
	spinnerFrames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	frameIndex := 0
	dotCount := 0

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			fmt.Print("\r\033[K") // Clear line
			return
		case <-ticker.C:
			dots := ""
			for i := 0; i < dotCount; i++ {
				dots += "."
			}

			fmt.Printf("\r%s %s%s", spinnerFrames[frameIndex], message, dots)

			frameIndex = (frameIndex + 1) % len(spinnerFrames)
			if frameIndex == 0 {
				dotCount = (dotCount + 1) % 4 // 0, 1, 2, 3 dots
			}
		}
	}
}

// analyzeProjectDirectly performs project analysis without CLI command infrastructure
func analyzeProjectDirectly(projectService *project.Service) {
	// Start animated spinner
	done := make(chan bool)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		animatedSpinner("Analyzing project... one moment", done)
	}()

	if !projectService.HasExistingProject() {
		// No real project found - skip analysis
		done <- true
		wg.Wait()
		return
	}

	// Analyze the project
	overview, err := projectService.AnalyzeExistingProject()
	if err != nil {
		done <- true
		wg.Wait()
		fmt.Printf("Project analysis failed: %v\n", err)
		return
	}

	// Create PRD files
	if err := projectService.CreatePRDFiles(overview); err != nil {
		done <- true
		wg.Wait()
		fmt.Printf("Failed to save project files: %v\n", err)
		return
	}

	// Stop spinner
	done <- true
	wg.Wait()

	// Analysis complete - no output needed
}

// updateProjectDirectly updates existing project documentation without CLI command infrastructure
func updateProjectDirectly(projectService *project.Service) {
	// Start animated spinner
	done := make(chan bool)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		animatedSpinner("Analyzing project... one moment", done)
	}()

	// Read existing AGENTS.md content
	agentMdPath := filepath.Join(workingDir, "AGENTS.md")
	contentBytes, err := os.ReadFile(agentMdPath)
	if err != nil {
		done <- true
		wg.Wait()
		fmt.Printf("Failed to read existing AGENTS.md: %v\n", err)
		return
	}
	existingContent := string(contentBytes)

	// Analyze current project
	overview, err := projectService.AnalyzeExistingProject()
	if err != nil {
		done <- true
		wg.Wait()
		fmt.Printf("Project analysis failed: %v\n", err)
		return
	}

	// Update AGENTS.md with current analysis
	updatedContent := projectService.UpdateProjectSummary(overview, existingContent)
	agentMdPath = filepath.Join(workingDir, "AGENTS.md")
	if err := os.WriteFile(agentMdPath, []byte(updatedContent), 0644); err != nil {
		done <- true
		wg.Wait()
		fmt.Printf("Failed to update AGENTS.md: %v\n", err)
		return
	}

	// Stop spinner
	done <- true
	wg.Wait()

	// Analysis complete - no output needed
}

var rootCmd = &cobra.Command{
	Use:   "codeforge [prompt]",
	Short: "AI-powered coding assistant",
	Long: `CodeForge is an AI coding assistant that helps with development tasks.

Usage:
  codeforge                    # Start interactive chat
  codeforge "your question"    # Get direct answer
  echo "question" | codeforge  # Pipe input

Features:
- 25+ LLM providers (OpenRouter, Anthropic, OpenAI, Google, Groq, and more)
- 300+ models with smart database caching
- Build and fix projects automatically
- Semantic code search and analysis
- LSP integration for code intelligence
- MCP tool integration`,
	DisableAutoGenTag: true,
	SilenceUsage:      true,
	SilenceErrors:     false,
	Args:              cobra.ArbitraryArgs, // Accept any arguments
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Setup logging to file (unless in debug mode)
		if err := setupLogging(workingDir, debug); err != nil {
			return fmt.Errorf("failed to setup logging: %w", err)
		}

		// Initialize CodeForge application with all integrated systems
		appConfig := &app.AppConfig{
			WorkspaceRoot:     workingDir,
			EnablePermissions: true,
			EnableContextMgmt: true,
			Debug:             debug,
		}

		ctx := context.Background()
		var err error
		codeforgeApp, err = app.NewApp(ctx, appConfig)
		if err != nil {
			return fmt.Errorf("failed to initialize CodeForge app: %w", err)
		}

		// Initialize LLM manager
		if err := llm.Initialize(codeforgeApp.Config); err != nil {
			return fmt.Errorf("failed to initialize LLM providers: %w", err)
		}

		// Start background model fetching for all providers
		providers.InitializeBackgroundFetching()

		// Initialize embedding service
		if err := embeddings.Initialize(codeforgeApp.Config); err != nil {
			return fmt.Errorf("failed to initialize embedding service: %w", err)
		}

		// Initialize LSP manager
		if err := lsp.Initialize(codeforgeApp.Config); err != nil {
			return fmt.Errorf("failed to initialize LSP clients: %w", err)
		}

		// Initialize ML service silently for model context (graceful degradation if it fails)
		ml.Initialize(codeforgeApp.Config) // Ignore errors - ML is for model context only

		// Auto-analyze existing projects (new projects handled by model tool)
		// Show a spinner/progress during analysis. Skip only when reading from stdin (piped input).
		if !hasStdinInput() {
			autoGenerateProjectOverview()
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default to TUI when no args and no piped input
		startTUI := tuiMode || (len(args) == 0 && !hasStdinInput())
		if startTUI {
			if err := tui.Run(codeforgeApp); err != nil {
				return fmt.Errorf("error running TUI: %w", err)
			}
			return nil
		}

		// Handle different input modes like Gemini CLI
		if len(args) > 0 {
			// Direct prompt mode: codeforge "question"
			prompt := strings.Join(args, " ")
			return handleDirectPrompt(prompt)
		} else {
			// Check for piped input
			if hasStdinInput() {
				return handlePipedInput()
			} else {
				// Interactive mode (default)
				return startInteractiveMode()
			}
		}
	},
}

func init() {
	// Get current working directory
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}

	// Add flags for the new CLI pattern
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug mode")
	rootCmd.PersistentFlags().StringVar(&workingDir, "wd", wd, "Working directory")
	rootCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Quiet mode - output only the answer")
	rootCmd.Flags().StringVarP(&model, "model", "m", "", "Specify the model to use")
	rootCmd.Flags().StringVarP(&provider, "provider", "p", "", "Specify the provider (anthropic, openai, openrouter, etc.)")
	rootCmd.Flags().StringVar(&format, "format", "text", "Output format (text, json, markdown)")
	rootCmd.Flags().BoolVar(&tuiMode, "tui", false, "Start in TUI (Terminal User Interface) mode")

}

func Execute() {
	// Setup cleanup on exit
	defer func() {
		if codeforgeApp != nil {
			codeforgeApp.Close()
		}
		cleanupLogging()
	}()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// handleDirectPrompt processes a direct prompt with integrated CodeForge app
func handleDirectPrompt(prompt string) error {
	// Use integrated app if available
	if codeforgeApp != nil {
		ctx := context.Background()
		response, err := codeforgeApp.ProcessChatMessage(ctx, "cli-session", prompt, model)
		if err != nil {
			if quiet {
				fmt.Fprintln(os.Stderr, "Error:", err)
			} else {
				fmt.Fprintln(os.Stderr, "Error processing message:", err)
			}
			return err
		}

		if quiet {
			fmt.Println(response)
		} else {
			fmt.Printf("%s\n", response)
		}
		return nil
	}

	// Fallback to original LLM integration
	// Determine model to use
	selectedModel := model
	if selectedModel == "" {
		selectedModel = chat.GetDefaultModel()
	}

	// Get API key for the model
	apiKey := chat.GetAPIKeyForModel(selectedModel)
	if apiKey == "" {
		if quiet {
			fmt.Fprintln(os.Stderr, "Error: No API key found. Set one of the supported provider API keys.")
		} else {
			fmt.Fprintln(os.Stderr, "Error: No API key found")
			fmt.Fprintln(os.Stderr, "Please set one of these environment variables:")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "🌐 Multi-Provider Platforms:")
			fmt.Fprintln(os.Stderr, "  - OPENROUTER_API_KEY (300+ models from 50+ providers)")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "🏢 Direct Provider Keys:")
			fmt.Fprintln(os.Stderr, "  - ANTHROPIC_API_KEY (Claude models)")
			fmt.Fprintln(os.Stderr, "  - OPENAI_API_KEY (GPT models)")
			fmt.Fprintln(os.Stderr, "  - GEMINI_API_KEY (Gemini models)")
			fmt.Fprintln(os.Stderr, "  - GROQ_API_KEY (ultra-fast inference)")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "⚡ Additional Providers:")
			fmt.Fprintln(os.Stderr, "  - TOGETHER_API_KEY (Together AI)")
			fmt.Fprintln(os.Stderr, "  - FIREWORKS_API_KEY (Fireworks AI)")
			fmt.Fprintln(os.Stderr, "  - DEEPSEEK_API_KEY (DeepSeek)")
			fmt.Fprintln(os.Stderr, "  - COHERE_API_KEY (Cohere)")
			fmt.Fprintln(os.Stderr, "  - MISTRAL_API_KEY (Mistral AI)")
			fmt.Fprintln(os.Stderr, "  - PERPLEXITY_API_KEY (Perplexity)")
			fmt.Fprintln(os.Stderr, "  - CEREBRAS_API_KEY (Cerebras)")
			fmt.Fprintln(os.Stderr, "  - SAMBANOVA_API_KEY (SambaNova)")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Tip: OPENROUTER_API_KEY gives you access to the most models!")
		}
		return fmt.Errorf("no API key found")
	}

	// Create chat session
	session, err := chat.NewChatSession(selectedModel, apiKey, provider, quiet, format)
	if err != nil {
		if quiet {
			fmt.Fprintln(os.Stderr, "Error:", err)
		} else {
			fmt.Fprintln(os.Stderr, "Error creating chat session:", err)
		}
		return err
	}

	// Process the message
	response, err := session.ProcessMessage(prompt)
	if err != nil {
		if quiet {
			fmt.Fprintln(os.Stderr, "Error:", err)
		} else {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
		return err
	}

	// In quiet mode, response is already printed during streaming
	// In non-quiet mode, we need to print it since streaming was shown
	if quiet {
		// Response was not streamed, so print it now
		fmt.Println(response)
	}
	return nil
}

func hasStdinInput() bool {
	// Check if stdin is not a terminal (pipe or redirect)
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}

	// If stdin is not a character device, it's piped or redirected
	return (stat.Mode() & os.ModeCharDevice) == 0
}

func handlePipedInput() error {
	fmt.Println("Reading from stdin...")

	// Read all input from stdin
	scanner := bufio.NewScanner(os.Stdin)
	var lines []string

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Error reading stdin:", err)
		return err
	}

	if len(lines) == 0 {
		fmt.Fprintln(os.Stderr, "No input received from stdin")
		return fmt.Errorf("no input received from stdin")
	}

	// Join all lines into a single prompt
	prompt := strings.Join(lines, "\n")

	// Handle as direct prompt
	return handleDirectPrompt(prompt)
}

func startInteractiveMode() error {
	// Determine model to use
	selectedModel := model
	if selectedModel == "" {
		selectedModel = chat.GetDefaultModel()
	}

	// Get API key for the model
	apiKey := chat.GetAPIKeyForModel(selectedModel)
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "Error: No API key found")
		fmt.Fprintln(os.Stderr, "Please set one of these environment variables:")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "🌐 Multi-Provider Platforms:")
		fmt.Fprintln(os.Stderr, "  - OPENROUTER_API_KEY (300+ models from 50+ providers)")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "🏢 Direct Provider Keys:")
		fmt.Fprintln(os.Stderr, "  - ANTHROPIC_API_KEY (Claude models)")
		fmt.Fprintln(os.Stderr, "  - OPENAI_API_KEY (GPT models)")
		fmt.Fprintln(os.Stderr, "  - GEMINI_API_KEY (Gemini models)")
		fmt.Fprintln(os.Stderr, "  - GROQ_API_KEY (ultra-fast inference)")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "⚡ Additional Providers:")
		fmt.Fprintln(os.Stderr, "  - TOGETHER_API_KEY, FIREWORKS_API_KEY, DEEPSEEK_API_KEY")
		fmt.Fprintln(os.Stderr, "  - COHERE_API_KEY, MISTRAL_API_KEY, PERPLEXITY_API_KEY")
		fmt.Fprintln(os.Stderr, "  - CEREBRAS_API_KEY, SAMBANOVA_API_KEY")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Tip: OPENROUTER_API_KEY gives you access to the most models!")
		return fmt.Errorf("no API key found")
	}

	// Create chat session
	session, err := chat.NewChatSession(selectedModel, apiKey, provider, quiet, format)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating chat session:", err)
		return err
	}

	// Start interactive chat
	if err := session.StartInteractive(); err != nil {
		fmt.Fprintln(os.Stderr, "Error in interactive mode:", err)
		return err
	}
	return nil
}

// init sets up signal handling for graceful shutdown
func init() {
	// Set up signal handling for graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Println("\nShutting down gracefully...")

		// Shutdown ML service
		ml.Shutdown()

		// Cleanup logging
		cleanupLogging()

		os.Exit(0)
	}()
}
