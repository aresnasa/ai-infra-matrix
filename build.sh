#!/bin/bash

# AI Infrastructure Matrix - 精简构建脚本
# 版本: v1.0.0
# 专注于 src/ 目录下的 Dockerfile 构建

set -e

# 操作系统检测
detect_os() {
    if [[ "$OSTYPE" == "darwin"* ]]; then
        echo "macOS"
    elif [[ "$OSTYPE" == "linux-gnu"* ]] || [[ "$OSTYPE" == "linux"* ]]; then
        echo "Linux"
    elif [[ "$OSTYPE" == "msys"* ]] || [[ "$OSTYPE" == "cygwin"* ]]; then
        echo "Windows"
    else
        # 备用检测方法
        if [[ "$(uname -s)" == "Darwin" ]]; then
            echo "macOS"
        elif [[ "$(uname -s)" == "Linux" ]]; then
            echo "Linux"
        else
            echo "Other"
        fi
    fi
}

# 全局变量
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VERSION="   1.0.0"
CONFIG_FILE="$SCRIPT_DIR/config.toml"
OS_TYPE=$(detect_os)
FORCE_REBUILD=false  # 强制重新构建标志

# 构建缓存相关变量
BUILD_CACHE_DIR="$SCRIPT_DIR/.build-cache"
BUILD_ID_FILE="$BUILD_CACHE_DIR/build-id.txt"
BUILD_HISTORY_FILE="$BUILD_CACHE_DIR/build-history.log"
SKIP_CACHE_CHECK=false  # 跳过缓存检查标志

# 基本输出函数（早期定义，供其他函数使用）
print_error() {
    echo -e "\033[31m[ERROR]\033[0m $1"
}

print_info() {
    echo -e "\033[32m[INFO]\033[0m $1"
}

# 跨平台兼容函数
# sed命令跨平台兼容包装器
sed_inplace() {
    if [[ "$OS_TYPE" == "macOS" ]]; then
        sed -i '.bak' "$@"
    else
        sed -i "$@"
    fi
}

# 清理备份文件
cleanup_backup_files() {
    local dir="${1:-.}"
    if [[ "$OS_TYPE" == "macOS" ]]; then
        find "$dir" -name "*.bak" -type f -delete 2>/dev/null || true
    fi
}

# 在指定环境文件中创建或更新一个变量（默认 .env）
# 用法: set_or_update_env_var VAR VALUE [FILE]
set_or_update_env_var() {
    local var_name="$1"
    local var_value="$2"
    local env_file="${3:-$SCRIPT_DIR/.env}"

    if [[ -z "$var_name" ]]; then
        return 1
    fi

    # 确保环境文件存在
    if [[ ! -f "$env_file" ]]; then
        touch "$env_file"
    fi

    # 更新或追加变量
    if grep -q "^${var_name}=" "$env_file" 2>/dev/null; then
        sed_inplace "s|^${var_name}=.*|${var_name}=${var_value}|g" "$env_file"
    else
        echo "${var_name}=${var_value}" >> "$env_file"
    fi

    cleanup_backup_files "$(dirname "$env_file")"
}

# 设置SaltStack默认配置
setup_saltstack_defaults() {
    local env_file="$1"
    
    if [[ -z "$env_file" ]] || [[ ! -f "$env_file" ]]; then
        print_error "环境文件不存在: $env_file"
        return 1
    fi
    
    print_info "设置SaltStack默认配置..."
    
    # SaltStack Master 主机配置
    if ! grep -q "^SALTSTACK_MASTER_HOST=" "$env_file" 2>/dev/null; then
        set_or_update_env_var "SALTSTACK_MASTER_HOST" "saltstack" "$env_file"
        print_info "✓ 设置默认值: SALTSTACK_MASTER_HOST=saltstack"
    fi
    
    # SaltStack API 端口配置（需要在 MASTER_URL 之前设置）
    if ! grep -q "^SALT_API_PORT=" "$env_file" 2>/dev/null; then
        set_or_update_env_var "SALT_API_PORT" "8002" "$env_file"
        print_info "✓ 设置默认值: SALT_API_PORT=8002"
    fi
    
    # SaltStack API 协议配置
    if ! grep -q "^SALT_API_SCHEME=" "$env_file" 2>/dev/null; then
        set_or_update_env_var "SALT_API_SCHEME" "http" "$env_file"
        print_info "✓ 设置默认值: SALT_API_SCHEME=http"
    fi
    
    # SaltStack Master 主机配置 (兼容旧版本)
    if ! grep -q "^SALT_MASTER_HOST=" "$env_file" 2>/dev/null; then
        set_or_update_env_var "SALT_MASTER_HOST" "saltstack" "$env_file"
        print_info "✓ 设置默认值: SALT_MASTER_HOST=saltstack"
    fi
    
    # SaltStack Master API URL（自动组合生成完整URL）
    if ! grep -q "^SALTSTACK_MASTER_URL=" "$env_file" 2>/dev/null; then
        # 读取已设置的值
        local salt_scheme=$(grep "^SALT_API_SCHEME=" "$env_file" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "http")
        local salt_host=$(grep "^SALT_MASTER_HOST=" "$env_file" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "saltstack")
        local salt_port=$(grep "^SALT_API_PORT=" "$env_file" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "8002")
        local default_url="${salt_scheme}://${salt_host}:${salt_port}"
        set_or_update_env_var "SALTSTACK_MASTER_URL" "$default_url" "$env_file"
        print_info "✓ 设置默认值: SALTSTACK_MASTER_URL=$default_url"
    else
        # 如果存在但为空，则自动填充
        local current_url=$(grep "^SALTSTACK_MASTER_URL=" "$env_file" | cut -d'=' -f2 | tr -d '"' | tr -d "'")
        if [[ -z "$current_url" ]]; then
            local salt_scheme=$(grep "^SALT_API_SCHEME=" "$env_file" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "http")
            local salt_host=$(grep "^SALT_MASTER_HOST=" "$env_file" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "saltstack")
            local salt_port=$(grep "^SALT_API_PORT=" "$env_file" 2>/dev/null | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "8002")
            local default_url="${salt_scheme}://${salt_host}:${salt_port}"
            set_or_update_env_var "SALTSTACK_MASTER_URL" "$default_url" "$env_file"
            print_info "✓ 自动填充空值: SALTSTACK_MASTER_URL=$default_url"
        fi
    fi
    
    # SaltStack API Token (可选)
    if ! grep -q "^SALTSTACK_API_TOKEN=" "$env_file" 2>/dev/null; then
        set_or_update_env_var "SALTSTACK_API_TOKEN" "" "$env_file"
        print_info "✓ 设置默认值: SALTSTACK_API_TOKEN=(空，可选配置)"
    fi
    
    # SaltStack API 认证配置
    if ! grep -q "^SALT_API_USERNAME=" "$env_file" 2>/dev/null; then
        set_or_update_env_var "SALT_API_USERNAME" "saltapi" "$env_file"
        print_info "✓ 设置默认值: SALT_API_USERNAME=saltapi"
    fi
    
    if ! grep -q "^SALT_API_PASSWORD=" "$env_file" 2>/dev/null; then
        set_or_update_env_var "SALT_API_PASSWORD" "" "$env_file"
        print_info "✓ 设置默认值: SALT_API_PASSWORD=(空)"
    fi
    
    if ! grep -q "^SALT_API_EAUTH=" "$env_file" 2>/dev/null; then
        set_or_update_env_var "SALT_API_EAUTH" "file" "$env_file"
        print_info "✓ 设置默认值: SALT_API_EAUTH=file"
    fi
    
    print_success "✓ SaltStack默认配置设置完成"
}

# 设置其他服务的默认配置
setup_services_defaults() {
    local env_file="$1"
    
    if [[ -z "$env_file" ]] || [[ ! -f "$env_file" ]]; then
        print_error "环境文件不存在: $env_file"
        return 1
    fi
    
    print_info "设置服务默认配置..."
    
    # LDAP 配置
    if ! grep -q "^LDAP_ORGANISATION=" "$env_file" 2>/dev/null; then
        set_or_update_env_var "LDAP_ORGANISATION" "AI Infrastructure" "$env_file"
    fi
    
    if ! grep -q "^LDAP_DOMAIN=" "$env_file" 2>/dev/null; then
        set_or_update_env_var "LDAP_DOMAIN" "ai-infra.com" "$env_file"
    fi
    
    # phpLDAPadmin 配置
    if ! grep -q "^PHPLDAPADMIN_HTTPS=" "$env_file" 2>/dev/null; then
        set_or_update_env_var "PHPLDAPADMIN_HTTPS" "false" "$env_file"
    fi
    
    # Gitea 配置
    if ! grep -q "^USER_UID=" "$env_file" 2>/dev/null; then
        set_or_update_env_var "USER_UID" "1000" "$env_file"
    fi
    
    if ! grep -q "^USER_GID=" "$env_file" 2>/dev/null; then
        set_or_update_env_var "USER_GID" "1000" "$env_file"
    fi
    
    if ! grep -q "^GITEA_PROTOCOL=" "$env_file" 2>/dev/null; then
        set_or_update_env_var "GITEA_PROTOCOL" "http" "$env_file"
    fi
    
    if ! grep -q "^GITEA_HTTP_PORT=" "$env_file" 2>/dev/null; then
        set_or_update_env_var "GITEA_HTTP_PORT" "3000" "$env_file"
    fi
    
    if ! grep -q "^GITEA_DATA_PATH=" "$env_file" 2>/dev/null; then
        set_or_update_env_var "GITEA_DATA_PATH" "/data/gitea" "$env_file"
    fi
    
    # K8s Proxy 配置
    if ! grep -q "^K8S_PROXY_LISTEN=" "$env_file" 2>/dev/null; then
        set_or_update_env_var "K8S_PROXY_LISTEN" "0.0.0.0:6443" "$env_file"
    fi
    
    if ! grep -q "^K8S_PROXY_TALK=" "$env_file" 2>/dev/null; then
        set_or_update_env_var "K8S_PROXY_TALK" "host.docker.internal:6443" "$env_file"
    fi
    
    if ! grep -q "^K8S_PROXY_PRE_RESOLVE=" "$env_file" 2>/dev/null; then
        set_or_update_env_var "K8S_PROXY_PRE_RESOLVE" "0" "$env_file"
    fi
    
    if ! grep -q "^K8S_PROXY_VERBOSE=" "$env_file" 2>/dev/null; then
        set_or_update_env_var "K8S_PROXY_VERBOSE" "1" "$env_file"
    fi
    
    # Docker 构建配置
    if ! grep -q "^BUILDKIT_INLINE_CACHE=" "$env_file" 2>/dev/null; then
        set_or_update_env_var "BUILDKIT_INLINE_CACHE" "1" "$env_file"
    fi
    
    print_success "✓ 服务默认配置设置完成"
}

# ==========================================
# IP地址检测和模板渲染功能（从env-manager.sh集成）
# ==========================================

# 网络接口配置（根据操作系统自动选择）
get_default_network_interface() {
    case "$OS_TYPE" in
        "macOS")
            echo "en0"  # macOS 默认以太网/Wi-Fi
            ;;
        *)
            echo "eth0"  # Linux 默认以太网
            ;;
    esac
}

get_fallback_interfaces() {
    case "$OS_TYPE" in
        "macOS")
            echo "en0 en1 en2 en3 en4 en5"  # macOS 网络接口
            ;;
        *)
            # Linux 常见网络接口类型：
            # - eth*: 传统命名
            # - enp*: 新式PCI网卡命名
            # - ens*: 新式系统命名
            # - bond*: 网卡绑定
            # - br*: 网桥接口
            # - wlan*/wlp*: 无线网卡
            echo "eth0 eth1 enp0s3 enp0s8 ens33 ens160 ens192 bond0 bond1 br0 br1 wlan0 wlp2s0"
            ;;
    esac
}

# 智能检测活跃的网络接口（优先级：物理网卡 > 绑定接口 > 网桥）
# 排除虚拟网卡：docker, veth, kubernetes, 虚拟机等
detect_active_interface() {
    local active_interfaces=()
    
    if command -v ip >/dev/null 2>&1; then
        # 获取所有 UP 状态且有 IPv4 地址的接口
        # 排除 loopback、docker、kubernetes、虚拟机等虚拟接口
        active_interfaces=($(ip -4 addr show | grep -E '^[0-9]+:' | grep 'state UP' | \
            grep -v 'lo:' | grep -v 'docker' | grep -v 'veth' | \
            grep -v 'virbr' | grep -v 'vboxnet' | grep -v 'vmnet' | \
            awk -F': ' '{print $2}' | awk '{print $1}'))
    elif command -v ifconfig >/dev/null 2>&1; then
        # 使用 ifconfig 获取活跃接口
        # macOS: 排除 bridge100 (Kubernetes)、vmnet (VMware)、utun (VPN) 等虚拟接口
        active_interfaces=($(ifconfig | grep -E '^[a-z]' | grep -v '^lo' | \
            grep -v 'docker' | grep -v 'veth' | grep -v 'bridge' | \
            grep -v 'vmnet' | grep -v 'vboxnet' | grep -v 'utun' | \
            awk '{print $1}' | tr -d ':'))
    fi
    
    # 优先级排序：eth > enp > ens > en (macOS) > bond > br > wlan
    for prefix in "eth" "enp" "ens" "en" "bond" "br" "wlan"; do
        for iface in "${active_interfaces[@]}"; do
            if [[ "$iface" =~ ^${prefix} ]]; then
                # 额外检查：确保不是 Kubernetes 虚拟网卡 (192.168.65.x, 10.96.x.x 等)
                local iface_ip=$(detect_interface_ip "$iface")
                if [[ -n "$iface_ip" ]] && [[ ! "$iface_ip" =~ ^192\.168\.65\. ]] && \
                   [[ ! "$iface_ip" =~ ^10\.96\. ]] && [[ ! "$iface_ip" =~ ^172\.1[6-9]\. ]] && \
                   [[ ! "$iface_ip" =~ ^172\.2[0-9]\. ]] && [[ ! "$iface_ip" =~ ^172\.3[0-1]\. ]]; then
                    echo "$iface"
                    return 0
                fi
            fi
        done
    done
    
    # 如果没有匹配的，返回第一个活跃接口（但仍需检查 IP 范围）
    if [[ ${#active_interfaces[@]} -gt 0 ]]; then
        for iface in "${active_interfaces[@]}"; do
            local iface_ip=$(detect_interface_ip "$iface")
            if [[ -n "$iface_ip" ]] && [[ ! "$iface_ip" =~ ^192\.168\.65\. ]] && \
               [[ ! "$iface_ip" =~ ^10\.96\. ]]; then
                echo "$iface"
                return 0
            fi
        done
    fi
    
    return 1
}

DEFAULT_NETWORK_INTERFACE=$(get_default_network_interface)
FALLBACK_INTERFACES=($(get_fallback_interfaces))

# 检测指定网卡的IP地址
detect_interface_ip() {
    local interface="${1:-$DEFAULT_NETWORK_INTERFACE}"
    local ip=""
    
    # 方法1: 使用ip命令（Linux优先）
    if command -v ip >/dev/null 2>&1; then
        # 使用 ip addr show 获取 IPv4 地址
        ip=$(ip addr show "$interface" 2>/dev/null | grep -E 'inet\s+[0-9.]+' | awk '{print $2}' | cut -d'/' -f1 | head -1)
    fi
    
    # 方法2: 使用ifconfig命令（macOS和旧版Linux）
    if [[ -z "$ip" ]] && command -v ifconfig >/dev/null 2>&1; then
        case "$OS_TYPE" in
            "macOS")
                # macOS ifconfig 格式: inet 192.168.1.100 netmask 0xffffff00 broadcast 192.168.1.255
                ip=$(ifconfig "$interface" 2>/dev/null | grep -E 'inet\s+[0-9.]+' | grep -v '127.0.0.1' | awk '{print $2}' | head -1)
                ;;
            *)
                # Linux ifconfig 旧格式: inet addr:192.168.1.100
                ip=$(ifconfig "$interface" 2>/dev/null | grep -E 'inet addr:' | awk -F: '{print $2}' | awk '{print $1}' | head -1)
                if [[ -z "$ip" ]]; then
                    # Linux ifconfig 新格式: inet 192.168.1.100
                    ip=$(ifconfig "$interface" 2>/dev/null | grep -E 'inet\s+[0-9.]+' | awk '{print $2}' | head -1)
                fi
                ;;
        esac
    fi
    
    echo "$ip"
}

# 自动检测外部主机IP（增强版）
auto_detect_external_ip_enhanced() {
    local detected_ip=""
    
    print_info "自动检测外部主机IP..."
    
    # 方法1: 智能检测活跃网卡
    local active_if=$(detect_active_interface)
    if [[ -n "$active_if" ]]; then
        print_info "检测到活跃网卡: $active_if"
        detected_ip=$(detect_interface_ip "$active_if")
        if [[ -n "$detected_ip" ]]; then
            print_success "在网卡 $active_if 上检测到IP: $detected_ip"
            echo "$detected_ip"
            return 0
        fi
    fi
    
    # 方法2: 优先检测指定网卡
    detected_ip=$(detect_interface_ip "$DEFAULT_NETWORK_INTERFACE")
    if [[ -n "$detected_ip" ]]; then
        print_success "在网卡 $DEFAULT_NETWORK_INTERFACE 上检测到IP: $detected_ip"
        echo "$detected_ip"
        return 0
    fi
    
    # 方法3: 如果指定网卡没有IP，尝试其他网卡
    for interface in "${FALLBACK_INTERFACES[@]}"; do
        print_info "尝试检测网卡: $interface"
        detected_ip=$(detect_interface_ip "$interface")
        if [[ -n "$detected_ip" ]]; then
            print_success "在网卡 $interface 上检测到IP: $detected_ip"
            echo "$detected_ip"
            return 0
        fi
    done
    
    # 方法4: 通过默认路由检测本地IP（不依赖外部网络）
    if [[ -z "$detected_ip" ]]; then
        if command -v ip >/dev/null 2>&1; then
            # Linux: 使用 ip route get 获取本地源地址
            # 使用内网地址避免依赖外网连接
            detected_ip=$(ip route get 1.1.1.1 2>/dev/null | grep -oE 'src [0-9.]+' | awk '{print $2}' | head -1)
            [[ -n "$detected_ip" ]] && print_success "通过默认路由检测到IP: $detected_ip" && echo "$detected_ip" && return 0
        elif command -v route >/dev/null 2>&1 && [[ "$OS_TYPE" == "macOS" ]]; then
            # macOS: 使用 route 命令获取默认网关对应的接口
            local default_if=$(route -n get default 2>/dev/null | grep 'interface:' | awk '{print $2}')
            if [[ -n "$default_if" ]]; then
                detected_ip=$(ifconfig "$default_if" 2>/dev/null | grep -E 'inet\s+[0-9.]+' | grep -v '127.0.0.1' | awk '{print $2}' | head -1)
                [[ -n "$detected_ip" ]] && print_success "通过默认网关接口 $default_if 检测到IP: $detected_ip" && echo "$detected_ip" && return 0
            fi
        fi
    fi
    
    # 方法5: 通过ifconfig检测任意可用IP（排除127.0.0.1）
    if [[ -z "$detected_ip" ]] && command -v ifconfig >/dev/null 2>&1; then
        case "$OS_TYPE" in
            "macOS")
                detected_ip=$(ifconfig | grep -E 'inet\s+[0-9.]+' | grep -v '127.0.0.1' | awk '{print $2}' | head -1)
                ;;
            *)
                detected_ip=$(ifconfig | grep -E 'inet\s+[0-9.]+' | grep -v '127.0.0.1' | awk '{print $2}' | head -1)
                ;;
        esac
        [[ -n "$detected_ip" ]] && print_success "通过ifconfig检测到IP: $detected_ip" && echo "$detected_ip" && return 0
    fi
    
    # 备用方案: 使用localhost
    detected_ip="localhost"
    print_warning "无法自动检测外部IP，使用默认值: localhost"
    echo "$detected_ip"
}

# 静默版本的IP检测（仅返回IP，不输出日志）
auto_detect_external_ip_silent() {
    local detected_ip=""
    
    # 方法1: 智能检测活跃网卡
    local active_if=$(detect_active_interface)
    if [[ -n "$active_if" ]]; then
        detected_ip=$(detect_interface_ip "$active_if")
        if [[ -n "$detected_ip" ]]; then
            echo "$detected_ip"
            return 0
        fi
    fi
    
    # 方法2: 优先检测指定网卡
    detected_ip=$(detect_interface_ip "$DEFAULT_NETWORK_INTERFACE")
    if [[ -n "$detected_ip" ]]; then
        echo "$detected_ip"
        return 0
    fi
    
    # 方法3: 如果指定网卡没有IP，尝试其他网卡
    for interface in "${FALLBACK_INTERFACES[@]}"; do
        detected_ip=$(detect_interface_ip "$interface")
        if [[ -n "$detected_ip" ]]; then
            echo "$detected_ip"
            return 0
        fi
    done
    
    # 方法4: 通过默认路由检测本地IP（不依赖外部网络）
    if command -v ip >/dev/null 2>&1; then
        # Linux: 使用内网地址避免依赖外网连接
        detected_ip=$(ip route get 1.1.1.1 2>/dev/null | grep -oE 'src [0-9.]+' | awk '{print $2}' | head -1)
        if [[ -n "$detected_ip" ]]; then
            echo "$detected_ip"
            return 0
        fi
    elif command -v route >/dev/null 2>&1 && [[ "$OS_TYPE" == "macOS" ]]; then
        # macOS: 获取默认网关对应的接口IP
        local default_if=$(route -n get default 2>/dev/null | grep 'interface:' | awk '{print $2}')
        if [[ -n "$default_if" ]]; then
            detected_ip=$(ifconfig "$default_if" 2>/dev/null | grep -E 'inet\s+[0-9.]+' | grep -v '127.0.0.1' | awk '{print $2}' | head -1)
            if [[ -n "$detected_ip" ]]; then
                echo "$detected_ip"
                return 0
            fi
        fi
    fi
    
    # 方法5: 通过ifconfig检测任意可用IP（排除127.0.0.1）
    if command -v ifconfig >/dev/null 2>&1; then
        case "$OS_TYPE" in
            "macOS")
                detected_ip=$(ifconfig | grep -E 'inet\s+[0-9.]+' | grep -v '127.0.0.1' | awk '{print $2}' | head -1)
                ;;
            *)
                detected_ip=$(ifconfig | grep -E 'inet\s+[0-9.]+' | grep -v '127.0.0.1' | awk '{print $2}' | head -1)
                ;;
        esac
        if [[ -n "$detected_ip" ]]; then
            echo "$detected_ip"
            return 0
        fi
    fi
    
    # 备用方案: 使用localhost
    echo "localhost"
}

# 增强型环境变量模板渲染
render_env_template_enhanced() {
    local template_file="$1"
    local output_file="$2"
    local external_host="$3"
    local external_port="${4:-8080}"
    local external_scheme="${5:-http}"
    local force="${6:-false}"
    
    if [[ ! -f "$template_file" ]]; then
        print_error "模板文件不存在: $template_file"
        return 1
    fi
    
    # 检查目标文件是否已存在
    if [[ -f "$output_file" ]] && [[ "$force" != "true" ]]; then
        print_warning "环境文件已存在: $output_file"
        print_info "如需强制覆盖，请使用 --force 参数"
        read -p "是否覆盖现有文件? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_info "操作已取消"
            return 0
        fi
    fi
    
    print_info "渲染环境变量模板..."
    print_info "  模板文件: $template_file"
    print_info "  输出文件: $output_file"
    print_info "  外部主机: $external_host"
    print_info "  外部端口: $external_port"
    print_info "  外部协议: $external_scheme"
    
    # 创建备份目录
    mkdir -p "$SCRIPT_DIR/backup"
    
    # 创建备份
    if [[ -f "$output_file" ]]; then
        local backup_name="$(basename "$output_file").backup-$(date +%Y%m%d-%H%M%S)"
        cp "$output_file" "$SCRIPT_DIR/backup/$backup_name"
        print_info "已备份原文件: $backup_name"
    fi
    
    # 读取模板内容并替换变量
    local temp_content
    temp_content=$(cat "$template_file")
    
    # 计算各种端口值
    local jupyterhub_port=$((external_port + 8))
    local gitea_port=$((external_port - 5070))
    local apphub_port=$((external_port + 45354))  # AppHub包仓库端口，用于内部包管理
    local https_port=$((external_port + 363))
    local debug_port=$((external_port - 79))
    
    # 从模板内容中提取 SaltStack 配置（如果存在）
    local salt_api_scheme=$(echo "$temp_content" | grep "^SALT_API_SCHEME=" | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "http")
    local salt_master_host=$(echo "$temp_content" | grep "^SALT_MASTER_HOST=" | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "saltstack")
    local salt_api_port=$(echo "$temp_content" | grep "^SALT_API_PORT=" | cut -d'=' -f2 | tr -d '"' | tr -d "'" || echo "8002")
    
    # 构建 SALTSTACK_MASTER_URL（按后端期望格式）
    local saltstack_master_url="${salt_api_scheme}://${salt_master_host}:${salt_api_port}"
    
    # 替换基本模板变量
    temp_content="${temp_content//\$\{EXTERNAL_HOST\}/$external_host}"
    temp_content="${temp_content//\$\{EXTERNAL_PORT\}/$external_port}"
    temp_content="${temp_content//\$\{EXTERNAL_SCHEME\}/$external_scheme}"
    
    # 替换计算后的端口变量
    temp_content="${temp_content//\$\{JUPYTERHUB_PORT\}/$jupyterhub_port}"
    temp_content="${temp_content//\$\{JUPYTERHUB_EXTERNAL_PORT\}/$jupyterhub_port}"
    temp_content="${temp_content//\$\{GITEA_PORT\}/$gitea_port}"
    temp_content="${temp_content//\$\{GITEA_EXTERNAL_PORT\}/$gitea_port}"
    temp_content="${temp_content//\$\{APPHUB_PORT\}/$apphub_port}"
    temp_content="${temp_content//\$\{HTTPS_PORT\}/$https_port}"
    temp_content="${temp_content//\$\{DEBUG_PORT\}/$debug_port}"
    
    # 替换 SALTSTACK_MASTER_URL（如果模板中为空，则填充拼装的值）
    if echo "$temp_content" | grep -q "^SALTSTACK_MASTER_URL=$"; then
        temp_content=$(echo "$temp_content" | sed "s|^SALTSTACK_MASTER_URL=$|SALTSTACK_MASTER_URL=$saltstack_master_url|")
    fi
    
    print_info "  计算的端口值:"
    print_info "    JupyterHub: $jupyterhub_port"
    print_info "    Gitea: $gitea_port"
    print_info "    AppHub: $apphub_port"
    print_info "    HTTPS: $https_port"
    print_info "    Debug: $debug_port"
    print_info "  SaltStack API: $saltstack_master_url"
    
    # 写入输出文件
    echo "$temp_content" > "$output_file"
    
    print_success "✓ 模板渲染完成: $output_file"
}

# Docker Compose命令兼容性检测
get_docker_compose_cmd() {
    if command -v docker-compose >/dev/null 2>&1; then
        echo "docker-compose"
    elif docker compose version >/dev/null 2>&1; then
        echo "docker compose"
    else
        print_error "未找到docker-compose或docker compose命令"
        return 1
    fi
}

# 获取网络接口命令（跨平台）
get_network_info_cmd() {
    case "$OS_TYPE" in
        "macOS")
            echo "ifconfig"
            ;;
        "Linux")
            if command -v ip >/dev/null 2>&1; then
                echo "ip"
            elif command -v ifconfig >/dev/null 2>&1; then
                echo "ifconfig"
            else
                echo "none"
            fi
            ;;
        *)
            echo "none"
            ;;
    esac
}

# 平台兼容性验证
verify_platform_compatibility() {
    print_info "检查平台兼容性..."
    print_info "检测到操作系统: $OS_TYPE"
    
    # 检查必要的命令
    local missing_commands=()
    local commands=("docker" "git" "curl" "awk" "sed" "find")
    
    for cmd in "${commands[@]}"; do
        if ! command -v "$cmd" >/dev/null 2>&1; then
            missing_commands+=("$cmd")
        fi
    done
    
    # 检查Docker Compose
    if ! get_docker_compose_cmd >/dev/null 2>&1; then
        missing_commands+=("docker-compose")
    fi
    
    if [[ ${#missing_commands[@]} -gt 0 ]]; then
        print_error "缺少必要的命令: ${missing_commands[*]}"
        print_info "安装建议："
        
        case "$OS_TYPE" in
            "macOS")
                print_info "  使用Homebrew安装: brew install ${missing_commands[*]}"
                ;;
            "Linux")
                print_info "  使用包管理器安装，例如："
                print_info "  Ubuntu/Debian: sudo apt-get install ${missing_commands[*]}"
                print_info "  CentOS/RHEL: sudo yum install ${missing_commands[*]}"
                ;;
        esac
        
        return 1
    fi
    
    print_success "✓ 平台兼容性检查通过"
    return 0
}

# ==========================================
# 配置文件解析功能
# ==========================================

# 读取TOML配置文件中的值
read_config() {
    local section="$1"
    local key="$2"
    local subsection="$3"
    
    if [[ ! -f "$CONFIG_FILE" ]]; then
        # 配置文件不存在时返回空值，由调用者处理默认值
        return 1
    fi
    
    if [[ -n "$subsection" ]]; then
        # 读取嵌套配置 [section.subsection]
        awk -F' *= *' -v section="$section" -v subsection="$subsection" -v key="$key" '
            /^\[[[:space:]]*[^.]+\.[^]]+\]/ {
                # 匹配 [section.subsection] 格式
                gsub(/^\[|\]$/, "")
                split($0, parts, "\\.")
                if (parts[1] == section && parts[2] == subsection) {
                    in_target = 1
                } else {
                    in_target = 0
                }
                next
            }
            /^\[/ { in_target = 0; next }
            in_target && $1 == key {
                gsub(/^"/, "", $2)
                gsub(/"$/, "", $2)
                print $2
                exit
            }
        ' "$CONFIG_FILE"
    else
        # 读取简单配置 [section]
        awk -F' *= *' -v section="$section" -v key="$key" '
            /^\[[[:space:]]*[^.]+\]/ {
                gsub(/^\[|\]$/, "")
                if ($0 == section) {
                    in_target = 1
                } else {
                    in_target = 0
                }
                next
            }
            /^\[/ { in_target = 0; next }
            in_target && $1 == key {
                gsub(/^"/, "", $2)
                gsub(/"$/, "", $2)
                print $2
                exit
            }
        ' "$CONFIG_FILE"
    fi
}

# 获取所有服务名称
get_all_services() {
    if [[ ! -f "$CONFIG_FILE" ]]; then
        echo "backend frontend jupyterhub nginx saltstack singleuser gitea backend-init apphub slurm-master test-containers"
        return
    fi
    
    awk '
        /^\[services\.[^]]+\]/ {
            gsub(/^\[services\.|\]$/, "")
            print $0
        }
    ' "$CONFIG_FILE" | sort
}

# 获取所有依赖镜像（包含测试工具和构建依赖）
get_all_dependencies() {
    if [[ ! -f "$CONFIG_FILE" ]]; then
        echo "postgres:15-alpine redis:7-alpine osixia/openldap:stable osixia/phpldapadmin:stable tecnativa/tcp-proxy redislabs/redisinsight:latest nginx:1.27-alpine minio/minio:latest node:22-alpine nginx:stable-alpine-perl golang:1.25-alpine python:3.13-alpine gitea/gitea:1.24.6 jupyter/base-notebook:latest"
        return
    fi
    
    awk -F' *= *' '
        /^\[dependencies\]/ { in_dependencies = 1; next }
        /^\[/ { in_dependencies = 0; next }
        in_dependencies && NF > 1 {
            gsub(/^"/, "", $2)
            gsub(/"$/, "", $2)
            print $2
        }
    ' "$CONFIG_FILE" | tr '\n' ' '
}

# 获取生产环境依赖镜像（移除测试工具和构建依赖）
get_production_dependencies() {
    if [[ ! -f "$CONFIG_FILE" ]]; then
        echo "postgres:15-alpine redis:7-alpine tecnativa/tcp-proxy nginx:1.27-alpine minio/minio:latest"
        return
    fi
    
    awk -F' *= *' '
        /^\[dependencies\]/ { in_dependencies = 1; next }
        /^\[/ { in_dependencies = 0; next }
        in_dependencies && NF > 1 {
            gsub(/^"/, "", $2)
            gsub(/"$/, "", $2)
            # 排除测试工具和LDAP服务
            if ($2 !~ /phpldapadmin/ && $2 !~ /redisinsight/ && $2 !~ /openldap/) {
                print $2
            }
        }
    ' "$CONFIG_FILE" | tr '\n' ' '
}

# 初始化配置
DEFAULT_IMAGE_TAG=$(read_config "project" "version" 2>/dev/null || echo "")
[[ -z "$DEFAULT_IMAGE_TAG" ]] && DEFAULT_IMAGE_TAG="v0.3.6-dev"

# 动态更新版本标签函数
update_version_if_provided() {
    local new_version=""
    local args=("$@")
    
    # 查找传入参数中的版本信息
    for i in "${!args[@]}"; do
        local arg="${args[i]}"
        
        # 检查是否是版本格式的参数 (v*.*.* 格式)
        if [[ "$arg" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9]+)?$ ]]; then
            new_version="$arg"
            print_info "检测到版本参数: $new_version，更新默认版本标签"
            break
        fi
        
        # 检查常见的版本标签格式 (如 test-v0.3.6-dev)
        if [[ "$arg" =~ ^[a-zA-Z0-9-]*v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9]+)?$ ]]; then
            new_version="$arg"
            print_info "检测到版本参数: $new_version，更新默认版本标签"
            break
        fi
    done
    
    # 如果找到新版本，更新默认标签和相关变量
    if [[ -n "$new_version" ]]; then
        # 提取纯版本号（去掉前缀）
        local clean_version=$(echo "$new_version" | sed -E 's/^[a-zA-Z0-9-]*(v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9]+)?)$/\1/')
        if [[ -n "$clean_version" ]]; then
            DEFAULT_IMAGE_TAG="$clean_version"
            print_success "版本标签已更新为: $DEFAULT_IMAGE_TAG"
            
            # 更新环境变量以确保一致性
            export IMAGE_TAG="$DEFAULT_IMAGE_TAG"
            
            # 动态更新相关的版本引用
            dynamic_update_version_refs "$DEFAULT_IMAGE_TAG"
        fi
    fi
}

# 动态更新版本引用函数
dynamic_update_version_refs() {
    local new_version="$1"
    
    # 更新JupyterHub镜像版本引用
    if [[ -n "$JUPYTERHUB_IMAGE" ]]; then
        # 提取镜像名称部分，替换版本标签
        local image_base=$(echo "$JUPYTERHUB_IMAGE" | cut -d':' -f1)
        export JUPYTERHUB_IMAGE="${image_base}:${new_version}"
        print_info "JupyterHub镜像版本已更新为: $JUPYTERHUB_IMAGE"
    fi
}

# 动态加载服务和依赖配置
SRC_SERVICES=$(get_all_services | tr '\n' ' ')
DEPENDENCY_IMAGES=$(get_all_dependencies | tr '\n' ' ')

# 动态收集依赖镜像函数
collect_dependency_images() {
    # 优先使用配置文件中的依赖镜像列表
    if [[ -f "$CONFIG_FILE" ]]; then
        echo "$DEPENDENCY_IMAGES"
        return
    fi
    
    # 使用统一的静态依赖列表，确保与get_all_dependencies一致
    echo "postgres:15-alpine redis:7-alpine osixia/openldap:stable osixia/phpldapadmin:stable tecnativa/tcp-proxy redislabs/redisinsight:latest nginx:1.27-alpine minio/minio:latest node:22-alpine nginx:stable-alpine-perl golang:1.25-alpine python:3.13-alpine gitea/gitea:1.24.6 jupyter/base-notebook:latest"
}

# 批量下载基础镜像
batch_download_base_images() {
    print_info "=========================================="
    print_info "批量下载基础镜像"
    print_info "=========================================="
    
    local all_base_images=""
    local unique_images=""
    
    # 1. 收集依赖镜像
    print_info "收集依赖镜像..."
    local dependency_images
    dependency_images=$(collect_dependency_images)
    for dep_image in $dependency_images; do
        if [[ -n "$dep_image" ]]; then
            all_base_images="$all_base_images $dep_image"
        fi
    done
    
    # 2. 收集服务基础镜像（从Dockerfile解析FROM指令）
    print_info "收集服务基础镜像..."
    for service in $SRC_SERVICES; do
        local service_path
        service_path=$(get_service_path "$service")
        
        if [[ -n "$service_path" && -f "$service_path/Dockerfile" ]]; then
            # 解析Dockerfile中的FROM指令
            local from_images
            from_images=$(grep -i '^FROM ' "$service_path/Dockerfile" | sed 's/^FROM //' | sed 's/ AS .*$//' | tr -d '\r')
            
            for from_image in $from_images; do
                # 跳过ARG变量和条件FROM
                if [[ "$from_image" != *'$'* && "$from_image" != *'${'* && "$from_image" != *'--'* ]]; then
                    all_base_images="$all_base_images $from_image"
                fi
            done
        fi
    done
    
    # 3. 去重镜像列表
    for image in $all_base_images; do
        if [[ ! " $unique_images " =~ " $image " ]]; then
            unique_images="$unique_images $image"
        fi
    done
    
    # 4. 批量下载镜像
    local image_count=$(echo "$unique_images" | wc -w)
    print_info "开始批量下载 $image_count 个基础镜像..."
    local success_count=0
    local total_count=0
    local failed_images=()
    
    # 重试下载函数
    retry_pull_image() {
        local image="$1"
        local max_retries="${2:-3}"
        local retry_count=0
        
        while [[ $retry_count -lt $max_retries ]]; do
            if docker pull "$image" 2>/dev/null; then
                return 0
            else
                retry_count=$((retry_count + 1))
                if [[ $retry_count -lt $max_retries ]]; then
                    print_warning "  ↻ 下载失败，重试 $retry_count/$max_retries: $image"
                    sleep 2  # 等待2秒后重试
                fi
            fi
        done
        return 1
    }
    
    for image in $unique_images; do
        if [[ -z "$image" ]]; then
            continue
        fi
        
        total_count=$((total_count + 1))
        print_info "→ 下载: $image"
        
        if retry_pull_image "$image"; then
            print_success "  ✓ 下载成功: $image"
            success_count=$((success_count + 1))
        else
            print_error "  ✗ 下载失败 (重试3次): $image"
            failed_images+=("$image")
        fi
    done
    
    print_info "=========================================="
    print_success "基础镜像下载完成: $success_count/$total_count 成功"
    
    if [[ ${#failed_images[@]} -gt 0 ]]; then
        print_warning "下载失败的镜像: ${failed_images[*]}"
        print_warning "这些镜像将在构建过程中重试下载"
        return 1
    else
        print_success "🎉 所有基础镜像下载成功！"
        return 0
    fi
}

# Mock 数据测试相关配置
MOCK_DATA_ENABLED="${MOCK_DATA_ENABLED:-false}"
MOCK_POSTGRES_IMAGE="postgres:15-alpine"
MOCK_REDIS_IMAGE="redis:7-alpine"

# 获取服务对应的路径
get_service_path() {
    local service="$1"
    
    # 从配置文件读取路径
    local path=$(read_config "services" "path" "$service" 2>/dev/null || echo "")
    
    # 如果配置文件中没有，使用后备方案
    if [[ -z "$path" ]]; then
        case "$service" in
            "backend") echo "src/backend" ;;
            "frontend") echo "src/frontend" ;;
            "jupyterhub") echo "src/jupyterhub" ;;
            "nginx") echo "src/nginx" ;;
            "saltstack") echo "src/saltstack" ;;
            "singleuser") echo "src/singleuser" ;;
            "gitea") echo "src/gitea" ;;
            "backend-init") echo "src/backend" ;;  # backend-init 使用 backend 的 Dockerfile
            "apphub") echo "src/apphub" ;;
            "slurm-master") echo "src/slurm-master" ;;
            "test-containers") echo "src/test-containers" ;;
            *) echo "" ;;
        esac
    else
        echo "$path"
    fi
}

# 颜色输出函数（扩展）
print_success() {
    echo -e "\033[32m[SUCCESS]\033[0m $1"
}

print_warning() {
    echo -e "\033[33m[WARNING]\033[0m $1"
}

# ==========================================
# 智能构建缓存系统
# ==========================================

# 初始化构建缓存目录
init_build_cache() {
    mkdir -p "$BUILD_CACHE_DIR"
    
    # 初始化构建ID文件
    if [[ ! -f "$BUILD_ID_FILE" ]]; then
        echo "0" > "$BUILD_ID_FILE"
    fi
    
    # 初始化构建历史文件
    if [[ ! -f "$BUILD_HISTORY_FILE" ]]; then
        touch "$BUILD_HISTORY_FILE"
    fi
}

# 生成新的构建ID
generate_build_id() {
    init_build_cache
    
    local last_id=$(cat "$BUILD_ID_FILE" 2>/dev/null || echo "0")
    local new_id=$((last_id + 1))
    local timestamp=$(date +%Y%m%d_%H%M%S)
    
    echo "${new_id}_${timestamp}"
}

# 保存构建ID
save_build_id() {
    local build_id="$1"
    init_build_cache
    
    # 提取数字ID部分
    local numeric_id=$(echo "$build_id" | cut -d'_' -f1)
    echo "$numeric_id" > "$BUILD_ID_FILE"
}

# 计算文件或目录的哈希值
calculate_hash() {
    local path="$1"
    
    if [[ ! -e "$path" ]]; then
        echo "NOT_EXIST"
        return 1
    fi
    
    if [[ -d "$path" ]]; then
        # 目录：计算所有文件的综合哈希
        # 排除常见的依赖和构建目录以提升性能
        find "$path" -type f \
            \( -name "*.py" -o -name "*.js" -o -name "*.ts" -o -name "*.tsx" -o -name "*.go" -o -name "*.conf" -o -name "*.yaml" -o -name "*.yml" -o -name "*.json" -o -name "Dockerfile" \) \
            ! -path "*/node_modules/*" \
            ! -path "*/build/*" \
            ! -path "*/dist/*" \
            ! -path "*/.next/*" \
            ! -path "*/vendor/*" \
            ! -path "*/__pycache__/*" \
            ! -path "*/.git/*" \
            -exec shasum -a 256 {} \; 2>/dev/null | sort | shasum -a 256 | awk '{print $1}'
    else
        # 文件：直接计算哈希
        shasum -a 256 "$path" 2>/dev/null | awk '{print $1}'
    fi
}

# 计算服务的综合哈希（包含源码、配置、Dockerfile）
calculate_service_hash() {
    local service="$1"
    local service_path=$(get_service_path "$service")
    
    if [[ -z "$service_path" ]]; then
        echo "INVALID_SERVICE"
        return 1
    fi
    
    local hash_data=""
    
    # 1. Dockerfile哈希
    local dockerfile="$SCRIPT_DIR/$service_path/Dockerfile"
    if [[ -f "$dockerfile" ]]; then
        hash_data+="$(calculate_hash "$dockerfile")\n"
    fi
    
    # 2. 源代码目录哈希
    local src_dir="$SCRIPT_DIR/$service_path"
    if [[ -d "$src_dir" ]]; then
        hash_data+="$(calculate_hash "$src_dir")\n"
    fi
    
    # 3. 配置文件哈希（如果有）
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
            if [[ -d "$SCRIPT_DIR/src/backend" ]]; then
                hash_data+="$(calculate_hash "$SCRIPT_DIR/src/backend")\n"
            fi
            ;;
    esac
    
    # 计算综合哈希
    echo -e "$hash_data" | shasum -a 256 | awk '{print $1}'
}

# 获取镜像中的构建信息标签
get_image_build_labels() {
    local image="$1"
    
    if ! docker image inspect "$image" >/dev/null 2>&1; then
        return 1
    fi
    
    # 提取所有 build.* 标签
    docker image inspect "$image" --format '{{range $k, $v := .Config.Labels}}{{if eq (slice $k 0 6) "build."}}{{$k}}={{$v}}{{"\n"}}{{end}}{{end}}' 2>/dev/null
}

# 检查服务是否需要重新构建
need_rebuild() {
    local service="$1"
    local tag="$2"
    local image="ai-infra-${service}:${tag}"
    
    # 强制重建模式
    if [[ "$FORCE_REBUILD" == "true" ]]; then
        echo "FORCE_REBUILD"
        return 0
    fi
    
    # 跳过缓存检查
    if [[ "$SKIP_CACHE_CHECK" == "true" ]]; then
        echo "SKIP_CACHE_CHECK"
        return 0
    fi
    
    # 镜像不存在，需要构建
    if ! docker image inspect "$image" >/dev/null 2>&1; then
        echo "IMAGE_NOT_EXIST"
        return 0
    fi
    
    # 计算当前文件哈希
    local current_hash=$(calculate_service_hash "$service")
    
    # 获取镜像中保存的哈希
    local image_hash=$(docker image inspect "$image" --format '{{index .Config.Labels "build.hash"}}' 2>/dev/null || echo "")
    
    # 如果镜像没有哈希标签，需要重建
    if [[ -z "$image_hash" ]]; then
        echo "NO_HASH_LABEL"
        return 0
    fi
    
    # 对比哈希值
    if [[ "$current_hash" != "$image_hash" ]]; then
        echo "HASH_CHANGED|old:${image_hash:0:8}|new:${current_hash:0:8}"
        return 0
    fi
    
    # 无需重建
    echo "NO_CHANGE"
    return 1
}

# 记录构建历史
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

# 保存服务构建信息到缓存
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

# 显示构建历史记录
show_build_history() {
    local filter_service="$1"
    local count="${2:-20}"
    
    init_build_cache
    
    if [[ ! -f "$BUILD_HISTORY_FILE" ]]; then
        print_info "📋 构建历史记录为空"
        print_info "提示: 执行构建命令后将自动记录历史"
        return 0
    fi
    
    print_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    print_info "📋 构建历史记录"
    print_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    if [[ -n "$filter_service" ]]; then
        print_info "🔍 过滤服务: $filter_service"
    fi
    print_info "📊 显示记录数: $count"
    echo
    
    # 过滤并显示记录
    local records
    if [[ -n "$filter_service" ]]; then
        records=$(grep "SERVICE=$filter_service " "$BUILD_HISTORY_FILE" | tail -n "$count")
    else
        records=$(tail -n "$count" "$BUILD_HISTORY_FILE")
    fi
    
    if [[ -z "$records" ]]; then
        print_info "没有找到匹配的记录"
        return 0
    fi
    
    # 表头
    printf "%-20s %-15s %-20s %-10s %-10s %-20s\n" \
        "时间" "BUILD_ID" "服务" "标签" "状态" "原因"
    echo "────────────────────────────────────────────────────────────────────────────────────────────────"
    
    # 显示记录（彩色输出）
    while IFS= read -r line; do
        # 提取字段
        local timestamp=$(echo "$line" | sed 's/^\[\([^]]*\)\].*/\1/')
        local build_id=$(echo "$line" | grep -o 'BUILD_ID=[^ ]*' | cut -d= -f2)
        local service=$(echo "$line" | grep -o 'SERVICE=[^ ]*' | cut -d= -f2)
        local tag=$(echo "$line" | grep -o 'TAG=[^ ]*' | cut -d= -f2)
        local status=$(echo "$line" | grep -o 'STATUS=[^ ]*' | cut -d= -f2)
        local reason=$(echo "$line" | grep -o 'REASON=.*' | cut -d= -f2 || echo "-")
        
        # 根据状态选择颜色
        case "$status" in
            "SUCCESS")
                printf "\033[32m%-20s %-15s %-20s %-10s ✓ SUCCESS  %-20s\033[0m\n" \
                    "$timestamp" "$build_id" "$service" "$tag" "$reason"
                ;;
            "FAILED")
                printf "\033[31m%-20s %-15s %-20s %-10s ✗ FAILED   %-20s\033[0m\n" \
                    "$timestamp" "$build_id" "$service" "$tag" "$reason"
                ;;
            "SKIPPED")
                printf "\033[33m%-20s %-15s %-20s %-10s ⊘ SKIPPED  %-20s\033[0m\n" \
                    "$timestamp" "$build_id" "$service" "$tag" "$reason"
                ;;
            *)
                printf "%-20s %-15s %-20s %-10s %-10s %-20s\n" \
                    "$timestamp" "$build_id" "$service" "$tag" "$status" "$reason"
                ;;
        esac
    done <<< "$records"
    
    echo
    print_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # 统计信息
    local total_count=$(echo "$records" | wc -l | tr -d ' ')
    local success_count=$(echo "$records" | grep -c "STATUS=SUCCESS" || echo "0")
    local failed_count=$(echo "$records" | grep -c "STATUS=FAILED" || echo "0")
    local skipped_count=$(echo "$records" | grep -c "STATUS=SKIPPED" || echo "0")
    
    print_info "📊 统计: 总计=$total_count | 成功=$success_count | 失败=$failed_count | 跳过=$skipped_count"
}

# 显示镜像构建信息
show_build_info() {
    local service="$1"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    local image="ai-infra-${service}:${tag}"
    
    print_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    print_info "🔍 镜像构建信息"
    print_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    print_info "服务: $service"
    print_info "镜像: $image"
    echo
    
    # 检查镜像是否存在
    if ! docker image inspect "$image" >/dev/null 2>&1; then
        print_error "镜像不存在: $image"
        print_info "提示: 使用 './build.sh build $service $tag' 构建镜像"
        return 1
    fi
    
    print_success "✓ 镜像存在"
    echo
    
    # 获取构建标签
    local labels=$(get_image_build_labels "$image")
    
    if [[ -z "$labels" ]]; then
        print_warning "镜像没有构建标签（可能是旧版本构建）"
        echo
        print_info "基本信息:"
        docker image inspect "$image" --format \
            '  创建时间: {{.Created}}
  大小: {{.Size}} bytes
  架构: {{.Architecture}}
  OS: {{.Os}}'
        return 0
    fi
    
    # 显示构建标签
    print_info "🏷️  构建标签:"
    echo "$labels" | while IFS='=' read -r key value; do
        case "$key" in
            "build.id")
                echo "  📋 BUILD_ID: $value"
                ;;
            "build.service")
                echo "  🔧 服务: $value"
                ;;
            "build.tag")
                echo "  🏷️  标签: $value"
                ;;
            "build.hash")
                echo "  #️⃣  哈希: ${value:0:16}..."
                ;;
            "build.timestamp")
                echo "  🕐 时间: $value"
                ;;
            "build.reason")
                echo "  📝 原因: $value"
                ;;
        esac
    done
    
    echo
    print_info "📦 镜像详情:"
    docker image inspect "$image" --format \
        '  创建时间: {{.Created}}
  大小: {{.Size}} bytes
  架构: {{.Architecture}}
  OS: {{.Os}}'
    
    # 检查缓存文件
    local cache_file="$BUILD_CACHE_DIR/$service/last-build.json"
    if [[ -f "$cache_file" ]]; then
        echo
        print_info "💾 缓存信息:"
        cat "$cache_file" | jq -r '. | "  BUILD_ID: \(.build_id)\n  哈希: \(.hash[0:16])...\n  时间戳: \(.timestamp)"' 2>/dev/null || cat "$cache_file"
    fi
}

# 显示构建缓存统计
show_build_cache_stats() {
    echo "=========================================="
    echo "构建缓存统计"
    echo "=========================================="
    
    if [[ ! -d "$BUILD_CACHE_DIR" ]]; then
        echo "缓存目录不存在"
        return
    fi
    
    local total_builds=$(cat "$BUILD_ID_FILE" 2>/dev/null || echo "0")
    echo "总构建次数: $total_builds"
    
    if [[ -f "$BUILD_HISTORY_FILE" ]]; then
        echo ""
        echo "最近10次构建:"
        tail -n 10 "$BUILD_HISTORY_FILE"
    fi
    
    echo ""
    echo "各服务缓存状态:"
    for service_dir in "$BUILD_CACHE_DIR"/*; do
        if [[ -d "$service_dir" ]]; then
            local service=$(basename "$service_dir")
            local build_info="$service_dir/last-build.json"
            if [[ -f "$build_info" ]]; then
                local last_tag=$(grep '"tag"' "$build_info" | cut -d'"' -f4)
                local last_time=$(grep '"timestamp"' "$build_info" | cut -d'"' -f4)
                echo "  • $service: tag=$last_tag, time=$last_time"
            fi
        fi
    done
}

# 清理构建缓存
clean_build_cache() {
    local service="${1:-}"
    
    if [[ -n "$service" ]]; then
        # 清理特定服务的缓存
        if [[ -d "$BUILD_CACHE_DIR/$service" ]]; then
            rm -rf "$BUILD_CACHE_DIR/$service"
            print_success "已清理 $service 的构建缓存"
        else
            print_warning "服务 $service 没有构建缓存"
        fi
    else
        # 清理所有缓存
        if [[ -d "$BUILD_CACHE_DIR" ]]; then
            rm -rf "$BUILD_CACHE_DIR"
            print_success "已清理所有构建缓存"
        else
            print_warning "构建缓存目录不存在"
        fi
    fi
}

# ==========================================
# 智能构建功能 - SingleUser 镜像优化
# ==========================================

# 检测网络环境（内网/外网）
detect_network_environment() {
    local timeout=5
    
    # 优先级1：检查强制环境变量（用于测试或特殊场景）
    # 注意：这是强制覆盖，仅在明确需要时设置
    if [[ -n "${AI_INFRA_NETWORK_ENV_OVERRIDE}" ]]; then
        echo "${AI_INFRA_NETWORK_ENV_OVERRIDE}"
        return 0
    fi
    
    # 优先级2：实际网络检测（推荐）
    # 检测方法1：尝试连接常见的外网地址
    if timeout $timeout ping -c 1 8.8.8.8 >/dev/null 2>&1 || 
       timeout $timeout ping -c 1 mirrors.aliyun.com >/dev/null 2>&1; then
        echo "external"
        return 0
    fi
    
    # 检测方法2：检查是否能访问公网服务
    if timeout $timeout curl -s --connect-timeout $timeout https://mirrors.aliyun.com/pypi/simple/ >/dev/null 2>&1; then
        echo "external"
        return 0
    fi
    
    # 优先级3：.env 文件配置（向后兼容，但不推荐）
    # 仅在网络检测失败且明确配置时使用
    if [[ "${AI_INFRA_NETWORK_ENV}" == "external" ]]; then
        echo "external"
        return 0
    fi
    
    # 默认判定为内网环境（安全起见）
    echo "internal"
}

# 检测外部主机地址
# 用于自动配置 EXTERNAL_HOST 变量
detect_external_host() {
    local detected_ip=""
    
    # 智能检测：排除虚拟网络接口，优先选择真实的以太网/Wi-Fi接口
    # macOS 和 Linux 通用方法
    
    # 方法1：使用 ifconfig（macOS 和 BSD）
    if command -v ifconfig &> /dev/null; then
        # 获取所有 inet 地址，排除：
        # - 127.0.0.1 (loopback)
        # - 10.211.* (Parallels 虚拟网络)
        # - 10.37.* (VMware 虚拟网络)
        # - 10.96.* (Kubernetes Service 网络)
        # - 192.168.64.* (Docker/虚拟机桥接)
        # - 192.168.65.* (Kubernetes Docker Desktop)
        # - 172.16-31.* (Docker 默认网络)
        detected_ip=$(ifconfig | grep "inet " | grep -v "127.0.0.1" | \
            grep -v "10.211." | grep -v "10.37." | grep -v "10.96." | \
            grep -v "192.168.64." | grep -v "192.168.65." | \
            grep -v "172.1[6-9]." | grep -v "172.2[0-9]." | grep -v "172.3[0-1]." | \
            awk '{print $2}' | head -n1)
    fi
    
    # 方法2：使用 ip（Linux）
    if [[ -z "$detected_ip" ]] && command -v ip &> /dev/null; then
        # 排除虚拟网络接口
        detected_ip=$(ip addr show | grep "inet " | grep -v "127.0.0.1" | \
            grep -v "10.211." | grep -v "10.37." | grep -v "10.96." | \
            grep -v "192.168.64." | grep -v "192.168.65." | \
            grep -v "172.1[6-9]." | grep -v "172.2[0-9]." | grep -v "172.3[0-1]." | \
            grep -v "docker" | grep -v "veth" | grep -v "bridge" | \
            awk '{print $2}' | cut -d'/' -f1 | head -n1)
    fi
    
    # 方法3：使用 hostname（通用降级方案）
    if [[ -z "$detected_ip" ]] && command -v hostname &> /dev/null; then
        detected_ip=$(hostname -I 2>/dev/null | awk '{print $1}')
        # 再次检查是否为虚拟IP
        if [[ "$detected_ip" =~ ^192\.168\.65\. ]] || [[ "$detected_ip" =~ ^10\.96\. ]]; then
            detected_ip=""
        fi
    fi
    
    # 方法4：从 .env 文件读取已配置的值（降级方案）
    if [[ -z "$detected_ip" ]] && [[ -f ".env" ]]; then
        detected_ip=$(grep "^EXTERNAL_HOST=" .env 2>/dev/null | cut -d'=' -f2)
    fi
    
    # 如果检测到 IP，返回；否则返回默认值
    if [[ -n "$detected_ip" ]]; then
        echo "$detected_ip"
    else
        echo "localhost"
    fi
}

# 检测或使用域名配置（K8s 集群扩展支持）
# 优先级: 环境变量 EXTERNAL_DOMAIN > .env 文件 > 自动检测的 IP
# 用法: detect_external_domain
detect_external_domain() {
    local domain=""
    
    # 优先级1: 环境变量（用于 K8s 部署时手动指定）
    if [[ -n "${EXTERNAL_DOMAIN}" ]]; then
        echo "${EXTERNAL_DOMAIN}"
        return 0
    fi
    
    # 优先级2: 从 .env 文件读取已配置的域名
    if [[ -f ".env" ]]; then
        domain=$(grep "^DOMAIN=" .env 2>/dev/null | cut -d'=' -f2)
        # 检查是否是域名（包含字母）而非纯 IP
        if [[ -n "$domain" ]] && [[ "$domain" =~ [a-zA-Z] ]]; then
            echo "$domain"
            return 0
        fi
    fi
    
    # 优先级3: 降级到 IP 地址检测
    detect_external_host
}

# 智能选择外部访问地址（域名优先，IP 降级）
# 返回: 域名或 IP 地址
# 用法: get_external_address
get_external_address() {
    local address=""
    
    # 首先尝试获取域名
    address=$(detect_external_domain)
    
    # 如果域名检测失败或返回 localhost，降级到 IP 检测
    if [[ -z "$address" ]] || [[ "$address" == "localhost" ]]; then
        address=$(detect_external_host)
    fi
    
    echo "$address"
}

# 判断地址是否为域名（包含字母）
# 用法: is_domain "example.com" && echo "是域名"
is_domain() {
    local address="$1"
    [[ "$address" =~ [a-zA-Z] ]]
}

# 判断是否在 K8s 环境中运行
# 检查方法: 
# 1. 环境变量 KUBERNETES_SERVICE_HOST
# 2. /var/run/secrets/kubernetes.io 目录
# 3. kubectl 命令可用且连接的是真实集群（非 Docker Desktop 本地集群）
detect_k8s_environment() {
    # 优先级0: 检查强制环境变量（用于明确指定）
    if [[ -n "${AI_INFRA_FORCE_K8S}" ]]; then
        echo "${AI_INFRA_FORCE_K8S}"
        return 0
    fi
    
    # 方法1: 检查 K8s 服务环境变量（在 Pod 内运行）
    if [[ -n "${KUBERNETES_SERVICE_HOST}" ]]; then
        echo "true"
        return 0
    fi
    
    # 方法2: 检查 K8s ServiceAccount 挂载（在 Pod 内运行）
    if [[ -d "/var/run/secrets/kubernetes.io" ]]; then
        echo "true"
        return 0
    fi
    
    # 方法3: 检查 kubectl 是否可用且连接的是真实集群
    if command -v kubectl &> /dev/null; then
        # 检查是否能连接集群
        if kubectl cluster-info &> /dev/null; then
            # 进一步检查是否为 Docker Desktop 本地集群
            local k8s_context=$(kubectl config current-context 2>/dev/null)
            
            # 排除 Docker Desktop 本地集群的上下文名称
            if [[ "$k8s_context" =~ docker-desktop|docker-for-desktop|minikube|kind ]]; then
                # 这是本地开发集群，不视为真实 K8s 环境
                echo "false"
                return 1
            fi
            
            # 检查节点数量，单节点很可能是本地环境
            local node_count=$(kubectl get nodes --no-headers 2>/dev/null | wc -l)
            if [[ $node_count -eq 1 ]]; then
                # 单节点可能是本地环境，进一步检查节点名称
                local node_name=$(kubectl get nodes --no-headers 2>/dev/null | awk '{print $1}')
                if [[ "$node_name" =~ docker-desktop|minikube|kind ]]; then
                    echo "false"
                    return 1
                fi
            fi
            
            # 通过所有检查，判定为真实 K8s 环境
            echo "true"
            return 0
        fi
    fi
    
    echo "false"
}

# 获取 K8s 服务的外部访问地址
# 支持 LoadBalancer、NodePort、Ingress 等多种暴露方式
# 用法: get_k8s_external_address <service-name> [namespace]
get_k8s_external_address() {
    local service_name="${1:-nginx}"
    local namespace="${2:-${K8S_NAMESPACE:-ai-infra}}"
    local address=""
    
    # 检查 kubectl 是否可用
    if ! command -v kubectl &> /dev/null; then
        return 1
    fi
    
    # 方法1: LoadBalancer 类型服务的 External IP
    address=$(kubectl get svc "$service_name" -n "$namespace" -o jsonpath='{.status.loadBalancer.ingress[0].ip}' 2>/dev/null)
    if [[ -n "$address" ]]; then
        echo "$address"
        return 0
    fi
    
    # 方法2: LoadBalancer 类型服务的 Hostname（AWS ELB 等）
    address=$(kubectl get svc "$service_name" -n "$namespace" -o jsonpath='{.status.loadBalancer.ingress[0].hostname}' 2>/dev/null)
    if [[ -n "$address" ]]; then
        echo "$address"
        return 0
    fi
    
    # 方法3: Ingress 的 Host
    address=$(kubectl get ingress -n "$namespace" -o jsonpath='{.items[0].spec.rules[0].host}' 2>/dev/null)
    if [[ -n "$address" ]]; then
        echo "$address"
        return 0
    fi
    
    # 方法4: 任意节点 IP（NodePort 模式）
    address=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="ExternalIP")].address}' 2>/dev/null)
    if [[ -n "$address" ]]; then
        echo "$address"
        return 0
    fi
    
    # 降级: 获取内部 IP
    address=$(kubectl get nodes -o jsonpath='{.items[0].status.addresses[?(@.type=="InternalIP")].address}' 2>/dev/null)
    if [[ -n "$address" ]]; then
        echo "$address"
        return 0
    fi
    
    return 1
}

# 更新 .env 文件中的变量
# 用法: update_env_variable "VAR_NAME" "var_value"
update_env_variable() {
    local var_name="$1"
    local var_value="$2"
    local env_file=".env"
    
    # 如果 .env 文件不存在，从示例文件创建
    if [[ ! -f "$env_file" ]]; then
        if [[ -f "docker-compose.yml.example" ]]; then
            print_info "创建 .env 文件（基于 docker-compose.yml.example）"
            # 提取示例文件中的环境变量
            grep "^[A-Z]" docker-compose.yml.example > "$env_file" 2>/dev/null || touch "$env_file"
        else
            print_info "创建空白 .env 文件"
            touch "$env_file"
        fi
    fi
    
    # 检查变量是否已存在
    if grep -q "^${var_name}=" "$env_file"; then
        # 更新现有变量
        # macOS 兼容的 sed 语法
        if [[ "$OSTYPE" == "darwin"* ]]; then
            sed -i '' "s|^${var_name}=.*|${var_name}=${var_value}|" "$env_file"
        else
            sed -i "s|^${var_name}=.*|${var_name}=${var_value}|" "$env_file"
        fi
        print_info "✓ 更新 ${var_name}=${var_value}"
    else
        # 添加新变量
        echo "${var_name}=${var_value}" >> "$env_file"
        print_info "✓ 添加 ${var_name}=${var_value}"
    fi
}

# 自动生成或更新 .env 文件
# 基于网络环境检测和系统配置
# 支持域名和 K8s 集群部署
generate_or_update_env_file() {
    print_info "=========================================="
    print_info "自动检测和配置环境变量"
    print_info "=========================================="
    
    # 1. 检测运行环境
    local is_k8s=$(detect_k8s_environment)
    if [[ "$is_k8s" == "true" ]]; then
        print_info "🎯 检测到 Kubernetes 环境"
    else
        print_info "🐳 检测到 Docker Compose 环境"
    fi
    
    # 2. 检测网络环境
    local detected_env=$(detect_network_environment)
    print_info "🌐 检测到网络环境: $detected_env"
    
    # 3. 智能检测外部访问地址（支持域名和 IP）
    local detected_address=""
    
    if [[ "$is_k8s" == "true" ]]; then
        # K8s 环境: 尝试获取 LoadBalancer/Ingress 地址
        detected_address=$(get_k8s_external_address "nginx" "${K8S_NAMESPACE:-ai-infra}")
        if [[ -z "$detected_address" ]]; then
            print_warning "⚠️  无法获取 K8s 外部地址，降级到本地检测"
            detected_address=$(get_external_address)
        else
            print_info "☸️  K8s 外部地址: $detected_address"
        fi
    else
        # Docker Compose 环境: 使用本地检测
        detected_address=$(get_external_address)
    fi
    
    # 判断是域名还是 IP
    local address_type="IP"
    if is_domain "$detected_address"; then
        address_type="域名"
    fi
    print_info "🖥️  检测到外部地址: $detected_address ($address_type)"
    
    # 4. 读取当前端口配置（如果存在）
    local current_port="${EXTERNAL_PORT:-8080}"
    if [[ -f ".env" ]]; then
        current_port=$(grep "^EXTERNAL_PORT=" .env 2>/dev/null | cut -d'=' -f2 || echo "8080")
    fi
    print_info "🔌 使用外部端口: $current_port"
    
    # 5. 构建完整的基础 URL
    local base_url="http://${detected_address}:${current_port}"
    print_info "🌍 基础访问地址: $base_url"
    
    # 6. 更新 .env 文件中的所有相关配置
    print_info ""
    print_info "📝 更新 .env 文件中的相关配置..."
    
    # 基础配置
    update_env_variable "AI_INFRA_NETWORK_ENV" "$detected_env"
    update_env_variable "EXTERNAL_HOST" "$detected_address"
    update_env_variable "DOMAIN" "$detected_address"
    
    # MinIO 配置
    update_env_variable "MINIO_CONSOLE_URL" "${base_url}/minio-console/"
    
    # JupyterHub 配置
    update_env_variable "JUPYTERHUB_PUBLIC_HOST" "${detected_address}:${current_port}"
    update_env_variable "JUPYTERHUB_BASE_URL" "${base_url}/jupyter/"
    update_env_variable "JUPYTERHUB_CORS_ORIGIN" "$base_url"
    
    # Gitea 配置
    update_env_variable "ROOT_URL" "${base_url}/gitea/"
    update_env_variable "STATIC_URL_PREFIX" "/gitea"
    
    # 7. 显示更新摘要
    print_info ""
    print_info "✅ 环境配置完成："
    print_info "   - 运行环境: $([ "$is_k8s" == "true" ] && echo "Kubernetes" || echo "Docker Compose")"
    print_info "   - 网络环境: $detected_env"
    print_info "   - 外部地址: $detected_address ($address_type)"
    print_info "   - 外部端口: $current_port"
    print_info "   - 基础URL: $base_url"
    print_info ""
    print_info "📋 已更新的配置项："
    print_info "   - DOMAIN → $detected_address"
    print_info "   - MINIO_CONSOLE_URL → ${base_url}/minio-console/"
    print_info "   - JUPYTERHUB_PUBLIC_HOST → ${detected_address}:${current_port}"
    print_info "   - JUPYTERHUB_BASE_URL → ${base_url}/jupyter/"
    print_info "   - JUPYTERHUB_CORS_ORIGIN → $base_url"
    print_info "   - ROOT_URL → ${base_url}/gitea/"
    print_info "   - STATIC_URL_PREFIX → /gitea"
    
    # 8. K8s 环境特殊提示
    if [[ "$is_k8s" == "true" ]]; then
        print_info ""
        print_info "💡 K8s 集群部署提示："
        print_info "   - 如需使用固定域名，请设置环境变量: export EXTERNAL_DOMAIN=your-domain.com"
        print_info "   - 如需更新服务地址，请重新运行: ./build.sh build-all"
    fi
    
    # 9. 重新加载环境变量
    if [[ -f ".env" ]]; then
        set -a
        source .env
        set +a
        print_info ""
        print_info "✅ 已重新加载 .env 文件"
    fi
    
    echo
}

# 生成离线友好的 Dockerfile 内容
generate_offline_singleuser_dockerfile() {
    # 获取当前版本标签，默认使用v0.3.6-dev
    local version_tag="${TARGET_TAG:-v0.3.6-dev}"
    local aiharbor_registry="${INTERNAL_REGISTRY:-aiharbor.msxf.local}"
    
    cat << OFFLINE_EOF
# ai-infra single-user notebook image - 离线部署版本
# 直接使用 aiharbor 内部已构建完成的镜像，无需重新构建
FROM ${aiharbor_registry}/aihpc/ai-infra-singleuser:${version_tag}

# Version metadata - 继承内部镜像版本
ARG VERSION="${version_tag}"
ENV APP_VERSION=\${VERSION}

# ========================================
# 离线部署优化配置
# ========================================
# 该镜像已在 aiharbor 内部完成所有构建和配置：
# - JupyterHub 5.3.x 兼容
# - JupyterLab 完整环境
# - 预装开发工具和科学计算包
# - 预构建扩展，无需运行时编译
# ========================================

USER \${NB_UID}

# 确保配置目录存在（防御性配置）
ENV JUPYTER_ENABLE_LAB=yes
ENV JUPYTERLAB_SETTINGS_DIR=/home/jovyan/.jupyter/lab/user-settings

# 验证内部镜像完整性
RUN echo "✓ 使用 aiharbor 内部预构建镜像: ${aiharbor_registry}/aihpc/ai-infra-singleuser:${version_tag}" && \
    python -c "import sys; print(f'✓ Python {sys.version}'); import jupyterhub, jupyterlab, ipykernel; print('✓ 核心组件已就绪')" && \
    jupyter --version

LABEL maintainer="AI Infrastructure Team" \
    org.opencontainers.image.title="ai-infra-singleuser-offline" \
    org.opencontainers.image.version="\${APP_VERSION}" \
    org.opencontainers.image.description="AI Infra Matrix - Singleuser Notebook (Offline Ready - Harbor Internal)" \
    org.opencontainers.image.source="${aiharbor_registry}/aihpc/ai-infra-singleuser:${version_tag}"

OFFLINE_EOF
}

# 离线构建模式的 Dockerfile 生成（当 aiharbor 镜像不可用时的回退方案）
generate_offline_build_dockerfile() {
    cat << 'OFFLINE_BUILD_EOF'
# ai-infra single-user notebook image pinned to JupyterHub 5.3.x
# Base on jupyter/docker-stacks base-notebook for a full Lab experience
FROM jupyter/base-notebook:latest

# Version metadata
ARG VERSION="dev"
ENV APP_VERSION=${VERSION}

USER root

# ========================================
# 构建阶段：预安装所有必要的Python包
# ========================================
# 配置pip镜像源（构建时使用，运行时不依赖网络）
RUN pip config set global.index-url https://mirrors.aliyun.com/pypi/simple/ && \
    pip config set global.trusted-host mirrors.aliyun.com

# 安装所有必要的Python包（构建阶段完成，运行时无需网络）
RUN pip install --no-cache-dir \
    "jupyterhub==5.3.*" \
    ipykernel \
    jupyterlab \
    jupyterlab-execute-time \
    jupyterlab-code-formatter \
    jupyterlab-lsp \
    python-lsp-server[all]

# 安装额外的开发工具（可选，在构建时决定是否包含）
RUN pip install --no-cache-dir \
    numpy \
    pandas \
    matplotlib \
    seaborn \
    scikit-learn \
    requests

# ========================================
# 配置阶段：设置Jupyter环境
# ========================================
# 预启用JupyterLab
ENV JUPYTER_ENABLE_LAB=yes

# 切换到普通用户进行配置
USER ${NB_UID}

# 预配置JupyterLab设置目录
ENV JUPYTERLAB_SETTINGS_DIR=/home/jovyan/.jupyter/lab/user-settings
RUN mkdir -p ${JUPYTERLAB_SETTINGS_DIR}/jupyterlab-execute-time

# 预启用执行时间显示扩展
RUN echo '{"enabled": true}' > ${JUPYTERLAB_SETTINGS_DIR}/jupyterlab-execute-time/plugin.jupyterlab-settings

# 预安装和配置Python内核（确保在构建时完成）
RUN python -m ipykernel install --user --name python3 --display-name "Python 3 (ipykernel)"

# 预构建JupyterLab扩展（避免运行时构建，使用更宽松的设置）
RUN jupyter lab build --dev-build=False --minimize=False || \
    jupyter lab build --dev-build=False --minimize=False --debug || \
    echo "Warning: JupyterLab build failed, will build at runtime"

# ========================================
# 验证阶段：确保所有组件正常工作
# ========================================
# 验证关键组件是否正确安装
RUN python -c "import jupyterhub, jupyterlab, ipykernel; print('✓ 核心组件验证成功')" && \
    jupyter --version && \
    jupyter lab --version

LABEL maintainer="AI Infrastructure Team" \
    org.opencontainers.image.title="ai-infra-singleuser" \
    org.opencontainers.image.version="${APP_VERSION}" \
    org.opencontainers.image.description="AI Infra Matrix - Singleuser Notebook (Offline Build Mode)"

OFFLINE_BUILD_EOF
}

# 生成在线友好的 Dockerfile 内容（原版）
generate_online_singleuser_dockerfile() {
    cat << 'ONLINE_EOF'
# ai-infra single-user notebook image pinned to JupyterHub 5.3.x
# Base on jupyter/docker-stacks base-notebook for a full Lab experience
FROM jupyter/base-notebook:latest

# Version metadata
ARG VERSION="dev"
ENV APP_VERSION=${VERSION}

USER root

# Align jupyterhub-singleuser with Hub 5.3.x to avoid auth/redirect quirks
RUN pip config set global.index-url https://mirrors.aliyun.com/pypi/simple/ && \
    pip config set global.trusted-host mirrors.aliyun.com && \
    pip install --no-cache-dir \
	"jupyterhub==5.3.*" \
	ipykernel \
	jupyterlab \
	jupyterlab-execute-time \
	jupyterlab-code-formatter \
	jupyterlab-lsp \
	python-lsp-server[all]

# Optional: pre-enable Lab (the base image already does, keep explicit)
ENV JUPYTER_ENABLE_LAB=yes

# Pre-enable execute time display
ENV JUPYTERLAB_SETTINGS_DIR=/home/jovyan/.jupyter/lab/user-settings
USER ${NB_UID}
RUN mkdir -p ${JUPYTERLAB_SETTINGS_DIR}/jupyterlab-execute-time && \
	echo '{"enabled": true}' > ${JUPYTERLAB_SETTINGS_DIR}/jupyterlab-execute-time/plugin.jupyterlab-settings || true

# Ensure ipykernel is available
RUN python -m ipykernel install --user --name python3 --display-name "Python 3 (ipykernel)" || true

LABEL maintainer="AI Infrastructure Team" \
	org.opencontainers.image.title="ai-infra-singleuser" \
	org.opencontainers.image.version="${APP_VERSION}" \
	org.opencontainers.image.description="AI Infra Matrix - Singleuser Notebook"

ONLINE_EOF
}

# 智能准备 SingleUser Dockerfile
prepare_singleuser_dockerfile() {
    local service_path="$1"
    local network_env="$2"
    local force_mode="${3:-auto}"  # auto, offline, online
    
    local dockerfile_path="$SCRIPT_DIR/$service_path/Dockerfile"
    local dockerfile_backup="$SCRIPT_DIR/$service_path/Dockerfile.backup"
    
    # 备份原始 Dockerfile（如果还没备份）
    if [[ ! -f "$dockerfile_backup" ]]; then
        if [[ -f "$dockerfile_path" ]]; then
            cp "$dockerfile_path" "$dockerfile_backup"
            print_info "已备份原始 Dockerfile: $dockerfile_backup"
        fi
    fi
    
    # 根据环境和强制模式决定使用哪种模板
    local use_offline=false
    case "$force_mode" in
        "offline")
            use_offline=true
            print_info "强制使用离线模式构建 SingleUser 镜像"
            ;;
        "online")
            use_offline=false
            print_info "强制使用在线模式构建 SingleUser 镜像"
            ;;
        "auto"|*)
            if [[ "$network_env" == "internal" ]]; then
                use_offline=true
                print_info "检测到内网环境，使用离线友好模式构建 SingleUser 镜像"
            else
                use_offline=false
                print_info "检测到外网环境，使用标准模式构建 SingleUser 镜像"
            fi
            ;;
    esac
    
    # 生成对应的 Dockerfile
    if [[ "$use_offline" == "true" ]]; then
        # 验证 aiharbor 镜像是否可用
        local version_tag="${TARGET_TAG:-v0.3.6-dev}"
        local aiharbor_registry="${INTERNAL_REGISTRY:-aiharbor.msxf.local}"
        local harbor_image="${aiharbor_registry}/aihpc/ai-infra-singleuser:${version_tag}"
        
        print_info "检查 aiharbor 内部镜像可用性..."
        if docker manifest inspect "$harbor_image" &>/dev/null; then
            print_success "✓ aiharbor 内部镜像可用: $harbor_image"
            generate_offline_singleuser_dockerfile > "$dockerfile_path"
            print_success "✓ 已生成离线模式 Dockerfile (使用 aiharbor 预构建镜像)"
        else
            print_warning "⚠ aiharbor 内部镜像不可用: $harbor_image"
            print_info "回退到离线构建模式 (预装依赖)..."
            generate_offline_build_dockerfile > "$dockerfile_path"
            print_success "✓ 已生成离线构建模式 Dockerfile (预装依赖)"
        fi
    else
        generate_online_singleuser_dockerfile > "$dockerfile_path"
        print_success "✓ 已生成标准的 SingleUser Dockerfile"
    fi
}

# 恢复 SingleUser Dockerfile 到原始状态
restore_singleuser_dockerfile() {
    local service_path="$1"
    local dockerfile_path="$SCRIPT_DIR/$service_path/Dockerfile"
    local dockerfile_backup="$SCRIPT_DIR/$service_path/Dockerfile.backup"
    
    if [[ -f "$dockerfile_backup" ]]; then
        cp "$dockerfile_backup" "$dockerfile_path"
        print_success "✓ 已恢复 SingleUser Dockerfile 到原始状态"
        return 0
    else
        print_warning "未找到 Dockerfile 备份文件，无法恢复"
        return 1
    fi
}

# ==========================================
# 模板渲染功能
# ==========================================

# 从 docker-compose.yml 加载环境变量
load_environment_variables() {
    local env_file="$SCRIPT_DIR/.env"
    
    # 检测外部主机地址
    local detected_host="localhost"
    local detected_port="8080"
    
    # 优先从.env文件读取EXTERNAL_HOST
    if [[ -f "$env_file" ]] && grep -q "^EXTERNAL_HOST=" "$env_file"; then
        detected_host=$(grep "^EXTERNAL_HOST=" "$env_file" | cut -d= -f2 | sed 's/"//g')
        print_info "从.env文件读取外部主机: $detected_host"
    elif [[ -f "$SCRIPT_DIR/scripts/detect-external-host.sh" ]]; then
        detected_host=$(cd "$SCRIPT_DIR" && bash scripts/detect-external-host.sh | grep "检测到的主机地址:" | cut -d: -f2 | xargs)
        if [[ -n "$detected_host" && "$detected_host" != "localhost" ]]; then
            print_info "自动检测到外部主机: $detected_host"
        else
            detected_host="localhost"
        fi
    fi
    
    # 从 .env.prod 文件加载变量并进行动态替换
    if [[ -f "$env_file" ]]; then
        while IFS='=' read -r key value; do
            # 跳过注释和空行
            [[ "$key" =~ ^[[:space:]]*# ]] && continue
            [[ -z "$key" ]] && continue
            
            # 移除引号
            value=${value#\"}
            value=${value%\"}
            value=${value#\'}
            value=${value%\'}
            
            # 动态替换变量中的占位符
            value=${value//\$\{EXTERNAL_HOST\}/$detected_host}
            value=${value//\$\{EXTERNAL_PORT\}/8080}
            value=${value//\$\{EXTERNAL_SCHEME\}/http}
            
            eval "ENV_${key}=\"$value\""
        done < "$env_file"
    fi
    
    # 设置动态变量并导出
    export EXTERNAL_HOST="${ENV_EXTERNAL_HOST:-$detected_host}"
    export EXTERNAL_PORT="${ENV_EXTERNAL_PORT:-8080}"
    export EXTERNAL_SCHEME="${ENV_EXTERNAL_SCHEME:-http}"
    
    # 从 docker-compose.yml 提取默认值
    if [[ -f "$SCRIPT_DIR/docker-compose.yml" ]]; then
        # 提取环境变量默认值
        export BACKEND_HOST="${ENV_BACKEND_HOST:-backend}"
        export BACKEND_PORT="${ENV_BACKEND_PORT:-8082}"
        export FRONTEND_HOST="${ENV_FRONTEND_HOST:-frontend}"
        export FRONTEND_PORT="${ENV_FRONTEND_PORT:-80}"
        export JUPYTERHUB_HOST="${ENV_JUPYTERHUB_HOST:-jupyterhub}"
        export JUPYTERHUB_PORT="${ENV_JUPYTERHUB_PORT:-8000}"
        export EXTERNAL_SCHEME="${ENV_EXTERNAL_SCHEME:-http}"
        export EXTERNAL_HOST="${ENV_EXTERNAL_HOST:-$detected_host}"
        export EXTERNAL_PORT="${ENV_EXTERNAL_PORT:-8080}"
        export GITEA_ALIAS_ADMIN_TO="${ENV_GITEA_ALIAS_ADMIN_TO:-admin}"
        export GITEA_ADMIN_EMAIL="${ENV_GITEA_ADMIN_EMAIL:-admin@example.com}"
    fi
}

# 渲染模板文件（纯 Bash 实现，兼容 macOS 和 Linux）
render_template() {
    local template_file="$1"
    local output_file="$2"
    
    if [[ ! -f "$template_file" ]]; then
        print_error "模板文件不存在: $template_file"
        return 1
    fi
    
    print_info "渲染模板: $template_file -> $output_file"
    
    # 创建输出目录
    local output_dir
    output_dir=$(dirname "$output_file")
    mkdir -p "$output_dir"
    
    # 读取模板内容
    local template_content
    template_content=$(<"$template_file")
    
    # 使用纯 Bash 进行变量替换
    # 支持 ${VAR} 和 {{VAR}} 格式，但保留 Nginx 变量（小写的 $var）
    local result="$template_content"
    
    # 定义需要替换的变量列表（大写变量名）
    local vars_to_replace=(
        "EXTERNAL_HOST"
        "EXTERNAL_PORT"
        "EXTERNAL_SCHEME"
        "BACKEND_HOST"
        "BACKEND_PORT"
        "FRONTEND_HOST"
        "FRONTEND_PORT"
        "JUPYTERHUB_HOST"
        "JUPYTERHUB_PORT"
        "GITEA_ALIAS_ADMIN_TO"
        "GITEA_ADMIN_EMAIL"
        "ENVIRONMENT"
        "AUTH_TYPE"
        "GENERATION_TIME"
        "JUPYTERHUB_HUB_PORT"
        "JUPYTERHUB_BASE_URL"
        "JUPYTERHUB_HUB_CONNECT_HOST"
        "JUPYTERHUB_PUBLIC_URL"
        "CONFIGPROXY_AUTH_TOKEN"
        "JUPYTERHUB_DB_URL"
        "JUPYTERHUB_LOG_LEVEL"
        "SESSION_TIMEOUT_DAYS"
        "SINGLEUSER_IMAGE"
        "DOCKER_NETWORK"
        "JUPYTERHUB_MEM_LIMIT"
        "JUPYTERHUB_CPU_LIMIT"
        "JUPYTERHUB_MEM_GUARANTEE"
        "JUPYTERHUB_CPU_GUARANTEE"
        "USER_STORAGE_CAPACITY"
        "JUPYTERHUB_STORAGE_CLASS"
        "SHARED_STORAGE_PATH"
        "AI_INFRA_BACKEND_URL"
        "KUBERNETES_NAMESPACE"
        "KUBERNETES_SERVICE_ACCOUNT"
        "JUPYTERHUB_START_TIMEOUT"
        "JUPYTERHUB_HTTP_TIMEOUT"
        "JWT_SECRET"
        "JUPYTERHUB_AUTO_LOGIN"
        "AUTH_REFRESH_AGE"
        "ADMIN_USERS"
        "AUTH_CONFIG"
        "SPAWNER_CONFIG"
        "SHARED_STORAGE_CONFIG"
        "ADDITIONAL_CONFIG"
    )
    
    # 定义可选变量（允许为空，会被替换为空字符串）
    local optional_vars=(
        "ADDITIONAL_CONFIG"
        "SHARED_STORAGE_CONFIG"
        "GENERATION_TIME"
    )
    
    # 对每个变量进行替换
    for var_name in "${vars_to_replace[@]}"; do
        # 获取变量值
        local var_value="${!var_name:-}"
        
        # 检查是否为可选变量
        local is_optional=false
        for opt_var in "${optional_vars[@]}"; do
            if [[ "$var_name" == "$opt_var" ]]; then
                is_optional=true
                break
            fi
        done
        
        # 如果变量为空且不是可选变量，跳过替换（保留模板中的占位符）
        if [[ -z "$var_value" ]] && [[ "$is_optional" == "false" ]]; then
            continue
        fi
        
        # 使用 Perl 进行替换（支持多行内容，兼容 macOS 和 Linux）
        # Perl 的 s/// 操作符可以正确处理包含换行符的替换内容
        if command -v perl >/dev/null 2>&1; then
            # 转义特殊字符用于 Perl 正则表达式
            local escaped_var_name
            escaped_var_name=$(printf '%s' "$var_name" | perl -pe 's/([\$\{\}\[\]\(\)\.\*\+\?\^\|\\])/\\$1/g')
            
            # 使用 Perl 的 quotemeta 函数自动转义替换内容
            # -0777 让 Perl 读取整个文件为一个字符串（支持多行匹配）
            result=$(printf '%s' "$result" | perl -0777 -pe "
                my \$val = q($var_value);
                s/\\\$\{$escaped_var_name\}/\$val/g;
                s/\{\{$escaped_var_name\}\}/\$val/g;
            ")
        else
            # 降级到 awk（更通用，但速度较慢）
            # 临时文件方案，避免 shell 转义问题
            local tmp_val_file
            tmp_val_file=$(mktemp)
            printf '%s' "$var_value" > "$tmp_val_file"
            
            result=$(awk -v var_name="$var_name" -v val_file="$tmp_val_file" '
                BEGIN {
                    # 读取替换值
                    while ((getline line < val_file) > 0) {
                        if (val != "") val = val "\n"
                        val = val line
                    }
                    close(val_file)
                }
                {
                    # 替换 ${VAR} 格式
                    gsub("\\$\\{" var_name "\\}", val)
                    # 替换 {{VAR}} 格式
                    gsub("\\{\\{" var_name "\\}\\}", val)
                    print
                }
            ' <<< "$result")
            
            rm -f "$tmp_val_file"
        fi
    done
    
    # 写入输出文件
    echo "$result" > "$output_file"
    
    if [[ $? -eq 0 ]]; then
        print_success "✓ 模板渲染完成: $output_file"
        return 0
    else
        print_error "模板渲染失败: $output_file"
        return 1
    fi
}

# 渲染所有nginx模板
render_nginx_templates() {
    print_info "===========================================" 
    print_info "渲染 Nginx 配置模板"
    print_info "==========================================="
    
    # 加载环境变量
    load_environment_variables
    
    local template_dir="$SCRIPT_DIR/src/nginx/templates"
    local output_dir="$SCRIPT_DIR/src/nginx"
    
    if [[ ! -d "$template_dir" ]]; then
        print_error "模板目录不存在: $template_dir"
        return 1
    fi
    
    # 渲染主配置文件
    render_template "$template_dir/conf.d/server-main.conf.tpl" "$output_dir/conf.d/server-main.conf"
    
    # 渲染includes配置文件  
    render_template "$template_dir/conf.d/includes/gitea.conf.tpl" "$output_dir/conf.d/includes/gitea.conf"
    render_template "$template_dir/conf.d/includes/jupyterhub.conf.tpl" "$output_dir/conf.d/includes/jupyterhub.conf"
    render_template "$template_dir/conf.d/includes/minio.conf.tpl" "$output_dir/conf.d/includes/minio.conf"
    
    print_success "✓ Nginx 模板渲染完成"
    echo
}

# 渲染JupyterHub配置模板
render_jupyterhub_templates() {
    print_info "===========================================" 
    print_info "渲染 JupyterHub 配置模板"
    print_info "==========================================="
    
    # 加载环境变量
    load_environment_variables
    
    local template_dir="$SCRIPT_DIR/src/jupyterhub/templates"
    local output_dir="$SCRIPT_DIR/src/jupyterhub"
    
    if [[ ! -d "$template_dir" ]]; then
        print_error "JupyterHub模板目录不存在: $template_dir"
        return 1
    fi
    
    # 设置JupyterHub特定的环境变量
    setup_jupyterhub_variables
    
    # 读取和渲染子模板内容
    local auth_config=""
    local spawner_config=""
    local shared_storage_config=""
    local additional_config=""
    
    # 根据环境和配置选择认证方式
    if [[ "${USE_CUSTOM_AUTH:-false}" == "true" ]]; then
        if [[ -f "$template_dir/auth_backend.py.tpl" ]]; then
            # 先渲染认证模板到临时文件，再读取内容
            local temp_auth_file="$output_dir/.temp_auth_config.py"
            render_template "$template_dir/auth_backend.py.tpl" "$temp_auth_file"
            if [[ -f "$temp_auth_file" ]]; then
                auth_config=$(<"$temp_auth_file")
                rm -f "$temp_auth_file"
            fi
        fi
    else
        if [[ -f "$template_dir/auth_local.py.tpl" ]]; then
            # 先渲染认证模板到临时文件，再读取内容
            local temp_auth_file="$output_dir/.temp_auth_config.py"
            render_template "$template_dir/auth_local.py.tpl" "$temp_auth_file"
            if [[ -f "$temp_auth_file" ]]; then
                auth_config=$(<"$temp_auth_file")
                rm -f "$temp_auth_file"
            fi
        fi
    fi
    
    # 根据环境选择Spawner配置
    if [[ "${ENVIRONMENT:-development}" == "production" || "${JUPYTERHUB_SPAWNER:-docker}" == "kubernetes" ]]; then
        if [[ -f "$template_dir/spawner_kubernetes.py.tpl" ]]; then
            # 先渲染Spawner模板到临时文件，再读取内容
            local temp_spawner_file="$output_dir/.temp_spawner_config.py"
            render_template "$template_dir/spawner_kubernetes.py.tpl" "$temp_spawner_file"
            if [[ -f "$temp_spawner_file" ]]; then
                spawner_config=$(<"$temp_spawner_file")
                rm -f "$temp_spawner_file"
            fi
            
            # 处理共享存储配置
            if [[ -f "$template_dir/shared_storage_k8s.py.tpl" ]]; then
                local temp_storage_file="$output_dir/.temp_storage_config.py"
                render_template "$template_dir/shared_storage_k8s.py.tpl" "$temp_storage_file"
                if [[ -f "$temp_storage_file" ]]; then
                    shared_storage_config=$(<"$temp_storage_file")
                    rm -f "$temp_storage_file"
                fi
            fi
        fi
    else
        if [[ -f "$template_dir/spawner_docker.py.tpl" ]]; then
            # 先渲染Spawner模板到临时文件，再读取内容
            local temp_spawner_file="$output_dir/.temp_spawner_config.py"
            render_template "$template_dir/spawner_docker.py.tpl" "$temp_spawner_file"
            if [[ -f "$temp_spawner_file" ]]; then
                spawner_config=$(<"$temp_spawner_file")
                rm -f "$temp_spawner_file"
            fi
        fi
    fi
    
    
    # 设置模板变量环境变量
    export GENERATION_TIME=$(date '+%Y-%m-%d %H:%M:%S %Z')
    export AUTH_CONFIG="$auth_config"
    export SPAWNER_CONFIG="$spawner_config"
    export SHARED_STORAGE_CONFIG="$shared_storage_config"
    export ADDITIONAL_CONFIG="$additional_config"
    
    # 渲染主配置文件
    if [[ -f "$template_dir/jupyterhub_config.py.tpl" ]]; then
        render_template "$template_dir/jupyterhub_config.py.tpl" "$output_dir/jupyterhub_config_generated.py"
    fi
    
    # 生成不同环境的配置文件
    ENVIRONMENT="development" AUTH_TYPE="local" render_template "$template_dir/jupyterhub_config.py.tpl" "$output_dir/jupyterhub_config_development_generated.py"
    ENVIRONMENT="production" AUTH_TYPE="backend" USE_CUSTOM_AUTH="true" JUPYTERHUB_SPAWNER="kubernetes" render_template "$template_dir/jupyterhub_config.py.tpl" "$output_dir/jupyterhub_config_production_generated.py"
    
    print_success "✓ JupyterHub 模板渲染完成"
    echo
}

# 复制Slurm包到apphub
copy_slurm_packages_to_apphub() {
    local tag="${1:-$DEFAULT_IMAGE_TAG}"
    
    print_info "==========================================="
    print_info "复制 Slurm 包到 apphub"
    print_info "==========================================="
    
    local apphub_container="ai-infra-apphub-temp"
    local apphub_image="ai-infra-apphub:$tag"
    local slurm_container="ai-infra-slurm-build-temp"
    local slurm_image="ai-infra-slurm-build:$tag"
    
    # 检查镜像是否存在
    if ! docker image inspect "$slurm_image" >/dev/null 2>&1; then
        print_error "Slurm构建镜像不存在: $slurm_image"
        return 1
    fi
    
    if ! docker image inspect "$apphub_image" >/dev/null 2>&1; then
        print_error "Apphub镜像不存在: $apphub_image"
        return 1
    fi
    
    # 创建临时容器来提取deb文件
    print_info "创建临时Slurm容器提取deb文件..."
    if ! docker create --name "$slurm_container" "$slurm_image" >/dev/null; then
        print_error "创建Slurm临时容器失败"
        return 1
    fi
    
    # 创建临时apphub容器准备接收文件
    print_info "创建临时apphub容器..."
    if ! docker create --name "$apphub_container" "$apphub_image" >/dev/null; then
        print_error "创建apphub临时容器失败"
        docker rm -f "$slurm_container" >/dev/null 2>&1 || true
        return 1
    fi
    
    # 从slurm容器复制deb文件到apphub容器
    print_info "复制deb文件到apphub..."
    local success=true
    local deb_copied=false

    # 启动apphub容器以便执行命令
    if [[ "$success" == "true" ]]; then
        print_info "启动apphub容器..."
        if docker start "$apphub_container" >/dev/null 2>&1; then
            print_info "✓ apphub容器启动成功"
        else
            print_error "启动apphub临时容器失败"
            success=false
        fi
    fi

    if [[ "$success" == "true" ]]; then
        if ! docker exec "$apphub_container" sh -c 'mkdir -p /usr/share/nginx/html/pkgs/slurm-deb'; then
            print_error "创建Slurm deb目录失败"
            success=false
        else
            docker exec "$apphub_container" sh -c 'rm -f /usr/share/nginx/html/pkgs/slurm-deb/*.deb 2>/dev/null || true' >/dev/null 2>&1 || true
        fi
    fi

    if [[ "$success" == "true" ]]; then
        # Docker不支持容器间直接复制，需要通过临时目录中转
        local temp_dir="/tmp/slurm-deb-temp-$$"
        mkdir -p "$temp_dir"
        
        # 步骤1: 从slurm容器复制到本地临时目录
        if docker cp "$slurm_container:/out/." "$temp_dir/" 2>/dev/null; then
            # 步骤2: 从本地临时目录复制到apphub容器
            if docker cp "$temp_dir/." "$apphub_container:/usr/share/nginx/html/pkgs/slurm-deb/" 2>/dev/null; then
                # 步骤3: 验证文件是否成功复制
                if docker exec "$apphub_container" sh -c 'ls /usr/share/nginx/html/pkgs/slurm-deb/*.deb >/dev/null 2>&1'; then
                    # 清理非deb文件
                    docker exec "$apphub_container" sh -c 'find /usr/share/nginx/html/pkgs/slurm-deb -maxdepth 1 -type f ! -name "*.deb" -delete' >/dev/null 2>&1 || true
                    
                    # 统计deb文件数量
                    local deb_count=$(docker exec "$apphub_container" sh -c 'ls /usr/share/nginx/html/pkgs/slurm-deb/*.deb 2>/dev/null | wc -l')
                    print_info "✓ 复制Slurm deb文件成功 (共 ${deb_count} 个)"
                    deb_copied=true
                else
                    print_warning "复制完成但未找到任何Slurm deb文件"
                    success=false
                fi
            else
                print_error "从临时目录复制到apphub容器失败"
                success=false
            fi
        else
            print_warning "从slurm容器复制deb文件失败"
            success=false
        fi
        
        # 清理临时目录
        rm -rf "$temp_dir"
    fi

    if [[ "$success" == "true" && "$deb_copied" == "true" ]]; then
        print_info "重新生成deb包索引..."
        if docker exec "$apphub_container" /entrypoint.sh regenerate-index; then
            print_info "✓ deb包索引更新成功"
        else
            print_warning "deb包索引更新失败"
            success=false
        fi
    fi

    # 停止apphub容器，准备提交
    docker stop "$apphub_container" >/dev/null 2>&1 || true

    if [[ "$success" == "true" && "$deb_copied" == "true" ]]; then
        print_info "提交更新后的apphub镜像..."
        local new_apphub_image="ai-infra-apphub:$tag"
        if docker commit "$apphub_container" "$new_apphub_image" >/dev/null; then
            print_success "✓ apphub镜像更新成功: $new_apphub_image"
        else
            print_error "apphub镜像提交失败"
            success=false
        fi
    else
        print_warning "跳过apphub镜像更新（未成功复制Slurm deb包）"
    fi

    # 清理临时容器
    print_info "清理临时容器..."
    docker rm -f "$slurm_container" >/dev/null 2>&1 || true
    docker rm -f "$apphub_container" >/dev/null 2>&1 || true

    if [[ "$success" == "true" && "$deb_copied" == "true" ]]; then
        print_success "✓ Slurm包复制到apphub完成"
        return 0
    else
        print_warning "Slurm包复制过程有问题，但不影响构建流程"
        return 1
    fi
}

# 渲染Docker Compose配置模板
render_docker_compose_templates() {
    # 处理帮助参数
    if [[ "$1" == "--help" || "$1" == "-h" ]]; then
        echo "render-docker-compose-templates - 渲染Docker Compose配置"
        echo
        echo "用法: $0 render-templates docker-compose [registry] [tag] [--oceanbase-init-dir <path>]"
        echo
        echo "参数:"
        echo "  registry    私有仓库地址 (可选，默认不替换为内部镜像)"
        echo "  tag         镜像标签 (可选，默认: $DEFAULT_IMAGE_TAG)"
        echo "  --oceanbase-init-dir, -O  指定宿主机上的 OceanBase 初始化脚本目录 (可选)"
        echo
        echo "说明:"
        echo "  从 docker-compose.yml.example 生成 docker-compose.yml"
        echo "  如果指定了 registry，将替换所有镜像为内部仓库版本"
        echo "  如果指定了 --oceanbase-init-dir，将把该路径写入 .env 文件中的 OCEANBASE_INIT_DIR 变量"
        echo
        echo "示例:"
        echo "  $0 render-templates docker-compose                                         # 基础渲染"
        echo "  $0 render-templates docker-compose aiharbor.msxf.local/aihpc v1.0.0       # 替换为内部镜像"
        echo "  $0 render-templates docker-compose --oceanbase-init-dir ./data/ob/init.d  # 指定OceanBase初始化目录"
        echo "  $0 render-templates docker-compose --openscow-db-dir ./data/openscow/mysql # 指定OpenSCOW MySQL数据目录"
        return 0
    fi

    print_info "===========================================" 
    print_info "渲染 Docker Compose 配置模板"
    print_info "==========================================="
    
    local registry=""
    local tag="$DEFAULT_IMAGE_TAG"
    local oceanbase_init_dir=""
    local openscow_db_dir=""

    # 简单参数解析：支持位置参数 (registry, tag) 和 --oceanbase-init-dir/-O
    # 收集非选项参数
    local positional=()
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --oceanbase-init-dir|-O)
                oceanbase_init_dir="$2"
                shift 2
                ;;
            --openscow-db-dir)
                openscow_db_dir="$2"
                shift 2
                ;;
            --)
                shift; break ;;
            -* )
                # 未知选项，跳过
                shift ;;
            * )
                positional+=("$1"); shift ;;
        esac
    done
    # 剩余的都作为位置参数追加
    while [[ $# -gt 0 ]]; do positional+=("$1"); shift; done
    # 解析位置参数：最多两个
    if [[ ${#positional[@]} -ge 1 ]]; then registry="${positional[0]}"; fi
    if [[ ${#positional[@]} -ge 2 ]]; then tag="${positional[1]}"; fi
    
    # 加载环境变量
    load_environment_variables
    
    local template_file="$SCRIPT_DIR/docker-compose.yml.example"
    local output_file="$SCRIPT_DIR/docker-compose.yml"
    
    if [[ ! -f "$template_file" ]]; then
        print_error "Docker Compose模板文件不存在: $template_file"
        return 1
    fi
    
    print_info "从模板生成 docker-compose.yml"
    print_info "模板文件: $template_file"
    print_info "输出文件: $output_file"
    if [[ -n "$registry" ]]; then
        print_info "内部镜像仓库: $registry"
        print_info "镜像标签: $tag"
    fi
    if [[ -n "$oceanbase_init_dir" ]]; then
        print_info "OceanBase 初始化目录: $oceanbase_init_dir"
    fi
    if [[ -n "$openscow_db_dir" ]]; then
        print_info "OpenSCOW MySQL 数据目录: $openscow_db_dir"
    fi
    
    # 创建备份
    if [[ -f "$output_file" ]]; then
        print_info "备份现有文件: ${output_file}.backup"
        cp "$output_file" "${output_file}.backup"
    fi
    
    # 设置默认的Kafka环境变量
    export KAFKA_ENABLED="${KAFKA_ENABLED:-true}"
    export KAFKA_EXTERNAL_PORT="${KAFKA_EXTERNAL_PORT:-9094}"
    export KAFKA_UI_PORT="${KAFKA_UI_PORT:-9095}"
    export KAFKA_RETENTION_HOURS="${KAFKA_RETENTION_HOURS:-168}"
    export KAFKA_CLUSTER_ID="${KAFKA_CLUSTER_ID:-gYf__u4_TgSoREBUnP-YzQ}"
    
    # 如果指定了 OceanBase 初始化目录，则写入 .env 以便 compose 中的占位符能取到值
    if [[ -n "$oceanbase_init_dir" ]]; then
        set_or_update_env_var "OCEANBASE_INIT_DIR" "$oceanbase_init_dir" "$SCRIPT_DIR/.env"
        print_success "✓ 已更新 .env: OCEANBASE_INIT_DIR=$oceanbase_init_dir"
    fi

    # 如果指定了 OpenSCOW MySQL 数据目录，则写入 .env
    if [[ -n "$openscow_db_dir" ]]; then
        set_or_update_env_var "OPENSCOW_DB_DIR" "$openscow_db_dir" "$SCRIPT_DIR/.env"
        print_success "✓ 已更新 .env: OPENSCOW_DB_DIR=$openscow_db_dir"
    fi

    # 复制模板文件到目标位置
    cp "$template_file" "$output_file"

    # 规范化缩进：修复 env_file 列表项缩进（部分模板中写成与键同缩进，导致 YAML 解析错误）
    # 规则：将形如
    #   env_file:\n    - .env
    # 修正为
    #   env_file:\n      - .env
    # 仅对下一行与 env_file: 同缩进且以 "-" 开头的情况做 2 空格缩进调整
    if command -v python3 >/dev/null 2>&1; then
        print_info "修正 docker-compose.yml 中 env_file 列表缩进..."
        python3 - << 'PY' 2>/dev/null || true
from pathlib import Path
import re

output_path = Path(r"$output_file")
text = output_path.read_text(encoding='utf-8')
lines = text.splitlines()

out = []
i = 0
while i < len(lines):
    line = lines[i]
    out.append(line)
    m = re.match(r'^(\s*)env_file:\s*$', line)
    if m and i + 1 < len(lines):
        indent = m.group(1)
        nxt = lines[i + 1]
        # 如果下一行与 env_file: 同缩进且是列表项，则补齐两个空格缩进
        if re.match(r'^' + re.escape(indent) + r'-\s', nxt):
            out.append(indent + '  ' + nxt[len(indent):])
            i += 2
            continue
    i += 1

output_path.write_text("\n".join(out) + ("\n" if text.endswith("\n") else ""), encoding='utf-8')
PY
    else
        print_warning "未检测到 python3，跳过 env_file 缩进修复，若解析失败请手动调整模板缩进"
    fi

    # 兼容性修复：如果模板/旧版本里仍有 openscow_db_data 命名卷引用，替换为绑定挂载变量
    if grep -q "openscow_db_data:/var/lib/mysql" "$output_file" 2>/dev/null; then
        sed_inplace "s|openscow_db_data:/var/lib/mysql|\${OPENSCOW_DB_DIR:-./data/openscow/mysql}:/var/lib/mysql|g" "$output_file"
        print_info "已将 legacy openscow_db_data 命名卷替换为绑定挂载"
    fi
    
    # 如果指定了registry，进行镜像替换
    if [[ -n "$registry" ]]; then
        print_info "替换镜像为内部仓库版本..."
        local backup_file=$(replace_images_in_compose_file "$output_file" "$registry" "$tag")
        if [[ -n "$backup_file" ]]; then
            print_success "✓ 镜像替换完成，备份文件: $backup_file"
        else
            print_info "未进行镜像替换"
        fi
    fi
    
    print_success "✓ Docker Compose 模板渲染完成"
    print_info "生成的文件: $output_file"
    
    # 验证生成的文件
    if docker compose -f "$output_file" config --quiet 2>/dev/null; then
        print_success "✓ Docker Compose 配置语法验证通过"
    else
        print_warning "⚠ Docker Compose 配置语法验证失败，请检查生成的文件"
    fi
    echo
}

# 同步 .env 和 .env.example 文件
sync_env_files() {
    print_info "==========================================="
    print_info "同步环境变量配置文件"
    print_info "==========================================="
    
    local env_file="$SCRIPT_DIR/.env"
    local env_example_file="$SCRIPT_DIR/.env.example"
    
    if [[ ! -f "$env_file" ]]; then
        print_error ".env 文件不存在: $env_file"
        return 1
    fi
    
    # 创建备份
    if [[ -f "$env_example_file" ]]; then
        local backup_name=".env.example.backup-$(date +%Y%m%d-%H%M%S)"
        cp "$env_example_file" "$SCRIPT_DIR/$backup_name"
        print_info "已备份现有 .env.example: $backup_name"
    fi
    
    print_info "从 .env 同步配置到 .env.example"
    print_info "源文件: $env_file"
    print_info "目标文件: $env_example_file"
    
    # 复制 .env 到 .env.example，并替换敏感值为占位符
    cp "$env_file" "$env_example_file"
    
    # 替换敏感信息为占位符变量
    sed_inplace 's/^EXTERNAL_HOST=.*/EXTERNAL_HOST=${EXTERNAL_HOST}/' "$env_example_file"
    sed_inplace 's/^DOMAIN=.*/DOMAIN=${EXTERNAL_HOST}/' "$env_example_file"
    sed_inplace 's/^EXTERNAL_PORT=.*/EXTERNAL_PORT=${EXTERNAL_PORT}/' "$env_example_file"
    sed_inplace 's/^EXTERNAL_SCHEME=.*/EXTERNAL_SCHEME=${EXTERNAL_SCHEME}/' "$env_example_file"
    sed_inplace 's/^NGINX_PORT=.*/NGINX_PORT=${EXTERNAL_PORT}/' "$env_example_file"
    # 强制对依赖 EXTERNAL_* 的 URL 使用占位符，避免写死 IP/端口
    sed_inplace 's#^MINIO_CONSOLE_URL=.*#MINIO_CONSOLE_URL=${EXTERNAL_SCHEME}://${EXTERNAL_HOST}:${EXTERNAL_PORT}/minio-console/#' "$env_example_file"
    sed_inplace 's/^JUPYTERHUB_EXTERNAL_PORT=.*/JUPYTERHUB_EXTERNAL_PORT=${JUPYTERHUB_PORT}/' "$env_example_file"
    sed_inplace 's/^GITEA_EXTERNAL_PORT=.*/GITEA_EXTERNAL_PORT=${GITEA_PORT}/' "$env_example_file"
    sed_inplace 's/^APPHUB_PORT=.*/APPHUB_PORT=${APPHUB_PORT}/' "$env_example_file"
    sed_inplace 's/^HTTPS_PORT=.*/HTTPS_PORT=${HTTPS_PORT}/' "$env_example_file"
    sed_inplace 's/^DEBUG_PORT=.*/DEBUG_PORT=${DEBUG_PORT}/' "$env_example_file"
    
    # 替换动态生成的 URL 配置为占位符
    sed_inplace 's/^JUPYTERHUB_PUBLIC_HOST=.*/JUPYTERHUB_PUBLIC_HOST=${EXTERNAL_HOST}:${EXTERNAL_PORT}/' "$env_example_file"
    sed_inplace 's|^JUPYTERHUB_BASE_URL=.*|JUPYTERHUB_BASE_URL=${EXTERNAL_SCHEME}://${EXTERNAL_HOST}:${EXTERNAL_PORT}/jupyter/|' "$env_example_file"
    sed_inplace 's|^JUPYTERHUB_CORS_ORIGIN=.*|JUPYTERHUB_CORS_ORIGIN=${EXTERNAL_SCHEME}://${EXTERNAL_HOST}:${EXTERNAL_PORT}|' "$env_example_file"
    sed_inplace 's|^ROOT_URL=.*|ROOT_URL=${EXTERNAL_SCHEME}://${EXTERNAL_HOST}:${EXTERNAL_PORT}/gitea/|' "$env_example_file"
    
    cleanup_backup_files "$SCRIPT_DIR"
    
    print_success "✓ 环境变量文件同步完成"
    print_info "已将 .env 中的配置同步到 .env.example，并将动态值替换为占位符变量"
    echo
}

# 同步所有配置文件
sync_all_configs() {
    local force_mode="${1:-false}"
    
    print_info "==========================================="
    print_info "同步所有配置文件"
    print_info "==========================================="
    
    # 1. 同步环境变量文件
    sync_env_files

    # 1.1 确保 .env 中具备必要的 MinIO 变量（为现有项目追加默认值）
    if [[ -f "$SCRIPT_DIR/.env" ]]; then
        set_or_update_env_var "MINIO_REGION" "${MINIO_REGION:-us-east-1}" "$SCRIPT_DIR/.env"
        set_or_update_env_var "MINIO_USE_SSL" "${MINIO_USE_SSL:-false}" "$SCRIPT_DIR/.env"
    fi
    
    # 2. 验证 docker-compose.yml 和 docker-compose.yml.example 是否同步
    local compose_file="$SCRIPT_DIR/docker-compose.yml"
    local compose_example_file="$SCRIPT_DIR/docker-compose.yml.example"
    
    if [[ -f "$compose_file" ]] && [[ -f "$compose_example_file" ]]; then
        # 比较两个文件的内容（忽略注释和空行）
        local compose_content=$(grep -v '^[[:space:]]*#' "$compose_file" | grep -v '^[[:space:]]*$' | sort)
        local example_content=$(grep -v '^[[:space:]]*#' "$compose_example_file" | grep -v '^[[:space:]]*$' | sort)
        
        if [[ "$compose_content" == "$example_content" ]]; then
            print_success "✓ docker-compose.yml 和 docker-compose.yml.example 已同步"
        else
            print_warning "⚠ docker-compose.yml 与模板不同步（以 docker-compose.yml.example 为准）"
            print_info "提示：请运行 ./build.sh render-templates docker-compose 以模板为源重新渲染 docker-compose.yml"
        fi
    else
        print_warning "⚠ docker-compose 文件缺失，建议运行模板渲染"
    fi
    
    # 3. 检查配置文件的一致性
    print_info "检查配置文件一致性..."
    
    local issues_found=0
    
    # 检查环境变量是否在两个文件中都存在
    if [[ -f "$SCRIPT_DIR/.env" ]] && [[ -f "$SCRIPT_DIR/.env.example" ]]; then
        local env_vars=$(grep -E '^[A-Z_]+=.*' "$SCRIPT_DIR/.env" | cut -d'=' -f1 | sort)
        local example_vars=$(grep -E '^[A-Z_]+=.*' "$SCRIPT_DIR/.env.example" | cut -d'=' -f1 | sort)
        
        # 检查 .env 中的变量是否都在 .env.example 中
        local missing_in_example=$(comm -23 <(echo "$env_vars") <(echo "$example_vars"))
        if [[ -n "$missing_in_example" ]]; then
            print_warning "⚠ 以下变量在 .env 中存在但在 .env.example 中缺失:"
            echo "$missing_in_example" | sed 's/^/    /'
            ((issues_found++))
        fi
        
        # 检查 .env.example 中的变量是否都在 .env 中
        local missing_in_env=$(comm -13 <(echo "$env_vars") <(echo "$example_vars"))
        if [[ -n "$missing_in_env" ]]; then
            print_warning "⚠ 以下变量在 .env.example 中存在但在 .env 中缺失:"
            echo "$missing_in_env" | sed 's/^/    /'
            ((issues_found++))
        fi
    fi
    
    if [[ $issues_found -eq 0 ]]; then
        print_success "✓ 配置文件一致性检查通过"
    else
        print_warning "⚠ 发现 $issues_found 个配置不一致问题"
        print_info "建议手动检查并修复上述问题"
    fi
    
    print_success "✓ 配置文件同步检查完成"
    echo
}

# 设置JupyterHub特定变量
setup_jupyterhub_variables() {
    # 从环境变量或.env文件中读取JupyterHub配置
    ENVIRONMENT="${ENVIRONMENT:-${ENV_ENVIRONMENT:-development}}"
    AUTH_TYPE="${AUTH_TYPE:-${ENV_AUTH_TYPE:-local}}"
    JUPYTERHUB_HUB_PORT="${JUPYTERHUB_HUB_PORT:-${ENV_JUPYTERHUB_HUB_PORT:-8081}}"
    
    # 从.env读取完整URL，然后提取路径部分用于配置
    local base_url_from_env="${JUPYTERHUB_BASE_URL:-${ENV_JUPYTERHUB_BASE_URL:-/jupyter/}}"
    # 如果是完整URL（包含http://或https://），提取路径部分
    if [[ "$base_url_from_env" =~ ^https?:// ]]; then
        # 提取URL的路径部分（从第三个/开始）
        JUPYTERHUB_BASE_URL=$(echo "$base_url_from_env" | sed -E 's|^https?://[^/]+||')
    else
        JUPYTERHUB_BASE_URL="$base_url_from_env"
    fi
    # 确保路径以/结尾
    [[ "$JUPYTERHUB_BASE_URL" != */ ]] && JUPYTERHUB_BASE_URL="${JUPYTERHUB_BASE_URL}/"
    
    JUPYTERHUB_HUB_CONNECT_HOST="${JUPYTERHUB_HUB_CONNECT_HOST:-${ENV_JUPYTERHUB_HUB_CONNECT_HOST:-jupyterhub}}"
    
    # 处理JUPYTERHUB_PUBLIC_URL，保持完整URL格式
    local public_url_from_env="${JUPYTERHUB_PUBLIC_URL:-${ENV_JUPYTERHUB_PUBLIC_URL:-http://localhost:8080/jupyter/}}"
    if [[ ! "$public_url_from_env" =~ ^https?:// ]]; then
        # 如果不是完整URL，从EXTERNAL_*变量构建
        JUPYTERHUB_PUBLIC_URL="${EXTERNAL_SCHEME:-http}://${EXTERNAL_HOST:-localhost}:${EXTERNAL_PORT:-8080}${JUPYTERHUB_BASE_URL}"
    else
        JUPYTERHUB_PUBLIC_URL="$public_url_from_env"
    fi
    
    CONFIGPROXY_AUTH_TOKEN="${CONFIGPROXY_AUTH_TOKEN:-${ENV_CONFIGPROXY_AUTH_TOKEN:-}}"
    JUPYTERHUB_DB_URL="${JUPYTERHUB_DB_URL:-${ENV_JUPYTERHUB_DB_URL:-sqlite:///jupyterhub.sqlite}}"
    JUPYTERHUB_LOG_LEVEL="${JUPYTERHUB_LOG_LEVEL:-${ENV_JUPYTERHUB_LOG_LEVEL:-INFO}}"
    SESSION_TIMEOUT_DAYS="${SESSION_TIMEOUT_DAYS:-${ENV_SESSION_TIMEOUT_DAYS:-7}}"
    SINGLEUSER_IMAGE="${SINGLEUSER_IMAGE:-${ENV_SINGLEUSER_IMAGE:-ai-infra-singleuser:latest}}"
    DOCKER_NETWORK="${DOCKER_NETWORK:-${ENV_DOCKER_NETWORK:-ai-infra-matrix_default}}"
    JUPYTERHUB_MEM_LIMIT="${JUPYTERHUB_MEM_LIMIT:-${ENV_JUPYTERHUB_MEM_LIMIT:-2G}}"
    JUPYTERHUB_CPU_LIMIT="${JUPYTERHUB_CPU_LIMIT:-${ENV_JUPYTERHUB_CPU_LIMIT:-1.0}}"
    JUPYTERHUB_MEM_GUARANTEE="${JUPYTERHUB_MEM_GUARANTEE:-${ENV_JUPYTERHUB_MEM_GUARANTEE:-1G}}"
    JUPYTERHUB_CPU_GUARANTEE="${JUPYTERHUB_CPU_GUARANTEE:-${ENV_JUPYTERHUB_CPU_GUARANTEE:-0.5}}"
    USER_STORAGE_CAPACITY="${USER_STORAGE_CAPACITY:-${ENV_USER_STORAGE_CAPACITY:-10Gi}}"
    JUPYTERHUB_STORAGE_CLASS="${JUPYTERHUB_STORAGE_CLASS:-${ENV_JUPYTERHUB_STORAGE_CLASS:-default}}"
    SHARED_STORAGE_PATH="${SHARED_STORAGE_PATH:-${ENV_SHARED_STORAGE_PATH:-/srv/shared-notebooks}}"
    AI_INFRA_BACKEND_URL="${AI_INFRA_BACKEND_URL:-${ENV_AI_INFRA_BACKEND_URL:-http://backend:8082}}"
    KUBERNETES_NAMESPACE="${KUBERNETES_NAMESPACE:-${ENV_KUBERNETES_NAMESPACE:-ai-infra-users}}"
    KUBERNETES_SERVICE_ACCOUNT="${KUBERNETES_SERVICE_ACCOUNT:-${ENV_KUBERNETES_SERVICE_ACCOUNT:-ai-infra-matrix-jupyterhub}}"
    JUPYTERHUB_START_TIMEOUT="${JUPYTERHUB_START_TIMEOUT:-${ENV_JUPYTERHUB_START_TIMEOUT:-300}}"
    JUPYTERHUB_HTTP_TIMEOUT="${JUPYTERHUB_HTTP_TIMEOUT:-${ENV_JUPYTERHUB_HTTP_TIMEOUT:-30}}"
    JWT_SECRET="${JWT_SECRET:-${ENV_JWT_SECRET:-}}"
    JUPYTERHUB_AUTO_LOGIN="${JUPYTERHUB_AUTO_LOGIN:-${ENV_JUPYTERHUB_AUTO_LOGIN:-False}}"
    AUTH_REFRESH_AGE="${AUTH_REFRESH_AGE:-${ENV_AUTH_REFRESH_AGE:-3600}}"
    ADMIN_USERS="${ADMIN_USERS:-${ENV_ADMIN_USERS:-'admin'}}"
    USE_CUSTOM_AUTH="${USE_CUSTOM_AUTH:-${ENV_USE_CUSTOM_AUTH:-false}}"
    JUPYTERHUB_SPAWNER="${JUPYTERHUB_SPAWNER:-${ENV_JUPYTERHUB_SPAWNER:-docker}}"
}

# ==========================================
# ==========================================
# 随机密码生成函数
# ==========================================

# 生成安全的随机密码
generate_random_password() {
    local length="${1:-24}"  # 默认长度24
    local password_type="${2:-standard}"  # standard, hex, alphanumeric
    
    case "$password_type" in
        "hex")
            # 64位十六进制密钥 (用于JupyterHub等需要特定长度的密钥)
            if [[ "$length" == "64" ]]; then
                openssl rand -hex 32
            else
                openssl rand -hex "$((length/2))"
            fi
            ;;
        "alphanumeric")
            # 字母数字组合，避免特殊字符
            LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom | head -c "$length"
            ;;
        "standard"|*)
            # 标准密码：字母、数字、部分安全特殊字符
            LC_ALL=C tr -dc 'A-Za-z0-9._-' < /dev/urandom | head -c "$length"
            ;;
    esac
}

# 生产环境强密码生成器 (集成自 scripts/generate-prod-passwords.sh)
generate_production_passwords() {
    local env_file="${1:-.env.prod}"
    local force="${2:-false}"
    
    print_info "======================================================================"
    print_info "🔧 AI Infrastructure Matrix 生产环境密码生成器"
    print_info "======================================================================"
    print_warning "⚠️  此脚本将生成新的系统服务密码"
    print_warning "⚠️  默认管理员账户 (admin/admin123) 不会被此脚本修改"
    print_warning "⚠️  请在系统部署后通过Web界面修改管理员密码"
    print_info "======================================================================"
    
    # 如果目标环境文件不存在，从 .env.example 复制
    if [[ ! -f "$env_file" ]]; then
        if [[ -f ".env.example" ]]; then
            print_info "环境文件不存在，从 .env.example 创建: $env_file"
            cp ".env.example" "$env_file"
            print_success "✓ 已从 .env.example 创建环境文件: $env_file"
        else
            print_error "环境文件不存在: $env_file"
            print_error "且模板文件 .env.example 也不存在"
            return 1
        fi
    fi
    
    # 创建备份
    local backup_file="${env_file}.backup.$(date +%Y%m%d_%H%M%S)"
    print_info "创建备份: $backup_file"
    cp "$env_file" "$backup_file"
    
    print_info "生成新的强密码..."
    
    # 生成新密码 (使用openssl更安全，确保没有换行符)
    local postgres_password=$(openssl rand -base64 32 | tr -d "=+/\n" | cut -c1-24)
    local redis_password=$(openssl rand -base64 32 | tr -d "=+/\n" | cut -c1-24)
    local jwt_secret=$(openssl rand -base64 64 | tr -d "=+/\n" | cut -c1-48)
    local configproxy_token=$(openssl rand -base64 64 | tr -d "=+/\n" | cut -c1-48)
    local jupyterhub_crypt_key=$(openssl rand -hex 32)
    local minio_access_key=$(openssl rand -base64 32 | tr -d "=+/\n" | cut -c1-20)
    local minio_secret_key=$(openssl rand -base64 64 | tr -d "=+/\n" | cut -c1-40)
    local gitea_admin_password=$(openssl rand -base64 32 | tr -d "=+/\n" | cut -c1-24)
    local gitea_db_passwd=$(openssl rand -base64 32 | tr -d "=+/\n" | cut -c1-24)
    local ldap_admin_password=$(openssl rand -base64 32 | tr -d "=+/\n" | cut -c1-24)
    local ldap_config_password=$(openssl rand -base64 32 | tr -d "=+/\n" | cut -c1-24)
    
    # 使用awk进行安全的替换（避免sed特殊字符问题）
    # 创建临时文件
    local temp_file="${env_file}.updating"
    
    # 使用awk替换，更安全地处理特殊字符
    awk -v pg_pass="$postgres_password" \
        -v redis_pass="$redis_password" \
        -v jwt_sec="$jwt_secret" \
        -v config_token="$configproxy_token" \
        -v hub_key="$jupyterhub_crypt_key" \
        -v minio_access="$minio_access_key" \
        -v minio_secret="$minio_secret_key" \
        -v gitea_admin="$gitea_admin_password" \
        -v gitea_db="$gitea_db_passwd" \
        -v ldap_admin="$ldap_admin_password" \
        -v ldap_config="$ldap_config_password" \
        '
        /^POSTGRES_PASSWORD=/ { print "POSTGRES_PASSWORD=" pg_pass; next }
        /^REDIS_PASSWORD=/ { print "REDIS_PASSWORD=" redis_pass; next }
        /^JWT_SECRET=/ { print "JWT_SECRET=" jwt_sec; next }
        /^CONFIGPROXY_AUTH_TOKEN=/ { print "CONFIGPROXY_AUTH_TOKEN=" config_token; next }
        /^JUPYTERHUB_CRYPT_KEY=/ { print "JUPYTERHUB_CRYPT_KEY=" hub_key; next }
        /^MINIO_ACCESS_KEY=/ { print "MINIO_ACCESS_KEY=" minio_access; next }
        /^MINIO_SECRET_KEY=/ { print "MINIO_SECRET_KEY=" minio_secret; next }
        /^GITEA_ADMIN_PASSWORD=/ { print "GITEA_ADMIN_PASSWORD=" gitea_admin; next }
        /^GITEA_DB_PASSWD=/ { print "GITEA_DB_PASSWD=" gitea_db; next }
        /^LDAP_ADMIN_PASSWORD=/ { print "LDAP_ADMIN_PASSWORD=" ldap_admin; next }
        /^LDAP_CONFIG_PASSWORD=/ { print "LDAP_CONFIG_PASSWORD=" ldap_config; next }
        { print }
        ' "$env_file" > "$temp_file"
    
    # 替换原文件
    mv "$temp_file" "$env_file"
    
    print_success "已生成并应用新的强密码"
    
    print_info "======================================================================"
    print_warning "🔑 重要！默认管理员账户信息："
    echo
    print_success "  用户名: admin"
    print_error "  初始密码: admin123"
    echo
    print_warning "⚠️  请在首次登录后立即更改管理员密码！"
    print_warning "⚠️  管理员密码未通过此脚本更改，需要在系统内修改！"
    print_info "======================================================================"
    
    print_info "系统服务密码信息:"
    echo "POSTGRES_PASSWORD: $postgres_password"
    echo "REDIS_PASSWORD: $redis_password"
    echo "JWT_SECRET: $jwt_secret"
    echo "CONFIGPROXY_AUTH_TOKEN: $configproxy_token"
    echo "JUPYTERHUB_CRYPT_KEY: $jupyterhub_crypt_key"
    echo "MINIO_ACCESS_KEY: $minio_access_key"
    echo "MINIO_SECRET_KEY: $minio_secret_key"
    echo "GITEA_ADMIN_PASSWORD: $gitea_admin_password"
    echo "GITEA_DB_PASSWD: $gitea_db_passwd"
    echo "LDAP_ADMIN_PASSWORD: $ldap_admin_password"
    echo "LDAP_CONFIG_PASSWORD: $ldap_config_password"
    
    print_warning "请妥善保存这些密码信息！"
    print_info "原配置文件已备份至: $backup_file"
    
    return 0
}

# 替换环境文件中的模板密码
replace_template_passwords() {
    local template_file="$1"
    local target_file="$2"
    local force="${3:-false}"
    
    if [[ ! -f "$template_file" ]]; then
        print_error "模板文件不存在: $template_file"
        return 1
    fi
    
    if [[ -f "$target_file" ]] && [[ "$force" != "true" ]]; then
        print_warning "目标文件已存在: $target_file"
        print_info "如需强制覆盖，请使用 --force 参数"
        return 1
    fi
    
    print_info "正在从模板生成环境文件: $target_file"
    
    # 复制模板文件
    cp "$template_file" "$target_file"
    
    # 生成所有需要的密码
    local postgres_password=$(generate_random_password 24 "alphanumeric")
    local redis_password=$(generate_random_password 24 "alphanumeric")
    local jwt_secret=$(generate_random_password 48 "standard")
    local configproxy_token=$(generate_random_password 48 "standard")
    local jupyterhub_crypt_key=$(generate_random_password 64 "hex")
    local minio_access_key=$(generate_random_password 20 "alphanumeric")
    local minio_secret_key=$(generate_random_password 40 "standard")
    local gitea_admin_password=$(generate_random_password 24 "alphanumeric")
    local gitea_db_password=$(generate_random_password 24 "alphanumeric")
    local ldap_admin_password=$(generate_random_password 24 "alphanumeric")
    local ldap_config_password=$(generate_random_password 24 "alphanumeric")
    
    # 替换模板中的密码占位符
    sed -i.bak \
        -e "s/TEMPLATE_POSTGRES_PASSWORD/$postgres_password/g" \
        -e "s/TEMPLATE_REDIS_PASSWORD/$redis_password/g" \
        -e "s/TEMPLATE_JWT_SECRET/$jwt_secret/g" \
        -e "s/TEMPLATE_CONFIGPROXY_AUTH_TOKEN/$configproxy_token/g" \
        -e "s/TEMPLATE_JUPYTERHUB_CRYPT_KEY/$jupyterhub_crypt_key/g" \
        -e "s/TEMPLATE_MINIO_ACCESS_KEY/$minio_access_key/g" \
        -e "s/TEMPLATE_MINIO_SECRET_KEY/$minio_secret_key/g" \
        -e "s/TEMPLATE_GITEA_ADMIN_PASSWORD/$gitea_admin_password/g" \
        -e "s/TEMPLATE_GITEA_DB_PASSWD/$gitea_db_password/g" \
        -e "s/TEMPLATE_LDAP_ADMIN_PASSWORD/$ldap_admin_password/g" \
        -e "s/TEMPLATE_LDAP_CONFIG_PASSWORD/$ldap_config_password/g" \
        "$target_file"
    
    # 处理环境变量展开的URL (替换 ${VARIABLE} 形式)
    # 读取当前文件内容并替换变量引用
    local temp_content=$(cat "$target_file")
    
    # 处理DATABASE_URL
    temp_content=$(echo "$temp_content" | sed "s|\\\${POSTGRES_USER}|postgres|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${POSTGRES_PASSWORD}|$postgres_password|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${POSTGRES_HOST}|postgres|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${POSTGRES_PORT}|5432|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${POSTGRES_DB}|aiinfra|g")
    
    # 处理REDIS_URL
    temp_content=$(echo "$temp_content" | sed "s|\\\${REDIS_PASSWORD}|$redis_password|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${REDIS_HOST}|redis|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${REDIS_PORT}|6379|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${REDIS_DB}|0|g")
    
    # 处理其他服务URL
    temp_content=$(echo "$temp_content" | sed "s|\\\${BACKEND_HOST}|backend|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${BACKEND_PORT}|8082|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${FRONTEND_HOST}|frontend|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${FRONTEND_PORT}|80|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${JUPYTERHUB_HOST}|jupyterhub|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${JUPYTERHUB_PORT}|8000|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${GITEA_HOST}|gitea|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${GITEA_PORT}|3000|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${GITEA_INTERNAL_URL}|http://gitea:3000|g")
    
    # 处理外部访问变量 (动态检测)
    load_environment_variables
    temp_content=$(echo "$temp_content" | sed "s|\\\${EXTERNAL_HOST}|$EXTERNAL_HOST|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${EXTERNAL_PORT}|$EXTERNAL_PORT|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${EXTERNAL_SCHEME}|$EXTERNAL_SCHEME|g")
    
    # 写回文件
    echo "$temp_content" > "$target_file"
    
    # 删除备份文件
    rm -f "${target_file}.bak"
    
    print_success "✓ 生成环境文件完成: $target_file"
    print_info "所有密码已自动生成，请妥善保管！"
    
    return 0
}

# ==========================================
# 环境变量管理函数
# ==========================================

# 生成环境文件从模板
create_env_from_template() {
    # 处理帮助参数
    if [[ "$1" == "--help" || "$1" == "-h" ]]; then
        echo "create-env - 从模板创建环境配置文件"
        echo
        echo "用法: $0 create-env [env_type] [--force]"
        echo
        echo "参数:"
        echo "  env_type    环境类型: dev|development|prod|production (默认: dev)"
        echo "  --force     强制覆盖已存在的配置文件"
        echo
        echo "说明:"
        echo "  从模板文件创建环境配置文件："
        echo "  • dev环境: .env.example → .env"
        echo "  • prod环境: .env.prod.example → .env.prod"
        echo "  • 自动生成安全密码"
        echo "  • 创建相关依赖配置文件"
        echo
        echo "环境类型:"
        echo "  dev/development  - 开发环境配置"
        echo "  prod/production  - 生产环境配置（包含密码生成）"
        echo
        echo "示例:"
        echo "  $0 create-env dev"
        echo "  $0 create-env prod --force"
        return 0
    fi
    
    local env_type="${1:-dev}"  # dev 或 prod
    local force="${2:-false}"
    
    print_info "正在创建环境配置文件..."
    
    case "$env_type" in
        "prod"|"production")
            local template_file=".env.prod.example"
            local target_file=".env.prod"
            ;;
        "dev"|"development"|*)
            local template_file=".env.example"
            local target_file=".env"
            ;;
    esac
    
    # 对于生产环境，使用密码替换功能
    if [[ "$env_type" == "prod" ]] || [[ "$env_type" == "production" ]]; then
        if replace_template_passwords "$template_file" "$target_file" "$force"; then
            # 检查并创建backend目录的环境文件
            if [[ ! -f "src/backend/.env" ]] && [[ -f "src/backend/.env.example" ]]; then
                cp "src/backend/.env.example" "src/backend/.env"
                print_success "✓ 创建后端环境文件: src/backend/.env"
            fi
            
            # 应用生产环境特殊配置
            print_info "应用生产环境配置..."
            sed_inplace 's/DEBUG_MODE=true/DEBUG_MODE=false/g' "$target_file" 2>/dev/null || true
            sed_inplace 's/LOG_LEVEL=debug/LOG_LEVEL=info/g' "$target_file" 2>/dev/null || true
            sed_inplace 's/BUILD_ENV=development/BUILD_ENV=production/g' "$target_file" 2>/dev/null || true
            cleanup_backup_files
            
            return 0
        else
            return 1
        fi
    fi
    
    # 检查模板文件是否存在
    if [[ ! -f "$template_file" ]]; then
        print_error "模板文件不存在: $template_file"
        return 1
    fi
    
    # 检查目标文件是否已存在
    if [[ -f "$target_file" ]] && [[ "$force" != "true" ]]; then
        print_warning "环境文件已存在: $target_file"
        print_info "如需强制覆盖，请使用 --force 参数"
        return 0
    fi
    
    # 复制模板文件并进行变量替换 (开发环境)
    if cp "$template_file" "$target_file"; then
        print_success "✓ 创建环境文件: $target_file (从 $template_file)"
        
        # 自动检测外部主机IP
        local detected_host
        if [[ "$force" == "true" ]] || [[ ! -f "$target_file" ]] || ! grep -q "^EXTERNAL_HOST=" "$target_file"; then
            detected_host=$(auto_detect_external_ip_silent)
        else
            # 从现有文件读取EXTERNAL_HOST
            detected_host=$(grep "^EXTERNAL_HOST=" "$target_file" | cut -d'=' -f2)
            if [[ -z "$detected_host" ]]; then
                detected_host=$(auto_detect_external_ip_silent)
            fi
        fi
        
        # 设置默认端口和协议
        local external_port="${EXTERNAL_PORT:-8080}"
        local external_scheme="${EXTERNAL_SCHEME:-http}"
        
        print_info "使用外部配置: HOST=$detected_host, PORT=$external_port, SCHEME=$external_scheme"
        
        # 使用增强型模板渲染
        render_env_template_enhanced "$template_file" "$target_file" "$detected_host" "$external_port" "$external_scheme" "true"
        
        # 设置SaltStack默认配置（如果未设置）
        setup_saltstack_defaults "$target_file"
        
        # 设置其他服务的默认配置（如果未设置）
        setup_services_defaults "$target_file"
        
        # 检查并创建backend目录的环境文件
        if [[ ! -f "src/backend/.env" ]] && [[ -f "src/backend/.env.example" ]]; then
            cp "src/backend/.env.example" "src/backend/.env"
            print_success "✓ 创建后端环境文件: src/backend/.env"
        fi
        
        return 0
    else
        print_error "创建环境文件失败"
        return 1
    fi
}

# 自动生成环境文件（用于自动修复）
auto_generate_env_files() {
    local force="${1:-false}"
    
    print_info "=========================================="
    print_info "自动生成环境配置文件"
    print_info "=========================================="
    
    local generated_count=0
    local failed_count=0
    
    # 生成主环境文件
    if [[ ! -f ".env" ]] || [[ "$force" == "true" ]]; then
        print_info "生成主环境文件 .env..."
        if create_env_from_template "dev" "$force"; then
            ((generated_count++))
        else
            ((failed_count++))
        fi
    else
        print_info "主环境文件 .env 已存在，跳过"
    fi
    
    # 生成生产环境文件
    if [[ ! -f ".env.prod" ]] || [[ "$force" == "true" ]]; then
        print_info "生成生产环境文件 .env.prod..."
        if create_env_from_template "prod" "$force"; then
            ((generated_count++))
        else
            ((failed_count++))
        fi
    else
        print_info "生产环境文件 .env.prod 已存在，跳过"
    fi
    
    # 检查并修复PostgreSQL密码一致性
    print_info "检查PostgreSQL密码配置一致性..."
    local env_postgres_password=$(grep -E '^POSTGRES_PASSWORD=' .env 2>/dev/null | cut -d'=' -f2 | tr -d '"' | tr -d "'")
    local env_postgres_user=$(grep -E '^POSTGRES_USER=' .env 2>/dev/null | cut -d'=' -f2 | tr -d '"' | tr -d "'")
    
    if [[ -n "$env_postgres_password" ]] && [[ -n "$env_postgres_user" ]]; then
        print_success "✓ PostgreSQL配置: 用户=$env_postgres_user, 密码=<已设置>"
    else
        print_warning "PostgreSQL密码配置可能有问题，请检查.env文件"
    fi
    
    # 检查Redis密码配置
    local redis_password=$(grep -E '^REDIS_PASSWORD=' .env 2>/dev/null | cut -d'=' -f2 | tr -d '"' | tr -d "'")
    if [[ -n "$redis_password" ]]; then
        print_success "✓ Redis密码配置正常"
    else
        print_warning "Redis密码配置可能有问题，请检查.env文件"
    fi
    
    print_info "=========================================="
    if [[ $failed_count -eq 0 ]]; then
        print_success "环境文件生成完成: $generated_count 个文件"
        print_info "建议重启所有服务以应用新配置"
        return 0
    else
        print_error "环境文件生成失败: $failed_count 个文件"
        return 1
    fi
}

# 检测并确定唯一的环境文件
detect_env_file() {
    local env_file=""
    
    # 优先级检查：.env.prod > .env > .env.example
    if [[ -f ".env.prod" ]]; then
        env_file=".env.prod"
        echo "使用生产环境配置: $env_file" >&2
    elif [[ -f ".env" ]]; then
        env_file=".env"
        echo "使用开发环境配置: $env_file" >&2
    elif [[ -f ".env.example" ]]; then
        echo "未找到环境配置文件，从模板创建..." >&2
        if create_env_from_template "dev"; then
            env_file=".env"
            echo "✓ 从.env.example创建了.env文件" >&2
        else
            echo "错误: 创建环境文件失败" >&2
            return 1
        fi
    else
        echo "错误: 未找到任何环境配置文件（.env.prod, .env, .env.example）" >&2
        return 1
    fi
    
    echo "$env_file"
    return 0
}

# 验证环境文件有效性
validate_env_file() {
    local env_file="$1"
    
    if [[ ! -f "$env_file" ]]; then
        echo "错误: 环境文件不存在: $env_file" >&2
        return 1
    fi
    
    # 检查关键变量是否存在
    local required_vars=("IMAGE_TAG" "COMPOSE_PROJECT_NAME")
    local missing_vars=()
    
    for var in "${required_vars[@]}"; do
        if ! grep -q "^${var}=" "$env_file" 2>/dev/null; then
            missing_vars+=("$var")
        fi
    done
    
    if [[ ${#missing_vars[@]} -gt 0 ]]; then
        echo "警告: 环境文件 $env_file 缺少必要变量: ${missing_vars[*]}" >&2
        echo "建议检查并补充这些变量" >&2
    fi
    
    return 0
}

# 更新外部主机配置
update_external_host_config() {
    local host_ip="${1:-auto}"
    
    print_info "=========================================="
    print_info "🌐 更新外部主机配置"
    print_info "=========================================="
    
    # 自动检测外部主机IP
    if [[ "$host_ip" == "auto" ]]; then
        print_info "自动检测外部主机IP..."
        
        # 尝试检测外部可访问的IP地址
        local detected_ip=""
        
        # 方法1: 通过默认路由检测
        if command -v ip >/dev/null 2>&1; then
            detected_ip=$(ip route get 8.8.8.8 2>/dev/null | sed -n 's/.*src \([0-9.]*\).*/\1/p' | head -1)
        fi
        
        # 方法2: 通过ifconfig检测（macOS兼容）
        if [[ -z "$detected_ip" ]] && command -v ifconfig >/dev/null 2>&1; then
            detected_ip=$(ifconfig | grep -E 'inet\s+([0-9]{1,3}\.){3}[0-9]{1,3}' | grep -v '127.0.0.1' | awk '{print $2}' | head -1)
        fi
        
        # 方法3: 通过route命令（macOS兼容）
        if [[ -z "$detected_ip" ]] && command -v route >/dev/null 2>&1; then
            detected_ip=$(route get default 2>/dev/null | grep interface | awk '{print $2}' | xargs ifconfig 2>/dev/null | grep -E 'inet\s+([0-9]{1,3}\.){3}[0-9]{1,3}' | grep -v '127.0.0.1' | awk '{print $2}' | head -1)
        fi
        
        # 备用方案: 使用localhost
        if [[ -z "$detected_ip" ]]; then
            detected_ip="localhost"
            print_warning "无法自动检测外部IP，使用默认值: localhost"
        else
            print_success "检测到外部IP: $detected_ip"
        fi
        
        host_ip="$detected_ip"
    fi
    
    print_info "目标主机IP: $host_ip"
    
    # 确定要更新的环境文件
    local env_files=()
    [[ -f ".env" ]] && env_files+=(".env")
    [[ -f ".env.prod" ]] && env_files+=(".env.prod")
    [[ -f ".env.example" ]] && env_files+=(".env.example")
    
    if [[ ${#env_files[@]} -eq 0 ]]; then
        print_error "未找到任何环境配置文件"
        return 1
    fi
    
    print_info "将更新以下环境文件: ${env_files[*]}"
    
    local success_count=0
    local total_count=${#env_files[@]}
    
    for env_file in "${env_files[@]}"; do
        print_info "→ 更新文件: $env_file"
        
        # 备份原文件
        local backup_file="${env_file}.backup.$(date +%Y%m%d_%H%M%S)"
        if cp "$env_file" "$backup_file"; then
            print_info "  ✓ 创建备份: $backup_file"
        else
            print_warning "  ⚠ 无法创建备份文件"
        fi
        
        # 更新EXTERNAL_HOST
        if grep -q "^EXTERNAL_HOST=" "$env_file"; then
            # 更新现有的EXTERNAL_HOST
            if sed_inplace "s/^EXTERNAL_HOST=.*/EXTERNAL_HOST=$host_ip/" "$env_file"; then
                cleanup_backup_files "$(dirname "$env_file")"
                print_success "  ✓ 更新EXTERNAL_HOST=$host_ip"
            else
                print_error "  ✗ 更新EXTERNAL_HOST失败"
                continue
            fi
        else
            # 添加新的EXTERNAL_HOST
            echo "EXTERNAL_HOST=$host_ip" >> "$env_file"
            print_success "  ✓ 添加EXTERNAL_HOST=$host_ip"
        fi
        
        # 确保其他动态配置变量存在
        local dynamic_vars=(
            "EXTERNAL_PORT=8080"
            "EXTERNAL_SCHEME=http"
        )
        
        for var_line in "${dynamic_vars[@]}"; do
            local var_name=$(echo "$var_line" | cut -d'=' -f1)
            if ! grep -q "^${var_name}=" "$env_file"; then
                echo "$var_line" >> "$env_file"
                print_success "  ✓ 添加默认配置: $var_line"
            fi
        done
        
        ((success_count++))
    done
    
    print_info "=========================================="
    if [[ $success_count -eq $total_count ]]; then
        print_success "✅ 外部主机配置更新完成: $success_count/$total_count 文件"
        print_info "新的外部主机: $host_ip"
        print_info "建议重新生成nginx配置并重启服务："
        print_info "  $0 build nginx"
        print_info "  docker compose restart nginx"
    else
        print_error "❌ 外部主机配置更新失败: $success_count/$total_count 文件"
        return 1
    fi
    
    return 0
}

# 更新外部端口配置
update_external_port_config() {
    local port="${1:-8080}"
    
    print_info "=========================================="
    print_info "🔌 更新外部端口配置"
    print_info "=========================================="
    print_info "目标端口: $port"
    
    # 验证端口号格式
    if ! [[ "$port" =~ ^[0-9]+$ ]] || [[ "$port" -lt 1 ]] || [[ "$port" -gt 65535 ]]; then
        print_error "无效的端口号: $port (必须是1-65535之间的数字)"
        return 1
    fi
    
    # 确定要更新的环境文件
    local env_files=()
    [[ -f ".env" ]] && env_files+=(".env")
    [[ -f ".env.prod" ]] && env_files+=(".env.prod")
    [[ -f ".env.example" ]] && env_files+=(".env.example")
    
    if [[ ${#env_files[@]} -eq 0 ]]; then
        print_error "未找到任何环境配置文件"
        return 1
    fi
    
    print_info "将更新以下环境文件: ${env_files[*]}"
    
    local success_count=0
    local total_count=${#env_files[@]}
    
    for env_file in "${env_files[@]}"; do
        print_info "→ 更新文件: $env_file"
        
        # 备份原文件
        local backup_file="${env_file}.backup.$(date +%Y%m%d_%H%M%S)"
        if cp "$env_file" "$backup_file"; then
            print_info "  ✓ 创建备份: $backup_file"
        else
            print_warning "  ⚠ 无法创建备份文件"
        fi
        
        # 更新EXTERNAL_PORT
        if grep -q "^EXTERNAL_PORT=" "$env_file"; then
            # 更新现有的EXTERNAL_PORT
            if sed_inplace "s/^EXTERNAL_PORT=.*/EXTERNAL_PORT=$port/" "$env_file"; then
                cleanup_backup_files "$(dirname "$env_file")"
                print_success "  ✓ 更新EXTERNAL_PORT=$port"
            else
                print_error "  ✗ 更新EXTERNAL_PORT失败"
                continue
            fi
        else
            # 添加新的EXTERNAL_PORT
            echo "EXTERNAL_PORT=$port" >> "$env_file"
            print_success "  ✓ 添加EXTERNAL_PORT=$port"
        fi
        
        # 计算并显示相关端口
        local jupyter_port=$((port + 8))
        local gitea_port=$((port - 5070))
        local debug_port=$((port - 79))
        
        print_info "  → 计算的端口配置:"
        print_info "    主入口端口: $port"
        print_info "    JupyterHub端口: $jupyter_port"
        print_info "    Gitea端口: $gitea_port"
        print_info "    调试端口: $debug_port"
        
        ((success_count++))
    done
    
    print_info "=========================================="
    if [[ $success_count -eq $total_count ]]; then
        print_success "✅ 外部端口配置更新完成: $success_count/$total_count 文件"
        print_info "新的外部端口: $port"
        print_info "端口映射:"
        print_info "  • 主入口: $port"
        print_info "  • JupyterHub: $((port + 8))"
        print_info "  • Gitea: $((port - 5070))"
        print_info "  • 调试端口: $((port - 79))"
        print_info ""
        print_info "建议重新生成配置并重启服务："
        print_info "  $0 build nginx --force"
        print_info "  docker compose down && docker compose up -d"
    else
        print_error "❌ 外部端口配置更新失败: $success_count/$total_count 文件"
        return 1
    fi
    
    return 0
}

# 一键更新端口并重新部署
quick_deploy_with_port() {
    local port="${1:-8080}"
    local host="${2:-auto}"
    
    print_info "=========================================="
    print_info "🚀 一键更新端口并重新部署"
    print_info "=========================================="
    print_info "目标端口: $port"
    print_info "目标主机: $host"
    
    # 步骤1: 更新外部主机配置
    print_info "步骤1: 更新外部主机配置..."
    if ! update_external_host_config "$host"; then
        print_error "外部主机配置更新失败"
        return 1
    fi
    
    # 步骤2: 更新端口配置
    print_info "步骤2: 更新端口配置..."
    if ! update_external_port_config "$port"; then
        print_error "端口配置更新失败"
        return 1
    fi
    
    # 步骤3: 重新构建nginx
    print_info "步骤3: 重新构建nginx配置..."
    FORCE_REBUILD=true
    if ! build_service "nginx" "$DEFAULT_IMAGE_TAG"; then
        print_error "nginx构建失败"
        return 1
    fi
    
    # 步骤4: 重启nginx服务
    print_info "步骤4: 重启nginx服务..."
    if docker compose restart nginx; then
        print_success "✓ nginx服务重启成功"
    else
        print_error "nginx服务重启失败"
        return 1
    fi
    
    # 显示服务信息
    print_info "=========================================="
    print_success "🎉 一键部署完成！"
    print_info "服务访问地址:"
    local current_host=$(grep "^EXTERNAL_HOST=" .env.example | cut -d'=' -f2)
    local current_scheme=$(grep "^EXTERNAL_SCHEME=" .env.example | cut -d'=' -f2)
    print_info "  • 主入口: ${current_scheme}://${current_host}:${port}"
    print_info "  • JupyterHub: ${current_scheme}://${current_host}:$((port + 8))/jupyter/"
    print_info "  • Gitea: ${current_scheme}://${current_host}:$((port - 5070))/gitea/"
    print_info "  • 调试接口: ${current_scheme}://${current_host}:$((port - 79))/debug/"
    print_info "=========================================="
    
    return 0
}

# 对比两个环境文件的差异
compare_env_files() {
    local env1="$1"
    local env2="$2"
    
    if [[ ! -f "$env1" ]] || [[ ! -f "$env2" ]]; then
        print_error "环境文件不存在: $env1 或 $env2"
        return 1
    fi
    
    print_info "对比环境文件: $env1 vs $env2"
    
    # 提取所有变量名（排除注释和空行）
    local vars1=$(grep -E "^[A-Z_][A-Z0-9_]*=" "$env1" | cut -d'=' -f1 | sort)
    local vars2=$(grep -E "^[A-Z_][A-Z0-9_]*=" "$env2" | cut -d'=' -f1 | sort)
    
    # 找出差异变量
    local only_in_1=$(comm -23 <(echo "$vars1") <(echo "$vars2"))
    local only_in_2=$(comm -13 <(echo "$vars1") <(echo "$vars2"))
    local common_vars=$(comm -12 <(echo "$vars1") <(echo "$vars2"))
    
    if [[ -n "$only_in_1" ]]; then
        print_warning "仅在 $env1 中存在的变量:"
        echo "$only_in_1" | while read var; do
            echo "  - $var"
        done
    fi
    
    if [[ -n "$only_in_2" ]]; then
        print_warning "仅在 $env2 中存在的变量:"
        echo "$only_in_2" | while read var; do
            echo "  - $var"
        done
    fi
    
    # 检查共同变量的值差异
    local diff_count=0
    echo "$common_vars" | while read var; do
        if [[ -n "$var" ]]; then
            local val1=$(grep "^${var}=" "$env1" | cut -d'=' -f2- | tr -d '"'"'"'"')
            local val2=$(grep "^${var}=" "$env2" | cut -d'=' -f2- | tr -d '"'"'"'"')
            if [[ "$val1" != "$val2" ]]; then
                if [[ $diff_count -eq 0 ]]; then
                    print_info "值不同的变量:"
                fi
                echo "  $var:"
                echo "    $env1: $val1"
                echo "    $env2: $val2"
                ((diff_count++))
            fi
        fi
    done
    
    if [[ -z "$only_in_1" ]] && [[ -z "$only_in_2" ]] && [[ $diff_count -eq 0 ]]; then
        print_success "✓ 环境文件配置一致"
    fi
    
    return 0
}

# 校验环境文件的完整性和一致性
validate_env_consistency() {
    local dev_env=".env"
    local prod_env=".env.prod"
    local example_env=".env.example"
    
    print_info "=========================================="
    print_info "环境文件一致性校验"
    print_info "=========================================="
    
    # 检查文件存在性
    local files_exist=()
    local files_missing=()
    
    for env_file in "$dev_env" "$prod_env" "$example_env"; do
        if [[ -f "$env_file" ]]; then
            files_exist+=("$env_file")
        else
            files_missing+=("$env_file")
        fi
    done
    
    print_info "存在的环境文件: ${files_exist[*]}"
    if [[ ${#files_missing[@]} -gt 0 ]]; then
        print_warning "缺失的环境文件: ${files_missing[*]}"
    fi
    
    # 如果开发环境和生产环境文件都存在，进行对比
    if [[ -f "$dev_env" ]] && [[ -f "$prod_env" ]]; then
        echo
        compare_env_files "$dev_env" "$prod_env"
    fi
    
    # 校验必要的变量
    echo
    for env_file in "${files_exist[@]}"; do
        print_info "校验 $env_file..."
        validate_env_file "$env_file"
    done
    
    return 0
}

# ==========================================
# Docker Compose 版本检测和适配
# ==========================================

# 检测Docker Compose版本并返回最佳命令
detect_compose_command() {
    local compose_cmd=""
    local compose_version=""
    
    # 优先使用docker compose (v2)
    if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
        compose_cmd="docker compose"
        compose_version=$(docker compose version --short 2>/dev/null || docker compose version | grep -o 'v[0-9.]*' | head -1)
        echo "$compose_cmd"
        return 0
    fi
    
    # 回退到docker-compose (v1)
    if command -v docker-compose >/dev/null 2>&1; then
        compose_cmd="docker-compose"
        compose_version=$(docker-compose version --short 2>/dev/null || docker-compose version | grep -o '[0-9.]*' | head -1)
        echo "$compose_cmd"
        return 0
    fi
    
    return 1
}

# 检查Docker Compose版本兼容性
check_compose_compatibility() {
    local compose_cmd
    compose_cmd=$(detect_compose_command)
    local exit_code=$?
    
    if [ $exit_code -ne 0 ]; then
        print_error "未找到Docker Compose命令"
        print_info "请安装Docker Compose v2.0+:"
        print_info "  https://docs.docker.com/compose/install/"
        return 1
    fi
    
    local version=""
    if [[ "$compose_cmd" == "docker compose" ]]; then
        version=$(docker compose version --short 2>/dev/null || docker compose version | grep -o 'v[0-9.]*' | head -1 | sed 's/v//')
        print_info "检测到Docker Compose v2: $version"
        
        # 清理版本号，移除v前缀和额外信息
        local clean_version=$(echo "$version" | sed 's/^v//' | sed 's/-.*$//')
        
        # 检查是否为v2.39.2或更高版本
        if command -v python3 >/dev/null 2>&1; then
            local is_compatible=$(python3 -c "
import sys
from packaging import version
try:
    current = version.parse('$clean_version')
    required = version.parse('2.39.2')
    print('true' if current >= required else 'false')
except Exception as e:
    print('true')  # 默认兼容
" 2>/dev/null || echo "true")
            
            if [[ "$is_compatible" == "true" ]]; then
                print_success "✓ Docker Compose版本兼容 (v$clean_version >= v2.39.2)"
            else
                print_warning "⚠ Docker Compose版本较旧 (v$clean_version < v2.39.2)，建议升级"
                print_info "当前版本应该仍可工作，但建议升级以获得最佳体验"
            fi
        else
            print_info "✓ 使用Docker Compose v2: $clean_version"
        fi
    else
        version=$(docker-compose version --short 2>/dev/null || docker-compose version | grep -o '[0-9.]*' | head -1)
        print_warning "检测到Docker Compose v1: $version"
        print_info "建议升级到Docker Compose v2以获得更好的性能和功能"
    fi
    
    echo "$compose_cmd"
    return 0
}

# 验证compose文件格式
validate_compose_file() {
    local file="$1"
    local compose_cmd="$2"
    
    if [[ ! -f "$file" ]]; then
        print_error "Compose文件不存在: $file"
        return 1
    fi
    
    print_info "验证compose文件: $file"
    
    if ! $compose_cmd -f "$file" config >/dev/null 2>&1; then
        print_error "Compose文件验证失败: $file"
        print_info "详细错误信息："
        $compose_cmd -f "$file" config 2>&1 | head -10
        return 1
    fi
    
    print_success "✓ Compose文件验证通过: $file"
    return 0
}

# 获取私有镜像名称（支持Harbor格式：registry/project）
get_private_image_name() {
    local original_image="$1"
    local registry="$2"
    
    if [[ -z "$registry" ]]; then
        echo "$original_image"
        return 0
    fi
    
    # 检查original_image是否已经包含了registry信息
    if [[ "$original_image" == "$registry"/* ]]; then
        echo "$original_image"
        return 0
    fi
    
    # 处理不同类型的registry格式
    local registry_base=""
    local project_path=""
    local is_harbor_style=false
    
    if [[ "$registry" == *"/"* ]]; then
        # Harbor格式：registry.xxx.com/project
        is_harbor_style=true
        registry_base="${registry%%/*}"  # 获取 registry.xxx.com
        project_path="${registry#*/}"    # 获取 project
    else
        # 传统格式：registry.xxx.com
        registry_base="$registry"
    fi
    
    # 处理镜像名称
    local image_name_tag=""
    
    if [[ "$original_image" == *"/"* ]]; then
        # 包含组织/用户名的镜像
        if [[ "$original_image" == *"."*"/"* ]]; then
            # 第三方仓库镜像 (如 quay.io/minio/minio:latest)
            image_name_tag="${original_image#*/}"  # 移除域名部分
        else
            # Docker Hub 组织镜像 (如 osixia/openldap:stable)
            image_name_tag="$original_image"
        fi
    else
        # 简单镜像名 (如 redis:7-alpine, postgres:15-alpine)
        image_name_tag="$original_image"
    fi
    
    # 构建最终镜像路径
    if [[ "$is_harbor_style" == "true" ]]; then
        # Harbor模式：registry.xxx.com/project/image:tag
        echo "${registry}/${image_name_tag}"
    else
        # 传统模式：registry.xxx.com/image:tag
        echo "${registry}/${image_name_tag}"
    fi
}

# 根据镜像映射配置获取私有镜像名称和版本
# 支持latest标签到git版本的映射
get_mapped_private_image() {
    local original_image="$1"
    local registry="$2"
    local target_tag="${3:-v0.3.6-dev}"  # 默认目标git版本
    local mapping_file="$SCRIPT_DIR/config/image-mapping.conf"
    
    if [[ -z "$registry" ]]; then
        echo "$original_image"
        return 0
    fi
    
    # 标准化镜像名称（移除tag用于匹配）
    local image_base=""
    local original_tag=""
    
    if [[ "$original_image" == *":"* ]]; then
        image_base="${original_image%%:*}"
        original_tag="${original_image##*:}"
    else
        image_base="$original_image"
        original_tag="latest"
    fi
    
    # 提取原始镜像的简短名称（不含namespace）
    local simple_name=""
    if [[ "$image_base" == *"/"* ]]; then
        # 处理带namespace的镜像，如 tecnativa/tcp-proxy -> tcp-proxy
        simple_name="${image_base##*/}"
    else
        # 直接使用镜像名，如 postgres -> postgres
        simple_name="$image_base"
    fi
    
    # 如果映射文件存在，尝试读取映射配置
    local mapped_project=""
    local mapped_version=""
    local found_mapping=false
    
    if [[ -f "$mapping_file" ]]; then
        while IFS='|' read -r pattern project version special; do
            # 跳过注释和空行
            [[ "$pattern" =~ ^[[:space:]]*# ]] && continue
            [[ -z "$pattern" ]] && continue
            
            # 检查是否匹配（支持精确匹配和基础名匹配）
            if [[ "$original_image" == "$pattern" ]] || 
               [[ "$image_base" == "$pattern" ]] ||
               [[ "$image_base:$original_tag" == "$pattern" ]]; then
                mapped_project="$project"
                mapped_version="$version"
                found_mapping=true
                break
            fi
        done < "$mapping_file"
    fi
    
    local final_version=""
    if [[ "$found_mapping" == "true" ]]; then
        # 处理特殊变量替换
        if [[ "$mapped_version" == *'${TARGET_TAG}'* ]]; then
            # 项目镜像，使用传入的target_tag
            final_version="${mapped_version//\$\{TARGET_TAG\}/$target_tag}"
        elif [[ "$mapped_version" == *'${IMAGE_TAG}'* ]]; then
            # 兼容旧格式
            final_version="${mapped_version//\$\{IMAGE_TAG\}/$target_tag}"
        else
            # 使用配置文件中的版本
            final_version="$mapped_version"
        fi
    else
        # 未找到映射，强制使用目标标签
        final_version="$target_tag"
    fi
    
    # 构建最终镜像名：registry/simple_name:final_version
    local final_image="${registry}/${simple_name}:${final_version}"
    
    echo "$final_image"
}

# 检查 Dockerfile 是否存在
check_dockerfile() {
    local service="$1"
    local service_path=$(get_service_path "$service")
    
    if [[ -z "$service_path" ]]; then
        print_error "未知服务: $service"
        return 1
    fi
    
    local dockerfile_path="$SCRIPT_DIR/$service_path/Dockerfile"
    
    if [[ ! -f "$dockerfile_path" ]]; then
        print_error "Dockerfile 不存在: $dockerfile_path"
        return 1
    fi
    return 0
}

# ========================================
# 镜像构建状态检查功能（需求32）
# ========================================

# 验证镜像是否正确构建
# 参数：
#   $1: 镜像名称（含标签）
# 返回：
#   0: 镜像存在且有效
#   1: 镜像不存在或无效
verify_image_build() {
    local image_name="$1"
    
    if [[ -z "$image_name" ]]; then
        return 1
    fi
    
    # 检查镜像是否存在
    if ! docker image inspect "$image_name" >/dev/null 2>&1; then
        return 1
    fi
    
    # 检查镜像是否有正确的标签和创建时间
    local image_info
    image_info=$(docker image inspect "$image_name" --format '{{.Created}}|{{.Size}}' 2>/dev/null)
    
    if [[ -z "$image_info" ]]; then
        return 1
    fi
    
    # 提取创建时间和大小
    local created_time="${image_info%%|*}"
    local image_size="${image_info##*|}"
    
    # 检查镜像大小（必须大于0）
    if [[ "$image_size" -eq 0 ]]; then
        print_error "  ⚠ 镜像大小为0，可能构建失败: $image_name"
        return 1
    fi
    
    # 检查镜像是否是 scratch 或 dangling
    local repo_tags
    repo_tags=$(docker image inspect "$image_name" --format '{{.RepoTags}}' 2>/dev/null)
    
    if [[ "$repo_tags" == "[]" ]] || [[ -z "$repo_tags" ]]; then
        print_error "  ⚠ 镜像没有有效标签: $image_name"
        return 1
    fi
    
    return 0
}

# 获取所有服务的构建状态
# 参数：
#   $1: 镜像标签（默认：$DEFAULT_IMAGE_TAG）
#   $2: 私有仓库地址（可选）
# 输出：
#   输出服务名称和构建状态到标准输出
#   格式：service_name|status|image_name
#   status: OK, MISSING, INVALID
get_build_status() {
    local tag="${1:-$DEFAULT_IMAGE_TAG}"
    local registry="${2:-}"
    
    local all_services="$SRC_SERVICES"
    
    for service in $all_services; do
        local base_image="ai-infra-${service}:${tag}"
        local target_image="$base_image"
        
        if [[ -n "$registry" ]]; then
            target_image=$(get_private_image_name "$base_image" "$registry")
        fi
        
        local status="MISSING"
        
        # 检查镜像是否存在
        if docker image inspect "$target_image" >/dev/null 2>&1; then
            # 验证镜像是否有效
            if verify_image_build "$target_image"; then
                status="OK"
            else
                status="INVALID"
            fi
        fi
        
        echo "${service}|${status}|${target_image}"
    done
}

# 显示构建状态报告
# 参数：
#   $1: 镜像标签（默认：$DEFAULT_IMAGE_TAG）
#   $2: 私有仓库地址（可选）
show_build_status() {
    local tag="${1:-$DEFAULT_IMAGE_TAG}"
    local registry="${2:-}"
    
    print_info "=========================================="
    print_info "镜像构建状态报告"
    print_info "=========================================="
    print_info "镜像标签: $tag"
    if [[ -n "$registry" ]]; then
        print_info "目标仓库: $registry"
    else
        print_info "目标仓库: 本地构建"
    fi
    echo
    
    local ok_count=0
    local missing_count=0
    local invalid_count=0
    local total_count=0
    
    # 使用数组存储不同状态的服务
    local ok_services=()
    local missing_services=()
    local invalid_services=()
    
    while IFS='|' read -r service status image_name; do
        total_count=$((total_count + 1))
        
        case "$status" in
            "OK")
                ok_count=$((ok_count + 1))
                ok_services+=("$service")
                ;;
            "MISSING")
                missing_count=$((missing_count + 1))
                missing_services+=("$service")
                ;;
            "INVALID")
                invalid_count=$((invalid_count + 1))
                invalid_services+=("$service")
                ;;
        esac
    done < <(get_build_status "$tag" "$registry")
    
    # 显示统计信息
    print_info "📊 构建状态统计:"
    print_success "  ✓ 构建成功: $ok_count/$total_count"
    if [[ $missing_count -gt 0 ]]; then
        print_error "  ✗ 缺失镜像: $missing_count/$total_count"
    fi
    if [[ $invalid_count -gt 0 ]]; then
        print_error "  ⚠ 无效镜像: $invalid_count/$total_count"
    fi
    echo
    
    # 显示成功的服务
    if [[ ${#ok_services[@]} -gt 0 ]]; then
        print_success "✓ 构建成功的服务 ($ok_count):"
        for service in "${ok_services[@]}"; do
            print_info "  • $service"
        done
        echo
    fi
    
    # 显示缺失的服务
    if [[ ${#missing_services[@]} -gt 0 ]]; then
        print_error "✗ 缺失镜像的服务 ($missing_count):"
        for service in "${missing_services[@]}"; do
            print_info "  • $service"
        done
        echo
    fi
    
    # 显示无效的服务
    if [[ ${#invalid_services[@]} -gt 0 ]]; then
        print_error "⚠ 镜像无效的服务 ($invalid_count):"
        for service in "${invalid_services[@]}"; do
            print_info "  • $service"
        done
        echo
    fi
    
    return 0
}

# 获取需要构建的服务列表
# 参数：
#   $1: 镜像标签（默认：$DEFAULT_IMAGE_TAG）
#   $2: 私有仓库地址（可选）
# 输出：
#   需要构建的服务名称（空格分隔）
get_services_to_build() {
    local tag="${1:-$DEFAULT_IMAGE_TAG}"
    local registry="${2:-}"
    
    local services_to_build=()
    
    while IFS='|' read -r service status image_name; do
        # 只构建缺失或无效的镜像
        if [[ "$status" != "OK" ]]; then
            services_to_build+=("$service")
        fi
    done < <(get_build_status "$tag" "$registry")
    
    # 输出服务列表（空格分隔）
    echo "${services_to_build[@]}"
}

# 提取 Dockerfile 中的基础镜像
extract_base_images() {
    local dockerfile_path="$1"
    
    if [[ ! -f "$dockerfile_path" ]]; then
        print_error "Dockerfile 不存在: $dockerfile_path"
        return 1
    fi
    
    # 提取所有 FROM 指令中的镜像名称
    # 支持: FROM image:tag, FROM image:tag AS stage, FROM --platform=xxx image:tag
    # 修复：确保正确提取镜像名称，不包含 FROM 关键字
    # macOS 兼容：使用 grep -i 而不是 sed //I
    grep -iE '^\s*FROM\s+' "$dockerfile_path" | \
        sed -E 's/^[[:space:]]*[Ff][Rr][Oo][Mm][[:space:]]+//' | \
        sed -E 's/--platform=[^[:space:]]+[[:space:]]+//' | \
        awk '{print $1}' | \
        grep -v '^$' | \
        grep -v '^#' | \
        sort -u
}

# 智能镜像tag函数 - 根据网络环境自动处理镜像别名
# 功能：
#   公网环境：确保原始镜像名称和 localhost/ 前缀版本都存在
#   内网环境：确保从 Harbor 仓库拉取的镜像有正确的别名
# 参数：
#   $1: 镜像名称（可以是原始名称、localhost/ 前缀或 Harbor 完整路径）
#   $2: 网络环境（可选，auto/external/internal，默认 auto）
#   $3: Harbor 仓库地址（可选，默认从环境变量读取）
# 返回：
#   0: 成功
#   1: 失败
tag_image_smart() {
    local image="$1"
    local network_env="${2:-auto}"
    local harbor_registry="${3:-${INTERNAL_REGISTRY:-aiharbor.msxf.local/aihpc}}"
    local auto_pull="${4:-true}"  # 是否自动拉取不存在的镜像（默认启用）
    
    if [[ -z "$image" ]]; then
        print_error "tag_image_smart: 镜像名称不能为空"
        return 1
    fi
    
    # 自动检测网络环境
    if [[ "$network_env" == "auto" ]]; then
        network_env=$(detect_network_environment)
    fi
    
    # 提取基础镜像名称（智能识别不同类型的镜像前缀）
    local base_image="$image"
    local original_image="$image"
    
    # 移除 localhost/ 前缀
    base_image="${base_image#localhost/}"
    
    # 智能移除 Harbor 仓库前缀（包含域名的私有仓库）
    # 规则：如果前缀包含点号（.），则认为是私有仓库域名
    # 例如：aiharbor.msxf.local/aihpc/redis:7-alpine → redis:7-alpine
    # 但保留：osixia/openldap:stable → osixia/openldap:stable
    if [[ "$base_image" =~ ^[^/]+\.[^/]+/ ]]; then
        # 包含域名的私有仓库，移除仓库前缀
        # 格式：domain.com/project/image:tag → image:tag
        base_image=$(echo "$base_image" | sed -E 's|^[^/]+\.[^/]+/[^/]+/||')
    fi
    
    # 提取短名称（移除 Docker Hub 命名空间）
    # 例如：osixia/openldap:stable → openldap:stable
    local short_name="$base_image"
    if [[ "$base_image" =~ ^[^/]+/[^/]+: ]]; then
        # 包含命名空间（如 osixia/openldap:stable）
        short_name=$(echo "$base_image" | sed -E 's|^[^/]+/||')
    fi
    
    # ========================================
    # 步骤 1: 检查本地是否已有镜像
    # ========================================
    local localhost_short="localhost/$short_name"
    local harbor_image="${harbor_registry}/${base_image}"
    
    local has_any_local=false
    local source_image=""
    
    # 根据网络环境，调整检查优先级
    if [[ "$network_env" == "internal" ]]; then
        # 内网环境：优先使用 Harbor 镜像
        for candidate in "$harbor_image" "$base_image" "$short_name" "$localhost_short"; do
            if docker image inspect "$candidate" >/dev/null 2>&1; then
                has_any_local=true
                source_image="$candidate"
                if [[ "$candidate" == "$harbor_image" ]]; then
                    print_info "  ✓ 本地已有 Harbor 镜像: $candidate"
                else
                    print_info "  ✓ 本地已有镜像: $candidate"
                fi
                break
            fi
        done
    else
        # 公网环境：按标准优先级检查
        for candidate in "$base_image" "$short_name" "$localhost_short"; do
            if docker image inspect "$candidate" >/dev/null 2>&1; then
                has_any_local=true
                source_image="$candidate"
                print_info "  ✓ 本地已有镜像: $candidate"
                break
            fi
        done
    fi
    
    # ========================================
    # 步骤 2: 如果本地不存在，根据网络环境拉取
    # ========================================
    if ! $has_any_local && [[ "$auto_pull" == "true" ]]; then
        print_info "  ⬇ 本地未找到镜像，开始拉取..."
        
        local pull_success=false
        case "$network_env" in
            "internal")
                # 内网环境：只从 Harbor 拉取，不拉取公共镜像
                # 理由：内网环境应该已经有 Harbor 中的镜像，避免访问公网
                print_info "  📦 内网环境：尝试从 Harbor 拉取 $harbor_image"
                if docker pull "$harbor_image" 2>/dev/null; then
                    print_success "  ✓ 从 Harbor 拉取成功: $harbor_image"
                    source_image="$harbor_image"
                    pull_success=true
                else
                    print_error "  ✗ Harbor 拉取失败: $harbor_image"
                    print_warning "  ⚠️  内网环境下不会尝试从公共仓库拉取"
                    print_info "  💡 请确保镜像已推送到 Harbor 仓库"
                fi
                ;;
            "external")
                # 公网环境：从公共仓库拉取，然后 tag 为 Harbor 镜像（准备推送）
                print_info "  🌐 公网环境：从公共仓库拉取 $base_image"
                if docker pull "$base_image" 2>/dev/null; then
                    print_success "  ✓ 拉取成功: $base_image"
                    source_image="$base_image"
                    pull_success=true
                    print_info "  💡 将为该镜像创建 Harbor tag，便于推送到私有仓库"
                else
                    print_error "  ✗ 拉取失败: $base_image"
                fi
                ;;
        esac
        
        if ! $pull_success; then
            print_warning "  ⚠️  镜像拉取失败，跳过 tag 操作"
            return 1
        fi
    fi
    
    # 如果仍然没有源镜像，则报错
    if [[ -z "$source_image" ]]; then
        print_warning "  ✗ 镜像不存在且拉取失败: $base_image"
        print_info "    💡 请手动拉取镜像:"
        if [[ "$network_env" == "internal" ]]; then
            print_info "       docker pull $harbor_image  # 或"
        fi
        print_info "       docker pull $base_image"
        return 1
    fi
    
    # ========================================
    # 步骤 3: 创建双向 tag（根据网络环境）
    # ========================================
    # 根据网络环境决定策略
    case "$network_env" in
        "external")
            # 公网环境：创建标准的双向别名 + Harbor tag（如果指定）
            print_info "  🌐 公网环境：创建 tag 别名"
            
            local has_base=false
            local has_short=false
            local has_localhost=false
            local has_harbor=false
            
            # 检查哪些 tag 已存在
            docker image inspect "$base_image" >/dev/null 2>&1 && has_base=true
            docker image inspect "$short_name" >/dev/null 2>&1 && has_short=true
            docker image inspect "$localhost_short" >/dev/null 2>&1 && has_localhost=true
            docker image inspect "$harbor_image" >/dev/null 2>&1 && has_harbor=true
            
            # 从源镜像创建所有需要的别名
            # 1. 标准名称 (base_image)
            if ! $has_base && [[ "$base_image" != "$short_name" ]]; then
                if docker tag "$source_image" "$base_image" 2>/dev/null; then
                    print_success "    ✓ 已创建别名: $source_image → $base_image"
                fi
            fi
            
            # 2. 短名称 (short_name)
            if ! $has_short; then
                if docker tag "$source_image" "$short_name" 2>/dev/null; then
                    print_success "    ✓ 已创建别名: $source_image → $short_name"
                fi
            fi
            
            # 3. localhost 别名 (localhost/short_name)
            if ! $has_localhost; then
                if docker tag "$source_image" "$localhost_short" 2>/dev/null; then
                    print_success "    ✓ 已创建别名: $source_image → $localhost_short"
                fi
            fi
            
            # 4. Harbor 完整路径（如果用户明确指定了 harbor_registry）
            # 这样可以方便后续 docker push 到 Harbor
            if [[ -n "$harbor_registry" ]] && [[ "$harbor_registry" != "${INTERNAL_REGISTRY:-aiharbor.msxf.local/aihpc}" ]]; then
                # 用户明确指定了非默认的 Harbor 地址
                if ! $has_harbor; then
                    if docker tag "$source_image" "$harbor_image" 2>/dev/null; then
                        print_success "    ✓ 已创建 Harbor 别名: $source_image → $harbor_image"
                    fi
                fi
            fi
            
            return 0
            ;;
            
        "internal")
            # 内网环境：创建标准的双向别名 + Harbor tag
            print_info "  🏢 内网环境：创建 tag 别名"
            
            local has_base=false
            local has_short=false
            local has_localhost=false
            local has_harbor=false
            
            # 检查哪些 tag 已存在
            docker image inspect "$base_image" >/dev/null 2>&1 && has_base=true
            docker image inspect "$short_name" >/dev/null 2>&1 && has_short=true
            docker image inspect "$localhost_short" >/dev/null 2>&1 && has_localhost=true
            docker image inspect "$harbor_image" >/dev/null 2>&1 && has_harbor=true
            
            # 从源镜像创建所有需要的别名
            # 1. 标准名称 (base_image)
            if ! $has_base && [[ "$base_image" != "$short_name" ]]; then
                if docker tag "$source_image" "$base_image" 2>/dev/null; then
                    print_success "    ✓ 已创建别名: $source_image → $base_image"
                fi
            fi
            
            # 2. 短名称 (short_name)
            if ! $has_short; then
                if docker tag "$source_image" "$short_name" 2>/dev/null; then
                    print_success "    ✓ 已创建别名: $source_image → $short_name"
                fi
            fi
            
            # 3. localhost 别名 (localhost/short_name)
            if ! $has_localhost; then
                if docker tag "$source_image" "$localhost_short" 2>/dev/null; then
                    print_success "    ✓ 已创建别名: $source_image → $localhost_short"
                fi
            fi
            
            # 4. Harbor 完整路径 (harbor_registry/base_image)
            # 只有当 harbor_registry 有效且不是源镜像本身时才创建
            if [[ -n "$harbor_registry" ]] && [[ "$source_image" != "$harbor_image" ]]; then
                if ! $has_harbor; then
                    if docker tag "$source_image" "$harbor_image" 2>/dev/null; then
                        print_success "    ✓ 已创建 Harbor 别名: $source_image → $harbor_image"
                    fi
                fi
            fi
            
            return 0
            ;;
            
        *)
            print_error "  ✗ 未知网络环境: $network_env"
            return 1
            ;;
    esac
}

# 双向镜像tag函数（兼容旧版本，内部调用 tag_image_smart）
tag_image_bidirectional() {
    local image="$1"
    tag_image_smart "$image" "auto"
}

# 批量智能tag镜像列表
# 参数：
#   $1: 网络环境（auto/external/internal）
#   $2: Harbor 仓库地址（可选）
#   ${@:3}: 镜像名称列表
# 返回：
#   0: 全部成功
#   非0: 部分或全部失败（返回失败的数量）
batch_tag_images_smart() {
    local network_env="${1:-auto}"
    local harbor_registry="${2:-${INTERNAL_REGISTRY:-aiharbor.msxf.local/aihpc}}"
    shift 2
    local images=("$@")
    
    local success_count=0
    local fail_count=0
    local skip_count=0
    local total=${#images[@]}
    
    if [[ $total -eq 0 ]]; then
        print_warning "批量智能tag: 镜像列表为空"
        return 0
    fi
    
    # 自动检测网络环境
    if [[ "$network_env" == "auto" ]]; then
        network_env=$(detect_network_environment)
    fi
    
    print_info "=========================================="
    print_info "🏷️  批量智能tag镜像 (总计: $total)"
    print_info "=========================================="
    print_info "网络环境: $network_env"
    if [[ "$network_env" == "internal" ]]; then
        print_info "Harbor 仓库: $harbor_registry"
    fi
    echo
    
    for image in "${images[@]}"; do
        # 跳过空行
        if [[ -z "$image" ]]; then
            continue
        fi
        
        print_info "处理镜像: $image"
        
        # 执行智能tag
        if tag_image_smart "$image" "$network_env" "$harbor_registry"; then
            ((success_count++))
        else
            ((fail_count++))
        fi
    done
    
    # 输出统计信息
    echo
    print_info "📊 智能tag统计:"
    print_info "  • 成功: $success_count"
    print_info "  • 失败: $fail_count"
    print_info "  • 总计: $total"
    echo
    
    return $fail_count
}

# 批量双向tag镜像列表（兼容旧版本）
# 参数：
#   $@: 镜像名称列表
# 返回：
#   0: 全部成功
#   非0: 部分或全部失败（返回失败的数量）
batch_tag_images_bidirectional() {
    batch_tag_images_smart "auto" "${INTERNAL_REGISTRY:-aiharbor.msxf.local/aihpc}" "$@"
}

# 拉取单个镜像（带重试机制 + 网络环境感知）
# 参数：
#   $1: 镜像名称
#   $2: 最大重试次数（默认3）
#   $3: Harbor 仓库地址（可选，默认 aiharbor.msxf.local/aihpc）
# 返回：
#   0: 拉取成功或镜像已存在
#   1: 拉取失败
pull_image_with_retry() {
    local image="$1"
    local max_retries="${2:-3}"
    local harbor_registry="${3:-${INTERNAL_REGISTRY:-aiharbor.msxf.local/aihpc}}"
    local retry_count=0
    
    # 检查镜像是否已存在
    if docker image inspect "$image" >/dev/null 2>&1; then
        return 0
    fi
    
    # 检测网络环境
    local network_env=$(detect_network_environment)
    
    # 提取基础镜像名（去除 Harbor 前缀）
    local base_image="$image"
    if [[ "$base_image" =~ ^[^/]+\.[^/]+/ ]]; then
        base_image=$(echo "$base_image" | sed -E 's|^[^/]+\.[^/]+/[^/]+/||')
    fi
    
    # 根据网络环境决定拉取策略
    case "$network_env" in
        "internal")
            # 内网环境：只从 Harbor 拉取
            local harbor_image="${harbor_registry}/${base_image}"
            
            while [[ $retry_count -lt $max_retries ]]; do
                retry_count=$((retry_count + 1))
                
                if [[ $retry_count -gt 1 ]]; then
                    print_info "  🔄 重试 $retry_count/$max_retries: $harbor_image"
                    sleep 2
                fi
                
                if docker pull "$harbor_image" 2>&1 | grep -v "Pulling from"; then
                    # 拉取成功后，tag 为标准名称
                    if [[ "$harbor_image" != "$image" ]]; then
                        docker tag "$harbor_image" "$image" 2>/dev/null || true
                    fi
                    return 0
                fi
            done
            
            print_error "  ✗ 从 Harbor 拉取失败（重试${max_retries}次）: $harbor_image"
            print_warning "  ⚠️  内网环境下不会尝试从公共仓库拉取"
            return 1
            ;;
            
        "external")
            # 公网环境：从公共仓库拉取
            while [[ $retry_count -lt $max_retries ]]; do
                retry_count=$((retry_count + 1))
                
                if [[ $retry_count -gt 1 ]]; then
                    print_info "  🔄 重试 $retry_count/$max_retries: $image"
                    sleep 2
                fi
                
                if docker pull "$image" 2>&1 | grep -v "Pulling from"; then
                    return 0
                fi
            done
            
            return 1
            ;;
    esac
}

# 预拉取 Dockerfile 中的依赖镜像（带重试机制）
prefetch_base_images() {
    local dockerfile_path="$1"
    local service_name="${2:-unknown}"
    local max_retries="${3:-3}"  # 默认重试3次
    
    print_info "📦 预拉取依赖镜像: $service_name"
    
    # 提取基础镜像列表
    local base_images
    base_images=$(extract_base_images "$dockerfile_path")
    
    if [[ -z "$base_images" ]]; then
        print_info "  → 未找到需要拉取的基础镜像"
        return 0
    fi
    
    local pull_count=0
    local skip_count=0
    local fail_count=0
    
    # 遍历并拉取每个镜像
    while IFS= read -r image; do
        # 跳过空行
        if [[ -z "$image" ]]; then
            continue
        fi
        
        # 跳过内部构建阶段（通常是小写字母开头的别名）
        if [[ "$image" =~ ^[a-z_-]+$ ]]; then
            print_info "  ⊙ 跳过内部阶段: $image"
            continue
        fi
        
        # 跳过注释
        if [[ "$image" =~ ^# ]]; then
            continue
        fi
        
        # 检查镜像是否已存在
        if docker image inspect "$image" >/dev/null 2>&1; then
            print_info "  ✓ 镜像已存在: $image"
            ((skip_count++))
            
            # 即使镜像已存在，也要创建双向tag（确保 localhost/ 别名存在）
            tag_image_smart "$image" "auto" "" "false" 2>/dev/null || true
            continue
        fi
        
        # 尝试拉取镜像（带重试机制）
        print_info "  ⬇ 正在拉取: $image"
        if pull_image_with_retry "$image" "$max_retries"; then
            print_success "  ✓ 拉取成功: $image"
            ((pull_count++))
            
            # 拉取成功后自动创建双向tag（localhost/ 前缀 ↔ 原始名称）
            tag_image_smart "$image" "auto" "" "false" 2>/dev/null || true
        else
            print_error "  ✗ 拉取失败（已重试${max_retries}次）: $image"
            ((fail_count++))
            
            # 允许某些可选镜像拉取失败（如 scratch）
            if [[ "$image" =~ ^(scratch)$ ]]; then
                print_info "  ℹ 可选镜像，继续构建流程"
            else
                print_warning "  ⚠ 关键镜像拉取失败，构建可能会失败"
            fi
        fi
    done <<< "$base_images"
    
    # 输出统计信息
    print_info "📊 预拉取统计:"
    print_info "  • 新拉取: $pull_count"
    print_info "  • 已存在: $skip_count"
    if [[ $fail_count -gt 0 ]]; then
        print_error "  • 失败: $fail_count (已重试${max_retries}次)"
        print_warning "⚠ 部分镜像拉取失败，但构建流程将继续"
    fi
    
    # 即使有失败也返回成功，让构建流程继续
    # Docker build 会在真正需要时再次尝试拉取
    return 0
}

# 构建单个服务镜像
build_service() {
    # 处理帮助参数
    if [[ "$1" == "--help" || "$1" == "-h" ]]; then
        echo "build-service - 构建指定服务"
        echo
        echo "用法: $0 build-service <service> [tag] [registry]"
        echo
        echo "参数:"
        echo "  service     服务名称 (必需)"
        echo "  tag         镜像标签 (默认: $DEFAULT_IMAGE_TAG)"
        echo "  registry    私有仓库地址 (可选)"
        echo
        echo "说明:"
        echo "  构建指定的服务Docker镜像，支持："
        echo "  • 本地构建和标记"
        echo "  • 私有仓库推送"
        echo "  • Dockerfile检查"
        echo "  • 自动化构建流程"
        echo
        echo "可用服务: $SRC_SERVICES"
        echo
        echo "示例:"
        echo "  $0 build-service frontend v1.0.0"
        echo "  $0 build-service api v1.0.0 harbor.company.com/ai-infra"
        return 0
    fi
    
    local service="$1"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    local registry="${3:-}"
    
    local service_path=$(get_service_path "$service")
    if [[ -z "$service_path" ]]; then
        print_error "未知服务: $service"
        print_info "可用服务: $SRC_SERVICES"
        return 1
    fi
    
    if ! check_dockerfile "$service"; then
        return 1
    fi
    
    local dockerfile_path="$SCRIPT_DIR/$service_path/Dockerfile"
    local base_image="ai-infra-${service}:${tag}"
    
    # 确定目标镜像名
    local target_image="$base_image"
    if [[ -n "$registry" ]]; then
        target_image=$(get_private_image_name "$base_image" "$registry")
    fi
    
    print_info "构建服务: $service"
    print_info "  Dockerfile: $service_path/Dockerfile"
    print_info "  目标镜像: $target_image"
    
    # ========================================
    # 智能缓存检查
    # ========================================
    local build_id=$(generate_build_id)
    local rebuild_reason=$(need_rebuild "$service" "$tag")
    local rebuild_code=$?
    
    if [[ $rebuild_code -ne 0 ]]; then
        # 无需重建
        print_success "  ✓ 镜像无变化，复用缓存: $target_image"
        print_info "  📋 BUILD_ID: $build_id (SKIPPED)"
        
        # 记录跳过的构建
        log_build_history "$build_id" "$service" "$tag" "SKIPPED" "NO_CHANGE"
        
        # 如果指定了registry，确保本地别名也存在
        if [[ -n "$registry" ]] && [[ "$target_image" != "$base_image" ]]; then
            if ! docker image inspect "$base_image" >/dev/null 2>&1; then
                if docker tag "$target_image" "$base_image"; then
                    print_info "  ✓ 创建本地别名: $base_image"
                fi
            fi
        fi
        
        return 0
    fi
    
    # 显示重建原因
    case "$rebuild_reason" in
        "FORCE_REBUILD")
            print_info "  🔨 强制重建模式"
            ;;
        "SKIP_CACHE_CHECK")
            print_info "  ⏭️  跳过缓存检查"
            ;;
        "IMAGE_NOT_EXIST")
            print_info "  🆕 镜像不存在，需要构建"
            ;;
        "NO_HASH_LABEL")
            print_info "  🏷️  镜像缺少哈希标签，需要重建"
            ;;
        HASH_CHANGED*)
            local old_hash=$(echo "$rebuild_reason" | cut -d'|' -f2 | cut -d':' -f2)
            local new_hash=$(echo "$rebuild_reason" | cut -d'|' -f3 | cut -d':' -f2)
            print_info "  🔄 文件已变化，需要重建"
            print_info "     旧哈希: $old_hash"
            print_info "     新哈希: $new_hash"
            ;;
    esac
    
    print_info "  📋 BUILD_ID: $build_id"
    
    # ========================================
    # 预拉取依赖镜像
    # ========================================
    print_info "  → 预拉取 Dockerfile 依赖镜像..."
    prefetch_base_images "$dockerfile_path" "$service"
    
    # ========================================
    # SingleUser 智能构建处理
    # ========================================
    if [[ "$service" == "singleuser" ]]; then
        print_info "  → 检测网络环境以优化 SingleUser 构建..."
        local network_env=$(detect_network_environment)
        print_info "  → 网络环境: $network_env"
        
        # 检查是否有强制模式参数
        local force_mode="auto"
        if [[ "${SINGLEUSER_BUILD_MODE:-}" == "offline" ]]; then
            force_mode="offline"
        elif [[ "${SINGLEUSER_BUILD_MODE:-}" == "online" ]]; then
            force_mode="online"
        fi
        
        # 智能准备 Dockerfile
        prepare_singleuser_dockerfile "$service_path" "$network_env" "$force_mode"
    fi
    
    # 特殊处理nginx和jupyterhub的构建上下文
    local build_context
    if [[ "$service" == "nginx" ]]; then
        # nginx构建前先渲染模板
        print_info "  → nginx构建前渲染配置模板..."
        render_nginx_templates
        build_context="$SCRIPT_DIR"  # 使用项目根目录作为构建上下文
    elif [[ "$service" == "jupyterhub" ]]; then
        # jupyterhub构建前先渲染配置模板
        print_info "  → jupyterhub构建前渲染配置模板..."
        render_jupyterhub_templates
        build_context="$SCRIPT_DIR/$service_path"
    else
        build_context="$SCRIPT_DIR/$service_path"
    fi
    
    local dockerfile_name="Dockerfile"
    
    # 统一处理：所有服务都使用各自的src子目录作为构建上下文
    local target_arg=""
    if [[ "$service" == "backend-init" ]]; then
        target_arg="--target backend-init"
    elif [[ "$service" == "backend" ]]; then
        target_arg="--target backend"
    fi
    
    # 添加 --no-cache 参数（当启用强制重建时）
    local cache_arg=""
    if [[ "$FORCE_REBUILD" == "true" ]]; then
        cache_arg="--no-cache"
    fi
    
    # 计算服务哈希并准备构建标签
    local service_hash=$(calculate_service_hash "$service")
    local build_timestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ)
    
    local label_args=""
    label_args+="--label build.id=$build_id "
    label_args+="--label build.service=$service "
    label_args+="--label build.tag=$tag "
    label_args+="--label build.hash=$service_hash "
    label_args+="--label build.timestamp=$build_timestamp "
    label_args+="--label build.reason=$rebuild_reason "
    
    # 显示详细的构建信息
    print_info "  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    print_info "  📦 Docker 构建配置:"
    print_info "     Dockerfile: $dockerfile_path"
    print_info "     构建上下文: $build_context"
    if [[ -n "$target_arg" ]]; then
        print_info "     构建目标: ${target_arg#--target }"
    fi
    if [[ "$FORCE_REBUILD" == "true" ]]; then
        print_info "     缓存策略: --no-cache (强制重建)"
    else
        print_info "     缓存策略: 使用 Docker 层缓存"
    fi
    print_info "     目标镜像: $target_image"
    print_info "  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo
    print_info "  🔨 开始构建镜像..."
    echo
    
    # 使用各自的src子目录作为构建上下文
    # 直接显示 docker build 的完整输出，不做过滤
    if docker build -f "$dockerfile_path" $target_arg $cache_arg $label_args -t "$target_image" "$build_context"; then
        echo
        print_success "✓ 构建成功: $target_image"
        
        # 保存构建ID
        save_build_id "$build_id"
        
        # 保存服务构建信息
        save_service_build_info "$service" "$tag" "$build_id" "$service_hash"
        
        # 记录构建历史
        log_build_history "$build_id" "$service" "$tag" "SUCCESS" "$rebuild_reason"
        
        # 如果指定了registry，同时创建本地别名
        if [[ -n "$registry" ]] && [[ "$target_image" != "$base_image" ]]; then
            if docker tag "$target_image" "$base_image"; then
                print_info "  ✓ 本地别名: $base_image"
            fi
        fi
        
        # ========================================
        # SingleUser 构建后清理
        # ========================================
        if [[ "$service" == "singleuser" ]]; then
            print_info "  → 恢复 SingleUser Dockerfile 到原始状态..."
            restore_singleuser_dockerfile "$service_path"
        fi
        
        return 0
    else
        print_error "✗ 构建失败: $target_image"
        
        # 记录失败的构建
        log_build_history "$build_id" "$service" "$tag" "FAILED" "$rebuild_reason"
        
        # ========================================
        # SingleUser 构建失败时也需要清理
        # ========================================
        if [[ "$service" == "singleuser" ]]; then
            print_info "  → 构建失败，恢复 SingleUser Dockerfile 到原始状态..."
            restore_singleuser_dockerfile "$service_path"
        fi
        
        return 1
    fi
}

# 前端构建函数 - 已移除本地npm构建，现在使用Docker构建
# 这个函数已被废弃，前端现在使用标准的Docker构建流程
build_frontend() {
    print_error "此函数已废弃，前端现在使用Docker容器构建"
    return 1
}

# 批量预拉取所有服务的依赖镜像（带重试机制）
prefetch_all_base_images() {
    local max_retries="${1:-3}"  # 默认重试3次
    
    print_info "=========================================="
    print_info "🚀 批量预拉取所有服务的依赖镜像"
    print_info "=========================================="
    
    # 收集所有 Dockerfile 中的基础镜像
    local all_images=()
    local services_list=($SRC_SERVICES)
    
    print_info "📋 扫描所有服务的 Dockerfile..."
    
    for service in "${services_list[@]}"; do
        local service_path=$(get_service_path "$service")
        if [[ -z "$service_path" ]]; then
            continue
        fi
        
        local dockerfile_path="$SCRIPT_DIR/$service_path/Dockerfile"
        if [[ ! -f "$dockerfile_path" ]]; then
            continue
        fi
        
        # 提取该 Dockerfile 的基础镜像
        local images
        images=$(extract_base_images "$dockerfile_path")
        
        if [[ -n "$images" ]]; then
            while IFS= read -r image; do
                # 跳过空行
                if [[ -z "$image" ]]; then
                    continue
                fi
                # 跳过内部构建阶段
                if [[ "$image" =~ ^[a-z_-]+$ ]]; then
                    continue
                fi
                # 跳过注释
                if [[ "$image" =~ ^# ]]; then
                    continue
                fi
                # 添加到数组（去重将在后面处理）
                all_images+=("$image")
            done <<< "$images"
        fi
    done
    
    # 去重
    local unique_images=($(printf '%s\n' "${all_images[@]}" | sort -u))
    
    print_info "📦 发现 ${#unique_images[@]} 个唯一的基础镜像"
    echo
    
    # 统计变量
    local total=${#unique_images[@]}
    local pull_count=0
    local skip_count=0
    local fail_count=0
    local current=0
    
    # 遍历并拉取
    for image in "${unique_images[@]}"; do
        ((current++))
        print_info "[$current/$total] 检查镜像: $image"
        
        # 检查镜像是否已存在
        if docker image inspect "$image" >/dev/null 2>&1; then
            print_success "  ✓ 已存在，跳过"
            ((skip_count++))
            continue
        fi
        
        # 尝试拉取镜像（带重试机制）
        print_info "  ⬇ 正在拉取..."
        if pull_image_with_retry "$image" "$max_retries"; then
            print_success "  ✓ 拉取成功"
            ((pull_count++))
            
            # 拉取成功后自动创建双向tag（localhost/ 前缀 ↔ 原始名称）
            tag_image_bidirectional "$image" 2>/dev/null || true
        else
            print_error "  ✗ 拉取失败（已重试${max_retries}次）"
            ((fail_count++))
        fi
        
        echo
    done
    
    # 输出最终统计
    print_info "=========================================="
    print_info "📊 预拉取完成统计"
    print_info "=========================================="
    print_info "  • 总镜像数: $total"
    print_info "  • 新拉取: $pull_count"
    print_info "  • 已存在: $skip_count"
    
    if [[ $fail_count -gt 0 ]]; then
        print_error "  • 失败: $fail_count (已重试${max_retries}次)"
        print_error "=========================================="
        print_error "❌ 基础镜像预拉取失败"
        print_error "=========================================="
        print_error "部分关键镜像无法下载，无法继续构建。"
        print_error ""
        print_error "失败的镜像数量: $fail_count"
        print_error "已重试次数: $max_retries"
        print_error ""
        print_error "可能的原因："
        print_error "  1. 网络连接问题"
        print_error "  2. Docker Hub 访问受限"
        print_error "  3. 镜像名称或标签错误"
        print_error ""
        print_error "解决方案："
        print_error "  1. 检查网络连接: ping mirrors.aliyun.com"
        print_error "  2. 配置 Docker 镜像加速器"
        print_error "  3. 手动拉取失败的镜像验证"
        print_error "  4. 使用 VPN 或代理"
        print_error ""
        print_error "构建已终止，请解决镜像拉取问题后重试。"
        echo
        return 1  # 返回失败，终止构建
    else
        print_success "✅ 所有依赖镜像已就绪！"
    fi
    
    echo
    return 0  # 返回成功，继续构建
}

# 构建所有服务镜像
build_all_services() {
    local tag="${1:-$DEFAULT_IMAGE_TAG}"
    local registry="${2:-}"
    
    print_info "=========================================="
    print_info "构建所有 AI-Infra 服务镜像"
    print_info "=========================================="
    print_info "镜像标签: $tag"
    if [[ -n "$registry" ]]; then
        print_info "目标仓库: $registry"
    else
        print_info "目标仓库: 本地构建"
    fi
    echo
    
    # ========================================
    # 步骤 -1: 环境检测和配置生成（自动化）
    # ========================================
    print_info "=========================================="
    print_info "步骤 -1/5: 环境检测和配置生成"
    print_info "=========================================="
    
    # 自动检测网络环境并生成/更新 .env 文件
    generate_or_update_env_file
    
    # ========================================
    # 步骤 0: 检查当前构建状态（需求32）
    # ========================================
    if [[ "$FORCE_REBUILD" == "false" ]]; then
        print_info "=========================================="
        print_info "步骤 0/6: 检查当前构建状态"
        print_info "=========================================="
        
        # 显示构建状态
        show_build_status "$tag" "$registry"
        
        # 获取需要构建的服务列表
        local services_to_build
        services_to_build=$(get_services_to_build "$tag" "$registry")
        
        if [[ -z "$services_to_build" ]]; then
            print_success "=========================================="
            print_success "✅ 所有服务镜像都已成功构建"
            print_success "=========================================="
            print_info "如需强制重建，请使用 --force 参数"
            return 0
        fi
        
        # 将字符串转换为数组
        local services_array=($services_to_build)
        local need_build_count=${#services_array[@]}
        
        print_info "📋 需要构建的服务数量: $need_build_count"
        print_info "服务列表: $services_to_build"
        echo
        
        # 更新要构建的服务列表
        BUILD_SERVICES="$services_to_build"
    else
        print_info "强制重建模式：将重新构建所有服务"
        BUILD_SERVICES="$SRC_SERVICES"
    fi
    echo
    
    # ========================================
    # 步骤 1: 智能镜像管理（拉取 + Tag）
    # ========================================
    print_info "=========================================="
    print_info "步骤 1/6: 智能镜像管理（拉取 + Tag）"
    print_info "=========================================="
    
    # 自动检测网络环境
    local network_env=$(detect_network_environment)
    print_info "🌐 检测到网络环境: $network_env"
    
    # 获取 Harbor 仓库地址
    local harbor_registry="${INTERNAL_REGISTRY:-aiharbor.msxf.local/aihpc}"
    if [[ "$network_env" == "internal" ]]; then
        print_info "📦 内网 Harbor 仓库: $harbor_registry"
    fi
    echo
    
    # 收集所有需要处理的镜像
    local all_images=()
    
    # 1. 从所有 Dockerfile 中提取基础镜像
    print_info "📋 步骤 1.1: 扫描 Dockerfile 中的基础镜像..."
    local services_list=($SRC_SERVICES)
    local dockerfile_count=0
    
    for service in "${services_list[@]}"; do
        local service_path=$(get_service_path "$service")
        if [[ -z "$service_path" ]]; then
            continue
        fi
        
        local dockerfile_path="$SCRIPT_DIR/$service_path/Dockerfile"
        if [[ ! -f "$dockerfile_path" ]]; then
            continue
        fi
        
        ((dockerfile_count++))
        
        # 提取该 Dockerfile 的基础镜像
        local images
        images=$(extract_base_images "$dockerfile_path")
        
        if [[ -n "$images" ]]; then
            while IFS= read -r image; do
                # 跳过空行、内部阶段、注释
                if [[ -z "$image" ]] || [[ "$image" =~ ^[a-z_-]+$ ]] || [[ "$image" =~ ^# ]]; then
                    continue
                fi
                all_images+=("$image")
            done <<< "$images"
        fi
    done
    
    print_info "  ✓ 扫描了 $dockerfile_count 个 Dockerfile"
    
    # 2. 从 docker-compose.yml 中提取第三方镜像
    print_info "📋 步骤 1.2: 扫描 docker-compose.yml 中的第三方镜像..."
    local compose_image_count=0
    
    if [[ -f "$SCRIPT_DIR/docker-compose.yml" ]]; then
        local compose_images=$(grep -E '^\s*image:' "$SCRIPT_DIR/docker-compose.yml" | \
            grep -v '\$' | \
            awk '{print $2}' | \
            sort -u)
        
        if [[ -n "$compose_images" ]]; then
            while IFS= read -r image; do
                if [[ -z "$image" ]]; then
                    continue
                fi
                
                # 跳过本项目构建的镜像（ai-infra-开头）
                if [[ "$image" =~ ^ai-infra- ]]; then
                    continue
                fi
                
                all_images+=("$image")
                ((compose_image_count++))
                print_info "  → 发现: $image"
            done <<< "$compose_images"
        fi
    fi
    
    print_info "  ✓ 发现 $compose_image_count 个第三方镜像"
    echo
    
    # 去重并排序
    local unique_images=($(printf '%s\n' "${all_images[@]}" | sort -u))
    local total_images=${#unique_images[@]}
    
    if [[ $total_images -eq 0 ]]; then
        print_warning "⚠️  未发现需要处理的镜像"
        echo
    else
        print_info "📊 汇总统计:"
        print_info "  • 发现唯一镜像: $total_images 个"
        print_info "  • 网络环境: $network_env"
        echo
        
        # 3. 批量智能处理镜像（拉取 + Tag）
        print_info "🔄 步骤 1.3: 批量处理镜像（拉取 + Tag）..."
        print_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
        echo
        
        # 调用智能 tag 函数（会自动处理拉取、降级、创建别名）
        if batch_tag_images_smart "$network_env" "$harbor_registry" "${unique_images[@]}"; then
            print_success "✅ 所有镜像处理成功"
        else
            print_warning "⚠️  部分镜像处理失败，但构建流程将继续"
            print_info "💡 提示: 构建可能会因缺少基础镜像而失败"
        fi
    fi
    
    print_success "✓ 智能镜像管理完成"
    echo

    # ========================================
    # 步骤 2: 同步配置文件
    # ========================================
    print_info "=========================================="
    print_info "步骤 2/6: 同步配置文件"
    print_info "=========================================="
    if sync_all_configs; then
        print_success "✓ 配置文件同步完成"
    else
        print_warning "配置文件同步过程中有警告，但构建流程将继续"
    fi
    echo

    # ========================================
    # 步骤 3: 渲染配置模板
    # ========================================
    print_info "=========================================="
    print_info "步骤 3/6: 渲染配置模板"
    print_info "=========================================="
    
    # 渲染 Nginx 配置模板
    print_info "渲染 Nginx 配置模板..."
    if render_nginx_templates; then
        print_success "✓ Nginx 模板渲染完成"
    else
        print_warning "Nginx 模板渲染失败，但构建流程将继续"
    fi
    
    # 渲染 JupyterHub 配置模板
    print_info "渲染 JupyterHub 配置模板..."
    if render_jupyterhub_templates; then
        print_success "✓ JupyterHub 模板渲染完成"
    else
        print_warning "JupyterHub 模板渲染失败，但构建流程将继续"
    fi
    
    # 渲染 Docker Compose 配置模板（如果需要）
    if [[ -f "$SCRIPT_DIR/docker-compose.yml.example" ]]; then
        print_info "渲染 Docker Compose 配置模板..."
        if render_docker_compose_templates "$registry" "$tag"; then
            print_success "✓ Docker Compose 模板渲染完成"
        else
            print_warning "Docker Compose 模板渲染失败，但构建流程将继续"
        fi
    fi
    
    print_success "✓ 所有模板渲染完成"
    echo
    
    # ========================================
    # ========================================
    # 步骤 4: 构建服务镜像（智能过滤）
    # ========================================
    print_info "=========================================="
    print_info "步骤 4/6: 构建服务镜像"
    print_info "=========================================="
    
    local success_count=0
    local total_count=0
    local failed_services=()
    
    # 使用智能过滤的服务列表（步骤0中设置的BUILD_SERVICES）
    local all_services="${BUILD_SERVICES:-$SRC_SERVICES}"
    
    # 计算服务总数
    for service in $all_services; do
        total_count=$((total_count + 1))
    done
    
    print_info "准备构建 $total_count 个服务"
    echo
    
    # 构建所有服务
    for service in $all_services; do
        print_info "构建服务: $service"
        if build_service "$service" "$tag" "$registry"; then
            success_count=$((success_count + 1))
        else
            failed_services+=("$service")
        fi
        echo
    done
    
    # ========================================
    # 步骤 5: 验证构建结果（需求32）
    # ========================================
    print_info "=========================================="
    print_info "步骤 5/6: 验证构建结果"
    print_info "=========================================="
    
    # 显示最终构建状态
    show_build_status "$tag" "$registry"
    
    print_info "=========================================="
    print_success "构建完成: $success_count/$total_count 成功"
    
    # SLURM包已集成到apphub多阶段构建中，无需单独复制
    # apphub现在包含完整的工具链和SLURM deb包
    
    if [[ ${#failed_services[@]} -gt 0 ]]; then
        print_warning "失败的服务: ${failed_services[*]}"
        return 1
    else
        print_success "🎉 所有服务构建成功！"
        return 0
    fi
}

# 组合式一键构建流程
# 用法: build_all_pipeline [tag] [registry]
# 行为:
#  1) 生成/刷新 .env（等价于: create-env dev [--force]）
#  2) 同步配置（等价于: sync-config [--force]）
#  3) 构建所有服务镜像（等价于: build-all [tag] [registry]）
build_all_pipeline() {
    local tag="${1:-$DEFAULT_IMAGE_TAG}"
    local registry="${2:-}"

    # 是否强制模式：沿用全局 FORCE_REBUILD（由 --force 开关控制）
    local force="false"
    if [[ "$FORCE_REBUILD" == "true" ]]; then
        force="true"
    fi

    print_info "=========================================="
    print_info "准备环境配置（create-env dev）"
    print_info "=========================================="
    # 自动检测网络环境（内网/外网），导出并写入.env，供后续步骤使用
    local NETWORK_ENV_DETECTED
    NETWORK_ENV_DETECTED=$(detect_network_environment)
    export AI_INFRA_NETWORK_ENV="$NETWORK_ENV_DETECTED"
    print_info "网络环境检测: $AI_INFRA_NETWORK_ENV"
    # 在.env中记录，便于模板/服务识别
    set_or_update_env_var "AI_INFRA_NETWORK_ENV" "$AI_INFRA_NETWORK_ENV" "$SCRIPT_DIR/.env" || true

    if ! create_env_from_template "dev" "$force"; then
        print_error "创建/渲染 .env 失败，停止构建"
        return 1
    fi
    # 再次写入AI_INFRA_NETWORK_ENV，确保在渲染.env之后持久化
    set_or_update_env_var "AI_INFRA_NETWORK_ENV" "$AI_INFRA_NETWORK_ENV" "$SCRIPT_DIR/.env" || true

    print_info "=========================================="
    print_info "同步配置（sync-config）"
    print_info "=========================================="
    if ! sync_all_configs "$force"; then
        print_error "同步配置失败，停止构建"
        return 1
    fi

    # 渲染模板（nginx/docker-compose），确保以源模板为准进行生成
    print_info "=========================================="
    print_info "渲染配置模板（nginx / docker-compose）"
    print_info "=========================================="
    render_nginx_templates || print_warning "Nginx 模板渲染出现问题，请稍后检查"
    render_docker_compose_templates "$registry" "$tag" || print_warning "Docker Compose 模板渲染出现问题，请稍后检查"

    print_info "=========================================="
    print_info "开始构建所有服务（build-all）"
    print_info "标签: $tag  仓库: ${registry:-<本地>}  强制: $force"
    print_info "=========================================="
    if ! build_all_services "$tag" "$registry"; then
        print_error "构建所有服务失败"
        return 1
    fi

    # 尝试启动（或重启）服务
    local compose_cmd
    compose_cmd=$(detect_compose_command || true)
    if [[ -n "$compose_cmd" ]]; then
        print_info "=========================================="
        print_info "启动（或重启）Docker Compose 服务"
        print_info "=========================================="
        # 优先验证配置
        if $compose_cmd -f "$SCRIPT_DIR/docker-compose.yml" config --quiet 2>/dev/null; then
            # 尝试优雅重启
            $compose_cmd down 2>/dev/null || true
            if $compose_cmd up -d; then
                print_success "✓ 服务已启动"
            else
                print_warning "⚠ 启动失败，请手动检查 docker compose 日志"
            fi
        else
            print_warning "⚠ docker-compose.yml 验证失败，请检查模板源文件和渲染逻辑"
        fi
    else
        print_warning "未检测到 Docker Compose 命令，跳过启动步骤"
    fi

    print_success "✓ 一键构建流程完成"
}

# 推送单个服务镜像
push_service() {
    local service="$1"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    local registry="$3"
    
    if [[ -z "$registry" ]]; then
        print_error "推送操作需要指定 registry"
        return 1
    fi
    
    local base_image="ai-infra-${service}:${tag}"
    local target_image=$(get_private_image_name "$base_image" "$registry")
    
    print_info "推送服务: $service"
    print_info "  原始镜像: $base_image"
    print_info "  目标镜像: $target_image"
    print_info "  Registry: $registry"
    
    # 检查镜像是否存在
    if ! docker image inspect "$base_image" >/dev/null 2>&1; then
        print_warning "本地镜像不存在: $base_image"
        print_info "尝试构建镜像..."
        if ! build_service "$service" "$tag" "$registry"; then
            print_error "构建失败，无法推送"
            return 1
        fi
    else
        print_success "✓ 本地镜像存在: $base_image"
    fi
    
    # 如果需要标记为目标镜像
    if [[ "$base_image" != "$target_image" ]]; then
        print_info "标记镜像: $base_image -> $target_image"
        if ! docker tag "$base_image" "$target_image"; then
            print_error "镜像标记失败"
            return 1
        fi
    fi
    
    # 推送镜像
    print_info "推送镜像: $target_image"
    if docker push "$target_image"; then
        print_success "✓ 推送成功: $target_image"
        return 0
    else
        print_error "✗ 推送失败: $target_image"
        return 1
    fi
}

# 推送所有服务镜像
push_all_services() {
    local tag="${1:-$DEFAULT_IMAGE_TAG}"
    local registry="$2"
    
    if [[ -z "$registry" ]]; then
        print_error "推送操作需要指定 registry"
        print_info "用法: $0 push-all <registry> [tag]"
        return 1
    fi
    
    print_info "=========================================="
    print_info "推送所有 AI-Infra 服务镜像"
    print_info "=========================================="
    print_info "目标仓库: $registry"
    print_info "镜像标签: $tag"
    echo
    
    local success_count=0
    local total_count=0
    local failed_services=()
    
    # 计算服务总数
    for service in $SRC_SERVICES; do
        total_count=$((total_count + 1))
    done
    
    for service in $SRC_SERVICES; do
        if push_service "$service" "$tag" "$registry"; then
            success_count=$((success_count + 1))
        else
            failed_services+=("$service")
        fi
        echo
    done
    
    print_info "=========================================="
    print_success "推送完成: $success_count/$total_count 成功"
    
    if [[ ${#failed_services[@]} -gt 0 ]]; then
        print_warning "失败的服务: ${failed_services[*]}"
        return 1
    else
        print_success "🚀 所有服务推送成功！"
        return 0
    fi
}

# 一键构建并推送
build_and_push_all() {
    # 处理帮助参数
    if [[ "$1" == "--help" || "$1" == "-h" ]]; then
        echo "build-push - 一键构建并推送所有服务"
        echo
        echo "用法: $0 build-push <registry> [tag]"
        echo
        echo "参数:"
        echo "  registry    目标仓库地址 (必需)"
        echo "  tag         镜像标签 (默认: $DEFAULT_IMAGE_TAG)"
        echo
        echo "说明:"
        echo "  自动化构建和推送所有AI-Infra服务，包括："
        echo "  • 第一阶段：构建所有服务镜像"
        echo "  • 第二阶段：推送所有镜像到目标仓库"
        echo "  • 错误处理和进度报告"
        echo "  • 完整的构建推送流程"
        echo
        echo "构建服务: $SRC_SERVICES"
        echo
        echo "示例:"
        echo "  $0 build-push harbor.company.com/ai-infra v1.0.0"
        echo "  $0 build-push registry.internal.com/project v0.3.6-dev"
        return 0
    fi
    
    local tag="${1:-$DEFAULT_IMAGE_TAG}"
    local registry="$2"
    
    if [[ -z "$registry" ]]; then
        print_error "一键构建推送需要指定 registry"
        print_info "用法: $0 build-push <registry> [tag]"
        return 1
    fi
    
    print_info "=========================================="
    print_info "一键构建并推送所有服务"
    print_info "=========================================="
    print_info "目标仓库: $registry"
    print_info "镜像标签: $tag"
    echo
    
    # 第一阶段：构建所有镜像
    print_info "🔨 第一阶段：构建所有镜像..."
    if ! build_all_services "$tag" "$registry"; then
        print_error "构建阶段失败，停止执行"
        return 1
    fi
    
    echo
    print_info "🚀 第二阶段：推送所有镜像..."
    if ! push_all_services "$tag" "$registry"; then
        print_error "推送阶段失败"
        return 1
    fi
    
    print_success "🎉 一键构建推送完成！"
}

# 拉取并标记依赖镜像
pull_and_tag_dependencies() {
    local registry="$1"
    local tag="${2:-latest}"
    
    if [[ -z "$registry" ]]; then
        print_error "需要指定 registry"
        print_info "用法: $0 deps-pull <registry> [tag]"
        return 1
    fi
    
    print_info "=========================================="
    print_info "拉取并标记依赖镜像到 $registry"
    print_info "=========================================="
    print_info "目标镜像标签: $tag (所有依赖镜像将统一使用此版本标签)"
    
    # 动态收集依赖镜像
    local dependency_images
    dependency_images=$(collect_dependency_images)
    print_info "收集到依赖镜像: $dependency_images"
    echo
    
    local success_count=0
    local total_count=0
    local failed_deps=()
    
    for dep_image in $dependency_images; do
        total_count=$((total_count + 1))
        print_info "处理依赖镜像: $dep_image"
        
        # 使用新的映射机制生成目标镜像名
        local target_image
        target_image=$(get_mapped_private_image "$dep_image" "$registry" "$tag")
        
        # 检查目标镜像是否已存在
        if [[ "$FORCE_REBUILD" == "false" ]] && docker image inspect "$target_image" >/dev/null 2>&1; then
            print_success "  ✓ 镜像已存在，跳过: $target_image"
            success_count=$((success_count + 1))
            continue
        fi
        
        # 检查原始镜像是否已存在本地
        if docker image inspect "$dep_image" >/dev/null 2>&1; then
            print_success "  ✓ 本地镜像已存在: $dep_image"
        else
            # 拉取原始镜像
            print_info "  → 正在拉取镜像: $dep_image"
            if ! docker pull "$dep_image"; then
                print_error "  ✗ 拉取失败: $dep_image"
                failed_deps+=("$dep_image")
                continue
            fi
            print_success "  ✓ 拉取成功: $dep_image"
        fi
        
        # 标记镜像
        if docker tag "$dep_image" "$target_image"; then
            print_success "  ✓ 标记成功: $target_image"
            success_count=$((success_count + 1))
        else
            print_error "  ✗ 标记失败: $target_image"
            failed_deps+=("$dep_image")
        fi
        echo
    done
    
    print_info "=========================================="
    print_success "依赖镜像处理完成: $success_count/$total_count 成功"
    
    if [[ ${#failed_deps[@]} -gt 0 ]]; then
        print_warning "失败的依赖镜像: ${failed_deps[*]}"
        return 1
    else
        print_success "🎉 所有依赖镜像处理成功！"
        return 0
    fi
}

# 推送依赖镜像
push_dependencies() {
    local registry="$1"
    local tag="${2:-latest}"
    
    if [[ -z "$registry" ]]; then
        print_error "需要指定 registry"
        print_info "用法: $0 deps-push <registry> [tag]"
        return 1
    fi
    
    print_info "=========================================="
    print_info "推送依赖镜像到 $registry"
    print_info "=========================================="
    print_info "目标镜像标签: $tag (所有依赖镜像将统一使用此版本标签)"
    
    # 动态收集依赖镜像
    local dependency_images
    dependency_images=$(collect_dependency_images)
    print_info "收集到依赖镜像: $dependency_images"
    echo
    
    local success_count=0
    local total_count=0
    local failed_deps=()
    
    for dep_image in $dependency_images; do
        total_count=$((total_count + 1))
        
        # 使用新的映射机制生成目标镜像名
        local target_image
        target_image=$(get_mapped_private_image "$dep_image" "$registry" "$tag")
        
        print_info "推送依赖镜像: $target_image"
        
        if docker push "$target_image"; then
            print_success "  ✓ 推送成功: $target_image"
            success_count=$((success_count + 1))
        else
            print_error "  ✗ 推送失败: $target_image"
            failed_deps+=("$target_image")
        fi
        echo
    done
    
    print_info "=========================================="
    print_success "依赖镜像推送完成: $success_count/$total_count 成功"
    
    if [[ ${#failed_deps[@]} -gt 0 ]]; then
        print_warning "失败的依赖镜像: ${failed_deps[*]}"
        return 1
    else
        print_success "🎉 所有依赖镜像推送成功！"
        return 0
    fi
}

# ==========================================
# 生产环境依赖镜像处理功能
# ==========================================

# 拉取并标记生产环境依赖镜像（排除测试工具）
pull_and_tag_production_dependencies() {
    local registry="$1"
    local tag="${2:-latest}"
    
    if [[ -z "$registry" ]]; then
        print_error "需要指定 registry"
        return 1
    fi
    
    print_info "=========================================="
    print_info "拉取并标记生产环境依赖镜像到 $registry"
    print_info "=========================================="
    print_info "目标镜像标签: $tag (所有生产环境依赖镜像将统一使用此版本标签)"
    
    # 使用生产环境依赖镜像列表
    local dependency_images
    dependency_images=$(get_production_dependencies | tr '\n' ' ')
    print_info "收集到生产环境依赖镜像: $dependency_images"
    echo
    
    local success_count=0
    local total_count=0
    local failed_deps=()
    
    for dep_image in $dependency_images; do
        if [[ -z "$dep_image" ]]; then
            continue
        fi
        
        ((total_count++))
        
        # 获取目标镜像名称
        local target_image
        target_image=$(get_mapped_private_image "$dep_image" "$registry" "$tag")
        
        # 检查镜像是否已存在
        if docker image inspect "$target_image" >/dev/null 2>&1; then
            print_success "  ✓ 镜像已存在，跳过: $target_image"
            ((success_count++))
            continue
        fi
        
        print_info "处理生产环境依赖镜像: $dep_image"
        
        # 拉取原始镜像
        if ! docker pull "$dep_image"; then
            print_error "  ✗ 拉取失败: $dep_image"
            failed_deps+=("$dep_image")
            continue
        fi
        
        # 标记为目标镜像
        if ! docker tag "$dep_image" "$target_image"; then
            print_error "  ✗ 标记失败: $dep_image -> $target_image"
            failed_deps+=("$dep_image")
            continue
        fi
        
        print_success "  ✓ 处理成功: $dep_image -> $target_image"
        ((success_count++))
    done
    you y
    print_info "=========================================="
    print_success "生产环境依赖镜像处理完成: $success_count/$total_count 成功"
    
    if [[ ${#failed_deps[@]} -gt 0 ]]; then
        print_warning "失败的依赖镜像: ${failed_deps[*]}"
        return 1
    else
        print_success "🎉 所有生产环境依赖镜像处理成功！"
        return 0
    fi
}

# 推送生产环境依赖镜像
push_production_dependencies() {
    local registry="$1"
    local tag="${2:-latest}"
    
    if [[ -z "$registry" ]]; then
        print_error "需要指定 registry"
        return 1
    fi
    
    print_info "=========================================="
    print_info "推送生产环境依赖镜像到 $registry"
    print_info "=========================================="
    print_info "源镜像标签: $tag (如果为latest则会映射到v0.3.6-dev)"
    
    # 使用生产环境依赖镜像列表
    local dependency_images
    dependency_images=$(get_production_dependencies | tr '\n' ' ')
    print_info "收集到生产环境依赖镜像: $dependency_images"
    echo
    
    local success_count=0
    local total_count=0
    local failed_deps=()
    
    for dep_image in $dependency_images; do
        if [[ -z "$dep_image" ]]; then
            continue
        fi
        
        ((total_count++))
        
        # 获取目标镜像名称
        local target_image
        target_image=$(get_mapped_private_image "$dep_image" "$registry" "$tag")
        
        print_info "推送生产环境依赖镜像: $target_image"
        
        if docker push "$target_image"; then
            print_success "  ✓ 推送成功: $target_image"
            ((success_count++))
        else
            print_error "  ✗ 推送失败: $target_image"
            failed_deps+=("$target_image")
        fi
    done
    
    print_info "=========================================="
    print_success "生产环境依赖镜像推送完成: $success_count/$total_count 成功"
    
    if [[ ${#failed_deps[@]} -gt 0 ]]; then
        print_warning "失败的依赖镜像: ${failed_deps[*]}"
        return 1
    else
        print_success "🎉 所有生产环境依赖镜像推送成功！"
        return 0
    fi
}

# 推送构建依赖镜像（仅包含构建时需要的镜像）
push_build_dependencies() {
    local registry="$1"
    local tag="${2:-latest}"
    
    if [[ -z "$registry" ]]; then
        print_error "需要指定 registry"
        print_info "用法: $0 build-deps-push <registry> [tag]"
        return 1
    fi
    
    print_info "=========================================="
    print_info "推送构建依赖镜像到 $registry"
    print_info "=========================================="
    print_info "目标镜像标签: $tag"
    
    # 定义构建依赖镜像
    local build_dependencies=(
        "node:22-alpine"
        "nginx:stable-alpine-perl"
        "golang:1.25-alpine"
        "python:3.13-alpine"
        "gitea/gitea:1.24.6"
        "jupyter/base-notebook:latest"
    )
    
    local success_count=0
    local total_count=${#build_dependencies[@]}
    local failed_deps=()
    
    for dep_image in "${build_dependencies[@]}"; do
        # 使用新的映射机制生成目标镜像名
        local target_image
        target_image=$(get_mapped_private_image "$dep_image" "$registry" "$tag")
        
        print_info "推送构建依赖镜像: $target_image"
        
        if docker push "$target_image"; then
            print_success "  ✓ 推送成功: $target_image"
            success_count=$((success_count + 1))
        else
            print_error "  ✗ 推送失败: $target_image"
            failed_deps+=("$target_image")
        fi
        echo
    done
    
    print_info "=========================================="
    print_success "构建依赖镜像推送完成: $success_count/$total_count 成功"
    
    if [[ ${#failed_deps[@]} -gt 0 ]]; then
        print_warning "失败的构建依赖镜像: ${failed_deps[*]}"
        return 1
    else
        print_success "🎉 所有构建依赖镜像推送成功！"
        return 0
    fi
}

# ==========================================
# AI Harbor 镜像拉取管理
# ==========================================

# 从 AI Harbor 拉取所有服务镜像
pull_aiharbor_services() {
    local registry="${1:-aiharbor.msxf.local/aihpc}"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    
    print_info "=========================================="
    print_info "🚢 从 AI Harbor 拉取服务镜像"
    print_info "=========================================="
    print_info "Harbor地址: $registry"
    print_info "镜像标签: $tag"
    echo
    
    local services=("backend" "frontend" "jupyterhub" "nginx" "saltstack" "singleuser" "gitea" "backend-init")
    local success_count=0
    local total_count=${#services[@]}
    local failed_services=()
    
    for service in "${services[@]}"; do
        local harbor_image="${registry}/ai-infra-${service}:${tag}"
        local local_image="ai-infra-${service}:${tag}"
        
        print_info "→ 拉取服务: $service"
        print_info "  Harbor镜像: $harbor_image"
        print_info "  本地标签: $local_image"
        
        # 尝试拉取镜像
        if docker pull "$harbor_image"; then
            print_success "  ✓ 拉取成功: $harbor_image"
            
            # 标记为本地镜像名
            if docker tag "$harbor_image" "$local_image"; then
                print_success "  ✓ 标记为本地镜像: $local_image"
                success_count=$((success_count + 1))
            else
                print_error "  ✗ 标记失败: $local_image"
                failed_services+=("$service")
            fi
        else
            print_error "  ✗ 拉取失败: $harbor_image"
            failed_services+=("$service")
        fi
        echo
    done
    
    print_info "=========================================="
    print_success "拉取完成: $success_count/$total_count 成功"
    
    if [[ ${#failed_services[@]} -gt 0 ]]; then
        print_warning "失败的服务: ${failed_services[*]}"
        print_info "可以尝试以下操作:"
        print_info "1. 检查 Harbor 仓库访问权限"
        print_info "2. 验证镜像标签是否存在: $tag"
        print_info "3. 确认网络连接正常"
        return 1
    else
        print_success "🚀 所有AI-Infra服务镜像拉取成功！"
        print_info "现在可以使用本地镜像启动服务："
        print_info "  docker compose -f docker-compose.yml.example up -d"
        return 0
    fi
}

# 从 AI Harbor 拉取依赖镜像  
pull_aiharbor_dependencies() {
    local registry="${1:-aiharbor.msxf.local/aihpc}"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    
    print_info "=========================================="
    print_info "🚢 从 AI Harbor 拉取依赖镜像"
    print_info "=========================================="
    print_info "Harbor地址: $registry"
    print_info "镜像标签: $tag"
    echo
    
    # 从配置文件或预定义列表收集依赖镜像
    local dependency_images=$(get_all_dependencies | tr '\n' ' ')
    if [[ -z "$dependency_images" ]]; then
        dependency_images="postgres:15-alpine redis:7-alpine nginx:1.27-alpine tecnativa/tcp-proxy minio/minio:latest osixia/openldap:stable osixia/phpldapadmin:stable redislabs/redisinsight:latest node:22-alpine nginx:stable-alpine-perl golang:1.25-alpine python:3.13-alpine gitea/gitea:1.24.6 jupyter/base-notebook:latest"
    fi
    
    print_info "依赖镜像列表: $dependency_images"
    echo
    
    local success_count=0
    local total_count=0
    local failed_deps=()
    
    for dep_image in $dependency_images; do
        if [[ -z "$dep_image" ]]; then
            continue
        fi
        
        ((total_count++))
        
        # 获取映射后的Harbor镜像名称
        local harbor_image
        harbor_image=$(get_mapped_private_image "$dep_image" "$registry" "$tag")
        
        print_info "→ 拉取依赖: $(basename "$dep_image")"
        print_info "  Harbor镜像: $harbor_image"
        print_info "  原始镜像: $dep_image"
        
        # 尝试拉取Harbor镜像
        if docker pull "$harbor_image"; then
            print_success "  ✓ 拉取成功: $harbor_image"
            
            # 标记为原始镜像名
            if docker tag "$harbor_image" "$dep_image"; then
                print_success "  ✓ 标记为原始镜像: $dep_image"
                success_count=$((success_count + 1))
            else
                print_error "  ✗ 标记失败: $dep_image"
                failed_deps+=("$dep_image")
            fi
        else
            print_warning "  ! Harbor拉取失败，尝试官方源: $dep_image"
            # 回退到官方镜像拉取
            if docker pull "$dep_image"; then
                print_success "  ✓ 从官方源拉取成功: $dep_image"
                success_count=$((success_count + 1))
            else
                print_error "  ✗ 所有源都拉取失败: $dep_image"
                failed_deps+=("$dep_image")
            fi
        fi
        echo
    done
    
    print_info "=========================================="
    print_success "依赖镜像拉取完成: $success_count/$total_count 成功"
    
    if [[ ${#failed_deps[@]} -gt 0 ]]; then
        print_warning "失败的依赖镜像: ${failed_deps[*]}"
        return 1
    else
        print_success "🚀 所有依赖镜像拉取成功！"
        return 0
    fi
}

# 从 AI Harbor 拉取所有镜像（服务+依赖）
pull_aiharbor_all() {
    local registry="${1:-aiharbor.msxf.local/aihpc}"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    
    print_info "=========================================="
    print_info "🚢 从 AI Harbor 拉取所有镜像"
    print_info "=========================================="
    print_info "Harbor地址: $registry"
    print_info "镜像标签: $tag"
    echo
    
    local overall_success=true
    
    # 先拉取依赖镜像
    print_info "步骤 1/2: 拉取依赖镜像..."
    if ! pull_aiharbor_dependencies "$registry" "$tag"; then
        print_warning "部分依赖镜像拉取失败，但继续拉取服务镜像..."
        overall_success=false
    fi
    
    echo
    print_info "步骤 2/2: 拉取服务镜像..."
    if ! pull_aiharbor_services "$registry" "$tag"; then
        print_error "服务镜像拉取失败"
        overall_success=false
    fi
    
    echo
    print_info "=========================================="
    if [[ "$overall_success" == "true" ]]; then
        print_success "🎉 所有镜像拉取完成！"
        print_info ""
        print_info "接下来可以："
        print_info "1. 启动服务: docker compose -f docker-compose.yml.example up -d"
        print_info "2. 查看状态: ./build.sh prod-status"
        return 0
    else
        print_warning "⚠️  部分镜像拉取失败，请检查错误信息"
        print_info "建议操作："
        print_info "1. 检查Harbor访问权限和网络连接"
        print_info "2. 验证镜像标签 $tag 是否存在"
        print_info "3. 重新运行失败的拉取命令"
        return 1
    fi
}

# ==========================================
# 双环境部署支持功能
# ==========================================

# 创建生产环境配置文件 (.env.prod)
create_production_env() {
    local mode="${1:-production}"  # production 或 intranet
    local registry="${2:-aiharbor.msxf.local/aihpc}"
    local tag="${3:-$DEFAULT_IMAGE_TAG}"
    
    local env_file=".env.prod"
    local template_file=".env.example"
    
    print_info "创建生产环境配置文件: $env_file"
    print_info "模式: $mode"
    print_info "镜像仓库: $registry"
    print_info "镜像标签: $tag"
    
    # 检查模板文件
    if [[ ! -f "$template_file" ]]; then
        print_error "模板文件不存在: $template_file"
        return 1
    fi
    
    # 复制模板文件
    cp "$template_file" "$env_file"
    
    # 根据模式配置不同的参数
    case "$mode" in
        "build"|"builder")
            # 构建环境配置
            sed -i.bak \
                -e "s|^IMAGE_TAG=.*|IMAGE_TAG=$tag|" \
                -e "s|^PRIVATE_REGISTRY=.*|PRIVATE_REGISTRY=$registry|" \
                -e "s|^BUILD_ENV=.*|BUILD_ENV=production|" \
                -e "s|^DEBUG_MODE=.*|DEBUG_MODE=false|" \
                -e "s|^LOG_LEVEL=.*|LOG_LEVEL=info|" \
                -e "s|^ENV_FILE=.*|ENV_FILE=.env.prod|" \
                -e "s|^DOMAIN=.*|DOMAIN=ai-infra.local|" \
                "$env_file"
            ;;
        "intranet"|"runtime")
            # 内网运行环境配置
            sed -i.bak \
                -e "s|^IMAGE_TAG=.*|IMAGE_TAG=$tag|" \
                -e "s|^PRIVATE_REGISTRY=.*|PRIVATE_REGISTRY=$registry|" \
                -e "s|^BUILD_ENV=.*|BUILD_ENV=production|" \
                -e "s|^DEBUG_MODE=.*|DEBUG_MODE=false|" \
                -e "s|^LOG_LEVEL=.*|LOG_LEVEL=info|" \
                -e "s|^ENV_FILE=.*|ENV_FILE=.env.prod|" \
                -e "s|^DOMAIN=.*|DOMAIN=ai-infra.local|" \
                "$env_file"
            ;;
        *)
            print_error "不支持的模式: $mode"
            print_info "支持的模式: build, intranet"
            return 1
            ;;
    esac
    
    # 删除备份文件
    rm -f "${env_file}.bak"
    
    print_success "✓ 已创建生产环境配置: $env_file"
    print_info "请根据实际环境调整配置文件中的参数"
    
    return 0
}

# 构建环境模式 - 构建并推送所有镜像
build_environment_deploy() {
    local registry="${1:-aiharbor.msxf.local/aihpc}"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    
    print_info "=========================================="
    print_info "构建环境部署模式"
    print_info "=========================================="
    print_info "镜像仓库: $registry"
    print_info "镜像标签: $tag"
    print_info "目标: 构建所有镜像并推送到仓库"
    echo
    
    # 1. 创建生产环境配置
    if ! create_production_env "build" "$registry" "$tag"; then
        return 1
    fi
    
    # 2. 构建所有服务镜像
    print_info "构建所有服务镜像..."
    if ! build_all_services "$tag" "$registry"; then
        print_error "服务镜像构建失败"
        return 1
    fi
    
    # 3. 推送所有镜像到仓库
    print_info "推送所有镜像到仓库..."
    if ! push_all_services "$tag" "$registry"; then
        print_error "镜像推送失败"
        return 1
    fi
    
    # 4. 推送依赖镜像
    print_info "推送依赖镜像..."
    if ! push_all_dependencies "$tag" "$registry"; then
        print_error "依赖镜像推送失败"
        return 1
    fi
    
    # 5. 生成生产环境docker-compose配置
    print_info "复制生产环境配置文件..."
    if [[ -f "docker-compose.yml.example" ]]; then
        cp docker-compose.yml.example docker-compose.yml
        print_success "✓ 已复制 docker-compose.yml.example 到 docker-compose.yml"
    else
        print_error "docker-compose.yml.example 文件不存在"
        return 1
    fi
    
    print_success "✅ 构建环境部署完成！"
    print_info "生成的文件:"
    print_info "  - .env.prod (生产环境配置)"
    print_info "  - docker-compose.prod.yml (生产环境编排文件)"
    print_info ""
    print_info "已推送到仓库的镜像:"
    print_info "  - 所有服务镜像 (标签: $tag)"
    print_info "  - 所有依赖镜像"
    print_info ""
    print_info "下一步: 将以下文件复制到内网环境："
    print_info "  - .env.prod"
    print_info "  - docker-compose.prod.yml"
    print_info "  - build.sh (用于内网部署)"
    
    return 0
}

# 内网环境模式 - 拉取镜像并启动服务
intranet_environment_deploy() {
    local registry="${1:-aiharbor.msxf.local/aihpc}"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    
    print_info "=========================================="
    print_info "内网环境部署模式"
    print_info "=========================================="
    print_info "镜像仓库: $registry"
    print_info "镜像标签: $tag"
    print_info "目标: 拉取镜像并启动所有服务"
    echo
    
    # 1. 检查或创建生产环境配置
    if [[ ! -f ".env.prod" ]]; then
        print_info "创建生产环境配置..."
        if ! create_production_env "intranet" "$registry" "$tag"; then
            return 1
        fi
    else
        print_info "使用现有的生产环境配置: .env.prod"
    fi
    
    # 2. 检查或生成docker-compose.prod.yml
    if [[ ! -f "docker-compose.prod.yml" ]]; then
        print_info "复制生产环境编排文件..."
        if [[ -f "docker-compose.yml.example" ]]; then
            cp docker-compose.yml.example docker-compose.prod.yml
            print_success "✓ 已复制 docker-compose.yml.example 到 docker-compose.prod.yml"
        else
            print_error "docker-compose.yml.example 文件不存在"
            return 1
        fi
    else
        print_info "使用现有的编排文件: docker-compose.prod.yml"
    fi
    
    # 3. 启动生产环境服务
    print_info "启动生产环境服务..."
    if ! start_production "$registry" "$tag" "false"; then
        print_error "服务启动失败"
        return 1
    fi
    
    print_success "✅ 内网环境部署完成！"
    print_info "服务状态:"
    production_status
    
    return 0
}

# ==========================================
# 生产环境部署相关功能
# ==========================================

# 部署到指定HOST（动态配置域名）
deploy_to_host() {
    local host="$1"
    local registry="$2"
    local tag="${3:-$DEFAULT_IMAGE_TAG}"
    
    if [[ -z "$host" ]]; then
        print_error "必须指定HOST地址"
        return 1
    fi
    
    print_info "===========================================" 
    print_info "部署AI-Infra到指定HOST: $host"
    print_info "==========================================="
    print_info "Host: $host"
    print_info "Registry: ${registry:-'(本地镜像)'}"
    print_info "Tag: $tag"
    echo
    
    # 备份原始.env.prod文件
    if [[ -f ".env.prod" ]]; then
        cp ".env.prod" ".env.prod.backup.$(date +%Y%m%d%H%M%S)"
        print_info "已备份原始.env.prod文件"
    fi
    
    # 检测HOST格式并设置PORT
    local nginx_port="8080"
    local public_host="$host:$nginx_port"
    local public_protocol="http"
    
    if [[ "$host" =~ ^https?:// ]]; then
        print_error "HOST不应包含协议前缀，请使用纯域名或IP，如: example.com 或 192.168.1.100"
        return 1
    fi
    
    if [[ "$host" =~ :[0-9]+$ ]]; then
        public_host="$host"
        print_info "检测到HOST包含端口: $public_host"
    else
        public_host="$host:$nginx_port"
        print_info "使用默认端口: $public_host"
    fi
    
    # 临时设置环境变量（用于生成配置）
    export AI_INFRA_HOST="$host"
    
    # 更新.env.prod文件中的HOST相关配置
    print_info "更新.env.prod中的HOST配置..."
    
    # 使用sed命令更新配置
    sed_inplace "s|^DOMAIN=.*|DOMAIN=$host|g" .env.prod
    sed_inplace "s|^PUBLIC_HOST=.*|PUBLIC_HOST=$public_host|g" .env.prod  
    sed_inplace "s|^JUPYTERHUB_PUBLIC_HOST=.*|JUPYTERHUB_PUBLIC_HOST=$public_host|g" .env.prod
    sed_inplace "s|^JUPYTERHUB_CORS_ORIGIN=.*|JUPYTERHUB_CORS_ORIGIN=$public_protocol://$public_host|g" .env.prod
    sed_inplace "s|^ROOT_URL=.*|ROOT_URL=$public_protocol://$public_host/gitea/|g" .env.prod
    cleanup_backup_files
    
    print_success "✓ HOST配置更新完成"
    
    # 复制生产环境配置文件
    print_info "复制生产环境配置文件..."
    if [[ -f "docker-compose.yml.example" ]]; then
        cp docker-compose.yml.example docker-compose.yml
        print_success "✓ 已复制 docker-compose.yml.example 到 docker-compose.yml"
    else
        print_error "docker-compose.yml.example 文件不存在"
        print_error "生产环境配置生成失败"
        return 1
    fi
    
    # 启动服务（使用本地镜像模式）
    print_info "启动生产环境服务..."
    if ! start_production "$registry" "$tag" "true"; then
        print_error "生产环境启动失败"
        return 1
    fi
    
    print_success "=========================================="
    print_success "🎉 部署完成！"
    print_success "=========================================="
    print_info "访问地址:"
    print_info "  主页: $public_protocol://$public_host/"
    print_info "  JupyterHub: $public_protocol://$public_host/jupyterhub/"
    print_info "  Gitea: $public_protocol://$public_host/gitea/"
    print_info ""
    print_info "管理命令:"
    print_info "  查看状态: $0 prod-status"
    print_info "  查看日志: $0 prod-logs [service]"
    print_info "  停止服务: $0 prod-down"
    echo
    
    return 0
}


# 从指定的私有仓库拉取镜像
pull_images_from_registry() {
    local registry="$1"
    local tag="$2"
    local env_file="$3"
    
    print_info "从私有仓库拉取镜像..."
    print_info "  仓库地址: $registry"
    print_info "  镜像标签: $tag"
    
    local success_count=0
    local total_count=0
    local failed_images=()
    
    # 拉取AI-Infra服务镜像
    print_info "拉取AI-Infra服务镜像..."
    for service in $SRC_SERVICES; do
        total_count=$((total_count + 1))
        local target_image="${registry}/ai-infra-${service}:${tag}"
        local local_image="ai-infra-${service}:${tag}"
        
        print_info "→ 拉取: $target_image"
        if docker pull "$target_image"; then
            # 标记为本地镜像名
            if docker tag "$target_image" "$local_image"; then
                print_success "  ✓ 拉取并标记成功: $local_image"
                success_count=$((success_count + 1))
            else
                print_error "  ✗ 标记失败: $local_image"
                failed_images+=("$target_image")
            fi
        else
            print_error "  ✗ 拉取失败: $target_image"
            failed_images+=("$target_image")
        fi
    done
    
    # 拉取依赖镜像
    print_info "拉取依赖镜像..."
    local dependency_images
    dependency_images=$(collect_dependency_images)
    
    for dep_image in $dependency_images; do
        if [[ -z "$dep_image" ]]; then
            continue
        fi
        
        total_count=$((total_count + 1))
        # 使用映射配置获取私有仓库中的镜像名
        local target_image
        target_image=$(get_mapped_private_image "$dep_image" "$registry" "$tag")
        
        print_info "→ 拉取依赖: $target_image"
        if docker pull "$target_image"; then
            # 标记为原始镜像名
            if docker tag "$target_image" "$dep_image"; then
                print_success "  ✓ 拉取并标记成功: $dep_image"
                success_count=$((success_count + 1))
            else
                print_error "  ✗ 标记失败: $dep_image"
                failed_images+=("$target_image")
            fi
        else
            print_warning "  ! 私有仓库拉取失败，尝试官方源: $dep_image"
            # 回退到官方镜像拉取
            if docker pull "$dep_image"; then
                print_success "  ✓ 从官方源拉取成功: $dep_image"
                success_count=$((success_count + 1))
            else
                print_error "  ✗ 所有源都拉取失败: $dep_image"
                failed_images+=("$dep_image")
            fi
        fi
    done
    
    print_info "=========================================="
    print_info "镜像拉取统计: $success_count/$total_count 成功"
    
    if [[ ${#failed_images[@]} -gt 0 ]]; then
        print_warning "以下镜像拉取失败:"
        for failed_image in "${failed_images[@]}"; do
            echo "  - $failed_image"
        done
        
        # 如果有镜像拉取失败，但不是全部失败，给出选择
        if [[ $success_count -gt 0 ]]; then
            print_warning "部分镜像拉取成功，是否继续启动服务？"
            return 0  # 允许继续，但会有警告
        else
            return 1  # 全部失败，返回错误
        fi
    else
        print_success "🎉 所有镜像拉取成功！"
        return 0
    fi
}

# ==========================================
# 镜像完整性检查和统一标记管理
# ==========================================

# 获取所有必需的镜像列表（从docker-compose配置提取）
get_required_images() {
    local compose_file="${1:-docker-compose.yml}"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    
    # AI-Infra服务镜像
    local ai_infra_images=(
        "ai-infra-backend:$tag"
        "ai-infra-backend-init:$tag"
        "ai-infra-frontend:$tag"
        "ai-infra-jupyterhub:$tag"
        "ai-infra-gitea:$tag"
        "ai-infra-nginx:$tag"
        "ai-infra-saltstack:$tag"
        "ai-infra-singleuser:$tag"
    )
    
    # 依赖镜像（从映射配置获取）
    local dependency_images=(
        "postgres:15-alpine"
        "redis:7-alpine"
        "nginx:1.27-alpine"
        "tecnativa/tcp-proxy:latest"
        "minio/minio:latest"
        "osixia/openldap:stable"
        "osixia/phpldapadmin:stable"
        "redislabs/redisinsight:latest"
        "confluentinc/cp-kafka:7.5.0"
        "provectuslabs/kafka-ui:latest"
    )
    
    # 合并所有镜像
    local all_images=("${ai_infra_images[@]}" "${dependency_images[@]}")
    printf '%s\n' "${all_images[@]}"
}

# 检查镜像完整性 - 验证所有必需的镜像是否存在
check_images_completeness() {
    local registry="$1"
    local tag="$2"
    local compose_file="${3:-docker-compose.yml}"
    
    print_info "检查镜像完整性..."
    
    # 获取所有必需的镜像
    local required_images
    mapfile -t required_images < <(get_required_images "$compose_file" "$tag")
    
    local missing_images=()
    local present_images=()
    local total_count=${#required_images[@]}
    
    for image in "${required_images[@]}"; do
        # 对于AI-Infra镜像，检查是否存在
        if [[ "$image" == ai-infra-* ]]; then
            if docker image inspect "$image" >/dev/null 2>&1; then
                present_images+=("$image")
            else
                missing_images+=("$image")
            fi
        else
            # 对于依赖镜像，如果指定了registry，检查转换后的镜像
            if [[ -n "$registry" ]]; then
                local target_image
                target_image=$(get_mapped_private_image "$image" "$registry" "$tag")
                if docker image inspect "$target_image" >/dev/null 2>&1; then
                    present_images+=("$target_image")
                elif docker image inspect "$image" >/dev/null 2>&1; then
                    # 原始镜像存在，但转换后的不存在
                    missing_images+=("$target_image (需要从 $image 转换)")
                else
                    missing_images+=("$target_image")
                fi
            else
                if docker image inspect "$image" >/dev/null 2>&1; then
                    present_images+=("$image")
                else
                    missing_images+=("$image")
                fi
            fi
        fi
    done
    
    print_info "镜像完整性检查结果:"
    print_success "  ✓ 可用镜像: ${#present_images[@]}/$total_count"
    
    if [[ ${#missing_images[@]} -gt 0 ]]; then
        print_warning "  ⚠ 缺失镜像: ${#missing_images[@]}/$total_count"
        for missing in "${missing_images[@]}"; do
            echo "    - $missing"
        done
        return 1
    else
        print_success "  🎉 所有镜像都已准备就绪！"
        return 0
    fi
}

# 统一标记转换函数 - 将公共镜像tag为aiharbor内部版本
convert_images_to_unified_tags() {
    local registry="$1"
    local tag="$2"
    
    if [[ -z "$registry" ]]; then
        print_info "未指定registry，跳过镜像统一标记"
        return 0
    fi
    
    print_info "=========================================="
    print_info "统一标记镜像到内部版本"
    print_info "=========================================="
    print_info "目标Registry: $registry"
    print_info "统一标签: $tag"
    echo
    
    # 获取需要转换的依赖镜像（不包括AI-Infra服务镜像）
    local dependency_images=(
        "postgres:15-alpine"
        "redis:7-alpine"
        "nginx:1.27-alpine"
        "tecnativa/tcp-proxy:latest"
        "minio/minio:latest"
        "osixia/openldap:stable"
        "osixia/phpldapadmin:stable"
        "redislabs/redisinsight:latest"
        "confluentinc/cp-kafka:7.5.0"
        "provectuslabs/kafka-ui:latest"
    )
    
    local converted_count=0
    local failed_count=0
    local skipped_count=0
    
    for source_image in "${dependency_images[@]}"; do
        # 使用映射配置生成目标镜像名
        local target_image
        target_image=$(get_mapped_private_image "$source_image" "$registry" "$tag")
        
        print_info "→ 处理依赖镜像: $source_image"
        print_info "  目标镜像: $target_image"
        
        # 检查目标镜像是否已存在
        if docker image inspect "$target_image" >/dev/null 2>&1; then
            print_success "  ✓ 目标镜像已存在，跳过转换"
            ((skipped_count++))
            continue
        fi
        
        # 检查源镜像是否存在
        if ! docker image inspect "$source_image" >/dev/null 2>&1; then
            print_warning "  ⚠ 源镜像不存在: $source_image"
            print_info "  → 尝试拉取源镜像..."
            if docker pull "$source_image"; then
                print_success "  ✓ 源镜像拉取成功"
            else
                print_error "  ✗ 源镜像拉取失败: $source_image"
                ((failed_count++))
                continue
            fi
        fi
        
        # 执行标记转换
        if docker tag "$source_image" "$target_image"; then
            print_success "  ✓ 镜像转换成功: $source_image → $target_image"
            ((converted_count++))
        else
            print_error "  ✗ 镜像转换失败: $source_image → $target_image"
            ((failed_count++))
        fi
        echo
    done
    
    # 对AI-Infra服务镜像进行registry标记
    print_info "标记AI-Infra服务镜像..."
    local ai_infra_services=("backend" "backend-init" "frontend" "jupyterhub" "gitea" "nginx" "saltstack" "singleuser")
    
    for service in "${ai_infra_services[@]}"; do
        local source_image="ai-infra-${service}:${tag}"
        local target_image="${registry}/ai-infra-${service}:${tag}"
        
        print_info "→ 处理服务镜像: $source_image"
        
        # 检查目标镜像是否已存在
        if docker image inspect "$target_image" >/dev/null 2>&1; then
            print_success "  ✓ 已存在: $target_image"
            ((skipped_count++))
            continue
        fi
        
        # 检查源镜像是否存在
        if docker image inspect "$source_image" >/dev/null 2>&1; then
            if docker tag "$source_image" "$target_image"; then
                print_success "  ✓ 标记成功: $source_image → $target_image"
                ((converted_count++))
            else
                print_error "  ✗ 标记失败: $source_image → $target_image"
                ((failed_count++))
            fi
        else
            print_warning "  ⚠ 源镜像不存在: $source_image (需要先构建或拉取)"
            ((failed_count++))
        fi
    done
    
    echo
    print_info "=========================================="
    print_info "镜像统一标记完成统计:"
    print_success "  ✓ 成功转换: $converted_count 个"
    print_info "  → 已存在跳过: $skipped_count 个"
    
    if [[ $failed_count -gt 0 ]]; then
        print_error "  ✗ 转换失败: $failed_count 个"
        return 1
    else
        print_success "  🎉 所有镜像统一标记完成！"
        return 0
    fi
}

# 智能镜像准备函数 - 组合完整性检查、统一标记和拉取
prepare_images_intelligently() {
    local registry="$1"
    local tag="$2"
    local compose_file="${3:-docker-compose.yml}"
    
    print_info "=========================================="
    print_info "智能镜像准备"
    print_info "=========================================="
    print_info "Registry: ${registry:-'(本地模式)'}"
    print_info "Tag: $tag"
    echo
    
    # 步骤1: 检查镜像完整性
    print_info "步骤 1/3: 检查镜像完整性..."
    local images_complete=false
    if check_images_completeness "$registry" "$tag" "$compose_file"; then
        images_complete=true
        print_success "✓ 镜像完整性检查通过"
    else
        print_warning "⚠ 存在缺失镜像，继续处理..."
    fi
    
    # 如果指定了registry且镜像不完整，尝试统一标记转换
    if [[ -n "$registry" && "$images_complete" == "false" ]]; then
        echo
        print_info "步骤 2/3: 统一标记镜像转换..."
        if convert_images_to_unified_tags "$registry" "$tag"; then
            print_success "✓ 镜像统一标记完成"
            
            # 再次检查完整性
            print_info "重新检查镜像完整性..."
            if check_images_completeness "$registry" "$tag" "$compose_file"; then
                images_complete=true
                print_success "✓ 镜像完整性检查通过"
            fi
        else
            print_warning "⚠ 部分镜像标记转换失败"
        fi
    fi
    
    # 步骤3: 如果仍不完整，尝试拉取缺失镜像
    if [[ "$images_complete" == "false" ]]; then
        echo
        print_info "步骤 3/3: 拉取缺失镜像..."
        if [[ -n "$registry" ]]; then
            # 从指定registry拉取
            if pull_images_from_registry "$registry" "$tag"; then
                print_success "✓ 缺失镜像拉取完成"
                images_complete=true
            else
                print_error "✗ 从registry拉取镜像失败"
            fi
        else
            # 使用docker-compose pull拉取官方镜像
            print_info "使用docker-compose拉取官方镜像..."
            local env_file
            env_file=$(detect_env_file)
            if ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" pull; then
                print_success "✓ 官方镜像拉取完成"
                images_complete=true
            else
                print_error "✗ 官方镜像拉取失败"
            fi
        fi
    fi
    
    echo
    print_info "=========================================="
    if [[ "$images_complete" == "true" ]]; then
        print_success "🎉 镜像准备完成，可以启动服务！"
        return 0
    else
        print_error "❌ 镜像准备失败，部分镜像仍然缺失"
        print_info "建议操作："
        print_info "1. 检查网络连接和仓库权限"
        print_info "2. 手动拉取缺失镜像"
        print_info "3. 或使用 --force 参数强制启动"
        return 1
    fi
}

# 替换 docker-compose.yml 中的镜像名称为内部映射版本
replace_images_in_compose_file() {
    local compose_file="$1"
    local registry="$2"
    local tag="$3"
    local backup_file="${compose_file}.backup.$(date +%s)"
    
    print_info "替换 compose 文件中的镜像名称为内部版本..."
    
    # 备份原始文件
    cp "$compose_file" "$backup_file"
    print_info "已备份原始文件: $backup_file"
    
    # 获取需要替换的镜像列表和映射
    local temp_compose="$compose_file.tmp"
    cp "$compose_file" "$temp_compose"
    
    # 替换第三方依赖镜像
    local dependency_replacements=(
        "confluentinc/cp-kafka:7.5.0|${registry}/cp-kafka:${tag}"
        "confluentinc/cp-kafka:7.4.0|${registry}/cp-kafka:${tag}"
        "confluentinc/cp-kafka:latest|${registry}/cp-kafka:${tag}"
        "provectuslabs/kafka-ui:latest|${registry}/kafka-ui:${tag}"
        "postgres:15-alpine|${registry}/postgres:${tag}"
        "postgres:latest|${registry}/postgres:${tag}"
        "redis:7-alpine|${registry}/redis:${tag}"
        "redis:latest|${registry}/redis:${tag}"
        "nginx:1.27-alpine|${registry}/nginx:${tag}"
        "nginx:stable-alpine-perl|${registry}/nginx:${tag}"
        "nginx:latest|${registry}/nginx:${tag}"
        "tecnativa/tcp-proxy:latest|${registry}/tcp-proxy:${tag}"
        "tecnativa/tcp-proxy|${registry}/tcp-proxy:${tag}"
        "minio/minio:latest|${registry}/minio:${tag}"
        "osixia/openldap:stable|${registry}/openldap:${tag}"
        "osixia/openldap:latest|${registry}/openldap:${tag}"
        "osixia/phpldapadmin:stable|${registry}/phpldapadmin:${tag}"
        "osixia/phpldapadmin:latest|${registry}/phpldapadmin:${tag}"
        "redislabs/redisinsight:latest|${registry}/redisinsight:${tag}"
        "quay.io/minio/minio:latest|${registry}/minio:${tag}"
        "gitea/gitea:1.24.6|${registry}/gitea:${tag}"
        "jupyter/base-notebook:latest|${registry}/base-notebook:${tag}"
        "node:22-alpine|${registry}/node:${tag}"
        "golang:1.25-alpine|${registry}/golang:${tag}"
        "python:3.13-alpine|${registry}/python:${tag}"
    )
    
    local replacement_count=0
    for replacement in "${dependency_replacements[@]}"; do
        local source_image="${replacement%%|*}"
        local target_image="${replacement##*|}"
        
        # 检查文件中是否包含该镜像
        if grep -q "$source_image" "$temp_compose"; then
            print_info "  替换: $source_image → $target_image"
            # 使用 sed 进行替换，处理可能的特殊字符
            sed_inplace "s|image: $source_image|image: $target_image|g" "$temp_compose"
            ((replacement_count++))
        fi
    done
    
    # 清理备份文件
    cleanup_backup_files "$(dirname "$temp_compose")"
    
    # 替换AI-Infra服务镜像（如果指定了registry）
    if [[ -n "$registry" ]]; then
        local ai_infra_services=("backend" "backend-init" "frontend" "jupyterhub" "gitea" "nginx" "saltstack" "singleuser")
        for service in "${ai_infra_services[@]}"; do
            local source_pattern="ai-infra-${service}:\${IMAGE_TAG:-v0.3.6-dev}"
            local target_replacement="${registry}/ai-infra-${service}:${tag}"
            
            if grep -q "ai-infra-${service}:" "$temp_compose"; then
                print_info "  替换服务镜像: ai-infra-${service} → $target_replacement"
                sed_inplace "s|image: ai-infra-${service}:\${IMAGE_TAG:-[^}]*}|image: $target_replacement|g" "$temp_compose"
                sed_inplace "s|image: ai-infra-${service}:\${IMAGE_TAG}|image: $target_replacement|g" "$temp_compose"
                sed_inplace "s|image: ai-infra-${service}:${tag}|image: $target_replacement|g" "$temp_compose"
                ((replacement_count++))
            fi
        done
        cleanup_backup_files "$(dirname "$temp_compose")"
    fi
    
    # 如果有替换，使用临时文件
    if [[ $replacement_count -gt 0 ]]; then
        mv "$temp_compose" "$compose_file"
        print_success "✓ 已替换 $replacement_count 个镜像名称"
        echo "$backup_file"  # 返回备份文件路径
    else
        rm -f "$temp_compose"
        print_info "未找到需要替换的镜像，保持原样"
        rm -f "$backup_file"  # 删除不需要的备份
        echo ""  # 返回空字符串
    fi
}

# 恢复原始 docker-compose.yml 文件
restore_compose_file() {
    local compose_file="$1"
    local backup_file="$2"
    
    if [[ -n "$backup_file" && -f "$backup_file" ]]; then
        print_info "恢复原始 compose 文件..."
        mv "$backup_file" "$compose_file"
        print_success "✓ 已恢复原始 docker-compose.yml"
    fi
}

# 启动生产环境
start_production() {
    # 处理帮助参数
    if [[ "$1" == "--help" || "$1" == "-h" ]]; then
        echo "prod-up - 启动生产环境"
        echo
        echo "用法: $0 prod-up [registry] [tag] [--force]"
        echo
        echo "参数:"
        echo "  registry    私有仓库地址 (可选，留空使用本地镜像)"
        echo "  tag         镜像标签 (默认: $DEFAULT_IMAGE_TAG)"
        echo "  --force     强制使用本地镜像，跳过镜像检查"
        echo
        echo "说明:"
        echo "  启动生产环境的所有服务，包括："
        echo "  • 智能镜像准备和检查"
        echo "  • 环境配置生成"
        echo "  • 服务启动和健康检查"
        echo "  • 自动化部署流程"
        echo
        echo "环境文件优先级:"
        echo "  1. .env.prod (生产环境专用)"
        echo "  2. .env (开发环境)"
        echo
        echo "示例:"
        echo "  $0 prod-up                                    # 使用本地镜像启动"
        echo "  $0 prod-up harbor.company.com/ai-infra v1.0.0 # 使用私有仓库镜像"
        echo "  $0 prod-up aiharbor.msxf.local/aihpc v1.0.0  # 使用内部仓库镜像"
        echo "  $0 prod-up registry.local v1.0.0 --force     # 强制使用本地镜像"
        return 0
    fi
    
    local registry="$1"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    local force_local="${3:-false}"  # 新增参数：是否强制使用本地镜像
    local compose_file="docker-compose.yml"
    
    # 自动检测外部主机地址
    if [[ -f "scripts/detect-external-host.sh" ]]; then
        print_info "自动检测外部主机地址..."
        source scripts/detect-external-host.sh
        print_info "使用检测到的主机地址: $EXTERNAL_HOST"
    fi
    
    # registry 可以为空（使用本地镜像）
    if [[ -z "$registry" ]]; then
        print_info "使用本地镜像（无 registry 前缀）"
        registry=""
    fi
    
    # 检测环境文件 - 统一使用 .env 文件
    local env_file=".env"
    if [[ ! -f "$env_file" ]]; then
        print_warning "环境文件不存在，从模板创建: $env_file"
        if [[ -f ".env.example" ]]; then
            cp ".env.example" "$env_file"
            print_success "✓ 已从 .env.example 创建环境文件"
        else
            print_error "模板文件 .env.example 不存在"
            return 1
        fi
    fi
    print_info "使用环境文件: $env_file"
    
    # 验证环境文件
    if ! validate_env_file "$env_file"; then
        return 1
    fi
    
    # 总是重新复制配置文件
    print_info "复制配置文件 (registry: $registry, tag: $tag)..."
    if [[ -f "docker-compose.yml.example" ]]; then
        cp docker-compose.yml.example docker-compose.yml
        print_success "✓ 已复制 docker-compose.yml.example 到 docker-compose.yml"
    else
        print_error "docker-compose.yml.example 文件不存在"
        return 1
    fi
    
    print_info "=========================================="
    print_info "启动生产环境"
    print_info "=========================================="
    print_info "配置文件: $compose_file"
    print_info "环境文件: $env_file"
    print_info "Registry: ${registry:-'(本地模式)'}"
    print_info "标签: $tag"
    if [[ "$force_local" == "true" ]]; then
        print_info "模式: 强制使用本地镜像 (跳过智能处理)"
    fi
    echo
    
    # 智能镜像处理策略
    if [[ "$force_local" == "true" ]]; then
        print_info "强制本地模式 - 跳过智能镜像处理..."
        
        # 如果指定了registry，只做简单标记
        if [[ -n "$registry" ]]; then
            print_info "为本地镜像添加 registry 标签..."
            tag_local_images_for_registry "$registry" "$tag"
        fi
        
        # 检查并构建缺失的镜像
        print_info "检查并构建需要的镜像..."
        if ! check_and_build_missing_images "$compose_file" "$env_file" "$registry" "$tag"; then
            print_warning "部分镜像构建失败，继续尝试启动..."
        fi
    else
        # 使用智能镜像准备功能
        print_info "执行智能镜像准备..."
        if ! prepare_images_intelligently "$registry" "$tag" "$compose_file"; then
            print_error "智能镜像准备失败"
            print_info ""
            print_info "可选的解决方案："
            print_info "1. 使用 --force 强制启动: $0 prod-up $registry $tag --force"
            print_info "2. 手动拉取镜像: $0 harbor-pull-all $registry $tag"
            print_info "3. 检查网络和仓库权限"
            return 1
        fi
        
        # 检查并构建需要构建的镜像（如有build配置的服务）
        print_info "检查并构建需要构建的镜像..."
        if ! check_and_build_missing_images "$compose_file" "$env_file" "$registry" "$tag"; then
            print_warning "部分镜像构建失败，继续尝试启动..."
        fi
    fi
    
    # 针对内部仓库的特殊处理：替换compose文件中的镜像名称
    local backup_file=""
    if [[ -n "$registry" ]]; then
        print_info "针对内部仓库进行镜像名称替换..."
        backup_file=$(replace_images_in_compose_file "$compose_file" "$registry" "$tag")
    fi
    
    print_info "启动生产环境..."
    local startup_success=false
    if ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" up -d; then
        startup_success=true
        print_success "✓ 生产环境启动成功"
        echo
        
        # 等待所有服务启动完成
        print_info "等待所有服务启动完成..."
        if wait_for_services_healthy "$compose_file" "$env_file"; then
            print_success "✓ 所有服务已启动并运行正常"
            echo
            print_info "最终服务状态:"
            ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" ps
            echo
            print_info "🎉 生产环境部署完成！"
            print_info "=========================================="
            print_info "访问地址:"
            print_info "  主页: http://localhost/"
            print_info "  JupyterHub: http://localhost/jupyterhub/"
            print_info "  Gitea: http://localhost/gitea/"
            print_info ""
            print_info "管理命令:"
            print_info "  查看状态: $0 prod-status"
            print_info "  查看日志: $0 prod-logs [service]"
            print_info "  停止服务: $0 prod-down"
        else
            print_error "✗ 部分服务启动失败，请检查日志"
            print_info "查看详细日志: $0 prod-logs"
            print_info "查看服务状态: $0 prod-status"
            startup_success=false
        fi
    else
        print_error "✗ 生产环境启动失败"
        print_info "请检查错误信息并查看日志: $0 prod-logs"
        startup_success=false
    fi
    
    # 恢复原始compose文件
    restore_compose_file "$compose_file" "$backup_file"
    
    if [[ "$startup_success" == "true" ]]; then
        return 0
    else
        return 1
    fi
}

# 等待所有服务启动完成并检查健康状态
wait_for_services_healthy() {
    local compose_file="$1"
    local env_file="$2"
    local max_wait_time=300  # 最大等待时间5分钟
    local check_interval=10  # 每10秒检查一次
    local elapsed=0
    
    print_info "开始监控服务健康状态..."
    
    while [[ $elapsed -lt $max_wait_time ]]; do
        # 获取所有服务的状态
        local services_status=$(ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" ps --format "table {{.Name}}\t{{.Status}}")
        
        # 跳过表头
        local services_info=$(echo "$services_status" | tail -n +2)
        
        # 检查是否有服务失败
        if echo "$services_info" | grep -q "Exit"; then
            print_error "发现服务启动失败:"
            echo "$services_info" | grep "Exit"
            return 1
        fi
        
        # 检查所有服务是否都健康或运行中
        local total_services=$(echo "$services_info" | wc -l)
        local healthy_services=$(echo "$services_info" | grep -E "(healthy|running|Up)" | wc -l)
        
        if [[ $healthy_services -eq $total_services ]]; then
            print_success "所有 $total_services 个服务都已启动并运行正常"
            return 0
        fi
        
        # 显示当前进度
        local progress=$((elapsed * 100 / max_wait_time))
        print_info "等待服务启动... ($elapsed/$max_wait_time 秒) - $healthy_services/$total_services 服务就绪"
        
        sleep $check_interval
        elapsed=$((elapsed + check_interval))
    done
    
    print_error "等待超时：部分服务未能正常启动"
    print_info "当前服务状态:"
    ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" ps
    return 1
}
tag_local_images_for_registry() {
    local registry="$1"
    local tag="$2"
    
    print_info "标记本地镜像为新的registry标签..."
    
    # 智能查找本地镜像的函数
    find_local_image() {
        local image_name="$1"
        local target_tag="$2"
        
        # 先尝试精确匹配
        if docker image inspect "${image_name}:${target_tag}" >/dev/null 2>&1; then
            echo "${image_name}:${target_tag}"
            return 0
        fi
        
        # 如果精确匹配失败，尝试查找包含目标标签的镜像
        local found_image=$(docker images --format "table {{.Repository}}:{{.Tag}}" | grep "^${image_name}:" | grep -E "(test-)?${target_tag}$" | head -n1)
        if [[ -n "$found_image" ]]; then
            echo "$found_image"
            return 0
        fi
        
        # 如果还是找不到，查找最新的镜像
        local latest_image=$(docker images --format "table {{.Repository}}:{{.Tag}}" | grep "^${image_name}:" | grep -v "<none>" | head -n1)
        if [[ -n "$latest_image" ]]; then
            echo "$latest_image"
            return 0
        fi
        
        return 1
    }
    
    # 定义需要标记的镜像基础名称
    local ai_infra_images=(
        "ai-infra-backend"
        "ai-infra-backend-init"
        "ai-infra-frontend"
        "ai-infra-jupyterhub"
        "ai-infra-gitea"
        "ai-infra-nginx"
        "ai-infra-saltstack"
        "ai-infra-singleuser"
    )
    
    # 定义依赖镜像
    local dependency_images=(
        "postgres:15-alpine"
        "redis:7-alpine"
        "nginx:1.27-alpine"
        "tecnativa/tcp-proxy:latest"
        "minio/minio:latest"
        "osixia/openldap:stable"
        "osixia/phpldapadmin:stable"
        "redislabs/redisinsight:latest"
        "node:22-alpine"
        "nginx:stable-alpine-perl"
        "golang:1.25-alpine"
        "python:3.13-alpine"
        "gitea/gitea:1.24.6"
        "jupyter/base-notebook:latest"
    )
    
    local tagged_count=0
    local missing_count=0
    
    # 处理AI-Infra自研镜像
    for image_name in "${ai_infra_images[@]}"; do
        local target_image="${registry}/${image_name}:${tag}"
        
        # 检查目标镜像是否已存在
        if docker image inspect "$target_image" >/dev/null 2>&1; then
            print_info "  ✓ 已存在: $target_image"
            continue
        fi
        
        # 智能查找本地镜像
        local source_image=$(find_local_image "$image_name" "$tag")
        if [[ -n "$source_image" ]]; then
            # 标记镜像
            if docker tag "$source_image" "$target_image" 2>/dev/null; then
                print_success "  ✓ 已标记: $source_image -> $target_image"
                tagged_count=$((tagged_count + 1))
            else
                print_warning "  ✗ 标记失败: $source_image -> $target_image"
            fi
        else
            print_warning "  ✗ 本地未找到镜像: $image_name"
            missing_count=$((missing_count + 1))
        fi
    done
    
    # 处理依赖镜像
    for source_image in "${dependency_images[@]}"; do
        # 计算目标镜像名（移除域名前缀）
        local clean_name=$(echo "$source_image" | sed 's|^[^/]*/||' | sed 's|^[^/]*/||')
        local target_image="${registry}/${clean_name}"
        
        # 检查目标镜像是否已存在
        if docker image inspect "$target_image" >/dev/null 2>&1; then
            print_info "  ✓ 已存在: $target_image"
            continue
        fi
        
        # 检查源镜像是否存在
        if docker image inspect "$source_image" >/dev/null 2>&1; then
            # 标记镜像
            if docker tag "$source_image" "$target_image" 2>/dev/null; then
                print_success "  ✓ 已标记: $source_image -> $target_image"
                tagged_count=$((tagged_count + 1))
            else
                print_warning "  ✗ 标记失败: $source_image -> $target_image"
            fi
        else
            print_warning "  ✗ 源镜像不存在: $source_image"
            missing_count=$((missing_count + 1))
        fi
    done
    
    print_info "镜像标记完成: 成功 $tagged_count 个，缺失 $missing_count 个"
    
    return 0
}

check_and_build_missing_images() {
    local compose_file="$1"
    local env_file="$2"
    local registry="$3"
    local tag="$4"
    
    if [[ ! -f "$compose_file" ]]; then
        print_error "compose文件不存在: $compose_file"
        return 1
    fi
    
    print_info "分析compose文件中需要的镜像..."
    
    # 直接构建已知的关键服务（简化方案）
    local critical_services=("backend-init" "gitea" "singleuser-builder")
    local missing_count=0
    
    for service in "${critical_services[@]}"; do
        # 构造预期的镜像名
        local expected_image="${registry}/ai-infra-${service}:${tag}"
        
        # 检查镜像是否存在
        if ! docker image inspect "$expected_image" >/dev/null 2>&1; then
            print_info "缺失镜像: $expected_image"
            if build_service_if_missing "$service" "$compose_file" "$env_file"; then
                # 构建成功后标记镜像
                local local_image="ai-infra-${service}:${tag}"
                if docker image inspect "$local_image" >/dev/null 2>&1; then
                    docker tag "$local_image" "$expected_image"
                    print_success "✓ 已标记: $local_image -> $expected_image"
                fi
            else
                missing_count=$((missing_count + 1))
            fi
        else
            print_success "✓ 镜像已存在: $expected_image"
        fi
    done
    
    if [[ $missing_count -eq 0 ]]; then
        print_success "所有关键镜像都已准备就绪"
        return 0
    else
        print_warning "有 $missing_count 个关键服务构建失败"
        return 1
    fi
}

# 构建单个服务（如果缺失）
build_service_if_missing() {
    local service="$1"
    local compose_file="$2"
    local env_file="$3"
    
    print_info "尝试构建服务: $service"
    
    # 使用docker-compose构建特定服务
    if ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" build "$service" 2>/dev/null; then
        print_success "✓ 构建成功: $service"
        return 0
    else
        print_warning "✗ 构建失败: $service (可能不存在build配置)"
        return 1
    fi
}

# 停止生产环境
stop_production() {
    # 处理帮助参数
    if [[ "$1" == "--help" || "$1" == "-h" ]]; then
        echo "prod-down - 停止生产环境"
        echo
        echo "用法: $0 prod-down"
        echo
        echo "说明:"
        echo "  安全停止生产环境的所有服务，包括："
        echo "  • 停止所有Docker Compose服务"
        echo "  • 清理临时数据"
        echo "  • 保留持久化数据"
        echo "  • 自动检测环境配置文件"
        echo
        echo "环境文件优先级:"
        echo "  1. .env.prod (生产环境专用)"
        echo "  2. .env (开发环境)"
        echo
        echo "示例:"
        echo "  $0 prod-down"
        return 0
    fi
    
    local compose_file="docker-compose.yml"
    
    if [[ ! -f "$compose_file" ]]; then
        print_error "生产配置文件不存在: $compose_file"
        return 1
    fi
    
    # 检测环境文件 - 生产环境优先使用 .env.prod
    local env_file
    if [[ -f ".env.prod" ]]; then
        env_file=".env.prod"
        print_info "使用生产环境文件: $env_file"
    else
        env_file=$(detect_env_file)
        if [[ $? -ne 0 ]]; then
            return 1
        fi
    fi
    
    print_info "=========================================="
    print_info "停止生产环境"
    print_info "=========================================="
    print_info "使用环境文件: $env_file"
    
    if ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" down; then
        print_success "✓ 生产环境已停止"
        return 0
    else
        print_error "✗ 生产环境停止失败"
        return 1
    fi
}

# 重启生产环境
restart_production() {
    # 处理帮助参数
    if [[ "$1" == "--help" || "$1" == "-h" ]]; then
        echo "prod-restart - 重启生产环境"
        echo
        echo "用法: $0 prod-restart [registry] [tag]"
        echo
        echo "参数:"
        echo "  registry    私有仓库地址 (可选，留空使用本地镜像)"
        echo "  tag         镜像标签 (默认: $DEFAULT_IMAGE_TAG)"
        echo
        echo "说明:"
        echo "  重启生产环境的所有服务，包括："
        echo "  • 安全停止所有服务"
        echo "  • 等待服务完全停止"
        echo "  • 重新启动所有服务"
        echo "  • 相当于先执行 prod-down 再执行 prod-up"
        echo
        echo "示例:"
        echo "  $0 prod-restart"
        echo "  $0 prod-restart harbor.company.com/ai-infra v1.0.0"
        return 0
    fi
    
    local registry="$1"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    
    print_info "=========================================="
    print_info "重启生产环境"
    print_info "=========================================="
    
    # 先停止
    stop_production
    
    # 等待一段时间
    sleep 2
    
    # 再启动
    start_production "$registry" "$tag"
}

# 查看生产环境状态
production_status() {
    # 处理帮助参数
    if [[ "$1" == "--help" || "$1" == "-h" ]]; then
        echo "prod-status - 查看生产环境状态"
        echo
        echo "用法: $0 prod-status"
        echo
        echo "说明:"
        echo "  查看生产环境所有服务的运行状态，包括："
        echo "  • 容器运行状态"
        echo "  • 端口映射信息"
        echo "  • 资源使用情况"
        echo "  • 健康检查状态"
        echo
        echo "显示信息:"
        echo "  • 服务名称和状态"
        echo "  • 启动时间和运行时长"
        echo "  • 端口映射"
        echo "  • 容器ID和镜像版本"
        echo
        echo "示例:"
        echo "  $0 prod-status"
        return 0
    fi
    
    local compose_file="docker-compose.yml"
    
    if [[ ! -f "$compose_file" ]]; then
        print_error "生产配置文件不存在: $compose_file"
        return 1
    fi
    
    # 检测环境文件 - 生产环境优先使用 .env.prod
    local env_file
    if [[ -f ".env.prod" ]]; then
        env_file=".env.prod"
        print_info "使用生产环境配置: $env_file"
    else
        env_file=$(detect_env_file)
        if [[ $? -ne 0 ]]; then
            return 1
        fi
    fi
    
    print_info "=========================================="
    print_info "生产环境状态"
    print_info "=========================================="
    print_info "使用环境文件: $env_file"
    
    ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" ps
}

# 查看生产环境日志
production_logs() {
    local compose_file="docker-compose.yml"
    local service="$1"
    local follow="${2:-false}"
    
    if [[ ! -f "$compose_file" ]]; then
        print_error "生产配置文件不存在: $compose_file"
        return 1
    fi
    
    # 检测环境文件
    local env_file
    env_file=$(detect_env_file)
    if [[ $? -ne 0 ]]; then
        return 1
    fi
    
    if [[ -z "$service" ]]; then
        # 显示所有服务的日志
        if [[ "$follow" == "true" ]]; then
            ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" logs -f
        else
            ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" logs --tail=100
        fi
    else
        # 显示指定服务的日志
        if [[ "$follow" == "true" ]]; then
            ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" logs -f "$service"
        else
            ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" logs --tail=100 "$service"
        fi
    fi
}

# ==========================================
# 服务列表功能
# ==========================================

# 列出所有服务和镜像
list_services() {
    local tag="${1:-$DEFAULT_IMAGE_TAG}"
    local registry="${2:-}"
    
    print_info "=========================================="
    print_info "AI-Infra 服务清单"
    print_info "=========================================="
    print_info "镜像标签: $tag"
    if [[ -n "$registry" ]]; then
        print_info "目标仓库: $registry"
    else
        print_info "目标仓库: 本地构建"
    fi
    echo
    
    local service_count=0
    for service in $SRC_SERVICES; do
        service_count=$((service_count + 1))
    done
    
    print_info "📦 源码服务 ($service_count 个):"
    for service in $SRC_SERVICES; do
        local service_path=$(get_service_path "$service")
        local dockerfile_path="$service_path/Dockerfile"
        local base_image="ai-infra-${service}:${tag}"
        local target_image="$base_image"
        
        if [[ -n "$registry" ]]; then
            target_image=$(get_private_image_name "$base_image" "$registry")
        fi
        
        # 检查 Dockerfile 是否存在
        local status="✅"
        if [[ ! -f "$SCRIPT_DIR/$dockerfile_path" ]]; then
            status="❌"
        fi
        
        echo "  $status $service"
        echo "       Dockerfile: $dockerfile_path"
        echo "       镜像名称: $target_image"
        echo
    done
    
    print_info "=========================================="
}

# ==========================================
# 镜像验证功能
# ==========================================

# 验证单个镜像是否可用
verify_image() {
    local image="$1"
    local timeout="${2:-10}"
    
    # 先尝试检查本地镜像
    if docker image inspect "$image" >/dev/null 2>&1; then
        return 0
    fi
    
    # 尝试拉取验证（用于远程镜像）
    if timeout "$timeout" docker pull "$image" >/dev/null 2>&1; then
        return 0
    fi
    
    return 1
}

# 验证私有仓库中的所有AI-Infra镜像
verify_private_images() {
    local registry="$1"
    local tag="${2:-v0.3.6-dev}"
    
    if [[ -z "$registry" ]]; then
        print_error "使用方法: verify <registry_base> [tag]"
        print_info "示例: verify aiharbor.msxf.local/aihpc v0.3.6-dev"
        return 1
    fi
    
    print_info "=== AI Infrastructure Matrix 镜像验证 ==="
    print_info "目标仓库: $registry"
    print_info "镜像标签: $tag"
    print_info "开始时间: $(date)"
    echo
    
    print_info "📋 Harbor项目检查："
    print_info "验证前请确保以下项目已在Harbor中创建："
    print_info "  • aihpc (主项目)"
    print_info "  • library (基础镜像)"
    print_info "  • tecnativa (第三方镜像)"
    print_info "  • redislabs (第三方镜像)"
    print_info "  • minio (第三方镜像)"
    echo
    print_info "如未创建，请参考: docs/HARBOR_PROJECT_SETUP.md"
    echo
    
    # 源码镜像列表
    local source_images=(
        "ai-infra-backend-init"
        "ai-infra-backend"
        "ai-infra-frontend"
        "ai-infra-jupyterhub"
        "ai-infra-singleuser"
        "ai-infra-saltstack"
        "ai-infra-nginx"
        "ai-infra-gitea"
    )
    
    # 基础镜像列表（从配置文件获取）
    local base_image_patterns=(
        "postgres:15-alpine"
        "redis:7-alpine"
        "nginx:1.27-alpine"
        "tecnativa/tcp-proxy:latest"
        "redislabs/redisinsight:latest"
        "quay.io/minio/minio:latest"
    )
    
    local total_images=$((${#source_images[@]} + ${#base_image_patterns[@]}))
    local success_count=0
    local failed_images=()
    
    print_info "计划验证 $total_images 个镜像"
    print_info "============================================"
    
    # 验证源码镜像
    print_info "验证源码镜像 (${#source_images[@]} 个):"
    for image_base in "${source_images[@]}"; do
        local target_image="${registry}/${image_base}:${tag}"
        
        printf "  检查: %-45s" "$target_image"
        if verify_image "$target_image" 5; then
            echo "    ✓ 可用"
            ((success_count++))
        else
            echo "    ✗ 不可用"
            failed_images+=("$target_image")
        fi
    done
    
    echo
    # 验证基础镜像
    print_info "验证基础镜像 (${#base_image_patterns[@]} 个):"
    for base_pattern in "${base_image_patterns[@]}"; do
        # 使用映射配置获取目标镜像名
        local target_image
        target_image=$(get_mapped_private_image "$base_pattern" "$registry" "$tag")
        
        printf "  检查: %-45s" "$target_image"
        if verify_image "$target_image" 5; then
            echo "    ✓ 可用"
            ((success_count++))
        else
            echo "    ✗ 不可用"
            failed_images+=("$target_image")
        fi
    done
    
    echo
    print_info "============================================"
    print_info "验证结果汇总:"
    print_info "总计镜像: $total_images"
    print_success "验证通过: $success_count"
    print_error "验证失败: $((total_images - success_count))"
    
    if [[ ${#failed_images[@]} -gt 0 ]]; then
        echo
        print_error "失败镜像列表:"
        for failed_image in "${failed_images[@]}"; do
            echo "  ✗ $failed_image"
        done
        
        echo
        print_info "建议操作:"
        print_info "1. 检查网络连接和仓库权限"
        print_info "2. 重新运行基础镜像迁移脚本:"
        print_info "   ./scripts/migrate-base-images.sh $registry"
        print_info "3. 重新构建和推送源码镜像:"
        print_info "   ./build.sh build-push $registry $tag"
        
        return 1
    else
        echo
        print_success "🎉 所有镜像验证通过！"
        return 0
    fi
}

# 快速验证关键镜像
verify_key_images() {
    local registry="$1"
    local tag="${2:-v0.3.6-dev}"
    
    if [[ -z "$registry" ]]; then
        print_error "使用方法: verify-key <registry_base> [tag]"
        return 1
    fi
    
    print_info "=== 快速验证关键镜像 ==="
    print_info "目标仓库: $registry"
    print_info "镜像标签: $tag"
    echo
    
    # 关键服务镜像
    local key_images=(
        "ai-infra-backend"
        "ai-infra-frontend" 
        "ai-infra-jupyterhub"
        "ai-infra-nginx"
    )
    
    # 关键基础镜像
    local key_base_images=(
        "postgres:15-alpine"
        "redis:7-alpine"
    )
    
    local success_count=0
    local total_count=$((${#key_images[@]} + ${#key_base_images[@]}))
    
    print_info "验证关键服务镜像:"
    for image_base in "${key_images[@]}"; do
        local target_image="${registry}/${image_base}:${tag}"
        printf "  %-40s" "$target_image"
        
        if verify_image "$target_image" 3; then
            echo " ✓"
            ((success_count++))
        else
            echo " ✗"
        fi
    done
    
    print_info "验证关键基础镜像:"
    for base_pattern in "${key_base_images[@]}"; do
        local target_image
        target_image=$(get_mapped_private_image "$base_pattern" "$registry" "$tag")
        printf "  %-40s" "$target_image"
        
        if verify_image "$target_image" 3; then
            echo " ✓"
            ((success_count++))
        else
            echo " ✗"
        fi
    done
    
    echo
    if [[ $success_count -eq $total_count ]]; then
        print_success "🎉 所有关键镜像验证通过 ($success_count/$total_count)"
        return 0
    else
        print_warning "⚠ 部分关键镜像验证失败 ($success_count/$total_count)"
        return 1
    fi
}

# ==========================================
# 清理功能
# ==========================================

# 清理本地镜像
clean_images() {
    local tag="${1:-$DEFAULT_IMAGE_TAG}"
    local force="${2:-false}"
    
    print_info "=========================================="
    print_info "清理本地 AI-Infra 镜像"
    print_info "=========================================="
    print_info "目标标签: $tag"
    echo
    
    local images_to_clean=()
    
    # 收集需要清理的镜像
    for service in $SRC_SERVICES; do
        local image="ai-infra-${service}:${tag}"
        if docker image inspect "$image" >/dev/null 2>&1; then
            images_to_clean+=("$image")
        fi
    done
    
    if [[ ${#images_to_clean[@]} -eq 0 ]]; then
        print_info "没有找到需要清理的镜像"
        return 0
    fi
    
    print_info "找到 ${#images_to_clean[@]} 个镜像:"
    for image in "${images_to_clean[@]}"; do
        echo "  • $image"
    done
    echo
    
    if [[ "$force" != "true" ]]; then
        read -p "确认删除这些镜像? (y/N): " confirm
        if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
            print_info "已取消清理操作"
            return 0
        fi
    fi
    
    # 删除镜像
    local success_count=0
    for image in "${images_to_clean[@]}"; do
        if docker rmi "$image" 2>/dev/null; then
            print_success "✓ 已删除: $image"
            success_count=$((success_count + 1))
        else
            print_error "✗ 删除失败: $image"
        fi
    done
    
    print_success "清理完成: $success_count/${#images_to_clean[@]} 成功"
}

# 清理所有AI-Infra相关资源（镜像、容器、数据卷、配置文件）
clean_all() {
    local force="${1:-false}"
    
    # 处理帮助参数
    if [[ "$1" == "--help" || "$1" == "-h" ]]; then
        echo "clean-all - 完整清理AI-Infra系统"
        echo
        echo "用法: $0 clean-all [--force]"
        echo
        echo "选项:"
        echo "  --force    跳过确认提示，强制执行清理"
        echo
        echo "说明:"
        echo "  此命令将删除所有AI-Infra相关的资源："
        echo "  • 容器和服务"
        echo "  • 镜像"
        echo "  • 数据卷（数据库、文件等）"
        echo "  • 配置文件（.env.prod, docker-compose.yml）"
        echo
        echo "警告: 此操作不可逆转，请谨慎使用！"
        return 0
    fi
    
    print_info "=========================================="
    print_info "完整清理 AI-Infra 系统"
    print_info "=========================================="
    print_warning "⚠️  这将删除所有AI-Infra相关的:"
    print_warning "   • 容器和服务"
    print_warning "   • 镜像"
    print_warning "   • 数据卷（数据库、文件等）"
    print_warning "   • 配置文件（.env.prod, docker-compose.yml）"
    echo
    
    if [[ "$force" != "true" ]]; then
        read -p "确认执行完整清理? 这将删除所有数据! (yes/NO): " confirm
        if [[ "$confirm" != "yes" ]]; then
            print_info "已取消清理操作"
            return 0
        fi
    fi
    
    # 1. 停止并删除容器
    print_info "1. 停止并删除容器..."
    if [[ -f "docker-compose.yml" && -f ".env.prod" ]]; then
        ENV_FILE=.env.prod docker-compose -f docker-compose.yml --env-file .env.prod down --remove-orphans 2>/dev/null || true
    fi
    
    # 删除所有ai-infra相关容器
    local containers=$(docker ps -aq --filter "name=ai-infra" 2>/dev/null || true)
    if [[ -n "$containers" ]]; then
        docker rm -f $containers 2>/dev/null || true
        print_success "✓ 容器清理完成"
    else
        print_info "  没有找到相关容器"
    fi
    
    # 2. 删除镜像
    print_info "2. 删除镜像..."
    local images=$(docker images --format "{{.Repository}}:{{.Tag}}" | grep "ai-infra" 2>/dev/null || true)
    if [[ -n "$images" ]]; then
        echo "$images" | xargs docker rmi -f 2>/dev/null || true
        print_success "✓ 镜像清理完成"
    else
        print_info "  没有找到相关镜像"
    fi
    
    # 3. 删除数据卷
    print_info "3. 删除数据卷..."
    local volumes=$(docker volume ls --format "{{.Name}}" | grep "ai-infra" 2>/dev/null || true)
    if [[ -n "$volumes" ]]; then
        echo "$volumes" | xargs docker volume rm -f 2>/dev/null || true
        print_success "✓ 数据卷清理完成"
    else
        print_info "  没有找到相关数据卷"
    fi
    
    # 4. 删除网络
    print_info "4. 删除网络..."
    local networks=$(docker network ls --format "{{.Name}}" | grep "ai-infra" 2>/dev/null || true)
    if [[ -n "$networks" ]]; then
        echo "$networks" | xargs docker network rm 2>/dev/null || true
        print_success "✓ 网络清理完成"
    else
        print_info "  没有找到相关网络"
    fi
    
    # 5. 删除配置文件
    print_info "5. 删除配置文件..."
    local files_removed=0
    if [[ -f ".env.prod" ]]; then
        rm -f .env.prod
        files_removed=$((files_removed + 1))
        print_info "  ✓ 删除 .env.prod"
    fi
    if [[ -f "docker-compose.yml" ]]; then
        rm -f docker-compose.yml
        files_removed=$((files_removed + 1))
        print_info "  ✓ 删除 docker-compose.yml"
    fi
    if [[ $files_removed -gt 0 ]]; then
        print_success "✓ 配置文件清理完成 ($files_removed 个文件)"
    else
        print_info "  没有找到配置文件"
    fi
    
    # 6. 清理备份文件
    print_info "6. 清理备份文件..."
    local backup_files=$(find . -maxdepth 1 -name "*.env.prod.backup.*" -o -name "docker-compose.yml.bak" 2>/dev/null || true)
    if [[ -n "$backup_files" ]]; then
        echo "$backup_files" | xargs rm -f 2>/dev/null || true
        print_success "✓ 备份文件清理完成"
    else
        print_info "  没有找到备份文件"
    fi
    
    echo
    print_success "🎉 完整清理完成！"
    print_info "提示: 使用以下命令重新部署系统:"
    print_info "  1. ./build.sh create-env-prod intranet \"\" v0.3.6"
    print_info "  2. ./build.sh build-all \"\" v0.3.6"
    print_info "  3. docker compose -f docker-compose.yml.example up -d"
}

# 重置数据库（仅删除数据库相关数据卷）
reset_database() {
    local force="${1:-false}"
    
    # 处理帮助参数
    if [[ "$1" == "--help" || "$1" == "-h" ]]; then
        echo "reset-db - 重置数据库"
        echo
        echo "用法: $0 reset-db [--force]"
        echo
        echo "选项:"
        echo "  --force    跳过确认提示，强制执行重置"
        echo
        echo "说明:"
        echo "  此命令将删除所有数据库相关的数据卷："
        echo "  • PostgreSQL 数据"
        echo "  • Redis 数据"
        echo "  • JupyterHub 数据"
        echo "  • Gitea 数据"
        echo
        echo "注意:"
        echo "  • 镜像和容器不会被删除"
        echo "  • 配置文件不会被删除"
        echo "  • 此操作不可逆转，请谨慎使用！"
        return 0
    fi
    
    print_info "=========================================="
    print_info "重置数据库"
    print_info "=========================================="
    print_warning "⚠️  这将删除所有数据库数据:"
    print_warning "   • PostgreSQL 数据"
    print_warning "   • Redis 数据"
    print_warning "   • JupyterHub 数据"
    print_warning "   • Gitea 数据"
    echo
    
    if [[ "$force" != "true" ]]; then
        read -p "确认重置数据库? 这将删除所有数据! (yes/NO): " confirm
        if [[ "$confirm" != "yes" ]]; then
            print_info "已取消重置操作"
            return 0
        fi
    fi
    
    # 停止相关服务
    print_info "停止数据库相关服务..."
    if [[ -f "docker-compose.yml" && -f ".env.prod" ]]; then
        ENV_FILE=.env.prod docker-compose -f docker-compose.yml --env-file .env.prod stop postgres redis jupyterhub gitea backend backend-init 2>/dev/null || true
    fi
    
    # 删除数据库相关数据卷
    print_info "删除数据库数据卷..."
    local db_volumes=(
        "ai-infra-postgres-data"
        "ai-infra-redis-data"
        "ai-infra-jupyterhub-data"
        "ai-infra-jupyterhub-notebooks"
        "ai-infra-gitea-data"
    )
    
    local removed_count=0
    for volume in "${db_volumes[@]}"; do
        if docker volume inspect "$volume" >/dev/null 2>&1; then
            if docker volume rm "$volume" 2>/dev/null; then
                print_info "  ✓ 删除 $volume"
                removed_count=$((removed_count + 1))
            else
                print_error "  ✗ 删除失败 $volume (可能被容器使用)"
            fi
        fi
    done
    
    if [[ $removed_count -gt 0 ]]; then
        print_success "✓ 数据库重置完成 ($removed_count 个数据卷)"
        print_info "提示: 使用 ./build.sh prod-up 重新启动服务"
    else
        print_info "没有找到需要重置的数据库数据卷"
    fi
}

# Kafka服务管理函数
# 启动Kafka服务 (KRaft模式)
start_kafka_services() {
    print_info "启动Kafka服务 (KRaft模式)..."
    local compose_file="${1:-docker-compose.yml}"
    
    # 首先渲染docker-compose.yml模板
    if [[ "$compose_file" == "docker-compose.yml" ]]; then
        print_info "渲染Docker Compose模板..."
        render_docker_compose_templates
    fi
    
    if [[ ! -f "$compose_file" ]]; then
        print_error "未找到 docker-compose 文件: $compose_file"
        return 1
    fi
    
    print_info "启动 Kafka (KRaft模式，无需Zookeeper)..."
    docker compose -f "$compose_file" up -d kafka
    
    # 等待Kafka启动
    print_info "等待 Kafka 启动..."
    sleep 20
    
    print_info "启动 Kafka UI..."
    docker compose -f "$compose_file" up -d kafka-ui
    
    print_success "✓ Kafka服务启动完成 (KRaft模式)"
    print_info "Kafka UI 访问地址: http://localhost:9095"
    print_info "Kafka Bootstrap Server: localhost:9094"
}

# 检查Kafka服务状态
check_kafka_status() {
    print_info "检查Kafka服务状态 (KRaft模式)..."
    local compose_file="${1:-docker-compose.yml}"
    
    # 检查服务状态
    echo "Kafka服务状态:"
    docker compose -f "$compose_file" ps kafka kafka-ui
    
    # 检查Kafka连接性
    print_info "检查Kafka连接性..."
    if docker compose -f "$compose_file" exec kafka kafka-topics --bootstrap-server localhost:9092 --list >/dev/null 2>&1; then
        print_success "✓ Kafka服务运行正常 (KRaft模式)"
        
        # 显示集群信息
        print_info "Kafka集群信息:"
        docker compose -f "$compose_file" exec kafka kafka-broker-api-versions --bootstrap-server localhost:9092 | head -5
    else
        print_error "✗ Kafka服务连接失败"
        return 1
    fi
}

# 创建Kafka测试主题
create_kafka_test_topic() {
    local topic_name="${1:-test-topic}"
    local partitions="${2:-3}"
    local replication_factor="${3:-1}"
    local compose_file="${4:-docker-compose.yml}"
    
    print_info "创建Kafka测试主题: $topic_name"
    
    docker compose -f "$compose_file" exec kafka kafka-topics \
        --create \
        --bootstrap-server localhost:9092 \
        --topic "$topic_name" \
        --partitions "$partitions" \
        --replication-factor "$replication_factor"
    
    if [[ $? -eq 0 ]]; then
        print_success "✓ 主题 '$topic_name' 创建成功"
    else
        print_error "✗ 主题 '$topic_name' 创建失败"
        return 1
    fi
}

# 列出Kafka主题
list_kafka_topics() {
    local compose_file="${1:-docker-compose.yml}"
    
    print_info "Kafka主题列表:"
    docker compose -f "$compose_file" exec kafka kafka-topics \
        --bootstrap-server localhost:9092 \
        --list
}

# 发送测试消息到Kafka
send_kafka_test_message() {
    local topic_name="${1:-test-topic}"
    local message="${2:-Hello Kafka from AI Infrastructure Matrix}"
    local compose_file="${3:-docker-compose.yml}"
    
    print_info "发送测试消息到主题: $topic_name"
    
    echo "$message" | docker compose -f "$compose_file" exec -T kafka kafka-console-producer \
        --bootstrap-server localhost:9092 \
        --topic "$topic_name"
    
    if [[ $? -eq 0 ]]; then
        print_success "✓ 消息发送成功"
    else
        print_error "✗ 消息发送失败"
        return 1
    fi
}

# 消费Kafka测试消息
consume_kafka_test_message() {
    local topic_name="${1:-test-topic}"
    local max_messages="${2:-5}"
    local compose_file="${3:-docker-compose.yml}"
    
    print_info "从主题 '$topic_name' 消费消息 (最多 $max_messages 条):"
    
    docker compose -f "$compose_file" exec kafka kafka-console-consumer \
        --bootstrap-server localhost:9092 \
        --topic "$topic_name" \
        --from-beginning \
        --max-messages "$max_messages"
}

# 完整的Kafka测试流程
test_kafka_full() {
    local compose_file="${1:-docker-compose.yml}"
    local topic_name="ai-infra-test-$(date +%s)"
    
    print_info "=========================================="
    print_info "开始Kafka完整测试流程"
    print_info "=========================================="
    
    # 1. 检查服务状态
    check_kafka_status "$compose_file" || return 1
    
    # 2. 创建测试主题
    create_kafka_test_topic "$topic_name" 3 1 "$compose_file" || return 1
    
    # 3. 列出主题
    list_kafka_topics "$compose_file"
    
    # 4. 发送测试消息
    send_kafka_test_message "$topic_name" "Test message 1: $(date)" "$compose_file" || return 1
    send_kafka_test_message "$topic_name" "Test message 2: System check" "$compose_file" || return 1
    send_kafka_test_message "$topic_name" "Test message 3: Integration test" "$compose_file" || return 1
    
    # 5. 消费消息
    print_info "等待消息传播..."
    sleep 2
    consume_kafka_test_message "$topic_name" 10 "$compose_file"
    
    # 6. 清理测试主题
    print_info "清理测试主题: $topic_name"
    docker compose -f "$compose_file" exec kafka kafka-topics \
        --delete \
        --bootstrap-server localhost:9092 \
        --topic "$topic_name"
    
    print_success "✓ Kafka完整测试完成"
    print_info "Kafka UI管理界面: http://localhost:9095"
}

# 停止Kafka服务
stop_kafka_services() {
    local compose_file="${1:-docker-compose.yml}"
    
    print_info "停止Kafka服务 (KRaft模式)..."
    docker compose -f "$compose_file" stop kafka-ui kafka
    print_success "✓ Kafka服务已停止"
}

# 重启Kafka服务
restart_kafka_services() {
    local compose_file="${1:-docker-compose.yml}"
    
    print_info "重启Kafka服务 (KRaft模式)..."
    stop_kafka_services "$compose_file"
    sleep 5
    start_kafka_services "$compose_file"
}

# 查看Kafka日志
show_kafka_logs() {
    local service="${1:-kafka}"
    local compose_file="${2:-docker-compose.yml}"
    local follow="${3:-}"
    
    case "$service" in
        "kafka-ui"|"ui")
            if [[ "$follow" == "--follow" || "$follow" == "-f" ]]; then
                docker compose -f "$compose_file" logs -f kafka-ui
            else
                docker compose -f "$compose_file" logs --tail=50 kafka-ui
            fi
            ;;
        "kafka"|*)
            if [[ "$follow" == "--follow" || "$follow" == "-f" ]]; then
                docker compose -f "$compose_file" logs -f kafka
            else
                docker compose -f "$compose_file" logs --tail=50 kafka
            fi
            ;;
    esac
}

# 显示帮助信息
show_help() {
    echo "AI Infrastructure Matrix - 构建脚本 v$VERSION"
    echo
    echo "用法: $0 [--force|--skip-pull|--skip-cache-check|--china-mirror|--no-source-maps] <命令> [参数...]"
    echo
    echo "全局选项:"
    echo "  --force              - 强制重新构建/跳过镜像拉取"
    echo "  --skip-pull          - 跳过镜像拉取，使用本地镜像"
    echo "  --skip-cache-check   - 跳过智能缓存检查，总是构建"
    echo "  --china-mirror       - 使用中国npm镜像加速前端构建"
    echo "  --no-source-maps     - 禁用源码映射生成（优化构建性能）"
    echo
    echo "主要命令:"
    echo "  list [tag] [registry]           - 列出所有服务和镜像"
    echo "  check-status [tag] [registry]   - 检查镜像构建状态（需求32）"
    echo "  build <service> [tag] [registry] - 构建单个服务"
    echo "  build-all [tag] [registry]      - 构建所有服务（智能过滤）"
    echo "  build-push <registry> [tag]     - 构建并推送所有服务"
    echo "  push-all <registry> [tag]       - 推送所有服务"
    echo
    echo "智能构建缓存（新增）:"
    echo "  cache-stats                     - 显示构建缓存统计信息"
    echo "  clean-cache [service]           - 清理构建缓存（不指定则清理所有）"
    echo "  build-info <service> [tag]      - 显示镜像的构建信息"
    echo "  • 自动检测文件变化，无变化则复用镜像"
    echo "  • 每次构建生成唯一BUILD_ID和时间戳"
    echo "  • 使用SHA256哈希追踪源码和配置变化"
    echo "  • 使用 --skip-cache-check 跳过缓存检查"
    echo
    echo "智能构建特性（需求32）:"
    echo "  • 自动检测镜像构建状态"
    echo "  • 只构建缺失或无效的镜像"
    echo "  • 避免 --no-cache 全量构建浪费时间"
    echo "  • 使用 --force 参数强制重建所有镜像"
    echo
    echo "CI/CD和生产环境命令 (重点推荐):"
    echo "  ci-build <registry> [tag] [host]     - CI/CD完整构建流程（外网环境）"
    echo "  prod-start [registry] [tag] [host] [port] - 生产环境服务启动（内网环境）"
    echo "    • ci-build: 适用于有外网访问的构建环境，完成构建、推送全流程"
    echo "    • prod-start: 适用于无外网访问的生产环境，拉取镜像并启动服务"
    echo
    echo "自动化补丁管理:"
    echo "  patch <patch-name> [service] [rebuild] - 应用代码补丁并重建服务"
    echo "  generate-patch <service> [output]    - 生成服务补丁文件"
    echo "    可用补丁: ldap-fix, cors-fix, frontend-build-fix, backend-auth-fix, custom"
    echo
    echo "依赖镜像:"
    echo "  deps-pull <registry> [tag]      - 拉取依赖镜像"
    echo "  deps-push <registry> [tag]      - 推送依赖镜像"
    echo "  deps-all <registry> [tag]       - 拉取、标记并推送依赖镜像"
    echo
    echo "AI Harbor镜像拉取:"
    echo "  harbor-pull-services [registry] [tag] - 从AI Harbor拉取AI-Infra服务镜像"
    echo "  harbor-pull-deps [registry] [tag]     - 从AI Harbor拉取依赖镜像"
    echo "  harbor-pull-all [registry] [tag]      - 从AI Harbor拉取所有镜像"
    echo
    echo "生产环境:"
    echo "  prod-deploy <host> [registry] [tag] - 部署到指定HOST（自动配置域名）"
    echo "  prod-up [registry] [tag]        - 启动生产环境"
    echo "  prod-down                       - 停止生产环境"
    echo "  prod-status                     - 查看状态"
    echo "  prod-logs [service] [--follow]  - 查看日志"
    echo "  generate-passwords [file] [--force] - 生成生产环境强密码"
    echo
    echo "Kafka服务管理 (KRaft模式):"
    echo "  kafka-start [compose-file]      - 启动Kafka服务 (KRaft模式，无需Zookeeper)"
    echo "  kafka-stop [compose-file]       - 停止Kafka服务"
    echo "  kafka-restart [compose-file]    - 重启Kafka服务"
    echo "  kafka-status [compose-file]     - 检查Kafka服务状态"
    echo "  kafka-test [compose-file]       - 运行完整Kafka测试流程"
    echo "  kafka-topics [compose-file]     - 列出Kafka主题"
    echo "  kafka-logs [service] [compose-file] [--follow] - 查看日志 (service: kafka|kafka-ui)"
    echo
    echo "离线部署:"
    echo "  export-offline [output_dir] [tag] [include_kafka] - 导出离线镜像包"
    echo "  push-to-internal <registry> [tag] [include_kafka] - 推送镜像到内部仓库"
    echo "  prepare-offline <registry> [tag] [output_dir] [include_kafka] - 准备完整离线部署包"
    echo
    echo "统一构建和部署 (公共参数接口):"
    echo "  unified-build <registry> <tag> <host> <port> <scheme>     - 统一构建所有镜像"
    echo "  unified-build-push <registry> <tag> <host> <port> <scheme> - 统一构建并推送所有镜像" 
    echo "  unified-deploy <registry> <tag> <host> <port> <scheme> [compose] - 统一部署服务"
    echo "  unified-all <registry> <tag> <host> <port> <scheme> [compose]    - 一键构建、推送、部署"
    echo "  all-in-one <registry> <tag> <host> <port> <scheme> [compose]     - 一键构建、推送、部署 (别名)"
    echo
    echo "  参数说明:"
    echo "    registry: 镜像仓库地址 (默认: aiharbor.msxf.local/aihpc)"
    echo "    tag:      镜像标签 (默认: $DEFAULT_IMAGE_TAG)" 
    echo "    host:     外部访问主机 (默认: 172.20.10.11)"
    echo "    port:     外部访问端口 (默认: 80)"
    echo "    scheme:   访问协议 (默认: http)"
    echo "    compose:  docker-compose文件 (默认: docker-compose.yml)"
    echo
    echo "SingleUser 智能构建:"
    echo "  build-singleuser [mode] [tag] [registry] - 智能构建SingleUser镜像"
    echo "    模式: auto (自动检测), offline (离线友好), online (标准模式)"
    echo "  detect-network                  - 检测当前网络环境"
    echo "  restore-singleuser              - 恢复SingleUser Dockerfile到原始状态"
    echo
    echo "工具命令:"
    echo "  clean [tag] [--force]           - 清理镜像"
    echo "  clean-all [--force]             - 完整清理（镜像、容器、数据卷、配置文件）"
    echo "  reset-db [--force]              - 重置数据库（仅删除数据库数据卷）"
    echo "  verify <registry> [tag]         - 验证镜像"
    echo "  create-env [dev|prod] [--force] - 创建环境配置"
    echo "  detect-ip [interface] [--all]   - 检测网卡IP地址（支持自动检测和指定网卡）"
    echo "  validate-env                    - 校验环境配置"
    echo "  render-templates [nginx|jupyterhub|docker-compose|env|all] - 渲染配置模板"
    echo "  sync-config [force] - 同步所有配置文件(.env, docker-compose.yml)"
    echo "    • docker-compose 额外参数: --oceanbase-init-dir <path> 指定 OceanBase 初始化目录"
    echo "  version                         - 显示版本"
    echo "  help                            - 显示帮助"
    echo
    echo "动态配置管理:"
    echo "  update-host [host|auto]         - 更新外部主机配置（auto=自动检测）"
    echo "  update-port <port>              - 更新外部端口配置（自动计算相关端口）"
    echo "  quick-deploy [port] [host]      - 一键更新配置并重新部署（默认8080 auto）"
    echo
    echo "===================================================================================="
    echo "🚀 CI/CD和生产环境部署实例 (强烈推荐):"
    echo "===================================================================================="
    echo "  # CI/CD环境 (有外网访问): 完整构建并推送到仓库"
    echo "  $0 ci-build harbor.company.com/ai-infra v1.0.0"
    echo "  $0 ci-build harbor.company.com/ai-infra v1.0.0 192.168.1.100   # 指定外部访问地址"
    echo
    echo "  # 生产环境 (无外网访问): 从内部仓库启动服务"
    echo "  $0 prod-start aiharbor.msxf.local/aihpc v1.0.0"
    echo "  $0 prod-start aiharbor.msxf.local/aihpc v1.0.0 192.168.1.100 8080   # 指定访问地址和端口"
    echo "  $0 prod-start \"\" v1.0.0                          # 使用本地镜像启动"
    echo
    echo "===================================================================================="
    echo "🔧 自动化补丁管理实例:"
    echo "===================================================================================="
    echo "  # 修复LDAP字段映射问题（自动应用补丁并重建）"
    echo "  $0 patch ldap-fix"
    echo
    echo "  # 应用补丁但不重建服务"
    echo "  $0 patch ldap-fix \"\" false"
    echo
    echo "  # 生成自定义补丁文件"
    echo "  $0 generate-patch backend ./backend-fix.patch"
    echo
    echo "  # 应用自定义补丁"
    echo "  $0 patch custom backend"
    echo
    echo "===================================================================================="
    echo "🔧 统一构建和部署实例 (高级用户使用):"
    echo "===================================================================================="
    echo "  # 一键构建、推送、部署到生产环境 (所有服务一条命令搞定)"
    echo "  $0 unified-all aiharbor.msxf.local/aihpc v1.2.0 172.20.10.11 80 http"
    echo
    echo "  # 分步骤统一操作"
    echo "  $0 unified-build-push aiharbor.msxf.local/aihpc v1.2.0 172.20.10.11 80 http   # 构建并推送"
    echo "  $0 unified-deploy aiharbor.msxf.local/aihpc v1.2.0 172.20.10.11 80 http       # 部署启动"
    echo
    echo "  # 本地开发环境快速启动 (使用默认参数)"
    echo "  $0 unified-all                                      # 使用所有默认值"
    echo "  # 等价于: $0 unified-all aiharbor.msxf.local/aihpc $DEFAULT_IMAGE_TAG 172.20.10.11 80 http"
    echo
    echo "  # 自定义域名和端口"
    echo "  $0 unified-all harbor.company.com/ai v2.0.0 ai.company.com 8080 https"
    echo "  # 访问地址: https://ai.company.com:8080"
    echo
    echo "===================================================================================="
    echo "�📦 CI/CD服务器运行实例 (构建和推送镜像):"
    echo "===================================================================================="
    echo "  # 构建所有服务并推送到私有仓库"
    echo "  $0 build-push harbor.example.com/ai-infra v1.2.0"
    echo
    echo "  # 推送依赖镜像到私有仓库"
    echo "  $0 deps-all harbor.example.com/ai-infra v1.2.0"
    echo
    echo "  # 分步骤操作（推荐用于CI/CD Pipeline）"
    echo "  $0 build-all v1.2.0                                    # 步骤1: 构建所有服务"
    echo "  $0 push-all harbor.example.com/ai-infra v1.2.0         # 步骤2: 推送项目镜像"
    echo "  $0 deps-push harbor.example.com/ai-infra v1.2.0        # 步骤3: 推送依赖镜像"
    echo
    echo "===================================================================================="
    echo "🚀 生产节点运行实例 (启动服务):"
    echo "===================================================================================="
    echo "  # 从AI Harbor拉取镜像完整部署流程"
    echo "  $0 harbor-pull-all aiharbor.msxf.local/aihpc v1.2.0    # 步骤1: 拉取所有镜像"
    echo "  $0 generate-passwords .env.prod --force                # 步骤2: 生成强密码"
    echo "  docker compose -f docker-compose.yml.example up -d     # 步骤3: 启动所有服务"
    echo
    echo "  # 标准私有仓库部署流程"
    echo "  $0 generate-passwords .env.prod --force                # 步骤1: 生成强密码"
    echo "  docker compose -f docker-compose.yml.example up -d     # 步骤2: 启动所有服务"
    echo
    echo "  # 快速启动 (生产配置已存在)"
    echo "  $0 prod-up harbor.example.com/ai-infra v1.2.0"
    echo
    echo "  # 本地镜像部署 (无需registry)"
    echo "  $0 generate-passwords .env.prod                        # 生成密码"
    echo "  docker compose -f docker-compose.yml.example up -d     # 启动服务"
    echo
    echo "  # 服务管理"
    echo "  $0 prod-status                                         # 查看服务状态"
    echo "  $0 prod-logs jupyterhub --follow                       # 查看实时日志"
    echo "  $0 prod-down                                           # 停止所有服务"
    echo
    echo "===================================================================================="
    echo "💡 常用开发实例:"
    echo "===================================================================================="
    echo "  # 从AI Harbor快速获取镜像进行本地开发"
    echo "  $0 harbor-pull-services aiharbor.msxf.local/aihpc v1.2.0  # 拉取AI-Infra服务"
    echo "  $0 harbor-pull-deps aiharbor.msxf.local/aihpc             # 拉取依赖镜像"
    echo "  docker compose -f docker-compose.yml.example up -d        # 启动服务"
    echo
    echo "  # 本地开发测试"
    echo "  $0 build-all test-v0.3.6-dev                          # 构建测试版本"
    echo "  $0 build frontend v0.3.6-dev                          # 构建前端（Docker容器内）"
    echo "  docker compose -f docker-compose.yml.example up -d backend frontend  # 启动核心服务"
    echo
    echo "  # 单服务调试"
    echo "  $0 build backend test-debug                           # 构建调试版本"
    echo "  docker compose up -d postgres redis                  # 启动依赖"
    echo "  docker run --rm -it ai-infra-backend:test-debug bash  # 交互式调试"
    echo
    echo "===================================================================================="
    echo "🔧 动态配置管理实例:"
    echo "===================================================================================="
    echo "  # 自动检测外部IP并更新配置"
    echo "  $0 update-host auto                                   # 自动检测外部主机IP"
    echo "  $0 build nginx --force && docker compose restart nginx  # 应用新配置"
    echo
    echo "  # 手动指定外部主机"
    echo "  $0 update-host 192.168.1.100                         # 设置外部主机为指定IP"
    echo
    echo "  # 修改外部端口（便捷部署不同环境）"
    echo "  $0 update-port 9090                                  # 更新外部端口为9090"
    echo "                                                        # 自动计算：主入口9090，JupyterHub9098，Gitea4020"
    echo "  $0 build nginx --force                               # 重新构建nginx配置"
    echo "  docker compose down && docker compose up -d          # 重启所有服务"
    echo
    echo "  # 快速切换部署端口"
    echo "  $0 update-port 8080 && $0 build nginx --force        # 切换到8080端口并更新配置"
    echo "  $0 update-port 9000 && $0 build nginx --force        # 切换到9000端口并更新配置"
    echo
    echo "===================================================================================="
    echo "📊 Kafka服务管理实例 (KRaft模式):"
    echo "===================================================================================="
    echo "  # 启动Kafka服务集群 (KRaft模式，性能更优)"
    echo "  $0 kafka-start                                       # 启动Kafka服务 (无需Zookeeper)"
    echo "  $0 kafka-status                                      # 检查服务状态"
    echo
    echo "  # 完整Kafka测试流程"
    echo "  $0 kafka-test                                        # 自动创建主题、发送消息、消费消息"
    echo "  $0 kafka-topics                                      # 列出所有主题"
    echo
    echo "  # 日志查看和调试"
    echo "  $0 kafka-logs kafka --follow                         # 查看Kafka实时日志"
    echo "  $0 kafka-logs kafka-ui                               # 查看Kafka UI日志"
    echo
    echo "  # 服务管理"
    echo "  $0 kafka-restart                                     # 重启Kafka服务"
    echo "  $0 kafka-stop                                        # 停止Kafka服务"
    echo
    echo "  # Kafka UI管理界面访问"
    echo "  # http://localhost:9095                              # Kafka管理界面"
    echo "  # Bootstrap Server: localhost:9094                  # 外部连接地址"
    echo
    echo "===================================================================================="
    echo "� 离线部署实例:"
    echo "===================================================================================="
    echo "  # 导出离线镜像包（包含Kafka）"
    echo "  $0 export-offline ./offline-images v1.2.0 true"
    echo
    echo "  # 推送镜像到内部仓库"
    echo "  $0 push-to-internal harbor.company.com/ai-infra v1.2.0 true"
    echo
    echo "  # 准备完整离线部署包（导出+推送+配置）"
    echo "  $0 prepare-offline harbor.company.com/ai-infra v1.2.0 ./offline-deployment true"
    echo
    echo "  # 离线环境部署流程"
    echo "  # 1. 复制离线部署包到目标环境"
    echo "  # 2. cd offline-deployment && ./deploy-offline.sh"
    echo "  # 3. 或手动: ./images/import-images.sh && docker compose up -d"
    echo
    echo "===================================================================================="
    echo "�📋 模板渲染和配置管理实例:"
    echo "===================================================================================="
    echo "  # 渲染docker-compose.yml配置"
    echo "  $0 render-templates docker-compose                   # 从example生成docker-compose.yml"
    echo "  $0 render-templates docker-compose --oceanbase-init-dir ./data/oceanbase/init.d"
    echo "  $0 render-templates all                              # 渲染所有配置模板"
    echo
    echo "  # 完整的Kafka部署流程"
    echo "  $0 render-templates docker-compose                   # 1. 生成最新配置"
    echo "  $0 kafka-start                                       # 2. 启动Kafka服务"
    echo "  $0 kafka-test                                        # 3. 测试Kafka功能"
    echo
    echo
    echo "===================================================================================="
    echo "⚠️  重要提醒:"
    echo "  • 首次部署必须运行 generate-passwords 生成强密码"
    echo "  • 默认管理员账户: admin / admin123 (部署后请立即修改)"
    echo "  • 生产环境配置文件 docker-compose.yml 会被自动生成，请勿手动编辑"
    echo "  • 服务访问端口: Web界面:8080, JupyterHub:8088, Gitea:3010"
    echo "===================================================================================="
}

# ==========================================
# 离线部署功能
# ==========================================

# 导出离线镜像
export_offline_images() {
    # 处理帮助参数
    if [[ "$1" == "--help" || "$1" == "-h" ]]; then
        echo "export-offline - 导出离线镜像包"
        echo
        echo "用法: $0 export-offline [output_dir] [tag] [include_kafka]"
        echo
        echo "参数:"
        echo "  output_dir     输出目录 (默认: ./offline-images)"
        echo "  tag           镜像标签 (默认: $DEFAULT_IMAGE_TAG)"
        echo "  include_kafka  是否包含Kafka镜像 (默认: true)"
        echo
        echo "说明:"
        echo "  导出所有AI-Infra服务镜像和依赖镜像到指定目录"
        echo "  自动生成镜像清单文件和导入脚本"
        echo "  支持包含或排除Kafka相关镜像"
        echo
        echo "示例:"
        echo "  $0 export-offline ./my-images v1.0.0 true"
        echo "  $0 export-offline ./images v0.3.6-dev false"
        return 0
    fi
    
    local output_dir="${1:-./offline-images}"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    local include_kafka="${3:-true}"
    
    print_info "=========================================="
    print_info "导出离线镜像"
    print_info "=========================================="
    print_info "输出目录: $output_dir"
    print_info "镜像标签: $tag"
    print_info "包含Kafka: $include_kafka"
    echo
    
    # 创建输出目录
    mkdir -p "$output_dir"
    
    # 导出AI-Infra服务镜像
    print_info "📦 导出AI-Infra服务镜像..."
    local services_exported=0
    local services_failed=()
    
    for service in $SRC_SERVICES; do
        local image_name="ai-infra-${service}:${tag}"
        local output_file="${output_dir}/ai-infra-${service}-${tag}.tar"
        
        print_info "→ 导出: $image_name"
        if docker image inspect "$image_name" >/dev/null 2>&1; then
            if docker save "$image_name" -o "$output_file"; then
                print_success "  ✓ 导出成功: $(basename "$output_file")"
                services_exported=$((services_exported + 1))
            else
                print_error "  ✗ 导出失败: $image_name"
                services_failed+=("$service")
            fi
        else
            print_warning "  ! 镜像不存在，跳过: $image_name"
            services_failed+=("$service")
        fi
    done
    
    # 导出依赖镜像
    print_info "📦 导出依赖镜像..."
    local dependencies_exported=0
    local dependencies_failed=()
    
    # 基础依赖镜像
    local base_dependencies=(
        "postgres:15-alpine"
        "redis:7-alpine"
        "nginx:1.27-alpine"
        "tecnativa/tcp-proxy:latest"
        "minio/minio:latest"
        "osixia/openldap:stable"
        "osixia/phpldapadmin:stable"
        "redislabs/redisinsight:latest"
    )
    
    # 如果包含Kafka，添加Kafka相关镜像
    if [[ "$include_kafka" == "true" ]]; then
        local kafka_dependencies=(
            "confluentinc/cp-kafka:7.5.0"
            "provectuslabs/kafka-ui:latest"
        )
        base_dependencies+=("${kafka_dependencies[@]}")
        print_info "  包含Kafka镜像: confluentinc/cp-kafka:7.5.0, provectuslabs/kafka-ui:latest"
    fi
    
    for dep_image in "${base_dependencies[@]}"; do
        # 生成安全的文件名
        local safe_name=$(echo "$dep_image" | sed 's|/|-|g' | sed 's|:|_|g')
        local output_file="${output_dir}/${safe_name}.tar"
        
        print_info "→ 导出: $dep_image"
        if docker image inspect "$dep_image" >/dev/null 2>&1; then
            if docker save "$dep_image" -o "$output_file"; then
                print_success "  ✓ 导出成功: $(basename "$output_file")"
                dependencies_exported=$((dependencies_exported + 1))
            else
                print_error "  ✗ 导出失败: $dep_image"
                dependencies_failed+=("$dep_image")
            fi
        else
            print_warning "  ! 镜像不存在，跳过: $dep_image"
            dependencies_failed+=("$dep_image")
        fi
    done
    
    # 生成镜像清单文件
    print_info "📋 生成镜像清单..."
    local manifest_file="${output_dir}/images-manifest.txt"
    cat > "$manifest_file" << EOF
# AI Infrastructure Matrix 离线镜像清单
# 生成时间: $(date)
# 镜像标签: $tag
# 包含Kafka: $include_kafka

# AI-Infra服务镜像 (${services_exported}个)
EOF
    
    for service in $SRC_SERVICES; do
        local image_name="ai-infra-${service}:${tag}"
        local output_file="ai-infra-${service}-${tag}.tar"
        if docker image inspect "$image_name" >/dev/null 2>&1; then
            echo "$image_name|$output_file" >> "$manifest_file"
        fi
    done
    
    echo "" >> "$manifest_file"
    echo "# 依赖镜像 (${dependencies_exported}个)" >> "$manifest_file"
    
    for dep_image in "${base_dependencies[@]}"; do
        local safe_name=$(echo "$dep_image" | sed 's|/|-|g' | sed 's|:|_|g')
        local output_file="${safe_name}.tar"
        if docker image inspect "$dep_image" >/dev/null 2>&1; then
            echo "$dep_image|$output_file" >> "$manifest_file"
        fi
    done
    
    # 生成导入脚本
    print_info "📜 生成导入脚本..."
    local import_script="${output_dir}/import-images.sh"
    cat > "$import_script" << 'EOF'
#!/bin/bash

# AI Infrastructure Matrix 离线镜像导入脚本
# 使用方法: ./import-images.sh [镜像目录]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
IMAGES_DIR="${1:-$SCRIPT_DIR}"
MANIFEST_FILE="${IMAGES_DIR}/images-manifest.txt"

print_info() {
    echo -e "\033[32m[INFO]\033[0m $1"
}

print_success() {
    echo -e "\033[32m[SUCCESS]\033[0m $1"
}

print_error() {
    echo -e "\033[31m[ERROR]\033[0m $1"
}

if [[ ! -f "$MANIFEST_FILE" ]]; then
    print_error "镜像清单文件不存在: $MANIFEST_FILE"
    exit 1
fi

print_info "=========================================="
print_info "导入离线镜像"
print_info "=========================================="
print_info "镜像目录: $IMAGES_DIR"
print_info "清单文件: $MANIFEST_FILE"
echo

imported_count=0
failed_count=0

while IFS='|' read -r image_name tar_file; do
    # 跳过注释和空行
    [[ "$image_name" =~ ^[[:space:]]*# ]] && continue
    [[ -z "$image_name" ]] && continue
    
    local tar_path="${IMAGES_DIR}/${tar_file}"
    
    if [[ -f "$tar_path" ]]; then
        print_info "→ 导入: $image_name"
        if docker load -i "$tar_path"; then
            print_success "  ✓ 导入成功: $image_name"
            imported_count=$((imported_count + 1))
        else
            print_error "  ✗ 导入失败: $image_name"
            failed_count=$((failed_count + 1))
        fi
    else
        print_error "  ✗ 镜像文件不存在: $tar_path"
        failed_count=$((failed_count + 1))
    fi
done < "$MANIFEST_FILE"

echo
print_info "=========================================="
print_success "导入完成: $imported_count 成功, $failed_count 失败"

if [[ $failed_count -eq 0 ]]; then
    print_success "🎉 所有镜像导入成功！"
    echo
    print_info "接下来可以："
    print_info "1. 检查镜像: docker images | grep -E 'ai-infra|postgres|redis'"
    print_info "2. 启动服务: docker compose -f docker-compose.yml.example up -d"
else
    print_error "部分镜像导入失败，请检查错误信息"
fi
EOF
    
    chmod +x "$import_script"
    
    # 生成统计信息
    print_info "=========================================="
    print_success "离线镜像导出完成！"
    echo
    print_info "📊 导出统计:"
    print_info "  • AI-Infra服务镜像: $services_exported 个"
    print_info "  • 依赖镜像: $dependencies_exported 个"
    print_info "  • 总计: $((services_exported + dependencies_exported)) 个"
    echo
    
    if [[ ${#services_failed[@]} -gt 0 || ${#dependencies_failed[@]} -gt 0 ]]; then
        print_warning "⚠️  部分镜像导出失败:"
        if [[ ${#services_failed[@]} -gt 0 ]]; then
            print_warning "  失败的服务: ${services_failed[*]}"
        fi
        if [[ ${#dependencies_failed[@]} -gt 0 ]]; then
            print_warning "  失败的依赖: ${dependencies_failed[*]}"
        fi
    fi
    
    print_info "📁 输出文件:"
    print_info "  • 镜像目录: $output_dir"
    print_info "  • 镜像清单: $manifest_file"
    print_info "  • 导入脚本: $import_script"
    echo
    print_info "📋 使用方法:"
    print_info "1. 将整个 $output_dir 目录复制到离线环境"
    print_info "2. 在离线环境运行: cd $output_dir && ./import-images.sh"
    print_info "3. 启动服务: docker compose -f docker-compose.yml.example up -d"
    
    return 0
}

# 推送镜像到内部仓库（用于离线部署准备）
push_to_internal_registry() {
    # 处理帮助参数
    if [[ "$1" == "--help" || "$1" == "-h" ]]; then
        echo "push-to-internal - 推送镜像到内部仓库"
        echo
        echo "用法: $0 push-to-internal <registry> [tag] [include_kafka]"
        echo
        echo "参数:"
        echo "  registry      内部仓库地址 (必需)"
        echo "  tag          镜像标签 (默认: $DEFAULT_IMAGE_TAG)"
        echo "  include_kafka 是否包含Kafka镜像 (默认: true)"
        echo
        echo "说明:"
        echo "  将所有AI-Infra服务镜像和依赖镜像推送到指定的内部仓库"
        echo "  支持Harbor等私有仓库格式"
        echo "  自动使用镜像映射配置进行标记转换"
        echo
        echo "示例:"
        echo "  $0 push-to-internal harbor.company.com/ai-infra v1.0.0 true"
        echo "  $0 push-to-internal registry.internal.com/project v0.3.6-dev false"
        return 0
    fi
    
    local registry="$1"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    local include_kafka="${3:-true}"
    
    if [[ -z "$registry" ]]; then
        print_error "请指定内部仓库地址"
        print_info "用法: push-to-internal <registry> [tag] [include_kafka]"
        print_info "示例: push-to-internal harbor.company.com/ai-infra v1.0.0 true"
        return 1
    fi
    
    print_info "=========================================="
    print_info "推送镜像到内部仓库"
    print_info "=========================================="
    print_info "内部仓库: $registry"
    print_info "镜像标签: $tag"
    print_info "包含Kafka: $include_kafka"
    echo
    
    local total_pushed=0
    local total_failed=0
    local failed_images=()
    
    # 推送AI-Infra服务镜像
    print_info "🚀 推送AI-Infra服务镜像..."
    for service in $SRC_SERVICES; do
        local local_image="ai-infra-${service}:${tag}"
        local target_image="${registry}/ai-infra-${service}:${tag}"
        
        print_info "→ 推送: $service"
        print_info "  本地镜像: $local_image"
        print_info "  目标镜像: $target_image"
        
        # 检查本地镜像是否存在
        if ! docker image inspect "$local_image" >/dev/null 2>&1; then
            print_error "  ✗ 本地镜像不存在: $local_image"
            failed_images+=("$local_image")
            total_failed=$((total_failed + 1))
            continue
        fi
        
        # 标记镜像
        if docker tag "$local_image" "$target_image"; then
            print_success "  ✓ 标记成功"
        else
            print_error "  ✗ 标记失败: $target_image"
            failed_images+=("$local_image")
            total_failed=$((total_failed + 1))
            continue
        fi
        
        # 推送镜像
        if docker push "$target_image"; then
            print_success "  ✓ 推送成功: $target_image"
            total_pushed=$((total_pushed + 1))
        else
            print_error "  ✗ 推送失败: $target_image"
            failed_images+=("$target_image")
            total_failed=$((total_failed + 1))
        fi
        echo
    done
    
    # 推送依赖镜像
    print_info "🚀 推送依赖镜像..."
    local base_dependencies=(
        "postgres:15-alpine"
        "redis:7-alpine"
        "nginx:1.27-alpine"
        "tecnativa/tcp-proxy:latest"
        "minio/minio:latest"
        "osixia/openldap:stable"
        "osixia/phpldapadmin:stable"
        "redislabs/redisinsight:latest"
    )
    
    # 如果包含Kafka，添加Kafka相关镜像
    if [[ "$include_kafka" == "true" ]]; then
        local kafka_dependencies=(
            "confluentinc/cp-kafka:7.5.0"
            "provectuslabs/kafka-ui:latest"
        )
        base_dependencies+=("${kafka_dependencies[@]}")
        print_info "  包含Kafka镜像推送"
    fi
    
    for dep_image in "${base_dependencies[@]}"; do
        # 使用映射配置生成目标镜像名
        local target_image
        target_image=$(get_mapped_private_image "$dep_image" "$registry" "$tag")
        
        print_info "→ 推送依赖: $dep_image"
        print_info "  目标镜像: $target_image"
        
        # 检查本地镜像是否存在
        if ! docker image inspect "$dep_image" >/dev/null 2>&1; then
            print_warning "  ! 本地镜像不存在，尝试拉取: $dep_image"
            if ! docker pull "$dep_image"; then
                print_error "  ✗ 拉取失败: $dep_image"
                failed_images+=("$dep_image")
                total_failed=$((total_failed + 1))
                continue
            fi
        fi
        
        # 标记镜像
        if docker tag "$dep_image" "$target_image"; then
            print_success "  ✓ 标记成功"
        else
            print_error "  ✗ 标记失败: $target_image"
            failed_images+=("$dep_image")
            total_failed=$((total_failed + 1))
            continue
        fi
        
        # 推送镜像
        if docker push "$target_image"; then
            print_success "  ✓ 推送成功: $target_image"
            total_pushed=$((total_pushed + 1))
        else
            print_error "  ✗ 推送失败: $target_image"
            failed_images+=("$target_image")
            total_failed=$((total_failed + 1))
        fi
        echo
    done
    
    # 输出统计信息
    print_info "=========================================="
    print_success "推送完成统计:"
    print_success "  • 成功推送: $total_pushed 个镜像"
    if [[ $total_failed -gt 0 ]]; then
        print_error "  • 失败推送: $total_failed 个镜像"
        print_warning "失败的镜像:"
        for failed_image in "${failed_images[@]}"; do
            echo "    - $failed_image"
        done
        return 1
    else
        print_success "🎉 所有镜像推送成功！"
        print_info ""
        print_info "内部仓库已准备就绪，现在可以在离线环境："
        print_info "1. 拉取镜像: ./build.sh harbor-pull-all $registry $tag"
        print_info "2. 启动服务: docker compose -f docker-compose.yml.example up -d"
        return 0
    fi
}

# 准备离线部署包（导出镜像 + 推送到内部仓库）
prepare_offline_deployment() {
    # 处理帮助参数
    if [[ "$1" == "--help" || "$1" == "-h" ]]; then
        echo "prepare-offline - 准备完整离线部署包"
        echo
        echo "用法: $0 prepare-offline <registry> [tag] [output_dir] [include_kafka]"
        echo
        echo "参数:"
        echo "  registry      内部仓库地址 (必需)"
        echo "  tag          镜像标签 (默认: $DEFAULT_IMAGE_TAG)"
        echo "  output_dir    输出目录 (默认: ./offline-deployment)"
        echo "  include_kafka 是否包含Kafka镜像 (默认: true)"
        echo
        echo "说明:"
        echo "  完整的离线部署包准备，包括:"
        echo "  • 导出离线镜像文件到本地"
        echo "  • 推送镜像到内部仓库"
        echo "  • 生成部署配置文件"
        echo "  • 创建自动部署脚本和文档"
        echo
        echo "示例:"
        echo "  $0 prepare-offline harbor.company.com/ai-infra v1.0.0 ./offline true"
        echo "  $0 prepare-offline registry.internal.com/project v0.3.6-dev ./deploy false"
        return 0
    fi
    
    local registry="$1"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    local output_dir="${3:-./offline-deployment}"
    local include_kafka="${4:-true}"
    
    if [[ -z "$registry" ]]; then
        print_error "请指定内部仓库地址"
        print_info "用法: prepare-offline <registry> [tag] [output_dir] [include_kafka]"
        print_info "示例: prepare-offline harbor.company.com/ai-infra v1.0.0 ./offline true"
        return 1
    fi
    
    print_info "=========================================="
    print_info "准备离线部署包"
    print_info "=========================================="
    print_info "内部仓库: $registry"
    print_info "镜像标签: $tag"
    print_info "输出目录: $output_dir"
    print_info "包含Kafka: $include_kafka"
    echo
    
    local overall_success=true
    
    # 步骤1: 导出离线镜像
    print_info "步骤 1/3: 导出离线镜像..."
    local images_dir="${output_dir}/images"
    if ! export_offline_images "$images_dir" "$tag" "$include_kafka"; then
        print_error "离线镜像导出失败"
        overall_success=false
    fi
    
    echo
    # 步骤2: 推送到内部仓库
    print_info "步骤 2/3: 推送镜像到内部仓库..."
    if ! push_to_internal_registry "$registry" "$tag" "$include_kafka"; then
        print_error "镜像推送到内部仓库失败"
        overall_success=false
    fi
    
    echo
    # 步骤3: 生成部署配置
    print_info "步骤 3/3: 生成部署配置..."
    mkdir -p "$output_dir"
    
    # 复制部署文件
    if [[ -f "docker-compose.yml.example" ]]; then
        cp "docker-compose.yml.example" "${output_dir}/docker-compose.yml.example"
        print_success "  ✓ 复制 docker-compose.yml.example"
    fi
    
    if [[ -f ".env.example" ]]; then
        cp ".env.example" "${output_dir}/.env.example"
        print_success "  ✓ 复制 .env.example"
    fi
    
    if [[ -f "build.sh" ]]; then
        cp "build.sh" "${output_dir}/build.sh"
        chmod +x "${output_dir}/build.sh"
        print_success "  ✓ 复制 build.sh"
    fi
    
    # 复制配置目录
    if [[ -d "config" ]]; then
        cp -r "config" "${output_dir}/"
        print_success "  ✓ 复制配置目录"
    fi
    
    # 生成离线部署脚本
    local deploy_script="${output_dir}/deploy-offline.sh"
    cat > "$deploy_script" << EOF
#!/bin/bash

# AI Infrastructure Matrix 离线部署脚本
# 使用方法: ./deploy-offline.sh [registry] [tag]

set -e

SCRIPT_DIR="\$(cd "\$(dirname "\${BASH_SOURCE[0]}")" && pwd)"
REGISTRY="${registry}"
TAG="${tag}"
INCLUDE_KAFKA="${include_kafka}"

print_info() {
    echo -e "\033[32m[INFO]\033[0m \$1"
}

print_success() {
    echo -e "\033[32m[SUCCESS]\033[0m \$1"
}

print_error() {
    echo -e "\033[31m[ERROR]\033[0m \$1"
}

print_info "=========================================="
print_info "AI Infrastructure Matrix 离线部署"
print_info "=========================================="
print_info "内部仓库: \${1:-\$REGISTRY}"
print_info "镜像标签: \${2:-\$TAG}"
print_info "包含Kafka: \$INCLUDE_KAFKA"
echo

FINAL_REGISTRY="\${1:-\$REGISTRY}"
FINAL_TAG="\${2:-\$TAG}"

# 检查Docker环境
if ! command -v docker >/dev/null 2>&1; then
    print_error "Docker未安装或不可用"
    exit 1
fi

if ! command -v docker-compose >/dev/null 2>&1 && ! docker compose version >/dev/null 2>&1; then
    print_error "Docker Compose未安装或不可用"
    exit 1
fi

# 选择部署方式
echo "请选择部署方式："
echo "1) 从内部仓库拉取镜像 (推荐)"
echo "2) 从本地tar文件导入镜像"
echo

read -p "请输入选择 (1-2): " deploy_mode

case "\$deploy_mode" in
    "1")
        print_info "从内部仓库拉取镜像..."
        if [[ -z "\$FINAL_REGISTRY" ]]; then
            print_error "请指定内部仓库地址"
            print_info "用法: ./deploy-offline.sh <registry> [tag]"
            exit 1
        fi
        
        # 使用build.sh拉取镜像
        if [[ -f "./build.sh" ]]; then
            print_info "拉取所有镜像..."
            if ./build.sh harbor-pull-all "\$FINAL_REGISTRY" "\$FINAL_TAG"; then
                print_success "✓ 镜像拉取成功"
            else
                print_error "镜像拉取失败"
                exit 1
            fi
        else
            print_error "build.sh文件不存在"
            exit 1
        fi
        ;;
        
    "2")
        print_info "从本地tar文件导入镜像..."
        if [[ -f "./images/import-images.sh" ]]; then
            cd images && ./import-images.sh
            cd ..
            print_success "✓ 镜像导入成功"
        else
            print_error "镜像导入脚本不存在: ./images/import-images.sh"
            exit 1
        fi
        ;;
        
    *)
        print_error "无效选择"
        exit 1
        ;;
esac

# 生成环境配置
print_info "生成环境配置..."
if [[ ! -f ".env" ]]; then
    if [[ -f ".env.example" ]]; then
        cp ".env.example" ".env"
        print_success "✓ 创建环境配置文件"
    else
        print_error "环境模板文件不存在"
        exit 1
    fi
fi

# 启动服务
print_info "启动服务..."
if docker compose -f docker-compose.yml.example up -d; then
    print_success "✅ 服务启动成功！"
    echo
    print_info "访问地址:"
    print_info "  • 主页: http://localhost:8080"
    print_info "  • JupyterHub: http://localhost:8088/jupyter/"
    print_info "  • Gitea: http://localhost:3010/gitea/"
    if [[ "\$INCLUDE_KAFKA" == "true" ]]; then
        print_info "  • Kafka UI: http://localhost:9095"
    fi
    echo
    print_info "管理命令:"
    print_info "  • 查看状态: docker compose ps"
    print_info "  • 查看日志: docker compose logs -f [service]"
    print_info "  • 停止服务: docker compose down"
else
    print_error "服务启动失败"
    exit 1
fi
EOF
    
    chmod +x "$deploy_script"
    print_success "  ✓ 生成离线部署脚本: $deploy_script"
    
    # 生成README文档
    local readme_file="${output_dir}/README.md"
    cat > "$readme_file" << EOF
# AI Infrastructure Matrix 离线部署包

## 概述

此离线部署包包含了 AI Infrastructure Matrix 在离线环境中运行所需的所有组件。

## 目录结构

\`\`\`
offline-deployment/
├── images/                    # 离线镜像文件
│   ├── *.tar                 # 镜像tar文件
│   ├── images-manifest.txt   # 镜像清单
│   └── import-images.sh      # 镜像导入脚本
├── config/                   # 配置文件目录
├── docker-compose.yml.example # Docker Compose配置
├── .env.example             # 环境变量模板
├── build.sh                 # 构建管理脚本
├── deploy-offline.sh        # 离线部署脚本
└── README.md               # 本文档
\`\`\`

## 部署信息

- **内部仓库**: \`${registry}\`
- **镜像标签**: \`${tag}\`
- **包含Kafka**: \`${include_kafka}\`
- **生成时间**: \`$(date)\`

## 快速部署

### 方式1: 使用自动部署脚本（推荐）

\`\`\`bash
chmod +x deploy-offline.sh
./deploy-offline.sh
\`\`\`

### 方式2: 手动部署

#### 从内部仓库拉取镜像

\`\`\`bash
# 1. 拉取所有镜像
./build.sh harbor-pull-all ${registry} ${tag}

# 2. 生成环境配置
cp .env.example .env

# 3. 启动服务
docker compose -f docker-compose.yml.example up -d
\`\`\`

#### 从本地镜像文件导入

\`\`\`bash
# 1. 导入镜像
cd images && ./import-images.sh && cd ..

# 2. 生成环境配置
cp .env.example .env

# 3. 启动服务
docker compose -f docker-compose.yml.example up -d
\`\`\`

## 访问地址

部署成功后，可以通过以下地址访问：

- **主页**: http://localhost:8080
- **JupyterHub**: http://localhost:8088/jupyter/
- **Gitea**: http://localhost:3010/gitea/
EOF

    if [[ "$include_kafka" == "true" ]]; then
        echo "- **Kafka UI**: http://localhost:9095" >> "$readme_file"
    fi

    cat >> "$readme_file" << EOF

## 管理命令

\`\`\`bash
# 查看服务状态
docker compose ps

# 查看服务日志
docker compose logs -f [service]

# 停止所有服务
docker compose down

# 重启服务
docker compose restart [service]
\`\`\`

## 故障排除

### 常见问题

1. **端口冲突**: 如果遇到端口冲突，修改 \`.env\` 文件中的端口配置
2. **镜像拉取失败**: 检查内部仓库连接和权限
3. **服务启动失败**: 查看具体服务日志 \`docker compose logs [service]\`

### 获取帮助

查看更多管理命令：
\`\`\`bash
./build.sh help
\`\`\`

## 技术支持

如需技术支持，请参考项目文档或联系管理员。
EOF
    
    print_success "  ✓ 生成README文档: $readme_file"
    
    # 最终汇总
    echo
    print_info "=========================================="
    if [[ "$overall_success" == "true" ]]; then
        print_success "🎉 离线部署包准备完成！"
        print_info ""
        print_info "📁 输出目录: $output_dir"
        print_info "📊 包含内容:"
        print_info "  • 离线镜像文件: $(ls "${images_dir}"/*.tar 2>/dev/null | wc -l) 个"
        print_info "  • 部署配置文件"
        print_info "  • 自动部署脚本"
        print_info "  • 详细文档"
        print_info ""
        print_info "📋 使用方法:"
        print_info "1. 将整个 $output_dir 目录复制到离线环境"
        print_info "2. 在离线环境运行: cd $output_dir && ./deploy-offline.sh"
        print_info ""
        print_info "🌐 内部仓库镜像已推送至: $registry"
        return 0
    else
        print_warning "⚠️  离线部署包准备部分完成"
        print_info "请检查上述错误信息并重新运行失败的步骤"
        return 1
    fi
}

# ====================================================
# 自动化补丁管理系统
# ====================================================

# 应用代码补丁
apply_patch() {
    local patch_name="${1:-}"
    local target_service="${2:-}"
    local rebuild="${3:-true}"
    
    if [[ -z "$patch_name" ]]; then
        print_error "请指定要应用的补丁名称"
        list_available_patches
        return 1
    fi
    
    print_info "=========================================="
    print_info "应用代码补丁: $patch_name"
    print_info "=========================================="
    
    case "$patch_name" in
        "ldap-fix"|"ldap-field-fix")
            apply_ldap_field_fix "$rebuild"
            ;;
        "cors-fix")
            apply_cors_fix "$rebuild"
            ;;
        "frontend-build-fix")
            apply_frontend_build_fix "$rebuild"
            ;;
        "backend-auth-fix")
            apply_backend_auth_fix "$rebuild"
            ;;
        "custom")
            if [[ -z "$target_service" ]]; then
                print_error "自定义补丁需要指定目标服务"
                return 1
            fi
            apply_custom_patch "$target_service" "$rebuild"
            ;;
        *)
            print_error "未知的补丁: $patch_name"
            list_available_patches
            return 1
            ;;
    esac
}

# 列出可用的补丁
list_available_patches() {
    print_info "可用的补丁:"
    echo "  • ldap-fix          - 修复LDAP字段映射问题"
    echo "  • cors-fix          - 修复CORS跨域问题"
    echo "  • frontend-build-fix - 修复前端构建问题"
    echo "  • backend-auth-fix  - 修复后端认证问题"
    echo "  • custom            - 应用自定义补丁 (需要指定服务)"
    echo
    echo "用法: $0 patch <patch-name> [service] [rebuild=true|false]"
    echo "示例:"
    echo "  $0 patch ldap-fix                    # 应用LDAP修复并重建"
    echo "  $0 patch ldap-fix \"\" false           # 应用LDAP修复但不重建"
    echo "  $0 patch custom backend              # 应用自定义后端补丁"
}

# LDAP字段修复补丁
apply_ldap_field_fix() {
    local rebuild="${1:-true}"
    
    print_info "应用LDAP字段映射修复补丁..."
    
    local models_file="$SCRIPT_DIR/src/backend/internal/models/models.go"
    local ldap_file="$SCRIPT_DIR/src/backend/internal/services/ldap.go"
    
    # 检查文件是否存在
    if [[ ! -f "$models_file" ]]; then
        print_error "找不到models文件: $models_file"
        return 1
    fi
    
    if [[ ! -f "$ldap_file" ]]; then
        print_error "找不到LDAP服务文件: $ldap_file"
        return 1
    fi
    
    print_info "步骤1: 备份原始文件..."
    cp "$models_file" "${models_file}.backup.$(date +%Y%m%d_%H%M%S)"
    cp "$ldap_file" "${ldap_file}.backup.$(date +%Y%m%d_%H%M%S)"
    
    print_info "步骤2: 修复LDAPTestRequest结构体..."
    # 修复models.go中的LDAPTestRequest结构体
    if grep -q "type LDAPTestRequest struct" "$models_file"; then
        # 使用临时文件进行替换
        local temp_file=$(mktemp)
        cat > "$temp_file" << 'MODELS_PATCH_EOF'
type LDAPTestRequest struct {
	Server         string `json:"server" validate:"required"`
	Port           int    `json:"port" validate:"required,min=1,max=65535"`
	BindDN         string `json:"bind_dn" validate:"required"`
	BindPassword   string `json:"bind_password" validate:"required"`
	BaseDN         string `json:"base_dn" validate:"required"`
	UserFilter     string `json:"user_filter"`
	// 支持前端的字段名
	EnableTLS      bool   `json:"enable_tls"`
	SkipTLSVerify  bool   `json:"skip_tls_verify"`
	// 兼容后端原有字段名
	UseSSL         bool   `json:"use_ssl"`
	SkipVerify     bool   `json:"skip_verify"`
}
MODELS_PATCH_EOF
        
        # 替换结构体定义
        awk '
        /^type LDAPTestRequest struct/ {
            # 输出新的结构体定义
            while ((getline line < "'$temp_file'") > 0) {
                print line
            }
            close("'$temp_file'")
            # 跳过原有的结构体定义直到找到下一个类型定义或空行
            while (getline && !/^type|^$|^\/\/|^func/) {
                continue
            }
            if ($0 ~ /^type|^func/) {
                print $0
            }
            next
        }
        { print }
        ' "$models_file" > "${models_file}.tmp" && mv "${models_file}.tmp" "$models_file"
        
        rm -f "$temp_file"
        print_success "✓ LDAPTestRequest结构体已更新"
    else
        print_warning "未找到LDAPTestRequest结构体定义"
    fi
    
    print_info "步骤3: 修复LDAP服务连接逻辑..."
    # 修复ldap.go中的TestLDAPConnection函数
    if grep -q "func.*TestLDAPConnection" "$ldap_file"; then
        # 创建临时补丁文件
        local temp_patch=$(mktemp)
        cat > "$temp_patch" << 'LDAP_PATCH_EOF'
	// 兼容前端字段名映射
	if req.EnableTLS && !req.UseSSL {
		req.UseSSL = req.EnableTLS
	}
	if req.SkipTLSVerify && !req.SkipVerify {
		req.SkipVerify = req.SkipTLSVerify
	}
LDAP_PATCH_EOF
        
        # 在TestLDAPConnection函数开始后插入映射逻辑
        awk -v patch_file="$temp_patch" '
        /func.*TestLDAPConnection.*{/ {
            print $0
            # 读取下一行
            if (getline > 0) {
                print $0
                # 插入补丁内容
                while ((getline line < patch_file) > 0) {
                    print line
                }
                close(patch_file)
            }
            next
        }
        { print }
        ' "$ldap_file" > "${ldap_file}.tmp" && mv "${ldap_file}.tmp" "$ldap_file"
        
        rm -f "$temp_patch"
        print_success "✓ LDAP连接逻辑已更新"
    else
        print_warning "未找到TestLDAPConnection函数"
    fi
    
    print_info "步骤4: 验证代码语法..."
    if command -v go >/dev/null 2>&1; then
        cd "$SCRIPT_DIR/src/backend" && go mod tidy >/dev/null 2>&1
        if go build -o /tmp/backend_test ./cmd/server >/dev/null 2>&1; then
            print_success "✓ 代码语法检查通过"
            rm -f /tmp/backend_test
        else
            print_error "代码语法检查失败，可能需要手动调整"
        fi
        cd "$SCRIPT_DIR"
    else
        print_warning "未安装Go，跳过语法检查"
    fi
    
    if [[ "$rebuild" == "true" ]]; then
        print_info "步骤5: 重建后端服务..."
        if rebuild_service "backend" "true"; then
            print_success "✓ 后端服务重建完成"
            
            print_info "步骤6: 重启后端服务..."
            if docker compose restart backend >/dev/null 2>&1; then
                print_success "✓ 后端服务重启完成"
                
                # 等待服务启动
                sleep 3
                
                # 检查服务状态
                if check_service_health "backend"; then
                    print_success "✓ LDAP修复补丁应用成功！"
                    print_info "现在可以测试LDAP连接功能"
                else
                    print_error "后端服务启动异常，请检查日志"
                    return 1
                fi
            else
                print_error "后端服务重启失败"
                return 1
            fi
        else
            print_error "后端服务重建失败"
            return 1
        fi
    else
        print_success "✓ LDAP修复补丁应用完成（未重建服务）"
        print_info "请手动重建并重启服务: $0 build backend && docker compose restart backend"
    fi
}

# 重建指定服务
rebuild_service() {
    local service="$1"
    local force_no_cache="${2:-false}"
    
    print_info "重建服务: $service"
    
    if [[ "$force_no_cache" == "true" ]]; then
        # 强制重建，不使用缓存
        if build_service "$service" "$DEFAULT_IMAGE_TAG" "" "--no-cache"; then
            return 0
        else
            return 1
        fi
    else
        # 正常重建
        if build_service "$service" "$DEFAULT_IMAGE_TAG"; then
            return 0
        else
            return 1
        fi
    fi
}

# 检查服务健康状态
check_service_health() {
    local service="$1"
    local timeout=30
    local count=0
    
    print_info "检查服务健康状态: $service"
    
    while [[ $count -lt $timeout ]]; do
        if docker compose ps --filter "status=running" 2>/dev/null | grep -q "$service"; then
            case "$service" in
                "backend")
                    # 检查后端API - 通过nginx代理
                    if curl -s -f --connect-timeout 5 "http://localhost:8080/api/health" >/dev/null 2>&1; then
                        return 0
                    fi
                    # 备用：直接检查后端端口（如果nginx未启动）
                    if curl -s -f --connect-timeout 5 "http://localhost:8082/api/health" >/dev/null 2>&1; then
                        return 0
                    fi
                    ;;
                "frontend")
                    # 检查前端 - 通过nginx代理或环境变量指定的端口
                    local frontend_port="${EXTERNAL_PORT:-8080}"
                    if curl -s -f --connect-timeout 5 "http://localhost:$frontend_port" >/dev/null 2>&1; then
                        return 0
                    fi
                    # 备用：检查容器内的80端口（如果直接访问容器）
                    if [ "$frontend_port" != "80" ] && curl -s -f --connect-timeout 5 "http://localhost:80" >/dev/null 2>&1; then
                        return 0
                    fi
                    ;;
                "nginx")
                    # 检查nginx主页
                    if curl -s -f --connect-timeout 5 "http://localhost:8080" >/dev/null 2>&1; then
                        return 0
                    fi
                    ;;
                *)
                    # 其他服务只检查容器状态
                    print_success "✓ $service 容器运行中"
                    return 0
                    ;;
            esac
        fi
        
        sleep 1
        count=$((count + 1))
    done
    
    print_error "服务健康检查失败: $service"
    print_info "容器状态:"
    docker compose ps "$service" 2>/dev/null || true
    print_info "最近的日志:"
    docker compose logs --tail=5 "$service" 2>/dev/null || true
    return 1
}

# CORS修复补丁
apply_cors_fix() {
    local rebuild="${1:-true}"
    
    print_info "应用CORS跨域修复补丁..."
    print_warning "CORS修复补丁尚未实现"
    # TODO: 实现CORS修复逻辑
}

# 前端构建修复补丁
apply_frontend_build_fix() {
    local rebuild="${1:-true}"
    
    print_info "应用前端构建修复补丁..."
    print_warning "前端构建修复补丁尚未实现" 
    # TODO: 实现前端构建修复逻辑
}

# 后端认证修复补丁
apply_backend_auth_fix() {
    local rebuild="${1:-true}"
    
    print_info "应用后端认证修复补丁..."
    print_warning "后端认证修复补丁尚未实现"
    # TODO: 实现后端认证修复逻辑
}

# 应用自定义补丁
apply_custom_patch() {
    local service="$1"
    local rebuild="${2:-true}"
    
    print_info "应用自定义补丁到服务: $service"
    
    local patch_dir="$SCRIPT_DIR/patches"
    local patch_file="$patch_dir/${service}.patch"
    
    if [[ ! -f "$patch_file" ]]; then
        print_error "找不到补丁文件: $patch_file"
        print_info "请在 $patch_dir 目录下创建 ${service}.patch 文件"
        return 1
    fi
    
    print_info "应用补丁文件: $patch_file"
    
    # 应用git patch
    if patch -p1 < "$patch_file" 2>/dev/null; then
        print_success "✓ 补丁应用成功"
        
        if [[ "$rebuild" == "true" ]]; then
            print_info "重建服务: $service"
            if rebuild_service "$service" "true"; then
                print_success "✓ 服务重建完成"
            else
                print_error "服务重建失败"
                return 1
            fi
        fi
    else
        print_error "补丁应用失败，请检查补丁文件格式"
        return 1
    fi
}

# 生成补丁文件
generate_patch() {
    local service="${1:-}"
    local output_file="${2:-}"
    
    if [[ -z "$service" ]]; then
        print_error "请指定要生成补丁的服务"
        print_info "可用服务: backend frontend nginx jupyterhub"
        return 1
    fi
    
    if [[ -z "$output_file" ]]; then
        output_file="$SCRIPT_DIR/patches/${service}_$(date +%Y%m%d_%H%M%S).patch"
    fi
    
    print_info "生成服务补丁: $service"
    
    # 创建patches目录
    mkdir -p "$SCRIPT_DIR/patches"
    
    local service_dir="$SCRIPT_DIR/src/$service"
    
    if [[ ! -d "$service_dir" ]]; then
        print_error "服务目录不存在: $service_dir"
        return 1
    fi
    
    # 生成git diff补丁
    cd "$SCRIPT_DIR"
    if git diff --no-index /dev/null "$service_dir" > "$output_file" 2>/dev/null; then
        print_success "✓ 补丁文件已生成: $output_file"
    else
        # 尝试生成基于当前变更的补丁
        if git diff HEAD -- "src/$service" > "$output_file" 2>/dev/null; then
            print_success "✓ 基于git变更的补丁文件已生成: $output_file"
        else
            print_error "补丁生成失败"
            return 1
        fi
    fi
    
    print_info "补丁文件大小: $(wc -l < "$output_file") 行"
}

# ====================================================
# CI/CD构建和生产环境启动函数
# ====================================================

# CI/CD完整构建流程 - 适用于有外网访问的构建环境
ci_build_complete() {
    local registry="$1"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    local external_host="$3"
    
    if [[ -z "$registry" ]]; then
        print_error "必须指定目标镜像仓库地址"
        return 1
    fi
    
    print_info "=========================================="
    print_info "CI/CD完整构建流程开始"
    print_info "目标仓库: $registry"
    print_info "镜像标签: $tag"
    print_info "=========================================="
    
    # 检测网络环境
    local network_env=$(detect_network_environment)
    if [[ "$network_env" == "internal" ]]; then
        print_warning "检测到内网环境，此命令适用于外网环境"
        print_info "如果确认有外网访问，请继续；否则请使用 prod-start 命令"
        read -p "是否继续? (y/N): " continue_build
        if [[ "$continue_build" != "y" && "$continue_build" != "Y" ]]; then
            print_info "构建已取消"
            return 0
        fi
    fi
    
    # 步骤1: 检测和设置外部主机地址
    if [[ -n "$external_host" ]]; then
        print_info "步骤1: 使用指定的外部主机地址: $external_host"
    else
        print_info "步骤1: 自动检测外部主机地址..."
        if [[ -f "$SCRIPT_DIR/scripts/detect-external-host.sh" ]]; then
            external_host=$(cd "$SCRIPT_DIR" && bash scripts/detect-external-host.sh | grep "检测到的主机地址:" | cut -d: -f2 | xargs)
            if [[ -n "$external_host" && "$external_host" != "localhost" ]]; then
                print_success "自动检测到外部主机: $external_host"
            else
                external_host="localhost"
                print_warning "未检测到外部主机，使用默认地址: $external_host"
            fi
        else
            external_host="localhost"
            print_warning "检测脚本不存在，使用默认地址: $external_host"
        fi
    fi
    
    # 步骤2: 生成配置模板
    print_info "步骤2: 生成配置模板..."
    if ! render_env_template "$external_host" "8080" "http"; then
        print_error "配置模板生成失败"
        return 1
    fi
    
    if ! render_nginx_templates; then
        print_error "Nginx模板渲染失败"
        return 1
    fi
    
    if ! render_jupyterhub_templates; then
        print_error "JupyterHub模板渲染失败"
        return 1
    fi
    
    # 步骤3: 拉取并重新标记依赖镜像
    print_info "步骤3: 拉取并重新标记依赖镜像..."
    if ! pull_and_tag_dependencies "$registry" "$tag"; then
        print_error "依赖镜像处理失败"
        return 1
    fi
    
    # 步骤4: 构建所有服务镜像
    print_info "步骤4: 构建所有服务镜像..."
    if ! build_all_services "$tag" "$registry"; then
        print_error "服务镜像构建失败"
        return 1
    fi
    
    # 步骤5: 推送所有镜像到仓库
    print_info "步骤5: 推送所有镜像到仓库..."
    if ! push_all_services "$tag" "$registry"; then
        print_error "服务镜像推送失败"
        return 1
    fi
    
    if ! push_dependencies "$registry" "$tag"; then
        print_error "依赖镜像推送失败"
        return 1
    fi
    
    # 步骤6: 生成生产环境配置文件
    print_info "步骤6: 生成生产环境配置文件..."
    if ! render_docker_compose_templates "$registry" "$tag"; then
        print_error "Docker Compose配置生成失败"
        return 1
    fi
    
    # 步骤7: 生成生产环境变量文件
    print_info "步骤7: 生成生产环境变量文件..."
    if ! create_production_env "production" "$registry" "$tag"; then
        print_error "生产环境变量文件生成失败"
        return 1
    fi
    
    print_success "=========================================="
    print_success "CI/CD构建流程完成！"
    print_success "=========================================="
    print_info "镜像仓库: $registry"
    print_info "镜像标签: $tag"
    print_info "外部访问: http://$external_host:8080"
    print_info ""
    print_info "生成的文件:"
    print_info "• docker-compose.yml - 生产环境服务配置"
    print_info "• .env.prod - 生产环境变量"
    print_info "• src/nginx/conf.d/ - Nginx配置文件"
    print_info "• src/jupyterhub/ - JupyterHub配置文件"
    print_info ""
    print_info "下一步: 将这些文件部署到生产环境，并运行："
    print_info "  $0 prod-start $registry $tag $external_host"
    
    return 0
}

# 生产环境服务启动 - 适用于无外网访问的生产环境
prod_start_complete() {
    local registry="$1"  # 可选，如果为空则使用本地镜像
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    local external_host="$3"
    local external_port="${4:-8080}"
    
    # 检测 docker compose 命令（优先 v2: docker compose，其次 v1: docker-compose）
    local COMPOSE_BIN=""
    if docker compose version >/dev/null 2>&1; then
        COMPOSE_BIN="docker compose"
    elif command -v docker-compose >/dev/null 2>&1; then
        COMPOSE_BIN="docker-compose"
    else
        print_error "未检测到 docker compose 或 docker-compose 命令"
        return 1
    fi
    
    print_info "=========================================="
    print_info "生产环境服务启动流程开始"
    if [[ -n "$registry" ]]; then
        print_info "镜像仓库: $registry"
    else
        print_info "使用本地镜像"
    fi
    print_info "镜像标签: $tag"
    print_info "外部端口: $external_port"
    print_info "=========================================="
    
    # 步骤1: 检测和设置外部主机地址
    if [[ -n "$external_host" ]]; then
        print_info "步骤1: 使用指定的外部主机地址: $external_host"
    else
        print_info "步骤1: 自动检测外部主机地址..."
        if [[ -f "$SCRIPT_DIR/scripts/detect-external-host.sh" ]]; then
            external_host=$(cd "$SCRIPT_DIR" && bash scripts/detect-external-host.sh | grep "检测到的主机地址:" | cut -d: -f2 | xargs)
            if [[ -n "$external_host" && "$external_host" != "localhost" ]]; then
                print_success "自动检测到外部主机: $external_host"
            else
                external_host="localhost"
                print_warning "未检测到外部主机，使用默认地址: $external_host"
            fi
        else
            external_host="localhost"
            print_warning "检测脚本不存在，使用默认地址: $external_host"
        fi
    fi
    
    # 步骤2: 从内部仓库拉取镜像（如果指定了registry）
    if [[ -n "$registry" ]]; then
        print_info "步骤2: 从内部仓库拉取镜像..."
        
        # 拉取服务镜像
        if ! pull_aiharbor_services "$registry" "$tag"; then
            print_warning "从内部仓库拉取服务镜像失败，尝试使用本地镜像"
        else
            print_success "服务镜像拉取完成"
        fi
        
        # 拉取依赖镜像
        if ! pull_aiharbor_dependencies "$registry" "$tag"; then
            print_warning "从内部仓库拉取依赖镜像失败，尝试使用本地镜像"
        else
            print_success "依赖镜像拉取完成"
        fi
    else
        print_info "步骤2: 跳过镜像拉取，使用本地镜像"
    fi
    
    # 步骤3: 生成配置模板
    print_info "步骤3: 生成生产环境配置..."
    if ! render_env_template "$external_host" "$external_port" "http"; then
        print_error "环境配置生成失败"
        return 1
    fi
    
    if ! render_nginx_templates; then
        print_error "Nginx配置生成失败"
        return 1
    fi
    
    if ! render_jupyterhub_templates; then
        print_error "JupyterHub配置生成失败"
        return 1
    fi
    
    # 步骤4: 生成Docker Compose配置
    print_info "步骤4: 生成Docker Compose配置..."
    if [[ -n "$registry" ]]; then
        if ! render_docker_compose_templates "$registry" "$tag"; then
            print_error "Docker Compose配置生成失败"
            return 1
        fi
    else
        if ! render_docker_compose_templates "" "$tag"; then
            print_error "Docker Compose配置生成失败"  
            return 1
        fi
    fi
    
    # 步骤5: 停止现有服务（如果正在运行）
    print_info "步骤5: 停止现有服务..."
    if $COMPOSE_BIN ps --services --filter "status=running" 2>/dev/null | grep -q .; then
        print_info "发现正在运行的服务，正在停止..."
        $COMPOSE_BIN down --remove-orphans >/dev/null 2>&1
        print_success "现有服务已停止"
    else
        print_info "没有正在运行的服务"
    fi
    
    # 步骤6: 启动所有服务
    print_info "步骤6: 启动所有服务..."
    if ! $COMPOSE_BIN up -d; then
        print_error "服务启动失败"
        return 1
    fi
    
    # 等待服务启动
    print_info "等待服务启动..."
    sleep 5
    
    # 步骤7: 检查服务状态
    print_info "步骤7: 检查服务状态..."
    local failed_services=()
    local total_services=0
    local running_services=0
    
    while IFS= read -r service; do
        if [[ -n "$service" ]]; then
            total_services=$((total_services + 1))
            local status=$($COMPOSE_BIN ps --services --filter "status=running" 2>/dev/null | grep "^${service}$" || echo "")
            if [[ -n "$status" ]]; then
                running_services=$((running_services + 1))
                print_success "✓ $service"
            else
                failed_services+=("$service")
                print_error "✗ $service"
            fi
        fi
    done < <($COMPOSE_BIN ps --services 2>/dev/null)
    
    # 步骤8: 显示结果
    print_info "=========================================="
    if [[ ${#failed_services[@]} -eq 0 ]]; then
        print_success "所有服务启动成功！($running_services/$total_services)"
        print_success "=========================================="
        print_info "系统访问地址: http://$external_host:$external_port"
        print_info "默认管理员: admin/admin123"
        print_info ""
        print_info "服务检查命令:"
        print_info "• 查看服务状态: $COMPOSE_BIN ps"
        print_info "• 查看服务日志: $COMPOSE_BIN logs [服务名]"
        print_info "• 停止所有服务: $COMPOSE_BIN down"
        print_info "• 重启所有服务: $COMPOSE_BIN restart"
    else
        print_warning "部分服务启动失败 ($running_services/$total_services)"
        print_warning "失败的服务: ${failed_services[*]}"
        print_info "=========================================="
        print_info "请检查失败服务的日志:"
        for service in "${failed_services[@]}"; do
            print_info "• $COMPOSE_BIN logs $service"
        done
        return 1
    fi
    
    return 0
}

# ====================================================
# 统一构建和部署函数 - 公共参数接口
# ====================================================

# 统一构建所有镜像
# 用法: build_all_unified <registry> <tag> <external_host> <external_port> <external_scheme>
build_all_unified() {
    local registry="${1:-aiharbor.msxf.local/aihpc}"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    local external_host="${3:-172.20.10.11}"
    local external_port="${4:-80}"
    local external_scheme="${5:-http}"
    
    print_info "开始统一构建所有镜像..."
    print_info "Registry: $registry"
    print_info "Tag: $tag"
    print_info "External Host: $external_host"
    print_info "External Port: $external_port"
    print_info "External Scheme: $external_scheme"
    
    # 渲染环境模板
    print_info "渲染环境配置模板..."
    if ! render_env_template "$external_host" "$external_port" "$external_scheme"; then
        print_error "环境模板渲染失败"
        return 1
    fi
    
    # 构建所有服务镜像
    print_info "构建所有服务镜像..."
    if ! build_all_services "$tag" "$registry"; then
        print_error "服务镜像构建失败"
        return 1
    fi
    
    print_success "统一构建完成！"
    print_info "镜像已构建到: $registry"
    print_info "镜像标签: $tag"
    return 0
}

# 统一构建并推送所有镜像
# 用法: build_and_push_unified <registry> <tag> <external_host> <external_port> <external_scheme>
build_and_push_unified() {
    local registry="${1:-aiharbor.msxf.local/aihpc}"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    local external_host="${3:-172.20.10.11}"
    local external_port="${4:-80}"
    local external_scheme="${5:-http}"
    
    print_info "开始统一构建和推送所有镜像..."
    print_info "Registry: $registry"
    print_info "Tag: $tag"
    print_info "External Host: $external_host"
    print_info "External Port: $external_port"
    print_info "External Scheme: $external_scheme"
    
    # 渲染环境模板
    print_info "渲染环境配置模板..."
    if ! render_env_template "$external_host" "$external_port" "$external_scheme"; then
        print_error "环境模板渲染失败"
        return 1
    fi
    
    # 构建和推送所有镜像
    print_info "构建和推送所有镜像..."
    if ! build_and_push_all "$registry" "$tag"; then
        print_error "镜像构建和推送失败"
        return 1
    fi
    
    print_success "统一构建和推送完成！"
    print_info "镜像已推送到: $registry"
    print_info "镜像标签: $tag"
    return 0
}

# 统一部署服务
# 用法: deploy_unified <registry> <tag> <external_host> <external_port> <external_scheme> [compose_file]
deploy_unified() {
    local registry="${1:-aiharbor.msxf.local/aihpc}"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    local external_host="${3:-172.20.10.11}"
    local external_port="${4:-80}"
    local external_scheme="${5:-http}"
    local compose_file="${6:-docker-compose.yml}"
    
    print_info "开始统一部署服务..."
    print_info "Registry: $registry"
    print_info "Tag: $tag"
    print_info "External Host: $external_host"
    print_info "External Port: $external_port"
    print_info "External Scheme: $external_scheme"
    print_info "Compose File: $compose_file"
    
    # 渲染环境模板
    print_info "渲染环境配置模板..."
    if ! render_env_template "$external_host" "$external_port" "$external_scheme"; then
        print_error "环境模板渲染失败"
        return 1
    fi

    # 渲染Nginx配置模板
    print_info "渲染Nginx配置模板..."
    if ! render_nginx_templates; then
        print_warning "Nginx模板渲染失败，但流程继续"
    fi

    # 渲染JupyterHub配置模板
    print_info "渲染JupyterHub配置模板..."
    if ! render_jupyterhub_templates; then
        print_warning "JupyterHub模板渲染失败，但流程继续"
    fi

    # 渲染Docker Compose文件
    print_info "渲染Docker Compose配置..."
    if ! render_compose_template "$compose_file"; then
        print_error "Docker Compose模板渲染失败"
        return 1
    fi

    # 启动服务
    print_info "启动生产环境服务..."
    if ! start_production "$compose_file"; then
        print_error "服务启动失败"
        return 1
    fi

    print_success "统一部署完成！"
    print_info "服务已启动，访问地址: $external_scheme://$external_host:$external_port"
    return 0
}

# 一键构建和部署
# 用法: build_deploy_all <registry> <tag> <external_host> <external_port> <external_scheme> [compose_file]
build_deploy_all() {
    local registry="${1:-aiharbor.msxf.local/aihpc}"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    local external_host="${3:-172.20.10.11}"
    local external_port="${4:-80}"
    local external_scheme="${5:-http}"
    local compose_file="${6:-docker-compose.yml}"
    
    print_info "开始一键构建和部署流程..."
    print_info "Registry: $registry"
    print_info "Tag: $tag"
    print_info "External Host: $external_host"
    print_info "External Port: $external_port"
    print_info "External Scheme: $external_scheme"
    print_info "Compose File: $compose_file"
    
    # Step 0: 渲染所有模板
    print_info "=== 第0步: 渲染所有配置模板 ==="
    if ! render_nginx_templates; then
        print_warning "Nginx模板渲染失败，但流程继续"
    fi
    if ! render_jupyterhub_templates; then
        print_warning "JupyterHub模板渲染失败，但流程继续"
    fi
    if [[ -f "$SCRIPT_DIR/docker-compose.yml.example" ]]; then
        if ! render_docker_compose_templates "$registry" "$tag"; then
            print_warning "Docker Compose模板渲染失败，但流程继续"
        fi
    fi

    # Step 1: 构建并推送镜像
    print_info "=== 第1步: 构建并推送镜像 ==="
    if ! build_and_push_unified "$registry" "$tag" "$external_host" "$external_port" "$external_scheme"; then
        print_error "构建和推送阶段失败"
        return 1
    fi

    # Step 2: 部署服务
    print_info "=== 第2步: 部署服务 ==="
    if ! deploy_unified "$registry" "$tag" "$external_host" "$external_port" "$external_scheme" "$compose_file"; then
        print_error "部署阶段失败"
        return 1
    fi

    print_success "一键构建和部署完成！"
    print_info "所有服务已成功构建、推送并启动"
    print_info "访问地址: $external_scheme://$external_host:$external_port"
    return 0
}

# 环境模板渲染函数
render_env_template() {
    local external_host="$1"
    local external_port="$2"
    local external_scheme="$3"
    
    if [[ ! -f ".env.example" ]]; then
        print_error "环境模板文件 .env.example 不存在"
        return 1
    fi
    
    # 导出环境变量供envsubst使用
    export EXTERNAL_HOST="$external_host"
    export EXTERNAL_PORT="$external_port"
    export EXTERNAL_SCHEME="$external_scheme"
    
    # 使用envsubst渲染模板
    if command -v envsubst >/dev/null 2>&1; then
        print_info "使用 envsubst 渲染环境模板..."
        if envsubst < .env.example > .env.tmp && mv .env.tmp .env; then
            print_success "环境模板渲染成功"
            return 0
        else
            print_error "envsubst 渲染失败"
            rm -f .env.tmp
            return 1
        fi
    else
        # 回退到简单的sed替换
        print_info "使用 sed 渲染环境模板..."
        if sed -e "s/\${EXTERNAL_HOST}/$external_host/g" \
               -e "s/\${EXTERNAL_PORT}/$external_port/g" \
               -e "s/\${EXTERNAL_SCHEME}/$external_scheme/g" \
               .env.example > .env.tmp && mv .env.tmp .env; then
            print_success "环境模板渲染成功"
            return 0
        else
            print_error "sed 渲染失败"
            rm -f .env.tmp
            return 1
        fi
    fi
}

# Docker Compose模板渲染函数
render_compose_template() {
    local compose_file="$1"
    local template_file="${compose_file}.example"
    
    if [[ ! -f "$template_file" ]]; then
        print_warning "Docker Compose模板文件 $template_file 不存在，跳过渲染"
        return 0
    fi
    
    print_info "渲染 $template_file 到 $compose_file..."
    if cp "$template_file" "$compose_file"; then
        print_success "Docker Compose模板渲染成功"
        return 0
    else
        print_error "Docker Compose模板渲染失败"
        return 1
    fi
}

# 主函数
main() {
    # 预处理命令行参数，检查各种标志
    local args=()
    for arg in "$@"; do
        if [[ "$arg" == "--force" ]]; then
            FORCE_REBUILD=true
            print_info "启用强制重新构建模式"
        elif [[ "$arg" == "--skip-pull" ]]; then
            SKIP_PULL=true
            print_info "启用跳过拉取模式"
        elif [[ "$arg" == "--skip-cache-check" ]]; then
            SKIP_CACHE_CHECK=true
            print_info "启用跳过缓存检查模式"
        elif [[ "$arg" == "--china-mirror" ]]; then
            USE_CHINA_MIRROR=true
            print_info "启用中国镜像加速"
        elif [[ "$arg" == "--no-source-maps" ]]; then
            DISABLE_SOURCE_MAPS=true
            print_info "禁用源码映射生成"
        else
            args+=("$arg")
        fi
    done
    
    # 重新设置位置参数
    set -- "${args[@]}"
    
    # 动态更新版本标签（如果提供了版本参数）
    update_version_if_provided "$@"
    
    # 早期Docker Compose兼容性检查
    if [[ "${1:-}" != "version" && "${1:-}" != "help" && "${1:-}" != "-h" && "${1:-}" != "--help" ]]; then
        if ! check_compose_compatibility; then
            exit 1
        fi
    fi
    
    case "${1:-help}" in
        "list")
            list_services "${2:-$DEFAULT_IMAGE_TAG}" "$3"
            ;;
        
        "check-status")
            # 检查镜像构建状态（需求32）
            if [[ "${2:-}" == "--help" || "${2:-}" == "-h" ]]; then
                echo "check-status - 检查所有服务的镜像构建状态"
                echo
                echo "用法: $0 check-status [tag] [registry]"
                echo
                echo "参数:"
                echo "  tag         镜像标签 (默认: $DEFAULT_IMAGE_TAG)"
                echo "  registry    目标镜像仓库 (可选)"
                echo
                echo "说明:"
                echo "  检查所有服务的镜像构建状态，识别："
                echo "  • ✓ OK      - 镜像构建成功且有效"
                echo "  • ✗ MISSING - 镜像不存在"
                echo "  • ⚠ INVALID - 镜像存在但无效（大小为0或无标签）"
                echo
                echo "示例:"
                echo "  $0 check-status"
                echo "  $0 check-status v1.0.0"
                echo "  $0 check-status v1.0.0 harbor.company.com/ai-infra"
                return 0
            fi
            show_build_status "${2:-$DEFAULT_IMAGE_TAG}" "$3"
            ;;
        
        "cache-stats")
            # 显示构建缓存统计信息
            if [[ "${2:-}" == "--help" || "${2:-}" == "-h" ]]; then
                echo "cache-stats - 显示构建缓存统计信息"
                echo
                echo "用法: $0 cache-stats"
                echo
                echo "说明:"
                echo "  显示构建缓存的详细信息，包括："
                echo "  • 总构建次数"
                echo "  • 最近构建历史"
                echo "  • 各服务缓存状态"
                echo
                echo "示例:"
                echo "  $0 cache-stats"
                return 0
            fi
            show_build_cache_stats
            ;;
        
        "clean-cache")
            # 清理构建缓存
            if [[ "${2:-}" == "--help" || "${2:-}" == "-h" ]]; then
                echo "clean-cache - 清理构建缓存"
                echo
                echo "用法: $0 clean-cache [service]"
                echo
                echo "参数:"
                echo "  service     服务名称 (可选，不指定则清理所有)"
                echo
                echo "说明:"
                echo "  清理构建缓存数据，包括构建历史和哈希记录"
                echo "  清理后下次构建将重新计算哈希并构建"
                echo
                echo "示例:"
                echo "  $0 clean-cache              # 清理所有缓存"
                echo "  $0 clean-cache frontend     # 只清理frontend的缓存"
                return 0
            fi
            clean_build_cache "$2"
            ;;
            
        "build-info")
            # 显示镜像的构建信息
            if [[ "${2:-}" == "--help" || "${2:-}" == "-h" ]]; then
                echo "build-info - 显示镜像的构建信息"
                echo
                echo "用法: $0 build-info <service> [tag]"
                echo
                echo "参数:"
                echo "  service     服务名称 (必需)"
                echo "  tag         镜像标签 (默认: $DEFAULT_IMAGE_TAG)"
                echo
                echo "说明:"
                echo "  显示镜像中嵌入的构建信息，包括："
                echo "  • 构建ID"
                echo "  • 构建时间"
                echo "  • 文件哈希"
                echo "  • 构建原因"
                echo
                echo "示例:"
                echo "  $0 build-info frontend"
                echo "  $0 build-info backend v1.0.0"
                return 0
            fi
            
            if [[ -z "$2" ]]; then
                print_error "请指定服务名称"
                exit 1
            fi
            
            local service="$2"
            local tag="${3:-$DEFAULT_IMAGE_TAG}"
            local image="ai-infra-${service}:${tag}"
            
            if ! docker image inspect "$image" >/dev/null 2>&1; then
                print_error "镜像不存在: $image"
                exit 1
            fi
            
            echo "=========================================="
            echo "镜像构建信息: $image"
            echo "=========================================="
            get_image_build_labels "$image"
            ;;
            
        "build")
            if [[ -z "$2" ]]; then
                print_error "请指定要构建的服务"
                print_info "可用服务: $SRC_SERVICES"
                exit 1
            fi
            
            # 支持逗号分隔的服务列表: ./build.sh build backend,backend-init --force
            local services="$2"
            local tag="${3:-$DEFAULT_IMAGE_TAG}"
            local registry="$4"
            
            # 检查是否有 --force 标志（可能在任意位置）
            for arg in "$@"; do
                if [[ "$arg" == "--force" ]]; then
                    FORCE_REBUILD=true
                    print_info "🔨 启用强制重建模式"
                    break
                fi
            done
            
            # 如果包含逗号，则分割服务列表
            if [[ "$services" == *","* ]]; then
                print_info "📦 批量构建模式：检测到多个服务"
                IFS=',' read -ra service_array <<< "$services"
                local total=${#service_array[@]}
                local current=0
                local failed_services=()
                
                print_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
                print_info "构建计划："
                for svc in "${service_array[@]}"; do
                    # 去除前后空格
                    svc=$(echo "$svc" | xargs)
                    echo "  • $svc"
                done
                print_info "总计: $total 个服务"
                print_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
                echo
                
                for svc in "${service_array[@]}"; do
                    # 去除前后空格
                    svc=$(echo "$svc" | xargs)
                    current=$((current + 1))
                    
                    echo
                    print_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
                    print_info "[$current/$total] 构建服务: $svc"
                    print_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
                    
                    if build_service "$svc" "$tag" "$registry"; then
                        print_success "✓ [$current/$total] $svc 构建成功"
                    else
                        print_error "✗ [$current/$total] $svc 构建失败"
                        failed_services+=("$svc")
                    fi
                done
                
                echo
                print_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
                print_info "📊 批量构建结果汇总"
                print_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
                print_info "总服务数: $total"
                print_info "成功: $((total - ${#failed_services[@]}))"
                print_info "失败: ${#failed_services[@]}"
                
                if [[ ${#failed_services[@]} -gt 0 ]]; then
                    print_error "失败的服务："
                    for svc in "${failed_services[@]}"; do
                        echo "  ✗ $svc"
                    done
                    exit 1
                else
                    print_success "🎉 所有服务构建成功！"
                fi
            else
                # 单个服务构建
                build_service "$services" "$tag" "$registry"
            fi
            ;;
            
        "build-all")
            # 当用户传入 --help/-h 时，仅打印帮助而不执行
            if [[ "${2:-}" == "--help" || "${2:-}" == "-h" || "${3:-}" == "--help" || "${3:-}" == "-h" ]]; then
                echo "build-all - 一键生成环境并构建所有服务"
                echo
                echo "用法: $0 build-all [tag] [registry] [--force]"
                echo
                echo "参数:"
                echo "  tag         镜像标签 (默认: $DEFAULT_IMAGE_TAG)"
                echo "  registry    目标镜像仓库 (可选，默认使用本地构建)"
                echo "  --force     全局开关：强制覆盖生成 .env 等（可放在任意位置）"
                echo
                echo "构建流程 (6个步骤):"
                echo "  0) check-status              - 检查当前构建状态，智能过滤"
                echo "  1) prefetch-images           - 预拉取所有依赖镜像"
                echo "  2) smart-tag                 - 智能镜像别名管理（公网/内网自适应）"
                echo "  3) sync-config               - 同步配置文件"
                echo "  4) render-templates          - 渲染配置模板"
                echo "  5) build-services            - 构建服务镜像"
                echo "  6) verify-result             - 验证构建结果"
                echo
                echo "智能功能:"
                echo "  • 智能构建：默认只构建缺失或无效的镜像"
                echo "  • 网络检测：自动检测公网/内网环境"
                echo "  • 镜像别名：根据环境自动创建合适的镜像别名"
                echo "    - 公网环境：原始镜像 → localhost/ 别名"
                echo "    - 内网环境：Harbor镜像 → 原始镜像 + localhost/ 别名"
                echo
                echo "环境变量:"
                echo "  INTERNAL_REGISTRY            - 内网 Harbor 仓库地址 (默认: aiharbor.msxf.local/aihpc)"
                echo "  AI_INFRA_NETWORK_ENV         - 强制指定网络环境 (external/internal)"
                echo
                echo "示例:"
                echo "  $0 build-all"
                echo "  $0 build-all v1.0.0"
                echo "  $0 build-all v1.0.0 harbor.company.com/ai-infra --force"
                echo "  INTERNAL_REGISTRY=my-harbor.com/repo $0 build-all"
                return 0
            fi

            # 将 build-all 封装为一键流程：create-env dev -> sync-config -> build-all
            # 仍然支持传入 [tag] [registry]，并继承 --force 标志
            build_all_pipeline "${2:-$DEFAULT_IMAGE_TAG}" "$3"
            ;;
            
        "test-push")
            if [[ -z "$2" ]]; then
                print_error "请指定要测试推送的服务"
                print_info "可用服务: $SRC_SERVICES"
                exit 1
            fi
            if [[ -z "$3" ]]; then
                print_error "请指定目标 registry"
                exit 1
            fi
            
            local service="$2"
            local registry="$3"
            local tag="${4:-$DEFAULT_IMAGE_TAG}"
            local base_image="ai-infra-${service}:${tag}"
            local target_image=$(get_private_image_name "$base_image" "$registry")
            
            print_info "=========================================="
            print_info "测试推送配置（不实际推送）"
            print_info "=========================================="
            print_info "服务名称: $service"
            print_info "Registry: $registry"
            print_info "标签: $tag"
            print_info "原始镜像: $base_image"
            print_info "目标镜像: $target_image"
            
            # 检查镜像是否存在
            if docker image inspect "$base_image" >/dev/null 2>&1; then
                print_success "✓ 本地镜像存在: $base_image"
            else
                print_warning "✗ 本地镜像不存在: $base_image"
                print_info "需要先构建镜像：./build.sh build $service $tag"
            fi
            
            print_info "推送命令预览："
            print_info "  docker tag $base_image $target_image"
            print_info "  docker push $target_image"
            ;;
            
        "push")
            if [[ -z "$2" ]]; then
                print_error "请指定要推送的服务"
                print_info "可用服务: $SRC_SERVICES"
                exit 1
            fi
            if [[ -z "$3" ]]; then
                print_error "请指定目标 registry"
                exit 1
            fi
            push_service "$2" "${4:-$DEFAULT_IMAGE_TAG}" "$3"
            ;;
            
        "push-all")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                exit 1
            fi
            push_all_services "${3:-$DEFAULT_IMAGE_TAG}" "$2"
            ;;
            
        "build-push")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                exit 1
            fi
            build_and_push_all "${3:-$DEFAULT_IMAGE_TAG}" "$2"
            ;;
            
        # 双环境部署命令
        "build-env")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                print_info "示例: $0 build-env aiharbor.msxf.local/aihpc v0.3.6-dev"
                exit 1
            fi
            build_environment_deploy "$2" "${3:-$DEFAULT_IMAGE_TAG}"
            ;;
            
        "intranet-env")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                print_info "示例: $0 intranet-env aiharbor.msxf.local/aihpc v0.3.6-dev"
                exit 1
            fi
            intranet_environment_deploy "$2" "${3:-$DEFAULT_IMAGE_TAG}"
            ;;
            
        "create-env-prod")
            local mode="${2:-production}"
            local registry="${3:-aiharbor.msxf.local/aihpc}"
            local tag="${4:-$DEFAULT_IMAGE_TAG}"
            create_production_env "$mode" "$registry" "$tag"
            ;;
            
        # 环境配置管理命令
        "create-env")
            local env_type="${2:-dev}"
            local force="false"
            
            # 使用全局 FORCE_REBUILD 标志
            if [[ "$FORCE_REBUILD" == "true" ]]; then
                force="true"
            fi
            
            create_env_from_template "$env_type" "$force"
            ;;
            
        # IP地址检测命令
        "detect-ip")
            local interface="${2:-$DEFAULT_NETWORK_INTERFACE}"
            local show_all="${3:-false}"
            
            if [[ "$2" == "--help" || "$2" == "-h" ]]; then
                echo "detect-ip - 检测网卡IP地址"
                echo
                echo "用法: $0 detect-ip [interface] [--all]"
                echo
                echo "参数:"
                echo "  interface   网卡名称 (默认: $DEFAULT_NETWORK_INTERFACE)"
                echo "  --all       显示所有网卡信息"
                echo
                echo "示例:"
                echo "  $0 detect-ip                # 检测默认网卡($DEFAULT_NETWORK_INTERFACE)"
                echo "  $0 detect-ip eth0           # 检测eth0网卡"
                echo "  $0 detect-ip --all          # 显示所有网卡信息"
                return 0
            fi
            
            if [[ "$show_all" == "--all" ]] || [[ "$show_all" == "-a" ]] || [[ "$interface" == "--all" ]] || [[ "$interface" == "-a" ]]; then
                print_info "检测所有网卡IP地址..."
                echo
                
                # 显示所有网卡信息
                local interfaces=("$DEFAULT_NETWORK_INTERFACE" "${FALLBACK_INTERFACES[@]}")
                for iface in "${interfaces[@]}"; do
                    local ip
                    ip=$(detect_interface_ip "$iface")
                    if [[ -n "$ip" ]]; then
                        echo "  $iface: $ip"
                    else
                        echo "  $iface: (未找到IP)"
                    fi
                done
                
                echo
                print_info "自动检测结果:"
                auto_detect_external_ip_enhanced
            else
                if [[ -n "$interface" ]] && [[ "$interface" != "$DEFAULT_NETWORK_INTERFACE" ]] && [[ "$interface" != "--all" ]] && [[ "$interface" != "-a" ]]; then
                    # 检测指定网卡
                    local ip
                    ip=$(detect_interface_ip "$interface")
                    if [[ -n "$ip" ]]; then
                        echo "$ip"
                    else
                        print_warning "网卡 $interface 未找到IP地址"
                        return 1
                    fi
                else
                    # 自动检测
                    auto_detect_external_ip_enhanced
                fi
            fi
            ;;
            
        # SingleUser 智能构建命令
        "build-singleuser")
            # 处理帮助参数
            if [[ "$2" == "--help" || "$2" == "-h" ]]; then
                echo "build-singleuser - SingleUser 镜像智能构建"
                echo
                echo "用法: $0 build-singleuser [mode] [tag] [registry]"
                echo
                echo "参数:"
                echo "  mode        构建模式 (默认: auto)"
                echo "  tag         镜像标签 (默认: $DEFAULT_IMAGE_TAG)"
                echo "  registry    私有仓库地址 (可选)"
                echo
                echo "构建模式:"
                echo "  auto        - 自动检测网络环境，选择合适的构建策略"
                echo "  offline     - 离线模式，直接使用 aiharbor 内部预构建镜像"
                echo "  online      - 标准模式，保持原始的构建策略"
                echo
                echo "说明:"
                echo "  智能构建 SingleUser Jupyter 镜像，根据网络环境选择最佳策略："
                echo "  • 离线模式：直接使用 aiharbor.msxf.local/aihpc/ai-infra-singleuser 预构建镜像"
                echo "  • 在线模式：使用标准构建流程，从源码重新构建"
                echo "  • 自动模式：检测网络环境，自动选择离线或在线模式"
                echo "  • 构建完成后自动恢复 Dockerfile 原始状态"
                echo
                echo "示例:"
                echo "  $0 build-singleuser auto                      # 自动检测环境"
                echo "  $0 build-singleuser offline v0.3.6-dev       # 使用内部预构建镜像"
                echo "  $0 build-singleuser online v1.0.0 harbor.com/ai # 在线模式推送"
                return 0
            fi
            
            local mode="${2:-auto}"  # auto, offline, online
            local tag="${3:-$DEFAULT_IMAGE_TAG}"
            local registry="${4:-}"
            
            case "$mode" in
                "auto"|"offline"|"online")
                    # 设置构建模式环境变量
                    export SINGLEUSER_BUILD_MODE="$mode"
                    print_info "设置 SingleUser 构建模式: $mode"
                    build_service "singleuser" "$tag" "$registry"
                    ;;
                *)
                    print_error "无效的构建模式: $mode"
                    print_info "可用模式: auto (自动检测), offline (离线友好), online (标准模式)"
                    exit 1
                    ;;
            esac
            ;;
            
        "detect-network")
            local env=$(detect_network_environment)
            print_info "当前网络环境: $env"
            case "$env" in
                "external")
                    print_success "✓ 检测到外网环境，可以正常访问外部服务"
                    ;;
                "internal")
                    print_warning "⚠ 检测到内网环境，建议使用离线友好的构建模式"
                    print_info "建议运行: $0 build-singleuser offline"
                    ;;
            esac
            ;;
            
        "restore-singleuser")
            local service_path="src/singleuser"
            if restore_singleuser_dockerfile "$service_path"; then
                print_success "✓ SingleUser Dockerfile 已恢复到原始状态"
            else
                print_error "✗ 恢复失败"
                exit 1
            fi
            ;;
            
        # 更新外部主机配置命令
        "update-host")
            local host_ip="${2:-auto}"
            update_external_host_config "$host_ip"
            ;;
            
        # 更新外部端口配置命令
        "update-port")
            local port="${2:-8080}"
            update_external_port_config "$port"
            ;;
            
        # 一键更新端口并重新部署
        "quick-deploy")
            local port="${2:-8080}"
            local host="${3:-auto}"
            quick_deploy_with_port "$port" "$host"
            ;;
            
        "auto-env")
            local force="false"
            
            # 使用全局 FORCE_REBUILD 标志
            if [[ "$FORCE_REBUILD" == "true" ]]; then
                force="true"
            fi
            
            auto_generate_env_files "$force"
            ;;
            
        # 生成生产环境密码命令
        "generate-passwords")
            local env_file="${2:-.env.prod}"
            local force="false"
            if [[ "$FORCE_REBUILD" == "true" || "$3" == "--force" ]]; then
                force="true"
            fi
            
            if generate_production_passwords "$env_file" "$force"; then
                print_success "✓ 生产环境密码生成完成"
            else
                print_error "密码生成失败"
                exit 1
            fi
            ;;
            
        # 依赖镜像管理命令
        "deps-pull")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                print_info "用法: $0 deps-pull <registry> [tag]"
                exit 1
            fi
            pull_and_tag_dependencies "$2" "${3:-v0.3.6-dev}"
            ;;
            
        "deps-push")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                print_info "用法: $0 deps-push <registry> [tag]"
                exit 1
            fi
            push_dependencies "$2" "${3:-v0.3.6-dev}"
            ;;
            
        "deps-all")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                exit 1
            fi
            local deps_tag="${3:-v0.3.6-dev}"
            print_info "执行完整的依赖镜像操作..."
            if pull_and_tag_dependencies "$2" "$deps_tag"; then
                push_dependencies "$2" "$deps_tag"
            else
                print_error "依赖镜像拉取失败，停止推送操作"
                exit 1
            fi
            ;;
            
        # AI Harbor 镜像拉取命令
        "harbor-pull-services")
            local harbor_registry="${2:-aiharbor.msxf.local/aihpc}"
            local harbor_tag="${3:-$DEFAULT_IMAGE_TAG}"
            pull_aiharbor_services "$harbor_registry" "$harbor_tag"
            ;;
            
        "harbor-pull-deps")
            local harbor_registry="${2:-aiharbor.msxf.local/aihpc}"
            local harbor_tag="${3:-$DEFAULT_IMAGE_TAG}"
            pull_aiharbor_dependencies "$harbor_registry" "$harbor_tag"
            ;;
            
        "harbor-pull-all")
            local harbor_registry="${2:-aiharbor.msxf.local/aihpc}"
            local harbor_tag="${3:-$DEFAULT_IMAGE_TAG}"
            pull_aiharbor_all "$harbor_registry" "$harbor_tag"
            ;;
            
        "deps-prod")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                exit 1
            fi
            local deps_tag="${3:-v0.3.6-dev}"
            print_info "执行生产环境依赖镜像操作（排除测试工具）..."
            if pull_and_tag_production_dependencies "$2" "$deps_tag"; then
                push_production_dependencies "$2" "$deps_tag"
            else
                print_error "生产环境依赖镜像拉取失败，停止推送操作"
                exit 1
            fi
            ;;
            
        "prod-deploy")
            if [[ -z "$2" ]]; then
                print_error "请指定部署的HOST地址"
                print_info "用法: $0 prod-deploy <host> [registry] [tag]"
                print_info "示例: $0 prod-deploy 192.168.1.100 harbor.company.com/ai-infra v1.0.0"
                print_info "示例: $0 prod-deploy example.com \"\" v1.0.0  # 使用本地镜像"
                exit 1
            fi
            deploy_to_host "$2" "${3:-}" "${4:-$DEFAULT_IMAGE_TAG}"
            ;;
            
        "prod-up")
            # registry 参数可以为空（使用本地镜像）
            # 检查是否有 --force 或 --skip-pull 参数
            local force_local="false"
            if [[ "$FORCE_REBUILD" == "true" || "$SKIP_PULL" == "true" ]]; then
                force_local="true"
            fi
            start_production "${2:-}" "${3:-$DEFAULT_IMAGE_TAG}" "$force_local"
            ;;
            
        "prod-down")
            stop_production
            ;;
            
        "prod-restart")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                exit 1
            fi
            restart_production "$2" "${3:-$DEFAULT_IMAGE_TAG}"
            ;;
            
        "prod-status")
            production_status
            ;;
            
        "prod-logs")
            local follow="false"
            if [[ "$3" == "--follow" || "$3" == "-f" ]]; then
                follow="true"
            fi
            production_logs "$2" "$follow"
            ;;
            
        # Mock 测试环境命令
        "mock-setup")
            setup_mock_environment "${2:-$DEFAULT_IMAGE_TAG}"
            ;;
            
        "mock-up"|"mock-start")
            run_mock_tests "${2:-$DEFAULT_IMAGE_TAG}" "up"
            ;;
            
        "mock-down"|"mock-stop")
            run_mock_tests "${2:-$DEFAULT_IMAGE_TAG}" "down"
            ;;
            
        "mock-restart")
            run_mock_tests "${2:-$DEFAULT_IMAGE_TAG}" "restart"
            ;;
            
        "mock-test")
            run_mock_tests "${2:-$DEFAULT_IMAGE_TAG}" "test"
            ;;
            
        # 镜像验证命令
        "verify")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                print_info "用法: $0 verify <registry> [tag]"
                exit 1
            fi
            verify_private_images "$2" "${3:-v0.3.6-dev}"
            ;;
            
        "verify-key")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                print_info "用法: $0 verify-key <registry> [tag]"
                exit 1
            fi
            verify_key_images "$2" "${3:-v0.3.6-dev}"
            ;;
            
        "clean")
            local clean_type="${2:-ai-infra}"
            local tag_or_force="$3"
            local force_flag="$4"
            local force="false"
            local tag="$DEFAULT_IMAGE_TAG"
            
            # 解析参数
            case "$clean_type" in
                "ai-infra"|*)
                    # 默认清理AI-Infra镜像（保持原有行为）
                    if [[ "$clean_type" != "ai-infra" && "$clean_type" != "--force" ]]; then
                        tag="$clean_type"
                    fi
                    if [[ "$tag_or_force" == "--force" ]]; then
                        force="true"
                    elif [[ -n "$tag_or_force" && "$tag_or_force" != "--force" && "$clean_type" == "ai-infra" ]]; then
                        tag="$tag_or_force"
                        if [[ "$force_flag" == "--force" ]]; then
                            force="true"
                        fi
                    fi
                    clean_images "$tag" "$force"
                    ;;
            esac
            ;;
            
        "clean-all")
            # 检查是否需要帮助
            if [[ "$2" == "--help" || "$2" == "-h" ]]; then
                clean_all "--help"
                exit 0
            fi
            
            # 使用全局FORCE_REBUILD变量
            local force="false"
            if [[ "$FORCE_REBUILD" == "true" ]]; then
                force="true"
            fi
            clean_all "$force"
            ;;
            
        # 智能镜像tag命令
        "tag-localhost")
            if [[ "$2" == "--help" || "$2" == "-h" ]]; then
                echo "tag-localhost - 智能镜像tag管理（支持公网/内网环境）"
                echo
                echo "用法: $0 tag-localhost [选项] [image...]"
                echo
                echo "选项:"
                echo "  --network <env>     指定网络环境 (auto/external/internal)"
                echo "  --harbor <registry> 指定 Harbor 仓库地址"
                echo
                echo "参数:"
                echo "  image               镜像名称（可指定多个）"
                echo "                      不指定镜像时，自动处理所有 Dockerfile 中的基础镜像"
                echo
                echo "网络环境策略:"
                echo "  auto (默认)         自动检测网络环境并选择合适的策略"
                echo "  external (公网)     优先使用原始镜像名称，同时创建 localhost/ 别名"
                echo "  internal (内网)     使用 Harbor 仓库镜像，创建原始名称和 localhost/ 别名"
                echo
                echo "功能:"
                echo "  公网环境："
                echo "    • 优先使用原始镜像名称（如 redis:7-alpine）"
                echo "    • 自动创建 localhost/ 前缀别名（兼容性）"
                echo "  内网环境："
                echo "    • 从 Harbor 仓库获取镜像（如 aiharbor.msxf.local/aihpc/redis:7-alpine）"
                echo "    • 创建原始名称别名（如 redis:7-alpine）"
                echo "    • 创建 localhost/ 别名（如 localhost/redis:7-alpine）"
                echo
                echo "应用场景:"
                echo "  • 公网环境：确保镜像可用，创建兼容性别名"
                echo "  • 内网环境：从 Harbor 拉取镜像，创建标准别名"
                echo "  • 混合环境：自动检测并应用最佳策略"
                echo
                echo "示例:"
                echo "  $0 tag-localhost                                    # 自动处理所有依赖镜像"
                echo "  $0 tag-localhost redis:7-alpine                     # 处理单个镜像"
                echo "  $0 tag-localhost --network external redis:7-alpine  # 强制公网模式"
                echo "  $0 tag-localhost --network internal                 # 内网模式处理所有镜像"
                echo "  $0 tag-localhost --harbor my-harbor.com/repo        # 指定 Harbor 仓库"
                return 0
            fi
            
            # 解析参数
            local network_env="auto"
            local harbor_registry="${INTERNAL_REGISTRY:-aiharbor.msxf.local/aihpc}"
            local images_to_process=()
            
            while [[ $# -gt 1 ]]; do
                case "$2" in
                    --network)
                        network_env="$3"
                        shift 2
                        ;;
                    --harbor)
                        harbor_registry="$3"
                        shift 2
                        ;;
                    *)
                        images_to_process+=("$2")
                        shift
                        ;;
                esac
            done
            
            # 如果没有指定镜像，自动从所有 Dockerfile 中提取基础镜像
            if [[ ${#images_to_process[@]} -eq 0 ]]; then
                print_info "未指定镜像，将从所有 Dockerfile 中提取基础镜像..."
                
                # 动态提取所有 Dockerfile 中的基础镜像
                local all_images=()
                local services_list=($SRC_SERVICES)
                
                print_info "📋 扫描所有服务的 Dockerfile..."
                
                for service in "${services_list[@]}"; do
                    local service_path=$(get_service_path "$service")
                    if [[ -z "$service_path" ]]; then
                        continue
                    fi
                    
                    local dockerfile_path="$SCRIPT_DIR/$service_path/Dockerfile"
                    if [[ ! -f "$dockerfile_path" ]]; then
                        continue
                    fi
                    
                    # 提取该 Dockerfile 的基础镜像
                    local images
                    images=$(extract_base_images "$dockerfile_path")
                    
                    if [[ -n "$images" ]]; then
                        while IFS= read -r image; do
                            # 跳过空行
                            if [[ -z "$image" ]]; then
                                continue
                            fi
                            # 跳过内部构建阶段（只包含小写字母、下划线、连字符的名称）
                            if [[ "$image" =~ ^[a-z_-]+$ ]]; then
                                continue
                            fi
                            # 跳过注释
                            if [[ "$image" =~ ^# ]]; then
                                continue
                            fi
                            # 添加到数组
                            all_images+=("$image")
                        done <<< "$images"
                    fi
                done
                
                # 去重并排序
                local unique_images=($(printf '%s\n' "${all_images[@]}" | sort -u))
                
                if [[ ${#unique_images[@]} -eq 0 ]]; then
                    print_warning "未找到任何基础镜像"
                    return 0
                fi
                
                print_info "📦 发现 ${#unique_images[@]} 个唯一的基础镜像"
                
                batch_tag_images_smart "$network_env" "$harbor_registry" "${unique_images[@]}"
            else
                # 处理用户指定的镜像
                batch_tag_images_smart "$network_env" "$harbor_registry" "${images_to_process[@]}"
            fi
            ;;
            
        "reset-db")
            # 检查是否需要帮助
            if [[ "$2" == "--help" || "$2" == "-h" ]]; then
                reset_database "--help"
                exit 0
            fi
            
            # 使用全局FORCE_REBUILD变量
            local force="false"
            if [[ "$FORCE_REBUILD" == "true" ]]; then
                force="true"
            fi
            reset_database "$force"
            ;;
            
        "render-templates")
            case "${2:-all}" in
                "nginx")
                    render_nginx_templates
                    ;;
                "jupyterhub")
                    render_jupyterhub_templates
                    ;;
                "docker-compose"|"compose")
                    # 支持 registry/tag 以及附加可选参数（例如 --oceanbase-init-dir）
                    # 将从第3个参数开始的所有参数透传给渲染函数
                    shift 2
                    render_docker_compose_templates "$@"
                    ;;
                "env")
                    # 同步 .env 和 .env.example 文件
                    sync_env_files
                    ;;
                "all")
                    render_nginx_templates
                    render_jupyterhub_templates
                    # 对于all模式，透传后续参数给 docker-compose 渲染
                    shift 2
                    render_docker_compose_templates "$@"
                    ;;
                *)
                    print_error "未知的模板类型: $2"
                    print_info "可用模板类型: nginx, jupyterhub, docker-compose, env, all"
                    exit 1
                    ;;
            esac
            ;;
            
        "sync-config")
            # 整体配置同步命令
            sync_all_configs "${2:-false}"
            ;;
            
        "version")
            echo "AI Infrastructure Matrix Build Script"
            echo "Version: $VERSION"
            echo "Default Tag: $DEFAULT_IMAGE_TAG"
            echo "Services: $SRC_SERVICES"
            echo
            echo "Dependency Images:"
            for dep in $DEPENDENCY_IMAGES; do
                echo "  • $dep"
            done
            ;;
            
        "validate-env")
            validate_env_consistency
            ;;
            
        "kafka-start")
            start_kafka_services "${2:-docker-compose.yml}"
            ;;
            
        "kafka-stop")
            stop_kafka_services "${2:-docker-compose.yml}"
            ;;
            
        "kafka-restart")
            restart_kafka_services "${2:-docker-compose.yml}"
            ;;
            
        "kafka-status")
            check_kafka_status "${2:-docker-compose.yml}"
            ;;
            
        "kafka-test")
            test_kafka_full "${2:-docker-compose.yml}"
            ;;
            
        "kafka-topics")
            list_kafka_topics "${2:-docker-compose.yml}"
            ;;
            
        "kafka-logs")
            if [[ -z "$2" ]]; then
                show_kafka_logs "kafka" "${3:-docker-compose.yml}" "$4"
            else
                show_kafka_logs "$2" "${3:-docker-compose.yml}" "$4"
            fi
            ;;
            
        # 离线部署命令
        "export-offline")
            local output_dir="${2:-./offline-images}"
            local tag="${3:-$DEFAULT_IMAGE_TAG}"
            local include_kafka="${4:-true}"
            export_offline_images "$output_dir" "$tag" "$include_kafka"
            ;;
            
        "push-to-internal")
            if [[ -z "$2" ]]; then
                print_error "请指定内部仓库地址"
                print_info "用法: $0 push-to-internal <registry> [tag] [include_kafka]"
                exit 1
            fi
            local registry="$2"
            local tag="${3:-$DEFAULT_IMAGE_TAG}"
            local include_kafka="${4:-true}"
            push_to_internal_registry "$registry" "$tag" "$include_kafka"
            ;;
            
        # 统一构建和部署命令
        "unified-build")
            local registry="${2:-aiharbor.msxf.local/aihpc}"
            local tag="${3:-$DEFAULT_IMAGE_TAG}"
            local external_host="${4:-172.20.10.11}"
            local external_port="${5:-80}"
            local external_scheme="${6:-http}"
            build_all_unified "$registry" "$tag" "$external_host" "$external_port" "$external_scheme"
            ;;
            
        "unified-build-push")
            local registry="${2:-aiharbor.msxf.local/aihpc}"
            local tag="${3:-$DEFAULT_IMAGE_TAG}"
            local external_host="${4:-172.20.10.11}"
            local external_port="${5:-80}"
            local external_scheme="${6:-http}"
            build_and_push_unified "$registry" "$tag" "$external_host" "$external_port" "$external_scheme"
            ;;
            
        "unified-deploy")
            local registry="${2:-aiharbor.msxf.local/aihpc}"
            local tag="${3:-$DEFAULT_IMAGE_TAG}"
            local external_host="${4:-172.20.10.11}"
            local external_port="${5:-80}"
            local external_scheme="${6:-http}"
            local compose_file="${7:-docker-compose.yml}"
            deploy_unified "$registry" "$tag" "$external_host" "$external_port" "$external_scheme" "$compose_file"
            ;;
            
        "unified-all"|"all-in-one")
            local registry="${2:-aiharbor.msxf.local/aihpc}"
            local tag="${3:-$DEFAULT_IMAGE_TAG}"
            local external_host="${4:-172.20.10.11}"
            local external_port="${5:-80}"
            local external_scheme="${6:-http}"
            local compose_file="${7:-docker-compose.yml}"
            build_deploy_all "$registry" "$tag" "$external_host" "$external_port" "$external_scheme" "$compose_file"
            ;;
            
        "prepare-offline")
            if [[ -z "$2" ]]; then
                print_error "请指定内部仓库地址"
                print_info "用法: $0 prepare-offline <registry> [tag] [output_dir] [include_kafka]"
                exit 1
            fi
            local registry="$2"
            local tag="${3:-$DEFAULT_IMAGE_TAG}"
            local output_dir="${4:-./offline-deployment}"
            local include_kafka="${5:-true}"
            prepare_offline_deployment "$registry" "$tag" "$output_dir" "$include_kafka"
            ;;
            
        # CI/CD构建命令（适用于能访问外网的构建环境）
        "ci-build")
            # 检查是否需要帮助
            if [[ "$2" == "--help" || "$2" == "-h" ]]; then
                echo "ci-build - CI/CD完整构建流程（适用于外网环境）"
                echo
                echo "用法: $0 ci-build <registry> [tag] [external_host]"
                echo
                echo "参数:"
                echo "  registry        目标镜像仓库地址 (必需)"
                echo "  tag             镜像标签 (默认: $DEFAULT_IMAGE_TAG)" 
                echo "  external_host   外部访问地址 (默认: 自动检测)"
                echo
                echo "功能:"
                echo "  • 自动生成配置模板"
                echo "  • 构建所有服务镜像"
                echo "  • 拉取并重新标记依赖镜像"
                echo "  • 推送所有镜像到指定仓库"
                echo "  • 生成生产环境配置文件"
                echo
                echo "示例:"
                echo "  $0 ci-build harbor.company.com/ai-infra"
                echo "  $0 ci-build harbor.company.com/ai-infra v1.0.0"
                echo "  $0 ci-build harbor.company.com/ai-infra v1.0.0 192.168.1.100"
                return 0
            fi
            
            if [[ -z "$2" ]]; then
                print_error "请指定目标镜像仓库地址"
                print_info "用法: $0 ci-build <registry> [tag] [external_host]"
                print_info "使用 '$0 ci-build --help' 查看详细说明"
                exit 1
            fi
            
            ci_build_complete "$2" "${3:-$DEFAULT_IMAGE_TAG}" "$4"
            ;;
            
        # 生产环境启动命令（适用于无外网访问的生产环境）
        "prod-start")
            # 检查是否需要帮助
            if [[ "$2" == "--help" || "$2" == "-h" ]]; then
                echo "prod-start - 生产环境服务启动（适用于内网环境）"
                echo
                echo "用法: $0 prod-start [registry] [tag] [external_host] [external_port]"
                echo
                echo "参数:"
                echo "  registry        内部镜像仓库地址 (可选，默认使用本地镜像)"
                echo "  tag             镜像标签 (默认: $DEFAULT_IMAGE_TAG)"
                echo "  external_host   外部访问地址 (默认: 自动检测)"
                echo "  external_port   外部访问端口 (默认: 8080)"
                echo
                echo "功能:"
                echo "  • 从内部仓库拉取镜像（如果指定）"
                echo "  • 生成生产环境配置"
                echo "  • 启动所有服务"
                echo "  • 检查服务状态"
                echo
                echo "示例:"
                echo "  $0 prod-start                                      # 使用本地镜像"
                echo "  $0 prod-start aiharbor.msxf.local/aihpc          # 从内部仓库拉取"
                echo "  $0 prod-start aiharbor.msxf.local/aihpc v1.0.0   # 指定版本"
                echo "  $0 prod-start \"\" v1.0.0 192.168.1.100 80         # 本地镜像+自定义地址"
                return 0
            fi
            
            prod_start_complete "${2:-}" "${3:-$DEFAULT_IMAGE_TAG}" "$4" "$5"
            ;;
            
        # 自动化补丁管理命令
        "patch")
            # 检查是否需要帮助
            if [[ "$2" == "--help" || "$2" == "-h" || -z "$2" ]]; then
                echo "patch - 自动化代码补丁管理"
                echo
                echo "用法: $0 patch <patch-name> [service] [rebuild]"
                echo
                echo "参数:"
                echo "  patch-name      补丁名称 (必需)"
                echo "  service         目标服务 (自定义补丁时必需)"
                echo "  rebuild         是否重建服务 (默认: true)"
                echo
                echo "功能:"
                echo "  • 自动应用预定义的代码修复"
                echo "  • 备份原始文件"
                echo "  • 验证代码语法"
                echo "  • 自动重建和重启服务"
                echo
                echo "可用补丁:"
                list_available_patches
                return 0
            fi
            
            apply_patch "$2" "$3" "$4"
            ;;
            
        "generate-patch")
            # 检查是否需要帮助
            if [[ "$2" == "--help" || "$2" == "-h" ]]; then
                echo "generate-patch - 生成服务补丁文件"
                echo
                echo "用法: $0 generate-patch <service> [output-file]"
                echo
                echo "参数:"
                echo "  service         目标服务名称 (必需)"
                echo "  output-file     输出补丁文件路径 (可选)"
                echo
                echo "功能:"
                echo "  • 基于当前代码变更生成补丁文件"
                echo "  • 支持git diff格式"
                echo "  • 可用于代码分发和应用"
                echo
                echo "示例:"
                echo "  $0 generate-patch backend"
                echo "  $0 generate-patch frontend ./my-frontend.patch"
                return 0
            fi
            
            generate_patch "$2" "$3"
            ;;
            
        "build-history")
            # 检查是否需要帮助
            if [[ "$2" == "--help" || "$2" == "-h" ]]; then
                echo "build-history - 查看构建历史记录"
                echo
                echo "用法: $0 build-history [service] [count]"
                echo
                echo "参数:"
                echo "  service         过滤指定服务 (可选)"
                echo "  count           显示最近N条记录 (默认: 20)"
                echo
                echo "功能:"
                echo "  • 显示构建历史记录"
                echo "  • 包含 BUILD_ID、服务、标签、状态"
                echo "  • 支持按服务过滤"
                echo "  • 彩色输出，易于阅读"
                echo
                echo "示例:"
                echo "  $0 build-history                    # 显示最近20条记录"
                echo "  $0 build-history backend            # 显示backend的构建历史"
                echo "  $0 build-history backend 50         # 显示backend最近50条记录"
                echo "  $0 build-history \"\" 100             # 显示所有服务最近100条记录"
                return 0
            fi
            
            show_build_history "${2:-}" "${3:-20}"
            ;;
            
        "build-info")
            # 检查是否需要帮助
            if [[ "$2" == "--help" || "$2" == "-h" || -z "$2" ]]; then
                echo "build-info - 查看镜像构建信息"
                echo
                echo "用法: $0 build-info <service> [tag]"
                echo
                echo "参数:"
                echo "  service         服务名称 (必需)"
                echo "  tag             镜像标签 (默认: $DEFAULT_IMAGE_TAG)"
                echo
                echo "功能:"
                echo "  • 显示镜像的构建标签"
                echo "  • 包含 BUILD_ID、哈希、时间戳等"
                echo "  • 验证镜像是否存在"
                echo
                echo "示例:"
                echo "  $0 build-info backend"
                echo "  $0 build-info frontend v1.0.0"
                return 0
            fi
            
            show_build_info "$2" "${3:-$DEFAULT_IMAGE_TAG}"
            ;;
            
        "help"|"-h"|"--help")
            show_help
            ;;
            
        *)
            print_error "未知命令: $1"
            print_info "使用 '$0 help' 查看可用命令"
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"
