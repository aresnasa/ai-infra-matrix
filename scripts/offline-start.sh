#!/bin/bash

# AI Infrastructure Matrix - 离线环境一键启动脚本
# 版本: v1.0.0
# 功能: 在完全离线环境中启动AI Infrastructure Matrix

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 打印函数
print_info() { echo -e "${BLUE}ℹ️  $1${NC}"; }
print_success() { echo -e "${GREEN}✅ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠️  $1${NC}"; }
print_error() { echo -e "${RED}❌ $1${NC}"; }

# 脚本配置
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$SCRIPT_DIR"
VERSION="${IMAGE_TAG:-v0.3.6-dev}"
OFFLINE_MODE=true

# 环境配置
ENV_FILE=".env.prod"
COMPOSE_FILE="docker-compose.yml"
PROJECT_NAME="ai-infra-matrix-offline"

# Docker Compose命令选择
DOCKER_COMPOSE=""
if command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE="docker-compose"
elif docker compose version &> /dev/null 2>&1; then
    DOCKER_COMPOSE="docker compose"
else
    print_error "未找到 docker-compose 或 docker compose 命令"
    exit 1
fi

# 检查系统依赖
check_dependencies() {
    print_info "检查系统依赖..."
    
    # 检查Docker
    if ! command -v docker &> /dev/null; then
        print_error "Docker 未安装，请先安装Docker"
        exit 1
    fi
    
    # 检查Docker服务状态
    if ! docker info >/dev/null 2>&1; then
        print_error "Docker服务未运行，请启动Docker"
        exit 1
    fi
    
    # 检查Docker Compose
    print_success "Docker和Docker Compose可用: $DOCKER_COMPOSE"
    
    # 检查必要的端口是否被占用
    check_ports
}

# 检查端口占用
check_ports() {
    local ports=(8080 5432 6379 9092 389 3000 8000)
    local occupied_ports=()
    
    for port in "${ports[@]}"; do
        if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
            occupied_ports+=($port)
        fi
    done
    
    if [ ${#occupied_ports[@]} -gt 0 ]; then
        print_warning "以下端口被占用: ${occupied_ports[*]}"
        print_info "这可能会导致服务启动失败"
        read -p "是否继续? (y/N): " -r
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_info "启动已取消"
            exit 0
        fi
    fi
}

# 检查镜像文件
check_image_files() {
    print_info "检查离线镜像文件..."
    
    local image_dir="$PROJECT_ROOT/offline-images"
    if [ ! -d "$image_dir" ]; then
        print_error "未找到离线镜像目录: $image_dir"
        print_info "请先运行 ./scripts/export-offline-images.sh 导出镜像"
        exit 1
    fi
    
    # 检查镜像文件
    local image_files=($(find "$image_dir" -name "*.tar" -o -name "*.tar.gz" 2>/dev/null))
    if [ ${#image_files[@]} -eq 0 ]; then
        print_error "未找到镜像文件"
        print_info "请确保已正确导出镜像到 $image_dir"
        exit 1
    fi
    
    print_success "找到 ${#image_files[@]} 个镜像文件"
    
    # 检查导入脚本
    local import_script="$image_dir/import-images.sh"
    if [ ! -x "$import_script" ]; then
        print_error "未找到镜像导入脚本或脚本不可执行: $import_script"
        exit 1
    fi
    
    print_success "镜像文件检查完成"
}

# 导入Docker镜像
import_images() {
    print_info "开始导入Docker镜像..."
    
    local image_dir="$PROJECT_ROOT/offline-images"
    local import_script="$image_dir/import-images.sh"
    
    # 检查是否已经导入镜像
    local required_images=(
        "postgres:15-alpine"
        "redis:7-alpine"
        "ai-infra-nginx:$VERSION"
        "ai-infra-backend:$VERSION"
        "ai-infra-frontend:$VERSION"
    )
    
    local missing_images=()
    for image in "${required_images[@]}"; do
        if ! docker image inspect "$image" >/dev/null 2>&1; then
            missing_images+=("$image")
        fi
    done
    
    if [ ${#missing_images[@]} -eq 0 ]; then
        print_success "所有必需镜像已存在，跳过导入"
        return 0
    fi
    
    print_info "需要导入 ${#missing_images[@]} 个镜像"
    
    # 执行导入
    cd "$image_dir"
    if ./import-images.sh; then
        print_success "镜像导入完成"
    else
        print_error "镜像导入失败"
        exit 1
    fi
    cd "$PROJECT_ROOT"
}

# 准备环境配置
prepare_environment() {
    print_info "准备环境配置..."
    
    # 检查环境文件
    if [ ! -f "$ENV_FILE" ]; then
        print_warning "环境文件不存在: $ENV_FILE"
        
        # 复制示例文件
        if [ -f "${ENV_FILE}.example" ]; then
            cp "${ENV_FILE}.example" "$ENV_FILE"
            print_info "已从示例文件创建: $ENV_FILE"
        else
            # 创建基础环境文件
            create_basic_env_file
        fi
    fi
    
    # 设置离线模式相关配置
    setup_offline_config
    
    print_success "环境配置准备完成"
}

# 创建基础环境文件
create_basic_env_file() {
    print_info "创建基础环境配置文件..."
    
    cat > "$ENV_FILE" << EOF
# AI Infrastructure Matrix - 离线环境配置
# 自动生成于: $(date)

# 基础配置
COMPOSE_PROJECT_NAME=$PROJECT_NAME
IMAGE_TAG=$VERSION
BUILD_ENV=production
DEBUG_MODE=false
TZ=Asia/Shanghai

# 外部访问配置
EXTERNAL_HOST=localhost
EXTERNAL_PORT=8080
EXTERNAL_SCHEME=http

# 数据库配置
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_DB=ai_infra
POSTGRES_USER=ai_infra_user
POSTGRES_PASSWORD=ai_infra_password_2024

# Redis配置
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=redis_password_2024

# LDAP配置
LDAP_HOST=openldap
LDAP_PORT=389
LDAP_ADMIN_PASSWORD=ldap_admin_2024
LDAP_CONFIG_PASSWORD=ldap_config_2024

# JWT配置
JWT_SECRET_KEY=jwt_secret_key_for_offline_environment_2024
JWT_ALGORITHM=HS256
JWT_ACCESS_TOKEN_EXPIRE_MINUTES=1440

# 应用配置
BACKEND_HOST=ai-infra-backend
BACKEND_PORT=8082
FRONTEND_PORT=80

# 文件存储
UPLOAD_PATH=/app/uploads
MAX_UPLOAD_SIZE=100MB

# 离线模式标识
OFFLINE_MODE=true
DISABLE_EXTERNAL_APIS=true
EOF
    
    print_success "基础环境配置文件创建完成"
}

# 设置离线配置
setup_offline_config() {
    # 确保离线模式配置
    if ! grep -q "OFFLINE_MODE=true" "$ENV_FILE"; then
        echo "OFFLINE_MODE=true" >> "$ENV_FILE"
    fi
    
    # 禁用外部API调用
    if ! grep -q "DISABLE_EXTERNAL_APIS=true" "$ENV_FILE"; then
        echo "DISABLE_EXTERNAL_APIS=true" >> "$ENV_FILE"
    fi
}

# 创建必要的目录
create_directories() {
    print_info "创建必要的数据目录..."
    
    local dirs=(
        "data/postgres"
        "data/redis"
        "data/kafka"
        "data/ldap"
        "data/gitea"
        "data/jupyter"
        "data/minio"
        "logs"
        "uploads"
    )
    
    for dir in "${dirs[@]}"; do
        mkdir -p "$dir"
        # 设置适当的权限
        chmod 755 "$dir"
    done
    
    print_success "数据目录创建完成"
}

# 检查Docker Compose文件
check_compose_file() {
    print_info "检查Docker Compose配置..."
    
    if [ ! -f "$COMPOSE_FILE" ]; then
        print_error "Docker Compose文件不存在: $COMPOSE_FILE"
        exit 1
    fi
    
    # 验证配置文件语法
    if ! $DOCKER_COMPOSE -f "$COMPOSE_FILE" config >/dev/null 2>&1; then
        print_error "Docker Compose配置文件有语法错误"
        $DOCKER_COMPOSE -f "$COMPOSE_FILE" config
        exit 1
    fi
    
    print_success "Docker Compose配置检查通过"
}

# 启动服务
start_services() {
    print_info "启动AI Infrastructure Matrix服务..."
    
    # 设置环境变量
    export COMPOSE_PROJECT_NAME="$PROJECT_NAME"
    export IMAGE_TAG="$VERSION"
    
    # 分阶段启动服务
    start_infrastructure_services
    start_application_services
    start_gateway_services
    
    print_success "所有服务启动完成"
}

# 启动基础设施服务
start_infrastructure_services() {
    print_info "启动基础设施服务 (数据库、缓存)..."
    
    local infra_services="postgres redis openldap"
    
    $DOCKER_COMPOSE up -d $infra_services
    
    # 等待服务就绪
    print_info "等待基础设施服务就绪..."
    sleep 30
    
    # 检查服务状态
    for service in $infra_services; do
        if check_service_health "$service"; then
            print_success "$service 服务启动成功"
        else
            print_warning "$service 服务可能未完全就绪"
        fi
    done
}

# 启动应用服务
start_application_services() {
    print_info "启动应用服务 (后端、前端、JupyterHub)..."
    
    # 先启动后端初始化
    $DOCKER_COMPOSE up -d backend-init
    print_info "等待数据库初始化完成..."
    
    # 等待初始化完成
    local max_wait=120
    local wait_time=0
    while [ $wait_time -lt $max_wait ]; do
        if ! $DOCKER_COMPOSE ps backend-init | grep -q "running"; then
            break
        fi
        sleep 5
        wait_time=$((wait_time + 5))
        print_info "数据库初始化中... ($wait_time/${max_wait}s)"
    done
    
    # 启动应用服务
    local app_services="backend frontend jupyterhub singleuser gitea"
    $DOCKER_COMPOSE up -d $app_services
    
    print_info "等待应用服务启动..."
    sleep 20
}

# 启动网关服务
start_gateway_services() {
    print_info "启动网关服务 (Nginx)..."
    
    $DOCKER_COMPOSE up -d nginx
    
    print_info "等待网关服务启动..."
    sleep 10
    
    if check_service_health "nginx"; then
        print_success "网关服务启动成功"
    else
        print_warning "网关服务可能未完全就绪"
    fi
}

# 检查服务健康状态
check_service_health() {
    local service="$1"
    local container_name="${PROJECT_NAME}_${service}_1"
    
    # 尝试多种容器名称格式
    local possible_names=(
        "${PROJECT_NAME}_${service}_1"
        "${PROJECT_NAME}-${service}-1"
        "ai-infra-${service}"
    )
    
    for name in "${possible_names[@]}"; do
        if docker ps --format "table {{.Names}}" | grep -q "^$name$"; then
            local status=$(docker inspect --format='{{.State.Health.Status}}' "$name" 2>/dev/null || echo "none")
            if [ "$status" = "healthy" ] || [ "$status" = "none" ]; then
                return 0
            fi
        fi
    done
    
    return 1
}

# 显示服务状态
show_service_status() {
    print_info "服务状态概览:"
    echo "=================================="
    
    $DOCKER_COMPOSE ps
    
    echo ""
    print_info "服务访问地址:"
    echo "🌐 主页面: http://localhost:8080"
    echo "🔐 SSO登录: http://localhost:8080/sso/"
    echo "📊 JupyterHub: http://localhost:8080/jupyter"
    echo "🔧 Gitea: http://localhost:8080/gitea/"
    echo "📊 Kafka UI: http://localhost:9095"
    echo "👥 LDAP Admin: http://localhost:8080/phpldapadmin/"
    echo "🗄️  Redis Insight: http://localhost:8001"
}

# 健康检查
health_check() {
    print_info "执行健康检查..."
    
    local services_to_check=(
        "http://localhost:8080" "主页面"
        "http://localhost:8080/api/health" "后端API"
    )
    
    local healthy_count=0
    local total_checks=$((${#services_to_check[@]} / 2))
    
    for ((i=0; i<${#services_to_check[@]}; i+=2)); do
        local url="${services_to_check[i]}"
        local name="${services_to_check[i+1]}"
        
        print_info "检查 $name..."
        if curl -s -f "$url" >/dev/null 2>&1; then
            print_success "$name 正常"
            healthy_count=$((healthy_count + 1))
        else
            print_warning "$name 不可访问"
        fi
    done
    
    echo ""
    if [ $healthy_count -eq $total_checks ]; then
        print_success "健康检查全部通过 ($healthy_count/$total_checks)"
    else
        print_warning "健康检查部分通过 ($healthy_count/$total_checks)"
    fi
}

# 显示使用说明
show_usage() {
    cat << EOF
AI Infrastructure Matrix - 离线环境启动脚本

用法: $0 [选项]

选项:
  start       启动所有服务 (默认)
  stop        停止所有服务
  restart     重启所有服务
  status      显示服务状态
  health      执行健康检查
  logs        显示服务日志
  clean       清理环境 (删除容器和数据卷)
  --help|-h   显示此帮助信息

示例:
  $0          # 启动所有服务
  $0 start    # 启动所有服务
  $0 status   # 查看服务状态
  $0 health   # 执行健康检查
  $0 logs nginx # 查看nginx日志

离线环境使用说明:
1. 确保已运行 ./scripts/export-offline-images.sh 导出镜像
2. 确保离线镜像文件位于 ./offline-images/ 目录
3. 运行本脚本启动服务
4. 访问 http://localhost:8080 使用系统

注意事项:
- 离线模式下无法访问外部API
- 某些功能可能受限
- 确保Docker服务正常运行
EOF
}

# 停止服务
stop_services() {
    print_info "停止AI Infrastructure Matrix服务..."
    $DOCKER_COMPOSE down
    print_success "服务已停止"
}

# 重启服务
restart_services() {
    print_info "重启AI Infrastructure Matrix服务..."
    stop_services
    sleep 5
    start_all_services
}

# 显示日志
show_logs() {
    local service="$1"
    if [ -n "$service" ]; then
        $DOCKER_COMPOSE logs -f "$service"
    else
        $DOCKER_COMPOSE logs -f
    fi
}

# 清理环境
clean_environment() {
    print_warning "这将删除所有容器、网络和数据卷"
    read -p "确认继续? (y/N): " -r
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_info "清理环境..."
        $DOCKER_COMPOSE down -v --remove-orphans
        docker system prune -f
        print_success "环境清理完成"
    else
        print_info "取消清理操作"
    fi
}

# 启动所有服务的完整流程
start_all_services() {
    check_dependencies
    check_image_files
    import_images
    prepare_environment
    create_directories
    check_compose_file
    start_services
    echo ""
    show_service_status
    echo ""
    print_success "🎉 AI Infrastructure Matrix 离线环境启动完成!"
    print_info "💡 运行 '$0 health' 进行健康检查"
    print_info "💡 运行 '$0 status' 查看服务状态"
}

# 主函数
main() {
    local command="${1:-start}"
    
    case "$command" in
        start)
            start_all_services
            ;;
        stop)
            stop_services
            ;;
        restart)
            restart_services
            ;;
        status)
            show_service_status
            ;;
        health)
            health_check
            ;;
        logs)
            show_logs "$2"
            ;;
        clean)
            clean_environment
            ;;
        --help|-h)
            show_usage
            ;;
        *)
            print_error "未知命令: $command"
            show_usage
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"