#!/bin/bash
# Singularity 安装脚本
# 从 AppHub 下载并安装 Singularity

set -e

APPHUB_URL="${APPHUB_URL:-http://apphub:8081}"
VERSION="${SINGULARITY_VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/singularity}"

echo "=========================================="
echo "安装 Singularity ${VERSION}"
echo "=========================================="
echo "AppHub URL: ${APPHUB_URL}"
echo "安装目录: ${INSTALL_DIR}"
echo ""

# 检测架构
ARCH=$(uname -m)
echo "检测到架构: ${ARCH}"

# 下载 Singularity
echo ">>> 下载 Singularity..."
PACKAGE_URL="${APPHUB_URL}/pkgs/singularity/singularity-${VERSION}-linux-${ARCH}.tar.gz"

if ! curl -fsSL "${PACKAGE_URL}" -o /tmp/singularity.tar.gz; then
    echo "ERROR: 下载失败: ${PACKAGE_URL}"
    exit 1
fi

# 解压并安装
echo ">>> 安装 Singularity..."
mkdir -p "${INSTALL_DIR}"
tar -xzf /tmp/singularity.tar.gz -C /
rm -f /tmp/singularity.tar.gz

# 创建符号链接
echo ">>> 创建符号链接..."
ln -sf "${INSTALL_DIR}/bin/singularity" /usr/local/bin/singularity

# 验证安装
if command -v singularity &> /dev/null; then
    INSTALLED_VERSION=$(singularity --version 2>&1 | grep -oP '\d+\.\d+\.\d+' || echo "unknown")
    echo ""
    echo "=========================================="
    echo "✅ Singularity 安装成功"
    echo "=========================================="
    echo "版本: ${INSTALLED_VERSION}"
    echo "路径: $(which singularity)"
    echo ""
    echo "使用方法:"
    echo "  singularity --version"
    echo "  singularity pull docker://alpine"
    echo "  singularity run alpine_latest.sif"
else
    echo "ERROR: Singularity 安装失败"
    exit 1
fi
