#!/bin/bash
# AI Infrastructure Matrix - 统一部署脚本
# 版本: v2.0.0 - Nginx统一访问入口版本

set -e

# 脚本配置
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$SCRIPT_DIR"
COMPOSE_FILE="$PROJECT_ROOT/docker-compose.yml"

# 颜色输出
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
AI Infrastructure Matrix - 统一部署脚本

使用方法:
    $0 [命令] [选项]

命令:
    up              启动所有服务
    down            停止所有服务
    restart         重启所有服务
    status          查看服务状态
    logs            查看服务日志
    clean           清理系统（删除容器、镜像、卷）
    build           重新构建所有镜像
    update          更新并重新部署
    dev             启动开发环境
    prod            启动生产环境
    health          健康检查
    init            仅运行数据库初始化（需要服务已启动）

配置文件 Profile 选项:
    --with-jupyterhub    启动 JupyterHub 服务
    --with-k8s          启动 Kubernetes 代理
    --with-monitoring   启动监控服务
    --with-admin        启动管理界面
    --all               启动所有服务

服务选项:
    --service <name>    指定单个服务操作

其他选项:
    --force            强制执行操作
    --verbose          详细输出
    --help             显示此帮助信息

示例:
    $0 up --with-jupyterhub          # 启动包含JupyterHub的基础服务
    $0 up --all                      # 启动所有服务
    $0 restart --service nginx       # 重启nginx服务
    $0 logs --service backend        # 查看后端服务日志
    $0 clean --force                 # 强制清理所有资源

访问地址:
    主页:              http://localhost:8080
    后端API:           http://localhost:8080/api
    JupyterHub:        http://localhost:8080/jupyter
    API文档:           http://localhost:8080/swagger
    LDAP管理:          http://localhost:8080/ldap-admin (需要 --with-admin)
    Redis监控:         http://localhost:8080/redis-monitor (需要 --with-monitoring)

EOF
}

# 检查依赖
check_dependencies() {
    log_info "检查系统依赖..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装或未在PATH中"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose 未安装或未在PATH中"
        exit 1
    fi
    
    # 检查Docker守护进程
    if ! docker info &> /dev/null; then
        log_error "Docker 守护进程未运行"
        exit 1
    fi
    
    log_success "依赖检查完成"
}

# 设置环境变量
setup_environment() {
    log_info "设置环境变量..."
    
    # 创建 .env 文件（如果不存在）
    if [[ ! -f "$PROJECT_ROOT/.env" ]]; then
        cat > "$PROJECT_ROOT/.env" << EOF
# AI Infrastructure Matrix 环境配置
COMPOSE_PROJECT_NAME=ai-infra-matrix
LOG_LEVEL=info

# JWT配置
JWT_SECRET=ai-infra-secret-key-change-in-production

# 数据库配置
POSTGRES_DB=ansible_playbook_generator
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres

# Redis配置
REDIS_PASSWORD=ansible-redis-password

# LDAP配置
LDAP_ADMIN_PASSWORD=admin123
LDAP_CONFIG_PASSWORD=config123

# JupyterHub配置
JUPYTERHUB_ADMIN_USERS=admin,jupyter-admin
CONFIGPROXY_AUTH_TOKEN=ai-infra-proxy-token
EOF
        log_success "创建了默认 .env 文件"
    fi
    
    # 加载环境变量
    set -a
    source "$PROJECT_ROOT/.env"
    set +a
    
    log_success "环境变量设置完成"
}

# 构建镜像
build_images() {
    log_info "构建Docker镜像..."
    
    local services=("backend" "frontend" "jupyterhub")
    local profiles=""
    
    # 解析profiles
    for arg in "$@"; do
        case $arg in
            --with-jupyterhub) profiles="$profiles --profile jupyterhub" ;;
            --with-k8s) profiles="$profiles --profile k8s" ;;
            --with-monitoring) profiles="$profiles --profile monitoring" ;;
            --with-admin) profiles="$profiles --profile admin" ;;
            --all) profiles="$profiles --profile jupyterhub --profile k8s --profile monitoring --profile admin" ;;
        esac
    done
    
    # 构建镜像
    docker-compose -f "$COMPOSE_FILE" $profiles build --no-cache
    
    log_success "镜像构建完成"
}

# 启动服务
start_services() {
    log_info "启动AI Infrastructure Matrix服务..."
    
    local profiles=""
    local service=""
    
    # 解析参数
    for arg in "$@"; do
        case $arg in
            --with-jupyterhub) profiles="$profiles --profile jupyterhub" ;;
            --with-k8s) profiles="$profiles --profile k8s" ;;
            --with-monitoring) profiles="$profiles --profile monitoring" ;;
            --with-admin) profiles="$profiles --profile admin" ;;
            --all) profiles="$profiles --profile jupyterhub --profile k8s --profile monitoring --profile admin" ;;
            --service) shift; service="$1" ;;
        esac
    done
    
    # 如果未指定profile，默认启动基础服务
    if [[ -z "$profiles" && -z "$service" ]]; then
        profiles="--profile jupyterhub"
    fi
    
    # 启动服务
    if [[ -n "$service" ]]; then
        docker-compose -f "$COMPOSE_FILE" up -d "$service"
        log_success "服务 $service 启动完成"
    else
        docker-compose -f "$COMPOSE_FILE" $profiles up -d
        log_success "所有服务启动完成"
    fi
    
    # 等待服务就绪
    wait_for_services
    
    # 初始化数据库
    initialize_database
    
    # 显示访问信息
    show_access_info
}

# 等待服务就绪
wait_for_services() {
    log_info "等待服务就绪..."
    
    local max_attempts=30
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        if curl -s http://localhost:8080/health > /dev/null 2>&1; then
            log_success "Nginx服务已就绪"
            break
        fi
        
        log_info "等待Nginx启动... ($attempt/$max_attempts)"
        sleep 5
        ((attempt++))
    done
    
    if [[ $attempt -gt $max_attempts ]]; then
        log_warning "Nginx服务启动超时，请检查日志"
    fi
}

# 初始化数据库
initialize_database() {
    log_info "初始化数据库..."
    
    # 等待PostgreSQL就绪
    local max_attempts=30
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        if docker-compose exec -T postgres pg_isready -U postgres > /dev/null 2>&1; then
            log_success "PostgreSQL服务已就绪"
            break
        fi
        
        log_info "等待PostgreSQL启动... ($attempt/$max_attempts)"
        sleep 3
        ((attempt++))
    done
    
    if [[ $attempt -gt $max_attempts ]]; then
        log_error "PostgreSQL服务启动超时"
        return 1
    fi
    
    # 等待后端服务就绪
    log_info "等待后端服务就绪..."
    attempt=1
    while [[ $attempt -le $max_attempts ]]; do
        if docker-compose exec backend echo "Backend service is ready" > /dev/null 2>&1; then
            log_success "后端服务已就绪"
            break
        fi
        
        log_info "等待后端服务启动... ($attempt/$max_attempts)"
        sleep 3
        ((attempt++))
    done
    
    if [[ $attempt -gt $max_attempts ]]; then
        log_error "后端服务启动超时"
        return 1
    fi
    
    # 调用后端 Go 程序的初始化命令
    log_info "运行后端初始化程序..."
    if docker-compose exec backend ./init; then
        log_success "数据库初始化完成"
    else
        log_error "数据库初始化失败"
        return 1
    fi
}

# 停止服务
stop_services() {
    log_info "停止AI Infrastructure Matrix服务..."
    
    local service=""
    
    # 解析参数
    for arg in "$@"; do
        case $arg in
            --service) shift; service="$1" ;;
        esac
    done
    
    if [[ -n "$service" ]]; then
        docker-compose -f "$COMPOSE_FILE" stop "$service"
        log_success "服务 $service 停止完成"
    else
        docker-compose -f "$COMPOSE_FILE" down
        log_success "所有服务停止完成"
    fi
}

# 重启服务
restart_services() {
    log_info "重启AI Infrastructure Matrix服务..."
    stop_services "$@"
    sleep 3
    start_services "$@"
}

# 查看服务状态
show_status() {
    log_info "AI Infrastructure Matrix服务状态:"
    docker-compose -f "$COMPOSE_FILE" ps
    
    echo
    log_info "Docker容器状态:"
    docker ps --filter "name=ai-infra-" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
}

# 查看日志
show_logs() {
    local service=""
    local follow=false
    
    # 解析参数
    for arg in "$@"; do
        case $arg in
            --service) shift; service="$1" ;;
            --follow|-f) follow=true ;;
        esac
    done
    
    if [[ -n "$service" ]]; then
        if [[ "$follow" == true ]]; then
            docker-compose -f "$COMPOSE_FILE" logs -f "$service"
        else
            docker-compose -f "$COMPOSE_FILE" logs --tail=100 "$service"
        fi
    else
        if [[ "$follow" == true ]]; then
            docker-compose -f "$COMPOSE_FILE" logs -f
        else
            docker-compose -f "$COMPOSE_FILE" logs --tail=50
        fi
    fi
}

# 健康检查
health_check() {
    log_info "执行健康检查..."
    
    local services=(
        "nginx:http://localhost:8080/health"
        "backend:http://localhost:8080/api/health"
        "frontend:http://localhost:8080"
        "postgres:localhost:5433"
        "redis:localhost:6379"
    )
    
    for service_check in "${services[@]}"; do
        local service="${service_check%%:*}"
        local endpoint="${service_check#*:}"
        
        if [[ "$endpoint" == http* ]]; then
            if curl -s -o /dev/null -w "%{http_code}" "$endpoint" | grep -q "200\|403"; then
                log_success "$service 服务健康"
            else
                log_error "$service 服务异常"
            fi
        else
            # 对于非HTTP服务，检查端口
            local host="${endpoint%%:*}"
            local port="${endpoint#*:}"
            if nc -z "$host" "$port" 2>/dev/null; then
                log_success "$service 服务健康"
            else
                log_error "$service 服务异常"
            fi
        fi
    done
}

# 清理系统
clean_system() {
    local force=false
    
    for arg in "$@"; do
        case $arg in
            --force) force=true ;;
        esac
    done
    
    if [[ "$force" != true ]]; then
        read -p "确定要清理所有AI Infrastructure Matrix资源吗？(y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "取消清理操作"
            return
        fi
    fi
    
    log_warning "清理AI Infrastructure Matrix资源..."
    
    # 停止并删除容器
    docker-compose -f "$COMPOSE_FILE" down -v --remove-orphans
    
    # 删除镜像
    docker images --filter "reference=ai-infra-*" -q | xargs -r docker rmi -f
    
    # 删除网络
    docker network ls --filter "name=ai-infra-network" -q | xargs -r docker network rm
    
    # 删除数据卷
    docker volume ls --filter "name=ai-infra-*" -q | xargs -r docker volume rm
    
    log_success "清理完成"
}

# 显示访问信息
show_access_info() {
    cat << EOF

${GREEN}=== AI Infrastructure Matrix 部署完成 ===${NC}

🌐 Web访问地址:
   主页:              http://localhost:8080
   后端API:           http://localhost:8080/api
   JupyterHub:        http://localhost:8080/jupyter
   API文档:           http://localhost:8080/swagger

🔧 管理界面 (通过Nginx统一访问):
   LDAP管理:          http://localhost:8080/ldap-admin
   Redis监控:         http://localhost:8080/redis-monitor

👥 默认用户:
   JupyterHub管理员:  admin / admin
   数据库用户:        postgres / postgres

📁 重要路径:
   配置文件:          $PROJECT_ROOT/docker-compose.yml
   环境变量:          $PROJECT_ROOT/.env
   日志查看:          $0 logs

🚀 快速操作:
   查看状态:          $0 status
   查看日志:          $0 logs
   重启服务:          $0 restart
   健康检查:          $0 health

EOF
}

# 主函数
main() {
    cd "$PROJECT_ROOT"
    
    # 解析命令
    case "${1:-help}" in
        up|start)
            shift
            check_dependencies
            setup_environment
            start_services "$@"
            ;;
        down|stop)
            shift
            stop_services "$@"
            ;;
        restart)
            shift
            check_dependencies
            setup_environment
            restart_services "$@"
            ;;
        status)
            show_status
            ;;
        logs)
            shift
            show_logs "$@"
            ;;
        build)
            shift
            check_dependencies
            setup_environment
            build_images "$@"
            ;;
        clean)
            shift
            clean_system "$@"
            ;;
        health)
            health_check
            ;;
        init)
            shift
            check_dependencies
            setup_environment
            initialize_database
            ;;
        update)
            shift
            check_dependencies
            setup_environment
            stop_services
            build_images "$@"
            start_services "$@"
            ;;
        dev)
            shift
            check_dependencies
            setup_environment
            start_services --with-admin --with-monitoring "$@"
            ;;
        prod)
            shift
            check_dependencies
            setup_environment
            start_services --with-jupyterhub "$@"
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            log_error "未知命令: $1"
            echo
            show_help
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"
