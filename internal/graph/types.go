package graph

import (
	"time"
)

// NodeType represents different types of nodes in the code graph
type NodeType string

const (
	NodeTypeFile          NodeType = "file"
	NodeTypeDirectory     NodeType = "directory"
	NodeTypeModule        NodeType = "module"
	NodeTypePackage       NodeType = "package"
	NodeTypeFunction      NodeType = "function"
	NodeTypeStruct        NodeType = "struct"
	NodeTypeInterface     NodeType = "interface"
	NodeTypeVariable      NodeType = "variable"
	NodeTypeConstant      NodeType = "constant"
	NodeTypeType          NodeType = "type"
	NodeTypeMethod        NodeType = "method"
	NodeTypeField         NodeType = "field"
	NodeTypeImport        NodeType = "import"
	NodeTypeConfig        NodeType = "config"
	NodeTypeTest          NodeType = "test"
	NodeTypeDocumentation NodeType = "documentation"
	NodeTypeAsset         NodeType = "asset"
	NodeTypeBinary        NodeType = "binary"
)

// EdgeType represents different types of relationships between nodes
type EdgeType string

const (
	EdgeTypeContains   EdgeType = "contains"   // Directory contains file
	EdgeTypeImports    EdgeType = "imports"    // File imports module
	EdgeTypeDefines    EdgeType = "defines"    // File defines function/struct
	EdgeTypeCalls      EdgeType = "calls"      // Function calls function
	EdgeTypeImplements EdgeType = "implements" // Struct implements interface
	EdgeTypeExtends    EdgeType = "extends"    // Struct embeds struct
	EdgeTypeUses       EdgeType = "uses"       // Function uses variable
	EdgeTypeReferences EdgeType = "references" // Code references symbol
	EdgeTypeDependsOn  EdgeType = "depends_on" // Module depends on module
	EdgeTypeTests      EdgeType = "tests"      // Test file tests code
	EdgeTypeDocuments  EdgeType = "documents"  // Doc file documents code
	EdgeTypeConfigures EdgeType = "configures" // Config file configures module
	EdgeTypeRelatedTo  EdgeType = "related_to" // General relationship
)

// Node represents a node in the code graph
type Node struct {
	ID           string                 `json:"id"`
	Type         NodeType               `json:"type"`
	Name         string                 `json:"name"`
	Path         string                 `json:"path,omitempty"`
	Purpose      string                 `json:"purpose,omitempty"`
	Language     string                 `json:"language,omitempty"`
	LastModified time.Time              `json:"last_modified,omitempty"`
	Size         int64                  `json:"size,omitempty"`
	Visibility   string                 `json:"visibility,omitempty"`
	Signature    string                 `json:"signature,omitempty"`
	DocComment   string                 `json:"doc_comment,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`

	// Code-specific fields
	StartLine    int      `json:"start_line,omitempty"`
	EndLine      int      `json:"end_line,omitempty"`
	Imports      []string `json:"imports,omitempty"`
	Exports      []string `json:"exports,omitempty"`
	Dependencies []string `json:"dependencies,omitempty"`

	// Enhanced API/Structure information
	APISignature  *APISignature  `json:"api_signature,omitempty"`  // Full API signature details
	DataStructure *DataStructure `json:"data_structure,omitempty"` // Struct/interface details
	FunctionInfo  *FunctionInfo  `json:"function_info,omitempty"`  // Function-specific details
	TypeInfo      *TypeInfo      `json:"type_info,omitempty"`      // Type definition details
}

// Edge represents an edge in the code graph
type Edge struct {
	ID       string                 `json:"id"`
	Type     EdgeType               `json:"type"`
	Source   string                 `json:"source"`
	Target   string                 `json:"target"`
	Weight   float64                `json:"weight,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// Graph represents the code graph structure
type Graph struct {
	Nodes map[string]*Node `json:"nodes"`
	Edges map[string]*Edge `json:"edges"`

	// Adjacency lists for efficient traversal
	OutEdges map[string][]string `json:"-"` // node_id -> []edge_id
	InEdges  map[string][]string `json:"-"` // node_id -> []edge_id

	// Indexes for fast lookups
	NodesByType map[NodeType][]string `json:"-"` // type -> []node_id
	NodesByPath map[string]string     `json:"-"` // path -> node_id
	EdgesByType map[EdgeType][]string `json:"-"` // type -> []edge_id
}

// GraphStats represents statistics about the graph
type GraphStats struct {
	NodeCount    int              `json:"node_count"`
	EdgeCount    int              `json:"edge_count"`
	NodesByType  map[NodeType]int `json:"nodes_by_type"`
	EdgesByType  map[EdgeType]int `json:"edges_by_type"`
	Languages    map[string]int   `json:"languages"`
	LastUpdated  time.Time        `json:"last_updated"`
	ScanDuration time.Duration    `json:"scan_duration"`
}

// QueryResult represents the result of a graph query
type QueryResult struct {
	Nodes    []*Node                `json:"nodes"`
	Edges    []*Edge                `json:"edges"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// PathInfo represents information about a file path
type PathInfo struct {
	Path         string    `json:"path"`
	IsDirectory  bool      `json:"is_directory"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"last_modified"`
	Language     string    `json:"language,omitempty"`
	IsIgnored    bool      `json:"is_ignored"`
}

// SymbolInfo represents information about a code symbol
type SymbolInfo struct {
	Name       string   `json:"name"`
	Type       NodeType `json:"type"`
	Signature  string   `json:"signature,omitempty"`
	DocComment string   `json:"doc_comment,omitempty"`
	Visibility string   `json:"visibility"`
	StartLine  int      `json:"start_line"`
	EndLine    int      `json:"end_line"`
	File       string   `json:"file"`
}

// ImportInfo represents information about an import
type ImportInfo struct {
	Path     string `json:"path"`
	Alias    string `json:"alias,omitempty"`
	IsStdLib bool   `json:"is_stdlib"`
	IsLocal  bool   `json:"is_local"`
	Line     int    `json:"line"`
}

// DependencyInfo represents dependency information
type DependencyInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	Type    string `json:"type"` // direct, indirect, dev, etc.
	Source  string `json:"source,omitempty"`
}

// APISignature represents a complete API signature
type APISignature struct {
	Name        string       `json:"name"`
	FullName    string       `json:"full_name"` // Package.Type.Method or Package.Function
	Parameters  []Parameter  `json:"parameters"`
	ReturnTypes []ReturnType `json:"return_types"`
	Receiver    *Receiver    `json:"receiver,omitempty"` // For methods
	IsExported  bool         `json:"is_exported"`
	IsMethod    bool         `json:"is_method"`
	IsVariadic  bool         `json:"is_variadic"`
	DocComment  string       `json:"doc_comment"`
	Examples    []string     `json:"examples,omitempty"`
	Tags        []string     `json:"tags,omitempty"` // build tags, etc.
}

// Parameter represents a function parameter
type Parameter struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	IsPointer    bool   `json:"is_pointer"`
	IsSlice      bool   `json:"is_slice"`
	IsMap        bool   `json:"is_map"`
	IsVariadic   bool   `json:"is_variadic"`
	DefaultValue string `json:"default_value,omitempty"`
	DocComment   string `json:"doc_comment,omitempty"`
}

// ReturnType represents a function return type
type ReturnType struct {
	Type      string `json:"type"`
	Name      string `json:"name,omitempty"` // Named returns
	IsPointer bool   `json:"is_pointer"`
	IsSlice   bool   `json:"is_slice"`
	IsMap     bool   `json:"is_map"`
	IsError   bool   `json:"is_error"`
}

// Receiver represents a method receiver
type Receiver struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	IsPointer bool   `json:"is_pointer"`
}

// DataStructure represents struct/interface/type definitions
type DataStructure struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"` // "struct", "interface", "type"
	Package     string   `json:"package"`
	Fields      []Field  `json:"fields,omitempty"`     // For structs
	Methods     []Method `json:"methods,omitempty"`    // For interfaces/structs
	Embedded    []string `json:"embedded,omitempty"`   // Embedded types
	Implements  []string `json:"implements,omitempty"` // Implemented interfaces
	IsExported  bool     `json:"is_exported"`
	DocComment  string   `json:"doc_comment"`
	Tags        []string `json:"tags,omitempty"`
	Constraints []string `json:"constraints,omitempty"` // Generic constraints
}

// Field represents a struct field
type Field struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Tag        string `json:"tag,omitempty"` // struct tags
	IsExported bool   `json:"is_exported"`
	IsPointer  bool   `json:"is_pointer"`
	IsSlice    bool   `json:"is_slice"`
	IsMap      bool   `json:"is_map"`
	DocComment string `json:"doc_comment,omitempty"`
}

// Method represents a method signature
type Method struct {
	Name        string       `json:"name"`
	Parameters  []Parameter  `json:"parameters"`
	ReturnTypes []ReturnType `json:"return_types"`
	IsExported  bool         `json:"is_exported"`
	DocComment  string       `json:"doc_comment,omitempty"`
}

// FunctionInfo represents detailed function information
type FunctionInfo struct {
	Name        string       `json:"name"`
	Package     string       `json:"package"`
	Parameters  []Parameter  `json:"parameters"`
	ReturnTypes []ReturnType `json:"return_types"`
	Receiver    *Receiver    `json:"receiver,omitempty"`
	IsExported  bool         `json:"is_exported"`
	IsMethod    bool         `json:"is_method"`
	IsVariadic  bool         `json:"is_variadic"`
	IsGeneric   bool         `json:"is_generic"`
	Complexity  int          `json:"complexity,omitempty"` // Cyclomatic complexity
	LineCount   int          `json:"line_count,omitempty"`
	CallsCount  int          `json:"calls_count,omitempty"` // Number of function calls
	CalledBy    []string     `json:"called_by,omitempty"`   // Functions that call this
	Calls       []string     `json:"calls,omitempty"`       // Functions this calls
	UsesTypes   []string     `json:"uses_types,omitempty"`  // Types used by this function
	DocComment  string       `json:"doc_comment"`
	Examples    []string     `json:"examples,omitempty"`
}

// TypeInfo represents type definition information
type TypeInfo struct {
	Name           string   `json:"name"`
	Package        string   `json:"package"`
	Kind           string   `json:"kind"`                      // "struct", "interface", "alias", "basic"
	UnderlyingType string   `json:"underlying_type,omitempty"` // For type aliases
	IsExported     bool     `json:"is_exported"`
	IsGeneric      bool     `json:"is_generic"`
	Constraints    []string `json:"constraints,omitempty"` // Generic type constraints
	UsedBy         []string `json:"used_by,omitempty"`     // Where this type is used
	DocComment     string   `json:"doc_comment"`
	Size           int      `json:"size,omitempty"` // Estimated size in bytes
}
