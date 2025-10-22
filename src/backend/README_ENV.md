# Backend 环境变量配置说明

## ⚠️ 重要变更

从 v0.3.8 版本开始，所有环境变量配置已归一化到项目根目录的 `.env` 文件中统一管理。

**不再使用** `src/backend/.env` 或 `src/backend/.env.example`！

## 配置文件位置

```
ai-infra-matrix/
├── .env                    # ✅ 统一的环境变量配置（主配置文件）
├── .env.example            # ✅ 环境变量示例文件（用于参考）
├── src/
│   └── backend/
│       ├── .env            # ❌ 已废弃，不再使用
│       └── .env.example    # ❌ 已废弃，不再使用
```

## 配置方法

### 1. 开发环境

1. 复制示例配置：
   ```bash
   cp .env.example .env
   ```

2. 修改配置：
   ```bash
   vim .env
   ```

3. 启动服务：
   ```bash
   docker-compose up -d backend
   ```

### 2. 生产环境

#### Docker Compose 部署

docker-compose.yml 已配置自动加载 `.env` 文件：

```yaml
backend:
  env_file:
    - .env  # 自动加载项目根目录的 .env
  environment:
    # 可以在这里覆盖特定变量
    DB_HOST: "${POSTGRES_HOST:-postgres}"
```

#### Kubernetes 部署

使用 ConfigMap 和 Secret：

```bash
# 从 .env 文件创建 ConfigMap
kubectl create configmap backend-config \
  --from-env-file=.env \
  --namespace=ai-infra

# 创建包含敏感信息的 Secret
kubectl create secret generic backend-secret \
  --from-literal=DB_PASSWORD="your-password" \
  --from-literal=SALT_API_PASSWORD="your-password" \
  --namespace=ai-infra
```

在 Deployment 中引用：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backend
spec:
  template:
    spec:
      containers:
      - name: backend
        envFrom:
        - configMapRef:
            name: backend-config
        - secretRef:
            name: backend-secret
```

## 环境变量说明

### 数据库配置

```bash
# PostgreSQL
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your-postgres-password
DB_NAME=ai-infra-matrix

# OceanBase（可选）
OB_ENABLED=false
OB_HOST=oceanbase
OB_PORT=2881
OB_USER=root@sys
OB_PASSWORD=
OB_DB=aimatrix
```

### Redis 配置

```bash
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=your-redis-password
REDIS_DB=0
```

### SaltStack API 配置

```bash
# 完整 URL 方式（推荐）
SALTSTACK_MASTER_URL=http://salt-master:8000

# 或者分别配置
SALT_API_SCHEME=http
SALT_MASTER_HOST=salt-master
SALT_API_PORT=8000

# 认证配置
SALT_API_USERNAME=saltapi
SALT_API_PASSWORD=saltapi123
SALT_API_EAUTH=file
SALT_API_TIMEOUT=8s
```

### LDAP 配置

```bash
LDAP_SERVER=openldap
LDAP_PORT=389
LDAP_BASE_DN=dc=ai-infra,dc=com
LDAP_BIND_DN=cn=admin,dc=ai-infra,dc=com
LDAP_ADMIN_PASSWORD=your-ldap-password
```

### JWT 配置

```bash
JWT_SECRET=your-jwt-secret-key-change-in-production
JWT_EXPIRY=24h
```

### AI 助手配置

```bash
# OpenAI
OPENAI_API_KEY=sk-proj-...
OPENAI_BASE_URL=https://api.openai.com/v1
OPENAI_DEFAULT_MODEL=gpt-4

# Claude
CLAUDE_API_KEY=sk-ant-...
CLAUDE_BASE_URL=https://api.anthropic.com
CLAUDE_DEFAULT_MODEL=claude-3-5-sonnet-20241022

# DeepSeek
DEEPSEEK_API_KEY=sk-...
DEEPSEEK_BASE_URL=https://api.deepseek.com/v1
DEEPSEEK_DEFAULT_MODEL=deepseek-chat
```

### 日志配置

```bash
LOG_LEVEL=info  # debug, info, warn, error
LOG_FORMAT=json
```

## 环境变量优先级

1. **docker-compose.yml 中的 environment 显式设置** - 最高优先级
2. **.env 文件中的配置** - 次优先级
3. **代码中的默认值** - 最低优先级

示例：

```yaml
# docker-compose.yml
backend:
  env_file:
    - .env                    # 加载所有变量
  environment:
    DB_HOST: postgres-prod    # 覆盖 .env 中的 DB_HOST
```

## 迁移指南

如果你之前在 `src/backend/.env` 中配置了环境变量，需要：

1. **备份旧配置**：
   ```bash
   cp src/backend/.env src/backend/.env.backup
   ```

2. **迁移到根目录**：
   ```bash
   cat src/backend/.env >> .env
   ```

3. **删除重复项**：
   ```bash
   # 手动编辑 .env，删除重复的配置项
   vim .env
   ```

4. **验证配置**：
   ```bash
   # 重启服务测试
   docker-compose restart backend
   docker-compose logs -f backend
   ```

5. **清理旧文件**（可选）：
   ```bash
   rm src/backend/.env
   ```

## 常见问题

### Q1: 为什么要统一到根目录？

**A:** 
- ✅ 所有服务共享同一份配置，避免配置不一致
- ✅ 更容易管理和维护
- ✅ 符合 Docker Compose 和 Kubernetes 的最佳实践
- ✅ 避免在多个地方修改相同的配置

### Q2: 修改 .env 后需要重启吗？

**A:** 是的，需要重启相关服务：

```bash
docker-compose restart backend
```

或者重新构建：

```bash
docker-compose up -d --build backend
```

### Q3: 如何查看当前生效的环境变量？

**A:** 使用以下命令：

```bash
docker exec ai-infra-backend env | sort
```

### Q4: 如何设置不同环境的配置？

**A:** 创建不同的 .env 文件：

```bash
# 开发环境
.env.development

# 测试环境
.env.test

# 生产环境
.env.production
```

使用时指定：

```bash
docker-compose --env-file .env.production up -d
```

### Q5: 敏感信息如何保护？

**A:** 

1. **.env 文件不要提交到 Git**：
   ```bash
   # .gitignore
   .env
   .env.local
   .env.*.local
   ```

2. **使用 .env.example 作为模板**：
   ```bash
   # .env.example 可以提交，包含配置项但不包含真实值
   DB_PASSWORD=your-password-here
   ```

3. **生产环境使用 Kubernetes Secret**：
   ```bash
   kubectl create secret generic backend-secret \
     --from-literal=DB_PASSWORD="$(openssl rand -base64 32)"
   ```

## 相关文档

- [环境变量示例文件](/.env.example)
- [Docker Compose 配置](/docker-compose.yml)
- [SaltStack 配置说明](/docs/SALTSTACK_INTEGRATION_ENV_CONFIG.md)
- [Kubernetes 部署指南](/helm/README.md)

## 更新日志

- **2025-10-21**: v0.3.8 - 统一环境变量管理到项目根目录
  - 废弃 `src/backend/.env`
  - 所有配置归一化到 `/.env`
  - 更新 Dockerfile 移除旧配置文件复制
  - 添加迁移指南
