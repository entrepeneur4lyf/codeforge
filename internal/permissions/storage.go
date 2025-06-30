package permissions

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// PermissionStorage handles persistent storage of permissions
type PermissionStorage struct {
	db *sql.DB
}

// NewPermissionStorage creates a new permission storage
func NewPermissionStorage(dbPath string) (*PermissionStorage, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	storage := &PermissionStorage{db: db}

	if err := storage.initTables(); err != nil {
		return nil, fmt.Errorf("failed to initialize tables: %w", err)
	}

	return storage, nil
}

// initTables creates the necessary database tables
func (ps *PermissionStorage) initTables() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS permissions (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			type TEXT NOT NULL,
			resource TEXT NOT NULL,
			scope TEXT NOT NULL,
			status TEXT NOT NULL,
			conditions TEXT,
			granted_at DATETIME NOT NULL,
			expires_at DATETIME,
			usage_count INTEGER DEFAULT 0,
			max_usage INTEGER DEFAULT 0,
			last_used_at DATETIME,
			granted_by TEXT NOT NULL,
			metadata TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS permission_requests (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			type TEXT NOT NULL,
			resource TEXT NOT NULL,
			scope TEXT NOT NULL,
			reason TEXT,
			context TEXT,
			requested_at DATETIME NOT NULL,
			expires_at DATETIME,
			metadata TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS session_trust (
			session_id TEXT PRIMARY KEY,
			trust_level INTEGER NOT NULL,
			auto_approve TEXT,
			created_at DATETIME NOT NULL,
			last_activity DATETIME NOT NULL,
			success_count INTEGER DEFAULT 0,
			violation_count INTEGER DEFAULT 0,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,

		`CREATE TABLE IF NOT EXISTS permission_audit (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			action TEXT NOT NULL,
			type TEXT NOT NULL,
			resource TEXT NOT NULL,
			status TEXT NOT NULL,
			user_id TEXT,
			reason TEXT,
			context TEXT,
			timestamp DATETIME NOT NULL,
			ip_address TEXT,
			user_agent TEXT
		)`,

		`CREATE INDEX IF NOT EXISTS idx_permissions_session ON permissions(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_permissions_type ON permissions(type)`,
		`CREATE INDEX IF NOT EXISTS idx_permissions_status ON permissions(status)`,
		`CREATE INDEX IF NOT EXISTS idx_permissions_expires ON permissions(expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_requests_session ON permission_requests(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_session ON permission_audit(session_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON permission_audit(timestamp)`,
	}

	for _, query := range queries {
		if _, err := ps.db.Exec(query); err != nil {
			return fmt.Errorf("failed to execute query: %w", err)
		}
	}

	return nil
}

// SavePermission saves a permission to persistent storage
func (ps *PermissionStorage) SavePermission(ctx context.Context, permission *Permission) error {
	conditionsJSON, _ := json.Marshal(permission.Conditions)
	metadataJSON, _ := json.Marshal(permission.Metadata)

	query := `INSERT OR REPLACE INTO permissions 
		(id, session_id, type, resource, scope, status, conditions, granted_at, expires_at, 
		 usage_count, max_usage, last_used_at, granted_by, metadata, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`

	_, err := ps.db.ExecContext(ctx, query,
		permission.ID,
		permission.SessionID,
		string(permission.Type),
		permission.Resource,
		string(permission.Scope),
		string(permission.Status),
		string(conditionsJSON),
		permission.GrantedAt,
		permission.ExpiresAt,
		permission.UsageCount,
		permission.MaxUsage,
		permission.LastUsedAt,
		permission.GrantedBy,
		string(metadataJSON),
	)

	return err
}

// LoadPermission loads a permission from persistent storage
func (ps *PermissionStorage) LoadPermission(ctx context.Context, id string) (*Permission, error) {
	query := `SELECT id, session_id, type, resource, scope, status, conditions, granted_at, 
		expires_at, usage_count, max_usage, last_used_at, granted_by, metadata
		FROM permissions WHERE id = ?`

	row := ps.db.QueryRowContext(ctx, query, id)

	var permission Permission
	var conditionsJSON, metadataJSON string
	var expiresAt, lastUsedAt sql.NullTime

	err := row.Scan(
		&permission.ID,
		&permission.SessionID,
		&permission.Type,
		&permission.Resource,
		&permission.Scope,
		&permission.Status,
		&conditionsJSON,
		&permission.GrantedAt,
		&expiresAt,
		&permission.UsageCount,
		&permission.MaxUsage,
		&lastUsedAt,
		&permission.GrantedBy,
		&metadataJSON,
	)

	if err != nil {
		return nil, err
	}

	// Parse JSON fields
	if conditionsJSON != "" {
		json.Unmarshal([]byte(conditionsJSON), &permission.Conditions)
	}
	if metadataJSON != "" {
		json.Unmarshal([]byte(metadataJSON), &permission.Metadata)
	}

	// Handle nullable timestamps
	if expiresAt.Valid {
		permission.ExpiresAt = &expiresAt.Time
	}
	if lastUsedAt.Valid {
		permission.LastUsedAt = &lastUsedAt.Time
	}

	return &permission, nil
}

// LoadPermissionsBySession loads all permissions for a session
func (ps *PermissionStorage) LoadPermissionsBySession(ctx context.Context, sessionID string) ([]*Permission, error) {
	query := `SELECT id, session_id, type, resource, scope, status, conditions, granted_at, 
		expires_at, usage_count, max_usage, last_used_at, granted_by, metadata
		FROM permissions WHERE session_id = ? AND status = 'approved'`

	rows, err := ps.db.QueryContext(ctx, query, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []*Permission
	for rows.Next() {
		var permission Permission
		var conditionsJSON, metadataJSON string
		var expiresAt, lastUsedAt sql.NullTime

		err := rows.Scan(
			&permission.ID,
			&permission.SessionID,
			&permission.Type,
			&permission.Resource,
			&permission.Scope,
			&permission.Status,
			&conditionsJSON,
			&permission.GrantedAt,
			&expiresAt,
			&permission.UsageCount,
			&permission.MaxUsage,
			&lastUsedAt,
			&permission.GrantedBy,
			&metadataJSON,
		)

		if err != nil {
			return nil, err
		}

		// Parse JSON fields
		if conditionsJSON != "" {
			json.Unmarshal([]byte(conditionsJSON), &permission.Conditions)
		}
		if metadataJSON != "" {
			json.Unmarshal([]byte(metadataJSON), &permission.Metadata)
		}

		// Handle nullable timestamps
		if expiresAt.Valid {
			permission.ExpiresAt = &expiresAt.Time
		}
		if lastUsedAt.Valid {
			permission.LastUsedAt = &lastUsedAt.Time
		}

		permissions = append(permissions, &permission)
	}

	return permissions, rows.Err()
}

// SaveSessionTrust saves session trust information
func (ps *PermissionStorage) SaveSessionTrust(ctx context.Context, trust *SessionTrust) error {
	autoApproveJSON, _ := json.Marshal(trust.AutoApprove)

	query := `INSERT OR REPLACE INTO session_trust 
		(session_id, trust_level, auto_approve, created_at, last_activity, 
		 success_count, violation_count, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`

	_, err := ps.db.ExecContext(ctx, query,
		trust.SessionID,
		trust.TrustLevel,
		string(autoApproveJSON),
		trust.CreatedAt,
		trust.LastActivity,
		trust.SuccessCount,
		trust.ViolationCount,
	)

	return err
}

// LoadSessionTrust loads session trust information
func (ps *PermissionStorage) LoadSessionTrust(ctx context.Context, sessionID string) (*SessionTrust, error) {
	query := `SELECT session_id, trust_level, auto_approve, created_at, last_activity, 
		success_count, violation_count FROM session_trust WHERE session_id = ?`

	row := ps.db.QueryRowContext(ctx, query, sessionID)

	var trust SessionTrust
	var autoApproveJSON string

	err := row.Scan(
		&trust.SessionID,
		&trust.TrustLevel,
		&autoApproveJSON,
		&trust.CreatedAt,
		&trust.LastActivity,
		&trust.SuccessCount,
		&trust.ViolationCount,
	)

	if err != nil {
		return nil, err
	}

	// Parse auto-approve list
	if autoApproveJSON != "" {
		json.Unmarshal([]byte(autoApproveJSON), &trust.AutoApprove)
	}

	return &trust, nil
}

// SaveAuditEntry saves an audit entry
func (ps *PermissionStorage) SaveAuditEntry(ctx context.Context, entry *PermissionAuditEntry) error {
	contextJSON, _ := json.Marshal(entry.Context)

	query := `INSERT INTO permission_audit 
		(id, session_id, action, type, resource, status, user_id, reason, context, 
		 timestamp, ip_address, user_agent)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := ps.db.ExecContext(ctx, query,
		entry.ID,
		entry.SessionID,
		entry.Action,
		string(entry.Type),
		entry.Resource,
		string(entry.Status),
		entry.UserID,
		entry.Reason,
		string(contextJSON),
		entry.Timestamp,
		entry.IPAddress,
		entry.UserAgent,
	)

	return err
}

// CleanupExpired removes expired permissions and requests
func (ps *PermissionStorage) CleanupExpired(ctx context.Context) error {
	now := time.Now()

	// Clean up expired permissions
	_, err := ps.db.ExecContext(ctx,
		"DELETE FROM permissions WHERE expires_at IS NOT NULL AND expires_at < ?", now)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired permissions: %w", err)
	}

	// Clean up expired requests
	_, err = ps.db.ExecContext(ctx,
		"DELETE FROM permission_requests WHERE expires_at IS NOT NULL AND expires_at < ?", now)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired requests: %w", err)
	}

	// Clean up old audit entries (keep last 30 days)
	cutoff := now.AddDate(0, 0, -30)
	_, err = ps.db.ExecContext(ctx,
		"DELETE FROM permission_audit WHERE timestamp < ?", cutoff)
	if err != nil {
		return fmt.Errorf("failed to cleanup old audit entries: %w", err)
	}

	log.Printf("Permission storage cleanup completed")
	return nil
}

// GetPermissionStats returns permission statistics
func (ps *PermissionStorage) GetPermissionStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Count active permissions
	var activeCount int
	err := ps.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM permissions WHERE status = 'approved' AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)").Scan(&activeCount)
	if err != nil {
		return nil, err
	}
	stats["active_permissions"] = activeCount

	// Count pending requests
	var pendingCount int
	err = ps.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM permission_requests WHERE expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP").Scan(&pendingCount)
	if err != nil {
		return nil, err
	}
	stats["pending_requests"] = pendingCount

	// Count sessions with trust data
	var sessionCount int
	err = ps.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM session_trust").Scan(&sessionCount)
	if err != nil {
		return nil, err
	}
	stats["trusted_sessions"] = sessionCount

	// Count audit entries from last 24 hours
	var recentAuditCount int
	err = ps.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM permission_audit WHERE timestamp > datetime('now', '-1 day')").Scan(&recentAuditCount)
	if err != nil {
		return nil, err
	}
	stats["recent_audit_entries"] = recentAuditCount

	return stats, nil
}

// Close closes the database connection
func (ps *PermissionStorage) Close() error {
	return ps.db.Close()
}

// PersistentPermissionManager manages persistent permissions
type PersistentPermissionManager struct {
	storage *PermissionStorage
	service *PermissionService
}

// NewPersistentPermissionManager creates a new persistent permission manager
func NewPersistentPermissionManager(storage *PermissionStorage, service *PermissionService) *PersistentPermissionManager {
	return &PersistentPermissionManager{
		storage: storage,
		service: service,
	}
}

// LoadSessionPermissions loads persistent permissions for a session
func (ppm *PersistentPermissionManager) LoadSessionPermissions(ctx context.Context, sessionID string) error {
	// Load permissions from storage
	permissions, err := ppm.storage.LoadPermissionsBySession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to load session permissions: %w", err)
	}

	// Add them to the service
	for _, permission := range permissions {
		// Only load non-expired permissions
		if permission.CanUse() {
			ppm.service.permissions[permission.ID] = permission
			log.Printf("Loaded persistent permission %s for session %s", permission.Type, sessionID)
		}
	}

	// Load session trust
	trust, err := ppm.storage.LoadSessionTrust(ctx, sessionID)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to load session trust: %w", err)
	}

	if trust != nil {
		ppm.service.sessions[sessionID] = trust
		log.Printf("Loaded session trust for %s: level=%d", sessionID, trust.TrustLevel)
	}

	return nil
}

// SaveSessionPermissions saves session permissions to persistent storage
func (ppm *PersistentPermissionManager) SaveSessionPermissions(ctx context.Context, sessionID string) error {
	// Save permissions
	for _, permission := range ppm.service.permissions {
		if permission.SessionID == sessionID && permission.Scope == ScopePersistent {
			if err := ppm.storage.SavePermission(ctx, permission); err != nil {
				log.Printf("Failed to save permission %s: %v", permission.ID, err)
			}
		}
	}

	// Save session trust
	if trust, exists := ppm.service.sessions[sessionID]; exists {
		if err := ppm.storage.SaveSessionTrust(ctx, trust); err != nil {
			log.Printf("Failed to save session trust for %s: %v", sessionID, err)
		}
	}

	return nil
}

// PromoteTopersistent promotes a session permission to persistent
func (ppm *PersistentPermissionManager) PromoteToersistent(ctx context.Context, permissionID string) error {
	permission, exists := ppm.service.permissions[permissionID]
	if !exists {
		return ErrPermissionNotFound
	}

	// Change scope to persistent
	permission.Scope = ScopePersistent
	permission.ExpiresAt = nil // Persistent permissions don't expire

	// Save to storage
	return ppm.storage.SavePermission(ctx, permission)
}

// DemoteToSession demotes a persistent permission to session-only
func (ppm *PersistentPermissionManager) DemoteToSession(ctx context.Context, permissionID string) error {
	permission, exists := ppm.service.permissions[permissionID]
	if !exists {
		return ErrPermissionNotFound
	}

	// Change scope to session
	permission.Scope = ScopeSession

	// Remove from persistent storage
	_, err := ppm.storage.db.ExecContext(ctx, "DELETE FROM permissions WHERE id = ?", permissionID)
	return err
}

// GetPersistentPermissions returns all persistent permissions for a session
func (ppm *PersistentPermissionManager) GetPersistentPermissions(ctx context.Context, sessionID string) ([]*Permission, error) {
	return ppm.storage.LoadPermissionsBySession(ctx, sessionID)
}

// CleanupExpiredPermissions removes expired permissions from storage
func (ppm *PersistentPermissionManager) CleanupExpiredPermissions(ctx context.Context) error {
	return ppm.storage.CleanupExpired(ctx)
}
