# CodeForge

**A NOTE FROM THE REPO OWNER** - This project is a WIP and may not be fully functional. I want to thank all of the OSS contributors and project owners that gave me the opportunity to LEARN or even straight up *borrow* from them. I hope to continue to give back to the community as I grow as an AI based software architect - even though I have 30 years experience as a developer.

Thanks to: Cline, Pocketbase, OpenRouter, Turso, SST/Opencode. Opencode.ai/Opencode, Codex, Gemini CLI, Claude Code, AMP, Sourcegraph in general starting with Cody AND SO MANY MORE! Let's not steal from each other as developers. Give thanks where it is due. :)

<div align="center">

**AI-Powered Coding Assistant with Advanced Intelligence**

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![OpenRouter](https://img.shields.io/badge/OpenRouter-300%2B%20Models-orange)](https://openrouter.ai/)
[![Vector DB](https://img.shields.io/badge/Vector%20DB-LibSQL-green)](https://github.com/tursodatabase/libsql)

*Production-ready AI coding assistant with 300+ models and comprehensive LLM support*

</div>

## Overview

**CodeForge** is a soon-to-be production-ready (**WARNING**: currently WIP) AI-powered coding assistant that provides intelligent, adaptive code assistance through comprehensive LLM support. With support for 20+ providers including official SDK integrations, vector-based semantic search, and robust MCP integration, CodeForge delivers enterprise-grade AI assistance for developers.

### Key Highlights

- **20+ AI Providers**: Official SDK integrations for Anthropic, OpenAI, Google, AWS, plus OpenRouter with 300+ models
- **Production-Ready Performance**: Thread-safe operations with multi-level caching and background processing
- **Vector Database**: LibSQL-based semantic search with 256-1536 dimension embeddings
- **Multiple Interfaces**: CLI, API server, web interface, and Model Context Protocol (MCP) support
- **Semantic Search**: Vector-based code search with intelligent context selection
- **Code Intelligence**: Graph-based codebase awareness with LSP and tree-sitter integration

## Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/entrepreneur4lyf/codeforge.git
cd codeforge

# Build CodeForge CLI
go build -o codeforge ./cmd/codeforge

# Build API Server
go build -o codeforge-api ./cmd/codeforge-api

# Set up your API keys (optional - many features work without keys)
export OPENROUTER_API_KEY="[REDACTED:api-key]"
export ANTHROPIC_API_KEY="[REDACTED:api-key]"
export OPENAI_API_KEY="[REDACTED:api-key]"
```

### Basic Usage

```bash
# Interactive chat mode
./codeforge

# Direct prompt execution
./codeforge "Explain this function"

# Use specific model
./codeforge -m claude-3-5-sonnet "Refactor this code"

# Pipe input
echo "How do I optimize this algorithm?" | ./codeforge

# Start MCP server
./codeforge mcp server

# Start API server for web interfaces
./codeforge-api

# Quick API usage
curl -X POST http://localhost:47000/api/v1/auth  # Get auth token
curl -H "Authorization: Bearer $TOKEN" http://localhost:47000/api/v1/providers  # List providers
curl -X PUT -H "Authorization: Bearer $TOKEN" -d '{"value":"sk-ant-key"}' \
  http://localhost:47000/api/v1/environment/ANTHROPIC_API_KEY  # Set API key
```

## Core Features

### AI-Powered Coding Assistant
- **Multi-Provider LLM Support**: 20+ providers with official SDK integrations (Anthropic, OpenAI, Google, AWS, OpenRouter, Groq, etc.)
- **Interactive Chat Interface**: Real-time streaming responses with conversation history via CLI and web interface
- **Direct Prompt Mode**: Single command execution with piped input support (`echo "question" | codeforge`)
- **Model Selection**: Interactive TUI model selector with favorites and provider filtering
- **API Key Management**: Environment variable-based configuration with automatic provider detection

### Advanced Code Intelligence
- **Semantic Code Search**: Vector-based similarity search using embeddings (Ollama/OpenAI/fallback)
- **Symbol Extraction**: LSP-enhanced symbol analysis with tree-sitter fallback parsing
- **AST-Based Analysis**: Tree-sitter integration for Go, Rust, Python, JavaScript/TypeScript, Java, C/C++, PHP
- **Code Chunking**: Multiple strategies (tree-sitter, function, class, file, text-based) with language-specific parsers
- **Documentation Extraction**: Automatic extraction of comments, docstrings, and code metadata

### Development Tools
- **Project Analysis**: Automatic project overview generation and AGENT.md creation
- **LSP Integration**: Language Server Protocol support with multi-language client management
- **File Management**: Read/write operations with workspace awareness and encoding detection
- **Git Integration**: Repository status tracking and change detection
- **Build System**: Project building with error detection and pattern learning

### Smart Codebase RAG
- **LibSQL Vector Integration**: Production-ready vector operations with JSON-based similarity search fallback
- **Multi-Dimensional Embeddings**: Support for 256-1536 dimension vectors (Ollama nomic-embed-text, OpenAI, hash fallback)
- **Hybrid Search**: Graph-based codebase awareness combined with vector similarity search
- **Intelligent Context**: Smart context selection for LLM interactions with relevance scoring

### Multi-Provider LLM Support

#### Enterprise Providers (Implemented)
- **Anthropic**: Claude models with official SDK integration (anthropic-sdk-go)
- **OpenAI**: GPT models with official SDK integration (openai-go)
- **Google**: Gemini models with official SDK integration (google.golang.org/genai)
- **AWS Bedrock**: Enterprise-grade model access with AWS SDK v2
- **Azure OpenAI**: Microsoft cloud integration via OpenAI SDK

#### Performance Providers (Implemented)
- **Groq**: Ultra-fast inference with API integration
- **Together AI**: Optimized model serving via API
- **Fireworks AI**: High-performance model hosting via API
- **Cerebras**: AI supercomputer integration via API
- **DeepSeek**: Advanced reasoning capabilities via API

#### Multi-Provider Platforms (Implemented)
- **OpenRouter**: 300+ models with official SDK integration and smart database caching
- **LiteLLM**: Universal API compatibility with OpenAI-compatible interface
- **Ollama**: Local model execution with embedding support (nomic-embed-text)
- **LM Studio**: Local model management via API

#### Additional Providers (Implemented)
- **xAI (Grok)**: Advanced reasoning via API integration
- **Mistral**: European AI with API integration
- **Cohere**: Enterprise AI platform via API
- **Perplexity**: Search-augmented AI via API

### Model Context Protocol (MCP)

#### MCP Tools (Fully Implemented)
- **semantic_search**: Vector-based semantic code search with embedding generation and similarity ranking
- **read_file**: Workspace file reading with encoding detection and path validation
- **write_file**: Safe file writing with backup creation and content validation
- **analyze_code**: Comprehensive code analysis with LSP symbol extraction and tree-sitter parsing
- **get_project_structure**: Directory tree generation with configurable depth limits

#### MCP Resources (Fully Implemented)
- **codeforge://project/metadata**: Project information including workspace root, version, and description
- **codeforge://files/{path}**: Direct file content access with MIME type detection
- **codeforge://git/status**: Git repository status and change tracking

#### MCP Prompts (Fully Implemented)
- **code_review**: Structured code review assistance with embedded file resources
- **debug_help**: Debugging guidance with context-aware suggestions
- **refactoring_guide**: Refactoring recommendations with step-by-step plans
- **documentation_help**: Documentation generation with usage examples
- **testing_help**: Test creation assistance with coverage recommendations

#### MCP Server Features
- **Multiple Transports**: stdio, HTTP, and Server-Sent Events (SSE) support
- **Permission System**: Permission-aware MCP server with session management and audit logging
- **Standalone Operation**: Independent MCP server (`codeforge mcp server`) or integrated mode

### Language Support

#### Fully Supported Languages
- **Go**: Complete LSP integration, symbol extraction, and chunking
- **Rust**: Advanced parsing with tree-sitter integration
- **Python**: Comprehensive analysis with docstring extraction
- **JavaScript/TypeScript**: Modern JS/TS support with React patterns
- **Java**: Enterprise Java development support
- **C/C++**: System programming language support
- **PHP**: Web development language support

#### Additional Language Features
- **Tree-Sitter Integration**: AST-based parsing for precise analysis
- **Language Detection**: Automatic language identification
- **Syntax Highlighting**: Rich syntax highlighting in web interface
- **LSP Client Management**: Per-language LSP server integration

### Web API & Interface

#### RESTful API (Port 47000) - Fully Implemented
- **Complete Provider Management**: Configure all 20+ LLM providers via API
- **Environment Variable Control**: Full CRUD operations for API keys and settings
- **Authentication System**: Token-based authentication with session management
- **Real-time Chat**: WebSocket-based chat with streaming responses
- **Server-Sent Events**: Live metrics and status updates
- **Project Management**: File browsing, search, and code analysis

#### Secure Localhost Authentication - Implemented
- **Session-Based Security**: Cryptographically secure tokens with session management
- **Permission System**: Session-based permission management with audit logging
- **No TLS Required**: Secure localhost development without certificate complexity
- **Bearer Token Authentication**: Standard OAuth-style authentication flow

#### Web Interface - Implemented
- **TUI-Style Interface**: Terminal-inspired web interface with dark theme and monospace fonts
- **File Browser**: Interactive file system navigation with project structure display
- **Code Editor**: Syntax highlighting with language detection and file content loading
- **Chat Interface**: Real-time AI conversation with message history and streaming responses
- **Complete API Coverage**: All CodeForge features accessible via REST API

## Command Line Interface

### Available Commands

**Core Commands:**
```bash
codeforge                    # Interactive chat mode
codeforge "prompt"           # Direct prompt execution
codeforge -m model "prompt"  # Specify model
echo "question" | codeforge  # Pipe input
```

**MCP Commands:**
```bash
codeforge mcp list    # List MCP capabilities
codeforge mcp server  # Start MCP server
```

**API Server Commands:**
```bash
codeforge-api                    # Start API server on port 47000
codeforge-api --port 8080        # Start on custom port
codeforge-api --debug            # Enable debug mode
```

### Advanced Usage Examples

```bash
# Interactive mode with codebase awareness
./codeforge

# Analyze specific files with context
./codeforge "Explain the authentication flow in auth.go"

# Use different models for different tasks
./codeforge -m claude-3-5-sonnet "Review this code for security issues"
./codeforge -m gpt-4 "Generate unit tests for this function"

# Pipe code for analysis
cat main.go | ./codeforge "Optimize this code"

# Start as MCP server for Claude Desktop
./codeforge mcp server

# Start API server for web interfaces
./codeforge-api

# Configure providers via API
curl -X PUT -H "Authorization: Bearer $TOKEN" \
  -d '{"value": "sk-ant-your-key"}' \
  http://localhost:47000/api/v1/environment/ANTHROPIC_API_KEY

# Remove API key
curl -X DELETE -H "Authorization: Bearer $TOKEN" \
  http://localhost:47000/api/v1/environment/OPENAI_API_KEY
```

## Configuration & Deployment

### Configuration Management (Implemented)
- **Environment Variables**: API key management via environment variables (ANTHROPIC_API_KEY, OPENAI_API_KEY, etc.)
- **Provider Settings**: Per-provider configuration with rate limiting, cost management, and health monitoring
- **Workspace Management**: Single workspace support with automatic project detection
- **Database Configuration**: SQLite-based configuration and state persistence

### Deployment Options (Implemented)
- **Standalone CLI**: Direct command-line usage with interactive and direct prompt modes
- **MCP Server**: Model Context Protocol server with stdio, HTTP, and SSE transport options
- **API Server**: Full REST API with authentication and WebSocket support (`codeforge-api`)
- **Web Interface**: Built-in web server with TUI-style interface
- **Multiple Binaries**: Separate binaries for CLI (`codeforge`) and API server (`codeforge-api`)

### Security & Performance

#### Security Features (Implemented)
- **API Key Management**: Environment variable-based secure credential handling
- **Workspace Isolation**: Path validation and workspace-relative file operations
- **Input Validation**: Comprehensive input sanitization and path validation
- **Permission System**: Session-based permission management with audit logging
- **Authentication**: Token-based authentication for API access with session management
- **Error Handling**: Graceful error recovery and reporting

#### Performance Optimizations (Implemented)
- **Database Caching**: SQLite-based caching with thread-safe operations using sync.Map
- **Background Processing**: Asynchronous model discovery and embedding generation
- **Concurrent Processing**: Thread-safe operations throughout with proper mutex usage
- **Memory Management**: Efficient memory usage with proper resource cleanup
- **Graceful Degradation**: Fallback mechanisms for all major features (embeddings, LSP, etc.)
- **Performance Monitoring**: Built-in timing and metrics (hidden from user interface)

## Implementation Status

### **Fully Implemented**
- **CLI Interface**: Complete command-line interface with interactive and direct modes
- **MCP Server**: Full Model Context Protocol server with 5 tools, 3 resources, and 5 prompts
- **RESTful API**: Complete API server with WebSocket and SSE support (port 47000)
- **Web Interface**: TUI-style web interface with file browser, code editor, and chat
- **Authentication**: Token-based authentication with session management
- **Provider Management**: Complete API control of all 20+ LLM providers with official SDKs
- **Environment Variables**: Full CRUD operations with validation and security protection
- **Multi-Provider LLM Support**: 20+ providers with automatic fallback and official SDK integrations
- **Vector Database**: LibSQL integration with semantic search and embedding generation
- **Code Intelligence**: Symbol extraction, chunking, and analysis with LSP and tree-sitter
- **Language Support**: Go, Rust, Python, JavaScript, TypeScript, Java, C++, PHP

### **In Development**
- **Additional Providers**: More LLM provider integrations
- **Enhanced Web Features**: Advanced web UI capabilities

### **Planned Features**
- **Docker Support**: Containerized deployment
- **Plugin System**: Extensible architecture
- **Team Collaboration**: Multi-user workspace support

## Contributing

CodeForge is built for the developer community. We welcome contributions!

### Development Setup
```bash
git clone https://github.com/entrepeneur4lyf/codeforge.git
cd codeforge
go mod download

# Build CLI interface
go build -o codeforge ./cmd/codeforge

# Build API server
go build -o codeforge-api ./cmd/codeforge-api
```

### Areas for Contribution
- **Web Interface Development**: Enhance the TUI-style web interface with additional features
- **Provider Integrations**: Add support for new LLM providers
- **Language Support**: Extend language analysis capabilities and tree-sitter integration
- **Code Intelligence**: Enhance semantic search and context generation algorithms
- **API Enhancements**: Extend the RESTful API capabilities
- **Environment Management**: Enhance variable validation and security features
- **Security Features**: Enhance authentication and authorization
- **Documentation**: Improve docs and examples

## License

MIT License - see [LICENSE](LICENSE) for details.

---

<div align="center">

**CodeForge: Where AI meets intelligent code understanding**

*Built with love for the developer community*

</div>
