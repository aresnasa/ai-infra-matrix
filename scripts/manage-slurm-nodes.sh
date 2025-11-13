#!/bin/bash
# SLURM节点管理脚本
# 用途：管理SLURM集群节点，包括初始化、密钥同步等
# 用法：./manage-slurm-nodes.sh <command> [options]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_URL="${BACKEND_URL:-http://localhost:8082}"
CLUSTER_ID="${CLUSTER_ID:-1}"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 函数：检查节点状态
check_nodes_status() {
    local nodes=("$@")
    
    info "检查节点状态..."
    
    for node in "${nodes[@]}"; do
        echo ""
        echo "=== $node ==="
        
        if docker exec "$node" bash -c "ps aux | egrep -v grep | egrep 'munged|slurmd'" 2>/dev/null; then
            info "$node: 服务运行中"
        else
            warn "$node: 服务未运行"
        fi
    done
}

# 函数：获取Munge密钥
get_munge_key() {
    info "从SLURM Master获取Munge密钥..."
    
    if [ -f /etc/munge/munge.key ]; then
        base64 /etc/munge/munge.key
    elif docker exec ai-infra-slurm-master cat /etc/munge/munge.key 2>/dev/null | base64; then
        info "从容器获取密钥成功"
    else
        error "无法获取Munge密钥"
        return 1
    fi
}

# 函数：同步Munge密钥到节点
sync_munge_key_to_node() {
    local node=$1
    local munge_key_base64=$2
    
    info "同步Munge密钥到 $node..."
    
    docker exec "$node" bash -c "
        mkdir -p /etc/munge /var/log/munge /var/lib/munge /run/munge
        echo '$munge_key_base64' | base64 -d > /etc/munge/munge.key
        chown -R munge:munge /etc/munge /var/log/munge /var/lib/munge /run/munge 2>/dev/null || chown -R 101:101 /etc/munge /var/log/munge /var/lib/munge /run/munge
        chmod 700 /etc/munge
        chmod 400 /etc/munge/munge.key
    "
    
    if [ $? -eq 0 ]; then
        info "$node: Munge密钥同步成功"
    else
        error "$node: Munge密钥同步失败"
        return 1
    fi
}

# 函数：同步SLURM配置到节点
sync_slurm_conf_to_node() {
    local node=$1
    
    info "同步SLURM配置到 $node..."
    
    # 从master获取配置
    local slurm_conf=$(docker exec ai-infra-slurm-master cat /etc/slurm/slurm.conf 2>/dev/null)
    
    if [ -z "$slurm_conf" ]; then
        error "无法从master获取SLURM配置"
        return 1
    fi
    
    docker exec "$node" bash -c "
        mkdir -p /etc/slurm /var/log/slurm /var/spool/slurmd /var/run/slurm
        cat > /etc/slurm/slurm.conf <<'SLURMEOF'
$slurm_conf
SLURMEOF
        chown -R slurm:slurm /etc/slurm /var/log/slurm /var/spool/slurmd /var/run/slurm 2>/dev/null || chown -R 64030:64030 /etc/slurm /var/log/slurm /var/spool/slurmd /var/run/slurm
        chmod 644 /etc/slurm/slurm.conf
    "
    
    if [ $? -eq 0 ]; then
        info "$node: SLURM配置同步成功"
    else
        error "$node: SLURM配置同步失败"
        return 1
    fi
}

# 函数：启动Munge服务
start_munge_on_node() {
    local node=$1
    
    info "启动 $node 的Munge服务..."
    
    docker exec "$node" bash -c "
        # 尝试systemctl
        if command -v systemctl >/dev/null 2>&1; then
            systemctl stop munge 2>/dev/null || true
            systemctl start munge || munged
        # 尝试rc-service (Alpine)
        elif command -v rc-service >/dev/null 2>&1; then
            rc-service munge stop 2>/dev/null || true
            rc-service munge start
        # 直接启动
        else
            pkill munged 2>/dev/null || true
            sleep 1
            munged
        fi
        
        sleep 2
        
        # 验证
        if ps aux | grep -v grep | grep munged >/dev/null; then
            echo 'Munge运行中'
        else
            echo 'Munge启动失败' >&2
            exit 1
        fi
    "
    
    if [ $? -eq 0 ]; then
        info "$node: Munge服务启动成功"
    else
        error "$node: Munge服务启动失败"
        return 1
    fi
}

# 函数：启动SLURMD服务
start_slurmd_on_node() {
    local node=$1
    
    info "启动 $node 的SLURMD服务..."
    
    docker exec "$node" bash -c "
        # 尝试systemctl
        if command -v systemctl >/dev/null 2>&1; then
            systemctl stop slurmd 2>/dev/null || true
            systemctl start slurmd || slurmd -D &
        # 尝试rc-service (Alpine)
        elif command -v rc-service >/dev/null 2>&1; then
            rc-service slurmd stop 2>/dev/null || true
            rc-service slurmd start
        # 直接启动
        else
            pkill slurmd 2>/dev/null || true
            sleep 1
            slurmd -D &
        fi
        
        sleep 2
        
        # 验证
        if ps aux | grep -v grep | grep slurmd >/dev/null; then
            echo 'SLURMD运行中'
        else
            echo 'SLURMD启动失败' >&2
            exit 1
        fi
    "
    
    if [ $? -eq 0 ]; then
        info "$node: SLURMD服务启动成功"
    else
        error "$node: SLURMD服务启动失败"
        return 1
    fi
}

# 函数：完整初始化节点
init_node() {
    local node=$1
    local munge_key_base64=$2
    
    info "开始初始化节点: $node"
    
    sync_munge_key_to_node "$node" "$munge_key_base64" || return 1
    sync_slurm_conf_to_node "$node" || return 1
    start_munge_on_node "$node" || return 1
    start_slurmd_on_node "$node" || return 1
    
    info "节点 $node 初始化完成"
}

# 函数：批量初始化节点
batch_init_nodes() {
    local nodes=("$@")
    
    # 获取Munge密钥
    local munge_key_base64=$(get_munge_key)
    
    if [ -z "$munge_key_base64" ]; then
        error "无法获取Munge密钥"
        return 1
    fi
    
    info "开始批量初始化 ${#nodes[@]} 个节点..."
    
    for node in "${nodes[@]}"; do
        echo ""
        info "处理节点: $node"
        
        if init_node "$node" "$munge_key_base64"; then
            info "✓ $node 初始化成功"
        else
            error "✗ $node 初始化失败"
        fi
    done
    
    echo ""
    info "批量初始化完成"
}

# 函数：同步SSH密钥到节点
sync_ssh_keys() {
    local nodes=("$@")
    
    info "同步SSH密钥到节点..."
    
    # 从master获取公钥
    local public_key=$(docker exec ai-infra-slurm-master cat /root/.ssh/id_rsa.pub 2>/dev/null)
    
    if [ -z "$public_key" ]; then
        warn "Master节点没有SSH公钥，正在生成..."
        docker exec ai-infra-slurm-master bash -c "
            mkdir -p /root/.ssh
            if [ ! -f /root/.ssh/id_rsa ]; then
                ssh-keygen -t rsa -b 2048 -f /root/.ssh/id_rsa -N ''
            fi
        "
        public_key=$(docker exec ai-infra-slurm-master cat /root/.ssh/id_rsa.pub)
    fi
    
    for node in "${nodes[@]}"; do
        info "同步SSH密钥到 $node..."
        
        docker exec "$node" bash -c "
            mkdir -p /root/.ssh
            echo '$public_key' >> /root/.ssh/authorized_keys
            chmod 700 /root/.ssh
            chmod 600 /root/.ssh/authorized_keys
            # 去重
            sort -u /root/.ssh/authorized_keys > /root/.ssh/authorized_keys.tmp
            mv /root/.ssh/authorized_keys.tmp /root/.ssh/authorized_keys
        "
        
        if [ $? -eq 0 ]; then
            info "$node: SSH密钥同步成功"
        else
            error "$node: SSH密钥同步失败"
        fi
    done
}

# 函数：测试SSH连接
test_ssh_connection() {
    local node=$1
    
    info "测试从Master到 $node 的SSH连接..."
    
    docker exec ai-infra-slurm-master bash -c "
        ssh -o StrictHostKeyChecking=no -o ConnectTimeout=5 root@$node 'echo SSH连接成功'
    "
    
    if [ $? -eq 0 ]; then
        info "✓ SSH连接测试成功"
    else
        error "✗ SSH连接测试失败"
        return 1
    fi
}

# 函数：显示使用说明
usage() {
    cat <<EOF
SLURM节点管理脚本

用法: $0 <command> [options]

命令:
    status <nodes...>           检查节点状态
    init <nodes...>             初始化节点（完整流程）
    sync-munge <nodes...>       仅同步Munge密钥
    sync-conf <nodes...>        仅同步SLURM配置
    sync-ssh <nodes...>         同步SSH密钥
    start-munge <nodes...>      启动Munge服务
    start-slurmd <nodes...>     启动SLURMD服务
    test-ssh <node>             测试SSH连接
    get-munge-key               获取Munge密钥（base64）

环境变量:
    BACKEND_URL                 后端API地址（默认: http://localhost:8082）
    CLUSTER_ID                  集群ID（默认: 1）

示例:
    # 检查所有节点状态
    $0 status test-rocky02 test-rocky03 test-ssh02 test-ssh03
    
    # 初始化所有节点
    $0 init test-rocky02 test-rocky03 test-ssh02 test-ssh03
    
    # 仅同步Munge密钥
    $0 sync-munge test-rocky02 test-rocky03
    
    # 同步SSH密钥
    $0 sync-ssh test-rocky02 test-rocky03 test-ssh02 test-ssh03
    
    # 测试SSH连接
    $0 test-ssh test-rocky02
EOF
}

# 主函数
main() {
    if [ $# -lt 1 ]; then
        usage
        exit 1
    fi
    
    local command=$1
    shift
    
    case "$command" in
        status)
            if [ $# -lt 1 ]; then
                error "请指定节点名称"
                exit 1
            fi
            check_nodes_status "$@"
            ;;
        init)
            if [ $# -lt 1 ]; then
                error "请指定节点名称"
                exit 1
            fi
            batch_init_nodes "$@"
            ;;
        sync-munge)
            if [ $# -lt 1 ]; then
                error "请指定节点名称"
                exit 1
            fi
            munge_key=$(get_munge_key)
            for node in "$@"; do
                sync_munge_key_to_node "$node" "$munge_key"
            done
            ;;
        sync-conf)
            if [ $# -lt 1 ]; then
                error "请指定节点名称"
                exit 1
            fi
            for node in "$@"; do
                sync_slurm_conf_to_node "$node"
            done
            ;;
        sync-ssh)
            if [ $# -lt 1 ]; then
                error "请指定节点名称"
                exit 1
            fi
            sync_ssh_keys "$@"
            ;;
        start-munge)
            if [ $# -lt 1 ]; then
                error "请指定节点名称"
                exit 1
            fi
            for node in "$@"; do
                start_munge_on_node "$node"
            done
            ;;
        start-slurmd)
            if [ $# -lt 1 ]; then
                error "请指定节点名称"
                exit 1
            fi
            for node in "$@"; do
                start_slurmd_on_node "$node"
            done
            ;;
        test-ssh)
            if [ $# -ne 1 ]; then
                error "请指定一个节点名称"
                exit 1
            fi
            test_ssh_connection "$1"
            ;;
        get-munge-key)
            get_munge_key
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            error "未知命令: $command"
            usage
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"
