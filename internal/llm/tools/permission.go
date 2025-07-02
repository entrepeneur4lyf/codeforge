package tools

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/config"
	"github.com/entrepeneur4lyf/codeforge/internal/permissions"
)

var ErrorPermissionDenied = errors.New("permission denied")

// CreatePermissionRequest interface
type CreatePermissionRequest struct {
	SessionID   string `json:"session_id"`
	ToolName    string `json:"tool_name"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Params      any    `json:"params"`
	Path        string `json:"path"`
}

// PermissionAdapter interface
type PermissionAdapter struct {
	service *permissions.PermissionService
}

// NewPermissionAdapter creates a new permission adapter
func NewPermissionAdapter(service *permissions.PermissionService) *PermissionAdapter {
	if service == nil {
		// Return a permissive adapter if no permission service is configured
		return &PermissionAdapter{service: nil}
	}
	return &PermissionAdapter{service: service}
}

// Request implements the simplified permission interface used by tools
func (p *PermissionAdapter) Request(opts CreatePermissionRequest) bool {
	// If no permission service, allow everything
	if p.service == nil {
		return true
	}

	ctx := context.Background()

	// Map tool actions to permission types
	permType := permissions.PermissionFileRead
	switch opts.Action {
	case "write", "create", "update":
		permType = permissions.PermissionFileWrite
	case "delete":
		permType = permissions.PermissionFileDelete
	case "execute", "run":
		permType = permissions.PermissionCodeExecute
	}

	// Ensure path is absolute
	path := opts.Path
	if !filepath.IsAbs(path) {
		path = filepath.Join(config.WorkingDirectory(), path)
	}

	// Create permission check
	check := &permissions.PermissionCheck{
		SessionID: opts.SessionID,
		Type:      permType,
		Resource:  path,
		Context: map[string]interface{}{
			"tool":        opts.ToolName,
			"action":      opts.Action,
			"description": opts.Description,
			"params":      opts.Params,
		},
	}

	// Check permission
	result, err := p.service.CheckPermission(ctx, check)
	if err != nil {
		return false
	}

	// If permission was denied but we can request it
	if !result.Allowed && result.RequiresApproval {
		// Create permission request
		req := &permissions.PermissionRequest{
			SessionID:   opts.SessionID,
			Type:        permType,
			Resource:    path,
			Reason:      opts.Description,
			Scope:       permissions.ScopeSession,
			RequestedAt: time.Now(),
			Context: map[string]interface{}{
				"tool":   opts.ToolName,
				"action": opts.Action,
				"params": opts.Params,
			},
		}

		// Request permission (this might block waiting for user approval)
		resp, err := p.service.RequestPermission(ctx, req)
		if err != nil {
			return false
		}

		return resp.Status == permissions.StatusApproved
	}

	return result.Allowed
}

// AutoApproveSession sets a session to auto-approve all permissions
func (p *PermissionAdapter) AutoApproveSession(sessionID string) {
	if p.service != nil {
		// Update session trust level for auto-approval
		p.service.UpdateSessionTrust(sessionID, true) // Mark session as trusted
	}
}
