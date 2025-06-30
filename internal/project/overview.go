package project

import (
	"fmt"
	"strings"
	"time"
)

// ProjectOverview represents a comprehensive project overview
type ProjectOverview struct {
	ProjectName     string            `json:"project_name"`
	Description     string            `json:"description"`
	AppType         string            `json:"app_type"`
	TechStack       TechStack         `json:"tech_stack"`
	TargetUsers     []string          `json:"target_users"`
	SimilarApps     []string          `json:"similar_apps"`
	DesignExamples  []string          `json:"design_examples"`
	Authentication  AuthConfig        `json:"authentication"`
	Billing         BillingConfig     `json:"billing"`
	AdditionalNotes string            `json:"additional_notes"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	Metadata        map[string]string `json:"metadata"`
}

// TechStack represents the technology stack for the project
type TechStack struct {
	Frontend   string   `json:"frontend"`
	Backend    string   `json:"backend"`
	Framework  string   `json:"framework"`
	Database   string   `json:"database"`
	Deployment string   `json:"deployment"`
	Additional []string `json:"additional"`
	Suggested  bool     `json:"suggested"` // Whether this was AI-suggested
	Reasoning  string   `json:"reasoning"` // Why this stack was chosen
}

// AuthConfig represents authentication configuration
type AuthConfig struct {
	Required bool     `json:"required"`
	Provider string   `json:"provider"`
	Features []string `json:"features"`
}

// BillingConfig represents billing configuration
type BillingConfig struct {
	Required bool   `json:"required"`
	Provider string `json:"provider"`
	Model    string `json:"model"` // subscription, one-time, usage-based
}

// PRDQuestions represents the questions to ask for PRD creation
type PRDQuestions struct {
	AppType         string
	TechStack       string
	TargetUsers     string
	SimilarApps     string
	DesignExamples  string
	Authentication  string
	Billing         string
	AdditionalNotes string
}

// CreatePRDFromQuestions creates a project overview from user responses
func (s *Service) CreatePRDFromQuestions(questions PRDQuestions) (*ProjectOverview, error) {
	// Parse tech stack preference
	techStack := s.parseTechStack(questions.TechStack, questions.AppType)

	// Parse authentication config
	authConfig := s.parseAuthConfig(questions.Authentication)

	// Parse billing config
	billingConfig := s.parseBillingConfig(questions.Billing)

	overview := &ProjectOverview{
		ProjectName:     s.extractProjectName(),
		Description:     s.generateDescription(questions),
		AppType:         questions.AppType,
		TechStack:       techStack,
		TargetUsers:     s.parseTargetUsers(questions.TargetUsers),
		SimilarApps:     s.parseSimilarApps(questions.SimilarApps),
		DesignExamples:  s.parseDesignExamples(questions.DesignExamples),
		Authentication:  authConfig,
		Billing:         billingConfig,
		AdditionalNotes: questions.AdditionalNotes,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		Metadata:        make(map[string]string),
	}

	return overview, nil
}

// generateDescription generates a project description from responses
func (s *Service) generateDescription(questions PRDQuestions) string {
	var desc strings.Builder

	desc.WriteString(fmt.Sprintf("A %s application", questions.AppType))

	if questions.TargetUsers != "" {
		desc.WriteString(fmt.Sprintf(" designed for %s", questions.TargetUsers))
	}

	if questions.SimilarApps != "" {
		desc.WriteString(fmt.Sprintf(", similar to %s", questions.SimilarApps))
	}

	desc.WriteString(".")

	return desc.String()
}

// parseTechStack parses tech stack from user input
func (s *Service) parseTechStack(stackInput, appType string) TechStack {
	if stackInput == "" || strings.ToLower(stackInput) == "suggest" || strings.ToLower(stackInput) == "analyze" {
		// AI should suggest based on app type
		return TechStack{
			Suggested: true,
			Reasoning: fmt.Sprintf("Tech stack to be suggested based on %s application requirements", appType),
		}
	}

	// Parse user-provided stack
	return TechStack{
		Frontend:  s.extractTechComponent(stackInput, "frontend", "html", "css", "javascript", "typescript", "bootstrap", "tailwind", "material-ui", "antd", "bulma", "semantic-ui", "shadcn-ui", "react aria"),
		Backend:   s.extractTechComponent(stackInput, "backend", "go", "javascript", "typescript", "python", "rust", "java", "c", "cpp", "csharp", "ruby", "php"),
		Framework: s.extractTechComponent(stackInput, "framework", "react", "nextjs", "remix", "vue", "angular", "svelte", "solid", "ember", "express", "koa", "nest", "fastify", "hapi", "sanic", "flask", "django", "rails", "spring", "laravel", "aspnet"),
		Database:  s.extractTechComponent(stackInput, "database", "postgres", "mysql", "mongodb", "sqlite", "redis", "pinecone", "weaviate", "chroma", "qdrant", "milvus", "pgvector", "supabase", "planetscale", "turso", "libsql"),
		Suggested: false,
		Reasoning: "User-specified technology stack",
	}
}

// extractTechComponent extracts a specific tech component from input
func (s *Service) extractTechComponent(input, category string, options ...string) string {
	inputLower := strings.ToLower(input)
	for _, option := range options {
		if strings.Contains(inputLower, option) {
			return option
		}
	}
	return ""
}

// parseAuthConfig parses authentication configuration
func (s *Service) parseAuthConfig(authInput string) AuthConfig {
	if authInput == "" || strings.ToLower(authInput) == "no" {
		return AuthConfig{Required: false}
	}

	config := AuthConfig{Required: true}

	inputLower := strings.ToLower(authInput)
	if strings.Contains(inputLower, "auth0") {
		config.Provider = "Auth0"
	} else if strings.Contains(inputLower, "firebase") {
		config.Provider = "Firebase Auth"
	} else if strings.Contains(inputLower, "supabase") {
		config.Provider = "Supabase Auth"
	} else if strings.Contains(inputLower, "clerk") {
		config.Provider = "Clerk"
	}

	return config
}

// parseBillingConfig parses billing configuration
func (s *Service) parseBillingConfig(billingInput string) BillingConfig {
	if billingInput == "" || strings.ToLower(billingInput) == "no" {
		return BillingConfig{Required: false}
	}

	config := BillingConfig{Required: true}

	inputLower := strings.ToLower(billingInput)
	if strings.Contains(inputLower, "stripe") {
		config.Provider = "Stripe"
	} else if strings.Contains(inputLower, "paddle") {
		config.Provider = "Paddle"
	}

	if strings.Contains(inputLower, "subscription") {
		config.Model = "subscription"
	} else if strings.Contains(inputLower, "usage") {
		config.Model = "usage-based"
	} else {
		config.Model = "one-time"
	}

	return config
}

// parseTargetUsers parses target users from input
func (s *Service) parseTargetUsers(input string) []string {
	if input == "" {
		return []string{}
	}

	users := strings.Split(input, ",")
	var result []string
	for _, user := range users {
		result = append(result, strings.TrimSpace(user))
	}
	return result
}

// parseSimilarApps parses similar apps from input
func (s *Service) parseSimilarApps(input string) []string {
	if input == "" {
		return []string{}
	}

	apps := strings.Split(input, ",")
	var result []string
	for _, app := range apps {
		result = append(result, strings.TrimSpace(app))
	}
	return result
}

// parseDesignExamples parses design examples from input
func (s *Service) parseDesignExamples(input string) []string {
	if input == "" {
		return []string{}
	}

	examples := strings.Split(input, ",")
	var result []string
	for _, example := range examples {
		result = append(result, strings.TrimSpace(example))
	}
	return result
}

// GenerateProjectOverviewMarkdown creates a markdown representation of the project overview
func (s *Service) GenerateProjectOverviewMarkdown(overview *ProjectOverview) string {
	var md strings.Builder

	md.WriteString(fmt.Sprintf("# %s - Project Overview\n\n", overview.ProjectName))

	md.WriteString("## Project Description\n")
	md.WriteString(fmt.Sprintf("%s\n\n", overview.Description))

	md.WriteString("## Application Type\n")
	md.WriteString(fmt.Sprintf("%s\n\n", overview.AppType))

	// Tech Stack
	md.WriteString("## Technology Stack\n")
	if overview.TechStack.Suggested {
		md.WriteString("**Status**: AI-Suggested\n")
		md.WriteString(fmt.Sprintf("**Reasoning**: %s\n\n", overview.TechStack.Reasoning))
	} else {
		md.WriteString("**Status**: User-Specified\n")
		if overview.TechStack.Frontend != "" {
			md.WriteString(fmt.Sprintf("- **Frontend**: %s\n", overview.TechStack.Frontend))
		}
		if overview.TechStack.Backend != "" {
			md.WriteString(fmt.Sprintf("- **Backend**: %s\n", overview.TechStack.Backend))
		}
		if overview.TechStack.Framework != "" {
			md.WriteString(fmt.Sprintf("- **Framework**: %s\n", overview.TechStack.Framework))
		}
		if overview.TechStack.Database != "" {
			md.WriteString(fmt.Sprintf("- **Database**: %s\n", overview.TechStack.Database))
		}
		if overview.TechStack.Deployment != "" {
			md.WriteString(fmt.Sprintf("- **Deployment**: %s\n", overview.TechStack.Deployment))
		}
		md.WriteString("\n")
	}

	// Target Users
	if len(overview.TargetUsers) > 0 {
		md.WriteString("## Target Users\n")
		for _, user := range overview.TargetUsers {
			md.WriteString(fmt.Sprintf("- %s\n", user))
		}
		md.WriteString("\n")
	}

	// Similar Apps
	if len(overview.SimilarApps) > 0 {
		md.WriteString("## Similar Applications\n")
		for _, app := range overview.SimilarApps {
			md.WriteString(fmt.Sprintf("- %s\n", app))
		}
		md.WriteString("\n")
	}

	// Design Examples
	if len(overview.DesignExamples) > 0 {
		md.WriteString("## Design References\n")
		for _, example := range overview.DesignExamples {
			md.WriteString(fmt.Sprintf("- %s\n", example))
		}
		md.WriteString("\n")
	}

	// Authentication
	md.WriteString("## Authentication\n")
	if overview.Authentication.Required {
		md.WriteString("**Required**: Yes\n")
		if overview.Authentication.Provider != "" {
			md.WriteString(fmt.Sprintf("**Provider**: %s\n", overview.Authentication.Provider))
		}
		if len(overview.Authentication.Features) > 0 {
			md.WriteString("**Features**:\n")
			for _, feature := range overview.Authentication.Features {
				md.WriteString(fmt.Sprintf("- %s\n", feature))
			}
		}
	} else {
		md.WriteString("**Required**: No\n")
	}
	md.WriteString("\n")

	// Billing
	md.WriteString("## Billing\n")
	if overview.Billing.Required {
		md.WriteString("**Required**: Yes\n")
		if overview.Billing.Provider != "" {
			md.WriteString(fmt.Sprintf("**Provider**: %s\n", overview.Billing.Provider))
		}
		if overview.Billing.Model != "" {
			md.WriteString(fmt.Sprintf("**Model**: %s\n", overview.Billing.Model))
		}
	} else {
		md.WriteString("**Required**: No\n")
	}
	md.WriteString("\n")

	// Additional Notes
	if overview.AdditionalNotes != "" {
		md.WriteString("## Additional Notes\n")
		md.WriteString(fmt.Sprintf("%s\n\n", overview.AdditionalNotes))
	}

	// Metadata
	md.WriteString("---\n")
	md.WriteString(fmt.Sprintf("**Created**: %s\n", overview.CreatedAt.Format("2006-01-02 15:04:05")))
	md.WriteString(fmt.Sprintf("**Updated**: %s\n", overview.UpdatedAt.Format("2006-01-02 15:04:05")))

	return md.String()
}

// GenerateProjectSummary creates a concise summary for AGENT.md (context only)
func (s *Service) GenerateProjectSummary(overview *ProjectOverview) string {
	var summary strings.Builder

	summary.WriteString(fmt.Sprintf("# %s\n\n", overview.ProjectName))
	summary.WriteString(fmt.Sprintf("**Type**: %s\n", overview.AppType))
	summary.WriteString(fmt.Sprintf("**Description**: %s\n\n", overview.Description))

	if !overview.TechStack.Suggested {
		summary.WriteString("**Tech Stack**:\n")
		if overview.TechStack.Frontend != "" {
			summary.WriteString(fmt.Sprintf("- Frontend: %s\n", overview.TechStack.Frontend))
		}
		if overview.TechStack.Backend != "" {
			summary.WriteString(fmt.Sprintf("- Backend: %s\n", overview.TechStack.Backend))
		}
		if overview.TechStack.Framework != "" {
			summary.WriteString(fmt.Sprintf("- Framework: %s\n", overview.TechStack.Framework))
		}
		if overview.TechStack.Database != "" {
			summary.WriteString(fmt.Sprintf("- Database: %s\n", overview.TechStack.Database))
		}
		summary.WriteString("\n")
	}

	if len(overview.TargetUsers) > 0 {
		summary.WriteString(fmt.Sprintf("**Target Users**: %s\n", strings.Join(overview.TargetUsers, ", ")))
	}

	if overview.Authentication.Required {
		summary.WriteString("**Authentication**: Required")
		if overview.Authentication.Provider != "" {
			summary.WriteString(fmt.Sprintf(" (%s)", overview.Authentication.Provider))
		}
		summary.WriteString("\n")
	}

	if overview.Billing.Required {
		summary.WriteString("**Billing**: Required")
		if overview.Billing.Provider != "" {
			summary.WriteString(fmt.Sprintf(" (%s)", overview.Billing.Provider))
		}
		summary.WriteString("\n")
	}

	return summary.String()
}
