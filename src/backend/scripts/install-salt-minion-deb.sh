#!/bin/bash
# =============================================================================
# SaltStack Minion 安装脚本 (Debian/Ubuntu)
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
ARCH=$(dpkg --print-architecture 2>/dev/null || echo "amd64")
echo "Detected architecture: ${ARCH}"

# 设置AppHub包路径
APPHUB_BASE="${APPHUB_URL}/pkgs/saltstack-deb"

# 下载所需的SaltStack包
cd /tmp
echo "Downloading SaltStack packages from AppHub..."

# 下载依赖包
echo "Downloading salt-common_${SALT_VERSION}_${ARCH}.deb..."
curl -fsSL "${APPHUB_BASE}/salt-common_${SALT_VERSION}_${ARCH}.deb" -o salt-common.deb || {
    echo "Failed to download salt-common from AppHub"
    exit 1
}

# 下载salt-minion
echo "Downloading salt-minion_${SALT_VERSION}_${ARCH}.deb..."
curl -fsSL "${APPHUB_BASE}/salt-minion_${SALT_VERSION}_${ARCH}.deb" -o salt-minion.deb || {
    echo "Failed to download salt-minion from AppHub"
    exit 1
}

echo "Installing SaltStack packages..."
# 更新包索引
sudo apt-get update -y || true

# 安装Python依赖
echo "Installing Python dependencies..."
sudo apt-get install -y python3 python3-pip python3-setuptools || true

# 安装salt-common（依赖包）
echo "Installing salt-common..."
sudo dpkg -i salt-common.deb || sudo apt-get install -f -y

# 安装salt-minion
echo "Installing salt-minion..."
sudo dpkg -i salt-minion.deb || sudo apt-get install -f -y

# 清理临时文件
rm -f salt-common.deb salt-minion.deb

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
