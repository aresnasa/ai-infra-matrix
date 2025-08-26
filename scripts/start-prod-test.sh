#!/bin/bash

# AI-Infra Matrix 生产环境测试快速启动脚本
# 使用 aiharbor.msxf.local/aihpc 镜像仓库进行生产部署测试

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
COMPOSE_FILE="docker-compose.prod-test.yml"
ENV_FILE=".env.prod-test"
PROJECT_NAME="ai-infra-matrix-prod-test"

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
AI-Infra Matrix 生产环境测试启动脚本

用法: $0 [命令] [选项]

命令:
    start           启动所有服务
    stop            停止所有服务
    restart         重启所有服务
    status          查看服务状态
    logs            查看服务日志
    clean           清理环境（删除容器和数据卷）
    rebuild         重新构建并启动
    health          检查服务健康状态
    urls            显示访问地址

选项:
    -h, --help      显示此帮助信息
    -f, --follow    跟随日志输出（用于logs命令）
    -s, --service   指定服务名称
    --retag         重新标记镜像

示例:
    $0 start                    # 启动所有服务
    $0 logs -f                  # 跟随查看所有日志
    $0 logs -s backend          # 查看后端服务日志
    $0 health                   # 检查服务健康状态
    $0 clean                    # 清理环境
    $0 rebuild --retag         # 重新标记镜像并重新构建

EOF
}

# 检查依赖
check_dependencies() {
    local deps=(docker docker-compose)
    local missing=()
    
    for dep in "${deps[@]}"; do
        if ! command -v "$dep" >/dev/null 2>&1; then
            missing+=("$dep")
        fi
    done
    
    if [[ ${#missing[@]} -gt 0 ]]; then
        log_error "缺少依赖: ${missing[*]}"
        log_info "请安装缺少的依赖后重试"
        exit 1
    fi
}

# 检查Docker是否运行
check_docker() {
    if ! docker info >/dev/null 2>&1; then
        log_error "Docker未运行或无法访问"
        exit 1
    fi
}

# 检查文件是否存在
check_files() {
    local files=("$COMPOSE_FILE" "$ENV_FILE")
    local missing=()
    
    for file in "${files[@]}"; do
        if [[ ! -f "$file" ]]; then
            missing+=("$file")
        fi
    done
    
    if [[ ${#missing[@]} -gt 0 ]]; then
        log_error "缺少文件: ${missing[*]}"
        exit 1
    fi
}

# 重新标记镜像
retag_images() {
    log_info "重新标记镜像..."
    if [[ -x "./scripts/retag-images-for-prod.sh" ]]; then
        ./scripts/retag-images-for-prod.sh --deps
    else
        log_warning "重新标记脚本不存在或不可执行"
    fi
}

# 启动服务
start_services() {
    log_info "启动生产测试环境..."
    
    # 创建必要的目录
    local dirs=(logs logs/nginx src/backend/outputs src/backend/uploads shared)
    for dir in "${dirs[@]}"; do
        if [[ ! -d "$dir" ]]; then
            mkdir -p "$dir"
            log_info "创建目录: $dir"
        fi
    done
    
    # 启动服务
    docker-compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" -p "$PROJECT_NAME" up -d
    
    if [[ $? -eq 0 ]]; then
        log_success "服务启动成功"
        show_urls
    else
        log_error "服务启动失败"
        exit 1
    fi
}

# 停止服务
stop_services() {
    log_info "停止生产测试环境..."
    docker-compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" -p "$PROJECT_NAME" down
    log_success "服务已停止"
}

# 重启服务
restart_services() {
    log_info "重启生产测试环境..."
    stop_services
    sleep 2
    start_services
}

# 查看服务状态
show_status() {
    log_info "服务状态:"
    docker-compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" -p "$PROJECT_NAME" ps
}

# 查看日志
show_logs() {
    local follow_flag=""
    local service=""
    
    # 解析参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -f|--follow)
                follow_flag="-f"
                shift
                ;;
            -s|--service)
                service="$2"
                shift 2
                ;;
            *)
                break
                ;;
        esac
    done
    
    if [[ -n "$service" ]]; then
        log_info "查看服务 [$service] 日志:"
        docker-compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" -p "$PROJECT_NAME" logs $follow_flag "$service"
    else
        log_info "查看所有服务日志:"
        docker-compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" -p "$PROJECT_NAME" logs $follow_flag
    fi
}

# 清理环境
clean_environment() {
    log_warning "这将删除所有容器、网络和数据卷！"
    read -p "确认继续? [y/N]: " -n 1 -r
    echo
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        log_info "清理环境..."
        docker-compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" -p "$PROJECT_NAME" down -v --remove-orphans
        
        # 删除相关镜像（可选）
        read -p "是否删除aiharbor.msxf.local相关镜像? [y/N]: " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            docker images | grep "aiharbor.msxf.local" | awk '{print $1":"$2}' | xargs docker rmi 2>/dev/null || true
        fi
        
        log_success "环境清理完成"
    else
        log_info "取消清理操作"
    fi
}

# 重新构建
rebuild_services() {
    local retag_flag=false
    
    # 检查是否需要重新标记
    while [[ $# -gt 0 ]]; do
        case $1 in
            --retag)
                retag_flag=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    if [[ "$retag_flag" == "true" ]]; then
        retag_images
    fi
    
    log_info "重新构建并启动..."
    stop_services
    docker-compose -f "$COMPOSE_FILE" --env-file "$ENV_FILE" -p "$PROJECT_NAME" build --no-cache
    start_services
}

# 检查服务健康状态
check_health() {
    log_info "检查服务健康状态..."
    
    local services=(
        "postgres:5432"
        "redis:6379"
        "backend:8082"
        "frontend:80"
        "nginx:80"
        "minio:9000"
        "gitea:3000"
        "jupyterhub:8000"
    )
    
    for service in "${services[@]}"; do
        local name=$(echo "$service" | cut -d':' -f1)
        local port=$(echo "$service" | cut -d':' -f2)
        local container="ai-infra-${name}-prod-test"
        
        if docker ps --format "table {{.Names}}" | grep -q "^${container}$"; then
            if docker exec "$container" nc -z localhost "$port" 2>/dev/null; then
                log_success "$name: 健康"
            else
                log_warning "$name: 端口 $port 不可访问"
            fi
        else
            log_error "$name: 容器未运行"
        fi
    done
}

# 显示访问地址
show_urls() {
    log_info "服务访问地址:"
    cat << EOF

🌐 主要服务:
   - 主界面 (Nginx):      http://localhost:8080
   - 后端API:            http://localhost:8082
   - 前端 (直接):         http://localhost:3000

📊 管理界面:
   - JupyterHub:         http://localhost:8088
   - Gitea:             http://localhost:3010
   - Gitea (调试):       http://localhost:3011
   - MinIO控制台:        http://localhost:9001
   - Redis Insight:     http://localhost:8001

🔧 默认登录凭据:
   - 管理员: admin / admin123prod
   - MinIO: minioadmin_prod / minioadmin_prod_2024_secure

📝 配置文件:
   - Docker Compose: $COMPOSE_FILE
   - 环境变量: $ENV_FILE

EOF
}

# 主函数
main() {
    # 检查依赖
    check_dependencies
    check_docker
    
    # 解析命令
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        start)
            check_files
            start_services
            ;;
        stop)
            stop_services
            ;;
        restart)
            check_files
            restart_services
            ;;
        status)
            show_status
            ;;
        logs)
            show_logs "$@"
            ;;
        clean)
            clean_environment
            ;;
        rebuild)
            check_files
            rebuild_services "$@"
            ;;
        health)
            check_health
            ;;
        urls)
            show_urls
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            log_error "未知命令: $command"
            show_help
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"
