package context

import (
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"time"

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

// fitWithinTokenLimit selects messages that fit within the token limit using advanced scoring
func (ss *SmartSelector) fitWithinTokenLimit(candidates []ConversationMessage, maxTokens int, modelID string) []ConversationMessage {
	if len(candidates) == 0 {
		return candidates
	}

	// Calculate comprehensive scores for each candidate
	scoredCandidates := ss.calculateMessageScores(candidates, modelID)

	// Sort by score (descending - highest priority first)
	sort.Slice(scoredCandidates, func(i, j int) bool {
		return scoredCandidates[i].Score > scoredCandidates[j].Score
	})

	// Select messages using greedy algorithm with token budget
	return ss.selectOptimalMessages(scoredCandidates, maxTokens, modelID)
}

// ScoredMessage represents a message with its calculated priority score
type ScoredMessage struct {
	Message ConversationMessage
	Score   float64
	Tokens  int
}

// calculateMessageScores calculates comprehensive priority scores for messages
func (ss *SmartSelector) calculateMessageScores(candidates []ConversationMessage, modelID string) []ScoredMessage {
	scored := make([]ScoredMessage, len(candidates))
	now := time.Now()

	// Calculate base metrics for normalization
	var totalLength, maxLength int
	var newestTime, oldestTime time.Time

	for i, msg := range candidates {
		length := len(msg.Content)
		totalLength += length
		if length > maxLength {
			maxLength = length
		}

		msgTime := time.Unix(msg.Timestamp, 0)
		if i == 0 || msgTime.After(newestTime) {
			newestTime = msgTime
		}
		if i == 0 || msgTime.Before(oldestTime) {
			oldestTime = msgTime
		}
	}

	avgLength := float64(totalLength) / float64(len(candidates))
	timeRange := newestTime.Sub(oldestTime).Seconds()
	if timeRange == 0 {
		timeRange = 1 // Avoid division by zero
	}

	// Score each message
	for i, msg := range candidates {
		score := ss.calculateIndividualScore(msg, now, avgLength, float64(maxLength), timeRange, newestTime)
		tokens := ss.tokenCounter.CountMessageTokens(msg, modelID).TotalTokens

		scored[i] = ScoredMessage{
			Message: msg,
			Score:   score,
			Tokens:  tokens,
		}
	}

	return scored
}

// calculateIndividualScore calculates the priority score for a single message
func (ss *SmartSelector) calculateIndividualScore(msg ConversationMessage, now time.Time, avgLength, maxLength float64, timeRange float64, newestTime time.Time) float64 {
	var score float64

	msgTime := time.Unix(msg.Timestamp, 0)
	content := msg.Content

	// 1. Recency Score (0-30 points) - More recent messages are more relevant
	timeDiff := newestTime.Sub(msgTime).Seconds()
	recencyScore := 30.0 * (1.0 - (timeDiff / timeRange))
	if recencyScore < 0 {
		recencyScore = 0
	}
	score += recencyScore

	// 2. Content Type Score (0-25 points) - Code and technical content prioritized
	contentScore := 0.0
	if ss.containsCode(content) {
		contentScore += 15.0
	}
	if ss.containsError(content) {
		contentScore += 10.0 // Error messages are important for debugging
	}
	if ss.containsQuestion(content) {
		contentScore += 8.0 // Questions often need context
	}
	if ss.containsDecision(content) {
		contentScore += 12.0 // Decisions and conclusions are valuable
	}
	score += math.Min(contentScore, 25.0)

	// 3. Role Priority Score (0-20 points) - Different roles have different importance
	roleScore := ss.getRoleScore(msg.Role)
	score += roleScore

	// 4. Length Score (0-15 points) - Moderate length preferred (not too short, not too long)
	lengthScore := ss.calculateLengthScore(float64(len(content)), avgLength, maxLength)
	score += lengthScore

	// 5. Context Relevance Score (0-10 points) - Messages that reference other messages
	contextScore := 0.0
	if ss.containsReferences(content) {
		contextScore += 5.0
	}
	if ss.containsFollowUp(content) {
		contextScore += 5.0
	}
	score += contextScore

	// Normalize to 0-100 scale
	return math.Min(score, 100.0)
}

// selectOptimalMessages uses a greedy algorithm to select the best messages within token limit
func (ss *SmartSelector) selectOptimalMessages(scoredCandidates []ScoredMessage, maxTokens int, modelID string) []ConversationMessage {
	var selected []ConversationMessage
	currentTokens := 0

	// Greedy selection: take highest scoring messages that fit
	for _, candidate := range scoredCandidates {
		if currentTokens+candidate.Tokens <= maxTokens {
			selected = append(selected, candidate.Message)
			currentTokens += candidate.Tokens
		}
	}

	// If we have room and didn't select enough, try to fit smaller messages
	if len(selected) < len(scoredCandidates)/2 && currentTokens < int(float64(maxTokens)*0.8) {
		// Sort remaining by tokens (ascending) to fit more messages
		remaining := make([]ScoredMessage, 0)
		selectedMap := make(map[int64]bool)

		for _, sel := range selected {
			selectedMap[sel.Timestamp] = true
		}

		for _, candidate := range scoredCandidates {
			if !selectedMap[candidate.Message.Timestamp] {
				remaining = append(remaining, candidate)
			}
		}

		sort.Slice(remaining, func(i, j int) bool {
			return remaining[i].Tokens < remaining[j].Tokens
		})

		for _, candidate := range remaining {
			if currentTokens+candidate.Tokens <= maxTokens {
				selected = append(selected, candidate.Message)
				currentTokens += candidate.Tokens
			}
		}
	}

	// Sort selected messages by timestamp to maintain conversation order
	sort.Slice(selected, func(i, j int) bool {
		return selected[i].Timestamp < selected[j].Timestamp
	})

	return selected
}

// Helper methods for content analysis
func (ss *SmartSelector) containsError(content string) bool {
	errorIndicators := []string{"error", "exception", "failed", "failure", "panic", "crash", "bug", "issue"}
	contentLower := strings.ToLower(content)
	for _, indicator := range errorIndicators {
		if strings.Contains(contentLower, indicator) {
			return true
		}
	}
	return false
}

func (ss *SmartSelector) containsQuestion(content string) bool {
	return strings.Contains(content, "?") ||
		strings.Contains(strings.ToLower(content), "how") ||
		strings.Contains(strings.ToLower(content), "what") ||
		strings.Contains(strings.ToLower(content), "why") ||
		strings.Contains(strings.ToLower(content), "when") ||
		strings.Contains(strings.ToLower(content), "where")
}

func (ss *SmartSelector) containsDecision(content string) bool {
	decisionWords := []string{"decided", "conclusion", "solution", "resolved", "implemented", "chosen", "selected"}
	contentLower := strings.ToLower(content)
	for _, word := range decisionWords {
		if strings.Contains(contentLower, word) {
			return true
		}
	}
	return false
}

func (ss *SmartSelector) containsReferences(content string) bool {
	return strings.Contains(content, "mentioned") ||
		strings.Contains(content, "discussed") ||
		strings.Contains(content, "as you said") ||
		strings.Contains(content, "referring to")
}

func (ss *SmartSelector) containsFollowUp(content string) bool {
	return strings.Contains(strings.ToLower(content), "follow up") ||
		strings.Contains(strings.ToLower(content), "next step") ||
		strings.Contains(strings.ToLower(content), "continue") ||
		strings.Contains(strings.ToLower(content), "additionally")
}

func (ss *SmartSelector) getRoleScore(role string) float64 {
	switch strings.ToLower(role) {
	case "user":
		return 20.0 // User messages are highest priority
	case "assistant":
		return 15.0 // Assistant responses are important
	case "system":
		return 10.0 // System messages provide context
	default:
		return 5.0 // Unknown roles get minimal score
	}
}

func (ss *SmartSelector) calculateLengthScore(length, avgLength, maxLength float64) float64 {
	// Optimal length is around average, with penalty for very short or very long
	if length < avgLength*0.3 {
		return 5.0 // Too short, likely not very informative
	}
	if length > avgLength*3.0 {
		return 8.0 // Too long, might be verbose
	}
	if length >= avgLength*0.7 && length <= avgLength*1.5 {
		return 15.0 // Optimal length range
	}
	return 12.0 // Decent length
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
