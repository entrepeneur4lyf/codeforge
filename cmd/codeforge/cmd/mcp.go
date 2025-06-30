package cmd

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/entrepeneur4lyf/codeforge/internal/app"
	"github.com/spf13/cobra"
)

// mcpCmd represents the mcp command
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Model Context Protocol operations",
	Long: `Model Context Protocol operations for CodeForge server including:
- Server management and monitoring
- Capability listing and information
- Server startup and configuration`,
}

// mcpListCmd lists available MCP server capabilities
var mcpListCmd = &cobra.Command{
	Use:   "list",
	Short: "List MCP server capabilities",
	Long:  "List the tools, resources, and prompts available in the CodeForge MCP server",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("CodeForge MCP Server Capabilities:")

		fmt.Println("🔧 Tools:")
		fmt.Println("   • semantic_search - Search for code using semantic similarity")
		fmt.Println("   • read_file - Read file contents from the workspace")
		fmt.Println("   • write_file - Write content to files in the workspace")
		fmt.Println("   • analyze_code - Analyze code structure and extract symbols")
		fmt.Println("   • get_project_structure - Get directory structure of the project")

		fmt.Println("\n📚 Resources:")
		fmt.Println("   • codeforge://project/metadata - Project information")
		fmt.Println("   • codeforge://files/{path} - File content access")
		fmt.Println("   • codeforge://git/status - Git repository status")

		fmt.Println("\n💡 Prompts:")
		fmt.Println("   • code_review - Code review assistance")
		fmt.Println("   • debug_help - Debugging guidance")
		fmt.Println("   • refactoring_guide - Refactoring recommendations")
		fmt.Println("   • documentation_help - Documentation generation")
		fmt.Println("   • testing_help - Test creation assistance")

		fmt.Println("\n🚀 Usage:")
		fmt.Println("   Start MCP server: codeforge mcp server")
		fmt.Println("   Or use standalone: ./mcp-server -workspace /path/to/project")

		return nil
	},
}

// mcpServerCmd starts the MCP server
var mcpServerCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the MCP server",
	Long:  "Start the CodeForge MCP server with stdio, HTTP, or SSE transport",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Get flags
		transport, _ := cmd.Flags().GetString("transport")
		addr, _ := cmd.Flags().GetString("addr")
		workspace, _ := cmd.Flags().GetString("workspace")
		dbPath, _ := cmd.Flags().GetString("db")

		// Resolve workspace root
		absWorkspace, err := filepath.Abs(workspace)
		if err != nil {
			return fmt.Errorf("failed to resolve workspace path: %w", err)
		}

		// Set up database path
		if dbPath == "" {
			dbPath = filepath.Join(absWorkspace, ".codeforge", "vector.db")
		}

		// Ensure .codeforge directory exists
		codeforgeDir := filepath.Dir(dbPath)
		if err := os.MkdirAll(codeforgeDir, 0755); err != nil {
			return fmt.Errorf("failed to create .codeforge directory: %w", err)
		}

		// Create application with all systems integrated
		appConfig := &app.AppConfig{
			WorkspaceRoot:     absWorkspace,
			DatabasePath:      dbPath,
			EnablePermissions: true,
			EnableContextMgmt: true,
			Debug:             false,
		}

		ctx := context.Background()
		codeforgeApp, err := app.NewApp(ctx, appConfig)
		if err != nil {
			return fmt.Errorf("failed to initialize CodeForge app: %w", err)
		}
		defer codeforgeApp.Close()

		// Get the integrated MCP server
		mcpServer := codeforgeApp.MCPServer
		if mcpServer == nil {
			return fmt.Errorf("MCP server not available in app")
		}

		log.Printf("Starting CodeForge MCP server...")
		log.Printf("Workspace: %s", absWorkspace)
		log.Printf("Database: %s", dbPath)
		log.Printf("Transport: %s", transport)

		// Start server based on transport type
		switch transport {
		case "stdio":
			return mcpServer.Start()
		case "sse":
			log.Printf("Starting SSE server on %s", addr)
			return mcpServer.StartSSE(addr)
		case "http":
			log.Printf("Starting HTTP server on %s", addr)
			return mcpServer.StartStreamableHTTP(addr)
		default:
			return fmt.Errorf("unknown transport type: %s", transport)
		}
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
	mcpCmd.AddCommand(mcpListCmd)
	mcpCmd.AddCommand(mcpServerCmd)

	// Add flags for the server command
	mcpServerCmd.Flags().StringP("transport", "t", "stdio", "Transport type (stdio, sse, http)")
	mcpServerCmd.Flags().StringP("addr", "a", ":8080", "Address for HTTP/SSE transport")
	mcpServerCmd.Flags().StringP("workspace", "w", ".", "Workspace root directory")
	mcpServerCmd.Flags().StringP("db", "d", "", "Path to vector database (default: workspace/.codeforge/vector.db)")
}
