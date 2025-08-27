#!/bin/bash
# 修复生产环境 backend 服务问题的脚本

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

# 检查当前 backend 容器状态
check_backend_status() {
    print_info "检查当前 backend 容器状态..."
    
    if docker ps -a --format "table {{.Names}}\t{{.Status}}\t{{.Command}}" | grep ai-infra-backend; then
        print_info "发现 backend 容器"
    else
        print_warning "未找到 backend 容器"
    fi
    
    # 检查容器的 CMD
    container_id=$(docker ps -aq --filter "name=ai-infra-backend" | head -1)
    if [[ -n "$container_id" ]]; then
        print_info "检查容器运行的命令..."
        docker inspect "$container_id" --format='{{.Config.Cmd}}' || true
        docker inspect "$container_id" --format='{{.Config.Entrypoint}}' || true
    fi
}

# 重新构建正确的 backend 镜像
rebuild_backend_image() {
    print_info "重新构建 backend 镜像..."
    
    # 确保我们在项目根目录
    cd "$(dirname "$0")"
    
    # 检查构建脚本
    if [[ ! -f "build.sh" ]]; then
        print_error "未找到 build.sh 脚本"
        return 1
    fi
    
    # 重新构建 backend 镜像（不包含 --target 参数，使用默认阶段）
    print_info "构建 backend 镜像（默认阶段）..."
    
    # 使用本地标签重新构建
    if docker build -f src/backend/Dockerfile -t ai-infra-backend:v0.3.5 .; then
        print_success "backend 镜像构建成功"
    else
        print_error "backend 镜像构建失败"
        return 1
    fi
    
    # 如果需要，也重新构建 backend-init 镜像
    print_info "构建 backend-init 镜像（backend-init 阶段）..."
    if docker build -f src/backend/Dockerfile --target backend-init -t ai-infra-backend-init:v0.3.5 .; then
        print_success "backend-init 镜像构建成功"
    else
        print_error "backend-init 镜像构建失败"
        return 1
    fi
}

# 验证镜像
verify_images() {
    print_info "验证构建的镜像..."
    
    # 检查 backend 镜像的 CMD
    print_info "backend 镜像的 CMD:"
    docker inspect ai-infra-backend:v0.3.5 --format='{{.Config.Cmd}}' || true
    
    print_info "backend-init 镜像的 CMD:"
    docker inspect ai-infra-backend-init:v0.3.5 --format='{{.Config.Cmd}}' || true
    
    # 验证 backend 镜像包含正确的二进制文件
    print_info "检查 backend 镜像中的文件..."
    docker run --rm ai-infra-backend:v0.3.5 ls -la /root/ | grep -E "(main|init)"
}

# 修复 docker-compose 配置
fix_compose_config() {
    local compose_file="${1:-docker-compose.yml}"
    
    print_info "检查 $compose_file 配置..."
    
    if [[ ! -f "$compose_file" ]]; then
        print_error "文件不存在: $compose_file"
        return 1
    fi
    
    # 检查 backend 服务是否有错误的 command 配置
    if grep -A 10 "^  backend:" "$compose_file" | grep -q "command:.*init"; then
        print_warning "发现 backend 服务中有 init 命令配置"
        print_info "请手动检查并移除错误的 command 配置"
    else
        print_success "backend 服务配置看起来正确"
    fi
}

# 重启服务
restart_services() {
    local compose_file="${1:-docker-compose.yml}"
    
    print_info "重启 backend 相关服务..."
    
    # 停止当前的 backend 容器
    print_info "停止 backend 容器..."
    docker-compose -f "$compose_file" stop backend || true
    
    # 删除容器以确保重新创建
    print_info "删除 backend 容器..."
    docker-compose -f "$compose_file" rm -f backend || true
    
    # 重新启动服务
    print_info "启动 backend 服务..."
    docker-compose -f "$compose_file" up -d backend
    
    # 等待几秒钟
    sleep 5
    
    # 检查状态
    print_info "检查服务状态..."
    docker-compose -f "$compose_file" ps backend
    
    # 查看日志
    print_info "查看 backend 服务日志（最后20行）..."
    docker-compose -f "$compose_file" logs --tail=20 backend
}

# 主函数
main() {
    echo "🔧 修复生产环境 backend 服务问题"
    echo "=================================="
    
    # 解析参数
    compose_file="docker-compose.yml"
    rebuild_flag=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            -f|--file)
                compose_file="$2"
                shift 2
                ;;
            --rebuild)
                rebuild_flag=true
                shift
                ;;
            -h|--help)
                echo "用法: $0 [选项]"
                echo "选项:"
                echo "  -f, --file FILE    指定 docker-compose 文件 (默认: docker-compose.yml)"
                echo "  --rebuild          重新构建镜像"
                echo "  -h, --help         显示帮助信息"
                exit 0
                ;;
            *)
                print_error "未知选项: $1"
                exit 1
                ;;
        esac
    done
    
    print_info "使用 docker-compose 文件: $compose_file"
    
    # 1. 检查当前状态
    check_backend_status
    
    # 2. 如果指定了重新构建，则重新构建镜像
    if [[ "$rebuild_flag" == true ]]; then
        rebuild_backend_image
        verify_images
    fi
    
    # 3. 修复配置
    fix_compose_config "$compose_file"
    
    # 4. 重启服务
    read -p "是否现在重启 backend 服务？(y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        restart_services "$compose_file"
    else
        print_info "跳过重启，请手动重启服务："
        print_info "docker-compose -f $compose_file up -d backend"
    fi
    
    print_success "修复脚本执行完成！"
    echo
    print_info "如果问题仍然存在，请检查："
    print_info "1. 确保使用的是正确的镜像标签"
    print_info "2. 检查镜像是否正确构建（没有使用 --target backend-init）"
    print_info "3. 查看详细的容器日志: docker-compose -f $compose_file logs backend"
}

# 运行主函数
main "$@"
