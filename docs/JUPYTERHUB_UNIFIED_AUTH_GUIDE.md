# JupyterHub 统一认证部署指南

## 🎯 概述

本指南详细介绍如何部署和配置 JupyterHub 与 AI 基础设施矩阵的统一认证系统，实现账号密码和 token 的统一管理。

## 🏗️ 系统架构

```
用户认证流程:
用户 → 前端登录 → AI基础设施矩阵后端 → JWT Token → JupyterHub统一认证器 → JupyterLab
     ↓
   统一的用户数据库和权限管理
```

### 核心组件

1. **AI基础设施矩阵后端**: 统一的用户认证和权限管理
2. **JupyterHub统一认证器**: 自定义认证器，与后端API集成
3. **前端统一认证管理界面**: 用户和管理员界面
4. **JWT Token管理**: 自动刷新和验证机制

## 📋 部署步骤

### 步骤1: 准备环境

```bash
# 确保conda环境已创建
conda create -n ai-infra-matrix python=3.9 -y

# 激活环境
conda activate ai-infra-matrix
```

### 步骤2: 配置环境变量

```bash
# 复制环境配置模板
cp .env.jupyterhub.example .env.jupyterhub

# 编辑配置文件
vim .env.jupyterhub
```

关键配置项：
```bash
# 后端API配置
AI_INFRA_BACKEND_URL=http://localhost:8080
AI_INFRA_API_TOKEN=your-secure-api-token

# JupyterHub管理员用户
JUPYTERHUB_ADMIN_USERS=admin,jupyter-admin

# 端口配置
JUPYTERHUB_PORT=8090
```

### 步骤3: 安装和配置JupyterHub

```bash
# 运行设置脚本
./scripts/start-jupyterhub.sh setup

# 检查安装状态
./scripts/start-jupyterhub.sh status
```

### 步骤4: 启动后端服务

```bash
# 进入后端目录
cd src/backend

# 启动后端API
go run cmd/main.go
```

### 步骤5: 启动JupyterHub

```bash
# 后台启动JupyterHub
./scripts/start-jupyterhub.sh daemon

# 检查服务状态
./scripts/start-jupyterhub.sh status
```

### 步骤6: 构建和部署前端

```bash
# 进入src目录
cd src

# 构建前端容器
docker-compose build frontend

# 启动前端服务
docker-compose up -d frontend
```

### 步骤7: 验证部署

```bash
# 运行认证集成测试
./scripts/test-jupyterhub-auth.sh
```

## 🔧 配置详解

### JupyterHub配置 (`third-party/jupyterhub/simple_jupyterhub_config.py`)

```python
# 统一认证器配置
c.JupyterHub.authenticator_class = AIInfraMatrixAuthenticator
c.AIInfraMatrixAuthenticator.backend_api_url = 'http://localhost:8080'
c.AIInfraMatrixAuthenticator.enable_auth_state = True
c.AIInfraMatrixAuthenticator.auto_login = True
```

### 认证器功能 (`third-party/jupyterhub/ai_infra_auth.py`)

- **JWT Token认证**: 支持使用后端签发的JWT token直接登录
- **用户名密码认证**: 通过后端API验证用户凭据
- **自动Token刷新**: 在token即将过期时自动刷新
- **环境变量注入**: 为用户环境注入认证信息

### 后端API端点

- `POST /api/auth/login` - 标准登录
- `POST /api/auth/jupyterhub-login` - JupyterHub专用登录
- `POST /api/auth/verify-token` - 验证JWT token
- `POST /api/auth/refresh-token` - 刷新JWT token

## 🌐 访问地址

- **前端界面**: http://localhost:3001
- **JupyterHub集成页面**: http://localhost:3001/jupyterhub
- **统一认证管理**: http://localhost:3001/jupyterhub/auth
- **JupyterHub直接访问**: http://localhost:8090
- **后端API**: http://localhost:8080

## 👥 用户管理

### 创建用户

1. **通过前端注册页面**:
   - 访问: http://localhost:3001/auth
   - 填写用户名、邮箱和密码

2. **通过API**:
   ```bash
   curl -X POST http://localhost:8080/api/auth/register \
     -H "Content-Type: application/json" \
     -d '{
       "username": "newuser",
       "email": "user@example.com",
       "password": "securepassword"
     }'
   ```

### 用户登录流程

1. **前端登录**: 用户在前端界面登录
2. **获取JWT**: 后端验证凭据并返回JWT token
3. **JupyterHub认证**: 用户访问JupyterHub时自动使用JWT认证
4. **环境设置**: JupyterHub自动设置用户环境和权限

## 🛡️ 安全特性

### JWT Token管理
- **自动过期**: Token有明确的过期时间
- **自动刷新**: 即将过期时自动刷新
- **安全存储**: Token存储在认证状态中

### 权限控制
- **角色基础**: 支持admin、user等角色
- **权限映射**: JupyterHub管理员权限与后端角色同步
- **会话管理**: 完整的会话生命周期管理

### 数据保护
- **密码哈希**: 使用bcrypt加密存储密码
- **Cookie安全**: 使用安全的cookie配置
- **API认证**: 所有API调用都需要有效认证

## 🔍 故障排除

### 常见问题

1. **JupyterHub启动失败**
   ```bash
   # 检查conda环境
   conda info --envs
   
   # 检查日志
   ./scripts/start-jupyterhub.sh logs
   ```

2. **认证失败**
   ```bash
   # 测试后端连接
   curl http://localhost:8080/api/health
   
   # 运行认证测试
   ./scripts/test-jupyterhub-auth.sh
   ```

3. **Token验证失败**
   ```bash
   # 检查token格式
   echo $JWT_TOKEN | cut -d'.' -f2 | base64 -d
   
   # 验证token
   curl -X POST http://localhost:8080/api/auth/verify-token \
     -H "Content-Type: application/json" \
     -d '{"token": "'$JWT_TOKEN'"}'
   ```

### 日志查看

```bash
# JupyterHub日志
tail -f log/jupyterhub.log

# 后端日志
cd src/backend && go run cmd/main.go

# 前端日志
docker-compose logs frontend
```

## 📊 监控和维护

### 服务状态检查

```bash
# 检查所有服务状态
./scripts/start-jupyterhub.sh status

# 检查端口占用
lsof -i :8080  # 后端
lsof -i :8090  # JupyterHub
lsof -i :3001  # 前端
```

### 性能监控

```bash
# 检查内存使用
ps aux | grep jupyterhub

# 检查数据库连接
sqlite3 data/jupyterhub/jupyterhub.sqlite ".tables"
```

## 🔮 扩展功能

### 计划中的功能

- [ ] LDAP集成支持
- [ ] 多租户管理
- [ ] GPU资源配额管理
- [ ] 作业队列监控
- [ ] 审计日志
- [ ] SSO集成

### 自定义配置

1. **修改认证策略**:
   编辑 `third-party/jupyterhub/ai_infra_auth.py`

2. **添加用户钩子**:
   在 `pre_spawn_start` 方法中添加自定义逻辑

3. **扩展API**:
   在后端 `handlers/user_handler.go` 中添加新端点

## 📚 相关文档

- [JupyterHub文档](https://jupyterhub.readthedocs.io/)
- [JWT规范](https://tools.ietf.org/html/rfc7519)
- [Go Gin框架](https://gin-gonic.com/)
- [React Ant Design](https://ant.design/)

---

*文档版本: v1.0*  
*最后更新: 2025年7月24日*
