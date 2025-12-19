#!/bin/bash
# =============================================================================
# SaltStack Minion 安装脚本模板
# 此脚本由 build.sh render 自动生成，变量已被替换为实际值
# =============================================================================
# 生成时间: 由 build.sh sync 命令渲染
# 
# 变量说明:
#   192.168.247.121 - Salt Master 外部访问地址
#   4505 - Salt Master 端口 (默认 4505)
#   28080 - AppHub 端口 (默认 8090)
# =============================================================================

set -eo pipefail

# ===========================================
# 环境配置 (已由模板渲染替换)
# ===========================================
SALT_MASTER="${SALT_MASTER:-192.168.247.121}"
SALT_MASTER_PORT="${SALT_MASTER_PORT:-4505}"
SALT_VERSION="${SALT_VERSION:-3007.1}"
MINION_ID="${MINION_ID:-}"
APPHUB_URL="${APPHUB_URL:-http://192.168.247.121:28080}"
USE_OFFICIAL="${USE_OFFICIAL:-false}"
# Master 公钥 URL (用于预同步 Master 公钥到 Minion)
# 支持两种模式:
#   1. 静态模式: 直接从 AppHub 获取 (需要预先同步公钥到 AppHub)
#   2. Token 模式: 通过 API 一次性 Token 安全获取 (推荐，由批量安装自动设置)
MASTER_PUB_URL="${MASTER_PUB_URL:-}"

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
  Salt Master: 192.168.247.121:4505
  AppHub URL:  http://192.168.247.121:28080

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
    curl -fsSL http://192.168.247.121:28080/packages/install-salt-minion.sh | bash
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
# 预同步 Salt Master 公钥到 Minion
# 这样 Minion 启动时就知道要信任哪个 Master
#
# 支持两种获取方式:
#   1. Token 模式: 通过 API 一次性 Token 安全获取（推荐，MASTER_PUB_URL 含 token 参数）
#   2. 静态模式: 从 AppHub 静态文件获取（需要预先同步公钥）
# ===========================================
sync_master_pubkey() {
    log_info "同步 Salt Master 公钥..."
    
    # 创建 Minion PKI 目录
    mkdir -p /etc/salt/pki/minion
    chmod 700 /etc/salt/pki/minion
    
    local master_pub_file="/etc/salt/pki/minion/minion_master.pub"
    local downloaded=false
    
    # 如果没有设置 MASTER_PUB_URL，跳过预同步
    if [[ -z "$MASTER_PUB_URL" ]]; then
        log_info "未配置 MASTER_PUB_URL，跳过 Master 公钥预同步"
        log_info "Minion 将在首次连接时自动获取 Master 公钥"
        return 0
    fi
    
    log_info "尝试获取 Master 公钥: $MASTER_PUB_URL"
    
    # 下载 Master 公钥
    if command -v curl >/dev/null 2>&1; then
        if curl -fsSL --connect-timeout 10 -o "$master_pub_file" "$MASTER_PUB_URL" 2>/dev/null; then
            if [[ -s "$master_pub_file" ]] && grep -q "BEGIN PUBLIC KEY\|BEGIN RSA PUBLIC KEY\|ssh-rsa" "$master_pub_file" 2>/dev/null; then
                downloaded=true
                log_info "✓ Master 公钥下载成功 (curl)"
            else
                rm -f "$master_pub_file"
                log_warn "下载的文件不是有效的公钥"
            fi
        fi
    elif command -v wget >/dev/null 2>&1; then
        if wget -q --timeout=10 -O "$master_pub_file" "$MASTER_PUB_URL" 2>/dev/null; then
            if [[ -s "$master_pub_file" ]] && grep -q "BEGIN PUBLIC KEY\|BEGIN RSA PUBLIC KEY\|ssh-rsa" "$master_pub_file" 2>/dev/null; then
                downloaded=true
                log_info "✓ Master 公钥下载成功 (wget)"
            else
                rm -f "$master_pub_file"
                log_warn "下载的文件不是有效的公钥"
            fi
        fi
    fi
    
    if [[ "$downloaded" == "true" ]]; then
        chmod 644 "$master_pub_file"
        log_info "Master 公钥已保存到: $master_pub_file"
        
        # 验证公钥内容
        log_info "公钥信息:"
        head -2 "$master_pub_file" | sed 's/^/  /'
        log_info "  ... ($(wc -l < "$master_pub_file") 行)"
        
        return 0
    else
        log_warn "无法获取 Master 公钥，将在首次连接时自动获取"
        log_warn "（这需要 Master 的 auto_accept 设置为 true）"
        return 0  # 不视为失败，Salt 可以在首次连接时获取公钥
    fi
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
# 修复 Salt 依赖问题 (针对新版 Python 系统)
# 特别针对 Fedora 43 (Python 3.14+) 等新系统
# 优先使用 AppHub 中预下载的 Python 包
# ===========================================
fix_salt_dependencies() {
    log_info "检查并修复 Salt 依赖..."
    
    # 检测系统 Python 版本
    local sys_python=$(command -v python3 2>/dev/null)
    local sys_version=""
    local sys_major=0
    local sys_minor=0
    
    if [[ -n "$sys_python" ]]; then
        sys_version=$("$sys_python" -c "import sys; print(f'{sys.version_info.major}.{sys.version_info.minor}')" 2>/dev/null)
        sys_major=$("$sys_python" -c "import sys; print(sys.version_info.major)" 2>/dev/null)
        sys_minor=$("$sys_python" -c "import sys; print(sys.version_info.minor)" 2>/dev/null)
        log_info "系统 Python 版本: $sys_version"
    fi
    
    # 检查是否使用 relenv 包 (官方独立 Python 环境)
    local salt_python="/opt/saltstack/salt/bin/python3"
    local salt_pip="/opt/saltstack/salt/bin/pip3"
    local use_relenv=false
    
    if [[ -x "$salt_python" ]]; then
        log_info "检测到 SaltStack relenv 环境: $salt_python"
        use_relenv=true
        
        # 确保 relenv Python 有必要的依赖
        if ! "$salt_python" -c "import looseversion" 2>/dev/null; then
            log_info "安装缺失依赖: looseversion (relenv 环境)"
            install_looseversion_from_apphub "$salt_pip" "$salt_python" || \
            "$salt_pip" install looseversion 2>/dev/null || true
        fi
    fi
    
    # Python 3.12+ 移除了 distutils.version.LooseVersion，需要安装 looseversion
    # 特别是 Fedora 43 (Python 3.14) 等新系统
    if [[ "$sys_major" -ge 3 ]] && [[ "$sys_minor" -ge 12 ]]; then
        log_info "Python $sys_version >= 3.12 需要安装 looseversion 模块..."
        
        # 确保 pip 可用
        if ! command -v pip3 >/dev/null 2>&1; then
            log_info "安装 pip..."
            if command -v dnf >/dev/null 2>&1; then
                dnf install -y python3-pip 2>/dev/null || true
            elif command -v apt-get >/dev/null 2>&1; then
                apt-get install -y python3-pip 2>/dev/null || true
            fi
        fi
        
        # 方法1: 优先从 AppHub 安装 (离线/内网环境友好)
        log_info "尝试从 AppHub 安装 looseversion..."
        if install_looseversion_from_apphub "pip3" "$sys_python"; then
            log_info "✓ 从 AppHub 安装 looseversion 成功"
        else
            # 方法2: 使用 pip3 安装到系统
            log_info "尝试使用 pip3 安装 looseversion..."
            pip3 install looseversion --break-system-packages 2>/dev/null || \
            pip3 install looseversion 2>/dev/null || true
            
            # 方法3: 使用 python3 -m pip 安装
            if ! "$sys_python" -c "import looseversion" 2>/dev/null; then
                log_info "尝试使用 python3 -m pip 安装 looseversion..."
                "$sys_python" -m pip install looseversion --break-system-packages 2>/dev/null || \
                "$sys_python" -m pip install looseversion 2>/dev/null || true
            fi
            
            # 方法4: 检查是否有系统包可用
            if ! "$sys_python" -c "import looseversion" 2>/dev/null; then
                log_info "尝试从系统包安装 python3-looseversion..."
                if command -v dnf >/dev/null 2>&1; then
                    dnf install -y python3-looseversion 2>/dev/null || true
                fi
            fi
        fi
        
        # 验证安装
        if "$sys_python" -c "import looseversion" 2>/dev/null; then
            log_info "✓ looseversion 模块安装成功"
        else
            log_warn "looseversion 安装失败，Salt Minion 可能无法启动"
            log_warn "请手动执行: pip3 install looseversion --break-system-packages"
        fi
    fi
    
    # 如果使用系统 Python (非 relenv)，确保 Salt 能找到所有依赖
    if [[ "$use_relenv" != "true" ]]; then
        # 检查 Salt 安装位置
        local salt_site_packages=""
        if [[ -d "/usr/lib/python${sys_version}/site-packages/salt" ]]; then
            salt_site_packages="/usr/lib/python${sys_version}/site-packages"
            log_info "Salt 安装在系统 site-packages: $salt_site_packages"
        elif [[ -d "/usr/lib64/python${sys_version}/site-packages/salt" ]]; then
            salt_site_packages="/usr/lib64/python${sys_version}/site-packages"
            log_info "Salt 安装在系统 site-packages: $salt_site_packages"
        fi
        
        # 确保 looseversion 对 Salt 可见
        if [[ -n "$salt_site_packages" ]]; then
            # 查找 looseversion 安装位置
            local looseversion_path=$("$sys_python" -c "import looseversion; print(looseversion.__file__)" 2>/dev/null | xargs dirname 2>/dev/null || true)
            
            if [[ -n "$looseversion_path" ]] && [[ ! "$looseversion_path" == "$salt_site_packages"* ]]; then
                log_info "looseversion 路径: $looseversion_path"
                # 如果 looseversion 不在 Salt 能访问的路径，创建软链接
                if [[ -d "$looseversion_path" ]] && [[ ! -e "$salt_site_packages/looseversion" ]]; then
                    log_info "创建 looseversion 软链接到 Salt site-packages..."
                    ln -sf "$looseversion_path" "$salt_site_packages/looseversion" 2>/dev/null || true
                fi
            fi
        fi
    fi
}

# ===========================================
# 从 AppHub 安装 looseversion
# 优先使用预下载的 Python 包，避免网络访问
# ===========================================
install_looseversion_from_apphub() {
    local pip_cmd="${1:-pip3}"
    local python_cmd="${2:-python3}"
    
    # AppHub Python 依赖包 URL
    local deps_url="${APPHUB_URL}/pkgs/python-deps"
    
    log_info "尝试从 AppHub 下载 Python 依赖包..."
    log_info "AppHub URL: $deps_url"
    
    # 创建临时目录
    local tmp_dir=$(mktemp -d)
    trap "rm -rf $tmp_dir" RETURN
    
    cd "$tmp_dir"
    
    # 尝试下载所有可用的 wheel 文件
    local downloaded=false
    
    # 首先获取目录列表来确定实际的文件名
    # 支持的文件名模式：looseversion-*.whl
    local wheel_patterns=(
        "looseversion-1.3.0-py2.py3-none-any.whl"
        "looseversion-1.3.0-py3-none-any.whl"
        "looseversion-1.3.0.tar.gz"
    )
    
    for whl in "${wheel_patterns[@]}"; do
        if command -v wget >/dev/null 2>&1; then
            if wget -q --timeout=10 "${deps_url}/${whl}" 2>/dev/null; then
                log_info "✓ 下载成功: $whl"
                downloaded=true
                break
            fi
        elif command -v curl >/dev/null 2>&1; then
            if curl -fsSL --connect-timeout 10 -o "$whl" "${deps_url}/${whl}" 2>/dev/null && [[ -s "$whl" ]]; then
                log_info "✓ 下载成功: $whl"
                downloaded=true
                break
            fi
        fi
    done
    
    if [[ "$downloaded" != "true" ]]; then
        log_warn "无法从 AppHub 下载 Python 依赖包"
        return 1
    fi
    
    # 安装下载的包
    local pkg_file=$(ls -1 looseversion-*.whl looseversion-*.tar.gz 2>/dev/null | head -1)
    
    if [[ -n "$pkg_file" ]] && [[ -f "$pkg_file" ]]; then
        log_info "安装 $pkg_file..."
        
        # 尝试多种安装方式
        if $pip_cmd install "$pkg_file" --break-system-packages 2>/dev/null; then
            log_info "✓ 使用 $pip_cmd 安装成功"
            return 0
        elif $pip_cmd install "$pkg_file" 2>/dev/null; then
            log_info "✓ 使用 $pip_cmd 安装成功"
            return 0
        elif $python_cmd -m pip install "$pkg_file" --break-system-packages 2>/dev/null; then
            log_info "✓ 使用 $python_cmd -m pip 安装成功"
            return 0
        elif $python_cmd -m pip install "$pkg_file" 2>/dev/null; then
            log_info "✓ 使用 $python_cmd -m pip 安装成功"
            return 0
        fi
    fi
    
    log_warn "从 AppHub 安装失败"
    return 1
}

# ===========================================
# 修复 Systemd 服务配置
# ===========================================
fix_systemd_service() {
    log_info "检查 Systemd 服务配置..."
    
    local service_file="/etc/systemd/system/salt-minion.service"
    local lib_service_file="/usr/lib/systemd/system/salt-minion.service"
    
    # 优先使用 /etc/systemd/system 下的配置
    if [[ ! -f "$service_file" ]] && [[ -f "$lib_service_file" ]]; then
        cp "$lib_service_file" "$service_file"
    fi
    
    # 检查是否使用 relenv 包
    local salt_python="/opt/saltstack/salt/bin/python3"
    local salt_minion_bin="/opt/saltstack/salt/salt-minion"
    
    if [[ -x "$salt_python" ]] && [[ -x "$salt_minion_bin" ]]; then
        log_info "配置使用 SaltStack relenv Python 环境..."
        
        # 创建/更新 systemd 服务文件
        cat > "$service_file" <<'EOF'
[Unit]
Description=The Salt Minion
Documentation=man:salt-minion(1) file:///usr/share/doc/salt/html/contents.html https://docs.saltproject.io
After=network.target

[Service]
Type=notify
NotifyAccess=all
LimitNOFILE=8192

# 使用 relenv 环境的 Python 和 Salt
ExecStart=/opt/saltstack/salt/salt-minion

KillMode=process
Restart=on-failure
RestartSec=3

# 环境变量确保使用正确的 Python
Environment="PATH=/opt/saltstack/salt/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"

[Install]
WantedBy=multi-user.target
EOF
        log_info "已更新 systemd 服务配置使用 relenv 环境"
    else
        # 检查系统安装的 salt-minion
        local sys_salt_minion=$(command -v salt-minion 2>/dev/null)
        if [[ -n "$sys_salt_minion" ]]; then
            log_info "使用系统安装的 salt-minion: $sys_salt_minion"
            
            # 检查现有服务文件
            if [[ -f "$service_file" ]] || [[ -f "$lib_service_file" ]]; then
                log_info "Systemd 服务文件已存在"
            else
                # 创建基本服务文件
                cat > "$service_file" <<EOF
[Unit]
Description=The Salt Minion
After=network.target

[Service]
Type=notify
NotifyAccess=all
ExecStart=$sys_salt_minion
KillMode=process
Restart=on-failure
RestartSec=3

[Install]
WantedBy=multi-user.target
EOF
                log_info "已创建 systemd 服务配置"
            fi
        fi
    fi
    
    systemctl daemon-reload
}

# ===========================================
# 启动服务
# ===========================================
start_service() {
    log_info "启动 Salt Minion 服务..."
    
    # 先修复依赖和服务配置
    fix_salt_dependencies
    fix_systemd_service
    
    systemctl daemon-reload || true
    systemctl enable salt-minion || true
    systemctl restart salt-minion || true
    
    sleep 3
    
    if systemctl is-active --quiet salt-minion; then
        log_info "Salt Minion 服务已启动"
        return 0
    else
        log_warn "Salt Minion 服务启动失败，尝试诊断..."
        
        # 诊断信息
        log_info "检查服务状态..."
        systemctl status salt-minion --no-pager -l 2>&1 | head -20 || true
        
        log_info "检查日志..."
        journalctl -u salt-minion --no-pager -n 20 2>&1 || true
        
        # 尝试手动启动获取错误
        log_info "尝试手动启动获取详细错误..."
        if [[ -x "/opt/saltstack/salt/salt-minion" ]]; then
            timeout 5 /opt/saltstack/salt/salt-minion --log-level=debug 2>&1 | head -30 || true
        elif command -v salt-minion >/dev/null 2>&1; then
            timeout 5 salt-minion --log-level=debug 2>&1 | head -30 || true
        fi
        
        return 1
    fi
}

# ===========================================
# 主函数
# ===========================================
main() {
    parse_args "$@"
    
    # 验证参数
    if [[ -z "$SALT_MASTER" ]] || [[ "$SALT_MASTER" == "192.168.247.121" ]]; then
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
    if [[ "$USE_OFFICIAL" != "true" ]] && [[ -n "$APPHUB_URL" ]] && [[ "$APPHUB_URL" != *"192.168.247.121"* ]]; then
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
    
    # 预同步 Master 公钥（可选，提高首次连接成功率）
    sync_master_pubkey
    
    # 启动服务
    if ! start_service; then
        log_error "Salt Minion 服务启动失败"
        exit 1
    fi
    
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
