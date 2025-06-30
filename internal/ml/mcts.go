package ml

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/graph"
)

// CodeMCTS implements Monte Carlo Tree Search for intelligent code exploration
type CodeMCTS struct {
	root             *MCTSNode
	graph            *graph.CodeGraph
	explorationParam float64 // UCB1 exploration parameter (typically √2)
	maxIterations    int
	timeLimit        time.Duration

	// Learning components
	valueNetwork *ValueNetwork
	experiences  []Experience
	mutex        sync.RWMutex
}

// MCTSNode represents a node in the MCTS tree for code exploration
type MCTSNode struct {
	// Code context
	NodeID   string             // Graph node ID
	CodePath string             // File/function path
	Query    string             // Current search query
	Context  map[string]float64 // Accumulated context relevance

	// MCTS data
	Parent       *MCTSNode
	Children     []*MCTSNode
	Visits       int
	Value        float64  // Average reward
	UntriedMoves []string // Unexplored neighboring nodes

	// State information
	Depth      int
	IsTerminal bool
	LastReward float64
}

// Experience type is defined in types.go

// CodeState is defined in types.go

// NewCodeMCTS creates a new MCTS instance for code exploration
func NewCodeMCTS(codeGraph *graph.CodeGraph) *CodeMCTS {
	return &CodeMCTS{
		graph:            codeGraph,
		explorationParam: math.Sqrt(2), // Standard UCB1 parameter
		maxIterations:    1000,
		timeLimit:        5 * time.Second,
		valueNetwork:     NewValueNetwork(),
		experiences:      make([]Experience, 0),
	}
}

// Search performs MCTS to find the most relevant code for a query
func (mcts *CodeMCTS) Search(ctx context.Context, query string, startNodeID string) (*SearchResult, error) {
	// Initialize root node
	mcts.root = &MCTSNode{
		NodeID:       startNodeID,
		Query:        query,
		Context:      make(map[string]float64),
		UntriedMoves: mcts.getNeighborNodes(startNodeID),
		Depth:        0,
	}

	// Set up timeout context
	searchCtx, cancel := context.WithTimeout(ctx, mcts.timeLimit)
	defer cancel()

	// MCTS main loop
	iterations := 0
	for iterations < mcts.maxIterations {
		select {
		case <-searchCtx.Done():
			return mcts.getBestResult(), nil
		default:
			// 1. Selection - traverse tree using UCB1
			leaf := mcts.selectNode(mcts.root)

			// 2. Expansion - add new child if not terminal
			if !leaf.IsTerminal && len(leaf.UntriedMoves) > 0 {
				leaf = mcts.expand(leaf)
			}

			// 3. Simulation - evaluate the leaf node
			reward := mcts.simulate(leaf, query)

			// 4. Backpropagation - update values up the tree
			mcts.backpropagate(leaf, reward)

			// Store experience for learning
			mcts.storeExperience(leaf, reward, query)

			iterations++
		}
	}

	// Return best path found
	return mcts.getBestResult(), nil
}

// selectNode traverses the tree using UCB1 to find the most promising leaf
func (mcts *CodeMCTS) selectNode(node *MCTSNode) *MCTSNode {
	for len(node.Children) > 0 && len(node.UntriedMoves) == 0 {
		node = mcts.selectBestChild(node)
	}
	return node
}

// selectBestChild chooses the child with highest UCB1 value
func (mcts *CodeMCTS) selectBestChild(node *MCTSNode) *MCTSNode {
	bestChild := node.Children[0]
	bestValue := mcts.calculateUCB1(bestChild, node.Visits)

	for _, child := range node.Children[1:] {
		value := mcts.calculateUCB1(child, node.Visits)
		if value > bestValue {
			bestValue = value
			bestChild = child
		}
	}

	return bestChild
}

// calculateUCB1 computes the Upper Confidence Bound for node selection
func (mcts *CodeMCTS) calculateUCB1(node *MCTSNode, parentVisits int) float64 {
	if node.Visits == 0 {
		return math.Inf(1) // Unvisited nodes have infinite value
	}

	exploitation := node.Value / float64(node.Visits)
	exploration := mcts.explorationParam * math.Sqrt(math.Log(float64(parentVisits))/float64(node.Visits))

	return exploitation + exploration
}

// expand adds a new child node to the tree
func (mcts *CodeMCTS) expand(node *MCTSNode) *MCTSNode {
	if len(node.UntriedMoves) == 0 {
		return node
	}

	// Select random untried move
	moveIndex := rand.Intn(len(node.UntriedMoves))
	selectedMove := node.UntriedMoves[moveIndex]

	// Remove from untried moves
	node.UntriedMoves = append(node.UntriedMoves[:moveIndex], node.UntriedMoves[moveIndex+1:]...)

	// Create new child node
	child := &MCTSNode{
		NodeID:       selectedMove,
		Query:        node.Query,
		Context:      mcts.copyContext(node.Context),
		Parent:       node,
		UntriedMoves: mcts.getNeighborNodes(selectedMove),
		Depth:        node.Depth + 1,
		IsTerminal:   mcts.isTerminalNode(selectedMove, node.Depth+1),
	}

	// Set code path
	if graphNode, exists := mcts.graph.GetNode(selectedMove); exists {
		child.CodePath = graphNode.Path
	}

	node.Children = append(node.Children, child)
	return child
}

// simulate evaluates a node by calculating its relevance to the query
func (mcts *CodeMCTS) simulate(node *MCTSNode, query string) float64 {
	// Get the graph node
	graphNode, exists := mcts.graph.GetNode(node.NodeID)
	if !exists {
		return 0.0
	}

	// Calculate base relevance score
	relevance := mcts.calculateRelevance(graphNode, query)

	// Use value network if available
	if mcts.valueNetwork != nil {
		state := mcts.nodeToState(node)
		networkValue := mcts.valueNetwork.Predict(state, query)
		relevance = 0.7*relevance + 0.3*networkValue // Blend heuristic and learned value
	}

	// Apply depth penalty (prefer shorter paths)
	depthPenalty := math.Exp(-0.1 * float64(node.Depth))

	// Apply diversity bonus (prefer exploring different types of nodes)
	diversityBonus := mcts.calculateDiversityBonus(node)

	finalScore := relevance * depthPenalty * diversityBonus
	node.LastReward = finalScore

	return finalScore
}

// calculateRelevance computes how relevant a code node is to the query
func (mcts *CodeMCTS) calculateRelevance(node *graph.Node, query string) float64 {
	score := 0.0
	queryLower := strings.ToLower(query)
	queryWords := strings.Fields(queryLower)

	// Name matching
	if strings.Contains(strings.ToLower(node.Name), queryLower) {
		score += 2.0
	}

	// Path matching
	if strings.Contains(strings.ToLower(node.Path), queryLower) {
		score += 1.5
	}

	// Purpose/doc matching
	if strings.Contains(strings.ToLower(node.Purpose), queryLower) {
		score += 1.0
	}

	if strings.Contains(strings.ToLower(node.DocComment), queryLower) {
		score += 1.0
	}

	// Word-level matching
	for _, word := range queryWords {
		allText := strings.ToLower(fmt.Sprintf("%s %s %s %s",
			node.Name, node.Path, node.Purpose, node.DocComment))
		if strings.Contains(allText, word) {
			score += 0.5
		}
	}

	// Type-based bonuses
	switch node.Type {
	case graph.NodeTypeFunction:
		if node.Visibility == "public" {
			score += 0.5
		}
	case graph.NodeTypeStruct, graph.NodeTypeInterface:
		score += 0.3
	case graph.NodeTypeFile:
		score += 0.2
	}

	// API signature bonus
	if node.APISignature != nil && node.APISignature.IsExported {
		score += 0.5
	}

	return math.Min(score, 5.0) // Cap at 5.0
}

// backpropagate updates node values up the tree
func (mcts *CodeMCTS) backpropagate(node *MCTSNode, reward float64) {
	current := node
	for current != nil {
		current.Visits++
		current.Value += reward
		current = current.Parent
	}
}

// Helper methods

func (mcts *CodeMCTS) getNeighborNodes(nodeID string) []string {
	neighbors := mcts.graph.GetNeighbors(nodeID)
	nodeIDs := make([]string, len(neighbors))
	for i, neighbor := range neighbors {
		nodeIDs[i] = neighbor.ID
	}
	return nodeIDs
}

func (mcts *CodeMCTS) isTerminalNode(nodeID string, depth int) bool {
	// Terminal conditions
	if depth >= 10 { // Max depth
		return true
	}

	neighbors := mcts.getNeighborNodes(nodeID)
	return len(neighbors) == 0 // No more neighbors to explore
}

func (mcts *CodeMCTS) copyContext(context map[string]float64) map[string]float64 {
	copy := make(map[string]float64)
	for k, v := range context {
		copy[k] = v
	}
	return copy
}

func (mcts *CodeMCTS) calculateDiversityBonus(node *MCTSNode) float64 {
	// Bonus for exploring different types of nodes
	visitedTypes := make(map[graph.NodeType]bool)
	current := node
	for current != nil {
		if graphNode, exists := mcts.graph.GetNode(current.NodeID); exists {
			visitedTypes[graphNode.Type] = true
		}
		current = current.Parent
	}

	return 1.0 + 0.1*float64(len(visitedTypes))
}

func (mcts *CodeMCTS) nodeToState(node *MCTSNode) CodeState {
	visitedNodes := make([]string, 0)
	current := node
	for current != nil {
		visitedNodes = append(visitedNodes, current.NodeID)
		current = current.Parent
	}

	return CodeState{
		CurrentNode:    node.NodeID,
		Query:          node.Query,
		QueryTerms:     strings.Fields(strings.ToLower(node.Query)),
		VisitedNodes:   visitedNodes,
		NodeTypes:      make(map[string]int), // Would need to be populated
		PathDepth:      node.Depth,
		RelevanceScore: node.LastReward,
		LastAction:     "", // Would need to track this
		Context:        node.Context,
	}
}

// storeExperience stores learning experience (placeholder)
func (mcts *CodeMCTS) storeExperience(node *MCTSNode, reward float64, query string) {
	// Create experience from node
	state := mcts.nodeToState(node)

	experience := Experience{
		State:     state,
		Action:    node.NodeID, // Simplified
		Reward:    reward,
		NextState: state, // Simplified
		Query:     query,
		Timestamp: time.Now(),
	}

	// Thread-safe experience storage
	mcts.mutex.Lock()
	mcts.experiences = append(mcts.experiences, experience)
	mcts.mutex.Unlock()
}

// getBestResult returns the best search result
func (mcts *CodeMCTS) getBestResult() *SearchResult {
	if mcts.root == nil {
		return &SearchResult{
			BestPath:    []string{},
			Confidence:  0.0,
			Relevance:   0.0,
			Explanation: "No search performed",
			Metadata:    map[string]any{},
			Experiences: []Experience{},
		}
	}

	// Find best child path
	bestPath := mcts.extractBestPath(mcts.root)

	// Calculate confidence based on visit counts
	confidence := 0.0
	if mcts.root.Visits > 0 {
		confidence = mcts.root.Value / float64(mcts.root.Visits)
	}

	// Thread-safe experience access
	mcts.mutex.RLock()
	experiencesCopy := make([]Experience, len(mcts.experiences))
	copy(experiencesCopy, mcts.experiences)
	mcts.mutex.RUnlock()

	return &SearchResult{
		BestPath:    bestPath,
		Confidence:  confidence,
		Relevance:   confidence, // Simplified
		Explanation: fmt.Sprintf("MCTS search with %d iterations", mcts.root.Visits),
		Metadata: map[string]any{
			"iterations": mcts.root.Visits,
			"algorithm":  "MCTS",
		},
		Experiences: experiencesCopy,
	}
}

// extractBestPath extracts the best path from the MCTS tree
func (mcts *CodeMCTS) extractBestPath(node *MCTSNode) []string {
	path := []string{node.NodeID}

	current := node
	for len(current.Children) > 0 {
		// Find child with highest value
		bestChild := current.Children[0]
		bestValue := 0.0
		if bestChild.Visits > 0 {
			bestValue = bestChild.Value / float64(bestChild.Visits)
		}

		for _, child := range current.Children[1:] {
			childValue := 0.0
			if child.Visits > 0 {
				childValue = child.Value / float64(child.Visits)
			}

			if childValue > bestValue {
				bestValue = childValue
				bestChild = child
			}
		}

		path = append(path, bestChild.NodeID)
		current = bestChild

		// Prevent infinite loops
		if len(path) > 10 {
			break
		}
	}

	return path
}
