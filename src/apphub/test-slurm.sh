#!/bin/bash
# =============================================================================
# AppHub SLURM 快速测试脚本
# 用于验证 SLURM 包集成是否正常工作
# =============================================================================

set -e

# 配置
APPHUB_PORT=8081
APPHUB_IMAGE="ai-infra-apphub:latest"
CONTAINER_NAME="apphub-slurm-test"

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
echo "  AppHub SLURM Integration Test"
echo "======================================================================"
echo ""

# Step 1: 检查镜像是否存在
log_step "Step 1/8: Checking AppHub image..."
if ! docker images | grep -q "${APPHUB_IMAGE%:*}"; then
    log_warn "AppHub image not found, building..."
    cd "$(dirname "$0")/../.."
    ./build.sh build apphub
    if [ $? -ne 0 ]; then
        log_error "Failed to build AppHub image"
        exit 1
    fi
fi
log_info "✓ AppHub image exists: ${APPHUB_IMAGE}"

# Step 2: 启动 AppHub 容器
log_step "Step 2/8: Starting AppHub container..."
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

# 查看容器日志
log_info "Container startup logs:"
docker logs ${CONTAINER_NAME} 2>&1 | tail -30

# Step 3: 检查 SLURM 包目录
log_step "Step 3/8: Checking SLURM package directories..."

# 检查 DEB 包
response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:${APPHUB_PORT}/pkgs/slurm-deb/)
log_info "SLURM DEB directory: HTTP $response"
if [ "$response" = "200" ]; then
    log_info "Available SLURM DEB packages:"
    curl -s http://localhost:${APPHUB_PORT}/pkgs/slurm-deb/ | grep -oP 'slurm[^"]+\.deb' | sort -u | while read file; do
        echo "  - $file"
    done
    
    # 检查 Packages 索引文件
    if curl -s -f http://localhost:${APPHUB_PORT}/pkgs/slurm-deb/Packages >/dev/null 2>&1; then
        log_info "✓ Packages index file exists"
        pkg_count=$(curl -s http://localhost:${APPHUB_PORT}/pkgs/slurm-deb/Packages | grep -c "^Package:" || echo 0)
        log_info "  Indexed packages: $pkg_count"
    else
        log_warn "⚠️  Packages index file missing"
    fi
    
    if curl -s -f http://localhost:${APPHUB_PORT}/pkgs/slurm-deb/Packages.gz >/dev/null 2>&1; then
        log_info "✓ Packages.gz exists"
    else
        log_warn "⚠️  Packages.gz missing"
    fi
fi

# 检查 RPM 包
response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:${APPHUB_PORT}/pkgs/slurm-rpm/)
log_info "SLURM RPM directory: HTTP $response"
if [ "$response" = "200" ]; then
    log_info "Available SLURM RPM packages:"
    curl -s http://localhost:${APPHUB_PORT}/pkgs/slurm-rpm/ | grep -oP 'slurm[^"]+\.rpm' | sort -u | while read file; do
        echo "  - $file"
    done
    
    # 检查 repodata 目录
    if curl -s -f http://localhost:${APPHUB_PORT}/pkgs/slurm-rpm/repodata/ >/dev/null 2>&1; then
        log_info "✓ RPM repodata directory exists"
        curl -s http://localhost:${APPHUB_PORT}/pkgs/slurm-rpm/repodata/ | grep -oP 'href="\K[^"]+\.xml[^"]*' | while read file; do
            log_info "  - repodata/$file"
        done
    else
        log_warn "⚠️  RPM repodata directory missing"
    fi
fi

# 检查二进制文件
response=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:${APPHUB_PORT}/pkgs/slurm-binaries/)
log_info "SLURM binaries directory: HTTP $response"
if [ "$response" = "200" ]; then
    log_info "Available SLURM binary architectures:"
    curl -s http://localhost:${APPHUB_PORT}/pkgs/slurm-binaries/ | grep -oP 'href="\K[^/"]+(?=/)' | while read arch; do
        echo "  - $arch"
        # 检查每个架构的二进制文件
        bin_count=$(curl -s http://localhost:${APPHUB_PORT}/pkgs/slurm-binaries/${arch}/bin/ 2>/dev/null | grep -c 'href="s' || echo 0)
        lib_count=$(curl -s http://localhost:${APPHUB_PORT}/pkgs/slurm-binaries/${arch}/lib/ 2>/dev/null | grep -c 'href="' || echo 0)
        log_info "    Binaries: $bin_count, Libraries: $lib_count"
    done
fi

# Step 4: 测试 DEB 包下载
log_step "Step 4/8: Testing DEB package download..."
TMPDIR=$(mktemp -d)
cd ${TMPDIR}

# 获取第一个 DEB 包名称
first_deb=$(curl -s http://localhost:${APPHUB_PORT}/pkgs/slurm-deb/ | grep -oP 'slurm[^"]+\.deb' | head -1)
if [ -n "$first_deb" ]; then
    log_info "Downloading: $first_deb"
    if wget -q http://localhost:${APPHUB_PORT}/pkgs/slurm-deb/${first_deb}; then
        size=$(stat -f%z ${first_deb} 2>/dev/null || stat -c%s ${first_deb})
        size_kb=$((size / 1024))
        log_info "✓ Downloaded ${first_deb} (${size_kb} KB)"
        
        # 检查 DEB 包信息
        if command -v dpkg-deb >/dev/null 2>&1; then
            log_info "Package information:"
            dpkg-deb -I ${first_deb} | head -20
        fi
    else
        log_error "Failed to download ${first_deb}"
    fi
else
    log_warn "No DEB packages found to test"
fi

# Step 5: 测试 RPM 包下载
log_step "Step 5/8: Testing RPM package download..."
cd ${TMPDIR}

first_rpm=$(curl -s http://localhost:${APPHUB_PORT}/pkgs/slurm-rpm/ | grep -oP 'slurm[^"]+\.rpm' | head -1)
if [ -n "$first_rpm" ]; then
    log_info "Downloading: $first_rpm"
    if wget -q http://localhost:${APPHUB_PORT}/pkgs/slurm-rpm/${first_rpm}; then
        size=$(stat -f%z ${first_rpm} 2>/dev/null || stat -c%s ${first_rpm})
        size_kb=$((size / 1024))
        log_info "✓ Downloaded ${first_rpm} (${size_kb} KB)"
        
        # 检查 RPM 包信息
        if command -v rpm >/dev/null 2>&1; then
            log_info "Package information:"
            rpm -qip ${first_rpm} 2>/dev/null | head -20 || log_warn "rpm command not available or package info failed"
        fi
    else
        log_error "Failed to download ${first_rpm}"
    fi
else
    log_warn "No RPM packages found to test"
fi

# Step 6: 测试二进制文件下载
log_step "Step 6/8: Testing binary download..."
cd ${TMPDIR}

# 检测系统架构
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)
        ARCH_DIR="x86_64"
        ;;
    aarch64|arm64)
        ARCH_DIR="arm64"
        ;;
    *)
        log_warn "Unsupported architecture: ${ARCH}, using x86_64"
        ARCH_DIR="x86_64"
        ;;
esac

log_info "Testing architecture: ${ARCH_DIR}"

# 测试下载 sinfo
if wget -q http://localhost:${APPHUB_PORT}/pkgs/slurm-binaries/${ARCH_DIR}/bin/sinfo; then
    size=$(stat -f%z sinfo 2>/dev/null || stat -c%s sinfo)
    size_kb=$((size / 1024))
    log_info "✓ Downloaded sinfo (${size_kb} KB)"
    
    # 检查文件类型
    if command -v file >/dev/null 2>&1; then
        file_type=$(file sinfo)
        log_info "File type: $file_type"
    fi
    
    # 检查是否可执行
    chmod +x sinfo
    if [ -x sinfo ]; then
        log_info "✓ Binary is executable"
    fi
else
    log_error "Failed to download sinfo binary"
fi

# Step 7: 测试在 Ubuntu 容器中安装 SLURM DEB 包
log_step "Step 7/8: Testing DEB installation in Ubuntu container..."

if [ -n "$first_deb" ] && [ -f "${TMPDIR}/${first_deb}" ]; then
    log_info "Starting Ubuntu test container..."
    
    # 创建临时测试容器
    test_container="slurm-deb-test-$$"
    docker run -d --name ${test_container} \
        -v ${TMPDIR}:/packages \
        --network container:${CONTAINER_NAME} \
        ubuntu:22.04 sleep 300 2>/dev/null || true
    
    if docker ps | grep -q ${test_container}; then
        log_info "Installing package in test container..."
        
        # 配置 apt 源指向 AppHub
        docker exec ${test_container} sh -c "
            echo 'deb [trusted=yes] http://localhost/pkgs/slurm-deb/ ./' > /etc/apt/sources.list.d/apphub-slurm.list
            apt-get update -o Dir::Etc::sourcelist=/etc/apt/sources.list.d/apphub-slurm.list 2>&1
        " | tail -10
        
        if [ $? -eq 0 ]; then
            log_info "✓ AppHub as APT source configured successfully"
            
            # 尝试安装
            docker exec ${test_container} sh -c "
                apt-cache search slurm 2>&1 | head -10
            "
        else
            log_error "Failed to add AppHub as APT source"
        fi
        
        # 清理测试容器
        docker stop ${test_container} >/dev/null 2>&1
        docker rm ${test_container} >/dev/null 2>&1
    else
        log_warn "Could not start test container"
    fi
else
    log_warn "Skipping DEB installation test (no package available)"
fi

# Step 8: 测试安装脚本
log_step "Step 8/8: Testing install-slurm.sh script..."

script_url="http://localhost:${APPHUB_PORT}/packages/install-slurm.sh"
if curl -s -f -o install-slurm.sh ${script_url}; then
    log_info "✓ Downloaded install-slurm.sh"
    
    # 检查脚本语法
    if bash -n install-slurm.sh 2>/dev/null; then
        log_info "✓ Script syntax valid"
    else
        log_error "Script syntax error"
        bash -n install-slurm.sh
    fi
    
    # 显示脚本内容摘要
    log_info "Script summary:"
    echo "  - APPHUB_URL: $(grep 'APPHUB_URL=' install-slurm.sh | head -1)"
    echo "  - Binaries to install: $(grep -oP 'for cmd in \K[^;]+' install-slurm.sh | head -1)"
else
    log_warn "install-slurm.sh not available at ${script_url}"
fi

# 清理临时文件
cd /
rm -rf ${TMPDIR}

echo ""
echo "======================================================================"
log_info "Test Summary"
echo "======================================================================"
echo ""
log_info "AppHub is running on http://localhost:${APPHUB_PORT}"
log_info "SLURM packages:"
echo "  - DEB: http://localhost:${APPHUB_PORT}/pkgs/slurm-deb/"
echo "  - RPM: http://localhost:${APPHUB_PORT}/pkgs/slurm-rpm/"
echo "  - Binaries: http://localhost:${APPHUB_PORT}/pkgs/slurm-binaries/"
echo "  - Install script: http://localhost:${APPHUB_PORT}/packages/install-slurm.sh"
echo ""
log_info "To stop the test container:"
echo "  docker stop ${CONTAINER_NAME}"
echo ""

# 不自动清理，让用户手动停止
trap - EXIT

exit 0
