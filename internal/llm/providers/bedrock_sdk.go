package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
	"github.com/entrepeneur4lyf/codeforge/internal/llm"
)

// BedrockSDKHandler implements the ApiHandler interface using the official AWS SDK v2
type BedrockSDKHandler struct {
	options llm.ApiHandlerOptions
	client  *bedrockruntime.Client
	region  string
}

// NewBedrockSDKHandler creates a new Bedrock handler using the official AWS SDK v2
func NewBedrockSDKHandler(options llm.ApiHandlerOptions) *BedrockSDKHandler {
	region := options.AWSRegion
	if region == "" {
		region = "us-east-1" // Default region
	}

	return &BedrockSDKHandler{
		options: options,
		region:  region,
		// Client will be created when needed with AWS credentials
	}
}

func (h *BedrockSDKHandler) GetModel() llm.ModelResponse {
	return llm.ModelResponse{
		ID:   h.options.ModelID,
		Info: h.getDefaultModelInfo(h.options.ModelID),
	}
}

func (h *BedrockSDKHandler) GetApiStreamUsage() (*llm.ApiStreamUsageChunk, error) {
	// Bedrock provides usage in the response
	return nil, nil
}

// CreateMessage sends a message to Bedrock and returns a streaming response
func (h *BedrockSDKHandler) CreateMessage(ctx context.Context, systemPrompt string, messages []llm.Message) (llm.ApiStream, error) {
	// Create client if not already created
	if h.client == nil {
		// Load AWS configuration
		cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(h.region))
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS config: %w", err)
		}

		// Override credentials if provided
		if h.options.AWSAccessKey != "" && h.options.AWSSecretKey != "" {
			cfg.Credentials = aws.NewCredentialsCache(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
				return aws.Credentials{
					AccessKeyID:     h.options.AWSAccessKey,
					SecretAccessKey: h.options.AWSSecretKey,
					SessionToken:    h.options.AWSSessionToken,
				}, nil
			}))
		}

		h.client = bedrockruntime.NewFromConfig(cfg)
	}

	// Convert messages to Bedrock format based on model provider
	body, err := h.convertMessages(systemPrompt, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to convert messages: %w", err)
	}

	// Create response channel
	responseChan := make(chan llm.ApiStreamChunk, 100)

	// Determine if streaming is supported
	if h.supportsStreaming() {
		// Use streaming API
		input := &bedrockruntime.InvokeModelWithResponseStreamInput{
			ModelId:     aws.String(h.options.ModelID),
			Body:        body,
			ContentType: aws.String("application/json"),
		}

		// Start goroutine to handle streaming
		go func() {
			defer close(responseChan)

			stream, err := h.client.InvokeModelWithResponseStream(ctx, input)
			if err != nil {
				fmt.Printf("Bedrock stream error: %v\n", err)
				return
			}
			defer stream.GetStream().Close()

			// Process streaming events
			for event := range stream.GetStream().Events() {
				switch e := event.(type) {
				case *types.ResponseStreamMemberChunk:
					// Parse the chunk and extract text
					if text := h.extractTextFromChunk(e.Value.Bytes); text != "" {
						responseChan <- llm.ApiStreamTextChunk{Text: text}
					}
				default:
					// Handle other event types including errors
					fmt.Printf("Bedrock stream event: %T\n", e)
				}
			}
		}()
	} else {
		// Use non-streaming API
		input := &bedrockruntime.InvokeModelInput{
			ModelId:     aws.String(h.options.ModelID),
			Body:        body,
			ContentType: aws.String("application/json"),
		}

		go func() {
			defer close(responseChan)

			result, err := h.client.InvokeModel(ctx, input)
			if err != nil {
				fmt.Printf("Bedrock invoke error: %v\n", err)
				return
			}

			// Parse response and extract text
			if text := h.extractTextFromResponse(result.Body); text != "" {
				responseChan <- llm.ApiStreamTextChunk{Text: text}
			}
		}()
	}

	return responseChan, nil
}

// extractTextFromContent extracts text content from ContentBlock slice
func (h *BedrockSDKHandler) extractTextFromContent(content []llm.ContentBlock) string {
	var text string
	for _, block := range content {
		if textBlock, ok := block.(llm.TextBlock); ok {
			text += textBlock.Text
		}
	}
	return text
}

// convertMessages converts messages to the appropriate format for the specific Bedrock model
func (h *BedrockSDKHandler) convertMessages(systemPrompt string, messages []llm.Message) ([]byte, error) {
	modelID := h.options.ModelID

	// Handle different model providers on Bedrock
	if strings.HasPrefix(modelID, "anthropic.") {
		return h.convertToAnthropicFormat(systemPrompt, messages)
	} else if strings.HasPrefix(modelID, "amazon.") {
		return h.convertToAmazonFormat(systemPrompt, messages)
	} else if strings.HasPrefix(modelID, "ai21.") {
		return h.convertToAI21Format(systemPrompt, messages)
	} else if strings.HasPrefix(modelID, "cohere.") {
		return h.convertToCohereFormat(systemPrompt, messages)
	} else if strings.HasPrefix(modelID, "meta.") {
		return h.convertToMetaFormat(systemPrompt, messages)
	} else if strings.HasPrefix(modelID, "mistral.") {
		return h.convertToMistralFormat(systemPrompt, messages)
	}

	// Default to Anthropic format
	return h.convertToAnthropicFormat(systemPrompt, messages)
}

// convertToAnthropicFormat converts messages to Anthropic Claude format on Bedrock
func (h *BedrockSDKHandler) convertToAnthropicFormat(systemPrompt string, messages []llm.Message) ([]byte, error) {
	type AnthropicMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	type AnthropicRequest struct {
		AnthropicVersion string             `json:"anthropic_version"`
		MaxTokens        int                `json:"max_tokens"`
		Messages         []AnthropicMessage `json:"messages"`
		System           string             `json:"system,omitempty"`
		Temperature      *float64           `json:"temperature,omitempty"`
	}

	var anthropicMessages []AnthropicMessage
	for _, msg := range messages {
		role := "user"
		if msg.Role == "assistant" {
			role = "assistant"
		}
		anthropicMessages = append(anthropicMessages, AnthropicMessage{
			Role:    role,
			Content: h.extractTextFromContent(msg.Content),
		})
	}

	model := h.GetModel()
	request := AnthropicRequest{
		AnthropicVersion: "bedrock-2023-05-31",
		MaxTokens:        model.Info.MaxTokens,
		Messages:         anthropicMessages,
		System:           systemPrompt,
		Temperature:      model.Info.Temperature,
	}

	return json.Marshal(request)
}

// convertToAmazonFormat converts messages to Amazon Titan format
func (h *BedrockSDKHandler) convertToAmazonFormat(systemPrompt string, messages []llm.Message) ([]byte, error) {
	// Amazon Titan format implementation
	type TitanRequest struct {
		InputText            string `json:"inputText"`
		TextGenerationConfig struct {
			MaxTokenCount int      `json:"maxTokenCount"`
			Temperature   *float64 `json:"temperature,omitempty"`
			TopP          *float64 `json:"topP,omitempty"`
		} `json:"textGenerationConfig"`
	}

	// Combine system prompt and messages into a single input text
	var inputText strings.Builder
	if systemPrompt != "" {
		inputText.WriteString(systemPrompt + "\n\n")
	}

	for _, msg := range messages {
		content := h.extractTextFromContent(msg.Content)
		if msg.Role == "user" {
			inputText.WriteString("Human: " + content + "\n\n")
		} else {
			inputText.WriteString("Assistant: " + content + "\n\n")
		}
	}
	inputText.WriteString("Assistant: ")

	model := h.GetModel()
	request := TitanRequest{
		InputText: inputText.String(),
	}
	request.TextGenerationConfig.MaxTokenCount = model.Info.MaxTokens
	request.TextGenerationConfig.Temperature = model.Info.Temperature

	return json.Marshal(request)
}

// convertToAI21Format converts messages to AI21 Jurassic format
func (h *BedrockSDKHandler) convertToAI21Format(systemPrompt string, messages []llm.Message) ([]byte, error) {
	type AI21CountPenalty struct {
		Scale               float64 `json:"scale"`
		ApplyToNumbers      bool    `json:"applyToNumbers"`
		ApplyToPunctuations bool    `json:"applyToPunctuations"`
		ApplyToStopwords    bool    `json:"applyToStopwords"`
		ApplyToWhitespaces  bool    `json:"applyToWhitespaces"`
		ApplyToEmojis       bool    `json:"applyToEmojis"`
	}

	type AI21PresencePenalty struct {
		Scale               float64 `json:"scale"`
		ApplyToNumbers      bool    `json:"applyToNumbers"`
		ApplyToPunctuations bool    `json:"applyToPunctuations"`
		ApplyToStopwords    bool    `json:"applyToStopwords"`
		ApplyToWhitespaces  bool    `json:"applyToWhitespaces"`
		ApplyToEmojis       bool    `json:"applyToEmojis"`
	}

	type AI21FrequencyPenalty struct {
		Scale               float64 `json:"scale"`
		ApplyToNumbers      bool    `json:"applyToNumbers"`
		ApplyToPunctuations bool    `json:"applyToPunctuations"`
		ApplyToStopwords    bool    `json:"applyToStopwords"`
		ApplyToWhitespaces  bool    `json:"applyToWhitespaces"`
		ApplyToEmojis       bool    `json:"applyToEmojis"`
	}

	type AI21Request struct {
		Prompt           string                `json:"prompt"`
		MaxTokens        int                   `json:"maxTokens"`
		Temperature      float64               `json:"temperature"`
		TopP             float64               `json:"topP"`
		StopSequences    []string              `json:"stopSequences,omitempty"`
		CountPenalty     *AI21CountPenalty     `json:"countPenalty,omitempty"`
		PresencePenalty  *AI21PresencePenalty  `json:"presencePenalty,omitempty"`
		FrequencyPenalty *AI21FrequencyPenalty `json:"frequencyPenalty,omitempty"`
	}

	// Build prompt in AI21 format
	var promptBuilder strings.Builder
	if systemPrompt != "" {
		promptBuilder.WriteString(systemPrompt + "\n\n")
	}

	for _, msg := range messages {
		content := h.extractTextFromContent(msg.Content)
		if msg.Role == "user" {
			promptBuilder.WriteString("Human: " + content + "\n\n")
		} else {
			promptBuilder.WriteString("Assistant: " + content + "\n\n")
		}
	}
	promptBuilder.WriteString("Assistant:")

	model := h.GetModel()
	temperature := 0.7
	if model.Info.Temperature != nil {
		temperature = *model.Info.Temperature
	}

	request := AI21Request{
		Prompt:        promptBuilder.String(),
		MaxTokens:     model.Info.MaxTokens,
		Temperature:   temperature,
		TopP:          1.0,
		StopSequences: []string{"Human:", "\n\nHuman:"},
		CountPenalty: &AI21CountPenalty{
			Scale:               0.0,
			ApplyToNumbers:      false,
			ApplyToPunctuations: false,
			ApplyToStopwords:    false,
			ApplyToWhitespaces:  false,
			ApplyToEmojis:       false,
		},
		PresencePenalty: &AI21PresencePenalty{
			Scale:               0.0,
			ApplyToNumbers:      false,
			ApplyToPunctuations: false,
			ApplyToStopwords:    false,
			ApplyToWhitespaces:  false,
			ApplyToEmojis:       false,
		},
		FrequencyPenalty: &AI21FrequencyPenalty{
			Scale:               0.0,
			ApplyToNumbers:      false,
			ApplyToPunctuations: false,
			ApplyToStopwords:    false,
			ApplyToWhitespaces:  false,
			ApplyToEmojis:       false,
		},
	}

	return json.Marshal(request)
}

func (h *BedrockSDKHandler) convertToCohereFormat(systemPrompt string, messages []llm.Message) ([]byte, error) {
	type CohereMessage struct {
		Role    string `json:"role"`
		Message string `json:"message"`
	}

	type CohereRequest struct {
		Message           string          `json:"message"`
		ChatHistory       []CohereMessage `json:"chat_history,omitempty"`
		Preamble          string          `json:"preamble,omitempty"`
		MaxTokens         int             `json:"max_tokens"`
		Temperature       float64         `json:"temperature"`
		P                 float64         `json:"p"`
		K                 int             `json:"k"`
		PromptTruncation  string          `json:"prompt_truncation"`
		FrequencyPenalty  float64         `json:"frequency_penalty"`
		PresencePenalty   float64         `json:"presence_penalty"`
		EndSequences      []string        `json:"end_sequences,omitempty"`
		ReturnLikelihoods string          `json:"return_likelihoods"`
	}

	var chatHistory []CohereMessage
	var currentMessage string

	// Process messages - last user message becomes the main message
	for i, msg := range messages {
		content := h.extractTextFromContent(msg.Content)
		if msg.Role == "user" {
			if i == len(messages)-1 {
				// Last user message becomes the main message
				currentMessage = content
			} else {
				// Previous user messages go to chat history
				chatHistory = append(chatHistory, CohereMessage{
					Role:    "USER",
					Message: content,
				})
			}
		} else if msg.Role == "assistant" {
			chatHistory = append(chatHistory, CohereMessage{
				Role:    "CHATBOT",
				Message: content,
			})
		}
	}

	// If no current message, use the last message or a default
	if currentMessage == "" && len(messages) > 0 {
		currentMessage = h.extractTextFromContent(messages[len(messages)-1].Content)
	}
	if currentMessage == "" {
		currentMessage = "Hello"
	}

	model := h.GetModel()
	temperature := 0.3
	if model.Info.Temperature != nil {
		temperature = *model.Info.Temperature
	}

	request := CohereRequest{
		Message:           currentMessage,
		ChatHistory:       chatHistory,
		Preamble:          systemPrompt,
		MaxTokens:         model.Info.MaxTokens,
		Temperature:       temperature,
		P:                 0.75,
		K:                 0,
		PromptTruncation:  "AUTO",
		FrequencyPenalty:  0.0,
		PresencePenalty:   0.0,
		EndSequences:      []string{},
		ReturnLikelihoods: "NONE",
	}

	return json.Marshal(request)
}

func (h *BedrockSDKHandler) convertToMetaFormat(systemPrompt string, messages []llm.Message) ([]byte, error) {
	type MetaRequest struct {
		Prompt      string  `json:"prompt"`
		MaxGenLen   int     `json:"max_gen_len"`
		Temperature float64 `json:"temperature"`
		TopP        float64 `json:"top_p"`
	}

	// Build prompt in Llama chat format
	var promptBuilder strings.Builder

	// Add system prompt if provided
	if systemPrompt != "" {
		promptBuilder.WriteString("<s>[INST] <<SYS>>\n")
		promptBuilder.WriteString(systemPrompt)
		promptBuilder.WriteString("\n<</SYS>>\n\n")
	} else {
		promptBuilder.WriteString("<s>[INST] ")
	}

	// Process messages in Llama chat format
	for i, msg := range messages {
		content := h.extractTextFromContent(msg.Content)

		if msg.Role == "user" {
			if i == 0 && systemPrompt != "" {
				// First user message after system prompt
				promptBuilder.WriteString(content + " [/INST]")
			} else if i == 0 {
				// First user message without system prompt
				promptBuilder.WriteString(content + " [/INST]")
			} else {
				// Subsequent user messages
				promptBuilder.WriteString("<s>[INST] " + content + " [/INST]")
			}
		} else if msg.Role == "assistant" {
			promptBuilder.WriteString(" " + content + " </s>")
		}
	}

	// If the last message was from user, we're ready for assistant response
	// If the last message was from assistant, add a new user turn
	if len(messages) > 0 && messages[len(messages)-1].Role == "assistant" {
		promptBuilder.WriteString("<s>[INST] Continue the conversation. [/INST]")
	}

	model := h.GetModel()
	temperature := 0.6
	if model.Info.Temperature != nil {
		temperature = *model.Info.Temperature
	}

	request := MetaRequest{
		Prompt:      promptBuilder.String(),
		MaxGenLen:   model.Info.MaxTokens,
		Temperature: temperature,
		TopP:        0.9,
	}

	return json.Marshal(request)
}

func (h *BedrockSDKHandler) convertToMistralFormat(systemPrompt string, messages []llm.Message) ([]byte, error) {
	type MistralMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}

	type MistralRequest struct {
		Messages    []MistralMessage `json:"messages"`
		MaxTokens   int              `json:"max_tokens"`
		Temperature float64          `json:"temperature"`
		TopP        float64          `json:"top_p"`
		TopK        int              `json:"top_k"`
		Stop        []string         `json:"stop,omitempty"`
	}

	var mistralMessages []MistralMessage

	// Add system message if provided
	if systemPrompt != "" {
		mistralMessages = append(mistralMessages, MistralMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	// Convert messages to Mistral format
	for _, msg := range messages {
		role := "user"
		if msg.Role == "assistant" {
			role = "assistant"
		} else if msg.Role == "system" {
			role = "system"
		}

		mistralMessages = append(mistralMessages, MistralMessage{
			Role:    role,
			Content: h.extractTextFromContent(msg.Content),
		})
	}

	model := h.GetModel()
	temperature := 0.7
	if model.Info.Temperature != nil {
		temperature = *model.Info.Temperature
	}

	request := MistralRequest{
		Messages:    mistralMessages,
		MaxTokens:   model.Info.MaxTokens,
		Temperature: temperature,
		TopP:        1.0,
		TopK:        -1,
		Stop:        []string{},
	}

	return json.Marshal(request)
}

// supportsStreaming checks if the model supports streaming
func (h *BedrockSDKHandler) supportsStreaming() bool {
	modelID := h.options.ModelID
	// Most Anthropic and Amazon models support streaming
	return strings.HasPrefix(modelID, "anthropic.") || strings.HasPrefix(modelID, "amazon.")
}

// extractTextFromChunk extracts text content from a streaming chunk
func (h *BedrockSDKHandler) extractTextFromChunk(data []byte) string {
	// Parse based on model type
	if strings.HasPrefix(h.options.ModelID, "anthropic.") {
		return h.extractAnthropicStreamText(data)
	} else if strings.HasPrefix(h.options.ModelID, "amazon.") {
		return h.extractAmazonStreamText(data)
	}
	return ""
}

// extractTextFromResponse extracts text from a non-streaming response
func (h *BedrockSDKHandler) extractTextFromResponse(data []byte) string {
	// Parse based on model type
	if strings.HasPrefix(h.options.ModelID, "anthropic.") {
		return h.extractAnthropicResponseText(data)
	} else if strings.HasPrefix(h.options.ModelID, "amazon.") {
		return h.extractAmazonResponseText(data)
	}
	return ""
}

// extractAnthropicStreamText extracts text from Anthropic streaming response
func (h *BedrockSDKHandler) extractAnthropicStreamText(data []byte) string {
	var response struct {
		Type  string `json:"type"`
		Delta struct {
			Text string `json:"text"`
		} `json:"delta"`
	}

	if err := json.Unmarshal(data, &response); err == nil && response.Type == "content_block_delta" {
		return response.Delta.Text
	}
	return ""
}

// extractAmazonStreamText extracts text from Amazon Titan streaming response
func (h *BedrockSDKHandler) extractAmazonStreamText(data []byte) string {
	var response struct {
		OutputText string `json:"outputText"`
	}

	if err := json.Unmarshal(data, &response); err == nil {
		return response.OutputText
	}
	return ""
}

// extractAnthropicResponseText extracts text from Anthropic non-streaming response
func (h *BedrockSDKHandler) extractAnthropicResponseText(data []byte) string {
	var response struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(data, &response); err == nil && len(response.Content) > 0 {
		return response.Content[0].Text
	}
	return ""
}

// extractAmazonResponseText extracts text from Amazon Titan non-streaming response
func (h *BedrockSDKHandler) extractAmazonResponseText(data []byte) string {
	var response struct {
		Results []struct {
			OutputText string `json:"outputText"`
		} `json:"results"`
	}

	if err := json.Unmarshal(data, &response); err == nil && len(response.Results) > 0 {
		return response.Results[0].OutputText
	}
	return ""
}

// getDefaultModelInfo returns default model information for Bedrock models
func (h *BedrockSDKHandler) getDefaultModelInfo(modelID string) llm.ModelInfo {
	// Default configuration varies by model provider
	info := llm.ModelInfo{
		MaxTokens:           4096,
		ContextWindow:       200000, // Most Bedrock models have large context
		SupportsImages:      false,
		SupportsPromptCache: false,
		Description:         fmt.Sprintf("AWS Bedrock model: %s", modelID),
	}

	// Adjust defaults based on model type
	if strings.HasPrefix(modelID, "anthropic.claude-3") {
		info.MaxTokens = 4096
		info.ContextWindow = 200000
		info.SupportsImages = true
		info.InputPrice = 3.0   // $3 per 1M tokens for Claude 3 Sonnet
		info.OutputPrice = 15.0 // $15 per 1M tokens for Claude 3 Sonnet
	} else if strings.HasPrefix(modelID, "amazon.titan") {
		info.MaxTokens = 4096
		info.ContextWindow = 32000
		info.InputPrice = 0.5  // $0.50 per 1M tokens for Titan
		info.OutputPrice = 1.5 // $1.50 per 1M tokens for Titan
	}

	return info
}
