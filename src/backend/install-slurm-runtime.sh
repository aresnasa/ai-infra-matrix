#!/bin/sh
set -e

# SLURM 客户端运行时安装脚本
# 在容器启动时从 AppHub 下载并安装 SLURM 客户端工具

echo "========================================"
echo "Installing SLURM client from AppHub"
echo "========================================"

# 检查是否已安装
if command -v sinfo >/dev/null 2>&1; then
    echo "✓ SLURM client already installed"
    sinfo --version 2>/dev/null || echo "Version: $(cat /usr/local/slurm/VERSION 2>/dev/null || echo 'unknown')"
    exit 0
fi

# 探测可用的 AppHub URL
APPHUB_URL=""
for URL in http://apphub http://ai-infra-apphub http://localhost:8081 http://192.168.0.200:8081; do
    echo ">>> Probing AppHub: $URL"
    if wget -q --spider --timeout=5 --tries=1 "$URL/packages/" 2>/dev/null; then
        APPHUB_URL="$URL"
        echo "  ✓ AppHub available at: $URL"
        break
    else
        echo "  ✗ AppHub not reachable at: $URL"
    fi
done

# 如果没有可用的 AppHub，跳过安装（允许容器启动）
if [ -z "$APPHUB_URL" ]; then
    echo ""
    echo "⚠️  WARNING: AppHub is not accessible"
    echo ""
    echo "SLURM client installation will be skipped."
    echo "You can install it later by running:"
    echo "  docker exec ai-infra-backend /install-slurm-runtime.sh"
    echo ""
    echo "Or ensure AppHub is running:"
    echo "  docker-compose up -d apphub"
    echo ""
    exit 0
fi

# 下载并执行安装脚本
echo ">>> Downloading SLURM installation script..."
cd /tmp
if wget --timeout=30 --tries=3 -O install-slurm.sh "${APPHUB_URL}/packages/install-slurm.sh" 2>/dev/null; then
    chmod +x install-slurm.sh
    echo "  ✓ Downloaded installation script"
    APPHUB_URL="${APPHUB_URL}" ./install-slurm.sh || {
        echo "❌ ERROR: SLURM installation failed"
        rm -f install-slurm.sh
        exit 1
    }
    rm -f install-slurm.sh
    echo "========================================"
    echo "✓ SLURM client installation completed"
    echo "========================================"
else
    echo ""
    echo "⚠️  WARNING: Failed to download installation script"
    echo ""
    echo "URL: ${APPHUB_URL}/packages/install-slurm.sh"
    echo ""
    echo "SLURM installation skipped. You can retry later."
    echo ""
    exit 0
fi
