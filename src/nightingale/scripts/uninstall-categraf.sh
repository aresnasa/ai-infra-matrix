#!/bin/bash
# Categraf 监控代理卸载脚本
#
# 使用示例:
#   curl -fsSL http://nightingale:17000/api/v1/scripts/uninstall-categraf.sh | bash

set -e

echo "=============================================="
echo "  Categraf Uninstaller"
echo "=============================================="

INSTALL_DIR="/opt/categraf"

echo "[1/3] Stopping Categraf service..."
if systemctl is-active --quiet categraf 2>/dev/null; then
    sudo systemctl stop categraf
    echo "  ✓ Service stopped"
else
    echo "  - Service not running"
fi

echo "[2/3] Disabling and removing service..."
if [ -f /etc/systemd/system/categraf.service ]; then
    sudo systemctl disable categraf 2>/dev/null || true
    sudo rm -f /etc/systemd/system/categraf.service
    sudo systemctl daemon-reload
    echo "  ✓ Service removed"
else
    echo "  - Service file not found"
fi

echo "[3/3] Removing installation directory..."
if [ -d "${INSTALL_DIR}" ]; then
    sudo rm -rf "${INSTALL_DIR}"
    echo "  ✓ Directory removed: ${INSTALL_DIR}"
else
    echo "  - Directory not found: ${INSTALL_DIR}"
fi

echo ""
echo "=============================================="
echo "✓ Categraf uninstalled successfully!"
echo "=============================================="
