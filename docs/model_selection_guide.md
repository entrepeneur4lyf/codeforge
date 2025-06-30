# CodeForge Model Selection Guide

## 🎯 Complete Model Selection Functionality

The CodeForge API provides comprehensive model selection and management capabilities with real-time provider detection, model filtering, and configuration management.

## 🔍 Model Discovery

### List All Available Models
```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:47000/api/v1/llm/models
```

**Response:**
```json
{
  "models": [
    {
      "id": "claude-3-5-sonnet-20241022",
      "name": "Claude 3.5 Sonnet",
      "provider": "anthropic",
      "description": "Most intelligent model for complex reasoning",
      "context_size": 200000,
      "input_cost": 3.0,
      "output_cost": 15.0,
      "capabilities": ["text", "code", "analysis", "reasoning"]
    },
    {
      "id": "gpt-4o",
      "name": "GPT-4o", 
      "provider": "openai",
      "description": "Multimodal flagship model",
      "context_size": 128000,
      "capabilities": ["text", "code", "vision", "audio"]
    }
  ],
  "total": 8
}
```

### Filter Models by Provider
```bash
# Anthropic models only
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:47000/api/v1/llm/models?provider=anthropic"

# OpenAI models only
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:47000/api/v1/llm/models?provider=openai"

# OpenRouter models only
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:47000/api/v1/llm/models?provider=openrouter"
```

### Get Provider-Specific Models
```bash
# Detailed Anthropic models
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:47000/api/v1/llm/models/anthropic

# Detailed OpenAI models
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:47000/api/v1/llm/models/openai
```

## 🏢 Provider Management

### List All Providers
```bash
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:47000/api/v1/llm/providers
```

**Response:**
```json
{
  "providers": [
    {
      "id": "anthropic",
      "name": "Anthropic",
      "description": "Claude models for advanced reasoning",
      "status": "configured",
      "model_count": 5,
      "last_updated": "2025-06-30T01:47:25Z"
    },
    {
      "id": "openai",
      "name": "OpenAI", 
      "description": "GPT models for general AI tasks",
      "status": "configured",
      "model_count": 8,
      "last_updated": "2025-06-30T01:47:25Z"
    },
    {
      "id": "ollama",
      "name": "Ollama",
      "description": "Local models for privacy and speed", 
      "status": "available",
      "model_count": 10,
      "last_updated": "2025-06-30T01:47:25Z"
    }
  ],
  "total": 6
}
```

### Provider Status Types
- **`configured`**: API key is set and provider is ready to use
- **`available`**: Provider is supported but not configured (no API key)
- **`unavailable`**: Provider service is not accessible

## ⚙️ Provider Configuration

### Set Provider as Default
```bash
curl -X PUT -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"enabled": true, "default": true}' \
  http://localhost:47000/api/v1/providers/anthropic
```

### Configure Provider Settings
```bash
curl -X PUT -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": true,
    "settings": {
      "model": "claude-3-5-sonnet-20241022",
      "temperature": 0.7,
      "max_tokens": 4096
    }
  }' \
  http://localhost:47000/api/v1/providers/anthropic
```

## 🔑 API Key Management

### Set API Keys
```bash
# Set Anthropic API key
curl -X PUT -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"value": "sk-ant-your-key-here"}' \
  http://localhost:47000/api/v1/environment/ANTHROPIC_API_KEY

# Set OpenAI API key
curl -X PUT -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"value": "sk-your-openai-key"}' \
  http://localhost:47000/api/v1/environment/OPENAI_API_KEY
```

### Check API Key Status
```bash
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:47000/api/v1/environment?category=llm"
```

## 🎛️ Model Selection Workflow

### 1. Check Available Providers
```bash
# See which providers are configured
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:47000/api/v1/llm/providers
```

### 2. Browse Models by Capability
```bash
# Find reasoning models
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:47000/api/v1/llm/models | \
  jq '.models[] | select(.capabilities[] | contains("reasoning"))'

# Find vision-capable models  
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:47000/api/v1/llm/models | \
  jq '.models[] | select(.capabilities[] | contains("vision"))'

# Find speed-optimized models
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:47000/api/v1/llm/models | \
  jq '.models[] | select(.capabilities[] | contains("speed"))'
```

### 3. Compare Model Costs
```bash
# Sort models by input cost
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:47000/api/v1/llm/models | \
  jq '.models | sort_by(.input_cost) | .[] | {name: .name, input_cost: .input_cost, output_cost: .output_cost}'
```

### 4. Select Model by Context Size
```bash
# Find models with large context windows
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:47000/api/v1/llm/models | \
  jq '.models[] | select(.context_size >= 100000) | {name: .name, context_size: .context_size}'
```

## 🔄 Dynamic Provider Detection

The API automatically detects provider availability based on:

### API Key Presence
- **Anthropic**: `ANTHROPIC_API_KEY`
- **OpenAI**: `OPENAI_API_KEY`
- **OpenRouter**: `OPENROUTER_API_KEY`
- **Gemini**: `GEMINI_API_KEY`
- **Groq**: `GROQ_API_KEY`

### Service Availability
- **Ollama**: Checks `http://localhost:11434` endpoint
- **Local Models**: Scans for available local model files

### Real-time Updates
- Provider status updates when API keys are added/removed
- Model counts refresh when providers are configured
- Automatic fallback to available providers

## 🌐 Web Interface Integration

### JavaScript Model Selector
```javascript
class ModelSelector {
  constructor(api) {
    this.api = api;
  }

  async getAvailableModels(filters = {}) {
    let url = '/llm/models';
    if (filters.provider) {
      url += `?provider=${filters.provider}`;
    }
    
    const response = await this.api.request(url);
    return response.models;
  }

  async getProviders() {
    const response = await this.api.request('/llm/providers');
    return response.providers;
  }

  async selectBestModel(requirements) {
    const models = await this.getAvailableModels();
    
    return models.filter(model => {
      if (requirements.capabilities) {
        return requirements.capabilities.every(cap => 
          model.capabilities.includes(cap)
        );
      }
      return true;
    }).sort((a, b) => {
      if (requirements.prioritize === 'cost') {
        return a.input_cost - b.input_cost;
      }
      if (requirements.prioritize === 'context') {
        return b.context_size - a.context_size;
      }
      return 0;
    })[0];
  }

  async configureProvider(providerId, settings) {
    return this.api.updateProvider(providerId, settings);
  }
}

// Usage
const selector = new ModelSelector(api);

// Find best reasoning model
const reasoningModel = await selector.selectBestModel({
  capabilities: ['reasoning', 'code'],
  prioritize: 'context'
});

// Configure Anthropic as default
await selector.configureProvider('anthropic', {
  enabled: true,
  default: true
});
```

## ✅ Functionality Status

### ✅ **Fully Working**
- **Provider Detection**: Real-time API key and service detection
- **Model Listing**: Complete model catalogs from all providers
- **Provider Filtering**: Filter models by provider, capability, cost
- **Configuration Management**: Set defaults, enable/disable providers
- **API Key Management**: Secure key storage and validation
- **Status Reporting**: Real-time provider and model availability

### 🔧 **Enhanced Features**
- **Cost Comparison**: Input/output pricing for all models
- **Capability Filtering**: Find models by specific capabilities
- **Context Size Filtering**: Filter by context window requirements
- **Real-time Updates**: Dynamic provider status updates
- **Secure Authentication**: Session-based API access

**The model selection functionality is fully operational with comprehensive provider management, real-time detection, and flexible filtering capabilities!** 🎯✅
