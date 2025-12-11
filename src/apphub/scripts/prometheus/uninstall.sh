#!/bin/bash
# =============================================================================
# Prometheus Uninstallation Script
# 卸载 Prometheus
# =============================================================================

set -e

INSTALL_DIR="${INSTALL_DIR:-/usr/local/prometheus}"
DATA_DIR="${DATA_DIR:-/var/lib/prometheus}"
CONFIG_DIR="${CONFIG_DIR:-/etc/prometheus}"
USER="${PROMETHEUS_USER:-prometheus}"
GROUP="${PROMETHEUS_GROUP:-prometheus}"

echo "==================================="
echo "Uninstalling Prometheus"
echo "==================================="

# 停止服务
if systemctl is-active prometheus > /dev/null 2>&1; then
    echo "Stopping Prometheus service..."
    systemctl stop prometheus
fi

# 禁用服务
if systemctl is-enabled prometheus > /dev/null 2>&1; then
    echo "Disabling Prometheus service..."
    systemctl disable prometheus
fi

# 删除服务文件
if [ -f /etc/systemd/system/prometheus.service ]; then
    echo "Removing systemd service..."
    rm -f /etc/systemd/system/prometheus.service
    systemctl daemon-reload
fi

# 删除二进制文件
if [ -d ${INSTALL_DIR} ]; then
    echo "Removing binaries..."
    rm -rf ${INSTALL_DIR}
fi

# 询问是否删除数据
read -p "Delete data directory (${DATA_DIR})? [y/N] " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    rm -rf ${DATA_DIR}
    echo "✓ Data directory removed"
fi

# 询问是否删除配置
read -p "Delete configuration (${CONFIG_DIR})? [y/N] " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    rm -rf ${CONFIG_DIR}
    echo "✓ Configuration removed"
fi

# 删除用户和组（可选）
read -p "Remove prometheus user and group? [y/N] " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    userdel ${USER} 2>/dev/null || true
    groupdel ${GROUP} 2>/dev/null || true
    echo "✓ User and group removed"
fi

echo ""
echo "✓ Prometheus uninstalled successfully"
