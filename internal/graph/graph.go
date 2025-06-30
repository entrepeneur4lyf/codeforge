package graph

import (
	"crypto/sha256"
	"fmt"
	"sync"
)

// CodeGraph implements a graph-based repository analysis system
type CodeGraph struct {
	graph *Graph
	mutex sync.RWMutex

	// Configuration
	rootPath    string
	ignoreRules []string

	// Statistics
	stats *GraphStats
}

// NewCodeGraph creates a new code graph
func NewCodeGraph(rootPath string) *CodeGraph {
	return &CodeGraph{
		graph: &Graph{
			Nodes:       make(map[string]*Node),
			Edges:       make(map[string]*Edge),
			OutEdges:    make(map[string][]string),
			InEdges:     make(map[string][]string),
			NodesByType: make(map[NodeType][]string),
			NodesByPath: make(map[string]string),
			EdgesByType: make(map[EdgeType][]string),
		},
		rootPath: rootPath,
		stats: &GraphStats{
			NodesByType: make(map[NodeType]int),
			EdgesByType: make(map[EdgeType]int),
			Languages:   make(map[string]int),
		},
	}
}

// AddNode adds a node to the graph
func (cg *CodeGraph) AddNode(node *Node) error {
	cg.mutex.Lock()
	defer cg.mutex.Unlock()

	if node.ID == "" {
		node.ID = cg.generateNodeID(node)
	}

	// Check if node already exists
	if _, exists := cg.graph.Nodes[node.ID]; exists {
		return fmt.Errorf("node with ID %s already exists", node.ID)
	}

	// Add to main nodes map
	cg.graph.Nodes[node.ID] = node

	// Update indexes
	cg.graph.NodesByType[node.Type] = append(cg.graph.NodesByType[node.Type], node.ID)
	if node.Path != "" {
		cg.graph.NodesByPath[node.Path] = node.ID
	}

	// Initialize edge lists
	cg.graph.OutEdges[node.ID] = make([]string, 0)
	cg.graph.InEdges[node.ID] = make([]string, 0)

	// Update statistics
	cg.stats.NodesByType[node.Type]++
	cg.stats.NodeCount++
	if node.Language != "" {
		cg.stats.Languages[node.Language]++
	}

	return nil
}

// AddEdge adds an edge to the graph
func (cg *CodeGraph) AddEdge(edge *Edge) error {
	cg.mutex.Lock()
	defer cg.mutex.Unlock()

	if edge.ID == "" {
		edge.ID = cg.generateEdgeID(edge)
	}

	// Check if source and target nodes exist
	if _, exists := cg.graph.Nodes[edge.Source]; !exists {
		return fmt.Errorf("source node %s does not exist", edge.Source)
	}
	if _, exists := cg.graph.Nodes[edge.Target]; !exists {
		return fmt.Errorf("target node %s does not exist", edge.Target)
	}

	// Check if edge already exists
	if _, exists := cg.graph.Edges[edge.ID]; exists {
		return fmt.Errorf("edge with ID %s already exists", edge.ID)
	}

	// Add to main edges map
	cg.graph.Edges[edge.ID] = edge

	// Update adjacency lists
	cg.graph.OutEdges[edge.Source] = append(cg.graph.OutEdges[edge.Source], edge.ID)
	cg.graph.InEdges[edge.Target] = append(cg.graph.InEdges[edge.Target], edge.ID)

	// Update indexes
	cg.graph.EdgesByType[edge.Type] = append(cg.graph.EdgesByType[edge.Type], edge.ID)

	// Update statistics
	cg.stats.EdgesByType[edge.Type]++
	cg.stats.EdgeCount++

	return nil
}

// GetNode retrieves a node by ID
func (cg *CodeGraph) GetNode(id string) (*Node, bool) {
	cg.mutex.RLock()
	defer cg.mutex.RUnlock()

	node, exists := cg.graph.Nodes[id]
	return node, exists
}

// GetNodeByPath retrieves a node by file path
func (cg *CodeGraph) GetNodeByPath(path string) (*Node, bool) {
	cg.mutex.RLock()
	defer cg.mutex.RUnlock()

	if nodeID, exists := cg.graph.NodesByPath[path]; exists {
		return cg.graph.Nodes[nodeID], true
	}
	return nil, false
}

// GetNodesByType retrieves all nodes of a specific type
func (cg *CodeGraph) GetNodesByType(nodeType NodeType) []*Node {
	cg.mutex.RLock()
	defer cg.mutex.RUnlock()

	nodeIDs := cg.graph.NodesByType[nodeType]
	nodes := make([]*Node, 0, len(nodeIDs))

	for _, id := range nodeIDs {
		if node, exists := cg.graph.Nodes[id]; exists {
			nodes = append(nodes, node)
		}
	}

	return nodes
}

// GetAllNodes returns all nodes in the graph
func (cg *CodeGraph) GetAllNodes() []*Node {
	cg.mutex.RLock()
	defer cg.mutex.RUnlock()

	nodes := make([]*Node, 0, len(cg.graph.Nodes))
	for _, node := range cg.graph.Nodes {
		nodes = append(nodes, node)
	}

	return nodes
}

// GetOutgoingEdges returns all outgoing edges from a node
func (cg *CodeGraph) GetOutgoingEdges(nodeID string) []*Edge {
	cg.mutex.RLock()
	defer cg.mutex.RUnlock()

	edgeIDs := cg.graph.OutEdges[nodeID]
	edges := make([]*Edge, 0, len(edgeIDs))

	for _, id := range edgeIDs {
		if edge, exists := cg.graph.Edges[id]; exists {
			edges = append(edges, edge)
		}
	}

	return edges
}

// GetIncomingEdges returns all incoming edges to a node
func (cg *CodeGraph) GetIncomingEdges(nodeID string) []*Edge {
	cg.mutex.RLock()
	defer cg.mutex.RUnlock()

	edgeIDs := cg.graph.InEdges[nodeID]
	edges := make([]*Edge, 0, len(edgeIDs))

	for _, id := range edgeIDs {
		if edge, exists := cg.graph.Edges[id]; exists {
			edges = append(edges, edge)
		}
	}

	return edges
}

// GetNeighbors returns all neighboring nodes (connected by any edge)
func (cg *CodeGraph) GetNeighbors(nodeID string) []*Node {
	cg.mutex.RLock()
	defer cg.mutex.RUnlock()

	neighborMap := make(map[string]*Node)

	// Add targets of outgoing edges
	for _, edgeID := range cg.graph.OutEdges[nodeID] {
		if edge, exists := cg.graph.Edges[edgeID]; exists {
			if target, exists := cg.graph.Nodes[edge.Target]; exists {
				neighborMap[edge.Target] = target
			}
		}
	}

	// Add sources of incoming edges
	for _, edgeID := range cg.graph.InEdges[nodeID] {
		if edge, exists := cg.graph.Edges[edgeID]; exists {
			if source, exists := cg.graph.Nodes[edge.Source]; exists {
				neighborMap[edge.Source] = source
			}
		}
	}

	// Convert map to slice
	neighbors := make([]*Node, 0, len(neighborMap))
	for _, node := range neighborMap {
		neighbors = append(neighbors, node)
	}

	return neighbors
}

// GetStats returns current graph statistics
func (cg *CodeGraph) GetStats() *GraphStats {
	cg.mutex.RLock()
	defer cg.mutex.RUnlock()

	// Create a copy to avoid race conditions
	stats := &GraphStats{
		NodeCount:    cg.stats.NodeCount,
		EdgeCount:    cg.stats.EdgeCount,
		NodesByType:  make(map[NodeType]int),
		EdgesByType:  make(map[EdgeType]int),
		Languages:    make(map[string]int),
		LastUpdated:  cg.stats.LastUpdated,
		ScanDuration: cg.stats.ScanDuration,
	}

	for k, v := range cg.stats.NodesByType {
		stats.NodesByType[k] = v
	}
	for k, v := range cg.stats.EdgesByType {
		stats.EdgesByType[k] = v
	}
	for k, v := range cg.stats.Languages {
		stats.Languages[k] = v
	}

	return stats
}

// generateNodeID generates a unique ID for a node
func (cg *CodeGraph) generateNodeID(node *Node) string {
	data := fmt.Sprintf("%s:%s:%s", node.Type, node.Name, node.Path)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("node_%x", hash[:8])
}

// generateEdgeID generates a unique ID for an edge
func (cg *CodeGraph) generateEdgeID(edge *Edge) string {
	data := fmt.Sprintf("%s:%s:%s", edge.Type, edge.Source, edge.Target)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("edge_%x", hash[:8])
}

// Clear removes all nodes and edges from the graph
func (cg *CodeGraph) Clear() {
	cg.mutex.Lock()
	defer cg.mutex.Unlock()

	cg.graph.Nodes = make(map[string]*Node)
	cg.graph.Edges = make(map[string]*Edge)
	cg.graph.OutEdges = make(map[string][]string)
	cg.graph.InEdges = make(map[string][]string)
	cg.graph.NodesByType = make(map[NodeType][]string)
	cg.graph.NodesByPath = make(map[string]string)
	cg.graph.EdgesByType = make(map[EdgeType][]string)

	cg.stats = &GraphStats{
		NodesByType: make(map[NodeType]int),
		EdgesByType: make(map[EdgeType]int),
		Languages:   make(map[string]int),
	}
}

// FindPath finds a path between two nodes using BFS
func (cg *CodeGraph) FindPath(sourceID, targetID string) ([]*Node, []*Edge, bool) {
	cg.mutex.RLock()
	defer cg.mutex.RUnlock()

	if sourceID == targetID {
		if node, exists := cg.graph.Nodes[sourceID]; exists {
			return []*Node{node}, []*Edge{}, true
		}
		return nil, nil, false
	}

	// BFS to find shortest path
	queue := []string{sourceID}
	visited := make(map[string]bool)
	parent := make(map[string]string)
	parentEdge := make(map[string]string)

	visited[sourceID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current == targetID {
			// Reconstruct path
			path := []string{}
			edges := []string{}

			for node := targetID; node != sourceID; node = parent[node] {
				path = append(path, node)
				edges = append(edges, parentEdge[node])
			}
			path = append(path, sourceID)

			// Reverse path and edges
			for i := 0; i < len(path)/2; i++ {
				path[i], path[len(path)-1-i] = path[len(path)-1-i], path[i]
			}
			for i := 0; i < len(edges)/2; i++ {
				edges[i], edges[len(edges)-1-i] = edges[len(edges)-1-i], edges[i]
			}

			// Convert to nodes and edges
			nodes := make([]*Node, len(path))
			for i, nodeID := range path {
				nodes[i] = cg.graph.Nodes[nodeID]
			}

			edgeObjs := make([]*Edge, len(edges))
			for i, edgeID := range edges {
				edgeObjs[i] = cg.graph.Edges[edgeID]
			}

			return nodes, edgeObjs, true
		}

		// Explore neighbors
		for _, edgeID := range cg.graph.OutEdges[current] {
			if edge, exists := cg.graph.Edges[edgeID]; exists {
				if !visited[edge.Target] {
					visited[edge.Target] = true
					parent[edge.Target] = current
					parentEdge[edge.Target] = edgeID
					queue = append(queue, edge.Target)
				}
			}
		}
	}

	return nil, nil, false
}

// GetConnectedComponents finds all connected components in the graph
func (cg *CodeGraph) GetConnectedComponents() [][]*Node {
	cg.mutex.RLock()
	defer cg.mutex.RUnlock()

	visited := make(map[string]bool)
	components := [][]*Node{}

	for nodeID := range cg.graph.Nodes {
		if !visited[nodeID] {
			component := cg.dfsComponent(nodeID, visited)
			if len(component) > 0 {
				components = append(components, component)
			}
		}
	}

	return components
}

// dfsComponent performs DFS to find a connected component
func (cg *CodeGraph) dfsComponent(startID string, visited map[string]bool) []*Node {
	stack := []string{startID}
	component := []*Node{}

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if visited[current] {
			continue
		}

		visited[current] = true
		if node, exists := cg.graph.Nodes[current]; exists {
			component = append(component, node)
		}

		// Add neighbors to stack
		for _, edgeID := range cg.graph.OutEdges[current] {
			if edge, exists := cg.graph.Edges[edgeID]; exists {
				if !visited[edge.Target] {
					stack = append(stack, edge.Target)
				}
			}
		}

		for _, edgeID := range cg.graph.InEdges[current] {
			if edge, exists := cg.graph.Edges[edgeID]; exists {
				if !visited[edge.Source] {
					stack = append(stack, edge.Source)
				}
			}
		}
	}

	return component
}
