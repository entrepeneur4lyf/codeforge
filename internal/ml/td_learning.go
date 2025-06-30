package ml

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/graph"
	"github.com/entrepeneur4lyf/codeforge/internal/vectordb"
)

// TDLearning implements Temporal Difference learning for code intelligence
type TDLearning struct {
	qlearning         *CodeQLearning
	lambda            float64            // TD(λ) eligibility trace decay
	eligibilityTraces map[string]float64 // State-action eligibility traces
	traceThreshold    float64            // Minimum trace value to keep
	maxTraces         int                // Maximum number of traces to maintain

	// Performance tracking
	stepCount      int
	totalTDError   float64
	lastUpdateTime time.Time
}

// TDConfig is defined in types.go

// NewTDLearning creates a new TD learning system
func NewTDLearning(db *sql.DB, codeGraph *graph.CodeGraph) (*TDLearning, error) {
	qlearning, err := NewCodeQLearning(db, codeGraph)
	if err != nil {
		return nil, fmt.Errorf("failed to create Q-learning: %w", err)
	}

	td := &TDLearning{
		qlearning:         qlearning,
		lambda:            0.9, // High trace decay for good credit assignment
		eligibilityTraces: make(map[string]float64),
		traceThreshold:    0.01, // Remove very small traces
		maxTraces:         1000, // Reasonable memory limit
		lastUpdateTime:    time.Now(),
	}

	return td, nil
}

// SearchWithTDLearning performs intelligent search using TD learning
func (td *TDLearning) SearchWithTDLearning(ctx context.Context, query string, startNodeID string) (*SearchResult, error) {
	// Reset eligibility traces for new episode
	td.eligibilityTraces = make(map[string]float64)

	currentState := td.createInitialState(query, startNodeID)
	path := []string{startNodeID}
	totalReward := 0.0
	stepRewards := make([]float64, 0)

	maxSteps := 10
	for step := 0; step < maxSteps; step++ {
		// Get action using epsilon-greedy policy
		action, err := td.qlearning.GetBestAction(ctx, currentState)
		if err != nil {
			log.Printf("TD Learning: Failed to get action: %v", err)
			break
		}

		// Execute action and get next state
		nextState, reward := td.executeAction(currentState, action)
		stepRewards = append(stepRewards, reward)

		// TD Update - Key improvement over standard Q-learning
		tdError := td.updateWithTD(currentState, action.Type, reward, nextState)

		// Track performance metrics
		td.stepCount++
		td.totalTDError += abs(tdError)

		// Update path and state
		path = append(path, action.TargetNode)
		currentState = nextState
		totalReward += reward

		// Early termination if high reward found
		if reward > 0.8 {
			log.Printf("TD Learning: Early termination with high reward: %.3f", reward)
			break
		}

		// Prevent infinite loops
		if td.hasVisitedRecently(currentState.CurrentNode, path) {
			break
		}
	}

	// Calculate confidence based on reward trajectory
	confidence := td.calculateConfidence(stepRewards, totalReward)

	return &SearchResult{
		BestPath:   path,
		Confidence: confidence,
		Relevance:  totalReward / float64(len(path)),
		Explanation: fmt.Sprintf("TD(λ=%.1f) learning search with %d steps, avg TD error: %.3f",
			td.lambda, len(path), td.totalTDError/float64(td.stepCount)),
		Metadata: map[string]interface{}{
			"algorithm":    "TD-Learning",
			"lambda":       td.lambda,
			"steps":        len(path),
			"total_reward": totalReward,
			"avg_td_error": td.totalTDError / float64(td.stepCount),
			"trace_count":  len(td.eligibilityTraces),
		},
		Experiences: []Experience{}, // TD learning doesn't need to store full episodes
	}, nil
}

// updateWithTD performs TD(λ) update with eligibility traces
func (td *TDLearning) updateWithTD(state CodeState, action string, reward float64, nextState CodeState) float64 {
	stateHash := td.qlearning.stateEncoder.Encode(state)
	nextStateHash := td.qlearning.stateEncoder.Encode(nextState)

	// Get current Q-values
	currentQ, _ := td.qlearning.getQValue(stateHash, action)
	_, nextMaxQ, _ := td.qlearning.getBestQValue(nextStateHash)

	// Calculate TD error: δ = r + γ*V(s') - V(s)
	tdError := reward + td.qlearning.discountFactor*nextMaxQ - currentQ

	// Update eligibility trace for current state-action pair
	traceKey := td.getTraceKey(stateHash, action)
	td.eligibilityTraces[traceKey] = 1.0

	// Update all state-action pairs with eligibility traces
	updatedCount := 0
	for key, trace := range td.eligibilityTraces {
		if trace < td.traceThreshold {
			delete(td.eligibilityTraces, key)
			continue
		}

		// Parse state-action key
		parts := strings.Split(key, ":")
		if len(parts) != 2 {
			continue
		}
		keyStateHash, keyAction := parts[0], parts[1]

		// Get current Q-value
		oldQ, _ := td.qlearning.getQValue(keyStateHash, keyAction)

		// TD(λ) update: Q(s,a) = Q(s,a) + α * δ * e(s,a)
		newQ := oldQ + td.qlearning.learningRate*tdError*trace

		// Store updated Q-value (create dummy state for interface compatibility)
		dummyState := CodeState{CurrentNode: keyStateHash}
		err := td.qlearning.setQValue(keyStateHash, keyAction, newQ, dummyState, reward, state.Query)
		if err != nil {
			log.Printf("TD Learning: Failed to update Q-value: %v", err)
		}

		updatedCount++

		// Decay eligibility trace: e(s,a) = γ * λ * e(s,a)
		td.eligibilityTraces[key] *= td.qlearning.discountFactor * td.lambda
	}

	// Memory management: limit number of traces
	if len(td.eligibilityTraces) > td.maxTraces {
		td.pruneTraces()
	}

	return tdError
}

// LearnFromFeedbackTD updates the model using TD learning with user feedback
func (td *TDLearning) LearnFromFeedbackTD(ctx context.Context, query string, selectedPath []string, userFeedback float64) error {
	if len(selectedPath) == 0 {
		return nil
	}

	// Convert path to state-action sequence
	for i := 0; i < len(selectedPath)-1; i++ {
		currentState := CodeState{
			CurrentNode: selectedPath[i],
			Query:       query,
			QueryTerms:  strings.Fields(strings.ToLower(query)),
			PathDepth:   i,
		}

		nextState := CodeState{
			CurrentNode: selectedPath[i+1],
			Query:       query,
			QueryTerms:  strings.Fields(strings.ToLower(query)),
			PathDepth:   i + 1,
		}

		// Use user feedback as reward signal
		reward := userFeedback * (1.0 - 0.1*float64(i)) // Decay reward over path length

		// Perform TD update
		td.updateWithTD(currentState, "user_selected", reward, nextState)
	}

	log.Printf("TD Learning: Updated from user feedback %.2f on path length %d", userFeedback, len(selectedPath))
	return nil
}

// GetTDStats returns TD learning performance statistics
func (td *TDLearning) GetTDStats() *TDStats {
	avgTDError := 0.0
	if td.stepCount > 0 {
		avgTDError = td.totalTDError / float64(td.stepCount)
	}

	return &TDStats{
		Lambda:         td.lambda,
		TotalSteps:     td.stepCount,
		AverageTDError: avgTDError,
		ActiveTraces:   len(td.eligibilityTraces),
		MaxTraces:      td.maxTraces,
		TraceThreshold: td.traceThreshold,
		LastUpdateTime: td.lastUpdateTime,
		LearningRate:   td.qlearning.learningRate,
		DiscountFactor: td.qlearning.discountFactor,
	}
}

// UpdateTDConfig updates TD learning configuration
func (td *TDLearning) UpdateTDConfig(config *TDConfig) {
	td.lambda = config.Lambda
	td.traceThreshold = config.TraceThreshold
	td.maxTraces = config.MaxTraces
	td.qlearning.learningRate = config.LearningRate

	log.Printf("TD Learning: Updated config - λ=%.2f, threshold=%.3f, max_traces=%d",
		td.lambda, td.traceThreshold, td.maxTraces)
}

// Helper methods

func (td *TDLearning) createInitialState(query string, nodeID string) CodeState {
	return CodeState{
		CurrentNode:    nodeID,
		Query:          query,
		QueryTerms:     strings.Fields(strings.ToLower(query)),
		VisitedNodes:   []string{nodeID},
		NodeTypes:      make(map[string]int),
		PathDepth:      0,
		RelevanceScore: 0.0,
		LastAction:     "",
		Context:        make(map[string]float64),
	}
}

func (td *TDLearning) executeAction(state CodeState, action *CodeAction) (CodeState, float64) {
	// Create next state
	nextState := state
	nextState.CurrentNode = action.TargetNode
	nextState.VisitedNodes = append(nextState.VisitedNodes, action.TargetNode)
	nextState.PathDepth++
	nextState.LastAction = action.Type

	// Calculate reward based on relevance
	if node, exists := td.qlearning.graph.GetNode(action.TargetNode); exists {
		relevance := td.calculateRelevance(node, state.Query)
		nextState.RelevanceScore = relevance

		// Update node type counts
		nextState.NodeTypes[string(node.Type)]++

		return nextState, relevance
	}

	return nextState, 0.0
}

func (td *TDLearning) calculateRelevance(node *graph.Node, query string) float64 {
	score := 0.0
	queryLower := strings.ToLower(query)

	// Name matching (highest weight)
	if strings.Contains(strings.ToLower(node.Name), queryLower) {
		score += 1.0
	}

	// Path matching
	if strings.Contains(strings.ToLower(node.Path), queryLower) {
		score += 0.8
	}

	// Purpose/documentation matching
	if strings.Contains(strings.ToLower(node.Purpose), queryLower) {
		score += 0.6
	}

	if strings.Contains(strings.ToLower(node.DocComment), queryLower) {
		score += 0.4
	}

	// Type-based bonuses
	switch node.Type {
	case graph.NodeTypeFunction:
		if node.Visibility == "public" {
			score += 0.3
		}
	case graph.NodeTypeStruct, graph.NodeTypeInterface:
		score += 0.2
	}

	// Normalize to 0-1 range using tanh
	return tanh(score)
}

func (td *TDLearning) getTraceKey(stateHash, action string) string {
	return fmt.Sprintf("%s:%s", stateHash, action)
}

func (td *TDLearning) pruneTraces() {
	// Remove traces below threshold
	for key, trace := range td.eligibilityTraces {
		if trace < td.traceThreshold {
			delete(td.eligibilityTraces, key)
		}
	}

	// If still too many, remove oldest/smallest traces
	if len(td.eligibilityTraces) > td.maxTraces {
		// Convert to slice for sorting
		type traceEntry struct {
			key   string
			value float64
		}

		traces := make([]traceEntry, 0, len(td.eligibilityTraces))
		for k, v := range td.eligibilityTraces {
			traces = append(traces, traceEntry{k, v})
		}

		// Sort by trace value (descending)
		for i := 0; i < len(traces)-1; i++ {
			for j := i + 1; j < len(traces); j++ {
				if traces[i].value < traces[j].value {
					traces[i], traces[j] = traces[j], traces[i]
				}
			}
		}

		// Keep only top traces
		td.eligibilityTraces = make(map[string]float64)
		for i := 0; i < td.maxTraces && i < len(traces); i++ {
			td.eligibilityTraces[traces[i].key] = traces[i].value
		}
	}
}

func (td *TDLearning) hasVisitedRecently(nodeID string, path []string) bool {
	count := 0
	for _, pathNode := range path {
		if pathNode == nodeID {
			count++
			if count > 2 { // Allow some revisiting but prevent loops
				return true
			}
		}
	}
	return false
}

func (td *TDLearning) calculateConfidence(stepRewards []float64, totalReward float64) float64 {
	if len(stepRewards) == 0 {
		return 0.0
	}

	// Base confidence from average reward
	avgReward := totalReward / float64(len(stepRewards))

	// Bonus for improving trajectory
	improvementBonus := 0.0
	if len(stepRewards) > 1 {
		lastReward := stepRewards[len(stepRewards)-1]
		firstReward := stepRewards[0]
		if lastReward > firstReward {
			improvementBonus = 0.2
		}
	}

	// Penalty for very long paths
	lengthPenalty := 0.0
	if len(stepRewards) > 7 {
		lengthPenalty = 0.1 * float64(len(stepRewards)-7)
	}

	confidence := avgReward + improvementBonus - lengthPenalty
	return max(0.0, min(1.0, confidence))
}

// TDStats represents TD learning statistics
type TDStats struct {
	Lambda         float64   `json:"lambda"`
	TotalSteps     int       `json:"total_steps"`
	AverageTDError float64   `json:"average_td_error"`
	ActiveTraces   int       `json:"active_traces"`
	MaxTraces      int       `json:"max_traces"`
	TraceThreshold float64   `json:"trace_threshold"`
	LastUpdateTime time.Time `json:"last_update_time"`
	LearningRate   float64   `json:"learning_rate"`
	DiscountFactor float64   `json:"discount_factor"`
}

// Helper functions
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// NewTDLearningWithVectorDB creates TD learning system using existing vectordb
func NewTDLearningWithVectorDB(vdb *vectordb.VectorDB, codeGraph *graph.CodeGraph) (*TDLearning, error) {
	// Create simple Q-learning first
	qlearning := NewSimpleQLearning(codeGraph)

	td := &TDLearning{
		qlearning:         qlearning,
		lambda:            0.9, // High trace decay for good credit assignment
		eligibilityTraces: make(map[string]float64),
		traceThreshold:    0.01, // Remove very small traces
		maxTraces:         1000, // Reasonable memory limit
		lastUpdateTime:    time.Now(),
	}

	return td, nil
}
