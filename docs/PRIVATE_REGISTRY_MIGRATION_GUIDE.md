# 私有仓库镜像迁移指南

## 概述

本指南说明如何将AI基础设施矩阵项目所需的所有Docker镜像从公共仓库迁移到私有Harbor仓库，以支持离线环境部署。

## 镜像映射表

### 源码镜像（需要构建和推送）
这些镜像由项目源码构建，使用 `./build.sh build-push` 命令处理：

```
原始镜像                              → 私有仓库镜像
ai-infra-backend-init:v0.3.5         → aiharbor.msxf.local/aihpc/ai-infra-backend-init:v0.3.5
ai-infra-backend:v0.3.5              → aiharbor.msxf.local/aihpc/ai-infra-backend:v0.3.5
ai-infra-frontend:v0.3.5             → aiharbor.msxf.local/aihpc/ai-infra-frontend:v0.3.5
ai-infra-jupyterhub:v0.3.5           → aiharbor.msxf.local/aihpc/ai-infra-jupyterhub:v0.3.5
ai-infra-singleuser:v0.3.5           → aiharbor.msxf.local/aihpc/ai-infra-singleuser:v0.3.5
ai-infra-saltstack:v0.3.5            → aiharbor.msxf.local/aihpc/ai-infra-saltstack:v0.3.5
ai-infra-nginx:v0.3.5                → aiharbor.msxf.local/aihpc/ai-infra-nginx:v0.3.5
ai-infra-gitea:v0.3.5                → aiharbor.msxf.local/aihpc/ai-infra-gitea:v0.3.5
```

### 基础镜像（需要从公网拉取并重新标签）
这些镜像需要从公共仓库拉取，然后推送到私有仓库：

```
公共镜像                              → 私有仓库镜像
postgres:15-alpine                   → aiharbor.msxf.local/aihpc/library/postgres:15-alpine
redis:7-alpine                       → aiharbor.msxf.local/aihpc/library/redis:7-alpine
nginx:1.27-alpine                    → aiharbor.msxf.local/aihpc/library/nginx:1.27-alpine
tecnativa/tcp-proxy:latest           → aiharbor.msxf.local/aihpc/tecnativa/tcp-proxy:latest
redislabs/redisinsight:latest        → aiharbor.msxf.local/aihpc/redislabs/redisinsight:latest
quay.io/minio/minio:latest           → aiharbor.msxf.local/aihpc/minio/minio:latest
```

## 迁移步骤

### 1. 准备Harbor私有仓库

确保Harbor私有仓库已经安装并配置，创建以下项目：
- `aihpc` (主项目)
- `library` (官方基础镜像)
- `tecnativa` (第三方镜像)
- `redislabs` (Redis相关镜像)
- `minio` (MinIO镜像)

### 2. 登录Harbor仓库

```bash
# 登录私有Harbor仓库
docker login aiharbor.msxf.local
```

### 3. 迁移基础镜像脚本

创建并运行以下脚本来迁移基础镜像：

```bash
#!/bin/bash
# 基础镜像迁移脚本

REGISTRY="aiharbor.msxf.local"

echo "=== 开始迁移基础镜像到私有仓库 ==="

# 镜像映射表
declare -A images=(
    ["postgres:15-alpine"]="$REGISTRY/aihpc/library/postgres:15-alpine"
    ["redis:7-alpine"]="$REGISTRY/aihpc/library/redis:7-alpine"
    ["nginx:1.27-alpine"]="$REGISTRY/aihpc/library/nginx:1.27-alpine"
    ["tecnativa/tcp-proxy:latest"]="$REGISTRY/aihpc/tecnativa/tcp-proxy:latest"
    ["redislabs/redisinsight:latest"]="$REGISTRY/aihpc/redislabs/redisinsight:latest"
    ["quay.io/minio/minio:latest"]="$REGISTRY/aihpc/minio/minio:latest"
)

# 拉取、标签和推送镜像
for source in "${!images[@]}"; do
    target="${images[$source]}"
    echo "处理镜像: $source → $target"
    
    # 拉取源镜像
    echo "  拉取: $source"
    docker pull "$source"
    
    # 重新标签
    echo "  标签: $target"
    docker tag "$source" "$target"
    
    # 推送到私有仓库
    echo "  推送: $target"
    docker push "$target"
    
    # 清理本地镜像（可选）
    # docker rmi "$source" "$target"
    
    echo "  ✓ 完成: $source"
    echo
done

echo "=== 基础镜像迁移完成 ==="
```

### 4. 构建和推送源码镜像

```bash
# 构建并推送所有源码镜像到私有仓库
./build.sh build-push aiharbor.msxf.local/aihpc v0.3.5
```

### 5. 验证镜像可用性

```bash
# 验证所有镜像都已推送成功
echo "=== 验证私有仓库镜像 ==="

# 检查源码镜像
for service in backend-init backend frontend jupyterhub singleuser saltstack nginx gitea; do
    echo "检查: aiharbor.msxf.local/aihpc/ai-infra-$service:v0.3.5"
    docker pull "aiharbor.msxf.local/aihpc/ai-infra-$service:v0.3.5" --quiet
    echo "  ✓ 可用"
done

# 检查基础镜像
declare -a base_images=(
    "aiharbor.msxf.local/aihpc/library/postgres:15-alpine"
    "aiharbor.msxf.local/aihpc/library/redis:7-alpine"
    "aiharbor.msxf.local/aihpc/library/nginx:1.27-alpine"
    "aiharbor.msxf.local/aihpc/tecnativa/tcp-proxy:latest"
    "aiharbor.msxf.local/aihpc/redislabs/redisinsight:latest"
    "aiharbor.msxf.local/aihpc/minio/minio:latest"
)

for image in "${base_images[@]}"; do
    echo "检查: $image"
    docker pull "$image" --quiet
    echo "  ✓ 可用"
done

echo "=== 所有镜像验证完成 ==="
```

## 使用私有仓库部署

完成镜像迁移后，可以使用以下命令在离线环境中部署：

```bash
# 生成使用私有仓库的生产配置
./build.sh prod-generate aiharbor.msxf.local/aihpc v0.3.5

# 启动生产环境
./build.sh prod-up aiharbor.msxf.local/aihpc v0.3.5

# 检查服务状态
./build.sh prod-status
```

## 故障排除

### 常见问题

1. **镜像拉取失败**
   ```
   Error: unknown: repository xxx not found
   ```
   **解决方案**: 确保镜像已正确推送到私有仓库，检查镜像名称和标签是否正确。

2. **认证失败**
   ```
   Error: unauthorized: authentication required
   ```
   **解决方案**: 重新登录Harbor仓库，确保有推送权限。

3. **网络连接问题**
   ```
   Error: dial tcp: lookup aiharbor.msxf.local
   ```
   **解决方案**: 确保DNS解析正确，网络连接正常。

### 调试命令

```bash
# 检查Docker登录状态
docker system info | grep Registry

# 查看本地镜像
docker images | grep aiharbor.msxf.local

# 手动测试拉取
docker pull aiharbor.msxf.local/aihpc/library/postgres:15-alpine

# 检查配置文件中的镜像
grep "image:" docker-compose.prod.yml
```

## 注意事项

1. **存储空间**: 确保私有仓库有足够的存储空间来存储所有镜像
2. **网络带宽**: 初始镜像同步可能需要较长时间，建议在网络条件良好时进行
3. **版本管理**: 建议定期更新基础镜像的安全补丁
4. **备份策略**: 制定镜像备份和恢复策略
5. **访问权限**: 合理配置Harbor项目的访问权限

## 自动化脚本

可以将上述步骤整合为自动化脚本，放置在 `scripts/migrate-to-private-registry.sh`：

```bash
#!/bin/bash
# 完整的私有仓库迁移脚本
# 使用方法: ./migrate-to-private-registry.sh aiharbor.msxf.local/aihpc v0.3.5

set -e

REGISTRY_BASE="$1"
TAG="$2"

if [[ -z "$REGISTRY_BASE" ]] || [[ -z "$TAG" ]]; then
    echo "使用方法: $0 <registry_base> <tag>"
    echo "示例: $0 aiharbor.msxf.local/aihpc v0.3.5"
    exit 1
fi

echo "=== 开始私有仓库迁移 ==="
echo "目标仓库: $REGISTRY_BASE"
echo "镜像标签: $TAG"
echo

# 1. 迁移基础镜像
echo "步骤 1: 迁移基础镜像..."
# (此处插入基础镜像迁移脚本)

# 2. 构建和推送源码镜像
echo "步骤 2: 构建和推送源码镜像..."
./build.sh build-push "$REGISTRY_BASE" "$TAG"

# 3. 验证镜像可用性
echo "步骤 3: 验证镜像可用性..."
# (此处插入验证脚本)

# 4. 生成生产配置
echo "步骤 4: 生成生产配置..."
./build.sh prod-generate "$REGISTRY_BASE" "$TAG"

echo "=== 私有仓库迁移完成 ==="
echo "现在可以使用以下命令启动生产环境:"
echo "  ./build.sh prod-up $REGISTRY_BASE $TAG"
```
