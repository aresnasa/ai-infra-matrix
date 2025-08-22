# 环境变量统一配置指南

## 概述

本项目已实现环境变量的统一管理，支持 Docker Compose 和 Helm 两种部署方式，通过统一的 `.env` 文件配置所有环境变量，避免了多个环境文件共存导致的配置冲突问题。

## 文件结构

```
├── .env                      # 当前激活的环境配置
├── .env.prod                 # 生产环境配置（备用）
├── .env.unified             # 开发环境模板
├── .env.prod.unified        # 生产环境模板
├── scripts/env-manager.sh   # 环境管理脚本
├── docker-compose.yml       # Docker Compose 配置
└── helm/ai-infra-matrix/
    └── values.yaml          # Helm Chart 配置
```

## 环境管理脚本

使用 `scripts/env-manager.sh` 脚本进行环境变量管理：

### 基本命令

```bash
# 查看帮助
./scripts/env-manager.sh help

# 切换到开发环境
./scripts/env-manager.sh switch dev

# 切换到生产环境
./scripts/env-manager.sh switch prod

# 验证当前环境配置
./scripts/env-manager.sh validate

# 同步环境变量到 Helm
./scripts/env-manager.sh helm-sync

# 比较开发和生产环境差异
./scripts/env-manager.sh compare

# 创建备份
./scripts/env-manager.sh backup

# 恢复备份
./scripts/env-manager.sh restore <backup-name>
```

### 高级选项

```bash
# 强制切换（跳过确认）
./scripts/env-manager.sh switch prod --force

# 预览模式（即将支持）
./scripts/env-manager.sh switch prod --dry-run
```

## 部署方式

### Docker Compose 部署

```bash
# 1. 切换到所需环境
./scripts/env-manager.sh switch dev

# 2. 验证配置
./scripts/env-manager.sh validate

# 3. 启动服务
docker-compose up -d
```

### Helm 部署

```bash
# 1. 切换到所需环境
./scripts/env-manager.sh switch prod

# 2. 同步到 Helm
./scripts/env-manager.sh helm-sync

# 3. 部署到 Kubernetes
helm install ai-infra ./helm/ai-infra-matrix
```

## 环境配置详解

### 开发环境特点

- `BUILD_ENV=development`
- `DEBUG_MODE=true`
- `LOG_LEVEL=debug`
- 使用简单密码（如 `postgres`）
- 启用开发调试功能
- 单副本部署
- 本地存储

### 生产环境特点

- `BUILD_ENV=production`
- `DEBUG_MODE=false`
- `LOG_LEVEL=info`
- 使用复杂密码（需要修改）
- 禁用调试功能
- 多副本高可用部署
- MinIO 存储
- SSL/TLS 安全配置

## 安全考虑

### 生产环境密码修改

生产环境配置中所有包含 `CHANGE_IN_PRODUCTION` 的变量都需要在部署前修改：

```bash
# 数据库密码
POSTGRES_PASSWORD=CHANGE_IN_PRODUCTION_PostgreSQL_2024!

# Redis 密码
REDIS_PASSWORD=CHANGE_IN_PRODUCTION_Redis_2024!

# LDAP 密码
LDAP_ADMIN_PASSWORD=CHANGE_IN_PRODUCTION_LDAP_2024!

# JWT 密钥
JWT_SECRET=CHANGE_IN_PRODUCTION_JWT_SECRET_2024_RANDOM_STRING_HERE

# 其他关键密钥...
```

### 自动化密码生成

可以使用以下命令生成安全密码：

```bash
# 生成随机密码
openssl rand -base64 32

# 生成 JWT 密钥
openssl rand -hex 64

# 生成 Crypt Key
openssl rand -hex 32
```

## 配置验证

### 环境变量检查

环境管理脚本会自动验证：

1. 必需文件是否存在
2. 关键环境变量是否配置
3. Docker Compose 配置是否有效
4. 生产环境密码安全性

### 手动验证

```bash
# 检查当前环境
grep "BUILD_ENV" .env

# 验证 Docker Compose 配置
docker-compose config

# 验证 Helm 配置
helm template ./helm/ai-infra-matrix
```

## 故障排除

### 常见问题

1. **环境切换失败**
   ```bash
   # 检查文件权限
   ls -la .env*
   
   # 手动恢复备份
   ./scripts/env-manager.sh restore <backup-name>
   ```

2. **Docker Compose 启动失败**
   ```bash
   # 验证配置
   ./scripts/env-manager.sh validate
   
   # 检查环境变量
   docker-compose config
   ```

3. **Helm 部署失败**
   ```bash
   # 重新同步环境变量
   ./scripts/env-manager.sh helm-sync
   
   # 检查 values.yaml
   helm template ./helm/ai-infra-matrix
   ```

### 日志位置

- 备份目录：`backup/`
- 脚本日志：终端输出
- Docker 日志：`docker-compose logs`
- Kubernetes 日志：`kubectl logs`

## 最佳实践

### 开发流程

1. 始终在切换环境前创建备份
2. 使用 `validate` 命令验证配置
3. 在生产环境中修改所有默认密码
4. 定期比较环境差异

### 部署流程

1. **开发环境**：
   ```bash
   ./scripts/env-manager.sh switch dev --force
   docker-compose up -d
   ```

2. **生产环境**：
   ```bash
   ./scripts/env-manager.sh switch prod
   # 修改生产密码
   ./scripts/env-manager.sh validate
   ./scripts/env-manager.sh helm-sync
   helm install ai-infra ./helm/ai-infra-matrix
   ```

### 备份策略

- 环境切换前自动备份
- 定期手动备份重要配置
- 保留多个历史版本
- 测试备份恢复流程

## 版本历史

- **v3.1.0**: 实现环境变量统一配置
- **v3.0.x**: 多环境文件分离管理
- **v2.x.x**: 基础环境配置

## 相关文档

- [Docker Compose 配置文档](./DOCKER_COMPOSE_GUIDE.md)
- [Helm Chart 部署指南](./HELM_DEPLOYMENT_GUIDE.md)
- [安全配置指南](./SECURITY_GUIDE.md)
- [故障排除手册](./TROUBLESHOOTING.md)
