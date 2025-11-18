# Docker Compose 环境变量化改造总结

## 概述

本次改造将 `docker-compose.yml.example` 和 `.env.prod.example` 中的硬编码主机地址进行环境变量化，使其支持 Kubernetes 部署和外部服务连接。

## 主要更改

### 1. .env.prod.example 新增环境变量

#### 数据库配置
```bash
# PostgreSQL 数据库配置
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_DB=ai-infra-matrix
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres

# 数据库连接字符串（支持外部 RDS）
DATABASE_URL=postgresql://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}
```

#### Redis 配置
```bash
# Redis 缓存配置
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=ansible-redis-password
REDIS_DB=0
REDIS_CACHE_DB=1

# Redis 连接字符串（支持外部 ElastiCache）
REDIS_URL=redis://:${REDIS_PASSWORD}@${REDIS_HOST}:${REDIS_PORT}/${REDIS_DB}
REDIS_CACHE_URL=redis://:${REDIS_PASSWORD}@${REDIS_HOST}:${REDIS_PORT}/${REDIS_CACHE_DB}
```

#### 服务通信配置
```bash
# 内部服务通信地址
BACKEND_URL=http://backend:8082
FRONTEND_URL=http://frontend:80
JUPYTERHUB_URL=http://jupyterhub:8000
GITEA_URL=http://gitea:3000
MINIO_URL=http://minio:9000
LDAP_URL=ldap://openldap:389

# 服务主机配置
BACKEND_HOST=backend
FRONTEND_HOST=frontend
JUPYTERHUB_HOST=jupyterhub
GITEA_HOST=gitea
MINIO_HOST=minio
LDAP_HOST=openldap
```

#### 外部服务示例配置
```bash
# 外部服务配置示例（K8s 环境）
# POSTGRES_HOST=my-rds-instance.region.rds.amazonaws.com
# REDIS_HOST=my-elasticache.region.cache.amazonaws.com
# LDAP_HOST=ldap.company.com
# MINIO_HOST=s3.amazonaws.com
```

### 2. docker-compose.yml.example 服务更新

#### Backend 服务
- `DB_HOST`: `"${POSTGRES_HOST:-postgres}"`
- `DB_PORT`: `"${POSTGRES_PORT:-5432}"`
- `REDIS_HOST`: `"${REDIS_HOST:-redis}"`
- `REDIS_PORT`: `"${REDIS_PORT:-6379}"`
- `LDAP_SERVER`: `"${LDAP_HOST:-openldap}"`

#### JupyterHub 服务
- `POSTGRES_HOST`: `${POSTGRES_HOST:-postgres}`
- `POSTGRES_PORT`: `${POSTGRES_PORT:-5432}`
- `REDIS_HOST`: `${REDIS_HOST:-redis}`
- `REDIS_PORT`: `${REDIS_PORT:-6379}`
- `AI_INFRA_BACKEND_URL`: `${BACKEND_URL:-http://backend:8082}`

#### Gitea 服务
- `GITEA_DB_HOST`: `"${POSTGRES_HOST:-postgres}:${POSTGRES_PORT:-5432}"`
- `GITEA_DB_USER`: `"${GITEA_DB_USER:-${POSTGRES_USER:-postgres}}"`
- `GITEA_DB_PASSWD`: `"${GITEA_DB_PASSWD:-${POSTGRES_PASSWORD:-postgres}}"`
- `MINIO_ENDPOINT`: `"${MINIO_HOST:-minio}:${MINIO_PORT:-9000}"`

#### MinIO 服务
- `MINIO_ROOT_USER`: `${MINIO_ACCESS_KEY:-minioadmin}`
- `MINIO_ROOT_PASSWORD`: `${MINIO_SECRET_KEY:-minioadmin}`

#### SaltStack 服务
- `AI_INFRA_BACKEND_URL`: `${BACKEND_URL:-http://backend:8082}`

## 部署模式支持

### 1. Docker Compose 模式（默认）
使用默认值，所有服务在同一网络内通信：
```bash
POSTGRES_HOST=postgres
REDIS_HOST=redis
LDAP_HOST=openldap
```

### 2. Kubernetes 模式
修改 `.env.prod` 文件：
```bash
POSTGRES_HOST=postgres-service
REDIS_HOST=redis-service
LDAP_HOST=openldap-service
```

### 3. 外部服务模式
连接外部托管服务：
```bash
POSTGRES_HOST=my-rds.us-west-2.rds.amazonaws.com
REDIS_HOST=my-elasticache.us-west-2.cache.amazonaws.com
LDAP_HOST=ldap.company.com
```

## 验证

运行以下命令验证配置语法：
```bash
docker-compose -f docker-compose.yml.example config --dry-run
```

## 下一步

1. **Kubernetes 部署**：基于此环境变量结构创建 Kubernetes manifests
2. **Helm Charts**：创建 Helm templates 支持不同部署模式
3. **外部服务集成**：添加对 AWS RDS、ElastiCache、Azure Database 等的支持
4. **配置验证**：添加启动时的配置验证脚本

## 兼容性

- ✅ 向后兼容：现有 Docker Compose 部署不受影响
- ✅ 渐进迁移：可以逐步替换单个服务为外部服务
- ✅ 开发调试：保持调试端口和调试模式支持
- ✅ 配置灵活：支持环境变量覆盖和默认值

## 相关文档

- [Kubernetes 部署指南](K8S_DEPLOYMENT_GUIDE.md)
- [外部服务集成指南](EXTERNAL_SERVICES_GUIDE.md)
- [环境变量配置参考](ENV_VARIABLES_REFERENCE.md)
