#!/bin/bash
# 基础镜像迁移脚本 - AI Infrastructure Matrix
# 用于将所有基础镜像从公共仓库迁移到私有Harbor仓库

set -e

# 配置
REGISTRY_BASE="${1:-aiharbor.msxf.local/aihpc}"

if [[ -z "$1" ]]; then
    echo "使用方法: $0 <registry_base>"
    echo "示例: $0 aiharbor.msxf.local/aihpc"
    exit 1
fi

echo "=== AI Infrastructure Matrix 基础镜像迁移 ==="
echo "目标仓库: $REGISTRY_BASE"
echo "开始时间: $(date)"
echo

# 镜像映射表
declare -A images=(
    ["postgres:15-alpine"]="$REGISTRY_BASE/library/postgres:15-alpine"
    ["redis:7-alpine"]="$REGISTRY_BASE/library/redis:7-alpine"
    ["nginx:1.27-alpine"]="$REGISTRY_BASE/library/nginx:1.27-alpine"
    ["tecnativa/tcp-proxy:latest"]="$REGISTRY_BASE/tecnativa/tcp-proxy:latest"
    ["redislabs/redisinsight:latest"]="$REGISTRY_BASE/redislabs/redisinsight:latest"
    ["quay.io/minio/minio:latest"]="$REGISTRY_BASE/minio/minio:latest"
)

# 统计信息
total_images=${#images[@]}
current_count=0
failed_images=()

echo "计划迁移 $total_images 个基础镜像"
echo "============================================"

# 拉取、标签和推送镜像
for source in "${!images[@]}"; do
    target="${images[$source]}"
    current_count=$((current_count + 1))
    
    echo "[$current_count/$total_images] 处理镜像: $source"
    echo "  目标: $target"
    
    # 拉取源镜像
    echo "  → 拉取源镜像..."
    if docker pull "$source"; then
        echo "    ✓ 拉取成功"
    else
        echo "    ✗ 拉取失败"
        failed_images+=("$source (拉取失败)")
        continue
    fi
    
    # 重新标签
    echo "  → 重新标签..."
    if docker tag "$source" "$target"; then
        echo "    ✓ 标签成功"
    else
        echo "    ✗ 标签失败"
        failed_images+=("$source (标签失败)")
        continue
    fi
    
    # 推送到私有仓库
    echo "  → 推送到私有仓库..."
    if docker push "$target"; then
        echo "    ✓ 推送成功"
    else
        echo "    ✗ 推送失败"
        failed_images+=("$source (推送失败)")
        continue
    fi
    
    # 验证推送结果
    echo "  → 验证镜像..."
    if docker pull "$target" --quiet >/dev/null 2>&1; then
        echo "    ✓ 验证成功"
    else
        echo "    ⚠ 验证失败（镜像可能仍在同步中）"
    fi
    
    echo "  ✓ 完成: $source → $target"
    echo
done

echo "============================================"
echo "迁移结果汇总:"
echo "总计镜像: $total_images"
echo "成功迁移: $((total_images - ${#failed_images[@]}))"
echo "失败镜像: ${#failed_images[@]}"

if [[ ${#failed_images[@]} -gt 0 ]]; then
    echo
    echo "失败镜像列表:"
    for failed in "${failed_images[@]}"; do
        echo "  ✗ $failed"
    done
    echo
    echo "请检查网络连接和仓库权限，然后重新运行脚本。"
    exit 1
else
    echo
    echo "🎉 所有基础镜像迁移成功！"
    echo
    echo "下一步："
    echo "1. 构建和推送源码镜像:"
    echo "   ./build.sh build-push $REGISTRY_BASE v0.3.5"
    echo
    echo "2. 生成生产配置:"
    echo "   ./build.sh prod-generate $REGISTRY_BASE v0.3.5"
    echo
    echo "3. 启动生产环境:"
    echo "   ./build.sh prod-up $REGISTRY_BASE v0.3.5"
fi

echo
echo "结束时间: $(date)"
echo "=== 基础镜像迁移完成 ==="
