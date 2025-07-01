package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AnalyzeExistingProject analyzes an existing codebase to generate a PRD
func (s *Service) AnalyzeExistingProject() (*ProjectOverview, error) {
	// Get project name from directory
	projectName := "CodeForge"
	if s.workingDir != "" {
		projectName = filepath.Base(s.workingDir)
	}

	// Perform detailed project analysis using direct file access to avoid permission hangs
	projectType := s.detectProjectTypeDirect()
	techStack := s.detectTechStackDirect()
	authConfig := s.detectAuthConfigDirect()
	description := s.generateAnalysisDescription(projectType, techStack)

	// Create comprehensive overview using detected information
	overview := &ProjectOverview{
		ProjectName:     projectName,
		Description:     description,
		AppType:         projectType,
		TechStack:       techStack,
		TargetUsers:     []string{},
		SimilarApps:     []string{},
		DesignExamples:  []string{},
		Authentication:  authConfig,
		Billing:         BillingConfig{Required: false},
		AdditionalNotes: "Generated from automatic codebase analysis",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		Metadata: map[string]string{
			"generated_from": "codebase_analysis",
			"analysis_date":  time.Now().Format("2006-01-02"),
		},
	}

	return overview, nil
}

// detectFrontendFromPackage detects frontend technology from package.json
func (s *Service) detectFrontendFromPackage(packageContent string) string {
	var pkg map[string]interface{}
	if err := json.Unmarshal([]byte(packageContent), &pkg); err != nil {
		return ""
	}

	deps := make(map[string]interface{})
	if dependencies, ok := pkg["dependencies"].(map[string]interface{}); ok {
		for k, v := range dependencies {
			deps[k] = v
		}
	}
	if devDeps, ok := pkg["devDependencies"].(map[string]interface{}); ok {
		for k, v := range devDeps {
			deps[k] = v
		}
	}

	if _, exists := deps["typescript"]; exists {
		return "typescript"
	}
	if _, exists := deps["@types/node"]; exists {
		return "typescript"
	}

	return "javascript"
}

// detectFrameworkFromPackage detects framework from package.json
func (s *Service) detectFrameworkFromPackage(packageContent string) string {
	frameworks := map[string]string{
		"react":    "react",
		"next":     "nextjs",
		"@remix":   "remix",
		"vue":      "vue",
		"@angular": "angular",
		"svelte":   "svelte",
		"express":  "express",
		"koa":      "koa",
		"@nestjs":  "nestjs",
		"fastify":  "fastify",
	}

	for framework, name := range frameworks {
		if strings.Contains(packageContent, framework) {
			return name
		}
	}

	return ""
}

// hasFile checks if a file or directory exists
func (s *Service) hasFile(path string) bool {
	fullPath := filepath.Join(s.workingDir, path)
	_, err := os.Stat(fullPath)
	return err == nil
}

// readFileDirect reads content from a file directly without permission system
func (s *Service) readFileDirect(path string) (string, error) {
	fullPath := filepath.Join(s.workingDir, path)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// detectProjectTypeDirect analyzes the codebase to determine project type using direct file access
func (s *Service) detectProjectTypeDirect() string {
	// Check for web application indicators
	if s.hasFile("package.json") {
		if packageContent, err := s.readFileDirect("package.json"); err == nil {
			if s.containsWebFramework(packageContent) {
				return "web application"
			}
			if s.containsNodeAPI(packageContent) {
				return "API service"
			}
		}
	}

	// Check for mobile app indicators
	if s.hasFile("pubspec.yaml") || s.hasFile("flutter.yaml") {
		return "mobile application (Flutter)"
	}

	if s.hasFile("ios/") && s.hasFile("android/") {
		return "mobile application (React Native)"
	}

	// Check for desktop app indicators
	if s.hasFile("tauri.conf.json") || s.hasFile("src-tauri/") {
		return "desktop application (Tauri)"
	}

	if s.hasFile("electron.js") || s.hasFile("main.js") {
		if packageContent, err := s.readFileDirect("package.json"); err == nil {
			if strings.Contains(packageContent, "electron") {
				return "desktop application (Electron)"
			}
		}
	}

	// Check for CLI tool indicators
	if s.hasFile("go.mod") {
		if s.hasFile("cmd/") || s.hasFile("main.go") {
			return "CLI tool"
		}
	}

	if s.hasFile("Cargo.toml") {
		if cargoContent, err := s.readFileDirect("Cargo.toml"); err == nil {
			if strings.Contains(cargoContent, `[[bin]]`) {
				return "CLI tool"
			}
		}
	}

	// Check for API service indicators
	if s.hasFile("requirements.txt") || s.hasFile("pyproject.toml") {
		if s.hasFile("app.py") || s.hasFile("main.py") || s.hasFile("api/") {
			return "API service"
		}
	}

	// Default fallback
	return "application"
}

// detectTechStackDirect analyzes the codebase to determine technology stack using direct file access
func (s *Service) detectTechStackDirect() TechStack {
	stack := TechStack{
		Suggested: false,
		Reasoning: "Detected from existing codebase analysis",
	}

	// Detect frontend technologies
	if s.hasFile("package.json") {
		if packageContent, err := s.readFileDirect("package.json"); err == nil {
			stack.Frontend = s.detectFrontendFromPackage(packageContent)
			stack.Framework = s.detectFrameworkFromPackage(packageContent)
		}
	}

	// Detect backend technologies
	if s.hasFile("go.mod") {
		stack.Backend = "go"
		if goModContent, err := s.readFileDirect("go.mod"); err == nil {
			if strings.Contains(goModContent, "gin-gonic") {
				stack.Framework = "gin"
			} else if strings.Contains(goModContent, "echo") {
				stack.Framework = "echo"
			} else if strings.Contains(goModContent, "fiber") {
				stack.Framework = "fiber"
			}
		}
	} else if s.hasFile("requirements.txt") || s.hasFile("pyproject.toml") {
		stack.Backend = "python"
		if reqContent, err := s.readFileDirect("requirements.txt"); err == nil {
			if strings.Contains(reqContent, "django") {
				stack.Framework = "django"
			} else if strings.Contains(reqContent, "flask") {
				stack.Framework = "flask"
			} else if strings.Contains(reqContent, "fastapi") {
				stack.Framework = "fastapi"
			}
		}
	} else if s.hasFile("Cargo.toml") {
		stack.Backend = "rust"
		if cargoContent, err := s.readFileDirect("Cargo.toml"); err == nil {
			if strings.Contains(cargoContent, "axum") {
				stack.Framework = "axum"
			} else if strings.Contains(cargoContent, "warp") {
				stack.Framework = "warp"
			} else if strings.Contains(cargoContent, "actix-web") {
				stack.Framework = "actix-web"
			}
		}
	}

	// Detect database
	stack.Database = s.detectDatabaseDirect()

	// Detect deployment
	stack.Deployment = s.detectDeploymentDirect()

	return stack
}

// detectDatabaseDirect detects database technology using direct file access
func (s *Service) detectDatabaseDirect() string {
	// Check for database config files
	if s.hasFile("prisma/schema.prisma") {
		if schemaContent, err := s.readFileDirect("prisma/schema.prisma"); err == nil {
			if strings.Contains(schemaContent, "postgresql") {
				return "postgres"
			}
			if strings.Contains(schemaContent, "mysql") {
				return "mysql"
			}
			if strings.Contains(schemaContent, "sqlite") {
				return "sqlite"
			}
		}
	}

	// Check for Docker Compose database services
	if s.hasFile("docker-compose.yml") || s.hasFile("docker-compose.yaml") {
		if dockerContent, err := s.readFileDirect("docker-compose.yml"); err == nil {
			if strings.Contains(dockerContent, "postgres") {
				return "postgres"
			}
			if strings.Contains(dockerContent, "mysql") {
				return "mysql"
			}
			if strings.Contains(dockerContent, "mongodb") {
				return "mongodb"
			}
			if strings.Contains(dockerContent, "redis") {
				return "redis"
			}
		}
	}

	// Check for database-specific files
	if s.hasFile("database.sqlite") || s.hasFile("db.sqlite3") {
		return "sqlite"
	}

	return ""
}

// detectDeploymentDirect detects deployment configuration using direct file access
func (s *Service) detectDeploymentDirect() string {
	// Check for containerization
	if s.hasFile("Dockerfile") {
		return "docker"
	}

	if s.hasFile("docker-compose.yml") || s.hasFile("docker-compose.yaml") {
		return "docker-compose"
	}

	// Check for cloud deployment configs
	if s.hasFile("vercel.json") || s.hasFile(".vercel/") {
		return "vercel"
	}

	if s.hasFile("netlify.toml") || s.hasFile("_redirects") {
		return "netlify"
	}

	if s.hasFile(".github/workflows/") {
		return "github-actions"
	}

	if s.hasFile("heroku.yml") || s.hasFile("Procfile") {
		return "heroku"
	}

	if s.hasFile("railway.json") {
		return "railway"
	}

	return ""
}

// detectAuthConfigDirect detects authentication configuration using direct file access
func (s *Service) detectAuthConfigDirect() AuthConfig {
	// Check for Next.js auth
	if s.hasFile("package.json") {
		if packageContent, err := s.readFileDirect("package.json"); err == nil {
			if strings.Contains(packageContent, "next-auth") {
				return AuthConfig{Required: true, Provider: "NextAuth.js"}
			}
			if strings.Contains(packageContent, "@auth0") {
				return AuthConfig{Required: true, Provider: "Auth0"}
			}
			if strings.Contains(packageContent, "firebase") {
				return AuthConfig{Required: true, Provider: "Firebase Auth"}
			}
			if strings.Contains(packageContent, "supabase") {
				return AuthConfig{Required: true, Provider: "Supabase Auth"}
			}
		}
	}

	// Check for auth-related files
	authFiles := []string{"auth.js", "auth.ts", "middleware.ts", "auth/"}
	for _, file := range authFiles {
		if s.hasFile(file) {
			return AuthConfig{Required: true, Provider: "Custom"}
		}
	}

	return AuthConfig{Required: false}
}

// containsWebFramework checks if package.json contains web framework dependencies
func (s *Service) containsWebFramework(packageContent string) bool {
	webFrameworks := []string{"react", "vue", "angular", "svelte", "next", "nuxt", "gatsby"}
	for _, framework := range webFrameworks {
		if strings.Contains(packageContent, framework) {
			return true
		}
	}
	return false
}

// containsNodeAPI checks if package.json contains Node.js API dependencies
func (s *Service) containsNodeAPI(packageContent string) bool {
	apiFrameworks := []string{"express", "koa", "fastify", "hapi", "nest"}
	for _, framework := range apiFrameworks {
		if strings.Contains(packageContent, framework) {
			return true
		}
	}
	return false
}

// generateAnalysisDescription generates a description based on project analysis
func (s *Service) generateAnalysisDescription(projectType string, techStack TechStack) string {
	var desc strings.Builder

	desc.WriteString(fmt.Sprintf("A %s", projectType))

	if techStack.Backend != "" {
		desc.WriteString(fmt.Sprintf(" built with %s", techStack.Backend))
	}

	if techStack.Framework != "" {
		desc.WriteString(fmt.Sprintf(" using %s", techStack.Framework))
	}

	if techStack.Database != "" {
		desc.WriteString(fmt.Sprintf(" and %s database", techStack.Database))
	}

	desc.WriteString(". Generated from existing codebase analysis.")

	return desc.String()
}
