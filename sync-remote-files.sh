#!/bin/bash
#===============================================================================
# 跨机器文件同步脚本
# 
# 流程：机器2 → 机器1 → 本地 → 机器3
# 
# 特点：
#   - 幂等执行（可重复运行，结果一致）
#   - 支持跳板机（通过机器1访问机器2）
#   - 使用 rsync 增量同步
#   - 自动创建目标目录
#   - 详细日志输出
#
# 使用方法：
#   ./sync-remote-files.sh [配置文件]
#   或直接修改脚本中的配置变量
#===============================================================================

set -euo pipefail

#-------------------------------------------------------------------------------
# 配置区域 - 请根据实际情况修改
#-------------------------------------------------------------------------------

# 机器1（跳板机）配置
JUMP_HOST_1="user1@192.168.1.10"
JUMP_HOST_1_PORT="22"
JUMP_HOST_1_KEY=""  # 留空则使用默认密钥，或指定: ~/.ssh/id_rsa_jump1

# 机器2（源数据机器，通过机器1访问）配置
SOURCE_HOST_2="user2@192.168.1.20"
SOURCE_HOST_2_PORT="22"
SOURCE_HOST_2_KEY=""  # 留空则使用默认密钥

# 机器3（最终目标机器）配置
TARGET_HOST_3="user3@192.168.1.30"
TARGET_HOST_3_PORT="22"
TARGET_HOST_3_KEY=""  # 留空则使用默认密钥

# 要同步的文件/文件夹配置
SOURCE_PATH_ON_HOST2="/data/important-files/"        # 机器2上的源路径
TEMP_PATH_ON_HOST1="/tmp/sync-staging/"              # 机器1上的临时存放路径
LOCAL_PATH="/tmp/sync-local/"                        # 本地临时存放路径
TARGET_PATH_ON_HOST3="/data/backup/"                 # 机器3上的目标路径

# rsync 选项
RSYNC_OPTS="-avz --progress --delete"  # -a归档 -v详细 -z压缩 --delete删除目标多余文件
# 如果不想删除目标多余文件，去掉 --delete

# SSH 选项
SSH_OPTS="-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -o ConnectTimeout=30"

# 日志配置
LOG_FILE="/tmp/sync-remote-files-$(date +%Y%m%d_%H%M%S).log"
VERBOSE=true

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

# 构建 SSH 命令选项
build_ssh_opts() {
    local port="$1"
    local key="$2"
    local opts="$SSH_OPTS -p $port"
    if [[ -n "$key" && -f "$key" ]]; then
        opts="$opts -i $key"
    fi
    echo "$opts"
}

# 检查远程主机连通性
check_host_connectivity() {
    local host="$1"
    local port="$2"
    local key="${3:-}"
    local via_jump="${4:-}"
    
    local ssh_opts=$(build_ssh_opts "$port" "$key")
    
    if [[ -n "$via_jump" ]]; then
        # 通过跳板机连接
        local jump_opts=$(build_ssh_opts "$JUMP_HOST_1_PORT" "$JUMP_HOST_1_KEY")
        ssh $jump_opts "$JUMP_HOST_1" "ssh $ssh_opts $host 'echo ok'" &>/dev/null
    else
        ssh $ssh_opts "$host" "echo ok" &>/dev/null
    fi
}

# 确保远程目录存在
ensure_remote_dir() {
    local host="$1"
    local port="$2"
    local key="$3"
    local dir="$4"
    local via_jump="${5:-}"
    
    local ssh_opts=$(build_ssh_opts "$port" "$key")
    
    log_info "确保远程目录存在: $host:$dir"
    
    if [[ -n "$via_jump" ]]; then
        local jump_opts=$(build_ssh_opts "$JUMP_HOST_1_PORT" "$JUMP_HOST_1_KEY")
        ssh $jump_opts "$JUMP_HOST_1" "ssh $ssh_opts $host 'mkdir -p $dir'"
    else
        ssh $ssh_opts "$host" "mkdir -p $dir"
    fi
}

# 步骤1: 从机器2同步到机器1（通过机器1执行）
sync_host2_to_host1() {
    log_step "步骤 1/4: 从机器2同步到机器1"
    log_info "源: $SOURCE_HOST_2:$SOURCE_PATH_ON_HOST2"
    log_info "目标: $JUMP_HOST_1:$TEMP_PATH_ON_HOST1"
    
    local jump_ssh_opts=$(build_ssh_opts "$JUMP_HOST_1_PORT" "$JUMP_HOST_1_KEY")
    local source_ssh_opts=$(build_ssh_opts "$SOURCE_HOST_2_PORT" "$SOURCE_HOST_2_KEY")
    
    # 在机器1上创建临时目录
    ssh $jump_ssh_opts "$JUMP_HOST_1" "mkdir -p $TEMP_PATH_ON_HOST1"
    
    # 在机器1上执行从机器2到机器1的rsync
    # 构建机器1上执行的rsync命令
    local rsync_ssh="ssh $source_ssh_opts"
    ssh $jump_ssh_opts "$JUMP_HOST_1" "rsync $RSYNC_OPTS -e '$rsync_ssh' $SOURCE_HOST_2:$SOURCE_PATH_ON_HOST2 $TEMP_PATH_ON_HOST1"
    
    log_info "✅ 步骤1完成: 数据已从机器2同步到机器1"
}

# 步骤2: 从机器1同步到本地
sync_host1_to_local() {
    log_step "步骤 2/4: 从机器1同步到本地"
    log_info "源: $JUMP_HOST_1:$TEMP_PATH_ON_HOST1"
    log_info "目标: $LOCAL_PATH"
    
    # 确保本地目录存在
    mkdir -p "$LOCAL_PATH"
    
    local jump_ssh_opts=$(build_ssh_opts "$JUMP_HOST_1_PORT" "$JUMP_HOST_1_KEY")
    local rsync_ssh="ssh $jump_ssh_opts"
    
    rsync $RSYNC_OPTS -e "$rsync_ssh" "$JUMP_HOST_1:$TEMP_PATH_ON_HOST1" "$LOCAL_PATH"
    
    log_info "✅ 步骤2完成: 数据已从机器1同步到本地"
}

# 步骤3: 从本地同步到机器3
sync_local_to_host3() {
    log_step "步骤 3/4: 从本地同步到机器3"
    log_info "源: $LOCAL_PATH"
    log_info "目标: $TARGET_HOST_3:$TARGET_PATH_ON_HOST3"
    
    local target_ssh_opts=$(build_ssh_opts "$TARGET_HOST_3_PORT" "$TARGET_HOST_3_KEY")
    local rsync_ssh="ssh $target_ssh_opts"
    
    # 确保目标目录存在
    ssh $target_ssh_opts "$TARGET_HOST_3" "mkdir -p $TARGET_PATH_ON_HOST3"
    
    rsync $RSYNC_OPTS -e "$rsync_ssh" "$LOCAL_PATH" "$TARGET_HOST_3:$TARGET_PATH_ON_HOST3"
    
    log_info "✅ 步骤3完成: 数据已从本地同步到机器3"
}

# 步骤4: 清理临时文件（可选）
cleanup_temp_files() {
    log_step "步骤 4/4: 清理临时文件"
    
    local cleanup_local="${CLEANUP_LOCAL:-false}"
    local cleanup_host1="${CLEANUP_HOST1:-false}"
    
    if [[ "$cleanup_host1" == "true" ]]; then
        log_info "清理机器1上的临时文件: $TEMP_PATH_ON_HOST1"
        local jump_ssh_opts=$(build_ssh_opts "$JUMP_HOST_1_PORT" "$JUMP_HOST_1_KEY")
        ssh $jump_ssh_opts "$JUMP_HOST_1" "rm -rf $TEMP_PATH_ON_HOST1" || log_warn "清理机器1临时文件失败"
    else
        log_info "跳过清理机器1临时文件 (设置 CLEANUP_HOST1=true 启用)"
    fi
    
    if [[ "$cleanup_local" == "true" ]]; then
        log_info "清理本地临时文件: $LOCAL_PATH"
        rm -rf "$LOCAL_PATH" || log_warn "清理本地临时文件失败"
    else
        log_info "跳过清理本地临时文件 (设置 CLEANUP_LOCAL=true 启用)"
    fi
    
    log_info "✅ 步骤4完成"
}

# 显示同步摘要
show_summary() {
    echo ""
    echo "================================================================================"
    echo "                              同步完成摘要"
    echo "================================================================================"
    echo ""
    echo "  同步路径:"
    echo "    [机器2] $SOURCE_HOST_2:$SOURCE_PATH_ON_HOST2"
    echo "        ↓"
    echo "    [机器1] $JUMP_HOST_1:$TEMP_PATH_ON_HOST1"
    echo "        ↓"
    echo "    [本地]  $LOCAL_PATH"
    echo "        ↓"
    echo "    [机器3] $TARGET_HOST_3:$TARGET_PATH_ON_HOST3"
    echo ""
    echo "  日志文件: $LOG_FILE"
    echo ""
    echo "================================================================================"
}

# 加载外部配置文件（如果提供）
load_config() {
    local config_file="$1"
    if [[ -f "$config_file" ]]; then
        log_info "加载配置文件: $config_file"
        # shellcheck source=/dev/null
        source "$config_file"
    else
        log_error "配置文件不存在: $config_file"
        exit 1
    fi
}

# 显示帮助
show_help() {
    cat << EOF
用法: $0 [选项] [配置文件]

跨机器文件同步脚本 - 从机器2通过机器1同步到本地，再同步到机器3

选项:
  -h, --help          显示帮助信息
  -c, --config FILE   指定配置文件
  -n, --dry-run       模拟运行（不实际执行）
  --step1-only        只执行步骤1（机器2→机器1）
  --step2-only        只执行步骤2（机器1→本地）
  --step3-only        只执行步骤3（本地→机器3）
  --cleanup           执行后清理临时文件

环境变量:
  CLEANUP_LOCAL=true   清理本地临时文件
  CLEANUP_HOST1=true   清理机器1临时文件

示例:
  $0                           使用脚本内置配置
  $0 -c /path/to/config.sh     使用外部配置文件
  $0 --step1-only              只执行机器2到机器1的同步
  CLEANUP_LOCAL=true $0        同步后清理本地临时文件

配置文件示例:
  # sync-config.sh
  JUMP_HOST_1="user@jump-server"
  SOURCE_HOST_2="user@source-server"
  TARGET_HOST_3="user@target-server"
  SOURCE_PATH_ON_HOST2="/data/files/"
  TARGET_PATH_ON_HOST3="/backup/files/"
EOF
}

# 模拟运行
dry_run() {
    log_info "=== 模拟运行模式 ==="
    log_info ""
    log_info "将执行以下操作:"
    log_info "  1. 从 $SOURCE_HOST_2:$SOURCE_PATH_ON_HOST2 同步到 $JUMP_HOST_1:$TEMP_PATH_ON_HOST1"
    log_info "  2. 从 $JUMP_HOST_1:$TEMP_PATH_ON_HOST1 同步到本地 $LOCAL_PATH"
    log_info "  3. 从本地 $LOCAL_PATH 同步到 $TARGET_HOST_3:$TARGET_PATH_ON_HOST3"
    log_info ""
    log_info "rsync 选项: $RSYNC_OPTS"
    log_info ""
}

#-------------------------------------------------------------------------------
# 主程序
#-------------------------------------------------------------------------------

main() {
    local dry_run_mode=false
    local step1_only=false
    local step2_only=false
    local step3_only=false
    local config_file=""
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case "$1" in
            -h|--help)
                show_help
                exit 0
                ;;
            -c|--config)
                config_file="$2"
                shift 2
                ;;
            -n|--dry-run)
                dry_run_mode=true
                shift
                ;;
            --step1-only)
                step1_only=true
                shift
                ;;
            --step2-only)
                step2_only=true
                shift
                ;;
            --step3-only)
                step3_only=true
                shift
                ;;
            --cleanup)
                CLEANUP_LOCAL=true
                CLEANUP_HOST1=true
                shift
                ;;
            *)
                # 假设是配置文件
                if [[ -f "$1" ]]; then
                    config_file="$1"
                else
                    log_error "未知参数: $1"
                    show_help
                    exit 1
                fi
                shift
                ;;
        esac
    done
    
    # 加载配置文件
    if [[ -n "$config_file" ]]; then
        load_config "$config_file"
    fi
    
    log_info "开始文件同步任务"
    log_info "日志文件: $LOG_FILE"
    
    # 模拟运行模式
    if [[ "$dry_run_mode" == "true" ]]; then
        dry_run
        exit 0
    fi
    
    # 执行同步步骤
    local start_time=$(date +%s)
    
    if [[ "$step1_only" == "true" ]]; then
        sync_host2_to_host1
    elif [[ "$step2_only" == "true" ]]; then
        sync_host1_to_local
    elif [[ "$step3_only" == "true" ]]; then
        sync_local_to_host3
    else
        # 执行完整流程
        sync_host2_to_host1
        sync_host1_to_local
        sync_local_to_host3
        cleanup_temp_files
    fi
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    show_summary
    log_info "总耗时: ${duration} 秒"
    log_info "🎉 同步任务完成!"
}

# 捕获错误
trap 'log_error "脚本执行失败，退出码: $?"; exit 1' ERR

# 执行主程序
main "$@"
