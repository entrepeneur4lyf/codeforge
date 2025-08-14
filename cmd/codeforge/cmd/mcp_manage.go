package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/mcp"
	"github.com/spf13/cobra"
)

var mcpManageCmd = &cobra.Command{
	Use:   "manage",
	Short: "Manage MCP server configurations",
	Long: `Manage Model Context Protocol (MCP) server configurations.
	
This command allows you to add, remove, enable, disable, and configure MCP servers
that provide additional tools, resources, and prompts to CodeForge.`,
}

var mcpListServersCmd = &cobra.Command{
	Use:   "list",
	Short: "List all MCP servers",
	Long:  "List all configured MCP servers with their status",
	RunE: func(cmd *cobra.Command, args []string) error {
		manager := getMCPManager()

		statuses, err := manager.ListServerStatuses()
		if err != nil {
			return fmt.Errorf("error listing MCP servers: %w", err)
		}

		if len(statuses) == 0 {
			fmt.Println("No MCP servers configured")
			return nil
		}

		fmt.Printf("%-20s %-10s %-10s %-15s %-10s %s\n", "NAME", "TYPE", "ENABLED", "CONNECTED", "TOOLS", "DESCRIPTION")
		fmt.Println(strings.Repeat("-", 80))

		for _, status := range statuses {
			connected := "No"
			if status.Connected {
				connected = "Yes"
			}

			enabled := "No"
			if status.Enabled {
				enabled = "Yes"
			}

			fmt.Printf("%-20s %-10s %-10s %-15s %-10d %s\n",
				status.Name,
				status.Type,
				enabled,
				connected,
				status.ToolCount,
				status.Description,
			)
		}
		return nil
	},
}

var mcpAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new MCP server",
	Long:  "Add a new MCP server configuration",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		serverType, _ := cmd.Flags().GetString("type")
		description, _ := cmd.Flags().GetString("description")
		command, _ := cmd.Flags().GetStringSlice("command")
		url, _ := cmd.Flags().GetString("url")
		enabled, _ := cmd.Flags().GetBool("enabled")

		if serverType == "" {
			return fmt.Errorf("--type is required")
		}

		config := &mcp.MCPServerConfig{
			Name:        name,
			Type:        mcp.MCPServerType(serverType),
			Description: description,
			Enabled:     enabled,
			Tools:       true,
			Resources:   true,
			Prompts:     true,
		}

		switch config.Type {
		case mcp.MCPServerTypeLocal:
			if len(command) == 0 {
				return fmt.Errorf("--command is required for local servers")
			}
			config.Command = command
		case mcp.MCPServerTypeRemote, mcp.MCPServerTypeSSE, mcp.MCPServerTypeHTTP:
			if url == "" {
				return fmt.Errorf("--url is required for remote servers")
			}
			config.URL = url
		default:
			return fmt.Errorf("unsupported server type: %s", serverType)
		}

		manager := getMCPManager()
		if err := manager.AddServer(config); err != nil {
			return fmt.Errorf("error adding MCP server: %w", err)
		}

		fmt.Printf("MCP server '%s' added successfully\n", name)
		return nil
	},
}

var mcpRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove an MCP server",
	Long:  "Remove an MCP server configuration",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		manager := getMCPManager()
		if err := manager.RemoveServer(name); err != nil {
			return fmt.Errorf("error removing MCP server: %w", err)
		}

		fmt.Printf("MCP server '%s' removed successfully\n", name)
		return nil
	},
}

var mcpEnableCmd = &cobra.Command{
	Use:   "enable <name>",
	Short: "Enable an MCP server",
	Long:  "Enable an MCP server and start it",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		manager := getMCPManager()
		if err := manager.EnableServer(name); err != nil {
			return fmt.Errorf("error enabling MCP server: %w", err)
		}

		fmt.Printf("MCP server '%s' enabled successfully\n", name)
		return nil
	},
}

var mcpDisableCmd = &cobra.Command{
	Use:   "disable <name>",
	Short: "Disable an MCP server",
	Long:  "Disable an MCP server and stop it",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		manager := getMCPManager()
		if err := manager.DisableServer(name); err != nil {
			return fmt.Errorf("error disabling MCP server: %w", err)
		}

		fmt.Printf("MCP server '%s' disabled successfully\n", name)
		return nil
	},
}

var mcpStatusCmd = &cobra.Command{
	Use:   "status <name>",
	Short: "Show MCP server status",
	Long:  "Show detailed status of an MCP server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		manager := getMCPManager()
		status, err := manager.GetServerStatus(name)
		if err != nil {
			return fmt.Errorf("error getting MCP server status: %w", err)
		}

		fmt.Printf("MCP Server: %s\n", status.Name)
		fmt.Printf("Type: %s\n", status.Type)
		fmt.Printf("Description: %s\n", status.Description)
		fmt.Printf("Enabled: %v\n", status.Enabled)
		fmt.Printf("Connected: %v\n", status.Connected)
		if status.LastSeen != nil {
			fmt.Printf("Last Seen: %s\n", status.LastSeen.Format(time.RFC3339))
		}
		fmt.Printf("Tools: %d\n", status.ToolCount)
		fmt.Printf("Resources: %d\n", status.ResourceCount)
		fmt.Printf("Prompts: %d\n", status.PromptCount)
		return nil
	},
}

var mcpDiscoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover available MCP servers",
	Long:  "Discover available MCP servers from the official repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		manager := getMCPManager()

		fmt.Println("Discovering MCP servers from repository...")
		servers, err := manager.DiscoverServers()
		if err != nil {
			return fmt.Errorf("error discovering MCP servers: %w", err)
		}

		if len(servers) == 0 {
			fmt.Println("No MCP servers found in repository")
			return nil
		}

		fmt.Printf("Found %d MCP servers:\n\n", len(servers))

		for _, server := range servers {
			fmt.Printf("Name: %s\n", server.Name)
			fmt.Printf("Description: %s\n", server.Description)
			fmt.Printf("Author: %s\n", server.Author)
			fmt.Printf("Category: %s\n", server.Category)
			fmt.Printf("Language: %s\n", server.Language)
			fmt.Printf("Install Command: %s\n", server.InstallCmd)
			if server.GitHubURL != "" {
				fmt.Printf("GitHub: %s\n", server.GitHubURL)
			}
			if len(server.Tags) > 0 {
				fmt.Printf("Tags: %v\n", server.Tags)
			}
			fmt.Println()
		}
		return nil
	},
}

var mcpHealthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check MCP server health",
	Long:  "Check the health status of all MCP servers",
	RunE: func(cmd *cobra.Command, args []string) error {
		manager := getMCPManager()

		health := manager.HealthCheck()
		if len(health) == 0 {
			fmt.Println("No MCP servers configured")
			return nil
		}

		fmt.Printf("%-20s %s\n", "SERVER", "STATUS")
		fmt.Println(strings.Repeat("-", 30))

		for name, healthy := range health {
			status := "Unhealthy"
			if healthy {
				status = "Healthy"
			}
			fmt.Printf("%-20s %s\n", name, status)
		}
		return nil
	},
}

// getMCPManager creates an MCP manager instance
func getMCPManager() *mcp.MCPManager {
	configDir := filepath.Join(workingDir, ".codeforge")
	return mcp.NewMCPManager(configDir)
}

func init() {
	// Add subcommands
	mcpManageCmd.AddCommand(mcpListServersCmd)
	mcpManageCmd.AddCommand(mcpAddCmd)
	mcpManageCmd.AddCommand(mcpRemoveCmd)
	mcpManageCmd.AddCommand(mcpEnableCmd)
	mcpManageCmd.AddCommand(mcpDisableCmd)
	mcpManageCmd.AddCommand(mcpStatusCmd)
	mcpManageCmd.AddCommand(mcpDiscoverCmd)
	mcpManageCmd.AddCommand(mcpHealthCmd)

	// Add flags for add command
	mcpAddCmd.Flags().String("type", "", "Server type (local, remote, sse, http)")
	mcpAddCmd.Flags().String("description", "", "Server description")
	mcpAddCmd.Flags().StringSlice("command", []string{}, "Command to run (for local servers)")
	mcpAddCmd.Flags().String("url", "", "Server URL (for remote servers)")
	mcpAddCmd.Flags().Bool("enabled", false, "Enable server after adding")

	// Add manage command to mcp command
	mcpCmd.AddCommand(mcpManageCmd)
}
