# AI基础设施矩阵 - 统一架构指南

## 🎯 核心理念
**Backend-Centric Architecture**: 后端作为整个集群的认证中心，所有服务通过后端进行统一认证和权限管理。

## 🏗️ 架构概览

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│     Nginx       │────│    Backend      │────│   PostgreSQL    │
│   (入口代理)     │    │  (认证中心)     │    │   (主数据库)    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │                       │                       │
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│    Frontend     │    │   JupyterHub    │    │     Redis       │
│   (前端界面)     │    │  (Notebook服务) │    │    (缓存)       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## 🔐 认证流程

1. **用户登录** → Frontend → Backend JWT认证
2. **JWT Token** → 所有服务间认证凭证
3. **权限验证** → Backend统一管理用户权限
4. **服务访问** → JWT Token透传，无需重复认证

## 🚀 快速部署

### 1. 环境准备
```bash
# 克隆项目
git clone https://github.com/aresnasa/ai-infra-matrix.git
cd ai-infra-matrix

# 设置环境变量
export JWT_SECRET="your-secret-key-change-in-production"
export BACKEND_URL="http://backend:8080"
```

### 2. 启动服务
```bash
# 启动核心服务
docker-compose up -d postgres redis backend

# 启动JupyterHub
docker-compose up -d jupyterhub

# 启动前端和代理
docker-compose up -d frontend nginx
```

### 3. 访问地址
- **主入口**: http://localhost:8080
- **后端API**: http://localhost:8080/api
- **JupyterHub**: http://localhost:8080/jupyter
- **前端**: http://localhost:8080/app

## 🛠️ 核心配置

### Backend认证配置
```go
// JWT配置
JWTSecret = "your-secret-key-change-in-production"
TokenExpiry = 24 * time.Hour

// 用户认证端点
POST /api/auth/login    // 用户登录
GET  /api/auth/verify   // Token验证
GET  /api/users/{id}    // 用户信息
```

### JupyterHub集成配置
```python
# 统一后端认证器
c.JupyterHub.authenticator_class = BackendIntegratedAuthenticator

# 数据库配置
c.JupyterHub.db_url = "postgresql://user:pass@postgres:5432/db"

# 认证配置
c.Authenticator.allow_all = True  # 权限由后端控制
c.Authenticator.enable_auth_state = True
```

### Docker构建优化
```dockerfile
# 时间戳层失效策略
RUN echo "Config build: $(date)" > /srv/build_info.txt

# 配置文件最后复制（缓存优化）
COPY jupyterhub_config.py /srv/jupyterhub/
COPY backend_integrated_config.py /srv/jupyterhub/
```

## 🔧 开发指南

### 添加新服务
1. **认证集成**: 调用Backend `/api/auth/verify`
2. **权限检查**: 使用JWT Token中的roles/permissions
3. **服务注册**: 添加到nginx路由配置

### 调试技巧
```bash
# 查看服务状态
docker-compose ps

# 查看日志
docker-compose logs -f jupyterhub

# JWT Token测试
curl -H "Authorization: Bearer $TOKEN" \
     http://localhost:8080/api/auth/verify
```

## 📊 性能优化

### 构建缓存策略
- **依赖层**: 先复制requirements.txt
- **配置层**: 使用时间戳强制重建
- **代码层**: 最后复制源码

### 数据库连接池
```python
# PostgreSQL连接池
max_connections = 20
pool_size = 5
pool_recycle = 3600
```

### Redis缓存策略
```python
# 会话缓存
session_timeout = 86400  # 24小时
cache_prefix = "ai_infra:"
```

## 🚨 故障排除

### 常见问题

#### 1. JWT认证失败
```bash
# 检查密钥一致性
echo $JWT_SECRET

# 验证Token格式
jwt-cli decode $TOKEN
```

#### 2. 数据库连接失败
```bash
# 检查PostgreSQL状态
docker-compose ps postgres

# 测试连接
psql postgresql://user:pass@localhost:5432/db
```

#### 3. 服务间通信问题
```bash
# 检查网络连接
docker network ls
docker network inspect ai-infra-matrix_default
```

## 📋 检查清单

### 部署前检查
- [ ] JWT_SECRET已设置
- [ ] 数据库连接正常
- [ ] Redis缓存可用
- [ ] Docker网络配置正确

### 运行时监控
- [ ] 所有容器健康
- [ ] 认证端点响应正常
- [ ] 日志无错误信息
- [ ] 性能指标正常

## 🎯 最佳实践

1. **统一认证**: 所有服务使用Backend认证，避免重复实现
2. **配置精简**: 删除冗余配置，保持单一真相源
3. **容器优化**: 使用多阶段构建，优化镜像大小
4. **缓存策略**: 合理使用Docker层缓存，提升构建速度
5. **监控告警**: 实施服务健康检查和日志监控

## 🔄 持续改进

### 版本控制
- 主配置文件版本化
- 数据库迁移脚本管理
- 构建流水线自动化

### 扩展性考虑
- 微服务架构预留
- 负载均衡配置
- 水平扩展支持

---

**记住**: 保持简洁，避免过度工程化。每个配置都应该有明确的目的和价值。
