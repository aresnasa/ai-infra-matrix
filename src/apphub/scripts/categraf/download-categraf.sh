#!/bin/bash
# =============================================================================
# Categraf Download Script for AppHub
# 下载 Categraf 预编译二进制到 AppHub (多架构支持)
#
# 注意: 此脚本已被整合到统一下载脚本:
#   - 项目根目录: scripts/download_third_party.sh
#   - AppHub 目录: scripts/download-github-release.sh
# 建议使用统一脚本进行下载，此脚本保留用于 AppHub 独立使用场景
# =============================================================================

set -e

# 配置
CATEGRAF_VERSION="${CATEGRAF_VERSION:-v0.4.25}"
# 如果版本号没有 v 前缀，添加 v 前缀 (Categraf release 使用 v 前缀)
if [[ ! "${CATEGRAF_VERSION}" == v* ]]; then
    CATEGRAF_VERSION="v${CATEGRAF_VERSION}"
fi
OUTPUT_DIR="${OUTPUT_DIR:-/usr/share/nginx/html/pkgs/categraf}"
GITHUB_MIRROR="${GITHUB_MIRROR:-https://gh-proxy.com/}"

echo "📦 Downloading Categraf ${CATEGRAF_VERSION}..."

# 创建输出目录
mkdir -p "${OUTPUT_DIR}"

# 下载函数 (带镜像回退)
download_categraf() {
    local arch=$1
    local filename="categraf-${CATEGRAF_VERSION}-linux-${arch}.tar.gz"
    local base_url="https://github.com/flashcatcloud/categraf/releases/download/${CATEGRAF_VERSION}/${filename}"
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
        echo "     Trying mirror..."
        if curl -fsSL -m 30 --retry 3 -o "${OUTPUT_DIR}/${filename}" "${mirror_url}" 2>/dev/null; then
            echo "  ✓ Downloaded ${filename} (via mirror)"
            if command -v sha256sum &> /dev/null; then
                sha256sum "${OUTPUT_DIR}/${filename}" > "${OUTPUT_DIR}/${filename}.sha256"
            fi
            return 0
        fi
        echo "  ⚠ Mirror failed, trying direct download..."
    fi
    
    # 直接下载
    if curl -fsSL -m 60 --retry 3 -o "${OUTPUT_DIR}/${filename}" "${base_url}"; then
        echo "  ✓ Downloaded ${filename}"
        if command -v sha256sum &> /dev/null; then
            sha256sum "${OUTPUT_DIR}/${filename}" > "${OUTPUT_DIR}/${filename}.sha256"
        fi
        return 0
    else
        echo "  ✗ Failed to download ${filename}"
        rm -f "${OUTPUT_DIR}/${filename}"
        return 1
    fi
}

# 创建 latest 符号链接
create_latest_symlinks() {
    local arch=$1
    local filename="categraf-${CATEGRAF_VERSION}-linux-${arch}.tar.gz"
    local latest="categraf-latest-linux-${arch}.tar.gz"
    
    if [ -f "${OUTPUT_DIR}/${filename}" ]; then
        # 先删除已存在的符号链接，避免自引用
        rm -f "${OUTPUT_DIR}/${latest}" 2>/dev/null || true
        cd "${OUTPUT_DIR}" && ln -sf "${filename}" "${latest}"
        echo "  ✓ Created symlink ${latest} -> ${filename}"
    fi
}

# 下载 amd64 版本
download_categraf "amd64"

# 下载 arm64 版本  
download_categraf "arm64"

# 创建 latest 符号链接
create_latest_symlinks "amd64"
create_latest_symlinks "arm64"

# 复制安装脚本
if [ -f "/scripts/categraf/install-categraf.sh" ]; then
    cp /scripts/categraf/install-categraf.sh "${OUTPUT_DIR}/install.sh"
    chmod +x "${OUTPUT_DIR}/install.sh"
    echo "  ✓ Copied install script"
elif [ -f "$(dirname "$0")/install-categraf.sh" ]; then
    cp "$(dirname "$0")/install-categraf.sh" "${OUTPUT_DIR}/install.sh"
    chmod +x "${OUTPUT_DIR}/install.sh"
    echo "  ✓ Copied install script"
fi

# 创建版本信息文件
cat > "${OUTPUT_DIR}/version.json" << EOF
{
    "name": "categraf",
    "version": "${CATEGRAF_VERSION}",
    "files": [
        {
            "filename": "categraf-${CATEGRAF_VERSION}-linux-amd64.tar.gz",
            "arch": "amd64",
            "os": "linux"
        },
        {
            "filename": "categraf-${CATEGRAF_VERSION}-linux-arm64.tar.gz",
            "arch": "arm64",
            "os": "linux"
        }
    ],
    "updated_at": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF

# 写入版本号文件
echo "${CATEGRAF_VERSION}" > "${OUTPUT_DIR}/VERSION"

echo ""
echo "✓ Categraf ${CATEGRAF_VERSION} downloaded successfully"
echo "  Location: ${OUTPUT_DIR}"
ls -la "${OUTPUT_DIR}"
