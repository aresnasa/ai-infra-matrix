#!/bin/bash
# 本地镜像验证脚本 - AI Infrastructure Matrix
# 用于验证所有镜像是否已正确标记在本地，无需网络连接

set -e

# 配置
REGISTRY_BASE="${1:-aiharbor.msxf.local/aihpc}"
TAG="${2:-v0.3.5}"

if [[ -z "$1" ]]; then
    echo "使用方法: $0 <registry_base> [tag]"
    echo "示例: $0 aiharbor.msxf.local/aihpc v0.3.5"
    exit 1
fi

echo "=== AI Infrastructure Matrix 本地镜像验证 ==="
echo "目标仓库: $REGISTRY_BASE"
echo "镜像标签: $TAG"
echo "开始时间: $(date)"
echo

echo "📋 本地镜像检查（不需要网络连接）："
echo "验证所有必需的镜像是否已在本地正确标记"
echo

# 源码镜像列表
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

# 基础镜像列表 - 使用统一的 aihpc 项目格式
declare -a base_images=(
    "$REGISTRY_BASE/postgres:$TAG"
    "$REGISTRY_BASE/redis:$TAG"
    "$REGISTRY_BASE/nginx:$TAG"
    "$REGISTRY_BASE/tcp-proxy:$TAG"
    "$REGISTRY_BASE/redisinsight:$TAG"
    "$REGISTRY_BASE/minio:$TAG"
    "$REGISTRY_BASE/openldap:$TAG"
    "$REGISTRY_BASE/phpldapadmin:$TAG"
)

# 统计变量
total_images=$((${#source_images[@]} + ${#base_images[@]}))
verified_count=0
failed_images=()

echo "计划验证 $total_images 个镜像"
echo "============================================"

# 验证源码镜像
echo "验证源码镜像 (${#source_images[@]} 个):"
for service in "${source_images[@]}"; do
    image="$REGISTRY_BASE/$service:$TAG"
    echo "  检查: $image"
    
    if docker image inspect "$image" >/dev/null 2>&1; then
        echo "    ✓ 本地可用"
        verified_count=$((verified_count + 1))
    else
        echo "    ✗ 本地不可用"
        failed_images+=("$image")
    fi
done

echo

# 验证基础镜像
echo "验证基础镜像 (${#base_images[@]} 个):"
for image in "${base_images[@]}"; do
    echo "  检查: $image"
    
    if docker image inspect "$image" >/dev/null 2>&1; then
        echo "    ✓ 本地可用"
        verified_count=$((verified_count + 1))
    else
        echo "    ✗ 本地不可用"
        failed_images+=("$image")
    fi
done

echo
echo "============================================"
echo "验证结果汇总:"
echo "总计镜像: $total_images"
echo "验证通过: $verified_count"
echo "验证失败: ${#failed_images[@]}"

if [[ ${#failed_images[@]} -gt 0 ]]; then
    echo
    echo "缺失镜像列表:"
    for failed in "${failed_images[@]}"; do
        echo "  ✗ $failed"
    done
    echo
    echo "建议操作:"
    echo "1. 重新运行镜像重新标记脚本:"
    echo "   ./scripts/retag-for-harbor-structure.sh $REGISTRY_BASE $TAG"
    echo "2. 检查基础镜像是否已构建:"
    echo "   ./build.sh deps-all $REGISTRY_BASE $TAG"
    echo "3. 重新构建源码镜像:"
    echo "   ./build.sh build-all $REGISTRY_BASE $TAG"
    echo
    exit 1
else
    echo
    echo "🎉 所有镜像本地验证通过！"
    echo
    echo "现在可以部署生产环境:"
    echo "  ./build.sh prod-up --force $REGISTRY_BASE $TAG"
    echo
    echo "或生成生产配置后部署:"
    echo "  ./build.sh prod-generate $REGISTRY_BASE $TAG"
    echo "  ./build.sh prod-up --force $REGISTRY_BASE $TAG"
fi

echo
echo "结束时间: $(date)"
echo "=== 本地镜像验证完成 ==="
