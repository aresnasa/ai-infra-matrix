#!/bin/bash

# JupyterHub Docker镜像构建脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
REGISTRY=${REGISTRY:-"localhost:5000"}
VERSION=${VERSION:-"latest"}
GPU_IMAGE_NAME="jupyterhub-python-gpu"
CPU_IMAGE_NAME="jupyterhub-python-cpu"

echo -e "${BLUE}=== JupyterHub Docker镜像构建脚本 ===${NC}"

# 函数：打印状态
print_status() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查Docker是否运行
check_docker() {
    if ! docker info >/dev/null 2>&1; then
        print_error "Docker未运行，请启动Docker"
        exit 1
    fi
    print_status "Docker运行正常"
}

# 构建GPU镜像
build_gpu_image() {
    print_status "开始构建GPU镜像: $GPU_IMAGE_NAME:$VERSION"
    
    if [ ! -f "docker/jupyterhub-gpu/Dockerfile" ]; then
        print_error "GPU Dockerfile不存在: docker/jupyterhub-gpu/Dockerfile"
        exit 1
    fi
    
    docker build \
        -t "$REGISTRY/$GPU_IMAGE_NAME:$VERSION" \
        -t "$REGISTRY/$GPU_IMAGE_NAME:latest" \
        -f docker/jupyterhub-gpu/Dockerfile \
        docker/jupyterhub-gpu/
    
    print_status "GPU镜像构建完成"
}

# 构建CPU镜像
build_cpu_image() {
    print_status "开始构建CPU镜像: $CPU_IMAGE_NAME:$VERSION"
    
    if [ ! -f "docker/jupyterhub-cpu/Dockerfile" ]; then
        print_error "CPU Dockerfile不存在: docker/jupyterhub-cpu/Dockerfile"
        exit 1
    fi
    
    docker build \
        -t "$REGISTRY/$CPU_IMAGE_NAME:$VERSION" \
        -t "$REGISTRY/$CPU_IMAGE_NAME:latest" \
        -f docker/jupyterhub-cpu/Dockerfile \
        docker/jupyterhub-cpu/
    
    print_status "CPU镜像构建完成"
}

# 推送镜像
push_images() {
    if [ "$PUSH" = "true" ]; then
        print_status "开始推送镜像到注册表: $REGISTRY"
        
        docker push "$REGISTRY/$GPU_IMAGE_NAME:$VERSION"
        docker push "$REGISTRY/$GPU_IMAGE_NAME:latest"
        docker push "$REGISTRY/$CPU_IMAGE_NAME:$VERSION"
        docker push "$REGISTRY/$CPU_IMAGE_NAME:latest"
        
        print_status "镜像推送完成"
    fi
}

# 测试镜像
test_images() {
    print_status "测试GPU镜像..."
    docker run --rm "$REGISTRY/$GPU_IMAGE_NAME:$VERSION" python -c "import torch; print(f'PyTorch版本: {torch.__version__}'); print(f'CUDA可用: {torch.cuda.is_available()}')"
    
    print_status "测试CPU镜像..."
    docker run --rm "$REGISTRY/$CPU_IMAGE_NAME:$VERSION" python -c "import numpy as np; import pandas as pd; print('基础库测试通过')"
    
    print_status "镜像测试完成"
}

# 显示使用说明
show_usage() {
    echo "使用方法:"
    echo "  $0 [选项]"
    echo ""
    echo "选项:"
    echo "  -h, --help     显示帮助信息"
    echo "  -p, --push     构建后推送镜像到注册表"
    echo "  -t, --test     构建后测试镜像"
    echo "  --gpu-only     只构建GPU镜像"
    echo "  --cpu-only     只构建CPU镜像"
    echo ""
    echo "环境变量:"
    echo "  REGISTRY       镜像注册表地址 (默认: localhost:5000)"
    echo "  VERSION        镜像版本标签 (默认: latest)"
    echo ""
    echo "示例:"
    echo "  $0 --push --test"
    echo "  REGISTRY=my-registry.com VERSION=v1.0.0 $0 --push"
}

# 主函数
main() {
    local BUILD_GPU=true
    local BUILD_CPU=true
    local PUSH=false
    local TEST=false
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_usage
                exit 0
                ;;
            -p|--push)
                PUSH=true
                shift
                ;;
            -t|--test)
                TEST=true
                shift
                ;;
            --gpu-only)
                BUILD_CPU=false
                shift
                ;;
            --cpu-only)
                BUILD_GPU=false
                shift
                ;;
            *)
                print_error "未知选项: $1"
                show_usage
                exit 1
                ;;
        esac
    done
    
    # 检查环境
    check_docker
    
    print_status "构建配置:"
    print_status "  注册表: $REGISTRY"
    print_status "  版本: $VERSION"
    print_status "  构建GPU镜像: $BUILD_GPU"
    print_status "  构建CPU镜像: $BUILD_CPU"
    print_status "  推送镜像: $PUSH"
    print_status "  测试镜像: $TEST"
    
    # 执行构建
    if [ "$BUILD_GPU" = "true" ]; then
        build_gpu_image
    fi
    
    if [ "$BUILD_CPU" = "true" ]; then
        build_cpu_image
    fi
    
    # 推送镜像
    push_images
    
    # 测试镜像
    if [ "$TEST" = "true" ]; then
        test_images
    fi
    
    print_status "所有操作完成！"
    
    # 显示镜像信息
    echo ""
    print_status "构建的镜像:"
    if [ "$BUILD_GPU" = "true" ]; then
        echo "  GPU: $REGISTRY/$GPU_IMAGE_NAME:$VERSION"
    fi
    if [ "$BUILD_CPU" = "true" ]; then
        echo "  CPU: $REGISTRY/$CPU_IMAGE_NAME:$VERSION"
    fi
}

# 运行主函数
main "$@"
