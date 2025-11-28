#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
THIRD_PARTY_DIR="$PROJECT_ROOT/third_party"
DOCKERFILE="$PROJECT_ROOT/src/apphub/Dockerfile"
ENV_FILE="$PROJECT_ROOT/.env"

mkdir -p "$THIRD_PARTY_DIR"

# GitHub 镜像加速 (可通过环境变量覆盖)
GITHUB_MIRROR="${GITHUB_MIRROR:-https://gh-proxy.com/}"

# Helper to extract ARG value from Dockerfile
get_arg() {
    local name=$1
    grep "ARG $name=" "$DOCKERFILE" | head -1 | cut -d'=' -f2 | tr -d '"' | tr -d ' '
}

# Helper to extract variable from RUN command
get_run_var() {
    local name=$1
    # Use sed to extract value between quotes or after =, handling quotes, spaces, semicolons, and backslashes
    grep "$name=" "$DOCKERFILE" | head -1 | sed -E "s/.*$name=\"?([^ \";\\\\]+)\"?.*/\1/"
}

# Helper to extract version from .env file
get_env_var() {
    local name=$1
    local default=$2
    if [ -f "$ENV_FILE" ]; then
        local val=$(grep "^$name=" "$ENV_FILE" | head -1 | cut -d'=' -f2 | tr -d '"' | tr -d ' ')
        echo "${val:-$default}"
    else
        echo "$default"
    fi
}

# 确保版本号以 v 开头
ensure_v_prefix() {
    local ver=$1
    if [[ ! "$ver" =~ ^v ]]; then
        echo "v${ver}"
    else
        echo "$ver"
    fi
}

# 确保版本号不以 v 开头
strip_v_prefix() {
    local ver=$1
    echo "${ver#v}"
}

# 通用下载函数
# Usage: download_file <url> <output_file> [use_mirror]
download_file() {
    local url=$1
    local output_file=$2
    local use_mirror=${3:-true}
    local final_url="$url"
    
    if [ "$use_mirror" = true ] && [[ "$url" == *"github.com"* ]] && [ -n "$GITHUB_MIRROR" ]; then
        final_url="${GITHUB_MIRROR}${url}"
    fi
    
    if [ -f "$output_file" ]; then
        echo "✓ Already exists: $(basename "$output_file")"
        return 0
    fi
    
    echo "📥 Downloading: $(basename "$output_file")..."
    
    # 首先尝试镜像
    if wget -nv -T 30 -t 3 "$final_url" -O "$output_file" 2>/dev/null; then
        echo "✓ Downloaded: $(basename "$output_file")"
        return 0
    fi
    
    # 镜像失败则尝试直接下载
    if [ "$final_url" != "$url" ]; then
        echo "⚠ Mirror failed, trying direct download..."
        if wget -nv -T 60 -t 3 "$url" -O "$output_file" 2>/dev/null; then
            echo "✓ Downloaded (direct): $(basename "$output_file")"
            return 0
        fi
    fi
    
    echo "✗ Failed to download: $(basename "$output_file")"
    rm -f "$output_file"
    return 1
}

# 通用 GitHub Release 下载函数
# Usage: download_github_release <owner/repo> <version> <filename_pattern> <output_dir> <architectures>
# filename_pattern 支持 {VERSION} 和 {ARCH} 占位符
download_github_release() {
    local repo=$1
    local version=$2
    local pattern=$3
    local output_dir=$4
    shift 4
    local archs=("$@")
    
    mkdir -p "$output_dir"
    
    for arch in "${archs[@]}"; do
        local filename=$(echo "$pattern" | sed "s/{VERSION}/$version/g" | sed "s/{ARCH}/$arch/g")
        local url="https://github.com/${repo}/releases/download/${version}/${filename}"
        local output_file="${output_dir}/${filename}"
        
        download_file "$url" "$output_file" true || true
    done
}

# 生成 version.json 文件
generate_version_json() {
    local output_dir=$1
    local component=$2
    local version=$3
    
    cat > "${output_dir}/version.json" << EOF
{
    "component": "${component}",
    "version": "${version}",
    "downloaded_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
}

# =============================================================================
# 获取版本号
# =============================================================================

SALTSTACK_VERSION=$(get_arg SALTSTACK_VERSION)
CATEGRAF_VERSION=$(get_arg CATEGRAF_VERSION)
SINGULARITY_VERSION=$(get_arg SINGULARITY_VERSION)
MUNGE_VERSION=$(get_run_var MUNGE_VERSION)
PROMETHEUS_VERSION=$(get_env_var PROMETHEUS_VERSION "v3.4.1")
NODE_EXPORTER_VERSION=$(get_env_var NODE_EXPORTER_VERSION "v1.8.2")
ALERTMANAGER_VERSION=$(get_env_var ALERTMANAGER_VERSION "v0.28.1")

# 默认值
[ -z "$MUNGE_VERSION" ] && MUNGE_VERSION="0.5.16"

echo "================================================================"
echo "  Third-Party Dependencies Downloader"
echo "================================================================"
echo "Versions detected:"
echo "  SaltStack:      $SALTSTACK_VERSION"
echo "  Categraf:       $CATEGRAF_VERSION"
echo "  Singularity:    $SINGULARITY_VERSION"
echo "  Munge:          $MUNGE_VERSION"
echo "  Prometheus:     $PROMETHEUS_VERSION"
echo "  Node Exporter:  $NODE_EXPORTER_VERSION"
echo "  Alertmanager:   $ALERTMANAGER_VERSION"
echo "================================================================"
echo ""

# =============================================================================
# 1. Prometheus (Tarball)
# =============================================================================
echo "----------------------------------------------------------------"
echo "📦 Processing Prometheus..."
PROMETHEUS_DIR="$THIRD_PARTY_DIR/prometheus"
mkdir -p "$PROMETHEUS_DIR"

PROMETHEUS_VERSION=$(ensure_v_prefix "$PROMETHEUS_VERSION")
PROMETHEUS_VER_NUM=$(strip_v_prefix "$PROMETHEUS_VERSION")

download_github_release \
    "prometheus/prometheus" \
    "$PROMETHEUS_VERSION" \
    "prometheus-{VERSION}-linux-{ARCH}.tar.gz" \
    "$PROMETHEUS_DIR" \
    "amd64" "arm64"

# 替换文件名中的版本号为不带 v 的格式
for arch in amd64 arm64; do
    old_file="${PROMETHEUS_DIR}/prometheus-${PROMETHEUS_VERSION}-linux-${arch}.tar.gz"
    new_file="${PROMETHEUS_DIR}/prometheus-${PROMETHEUS_VER_NUM}.linux-${arch}.tar.gz"
    if [ -f "$old_file" ] && [ ! -f "$new_file" ]; then
        mv "$old_file" "$new_file"
    fi
done

# Prometheus 使用不同的命名格式，重新下载正确的文件
for arch in amd64 arm64; do
    filename="prometheus-${PROMETHEUS_VER_NUM}.linux-${arch}.tar.gz"
    url="https://github.com/prometheus/prometheus/releases/download/${PROMETHEUS_VERSION}/${filename}"
    output_file="${PROMETHEUS_DIR}/${filename}"
    download_file "$url" "$output_file" true || true
done

generate_version_json "$PROMETHEUS_DIR" "prometheus" "$PROMETHEUS_VERSION"

# =============================================================================
# 2. Node Exporter (Tarball)
# =============================================================================
echo "----------------------------------------------------------------"
echo "📦 Processing Node Exporter..."
NODE_EXPORTER_DIR="$THIRD_PARTY_DIR/node_exporter"
mkdir -p "$NODE_EXPORTER_DIR"

NODE_EXPORTER_VERSION=$(ensure_v_prefix "$NODE_EXPORTER_VERSION")
NODE_EXPORTER_VER_NUM=$(strip_v_prefix "$NODE_EXPORTER_VERSION")

for arch in amd64 arm64; do
    filename="node_exporter-${NODE_EXPORTER_VER_NUM}.linux-${arch}.tar.gz"
    url="https://github.com/prometheus/node_exporter/releases/download/${NODE_EXPORTER_VERSION}/${filename}"
    output_file="${NODE_EXPORTER_DIR}/${filename}"
    download_file "$url" "$output_file" true || true
done

generate_version_json "$NODE_EXPORTER_DIR" "node_exporter" "$NODE_EXPORTER_VERSION"

# =============================================================================
# 3. Alertmanager (Tarball)
# =============================================================================
echo "----------------------------------------------------------------"
echo "📦 Processing Alertmanager..."
ALERTMANAGER_DIR="$THIRD_PARTY_DIR/alertmanager"
mkdir -p "$ALERTMANAGER_DIR"

ALERTMANAGER_VERSION=$(ensure_v_prefix "$ALERTMANAGER_VERSION")
ALERTMANAGER_VER_NUM=$(strip_v_prefix "$ALERTMANAGER_VERSION")

for arch in amd64 arm64; do
    filename="alertmanager-${ALERTMANAGER_VER_NUM}.linux-${arch}.tar.gz"
    url="https://github.com/prometheus/alertmanager/releases/download/${ALERTMANAGER_VERSION}/${filename}"
    output_file="${ALERTMANAGER_DIR}/${filename}"
    download_file "$url" "$output_file" true || true
done

generate_version_json "$ALERTMANAGER_DIR" "alertmanager" "$ALERTMANAGER_VERSION"

# =============================================================================
# 4. Categraf (Tarball)
# =============================================================================
echo "----------------------------------------------------------------"
echo "📦 Processing Categraf..."
CATEGRAF_DIR="$THIRD_PARTY_DIR/categraf"
mkdir -p "$CATEGRAF_DIR"

CATEGRAF_VERSION=$(ensure_v_prefix "$CATEGRAF_VERSION")

for arch in amd64 arm64; do
    filename="categraf-${CATEGRAF_VERSION}-linux-${arch}.tar.gz"
    url="https://github.com/flashcatcloud/categraf/releases/download/${CATEGRAF_VERSION}/${filename}"
    output_file="${CATEGRAF_DIR}/${filename}"
    download_file "$url" "$output_file" true || true
done

generate_version_json "$CATEGRAF_DIR" "categraf" "$CATEGRAF_VERSION"

# =============================================================================
# 5. Munge (Tarball)
# =============================================================================
echo "----------------------------------------------------------------"
echo "📦 Processing Munge..."
MUNGE_DIR="$THIRD_PARTY_DIR/munge"
mkdir -p "$MUNGE_DIR"

MUNGE_FILE="munge-${MUNGE_VERSION}.tar.xz"
url="https://github.com/dun/munge/releases/download/munge-${MUNGE_VERSION}/${MUNGE_FILE}"
download_file "$url" "${MUNGE_DIR}/${MUNGE_FILE}" true || true

generate_version_json "$MUNGE_DIR" "munge" "$MUNGE_VERSION"

# =============================================================================
# 6. Singularity (DEB)
# =============================================================================
echo "----------------------------------------------------------------"
echo "📦 Processing Singularity..."
SINGULARITY_DIR="$THIRD_PARTY_DIR/singularity"
mkdir -p "$SINGULARITY_DIR"

SINGULARITY_VERSION=$(ensure_v_prefix "$SINGULARITY_VERSION")
SINGULARITY_VER_NUM=$(strip_v_prefix "$SINGULARITY_VERSION")

for arch in amd64 arm64; do
    filename="singularity-ce_${SINGULARITY_VER_NUM}-1~ubuntu22.04_${arch}.deb"
    url="https://github.com/sylabs/singularity/releases/download/${SINGULARITY_VERSION}/${filename}"
    output_file="${SINGULARITY_DIR}/${filename}"
    download_file "$url" "$output_file" true || true
done

generate_version_json "$SINGULARITY_DIR" "singularity" "$SINGULARITY_VERSION"

# =============================================================================
# 7. SaltStack (DEB & RPM)
# =============================================================================
echo "----------------------------------------------------------------"
echo "📦 Processing SaltStack..."
SALT_DIR="$THIRD_PARTY_DIR/saltstack"
mkdir -p "$SALT_DIR"

SALTSTACK_VERSION=$(ensure_v_prefix "$SALTSTACK_VERSION")
SALT_VER_NUM=$(strip_v_prefix "$SALTSTACK_VERSION")

# DEB packages
for arch in amd64 arm64; do
    for pkg in salt-common salt-master salt-minion salt-api salt-ssh salt-syndic salt-cloud; do
        filename="${pkg}_${SALT_VER_NUM}_${arch}.deb"
        url="https://github.com/saltstack/salt/releases/download/${SALTSTACK_VERSION}/${filename}"
        output_file="${SALT_DIR}/${filename}"
        download_file "$url" "$output_file" true || true
    done
done

# RPM packages
for arch in x86_64 aarch64; do
    for pkg in salt salt-master salt-minion salt-api salt-ssh salt-syndic salt-cloud; do
        filename="${pkg}-${SALT_VER_NUM}-0.${arch}.rpm"
        url="https://github.com/saltstack/salt/releases/download/${SALTSTACK_VERSION}/${filename}"
        output_file="${SALT_DIR}/${filename}"
        download_file "$url" "$output_file" true || true
    done
done

generate_version_json "$SALT_DIR" "saltstack" "$SALTSTACK_VERSION"

# =============================================================================
# 完成
# =============================================================================
echo ""
echo "================================================================"
echo "✅ All third-party dependencies processed."
echo "================================================================"
echo ""
echo "Downloaded files location: $THIRD_PARTY_DIR"
echo ""
ls -la "$THIRD_PARTY_DIR"
