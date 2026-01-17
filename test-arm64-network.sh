#!/bin/bash

# ARM64 网络构建测试脚本
# 用于验证 host 网络配置是否正确生效

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  ARM64 构建网络配置验证${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo

# ============================================================================
# 检查前置条件
# ============================================================================
echo -e "${YELLOW}【第一步】检查 Docker 和 BuildX 配置${NC}"
echo

# 检查 Docker daemon
echo -n "✓ Docker daemon: "
if docker ps >/dev/null 2>&1; then
    echo -e "${GREEN}运行中${NC}"
else
    echo -e "${RED}未运行${NC}"
    exit 1
fi

# 检查 buildx
echo -n "✓ Docker buildx: "
if docker buildx version >/dev/null 2>&1; then
    echo -e "${GREEN}可用${NC}"
    docker buildx version | head -1 | sed 's/^/  /'
else
    echo -e "${RED}不可用${NC}"
    exit 1
fi

# 检查 QEMU
echo -n "✓ QEMU (arm64 support): "
if docker run --rm --platform linux/arm64 alpine:3.18 uname -m >/dev/null 2>&1; then
    echo -e "${GREEN}已安装${NC}"
else
    echo -e "${YELLOW}未安装，尝试安装...${NC}"
    if docker run --rm --privileged tonistiigi/binfmt --install arm64 >/dev/null 2>&1; then
        echo -e "${GREEN}安装成功${NC}"
    else
        echo -e "${YELLOW}QEMU 安装可选，某些平台自动支持${NC}"
    fi
fi
echo

# ============================================================================
# 检查 multiarch-builder 配置
# ============================================================================
echo -e "${YELLOW}【第二步】检查 multiarch-builder 配置${NC}"
echo

BUILDER_NAME="multiarch-builder"

# 检查 builder 是否存在
if docker buildx ls | grep -q "^$BUILDER_NAME"; then
    echo "✓ Builder exists: $BUILDER_NAME"
    
    # 显示 builder 详情
    echo
    echo "Builder Details:"
    docker buildx inspect "$BUILDER_NAME" | while read line; do
        [[ -n "$line" ]] && echo "  $line"
    done
else
    echo -e "${YELLOW}⚠ Builder '$BUILDER_NAME' not found${NC}"
    echo "  Creating new builder with host network support..."
    echo
    
    if docker buildx create --name "$BUILDER_NAME" \
        --driver docker-container \
        --driver-opt network=host \
        --buildkitd-flags '--allow-insecure-entitlement network.host' \
        --bootstrap 2>&1; then
        echo -e "${GREEN}✓ Builder created successfully${NC}"
    else
        echo -e "${RED}✗ Failed to create builder${NC}"
        exit 1
    fi
fi
echo

# ============================================================================
# 网络连接测试
# ============================================================================
echo -e "${YELLOW}【第三步】测试网络连接${NC}"
echo

# 测试 Docker Hub 连接
echo -n "✓ Docker Hub connection: "
if curl -s -I https://docker.io >/dev/null 2>&1; then
    echo -e "${GREEN}正常${NC}"
else
    echo -e "${YELLOW}超时或无网络${NC}"
fi

# 测试 OAuth token 获取（最关键）
echo -n "✓ Docker Hub OAuth (critical): "
if curl -s -X POST "https://auth.docker.io/v2/token?service=registry.docker.io&scope=repository:library/ubuntu:pull" \
    -H "User-Agent: Docker-Client" >/dev/null 2>&1; then
    echo -e "${GREEN}正常${NC}"
else
    echo -e "${YELLOW}超时或网络问题${NC}"
fi

# 测试 DNS 解析
echo -n "✓ DNS resolution (docker.io): "
if nslookup docker.io >/dev/null 2>&1; then
    echo -e "${GREEN}正常${NC}"
else
    echo -e "${YELLOW}失败${NC}"
fi
echo

# ============================================================================
# 简单的镜像拉取测试
# ============================================================================
echo -e "${YELLOW}【第四步】测试镜像拉取（amd64）${NC}"
echo

# 尝试拉取一个小镜像用于测试
TEST_IMAGE="alpine:3.18"
echo "Pulling $TEST_IMAGE for amd64..."

if docker pull --platform linux/amd64 "$TEST_IMAGE" >/dev/null 2>&1; then
    echo -e "${GREEN}✓ Successfully pulled $TEST_IMAGE for amd64${NC}"
else
    echo -e "${YELLOW}⚠ Failed to pull (may indicate network issues)${NC}"
fi
echo

# ============================================================================
# 显示构建命令建议
# ============================================================================
echo -e "${YELLOW}【第五步】后续测试步骤${NC}"
echo

cat << 'EOF'
1. 测试单个服务的 arm64 构建：
   ./build.sh build-component redis linux/arm64
   
   或选择更小的服务（如果 redis 较大）：
   ./build.sh build-component shared linux/arm64

2. 观察构建过程中的日志：
   - 应该看到 "Creating multiarch-builder with host network support"
   - 应该看到 "Network configuration: CRITICAL for arm64"
   - 应该看到 "--network host" 在构建命令中

3. 如果仍然超时，检查详细日志：
   tail -100 .build-failures.log | grep -iE "network|timeout|error"

4. 查看 builder 的实时日志：
   docker buildx du
   docker buildx prune

5. 如果需要重新创建 builder：
   docker buildx rm multiarch-builder
   # 脚本会在下次构建时自动创建新的 builder
EOF

echo
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}验证完成！现在可以开始 ARM64 构建测试${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo

# 显示 builder 命令建议
echo -e "${YELLOW}快速命令：${NC}"
echo
echo "# 查看 builder 状态"
echo "  docker buildx ls | grep multiarch"
echo
echo "# 查看 builder 详细配置"
echo "  docker buildx inspect multiarch-builder"
echo
echo "# 开始 arm64 构建"
echo "  ./build.sh build-platform arm64 --force"
echo
