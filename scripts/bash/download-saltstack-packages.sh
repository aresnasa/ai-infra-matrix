#!/bin/bash

# SaltStack Package Downloader
# 下载SaltStack Minion的deb包

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
DEB_DIR="$PROJECT_DIR/pkgs/saltstack-deb"
RPM_DIR="$PROJECT_DIR/pkgs/saltstack-rpm"

# 创建目录
mkdir -p "$DEB_DIR" "$RPM_DIR"

echo "开始下载SaltStack Minion包..."

# DEB包（Ubuntu 22.04 / Debian）
echo "下载DEB包..."
cd "$DEB_DIR"

# SaltStack官方包源
SALT_VERSION="3007.1"
BASE_URL="https://repo.saltproject.io/salt/py3/ubuntu/22.04/amd64/archive/${SALT_VERSION}"

# 下载主要的deb包
packages=(
    "salt-common_${SALT_VERSION}~ubuntu22.04_all.deb"
    "salt-minion_${SALT_VERSION}~ubuntu22.04_all.deb"
    "python3-salt_${SALT_VERSION}~ubuntu22.04_all.deb"
)

for package in "${packages[@]}"; do
    if [ ! -f "$package" ]; then
        echo "下载: $package"
        curl -L -s -o "$package" "$BASE_URL/$package" || {
            echo "警告: 无法下载 $package"
            rm -f "$package"  # 删除可能的空文件
        }
    else
        echo "已存在: $package"
    fi
done

# 如果直接下载失败，尝试使用Docker容器获取包
if [ ! -f "salt-minion_${SALT_VERSION}"* ] && [ ! -f "salt-minion_"* ]; then
    echo "尝试从Docker容器获取SaltStack包..."
    
    # 使用Ubuntu容器安装并提取包
    docker run --rm -v "$DEB_DIR:/output" ubuntu:22.04 /bin/bash -c "
        export DEBIAN_FRONTEND=noninteractive
        apt-get update -qq
        apt-get download salt-minion salt-common python3-salt 2>/dev/null || true
        cp *.deb /output/ 2>/dev/null || true
        ls -la /output/
    " || echo "Docker下载也失败，将使用在线安装方式"
fi

# RPM包（CentOS/RHEL）
echo "下载RPM包..."
cd "$RPM_DIR"

# 尝试下载RPM包
RPM_BASE_URL="https://repo.saltproject.io/salt/py3/redhat/8/x86_64/archive/${SALT_VERSION}"
rpm_packages=(
    "salt-minion-${SALT_VERSION}-1.el8.noarch.rpm"
    "salt-${SALT_VERSION}-1.el8.noarch.rpm"
    "python38-salt-${SALT_VERSION}-1.el8.noarch.rpm"
)

for package in "${rpm_packages[@]}"; do
    if [ ! -f "$package" ]; then
        echo "下载: $package"
        curl -L -s -o "$package" "$RPM_BASE_URL/$package" || {
            echo "警告: 无法下载 $package"
            rm -f "$package"  # 删除可能的空文件
        }
    else
        echo "已存在: $package"
    fi
done

echo "SaltStack包下载完成！"
echo "DEB包位置: $DEB_DIR"
echo "RPM包位置: $RPM_DIR"

# 显示下载的文件
echo -e "\n下载的DEB包:"
ls -la "$DEB_DIR" | grep "\.deb" || echo "无deb包"

echo -e "\n下载的RPM包:"
ls -la "$RPM_DIR" | grep "\.rpm" || echo "无rpm包"