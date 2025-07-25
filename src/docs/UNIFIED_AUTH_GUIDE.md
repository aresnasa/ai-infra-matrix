# AI-Infra-Matrix 统一认证系统配置指南

## 概述

AI-Infra-Matrix现已支持JupyterHub统一认证系统，使用PostgreSQL作为用户数据库，Redis作为会话缓存，实现与后端系统完全一致的用户账户体系。

## 架构变更

### 原有架构
- JupyterHub独立认证（DummyAuthenticator/LocalProcessSpawner）
- SQLite本地数据库
- 用户需要单独注册JupyterHub账户

### 新架构
- JupyterHub统一认证（PostgreSQLRedisAuthenticator）
- PostgreSQL共享数据库（与后端系统共享）
- Redis会话缓存
- 统一用户体系，一次登录全系统可用

## 环境变量配置

### 数据库配置

```bash
# PostgreSQL数据库配置（与后端共享）
DB_HOST=localhost
DB_PORT=5432
DB_NAME=ansible_playbook_generator
DB_USER=postgres
DB_PASSWORD=postgres
```

### Redis配置

```bash
# Redis缓存配置
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=ansible-redis-password
REDIS_DB=1  # 使用不同的数据库避免冲突
```

### JupyterHub配置

```bash
# JWT密钥（必须与后端保持一致）
JWT_SECRET=your-secret-key-change-in-production

# 会话配置
SESSION_TIMEOUT=86400  # 24小时会话过期

# 管理员用户
JUPYTERHUB_ADMIN_USERS=admin,jupyter-admin

# 代理令牌
CONFIGPROXY_AUTH_TOKEN=ai-infra-proxy-token

# Spawner配置
JUPYTERHUB_NOTEBOOK_IMAGE=jupyter/base-notebook:latest
JUPYTERHUB_NETWORK=ai-infra-matrix_ansible-network
JUPYTERHUB_MEM_LIMIT=2G
JUPYTERHUB_CPU_LIMIT=1.0

# 功能开关
JUPYTERHUB_IDLE_CULLER_ENABLED=true
JUPYTERHUB_IDLE_TIMEOUT=3600      # 1小时空闲超时
JUPYTERHUB_CULL_TIMEOUT=7200      # 2小时清理超时
JUPYTERHUB_DEBUG=false
JUPYTERHUB_LOG_LEVEL=INFO
JUPYTERHUB_ACCESS_LOG=true

# CORS配置
JUPYTERHUB_CORS_ORIGIN=http://localhost:3001
```

### 前端配置

```bash
# 前端环境变量 (.env)
REACT_APP_API_URL=http://localhost:8082
REACT_APP_JUPYTERHUB_URL=http://localhost:8088
REACT_APP_JUPYTERHUB_UNIFIED_AUTH=true
REACT_APP_JUPYTERHUB_LEGACY_MODE=false
REACT_APP_DEBUG_MODE=false
```

## 部署方式

### 1. 统一认证版本（推荐）

```bash
# 启动统一认证版本的JupyterHub
docker-compose --profile jupyterhub-unified up -d

# 或者启动完整系统
docker-compose --profile full up -d
```

### 2. 测试对比版本

```bash
# 启动旧版本JupyterHub（用于对比测试）
docker-compose --profile jupyterhub-legacy up -d

# 启动测试环境
docker-compose --profile testing up -d
```

## 功能特性

### 统一认证流程

1. **用户登录后端系统** → 获得JWT令牌
2. **前端调用统一登录接口** → 生成JupyterHub专用令牌
3. **自动跳转JupyterHub** → 使用统一认证免密登录
4. **会话状态同步** → Redis缓存保持会话一致性

### 权限管理

- **管理员用户**: 具有`admin`或`super-admin`角色的用户自动获得JupyterHub管理权限
- **普通用户**: 普通用户可启动个人Notebook服务器
- **用户组支持**: 支持用户组权限管理（后续扩展）

### 会话管理

- **Redis缓存**: 用户会话信息缓存到Redis，提高认证性能
- **自动过期**: 24小时会话自动过期，支持自定义配置
- **统一登出**: 支持一键清除所有系统会话

### Docker支持

- **智能Spawner选择**: 自动检测Docker可用性，智能选择DockerSpawner或LocalProcessSpawner
- **资源限制**: 支持内存和CPU限制配置
- **卷管理**: 自动管理用户数据卷和共享目录

## 使用指南

### 1. 管理员首次设置

1. 确保PostgreSQL和Redis服务正常运行
2. 在后端系统中创建管理员用户
3. 设置环境变量
4. 启动统一认证版本JupyterHub
5. 使用管理员账户登录测试

### 2. 用户使用流程

1. 登录AI-Infra-Matrix主系统
2. 访问"JupyterHub管理"页面
3. 点击"统一认证登录"按钮
4. 自动跳转到JupyterHub并完成登录
5. 启动Notebook服务器开始工作

### 3. 开发者集成

前端组件：
```javascript
import UnifiedJupyterHubIntegration from './components/UnifiedJupyterHubIntegration';

// 在路由中使用
<Route path="/jupyterhub" component={UnifiedJupyterHubIntegration} />
```

后端API：
```go
// 生成JupyterHub登录令牌
POST /auth/jupyterhub-login

// 检查JupyterHub状态
GET /jupyterhub/status

// 管理Notebook服务器
POST /jupyterhub/start-server
POST /jupyterhub/stop-server
```

## 故障排除

### 常见问题

1. **数据库连接失败**
   - 检查PostgreSQL服务状态
   - 验证数据库连接参数
   - 确保用户表已正确创建

2. **Redis连接失败**
   - 检查Redis服务状态
   - 验证Redis密码配置
   - 检查网络连接

3. **JupyterHub启动失败**
   - 查看容器日志：`docker logs ai-infra-jupyterhub-unified`
   - 检查端口冲突
   - 验证环境变量配置

4. **用户认证失败**
   - 确保JWT_SECRET与后端一致
   - 检查用户数据库中的用户状态
   - 验证用户角色配置

### 日志查看

```bash
# JupyterHub日志
docker logs ai-infra-jupyterhub-unified

# PostgreSQL日志
docker logs ansible-postgres

# Redis日志
docker logs ansible-redis

# 后端日志
docker logs ansible-backend
```

### 调试模式

启用调试模式：
```bash
export JUPYTERHUB_DEBUG=true
export JUPYTERHUB_LOG_LEVEL=DEBUG
export REACT_APP_DEBUG_MODE=true
```

## 迁移指南

### 从旧版本迁移

1. **备份数据**
   ```bash
   docker-compose exec postgres pg_dump -U postgres ansible_playbook_generator > backup.sql
   ```

2. **停止旧版本服务**
   ```bash
   docker-compose --profile jupyterhub-legacy down
   ```

3. **启动统一认证版本**
   ```bash
   docker-compose --profile jupyterhub-unified up -d
   ```

4. **验证用户数据**
   - 检查用户表中的数据完整性
   - 测试用户登录功能
   - 验证角色权限配置

## 安全建议

1. **生产环境配置**
   - 修改默认JWT_SECRET
   - 使用强密码配置数据库
   - 启用SSL/TLS加密
   - 配置防火墙规则

2. **会话安全**
   - 定期清理过期会话
   - 监控异常登录行为
   - 设置合理的会话超时时间

3. **数据备份**
   - 定期备份PostgreSQL数据库
   - 备份JupyterHub配置文件
   - 监控系统资源使用情况

## 性能优化

1. **Redis优化**
   - 配置合适的内存限制
   - 启用数据持久化
   - 监控缓存命中率

2. **PostgreSQL优化**
   - 配置连接池
   - 优化查询索引
   - 监控数据库性能

3. **JupyterHub优化**
   - 配置合理的资源限制
   - 启用空闲会话清理
   - 监控容器资源使用

## 联系支持

如遇到问题，请：
1. 查看日志文件获取详细错误信息
2. 参考本文档的故障排除部分
3. 在项目GitHub仓库创建Issue
4. 提供详细的环境信息和错误日志

---

**版本**: v1.0.0  
**更新时间**: 2025年7月24日  
**适用范围**: AI-Infra-Matrix统一认证系统
