#!/bin/bash

# AI-Infra-Matrix 构建脚本
# 支持开发模式和生产模式

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# 显示帮助信息
show_help() {
    echo "AI-Infra-Matrix 构建脚本"
    echo ""
    echo "用法: $0 [模式] [选项]"
    echo ""
    echo "模式:"
    echo "  dev, development     - 开发模式 (启用调试工具)"
    echo "  prod, production     - 生产模式 (禁用调试工具)"
    echo ""
    echo "选项:"
    echo "  --no-cache          - 无缓存构建"
    echo "  --rebuild           - 强制重建所有服务"
    echo "  --nginx-only        - 只构建nginx服务"
    echo "  -h, --help          - 显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0 dev              - 开发模式构建"
    echo "  $0 prod --no-cache  - 生产模式无缓存构建"
    echo "  $0 dev --nginx-only - 开发模式只构建nginx"
}

# 默认参数
MODE="production"
NO_CACHE=""
REBUILD=""
NGINX_ONLY=""

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    case $1 in
        dev|development)
            MODE="development"
            shift
            ;;
        prod|production)
            MODE="production"
            shift
            ;;
        --no-cache)
            NO_CACHE="--no-cache"
            shift
            ;;
        --rebuild)
            REBUILD="--force-recreate"
            shift
            ;;
        --nginx-only)
            NGINX_ONLY="nginx"
            shift
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            print_error "未知参数: $1"
            show_help
            exit 1
            ;;
    esac
done

# 显示构建信息
echo "🚀 AI-Infra-Matrix 构建开始"
echo "================================"
print_info "构建模式: $MODE"
print_info "构建时间: $(date)"

# 设置环境变量文件
if [ "$MODE" = "development" ]; then
    ENV_FILE=".env.development"
    export DEBUG_MODE=true
    export BUILD_ENV=development
    print_info "使用开发环境配置: $ENV_FILE"
    print_warning "调试工具将被启用"
else
    ENV_FILE=".env.production"
    export DEBUG_MODE=false
    export BUILD_ENV=production
    print_info "使用生产环境配置: $ENV_FILE"
    print_warning "调试工具将被禁用"
fi

# 检查环境文件是否存在
if [ ! -f "$ENV_FILE" ]; then
    print_warning "环境文件 $ENV_FILE 不存在，使用默认配置"
else
    print_success "环境文件 $ENV_FILE 已找到"
fi

# 检查Docker是否可用
if ! command -v docker &> /dev/null; then
    print_error "Docker 未安装或不可用"
    exit 1
fi

if ! command -v docker-compose &> /dev/null; then
    print_error "Docker Compose 未安装或不可用"
    exit 1
fi

# 构建服务
print_info "开始构建服务..."

if [ -n "$NGINX_ONLY" ]; then
    print_info "仅构建 nginx 服务"
    SERVICES="nginx"
else
    SERVICES=""
fi

# 构建命令
BUILD_CMD="docker-compose"

# 添加环境文件参数
if [ -f "$ENV_FILE" ]; then
    BUILD_CMD="$BUILD_CMD --env-file $ENV_FILE"
fi

BUILD_CMD="$BUILD_CMD build $NO_CACHE $SERVICES"

print_info "执行构建命令: $BUILD_CMD"
eval $BUILD_CMD

if [ $? -eq 0 ]; then
    print_success "构建完成!"
else
    print_error "构建失败!"
    exit 1
fi

# 启动服务（如果需要）
if [ -n "$REBUILD" ] || [ -n "$NGINX_ONLY" ]; then
    print_info "重启服务..."
    
    START_CMD="docker-compose"
    if [ -f "$ENV_FILE" ]; then
        START_CMD="$START_CMD --env-file $ENV_FILE"
    fi
    
    if [ -n "$NGINX_ONLY" ]; then
        START_CMD="$START_CMD up -d $REBUILD nginx"
    else
        START_CMD="$START_CMD up -d $REBUILD"
    fi
    
    print_info "执行启动命令: $START_CMD"
    eval $START_CMD
    
    if [ $? -eq 0 ]; then
        print_success "服务启动完成!"
    else
        print_error "服务启动失败!"
        exit 1
    fi
fi

# 显示完成信息
echo ""
echo "🎉 构建完成!"
echo "================================"
print_info "构建模式: $MODE"
print_info "服务访问:"
echo "  🌐 前端应用: http://localhost:8080"
echo "  🔐 SSO登录: http://localhost:8080/sso/"
echo "  📊 JupyterHub: http://localhost:8080/jupyterhub"

if [ "$MODE" = "development" ]; then
    echo "  🔧 调试工具: http://localhost:8080/debug/"
    print_warning "调试模式已启用，生产环境请使用 prod 模式构建"
fi

print_info "查看服务状态: docker-compose ps"
print_info "查看日志: docker-compose logs -f [服务名]"
