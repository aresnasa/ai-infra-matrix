#!/bin/bash

# AI-Infra-Matrix 智能启动脚本
# 自动检测环境并启动相应模式

set -e

# 颜色定义
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

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

echo "🚀 AI-Infra-Matrix 智能启动"
echo "================================"

# 检查命令行参数
MODE="auto"
if [ "$1" = "dev" ] || [ "$1" = "development" ]; then
    MODE="development"
elif [ "$1" = "prod" ] || [ "$1" = "production" ]; then
    MODE="production"
fi

# 自动检测环境
if [ "$MODE" = "auto" ]; then
    if [ -f ".env.development" ] && [ "$(hostname)" = "localhost" ] || [ "$(whoami)" != "root" ]; then
        MODE="development"
        print_info "自动检测到开发环境"
    else
        MODE="production"
        print_info "自动检测到生产环境"
    fi
fi

# 检查必要文件
if [ ! -f "scripts/build.sh" ]; then
    print_error "构建脚本不存在: scripts/build.sh"
    exit 1
fi

# 启动相应模式
print_info "启动模式: $MODE"

if [ "$MODE" = "development" ]; then
    print_warning "启用调试模式"
    ./scripts/build.sh dev --rebuild
    print_success "开发环境启动完成!"
    echo ""
    print_info "🔧 调试工具: http://localhost:8080/debug/"
else
    print_info "启用生产模式"
    ./scripts/build.sh prod --rebuild
    print_success "生产环境启动完成!"
fi

echo ""
print_info "🌐 主要访问地址:"
echo "  前端应用: http://localhost:8080"
echo "  SSO登录: http://localhost:8080/sso/"
echo "  JupyterHub: http://localhost:8080/jupyterhub"

echo ""
print_info "📋 管理命令:"
echo "  查看服务状态: docker-compose ps"
echo "  查看日志: docker-compose logs -f [服务名]"
echo "  停止服务: docker-compose down"
