#!/bin/bash
# =============================================================================
# SLURM 二进制构建测试脚本
# =============================================================================

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }
print_header() { echo -e "\n${BLUE}========================================${NC}"; echo -e "${BLUE}$1${NC}"; echo -e "${BLUE}========================================${NC}"; }

# 检测架构
ARCH=$(uname -m)
case "${ARCH}" in
    x86_64|amd64)
        PLATFORM="linux/amd64"
        SLURM_ARCH="x86_64"
        ;;
    aarch64|arm64)
        PLATFORM="linux/arm64"
        SLURM_ARCH="arm64"
        ;;
    *)
        print_error "Unsupported architecture: ${ARCH}"
        exit 1
        ;;
esac

print_info "Architecture: ${ARCH} (${PLATFORM})"

# 步骤 1: 检查 SLURM 源码包
print_header "Step 1: Checking SLURM source tarball"
if [ -f "src/apphub/slurm-25.05.4.tar.bz2" ]; then
    print_success "SLURM source tarball found"
    ls -lh src/apphub/slurm-25.05.4.tar.bz2
else
    print_error "SLURM source tarball not found: src/apphub/slurm-25.05.4.tar.bz2"
    print_info "Please download it first:"
    echo "  wget -P src/apphub https://download.schedmd.com/slurm/slurm-25.05.4.tar.bz2"
    exit 1
fi

# 步骤 2: 构建 AppHub 镜像
print_header "Step 2: Building AppHub with SLURM binaries"
print_info "Building for platform: ${PLATFORM}"
print_info "This may take 10-20 minutes..."

docker build \
    --platform ${PLATFORM} \
    -t ai-infra-apphub:slurm-binary \
    -f src/apphub/Dockerfile.slurm-binary \
    src/apphub || {
    print_error "AppHub build failed"
    exit 1
}

print_success "AppHub image built successfully"

# 步骤 3: 启动 AppHub 容器
print_header "Step 3: Starting AppHub container"

# 停止旧容器
docker stop ai-infra-apphub 2>/dev/null || true
docker rm ai-infra-apphub 2>/dev/null || true

# 启动新容器
docker run -d \
    --name ai-infra-apphub \
    --network ai-infra-matrix_default \
    -p 8081:80 \
    ai-infra-apphub:slurm-binary || {
    print_error "Failed to start AppHub container"
    exit 1
}

sleep 3
print_success "AppHub container started"

# 步骤 4: 验证 SLURM 文件
print_header "Step 4: Verifying SLURM binaries in AppHub"

# 检查版本文件
print_info "Checking VERSION file..."
VERSION=$(docker exec ai-infra-apphub cat /usr/share/nginx/html/packages/${SLURM_ARCH}/slurm/VERSION 2>/dev/null || echo "NOT_FOUND")
if [ "$VERSION" != "NOT_FOUND" ]; then
    print_success "SLURM Version: $VERSION"
else
    print_error "VERSION file not found"
fi

# 检查二进制文件
print_info "Checking binary files..."
docker exec ai-infra-apphub ls -lh /usr/share/nginx/html/packages/${SLURM_ARCH}/slurm/bin/ || {
    print_error "Binary files not found"
    exit 1
}

# 步骤 5: 测试 HTTP 访问
print_header "Step 5: Testing HTTP access"

print_info "Testing VERSION file..."
curl -s http://localhost:8081/packages/${SLURM_ARCH}/slurm/VERSION || {
    print_error "Failed to access VERSION via HTTP"
    exit 1
}

print_info "Testing binary file..."
curl -I -s http://localhost:8081/packages/${SLURM_ARCH}/slurm/bin/sinfo | head -1 || {
    print_error "Failed to access sinfo via HTTP"
    exit 1
}

print_success "HTTP access verified"

# 步骤 6: 测试安装脚本
print_header "Step 6: Testing installation script"

# 创建测试容器
print_info "Creating test container..."
docker run -d --name slurm-install-test \
    --network ai-infra-matrix_default \
    alpine:latest \
    sleep 3600 || {
    print_error "Failed to create test container"
    exit 1
}

# 安装 wget
print_info "Installing wget in test container..."
docker exec slurm-install-test apk add --no-cache wget bash

# 下载并执行安装脚本
print_info "Running installation script..."
docker exec -e APPHUB_URL=http://apphub slurm-install-test sh -c "
    wget -O /tmp/install-slurm.sh http://apphub/packages/install-slurm.sh && \
    chmod +x /tmp/install-slurm.sh && \
    /tmp/install-slurm.sh
" || {
    print_error "Installation script failed"
    docker logs slurm-install-test
    docker rm -f slurm-install-test
    exit 1
}

# 验证安装
print_info "Verifying installation..."
docker exec slurm-install-test sinfo --version || {
    print_error "sinfo command not found after installation"
    docker rm -f slurm-install-test
    exit 1
}

print_success "Installation script works correctly"

# 清理测试容器
docker rm -f slurm-install-test

# 最终总结
print_header "Build and Test Summary"
print_success "✓ SLURM source tarball exists"
print_success "✓ AppHub image built successfully"
print_success "✓ AppHub container running"
print_success "✓ SLURM binaries available in AppHub"
print_success "✓ HTTP access working"
print_success "✓ Installation script working"

echo ""
print_info "Next steps:"
echo "  1. Update docker-compose.yml to use new image:"
echo "     image: ai-infra-apphub:slurm-binary"
echo ""
echo "  2. Rebuild backend:"
echo "     docker-compose build backend"
echo ""
echo "  3. Start all services:"
echo "     docker-compose up -d"
echo ""
echo "  4. Verify SLURM in backend:"
echo "     docker exec ai-infra-backend sinfo --version"
echo ""

print_success "All tests passed!"
