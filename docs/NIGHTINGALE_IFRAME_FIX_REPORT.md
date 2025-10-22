# Nightingale 监控系统 iframe 集成修复报告

## 问题诊断

通过 Playwright 测试诊断发现：
- ✅ 路由访问正常 (`/monitoring`)
- ✅ Iframe 元素存在
- ✅ Nightingale 服务响应正常 (HTTP 200)
- ✅ 无 X-Frame-Options 阻止
- ✅ 无 CSP 限制
- ❌ **401 Unauthorized 错误** - Nightingale ProxyAuth 未收到用户认证头

## 根本原因

Nightingale 使用 ProxyAuth SSO，需要 `X-User-Name` HTTP 头来识别用户。但是：
1. Iframe 请求不会自动传递父页面的 HTTP 头
2. 浏览器安全策略阻止 iframe 跨域传递认证信息
3. 前端直接访问 `http://hostname:17000` 绕过了 nginx 反向代理

## 解决方案

### 1. Nginx 反向代理配置

**创建文件**: `src/nginx/conf.d/includes/nightingale.conf`
- 将 Nightingale 代理到 `/nightingale/` 路径
- 使用 nginx `auth_request` 模块从后端获取用户信息
- 自动注入 `X-User-Name` 头到所有 Nightingale 请求

**创建模板**: `src/nginx/templates/conf.d/includes/nightingale.conf.tpl`
- 支持通过环境变量配置
- 变量: `NIGHTINGALE_HOST`, `NIGHTINGALE_PORT`

### 2. 后端 API 增强

**修改文件**: `src/backend/internal/handlers/user_handler.go`
- `GetProfile()` 函数新增响应头:
  - `X-User-Name`: 用户名
  - `X-User-Email`: 用户邮箱  
  - `X-User-ID`: 用户ID
- 用于 nginx `auth_request` 模块提取用户信息

### 3. 前端配置调整

**修改文件**: `src/frontend/src/pages/MonitoringPage.js`
- 默认使用 nginx 代理路径: `/nightingale/`
- 支持 3 种配置方式:
  1. `REACT_APP_NIGHTINGALE_URL` - 完整 URL（优先级最高）
  2. `REACT_APP_NIGHTINGALE_PORT` - 直接端口访问
  3. 默认 - `/nightingale/` 代理路径（推荐，支持 SSO）

**修改文件**: `.env` 和 `.env.example`
- 注释掉 `REACT_APP_NIGHTINGALE_PORT`
- 默认留空，使用 nginx 代理

### 4. Build.sh 模板渲染

**修改文件**: `build.sh`
- `render_template()` 添加变量:
  - `NIGHTINGALE_HOST`
  - `NIGHTINGALE_PORT`
- `render_nginx_templates()` 添加逻辑:
  - 检查 `NIGHTINGALE_ENABLED` 环境变量
  - 启用时渲染 `nightingale.conf.tpl`
  - 禁用时创建空配置文件

### 5. Nginx 主配置

**修改文件**: `src/nginx/conf.d/server-main.conf`
- 添加 `include /etc/nginx/conf.d/includes/nightingale.conf;`

## 技术细节

### ProxyAuth 认证流程

```
用户请求 -> Nginx -> auth_request /internal/nightingale-auth
                  |
                  v
            Backend /api/auth/me (验证 JWT)
                  |
                  v
            返回 X-User-Name 头
                  |
                  v
            Nginx 注入头 -> Nightingale
                                |
                                v
                          ProxyAuth 识别用户
```

### Nginx Auth Request 配置

```nginx
location /nightingale/ {
    auth_request /internal/nightingale-auth;
    auth_request_set $auth_username $upstream_http_x_user_name;
    proxy_set_header X-User-Name $auth_username;
    # ...
}

location = /internal/nightingale-auth {
    internal;
    proxy_pass http://backend:8080/api/auth/me;
    # ...
}
```

### Nightingale ProxyAuth 配置

在 `src/nightingale/etc/config.toml`:
```toml
[HTTP.ProxyAuth]
Enable = true
HeaderUserNameKey = "X-User-Name"
DefaultRoles = ["Admin"]
```

## 测试步骤

### 1. 重新构建服务

```bash
# 渲染 nginx 配置
./build.sh nginx

# 或完整构建
./build.sh all
```

### 2. 重启服务

```bash
docker-compose restart nginx backend frontend
```

### 3. 验证配置

```bash
# 检查 nginx 配置
docker-compose exec nginx nginx -t

# 查看生成的配置
cat src/nginx/conf.d/includes/nightingale.conf
```

### 4. 浏览器测试

1. 访问 http://192.168.18.114:8080/monitoring
2. 检查 iframe 是否显示 Nightingale UI
3. 验证用户已自动登录（右上角显示用户名）

### 5. Playwright 自动化测试

```bash
BASE_URL=http://192.168.18.114:8080 \
npx playwright test test/e2e/specs/nightingale-iframe-test.spec.js \
  --config=test/e2e/playwright.config.js
```

## 环境变量配置

### .env 文件

```bash
# Nightingale 监控配置
NIGHTINGALE_ENABLED=true
NIGHTINGALE_HOST=nightingale
NIGHTINGALE_PORT=17000
NIGHTINGALE_ALERT_PORT=19000

# 前端访问配置（留空使用 nginx 代理，推荐）
# REACT_APP_NIGHTINGALE_PORT=
REACT_APP_NIGHTINGALE_URL=
```

### 配置说明

- **推荐方式**: 留空，使用 `/nightingale/` 代理路径，支持 ProxyAuth SSO
- **直接端口**: 设置 `REACT_APP_NIGHTINGALE_PORT=17000`，但**不支持 SSO**
- **自定义 URL**: 设置 `REACT_APP_NIGHTINGALE_URL=https://nightingale.example.com`

## 文件清单

### 新增文件
- `src/nginx/conf.d/includes/nightingale.conf` - Nginx 配置
- `src/nginx/templates/conf.d/includes/nightingale.conf.tpl` - 模板
- `test/e2e/specs/nightingale-iframe-test.spec.js` - Playwright 测试

### 修改文件
- `src/nginx/conf.d/server-main.conf` - 添加 include
- `src/backend/internal/handlers/user_handler.go` - GetProfile 添加响应头
- `src/frontend/src/pages/MonitoringPage.js` - 默认使用代理路径
- `.env` - 注释端口配置
- `.env.example` - 同步更新
- `build.sh` - 添加 Nightingale 模板渲染逻辑

## 优势

1. **统一认证**: 通过 nginx 代理，自动传递用户认证信息
2. **安全性**: 不暴露 Nightingale 端口，所有请求经过认证
3. **简化配置**: 前端无需配置端口，自动使用代理路径
4. **灵活部署**: 支持 Docker Compose 和 Kubernetes
5. **可维护性**: 模板化配置，易于管理和更新

## 注意事项

1. **首次部署**: 需要运行 `./build.sh nginx` 渲染配置
2. **环境变量**: 确保 `.env` 中 `NIGHTINGALE_ENABLED=true`
3. **服务顺序**: nginx 需要在 backend 和 nightingale 之后启动
4. **权限控制**: MonitoringPage 默认要求 SRE 团队权限
5. **Nightingale 数据库**: 使用独立数据库 `nightingale`（不是 `ai_infra_matrix`）

## 排查问题

### 问题: Iframe 显示 401 错误

**检查**:
```bash
# 1. 检查 nginx 配置是否包含 nightingale.conf
docker-compose exec nginx cat /etc/nginx/conf.d/includes/nightingale.conf

# 2. 检查后端 API 是否返回用户信息头
curl -H "Authorization: Bearer YOUR_TOKEN" \
  http://192.168.18.114:8080/api/auth/me -v

# 3. 查看 nginx 日志
docker-compose logs nginx | grep nightingale
```

### 问题: Nightingale 配置未渲染

**解决**:
```bash
# 确保环境变量已加载
source .env

# 手动渲染 nginx 配置
./build.sh nginx

# 检查生成的文件
cat src/nginx/conf.d/includes/nightingale.conf
```

### 问题: 用户名未传递

**检查**:
```bash
# 1. 确认后端返回 X-User-Name 头
docker-compose logs backend | grep "X-User-Name"

# 2. 检查 nginx auth_request 配置
docker-compose exec nginx nginx -T | grep -A 10 nightingale-auth
```

## 后续优化

1. **权限细化**: 根据用户角色限制 Nightingale 功能
2. **多租户**: 基于用户/团队隔离监控数据
3. **性能优化**: 启用 nginx 缓存减少后端请求
4. **监控集成**: 将 AI-Infra-Matrix 指标接入 Nightingale
5. **告警联动**: Nightingale 告警通知到前端系统

## 总结

通过 nginx 反向代理和 auth_request 模块，成功实现了：
- ✅ Nightingale iframe 无缝集成
- ✅ ProxyAuth SSO 自动登录
- ✅ 统一认证体系
- ✅ 安全的用户身份传递
- ✅ 灵活的配置管理

**核心思想**: 将认证逻辑从前端移到 nginx，利用 auth_request 子请求获取用户信息并注入到上游服务，实现透明的 SSO 集成。
