#!/bin/bash
# =============================================================================
# SaltStack Minion 安装脚本 (CentOS/RHEL/Rocky Linux)
# 从 AppHub 下载并安装 SaltStack Minion
# =============================================================================

set -e

# 参数检查
if [ $# -lt 2 ]; then
    echo "Usage: $0 <APPHUB_URL> <SALT_VERSION>"
    echo "Example: $0 http://192.168.0.200:53434 3007.8"
    exit 1
fi

APPHUB_URL="$1"
SALT_VERSION="$2"

echo "=========================================="
echo "Installing SaltStack Minion from AppHub"
echo "AppHub URL: ${APPHUB_URL}"
echo "SaltStack Version: ${SALT_VERSION}"
echo "=========================================="

# 检测架构
ARCH=$(uname -m)
echo "Detected architecture: ${ARCH}"

# 设置AppHub包路径
APPHUB_BASE="${APPHUB_URL}/pkgs/saltstack-rpm"
VERSION="${SALT_VERSION}-0"

# 下载所需的SaltStack包
cd /tmp
echo "Downloading SaltStack packages from AppHub..."

# 下载salt-minion RPM
echo "Downloading salt-minion-${VERSION}.${ARCH}.rpm..."
curl -fsSL "${APPHUB_BASE}/salt-minion-${VERSION}.${ARCH}.rpm" -o salt-minion.rpm || {
    echo "Failed to download salt-minion from AppHub"
    exit 1
}

echo "Installing SaltStack packages..."
# 安装Python依赖
echo "Installing Python dependencies..."
sudo yum install -y python3 python3-pip 2>/dev/null || sudo dnf install -y python3 python3-pip || true

# 安装salt-minion
echo "Installing salt-minion..."
sudo yum install -y ./salt-minion.rpm 2>/dev/null || sudo dnf install -y ./salt-minion.rpm

# 清理临时文件
rm -f salt-minion.rpm

# 验证安装
echo "Verifying installation..."
salt-minion --version || {
    echo "Salt Minion installation verification failed"
    exit 1
}

echo "=========================================="
echo "SaltStack Minion installed successfully from AppHub"
echo "Version: $(salt-minion --version)"
echo "=========================================="
