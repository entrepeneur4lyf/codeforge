package ml

import (
	"strings"
	"time"
)

// Experience represents a learning experience for ML algorithms
type Experience struct {
	State     CodeState `json:"state"`
	Action    string    `json:"action"` // Action taken (node ID or action type)
	Reward    float64   `json:"reward"` // Quality/relevance score
	NextState CodeState `json:"next_state"`
	Query     string    `json:"query"` // Original search query
	Timestamp time.Time `json:"timestamp"`
}

// CodeState represents the current state in code exploration/navigation
type CodeState struct {
	CurrentNode    string             `json:"current_node"`
	Query          string             `json:"query"`
	QueryTerms     []string           `json:"query_terms"`
	VisitedNodes   []string           `json:"visited_nodes"`
	NodeTypes      map[string]int     `json:"node_types"` // Count of visited node types
	PathDepth      int                `json:"path_depth"`
	RelevanceScore float64            `json:"relevance_score"`
	LastAction     string             `json:"last_action"`
	Context        map[string]float64 `json:"context"` // Accumulated context relevance
}

// CodeAction represents possible navigation/exploration actions
type CodeAction struct {
	Type       string  `json:"type"`        // Action type (e.g., "follow_import", "explore_caller")
	TargetNode string  `json:"target_node"` // Target node ID
	Confidence float64 `json:"confidence"`  // Action confidence score
}

// SearchResult represents the result of an ML-based search
type SearchResult struct {
	BestPath    []string       `json:"best_path"`   // Sequence of node IDs
	Confidence  float64        `json:"confidence"`  // Overall confidence
	Relevance   float64        `json:"relevance"`   // Relevance score
	Explanation string         `json:"explanation"` // Human-readable explanation
	Metadata    map[string]any `json:"metadata"`    // Additional metadata
	Experiences []Experience   `json:"experiences"` // Learning experiences collected
}

// ValueNetwork represents a simple neural network for value estimation
type ValueNetwork struct {
	weights      map[string]float64 // Feature weights
	bias         float64            // Bias term
	learningRate float64            // Learning rate for updates
}

// NewValueNetwork creates a new value network
func NewValueNetwork() *ValueNetwork {
	return &ValueNetwork{
		weights: map[string]float64{
			"name_match":    2.0,
			"path_match":    1.5,
			"purpose_match": 1.0,
			"doc_match":     0.8,
			"type_bonus":    0.5,
			"visibility":    0.3,
			"depth_penalty": -0.1,
		},
		bias:         0.0,
		learningRate: 0.01,
	}
}

// Predict estimates the value of a state-query combination
func (vn *ValueNetwork) Predict(state CodeState, query string) float64 {
	// Extract features
	features := vn.extractFeatures(state, query)

	// Linear combination
	value := vn.bias
	for feature, weight := range vn.weights {
		if featureValue, exists := features[feature]; exists {
			value += weight * featureValue
		}
	}

	// Apply activation function (tanh for bounded output)
	return tanh(value)
}

// Train updates the network weights based on experience
func (vn *ValueNetwork) Train(experiences []Experience) {
	for _, exp := range experiences {
		// Get current prediction
		predicted := vn.Predict(exp.State, exp.Query)

		// Calculate error
		error := exp.Reward - predicted

		// Extract features
		features := vn.extractFeatures(exp.State, exp.Query)

		// Update weights using gradient descent
		for feature, featureValue := range features {
			if weight, exists := vn.weights[feature]; exists {
				vn.weights[feature] = weight + vn.learningRate*error*featureValue
			}
		}

		// Update bias
		vn.bias += vn.learningRate * error
	}
}

// extractFeatures extracts numerical features from state and query
func (vn *ValueNetwork) extractFeatures(state CodeState, query string) map[string]float64 {
	features := make(map[string]float64)

	// Query matching features using actual query
	queryLower := strings.ToLower(query)
	features["name_match"] = calculateStringMatch(strings.ToLower(state.CurrentNode), queryLower)
	features["query_match"] = calculateStringMatch(strings.ToLower(state.Query), queryLower)

	// Calculate match with query terms
	termMatch := 0.0
	for _, term := range state.QueryTerms {
		if strings.Contains(queryLower, strings.ToLower(term)) {
			termMatch += 1.0
		}
	}
	if len(state.QueryTerms) > 0 {
		features["term_match"] = termMatch / float64(len(state.QueryTerms))
	} else {
		features["term_match"] = 0.0
	}

	// State-based features
	features["depth_penalty"] = float64(state.PathDepth)
	features["relevance_score"] = state.RelevanceScore
	features["visited_count"] = float64(len(state.VisitedNodes))

	// Node type diversity
	features["type_diversity"] = float64(len(state.NodeTypes))

	return features
}

// Helper function for tanh activation
func tanh(x float64) float64 {
	if x > 20 {
		return 1.0
	}
	if x < -20 {
		return -1.0
	}

	exp2x := exp(2 * x)
	return (exp2x - 1) / (exp2x + 1)
}

// Helper function for exponential
func exp(x float64) float64 {
	// Simple approximation for small values
	if x < -10 {
		return 0.0
	}
	if x > 10 {
		return 22026.0 // e^10 approximately
	}

	// Taylor series approximation for moderate values
	result := 1.0
	term := 1.0
	for i := 1; i < 20; i++ {
		term *= x / float64(i)
		result += term
		if term < 1e-10 {
			break
		}
	}

	return result
}

// MLConfig represents configuration for ML algorithms
type MLConfig struct {
	// MCTS Configuration
	MCTSIterations   int     `json:"mcts_iterations"`
	MCTSTimeLimit    int     `json:"mcts_time_limit_ms"`
	ExplorationParam float64 `json:"exploration_param"`

	// Q-Learning Configuration
	LearningRate   float64 `json:"learning_rate"`
	DiscountFactor float64 `json:"discount_factor"`
	Epsilon        float64 `json:"epsilon"`
	EpsilonDecay   float64 `json:"epsilon_decay"`
	MinEpsilon     float64 `json:"min_epsilon"`

	// General Configuration
	MaxDepth       int  `json:"max_depth"`
	MaxExperiences int  `json:"max_experiences"`
	BatchSize      int  `json:"batch_size"`
	EnableLearning bool `json:"enable_learning"`
}

// DefaultMLConfig returns default ML configuration
func DefaultMLConfig() *MLConfig {
	return &MLConfig{
		MCTSIterations:   1000,
		MCTSTimeLimit:    5000,  // 5 seconds
		ExplorationParam: 1.414, // sqrt(2)

		LearningRate:   0.1,
		DiscountFactor: 0.9,
		Epsilon:        0.3,
		EpsilonDecay:   0.995,
		MinEpsilon:     0.05,

		MaxDepth:       10,
		MaxExperiences: 10000,
		BatchSize:      32,
		EnableLearning: true,
	}
}

// DefaultTDConfig is defined later in this file

// MLMetrics represents performance metrics for ML algorithms
type MLMetrics struct {
	// MCTS Metrics
	MCTSIterationsCompleted int     `json:"mcts_iterations_completed"`
	MCTSAverageReward       float64 `json:"mcts_average_reward"`
	MCTSBestReward          float64 `json:"mcts_best_reward"`

	// Q-Learning Metrics
	QLearningTotalEntries   int     `json:"qlearning_total_entries"`
	QLearningAverageQ       float64 `json:"qlearning_average_q"`
	QLearningCurrentEpsilon float64 `json:"qlearning_current_epsilon"`

	// General Metrics
	TotalExperiences int       `json:"total_experiences"`
	AverageReward    float64   `json:"average_reward"`
	LearningProgress float64   `json:"learning_progress"`
	LastUpdated      time.Time `json:"last_updated"`
}

// TDConfig represents configuration for TD learning
type TDConfig struct {
	Lambda         float64 `json:"lambda"`          // Eligibility trace decay (0.0-1.0)
	LearningRate   float64 `json:"learning_rate"`   // TD learning rate
	TraceThreshold float64 `json:"trace_threshold"` // Minimum trace value
	MaxTraces      int     `json:"max_traces"`      // Memory limit for traces
	UpdateFreq     int     `json:"update_freq"`     // Batch update frequency
}

// DefaultTDConfig returns default TD learning configuration
func DefaultTDConfig() *TDConfig {
	return &TDConfig{
		Lambda:         0.9,  // High eligibility trace decay
		LearningRate:   0.1,  // Same as Q-learning
		TraceThreshold: 0.01, // Remove very small traces
		MaxTraces:      1000, // Reasonable memory limit
		UpdateFreq:     10,   // Batch updates every 10 steps
	}
}

// calculateStringMatch calculates similarity between two strings
func calculateStringMatch(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}
	if s1 == "" || s2 == "" {
		return 0.0
	}

	// Simple substring matching
	if strings.Contains(s1, s2) || strings.Contains(s2, s1) {
		return 0.8
	}

	// Check for common words
	words1 := strings.Fields(s1)
	words2 := strings.Fields(s2)
	commonWords := 0

	for _, w1 := range words1 {
		for _, w2 := range words2 {
			if w1 == w2 {
				commonWords++
				break
			}
		}
	}

	if len(words1) > 0 && len(words2) > 0 {
		maxLen := len(words1)
		if len(words2) > maxLen {
			maxLen = len(words2)
		}
		return float64(commonWords) / float64(maxLen)
	}

	return 0.0
}
