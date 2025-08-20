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
PULL=""
TAG_LATEST=""
DIRECT_BUILD="true"  # 默认使用直接 docker build，不依赖 docker-compose
NO_CACHE=""
MODE="production"
DO_UP=""
DO_TEST=""
PLATFORMS=""
USE_BUILDX=""
BUILDX_PUSHED=""
DO_EXPORT=""
EXPORT_ARCH=""
EXPORT_DIR="./exports"

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

# 获取镜像在目标注册表中的完整名称
get_target_image_name() {
    local source_name="$1"
    local version="$2"
    
    if [ -z "$REGISTRY" ]; then
        echo "${source_name}:${version}"
        return
    fi
    
    # 检查是否是阿里云ACR格式 (*.aliyuncs.com)
    if echo "$REGISTRY" | grep -q "\.aliyuncs\.com"; then
        # 阿里云ACR格式: registry/namespace/repository:tag
        # 例如: xxx.aliyuncs.com/ai-infra-matrix/ai-infra-matrix:v0.0.3.3
        
        # 从REGISTRY中提取namespace（假设格式为 registry.com/namespace 或直接是 registry.com）
        local registry_host
        local namespace
        
        if echo "$REGISTRY" | grep -q "/"; then
            registry_host=$(echo "$REGISTRY" | cut -d'/' -f1)
            namespace=$(echo "$REGISTRY" | cut -d'/' -f2-)
        else
            registry_host="$REGISTRY"
            namespace="ai-infra-matrix"  # 默认命名空间
        fi
        
        # 对于阿里云ACR，将所有ai-infra组件映射到统一的repository名称
        case "$source_name" in
            ai-infra-*)
                # 所有ai-infra组件使用相同的repository名，通过tag区分
                echo "${registry_host}/${namespace}/ai-infra-matrix:${source_name#ai-infra-}-${version}"
                ;;
            *)
                # 非ai-infra组件保持原名
                echo "${registry_host}/${namespace}/${source_name}:${version}"
                ;;
        esac
    else
        # 其他注册表保持原有逻辑
        echo "${REGISTRY}/${source_name}:${version}"
    fi
}

tag_args() {
    local name="$1"; shift
    local args=("-t" "${name}:$VERSION")
    
    # 添加目标注册表标签
    if [ -n "$REGISTRY" ]; then
        local target_image
        target_image=$(get_target_image_name "$name" "$VERSION")
        args+=("-t" "$target_image")
    fi
    
    # 添加latest标签
    if [ -n "$TAG_LATEST" ]; then
        args+=("-t" "${name}:latest")
        if [ -n "$REGISTRY" ]; then
            local target_latest
            target_latest=$(get_target_image_name "$name" "latest")
            args+=("-t" "$target_latest")
        fi
    fi
    
    printf '%s\n' "${args[@]}"
}

#============================
# 单个组件构建器
#============================

# 生成buildx标签参数
buildx_tag_args() {
    local name="$1"
    local tags=()
    
    # 本地标签
    tags+=("--tag" "${name}:$VERSION")
    if [ -n "$TAG_LATEST" ]; then
        tags+=("--tag" "${name}:latest")
    fi
    
    # 目标注册表标签
    if [ -n "$REGISTRY" ]; then
        local target_image
        target_image=$(get_target_image_name "$name" "$VERSION")
        tags+=("--tag" "$target_image")
        
        if [ -n "$TAG_LATEST" ]; then
            local target_latest
            target_latest=$(get_target_image_name "$name" "latest")
            tags+=("--tag" "$target_latest")
        fi
    fi
    
    printf '%s\n' "${tags[@]}"
}

build_backend() {
    print_info "构建 backend 与 backend-init (VERSION=$VERSION)"
    if [ -n "$USE_BUILDX" ]; then
        local name="ai-infra-backend"
        local tags=()
        readarray -t tags < <(buildx_tag_args "$name")
        
        docker buildx build ${NO_CACHE} \
            --platform "$PLATFORMS" \
            -f src/backend/Dockerfile \
            --build-arg VERSION="$VERSION" \
            "${tags[@]}" \
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
            local target_init
            target_init=$(get_target_image_name "ai-infra-backend-init" "$VERSION")
            docker tag ai-infra-backend:"$VERSION" "$target_init"
        fi
        if [ -n "$TAG_LATEST" ]; then
            docker tag ai-infra-backend:"$VERSION" ai-infra-backend:latest || true
            docker tag ai-infra-backend:"$VERSION" ai-infra-backend-init:latest || true
            if [ -n "$REGISTRY" ]; then
                local target_backend_latest target_init_latest
                target_backend_latest=$(get_target_image_name "ai-infra-backend" "latest")
                target_init_latest=$(get_target_image_name "ai-infra-backend-init" "latest")
                docker tag ai-infra-backend:"$VERSION" "$target_backend_latest" || true
                docker tag ai-infra-backend:"$VERSION" "$target_init_latest" || true
            fi
        fi
    fi
}

build_frontend() {
    print_info "构建 frontend (VERSION=$VERSION)"
    if [ -n "$USE_BUILDX" ]; then
        local name="ai-infra-frontend"
        local tags=()
        readarray -t tags < <(buildx_tag_args "$name")
        
        docker buildx build ${NO_CACHE} \
            --platform "$PLATFORMS" \
            -f src/frontend/Dockerfile \
            --build-arg VERSION="$VERSION" \
            --build-arg REACT_APP_API_URL="${REACT_APP_API_URL:-/api}" \
            --build-arg REACT_APP_JUPYTERHUB_URL="${REACT_APP_JUPYTERHUB_URL:-/jupyter}" \
            "${tags[@]}" \
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
        local name="ai-infra-singleuser"
        local tags=()
        readarray -t tags < <(buildx_tag_args "$name")
        
        docker buildx build ${NO_CACHE} \
            --platform "$PLATFORMS" \
            -f docker/singleuser/Dockerfile \
            --build-arg VERSION="$VERSION" \
            "${tags[@]}" \
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
        local name="ai-infra-jupyterhub"
        local tags=()
        readarray -t tags < <(buildx_tag_args "$name")
        
        docker buildx build ${NO_CACHE} \
            --platform "$PLATFORMS" \
            -f src/jupyterhub/Dockerfile \
            --build-arg VERSION="$VERSION" \
            "${tags[@]}" \
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
        local name="ai-infra-nginx"
        local tags=()
        readarray -t tags < <(buildx_tag_args "$name")
        
        docker buildx build ${NO_CACHE} \
            --platform "$PLATFORMS" \
            -f src/nginx/Dockerfile \
            --build-arg VERSION="$VERSION" \
            --build-arg DEBUG_MODE="${DEBUG_MODE:-false}" \
            --build-arg BUILD_ENV="${BUILD_ENV:-$MODE}" \
            "${tags[@]}" \
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
        local name="ai-infra-gitea"
        local tags=()
        readarray -t tags < <(buildx_tag_args "$name")
        
        docker buildx build ${NO_CACHE} \
            --platform "$PLATFORMS" \
            -f third-party/gitea/Dockerfile \
            --build-arg VERSION="$VERSION" \
            "${tags[@]}" \
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

build_saltstack() {
    print_info "构建 saltstack (VERSION=$VERSION)"
    if [ -n "$USE_BUILDX" ]; then
        local name="ai-infra-saltstack"
        local tags=()
        readarray -t tags < <(buildx_tag_args "$name")
        
        docker buildx build ${NO_CACHE} \
            --platform "$PLATFORMS" \
            -f src/saltstack/Dockerfile \
            --build-arg VERSION="$VERSION" \
            "${tags[@]}" \
            --push \
            src/saltstack
    else
        docker build ${NO_CACHE} \
            -f src/saltstack/Dockerfile \
            --build-arg VERSION="$VERSION" \
            $(tag_args ai-infra-saltstack) \
            src/saltstack
    fi
}

push_image_if_needed() {
    local name="$1"
    if [ -z "$PUSH" ] || [ -z "$REGISTRY" ]; then
        return 0
    fi
    
    local target_image
    target_image=$(get_target_image_name "$name" "$VERSION")
    print_info "推送镜像到 $REGISTRY: $target_image"
    
    if docker push "$target_image"; then
        print_success "推送成功: $target_image"
    else
        print_error "推送失败: $target_image"
        return 1
    fi
    
    if [ -n "$TAG_LATEST" ]; then
        local target_latest
        target_latest=$(get_target_image_name "$name" "latest")
        print_info "推送latest标签: $target_latest"
        if docker push "$target_latest"; then
            print_success "推送latest成功: $target_latest"
        else
            print_warning "推送latest失败: $target_latest"
        fi
    fi
}

push_all_if_needed() {
    for n in ai-infra-backend ai-infra-backend-init ai-infra-frontend ai-infra-singleuser ai-infra-jupyterhub ai-infra-nginx ai-infra-gitea ai-infra-saltstack; do
        push_image_if_needed "$n"
    done
}

#============================
# 镜像拉取功能
#============================

# 拉取单个镜像并重新标记为本地标签
pull_image_from_registry() {
    local name="$1"
    if [ -z "$REGISTRY" ]; then
        print_error "拉取镜像需要指定 --registry 参数"
        return 1
    fi
    
    local target_image
    target_image=$(get_target_image_name "$name" "$VERSION")
    print_info "从注册表拉取镜像: $target_image"
    
    if docker pull "$target_image"; then
        print_success "拉取成功: $target_image"
        
        # 重新标记为本地标签（去掉注册表前缀）
        local local_image="${name}:${VERSION}"
        if docker tag "$target_image" "$local_image"; then
            print_info "重新标记为本地镜像: $local_image"
        else
            print_warning "重新标记失败: $target_image -> $local_image"
        fi
        
        # 如果需要latest标签
        if [ -n "$TAG_LATEST" ]; then
            local target_latest
            target_latest=$(get_target_image_name "$name" "latest")
            print_info "拉取latest标签: $target_latest"
            if docker pull "$target_latest"; then
                docker tag "$target_latest" "${name}:latest" || print_warning "latest标签重新标记失败"
                print_success "拉取latest成功: $target_latest"
            else
                print_warning "拉取latest失败: $target_latest"
            fi
        fi
        
        return 0
    else
        print_error "拉取失败: $target_image"
        return 1
    fi
}

# 拉取所有AI-Infra-Matrix组件镜像
pull_all_images() {
    if [ -z "$REGISTRY" ]; then
        print_error "拉取镜像需要指定 --registry 参数"
        exit 1
    fi
    
    print_info "开始从注册表拉取所有AI-Infra-Matrix镜像"
    print_info "注册表: $REGISTRY"
    print_info "版本: $VERSION"
    echo "================================"
    
    local images=(
        "ai-infra-backend"
        "ai-infra-backend-init"
        "ai-infra-frontend"
        "ai-infra-singleuser"
        "ai-infra-jupyterhub"
        "ai-infra-nginx"
        "ai-infra-gitea"
        "ai-infra-saltstack"
    )
    
    local success_count=0
    local fail_count=0
    local failed_images=()
    
    for img in "${images[@]}"; do
        echo "--------------------"
        if pull_image_from_registry "$img"; then
            success_count=$((success_count + 1))
        else
            fail_count=$((fail_count + 1))
            failed_images+=("$img")
        fi
    done
    
    # 显示拉取结果摘要
    echo ""
    echo "🎉 镜像拉取完成！"
    echo "================================"
    print_success "成功拉取: $success_count 个镜像"
    if [ $fail_count -gt 0 ]; then
        print_error "拉取失败: $fail_count 个镜像"
        echo "失败的镜像:"
        for img in "${failed_images[@]}"; do
            echo "  ❌ $img"
        done
    fi
    
    # 显示本地可用的镜像
    if [ $success_count -gt 0 ]; then
        echo ""
        print_info "本地现在可用的AI-Infra-Matrix镜像:"
        docker images | grep "ai-infra-" | grep "${VERSION}" || true
        
        echo ""
        print_info "现在您可以使用以下命令启动服务:"
        echo "  $0 --up                        # 启动所有服务"
        echo "  docker compose up -d           # 或直接使用compose启动"
    fi
    
    return $fail_count
}

#============================
# 推送依赖镜像到Docker Hub
#============================

# 推送单个依赖镜像到Docker Hub
push_dependency_image() {
    local original_image="$1"
    local target_registry="${2:-docker.io}"
    local namespace="${3:-aresnasa}"
    
    # 解析镜像名称和标签
    local image_name_tag="$original_image"
    local image_name
    local image_tag="latest"
    
    if echo "$original_image" | grep -q ':'; then
        image_name=$(echo "$original_image" | cut -d':' -f1)
        image_tag=$(echo "$original_image" | cut -d':' -f2)
    else
        image_name="$original_image"
    fi
    
    # 去掉可能的仓库前缀，只保留镜像名
    local clean_name
    clean_name=$(echo "$image_name" | sed 's|.*/||')
    
    # 构建目标镜像名
    local target_image="${target_registry}/${namespace}/ai-infra-dep-${clean_name}:${image_tag}"
    
    print_info "推送依赖镜像: $original_image -> $target_image"
    
    # 检查原始镜像是否存在
    if ! docker image inspect "$original_image" >/dev/null 2>&1; then
        print_warning "原始镜像不存在，尝试拉取: $original_image"
        if ! docker pull "$original_image"; then
            print_error "无法拉取镜像: $original_image"
            return 1
        fi
    fi
    
    # 重新标记镜像
    if docker tag "$original_image" "$target_image"; then
        print_info "重新标记成功: $target_image"
    else
        print_error "重新标记失败: $original_image -> $target_image"
        return 1
    fi
    
    # 推送到Docker Hub
    print_info "推送镜像到Docker Hub: $target_image"
    if docker push "$target_image"; then
        print_success "推送成功: $target_image"
        
        # 创建latest标签（如果不是latest）
        if [ "$image_tag" != "latest" ]; then
            local latest_target="${target_registry}/${namespace}/ai-infra-dep-${clean_name}:latest"
            docker tag "$original_image" "$latest_target"
            print_info "推送latest标签: $latest_target"
            docker push "$latest_target" || print_warning "推送latest标签失败: $latest_target"
        fi
        
        return 0
    else
        print_error "推送失败: $target_image"
        return 1
    fi
}

# 推送所有依赖镜像到Docker Hub
push_all_dependencies() {
    local target_registry="${1:-docker.io}"
    local namespace="${2:-aresnasa}"
    local skip_existing="${3:-false}"
    
    print_info "开始推送所有依赖镜像到Docker Hub"
    print_info "目标仓库: $target_registry"
    print_info "命名空间: $namespace"
    echo "================================"
    
    # 检查Docker Hub登录状态
    if ! docker info | grep -q "Username:"; then
        print_warning "未检测到Docker Hub登录状态，请确保已登录"
        print_info "请运行: docker login"
        if [ "$skip_existing" != "force" ]; then
            read -p "是否继续推送？(y/N): " -r
            if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                print_info "取消推送操作"
                return 0
            fi
        fi
    fi
    
    # 收集依赖镜像列表
    print_info "收集依赖镜像列表..."
    collect_compose_images
    
    if [ ${#BASE_IMAGES[@]} -eq 0 ]; then
        print_warning "未找到依赖镜像，请检查docker-compose.yml文件"
        return 1
    fi
    
    print_info "找到 ${#BASE_IMAGES[@]} 个依赖镜像:"
    for img in "${BASE_IMAGES[@]}"; do
        echo "  - $img"
    done
    echo ""
    
    # 统计推送结果
    local success_count=0
    local fail_count=0
    local skipped_count=0
    local failed_images=()
    
    # 逐个推送依赖镜像
    for img in "${BASE_IMAGES[@]}"; do
        echo "--------------------"
        
        # 检查是否跳过已存在的镜像
        if [ "$skip_existing" = "true" ]; then
            local clean_name
            clean_name=$(echo "$img" | sed 's|.*/||' | cut -d':' -f1)
            local check_image="${target_registry}/${namespace}/ai-infra-dep-${clean_name}:latest"
            
            # 简单检查镜像是否可能已存在（通过尝试pull manifest）
            if docker manifest inspect "$check_image" >/dev/null 2>&1; then
                print_info "镜像可能已存在，跳过: $check_image"
                skipped_count=$((skipped_count + 1))
                continue
            fi
        fi
        
        if push_dependency_image "$img" "$target_registry" "$namespace"; then
            success_count=$((success_count + 1))
        else
            fail_count=$((fail_count + 1))
            failed_images+=("$img")
        fi
    done
    
    # 显示推送结果摘要
    echo ""
    echo "🎉 依赖镜像推送完成！"
    echo "================================"
    print_success "成功推送: $success_count 个镜像"
    if [ $skipped_count -gt 0 ]; then
        print_info "跳过镜像: $skipped_count 个镜像"
    fi
    if [ $fail_count -gt 0 ]; then
        print_error "推送失败: $fail_count 个镜像"
        echo "失败的镜像:"
        for img in "${failed_images[@]}"; do
            echo "  ❌ $img"
        done
    fi
    
    # 显示推送的镜像访问信息
    if [ $success_count -gt 0 ]; then
        echo ""
        print_info "推送的镜像可通过以下方式访问:"
        echo "  docker pull ${target_registry}/${namespace}/ai-infra-dep-<镜像名>:latest"
        echo ""
        print_info "示例镜像列表:"
        for img in "${BASE_IMAGES[@]:0:3}"; do
            local clean_name
            clean_name=$(echo "$img" | sed 's|.*/||' | cut -d':' -f1)
            echo "  docker pull ${target_registry}/${namespace}/ai-infra-dep-${clean_name}:latest"
        done
        if [ ${#BASE_IMAGES[@]} -gt 3 ]; then
            echo "  ... 还有 $((${#BASE_IMAGES[@]} - 3)) 个镜像"
        fi
    fi
    
    return $fail_count
}

#============================
# 镜像导出功能
#============================

# 获取所有已构建的ai-infra镜像列表
get_built_images() {
    local version="$1"
    local arch_filter="$2"
    local images=()
    
    # 基础镜像列表（不包括init，因为它只是backend的别名）
    local base_images=(
        "ai-infra-backend"
        "ai-infra-backend-init"
        "ai-infra-frontend"
        "ai-infra-singleuser"
        "ai-infra-jupyterhub"
        "ai-infra-nginx"
        "ai-infra-gitea"
        "ai-infra-saltstack"
    )
    
    for image in "${base_images[@]}"; do
        # 检查镜像是否存在
        if docker images --format "{{.Repository}}:{{.Tag}}" | grep -q "^${image}:${version}$"; then
            # 暂时跳过架构过滤，直接添加所有找到的镜像
            images+=("${image}:${version}")
        fi
    done
    
    printf '%s\n' "${images[@]}"
}

# 导出镜像到tar文件
export_images() {
    local arch="$1"
    local version="$2"
    local export_dir="$3"
    
    print_info "Exporting $arch architecture images (version: $version)"
    
    # 创建导出目录
    if [ ! -d "$export_dir" ]; then
        mkdir -p "$export_dir"
        print_info "Creating export directory: $export_dir"
    fi
    
    # 获取要导出的镜像列表
    local images_list
    images_list=$(get_built_images "$version" "$arch")
    
    if [ -z "$images_list" ]; then
        print_warning "No built images found for $arch architecture (version: $version)"
        return 1
    fi
    
    # 转换为数组
    local images=()
    while IFS= read -r line; do
        [ -n "$line" ] && images+=("$line")
    done <<< "$images_list"
    
    print_info "Found ${#images[@]} images to export:"
    for img in "${images[@]}"; do
        echo "  - $img"
    done
    
    # 生成导出文件名
    local timestamp
    timestamp=$(date +"%Y%m%d_%H%M%S")
    local export_file="${export_dir}/ai-infra-matrix-${version}-${arch}-${timestamp}.tar"
    
    print_info "Export file: $export_file"
    print_info "Starting image export, this may take several minutes..."
    
    # 执行导出
    if docker save "${images[@]}" -o "$export_file"; then
        local file_size
        file_size=$(du -h "$export_file" | cut -f1)
        print_success "Image export successful!"
        print_info "Export file: $export_file"
        print_info "File size: $file_size"
        
        # 生成导入脚本
        local import_script="${export_dir}/import-${version}-${arch}-${timestamp}.sh"
        cat > "$import_script" << EOF
#!/bin/bash
# AI-Infra-Matrix 镜像导入脚本
# 生成时间: $(date)
# 架构: $arch
# 版本: $version

set -e

SCRIPT_DIR=\$(cd "\$(dirname "\$0")" && pwd)
TAR_FILE="\$SCRIPT_DIR/$(basename "$export_file")"

echo "🚀 开始导入 AI-Infra-Matrix 镜像..."
echo "架构: $arch"
echo "版本: $version"
echo "文件: \$TAR_FILE"

if [ ! -f "\$TAR_FILE" ]; then
    echo "❌ 错误: 找不到镜像文件 \$TAR_FILE"
    exit 1
fi

echo "⏳ 正在导入镜像..."
if docker load -i "\$TAR_FILE"; then
    echo "✅ 镜像导入成功!"
    echo ""
    echo "📊 已导入的镜像:"
    docker images | grep "ai-infra-" | grep "$version"
else
    echo "❌ 镜像导入失败!"
    exit 1
fi
EOF
        chmod +x "$import_script"
        print_info "Generated import script: $import_script"
        
        # 生成镜像列表文件
        local manifest_file="${export_dir}/manifest-${version}-${arch}-${timestamp}.txt"
        cat > "$manifest_file" << EOF
# AI-Infra-Matrix 镜像清单
# 生成时间: $(date)
# 架构: $arch
# 版本: $version
# 导出文件: $(basename "$export_file")

EOF
        for img in "${images[@]}"; do
            echo "$img" >> "$manifest_file"
        done
        print_info "Generated image manifest: $manifest_file"
        
    else
        print_error "Image export failed!"
        return 1
    fi
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
    echo "  --pull              - 从指定注册表拉取所有AI-Infra-Matrix镜像（需要 --registry）"
    echo "  --tag-latest        - 额外打 latest 标签"
    echo "  --no-cache          - 无缓存构建"
    echo "  --rebuild           - (仅compose路径) 强制重建所有服务"
    echo "  --multi-arch        - 多架构构建 (linux/amd64,linux/arm64)，需配合 --registry --push 使用"
    echo "  --platforms P       - 指定平台列表 (例如 linux/amd64,linux/arm64)，需配合 --registry --push 使用"
    echo "  --service S         - 只构建指定服务 (backend|frontend|singleuser|jupyterhub|nginx|gitea|saltstack)"
    echo "  --nginx-only        - 只构建nginx服务"
    echo "  --skip-prepull      - 跳过预拉取基础镜像"
    echo "  --update-images     - 强制更新（即使本地存在也重新拉取）"
    echo "  --compose           - 使用 docker-compose build（默认直接 docker build）"
    echo "  --up                - 构建后通过 compose 启动/更新服务 (up -d)"
    echo "  --test              - 构建/启动后运行 scripts/test-health.sh 健康检查"
    echo "  --export-x86        - 导出所有已构建镜像的 x86_64/amd64 版本为 tar 文件"
    echo "  --export-arm64      - 导出所有已构建镜像的 arm64 版本为 tar 文件"
    echo "  --export-dir DIR    - 指定导出目录（默认：./exports）"
    echo "  --push-deps         - 推送所有依赖镜像到Docker Hub"
    echo "  --deps-namespace NS - 指定依赖镜像的命名空间（默认：aresnasa）"
    echo "  --skip-existing-deps - 跳过已存在的依赖镜像"
    echo "  -h, --help          - 显示此帮助信息"
    echo ""
    echo "示例:"
    echo "  $0 dev                          - 开发模式构建（自动版本）"
    echo "  $0 prod --version v0.0.3.3      - 指定版本号构建"
    echo "  $0 prod --service saltstack     - 只构建 saltstack 服务"
    echo "  $0 prod --service backend,frontend  - 只构建 backend 和 frontend 服务"
    echo "  $0 prod --registry localhost:5000 --push --tag-latest  - 构建并推送到本地仓库"
    echo "  $0 prod --registry xxx.aliyuncs.com/ai-infra-matrix --push --version v0.0.3.3  - 推送到阿里云ACR"
    echo "  $0 prod --registry xxx.aliyuncs.com/ai-infra-matrix --pull --version v0.0.3.3  - 从阿里云ACR拉取镜像"
    echo "  $0 prod --export-x86            - 构建并导出所有 x86_64 版本镜像"
    echo "  $0 prod --export-arm64 --export-dir /tmp/images  - 导出 arm64 版本到指定目录"
    echo "  $0 prod --push-deps --deps-namespace myuser  - 推送依赖镜像到Docker Hub myuser命名空间"
}

# 验证服务名称是否有效
validate_services() {
    local services="$1"
    local valid_services="backend frontend singleuser jupyterhub nginx gitea saltstack"
    
    # 使用逗号分割服务列表
    IFS=',' read -ra service_array <<< "$services"
    for service in "${service_array[@]}"; do
        # 去除空格
        service=$(echo "$service" | xargs)
        if [[ ! " $valid_services " =~ " $service " ]]; then
            print_error "无效的服务名称: '$service'"
            print_error "有效的服务: $valid_services"
            exit 1
        fi
    done
}

# 检查是否应该构建指定服务
should_build_service() {
    local service="$1"
    
    # 如果没有指定 SERVICE_ONLY，则构建所有服务（除非是 NGINX_ONLY）
    if [ -z "$SERVICE_ONLY" ]; then
        return 0
    fi
    
    # 检查服务是否在指定列表中
    IFS=',' read -ra service_array <<< "$SERVICE_ONLY"
    for s in "${service_array[@]}"; do
        s=$(echo "$s" | xargs)  # 去除空格
        if [ "$s" = "$service" ]; then
            return 0
        fi
    done
    return 1
}

# 其他默认参数
REBUILD=""
NGINX_ONLY=""
SERVICE_ONLY=""
SKIP_PREPULL=""
UPDATE_IMAGES=""
PUSH_DEPS=""
DEPS_NAMESPACE="aresnasa"
SKIP_EXISTING_DEPS=""

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
        --pull)
            PULL="true"; shift ;;
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
        --service)
            SERVICE_ONLY="$2"; shift 2 ;;
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
        --export-x86)
            DO_EXPORT="true"
            EXPORT_ARCH="amd64"
            shift
            ;;
        --export-arm64)
            DO_EXPORT="true"
            EXPORT_ARCH="arm64"
            shift
            ;;
        --export-dir)
            EXPORT_DIR="$2"; shift 2 ;;
        --push-deps)
            PUSH_DEPS="true"
            shift
            ;;
        --deps-namespace)
            DEPS_NAMESPACE="$2"; shift 2 ;;
        --skip-existing-deps)
            SKIP_EXISTING_DEPS="true"
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

# 验证服务参数
if [ -n "$SERVICE_ONLY" ]; then
    validate_services "$SERVICE_ONLY"
    print_info "只构建指定服务: $SERVICE_ONLY"
elif [ -n "$NGINX_ONLY" ]; then
    print_info "只构建 nginx 服务"
fi

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

# 如果是拉取模式，直接执行拉取操作并退出
if [ -n "$PULL" ]; then
    echo ""
    echo "🔽 AI-Infra-Matrix 镜像拉取模式"
    echo "================================"
    print_info "拉取模式: 从注册表拉取镜像"
    print_info "注册表: ${REGISTRY:-未指定}"
    print_info "镜像版本: ${VERSION}"
    print_info "拉取时间: $(date)"
    echo ""
    
    if pull_all_images; then
        echo ""
        print_success "🎉 镜像拉取完成！"
        print_info "现在您可以使用拉取的镜像启动服务"
        exit 0
    else
        print_error "❌ 镜像拉取失败！"
        exit 1
    fi
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
    elif [ -n "$SERVICE_ONLY" ]; then
        print_info "仅构建指定服务 (compose): $SERVICE_ONLY"
        # 将逗号分隔的服务转换为空格分隔
        SERVICES=$(echo "$SERVICE_ONLY" | tr ',' ' ')
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
    should_build_service "backend" && [ -z "$NGINX_ONLY" ] && build_backend
    should_build_service "frontend" && [ -z "$NGINX_ONLY" ] && build_frontend
    should_build_service "singleuser" && [ -z "$NGINX_ONLY" ] && build_singleuser
    should_build_service "jupyterhub" && [ -z "$NGINX_ONLY" ] && build_jupyterhub
    should_build_service "gitea" && [ -z "$NGINX_ONLY" ] && build_gitea
    should_build_service "saltstack" && [ -z "$NGINX_ONLY" ] && build_saltstack
    should_build_service "nginx" && build_nginx
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

# 执行镜像导出（如果需要）
if [ -n "$DO_EXPORT" ]; then
    echo ""
    print_info "Starting image export..."
    if export_images "$EXPORT_ARCH" "$VERSION" "$EXPORT_DIR"; then
        print_success "Image export completed!"
        echo ""
        print_info "Export directory: $EXPORT_DIR"
        print_info "Use the generated import script to import images on other machines"
    else
        print_error "Image export failed!"
        exit 1
    fi
fi

# 执行依赖镜像推送（如果需要）
if [ -n "$PUSH_DEPS" ]; then
    echo ""
    print_info "Starting dependency images push to Docker Hub..."
    
    # 设置跳过已存在镜像的选项
    skip_mode=""
    if [ -n "$SKIP_EXISTING_DEPS" ]; then
        skip_mode="true"
    fi
    
    if push_all_dependencies "docker.io" "$DEPS_NAMESPACE" "$skip_mode"; then
        print_success "Dependency images push completed!"
        echo ""
        print_info "All dependency images are now available on Docker Hub"
        print_info "Namespace: $DEPS_NAMESPACE"
        print_info "You can now pull them using: docker pull docker.io/$DEPS_NAMESPACE/ai-infra-dep-<image-name>:latest"
    else
        exit_code=$?
        print_error "Some dependency images failed to push!"
        print_warning "Check the output above for failed images"
        print_info "You can retry with --skip-existing-deps to skip already pushed images"
        exit $exit_code
    fi
fi
