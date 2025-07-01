package commands

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/config"
	"github.com/entrepeneur4lyf/codeforge/internal/permissions"
	"github.com/entrepeneur4lyf/codeforge/internal/project"
)

// PRDCommand handles PRD-related operations
type PRDCommand struct {
	projectService *project.Service
	workflow       *project.PRDWorkflow
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

// NewPRDCommand creates a new PRD command
func NewPRDCommand(cfg *config.Config, workspaceRoot string, fileManager *permissions.FileOperationManager) *PRDCommand {
	projectService := project.NewService(cfg, workspaceRoot, fileManager)
	workflow := project.NewPRDWorkflow(projectService)

	return &PRDCommand{
		projectService: projectService,
		workflow:       workflow,
	}
}

// Execute runs the PRD command
func (cmd *PRDCommand) Execute(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return cmd.showHelp()
	}

	switch args[0] {
	case "create":
		return cmd.createPRD(ctx)
	case "analyze":
		return cmd.analyzePRD(ctx)
	case "update":
		return cmd.updatePRD(ctx)
	case "check":
		return cmd.checkPRD(ctx)
	case "help", "--help", "-h":
		return cmd.showHelp()
	default:
		fmt.Printf("Unknown PRD command: %s\n", args[0])
		return cmd.showHelp()
	}
}

// createPRD creates a new PRD through interactive workflow
func (cmd *PRDCommand) createPRD(ctx context.Context) error {
	fmt.Println("Starting PRD creation workflow...")

	overview, err := cmd.workflow.RunInteractivePRDCreation(ctx)
	if err != nil {
		return fmt.Errorf("PRD creation failed: %w", err)
	}

	if overview != nil {
		fmt.Printf("PRD created successfully for project: %s\n", overview.ProjectName)
	}

	return nil
}

// analyzePRD analyzes existing project and creates PRD
func (cmd *PRDCommand) analyzePRD(ctx context.Context) error {
	// Start animated spinner
	done := make(chan bool)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		animatedSpinner("Analyzing project... one moment", done)
	}()

	if !cmd.projectService.HasExistingProject() {
		// No real project found - start PRD workflow instead
		done <- true
		wg.Wait()
		return cmd.createPRD(ctx)
	}

	overview, err := cmd.workflow.QuickPRDFromExisting(ctx)

	// Stop spinner
	done <- true
	wg.Wait()

	if err != nil {
		return fmt.Errorf("project analysis failed: %w", err)
	}

	if overview != nil {
		fmt.Printf("Project analysis complete! Generated: %s\n", overview.ProjectName)
		fmt.Println("\nWhat can I do for you today?")
	}

	return nil
}

// updatePRD reads existing AGENTS.md, analyzes current project, and updates documentation
func (cmd *PRDCommand) updatePRD(ctx context.Context) error {
	// Start animated spinner
	done := make(chan bool)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		animatedSpinner("Analyzing project... one moment", done)
	}()

	overview, err := cmd.workflow.UpdateExistingPRD(ctx)

	// Stop spinner
	done <- true
	wg.Wait()

	if err != nil {
		return fmt.Errorf("project update failed: %w", err)
	}

	if overview != nil {
		fmt.Printf("Project documentation updated: %s\n", overview.ProjectName)
		fmt.Println("\nWhat can I do for you today?")
	}

	return nil
}

// checkPRD checks for existing PRD and offers creation
func (cmd *PRDCommand) checkPRD(ctx context.Context) error {
	hasExisting, content, err := cmd.projectService.CheckForExistingPRD()
	if err != nil {
		return fmt.Errorf("failed to check for existing PRD: %w", err)
	}

	if hasExisting {
		fmt.Println("Existing PRD found:")
		fmt.Println("================")
		// Show first 300 characters
		preview := content
		if len(content) > 300 {
			preview = content[:300] + "..."
		}
		fmt.Println(preview)
		fmt.Println("================")
		return nil
	}

	fmt.Println("No PRD found.")

	// Offer to create one
	overview, err := cmd.workflow.CheckAndOfferPRDCreation(ctx)
	if err != nil {
		return fmt.Errorf("PRD creation workflow failed: %w", err)
	}

	if overview != nil {
		fmt.Printf("PRD created: %s\n", overview.ProjectName)
	} else {
		fmt.Println("ℹ️  PRD creation skipped.")
	}

	return nil
}

// showHelp displays help information for PRD commands
func (cmd *PRDCommand) showHelp() error {
	fmt.Println("CodeForge PRD (Project Requirements Document) Commands")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Println("  codeforge prd <command>")
	fmt.Println()
	fmt.Println("COMMANDS:")
	fmt.Println("  create    Create a new PRD through interactive questions")
	fmt.Println("  analyze   Analyze existing project and generate PRD automatically")
	fmt.Println("  check     Check for existing PRD and offer to create one")
	fmt.Println("  help      Show this help message")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  codeforge prd create     # Interactive PRD creation")
	fmt.Println("  codeforge prd analyze    # Auto-generate from existing code")
	fmt.Println("  codeforge prd check      # Check status and offer creation")
	fmt.Println()
	fmt.Println("FILES CREATED:")
	fmt.Println("  project-overview.md      # Comprehensive project documentation")
	fmt.Println("  AGENTS.md                 # Concise context for AI (auto-included)")
	fmt.Println()
	fmt.Println("The PRD system helps create structured project documentation that")
	fmt.Println("provides context for all AI interactions in CodeForge.")

	return nil
}

// GetName returns the command name
func (cmd *PRDCommand) GetName() string {
	return "prd"
}

// GetDescription returns the command description
func (cmd *PRDCommand) GetDescription() string {
	return "Create and manage Project Requirements Documents (PRD)"
}

// GetUsage returns the command usage
func (cmd *PRDCommand) GetUsage() string {
	return "prd <create|analyze|check|help>"
}

// ValidateArgs validates command arguments
func (cmd *PRDCommand) ValidateArgs(args []string) error {
	if len(args) == 0 {
		return nil // Will show help
	}

	validCommands := map[string]bool{
		"create":  true,
		"analyze": true,
		"update":  true, // Internal use only
		"check":   true,
		"help":    true,
		"--help":  true,
		"-h":      true,
	}

	if !validCommands[args[0]] {
		return fmt.Errorf("invalid PRD command: %s", args[0])
	}

	return nil
}

// RequiresWorkspace returns whether the command requires a workspace
func (cmd *PRDCommand) RequiresWorkspace() bool {
	return true
}

// RequiresConfig returns whether the command requires configuration
func (cmd *PRDCommand) RequiresConfig() bool {
	return false
}

// SupportsInteractiveMode returns whether the command supports interactive mode
func (cmd *PRDCommand) SupportsInteractiveMode() bool {
	return true
}

// GetExamples returns command examples
func (cmd *PRDCommand) GetExamples() []string {
	return []string{
		"codeforge prd create     # Interactive PRD creation",
		"codeforge prd analyze    # Auto-generate from existing code",
		"codeforge prd check      # Check status and offer creation",
	}
}

// GetFlags returns available flags for the command
func (cmd *PRDCommand) GetFlags() map[string]string {
	return map[string]string{
		"--help, -h": "Show help for PRD commands",
	}
}

// IsHidden returns whether the command should be hidden from help
func (cmd *PRDCommand) IsHidden() bool {
	return false
}

// GetCategory returns the command category
func (cmd *PRDCommand) GetCategory() string {
	return "Project Management"
}

// CanRunWithoutArgs returns whether the command can run without arguments
func (cmd *PRDCommand) CanRunWithoutArgs() bool {
	return true // Will show help
}

// GetMinArgs returns the minimum number of arguments required
func (cmd *PRDCommand) GetMinArgs() int {
	return 0
}

// GetMaxArgs returns the maximum number of arguments allowed
func (cmd *PRDCommand) GetMaxArgs() int {
	return 1
}

// ShouldShowInHelp returns whether the command should appear in general help
func (cmd *PRDCommand) ShouldShowInHelp() bool {
	return true
}

// GetPriority returns the command priority for ordering in help
func (cmd *PRDCommand) GetPriority() int {
	return 50 // Medium priority
}

// Cleanup performs any necessary cleanup after command execution
func (cmd *PRDCommand) Cleanup() error {
	return nil
}

// PreExecute performs any setup before command execution
func (cmd *PRDCommand) PreExecute(ctx context.Context, args []string) error {
	return nil
}

// PostExecute performs any cleanup after command execution
func (cmd *PRDCommand) PostExecute(ctx context.Context, args []string, err error) error {
	return nil
}

// GetCompletions returns shell completion suggestions
func (cmd *PRDCommand) GetCompletions(args []string) []string {
	if len(args) == 0 {
		return []string{"create", "analyze", "check", "help"}
	}
	return []string{}
}
