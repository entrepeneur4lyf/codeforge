# CodeForge Features

## 📚 Introduction

**ATTN**: This project is a WIP and may not be fully functional.

This document outlines the **actual implemented features** of CodeForge based on comprehensive code analysis.

## 🚀 Core Features

### 🤖 AI-Powered Coding Assistant
- **Multi-Provider LLM Support**: 20+ providers including Anthropic, OpenAI, Gemini, OpenRouter, Groq, DeepSeek, Together, Fireworks, Cerebras, Mistral, XAI, Ollama, LM Studio, and more
- **Interactive Chat Interface**: Real-time streaming responses with conversation history via CLI and web interface
- **Direct Prompt Mode**: Single command execution with piped input support (`echo "question" | codeforge`)
- **Model Selection**: Interactive TUI model selector with favorites and provider filtering
- **API Key Management**: Environment variable-based configuration with automatic provider detection

### 🧠 Advanced Code Intelligence
- **Semantic Code Search**: Vector-based similarity search using embeddings (Ollama/OpenAI/fallback)
- **Symbol Extraction**: LSP-enhanced symbol analysis with tree-sitter fallback parsing
- **AST-Based Analysis**: Tree-sitter integration for Go, Rust, Python, JavaScript/TypeScript, Java, C/C++, PHP
- **Code Chunking**: Multiple strategies (tree-sitter, function, class, file, text-based) with language-specific parsers
- **Documentation Extraction**: Automatic extraction of comments, docstrings, and code metadata

### 🔧 Development Tools
- **Project Analysis**: Automatic project overview generation and AGENT.md creation
- **LSP Integration**: Language Server Protocol support with multi-language client management
- **File Management**: Read/write operations with workspace awareness and encoding detection
- **Git Integration**: Repository status tracking and change detection
- **Build System**: Project building with error detection and pattern learning

## 🎯 Code Intelligence Features

### 🧠 Intelligent Code Analysis
- **Hybrid Search**: Graph-based codebase awareness combined with vector similarity search
- **Context Generation**: Intelligent context selection for LLM interactions with multi-factor scoring
- **Pattern Recognition**: Error pattern learning and recognition for debugging assistance
- **Smart Selection**: Advanced message scoring and greedy selection optimization
- **Semantic Expansion**: Multi-layered semantic expansion with weighted relationships

### 📊 Performance Optimization
- **Caching System**: Multi-level caching (database, memory, API response) with TTL management
- **Background Processing**: Non-blocking operations with concurrent processing
- **Memory Management**: Efficient memory usage with automatic cleanup
- **Performance Monitoring**: Built-in metrics and timing analysis (hidden from UI)

## 🗄️ Vector Database System

### 📚 Semantic Storage
- **LibSQL Vector Integration**: Production-ready vector operations with JSON-based similarity search fallback
- **Multi-Dimensional Embeddings**: Support for 256-1536 dimension vectors (Ollama nomic-embed-text, OpenAI, hash fallback)
- **Caching System**: Thread-safe caching with sync.Map for frequently accessed code chunks
- **Hybrid Search**: Vector similarity combined with metadata filtering and text-based search
- **Metadata Enrichment**: Rich metadata storage including symbols, imports, and chunk relationships

### 🔍 Search Capabilities
- **Cosine Similarity Search**: Mathematical similarity calculation with configurable result limits
- **Language Filtering**: Language-specific search and filtering by programming language
- **Chunk Type Filtering**: Search by code structure (function, class, module, etc.)
- **Error Pattern Recognition**: Specialized search for error patterns and debugging assistance

### 🚀 Model Management
- **Database-First Architecture**: Model information stored in SQLite with proper indexing
- **Background Fetching**: Asynchronous model discovery and caching for major providers
- **Provider Detection**: Automatic provider type detection based on model IDs and API keys
- **Fallback Mechanisms**: Graceful degradation when providers are unavailable
- **Performance Optimization**: Efficient model lookup and caching strategies

## 🌐 Model Context Protocol (MCP)

### 🔧 MCP Tools (Fully Implemented)
- **semantic_search**: Vector-based semantic code search with embedding generation and similarity ranking
- **read_file**: Workspace file reading with encoding detection and path validation
- **write_file**: Safe file writing with backup creation and content validation
- **analyze_code**: Comprehensive code analysis with LSP symbol extraction and tree-sitter parsing
- **get_project_structure**: Directory tree generation with configurable depth limits

### 📚 MCP Resources (Fully Implemented)
- **codeforge://project/metadata**: Project information including workspace root, version, and description
- **codeforge://files/{path}**: Direct file content access with MIME type detection
- **codeforge://git/status**: Git repository status and change tracking

### 💡 MCP Prompts (Fully Implemented)
- **code_review**: Structured code review assistance with embedded file resources
- **debug_help**: Debugging guidance with context-aware suggestions
- **refactoring_guide**: Refactoring recommendations with step-by-step plans
- **documentation_help**: Documentation generation with usage examples
- **testing_help**: Test creation assistance with coverage recommendations

### 🚀 MCP Server Features
- **Multiple Transports**: stdio, HTTP, and Server-Sent Events (SSE) support
- **Permission System**: Permission-aware MCP server with session management and audit logging
- **Standalone Operation**: Independent MCP server (`codeforge mcp server`) or integrated mode

## 🌍 Multi-Provider LLM Support

### 🏢 Enterprise Providers (Implemented)
- **Anthropic**: Claude models with official SDK integration (anthropic-sdk-go)
- **OpenAI**: GPT models with official SDK integration (openai-go)
- **Google**: Gemini models with official SDK integration (google.golang.org/genai)
- **AWS Bedrock**: Enterprise-grade model access with AWS SDK v2
- **Azure OpenAI**: Microsoft cloud integration via OpenAI SDK

### ⚡ Performance Providers (Implemented)
- **Groq**: Ultra-fast inference with API integration
- **Together AI**: Optimized model serving via API
- **Fireworks AI**: High-performance model hosting via API
- **Cerebras**: AI supercomputer integration via API
- **DeepSeek**: Advanced reasoning capabilities via API

### 🌐 Multi-Provider Platforms (Implemented)
- **OpenRouter**: 300+ models with official SDK integration and smart database caching
- **LiteLLM**: Universal API compatibility with OpenAI-compatible interface
- **Ollama**: Local model execution with embedding support (nomic-embed-text)
- **LM Studio**: Local model management via API

### 🚀 Specialized Providers (Implemented)
- **xAI (Grok)**: Advanced reasoning via API integration
- **Mistral**: European AI with API integration
- **Qwen**: Alibaba's multilingual models via API
- **Cohere**: Enterprise AI platform via API
- **Perplexity**: Search-augmented AI via API

### 🏭 Additional Providers (Implemented)
- **Doubao (ByteDance)**: Regional provider support
- **Sambanova**: Enterprise AI solutions via API
- **Nebius**: Cloud-native AI platform
- **Replicate**: Model hosting platform via API

## 🏗️ Graph-Based Codebase Awareness

### 🕸️ Code Graph System (Implemented)
- **Relationship Mapping**: Function calls, imports, and dependencies tracked via symbol extraction
- **File Watching**: Real-time codebase change detection with filesystem watchers and debounced updates
- **Hybrid Search**: Graph traversal combined with vector similarity search for enhanced code discovery
- **Context Generation**: Intelligent context selection for LLM interactions with relevance scoring
- **Performance Optimization**: Efficient graph operations with caching and concurrent processing

### 📊 Codebase Analytics (Implemented)
- **Symbol Extraction**: Comprehensive symbol analysis using LSP and tree-sitter parsers
- **Import Analysis**: Dependency tracking and relationship mapping across files
- **Code Structure**: Function, class, and module relationship analysis
- **Project Overview**: Automatic project analysis and AGENT.md generation

## 🎨 Web Interface

### 🖥️ Modern Web UI (Implemented)
- **TUI-Style Interface**: Terminal-inspired web interface with dark theme and monospace fonts
- **File Browser**: Interactive file system navigation with project structure display
- **Code Editor**: Syntax highlighting with language detection and file content loading
- **Chat Interface**: Real-time AI conversation with message history and streaming responses
- **Project Management**: Workspace awareness and project file management

### 🔌 API Endpoints (Fully Implemented)
- **RESTful API**: Complete programmatic access with authentication and CORS support
- **WebSocket Support**: Real-time chat communication and notifications
- **Server-Sent Events**: Live metrics and status updates
- **File Operations**: Read, write, and project structure access
- **Search API**: Semantic code search and project analysis
- **Provider Management**: LLM provider configuration and model selection
- **Authentication**: Token-based authentication with session management

## 🛠️ Language Support

### 📝 Fully Supported Languages (Implemented)
- **Go**: Complete LSP integration, symbol extraction, function chunking, and build system support
- **Rust**: Advanced parsing with tree-sitter integration and function-based chunking
- **Python**: Comprehensive analysis with docstring extraction and function parsing
- **JavaScript/TypeScript**: Modern JS/TS support with function detection and React patterns
- **Java**: Enterprise Java development support with class and method parsing
- **C/C++**: System programming language support with function extraction
- **PHP**: Web development support with function and class parsing

### 🔧 Additional Language Features (Implemented)
- **Tree-Sitter Integration**: AST-based parsing for precise code structure analysis
- **Language Detection**: Automatic language identification by file extension and content
- **Symbol Extraction**: LSP-enhanced symbol analysis with tree-sitter fallback
- **Code Chunking**: Language-specific chunking strategies for optimal code segmentation

## ⚙️ Configuration & Deployment

### 📋 Configuration Management (Implemented)
- **Environment Variables**: API key management via environment variables (ANTHROPIC_API_KEY, OPENAI_API_KEY, etc.)
- **Provider Settings**: Per-provider configuration with rate limiting, cost management, and health monitoring
- **Workspace Management**: Single workspace support with automatic project detection
- **Database Configuration**: SQLite-based configuration and state persistence

### 🚀 Deployment Options (Implemented)
- **Standalone CLI**: Direct command-line usage with interactive and direct prompt modes
- **MCP Server**: Model Context Protocol server with stdio, HTTP, and SSE transport options
- **API Server**: Full REST API with authentication and WebSocket support (`codeforge-api`)
- **Web Interface**: Built-in web server with TUI-style interface
- **Multiple Binaries**: Separate binaries for CLI (`codeforge`) and API server (`codeforge-api`)

## 🔒 Security & Performance

### 🛡️ Security Features (Implemented)
- **API Key Management**: Environment variable-based secure credential handling
- **Workspace Isolation**: Path validation and workspace-relative file operations
- **Input Validation**: Comprehensive input sanitization and path validation
- **Permission System**: Session-based permission management with audit logging
- **Authentication**: Token-based authentication for API access with session management

### ⚡ Performance Optimizations (Implemented)
- **Database Caching**: SQLite-based caching with thread-safe operations using sync.Map
- **Background Processing**: Asynchronous model discovery and embedding generation
- **Concurrent Processing**: Thread-safe operations throughout with proper mutex usage
- **Memory Management**: Efficient memory usage with proper resource cleanup
- **Graceful Degradation**: Fallback mechanisms for all major features (embeddings, LSP, etc.)
- **Performance Monitoring**: Built-in timing and metrics (hidden from user interface)

## 📊 Usage Examples

### 🖥️ CLI Usage (Verified Commands)
```bash
# Interactive mode with model selector
./codeforge

# Direct prompt mode
./codeforge "Explain this function"

# Pipe input for code analysis
cat main.go | ./codeforge "Optimize this code"

# Model selection with flags
./codeforge -m claude-3-5-sonnet "Review this code"
./codeforge --model gpt-4o "Generate unit tests"

# Provider selection
./codeforge -p anthropic "Generate tests"

# Quiet mode and format options
./codeforge --quiet --format json "Analyze project structure"

# Debug mode
./codeforge --debug "Debug this error"
```

### 🌐 MCP Integration (Verified)
```bash
# Start MCP server (stdio transport)
./codeforge mcp server

# Start MCP server with HTTP transport
./codeforge mcp server --transport http --port 3000

# Use with Claude Desktop - add to claude_desktop_config.json:
{
  "mcpServers": {
    "codeforge": {
      "command": "/path/to/codeforge",
      "args": ["mcp", "server"]
    }
  }
}
```

### 🔌 API Usage (Verified Endpoints)
```bash
# Start API server
./codeforge-api --port 47000

# Authentication
curl -X POST http://localhost:47000/api/v1/auth \
  -H "Content-Type: application/json" \
  -d '{"client_name": "My App"}'

# Chat with AI (requires authentication)
curl -X POST http://localhost:47000/api/v1/chat/sessions/session-123/messages \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello CodeForge!"}'

# Get providers and models
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:47000/api/v1/llm/providers

# Project structure
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:47000/api/v1/project/structure
```

## 🎯 Performance Highlights

- **Production-Ready**: Comprehensive error handling and graceful degradation across all features
- **Thread-Safe**: Concurrent operations with proper mutex usage and sync.Map caching
- **Efficient Caching**: Multi-level caching (database, memory, API) with TTL management
- **Background Processing**: Asynchronous model discovery and embedding generation
- **Graceful Fallback**: Robust fallback mechanisms (embeddings, LSP, vector search)
- **Memory Optimized**: Efficient resource management with automatic cleanup
- **Fast Vector Search**: Cosine similarity with JSON fallback when native indexing unavailable
- **Smart Context**: Intelligent context selection and relevance scoring for LLM interactions

## 🚀 Provider Integration Highlights

### 📊 Multi-Provider Architecture
- **Official SDK Integration**: Uses official SDKs for Anthropic, OpenAI, Google, AWS, and OpenRouter
- **Provider Detection**: Automatic provider type detection based on model IDs and API keys
- **Fallback Mechanisms**: Graceful degradation when providers are unavailable
- **Model Caching**: Database-based model information storage with background fetching
- **Rate Limiting**: Built-in rate limiting and cost management per provider

### ⚡ Performance Features
- **Background Model Discovery**: Asynchronous model fetching and caching
- **Database-First**: SQLite-based model storage with proper indexing
- **Smart Caching**: TTL-based cache refresh with background updates
- **Concurrent Processing**: Thread-safe operations with proper synchronization
- **Memory Efficient**: Optimized memory usage with automatic resource cleanup

## 🎯 Summary

CodeForge is a production-ready AI coding assistant that combines semantic code intelligence with comprehensive multi-provider LLM support, vector-based search capabilities, and robust MCP integration to provide intelligent, context-aware code assistance through CLI, API, and web interfaces.
