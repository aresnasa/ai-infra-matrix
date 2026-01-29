#!/bin/bash
# ==============================================================================
# Nightingale (N9E) 告警配置自动化工具
# 
# 用途：自动化配置 Nightingale 的监控和告警规则
# 
# 使用方法:
#   ./scripts/n9e-setup.sh init              # 初始化监控配置
#   ./scripts/n9e-setup.sh status            # 查看系统状态
#   ./scripts/n9e-setup.sh import <file>     # 导入告警规则
#   ./scripts/n9e-setup.sh export <file>     # 导出告警规则
#   ./scripts/n9e-setup.sh add-rule <name> <promql> [severity]
#   ./scripts/n9e-setup.sh list-groups       # 列出业务组
#   ./scripts/n9e-setup.sh list-rules <gid>  # 列出告警规则
#   ./scripts/n9e-setup.sh help              # 显示帮助
# ==============================================================================

set -e

# 脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 加载 .env 配置
if [ -f "$PROJECT_DIR/.env" ]; then
    set -a
    source "$PROJECT_DIR/.env"
    set +a
fi

# 默认配置 - 使用 Service API (Basic Auth)
# 通过 nginx 代理访问 /v1/n9e/* 端点
N9E_HOST="${N9E_HOST:-${NIGHTINGALE_HOST:-localhost}}"
N9E_PORT="${N9E_PORT:-80}"  # 通过 nginx 访问
N9E_API_USER="${N9E_API_USER:-n9e-api}"  # Service API 用户名
N9E_API_PASSWORD="${N9E_API_PASSWORD:-123456}"  # Service API 密码
N9E_API_MODE="${N9E_API_MODE:-service}"  # 使用 service API 模式

# Python 脚本路径
PYTHON_SCRIPT="$SCRIPT_DIR/n9e-alert-config.py"

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_debug() {
    if [ "$DEBUG" = "true" ]; then
        echo -e "${BLUE}[DEBUG]${NC} $1"
    fi
}

# 检查依赖
check_dependencies() {
    log_info "检查依赖..."
    
    # 检查 Python
    if ! command -v python3 &> /dev/null; then
        log_error "Python3 未安装"
        exit 1
    fi
    
    # 检查必要的 Python 包
    python3 -c "import requests" 2>/dev/null || {
        log_warn "安装 Python 依赖..."
        pip3 install requests pyyaml python-dotenv
    }
    
    # 检查 Python 脚本是否存在
    if [ ! -f "$PYTHON_SCRIPT" ]; then
        log_error "Python 脚本不存在: $PYTHON_SCRIPT"
        exit 1
    fi
    
    log_info "依赖检查完成"
}

# 等待 N9E 服务就绪
wait_for_n9e() {
    local max_retries=${1:-30}
    local retry=0
    
    log_info "等待 Nightingale 服务就绪..."
    log_debug "检查地址: http://${N9E_HOST}:${N9E_PORT}/api/n9e/versions"
    
    while [ $retry -lt $max_retries ]; do
        # 尝试访问版本接口（不需要认证）
        if curl -s "http://${N9E_HOST}:${N9E_PORT}/api/n9e/versions" > /dev/null 2>&1; then
            log_info "Nightingale 服务已就绪"
            return 0
        fi
        # 也尝试检查 v1 API 端点
        if curl -s -u "${N9E_API_USER}:${N9E_API_PASSWORD}" "http://${N9E_HOST}:${N9E_PORT}/v1/n9e/busi-groups" > /dev/null 2>&1; then
            log_info "Nightingale Service API 已就绪"
            return 0
        fi
        retry=$((retry + 1))
        log_debug "等待中... ($retry/$max_retries)"
        sleep 2
    done
    
    log_error "Nightingale 服务未就绪"
    return 1
}

# 运行 Python 脚本
run_python() {
    # 设置环境变量给 Python 脚本
    export N9E_HOST="$N9E_HOST"
    export N9E_PORT="$N9E_PORT"
    export N9E_API_USER="$N9E_API_USER"
    export N9E_API_PASSWORD="$N9E_API_PASSWORD"
    export N9E_API_MODE="$N9E_API_MODE"
    
    python3 "$PYTHON_SCRIPT" \
        --host "$N9E_HOST" \
        --port "$N9E_PORT" \
        --username "$N9E_API_USER" \
        --password "$N9E_API_PASSWORD" \
        "$@"
}

# 初始化监控配置
cmd_init() {
    local group_name="${1:-Default BusiGroup}"
    
    log_info "初始化监控配置..."
    log_info "业务组名称: $group_name"
    
    wait_for_n9e || exit 1
    
    run_python init --group-name "$group_name"
    
    log_info "初始化完成"
}

# 查看系统状态
cmd_status() {
    wait_for_n9e || exit 1
    run_python status
}

# 导入告警规则
cmd_import() {
    local file="$1"
    local group_id="$2"
    
    if [ -z "$file" ]; then
        log_error "请指定要导入的文件"
        echo "用法: $0 import <file> <group_id>"
        exit 1
    fi
    
    if [ -z "$group_id" ]; then
        log_error "请指定业务组 ID"
        echo "用法: $0 import <file> <group_id>"
        echo "使用 '$0 list-groups' 查看业务组列表"
        exit 1
    fi
    
    if [ ! -f "$file" ]; then
        log_error "文件不存在: $file"
        exit 1
    fi
    
    log_info "导入告警规则: $file -> 业务组 $group_id"
    
    wait_for_n9e || exit 1
    run_python import-rules --file "$file" --group-id "$group_id"
}

# 导出告警规则
cmd_export() {
    local file="$1"
    local group_id="$2"
    
    if [ -z "$file" ]; then
        log_error "请指定输出文件"
        echo "用法: $0 export <file> <group_id>"
        exit 1
    fi
    
    if [ -z "$group_id" ]; then
        log_error "请指定业务组 ID"
        echo "用法: $0 export <file> <group_id>"
        echo "使用 '$0 list-groups' 查看业务组列表"
        exit 1
    fi
    
    log_info "导出告警规则: 业务组 $group_id -> $file"
    
    wait_for_n9e || exit 1
    run_python export-rules --group-id "$group_id" --output "$file"
}

# 添加告警规则
cmd_add_rule() {
    local name="$1"
    local promql="$2"
    local severity="${3:-2}"
    local group_id="${4:-1}"
    
    if [ -z "$name" ] || [ -z "$promql" ]; then
        log_error "请指定规则名称和 PromQL"
        echo "用法: $0 add-rule <name> <promql> [severity] [group_id]"
        echo "  severity: 1=紧急, 2=警告(默认), 3=通知"
        exit 1
    fi
    
    log_info "添加告警规则: $name"
    
    wait_for_n9e || exit 1
    run_python add-rule --name "$name" --prom-ql "$promql" --severity "$severity" --group-id "$group_id"
}

# 列出业务组
cmd_list_groups() {
    wait_for_n9e || exit 1
    run_python list-groups
}

# 列出告警规则
cmd_list_rules() {
    local group_id="$1"
    
    if [ -z "$group_id" ]; then
        log_error "请指定业务组 ID"
        echo "用法: $0 list-rules <group_id>"
        echo "使用 '$0 list-groups' 查看业务组列表"
        exit 1
    fi
    
    wait_for_n9e || exit 1
    run_python list-rules --group-id "$group_id"
}

# 添加预设告警规则
cmd_add_preset() {
    local type="${1:-all}"
    local group_id="${2:-1}"
    local threshold="$3"
    
    log_info "添加预设告警规则: 类型=$type, 业务组=$group_id"
    
    wait_for_n9e || exit 1
    
    if [ -n "$threshold" ]; then
        run_python add-preset --group-id "$group_id" --type "$type" --threshold "$threshold"
    else
        run_python add-preset --group-id "$group_id" --type "$type"
    fi
}

# 完整安装（Categraf + 告警规则）
cmd_full_setup() {
    local group_name="${1:-AI Infrastructure}"
    
    log_info "开始完整安装..."
    
    # 1. 检查依赖
    check_dependencies
    
    # 2. 等待服务就绪
    wait_for_n9e || exit 1
    
    # 3. 初始化业务组和基础告警规则
    log_info "步骤 1/3: 初始化业务组..."
    run_python init --group-name "$group_name"
    
    # 4. 获取业务组 ID
    log_info "步骤 2/3: 获取业务组信息..."
    local group_info
    group_info=$(run_python list-groups 2>/dev/null | grep "$group_name" | head -1)
    local group_id
    group_id=$(echo "$group_info" | grep -oP 'ID:\s*\K\d+')
    
    if [ -z "$group_id" ]; then
        log_warn "无法获取业务组 ID，使用默认值 1"
        group_id=1
    fi
    
    log_info "业务组 ID: $group_id"
    
    # 5. 导入示例告警规则
    log_info "步骤 3/3: 导入告警规则..."
    local rules_file="$PROJECT_DIR/config/n9e-alert-rules-example.yaml"
    if [ -f "$rules_file" ]; then
        run_python import-rules --file "$rules_file" --group-id "$group_id" || true
    else
        log_warn "示例规则文件不存在，跳过导入"
    fi
    
    # 6. 显示状态
    log_info "安装完成！"
    echo ""
    run_python status
    
    echo ""
    log_info "下一步："
    echo "  1. 访问 Nightingale: http://${N9E_HOST}:${N9E_PORT}"
    echo "  2. 默认账号: ${N9E_USERNAME}"
    echo "  3. 配置通知渠道（邮件、钉钉、企业微信等）"
    echo "  4. 部署 Categraf 到需要监控的主机"
}

# 显示帮助
show_help() {
    cat << EOF
Nightingale (N9E) 告警配置自动化工具

用法:
  $(basename "$0") <command> [options]

命令:
  init [group_name]           初始化监控配置
  status                      查看系统状态
  import <file> <group_id>    导入告警规则
  export <file> <group_id>    导出告警规则
  add-rule <name> <promql> [severity] [group_id]
                              添加告警规则
  add-preset [type] [group_id] [threshold]
                              添加预设告警规则
  list-groups                 列出业务组
  list-rules <group_id>       列出告警规则
  full-setup [group_name]     完整安装（推荐）
  help                        显示帮助信息

预设规则类型 (add-preset):
  all       所有预设规则
  cpu       CPU使用率告警
  memory    内存使用率告警
  disk      磁盘使用率告警
  host      主机宕机告警
  network   网络错误告警
  load      系统负载告警
  diskio    磁盘IO告警
  docker    Docker容器告警

示例:
  # 完整安装（推荐）
  $(basename "$0") full-setup "AI Infrastructure"

  # 初始化监控
  $(basename "$0") init "My Business Group"

  # 添加告警规则
  $(basename "$0") add-rule "CPU告警" 'cpu_usage > 80' 2 1

  # 导入告警规则
  $(basename "$0") import config/n9e-alert-rules-example.yaml 1

  # 添加所有预设规则
  $(basename "$0") add-preset all 1

  # 添加CPU告警规则（阈值90%）
  $(basename "$0") add-preset cpu 1 90

环境变量:
  NIGHTINGALE_HOST     N9E 主机地址 (默认: localhost)
  NIGHTINGALE_PORT     N9E 端口 (默认: 17000)
  N9E_USERNAME         用户名 (默认: root)
  N9E_PASSWORD         密码 (默认: root.2020)
  DEBUG                启用调试模式 (true/false)

EOF
}

# 主入口
main() {
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        init)
            check_dependencies
            cmd_init "$@"
            ;;
        status)
            check_dependencies
            cmd_status
            ;;
        import)
            check_dependencies
            cmd_import "$@"
            ;;
        export)
            check_dependencies
            cmd_export "$@"
            ;;
        add-rule)
            check_dependencies
            cmd_add_rule "$@"
            ;;
        add-preset)
            check_dependencies
            cmd_add_preset "$@"
            ;;
        list-groups)
            check_dependencies
            cmd_list_groups
            ;;
        list-rules)
            check_dependencies
            cmd_list_rules "$@"
            ;;
        full-setup)
            cmd_full_setup "$@"
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            log_error "未知命令: $command"
            echo ""
            show_help
            exit 1
            ;;
    esac
}

main "$@"
