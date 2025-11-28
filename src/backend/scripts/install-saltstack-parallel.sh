#!/bin/bash
# =============================================================================
# SaltStack Minion 并行批量安装脚本
# 在多台主机上并行安装 SaltStack Minion
# =============================================================================
# 用法: ./install-saltstack-parallel.sh [OPTIONS] <hosts_file>
# 示例: ./install-saltstack-parallel.sh -u ubuntu -i ~/.ssh/id_rsa --master 192.168.1.10 hosts.txt
# =============================================================================

set -eo pipefail

# 获取脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 默认配置
SSH_USER="root"
SSH_PORT="22"
SSH_KEY=""
SSH_PASSWORD=""
SALT_MASTER="${SALT_MASTER:-salt-master}"
SALT_VERSION="${SALT_VERSION:-3007.1}"
APPHUB_URL="${APPHUB_URL:-}"
CONCURRENT=5
HOSTS_FILE=""
LOG_DIR="/tmp/saltstack-install-logs"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}[INFO]${NC} $*"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
log_error() { echo -e "${RED}[ERROR]${NC} $*"; }
log_progress() { echo -e "${BLUE}[PROGRESS]${NC} $*"; }

usage() {
    cat <<EOF
SaltStack Minion 并行批量安装脚本

用法: $0 [OPTIONS] <hosts_file>

选项:
    -h, --help              显示帮助信息
    -u, --user USER         SSH 用户名 (默认: root)
    -p, --port PORT         SSH 端口 (默认: 22)
    -i, --identity KEY_PATH SSH 私钥路径
    -P, --password PASSWORD SSH 密码 (不推荐用于生产环境)
    -c, --concurrent COUNT  并发安装数量 (默认: 5)
    --master MASTER         SaltStack Master 地址 (默认: salt-master)
    --apphub-url URL        AppHub URL (默认: 使用 APPHUB_URL 环境变量)
    --salt-version VERSION  SaltStack 版本 (默认: 3007.1)

hosts_file 格式:
    每行一个主机，格式如下:
    hostname_or_ip[:port] [minion_id]
    
    以 # 开头的行为注释

示例 hosts_file:
    # 简单主机条目
    192.168.1.100
    
    # 自定义 Minion ID
    192.168.1.101 worker01
    
    # 自定义端口
    192.168.1.102:2222
    
    # 自定义端口和 Minion ID
    192.168.1.103:2201 worker03

示例:
    # 使用密钥认证批量安装
    $0 -u ubuntu -i ~/.ssh/id_rsa --master 192.168.1.10 -c 10 hosts.txt
    
    # 使用密码认证批量安装
    $0 -u centos -P password123 --master 192.168.1.10 hosts.txt
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
            -c|--concurrent)
                CONCURRENT="$2"
                shift 2
                ;;
            --master)
                SALT_MASTER="$2"
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
                HOSTS_FILE="$1"
                shift
                ;;
        esac
    done

    if [[ -z "$HOSTS_FILE" ]]; then
        log_error "未指定 hosts 文件"
        usage
    fi
    
    if [[ ! -f "$HOSTS_FILE" ]]; then
        log_error "hosts 文件不存在: $HOSTS_FILE"
        exit 1
    fi
}

# 解析主机文件
parse_hosts_file() {
    local hosts_file="$1"
    local hosts=()
    
    while IFS= read -r line || [[ -n "$line" ]]; do
        # 跳过空行和注释
        [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]] && continue
        
        # 去除前后空格
        line=$(echo "$line" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        
        # 解析格式: host[:port] [minion_id]
        local host port minion_id
        
        # 提取主机和端口
        if [[ "$line" =~ ^([^[:space:]:]+):([0-9]+)[[:space:]]*(.*)$ ]]; then
            host="${BASH_REMATCH[1]}"
            port="${BASH_REMATCH[2]}"
            minion_id="${BASH_REMATCH[3]}"
        elif [[ "$line" =~ ^([^[:space:]:]+)[[:space:]]*(.*)$ ]]; then
            host="${BASH_REMATCH[1]}"
            port="$SSH_PORT"
            minion_id="${BASH_REMATCH[2]}"
        else
            log_warn "无法解析行: $line"
            continue
        fi
        
        # 去除 minion_id 的前后空格
        minion_id=$(echo "$minion_id" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
        
        hosts+=("$host:$port:$minion_id")
    done < "$hosts_file"
    
    echo "${hosts[@]}"
}

# 安装单个主机
install_single_host() {
    local host_entry="$1"
    local log_file="$2"
    
    # 解析 host:port:minion_id
    IFS=':' read -r host port minion_id <<< "$host_entry"
    
    local install_script="$SCRIPT_DIR/install-saltstack-remote.sh"
    
    if [[ ! -x "$install_script" ]]; then
        log_error "安装脚本不存在或不可执行: $install_script"
        return 1
    fi
    
    # 构建参数
    local args=()
    args+=("-u" "$SSH_USER")
    args+=("-p" "$port")
    args+=("--master" "$SALT_MASTER")
    args+=("--salt-version" "$SALT_VERSION")
    
    [[ -n "$SSH_KEY" ]] && args+=("-i" "$SSH_KEY")
    [[ -n "$SSH_PASSWORD" ]] && args+=("-P" "$SSH_PASSWORD")
    [[ -n "$APPHUB_URL" ]] && args+=("--apphub-url" "$APPHUB_URL")
    [[ -n "$minion_id" ]] && args+=("--minion-id" "$minion_id")
    
    args+=("$host")
    
    # 执行安装
    "$install_script" "${args[@]}" >> "$log_file" 2>&1
    return $?
}

# 并行安装
parallel_install() {
    local hosts=("$@")
    local total=${#hosts[@]}
    local success=0
    local failed=0
    local pids=()
    local host_pids=()
    
    # 创建日志目录
    mkdir -p "$LOG_DIR"
    local timestamp=$(date +%Y%m%d_%H%M%S)
    
    log_info "开始并行安装 SaltStack Minion"
    log_info "总计主机数: $total"
    log_info "并发数: $CONCURRENT"
    log_info "日志目录: $LOG_DIR"
    echo ""
    
    local current=0
    for host_entry in "${hosts[@]}"; do
        ((current++))
        
        # 解析主机信息
        IFS=':' read -r host port minion_id <<< "$host_entry"
        local log_file="$LOG_DIR/${host}_${timestamp}.log"
        
        log_progress "[$current/$total] 开始安装: $host (端口: $port)"
        
        # 后台执行安装
        (
            if install_single_host "$host_entry" "$log_file"; then
                echo "SUCCESS:$host" >> "$LOG_DIR/results_${timestamp}.txt"
            else
                echo "FAILED:$host" >> "$LOG_DIR/results_${timestamp}.txt"
            fi
        ) &
        
        pids+=($!)
        host_pids+=("$host:$!")
        
        # 控制并发数
        if [[ ${#pids[@]} -ge $CONCURRENT ]]; then
            # 等待任一进程完成
            wait -n 2>/dev/null || true
            
            # 清理已完成的进程
            local new_pids=()
            for pid in "${pids[@]}"; do
                if kill -0 "$pid" 2>/dev/null; then
                    new_pids+=("$pid")
                fi
            done
            pids=("${new_pids[@]}")
        fi
    done
    
    # 等待所有剩余进程完成
    log_info "等待所有安装任务完成..."
    for pid in "${pids[@]}"; do
        wait "$pid" 2>/dev/null || true
    done
    
    # 统计结果
    echo ""
    log_info "=========================================="
    log_info "安装完成统计"
    log_info "=========================================="
    
    if [[ -f "$LOG_DIR/results_${timestamp}.txt" ]]; then
        success=$(grep -c "^SUCCESS:" "$LOG_DIR/results_${timestamp}.txt" 2>/dev/null || echo 0)
        failed=$(grep -c "^FAILED:" "$LOG_DIR/results_${timestamp}.txt" 2>/dev/null || echo 0)
        
        echo ""
        log_info "成功: $success / $total"
        [[ $failed -gt 0 ]] && log_error "失败: $failed / $total"
        
        if [[ $failed -gt 0 ]]; then
            echo ""
            log_error "以下主机安装失败:"
            grep "^FAILED:" "$LOG_DIR/results_${timestamp}.txt" | cut -d: -f2 | while read -r h; do
                echo "  - $h (日志: $LOG_DIR/${h}_${timestamp}.log)"
            done
        fi
        
        if [[ $success -gt 0 ]]; then
            echo ""
            log_info "安装成功的主机:"
            grep "^SUCCESS:" "$LOG_DIR/results_${timestamp}.txt" | cut -d: -f2 | while read -r h; do
                echo "  - $h"
            done
        fi
    fi
    
    echo ""
    log_info "详细日志目录: $LOG_DIR"
    log_info "=========================================="
    
    # 返回是否全部成功
    [[ $failed -eq 0 ]]
}

# 主函数
main() {
    parse_args "$@"
    
    log_info "=========================================="
    log_info "SaltStack Minion 批量安装"
    log_info "=========================================="
    log_info "Salt Master: $SALT_MASTER"
    log_info "Salt Version: $SALT_VERSION"
    log_info "SSH 用户: $SSH_USER"
    log_info "并发数: $CONCURRENT"
    [[ -n "$APPHUB_URL" ]] && log_info "AppHub URL: $APPHUB_URL"
    echo ""
    
    # 解析主机文件
    log_info "解析主机文件: $HOSTS_FILE"
    hosts_str=$(parse_hosts_file "$HOSTS_FILE")
    
    if [[ -z "$hosts_str" ]]; then
        log_error "主机文件为空或格式错误"
        exit 1
    fi
    
    # 转换为数组
    read -ra hosts <<< "$hosts_str"
    
    log_info "解析到 ${#hosts[@]} 台主机"
    echo ""
    
    # 并行安装
    if parallel_install "${hosts[@]}"; then
        log_info "所有主机安装成功!"
        exit 0
    else
        log_error "部分主机安装失败，请检查日志"
        exit 1
    fi
}

main "$@"
