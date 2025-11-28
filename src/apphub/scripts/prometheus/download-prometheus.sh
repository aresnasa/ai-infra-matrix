#!/bin/bash
# =============================================================================
# Prometheus Download Script for AppHub
# 下载 Prometheus 预编译二进制到 AppHub
#
# 注意: 此脚本已被整合到统一下载脚本:
#   - 项目根目录: scripts/download_third_party.sh
#   - AppHub 目录: scripts/download-github-release.sh
# 建议使用统一脚本进行下载，此脚本保留用于 AppHub 独立使用场景
# =============================================================================

set -e

# 配置
PROMETHEUS_VERSION="${PROMETHEUS_VERSION:-3.7.3}"
# 去掉版本号前的 v 前缀 (如果有)
PROMETHEUS_VERSION="${PROMETHEUS_VERSION#v}"
OUTPUT_DIR="${OUTPUT_DIR:-/usr/share/nginx/html/pkgs/prometheus}"
GITHUB_MIRROR="${GITHUB_MIRROR:-https://gh-proxy.com/}"

echo "📦 Downloading Prometheus ${PROMETHEUS_VERSION}..."

# 创建输出目录
mkdir -p "${OUTPUT_DIR}"

# 下载函数 (带镜像回退)
download_prometheus() {
    local arch=$1
    local filename="prometheus-${PROMETHEUS_VERSION}.linux-${arch}.tar.gz"
    local base_url="https://github.com/prometheus/prometheus/releases/download/v${PROMETHEUS_VERSION}/${filename}"
    # 移除 https:// 前缀避免重复
    local base_url_without_scheme="${base_url#https://}"
    local mirror_url="${GITHUB_MIRROR}${base_url_without_scheme}"
    
    if [ -f "${OUTPUT_DIR}/${filename}" ]; then
        echo "  ✓ ${filename} already exists, skipping"
        return 0
    fi
    
    echo "  📥 Downloading ${filename}..."
    
    # 首先尝试镜像
    if [ -n "${GITHUB_MIRROR}" ]; then
        if curl -fsSL -m 30 --retry 3 -o "${OUTPUT_DIR}/${filename}" "${mirror_url}" 2>/dev/null; then
            echo "  ✓ Downloaded ${filename} (via mirror)"
            sha256sum "${OUTPUT_DIR}/${filename}" > "${OUTPUT_DIR}/${filename}.sha256"
            return 0
        fi
        echo "  ⚠ Mirror failed, trying direct download..."
    fi
    
    # 直接下载
    if curl -fsSL -m 60 --retry 3 -o "${OUTPUT_DIR}/${filename}" "${base_url}"; then
        echo "  ✓ Downloaded ${filename}"
        sha256sum "${OUTPUT_DIR}/${filename}" > "${OUTPUT_DIR}/${filename}.sha256"
        return 0
    else
        echo "  ✗ Failed to download ${filename}"
        rm -f "${OUTPUT_DIR}/${filename}"
        return 1
    fi
}

# 下载 amd64 版本
download_prometheus "amd64"

# 下载 arm64 版本
download_prometheus "arm64"

# 创建版本信息文件
cat > "${OUTPUT_DIR}/version.json" << EOF
{
    "name": "prometheus",
    "version": "${PROMETHEUS_VERSION}",
    "files": [
        {
            "filename": "prometheus-${PROMETHEUS_VERSION}.linux-amd64.tar.gz",
            "arch": "amd64",
            "os": "linux"
        },
        {
            "filename": "prometheus-${PROMETHEUS_VERSION}.linux-arm64.tar.gz",
            "arch": "arm64",
            "os": "linux"
        }
    ],
    "updated_at": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF

echo ""
echo "✓ Prometheus ${PROMETHEUS_VERSION} downloaded successfully"
echo "  Location: ${OUTPUT_DIR}"
ls -la "${OUTPUT_DIR}"
