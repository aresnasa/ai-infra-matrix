#!/bin/bash
#
# Third-Party Dependencies Downloader
# 统一的第三方依赖下载脚本，支持 GitHub 镜像加速
#
# 用法:
#   ./download_third_party.sh [options] [component...]
#
# 选项:
#   -h, --help          显示帮助信息
#   -l, --list          列出所有可用组件
#   -v, --version VER   指定组件版本 (仅当下载单个组件时有效)
#   -a, --arch ARCH     指定架构 (amd64, arm64, all), 默认: all
#   -m, --mirror URL    设置 GitHub 镜像 URL
#   --no-mirror         禁用 GitHub 镜像
#
# 示例:
#   ./download_third_party.sh                    # 下载所有组件
#   ./download_third_party.sh prometheus         # 只下载 Prometheus
#   ./download_third_party.sh -v 3.4.1 prometheus # 下载指定版本
#   ./download_third_party.sh --no-mirror prometheus  # 禁用镜像下载
#   GITHUB_MIRROR="https://ghproxy.net/" ./download_third_party.sh prometheus
#
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
THIRD_PARTY_DIR="$PROJECT_ROOT/third_party"
COMPONENTS_JSON="$THIRD_PARTY_DIR/components.json"
ENV_FILE="$PROJECT_ROOT/.env"
DOCKERFILE="$PROJECT_ROOT/src/apphub/Dockerfile"

# GitHub 镜像加速 (可通过环境变量或参数覆盖)
GITHUB_MIRROR="${GITHUB_MIRROR:-https://gh-proxy.com/}"

# 默认架构
TARGET_ARCH="all"

# 指定版本
SPECIFIED_VERSION=""

# =============================================================================
# 工具函数
# =============================================================================

# 显示帮助信息
show_help() {
    head -30 "$0" | grep -E "^#" | sed 's/^# \?//'
    echo ""
    echo "可用组件:"
    list_components
}

# 列出所有可用组件
list_components() {
    if [ ! -f "$COMPONENTS_JSON" ]; then
        echo "❌ 配置文件不存在: $COMPONENTS_JSON"
        exit 1
    fi
    
    echo ""
    echo "组件名称          描述"
    echo "---------------   --------------------------------------------------"
    
    # 使用 jq 解析 JSON
    if command -v jq &> /dev/null; then
        jq -r '.components | to_entries[] | "\(.key)\t\(.value.description)"' "$COMPONENTS_JSON" | \
            while IFS=$'\t' read -r name desc; do
                printf "%-17s %s\n" "$name" "$desc"
            done
    else
        # 简单的 grep 解析
        grep -E '"[a-z_]+":' "$COMPONENTS_JSON" | head -20 | sed 's/.*"\([^"]*\)".*/\1/' | grep -v "components"
    fi
    echo ""
}

# 从 JSON 获取组件属性
get_component_prop() {
    local component=$1
    local prop=$2
    local default=${3:-}
    
    if command -v jq &> /dev/null; then
        local val=$(jq -r ".components.${component}.${prop} // empty" "$COMPONENTS_JSON" 2>/dev/null)
        echo "${val:-$default}"
    else
        echo "$default"
    fi
}

# 从 JSON 获取数组属性
get_component_array() {
    local component=$1
    local prop=$2
    
    if command -v jq &> /dev/null; then
        jq -r ".components.${component}.${prop}[]? // empty" "$COMPONENTS_JSON" 2>/dev/null
    fi
}

# 从环境变量或 .env 文件获取版本
# 优先级: 已加载的环境变量 > .env 文件 > 默认值
get_env_version() {
    local var_name=$1
    local default=$2
    
    # 优先使用已加载的环境变量
    local env_val="${!var_name:-}"
    if [ -n "$env_val" ]; then
        echo "$env_val"
        return
    fi
    
    # 其次从 .env 文件读取
    if [ -f "$ENV_FILE" ]; then
        local val=$(grep "^${var_name}=" "$ENV_FILE" 2>/dev/null | head -1 | cut -d'=' -f2 | tr -d '"' | tr -d ' ')
        echo "${val:-$default}"
    else
        echo "$default"
    fi
}

# 从 Dockerfile 获取 ARG 值
get_dockerfile_arg() {
    local name=$1
    local default=$2
    
    if [ -f "$DOCKERFILE" ]; then
        local val=$(grep "ARG $name=" "$DOCKERFILE" 2>/dev/null | head -1 | cut -d'=' -f2 | tr -d '"' | tr -d ' ')
        echo "${val:-$default}"
    else
        echo "$default"
    fi
}

# 确保版本号有正确的前缀
ensure_prefix() {
    local ver=$1
    local prefix=$2
    
    if [ -z "$prefix" ] || [ "$prefix" = "v" ]; then
        if [[ ! "$ver" =~ ^v ]]; then
            echo "v${ver}"
        else
            echo "$ver"
        fi
    elif [ "$prefix" = "munge-" ]; then
        if [[ ! "$ver" =~ ^munge- ]]; then
            echo "munge-${ver}"
        else
            echo "$ver"
        fi
    elif [ "$prefix" = "slurm-" ]; then
        if [[ ! "$ver" =~ ^slurm- ]]; then
            echo "slurm-${ver}"
        else
            echo "$ver"
        fi
    else
        echo "${ver}"
    fi
}

# 去除版本前缀
strip_prefix() {
    local ver=$1
    ver="${ver#v}"
    ver="${ver#munge-}"
    ver="${ver#slurm-}"
    echo "$ver"
}

# 通用下载函数
download_file() {
    local url=$1
    local output_file=$2
    local use_mirror=${3:-true}
    local final_url="$url"
    
    # 应用 GitHub 镜像
    if [ "$use_mirror" = true ] && [[ "$url" == *"github.com"* ]] && [ -n "$GITHUB_MIRROR" ]; then
        local url_without_scheme="${url#https://}"
        final_url="${GITHUB_MIRROR}${url_without_scheme}"
    fi
    
    # 检查文件是否已存在且非空
    if [ -f "$output_file" ] && [ -s "$output_file" ]; then
        echo "  ✓ 已存在: $(basename "$output_file")"
        return 0
    fi
    
    # 删除可能存在的空文件
    [ -f "$output_file" ] && rm -f "$output_file"
    
    echo "  📥 下载中: $(basename "$output_file")"
    echo "     URL: $final_url"
    
    # 首先尝试镜像 (10秒超时)
    if wget -q --show-progress -T 10 -t 2 "$final_url" -O "$output_file" 2>/dev/null; then
        if [ -s "$output_file" ]; then
            echo "  ✓ 下载成功: $(basename "$output_file")"
            return 0
        fi
    fi
    
    # 镜像失败则尝试直接下载 (30秒超时)
    if [ "$final_url" != "$url" ]; then
        echo "  ⚠ 镜像下载失败，尝试直接下载..."
        rm -f "$output_file"
        if wget -q --show-progress -T 30 -t 2 "$url" -O "$output_file" 2>/dev/null; then
            if [ -s "$output_file" ]; then
                echo "  ✓ 直接下载成功: $(basename "$output_file")"
                return 0
            fi
        fi
    fi
    
    echo "  ✗ 下载失败: $(basename "$output_file")"
    rm -f "$output_file"
    return 1
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
# 组件下载函数
# =============================================================================

# 通用 GitHub Release 下载
download_component() {
    local component=$1
    
    echo ""
    echo "================================================================"
    
    local name=$(get_component_prop "$component" "name" "$component")
    local desc=$(get_component_prop "$component" "description" "")
    local github_repo=$(get_component_prop "$component" "github_repo")
    local version_env=$(get_component_prop "$component" "version_env")
    local default_version=$(get_component_prop "$component" "default_version")
    local version_prefix=$(get_component_prop "$component" "version_prefix" "v")
    local filename_version_prefix=$(get_component_prop "$component" "filename_version_prefix" "")
    local filename_pattern=$(get_component_prop "$component" "filename_pattern")
    
    # 获取版本号: 命令行参数 > 环境变量 > .env文件 > Dockerfile > 默认值
    local version=""
    if [ -n "$SPECIFIED_VERSION" ]; then
        version="$SPECIFIED_VERSION"
    elif [ -n "$version_env" ]; then
        version=$(get_env_version "$version_env" "")
        [ -z "$version" ] && version=$(get_dockerfile_arg "$version_env" "")
    fi
    [ -z "$version" ] && version="$default_version"
    
    # 处理版本号前缀
    local tag_version=$(ensure_prefix "$version" "$version_prefix")
    local file_version="$version"
    if [ -n "$filename_version_prefix" ]; then
        file_version="${filename_version_prefix}$(strip_prefix "$version")"
    else
        file_version="$(strip_prefix "$version")"
    fi
    
    echo "📦 $name ($component)"
    [ -n "$desc" ] && echo "   $desc"
    echo "   版本: $tag_version"
    echo "   仓库: $github_repo"
    echo "================================================================"
    
    local output_dir="$THIRD_PARTY_DIR/$component"
    mkdir -p "$output_dir"
    
    # 获取架构列表
    local archs=()
    while IFS= read -r arch; do
        [ -n "$arch" ] && archs+=("$arch")
    done < <(get_component_array "$component" "architectures")
    
    # 如果没有架构配置，默认使用 amd64 和 arm64
    [ ${#archs[@]} -eq 0 ] && archs=("amd64" "arm64")
    
    # 过滤架构
    if [ "$TARGET_ARCH" != "all" ]; then
        local filtered_archs=()
        for arch in "${archs[@]}"; do
            if [ "$arch" = "$TARGET_ARCH" ] || [ -z "$arch" ]; then
                filtered_archs+=("$arch")
            fi
        done
        archs=("${filtered_archs[@]}")
    fi
    
    # 特殊处理: SaltStack 有多个包和格式
    if [ "$component" = "saltstack" ]; then
        download_saltstack "$tag_version" "$file_version" "$output_dir"
    # 特殊处理: code-server (DEB + RPM)
    elif [ "$component" = "code_server" ]; then
        download_code_server "$tag_version" "$file_version" "$output_dir"
    # 特殊处理: singularity (DEB + RPM, 多发行版)
    elif [ "$component" = "singularity" ]; then
        download_singularity "$tag_version" "$file_version" "$output_dir"
    else
        # 通用下载逻辑
        for arch in "${archs[@]}"; do
            local filename=$(echo "$filename_pattern" | sed "s/{VERSION}/$file_version/g" | sed "s/{ARCH}/$arch/g")
            local url="https://github.com/${github_repo}/releases/download/${tag_version}/${filename}"
            
            download_file "$url" "${output_dir}/${filename}" true || true
        done
    fi
    
    generate_version_json "$output_dir" "$component" "$tag_version"
    echo ""
}

# SaltStack 特殊下载 (DEB + RPM)
download_saltstack() {
    local tag_version=$1
    local file_version=$2
    local output_dir=$3
    
    local packages=()
    while IFS= read -r pkg; do
        [ -n "$pkg" ] && packages+=("$pkg")
    done < <(get_component_array "saltstack" "packages")
    
    # DEB packages
    echo ""
    echo "  📦 下载 DEB 包..."
    for arch in amd64 arm64; do
        if [ "$TARGET_ARCH" != "all" ] && [ "$arch" != "$TARGET_ARCH" ]; then
            continue
        fi
        for pkg in "${packages[@]}"; do
            local filename="${pkg}_${file_version}_${arch}.deb"
            local url="https://github.com/saltstack/salt/releases/download/${tag_version}/${filename}"
            download_file "$url" "${output_dir}/${filename}" true || true
        done
    done
    
    # RPM packages
    echo ""
    echo "  📦 下载 RPM 包..."
    for arch in amd64 arm64; do
        if [ "$TARGET_ARCH" != "all" ] && [ "$arch" != "$TARGET_ARCH" ]; then
            continue
        fi
        local rpm_arch="x86_64"
        [ "$arch" = "arm64" ] && rpm_arch="aarch64"
        
        for pkg in "${packages[@]}"; do
            # RPM 包名去掉 -common 后缀
            local rpm_pkg="${pkg/-common/}"
            local filename="${rpm_pkg}-${file_version}-0.${rpm_arch}.rpm"
            local url="https://github.com/saltstack/salt/releases/download/${tag_version}/${filename}"
            download_file "$url" "${output_dir}/${filename}" true || true
        done
    done
}

# code-server 特殊下载 (DEB + RPM)
download_code_server() {
    local tag_version=$1
    local file_version=$2
    local output_dir=$3
    local github_repo="coder/code-server"
    
    # DEB packages
    echo ""
    echo "  📦 下载 DEB 包..."
    for arch in amd64 arm64; do
        if [ "$TARGET_ARCH" != "all" ] && [ "$arch" != "$TARGET_ARCH" ]; then
            continue
        fi
        local filename="code-server_${file_version}_${arch}.deb"
        local url="https://github.com/${github_repo}/releases/download/${tag_version}/${filename}"
        download_file "$url" "${output_dir}/${filename}" true || true
    done
    
    # RPM packages
    # 正确格式: code-server-4.107.0-arm64.rpm (不需要 -1. 和架构映射)
    echo ""
    echo "  📦 下载 RPM 包..."
    for arch in amd64 arm64; do
        if [ "$TARGET_ARCH" != "all" ] && [ "$arch" != "$TARGET_ARCH" ]; then
            continue
        fi
        local filename="code-server-${file_version}-${arch}.rpm"
        local url="https://github.com/${github_repo}/releases/download/${tag_version}/${filename}"
        download_file "$url" "${output_dir}/${filename}" true || true
    done
}

# singularity 特殊下载 (DEB + RPM, 多发行版支持)
download_singularity() {
    local tag_version=$1
    local file_version=$2
    local output_dir=$3
    local github_repo="sylabs/singularity"
    
    # 注意: Singularity CE 4.3.x 官方只提供 amd64/x86_64 预编译包
    # ARM64 用户需要从源码编译: singularity-ce-${file_version}.tar.gz
    # 参考: https://github.com/sylabs/singularity/releases
    
    # DEB packages (Ubuntu) - 仅 amd64
    # 格式: singularity-ce_4.3.6-noble_amd64.deb
    echo ""
    echo "  📦 下载 DEB 包 (Ubuntu)..."
    echo "  ⚠️  注意: Singularity 官方仅提供 amd64 预编译包，ARM64 需从源码编译"
    local ubuntu_codenames=("noble" "jammy")
    for codename in "${ubuntu_codenames[@]}"; do
        # Singularity 仅提供 amd64 预编译包
        if [ "$TARGET_ARCH" = "arm64" ]; then
            echo "  ⏭️  跳过 DEB (arm64): Singularity 官方不提供 ARM64 预编译包"
            continue
        fi
        local filename="singularity-ce_${file_version}-${codename}_amd64.deb"
        local url="https://github.com/${github_repo}/releases/download/${tag_version}/${filename}"
        download_file "$url" "${output_dir}/${filename}" true || true
    done
    
    # RPM packages (RHEL/CentOS/Rocky) - 仅 x86_64
    # 格式: singularity-ce-4.3.6-1.el9.x86_64.rpm
    echo ""
    echo "  📦 下载 RPM 包 (RHEL/CentOS)..."
    local el_versions=("el8" "el9" "el10")
    for el_ver in "${el_versions[@]}"; do
        # Singularity 仅提供 x86_64 预编译包
        if [ "$TARGET_ARCH" = "arm64" ]; then
            echo "  ⏭️  跳过 RPM (aarch64): Singularity 官方不提供 ARM64 预编译包"
            continue
        fi
        local filename="singularity-ce-${file_version}-1.${el_ver}.x86_64.rpm"
        local url="https://github.com/${github_repo}/releases/download/${tag_version}/${filename}"
        download_file "$url" "${output_dir}/${filename}" true || true
    done
    
    # 下载源码包 (适用于所有架构，包括 ARM64)
    echo ""
    echo "  📦 下载源码包 (适用于所有架构)..."
    local source_filename="singularity-ce-${file_version}.tar.gz"
    local source_url="https://github.com/${github_repo}/releases/download/${tag_version}/${source_filename}"
    download_file "$source_url" "${output_dir}/${source_filename}" true || true
}

# =============================================================================
# 主程序
# =============================================================================

# 检查 jq 是否可用
check_jq() {
    if ! command -v jq &> /dev/null; then
        echo "⚠ 警告: jq 未安装，部分功能可能受限"
        echo "  安装方法: brew install jq (macOS) 或 apt install jq (Linux)"
        echo ""
    fi
}

# 获取所有组件列表
get_all_components() {
    if command -v jq &> /dev/null; then
        jq -r '.components | keys[]' "$COMPONENTS_JSON" 2>/dev/null
    else
        # 简单解析
        grep -E '^\s+"[a-z_]+":' "$COMPONENTS_JSON" | sed 's/.*"\([^"]*\)".*/\1/' | grep -v "components"
    fi
}

# 存储要下载的组件
DOWNLOAD_COMPONENTS=()

# 解析命令行参数
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -l|--list)
                list_components
                exit 0
                ;;
            -v|--version)
                SPECIFIED_VERSION="$2"
                shift 2
                ;;
            -a|--arch)
                TARGET_ARCH="$2"
                shift 2
                ;;
            -m|--mirror)
                GITHUB_MIRROR="$2"
                shift 2
                ;;
            --no-mirror)
                GITHUB_MIRROR=""
                shift
                ;;
            -*)
                echo "❌ 未知选项: $1"
                show_help
                exit 1
                ;;
            *)
                DOWNLOAD_COMPONENTS+=("$1")
                shift
                ;;
        esac
    done
}

main() {
    check_jq
    
    # 检查配置文件
    if [ ! -f "$COMPONENTS_JSON" ]; then
        echo "❌ 配置文件不存在: $COMPONENTS_JSON"
        exit 1
    fi
    
    # 解析参数
    parse_args "$@"
    
    # 如果没有指定组件，下载所有组件
    if [ ${#DOWNLOAD_COMPONENTS[@]} -eq 0 ]; then
        while IFS= read -r comp; do
            [ -n "$comp" ] && DOWNLOAD_COMPONENTS+=("$comp")
        done < <(get_all_components)
    fi
    
    echo ""
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║          Third-Party Dependencies Downloader                   ║"
    echo "╚════════════════════════════════════════════════════════════════╝"
    echo ""
    echo "GitHub 镜像: ${GITHUB_MIRROR:-<已禁用>}"
    echo "目标架构:    ${TARGET_ARCH}"
    echo "输出目录:    ${THIRD_PARTY_DIR}"
    echo "组件数量:    ${#DOWNLOAD_COMPONENTS[@]}"
    echo ""
    
    mkdir -p "$THIRD_PARTY_DIR"
    
    local success=0
    local failed=0
    
    for component in "${DOWNLOAD_COMPONENTS[@]}"; do
        # 检查组件是否存在
        local comp_name=$(get_component_prop "$component" "name")
        if [ -z "$comp_name" ] || [ "$comp_name" = "null" ]; then
            echo "⚠ 跳过未知组件: $component"
            ((failed++))
            continue
        fi
        
        if download_component "$component"; then
            ((success++))
        else
            ((failed++))
        fi
    done
    
    echo ""
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║                         下载完成                               ║"
    echo "╚════════════════════════════════════════════════════════════════╝"
    echo ""
    echo "成功: $success / 总计: $((success + failed))"
    echo ""
    echo "文件位置: $THIRD_PARTY_DIR"
    echo ""
    ls -la "$THIRD_PARTY_DIR"
}

main "$@"
