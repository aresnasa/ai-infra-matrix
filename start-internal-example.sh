#!/bin/bash

# AI-Infra-Matrix 内部仓库启动示例
# 请根据您的实际镜像仓库地址修改以下变量

# =============================================================================
# 配置区域 - 请修改为您的实际配置
# =============================================================================

# 内部镜像仓库地址（请修改为您的实际地址）
INTERNAL_REGISTRY="registry.your-company.com/ai-infra/"

# 镜像标签（请修改为您推送的实际标签）
IMAGE_TAG="v0.3.5"

# =============================================================================
# 启动脚本
# =============================================================================

echo "🚀 AI-Infra-Matrix 内部仓库启动示例"
echo "=================================="
echo "镜像仓库: $INTERNAL_REGISTRY"
echo "镜像标签: $IMAGE_TAG"
echo ""

# 检查配置
if [[ "$INTERNAL_REGISTRY" == "registry.your-company.com/ai-infra/" ]]; then
    echo "⚠️  请先修改脚本中的 INTERNAL_REGISTRY 变量为您的实际仓库地址"
    echo "   编辑文件: $(basename "$0")"
    echo "   修改第11行: INTERNAL_REGISTRY=\"your-actual-registry/\""
    exit 1
fi

# 启动服务
echo "启动服务..."
./build.sh start-internal "$INTERNAL_REGISTRY" "$IMAGE_TAG"
