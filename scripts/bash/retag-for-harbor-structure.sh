#!/bin/bash
# 重新标记镜像以匹配Harbor项目结构
# 用于本地测试和验证，无需实际推送到Harbor

set -e

REGISTRY_BASE="${1:-aiharbor.msxf.local/aihpc}"
TAG="${2:-v0.3.5}"

if [[ -z "$1" ]]; then
    echo "使用方法: $0 <registry_base> [tag]"
    echo "示例: $0 aiharbor.msxf.local/aihpc v0.3.5"
    exit 1
fi

echo "=== AI Infrastructure Matrix 镜像重新标记 ==="
echo "目标仓库: $REGISTRY_BASE"
echo "镜像标签: $TAG"
echo "开始时间: $(date)"
echo

# 函数：安全标记镜像
safe_tag_image() {
    local source_image="$1"
    local target_image="$2"
    
    if docker image inspect "$source_image" >/dev/null 2>&1; then
        echo "  ✓ 找到源镜像: $source_image"
        docker tag "$source_image" "$target_image"
        echo "  → 已标记为: $target_image"
        return 0
    else
        echo "  ✗ 未找到源镜像: $source_image"
        return 1
    fi
}

echo "📋 重新标记基础镜像以匹配Harbor项目结构："
echo

# 基础镜像重新标记
echo "1. PostgreSQL:"
safe_tag_image "$REGISTRY_BASE/postgres:$TAG" "$REGISTRY_BASE/library/postgres:15-alpine"

echo
echo "2. Redis:"
safe_tag_image "$REGISTRY_BASE/redis:$TAG" "$REGISTRY_BASE/library/redis:7-alpine"

echo
echo "3. Nginx:"
safe_tag_image "$REGISTRY_BASE/nginx:$TAG" "$REGISTRY_BASE/library/nginx:1.27-alpine"

echo
echo "4. TCP Proxy:"
# 尝试从不同可能的源标记
if docker image inspect "tecnativa/tcp-proxy:latest" >/dev/null 2>&1; then
    docker tag "tecnativa/tcp-proxy:latest" "$REGISTRY_BASE/tecnativa/tcp-proxy:latest"
    echo "  ✓ 已标记 tecnativa/tcp-proxy:latest → $REGISTRY_BASE/tecnativa/tcp-proxy:latest"
elif docker image inspect "$REGISTRY_BASE/tcp-proxy:$TAG" >/dev/null 2>&1; then
    docker tag "$REGISTRY_BASE/tcp-proxy:$TAG" "$REGISTRY_BASE/tecnativa/tcp-proxy:latest"
    echo "  ✓ 已标记 $REGISTRY_BASE/tcp-proxy:$TAG → $REGISTRY_BASE/tecnativa/tcp-proxy:latest"
else
    echo "  ✗ 未找到 TCP Proxy 镜像"
fi

echo
echo "5. RedisInsight:"
safe_tag_image "$REGISTRY_BASE/redisinsight:$TAG" "$REGISTRY_BASE/redislabs/redisinsight:latest"

echo
echo "6. MinIO:"
safe_tag_image "$REGISTRY_BASE/minio:$TAG" "$REGISTRY_BASE/minio/minio:latest"

echo
echo "📋 检查源码镜像标记状态："
echo

# 检查源码镜像（这些应该已经正确标记）
declare -a source_images=(
    "ai-infra-backend-init"
    "ai-infra-backend" 
    "ai-infra-frontend"
    "ai-infra-jupyterhub"
    "ai-infra-singleuser"
    "ai-infra-saltstack"
    "ai-infra-nginx"
    "ai-infra-gitea"
)

for service in "${source_images[@]}"; do
    target_image="$REGISTRY_BASE/$service:$TAG"
    if docker image inspect "$target_image" >/dev/null 2>&1; then
        echo "  ✓ $service: 已正确标记"
    else
        echo "  ✗ $service: 缺失标记"
        # 尝试从本地镜像标记
        if docker image inspect "$service:$TAG" >/dev/null 2>&1; then
            docker tag "$service:$TAG" "$target_image"
            echo "    → 已从 $service:$TAG 重新标记"
        fi
    fi
done

echo
echo "============================================"
echo "重新标记完成！"
echo
echo "现在可以重新运行验证脚本："
echo "  ./scripts/verify-private-images.sh $REGISTRY_BASE $TAG"
echo
echo "或直接运行生产环境："
echo "  ./build.sh prod-up --force $REGISTRY_BASE $TAG"
echo
echo "结束时间: $(date)"
echo "=== 镜像重新标记完成 ==="
