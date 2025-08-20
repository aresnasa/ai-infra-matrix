#!/bin/bash

# AI基础设施矩阵 - Docker构建脚本
# 包含JupyterHub统一认证集成

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查Docker和Docker Compose
check_dependencies() {
    log_info "检查依赖项..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker未安装，请先安装Docker"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose未安装，请先安装Docker Compose"
        exit 1
    fi
    
    log_success "依赖项检查完成"
}

# 构建所有镜像
build_images() {
    log_info "开始构建Docker镜像..."
    
    # 进入项目根目录
    cd "$(dirname "$0")"
    
    # 构建JupyterHub镜像
    log_info "构建JupyterHub镜像..."
    docker build -f Dockerfile.jupyterhub -t ai-infra-jupyterhub:latest .
    
    # 使用docker-compose构建其他服务
    log_info "构建其他服务镜像..."
    cd src
    docker-compose build --no-cache
    
    log_success "所有镜像构建完成"
}

# 启动服务
start_services() {
    log_info "启动AI基础设施矩阵服务..."
    
    cd src
    
    # 启动核心服务
    docker-compose up -d postgres redis openldap
    
    # 等待数据库服务启动
    log_info "等待数据库服务启动..."
    sleep 30
    
    # 启动后端服务
    log_info "启动后端服务..."
    docker-compose up -d backend
    
    # 等待后端服务启动
    log_info "等待后端服务启动..."
    sleep 20
    
    # 启动前端和JupyterHub
    log_info "启动前端和JupyterHub服务..."
    docker-compose up -d frontend jupyterhub
    
    log_success "所有服务启动完成"
}

# 显示服务状态
show_status() {
    log_info "服务状态："
    cd src
    docker-compose ps
    
    echo ""
    log_info "服务访问地址："
    echo "  前端界面:      http://localhost:3001"
    echo "  后端API:       http://localhost:8082"
    echo "  JupyterHub:    http://localhost:8090"
    echo "  LDAP管理:      http://localhost:8081"
    echo "  PostgreSQL:    localhost:5433"
    echo "  Redis:         localhost:6379"
}

# 停止服务
stop_services() {
    log_info "停止所有服务..."
    cd src
    docker-compose down
    log_success "所有服务已停止"
}

# 清理数据
clean_data() {
    log_warning "这将删除所有持久化数据，是否继续？(y/N)"
    read -r response
    if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
        log_info "清理数据..."
        cd src
        docker-compose down -v
        docker system prune -f
        log_success "数据清理完成"
    else
        log_info "取消清理操作"
    fi
}

# 查看日志
view_logs() {
    local service=$1
    if [ -z "$service" ]; then
        log_info "可用服务: postgres, redis, openldap, backend, frontend, jupyterhub"
        log_info "使用方法: $0 logs <service_name>"
        return 1
    fi
    
    log_info "查看 $service 服务日志..."
    cd src
    docker-compose logs -f "$service"
}

# 重建特定服务
rebuild_service() {
    local service=$1
    if [ -z "$service" ]; then
        log_info "可用服务: backend, frontend, jupyterhub"
        log_info "使用方法: $0 rebuild <service_name>"
        return 1
    fi
    
    log_info "重建 $service 服务..."
    cd src
    docker-compose stop "$service"
    docker-compose build --no-cache "$service"
    docker-compose up -d "$service"
    log_success "$service 服务重建完成"
}

# 主函数
main() {
    case "$1" in
        "build")
            check_dependencies
            build_images
            ;;
        "start")
            check_dependencies
            start_services
            show_status
            ;;
        "stop")
            stop_services
            ;;
        "restart")
            stop_services
            start_services
            show_status
            ;;
        "status")
            show_status
            ;;
        "logs")
            view_logs "$2"
            ;;
        "rebuild")
            rebuild_service "$2"
            ;;
        "clean")
            clean_data
            ;;
        "full")
            check_dependencies
            build_images
            start_services
            show_status
            ;;
        *)
            echo "AI基础设施矩阵 - Docker管理脚本"
            echo ""
            echo "用法: $0 {build|start|stop|restart|status|logs|rebuild|clean|full}"
            echo ""
            echo "命令说明："
            echo "  build      - 构建所有Docker镜像"
            echo "  start      - 启动所有服务"
            echo "  stop       - 停止所有服务"
            echo "  restart    - 重启所有服务"
            echo "  status     - 显示服务状态"
            echo "  logs <srv> - 查看指定服务日志"
            echo "  rebuild <srv> - 重建指定服务"
            echo "  clean      - 清理所有数据"
            echo "  full       - 完整构建并启动"
            echo ""
            echo "示例："
            echo "  $0 full                    # 完整部署"
            echo "  $0 logs jupyterhub         # 查看JupyterHub日志"
            echo "  $0 rebuild backend         # 重建后端服务"
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"
