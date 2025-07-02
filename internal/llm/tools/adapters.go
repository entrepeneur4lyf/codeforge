package tools

import (
	"context"
	"net/http"
	"time"
	
	"github.com/entrepeneur4lyf/codeforge/internal/lsp"
)

// Tool adapter types that use the permission adapter

// EditToolAdapter wraps the edit tool with permission adapter
type EditToolAdapter struct {
	editTool
}

func NewEditToolAdapter(lspClients map[string]*lsp.Client, permissions *PermissionAdapter, history *HistoryService) BaseTool {
	return &EditToolAdapter{
		editTool: editTool{
			lspClients:  lspClients,
			permissions: permissions,
			history:     history,
		},
	}
}

// WriteToolAdapter wraps the write tool with permission adapter
type WriteToolAdapter struct {
	writeTool
}

func NewWriteToolAdapter(lspClients map[string]*lsp.Client, permissions *PermissionAdapter, history *HistoryService) BaseTool {
	return &WriteToolAdapter{
		writeTool: writeTool{
			lspClients:  lspClients,
			permissions: permissions,
			history:     history,
		},
	}
}

// BashToolAdapter wraps the bash tool with permission adapter
type BashToolAdapter struct {
	bashTool
}

func NewBashToolAdapter(permissions *PermissionAdapter) BaseTool {
	return &BashToolAdapter{
		bashTool: bashTool{
			permissions: permissions,
		},
	}
}

// PatchToolAdapter wraps the patch tool with permission adapter
type PatchToolAdapter struct {
	patchTool
}

func NewPatchToolAdapter(lspClients map[string]*lsp.Client, permissions *PermissionAdapter, history *HistoryService) BaseTool {
	return &PatchToolAdapter{
		patchTool: patchTool{
			lspClients:  lspClients,
			permissions: permissions,
			history:     history,
			workingDir:  "", // Will be set from config
		},
	}
}

// FetchToolAdapter wraps the fetch tool with permission adapter
type FetchToolAdapter struct {
	fetchTool
}

func NewFetchToolAdapter(permissions *PermissionAdapter) BaseTool {
	return &FetchToolAdapter{
		fetchTool: fetchTool{
			client: &http.Client{
				Timeout: 30 * time.Second,
			},
			permissions: permissions,
		},
	}
}

// Update the permission interfaces in tools to use the adapter

type permissionService interface {
	Request(opts CreatePermissionRequest) bool
}

type fileService interface {
	Create(ctx context.Context, sessionID, path, content string) (FileHistory, error)
	CreateVersion(ctx context.Context, sessionID, path, content string) (FileHistory, error)
	GetByPathAndSession(ctx context.Context, path, sessionID string) (FileHistory, error)
}