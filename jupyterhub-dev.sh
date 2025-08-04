#!/bin/bash

# AI-Infra-Matrix JupyterHub 开发助手脚本
# 用于优化Python配置文件的开发和部署流程

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$SCRIPT_DIR"
JUPYTERHUB_DIR="$PROJECT_ROOT/src/jupyterhub"

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

# 显示帮助信息
show_help() {
    cat << EOF
AI-Infra-Matrix JupyterHub 开发助手

用法: $0 [命令] [选项]

命令:
  dev-restart    重启JupyterHub容器（使用volume挂载，无需重建）
  full-rebuild   完整重建镜像（当依赖变化时使用）
  logs          查看JupyterHub日志
  test-auth     测试认证端点
  status        检查容器状态
  help          显示此帮助信息

开发模式特性:
  - Python配置文件通过volume挂载，修改后只需重启容器
  - 避免每次都重新构建镜像，大大加快开发速度
  - 支持实时日志查看和调试

示例:
  $0 dev-restart          # 快速重启JupyterHub（推荐）
  $0 full-rebuild         # 完整重建（依赖变化时）
  $0 logs                 # 查看实时日志
  $0 test-auth           # 测试认证功能

EOF
}

# 切换到项目目录
cd "$PROJECT_ROOT"

# 检查Docker和docker-compose
check_prerequisites() {
    if ! command -v docker &> /dev/null; then
        log_error "Docker未安装或不在PATH中"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        log_error "docker-compose未安装或不在PATH中"
        exit 1
    fi
}

# 开发模式重启（仅重启容器，使用volume挂载的配置）
dev_restart() {
    log_info "开发模式重启JupyterHub..."
    log_info "使用volume挂载的配置文件，无需重建镜像"
    
    # 停止容器
    log_info "停止JupyterHub容器..."
    docker-compose -f docker-compose.yml stop jupyterhub || log_warning "容器可能已经停止"
    
    # 启动容器
    log_info "启动JupyterHub容器..."
    docker-compose -f docker-compose.yml up -d jupyterhub
    
    # 等待健康检查
    log_info "等待服务启动..."
    sleep 10
    
    # 检查状态
    if docker-compose -f docker-compose.yml ps jupyterhub | grep -q "Up"; then
        log_success "JupyterHub重启成功！"
        log_info "访问地址: http://localhost:8000"
        log_info "使用 '$0 logs' 查看日志"
    else
        log_error "JupyterHub启动失败"
        log_info "使用 '$0 logs' 查看错误日志"
        exit 1
    fi
}

# 完整重建（当依赖或Dockerfile变化时）
full_rebuild() {
    log_info "开始完整重建JupyterHub镜像..."
    log_warning "这可能需要几分钟时间"
    
    # 停止并删除容器
    log_info "停止并删除现有容器..."
    docker-compose -f docker-compose.yml stop jupyterhub || log_warning "容器可能已经停止"
    docker-compose -f docker-compose.yml rm -f jupyterhub || log_warning "容器可能不存在"
    
    # 重建镜像
    log_info "重建JupyterHub镜像（无缓存）..."
    docker-compose -f docker-compose.yml build --no-cache jupyterhub
    
    # 启动容器
    log_info "启动新容器..."
    docker-compose -f docker-compose.yml up -d jupyterhub
    
    # 等待健康检查
    log_info "等待服务启动..."
    sleep 15
    
    # 检查状态
    if docker-compose -f docker-compose.yml ps jupyterhub | grep -q "Up"; then
        log_success "JupyterHub重建成功！"
        log_info "访问地址: http://localhost:8000"
    else
        log_error "JupyterHub启动失败"
        log_info "使用 '$0 logs' 查看错误日志"
        exit 1
    fi
}

# 查看日志
show_logs() {
    log_info "显示JupyterHub日志（按Ctrl+C退出）..."
    docker-compose -f docker-compose.yml logs -f jupyterhub
}

# 测试认证端点
test_auth() {
    log_info "测试JupyterHub认证端点..."
    
    # 检查JupyterHub健康状态
    if docker-compose -f docker-compose.yml exec jupyterhub curl -s -f http://localhost:8000/hub/health > /dev/null; then
        log_success "JupyterHub健康检查通过"
    else
        log_error "JupyterHub健康检查失败"
        return 1
    fi
    
    # 测试后端连接
    log_info "测试后端连接..."
    if docker-compose -f docker-compose.yml exec jupyterhub curl -s -f http://backend:8082/api/health > /dev/null; then
        log_success "后端连接正常"
    else
        log_error "无法连接到后端"
        return 1
    fi
    
    # 测试JWT验证端点
    log_info "测试JWT验证端点..."
    RESPONSE=$(docker-compose -f docker-compose.yml exec jupyterhub curl -s -X POST \
        -H "Content-Type: application/json" \
        -d '{"token":"test-token"}' \
        http://backend:8082/api/auth/verify-token)
    
    if echo "$RESPONSE" | grep -q "error"; then
        log_success "JWT验证端点响应正常（预期的错误响应）"
    else
        log_warning "JWT验证端点响应异常"
    fi
    
    log_success "认证测试完成"
}

# 检查容器状态
show_status() {
    log_info "检查容器状态..."
    echo ""
    docker-compose -f docker-compose.yml ps jupyterhub backend postgres redis
    echo ""
    
    # 检查JupyterHub健康状态
    if docker-compose -f docker-compose.yml exec jupyterhub curl -s -f http://localhost:8000/hub/health > /dev/null 2>&1; then
        log_success "JupyterHub: 健康"
    else
        log_error "JupyterHub: 不健康"
    fi
    
    # 检查后端健康状态
    if docker-compose -f docker-compose.yml exec backend curl -s -f http://localhost:8082/api/health > /dev/null 2>&1; then
        log_success "Backend: 健康"
    else
        log_error "Backend: 不健康"
    fi
}

# 主逻辑
main() {
    check_prerequisites
    
    case "${1:-help}" in
        "dev-restart"|"restart")
            dev_restart
            ;;
        "full-rebuild"|"rebuild")
            full_rebuild
            ;;
        "logs"|"log")
            show_logs
            ;;
        "test-auth"|"test")
            test_auth
            ;;
        "status"|"ps")
            show_status
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            log_error "未知命令: $1"
            echo ""
            show_help
            exit 1
            ;;
    esac
}

# 运行主函数
main "$@"
