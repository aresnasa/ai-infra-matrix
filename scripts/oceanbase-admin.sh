#!/bin/bash
# OceanBase 数据库管理脚本
# 用于管理 OceanBase 容器的启动、停止、备份、恢复等操作
# 参考: https://github.com/oceanbase/oceanbase
# 参考: https://github.com/oceanbase/docker-images/blob/main/oceanbase-ce/README.md

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 容器名称
CONTAINER_NAME="ai-infra-oceanbase"

# 帮助信息
function show_help() {
    cat << EOF
OceanBase 数据库管理脚本

用法: $0 <command> [options]

命令:
  start           启动 OceanBase 容器
  stop            停止 OceanBase 容器
  restart         重启 OceanBase 容器
  status          查看 OceanBase 状态
  logs            查看 OceanBase 日志
  connect         连接到 OceanBase (sys 租户)
  connect-tenant  连接到 OceanBase (MySQL 租户)
  exec            在容器中执行命令
  sysbench        运行 sysbench 性能测试
  backup          备份 OceanBase 数据
  restore         恢复 OceanBase 数据
  clean           清理 OceanBase 数据（危险操作！）
  info            显示 OceanBase 配置信息
  help            显示此帮助信息

示例:
  $0 start                  # 启动 OceanBase
  $0 connect                # 连接到 sys 租户
  $0 connect-tenant         # 连接到 MySQL 租户
  $0 logs -f                # 实时查看日志
  $0 exec "obd cluster list" # 执行 OBD 命令
  $0 sysbench               # 运行性能测试

EOF
}

# 打印信息
function info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

function success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

function warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

function error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查容器是否存在
function check_container() {
    if ! docker ps -a --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
        error "容器 ${CONTAINER_NAME} 不存在"
        info "请先运行: docker-compose up -d oceanbase"
        exit 1
    fi
}

# 检查容器是否运行
function is_running() {
    docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"
}

# 启动 OceanBase
function start_oceanbase() {
    info "启动 OceanBase 容器..."
    docker-compose up -d oceanbase
    
    info "等待 OceanBase 启动完成（这可能需要 3-5 分钟）..."
    local count=0
    local max_attempts=60
    
    while [ $count -lt $max_attempts ]; do
        if docker logs ${CONTAINER_NAME} 2>&1 | grep -q "boot success"; then
            success "OceanBase 启动成功！"
            show_connection_info
            return 0
        fi
        
        echo -n "."
        sleep 5
        count=$((count + 1))
    done
    
    error "OceanBase 启动超时，请检查日志: docker logs ${CONTAINER_NAME}"
    exit 1
}

# 停止 OceanBase
function stop_oceanbase() {
    info "停止 OceanBase 容器..."
    docker-compose stop oceanbase
    success "OceanBase 已停止"
}

# 重启 OceanBase
function restart_oceanbase() {
    info "重启 OceanBase 容器..."
    stop_oceanbase
    start_oceanbase
}

# 查看状态
function show_status() {
    check_container
    
    echo -e "\n${BLUE}=== OceanBase 容器状态 ===${NC}"
    docker ps -a --filter "name=${CONTAINER_NAME}" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
    
    if is_running; then
        echo -e "\n${BLUE}=== OceanBase 进程状态 ===${NC}"
        docker exec ${CONTAINER_NAME} pgrep -a observer || echo "observer 进程未运行"
        
        echo -e "\n${BLUE}=== OceanBase 端口监听 ===${NC}"
        docker exec ${CONTAINER_NAME} netstat -tln | grep 2881 || echo "端口 2881 未监听"
        
        echo -e "\n${BLUE}=== OceanBase 启动日志 ===${NC}"
        if docker logs ${CONTAINER_NAME} 2>&1 | tail -20 | grep -q "boot success"; then
            success "OceanBase 已成功启动"
        else
            warning "OceanBase 可能仍在启动中"
        fi
    fi
}

# 查看日志
function show_logs() {
    check_container
    info "查看 OceanBase 日志..."
    docker logs "$@" ${CONTAINER_NAME}
}

# 连接到 OceanBase (sys 租户)
function connect_sys() {
    check_container
    
    if ! is_running; then
        error "容器未运行，请先启动: $0 start"
        exit 1
    fi
    
    info "连接到 OceanBase sys 租户..."
    info "用户名: root (sys 租户)"
    info "如需退出，输入: exit"
    echo ""
    
    docker exec -it ${CONTAINER_NAME} obclient -h127.0.0.1 -P2881 -uroot
}

# 连接到 OceanBase (MySQL 租户)
function connect_tenant() {
    check_container
    
    if ! is_running; then
        error "容器未运行，请先启动: $0 start"
        exit 1
    fi
    
    # 获取租户名称
    local tenant_name="${OB_TENANT_NAME:-ai_infra}"
    
    info "连接到 OceanBase MySQL 租户: ${tenant_name}"
    info "用户名: root@${tenant_name}"
    info "如需退出，输入: exit"
    echo ""
    
    docker exec -it ${CONTAINER_NAME} obclient -h127.0.0.1 -P2881 -uroot@${tenant_name}
}

# 执行命令
function exec_command() {
    check_container
    
    if ! is_running; then
        error "容器未运行，请先启动: $0 start"
        exit 1
    fi
    
    info "执行命令: $@"
    docker exec -it ${CONTAINER_NAME} "$@"
}

# 运行 sysbench 测试
function run_sysbench() {
    check_container
    
    if ! is_running; then
        error "容器未运行，请先启动: $0 start"
        exit 1
    fi
    
    info "运行 sysbench 性能测试..."
    info "这可能需要几分钟时间..."
    
    docker exec -it ${CONTAINER_NAME} obd test sysbench obcluster
}

# 备份数据
function backup_data() {
    check_container
    
    local backup_dir="./backup/oceanbase/$(date +%Y%m%d_%H%M%S)"
    
    info "备份 OceanBase 数据到: ${backup_dir}"
    mkdir -p "${backup_dir}"
    
    # 备份数据目录
    if [ -d "./data/oceanbase/ob" ]; then
        info "备份数据目录..."
        cp -r ./data/oceanbase/ob "${backup_dir}/"
    fi
    
    # 备份配置目录
    if [ -d "./data/oceanbase/obd" ]; then
        info "备份配置目录..."
        cp -r ./data/oceanbase/obd "${backup_dir}/"
    fi
    
    # 导出数据库 schema（如果容器在运行）
    if is_running; then
        info "导出数据库 schema..."
        docker exec ${CONTAINER_NAME} obclient -h127.0.0.1 -P2881 -uroot -e "SHOW DATABASES;" > "${backup_dir}/databases.sql" 2>/dev/null || true
    fi
    
    success "备份完成: ${backup_dir}"
}

# 恢复数据
function restore_data() {
    if [ -z "$1" ]; then
        error "请指定备份目录"
        info "用法: $0 restore <backup_dir>"
        exit 1
    fi
    
    local backup_dir="$1"
    
    if [ ! -d "${backup_dir}" ]; then
        error "备份目录不存在: ${backup_dir}"
        exit 1
    fi
    
    warning "此操作将覆盖现有数据！"
    read -p "确认恢复数据？(yes/no): " confirm
    
    if [ "$confirm" != "yes" ]; then
        info "操作已取消"
        exit 0
    fi
    
    # 停止容器
    if is_running; then
        info "停止 OceanBase 容器..."
        docker-compose stop oceanbase
    fi
    
    # 恢复数据
    info "恢复数据目录..."
    if [ -d "${backup_dir}/ob" ]; then
        rm -rf ./data/oceanbase/ob
        cp -r "${backup_dir}/ob" ./data/oceanbase/
    fi
    
    info "恢复配置目录..."
    if [ -d "${backup_dir}/obd" ]; then
        rm -rf ./data/oceanbase/obd
        cp -r "${backup_dir}/obd" ./data/oceanbase/
    fi
    
    success "数据恢复完成"
    info "请启动容器: $0 start"
}

# 清理数据
function clean_data() {
    warning "此操作将删除所有 OceanBase 数据！"
    read -p "确认清理数据？(yes/no): " confirm
    
    if [ "$confirm" != "yes" ]; then
        info "操作已取消"
        exit 0
    fi
    
    # 停止并删除容器
    info "停止并删除容器..."
    docker-compose down oceanbase || true
    
    # 删除数据目录
    info "删除数据目录..."
    rm -rf ./data/oceanbase/ob
    rm -rf ./data/oceanbase/obd
    
    success "数据清理完成"
    info "如需重新启动，请运行: $0 start"
}

# 显示配置信息
function show_info() {
    echo -e "\n${BLUE}=== OceanBase 配置信息 ===${NC}\n"
    
    # 从 .env 文件读取配置
    if [ -f ".env" ]; then
        echo "镜像版本: $(grep OCEANBASE_VERSION .env | cut -d= -f2)"
        echo "运行模式: $(grep OB_MODE .env | cut -d= -f2)"
        echo "集群名称: $(grep OB_CLUSTER_NAME .env | cut -d= -f2)"
        echo "租户名称: $(grep OB_TENANT_NAME .env | cut -d= -f2)"
        echo "端口: $(grep OB_PORT .env | cut -d= -f2)"
        echo "内存限制: $(grep OB_MEMORY_LIMIT= .env | cut -d= -f2)"
        echo "数据目录: $(grep OB_DATA_DIR .env | cut -d= -f2)"
    else
        warning ".env 文件不存在，使用默认配置"
    fi
    
    echo ""
    show_connection_info
}

# 显示连接信息
function show_connection_info() {
    echo -e "\n${GREEN}=== 连接信息 ===${NC}\n"
    echo "sys 租户连接:"
    echo "  docker exec -it ${CONTAINER_NAME} obclient -h127.0.0.1 -P2881 -uroot"
    echo ""
    echo "MySQL 租户连接:"
    echo "  docker exec -it ${CONTAINER_NAME} obclient -h127.0.0.1 -P2881 -uroot@${OB_TENANT_NAME:-ai_infra}"
    echo ""
    echo "或使用快捷命令:"
    echo "  $0 connect          # sys 租户"
    echo "  $0 connect-tenant   # MySQL 租户"
    echo ""
}

# 主函数
function main() {
    case "${1:-help}" in
        start)
            start_oceanbase
            ;;
        stop)
            stop_oceanbase
            ;;
        restart)
            restart_oceanbase
            ;;
        status)
            show_status
            ;;
        logs)
            shift
            show_logs "$@"
            ;;
        connect)
            connect_sys
            ;;
        connect-tenant)
            connect_tenant
            ;;
        exec)
            shift
            exec_command "$@"
            ;;
        sysbench)
            run_sysbench
            ;;
        backup)
            backup_data
            ;;
        restore)
            shift
            restore_data "$@"
            ;;
        clean)
            clean_data
            ;;
        info)
            show_info
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            error "未知命令: $1"
            show_help
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"
