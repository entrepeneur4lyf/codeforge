package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// Prompt Handlers

// handleCodeReviewPrompt handles code review assistance prompts
func (cfs *CodeForgeServer) handleCodeReviewPrompt(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	filePath := request.Params.Arguments["file_path"]
	if filePath == "" {
		return nil, fmt.Errorf("file_path argument is required")
	}

	// Validate and resolve path
	fullPath, err := cfs.validatePath(filePath)
	if err != nil {
		return nil, fmt.Errorf("invalid file path: %v", err)
	}

	// Check if file exists
	if !cfs.fileExists(fullPath) {
		return nil, fmt.Errorf("file not found: %s", filePath)
	}

	// Create prompt messages
	messages := []mcp.PromptMessage{
		mcp.NewPromptMessage(
			mcp.RoleUser,
			mcp.NewTextContent("Please review the following code file and provide constructive feedback on:"),
		),
		mcp.NewPromptMessage(
			mcp.RoleUser,
			mcp.NewTextContent("1. Code quality and best practices\n2. Potential bugs or issues\n3. Performance considerations\n4. Readability and maintainability\n5. Security concerns (if applicable)"),
		),
		mcp.NewPromptMessage(
			mcp.RoleUser,
			mcp.NewEmbeddedResource(mcp.TextResourceContents{
				URI:      fmt.Sprintf("codeforge://files/%s", filePath),
				MIMEType: detectMIMEType(filePath),
			}),
		),
		mcp.NewPromptMessage(
			mcp.RoleUser,
			mcp.NewTextContent("Please provide specific, actionable feedback with examples where possible."),
		),
	}

	return mcp.NewGetPromptResult(
		fmt.Sprintf("Code review for %s", filePath),
		messages,
	), nil
}

// handleDebuggingPrompt handles debugging assistance prompts
func (cfs *CodeForgeServer) handleDebuggingPrompt(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	errorMessage := request.Params.Arguments["error_message"]
	filePath := request.Params.Arguments["file_path"]

	var messages []mcp.PromptMessage

	// Start with the debugging context
	messages = append(messages, mcp.NewPromptMessage(
		mcp.RoleUser,
		mcp.NewTextContent("I need help debugging an issue. Please analyze the problem and provide step-by-step debugging guidance."),
	))

	// Add error message if provided
	if errorMessage != "" {
		messages = append(messages, mcp.NewPromptMessage(
			mcp.RoleUser,
			mcp.NewTextContent(fmt.Sprintf("Error message: %s", errorMessage)),
		))
	}

	// Add file content if provided
	if filePath != "" {
		// Validate and resolve path
		fullPath, err := cfs.validatePath(filePath)
		if err != nil {
			return nil, fmt.Errorf("invalid file path: %v", err)
		}

		// Check if file exists
		if cfs.fileExists(fullPath) {
			messages = append(messages, mcp.NewPromptMessage(
				mcp.RoleUser,
				mcp.NewTextContent(fmt.Sprintf("Relevant code file (%s):", filePath)),
			))
			messages = append(messages, mcp.NewPromptMessage(
				mcp.RoleUser,
				mcp.NewEmbeddedResource(mcp.TextResourceContents{
					URI:      fmt.Sprintf("codeforge://files/%s", filePath),
					MIMEType: detectMIMEType(filePath),
				}),
			))
		}
	}

	// Add debugging guidance request
	messages = append(messages, mcp.NewPromptMessage(
		mcp.RoleUser,
		mcp.NewTextContent("Please provide:\n1. Possible causes of the issue\n2. Step-by-step debugging approach\n3. Specific things to check or test\n4. Potential solutions\n5. Prevention strategies for the future"),
	))

	return mcp.NewGetPromptResult(
		"Debugging assistance",
		messages,
	), nil
}

// handleRefactoringPrompt handles refactoring guidance prompts
func (cfs *CodeForgeServer) handleRefactoringPrompt(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	target := request.Params.Arguments["target"]
	if target == "" {
		return nil, fmt.Errorf("target argument is required")
	}

	goal := request.Params.Arguments["goal"]
	if goal == "" {
		goal = "improve code quality"
	}

	// Create prompt messages
	messages := []mcp.PromptMessage{
		mcp.NewPromptMessage(
			mcp.RoleUser,
			mcp.NewTextContent(fmt.Sprintf("I need guidance on refactoring %s with the goal of %s.", target, goal)),
		),
		mcp.NewPromptMessage(
			mcp.RoleUser,
			mcp.NewTextContent("Please provide:"),
		),
		mcp.NewPromptMessage(
			mcp.RoleUser,
			mcp.NewTextContent("1. Analysis of the current code structure\n2. Specific refactoring recommendations\n3. Step-by-step refactoring plan\n4. Potential risks and how to mitigate them\n5. Testing strategy to ensure functionality is preserved\n6. Expected benefits of the refactoring"),
		),
	}

	// Add project context
	messages = append(messages, mcp.NewPromptMessage(
		mcp.RoleUser,
		mcp.NewTextContent("Project context:"),
	))
	messages = append(messages, mcp.NewPromptMessage(
		mcp.RoleUser,
		mcp.NewEmbeddedResource(mcp.TextResourceContents{
			URI:      "codeforge://project/metadata",
			MIMEType: "application/json",
		}),
	))

	return mcp.NewGetPromptResult(
		fmt.Sprintf("Refactoring guidance for %s", target),
		messages,
	), nil
}

// Additional prompt handlers can be added here for:
// - Documentation generation
// - Testing assistance
// - Performance optimization
// - Security review
// - Architecture guidance
// - etc.

// handleDocumentationPrompt handles documentation generation prompts
func (cfs *CodeForgeServer) handleDocumentationPrompt(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	filePath := request.Params.Arguments["file_path"]
	if filePath == "" {
		return nil, fmt.Errorf("file_path argument is required")
	}

	docType := request.Params.Arguments["type"]
	if docType == "" {
		docType = "api"
	}

	// Validate and resolve path
	fullPath, err := cfs.validatePath(filePath)
	if err != nil {
		return nil, fmt.Errorf("invalid file path: %v", err)
	}

	// Check if file exists
	if !cfs.fileExists(fullPath) {
		return nil, fmt.Errorf("file not found: %s", filePath)
	}

	// Create prompt messages
	messages := []mcp.PromptMessage{
		mcp.NewPromptMessage(
			mcp.RoleUser,
			mcp.NewTextContent(fmt.Sprintf("Please generate %s documentation for the following code:", docType)),
		),
		mcp.NewPromptMessage(
			mcp.RoleUser,
			mcp.NewEmbeddedResource(mcp.TextResourceContents{
				URI:      fmt.Sprintf("codeforge://files/%s", filePath),
				MIMEType: detectMIMEType(filePath),
			}),
		),
		mcp.NewPromptMessage(
			mcp.RoleUser,
			mcp.NewTextContent("Include:\n1. Clear descriptions of functionality\n2. Parameter and return value documentation\n3. Usage examples\n4. Any important notes or warnings"),
		),
	}

	return mcp.NewGetPromptResult(
		fmt.Sprintf("Documentation generation for %s", filePath),
		messages,
	), nil
}

// handleTestingPrompt handles testing assistance prompts
func (cfs *CodeForgeServer) handleTestingPrompt(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	filePath := request.Params.Arguments["file_path"]
	if filePath == "" {
		return nil, fmt.Errorf("file_path argument is required")
	}

	testType := request.Params.Arguments["test_type"]
	if testType == "" {
		testType = "unit"
	}

	// Validate and resolve path
	fullPath, err := cfs.validatePath(filePath)
	if err != nil {
		return nil, fmt.Errorf("invalid file path: %v", err)
	}

	// Check if file exists
	if !cfs.fileExists(fullPath) {
		return nil, fmt.Errorf("file not found: %s", filePath)
	}

	// Create prompt messages
	messages := []mcp.PromptMessage{
		mcp.NewPromptMessage(
			mcp.RoleUser,
			mcp.NewTextContent(fmt.Sprintf("Please help me create %s tests for the following code:", testType)),
		),
		mcp.NewPromptMessage(
			mcp.RoleUser,
			mcp.NewEmbeddedResource(mcp.TextResourceContents{
				URI:      fmt.Sprintf("codeforge://files/%s", filePath),
				MIMEType: detectMIMEType(filePath),
			}),
		),
		mcp.NewPromptMessage(
			mcp.RoleUser,
			mcp.NewTextContent("Please provide:\n1. Test cases covering normal scenarios\n2. Edge case tests\n3. Error condition tests\n4. Mock/stub suggestions if needed\n5. Test structure and organization recommendations"),
		),
	}

	return mcp.NewGetPromptResult(
		fmt.Sprintf("Testing assistance for %s", filePath),
		messages,
	), nil
}
