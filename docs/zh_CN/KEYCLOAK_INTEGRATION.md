# Keycloak 单点登录集成指南

## 概述

Keycloak 是 AI Infrastructure Matrix 的统一身份认证管理 (IAM) 服务，提供：
- **SSO 单点登录**: 所有服务共享统一的用户认证
- **OIDC/OAuth2**: 标准的 OpenID Connect 协议支持
- **LDAP 集成**: 与现有 OpenLDAP 目录服务联合认证
- **细粒度 RBAC**: 基于角色的访问控制

## 架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                        用户浏览器                                │
└─────────────────────────┬───────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Nginx 反向代理                               │
│                   (auth_request 验证)                            │
└─────────────────────────┬───────────────────────────────────────┘
                          │
           ┌──────────────┼──────────────┐
           ▼              ▼              ▼
┌─────────────────┐ ┌─────────────┐ ┌─────────────────┐
│    Keycloak     │ │   Backend   │ │    Frontend     │
│  (身份认证中心)  │ │   (API)     │ │   (Web UI)      │
└────────┬────────┘ └──────┬──────┘ └─────────────────┘
         │                 │
         │    ┌────────────┼────────────┐
         │    ▼            ▼            ▼
         │ ┌───────┐ ┌──────────┐ ┌──────────┐
         │ │ Gitea │ │Nightingale│ │  ArgoCD  │
         │ └───────┘ └──────────┘ └──────────┘
         │
         ▼
┌─────────────────┐
│    OpenLDAP     │
│  (用户目录)     │
└─────────────────┘
```

## 配置步骤

### 1. 启用 Keycloak 服务

在 `.env` 文件中设置：

```bash
# 启用 Keycloak
KEYCLOAK_ENABLED=true
KEYCLOAK_VERSION=26.0
KEYCLOAK_HTTP_PORT=8180

# 管理员密码（请更改为强密码）
KEYCLOAK_ADMIN_PASSWORD=your-secure-password

# 数据库配置
KEYCLOAK_DB_HOST=postgres
KEYCLOAK_DB_NAME=keycloak
KEYCLOAK_DB_USER=keycloak
KEYCLOAK_DB_PASSWORD=keycloak-db-password
```

### 2. 客户端密钥配置

为每个服务配置 OIDC 客户端密钥：

```bash
# 后端服务
KEYCLOAK_BACKEND_CLIENT_SECRET=backend-secret-change-me

# Gitea
KEYCLOAK_GITEA_CLIENT_SECRET=gitea-secret-change-me

# Nightingale 监控
KEYCLOAK_N9E_CLIENT_SECRET=n9e-secret-change-me

# ArgoCD
KEYCLOAK_ARGOCD_CLIENT_SECRET=argocd-secret-change-me

# JupyterHub
KEYCLOAK_JUPYTERHUB_CLIENT_SECRET=jupyterhub-secret-change-me
```

### 3. 启动服务

```bash
# 使用 keycloak profile 启动
./build.sh up --profile keycloak

# 或者使用 full profile（包含所有可选服务）
./build.sh up --profile full
```

### 4. 初始化 Realm

首次启动时，Keycloak 会自动导入 `ai-infra-realm.json` 配置，包含：
- 预配置的 OIDC 客户端
- 角色定义（admin, sre, engineer, model-developer, data-developer, viewer）
- 组定义（administrators, sre-team, engineering, data-science, viewers）
- LDAP 联合配置

### 5. 配置 LDAP 联合

1. 登录 Keycloak 管理控制台: `http://your-host/auth/admin`
2. 进入 `ai-infra` Realm
3. 导航到 `User Federation` > `ldap`
4. 更新连接 URL 和绑定凭证
5. 点击 `Synchronize all users`

## 客户端集成

### Backend 服务

后端服务使用 OIDC 进行用户认证：

```go
// 配置示例
config := keycloak.Config{
    URL:          "http://keycloak:8080/auth",
    Realm:        "ai-infra",
    ClientID:     "ai-infra-backend",
    ClientSecret: os.Getenv("KEYCLOAK_BACKEND_CLIENT_SECRET"),
}
```

### Gitea

Gitea 使用 OAuth2 进行认证：

1. 进入 Gitea 管理后台
2. 配置 OAuth2 提供者
3. 填写 Keycloak 的 OIDC 端点信息

### Nightingale

Nightingale 使用反向代理认证模式：

```nginx
location /n9e/ {
    auth_request /auth/verify;
    auth_request_set $user $upstream_http_x_user;
    proxy_set_header X-User-Name $user;
    proxy_pass http://nightingale:17000/;
}
```

### ArgoCD

ArgoCD 通过 Dex 集成 Keycloak：

```yaml
# argocd-cm.yaml
dex.config: |
  connectors:
  - type: oidc
    id: keycloak
    name: Keycloak
    config:
      issuer: http://keycloak:8080/auth/realms/ai-infra
      clientID: argocd
      clientSecret: $KEYCLOAK_ARGOCD_CLIENT_SECRET
```

## 用户同步

当用户通过 Keycloak 注册或登录时，系统会自动同步用户信息到：
1. **Gitea**: 创建对应的 Git 账户
2. **Nightingale**: 创建监控系统账户
3. **其他服务**: 通过 LDAP 联合自动同步

同步逻辑在 `user_sync_service.go` 中实现。

## 角色映射

| Keycloak 角色 | 系统权限 |
|--------------|---------|
| admin | 系统管理员，所有权限 |
| sre | SRE 团队，基础设施管理 |
| engineer | 工程师，项目开发 |
| model-developer | 模型开发者，AI/ML 功能 |
| data-developer | 数据开发者，JupyterHub 访问 |
| viewer | 只读访问 |

## 故障排除

### 问题：无法登录

1. 检查 Keycloak 服务状态: `docker-compose ps keycloak`
2. 查看日志: `docker-compose logs keycloak`
3. 验证数据库连接
4. 确认客户端密钥正确

### 问题：LDAP 同步失败

1. 检查 LDAP 连接配置
2. 验证绑定 DN 和密码
3. 确认用户搜索基础 DN 正确

### 问题：SSO 跳转失败

1. 检查回调 URL 配置
2. 验证客户端重定向 URI
3. 确认 Nginx 代理配置正确

## 安全建议

1. **更改默认密码**: 首次部署后立即更改管理员密码
2. **使用 HTTPS**: 生产环境必须启用 TLS
3. **定期轮换密钥**: 定期更新客户端密钥
4. **启用 MFA**: 为管理员账户启用多因素认证
5. **审计日志**: 定期检查 Keycloak 审计日志

## 参考资料

- [Keycloak 官方文档](https://www.keycloak.org/documentation)
- [OIDC 协议规范](https://openid.net/specs/openid-connect-core-1_0.html)
- [OAuth 2.0 规范](https://oauth.net/2/)
