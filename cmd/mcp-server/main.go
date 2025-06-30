package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/entrepeneur4lyf/codeforge/internal/config"
	"github.com/entrepeneur4lyf/codeforge/internal/mcp"
	"github.com/entrepeneur4lyf/codeforge/internal/vectordb"
)

func main() {
	var (
		workspaceRoot = flag.String("workspace", ".", "Workspace root directory")
		transport     = flag.String("transport", "stdio", "Transport type (stdio, sse, http)")
		addr          = flag.String("addr", ":8080", "Address for HTTP/SSE transport")
		dbPath        = flag.String("db", "", "Path to vector database (default: workspace/.codeforge/vector.db)")
	)
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*workspaceRoot, false)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Resolve workspace root
	absWorkspace, err := filepath.Abs(*workspaceRoot)
	if err != nil {
		log.Fatalf("Failed to resolve workspace path: %v", err)
	}

	// Set up database path
	if *dbPath == "" {
		*dbPath = filepath.Join(absWorkspace, ".codeforge", "vector.db")
	}

	// Ensure .codeforge directory exists
	codeforgeDir := filepath.Dir(*dbPath)
	if err := os.MkdirAll(codeforgeDir, 0755); err != nil {
		log.Fatalf("Failed to create .codeforge directory: %v", err)
	}

	// Initialize vector database
	if err := vectordb.Initialize(cfg); err != nil {
		log.Fatalf("Failed to initialize vector database: %v", err)
	}
	vdb := vectordb.Get()
	defer vdb.Close()

	// Create MCP server
	mcpServer := mcp.NewCodeForgeServer(cfg, vdb, absWorkspace)

	log.Printf("Starting CodeForge MCP server...")
	log.Printf("Workspace: %s", absWorkspace)
	log.Printf("Database: %s", *dbPath)
	log.Printf("Transport: %s", *transport)

	// Start server based on transport type
	switch *transport {
	case "stdio":
		if err := mcpServer.Start(); err != nil {
			log.Fatalf("Failed to start MCP server: %v", err)
		}
	case "sse":
		log.Printf("Starting SSE server on %s", *addr)
		if err := mcpServer.StartSSE(*addr); err != nil {
			log.Fatalf("Failed to start SSE server: %v", err)
		}
	case "http":
		log.Printf("Starting HTTP server on %s", *addr)
		if err := mcpServer.StartStreamableHTTP(*addr); err != nil {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	default:
		log.Fatalf("Unknown transport type: %s", *transport)
	}
}
