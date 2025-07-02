package storage

import (
	"os"
	"path/filepath"
	"runtime"
)

// PathManager handles cross-platform path resolution for CodeForge storage
type PathManager struct {
	homeDir     string
	codeforgeDir string
}

// NewPathManager creates a new path manager with platform-aware defaults
func NewPathManager() *PathManager {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory if home dir is not available
		homeDir = "."
	}

	codeforgeDir := filepath.Join(homeDir, ".codeforge")

	return &PathManager{
		homeDir:     homeDir,
		codeforgeDir: codeforgeDir,
	}
}

// GetCodeForgeDir returns the main CodeForge configuration directory
// Creates the directory if it doesn't exist
func (pm *PathManager) GetCodeForgeDir() (string, error) {
	if err := os.MkdirAll(pm.codeforgeDir, 0755); err != nil {
		return "", err
	}
	return pm.codeforgeDir, nil
}

// GetChatDatabasePath returns the path for the chat database
func (pm *PathManager) GetChatDatabasePath() (string, error) {
	dir, err := pm.GetCodeForgeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "chat.db"), nil
}

// GetPermissionDatabasePath returns the path for the permissions database
func (pm *PathManager) GetPermissionDatabasePath() (string, error) {
	dir, err := pm.GetCodeForgeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "permissions.db"), nil
}

// GetVectorDatabasePath returns the path for the vector database
func (pm *PathManager) GetVectorDatabasePath() (string, error) {
	dir, err := pm.GetCodeForgeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "vector.db"), nil
}

// GetConfigPath returns the path for the main configuration file
func (pm *PathManager) GetConfigPath() (string, error) {
	dir, err := pm.GetCodeForgeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

// GetLogsDir returns the directory for log files
func (pm *PathManager) GetLogsDir() (string, error) {
	dir, err := pm.GetCodeForgeDir()
	if err != nil {
		return "", err
	}
	logsDir := filepath.Join(dir, "logs")
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return "", err
	}
	return logsDir, nil
}

// GetCacheDir returns the directory for cache files
func (pm *PathManager) GetCacheDir() (string, error) {
	dir, err := pm.GetCodeForgeDir()
	if err != nil {
		return "", err
	}
	cacheDir := filepath.Join(dir, "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return "", err
	}
	return cacheDir, nil
}

// GetMCPConfigDir returns the directory for MCP server configurations
func (pm *PathManager) GetMCPConfigDir() (string, error) {
	dir, err := pm.GetCodeForgeDir()
	if err != nil {
		return "", err
	}
	mcpDir := filepath.Join(dir, "mcp")
	if err := os.MkdirAll(mcpDir, 0755); err != nil {
		return "", err
	}
	return mcpDir, nil
}

// GetBackupDir returns the directory for database backups
func (pm *PathManager) GetBackupDir() (string, error) {
	dir, err := pm.GetCodeForgeDir()
	if err != nil {
		return "", err
	}
	backupDir := filepath.Join(dir, "backups")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", err
	}
	return backupDir, nil
}

// GetTempDir returns a platform-appropriate temporary directory for CodeForge
func (pm *PathManager) GetTempDir() (string, error) {
	var tempBase string
	
	switch runtime.GOOS {
	case "windows":
		tempBase = os.Getenv("TEMP")
		if tempBase == "" {
			tempBase = os.Getenv("TMP")
		}
		if tempBase == "" {
			tempBase = "C:\\temp"
		}
	case "darwin", "linux":
		tempBase = "/tmp"
	default:
		tempBase = os.TempDir()
	}
	
	tempDir := filepath.Join(tempBase, "codeforge")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", err
	}
	return tempDir, nil
}

// GetHomeDir returns the user's home directory
func (pm *PathManager) GetHomeDir() string {
	return pm.homeDir
}

// GetPlatformInfo returns platform-specific information
func (pm *PathManager) GetPlatformInfo() map[string]string {
	return map[string]string{
		"os":           runtime.GOOS,
		"arch":         runtime.GOARCH,
		"home_dir":     pm.homeDir,
		"codeforge_dir": pm.codeforgeDir,
	}
}

// ValidatePaths ensures all necessary directories exist
func (pm *PathManager) ValidatePaths() error {
	// Ensure main directory exists
	if _, err := pm.GetCodeForgeDir(); err != nil {
		return err
	}

	// Ensure subdirectories exist
	if _, err := pm.GetLogsDir(); err != nil {
		return err
	}

	if _, err := pm.GetCacheDir(); err != nil {
		return err
	}

	if _, err := pm.GetMCPConfigDir(); err != nil {
		return err
	}

	if _, err := pm.GetBackupDir(); err != nil {
		return err
	}

	return nil
}

// DefaultPathManager is a global instance for convenience
var DefaultPathManager = NewPathManager()