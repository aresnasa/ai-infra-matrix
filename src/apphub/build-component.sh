#!/bin/bash
# AppHub 组件构建脚本
# 支持选择性构建特定组件

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 默认值
BUILD_SLURM=true
BUILD_SALTSTACK=true
BUILD_CATEGRAF=true
BUILD_SINGULARITY=false
SLURM_VERSION="25.05.4"
SALTSTACK_VERSION="v3007.8"
CATEGRAF_VERSION="v0.4.23"
TAG="latest"

# 帮助信息
show_help() {
    cat << EOF
AppHub Component Builder

Usage: $0 [OPTIONS]

Options:
    --component COMP      Build specific component (slurm, saltstack, categraf, singularity, all)
    --slurm-only          Build only SLURM packages
    --saltstack-only      Build only SaltStack packages
    --categraf-only       Build only Categraf packages
    --no-slurm            Skip SLURM build
    --no-saltstack        Skip SaltStack build
    --no-categraf         Skip Categraf build
    --slurm-version VER   SLURM version (default: ${SLURM_VERSION})
    --salt-version VER    SaltStack version (default: ${SALTSTACK_VERSION})
    --categraf-version VER Categraf version (default: ${CATEGRAF_VERSION})
    --tag TAG             Docker image tag (default: ${TAG})
    --help                Show this help message

Examples:
    # Build all components
    $0

    # Build only SLURM
    $0 --slurm-only

    # Build SLURM and SaltStack, skip Categraf
    $0 --no-categraf

    # Build with specific SLURM version
    $0 --slurm-version 25.05.5

    # Build specific component
    $0 --component slurm
EOF
}

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    case $1 in
        --component)
            COMPONENT="$2"
            case "$COMPONENT" in
                slurm)
                    BUILD_SLURM=true
                    BUILD_SALTSTACK=false
                    BUILD_CATEGRAF=false
                    ;;
                saltstack|salt)
                    BUILD_SLURM=false
                    BUILD_SALTSTACK=true
                    BUILD_CATEGRAF=false
                    ;;
                categraf)
                    BUILD_SLURM=false
                    BUILD_SALTSTACK=false
                    BUILD_CATEGRAF=true
                    ;;
                all)
                    BUILD_SLURM=true
                    BUILD_SALTSTACK=true
                    BUILD_CATEGRAF=true
                    ;;
                *)
                    echo "Error: Unknown component: $COMPONENT"
                    exit 1
                    ;;
            esac
            shift 2
            ;;
        --slurm-only)
            BUILD_SLURM=true
            BUILD_SALTSTACK=false
            BUILD_CATEGRAF=false
            shift
            ;;
        --saltstack-only|--salt-only)
            BUILD_SLURM=false
            BUILD_SALTSTACK=true
            BUILD_CATEGRAF=false
            shift
            ;;
        --categraf-only)
            BUILD_SLURM=false
            BUILD_SALTSTACK=false
            BUILD_CATEGRAF=true
            shift
            ;;
        --no-slurm)
            BUILD_SLURM=false
            shift
            ;;
        --no-saltstack|--no-salt)
            BUILD_SALTSTACK=false
            shift
            ;;
        --no-categraf)
            BUILD_CATEGRAF=false
            shift
            ;;
        --slurm-version)
            SLURM_VERSION="$2"
            shift 2
            ;;
        --salt-version|--saltstack-version)
            SALTSTACK_VERSION="$2"
            shift 2
            ;;
        --categraf-version)
            CATEGRAF_VERSION="$2"
            shift 2
            ;;
        --tag)
            TAG="$2"
            shift 2
            ;;
        --help|-h)
            show_help
            exit 0
            ;;
        *)
            echo "Error: Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# 显示构建配置
echo "========================================"
echo "AppHub Component Build Configuration"
echo "========================================"
echo "SLURM:      ${BUILD_SLURM} (v${SLURM_VERSION})"
echo "SaltStack:  ${BUILD_SALTSTACK} (${SALTSTACK_VERSION})"
echo "Categraf:   ${BUILD_CATEGRAF} (${CATEGRAF_VERSION})"
echo "Singularity: ${BUILD_SINGULARITY}"
echo "Tag:        ${TAG}"
echo "========================================"
echo ""

# 检查 SLURM tarball
if [ "$BUILD_SLURM" = "true" ]; then
    SLURM_TARBALL="slurm-${SLURM_VERSION}.tar.bz2"
    if [ ! -f "$SLURM_TARBALL" ]; then
        echo "❌ Error: SLURM tarball not found: $SLURM_TARBALL"
        echo ""
        echo "Please download it from:"
        echo "  https://download.schedmd.com/slurm/slurm-${SLURM_VERSION}.tar.bz2"
        echo ""
        echo "Or run:"
        echo "  wget https://download.schedmd.com/slurm/slurm-${SLURM_VERSION}.tar.bz2"
        echo ""
        exit 1
    fi
    echo "✓ Found SLURM tarball: $SLURM_TARBALL"
fi

# 构建 Docker 镜像
echo ""
echo "Starting Docker build..."
echo ""

docker build \
    --build-arg BUILD_SLURM="$BUILD_SLURM" \
    --build-arg BUILD_SALTSTACK="$BUILD_SALTSTACK" \
    --build-arg BUILD_CATEGRAF="$BUILD_CATEGRAF" \
    --build-arg BUILD_SINGULARITY="$BUILD_SINGULARITY" \
    --build-arg SLURM_VERSION="$SLURM_VERSION" \
    --build-arg SALTSTACK_VERSION="$SALTSTACK_VERSION" \
    --build-arg CATEGRAF_VERSION="$CATEGRAF_VERSION" \
    -t "ai-infra-apphub:${TAG}" \
    -f Dockerfile \
    .

echo ""
echo "========================================"
echo "✓ Build completed successfully"
echo "========================================"
echo "Image: ai-infra-apphub:${TAG}"
echo ""
echo "To run the container:"
echo "  docker run -d -p 8081:80 ai-infra-apphub:${TAG}"
echo ""
