#!/bin/bash
#===============================================================================
# 跨机器文件同步脚本
# 
# 流程：机器2 → 机器1 → 本地 → 机器3
# 
# 特点：
#   - 幂等执行（可重复运行，结果一致）
#   - 支持跳板机（通过机器1访问机器2）
#   - 使用 scp 同步文件/文件夹
#   - 自动创建目标目录
#   - 详细日志输出
#===============================================================================

set -euo pipefail

#-------------------------------------------------------------------------------
# 配置区域
#-------------------------------------------------------------------------------

JUMP_HOST_1="user1@192.168.1.10"
JUMP_HOST_1_PORT="22"
JUMP_HOST_1_KEY=""

SOURCE_HOST_2="user2@192.168.1.20"
SOURCE_HOST_2_PORT="22"
SOURCE_HOST_2_KEY=""

TARGET_HOST_3="user3@192.168.1.30"
TARGET_HOST_3_PORT="22"
TARGET_HOST_3_KEY=""

SOURCE_PATH_ON_HOST2="/data/important-files/"
TEMP_PATH_ON_HOST1="/tmp/sync-staging/"
LOCAL_PATH="/tmp/sync-local/"
TARGET_PATH_ON_HOST3="/data/backup/"

SSH_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=30"
LOG_FILE="/tmp/sync-remote-files-$(date +%Y%m%d_%H%M%S).log"

#-------------------------------------------------------------------------------
# 函数定义
#-------------------------------------------------------------------------------

log() {
    local level="$1"
    shift
    local msg="$*"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    local line="[$timestamp] [$level] $msg"
    echo "$line" | tee -a "$LOG_FILE"
}

log_info() { log "INFO" "$@"; }
log_warn() { log "WARN" "$@"; }
log_error() { log "ERROR" "$@"; }
log_step() { 
    echo ""
    log "STEP" "=========================================="
    log "STEP" "$@"
    log "STEP" "=========================================="
}

build_ssh_opts() {
    local port="$1"
    local key="$2"
    local opts="$SSH_OPTS -p $port"
    if [[ -n "$key" ]]; then
        local expanded_key="${key/#\~/$HOME}"
        if [[ -f "$expanded_key" ]]; then
            opts="$opts -i $expanded_key"
        fi
    fi
    echo "$opts"
}

build_scp_opts() {
    local port="$1"
    local key="$2"
    local opts="$SSH_OPTS -P $port"
    if [[ -n "$key" ]]; then
        local expanded_key="${key/#\~/$HOME}"
        if [[ -f "$expanded_key" ]]; then
            opts="$opts -i $expanded_key"
        fi
    fi
    echo "$opts"
}

sync_host2_to_host1() {
    log_step "步骤 1/4: 从机器2同步到机器1"
    log_info "源: $SOURCE_HOST_2:$SOURCE_PATH_ON_HOST2"
    log_info "目标: $JUMP_HOST_1:$TEMP_PATH_ON_HOST1"
    
    local jump_ssh_opts=$(build_ssh_opts "$JUMP_HOST_1_PORT" "$JUMP_HOST_1_KEY")
    local target_dir=$(dirname "$TEMP_PATH_ON_HOST1")
    ssh $jump_ssh_opts "$JUMP_HOST_1" "mkdir -p $target_dir"
    
    local scp_opts="-o StrictHostKeyChecking=no -P $SOURCE_HOST_2_PORT"
    if [[ "$SOURCE_PATH_ON_HOST2" =~ /$ ]]; then
        log_info "使用 scp -r 同步目录..."
        ssh $jump_ssh_opts "$JUMP_HOST_1" "scp -r $scp_opts $SOURCE_HOST_2:$SOURCE_PATH_ON_HOST2 $TEMP_PATH_ON_HOST1"
    else
        log_info "使用 scp 同步文件..."
        ssh $jump_ssh_opts "$JUMP_HOST_1" "scp $scp_opts $SOURCE_HOST_2:$SOURCE_PATH_ON_HOST2 $TEMP_PATH_ON_HOST1"
    fi
    
    log_info "✅ 步骤1完成"
}

sync_host1_to_local() {
    log_step "步骤 2/4: 从机器1同步到本地"
    log_info "源: $JUMP_HOST_1:$TEMP_PATH_ON_HOST1"
    log_info "目标: $LOCAL_PATH"
    
    mkdir -p "$LOCAL_PATH"
    local jump_scp_opts=$(build_scp_opts "$JUMP_HOST_1_PORT" "$JUMP_HOST_1_KEY")
    
    if [[ "$TEMP_PATH_ON_HOST1" =~ /$ ]]; then
        log_info "使用 scp -r 同步目录..."
        scp -r $jump_scp_opts "$JUMP_HOST_1:$TEMP_PATH_ON_HOST1" "$LOCAL_PATH/"
    else
        log_info "使用 scp 同步文件..."
        scp $jump_scp_opts "$JUMP_HOST_1:$TEMP_PATH_ON_HOST1" "$LOCAL_PATH/"
    fi
    
    log_info "✅ 步骤2完成"
}

sync_local_to_host3() {
    log_step "步骤 3/4: 从本地同步到机器3"
    log_info "源: $LOCAL_PATH"
    log_info "目标: $TARGET_HOST_3:$TARGET_PATH_ON_HOST3"
    
    local target_ssh_opts=$(build_ssh_opts "$TARGET_HOST_3_PORT" "$TARGET_HOST_3_KEY")
    local target_scp_opts=$(build_scp_opts "$TARGET_HOST_3_PORT" "$TARGET_HOST_3_KEY")
    
    ssh $target_ssh_opts "$TARGET_HOST_3" "mkdir -p $TARGET_PATH_ON_HOST3"
    
    local filename=$(basename "$TEMP_PATH_ON_HOST1")
    local local_source="$LOCAL_PATH/$filename"
    
    if [[ -d "$local_source" ]]; then
        log_info "使用 scp -r 同步目录: $local_source"
        scp -r $target_scp_opts "$local_source" "$TARGET_HOST_3:$TARGET_PATH_ON_HOST3/"
    else
        log_info "使用 scp 同步文件: $local_source"
        scp $target_scp_opts "$local_source" "$TARGET_HOST_3:$TARGET_PATH_ON_HOST3/"
    fi
    
    log_info "✅ 步骤3完成"
}

cleanup_temp_files() {
    log_step "步骤 4/4: 清理临时文件"
    
    local cleanup_local="${CLEANUP_LOCAL:-false}"
    local cleanup_host1="${CLEANUP_HOST1:-false}"
    
    if [[ "$cleanup_host1" == "true" ]]; then
        log_info "清理机器1: $TEMP_PATH_ON_HOST1"
        local jump_ssh_opts=$(build_ssh_opts "$JUMP_HOST_1_PORT" "$JUMP_HOST_1_KEY")
        ssh $jump_ssh_opts "$JUMP_HOST_1" "rm -rf $TEMP_PATH_ON_HOST1" || log_warn "清理失败"
    else
        log_info "跳过清理机器1 (CLEANUP_HOST1=true 启用)"
    fi
    
    if [[ "$cleanup_local" == "true" ]]; then
        log_info "清理本地: $LOCAL_PATH"
        rm -rf "$LOCAL_PATH" || log_warn "清理失败"
    else
        log_info "跳过清理本地 (CLEANUP_LOCAL=true 启用)"
    fi
    
    log_info "✅ 步骤4完成"
}

show_summary() {
    echo ""
    echo "================================================================================"
    echo "                              同步完成摘要"
    echo "================================================================================"
    echo ""
    echo "  [机器2] $SOURCE_HOST_2:$SOURCE_PATH_ON_HOST2"
    echo "      ↓"
    echo "  [机器1] $JUMP_HOST_1:$TEMP_PATH_ON_HOST1"
    echo "      ↓"
    echo "  [本地]  $LOCAL_PATH"
    echo "      ↓"
    echo "  [机器3] $TARGET_HOST_3:$TARGET_PATH_ON_HOST3"
    echo ""
    echo "  日志: $LOG_FILE"
    echo "================================================================================"
}

load_config() {
    local config_file="$1"
    if [[ -f "$config_file" ]]; then
        log_info "加载配置: $config_file"
        source "$config_file"
    else
        log_error "配置文件不存在: $config_file"
        exit 1
    fi
}

show_help() {
    echo "用法: $0 [选项] [配置文件]"
    echo ""
    echo "跨机器文件同步脚本 - 使用 scp"
    echo ""
    echo "选项:"
    echo "  -h, --help          帮助"
    echo "  -c, --config FILE   配置文件"
    echo "  -n, --dry-run       模拟运行"
    echo "  --step1-only        只执行步骤1"
    echo "  --step2-only        只执行步骤2"
    echo "  --step3-only        只执行步骤3"
    echo "  --cleanup           清理临时文件"
}

dry_run() {
    log_info "=== 模拟运行 ==="
    log_info "  1. scp: $SOURCE_HOST_2:$SOURCE_PATH_ON_HOST2 → $JUMP_HOST_1:$TEMP_PATH_ON_HOST1"
    log_info "  2. scp: $JUMP_HOST_1:$TEMP_PATH_ON_HOST1 → 本地 $LOCAL_PATH"
    log_info "  3. scp: 本地 → $TARGET_HOST_3:$TARGET_PATH_ON_HOST3"
}

main() {
    local dry_run_mode=false
    local step1_only=false
    local step2_only=false
    local step3_only=false
    local config_file=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -h|--help) show_help; exit 0 ;;
            -c|--config) config_file="$2"; shift 2 ;;
            -n|--dry-run) dry_run_mode=true; shift ;;
            --step1-only) step1_only=true; shift ;;
            --step2-only) step2_only=true; shift ;;
            --step3-only) step3_only=true; shift ;;
            --cleanup) CLEANUP_LOCAL=true; CLEANUP_HOST1=true; shift ;;
            *)
                if [[ -f "$1" ]]; then
                    config_file="$1"
                else
                    log_error "未知参数: $1"
                    exit 1
                fi
                shift
                ;;
        esac
    done
    
    if [[ -n "$config_file" ]]; then
        load_config "$config_file"
    fi
    
    log_info "开始同步 (scp)"
    log_info "日志: $LOG_FILE"
    
    if [[ "$dry_run_mode" == "true" ]]; then
        dry_run
        exit 0
    fi
    
    local start_time=$(date +%s)
    
    if [[ "$step1_only" == "true" ]]; then
        sync_host2_to_host1
    elif [[ "$step2_only" == "true" ]]; then
        sync_host1_to_local
    elif [[ "$step3_only" == "true" ]]; then
        sync_local_to_host3
    else
        sync_host2_to_host1
        sync_host1_to_local
        sync_local_to_host3
        cleanup_temp_files
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    show_summary
    log_info "耗时: ${duration} 秒"
    log_info "🎉 完成!"
}

trap 'log_error "失败，退出码: $?"; exit 1' ERR
main "$@"
