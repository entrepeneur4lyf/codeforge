package ml

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/entrepeneur4lyf/codeforge/internal/graph"
	"github.com/entrepeneur4lyf/codeforge/internal/vectordb"
)

// CodeIntelligence combines MCTS, Q-Learning, and TD Learning for intelligent code analysis
type CodeIntelligence struct {
	// Core components
	graph      *graph.CodeGraph
	mcts       *CodeMCTS
	qlearning  *CodeQLearning
	tdlearning *TDLearning

	// Configuration
	config   *MLConfig
	tdConfig *TDConfig

	// State
	mutex   sync.RWMutex
	enabled bool
	useTD   bool // Whether to use TD learning
	metrics *MLMetrics

	// Learning data
	experiences    []Experience
	maxExperiences int
}

// NewCodeIntelligence creates a new ML-powered code intelligence system
func NewCodeIntelligence(codeGraph *graph.CodeGraph, vdb *vectordb.VectorDB) (*CodeIntelligence, error) {
	// Create MCTS component
	mcts := NewCodeMCTS(codeGraph)

	// Create Q-Learning component using vectordb
	qlearning, err := NewCodeQLearningWithVectorDB(vdb, codeGraph)
	if err != nil {
		return nil, fmt.Errorf("failed to create Q-learning: %w", err)
	}

	// Create TD Learning component using vectordb
	tdlearning, err := NewTDLearningWithVectorDB(vdb, codeGraph)
	if err != nil {
		return nil, fmt.Errorf("failed to create TD learning: %w", err)
	}

	ci := &CodeIntelligence{
		graph:          codeGraph,
		mcts:           mcts,
		qlearning:      qlearning,
		tdlearning:     tdlearning,
		config:         DefaultMLConfig(),
		tdConfig:       DefaultTDConfig(),
		enabled:        true,
		useTD:          true, // Enable TD learning by default
		metrics:        &MLMetrics{LastUpdated: time.Now()},
		experiences:    make([]Experience, 0),
		maxExperiences: 10000,
	}

	return ci, nil
}

// SmartSearch performs intelligent code search using ML algorithms
func (ci *CodeIntelligence) SmartSearch(ctx context.Context, query string, startNodeID string) (*SearchResult, error) {
	ci.mutex.RLock()
	defer ci.mutex.RUnlock()

	if !ci.enabled {
		return ci.fallbackSearch(query, startNodeID), nil
	}

	// Use TD Learning if enabled (faster and more efficient)
	if ci.useTD {
		return ci.tdlearning.SearchWithTDLearning(ctx, query, startNodeID)
	}

	// Fallback to MCTS + Q-Learning combination
	log.Printf("🧠 Using MCTS + Q-Learning for search: %s", query)

	// Use both MCTS and Q-Learning for comprehensive search
	mctsResult, err := ci.mcts.Search(ctx, query, startNodeID)
	if err != nil {
		log.Printf("MCTS search failed: %v", err)
		return ci.fallbackSearch(query, startNodeID), nil
	}

	// Get Q-Learning recommendations
	qState := ci.createStateFromNode(query, startNodeID)
	qAction, err := ci.qlearning.GetBestAction(ctx, qState)
	if err != nil {
		log.Printf("Q-Learning action failed: %v", err)
	}

	// Combine results intelligently
	result := ci.combineResults(mctsResult, qAction, query)

	// Store experiences for learning
	if ci.config.EnableLearning {
		ci.storeExperiences(result.Experiences)
	}

	return result, nil
}

// LearnFromFeedback updates the ML models based on user feedback
func (ci *CodeIntelligence) LearnFromFeedback(ctx context.Context, query string, selectedPath []string, userFeedback float64) error {
	ci.mutex.Lock()
	defer ci.mutex.Unlock()

	if !ci.enabled || !ci.config.EnableLearning {
		return nil
	}

	// Use TD Learning if enabled (more efficient learning)
	if ci.useTD {
		err := ci.tdlearning.LearnFromFeedbackTD(ctx, query, selectedPath, userFeedback)
		if err != nil {
			log.Printf("TD Learning update failed: %v", err)
		} else {
			log.Printf("🧠 TD Learning updated with user feedback: %.2f", userFeedback)
		}
	} else {
		// Fallback to Q-Learning
		if len(selectedPath) > 0 {
			err := ci.qlearning.Learn(ctx, query, selectedPath[0], userFeedback)
			if err != nil {
				log.Printf("Q-Learning update failed: %v", err)
			}
		}

		// Update MCTS value network
		if len(ci.experiences) > 0 {
			// Adjust experience rewards based on user feedback
			adjustedExperiences := ci.adjustExperienceRewards(ci.experiences, userFeedback)
			ci.mcts.valueNetwork.Train(adjustedExperiences)
		}

		log.Printf("🧠 Q-Learning + MCTS updated with user feedback: %.2f", userFeedback)
	}

	// Update metrics
	ci.updateMetrics()

	return nil
}

// GetIntelligentContext provides ML-enhanced context for LLMs
func (ci *CodeIntelligence) GetIntelligentContext(ctx context.Context, query string, maxNodes int) (string, error) {
	// Find best starting points using ML
	startNodes := ci.findBestStartingNodes(query, 3)

	var allResults []*SearchResult

	// Search from multiple starting points
	for _, startNode := range startNodes {
		result, err := ci.SmartSearch(ctx, query, startNode)
		if err != nil {
			continue
		}
		allResults = append(allResults, result)
	}

	// Generate intelligent context
	return ci.generateIntelligentContext(allResults, query, maxNodes), nil
}

// GetMLStats returns current ML performance statistics
func (ci *CodeIntelligence) GetMLStats(ctx context.Context) (*MLMetrics, error) {
	ci.mutex.RLock()
	defer ci.mutex.RUnlock()

	if ci.useTD {
		// Get TD Learning stats
		tdStats := ci.tdlearning.GetTDStats()

		// Update metrics with TD stats
		ci.metrics.QLearningTotalEntries = tdStats.TotalSteps
		ci.metrics.QLearningAverageQ = tdStats.AverageTDError
		ci.metrics.QLearningCurrentEpsilon = tdStats.Lambda
		ci.metrics.TotalExperiences = tdStats.ActiveTraces
		ci.metrics.AverageReward = 1.0 - tdStats.AverageTDError // Lower TD error = better performance
		ci.metrics.LastUpdated = time.Now()

		log.Printf("📊 TD Learning Stats - Steps: %d, Avg TD Error: %.3f, Active Traces: %d",
			tdStats.TotalSteps, tdStats.AverageTDError, tdStats.ActiveTraces)
	} else {
		// Get Q-Learning stats
		qStats, err := ci.qlearning.GetQLearningStats(ctx)
		if err != nil {
			return nil, err
		}

		// Update metrics
		ci.metrics.QLearningTotalEntries = qStats.TotalQEntries
		ci.metrics.QLearningAverageQ = qStats.AverageQValue
		ci.metrics.QLearningCurrentEpsilon = qStats.CurrentEpsilon
		ci.metrics.TotalExperiences = len(ci.experiences)
		ci.metrics.LastUpdated = time.Now()

		// Calculate learning progress (simplified)
		if len(ci.experiences) > 0 {
			recentReward := 0.0
			recentCount := 0
			cutoff := time.Now().Add(-24 * time.Hour)

			for _, exp := range ci.experiences {
				if exp.Timestamp.After(cutoff) {
					recentReward += exp.Reward
					recentCount++
				}
			}

			if recentCount > 0 {
				ci.metrics.AverageReward = recentReward / float64(recentCount)
			}
		}
	}

	return ci.metrics, nil
}

// Enable/Disable ML functionality
func (ci *CodeIntelligence) SetEnabled(enabled bool) {
	ci.mutex.Lock()
	defer ci.mutex.Unlock()
	ci.enabled = enabled

	if enabled {
		log.Println("🧠 ML code intelligence enabled")
	} else {
		log.Println("🧠 ML code intelligence disabled")
	}
}

// UpdateConfig updates ML configuration
func (ci *CodeIntelligence) UpdateConfig(config *MLConfig) {
	ci.mutex.Lock()
	defer ci.mutex.Unlock()
	ci.config = config

	// Update component configurations
	ci.mcts.maxIterations = config.MCTSIterations
	ci.mcts.timeLimit = time.Duration(config.MCTSTimeLimit) * time.Millisecond
	ci.mcts.explorationParam = config.ExplorationParam

	ci.qlearning.learningRate = config.LearningRate
	ci.qlearning.discountFactor = config.DiscountFactor
	ci.qlearning.epsilon = config.Epsilon
	ci.qlearning.epsilonDecay = config.EpsilonDecay
	ci.qlearning.minEpsilon = config.MinEpsilon

	// Configuration updated silently
}

// UpdateTDConfig updates TD learning configuration
func (ci *CodeIntelligence) UpdateTDConfig(config *TDConfig) {
	ci.mutex.Lock()
	defer ci.mutex.Unlock()
	ci.tdConfig = config

	// Update TD learning configuration
	ci.tdlearning.UpdateTDConfig(config)

	// TD Learning configuration updated silently
}

// SetUseTD enables or disables TD learning
func (ci *CodeIntelligence) SetUseTD(useTD bool) {
	ci.mutex.Lock()
	defer ci.mutex.Unlock()
	ci.useTD = useTD

	// TD Learning mode set silently
}

// GetTDStats returns TD learning specific statistics
func (ci *CodeIntelligence) GetTDStats() *TDStats {
	ci.mutex.RLock()
	defer ci.mutex.RUnlock()

	if ci.useTD {
		return ci.tdlearning.GetTDStats()
	}

	return nil
}

// Helper methods

func (ci *CodeIntelligence) fallbackSearch(query string, startNodeID string) *SearchResult {
	// Simple fallback when ML is disabled or fails
	return &SearchResult{
		BestPath:    []string{startNodeID},
		Confidence:  0.1,
		Relevance:   0.1,
		Explanation: fmt.Sprintf("Fallback search for '%s' (ML disabled)", query),
		Metadata:    map[string]interface{}{"method": "fallback", "query": query},
		Experiences: []Experience{},
	}
}

func (ci *CodeIntelligence) createStateFromNode(query string, nodeID string) CodeState {
	return CodeState{
		CurrentNode:    nodeID,
		Query:          query,
		QueryTerms:     []string{}, // Would be populated properly
		VisitedNodes:   []string{nodeID},
		NodeTypes:      make(map[string]int),
		PathDepth:      0,
		RelevanceScore: 0.0,
		LastAction:     "",
		Context:        make(map[string]float64),
	}
}

func (ci *CodeIntelligence) combineResults(mctsResult *SearchResult, qAction *CodeAction, query string) *SearchResult {
	// Intelligent combination of MCTS and Q-Learning results
	result := mctsResult

	// Add query context to metadata
	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}
	result.Metadata["original_query"] = query

	if qAction != nil && qAction.Confidence > 0.5 {
		// Q-Learning suggests a high-confidence action
		if len(result.BestPath) > 0 {
			// Append Q-Learning suggestion to MCTS path
			result.BestPath = append(result.BestPath, qAction.TargetNode)
		}

		// Boost confidence if both algorithms agree
		result.Confidence = (result.Confidence + qAction.Confidence) / 2.0
		result.Explanation += fmt.Sprintf(" Enhanced with Q-Learning for '%s' (confidence: %.2f)", query, qAction.Confidence)
	}

	return result
}

func (ci *CodeIntelligence) storeExperiences(experiences []Experience) {
	ci.experiences = append(ci.experiences, experiences...)

	// Limit experience buffer size
	if len(ci.experiences) > ci.maxExperiences {
		// Keep most recent experiences
		ci.experiences = ci.experiences[len(ci.experiences)-ci.maxExperiences:]
	}
}

func (ci *CodeIntelligence) adjustExperienceRewards(experiences []Experience, userFeedback float64) []Experience {
	adjusted := make([]Experience, len(experiences))
	copy(adjusted, experiences)

	// Adjust rewards based on user feedback
	for i := range adjusted {
		adjusted[i].Reward = 0.7*adjusted[i].Reward + 0.3*userFeedback
	}

	return adjusted
}

func (ci *CodeIntelligence) findBestStartingNodes(query string, maxNodes int) []string {
	// Use simple heuristics to find good starting points
	// This could be enhanced with ML in the future

	nodes := []string{}

	// Find nodes that match the query
	for _, node := range ci.graph.GetAllNodes() {
		if len(nodes) >= maxNodes {
			break
		}

		if ci.nodeMatchesQuery(node, query) {
			nodes = append(nodes, node.ID)
		}
	}

	// If no matches, use some default starting points
	if len(nodes) == 0 {
		allNodes := ci.graph.GetAllNodes()
		if len(allNodes) > 0 {
			nodes = append(nodes, allNodes[0].ID)
		}
	}

	return nodes
}

func (ci *CodeIntelligence) nodeMatchesQuery(node *graph.Node, query string) bool {
	// Simple matching logic
	queryLower := strings.ToLower(query)
	return strings.Contains(strings.ToLower(node.Name), queryLower) ||
		strings.Contains(strings.ToLower(node.Path), queryLower) ||
		strings.Contains(strings.ToLower(node.Purpose), queryLower)
}

func (ci *CodeIntelligence) generateIntelligentContext(results []*SearchResult, query string, maxNodes int) string {
	// Generate intelligent context from ML results
	context := "# 🧠 ML-Enhanced Code Context\n\n"
	context += fmt.Sprintf("**Query:** %s\n", query)
	context += "**Analysis Method:** MCTS + Q-Learning\n\n"

	if len(results) == 0 {
		context += "No relevant code found.\n"
		return context
	}

	context += "## Most Relevant Code Paths\n\n"

	nodeCount := 0
	for i, result := range results {
		if nodeCount >= maxNodes {
			break
		}

		context += fmt.Sprintf("### Path %d (Confidence: %.2f)\n", i+1, result.Confidence)

		for j, nodeID := range result.BestPath {
			if nodeCount >= maxNodes {
				break
			}

			if node, exists := ci.graph.GetNode(nodeID); exists {
				context += fmt.Sprintf("%d. `%s` - %s\n", j+1, node.Path, node.Purpose)
				nodeCount++
			}
		}

		if result.Explanation != "" {
			context += fmt.Sprintf("**Reasoning:** %s\n", result.Explanation)
		}

		context += "\n"
	}

	return context
}

func (ci *CodeIntelligence) updateMetrics() {
	// Update internal metrics
	ci.metrics.LastUpdated = time.Now()
	ci.metrics.TotalExperiences = len(ci.experiences)

	// Calculate average reward from recent experiences
	if len(ci.experiences) > 0 {
		totalReward := 0.0
		for _, exp := range ci.experiences {
			totalReward += exp.Reward
		}
		ci.metrics.AverageReward = totalReward / float64(len(ci.experiences))
	}
}
