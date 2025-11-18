# Nginx 配置模板更新总结

## 更新日期
2025年10月24日

## 更新内容

### 1. server-main.conf.tpl
**位置**: `/src/nginx/templates/conf.d/server-main.conf.tpl`

**关键更新**:
- ✅ 在 `/api/n9e/` location 中添加了 SSO 支持
- 添加了 `auth_request /__auth/verify`
- 添加了 `auth_request_set $auth_username $upstream_http_x_user`
- 添加了 `proxy_set_header X-User-Name $auth_username`

**更新的配置块**:
```nginx
location ^~ /api/n9e/ {
    # SSO Integration: Extract username from JWT token via auth_request
    auth_request /__auth/verify;
    auth_request_set $auth_username $upstream_http_x_user;
    
    # 不需要 rewrite，直接代理到 Nightingale，保持完整路径
    proxy_pass http://nightingale_console;
    
    # Pass the authenticated username to Nightingale for ProxyAuth
    proxy_set_header X-User-Name $auth_username;
    
    proxy_set_header Host $http_host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    
    # WebSocket support
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection $connection_upgrade;
}
```

### 2. nightingale.conf.tpl
**位置**: `/src/nginx/templates/conf.d/includes/nightingale.conf.tpl`

**状态**: ✅ 已经是最新版本，包含 SSO 支持
- 已包含 `auth_request /__auth/verify`
- 已包含 `auth_request_set $auth_username $upstream_http_x_user`
- 已包含 `proxy_set_header X-User-Name $auth_username`

### 3. jupyterhub.conf.tpl
**位置**: `/src/nginx/templates/conf.d/includes/jupyterhub.conf.tpl`

**状态**: ✅ 已匹配当前配置，无需更新

### 4. gitea.conf.tpl
**位置**: `/src/nginx/templates/conf.d/includes/gitea.conf.tpl`

**状态**: ✅ 已匹配当前配置，无需更新
- 使用环境变量 `${GITEA_ALIAS_ADMIN_TO}` 和 `${GITEA_ADMIN_EMAIL}`
- 支持 SSO 认证头传递

### 5. minio.conf.tpl
**位置**: `/src/nginx/templates/conf.d/includes/minio.conf.tpl`

**状态**: ✅ 已匹配当前配置，无需更新

## 模板变量说明

所有 `.tpl` 文件使用以下模板变量：

### 通用变量
- `{{EXTERNAL_SCHEME}}` - 外部访问协议（http/https）
- `{{EXTERNAL_HOST}}` - 外部访问地址（IP:Port）
- `{{FRONTEND_HOST}}` - 前端服务主机名
- `{{FRONTEND_PORT}}` - 前端服务端口
- `{{BACKEND_HOST}}` - 后端服务主机名
- `{{BACKEND_PORT}}` - 后端服务端口
- `{{JUPYTERHUB_HOST}}` - JupyterHub 服务主机名
- `{{JUPYTERHUB_PORT}}` - JupyterHub 服务端口

### Nightingale 变量
- `{{NIGHTINGALE_HOST}}` - Nightingale 服务主机名
- `{{NIGHTINGALE_PORT}}` - Nightingale 服务端口

### Gitea 变量
- `${GITEA_ALIAS_ADMIN_TO}` - Gitea 管理员用户名映射
- `${GITEA_ADMIN_EMAIL}` - Gitea 管理员邮箱

## SSO 认证流程

### Nightingale SSO 工作原理

1. **用户访问 `/monitoring` 或 `/nightingale/`**
   - Nginx 触发 `auth_request /__auth/verify`

2. **Backend 认证验证**
   - 从 Cookie 中读取 `auth_token`
   - 验证 JWT token
   - 返回 `X-User` 头（包含用户名）

3. **Nginx 提取用户信息**
   - 使用 `auth_request_set $auth_username $upstream_http_x_user`
   - 将用户名存储到变量 `$auth_username`

4. **传递给 Nightingale**
   - 设置 `proxy_set_header X-User-Name $auth_username`
   - Nightingale 的 ProxyAuth 读取此头部
   - 自动登录用户，无需单独认证

### Nightingale API SSO

所有 Nightingale API 调用（`/api/n9e/`）都通过相同的 SSO 流程：
- `/api/n9e/self/profile` - 获取用户信息
- `/api/n9e/busi-groups` - 获取业务组
- `/api/n9e/self/perms` - 获取权限
- `/api/n9e/datasource/brief` - 获取数据源

所有这些 API 都会自动附带 `X-User-Name` 头部，确保用户身份在整个会话中保持一致。

## 配置生成

模板文件在容器启动时通过 `docker-entrypoint.sh` 处理：

```bash
# 示例：替换模板变量
envsubst '${EXTERNAL_SCHEME} ${EXTERNAL_HOST} ...' \
  < /templates/conf.d/server-main.conf.tpl \
  > /etc/nginx/conf.d/server-main.conf
```

## 验证

所有模板已通过以下测试：
- ✅ Playwright E2E 测试（3/3 通过）
- ✅ Nightingale SSO 登录测试
- ✅ API 调用返回 200 状态码
- ✅ iframe 内容正常加载

## 相关文件

- 实际配置文件：`/src/nginx/conf.d/`
- 模板文件：`/src/nginx/templates/conf.d/`
- 启动脚本：`/src/nginx/docker-entrypoint.sh`
- Nightingale 配置：`/src/nightingale/etc/config.toml`
- Backend 认证：`/src/backend/internal/handlers/user_handler.go`

## 注意事项

1. **模板变量格式**
   - Nginx 变量使用 `$variable`
   - 模板变量使用 `{{VARIABLE}}` 或 `${VARIABLE}`
   - 不要混淆两者

2. **SSO 依赖**
   - 需要 Backend 的 `/api/auth/verify` 端点支持 Cookie 认证
   - 需要 Nightingale 启用 ProxyAuth 模式
   - 需要正确配置 `HeaderUserNameKey = "X-User-Name"`

3. **测试环境**
   - 开发环境：`http://192.168.18.154:8080`
   - 测试环境：`http://192.168.0.200:8080`
   - 确保 `$external_host` 变量正确设置

## 更新日志

### v0.3.8 (2025-10-24)
- ✅ 添加 Nightingale API SSO 支持（`/api/n9e/`）
- ✅ 更新 server-main.conf.tpl 模板
- ✅ 验证所有其他模板已是最新版本
- ✅ 通过完整的 E2E 测试
