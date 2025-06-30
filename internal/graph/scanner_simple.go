package graph

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SimpleScanner provides basic code analysis without external dependencies
type SimpleScanner struct {
	graph   *CodeGraph
	fileSet *token.FileSet

	// Configuration
	maxFileSize    int64
	includeTests   bool
	includeDocs    bool
	includeConfigs bool
}

// NewSimpleScanner creates a new simple scanner
func NewSimpleScanner(graph *CodeGraph) *SimpleScanner {
	return &SimpleScanner{
		graph:          graph,
		fileSet:        token.NewFileSet(),
		maxFileSize:    10 * 1024 * 1024, // 10MB max file size
		includeTests:   true,
		includeDocs:    true,
		includeConfigs: true,
	}
}

// ScanRepository scans a repository and builds the graph
func (s *SimpleScanner) ScanRepository(rootPath string) error {
	startTime := time.Now()

	// Clear existing graph
	s.graph.Clear()

	// Create root directory node
	rootNode := &Node{
		Type:         NodeTypeDirectory,
		Name:         filepath.Base(rootPath),
		Path:         "",
		Purpose:      "Repository root directory",
		LastModified: time.Now(),
	}

	if err := s.graph.AddNode(rootNode); err != nil {
		return fmt.Errorf("failed to add root node: %w", err)
	}

	// Walk the directory tree
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files with errors
		}

		// Get relative path
		relPath, _ := filepath.Rel(rootPath, path)
		if relPath == "." {
			return nil // Skip root
		}

		// Skip hidden files and common ignore patterns
		if s.shouldIgnore(relPath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip large files
		if !info.IsDir() && info.Size() > s.maxFileSize {
			return nil
		}

		if info.IsDir() {
			return s.scanDirectory(relPath, info)
		} else {
			return s.scanFile(relPath, info, rootPath)
		}
	})

	if err != nil {
		return fmt.Errorf("failed to scan repository: %w", err)
	}

	// Update statistics
	s.graph.mutex.Lock()
	s.graph.stats.LastUpdated = time.Now()
	s.graph.stats.ScanDuration = time.Since(startTime)
	s.graph.mutex.Unlock()

	return nil
}

// shouldIgnore checks if a path should be ignored
func (s *SimpleScanner) shouldIgnore(path string) bool {
	// Common ignore patterns
	ignorePatterns := []string{
		".git", ".svn", ".hg",
		"node_modules", "vendor", "target",
		".vscode", ".idea", ".vs",
		"build", "dist", "out",
		"__pycache__", ".pytest_cache",
		"*.pyc", "*.pyo", "*.pyd",
		"*.class", "*.jar", "*.war",
		"*.exe", "*.dll", "*.so", "*.dylib",
		"*.log", "*.tmp", "*.temp",
	}

	pathLower := strings.ToLower(path)

	for _, pattern := range ignorePatterns {
		if strings.Contains(pathLower, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

// scanDirectory processes a directory
func (s *SimpleScanner) scanDirectory(relPath string, info os.FileInfo) error {
	node := &Node{
		Type:         NodeTypeDirectory,
		Name:         info.Name(),
		Path:         relPath,
		Purpose:      s.inferDirectoryPurpose(info.Name()),
		LastModified: info.ModTime(),
		Size:         info.Size(),
	}

	if err := s.graph.AddNode(node); err != nil {
		return err
	}

	// Create containment edge from parent directory
	parentPath := filepath.Dir(relPath)
	if parentPath == "." {
		parentPath = ""
	}

	if parentNode, exists := s.graph.GetNodeByPath(parentPath); exists {
		edge := &Edge{
			Type:   EdgeTypeContains,
			Source: parentNode.ID,
			Target: node.ID,
		}
		s.graph.AddEdge(edge)
	}

	return nil
}

// scanFile processes a file
func (s *SimpleScanner) scanFile(relPath string, info os.FileInfo, rootPath string) error {
	language := s.detectLanguage(relPath)
	nodeType := s.inferFileNodeType(relPath, language)

	// Skip certain file types based on configuration
	if !s.shouldIncludeFile(nodeType) {
		return nil
	}

	node := &Node{
		Type:         nodeType,
		Name:         info.Name(),
		Path:         relPath,
		Purpose:      s.inferFilePurpose(relPath, language),
		Language:     language,
		LastModified: info.ModTime(),
		Size:         info.Size(),
	}

	if err := s.graph.AddNode(node); err != nil {
		return err
	}

	// Create containment edge from parent directory
	parentPath := filepath.Dir(relPath)
	if parentPath == "." {
		parentPath = ""
	}

	if parentNode, exists := s.graph.GetNodeByPath(parentPath); exists {
		edge := &Edge{
			Type:   EdgeTypeContains,
			Source: parentNode.ID,
			Target: node.ID,
		}
		s.graph.AddEdge(edge)
	}

	// Parse file content for deeper analysis
	if language == "go" {
		fullPath := filepath.Join(rootPath, relPath)
		s.parseGoFile(fullPath, node)
	}

	return nil
}

// parseGoFile parses Go source files
func (s *SimpleScanner) parseGoFile(path string, fileNode *Node) {
	src, err := os.ReadFile(path)
	if err != nil {
		return
	}

	// Parse Go AST
	file, err := parser.ParseFile(s.fileSet, path, src, parser.ParseComments)
	if err != nil {
		return
	}

	// Extract package information
	if file.Name != nil {
		packageNode := &Node{
			Type:     NodeTypePackage,
			Name:     file.Name.Name,
			Path:     filepath.Dir(fileNode.Path),
			Language: "go",
			Purpose:  "Go package",
		}

		if err := s.graph.AddNode(packageNode); err == nil {
			edge := &Edge{
				Type:   EdgeTypeDefines,
				Source: fileNode.ID,
				Target: packageNode.ID,
			}
			s.graph.AddEdge(edge)
		}
	}

	// Extract imports
	for _, imp := range file.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)

		importNode := &Node{
			Type:     NodeTypeImport,
			Name:     importPath,
			Path:     importPath,
			Language: "go",
			Purpose:  "Go import",
		}

		if err := s.graph.AddNode(importNode); err == nil {
			edge := &Edge{
				Type:   EdgeTypeImports,
				Source: fileNode.ID,
				Target: importNode.ID,
			}
			s.graph.AddEdge(edge)
		}
	}

	// Extract functions, types, etc.
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			s.extractGoFunction(node, fileNode)
		case *ast.TypeSpec:
			s.extractGoType(node, fileNode)
		}
		return true
	})
}

// extractGoFunction extracts Go function information with detailed API signature
func (s *SimpleScanner) extractGoFunction(funcDecl *ast.FuncDecl, fileNode *Node) {
	if funcDecl.Name == nil {
		return
	}

	pos := s.fileSet.Position(funcDecl.Pos())
	end := s.fileSet.Position(funcDecl.End())

	visibility := "private"
	isExported := funcDecl.Name.IsExported()
	if isExported {
		visibility = "public"
	}

	// Extract detailed function information
	funcInfo := s.extractFunctionInfo(funcDecl, fileNode.Path)
	apiSig := s.extractAPISignature(funcDecl, fileNode.Path)

	funcNode := &Node{
		Type:         NodeTypeFunction,
		Name:         funcDecl.Name.Name,
		Path:         fileNode.Path,
		Language:     "go",
		Purpose:      "Go function",
		Visibility:   visibility,
		StartLine:    pos.Line,
		EndLine:      end.Line,
		Signature:    s.buildFunctionSignature(funcDecl),
		DocComment:   s.extractDocComment(funcDecl.Doc),
		FunctionInfo: funcInfo,
		APISignature: apiSig,
	}

	if err := s.graph.AddNode(funcNode); err == nil {
		edge := &Edge{
			Type:   EdgeTypeDefines,
			Source: fileNode.ID,
			Target: funcNode.ID,
		}
		s.graph.AddEdge(edge)
	}
}

// extractGoType extracts Go type information with detailed structure
func (s *SimpleScanner) extractGoType(typeSpec *ast.TypeSpec, fileNode *Node) {
	if typeSpec.Name == nil {
		return
	}

	nodeType := NodeTypeType
	var dataStructure *DataStructure
	var typeInfo *TypeInfo

	switch typeSpec.Type.(type) {
	case *ast.StructType:
		nodeType = NodeTypeStruct
		dataStructure = s.extractStructInfo(typeSpec, fileNode.Path)
	case *ast.InterfaceType:
		nodeType = NodeTypeInterface
		dataStructure = s.extractInterfaceInfo(typeSpec, fileNode.Path)
	default:
		typeInfo = s.extractTypeInfo(typeSpec, fileNode.Path)
	}

	visibility := "private"
	isExported := typeSpec.Name.IsExported()
	if isExported {
		visibility = "public"
	}

	typeNode := &Node{
		Type:          nodeType,
		Name:          typeSpec.Name.Name,
		Path:          fileNode.Path,
		Language:      "go",
		Purpose:       fmt.Sprintf("Go %s", nodeType),
		Visibility:    visibility,
		DocComment:    s.extractDocComment(nil), // Will be enhanced with actual doc
		DataStructure: dataStructure,
		TypeInfo:      typeInfo,
	}

	if err := s.graph.AddNode(typeNode); err == nil {
		edge := &Edge{
			Type:   EdgeTypeDefines,
			Source: fileNode.ID,
			Target: typeNode.ID,
		}
		s.graph.AddEdge(edge)
	}
}

// extractStructInfo extracts detailed struct information
func (s *SimpleScanner) extractStructInfo(typeSpec *ast.TypeSpec, filePath string) *DataStructure {
	structType, ok := typeSpec.Type.(*ast.StructType)
	if !ok {
		return nil
	}

	ds := &DataStructure{
		Name:       typeSpec.Name.Name,
		Type:       "struct",
		Package:    s.extractPackageName(filePath),
		IsExported: typeSpec.Name.IsExported(),
		Fields:     make([]Field, 0),
		Embedded:   make([]string, 0),
	}

	// Extract fields
	if structType.Fields != nil {
		for _, field := range structType.Fields.List {
			fieldType := s.typeToString(field.Type)

			// Handle embedded fields (no names)
			if len(field.Names) == 0 {
				ds.Embedded = append(ds.Embedded, fieldType)
				continue
			}

			// Handle named fields
			for _, name := range field.Names {
				f := Field{
					Name:       name.Name,
					Type:       fieldType,
					IsExported: name.IsExported(),
					IsPointer:  s.isPointerType(field.Type),
					IsSlice:    s.isSliceType(field.Type),
					IsMap:      s.isMapType(field.Type),
					DocComment: s.extractDocComment(field.Doc),
				}

				// Extract struct tags
				if field.Tag != nil {
					f.Tag = field.Tag.Value
				}

				ds.Fields = append(ds.Fields, f)
			}
		}
	}

	return ds
}

// extractInterfaceInfo extracts detailed interface information
func (s *SimpleScanner) extractInterfaceInfo(typeSpec *ast.TypeSpec, filePath string) *DataStructure {
	interfaceType, ok := typeSpec.Type.(*ast.InterfaceType)
	if !ok {
		return nil
	}

	ds := &DataStructure{
		Name:       typeSpec.Name.Name,
		Type:       "interface",
		Package:    s.extractPackageName(filePath),
		IsExported: typeSpec.Name.IsExported(),
		Methods:    make([]Method, 0),
		Embedded:   make([]string, 0),
	}

	// Extract methods and embedded interfaces
	if interfaceType.Methods != nil {
		for _, method := range interfaceType.Methods.List {
			// Handle embedded interfaces (no names)
			if len(method.Names) == 0 {
				embeddedType := s.typeToString(method.Type)
				ds.Embedded = append(ds.Embedded, embeddedType)
				continue
			}

			// Handle method signatures
			for _, name := range method.Names {
				if funcType, ok := method.Type.(*ast.FuncType); ok {
					m := Method{
						Name:        name.Name,
						IsExported:  name.IsExported(),
						Parameters:  make([]Parameter, 0),
						ReturnTypes: make([]ReturnType, 0),
						DocComment:  s.extractDocComment(method.Doc),
					}

					// Extract parameters
					if funcType.Params != nil {
						for _, param := range funcType.Params.List {
							paramType := s.typeToString(param.Type)

							if len(param.Names) > 0 {
								for _, paramName := range param.Names {
									m.Parameters = append(m.Parameters, Parameter{
										Name:       paramName.Name,
										Type:       paramType,
										IsPointer:  s.isPointerType(param.Type),
										IsSlice:    s.isSliceType(param.Type),
										IsVariadic: s.isVariadicType(param.Type),
									})
								}
							} else {
								m.Parameters = append(m.Parameters, Parameter{
									Type:       paramType,
									IsPointer:  s.isPointerType(param.Type),
									IsSlice:    s.isSliceType(param.Type),
									IsVariadic: s.isVariadicType(param.Type),
								})
							}
						}
					}

					// Extract return types
					if funcType.Results != nil {
						for _, result := range funcType.Results.List {
							resultType := s.typeToString(result.Type)

							if len(result.Names) > 0 {
								for _, resultName := range result.Names {
									m.ReturnTypes = append(m.ReturnTypes, ReturnType{
										Name:      resultName.Name,
										Type:      resultType,
										IsPointer: s.isPointerType(result.Type),
										IsSlice:   s.isSliceType(result.Type),
										IsError:   resultType == "error",
									})
								}
							} else {
								m.ReturnTypes = append(m.ReturnTypes, ReturnType{
									Type:      resultType,
									IsPointer: s.isPointerType(result.Type),
									IsSlice:   s.isSliceType(result.Type),
									IsError:   resultType == "error",
								})
							}
						}
					}

					ds.Methods = append(ds.Methods, m)
				}
			}
		}
	}

	return ds
}

// extractTypeInfo extracts type alias/definition information
func (s *SimpleScanner) extractTypeInfo(typeSpec *ast.TypeSpec, filePath string) *TypeInfo {
	return &TypeInfo{
		Name:           typeSpec.Name.Name,
		Package:        s.extractPackageName(filePath),
		Kind:           "alias",
		UnderlyingType: s.typeToString(typeSpec.Type),
		IsExported:     typeSpec.Name.IsExported(),
	}
}

// Helper functions

// detectLanguage detects the programming language of a file
func (s *SimpleScanner) detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))

	languageMap := map[string]string{
		".go":   "go",
		".py":   "python",
		".js":   "javascript",
		".ts":   "typescript",
		".rs":   "rust",
		".java": "java",
		".cpp":  "cpp",
		".c":    "c",
		".cs":   "csharp",
		".php":  "php",
		".rb":   "ruby",
		".sh":   "shell",
		".sql":  "sql",
		".html": "html",
		".css":  "css",
		".json": "json",
		".yaml": "yaml",
		".yml":  "yaml",
		".xml":  "xml",
		".md":   "markdown",
		".txt":  "text",
	}

	if lang, exists := languageMap[ext]; exists {
		return lang
	}

	return "unknown"
}

// inferFileNodeType infers the node type based on file characteristics
func (s *SimpleScanner) inferFileNodeType(path string, language string) NodeType {
	filename := strings.ToLower(filepath.Base(path))

	// Test files
	if strings.Contains(filename, "test") || strings.Contains(filename, "spec") {
		return NodeTypeTest
	}

	// Documentation files
	if language == "markdown" || strings.Contains(filename, "readme") ||
		strings.Contains(filename, "doc") {
		return NodeTypeDocumentation
	}

	// Configuration files
	configFiles := []string{"config", "settings", "env", ".env", "dockerfile", "makefile"}
	for _, configFile := range configFiles {
		if strings.Contains(filename, configFile) {
			return NodeTypeConfig
		}
	}

	// Default to file
	return NodeTypeFile
}

// shouldIncludeFile determines if a file should be included based on configuration
func (s *SimpleScanner) shouldIncludeFile(nodeType NodeType) bool {
	switch nodeType {
	case NodeTypeTest:
		return s.includeTests
	case NodeTypeDocumentation:
		return s.includeDocs
	case NodeTypeConfig:
		return s.includeConfigs
	default:
		return true
	}
}

// inferDirectoryPurpose infers the purpose of a directory
func (s *SimpleScanner) inferDirectoryPurpose(name string) string {
	name = strings.ToLower(name)

	purposeMap := map[string]string{
		"src":          "Source code directory",
		"lib":          "Library directory",
		"bin":          "Binary directory",
		"build":        "Build output directory",
		"dist":         "Distribution directory",
		"test":         "Test directory",
		"tests":        "Test directory",
		"doc":          "Documentation directory",
		"docs":         "Documentation directory",
		"config":       "Configuration directory",
		"assets":       "Asset directory",
		"static":       "Static files directory",
		"public":       "Public files directory",
		"vendor":       "Vendor dependencies directory",
		"node_modules": "Node.js dependencies directory",
		"target":       "Rust build directory",
		".git":         "Git repository directory",
		".vscode":      "VS Code configuration directory",
		".idea":        "IntelliJ IDEA configuration directory",
	}

	if purpose, exists := purposeMap[name]; exists {
		return purpose
	}

	return "Directory"
}

// inferFilePurpose infers the purpose of a file
func (s *SimpleScanner) inferFilePurpose(path, language string) string {
	filename := strings.ToLower(filepath.Base(path))

	// Special files
	specialFiles := map[string]string{
		"readme.md":        "Project documentation",
		"license":          "License file",
		"changelog.md":     "Change log",
		"makefile":         "Build configuration",
		"dockerfile":       "Docker configuration",
		"go.mod":           "Go module definition",
		"package.json":     "Node.js package definition",
		"cargo.toml":       "Rust package definition",
		"requirements.txt": "Python dependencies",
		"setup.py":         "Python setup script",
		".gitignore":       "Git ignore rules",
	}

	if purpose, exists := specialFiles[filename]; exists {
		return purpose
	}

	// Language-based purposes
	switch language {
	case "go":
		return "Go source file"
	case "python":
		return "Python source file"
	case "javascript":
		return "JavaScript source file"
	case "typescript":
		return "TypeScript source file"
	case "rust":
		return "Rust source file"
	case "markdown":
		return "Documentation file"
	case "json":
		return "JSON configuration file"
	case "yaml":
		return "YAML configuration file"
	default:
		return "Source file"
	}
}

// extractFunctionInfo extracts detailed function information
func (s *SimpleScanner) extractFunctionInfo(funcDecl *ast.FuncDecl, filePath string) *FunctionInfo {
	if funcDecl.Name == nil {
		return nil
	}

	info := &FunctionInfo{
		Name:       funcDecl.Name.Name,
		Package:    s.extractPackageName(filePath),
		IsExported: funcDecl.Name.IsExported(),
		IsMethod:   funcDecl.Recv != nil,
		DocComment: s.extractDocComment(funcDecl.Doc),
	}

	// Extract parameters
	if funcDecl.Type.Params != nil {
		for _, param := range funcDecl.Type.Params.List {
			paramType := s.typeToString(param.Type)

			// Handle multiple names for same type (e.g., a, b int)
			if len(param.Names) > 0 {
				for _, name := range param.Names {
					info.Parameters = append(info.Parameters, Parameter{
						Name:       name.Name,
						Type:       paramType,
						IsPointer:  s.isPointerType(param.Type),
						IsSlice:    s.isSliceType(param.Type),
						IsVariadic: s.isVariadicType(param.Type),
					})
				}
			} else {
				// Unnamed parameter
				info.Parameters = append(info.Parameters, Parameter{
					Type:       paramType,
					IsPointer:  s.isPointerType(param.Type),
					IsSlice:    s.isSliceType(param.Type),
					IsVariadic: s.isVariadicType(param.Type),
				})
			}
		}
	}

	// Extract return types
	if funcDecl.Type.Results != nil {
		for _, result := range funcDecl.Type.Results.List {
			resultType := s.typeToString(result.Type)

			returnType := ReturnType{
				Type:      resultType,
				IsPointer: s.isPointerType(result.Type),
				IsSlice:   s.isSliceType(result.Type),
				IsError:   resultType == "error",
			}

			// Handle named returns
			if len(result.Names) > 0 {
				for _, name := range result.Names {
					namedReturn := returnType
					namedReturn.Name = name.Name
					info.ReturnTypes = append(info.ReturnTypes, namedReturn)
				}
			} else {
				info.ReturnTypes = append(info.ReturnTypes, returnType)
			}
		}
	}

	// Extract receiver for methods
	if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
		recv := funcDecl.Recv.List[0]
		recvType := s.typeToString(recv.Type)

		receiver := &Receiver{
			Type:      recvType,
			IsPointer: s.isPointerType(recv.Type),
		}

		if len(recv.Names) > 0 {
			receiver.Name = recv.Names[0].Name
		}

		info.Receiver = receiver
	}

	// Check if variadic
	info.IsVariadic = s.isFunctionVariadic(funcDecl)

	// Calculate line count
	pos := s.fileSet.Position(funcDecl.Pos())
	end := s.fileSet.Position(funcDecl.End())
	info.LineCount = end.Line - pos.Line + 1

	return info
}

// extractAPISignature extracts API signature information
func (s *SimpleScanner) extractAPISignature(funcDecl *ast.FuncDecl, filePath string) *APISignature {
	if funcDecl.Name == nil {
		return nil
	}

	funcInfo := s.extractFunctionInfo(funcDecl, filePath)
	if funcInfo == nil {
		return nil
	}

	sig := &APISignature{
		Name:        funcInfo.Name,
		FullName:    fmt.Sprintf("%s.%s", funcInfo.Package, funcInfo.Name),
		Parameters:  funcInfo.Parameters,
		ReturnTypes: funcInfo.ReturnTypes,
		Receiver:    funcInfo.Receiver,
		IsExported:  funcInfo.IsExported,
		IsMethod:    funcInfo.IsMethod,
		IsVariadic:  funcInfo.IsVariadic,
		DocComment:  funcInfo.DocComment,
	}

	// Build full name with receiver for methods
	if sig.IsMethod && sig.Receiver != nil {
		sig.FullName = fmt.Sprintf("%s.%s.%s", funcInfo.Package, sig.Receiver.Type, funcInfo.Name)
	}

	return sig
}

// buildFunctionSignature builds a human-readable function signature
func (s *SimpleScanner) buildFunctionSignature(funcDecl *ast.FuncDecl) string {
	var sig strings.Builder

	sig.WriteString("func ")

	// Add receiver for methods
	if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
		sig.WriteString("(")
		recv := funcDecl.Recv.List[0]
		if len(recv.Names) > 0 {
			sig.WriteString(recv.Names[0].Name)
			sig.WriteString(" ")
		}
		sig.WriteString(s.typeToString(recv.Type))
		sig.WriteString(") ")
	}

	sig.WriteString(funcDecl.Name.Name)

	// Add parameters
	sig.WriteString("(")
	if funcDecl.Type.Params != nil {
		for i, param := range funcDecl.Type.Params.List {
			if i > 0 {
				sig.WriteString(", ")
			}

			if len(param.Names) > 0 {
				for j, name := range param.Names {
					if j > 0 {
						sig.WriteString(", ")
					}
					sig.WriteString(name.Name)
				}
				sig.WriteString(" ")
			}
			sig.WriteString(s.typeToString(param.Type))
		}
	}
	sig.WriteString(")")

	// Add return types
	if funcDecl.Type.Results != nil && len(funcDecl.Type.Results.List) > 0 {
		sig.WriteString(" ")
		if len(funcDecl.Type.Results.List) > 1 {
			sig.WriteString("(")
		}

		for i, result := range funcDecl.Type.Results.List {
			if i > 0 {
				sig.WriteString(", ")
			}

			if len(result.Names) > 0 {
				for j, name := range result.Names {
					if j > 0 {
						sig.WriteString(", ")
					}
					sig.WriteString(name.Name)
					sig.WriteString(" ")
				}
			}
			sig.WriteString(s.typeToString(result.Type))
		}

		if len(funcDecl.Type.Results.List) > 1 {
			sig.WriteString(")")
		}
	}

	return sig.String()
}

// Helper methods for type analysis

// extractDocComment extracts documentation comment
func (s *SimpleScanner) extractDocComment(commentGroup *ast.CommentGroup) string {
	if commentGroup == nil {
		return ""
	}

	var doc strings.Builder
	for _, comment := range commentGroup.List {
		text := comment.Text
		// Remove // or /* */ prefixes
		if strings.HasPrefix(text, "//") {
			text = strings.TrimSpace(text[2:])
		} else if strings.HasPrefix(text, "/*") && strings.HasSuffix(text, "*/") {
			text = strings.TrimSpace(text[2 : len(text)-2])
		}

		if doc.Len() > 0 {
			doc.WriteString(" ")
		}
		doc.WriteString(text)
	}

	return doc.String()
}

// extractPackageName extracts package name from file path
func (s *SimpleScanner) extractPackageName(filePath string) string {
	dir := filepath.Dir(filePath)
	if dir == "." || dir == "" {
		return "main"
	}
	return filepath.Base(dir)
}

// typeToString converts an AST type to string representation
func (s *SimpleScanner) typeToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + s.typeToString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + s.typeToString(t.Elt)
		}
		return "[" + s.typeToString(t.Len) + "]" + s.typeToString(t.Elt)
	case *ast.MapType:
		return "map[" + s.typeToString(t.Key) + "]" + s.typeToString(t.Value)
	case *ast.ChanType:
		switch t.Dir {
		case ast.SEND:
			return "chan<- " + s.typeToString(t.Value)
		case ast.RECV:
			return "<-chan " + s.typeToString(t.Value)
		default:
			return "chan " + s.typeToString(t.Value)
		}
	case *ast.FuncType:
		return "func" // Simplified for now
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.StructType:
		return "struct{}"
	case *ast.SelectorExpr:
		return s.typeToString(t.X) + "." + t.Sel.Name
	case *ast.Ellipsis:
		return "..." + s.typeToString(t.Elt)
	default:
		return "unknown"
	}
}

// isPointerType checks if a type is a pointer
func (s *SimpleScanner) isPointerType(expr ast.Expr) bool {
	_, ok := expr.(*ast.StarExpr)
	return ok
}

// isSliceType checks if a type is a slice
func (s *SimpleScanner) isSliceType(expr ast.Expr) bool {
	if arr, ok := expr.(*ast.ArrayType); ok {
		return arr.Len == nil // slice has no length
	}
	return false
}

// isMapType checks if a type is a map
func (s *SimpleScanner) isMapType(expr ast.Expr) bool {
	_, ok := expr.(*ast.MapType)
	return ok
}

// isVariadicType checks if a type is variadic
func (s *SimpleScanner) isVariadicType(expr ast.Expr) bool {
	_, ok := expr.(*ast.Ellipsis)
	return ok
}

// isFunctionVariadic checks if a function is variadic
func (s *SimpleScanner) isFunctionVariadic(funcDecl *ast.FuncDecl) bool {
	if funcDecl.Type.Params == nil || len(funcDecl.Type.Params.List) == 0 {
		return false
	}

	lastParam := funcDecl.Type.Params.List[len(funcDecl.Type.Params.List)-1]
	return s.isVariadicType(lastParam.Type)
}
