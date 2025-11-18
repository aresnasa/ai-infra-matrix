# Nightingale 登出问题修复报告

## 问题描述

用户报告：**"logout is not supported when proxy auth is enabled，无法退出 nightingale"**

访问地址：http://192.168.18.114:8080/monitoring

## 问题诊断

### 测试结果

运行 Playwright 测试 `test/e2e/specs/nightingale-logout.spec.js`：

```
✅ 4/4 测试通过
   - ✓ 登出按钮存在（在 avatar 菜单中）
   - ✓ 点击登出成功
   - ✓ 返回到登录页面
   - ✓ 配置检查完成
```

### 根本原因

#### 1. Nightingale 启用了 ProxyAuth

**配置文件**: `/app/etc/server.conf` (在 nightingale 容器中)

```toml
[HTTP.ProxyAuth]
# if proxy auth enabled, jwt auth is disabled
Enable = true
HeaderUserNameKey = "X-User-Name"
DefaultRoles = ["Admin"]
```

**影响**:
- JWT 认证被禁用
- 每次请求从 HTTP header `X-User-Name` 读取用户名
- 自动登录该用户（无需密码）
- **登出功能被禁用**（因为下次请求会自动重新登录）

#### 2. Nginx 设置了固定的用户名 header

**配置文件**: `src/nginx/conf.d/includes/nightingale.conf`

```nginx
location ^~ /nightingale/ {
    proxy_pass http://nightingale:17000/;
    
    # ProxyAuth - set default anonymous user
    proxy_set_header X-User-Name "anonymous";
    
    # ... other headers
}
```

**影响**:
- 所有访问 Nightingale 的请求都被认证为 `anonymous` 用户
- 即使点击登出，下次请求仍然自动以 `anonymous` 身份登录
- 无法切换用户（永远是 anonymous）

### 为什么测试显示登出成功？

测试中确实能点击登出按钮并返回登录页面，但问题是：
- 当再次访问 `/monitoring` 时，nginx 再次发送 `X-User-Name: anonymous`
- Nightingale 自动登录 `anonymous` 用户
- **实际上无法真正登出系统**

---

## 解决方案

### 🎯 方案 1: 禁用 ProxyAuth（推荐）

**优点**:
- 恢复正常的登录/登出流程
- 每个用户使用独立的账号（admin, root 等）
- 更安全的认证机制（JWT + 密码）
- 支持会话管理

**实施步骤**:

#### 步骤 1: 修改 Nightingale 配置

需要将 Nightingale 容器的配置文件更新为：

```toml
[HTTP.ProxyAuth]
# Disable ProxyAuth to enable normal login/logout
Enable = false
HeaderUserNameKey = "X-User-Name"
DefaultRoles = ["Admin"]
```

#### 步骤 2: 移除 Nginx 的 ProxyAuth header

修改 `src/nginx/conf.d/includes/nightingale.conf`：

```nginx
location ^~ /nightingale/ {
    proxy_pass http://nightingale:17000/;
    
    # Remove ProxyAuth header to enable normal authentication
    # proxy_set_header X-User-Name "anonymous";  # <-- REMOVE THIS LINE
    
    # Standard proxy headers
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    
    # ... rest of config
}
```

#### 步骤 3: 重启服务

```bash
# Rebuild and restart Nightingale (if config is baked into image)
docker-compose build nightingale
docker-compose up -d nightingale

# Restart Nginx
docker-compose restart nginx
```

#### 步骤 4: 验证

```bash
# 测试登录和登出
BASE_URL=http://192.168.18.114:8080 npx playwright test test/e2e/specs/nightingale-login.spec.js

# 测试登出后不会自动重新登录
BASE_URL=http://192.168.18.114:8080 npx playwright test test/e2e/specs/nightingale-logout.spec.js
```

---

### 🔄 方案 2: 动态传递用户名（集成方案）

如果希望主系统用户自动登录 Nightingale（单点登录效果），需要：

**优点**:
- 主系统和 Nightingale 用户统一
- 无需在 Nightingale 单独登录

**缺点**:
- 仍然无法在 Nightingale 中登出（ProxyAuth 特性）
- 需要前端和后端配合传递用户信息

**实施步骤**:

#### 步骤 1: 后端添加用户名传递

修改 `src/backend` 的 proxy handler，在转发到 Nightingale 时：

```go
// 从 JWT token 或 session 获取当前用户名
username := c.GetString("username")

// 设置 X-User-Name header
c.Request.Header.Set("X-User-Name", username)
```

#### 步骤 2: Nginx 使用变量传递用户名

```nginx
location ^~ /nightingale/ {
    proxy_pass http://nightingale:17000/;
    
    # Pass through X-User-Name from backend
    proxy_set_header X-User-Name $http_x_user_name;
    
    # ... rest of config
}
```

#### 步骤 3: 接受登出限制

由于 ProxyAuth 的特性，用户需要：
- 在主系统中登出（这会清除 JWT token）
- 然后访问 Nightingale 才会失去访问权限

**注意**: 这不是真正的"登出 Nightingale"，而是"登出主系统"。

---

### 🚫 方案 3: 完全移除 ProxyAuth（最简单）

如果不需要 SSO 功能：

#### 步骤 1: 修改 Nightingale docker-compose

确保配置文件挂载或环境变量设置为：

```yaml
nightingale:
  image: flashcatcloud/nightingale:latest
  environment:
    - HTTP_PROXYAUTH_ENABLE=false
  # 或挂载自定义配置文件
  volumes:
    - ./config/nightingale.conf:/app/etc/server.conf
```

#### 步骤 2: 移除 Nginx header

```nginx
# Remove this line completely
# proxy_set_header X-User-Name "anonymous";
```

#### 步骤 3: 重启并测试

```bash
docker-compose up -d nightingale nginx
```

---

## 推荐实施方案

### 🎯 立即修复（方案 1）

1. **修改 Nginx 配置**（最快，无需重启 Nightingale）

```bash
# 编辑 nginx 配置
vi src/nginx/conf.d/includes/nightingale.conf

# 注释掉或删除这一行：
# proxy_set_header X-User-Name "anonymous";

# 重启 nginx
docker-compose restart nginx
```

2. **修改 Nightingale 配置**

创建自定义配置文件或通过环境变量禁用 ProxyAuth：

**方法 A: 环境变量**（如果 Nightingale 支持）

```yaml
# docker-compose.yml
nightingale:
  environment:
    - N9E_HTTP_PROXYAUTH_ENABLE=false
```

**方法 B: 配置文件挂载**

```bash
# 1. 从容器复制配置文件
docker cp ai-infra-nightingale:/app/etc/server.conf ./config/nightingale.conf

# 2. 修改配置
vi ./config/nightingale.conf
# 找到 [HTTP.ProxyAuth] 部分，设置 Enable = false

# 3. 在 docker-compose.yml 中挂载
volumes:
  - ./config/nightingale.conf:/app/etc/server.conf:ro

# 4. 重启 Nightingale
docker-compose up -d nightingale
```

---

## 验证步骤

### 1. 验证配置生效

```bash
# 检查 Nightingale 配置
docker exec ai-infra-nightingale cat /app/etc/server.conf | grep -A5 ProxyAuth

# 应该看到：
# [HTTP.ProxyAuth]
# Enable = false

# 检查 Nginx 配置
docker exec ai-infra-nginx cat /etc/nginx/conf.d/includes/nightingale.conf | grep X-User-Name

# 应该没有输出（或被注释）
```

### 2. 功能测试

```bash
# 运行完整测试套件
BASE_URL=http://192.168.18.114:8080 npx playwright test test/e2e/specs/nightingale-login.spec.js
BASE_URL=http://192.168.18.114:8080 npx playwright test test/e2e/specs/nightingale-logout.spec.js
```

### 3. 手动测试

1. 访问 http://192.168.18.114:8080/monitoring
2. 应该看到登录页面（不是自动登录）
3. 使用 admin/admin123 登录
4. 点击右上角用户菜单
5. 点击"退出/登出"
6. 应该返回登录页面
7. **关键验证**: 刷新页面，应该仍然在登录页面（不会自动登录）

---

## 技术说明

### ProxyAuth 工作原理

```
用户请求 → Nginx → 设置 X-User-Name header → Nightingale
                                                    ↓
                                              自动登录该用户
                                              (无需密码验证)
```

**问题**:
- 每次请求都携带 `X-User-Name` header
- Nightingale 每次都自动登录
- 点击登出后，下次请求仍然自动登录
- **无法真正退出系统**

### JWT 认证工作原理（禁用 ProxyAuth 后）

```
用户请求登录 → Nightingale 验证密码 → 返回 JWT token
                                          ↓
用户携带 token 访问 → Nightingale 验证 token → 允许访问
                         ↓
                    点击登出 → 删除 token
                         ↓
                    下次访问 → 无 token → 跳转登录页 ✓
```

---

## 相关文件

### 需要修改的文件

1. **`src/nginx/conf.d/includes/nightingale.conf`**
   - 移除 `proxy_set_header X-User-Name "anonymous";`

2. **`config/nightingale.conf`** (需要创建/修改)
   - 设置 `[HTTP.ProxyAuth] Enable = false`

3. **`docker-compose.yml`** (可选)
   - 添加配置文件挂载或环境变量

### 测试文件

1. **`test/e2e/specs/nightingale-login.spec.js`** - 登录功能测试
2. **`test/e2e/specs/nightingale-logout.spec.js`** - 登出功能测试（新建）

---

## 实施清单

- [ ] 备份当前配置文件
- [ ] 修改 Nginx 配置（移除 X-User-Name header）
- [ ] 修改 Nightingale 配置（禁用 ProxyAuth）
- [ ] 重启 Nginx 服务
- [ ] 重启 Nightingale 服务
- [ ] 运行自动化测试验证
- [ ] 手动测试登录/登出流程
- [ ] 验证登出后不会自动重新登录
- [ ] 更新文档说明新的登录方式

---

## 后续影响

### 用户体验变化

**修改前**:
- 访问 /monitoring 自动以 anonymous 身份登录
- 无需输入密码
- 无法登出（或登出无效）

**修改后**:
- 访问 /monitoring 显示登录页面
- 需要输入用户名密码（admin/admin123）
- 可以正常登出
- 登出后不会自动重新登录 ✓

### 安全性提升

- ✅ 需要密码认证（不是自动登录）
- ✅ 支持会话管理和超时
- ✅ 可以审计不同用户的操作
- ✅ 防止未授权访问

---

## 总结

**问题**: ProxyAuth 导致无法真正登出 Nightingale

**原因**: 
- Nginx 硬编码 `X-User-Name: anonymous` header
- Nightingale 启用 ProxyAuth 自动登录
- 每次请求都重新自动登录

**解决**: 
- 禁用 Nightingale ProxyAuth
- 移除 Nginx X-User-Name header
- 使用正常的 JWT 认证流程

**结果**: 
- ✅ 支持正常的登录/登出
- ✅ 用户可以切换账号
- ✅ 更安全的认证机制

---

**修复优先级**: 🔴 高（影响用户体验和安全性）

**预计修复时间**: 15-30 分钟（配置修改 + 测试验证）

**风险评估**: 低（配置回退简单，不影响数据）
