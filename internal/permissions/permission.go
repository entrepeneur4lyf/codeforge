package permissions

import (
	"errors"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/config"
	"github.com/entrepeneur4lyf/codeforge/internal/pubsub"
	"github.com/google/uuid"
)

var ErrorPermissionDenied = errors.New("permission denied")

type CreatePermissionRequest struct {
	SessionID   string `json:"session_id"`
	ToolName    string `json:"tool_name"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Params      any    `json:"params"`
	Path        string `json:"path"`
}

type Service interface {
	GrantPersistent(permission PermissionRequest)
	Grant(permission PermissionRequest)
	Deny(permission PermissionRequest)
	Request(opts CreatePermissionRequest) bool
	AutoApproveSession(sessionID string)
	Publish(eventType pubsub.EventType, permission PermissionRequest)
}

type permissionService struct {
	*pubsub.Broker[PermissionRequest]

	sessionPermissions  []PermissionRequest
	pendingRequests     sync.Map
	autoApproveSessions []string
}

func (s *permissionService) GrantPersistent(permission PermissionRequest) {
	respCh, ok := s.pendingRequests.Load(permission.ID)
	if ok {
		respCh.(chan bool) <- true
	}
	s.sessionPermissions = append(s.sessionPermissions, permission)
}

func (s *permissionService) Grant(permission PermissionRequest) {
	respCh, ok := s.pendingRequests.Load(permission.ID)
	if ok {
		respCh.(chan bool) <- true
	}
}

func (s *permissionService) Deny(permission PermissionRequest) {
	respCh, ok := s.pendingRequests.Load(permission.ID)
	if ok {
		respCh.(chan bool) <- false
	}
}

func (s *permissionService) Request(opts CreatePermissionRequest) bool {
	if slices.Contains(s.autoApproveSessions, opts.SessionID) {
		return true
	}
	dir := filepath.Dir(opts.Path)
	if dir == "." {
		dir = config.WorkingDirectory()
	}
	permission := PermissionRequest{
		ID:        uuid.New().String(),
		SessionID: opts.SessionID,
		Type:      PermissionFileRead, // Default to file read, should be determined by action
		Resource:  dir,
		Scope:     ScopeSession,
		Reason:    opts.Description,
		Context: map[string]interface{}{
			"tool":   opts.ToolName,
			"action": opts.Action,
			"params": opts.Params,
		},
		RequestedAt: time.Now(),
	}

	for _, p := range s.sessionPermissions {
		toolName, _ := p.Context["tool"].(string)
		action, _ := p.Context["action"].(string)
		permToolName, _ := permission.Context["tool"].(string)
		permAction, _ := permission.Context["action"].(string)
		if toolName == permToolName && action == permAction && p.SessionID == permission.SessionID && p.Resource == permission.Resource {
			return true
		}
	}

	respCh := make(chan bool, 1)

	s.pendingRequests.Store(permission.ID, respCh)
	defer s.pendingRequests.Delete(permission.ID)

	s.Publish(pubsub.CreatedEvent, permission)

	// Wait for the response with a timeout
	resp := <-respCh
	return resp
}

func (s *permissionService) AutoApproveSession(sessionID string) {
	s.autoApproveSessions = append(s.autoApproveSessions, sessionID)
}

func (s *permissionService) Publish(eventType pubsub.EventType, permission PermissionRequest) {
	s.Broker.Publish(eventType, permission)
}
