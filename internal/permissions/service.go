package permissions

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"sync"
	"time"
)

// PermissionService manages permissions and authorization
type PermissionService struct {
	config      *PermissionConfig
	permissions map[string]*Permission        // permission ID -> permission
	requests    map[string]*PermissionRequest // request ID -> request
	sessions    map[string]*SessionTrust      // session ID -> trust
	audit       []PermissionAuditEntry        // audit log
	policies    []PermissionPolicy            // permission policies
	mutex       sync.RWMutex

	// Callbacks
	onPermissionRequest func(*PermissionRequest) (*PermissionResponse, error)
	onPermissionUsed    func(*Permission, *PermissionCheck)
}

// NewPermissionService creates a new permission service
func NewPermissionService(config *PermissionConfig) *PermissionService {
	if config == nil {
		config = &PermissionConfig{
			DefaultScope:             ScopeSession,
			DefaultExpiration:        24 * time.Hour,
			RequireApproval:          true,
			AutoApproveThreshold:     80,
			MaxPermissionsPerSession: 100,
			AuditEnabled:             true,
			CleanupInterval:          1 * time.Hour,
		}
	}

	service := &PermissionService{
		config:      config,
		permissions: make(map[string]*Permission),
		requests:    make(map[string]*PermissionRequest),
		sessions:    make(map[string]*SessionTrust),
		audit:       make([]PermissionAuditEntry, 0),
		policies:    config.Policies,
	}

	// Start cleanup routine
	go service.cleanupRoutine()

	return service
}

// RequestPermission requests a new permission
func (ps *PermissionService) RequestPermission(ctx context.Context, req *PermissionRequest) (*PermissionResponse, error) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	// Generate request ID if not provided
	if req.ID == "" {
		req.ID = ps.generateID()
	}

	// Set request timestamp
	req.RequestedAt = time.Now()

	// Set expiration if not provided
	if req.ExpiresAt == nil && req.Scope == ScopeTemporary {
		expiry := time.Now().Add(ps.config.DefaultExpiration)
		req.ExpiresAt = &expiry
	}

	// Store the request
	ps.requests[req.ID] = req

	// Log the request
	ps.logAudit("request", req.Type, req.Resource, StatusPending, req.SessionID, req.Reason, req.Context)

	// Check if auto-approval is possible
	if ps.canAutoApprove(req) {
		response := &PermissionResponse{
			RequestID:   req.ID,
			Status:      StatusApproved,
			Reason:      "Auto-approved based on trust level",
			RespondedAt: time.Now(),
			RespondedBy: "auto",
		}

		// Grant the permission immediately
		_, err := ps.grantPermission(req, response)
		if err != nil {
			return nil, err
		}

		ps.logAudit("approve", req.Type, req.Resource, StatusApproved, req.SessionID, "Auto-approved", req.Context)
		log.Printf("Auto-approved permission %s for session %s", req.Type, req.SessionID)

		return response, nil
	}

	// Check policies
	policyResult := ps.checkPolicies(req)
	if policyResult == "deny" {
		response := &PermissionResponse{
			RequestID:   req.ID,
			Status:      StatusDenied,
			Reason:      "Denied by policy",
			RespondedAt: time.Now(),
			RespondedBy: "policy",
		}

		ps.logAudit("deny", req.Type, req.Resource, StatusDenied, req.SessionID, "Denied by policy", req.Context)
		return response, nil
	}

	// If we have a callback for handling requests, use it
	if ps.onPermissionRequest != nil {
		response, err := ps.onPermissionRequest(req)
		if err != nil {
			return nil, err
		}

		if response.Status == StatusApproved {
			_, err := ps.grantPermission(req, response)
			if err != nil {
				return nil, err
			}
		}

		return response, nil
	}

	// Otherwise, return pending status (requires manual approval)
	return &PermissionResponse{
		RequestID:   req.ID,
		Status:      StatusPending,
		Reason:      "Awaiting approval",
		RespondedAt: time.Now(),
		RespondedBy: "system",
	}, nil
}

// CheckPermission checks if an operation is permitted
func (ps *PermissionService) CheckPermission(ctx context.Context, check *PermissionCheck) (*PermissionCheckResult, error) {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	// Look for existing permission
	for _, permission := range ps.permissions {
		if permission.SessionID == check.SessionID &&
			permission.Type == check.Type &&
			ResourceMatches(permission.Resource, check.Resource) &&
			permission.CanUse() {

			// Use the permission
			permission.IncrementUsage()

			// Log usage
			ps.logAudit("use", check.Type, check.Resource, StatusApproved, check.SessionID, "Permission used", check.Context)

			// Callback for permission usage
			if ps.onPermissionUsed != nil {
				ps.onPermissionUsed(permission, check)
			}

			return &PermissionCheckResult{
				Allowed:    true,
				Permission: permission,
				Reason:     "Permission granted",
			}, nil
		}
	}

	// Check if auto-approval is possible
	if ps.canAutoApproveCheck(check) {
		return &PermissionCheckResult{
			Allowed:      true,
			AutoApproved: true,
			Reason:       "Auto-approved based on trust level",
		}, nil
	}

	// Check policies for immediate allow/deny
	req := &PermissionRequest{
		SessionID: check.SessionID,
		Type:      check.Type,
		Resource:  check.Resource,
		Context:   check.Context,
	}

	policyResult := ps.checkPolicies(req)
	if policyResult == "allow" {
		return &PermissionCheckResult{
			Allowed: true,
			Reason:  "Allowed by policy",
		}, nil
	}

	if policyResult == "deny" {
		ps.logAudit("deny", check.Type, check.Resource, StatusDenied, check.SessionID, "Denied by policy", check.Context)
		return &PermissionCheckResult{
			Allowed: false,
			Reason:  "Denied by policy",
		}, nil
	}

	// Permission not found and requires approval
	return &PermissionCheckResult{
		Allowed:          false,
		RequiresApproval: true,
		Reason:           "Permission required",
	}, nil
}

// ApproveRequest approves a permission request
func (ps *PermissionService) ApproveRequest(ctx context.Context, requestID string, approverID string, conditions []string) (*Permission, error) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	req, exists := ps.requests[requestID]
	if !exists {
		return nil, ErrPermissionNotFound
	}

	if req.IsExpired() {
		return nil, ErrPermissionExpired
	}

	response := &PermissionResponse{
		RequestID:   requestID,
		Status:      StatusApproved,
		Conditions:  conditions,
		RespondedAt: time.Now(),
		RespondedBy: approverID,
	}

	permission, err := ps.grantPermission(req, response)
	if err != nil {
		return nil, err
	}

	ps.logAudit("approve", req.Type, req.Resource, StatusApproved, req.SessionID, "Manually approved", req.Context)
	log.Printf("Approved permission %s for session %s by %s", req.Type, req.SessionID, approverID)

	return permission, nil
}

// DenyRequest denies a permission request
func (ps *PermissionService) DenyRequest(ctx context.Context, requestID string, denierID string, reason string) error {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	req, exists := ps.requests[requestID]
	if !exists {
		return ErrPermissionNotFound
	}

	ps.logAudit("deny", req.Type, req.Resource, StatusDenied, req.SessionID, reason, req.Context)
	log.Printf("Denied permission %s for session %s by %s: %s", req.Type, req.SessionID, denierID, reason)

	// Remove the request
	delete(ps.requests, requestID)

	return nil
}

// RevokePermission revokes an existing permission
func (ps *PermissionService) RevokePermission(ctx context.Context, permissionID string, revokerID string, reason string) error {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	permission, exists := ps.permissions[permissionID]
	if !exists {
		return ErrPermissionNotFound
	}

	permission.Status = StatusRevoked

	ps.logAudit("revoke", permission.Type, permission.Resource, StatusRevoked, permission.SessionID, reason, nil)
	log.Printf("Revoked permission %s for session %s by %s: %s", permission.Type, permission.SessionID, revokerID, reason)

	return nil
}

// GetSessionTrust gets or creates session trust information
func (ps *PermissionService) GetSessionTrust(sessionID string) *SessionTrust {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	trust, exists := ps.sessions[sessionID]
	if !exists {
		trust = &SessionTrust{
			SessionID:      sessionID,
			TrustLevel:     50, // Default trust level
			AutoApprove:    []PermissionType{PermissionFileRead, PermissionSystemInfo},
			CreatedAt:      time.Now(),
			LastActivity:   time.Now(),
			SuccessCount:   0,
			ViolationCount: 0,
		}
		ps.sessions[sessionID] = trust
	}

	return trust
}

// UpdateSessionTrust updates session trust based on behavior
func (ps *PermissionService) UpdateSessionTrust(sessionID string, success bool) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	trust := ps.GetSessionTrust(sessionID)
	trust.LastActivity = time.Now()

	if success {
		trust.SuccessCount++
		// Increase trust level gradually
		if trust.TrustLevel < 100 {
			trust.TrustLevel = min(100, trust.TrustLevel+1)
		}
	} else {
		trust.ViolationCount++
		// Decrease trust level more aggressively
		trust.TrustLevel = max(0, trust.TrustLevel-5)
	}

	log.Printf("Updated trust for session %s: level=%d, success=%d, violations=%d",
		sessionID, trust.TrustLevel, trust.SuccessCount, trust.ViolationCount)
}

// Helper methods

func (ps *PermissionService) grantPermission(req *PermissionRequest, response *PermissionResponse) (*Permission, error) {
	permission := &Permission{
		ID:         ps.generateID(),
		SessionID:  req.SessionID,
		Type:       req.Type,
		Resource:   req.Resource,
		Scope:      req.Scope,
		Status:     StatusApproved,
		Conditions: response.Conditions,
		GrantedAt:  time.Now(),
		ExpiresAt:  req.ExpiresAt,
		UsageCount: 0,
		GrantedBy:  response.RespondedBy,
		Metadata:   req.Metadata,
	}

	// Set max usage based on scope
	switch req.Scope {
	case ScopeOneTime:
		permission.MaxUsage = 1
	case ScopeTemporary:
		permission.MaxUsage = 10 // Reasonable limit for temporary permissions
	default:
		permission.MaxUsage = 0 // Unlimited
	}

	ps.permissions[permission.ID] = permission

	// Remove the request
	delete(ps.requests, req.ID)

	return permission, nil
}

func (ps *PermissionService) canAutoApprove(req *PermissionRequest) bool {
	if !ps.config.RequireApproval {
		return true
	}

	trust := ps.GetSessionTrust(req.SessionID)
	if trust.TrustLevel >= ps.config.AutoApproveThreshold {
		// Check if this permission type is in auto-approve list
		for _, autoType := range trust.AutoApprove {
			if autoType == req.Type {
				return true
			}
		}
	}

	// Auto-approve low-risk operations for trusted sessions
	if trust.TrustLevel >= 70 && GetRiskLevel(req.Type) <= 3 {
		return true
	}

	return false
}

func (ps *PermissionService) canAutoApproveCheck(check *PermissionCheck) bool {
	trust := ps.GetSessionTrust(check.SessionID)

	// Auto-approve very low-risk operations
	if GetRiskLevel(check.Type) <= 2 {
		return true
	}

	// Auto-approve for highly trusted sessions
	if trust.TrustLevel >= 90 && GetRiskLevel(check.Type) <= 5 {
		return true
	}

	return false
}

func (ps *PermissionService) checkPolicies(req *PermissionRequest) string {
	// Sort policies by priority (higher priority first)
	policies := make([]PermissionPolicy, len(ps.policies))
	copy(policies, ps.policies)

	for _, policy := range policies {
		if !policy.Enabled {
			continue
		}

		for _, rule := range policy.Rules {
			if !rule.Enabled {
				continue
			}

			if rule.Type == req.Type && ResourceMatches(rule.Resource, req.Resource) {
				// Check conditions
				if ps.evaluateConditions(rule.Conditions, req) {
					return rule.Action
				}
			}
		}
	}

	return "require_approval" // Default action
}

func (ps *PermissionService) evaluateConditions(conditions []RuleCondition, req *PermissionRequest) bool {
	for _, condition := range conditions {
		if !ps.evaluateCondition(condition, req) {
			return false
		}
	}
	return true
}

func (ps *PermissionService) evaluateCondition(condition RuleCondition, req *PermissionRequest) bool {
	// Simple condition evaluation - can be extended
	switch condition.Field {
	case "session_id":
		return condition.Value == req.SessionID
	case "trust_level":
		trust := ps.GetSessionTrust(req.SessionID)
		switch condition.Operator {
		case "gt":
			if threshold, ok := condition.Value.(float64); ok {
				return trust.TrustLevel > int(threshold)
			}
		case "lt":
			if threshold, ok := condition.Value.(float64); ok {
				return trust.TrustLevel < int(threshold)
			}
		}
	}
	return true
}

func (ps *PermissionService) logAudit(action string, permType PermissionType, resource string, status PermissionStatus, sessionID string, reason string, context map[string]interface{}) {
	if !ps.config.AuditEnabled {
		return
	}

	entry := PermissionAuditEntry{
		ID:        ps.generateID(),
		SessionID: sessionID,
		Action:    action,
		Type:      permType,
		Resource:  resource,
		Status:    status,
		Reason:    reason,
		Context:   context,
		Timestamp: time.Now(),
	}

	ps.audit = append(ps.audit, entry)

	// Keep audit log size manageable
	if len(ps.audit) > 10000 {
		ps.audit = ps.audit[1000:] // Keep last 9000 entries
	}
}

func (ps *PermissionService) generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (ps *PermissionService) cleanupRoutine() {
	ticker := time.NewTicker(ps.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		ps.cleanup()
	}
}

func (ps *PermissionService) cleanup() {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	now := time.Now()

	// Clean up expired permissions
	for id, permission := range ps.permissions {
		if permission.IsExpired() || permission.Status == StatusRevoked {
			delete(ps.permissions, id)
		}
	}

	// Clean up expired requests
	for id, request := range ps.requests {
		if request.IsExpired() {
			delete(ps.requests, id)
		}
	}

	// Clean up old sessions (inactive for more than 24 hours)
	for id, session := range ps.sessions {
		if now.Sub(session.LastActivity) > 24*time.Hour {
			delete(ps.sessions, id)
		}
	}

	log.Printf("Permission cleanup completed: %d permissions, %d requests, %d sessions",
		len(ps.permissions), len(ps.requests), len(ps.sessions))
}

// Utility functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
