package permissions

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// FileOperationManager handles permission-aware file operations
type FileOperationManager struct {
	service       *PermissionService
	pathValidator *PathValidator
	workspaceRoot string
}

// NewFileOperationManager creates a new file operation manager
func NewFileOperationManager(service *PermissionService, workspaceRoot string) *FileOperationManager {
	pathValidator := NewPathValidator(workspaceRoot)

	// Configure safe defaults
	pathValidator.AddAllowedPath(workspaceRoot + "/*")
	pathValidator.AddDeniedPath("/etc/*")
	pathValidator.AddDeniedPath("/sys/*")
	pathValidator.AddDeniedPath("/proc/*")
	pathValidator.AddDeniedPath("/dev/*")
	pathValidator.AddDeniedPath("/boot/*")

	return &FileOperationManager{
		service:       service,
		pathValidator: pathValidator,
		workspaceRoot: workspaceRoot,
	}
}

// FileOperationRequest represents a file operation request
type FileOperationRequest struct {
	SessionID string                 `json:"session_id"`
	Operation string                 `json:"operation"` // read, write, create, delete, list
	Path      string                 `json:"path"`
	Content   []byte                 `json:"content,omitempty"`
	Mode      os.FileMode            `json:"mode,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

// FileOperationResult represents the result of a file operation
type FileOperationResult struct {
	Success    bool                   `json:"success"`
	Content    []byte                 `json:"content,omitempty"`
	FileInfo   os.FileInfo            `json:"file_info,omitempty"`
	Error      string                 `json:"error,omitempty"`
	Permission *Permission            `json:"permission,omitempty"`
	PathResult *PathValidationResult  `json:"path_result,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ReadFile reads a file with permission checking
func (fom *FileOperationManager) ReadFile(ctx context.Context, req *FileOperationRequest) (*FileOperationResult, error) {
	// Check permissions
	permResult, err := fom.checkFilePermission(ctx, req, PermissionFileRead)
	if err != nil {
		return &FileOperationResult{
			Success: false,
			Error:   fmt.Sprintf("permission check failed: %v", err),
		}, nil
	}

	if !permResult.Allowed {
		return &FileOperationResult{
			Success: false,
			Error:   fmt.Sprintf("permission denied: %s", permResult.Reason),
		}, nil
	}

	// Validate path
	pathResult, err := fom.pathValidator.ValidatePath(req.Path, PermissionFileRead)
	if err != nil {
		return &FileOperationResult{
			Success:    false,
			Error:      fmt.Sprintf("path validation failed: %v", err),
			PathResult: pathResult,
		}, nil
	}

	if !pathResult.Allowed {
		return &FileOperationResult{
			Success:    false,
			Error:      fmt.Sprintf("path access denied: %s", pathResult.Reason),
			PathResult: pathResult,
		}, nil
	}

	// Perform the file read
	content, err := os.ReadFile(pathResult.NormalizedPath)
	if err != nil {
		return &FileOperationResult{
			Success:    false,
			Error:      fmt.Sprintf("file read failed: %v", err),
			PathResult: pathResult,
		}, nil
	}

	// Get file info
	fileInfo, _ := os.Stat(pathResult.NormalizedPath)

	return &FileOperationResult{
		Success:    true,
		Content:    content,
		FileInfo:   fileInfo,
		PathResult: pathResult,
		Metadata: map[string]interface{}{
			"bytes_read": len(content),
			"file_size":  fileInfo.Size(),
		},
	}, nil
}

// WriteFile writes a file with permission checking
func (fom *FileOperationManager) WriteFile(ctx context.Context, req *FileOperationRequest) (*FileOperationResult, error) {
	// Check permissions
	permResult, err := fom.checkFilePermission(ctx, req, PermissionFileWrite)
	if err != nil {
		return &FileOperationResult{
			Success: false,
			Error:   fmt.Sprintf("permission check failed: %v", err),
		}, nil
	}

	if !permResult.Allowed {
		return &FileOperationResult{
			Success: false,
			Error:   fmt.Sprintf("permission denied: %s", permResult.Reason),
		}, nil
	}

	// Validate path
	pathResult, err := fom.pathValidator.ValidatePath(req.Path, PermissionFileWrite)
	if err != nil {
		return &FileOperationResult{
			Success:    false,
			Error:      fmt.Sprintf("path validation failed: %v", err),
			PathResult: pathResult,
		}, nil
	}

	if !pathResult.Allowed {
		return &FileOperationResult{
			Success:    false,
			Error:      fmt.Sprintf("path access denied: %s", pathResult.Reason),
			PathResult: pathResult,
		}, nil
	}

	// Create directory if needed
	dir := filepath.Dir(pathResult.NormalizedPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &FileOperationResult{
			Success:    false,
			Error:      fmt.Sprintf("failed to create directory: %v", err),
			PathResult: pathResult,
		}, nil
	}

	// Determine file mode
	mode := req.Mode
	if mode == 0 {
		mode = 0644 // Default file mode
	}

	// Perform the file write
	err = os.WriteFile(pathResult.NormalizedPath, req.Content, mode)
	if err != nil {
		return &FileOperationResult{
			Success:    false,
			Error:      fmt.Sprintf("file write failed: %v", err),
			PathResult: pathResult,
		}, nil
	}

	// Get file info
	fileInfo, _ := os.Stat(pathResult.NormalizedPath)

	return &FileOperationResult{
		Success:    true,
		FileInfo:   fileInfo,
		PathResult: pathResult,
		Metadata: map[string]interface{}{
			"bytes_written": len(req.Content),
			"file_mode":     mode,
		},
	}, nil
}

// CreateFile creates a new file with permission checking
func (fom *FileOperationManager) CreateFile(ctx context.Context, req *FileOperationRequest) (*FileOperationResult, error) {
	// Check permissions
	permResult, err := fom.checkFilePermission(ctx, req, PermissionFileCreate)
	if err != nil {
		return &FileOperationResult{
			Success: false,
			Error:   fmt.Sprintf("permission check failed: %v", err),
		}, nil
	}

	if !permResult.Allowed {
		return &FileOperationResult{
			Success: false,
			Error:   fmt.Sprintf("permission denied: %s", permResult.Reason),
		}, nil
	}

	// Validate path
	pathResult, err := fom.pathValidator.ValidatePath(req.Path, PermissionFileCreate)
	if err != nil {
		return &FileOperationResult{
			Success:    false,
			Error:      fmt.Sprintf("path validation failed: %v", err),
			PathResult: pathResult,
		}, nil
	}

	if !pathResult.Allowed {
		return &FileOperationResult{
			Success:    false,
			Error:      fmt.Sprintf("path access denied: %s", pathResult.Reason),
			PathResult: pathResult,
		}, nil
	}

	// Check if file already exists
	if _, err := os.Stat(pathResult.NormalizedPath); err == nil {
		return &FileOperationResult{
			Success:    false,
			Error:      "file already exists",
			PathResult: pathResult,
		}, nil
	}

	// Create directory if needed
	dir := filepath.Dir(pathResult.NormalizedPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return &FileOperationResult{
			Success:    false,
			Error:      fmt.Sprintf("failed to create directory: %v", err),
			PathResult: pathResult,
		}, nil
	}

	// Determine file mode
	mode := req.Mode
	if mode == 0 {
		mode = 0644 // Default file mode
	}

	// Create the file
	file, err := os.OpenFile(pathResult.NormalizedPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, mode)
	if err != nil {
		return &FileOperationResult{
			Success:    false,
			Error:      fmt.Sprintf("file creation failed: %v", err),
			PathResult: pathResult,
		}, nil
	}

	// Write content if provided
	if req.Content != nil {
		_, err = file.Write(req.Content)
		if err != nil {
			file.Close()
			os.Remove(pathResult.NormalizedPath) // Clean up on error
			return &FileOperationResult{
				Success:    false,
				Error:      fmt.Sprintf("failed to write content: %v", err),
				PathResult: pathResult,
			}, nil
		}
	}

	file.Close()

	// Get file info
	fileInfo, _ := os.Stat(pathResult.NormalizedPath)

	return &FileOperationResult{
		Success:    true,
		FileInfo:   fileInfo,
		PathResult: pathResult,
		Metadata: map[string]interface{}{
			"created":       true,
			"bytes_written": len(req.Content),
			"file_mode":     mode,
		},
	}, nil
}

// DeleteFile deletes a file with permission checking
func (fom *FileOperationManager) DeleteFile(ctx context.Context, req *FileOperationRequest) (*FileOperationResult, error) {
	// Check permissions
	permResult, err := fom.checkFilePermission(ctx, req, PermissionFileDelete)
	if err != nil {
		return &FileOperationResult{
			Success: false,
			Error:   fmt.Sprintf("permission check failed: %v", err),
		}, nil
	}

	if !permResult.Allowed {
		return &FileOperationResult{
			Success: false,
			Error:   fmt.Sprintf("permission denied: %s", permResult.Reason),
		}, nil
	}

	// Validate path
	pathResult, err := fom.pathValidator.ValidatePath(req.Path, PermissionFileDelete)
	if err != nil {
		return &FileOperationResult{
			Success:    false,
			Error:      fmt.Sprintf("path validation failed: %v", err),
			PathResult: pathResult,
		}, nil
	}

	if !pathResult.Allowed {
		return &FileOperationResult{
			Success:    false,
			Error:      fmt.Sprintf("path access denied: %s", pathResult.Reason),
			PathResult: pathResult,
		}, nil
	}

	// Get file info before deletion
	fileInfo, err := os.Stat(pathResult.NormalizedPath)
	if err != nil {
		return &FileOperationResult{
			Success:    false,
			Error:      fmt.Sprintf("file not found: %v", err),
			PathResult: pathResult,
		}, nil
	}

	// Perform the deletion
	err = os.Remove(pathResult.NormalizedPath)
	if err != nil {
		return &FileOperationResult{
			Success:    false,
			Error:      fmt.Sprintf("file deletion failed: %v", err),
			PathResult: pathResult,
		}, nil
	}

	return &FileOperationResult{
		Success:    true,
		FileInfo:   fileInfo,
		PathResult: pathResult,
		Metadata: map[string]interface{}{
			"deleted":   true,
			"file_size": fileInfo.Size(),
		},
	}, nil
}

// ListDirectory lists directory contents with permission checking
func (fom *FileOperationManager) ListDirectory(ctx context.Context, req *FileOperationRequest) (*FileOperationResult, error) {
	// Check permissions
	permResult, err := fom.checkFilePermission(ctx, req, PermissionDirRead)
	if err != nil {
		return &FileOperationResult{
			Success: false,
			Error:   fmt.Sprintf("permission check failed: %v", err),
		}, nil
	}

	if !permResult.Allowed {
		return &FileOperationResult{
			Success: false,
			Error:   fmt.Sprintf("permission denied: %s", permResult.Reason),
		}, nil
	}

	// Validate path
	pathResult, err := fom.pathValidator.ValidatePath(req.Path, PermissionDirRead)
	if err != nil {
		return &FileOperationResult{
			Success:    false,
			Error:      fmt.Sprintf("path validation failed: %v", err),
			PathResult: pathResult,
		}, nil
	}

	if !pathResult.Allowed {
		return &FileOperationResult{
			Success:    false,
			Error:      fmt.Sprintf("path access denied: %s", pathResult.Reason),
			PathResult: pathResult,
		}, nil
	}

	// Read directory
	entries, err := os.ReadDir(pathResult.NormalizedPath)
	if err != nil {
		return &FileOperationResult{
			Success:    false,
			Error:      fmt.Sprintf("directory read failed: %v", err),
			PathResult: pathResult,
		}, nil
	}

	// Convert entries to metadata
	var files []map[string]interface{}
	for _, entry := range entries {
		info, _ := entry.Info()
		files = append(files, map[string]interface{}{
			"name":     entry.Name(),
			"is_dir":   entry.IsDir(),
			"size":     info.Size(),
			"mode":     info.Mode().String(),
			"mod_time": info.ModTime(),
		})
	}

	return &FileOperationResult{
		Success:    true,
		PathResult: pathResult,
		Metadata: map[string]interface{}{
			"files":      files,
			"file_count": len(files),
		},
	}, nil
}

// CopyFile copies a file with permission checking
func (fom *FileOperationManager) CopyFile(ctx context.Context, srcPath, destPath, sessionID string) (*FileOperationResult, error) {
	// Check read permission for source
	srcReq := &FileOperationRequest{
		SessionID: sessionID,
		Operation: "read",
		Path:      srcPath,
	}

	srcResult, err := fom.ReadFile(ctx, srcReq)
	if err != nil || !srcResult.Success {
		return srcResult, err
	}

	// Check write permission for destination
	destReq := &FileOperationRequest{
		SessionID: sessionID,
		Operation: "write",
		Path:      destPath,
		Content:   srcResult.Content,
	}

	return fom.WriteFile(ctx, destReq)
}

// checkFilePermission checks if a file operation is permitted
func (fom *FileOperationManager) checkFilePermission(ctx context.Context, req *FileOperationRequest, permType PermissionType) (*PermissionCheckResult, error) {
	check := &PermissionCheck{
		SessionID: req.SessionID,
		Type:      permType,
		Resource:  req.Path,
		Context:   req.Context,
	}

	return fom.service.CheckPermission(ctx, check)
}

// GetPathValidator returns the path validator
func (fom *FileOperationManager) GetPathValidator() *PathValidator {
	return fom.pathValidator
}

// SetWorkspaceRoot updates the workspace root
func (fom *FileOperationManager) SetWorkspaceRoot(root string) {
	fom.workspaceRoot = root
	fom.pathValidator.SetSandboxRoot(root)
}

// SafeFileReader provides a permission-aware file reader
type SafeFileReader struct {
	manager   *FileOperationManager
	sessionID string
}

// NewSafeFileReader creates a new safe file reader
func NewSafeFileReader(manager *FileOperationManager, sessionID string) *SafeFileReader {
	return &SafeFileReader{
		manager:   manager,
		sessionID: sessionID,
	}
}

// ReadFile reads a file safely with permission checking
func (sfr *SafeFileReader) ReadFile(ctx context.Context, path string) ([]byte, error) {
	req := &FileOperationRequest{
		SessionID: sfr.sessionID,
		Operation: "read",
		Path:      path,
	}

	result, err := sfr.manager.ReadFile(ctx, req)
	if err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, fmt.Errorf("%s", result.Error)
	}

	return result.Content, nil
}

// SafeFileWriter provides a permission-aware file writer
type SafeFileWriter struct {
	manager   *FileOperationManager
	sessionID string
}

// NewSafeFileWriter creates a new safe file writer
func NewSafeFileWriter(manager *FileOperationManager, sessionID string) *SafeFileWriter {
	return &SafeFileWriter{
		manager:   manager,
		sessionID: sessionID,
	}
}

// WriteFile writes a file safely with permission checking
func (sfw *SafeFileWriter) WriteFile(ctx context.Context, path string, content []byte, mode os.FileMode) error {
	req := &FileOperationRequest{
		SessionID: sfw.sessionID,
		Operation: "write",
		Path:      path,
		Content:   content,
		Mode:      mode,
	}

	result, err := sfw.manager.WriteFile(ctx, req)
	if err != nil {
		return err
	}

	if !result.Success {
		return fmt.Errorf("%s", result.Error)
	}

	return nil
}

// CreateFile creates a file safely with permission checking
func (sfw *SafeFileWriter) CreateFile(ctx context.Context, path string, content []byte, mode os.FileMode) error {
	req := &FileOperationRequest{
		SessionID: sfw.sessionID,
		Operation: "create",
		Path:      path,
		Content:   content,
		Mode:      mode,
	}

	result, err := sfw.manager.CreateFile(ctx, req)
	if err != nil {
		return err
	}

	if !result.Success {
		return fmt.Errorf("%s", result.Error)
	}

	return nil
}
