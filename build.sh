#!/bin/bash

# AI-Infra-Matrix 简化构建和启动脚本
# 支持使用内部镜像仓库启动服务

set -euo pipefail

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

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 使用内部仓库启动服务
start_with_internal_registry() {
    local registry_prefix="$1"
    local tag="${2:-v0.3.5}"
    
    print_info "使用内部镜像仓库启动服务"
    print_info "仓库前缀: $registry_prefix"
    print_info "镜像标签: $tag"
    
    # 创建 override 文件
    local override_file="docker-compose.override.yml"
    
    print_info "生成 docker-compose override 配置..."
    
    cat > "$override_file" << EOF
# Auto-generated docker-compose override for internal registry
# 生成时间: $(date)
# 仓库前缀: $registry_prefix

version: '3.8'

services:
  backend-init:
    image: ${registry_prefix}ai-infra-backend-init:${tag}
    build: null
  
  backend:
    image: ${registry_prefix}ai-infra-backend:${tag}
    build: null
  
  frontend:
    image: ${registry_prefix}ai-infra-frontend:${tag}
    build: null
  
  jupyterhub:
    image: ${registry_prefix}ai-infra-jupyterhub:${tag}
    build: null
    environment:
      - JUPYTERHUB_IMAGE=${registry_prefix}ai-infra-singleuser:${tag}
  
  singleuser-builder:
    image: ${registry_prefix}ai-infra-singleuser:${tag}
    build: null
  
  saltstack:
    image: ${registry_prefix}ai-infra-saltstack:${tag}
    build: null
  
  nginx:
    image: ${registry_prefix}ai-infra-nginx:${tag}
    build: null
  
  gitea:
    image: ${registry_prefix}ai-infra-gitea:${tag}
    build: null
EOF
    
    print_success "override 文件已生成: $override_file"
    
    # 设置环境变量
    export IMAGE_REGISTRY_PREFIX="$registry_prefix"
    export IMAGE_TAG="$tag"
    export ENV_FILE=".env.prod"
    
    # 启动服务
    print_info "启动服务..."
    
    if command -v docker-compose &> /dev/null; then
        docker-compose --env-file .env.prod up -d
    elif docker compose version &> /dev/null; then
        docker compose --env-file .env.prod up -d
    else
        print_error "Docker Compose 未找到"
        exit 1
    fi
    
    print_success "服务已启动"
    
    # 显示服务状态
    print_info "检查服务状态..."
    sleep 5
    
    if command -v docker-compose &> /dev/null; then
        docker-compose ps
    else
        docker compose ps
    fi
}

# 停止服务
stop_services() {
    print_info "停止所有服务..."
    
    if command -v docker-compose &> /dev/null; then
        docker-compose down
    elif docker compose version &> /dev/null; then
        docker compose down
    else
        print_error "Docker Compose 未找到"
        exit 1
    fi
    
    print_success "服务已停止"
}

# 显示帮助信息
show_help() {
    cat << EOF
AI-Infra-Matrix 构建和启动脚本

用法: $0 <command> [options]

命令:
  start-internal <registry_prefix> [tag]  使用内部仓库启动服务
  stop                                   停止所有服务
  help                                   显示帮助信息
  
  其他命令会传递给 scripts/all-ops.sh 处理

示例:
  # 使用内部仓库启动服务
  $0 start-internal registry.company.com/ai-infra/ v0.3.5
  
  # 使用默认标签启动
  $0 start-internal registry.company.com/ai-infra/
  
  # 停止服务
  $0 stop
  
  # 构建镜像（传递给 all-ops.sh）
  $0 build
  
  # 生产环境构建
  $0 prod build

环境变量:
  IMAGE_REGISTRY_PREFIX  镜像仓库前缀 (例如: registry.company.com/ai-infra/)
  IMAGE_TAG              镜像标签 (默认: v0.3.5)
EOF
}

# 主逻辑
main() {
    local command="${1:-help}"
    
    case "$command" in
        "start-internal")
            if [[ $# -lt 2 ]]; then
                print_error "用法: $0 start-internal <registry_prefix> [tag]"
                print_info "示例: $0 start-internal registry.company.com/ai-infra/ v0.3.5"
                exit 1
            fi
            start_with_internal_registry "$2" "${3:-v0.3.5}"
            ;;
        "stop")
            stop_services
            ;;
        "help"|"--help"|"-h")
            show_help
            ;;
        *)
            # 传递给 all-ops.sh 处理其他命令
            ALL_OPS_SCRIPT="$SCRIPT_DIR/scripts/all-ops.sh"
            
            if [ ! -f "$ALL_OPS_SCRIPT" ]; then
                print_error "找不到 $ALL_OPS_SCRIPT"
                exit 1
            fi
            
            # 确保脚本可执行
            chmod +x "$ALL_OPS_SCRIPT"
            
            # 传递所有参数给 all-ops.sh
            exec "$ALL_OPS_SCRIPT" "$@"
            ;;
    esac
}

# 执行主函数
main "$@"
