# CodeForge Provider Management API

## 🔧 Complete Provider Control

The CodeForge API provides comprehensive management of all providers and settings that users can change through a secure localhost interface.

## 🤖 LLM Provider Management

### List All LLM Providers
```bash
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:47000/api/v1/providers?type=llm"
```

**Response:**
```json
{
  "providers": [
    {
      "id": "anthropic",
      "name": "Anthropic",
      "type": "llm",
      "enabled": true,
      "default": true,
      "settings": {
        "api_key": "sk-ant...xyz",
        "model": "claude-3-5-sonnet-20241022",
        "temperature": 0.7,
        "max_tokens": 4096
      },
      "status": "configured",
      "models": ["claude-3-5-sonnet-20241022", "claude-3-haiku-20240307"]
    },
    {
      "id": "openai",
      "name": "OpenAI", 
      "type": "llm",
      "enabled": false,
      "default": false,
      "settings": {
        "api_key": "",
        "model": "gpt-4o"
      },
      "status": "available"
    }
  ]
}
```

### Update LLM Provider Settings
```bash
curl -X PUT -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": true,
    "default": true,
    "settings": {
      "model": "claude-3-5-sonnet-20241022",
      "temperature": 0.8,
      "max_tokens": 8192
    }
  }' \
  http://localhost:47000/api/v1/providers/anthropic
```

## 🧠 Embedding Provider Management

### Get Current Embedding Provider
```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:47000/api/v1/providers/embedding
```

### Change Embedding Provider
```bash
# Switch to Ollama embeddings
curl -X PUT -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "ollama",
    "model": "nomic-embed-text",
    "dimensions": 768,
    "endpoint": "http://localhost:11434"
  }' \
  http://localhost:47000/api/v1/providers/embedding

# Switch to OpenAI embeddings
curl -X PUT -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "openai",
    "model": "text-embedding-3-small",
    "dimensions": 1536
  }' \
  http://localhost:47000/api/v1/providers/embedding
```

## 🔑 Environment Variable Management

### List All API Keys and Settings
```bash
# All environment variables
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:47000/api/v1/environment

# LLM API keys only
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:47000/api/v1/environment?category=llm"

# Embedding settings only
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:47000/api/v1/environment?category=embedding"
```

**Response:**
```json
{
  "variables": [
    {
      "name": "ANTHROPIC_API_KEY",
      "value": "sk-a...xyz",
      "masked": true,
      "description": "API key for Anthropic Claude models",
      "required": false,
      "category": "llm"
    },
    {
      "name": "OPENAI_API_KEY", 
      "value": "",
      "masked": true,
      "description": "API key for OpenAI GPT models",
      "required": false,
      "category": "llm"
    }
  ],
  "note": "Masked values show only first/last 4 characters for security"
}
```

### Set Individual API Keys
```bash
# Set Anthropic API key
curl -X PUT -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"value": "sk-ant-your-actual-key-here"}' \
  http://localhost:47000/api/v1/environment/ANTHROPIC_API_KEY

# Set OpenAI API key
curl -X PUT -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"value": "sk-your-openai-key-here"}' \
  http://localhost:47000/api/v1/environment/OPENAI_API_KEY

# Set OpenRouter API key (300+ models)
curl -X PUT -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"value": "sk-or-your-openrouter-key"}' \
  http://localhost:47000/api/v1/environment/OPENROUTER_API_KEY
```

**Response Example:**
```json
{
  "success": true,
  "variable": "ANTHROPIC_API_KEY",
  "action": "updated",
  "previous_value": "sk-a...xyz",
  "new_value": "sk-a...abc",
  "message": "Environment variable updated successfully",
  "category": "llm"
}
```

### Delete API Keys
```bash
# Remove Anthropic API key
curl -X DELETE -H "Authorization: Bearer $TOKEN" \
  http://localhost:47000/api/v1/environment/ANTHROPIC_API_KEY

# Remove OpenAI API key
curl -X DELETE -H "Authorization: Bearer $TOKEN" \
  http://localhost:47000/api/v1/environment/OPENAI_API_KEY
```

**Response Example:**
```json
{
  "success": true,
  "variable": "ANTHROPIC_API_KEY",
  "action": "deleted",
  "previous_value": "sk-a...xyz",
  "message": "Environment variable removed successfully",
  "category": "llm",
  "note": "Variable removed from current session only. Update shell profile for persistence."
}
```

### Batch Update Multiple Variables
```bash
curl -X PUT -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "variables": {
      "ANTHROPIC_API_KEY": "sk-ant-your-key",
      "OPENAI_API_KEY": "sk-your-openai-key",
      "OPENROUTER_API_KEY": "sk-or-your-key",
      "CODEFORGE_EMBEDDING_PROVIDER": "ollama",
      "OLLAMA_ENDPOINT": "http://localhost:11434"
    }
  }' \
  http://localhost:47000/api/v1/environment
```

## 🎯 Supported Providers

### LLM Providers (25+ supported)
- **Anthropic**: `ANTHROPIC_API_KEY`
- **OpenAI**: `OPENAI_API_KEY` 
- **OpenRouter**: `OPENROUTER_API_KEY` (300+ models)
- **Google**: `GEMINI_API_KEY`
- **Groq**: `GROQ_API_KEY`
- **Together AI**: `TOGETHER_API_KEY`
- **Fireworks**: `FIREWORKS_API_KEY`
- **DeepSeek**: `DEEPSEEK_API_KEY`
- **Cohere**: `COHERE_API_KEY`
- **Mistral**: `MISTRAL_API_KEY`
- **Ollama**: Local models (no API key needed)

### Embedding Providers
- **Ollama**: `nomic-embed-text` (768D), `all-minilm` (384D)
- **OpenAI**: `text-embedding-3-small` (1536D)
- **Fallback**: Hash-based embeddings (384D)

### Configuration Variables
- **Database**: `CODEFORGE_DB_PATH`, `CODEFORGE_DATA_DIR`
- **Embedding**: `CODEFORGE_EMBEDDING_PROVIDER`, `OLLAMA_ENDPOINT`
- **Development**: `CODEFORGE_DEBUG`, `CODEFORGE_LOG_LEVEL`

## 🔒 Security Features

### API Key Protection
- **Masked Display**: Only first/last 4 characters shown
- **Secure Storage**: Environment variables only
- **Localhost Only**: No external access to sensitive data
- **Session-based**: Changes apply to current session

### Validation
- **Provider Availability**: Check if services are running
- **API Key Format**: Validate key formats before setting
- **Model Compatibility**: Ensure models exist for providers
- **Dimension Matching**: Validate embedding dimensions

## 🌐 Web Interface Integration

### JavaScript Provider Manager
```javascript
class ProviderManager {
  constructor(api) {
    this.api = api;
  }

  async setupAnthropic(apiKey) {
    // Set API key
    await this.api.setEnvironmentVariable('ANTHROPIC_API_KEY', apiKey);
    
    // Enable and set as default
    await this.api.updateProvider('anthropic', {
      enabled: true,
      default: true,
      settings: {
        model: 'claude-3-5-sonnet-20241022',
        temperature: 0.7
      }
    });
  }

  async setupOpenRouter(apiKey) {
    await this.api.setEnvironmentVariable('OPENROUTER_API_KEY', apiKey);
    await this.api.updateProvider('openrouter', {
      enabled: true,
      settings: {
        model: 'anthropic/claude-3.5-sonnet'
      }
    });
  }

  async setupOllamaEmbeddings() {
    await this.api.setEmbeddingProvider('ollama', 'nomic-embed-text', 768);
    await this.api.setEnvironmentVariable('OLLAMA_ENDPOINT', 'http://localhost:11434');
  }

  async getProviderStatus() {
    const [llmProviders, embeddingProvider, envVars] = await Promise.all([
      this.api.getProviders('llm'),
      this.api.request('/providers/embedding'),
      this.api.getEnvironmentVariables('llm')
    ]);

    return {
      llm: llmProviders.providers,
      embedding: embeddingProvider,
      apiKeys: envVars.variables
    };
  }
}

// Usage
const manager = new ProviderManager(api);
await manager.setupAnthropic('sk-ant-your-key');
const status = await manager.getProviderStatus();
```

## 🎉 Benefits

### Complete Control
- **All 25+ LLM providers** configurable via API
- **All embedding providers** (Ollama, OpenAI, fallback)
- **All environment variables** that affect CodeForge
- **Real-time updates** without restart

### User-Friendly
- **Masked sensitive data** for security
- **Clear provider status** (available, configured, error)
- **Batch operations** for efficiency
- **Validation and error handling**

### Web Interface Ready
- **RESTful API** for easy integration
- **JSON responses** for web frameworks
- **Secure authentication** for localhost
- **Real-time provider switching**

**Everything a user can change in CodeForge is now accessible through the secure localhost API!** 🚀
