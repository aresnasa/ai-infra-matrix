# 认证系统设计

## 概述

AI Infrastructure Matrix 的认证系统基于 JWT (JSON Web Token)，提供统一的身份认证和授权机制。

## 认证流程

### 基本认证流程

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

## JWT Token 结构

### Token 组成

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

## 用户模型

### 数据结构

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

### 角色权限

| 角色 | 权限 | 说明 |
|------|------|------|
| admin | 全部权限 | 系统管理员 |
| user | 读写权限 | 普通用户 |
| readonly | 只读权限 | 只读用户 |
| guest | 限制权限 | 访客用户 |

## API 认证实现

### 登录接口

```go
// POST /api/auth/login
func Login(c *gin.Context) {
    var req LoginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "Invalid request"})
        return
    }

    // 1. 查询用户
    var user User
    if err := db.Where("username = ?", req.Username).First(&user).Error; err != nil {
        c.JSON(401, gin.H{"error": "Invalid credentials"})
        return
    }

    // 2. 验证密码
    if err := bcrypt.CompareHashAndPassword(
        []byte(user.Password),
        []byte(req.Password),
    ); err != nil {
        c.JSON(401, gin.H{"error": "Invalid credentials"})
        return
    }

    // 3. 生成 JWT token
    token, err := generateJWT(user.ID, user.Username, user.Role)
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to generate token"})
        return
    }

    // 4. 返回 token
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

### Token 生成

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

### 认证中间件

```go
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. 获取 token
        authHeader := c.GetHeader("Authorization")
        if authHeader == "" {
            c.JSON(401, gin.H{"error": "Missing authorization header"})
            c.Abort()
            return
        }

        tokenString := strings.TrimPrefix(authHeader, "Bearer ")

        // 2. 解析 token
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

        // 3. 提取用户信息
        if claims, ok := token.Claims.(jwt.MapClaims); ok {
            c.Set("user_id", uint(claims["user_id"].(float64)))
            c.Set("username", claims["username"].(string))
            c.Set("role", claims["role"].(string))
        }

        c.Next()
    }
}
```

### 权限检查

```go
func RequireRole(role string) gin.HandlerFunc {
    return func(c *gin.Context) {
        userRole := c.GetString("role")
        
        // admin 拥有所有权限
        if userRole == "admin" {
            c.Next()
            return
        }

        // 检查角色匹配
        if userRole != role {
            c.JSON(403, gin.H{"error": "Insufficient permissions"})
            c.Abort()
            return
        }

        c.Next()
    }
}

// 使用示例
router.DELETE("/api/users/:id", AuthMiddleware(), RequireRole("admin"), DeleteUser)
```

## 前端集成

### Token 存储

```typescript
// 登录后存储 token
const login = async (username: string, password: string) => {
  const response = await fetch('/api/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ username, password })
  });

  const data = await response.json();
  if (response.ok) {
    // 存储到 localStorage
    localStorage.setItem('token', data.token);
    localStorage.setItem('user', JSON.stringify(data.user));
  }
};
```

### API 请求拦截

```typescript
// Axios 拦截器
import axios from 'axios';

const api = axios.create({
  baseURL: '/api',
});

// 请求拦截器 - 添加 token
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

// 响应拦截器 - 处理认证错误
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Token 过期，跳转登录页
      localStorage.removeItem('token');
      localStorage.removeItem('user');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);
```

### 路由守卫

```typescript
// React Router 路由守卫
import { Navigate } from 'react-router-dom';

const PrivateRoute = ({ children }: { children: JSX.Element }) => {
  const token = localStorage.getItem('token');
  
  if (!token) {
    return <Navigate to="/login" replace />;
  }

  return children;
};

// 使用
<Route path="/dashboard" element={
  <PrivateRoute>
    <Dashboard />
  </PrivateRoute>
} />
```

## JupyterHub 认证集成

### 自定义 Authenticator

```python
# jupyterhub_config.py
from jupyterhub.auth import Authenticator
import requests

class CustomAuthenticator(Authenticator):
    async def authenticate(self, handler, data):
        username = data['username']
        password = data['password']

        # 调用后端 API 验证
        response = requests.post(
            'http://backend:8000/api/auth/verify',
            json={'username': username, 'password': password}
        )

        if response.status_code == 200:
            return username
        
        return None

c.JupyterHub.authenticator_class = CustomAuthenticator
```

## Gitea 认证集成

Gitea 使用自己的用户系统，但可以通过以下方式集成：

### 1. LDAP/OAuth 集成（未来）

```ini
[oauth2_client]
ENABLE_AUTO_REGISTRATION = true
```

### 2. API 同步用户

创建用户时同步到 Gitea：

```go
func CreateUser(username, email, password string) error {
    // 1. 创建内部用户
    user := User{Username: username, Email: email}
    db.Create(&user)

    // 2. 同步到 Gitea
    giteaAPI := fmt.Sprintf("http://gitea:3000/api/v1/admin/users")
    payload := map[string]string{
        "username": username,
        "email":    email,
        "password": password,
    }
    
    // POST to Gitea API...
}
```

## 安全最佳实践

### 1. 密码安全

```go
// 使用 bcrypt 加密密码
hashedPassword, _ := bcrypt.GenerateFromPassword(
    []byte(plainPassword),
    bcrypt.DefaultCost,
)

// 设置密码复杂度要求
func validatePassword(password string) error {
    if len(password) < 8 {
        return errors.New("password must be at least 8 characters")
    }
    // 检查大小写、数字、特殊字符...
}
```

### 2. Token 安全

- 使用强随机密钥（至少 256 位）
- 设置合理的过期时间（通常 24小时）
- 支持 token 刷新机制
- 实现 token 黑名单（用于登出）

### 3. HTTPS

生产环境必须使用 HTTPS：

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

### 4. CORS 配置

```go
router.Use(cors.New(cors.Config{
    AllowOrigins:     []string{"https://ai-infra.example.com"},
    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
    AllowHeaders:     []string{"Authorization", "Content-Type"},
    AllowCredentials: true,
    MaxAge:           12 * time.Hour,
}))
```

### 5. 速率限制

```go
// 限制登录请求频率
import "github.com/ulule/limiter/v3"

rateLimiter := limiter.Rate{
    Period: 1 * time.Minute,
    Limit:  5, // 每分钟最多5次登录尝试
}
```

### 6. 审计日志

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

## 故障排查

### Token 无效

检查项：
1. JWT_SECRET 是否配置正确
2. Token 是否过期
3. Token 格式是否正确（Bearer token）

### 登录失败

检查项：
1. 用户名密码是否正确
2. 用户是否被禁用
3. 数据库连接是否正常

### 权限错误

检查项：
1. 用户角色是否正确
2. API 路由权限配置
3. 中间件顺序

## 相关文档

- [API 文档](API_REFERENCE.md)
- [系统架构](ARCHITECTURE.md)
- [安全指南](SECURITY_GUIDE.md)
