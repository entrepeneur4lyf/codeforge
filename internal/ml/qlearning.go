package ml

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/graph"
	"github.com/entrepeneur4lyf/codeforge/internal/vectordb"
)

// CodeQLearning implements Q-Learning for intelligent code navigation
type CodeQLearning struct {
	db             *sql.DB
	graph          *graph.CodeGraph
	learningRate   float64 // Alpha: how much to update Q-values
	discountFactor float64 // Gamma: importance of future rewards
	epsilon        float64 // Exploration rate
	epsilonDecay   float64 // How fast epsilon decreases
	minEpsilon     float64 // Minimum exploration rate

	// State and action spaces
	stateEncoder *StateEncoder
	actionSpace  []string // Possible actions (navigation types)

	// In-memory Q-table for when db is nil
	qTable map[string]map[string]float64 // state_hash -> action -> q_value
	mutex  sync.RWMutex                  // Thread-safe access to qTable
}

// QEntry represents a Q-table entry in the database
type QEntry struct {
	ID          int64     `json:"id"`
	StateHash   string    `json:"state_hash"` // Hash of the state
	Action      string    `json:"action"`     // Action taken
	QValue      float64   `json:"q_value"`    // Q-value
	Visits      int       `json:"visits"`     // Number of times visited
	LastUpdated time.Time `json:"last_updated"`

	// Additional metadata
	StateData string  `json:"state_data"`  // JSON of full state
	Reward    float64 `json:"last_reward"` // Last reward received
	Query     string  `json:"query"`       // Original query context
}

// Types are now defined in types.go

// StateEncoder encodes code states into consistent hash keys
type StateEncoder struct {
	featureWeights map[string]float64
}

// NewCodeQLearning creates a new Q-Learning agent for code navigation
func NewCodeQLearning(db *sql.DB, codeGraph *graph.CodeGraph) (*CodeQLearning, error) {
	ql := &CodeQLearning{
		db:             db,
		graph:          codeGraph,
		learningRate:   0.1,
		discountFactor: 0.9,
		epsilon:        0.3,
		epsilonDecay:   0.995,
		minEpsilon:     0.05,
		stateEncoder:   NewStateEncoder(),
		actionSpace: []string{
			"follow_import",
			"explore_caller",
			"check_definition",
			"find_usage",
			"explore_sibling",
			"go_to_parent",
			"explore_child",
			"find_related",
		},
	}

	// Initialize database tables
	if err := ql.initializeTables(); err != nil {
		return nil, fmt.Errorf("failed to initialize Q-learning tables: %w", err)
	}

	return ql, nil
}

// initializeTables creates the necessary database tables
func (ql *CodeQLearning) initializeTables() error {
	createQTableSQL := `
	CREATE TABLE IF NOT EXISTS q_learning_table (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		state_hash TEXT NOT NULL,
		action TEXT NOT NULL,
		q_value REAL NOT NULL DEFAULT 0.0,
		visits INTEGER NOT NULL DEFAULT 0,
		last_updated DATETIME DEFAULT CURRENT_TIMESTAMP,
		state_data TEXT,
		last_reward REAL DEFAULT 0.0,
		query TEXT,
		UNIQUE(state_hash, action)
	);
	
	CREATE INDEX IF NOT EXISTS idx_q_state_hash ON q_learning_table(state_hash);
	CREATE INDEX IF NOT EXISTS idx_q_action ON q_learning_table(action);
	CREATE INDEX IF NOT EXISTS idx_q_updated ON q_learning_table(last_updated);
	`

	createExperienceSQL := `
	CREATE TABLE IF NOT EXISTS q_learning_experiences (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		state_hash TEXT NOT NULL,
		action TEXT NOT NULL,
		reward REAL NOT NULL,
		next_state_hash TEXT NOT NULL,
		query TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		user_feedback REAL,
		session_id TEXT
	);
	
	CREATE INDEX IF NOT EXISTS idx_exp_timestamp ON q_learning_experiences(timestamp);
	CREATE INDEX IF NOT EXISTS idx_exp_query ON q_learning_experiences(query);
	`

	if _, err := ql.db.Exec(createQTableSQL); err != nil {
		return fmt.Errorf("failed to create Q-table: %w", err)
	}

	if _, err := ql.db.Exec(createExperienceSQL); err != nil {
		return fmt.Errorf("failed to create experience table: %w", err)
	}

	return nil
}

// Learn performs Q-learning update based on experience
func (ql *CodeQLearning) Learn(ctx context.Context, query string, startNodeID string, userFeedback float64) error {
	// Navigate and collect experiences
	experiences, err := ql.collectExperiences(ctx, query, startNodeID)
	if err != nil {
		return fmt.Errorf("failed to collect experiences: %w", err)
	}

	// Update Q-values based on experiences
	for _, exp := range experiences {
		if err := ql.updateQValue(exp, userFeedback); err != nil {
			return fmt.Errorf("failed to update Q-value: %w", err)
		}

		// Store experience in database
		if err := ql.storeExperience(exp, userFeedback); err != nil {
			return fmt.Errorf("failed to store experience: %w", err)
		}
	}

	// Decay epsilon (reduce exploration over time)
	ql.epsilon = math.Max(ql.epsilon*ql.epsilonDecay, ql.minEpsilon)

	return nil
}

// GetBestAction returns the best action for a given state using epsilon-greedy policy
func (ql *CodeQLearning) GetBestAction(ctx context.Context, state CodeState) (*CodeAction, error) {
	stateHash := ql.stateEncoder.Encode(state)

	// Epsilon-greedy action selection
	if rand.Float64() < ql.epsilon {
		// Explore: random action
		action := ql.actionSpace[rand.Intn(len(ql.actionSpace))]
		targetNode := ql.getRandomTarget(state.CurrentNode)
		return &CodeAction{
			Type:       action,
			TargetNode: targetNode,
			Confidence: 0.0, // Low confidence for exploration
		}, nil
	}

	// Exploit: best known action
	bestAction, qValue, err := ql.getBestQValue(stateHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get best Q-value: %w", err)
	}

	if bestAction == "" {
		// No learned action, use heuristic
		return ql.getHeuristicAction(state), nil
	}

	targetNode := ql.selectTargetForAction(state.CurrentNode, bestAction)

	return &CodeAction{
		Type:       bestAction,
		TargetNode: targetNode,
		Confidence: math.Tanh(qValue), // Normalize confidence
	}, nil
}

// collectExperiences navigates the code and collects learning experiences
func (ql *CodeQLearning) collectExperiences(ctx context.Context, query string, startNodeID string) ([]Experience, error) {
	experiences := make([]Experience, 0)
	currentState := ql.createInitialState(query, startNodeID)

	maxSteps := 10
	for step := 0; step < maxSteps; step++ {
		// Get action using current policy
		action, err := ql.GetBestAction(ctx, currentState)
		if err != nil {
			return nil, err
		}

		// Execute action and get next state
		nextState, reward := ql.executeAction(currentState, action)

		// Create experience
		exp := Experience{
			State:     currentState,
			Action:    action.Type,
			Reward:    reward,
			NextState: nextState,
			Query:     query,
			Timestamp: time.Now(),
		}

		experiences = append(experiences, exp)

		// Move to next state
		currentState = nextState

		// Stop if we found highly relevant code
		if reward > 0.8 {
			break
		}
	}

	return experiences, nil
}

// updateQValue updates the Q-value using the Q-learning formula
func (ql *CodeQLearning) updateQValue(exp Experience, userFeedback float64) error {
	stateHash := ql.stateEncoder.Encode(exp.State)
	nextStateHash := ql.stateEncoder.Encode(exp.NextState)

	// Get current Q-value
	currentQ, err := ql.getQValue(stateHash, exp.Action)
	if err != nil {
		return err
	}

	// Get max Q-value for next state
	_, maxNextQ, err := ql.getBestQValue(nextStateHash)
	if err != nil {
		maxNextQ = 0.0 // Default if no Q-values exist
	}

	// Incorporate user feedback into reward
	adjustedReward := exp.Reward
	if userFeedback != 0 {
		adjustedReward = 0.7*exp.Reward + 0.3*userFeedback
	}

	// Q-learning update: Q(s,a) = Q(s,a) + α[r + γ*max(Q(s',a')) - Q(s,a)]
	newQ := currentQ + ql.learningRate*(adjustedReward+ql.discountFactor*maxNextQ-currentQ)

	// Store updated Q-value
	return ql.setQValue(stateHash, exp.Action, newQ, exp.State, adjustedReward, exp.Query)
}

// Database operations

func (ql *CodeQLearning) getQValue(stateHash, action string) (float64, error) {
	// Handle in-memory mode
	if ql.db == nil {
		return ql.getQValueInMemory(stateHash, action)
	}

	var qValue float64
	err := ql.db.QueryRow(
		"SELECT q_value FROM q_learning_table WHERE state_hash = ? AND action = ?",
		stateHash, action,
	).Scan(&qValue)

	if err == sql.ErrNoRows {
		return 0.0, nil // Default Q-value
	}

	return qValue, err
}

func (ql *CodeQLearning) setQValue(stateHash, action string, qValue float64, state CodeState, reward float64, query string) error {
	// Handle in-memory mode
	if ql.db == nil {
		return ql.setQValueInMemory(stateHash, action, qValue)
	}

	stateData, _ := json.Marshal(state)

	_, err := ql.db.Exec(`
		INSERT OR REPLACE INTO q_learning_table
		(state_hash, action, q_value, visits, last_updated, state_data, last_reward, query)
		VALUES (?, ?, ?,
			COALESCE((SELECT visits FROM q_learning_table WHERE state_hash = ? AND action = ?), 0) + 1,
			CURRENT_TIMESTAMP, ?, ?, ?)
	`, stateHash, action, qValue, stateHash, action, string(stateData), reward, query)

	return err
}

func (ql *CodeQLearning) getBestQValue(stateHash string) (string, float64, error) {
	// Handle in-memory mode
	if ql.db == nil {
		return ql.getBestQValueInMemory(stateHash)
	}

	var action string
	var qValue float64

	err := ql.db.QueryRow(`
		SELECT action, q_value FROM q_learning_table
		WHERE state_hash = ?
		ORDER BY q_value DESC
		LIMIT 1
	`, stateHash).Scan(&action, &qValue)

	if err == sql.ErrNoRows {
		return "", 0.0, nil
	}

	return action, qValue, err
}

func (ql *CodeQLearning) storeExperience(exp Experience, userFeedback float64) error {
	stateHash := ql.stateEncoder.Encode(exp.State)
	nextStateHash := ql.stateEncoder.Encode(exp.NextState)

	_, err := ql.db.Exec(`
		INSERT INTO q_learning_experiences 
		(state_hash, action, reward, next_state_hash, query, user_feedback, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, stateHash, exp.Action, exp.Reward, nextStateHash, exp.Query, userFeedback, exp.Timestamp)

	return err
}

// Helper methods

func (ql *CodeQLearning) createInitialState(query string, nodeID string) CodeState {
	return CodeState{
		CurrentNode:    nodeID,
		Query:          query,
		QueryTerms:     strings.Fields(strings.ToLower(query)),
		VisitedNodes:   []string{nodeID},
		NodeTypes:      make(map[string]int),
		PathDepth:      0,
		RelevanceScore: 0.0,
		Context:        make(map[string]float64),
	}
}

func (ql *CodeQLearning) executeAction(state CodeState, action *CodeAction) (CodeState, float64) {
	// Create next state
	nextState := state
	nextState.CurrentNode = action.TargetNode
	nextState.VisitedNodes = append(nextState.VisitedNodes, action.TargetNode)
	nextState.PathDepth++
	nextState.LastAction = action.Type

	// Calculate reward based on relevance
	if node, exists := ql.graph.GetNode(action.TargetNode); exists {
		relevance := ql.calculateRelevance(node, state.Query)
		nextState.RelevanceScore = relevance

		// Update node type counts
		nextState.NodeTypes[string(node.Type)]++

		return nextState, relevance
	}

	return nextState, 0.0
}

func (ql *CodeQLearning) calculateRelevance(node *graph.Node, query string) float64 {
	// Similar to MCTS relevance calculation
	score := 0.0
	queryLower := strings.ToLower(query)

	if strings.Contains(strings.ToLower(node.Name), queryLower) {
		score += 1.0
	}

	if strings.Contains(strings.ToLower(node.Path), queryLower) {
		score += 0.8
	}

	if strings.Contains(strings.ToLower(node.Purpose), queryLower) {
		score += 0.6
	}

	// Normalize to 0-1 range
	return math.Tanh(score)
}

func (ql *CodeQLearning) getRandomTarget(currentNodeID string) string {
	neighbors := ql.graph.GetNeighbors(currentNodeID)
	if len(neighbors) == 0 {
		return currentNodeID
	}
	return neighbors[rand.Intn(len(neighbors))].ID
}

func (ql *CodeQLearning) selectTargetForAction(currentNodeID, actionType string) string {
	// Select appropriate target based on action type
	neighbors := ql.graph.GetNeighbors(currentNodeID)
	if len(neighbors) == 0 {
		return currentNodeID
	}

	// Filter neighbors based on action type
	switch actionType {
	case "explore_function":
		// Prefer function nodes
		for _, neighbor := range neighbors {
			if neighbor.Type == "function" {
				return neighbor.ID
			}
		}
	case "explore_class":
		// Prefer class/struct nodes
		for _, neighbor := range neighbors {
			if neighbor.Type == "class" || neighbor.Type == "struct" {
				return neighbor.ID
			}
		}
	case "follow_import":
		// Prefer import/dependency nodes
		for _, neighbor := range neighbors {
			if neighbor.Type == "import" || neighbor.Type == "dependency" {
				return neighbor.ID
			}
		}
	}

	// Simple heuristic - can be enhanced
	return neighbors[0].ID
}

func (ql *CodeQLearning) getHeuristicAction(state CodeState) *CodeAction {
	// Fallback heuristic when no learned policy exists
	return &CodeAction{
		Type:       "explore_child",
		TargetNode: ql.getRandomTarget(state.CurrentNode),
		Confidence: 0.1,
	}
}

// NewStateEncoder creates a new state encoder
func NewStateEncoder() *StateEncoder {
	return &StateEncoder{
		featureWeights: map[string]float64{
			"current_node":     1.0,
			"query_similarity": 2.0,
			"path_depth":       0.5,
			"node_types":       0.8,
			"relevance_score":  1.5,
			"last_action":      0.3,
		},
	}
}

// Encode converts a CodeState into a consistent hash string
func (se *StateEncoder) Encode(state CodeState) string {
	// Create a normalized representation of the state
	features := make(map[string]any)

	// Current node (most important)
	features["current_node"] = state.CurrentNode

	// Query terms (sorted for consistency)
	sortedTerms := make([]string, len(state.QueryTerms))
	copy(sortedTerms, state.QueryTerms)
	sort.Strings(sortedTerms)
	features["query_terms"] = strings.Join(sortedTerms, ",")

	// Path depth (bucketed)
	features["depth_bucket"] = state.PathDepth / 3 // Group by depth ranges

	// Node type distribution (normalized)
	totalTypes := 0
	for _, count := range state.NodeTypes {
		totalTypes += count
	}

	typeDistribution := make(map[string]float64)
	for nodeType, count := range state.NodeTypes {
		if totalTypes > 0 {
			typeDistribution[nodeType] = float64(count) / float64(totalTypes)
		}
	}
	features["node_types"] = typeDistribution

	// Relevance score (bucketed)
	features["relevance_bucket"] = int(state.RelevanceScore * 10) // 0-10 buckets

	// Last action
	features["last_action"] = state.LastAction

	// Convert to JSON for consistent hashing
	jsonBytes, _ := json.Marshal(features)
	return fmt.Sprintf("%x", jsonBytes) // Use hex encoding of JSON
}

// GetQLearningStats returns statistics about the Q-learning performance
func (ql *CodeQLearning) GetQLearningStats(ctx context.Context) (*QLearningStats, error) {
	stats := &QLearningStats{}

	// Total Q-table entries
	err := ql.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM q_learning_table").Scan(&stats.TotalQEntries)
	if err != nil {
		return nil, err
	}

	// Total experiences
	err = ql.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM q_learning_experiences").Scan(&stats.TotalExperiences)
	if err != nil {
		return nil, err
	}

	// Average Q-value
	err = ql.db.QueryRowContext(ctx, "SELECT AVG(q_value) FROM q_learning_table").Scan(&stats.AverageQValue)
	if err != nil {
		return nil, err
	}

	// Most visited state-action pairs
	rows, err := ql.db.QueryContext(ctx, `
		SELECT action, COUNT(*) as count, AVG(q_value) as avg_q
		FROM q_learning_table
		GROUP BY action
		ORDER BY count DESC
		LIMIT 10
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats.TopActions = make([]ActionStats, 0)
	for rows.Next() {
		var actionStat ActionStats
		err := rows.Scan(&actionStat.Action, &actionStat.Count, &actionStat.AverageQValue)
		if err != nil {
			return nil, err
		}
		stats.TopActions = append(stats.TopActions, actionStat)
	}

	// Recent learning rate
	err = ql.db.QueryRowContext(ctx, `
		SELECT AVG(reward)
		FROM q_learning_experiences
		WHERE timestamp > datetime('now', '-1 day')
	`).Scan(&stats.RecentAverageReward)
	if err != nil {
		stats.RecentAverageReward = 0.0
	}

	stats.CurrentEpsilon = ql.epsilon
	stats.LearningRate = ql.learningRate

	return stats, nil
}

// QLearningStats represents Q-learning performance statistics
type QLearningStats struct {
	TotalQEntries       int           `json:"total_q_entries"`
	TotalExperiences    int           `json:"total_experiences"`
	AverageQValue       float64       `json:"average_q_value"`
	RecentAverageReward float64       `json:"recent_average_reward"`
	CurrentEpsilon      float64       `json:"current_epsilon"`
	LearningRate        float64       `json:"learning_rate"`
	TopActions          []ActionStats `json:"top_actions"`
}

// ActionStats represents statistics for a specific action
type ActionStats struct {
	Action        string  `json:"action"`
	Count         int     `json:"count"`
	AverageQValue float64 `json:"average_q_value"`
}

// Experience type is defined in types.go

// NewSimpleQLearning creates a simplified in-memory Q-learning system
func NewSimpleQLearning(codeGraph *graph.CodeGraph) *CodeQLearning {
	return &CodeQLearning{
		db:             nil, // In-memory mode
		graph:          codeGraph,
		learningRate:   0.1,
		discountFactor: 0.9,
		epsilon:        0.3,
		epsilonDecay:   0.995,
		minEpsilon:     0.05,
		stateEncoder:   &StateEncoder{},
		actionSpace:    []string{"navigate", "search", "explore"},
		qTable:         make(map[string]map[string]float64),
		mutex:          sync.RWMutex{},
	}
}

// NewCodeQLearningWithVectorDB creates Q-learning system using existing vectordb
func NewCodeQLearningWithVectorDB(vdb *vectordb.VectorDB, codeGraph *graph.CodeGraph) (*CodeQLearning, error) {
	// Create ML tables in the vectordb database
	err := createMLTables(vdb)
	if err != nil {
		log.Printf("Failed to create ML tables, falling back to in-memory: %v", err)
		// Fallback to in-memory version
		ql := NewSimpleQLearning(codeGraph)
		ql.qTable = make(map[string]map[string]float64)
		return ql, nil
	}

	// Create Q-learning with in-memory backend (for now)
	ql := &CodeQLearning{
		db:             nil, // Using in-memory storage for now
		graph:          codeGraph,
		learningRate:   0.1,
		discountFactor: 0.9,
		epsilon:        0.3,
		epsilonDecay:   0.995,
		minEpsilon:     0.05,
		stateEncoder:   &StateEncoder{},
		actionSpace:    []string{"navigate", "search", "explore"},
		qTable:         make(map[string]map[string]float64),
	}

	return ql, nil
}

// createMLTables creates the necessary ML tables in the database
func createMLTables(vdb *vectordb.VectorDB) error {
	// For now, we'll use in-memory storage since we don't want to modify the vectordb structure
	// In a future version, we could add ML-specific tables to the vectordb schema
	log.Printf("🧠 ML: Using in-memory storage for Q-learning tables (VectorDB available: %t)", vdb != nil)

	// Future enhancement: could use vdb.GetDB() to create ML tables
	// For now, gracefully fallback to in-memory storage
	return nil
}

// In-memory Q-learning helper methods

func (ql *CodeQLearning) getBestQValueInMemory(stateHash string) (string, float64, error) {
	ql.mutex.RLock()
	defer ql.mutex.RUnlock()

	stateActions, exists := ql.qTable[stateHash]
	if !exists || len(stateActions) == 0 {
		return "", 0.0, nil
	}

	bestAction := ""
	bestValue := -1000.0

	for action, qValue := range stateActions {
		if qValue > bestValue {
			bestValue = qValue
			bestAction = action
		}
	}

	return bestAction, bestValue, nil
}

func (ql *CodeQLearning) getQValueInMemory(stateHash, action string) (float64, error) {
	ql.mutex.RLock()
	defer ql.mutex.RUnlock()

	if stateActions, exists := ql.qTable[stateHash]; exists {
		if qValue, actionExists := stateActions[action]; actionExists {
			return qValue, nil
		}
	}

	return 0.0, nil // Default Q-value for unseen state-action pairs
}

func (ql *CodeQLearning) setQValueInMemory(stateHash, action string, qValue float64) error {
	ql.mutex.Lock()
	defer ql.mutex.Unlock()

	if ql.qTable[stateHash] == nil {
		ql.qTable[stateHash] = make(map[string]float64)
	}

	ql.qTable[stateHash][action] = qValue
	return nil
}
