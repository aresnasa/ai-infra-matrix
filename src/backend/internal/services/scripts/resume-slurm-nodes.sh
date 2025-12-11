#!/bin/bash
#
# resume-slurm-nodes.sh - Resume DOWN nodes after successful installation
#
# This script checks for DOWN nodes and resumes them if slurmd is running
#
# Usage: 
#   ./resume-slurm-nodes.sh [node1,node2,...]
#
# Arguments:
#   nodes - Comma-separated list of node names (optional, defaults to all DOWN nodes)
#

set -euo pipefail

# 配置变量
NODES="${1:-}"
MAX_RETRIES=5
RETRY_INTERVAL=3

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

log_success() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] [SUCCESS] $*"
}

# 检查是否有 scontrol 命令
check_scontrol() {
    if ! command -v scontrol &>/dev/null; then
        log_error "scontrol command not found. This script must be run on SLURM master."
        exit 1
    fi
}

# 获取所有 DOWN 状态的节点
get_down_nodes() {
    log_info "Checking for DOWN nodes..."
    
    # 使用 sinfo 获取 DOWN 状态的节点
    DOWN_NODES=$(sinfo -h -o "%N %T" | grep -E "down|drain|drng|drained" | awk '{print $1}' | tr '\n' ',' | sed 's/,$//')
    
    if [ -z "$DOWN_NODES" ]; then
        log_info "No DOWN nodes found"
        return 1
    fi
    
    log_info "Found DOWN nodes: $DOWN_NODES"
    echo "$DOWN_NODES"
}

# 检查节点的 slurmd 状态
check_node_slurmd() {
    local node="$1"
    
    # 使用 scontrol show node 检查节点详细信息
    local node_info=$(scontrol show node "$node" 2>/dev/null)
    
    if [ -z "$node_info" ]; then
        log_warn "Cannot get info for node: $node"
        return 1
    fi
    
    # 检查节点是否响应
    if echo "$node_info" | grep -q "State=DOWN"; then
        # 检查原因
        local reason=$(echo "$node_info" | grep "Reason=" | sed 's/.*Reason=\([^,]*\).*/\1/')
        log_info "Node $node is DOWN. Reason: $reason"
        
        # 如果原因是"新添加节点"或"节点未响应"，尝试恢复
        if echo "$reason" | grep -qE "新添加节点|节点未响应|Not responding"; then
            return 0
        fi
    fi
    
    return 1
}

# 恢复节点状态
resume_node() {
    local node="$1"
    
    log_info "Attempting to resume node: $node"
    
    # 执行 resume 命令
    if scontrol update NodeName="$node" State=RESUME; then
        log_success "✓ Resume command executed for node: $node"
        return 0
    else
        log_error "Failed to execute resume command for node: $node"
        return 1
    fi
}

# 等待并验证节点状态
verify_node_state() {
    local node="$1"
    local max_wait=30
    local waited=0
    
    log_info "Verifying node state: $node"
    
    while [ $waited -lt $max_wait ]; do
        local state=$(sinfo -h -n "$node" -o "%T" 2>/dev/null | head -1)
        
        case "$state" in
            idle|alloc|mix|allocated|mixed)
                log_success "✓ Node $node is now in state: $state"
                return 0
                ;;
            down|drain|drained|draining)
                log_info "Node $node is still in state: $state (waited ${waited}s)"
                ;;
            *)
                log_warn "Node $node has unexpected state: $state"
                ;;
        esac
        
        sleep 2
        waited=$((waited + 2))
    done
    
    log_warn "Node $node state verification timeout after ${max_wait}s"
    return 1
}

# 处理单个节点
process_node() {
    local node="$1"
    
    log_info "=========================================="
    log_info "Processing node: $node"
    log_info "=========================================="
    
    # 检查当前状态
    local current_state=$(sinfo -h -n "$node" -o "%T" 2>/dev/null | head -1)
    log_info "Current state: $current_state"
    
    # 如果节点已经是正常状态，跳过
    case "$current_state" in
        idle|alloc|mix|allocated|mixed)
            log_success "✓ Node $node is already in healthy state: $current_state"
            return 0
            ;;
    esac
    
    # 检查节点是否需要恢复
    if ! check_node_slurmd "$node"; then
        log_warn "Node $node may not be ready for resume (reason check failed)"
        # 继续尝试恢复
    fi
    
    # 尝试恢复节点（带重试）
    local attempt=1
    while [ $attempt -le $MAX_RETRIES ]; do
        log_info "Attempt $attempt/$MAX_RETRIES to resume node: $node"
        
        if resume_node "$node"; then
            # 等待状态变化
            sleep $RETRY_INTERVAL
            
            # 验证状态
            if verify_node_state "$node"; then
                log_success "✓ Node $node resumed successfully"
                return 0
            fi
        fi
        
        if [ $attempt -lt $MAX_RETRIES ]; then
            log_info "Retry in ${RETRY_INTERVAL}s..."
            sleep $RETRY_INTERVAL
        fi
        
        attempt=$((attempt + 1))
    done
    
    log_error "Failed to resume node $node after $MAX_RETRIES attempts"
    return 1
}

# 主函数
main() {
    log_info "=========================================="
    log_info "SLURM Node Resume Script"
    log_info "=========================================="
    log_info ""
    
    # 检查环境
    check_scontrol
    
    # 确定要处理的节点列表
    local node_list=""
    if [ -n "$NODES" ]; then
        node_list="$NODES"
        log_info "Target nodes (from argument): $node_list"
    else
        # 自动获取 DOWN 节点
        node_list=$(get_down_nodes) || {
            log_success "No DOWN nodes found. Nothing to do."
            exit 0
        }
        log_info "Target nodes (auto-detected): $node_list"
    fi
    
    # 将逗号分隔的节点列表转换为数组
    IFS=',' read -ra NODE_ARRAY <<< "$node_list"
    
    # 统计结果
    local total=${#NODE_ARRAY[@]}
    local success=0
    local failed=0
    
    log_info ""
    log_info "Processing $total node(s)..."
    log_info ""
    
    # 处理每个节点
    for node in "${NODE_ARRAY[@]}"; do
        # 移除可能的空格
        node=$(echo "$node" | xargs)
        
        if [ -z "$node" ]; then
            continue
        fi
        
        if process_node "$node"; then
            success=$((success + 1))
        else
            failed=$((failed + 1))
        fi
        
        log_info ""
    done
    
    # 显示最终结果
    log_info "=========================================="
    log_info "Summary"
    log_info "=========================================="
    log_info "Total nodes: $total"
    log_info "Successful: $success"
    log_info "Failed: $failed"
    log_info ""
    
    # 显示当前集群状态
    log_info "Current cluster state:"
    sinfo
    
    # 如果有失败的节点，返回错误码
    if [ $failed -gt 0 ]; then
        log_error "Some nodes failed to resume. Please check manually."
        exit 1
    fi
    
    log_success "All nodes resumed successfully!"
    exit 0
}

# 执行主函数
main "$@"
