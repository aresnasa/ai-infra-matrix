#!/bin/bash

# AI-Infra-Matrix JupyterHub Docker管理脚本
# 使用Python 3.13和最新版本的JupyterHub

set -e

COMPOSE_FILE="src/docker-compose.yml"
PROJECT_NAME="ai-infra-matrix"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 帮助信息
show_help() {
    echo "AI-Infra-Matrix JupyterHub Docker管理脚本"
    echo ""
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  build               构建所有Docker镜像"
    echo "  build-jupyterhub    仅构建JupyterHub镜像"
    echo "  up                  启动所有服务"
    echo "  up-jupyterhub       仅启动JupyterHub相关服务"
    echo "  down                停止所有服务"
    echo "  restart             重启所有服务"
    echo "  restart-jupyterhub  重启JupyterHub服务"
    echo "  logs                显示所有服务日志"
    echo "  logs-jupyterhub     显示JupyterHub日志"
    echo "  status              显示服务状态"
    echo "  clean               清理所有容器和镜像"
    echo "  reset               重置所有数据和容器"
    echo "  health              检查服务健康状态"
    echo "  shell-jupyterhub    进入JupyterHub容器shell"
    echo "  -h, --help          显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0 build              # 构建所有镜像"
    echo "  $0 up-jupyterhub      # 启动JupyterHub和依赖服务"
    echo "  $0 logs-jupyterhub    # 查看JupyterHub日志"
}

# 检查Docker和docker-compose
check_requirements() {
    if ! command -v docker &> /dev/null; then
        print_error "Docker未安装或不在PATH中"
        exit 1
    fi

    if ! command -v docker-compose &> /dev/null; then
        print_error "docker-compose未安装或不在PATH中"
        exit 1
    fi

    if [[ ! -f "$COMPOSE_FILE" ]]; then
        print_error "docker-compose文件不存在: $COMPOSE_FILE"
        exit 1
    fi
}

# 构建所有镜像
build_all() {
    print_status "构建所有Docker镜像..."
    docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" build
    print_success "所有镜像构建完成"
}

# 仅构建JupyterHub镜像
build_jupyterhub() {
    print_status "构建JupyterHub镜像 (Python 3.13 + conda环境版本)..."
    docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" build jupyterhub
    print_success "JupyterHub镜像构建完成"
}

# 启动所有服务
start_all() {
    print_status "启动所有服务..."
    docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" up -d
    print_success "所有服务已启动"
    show_access_info
}

# 启动JupyterHub相关服务
start_jupyterhub() {
    print_status "启动JupyterHub和依赖服务..."
    docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" --profile jupyterhub up -d postgres redis backend jupyterhub
    print_success "JupyterHub服务已启动"
    show_jupyterhub_info
}

# 停止所有服务
stop_all() {
    print_status "停止所有服务..."
    docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" down
    print_success "所有服务已停止"
}

# 重启所有服务
restart_all() {
    print_status "重启所有服务..."
    docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" restart
    print_success "所有服务已重启"
}

# 重启JupyterHub服务
restart_jupyterhub() {
    print_status "重启JupyterHub服务..."
    docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" restart jupyterhub
    print_success "JupyterHub服务已重启"
}

# 显示日志
show_logs() {
    print_status "显示所有服务日志..."
    docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" logs -f
}

# 显示JupyterHub日志
show_jupyterhub_logs() {
    print_status "显示JupyterHub日志..."
    docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" logs -f jupyterhub
}

# 显示服务状态
show_status() {
    print_status "服务状态:"
    docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" ps
}

# 清理容器和镜像
clean_all() {
    print_warning "这将删除所有容器和镜像，确定继续吗? (y/N)"
    read -r response
    if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
        print_status "清理容器和镜像..."
        docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" down --rmi all --volumes --remove-orphans
        print_success "清理完成"
    else
        print_status "取消清理操作"
    fi
}

# 重置所有数据
reset_all() {
    print_warning "这将删除所有数据、容器和镜像，确定继续吗? (y/N)"
    read -r response
    if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
        print_status "重置所有数据..."
        docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" down --volumes --remove-orphans
        docker system prune -af --volumes
        print_success "重置完成"
    else
        print_status "取消重置操作"
    fi
}

# 健康检查
check_health() {
    print_status "检查服务健康状态..."
    
    services=("postgres" "redis" "backend" "jupyterhub")
    
    for service in "${services[@]}"; do
        if docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" ps | grep -q "$service.*Up.*healthy"; then
            print_success "$service: 健康"
        elif docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" ps | grep -q "$service.*Up"; then
            print_warning "$service: 运行中 (健康检查中)"
        else
            print_error "$service: 未运行或不健康"
        fi
    done
}

# 进入JupyterHub容器shell
shell_jupyterhub() {
    print_status "进入JupyterHub容器..."
    docker-compose -f "$COMPOSE_FILE" -p "$PROJECT_NAME" exec jupyterhub /bin/bash
}

# 显示访问信息
show_access_info() {
    echo ""
    print_success "=== 服务访问信息 ==="
    echo "Frontend:     http://localhost:3000"
    echo "Backend API:  http://localhost:8080"
    echo "JupyterHub:   http://localhost:8888"
    echo "PostgreSQL:   localhost:5433"
    echo "Redis:        localhost:6379"
    echo "OpenLDAP:     localhost:389"
    echo ""
}

# 显示JupyterHub信息
show_jupyterhub_info() {
    echo ""
    print_success "=== JupyterHub访问信息 ==="
    echo "JupyterHub:   http://localhost:8888"
    echo "管理面板:     http://localhost:8888/hub/admin"
    echo "API文档:      http://localhost:8888/hub/api"
    echo ""
    print_status "使用AI-Infra-Matrix账号登录即可自动跳转到JupyterHub"
    echo ""
}

# 主逻辑
main() {
    check_requirements
    
    case "${1:-}" in
        "build")
            build_all
            ;;
        "build-jupyterhub")
            build_jupyterhub
            ;;
        "up")
            start_all
            ;;
        "up-jupyterhub")
            start_jupyterhub
            ;;
        "down")
            stop_all
            ;;
        "restart")
            restart_all
            ;;
        "restart-jupyterhub")
            restart_jupyterhub
            ;;
        "logs")
            show_logs
            ;;
        "logs-jupyterhub")
            show_jupyterhub_logs
            ;;
        "status")
            show_status
            ;;
        "clean")
            clean_all
            ;;
        "reset")
            reset_all
            ;;
        "health")
            check_health
            ;;
        "shell-jupyterhub")
            shell_jupyterhub
            ;;
        "-h"|"--help"|"")
            show_help
            ;;
        *)
            print_error "未知选项: $1"
            echo ""
            show_help
            exit 1
            ;;
    esac
}

# 运行主函数
main "$@"
