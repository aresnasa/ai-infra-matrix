#!/bin/bash
# ====================================================================
# Salt Minion 安装脚本 - 仅使用AppHub内网源
# ====================================================================
# 功能：从AppHub内网仓库安装salt-minion，不依赖任何外网源
# 环境变量:
#   APPHUB_URL - AppHub服务地址 (默认: http://apphub:80)
# ====================================================================

set -e

APPHUB_URL="${APPHUB_URL:-http://apphub:80}"

echo "=== 开始安装 Salt Minion (仅AppHub) ==="
echo "AppHub URL: ${APPHUB_URL}"

# 检测操作系统类型
if command -v apt-get >/dev/null 2>&1; then
    OS_TYPE="debian"
    echo "检测到系统: Debian/Ubuntu"
elif command -v dnf >/dev/null 2>&1; then
    OS_TYPE="rhel-dnf"
    echo "检测到系统: Rocky/Fedora (DNF)"
elif command -v yum >/dev/null 2>&1; then
    OS_TYPE="rhel-yum"
    echo "检测到系统: RHEL/CentOS (YUM)"
elif command -v zypper >/dev/null 2>&1; then
    OS_TYPE="suse"
    echo "检测到系统: SUSE"
else
    echo "❌ 不支持的操作系统"
    exit 1
fi

# ==================== Debian/Ubuntu 系统 ====================
if [ "$OS_TYPE" = "debian" ]; then
    export DEBIAN_FRONTEND=noninteractive
    
    echo "[AppHub] 检查 APT 仓库: ${APPHUB_URL}/pkgs/saltstack-deb/"
    
    # 检查 Packages 文件是否存在
    if ! timeout 10 wget -q --spider "${APPHUB_URL}/pkgs/saltstack-deb/Packages"; then
        echo "❌ AppHub DEB仓库不可用"
        echo "   检查: ${APPHUB_URL}/pkgs/saltstack-deb/Packages"
        exit 1
    fi
    
    echo "✓ AppHub DEB仓库可用"
    
    # 配置 APT 源
    echo "deb [trusted=yes] ${APPHUB_URL}/pkgs/saltstack-deb ./" > /etc/apt/sources.list.d/ai-infra-salt.list
    
    # 更新并安装
    echo "[AppHub] 更新包列表..."
    apt-get update -o Dir::Etc::sourcelist="sources.list.d/ai-infra-salt.list" \
                   -o Dir::Etc::sourceparts="-" \
                   -o APT::Get::List-Cleanup="0"
    
    echo "[AppHub] 安装 salt-minion..."
    if apt-get install -y --no-install-recommends salt-minion; then
        echo "✓ Salt Minion 安装成功"
    else
        echo "❌ 安装失败"
        exit 1
    fi

# ==================== RHEL/Rocky/CentOS 系统 ====================
elif [ "$OS_TYPE" = "rhel-dnf" ] || [ "$OS_TYPE" = "rhel-yum" ]; then
    PKG_MGR="dnf"
    [ "$OS_TYPE" = "rhel-yum" ] && PKG_MGR="yum"
    
    echo "[AppHub] 检查 RPM 仓库: ${APPHUB_URL}/pkgs/saltstack-rpm/"
    
    # 检查 repodata 是否存在
    if ! timeout 10 wget -q --spider "${APPHUB_URL}/pkgs/saltstack-rpm/repodata/repomd.xml"; then
        echo "❌ AppHub RPM仓库不可用"
        echo "   检查: ${APPHUB_URL}/pkgs/saltstack-rpm/repodata/repomd.xml"
        exit 1
    fi
    
    echo "✓ AppHub RPM仓库可用"
    
    # 配置 YUM/DNF 源
    cat > /etc/yum.repos.d/ai-infra-salt.repo <<EOF
[ai-infra-salt]
name=AI Infra Salt RPMs (AppHub)
baseurl=${APPHUB_URL}/pkgs/saltstack-rpm
enabled=1
gpgcheck=0
priority=1
EOF
    
    # 清理缓存并安装
    echo "[AppHub] 清理缓存..."
    ${PKG_MGR} clean all
    
    echo "[AppHub] 更新元数据..."
    ${PKG_MGR} makecache
    
    echo "[AppHub] 安装 salt-minion..."
    if ${PKG_MGR} install -y salt-minion; then
        echo "✓ Salt Minion 安装成功"
    else
        echo "❌ 安装失败"
        exit 1
    fi

# ==================== SUSE 系统 ====================
elif [ "$OS_TYPE" = "suse" ]; then
    echo "❌ SUSE系统暂不支持AppHub安装"
    echo "   请手动配置zypper源指向 ${APPHUB_URL}/pkgs/saltstack-rpm/"
    exit 1
fi

# 验证安装
echo ""
echo "========================================="
if command -v salt-minion >/dev/null 2>&1; then
    echo "✓ Salt Minion 安装成功"
    salt-minion --version
    exit 0
else
    echo "❌ Salt Minion 安装失败"
    echo "   salt-minion 命令未找到"
    exit 1
fi
