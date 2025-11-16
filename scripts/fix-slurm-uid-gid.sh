#!/bin/bash
#
# fix-slurm-uid-gid.sh - 批量修复 SLURM 计算节点的 UID/GID 不一致问题
#
# 用途：统一所有计算节点上的 slurm 和 munge 用户 UID/GID
# 目标 UID/GID:
#   - munge: UID=998, GID=998
#   - slurm: UID=999, GID=999
#

set -euo pipefail

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 默认节点列表（可通过参数覆盖）
DEFAULT_NODES="test-ssh01 test-ssh02 test-ssh03 test-rocky01 test-rocky02 test-rocky03"
NODES="${@:-$DEFAULT_NODES}"

# 目标 UID/GID
TARGET_MUNGE_UID=998
TARGET_MUNGE_GID=998
TARGET_SLURM_UID=999
TARGET_SLURM_GID=999

log_info() {
    echo -e "${BLUE}[INFO]${NC} $*"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $*"
}

# 检查节点是否可访问
check_node() {
    local node=$1
    if docker exec "$node" echo "ping" >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# 修复单个节点的 UID/GID
fix_node() {
    local node=$1
    
    echo ""
    echo "=========================================="
    log_info "Processing node: $node"
    echo "=========================================="
    
    # 检查节点可访问性
    if ! check_node "$node"; then
        log_error "Cannot access node $node (container not running?)"
        return 1
    fi
    
    # 在节点上执行修复脚本
    docker exec "$node" bash -s <<'EOSCRIPT'
        # 停止服务
        echo "Stopping services..."
        systemctl stop slurmd 2>/dev/null || pkill -9 slurmd 2>/dev/null || true
        systemctl stop munge 2>/dev/null || pkill -9 munged 2>/dev/null || true
        sleep 2
        
        # 修复 munge 用户
        echo "Checking munge user..."
        if id munge >/dev/null 2>&1; then
            CURRENT_MUNGE_UID=$(id -u munge)
            CURRENT_MUNGE_GID=$(id -g munge)
            echo "  Current munge: UID=$CURRENT_MUNGE_UID, GID=$CURRENT_MUNGE_GID"
            
            if [ "$CURRENT_MUNGE_UID" != "998" ] || [ "$CURRENT_MUNGE_GID" != "998" ]; then
                echo "  Fixing munge UID/GID to 998..."
                groupmod -g 998 munge 2>/dev/null || true
                usermod -u 998 -g 998 munge 2>/dev/null || true
                
                # 更新文件所有权
                echo "  Updating munge file ownership..."
                find /etc/munge /var/lib/munge /var/log/munge /run/munge -user $CURRENT_MUNGE_UID -exec chown 998:998 {} \; 2>/dev/null || true
                find /etc/munge /var/lib/munge /var/log/munge /run/munge -group $CURRENT_MUNGE_GID -exec chgrp 998 {} \; 2>/dev/null || true
                
                echo "  ✓ munge fixed: $CURRENT_MUNGE_UID -> 998"
            else
                echo "  ✓ munge UID/GID already correct"
            fi
        else
            echo "  ! munge user does not exist (will be created during SLURM installation)"
        fi
        
        # 修复 slurm 用户
        echo "Checking slurm user..."
        if id slurm >/dev/null 2>&1; then
            CURRENT_SLURM_UID=$(id -u slurm)
            CURRENT_SLURM_GID=$(id -g slurm)
            echo "  Current slurm: UID=$CURRENT_SLURM_UID, GID=$CURRENT_SLURM_GID"
            
            if [ "$CURRENT_SLURM_UID" != "999" ] || [ "$CURRENT_SLURM_GID" != "999" ]; then
                echo "  Fixing slurm UID/GID to 999..."
                groupmod -g 999 slurm 2>/dev/null || true
                usermod -u 999 -g 999 slurm 2>/dev/null || true
                
                # 更新文件所有权
                echo "  Updating slurm file ownership..."
                find /var/spool/slurm /var/log/slurm /run/slurm /etc/slurm -user $CURRENT_SLURM_UID -exec chown 999:999 {} \; 2>/dev/null || true
                find /var/spool/slurm /var/log/slurm /run/slurm /etc/slurm -group $CURRENT_SLURM_GID -exec chgrp 999 {} \; 2>/dev/null || true
                
                echo "  ✓ slurm fixed: $CURRENT_SLURM_UID -> 999"
            else
                echo "  ✓ slurm UID/GID already correct"
            fi
        else
            echo "  ! slurm user does not exist (will be created during SLURM installation)"
        fi
        
        # 重启服务
        echo "Restarting services..."
        systemctl start munge 2>/dev/null && echo "  ✓ munge started" || echo "  ! munge failed to start"
        sleep 2
        systemctl start slurmd 2>/dev/null && echo "  ✓ slurmd started" || echo "  ! slurmd failed to start"
        
        # 最终验证
        echo ""
        echo "Final UID/GID verification:"
        if id munge >/dev/null 2>&1; then
            echo "  munge: $(id munge)"
        fi
        if id slurm >/dev/null 2>&1; then
            echo "  slurm: $(id slurm)"
        fi
EOSCRIPT
    
    local exit_code=$?
    
    if [ $exit_code -eq 0 ]; then
        log_success "Node $node processed successfully"
    else
        log_error "Failed to process node $node (exit code: $exit_code)"
    fi
    
    return $exit_code
}

# 主函数
main() {
    log_info "=========================================="
    log_info "SLURM UID/GID Batch Fix Script"
    log_info "=========================================="
    log_info "Target UID/GID:"
    log_info "  - munge: UID=$TARGET_MUNGE_UID, GID=$TARGET_MUNGE_GID"
    log_info "  - slurm: UID=$TARGET_SLURM_UID, GID=$TARGET_SLURM_GID"
    log_info ""
    log_info "Nodes to process: $NODES"
    log_info ""
    
    local success_count=0
    local fail_count=0
    local skip_count=0
    
    for node in $NODES; do
        if fix_node "$node"; then
            ((success_count++))
        else
            ((fail_count++))
        fi
    done
    
    echo ""
    echo "=========================================="
    log_info "Summary"
    echo "=========================================="
    log_success "Successfully processed: $success_count nodes"
    
    if [ $fail_count -gt 0 ]; then
        log_error "Failed: $fail_count nodes"
    fi
    
    echo ""
    
    if [ $fail_count -eq 0 ]; then
        log_success "All nodes processed successfully!"
        log_info ""
        log_info "Next steps:"
        log_info "  1. Verify SLURM cluster status: docker exec ai-infra-slurm-master sinfo"
        log_info "  2. Test job submission: docker exec ai-infra-slurm-master srun -N1 hostname"
        log_info "  3. Check for errors: docker exec ai-infra-slurm-master tail -50 /var/log/slurm/slurmctld.log"
        return 0
    else
        log_error "Some nodes failed to process. Please check the logs above."
        return 1
    fi
}

# 执行主函数
main "$@"
