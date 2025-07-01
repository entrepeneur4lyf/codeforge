package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/entrepeneur4lyf/codeforge/internal/config"
	"github.com/entrepeneur4lyf/codeforge/internal/vectordb"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestCodeForgeServer_Creation(t *testing.T) {
	// Create temporary workspace
	tempDir, err := os.MkdirTemp("", "codeforge-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Load config
	cfg, err := config.Load(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Initialize vector database
	if err := vectordb.Initialize(cfg); err != nil {
		t.Fatalf("Failed to initialize vector database: %v", err)
	}
	vdb := vectordb.Get()
	defer vdb.Close()

	// Create MCP server
	server := NewCodeForgeServer(cfg, vdb, tempDir)
	if server == nil {
		t.Fatal("Failed to create MCP server")
	}

	// Verify server has the expected capabilities
	mcpServer := server.GetServer()
	if mcpServer == nil {
		t.Fatal("MCP server instance is nil")
	}

	t.Log("MCP server created successfully")
}

func TestCodeForgeServer_Tools(t *testing.T) {
	// Create temporary workspace
	tempDir, err := os.MkdirTemp("", "codeforge-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test file
	testFile := filepath.Join(tempDir, "test.go")
	testContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
`
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Load config
	cfg, err := config.Load(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Initialize vector database
	if err := vectordb.Initialize(cfg); err != nil {
		t.Fatalf("Failed to initialize vector database: %v", err)
	}
	vdb := vectordb.Get()
	defer vdb.Close()

	// Create MCP server
	server := NewCodeForgeServer(cfg, vdb, tempDir)

	ctx := context.Background()

	// Test read_file tool
	t.Run("read_file", func(t *testing.T) {
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "read_file",
				Arguments: map[string]interface{}{
					"path": "test.go",
				},
			},
		}

		result, err := server.handleReadFile(ctx, request)
		if err != nil {
			t.Fatalf("read_file failed: %v", err)
		}

		if result == nil {
			t.Fatal("read_file returned nil result")
		}

		// Check that the content matches
		if len(result.Content) == 0 {
			t.Fatal("read_file returned empty content")
		}

		t.Log("read_file tool works correctly")
	})

	// Test write_file tool
	t.Run("write_file", func(t *testing.T) {
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "write_file",
				Arguments: map[string]interface{}{
					"path":    "new_test.go",
					"content": "package test\n\n// Test file\n",
				},
			},
		}

		result, err := server.handleWriteFile(ctx, request)
		if err != nil {
			t.Fatalf("write_file failed: %v", err)
		}

		if result == nil {
			t.Fatal("write_file returned nil result")
		}

		// Verify file was created
		newFile := filepath.Join(tempDir, "new_test.go")
		if _, err := os.Stat(newFile); os.IsNotExist(err) {
			t.Fatal("write_file did not create the file")
		}

		t.Log("write_file tool works correctly")
	})

	// Test get_project_structure tool
	t.Run("get_project_structure", func(t *testing.T) {
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "get_project_structure",
				Arguments: map[string]interface{}{
					"path":      ".",
					"max_depth": 2,
				},
			},
		}

		result, err := server.handleProjectStructure(ctx, request)
		if err != nil {
			t.Fatalf("get_project_structure failed: %v", err)
		}

		if result == nil {
			t.Fatal("get_project_structure returned nil result")
		}

		t.Log("get_project_structure tool works correctly")
	})

	// Test semantic_search tool
	t.Run("semantic_search", func(t *testing.T) {
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "semantic_search",
				Arguments: map[string]interface{}{
					"query":       "hello world function",
					"max_results": 5,
					"language":    "go",
				},
			},
		}

		result, err := server.handleSemanticSearch(ctx, request)
		if err != nil {
			t.Fatalf("semantic_search failed: %v", err)
		}

		if result == nil {
			t.Fatal("semantic_search returned nil result")
		}

		t.Log("semantic_search tool works correctly")
	})
}

func TestCodeForgeServer_Resources(t *testing.T) {
	// Create temporary workspace
	tempDir, err := os.MkdirTemp("", "codeforge-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Load config
	cfg, err := config.Load(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Initialize vector database
	if err := vectordb.Initialize(cfg); err != nil {
		t.Fatalf("Failed to initialize vector database: %v", err)
	}
	vdb := vectordb.Get()
	defer vdb.Close()

	// Create MCP server
	server := NewCodeForgeServer(cfg, vdb, tempDir)

	ctx := context.Background()

	// Test project metadata resource
	t.Run("project_metadata", func(t *testing.T) {
		request := mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{
				URI: "codeforge://project/metadata",
			},
		}

		result, err := server.handleProjectMetadata(ctx, request)
		if err != nil {
			t.Fatalf("project metadata resource failed: %v", err)
		}

		if len(result) == 0 {
			t.Fatal("project metadata returned empty result")
		}

		t.Log("project metadata resource works correctly")
	})

	// Test git status resource
	t.Run("git_status", func(t *testing.T) {
		request := mcp.ReadResourceRequest{
			Params: mcp.ReadResourceParams{
				URI: "codeforge://git/status",
			},
		}

		result, err := server.handleGitStatus(ctx, request)
		if err != nil {
			t.Fatalf("git status resource failed: %v", err)
		}

		if len(result) == 0 {
			t.Fatal("git status returned empty result")
		}

		t.Log("git status resource works correctly")
	})
}

func TestCodeForgeServer_PathValidation(t *testing.T) {
	// Create temporary workspace
	tempDir, err := os.MkdirTemp("", "codeforge-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Load config
	cfg, err := config.Load(tempDir, false)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Initialize vector database
	if err := vectordb.Initialize(cfg); err != nil {
		t.Fatalf("Failed to initialize vector database: %v", err)
	}
	vdb := vectordb.Get()
	defer vdb.Close()

	// Create MCP server
	server := NewCodeForgeServer(cfg, vdb, tempDir)

	// Test valid path
	t.Run("valid_path", func(t *testing.T) {
		validPath, err := server.validatePath("test.go")
		if err != nil {
			t.Fatalf("Valid path validation failed: %v", err)
		}
		if validPath == "" {
			t.Fatal("Valid path returned empty string")
		}
		t.Log("Valid path validation works")
	})

	// Test path traversal attack
	t.Run("path_traversal", func(t *testing.T) {
		_, err := server.validatePath("../../../etc/passwd")
		if err == nil {
			t.Fatal("Path traversal attack was not prevented")
		}
		t.Log("Path traversal prevention works")
	})

	// Test absolute path outside workspace
	t.Run("absolute_path_outside", func(t *testing.T) {
		_, err := server.validatePath("/etc/passwd")
		if err == nil {
			t.Fatal("Absolute path outside workspace was not prevented")
		}
		t.Log("Absolute path prevention works")
	})
}
