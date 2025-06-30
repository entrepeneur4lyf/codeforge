package graph

import (
	"regexp"
	"sort"
	"strings"
)

// QueryBuilder provides a fluent interface for building graph queries
type QueryBuilder struct {
	graph   *CodeGraph
	filters []QueryFilter
}

// QueryFilter represents a filter condition for graph queries
type QueryFilter struct {
	Type      string      // "node_type", "edge_type", "name", "path", "language", etc.
	Operation string      // "equals", "contains", "matches", "starts_with", "ends_with"
	Value     interface{} // The value to compare against
}

// NewQueryBuilder creates a new query builder
func NewQueryBuilder(graph *CodeGraph) *QueryBuilder {
	return &QueryBuilder{
		graph:   graph,
		filters: make([]QueryFilter, 0),
	}
}

// NodeType filters nodes by type
func (qb *QueryBuilder) NodeType(nodeType NodeType) *QueryBuilder {
	qb.filters = append(qb.filters, QueryFilter{
		Type:      "node_type",
		Operation: "equals",
		Value:     nodeType,
	})
	return qb
}

// EdgeType filters edges by type
func (qb *QueryBuilder) EdgeType(edgeType EdgeType) *QueryBuilder {
	qb.filters = append(qb.filters, QueryFilter{
		Type:      "edge_type",
		Operation: "equals",
		Value:     edgeType,
	})
	return qb
}

// NameContains filters by name containing a substring
func (qb *QueryBuilder) NameContains(substring string) *QueryBuilder {
	qb.filters = append(qb.filters, QueryFilter{
		Type:      "name",
		Operation: "contains",
		Value:     substring,
	})
	return qb
}

// NameMatches filters by name matching a regex pattern
func (qb *QueryBuilder) NameMatches(pattern string) *QueryBuilder {
	qb.filters = append(qb.filters, QueryFilter{
		Type:      "name",
		Operation: "matches",
		Value:     pattern,
	})
	return qb
}

// PathContains filters by path containing a substring
func (qb *QueryBuilder) PathContains(substring string) *QueryBuilder {
	qb.filters = append(qb.filters, QueryFilter{
		Type:      "path",
		Operation: "contains",
		Value:     substring,
	})
	return qb
}

// Language filters by programming language
func (qb *QueryBuilder) Language(language string) *QueryBuilder {
	qb.filters = append(qb.filters, QueryFilter{
		Type:      "language",
		Operation: "equals",
		Value:     language,
	})
	return qb
}

// Visibility filters by visibility (public, private, etc.)
func (qb *QueryBuilder) Visibility(visibility string) *QueryBuilder {
	qb.filters = append(qb.filters, QueryFilter{
		Type:      "visibility",
		Operation: "equals",
		Value:     visibility,
	})
	return qb
}

// Execute executes the query and returns matching nodes and edges
func (qb *QueryBuilder) Execute() *QueryResult {
	qb.graph.mutex.RLock()
	defer qb.graph.mutex.RUnlock()

	result := &QueryResult{
		Nodes:    make([]*Node, 0),
		Edges:    make([]*Edge, 0),
		Metadata: make(map[string]interface{}),
	}

	// Filter nodes
	for _, node := range qb.graph.graph.Nodes {
		if qb.matchesNodeFilters(node) {
			result.Nodes = append(result.Nodes, node)
		}
	}

	// Filter edges
	for _, edge := range qb.graph.graph.Edges {
		if qb.matchesEdgeFilters(edge) {
			result.Edges = append(result.Edges, edge)
		}
	}

	// Add metadata
	result.Metadata["total_nodes"] = len(result.Nodes)
	result.Metadata["total_edges"] = len(result.Edges)

	return result
}

// matchesNodeFilters checks if a node matches all node filters
func (qb *QueryBuilder) matchesNodeFilters(node *Node) bool {
	for _, filter := range qb.filters {
		if !qb.matchesNodeFilter(node, filter) {
			return false
		}
	}
	return true
}

// matchesEdgeFilters checks if an edge matches all edge filters
func (qb *QueryBuilder) matchesEdgeFilters(edge *Edge) bool {
	for _, filter := range qb.filters {
		if !qb.matchesEdgeFilter(edge, filter) {
			return false
		}
	}
	return true
}

// matchesNodeFilter checks if a node matches a specific filter
func (qb *QueryBuilder) matchesNodeFilter(node *Node, filter QueryFilter) bool {
	switch filter.Type {
	case "node_type":
		return filter.Operation == "equals" && node.Type == filter.Value.(NodeType)
	case "name":
		return qb.matchesStringFilter(node.Name, filter)
	case "path":
		return qb.matchesStringFilter(node.Path, filter)
	case "language":
		return filter.Operation == "equals" && node.Language == filter.Value.(string)
	case "visibility":
		return filter.Operation == "equals" && node.Visibility == filter.Value.(string)
	default:
		return true // Unknown filter types are ignored
	}
}

// matchesEdgeFilter checks if an edge matches a specific filter
func (qb *QueryBuilder) matchesEdgeFilter(edge *Edge, filter QueryFilter) bool {
	switch filter.Type {
	case "edge_type":
		return filter.Operation == "equals" && edge.Type == filter.Value.(EdgeType)
	default:
		return true // Unknown filter types are ignored
	}
}

// matchesStringFilter checks if a string matches a string filter
func (qb *QueryBuilder) matchesStringFilter(value string, filter QueryFilter) bool {
	filterValue := filter.Value.(string)

	switch filter.Operation {
	case "equals":
		return value == filterValue
	case "contains":
		return strings.Contains(value, filterValue)
	case "starts_with":
		return strings.HasPrefix(value, filterValue)
	case "ends_with":
		return strings.HasSuffix(value, filterValue)
	case "matches":
		if regex, err := regexp.Compile(filterValue); err == nil {
			return regex.MatchString(value)
		}
		return false
	default:
		return false
	}
}

// FindDependencies finds all dependencies of a given node
func (cg *CodeGraph) FindDependencies(nodeID string, maxDepth int) *QueryResult {
	cg.mutex.RLock()
	defer cg.mutex.RUnlock()

	result := &QueryResult{
		Nodes:    make([]*Node, 0),
		Edges:    make([]*Edge, 0),
		Metadata: make(map[string]interface{}),
	}

	visited := make(map[string]bool)
	queue := []struct {
		nodeID string
		depth  int
	}{{nodeID, 0}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current.nodeID] || current.depth > maxDepth {
			continue
		}

		visited[current.nodeID] = true

		if node, exists := cg.graph.Nodes[current.nodeID]; exists {
			result.Nodes = append(result.Nodes, node)
		}

		// Follow dependency edges
		for _, edgeID := range cg.graph.OutEdges[current.nodeID] {
			if edge, exists := cg.graph.Edges[edgeID]; exists {
				if edge.Type == EdgeTypeDependsOn || edge.Type == EdgeTypeImports || edge.Type == EdgeTypeUses {
					result.Edges = append(result.Edges, edge)
					if !visited[edge.Target] {
						queue = append(queue, struct {
							nodeID string
							depth  int
						}{edge.Target, current.depth + 1})
					}
				}
			}
		}
	}

	result.Metadata["max_depth"] = maxDepth
	result.Metadata["total_nodes"] = len(result.Nodes)
	result.Metadata["total_edges"] = len(result.Edges)

	return result
}

// FindDependents finds all nodes that depend on a given node
func (cg *CodeGraph) FindDependents(nodeID string, maxDepth int) *QueryResult {
	cg.mutex.RLock()
	defer cg.mutex.RUnlock()

	result := &QueryResult{
		Nodes:    make([]*Node, 0),
		Edges:    make([]*Edge, 0),
		Metadata: make(map[string]interface{}),
	}

	visited := make(map[string]bool)
	queue := []struct {
		nodeID string
		depth  int
	}{{nodeID, 0}}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if visited[current.nodeID] || current.depth > maxDepth {
			continue
		}

		visited[current.nodeID] = true

		if node, exists := cg.graph.Nodes[current.nodeID]; exists {
			result.Nodes = append(result.Nodes, node)
		}

		// Follow incoming dependency edges
		for _, edgeID := range cg.graph.InEdges[current.nodeID] {
			if edge, exists := cg.graph.Edges[edgeID]; exists {
				if edge.Type == EdgeTypeDependsOn || edge.Type == EdgeTypeImports || edge.Type == EdgeTypeUses {
					result.Edges = append(result.Edges, edge)
					if !visited[edge.Source] {
						queue = append(queue, struct {
							nodeID string
							depth  int
						}{edge.Source, current.depth + 1})
					}
				}
			}
		}
	}

	result.Metadata["max_depth"] = maxDepth
	result.Metadata["total_nodes"] = len(result.Nodes)
	result.Metadata["total_edges"] = len(result.Edges)

	return result
}

// GetMostConnectedNodes returns nodes with the highest degree (most connections)
func (cg *CodeGraph) GetMostConnectedNodes(limit int) []*Node {
	cg.mutex.RLock()
	defer cg.mutex.RUnlock()

	type nodeWithDegree struct {
		node   *Node
		degree int
	}

	nodes := make([]nodeWithDegree, 0, len(cg.graph.Nodes))

	for nodeID, node := range cg.graph.Nodes {
		degree := len(cg.graph.OutEdges[nodeID]) + len(cg.graph.InEdges[nodeID])
		nodes = append(nodes, nodeWithDegree{node, degree})
	}

	// Sort by degree (descending)
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].degree > nodes[j].degree
	})

	// Return top nodes
	result := make([]*Node, 0, limit)
	for i, nodeWithDeg := range nodes {
		if i >= limit {
			break
		}
		result = append(result, nodeWithDeg.node)
	}

	return result
}
