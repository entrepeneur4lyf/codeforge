# CodeForge

**A NOTE FROM THE REPO OWNER** - This project is a WIP and may not be fully functional. I want to thank all of the OSS contributors and project owners that gave me the opportunity to LEARN or even straight up *borrow* from them. I hope to continue to give back to the community as I grow as an AI based software architect - even though I have 30 years experience as a developer.

Thanks to: Cline, Pocketbase, OpenRouter, Turso, SST/Opencode. Opencode.ai/Opencode, Codex, Gemini CLI, Claude Code, AMP, Sourcegraph in general starting with Cody AND SO MANY MORE! Let's not steal from each other as developers. Give thanks where it is due. :)

<div align="center">

**🚀 AI-Powered Coding Assistant with Advanced Intelligence**

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![OpenRouter](https://img.shields.io/badge/OpenRouter-300%2B%20Models-orange)](https://openrouter.ai/)
[![Vector DB](https://img.shields.io/badge/Vector%20DB-LibSQL-green)](https://github.com/tursodatabase/libsql)

*Production-ready AI coding assistant with 300+ models and comprehensive LLM support*

</div>

## 📚 Overview

**CodeForge** is a soon-to-be production-ready (**WARNING**: currently WIP)  AI-powered coding assistant that provides intelligent, adaptive code assistance through comprehensive LLM support. With support for 300+ models from 50+ providers, smart database caching, and advanced code intelligence, CodeForge delivers enterprise-grade AI assistance for developers.

### � Key Highlights

- **🤖 300+ AI Models**: Access to models from Anthropic, OpenAI, Google, OpenRouter, and 50+ providers
- **⚡ 99% Performance Improvement**: Smart database caching reduces model sync from minutes to seconds
- **�️ Production Database**: LibSQL vector database with automatic TTL enforcement
- **🌐 Multiple Interfaces**: CLI and Model Context Protocol (MCP) support
- **� Semantic Search**: Vector-based code search with multi-dimensional embeddings
- **🏗️ Graph-Based Analysis**: Comprehensive codebase relationship mapping

## � Quick Start

### Installation

```bash
# Clone the repository
git clone https://github.com/your-org/codeforge.git
cd codeforge

# Build CodeForge CLI
go build -o codeforge ./cmd/codeforge

# Build API Server
go build -o codeforge-api ./cmd/codeforge-api

# Set up your API keys (optional - many features work without keys)
export OPENROUTER_API_KEY="your-key-here"
export ANTHROPIC_API_KEY="your-key-here"
export OPENAI_API_KEY="your-key-here"
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

## 🌟 Core Features

### 🤖 AI-Powered Coding Assistant
- **Multi-Provider LLM Support**: 25+ providers including Anthropic, OpenAI, Gemini, OpenRouter, Groq
- **Interactive Chat Interface**: Real-time streaming responses with conversation history
- **Direct Prompt Mode**: Single command execution with piped input support
- **Model Selection**: Dynamic model switching with provider-specific optimizations
- **API Key Management**: Automatic provider detection and fallback mechanisms

### 🧠 Advanced Code Intelligence
- **Semantic Code Search**: Vector-based similarity search with embedding generation
- **Symbol Extraction**: LSP-enhanced symbol analysis with fallback parsing
- **AST-Based Analysis**: Tree-sitter integration for precise code structure analysis
- **Code Chunking**: Multiple strategies (function, class, file, semantic, text-based)
- **Documentation Extraction**: Automatic extraction of comments and docstrings

### 🔧 Development Tools
- **Project Building**: Automated build system with error detection
- **LSP Integration**: Full Language Server Protocol support with multi-language clients
- **File Management**: Read/write operations with workspace awareness
- **Git Integration**: Repository status and change tracking
- **Error Pattern Recognition**: Learning from build failures and fixes

### 🗄️ Smart Codebase RAG
- **LibSQL Vector Integration**: Production-ready vector operations with native indexing
- **Multi-Dimensional Embeddings**: Support for 384-1536+ dimension vectors with optimized storage

### 🌐 Multi-Provider LLM Support

#### 🏢 Enterprise Providers
- **Anthropic**: Claude models with prompt caching and thinking modes
- **OpenAI**: GPT models with Azure support and O1/O3 reasoning
- **Google**: Gemini models with Vertex AI integration
- **AWS Bedrock**: Enterprise-grade model access
- **Azure OpenAI**: Microsoft cloud integration

#### ⚡ Performance Providers
- **Groq**: Ultra-fast inference with specialized hardware
- **Together AI**: Optimized model serving
- **Fireworks AI**: High-performance model hosting
- **Cerebras**: AI supercomputer integration
- **DeepSeek**: Advanced reasoning capabilities

#### 🌐 Multi-Provider Platforms
- **OpenRouter**: 300+ models with smart database caching and comprehensive metadata
- **LiteLLM**: Universal API compatibility
- **Ollama**: Local model execution
- **LM Studio**: Local model management

### 🌍 Model Context Protocol (MCP)

#### 🔧 MCP Tools
- **semantic_search**: Advanced semantic code search with vector similarity
- **read_file**: Workspace file reading with encoding detection
- **write_file**: Safe file writing with backup and validation
- **analyze_code**: Comprehensive code analysis with symbol extraction
- **get_project_structure**: Intelligent project structure mapping

#### 📚 MCP Resources
- **codeforge://project/metadata**: Project information and statistics
- **codeforge://files/{path}**: Direct file content access
- **codeforge://git/status**: Git repository status and changes

#### 💡 MCP Prompts
- **code_review**: Automated code review assistance
- **debug_help**: Intelligent debugging guidance
- **refactoring_guide**: Refactoring recommendations
- **documentation_help**: Documentation generation assistance
- **testing_help**: Test creation and improvement suggestions

### 🛠️ Language Support

#### 📝 Fully Supported Languages
- **Go**: Complete LSP integration, symbol extraction, and chunking
- **Rust**: Advanced parsing with tree-sitter integration
- **Python**: Comprehensive analysis with docstring extraction
- **JavaScript/TypeScript**: Modern JS/TS support with React patterns
- **Java**: Enterprise Java development support
- **C/C++**: System programming language support
- **PHP**: Web development language support

#### 🔧 Additional Language Features
- **Tree-Sitter Integration**: AST-based parsing for precise analysis
- **Language Detection**: Automatic language identification
- **Syntax Highlighting**: Rich syntax highlighting in web interface
- **LSP Client Management**: Per-language LSP server integration

### 🌐 Web API & Interface

#### � RESTful API (Port 47000)
- **Complete Provider Management**: Configure all 25+ LLM providers via API
- **Environment Variable Control**: Full CRUD operations for API keys and settings
- **Enhanced Edit/Delete**: Detailed feedback, validation, and security protection
- **Real-time Chat**: WebSocket-based chat with streaming responses
- **Server-Sent Events**: Live metrics and status updates
- **Project Management**: File browsing, search, and code analysis

#### 🔐 Secure Localhost Authentication
- **Session-Based Security**: Cryptographically secure tokens with salt protection
- **Session Hijacking Prevention**: IP + User-Agent binding with multi-factor validation
- **No TLS Required**: Enterprise-grade security for localhost development
- **Bearer Token Authentication**: Standard OAuth-style authentication flow

#### 🎨 Web Interface (Ready for Development)
- **Complete API Coverage**: All CodeForge features accessible via REST API
- **WebSocket Support**: Real-time communication for chat and updates
- **Provider Configuration**: Web-based setup for all LLM providers
- **Security Model**: Localhost-only access with enhanced authentication

## 📋 Command Line Interface

### 🎯 Available Commands

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

### 🚀 Advanced Usage Examples

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

## ⚙️ Configuration & Deployment

### 📋 Configuration Management
- **YAML Configuration**: Comprehensive configuration system
- **Environment Variables**: Flexible deployment options
- **Provider Settings**: Per-provider configuration and optimization
- **Workspace Management**: Multi-workspace support
- **State Persistence**: Configuration and state management

### 🚀 Deployment Options
- **Standalone CLI**: Direct command-line usage
- **MCP Server**: Model Context Protocol server mode
- **API Server**: RESTful API with WebSocket support (port 47000)
- **Multi-Transport**: stdio transport for MCP integration
- **Web Interface Ready**: Complete API for building web UIs

### 🔒 Security & Performance

#### 🛡️ Security Features
- **Enhanced Localhost Authentication**: Session-based security with salt protection
- **Session Hijacking Prevention**: Multi-factor validation (IP + User-Agent + Salt)
- **API Key Management**: Secure credential handling with masked display
- **Environment Variable Control**: Full CRUD with validation and critical variable protection
- **Enhanced Edit/Delete Operations**: Detailed feedback and security validation
- **Workspace Isolation**: Sandboxed workspace operations
- **Input Validation**: Comprehensive input sanitization
- **Error Handling**: Graceful error recovery and reporting

#### ⚡ Performance Optimizations
- **Smart Database Caching**: Two-table architecture with automatic cleanup triggers
- **Efficient Model Sync**: (318 models in seconds)
- **On-Demand Loading**: Comprehensive metadata fetched only when needed
- **Multi-Level Caching**: Database, memory, and API response caching
- **Concurrent Processing**: Thread-safe operations throughout
- **Memory Management**: Efficient memory usage and cleanup
- **Background Processing**: Non-blocking operations where possible
- **Performance Monitoring**: Built-in performance metrics and logging

## 🎯 Implementation Status

### ✅ **Fully Implemented**
- **CLI Interface**: Complete command-line interface with interactive and direct modes
- **MCP Server**: Full Model Context Protocol server implementation
- **RESTful API**: Complete API server with WebSocket and SSE support (port 47000)
- **Secure Authentication**: Enhanced localhost authentication with salt protection
- **Provider Management**: Complete API control of all 25+ LLM providers
- **Environment Variables**: Full CRUD operations with validation and security protection
- **OpenRouter Integration**: 300+ models with smart database caching and TTL enforcement
- **Multi-Provider LLM Support**: 25+ providers with automatic fallback
- **Vector Database**: LibSQL integration with semantic search
- **Code Intelligence**: Symbol extraction, chunking, and analysis
- **Language Support**: Go, Rust, Python, JavaScript, TypeScript, Java, C++, PHP

### 🚧 **In Development**
- **Web Interface**: Modern web UI (API foundation complete)
- **Additional Providers**: More LLM provider integrations

### 🔮 **Planned Features**
- **Docker Support**: Containerized deployment
- **Plugin System**: Extensible architecture
- **Team Collaboration**: Multi-user workspace support

## 🤝 Contributing

CodeForge is built for the developer community. We welcome contributions!

### 🛠️ Development Setup
```bash
git clone https://github.com/your-org/codeforge.git
cd codeforge
go mod download

# Build CLI interface
go build -o codeforge ./cmd/codeforge

# Build API server
go build -o codeforge-api ./cmd/codeforge-api
```

### 📋 Areas for Contribution
- **Web Interface Development**: Build modern web UI using the complete API
- **Provider Integrations**: Add support for new LLM providers
- **Language Support**: Extend language analysis capabilities
- **ML Algorithms**: Enhance machine learning features
- **API Enhancements**: Extend the RESTful API capabilities
- **Environment Management**: Enhance variable validation and security features
- **Security Features**: Enhance authentication and authorization
- **Documentation**: Improve docs and examples

## 📄 License

MIT License - see [LICENSE](LICENSE) for details.

---

<div align="center">

**CodeForge: Where AI meets intelligent code understanding** 🚀🧠

*Built with ❤️ for the developer community*

</div>
