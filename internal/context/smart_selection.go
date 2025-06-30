package context

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/entrepeneur4lyf/codeforge/internal/config"
)

// SmartSelector handles intelligent context selection based on query analysis
type SmartSelector struct {
	config            *config.Config
	tokenCounter      *TokenCounter
	relevanceScorer   *RelevanceScorer
	dependencyTracker *DependencyTracker
}

// NewSmartSelector creates a new smart context selector
func NewSmartSelector(cfg *config.Config) *SmartSelector {
	return &SmartSelector{
		config:            cfg,
		tokenCounter:      NewTokenCounter(),
		relevanceScorer:   NewRelevanceScorer(),
		dependencyTracker: NewDependencyTracker(),
	}
}

// SelectionStrategy defines how context should be selected
type SelectionStrategy struct {
	MaxTokens           int     `json:"max_tokens"`
	RelevanceThreshold  float64 `json:"relevance_threshold"`
	IncludeDependencies bool    `json:"include_dependencies"`
	PreserveOrder       bool    `json:"preserve_order"`
	IncludeRecent       bool    `json:"include_recent"`
	RecentCount         int     `json:"recent_count"`
	BalanceRoles        bool    `json:"balance_roles"`
	PrioritizeCode      bool    `json:"prioritize_code"`
}

// SmartSelectionResult represents the result of smart context selection
type SmartSelectionResult struct {
	SelectedMessages   []ConversationMessage `json:"selected_messages"`
	OriginalCount      int                   `json:"original_count"`
	SelectedCount      int                   `json:"selected_count"`
	TotalTokens        int                   `json:"total_tokens"`
	Strategy           SelectionStrategy     `json:"strategy"`
	RelevanceScores    []RelevanceScore      `json:"relevance_scores"`
	DependencyGraph    *DependencyGraph      `json:"dependency_graph,omitempty"`
	SelectionReasoning string                `json:"selection_reasoning"`
	CoverageScore      float64               `json:"coverage_score"`
}

// SelectSmartContext intelligently selects context based on query analysis
func (ss *SmartSelector) SelectSmartContext(messages []ConversationMessage, query string, modelID string) (*SmartSelectionResult, error) {
	// Analyze query to determine optimal strategy
	strategy := ss.analyzeQueryAndDetermineStrategy(query, modelID)

	return ss.SelectWithStrategy(messages, query, modelID, strategy)
}

// SelectWithStrategy selects context using a specific strategy
func (ss *SmartSelector) SelectWithStrategy(messages []ConversationMessage, query string, modelID string, strategy SelectionStrategy) (*SmartSelectionResult, error) {
	log.Printf("Smart context selection: %d messages, query: '%.50s...', strategy: max_tokens=%d",
		len(messages), query, strategy.MaxTokens)

	// Step 1: Score relevance
	relevanceResult, err := ss.relevanceScorer.ScoreRelevance(messages, query, strategy.RelevanceThreshold)
	if err != nil {
		return nil, fmt.Errorf("relevance scoring failed: %w", err)
	}

	// Step 2: Build dependency graph if needed
	var dependencyGraph *DependencyGraph
	if strategy.IncludeDependencies {
		dependencyGraph = ss.dependencyTracker.BuildDependencyGraph(messages)
	}

	// Step 3: Select messages based on strategy
	selectedMessages := ss.selectMessagesByStrategy(messages, relevanceResult, dependencyGraph, strategy, modelID)

	// Step 4: Calculate final metrics
	totalTokens := ss.tokenCounter.CountConversationTokens(selectedMessages, modelID).TotalTokens
	coverageScore := ss.calculateCoverageScore(selectedMessages, messages, relevanceResult)
	reasoning := ss.generateSelectionReasoning(strategy, relevanceResult, len(selectedMessages), totalTokens)

	return &SmartSelectionResult{
		SelectedMessages:   selectedMessages,
		OriginalCount:      len(messages),
		SelectedCount:      len(selectedMessages),
		TotalTokens:        totalTokens,
		Strategy:           strategy,
		RelevanceScores:    relevanceResult.Scores,
		DependencyGraph:    dependencyGraph,
		SelectionReasoning: reasoning,
		CoverageScore:      coverageScore,
	}, nil
}

// analyzeQueryAndDetermineStrategy analyzes the query to determine optimal selection strategy
func (ss *SmartSelector) analyzeQueryAndDetermineStrategy(query string, modelID string) SelectionStrategy {
	queryLower := strings.ToLower(query)
	modelConfig := ss.config.GetModelConfig(modelID)

	// Base strategy
	strategy := SelectionStrategy{
		MaxTokens:           int(float64(modelConfig.ContextWindow) * 0.8), // Use 80% of context window
		RelevanceThreshold:  0.3,
		IncludeDependencies: true,
		PreserveOrder:       true,
		IncludeRecent:       true,
		RecentCount:         5,
		BalanceRoles:        true,
		PrioritizeCode:      false,
	}

	// Adjust based on query characteristics

	// Code-related queries
	if ss.isCodeQuery(queryLower) {
		strategy.PrioritizeCode = true
		strategy.RelevanceThreshold = 0.4   // Higher threshold for code
		strategy.IncludeDependencies = true // Code often has dependencies
	}

	// Question queries
	if ss.isQuestionQuery(queryLower) {
		strategy.IncludeRecent = true
		strategy.RecentCount = 10 // Include more recent context for questions
		strategy.BalanceRoles = true
	}

	// Debugging queries
	if ss.isDebuggingQuery(queryLower) {
		strategy.PrioritizeCode = true
		strategy.IncludeDependencies = true
		strategy.RelevanceThreshold = 0.2 // Lower threshold to include more context
	}

	// Summary/overview queries
	if ss.isSummaryQuery(queryLower) {
		strategy.BalanceRoles = true
		strategy.RelevanceThreshold = 0.5    // Higher threshold for summaries
		strategy.IncludeDependencies = false // Less important for summaries
	}

	// Long context queries
	if ss.isLongContextQuery(queryLower) {
		strategy.MaxTokens = int(float64(modelConfig.ContextWindow) * 0.95) // Use more context
		strategy.RelevanceThreshold = 0.2                                   // Lower threshold to include more
		strategy.IncludeRecent = true
		strategy.RecentCount = 20
	}

	return strategy
}

// isCodeQuery checks if the query is code-related
func (ss *SmartSelector) isCodeQuery(query string) bool {
	codeKeywords := []string{
		"function", "class", "method", "variable", "code", "implement", "debug",
		"error", "bug", "syntax", "compile", "run", "execute", "algorithm",
		"refactor", "optimize", "test", "unit test", "integration",
	}

	for _, keyword := range codeKeywords {
		if strings.Contains(query, keyword) {
			return true
		}
	}

	return false
}

// isQuestionQuery checks if the query is a question
func (ss *SmartSelector) isQuestionQuery(query string) bool {
	return strings.Contains(query, "?") ||
		strings.HasPrefix(query, "how ") ||
		strings.HasPrefix(query, "what ") ||
		strings.HasPrefix(query, "why ") ||
		strings.HasPrefix(query, "when ") ||
		strings.HasPrefix(query, "where ") ||
		strings.HasPrefix(query, "which ") ||
		strings.HasPrefix(query, "who ")
}

// isDebuggingQuery checks if the query is about debugging
func (ss *SmartSelector) isDebuggingQuery(query string) bool {
	debugKeywords := []string{
		"debug", "error", "bug", "issue", "problem", "fix", "broken",
		"not working", "fails", "crash", "exception", "stack trace",
	}

	for _, keyword := range debugKeywords {
		if strings.Contains(query, keyword) {
			return true
		}
	}

	return false
}

// isSummaryQuery checks if the query is asking for a summary
func (ss *SmartSelector) isSummaryQuery(query string) bool {
	summaryKeywords := []string{
		"summary", "summarize", "overview", "recap", "review",
		"what happened", "what did we", "progress", "status",
	}

	for _, keyword := range summaryKeywords {
		if strings.Contains(query, keyword) {
			return true
		}
	}

	return false
}

// isLongContextQuery checks if the query needs long context
func (ss *SmartSelector) isLongContextQuery(query string) bool {
	longContextKeywords := []string{
		"entire", "all", "complete", "full context", "everything",
		"comprehensive", "detailed", "thorough", "full picture",
	}

	for _, keyword := range longContextKeywords {
		if strings.Contains(query, keyword) {
			return true
		}
	}

	return false
}

// selectMessagesByStrategy selects messages based on the given strategy
func (ss *SmartSelector) selectMessagesByStrategy(messages []ConversationMessage, relevanceResult *RelevanceResult, dependencyGraph *DependencyGraph, strategy SelectionStrategy, modelID string) []ConversationMessage {
	// Start with relevant messages
	candidates := ss.getRelevantCandidates(messages, relevanceResult, strategy)

	// Add dependencies if needed
	if strategy.IncludeDependencies && dependencyGraph != nil {
		candidates = ss.addDependencies(candidates, dependencyGraph, messages)
	}

	// Add recent messages if needed
	if strategy.IncludeRecent {
		candidates = ss.addRecentMessages(candidates, messages, strategy.RecentCount)
	}

	// Balance roles if needed
	if strategy.BalanceRoles {
		candidates = ss.balanceRoles(candidates)
	}

	// Prioritize code if needed
	if strategy.PrioritizeCode {
		candidates = ss.prioritizeCodeMessages(candidates)
	}

	// Fit within token limit
	selected := ss.fitWithinTokenLimit(candidates, strategy.MaxTokens, modelID)

	// Preserve order if needed
	if strategy.PreserveOrder {
		selected = ss.preserveOriginalOrder(selected, messages)
	}

	return selected
}

// getRelevantCandidates gets messages that meet the relevance threshold
func (ss *SmartSelector) getRelevantCandidates(messages []ConversationMessage, relevanceResult *RelevanceResult, strategy SelectionStrategy) []ConversationMessage {
	var candidates []ConversationMessage

	for _, score := range relevanceResult.Scores {
		if score.Score >= strategy.RelevanceThreshold {
			candidates = append(candidates, messages[score.MessageIndex])
		}
	}

	return candidates
}

// addDependencies adds messages that are dependencies of selected messages
func (ss *SmartSelector) addDependencies(candidates []ConversationMessage, dependencyGraph *DependencyGraph, allMessages []ConversationMessage) []ConversationMessage {
	// Create a set of selected message indices
	selectedIndices := make(map[int]bool)
	for _, candidate := range candidates {
		for i, msg := range allMessages {
			if msg.Timestamp == candidate.Timestamp && msg.Content == candidate.Content {
				selectedIndices[i] = true
				break
			}
		}
	}

	// Add dependencies
	for selectedIndex := range selectedIndices {
		msgID := fmt.Sprintf("msg_%d", selectedIndex)
		required := ss.dependencyTracker.GetRequiredMessages(dependencyGraph, msgID)

		for _, reqMsgID := range required {
			var reqIndex int
			fmt.Sscanf(reqMsgID, "msg_%d", &reqIndex)
			if !selectedIndices[reqIndex] && reqIndex < len(allMessages) {
				candidates = append(candidates, allMessages[reqIndex])
				selectedIndices[reqIndex] = true
			}
		}
	}

	return candidates
}

// addRecentMessages adds recent messages to ensure context continuity
func (ss *SmartSelector) addRecentMessages(candidates []ConversationMessage, allMessages []ConversationMessage, recentCount int) []ConversationMessage {
	if recentCount <= 0 || len(allMessages) == 0 {
		return candidates
	}

	// Get the most recent messages
	startIndex := len(allMessages) - recentCount
	if startIndex < 0 {
		startIndex = 0
	}

	recentMessages := allMessages[startIndex:]

	// Add recent messages that aren't already included
	candidateSet := make(map[string]bool)
	for _, candidate := range candidates {
		key := fmt.Sprintf("%d:%s", candidate.Timestamp, candidate.Content[:min(50, len(candidate.Content))])
		candidateSet[key] = true
	}

	for _, recent := range recentMessages {
		key := fmt.Sprintf("%d:%s", recent.Timestamp, recent.Content[:min(50, len(recent.Content))])
		if !candidateSet[key] {
			candidates = append(candidates, recent)
			candidateSet[key] = true
		}
	}

	return candidates
}

// balanceRoles ensures a good balance of user and assistant messages
func (ss *SmartSelector) balanceRoles(candidates []ConversationMessage) []ConversationMessage {
	userMsgs := []ConversationMessage{}
	assistantMsgs := []ConversationMessage{}
	otherMsgs := []ConversationMessage{}

	for _, msg := range candidates {
		switch msg.Role {
		case "user":
			userMsgs = append(userMsgs, msg)
		case "assistant":
			assistantMsgs = append(assistantMsgs, msg)
		default:
			otherMsgs = append(otherMsgs, msg)
		}
	}

	// Balance user and assistant messages (aim for roughly equal numbers)
	maxRole := max(len(userMsgs), len(assistantMsgs))
	if maxRole > 0 {
		userLimit := min(len(userMsgs), maxRole)
		assistantLimit := min(len(assistantMsgs), maxRole)

		balanced := append(userMsgs[:userLimit], assistantMsgs[:assistantLimit]...)
		balanced = append(balanced, otherMsgs...)
		return balanced
	}

	return candidates
}

// prioritizeCodeMessages gives priority to messages containing code
func (ss *SmartSelector) prioritizeCodeMessages(candidates []ConversationMessage) []ConversationMessage {
	codeMsgs := []ConversationMessage{}
	nonCodeMsgs := []ConversationMessage{}

	for _, msg := range candidates {
		if ss.containsCode(msg.Content) {
			codeMsgs = append(codeMsgs, msg)
		} else {
			nonCodeMsgs = append(nonCodeMsgs, msg)
		}
	}

	// Put code messages first
	return append(codeMsgs, nonCodeMsgs...)
}

// containsCode checks if content contains code
func (ss *SmartSelector) containsCode(content string) bool {
	codeIndicators := []string{"```", "`", "function", "class", "def ", "var ", "let ", "const "}
	contentLower := strings.ToLower(content)

	for _, indicator := range codeIndicators {
		if strings.Contains(contentLower, indicator) {
			return true
		}
	}

	return false
}

// fitWithinTokenLimit selects messages that fit within the token limit
func (ss *SmartSelector) fitWithinTokenLimit(candidates []ConversationMessage, maxTokens int, modelID string) []ConversationMessage {
	if len(candidates) == 0 {
		return candidates
	}

	// Sort candidates by priority (this could be enhanced with more sophisticated scoring)
	sort.Slice(candidates, func(i, j int) bool {
		// Prioritize recent messages and those with code
		iRecent := candidates[i].Timestamp > candidates[j].Timestamp
		jRecent := candidates[j].Timestamp > candidates[i].Timestamp
		iCode := ss.containsCode(candidates[i].Content)
		jCode := ss.containsCode(candidates[j].Content)

		if iCode && !jCode {
			return true
		}
		if !iCode && jCode {
			return false
		}

		return iRecent && !jRecent
	})

	// Select messages that fit within token limit
	var selected []ConversationMessage
	currentTokens := 0

	for _, candidate := range candidates {
		msgTokens := ss.tokenCounter.CountMessageTokens(candidate, modelID).TotalTokens
		if currentTokens+msgTokens <= maxTokens {
			selected = append(selected, candidate)
			currentTokens += msgTokens
		}
	}

	return selected
}

// preserveOriginalOrder sorts selected messages by their original order
func (ss *SmartSelector) preserveOriginalOrder(selected []ConversationMessage, original []ConversationMessage) []ConversationMessage {
	// Create a map of timestamp to original index
	timestampToIndex := make(map[int64]int)
	for i, msg := range original {
		timestampToIndex[msg.Timestamp] = i
	}

	// Sort selected messages by original index
	sort.Slice(selected, func(i, j int) bool {
		iIndex := timestampToIndex[selected[i].Timestamp]
		jIndex := timestampToIndex[selected[j].Timestamp]
		return iIndex < jIndex
	})

	return selected
}

// calculateCoverageScore calculates how well the selection covers the original conversation
func (ss *SmartSelector) calculateCoverageScore(selected, original []ConversationMessage, relevanceResult *RelevanceResult) float64 {
	if len(original) == 0 {
		return 1.0
	}

	// Calculate coverage based on relevance scores of selected messages
	totalRelevance := 0.0
	selectedRelevance := 0.0

	selectedSet := make(map[string]bool)
	for _, msg := range selected {
		key := fmt.Sprintf("%d:%s", msg.Timestamp, msg.Content[:min(50, len(msg.Content))])
		selectedSet[key] = true
	}

	for i, score := range relevanceResult.Scores {
		totalRelevance += score.Score

		if i < len(original) {
			msg := original[i]
			key := fmt.Sprintf("%d:%s", msg.Timestamp, msg.Content[:min(50, len(msg.Content))])
			if selectedSet[key] {
				selectedRelevance += score.Score
			}
		}
	}

	if totalRelevance == 0 {
		return float64(len(selected)) / float64(len(original))
	}

	return selectedRelevance / totalRelevance
}

// generateSelectionReasoning generates human-readable reasoning for the selection
func (ss *SmartSelector) generateSelectionReasoning(strategy SelectionStrategy, relevanceResult *RelevanceResult, selectedCount, totalTokens int) string {
	var reasons []string

	reasons = append(reasons, fmt.Sprintf("Selected %d/%d messages (%d tokens)",
		selectedCount, relevanceResult.OriginalCount, totalTokens))

	if strategy.RelevanceThreshold > 0 {
		reasons = append(reasons, fmt.Sprintf("relevance threshold: %.2f", strategy.RelevanceThreshold))
	}

	if strategy.IncludeDependencies {
		reasons = append(reasons, "included dependencies")
	}

	if strategy.IncludeRecent {
		reasons = append(reasons, fmt.Sprintf("included %d recent messages", strategy.RecentCount))
	}

	if strategy.PrioritizeCode {
		reasons = append(reasons, "prioritized code messages")
	}

	if strategy.BalanceRoles {
		reasons = append(reasons, "balanced user/assistant roles")
	}

	return strings.Join(reasons, ", ")
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
