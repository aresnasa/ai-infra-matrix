#!/bin/bash
set -e

echo "Uninstalling Categraf..."

# 停止并禁用服务
if [ -f /etc/systemd/system/categraf.service ]; then
    systemctl stop categraf 2>/dev/null || true
    systemctl disable categraf 2>/dev/null || true
    rm -f /etc/systemd/system/categraf.service
    systemctl daemon-reload
fi

# 删除安装目录
rm -rf /usr/local/categraf

# 从 PATH 移除
if [ -f /etc/profile ]; then
    sed -i '/\/usr\/local\/categraf\/bin/d' /etc/profile
fi

echo "✓ Categraf uninstalled"
