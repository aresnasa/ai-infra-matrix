#!/bin/bash
# SLURM节点初始化脚本
# 用途：通过SSH远程初始化SLURM计算节点
# 用法：./init-slurm-node.sh <node_hostname>

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 配置参数（可通过环境变量覆盖）
SLURM_MASTER_HOST="${SLURM_MASTER_HOST:-ai-infra-slurm-master}"
SLURM_MASTER_PORT="${SLURM_MASTER_PORT:-6817}"
MUNGE_KEY_PATH="${MUNGE_KEY_PATH:-/etc/munge/munge.key}"
SLURM_USER="${SLURM_USER:-slurm}"
MUNGE_USER="${MUNGE_USER:-munge}"

# 函数：检查命令是否存在
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# 函数：检测操作系统类型
detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        echo "$ID"
    elif [ -f /etc/redhat-release ]; then
        echo "rhel"
    else
        echo "unknown"
    fi
}

# 函数：安装必要的包
install_packages() {
    local os_type=$(detect_os)
    
    echo "检测到操作系统: $os_type"
    
    case "$os_type" in
        ubuntu|debian)
            echo "更新包索引..."
            apt-get update -qq
            echo "安装SLURM和Munge..."
            apt-get install -y slurmd munge libmunge-dev
            ;;
        centos|rhel|rocky|almalinux)
            echo "安装SLURM和Munge..."
            yum install -y epel-release
            yum install -y slurm slurm-slurmd munge munge-libs
            ;;
        alpine)
            echo "安装SLURM和Munge..."
            apk add --no-cache slurm munge
            ;;
        *)
            echo "错误：不支持的操作系统 $os_type"
            return 1
            ;;
    esac
    
    echo "包安装完成"
}

# 函数：创建必要的目录和设置权限
setup_directories() {
    echo "创建SLURM和Munge目录..."
    
    # Munge目录
    mkdir -p /etc/munge
    mkdir -p /var/log/munge
    mkdir -p /var/lib/munge
    mkdir -p /run/munge
    
    # SLURM目录
    mkdir -p /etc/slurm
    mkdir -p /var/log/slurm
    mkdir -p /var/spool/slurmd
    mkdir -p /var/run/slurm
    
    # 创建用户（如果不存在）
    if ! id -u $MUNGE_USER >/dev/null 2>&1; then
        echo "创建munge用户..."
        case $(detect_os) in
            alpine)
                adduser -D -s /bin/false $MUNGE_USER
                ;;
            *)
                useradd -r -s /bin/false $MUNGE_USER || true
                ;;
        esac
    fi
    
    if ! id -u $SLURM_USER >/dev/null 2>&1; then
        echo "创建slurm用户..."
        case $(detect_os) in
            alpine)
                adduser -D -s /bin/false $SLURM_USER
                ;;
            *)
                useradd -r -s /bin/false $SLURM_USER || true
                ;;
        esac
    fi
    
    # 设置目录权限
    chown -R $MUNGE_USER:$MUNGE_USER /etc/munge /var/log/munge /var/lib/munge /run/munge
    chmod 700 /etc/munge
    chmod 711 /var/lib/munge
    chmod 755 /var/log/munge /run/munge
    
    chown -R $SLURM_USER:$SLURM_USER /var/log/slurm /var/spool/slurmd /var/run/slurm
    chmod 755 /var/log/slurm /var/spool/slurmd /var/run/slurm
    
    echo "目录设置完成"
}

# 函数：配置Munge密钥
configure_munge_key() {
    local key_content="$1"
    
    echo "配置Munge密钥..."
    
    if [ -z "$key_content" ]; then
        echo "错误：Munge密钥内容为空"
        return 1
    fi
    
    # 写入密钥文件
    echo "$key_content" | base64 -d > "$MUNGE_KEY_PATH"
    
    # 设置权限
    chown $MUNGE_USER:$MUNGE_USER "$MUNGE_KEY_PATH"
    chmod 400 "$MUNGE_KEY_PATH"
    
    echo "Munge密钥配置完成"
}

# 函数：配置SLURM配置文件
configure_slurm_conf() {
    local config_content="$1"
    
    echo "配置SLURM配置文件..."
    
    if [ -z "$config_content" ]; then
        echo "错误：SLURM配置内容为空"
        return 1
    fi
    
    # 写入配置文件
    echo "$config_content" > /etc/slurm/slurm.conf
    
    # 设置权限
    chown root:root /etc/slurm/slurm.conf
    chmod 644 /etc/slurm/slurm.conf
    
    echo "SLURM配置文件设置完成"
}

# 函数：配置cgroup（可选，用于资源隔离）
configure_cgroup() {
    echo "配置cgroup..."
    
    cat > /etc/slurm/cgroup.conf <<'EOF'
CgroupAutomount=yes
ConstrainCores=yes
ConstrainRAMSpace=yes
ConstrainSwapSpace=yes
EOF
    
    chown root:root /etc/slurm/cgroup.conf
    chmod 644 /etc/slurm/cgroup.conf
    
    echo "cgroup配置完成"
}

# 函数：启动Munge服务
start_munge() {
    echo "启动Munge服务..."
    
    # 停止现有服务（如果运行中）
    if command_exists systemctl; then
        systemctl stop munge 2>/dev/null || true
        systemctl enable munge
        systemctl start munge
        sleep 2
        systemctl status munge --no-pager || true
    elif command_exists rc-service; then
        # Alpine Linux
        rc-service munge stop 2>/dev/null || true
        rc-update add munge default
        rc-service munge start
        sleep 2
        rc-service munge status || true
    else
        # 直接启动
        munged -f &
        sleep 2
    fi
    
    # 验证Munge
    if munge -n | unmunge >/dev/null 2>&1; then
        echo "Munge服务运行正常"
    else
        echo "警告：Munge验证失败"
        return 1
    fi
}

# 函数：启动SLURMD服务
start_slurmd() {
    echo "启动SLURMD服务..."
    
    # 停止现有服务（如果运行中）
    if command_exists systemctl; then
        systemctl stop slurmd 2>/dev/null || true
        systemctl enable slurmd
        systemctl start slurmd
        sleep 2
        systemctl status slurmd --no-pager || true
    elif command_exists rc-service; then
        # Alpine Linux
        rc-service slurmd stop 2>/dev/null || true
        rc-update add slurmd default
        rc-service slurmd start
        sleep 2
        rc-service slurmd status || true
    else
        # 直接启动
        slurmd -D &
        sleep 2
    fi
    
    echo "SLURMD服务已启动"
}

# 函数：验证节点状态
verify_node() {
    local node_name="$1"
    
    echo "验证节点状态..."
    
    # 检查进程
    if pgrep -x munged >/dev/null; then
        echo "✓ Munge守护进程运行中"
    else
        echo "✗ Munge守护进程未运行"
        return 1
    fi
    
    if pgrep -x slurmd >/dev/null; then
        echo "✓ SLURMD守护进程运行中"
    else
        echo "✗ SLURMD守护进程未运行"
        return 1
    fi
    
    # 检查日志
    if [ -f /var/log/slurm/slurmd.log ]; then
        echo "最近的SLURMD日志："
        tail -n 5 /var/log/slurm/slurmd.log || true
    fi
    
    echo "节点验证完成"
}

# 函数：显示使用说明
usage() {
    cat <<EOF
用法: $0 [选项]

选项:
    --install-packages      安装SLURM和Munge包
    --setup-dirs            创建目录并设置权限
    --munge-key <content>   配置Munge密钥（base64编码）
    --slurm-conf <content>  配置SLURM配置文件
    --start-munge           启动Munge服务
    --start-slurmd          启动SLURMD服务
    --verify <node_name>    验证节点状态
    --full-init             执行完整初始化（所有步骤）
    -h, --help              显示此帮助信息

环境变量:
    SLURM_MASTER_HOST       SLURM控制器主机名（默认: ai-infra-slurm-master）
    SLURM_MASTER_PORT       SLURM控制器端口（默认: 6817）
    MUNGE_KEY_PATH          Munge密钥文件路径（默认: /etc/munge/munge.key）

示例:
    # 完整初始化
    $0 --full-init --munge-key "\$(base64 /etc/munge/munge.key)" --slurm-conf "\$(cat /etc/slurm/slurm.conf)"
    
    # 仅安装包
    $0 --install-packages
    
    # 仅启动服务
    $0 --start-munge --start-slurmd
EOF
}

# 主函数
main() {
    local do_install=false
    local do_setup_dirs=false
    local do_start_munge=false
    local do_start_slurmd=false
    local do_verify=false
    local do_full_init=false
    local munge_key_content=""
    local slurm_conf_content=""
    local node_name=""
    
    # 解析参数
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --install-packages)
                do_install=true
                shift
                ;;
            --setup-dirs)
                do_setup_dirs=true
                shift
                ;;
            --munge-key)
                munge_key_content="$2"
                shift 2
                ;;
            --slurm-conf)
                slurm_conf_content="$2"
                shift 2
                ;;
            --start-munge)
                do_start_munge=true
                shift
                ;;
            --start-slurmd)
                do_start_slurmd=true
                shift
                ;;
            --verify)
                do_verify=true
                node_name="$2"
                shift 2
                ;;
            --full-init)
                do_full_init=true
                shift
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                echo "错误：未知选项 $1"
                usage
                exit 1
                ;;
        esac
    done
    
    # 完整初始化模式
    if [ "$do_full_init" = true ]; then
        do_install=true
        do_setup_dirs=true
        do_start_munge=true
        do_start_slurmd=true
    fi
    
    # 执行步骤
    if [ "$do_install" = true ]; then
        install_packages
    fi
    
    if [ "$do_setup_dirs" = true ]; then
        setup_directories
    fi
    
    if [ -n "$munge_key_content" ]; then
        configure_munge_key "$munge_key_content"
    fi
    
    if [ -n "$slurm_conf_content" ]; then
        configure_slurm_conf "$slurm_conf_content"
        configure_cgroup
    fi
    
    if [ "$do_start_munge" = true ]; then
        start_munge
    fi
    
    if [ "$do_start_slurmd" = true ]; then
        start_slurmd
    fi
    
    if [ "$do_verify" = true ]; then
        verify_node "$node_name"
    fi
    
    echo "初始化完成！"
}

# 执行主函数
main "$@"
