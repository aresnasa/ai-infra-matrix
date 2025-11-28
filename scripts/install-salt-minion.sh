#!/bin/bash
# =============================================================================
# SaltStack Minion 安装脚本模板
# 此脚本由 build.sh render 自动生成，变量已被替换为实际值
# =============================================================================
# 生成时间: 由 build.sh sync 命令渲染
# 
# 变量说明:
#   192.168.0.200 - Salt Master 外部访问地址
#   4505 - Salt Master 端口 (默认 4505)
#   28080 - AppHub 端口 (默认 8090)
# =============================================================================

set -eo pipefail

# ===========================================
# 环境配置 (已由模板渲染替换)
# ===========================================
SALT_MASTER="${SALT_MASTER:-192.168.0.200}"
SALT_MASTER_PORT="${SALT_MASTER_PORT:-4505}"
SALT_VERSION="${SALT_VERSION:-3007.1}"
MINION_ID="${MINION_ID:-}"
APPHUB_URL="${APPHUB_URL:-http://192.168.0.200:28080}"
USE_OFFICIAL="${USE_OFFICIAL:-false}"

# ===========================================
# 颜色输出
# ===========================================
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $*"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }

# ===========================================
# 帮助信息
# ===========================================
usage() {
    cat <<EOF
SaltStack Minion 安装脚本 (预配置模板)

此脚本已预配置以下默认值:
  Salt Master: 192.168.0.200:4505
  AppHub URL:  http://192.168.0.200:28080

用法: $0 [OPTIONS]

选项:
    -h, --help              显示帮助信息
    --master MASTER         覆盖 Salt Master 地址
    --minion-id MINION_ID   设置 Minion ID (默认使用主机名)
    --salt-version VERSION  SaltStack 版本 (默认: 3007.1)
    --use-official          强制使用官方源

环境变量:
    SALT_MASTER             覆盖 Salt Master 地址
    MINION_ID               Minion ID
    APPHUB_URL              AppHub URL

示例:
    # 使用预配置的默认值安装
    $0

    # 指定 Minion ID
    $0 --minion-id worker01

    # 使用 curl 管道安装 (已知 Master 地址)
    curl -fsSL http://192.168.0.200:28080/packages/install-salt-minion.sh | bash
EOF
    exit 0
}

# ===========================================
# 参数解析
# ===========================================
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                usage
                ;;
            --master)
                SALT_MASTER="$2"
                shift 2
                ;;
            --minion-id)
                MINION_ID="$2"
                shift 2
                ;;
            --salt-version)
                SALT_VERSION="$2"
                shift 2
                ;;
            --use-official)
                USE_OFFICIAL="true"
                shift
                ;;
            -*)
                log_error "未知选项: $1"
                usage
                ;;
            *)
                shift
                ;;
        esac
    done
}

# ===========================================
# 系统检测函数
# ===========================================
detect_os() {
    if command -v apt-get >/dev/null 2>&1; then
        echo "debian"
    elif command -v dnf >/dev/null 2>&1; then
        echo "rhel-dnf"
    elif command -v yum >/dev/null 2>&1; then
        echo "rhel-yum"
    else
        echo "unknown"
    fi
}

get_os_version() {
    if [[ -f /etc/os-release ]]; then
        . /etc/os-release
        echo "$ID:$VERSION_ID"
    elif [[ -f /etc/redhat-release ]]; then
        local version=$(cat /etc/redhat-release | grep -oE '[0-9]+' | head -1)
        echo "rhel:$version"
    else
        echo "unknown:0"
    fi
}

# ===========================================
# AppHub 检查
# ===========================================
check_apphub() {
    local url="$1"
    
    if [[ -z "$url" ]]; then
        return 1
    fi
    
    log_info "检查 AppHub 可用性: $url"
    
    if command -v wget >/dev/null 2>&1; then
        if timeout 10 wget -q --spider "$url/pkgs/saltstack-deb/Packages" 2>/dev/null || \
           timeout 10 wget -q --spider "$url/pkgs/saltstack-rpm/repodata/repomd.xml" 2>/dev/null; then
            return 0
        fi
    fi
    
    if command -v curl >/dev/null 2>&1; then
        if timeout 10 curl -fsL "$url/pkgs/saltstack-deb/Packages" >/dev/null 2>&1 || \
           timeout 10 curl -fsL "$url/pkgs/saltstack-rpm/repodata/repomd.xml" >/dev/null 2>&1; then
            return 0
        fi
    fi
    
    return 1
}

# ===========================================
# 安装函数 - AppHub
# ===========================================
install_apphub_debian() {
    local apphub_url="$1"
    
    log_info "从 AppHub 安装 SaltStack (Debian/Ubuntu)..."
    
    export DEBIAN_FRONTEND=noninteractive
    
    echo "deb [trusted=yes] ${apphub_url}/pkgs/saltstack-deb ./" > /etc/apt/sources.list.d/ai-infra-salt.list
    
    apt-get update
    apt-get install -y --no-install-recommends salt-minion
}

install_apphub_rhel() {
    local apphub_url="$1"
    local pkg_mgr="$2"
    
    log_info "从 AppHub 安装 SaltStack (RHEL/CentOS)..."
    
    cat > /etc/yum.repos.d/ai-infra-salt.repo <<EOF
[ai-infra-salt]
name=AI Infra Salt RPMs (AppHub)
baseurl=${apphub_url}/pkgs/saltstack-rpm
enabled=1
gpgcheck=0
priority=1
EOF
    
    ${pkg_mgr} clean all
    ${pkg_mgr} makecache
    ${pkg_mgr} install -y salt-minion
}

# ===========================================
# 安装函数 - 官方源
# ===========================================
install_official_debian() {
    log_info "从官方源安装 SaltStack (Debian/Ubuntu)..."
    
    export DEBIAN_FRONTEND=noninteractive
    
    apt-get update
    apt-get install -y curl gnupg2
    
    local debian_version
    if [[ -f /etc/debian_version ]]; then
        debian_version=$(cat /etc/debian_version | cut -d. -f1)
    else
        local ubuntu_version=$(grep VERSION_ID /etc/os-release | cut -d'"' -f2 | cut -d. -f1)
        case $ubuntu_version in
            22|23|24) debian_version="12" ;;
            20|21) debian_version="11" ;;
            *) debian_version="12" ;;
        esac
    fi
    
    curl -fsSL "https://repo.saltproject.io/salt/py3/debian/${debian_version}/amd64/SALT-PROJECT-GPG-PUBKEY-2023.pub" | \
        gpg --dearmor -o /usr/share/keyrings/salt-archive-keyring.gpg 2>/dev/null || true
    
    echo "deb [signed-by=/usr/share/keyrings/salt-archive-keyring.gpg] https://repo.saltproject.io/salt/py3/debian/${debian_version}/amd64/latest ./" \
        > /etc/apt/sources.list.d/salt.list
    
    apt-get update
    apt-get install -y salt-minion
}

install_official_rhel() {
    local pkg_mgr="$1"
    
    log_info "从官方源安装 SaltStack (RHEL/CentOS)..."
    
    local rhel_version
    if [[ -f /etc/redhat-release ]]; then
        rhel_version=$(cat /etc/redhat-release | grep -oE '[0-9]+' | head -1)
    else
        rhel_version=$(grep VERSION_ID /etc/os-release | cut -d'"' -f2 | cut -d. -f1)
    fi
    
    rpm --import "https://repo.saltproject.io/salt/py3/redhat/${rhel_version}/x86_64/SALT-PROJECT-GPG-PUBKEY-2023.pub" 2>/dev/null || true
    
    cat > /etc/yum.repos.d/salt.repo <<EOF
[salt-latest]
name=Salt repo
baseurl=https://repo.saltproject.io/salt/py3/redhat/\$releasever/\$basearch/latest
enabled=1
gpgcheck=1
gpgkey=https://repo.saltproject.io/salt/py3/redhat/\$releasever/\$basearch/SALT-PROJECT-GPG-PUBKEY-2023.pub
EOF
    
    ${pkg_mgr} clean all
    ${pkg_mgr} install -y salt-minion
}

# ===========================================
# 配置 Minion
# ===========================================
configure_minion() {
    local master="$1"
    local minion_id="$2"
    
    log_info "配置 Salt Minion..."
    
    mkdir -p /etc/salt
    mkdir -p /var/log/salt
    
    cat > /etc/salt/minion <<EOF
# =============================================================================
# SaltStack Minion 配置
# Auto-generated by install-salt-minion.sh
# Generated at: $(date)
# =============================================================================

# Salt Master 地址
master: ${master}

# Minion ID
id: ${minion_id}

# 日志配置
log_level: info
log_file: /var/log/salt/minion

# 连接配置
retry_dns: 30
master_alive_interval: 30
master_tries: -1
acceptance_wait_time: 10
acceptance_wait_time_max: 60

# 网络配置
tcp_keepalive: true
tcp_keepalive_idle: 300
tcp_keepalive_cnt: 3
tcp_keepalive_intvl: 30
EOF
    
    log_info "配置已写入 /etc/salt/minion"
}

# ===========================================
# 启动服务
# ===========================================
start_service() {
    log_info "启动 Salt Minion 服务..."
    
    systemctl daemon-reload || true
    systemctl enable salt-minion || true
    systemctl restart salt-minion || true
    
    sleep 2
    
    if systemctl is-active --quiet salt-minion; then
        log_info "Salt Minion 服务已启动"
        return 0
    else
        log_warn "Salt Minion 服务可能未正常启动"
        systemctl status salt-minion --no-pager || true
        return 1
    fi
}

# ===========================================
# 主函数
# ===========================================
main() {
    parse_args "$@"
    
    # 验证参数
    if [[ -z "$SALT_MASTER" ]] || [[ "$SALT_MASTER" == "192.168.0.200" ]]; then
        log_error "Salt Master 地址未配置"
        log_error "请确保模板已正确渲染，或使用 --master 参数指定"
        exit 1
    fi
    
    # 设置 Minion ID
    if [[ -z "$MINION_ID" ]]; then
        MINION_ID=$(hostname)
    fi
    
    log_info "=========================================="
    log_info "SaltStack Minion 安装"
    log_info "=========================================="
    log_info "Salt Master: $SALT_MASTER"
    log_info "Minion ID: $MINION_ID"
    log_info "Salt Version: $SALT_VERSION"
    [[ -n "$APPHUB_URL" ]] && log_info "AppHub URL: $APPHUB_URL"
    echo ""
    
    # 检测操作系统
    OS_TYPE=$(detect_os)
    OS_VERSION=$(get_os_version)
    log_info "操作系统: $OS_TYPE ($OS_VERSION)"
    
    if [[ "$OS_TYPE" == "unknown" ]]; then
        log_error "不支持的操作系统"
        exit 1
    fi
    
    # 确定包管理器
    PKG_MGR="dnf"
    [[ "$OS_TYPE" == "rhel-yum" ]] && PKG_MGR="yum"
    
    # 安装 Salt Minion
    local installed=false
    
    # 优先从 AppHub 安装
    if [[ "$USE_OFFICIAL" != "true" ]] && [[ -n "$APPHUB_URL" ]] && [[ "$APPHUB_URL" != *"192.168.0.200"* ]]; then
        if check_apphub "$APPHUB_URL"; then
            log_info "AppHub 可用，从 AppHub 安装..."
            
            if [[ "$OS_TYPE" == "debian" ]]; then
                if install_apphub_debian "$APPHUB_URL"; then
                    installed=true
                fi
            else
                if install_apphub_rhel "$APPHUB_URL" "$PKG_MGR"; then
                    installed=true
                fi
            fi
        else
            log_warn "AppHub 不可用，将使用官方源"
        fi
    fi
    
    # 从官方源安装
    if [[ "$installed" != "true" ]]; then
        log_info "从官方源安装..."
        
        if [[ "$OS_TYPE" == "debian" ]]; then
            install_official_debian
        else
            install_official_rhel "$PKG_MGR"
        fi
    fi
    
    # 验证安装
    if ! command -v salt-minion >/dev/null 2>&1; then
        log_error "Salt Minion 安装失败"
        exit 1
    fi
    
    log_info "Salt Minion 安装成功: $(salt-minion --version)"
    
    # 配置
    configure_minion "$SALT_MASTER" "$MINION_ID"
    
    # 启动服务
    start_service
    
    # 输出结果
    echo ""
    log_info "=========================================="
    log_info "✓ SaltStack Minion 安装完成"
    log_info "=========================================="
    log_info "Master: $SALT_MASTER"
    log_info "Minion ID: $MINION_ID"
    log_info "版本: $(salt-minion --version 2>/dev/null || echo 'unknown')"
    log_info "服务状态: $(systemctl is-active salt-minion 2>/dev/null || echo 'unknown')"
    echo ""
    log_info "Minion 密钥将自动被 Master 接受 (auto_accept: true)"
    log_info "如需手动确认，在 Master 上执行: salt-key -a $MINION_ID"
    log_info "=========================================="
}

main "$@"
