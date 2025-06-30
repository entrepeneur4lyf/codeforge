package transform

import (
	"encoding/json"
	"fmt"
)

// Local type definitions to avoid circular imports

// ContentBlock represents a content block in a message
type ContentBlock interface {
	IsContentBlock()
}

// TextBlock represents a text content block
type TextBlock struct {
	Text string `json:"text"`
}

func (TextBlock) IsContentBlock() {}

// ImageBlock represents an image content block
type ImageBlock struct {
	Source ImageSource `json:"source"`
}

func (ImageBlock) IsContentBlock() {}

// ImageSource represents the source of an image
type ImageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

// ToolUseBlock represents a tool use content block
type ToolUseBlock struct {
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

func (ToolUseBlock) IsContentBlock() {}

// ToolResultBlock represents a tool result content block
type ToolResultBlock struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

func (ToolResultBlock) IsContentBlock() {}

// Message represents a message in the conversation
type Message struct {
	Role    string         `json:"role"`
	Content []ContentBlock `json:"content"`
}

// OpenAIMessage represents an OpenAI API message
type OpenAIMessage struct {
	Role       string           `json:"role"`
	Content    interface{}      `json:"content,omitempty"`
	ToolCalls  []OpenAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	Name       string           `json:"name,omitempty"`
}

// OpenAIToolCall represents an OpenAI tool call
type OpenAIToolCall struct {
	ID       string             `json:"id"`
	Type     string             `json:"type"`
	Function OpenAIFunctionCall `json:"function"`
}

// OpenAIFunctionCall represents an OpenAI function call
type OpenAIFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// OpenAIContentPart represents different types of content in OpenAI format
type OpenAIContentPart struct {
	Type     string          `json:"type"`
	Text     string          `json:"text,omitempty"`
	ImageURL *OpenAIImageURL `json:"image_url,omitempty"`
}

// OpenAIImageURL represents an image URL in OpenAI format
type OpenAIImageURL struct {
	URL string `json:"url"`
}

// ConvertToOpenAIMessages converts Anthropic-style messages to OpenAI format
// Based on Cline's convertToOpenAiMessages function
func ConvertToOpenAIMessages(messages []Message) ([]OpenAIMessage, error) {
	var openAIMessages []OpenAIMessage

	for _, message := range messages {
		if len(message.Content) == 0 {
			continue
		}

		// Handle simple text content
		if len(message.Content) == 1 {
			if textBlock, ok := message.Content[0].(TextBlock); ok {
				openAIMessages = append(openAIMessages, OpenAIMessage{
					Role:    message.Role,
					Content: textBlock.Text,
				})
				continue
			}
		}

		// Handle complex content with multiple blocks
		if message.Role == "user" {
			userMessage, err := convertUserMessage(message)
			if err != nil {
				return nil, fmt.Errorf("failed to convert user message: %w", err)
			}
			openAIMessages = append(openAIMessages, userMessage...)
		} else if message.Role == "assistant" {
			assistantMessage, err := convertAssistantMessage(message)
			if err != nil {
				return nil, fmt.Errorf("failed to convert assistant message: %w", err)
			}
			openAIMessages = append(openAIMessages, assistantMessage)
		} else {
			// Handle system and other roles as simple text
			text := extractTextFromContent(message.Content)
			openAIMessages = append(openAIMessages, OpenAIMessage{
				Role:    message.Role,
				Content: text,
			})
		}
	}

	return openAIMessages, nil
}

// convertUserMessage converts a user message with potentially complex content
func convertUserMessage(message Message) ([]OpenAIMessage, error) {
	var messages []OpenAIMessage
	var nonToolContent []OpenAIContentPart
	var toolResults []ToolResultBlock

	// Separate tool results from other content
	for _, block := range message.Content {
		switch b := block.(type) {
		case ToolResultBlock:
			toolResults = append(toolResults, b)
		case TextBlock:
			nonToolContent = append(nonToolContent, OpenAIContentPart{
				Type: "text",
				Text: b.Text,
			})
		case ImageBlock:
			imageURL := fmt.Sprintf("data:%s;base64,%s", b.Source.MediaType, b.Source.Data)
			nonToolContent = append(nonToolContent, OpenAIContentPart{
				Type: "image_url",
				ImageURL: &OpenAIImageURL{
					URL: imageURL,
				},
			})
		}
	}

	// Process tool results first (they must follow tool use messages)
	for _, toolResult := range toolResults {
		content := toolResult.Content
		messages = append(messages, OpenAIMessage{
			Role:       "tool",
			ToolCallID: toolResult.ToolUseID,
			Content:    content,
		})
	}

	// Process non-tool content
	if len(nonToolContent) > 0 {
		var content interface{}
		if len(nonToolContent) == 1 && nonToolContent[0].Type == "text" {
			content = nonToolContent[0].Text
		} else {
			content = nonToolContent
		}

		messages = append(messages, OpenAIMessage{
			Role:    "user",
			Content: content,
		})
	}

	return messages, nil
}

// convertAssistantMessage converts an assistant message with potentially tool calls
func convertAssistantMessage(message Message) (OpenAIMessage, error) {
	var textContent []string
	var toolCalls []OpenAIToolCall

	for _, block := range message.Content {
		switch b := block.(type) {
		case TextBlock:
			textContent = append(textContent, b.Text)
		case ToolUseBlock:
			arguments, err := json.Marshal(b.Input)
			if err != nil {
				return OpenAIMessage{}, fmt.Errorf("failed to marshal tool arguments: %w", err)
			}

			toolCalls = append(toolCalls, OpenAIToolCall{
				ID:   b.ID,
				Type: "function",
				Function: OpenAIFunctionCall{
					Name:      b.Name,
					Arguments: string(arguments),
				},
			})
		}
	}

	// Combine text content
	var content interface{}
	if len(textContent) > 0 {
		combinedText := ""
		for _, text := range textContent {
			combinedText += text
		}
		content = combinedText
	}

	openAIMessage := OpenAIMessage{
		Role:    "assistant",
		Content: content,
	}

	// Add tool calls if present (cannot be empty array)
	if len(toolCalls) > 0 {
		openAIMessage.ToolCalls = toolCalls
	}

	return openAIMessage, nil
}

// extractTextFromContent extracts text from content blocks
func extractTextFromContent(content []ContentBlock) string {
	var text string
	for _, block := range content {
		if textBlock, ok := block.(TextBlock); ok {
			text += textBlock.Text
		}
	}
	return text
}

// ConvertFromOpenAIMessage converts an OpenAI message back to Anthropic format
func ConvertFromOpenAIMessage(openAIMsg OpenAIMessage) (Message, error) {
	message := Message{
		Role:    openAIMsg.Role,
		Content: make([]ContentBlock, 0),
	}

	// Handle content
	if openAIMsg.Content != nil {
		switch content := openAIMsg.Content.(type) {
		case string:
			if content != "" {
				message.Content = append(message.Content, TextBlock{Text: content})
			}
		case []interface{}:
			for _, part := range content {
				if partMap, ok := part.(map[string]interface{}); ok {
					if partType, exists := partMap["type"]; exists && partType == "text" {
						if text, exists := partMap["text"]; exists {
							if textStr, ok := text.(string); ok {
								message.Content = append(message.Content, TextBlock{Text: textStr})
							}
						}
					}
				}
			}
		}
	}

	// Handle tool calls
	for i, toolCall := range openAIMsg.ToolCalls {
		var input map[string]interface{}
		if toolCall.Function.Arguments != "" {
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &input); err != nil {
				return message, fmt.Errorf("failed to unmarshal tool arguments: %w", err)
			}
		}

		toolUseID := toolCall.ID
		if toolUseID == "" {
			toolUseID = fmt.Sprintf("call_%d", i)
		}

		message.Content = append(message.Content, ToolUseBlock{
			ID:    toolUseID,
			Name:  toolCall.Function.Name,
			Input: input,
		})
	}

	return message, nil
}

// CreateSystemMessage creates a system message in OpenAI format
func CreateSystemMessage(systemPrompt string) OpenAIMessage {
	return OpenAIMessage{
		Role:    "system",
		Content: systemPrompt,
	}
}

// ConvertToOpenAI converts system prompt and messages to OpenAI format
func ConvertToOpenAI(systemPrompt string, messages []Message) ([]OpenAIMessage, error) {
	var result []OpenAIMessage

	// Add system message if provided
	if systemPrompt != "" {
		result = append(result, CreateSystemMessage(systemPrompt))
	}

	// Convert and add user/assistant messages
	openAIMessages, err := ConvertToOpenAIMessages(messages)
	if err != nil {
		return nil, err
	}

	result = append(result, openAIMessages...)
	return result, nil
}
