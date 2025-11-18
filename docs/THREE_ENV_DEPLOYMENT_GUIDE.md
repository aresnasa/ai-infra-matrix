# AI Infrastructure Matrix 三环境部署指南

## 概述

AI Infrastructure Matrix 支持三种环境的统一部署管理：

1. **开发环境 (Development)** - 本地开发和测试
2. **CI/CD环境 (CI/CD Server)** - 镜像构建和转发
3. **生产环境 (Production)** - 内网隔离部署

## 环境架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   开发环境       │    │   CI/CD服务器    │    │   生产环境       │
│                │    │                │    │                │
│ • 本地开发       │    │ • 外网连接       │    │ • 内网隔离       │
│ • 镜像构建       │────▶│ • 镜像转发       │────▶│ • 服务部署       │
│ • 功能测试       │    │ • 配置打包       │    │ • 运行维护       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## 快速开始

### 1. 开发环境使用

```bash
# 设置环境类型
export AI_INFRA_ENV_TYPE=development

# 构建所有镜像
./build.sh build v0.3.5

# 启动开发环境
./build.sh dev-start

# 停止服务
./build.sh dev-stop
```

### 2. CI/CD服务器使用

```bash
# 设置环境类型
export AI_INFRA_ENV_TYPE=cicd

# 转发镜像到内网仓库
./build.sh transfer registry.internal.com/ai-infra v0.3.5

# 打包所有配置文件
./build.sh package registry.internal.com/ai-infra v0.3.5

# 清理本地镜像（可选）
./build.sh transfer registry.internal.com/ai-infra v0.3.5 --cleanup
```

### 3. 生产环境使用

```bash
# 设置环境类型
export AI_INFRA_ENV_TYPE=production

# 使用 Docker Compose 部署
./build.sh deploy-compose registry.internal.com/ai-infra v0.3.5

# 或使用 Kubernetes 部署
./build.sh deploy-helm registry.internal.com/ai-infra v0.3.5
```

## 环境配置

### 环境类型设置

1. **环境变量方式**：
   ```bash
   export AI_INFRA_ENV_TYPE=development  # 或 cicd, production
   ```

2. **系统配置文件**：
   ```bash
   # 在目标服务器创建环境标识文件
   echo "production" | sudo tee /etc/ai-infra-env
   ```

3. **自动检测**：
   - 检测到 Kubernetes 环境 → `production`
   - 检测到 CI/CD 环境变量 → `cicd`
   - 默认 → `development`

### 配置文件说明

| 文件 | 用途 | 环境 |
|------|------|------|
| `.env` | 开发/CI/CD环境配置 | development, cicd |
| `.env.prod` | 生产环境配置 | production |
| `.env.unified` | 开发环境模板 | development |
| `.env.prod.unified` | 生产环境模板 | production |

## 详细使用说明

### 开发环境 (Development)

**目标**：本地开发和功能验证

```bash
# 1. 初始化环境
export AI_INFRA_ENV_TYPE=development
./build.sh env  # 查看当前配置

# 2. 构建镜像
./build.sh build v0.3.5

# 3. 启动服务
./build.sh dev-start

# 4. 查看服务状态
docker-compose ps

# 5. 停止服务
./build.sh dev-stop
```

**特点**：
- 使用本地Docker构建
- 启用调试模式
- 简单密码配置
- 单副本部署

### CI/CD环境 (CI/CD Server)

**目标**：镜像转发和配置打包

```bash
# 1. 设置环境
export AI_INFRA_ENV_TYPE=cicd
export CLEANUP_LOCAL_IMAGES=true  # 可选：清理本地镜像

# 2. 转发镜像到内网仓库
./build.sh transfer registry.internal.com/ai-infra v0.3.5

# 3. 打包配置文件
./build.sh package registry.internal.com/ai-infra v0.3.5

# 4. 生成的部署包
ls -la ai-infra-deploy-v0.3.5.tar.gz
```

**功能**：
- 从外网拉取镜像
- 推送到内网仓库
- 打包部署配置
- 可选清理本地镜像

### 生产环境 (Production)

**目标**：内网隔离部署

#### Docker Compose 部署

```bash
# 1. 设置环境
export AI_INFRA_ENV_TYPE=production

# 2. 部署服务
./build.sh deploy-compose registry.internal.com/ai-infra v0.3.5

# 3. 查看服务状态
docker-compose ps
```

#### Kubernetes 部署

```bash
# 1. 设置环境
export AI_INFRA_ENV_TYPE=production

# 2. 检查集群连接
kubectl cluster-info

# 3. 部署到 Kubernetes
./build.sh deploy-helm registry.internal.com/ai-infra v0.3.5

# 4. 查看部署状态
kubectl get pods -n ai-infra-prod
kubectl get services -n ai-infra-prod
```

**特点**：
- 使用内网仓库镜像
- 生产级安全配置
- 多副本高可用
- 完整监控日志

## 高级配置

### 强制模式

在非目标环境中执行特定命令：

```bash
# 在开发环境中执行镜像转发
./build.sh transfer registry.internal.com/ai-infra v0.3.5 --force

# 在CI/CD环境中部署服务
./build.sh deploy-compose registry.internal.com/ai-infra v0.3.5 --force
```

### 详细输出

```bash
# 启用详细日志
./build.sh build v0.3.5 --verbose
```

### 预览模式

```bash
# 预览操作（暂未实现）
./build.sh deploy-helm registry.internal.com/ai-infra v0.3.5 --dry-run
```

## 生产环境安全配置

### 密码修改

生产环境部署前，必须修改 `.env.prod` 中的密码：

```bash
# 生成安全密码
openssl rand -base64 32

# 需要修改的配置项
POSTGRES_PASSWORD=CHANGE_IN_PRODUCTION_PostgreSQL_2024!
REDIS_PASSWORD=CHANGE_IN_PRODUCTION_Redis_2024!
LDAP_ADMIN_PASSWORD=CHANGE_IN_PRODUCTION_LDAP_2024!
JWT_SECRET=CHANGE_IN_PRODUCTION_JWT_SECRET_2024_RANDOM_STRING_HERE
```

### 网络安全

```bash
# 限制信任的代理IP
REVERSE_PROXY_TRUSTED_PROXIES=172.16.0.0/12,192.168.0.0/16,10.0.0.0/8

# 启用SSL/TLS
SSL_ENABLED=true
SSL_REDIRECT=true
HSTS_ENABLED=true
```

## 故障排除

### 常见问题

1. **环境检测错误**
   ```bash
   ./build.sh env  # 查看当前环境配置
   export AI_INFRA_ENV_TYPE=production  # 手动设置
   ```

2. **Docker 连接失败**
   ```bash
   docker info  # 检查Docker状态
   sudo systemctl start docker  # 启动Docker服务
   ```

3. **镜像推送失败**
   ```bash
   docker login registry.internal.com  # 登录私有仓库
   ```

4. **Kubernetes 部署失败**
   ```bash
   kubectl cluster-info  # 检查集群连接
   helm version  # 检查Helm版本
   ```

### 日志查看

```bash
# Docker Compose 日志
docker-compose logs -f [service_name]

# Kubernetes 日志
kubectl logs -f deployment/[deployment_name] -n ai-infra-prod
```

## 最佳实践

### 开发流程

1. 在开发环境进行功能开发和测试
2. 使用 CI/CD 服务器构建和转发镜像
3. 在生产环境进行部署和运维

### 版本管理

```bash
# 使用Git标签作为镜像版本
IMAGE_TAG=$(git describe --tags --always)
./build.sh build $IMAGE_TAG
```

### 备份策略

```bash
# 环境配置备份
./scripts/env-manager.sh backup

# 数据库备份（生产环境）
kubectl exec -n ai-infra-prod deployment/postgres -- pg_dump > backup.sql
```

### 监控建议

1. 配置健康检查端点
2. 设置资源使用告警
3. 监控关键服务状态
4. 定期检查安全配置

## 相关文档

- [环境变量统一配置指南](./ENV_UNIFIED_GUIDE.md)
- [Docker Compose 部署文档](./DOCKER_COMPOSE_GUIDE.md)
- [Kubernetes Helm 部署指南](./HELM_DEPLOYMENT_GUIDE.md)
- [安全配置指南](./SECURITY_GUIDE.md)
