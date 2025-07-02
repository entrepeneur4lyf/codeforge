package tools

import (
	"github.com/entrepeneur4lyf/codeforge/internal/lsp"
	"github.com/entrepeneur4lyf/codeforge/internal/permissions"
)

// ToolRegistry holds all available tools
type ToolRegistry struct {
	tools map[string]BaseTool
}

// NewToolRegistry creates a new tool registry with all available tools
func NewToolRegistry(lspManager *lsp.Manager, permissionService *permissions.PermissionService) *ToolRegistry {
	// Create adapters
	permAdapter := NewPermissionAdapter(permissionService)
	historyService := NewHistoryService()
	
	// Get LSP clients
	lspClients := make(map[string]*lsp.Client)
	if lspManager != nil {
		for lang, client := range lspManager.GetAllClients() {
			lspClients[lang] = client
		}
	}
	
	// Create tools
	tools := map[string]BaseTool{
		ViewToolName:        NewViewTool(lspClients),
		EditToolName:        NewEditToolAdapter(lspClients, permAdapter, historyService),
		WriteToolName:       NewWriteToolAdapter(lspClients, permAdapter, historyService),
		BashToolName:        NewBashToolAdapter(permAdapter),
		DiagnosticsToolName: NewDiagnosticsTool(lspClients),
		GlobToolName:        NewGlobTool(),
		GrepToolName:        NewGrepTool(),
		LSToolName:          NewLsTool(),
		PatchToolName:       NewPatchToolAdapter(lspClients, permAdapter, historyService),
		FetchToolName:       NewFetchToolAdapter(permAdapter),
	}
	
	return &ToolRegistry{
		tools: tools,
	}
}

// GetTool returns a tool by name
func (r *ToolRegistry) GetTool(name string) (BaseTool, bool) {
	tool, exists := r.tools[name]
	return tool, exists
}

// GetAllTools returns all registered tools
func (r *ToolRegistry) GetAllTools() map[string]BaseTool {
	return r.tools
}

// GetToolInfos returns information about all tools
func (r *ToolRegistry) GetToolInfos() []ToolInfo {
	var infos []ToolInfo
	for _, tool := range r.tools {
		infos = append(infos, tool.Info())
	}
	return infos
}