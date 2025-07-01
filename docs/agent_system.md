# CodeForge Agent & Prompt System

The CodeForge Agent & Prompt System provides a sophisticated framework for managing AI agents with specialized prompts and real-time event handling.

## Overview

The system consists of two main components:

1. **Prompt System** (`internal/llm/prompt/`) - Manages agent-specific prompts with context integration
2. **Agent Service** (`internal/llm/agent/`) - Orchestrates agent execution with event streaming

## Architecture

### Prompt System

The prompt system provides specialized prompts for different agent types:

- **Coder Agent** - Full-featured coding assistant with environment awareness
- **Title Agent** - Generates concise titles for conversations
- **Task Agent** - Handles specific task-oriented queries
- **Summarizer Agent** - Creates conversation summaries

#### Key Features

- **Provider-specific prompts** - Different prompts for OpenAI vs Anthropic models
- **Context integration** - Automatically includes project-specific context from configured paths
- **Environment awareness** - Includes working directory, git status, platform info
- **File size limits** - Prevents context overflow with 10KB file limits

#### Usage

```go
import "github.com/entrepeneur4lyf/codeforge/internal/llm/prompt"

// Get prompt for specific agent and provider
coderPrompt := prompt.GetAgentPrompt(config.AgentCoder, models.ProviderAnthropic)
titlePrompt := prompt.GetAgentPrompt(config.AgentTitle, models.ProviderOpenAI)
```

### Agent Service

The agent service manages the execution of AI agents with real-time event streaming.

#### Key Features

- **Multi-agent support** - Manages multiple agent types simultaneously
- **Session management** - Tracks active sessions and prevents conflicts
- **Event streaming** - Real-time events for text chunks, usage, errors
- **Model flexibility** - Dynamic model switching per agent
- **Context integration** - Seamless integration with prompt system

#### Agent Events

The system emits various event types:

```go
type AgentEventType string

const (
    AgentEventStarted    AgentEventType = "agent_started"
    AgentEventCompleted  AgentEventType = "agent_completed"
    AgentEventError      AgentEventType = "agent_error"
    AgentEventCancelled  AgentEventType = "agent_cancelled"
    AgentEventTextChunk  AgentEventType = "agent_text_chunk"
    AgentEventToolCall   AgentEventType = "agent_tool_call"
    AgentEventToolResult AgentEventType = "agent_tool_result"
    AgentEventUsage      AgentEventType = "agent_usage"
)
```

## Integration with CodeForge

### Chat Session Integration

The chat session has been enhanced to support agent-based processing:

```go
// Set up agent service for chat session
chatSession.SetAgentService(agentService, eventManager, sessionID)

// Process messages with agent system
response, err := chatSession.ProcessMessageWithAgent(userInput)
```

### Event System Integration

The agent service integrates with CodeForge's event system for:

- Real-time WebSocket updates
- Event persistence and replay
- Cross-component communication
- Audit logging

## Configuration

### Agent Configuration

```go
cfg := &config.Config{
    Agents: map[config.AgentName]config.Agent{
        config.AgentCoder: {
            Model:     models.ModelClaude35Sonnet,
            MaxTokens: 4096,
        },
        config.AgentTitle: {
            Model:     models.ModelGPT4o,
            MaxTokens: 100,
        },
    },
    ContextPaths: []string{
        "README.md",
        "docs/",
        ".codeforge/",
    },
}
```

### Provider Configuration

```go
cfg.Providers = map[models.ModelProvider]config.Provider{
    models.ProviderAnthropic: {
        APIKey:   os.Getenv("ANTHROPIC_API_KEY"),
        Disabled: false,
    },
    models.ProviderOpenAI: {
        APIKey:   os.Getenv("OPENAI_API_KEY"),
        Disabled: false,
    },
}
```

## Usage Examples

### Basic Agent Execution

```go
// Create agent service
agentService, err := agent.NewAgentService(cfg, eventManager)
if err != nil {
    return err
}

// Run agent
ctx := context.Background()
sessionID := "user-session-123"
eventChan, err := agentService.Run(ctx, sessionID, config.AgentCoder, "Write a hello world function")

// Process events
for event := range eventChan {
    switch event.Type {
    case agent.AgentEventTextChunk:
        fmt.Print(event.Data["text"])
    case agent.AgentEventCompleted:
        fmt.Println("Agent completed")
    case agent.AgentEventError:
        fmt.Printf("Error: %s\n", event.Data["error"])
    }
}
```

### Dynamic Agent Configuration

```go
// Update agent model
newModel, err := agentService.UpdateAgent(config.AgentCoder, models.ModelGPT4o)
if err != nil {
    return err
}

// Check current model
model, err := agentService.GetModel(config.AgentCoder)
```

### Session Management

```go
// Check if session is busy
if agentService.IsSessionBusy(sessionID) {
    return errors.New("session already running")
}

// Cancel running agent
agentService.Cancel(sessionID)

// Check if any agent is running
if agentService.IsBusy() {
    fmt.Println("Agents are currently running")
}
```

## Context Management

The prompt system automatically includes context from configured paths:

1. **File Processing** - Reads files up to 10KB limit
2. **Directory Traversal** - Recursively processes directories
3. **Deduplication** - Prevents duplicate file processing
4. **Concurrent Processing** - Uses goroutines for performance
5. **Error Handling** - Gracefully handles file access errors

### Context Paths Configuration

```go
cfg.ContextPaths = []string{
    "README.md",           // Single file
    "docs/",              // Directory (recursive)
    ".codeforge/",        // Hidden directory
    "src/main.go",        // Specific source file
}
```

## Error Handling

The system provides comprehensive error handling:

- **Agent initialization errors** - Invalid models or missing API keys
- **Runtime errors** - Network issues, API errors, context cancellation
- **Session conflicts** - Preventing multiple agents per session
- **Context errors** - File access issues, size limits

## Performance Considerations

- **Concurrent execution** - Multiple agents can run simultaneously
- **Event buffering** - Configurable buffer sizes for event channels
- **Context caching** - One-time context gathering per session
- **Resource cleanup** - Automatic session cleanup and channel closing

## Future Enhancements

- **Tool integration** - Support for agent tool calls
- **Streaming improvements** - Enhanced streaming with backpressure
- **Agent chaining** - Sequential agent execution
- **Custom prompts** - User-defined prompt templates
- **Agent metrics** - Performance and usage analytics
