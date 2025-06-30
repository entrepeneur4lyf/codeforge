# Environment Variable Management API

## 🔧 Complete Environment Variable Control

The CodeForge API provides comprehensive management of environment variables with enhanced edit and delete functionality, including validation, security checks, and detailed feedback.

## 📋 List Environment Variables

### Get All Variables
```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:47000/api/v1/environment
```

### Get by Category
```bash
# LLM API keys only
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:47000/api/v1/environment?category=llm"

# Embedding settings
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:47000/api/v1/environment?category=embedding"

# Database configuration
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:47000/api/v1/environment?category=database"
```

## ✏️ Edit Individual Variables

### Set New API Key
```bash
curl -X PUT -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"value": "sk-ant-your-new-key-here"}' \
  http://localhost:47000/api/v1/environment/ANTHROPIC_API_KEY
```

**Response (New Variable):**
```json
{
  "success": true,
  "variable": "ANTHROPIC_API_KEY",
  "action": "created",
  "previous_value": "",
  "new_value": "sk-a...ere",
  "message": "Environment variable created successfully",
  "category": "llm"
}
```

### Update Existing API Key
```bash
curl -X PUT -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"value": "sk-ant-updated-key-here"}' \
  http://localhost:47000/api/v1/environment/ANTHROPIC_API_KEY
```

**Response (Updated Variable):**
```json
{
  "success": true,
  "variable": "ANTHROPIC_API_KEY",
  "action": "updated",
  "previous_value": "sk-a...ere",
  "new_value": "sk-a...ere",
  "message": "Environment variable updated successfully",
  "category": "llm"
}
```

### Set Configuration Variables
```bash
# Set embedding provider
curl -X PUT -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"value": "ollama"}' \
  http://localhost:47000/api/v1/environment/CODEFORGE_EMBEDDING_PROVIDER

# Set Ollama endpoint
curl -X PUT -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"value": "http://localhost:11434"}' \
  http://localhost:47000/api/v1/environment/OLLAMA_ENDPOINT

# Enable debug mode
curl -X PUT -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"value": "true"}' \
  http://localhost:47000/api/v1/environment/CODEFORGE_DEBUG
```

## 🗑️ Delete Variables

### Remove API Key
```bash
curl -X DELETE -H "Authorization: Bearer $TOKEN" \
  http://localhost:47000/api/v1/environment/ANTHROPIC_API_KEY
```

**Response (Successful Deletion):**
```json
{
  "success": true,
  "variable": "ANTHROPIC_API_KEY",
  "action": "deleted",
  "previous_value": "sk-a...ere",
  "message": "Environment variable removed successfully",
  "category": "llm",
  "note": "Variable removed from current session only. Update shell profile for persistence."
}
```

### Remove Multiple Variables
```bash
# Remove all OpenAI related variables
curl -X DELETE -H "Authorization: Bearer $TOKEN" \
  http://localhost:47000/api/v1/environment/OPENAI_API_KEY

curl -X DELETE -H "Authorization: Bearer $TOKEN" \
  http://localhost:47000/api/v1/environment/OPENAI_ORG_ID
```

## 🔒 Security & Validation

### Protected Variables
```bash
# Attempt to delete critical system variable (will fail)
curl -X DELETE -H "Authorization: Bearer $TOKEN" \
  http://localhost:47000/api/v1/environment/PATH
```

**Response (Forbidden):**
```json
{
  "error": "Cannot delete critical system variable",
  "status": 403
}
```

### Invalid Variable Names
```bash
# Attempt to set invalid variable (will fail)
curl -X PUT -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"value": "test"}' \
  http://localhost:47000/api/v1/environment/INVALID_VAR
```

**Response (Bad Request):**
```json
{
  "error": "Invalid environment variable name",
  "status": 400
}
```

### Variable Not Found
```bash
# Attempt to delete non-existent variable
curl -X DELETE -H "Authorization: Bearer $TOKEN" \
  http://localhost:47000/api/v1/environment/NONEXISTENT_KEY
```

**Response (Not Found):**
```json
{
  "error": "Environment variable not found",
  "status": 404
}
```

## 📊 Get Individual Variable Info

```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:47000/api/v1/environment/ANTHROPIC_API_KEY
```

**Response:**
```json
{
  "name": "ANTHROPIC_API_KEY",
  "value": "sk-a...xyz",
  "masked": true,
  "description": "API key for Anthropic Claude models",
  "required": false,
  "category": "llm"
}
```

## 🔄 Batch Operations

### Update Multiple Variables
```bash
curl -X PUT -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "variables": {
      "ANTHROPIC_API_KEY": "sk-ant-your-key",
      "OPENAI_API_KEY": "sk-your-openai-key",
      "OPENROUTER_API_KEY": "sk-or-your-key",
      "CODEFORGE_EMBEDDING_PROVIDER": "ollama",
      "CODEFORGE_DEBUG": "true"
    }
  }' \
  http://localhost:47000/api/v1/environment
```

**Response:**
```json
{
  "success": true,
  "updated": [
    "ANTHROPIC_API_KEY",
    "OPENAI_API_KEY", 
    "OPENROUTER_API_KEY",
    "CODEFORGE_EMBEDDING_PROVIDER",
    "CODEFORGE_DEBUG"
  ],
  "errors": [],
  "message": "Environment variables updated",
  "note": "Changes apply to current session only. For persistence, update your shell profile."
}
```

## 🎯 Supported Variables

### LLM Provider API Keys
- `ANTHROPIC_API_KEY` - Anthropic Claude models
- `OPENAI_API_KEY` - OpenAI GPT models  
- `OPENROUTER_API_KEY` - 300+ models via OpenRouter
- `GEMINI_API_KEY` - Google Gemini models
- `GROQ_API_KEY` - Groq ultra-fast inference
- `TOGETHER_API_KEY` - Together AI
- `FIREWORKS_API_KEY` - Fireworks AI
- `DEEPSEEK_API_KEY` - DeepSeek
- `COHERE_API_KEY` - Cohere
- `MISTRAL_API_KEY` - Mistral AI

### Configuration Variables
- `CODEFORGE_EMBEDDING_PROVIDER` - Default embedding provider
- `CODEFORGE_DEBUG` - Enable debug mode
- `CODEFORGE_LOG_LEVEL` - Log level (debug, info, warn, error)
- `CODEFORGE_DB_PATH` - Database file path
- `CODEFORGE_DATA_DIR` - Data directory path
- `OLLAMA_ENDPOINT` - Ollama server endpoint

### Protected System Variables
These cannot be deleted for security:
- `PATH`, `HOME`, `USER`, `SHELL`, `TERM`
- `PWD`, `OLDPWD`, `LANG`, `LC_ALL`
- `GOPATH`, `GOROOT`, `GOPROXY`

## 🌐 JavaScript Integration

```javascript
class EnvironmentManager {
  constructor(api) {
    this.api = api;
  }

  async setAPIKey(provider, key) {
    const varName = `${provider.toUpperCase()}_API_KEY`;
    return this.api.setEnvironmentVariable(varName, key);
  }

  async removeAPIKey(provider) {
    const varName = `${provider.toUpperCase()}_API_KEY`;
    return this.api.request(`/environment/${varName}`, { method: 'DELETE' });
  }

  async configureProvider(provider, config) {
    const updates = {};
    
    if (config.apiKey) {
      updates[`${provider.toUpperCase()}_API_KEY`] = config.apiKey;
    }
    
    if (config.endpoint) {
      updates[`${provider.toUpperCase()}_ENDPOINT`] = config.endpoint;
    }
    
    return this.api.updateEnvironmentVariables(updates);
  }

  async getProviderStatus() {
    const vars = await this.api.getEnvironmentVariables('llm');
    return vars.variables.reduce((status, variable) => {
      const provider = variable.name.replace('_API_KEY', '').toLowerCase();
      status[provider] = {
        configured: variable.value !== '',
        masked_value: variable.value
      };
      return status;
    }, {});
  }
}

// Usage
const envManager = new EnvironmentManager(api);

// Set up Anthropic
await envManager.setAPIKey('anthropic', 'sk-ant-your-key');

// Remove OpenAI
await envManager.removeAPIKey('openai');

// Configure Ollama
await envManager.configureProvider('ollama', {
  endpoint: 'http://localhost:11434'
});

// Check status
const status = await envManager.getProviderStatus();
console.log('Provider Status:', status);
```

## ✅ Benefits

### Enhanced Functionality
- **Complete CRUD Operations**: Create, Read, Update, Delete
- **Detailed Feedback**: Action type, previous/new values, categories
- **Security Validation**: Prevent deletion of critical variables
- **Masked Display**: Secure handling of sensitive data

### Developer Experience
- **Clear Error Messages**: Specific validation and error feedback
- **Batch Operations**: Update multiple variables efficiently
- **Category Filtering**: Organize variables by purpose
- **Session Persistence**: Changes apply immediately to current session

**Complete environment variable management with enterprise-grade security and validation!** 🔧🔒
