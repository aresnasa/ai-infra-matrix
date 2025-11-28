#!/bin/bash
# =============================================================================
# Prometheus Download Script for AppHub
# 下载 Prometheus 预编译二进制到 AppHub
# =============================================================================

set -e

# 配置
PROMETHEUS_VERSION="${PROMETHEUS_VERSION:-3.7.3}"
OUTPUT_DIR="${OUTPUT_DIR:-/usr/share/nginx/html/pkgs/prometheus}"
GITHUB_MIRROR="${GITHUB_MIRROR:-}"

echo "📦 Downloading Prometheus ${PROMETHEUS_VERSION}..."

# 创建输出目录
mkdir -p "${OUTPUT_DIR}"

# 下载函数
download_prometheus() {
    local arch=$1
    local filename="prometheus-${PROMETHEUS_VERSION}.linux-${arch}.tar.gz"
    local url="https://github.com/prometheus/prometheus/releases/download/v${PROMETHEUS_VERSION}/${filename}"
    
    # 如果配置了 GitHub 镜像
    if [ -n "${GITHUB_MIRROR}" ]; then
        url="${GITHUB_MIRROR}/https://github.com/prometheus/prometheus/releases/download/v${PROMETHEUS_VERSION}/${filename}"
    fi
    
    echo "  Downloading ${filename}..."
    
    if [ -f "${OUTPUT_DIR}/${filename}" ]; then
        echo "  ✓ ${filename} already exists, skipping"
        return 0
    fi
    
    if curl -fsSL -o "${OUTPUT_DIR}/${filename}" "${url}"; then
        echo "  ✓ Downloaded ${filename}"
        
        # 生成校验和
        sha256sum "${OUTPUT_DIR}/${filename}" > "${OUTPUT_DIR}/${filename}.sha256"
        echo "  ✓ Generated checksum"
        
        return 0
    else
        echo "  ✗ Failed to download ${filename}"
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
