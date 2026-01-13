#!/bin/bash
# =============================================================================
# 通用 GitHub Release 下载脚本
# 用于下载 Prometheus、Node Exporter、Categraf 等 GitHub Release 组件
#
# 使用方法:
#   ./download-github-release.sh <component> [version]
#
# 示例:
#   ./download-github-release.sh prometheus 3.7.3
#   ./download-github-release.sh node_exporter 1.8.2
#   ./download-github-release.sh categraf v0.4.25
#   ./download-github-release.sh all  # 下载所有组件
#
# 环境变量:
#   GITHUB_MIRROR  - GitHub 镜像 (默认: https://gh-proxy.com/)
#   OUTPUT_DIR     - 输出目录 (默认: /usr/share/nginx/html/pkgs/<component>)
# =============================================================================
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APPHUB_DIR="$(dirname "$SCRIPT_DIR")"
PROJECT_ROOT="$(dirname "$(dirname "$APPHUB_DIR")")"
CONFIG_FILE="$APPHUB_DIR/build-config.yaml"

# GitHub 镜像加速 (可通过环境变量覆盖)
GITHUB_MIRROR="${GITHUB_MIRROR:-https://gh-proxy.com/}"
# GitHub 代理 (可选，用于 curl --proxy)
GITHUB_PROXY="${GITHUB_PROXY:-}"

# =============================================================================
# 组件配置 (从 build-config.yaml 读取或使用默认值)
# =============================================================================
declare -A COMPONENT_CONFIG

# Prometheus
COMPONENT_CONFIG[prometheus_repo]="prometheus/prometheus"
COMPONENT_CONFIG[prometheus_version]="3.7.3"
COMPONENT_CONFIG[prometheus_pattern]="prometheus-{VERSION}.linux-{ARCH}.tar.gz"
COMPONENT_CONFIG[prometheus_version_prefix]="v"

# Node Exporter
COMPONENT_CONFIG[node_exporter_repo]="prometheus/node_exporter"
COMPONENT_CONFIG[node_exporter_version]="1.8.2"
COMPONENT_CONFIG[node_exporter_pattern]="node_exporter-{VERSION}.linux-{ARCH}.tar.gz"
COMPONENT_CONFIG[node_exporter_version_prefix]="v"

# Alertmanager
COMPONENT_CONFIG[alertmanager_repo]="prometheus/alertmanager"
COMPONENT_CONFIG[alertmanager_version]="0.28.1"
COMPONENT_CONFIG[alertmanager_pattern]="alertmanager-{VERSION}.linux-{ARCH}.tar.gz"
COMPONENT_CONFIG[alertmanager_version_prefix]="v"

# Categraf
COMPONENT_CONFIG[categraf_repo]="flashcatcloud/categraf"
COMPONENT_CONFIG[categraf_version]="v0.4.25"
COMPONENT_CONFIG[categraf_pattern]="categraf-{VERSION}-linux-{ARCH}.tar.gz"
COMPONENT_CONFIG[categraf_version_prefix]=""

# 支持的架构
ARCHITECTURES=("amd64" "arm64")

# =============================================================================
# 工具函数
# =============================================================================

# 去掉版本号的 v 前缀
strip_v_prefix() {
    local ver=$1
    echo "${ver#v}"
}

# 添加 v 前缀
ensure_v_prefix() {
    local ver=$1
    if [[ ! "$ver" =~ ^v ]]; then
        echo "v${ver}"
    else
        echo "$ver"
    fi
}

# 构建下载 URL
build_download_url() {
    local repo=$1
    local tag=$2
    local filename=$3
    echo "https://github.com/${repo}/releases/download/${tag}/${filename}"
}

# 通用下载函数 (支持 GITHUB_MIRROR 和 GITHUB_PROXY 多种方式)
download_file() {
    local url=$1
    local output_file=$2
    
    if [ -f "$output_file" ]; then
        echo "  ✓ Already exists: $(basename "$output_file")"
        return 0
    fi
    
    echo "  📥 Downloading: $(basename "$output_file")..."
    
    # 方式1: 尝试使用 GITHUB_MIRROR 加速下载
    # 注意：不同的镜像服务有不同的 URL 格式
    # - ghfast.top: https://ghfast.top/https://github.com/...
    # - gh-proxy.com: https://gh-proxy.com/https://github.com/...
    # 统一使用完整 URL 拼接方式
    if [[ "$url" == *"github.com"* ]] && [ -n "$GITHUB_MIRROR" ]; then
        local mirror_url="${GITHUB_MIRROR}${url}"
        echo "     [方式1] GITHUB_MIRROR: ${mirror_url}"
        if curl -fsSL --connect-timeout 30 --max-time 300 --retry 3 -o "$output_file" "$mirror_url" 2>/dev/null; then
            echo "  ✓ Downloaded via GITHUB_MIRROR: $(basename "$output_file")"
            if command -v sha256sum &> /dev/null; then
                sha256sum "$output_file" > "${output_file}.sha256"
            fi
            return 0
        fi
        echo "  ⚠️  GITHUB_MIRROR failed, trying next method..."
    fi
    
    # 方式2: 尝试使用 GITHUB_PROXY 代理
    if [ -n "$GITHUB_PROXY" ]; then
        echo "     [方式2] GITHUB_PROXY: ${url}"
        echo "     Using proxy: ${GITHUB_PROXY}"
        if curl --proxy "$GITHUB_PROXY" -fsSL --connect-timeout 30 --max-time 300 --retry 3 -o "$output_file" "$url" 2>/dev/null; then
            echo "  ✓ Downloaded via GITHUB_PROXY: $(basename "$output_file")"
            if command -v sha256sum &> /dev/null; then
                sha256sum "$output_file" > "${output_file}.sha256"
            fi
            return 0
        fi
        echo "  ⚠️  GITHUB_PROXY failed, trying direct download..."
    fi
    
    # 方式3: 直接下载
    echo "     [方式3] Direct: ${url}"
    if curl -fsSL --connect-timeout 30 --max-time 300 --retry 3 -o "$output_file" "$url"; then
        echo "  ✓ Downloaded directly: $(basename "$output_file")"
        if command -v sha256sum &> /dev/null; then
            sha256sum "$output_file" > "${output_file}.sha256"
        fi
        return 0
    fi
    
    echo "  ❌ All download methods failed: $(basename "$output_file")"
    rm -f "$output_file"
    return 1
}

# 生成版本信息文件
generate_version_json() {
    local output_dir=$1
    local component=$2
    local version=$3
    shift 3
    local files=("$@")
    
    local files_json=""
    for f in "${files[@]}"; do
        local arch="amd64"
        [[ "$f" == *"arm64"* || "$f" == *"aarch64"* ]] && arch="arm64"
        files_json="${files_json}
        {
            \"filename\": \"${f}\",
            \"arch\": \"${arch}\",
            \"os\": \"linux\"
        },"
    done
    # 去掉最后的逗号
    files_json="${files_json%,}"
    
    cat > "${output_dir}/version.json" << EOF
{
    "name": "${component}",
    "version": "${version}",
    "files": [${files_json}
    ],
    "updated_at": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF
}

# =============================================================================
# 组件下载函数
# =============================================================================

download_component() {
    local component=$1
    local version_override=$2
    
    local repo_key="${component}_repo"
    local version_key="${component}_version"
    local pattern_key="${component}_pattern"
    local prefix_key="${component}_version_prefix"
    
    local repo="${COMPONENT_CONFIG[$repo_key]}"
    local default_version="${COMPONENT_CONFIG[$version_key]}"
    local pattern="${COMPONENT_CONFIG[$pattern_key]}"
    local version_prefix="${COMPONENT_CONFIG[$prefix_key]}"
    
    if [ -z "$repo" ]; then
        echo "❌ Unknown component: ${component}"
        echo "   Supported components: prometheus, node_exporter, alertmanager, categraf"
        return 1
    fi
    
    # 使用覆盖版本或默认版本
    local version="${version_override:-$default_version}"
    
    # 处理版本前缀
    local version_num=$(strip_v_prefix "$version")
    local version_tag
    if [ -n "$version_prefix" ]; then
        version_tag="${version_prefix}${version_num}"
    else
        version_tag="$version"
    fi
    
    # 确定输出目录
    local output_dir="${OUTPUT_DIR:-/usr/share/nginx/html/pkgs/${component}}"
    
    echo "================================================================"
    echo "📦 Downloading ${component} ${version_tag}"
    echo "   Repository: ${repo}"
    echo "   Output: ${output_dir}"
    echo "   Mirror: ${GITHUB_MIRROR:-<disabled>}"
    echo "================================================================"
    
    mkdir -p "$output_dir"
    
    local downloaded_files=()
    
    for arch in "${ARCHITECTURES[@]}"; do
        # 替换模式中的占位符
        local filename=$(echo "$pattern" | sed "s/{VERSION}/${version_num}/g" | sed "s/{ARCH}/${arch}/g")
        local url=$(build_download_url "$repo" "$version_tag" "$filename")
        local output_file="${output_dir}/${filename}"
        
        if download_file "$url" "$output_file"; then
            downloaded_files+=("$filename")
        fi
    done
    
    # 生成版本信息
    if [ ${#downloaded_files[@]} -gt 0 ]; then
        generate_version_json "$output_dir" "$component" "$version_tag" "${downloaded_files[@]}"
        echo "$version_tag" > "${output_dir}/VERSION"
        echo ""
        echo "✓ ${component} ${version_tag} download complete"
        echo "  Files: ${downloaded_files[*]}"
    else
        echo ""
        echo "⚠️  No files downloaded for ${component}"
    fi
    
    return 0
}

# =============================================================================
# 主程序
# =============================================================================

usage() {
    cat << EOF
Usage: $(basename "$0") <component> [version]

Download GitHub Release artifacts for AppHub components.

Components:
  prometheus     - Prometheus monitoring system
  node_exporter  - Prometheus node exporter
  alertmanager   - Prometheus alertmanager
  categraf       - Categraf monitoring agent
  all            - Download all components

Environment Variables:
  GITHUB_MIRROR  - GitHub mirror URL (default: https://gh-proxy.com/)
  GITHUB_PROXY   - HTTP proxy for GitHub access (e.g., http://proxy:8080)
  OUTPUT_DIR     - Override output directory

Examples:
  $(basename "$0") prometheus 3.7.3
  $(basename "$0") node_exporter
  GITHUB_MIRROR="" $(basename "$0") categraf v0.4.25
  $(basename "$0") all

EOF
    exit 1
}

# 解析参数
COMPONENT="${1:-}"
VERSION="${2:-}"

if [ -z "$COMPONENT" ]; then
    usage
fi

echo "================================================================"
echo "  GitHub Release Downloader for AppHub"
echo "================================================================"
echo "GitHub Mirror: ${GITHUB_MIRROR:-<disabled>}"
echo "GitHub Proxy:  ${GITHUB_PROXY:-<disabled>}"
echo ""

case "$COMPONENT" in
    prometheus|node_exporter|alertmanager|categraf)
        download_component "$COMPONENT" "$VERSION"
        ;;
    all)
        for comp in prometheus node_exporter alertmanager categraf; do
            download_component "$comp" "" || true
            echo ""
        done
        ;;
    *)
        echo "❌ Unknown component: ${COMPONENT}"
        usage
        ;;
esac

echo ""
echo "✅ Download complete!"
