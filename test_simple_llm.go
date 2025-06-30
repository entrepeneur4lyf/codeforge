package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/entrepeneur4lyf/codeforge/internal/llm"
	"github.com/entrepeneur4lyf/codeforge/internal/llm/providers"
)

func main() {
	fmt.Println("🧪 Testing Simple LLM Call")
	fmt.Println("==========================")

	// Check API key
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("ANTHROPIC_API_KEY not set")
	}
	fmt.Printf("✅ API key found (length: %d)\n", len(apiKey))

	// Create handler
	options := llm.ApiHandlerOptions{
		ModelID: "claude-3-haiku-20240307",
		APIKey:  apiKey,
	}

	handler, err := providers.BuildApiHandler(options)
	if err != nil {
		log.Fatalf("Failed to create handler: %v", err)
	}
	fmt.Printf("✅ Handler created: %T\n", handler)

	// Simple test message
	systemPrompt := "You are a helpful assistant."
	messages := []llm.Message{
		{
			Role: "user",
			Content: []llm.ContentBlock{
				llm.TextBlock{Text: "Say hello in exactly 3 words."},
			},
		},
	}

	fmt.Println("🔍 Calling CreateMessage...")
	stream, err := handler.CreateMessage(context.Background(), systemPrompt, messages)
	if err != nil {
		log.Fatalf("CreateMessage failed: %v", err)
	}
	fmt.Println("✅ Stream created successfully")

	// Collect response
	var response string
	chunkCount := 0
	for chunk := range stream {
		chunkCount++
		fmt.Printf("🔍 Chunk %d: %T\n", chunkCount, chunk)
		if textChunk, ok := chunk.(llm.ApiStreamTextChunk); ok {
			fmt.Printf("   Text: '%s'\n", textChunk.Text)
			response += textChunk.Text
		}
	}

	fmt.Printf("✅ Total chunks: %d\n", chunkCount)
	fmt.Printf("✅ Final response: '%s'\n", response)

	if response == "" {
		fmt.Println("❌ Empty response received")
	} else {
		fmt.Println("🎉 Test successful!")
	}
}
