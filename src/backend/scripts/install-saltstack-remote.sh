#!/bin/bash
# =============================================================================
# SaltStack Minion 远程安装脚本
# 通过 SSH 在目标主机上安装 SaltStack Minion
# =============================================================================
# 用法: ./install-saltstack-remote.sh [OPTIONS] <host>
# 示例: ./install-saltstack-remote.sh -u ubuntu -i ~/.ssh/id_rsa --master 192.168.1.10 192.168.1.100
# =============================================================================

set -eo pipefail

# 默认配置
SSH_USER="root"
SSH_PORT="22"
SSH_KEY=""
SSH_PASSWORD=""
SALT_MASTER="${SALT_MASTER:-salt-master}"
SALT_VERSION="${SALT_VERSION:-3007.1}"
MINION_ID=""
APPHUB_URL="${APPHUB_URL:-}"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}[INFO]${NC} $*"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }

usage() {
    cat <<EOF
SaltStack Minion 远程安装脚本

用法: $0 [OPTIONS] <host>

选项:
    -h, --help              显示帮助信息
    -u, --user USER         SSH 用户名 (默认: root)
    -p, --port PORT         SSH 端口 (默认: 22)
    -i, --identity KEY_PATH SSH 私钥路径
    -P, --password PASSWORD SSH 密码 (不推荐用于生产环境)
    --master MASTER         SaltStack Master 地址 (默认: salt-master)
    --minion-id MINION_ID   Minion ID (默认: 目标主机名)
    --apphub-url URL        AppHub URL (默认: 使用 APPHUB_URL 环境变量)
    --salt-version VERSION  SaltStack 版本 (默认: 3007.1)

示例:
    # 使用密钥认证安装
    $0 -u ubuntu -i ~/.ssh/id_rsa --master 192.168.1.10 192.168.1.100

    # 使用密码认证安装 (需要 sshpass)
    $0 -u centos -P password123 --minion-id worker01 192.168.1.101

    # 指定 AppHub URL
    $0 --apphub-url http://192.168.1.10:53434 --master 192.168.1.10 192.168.1.100
EOF
    exit 0
}

# 解析命令行参数
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                usage
                ;;
            -u|--user)
                SSH_USER="$2"
                shift 2
                ;;
            -p|--port)
                SSH_PORT="$2"
                shift 2
                ;;
            -i|--identity)
                SSH_KEY="$2"
                shift 2
                ;;
            -P|--password)
                SSH_PASSWORD="$2"
                shift 2
                ;;
            --master)
                SALT_MASTER="$2"
                shift 2
                ;;
            --minion-id)
                MINION_ID="$2"
                shift 2
                ;;
            --apphub-url)
                APPHUB_URL="$2"
                shift 2
                ;;
            --salt-version)
                SALT_VERSION="$2"
                shift 2
                ;;
            -*)
                log_error "未知选项: $1"
                usage
                ;;
            *)
                TARGET_HOST="$1"
                shift
                ;;
        esac
    done

    if [[ -z "$TARGET_HOST" ]]; then
        log_error "未指定目标主机"
        usage
    fi
}

# 构建 SSH 命令
build_ssh_cmd() {
    local ssh_opts="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=30"
    
    if [[ -n "$SSH_KEY" ]]; then
        ssh_opts="$ssh_opts -i $SSH_KEY"
    fi
    
    ssh_opts="$ssh_opts -p $SSH_PORT"
    
    if [[ -n "$SSH_PASSWORD" ]]; then
        if ! command -v sshpass &>/dev/null; then
            log_error "使用密码认证需要安装 sshpass"
            exit 1
        fi
        echo "sshpass -p '$SSH_PASSWORD' ssh $ssh_opts"
    else
        echo "ssh $ssh_opts"
    fi
}

# 构建 SCP 命令
build_scp_cmd() {
    local scp_opts="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=30"
    
    if [[ -n "$SSH_KEY" ]]; then
        scp_opts="$scp_opts -i $SSH_KEY"
    fi
    
    scp_opts="$scp_opts -P $SSH_PORT"
    
    if [[ -n "$SSH_PASSWORD" ]]; then
        echo "sshpass -p '$SSH_PASSWORD' scp $scp_opts"
    else
        echo "scp $scp_opts"
    fi
}

# 检测远程主机操作系统
detect_remote_os() {
    local ssh_cmd="$1"
    local target="$2"
    
    log_info "检测远程主机操作系统..."
    
    local os_info
    os_info=$($ssh_cmd $SSH_USER@$target "cat /etc/os-release 2>/dev/null || cat /etc/redhat-release 2>/dev/null || echo 'unknown'")
    
    if echo "$os_info" | grep -qiE "ubuntu|debian"; then
        echo "debian"
    elif echo "$os_info" | grep -qiE "centos|rhel|rocky|fedora|almalinux"; then
        echo "rhel"
    else
        echo "unknown"
    fi
}

# 生成远程安装脚本
generate_install_script() {
    local os_type="$1"
    local minion_id="$2"
    
    cat <<'SCRIPT_EOF'
#!/bin/bash
set -eo pipefail

SALT_MASTER="__SALT_MASTER__"
SALT_VERSION="__SALT_VERSION__"
MINION_ID="__MINION_ID__"
APPHUB_URL="__APPHUB_URL__"

echo "=========================================="
echo "安装 SaltStack Minion"
echo "Master: ${SALT_MASTER}"
echo "Version: ${SALT_VERSION}"
echo "Minion ID: ${MINION_ID}"
echo "=========================================="

# 如果未指定 Minion ID，使用主机名
if [ -z "$MINION_ID" ]; then
    MINION_ID=$(hostname)
fi

# 检测操作系统
if command -v apt-get >/dev/null 2>&1; then
    OS_TYPE="debian"
elif command -v dnf >/dev/null 2>&1 || command -v yum >/dev/null 2>&1; then
    OS_TYPE="rhel"
else
    echo "❌ 不支持的操作系统"
    exit 1
fi

install_from_apphub() {
    echo "从 AppHub 安装 SaltStack..."
    
    if [ "$OS_TYPE" = "debian" ]; then
        export DEBIAN_FRONTEND=noninteractive
        
        # 配置 AppHub 源
        echo "deb [trusted=yes] ${APPHUB_URL}/pkgs/saltstack-deb ./" > /etc/apt/sources.list.d/ai-infra-salt.list
        apt-get update
        apt-get install -y --no-install-recommends salt-minion
        
    elif [ "$OS_TYPE" = "rhel" ]; then
        PKG_MGR="dnf"
        command -v dnf >/dev/null 2>&1 || PKG_MGR="yum"
        
        # 配置 AppHub 源
        cat > /etc/yum.repos.d/ai-infra-salt.repo <<EOF
[ai-infra-salt]
name=AI Infra Salt RPMs
baseurl=${APPHUB_URL}/pkgs/saltstack-rpm
enabled=1
gpgcheck=0
priority=1
EOF
        ${PKG_MGR} clean all
        ${PKG_MGR} makecache
        ${PKG_MGR} install -y salt-minion
    fi
}

install_from_official() {
    echo "从官方源安装 SaltStack..."
    
    if [ "$OS_TYPE" = "debian" ]; then
        export DEBIAN_FRONTEND=noninteractive
        
        # 安装依赖
        apt-get update
        apt-get install -y curl gnupg2
        
        # 添加 Salt 官方源
        curl -fsSL https://repo.saltproject.io/salt/py3/debian/$(cat /etc/debian_version | cut -d. -f1)/amd64/SALT-PROJECT-GPG-PUBKEY-2023.pub | gpg --dearmor -o /usr/share/keyrings/salt-archive-keyring.gpg 2>/dev/null || true
        echo "deb [signed-by=/usr/share/keyrings/salt-archive-keyring.gpg] https://repo.saltproject.io/salt/py3/debian/$(cat /etc/debian_version | cut -d. -f1)/amd64/latest ./" > /etc/apt/sources.list.d/salt.list
        
        apt-get update
        apt-get install -y salt-minion
        
    elif [ "$OS_TYPE" = "rhel" ]; then
        PKG_MGR="dnf"
        command -v dnf >/dev/null 2>&1 || PKG_MGR="yum"
        
        # 添加 Salt 官方源
        rpm --import https://repo.saltproject.io/salt/py3/redhat/9/x86_64/SALT-PROJECT-GPG-PUBKEY-2023.pub 2>/dev/null || true
        
        cat > /etc/yum.repos.d/salt.repo <<EOF
[salt-latest]
name=Salt repo
baseurl=https://repo.saltproject.io/salt/py3/redhat/\$releasever/\$basearch/latest
enabled=1
gpgcheck=1
gpgkey=https://repo.saltproject.io/salt/py3/redhat/\$releasever/\$basearch/SALT-PROJECT-GPG-PUBKEY-2023.pub
EOF
        ${PKG_MGR} clean all
        ${PKG_MGR} install -y salt-minion
    fi
}

# 优先从 AppHub 安装，失败则回退到官方源
if [ -n "$APPHUB_URL" ]; then
    if install_from_apphub; then
        echo "✓ 从 AppHub 安装成功"
    else
        echo "AppHub 安装失败，尝试官方源..."
        install_from_official
    fi
else
    install_from_official
fi

# 验证安装
if ! command -v salt-minion >/dev/null 2>&1; then
    echo "❌ Salt Minion 安装失败"
    exit 1
fi

echo "✓ Salt Minion 安装成功: $(salt-minion --version)"

# 配置 Minion
echo "配置 Salt Minion..."
mkdir -p /etc/salt/minion.d

cat > /etc/salt/minion <<EOF
# SaltStack Minion 配置
# Auto-generated by install-saltstack-remote.sh

master: ${SALT_MASTER}
id: ${MINION_ID}

# 日志配置
log_level: info
log_file: /var/log/salt/minion

# 连接配置
retry_dns: 30
master_alive_interval: 30
master_tries: -1
acceptance_wait_time: 10
EOF

# 启动服务
echo "启动 Salt Minion 服务..."
systemctl daemon-reload || true
systemctl enable salt-minion || true
systemctl restart salt-minion || true

sleep 2

if systemctl is-active --quiet salt-minion; then
    echo "✓ Salt Minion 服务运行中"
else
    echo "⚠ Salt Minion 服务可能未正常启动"
    systemctl status salt-minion --no-pager || true
fi

echo ""
echo "=========================================="
echo "✓ SaltStack Minion 安装完成"
echo "  Master: ${SALT_MASTER}"
echo "  Minion ID: ${MINION_ID}"
echo ""
echo "后续步骤:"
echo "1. 在 Salt Master 上接受密钥:"
echo "   salt-key -a ${MINION_ID}"
echo "2. 测试连接:"
echo "   salt '${MINION_ID}' test.ping"
echo "=========================================="
SCRIPT_EOF
}

# 主函数
main() {
    parse_args "$@"
    
    log_info "开始安装 SaltStack Minion"
    log_info "目标主机: $TARGET_HOST"
    log_info "SSH 用户: $SSH_USER"
    log_info "Salt Master: $SALT_MASTER"
    
    # 构建 SSH 命令
    SSH_CMD=$(build_ssh_cmd)
    SCP_CMD=$(build_scp_cmd)
    
    # 测试 SSH 连接
    log_info "测试 SSH 连接..."
    if ! eval "$SSH_CMD $SSH_USER@$TARGET_HOST 'echo ok'" &>/dev/null; then
        log_error "无法连接到 $TARGET_HOST"
        exit 1
    fi
    log_info "SSH 连接成功"
    
    # 检测操作系统
    OS_TYPE=$(detect_remote_os "$SSH_CMD" "$TARGET_HOST")
    log_info "检测到操作系统类型: $OS_TYPE"
    
    if [[ "$OS_TYPE" == "unknown" ]]; then
        log_error "不支持的操作系统"
        exit 1
    fi
    
    # 设置 Minion ID
    if [[ -z "$MINION_ID" ]]; then
        MINION_ID=$(eval "$SSH_CMD $SSH_USER@$TARGET_HOST 'hostname'" 2>/dev/null)
        log_info "使用主机名作为 Minion ID: $MINION_ID"
    fi
    
    # 生成安装脚本
    log_info "生成安装脚本..."
    local tmp_script="/tmp/install-salt-minion-$$.sh"
    generate_install_script "$OS_TYPE" "$MINION_ID" > "$tmp_script"
    
    # 替换变量
    sed -i.bak "s|__SALT_MASTER__|$SALT_MASTER|g" "$tmp_script"
    sed -i.bak "s|__SALT_VERSION__|$SALT_VERSION|g" "$tmp_script"
    sed -i.bak "s|__MINION_ID__|$MINION_ID|g" "$tmp_script"
    sed -i.bak "s|__APPHUB_URL__|$APPHUB_URL|g" "$tmp_script"
    rm -f "$tmp_script.bak"
    
    # 上传脚本
    log_info "上传安装脚本到目标主机..."
    eval "$SCP_CMD $tmp_script $SSH_USER@$TARGET_HOST:/tmp/install-salt-minion.sh"
    
    # 执行安装
    log_info "执行安装脚本..."
    eval "$SSH_CMD $SSH_USER@$TARGET_HOST 'chmod +x /tmp/install-salt-minion.sh && sudo /tmp/install-salt-minion.sh'"
    
    # 清理
    rm -f "$tmp_script"
    eval "$SSH_CMD $SSH_USER@$TARGET_HOST 'rm -f /tmp/install-salt-minion.sh'" || true
    
    log_info "SaltStack Minion 安装完成: $TARGET_HOST"
}

main "$@"
