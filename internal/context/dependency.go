package context

import (
	"fmt"
	"regexp"
	"strings"
)

// DependencyTracker tracks dependencies between context elements
type DependencyTracker struct {
	dependencies map[string][]string // message ID -> dependent message IDs
	references   map[string][]string // message ID -> referenced message IDs
	entities     map[string][]string // entity -> message IDs that mention it
}

// NewDependencyTracker creates a new dependency tracker
func NewDependencyTracker() *DependencyTracker {
	return &DependencyTracker{
		dependencies: make(map[string][]string),
		references:   make(map[string][]string),
		entities:     make(map[string][]string),
	}
}

// Dependency represents a dependency between messages
type Dependency struct {
	FromMessageID string  `json:"from_message_id"`
	ToMessageID   string  `json:"to_message_id"`
	Type          string  `json:"type"`     // "reference", "continuation", "answer", "code_dependency"
	Strength      float64 `json:"strength"` // 0.0 to 1.0
	Reason        string  `json:"reason"`   // Human-readable explanation
}

// DependencyGraph represents the complete dependency graph
type DependencyGraph struct {
	Messages     []ConversationMessage `json:"messages"`
	Dependencies []Dependency          `json:"dependencies"`
	Entities     map[string][]string   `json:"entities"` // entity -> message IDs
	Clusters     []MessageCluster      `json:"clusters"` // Related message groups
}

// MessageCluster represents a group of related messages
type MessageCluster struct {
	ID          string   `json:"id"`
	MessageIDs  []string `json:"message_ids"`
	Topic       string   `json:"topic"`
	Strength    float64  `json:"strength"`
	Description string   `json:"description"`
}

// BuildDependencyGraph analyzes messages and builds a dependency graph
func (dt *DependencyTracker) BuildDependencyGraph(messages []ConversationMessage) *DependencyGraph {
	// Reset internal state
	dt.dependencies = make(map[string][]string)
	dt.references = make(map[string][]string)
	dt.entities = make(map[string][]string)

	var dependencies []Dependency

	// Extract entities from all messages first
	for i, msg := range messages {
		msgID := fmt.Sprintf("msg_%d", i)
		entities := dt.extractEntities(msg.Content)

		for _, entity := range entities {
			dt.entities[entity] = append(dt.entities[entity], msgID)
		}
	}

	// Analyze dependencies between messages
	for i, msg := range messages {
		msgID := fmt.Sprintf("msg_%d", i)

		// Look for dependencies with previous messages
		for j := 0; j < i; j++ {
			prevMsg := messages[j]
			prevMsgID := fmt.Sprintf("msg_%d", j)

			deps := dt.findDependencies(msg, prevMsg, msgID, prevMsgID)
			dependencies = append(dependencies, deps...)
		}
	}

	// Build clusters based on dependencies and entities
	clusters := dt.buildClusters(messages, dependencies)

	return &DependencyGraph{
		Messages:     messages,
		Dependencies: dependencies,
		Entities:     dt.entities,
		Clusters:     clusters,
	}
}

// findDependencies finds dependencies between two messages
func (dt *DependencyTracker) findDependencies(msg, prevMsg ConversationMessage, msgID, prevMsgID string) []Dependency {
	var deps []Dependency

	// 1. Direct reference dependency
	if refDep := dt.findReferenceDepedency(msg, prevMsg, msgID, prevMsgID); refDep != nil {
		deps = append(deps, *refDep)
	}

	// 2. Question-answer dependency
	if qaDep := dt.findQuestionAnswerDependency(msg, prevMsg, msgID, prevMsgID); qaDep != nil {
		deps = append(deps, *qaDep)
	}

	// 3. Code dependency
	if codeDep := dt.findCodeDependency(msg, prevMsg, msgID, prevMsgID); codeDep != nil {
		deps = append(deps, *codeDep)
	}

	// 4. Entity-based dependency
	if entityDep := dt.findEntityDependency(msg, prevMsg, msgID, prevMsgID); entityDep != nil {
		deps = append(deps, *entityDep)
	}

	// 5. Continuation dependency
	if contDep := dt.findContinuationDependency(msg, prevMsg, msgID, prevMsgID); contDep != nil {
		deps = append(deps, *contDep)
	}

	return deps
}

// findReferenceDepedency finds explicit references between messages
func (dt *DependencyTracker) findReferenceDepedency(msg, _ ConversationMessage, msgID, prevMsgID string) *Dependency {
	content := strings.ToLower(msg.Content)

	// Look for reference patterns
	referencePatterns := []string{
		"as mentioned", "as discussed", "as shown", "as you said", "like you said",
		"from above", "from before", "previously", "earlier", "as we saw",
		"referring to", "based on", "following up", "continuing from",
	}

	for _, pattern := range referencePatterns {
		if strings.Contains(content, pattern) {
			return &Dependency{
				FromMessageID: msgID,
				ToMessageID:   prevMsgID,
				Type:          "reference",
				Strength:      0.8,
				Reason:        fmt.Sprintf("Contains reference pattern: '%s'", pattern),
			}
		}
	}

	return nil
}

// findQuestionAnswerDependency finds question-answer relationships
func (dt *DependencyTracker) findQuestionAnswerDependency(msg, prevMsg ConversationMessage, msgID, prevMsgID string) *Dependency {
	// Check if previous message is a question and current is an answer
	if prevMsg.Role == "user" && msg.Role == "assistant" {
		prevContent := strings.ToLower(prevMsg.Content)
		currentContent := strings.ToLower(msg.Content)

		// Check for question indicators in previous message
		hasQuestion := strings.Contains(prevMsg.Content, "?") ||
			strings.Contains(prevContent, "how") ||
			strings.Contains(prevContent, "what") ||
			strings.Contains(prevContent, "why") ||
			strings.Contains(prevContent, "when") ||
			strings.Contains(prevContent, "where")

		// Check for answer indicators in current message
		hasAnswer := strings.Contains(currentContent, "here's") ||
			strings.Contains(currentContent, "you can") ||
			strings.Contains(currentContent, "to do this") ||
			strings.Contains(currentContent, "the answer")

		if hasQuestion {
			strength := 0.9
			if hasAnswer {
				strength = 1.0
			}

			return &Dependency{
				FromMessageID: msgID,
				ToMessageID:   prevMsgID,
				Type:          "answer",
				Strength:      strength,
				Reason:        "Assistant response to user question",
			}
		}
	}

	return nil
}

// findCodeDependency finds code-related dependencies
func (dt *DependencyTracker) findCodeDependency(msg, prevMsg ConversationMessage, msgID, prevMsgID string) *Dependency {
	// Extract code blocks and function names
	currentCode := dt.extractCodeElements(msg.Content)
	prevCode := dt.extractCodeElements(prevMsg.Content)

	if len(currentCode) == 0 || len(prevCode) == 0 {
		return nil
	}

	// Look for shared code elements
	sharedElements := 0
	for _, curr := range currentCode {
		for _, prev := range prevCode {
			if curr == prev {
				sharedElements++
			}
		}
	}

	if sharedElements > 0 {
		strength := float64(sharedElements) / float64(len(currentCode))
		if strength > 1.0 {
			strength = 1.0
		}

		return &Dependency{
			FromMessageID: msgID,
			ToMessageID:   prevMsgID,
			Type:          "code_dependency",
			Strength:      strength,
			Reason:        fmt.Sprintf("Shares %d code elements", sharedElements),
		}
	}

	return nil
}

// findEntityDependency finds entity-based dependencies
func (dt *DependencyTracker) findEntityDependency(msg, prevMsg ConversationMessage, msgID, prevMsgID string) *Dependency {
	currentEntities := dt.extractEntities(msg.Content)
	prevEntities := dt.extractEntities(prevMsg.Content)

	if len(currentEntities) == 0 || len(prevEntities) == 0 {
		return nil
	}

	// Find shared entities
	sharedEntities := 0
	for _, curr := range currentEntities {
		for _, prev := range prevEntities {
			if curr == prev {
				sharedEntities++
			}
		}
	}

	if sharedEntities > 0 {
		strength := float64(sharedEntities) / float64(len(currentEntities))
		if strength > 1.0 {
			strength = 1.0
		}

		return &Dependency{
			FromMessageID: msgID,
			ToMessageID:   prevMsgID,
			Type:          "entity_dependency",
			Strength:      strength,
			Reason:        fmt.Sprintf("Shares %d entities", sharedEntities),
		}
	}

	return nil
}

// findContinuationDependency finds continuation relationships
func (dt *DependencyTracker) findContinuationDependency(msg, prevMsg ConversationMessage, msgID, prevMsgID string) *Dependency {
	// Check if messages are from the same role and close in time
	if msg.Role == prevMsg.Role {
		content := strings.ToLower(msg.Content)

		continuationPatterns := []string{
			"also", "additionally", "furthermore", "moreover", "and",
			"next", "then", "after that", "following", "continuing",
		}

		for _, pattern := range continuationPatterns {
			if strings.HasPrefix(content, pattern) {
				return &Dependency{
					FromMessageID: msgID,
					ToMessageID:   prevMsgID,
					Type:          "continuation",
					Strength:      0.7,
					Reason:        fmt.Sprintf("Continuation pattern: '%s'", pattern),
				}
			}
		}
	}

	return nil
}

// extractEntities extracts named entities from text
func (dt *DependencyTracker) extractEntities(content string) []string {
	var entities []string

	// Extract function names
	funcPattern := regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)\s*\(`)
	funcMatches := funcPattern.FindAllStringSubmatch(content, -1)
	for _, match := range funcMatches {
		if len(match) > 1 {
			entities = append(entities, "func:"+match[1])
		}
	}

	// Extract class names (capitalized words)
	classPattern := regexp.MustCompile(`\b([A-Z][a-zA-Z0-9_]*)\b`)
	classMatches := classPattern.FindAllStringSubmatch(content, -1)
	for _, match := range classMatches {
		if len(match) > 1 && len(match[1]) > 2 {
			entities = append(entities, "class:"+match[1])
		}
	}

	// Extract file names
	filePattern := regexp.MustCompile(`\b([a-zA-Z0-9_-]+\.[a-zA-Z0-9]+)\b`)
	fileMatches := filePattern.FindAllStringSubmatch(content, -1)
	for _, match := range fileMatches {
		if len(match) > 1 {
			entities = append(entities, "file:"+match[1])
		}
	}

	// Extract variables (simple heuristic)
	varPattern := regexp.MustCompile(`\b([a-z][a-zA-Z0-9_]*)\b`)
	varMatches := varPattern.FindAllStringSubmatch(content, -1)
	for _, match := range varMatches {
		if len(match) > 1 && len(match[1]) > 3 {
			// Filter out common words
			word := match[1]
			if !dt.isCommonWord(word) {
				entities = append(entities, "var:"+word)
			}
		}
	}

	return dt.deduplicateEntities(entities)
}

// extractCodeElements extracts code-specific elements
func (dt *DependencyTracker) extractCodeElements(content string) []string {
	var elements []string

	// Extract function calls
	funcPattern := regexp.MustCompile(`\b([a-zA-Z_][a-zA-Z0-9_]*)\s*\(`)
	funcMatches := funcPattern.FindAllStringSubmatch(content, -1)
	for _, match := range funcMatches {
		if len(match) > 1 {
			elements = append(elements, match[1])
		}
	}

	// Extract imports/includes
	importPattern := regexp.MustCompile(`(?:import|include|require)\s+['""]([^'""]+)['""]\s*`)
	importMatches := importPattern.FindAllStringSubmatch(content, -1)
	for _, match := range importMatches {
		if len(match) > 1 {
			elements = append(elements, match[1])
		}
	}

	return dt.deduplicateEntities(elements)
}

// isCommonWord checks if a word is a common English word
func (dt *DependencyTracker) isCommonWord(word string) bool {
	commonWords := map[string]bool{
		"this": true, "that": true, "with": true, "have": true, "will": true,
		"from": true, "they": true, "know": true, "want": true, "been": true,
		"good": true, "much": true, "some": true, "time": true, "very": true,
		"when": true, "come": true, "here": true, "just": true, "like": true,
		"long": true, "make": true, "many": true, "over": true, "such": true,
		"take": true, "than": true, "them": true, "well": true, "were": true,
	}

	return commonWords[word]
}

// deduplicateEntities removes duplicate entities
func (dt *DependencyTracker) deduplicateEntities(entities []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, entity := range entities {
		if !seen[entity] {
			seen[entity] = true
			result = append(result, entity)
		}
	}

	return result
}

// buildClusters groups related messages into clusters
func (dt *DependencyTracker) buildClusters(messages []ConversationMessage, dependencies []Dependency) []MessageCluster {
	// Create adjacency list from dependencies
	adjacency := make(map[string][]string)
	for _, dep := range dependencies {
		if dep.Strength > 0.5 { // Only consider strong dependencies
			adjacency[dep.FromMessageID] = append(adjacency[dep.FromMessageID], dep.ToMessageID)
			adjacency[dep.ToMessageID] = append(adjacency[dep.ToMessageID], dep.FromMessageID)
		}
	}

	// Find connected components (clusters)
	visited := make(map[string]bool)
	var clusters []MessageCluster
	clusterID := 0

	for i := range messages {
		msgID := fmt.Sprintf("msg_%d", i)
		if !visited[msgID] {
			cluster := dt.findCluster(msgID, adjacency, visited, messages)
			if len(cluster.MessageIDs) > 1 { // Only include clusters with multiple messages
				cluster.ID = fmt.Sprintf("cluster_%d", clusterID)
				clusters = append(clusters, cluster)
				clusterID++
			}
		}
	}

	return clusters
}

// findCluster performs DFS to find a connected component
func (dt *DependencyTracker) findCluster(startID string, adjacency map[string][]string, visited map[string]bool, messages []ConversationMessage) MessageCluster {
	var messageIDs []string
	var topics []string

	// DFS
	stack := []string{startID}

	for len(stack) > 0 {
		msgID := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if visited[msgID] {
			continue
		}

		visited[msgID] = true
		messageIDs = append(messageIDs, msgID)

		// Extract topic from message content
		if msgIndex := dt.extractMessageIndex(msgID); msgIndex < len(messages) {
			topic := dt.extractTopic(messages[msgIndex].Content)
			if topic != "" {
				topics = append(topics, topic)
			}
		}

		// Add neighbors to stack
		for _, neighbor := range adjacency[msgID] {
			if !visited[neighbor] {
				stack = append(stack, neighbor)
			}
		}
	}

	// Determine cluster topic
	clusterTopic := "General Discussion"
	if len(topics) > 0 {
		clusterTopic = dt.findMostCommonTopic(topics)
	}

	return MessageCluster{
		MessageIDs:  messageIDs,
		Topic:       clusterTopic,
		Strength:    float64(len(messageIDs)) / float64(len(messages)),
		Description: fmt.Sprintf("Cluster of %d related messages about %s", len(messageIDs), clusterTopic),
	}
}

// extractMessageIndex extracts message index from message ID
func (dt *DependencyTracker) extractMessageIndex(msgID string) int {
	var index int
	fmt.Sscanf(msgID, "msg_%d", &index)
	return index
}

// extractTopic extracts a topic from message content
func (dt *DependencyTracker) extractTopic(content string) string {
	// Simple topic extraction - look for key programming terms
	content = strings.ToLower(content)

	topics := map[string]string{
		"function":    "Functions",
		"class":       "Classes",
		"variable":    "Variables",
		"error":       "Error Handling",
		"test":        "Testing",
		"database":    "Database",
		"api":         "API",
		"algorithm":   "Algorithms",
		"performance": "Performance",
		"security":    "Security",
		"config":      "Configuration",
		"deploy":      "Deployment",
	}

	for keyword, topic := range topics {
		if strings.Contains(content, keyword) {
			return topic
		}
	}

	return ""
}

// findMostCommonTopic finds the most common topic in a list
func (dt *DependencyTracker) findMostCommonTopic(topics []string) string {
	counts := make(map[string]int)
	for _, topic := range topics {
		counts[topic]++
	}

	maxCount := 0
	mostCommon := "General Discussion"
	for topic, count := range counts {
		if count > maxCount {
			maxCount = count
			mostCommon = topic
		}
	}

	return mostCommon
}

// GetDependentMessages returns messages that depend on a given message
func (dt *DependencyTracker) GetDependentMessages(graph *DependencyGraph, messageID string) []string {
	var dependents []string

	for _, dep := range graph.Dependencies {
		if dep.ToMessageID == messageID {
			dependents = append(dependents, dep.FromMessageID)
		}
	}

	return dependents
}

// GetRequiredMessages returns messages required by a given message
func (dt *DependencyTracker) GetRequiredMessages(graph *DependencyGraph, messageID string) []string {
	var required []string

	for _, dep := range graph.Dependencies {
		if dep.FromMessageID == messageID {
			required = append(required, dep.ToMessageID)
		}
	}

	return required
}

// SortByDependencies sorts messages respecting dependency order
func (dt *DependencyTracker) SortByDependencies(graph *DependencyGraph) []ConversationMessage {
	// Topological sort based on dependencies
	inDegree := make(map[string]int)
	adjacency := make(map[string][]string)

	// Initialize
	for i := range graph.Messages {
		msgID := fmt.Sprintf("msg_%d", i)
		inDegree[msgID] = 0
	}

	// Build graph
	for _, dep := range graph.Dependencies {
		adjacency[dep.ToMessageID] = append(adjacency[dep.ToMessageID], dep.FromMessageID)
		inDegree[dep.FromMessageID]++
	}

	// Topological sort
	var queue []string
	for msgID, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, msgID)
		}
	}

	var sorted []string
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		sorted = append(sorted, current)

		for _, neighbor := range adjacency[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// Convert back to messages
	var result []ConversationMessage
	for _, msgID := range sorted {
		if index := dt.extractMessageIndex(msgID); index < len(graph.Messages) {
			result = append(result, graph.Messages[index])
		}
	}

	return result
}
