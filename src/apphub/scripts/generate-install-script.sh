#!/bin/sh
# 生成 SLURM 安装脚本
# 从模板生成实际的安装脚本并放置到 nginx 目录

set -e

TEMPLATE_FILE="${1:-/app/scripts/install-slurm.sh.tmpl}"
OUTPUT_FILE="${2:-/usr/share/nginx/html/packages/install-slurm.sh}"
SLURM_VERSION="${SLURM_VERSION:-25.05.4}"
APPHUB_BASE_URL="${APPHUB_BASE_URL:-http://localhost:8081}"

echo "Generating SLURM installation script..."
echo "  Template: ${TEMPLATE_FILE}"
echo "  Output: ${OUTPUT_FILE}"
echo "  Version: ${SLURM_VERSION}"
echo "  Base URL: ${APPHUB_BASE_URL}"

# 创建输出目录
mkdir -p "$(dirname "${OUTPUT_FILE}")"

# 使用 sed 替换模板变量
if [ -f "${TEMPLATE_FILE}" ]; then
    sed -e "s|{{SLURM_VERSION}}|${SLURM_VERSION}|g" \
        -e "s|{{APPHUB_BASE_URL}}|${APPHUB_BASE_URL}|g" \
        "${TEMPLATE_FILE}" > "${OUTPUT_FILE}"
    
    chmod +x "${OUTPUT_FILE}"
    echo "✓ Generated: ${OUTPUT_FILE}"
else
    echo "❌ Template file not found: ${TEMPLATE_FILE}"
    exit 1
fi
