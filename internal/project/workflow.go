package project

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
)

// PRDWorkflow handles the interactive PRD creation process
type PRDWorkflow struct {
	service *Service
	scanner *bufio.Scanner
}

// NewPRDWorkflow creates a new PRD workflow
func NewPRDWorkflow(service *Service) *PRDWorkflow {
	return &PRDWorkflow{
		service: service,
		scanner: bufio.NewScanner(os.Stdin),
	}
}

// RunInteractivePRDCreation runs the interactive PRD creation workflow
func (w *PRDWorkflow) RunInteractivePRDCreation(ctx context.Context) (*ProjectOverview, error) {
	fmt.Println("CodeForge PRD Creation Workflow")
	fmt.Println("Let's create a comprehensive project overview for your application.")
	fmt.Println()

	// Check if existing project
	if w.service.HasExistingProject() {
		fmt.Println("Existing project detected. Would you like to:")
		fmt.Println("1. Analyze existing codebase automatically")
		fmt.Println("2. Create PRD manually through questions")
		fmt.Print("Choose option (1 or 2): ")

		choice := w.readInput()
		if choice == "1" {
			return w.service.AnalyzeExistingProject()
		}
	}

	// Gather PRD information through questions
	questions := w.askPRDQuestions()

	// Create project overview from responses
	overview, err := w.service.CreatePRDFromQuestions(questions)
	if err != nil {
		return nil, fmt.Errorf("failed to create PRD from questions: %w", err)
	}

	// Show preview and get approval
	if approved := w.showPreviewAndGetApproval(overview); !approved {
		fmt.Println("PRD creation cancelled.")
		return nil, fmt.Errorf("PRD creation cancelled by user")
	}

	// Save the PRD files
	if err := w.service.CreatePRDFiles(overview); err != nil {
		return nil, fmt.Errorf("failed to save PRD files: %w", err)
	}

	fmt.Println("PRD created successfully!")
	fmt.Println("Files created:")
	fmt.Println("  - project-overview.md (comprehensive documentation)")
	fmt.Println("  - AGENTS.md (concise context for AI)")
	fmt.Println()

	return overview, nil
}

// askPRDQuestions asks the essential PRD questions
func (w *PRDWorkflow) askPRDQuestions() PRDQuestions {
	fmt.Println("Please answer the following questions to create your PRD:")
	fmt.Println()

	questions := PRDQuestions{}

	// Question 1: App Type
	fmt.Print("1. What kind of app are we building? (web app, mobile app, CLI tool, API service, desktop app): ")
	questions.AppType = w.readInput()

	// Question 2: Tech Stack
	fmt.Print("2. Do you have a tech stack in mind or should I analyze and make suggestions? (describe stack or 'suggest'): ")
	questions.TechStack = w.readInput()

	// Question 3: Target Users
	fmt.Print("3. Who will use this application? (describe target users): ")
	questions.TargetUsers = w.readInput()

	// Question 4: Similar Apps
	fmt.Print("4. Do you have any examples of app(s) that is similar to what you want? (comma-separated): ")
	questions.SimilarApps = w.readInput()

	// Question 5: Design Examples
	fmt.Print("5. Do you have examples of what you want the app to look like? (design references): ")
	questions.DesignExamples = w.readInput()

	// Question 6: Authentication & Billing
	fmt.Print("6. Will the application have authentication? If so, do you have providers in mind? (Auth0, Firebase, Supabase, Clerk, or 'no'): ")
	questions.Authentication = w.readInput()

	fmt.Print("7. Will the application have billing? If so, do you have providers in mind? (Stripe, Paddle, or 'no'): ")
	questions.Billing = w.readInput()

	// Question 7: Additional Notes
	fmt.Print("8. Are there any other details about the app that I should know about? (additional notes): ")
	questions.AdditionalNotes = w.readInput()

	return questions
}

// readInput reads user input from stdin
func (w *PRDWorkflow) readInput() string {
	if w.scanner.Scan() {
		return strings.TrimSpace(w.scanner.Text())
	}
	return ""
}

// showPreviewAndGetApproval shows the PRD preview and gets user approval
func (w *PRDWorkflow) showPreviewAndGetApproval(overview *ProjectOverview) bool {
	fmt.Println()
	fmt.Println("📖 PRD Preview:")
	fmt.Println("================")

	// Show concise summary (what will go in AGENTS.md)
	summary := w.service.GenerateProjectSummary(overview)
	fmt.Println(summary)

	fmt.Println("================")
	fmt.Println()
	fmt.Print("What do you think of this overview? Do you have any changes? (approve/edit/cancel): ")

	response := strings.ToLower(w.readInput())

	switch response {
	case "approve", "yes", "y", "ok", "good", "":
		return true
	case "edit", "modify", "change":
		return w.handleEdits(overview)
	case "cancel", "no", "n":
		return false
	default:
		fmt.Println("Please respond with 'approve', 'edit', or 'cancel'.")
		return w.showPreviewAndGetApproval(overview)
	}
}

// handleEdits handles user edits to the PRD
func (w *PRDWorkflow) handleEdits(overview *ProjectOverview) bool {
	fmt.Println()
	fmt.Println("🔧 What would you like to edit?")
	fmt.Println("1. Project description")
	fmt.Println("2. Application type")
	fmt.Println("3. Tech stack")
	fmt.Println("4. Target users")
	fmt.Println("5. Similar apps")
	fmt.Println("6. Design examples")
	fmt.Println("7. Authentication")
	fmt.Println("8. Billing")
	fmt.Println("9. Additional notes")
	fmt.Println("0. Done editing")
	fmt.Print("Choose option (0-9): ")

	choice := w.readInput()

	switch choice {
	case "1":
		fmt.Print("New project description: ")
		overview.Description = w.readInput()
	case "2":
		fmt.Print("New application type: ")
		overview.AppType = w.readInput()
	case "3":
		fmt.Print("New tech stack (or 'suggest' for AI suggestions): ")
		techStackInput := w.readInput()
		overview.TechStack = w.service.parseTechStack(techStackInput, overview.AppType)
	case "4":
		fmt.Print("New target users (comma-separated): ")
		usersInput := w.readInput()
		overview.TargetUsers = w.service.parseTargetUsers(usersInput)
	case "5":
		fmt.Print("New similar apps (comma-separated): ")
		appsInput := w.readInput()
		overview.SimilarApps = w.service.parseSimilarApps(appsInput)
	case "6":
		fmt.Print("New design examples (comma-separated): ")
		designInput := w.readInput()
		overview.DesignExamples = w.service.parseDesignExamples(designInput)
	case "7":
		fmt.Print("New authentication config (provider or 'no'): ")
		authInput := w.readInput()
		overview.Authentication = w.service.parseAuthConfig(authInput)
	case "8":
		fmt.Print("New billing config (provider or 'no'): ")
		billingInput := w.readInput()
		overview.Billing = w.service.parseBillingConfig(billingInput)
	case "9":
		fmt.Print("New additional notes: ")
		overview.AdditionalNotes = w.readInput()
	case "0":
		return w.showPreviewAndGetApproval(overview)
	default:
		fmt.Println("Invalid option. Please choose 0-9.")
		return w.handleEdits(overview)
	}

	// Continue editing
	return w.handleEdits(overview)
}

// QuickPRDFromExisting creates a PRD from existing project analysis
func (w *PRDWorkflow) QuickPRDFromExisting(ctx context.Context) (*ProjectOverview, error) {
	overview, err := w.service.AnalyzeExistingProject()
	if err != nil {
		return nil, fmt.Errorf("failed to analyze existing project: %w", err)
	}

	// Analysis complete (spinner handled by command level)
	// Skip approval for automatic analysis - just save the files

	// Save the PRD files
	if err := w.service.CreatePRDFiles(overview); err != nil {
		return nil, fmt.Errorf("failed to save PRD files: %w", err)
	}

	// Files created silently for automatic analysis

	return overview, nil
}

// UpdateExistingPRD reads existing AGENTS.md, analyzes current project, and updates documentation
func (w *PRDWorkflow) UpdateExistingPRD(ctx context.Context) (*ProjectOverview, error) {
	// Read existing AGENTS.md content
	existingContent, err := w.service.readFile("AGENTS.md")
	if err != nil {
		return nil, fmt.Errorf("failed to read existing AGENTS.md: %w", err)
	}

	// Analyze current project state
	overview, err := w.service.AnalyzeExistingProject()
	if err != nil {
		return nil, fmt.Errorf("failed to analyze current project: %w", err)
	}

	// Update AGENTS.md with current analysis (incremental update)
	updatedSummary := w.service.UpdateProjectSummary(overview, existingContent)
	err = w.service.writeFile("AGENTS.md", []byte(updatedSummary))
	if err != nil {
		return nil, fmt.Errorf("failed to update AGENTS.md: %w", err)
	}

	// Create/update comprehensive project-overview.md
	err = w.service.SaveProjectOverview(overview)
	if err != nil {
		return nil, fmt.Errorf("failed to save project overview: %w", err)
	}

	// Clear spinner and show completion
	fmt.Print("\r")
	fmt.Println("Project documentation updated!")

	return overview, nil
}

// CheckAndOfferPRDCreation checks for existing PRD and offers to create one
func (w *PRDWorkflow) CheckAndOfferPRDCreation(ctx context.Context) (*ProjectOverview, error) {
	// Check for existing PRD
	hasExisting, content, err := w.service.CheckForExistingPRD()
	if err != nil {
		return nil, fmt.Errorf("failed to check for existing PRD: %w", err)
	}

	if hasExisting {
		fmt.Println("Existing project documentation found.")
		fmt.Printf("Content preview:\n%s\n", content[:min(200, len(content))])
		fmt.Print("Would you like to create a new PRD anyway? (y/n): ")

		if strings.ToLower(w.readInput()) != "y" {
			return nil, nil // User doesn't want to create new PRD
		}
	}

	// Offer PRD creation
	fmt.Println("No project overview found.")
	fmt.Print("Would you like to create a Project Requirements Document (PRD)? (y/n): ")

	if strings.ToLower(w.readInput()) != "y" {
		return nil, nil // User doesn't want to create PRD
	}

	return w.RunInteractivePRDCreation(ctx)
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
