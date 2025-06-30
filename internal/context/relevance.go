package context

import (
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
)

// RelevanceScorer handles context relevance scoring and filtering
type RelevanceScorer struct {
	tokenCounter *TokenCounter
}

// NewRelevanceScorer creates a new relevance scorer
func NewRelevanceScorer() *RelevanceScorer {
	return &RelevanceScorer{
		tokenCounter: NewTokenCounter(),
	}
}

// RelevanceScore represents the relevance score of a message
type RelevanceScore struct {
	MessageIndex    int     `json:"message_index"`
	Score           float64 `json:"score"`
	Factors         map[string]float64 `json:"factors"`
	Reasoning       string  `json:"reasoning"`
	ShouldInclude   bool    `json:"should_include"`
}

// RelevanceResult represents the result of relevance scoring
type RelevanceResult struct {
	Scores          []RelevanceScore `json:"scores"`
	FilteredMessages []ConversationMessage `json:"filtered_messages"`
	OriginalCount   int              `json:"original_count"`
	FilteredCount   int              `json:"filtered_count"`
	AverageScore    float64          `json:"average_score"`
	Threshold       float64          `json:"threshold"`
	Query           string           `json:"query"`
}

// ScoreRelevance scores messages based on relevance to a query
func (rs *RelevanceScorer) ScoreRelevance(messages []ConversationMessage, query string, threshold float64) (*RelevanceResult, error) {
	if len(messages) == 0 {
		return &RelevanceResult{
			Scores:        []RelevanceScore{},
			FilteredMessages: []ConversationMessage{},
			OriginalCount: 0,
			FilteredCount: 0,
			Threshold:     threshold,
			Query:         query,
		}, nil
	}

	queryTerms := rs.extractQueryTerms(query)
	scores := make([]RelevanceScore, len(messages))
	totalScore := 0.0

	// Score each message
	for i, msg := range messages {
		score := rs.scoreMessage(msg, queryTerms, query, i, len(messages))
		scores[i] = score
		totalScore += score.Score
	}

	// Calculate average score
	averageScore := totalScore / float64(len(messages))

	// Filter messages based on threshold
	var filteredMessages []ConversationMessage
	for i, score := range scores {
		if score.Score >= threshold {
			score.ShouldInclude = true
			filteredMessages = append(filteredMessages, messages[i])
		}
		scores[i] = score
	}

	return &RelevanceResult{
		Scores:           scores,
		FilteredMessages: filteredMessages,
		OriginalCount:    len(messages),
		FilteredCount:    len(filteredMessages),
		AverageScore:     averageScore,
		Threshold:        threshold,
		Query:            query,
	}, nil
}

// scoreMessage calculates relevance score for a single message
func (rs *RelevanceScorer) scoreMessage(msg ConversationMessage, queryTerms []string, query string, index, totalMessages int) RelevanceScore {
	factors := make(map[string]float64)
	
	// Factor 1: Content similarity (40% weight)
	contentScore := rs.calculateContentSimilarity(msg.Content, queryTerms, query)
	factors["content_similarity"] = contentScore

	// Factor 2: Recency (20% weight)
	recencyScore := rs.calculateRecencyScore(msg.Timestamp, index, totalMessages)
	factors["recency"] = recencyScore

	// Factor 3: Role importance (15% weight)
	roleScore := rs.calculateRoleScore(msg.Role)
	factors["role"] = roleScore

	// Factor 4: Message length (10% weight)
	lengthScore := rs.calculateLengthScore(msg.Content)
	factors["length"] = lengthScore

	// Factor 5: Code presence (10% weight)
	codeScore := rs.calculateCodeScore(msg.Content)
	factors["code_presence"] = codeScore

	// Factor 6: Question/Answer pattern (5% weight)
	qaScore := rs.calculateQAScore(msg.Content, msg.Role)
	factors["qa_pattern"] = qaScore

	// Calculate weighted total score
	totalScore := (contentScore * 0.4) + 
				  (recencyScore * 0.2) + 
				  (roleScore * 0.15) + 
				  (lengthScore * 0.1) + 
				  (codeScore * 0.1) + 
				  (qaScore * 0.05)

	// Generate reasoning
	reasoning := rs.generateReasoning(factors, totalScore)

	return RelevanceScore{
		MessageIndex:  index,
		Score:         totalScore,
		Factors:       factors,
		Reasoning:     reasoning,
		ShouldInclude: false, // Will be set later based on threshold
	}
}

// extractQueryTerms extracts important terms from the query
func (rs *RelevanceScorer) extractQueryTerms(query string) []string {
	// Convert to lowercase and split by whitespace
	words := strings.Fields(strings.ToLower(query))
	
	// Filter out common stop words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "is": true, "are": true, "was": true, "were": true,
		"be": true, "been": true, "have": true, "has": true, "had": true, "do": true,
		"does": true, "did": true, "will": true, "would": true, "could": true, "should": true,
		"can": true, "may": true, "might": true, "must": true, "shall": true,
	}
	
	var terms []string
	for _, word := range words {
		// Remove punctuation
		word = regexp.MustCompile(`[^\w]`).ReplaceAllString(word, "")
		if len(word) > 2 && !stopWords[word] {
			terms = append(terms, word)
		}
	}
	
	return terms
}

// calculateContentSimilarity calculates similarity between message content and query
func (rs *RelevanceScorer) calculateContentSimilarity(content string, queryTerms []string, query string) float64 {
	if len(queryTerms) == 0 {
		return 0.0
	}

	contentLower := strings.ToLower(content)
	queryLower := strings.ToLower(query)
	
	// Exact phrase match (highest score)
	if strings.Contains(contentLower, queryLower) {
		return 1.0
	}
	
	// Term matching
	matches := 0
	for _, term := range queryTerms {
		if strings.Contains(contentLower, term) {
			matches++
		}
	}
	
	// Partial term matching
	partialMatches := 0
	contentWords := strings.Fields(contentLower)
	for _, term := range queryTerms {
		for _, word := range contentWords {
			if len(word) > 3 && strings.Contains(word, term) {
				partialMatches++
				break
			}
		}
	}
	
	// Calculate score
	exactRatio := float64(matches) / float64(len(queryTerms))
	partialRatio := float64(partialMatches) / float64(len(queryTerms))
	
	return math.Min(1.0, exactRatio + (partialRatio * 0.3))
}

// calculateRecencyScore gives higher scores to more recent messages
func (rs *RelevanceScorer) calculateRecencyScore(timestamp int64, index, totalMessages int) float64 {
	// Position-based recency (more recent = higher index)
	positionScore := float64(index) / float64(totalMessages-1)
	
	// Time-based recency (if timestamp is available)
	timeScore := 0.5 // Default neutral score
	if timestamp > 0 {
		now := time.Now().Unix()
		age := now - timestamp
		
		// Messages within last hour get full score
		if age < 3600 {
			timeScore = 1.0
		} else if age < 86400 { // Last day
			timeScore = 0.8
		} else if age < 604800 { // Last week
			timeScore = 0.6
		} else {
			timeScore = 0.4
		}
	}
	
	return (positionScore + timeScore) / 2.0
}

// calculateRoleScore assigns scores based on message role
func (rs *RelevanceScorer) calculateRoleScore(role string) float64 {
	switch role {
	case "user":
		return 0.9 // User questions are usually important
	case "assistant":
		return 0.8 // Assistant responses are important
	case "system":
		return 0.6 // System messages are moderately important
	default:
		return 0.5 // Unknown roles get neutral score
	}
}

// calculateLengthScore gives moderate preference to medium-length messages
func (rs *RelevanceScorer) calculateLengthScore(content string) float64 {
	length := len(content)
	
	// Optimal length is around 200-1000 characters
	if length < 50 {
		return 0.3 // Too short
	} else if length < 200 {
		return 0.6 // Short but okay
	} else if length < 1000 {
		return 1.0 // Good length
	} else if length < 2000 {
		return 0.8 // Long but manageable
	} else {
		return 0.5 // Very long
	}
}

// calculateCodeScore gives higher scores to messages containing code
func (rs *RelevanceScorer) calculateCodeScore(content string) float64 {
	codeIndicators := []string{
		"```", "`", "function", "class", "def ", "var ", "let ", "const ",
		"import ", "export ", "return ", "if (", "for (", "while (",
		"{", "}", "[", "]", "//", "/*", "*/", "#include", "package ",
	}
	
	contentLower := strings.ToLower(content)
	matches := 0
	
	for _, indicator := range codeIndicators {
		if strings.Contains(contentLower, indicator) {
			matches++
		}
	}
	
	// Normalize score
	if matches == 0 {
		return 0.3 // No code
	} else if matches < 3 {
		return 0.6 // Some code
	} else if matches < 6 {
		return 0.9 // Moderate code
	} else {
		return 1.0 // Lots of code
	}
}

// calculateQAScore identifies question/answer patterns
func (rs *RelevanceScorer) calculateQAScore(content string, role string) float64 {
	contentLower := strings.ToLower(content)
	
	// Question indicators
	questionWords := []string{"what", "how", "why", "when", "where", "which", "who"}
	hasQuestion := strings.Contains(content, "?")
	
	questionScore := 0.0
	for _, word := range questionWords {
		if strings.Contains(contentLower, word) {
			questionScore += 0.2
		}
	}
	
	if hasQuestion {
		questionScore += 0.4
	}
	
	// Answer indicators
	answerWords := []string{"because", "since", "therefore", "however", "although", "here's", "you can"}
	answerScore := 0.0
	for _, word := range answerWords {
		if strings.Contains(contentLower, word) {
			answerScore += 0.1
		}
	}
	
	// Role-based adjustment
	if role == "user" && questionScore > 0 {
		return math.Min(1.0, questionScore + 0.2)
	} else if role == "assistant" && answerScore > 0 {
		return math.Min(1.0, answerScore + 0.3)
	}
	
	return math.Min(1.0, (questionScore + answerScore) / 2.0)
}

// generateReasoning creates human-readable reasoning for the score
func (rs *RelevanceScorer) generateReasoning(factors map[string]float64, totalScore float64) string {
	var reasons []string
	
	if factors["content_similarity"] > 0.7 {
		reasons = append(reasons, "high content similarity")
	} else if factors["content_similarity"] > 0.4 {
		reasons = append(reasons, "moderate content similarity")
	}
	
	if factors["recency"] > 0.8 {
		reasons = append(reasons, "recent message")
	}
	
	if factors["code_presence"] > 0.7 {
		reasons = append(reasons, "contains code")
	}
	
	if factors["qa_pattern"] > 0.7 {
		reasons = append(reasons, "question/answer pattern")
	}
	
	if len(reasons) == 0 {
		if totalScore > 0.7 {
			reasons = append(reasons, "generally relevant")
		} else if totalScore > 0.4 {
			reasons = append(reasons, "somewhat relevant")
		} else {
			reasons = append(reasons, "low relevance")
		}
	}
	
	return strings.Join(reasons, ", ")
}

// FilterByRelevance filters messages based on relevance scores
func (rs *RelevanceScorer) FilterByRelevance(messages []ConversationMessage, query string, threshold float64, maxMessages int) ([]ConversationMessage, *RelevanceResult, error) {
	result, err := rs.ScoreRelevance(messages, query, threshold)
	if err != nil {
		return nil, nil, err
	}
	
	// Sort by score (highest first)
	sort.Slice(result.Scores, func(i, j int) bool {
		return result.Scores[i].Score > result.Scores[j].Score
	})
	
	// Apply max message limit
	var filtered []ConversationMessage
	for i, score := range result.Scores {
		if i >= maxMessages {
			break
		}
		if score.ShouldInclude {
			filtered = append(filtered, messages[score.MessageIndex])
		}
	}
	
	result.FilteredMessages = filtered
	result.FilteredCount = len(filtered)
	
	return filtered, result, nil
}
