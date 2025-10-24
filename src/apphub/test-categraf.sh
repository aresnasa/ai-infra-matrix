#!/bin/bash
# =============================================================================
# AppHub Categraf 快速测试脚本
# 用于验证 Categraf 集成是否正常工作
# =============================================================================

set -e

# 配置
APPHUB_PORT=8081
APPHUB_IMAGE="ai-infra-apphub:latest"
CONTAINER_NAME="apphub-categraf-test"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# 清理函数
cleanup() {
    if [ "$1" != "skip" ]; then
        log_info "Cleaning up..."
        docker stop ${CONTAINER_NAME} 2>/dev/null || true
        docker rm ${CONTAINER_NAME} 2>/dev/null || true
    fi
}

# 捕获错误并清理
trap cleanup EXIT

# 检查 Docker
if ! command -v docker &> /dev/null; then
    log_error "Docker is not installed"
    exit 1
fi

echo "======================================================================"
echo "  AppHub Categraf Integration Test"
echo "======================================================================"
echo ""

# Step 1: 检查镜像是否存在
log_step "Step 1/6: Checking AppHub image..."
if ! docker images | grep -q "${APPHUB_IMAGE%:*}"; then
    log_warn "AppHub image not found, building..."
    cd "$(dirname "$0")/.."
    docker build -t ${APPHUB_IMAGE} -f src/apphub/Dockerfile src/apphub
    if [ $? -ne 0 ]; then
        log_error "Failed to build AppHub image"
        exit 1
    fi
fi
log_info "✓ AppHub image exists: ${APPHUB_IMAGE}"

# Step 2: 启动 AppHub 容器
log_step "Step 2/6: Starting AppHub container..."
# 清理旧容器
docker stop ${CONTAINER_NAME} 2>/dev/null || true
docker rm ${CONTAINER_NAME} 2>/dev/null || true

docker run -d \
    --name ${CONTAINER_NAME} \
    -p ${APPHUB_PORT}:80 \
    ${APPHUB_IMAGE}

if [ $? -ne 0 ]; then
    log_error "Failed to start container"
    exit 1
fi

# 等待容器启动
log_info "Waiting for container to be ready..."
sleep 5

# 检查容器状态
if ! docker ps | grep -q ${CONTAINER_NAME}; then
    log_error "Container is not running"
    docker logs ${CONTAINER_NAME}
    exit 1
fi
log_info "✓ AppHub container started (port ${APPHUB_PORT})"

# Step 3: 检查 Categraf 目录
log_step "Step 3/6: Checking Categraf directory..."
response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:${APPHUB_PORT}/pkgs/categraf/)
if [ "$response" != "200" ]; then
    log_error "Categraf directory not accessible (HTTP $response)"
    exit 1
fi
log_info "✓ Categraf directory accessible"

# 列出文件
log_info "Available Categraf packages:"
curl -s http://localhost:${APPHUB_PORT}/pkgs/categraf/ | grep -oP 'categraf-[^"]+\.tar\.gz' | sort -u | while read file; do
    echo "  - $file"
done

# Step 4: 测试 AMD64 包下载
log_step "Step 4/6: Testing AMD64 package download..."
TMPDIR=$(mktemp -d)
cd ${TMPDIR}

wget -q http://localhost:${APPHUB_PORT}/pkgs/categraf/categraf-latest-linux-amd64.tar.gz
if [ $? -ne 0 ]; then
    log_error "Failed to download AMD64 package"
    exit 1
fi

# 检查文件大小
size=$(stat -f%z categraf-latest-linux-amd64.tar.gz 2>/dev/null || stat -c%s categraf-latest-linux-amd64.tar.gz)
size_mb=$((size / 1024 / 1024))
log_info "✓ AMD64 package downloaded (${size_mb} MB)"

# 解压并验证
tar xzf categraf-latest-linux-amd64.tar.gz
if [ $? -ne 0 ]; then
    log_error "Failed to extract AMD64 package"
    exit 1
fi

cd categraf-*-linux-amd64
log_info "✓ AMD64 package extracted"

# 检查必需文件
missing_files=""
for file in bin/categraf install.sh uninstall.sh README.md categraf.service; do
    if [ ! -f "$file" ]; then
        missing_files="${missing_files} $file"
    fi
done

if [ -n "$missing_files" ]; then
    log_error "Missing files:${missing_files}"
    exit 1
fi
log_info "✓ All required files present"

# 检查二进制架构
if command -v file &> /dev/null; then
    arch_info=$(file bin/categraf)
    if echo "$arch_info" | grep -q "x86-64\|x86_64"; then
        log_info "✓ Binary architecture: x86_64"
    else
        log_warn "Binary architecture: $arch_info"
    fi
fi

# 检查脚本语法
bash -n install.sh
bash -n uninstall.sh
log_info "✓ Scripts syntax valid"

# Step 5: 测试 ARM64 包下载
log_step "Step 5/6: Testing ARM64 package download..."
cd ${TMPDIR}
rm -rf categraf-*

wget -q http://localhost:${APPHUB_PORT}/pkgs/categraf/categraf-latest-linux-arm64.tar.gz
if [ $? -ne 0 ]; then
    log_error "Failed to download ARM64 package"
    exit 1
fi

size=$(stat -f%z categraf-latest-linux-arm64.tar.gz 2>/dev/null || stat -c%s categraf-latest-linux-arm64.tar.gz)
size_mb=$((size / 1024 / 1024))
log_info "✓ ARM64 package downloaded (${size_mb} MB)"

tar xzf categraf-latest-linux-arm64.tar.gz
if [ $? -ne 0 ]; then
    log_error "Failed to extract ARM64 package"
    exit 1
fi
log_info "✓ ARM64 package extracted"

# Step 6: 验证版本信息
log_step "Step 6/6: Checking version information..."
cd ${TMPDIR}/categraf-*-linux-arm64

if [ -f README.md ]; then
    version=$(grep -oP 'Categraf \Kv[\d.]+' README.md | head -1)
    if [ -n "$version" ]; then
        log_info "✓ Categraf version: $version"
    fi
fi

# 清理临时文件
cd /
rm -rf ${TMPDIR}

echo ""
echo "======================================================================"
log_info "✓ All tests passed successfully!"
echo "======================================================================"
echo ""
log_info "AppHub is running on http://localhost:${APPHUB_PORT}"
log_info "Categraf packages: http://localhost:${APPHUB_PORT}/pkgs/categraf/"
echo ""
log_info "To download packages:"
echo "  AMD64: wget http://localhost:${APPHUB_PORT}/pkgs/categraf/categraf-latest-linux-amd64.tar.gz"
echo "  ARM64: wget http://localhost:${APPHUB_PORT}/pkgs/categraf/categraf-latest-linux-arm64.tar.gz"
echo ""
log_info "To stop the test container:"
echo "  docker stop ${CONTAINER_NAME}"
echo ""

# 不自动清理，让用户手动停止
trap - EXIT
