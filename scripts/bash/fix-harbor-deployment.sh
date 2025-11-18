#!/bin/bash

# AI-Infra-Matrix Harbor部署修复脚本
# 解决依赖镜像缺失问题

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

# 默认参数
REGISTRY="aiharbor.msxf.local/aihpc"
TAG="v0.3.6-dev"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 显示使用说明
show_usage() {
    echo "AI-Infra-Matrix Harbor部署修复脚本"
    echo ""
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  --registry <registry>   指定Harbor仓库地址 (默认: $REGISTRY)"
    echo "  --tag <tag>            指定镜像标签 (默认: $TAG)"
    echo "  --check-only           只检查状态，不执行修复"
    echo "  --skip-deps            跳过依赖镜像推送"
    echo "  --help                 显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0                                    # 使用默认配置修复"
    echo "  $0 --registry my.harbor.com/ai --tag v1.0.0  # 自定义配置"
    echo "  $0 --check-only                      # 只检查状态"
}

# 检查Docker和Harbor连接
check_prerequisites() {
    print_info "检查先决条件..."
    
    # 检查Docker
    if ! command -v docker &> /dev/null; then
        print_error "Docker 未安装或不可用"
        return 1
    fi
    
    # 检查Docker Compose
    if ! command -v docker-compose &> /dev/null; then
        print_error "Docker Compose 未安装或不可用"
        return 1
    fi
    
    # 检查build.sh脚本
    if [[ ! -f "$SCRIPT_DIR/build.sh" ]]; then
        print_error "build.sh 脚本未找到"
        return 1
    fi
    
    # 检查Harbor登录状态
    print_info "检查Harbor登录状态..."
    local harbor_host=$(echo "$REGISTRY" | cut -d'/' -f1)
    
    if ! docker system info 2>/dev/null | grep -q "$harbor_host"; then
        print_warning "未检测到Harbor登录状态"
        print_info "请先登录Harbor:"
        echo "  docker login $harbor_host"
        
        # 提示用户登录
        read -p "是否现在登录Harbor? (y/N): " -r
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            if ! docker login "$harbor_host"; then
                print_error "Harbor登录失败"
                return 1
            fi
        else
            print_warning "跳过登录，可能会遇到推送权限问题"
        fi
    else
        print_success "Harbor登录状态正常"
    fi
    
    return 0
}

# 检查当前状态
check_current_status() {
    print_info "检查当前部署状态..."
    
    # 检查是否有运行的容器
    local running_containers
    running_containers=$(docker ps --format "table {{.Names}}\t{{.Status}}" | grep "ai-infra" || true)
    
    if [[ -n "$running_containers" ]]; then
        print_info "发现运行中的AI-Infra容器:"
        echo "$running_containers"
        echo ""
    else
        print_info "没有发现运行中的AI-Infra容器"
    fi
    
    # 检查配置文件
    if [[ -f "docker-compose.prod.yml" ]]; then
        print_info "发现生产配置文件: docker-compose.prod.yml"
        
        # 检查配置文件中的registry
        local config_registry
        config_registry=$(grep -o "$REGISTRY" docker-compose.prod.yml | head -1 || true)
        
        if [[ -n "$config_registry" ]]; then
            print_success "配置文件中的registry匹配: $config_registry"
        else
            print_warning "配置文件中的registry不匹配，需要重新生成"
        fi
    else
        print_warning "未找到生产配置文件，需要生成"
    fi
}

# 检查Harbor中的镜像
check_harbor_images() {
    print_info "检查Harbor中的镜像状态..."
    
    # 关键依赖镜像列表
    local deps_images=(
        "library/postgres:$TAG"
        "library/redis:$TAG"
        "library/nginx:$TAG"
        "minio/minio:$TAG"
        "tecnativa/tcp-proxy:$TAG"
        "redislabs/redisinsight:$TAG"
    )
    
    # 项目镜像列表
    local project_images=(
        "ai-infra-backend:$TAG"
        "ai-infra-frontend:$TAG"
        "ai-infra-jupyterhub:$TAG"
        "ai-infra-nginx:$TAG"
        "ai-infra-gitea:$TAG"
        "ai-infra-saltstack:$TAG"
    )
    
    local missing_deps=()
    local missing_project=()
    
    print_info "检查依赖镜像..."
    for image in "${deps_images[@]}"; do
        local full_image="$REGISTRY/$image"
        if docker manifest inspect "$full_image" >/dev/null 2>&1; then
            print_success "✓ $image"
        else
            print_error "✗ $image"
            missing_deps+=("$image")
        fi
    done
    
    print_info "检查项目镜像..."
    for image in "${project_images[@]}"; do
        local full_image="$REGISTRY/$image"
        if docker manifest inspect "$full_image" >/dev/null 2>&1; then
            print_success "✓ $image"
        else
            print_error "✗ $image"
            missing_project+=("$image")
        fi
    done
    
    echo ""
    print_info "镜像状态总结:"
    print_info "缺失的依赖镜像: ${#missing_deps[@]}"
    print_info "缺失的项目镜像: ${#missing_project[@]}"
    
    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        echo ""
        print_warning "缺失的依赖镜像:"
        for img in "${missing_deps[@]}"; do
            echo "  - $img"
        done
    fi
    
    if [[ ${#missing_project[@]} -gt 0 ]]; then
        echo ""
        print_warning "缺失的项目镜像:"
        for img in "${missing_project[@]}"; do
            echo "  - $img"
        done
    fi
    
    # 返回状态
    if [[ ${#missing_deps[@]} -eq 0 && ${#missing_project[@]} -eq 0 ]]; then
        return 0
    else
        return 1
    fi
}

# 修复依赖镜像
fix_dependencies() {
    print_info "开始修复依赖镜像..."
    
    print_info "执行依赖镜像拉取、标记和推送..."
    if "$SCRIPT_DIR/build.sh" deps-all "$REGISTRY" "$TAG"; then
        print_success "依赖镜像推送完成"
        return 0
    else
        print_error "依赖镜像推送失败"
        return 1
    fi
}

# 修复项目镜像
fix_project_images() {
    print_info "开始修复项目镜像..."
    
    print_info "构建并推送项目镜像..."
    if "$SCRIPT_DIR/build.sh" build-push "$REGISTRY" "$TAG"; then
        print_success "项目镜像构建推送完成"
        return 0
    else
        print_error "项目镜像构建推送失败"
        return 1
    fi
}

# 启动生产环境
start_production_env() {
    print_info "启动生产环境..."
    
    if "$SCRIPT_DIR/build.sh" prod-up "$REGISTRY" "$TAG"; then
        print_success "生产环境启动成功"
        return 0
    else
        print_error "生产环境启动失败"
        return 1
    fi
}

# 主修复流程
main_fix() {
    local check_only="$1"
    local skip_deps="$2"
    
    print_info "=========================================="
    print_info "AI-Infra-Matrix Harbor部署修复"
    print_info "=========================================="
    print_info "Registry: $REGISTRY"
    print_info "Tag: $TAG"
    echo ""
    
    # 检查先决条件
    if ! check_prerequisites; then
        return 1
    fi
    
    # 检查当前状态
    check_current_status
    echo ""
    
    # 检查Harbor镜像
    if check_harbor_images; then
        print_success "所有镜像都已存在于Harbor中"
        
        if [[ "$check_only" == "true" ]]; then
            print_info "检查完成，所有镜像就绪"
            return 0
        fi
        
        # 镜像都存在，直接启动
        start_production_env
        return $?
    else
        if [[ "$check_only" == "true" ]]; then
            print_warning "检查完成，发现缺失镜像"
            return 1
        fi
        
        print_warning "发现缺失镜像，开始修复..."
        echo ""
        
        # 修复依赖镜像
        if [[ "$skip_deps" != "true" ]]; then
            if ! fix_dependencies; then
                print_error "依赖镜像修复失败"
                return 1
            fi
            echo ""
        fi
        
        # 修复项目镜像
        if ! fix_project_images; then
            print_error "项目镜像修复失败"
            return 1
        fi
        echo ""
        
        # 启动生产环境
        start_production_env
        return $?
    fi
}

# 解析命令行参数
check_only="false"
skip_deps="false"

while [[ $# -gt 0 ]]; do
    case $1 in
        --registry)
            REGISTRY="$2"
            shift 2
            ;;
        --tag)
            TAG="$2"
            shift 2
            ;;
        --check-only)
            check_only="true"
            shift
            ;;
        --skip-deps)
            skip_deps="true"
            shift
            ;;
        --help)
            show_usage
            exit 0
            ;;
        *)
            print_error "未知参数: $1"
            show_usage
            exit 1
            ;;
    esac
done

# 运行主修复流程
main_fix "$check_only" "$skip_deps"
