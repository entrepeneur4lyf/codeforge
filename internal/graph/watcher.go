package graph

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileWatcher watches filesystem changes and updates the graph
type FileWatcher struct {
	graph   *CodeGraph
	scanner *SimpleScanner
	watcher *fsnotify.Watcher

	// Debouncing
	debounceDelay time.Duration
	pendingFiles  map[string]time.Time
	mutex         sync.Mutex

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	done   chan struct{}

	// Configuration
	rootPath     string
	ignoreRules  []string
	watchTests   bool
	watchDocs    bool
	watchConfigs bool
}

// NewFileWatcher creates a new filesystem watcher
func NewFileWatcher(graph *CodeGraph, rootPath string) (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create filesystem watcher: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	fw := &FileWatcher{
		graph:         graph,
		scanner:       NewSimpleScanner(graph),
		watcher:       watcher,
		debounceDelay: 500 * time.Millisecond, // 500ms debounce
		pendingFiles:  make(map[string]time.Time),
		ctx:           ctx,
		cancel:        cancel,
		done:          make(chan struct{}),
		rootPath:      rootPath,
		watchTests:    true,
		watchDocs:     true,
		watchConfigs:  true,
		ignoreRules: []string{
			".git", ".svn", ".hg",
			"node_modules", "vendor", "target",
			".vscode", ".idea", ".vs",
			"build", "dist", "out",
			"__pycache__", ".pytest_cache",
			".codeforge", // Exclude entire .codeforge directory to prevent log file spam
			"*.log", "*.tmp", "*.temp",
			"*.exe", "*.dll", "*.so", "*.dylib",
		},
	}

	return fw, nil
}

// Start begins watching the filesystem
func (fw *FileWatcher) Start() error {
	// Add root directory to watcher
	if err := fw.addDirectoryRecursive(fw.rootPath); err != nil {
		return fmt.Errorf("failed to add root directory to watcher: %w", err)
	}

	// Start the event processing goroutine
	go fw.processEvents()

	// Start the debounce processing goroutine
	go fw.processDebounced()

	log.Printf("File watcher started for: %s", fw.rootPath)
	return nil
}

// Stop stops the filesystem watcher
func (fw *FileWatcher) Stop() {
	fw.cancel()
	fw.watcher.Close()
	<-fw.done
	log.Println("File watcher stopped")
}

// SetIgnoreRules sets custom ignore rules
func (fw *FileWatcher) SetIgnoreRules(rules []string) {
	fw.ignoreRules = rules
}

// SetDebounceDelay sets the debounce delay
func (fw *FileWatcher) SetDebounceDelay(delay time.Duration) {
	fw.debounceDelay = delay
}

// processEvents processes filesystem events
func (fw *FileWatcher) processEvents() {
	defer close(fw.done)

	for {
		select {
		case <-fw.ctx.Done():
			return

		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}

			fw.handleEvent(event)

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("File watcher error: %v", err)
		}
	}
}

// handleEvent handles a single filesystem event
func (fw *FileWatcher) handleEvent(event fsnotify.Event) {
	// Get relative path
	relPath, err := filepath.Rel(fw.rootPath, event.Name)
	if err != nil {
		return
	}

	// Check if we should ignore this file
	if fw.shouldIgnore(relPath) {
		return
	}

	// Add to pending files for debouncing
	fw.mutex.Lock()
	fw.pendingFiles[relPath] = time.Now()
	fw.mutex.Unlock()

	log.Printf("📝 File change detected: %s (%s)", relPath, event.Op.String())
}

// processDebounced processes debounced file changes
func (fw *FileWatcher) processDebounced() {
	ticker := time.NewTicker(100 * time.Millisecond) // Check every 100ms
	defer ticker.Stop()

	for {
		select {
		case <-fw.ctx.Done():
			return

		case <-ticker.C:
			fw.processPendingFiles()
		}
	}
}

// processPendingFiles processes files that have been pending for the debounce delay
func (fw *FileWatcher) processPendingFiles() {
	fw.mutex.Lock()
	now := time.Now()
	readyFiles := make([]string, 0)

	for filePath, timestamp := range fw.pendingFiles {
		if now.Sub(timestamp) >= fw.debounceDelay {
			readyFiles = append(readyFiles, filePath)
			delete(fw.pendingFiles, filePath)
		}
	}
	fw.mutex.Unlock()

	// Process ready files
	if len(readyFiles) > 0 {
		fw.updateGraphForFiles(readyFiles)
	}
}

// updateGraphForFiles updates the graph for changed files
func (fw *FileWatcher) updateGraphForFiles(files []string) {
	log.Printf("Updating graph for %d files", len(files))

	for _, relPath := range files {
		fullPath := filepath.Join(fw.rootPath, relPath)

		// Check if file still exists
		if info, err := os.Lstat(fullPath); err == nil {
			// File exists - update or add
			if info.IsDir() {
				fw.updateDirectory(relPath, info)
			} else {
				fw.updateFile(relPath, info, fullPath)
			}
		} else {
			// File was deleted - remove from graph
			fw.removeFromGraph(relPath)
		}
	}

	log.Printf("Graph updated successfully")
}

// updateDirectory updates a directory in the graph
func (fw *FileWatcher) updateDirectory(relPath string, info os.FileInfo) {
	// Remove existing directory node if it exists
	if existingNode, exists := fw.graph.GetNodeByPath(relPath); exists {
		fw.removeNodeAndEdges(existingNode.ID)
	}

	// Add updated directory node
	node := &Node{
		Type:         NodeTypeDirectory,
		Name:         info.Name(),
		Path:         relPath,
		Purpose:      fw.scanner.inferDirectoryPurpose(info.Name()),
		LastModified: info.ModTime(),
		Size:         info.Size(),
	}

	if err := fw.graph.AddNode(node); err != nil {
		log.Printf("Failed to add directory node: %v", err)
		return
	}

	// Create containment edge from parent
	fw.createParentEdge(node)
}

// updateFile updates a file in the graph
func (fw *FileWatcher) updateFile(relPath string, info os.FileInfo, fullPath string) {
	// Remove existing file node if it exists
	if existingNode, exists := fw.graph.GetNodeByPath(relPath); exists {
		fw.removeNodeAndEdges(existingNode.ID)
	}

	language := fw.scanner.detectLanguage(relPath)
	nodeType := fw.scanner.inferFileNodeType(relPath, language)

	// Skip certain file types based on configuration
	if !fw.shouldIncludeFile(nodeType) {
		return
	}

	node := &Node{
		Type:         nodeType,
		Name:         info.Name(),
		Path:         relPath,
		Purpose:      fw.scanner.inferFilePurpose(relPath, language),
		Language:     language,
		LastModified: info.ModTime(),
		Size:         info.Size(),
	}

	if err := fw.graph.AddNode(node); err != nil {
		log.Printf("Failed to add file node: %v", err)
		return
	}

	// Create containment edge from parent
	fw.createParentEdge(node)

	// Parse file content for deeper analysis
	if language == "go" {
		fw.scanner.parseGoFile(fullPath, node)
	}
}

// removeFromGraph removes a file/directory from the graph
func (fw *FileWatcher) removeFromGraph(relPath string) {
	if node, exists := fw.graph.GetNodeByPath(relPath); exists {
		fw.removeNodeAndEdges(node.ID)
		log.Printf("🗑️ Removed from graph: %s", relPath)
	}
}

// removeNodeAndEdges removes a node and all its edges
func (fw *FileWatcher) removeNodeAndEdges(nodeID string) {
	// Remove all outgoing edges
	outEdges := fw.graph.GetOutgoingEdges(nodeID)
	for _, edge := range outEdges {
		delete(fw.graph.graph.Edges, edge.ID)
	}

	// Remove all incoming edges
	inEdges := fw.graph.GetIncomingEdges(nodeID)
	for _, edge := range inEdges {
		delete(fw.graph.graph.Edges, edge.ID)
	}

	// Remove the node itself
	if node, exists := fw.graph.graph.Nodes[nodeID]; exists {
		delete(fw.graph.graph.Nodes, nodeID)
		delete(fw.graph.graph.NodesByPath, node.Path)
		delete(fw.graph.graph.OutEdges, nodeID)
		delete(fw.graph.graph.InEdges, nodeID)

		// Update statistics
		fw.graph.mutex.Lock()
		fw.graph.stats.NodesByType[node.Type]--
		fw.graph.stats.NodeCount--
		if node.Language != "" {
			fw.graph.stats.Languages[node.Language]--
			if fw.graph.stats.Languages[node.Language] <= 0 {
				delete(fw.graph.stats.Languages, node.Language)
			}
		}
		fw.graph.mutex.Unlock()
	}
}

// createParentEdge creates a containment edge from the parent directory
func (fw *FileWatcher) createParentEdge(node *Node) {
	parentPath := filepath.Dir(node.Path)
	if parentPath == "." {
		parentPath = ""
	}

	if parentNode, exists := fw.graph.GetNodeByPath(parentPath); exists {
		edge := &Edge{
			Type:   EdgeTypeContains,
			Source: parentNode.ID,
			Target: node.ID,
		}
		fw.graph.AddEdge(edge)
	}
}

// addDirectoryRecursive adds a directory and all subdirectories to the watcher
func (fw *FileWatcher) addDirectoryRecursive(path string) error {
	return filepath.Walk(path, func(walkPath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip directories with errors
		}

		if !info.IsDir() {
			return nil // Only watch directories
		}

		// Get relative path for ignore checking
		relPath, _ := filepath.Rel(fw.rootPath, walkPath)
		if fw.shouldIgnore(relPath) {
			return filepath.SkipDir
		}

		// Add directory to watcher
		if err := fw.watcher.Add(walkPath); err != nil {
			log.Printf("Failed to watch directory %s: %v", walkPath, err)
		}

		return nil
	})
}

// shouldIgnore checks if a path should be ignored
func (fw *FileWatcher) shouldIgnore(path string) bool {
	pathLower := strings.ToLower(path)

	for _, pattern := range fw.ignoreRules {
		if strings.Contains(pathLower, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// shouldIncludeFile determines if a file should be included based on configuration
func (fw *FileWatcher) shouldIncludeFile(nodeType NodeType) bool {
	switch nodeType {
	case NodeTypeTest:
		return fw.watchTests
	case NodeTypeDocumentation:
		return fw.watchDocs
	case NodeTypeConfig:
		return fw.watchConfigs
	default:
		return true
	}
}
