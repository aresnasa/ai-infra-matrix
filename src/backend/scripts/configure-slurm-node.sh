#!/bin/bash
#
# configure-slurm-node.sh - Configure and start SLURM node after installation
#
# This script is executed remotely via SaltStack to configure SLURM nodes
# It deploys munge key and slurm.conf, then starts services
#
# Usage: 
#   ./configure-slurm-node.sh <master_host> <munge_key_b64> <slurm_conf_b64>
#
# Arguments:
#   master_host     - SLURM master hostname (e.g., ai-infra-slurm-master)
#   munge_key_b64   - Base64-encoded munge key
#   slurm_conf_b64  - Base64-encoded slurm.conf (optional, use "-" to skip)
#

set -euo pipefail

# 配置变量
MASTER_HOST="${1:-ai-infra-slurm-master}"
MUNGE_KEY_B64="${2:-}"
SLURM_CONF_B64="${3:--}"

# 日志函数
log_info() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] [INFO] $*"
}

log_error() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] [ERROR] $*" >&2
}

log_warn() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] [WARN] $*"
}

# 检测操作系统类型
detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS_ID="$ID"
        OS_VERSION="$VERSION_ID"
        
        case "$OS_ID" in
            rocky|centos|rhel|fedora)
                OS_TYPE="rpm"
                ;;
            ubuntu|debian)
                OS_TYPE="deb"
                ;;
            *)
                log_error "Unsupported OS: $OS_ID"
                exit 1
                ;;
        esac
    else
        log_error "Cannot detect OS type"
        exit 1
    fi
}

# 部署 munge key
deploy_munge_key() {
    if [ -z "$MUNGE_KEY_B64" ]; then
        log_error "Munge key not provided"
        return 1
    fi
    
    log_info "Deploying munge key..."
    
    # 解码并保存 key
    echo "$MUNGE_KEY_B64" | base64 -d > /etc/munge/munge.key
    
    # 设置权限 (munge key 必须 munge 用户所有)
    chown munge:munge /etc/munge/munge.key
    chmod 400 /etc/munge/munge.key
    
    # 验证 key
    KEY_MD5=$(md5sum /etc/munge/munge.key | cut -d' ' -f1)
    log_info "✓ Munge key deployed (MD5: $KEY_MD5)"
}

# 部署 slurm.conf
deploy_slurm_conf() {
    if [ "$SLURM_CONF_B64" = "-" ]; then
        log_info "Skipping slurm.conf deployment (not provided)"
        return 0
    fi
    
    log_info "Deploying slurm.conf..."
    
    # 根据 OS 类型确定配置文件路径
    if [ "$OS_TYPE" = "rpm" ]; then
        # Rocky: 使用 /usr/etc/slurm/slurm.conf
        SLURM_CONF_PATH="/usr/etc/slurm/slurm.conf"
        mkdir -p /usr/etc/slurm
        
        # 解码并保存配置
        echo "$SLURM_CONF_B64" | base64 -d > "$SLURM_CONF_PATH"
        
        # 如果 /etc/slurm 不是符号链接，创建它
        if [ ! -L /etc/slurm ]; then
            if [ -d /etc/slurm ]; then
                mv /etc/slurm /etc/slurm.backup.$(date +%s)
            fi
            ln -sf /usr/etc/slurm /etc/slurm
            log_info "Created symlink: /etc/slurm -> /usr/etc/slurm"
        fi
        
    elif [ "$OS_TYPE" = "deb" ]; then
        # Ubuntu: 使用 /etc/slurm/slurm.conf
        SLURM_CONF_PATH="/etc/slurm/slurm.conf"
        mkdir -p /etc/slurm
        
        # 解码并保存配置
        echo "$SLURM_CONF_B64" | base64 -d > "$SLURM_CONF_PATH"
    fi
    
    # 设置权限
    chmod 644 "$SLURM_CONF_PATH"
    chown root:root "$SLURM_CONF_PATH"
    
    # 验证配置
    CONF_MD5=$(md5sum "$SLURM_CONF_PATH" | cut -d' ' -f1)
    log_info "✓ slurm.conf deployed to $SLURM_CONF_PATH (MD5: $CONF_MD5)"
}

# 启动 munge 服务
start_munge() {
    log_info "Starting munge authentication service..."
    
    # 确保 munge 目录权限正确
    mkdir -p /etc/munge /var/lib/munge /var/log/munge /run/munge
    chown -R root:root /var/log/munge /var/lib/munge
    chown -R munge:munge /etc/munge /run/munge
    chmod 700 /etc/munge /var/lib/munge /var/log/munge
    chmod 755 /run/munge
    
    # 停止已有的 munge 进程
    systemctl stop munge 2>/dev/null || pkill -9 munged 2>/dev/null || true
    sleep 1
    
    # 清理旧的 pid 文件
    rm -f /var/run/munge/munged.pid /run/munge/munged.pid
    
    # 启动 munge
    systemctl start munge
    
    # 等待启动
    sleep 2
    
    # 验证
    if systemctl is-active munge > /dev/null 2>&1; then
        log_info "✓ Munge service started successfully"
        
        # 测试 munge
        if munge -n | unmunge > /dev/null 2>&1; then
            log_info "✓ Munge authentication test passed"
        else
            log_warn "Munge authentication test failed"
        fi
    else
        log_error "Failed to start munge service"
        # 显示日志帮助调试
        journalctl -u munge -n 20 --no-pager || tail -20 /var/log/munge/munged.log
        return 1
    fi
}

# 启动 slurmd 服务
start_slurmd() {
    log_info "Starting slurmd daemon..."
    
    # 停止已有的 slurmd 进程
    pkill -9 slurmd 2>/dev/null || true
    systemctl stop slurmd 2>/dev/null || true
    
    # 根据 OS 类型选择启动方式
    if [ "$OS_TYPE" = "deb" ]; then
        # Ubuntu: 使用 systemd
        systemctl start slurmd
        sleep 2
        
        if systemctl is-active slurmd > /dev/null 2>&1; then
            log_info "✓ slurmd service started successfully (systemd)"
        else
            log_warn "systemd start failed, trying direct start..."
            /usr/sbin/slurmd
            sleep 2
        fi
    else
        # Rocky: 直接启动（避免 systemd 超时）
        /usr/sbin/slurmd
        sleep 2
    fi
    
    # 验证进程是否运行
    if pgrep -x slurmd > /dev/null; then
        log_info "✓ slurmd daemon is running"
        log_info "  PID: $(pgrep -x slurmd | tr '\n' ' ')"
    else
        log_error "Failed to start slurmd daemon"
        return 1
    fi
}

# 恢复节点状态到 IDLE
resume_node_state() {
    log_info "Resuming node state on SLURM master..."
    
    # 获取本机主机名
    local hostname=$(hostname)
    log_info "  Node hostname: $hostname"
    
    # 等待节点注册到 slurmctld
    log_info "  Waiting for node registration..."
    sleep 5
    
    # 尝试通过 SSH 在 master 上执行 scontrol 命令
    # 使用环境变量中的密码（如果提供）
    local master_password="${SLURM_MASTER_SSH_PASSWORD:-}"
    
    if [ -n "$master_password" ]; then
        # 使用 sshpass 连接到 master
        if command -v sshpass >/dev/null 2>&1; then
            log_info "  Attempting to resume node state via SSH..."
            if sshpass -p "$master_password" ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10 \
                root@"$MASTER_HOST" "scontrol update NodeName=$hostname State=RESUME" 2>/dev/null; then
                log_info "✓ Node state resumed to IDLE on master"
            else
                log_warn "Failed to resume node state via SSH (master may handle this automatically)"
            fi
        else
            log_warn "sshpass not available, skipping automatic state resume"
            log_info "  Please manually run on master: scontrol update NodeName=$hostname State=RESUME"
        fi
    else
        log_warn "SLURM_MASTER_SSH_PASSWORD not set, skipping automatic state resume"
        log_info "  Please manually run on master: scontrol update NodeName=$hostname State=RESUME"
    fi
}

# 主函数
main() {
    log_info "=========================================="
    log_info "SLURM Node Configuration Script"
    log_info "=========================================="
    log_info "Master Host: $MASTER_HOST"
    log_info ""
    
    # 检测操作系统
    detect_os
    log_info "Detected OS: $OS_ID $OS_VERSION ($OS_TYPE)"
    
    # 部署配置
    deploy_munge_key
    deploy_slurm_conf
    
    # 启动服务
    start_munge
    start_slurmd
    
    # 恢复节点状态
    resume_node_state
    
    log_info ""
    log_info "=========================================="
    log_info "✓ SLURM node configuration completed successfully"
    log_info "=========================================="
    log_info ""
    log_info "Node is now ready to accept jobs from: $MASTER_HOST"
    log_info ""
    log_info "Next steps:"
    log_info "  1. Verify node status: sinfo (on master)"
    log_info "  2. If node is still DOWN, manually resume: scontrol update NodeName=$(hostname) State=RESUME"
    log_info "  3. Test job submission: srun -N1 -w $(hostname) hostname"
    log_info ""
}

# 执行主函数
main "$@"
