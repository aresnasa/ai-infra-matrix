#!/bin/bash

# AI Infrastructure Matrix - Nginx 统一访问入口部署脚本
# 作者: AI Infrastructure Team
# 版本: 2.0.0

set -e

echo "🚀 开始部署AI基础设施矩阵 - Nginx统一访问入口版本..."

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_message() {
    echo -e "${2}${1}${NC}"
}

# 检查 Docker 和 Docker Compose
check_prerequisites() {
    print_message "📋 检查系统依赖..." $BLUE
    
    if ! command -v docker &> /dev/null; then
        print_message "❌ Docker 未安装，请先安装 Docker" $RED
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        print_message "❌ Docker Compose 未安装，请先安装 Docker Compose" $RED
        exit 1
    fi
    
    print_message "✅ 系统依赖检查完成" $GREEN
}

# 停止现有服务
stop_existing_services() {
    print_message "🛑 停止现有服务..." $YELLOW
    
    # 停止所有相关的 profiles
    docker-compose --profile jupyterhub-unified down || true
    docker-compose down || true
    
    print_message "✅ 现有服务已停止" $GREEN
}

# 清理和重建
rebuild_services() {
    print_message "🔨 重新构建服务..." $BLUE
    
    # 构建前端 (使用新的 Nginx 配置)
    print_message "🏗️  构建前端服务..." $BLUE
    docker-compose build frontend
    
    # 构建后端
    print_message "🏗️  构建后端服务..." $BLUE
    docker-compose build backend
    
    # 构建统一认证 JupyterHub
    print_message "🏗️  构建JupyterHub统一认证服务..." $BLUE
    docker-compose --profile jupyterhub-unified build jupyterhub-unified
    
    print_message "✅ 服务构建完成" $GREEN
}

# 启动基础服务
start_infrastructure() {
    print_message "🚀 启动基础设施服务..." $BLUE
    
    # 启动数据库和缓存
    docker-compose up -d postgres redis openldap
    
    # 等待服务就绪
    print_message "⏳ 等待数据库和缓存服务就绪..." $YELLOW
    sleep 15
    
    # 检查健康状态
    docker-compose ps
    
    print_message "✅ 基础设施服务启动完成" $GREEN
}

# 启动应用服务
start_application() {
    print_message "🚀 启动应用服务..." $BLUE
    
    # 启动后端
    docker-compose up -d backend
    
    # 等待后端就绪
    print_message "⏳ 等待后端服务就绪..." $YELLOW
    sleep 20
    
    # 启动前端
    docker-compose up -d frontend
    
    # 等待前端就绪
    print_message "⏳ 等待前端服务就绪..." $YELLOW
    sleep 10
    
    print_message "✅ 应用服务启动完成" $GREEN
}

# 启动 JupyterHub 统一认证
start_jupyterhub() {
    print_message "🚀 启动JupyterHub统一认证服务..." $BLUE
    
    # 启动 JupyterHub
    docker-compose --profile jupyterhub-unified up -d jupyterhub-unified
    
    # 等待服务就绪
    print_message "⏳ 等待JupyterHub服务就绪..." $YELLOW
    sleep 15
    
    print_message "✅ JupyterHub统一认证服务启动完成" $GREEN
}

# 启动 Nginx 反向代理
start_nginx() {
    print_message "🚀 启动Nginx反向代理..." $BLUE
    
    # 启动 Nginx
    docker-compose up -d nginx
    
    # 等待服务就绪
    print_message "⏳ 等待Nginx服务就绪..." $YELLOW
    sleep 5
    
    print_message "✅ Nginx反向代理启动完成" $GREEN
}

# 验证部署
verify_deployment() {
    print_message "🔍 验证部署状态..." $BLUE
    
    echo "=== 服务状态 ==="
    docker-compose ps
    
    echo ""
    echo "=== 健康检查 ==="
    
    # 检查 Nginx
    if curl -f -s http://localhost/health > /dev/null; then
        print_message "✅ Nginx 反向代理: 健康" $GREEN
    else
        print_message "❌ Nginx 反向代理: 异常" $RED
    fi
    
    # 检查后端 API
    if curl -f -s http://localhost/api/health > /dev/null; then
        print_message "✅ 后端 API: 健康" $GREEN
    else
        print_message "❌ 后端 API: 异常" $RED
    fi
    
    # 检查前端
    if curl -f -s http://localhost/ > /dev/null; then
        print_message "✅ 前端应用: 健康" $GREEN
    else
        print_message "❌ 前端应用: 异常" $RED
    fi
    
    # 检查 JupyterHub
    if curl -f -s http://localhost/jupyter/ > /dev/null; then
        print_message "✅ JupyterHub: 健康" $GREEN
    else
        print_message "❌ JupyterHub: 异常" $RED
    fi
}

# 显示访问信息
show_access_info() {
    print_message "🎉 部署完成！" $GREEN
    
    echo ""
    echo "=== 🌐 访问信息 ==="
    echo "┌─────────────────────────────────────────────────────────────┐"
    echo "│                     🚀 AI 基础设施矩阵                      │"
    echo "├─────────────────────────────────────────────────────────────┤"
    echo "│  🏠 主应用入口:     http://localhost                       │"
    echo "│  📊 后端API:        http://localhost/api                   │"
    echo "│  📔 JupyterHub:     http://localhost/jupyter               │"
    echo "│  📚 API文档:        http://localhost/swagger               │"
    echo "│  🩺 健康检查:       http://localhost/health                 │"
    echo "└─────────────────────────────────────────────────────────────┘"
    echo ""
    
    echo "=== 🔧 管理命令 ==="
    echo "查看日志:     docker-compose logs -f [service_name]"
    echo "停止服务:     docker-compose down"
    echo "重启服务:     docker-compose restart [service_name]"
    echo "查看状态:     docker-compose ps"
    echo ""
    
    print_message "💡 提示: 现在所有服务都通过 Nginx 统一入口访问，无需记住各种端口号！" $YELLOW
}

# 主函数
main() {
    print_message "🌟 AI基础设施矩阵 - Nginx统一访问入口部署" $BLUE
    print_message "=================================================" $BLUE
    
    check_prerequisites
    stop_existing_services
    rebuild_services
    start_infrastructure
    start_application
    start_jupyterhub
    start_nginx
    verify_deployment
    show_access_info
    
    print_message "🎊 部署成功完成！系统已就绪！" $GREEN
}

# 错误处理
trap 'print_message "❌ 部署过程中出现错误，请检查日志" $RED; exit 1' ERR

# 执行主函数
main "$@"
