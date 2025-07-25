#!/bin/bash

# JupyterHub Docker构建脚本 - 带网络重试机制
set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 默认参数
IMAGE_NAME="ai-infra-jupyterhub"
TAG="conda-env"
MAX_RETRIES=3
RETRY_DELAY=10

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    case $1 in
        --image-name)
            IMAGE_NAME="$2"
            shift 2
            ;;
        --tag)
            TAG="$2"
            shift 2
            ;;
        --max-retries)
            MAX_RETRIES="$2"
            shift 2
            ;;
        --retry-delay)
            RETRY_DELAY="$2"
            shift 2
            ;;
        -h|--help)
            echo "用法: $0 [选项]"
            echo "选项:"
            echo "  --image-name NAME    设置镜像名称 (默认: ai-infra-jupyterhub)"
            echo "  --tag TAG           设置镜像标签 (默认: conda-env)"
            echo "  --max-retries N     最大重试次数 (默认: 3)"
            echo "  --retry-delay N     重试间隔秒数 (默认: 10)"
            echo "  -h, --help          显示此帮助信息"
            exit 0
            ;;
        *)
            print_error "未知选项: $1"
            exit 1
            ;;
    esac
done

# 检查Docker是否运行
check_docker() {
    if ! docker info >/dev/null 2>&1; then
        print_error "Docker未运行或无法访问"
        exit 1
    fi
}

# 网络连通性测试
test_network() {
    print_info "测试网络连通性..."
    
    if curl -s --connect-timeout 10 https://registry-1.docker.io/v2/ >/dev/null; then
        print_success "Docker Hub连接正常"
        return 0
    else
        print_warning "Docker Hub连接异常，将使用重试机制"
        return 1
    fi
}

# 构建Docker镜像
build_image() {
    local attempt=$1
    local full_image_name="${IMAGE_NAME}:${TAG}"
    
    print_info "尝试构建镜像 (第 ${attempt}/${MAX_RETRIES} 次): ${full_image_name}"
    
    # 设置构建参数
    local build_args=(
        "--progress=plain"
        "--no-cache"
        "-t" "${full_image_name}"
        "."
    )
    
    # 如果不是第一次尝试，使用更长的超时时间
    if [ "$attempt" -gt 1 ]; then
        build_args+=("--network=host")
    fi
    
    # 执行构建
    if timeout 1800 docker build "${build_args[@]}"; then
        print_success "镜像构建成功: ${full_image_name}"
        return 0
    else
        print_error "构建失败 (第 ${attempt} 次尝试)"
        return 1
    fi
}

# 清理Docker资源
cleanup_docker() {
    print_info "清理Docker构建缓存..."
    docker builder prune -f >/dev/null 2>&1 || true
    docker system prune -f >/dev/null 2>&1 || true
}

# 主构建流程
main() {
    print_info "开始构建JupyterHub Docker镜像"
    print_info "镜像名称: ${IMAGE_NAME}:${TAG}"
    print_info "最大重试次数: ${MAX_RETRIES}"
    
    # 检查Docker
    check_docker
    
    # 测试网络
    test_network
    
    # 尝试构建
    for attempt in $(seq 1 $MAX_RETRIES); do
        if build_image "$attempt"; then
            print_success "构建完成！"
            
            # 显示镜像信息
            print_info "镜像详情:"
            docker images "${IMAGE_NAME}:${TAG}" --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}"
            
            # 运行快速测试
            print_info "运行快速测试..."
            if docker run --rm "${IMAGE_NAME}:${TAG}" python --version; then
                print_success "镜像测试通过"
            else
                print_warning "镜像测试失败，但构建成功"
            fi
            
            exit 0
        fi
        
        if [ "$attempt" -lt "$MAX_RETRIES" ]; then
            print_warning "等待 ${RETRY_DELAY} 秒后重试..."
            cleanup_docker
            sleep "$RETRY_DELAY"
        fi
    done
    
    print_error "所有构建尝试均失败"
    print_info "建议检查网络连接或稍后重试"
    exit 1
}

# 运行主函数
main "$@"
