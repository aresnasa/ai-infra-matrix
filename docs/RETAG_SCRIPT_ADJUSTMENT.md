# 镜像标记脚本调整说明

## 调整概述

根据用户要求，已将镜像重新标记脚本调整为：
- **原始镜像**: 公有基础镜像（如 postgres:15-alpine, redis:7-alpine 等）
- **目标镜像**: 内网仓库 `aiharbor.msxf.local/aihpc` 命名空间

## 主要调整内容

### 1. 基础镜像映射调整

将所有公有基础镜像统一映射到 `aiharbor.msxf.local/aihpc` 命名空间：

| 原始公有镜像 | 目标内网镜像 |
|-------------|------------|
| postgres:15-alpine | aiharbor.msxf.local/aihpc/postgres:v0.3.5 |
| redis:7-alpine | aiharbor.msxf.local/aihpc/redis:v0.3.5 |
| nginx:1.27-alpine | aiharbor.msxf.local/aihpc/nginx:v0.3.5 |
| quay.io/minio/minio:latest | aiharbor.msxf.local/aihpc/minio:v0.3.5 |
| tecnativa/tcp-proxy:latest | aiharbor.msxf.local/aihpc/tcp-proxy:v0.3.5 |
| redislabs/redisinsight:latest | aiharbor.msxf.local/aihpc/redisinsight:v0.3.5 |

### 2. 已有内网镜像重新标记

将现有的不同命名空间镜像统一到 `aihpc` 命名空间：

```bash
# 从 library 命名空间迁移
aiharbor.msxf.local/library/postgres:v0.3.5 → aiharbor.msxf.local/aihpc/postgres:v0.3.5
aiharbor.msxf.local/library/redis:v0.3.5 → aiharbor.msxf.local/aihpc/redis:v0.3.5
aiharbor.msxf.local/library/nginx:v0.3.5 → aiharbor.msxf.local/aihpc/nginx:v0.3.5

# 从 minio 命名空间迁移
aiharbor.msxf.local/minio/minio:v0.3.5 → aiharbor.msxf.local/aihpc/minio:v0.3.5
```

### 3. Docker Compose 配置更新

`docker-compose.prod-test.yml` 文件中的镜像引用已更新：

```yaml
services:
  postgres:
    image: aiharbor.msxf.local/aihpc/postgres:v0.3.5  # 已更新
    
  redis:
    image: aiharbor.msxf.local/aihpc/redis:v0.3.5     # 已更新
    
  # 其他服务...
```

## 脚本功能验证

### 执行结果

✅ **主要镜像标记**: 8/8 成功
- ai-infra-backend:v0.3.5
- ai-infra-frontend:v0.3.5
- ai-infra-nginx:v0.3.5
- ai-infra-jupyterhub:v0.3.5
- ai-infra-gitea:v0.3.5
- ai-infra-singleuser:v0.3.5
- ai-infra-saltstack:v0.3.5
- ai-infra-backend-init:v0.3.5

✅ **依赖镜像标记**: 成功标记多个基础镜像
- postgres 相关镜像
- redis 相关镜像
- nginx 相关镜像
- minio 相关镜像
- tcp-proxy 相关镜像
- redisinsight 相关镜像

### 验证命令

```bash
# 查看所有aihpc命名空间的镜像
docker images | grep "aiharbor.msxf.local/aihpc"

# 查看基础镜像标记结果
docker images | grep "aiharbor.msxf.local/aihpc" | grep -E "(postgres|redis|nginx|minio|tcp-proxy|redisinsight)"
```

## 使用指南

### 1. 重新标记镜像

```bash
# 标记主要AI-Infra镜像
./scripts/retag-images-for-prod.sh

# 包含依赖镜像的完整标记
./scripts/retag-images-for-prod.sh --deps

# 预览操作（不实际执行）
./scripts/retag-images-for-prod.sh --deps --dry-run
```

### 2. 启动生产测试环境

```bash
# 启动环境
./scripts/start-prod-test.sh start

# 检查状态
./scripts/start-prod-test.sh status

# 查看服务地址
./scripts/start-prod-test.sh urls
```

## 优势

1. **统一命名空间**: 所有镜像统一在 `aiharbor.msxf.local/aihpc` 命名空间下
2. **内网部署**: 无需依赖外部公有仓库，提高部署稳定性
3. **版本控制**: 统一使用 v0.3.5 标签，便于版本管理
4. **自动化**: 脚本自动处理多种来源镜像的标记
5. **向后兼容**: 保持与现有配置的兼容性

## 注意事项

1. **网络要求**: 确保能够访问 `aiharbor.msxf.local` 内网仓库
2. **权限配置**: 确保有推送到目标仓库的权限
3. **存储空间**: 标记操作会创建新的镜像引用，注意磁盘空间
4. **镜像清理**: 可根据需要清理不再使用的旧镜像标记

## 下一步

1. 测试生产环境部署
2. 验证所有服务功能
3. 根据需要调整配置
4. 制定镜像更新策略
