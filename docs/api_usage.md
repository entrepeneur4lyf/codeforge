# CodeForge API Usage Examples

## 🔐 Secure Localhost Authentication

The CodeForge API provides secure authentication for localhost connections without requiring TLS certificates.

### Base URL
```
http://localhost:47000/api/v1
```

## 🚀 Getting Started

### 1. Check API Health
```bash
curl http://localhost:47000/api/v1/health
```

### 2. Get Authentication Info
```bash
curl http://localhost:47000/api/v1/auth/info
```

### 3. Login (Get Token)
```bash
curl -X POST http://localhost:47000/api/v1/auth \
  -H "Content-Type: application/json" \
  -d '{"device_name": "My Development Machine"}'
```

Response:
```json
{
  "success": true,
  "token": "a1b2c3d4e5f6...",
  "session_id": "session-20240630-123456",
  "expires_at": "2024-07-01T12:34:56Z",
  "message": "Authentication successful for localhost"
}
```

### 4. Use Token for Protected Endpoints
```bash
# Set your token
TOKEN="your-token-here"

# Access protected endpoints
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:47000/api/v1/chat/sessions

# List all providers
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:47000/api/v1/providers

# Update Anthropic provider
curl -X PUT -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"enabled": true, "default": true}' \
  http://localhost:47000/api/v1/providers/anthropic

# Set API key
curl -X PUT -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"value": "sk-ant-your-key-here"}' \
  http://localhost:47000/api/v1/environment/ANTHROPIC_API_KEY

# Delete API key
curl -X DELETE -H "Authorization: Bearer $TOKEN" \
  http://localhost:47000/api/v1/environment/ANTHROPIC_API_KEY

# Change embedding provider
curl -X PUT -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"provider": "ollama", "model": "nomic-embed-text", "dimensions": 768}' \
  http://localhost:47000/api/v1/providers/embedding
```

## 📡 API Endpoints

### Authentication (Public)
- `GET /auth/info` - Authentication system information
- `POST /auth` - Login and get token
- `GET /auth` - Check authentication status
- `DELETE /auth` - Logout

### Chat (Protected)
- `GET /chat/sessions` - List chat sessions
- `POST /chat/sessions` - Create new session
- `GET /chat/sessions/{id}` - Get session details
- `DELETE /chat/sessions/{id}` - Delete session
- `GET /chat/sessions/{id}/messages` - Get messages
- `POST /chat/sessions/{id}/messages` - Send message

### WebSocket Chat (Protected)
```javascript
// Connect with token in URL
const ws = new WebSocket('ws://localhost:47000/api/v1/chat/ws/session-123?token=your-token');

// Send message
ws.send(JSON.stringify({
  type: 'chat_message',
  data: { message: 'Hello, CodeForge!' },
  event_id: 'msg-001'
}));
```

### Server-Sent Events (Protected)
```javascript
// Metrics stream
const metrics = new EventSource('http://localhost:47000/api/v1/events/metrics', {
  headers: { 'Authorization': 'Bearer your-token' }
});

metrics.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Metrics:', data);
};

// Status stream
const status = new EventSource('http://localhost:47000/api/v1/events/status', {
  headers: { 'Authorization': 'Bearer your-token' }
});
```

### Project Management (Protected)
- `GET /project/structure` - Get project file structure
- `GET /project/files` - List project files
- `POST /project/search` - Search project

### Code Analysis (Protected)
- `POST /code/analyze` - Analyze code
- `POST /code/symbols` - Extract symbols

### LLM Integration (Protected)
- `GET /llm/providers` - List LLM providers
- `GET /llm/models` - List all models
- `GET /llm/models/{provider}` - Get provider models

### Provider Management (Protected)
- `GET /providers` - List all providers (LLM, embedding)
- `GET /providers?type=llm` - List LLM providers only
- `GET /providers?type=embedding` - List embedding providers only
- `GET /providers/{id}` - Get specific provider configuration
- `PUT /providers/{id}` - Update provider settings
- `DELETE /providers/{id}` - Disable provider
- `GET /providers/embedding` - Get current embedding provider
- `PUT /providers/embedding` - Change embedding provider

### Environment Variables (Protected)
- `GET /environment` - List all environment variables
- `GET /environment?category=llm` - List LLM API keys
- `GET /environment?category=embedding` - List embedding settings
- `PUT /environment` - Update multiple environment variables
- `GET /environment/{name}` - Get specific environment variable
- `PUT /environment/{name}` - Set specific environment variable
- `DELETE /environment/{name}` - Remove environment variable

### Configuration (Protected)
- `GET /config` - Get current configuration
- `PUT /config` - Update configuration

## 🔒 Security Features

### Localhost-Only Access
- ✅ **IP Validation**: Only localhost/loopback addresses allowed
- ✅ **Origin Checking**: WebSocket connections restricted to localhost origins
- ✅ **No TLS Required**: Secure for local development without certificates

### Token Security
- ✅ **Cryptographically Secure**: 256-bit random tokens
- ✅ **SHA-256 Hashing**: Tokens hashed before storage
- ✅ **Automatic Expiration**: 24-hour session timeout
- ✅ **Session Management**: Track and invalidate sessions

### Request Validation
- ✅ **Bearer Token**: Standard Authorization header
- ✅ **Query Parameter**: Fallback for WebSocket connections
- ✅ **Session Context**: User session available in request context

## 🌐 Web Interface Integration

### JavaScript Example
```javascript
class CodeForgeAPI {
  constructor(baseURL = 'http://localhost:47000/api/v1') {
    this.baseURL = baseURL;
    this.token = localStorage.getItem('codeforge_token');
  }

  async login(deviceName = 'Web Browser') {
    const response = await fetch(`${this.baseURL}/auth`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ device_name: deviceName })
    });
    
    const data = await response.json();
    if (data.success) {
      this.token = data.token;
      localStorage.setItem('codeforge_token', this.token);
    }
    return data;
  }

  async request(endpoint, options = {}) {
    const headers = {
      'Content-Type': 'application/json',
      ...options.headers
    };

    if (this.token) {
      headers.Authorization = `Bearer ${this.token}`;
    }

    const response = await fetch(`${this.baseURL}${endpoint}`, {
      ...options,
      headers
    });

    return response.json();
  }

  async getChatSessions() {
    return this.request('/chat/sessions');
  }

  async sendMessage(sessionId, message) {
    return this.request(`/chat/sessions/${sessionId}/messages`, {
      method: 'POST',
      body: JSON.stringify({ message })
    });
  }

  // Provider Management
  async getProviders(type = 'all') {
    return this.request(`/providers?type=${type}`);
  }

  async updateProvider(providerId, settings) {
    return this.request(`/providers/${providerId}`, {
      method: 'PUT',
      body: JSON.stringify(settings)
    });
  }

  async setEmbeddingProvider(provider, model, dimensions) {
    return this.request('/providers/embedding', {
      method: 'PUT',
      body: JSON.stringify({ provider, model, dimensions })
    });
  }

  // Environment Variables
  async getEnvironmentVariables(category = 'all') {
    return this.request(`/environment?category=${category}`);
  }

  async setEnvironmentVariable(name, value) {
    return this.request(`/environment/${name}`, {
      method: 'PUT',
      body: JSON.stringify({ value })
    });
  }

  async updateEnvironmentVariables(variables) {
    return this.request('/environment', {
      method: 'PUT',
      body: JSON.stringify({ variables })
    });
  }

  connectWebSocket(sessionId) {
    const ws = new WebSocket(
      `ws://localhost:47000/api/v1/chat/ws/${sessionId}?token=${this.token}`
    );
    return ws;
  }
}

// Usage Examples
const api = new CodeForgeAPI();

// 1. Authentication
await api.login('My Web App');

// 2. Chat Management
const sessions = await api.getChatSessions();
await api.sendMessage('session-123', 'Hello CodeForge!');

// 3. Provider Management
const providers = await api.getProviders('llm');
await api.updateProvider('anthropic', {
  enabled: true,
  default: true,
  settings: { model: 'claude-3-5-sonnet-20241022' }
});

// 4. Environment Variables
await api.setEnvironmentVariable('ANTHROPIC_API_KEY', 'sk-ant-...');
await api.updateEnvironmentVariables({
  'OPENAI_API_KEY': 'sk-...',
  'OPENROUTER_API_KEY': 'sk-or-...'
});

// 5. Embedding Provider
await api.setEmbeddingProvider('ollama', 'nomic-embed-text', 768);
```

## 🎯 Benefits

### For Web Interfaces
- **No CORS Issues**: Proper CORS headers for localhost
- **Secure Tokens**: Cryptographically secure authentication
- **Real-time Communication**: WebSocket + SSE support
- **Session Management**: Automatic cleanup and expiration

### For Development
- **No TLS Setup**: Works immediately on localhost
- **Standard Auth**: Bearer token authentication
- **Multiple Interfaces**: REST + WebSocket + SSE
- **Easy Integration**: Simple token-based auth flow

The API is now ready for building web interfaces, desktop applications, or any other client that needs to interact with CodeForge!
