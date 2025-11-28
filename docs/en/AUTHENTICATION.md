```markdown
# Authentication System Design

## Overview

The authentication system for AI Infrastructure Matrix is based on JWT (JSON Web Token), providing a unified identity authentication and authorization mechanism.

## Authentication Flow

### Basic Authentication Flow

```text
┌──────────┐                 ┌──────────┐                 ┌──────────┐
│  Client  │                 │ Backend  │                 │   DB     │
└────┬─────┘                 └────┬─────┘                 └────┬─────┘
     │                            │                            │
     │ 1. POST /api/auth/login    │                            │
     │   {username, password}     │                            │
     ├───────────────────────────►│                            │
     │                            │ 2. Query user              │
     │                            ├───────────────────────────►│
     │                            │                            │
     │                            │◄───────────────────────────┤
     │                            │ 3. User data               │
     │                            │                            │
     │                            │ 4. Verify password         │
     │                            │    (bcrypt.Compare)        │
     │                            │                            │
     │                            │ 5. Generate JWT token      │
     │                            │    sign({user_id, role})   │
     │                            │                            │
     │◄───────────────────────────┤ 6. Return token            │
     │ {token, user}              │                            │
     │                            │                            │
     │ 7. Store token             │                            │
     │    (localStorage)          │                            │
     │                            │                            │
     │ 8. API Request             │                            │
     │    Authorization: Bearer <token>                        │
     ├───────────────────────────►│                            │
     │                            │ 9. Verify token            │
     │                            │    jwt.Parse(token)        │
     │                            │                            │
     │◄───────────────────────────┤ 10. Response               │
     │ {data}                     │                            │
     │                            │                            │
```

## JWT Token Structure

### Token Composition

```text
Header.Payload.Signature
```

### Header

```json
{
  "alg": "HS256",
  "typ": "JWT"
}
```

### Payload

```json
{
  "user_id": 1,
  "username": "admin",
  "role": "admin",
  "exp": 1700308800,
  "iat": 1700222400
}
```

### Signature

```text
HMACSHA256(
  base64UrlEncode(header) + "." +
  base64UrlEncode(payload),
  secret_key
)
```

## User Model

### Data Structure

```go
type User struct {
    ID        uint      `gorm:"primaryKey"`
    Username  string    `gorm:"uniqueIndex;not null"`
    Password  string    `gorm:"not null"` // bcrypt hash
    Email     string    `gorm:"uniqueIndex"`
    Role      string    `gorm:"default:user"` // admin, user, readonly
    Active    bool      `gorm:"default:true"`
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### Role Permissions

| Role | Permissions | Description |
|------|-------------|-------------|
| admin | Full access | System administrator |
| user | Read/write access | Regular user |
| readonly | Read-only access | Read-only user |
| guest | Limited access | Guest user |

## API Authentication Implementation

### Login Endpoint

```go
// POST /api/auth/login
func Login(c *gin.Context) {
    var req LoginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "Invalid request"})
        return
    }

    // 1. Query user
    var user User
    if err := db.Where("username = ?", req.Username).First(&user).Error; err != nil {
        c.JSON(401, gin.H{"error": "Invalid credentials"})
        return
    }

    // 2. Verify password
    if err := bcrypt.CompareHashAndPassword(
        []byte(user.Password),
        []byte(req.Password),
    ); err != nil {
        c.JSON(401, gin.H{"error": "Invalid credentials"})
        return
    }

    // 3. Generate JWT token
    token, err := generateJWT(user.ID, user.Username, user.Role)
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to generate token"})
        return
    }

    // 4. Return token
    c.JSON(200, gin.H{
        "token": token,
        "user": gin.H{
            "id":       user.ID,
            "username": user.Username,
            "role":     user.Role,
        },
    })
}
```

### Token Generation

```go
func generateJWT(userID uint, username, role string) (string, error) {
    claims := jwt.MapClaims{
        "user_id":  userID,
        "username": username,
        "role":     role,
        "exp":      time.Now().Add(24 * time.Hour).Unix(),
        "iat":      time.Now().Unix(),
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}
```

### Authentication Middleware

```go
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. Get token
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(401, gin.H{"error": "Missing authorization header"})
            c.Abort()
            return
        }

        tokenString := strings.TrimPrefix(authHeader, "Bearer ")

        // 2. Parse token
        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fmt.Errorf("unexpected signing method")
            }
            return []byte(os.Getenv("JWT_SECRET")), nil
        })

        if err != nil || !token.Valid {
            c.JSON(401, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }

        // 3. Extract user information
        if claims, ok := token.Claims.(jwt.MapClaims); ok {
            c.Set("user_id", uint(claims["user_id"].(float64)))
            c.Set("username", claims["username"].(string))
            c.Set("role", claims["role"].(string))
        }

        c.Next()
    }
}
```

### Permission Check

```go
func RequireRole(role string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userRole := c.GetString("role")
        
        // admin has all permissions
        if userRole == "admin" {
            c.Next()
            return
        }

        // Check role match
        if userRole != role {
            c.JSON(403, gin.H{"error": "Insufficient permissions"})
            c.Abort()
            return
        }

        c.Next()
    }
}

// Usage example
router.DELETE("/api/users/:id", AuthMiddleware(), RequireRole("admin"), DeleteUser)
```

## Frontend Integration

### Token Storage

```typescript
// Store token after login
const login = async (username: string, password: string) => {
  const response = await fetch('/api/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password })
  });

  const data = await response.json();
  if (response.ok) {
    // Store in localStorage
    localStorage.setItem('token', data.token);
    localStorage.setItem('user', JSON.stringify(data.user));
  }
};
```

### API Request Interceptor

```typescript
// Axios interceptor
import axios from 'axios';

const api = axios.create({
  baseURL: '/api',
});

// Request interceptor - add token
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// Response interceptor - handle authentication errors
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Token expired, redirect to login page
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);
```

### Route Guard

```typescript
// React Router route guard
import { Navigate } from 'react-router-dom';

const PrivateRoute = ({ children }: { children: JSX.Element }) => {
  const token = localStorage.getItem('token');
  
  if (!token) {
    return <Navigate to="/login" replace />;
  }

  return children;
};

// Usage
<Route path="/dashboard" element={
  <PrivateRoute>
    <Dashboard />
  </PrivateRoute>
} />
```

## JupyterHub Authentication Integration

### Custom Authenticator

```python
# jupyterhub_config.py
from jupyterhub.auth import Authenticator
import requests

class CustomAuthenticator(Authenticator):
    async def authenticate(self, handler, data):
        username = data['username']
        password = data['password']

        # Call backend API for verification
        response = requests.post(
            'http://backend:8000/api/auth/verify',
            json={'username': username, 'password': password}
        )

        if response.status_code == 200:
            return username
        
        return None

c.JupyterHub.authenticator_class = CustomAuthenticator
```

## Gitea Authentication Integration

Gitea uses its own user system, but can be integrated through the following methods:

### 1. LDAP/OAuth Integration (Future)

```ini
[oauth2_client]
ENABLE_AUTO_REGISTRATION = true
```

### 2. User Synchronization via API

Synchronize to Gitea when creating users:

```go
func CreateUser(username, email, password string) error {
    // 1. Create internal user
    user := User{Username: username, Email: email}
    db.Create(&user)

    // 2. Synchronize to Gitea
    giteaAPI := fmt.Sprintf("http://gitea:3000/api/v1/admin/users")
    payload := map[string]string{
        "username": username,
        "email":    email,
        "password": password,
    }
    
    // POST to Gitea API...
}
```

## Security Best Practices

### 1. Password Security

```go
// Use bcrypt to encrypt passwords
hashedPassword, _ := bcrypt.GenerateFromPassword(
    []byte(plainPassword),
    bcrypt.DefaultCost,
)

// Set password complexity requirements
func validatePassword(password string) error {
    if len(password) < 8 {
        return errors.New("password must be at least 8 characters")
    }
    // Check for uppercase, lowercase, numbers, special characters...
}
```

### 2. Token Security

- Use strong random keys (at least 256 bits)
- Set reasonable expiration time (typically 24 hours)
- Support token refresh mechanism
- Implement token blacklist (for logout)

### 3. HTTPS

HTTPS is required in production:

```nginx
server {
    listen 443 ssl http2;
    server_name ai-infra.example.com;

    ssl_certificate /etc/ssl/certs/cert.pem;
    ssl_certificate_key /etc/ssl/private/key.pem;
    
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
}
```

### 4. CORS Configuration

```go
router.Use(cors.New(cors.Config{
    AllowOrigins:     []string{"https://ai-infra.example.com"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
    AllowHeaders:     []string{"Authorization", "Content-Type"},
    AllowCredentials: true,
    MaxAge:           12 * time.Hour,
}))
```

### 5. Rate Limiting

```go
// Limit login request frequency
import "github.com/ulule/limiter/v3"

rateLimiter := limiter.Rate{
    Period: 1 * time.Minute,
    Limit:  5, // Maximum 5 login attempts per minute
}
```

### 6. Audit Logging

```go
func LogAuthEvent(userID uint, action, result string) {
    db.Create(&AuthLog{
        UserID:    userID,
        Action:    action,  // "login", "logout", "token_refresh"
        Result:    result,  // "success", "failure"
        IP:        c.ClientIP(),
        UserAgent: c.GetHeader("User-Agent"),
        Timestamp: time.Now(),
    })
}
```

## Troubleshooting

### Invalid Token

Checklist:
1. Is JWT_SECRET configured correctly
2. Has the token expired
3. Is the token format correct (Bearer token)

### Login Failure

Checklist:
1. Are username and password correct
2. Is the user disabled
3. Is the database connection working

### Permission Error

Checklist:
1. Is the user role correct
2. API route permission configuration
3. Middleware order

## Related Documentation

- [API Documentation](API_REFERENCE.md)
- [System Architecture](ARCHITECTURE.md)
- [Security Guide](SECURITY_GUIDE.md)

```
