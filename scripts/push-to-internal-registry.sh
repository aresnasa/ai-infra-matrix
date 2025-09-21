#!/bin/bash
# AI Infrastructure Matrix - 内部仓库推送脚本
# 专门用于推送所有依赖镜像到内部Harbor仓库

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() { echo -e "${BLUE}ℹ️  $1${NC}"; }
print_success() { echo -e "${GREEN}✅ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠️  $1${NC}"; }
print_error() { echo -e "${RED}❌ $1${NC}"; }

# 配置参数
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
VERSION="${VERSION:-v0.3.6-dev}"
INTERNAL_REGISTRY="${INTERNAL_REGISTRY:-aiharbor.msxf.local/aihpc}"
DRY_RUN="${DRY_RUN:-false}"
PUSH_SELF_BUILT="${PUSH_SELF_BUILT:-true}"
SKIP_EXISTING="${SKIP_EXISTING:-false}"

print_info "🚀 AI Infrastructure Matrix 内部仓库推送工具"
print_info "=========================================="
print_info "项目版本: $VERSION"
print_info "内部仓库: $INTERNAL_REGISTRY"
print_info "包含自建镜像: $PUSH_SELF_BUILT"
print_info "跳过已存在: $SKIP_EXISTING"
if [[ "$DRY_RUN" == "true" ]]; then
    print_warning "🧪 DRY RUN 模式 - 不会实际推送镜像"
fi
echo

# 从docker-compose.yml提取依赖镜像
extract_dependency_images() {
    print_info "📋 从 docker-compose.yml 提取依赖镜像..."
    
    local compose_file="$PROJECT_ROOT/docker-compose.yml"
    if [[ ! -f "$compose_file" ]]; then
        print_error "找不到 docker-compose.yml 文件: $compose_file"
        return 1
    fi
    
    # 提取所有非自建镜像
    local images=($(grep -E '^\s*image:\s*' "$compose_file" | \
                   sed -E 's/^\s*image:\s*//' | \
                   sed 's/${[^}]*}/'"$VERSION"'/g' | \
                   tr -d '"' | tr -d "'" | \
                   grep -v '^ai-infra-' | \
                   sort -u))
    
    printf '%s\n' "${images[@]}"
}

# 获取自建镜像列表
get_self_built_images() {
    print_info "🏗️ 获取自建镜像列表..."
    
    local custom_images=(
        "ai-infra-backend:$VERSION"
        "ai-infra-backend-init:$VERSION"  
        "ai-infra-frontend:$VERSION"
        "ai-infra-jupyterhub:$VERSION"
        "ai-infra-nginx:$VERSION"
        "ai-infra-singleuser:$VERSION"
        "ai-infra-gitea:$VERSION"
        "ai-infra-saltstack:$VERSION"
    )
    
    printf '%s\n' "${custom_images[@]}"
}

# 检查镜像是否存在于本地
check_local_image_exists() {
    local image="$1"
    docker image inspect "$image" >/dev/null 2>&1
}

# 检查镜像是否存在于远程仓库
check_remote_image_exists() {
    local image="$1"
    # 尝试拉取 manifest 来检查镜像是否存在
    docker manifest inspect "$image" >/dev/null 2>&1
}

# 将镜像名映射到内部仓库格式
map_to_internal_registry() {
    local original_image="$1"
    local target_version="$2"
    
    # 解析原始镜像名
    local image_name=""
    local image_tag=""
    
    if [[ "$original_image" == *":"* ]]; then
        image_name="${original_image%%:*}"
        image_tag="${original_image##*:}"
    else
        image_name="$original_image"
        image_tag="latest"
    fi
    
    # 获取简单名称（去掉可能的namespace）
    local simple_name=""
    if [[ "$image_name" == *"/"* ]]; then
        simple_name="${image_name##*/}"
    else
        simple_name="$image_name"
    fi
    
    # 对于依赖镜像，映射到内部仓库格式
    if [[ "$original_image" == ai-infra-* ]]; then
        # 自建镜像直接使用原始名称和版本
        echo "$INTERNAL_REGISTRY/$original_image"
    else
        # 第三方依赖镜像使用统一版本标签
        echo "$INTERNAL_REGISTRY/$simple_name:$target_version"
    fi
}

# 拉取并推送单个镜像
pull_and_push_image() {
    local original_image="$1"
    local target_image="$2"
    local is_dependency="${3:-true}"
    
    print_info "处理镜像: $original_image"
    
    # 检查跳过已存在的镜像
    if [[ "$SKIP_EXISTING" == "true" ]] && check_remote_image_exists "$target_image"; then
        print_success "  ⏭️ 镜像已存在，跳过: $target_image"
        return 0
    fi
    
    # 检查本地是否有原始镜像
    if ! check_local_image_exists "$original_image"; then
        if [[ "$is_dependency" == "true" ]]; then
            print_info "  ⬇️ 拉取依赖镜像: $original_image"
            if ! docker pull "$original_image"; then
                print_error "  ❌ 拉取失败: $original_image"
                return 1
            fi
        else
            print_error "  ❌ 自建镜像不存在，请先构建: $original_image"
            return 1
        fi
    fi
    
    # 标记镜像
    print_info "  🏷️ 标记镜像: $target_image"
    if ! docker tag "$original_image" "$target_image"; then
        print_error "  ❌ 标记失败: $original_image -> $target_image"
        return 1
    fi
    
    # 推送镜像
    if [[ "$DRY_RUN" == "true" ]]; then
        print_warning "  🧪 DRY RUN: 将推送 $target_image"
        return 0
    fi
    
    print_info "  ⬆️ 推送镜像: $target_image"
    if docker push "$target_image"; then
        print_success "  ✅ 推送成功: $target_image"
        return 0
    else
        print_error "  ❌ 推送失败: $target_image"
        return 1
    fi
}

# 推送依赖镜像
push_dependency_images() {
    local images=()
    local success_count=0
    local fail_count=0
    local failed_images=()
    
    # 收集依赖镜像
    while IFS= read -r image; do
        [[ -n "$image" ]] && images+=("$image")
    done < <(extract_dependency_images)
    
    print_info "🔄 推送第三方依赖镜像 (${#images[@]} 个)..."
    echo
    
    for original_image in "${images[@]}"; do
        local target_image
        target_image=$(map_to_internal_registry "$original_image" "$VERSION")
        
        if pull_and_push_image "$original_image" "$target_image" "true"; then
            ((success_count++))
        else
            ((fail_count++))
            failed_images+=("$original_image")
        fi
        echo
    done
    
    print_info "依赖镜像推送结果: $success_count 成功, $fail_count 失败"
    return $fail_count
}

# 推送自建镜像
push_self_built_images() {
    local images=()
    local success_count=0
    local fail_count=0
    local failed_images=()
    
    # 收集自建镜像
    while IFS= read -r image; do
        [[ -n "$image" ]] && images+=("$image")
    done < <(get_self_built_images)
    
    print_info "🏗️ 推送自建镜像 (${#images[@]} 个)..."
    echo
    
    for original_image in "${images[@]}"; do
        local target_image
        target_image=$(map_to_internal_registry "$original_image" "$VERSION")
        
        if pull_and_push_image "$original_image" "$target_image" "false"; then
            ((success_count++))
        else
            ((fail_count++))
            failed_images+=("$original_image")
        fi
        echo
    done
    
    print_info "自建镜像推送结果: $success_count 成功, $fail_count 失败"
    return $fail_count
}

# 生成离线部署docker-compose文件
generate_internal_compose() {
    print_info "📝 生成使用内部仓库的 docker-compose 文件..."
    
    local internal_compose="$PROJECT_ROOT/docker-compose-internal.yml"
    local original_compose="$PROJECT_ROOT/docker-compose.yml"
    
    # 复制原始compose文件并替换镜像地址
    sed "s|image: \\([^a][^i].*\\)|image: $INTERNAL_REGISTRY/\\1|g" "$original_compose" | \
    sed "s|image: ai-infra-|image: $INTERNAL_REGISTRY/ai-infra-|g" > "$internal_compose"
    
    print_success "内部仓库compose文件已生成: $internal_compose"
    
    # 生成说明文件
    cat > "$PROJECT_ROOT/INTERNAL-REGISTRY-USAGE.md" << EOF
# 内部仓库使用指南

## 概述
所有镜像已推送到内部Harbor仓库: \`$INTERNAL_REGISTRY\`

## 使用方法

### 1. 使用内部仓库compose文件
\`\`\`bash
docker-compose -f docker-compose-internal.yml up -d
\`\`\`

### 2. 手动设置镜像仓库
\`\`\`bash
export REGISTRY_PREFIX="$INTERNAL_REGISTRY/"
docker-compose up -d
\`\`\`

## 推送的镜像列表

### 第三方依赖镜像
EOF
    
    # 添加依赖镜像列表
    while IFS= read -r image; do
        local target_image
        target_image=$(map_to_internal_registry "$image" "$VERSION")
        echo "- \`$image\` → \`$target_image\`" >> "$PROJECT_ROOT/INTERNAL-REGISTRY-USAGE.md"
    done < <(extract_dependency_images)
    
    cat >> "$PROJECT_ROOT/INTERNAL-REGISTRY-USAGE.md" << EOF

### 自建镜像
EOF
    
    # 添加自建镜像列表
    if [[ "$PUSH_SELF_BUILT" == "true" ]]; then
        while IFS= read -r image; do
            local target_image
            target_image=$(map_to_internal_registry "$image" "$VERSION")
            echo "- \`$image\` → \`$target_image\`" >> "$PROJECT_ROOT/INTERNAL-REGISTRY-USAGE.md"
        done < <(get_self_built_images)
    fi
    
    cat >> "$PROJECT_ROOT/INTERNAL-REGISTRY-USAGE.md" << EOF

## 镜像拉取验证
\`\`\`bash
# 验证依赖镜像
docker pull $INTERNAL_REGISTRY/postgres:$VERSION
docker pull $INTERNAL_REGISTRY/redis:$VERSION
docker pull $INTERNAL_REGISTRY/cp-kafka:$VERSION

# 验证自建镜像（如果已推送）
docker pull $INTERNAL_REGISTRY/ai-infra-backend:$VERSION
docker pull $INTERNAL_REGISTRY/ai-infra-frontend:$VERSION
\`\`\`

## 故障排除
1. 确保已登录内部Harbor仓库
2. 检查网络连接到内部仓库
3. 验证镜像标签是否正确

生成时间: $(date)
版本: $VERSION
EOF
    
    print_success "内部仓库使用指南已生成: $PROJECT_ROOT/INTERNAL-REGISTRY-USAGE.md"
}

# 显示帮助信息
show_help() {
    echo "AI Infrastructure Matrix 内部仓库推送工具"
    echo ""
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  --registry REGISTRY     内部仓库地址 (默认: aiharbor.msxf.local/aihpc)"
    echo "  --version VERSION       项目版本 (默认: v0.3.6-dev)"
    echo "  --no-self-built        不推送自建镜像"
    echo "  --skip-existing        跳过已存在的镜像"
    echo "  --dry-run              只显示将要执行的操作，不实际推送"
    echo "  --help, -h             显示此帮助信息"
    echo ""
    echo "环境变量:"
    echo "  VERSION                项目版本"
    echo "  INTERNAL_REGISTRY      内部仓库地址"
    echo "  DRY_RUN               干运行模式"
    echo "  PUSH_SELF_BUILT       是否推送自建镜像"
    echo "  SKIP_EXISTING         跳过已存在镜像"
    echo ""
    echo "示例:"
    echo "  $0                                          # 使用默认配置"
    echo "  $0 --registry hub.company.com/ai-infra     # 指定内部仓库"
    echo "  $0 --version v1.0.0 --skip-existing        # 指定版本并跳过已存在"
    echo "  $0 --dry-run                               # 干运行模式"
}

# 主函数
main() {
    print_info "开始推送镜像到内部仓库..."
    echo
    
    local dependency_fail=0
    local self_built_fail=0
    
    # 推送依赖镜像
    if ! push_dependency_images; then
        dependency_fail=$?
    fi
    
    # 推送自建镜像（如果启用）
    if [[ "$PUSH_SELF_BUILT" == "true" ]]; then
        if ! push_self_built_images; then
            self_built_fail=$?
        fi
    fi
    
    # 生成配置文件
    if [[ "$DRY_RUN" != "true" ]]; then
        generate_internal_compose
    fi
    
    # 总结
    echo
    print_info "=========================================="
    
    if [[ $dependency_fail -eq 0 && $self_built_fail -eq 0 ]]; then
        print_success "🎉 所有镜像推送成功!"
    else
        print_error "❌ 部分镜像推送失败:"
        [[ $dependency_fail -gt 0 ]] && print_error "  - 依赖镜像: $dependency_fail 个失败"
        [[ $self_built_fail -gt 0 ]] && print_error "  - 自建镜像: $self_built_fail 个失败"
    fi
    
    if [[ "$DRY_RUN" != "true" ]]; then
        print_info "📖 请查看 INTERNAL-REGISTRY-USAGE.md 了解使用方法"
    fi
    
    return $((dependency_fail + self_built_fail))
}

# 命令行参数处理
while [[ $# -gt 0 ]]; do
    case $1 in
        --registry)
            INTERNAL_REGISTRY="$2"
            shift 2
            ;;
        --version)
            VERSION="$2"
            shift 2
            ;;
        --no-self-built)
            PUSH_SELF_BUILT="false"
            shift
            ;;
        --skip-existing)
            SKIP_EXISTING="true"
            shift
            ;;
        --dry-run)
            DRY_RUN="true"
            shift
            ;;
        --help|-h)
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

# 检查必需工具
if ! command -v docker >/dev/null 2>&1; then
    print_error "Docker 未安装或不在 PATH 中"
    exit 1
fi

# 检查Docker登录状态
if [[ "$DRY_RUN" != "true" ]]; then
    registry_host=$(echo "$INTERNAL_REGISTRY" | cut -d'/' -f1)
    if ! docker info 2>/dev/null | grep -q "Registry:" || ! docker login "$registry_host" --password-stdin <<< "" 2>/dev/null; then
        print_warning "⚠️  未检测到 Docker 仓库登录状态"
        print_info "请先登录内部仓库: docker login $registry_host"
        read -p "是否继续? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            print_info "已取消操作"
            exit 0
        fi
    fi
fi

# 执行主函数
main