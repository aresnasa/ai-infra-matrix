#!/bin/bash

# AI-Infra-Matrix 构建脚本（增强版）
# 目标：一键构建并打包所有组件镜像，版本号自动来自 Git（可覆盖）
# 兼容 macOS bash 3.2

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

#============================
# 版本号与注册表配置
#============================

VERSION=""
REGISTRY="${REGISTRY:-}"
PUSH=""
TAG_LATEST=""
DIRECT_BUILD="true"  # 默认使用直接 docker build，不依赖 docker-compose
NO_CACHE=""
MODE="production"
DO_UP=""
DO_TEST=""
PLATFORMS=""
USE_BUILDX=""
BUILDX_PUSHED=""

# 加载 .env 文件中的环境变量（兼容注释与引号）
source_env_file() {
    local file="$1"
    [ -f "$file" ] || return 0
    while IFS= read -r line || [ -n "$line" ]; do
        # 跳过空行和注释
        [[ -z "$line" || "$line" =~ ^[[:space:]]*# ]] && continue
        # 仅处理 KEY=VALUE 形式
        if [[ "$line" =~ ^([A-Za-z_][A-Za-z0-9_]*)=(.*)$ ]]; then
            local key="${BASH_REMATCH[1]}"
            local val="${BASH_REMATCH[2]}"
            # 去掉首尾空白
            val="${val%%[[:space:]]}"
            val="${val##[[:space:]]}"
            # 去掉包裹引号
            if [[ "$val" =~ ^\".*\"$ ]]; then
                val="${val:1:${#val}-2}"
            elif [[ "$val" =~ ^\'.*\'$ ]]; then
                val="${val:1:${#val}-2}"
            fi
            export "$key=$val"
        fi
    done < "$file"
}

# 推导 Git 版本，回退为分支名或短哈希
detect_version() {
    # 优先从参数/环境获取
    if [ -n "${VERSION:-}" ]; then
        echo "$VERSION"
        return 0
    fi
    local v
    # 尝试使用当前分支名（若形如 vX.Y.Z）
    if v=$(git rev-parse --abbrev-ref HEAD 2>/dev/null); then
        case "$v" in
            v[0-9]*) VERSION="$v" ;;
        esac
    fi
    # 若仍未得到，尝试最近 tag
    if [ -z "$VERSION" ]; then
        if v=$(git describe --tags --abbrev=0 2>/dev/null); then
            VERSION="$v"
        fi
    fi
    # 若仍未得到，用短哈希
    if [ -z "$VERSION" ]; then
        if v=$(git rev-parse --short HEAD 2>/dev/null); then
            VERSION="dev-$v"
        else
            VERSION="dev-unknown"
        fi
    fi
    echo "$VERSION"
}

registry_prefix() {
    if [ -n "$REGISTRY" ]; then
        echo "$REGISTRY/"
    else
        echo ""
    fi
}

tag_args() {
    local name="$1"; shift
    local prefix; prefix=$(registry_prefix)
    local args=("-t" "${name}:$VERSION")
    if [ -n "$prefix" ]; then
        args+=("-t" "${prefix}${name}:$VERSION")
    fi
    if [ -n "$TAG_LATEST" ]; then
        args+=("-t" "${name}:latest")
        if [ -n "$prefix" ]; then
            args+=("-t" "${prefix}${name}:latest")
        fi
    fi
    printf '%s\n' "${args[@]}"
}

#============================
# 单个组件构建器
#============================

build_backend() {
    print_info "构建 backend 与 backend-init (VERSION=$VERSION)"
    if [ -n "$USE_BUILDX" ]; then
        local prefix; prefix=$(registry_prefix)
        local name="ai-infra-backend"
        local tags=()
        if [ -n "$prefix" ]; then
            tags+=("--tag" "${prefix}${name}:$VERSION")
        fi
        tags+=("--tag" "${name}:$VERSION")
        [ -n "$TAG_LATEST" ] && tags+=("--tag" "${name}:latest") && [ -n "$prefix" ] && tags+=("--tag" "${prefix}${name}:latest")
        docker buildx build ${NO_CACHE} \
            --platform "$PLATFORMS" \
            -f src/backend/Dockerfile \
            --build-arg VERSION="$VERSION" \
            ${tags[@]} \
            --push \
            src/backend
        # backend-init uses same image; extra tagging happens on pull side if needed
        docker image inspect "${name}:$VERSION" >/dev/null 2>&1 || true
        BUILDX_PUSHED="true"
    else
        docker build ${NO_CACHE} \
            -f src/backend/Dockerfile \
            --build-arg VERSION="$VERSION" \
            $(tag_args ai-infra-backend) \
            src/backend
        # 派生一份 init 标签（共用同一镜像内容，便于引用）
        docker tag ai-infra-backend:"$VERSION" ai-infra-backend-init:"$VERSION"
        if [ -n "$REGISTRY" ]; then
            docker tag ai-infra-backend:"$VERSION" "$(registry_prefix)"ai-infra-backend-init:"$VERSION"
        fi
        if [ -n "$TAG_LATEST" ]; then
            docker tag ai-infra-backend:"$VERSION" ai-infra-backend:latest || true
            docker tag ai-infra-backend:"$VERSION" ai-infra-backend-init:latest || true
            if [ -n "$REGISTRY" ]; then
                docker tag ai-infra-backend:"$VERSION" "$(registry_prefix)"ai-infra-backend:latest || true
                docker tag ai-infra-backend:"$VERSION" "$(registry_prefix)"ai-infra-backend-init:latest || true
            fi
        fi
    fi
}

build_frontend() {
    print_info "构建 frontend (VERSION=$VERSION)"
    if [ -n "$USE_BUILDX" ]; then
        local prefix; prefix=$(registry_prefix)
        local name="ai-infra-frontend"
        local tags=()
        if [ -n "$prefix" ]; then
            tags+=("--tag" "${prefix}${name}:$VERSION")
        fi
        tags+=("--tag" "${name}:$VERSION")
        [ -n "$TAG_LATEST" ] && tags+=("--tag" "${name}:latest") && [ -n "$prefix" ] && tags+=("--tag" "${prefix}${name}:latest")
        docker buildx build ${NO_CACHE} \
            --platform "$PLATFORMS" \
            -f src/frontend/Dockerfile \
            --build-arg VERSION="$VERSION" \
            --build-arg REACT_APP_API_URL="${REACT_APP_API_URL:-/api}" \
            --build-arg REACT_APP_JUPYTERHUB_URL="${REACT_APP_JUPYTERHUB_URL:-/jupyter}" \
            ${tags[@]} \
            --push \
            src/frontend
    else
        docker build ${NO_CACHE} \
            -f src/frontend/Dockerfile \
            --build-arg VERSION="$VERSION" \
            --build-arg REACT_APP_API_URL="${REACT_APP_API_URL:-/api}" \
            --build-arg REACT_APP_JUPYTERHUB_URL="${REACT_APP_JUPYTERHUB_URL:-/jupyter}" \
            $(tag_args ai-infra-frontend) \
            src/frontend
    fi
}

build_singleuser() {
    print_info "构建 singleuser (VERSION=$VERSION)"
    if [ -n "$USE_BUILDX" ]; then
        local prefix; prefix=$(registry_prefix)
        local name="ai-infra-singleuser"
        local tags=()
        if [ -n "$prefix" ]; then
            tags+=("--tag" "${prefix}${name}:$VERSION")
        fi
        tags+=("--tag" "${name}:$VERSION")
        [ -n "$TAG_LATEST" ] && tags+=("--tag" "${name}:latest") && [ -n "$prefix" ] && tags+=("--tag" "${prefix}${name}:latest")
        docker buildx build ${NO_CACHE} \
            --platform "$PLATFORMS" \
            -f docker/singleuser/Dockerfile \
            --build-arg VERSION="$VERSION" \
            ${tags[@]} \
            --push \
            docker/singleuser
    else
        docker build ${NO_CACHE} \
            -f docker/singleuser/Dockerfile \
            --build-arg VERSION="$VERSION" \
            $(tag_args ai-infra-singleuser) \
            docker/singleuser
    fi
}

build_jupyterhub() {
    print_info "构建 jupyterhub (VERSION=$VERSION)"
    if [ -n "$USE_BUILDX" ]; then
        local prefix; prefix=$(registry_prefix)
        local name="ai-infra-jupyterhub"
        local tags=()
        if [ -n "$prefix" ]; then
            tags+=("--tag" "${prefix}${name}:$VERSION")
        fi
        tags+=("--tag" "${name}:$VERSION")
        [ -n "$TAG_LATEST" ] && tags+=("--tag" "${name}:latest") && [ -n "$prefix" ] && tags+=("--tag" "${prefix}${name}:latest")
        docker buildx build ${NO_CACHE} \
            --platform "$PLATFORMS" \
            -f src/jupyterhub/Dockerfile \
            --build-arg VERSION="$VERSION" \
            ${tags[@]} \
            --push \
            src/jupyterhub
    else
        docker build ${NO_CACHE} \
            -f src/jupyterhub/Dockerfile \
            --build-arg VERSION="$VERSION" \
            $(tag_args ai-infra-jupyterhub) \
            src/jupyterhub
    fi
}

build_nginx() {
    print_info "构建 nginx (VERSION=$VERSION)"
    # 注意：nginx Dockerfile 复制了 repo 根下的资源，构建上下文必须为仓库根目录
    if [ -n "$USE_BUILDX" ]; then
        local prefix; prefix=$(registry_prefix)
        local name="ai-infra-nginx"
        local tags=()
        if [ -n "$prefix" ]; then
            tags+=("--tag" "${prefix}${name}:$VERSION")
        fi
        tags+=("--tag" "${name}:$VERSION")
        [ -n "$TAG_LATEST" ] && tags+=("--tag" "${name}:latest") && [ -n "$prefix" ] && tags+=("--tag" "${prefix}${name}:latest")
        docker buildx build ${NO_CACHE} \
            --platform "$PLATFORMS" \
            -f src/nginx/Dockerfile \
            --build-arg VERSION="$VERSION" \
            --build-arg DEBUG_MODE="${DEBUG_MODE:-false}" \
            --build-arg BUILD_ENV="${BUILD_ENV:-$MODE}" \
            ${tags[@]} \
            --push \
            .
    else
        docker build ${NO_CACHE} \
            -f src/nginx/Dockerfile \
            --build-arg VERSION="$VERSION" \
            --build-arg DEBUG_MODE="${DEBUG_MODE:-false}" \
            --build-arg BUILD_ENV="${BUILD_ENV:-$MODE}" \
            $(tag_args ai-infra-nginx) \
            .
    fi
}

build_gitea() {
    print_info "构建 gitea (VERSION=$VERSION)"
    if [ -n "$USE_BUILDX" ]; then
        local prefix; prefix=$(registry_prefix)
        local name="ai-infra-gitea"
        local tags=()
        if [ -n "$prefix" ]; then
            tags+=("--tag" "${prefix}${name}:$VERSION")
        fi
        tags+=("--tag" "${name}:$VERSION")
        [ -n "$TAG_LATEST" ] && tags+=("--tag" "${name}:latest") && [ -n "$prefix" ] && tags+=("--tag" "${prefix}${name}:latest")
        docker buildx build ${NO_CACHE} \
            --platform "$PLATFORMS" \
            -f third-party/gitea/Dockerfile \
            --build-arg VERSION="$VERSION" \
            ${tags[@]} \
            --push \
            third-party/gitea
    else
        docker build ${NO_CACHE} \
            -f third-party/gitea/Dockerfile \
            --build-arg VERSION="$VERSION" \
            $(tag_args ai-infra-gitea) \
            third-party/gitea
    fi
}

push_image_if_needed() {
    local name="$1"
    if [ -z "$PUSH" ] || [ -z "$REGISTRY" ]; then
        return 0
    fi
    local prefix; prefix=$(registry_prefix)
    print_info "推送镜像到 $REGISTRY: $name:$VERSION"
    docker push "${prefix}${name}:$VERSION"
    if [ -n "$TAG_LATEST" ]; then
        docker push "${prefix}${name}:latest" || true
    fi
}

push_all_if_needed() {
    for n in ai-infra-backend ai-infra-backend-init ai-infra-frontend ai-infra-singleuser ai-infra-jupyterhub ai-infra-nginx ai-infra-gitea; do
        push_image_if_needed "$n"
    done
}

# 预拉取基础镜像（支持国内镜像源回退）
MIRRORS=(
    "docker.m.daocloud.io"
    "dockerproxy.com"
    "hub-mirror.c.163.com"
    "registry.docker-cn.com"
)

BASE_IMAGES=()

# 解析 docker-compose.yml 中的镜像列表（兼容 macOS bash 3.2）
collect_compose_images() {
    local compose_files=()
    local script_dir
    script_dir=$(cd "$(dirname "$0")" && pwd)
    local repo_root
    repo_root=$(cd "$script_dir/.." && pwd)

    # 收集候选 compose 文件（根目录 + 生产目录）
    [ -f "$repo_root/docker-compose.yml" ] && compose_files+=("$repo_root/docker-compose.yml")
    [ -f "$repo_root/src/docker/production/docker-compose.yml" ] && compose_files+=("$repo_root/src/docker/production/docker-compose.yml")

    local images_list
    images_list=$(
        for f in "${compose_files[@]}"; do
            grep -E '^[[:space:]]*image:[[:space:]]' "$f" | \
                sed -E 's/^[[:space:]]*image:[[:space:]]*//' | \
                sed -E 's/[[:space:]]+#.*$//' | \
                tr -d '"' | tr -d "'" || true
        done | \
        grep -vE '^(ai-infra-|\$\{)' | \
        awk 'NF{print $1}' | sort -u
    )

    BASE_IMAGES=()
    while IFS= read -r img; do
        [ -n "$img" ] && BASE_IMAGES+=("$img")
    done <<< "$images_list"
}

pull_image() {
    local image="$1"
    local pulled=false

    # 若本地已存在且未强制更新，则跳过
    if [ -z "$UPDATE_IMAGES" ]; then
        if docker image inspect "$image" >/dev/null 2>&1; then
            print_info "镜像已存在，跳过拉取: $image"
            return 0
        fi
    fi
    for mirror in "${MIRRORS[@]}"; do
        # 官方库镜像尝试 library/ 前缀
        local candidates=()
        if echo "$image" | grep -q '/'; then
            candidates+=("$mirror/$image")
        else
            candidates+=("$mirror/library/$image" "$mirror/$image")
        fi
        for mirrored_image in "${candidates[@]}"; do
            print_info "尝试从镜像源拉取: $mirrored_image"
            if docker pull "$mirrored_image" >/dev/null 2>&1; then
                print_success "从镜像源拉取成功: $mirrored_image"
                docker tag "$mirrored_image" "$image" >/dev/null 2>&1 || true
                pulled=true
                break
            else
                print_warning "镜像源拉取失败: $mirrored_image"
            fi
        done
        [ "$pulled" = true ] && break
    done
    if [ "$pulled" != true ]; then
        print_info "从官方 Docker Hub 拉取: $image"
        if docker pull "$image"; then
            print_success "官方拉取成功: $image"
        else
            print_warning "官方拉取失败（可能是网络超时）: $image"
        fi
    fi
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
    echo "  --version X         - 指定镜像版本（默认从git自动推导）"
    echo "  --registry R        - 指定镜像注册表前缀（如 registry.local:5000）"
    echo "  --push              - 构建后推送到注册表（需要 --registry）"
    echo "  --tag-latest        - 额外打 latest 标签"
    echo "  --no-cache          - 无缓存构建"
    echo "  --rebuild           - (仅compose路径) 强制重建所有服务"
    echo "  --multi-arch        - 多架构构建 (linux/amd64,linux/arm64)，需配合 --registry --push 使用"
    echo "  --platforms P       - 指定平台列表 (例如 linux/amd64,linux/arm64)，需配合 --registry --push 使用"
    echo "  --nginx-only        - 只构建nginx服务"
    echo "  --skip-prepull      - 跳过预拉取基础镜像"
    echo "  --update-images     - 强制更新（即使本地存在也重新拉取）"
    echo "  --compose           - 使用 docker-compose build（默认直接 docker build）"
    echo "  --up                - 构建后通过 compose 启动/更新服务 (up -d)"
    echo "  --test              - 构建/启动后运行 scripts/test-health.sh 健康检查"
    echo "  -h, --help          - 显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0 dev                          - 开发模式构建（自动版本）"
    echo "  $0 prod --version v0.0.3.3      - 指定版本号构建"
    echo "  $0 prod --registry localhost:5000 --push --tag-latest  - 构建并推送到本地仓库"
}

# 其他默认参数
REBUILD=""
NGINX_ONLY=""
SKIP_PREPULL=""
UPDATE_IMAGES=""

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
        --version)
            VERSION="$2"; shift 2 ;;
        --registry)
            REGISTRY="$2"; shift 2 ;;
        --push)
            PUSH="true"; shift ;;
        --tag-latest)
            TAG_LATEST="true"; shift ;;
        --no-cache)
            NO_CACHE="--no-cache"
            shift
            ;;
        --multi-arch)
            PLATFORMS="linux/amd64,linux/arm64"
            shift
            ;;
        --platforms)
            PLATFORMS="$2"; shift 2 ;;
        --rebuild)
            REBUILD="--force-recreate"
            shift
            ;;
        --nginx-only)
            NGINX_ONLY="nginx"
            shift
            ;;
        --skip-prepull)
            SKIP_PREPULL="true"
            shift
            ;;
        --update-images)
            UPDATE_IMAGES="true"
            shift
            ;;
        --compose)
            DIRECT_BUILD=""  # 关闭直接构建，走 compose
            shift
            ;;
        --up)
            DO_UP="true"
            shift
            ;;
        --test)
            DO_TEST="true"
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
VERSION=$(detect_version)
export IMAGE_TAG="$VERSION"
print_info "镜像版本: ${VERSION}"
print_info "构建时间: $(date)"

# 判断是否启用 buildx（当指定了平台并且需要推送时）
if [ -n "$PLATFORMS" ]; then
    if [ -n "$PUSH" ] && [ -n "$REGISTRY" ]; then
        USE_BUILDX="true"
        print_info "启用 Buildx 多架构构建: $PLATFORMS (将直接 --push)"
    else
        print_warning "检测到 --platforms，但未指定 --registry/--push；将回退为单架构本地构建"
        PLATFORMS=""
    fi
fi

# 设置环境变量文件
if [ "$MODE" = "development" ]; then
    ENV_FILE=".env"
    export DEBUG_MODE=true
    export BUILD_ENV=development
    print_info "使用开发环境配置: $ENV_FILE"
    print_warning "调试工具将被启用"
else
    ENV_FILE=".env.prod"
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

# 优先加载根目录 .env（通用变量），再加载模式专用 env 文件（覆盖）
if [ -f ".env" ] && [ "$ENV_FILE" != ".env" ]; then
    print_info "加载通用环境变量: .env"
    source_env_file ".env"
fi
if [ -f "$ENV_FILE" ]; then
    print_info "加载模式环境变量: $ENV_FILE"
    source_env_file "$ENV_FILE"
fi

# 检查Docker是否可用
if ! command -v docker &> /dev/null; then
    print_error "Docker 未安装或不可用"
    exit 1
fi

# 选择 docker compose 命令（优先 v2: docker compose，其次 v1: docker-compose）
COMPOSE_BIN=""
if docker compose version >/dev/null 2>&1; then
    COMPOSE_BIN="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
    COMPOSE_BIN="docker-compose"
fi

if [ -z "$DIRECT_BUILD" ]; then
    if [ -z "$COMPOSE_BIN" ]; then
        print_error "未检测到 docker compose 或 docker-compose"
        exit 1
    fi
fi

# 构建服务
print_info "开始构建服务..."

if [ -z "$SKIP_PREPULL" ]; then
    # 先预拉取基础镜像，减少构建阶段超时
    print_info "扫描 docker-compose.yml 以收集基础镜像..."
    collect_compose_images
    if [ ${#BASE_IMAGES[@]} -eq 0 ]; then
        print_warning "未在 compose 中发现可预拉取的镜像，跳过"
    else
        print_info "将预拉取以下镜像 (${#BASE_IMAGES[@]}): ${BASE_IMAGES[*]}"
    fi
    print_info "开始预拉取基础镜像以提高构建稳定性..."
    for img in "${BASE_IMAGES[@]}"; do
        pull_image "$img"
    done
    print_success "基础镜像预拉取完成"
else
    print_warning "跳过基础镜像预拉取 (--skip-prepull)"
fi

if [ -z "$DIRECT_BUILD" ]; then
    # 使用 docker-compose 构建路径（兼容旧流程）
    if [ -n "$NGINX_ONLY" ]; then
        print_info "仅构建 nginx 服务 (compose)"
        SERVICES="nginx"
    else
        SERVICES=""
    fi
    BUILD_CMD="$COMPOSE_BIN"
    # 仅当为 v2 (docker compose) 才支持 --env-file
    if [ -f "$ENV_FILE" ] && [ "$COMPOSE_BIN" = "docker compose" ]; then
        BUILD_CMD="$BUILD_CMD --env-file $ENV_FILE"
    fi
    # 让 compose 也能获得版本号
    export IMAGE_TAG
    BUILD_CMD="$BUILD_CMD build $NO_CACHE $SERVICES"
    print_info "执行构建命令: $BUILD_CMD"
    eval $BUILD_CMD
else
    # 直接 docker build 路径
    [ -z "$NGINX_ONLY" ] && build_backend
    [ -z "$NGINX_ONLY" ] && build_frontend
    [ -z "$NGINX_ONLY" ] && build_singleuser
    [ -z "$NGINX_ONLY" ] && build_jupyterhub
    [ -z "$NGINX_ONLY" ] && build_gitea
    build_nginx
fi

print_success "镜像构建完成"
if [ -n "$USE_BUILDX" ] && [ -n "$PUSH" ]; then
    print_info "已通过 buildx --push 推送多架构镜像，跳过二次推送"
else
    push_all_if_needed
fi

# 启动服务（--up 时执行）
if [ -n "$DO_UP" ]; then
    if [ -z "$COMPOSE_BIN" ]; then
        print_warning "未检测到 compose，跳过启动 (--up)"
    else
        print_info "启动/更新服务..."
        
        # 检查是否有改进的启动脚本
        SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
        IMPROVED_STARTUP="$SCRIPT_DIR/start-services-improved.sh"
        
        if [ -x "$IMPROVED_STARTUP" ]; then
            print_info "使用改进的分阶段启动脚本..."
            if [ -n "$DO_TEST" ]; then
                "$IMPROVED_STARTUP" --test
            else
                "$IMPROVED_STARTUP"
            fi
        else
            # 原有的启动逻辑
            START_CMD="$COMPOSE_BIN"
            # 仅 v2 支持 --env-file
            if [ -f "$ENV_FILE" ] && [ "$COMPOSE_BIN" = "docker compose" ]; then
                START_CMD="$START_CMD --env-file $ENV_FILE"
            fi
            if [ -n "$NGINX_ONLY" ]; then
                START_CMD="$START_CMD up -d $REBUILD nginx"
            else
                START_CMD="$START_CMD up -d $REBUILD"
            fi
            print_info "执行启动命令: $START_CMD"
            if eval $START_CMD; then
                print_success "服务启动完成!"
                print_info "等待服务稳定..."
                sleep 30
            else
                print_error "服务启动失败!"
                exit 1
            fi
        fi
    fi
fi

# 运行健康检查（--test 时执行，但如果已经在启动脚本中运行过则跳过）
if [ -n "$DO_TEST" ]; then
    SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
    IMPROVED_STARTUP="$SCRIPT_DIR/start-services-improved.sh"
    
    # 如果使用了改进的启动脚本且已经运行过测试，则跳过
    if [ -x "$IMPROVED_STARTUP" ] && [ -n "$DO_UP" ]; then
        print_info "健康检查已在启动脚本中执行，跳过重复检查"
    elif [ -x "$SCRIPT_DIR/test-health.sh" ]; then
        print_info "运行健康检查脚本: $SCRIPT_DIR/test-health.sh"
        if "$SCRIPT_DIR/test-health.sh"; then
            print_success "健康检查通过"
        else
            print_error "健康检查失败"
            exit 1
        fi
    else
        print_warning "未找到可执行的测试脚本: $SCRIPT_DIR/test-health.sh"
    fi
fi

# 显示完成信息
echo ""
echo "🎉 构建完成!"
echo "================================"
# 若 .env 中提供了 IMAGE_TAG 或 VERSION，优先生效
if [ -z "${VERSION:-}" ] && [ -n "${IMAGE_TAG:-}" ]; then
    VERSION="$IMAGE_TAG"
fi
print_info "构建模式: $MODE"
print_info "镜像版本: ${VERSION}"
print_info "服务访问:"
echo "  🌐 前端应用: http://localhost:8080"
echo "  🔐 SSO登录: http://localhost:8080/sso/"
echo "  📊 JupyterHub: http://localhost:8080/jupyter"
echo "  🗃️  Gitea: http://localhost:8080/gitea/"

if [ "$MODE" = "development" ]; then
    echo "  🔧 调试工具: http://localhost:8080/debug/"
    print_warning "调试模式已启用，生产环境请使用 prod 模式构建"
fi

if [ -n "$COMPOSE_BIN" ]; then
    print_info "查看服务状态: $COMPOSE_BIN ps"
    print_info "查看日志: $COMPOSE_BIN logs -f [服务名]"
else
    print_info "查看服务状态: docker compose ps"
    print_info "查看日志: docker compose logs -f [服务名]"
fi

# 输出镜像摘要
echo ""
print_info "本地镜像（ai-infra-*:${VERSION}）:"
docker images | grep "ai-infra-" | grep "${VERSION}" || true
