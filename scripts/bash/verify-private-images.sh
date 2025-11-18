#!/bin/bash
# 私有仓库镜像验证脚本 - AI Infrastructure Matrix
# 用于验证所有镜像是否已正确推送到私有仓库

set -e

# 配置
REGISTRY_BASE="${1:-aiharbor.msxf.local/aihpc}"
TAG="${2:-v0.3.5}"

if [[ -z "$1" ]]; then
    echo "使用方法: $0 <registry_base> [tag]"
    echo "示例: $0 aiharbor.msxf.local/aihpc v0.3.5"
    exit 1
fi

echo "=== AI Infrastructure Matrix 镜像验证 ==="
echo "目标仓库: $REGISTRY_BASE"
echo "镜像标签: $TAG"
echo "开始时间: $(date)"
echo

echo "📋 Harbor项目检查："
echo "验证前请确保以下项目已在Harbor中创建："
echo "  • aihpc (主项目)"
echo "  • library (基础镜像)"
echo "  • tecnativa (第三方镜像)"
echo "  • redislabs (第三方镜像)"
echo "  • minio (第三方镜像)"
echo
echo "如未创建，请参考: docs/HARBOR_PROJECT_SETUP.md"
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

# 基础镜像列表
declare -a base_images=(
    "$REGISTRY_BASE/library/postgres:15-alpine"
    "$REGISTRY_BASE/library/redis:7-alpine"
    "$REGISTRY_BASE/library/nginx:1.27-alpine"
    "$REGISTRY_BASE/tecnativa/tcp-proxy:latest"
    "$REGISTRY_BASE/redislabs/redisinsight:latest"
    "$REGISTRY_BASE/minio/minio:latest"
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
    
    if docker pull "$image" --quiet >/dev/null 2>&1; then
        echo "    ✓ 可用"
        verified_count=$((verified_count + 1))
    else
        echo "    ✗ 不可用"
        failed_images+=("$image")
    fi
done

echo

# 验证基础镜像
echo "验证基础镜像 (${#base_images[@]} 个):"
for image in "${base_images[@]}"; do
    echo "  检查: $image"
    
    if docker pull "$image" --quiet >/dev/null 2>&1; then
        echo "    ✓ 可用"
        verified_count=$((verified_count + 1))
    else
        echo "    ✗ 不可用"
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
    echo "失败镜像列表:"
    for failed in "${failed_images[@]}"; do
        echo "  ✗ $failed"
    done
    echo
    echo "建议操作:"
    echo "1. 检查网络连接和仓库权限"
    echo "2. 重新运行基础镜像迁移脚本:"
    echo "   ./scripts/migrate-base-images.sh $REGISTRY_BASE"
    echo "3. 重新构建和推送源码镜像:"
    echo "   ./build.sh build-push $REGISTRY_BASE $TAG"
    echo
    exit 1
else
    echo
    echo "🎉 所有镜像验证通过！"
    echo
    echo "现在可以部署生产环境:"
    echo "  ./build.sh prod-up $REGISTRY_BASE $TAG"
fi

echo
echo "结束时间: $(date)"
echo "=== 镜像验证完成 ==="
