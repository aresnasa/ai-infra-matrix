#!/usr/bin/env bash
set -e

# ==============================================================================
# AI Infrastructure Matrix - Refactored Build Script
# ==============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="$SCRIPT_DIR/.env"
ENV_EXAMPLE="$SCRIPT_DIR/.env.example"
SRC_DIR="$SCRIPT_DIR/src"

# ==============================================================================
# Early Help Detection (避免初始化逻辑)
# --help/-h/help 命令应仅打印帮助，不触发 .env 生成或其他初始化
# ==============================================================================
_SHOW_HELP_ONLY=false
_SKIP_PORT_CHECK=false
for _arg in "$@"; do
    case "$_arg" in
        --help|-h|help)
            _SHOW_HELP_ONLY=true
            break
            ;;
        # 清理和状态类命令不需要端口检查
        clean-all|clean-images|clean-volumes|cache-status|build-history|clear-cache|status|logs|db-check)
            _SKIP_PORT_CHECK=true
            break
            ;;
    esac
done

# ==============================================================================
# Build Cache Configuration
# ==============================================================================
BUILD_CACHE_DIR="$SCRIPT_DIR/.build-cache"
BUILD_ID_FILE="$BUILD_CACHE_DIR/build-id.txt"
BUILD_HISTORY_FILE="$BUILD_CACHE_DIR/build-history.log"
SKIP_CACHE_CHECK=false
FORCE_REBUILD=false

# Parallel Build Configuration
PARALLEL_JOBS=${PARALLEL_JOBS:-4}  # Default 4 parallel jobs
ENABLE_PARALLEL=false              # Disabled by default

# SSL Configuration
ENABLE_SSL=${ENABLE_SSL:-true}     # Enabled by default
SSL_DOMAIN=${SSL_DOMAIN:-}         # SSL domain, auto-detect from EXTERNAL_HOST if empty
SSL_CERT_DIR="$SCRIPT_DIR/ssl-certs"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_cache() { echo -e "${CYAN}[CACHE]${NC} $1"; }
log_parallel() { echo -e "${BLUE}[PARALLEL]${NC} $1"; }
log_step() { echo -e "${CYAN}[STEP]${NC} $1"; }
log_debug() { [[ "${DEBUG:-false}" == "true" ]] && echo -e "${BLUE}[DEBUG]${NC} $1"; return 0; }

# ==============================================================================
# Network Diagnostics Functions - 网络诊断函数
# ==============================================================================

# Check basic network connectivity
# Returns: 0 if connected, 1 if disconnected
_check_network_connectivity() {
    local test_hosts=("8.8.8.8" "1.1.1.1" "114.114.114.114")
    local timeout=3
    
    for host in "${test_hosts[@]}"; do
        if ping -c 1 -W $timeout "$host" >/dev/null 2>&1; then
            return 0
        fi
    done
    
    return 1
}

# Get network interface information
_get_network_info() {
    echo "Network Information:"
    
    if command -v ifconfig >/dev/null 2>&1; then
        ifconfig | grep -E "^[a-z]|inet " || true
    elif command -v ip >/dev/null 2>&1; then
        ip addr show | grep -E "^[0-9]+:|inet " || true
    else
        echo "  (No network tools available)"
    fi
}

# Diagnose network issues
diagnose_network() {
    log_error "Network connectivity issue detected!"
    echo ""
    log_step "Running network diagnostics..."
    echo ""
    
    # Check DNS resolution
    echo "🔍 DNS Resolution Test:"
    if ping -c 1 -W 2 docker.io >/dev/null 2>&1; then
        echo "  ✓ docker.io is reachable"
    else
        echo "  ✗ docker.io is NOT reachable"
        echo "    - Check your internet connection"
        echo "    - Check DNS configuration: cat /etc/resolv.conf"
        echo "    - Try using different DNS: 8.8.8.8, 1.1.1.1, 114.114.114.114"
    fi
    echo ""
    
    # Check Docker daemon
    echo "🐳 Docker Status:"
    if docker info >/dev/null 2>&1; then
        local docker_version=$(docker version --format='{{.Server.Version}}' 2>/dev/null || echo "unknown")
        echo "  ✓ Docker daemon is running (version: $docker_version)"
    else
        echo "  ✗ Docker daemon is NOT running"
        echo "    - Start Docker: docker daemon"
        echo "    - Or start Docker Desktop on macOS/Windows"
    fi
    echo ""
    
    # Check buildx builder
    echo "🏗️ BuildKit Builder:"
    if docker buildx ls >/dev/null 2>&1; then
        echo "  ✓ docker buildx is available"
        docker buildx ls | sed 's/^/    /'
    else
        echo "  ✗ docker buildx is NOT available"
    fi
    echo ""
    
    # Show network interfaces
    echo "🌐 Network Interfaces:"
    _get_network_info | sed 's/^/    /'
    echo ""
    
    # Check DNS
    echo "🔎 DNS Servers:"
    if [[ -f "/etc/resolv.conf" ]]; then
        grep "^nameserver" /etc/resolv.conf | sed 's/^/    /' || echo "    (No nameservers configured)"
    elif [[ -f "/etc/wsl.conf" ]]; then
        echo "    (WSL2 system - DNS inherited from Windows)"
    else
        echo "    (Unable to check DNS configuration)"
    fi
    echo ""
    
    log_info "Recommended solutions:"
    echo "  1. Check internet connection:  ping 8.8.8.8"
    echo "  2. Check DNS:                  nslookup docker.io"
    echo "  3. Check Docker network:       docker network ls"
    echo "  4. Restart Docker:             docker restart"
    echo "  5. Clear Docker build cache:   docker builder prune --all"
    echo ""
}

# Test if a buildx builder can access Docker Hub registry
# This is critical for docker-container drivers which may have network isolation issues
# Returns: 0 if network works, 1 if network fails
# Usage: _test_buildx_network "multiarch-builder" "linux/arm64"
_test_buildx_network() {
    local builder_name="$1"
    local platform="${2:-linux/amd64}"
    local timeout_seconds=20
    
    # Create a temporary build context directory with minimal Dockerfile
    local test_dir=$(mktemp -d)
    cat > "$test_dir/Dockerfile" << 'TESTEOF'
FROM alpine:latest
RUN echo "Network test passed"
TESTEOF
    
    # Try to build with the specified builder
    # Use --no-cache to ensure we actually test network connectivity
    local result=0
    if timeout "$timeout_seconds" docker buildx build \
        --builder "$builder_name" \
        --platform "$platform" \
        --no-cache \
        --progress=quiet \
        "$test_dir" >/dev/null 2>&1; then
        result=0
    else
        result=1
    fi
    
    rm -rf "$test_dir"
    return $result
}

# Select the best available buildx builder for the given platform
# Intelligent builder selection for buildx
# - For native architecture: prefer docker driver (desktop-linux) for reliable --load
# - For cross-platform: use docker-container driver (multiarch-builder)
# Returns: builder name via echo
# Usage: builder=$(_select_best_builder "linux/arm64")
_select_best_builder() {
    local platform="${1:-linux/amd64}"
    local arch_name="${platform##*/}"
    local native_platform=$(_detect_docker_platform)
    local native_arch="${native_platform##*/}"
    
    # Builder names
    local docker_driver_builder="desktop-linux"
    local container_driver_builder="multiarch-builder"
    
    # =========================================================================
    # KEY FIX: For native architecture, prefer docker driver (desktop-linux)
    # 
    # The docker driver's --load is more reliable, especially in parallel builds.
    # The docker-container driver (multiarch-builder) may have issues with
    # concurrent --load operations, causing images to not be properly loaded
    # into the local Docker daemon.
    # =========================================================================
    if [[ "$arch_name" == "$native_arch" ]]; then
        # Native architecture: use docker driver for reliable --load
        if docker buildx inspect "$docker_driver_builder" >/dev/null 2>&1; then
            echo "  [$arch_name] Using $docker_driver_builder (native arch, docker driver)" >&2
            echo "$docker_driver_builder"
            return 0
        fi
        # Fallback to default if desktop-linux doesn't exist
        if docker buildx inspect "default" >/dev/null 2>&1; then
            echo "  [$arch_name] Using default builder (native arch)" >&2
            echo "default"
            return 0
        fi
    fi
    
    # Cross-platform: need docker-container driver for QEMU/Rosetta emulation
    # Check if multiarch-builder exists
    if ! docker buildx inspect "$container_driver_builder" >/dev/null 2>&1; then
        echo "  [$arch_name] ⚠️  $container_driver_builder not found, using $docker_driver_builder" >&2
        echo "$docker_driver_builder"
        return 0
    fi
    
    # Test network connectivity with the preferred builder
    # This is critical because docker-container driver may have proxy/network issues
    # Note: Use >&2 to send logs to stderr so they don't pollute the return value
    echo "  [$arch_name] Testing buildx network connectivity..." >&2
    
    if _test_buildx_network "$container_driver_builder" "$platform"; then
        echo "  [$arch_name] ✓ Using $container_driver_builder (cross-platform, network test passed)" >&2
        echo "$container_driver_builder"
        return 0
    else
        echo "  [$arch_name] ⚠️  $container_driver_builder network test failed" >&2
        
        # Test fallback builder
        if docker buildx inspect "$docker_driver_builder" >/dev/null 2>&1; then
            if _test_buildx_network "$docker_driver_builder" "$platform"; then
                echo "  [$arch_name] ✓ Falling back to $docker_driver_builder (network works)" >&2
                echo "$docker_driver_builder"
                return 0
            fi
        fi
        
        # Both failed, run full network diagnostics
        echo "  [$arch_name] ✗ All builders failed network test" >&2
        diagnose_network >&2
        
        # Return container driver builder anyway, let the actual build fail with proper error
        echo "$container_driver_builder"
        return 1
    fi
}

# ==============================================================================
# 通用工具函数
# ==============================================================================

# 验证 registry 路径是否完整 (Harbor 需要 project 名称)
# 返回: 0 = 验证通过或用户确认继续, 1 = 用户取消
# 用法: validate_registry_path "harbor.example.com/ai-infra" "v0.3.8"
validate_registry_path() {
    local registry="$1"
    local tag="${2:-}"
    
    # 如果 registry 为空，直接返回成功（不需要验证）
    [[ -z "$registry" ]] && return 0
    
    # 检查是否包含项目路径（应至少有一个 /）
    if [[ ! "$registry" =~ / ]]; then
        log_warn "=========================================="
        log_warn "⚠️  Registry path may be incomplete!"
        log_warn "=========================================="
        log_warn "Provided: $registry"
        log_warn ""
        log_warn "Harbor registries require a project name in the path:"
        log_warn "  ✓ $registry/ai-infra    (recommended)"
        log_warn "  ✓ $registry/<project>   (your project name)"
        log_warn ""
        if [[ -n "$tag" ]]; then
            log_warn "Example usage:"
            log_warn "  $0 [command] $registry/ai-infra $tag"
        fi
        log_warn ""
        
        # 非交互模式下返回失败
        if [[ ! -t 0 ]]; then
            log_warn "Non-interactive mode, aborting."
            return 1
        fi
        
        read -p "Continue anyway? [y/N] " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Cancelled. Please use correct registry path."
            return 1
        fi
        log_warn "Continuing with incomplete registry path..."
    fi
    
    return 0
}

# ==============================================================================
# 1. Configuration & Environment
# ==============================================================================

# 检测外部主机地址 (智能检测真实网络出口IP)
# 支持 Linux (ip addr) 和 macOS (ifconfig)
# 自动过滤 loopback、docker、k8s、虚拟网桥等非物理接口
detect_external_host() {
    local detected_ip=""
    
    # 方法1：使用默认路由检测出口IP (最准确的方法)
    # Linux: ip route get 1.1.1.1
    # 这个方法直接获取访问外网时使用的源IP
    if command -v ip &> /dev/null; then
        detected_ip=$(ip route get 1.1.1.1 2>/dev/null | grep -oP 'src \K[0-9.]+' | head -n1)
        # 备用方法：ip route get 1
        if [[ -z "$detected_ip" ]]; then
            detected_ip=$(ip route get 1 2>/dev/null | awk '{for(i=1;i<=NF;i++) if($i=="src") print $(i+1)}' | head -n1)
        fi
    fi
    
    # 方法2：使用 ip addr 枚举接口 (Linux 备用)
    if [[ -z "$detected_ip" ]] && command -v ip &> /dev/null; then
        detected_ip=$(ip -4 addr show scope global 2>/dev/null | \
            grep -v -E "(docker|veth|br-|cni|flannel|calico|weave|kube|virbr|vboxnet|vmnet|tun|tap|lo:)" | \
            grep "inet " | \
            awk '{print $2}' | cut -d'/' -f1 | \
            grep -v -E "^(127\.|10\.96\.|10\.244\.|172\.17\.|172\.18\.|172\.19\.|192\.168\.49\.)" | \
            head -n1)
    fi
    
    # 方法3：使用 ifconfig (macOS/BSD)
    if [[ -z "$detected_ip" ]] && command -v ifconfig &> /dev/null; then
        # macOS: 优先检测 en0 (通常是主网卡)
        if [[ "$OSTYPE" == "darwin"* ]]; then
            detected_ip=$(ifconfig en0 2>/dev/null | grep "inet " | awk '{print $2}')
            # 如果 en0 没有IP，尝试其他接口
            if [[ -z "$detected_ip" ]]; then
                detected_ip=$(ifconfig | awk '
                    /^[a-z0-9]+:/ { iface=$1; sub(/:/, "", iface) }
                    /inet / && !/127\.0\.0\.1/ {
                        # 排除虚拟接口
                        if (iface !~ /^(lo|docker|veth|br|vmnet|vboxnet|tun|tap|virbr|utun|bridge|awdl|llw)/)
                            print $2
                    }' | head -n1)
            fi
        else
            # 其他 BSD 系统
            detected_ip=$(ifconfig | awk '
                /^[a-z0-9]+:/ { iface=$1; sub(/:/, "", iface) }
                /inet / && !/127\.0\.0\.1/ {
                    if (iface !~ /^(lo|docker|veth|br|vmnet|vboxnet|tun|tap|virbr)/)
                        print $2
                }' | \
                grep -v -E "^(10\.96\.|10\.244\.|172\.17\.|172\.18\.|172\.19\.)" | \
                head -n1)
        fi
    fi
    
    # 方法4：从现有 .env 读取（如果已配置且不是自引用）
    if [[ -z "$detected_ip" ]] && [[ -f "$ENV_FILE" ]]; then
        local env_ip=$(grep "^EXTERNAL_HOST=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2)
        # 忽略自引用、空值和 localhost
        if [[ -n "$env_ip" ]] && [[ ! "$env_ip" =~ \$\{ ]] && [[ "$env_ip" != "localhost" ]]; then
            detected_ip="$env_ip"
        fi
    fi
    
    # 方法5：hostname -I (Linux 最后备用)
    if [[ -z "$detected_ip" ]] && command -v hostname &> /dev/null; then
        detected_ip=$(hostname -I 2>/dev/null | awk '{print $1}')
    fi
    
    # 返回检测到的IP或默认值
    echo "${detected_ip:-localhost}"
}

# 检测是否为私有 IP 地址
# 返回: 0 如果是私有 IP，1 如果是公网 IP 或域名
is_private_ip() {
    local addr="$1"
    
    # 空值不是私有 IP
    [[ -z "$addr" ]] && return 1
    
    # 检查是否为有效 IP 格式
    if ! [[ "$addr" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        return 1  # 不是 IP 格式（可能是域名）
    fi
    
    # 检查私有 IP 范围
    # 10.0.0.0/8
    [[ "$addr" =~ ^10\. ]] && return 0
    # 172.16.0.0/12 (172.16.x.x - 172.31.x.x)
    [[ "$addr" =~ ^172\.(1[6-9]|2[0-9]|3[0-1])\. ]] && return 0
    # 192.168.0.0/16
    [[ "$addr" =~ ^192\.168\. ]] && return 0
    # 127.0.0.0/8 (localhost)
    [[ "$addr" =~ ^127\. ]] && return 0
    # 169.254.0.0/16 (link-local)
    [[ "$addr" =~ ^169\.254\. ]] && return 0
    
    return 1
}

# 检测是否为有效域名
# 返回: 0 如果是域名，1 如果是 IP 或其他
is_valid_domain() {
    local addr="$1"
    
    [[ -z "$addr" ]] && return 1
    
    # 如果是 IP 格式，不是域名
    [[ "$addr" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]] && return 1
    
    # 如果是 localhost，不是有效外部域名
    [[ "$addr" == "localhost" ]] && return 1
    
    # 基本域名格式检查 (至少包含一个点)
    [[ "$addr" =~ \. ]] && return 0
    
    return 1
}

# 检测 SSL 证书域名是否与 EXTERNAL_HOST 匹配
# 返回: 0 如果匹配，1 如果不匹配或证书不存在
check_ssl_cert_domain_match() {
    local expected_domain="$1"
    local cert_file="${SSL_OUTPUT_DIR:-./src/nginx/ssl}/server.crt"
    
    # 证书不存在
    [[ ! -f "$cert_file" ]] && return 1
    
    # 获取证书的 CN 和 SAN
    local cert_cn=$(openssl x509 -in "$cert_file" -noout -subject 2>/dev/null | sed -n 's/.*CN *= *\([^,]*\).*/\1/p')
    local cert_san=$(openssl x509 -in "$cert_file" -noout -ext subjectAltName 2>/dev/null | grep -oE 'DNS:[^,]+' | sed 's/DNS://g' | tr '\n' ' ')
    
    # 检查 CN 是否匹配
    [[ "$cert_cn" == "$expected_domain" ]] && return 0
    [[ "$cert_cn" == "www.$expected_domain" ]] && return 0
    [[ "$cert_cn" == "${expected_domain#www.}" ]] && return 0
    
    # 检查 SAN 是否包含该域名
    [[ "$cert_san" == *"$expected_domain"* ]] && return 0
    [[ "$cert_san" == *"www.$expected_domain"* ]] && return 0
    
    return 1
}

# 显示 SSL/域名配置建议
show_ssl_domain_recommendations() {
    local external_host="${1:-$EXTERNAL_HOST}"
    local ssl_domain="${2:-$SSL_DOMAIN}"
    
    echo ""
    log_info "════════════════════════════════════════════════════════════════"
    log_info "🔒 SSL/域名配置检测"
    log_info "════════════════════════════════════════════════════════════════"
    
    # 检查 EXTERNAL_HOST 类型
    if is_valid_domain "$external_host"; then
        log_info "✅ EXTERNAL_HOST='$external_host' 是有效域名"
        
        # 检查证书匹配
        if check_ssl_cert_domain_match "$external_host"; then
            log_info "✅ SSL 证书域名与 EXTERNAL_HOST 匹配"
        else
            log_warn "⚠️  SSL 证书域名可能与 EXTERNAL_HOST 不匹配"
            log_info ""
            log_info "建议操作："
            log_info "  1. 重新生成 Let's Encrypt 证书（包含所有域名）:"
            log_info "     certbot certonly --dns-cloudflare \\"
            log_info "       --dns-cloudflare-credentials ~/.secrets/cloudflare.ini \\"
            log_info "       -d $external_host \\"
            log_info "       -d www.$external_host \\"
            log_info "       --cert-name $external_host --force-renewal"
            log_info ""
            log_info "  2. 或使用 build.sh 生成自签名证书:"
            log_info "     ./build.sh ssl-setup $external_host --force"
        fi
    elif is_private_ip "$external_host"; then
        log_warn "⚠️  EXTERNAL_HOST='$external_host' 是私有 IP 地址"
        log_info ""
        log_info "在公有云环境中，建议使用域名而非私有 IP:"
        log_info "  1. 配置域名指向服务器公网 IP"
        log_info "  2. 修改 .env 中的 EXTERNAL_HOST 为域名"
        log_info "  3. 使用 Let's Encrypt 申请正式证书"
        log_info ""
        log_info "示例配置:"
        log_info "  EXTERNAL_HOST=your-domain.com"
        log_info "  SSL_DOMAIN=your-domain.com"
        log_info "  LETSENCRYPT_EMAIL=admin@your-domain.com"
    else
        log_info "ℹ️  EXTERNAL_HOST='$external_host'"
        if [[ "$external_host" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
            log_info "   这是公网 IP，可使用自签名证书或 IP-based 证书"
        fi
    fi
    
    # 显示当前证书信息
    local cert_file="${SSL_OUTPUT_DIR:-./src/nginx/ssl}/server.crt"
    if [[ -f "$cert_file" ]]; then
        echo ""
        log_info "📜 当前 SSL 证书信息:"
        local cert_cn=$(openssl x509 -in "$cert_file" -noout -subject 2>/dev/null | sed -n 's/.*CN *= *\([^,]*\).*/\1/p')
        local cert_expire=$(openssl x509 -in "$cert_file" -noout -enddate 2>/dev/null | cut -d= -f2)
        local cert_san=$(openssl x509 -in "$cert_file" -noout -ext subjectAltName 2>/dev/null | grep -oE 'DNS:[^,]+' | sed 's/DNS://g' | tr '\n' ', ' | sed 's/,$//')
        log_info "   证书主体 (CN): $cert_cn"
        [[ -n "$cert_san" ]] && log_info "   备用名称 (SAN): $cert_san"
        log_info "   过期时间: $cert_expire"
    else
        log_warn "⚠️  未找到 SSL 证书文件"
        log_info "   运行 ./build.sh ssl-setup 生成证书"
    fi
    
    echo ""
}

# 检测 cgroup 版本 (v1 或 v2)
# cgroupv2 使用统一的层次结构，通常挂载在 /sys/fs/cgroup
# cgroupv1 使用多层次结构，有多个子系统目录
detect_cgroup_version() {
    local cgroup_version="v1"  # 默认 v1
    
    # 方法0：macOS/Docker Desktop - 通过 docker info 检测
    if [[ "$OSTYPE" == "darwin"* ]] && command -v docker &> /dev/null; then
        local docker_cgroup_ver
        docker_cgroup_ver=$(docker info 2>/dev/null | grep -i "Cgroup Version" | awk '{print $NF}')
        if [[ "$docker_cgroup_ver" == "2" ]]; then
            cgroup_version="v2"
            echo "$cgroup_version"
            return
        elif [[ "$docker_cgroup_ver" == "1" ]]; then
            cgroup_version="v1"
            echo "$cgroup_version"
            return
        fi
    fi
    
    # 方法1：检查 /sys/fs/cgroup/cgroup.controllers (cgroupv2 特有) - Linux
    if [[ -f "/sys/fs/cgroup/cgroup.controllers" ]]; then
        cgroup_version="v2"
    # 方法2：检查 /sys/fs/cgroup 的挂载类型 - Linux
    elif [[ -d "/sys/fs/cgroup" ]] && command -v stat &> /dev/null; then
        local cgroup_fstype
        cgroup_fstype=$(stat -f -c %T /sys/fs/cgroup 2>/dev/null || stat -f %T /sys/fs/cgroup 2>/dev/null)
        if [[ "$cgroup_fstype" == "cgroup2fs" ]] || [[ "$cgroup_fstype" == "cgroup2" ]]; then
            cgroup_version="v2"
        fi
    # 方法3：通过 mount 命令检查 - Linux
    elif [[ -d "/sys/fs/cgroup" ]] && command -v mount &> /dev/null; then
        if mount | grep -q "cgroup2 on /sys/fs/cgroup"; then
            cgroup_version="v2"
        fi
    # 方法4：通过 docker info 检测（非 macOS 但有 docker）
    elif command -v docker &> /dev/null; then
        local docker_cgroup_ver
        docker_cgroup_ver=$(docker info 2>/dev/null | grep -i "Cgroup Version" | awk '{print $NF}')
        if [[ "$docker_cgroup_ver" == "2" ]]; then
            cgroup_version="v2"
        fi
    fi
    
    echo "$cgroup_version"
}

# 根据 cgroup 版本生成适当的挂载配置
# cgroupv1: 需要挂载 /sys/fs/cgroup:/sys/fs/cgroup:ro 或 :rw
# cgroupv2: 可能需要 /sys/fs/cgroup:/sys/fs/cgroup:rw,rslave 或仅使用 cgroup: host
get_cgroup_mount() {
    local cgroup_version="${1:-$(detect_cgroup_version)}"
    
    case "$cgroup_version" in
        v2)
            # cgroupv2 挂载配置
            # 使用 rw,rslave 确保 systemd 可以正常工作
            echo "/sys/fs/cgroup:/sys/fs/cgroup:rw"
            ;;
        v1|*)
            # cgroupv1 挂载配置
            echo "/sys/fs/cgroup:/sys/fs/cgroup:rw"
            ;;
    esac
}

# 更新 .env 文件中的变量
# 用法: update_env_variable "VAR_NAME" "var_value"
update_env_variable() {
    local var_name="$1"
    local var_value="$2"
    local env_file="$ENV_FILE"
    
    # 防御性检查：变量名不能为空
    if [[ -z "$var_name" ]]; then
        log_warn "update_env_variable: empty var_name, skipping"
        return 1
    fi
    
    if [[ ! -f "$env_file" ]]; then
        return 1
    fi
    
    # 检查变量是否已存在
    if grep -q "^${var_name}=" "$env_file"; then
        # 更新现有变量 (macOS 兼容)
        if [[ "$OSTYPE" == "darwin"* ]]; then
            sed -i '' "s|^${var_name}=.*|${var_name}=${var_value}|" "$env_file"
        else
            sed -i "s|^${var_name}=.*|${var_name}=${var_value}|" "$env_file"
        fi
    else
        # 添加新变量
        echo "${var_name}=${var_value}" >> "$env_file"
    fi
}

# 同步 .env 与 .env.example
# 功能：
#   1. 如果 .env.prod 存在，优先复制为 .env（生产环境配置优先）
#   2. 添加 .env.example 中新增的变量
#   3. 只同步版本类变量 (VERSION, TAG, VER, RELEASE)，保留用户自定义配置
# 用法: sync_env_with_example
sync_env_with_example() {
    local env_file="$ENV_FILE"
    local example_file="$ENV_EXAMPLE"
    local prod_file="$SCRIPT_DIR/.env.prod"
    
    if [[ ! -f "$example_file" ]]; then
        log_error ".env.example not found: $example_file"
        return 1
    fi
    
    # 优先使用 .env.prod（生产环境配置）
    if [[ -f "$prod_file" ]] && [[ ! -f "$env_file" ]]; then
        log_info "Found .env.prod, using it as .env..."
        cp "$prod_file" "$env_file"
        log_info "✓ Created .env from .env.prod"
    elif [[ ! -f "$env_file" ]]; then
        log_info "Creating .env from .env.example..."
        cp "$example_file" "$env_file"
        return 0
    fi
    
    local missing_vars=()
    local updated_vars=()
    
    # 读取 .env.example 中的所有变量，同步到 .env
    while IFS= read -r line || [[ -n "$line" ]]; do
        # 跳过注释和空行
        if [[ "$line" =~ ^[[:space:]]*# ]] || [[ -z "$line" ]] || [[ "$line" =~ ^[[:space:]]*$ ]]; then
            continue
        fi
        
        # 提取变量名和值
        if [[ "$line" =~ ^([A-Za-z_][A-Za-z0-9_]*)=(.*)$ ]]; then
            local var_name="${BASH_REMATCH[1]}"
            local example_value="${BASH_REMATCH[2]}"
            
            # 防御性检查：确保变量名不为空
            if [[ -z "$var_name" ]]; then
                continue
            fi
            
            # 检查 .env 中是否存在该变量
            if ! grep -q "^${var_name}=" "$env_file" 2>/dev/null; then
                # 变量不存在，添加到文件末尾
                echo "${var_name}=${example_value}" >> "$env_file"
                missing_vars+=("$var_name")
            else
                # 变量存在，只同步版本类变量 (VERSION, TAG, VER, RELEASE)
                if [[ "$var_name" =~ (VERSION|_TAG$|_VER$|_RELEASE$) ]]; then
                    local current_value
                    current_value=$(grep "^${var_name}=" "$env_file" | head -1 | cut -d'=' -f2-)
                    
                    # 如果值不同，用 example 的值更新
                    if [[ "$current_value" != "$example_value" ]]; then
                        update_env_variable "$var_name" "$example_value"
                        updated_vars+=("$var_name: $current_value → $example_value")
                    fi
                fi
            fi
        fi
    done < "$example_file"
    
    # 显示同步结果
    if [[ ${#missing_vars[@]} -gt 0 ]]; then
        log_info "Added ${#missing_vars[@]} new variables from .env.example:"
        for var in "${missing_vars[@]}"; do
            log_info "  + $var"
        done
    fi
    
    if [[ ${#updated_vars[@]} -gt 0 ]]; then
        log_info "Updated ${#updated_vars[@]} version variables from .env.example:"
        for var in "${updated_vars[@]}"; do
            log_info "  ↻ $var"
        done
    fi
    
    if [[ ${#missing_vars[@]} -eq 0 ]] && [[ ${#updated_vars[@]} -eq 0 ]]; then
        log_info "✓ .env is in sync with .env.example"
    else
        log_info "✓ Synced ${#missing_vars[@]} new + ${#updated_vars[@]} version variables"
    fi
    
    # 检测配置差异并警告用户
    check_env_config_drift
    
    return 0
}

# 检测 .env 与 .env.example 之间的配置差异
# 功能：
#   1. 检测非版本类变量的值差异
#   2. 特别关注关键配置项（如 EXTERNAL_SCHEME 与 ENABLE_TLS 的一致性）
#   3. 警告用户可能的配置问题
# 用法: check_env_config_drift
check_env_config_drift() {
    local env_file="$ENV_FILE"
    local example_file="$ENV_EXAMPLE"
    
    if [[ ! -f "$example_file" ]] || [[ ! -f "$env_file" ]]; then
        return 0
    fi
    
    local drift_vars=()
    local critical_drifts=()
    
    # 定义需要检测差异的关键配置变量（非版本类，非用户自定义类）
    # 这些变量的默认值通常应该与 .env.example 保持一致
    local -a check_vars=(
        "EXTERNAL_SCHEME"
        "ENABLE_TLS"
        "ENABLE_OAUTH"
        "ENABLE_LDAP"
        "SSO_ENABLED"
        "JWT_SECRET_KEY"
        "HTTP_PORT"
        "HTTPS_PORT"
    )
    
    # 读取 .env.example 中的变量
    while IFS= read -r line || [[ -n "$line" ]]; do
        # 跳过注释和空行
        if [[ "$line" =~ ^[[:space:]]*# ]] || [[ -z "$line" ]] || [[ "$line" =~ ^[[:space:]]*$ ]]; then
            continue
        fi
        
        # 提取变量名和值
        if [[ "$line" =~ ^([A-Za-z_][A-Za-z0-9_]*)=(.*)$ ]]; then
            local var_name="${BASH_REMATCH[1]}"
            local example_value="${BASH_REMATCH[2]}"
            
            # 跳过版本类变量（已在 sync_env_with_example 中处理）
            if [[ "$var_name" =~ (VERSION|_TAG$|_VER$|_RELEASE$) ]]; then
                continue
            fi
            
            # 跳过用户自定义类变量（如密码、域名等）
            if [[ "$var_name" =~ (PASSWORD|SECRET|_HOST$|_DOMAIN$|_USER$|_NAME$|_PATH$|_DIR$|_EMAIL$) ]]; then
                continue
            fi
            
            # 检查关键配置变量
            local is_critical=false
            for check_var in "${check_vars[@]}"; do
                if [[ "$var_name" == "$check_var" ]]; then
                    is_critical=true
                    break
                fi
            done
            
            if [[ "$is_critical" == "true" ]]; then
                # 获取 .env 中的当前值
                local current_value
                current_value=$(grep "^${var_name}=" "$env_file" 2>/dev/null | head -1 | cut -d'=' -f2-)
                
                if [[ -n "$current_value" ]] && [[ "$current_value" != "$example_value" ]]; then
                    drift_vars+=("$var_name: '$current_value' (当前) vs '$example_value' (推荐)")
                fi
            fi
        fi
    done < "$example_file"
    
    # 特殊检查：ENABLE_TLS=true 但 EXTERNAL_SCHEME=http 的不一致
    local enable_tls
    local external_scheme
    local external_port
    enable_tls=$(grep "^ENABLE_TLS=" "$env_file" 2>/dev/null | head -1 | cut -d'=' -f2-)
    external_scheme=$(grep "^EXTERNAL_SCHEME=" "$env_file" 2>/dev/null | head -1 | cut -d'=' -f2-)
    external_port=$(grep "^EXTERNAL_PORT=" "$env_file" 2>/dev/null | head -1 | cut -d'=' -f2-)
    
    # 如果使用 443 端口，说明是生产环境部署（可能在 CDN/反向代理后面），跳过此警告
    if [[ "$enable_tls" == "true" ]] && [[ "$external_scheme" == "http" ]] && [[ "$external_port" != "443" ]]; then
        critical_drifts+=("⚠️  配置不一致: ENABLE_TLS=true 但 EXTERNAL_SCHEME=http")
        critical_drifts+=("   建议: 运行 './build.sh enable-ssl' 或手动设置 EXTERNAL_SCHEME=https")
    fi
    
    # 如果使用 443 端口，说明是生产环境部署（可能在 CDN/反向代理后面），跳过此警告
    if [[ "$enable_tls" == "false" ]] && [[ "$external_scheme" == "https" ]] && [[ "$external_port" != "443" ]]; then
        critical_drifts+=("⚠️  配置不一致: ENABLE_TLS=false 但 EXTERNAL_SCHEME=https")
        critical_drifts+=("   建议: 设置 ENABLE_TLS=true 或 EXTERNAL_SCHEME=http")
    fi
    
    # 显示差异警告
    if [[ ${#drift_vars[@]} -gt 0 ]]; then
        log_warn "检测到 ${#drift_vars[@]} 个配置与 .env.example 默认值不同:"
        for drift in "${drift_vars[@]}"; do
            log_warn "  ≠ $drift"
        done
        log_info "提示: 如果这是有意修改，可以忽略此警告"
    fi
    
    # 显示严重配置问题
    if [[ ${#critical_drifts[@]} -gt 0 ]]; then
        echo ""
        log_error "发现关键配置问题:"
        for critical in "${critical_drifts[@]}"; do
            echo -e "  ${critical}"
        done
        echo ""
    fi
    
    # 检查端口冲突
    check_port_conflicts
    
    return 0
}

# 检查端口配置冲突
# 确保只有 nginx 使用 80/443 端口，其他服务不能占用这些端口
check_port_conflicts() {
    # 跳过清理和状态类命令的端口检查
    if [[ "$_SKIP_PORT_CHECK" == "true" ]]; then
        return 0
    fi
    
    local env_file="$ENV_FILE"
    
    if [[ ! -f "$env_file" ]]; then
        return 0
    fi
    
    local conflicts=()
    local reserved_ports=("80" "443")
    
    # 定义不应该使用 80/443 的服务端口变量
    local -a service_ports=(
        "JUPYTERHUB_EXTERNAL_PORT:JupyterHub"
        "GITEA_EXTERNAL_PORT:Gitea"
        "APPHUB_PORT:AppHub"
        "BACKEND_DEBUG_PORT:Backend Debug"
        "DEBUG_PORT:Debug"
        "PROMETHEUS_EXTERNAL_PORT:Prometheus"
        "GRAFANA_EXTERNAL_PORT:Grafana"
        "ALERTMANAGER_EXTERNAL_PORT:Alertmanager"
    )
    
    for service_port in "${service_ports[@]}"; do
        local var_name="${service_port%%:*}"
        local service_name="${service_port#*:}"
        
        local port_value
        port_value=$(grep "^${var_name}=" "$env_file" 2>/dev/null | head -1 | cut -d'=' -f2-)
        
        if [[ -n "$port_value" ]]; then
            for reserved in "${reserved_ports[@]}"; do
                if [[ "$port_value" == "$reserved" ]]; then
                    conflicts+=("$service_name ($var_name=$port_value)")
                fi
            done
        fi
    done
    
    if [[ ${#conflicts[@]} -gt 0 ]]; then
        echo ""
        log_error "🚨 端口冲突检测: 以下服务不应使用 80/443 端口（这些端口应保留给 Nginx）:"
        for conflict in "${conflicts[@]}"; do
            log_error "  ✗ $conflict"
        done
        echo ""
        log_warn "请修改 .env 文件，为这些服务分配其他端口。参考 .env.example 中的默认值:"
        log_info "  JUPYTERHUB_EXTERNAL_PORT=8088"
        log_info "  GITEA_EXTERNAL_PORT=3010"
        log_info "  APPHUB_PORT=28080"
        echo ""
        log_info "只有 EXTERNAL_PORT (Nginx HTTP) 和 HTTPS_PORT (Nginx HTTPS) 可以使用 80/443"
        return 1
    fi
    
    return 0
}

# ==============================================================================
# Build Cache Functions (Smart Incremental Build)
# ==============================================================================

# Initialize build cache directory and files
init_build_cache() {
    mkdir -p "$BUILD_CACHE_DIR"
    
    # Initialize build ID file
    if [[ ! -f "$BUILD_ID_FILE" ]]; then
        echo "0" > "$BUILD_ID_FILE"
    fi
    
    # Initialize build history file
    if [[ ! -f "$BUILD_HISTORY_FILE" ]]; then
        touch "$BUILD_HISTORY_FILE"
    fi
}

# Generate new build ID
generate_build_id() {
    init_build_cache
    
    local last_id=$(cat "$BUILD_ID_FILE" 2>/dev/null || echo "0")
    local new_id=$((last_id + 1))
    local timestamp=$(date +%Y%m%d_%H%M%S)
    
    echo "${new_id}_${timestamp}"
}

# Save build ID
save_build_id() {
    local build_id="$1"
    init_build_cache
    
    # Extract numeric ID part
    local numeric_id=$(echo "$build_id" | cut -d'_' -f1)
    echo "$numeric_id" > "$BUILD_ID_FILE"
}

# Calculate hash for file or directory
calculate_hash() {
    local path="$1"
    
    if [[ ! -e "$path" ]]; then
        echo "NOT_EXIST"
        return 1
    fi
    
    if [[ -d "$path" ]]; then
        # Directory: Calculate combined hash of all relevant files
        # Exclude common dependency and build directories for performance
        find "$path" -type f \
            \( -name "*.py" -o -name "*.js" -o -name "*.ts" -o -name "*.tsx" -o -name "*.go" \
               -o -name "*.conf" -o -name "*.yaml" -o -name "*.yml" -o -name "*.json" \
               -o -name "Dockerfile" -o -name "Dockerfile.tpl" -o -name "*.sh" \
               -o -name "*.html" -o -name "*.css" -o -name "*.scss" \) \
            ! -path "*/node_modules/*" \
            ! -path "*/build/*" \
            ! -path "*/dist/*" \
            ! -path "*/.next/*" \
            ! -path "*/vendor/*" \
            ! -path "*/__pycache__/*" \
            ! -path "*/.git/*" \
            ! -path "*/test-results/*" \
            ! -path "*/playwright-report/*" \
            -print0 2>/dev/null | xargs -0 shasum -a 256 2>/dev/null | sort | shasum -a 256 | awk '{print $1}'
    else
        # File: Calculate hash directly
        shasum -a 256 "$path" 2>/dev/null | awk '{print $1}'
    fi
}

# Calculate combined hash for a service (source code + config + Dockerfile)
calculate_service_hash() {
    local service="$1"
    local service_path="$SRC_DIR/$service"
    
    if [[ ! -d "$service_path" ]]; then
        echo "INVALID_SERVICE"
        return 1
    fi
    
    local hash_data=""
    
    # 1. Dockerfile hash
    local dockerfile="$service_path/Dockerfile"
    local dockerfile_tpl="$service_path/Dockerfile.tpl"
    if [[ -f "$dockerfile" ]]; then
        hash_data+="$(calculate_hash "$dockerfile")\n"
    fi
    if [[ -f "$dockerfile_tpl" ]]; then
        hash_data+="$(calculate_hash "$dockerfile_tpl")\n"
    fi
    
    # 2. Source code directory hash
    if [[ -d "$service_path" ]]; then
        hash_data+="$(calculate_hash "$service_path")\n"
    fi
    
    # 3. Configuration file hashes (service-specific)
    case "$service" in
        "nginx")
            if [[ -d "$SCRIPT_DIR/config/nginx" ]]; then
                hash_data+="$(calculate_hash "$SCRIPT_DIR/config/nginx")\n"
            fi
            ;;
        "jupyterhub")
            if [[ -f "$SCRIPT_DIR/config/jupyterhub_config.py" ]]; then
                hash_data+="$(calculate_hash "$SCRIPT_DIR/config/jupyterhub_config.py")\n"
            fi
            ;;
        "backend"|"backend-init")
            # Backend shares src/backend code
            if [[ -d "$SCRIPT_DIR/src/backend" ]]; then
                hash_data+="$(calculate_hash "$SCRIPT_DIR/src/backend")\n"
            fi
            ;;
        "saltstack")
            if [[ -d "$SCRIPT_DIR/config/salt" ]]; then
                hash_data+="$(calculate_hash "$SCRIPT_DIR/config/salt")\n"
            fi
            ;;
    esac
    
    # Calculate combined hash
    echo -e "$hash_data" | shasum -a 256 | awk '{print $1}'
}

# Check if service needs to be rebuilt
# Returns: FORCE_REBUILD, SKIP_CACHE_CHECK, IMAGE_NOT_EXIST, NO_HASH_LABEL, HASH_CHANGED, NO_CHANGE
need_rebuild() {
    local service="$1"
    local tag="${2:-${IMAGE_TAG:-latest}}"
    local image="ai-infra-${service}:${tag}"
    
    # Force rebuild mode
    if [[ "$FORCE_REBUILD" == "true" ]]; then
        echo "FORCE_REBUILD"
        return 0
    fi
    
    # Skip cache check mode
    if [[ "$SKIP_CACHE_CHECK" == "true" ]]; then
        echo "SKIP_CACHE_CHECK"
        return 0
    fi
    
    # Check if it's a dependency service (external image)
    local dep_conf="$SRC_DIR/$service/dependency.conf"
    if [[ -f "$dep_conf" ]]; then
        # For dependencies, just check if local image exists
        if docker image inspect "$image" >/dev/null 2>&1; then
            echo "NO_CHANGE"
            return 1
        else
            echo "IMAGE_NOT_EXIST"
            return 0
        fi
    fi
    
    # Image doesn't exist, need to build
    if ! docker image inspect "$image" >/dev/null 2>&1; then
        echo "IMAGE_NOT_EXIST"
        return 0
    fi
    
    # Calculate current file hash
    local current_hash=$(calculate_service_hash "$service")
    
    # Get hash stored in image label
    local image_hash=$(docker image inspect "$image" --format '{{index .Config.Labels "build.hash"}}' 2>/dev/null || echo "")
    
    # No hash label in image, need to rebuild
    if [[ -z "$image_hash" ]]; then
        echo "NO_HASH_LABEL"
        return 0
    fi
    
    # Compare hashes
    if [[ "$current_hash" != "$image_hash" ]]; then
        echo "HASH_CHANGED|old:${image_hash:0:8}|new:${current_hash:0:8}"
        return 0
    fi
    
    # No need to rebuild
    echo "NO_CHANGE"
    return 1
}

# Check if service needs to be rebuilt for a specific platform
# This is used by build_component_for_platform to check platform-specific images
# Returns: FORCE_REBUILD, SKIP_CACHE_CHECK, IMAGE_NOT_EXIST, NO_HASH_LABEL, HASH_CHANGED, NO_CHANGE
need_rebuild_for_platform() {
    local service="$1"
    local platform="$2"
    local tag="${3:-${IMAGE_TAG:-latest}}"
    
    # Normalize platform and get arch name
    if [[ "$platform" != *"/"* ]]; then
        platform="linux/$platform"
    fi
    local arch_name="${platform##*/}"
    
    # Always use architecture-suffixed tag for multi-platform builds
    # This matches the image naming convention in build_component_for_platform()
    # Examples: ai-infra-apphub:v0.3.8-arm64, ai-infra-apphub:v0.3.8-amd64
    local arch_suffix="-${arch_name}"
    local image="ai-infra-${service}:${tag}${arch_suffix}"
    
    # Force rebuild mode
    if [[ "$FORCE_REBUILD" == "true" ]]; then
        echo "FORCE_REBUILD"
        return 0
    fi
    
    # Skip cache check mode
    if [[ "$SKIP_CACHE_CHECK" == "true" ]]; then
        echo "SKIP_CACHE_CHECK"
        return 0
    fi
    
    # Check if it's a dependency service (external image)
    local dep_conf="$SRC_DIR/$service/dependency.conf"
    if [[ -f "$dep_conf" ]]; then
        # For dependencies, just check if local image exists
        if docker image inspect "$image" >/dev/null 2>&1; then
            echo "NO_CHANGE"
            return 1
        else
            echo "IMAGE_NOT_EXIST"
            return 0
        fi
    fi
    
    # Image doesn't exist, need to build
    if ! docker image inspect "$image" >/dev/null 2>&1; then
        echo "IMAGE_NOT_EXIST"
        return 0
    fi
    
    # Calculate current file hash
    local current_hash=$(calculate_service_hash "$service")
    
    # Get hash stored in image label
    local image_hash=$(docker image inspect "$image" --format '{{index .Config.Labels "build.hash"}}' 2>/dev/null || echo "")
    
    # No hash label in image, need to rebuild
    if [[ -z "$image_hash" ]]; then
        echo "NO_HASH_LABEL"
        return 0
    fi
    
    # Compare hashes
    if [[ "$current_hash" != "$image_hash" ]]; then
        echo "HASH_CHANGED|old:${image_hash:0:8}|new:${current_hash:0:8}"
        return 0
    fi
    
    # No need to rebuild
    echo "NO_CHANGE"
    return 1
}

# Log build history
log_build_history() {
    local build_id="$1"
    local service="$2"
    local tag="$3"
    local status="$4"  # SUCCESS/FAILED/SKIPPED
    local reason="${5:-}"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    
    init_build_cache
    
    local log_entry="[$timestamp] BUILD_ID=$build_id SERVICE=$service TAG=$tag STATUS=$status"
    if [[ -n "$reason" ]]; then
        log_entry+=" REASON=$reason"
    fi
    
    echo "$log_entry" >> "$BUILD_HISTORY_FILE"
}

# Save service build info to cache
save_service_build_info() {
    local service="$1"
    local tag="$2"
    local build_id="$3"
    local service_hash="$4"
    
    local cache_dir="$BUILD_CACHE_DIR/$service"
    mkdir -p "$cache_dir"
    
    local build_info_file="$cache_dir/last-build.json"
    
    cat > "$build_info_file" <<EOF
{
  "service": "$service",
  "tag": "$tag",
  "build_id": "$build_id",
  "hash": "$service_hash",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "image": "ai-infra-${service}:${tag}"
}
EOF
}

# Show build history
show_build_history() {
    local filter_service="$1"
    local count="${2:-20}"
    
    init_build_cache
    
    if [[ ! -f "$BUILD_HISTORY_FILE" ]] || [[ ! -s "$BUILD_HISTORY_FILE" ]]; then
        log_info "📋 Build history is empty"
        log_info "Tip: History will be recorded after builds"
        return 0
    fi
    
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "📋 Build History (last $count entries)"
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    if [[ -n "$filter_service" ]]; then
        grep "SERVICE=$filter_service" "$BUILD_HISTORY_FILE" | tail -n "$count"
    else
        tail -n "$count" "$BUILD_HISTORY_FILE"
    fi
    
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

# Show cache status for all services
show_cache_status() {
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "📊 Build Cache Status"
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    local tag="${IMAGE_TAG:-latest}"
    local need_build=0
    local skip_build=0
    
    # Discover services
    discover_services 2>/dev/null
    
    local all_services=("${DEPENDENCY_SERVICES[@]}" "${FOUNDATION_SERVICES[@]}" "${DEPENDENT_SERVICES[@]}")
    
    printf "%-25s %-15s %-50s\n" "SERVICE" "STATUS" "REASON"
    printf "%-25s %-15s %-50s\n" "-------" "------" "------"
    
    for service in "${all_services[@]}"; do
        local result=$(need_rebuild "$service" "$tag")
        local status
        
        if [[ "$result" == "NO_CHANGE" ]]; then
            status="${GREEN}✓ CACHED${NC}"
            skip_build=$((skip_build + 1))
        else
            status="${YELLOW}○ REBUILD${NC}"
            need_build=$((need_build + 1))
        fi
        
        printf "%-25s %-15b %-50s\n" "$service" "$status" "$result"
    done
    
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "Summary: $skip_build cached, $need_build need rebuild"
}

# Clear build cache
clear_build_cache() {
    local service="$1"
    
    if [[ -n "$service" ]]; then
        local cache_dir="$BUILD_CACHE_DIR/$service"
        if [[ -d "$cache_dir" ]]; then
            rm -rf "$cache_dir"
            log_info "✓ Cleared cache for: $service"
        else
            log_warn "No cache found for: $service"
        fi
    else
        if [[ -d "$BUILD_CACHE_DIR" ]]; then
            rm -rf "$BUILD_CACHE_DIR"
            log_info "✓ Cleared all build cache"
        else
            log_warn "No build cache found"
        fi
    fi
}

# ==============================================================================
# Production Environment Password Generator
# ==============================================================================

# Generate random password with specified length and type
generate_random_password() {
    local length="${1:-24}"
    local password_type="${2:-standard}"  # standard, hex, alphanumeric
    
    case "$password_type" in
        "hex")
            # Hex key (for JupyterHub crypt key etc.)
            openssl rand -hex "$((length/2))"
            ;;
        "alphanumeric")
            # Alphanumeric only, avoid special characters
            LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom | head -c "$length"
            ;;
        "standard"|*)
            # Standard password: letters, numbers, safe special chars
            openssl rand -base64 "$((length * 3 / 4))" | tr -d "=+/\n" | cut -c1-"$length"
            ;;
    esac
}

# Generate production environment with strong random passwords
# Usage: ./build.sh gen-prod-env [output_file] [--force]
generate_production_env() {
    local env_file="${1:-.env.prod}"
    local force="${2:-false}"
    
    # Ensure UTF-8 locale for proper handling of Chinese characters in .env.example
    # This is critical to avoid garbled text in generated .env files
    local _orig_lc_all="${LC_ALL:-}"
    local _orig_lang="${LANG:-}"
    
    # Try different UTF-8 locale names (different systems use different names)
    if locale -a 2>/dev/null | grep -qiE '^(C\.UTF-8|en_US\.UTF-8|en_US\.utf8)$'; then
        if locale -a 2>/dev/null | grep -qi '^C\.UTF-8$'; then
            export LC_ALL=C.UTF-8
            export LANG=C.UTF-8
        elif locale -a 2>/dev/null | grep -qi '^en_US\.UTF-8$'; then
            export LC_ALL=en_US.UTF-8
            export LANG=en_US.UTF-8
        elif locale -a 2>/dev/null | grep -qi '^en_US\.utf8$'; then
            export LC_ALL=en_US.utf8
            export LANG=en_US.utf8
        fi
    fi
    
    log_info "======================================================================"
    log_info "🔧 AI Infrastructure Matrix - Production Environment Generator"
    log_info "======================================================================"
    log_warn "⚠️  This will generate new random passwords for all services"
    log_warn "⚠️  Default admin account (admin/admin123) is NOT changed by this script"
    log_warn "⚠️  Please change admin password via Web UI after deployment"
    log_info "======================================================================"
    
    # Detect external IP address
    log_info "Detecting external IP address..."
    local detected_external_host=$(detect_external_host)
    if [[ "$detected_external_host" == "localhost" ]]; then
        log_warn "⚠️  Could not detect external IP, using 'localhost'"
        log_warn "⚠️  Please manually set EXTERNAL_HOST in $env_file if needed"
    else
        log_info "✅ Detected external IP: $detected_external_host"
    fi
    
    # Check if target file exists
    if [[ -f "$env_file" ]] && [[ "$force" != "true" ]]; then
        log_warn "Target file already exists: $env_file"
        log_info "Use --force to overwrite, or specify a different filename"
        return 1
    fi
    
    # Create from .env.example if template exists
    if [[ ! -f ".env.example" ]]; then
        log_error ".env.example template not found!"
        return 1
    fi
    
    log_info "Creating production environment file: $env_file"
    
    # Create backup if overwriting
    if [[ -f "$env_file" ]] && [[ "$force" == "true" ]]; then
        local backup_file="${env_file}.backup.$(date +%Y%m%d_%H%M%S)"
        log_info "Creating backup: $backup_file"
        cp "$env_file" "$backup_file"
    fi
    
    cp ".env.example" "$env_file"
    
    log_info "Generating strong random passwords..."
    
    # Generate all passwords (organized by category)
    # Database passwords
    local postgres_password=$(generate_random_password 24 "alphanumeric")
    local jupyterhub_db_password=$(generate_random_password 24 "alphanumeric")
    local mysql_root_password=$(generate_random_password 24 "alphanumeric")
    local mysql_password=$(generate_random_password 24 "alphanumeric")
    local redis_password=$(generate_random_password 24 "alphanumeric")
    local ai_db_password=$(generate_random_password 24 "alphanumeric")
    local grafana_admin_password=$(generate_random_password 24 "alphanumeric")
    
    # Authentication secrets
    local jwt_secret=$(generate_random_password 48 "standard")
    local encryption_key=$(generate_random_password 32 "alphanumeric")
    local session_secret=$(generate_random_password 48 "standard")
    local configproxy_token=$(generate_random_password 48 "standard")
    local ai_infra_api_token=$(generate_random_password 48 "standard")
    
    # JupyterHub
    local jupyterhub_crypt_key=$(generate_random_password 64 "hex")
    
    # SeaweedFS storage
    local seaweedfs_access_key=$(generate_random_password 20 "alphanumeric")
    local seaweedfs_secret_key=$(generate_random_password 40 "standard")
    local seaweedfs_app_access=$(generate_random_password 20 "alphanumeric")
    local seaweedfs_app_secret=$(generate_random_password 40 "standard")
    local seaweedfs_readonly_access=$(generate_random_password 20 "alphanumeric")
    local seaweedfs_readonly_secret=$(generate_random_password 40 "standard")
    
    # Gitea
    local gitea_admin_password=$(generate_random_password 24 "alphanumeric")
    local gitea_admin_token=$(generate_random_password 40 "alphanumeric")
    
    # LDAP
    local ldap_admin_password=$(generate_random_password 24 "alphanumeric")
    local ldap_config_password=$(generate_random_password 24 "alphanumeric")
    
    # Slurm HPC
    local slurm_db_password=$(generate_random_password 24 "alphanumeric")
    local slurm_munge_key=$(generate_random_password 48 "standard")
    local slurm_node_ssh_password=$(generate_random_password 24 "alphanumeric")
    
    # SaltStack
    local salt_api_password=$(generate_random_password 24 "alphanumeric")
    local saltstack_api_token=$(generate_random_password 48 "standard")
    
    # Test/SSH passwords
    local test_ssh_password=$(generate_random_password 20 "alphanumeric")
    local test_root_password=$(generate_random_password 20 "alphanumeric")
    
    # Use awk for safe replacement (handles special characters)
    # IMPORTANT: Explicitly set UTF-8 locale to preserve Chinese characters in comments
    local temp_file="${env_file}.updating"
    
    LC_ALL=C.UTF-8 LANG=C.UTF-8 awk -v pg_pass="$postgres_password" \
        -v hub_db_pass="$jupyterhub_db_password" \
        -v mysql_root="$mysql_root_password" \
        -v mysql_pass="$mysql_password" \
        -v redis_pass="$redis_password" \
        -v ai_db_pass="$ai_db_password" \
        -v grafana_pass="$grafana_admin_password" \
        -v jwt_sec="$jwt_secret" \
        -v enc_key="$encryption_key" \
        -v sess_sec="$session_secret" \
        -v config_token="$configproxy_token" \
        -v api_token="$ai_infra_api_token" \
        -v hub_key="$jupyterhub_crypt_key" \
        -v sw_access="$seaweedfs_access_key" \
        -v sw_secret="$seaweedfs_secret_key" \
        -v sw_app_access="$seaweedfs_app_access" \
        -v sw_app_secret="$seaweedfs_app_secret" \
        -v sw_ro_access="$seaweedfs_readonly_access" \
        -v sw_ro_secret="$seaweedfs_readonly_secret" \
        -v gitea_admin="$gitea_admin_password" \
        -v gitea_token="$gitea_admin_token" \
        -v ldap_admin="$ldap_admin_password" \
        -v ldap_config="$ldap_config_password" \
        -v slurm_db="$slurm_db_password" \
        -v munge_key="$slurm_munge_key" \
        -v slurm_ssh="$slurm_node_ssh_password" \
        -v salt_api="$salt_api_password" \
        -v salt_token="$saltstack_api_token" \
        -v test_ssh="$test_ssh_password" \
        -v test_root="$test_root_password" \
        -v ext_host="$detected_external_host" \
        '
        /^EXTERNAL_HOST=/ { print "EXTERNAL_HOST=" ext_host; next }
        /^EXTERNAL_PORT=/ { print "EXTERNAL_PORT=80"; next }
        /^HTTPS_PORT=/ { print "HTTPS_PORT=443"; next }
        /^EXTERNAL_SCHEME=/ { print "EXTERNAL_SCHEME=https"; next }
        /^POSTGRES_PASSWORD=/ { print "POSTGRES_PASSWORD=" pg_pass; next }
        /^JUPYTERHUB_DB_PASSWORD=/ { print "JUPYTERHUB_DB_PASSWORD=" hub_db_pass; next }
        /^MYSQL_ROOT_PASSWORD=/ { print "MYSQL_ROOT_PASSWORD=" mysql_root; next }
        /^MYSQL_PASSWORD=/ { print "MYSQL_PASSWORD=" mysql_pass; next }
        /^REDIS_PASSWORD=/ { print "REDIS_PASSWORD=" redis_pass; next }
        /^AI_DB_PASSWORD=/ { print "AI_DB_PASSWORD=" ai_db_pass; next }
        /^GRAFANA_ADMIN_PASSWORD=/ { print "GRAFANA_ADMIN_PASSWORD=" grafana_pass; next }
        /^JWT_SECRET=/ { print "JWT_SECRET=" jwt_sec; next }
        /^ENCRYPTION_KEY=/ { print "ENCRYPTION_KEY=" enc_key; next }
        /^SESSION_SECRET=/ { print "SESSION_SECRET=" sess_sec; next }
        /^CONFIGPROXY_AUTH_TOKEN=/ { print "CONFIGPROXY_AUTH_TOKEN=" config_token; next }
        /^AI_INFRA_API_TOKEN=/ { print "AI_INFRA_API_TOKEN=" api_token; next }
        /^JUPYTERHUB_CRYPT_KEY=/ { print "JUPYTERHUB_CRYPT_KEY=" hub_key; next }
        /^SEAWEEDFS_ACCESS_KEY=/ { print "SEAWEEDFS_ACCESS_KEY=" sw_access; next }
        /^SEAWEEDFS_SECRET_KEY=/ { print "SEAWEEDFS_SECRET_KEY=" sw_secret; next }
        /^SEAWEEDFS_APP_ACCESS_KEY=/ { print "SEAWEEDFS_APP_ACCESS_KEY=" sw_app_access; next }
        /^SEAWEEDFS_APP_SECRET_KEY=/ { print "SEAWEEDFS_APP_SECRET_KEY=" sw_app_secret; next }
        /^SEAWEEDFS_READONLY_ACCESS_KEY=/ { print "SEAWEEDFS_READONLY_ACCESS_KEY=" sw_ro_access; next }
        /^SEAWEEDFS_READONLY_SECRET_KEY=/ { print "SEAWEEDFS_READONLY_SECRET_KEY=" sw_ro_secret; next }
        /^GITEA_ADMIN_PASSWORD=/ { print "GITEA_ADMIN_PASSWORD=" gitea_admin; next }
        /^GITEA_ADMIN_TOKEN=/ { print "GITEA_ADMIN_TOKEN=" gitea_token; next }
        /^LDAP_ADMIN_PASSWORD=/ { print "LDAP_ADMIN_PASSWORD=" ldap_admin; next }
        /^LDAP_CONFIG_PASSWORD=/ { print "LDAP_CONFIG_PASSWORD=" ldap_config; next }
        /^SLURM_DB_PASSWORD=/ { print "SLURM_DB_PASSWORD=" slurm_db; next }
        /^SLURM_MUNGE_KEY=/ { print "SLURM_MUNGE_KEY=" munge_key; next }
        /^SLURM_NODE_SSH_PASSWORD=/ { print "SLURM_NODE_SSH_PASSWORD=" slurm_ssh; next }
        /^SALT_API_PASSWORD=/ { print "SALT_API_PASSWORD=" salt_api; next }
        /^SALTSTACK_API_TOKEN=/ { print "SALTSTACK_API_TOKEN=" salt_token; next }
        /^TEST_SSH_PASSWORD=/ { print "TEST_SSH_PASSWORD=" test_ssh; next }
        /^TEST_ROOT_PASSWORD=/ { print "TEST_ROOT_PASSWORD=" test_root; next }
        { print }
        ' "$env_file" > "$temp_file"
    
    mv "$temp_file" "$env_file"
    
    log_info "======================================================================"
    log_warn "🔑 IMPORTANT: Default Admin Account"
    echo ""
    log_info "  Username: admin"
    log_error "  Password: admin123"
    echo ""
    log_warn "⚠️  Please change admin password after first login!"
    log_warn "⚠️  Admin password is NOT modified by this script!"
    log_info "======================================================================"
    
    log_info "Generated passwords for services:"
    echo ""
    echo "  📦 Database Passwords:"
    echo "    POSTGRES_PASSWORD=$postgres_password"
    echo "    JUPYTERHUB_DB_PASSWORD=$jupyterhub_db_password"
    echo "    MYSQL_ROOT_PASSWORD=$mysql_root_password"
    echo "    MYSQL_PASSWORD=$mysql_password"
    echo "    REDIS_PASSWORD=$redis_password"
    echo "    AI_DB_PASSWORD=$ai_db_password"
    echo "    GRAFANA_ADMIN_PASSWORD=$grafana_admin_password"
    echo ""
    echo "  🔐 Authentication Secrets:"
    echo "    JWT_SECRET=$jwt_secret"
    echo "    ENCRYPTION_KEY=$encryption_key"
    echo "    SESSION_SECRET=$session_secret"
    echo "    CONFIGPROXY_AUTH_TOKEN=$configproxy_token"
    echo "    AI_INFRA_API_TOKEN=$ai_infra_api_token"
    echo "    JUPYTERHUB_CRYPT_KEY=$jupyterhub_crypt_key"
    echo ""
    echo "  📁 SeaweedFS Storage:"
    echo "    SEAWEEDFS_ACCESS_KEY=$seaweedfs_access_key"
    echo "    SEAWEEDFS_SECRET_KEY=$seaweedfs_secret_key"
    echo "    SEAWEEDFS_APP_ACCESS_KEY=$seaweedfs_app_access"
    echo "    SEAWEEDFS_APP_SECRET_KEY=$seaweedfs_app_secret"
    echo "    SEAWEEDFS_READONLY_ACCESS_KEY=$seaweedfs_readonly_access"
    echo "    SEAWEEDFS_READONLY_SECRET_KEY=$seaweedfs_readonly_secret"
    echo ""
    echo "  🔧 Services:"
    echo "    GITEA_ADMIN_PASSWORD=$gitea_admin_password"
    echo "    GITEA_ADMIN_TOKEN=$gitea_admin_token"
    echo "    LDAP_ADMIN_PASSWORD=$ldap_admin_password"
    echo "    LDAP_CONFIG_PASSWORD=$ldap_config_password"
    echo ""
    echo "  🖥️ HPC & Automation:"
    echo "    SLURM_DB_PASSWORD=$slurm_db_password"
    echo "    SLURM_MUNGE_KEY=$slurm_munge_key"
    echo "    SLURM_NODE_SSH_PASSWORD=$slurm_node_ssh_password"
    echo "    SALT_API_PASSWORD=$salt_api_password"
    echo "    SALTSTACK_API_TOKEN=$saltstack_api_token"
    echo ""
    echo "  🧪 Test Passwords:"
    echo "    TEST_SSH_PASSWORD=$test_ssh_password"
    echo "    TEST_ROOT_PASSWORD=$test_root_password"
    
    log_warn ""
    log_warn "⚠️  Please save these passwords securely!"
    log_info "Production environment file created: $env_file"
    
    # Display detected external host
    log_info ""
    log_info "🌐 Network Configuration:"
    echo "    EXTERNAL_HOST=$detected_external_host"
    echo "    EXTERNAL_PORT=80      (HTTP redirect port)"
    echo "    HTTPS_PORT=443        (HTTPS main service port)"
    echo "    EXTERNAL_SCHEME=https"
    log_info ""
    log_info "📋 Port Mapping (for Cloudflare Full mode):"
    echo "    外部 80  → 容器 80  (HTTP → HTTPS 重定向)"
    echo "    外部 443 → 容器 443 (HTTPS 主服务)"
    
    # Auto copy .env.prod to .env
    log_info ""
    log_info "Automatically copying $env_file to .env..."
    cp "$env_file" .env
    if [[ $? -eq 0 ]]; then
        log_info "✅ Copied $env_file to .env successfully"
    else
        log_error "❌ Failed to copy $env_file to .env"
        log_info "  Please manually run: cp $env_file .env"
    fi
    
    log_info ""
    log_info "Next steps:"
    log_info "  1. Review and edit .env if needed (adjust DOMAIN, ports, etc.)"
    log_info "  2. Render templates: ./build.sh render"
    log_info "  3. Build and deploy: ./build.sh build-all && docker compose up -d"
    
    # Restore original locale settings
    if [[ -n "$_orig_lc_all" ]]; then
        export LC_ALL="$_orig_lc_all"
    else
        unset LC_ALL
    fi
    if [[ -n "$_orig_lang" ]]; then
        export LANG="$_orig_lang"
    else
        unset LANG
    fi
    
    return 0
}

# 初始化或同步 .env 文件
# 自动检测 EXTERNAL_HOST 等关键变量
init_env_file() {
    local force="${1:-false}"
    
    # 检测外部地址
    local detected_host=$(detect_external_host)
    local detected_port="${EXTERNAL_PORT:-8080}"
    
    if [[ ! -f "$ENV_FILE" ]]; then
        log_info "Creating .env from .env.example..."
        if [[ -f "$ENV_EXAMPLE" ]]; then
            cp "$ENV_EXAMPLE" "$ENV_FILE"
        else
            log_error ".env.example not found!"
            return 1
        fi
        force="true"
    fi
    
    # 同步 .env.example 中的新变量到 .env
    sync_env_with_example
    
    # 从 .env 读取当前的 scheme 设置（保留用户/example 的配置）
    local current_scheme=$(grep "^EXTERNAL_SCHEME=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2)
    local detected_scheme="${current_scheme:-https}"
    
    # 检查是否需要更新关键变量
    local current_host=$(grep "^EXTERNAL_HOST=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2)
    
    # 如果是自引用或空值，需要更新
    if [[ "$force" == "true" ]] || [[ "$current_host" =~ \$\{ ]] || [[ -z "$current_host" ]]; then
        log_info "Initializing environment variables..."
        log_info "  EXTERNAL_HOST=$detected_host"
        log_info "  EXTERNAL_PORT=$detected_port"
        log_info "  EXTERNAL_SCHEME=$detected_scheme"
        
        update_env_variable "EXTERNAL_HOST" "$detected_host"
        update_env_variable "DOMAIN" "$detected_host"
        update_env_variable "EXTERNAL_PORT" "$detected_port"
        update_env_variable "EXTERNAL_SCHEME" "$detected_scheme"
        
        log_info "✓ Environment variables initialized"
    else
        # .env 已存在且有有效的 EXTERNAL_HOST，检查 IP 是否变更
        check_ip_change "$current_host" "$detected_host"
    fi
}

# 检查 IP 是否变更，如果变更则提示用户
check_ip_change() {
    local current_host="$1"
    local detected_host="$2"
    
    # 如果当前配置的是域名而非 IP，跳过检查
    if [[ ! "$current_host" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        return 0
    fi
    
    # 如果检测到的也不是 IP（比如是 localhost），跳过检查
    if [[ ! "$detected_host" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        return 0
    fi
    
    # 比较 IP 是否变更
    if [[ "$current_host" != "$detected_host" ]]; then
        echo ""
        log_warn "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        log_warn "⚠️  检测到 IP 地址变更 / IP Address Change Detected"
        log_warn "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        log_warn "  当前配置 / Current:  EXTERNAL_HOST=$current_host"
        log_warn "  检测到的 / Detected: EXTERNAL_HOST=$detected_host"
        log_warn "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo ""
        
        # 交互式询问用户
        if [[ -t 0 ]]; then
            # 终端交互模式
            read -p "是否更新为新检测到的 IP? / Update to detected IP? [y/N]: " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                log_info "正在更新 EXTERNAL_HOST 为 $detected_host..."
                update_env_variable "EXTERNAL_HOST" "$detected_host"
                update_env_variable "DOMAIN" "$detected_host"
                log_info "✓ IP 地址已更新 / IP address updated"
                log_info "  请重新执行 render 以更新配置文件 / Please re-run render to update config files"
                log_info "  命令 / Command: ./build.sh render"
            else
                log_info "保持当前配置 / Keeping current configuration: $current_host"
                log_info "  如需手动更新，请编辑 .env 文件 / To update manually, edit .env file"
            fi
        else
            # 非交互模式，仅提示
            log_warn "非交互模式，保持当前配置 / Non-interactive mode, keeping current config"
            log_warn "如需更新，请执行 / To update, run:"
            log_warn "  ./build.sh init --force"
            log_warn "  或手动编辑 .env 文件 / or edit .env file manually"
        fi
        echo ""
    fi
}

# ==============================================================================
# 环境初始化 (跳过 --help 模式)
# ==============================================================================
if [[ "$_SHOW_HELP_ONLY" != "true" ]]; then
    # 初始化环境
    init_env_file

    # Load .env variables
    set -a
    source "$ENV_FILE"
    set +a
fi

# Detect host platform for multi-arch image pulls
# Returns: linux/amd64 or linux/arm64
_detect_docker_platform() {
    local arch=$(uname -m)
    case "$arch" in
        x86_64|amd64)
            echo "linux/amd64"
            ;;
        aarch64|arm64)
            echo "linux/arm64"
            ;;
        armv7l|armhf)
            echo "linux/arm/v7"
            ;;
        *)
            # Fallback: let docker decide
            echo ""
            ;;
    esac
}

# Host platform for pulling correct architecture images
DOCKER_HOST_PLATFORM="${DOCKER_HOST_PLATFORM:-$(_detect_docker_platform)}"

# Initialize COMMON_IMAGES array after loading .env
# This ensures version variables are available
# Note: These use default values if .env not loaded (help mode)
COMMON_IMAGES=(
    "postgres:${POSTGRES_VERSION:-15-alpine}"
    "mysql:${MYSQL_VERSION:-8.0}"
    "redis:${REDIS_VERSION:-7-alpine}"
    "confluentinc/cp-kafka:${KAFKA_VERSION:-7.5.0}"
    "provectuslabs/kafka-ui:${KAFKA_UI_VERSION:-latest}"
    "osixia/openldap:${OPENLDAP_VERSION:-stable}"
    "osixia/phpldapadmin:${PHPLDAPADMIN_VERSION:-stable}"
    "redis/redisinsight:${REDISINSIGHT_VERSION:-2.68}"
    "chrislusf/seaweedfs:${SEAWEEDFS_VERSION:-3.80}"
    "oceanbase/oceanbase-ce:${OCEANBASE_VERSION:-4.3.5-lts}"
    "victoriametrics/victoria-metrics:${VICTORIAMETRICS_VERSION:-v1.115.0}"
    "${GITEA_IMAGE:-gitea/gitea:${GITEA_VERSION:-1.25.3}}"
)

# Initialize SAFELINE_IMAGES array for SafeLine WAF
# Architecture suffix is auto-detected: -arm for ARM/aarch64, empty for x86_64
# Configuration is defined in config/images.yaml
_detect_safeline_arch_suffix() {
    local arch=$(uname -m)
    if [[ "$arch" =~ "aarch" || "$arch" =~ "arm" ]]; then
        echo "-arm"
    else
        echo ""
    fi
}

SAFELINE_ARCH_SUFFIX="${SAFELINE_ARCH_SUFFIX:-$(_detect_safeline_arch_suffix)}"
SAFELINE_IMAGE_PREFIX="${SAFELINE_IMAGE_PREFIX:-chaitin}"
SAFELINE_IMAGE_TAG="${SAFELINE_IMAGE_TAG:-9.3.0}"
SAFELINE_REGION="${SAFELINE_REGION:-}"

SAFELINE_IMAGES=(
    "${SAFELINE_IMAGE_PREFIX}/safeline-postgres${SAFELINE_ARCH_SUFFIX}:15.2"
    "${SAFELINE_IMAGE_PREFIX}/safeline-mgt${SAFELINE_REGION}${SAFELINE_ARCH_SUFFIX}:${SAFELINE_IMAGE_TAG}"
    "${SAFELINE_IMAGE_PREFIX}/safeline-detector${SAFELINE_REGION}${SAFELINE_ARCH_SUFFIX}:${SAFELINE_IMAGE_TAG}"
    "${SAFELINE_IMAGE_PREFIX}/safeline-tengine${SAFELINE_REGION}${SAFELINE_ARCH_SUFFIX}:${SAFELINE_IMAGE_TAG}"
    "${SAFELINE_IMAGE_PREFIX}/safeline-luigi${SAFELINE_REGION}${SAFELINE_ARCH_SUFFIX}:${SAFELINE_IMAGE_TAG}"
    "${SAFELINE_IMAGE_PREFIX}/safeline-fvm${SAFELINE_REGION}${SAFELINE_ARCH_SUFFIX}:${SAFELINE_IMAGE_TAG}"
    "${SAFELINE_IMAGE_PREFIX}/safeline-chaos${SAFELINE_REGION}${SAFELINE_ARCH_SUFFIX}:${SAFELINE_IMAGE_TAG}"
)

# Ensure SSH Keys (skip in help mode)
SSH_KEY_DIR="$SCRIPT_DIR/ssh-key"
if [[ "$_SHOW_HELP_ONLY" != "true" ]] && [ ! -f "$SSH_KEY_DIR/id_rsa" ]; then
    log_info "Generating SSH keys..."
    mkdir -p "$SSH_KEY_DIR"
    ssh-keygen -t rsa -b 4096 -f "$SSH_KEY_DIR/id_rsa" -N "" -C "ai-infra-system@shared"
fi

# Ensure Third Party Directory (skip in help mode)
if [[ "$_SHOW_HELP_ONLY" != "true" ]]; then
    mkdir -p "$SCRIPT_DIR/third_party"
fi

# ==============================================================================
# 2. Helper Functions
# ==============================================================================

# ==============================================================================
# SSL Certificate Generation Functions (内置，无需外部脚本)
# ==============================================================================

# SSL 配置常量
SSL_VALID_DAYS=${SSL_VALID_DAYS:-3650}  # 10年有效期
SSL_KEY_SIZE=${SSL_KEY_SIZE:-2048}
SSL_COUNTRY=${SSL_COUNTRY:-CN}
SSL_STATE=${SSL_STATE:-Beijing}
SSL_CITY=${SSL_CITY:-Beijing}
SSL_ORG=${SSL_ORG:-AI-Infra-Matrix}
SSL_CA_NAME=${SSL_CA_NAME:-AI-Infra-Matrix-CA}
LETSENCRYPT_EMAIL=${LETSENCRYPT_EMAIL:-${ACME_EMAIL:-}}
LETSENCRYPT_STAGING=${LETSENCRYPT_STAGING:-false}
LETSENCRYPT_EXTRA_DOMAINS=${LETSENCRYPT_EXTRA_DOMAINS:-}
# SSL 证书输出目录 (放在 src/nginx/ssl 以便打包到镜像)
SSL_OUTPUT_DIR="$SCRIPT_DIR/src/nginx/ssl"

is_existing_cert_valid() {
    local cert_file="$1"
    local seconds=${2:-2592000}
    openssl x509 -checkend "$seconds" -noout -in "$cert_file" >/dev/null 2>&1
}

# 生成 CA 根证书
generate_ca_certificate() {
    local output_dir="$SSL_OUTPUT_DIR"
    local ca_dir="$output_dir/ca"
    
    log_step "生成 CA 根证书..."
    
    mkdir -p "$ca_dir"
    
    # 生成 CA 私钥
    log_info "生成 CA 私钥..."
    openssl genrsa -out "$ca_dir/ca.key" $SSL_KEY_SIZE 2>/dev/null
    chmod 600 "$ca_dir/ca.key"
    
    # 生成 CA 证书
    log_info "生成 CA 证书..."
    openssl req -new -x509 -days $SSL_VALID_DAYS -key "$ca_dir/ca.key" \
        -out "$ca_dir/ca.crt" \
        -subj "/C=$SSL_COUNTRY/ST=$SSL_STATE/L=$SSL_CITY/O=$SSL_ORG/CN=$SSL_CA_NAME" \
        2>/dev/null
    
    if [[ $? -eq 0 ]]; then
        log_info "✓ CA 根证书生成成功"
        log_info "  私钥: $ca_dir/ca.key"
        log_info "  证书: $ca_dir/ca.crt"
        return 0
    else
        log_error "CA 证书生成失败"
        return 1
    fi
}

# 生成服务器证书
generate_server_certificate() {
    local domain="$1"
    local output_dir="$SSL_OUTPUT_DIR"
    local ca_dir="$output_dir/ca"
    
    if [[ -z "$domain" ]]; then
        log_error "域名不能为空"
        return 1
    fi
    
    # 检查 CA 是否存在
    if [[ ! -f "$ca_dir/ca.key" ]] || [[ ! -f "$ca_dir/ca.crt" ]]; then
        log_error "CA 证书不存在，请先生成 CA"
        return 1
    fi
    
    log_step "为域名生成服务器证书: $domain"
    
    # 安全文件名 (将 * 替换为 _wildcard_)
    local safe_name=$(echo "$domain" | sed 's/\*/_wildcard_/g')
    
    # 生成服务器私钥
    log_info "生成服务器私钥..."
    openssl genrsa -out "$output_dir/$safe_name.key" $SSL_KEY_SIZE 2>/dev/null
    chmod 600 "$output_dir/$safe_name.key"
    
    # 创建 SAN 扩展配置
    local san_config=$(mktemp)
    cat > "$san_config" << EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[req_distinguished_name]
C = $SSL_COUNTRY
ST = $SSL_STATE
L = $SSL_CITY
O = $SSL_ORG
CN = $domain

[v3_req]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
subjectAltName = @alt_names

[alt_names]
EOF

    # 添加 SAN (支持 IP 和域名)
    local san_index=1
    if [[ "$domain" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        echo "IP.$san_index = $domain" >> "$san_config"
        san_index=$((san_index + 1))
        # 添加 localhost
        echo "IP.$san_index = 127.0.0.1" >> "$san_config"
        echo "DNS.1 = localhost" >> "$san_config"
    else
        echo "DNS.$san_index = $domain" >> "$san_config"
        san_index=$((san_index + 1))
        echo "DNS.$san_index = localhost" >> "$san_config"
        echo "IP.1 = 127.0.0.1" >> "$san_config"
    fi
    
    # 生成证书签名请求 (CSR)
    log_info "生成证书签名请求..."
    openssl req -new -key "$output_dir/$safe_name.key" \
        -out "$output_dir/$safe_name.csr" \
        -config "$san_config" \
        2>/dev/null
    
    # 创建签名扩展配置
    local ext_config=$(mktemp)
    cat > "$ext_config" << EOF
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
subjectAltName = @alt_names

[alt_names]
EOF
    
    # 复制 SAN 配置
    grep -E "^(DNS|IP)\." "$san_config" >> "$ext_config"
    
    # 使用 CA 签发证书
    log_info "使用 CA 签发证书..."
    openssl x509 -req -days $SSL_VALID_DAYS \
        -in "$output_dir/$safe_name.csr" \
        -CA "$ca_dir/ca.crt" \
        -CAkey "$ca_dir/ca.key" \
        -CAcreateserial \
        -out "$output_dir/$safe_name.crt" \
        -extfile "$ext_config" \
        2>/dev/null
    
    # 创建证书链
    cat "$output_dir/$safe_name.crt" "$ca_dir/ca.crt" > "$output_dir/$safe_name.chain.crt"
    
    # 创建通用名称的符号链接 (server.crt/server.key)
    ln -sf "$safe_name.crt" "$output_dir/server.crt"
    ln -sf "$safe_name.key" "$output_dir/server.key"
    ln -sf "$safe_name.chain.crt" "$output_dir/server.chain.crt"
    cp "$ca_dir/ca.crt" "$output_dir/ca.crt"
    
    # 清理临时文件
    rm -f "$san_config" "$ext_config" "$output_dir/$safe_name.csr"
    
    if [[ -f "$output_dir/$safe_name.crt" ]]; then
        local cert_subject=$(openssl x509 -in "$output_dir/$safe_name.crt" -noout -subject 2>/dev/null | sed 's/subject=//')
        local cert_expire=$(openssl x509 -in "$output_dir/$safe_name.crt" -noout -enddate 2>/dev/null | sed 's/notAfter=//')
        
        log_info "✓ 服务器证书生成成功"
        log_info "  私钥: $output_dir/$safe_name.key"
        log_info "  证书: $output_dir/$safe_name.crt"
        log_info "  证书链: $output_dir/$safe_name.chain.crt"
        log_info "  通用链接: $output_dir/server.crt -> $safe_name.crt"
        log_info ""
        log_info "  证书主题: $cert_subject"
        log_info "  有效期至: $cert_expire"
        return 0
    else
        log_error "服务器证书生成失败"
        return 1
    fi
}

# 设置 SSL 证书 (生成 CA + 服务器证书)
setup_ssl_certificates() {
    local domain="${1:-}"
    local force="${2:-false}"
    
    # 自动检测域名
    if [[ -z "$domain" ]]; then
        domain="${SSL_DOMAIN:-}"
    fi
    if [[ -z "$domain" ]]; then
        domain="${EXTERNAL_HOST:-}"
    fi
    if [[ -z "$domain" ]]; then
        domain=$(detect_external_host)
    fi
    
    if [[ -z "$domain" ]]; then
        log_error "Unable to determine domain for SSL certificate"
        log_info "Please specify domain: ./build.sh ssl-setup your-domain.com"
        return 1
    fi
    
    log_info "🔒 Setting up SSL certificates for: $domain"
    log_info "   Output directory: $SSL_OUTPUT_DIR"
    
    # 检查 OpenSSL 是否可用
    if ! command -v openssl &> /dev/null; then
        log_error "OpenSSL not found. Please install OpenSSL first."
        return 1
    fi
    
    local openssl_version=$(openssl version 2>/dev/null)
    log_info "   OpenSSL: $openssl_version"
    
    # 安全文件名
    local safe_name=$(echo "$domain" | sed 's/\*/_wildcard_/g')
    
    # 检查证书是否已存在
    if [[ -f "$SSL_OUTPUT_DIR/$safe_name.crt" ]] && [[ "$force" != "true" ]]; then
        log_info "SSL certificates already exist for $domain"
        log_info "Use --force to regenerate: ./build.sh ssl-setup --force"
        return 0
    fi
    
    # 创建输出目录
    mkdir -p "$SSL_OUTPUT_DIR"
    
    # 生成 CA (如果不存在或强制重新生成)
    if [[ ! -f "$SSL_OUTPUT_DIR/ca/ca.crt" ]] || [[ "$force" == "true" ]]; then
        if ! generate_ca_certificate; then
            return 1
        fi
    else
        log_info "使用已存在的 CA 证书"
    fi
    
    # 生成服务器证书
    if ! generate_server_certificate "$domain"; then
        return 1
    fi
    
    # 更新 .env 文件中的 SSL 相关变量
    log_step "Updating .env configuration..."
    update_env_variable "ENABLE_TLS" "true"
    update_env_variable "EXTERNAL_SCHEME" "https"
    update_env_variable "SSL_CERT_DIR" "./src/nginx/ssl"
    
    echo ""
    log_info "═══════════════════════════════════════════════════════════════════"
    log_info "✅ SSL certificates generated successfully!"
    log_info "═══════════════════════════════════════════════════════════════════"
    log_info ""
    log_info "📁 Certificate files (will be bundled into nginx image):"
    log_info "   CA Certificate:     $SSL_OUTPUT_DIR/ca/ca.crt"
    log_info "   Server Certificate: $SSL_OUTPUT_DIR/server.crt"
    log_info "   Server Key:         $SSL_OUTPUT_DIR/server.key"
    log_info ""
    log_info "📋 Next steps:"
    log_info "   1. Rebuild nginx:  ./build.sh nginx"
    log_info "   2. Restart:        docker compose restart nginx"
    log_info ""
    log_info "💡 To trust the CA on client machines, import:"
    log_info "   $SSL_OUTPUT_DIR/ca/ca.crt"
    
    return 0
}

# 使用 Let's Encrypt (certbot) 申请正式证书并放入 SSL_OUTPUT_DIR
setup_letsencrypt_certificates() {
    local domain="${1:-}"
    local email="${2:-${LETSENCRYPT_EMAIL:-}}"
    local staging="${3:-${LETSENCRYPT_STAGING:-false}}"
    local force="${4:-false}"

    # 自动检测域名
    if [[ -z "$domain" ]]; then
        domain="${SSL_DOMAIN:-}"
    fi
    if [[ -z "$domain" ]]; then
        domain="${EXTERNAL_HOST:-}"
    fi
    if [[ -z "$domain" ]]; then
        domain=$(detect_external_host)
    fi

    if [[ -z "$domain" ]]; then
        log_error "Unable to determine domain for Let's Encrypt"
        log_info "Please specify domain: ./build.sh ssl-setup-le your-domain.com"
        return 1
    fi

    if [[ -z "$email" ]]; then
        log_error "Let's Encrypt email is required. Set LETSENCRYPT_EMAIL or provide as second argument."
        return 1
    fi

    if ! command -v certbot >/dev/null 2>&1; then
        log_error "certbot not found. Install certbot (https://letsencrypt.org/getting-started/)."
        return 1
    fi

    local safe_name=$(echo "$domain" | sed 's/\*/_wildcard_/g')

    if [[ -f "$SSL_OUTPUT_DIR/server.crt" ]] && [[ "$force" != "true" ]]; then
        if is_existing_cert_valid "$SSL_OUTPUT_DIR/server.crt"; then
            log_info "Existing certificate is still valid. Use --force to renew."
            return 0
        fi
    fi

    local le_root="$SSL_OUTPUT_DIR/letsencrypt"
    local le_config="$le_root/config"
    local le_work="$le_root/work"
    local le_logs="$le_root/logs"
    mkdir -p "$le_config" "$le_work" "$le_logs" "$SSL_OUTPUT_DIR"

    local staging_flag=""
    [[ "$staging" == "true" ]] && staging_flag="--staging"

    local -a domain_args
    domain_args+=("-d" "$domain")
    if [[ -n "$LETSENCRYPT_EXTRA_DOMAINS" ]]; then
        IFS=',' read -ra extra_domains <<< "$LETSENCRYPT_EXTRA_DOMAINS"
        for extra_domain in "${extra_domains[@]}"; do
            [[ -z "$extra_domain" ]] && continue
            domain_args+=("-d" "$extra_domain")
        done
    fi

    # Check if port 80 is in use and try to stop nginx temporarily
    local nginx_was_running=false
    if command -v lsof >/dev/null 2>&1; then
        if lsof -iTCP:80 -sTCP:LISTEN >/dev/null 2>&1; then
            log_warn "Port 80 is in use. Attempting to stop nginx temporarily..."
            if docker ps --format '{{.Names}}' | grep -q "ai-infra-nginx"; then
                docker stop ai-infra-nginx >/dev/null 2>&1 && nginx_was_running=true
                log_info "Nginx container stopped temporarily"
                sleep 2
            else
                log_warn "Port 80 is in use by non-nginx process. certbot --standalone may fail."
            fi
        fi
    fi

    log_step "Requesting Let's Encrypt certificate for: $domain"
    local certbot_result=0
    if ! certbot certonly --standalone --preferred-challenges http \
        --agree-tos --non-interactive \
        -m "$email" "${domain_args[@]}" \
        --config-dir "$le_config" \
        --work-dir "$le_work" \
        --logs-dir "$le_logs" \
        $staging_flag; then
        certbot_result=1
    fi

    # Restart nginx if it was running before
    if [[ "$nginx_was_running" == "true" ]]; then
        log_info "Restarting nginx container..."
        docker start ai-infra-nginx >/dev/null 2>&1 || true
    fi

    if [[ "$certbot_result" -ne 0 ]]; then
        log_error "Let's Encrypt issuance failed"
        return 1
    fi

    local live_dir="$le_config/live/$domain"
    if [[ ! -f "$live_dir/fullchain.pem" ]] || [[ ! -f "$live_dir/privkey.pem" ]]; then
        log_error "Issued certificate files not found in $live_dir"
        return 1
    fi

    cp "$live_dir/fullchain.pem" "$SSL_OUTPUT_DIR/$safe_name.crt"
    cp "$live_dir/privkey.pem" "$SSL_OUTPUT_DIR/$safe_name.key"
    cp "$live_dir/chain.pem" "$SSL_OUTPUT_DIR/$safe_name.chain.crt" 2>/dev/null || true
    ln -sf "$safe_name.crt" "$SSL_OUTPUT_DIR/server.crt"
    ln -sf "$safe_name.key" "$SSL_OUTPUT_DIR/server.key"
    ln -sf "$safe_name.chain.crt" "$SSL_OUTPUT_DIR/server.chain.crt"
    chmod 600 "$SSL_OUTPUT_DIR/$safe_name.key" "$SSL_OUTPUT_DIR/server.key" 2>/dev/null || true

    update_env_variable "ENABLE_TLS" "true"
    update_env_variable "EXTERNAL_SCHEME" "https"
    update_env_variable "SSL_CERT_DIR" "./src/nginx/ssl"

    log_info "✅ Let's Encrypt certificate ready for $domain"
    log_info "   Email: $email"
    [[ "$staging" == "true" ]] && log_warn "Using staging endpoint; set LETSENCRYPT_STAGING=false for production."
    log_info "   Stored at: $SSL_OUTPUT_DIR"

    return 0
}

# 使用 Cloudflare DNS 验证申请 Let's Encrypt 证书 (支持通配符证书)
# Usage: ./build.sh ssl-cloudflare <domain> [--staging] [--force]
#        ./build.sh ssl-cloudflare ai-infra-matrix.top
#        ./build.sh ssl-cloudflare ai-infra-matrix.top --wildcard
setup_cloudflare_certificates() {
    local domain="${1:-}"
    local wildcard="${2:-false}"
    local staging="${3:-${LETSENCRYPT_STAGING:-false}}"
    local force="${4:-false}"
    local credentials_file="${CLOUDFLARE_CREDENTIALS:-$HOME/.secrets/cloudflare.ini}"

    # 自动检测域名
    if [[ -z "$domain" ]]; then
        domain="${SSL_DOMAIN:-}"
    fi
    if [[ -z "$domain" ]]; then
        domain="${EXTERNAL_HOST:-}"
    fi
    if [[ -z "$domain" ]]; then
        log_error "Unable to determine domain for Cloudflare DNS validation"
        log_info "Please specify domain: ./build.sh ssl-cloudflare your-domain.com"
        return 1
    fi

    # 检查 certbot 是否安装
    if ! command -v certbot >/dev/null 2>&1; then
        log_error "certbot not found. Please install certbot first."
        log_info "  Ubuntu/Debian: apt install certbot python3-certbot-dns-cloudflare"
        log_info "  macOS: brew install certbot"
        return 1
    fi

    # 检查 cloudflare 插件是否安装
    if ! certbot plugins 2>/dev/null | grep -q "dns-cloudflare"; then
        log_error "certbot-dns-cloudflare plugin not found."
        log_info "  Ubuntu/Debian: apt install python3-certbot-dns-cloudflare"
        log_info "  pip: pip install certbot-dns-cloudflare"
        return 1
    fi

    # 检查 Cloudflare 凭证文件
    if [[ ! -f "$credentials_file" ]]; then
        log_error "Cloudflare credentials file not found: $credentials_file"
        log_info ""
        log_info "Please create the credentials file:"
        log_info "  mkdir -p ~/.secrets"
        log_info "  cat > ~/.secrets/cloudflare.ini << 'EOF'"
        log_info "dns_cloudflare_api_token = YOUR_CLOUDFLARE_API_TOKEN"
        log_info "EOF"
        log_info "  chmod 600 ~/.secrets/cloudflare.ini"
        log_info ""
        log_info "Or set CLOUDFLARE_CREDENTIALS environment variable to your credentials file path."
        return 1
    fi

    # 检查凭证文件权限 (certbot 要求 600)
    local file_perms=$(stat -f "%OLp" "$credentials_file" 2>/dev/null || stat -c "%a" "$credentials_file" 2>/dev/null)
    if [[ "$file_perms" != "600" ]]; then
        log_warn "Cloudflare credentials file permissions should be 600 (current: $file_perms)"
        log_info "Fixing permissions..."
        chmod 600 "$credentials_file"
    fi

    local safe_name=$(echo "$domain" | sed 's/\*/_wildcard_/g')

    # 检查现有证书是否有效
    if [[ -f "$SSL_OUTPUT_DIR/server.crt" ]] && [[ "$force" != "true" ]]; then
        if is_existing_cert_valid "$SSL_OUTPUT_DIR/server.crt"; then
            log_info "Existing certificate is still valid. Use --force to renew."
            return 0
        fi
    fi

    # 创建 Let's Encrypt 目录
    local le_root="$SSL_OUTPUT_DIR/letsencrypt"
    local le_config="$le_root/config"
    local le_work="$le_root/work"
    local le_logs="$le_root/logs"
    mkdir -p "$le_config" "$le_work" "$le_logs" "$SSL_OUTPUT_DIR"

    local staging_flag=""
    [[ "$staging" == "true" ]] && staging_flag="--staging"

    # 构建域名参数
    local -a domain_args
    if [[ "$wildcard" == "true" ]]; then
        # 通配符证书：包含根域名和通配符
        domain_args+=("-d" "$domain")
        domain_args+=("-d" "*.$domain")
        log_info "📜 Requesting wildcard certificate for: $domain and *.$domain"
    else
        # 标准证书：包含根域名和 www 子域名
        domain_args+=("-d" "$domain")
        domain_args+=("-d" "www.$domain")
        log_info "📜 Requesting certificate for: $domain and www.$domain"
    fi

    # 添加额外域名
    if [[ -n "$LETSENCRYPT_EXTRA_DOMAINS" ]]; then
        IFS=',' read -ra extra_domains <<< "$LETSENCRYPT_EXTRA_DOMAINS"
        for extra_domain in "${extra_domains[@]}"; do
            [[ -z "$extra_domain" ]] && continue
            domain_args+=("-d" "$extra_domain")
            log_info "  + Extra domain: $extra_domain"
        done
    fi

    log_step "🔐 Requesting Let's Encrypt certificate via Cloudflare DNS..."
    log_info "   Domain: $domain"
    log_info "   Credentials: $credentials_file"
    [[ "$staging" == "true" ]] && log_warn "   Using STAGING endpoint (test certificate)"

    if ! certbot certonly \
        --dns-cloudflare \
        --dns-cloudflare-credentials "$credentials_file" \
        --dns-cloudflare-propagation-seconds 30 \
        --agree-tos --non-interactive \
        -m "${LETSENCRYPT_EMAIL:-admin@$domain}" \
        "${domain_args[@]}" \
        --config-dir "$le_config" \
        --work-dir "$le_work" \
        --logs-dir "$le_logs" \
        $staging_flag; then
        log_error "Let's Encrypt certificate issuance failed"
        log_info "Check logs at: $le_logs"
        return 1
    fi

    # 查找证书目录 (可能是根域名或通配符名称)
    local live_dir="$le_config/live/$domain"
    if [[ ! -d "$live_dir" ]]; then
        # 尝试查找其他可能的目录名
        live_dir=$(find "$le_config/live" -maxdepth 1 -type d -name "${domain}*" | head -1)
    fi

    if [[ ! -f "$live_dir/fullchain.pem" ]] || [[ ! -f "$live_dir/privkey.pem" ]]; then
        log_error "Certificate files not found in $live_dir"
        return 1
    fi

    # 复制证书到 SSL 目录
    cp "$live_dir/fullchain.pem" "$SSL_OUTPUT_DIR/$safe_name.crt"
    cp "$live_dir/privkey.pem" "$SSL_OUTPUT_DIR/$safe_name.key"
    cp "$live_dir/chain.pem" "$SSL_OUTPUT_DIR/$safe_name.chain.crt" 2>/dev/null || true

    # 创建符号链接
    ln -sf "$safe_name.crt" "$SSL_OUTPUT_DIR/server.crt"
    ln -sf "$safe_name.key" "$SSL_OUTPUT_DIR/server.key"
    ln -sf "$safe_name.chain.crt" "$SSL_OUTPUT_DIR/server.chain.crt" 2>/dev/null || true
    chmod 600 "$SSL_OUTPUT_DIR/$safe_name.key" "$SSL_OUTPUT_DIR/server.key" 2>/dev/null || true

    # 更新 .env 配置
    update_env_variable "ENABLE_TLS" "true"
    update_env_variable "EXTERNAL_SCHEME" "https"
    update_env_variable "SSL_CERT_DIR" "./src/nginx/ssl"

    log_info ""
    log_info "═══════════════════════════════════════════════════════════════════"
    log_info "✅ Cloudflare DNS 验证证书申请成功!"
    log_info "═══════════════════════════════════════════════════════════════════"
    log_info "   域名: $domain"
    [[ "$wildcard" == "true" ]] && log_info "   通配符: *.$domain"
    log_info "   证书路径: $SSL_OUTPUT_DIR"
    log_info "   有效期: $(openssl x509 -in "$SSL_OUTPUT_DIR/server.crt" -noout -enddate 2>/dev/null | cut -d= -f2)"
    [[ "$staging" == "true" ]] && log_warn "   ⚠️  这是测试证书，请设置 LETSENCRYPT_STAGING=false 申请正式证书"
    log_info ""
    log_info "📋 下一步:"
    log_info "   1. 重新渲染配置: ./build.sh render"
    log_info "   2. 重启 nginx: docker compose restart nginx"
    log_info "═══════════════════════════════════════════════════════════════════"

    return 0
}

# 显示 SSL 证书信息
show_ssl_info() {
    local domain="${1:-}"
    
    # 自动检测域名
    if [[ -z "$domain" ]]; then
        domain="${EXTERNAL_HOST:-$(detect_external_host)}"
    fi
    
    local safe_name=$(echo "$domain" | sed 's/\*/_wildcard_/g')
    local cert_file="$SSL_OUTPUT_DIR/server.crt"
    local ca_file="$SSL_OUTPUT_DIR/ca/ca.crt"
    
    echo ""
    log_info "═══════════════════════════════════════════════════════════════════"
    log_info "SSL 证书信息"
    log_info "═══════════════════════════════════════════════════════════════════"
    
    if [[ -f "$ca_file" ]]; then
        echo ""
        log_info "CA 证书:"
        openssl x509 -in "$ca_file" -noout -subject -issuer -dates 2>/dev/null | sed 's/^/   /'
    else
        log_warn "CA 证书不存在: $ca_file"
    fi
    
    if [[ -f "$cert_file" ]]; then
        echo ""
        log_info "服务器证书:"
        openssl x509 -in "$cert_file" -noout -subject -issuer -dates 2>/dev/null | sed 's/^/   /'
        echo ""
        log_info "SAN (Subject Alternative Names):"
        openssl x509 -in "$cert_file" -noout -ext subjectAltName 2>/dev/null | sed 's/^/   /'
    else
        log_warn "服务器证书不存在: $cert_file"
        log_info "请运行: ./build.sh ssl-setup"
    fi
    
    echo ""
}

# 清理 SSL 证书
clean_ssl_certificates() {
    log_info "🗑️  Cleaning SSL certificates..."
    
    if [[ -d "$SSL_OUTPUT_DIR" ]]; then
        rm -rf "$SSL_OUTPUT_DIR"
        log_info "Removed: $SSL_OUTPUT_DIR"
    else
        log_info "SSL directory does not exist: $SSL_OUTPUT_DIR"
    fi
    
    # 恢复 .env 中的 SSL 相关设置
    update_env_variable "ENABLE_TLS" "false"
    update_env_variable "EXTERNAL_SCHEME" "http"
    
    log_info "✅ SSL certificates cleaned"
}

detect_compose_command() {
    if command -v docker-compose >/dev/null 2>&1; then
        echo "docker-compose"
    elif docker compose version >/dev/null 2>&1; then
        echo "docker compose"
    else
        return 1
    fi
}

# Check if SLURM deb packages are available in AppHub for specific architecture
# Args: $1 - target architecture (amd64, arm64), defaults to current arch
# Returns 0 if available, 1 if not
check_apphub_slurm_available() {
    local target_arch="${1:-$(dpkg --print-architecture 2>/dev/null || uname -m)}"
    local apphub_port="${APPHUB_PORT:-28080}"
    local external_host="${EXTERNAL_HOST:-$(detect_external_host)}"
    local apphub_url="http://${external_host}:${apphub_port}"
    
    # Normalize architecture name
    case "$target_arch" in
        x86_64|x64) target_arch="amd64" ;;
        aarch64) target_arch="arm64" ;;
    esac
    
    # Check if Packages file exists
    local packages_content
    packages_content=$(curl -sf --connect-timeout 2 --max-time 10 "${apphub_url}/pkgs/slurm-deb/Packages" 2>/dev/null)
    if [[ -z "$packages_content" ]]; then
        log_debug "SLURM Packages file not found in AppHub"
        return 1
    fi
    
    # Check if packages for target architecture exist
    # Look for "Architecture: <target_arch>" in the Packages file
    if echo "$packages_content" | grep -q "^Architecture: ${target_arch}$"; then
        log_debug "Found SLURM packages for $target_arch in AppHub"
        return 0
    else
        log_debug "No SLURM packages for $target_arch in AppHub (found: $(echo "$packages_content" | grep "^Architecture:" | sort -u | tr '\n' ' '))"
        return 1
    fi
}

wait_for_apphub_ready() {
    local timeout="${1:-300}"
    local container_name="ai-infra-apphub"
    local check_interval=5
    local elapsed=0
    
    local apphub_port="${APPHUB_PORT:-28080}"
    local external_host="${EXTERNAL_HOST:-$(detect_external_host)}"
    local apphub_url="http://${external_host}:${apphub_port}"
    
    log_info "Waiting for AppHub at $apphub_url (Timeout: ${timeout}s)..."
    
    while [[ $elapsed -lt $timeout ]]; do
        # Check if container is running
        if ! docker ps --filter "name=$container_name" --filter "status=running" | grep -q "$container_name"; then
            log_warn "[${elapsed}s] Container not running..."
            sleep $check_interval
            elapsed=$((elapsed + check_interval))
            continue
        fi
        
        # Check health endpoint (primary check)
        if curl -sf --connect-timeout 2 --max-time 5 "${apphub_url}/health" >/dev/null 2>&1; then
            # Health check passed, verify nginx is serving content
            if curl -sf --connect-timeout 2 --max-time 5 "${apphub_url}/" >/dev/null 2>&1; then
                log_info "✅ AppHub is ready!"
                # Show available SLURM package architectures
                local packages_content
                packages_content=$(curl -sf --connect-timeout 2 --max-time 5 "${apphub_url}/pkgs/slurm-deb/Packages" 2>/dev/null || true)
                if [[ -n "$packages_content" ]]; then
                    local available_archs
                    available_archs=$(echo "$packages_content" | grep "^Architecture:" | awk '{print $2}' | sort -u | tr '\n' ' ')
                    log_info "📦 SLURM packages available for: ${available_archs:-none}"
                else
                    log_warn "⚠️  Note: SLURM deb packages not available (may have been skipped during build)"
                fi
                return 0
            fi
        fi
        
        log_warn "[${elapsed}s] AppHub not ready yet..."
        sleep $check_interval
        elapsed=$((elapsed + check_interval))
    done
    
    log_error "❌ AppHub failed to become ready."
    return 1
}

# ==============================================================================
# Third Party Version Sync - 同步第三方组件版本
# ==============================================================================

# Sync third_party version files with .env variables
# Updates version.json files and components.json based on current .env settings
sync_third_party_versions() {
    local third_party_dir="$SCRIPT_DIR/third_party"
    local components_json="$third_party_dir/components.json"
    local updated_count=0
    
    log_info "Syncing third_party versions with .env..."
    
    # Check if jq is available
    if ! command -v jq &> /dev/null; then
        log_warn "jq not found, skipping third_party version sync"
        return 0
    fi
    
    # Check if components.json exists
    if [[ ! -f "$components_json" ]]; then
        log_warn "components.json not found at $components_json"
        return 0
    fi
    
    # Get list of components from components.json
    local components=$(jq -r '.components | keys[]' "$components_json" 2>/dev/null)
    
    for component in $components; do
        # Get version_env variable name from components.json
        local version_env=$(jq -r ".components.${component}.version_env // empty" "$components_json")
        local version_prefix=$(jq -r ".components.${component}.version_prefix // \"v\"" "$components_json")
        local default_version=$(jq -r ".components.${component}.default_version // empty" "$components_json")
        
        if [[ -z "$version_env" ]]; then
            continue
        fi
        
        # Get current version from environment (loaded from .env)
        local env_version="${!version_env:-}"
        
        if [[ -z "$env_version" ]]; then
            continue
        fi
        
        # Strip prefix for comparison if present in env_version
        local clean_env_version="${env_version#v}"
        clean_env_version="${clean_env_version#V}"
        
        # Check if version differs from default in components.json
        if [[ "$clean_env_version" != "$default_version" ]]; then
            log_info "  Updating $component: $default_version -> $clean_env_version"
            
            # Update components.json default_version
            local tmp_file=$(mktemp)
            jq ".components.${component}.default_version = \"$clean_env_version\"" "$components_json" > "$tmp_file" && \
                mv "$tmp_file" "$components_json"
            updated_count=$((updated_count + 1))
        fi
        
        # Update version.json if component directory exists
        local component_dir="$third_party_dir/$component"
        local version_json="$component_dir/version.json"
        
        if [[ -d "$component_dir" ]] && [[ -f "$version_json" ]]; then
            local current_version=$(jq -r '.version // empty' "$version_json" 2>/dev/null)
            local current_clean="${current_version#v}"
            current_clean="${current_clean#V}"
            
            if [[ "$current_clean" != "$clean_env_version" ]]; then
                log_info "  Updating $component/version.json: $current_version -> ${version_prefix}${clean_env_version}"
                
                # Update version.json
                cat > "$version_json" << EOF
{
    "component": "${component}",
    "version": "${version_prefix}${clean_env_version}",
    "downloaded_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
                updated_count=$((updated_count + 1))
            fi
        fi
    done
    
    if [[ $updated_count -gt 0 ]]; then
        log_info "  ✓ Updated $updated_count third_party version entries"
    else
        log_info "  ✓ All third_party versions are in sync"
    fi
}

# ==============================================================================
# Third Party Download Functions - 第三方依赖下载功能
# ==============================================================================

# Third party download configuration
THIRD_PARTY_DIR="$SCRIPT_DIR/third_party"
COMPONENTS_JSON="$THIRD_PARTY_DIR/components.json"
APPHUB_DOCKERFILE="$SCRIPT_DIR/src/apphub/Dockerfile"

# Download target architecture (all, amd64, arm64)
DOWNLOAD_TARGET_ARCH="all"
# Specified version for download
DOWNLOAD_SPECIFIED_VERSION=""
# GitHub mirror for download acceleration
DOWNLOAD_GITHUB_MIRROR="${GITHUB_MIRROR:-https://gh-proxy.com/}"

# Component alias mapping (user-friendly names to actual component names)
declare -A COMPONENT_ALIASES=(
    ["vscode"]="code_server"
    ["code-server"]="code_server"
    ["codeserver"]="code_server"
    ["node-exporter"]="node_exporter"
    ["nodeexporter"]="node_exporter"
    ["salt"]="saltstack"
)

# Resolve component alias to actual component name
resolve_component_alias() {
    local input="$1"
    local lower_input=$(echo "$input" | tr '[:upper:]' '[:lower:]')
    
    # Check if it's an alias
    if [[ -n "${COMPONENT_ALIASES[$lower_input]:-}" ]]; then
        echo "${COMPONENT_ALIASES[$lower_input]}"
    else
        echo "$input"
    fi
}

# Get component property from components.json
get_download_component_prop() {
    local component=$1
    local prop=$2
    local default=${3:-}
    
    if command -v jq &> /dev/null && [[ -f "$COMPONENTS_JSON" ]]; then
        local val=$(jq -r ".components.${component}.${prop} // empty" "$COMPONENTS_JSON" 2>/dev/null)
        echo "${val:-$default}"
    else
        echo "$default"
    fi
}

# Get array property from components.json
get_download_component_array() {
    local component=$1
    local prop=$2
    
    if command -v jq &> /dev/null && [[ -f "$COMPONENTS_JSON" ]]; then
        jq -r ".components.${component}.${prop}[]? // empty" "$COMPONENTS_JSON" 2>/dev/null
    fi
}

# Get version from environment or .env file
# Priority: already loaded env > .env file > default
get_download_env_version() {
    local var_name=$1
    local default=$2
    
    # First check already loaded environment variable
    local env_val="${!var_name:-}"
    if [[ -n "$env_val" ]]; then
        echo "$env_val"
        return
    fi
    
    # Then check .env file
    if [[ -f "$ENV_FILE" ]]; then
        local val=$(grep "^${var_name}=" "$ENV_FILE" 2>/dev/null | head -1 | cut -d'=' -f2 | tr -d '"' | tr -d ' ')
        echo "${val:-$default}"
    else
        echo "$default"
    fi
}

# Get ARG value from Dockerfile
get_download_dockerfile_arg() {
    local name=$1
    local default=$2
    
    if [[ -f "$APPHUB_DOCKERFILE" ]]; then
        local val=$(grep "ARG $name=" "$APPHUB_DOCKERFILE" 2>/dev/null | head -1 | cut -d'=' -f2 | tr -d '"' | tr -d ' ')
        echo "${val:-$default}"
    else
        echo "$default"
    fi
}

# Ensure version has correct prefix
download_ensure_prefix() {
    local ver=$1
    local prefix=$2
    
    if [[ -z "$prefix" ]] || [[ "$prefix" = "v" ]]; then
        if [[ ! "$ver" =~ ^v ]]; then
            echo "v${ver}"
        else
            echo "$ver"
        fi
    elif [[ "$prefix" = "munge-" ]]; then
        if [[ ! "$ver" =~ ^munge- ]]; then
            echo "munge-${ver}"
        else
            echo "$ver"
        fi
    elif [[ "$prefix" = "slurm-" ]]; then
        if [[ ! "$ver" =~ ^slurm- ]]; then
            echo "slurm-${ver}"
        else
            echo "$ver"
        fi
    else
        echo "${ver}"
    fi
}

# Strip version prefix
download_strip_prefix() {
    local ver=$1
    ver="${ver#v}"
    ver="${ver#munge-}"
    ver="${ver#slurm-}"
    echo "$ver"
}

# Generic download function with mirror support
download_single_file() {
    local url=$1
    local output_file=$2
    local use_mirror=${3:-true}
    local final_url="$url"
    
    # Apply GitHub mirror (使用完整 URL 拼接方式: ghfast.top/gh-proxy.com 等镜像服务的标准格式)
    if [[ "$use_mirror" = true ]] && [[ "$url" == *"github.com"* ]] && [[ -n "$DOWNLOAD_GITHUB_MIRROR" ]]; then
        final_url="${DOWNLOAD_GITHUB_MIRROR}${url}"
    fi
    
    # Check if file already exists and is non-empty
    if [[ -f "$output_file" ]] && [[ -s "$output_file" ]]; then
        log_info "  ✓ Already exists: $(basename "$output_file")"
        return 0
    fi
    
    # Remove possibly empty file
    [[ -f "$output_file" ]] && rm -f "$output_file"
    
    log_info "  📥 Downloading: $(basename "$output_file")"
    log_info "     URL: $final_url"
    
    # First try mirror (10s timeout)
    if wget -q --show-progress -T 10 -t 2 "$final_url" -O "$output_file" 2>/dev/null; then
        if [[ -s "$output_file" ]]; then
            log_info "  ✓ Download successful: $(basename "$output_file")"
            return 0
        fi
    fi
    
    # If mirror fails, try direct download (30s timeout)
    if [[ "$final_url" != "$url" ]]; then
        log_warn "  ⚠ Mirror download failed, trying direct download..."
        rm -f "$output_file"
        if wget -q --show-progress -T 30 -t 2 "$url" -O "$output_file" 2>/dev/null; then
            if [[ -s "$output_file" ]]; then
                log_info "  ✓ Direct download successful: $(basename "$output_file")"
                return 0
            fi
        fi
    fi
    
    log_error "  ✗ Download failed: $(basename "$output_file")"
    rm -f "$output_file"
    return 1
}

# Generate version.json for downloaded component
generate_download_version_json() {
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

# Download SaltStack packages (DEB + RPM)
download_saltstack_packages() {
    local tag_version=$1
    local file_version=$2
    local output_dir=$3
    
    local packages=()
    while IFS= read -r pkg; do
        [[ -n "$pkg" ]] && packages+=("$pkg")
    done < <(get_download_component_array "saltstack" "packages")
    
    # DEB packages
    log_info ""
    log_info "  📦 Downloading DEB packages..."
    for arch in amd64 arm64; do
        if [[ "$DOWNLOAD_TARGET_ARCH" != "all" ]] && [[ "$arch" != "$DOWNLOAD_TARGET_ARCH" ]]; then
            continue
        fi
        for pkg in "${packages[@]}"; do
            local filename="${pkg}_${file_version}_${arch}.deb"
            local url="https://github.com/saltstack/salt/releases/download/${tag_version}/${filename}"
            download_single_file "$url" "${output_dir}/${filename}" true || true
        done
    done
    
    # RPM packages
    log_info ""
    log_info "  📦 Downloading RPM packages..."
    for arch in amd64 arm64; do
        if [[ "$DOWNLOAD_TARGET_ARCH" != "all" ]] && [[ "$arch" != "$DOWNLOAD_TARGET_ARCH" ]]; then
            continue
        fi
        local rpm_arch="x86_64"
        [[ "$arch" = "arm64" ]] && rpm_arch="aarch64"
        
        for pkg in "${packages[@]}"; do
            # RPM package name without -common suffix
            local rpm_pkg="${pkg/-common/}"
            local filename="${rpm_pkg}-${file_version}-0.${rpm_arch}.rpm"
            local url="https://github.com/saltstack/salt/releases/download/${tag_version}/${filename}"
            download_single_file "$url" "${output_dir}/${filename}" true || true
        done
    done
}

# Download code-server packages (DEB + RPM)
download_code_server_packages() {
    local tag_version=$1
    local file_version=$2
    local output_dir=$3
    local github_repo="coder/code-server"
    
    # DEB packages
    log_info ""
    log_info "  📦 Downloading DEB packages..."
    for arch in amd64 arm64; do
        if [[ "$DOWNLOAD_TARGET_ARCH" != "all" ]] && [[ "$arch" != "$DOWNLOAD_TARGET_ARCH" ]]; then
            continue
        fi
        local filename="code-server_${file_version}_${arch}.deb"
        local url="https://github.com/${github_repo}/releases/download/${tag_version}/${filename}"
        download_single_file "$url" "${output_dir}/${filename}" true || true
    done
    
    # RPM packages
    log_info ""
    log_info "  📦 Downloading RPM packages..."
    for arch in amd64 arm64; do
        if [[ "$DOWNLOAD_TARGET_ARCH" != "all" ]] && [[ "$arch" != "$DOWNLOAD_TARGET_ARCH" ]]; then
            continue
        fi
        local filename="code-server-${file_version}-${arch}.rpm"
        local url="https://github.com/${github_repo}/releases/download/${tag_version}/${filename}"
        download_single_file "$url" "${output_dir}/${filename}" true || true
    done
}

# Download Singularity packages (DEB + RPM + source)
download_singularity_packages() {
    local tag_version=$1
    local file_version=$2
    local output_dir=$3
    local github_repo="sylabs/singularity"
    
    # DEB packages (Ubuntu) - amd64 only
    log_info ""
    log_info "  📦 Downloading DEB packages (Ubuntu)..."
    log_warn "  ⚠️  Note: Singularity only provides amd64 prebuilt packages, ARM64 needs source compilation"
    local ubuntu_codenames=("noble" "jammy")
    for codename in "${ubuntu_codenames[@]}"; do
        if [[ "$DOWNLOAD_TARGET_ARCH" = "arm64" ]]; then
            log_info "  ⏭️  Skipping DEB (arm64): Singularity doesn't provide ARM64 prebuilt packages"
            continue
        fi
        local filename="singularity-ce_${file_version}-${codename}_amd64.deb"
        local url="https://github.com/${github_repo}/releases/download/${tag_version}/${filename}"
        download_single_file "$url" "${output_dir}/${filename}" true || true
    done
    
    # RPM packages (RHEL/CentOS/Rocky) - x86_64 only
    log_info ""
    log_info "  📦 Downloading RPM packages (RHEL/CentOS)..."
    local el_versions=("el8" "el9" "el10")
    for el_ver in "${el_versions[@]}"; do
        if [[ "$DOWNLOAD_TARGET_ARCH" = "arm64" ]]; then
            log_info "  ⏭️  Skipping RPM (aarch64): Singularity doesn't provide ARM64 prebuilt packages"
            continue
        fi
        local filename="singularity-ce-${file_version}-1.${el_ver}.x86_64.rpm"
        local url="https://github.com/${github_repo}/releases/download/${tag_version}/${filename}"
        download_single_file "$url" "${output_dir}/${filename}" true || true
    done
    
    # Source package (for all architectures including ARM64)
    log_info ""
    log_info "  📦 Downloading source package (for all architectures)..."
    local source_filename="singularity-ce-${file_version}.tar.gz"
    local source_url="https://github.com/${github_repo}/releases/download/${tag_version}/${source_filename}"
    download_single_file "$source_url" "${output_dir}/${source_filename}" true || true
}

# Download a single component
download_third_party_component() {
    local component=$1
    
    # Resolve alias
    component=$(resolve_component_alias "$component")
    
    echo ""
    log_info "================================================================"
    
    local name=$(get_download_component_prop "$component" "name" "$component")
    local desc=$(get_download_component_prop "$component" "description" "")
    local github_repo=$(get_download_component_prop "$component" "github_repo")
    local version_env=$(get_download_component_prop "$component" "version_env")
    local default_version=$(get_download_component_prop "$component" "default_version")
    local version_prefix=$(get_download_component_prop "$component" "version_prefix" "v")
    local filename_version_prefix=$(get_download_component_prop "$component" "filename_version_prefix" "")
    local filename_pattern=$(get_download_component_prop "$component" "filename_pattern")
    
    # Check if component exists
    if [[ -z "$name" ]] || [[ "$name" = "null" ]]; then
        log_error "Unknown component: $component"
        log_info "Use 'download --list' to see available components"
        return 1
    fi
    
    # Get version: command line > env var > .env file > Dockerfile > default
    local version=""
    if [[ -n "$DOWNLOAD_SPECIFIED_VERSION" ]]; then
        version="$DOWNLOAD_SPECIFIED_VERSION"
    elif [[ -n "$version_env" ]]; then
        version=$(get_download_env_version "$version_env" "")
        [[ -z "$version" ]] && version=$(get_download_dockerfile_arg "$version_env" "")
    fi
    [[ -z "$version" ]] && version="$default_version"
    
    # Process version prefix
    local tag_version=$(download_ensure_prefix "$version" "$version_prefix")
    local file_version="$version"
    if [[ -n "$filename_version_prefix" ]]; then
        file_version="${filename_version_prefix}$(download_strip_prefix "$version")"
    else
        file_version="$(download_strip_prefix "$version")"
    fi
    
    log_info "📦 $name ($component)"
    [[ -n "$desc" ]] && log_info "   $desc"
    log_info "   Version: $tag_version"
    log_info "   Repository: $github_repo"
    log_info "================================================================"
    
    local output_dir="$THIRD_PARTY_DIR/$component"
    mkdir -p "$output_dir"
    
    # Get architecture list
    local archs=()
    while IFS= read -r arch; do
        [[ -n "$arch" ]] && archs+=("$arch")
    done < <(get_download_component_array "$component" "architectures")
    
    # Default to amd64 and arm64 if no architecture configured
    [[ ${#archs[@]} -eq 0 ]] && archs=("amd64" "arm64")
    
    # Filter architecture
    if [[ "$DOWNLOAD_TARGET_ARCH" != "all" ]]; then
        local filtered_archs=()
        for arch in "${archs[@]}"; do
            if [[ "$arch" = "$DOWNLOAD_TARGET_ARCH" ]] || [[ -z "$arch" ]]; then
                filtered_archs+=("$arch")
            fi
        done
        archs=("${filtered_archs[@]}")
    fi
    
    # Special handling for different components
    case "$component" in
        saltstack)
            download_saltstack_packages "$tag_version" "$file_version" "$output_dir"
            ;;
        code_server)
            download_code_server_packages "$tag_version" "$file_version" "$output_dir"
            ;;
        singularity)
            download_singularity_packages "$tag_version" "$file_version" "$output_dir"
            ;;
        *)
            # Generic download logic
            for arch in "${archs[@]}"; do
                local filename=$(echo "$filename_pattern" | sed "s/{VERSION}/$file_version/g" | sed "s/{ARCH}/$arch/g")
                local url="https://github.com/${github_repo}/releases/download/${tag_version}/${filename}"
                
                download_single_file "$url" "${output_dir}/${filename}" true || true
            done
            ;;
    esac
    
    generate_download_version_json "$output_dir" "$component" "$tag_version"
    echo ""
    return 0
}

# List available components
list_download_components() {
    if [[ ! -f "$COMPONENTS_JSON" ]]; then
        log_error "Configuration file not found: $COMPONENTS_JSON"
        return 1
    fi
    
    echo ""
    echo "Available Components:"
    echo "====================="
    echo ""
    printf "%-17s %s\n" "Component" "Description"
    printf "%-17s %s\n" "-----------------" "--------------------------------------------------"
    
    if command -v jq &> /dev/null; then
        jq -r '.components | to_entries[] | "\(.key)\t\(.value.description)"' "$COMPONENTS_JSON" | \
            while IFS=$'\t' read -r name desc; do
                printf "%-17s %s\n" "$name" "$desc"
            done
    else
        grep -E '"[a-z_]+":' "$COMPONENTS_JSON" | head -20 | sed 's/.*"\([^"]*\)".*/\1/' | grep -v "components"
    fi
    
    echo ""
    echo "Aliases:"
    echo "--------"
    for alias in "${!COMPONENT_ALIASES[@]}"; do
        printf "  %-15s -> %s\n" "$alias" "${COMPONENT_ALIASES[$alias]}"
    done
    echo ""
}

# ==============================================================================
# Go Dependencies Update Functions - Go 依赖更新功能
# ==============================================================================

update_go_dependencies() {
    local projects=("backend" "nightingale")
    local update_all=true
    local project_to_update=""
    local update_mode="--patch"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                echo "Usage: $0 update-go-deps [project] [options]"
                echo ""
                echo "Projects:"
                echo "  backend          Update backend Go dependencies"
                echo "  nightingale      Update Nightingale Go dependencies"
                echo "  all              Update all projects (default)"
                echo ""
                echo "Options:"
                echo "  -h, --help       Show this help message"
                echo "  --patch          Only update patch versions (default)"
                echo "  --minor          Update minor and patch versions"
                echo "  --tidy           Only run go mod tidy (no upgrades)"
                echo ""
                echo "Examples:"
                echo "  $0 update-go-deps              # Update all projects (patch only)"
                echo "  $0 update-go-deps backend      # Update backend only"
                echo "  $0 update-go-deps --tidy       # Only tidy all projects"
                return 0
                ;;
            --patch|--minor|--tidy)
                update_mode="$1"
                shift
                ;;
            backend|nightingale|all)
                project_to_update="$1"
                update_all=false
                shift
                ;;
            *)
                log_error "Unknown option: $1"
                return 1
                ;;
        esac
    done
    
    export GOPROXY="https://goproxy.cn,https://goproxy.io,direct"
    
    local success=0
    local failed=0
    
    echo ""
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║          Go Dependencies Update Tool                           ║"
    echo "╚════════════════════════════════════════════════════════════════╝"
    echo ""
    log_info "GOPROXY: $GOPROXY"
    log_info "Update Mode: $update_mode"
    echo ""
    
    # Determine which projects to update
    if [[ "$update_all" == true ]] || [[ "$project_to_update" == "all" ]]; then
        # Update all projects
        for project in "${projects[@]}"; do
            _update_go_project "$project" "$update_mode" || failed=$((failed + 1))
            success=$((success + 1))
        done
    else
        # Update specific project
        if [[ " ${projects[*]} " =~ " ${project_to_update} " ]]; then
            _update_go_project "$project_to_update" "$update_mode" || failed=$((failed + 1))
            success=$((success + 1))
        else
            log_error "Unknown project: $project_to_update"
            log_info "Valid projects: ${projects[*]}"
            return 1
        fi
    fi
    
    echo ""
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║                   Update Complete                              ║"
    echo "╚════════════════════════════════════════════════════════════════╝"
    echo ""
    log_info "Processed: $success project(s)"
    [[ $failed -gt 0 ]] && log_error "Failed: $failed project(s)" || log_info "All projects updated successfully ✓"
    echo ""
    
    return $([[ $failed -eq 0 ]] && echo 0 || echo 1)
}

_update_go_project() {
    local project="$1"
    local update_mode="$2"
    local project_path=""
    
    case "$project" in
        backend)
            project_path="$SCRIPT_DIR/src/backend"
            ;;
        nightingale)
            project_path="$SCRIPT_DIR/third_party/nightingale"
            ;;
        *)
            log_error "Unknown project: $project"
            return 1
            ;;
    esac
    
    if [[ ! -f "$project_path/go.mod" ]]; then
        log_error "go.mod not found: $project_path/go.mod"
        return 1
    fi
    
    log_step "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_step "Updating: $project"
    log_step "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    cd "$project_path" || return 1
    
    # Step 1: go mod tidy
    log_info "[1/3] Running go mod tidy..."
    if ! timeout 120 go mod tidy 2>&1 | head -20; then
        log_error "Failed to run go mod tidy"
        cd "$SCRIPT_DIR"
        return 1
    fi
    
    # Step 2: go mod verify
    log_info "[2/3] Verifying go.mod..."
    if ! go mod verify 2>&1 | head -10; then
        log_error "go mod verify failed"
        cd "$SCRIPT_DIR"
        return 1
    fi
    
    # Step 3: Update dependencies (based on mode)
    case "$update_mode" in
        --tidy)
            log_info "[3/3] Skipping dependency updates (--tidy mode)"
            ;;
        --patch)
            log_info "[3/3] Updating patch versions (go get -u=patch)..."
            if ! timeout 300 go get -u=patch ./... 2>&1 | head -30; then
                log_warn "Some patch updates may have failed, but continuing..."
            fi
            ;;
        --minor)
            log_info "[3/3] Updating minor and patch versions (go get -u)..."
            if ! timeout 300 go get -u ./... 2>&1 | head -30; then
                log_warn "Some updates may have failed, but continuing..."
            fi
            ;;
    esac
    
    log_info "✓ $project updated successfully"
    echo ""
    
    cd "$SCRIPT_DIR" || return 1
    return 0
}

# Get all component names from components.json
get_all_download_components() {
    if command -v jq &> /dev/null && [[ -f "$COMPONENTS_JSON" ]]; then
        jq -r '.components | keys[]' "$COMPONENTS_JSON" 2>/dev/null
    else
        grep -E '^\s+"[a-z_]+":' "$COMPONENTS_JSON" | sed 's/.*"\([^"]*\)".*/\1/' | grep -v "components"
    fi
}

# Main download function
# Usage: download_third_party [options] [component...]
# Options:
#   --list, -l           List available components
#   --version VER, -v    Specify version
#   --arch ARCH, -a      Specify architecture (amd64, arm64, all)
#   --mirror URL, -m     Set GitHub mirror URL
#   --no-mirror          Disable GitHub mirror
download_third_party() {
    local components_to_download=()
    
    # Check jq availability
    if ! command -v jq &> /dev/null; then
        log_warn "jq not installed, some features may be limited"
        log_info "Install with: brew install jq (macOS) or apt install jq (Linux)"
        echo ""
    fi
    
    # Check configuration file
    if [[ ! -f "$COMPONENTS_JSON" ]]; then
        log_error "Configuration file not found: $COMPONENTS_JSON"
        return 1
    fi
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                echo "Usage: $0 download [options] [component...]"
                echo ""
                echo "Options:"
                echo "  -h, --help          Show this help"
                echo "  -l, --list          List available components"
                echo "  -v, --version VER   Specify component version"
                echo "  -a, --arch ARCH     Specify architecture (amd64, arm64, all)"
                echo "  -m, --mirror URL    Set GitHub mirror URL"
                echo "  --no-mirror         Disable GitHub mirror"
                echo ""
                echo "Examples:"
                echo "  $0 download                         # Download all components"
                echo "  $0 download prometheus              # Download Prometheus only"
                echo "  $0 download vscode                  # Download VS Code Server (alias)"
                echo "  $0 download -v 4.107.0 vscode       # Download specific version"
                echo "  $0 download --arch amd64 prometheus # Download amd64 only"
                echo "  $0 download --no-mirror prometheus  # Download without mirror"
                return 0
                ;;
            -l|--list)
                list_download_components
                return 0
                ;;
            -v|--version)
                DOWNLOAD_SPECIFIED_VERSION="$2"
                shift 2
                ;;
            -a|--arch)
                DOWNLOAD_TARGET_ARCH="$2"
                shift 2
                ;;
            -m|--mirror)
                DOWNLOAD_GITHUB_MIRROR="$2"
                shift 2
                ;;
            --no-mirror)
                DOWNLOAD_GITHUB_MIRROR=""
                shift
                ;;
            -*)
                log_error "Unknown option: $1"
                return 1
                ;;
            *)
                components_to_download+=("$1")
                shift
                ;;
        esac
    done
    
    # If no components specified, download all
    if [[ ${#components_to_download[@]} -eq 0 ]]; then
        while IFS= read -r comp; do
            [[ -n "$comp" ]] && components_to_download+=("$comp")
        done < <(get_all_download_components)
    fi
    
    echo ""
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║          Third-Party Dependencies Downloader                   ║"
    echo "╚════════════════════════════════════════════════════════════════╝"
    echo ""
    log_info "GitHub Mirror: ${DOWNLOAD_GITHUB_MIRROR:-<disabled>}"
    log_info "Target Arch:   ${DOWNLOAD_TARGET_ARCH}"
    log_info "Output Dir:    ${THIRD_PARTY_DIR}"
    log_info "Components:    ${#components_to_download[@]}"
    echo ""
    
    mkdir -p "$THIRD_PARTY_DIR"
    
    local success=0
    local failed=0
    
    for component in "${components_to_download[@]}"; do
        if download_third_party_component "$component"; then
            success=$((success + 1))
        else
            failed=$((failed + 1))
        fi
    done
    
    echo ""
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║                      Download Complete                         ║"
    echo "╚════════════════════════════════════════════════════════════════╝"
    echo ""
    log_info "Success: $success / Total: $((success + failed))"
    echo ""
    log_info "Files location: $THIRD_PARTY_DIR"
    echo ""
    ls -la "$THIRD_PARTY_DIR"
    
    return 0
}

# ==============================================================================
# Template Rendering Functions - 模板渲染功能
# ==============================================================================

# Define variables that need to be rendered in templates
# These are read from .env and used to replace {{VARIABLE}} placeholders
# 
# IMPORTANT: Variables are divided into two categories:
# 1. BUILD-TIME variables (Dockerfile.tpl) - Used during docker build
# 2. RUNTIME variables (docker-compose.yml.tpl, config templates) - Used at container startup
#
# Build-time variables are baked into the image and cannot be changed at runtime
# Runtime variables can be overridden via environment when starting containers
TEMPLATE_VARIABLES=(
    # ===========================================
    # Mirror configurations (Build-time)
    # Used in Dockerfile.tpl for package downloads during build
    # ===========================================
    "GITHUB_MIRROR"      # GitHub download accelerator (e.g., https://ghfast.top/)
    "APT_MIRROR"         # APT mirror for Debian/Ubuntu (e.g., mirrors.aliyun.com)
    "YUM_MIRROR"         # YUM mirror for AlmaLinux/CentOS (e.g., mirrors.aliyun.com)
    "ALPINE_MIRROR"      # Alpine mirror (e.g., mirrors.aliyun.com)
    "GO_PROXY"           # Go module proxy (e.g., https://goproxy.cn,direct)
    "PYPI_INDEX_URL"     # PyPI mirror (e.g., https://mirrors.aliyun.com/pypi/simple/)
    "NPM_REGISTRY"       # npm registry mirror (e.g., https://registry.npmmirror.com)
    "INTERNAL_FILE_SERVER"  # Internal file server for intranet (e.g., http://192.168.1.100:8080/packages)
    
    # ===========================================
    # Base image versions (Build-time)
    # Used in Dockerfile.tpl FROM statements
    # ===========================================
    "UBUNTU_VERSION"              # Ubuntu base image (e.g., 22.04)
    "ALMALINUX_VERSION"           # AlmaLinux version (e.g., 9.3-minimal)
    "ALPINE_VERSION"              # Alpine version (e.g., 3.22)
    "NGINX_VERSION"               # Nginx version (e.g., stable-alpine-perl)
    "NGINX_ALPINE_VERSION"        # Nginx Alpine version (e.g., 1.27-alpine)
    "PYTHON_VERSION"              # Python version (e.g., 3.14)
    "PYTHON_ALPINE_VERSION"       # Python Alpine version (e.g., 3.14-alpine)
    "NODE_VERSION"                # Node.js major version (e.g., 22)
    "NODE_ALPINE_VERSION"         # Node.js Alpine version (e.g., 22-alpine)
    "NODE_BOOKWORM_VERSION"       # Node.js Bookworm version (e.g., 22-bookworm)
    "NODE_JS_VERSION"             # Node.js full version for prebuilt binaries (e.g., 22.11.0)
    "NODE_IMAGE_VERSION"          # Node.js image version for build (e.g., 22-bookworm)
    "GOLANG_VERSION"              # Go version (e.g., 1.25)
    "GOLANG_IMAGE_VERSION"        # Go image version (e.g., 1.25-bookworm)
    "JUPYTER_BASE_NOTEBOOK_VERSION"  # Jupyter base notebook version (e.g., latest)
    
    # ===========================================
    # Full base image names (for private registry support)
    # 完整基础镜像名称 (支持内网私有仓库)
    # Internet: golang:1.25-bookworm
    # Intranet: harbor.example.com/library/golang:1.25-bookworm
    # ===========================================
    "GOLANG_IMAGE"                # Full golang image name (e.g., golang:1.25-bookworm)
    "UBUNTU_IMAGE"                # Full ubuntu image name (e.g., ubuntu:22.04)
    "ALMALINUX_IMAGE"             # Full almalinux image name (e.g., almalinux:9.3-minimal)
    "NODE_IMAGE"                  # Full node image name for build (e.g., node:22-bookworm)
    "NODE_ALPINE_IMAGE"           # Full node alpine image name (e.g., node:22-alpine)
    "NODE_BOOKWORM_IMAGE"         # Full node bookworm image name (e.g., node:22-bookworm)
    "JUPYTER_BASE_IMAGE"          # Full jupyter base image name (e.g., jupyter/base-notebook:latest)
    "GITEA_IMAGE"                 # Full gitea image name (e.g., gitea/gitea:1.25.3)
    
    # ===========================================
    # Component/Application versions (Build-time)
    # Used in Dockerfile.tpl for building specific components
    # ===========================================
    "SLURM_VERSION"       # SLURM version (e.g., 24.11.5)
    "SALTSTACK_VERSION"   # SaltStack version (e.g., 3007.8)
    "CATEGRAF_VERSION"    # Categraf version (e.g., 0.4.6)
    "NODE_EXPORTER_VERSION" # Node Exporter version (e.g., v1.8.2)
    "SINGULARITY_VERSION" # Singularity version
    "GITEA_VERSION"       # Gitea version (e.g., 1.25.3)
    "JUPYTERHUB_VERSION"  # JupyterHub version (e.g., 5.3.*)
    "PIP_VERSION"         # pip version (e.g., 24.2)
    "N9E_FE_VERSION"      # Nightingale frontend version (e.g., v7.7.2, empty for auto-detect)
    "CODE_SERVER_VERSION" # Code Server version (e.g., 4.96.4)
    "GITHUB_PROXY"        # GitHub proxy for downloading packages (e.g., http://192.168.0.200:7890)
    
    # ===========================================
    # Project settings (Build-time & Runtime)
    # ===========================================
    "IMAGE_TAG"           # Docker image tag (e.g., v0.3.8)
    "TZ"                  # Timezone (e.g., Asia/Shanghai)
    
    # ===========================================
    # Network configuration (Runtime)
    # ===========================================
    "BIND_HOST"           # Host IP for port binding (default: 0.0.0.0, auto-detected)
    
    # ===========================================
    # Nginx configuration variables (Runtime)
    # Used in src/nginx/templates/*.conf.tpl
    # ===========================================
    "EXTERNAL_HOST"       # External host IP/domain (for URLs, not port binding)
    "EXTERNAL_SCHEME"     # http or https
    "FRONTEND_HOST"       # Frontend service host (default: frontend)
    "FRONTEND_PORT"       # Frontend service port (default: 3000)
    "BACKEND_HOST"        # Backend service host (default: backend)
    "BACKEND_PORT"        # Backend service port (default: 8082)
    "JUPYTERHUB_HOST"     # JupyterHub service host (default: jupyterhub)
    "JUPYTERHUB_PORT"     # JupyterHub service port (default: 8000)
    "NIGHTINGALE_HOST"    # Nightingale service host (default: nightingale)
    "NIGHTINGALE_PORT"    # Nightingale service port (default: 17000)
    "EXTERNAL_PORT"       # Main Nginx port (default: 8080)
    "HTTPS_PORT"          # HTTPS port (default: 8443)
    "EXTERNAL_HOST"       # External host for CSP headers
    
    # ===========================================
    # Third-party image versions (for docker-compose.yml.tpl)
    # ===========================================
    "POSTGRES_VERSION"    # PostgreSQL version (e.g., 15-alpine)
    "MYSQL_VERSION"       # MySQL version (e.g., 8.0)
    "REDIS_VERSION"       # Redis version (e.g., 7-alpine)
    "KAFKA_VERSION"       # Kafka version (e.g., 7.5.0)
    "KAFKA_UI_VERSION"    # Kafka UI version (e.g., latest)
    "OPENLDAP_VERSION"    # OpenLDAP version (e.g., stable)
    "PHPLDAPADMIN_VERSION" # phpLDAPadmin version (e.g., stable)
    "SEAWEEDFS_IMAGE"     # SeaweedFS image name (e.g., chrislusf/seaweedfs)
    "SEAWEEDFS_VERSION"   # SeaweedFS version (e.g., latest)
    "SEAWEEDFS_ACCESS_KEY"  # SeaweedFS S3 admin access key
    "SEAWEEDFS_SECRET_KEY"  # SeaweedFS S3 admin secret key
    "SEAWEEDFS_APP_ACCESS_KEY"  # SeaweedFS S3 app access key
    "SEAWEEDFS_APP_SECRET_KEY"  # SeaweedFS S3 app secret key
    "SEAWEEDFS_READONLY_ACCESS_KEY"  # SeaweedFS S3 readonly access key
    "SEAWEEDFS_READONLY_SECRET_KEY"  # SeaweedFS S3 readonly secret key
    "OCEANBASE_VERSION"   # OceanBase version (e.g., 4.3.5-lts)
    "PROMETHEUS_VERSION"  # Prometheus version (e.g., latest)
    "VICTORIAMETRICS_VERSION" # VictoriaMetrics version (e.g., v1.115.0)
    "GRAFANA_VERSION"     # Grafana version (e.g., latest)
    "ALERTMANAGER_VERSION" # AlertManager version (e.g., latest)
    "REDISINSIGHT_VERSION" # RedisInsight version (e.g., latest)
    
    # ===========================================
    # Gitea SSO configuration (Runtime)
    # Used in src/nginx/templates/conf.d/includes/gitea.conf.tpl
    # ===========================================
    "GITEA_ALIAS_ADMIN_TO"  # SSO admin user mapping for Gitea (default: admin)
    "GITEA_ADMIN_EMAIL"     # SSO admin email for Gitea (default: admin@example.com)
    
    # ===========================================
    # SaltStack configuration (Runtime)
    # Used for external node minion installation
    # ===========================================
    "SALT_MASTER_HOST"    # Salt Master host for container internal (e.g., saltstack)
    "SALT_MASTER_PORT"    # Salt Master publish port (e.g., 4505)
    "SALT_RETURN_PORT"    # Salt Master return port (e.g., 4506)
    "SALT_API_PORT"       # Salt API port (e.g., 8002)
    
    # ===========================================
    # AppHub configuration (Runtime)
    # ===========================================
    "APPHUB_PORT"         # AppHub port for package download (e.g., 28080)
    
    # ===========================================
    # Cgroup configuration (Runtime - auto-detected)
    # These variables are auto-detected and set during template rendering
    # ===========================================
    "CGROUP_VERSION"      # Cgroup version: v1 or v2 (auto-detected)
    "CGROUP_MOUNT"        # Cgroup mount path for docker-compose volumes
    "SAFELINE_IMAGE_PREFIX" # SafeLine image prefix (e.g., chaitin)
    "SAFELINE_IMAGE_TAG"    # SafeLine image tag (e.g., latest)
    "SAFELINE_ARCH_SUFFIX"  # SafeLine architecture suffix (-arm for ARM, empty for x86_64)
    "SAFELINE_REGION"       # SafeLine region suffix (optional)
    
    # ===========================================
    # Docker platform configuration (Runtime - auto-detected)
    # For multi-arch image support (ARM/AMD64)
    # ===========================================
    "DOCKER_HOST_PLATFORM"  # Docker host platform (linux/amd64 or linux/arm64, auto-detected)
)

# Render a single template file
# Args: $1 = template file (.tpl), $2 = output file (optional, defaults to removing .tpl extension)
render_template() {
    local template_file="$1"
    local output_file="${2:-${template_file%.tpl}}"
    
    if [[ ! -f "$template_file" ]]; then
        log_error "Template file not found: $template_file"
        return 1
    fi
    
    log_info "Rendering: $template_file -> $output_file"
    
    # Copy template to output file first
    cp "$template_file" "$output_file"
    
    # Replace each {{VARIABLE}} with its value from environment
    # Use perl for reliability across different platforms
    for var in "${TEMPLATE_VARIABLES[@]}"; do
        local value="${!var}"
        if [[ -n "$value" ]]; then
            # Use perl with proper escaping - escape only regex special chars in pattern, not in replacement
            # The replacement value needs & and \ escaped for perl's s/// operator
            local escaped_value
            escaped_value=$(printf '%s' "$value" | sed -e 's/\\/\\\\/g' -e 's/&/\\&/g' -e 's/\//\\\//g')
            perl -i -pe "s/\\{\\{${var}\\}\\}/${escaped_value}/g" "$output_file" 2>/dev/null || {
                # Fallback to sed if perl is not available
                escaped_value=$(printf '%s' "$value" | sed 's/[&/\]/\\&/g')
                sed -i.bak "s|{{${var}}}|${escaped_value}|g" "$output_file" 2>/dev/null || \
                sed -i '' "s|{{${var}}}|${escaped_value}|g" "$output_file"
                rm -f "${output_file}.bak" 2>/dev/null
            }
        else
            # If variable is empty, replace with empty string
            perl -i -pe "s/\\{\\{${var}\\}\\}//g" "$output_file" 2>/dev/null || {
                sed -i.bak "s|{{${var}}}||g" "$output_file" 2>/dev/null || \
                sed -i '' "s|{{${var}}}||g" "$output_file"
                rm -f "${output_file}.bak" 2>/dev/null
            }
        fi
    done
    
    # Check for any remaining unreplaced placeholders
    local remaining
    remaining=$(grep -o '{{[A-Z_]*}}' "$output_file" 2>/dev/null | sort -u | head -5)
    if [[ -n "$remaining" ]]; then
        log_warn "  ⚠️  Unreplaced placeholders found: $remaining"
    fi
    
    log_info "  ✓ Rendered successfully"
    return 0
}

# Render all Dockerfile.tpl files in src/*/ and docker-compose.yml.tpl
render_all_templates() {
    local force="${1:-false}"
    
    log_info "=========================================="
    log_info "🔧 Rendering templates"
    log_info "=========================================="
    
    # Step 1: Sync .env with .env.example (add missing variables)
    log_info "Step 1: Syncing .env with .env.example..."
    sync_env_with_example
    
    # Reload .env after sync
    set -a
    source "$ENV_FILE"
    set +a
    
    # Step 1.5: Auto-detect cgroup version and set environment variables
    log_info ""
    log_info "Step 1.5: Detecting cgroup version..."
    local detected_cgroup_version=$(detect_cgroup_version)
    local detected_cgroup_mount=$(get_cgroup_mount "$detected_cgroup_version")
    
    # Export cgroup variables for template rendering
    export CGROUP_VERSION="$detected_cgroup_version"
    export CGROUP_MOUNT="$detected_cgroup_mount"
    
    log_info "  CGROUP_VERSION=$CGROUP_VERSION"
    log_info "  CGROUP_MOUNT=$CGROUP_MOUNT"
    
    # Step 1.6: Auto-detect CPU architecture for SafeLine
    log_info ""
    log_info "Step 1.6: Detecting CPU architecture for SafeLine..."
    local arch=$(uname -m)
    if [[ "$arch" =~ "aarch" || "$arch" =~ "arm" ]]; then
        export SAFELINE_ARCH_SUFFIX="-arm"
    else
        export SAFELINE_ARCH_SUFFIX=""
    fi
    log_info "  CPU Architecture: $arch"
    log_info "  SAFELINE_ARCH_SUFFIX=${SAFELINE_ARCH_SUFFIX:-<empty>}"
    
    # Step 1.7: Validate and auto-fix EXTERNAL_HOST for port binding
    log_info ""
    log_info "Step 1.7: Validating network configuration..."
    
    # Check if EXTERNAL_HOST is a Docker internal IP (172.x.x.x, 10.x.x.x in Docker range)
    if [[ "$EXTERNAL_HOST" =~ ^172\.(1[6-9]|2[0-9]|3[0-1])\. ]] || \
       [[ "$EXTERNAL_HOST" =~ ^10\.0\. ]]; then
        log_warn "  ⚠️  EXTERNAL_HOST='$EXTERNAL_HOST' appears to be a Docker internal IP!"
        log_warn "  ⚠️  This will cause services to be inaccessible from outside Docker network."
        log_warn "  ⚠️  Consider setting EXTERNAL_HOST to your server's public/private IP, domain name, or '0.0.0.0'"
        
        # Auto-set BIND_HOST to 0.0.0.0 for port binding if not explicitly set
        if [[ -z "$BIND_HOST" ]]; then
            export BIND_HOST="0.0.0.0"
            log_info "  ℹ️  Auto-setting BIND_HOST=0.0.0.0 for port binding"
        fi
    fi
    
    # Set BIND_HOST default to 0.0.0.0 if not set (for universal access)
    export BIND_HOST="${BIND_HOST:-0.0.0.0}"
    log_info "  EXTERNAL_HOST=${EXTERNAL_HOST:-<empty>}"
    log_info "  BIND_HOST=${BIND_HOST}"
    
    # Step 1.8: Check SSL/domain configuration for cloud deployments
    log_info ""
    log_info "Step 1.8: Checking SSL/domain configuration..."
    
    local external_host_type="unknown"
    if is_valid_domain "$EXTERNAL_HOST"; then
        external_host_type="domain"
        log_info "  ✅ EXTERNAL_HOST='$EXTERNAL_HOST' is a valid domain name"
        
        # Check if SSL cert matches the domain
        if [[ "${ENABLE_TLS:-false}" == "true" ]]; then
            if check_ssl_cert_domain_match "$EXTERNAL_HOST"; then
                log_info "  ✅ SSL certificate matches EXTERNAL_HOST domain"
            else
                log_warn "  ⚠️  SSL certificate may not match EXTERNAL_HOST domain!"
                log_warn "  ⚠️  This can cause SSL handshake failures (Error 525) with Cloudflare/proxies"
                log_info ""
                log_info "  💡 Recommended actions:"
                log_info "     1. Generate certificate with correct domain:"
                log_info "        certbot certonly --dns-cloudflare \\"
                log_info "          --dns-cloudflare-credentials ~/.secrets/cloudflare.ini \\"
                log_info "          -d $EXTERNAL_HOST -d www.$EXTERNAL_HOST \\"
                log_info "          --cert-name $EXTERNAL_HOST --force-renewal"
                log_info "     2. Copy certs to nginx ssl dir and restart"
            fi
        fi
    elif is_private_ip "$EXTERNAL_HOST"; then
        external_host_type="private_ip"
        log_warn "  ⚠️  EXTERNAL_HOST='$EXTERNAL_HOST' is a private IP address"
        log_info ""
        log_info "  💡 For public cloud deployments with domain access:"
        log_info "     1. Configure DNS to point your domain to the server's public IP"
        log_info "     2. Update .env:"
        log_info "        EXTERNAL_HOST=your-domain.com"
        log_info "        SSL_DOMAIN=your-domain.com"
        log_info "        LETSENCRYPT_EMAIL=admin@your-domain.com"
        log_info "     3. Generate Let's Encrypt certificate:"
        log_info "        ./build.sh ssl-setup-le your-domain.com admin@your-domain.com"
    elif [[ "$EXTERNAL_HOST" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        external_host_type="public_ip"
        log_info "  ℹ️  EXTERNAL_HOST='$EXTERNAL_HOST' is a public IP address"
        log_info "     Self-signed certificates can be used for IP-based access"
    else
        log_info "  ℹ️  EXTERNAL_HOST='$EXTERNAL_HOST'"
    fi
    
    log_info ""
    log_info "Step 2: Rendering template files..."
    log_info "Source: .env / .env.example"
    log_info "Pattern: src/*/Dockerfile.tpl, docker-compose.yml.tpl"
    echo
    
    # Show key variables being used
    log_info "Template variables:"
    log_info "  GITHUB_MIRROR=${GITHUB_MIRROR:-<empty>}"
    log_info "  APT_MIRROR=${APT_MIRROR:-<empty>}"
    log_info "  YUM_MIRROR=${YUM_MIRROR:-<empty>}"
    log_info "  ALPINE_MIRROR=${ALPINE_MIRROR:-<empty>}"
    log_info "  UBUNTU_VERSION=${UBUNTU_VERSION:-<empty>}"
    log_info "  SLURM_VERSION=${SLURM_VERSION:-<empty>}"
    log_info "  SALTSTACK_VERSION=${SALTSTACK_VERSION:-<empty>}"
    log_info "  CATEGRAF_VERSION=${CATEGRAF_VERSION:-<empty>}"
    log_info "  IMAGE_TAG=${IMAGE_TAG:-<empty>}"
    echo
    
    local success_count=0
    local fail_count=0
    local skip_count=0
    
    # Render docker-compose.yml.tpl if exists
    local compose_tpl="${SCRIPT_DIR}/docker-compose.yml.tpl"
    if [[ -f "$compose_tpl" ]]; then
        local compose_output="${SCRIPT_DIR}/docker-compose.yml"
        
        # Check if output file exists and is newer than template
        if [[ "$force" != "true" ]] && [[ -f "$compose_output" ]]; then
            if [[ "$compose_output" -nt "$compose_tpl" ]] && [[ "$compose_output" -nt "$ENV_FILE" ]]; then
                log_info "Skipping docker-compose.yml (up to date)"
                skip_count=$((skip_count + 1))
            else
                if render_template "$compose_tpl" "$compose_output"; then
                    success_count=$((success_count + 1))
                else
                    fail_count=$((fail_count + 1))
                fi
            fi
        else
            if render_template "$compose_tpl" "$compose_output"; then
                success_count=$((success_count + 1))
            else
                fail_count=$((fail_count + 1))
            fi
        fi
    fi
    
    # Find all Dockerfile.tpl files
    while IFS= read -r -d '' template_file; do
        local output_file="${template_file%.tpl}"
        local component_name=$(basename "$(dirname "$template_file")")
        
        # Check if output file exists and is newer than template
        if [[ "$force" != "true" ]] && [[ -f "$output_file" ]]; then
            if [[ "$output_file" -nt "$template_file" ]] && [[ "$output_file" -nt "$ENV_FILE" ]]; then
                log_info "Skipping $component_name (up to date)"
                skip_count=$((skip_count + 1))
                continue
            fi
        fi
        
        if render_template "$template_file" "$output_file"; then
            success_count=$((success_count + 1))
        else
            fail_count=$((fail_count + 1))
        fi
    done < <(find "$SRC_DIR" -name "Dockerfile.tpl" -print0 2>/dev/null)
    
    # ===========================================
    # Render dependency.conf.tpl files (external image versions)
    # ===========================================
    log_info "Rendering dependency configuration templates..."
    while IFS= read -r -d '' template_file; do
        local output_file="${template_file%.tpl}"
        local component_name=$(basename "$(dirname "$template_file")")
        
        # Check if output file exists and is newer than template
        if [[ "$force" != "true" ]] && [[ -f "$output_file" ]]; then
            if [[ "$output_file" -nt "$template_file" ]] && [[ "$output_file" -nt "$ENV_FILE" ]]; then
                log_info "Skipping $component_name/dependency.conf (up to date)"
                skip_count=$((skip_count + 1))
                continue
            fi
        fi
        
        if render_template "$template_file" "$output_file"; then
            success_count=$((success_count + 1))
        else
            fail_count=$((fail_count + 1))
        fi
    done < <(find "$SRC_DIR" -name "dependency.conf.tpl" -print0 2>/dev/null)
    
    # ===========================================
    # Render Nginx configuration templates
    # ===========================================
    local nginx_template_dir="${SCRIPT_DIR}/src/nginx/templates"
    local nginx_output_dir="${SCRIPT_DIR}/src/nginx"
    
    if [[ -d "$nginx_template_dir" ]]; then
        log_info "Rendering Nginx configuration templates..."
        
        # Render main server config (HTTP)
        local main_conf_tpl="$nginx_template_dir/conf.d/server-main.conf.tpl"
        if [[ -f "$main_conf_tpl" ]]; then
            local main_conf_out="$nginx_output_dir/conf.d/server-main.conf"
            mkdir -p "$(dirname "$main_conf_out")"
            if render_template "$main_conf_tpl" "$main_conf_out"; then
                success_count=$((success_count + 1))
            else
                fail_count=$((fail_count + 1))
            fi
        fi
        
        # Render TLS server config (HTTPS)
        local tls_conf_tpl="$nginx_template_dir/conf.d/server-main-tls.conf.tpl"
        if [[ -f "$tls_conf_tpl" ]]; then
            local tls_conf_out="$nginx_output_dir/conf.d/server-main-tls.conf"
            mkdir -p "$(dirname "$tls_conf_out")"
            if render_template "$tls_conf_tpl" "$tls_conf_out"; then
                success_count=$((success_count + 1))
                log_info "  ✓ Rendered server-main-tls.conf (for HTTPS mode)"
            else
                fail_count=$((fail_count + 1))
            fi
        fi
        
        # Render includes configs
        local includes_dir="$nginx_template_dir/conf.d/includes"
        if [[ -d "$includes_dir" ]]; then
            mkdir -p "$nginx_output_dir/conf.d/includes"
            while IFS= read -r -d '' tpl_file; do
                local out_file="$nginx_output_dir/conf.d/includes/$(basename "${tpl_file%.tpl}")"
                if render_template "$tpl_file" "$out_file"; then
                    success_count=$((success_count + 1))
                else
                    fail_count=$((fail_count + 1))
                fi
            done < <(find "$includes_dir" -name "*.tpl" -print0 2>/dev/null)
        fi
        
        # Render stream.d configs (Salt Master HA)
        local stream_dir="$nginx_template_dir/stream.d"
        if [[ -d "$stream_dir" ]]; then
            mkdir -p "$nginx_output_dir/stream.d"
            while IFS= read -r -d '' tpl_file; do
                local out_file="$nginx_output_dir/stream.d/$(basename "${tpl_file%.tpl}")"
                if render_template "$tpl_file" "$out_file"; then
                    success_count=$((success_count + 1))
                else
                    fail_count=$((fail_count + 1))
                fi
            done < <(find "$stream_dir" -name "*.tpl" -print0 2>/dev/null)
        fi
    fi
    
    # ===========================================
    # Render scripts/templates (e.g., install-salt-minion.sh.tpl)
    # These scripts contain EXTERNAL_HOST and other runtime variables
    # ===========================================
    local scripts_template_dir="${SCRIPT_DIR}/scripts/templates"
    if [[ -d "$scripts_template_dir" ]]; then
        log_info "Rendering script templates (scripts/templates/)..."
        
        while IFS= read -r -d '' tpl_file; do
            local tpl_basename=$(basename "$tpl_file")
            local out_file="${SCRIPT_DIR}/scripts/${tpl_basename%.tpl}"
            
            # Check if output file exists and is newer than template
            if [[ "$force" != "true" ]] && [[ -f "$out_file" ]]; then
                if [[ "$out_file" -nt "$tpl_file" ]] && [[ "$out_file" -nt "$ENV_FILE" ]]; then
                    log_info "Skipping $(basename "$out_file") (up to date)"
                    skip_count=$((skip_count + 1))
                    continue
                fi
            fi
            
            if render_template "$tpl_file" "$out_file"; then
                chmod +x "$out_file" 2>/dev/null || true
                success_count=$((success_count + 1))
            else
                fail_count=$((fail_count + 1))
            fi
        done < <(find "$scripts_template_dir" -name "*.tpl" -print0 2>/dev/null)
    fi
    
    # ===========================================
    # Render config templates (e.g., seaweedfs/s3.json.tpl)
    # ===========================================
    local config_template_dir="${SCRIPT_DIR}/config"
    if [[ -d "$config_template_dir" ]]; then
        log_info "Rendering config templates (config/*/*.tpl)..."
        
        while IFS= read -r -d '' tpl_file; do
            local out_file="${tpl_file%.tpl}"
            local config_name=$(basename "$out_file")
            
            # Check if output file exists and is newer than template
            if [[ "$force" != "true" ]] && [[ -f "$out_file" ]]; then
                if [[ "$out_file" -nt "$tpl_file" ]] && [[ "$out_file" -nt "$ENV_FILE" ]]; then
                    log_info "Skipping $config_name (up to date)"
                    skip_count=$((skip_count + 1))
                    continue
                fi
            fi
            
            if render_template "$tpl_file" "$out_file"; then
                success_count=$((success_count + 1))
            else
                fail_count=$((fail_count + 1))
            fi
        done < <(find "$config_template_dir" -name "*.tpl" -print0 2>/dev/null)
    fi
    
    # ===========================================
    # Render SaltStack pillar and state templates
    # These templates contain EXTERNAL_HOST and other runtime variables
    # for configuring node-metrics callback URLs, etc.
    # ===========================================
    local saltstack_dir="${SCRIPT_DIR}/src/saltstack"
    if [[ -d "$saltstack_dir" ]]; then
        log_info "Rendering SaltStack configuration templates..."
        
        # Render pillar templates (*.sls.tpl in salt-pillar/)
        local pillar_dir="$saltstack_dir/salt-pillar"
        if [[ -d "$pillar_dir" ]]; then
            while IFS= read -r -d '' tpl_file; do
                local out_file="${tpl_file%.tpl}"
                local config_name=$(basename "$out_file")
                
                # Check if output file exists and is newer than template
                if [[ "$force" != "true" ]] && [[ -f "$out_file" ]]; then
                    if [[ "$out_file" -nt "$tpl_file" ]] && [[ "$out_file" -nt "$ENV_FILE" ]]; then
                        log_info "Skipping $config_name (up to date)"
                        skip_count=$((skip_count + 1))
                        continue
                    fi
                fi
                
                if render_template "$tpl_file" "$out_file"; then
                    success_count=$((success_count + 1))
                else
                    fail_count=$((fail_count + 1))
                fi
            done < <(find "$pillar_dir" -name "*.sls.tpl" -print0 2>/dev/null)
        fi
        
        # Render state templates (*.sls.tpl in salt-states/)
        local states_dir="$saltstack_dir/salt-states"
        if [[ -d "$states_dir" ]]; then
            while IFS= read -r -d '' tpl_file; do
                local out_file="${tpl_file%.tpl}"
                local config_name=$(basename "$out_file")
                
                # Check if output file exists and is newer than template
                if [[ "$force" != "true" ]] && [[ -f "$out_file" ]]; then
                    if [[ "$out_file" -nt "$tpl_file" ]] && [[ "$out_file" -nt "$ENV_FILE" ]]; then
                        log_info "Skipping $config_name (up to date)"
                        skip_count=$((skip_count + 1))
                        continue
                    fi
                fi
                
                if render_template "$tpl_file" "$out_file"; then
                    success_count=$((success_count + 1))
                else
                    fail_count=$((fail_count + 1))
                fi
            done < <(find "$states_dir" -name "*.sls.tpl" -print0 2>/dev/null)
        fi
        
        # Render script templates in salt-states/files/
        local files_dir="$saltstack_dir/salt-states/files"
        if [[ -d "$files_dir" ]]; then
            while IFS= read -r -d '' tpl_file; do
                local out_file="${tpl_file%.tpl}"
                local config_name=$(basename "$out_file")
                
                # Check if output file exists and is newer than template
                if [[ "$force" != "true" ]] && [[ -f "$out_file" ]]; then
                    if [[ "$out_file" -nt "$tpl_file" ]] && [[ "$out_file" -nt "$ENV_FILE" ]]; then
                        log_info "Skipping $config_name (up to date)"
                        skip_count=$((skip_count + 1))
                        continue
                    fi
                fi
                
                if render_template "$tpl_file" "$out_file"; then
                    chmod +x "$out_file" 2>/dev/null || true
                    success_count=$((success_count + 1))
                else
                    fail_count=$((fail_count + 1))
                fi
            done < <(find "$files_dir" -name "*.tpl" -print0 2>/dev/null)
        fi
    fi
    
    # ===========================================
    # Sync third_party version files with .env
    # Update version.json and components.json based on .env variables
    # ===========================================
    sync_third_party_versions
    
    echo
    log_info "=========================================="
    log_info "Template rendering complete:"
    log_info "  ✓ Success: $success_count"
    [[ $skip_count -gt 0 ]] && log_info "  ⏭️  Skipped: $skip_count"
    [[ $fail_count -gt 0 ]] && log_warn "  ✗ Failed: $fail_count"
    log_info "=========================================="
    
    # ===========================================
    # Print component versions summary (dynamically discovered)
    # ===========================================
    echo
    log_info "=========================================="
    log_info "📦 Component Versions Summary"
    log_info "=========================================="
    echo
    printf "%-30s %-15s %s\n" "Component" "Type" "Version/Image"
    printf "%-30s %-15s %s\n" "------------------------------" "---------------" "--------------------"
    
    # Project version
    printf "%-30s %-15s %s\n" "AI-Infra-Matrix" "project" "${IMAGE_TAG:-N/A}"
    echo
    
    # Discover components from src/ directory
    local build_components=()
    local dependency_components=()
    
    for component_dir in "$SRC_DIR"/*/; do
        local component_name=$(basename "$component_dir")
        
        # Skip hidden directories and special dirs
        [[ "$component_name" == "shared" ]] && continue
        [[ "$component_name" =~ ^\. ]] && continue
        
        local has_dockerfile=false
        local has_dependency=false
        local version_info=""
        local component_type=""
        
        # Check for Dockerfile (build component)
        if [[ -f "${component_dir}Dockerfile" ]] || [[ -f "${component_dir}Dockerfile.tpl" ]]; then
            has_dockerfile=true
        fi
        
        # Check for dependency.conf (external image)
        if [[ -f "${component_dir}dependency.conf" ]]; then
            has_dependency=true
            # Read first non-comment, non-empty line as version info
            version_info=$(grep -v '^#' "${component_dir}dependency.conf" | grep -v '^[[:space:]]*$' | head -n 1)
        fi
        
        # Determine component type and version
        if [[ "$has_dockerfile" == "true" ]] && [[ "$has_dependency" == "true" ]]; then
            component_type="build+dep"
            build_components+=("$component_name|$component_type|${IMAGE_TAG:-latest} (dep: $version_info)")
        elif [[ "$has_dockerfile" == "true" ]]; then
            component_type="build"
            build_components+=("$component_name|$component_type|${IMAGE_TAG:-latest}")
        elif [[ "$has_dependency" == "true" ]]; then
            component_type="dependency"
            dependency_components+=("$component_name|$component_type|$version_info")
        fi
    done
    
    # Print build components
    if [[ ${#build_components[@]} -gt 0 ]]; then
        echo "--- Build Components (Dockerfile) ---"
        for item in "${build_components[@]}"; do
            IFS='|' read -r name type version <<< "$item"
            printf "%-30s %-15s %s\n" "$name" "$type" "$version"
        done
        echo
    fi
    
    # Print dependency components
    if [[ ${#dependency_components[@]} -gt 0 ]]; then
        echo "--- External Dependencies (dependency.conf) ---"
        for item in "${dependency_components[@]}"; do
            IFS='|' read -r name type version <<< "$item"
            printf "%-30s %-15s %s\n" "$name" "$type" "$version"
        done
        echo
    fi
    
    # Print key environment versions from .env
    echo "--- Key Environment Versions (.env) ---"
    local env_versions=(
        "GOLANG_IMAGE_VERSION:Golang"
        "UBUNTU_VERSION:Ubuntu"
        "NODE_VERSION:Node.js"
        "PYTHON_VERSION:Python"
        "ALPINE_VERSION:Alpine"
        "SALTSTACK_VERSION:SaltStack"
        "SLURM_VERSION:SLURM"
        "CATEGRAF_VERSION:Categraf"
        "SINGULARITY_VERSION:Singularity"
        "CODE_SERVER_VERSION:Code Server"
        "PROMETHEUS_VERSION:Prometheus"
        "GRAFANA_VERSION:Grafana"
        "VICTORIAMETRICS_VERSION:VictoriaMetrics"
        "N9E_FE_VERSION:Nightingale FE"
        "SAFELINE_IMAGE_TAG:SafeLine WAF"
    )
    
    for item in "${env_versions[@]}"; do
        IFS=':' read -r var_name display_name <<< "$item"
        local var_value="${!var_name:-N/A}"
        printf "%-30s %-15s %s\n" "$display_name" "env" "$var_value"
    done
    echo
    
    log_info "=========================================="
    log_info "Total: ${#build_components[@]} build + ${#dependency_components[@]} dependency components"
    log_info "=========================================="
    
    if [[ $fail_count -gt 0 ]]; then
        return 1
    fi
    return 0
}

# Sync templates - alias for render_all_templates with force
sync_templates() {
    render_all_templates "true"
}

# ==============================================================================
# Pull/Push Functions - 镜像操作功能 (优化后的统一重试机制)
# ==============================================================================

# Default retry settings
DEFAULT_MAX_RETRIES=3
DEFAULT_RETRY_DELAY=5

# Log file for tracking failures
FAILURE_LOG="${SCRIPT_DIR}/.build-failures.log"

# Log failure to file
log_failure() {
    local operation="$1"
    local target="$2"
    local error_msg="$3"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[$timestamp] $operation FAILED: $target - $error_msg" >> "$FAILURE_LOG"
    log_error "[$timestamp] $operation FAILED: $target - $error_msg"
}

# 通用的 Docker 命令重试执行器
# 用法: docker_with_retry <operation> <image> [max_retries] [retry_delay] [skip_exists_check]
# operation: pull, push, tag
# skip_exists_check: 对于 push 操作设为 true
# 对于 pull 操作，会自动添加 --platform 参数确保拉取正确架构的镜像
docker_with_retry() {
    local operation="$1"
    local image="$2"
    local max_retries="${3:-$DEFAULT_MAX_RETRIES}"
    local retry_delay="${4:-$DEFAULT_RETRY_DELAY}"
    local skip_exists_check="${5:-false}"
    local retry_count=0
    local last_error=""
    
    # 操作符号和显示文本映射 (macOS bash 3.x 兼容)
    local op_icon="" op_verb="" op_past=""
    case "$operation" in
        pull) op_icon="⬇"; op_verb="Pulling"; op_past="Pulled" ;;
        push) op_icon="⬆"; op_verb="Pushing"; op_past="Pushed" ;;
        *)    op_icon="⚙"; op_verb="Processing"; op_past="Processed" ;;
    esac
    
    # 对于 pull 操作，检查镜像是否已存在且架构匹配
    if [[ "$operation" == "pull" ]] && [[ "$skip_exists_check" != "true" ]]; then
        if docker image inspect "$image" >/dev/null 2>&1; then
            # 检查已存在镜像的架构是否与目标平台匹配
            local existing_arch=$(docker image inspect "$image" --format '{{.Architecture}}' 2>/dev/null)
            local expected_arch=""
            case "$DOCKER_HOST_PLATFORM" in
                linux/amd64) expected_arch="amd64" ;;
                linux/arm64) expected_arch="arm64" ;;
                linux/arm/v7) expected_arch="arm" ;;
            esac
            
            # 如果指定了目标架构，且已有镜像架构不匹配或为空，需要重新拉取
            if [[ -n "$expected_arch" ]]; then
                if [[ -z "$existing_arch" ]] || [[ "$existing_arch" != "$expected_arch" ]]; then
                    if [[ -z "$existing_arch" ]]; then
                        log_warn "  ⚠ Image exists but arch unknown, re-pulling for: $expected_arch"
                    else
                        log_warn "  ⚠ Image exists but arch mismatch: $existing_arch (expected: $expected_arch)"
                    fi
                    log_info "  🗑 Removing wrong-arch image before re-pulling..."
                    docker rmi "$image" >/dev/null 2>&1 || true
                    # 继续执行 pull 操作
                else
                    log_info "  ✓ Image exists: $image (arch: $existing_arch)"
                    return 0
                fi
            else
                # 未指定目标架构，镜像存在就跳过
                log_info "  ✓ Image exists: $image (arch: ${existing_arch:-unknown})"
                return 0
            fi
        fi
    fi
    
    while [[ $retry_count -lt $max_retries ]]; do
        retry_count=$((retry_count + 1))
        
        if [[ $retry_count -gt 1 ]]; then
            log_warn "  🔄 Retry $retry_count/$max_retries: $image (waiting ${retry_delay}s...)"
            sleep $retry_delay
        else
            log_info "  $op_icon $op_verb: $image"
        fi
        
        # 执行 Docker 命令
        # 对于 pull 操作，添加 --platform 参数确保拉取正确架构
        local output
        if [[ "$operation" == "pull" ]] && [[ -n "$DOCKER_HOST_PLATFORM" ]]; then
            if output=$(docker pull --platform "$DOCKER_HOST_PLATFORM" "$image" 2>&1); then
                log_info "  ✓ $op_past: $image (platform: $DOCKER_HOST_PLATFORM)"
                return 0
            else
                last_error="$output"
                log_warn "  ⚠ Attempt $retry_count failed: $(echo "$last_error" | head -1)"
            fi
        else
            if output=$(docker "$operation" "$image" 2>&1); then
                log_info "  ✓ $op_past: $image"
                return 0
            else
                last_error="$output"
                log_warn "  ⚠ Attempt $retry_count failed: $(echo "$last_error" | head -1)"
            fi
        fi
    done
    
    # 所有重试失败
    log_failure "${operation^^}" "$image" "Failed after $max_retries attempts. Last error: $(echo "$last_error" | head -1)"
    return 1
}

# Pull single image with retry mechanism (使用通用重试器)
# Args: $1 = image name, $2 = max retries (default 3), $3 = retry delay (default 5)
pull_image_with_retry() {
    local image="$1"
    local max_retries="${2:-$DEFAULT_MAX_RETRIES}"
    local retry_delay="${3:-$DEFAULT_RETRY_DELAY}"
    
    # 使用通用重试器
    docker_with_retry "pull" "$image" "$max_retries" "$retry_delay" "false"
}

# Push single image with retry mechanism (使用通用重试器)
# Args: $1 = image, $2 = max retries (default 3), $3 = retry delay (default 5)
push_image_with_retry() {
    local image="$1"
    local max_retries="${2:-$DEFAULT_MAX_RETRIES}"
    local retry_delay="${3:-$DEFAULT_RETRY_DELAY}"
    
    # push 操作不检查本地镜像是否存在
    docker_with_retry "push" "$image" "$max_retries" "$retry_delay" "true"
}

# Extract base images from Dockerfile
# Args: $1 = Dockerfile path
extract_base_images() {
    local dockerfile="$1"
    
    if [[ ! -f "$dockerfile" ]]; then
        return 1
    fi
    
    # Extract ARG default values for variable substitution
    # Format: ARG VAR_NAME=default_value
    declare -A arg_defaults
    while IFS= read -r line; do
        if [[ "$line" =~ ^[[:space:]]*ARG[[:space:]]+([A-Za-z_][A-Za-z0-9_]*)=(.+) ]]; then
            local var_name="${BASH_REMATCH[1]}"
            local var_value="${BASH_REMATCH[2]}"
            # Remove quotes if present
            var_value="${var_value%\"}"
            var_value="${var_value#\"}"
            arg_defaults["$var_name"]="$var_value"
        fi
    done < "$dockerfile"
    
    # Extract FROM statements
    # Pattern: FROM image:tag [AS alias]
    # Handles: ARG variables (${...}), platform flags, local build stages
    grep -E "^FROM\s+" "$dockerfile" 2>/dev/null | \
        while read -r from_line; do
            # Remove FROM keyword and any --platform flags
            local img=$(echo "$from_line" | sed 's/^FROM\s*//; s/--platform=[^ ]*\s*//g' | awk '{print $1}')
            
            # Skip if no image specified
            [[ -z "$img" ]] && continue
            
            # Check for ARG variable substitution (e.g., ubuntu:${UBUNTU_VERSION})
            if [[ "$img" =~ \$\{([A-Za-z_][A-Za-z0-9_]*)\} ]]; then
                local var_name="${BASH_REMATCH[1]}"
                local var_value="${arg_defaults[$var_name]:-}"
                if [[ -n "$var_value" ]]; then
                    # Substitute the variable with its default value
                    img=$(echo "$img" | sed "s/\${${var_name}}/${var_value}/g")
                else
                    # Skip images with unresolved variables
                    continue
                fi
            fi
            
            # Skip if no colon and no slash (likely a build stage alias like "builder")
            if [[ ! "$img" =~ [:\/] ]]; then
                continue
            fi
            
            echo "$img"
        done | sort -u
}

# Prefetch base images from Dockerfiles
# Args: $1 = service name (optional, if empty prefetch all)
prefetch_base_images() {
    local service_name="$1"
    local max_retries="${2:-3}"
    
    log_info "📦 Prefetching base images..."
    
    local dockerfiles=()
    
    if [[ -n "$service_name" ]]; then
        local dockerfile="$SRC_DIR/$service_name/Dockerfile"
        if [[ -f "$dockerfile" ]]; then
            dockerfiles+=("$dockerfile")
        fi
    else
        # Find all Dockerfiles, excluding node_modules and other irrelevant directories
        while IFS= read -r df; do
            dockerfiles+=("$df")
        done < <(find "$SRC_DIR" -name "Dockerfile" -type f \
            -not -path "*/node_modules/*" \
            -not -path "*/.git/*" \
            -not -path "*/vendor/*" \
            -not -path "*/__pycache__/*" \
            2>/dev/null)
    fi
    
    local all_images=()
    local pull_count=0
    local skip_count=0
    local fail_count=0
    
    # Extract all base images
    for dockerfile in "${dockerfiles[@]}"; do
        local images
        images=$(extract_base_images "$dockerfile")
        while IFS= read -r img; do
            [[ -z "$img" ]] && continue
            [[ "$img" =~ ^[a-z_-]+$ ]] && continue  # Skip internal build stages
            all_images+=("$img")
        done <<< "$images"
    done
    
    # Remove duplicates
    local unique_images=($(printf '%s\n' "${all_images[@]}" | sort -u))
    
    log_info "Found ${#unique_images[@]} unique base images to check"
    
    for image in "${unique_images[@]}"; do
        if docker image inspect "$image" >/dev/null 2>&1; then
            log_info "  ✓ Exists: $image"
            skip_count=$((skip_count + 1))
        else
            log_info "  ⬇ Pulling: $image"
            if pull_image_with_retry "$image" "$max_retries"; then
                pull_count=$((pull_count + 1))
            else
                fail_count=$((fail_count + 1))
            fi
        fi
    done
    
    log_info "📊 Prefetch summary: pulled=$pull_count, skipped=$skip_count, failed=$fail_count"
    return 0
}

# Pull all project images from registry
# Smart mode:
#   - No registry: Pull only common images from Docker Hub (internet mode)
#   - With registry: Pull all images from private registry (intranet mode)
# Args: $1 = registry (optional), $2 = tag
# 
# For Harbor/private registries, registry path should include project name:
#   ✓ harbor.example.com/ai-infra    (correct - includes project)
#   ✗ harbor.example.com             (wrong - missing project)
pull_all_services() {
    local registry="$1"
    local tag="${2:-${IMAGE_TAG:-latest}}"
    local max_retries="${3:-$DEFAULT_MAX_RETRIES}"
    local platforms="${4:-}"  # Optional: amd64,arm64 or empty for unified tag
    
    discover_services
    
    local success_count=0
    local total_count=0
    local failed_services=()
    
    # 使用通用函数验证 registry 路径
    if ! validate_registry_path "$registry" "$tag"; then
        return 1
    fi
    
    # Parse platforms parameter (--platform=amd64,arm64 -> amd64,arm64)
    local platforms_value=""
    if [[ -n "$platforms" ]]; then
        if [[ "$platforms" == --platform=* ]]; then
            platforms_value="${platforms#--platform=}"
        else
            platforms_value="$platforms"
        fi
    fi
    
    # Normalize and parse platforms into array
    local -a PLATFORM_ARRAY=()
    if [[ -n "$platforms_value" ]]; then
        IFS=',' read -ra PLATFORM_ARRAY <<< "$platforms_value"
        # Normalize platform names
        for i in "${!PLATFORM_ARRAY[@]}"; do
            local p="${PLATFORM_ARRAY[$i]}"
            case "$p" in
                linux/amd64|amd64|x86_64) PLATFORM_ARRAY[$i]="amd64" ;;
                linux/arm64|arm64|aarch64) PLATFORM_ARRAY[$i]="arm64" ;;
                *)
                    log_warn "Unknown platform: $p, skipping..."
                    unset 'PLATFORM_ARRAY[$i]'
                    ;;
            esac
        done
    fi
    
    if [[ -z "$registry" ]]; then
        # ==========================================
        # Mode 1: Pull from Docker Hub (internet mode)
        # Only pulls public common/third-party images
        # ==========================================
        log_info "=========================================="
        log_info "Pulling images from Docker Hub (Internet Mode)"
        log_info "=========================================="
        log_info "Mode: Public (Docker Hub)"
        log_info "Max retries: $max_retries"
        echo
        
        # Phase 1: Pull common/third-party images from Docker Hub
        log_info "=== Phase 1: Common/third-party images ==="
        for image in "${COMMON_IMAGES[@]}"; do
            total_count=$((total_count + 1))
            log_info "[$total_count/${#COMMON_IMAGES[@]}] $image"
            
            # 使用 pull_image_with_retry 进行拉取（内部会处理架构验证）
            # 不再单独检查镜像是否存在，因为可能存在架构不匹配的情况
            if pull_image_with_retry "$image" "$max_retries"; then
                success_count=$((success_count + 1))
            else
                log_warn "  ✗ Failed"
                failed_services+=("common:$image")
            fi
        done
        echo
        
        # Phase 1.5: Pull SafeLine WAF images from Docker Hub
        log_info "=== Phase 1.5: SafeLine WAF images ==="
        log_info "Architecture suffix: ${SAFELINE_ARCH_SUFFIX:-<none>}"
        local safeline_count=0
        for image in "${SAFELINE_IMAGES[@]}"; do
            safeline_count=$((safeline_count + 1))
            total_count=$((total_count + 1))
            log_info "[$safeline_count/${#SAFELINE_IMAGES[@]}] $image"
            
            # 使用 pull_image_with_retry 进行拉取（内部会处理架构验证）
            if pull_image_with_retry "$image" "$max_retries"; then
                success_count=$((success_count + 1))
            else
                log_warn "  ✗ Failed"
                failed_services+=("safeline:$image")
            fi
        done
        echo
        
        log_info "=== Phase 2: Project services (skipped - need registry) ==="
        log_info "ℹ️  Project images require registry to pull"
        log_info "💡 Usage: $0 pull-all <registry> [tag]"
        echo
        
    else
        # ==========================================
        # Mode 2: Pull from private registry (intranet mode)
        # Pulls all images: common + dependency + project
        # ==========================================
        registry="${registry%/}"  # Remove trailing slash
        
        log_info "=========================================="
        log_info "Pulling images from Private Registry (Intranet Mode)"
        log_info "=========================================="
        log_info "Mode: Private Registry"
        log_info "Registry: $registry"
        log_info "Tag: $tag"
        log_info "Max retries: $max_retries"
        if [[ ${#PLATFORM_ARRAY[@]} -gt 0 ]]; then
            log_info "Platforms: ${PLATFORM_ARRAY[*]}"
            log_info "Mode: Multi-architecture (方案一 - pull arch-specific tags)"
        fi
        echo
        
        # If platforms specified, iterate through each platform for project services
        if [[ ${#PLATFORM_ARRAY[@]} -gt 0 ]]; then
            # Multi-architecture pull mode
            # Phase 1 and 2 (common/dependency) are skipped in multi-arch mode
            # as they are architecture-agnostic and should be pulled without platform suffix
            
            for platform in "${PLATFORM_ARRAY[@]}"; do
                log_info "=========================================="
                log_info "Pulling platform: $platform"
                log_info "=========================================="
                echo
                
                # Phase 3: Pull project services with architecture suffix
                log_info "=== Project services (tag: ${tag}-${platform}) ==="
                for service in "${FOUNDATION_SERVICES[@]}" "${DEPENDENT_SERVICES[@]}"; do
                    total_count=$((total_count + 1))
                    local image_name="ai-infra-${service}:${tag}-${platform}"
                    local remote_image="${registry}/ai-infra-${service}:${tag}-${platform}"
                    
                    log_info "[$total_count] $remote_image"
                    
                    if pull_image_with_retry "$remote_image" "$max_retries"; then
                        if docker tag "$remote_image" "$image_name"; then
                            log_info "  ✓ Tagged as $image_name"
                            success_count=$((success_count + 1))
                        else
                            log_warn "  ⚠ Pulled but failed to tag"
                            success_count=$((success_count + 1))
                        fi
                    else
                        log_warn "  ✗ Failed"
                        failed_services+=("${service}-${platform}")
                    fi
                done
                echo
                
                # Phase 4: Pull special images with architecture suffix
                log_info "=== Special images (tag: ${tag}-${platform}) ==="
                local special_images=("backend-init")
                for special in "${special_images[@]}"; do
                    total_count=$((total_count + 1))
                    local image_name="ai-infra-${special}:${tag}-${platform}"
                    local remote_image="${registry}/ai-infra-${special}:${tag}-${platform}"
                    
                    log_info "[$total_count] $remote_image"
                    
                    if pull_image_with_retry "$remote_image" "$max_retries"; then
                        if docker tag "$remote_image" "$image_name"; then
                            log_info "  ✓ Tagged as $image_name"
                            success_count=$((success_count + 1))
                        else
                            log_warn "  ⚠ Pulled but failed to tag"
                            success_count=$((success_count + 1))
                        fi
                    else
                        log_warn "  ✗ Failed"
                        failed_services+=("special:${special}-${platform}")
                    fi
                done
                echo
            done
            
            log_info "Note: Common and dependency images should be pulled separately without"
            log_info "      platform suffix using: ./build.sh pull-all $registry $tag"
            echo
        else
            # Original unified tag mode (no platform specified)
            
            # Phase 1: Pull common images from private registry
            log_info "=== Phase 1: Common/third-party images ==="
            for image in "${COMMON_IMAGES[@]}"; do
                total_count=$((total_count + 1))
                
                # Extract short name (e.g., confluentinc/cp-kafka:7.5.0 -> cp-kafka)
                local image_name="${image%%:*}"
                local image_tag="${image##*:}"
                local short_name="${image_name##*/}"
                local remote_image="${registry}/${short_name}:${image_tag}"
                
                log_info "[$total_count] $remote_image"
                
                # 使用 pull_image_with_retry 进行拉取（内部会处理架构验证）
                # 即使本地镜像存在，也需要检查架构是否匹配
                if pull_image_with_retry "$remote_image" "$max_retries"; then
                    # Tag as original image name for docker-compose compatibility
                    if docker tag "$remote_image" "$image"; then
                        log_info "  ✓ Tagged as $image"
                        success_count=$((success_count + 1))
                    else
                        log_warn "  ⚠ Pulled but failed to tag"
                        success_count=$((success_count + 1))
                    fi
                else
                    log_warn "  ✗ Failed to pull from registry"
                    failed_services+=("common:$short_name")
                fi
            done
            echo
            
            # Phase 2: Pull dependency images with project tag
            log_info "=== Phase 2: Dependency images (tag: $tag) ==="
            local dependencies=($(get_dependency_mappings))
            for mapping in "${dependencies[@]}"; do
                total_count=$((total_count + 1))
                
                local source_image="${mapping%%|*}"
                local short_name="${mapping##*|}"
                local remote_image="${registry}/${short_name}:${tag}"
                
                log_info "[$total_count] $remote_image -> $source_image"
                
                # 使用 pull_image_with_retry 进行拉取（内部会处理架构验证）
                if pull_image_with_retry "$remote_image" "$max_retries"; then
                    # Tag as original image name for docker-compose compatibility
                    if docker tag "$remote_image" "$source_image"; then
                        log_info "  ✓ Tagged as $source_image"
                        success_count=$((success_count + 1))
                    else
                        log_warn "  ⚠ Pulled but failed to tag"
                        success_count=$((success_count + 1))
                    fi
                else
                    log_warn "  ✗ Failed"
                    failed_services+=("dep:$short_name")
                fi
            done
            echo
            
            # Phase 3: Pull project services
            log_info "=== Phase 3: Project services (tag: $tag) ==="
            for service in "${FOUNDATION_SERVICES[@]}" "${DEPENDENT_SERVICES[@]}"; do
                total_count=$((total_count + 1))
                local image_name="ai-infra-${service}:${tag}"
                local remote_image="${registry}/${image_name}"
                
                log_info "[$total_count] $remote_image"
                
                # 使用 pull_image_with_retry 进行拉取（内部会处理架构验证）
                if pull_image_with_retry "$remote_image" "$max_retries"; then
                    if docker tag "$remote_image" "$image_name"; then
                        log_info "  ✓ Tagged as $image_name"
                        success_count=$((success_count + 1))
                    else
                        log_warn "  ⚠ Pulled but failed to tag"
                        success_count=$((success_count + 1))
                    fi
                else
                    log_warn "  ✗ Failed"
                    failed_services+=("$service")
                fi
            done
            echo
            
            # Phase 4: Pull special images (multi-stage build targets, etc.)
            # These are images that don't have their own src/ directory
            log_info "=== Phase 4: Special images (tag: $tag) ==="
            local special_images=(
                "backend-init"    # Multi-stage build target from backend
            )
            for special in "${special_images[@]}"; do
                total_count=$((total_count + 1))
                local image_name="ai-infra-${special}:${tag}"
                local remote_image="${registry}/${image_name}"
                
                log_info "[$total_count] $remote_image"
                
                if pull_image_with_retry "$remote_image" "$max_retries"; then
                    if docker tag "$remote_image" "$image_name"; then
                        log_info "  ✓ Pulled and tagged as $image_name"
                        success_count=$((success_count + 1))
                    else
                        log_warn "  ⚠ Pulled but failed to tag"
                        success_count=$((success_count + 1))
                    fi
                else
                    log_warn "  ✗ Failed"
                    failed_services+=("special:$special")
                fi
            done
            echo
        fi
    fi
    
    log_info "=========================================="
    log_info "Pull completed: $success_count/$total_count successful"
    
    if [[ ${#failed_services[@]} -gt 0 ]]; then
        log_warn "Failed: ${failed_services[*]}"
        log_info "Check failure log: $FAILURE_LOG"
        return 1
    fi
    
    log_info "🎉 All images pulled successfully!"
    return 0
}

# Pull only common/third-party images (no registry required)
# Useful for preparing environment before starting services
pull_common_images() {
    local max_retries="${1:-$DEFAULT_MAX_RETRIES}"
    
    log_info "=========================================="
    log_info "Pulling common/third-party images"
    log_info "=========================================="
    log_info "Images to pull: ${#COMMON_IMAGES[@]}"
    log_info "Max retries: $max_retries"
    echo
    
    local success_count=0
    local total_count=0
    local failed_images=()
    
    for image in "${COMMON_IMAGES[@]}"; do
        total_count=$((total_count + 1))
        log_info "[$total_count/${#COMMON_IMAGES[@]}] Pulling: $image"
        
        # Check if image already exists locally
        if docker image inspect "$image" &>/dev/null; then
            log_info "  ✓ Already exists: $image"
            success_count=$((success_count + 1))
            continue
        fi
        
        if pull_image_with_retry "$image" "$max_retries"; then
            log_info "  ✓ Pulled: $image"
            success_count=$((success_count + 1))
        else
            log_warn "  ✗ Failed: $image"
            failed_images+=("$image")
        fi
    done
    
    echo
    log_info "=========================================="
    log_info "Pulling SafeLine WAF images"
    log_info "=========================================="
    log_info "Images to pull: ${#SAFELINE_IMAGES[@]}"
    log_info "Architecture suffix: ${SAFELINE_ARCH_SUFFIX:-<none>}"
    echo
    
    for image in "${SAFELINE_IMAGES[@]}"; do
        total_count=$((total_count + 1))
        log_info "[SafeLine] Pulling: $image"
        
        # Check if image already exists locally
        if docker image inspect "$image" &>/dev/null; then
            log_info "  ✓ Already exists: $image"
            success_count=$((success_count + 1))
            continue
        fi
        
        if pull_image_with_retry "$image" "$max_retries"; then
            log_info "  ✓ Pulled: $image"
            success_count=$((success_count + 1))
        else
            log_warn "  ✗ Failed: $image"
            failed_images+=("$image")
        fi
    done
    
    echo
    log_info "=========================================="
    log_info "Pull completed: $success_count/$total_count successful"
    
    if [[ ${#failed_images[@]} -gt 0 ]]; then
        log_warn "Failed images: ${failed_images[*]}"
        return 1
    fi
    
    log_info "🎉 All common images pulled successfully!"
    return 0
}

# ==============================================================================
# Push Functions - 镜像推送功能 (支持多架构)
# ==============================================================================

# Push single service image for specific platform
# Args: $1 = service, $2 = tag, $3 = registry, $4 = max_retries, $5 = platform (optional), $6 = arch_tag (optional)
#
# Platform and arch_tag behavior:
#   - platform: Determines which local image to push (e.g., ai-infra-xxx:v0.3.8-amd64)
#   - arch_tag: Controls whether remote tag includes architecture suffix
#     - "true" (default when platform set): Remote tag includes arch suffix (Harbor mode)
#     - "false": Remote tag without arch suffix (Docker Hub mode, uses multi-manifest)
#
# Examples:
#   push_service backend v0.3.8 docker.io/user 3 amd64 true   # -> docker.io/user/ai-infra-backend:v0.3.8-amd64
#   push_service backend v0.3.8 docker.io/user 3 amd64 false  # -> docker.io/user/ai-infra-backend:v0.3.8
#   push_service backend v0.3.8 docker.io/user 3              # -> docker.io/user/ai-infra-backend:v0.3.8
push_service() {
    local service="$1"
    local tag="${2:-${IMAGE_TAG:-latest}}"
    local registry="$3"
    local max_retries="${4:-$DEFAULT_MAX_RETRIES}"
    local platform="${5:-}"      # Optional: amd64, arm64, or empty
    local arch_tag="${6:-true}"  # Whether to include arch suffix in remote tag
    
    if [[ -z "$registry" ]]; then
        log_error "Registry is required for push"
        return 1
    fi
    
    # Determine image names based on platform
    local arch_suffix=""
    local remote_arch_suffix=""
    if [[ -n "$platform" ]]; then
        # Normalize platform name for local image
        case "$platform" in
            linux/amd64|amd64|x86_64) 
                arch_suffix="-amd64"
                ;;
            linux/arm64|arm64|aarch64) 
                arch_suffix="-arm64"
                ;;
        esac
        # Only add arch suffix to remote tag if arch_tag is true
        if [[ "$arch_tag" == "true" ]]; then
            remote_arch_suffix="$arch_suffix"
        fi
    fi
    
    local base_image="ai-infra-${service}:${tag}${arch_suffix}"
    local target_image="$registry/ai-infra-${service}:${tag}${remote_arch_suffix}"
    
    log_info "Pushing service: $service${arch_suffix:+ ($arch_suffix)}"
    log_info "  Source: $base_image"
    log_info "  Target: $target_image"
    if [[ -n "$platform" ]] && [[ "$arch_tag" == "false" ]]; then
        log_info "  Mode: Docker Hub (no arch suffix in remote tag)"
    fi
    
    # Check if source image exists
    if ! docker image inspect "$base_image" >/dev/null 2>&1; then
        log_warn "Local image not found: $base_image"
        if [[ -n "$platform" ]]; then
            log_info "Hint: Build with './build.sh $service --platform=${platform##*/}'"
        else
            log_info "Building image first..."
            if ! build_component "$service"; then
                log_failure "BUILD" "$base_image" "Build failed before push"
                return 1
            fi
        fi
        return 1
    fi
    
    # Tag for registry with retry
    if [[ "$base_image" != "$target_image" ]]; then
        log_info "  Tagging: $base_image -> $target_image"
        if ! docker tag "$base_image" "$target_image"; then
            log_failure "TAG" "$target_image" "Failed to tag image"
            return 1
        fi
    fi
    
    # Push to registry with retry
    log_info "  Pushing: $target_image"
    if push_image_with_retry "$target_image" "$max_retries"; then
        return 0
    else
        return 1
    fi
}

# Push all service images (including common/dependency images)
# This function pushes images in 3 phases for complete offline deployment:
#   Phase 1: Common images (original tags) - for general use
#   Phase 2: Dependency images (project tag) - for version-controlled deployment
#   Phase 3: Project services (project tag) - the main application images
# Args: $1 = registry, $2 = tag, $3 = max_retries, $4 = platforms (optional), $5 = arch_tag (optional)
#
# For Harbor/private registries, registry path should include project name:
#   ✓ harbor.example.com/ai-infra    (correct - includes project)
#   ✗ harbor.example.com             (wrong - missing project)
#
# Multi-architecture support (方案一):
#   Default: builds both amd64 and arm64, pushes without arch suffix (Docker Hub mode).
#   Use --arch-tag for Harbor (remote tag with arch suffix like v0.3.8-amd64).
#   
#   Usage examples:
#   ./build.sh push-all docker.io/user v0.3.8                                           # Docker Hub: v0.3.8 (default)
#   ./build.sh push-all docker.io/user v0.3.8 --platform=amd64                          # Docker Hub: v0.3.8 (single arch)
#   ./build.sh push-all harbor.example.com/ai-infra v0.3.8 --arch-tag                   # Harbor: v0.3.8-amd64, v0.3.8-arm64
#   ./build.sh push-all harbor.example.com/ai-infra v0.3.8 --platform=amd64 --arch-tag  # Harbor: v0.3.8-amd64
push_all_services() {
    local registry="$1"
    local tag="${2:-${IMAGE_TAG:-latest}}"
    local max_retries="${3:-$DEFAULT_MAX_RETRIES}"
    local platforms="${4:-}"    # Optional: amd64,arm64 or empty for unified tag
    local arch_tag="${5:-false}" # Whether to include arch suffix in remote tag (--arch-tag sets to true)
    
    if [[ -z "$registry" ]]; then
        log_error "Registry is required for push-all"
        log_info "Usage: $0 push-all <registry/project> [tag] [--platform=amd64,arm64] [--arch-tag]"
        log_info ""
        log_info "Examples:"
        log_info "  $0 push-all docker.io/user v0.3.8                                        # Docker Hub: v0.3.8 (default)"
        log_info "  $0 push-all docker.io/user v0.3.8 --platform=amd64                       # Docker Hub: single arch"
        log_info "  $0 push-all harbor.example.com/ai-infra v0.3.8 --arch-tag                # Harbor: v0.3.8-amd64, v0.3.8-arm64"
        return 1
    fi
    
    # 使用通用函数验证 registry 路径
    if ! validate_registry_path "$registry" "$tag"; then
        return 1
    fi
    
    # Ensure registry ends without trailing slash
    registry="${registry%/}"
    
    # Parse platforms parameter (--platform=amd64,arm64 -> amd64,arm64)
    local platforms_value=""
    if [[ -n "$platforms" ]]; then
        if [[ "$platforms" == --platform=* ]]; then
            platforms_value="${platforms#--platform=}"
        else
            platforms_value="$platforms"
        fi
    fi
    
    # Normalize and parse platforms into array
    local -a PLATFORM_ARRAY=()
    if [[ -n "$platforms_value" ]]; then
        IFS=',' read -ra PLATFORM_ARRAY <<< "$platforms_value"
        # Normalize platform names
        for i in "${!PLATFORM_ARRAY[@]}"; do
            local p="${PLATFORM_ARRAY[$i]}"
            case "$p" in
                linux/amd64|amd64|x86_64) PLATFORM_ARRAY[$i]="amd64" ;;
                linux/arm64|arm64|aarch64) PLATFORM_ARRAY[$i]="arm64" ;;
                *)
                    log_warn "Unknown platform: $p, skipping..."
                    unset 'PLATFORM_ARRAY[$i]'
                    ;;
            esac
        done
    fi
    
    log_info "=========================================="
    log_info "Pushing ALL images to registry"
    log_info "=========================================="
    log_info "Registry: $registry"
    log_info "Tag: $tag"
    log_info "Max retries: $max_retries"
    if [[ ${#PLATFORM_ARRAY[@]} -gt 0 ]]; then
        log_info "Platforms: ${PLATFORM_ARRAY[*]}"
        if [[ "$arch_tag" == "true" ]]; then
            log_info "Mode: Multi-arch with arch suffix (Harbor mode: v0.3.8-amd64)"
        else
            log_info "Mode: Multi-arch without arch suffix (Docker Hub mode: v0.3.8)"
        fi
    else
        log_info "Mode: Unified tags (no platform specified)"
    fi
    echo
    
    discover_services
    
    local success_count=0
    local total_count=0
    local failed_services=()
    
    # If platforms specified, iterate through each platform
    if [[ ${#PLATFORM_ARRAY[@]} -gt 0 ]]; then
        for platform in "${PLATFORM_ARRAY[@]}"; do
            log_info "=========================================="
            log_info "Processing platform: $platform"
            log_info "=========================================="
            echo
            
            # Determine remote tag based on arch_tag setting
            local remote_tag_display="${tag}"
            if [[ "$arch_tag" == "true" ]]; then
                remote_tag_display="${tag}-${platform}"
            fi
            
            # Phase 3 (only for multi-arch): Push project services
            log_info "=== Project services (remote tag: ${remote_tag_display}) ==="
            log_info "Main application images built from src/*"
            echo
            for service in "${FOUNDATION_SERVICES[@]}" "${DEPENDENT_SERVICES[@]}"; do
                total_count=$((total_count + 1))
                
                if push_service "$service" "$tag" "$registry" "$max_retries" "$platform" "$arch_tag"; then
                    log_info "  ✓ $service (${platform}) pushed"
                    success_count=$((success_count + 1))
                else
                    failed_services+=("$service-${platform}")
                fi
            done
            echo
            
            # Phase 4 (multi-arch): Push special images
            log_info "=== Special images (remote tag: ${remote_tag_display}) ==="
            echo
            local special_images=(
                "backend-init"
            )
            for special in "${special_images[@]}"; do
                total_count=$((total_count + 1))
                local image_name="ai-infra-${special}:${tag}-${platform}"
                # Determine target image based on arch_tag
                local target_image
                if [[ "$arch_tag" == "true" ]]; then
                    target_image="${registry}/ai-infra-${special}:${tag}-${platform}"
                else
                    target_image="${registry}/ai-infra-${special}:${tag}"
                fi
                
                log_info "[$total_count] $image_name -> $target_image"
                
                if ! docker image inspect "$image_name" >/dev/null 2>&1; then
                    log_warn "  ✗ Source image not found: $image_name"
                    log_info "    Hint: Build with '--platform=${platform}'"
                    failed_services+=("special:${special}-${platform}")
                    continue
                fi
                
                if ! docker tag "$image_name" "$target_image"; then
                    log_warn "  ✗ Failed to tag: $target_image"
                    failed_services+=("special:${special}-${platform}")
                    continue
                fi
                
                if push_image_with_retry "$target_image" "$max_retries"; then
                    log_info "  ✓ Pushed"
                    success_count=$((success_count + 1))
                else
                    failed_services+=("special:${special}-${platform}")
                fi
            done
            echo
        done
        
        # For multi-arch mode, we skip Phase 1 and Phase 2 (common/dependency images)
        # These are pulled from Docker Hub and don't need arch suffix in our tag scheme
        log_info "Note: Common and dependency images are architecture-agnostic and"
        log_info "      should be pushed separately without platform suffix using:"
        log_info "      ./build.sh push-all $registry $tag"
        echo
    else
        # Original unified tag mode (no platform specified)
        
        # Phase 1: Push common/third-party images with original tags
        log_info "=== Phase 1: Common/third-party images (original tags) ==="
        log_info "These images keep their original tags for general compatibility"
        echo
        for image in "${COMMON_IMAGES[@]}"; do
            total_count=$((total_count + 1))
            
            local image_name="${image%%:*}"
            local image_tag="${image##*:}"
            local short_name="${image_name##*/}"
            local target_image="${registry}/${short_name}:${image_tag}"
            
            log_info "[$total_count] $image -> $target_image"
            if ! docker image inspect "$image" >/dev/null 2>&1; then
                log_info "  Pulling source image..."
                if ! pull_image_with_retry "$image" "$max_retries"; then
                    log_warn "  ✗ Failed to pull: $image"
                    failed_services+=("common:$image")
                    continue
                fi
            fi
            
            if ! docker tag "$image" "$target_image"; then
                log_warn "  ✗ Failed to tag: $target_image"
                failed_services+=("common:$image")
                continue
            fi
            
            if push_image_with_retry "$target_image" "$max_retries"; then
                log_info "  ✓ Pushed"
                success_count=$((success_count + 1))
            else
                failed_services+=("common:$image")
            fi
        done
        echo
        
        # Phase 2: Push dependency images with project tag
        log_info "=== Phase 2: Dependency images (tag: $tag) ==="
        log_info "These images are tagged with project version for version-controlled deployment"
        echo
        local dependencies=($(get_dependency_mappings))
        for mapping in "${dependencies[@]}"; do
            total_count=$((total_count + 1))
            
            local source_image="${mapping%%|*}"
            local short_name="${mapping##*|}"
            local target_image="${registry}/${short_name}:${tag}"
            
            log_info "[$total_count] $source_image -> $target_image"
            
            if ! docker image inspect "$source_image" >/dev/null 2>&1; then
                log_info "  Pulling source image..."
                if ! pull_image_with_retry "$source_image" "$max_retries"; then
                    log_warn "  ✗ Failed to pull: $source_image"
                    failed_services+=("dep:$short_name")
                    continue
                fi
            fi
            
            if ! docker tag "$source_image" "$target_image"; then
                log_warn "  ✗ Failed to tag: $target_image"
                failed_services+=("dep:$short_name")
                continue
            fi
            
            if push_image_with_retry "$target_image" "$max_retries"; then
                log_info "  ✓ Pushed"
                success_count=$((success_count + 1))
            else
                failed_services+=("dep:$short_name")
            fi
        done
        echo
        
        # Phase 3: Push project services (unified tags)
        log_info "=== Phase 3: Project services (tag: $tag) ==="
        log_info "Main application images built from src/*"
        echo
        for service in "${FOUNDATION_SERVICES[@]}" "${DEPENDENT_SERVICES[@]}"; do
            total_count=$((total_count + 1))
            
            if push_service "$service" "$tag" "$registry" "$max_retries"; then
                log_info "  ✓ $service pushed"
                success_count=$((success_count + 1))
            else
                failed_services+=("$service")
            fi
        done
        echo
        
        # Phase 4: Push special images (multi-stage build targets, etc.)
        log_info "=== Phase 4: Special images (tag: $tag) ==="
        log_info "Images from multi-stage builds that don't have their own src/ directory"
        echo
        local special_images=(
            "backend-init"
        )
        for special in "${special_images[@]}"; do
            total_count=$((total_count + 1))
            local image_name="ai-infra-${special}:${tag}"
            local target_image="${registry}/${image_name}"
            
            log_info "[$total_count] $image_name -> $target_image"
            
            if ! docker image inspect "$image_name" >/dev/null 2>&1; then
                log_warn "  ✗ Source image not found: $image_name"
                log_info "    Hint: Build with 'docker compose build backend-init'"
                failed_services+=("special:$special")
                continue
            fi
            
            if ! docker tag "$image_name" "$target_image"; then
                log_warn "  ✗ Failed to tag: $target_image"
                failed_services+=("special:$special")
                continue
            fi
            
            if push_image_with_retry "$target_image" "$max_retries"; then
                log_info "  ✓ Pushed"
                success_count=$((success_count + 1))
            else
                failed_services+=("special:$special")
            fi
        done
        echo
    fi
    
    log_info "=========================================="
    log_info "Push completed: $success_count/$total_count successful"
    
    if [[ ${#failed_services[@]} -gt 0 ]]; then
        log_warn "Failed: ${failed_services[*]}"
        log_info "Check failure log: $FAILURE_LOG"
        return 1
    fi
    
    log_info "🚀 All images pushed successfully!"
    return 0
}

# Get dependency image mappings
get_dependency_mappings() {
    local mappings=(
        "confluentinc/cp-kafka:${KAFKA_VERSION:-7.5.0}|cp-kafka"
        "provectuslabs/kafka-ui:${KAFKAUI_VERSION:-latest}|kafka-ui"
        "postgres:${POSTGRES_VERSION:-15-alpine}|postgres"
        "redis:${REDIS_VERSION:-7-alpine}|redis"
        "redis/redisinsight:${REDISINSIGHT_VERSION:-2.68}|redisinsight"
        "chrislusf/seaweedfs:${SEAWEEDFS_VERSION:-3.80}|seaweedfs"
        "osixia/openldap:${OPENLDAP_VERSION:-stable}|openldap"
        "osixia/phpldapadmin:${PHPLDAPADMIN_VERSION:-stable}|phpldapadmin"
        "mysql:${MYSQL_VERSION:-8.0}|mysql"
        "victoriametrics/victoria-metrics:${VICTORIAMETRICS_VERSION:-v1.115.0}|victoria-metrics"
    )
    echo "${mappings[@]}"
}

# Push all dependency images
# Args: $1 = registry, $2 = tag
push_all_dependencies() {
    local registry="$1"
    local tag="${2:-${IMAGE_TAG:-latest}}"
    local max_retries="${3:-$DEFAULT_MAX_RETRIES}"
    
    if [[ -z "$registry" ]]; then
        log_error "Registry is required for push-dep"
        log_info "Usage: $0 push-dep <registry> [tag]"
        return 1
    fi
    
    # Ensure registry ends without trailing slash for consistent handling
    registry="${registry%/}"
    
    log_info "=========================================="
    log_info "Pushing all dependency images"
    log_info "=========================================="
    log_info "Registry: $registry"
    log_info "Tag: $tag"
    log_info "Max retries: $max_retries"
    echo
    
    local dependencies=($(get_dependency_mappings))
    local success_count=0
    local total_count=${#dependencies[@]}
    local failed_images=()
    
    for mapping in "${dependencies[@]}"; do
        local source_image="${mapping%%|*}"
        local short_name="${mapping##*|}"
        local target_image="${registry}/${short_name}:${tag}"
        
        log_info "Processing: $source_image"
        log_info "  → Target: $target_image"
        
        # 1. Pull or check source image (with retry)
        log_info "  [1/3] Checking source image..."
        if docker image inspect "$source_image" >/dev/null 2>&1; then
            log_info "  ✓ Image exists locally"
        else
            if ! pull_image_with_retry "$source_image" "$max_retries"; then
                failed_images+=("$source_image")
                echo
                continue
            fi
        fi
        
        # 2. Tag for registry
        log_info "  [2/3] Tagging image..."
        if ! docker tag "$source_image" "$target_image"; then
            log_failure "TAG" "$target_image" "Failed to tag from $source_image"
            failed_images+=("$source_image")
            echo
            continue
        fi
        log_info "  ✓ Tagged"
        
        # 3. Push to registry (with retry)
        log_info "  [3/3] Pushing image..."
        if push_image_with_retry "$target_image" "$max_retries"; then
            success_count=$((success_count + 1))
        else
            failed_images+=("$source_image")
        fi
        echo
    done
    
    log_info "=========================================="
    log_info "Dependency push completed: $success_count/$total_count successful"
    
    if [[ ${#failed_images[@]} -gt 0 ]]; then
        log_warn "Failed images: ${failed_images[*]}"
        log_info "Check failure log: $FAILURE_LOG"
        return 1
    fi
    
    log_info "🚀 All dependency images pushed successfully!"
    return 0
}

# Pull and tag dependencies from registry
# Args: $1 = registry, $2 = tag
pull_and_tag_dependencies() {
    local registry="$1"
    local tag="${2:-${IMAGE_TAG:-latest}}"
    local max_retries="${3:-$DEFAULT_MAX_RETRIES}"
    
    if [[ -z "$registry" ]]; then
        log_error "Registry is required"
        log_info "Usage: $0 deps-pull <registry> [tag]"
        return 1
    fi
    
    registry="${registry%/}"
    
    log_info "=========================================="
    log_info "Pulling dependencies from: $registry"
    log_info "=========================================="
    log_info "Tag: $tag"
    log_info "Max retries: $max_retries"
    echo
    
    local dependencies=($(get_dependency_mappings))
    local success_count=0
    local total_count=${#dependencies[@]}
    local failed_deps=()
    
    for mapping in "${dependencies[@]}"; do
        local source_image="${mapping%%|*}"
        local short_name="${mapping##*|}"
        local remote_image="${registry}/${short_name}:${tag}"
        
        log_info "Pulling: $remote_image"
        
        if pull_image_with_retry "$remote_image" "$max_retries"; then
            # Tag as original image name
            if docker tag "$remote_image" "$source_image"; then
                log_info "  ✓ Tagged: $source_image"
                success_count=$((success_count + 1))
            else
                log_failure "TAG" "$source_image" "Failed to tag from $remote_image"
                failed_deps+=("$short_name")
            fi
        else
            failed_deps+=("$short_name")
        fi
    done
    
    echo
    log_info "=========================================="
    log_info "Dependencies pull completed: $success_count/$total_count"
    
    if [[ ${#failed_deps[@]} -gt 0 ]]; then
        log_warn "Failed: ${failed_deps[*]}"
        log_info "Check failure log: $FAILURE_LOG"
        return 1
    fi
    
    log_info "🎉 All dependencies pulled successfully!"
    return 0
}

# ==============================================================================
# 3. Build Logic
# ==============================================================================

# Global flag for force rebuild (--no-cache)
FORCE_BUILD=false

# Prepare base build args
BASE_BUILD_ARGS=()
if [ -f "$ENV_EXAMPLE" ]; then
    while IFS='=' read -r key value; do
        [[ "$key" =~ ^#.*$ ]] && continue
        [[ -z "$key" ]] && continue
        curr_val="${!key}"
        if [ -n "$curr_val" ]; then
            # For proxy-related args, convert localhost/127.0.0.1 to host.docker.internal
            # This ensures buildkit container can access host's proxy
            if [[ "$key" =~ ^(HTTP_PROXY|HTTPS_PROXY|http_proxy|https_proxy)$ ]]; then
                curr_val="${curr_val//127.0.0.1/host.docker.internal}"
                curr_val="${curr_val//localhost/host.docker.internal}"
            fi
            BASE_BUILD_ARGS+=("--build-arg" "$key=$curr_val")
        fi
    done < <(grep -v '^#' "$ENV_EXAMPLE")
fi
BASE_BUILD_ARGS+=("--build-arg" "BUILD_ENV=${BUILD_ENV:-production}")

build_component() {
    local component="$1"
    local extra_args=("${@:2}") # Capture all remaining arguments
    local component_dir="$SRC_DIR/$component"
    local tag="${IMAGE_TAG:-latest}"
    local build_id="${CURRENT_BUILD_ID:-$(generate_build_id)}"
    
    if [ ! -d "$component_dir" ]; then
        log_error "Component directory not found: $component_dir"
        return 1
    fi

    # ===== Build Cache Check =====
    local rebuild_reason=$(need_rebuild "$component" "$tag")
    if [[ "$rebuild_reason" == "NO_CHANGE" ]]; then
        log_cache "⏭️  Skipping $component (no changes detected)"
        log_build_history "$build_id" "$component" "$tag" "SKIPPED" "NO_CHANGE"
        return 0
    fi
    
    log_cache "🔄 Rebuilding $component: $rebuild_reason"

    # Check if template exists and render it
    local template_file="$component_dir/Dockerfile.tpl"
    if [ -f "$template_file" ]; then
        log_info "Rendering template for $component..."
        if ! render_template "$template_file"; then
            log_error "Failed to render template for $component"
            log_build_history "$build_id" "$component" "$tag" "FAILED" "TEMPLATE_ERROR"
            return 1
        fi
    fi

    # Check for dependency configuration (External Image)
    local dep_conf="$component_dir/dependency.conf"
    if [ -f "$dep_conf" ]; then
        # Get first non-comment, non-empty line
        local upstream_image=$(grep -v '^#' "$dep_conf" | grep -v '^[[:space:]]*$' | head -n 1 | tr -d '[:space:]')
        if [ -z "$upstream_image" ]; then
            log_error "Empty dependency config for $component"
            return 1
        fi
        
        # Check if this component also has a Dockerfile (custom build based on dependency)
        if [ -f "$component_dir/Dockerfile" ]; then
            # This is a custom build that uses a base image from dependency.conf
            # The Dockerfile should reference the upstream image, we just need to ensure it's available
            log_info "Processing $component: dependency + custom Dockerfile"
            log_info "  Base image: $upstream_image"
            
            # Pull the base image first to ensure it's available for the build
            if ! pull_image_with_retry "$upstream_image" 3 5; then
                log_error "✗ Failed to pull base image $upstream_image for $component"
                log_build_history "$build_id" "$component" "$tag" "FAILED" "BASE_PULL_ERROR"
                return 1
            fi
            log_info "✓ Base image ready: $upstream_image"
            # Continue to Dockerfile build below (don't return)
        else
            # Pure dependency - just pull and tag
            local target_image="ai-infra-$component:${tag}"
            if [ -n "$PRIVATE_REGISTRY" ]; then
                target_image="$PRIVATE_REGISTRY/$target_image"
            fi
            
            log_info "Processing dependency $component: $upstream_image -> $target_image"
            
            if pull_image_with_retry "$upstream_image" 3 5; then
                if docker tag "$upstream_image" "$target_image"; then
                    log_info "✓ Dependency ready: $target_image"
                    log_build_history "$build_id" "$component" "$tag" "SUCCESS" "DEPENDENCY_PULLED"
                    return 0
                else
                    log_error "✗ Failed to tag $upstream_image"
                    log_build_history "$build_id" "$component" "$tag" "FAILED" "TAG_ERROR"
                    return 1
                fi
            else
                log_error "✗ Failed to pull $upstream_image after retries"
                log_build_history "$build_id" "$component" "$tag" "FAILED" "PULL_ERROR"
                return 1
            fi
        fi
    fi
    
    if [ ! -f "$component_dir/Dockerfile" ]; then
        log_warn "No Dockerfile or dependency.conf in $component, skipping..."
        return 0
    fi

    # Calculate service hash for build label
    local service_hash=$(calculate_service_hash "$component")

    # Check for build-targets.conf
    local targets_file="$component_dir/build-targets.conf"
    local targets=()
    local images=()
    
    if [ -f "$targets_file" ]; then
        while read -r target image_suffix || [ -n "$target" ]; do
            [[ "$target" =~ ^#.*$ ]] && continue
            [[ -z "$target" ]] && continue
            targets+=("$target")
            images+=("$image_suffix")
        done < "$targets_file"
    else
        targets+=("default")
        images+=("ai-infra-$component")
    fi

    for i in "${!targets[@]}"; do
        local target="${targets[$i]}"
        local image_name="${images[$i]}"
        local full_image_name="${image_name}:${tag}"
        
        if [ -n "$PRIVATE_REGISTRY" ]; then
            full_image_name="$PRIVATE_REGISTRY/$full_image_name"
        fi
        
        log_info "Building $component [$target] -> $full_image_name"
        
        local cmd=("docker" "build")
        
        # Add --no-cache if force build is enabled
        if [[ "$FORCE_BUILD" == "true" ]]; then
            cmd+=("--no-cache")
        fi
        
        # Add build cache labels for incremental builds
        cmd+=("--label" "build.hash=$service_hash")
        cmd+=("--label" "build.id=$build_id")
        cmd+=("--label" "build.timestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ)")
        cmd+=("--label" "build.component=$component")
        
        cmd+=("${BASE_BUILD_ARGS[@]}" "${extra_args[@]}" "-t" "$full_image_name" "-f" "$component_dir/Dockerfile")
        
        if [ "$target" != "default" ]; then
            cmd+=("--target" "$target")
        fi
        
        # Add build context (project root)
        cmd+=("$SCRIPT_DIR")
        
        if "${cmd[@]}"; then
            log_info "✓ Build success: $full_image_name"
            log_build_history "$build_id" "$component" "$tag" "SUCCESS" "$rebuild_reason"
            save_service_build_info "$component" "$tag" "$build_id" "$service_hash"
        else
            log_error "✗ Build failed: $full_image_name"
            log_build_history "$build_id" "$component" "$tag" "FAILED" "BUILD_ERROR"
            return 1
        fi
    done
}

discover_services() {
    log_info "Discovering components in $SRC_DIR..."
    DEPENDENCY_SERVICES=()
    FOUNDATION_SERVICES=()
    DEPENDENT_SERVICES=()

    # Optional components that are disabled by default
    # These require separate initialization (e.g., ./build.sh init-safeline)
    local safeline_enabled="${SAFELINE_ENABLED:-false}"

    # Use find to avoid issues if directory is empty and sort for deterministic order
    while IFS= read -r dir; do
        local component=$(basename "$dir")
        
        # Skip optional components when disabled
        if [[ "$component" == "safeline" ]] && [[ "$safeline_enabled" != "true" ]]; then
            log_info "Skipping optional component: $component (SAFELINE_ENABLED=false)"
            continue
        fi
        
        # Check for Dockerfile first (takes priority for build phase classification)
        # Even if dependency.conf exists, if there's a Dockerfile, it's a buildable component
        if [ -f "$dir/Dockerfile" ]; then
            local phase="dependent" # Default phase
            
            # Check for build.conf override
            if [ -f "$dir/build.conf" ]; then
                local conf_phase=$(grep "^BUILD_PHASE=" "$dir/build.conf" | cut -d= -f2 | tr -d '[:space:]')
                if [ -n "$conf_phase" ]; then
                    phase="$conf_phase"
                fi
            fi
            
            if [ "$phase" == "foundation" ]; then
                FOUNDATION_SERVICES+=("$component")
            else
                DEPENDENT_SERVICES+=("$component")
            fi
            continue
        fi
        
        # Check for dependency.conf only (Pure External Image, no custom Dockerfile)
        if [ -f "$dir/dependency.conf" ]; then
            DEPENDENCY_SERVICES+=("$component")
            continue
        fi
    done < <(find "$SRC_DIR" -mindepth 1 -maxdepth 1 -type d | sort)
    
    log_info "Found ${#DEPENDENCY_SERVICES[@]} dependency services: ${DEPENDENCY_SERVICES[*]}"
    log_info "Found ${#FOUNDATION_SERVICES[@]} foundation services: ${FOUNDATION_SERVICES[*]}"
    log_info "Found ${#DEPENDENT_SERVICES[@]} dependent services: ${DEPENDENT_SERVICES[*]}"
}

# Build services in parallel with controlled concurrency
# Usage: build_parallel <service1> <service2> ... [-- extra_build_args]
build_parallel() {
    local services=()
    local extra_args=()
    local found_separator=false
    
    # Parse arguments: services before --, extra args after --
    for arg in "$@"; do
        if [[ "$arg" == "--" ]]; then
            found_separator=true
            continue
        fi
        if [[ "$found_separator" == "true" ]]; then
            extra_args+=("$arg")
        else
            services+=("$arg")
        fi
    done
    
    if [[ ${#services[@]} -eq 0 ]]; then
        log_warn "No services to build in parallel"
        return 0
    fi
    
    local max_jobs="${PARALLEL_JOBS:-4}"
    local total=${#services[@]}
    local completed=0
    local failed=0
    local pids=()
    local service_map=()
    
    log_parallel "🚀 Starting parallel build: $total services, max $max_jobs concurrent jobs"
    log_parallel "Services: ${services[*]}"
    
    # Build services in batches
    for service in "${services[@]}"; do
        # Wait if we've reached max concurrent jobs
        while [[ ${#pids[@]} -ge $max_jobs ]]; do
            # Wait for any job to complete
            local new_pids=()
            for i in "${!pids[@]}"; do
                if kill -0 "${pids[$i]}" 2>/dev/null; then
                    new_pids+=("${pids[$i]}")
                else
                    wait "${pids[$i]}" 2>/dev/null
                    local exit_code=$?
                    local svc="${service_map[$i]}"
                    if [[ $exit_code -eq 0 ]]; then
                        completed=$((completed + 1))
                        log_parallel "✓ Completed: $svc ($completed/$total)"
                    else
                        failed=$((failed + 1))
                        log_error "✗ Failed: $svc (exit code: $exit_code)"
                    fi
                fi
            done
            pids=("${new_pids[@]}")
            
            # Update service_map to match pids
            local new_map=()
            for i in "${!pids[@]}"; do
                # Find corresponding service
                for j in "${!service_map[@]}"; do
                    if kill -0 "${pids[$i]}" 2>/dev/null && [[ "${pids[$i]}" == "$(jobs -p | head -n $((i+1)) | tail -1)" ]]; then
                        new_map+=("${service_map[$j]}")
                        break
                    fi
                done
            done
            service_map=("${new_map[@]}")
            
            sleep 0.5
        done
        
        # Start new build job in background
        log_parallel "🔨 Starting: $service"
        (
            if [[ ${#extra_args[@]} -gt 0 ]]; then
                build_component "$service" "${extra_args[@]}"
            else
                build_component "$service"
            fi
        ) &
        pids+=($!)
        service_map+=("$service")
    done
    
    # Wait for remaining jobs
    log_parallel "⏳ Waiting for remaining builds to complete..."
    for i in "${!pids[@]}"; do
        wait "${pids[$i]}" 2>/dev/null
        local exit_code=$?
        local svc="${service_map[$i]}"
        if [[ $exit_code -eq 0 ]]; then
            completed=$((completed + 1))
            log_parallel "✓ Completed: $svc ($completed/$total)"
        else
            failed=$((failed + 1))
            log_error "✗ Failed: $svc (exit code: $exit_code)"
        fi
    done
    
    log_parallel "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_parallel "📊 Parallel Build Summary: $completed succeeded, $failed failed, $total total"
    log_parallel "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    if [[ $failed -gt 0 ]]; then
        return 1
    fi
    return 0
}

# Parallel build for a specific platform
# Usage: build_parallel_for_platform <platform> <services...> [-- extra_args...]
# Example: build_parallel_for_platform linux/amd64 nginx backend frontend -- --build-arg FOO=bar
build_parallel_for_platform() {
    local platform="$1"
    shift
    
    local services=()
    local extra_args=()
    local found_separator=false
    
    # Parse arguments: services before --, extra args after --
    for arg in "$@"; do
        if [[ "$arg" == "--" ]]; then
            found_separator=true
            continue
        fi
        if [[ "$found_separator" == "true" ]]; then
            extra_args+=("$arg")
        else
            services+=("$arg")
        fi
    done
    
    if [[ ${#services[@]} -eq 0 ]]; then
        log_warn "No services to build in parallel"
        return 0
    fi
    
    local arch_name="${platform##*/}"
    local max_jobs="${PARALLEL_JOBS:-4}"
    local total=${#services[@]}
    local completed=0
    local failed=0
    local pids=()
    local service_map=()
    
    log_parallel "🚀 [$arch_name] Starting parallel build: $total services, max $max_jobs concurrent jobs"
    log_parallel "[$arch_name] Services: ${services[*]}"
    
    # Create log directory for parallel build output
    local parallel_log_dir="/tmp/ai-infra-parallel-build-$$"
    mkdir -p "$parallel_log_dir"
    
    # Build services in batches
    for service in "${services[@]}"; do
        # Wait if we've reached max concurrent jobs
        while [[ ${#pids[@]} -ge $max_jobs ]]; do
            # Wait for any job to complete
            local new_pids=()
            local new_map=()
            for i in "${!pids[@]}"; do
                if kill -0 "${pids[$i]}" 2>/dev/null; then
                    new_pids+=("${pids[$i]}")
                    new_map+=("${service_map[$i]}")
                else
                    wait "${pids[$i]}" 2>/dev/null
                    local exit_code=$?
                    local svc="${service_map[$i]}"
                    if [[ $exit_code -eq 0 ]]; then
                        completed=$((completed + 1))
                        log_parallel "[$arch_name] ✓ Completed: $svc ($completed/$total)"
                    else
                        failed=$((failed + 1))
                        log_error "[$arch_name] ✗ Failed: $svc (exit code: $exit_code)"
                        # Show tail of log file for failed build
                        local logfile="$parallel_log_dir/${svc}-${arch_name}.log"
                        if [[ -f "$logfile" ]]; then
                            log_error "[$arch_name]    Last 20 lines of build log:"
                            tail -20 "$logfile" | while read line; do
                                log_error "[$arch_name]    $line"
                            done
                        fi
                    fi
                fi
            done
            pids=("${new_pids[@]}")
            service_map=("${new_map[@]}")
            sleep 0.5
        done
        
        # Start new build job in background with output logged to file
        log_parallel "[$arch_name] 🔨 Starting: $service"
        local logfile="$parallel_log_dir/${service}-${arch_name}.log"
        (
            if [[ ${#extra_args[@]} -gt 0 ]]; then
                build_component_for_platform "$service" "$platform" "${extra_args[@]}" 2>&1 | tee "$logfile"
                exit ${PIPESTATUS[0]}
            else
                build_component_for_platform "$service" "$platform" 2>&1 | tee "$logfile"
                exit ${PIPESTATUS[0]}
            fi
        ) &
        pids+=($!)
        service_map+=("$service")
    done
    
    # Wait for remaining jobs
    log_parallel "[$arch_name] ⏳ Waiting for remaining builds to complete..."
    for i in "${!pids[@]}"; do
        wait "${pids[$i]}" 2>/dev/null
        local exit_code=$?
        local svc="${service_map[$i]}"
        if [[ $exit_code -eq 0 ]]; then
            completed=$((completed + 1))
            log_parallel "[$arch_name] ✓ Completed: $svc ($completed/$total)"
        else
            failed=$((failed + 1))
            log_error "[$arch_name] ✗ Failed: $svc (exit code: $exit_code)"
            # Show tail of log file for failed build
            local logfile="$parallel_log_dir/${svc}-${arch_name}.log"
            if [[ -f "$logfile" ]]; then
                log_error "[$arch_name]    Last 20 lines of build log:"
                tail -20 "$logfile" | while read line; do
                    log_error "[$arch_name]    $line"
                done
            fi
        fi
    done
    
    log_parallel "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_parallel "[$arch_name] 📊 Parallel Build Summary: $completed succeeded, $failed failed, $total total"
    log_parallel "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # =========================================================================
    # POST-BUILD VERIFICATION: Check if images were actually loaded to daemon
    # 
    # Due to potential race conditions with docker-container driver's --load,
    # some images may not be properly loaded. We verify and retry if needed.
    # =========================================================================
    local tag="${IMAGE_TAG:-latest}"
    local missing_services=()
    
    log_parallel "[$arch_name] 🔍 Verifying images were loaded to Docker daemon..."
    for service in "${services[@]}"; do
        local expected_image="ai-infra-${service}:${tag}-${arch_name}"
        if ! docker image inspect "$expected_image" >/dev/null 2>&1; then
            missing_services+=("$service")
            log_warn "[$arch_name]    ⚠️  Image not found: $expected_image"
        else
            # Also verify the architecture is correct
            local loaded_arch=$(docker image inspect "$expected_image" --format '{{.Architecture}}' 2>/dev/null || echo "unknown")
            if [[ "$loaded_arch" != "$arch_name" ]]; then
                missing_services+=("$service")
                log_warn "[$arch_name]    ⚠️  Image has wrong architecture: $expected_image (expected: $arch_name, got: $loaded_arch)"
            fi
        fi
    done
    
    # If some images are missing, retry them serially
    if [[ ${#missing_services[@]} -gt 0 ]]; then
        log_warn "[$arch_name] ⚠️  ${#missing_services[@]} images not found in Docker daemon after parallel build"
        log_parallel "[$arch_name] 🔄 Retrying missing services serially..."
        
        local retry_failed=0
        for service in "${missing_services[@]}"; do
            log_parallel "[$arch_name]    → Retrying: $service"
            if [[ ${#extra_args[@]} -gt 0 ]]; then
                if build_component_for_platform "$service" "$platform" "${extra_args[@]}"; then
                    # Verify image exists after retry
                    local expected_image="ai-infra-${service}:${tag}-${arch_name}"
                    if docker image inspect "$expected_image" >/dev/null 2>&1; then
                        log_parallel "[$arch_name]    ✓ Retry succeeded: $service"
                    else
                        log_error "[$arch_name]    ✗ Retry completed but image still not found: $service"
                        retry_failed=$((retry_failed + 1))
                    fi
                else
                    log_error "[$arch_name]    ✗ Retry failed: $service"
                    retry_failed=$((retry_failed + 1))
                fi
            else
                if build_component_for_platform "$service" "$platform"; then
                    # Verify image exists after retry
                    local expected_image="ai-infra-${service}:${tag}-${arch_name}"
                    if docker image inspect "$expected_image" >/dev/null 2>&1; then
                        log_parallel "[$arch_name]    ✓ Retry succeeded: $service"
                    else
                        log_error "[$arch_name]    ✗ Retry completed but image still not found: $service"
                        retry_failed=$((retry_failed + 1))
                    fi
                else
                    log_error "[$arch_name]    ✗ Retry failed: $service"
                    retry_failed=$((retry_failed + 1))
                fi
            fi
        done
        
        if [[ $retry_failed -gt 0 ]]; then
            failed=$((failed + retry_failed))
            log_error "[$arch_name] ❌ $retry_failed service(s) failed even after serial retry"
        else
            log_parallel "[$arch_name] ✓ All missing services recovered via serial retry"
            failed=0  # Reset failed count since we recovered
        fi
    fi
    
    # Clean up log directory if all builds succeeded
    if [[ $failed -eq 0 ]]; then
        rm -rf "$parallel_log_dir" 2>/dev/null || true
    else
        log_warn "[$arch_name] Build logs saved to: $parallel_log_dir"
    fi
    
    if [[ $failed -gt 0 ]]; then
        return 1
    fi
    return 0
}

# Multi-Architecture Build Support
# Uses docker buildx to build images for multiple platforms
# Requires: docker buildx (included in Docker Desktop, or install separately)

# Check and setup buildx builder
setup_buildx_builder() {
    local builder_name="ai-infra-multiarch"
    local use_container_driver="${1:-false}"  # Default to docker driver for better mirror support
    
    # Check if buildx is available
    if ! docker buildx version >/dev/null 2>&1; then
        log_error "docker buildx is not available"
        log_info "Please install Docker Desktop or docker-buildx plugin"
        return 1
    fi
    
    # Ensure QEMU is installed for cross-platform builds
    _ensure_qemu_installed
    
    # For simple single-platform builds, prefer the default docker driver
    # It inherits registry-mirrors from daemon.json
    if [[ "$use_container_driver" != "true" ]]; then
        # Use default docker driver (desktop-linux on Mac)
        local default_builder=$(docker buildx ls 2>/dev/null | grep -E "^desktop-linux|^default" | head -1 | awk '{print $1}')
        if [[ -n "$default_builder" ]]; then
            docker buildx use "$default_builder" 2>/dev/null
            log_info "✓ Using buildx builder: $default_builder (docker driver, inherits mirror config)"
            return 0
        fi
    fi
    
    # For docker-container driver (needed for some cross-platform scenarios)
    # Create with registry mirror configuration
    if docker buildx inspect "$builder_name" >/dev/null 2>&1; then
        log_info "Using existing buildx builder: $builder_name"
    else
        log_info "Creating new buildx builder: $builder_name"
        
        # Generate buildkit config with registry mirrors from daemon.json
        local buildkit_config="/tmp/buildkitd-${builder_name}.toml"
        _generate_buildkit_config "$buildkit_config"
        
        local create_args=("--name" "$builder_name" "--driver" "docker-container" "--bootstrap")
        if [[ -f "$buildkit_config" ]]; then
            create_args+=("--config" "$buildkit_config")
            log_info "Using registry mirrors from Docker daemon config"
        fi
        
        if ! docker buildx create "${create_args[@]}"; then
            log_error "Failed to create buildx builder"
            return 1
        fi
    fi
    
    # Use this builder
    docker buildx use "$builder_name"
    log_info "✓ Buildx builder ready: $builder_name"
    return 0
}

# Ensure QEMU user-mode emulation is installed for cross-platform builds
_ensure_qemu_installed() {
    local host_arch=$(uname -m)
    local need_qemu=false
    
    # Check if we need QEMU based on host architecture and target platforms
    if [[ -n "$BUILD_PLATFORMS" ]]; then
        case "$host_arch" in
            arm64|aarch64)
                # On ARM, need QEMU for amd64
                if echo "$BUILD_PLATFORMS" | grep -qE "amd64|x86_64"; then
                    need_qemu=true
                fi
                ;;
            x86_64|amd64)
                # On x86, need QEMU for arm64
                if echo "$BUILD_PLATFORMS" | grep -qE "arm64|aarch64"; then
                    need_qemu=true
                fi
                ;;
        esac
    fi
    
    if [[ "$need_qemu" != "true" ]]; then
        return 0
    fi
    
    # Test if QEMU is working
    local test_platform
    case "$host_arch" in
        arm64|aarch64) test_platform="linux/amd64" ;;
        x86_64|amd64) test_platform="linux/arm64" ;;
    esac
    
    log_info "Checking QEMU emulation for cross-platform builds..."
    
    if docker run --rm --platform "$test_platform" alpine:3.18 uname -m >/dev/null 2>&1; then
        log_info "✓ QEMU emulation is working"
        return 0
    fi
    
    log_info "Installing QEMU user-mode emulation for cross-platform builds..."
    if docker run --rm --privileged tonistiigi/binfmt --install all >/dev/null 2>&1; then
        log_info "✓ QEMU emulation installed successfully"
        return 0
    else
        log_warn "⚠️  Failed to install QEMU emulation"
        log_warn "   Cross-platform builds may fail"
        log_warn "   Try manually: docker run --rm --privileged tonistiigi/binfmt --install all"
        return 1
    fi
}

# Generate advanced buildkit config with mirrors, proxy and insecure registries
# This is critical for environments with custom proxies and registry mirrors
_generate_buildkit_config_with_proxy() {
    local output_file="$1"
    local daemon_json="${HOME}/.docker/daemon.json"
    
    # Initialize config with basic settings
    cat > "$output_file" << 'EOF'
# Auto-generated buildkit config for registry mirrors and proxy support
debug = false

[worker.oci]
  # Use overlayfs snapshotter for better performance
EOF
    
    # Extract and add proxy settings if present in daemon.json
    if [[ -f "$daemon_json" ]]; then
        local http_proxy=$(grep -o '"httpProxy": "[^"]*"' "$daemon_json" 2>/dev/null | cut -d'"' -f4)
        local https_proxy=$(grep -o '"httpsProxy": "[^"]*"' "$daemon_json" 2>/dev/null | cut -d'"' -f4)
        local no_proxy=$(grep -o '"noProxy": "[^"]*"' "$daemon_json" 2>/dev/null | cut -d'"' -f4)
        
        if [[ -n "$http_proxy" || -n "$https_proxy" ]]; then
            cat >> "$output_file" << EOF

[buildkitd]
  # Proxy settings for pulling images and running RUN commands
EOF
            if [[ -n "$http_proxy" ]]; then
                echo "  httpProxy = \"$http_proxy\"" >> "$output_file"
            fi
            if [[ -n "$https_proxy" ]]; then
                echo "  httpsProxy = \"$https_proxy\"" >> "$output_file"
            fi
            if [[ -n "$no_proxy" ]]; then
                echo "  noProxy = \"$no_proxy\"" >> "$output_file"
            fi
        fi
        
        # Extract registry mirrors - these are the acceleration mirrors configured in daemon.json
        local mirrors=$(grep -A10 '"registry-mirrors"' "$daemon_json" 2>/dev/null | grep -oE 'https?://[^"]+')
        local is_insecure_registry=$(grep -o '"insecure-registries"' "$daemon_json" 2>/dev/null | wc -l)
        
        if [[ -n "$mirrors" ]]; then
            # Get first mirror as the primary acceleration endpoint
            local first_mirror=$(echo "$mirrors" | head -1)
            local mirror_host=$(echo "$first_mirror" | sed -E 's|https?://||' | sed 's|/.*||')
            local is_http=$(echo "$first_mirror" | grep -c "^http://")
            
            # Configure docker.io to use the primary mirror
            # This tells BuildKit to use the mirror for docker.io requests
            cat >> "$output_file" << EOF

# Registry configuration for Docker Hub acceleration
[registry."docker.io"]
  # Use the configured mirror as the primary endpoint
  # BuildKit will try this mirror first for docker.io requests
  mirrors = ["$mirror_host"]

# Configure the mirror itself as a registry endpoint
[registry."$mirror_host"]
EOF
            if [[ "$is_http" -gt 0 ]]; then
                echo "  http = true" >> "$output_file"
            fi
            
            # Mark as insecure if configured in daemon.json
            if grep -q "\"$mirror_host\"" "$daemon_json"; then
                echo "  insecure = true" >> "$output_file"
            fi
        fi
        
        # Also configure registry-1.docker.io (used by some images) to use the same mirror
        if [[ -n "$mirrors" ]]; then
            local first_mirror=$(echo "$mirrors" | head -1)
            local mirror_host=$(echo "$first_mirror" | sed -E 's|https?://||' | sed 's|/.*||')
            
            cat >> "$output_file" << EOF

# Also redirect registry-1.docker.io to mirror
[registry."registry-1.docker.io"]
  mirrors = ["$mirror_host"]
EOF
        fi
    fi
    
    log_info "Generated advanced buildkit config: $output_file"
    return 0
}

# Generate buildkit config with registry mirrors from Docker daemon.json
# (Legacy version, kept for compatibility)
_generate_buildkit_config() {
    local output_file="$1"
    local daemon_json="${HOME}/.docker/daemon.json"
    
    # Check if daemon.json exists and has registry-mirrors
    if [[ ! -f "$daemon_json" ]]; then
        return 1
    fi
    
    # Extract registry mirrors (simple parsing)
    local mirrors=$(grep -A10 '"registry-mirrors"' "$daemon_json" 2>/dev/null | grep -oE 'https?://[^"]+' | head -3)
    if [[ -z "$mirrors" ]]; then
        return 1
    fi
    
    # Generate buildkit TOML config
    cat > "$output_file" << 'EOF'
# Auto-generated buildkit config for registry mirrors
debug = false

[registry."docker.io"]
EOF
    
    # Add mirrors
    echo "  mirrors = [" >> "$output_file"
    local first=true
    for mirror in $mirrors; do
        # Extract hostname from URL
        local host=$(echo "$mirror" | sed -E 's|https?://||' | sed 's|/.*||')
        if [[ "$first" == "true" ]]; then
            echo "    \"$host\"" >> "$output_file"
            first=false
        else
            echo "    ,\"$host\"" >> "$output_file"
        fi
    done
    echo "  ]" >> "$output_file"
    
    # Add insecure registry configs
    for mirror in $mirrors; do
        local host=$(echo "$mirror" | sed -E 's|https?://||' | sed 's|/.*||')
        local is_http=$(echo "$mirror" | grep -c "^http://")
        cat >> "$output_file" << EOF

[registry."$host"]
  http = $([ "$is_http" -gt 0 ] && echo "true" || echo "false")
  insecure = true
EOF
    done
    
    log_info "Generated buildkit config: $output_file"
    return 0
}

# Build a single component for multiple architectures
# Enhanced version with dependency.conf base image support
# Usage: build_component_multiarch <component> <platforms> [extra_args...]
# Example: build_component_multiarch backend "linux/amd64,linux/arm64"
#          build_component_multiarch apphub "linux/amd64,linux/arm64"
build_component_multiarch() {
    local component="$1"
    local platforms="$2"
    local extra_args=("${@:3}")
    local component_dir="$SRC_DIR/$component"
    local tag="${IMAGE_TAG:-latest}"
    local build_id="${CURRENT_BUILD_ID:-$(generate_build_id)}"
    
    if [ ! -d "$component_dir" ]; then
        log_error "Component directory not found: $component_dir"
        return 1
    fi
    
    # ===== Build Cache Check =====
    if [[ "$FORCE_BUILD" != "true" ]]; then
        local rebuild_reason=$(need_rebuild "$component" "$tag")
        if [[ "$rebuild_reason" == "NO_CHANGE" ]]; then
            log_cache "⏭️  Skipping $component (no changes detected)"
            log_build_history "$build_id" "$component" "$tag" "SKIPPED" "NO_CHANGE (multiarch)"
            return 0
        fi
        log_cache "🔄 Rebuilding $component: $rebuild_reason"
    fi
    
    # Check for template and render if needed
    local template_file="$component_dir/Dockerfile.tpl"
    if [ -f "$template_file" ]; then
        log_info "Rendering template for $component..."
        if ! render_template "$template_file"; then
            log_error "Failed to render template for $component"
            log_build_history "$build_id" "$component" "$tag" "FAILED" "TEMPLATE_ERROR"
            return 1
        fi
    fi
    
    # ===== Dependency Configuration (External Base Image) =====
    local dep_conf="$component_dir/dependency.conf"
    if [ -f "$dep_conf" ]; then
        local upstream_image=$(grep -v '^#' "$dep_conf" | grep -v '^[[:space:]]*$' | head -n 1 | tr -d '[:space:]')
        if [ -z "$upstream_image" ]; then
            log_error "Empty dependency config for $component"
            return 1
        fi
        
        # Check if this component also has a Dockerfile (custom build based on dependency)
        if [ -f "$component_dir/Dockerfile" ]; then
            log_info "Processing $component: dependency + custom Dockerfile"
            log_info "  Base image: $upstream_image"
            
            # Pull the base image for all target platforms first
            log_info "Pulling base image for all target platforms..."
            IFS=',' read -ra platform_array <<< "$platforms"
            for platform in "${platform_array[@]}"; do
                local arch_name="${platform##*/}"
                log_info "  Pulling $upstream_image for $arch_name..."
                if ! docker pull --platform "$platform" "$upstream_image" >/dev/null 2>&1; then
                    log_warn "  Retrying pull for $arch_name..."
                    if ! docker pull --platform "$platform" "$upstream_image"; then
                        log_error "✗ Failed to pull base image $upstream_image for $arch_name"
                        log_build_history "$build_id" "$component" "$tag" "FAILED" "BASE_PULL_ERROR ($arch_name)"
                        return 1
                    fi
                fi
                log_info "  ✓ $arch_name ready"
            done
            log_info "✓ Base image ready for all platforms: $upstream_image"
            # Continue to Dockerfile build below (don't return)
        else
            # Pure dependency - just pull and tag for all target platforms
            local target_image="ai-infra-$component:${tag}"
            if [ -n "$PRIVATE_REGISTRY" ]; then
                target_image="$PRIVATE_REGISTRY/$target_image"
            fi
            
            log_info "Processing dependency $component: $upstream_image -> $target_image (multiarch)"
            
            IFS=',' read -ra platform_array <<< "$platforms"
            for platform in "${platform_array[@]}"; do
                local arch_name="${platform##*/}"
                log_info "  [$arch_name] Pulling and tagging..."
                if docker pull --platform "$platform" "$upstream_image" >/dev/null 2>&1; then
                    docker tag "$upstream_image" "$target_image" >/dev/null 2>&1
                    log_info "  [$arch_name] ✓ Ready: $target_image"
                else
                    log_error "  [$arch_name] ✗ Failed to pull $upstream_image"
                    log_build_history "$build_id" "$component" "$tag" "FAILED" "PULL_ERROR ($arch_name)"
                    return 1
                fi
            done
            
            log_build_history "$build_id" "$component" "$tag" "SUCCESS" "DEPENDENCY_PULLED (multiarch)"
            return 0
        fi
    fi
    
    if [ ! -f "$component_dir/Dockerfile" ]; then
        log_warn "No Dockerfile or dependency.conf in $component, skipping..."
        return 0
    fi
    
    # Calculate service hash for build label
    local service_hash=$(calculate_service_hash "$component")
    
    # Check for build-targets.conf
    local targets_file="$component_dir/build-targets.conf"
    local targets=()
    local images=()
    
    if [ -f "$targets_file" ]; then
        while read -r target image_suffix || [ -n "$target" ]; do
            [[ "$target" =~ ^#.*$ ]] && continue
            [[ -z "$target" ]] && continue
            targets+=("$target")
            images+=("$image_suffix")
        done < "$targets_file"
    else
        targets+=("default")
        images+=("ai-infra-$component")
    fi
    
    for i in "${!targets[@]}"; do
        local target="${targets[$i]}"
        local image_name="${images[$i]}"
        local full_image_name="${image_name}:${tag}"
        
        if [ -n "$PRIVATE_REGISTRY" ]; then
            full_image_name="$PRIVATE_REGISTRY/$full_image_name"
        fi
        
        log_info "Building $component [$target] for platforms: $platforms -> $full_image_name"
        
        local cmd=("docker" "buildx" "build")
        
        # Multi-platform specification
        cmd+=("--platform" "$platforms")
        
        # Output to local docker images (load only works for single platform)
        # For multi-platform, we need to use --output type=oci or push to registry
        # Here we use --output type=docker for single platform compatibility
        # or --output type=image,push=false for multi-platform local storage
        
        # Count platforms
        local platform_count=$(echo "$platforms" | tr ',' '\n' | wc -l | tr -d ' ')
        
        if [[ $platform_count -eq 1 ]]; then
            # Single platform: can use --load
            cmd+=("--load")
        else
            # Multi-platform: output to OCI tarball in output directory
            local output_dir="${MULTIARCH_OUTPUT_DIR:-./multiarch-images}"
            mkdir -p "$output_dir"
            local safe_name=$(echo "$full_image_name" | sed 's|/|-|g' | sed 's|:|_|g')
            cmd+=("--output" "type=oci,dest=${output_dir}/${safe_name}.tar")
        fi
        
        # Add --no-cache if force build is enabled
        if [[ "$FORCE_BUILD" == "true" ]]; then
            cmd+=("--no-cache")
        fi
        
        # Add build cache labels
        cmd+=("--label" "build.hash=$service_hash")
        cmd+=("--label" "build.id=$build_id")
        cmd+=("--label" "build.timestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ)")
        cmd+=("--label" "build.component=$component")
        cmd+=("--label" "build.platforms=$platforms")
        
        cmd+=("${BASE_BUILD_ARGS[@]}" "${extra_args[@]}" "-t" "$full_image_name" "-f" "$component_dir/Dockerfile")
        
        if [ "$target" != "default" ]; then
            cmd+=("--target" "$target")
        fi
        
        # Add build context (project root)
        cmd+=("$SCRIPT_DIR")
        
        log_info "Executing: ${cmd[*]}"
        
        if "${cmd[@]}"; then
            log_info "✓ Multi-arch build success: $full_image_name ($platforms)"
            log_build_history "$build_id" "$component" "$tag" "SUCCESS" "BUILT (multiarch: $platforms)"
            save_service_build_info "$component" "$tag" "$build_id" "$service_hash"
        else
            log_error "✗ Multi-arch build failed: $full_image_name"
            log_build_history "$build_id" "$component" "$tag" "FAILED" "BUILD_ERROR (multiarch)"
            return 1
        fi
    done
    
    return 0
}

# Build all services for multiple architectures
# Usage: build_multiarch [platforms] [--force]
# Example: build_multiarch "linux/amd64,linux/arm64"
#          build_multiarch amd64,arm64 --force
build_multiarch() {
    local platforms="${1:-linux/amd64,linux/arm64}"
    local force="${2:-false}"
    
    # Normalize platform format
    if [[ "$platforms" != *"linux/"* ]]; then
        # Convert short form (amd64,arm64) to full form (linux/amd64,linux/arm64)
        platforms=$(echo "$platforms" | sed 's/\([^,]*\)/linux\/\1/g')
    fi
    
    # Show help
    if [[ "$1" == "--help" ]] || [[ "$1" == "-h" ]]; then
        echo "Usage: $0 build-multiarch [platforms] [--force]"
        echo ""
        echo "Arguments:"
        echo "  platforms   Comma-separated list of target platforms"
        echo "              Default: linux/amd64,linux/arm64"
        echo "              Short form: amd64,arm64 (auto-converted to linux/amd64,linux/arm64)"
        echo "  --force     Force rebuild without cache"
        echo ""
        echo "Description:"
        echo "  Build all AI-Infra service images for multiple architectures"
        echo "  Uses docker buildx for cross-platform builds"
        echo "  For single platform: images are loaded to local docker"
        echo "  For multi-platform: images are saved as OCI tarballs"
        echo ""
        echo "Examples:"
        echo "  $0 build-multiarch                              # Build for amd64 and arm64"
        echo "  $0 build-multiarch linux/amd64                  # Build for amd64 only"
        echo "  $0 build-multiarch linux/arm64 --force          # Force rebuild for arm64"
        echo "  $0 build-multiarch amd64,arm64                  # Short form"
        echo ""
        echo "Output:"
        echo "  Single platform: Images loaded to local docker daemon"
        echo "  Multi-platform: OCI tarballs in ./multiarch-images/"
        echo ""
        echo "Note:"
        echo "  - Requires docker buildx (included in Docker Desktop)"
        echo "  - Cross-platform builds use QEMU emulation (slower)"
        echo "  - For production, consider using CI/CD with native runners"
        return 0
    fi
    
    if [[ "$2" == "--force" ]] || [[ "$2" == "-f" ]]; then
        force="true"
    fi
    
    log_info "=========================================="
    log_info "🏗️  Multi-Architecture Build"
    log_info "=========================================="
    log_info "Target platforms: $platforms"
    log_info "Force rebuild: $force"
    echo
    
    # Setup buildx
    if ! setup_buildx_builder; then
        log_error "Failed to setup buildx builder"
        return 1
    fi
    echo
    
    # Set force flag
    if [[ "$force" == "true" ]]; then
        FORCE_BUILD=true
    fi
    
    # Initialize build
    init_build_cache
    CURRENT_BUILD_ID=$(generate_build_id)
    save_build_id "$CURRENT_BUILD_ID"
    
    log_info "Build Session: $CURRENT_BUILD_ID"
    echo
    
    # Render templates
    log_info "=== Phase 0: Rendering Dockerfile Templates ==="
    if ! render_all_templates "$force"; then
        log_error "Template rendering failed. Aborting build."
        return 1
    fi
    echo
    
    # Discover services
    discover_services
    
    local success_count=0
    local fail_count=0
    local skip_count=0
    
    # Build Foundation Services
    log_info "=== Phase 1: Building Foundation Services (Multi-Arch) ==="
    for service in "${FOUNDATION_SERVICES[@]}"; do
        log_info "━━━ Building: $service ━━━"
        if build_component_multiarch "$service" "$platforms"; then
            success_count=$((success_count + 1))
        else
            fail_count=$((fail_count + 1))
            log_error "Failed to build $service"
        fi
    done
    echo
    
    # For dependent services, we need AppHub running
    # But in multi-arch mode, we skip dependent services that require runtime AppHub
    log_info "=== Phase 2: Building Dependent Services (Multi-Arch) ==="
    log_warn "Note: Some dependent services may require AppHub to be running"
    log_warn "For cross-platform builds, ensure dependencies are pre-downloaded"
    
    # Determine AppHub URL for build args (may not be accessible in cross-compile)
    local apphub_port="${APPHUB_PORT:-28080}"
    local external_host="${EXTERNAL_HOST:-localhost}"
    local apphub_url="http://${external_host}:${apphub_port}"
    
    for service in "${DEPENDENT_SERVICES[@]}"; do
        log_info "━━━ Building: $service ━━━"
        if build_component_multiarch "$service" "$platforms" "--build-arg" "APPHUB_URL=$apphub_url"; then
            success_count=$((success_count + 1))
        else
            fail_count=$((fail_count + 1))
            log_error "Failed to build $service"
        fi
    done
    echo
    
    # Summary
    log_info "=========================================="
    log_info "🎉 Multi-Architecture Build Complete"
    log_info "=========================================="
    log_info "Platforms: $platforms"
    log_info "Success: $success_count"
    log_info "Failed: $fail_count"
    log_info "Build ID: $CURRENT_BUILD_ID"
    
    # Count platforms
    local platform_count=$(echo "$platforms" | tr ',' '\n' | wc -l | tr -d ' ')
    if [[ $platform_count -gt 1 ]]; then
        log_info ""
        log_info "📁 Output: ./multiarch-images/"
        log_info "   Each image is saved as an OCI tarball"
        log_info ""
        log_info "To import on target machine:"
        log_info "   docker load < ./multiarch-images/<image>.tar"
    fi
    
    if [[ $fail_count -gt 0 ]]; then
        return 1
    fi
    return 0
}

# Build single platform for export (useful for building other arch on dev machine)
# This builds images for a specific platform and loads them locally
# Usage: build_for_platform <platform> [--force]
build_for_platform() {
    local platform="${1:-linux/amd64}"
    local force="${2:-false}"
    
    # Normalize platform
    if [[ "$platform" != "linux/"* ]]; then
        platform="linux/$platform"
    fi
    
    if [[ "$1" == "--help" ]] || [[ "$1" == "-h" ]]; then
        echo "Usage: $0 build-platform <platform> [--force]"
        echo ""
        echo "Build all AI-Infra images for a specific platform"
        echo "Images are loaded to local docker daemon"
        echo ""
        echo "Arguments:"
        echo "  platform    Target platform (default: linux/amd64)"
        echo "              Examples: linux/amd64, linux/arm64, amd64, arm64"
        echo "  --force     Force rebuild without cache"
        echo ""
        echo "Examples:"
        echo "  $0 build-platform amd64          # Build AMD64 images on ARM Mac"
        echo "  $0 build-platform arm64 --force  # Force rebuild ARM64"
        return 0
    fi
    
    if [[ "$2" == "--force" ]] || [[ "$2" == "-f" ]]; then
        force="true"
    fi
    
    log_info "Building for single platform: $platform"
    build_multiarch "$platform" "$force"
}

# ==============================================================================
# Unified Image Tag Management (方案一：统一镜像标签)
# ==============================================================================
# 
# Strategy: 
# - All builds produce architecture-suffixed images: ai-infra-xxx:v0.3.8-amd64, ai-infra-xxx:v0.3.8-arm64
# - After build, create unified tags for native architecture: ai-infra-xxx:v0.3.8 -> ai-infra-xxx:v0.3.8-arm64
# - docker-compose uses unified tags (without suffix)
# - start-all checks if images match local CPU architecture
#
# Benefits:
# - Can store both architectures on the same machine
# - Easy to export specific architecture for deployment
# - Clear architecture identification via image tag

# Create unified tags for native architecture
# Maps ai-infra-xxx:tag-arch to ai-infra-xxx:tag for docker-compose compatibility
# Usage: create_unified_tags_for_native <arch> <tag>
create_unified_tags_for_native() {
    local arch="$1"
    local tag="${2:-${IMAGE_TAG:-latest}}"
    local created=0
    local skipped=0
    local failed=0
    
    # Find all ai-infra images with architecture suffix
    local arch_pattern=":${tag}-${arch}"
    local images=$(docker images --format '{{.Repository}}:{{.Tag}}' | grep "ai-infra" | grep "$arch_pattern" || true)
    
    if [[ -z "$images" ]]; then
        log_warn "No ai-infra images found with tag pattern: ${arch_pattern}"
        return 0
    fi
    
    log_info "Found $(echo "$images" | wc -l | tr -d ' ') images to tag"
    
    while IFS= read -r image; do
        [[ -z "$image" ]] && continue
        
        # Extract base image name (remove architecture suffix)
        # ai-infra-nginx:v0.3.8-arm64 -> ai-infra-nginx:v0.3.8
        local unified_tag="${image%-${arch}}"
        
        # Check if unified tag already exists and matches
        local existing_id=$(docker images -q "$unified_tag" 2>/dev/null || true)
        local source_id=$(docker images -q "$image" 2>/dev/null || true)
        
        if [[ -n "$existing_id" ]] && [[ "$existing_id" == "$source_id" ]]; then
            log_info "  ⏭️  Skip (same): $unified_tag"
            skipped=$((skipped + 1))
            continue
        fi
        
        if docker tag "$image" "$unified_tag" 2>/dev/null; then
            log_info "  ✓ Tagged: $image -> $unified_tag"
            created=$((created + 1))
        else
            log_warn "  ✗ Failed: $image -> $unified_tag"
            failed=$((failed + 1))
        fi
    done <<< "$images"
    
    log_info "Unified tags: $created created, $skipped skipped, $failed failed"
}

# Check if local images match the native CPU architecture
# Returns 0 if all images match, 1 if mismatch detected
check_images_architecture() {
    local native_platform=$(_detect_docker_platform)
    local native_arch="${native_platform##*/}"
    local tag="${IMAGE_TAG:-latest}"
    local compose_cmd=$(detect_compose_command)
    
    log_info "Checking image architectures for native platform: $native_arch"
    
    # Get list of ai-infra images from docker-compose
    local images=$($compose_cmd config --images 2>/dev/null | grep "ai-infra" | sort -u || true)
    
    if [[ -z "$images" ]]; then
        log_warn "No ai-infra images found in docker-compose config"
        return 0
    fi
    
    local total=0
    local matched=0
    local mismatched=0
    local missing=0
    local mismatch_list=()
    local missing_list=()
    
    while IFS= read -r image; do
        [[ -z "$image" ]] && continue
        total=$((total + 1))
        
        # Check if image exists
        if ! docker image inspect "$image" >/dev/null 2>&1; then
            missing=$((missing + 1))
            missing_list+=("$image")
            continue
        fi
        
        # Get image architecture
        local image_arch=$(docker image inspect "$image" --format '{{.Architecture}}' 2>/dev/null || echo "unknown")
        
        if [[ "$image_arch" == "$native_arch" ]]; then
            matched=$((matched + 1))
        else
            mismatched=$((mismatched + 1))
            mismatch_list+=("$image ($image_arch)")
        fi
    done <<< "$images"
    
    log_info "Architecture check: $matched/$total matched ($native_arch)"
    
    # Report missing images
    if [[ $missing -gt 0 ]]; then
        log_warn "Missing $missing images:"
        for img in "${missing_list[@]}"; do
            log_warn "  - $img"
        done
    fi
    
    # Report mismatched images
    if [[ $mismatched -gt 0 ]]; then
        log_error "Found $mismatched images with wrong architecture:"
        for img in "${mismatch_list[@]}"; do
            log_error "  - $img"
        done
        log_error ""
        log_error "Your machine is $native_arch, but some images are built for different architecture."
        log_error "Options to fix:"
        log_error "  1. Rebuild for native: ./build.sh build-all"
        log_error "  2. Rebuild for specific arch: ./build.sh build-all --platform=$native_arch"
        log_error "  3. Tag existing arch images: docker tag ai-infra-xxx:${tag}-${native_arch} ai-infra-xxx:${tag}"
        return 1
    fi
    
    if [[ $missing -gt 0 ]]; then
        log_warn "Some images are missing. Run: ./build.sh build-all"
        # Don't fail on missing - docker-compose will handle it
    fi
    
    log_info "✓ All images match native architecture ($native_arch)"
    return 0
}

# Build all services for multiple platforms sequentially
# This is an enhanced version of build_all that builds for each platform
# Usage: build_all_multiplatform <platforms> [force]
# Example: build_all_multiplatform "amd64,arm64" "true"
build_all_multiplatform() {
    local platforms="${1:-amd64,arm64}"
    local force="${2:-false}"
    
    # Normalize platform format
    IFS=',' read -ra PLATFORM_ARRAY <<< "$platforms"
    local normalized_platforms=()
    for p in "${PLATFORM_ARRAY[@]}"; do
        p=$(echo "$p" | tr -d '[:space:]')
        case "$p" in
            amd64|x86_64) normalized_platforms+=("linux/amd64") ;;
            arm64|aarch64) normalized_platforms+=("linux/arm64") ;;
            linux/amd64|linux/arm64) normalized_platforms+=("$p") ;;
            *) log_warn "Unknown platform: $p, skipping" ;;
        esac
    done
    
    if [[ ${#normalized_platforms[@]} -eq 0 ]]; then
        log_error "No valid platforms specified"
        return 1
    fi
    
    log_info "=========================================="
    log_info "🏗️  Multi-Platform Build (build-all)"
    log_info "=========================================="
    log_info "Target platforms: ${normalized_platforms[*]}"
    log_info "Force rebuild: $force"
    echo
    
    # Setup buildx builder
    if ! setup_buildx_builder; then
        log_error "Failed to setup buildx builder"
        return 1
    fi
    echo
    
    # Set force flag globally
    if [[ "$force" == "true" ]]; then
        FORCE_BUILD=true
        FORCE_REBUILD=true
    fi
    
    # Initialize build cache
    init_build_cache
    CURRENT_BUILD_ID=$(generate_build_id)
    save_build_id "$CURRENT_BUILD_ID"
    
    log_info "Build Session: $CURRENT_BUILD_ID"
    echo
    
    # ==========================================================================
    # Phase 1: 准备 - 渲染模板、预取镜像、处理依赖
    # ==========================================================================
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "📦 Phase 1: Preparation (Templates, Base Images, Dependencies)"
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo
    
    # 1.1 Render all templates
    log_info "  [1.1] Rendering Dockerfile Templates..."
    if ! render_all_templates "$force"; then
        log_error "Template rendering failed. Aborting build."
        return 1
    fi
    echo
    
    # 1.2 Prefetch base images for all platforms
    log_info "  [1.2] Prefetching Base Images (Multi-Platform)..."
    for platform in "${normalized_platforms[@]}"; do
        local arch_name="${platform##*/}"
        log_info "    Prefetching base images for $arch_name..."
        prefetch_base_images_for_platform "$platform"
    done
    echo
    
    # 1.2.5 Pull common/third-party images used in docker-compose.yml
    # These images (mysql, redis, victoriametrics, etc.) are NOT in Dockerfiles
    # but are directly referenced in docker-compose.yml
    log_info "  [1.2.5] Pulling Common/Third-party Images (docker-compose dependencies)..."
    if ! pull_common_images "$DEFAULT_MAX_RETRIES"; then
        log_warn "  ⚠️  Some common images failed to pull, services may fail to start"
        log_warn "  💡 You can retry later with: ./build.sh pull-common"
    fi
    echo
    
    # 1.3 Discover and process dependency services
    discover_services
    
    log_info "  [1.3] Processing Dependency Services..."
    for service in "${DEPENDENCY_SERVICES[@]}"; do
        log_info "    Processing dependency: $service"
        for platform in "${normalized_platforms[@]}"; do
            local arch_name="${platform##*/}"
            pull_dependency_for_platform "$service" "$platform"
        done
    done
    echo
    
    # Detect native platform for later use
    local native_platform=$(_detect_docker_platform)
    local native_arch="${native_platform##*/}"
    local tag="${IMAGE_TAG:-latest}"
    
    # Check if native platform is in target platforms
    local has_native_platform=false
    for platform in "${normalized_platforms[@]}"; do
        local arch="${platform##*/}"
        if [[ "$arch" == "$native_arch" ]]; then
            has_native_platform=true
            break
        fi
    done
    
    # ==========================================================================
    # Phase 2: 构建 - 构建所有服务镜像（新架构：先构建所有基础服务，再统一启动AppHub）
    # ==========================================================================
    # 新流程说明：
    # 1. 先构建所有架构的 Foundation Services（包括 AppHub）
    # 2. AppHub 镜像包含所有架构的包（SaltStack/SLURM deb/rpm）
    # 3. 只启动一个原生架构的 AppHub 实例
    # 4. 使用该 AppHub 为所有架构的依赖服务提供包
    # 
    # 优点：
    # - 无需在构建过程中切换 AppHub
    # - 避免 QEMU 模拟运行 AppHub 的性能问题
    # - 简化构建流程
    # ==========================================================================
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "🔨 Phase 2: Build All Services (Multi-Platform)"
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "Platforms: ${normalized_platforms[*]}"
    if [[ ${#normalized_platforms[@]} -gt 1 ]]; then
        log_info "⚙️  Multi-arch build: AppHub will be built for all platforms, then started once"
    fi
    
    # --------------------------------------------------------------------------
    # Pre-build Summary: Show which components will be built or skipped
    # --------------------------------------------------------------------------
    if [[ "$FORCE_REBUILD" != "true" ]]; then
        log_info ""
        log_info "📋 Pre-build Analysis (checking existing images)..."
        local to_build=()
        local to_skip=()
        
        for platform in "${normalized_platforms[@]}"; do
            local arch_name="${platform##*/}"
            for service in "${FOUNDATION_SERVICES[@]}" "${DEPENDENT_SERVICES[@]}"; do
                local rebuild_reason=$(need_rebuild_for_platform "$service" "$platform" "$tag")
                if [[ "$rebuild_reason" == "NO_CHANGE" ]]; then
                    to_skip+=("$service:$arch_name")
                else
                    to_build+=("$service:$arch_name ($rebuild_reason)")
                fi
            done
        done
        
        if [[ ${#to_skip[@]} -gt 0 ]]; then
            log_cache "  ⏭️  Will skip ${#to_skip[@]} component(s): ${to_skip[*]}"
        fi
        if [[ ${#to_build[@]} -gt 0 ]]; then
            log_info "  🔧 Will build ${#to_build[@]} component(s)"
            for item in "${to_build[@]}"; do
                log_info "      → $item"
            done
        fi
        
        if [[ ${#to_build[@]} -eq 0 ]]; then
            log_info ""
            log_info "✅ All components are up-to-date! No rebuild needed."
            log_info "   Use --force to rebuild anyway: ./build.sh --force all --platform=${platforms}"
            return 0
        fi
        echo
    fi
    
    local compose_cmd=$(detect_compose_command)
    if [ -z "$compose_cmd" ]; then
        log_error "docker-compose not found! Cannot manage AppHub."
        return 1
    fi
    
    # Stop any existing AppHub before multi-platform build to ensure clean state
    if docker ps -q -f name=ai-infra-apphub 2>/dev/null | grep -q .; then
        log_info "🛑 Stopping existing AppHub container before build..."
        $compose_cmd stop apphub 2>/dev/null || docker stop ai-infra-apphub 2>/dev/null || true
        docker rm -f ai-infra-apphub 2>/dev/null || true
        log_info "   ✓ Existing AppHub stopped"
    fi
    
    local apphub_port="${APPHUB_PORT:-28080}"
    local external_host="${EXTERNAL_HOST:-$(detect_external_host)}"
    local apphub_url="http://${external_host}:${apphub_port}"
    
    # --------------------------------------------------------------------------
    # Phase 2.1: Build ALL Foundation Services for ALL platforms first
    # --------------------------------------------------------------------------
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "📦 Phase 2.1: Build Foundation Services (All Platforms)"
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # Validate FOUNDATION_SERVICES array
    log_info "  Foundation services to build: ${FOUNDATION_SERVICES[*]}"
    if [[ ${#FOUNDATION_SERVICES[@]} -eq 0 ]]; then
        log_error "  ❌ FOUNDATION_SERVICES is empty! Check discover_services function."
        return 1
    fi
    
    # Ensure apphub is in FOUNDATION_SERVICES (required for dependent builds)
    local apphub_in_foundation=false
    for svc in "${FOUNDATION_SERVICES[@]}"; do
        if [[ "$svc" == "apphub" ]]; then
            apphub_in_foundation=true
            break
        fi
    done
    if [[ "$apphub_in_foundation" != "true" ]]; then
        log_error "  ❌ 'apphub' is not in FOUNDATION_SERVICES!"
        log_error "     This is likely a configuration error."
        log_error "     Check: src/apphub/build.conf should contain BUILD_PHASE=foundation"
        return 1
    fi
    
    local foundation_build_failed=false
    for platform in "${normalized_platforms[@]}"; do
        local arch_name="${platform##*/}"
        
        log_info "┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓"
        log_info "┃ 🏗️  Building Foundation Services for [$arch_name]"
        log_info "┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛"
        
        if [[ "$ENABLE_PARALLEL" == "true" ]] && [[ ${#FOUNDATION_SERVICES[@]} -gt 1 ]]; then
            log_parallel "    🚀 Parallel build enabled (max $PARALLEL_JOBS jobs)"
            if ! build_parallel_for_platform "$platform" "${FOUNDATION_SERVICES[@]}"; then
                log_error "    ❌ Some foundation services failed to build for $arch_name"
                foundation_build_failed=true
            fi
        else
            for service in "${FOUNDATION_SERVICES[@]}"; do
                log_info "    → Building $service for $arch_name..."
                if ! build_component_for_platform "$service" "$platform"; then
                    log_error "    ❌ Failed to build $service for $arch_name"
                    foundation_build_failed=true
                fi
            done
        fi
        echo
    done
    
    # Check if any foundation service build failed
    if [[ "$foundation_build_failed" == "true" ]]; then
        log_error "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        log_error "❌ Some foundation services failed to build."
        log_error "   Please check the build logs above for details."
        log_error "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        # Don't return immediately - continue to check AppHub specifically
    fi
    
    # Check if ANY AppHub image exists (required for Phase 2.3)
    # AppHub contains packages for ALL architectures, so we can use any available AppHub
    # Priority: 1) Target platform AppHub, 2) Native platform AppHub, 3) Any AppHub
    local apphub_built=false
    local apphub_available_arch=""
    
    # First, check target platforms
    for platform in "${normalized_platforms[@]}"; do
        local arch="${platform##*/}"
        local apphub_image="ai-infra-apphub:${tag}-${arch}"
        if docker image inspect "$apphub_image" >/dev/null 2>&1; then
            apphub_built=true
            apphub_available_arch="$arch"
            log_info "  ✓ AppHub image found: $apphub_image"
            break
        fi
    done
    
    # If not found, check native platform (e.g., arm64 on Mac)
    if [[ "$apphub_built" != "true" ]]; then
        local native_apphub="ai-infra-apphub:${tag}-${native_arch}"
        if docker image inspect "$native_apphub" >/dev/null 2>&1; then
            apphub_built=true
            apphub_available_arch="$native_arch"
            log_info "  ✓ AppHub image found (native arch): $native_apphub"
            log_info "    📦 AppHub contains packages for ALL architectures, can be used for cross-platform builds"
        fi
    fi
    
    # Last resort: check for any AppHub with matching tag
    if [[ "$apphub_built" != "true" ]]; then
        local any_apphub=$(docker images --format "{{.Repository}}:{{.Tag}}" 2>/dev/null | grep "ai-infra-apphub:${tag}" | head -1)
        if [[ -n "$any_apphub" ]]; then
            apphub_built=true
            apphub_available_arch="${any_apphub##*-}"
            log_info "  ✓ AppHub image found (fallback): $any_apphub"
        fi
    fi
    
    if [[ "$apphub_built" != "true" ]]; then
        log_error "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        log_error "❌ CRITICAL: No AppHub image found!"
        log_error "   AppHub is required to serve packages for dependent services."
        log_error ""
        log_error "   Debug steps:"
        log_error "   1. Check build logs above for AppHub errors"
        log_error "   2. Build AppHub for native arch first: ./build.sh apphub --platform=${native_arch}"
        log_error "   3. Check AppHub Dockerfile: src/apphub/Dockerfile"
        log_error "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        return 1
    fi
    
    # --------------------------------------------------------------------------
    # Phase 2.2: Start ONE AppHub instance (native architecture preferred)
    # --------------------------------------------------------------------------
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "🚀 Phase 2.2: Start AppHub (Single Instance for All Architectures)"
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # Clean up any existing AppHub container
    if docker ps -a --filter "name=^ai-infra-apphub$" --format "{{.ID}}" | grep -q .; then
        log_info "  🧹 Removing existing AppHub container..."
        docker stop ai-infra-apphub 2>/dev/null || true
        docker rm -f ai-infra-apphub 2>/dev/null || true
    fi
    
    # Decide which AppHub image to use (with fallback logic)
    local apphub_image_to_use=""
    local apphub_platform=""
    local apphub_arch_selected=""
    
    # Try to find an available AppHub image in order of preference:
    # 1. Native architecture (best performance)
    # 2. First platform in the build list
    # 3. Any available AppHub image
    
    if [[ "$has_native_platform" == "true" ]]; then
        local native_image="ai-infra-apphub:${tag}-${native_arch}"
        if docker image inspect "$native_image" >/dev/null 2>&1; then
            apphub_image_to_use="$native_image"
            apphub_platform="linux/${native_arch}"
            apphub_arch_selected="$native_arch"
            log_info "  📌 Using native architecture AppHub: $apphub_image_to_use"
        fi
    fi
    
    # Fallback: try each platform in the build list
    if [[ -z "$apphub_image_to_use" ]]; then
        for platform in "${normalized_platforms[@]}"; do
            local arch="${platform##*/}"
            local test_image="ai-infra-apphub:${tag}-${arch}"
            if docker image inspect "$test_image" >/dev/null 2>&1; then
                apphub_image_to_use="$test_image"
                apphub_platform="linux/${arch}"
                apphub_arch_selected="$arch"
                log_warn "  ⚠️  Native AppHub not available, using: $apphub_image_to_use"
                break
            fi
        done
    fi
    
    # Last resort: find any AppHub image with matching tag
    if [[ -z "$apphub_image_to_use" ]]; then
        local any_apphub=$(docker images --format "{{.Repository}}:{{.Tag}}" | grep "ai-infra-apphub:${tag}" | head -1)
        if [[ -n "$any_apphub" ]]; then
            apphub_image_to_use="$any_apphub"
            # Extract arch from tag (e.g., v0.3.8-amd64 -> amd64)
            apphub_arch_selected="${any_apphub##*-}"
            apphub_platform="linux/${apphub_arch_selected}"
            log_warn "  ⚠️  Using available AppHub image: $apphub_image_to_use"
        fi
    fi
    
    # Verify we found an AppHub image
    if [[ -z "$apphub_image_to_use" ]]; then
        log_error "  ❌ No AppHub image found for tag: ${tag}"
        log_error "     Available images:"
        docker images --format "  {{.Repository}}:{{.Tag}}" | grep apphub || echo "     (none)"
        log_error ""
        log_error "     Please build AppHub first:"
        log_error "       ./build.sh build-platform linux/${native_arch} apphub"
        return 1
    fi
    
    # Start AppHub
    log_info "  🚀 Starting AppHub..."
    docker network create ai-infra-network 2>/dev/null || true
    
    # Determine if we should use docker-compose or docker run
    # Use docker-compose only if the selected AppHub is native architecture
    if [[ "$apphub_arch_selected" == "$native_arch" ]]; then
        # Use docker-compose for native platform (better integration)
        $compose_cmd up -d apphub
    else
        # Use docker run for cross-platform (via QEMU/Rosetta)
        docker run -d \
            --name ai-infra-apphub \
            --platform "$apphub_platform" \
            --network ai-infra-network \
            -p "${apphub_port}:80" \
            -v "${SCRIPT_DIR}/third_party:/app/third_party:ro" \
            -v "${SCRIPT_DIR}/src:/app/src:ro" \
            "$apphub_image_to_use"
        
        if [[ $? -ne 0 ]]; then
            log_error "  ❌ Failed to start AppHub"
            return 1
        fi
    fi
    
    # Wait for AppHub to be ready
    log_info "  ⏳ Waiting for AppHub to be ready..."
    if ! wait_for_apphub_ready 300; then
        log_error "  ❌ AppHub failed to become ready. Aborting build."
        return 1
    fi
    
    # Record which architecture's AppHub is currently running
    local current_apphub_arch="$apphub_arch_selected"
    log_info "  ✓ AppHub is ready at $apphub_url (arch: $current_apphub_arch)"
    echo
    
    # --------------------------------------------------------------------------
    # Phase 2.3: Build Dependent Services for ALL platforms
    # --------------------------------------------------------------------------
    # 架构匹配说明：
    # - AppHub 中的 SLURM 包是架构相关的（arm64 包只能用于 arm64 构建）
    # - 当构建目标架构与当前 AppHub 架构不匹配时：
    #   方案 A: 切换到对应架构的 AppHub（通过 QEMU/Rosetta）
    #   方案 B: 启用系统回退 ALLOW_SYSTEM_SLURM=true
    # 
    # Mac ARM64 特殊性：
    # - Mac 有 Rosetta 转译层，可以高效运行 amd64 容器
    # - 在 Mac 上切换 AppHub 架构是可行的
    # 
    # Linux ARM64：
    # - 需要 QEMU 模拟运行 amd64 容器（较慢）
    # - 建议使用系统回退或在原生机器上构建
    # --------------------------------------------------------------------------
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "📦 Phase 2.3: Build Dependent Services (All Platforms)"
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "  Using AppHub URL: $apphub_url"
    log_info "  Current AppHub arch: $current_apphub_arch"
    
    # Detect if running on macOS (has Rosetta for efficient cross-arch)
    local is_macos=false
    if [[ "$(uname -s)" == "Darwin" ]]; then
        is_macos=true
        log_info "  🍎 Running on macOS (Rosetta available for cross-arch)"
    fi
    echo
    
    for platform in "${normalized_platforms[@]}"; do
        local arch_name="${platform##*/}"
        local need_apphub_switch=false
        local use_system_fallback=false
        
        log_info "┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┓"
        log_info "┃ 🏗️  Building Dependent Services for [$arch_name]"
        log_info "┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━┛"
        
        # Check if target arch matches current AppHub arch
        if [[ "$arch_name" != "$current_apphub_arch" ]]; then
            log_warn "    ⚠️  Target arch ($arch_name) differs from AppHub arch ($current_apphub_arch)"
            
            # Check if we have the target arch's AppHub image
            local target_apphub_image="ai-infra-apphub:${tag}-${arch_name}"
            if docker image inspect "$target_apphub_image" >/dev/null 2>&1; then
                if [[ "$is_macos" == "true" ]]; then
                    # On macOS, we can switch AppHub efficiently via Rosetta
                    log_info "    🔄 Switching AppHub to $arch_name (via Rosetta/QEMU)..."
                    need_apphub_switch=true
                else
                    # On Linux, QEMU is slower, prefer system fallback
                    log_warn "    📦 Using system SLURM fallback (QEMU would be slow)"
                    use_system_fallback=true
                fi
            else
                # No target arch AppHub image, must use system fallback
                log_warn "    📦 AppHub image for $arch_name not found, using system fallback"
                use_system_fallback=true
            fi
        fi
        
        # Switch AppHub if needed
        if [[ "$need_apphub_switch" == "true" ]]; then
            log_info "    🧹 Stopping current AppHub..."
            docker stop ai-infra-apphub 2>/dev/null || true
            docker rm -f ai-infra-apphub 2>/dev/null || true
            
            local target_apphub_image="ai-infra-apphub:${tag}-${arch_name}"
            log_info "    🚀 Starting AppHub for $arch_name..."
            
            docker run -d \
                --name ai-infra-apphub \
                --platform "linux/$arch_name" \
                --network ai-infra-network \
                -p "${apphub_port}:80" \
                -v "${SCRIPT_DIR}/third_party:/app/third_party:ro" \
                -v "${SCRIPT_DIR}/src:/app/src:ro" \
                "$target_apphub_image"
            
            if [[ $? -ne 0 ]]; then
                log_warn "    ⚠️  Failed to start $arch_name AppHub, using system fallback"
                use_system_fallback=true
            else
                log_info "    ⏳ Waiting for AppHub to be ready..."
                if wait_for_apphub_ready 180; then
                    current_apphub_arch="$arch_name"
                    log_info "    ✓ AppHub switched to $arch_name"
                else
                    log_warn "    ⚠️  AppHub not ready, using system fallback"
                    use_system_fallback=true
                fi
            fi
        fi
        
        # Determine SLURM build args
        local slurm_build_args="--build-arg APPHUB_URL=$apphub_url"
        if [[ "$use_system_fallback" == "true" ]]; then
            slurm_build_args="$slurm_build_args --build-arg ALLOW_SYSTEM_SLURM=true"
            log_info "    📦 System SLURM fallback enabled for $arch_name"
        elif ! check_apphub_slurm_available "$arch_name"; then
            log_warn "    ⚠️  SLURM packages for $arch_name not available in AppHub, enabling system fallback"
            slurm_build_args="$slurm_build_args --build-arg ALLOW_SYSTEM_SLURM=true"
        else
            log_info "    ✓ AppHub SLURM packages available for $arch_name"
        fi
        
        local dependent_build_failed=false
        if [[ "$ENABLE_PARALLEL" == "true" ]] && [[ ${#DEPENDENT_SERVICES[@]} -gt 1 ]]; then
            log_parallel "    🚀 Parallel build enabled (max $PARALLEL_JOBS jobs)"
            # For parallel builds, we need to pass ALLOW_SYSTEM_SLURM for ALL services
            # because slurm-master will be built in parallel with others
            # The extra arg will be ignored by non-slurm services
            local parallel_extra_args="--build-arg APPHUB_URL=$apphub_url"
            if ! check_apphub_slurm_available "$arch_name"; then
                parallel_extra_args="$parallel_extra_args --build-arg ALLOW_SYSTEM_SLURM=true"
                log_info "    📦 ALLOW_SYSTEM_SLURM=true will be used for slurm-master"
            fi
            if ! build_parallel_for_platform "$platform" "${DEPENDENT_SERVICES[@]}" -- $parallel_extra_args; then
                log_error "    ❌ Some dependent services failed to build for $arch_name"
                dependent_build_failed=true
            fi
        else
            for service in "${DEPENDENT_SERVICES[@]}"; do
                log_info "    → Building $service for $arch_name..."
                # Use slurm_build_args for slurm-master, regular args for others
                if [[ "$service" == "slurm-master" ]]; then
                    if ! build_component_for_platform "$service" "$platform" $slurm_build_args; then
                        log_error "    ❌ Failed to build $service for $arch_name"
                        dependent_build_failed=true
                    fi
                else
                    if ! build_component_for_platform "$service" "$platform" "--build-arg" "APPHUB_URL=$apphub_url"; then
                        log_error "    ❌ Failed to build $service for $arch_name"
                        dependent_build_failed=true
                    fi
                fi
            done
        fi
        
        if [[ "$dependent_build_failed" == "true" ]]; then
            log_warn "    ⚠️  Some dependent services failed to build for $arch_name"
        fi
        echo
    done
    
    # ==========================================================================
    # Phase 3: 完成 - 创建统一标签
    # ==========================================================================
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "🏷️  Phase 3: Finalize (Create Unified Tags)"
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo
    
    if [[ "$has_native_platform" == "true" ]]; then
        log_info "Creating unified tags for native architecture: $native_arch"
        create_unified_tags_for_native "$native_arch" "$tag"
    else
        log_warn "Native architecture ($native_arch) was not built."
        log_warn "To start services on this machine, you need to build for $native_arch first."
        log_info "Or manually tag images: docker tag ai-infra-xxx:${tag}-<arch> ai-infra-xxx:${tag}"
    fi
    echo

    # Build summary
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "🎉 Multi-Platform Build Session $CURRENT_BUILD_ID Completed"
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "Platforms built: ${normalized_platforms[*]}"
    log_info ""
    log_info "📋 Next steps:"
    log_info "   Start services: ./build.sh start-all"
    log_info "   Export for offline: ./build.sh export-offline ./offline ${IMAGE_TAG:-latest} true ${platforms}"
    log_info ""
    log_info "View build history: ./build.sh build-history"
}

# Helper: Prefetch base images for a specific platform
# Dynamically discovers and prefetches all base images from Dockerfiles
# Features:
#   - Renders all Dockerfile templates first
#   - Extracts base images dynamically from src/*/Dockerfile
#   - No hardcoded image list - all from actual Dockerfiles
#   - Retry mechanism with exponential backoff
# Args: $1 = platform (e.g., linux/amd64, linux/arm64)
#       $2 = max_retries (default 5)
#       $3 = retry_delay (default 10)
prefetch_base_images_for_platform() {
    local platform="$1"
    local max_retries="${2:-5}"
    local retry_delay="${3:-10}"
    local arch_name="${platform##*/}"
    
    log_info "  [$arch_name] Phase 1: Rendering Dockerfile templates..."
    
    # Render all templates to ensure Dockerfiles are up-to-date
    if ! render_all_templates false 2>/dev/null; then
        log_warn "  [$arch_name] Template rendering had issues, continuing with existing Dockerfiles..."
    fi
    
    log_info "  [$arch_name] Phase 2: Discovering base images from Dockerfiles..."
    
    # Dynamically extract all base images from Dockerfiles
    local all_base_images=()
    local dockerfiles=()
    
    # Find all Dockerfiles (including rendered ones from templates)
    while IFS= read -r dockerfile; do
        dockerfiles+=("$dockerfile")
    done < <(find "$SRC_DIR" -name "Dockerfile" -type f \
        -not -path "*/node_modules/*" \
        -not -path "*/.git/*" \
        -not -path "*/vendor/*" \
        -not -path "*/__pycache__/*" \
        2>/dev/null)
    
    if [[ ${#dockerfiles[@]} -eq 0 ]]; then
        log_warn "  [$arch_name] No Dockerfiles found in $SRC_DIR"
        return 0
    fi
    
    log_info "  [$arch_name] Found ${#dockerfiles[@]} Dockerfiles to scan"
    
    # Extract base images from all Dockerfiles
    for dockerfile in "${dockerfiles[@]}"; do
        local images
        if images=$(extract_base_images "$dockerfile" 2>/dev/null); then
            while IFS= read -r image; do
                [[ -z "$image" ]] && continue
                # Skip internal build stages (single word without / or :)
                [[ "$image" =~ ^[a-z_-]+$ ]] && continue
                all_base_images+=("$image")
            done <<< "$images"
        fi
    done
    
    # Remove duplicates and sort
    local unique_base_images=($(printf '%s\n' "${all_base_images[@]}" | sort -u))
    
    if [[ ${#unique_base_images[@]} -eq 0 ]]; then
        log_warn "  [$arch_name] No base images found in Dockerfiles"
        return 0
    fi
    
    log_info "  [$arch_name] Phase 3: Prefetching ${#unique_base_images[@]} unique base images with retry..."
    
    # Ensure multiarch-builder exists before prefetching
    local builder_name="multiarch-builder"
    if ! docker buildx inspect "$builder_name" >/dev/null 2>&1; then
        log_info "  [$arch_name] Pre-creating multiarch-builder for prefetch..."
        
        # Prepare BuildKit config with mirrors and proxy support
        local buildkit_config="/tmp/buildkitd-multiarch.toml"
        _generate_buildkit_config_with_proxy "$buildkit_config"
        
        # Create builder with docker-container driver and host network
        local create_args=(
            "--name" "$builder_name"
            "--driver" "docker-container"
            "--driver-opt" "network=host"
            "--buildkitd-flags" "--allow-insecure-entitlement network.host"
            "--bootstrap"
        )
        
        # Add config file if it was generated successfully
        if [[ -f "$buildkit_config" ]]; then
            create_args+=("--config" "$buildkit_config")
            log_info "  [$arch_name] Using custom buildkit config with mirrors"
        fi
        
        if ! docker buildx create "${create_args[@]}" 2>&1 | grep -v "^$"; then
            log_warn "  [$arch_name] Failed to pre-create multiarch-builder, prefetch may fail"
        fi
    fi
    
    # Pre-pull images directly using docker pull
    # This is simpler and more reliable than using docker buildx build for prefetch
    # Docker daemon handles mirror acceleration automatically
    
    local success_count=0
    local fail_count=0
    local skip_count=0
    
    for image in "${unique_base_images[@]}"; do
        # =====================================================================
        # CRITICAL FIX: Check if image exists WITH CORRECT ARCHITECTURE
        # 
        # docker image inspect doesn't check architecture!
        # We need to verify the image's architecture matches our target platform.
        # If architecture doesn't match, we must re-pull with --platform flag.
        # =====================================================================
        local image_exists=false
        local arch_matches=false
        
        if docker image inspect "$image" >/dev/null 2>&1; then
            image_exists=true
            # Check if the image architecture matches our target
            local image_arch=$(docker image inspect "$image" --format '{{.Architecture}}' 2>/dev/null || echo "unknown")
            if [[ "$image_arch" == "$arch_name" ]]; then
                arch_matches=true
            fi
        fi
        
        if [[ "$image_exists" == "true" ]] && [[ "$arch_matches" == "true" ]]; then
            log_info "    ✓ $image (cached, arch: $arch_name)"
            skip_count=$((skip_count + 1))
            continue
        elif [[ "$image_exists" == "true" ]] && [[ "$arch_matches" == "false" ]]; then
            local current_arch=$(docker image inspect "$image" --format '{{.Architecture}}' 2>/dev/null || echo "unknown")
            log_info "    🔄 $image (exists as $current_arch, need $arch_name)"
        fi
        
        log_info "    ⬇ $image"
        
        # Try to pull image with retries
        local retry_count=0
        local pulled=false
        local pull_output=""
        local pull_timeout="${BASE_IMAGE_PULL_TIMEOUT:-180}"  # Default 3 minutes per base image
        
        while [[ $retry_count -lt $max_retries ]]; do
            retry_count=$((retry_count + 1))
            
            if [[ $retry_count -gt 1 ]]; then
                local wait_time=$((retry_delay * retry_count))
                log_warn "    🔄 Retry $retry_count/$max_retries: $image (waiting ${wait_time}s...)"
                sleep "$wait_time"
            fi
            
            # Pull image with platform specification and timeout
            # This will use Docker daemon's configured mirrors automatically
            if pull_output=$(timeout "$pull_timeout" docker pull --platform "$platform" "$image" 2>&1); then
                # Check if pull succeeded
                if echo "$pull_output" | grep -qiE "(Digest:|Status:|Downloaded|already exists)"; then
                    log_info "    ✓ $image"
                    success_count=$((success_count + 1))
                    pulled=true
                    break
                fi
            fi
            
            local exit_code=$?
            
            # Check for timeout first
            if [[ $exit_code -eq 124 ]]; then
                log_warn "    ⏱️ Timeout (>${pull_timeout}s): $image - will retry"
                continue
            fi
            
            # Pull failed - check error type
            if echo "$pull_output" | grep -qiE "DeadlineExceeded|timeout|temporary failure|connection refused|i/o timeout"; then
                # Network error - worth retrying
                log_warn "    ⚠️  Network error, will retry: $image"
                continue
            elif echo "$pull_output" | grep -qiE "not found|unknown manifest|no matching manifest"; then
                # Image doesn't exist in registry - don't retry
                log_warn "    ✗ $image (not found in registry)"
                fail_count=$((fail_count + 1))
                pulled=false
                break
            else
                # Other error - log and retry
                local error_msg=$(echo "$pull_output" | grep -iE "error|failed" | head -1 | cut -c1-100)
                log_warn "    ⚠️  Pull error (will retry): ${error_msg:-unknown error}"
                continue
            fi
        done
        
        if [[ "$pulled" == false ]]; then
            log_warn "    ✗ $image (failed after $max_retries attempts)"
            fail_count=$((fail_count + 1))
        fi
    done
    
    log_info "  [$arch_name] Phase 3 complete: $success_count pulled, $skip_count cached, $fail_count failed"
    
    return 0
}

# Helper: Pull dependency image for a specific platform
pull_dependency_for_platform() {
    local component="$1"
    local platform="$2"
    local component_dir="$SRC_DIR/$component"
    local tag="${IMAGE_TAG:-latest}"
    local arch_name="${platform##*/}"
    local pull_timeout="${DEPENDENCY_PULL_TIMEOUT:-120}"  # Default 2 minutes timeout per image
    
    # Check for dependency configuration
    local dep_conf="$component_dir/dependency.conf"
    if [ ! -f "$dep_conf" ]; then
        return 0
    fi
    
    # Skip if there's a Dockerfile (custom build)
    if [ -f "$component_dir/Dockerfile" ]; then
        return 0
    fi
    
    local upstream_image=$(grep -v '^#' "$dep_conf" | grep -v '^[[:space:]]*$' | head -n 1 | tr -d '[:space:]')
    if [ -z "$upstream_image" ]; then
        return 0
    fi
    
    # Use architecture-suffixed tag for ALL pulls (方案一：统一镜像管理)
    local arch_suffix="-${arch_name}"
    
    local target_image="ai-infra-$component:${tag}${arch_suffix}"
    
    # Skip if target image already exists
    if docker image inspect "$target_image" >/dev/null 2>&1; then
        log_info "  [$arch_name] ✓ $component (cached: $target_image)"
        return 0
    fi
    
    log_info "  [$arch_name] Pulling: $upstream_image -> $target_image (timeout: ${pull_timeout}s)"
    
    # Use timeout to prevent hanging on slow/unavailable images
    if timeout "$pull_timeout" docker pull --platform "$platform" "$upstream_image" >/dev/null 2>&1; then
        docker tag "$upstream_image" "$target_image" >/dev/null 2>&1
        log_info "  [$arch_name] ✓ Ready: $target_image"
    else
        local exit_code=$?
        if [[ $exit_code -eq 124 ]]; then
            log_warn "  [$arch_name] ⏱️ Timeout: $upstream_image (>${pull_timeout}s) - skipping"
        else
            log_warn "  [$arch_name] ✗ Failed to pull: $upstream_image"
        fi
        # Don't fail the build for dependency pulls - these are optional external images
        return 0
    fi
}

# Helper: Build a component for a specific platform using buildx
# Build a single component for a specific platform
# Enhanced version with full feature parity with build_component()
# Features: build cache check, dependency.conf support, build labels, history logging, private registry
# Usage: build_component_for_platform <component> <platform> [extra_args...]
# Example: build_component_for_platform apphub linux/amd64
#          build_component_for_platform backend linux/arm64 --build-arg APPHUB_URL=http://...
build_component_for_platform() {
    local component="$1"
    local platform="$2"
    local extra_args=("${@:3}")
    local component_dir="$SRC_DIR/$component"
    local tag="${IMAGE_TAG:-latest}"
    
    # Normalize platform format: ensure it has "linux/" prefix
    # Accept: amd64, arm64, linux/amd64, linux/arm64
    if [[ "$platform" != *"/"* ]]; then
        platform="linux/$platform"
    fi
    
    local arch_name="${platform##*/}"
    local build_id="${CURRENT_BUILD_ID:-$(generate_build_id)}"
    
    # For cross-platform builds, we need docker-container driver (docker driver doesn't support cross-platform pull)
    # The docker-container driver runs buildkit in a container with QEMU support
    # However, docker-container driver may have network isolation issues with proxies
    # Use intelligent builder selection to handle this
    local native_platform=$(_detect_docker_platform)
    local native_arch="${native_platform##*/}"
    
    # Intelligent builder selection: test network connectivity before choosing
    # This handles the common case where docker-container driver can't access Docker Hub
    # due to proxy misconfiguration or network isolation
    local builder_name
    
    # First, ensure multiarch-builder exists
    if ! docker buildx inspect "multiarch-builder" >/dev/null 2>&1; then
        log_info "  [$arch_name] Creating multiarch-builder with host network support..."
        
        # Prepare BuildKit config with mirrors and proxy support
        # This is needed because docker-container driver may not inherit daemon.json settings
        local buildkit_config="/tmp/buildkitd-multiarch.toml"
        _generate_buildkit_config_with_proxy "$buildkit_config"
        
        # Create builder with docker-container driver and host network
        # Using network=host allows buildkit to access the internet for apt/yum operations
        # For accessing docker containers (like apphub), we'll use --add-host in the build command
        # --buildkitd-flags enables network.host entitlement for --network=host in build commands
        # This is critical for arm64 (cross-platform) builds to avoid timeout issues
        local create_args=(
            "--name" "multiarch-builder"
            "--driver" "docker-container"
            "--driver-opt" "network=host"
            "--buildkitd-flags" "--allow-insecure-entitlement network.host"
            "--bootstrap"
        )
        
        # Add config file if it was generated successfully
        if [[ -f "$buildkit_config" ]]; then
            create_args+=("--config" "$buildkit_config")
            log_info "  [$arch_name] Using custom buildkit config with mirrors"
        fi
        
        if ! docker buildx create "${create_args[@]}" 2>&1; then
            log_warn "  [$arch_name] Failed to create multiarch-builder with network=host"
            log_warn "  [$arch_name] Attempting to create with default docker-container driver..."
            docker buildx create --name "multiarch-builder" --driver docker-container --bootstrap 2>&1 || true
        fi
    fi
    
    # Select the best builder based on network connectivity test
    # This is critical for environments where docker-container driver has proxy issues
    builder_name=$(_select_best_builder "$platform")
    local select_result=$?
    
    if [[ $select_result -ne 0 ]]; then
        log_warn "  [$arch_name] ⚠️  Network connectivity issues detected, build may fail"
        log_warn "  [$arch_name]    Please check your network/proxy configuration"
    fi
    
    if [ ! -d "$component_dir" ]; then
        log_error "[$arch_name] Component directory not found: $component_dir"
        return 1
    fi
    
    # ===== Build Cache Check =====
    # Skip cache check if FORCE_BUILD is enabled
    # Use platform-specific check to ensure each architecture is built independently
    if [[ "$FORCE_BUILD" != "true" ]]; then
        local rebuild_reason=$(need_rebuild_for_platform "$component" "$platform" "$tag")
        if [[ "$rebuild_reason" == "NO_CHANGE" ]]; then
            log_cache "  [$arch_name] ⏭️  Skipping $component (no changes detected)"
            log_build_history "$build_id" "$component" "$tag" "SKIPPED" "NO_CHANGE ($arch_name)"
            return 0
        fi
        log_cache "  [$arch_name] 🔄 Rebuilding $component: $rebuild_reason"
    fi
    
    # Check for template and render if needed
    local template_file="$component_dir/Dockerfile.tpl"
    if [ -f "$template_file" ]; then
        log_info "  [$arch_name] Rendering template for $component..."
        if ! render_template "$template_file" >/dev/null 2>&1; then
            log_error "  [$arch_name] Failed to render template for $component"
            log_build_history "$build_id" "$component" "$tag" "FAILED" "TEMPLATE_ERROR ($arch_name)"
            return 1
        fi
    fi
    
    # ===== Dependency Configuration (External Base Image) =====
    local dep_conf="$component_dir/dependency.conf"
    if [ -f "$dep_conf" ]; then
        local upstream_image=$(grep -v '^#' "$dep_conf" | grep -v '^[[:space:]]*$' | head -n 1 | tr -d '[:space:]')
        if [ -z "$upstream_image" ]; then
            log_error "  [$arch_name] Empty dependency config for $component"
            return 1
        fi
        
        # Check if this component also has a Dockerfile (custom build based on dependency)
        if [ -f "$component_dir/Dockerfile" ]; then
            # Custom build that uses a base image from dependency.conf
            log_info "  [$arch_name] Processing $component: dependency + custom Dockerfile"
            log_info "  [$arch_name]   Base image: $upstream_image"
            
            # Pull the base image for the target platform first
            log_info "  [$arch_name] Pulling base image for $arch_name..."
            if ! docker pull --platform "$platform" "$upstream_image" >/dev/null 2>&1; then
                # Retry with verbose output
                log_warn "  [$arch_name] Retrying base image pull..."
                if ! docker pull --platform "$platform" "$upstream_image"; then
                    log_error "  [$arch_name] ✗ Failed to pull base image $upstream_image"
                    log_build_history "$build_id" "$component" "$tag" "FAILED" "BASE_PULL_ERROR ($arch_name)"
                    return 1
                fi
            fi
            log_info "  [$arch_name] ✓ Base image ready: $upstream_image"
            # Continue to Dockerfile build below (don't return)
        else
            # Pure dependency - just pull and tag for target platform
            local target_image="ai-infra-$component:${tag}"
            if [ -n "$PRIVATE_REGISTRY" ]; then
                target_image="$PRIVATE_REGISTRY/$target_image"
            fi
            
            log_info "  [$arch_name] Processing dependency $component: $upstream_image -> $target_image"
            
            if docker pull --platform "$platform" "$upstream_image" >/dev/null 2>&1; then
                if docker tag "$upstream_image" "$target_image" >/dev/null 2>&1; then
                    log_info "  [$arch_name] ✓ Dependency ready: $target_image"
                    log_build_history "$build_id" "$component" "$tag" "SUCCESS" "DEPENDENCY_PULLED ($arch_name)"
                    return 0
                else
                    log_error "  [$arch_name] ✗ Failed to tag $upstream_image"
                    log_build_history "$build_id" "$component" "$tag" "FAILED" "TAG_ERROR ($arch_name)"
                    return 1
                fi
            else
                log_error "  [$arch_name] ✗ Failed to pull $upstream_image"
                log_build_history "$build_id" "$component" "$tag" "FAILED" "PULL_ERROR ($arch_name)"
                return 1
            fi
        fi
    fi
    
    if [ ! -f "$component_dir/Dockerfile" ]; then
        log_warn "  [$arch_name] No Dockerfile or dependency.conf in $component, skipping..."
        return 0
    fi
    
    # Calculate service hash for build label
    local service_hash=$(calculate_service_hash "$component")
    
    # Check for build-targets.conf
    local targets_file="$component_dir/build-targets.conf"
    local targets=()
    local images=()
    
    if [ -f "$targets_file" ]; then
        while read -r target image_suffix || [ -n "$target" ]; do
            [[ "$target" =~ ^#.*$ ]] && continue
            [[ -z "$target" ]] && continue
            targets+=("$target")
            images+=("$image_suffix")
        done < "$targets_file"
    else
        targets+=("default")
        images+=("ai-infra-$component")
    fi
    
    for i in "${!targets[@]}"; do
        local target="${targets[$i]}"
        local image_name="${images[$i]}"
        
        # Use architecture-suffixed tag for ALL builds (方案一：统一镜像管理)
        # This allows building and storing both amd64 and arm64 images on the same machine
        # After build, we'll create unified tags for the native architecture
        local arch_suffix="-${arch_name}"
        
        local full_image_name="${image_name}:${tag}${arch_suffix}"
        
        if [ -n "$PRIVATE_REGISTRY" ]; then
            full_image_name="$PRIVATE_REGISTRY/$full_image_name"
        fi
        
        log_info "  [$arch_name] Building: $component [$target] -> $full_image_name"
        
        # For cross-platform builds using docker-container driver with network=host,
        # the buildkit container can access the network directly without proxy.
        # We don't pass proxy env vars to avoid DNS resolution issues with host.docker.internal
        local cmd=("docker" "buildx" "build")
        cmd+=("--builder" "$builder_name")  # Use the detected builder for cross-platform builds
        cmd+=("--platform" "$platform")
        
        # Network configuration: CRITICAL for arm64 (cross-platform) builds
        # arm64 on amd64 host needs explicit host network to avoid oauth token timeouts
        # Both amd64 and arm64 should use host network when available (multiarch-builder)
        if [[ "$builder_name" == "multiarch-builder" ]]; then
            cmd+=("--network" "host")  # Allow build steps to access host network directly
            cmd+=("--allow" "network.host")  # Grant network.host entitlement
        fi
        
        # =====================================================================
        # IMPORTANT: --load behavior differs between docker and docker-container drivers
        # 
        # - docker driver (desktop-linux): --load works reliably
        # - docker-container driver (multiarch-builder): --load may have issues
        #   especially in parallel builds or cross-platform scenarios
        #
        # We use --load for simplicity, but verify the result and retry if needed.
        # For docker-container driver, --output=type=docker is equivalent to --load
        # but some versions may handle it differently.
        # =====================================================================
        cmd+=("--load")  # Load to local docker daemon
        
        # Network proxy and mirror configuration for buildx docker-container driver
        # The buildkit container may be isolated from daemon.json settings
        # We need to explicitly pass proxy and registry mirror configs
        
        # Check if daemon.json has proxy settings
        local daemon_json="${HOME}/.docker/daemon.json"
        if [[ -f "$daemon_json" ]]; then
            # Extract proxy settings if present (httpProxy, httpsProxy, noProxy)
            local http_proxy=$(grep -o '"httpProxy": "[^"]*"' "$daemon_json" 2>/dev/null | cut -d'"' -f4)
            local https_proxy=$(grep -o '"httpsProxy": "[^"]*"' "$daemon_json" 2>/dev/null | cut -d'"' -f4)
            local no_proxy=$(grep -o '"noProxy": "[^"]*"' "$daemon_json" 2>/dev/null | cut -d'"' -f4)
            
            # Pass proxy settings to build environment
            if [[ -n "$http_proxy" ]]; then
                cmd+=("--build-arg" "HTTP_PROXY=$http_proxy")
                cmd+=("--build-arg" "http_proxy=$http_proxy")
                log_info "  [$arch_name] Using HTTP proxy: $http_proxy"
            fi
            if [[ -n "$https_proxy" ]]; then
                cmd+=("--build-arg" "HTTPS_PROXY=$https_proxy")
                cmd+=("--build-arg" "https_proxy=$https_proxy")
                log_info "  [$arch_name] Using HTTPS proxy: $https_proxy"
            fi
            if [[ -n "$no_proxy" ]]; then
                cmd+=("--build-arg" "NO_PROXY=$no_proxy")
                cmd+=("--build-arg" "no_proxy=$no_proxy")
            fi
        fi
        
        # Add environment variables from Docker daemon for registry mirrors and other settings
        # This ensures buildkit respects the same configurations as docker daemon
        if [[ -n "$http_proxy" || -n "$https_proxy" ]]; then
            log_info "  [$arch_name] ⚠️  Proxy detected, may slow down image pulls"
            log_info "  [$arch_name]   Ensure proxy server is running: http_proxy=$http_proxy"
        fi
        
        # Add --add-host to map 'apphub' to the actual container IP
        # This allows buildkit (running with host network) to access apphub container
        local apphub_ip=""
        local apphub_container_status=$(docker ps --filter "name=^ai-infra-apphub$" --filter "status=running" --format "{{.ID}}" 2>/dev/null || echo "")
        
        if [[ -z "$apphub_container_status" ]]; then
            log_warn "  [$arch_name] ⚠️  AppHub container not running, using default hostname"
            # Fall back to using hostname instead of IP
            cmd+=("--add-host" "apphub:127.0.0.1")
            log_info "  [$arch_name] Using apphub at 127.0.0.1 (fallback)"
        else
            # Get the first IP from the ai-infra-network
            apphub_ip=$(docker inspect -f '{{index .NetworkSettings.Networks "ai-infra-network" .IPAddress}}' ai-infra-apphub 2>/dev/null || echo "")
            
            # Validate IP address format (basic check for IPv4)
            if [[ $apphub_ip =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]]; then
                cmd+=("--add-host" "apphub:$apphub_ip")
                log_info "  [$arch_name] Using apphub at $apphub_ip"
            else
                log_warn "  [$arch_name] ⚠️  Invalid AppHub IP: '$apphub_ip', using localhost fallback"
                cmd+=("--add-host" "apphub:127.0.0.1")
                log_info "  [$arch_name] Using apphub at 127.0.0.1 (fallback)"
            fi
        fi
        
        # For docker-container driver with host network, we also need to handle localhost properly
        # Add docker.internal for compatibility (maps to host's localhost)
        if [[ "$builder_name" == "multiarch-builder" ]]; then
            cmd+=("--add-host" "host.docker.internal:127.0.0.1")
        fi
        
        # Add --no-cache if force build is enabled
        if [[ "$FORCE_BUILD" == "true" ]]; then
            cmd+=("--no-cache")
        fi
        
        # Add build cache labels for incremental builds
        cmd+=("--label" "build.hash=$service_hash")
        cmd+=("--label" "build.id=$build_id")
        cmd+=("--label" "build.timestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ)")
        cmd+=("--label" "build.component=$component")
        cmd+=("--label" "build.platform=$platform")
        
        cmd+=("${BASE_BUILD_ARGS[@]}" "${extra_args[@]}" "-t" "$full_image_name" "-f" "$component_dir/Dockerfile")
        
        if [ "$target" != "default" ]; then
            cmd+=("--target" "$target")
        fi
        
        cmd+=("$SCRIPT_DIR")
        
        # Debug: print the actual command being executed
        # log_info "DEBUG CMD: ${cmd[*]}"
        
        # For cross-platform builds, with network retry mechanism
        # Handle temporary network issues and transient failures
        local build_attempt=0
        local max_build_attempts=3
        local build_retry_delay=15
        local build_success=false
        
        while [[ $build_attempt -lt $max_build_attempts ]]; do
            build_attempt=$((build_attempt + 1))
            
            if [[ $build_attempt -gt 1 ]]; then
                log_warn "  [$arch_name] 🔄 Build retry attempt $build_attempt/$max_build_attempts (waiting ${build_retry_delay}s)..."
                sleep "$build_retry_delay"
                
                # Check network connectivity before retry
                if ! _check_network_connectivity >/dev/null 2>&1; then
                    log_warn "  [$arch_name] ⚠️  Network connectivity issues detected"
                    log_warn "  [$arch_name]    Waiting additional ${build_retry_delay}s before retry..."
                    sleep "$build_retry_delay"
                fi
            fi
            
            # Execute build command
            if "${cmd[@]}" 2>&1 | tee -a "$FAILURE_LOG"; then
                # Build succeeded, now verify image was loaded WITH CORRECT ARCHITECTURE
                if docker image inspect "$full_image_name" >/dev/null 2>&1; then
                    # CRITICAL: Also verify the loaded image has the correct architecture
                    local loaded_arch=$(docker image inspect "$full_image_name" --format '{{.Architecture}}' 2>/dev/null || echo "unknown")
                    if [[ "$loaded_arch" == "$arch_name" ]]; then
                        log_info "  [$arch_name] ✓ Built: $full_image_name (verified arch: $loaded_arch)"
                        log_build_history "$build_id" "$component" "$tag" "SUCCESS" "BUILT ($arch_name)"
                        save_service_build_info "$component" "$tag" "$build_id" "$service_hash"
                        build_success=true
                        break
                    else
                        log_warn "  [$arch_name] ⚠ Image loaded but wrong architecture: expected $arch_name, got $loaded_arch"
                        log_warn "  [$arch_name]   This may be a buildx --load issue with cross-platform builds"
                        # Don't mark as success, let it retry or fail
                    fi
                else
                    log_warn "  [$arch_name] ⚠ Image not found in Docker daemon after build"
                    if [[ $build_attempt -lt $max_build_attempts ]]; then
                        log_warn "  [$arch_name]   Retrying with --no-cache..."
                        # Retry with --no-cache to force re-export
                        # Build command array carefully to avoid argument issues
                        local retry_cmd=()
                        local found_no_cache=false
                        
                        for item in "${cmd[@]}"; do
                            if [[ "$item" == "--no-cache" ]]; then
                                found_no_cache=true
                            else
                                retry_cmd+=("$item")
                            fi
                        done
                        
                        # Always add --no-cache for retry
                        retry_cmd+=("--no-cache")
                        
                        # Verify command array is not empty
                        if [[ ${#retry_cmd[@]} -gt 0 ]]; then
                            if "${retry_cmd[@]}" 2>&1 | tee -a "$FAILURE_LOG"; then
                                if docker image inspect "$full_image_name" >/dev/null 2>&1; then
                                    # Also verify architecture after no-cache retry
                                    local loaded_arch=$(docker image inspect "$full_image_name" --format '{{.Architecture}}' 2>/dev/null || echo "unknown")
                                    if [[ "$loaded_arch" == "$arch_name" ]]; then
                                        log_info "  [$arch_name] ✓ Built (no-cache retry): $full_image_name (verified arch: $loaded_arch)"
                                        log_build_history "$build_id" "$component" "$tag" "SUCCESS" "BUILT_NOCACHE ($arch_name)"
                                        save_service_build_info "$component" "$tag" "$build_id" "$service_hash"
                                        build_success=true
                                        break
                                    else
                                        log_warn "  [$arch_name] ⚠ Image loaded but wrong architecture: expected $arch_name, got $loaded_arch"
                                    fi
                                fi
                            fi
                        else
                            log_error "  [$arch_name] ✗ Failed to build retry command (empty array)"
                        fi
                    fi
                fi
            else
                # Build failed - check error message to determine if it's a network issue
                local last_error=$(tail -20 "$FAILURE_LOG" 2>/dev/null | grep -i "network\|timeout\|connection\|read" || echo "unknown error")
                
                if [[ $build_attempt -lt $max_build_attempts ]]; then
                    if echo "$last_error" | grep -qiE "network|timeout|connection|read.*down"; then
                        log_warn "  [$arch_name] ⚠️  Network/timeout error detected: $last_error"
                        log_warn "  [$arch_name]   This may be transient, will retry..."
                        continue
                    else
                        log_error "  [$arch_name] ✗ Build failed (non-network error): $last_error"
                        log_build_history "$build_id" "$component" "$tag" "FAILED" "BUILD_ERROR ($arch_name)"
                        return 1
                    fi
                else
                    log_error "  [$arch_name] ✗ Failed after $max_build_attempts attempts: $full_image_name"
                    log_error "  [$arch_name]   Last error: $last_error"
                    log_build_history "$build_id" "$component" "$tag" "FAILED" "BUILD_FAILED_RETRIES ($arch_name)"
                    return 1
                fi
            fi
        done
        
        if [[ "$build_success" != "true" ]]; then
            log_error "  [$arch_name] ✗ Failed: $full_image_name (after $max_build_attempts attempts)"
            log_build_history "$build_id" "$component" "$tag" "FAILED" "BUILD_ERROR ($arch_name)"
            return 1
        fi
    done
    
    return 0
}

# ==============================================================================
# 自诊断和自修复系统
# ==============================================================================
# 在构建前对系统进行全面的诊断，并自动修复可以修复的问题

# 诊断 SSL 证书
diagnose_ssl_certificates() {
    log_info "  📋 Checking SSL certificates..."
    
    local ssl_dir="src/nginx/ssl"
    local crt_file="$ssl_dir/server.crt"
    local key_file="$ssl_dir/server.key"
    
    if [[ ! -f "$crt_file" ]] || [[ ! -f "$key_file" ]]; then
        log_warn "  ⚠️  SSL certificates not found in $ssl_dir"
        log_info "  🔧 Auto-generating SSL certificates..."
        
        if setup_ssl_certificates "" false; then
            log_info "  ✅ SSL certificates generated successfully"
            # Rebuild nginx to bundle new certificates
            log_info "  🔨 Rebuilding nginx with new certificates..."
            build_component "nginx"
            return 0
        else
            log_warn "  ⚠️  Failed to generate SSL certificates (will continue without SSL)"
            return 1
        fi
    else
        log_info "  ✅ SSL certificates found"
        
        # Check certificate validity
        if command -v openssl &> /dev/null; then
            local expiry_date=$(openssl x509 -in "$crt_file" -noout -enddate 2>/dev/null | cut -d= -f2)
            if [[ -n "$expiry_date" ]]; then
                log_info "     Certificate expires: $expiry_date"
            fi
        fi
        return 0
    fi
}

# 诊断 .env 文件
diagnose_env_file() {
    log_info "  📋 Checking environment configuration..."
    
    if [[ ! -f ".env" ]]; then
        log_warn "  ⚠️  .env file not found"
        log_info "  🔧 Initializing .env from .env.example..."
        
        if ! ./build.sh init-env; then
            log_error "  ✗ Failed to initialize .env"
            return 1
        fi
        
        log_info "  ✅ .env file initialized"
        return 0
    else
        log_info "  ✅ .env file exists"
        
        # Check for required variables
        local required_vars=("COMPOSE_PROJECT_NAME" "IMAGE_TAG" "EXTERNAL_HOST" "TZ")
        local missing_vars=()
        
        for var in "${required_vars[@]}"; do
            if ! grep -q "^$var=" .env; then
                missing_vars+=("$var")
            fi
        done
        
        if [[ ${#missing_vars[@]} -gt 0 ]]; then
            log_warn "  ⚠️  Missing environment variables: ${missing_vars[*]}"
            log_info "  🔧 Re-syncing .env with .env.example..."
            
            if ./build.sh render; then
                log_info "  ✅ Environment variables synced"
                return 0
            else
                log_warn "  ⚠️  Failed to sync .env (some variables may be missing)"
                return 1
            fi
        fi
        
        return 0
    fi
}

# 诊断 Docker 和 Docker Compose
diagnose_docker() {
    log_info "  📋 Checking Docker installation..."
    
    if ! command -v docker &> /dev/null; then
        log_error "  ✗ Docker not found"
        return 1
    fi
    
    local docker_version=$(docker --version 2>/dev/null)
    log_info "  ✅ Docker: $docker_version"
    
    # Check Docker daemon is running
    if ! docker ps &>/dev/null; then
        log_error "  ✗ Docker daemon is not running"
        return 1
    fi
    
    # Check Docker Compose
    local compose_cmd=$(detect_compose_command)
    if [[ -z "$compose_cmd" ]]; then
        log_error "  ✗ Docker Compose not found"
        return 1
    fi
    
    local compose_version=$($compose_cmd version 2>/dev/null | head -1)
    log_info "  ✅ Docker Compose: $compose_version"
    
    return 0
}

# 诊断镜像加速和网络配置
diagnose_network_and_mirrors() {
    log_info "  📋 Checking network and mirror configuration..."
    
    # Check daemon.json configuration
    local daemon_json="$HOME/.docker/daemon.json"
    if [[ -f "$daemon_json" ]]; then
        local mirror_count=$(grep -o '"registry-mirrors"' "$daemon_json" | wc -l)
        if [[ $mirror_count -gt 0 ]]; then
            local mirrors=$(grep -A10 '"registry-mirrors"' "$daemon_json" 2>/dev/null | grep -oE 'https?://[^"]+' | head -3)
            log_info "  ✅ Registry mirrors configured:"
            echo "$mirrors" | while read -r mirror; do
                log_info "     - $mirror"
            done
        fi
    fi
    
    # Check network connectivity
    log_info "  📡 Testing network connectivity..."
    if ping -c 1 8.8.8.8 &>/dev/null 2>&1; then
        log_info "  ✅ Internet connectivity OK"
    else
        log_warn "  ⚠️  Internet connectivity check failed"
    fi
    
    return 0
}

# 诊断磁盘空间
diagnose_disk_space() {
    log_info "  📋 Checking disk space..."
    
    local disk_usage=$(df -h . | tail -1 | awk '{print $5}')
    local disk_available=$(df -h . | tail -1 | awk '{print $4}')
    
    log_info "  ✅ Disk usage: $disk_usage available: $disk_available"
    
    # Warn if less than 10GB available
    local available_gb=$(df . | tail -1 | awk '{print int($4/1024/1024)}')
    if [[ $available_gb -lt 10 ]]; then
        log_warn "  ⚠️  Low disk space: Only ${available_gb}GB available (recommended: 50GB+)"
        return 1
    fi
    
    return 0
}

# 诊断数据库配置
diagnose_database() {
    log_info "  📋 Checking database configuration..."
    
    local db_type=$(grep "^DB_TYPE=" .env 2>/dev/null | cut -d'=' -f2)
    if [[ -n "$db_type" ]]; then
        log_info "  ✅ Database type: $db_type"
    fi
    
    # Check if database directory exists for local databases
    if [[ "$db_type" == "postgres" ]]; then
        if [[ -d "data/postgres" ]]; then
            log_info "  ✅ PostgreSQL data directory exists"
        else
            log_info "  ℹ️  PostgreSQL data directory will be created on first run"
        fi
    fi
    
    return 0
}

# 主诊断函数：执行所有诊断
pre_build_diagnosis() {
    local auto_fix="${1:-true}"  # 是否自动修复
    
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "🔍 Pre-Build System Diagnosis"
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    local diagnosis_passed=true
    
    # 执行各项诊断
    diagnose_env_file || diagnosis_passed=false
    diagnose_docker || diagnosis_passed=false
    diagnose_ssl_certificates || true  # SSL is optional
    diagnose_network_and_mirrors || true  # Network issues are not fatal
    diagnose_disk_space || diagnosis_passed=false
    diagnose_database || true  # Database issues are not fatal
    
    echo ""
    
    if [[ "$diagnosis_passed" == "true" ]]; then
        log_info "✅ All critical diagnostics passed"
        log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        return 0
    else
        log_error "❌ Some critical diagnostics failed"
        log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        return 1
    fi
}

# Tag private registry images as local images
# This allows docker-compose to find images that were pulled from a private registry
# and use them with local names (without the registry prefix)
#
# Two modes:
# 1. If PRIVATE_REGISTRY is set: tag images from that specific registry
# 2. Auto-detect mode: scan local images for any registry-prefixed ai-infra images
tag_private_images_as_local() {
    local private_registry="${PRIVATE_REGISTRY:-}"
    local image_tag="${IMAGE_TAG:-v0.3.8}"
    
    # Detect native platform for architecture suffix handling
    local native_platform=$(_detect_docker_platform)
    local native_arch="${native_platform##*/}"
    
    # List of ai-infra images that may need tagging
    local images=(
        "ai-infra-frontend"
        "ai-infra-backend"
        "ai-infra-backend-init"
        "ai-infra-nginx"
        "ai-infra-apphub"
        "ai-infra-saltstack"
        "ai-infra-slurm-master"
        "ai-infra-jupyterhub"
        "ai-infra-singleuser"
        "ai-infra-gitea"
        "ai-infra-nightingale"
        "ai-infra-prometheus"
        "ai-infra-test-containers"
    )
    
    local tagged=0
    local skipped=0
    
    log_info "Checking for images that need local tagging..."
    log_info "Native architecture: $native_arch"
    
    # Mode 1: If PRIVATE_REGISTRY is configured, use it directly
    if [[ -n "$private_registry" ]]; then
        log_info "Using configured private registry: ${private_registry}"
        
        for img in "${images[@]}"; do
            local private_image="${private_registry}${img}:${image_tag}"
            local local_image="${img}:${image_tag}"
            
            # Check if private image exists locally
            if docker image inspect "$private_image" &>/dev/null; then
                # Check if local image already exists
                if docker image inspect "$local_image" &>/dev/null; then
                    skipped=$((skipped + 1))
                else
                    # Tag private image as local
                    if docker tag "$private_image" "$local_image"; then
                        log_info "  ✓ Tagged: ${private_image} -> ${local_image}"
                        tagged=$((tagged + 1))
                    else
                        log_warn "  ✗ Failed to tag: ${private_image}"
                    fi
                fi
            fi
        done
    else
        # Mode 2: Auto-detect registry-prefixed images AND architecture-suffixed images
        log_info "Auto-detecting registry-prefixed and architecture-suffixed images..."
        
        for img in "${images[@]}"; do
            local local_image="${img}:${image_tag}"
            
            # Check if local image already exists WITH CORRECT ARCHITECTURE
            if docker image inspect "$local_image" &>/dev/null; then
                local existing_arch=$(docker image inspect "$local_image" --format '{{.Architecture}}' 2>/dev/null)
                if [[ "$existing_arch" == "$native_arch" ]]; then
                    skipped=$((skipped + 1))
                    continue
                else
                    # Image exists but wrong architecture - need to replace it
                    log_warn "  ⚠ $local_image exists but has wrong architecture: $existing_arch (expected $native_arch)"
                    docker rmi "$local_image" &>/dev/null || true
                fi
            fi
            
            local found_image=""
            
            # Priority 1: Check for architecture-suffixed image (from cross-platform builds)
            # This handles images built with --platform=amd64 on ARM64 Mac or vice versa
            local arch_suffixed_image="${img}:${image_tag}-${native_arch}"
            if docker image inspect "$arch_suffixed_image" &>/dev/null; then
                # Verify architecture matches
                local img_arch=$(docker image inspect "$arch_suffixed_image" --format '{{.Architecture}}' 2>/dev/null)
                if [[ "$img_arch" == "$native_arch" ]]; then
                    found_image="$arch_suffixed_image"
                fi
            fi
            
            # Priority 2: Search for any registry-prefixed version of this image
            # Pattern: */ai-infra-xxx:tag or */*/*/ai-infra-xxx:tag
            if [[ -z "$found_image" ]]; then
                found_image=$(docker images --format '{{.Repository}}:{{.Tag}}' | grep "/${img}:${image_tag}$" | head -1)
            fi
            
            if [[ -n "$found_image" ]]; then
                if docker tag "$found_image" "$local_image"; then
                    log_info "  ✓ Tagged: ${found_image} -> ${local_image}"
                    tagged=$((tagged + 1))
                else
                    log_warn "  ✗ Failed to tag: ${found_image}"
                fi
            fi
        done
    fi
    
    if [[ $tagged -gt 0 ]]; then
        log_info "Image tagging complete: $tagged tagged, $skipped already exist"
    elif [[ $skipped -gt 0 ]]; then
        log_info "All $skipped images already exist locally"
    else
        log_info "No registry-prefixed or architecture-suffixed images found to tag"
    fi
}

# 更新运行时环境变量（启动阶段使用）
# 与构建阶段不同，运行时需要检测当前机器的真实 IP
update_runtime_env() {
    log_info "=========================================="
    log_info "🔄 更新运行时环境变量"
    log_info "=========================================="
    
    # 检测当前机器的外部地址
    local detected_host=$(detect_external_host)
    local current_host=$(grep "^EXTERNAL_HOST=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2)
    
    log_info "当前配置的 EXTERNAL_HOST: ${current_host:-<未设置>}"
    log_info "检测到的本机地址: $detected_host"
    
    # 如果 IP 不同，说明是在不同机器上运行
    if [[ "$current_host" != "$detected_host" ]]; then
        log_info "⚠️  检测到环境变化（可能是从其他机器构建的镜像）"
        log_info "   正在更新 EXTERNAL_HOST: $current_host -> $detected_host"
        
        # 更新 .env 文件中的 EXTERNAL_HOST
        update_env_variable "EXTERNAL_HOST" "$detected_host"
        update_env_variable "DOMAIN" "$detected_host"
        
        # 重新加载环境变量
        set -a
        source "$ENV_FILE"
        set +a
        
        # 重新渲染配置模板
        log_info "🔧 重新渲染配置模板..."
        render_all_templates "true"
        
        log_info "✓ 运行时环境变量已更新"
    else
        log_info "✓ EXTERNAL_HOST 配置正确，无需更新"
    fi
}

# ==============================================================================
# Database Safety Functions - 数据库安全函数
# ==============================================================================

# 检查 PostgreSQL 数据库是否包含生产数据
# Returns: 0 if has data, 1 if empty/not exists
check_postgres_has_data() {
    local db_name="${1:-ai_infra}"
    local db_host="${DB_HOST:-postgres}"
    local db_port="${DB_PORT:-5432}"
    local db_user="${DB_USER:-postgres}"
    local db_password="${DB_PASSWORD:-postgres}"
    
    log_info "🔍 Checking PostgreSQL database for existing data..."
    
    # 检查 postgres 容器是否运行
    if ! docker ps --format '{{.Names}}' | grep -q "ai-infra-postgres"; then
        log_info "PostgreSQL container not running, no data check needed"
        return 1
    fi
    
    # 检查数据库是否存在
    local db_exists=$(docker exec ai-infra-postgres psql -U "$db_user" -tAc \
        "SELECT 1 FROM pg_database WHERE datname='$db_name'" 2>/dev/null || echo "0")
    
    if [[ "$db_exists" != "1" ]]; then
        log_info "Database '$db_name' does not exist"
        return 1
    fi
    
    # 检查关键表中是否有数据
    local total_records=0
    local critical_tables=("users" "roles" "clusters" "tasks" "gpu_configs")
    
    for table in "${critical_tables[@]}"; do
        local count=$(docker exec ai-infra-postgres psql -U "$db_user" -d "$db_name" -tAc \
            "SELECT COUNT(*) FROM $table" 2>/dev/null || echo "0")
        count=${count//[^0-9]/}  # 移除非数字字符
        total_records=$((total_records + ${count:-0}))
    done
    
    if [[ $total_records -gt 0 ]]; then
        log_warn "⚠️  Found $total_records records in critical tables"
        return 0
    else
        log_info "Database exists but has no critical data"
        return 1
    fi
}

# 备份 PostgreSQL 数据库
backup_postgres_database() {
    local db_name="${1:-ai_infra}"
    local backup_dir="${2:-./backup/postgres}"
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local backup_file="${backup_dir}/${db_name}_${timestamp}.sql"
    
    log_info "📦 Creating PostgreSQL backup..."
    
    # 确保备份目录存在
    mkdir -p "$backup_dir"
    
    # 检查 postgres 容器是否运行
    if ! docker ps --format '{{.Names}}' | grep -q "ai-infra-postgres"; then
        log_error "PostgreSQL container not running, cannot backup"
        return 1
    fi
    
    # 执行备份
    local db_user="${DB_USER:-postgres}"
    if docker exec ai-infra-postgres pg_dump -U "$db_user" -d "$db_name" > "$backup_file" 2>/dev/null; then
        # 压缩备份
        gzip "$backup_file"
        log_info "✅ Backup created: ${backup_file}.gz"
        
        # 清理旧备份（保留最近10个）
        local backup_count=$(ls -1 "$backup_dir"/${db_name}_*.sql.gz 2>/dev/null | wc -l)
        if [[ $backup_count -gt 10 ]]; then
            ls -1t "$backup_dir"/${db_name}_*.sql.gz | tail -n +11 | xargs rm -f
            log_info "🧹 Cleaned old backups, keeping 10 most recent"
        fi
        
        return 0
    else
        log_error "Failed to create backup"
        return 1
    fi
}

# 恢复 PostgreSQL 数据库
restore_postgres_database() {
    local backup_file="$1"
    local db_name="${2:-ai_infra}"
    
    if [[ ! -f "$backup_file" ]]; then
        log_error "Backup file not found: $backup_file"
        return 1
    fi
    
    log_info "🔄 Restoring PostgreSQL database from backup..."
    
    local db_user="${DB_USER:-postgres}"
    
    # 如果是压缩文件，先解压
    if [[ "$backup_file" == *.gz ]]; then
        gunzip -c "$backup_file" | docker exec -i ai-infra-postgres psql -U "$db_user" -d "$db_name"
    else
        docker exec -i ai-infra-postgres psql -U "$db_user" -d "$db_name" < "$backup_file"
    fi
    
    if [[ $? -eq 0 ]]; then
        log_info "✅ Database restored successfully"
        return 0
    else
        log_error "Failed to restore database"
        return 1
    fi
}

# 获取数据库初始化模式
get_db_init_mode() {
    local mode="${DB_INIT_MODE:-safe_init}"
    echo "$mode"
}

# 设置数据库初始化模式
set_db_init_mode() {
    local mode="$1"
    export DB_INIT_MODE="$mode"
    log_info "DB_INIT_MODE set to: $mode"
}

# 交互式确认数据库重置
confirm_database_reset() {
    local db_name="${1:-ai_infra}"
    
    echo ""
    log_warn "═══════════════════════════════════════════════════════════════"
    log_warn "⚠️  DATABASE RESET WARNING"
    log_warn "═══════════════════════════════════════════════════════════════"
    log_warn "Database '$db_name' contains production data!"
    log_warn ""
    log_warn "Options:"
    log_warn "  1. BACKUP and RESET - Create backup, then reset database"
    log_warn "  2. UPGRADE ONLY     - Keep data, only run migrations"
    log_warn "  3. ABORT            - Cancel operation"
    log_warn "═══════════════════════════════════════════════════════════════"
    echo ""
    
    read -p "Enter choice [1/2/3]: " choice
    
    case "$choice" in
        1)
            log_info "Selected: Backup and Reset"
            backup_postgres_database "$db_name"
            set_db_init_mode "force_reset"
            return 0
            ;;
        2)
            log_info "Selected: Upgrade Only"
            set_db_init_mode "upgrade"
            return 0
            ;;
        3|*)
            log_info "Operation aborted by user"
            return 1
            ;;
    esac
}

# 安全检查包装函数 - 用于 build-all 和 start-all
pre_deployment_safety_check() {
    local force="${1:-false}"
    
    # 如果设置了 SKIP_DB_CHECK，跳过检查
    if [[ "${SKIP_DB_CHECK:-false}" == "true" ]]; then
        log_info "Database safety check skipped (SKIP_DB_CHECK=true)"
        return 0
    fi
    
    # 检查是否有生产数据
    if check_postgres_has_data; then
        if [[ "$force" == "true" ]]; then
            log_warn "⚠️  Force mode enabled with existing data!"
            
            # 非交互模式下，根据 DB_INIT_MODE 决定行为
            local init_mode=$(get_db_init_mode)
            if [[ "$init_mode" == "force_reset" ]]; then
                log_warn "DB_INIT_MODE=force_reset, proceeding with backup and reset"
                backup_postgres_database
                return 0
            elif [[ "$init_mode" == "upgrade" ]]; then
                log_info "DB_INIT_MODE=upgrade, keeping existing data"
                return 0
            else
                # 安全模式 - 交互确认
                if [[ -t 0 ]]; then
                    # 终端模式，交互确认
                    if ! confirm_database_reset; then
                        return 1
                    fi
                else
                    # 非交互模式，默认安全（不重置）
                    log_warn "Non-interactive mode with existing data, using safe mode"
                    set_db_init_mode "safe_init"
                    return 0
                fi
            fi
        else
            log_info "Existing data detected, using upgrade mode"
            set_db_init_mode "upgrade"
        fi
    else
        log_info "No existing production data, proceeding with initialization"
        set_db_init_mode "safe_init"
    fi
    
    return 0
}

start_all() {
    log_info "Starting all services (with HA profile for SaltStack multi-master)..."
    local compose_cmd=$(detect_compose_command)
    if [ -z "$compose_cmd" ]; then
        log_error "docker-compose not found!"
        exit 1
    fi
    
    # 【架构检查】确保镜像与本机 CPU 架构匹配
    log_info "=== Architecture Compatibility Check ==="
    if ! check_images_architecture; then
        log_error "Architecture check failed. Please rebuild images for your platform."
        log_info "Run: ./build.sh build-all"
        exit 1
    fi
    
    # 【防御性检查】启动前检查数据库状态
    log_info "=== Pre-deployment Safety Check ==="
    if ! pre_deployment_safety_check "false"; then
        log_error "Safety check failed or aborted. Exiting."
        exit 1
    fi
    
    # 【关键】在启动前更新运行时环境变量
    # 这解决了构建阶段与运行阶段在不同机器上 IP 不一致的问题
    update_runtime_env
    
    # Tag private registry images as local if needed
    tag_private_images_as_local
    
    # 先清理可能残留的旧容器（避免 "No such container" 错误）
    log_info "Cleaning up any stale containers..."
    docker rm -f ai-infra-salt-master-1 ai-infra-salt-master-2 >/dev/null 2>&1 || true
    
    # 使用重试机制启动服务
    local max_retries=3
    local retry_count=0
    local success=false
    
    while [[ $retry_count -lt $max_retries ]] && [[ "$success" != "true" ]]; do
        retry_count=$((retry_count + 1))
        log_info "Starting services (attempt $retry_count/$max_retries)..."
        
        # Use --no-build to prevent rebuilding when images already exist
        # Use --pull never to prevent checking remote registry (important for offline/intranet environments)
        # Use --profile ha to enable SaltStack multi-master high availability
        if $compose_cmd --profile ha up -d --no-build --pull never 2>&1; then
            success=true
            log_info "All services started successfully (SaltStack HA enabled)."
        else
            if [[ $retry_count -lt $max_retries ]]; then
                log_warn "Failed to start services, retrying in 5 seconds..."
                sleep 5
                # 清理可能的残留容器
                docker rm -f ai-infra-salt-master-1 ai-infra-salt-master-2 >/dev/null 2>&1 || true
            else
                log_error "Failed to start services after $max_retries attempts"
                return 1
            fi
        fi
    done
    
    # 验证关键服务是否运行
    log_info "Verifying services..."
    sleep 3
    local running_containers=$(docker ps --format '{{.Names}}' 2>/dev/null | grep -c "ai-infra" || echo "0")
    log_info "Running AI-Infra containers: $running_containers"
}

# ==============================================================================
# Clean Functions - 清理功能
# ==============================================================================

# Clean project images
# Args: $1 = tag (optional), $2 = force (optional)
clean_images() {
    local tag="${1:-}"
    local force="${2:-false}"
    
    log_info "=========================================="
    log_info "Cleaning AI-Infra Docker images"
    log_info "=========================================="
    
    local images_to_remove=()
    
    # Find all ai-infra images
    if [[ -n "$tag" ]]; then
        log_info "Finding images with tag: $tag"
        while IFS= read -r img; do
            [[ -n "$img" ]] && images_to_remove+=("$img")
        done < <(docker images --format '{{.Repository}}:{{.Tag}}' | grep "ai-infra" | grep ":${tag}$")
    else
        log_info "Finding all ai-infra images"
        while IFS= read -r img; do
            [[ -n "$img" ]] && images_to_remove+=("$img")
        done < <(docker images --format '{{.Repository}}:{{.Tag}}' | grep "ai-infra")
    fi
    
    if [[ ${#images_to_remove[@]} -eq 0 ]]; then
        log_info "No ai-infra images found to clean"
        return 0
    fi
    
    log_info "Found ${#images_to_remove[@]} images to remove:"
    for img in "${images_to_remove[@]}"; do
        echo "  • $img"
    done
    
    if [[ "$force" != "true" ]]; then
        echo
        read -p "Are you sure you want to remove these images? [y/N] " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Cancelled"
            return 0
        fi
    fi
    
    local removed=0
    local failed=0
    
    for img in "${images_to_remove[@]}"; do
        if docker rmi "$img" 2>/dev/null; then
            log_info "  ✓ Removed: $img"
            removed=$((removed + 1))
        else
            log_warn "  ✗ Failed to remove: $img (may be in use)"
            failed=$((failed + 1))
        fi
    done
    
    log_info "=========================================="
    log_info "Removed: $removed, Failed: $failed"
    return 0
}

# Clean project volumes
# Args: $1 = force (optional)
clean_volumes() {
    local force="${1:-false}"
    
    log_info "=========================================="
    log_info "Cleaning AI-Infra Docker volumes"
    log_info "=========================================="
    
    local volumes_to_remove=()
    
    # Find all ai-infra related volumes
    while IFS= read -r vol; do
        [[ -n "$vol" ]] && volumes_to_remove+=("$vol")
    done < <(docker volume ls --format '{{.Name}}' | grep -E "ai-infra|ai_infra")
    
    # Also check for compose project volumes
    local compose_project="ai-infra-matrix"
    while IFS= read -r vol; do
        [[ -n "$vol" ]] && volumes_to_remove+=("$vol")
    done < <(docker volume ls --format '{{.Name}}' | grep -E "^${compose_project}_")
    
    # Remove duplicates
    volumes_to_remove=($(printf '%s\n' "${volumes_to_remove[@]}" | sort -u))
    
    if [[ ${#volumes_to_remove[@]} -eq 0 ]]; then
        log_info "No ai-infra volumes found to clean"
        return 0
    fi
    
    log_info "Found ${#volumes_to_remove[@]} volumes to remove:"
    for vol in "${volumes_to_remove[@]}"; do
        echo "  • $vol"
    done
    
    if [[ "$force" != "true" ]]; then
        echo
        read -p "Are you sure you want to remove these volumes? This will DELETE ALL DATA! [y/N] " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Cancelled"
            return 0
        fi
    fi
    
    local removed=0
    local failed=0
    
    for vol in "${volumes_to_remove[@]}"; do
        if docker volume rm "$vol" 2>/dev/null; then
            log_info "  ✓ Removed: $vol"
            removed=$((removed + 1))
        else
            log_warn "  ✗ Failed to remove: $vol (may be in use)"
            failed=$((failed + 1))
        fi
    done
    
    log_info "=========================================="
    log_info "Removed: $removed, Failed: $failed"
    return 0
}

# Stop all project containers
stop_all() {
    log_info "Stopping all AI-Infra services..."
    local compose_cmd=$(detect_compose_command)
    if [ -z "$compose_cmd" ]; then
        log_error "docker-compose not found!"
        return 1
    fi
    
    # 停止所有 profile 下的服务（ha, safeline 等）
    # 需要同时指定所有可能的 profile 才能确保完全停止
    $compose_cmd --profile ha --profile safeline down 2>/dev/null || true
    
    # 作为兜底，再执行不带 profile 的 down
    $compose_cmd down 2>/dev/null || true
    
    # 清理可能残留的 HA 容器
    if docker ps -a --format '{{.Names}}' 2>/dev/null | grep -q '^ai-infra-salt-master-2$'; then
        log_info "Cleaning up HA container: ai-infra-salt-master-2"
        docker rm -f ai-infra-salt-master-2 >/dev/null 2>&1 || true
    fi
    
    # 清理可能残留的 SafeLine 容器
    local safeline_containers=$(docker ps -a --format '{{.Names}}' 2>/dev/null | grep '^safeline-' || true)
    if [[ -n "$safeline_containers" ]]; then
        log_info "Cleaning up SafeLine containers..."
        echo "$safeline_containers" | xargs -r docker rm -f >/dev/null 2>&1 || true
    fi
    
    log_info "All services stopped."
}

# Clean all: stop containers, remove images and volumes
# Args: $1 = force (optional, "--force" or "true")
clean_all() {
    local force="false"
    
    if [[ "$1" == "--force" || "$1" == "-f" || "$1" == "true" ]]; then
        force="true"
    fi
    
    if [[ "$1" == "--help" || "$1" == "-h" ]]; then
        echo "clean-all - Clean all project Docker resources"
        echo ""
        echo "Usage: $0 clean-all [--force]"
        echo ""
        echo "Options:"
        echo "  --force, -f    Skip confirmation prompts"
        echo ""
        echo "This command will:"
        echo "  1. Stop all running containers"
        echo "  2. Remove all ai-infra Docker images"
        echo "  3. Remove all ai-infra Docker volumes"
        echo "  4. Clean dangling images and build cache"
        echo ""
        echo "⚠️  WARNING: This will DELETE ALL DATA in volumes!"
        return 0
    fi
    
    log_info "=========================================="
    log_info "🧹 Complete cleanup of AI-Infra resources"
    log_info "=========================================="
    
    if [[ "$force" != "true" ]]; then
        echo
        log_warn "⚠️  This will stop all containers, remove all images and DELETE ALL DATA!"
        read -p "Are you sure you want to continue? [y/N] " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Cancelled"
            return 0
        fi
        # Set force=true for subsequent operations to avoid repeated prompts
        force="true"
    fi
    
    echo
    log_info "Step 1/4: Stopping all containers..."
    stop_all 2>/dev/null || log_warn "No containers to stop or compose not available"
    
    # 清理可能存在的 HA 模式容器（不在 docker-compose 标准管理下的容器）
    # 使用 >/dev/null 2>&1 同时重定向 stdout 和 stderr 以完全静默输出
    if docker ps -a --format '{{.Names}}' 2>/dev/null | grep -q '^ai-infra-salt-master-2$'; then
        log_info "Cleaning up HA profile container: ai-infra-salt-master-2"
        docker rm -f ai-infra-salt-master-2 >/dev/null 2>&1 || true
    fi
    
    echo
    log_info "Step 2/4: Removing project images..."
    clean_images "" "$force"
    
    echo
    log_info "Step 3/4: Removing project volumes..."
    clean_volumes "$force"
    
    echo
    log_info "Step 4/4: Cleaning dangling resources..."
    # Remove dangling images
    local dangling_count=$(docker images -f "dangling=true" -q | wc -l | tr -d ' ')
    if [[ "$dangling_count" -gt 0 ]]; then
        log_info "Removing $dangling_count dangling images..."
        docker image prune -f 2>/dev/null || true
    fi
    
    # Clean build cache (optional, only if --force)
    if [[ "$force" == "true" ]]; then
        log_info "Cleaning build cache..."
        docker builder prune -f 2>/dev/null || true
    fi
    
    echo
    log_info "=========================================="
    log_info "🎉 Cleanup completed!"
    log_info "=========================================="
    
    # Show remaining resources - use tr to remove newlines and ensure clean numeric output
    local remaining_images
    local remaining_volumes
    remaining_images=$(docker images --format '{{.Repository}}:{{.Tag}}' 2>/dev/null | grep "ai-infra" 2>/dev/null | wc -l | tr -d ' \n' || echo "0")
    remaining_volumes=$(docker volume ls --format '{{.Name}}' 2>/dev/null | grep -E "ai-infra|ai_infra" 2>/dev/null | wc -l | tr -d ' \n' || echo "0")
    
    # Ensure numeric values (default to 0 if empty)
    [[ -z "$remaining_images" ]] && remaining_images=0
    [[ -z "$remaining_volumes" ]] && remaining_volumes=0
    
    if [[ "$remaining_images" != "0" ]] || [[ "$remaining_volumes" != "0" ]]; then
        log_warn "Some resources could not be removed (may be in use):"
        [[ "$remaining_images" != "0" ]] && log_warn "  Images: $remaining_images"
        [[ "$remaining_volumes" != "0" ]] && log_warn "  Volumes: $remaining_volumes"
    fi
    
    return 0
}

# ==============================================================================
# Export Offline Images
# ==============================================================================

# Export all images to tar files for offline deployment
# Args: $1 = output_dir (default: ./offline-images), $2 = tag, $3 = include common images (default: true)
export_offline_images() {
    local output_dir="${1:-./offline-images}"
    local tag="${2:-${IMAGE_TAG:-latest}}"
    local include_common="${3:-true}"
    local platforms="${4:-amd64,arm64}"
    
    # Show help
    if [[ "$1" == "--help" ]] || [[ "$1" == "-h" ]]; then
        echo "Usage: $0 export-offline [output_dir] [tag] [include_common] [platforms]"
        echo ""
        echo "Arguments:"
        echo "  output_dir      Output directory (default: ./offline-images)"
        echo "  tag             Image tag (default: $IMAGE_TAG)"
        echo "  include_common  Include common images like mysql, redis, kafka (default: true)"
        echo "  platforms       Comma-separated architectures to export (default: amd64,arm64)"
        echo "                  Supported: amd64, arm64, or both"
        echo ""
        echo "Description:"
        echo "  Export all AI-Infra service images and dependency images to tar files"
        echo "  Supports multi-architecture export for offline deployment"
        echo "  Automatically generates image manifest and import script"
        echo ""
        echo "Examples:"
        echo "  $0 export-offline ./my-images v0.3.8 true amd64,arm64  # Both architectures"
        echo "  $0 export-offline ./images v0.3.8 true amd64           # AMD64 only"
        echo "  $0 export-offline ./images v0.3.8 false arm64          # ARM64 only, no common"
        return 0
    fi
    
    # Parse platforms into array
    IFS=',' read -ra PLATFORM_ARRAY <<< "$platforms"
    local valid_platforms=()
    for p in "${PLATFORM_ARRAY[@]}"; do
        p=$(echo "$p" | tr -d '[:space:]')
        case "$p" in
            amd64|x86_64) valid_platforms+=("linux/amd64") ;;
            arm64|aarch64) valid_platforms+=("linux/arm64") ;;
            *) log_warn "Unknown platform: $p, skipping" ;;
        esac
    done
    
    if [[ ${#valid_platforms[@]} -eq 0 ]]; then
        log_error "No valid platforms specified"
        return 1
    fi
    
    log_info "=========================================="
    log_info "📦 Exporting Offline Images (Multi-Arch)"
    log_info "=========================================="
    log_info "Output directory: $output_dir"
    log_info "Image tag: $tag"
    log_info "Include common images: $include_common"
    log_info "Target platforms: ${valid_platforms[*]}"
    echo
    
    # Create output directories for each platform
    mkdir -p "$output_dir"
    for platform in "${valid_platforms[@]}"; do
        local arch_name="${platform##*/}"
        mkdir -p "${output_dir}/${arch_name}"
    done
    
    discover_services
    
    local exported_count=0
    local failed_count=0
    local failed_images=()
    
    # Helper function to export single image for specific platform
    # This function expects images to be pre-pulled (use pull-all --platform=xxx first)
    # Note: On Docker Desktop (Mac/Windows), docker inspect may return incorrect architecture
    # for cross-platform pulled images. We trust that pull-all --platform=xxx did the right thing.
    _export_image_for_platform() {
        local image_name="$1"
        local platform="$2"
        local arch_name="${platform##*/}"
        local safe_name=$(echo "$image_name" | sed 's|/|-|g' | sed 's|:|_|g')
        local output_file="${output_dir}/${arch_name}/${safe_name}.tar"
        
        # Check if image is a local build (ai-infra-*) or remote image
        local is_local_image=false
        if [[ "$image_name" == ai-infra-* ]]; then
            is_local_image=true
        fi
        
        # For local images, check if it exists locally
        if $is_local_image; then
            if ! docker image inspect "$image_name" >/dev/null 2>&1; then
                echo "not_found"
                return
            fi
            # Verify architecture matches (only for local builds where we control the tag)
            local actual_arch=$(docker image inspect "$image_name" --format '{{.Architecture}}' 2>/dev/null)
            if [[ -n "$actual_arch" ]] && [[ "$actual_arch" != "$arch_name" ]]; then
                echo "arch_mismatch:$actual_arch"
                return
            fi
            # Export local image
            if docker save "$image_name" -o "$output_file" 2>/dev/null; then
                du -h "$output_file" | cut -f1
            else
                echo "failed"
            fi
        else
            # For remote images: Docker Desktop with containerd image store has a known issue
            # where "docker save" fails with "content digest not found" for multi-arch images.
            # 
            # Solution: Use "docker buildx build --output type=docker" to create a single-arch
            # image that can be properly saved. This essentially re-packages the image layers
            # into a format compatible with docker save.
            # 
            # We output with the original image name so docker load will create the correct tag.
            
            # Use buildx to create a loadable single-arch image with original name
            # The "FROM image" dockerfile trick forces buildx to resolve to the specific platform
            if echo "FROM $image_name" | docker buildx build \
                --platform="$platform" \
                --output "type=docker,name=${image_name}" \
                --quiet \
                -f - . >/dev/null 2>&1; then
                
                # Now save the image (buildx should have replaced the multi-arch manifest with single-arch)
                if docker save "$image_name" -o "$output_file" 2>/dev/null; then
                    du -h "$output_file" | cut -f1
                else
                    echo "failed"
                fi
            else
                # Buildx failed, try direct pull+save as fallback (may work on native Docker)
                if docker pull --platform="$platform" "$image_name" -q >/dev/null 2>&1; then
                    if docker save "$image_name" -o "$output_file" 2>/dev/null; then
                        du -h "$output_file" | cut -f1
                    else
                        echo "failed"
                    fi
                else
                    echo "not_found"
                fi
            fi
        fi
    }
    
    # Phase 1: Export AI-Infra project images
    log_info "=== Phase 1: Exporting AI-Infra service images ==="
    log_info "Note: Checking both native and cross-platform built images"
    
    local all_services=("${FOUNDATION_SERVICES[@]}" "${DEPENDENT_SERVICES[@]}")
    local native_platform=$(_detect_docker_platform)
    local native_arch="${native_platform##*/}"
    
    for service in "${all_services[@]}"; do
        log_info "→ Exporting: ai-infra-${service}"
        
        # Try to export for each requested platform
        for platform in "${valid_platforms[@]}"; do
            local arch_name="${platform##*/}"
            local arch_suffix="-${arch_name}"
            local image_name=""
            local image_found=false
            
            # Priority order for finding images:
            # 1. Check architecture-suffixed tag first (cross-platform builds always use suffix)
            # 2. Then check base tag (native builds)
            local suffixed_image="ai-infra-${service}:${tag}${arch_suffix}"
            local base_image="ai-infra-${service}:${tag}"
            
            # Try suffixed image first (preferred for clarity)
            if docker image inspect "$suffixed_image" >/dev/null 2>&1; then
                image_name="$suffixed_image"
                image_found=true
            elif docker image inspect "$base_image" >/dev/null 2>&1; then
                # Verify base image has correct architecture
                local base_arch=$(docker image inspect "$base_image" --format '{{.Architecture}}' 2>/dev/null)
                if [[ "$base_arch" == "$arch_name" ]]; then
                    image_name="$base_image"
                    image_found=true
                fi
            fi
            
            local safe_name=$(echo "ai-infra-${service}_${tag}" | sed 's|:|_|g')
            local output_file="${output_dir}/${arch_name}/${safe_name}.tar"
            
            if $image_found; then
                # Verify the image architecture matches what we expect
                local actual_arch=$(docker image inspect "$image_name" --format '{{.Architecture}}' 2>/dev/null)
                if [[ "$actual_arch" == "$arch_name" ]] || [[ "$actual_arch" == "amd64" && "$arch_name" == "amd64" ]] || [[ "$actual_arch" == "arm64" && "$arch_name" == "arm64" ]]; then
                    if docker save "$image_name" -o "$output_file"; then
                        local file_size=$(du -h "$output_file" | cut -f1)
                        log_info "  ✓ [$arch_name] Exported: $(basename "$output_file") ($file_size)"
                        exported_count=$((exported_count + 1))
                    else
                        log_warn "  ✗ [$arch_name] Failed to export: $image_name"
                        failed_images+=("${image_name}@${arch_name}")
                        failed_count=$((failed_count + 1))
                    fi
                else
                    log_warn "  ! [$arch_name] Image $image_name has wrong architecture: $actual_arch (expected $arch_name)"
                    failed_images+=("${image_name}@${arch_name}")
                    failed_count=$((failed_count + 1))
                fi
            else
                log_warn "  ✗ [$arch_name] Image not found (tried: $suffixed_image, $base_image)"
                failed_images+=("ai-infra-${service}:${tag}@${arch_name}")
                failed_count=$((failed_count + 1))
            fi
        done
    done
    echo
    
    # Phase 2: Export dependency images (from deps.yaml mapping) - multi-arch
    log_info "=== Phase 2: Exporting dependency images (multi-arch) ==="
    log_info "Note: Run './build.sh pull-all --platform=xxx' first to pull correct architecture"
    local dependencies=($(get_dependency_mappings))
    
    for mapping in "${dependencies[@]}"; do
        local source_image="${mapping%%|*}"
        local short_name="${mapping##*|}"
        
        log_info "→ Exporting: $source_image"
        
        for platform in "${valid_platforms[@]}"; do
            local arch_name="${platform##*/}"
            local result=$(_export_image_for_platform "$source_image" "$platform")
            
            case "$result" in
                not_found)
                    log_warn "  ! [$arch_name] Image not found (run: ./build.sh pull-all --platform=$arch_name)"
                    failed_images+=("${source_image}@${arch_name}")
                    failed_count=$((failed_count + 1))
                    ;;
                arch_mismatch:*)
                    local actual_arch="${result#arch_mismatch:}"
                    log_warn "  ! [$arch_name] Architecture mismatch: $actual_arch (expected $arch_name)"
                    log_warn "    → Run: ./build.sh pull-all --platform=$arch_name"
                    failed_images+=("${source_image}@${arch_name}")
                    failed_count=$((failed_count + 1))
                    ;;
                failed)
                    log_warn "  ✗ [$arch_name] Failed to export"
                    failed_images+=("${source_image}@${arch_name}")
                    failed_count=$((failed_count + 1))
                    ;;
                *)
                    log_info "  ✓ [$arch_name] Exported ($result)"
                    exported_count=$((exported_count + 1))
                    ;;
            esac
        done
    done
    echo
    
    # Phase 3: Export common/third-party images - multi-arch
    if [[ "$include_common" == "true" ]]; then
        log_info "=== Phase 3: Exporting common/third-party images (multi-arch) ==="
        log_info "Note: Run './build.sh pull-all --platform=xxx' first to pull correct architecture"
        
        for image in "${COMMON_IMAGES[@]}"; do
            log_info "→ Exporting: $image"
            
            for platform in "${valid_platforms[@]}"; do
                local arch_name="${platform##*/}"
                local result=$(_export_image_for_platform "$image" "$platform")
                
                case "$result" in
                    not_found)
                        log_warn "  ! [$arch_name] Image not found (run: ./build.sh pull-all --platform=$arch_name)"
                        failed_images+=("${image}@${arch_name}")
                        failed_count=$((failed_count + 1))
                        ;;
                    arch_mismatch:*)
                        local actual_arch="${result#arch_mismatch:}"
                        log_warn "  ! [$arch_name] Architecture mismatch: $actual_arch (expected $arch_name)"
                        log_warn "    → Run: ./build.sh pull-all --platform=$arch_name"
                        failed_images+=("${image}@${arch_name}")
                        failed_count=$((failed_count + 1))
                        ;;
                    failed)
                        log_warn "  ✗ [$arch_name] Failed to export"
                        failed_images+=("${image}@${arch_name}")
                        failed_count=$((failed_count + 1))
                        ;;
                    *)
                        log_info "  ✓ [$arch_name] Exported ($result)"
                        exported_count=$((exported_count + 1))
                        ;;
                esac
            done
        done
        echo
    fi
    
    # Generate image manifest file for each architecture
    log_info "📋 Generating image manifests..."
    
    for platform in "${valid_platforms[@]}"; do
        local arch_name="${platform##*/}"
        local manifest_file="${output_dir}/${arch_name}/images-manifest.txt"
        
        cat > "$manifest_file" << EOF
# AI Infrastructure Matrix - Offline Images Manifest
# Generated: $(date)
# Image Tag: $tag
# Architecture: $arch_name
# Include Common Images: $include_common

# AI-Infra Service Images
EOF
        
        for service in "${all_services[@]}"; do
            local image_name="ai-infra-${service}:${tag}"
            local safe_name=$(echo "$image_name" | sed 's|:|_|g')
            local tar_file="${safe_name}.tar"
            if [[ -f "${output_dir}/${arch_name}/${tar_file}" ]]; then
                echo "$image_name|$tar_file" >> "$manifest_file"
            fi
        done
        
        echo "" >> "$manifest_file"
        echo "# Dependency Images" >> "$manifest_file"
        
        for mapping in "${dependencies[@]}"; do
            local source_image="${mapping%%|*}"
            local safe_name=$(echo "$source_image" | sed 's|/|-|g' | sed 's|:|_|g')
            local tar_file="${safe_name}.tar"
            if [[ -f "${output_dir}/${arch_name}/${tar_file}" ]]; then
                echo "$source_image|$tar_file" >> "$manifest_file"
            fi
        done
    
        if [[ "$include_common" == "true" ]]; then
            echo "" >> "$manifest_file"
            echo "# Common/Third-party Images" >> "$manifest_file"
            
            for image in "${COMMON_IMAGES[@]}"; do
                local safe_name=$(echo "$image" | sed 's|/|-|g' | sed 's|:|_|g')
                local tar_file="${safe_name}.tar"
                if [[ -f "${output_dir}/${arch_name}/${tar_file}" ]]; then
                    echo "$image|$tar_file" >> "$manifest_file"
                fi
            done
        fi
        
        log_info "  ✓ Generated manifest for $arch_name"
    done
    
    # Generate import script (architecture-aware)
    log_info "📜 Generating import script..."
    local import_script="${output_dir}/import-images.sh"
    cat > "$import_script" << 'IMPORT_SCRIPT_EOF'
#!/bin/bash

# AI Infrastructure Matrix - Offline Images Import Script (Multi-Arch)
# Usage: ./import-images.sh [architecture]
# Architecture: amd64, arm64, or auto (default: auto-detect)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REQUESTED_ARCH="${1:-auto}"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Detect host architecture
detect_arch() {
    local arch=$(uname -m)
    case "$arch" in
        x86_64|amd64) echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *) echo "amd64" ;;  # Default fallback
    esac
}

# Determine which architecture to use
if [[ "$REQUESTED_ARCH" == "auto" ]]; then
    ARCH=$(detect_arch)
    log_info "Auto-detected architecture: $ARCH"
else
    ARCH="$REQUESTED_ARCH"
fi

IMAGES_DIR="${SCRIPT_DIR}/${ARCH}"
MANIFEST_FILE="${IMAGES_DIR}/images-manifest.txt"

# Check if architecture directory exists
if [[ ! -d "$IMAGES_DIR" ]]; then
    log_error "Architecture directory not found: $IMAGES_DIR"
    log_info "Available architectures:"
    for dir in "$SCRIPT_DIR"/*/; do
        if [[ -f "${dir}images-manifest.txt" ]]; then
            log_info "  - $(basename "$dir")"
        fi
    done
    exit 1
fi

if [[ ! -f "$MANIFEST_FILE" ]]; then
    log_error "Manifest file not found: $MANIFEST_FILE"
    exit 1
fi

log_info "=========================================="
log_info "Importing Offline Images"
log_info "=========================================="
log_info "Architecture: $ARCH"
log_info "Images directory: $IMAGES_DIR"
log_info "Manifest file: $MANIFEST_FILE"
echo

imported_count=0
failed_count=0

while IFS='|' read -r image_name tar_file; do
    # Skip comments and empty lines
    [[ "$image_name" =~ ^[[:space:]]*# ]] && continue
    [[ -z "$image_name" ]] && continue
    
    tar_path="${IMAGES_DIR}/${tar_file}"
    
    if [[ -f "$tar_path" ]]; then
        log_info "→ Importing: $image_name"
        if docker load -i "$tar_path"; then
            log_info "  ✓ Imported successfully: $image_name"
            imported_count=$((imported_count + 1))
        else
            log_error "  ✗ Failed to import: $image_name"
            failed_count=$((failed_count + 1))
        fi
    else
        log_error "  ✗ Tar file not found: $tar_path"
        failed_count=$((failed_count + 1))
    fi
done < "$MANIFEST_FILE"

echo
log_info "=========================================="
log_info "Import completed: $imported_count success, $failed_count failed"

if [[ $failed_count -eq 0 ]]; then
    log_info "🎉 All images imported successfully!"
    echo
    log_info "Next steps:"
    log_info "  1. Check images: docker images | grep -E 'ai-infra|postgres|redis'"
    log_info "  2. Start services: docker compose --profile ha up -d"
else
    log_error "Some images failed to import. Please check the errors above."
fi
IMPORT_SCRIPT_EOF
    
    chmod +x "$import_script"
    
    # Calculate total size for each architecture
    log_info "📊 Calculating sizes..."
    local total_size=$(du -sh "$output_dir" | cut -f1)
    
    # Print summary
    log_info "=========================================="
    log_info "🎉 Offline Export Complete!"
    log_info "=========================================="
    echo
    log_info "📊 Export Statistics:"
    log_info "  • Exported: $exported_count images"
    log_info "  • Failed: $failed_count images"
    log_info "  • Total size: $total_size"
    log_info "  • Architectures:"
    for platform in "${valid_platforms[@]}"; do
        local arch_name="${platform##*/}"
        if [[ -d "${output_dir}/${arch_name}" ]]; then
            local arch_size=$(du -sh "${output_dir}/${arch_name}" | cut -f1)
            local arch_count=$(find "${output_dir}/${arch_name}" -name "*.tar" 2>/dev/null | wc -l | tr -d ' ')
            log_info "    - ${arch_name}: ${arch_count} images (${arch_size})"
        fi
    done
    echo
    
    if [[ ${#failed_images[@]} -gt 0 ]]; then
        log_warn "⚠️  Failed images:"
        for img in "${failed_images[@]}"; do
            log_warn "    - $img"
        done
        echo
        
        # Provide helpful hint
        log_info "💡 To fix missing/wrong-arch images, run:"
        for platform in "${valid_platforms[@]}"; do
            local arch_name="${platform##*/}"
            log_info "   ./build.sh pull-all --platform=$arch_name   # Pull all $arch_name images"
        done
        log_info "   Then re-run: ./build.sh export-offline --platform=<arch>"
        echo
    fi
    
    log_info "📁 Output files:"
    log_info "  • Images directory: $output_dir"
    for platform in "${valid_platforms[@]}"; do
        local arch_name="${platform##*/}"
        log_info "    - ${arch_name}/: Images and manifest for $arch_name"
    done
    log_info "  • Import script: $import_script"
    echo
    log_info "📋 Usage instructions:"
    log_info "  1. Copy the entire '$output_dir' directory to the offline environment"
    log_info "  2. Run import with auto-detection: cd $output_dir && ./import-images.sh"
    log_info "     Or specify architecture: ./import-images.sh amd64"
    log_info "                              ./import-images.sh arm64"
    log_info "  3. Start services: docker compose --profile ha up -d"
    
    return 0
}

print_help() {
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Global Options (can be used with any command):"
    echo "  --force, -f, --no-cache    Force rebuild without Docker cache"
    echo "  --parallel, -p             Enable parallel builds (default: $PARALLEL_JOBS jobs)"
    echo "  --parallel=N, -pN          Enable parallel builds with N concurrent jobs"
    echo "  --no-ssl                   Disable SSL/HTTPS mode (SSL enabled by default)"
    echo "  --ssl=DOMAIN               Enable SSL with specific domain"
    echo "  --skip-cache               Skip build cache check (always rebuild)"
    echo ""
    echo "Environment Commands:"
    echo "  init-env [host]     Initialize/sync .env file (auto-detect EXTERNAL_HOST)"
    echo "  init-env --force    Force re-initialize all environment variables"
    echo "  gen-prod-env [file] Generate production .env with random strong passwords"
    echo "                      default output: .env.prod"
    echo ""
    echo "SSL/HTTPS Commands (SSL enabled by default):"
    echo "  ssl-setup [domain]  Generate self-signed SSL certificates to src/nginx/ssl/"
    echo "                      Certificates are bundled into nginx image during build"
    echo "  ssl-setup --force   Regenerate existing certificates"
    echo "  ssl-setup-le [domain] [email]  Issue Let's Encrypt cert via certbot --standalone"
    echo "                          Uses LETSENCRYPT_EMAIL/LETSENCRYPT_STAGING if omitted"
    echo "  ssl-cloudflare <domain> [--wildcard] [--staging] [--force]"
    echo "                      Issue Let's Encrypt cert via Cloudflare DNS validation"
    echo "                      Credentials: ~/.secrets/cloudflare.ini or CLOUDFLARE_CREDENTIALS"
    echo "                      --wildcard: Include *.<domain> wildcard certificate"
    echo "  ssl-info [domain]   Display SSL certificate information"
    echo "  ssl-check           Diagnose SSL/domain configuration for cloud deployments"
    echo "                      Detects domain mismatch, private IP issues, etc."
    echo "  ssl-clean           Remove all generated SSL certificates and disable SSL"
    echo ""
    echo "Optional Components:"
    echo "  init-safeline       Initialize SafeLine WAF (optional sidecar component)"
    echo "                      SafeLine is NOT included in 'build.sh all'"
    echo "                      After init, start with: docker-compose --profile safeline up -d"
    echo ""
    echo "Build Commands:"
    echo "  build-all, all           Build all components (SSL enabled by default)"
    echo "  build-all --force        Force rebuild all (no cache, re-render templates)"
    echo "  build-all --parallel     Parallel build with smart caching"
    echo "  build-all --no-ssl       Build without SSL/HTTPS"
    echo "  build-all --platform=amd64,arm64   Build for multiple architectures"
    echo "  [component]              Build a specific component (e.g., backend, frontend)"
    echo "  [component] --force      Force rebuild a component without cache"
    echo ""
    echo "Multi-Architecture Build Commands:"
    echo "  --platform=<arch>        Global option to specify target platform(s)"
    echo "                          Can be used with 'build-all' or single component"
    echo "                          Values: amd64, arm64, or amd64,arm64 (both)"
    echo "  build-multiarch [platforms] [--force]  Dedicated multi-arch build command"
    echo "  build-platform <arch> [--force]        Build for single target platform"
    echo ""
    echo "Build Cache Commands:"
    echo "  cache-status        Show build cache status for all services"
    echo "  build-history [N]   Show last N build history entries (default: 20)"
    echo "  clear-cache [svc]   Clear build cache (all or specific service)"
    echo ""
    echo "Template Commands:"
    echo "  render, sync        Render all Dockerfile.tpl templates from .env config"
    echo "  render --force      Force re-render all templates (ignore cache)"
    echo ""
    echo "Service Commands:"
    echo "  start-all           Start all services (with SaltStack HA multi-master)"
    echo "  stop-all            Stop all services"
    echo "  tag-images          Tag private registry images as local (for intranet)"
    echo ""
    echo "Database Safety Commands:"
    echo "  db-check            Check if PostgreSQL has production data"
    echo "  db-backup [name]    Backup PostgreSQL database"
    echo "  db-restore <file>   Restore PostgreSQL database from backup"
    echo ""
    echo "  Environment Variables for DB Init:"
    echo "    DB_INIT_MODE=safe_init   (default) Skip reset if data exists"
    echo "    DB_INIT_MODE=upgrade     Keep data, only run migrations"
    echo "    DB_INIT_MODE=force_reset Backup and reset (for dev/test)"
    echo "    SKIP_DB_CHECK=true       Skip database safety check"
    echo ""
    echo "Pull Commands (Smart Mode):"
    echo "  prefetch            Prefetch all base images from Dockerfiles"
    echo "  pull-common         Pull common/third-party images (mysql, kafka, redis, etc.)"
    echo "  pull-all                              Internet mode: pull from Docker Hub"
    echo "  pull-all <registry/project> [tag]    Intranet mode: pull from private registry"
    echo "  deps-pull <registry/project> [tag]   Pull dependency images from registry"
    echo ""
    echo "Push Commands:"
    echo "  push <service> <registry/project> [tag]  Push single service to registry"
    echo "  push-all <registry/project> [tag]        Push all images (4 phases)"
    echo "  push-dep <registry/project> [tag]        Push dependency images to registry"
    echo ""
    echo "  ⚠️  Harbor registries require project name in path:"
    echo "     ✓ harbor.example.com/ai-infra     (correct)"
    echo "     ✗ harbor.example.com              (wrong - missing project)"
    echo ""
    echo "Clean Commands:"
    echo "  clean-images [tag]  Remove ai-infra Docker images (optional: specific tag)"
    echo "  clean-volumes       Remove ai-infra Docker volumes"
    echo "  clean-all [--force] Remove all images, volumes and stop containers"
    echo ""
    echo "Download Commands:"
    echo "  download [options] [component...]   Download third-party dependencies"
    echo "  download-deps                       Alias for 'download'"
    echo "    Options:"
    echo "      -l, --list          List available components"
    echo "      -v, --version VER   Specify version"
    echo "      -a, --arch ARCH     Target architecture (amd64, arm64, all)"
    echo "      --no-mirror         Disable GitHub mirror acceleration"
    echo "    Components: prometheus, node_exporter, alertmanager, categraf,"
    echo "                code_server (alias: vscode), saltstack, slurm, etc."
    echo ""
    echo "Go Dependencies Update Commands:"
    echo "  update-go-deps [project] [options]  Update Go module dependencies"
    echo "    Projects:"
    echo "      backend          Update backend Go dependencies"
    echo "      nightingale      Update Nightingale Go dependencies"
    echo "      all              Update all projects (default)"
    echo "    Options:"
    echo "      -h, --help       Show help"
    echo "      --patch          Only update patch versions (default)"
    echo "      --minor          Update minor and patch versions"
    echo "      --tidy           Only run go mod tidy (no upgrades)"
    echo "    Examples:"
    echo "      $0 update-go-deps              # Update all with patch versions"
    echo "      $0 update-go-deps backend      # Update backend only"
    echo "      $0 update-go-deps --minor      # Update all with minor versions"
    echo "      $0 update-go-deps --tidy       # Only tidy all projects"
    echo ""
    echo "Offline Export Commands:"
    echo "  export-offline [dir] [tag] [include_common] [platforms]"
    echo "                          Export images to tar files (multi-arch support)"
    echo "                          dir: output directory (default: ./offline-images)"
    echo "                          tag: image tag (default: latest)"
    echo "                          include_common: include third-party images (default: true)"
    echo "                          platforms: amd64,arm64 or single arch (default: amd64,arm64)"
    echo ""
    echo "Template Variables (from .env):"
    echo "  === Mirror Configuration (Build-time) ==="
    echo "  GITHUB_MIRROR       GitHub mirror URL prefix (e.g., https://ghfast.top/)"
    echo "  APT_MIRROR          APT mirror for Ubuntu/Debian (e.g., mirrors.aliyun.com)"
    echo "  YUM_MIRROR          YUM mirror for AlmaLinux/CentOS"
    echo "  ALPINE_MIRROR       Alpine mirror"
    echo "  GO_PROXY            Go module proxy"
    echo "  PYPI_INDEX_URL      PyPI mirror"
    echo "  NPM_REGISTRY        npm registry mirror"
    echo ""
    echo "  === Base Image Versions ==="
    echo "  UBUNTU_VERSION      Ubuntu base image version"
    echo "  ALMALINUX_VERSION   AlmaLinux version"
    echo "  GOLANG_VERSION      Go version"
    echo ""
    echo "  === Component Versions ==="
    echo "  SLURM_VERSION       SLURM version to build"
    echo "  SALTSTACK_VERSION   SaltStack version"
    echo "  CATEGRAF_VERSION    Categraf version"
    echo "  GITEA_VERSION       Gitea version"
    echo ""
    echo "Examples:"
    echo "  # Environment setup"
    echo "  $0 init-env                        # Auto-detect and initialize .env"
    echo "  $0 init-env 192.168.0.100          # Set specific EXTERNAL_HOST"
    echo "  $0 init-env --force                # Force re-initialize"
    echo ""
    echo "  # SSL/HTTPS setup (certificates bundled into nginx image)"
    echo "  $0 ssl-setup                       # Generate certs for auto-detected domain"
    echo "  $0 ssl-setup example.com           # Generate certs for specific domain"
    echo "  $0 ssl-setup --force               # Regenerate existing certificates"
    echo "  $0 ssl-setup-le example.com user@example.com   # Request Let's Encrypt cert"
    echo "  $0 ssl-cloudflare ai-infra-matrix.top          # Cloudflare DNS validation"
    echo "  $0 ssl-cloudflare ai-infra-matrix.top --wildcard  # Wildcard cert (*.<domain>)"
    echo "  $0 ssl-info                        # Show certificate details"
    echo "  $0 ssl-check                       # Diagnose SSL/domain config issues"
    echo "  $0 nginx                           # Rebuild nginx with SSL certs bundled"
    echo ""
    echo "  # Template rendering"
    echo "  $0 render                          # Render templates from .env"
    echo "  $0 render --force                  # Force re-render all templates"
    echo ""
    echo "  # Building (with smart caching, SSL enabled by default)"
    echo "  $0 build-all                       # Build all services with SSL"
    echo "  $0 build-all --parallel            # Parallel build (default 4 jobs)"
    echo "  $0 build-all --parallel=8          # Parallel build with 8 jobs"
    echo "  $0 build-all --force               # Force rebuild all (ignore cache)"
    echo "  $0 build-all --no-ssl              # Build without HTTPS"
    echo "  $0 backend                         # Build single service"
    echo "  $0 backend --force                 # Force rebuild single service"
    echo ""
    echo "  # Multi-architecture builds (recommended for offline deployment)"
    echo "  $0 build-all --platform=amd64,arm64    # Build all services for both archs"
    echo "  $0 build-all --platform=amd64          # Build all for amd64 only"
    echo "  $0 build-all --platform=arm64 --force  # Force rebuild for arm64"
    echo "  $0 build-multiarch                     # Alternative: dedicated command"
    echo "  $0 build-platform amd64                # Build single platform"
    echo ""
    echo "  # Build cache management"
    echo "  $0 cache-status                    # Show which services need rebuild"
    echo "  $0 build-history                   # Show recent build history"
    echo "  $0 clear-cache                     # Clear all build cache"
    echo "  $0 clear-cache backend             # Clear cache for specific service"
    echo ""
    echo "  # Internet mode (Docker Hub)"
    echo "  $0 prefetch                        # Prefetch base images"
    echo "  $0 pull-all                        # Pull common images from Docker Hub"
    echo ""
    echo "  # Intranet mode (Private Registry)"
    echo "  $0 push-all harbor.example.com/ai-infra v0.3.8    # Push to registry"
    echo "  $0 pull-all harbor.example.com/ai-infra v0.3.8    # Pull from registry"
    echo ""
    echo "  # Offline export (multi-arch)"
    echo "  $0 export-offline ./offline v0.3.8 true amd64,arm64  # Export both architectures"
    echo "  $0 export-offline ./offline v0.3.8 true amd64        # Export AMD64 only"
    echo "  $0 export-offline ./offline v0.3.8 true arm64        # Export ARM64 only"
    echo "  $0 export-offline ./images v0.3.8 false              # Export without common images"
    echo ""
    echo "  # Cleanup"
    echo "  $0 clean-all --force"
    echo ""
    echo "  # Download third-party dependencies (for faster AppHub builds)"
    echo "  $0 download                            # Download all to third_party/"
    echo "  $0 download --list                     # List available components"
    echo "  $0 download vscode                     # Download VS Code Server only"
    echo "  $0 download -v 4.107.0 vscode          # Download specific version"
    echo "  $0 download --arch amd64 prometheus    # Download amd64 architecture only"
}

# ==============================================================================
# 4. Main Execution
# ==============================================================================

if [ $# -eq 0 ]; then
    print_help
    exit 0
fi

# Parse global options first (--force, --no-cache, -f, --parallel, --ssl, --platform, etc.)
# These can appear anywhere in the command line
FORCE_BUILD=false
FORCE_RENDER=false
FORCE_REBUILD=false
ENABLE_PARALLEL=false
ENABLE_SSL=false
SKIP_CACHE_CHECK=false
BUILD_PLATFORMS="amd64,arm64"  # Default: build both architectures. Can be: amd64, arm64, amd64,arm64
NO_ARCH_TAG=true    # Default: remote tag won't include arch suffix (Docker Hub mode). Use --arch-tag for Harbor mode
REMAINING_ARGS=()

for arg in "$@"; do
    case "$arg" in
        --force|-f|--no-cache)
            FORCE_BUILD=true
            FORCE_RENDER=true
            FORCE_REBUILD=true
            ;;
        --parallel|-p)
            ENABLE_PARALLEL=true
            ;;
        --parallel=*|-p=*)
            ENABLE_PARALLEL=true
            PARALLEL_JOBS="${arg#*=}"
            ;;
        -p[0-9]*)
            ENABLE_PARALLEL=true
            PARALLEL_JOBS="${arg#-p}"
            ;;
        --platform=*)
            BUILD_PLATFORMS="${arg#*=}"
            ;;
        --platform)
            # Next arg should be the platform value, handled by shift logic below
            ;;
        --arch-tag)
            # For Harbor: add architecture suffix to remote tag (e.g., v0.3.8-amd64)
            NO_ARCH_TAG=false
            ;;
        --ssl)
            ENABLE_SSL=true
            ;;
        --ssl=*)
            ENABLE_SSL=true
            SSL_DOMAIN="${arg#*=}"
            ;;
        --no-ssl)
            ENABLE_SSL=false
            ;;
        --skip-cache)
            SKIP_CACHE_CHECK=true
            ;;
        *)
            REMAINING_ARGS+=("$arg")
            ;;
    esac
done

# Show mode messages after parsing
if [[ "$FORCE_BUILD" == "true" ]]; then
    log_info "🔧 Force mode enabled (--no-cache for Docker builds)"
fi
if [[ "$ENABLE_PARALLEL" == "true" ]]; then
    log_parallel "🚀 Parallel mode enabled (max $PARALLEL_JOBS concurrent jobs)"
fi
if [[ "$ENABLE_SSL" == "true" ]]; then
    log_info "🔒 SSL/HTTPS mode enabled"
fi
if [[ "$SKIP_CACHE_CHECK" == "true" ]]; then
    log_cache "⏭️  Cache check skipped (--skip-cache)"
fi
if [[ -n "$BUILD_PLATFORMS" ]]; then
    log_info "🏗️  Multi-platform build enabled: $BUILD_PLATFORMS"
    # Update DOCKER_HOST_PLATFORM to use the first specified platform for pull operations
    # This ensures docker pull fetches images for the target architecture, not native
    first_platform="${BUILD_PLATFORMS%%,*}"  # Get first platform (before comma)
    case "$first_platform" in
        amd64|x86_64)
            DOCKER_HOST_PLATFORM="linux/amd64"
            ;;
        arm64|aarch64)
            DOCKER_HOST_PLATFORM="linux/arm64"
            ;;
    esac
    log_info "🎯 Target platform for pull: $DOCKER_HOST_PLATFORM"
fi
if [[ "$NO_ARCH_TAG" == "true" ]]; then
    log_info "☁️  Docker Hub mode: remote tags without arch suffix"
fi

COMMAND="${REMAINING_ARGS[0]:-}"
ARG2="${REMAINING_ARGS[1]:-}"
ARG3="${REMAINING_ARGS[2]:-}"
ARG4="${REMAINING_ARGS[3]:-}"

case "$COMMAND" in
    init-env)
        # 初始化或同步 .env 文件
        if [[ "$FORCE_BUILD" == "true" ]]; then
            log_info "Force re-initializing .env..."
            init_env_file "true"
        elif [[ -n "$ARG2" ]]; then
            # 使用指定的 EXTERNAL_HOST
            log_info "Setting EXTERNAL_HOST=$ARG2..."
            update_env_variable "EXTERNAL_HOST" "$ARG2"
            update_env_variable "DOMAIN" "$ARG2"
            log_info "✓ EXTERNAL_HOST updated to $ARG2"
        else
            init_env_file "true"
        fi
        # 显示当前配置
        echo
        log_info "Current environment configuration:"
        grep -E "^(EXTERNAL_HOST|DOMAIN|EXTERNAL_PORT|EXTERNAL_SCHEME)=" "$ENV_FILE"
        ;;
    init-safeline)
        # 初始化 SafeLine WAF 数据目录
        log_info "📦 Initializing SafeLine WAF data directories..."
        
        # 从 .env 获取 SAFELINE_DIR，默认 ./data/safeline
        safeline_dir="${SAFELINE_DIR:-./data/safeline}"
        
        # 创建必要的目录
        mkdir -p "$safeline_dir"/{resources,logs,run}
        mkdir -p "$safeline_dir"/resources/{postgres/data,mgt,sock,nginx,detector,chaos,cache,luigi}
        mkdir -p "$safeline_dir"/logs/{nginx,detector}
        
        # 设置安全的目录权限
        # - 数据目录: 755 (所有者读写执行，其他人只读执行)
        # - run/sock 目录需要被容器内进程访问: 750
        # - postgres 数据目录: 700 (仅所有者访问，数据库安全要求)
        # - logs 目录: 755 (允许读取日志)
        
        # 设置基础目录权限
        chmod 755 "$safeline_dir"
        chmod 755 "$safeline_dir"/resources
        chmod 755 "$safeline_dir"/logs
        chmod 750 "$safeline_dir"/run
        
        # 资源子目录权限
        chmod 700 "$safeline_dir"/resources/postgres      # PostgreSQL 数据需要严格权限
        chmod 700 "$safeline_dir"/resources/postgres/data
        chmod 755 "$safeline_dir"/resources/mgt
        chmod 750 "$safeline_dir"/resources/sock          # Socket 目录
        chmod 755 "$safeline_dir"/resources/nginx
        chmod 755 "$safeline_dir"/resources/detector
        chmod 755 "$safeline_dir"/resources/chaos
        chmod 755 "$safeline_dir"/resources/cache
        chmod 755 "$safeline_dir"/resources/luigi
        
        # 日志目录权限
        chmod 755 "$safeline_dir"/logs/nginx
        chmod 755 "$safeline_dir"/logs/detector
        
        log_info "✅ SafeLine directories created at: $safeline_dir"
        log_info ""
        log_info "Directory structure (with permissions):"
        find "$safeline_dir" -type d -exec ls -ld {} \; 2>/dev/null | head -20 | sed 's/^/  /'
        log_info ""
        log_info "⚠️  Note: SafeLine containers run as root inside, so directory ownership"
        log_info "   is managed by the containers themselves. If you encounter permission"
        log_info "   issues, run: sudo chown -R \$(id -u):\$(id -g) $safeline_dir"
        log_info ""
        log_info "💡 SafeLine is an optional sidecar component (not included in 'build.sh all')"
        log_info ""
        log_info "📋 To start SafeLine services (using docker-compose profiles):"
        log_info "   docker-compose --profile safeline up -d"
        log_info ""
        log_info "🔐 To get/reset admin password:"
        log_info "   docker exec safeline-mgt /app/mgt-cli reset-admin"
        log_info ""
        log_info "🌐 Access SafeLine management console at: https://<host>:${SAFELINE_MGT_PORT:-9443}"
        ;;
    ssl-setup|ssl-init|ssl|setup-ssl)
        # 设置 SSL 证书
        ssl_domain="${ARG2:-}"
        setup_ssl_certificates "$ssl_domain" "$FORCE_BUILD"
        ;;
    ssl-setup-le|ssl-letsencrypt|ssl-le)
        # 使用 Let's Encrypt 申请正式证书
        ssl_domain="${ARG2:-}"
        ssl_email="${ARG3:-${LETSENCRYPT_EMAIL:-}}"
        ssl_staging="${ARG4:-${LETSENCRYPT_STAGING:-false}}"
        setup_letsencrypt_certificates "$ssl_domain" "$ssl_email" "$ssl_staging" "$FORCE_BUILD"
        ;;
    ssl-cloudflare|ssl-cf|ssl-dns)
        # 使用 Cloudflare DNS 验证申请 Let's Encrypt 证书
        ssl_domain="${ARG2:-}"
        ssl_wildcard="false"
        ssl_staging="${LETSENCRYPT_STAGING:-false}"
        # 解析参数
        shift 2 2>/dev/null || true
        for arg in "$@"; do
            case "$arg" in
                --wildcard) ssl_wildcard="true" ;;
                --staging) ssl_staging="true" ;;
                --force) FORCE_BUILD="true" ;;
            esac
        done
        setup_cloudflare_certificates "$ssl_domain" "$ssl_wildcard" "$ssl_staging" "$FORCE_BUILD"
        ;;
    ssl-info)
        # 显示 SSL 证书信息
        show_ssl_info "${ARG2:-}"
        ;;
    ssl-check|ssl-diagnose|ssl-domain)
        # 检查 SSL/域名配置是否正确（云部署诊断）
        show_ssl_domain_recommendations "${ARG2:-$EXTERNAL_HOST}" "${ARG3:-$SSL_DOMAIN}"
        ;;
    ssl-clean)
        # 清理 SSL 证书
        clean_ssl_certificates
        ;;
    enable-ssl)
        # 启用 SSL 模式（更新 .env 配置）
        log_info "🔒 Enabling SSL mode..."
        update_env_variable "ENABLE_TLS" "true"
        update_env_variable "EXTERNAL_SCHEME" "https"
        log_info "✓ SSL mode enabled"
        log_info "  ENABLE_TLS=true"
        log_info "  EXTERNAL_SCHEME=https"
        log_info ""
        log_info "📋 Next steps:"
        log_info "   1. Generate certificates: ./build.sh ssl-setup"
        log_info "   2. Rebuild nginx:         ./build.sh nginx"
        log_info "   3. Restart services:      docker compose restart nginx"
        ;;
    disable-ssl)
        # 禁用 SSL 模式（更新 .env 配置）
        log_info "🔓 Disabling SSL mode..."
        update_env_variable "ENABLE_TLS" "false"
        update_env_variable "EXTERNAL_SCHEME" "http"
        log_info "✓ SSL mode disabled"
        log_info "  ENABLE_TLS=false"
        log_info "  EXTERNAL_SCHEME=http"
        log_info ""
        log_info "📋 Next steps:"
        log_info "   1. Rebuild nginx: ./build.sh nginx"
        log_info "   2. Restart:       docker compose restart nginx"
        ;;
    gen-prod-env)
        # 生成生产环境配置文件（使用强随机密码）
        generate_production_env "${ARG2:-.env.prod}" "$FORCE_BUILD"
        ;;
    build-all|all)
        # 如果启用了 SSL，先设置 SSL 证书
        if [[ "$ENABLE_SSL" == "true" ]]; then
            log_info "🔒 SSL mode enabled, setting up certificates first..."
            setup_ssl_certificates "$SSL_DOMAIN" "$FORCE_BUILD"
        fi
        
        # Determine build platforms
        # - If --platform=xxx is specified, use that (can be single or multiple)
        # - If not specified, auto-detect native platform for local build
        _target_platforms=""
        if [[ -n "$BUILD_PLATFORMS" ]]; then
            # User explicitly specified platform(s)
            _target_platforms="$BUILD_PLATFORMS"
            log_info "🏗️  Build platforms (user specified): $_target_platforms"
        else
            # Auto-detect native platform
            _native_platform=$(_detect_docker_platform)
            _native_arch="${_native_platform##*/}"
            _target_platforms="$_native_arch"
            log_info "🏗️  Build platform (auto-detected native): $_target_platforms"
        fi
        
        # Always use multiplatform build function for consistent behavior
        # This ensures buildx is used for all builds (better architecture handling)
        if [[ "$FORCE_BUILD" == "true" ]]; then
            build_all_multiplatform "$_target_platforms" "true"
        else
            build_all_multiplatform "$_target_platforms"
        fi
        ;;
    build-multiarch|multiarch)
        # Build for multiple architectures using buildx
        build_multiarch "$ARG2" "$ARG3"
        ;;
    build-platform)
        # Build for a specific platform (e.g., amd64 on arm64 mac)
        build_for_platform "$ARG2" "$ARG3"
        ;;
    cache-status)
        # Show build cache status
        show_cache_status
        ;;
    build-history)
        # Show build history
        show_build_history "$ARG2" "${ARG3:-20}"
        ;;
    clear-cache)
        # Clear build cache
        clear_build_cache "$ARG2"
        ;;
    render|sync|sync-templates)
        if [[ "$FORCE_BUILD" == "true" ]] || [[ "$FORCE_RENDER" == "true" ]]; then
            render_all_templates "true"
        else
            render_all_templates
        fi
        ;;
    sync-env)
        # 同步 .env 与 .env.example，并检测配置差异
        sync_env_with_example
        ;;
    check-env)
        # 仅检测 .env 配置差异，不同步
        check_env_config_drift
        ;;
    start-all)
        start_all
        ;;
    tag-images)
        tag_private_images_as_local
        ;;
    stop-all)
        stop_all
        ;;
    db-check)
        check_postgres_has_data "${ARG2:-ai_infra}"
        ;;
    db-backup)
        backup_postgres_database "${ARG2:-ai_infra}" "./backup/postgres"
        ;;
    db-restore)
        if [[ -z "$ARG2" ]]; then
            log_error "Backup file required"
            log_info "Usage: $0 db-restore <backup_file.sql.gz> [database_name]"
            exit 1
        fi
        restore_postgres_database "$ARG2" "${ARG3:-ai_infra}"
        ;;
    clean-images)
        clean_images "$ARG2" "${ARG3:-false}"
        ;;
    clean-volumes)
        clean_volumes "${ARG2:-false}"
        ;;
    clean-all)
        clean_all "$ARG2"
        ;;
    prefetch)
        prefetch_base_images "$ARG2"
        ;;
    pull-common)
        pull_common_images
        ;;
    pull-all)
        # Smart mode: without registry -> Docker Hub, with registry -> private registry
        # Supports multi-architecture: ./build.sh pull-all <registry> <tag> --platform=amd64,arm64
        pull_all_services "$ARG2" "${ARG3:-${IMAGE_TAG:-latest}}" "$DEFAULT_MAX_RETRIES" "${BUILD_PLATFORMS:-}"
        ;;
    deps-pull)
        if [[ -z "$ARG2" ]]; then
            log_error "Registry is required"
            log_info "Usage: $0 deps-pull <registry> [tag]"
            exit 1
        fi
        pull_and_tag_dependencies "$ARG2" "${ARG3:-${IMAGE_TAG:-latest}}"
        ;;
    push)
        if [[ -z "$ARG2" ]]; then
            log_error "Service name is required"
            log_info "Usage: $0 push <service> <registry> [tag] [--platform=amd64|arm64]"
            exit 1
        fi
        if [[ -z "$ARG3" ]]; then
            log_error "Registry is required"
            log_info "Usage: $0 push <service> <registry> [tag] [--platform=amd64|arm64]"
            exit 1
        fi
        # Pass BUILD_PLATFORMS if specified via --platform flag
        # Pass BUILD_PLATFORMS and arch_tag flag
        # Default: no arch suffix. Use --arch-tag for Harbor mode
        _arch_tag="false"
        [[ "$NO_ARCH_TAG" == "false" ]] && _arch_tag="true"
        push_service "$ARG2" "${ARG4:-${IMAGE_TAG:-latest}}" "$ARG3" "$DEFAULT_MAX_RETRIES" "${BUILD_PLATFORMS:-}" "$_arch_tag"
        ;;
    push-all)
        if [[ -z "$ARG2" ]]; then
            log_error "Registry is required"
            log_info "Usage: $0 push-all <registry> [tag] [--platform=amd64,arm64] [--arch-tag]"
            log_info ""
            log_info "Examples:"
            log_info "  $0 push-all docker.io/user v0.3.8                                           # Docker Hub: v0.3.8 (default)"
            log_info "  $0 push-all docker.io/user v0.3.8 --platform=amd64                          # Single arch, no suffix"
            log_info "  $0 push-all harbor.example.com/ai-infra v0.3.8 --arch-tag                   # Harbor: v0.3.8-amd64, v0.3.8-arm64"
            log_info "  $0 push-all harbor.example.com/ai-infra v0.3.8 --platform=amd64 --arch-tag  # Harbor: v0.3.8-amd64"
            exit 1
        fi
        # Default: no arch suffix (Docker Hub mode). Use --arch-tag for Harbor mode
        _arch_tag="false"
        [[ "$NO_ARCH_TAG" == "false" ]] && _arch_tag="true"
        # Pass BUILD_PLATFORMS and arch_tag flag
        push_all_services "$ARG2" "${ARG3:-${IMAGE_TAG:-latest}}" "$DEFAULT_MAX_RETRIES" "${BUILD_PLATFORMS:-}" "$_arch_tag"
        ;;
    push-dep|push-dependencies)
        if [[ -z "$ARG2" ]]; then
            log_error "Registry is required"
            log_info "Usage: $0 push-dep <registry> [tag]"
            exit 1
        fi
        push_all_dependencies "$ARG2" "${ARG3:-${IMAGE_TAG:-latest}}"
        ;;
    export-offline)
        # Export all images to tar files for offline deployment (multi-arch)
        # Priority: --platform= flag > ARG5 position parameter > default (amd64,arm64)
        _export_platforms="${BUILD_PLATFORMS:-${ARG5:-amd64,arm64}}"
        export_offline_images "$ARG2" "${ARG3:-${IMAGE_TAG:-latest}}" "${ARG4:-true}" "$_export_platforms"
        ;;
    download|download-deps)
        # Download third-party dependencies to third_party/
        log_info "📦 Downloading third-party dependencies..."
        
        # 加载 .env 文件中的环境变量，供下载函数使用
        if [[ -f "$ENV_FILE" ]]; then
            log_info "Loading environment from $ENV_FILE..."
            set -a  # 自动导出所有变量
            source "$ENV_FILE"
            set +a
        fi
        
        # 构建传递给下载函数的参数（排除命令本身）
        _download_args=("${REMAINING_ARGS[@]:1}")
        
        # 使用内置的下载函数
        download_third_party "${_download_args[@]}"
        
        log_info "✅ Third-party dependencies downloaded to third_party/"
        log_info "💡 These files will be used during AppHub build for faster builds"
        ;;
    update-go-deps)
        # Update Go module dependencies
        log_info "🔄 Updating Go dependencies..."
        
        # 构建传递给更新函数的参数（排除命令本身）
        _update_args=("${REMAINING_ARGS[@]:1}")
        
        # 使用内置的更新函数
        update_go_dependencies "${_update_args[@]}"
        ;;
    help|--help|-h)
        print_help
        ;;
    "")
        print_help
        ;;
    *)
        # Single component build - collect all non-option arguments as components
        components=()
        for arg in "${REMAINING_ARGS[@]}"; do
            # Skip if it's an option (starts with -)
            [[ "$arg" == -* ]] && continue
            components+=("$arg")
        done
        
        if [[ ${#components[@]} -eq 0 ]]; then
            log_error "No component specified"
            print_help
            exit 1
        fi
        
        log_info "Building components: ${components[*]}"
        [[ "$FORCE_BUILD" == "true" ]] && log_info "  with --no-cache (force rebuild)"
        
        # Check if multi-platform build is requested
        if [[ -n "$BUILD_PLATFORMS" ]]; then
            log_info "🏗️  Multi-platform build mode: $BUILD_PLATFORMS"
            # Parse comma-separated platforms
            IFS=',' read -ra platform_list <<< "$BUILD_PLATFORMS"
            for platform in "${platform_list[@]}"; do
                platform=$(echo "$platform" | xargs) # Trim whitespace
                log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
                log_info "🏗️  Building for platform: $platform"
                log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
                for component in "${components[@]}"; do
                    build_component_for_platform "$component" "$platform"
                done
            done
        else
            # Standard single-platform build
            for component in "${components[@]}"; do
                build_component "$component"
            done
        fi
        ;;
esac
