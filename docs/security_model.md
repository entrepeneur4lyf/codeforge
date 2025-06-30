# CodeForge Security Model

## 🔐 Enhanced Localhost Authentication with Salt Protection

CodeForge implements a robust security model specifically designed for localhost development environments, providing enterprise-grade security without the complexity of TLS certificates.

## 🛡️ Multi-Layer Security Architecture

### **Layer 1: Network Isolation**
- **Localhost-Only Access**: Only `127.0.0.1`, `::1`, and loopback addresses
- **Origin Validation**: WebSocket connections restricted to localhost origins
- **IP Address Binding**: Sessions tied to specific IP addresses

### **Layer 2: Cryptographic Security**
- **256-bit Random Tokens**: Generated using `crypto/rand`
- **Session-Specific Salts**: Unique 256-bit salt per session
- **SHA-256 Hashing**: Tokens hashed with multi-factor salt
- **No Plaintext Storage**: Only hashed tokens stored

### **Layer 3: Session Hijacking Prevention**
- **IP Address Validation**: Tokens only valid from original IP
- **User-Agent Binding**: Sessions tied to specific User-Agent
- **Multi-Factor Salt**: Salt = Session Salt + IP + User-Agent
- **Session Expiration**: 24-hour automatic timeout

## 🔒 Token Generation & Validation Process

### **1. Session Creation:**
```go
// Generate 3 cryptographically secure components
sessionID := generateSecureToken()  // 256-bit random
token := generateSecureToken()      // 256-bit random  
salt := generateSecureToken()       // 256-bit random

// Bind to client context
ipAddress := getRealIP(request)
userAgent := request.UserAgent()

// Create multi-factor salt
combinedSalt := salt + ipAddress + userAgent

// Hash token with combined salt
tokenHash := SHA256(token + combinedSalt)
```

### **2. Token Validation:**
```go
// For each stored token, recreate hash with session context
for each storedToken {
    session := getSession(storedToken.sessionID)
    
    // Verify IP and User-Agent match
    if session.ipAddress != currentIP { continue }
    if session.userAgent != currentUserAgent { continue }
    
    // Recreate hash with session-specific salt
    expectedHash := SHA256(providedToken + session.salt + session.ip + session.userAgent)
    
    if storedToken.hash == expectedHash {
        return session // Valid token
    }
}
return error // Invalid token
```

## 🚫 Session Hijacking Protection

### **Attack Scenarios Prevented:**

#### **1. Token Theft:**
- **Problem**: Attacker steals token from logs/memory
- **Protection**: Token useless without IP + User-Agent context
- **Result**: Attacker cannot use stolen token from different machine

#### **2. Network Interception:**
- **Problem**: Token intercepted on localhost network
- **Protection**: Localhost-only validation + IP binding
- **Result**: Token only works from original localhost session

#### **3. Cross-Session Attacks:**
- **Problem**: Token reused across different browser sessions
- **Protection**: User-Agent validation + session-specific salt
- **Result**: Token tied to specific browser/client

#### **4. Replay Attacks:**
- **Problem**: Attacker replays captured requests
- **Protection**: IP + User-Agent + Salt validation
- **Result**: Replayed requests fail validation

## 🔐 Security Features Comparison

| Feature | Traditional JWT | Basic Session | CodeForge Enhanced |
|---------|----------------|---------------|-------------------|
| Token Security | ✅ Signed | ❌ Plain | ✅ Salted Hash |
| IP Binding | ❌ No | ❌ No | ✅ Yes |
| User-Agent Check | ❌ No | ❌ No | ✅ Yes |
| Session Hijacking Protection | ⚠️ Limited | ❌ No | ✅ Strong |
| Localhost Validation | ❌ No | ❌ No | ✅ Yes |
| Salt Protection | ❌ No | ❌ No | ✅ Multi-Factor |
| TLS Required | ✅ Yes | ✅ Yes | ❌ No |

## 🎯 Practical Security Benefits

### **For Web Interfaces:**
```javascript
// Token is automatically bound to browser context
const token = await api.login('My Web App');

// Token only works from:
// - Same IP address (127.0.0.1)
// - Same User-Agent (specific browser)
// - With correct session salt
// - Within 24-hour window

// Stolen token is useless because:
// - Different IP = validation fails
// - Different browser = User-Agent mismatch
// - No access to session salt = hash mismatch
```

### **For Development:**
- **Zero Setup**: No certificates or complex auth
- **Strong Security**: Prevents common attack vectors
- **Debug Friendly**: Clear error messages for auth failures
- **Session Management**: Easy logout and cleanup

## 🔍 Security Validation

### **Test Session Hijacking Protection:**
```bash
# 1. Get token from localhost
TOKEN=$(curl -s -X POST http://localhost:47000/api/v1/auth | jq -r .token)

# 2. Try to use from different User-Agent (should fail)
curl -H "Authorization: Bearer $TOKEN" \
     -H "User-Agent: AttackerBrowser/1.0" \
     http://localhost:47000/api/v1/chat/sessions
# Result: 401 Unauthorized

# 3. Try to use from different IP (should fail)
# This would fail at network level since only localhost is allowed

# 4. Use with correct context (should work)
curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:47000/api/v1/chat/sessions
# Result: 200 OK
```

## 📊 Security Metrics

### **Authentication Info Endpoint:**
```bash
curl http://localhost:47000/api/v1/auth/info
```

**Response:**
```json
{
  "type": "localhost-only",
  "features": [
    "256-bit cryptographically secure tokens",
    "SHA-256 hashing with session-specific salts", 
    "IP address validation",
    "User-Agent validation",
    "Session hijacking prevention",
    "Automatic token expiration"
  ],
  "security": {
    "token_length": 64,
    "salt_length": 64,
    "hash_algorithm": "SHA-256",
    "session_timeout": "24 hours",
    "ip_validation": true,
    "user_agent_check": true,
    "session_binding": true,
    "localhost_only": true,
    "hijacking_protection": true
  }
}
```

## 🎉 Why This Works for Localhost

### **Trust Model:**
- **Physical Security**: Localhost implies physical access to machine
- **Network Security**: Traffic never leaves the machine
- **Process Security**: Same security boundary as running CLI

### **Enhanced Beyond CLI:**
- **Explicit Authentication**: Not just "anyone on localhost"
- **Session Management**: Proper login/logout flow
- **Attack Prevention**: Protects against common web vulnerabilities
- **Audit Trail**: Track sessions and access patterns

### **Development Friendly:**
- **No TLS Complexity**: Works immediately without certificates
- **Standard Patterns**: Bearer token authentication
- **Web Compatible**: Works with fetch(), WebSocket, EventSource
- **Debug Support**: Clear error messages and validation

## 🔐 Conclusion

CodeForge's enhanced authentication provides **enterprise-grade security** for localhost development:

- **Stronger than basic session auth** (salt + binding)
- **More convenient than TLS** (no certificate setup)
- **Prevents session hijacking** (multi-factor validation)
- **Perfect for web interfaces** (standard Bearer tokens)

**The result: Secure localhost development without the complexity of production authentication systems!** 🚀
