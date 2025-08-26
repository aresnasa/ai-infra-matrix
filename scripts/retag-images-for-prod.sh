#!/bin/bash

# AI-Infra镜像重新标记脚本
# 用于将本地镜像标记为aiharbor.msxf.local/aihpc格式，方便生产环境测试

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
TARGET_REGISTRY="aiharbor.msxf.local"
TARGET_NAMESPACE="aihpc"
DEFAULT_TAG="v0.3.5"

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 获取当前ai-infra相关镜像
get_ai_infra_images() {
    docker images --format "table {{.Repository}}:{{.Tag}}" | grep -E "^ai-infra-" | grep -v "<none>"
}

# 重新标记单个镜像
retag_image() {
    local source_image="$1"
    local image_name=$(echo "$source_image" | cut -d':' -f1 | sed 's/^ai-infra-//')
    local current_tag=$(echo "$source_image" | cut -d':' -f2)
    
    # 如果没有标签，使用默认标签
    if [[ "$current_tag" == "$source_image" ]]; then
        current_tag="$DEFAULT_TAG"
    fi
    
    local target_image="${TARGET_REGISTRY}/${TARGET_NAMESPACE}/ai-infra-${image_name}:${current_tag}"
    
    log_info "重新标记: $source_image -> $target_image"
    
    if docker tag "$source_image" "$target_image"; then
        log_success "标记成功: $target_image"
        return 0
    else
        log_error "标记失败: $source_image"
        return 1
    fi
}

# 重新标记第三方依赖镜像
retag_deps_images() {
    log_info "重新标记第三方依赖镜像..."
    
    # 第三方镜像映射（公有基础镜像 -> 内网aiharbor.msxf.local/aihpc）
    local deps_images=(
        # 公有基础镜像映射到内网
        "postgres:15-alpine:aiharbor.msxf.local/aihpc/postgres:v0.3.5"
        "postgres:15:aiharbor.msxf.local/aihpc/postgres:v0.3.5"
        "postgres:latest:aiharbor.msxf.local/aihpc/postgres:v0.3.5"
        "redis:7-alpine:aiharbor.msxf.local/aihpc/redis:v0.3.5"
        "redis:7:aiharbor.msxf.local/aihpc/redis:v0.3.5"
        "redis:latest:aiharbor.msxf.local/aihpc/redis:v0.3.5"
        "nginx:1.27-alpine:aiharbor.msxf.local/aihpc/nginx:v0.3.5"
        "nginx:alpine:aiharbor.msxf.local/aihpc/nginx:v0.3.5"
        "nginx:latest:aiharbor.msxf.local/aihpc/nginx:v0.3.5"
        "quay.io/minio/minio:latest:aiharbor.msxf.local/aihpc/minio:v0.3.5"
        "minio/minio:latest:aiharbor.msxf.local/aihpc/minio:v0.3.5"
        "minio/minio:RELEASE-2024-08-03T04-33-23Z:aiharbor.msxf.local/aihpc/minio:v0.3.5"
        "tecnativa/tcp-proxy:latest:aiharbor.msxf.local/aihpc/tcp-proxy:v0.3.5"
        "redislabs/redisinsight:latest:aiharbor.msxf.local/aihpc/redisinsight:v0.3.5"
        "redislabs/redisinsight:2.54:aiharbor.msxf.local/aihpc/redisinsight:v0.3.5"
        # 已有内网镜像重新标记
        "registry.example.com/ai-infra/ai-infra-deps-postgres:v1.0.0:aiharbor.msxf.local/aihpc/postgres:v0.3.5"
        "registry.example.com/ai-infra/ai-infra-deps-redis:v1.0.0:aiharbor.msxf.local/aihpc/redis:v0.3.5"
        "registry.example.com/ai-infra/ai-infra-deps-nginx:v1.0.0:aiharbor.msxf.local/aihpc/nginx:v0.3.5"
        "registry.example.com/ai-infra/ai-infra-deps-minio:v1.0.0:aiharbor.msxf.local/aihpc/minio:v0.3.5"
        "registry.example.com/ai-infra/ai-infra-deps-tcp-proxy:v1.0.0:aiharbor.msxf.local/aihpc/tcp-proxy:v0.3.5"
        "registry.example.com/ai-infra/ai-infra-deps-redisinsight:v1.0.0:aiharbor.msxf.local/aihpc/redisinsight:v0.3.5"
        "registry.local/ai-infra/ai-infra-deps-postgres:latest:aiharbor.msxf.local/aihpc/postgres:v0.3.5"
        "registry.local/ai-infra/ai-infra-deps-redis:latest:aiharbor.msxf.local/aihpc/redis:v0.3.5"
        "registry.local/ai-infra/ai-infra-deps-nginx:latest:aiharbor.msxf.local/aihpc/nginx:v0.3.5"
        "registry.local/ai-infra/ai-infra-deps-minio:latest:aiharbor.msxf.local/aihpc/minio:v0.3.5"
        "registry.local/ai-infra/ai-infra-deps-tcp-proxy:latest:aiharbor.msxf.local/aihpc/tcp-proxy:v0.3.5"
        "registry.local/ai-infra/ai-infra-deps-redisinsight:latest:aiharbor.msxf.local/aihpc/redisinsight:v0.3.5"
        # 现有 aiharbor 镜像库镜像保持不变（避免重复标记）
        "aiharbor.msxf.local/library/postgres:v0.3.5:aiharbor.msxf.local/aihpc/postgres:v0.3.5"
        "aiharbor.msxf.local/library/redis:v0.3.5:aiharbor.msxf.local/aihpc/redis:v0.3.5"
        "aiharbor.msxf.local/library/nginx:v0.3.5:aiharbor.msxf.local/aihpc/nginx:v0.3.5"
        "aiharbor.msxf.local/minio/minio:v0.3.5:aiharbor.msxf.local/aihpc/minio:v0.3.5"
    )
    
    for mapping in "${deps_images[@]}"; do
        local source_image=$(echo "$mapping" | cut -d':' -f1-2)
        local target_image=$(echo "$mapping" | cut -d':' -f3-4)
        
        if docker images --format "{{.Repository}}:{{.Tag}}" | grep -q "^${source_image}$"; then
            log_info "重新标记依赖镜像: $source_image -> $target_image"
            if docker tag "$source_image" "$target_image"; then
                log_success "依赖镜像标记成功: $target_image"
            else
                log_warning "依赖镜像标记失败: $source_image (可能镜像不存在)"
            fi
        else
            log_warning "依赖镜像不存在，跳过: $source_image"
        fi
    done
}

# 显示帮助信息
show_help() {
    cat << EOF
AI-Infra镜像重新标记脚本

用法: $0 [选项]

选项:
    -h, --help          显示此帮助信息
    -t, --tag TAG       指定标签 (默认: $DEFAULT_TAG)
    -r, --registry REG  指定目标仓库 (默认: $TARGET_REGISTRY)
    -n, --namespace NS  指定命名空间 (默认: $TARGET_NAMESPACE)
    --dry-run           仅显示将要执行的操作，不实际执行
    --deps              同时标记第三方依赖镜像
    --list              列出当前的ai-infra镜像

示例:
    $0                                  # 使用默认设置重新标记
    $0 --tag v0.3.6                   # 使用指定标签
    $0 --deps                          # 同时标记依赖镜像
    $0 --dry-run                       # 预览操作
    $0 --list                          # 列出镜像

EOF
}

# 列出当前镜像
list_images() {
    log_info "当前ai-infra相关镜像:"
    echo
    docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}" | head -1
    docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}\t{{.CreatedAt}}" | grep -E "ai-infra-|aiharbor.msxf.local" | sort
    echo
}

# 主函数
main() {
    local dry_run=false
    local include_deps=false
    local list_only=false
    
    # 解析命令行参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            -h|--help)
                show_help
                exit 0
                ;;
            -t|--tag)
                DEFAULT_TAG="$2"
                shift 2
                ;;
            -r|--registry)
                TARGET_REGISTRY="$2"
                shift 2
                ;;
            -n|--namespace)
                TARGET_NAMESPACE="$2"
                shift 2
                ;;
            --dry-run)
                dry_run=true
                shift
                ;;
            --deps)
                include_deps=true
                shift
                ;;
            --list)
                list_only=true
                shift
                ;;
            *)
                log_error "未知选项: $1"
                show_help
                exit 1
                ;;
        esac
    done
    
    if [[ "$list_only" == "true" ]]; then
        list_images
        exit 0
    fi
    
    log_info "AI-Infra镜像重新标记脚本"
    log_info "目标仓库: $TARGET_REGISTRY"
    log_info "目标命名空间: $TARGET_NAMESPACE"
    log_info "默认标签: $DEFAULT_TAG"
    
    if [[ "$dry_run" == "true" ]]; then
        log_warning "DRY RUN模式 - 仅显示操作，不实际执行"
    fi
    
    echo
    
    # 获取当前镜像列表
    local images=($(get_ai_infra_images))
    
    if [[ ${#images[@]} -eq 0 ]]; then
        log_warning "未找到ai-infra相关镜像"
        exit 0
    fi
    
    log_info "找到 ${#images[@]} 个ai-infra镜像"
    
    local success_count=0
    local total_count=${#images[@]}
    
    # 重新标记每个镜像
    for image in "${images[@]}"; do
        if [[ "$dry_run" == "true" ]]; then
            local image_name=$(echo "$image" | cut -d':' -f1 | sed 's/^ai-infra-//')
            local current_tag=$(echo "$image" | cut -d':' -f2)
            if [[ "$current_tag" == "$image" ]]; then
                current_tag="$DEFAULT_TAG"
            fi
            local target_image="${TARGET_REGISTRY}/${TARGET_NAMESPACE}/ai-infra-${image_name}:${current_tag}"
            log_info "[DRY RUN] 将标记: $image -> $target_image"
        else
            if retag_image "$image"; then
                ((success_count++))
            fi
        fi
    done
    
    # 处理依赖镜像
    if [[ "$include_deps" == "true" ]]; then
        if [[ "$dry_run" == "true" ]]; then
            log_info "[DRY RUN] 将重新标记第三方依赖镜像"
        else
            retag_deps_images
        fi
    fi
    
    echo
    
    if [[ "$dry_run" == "false" ]]; then
        log_success "重新标记完成: $success_count/$total_count 成功"
        
        if [[ $success_count -lt $total_count ]]; then
            log_warning "部分镜像标记失败，请检查错误信息"
            exit 1
        fi
        
        log_info "查看重新标记后的镜像:"
        docker images | grep "$TARGET_REGISTRY" | head -10
        
        if [[ "$include_deps" == "true" ]]; then
            echo
            log_info "提示: 包含了依赖镜像的标记"
        fi
        
        echo
        log_success "所有镜像已成功重新标记为 $TARGET_REGISTRY/$TARGET_NAMESPACE 格式"
        log_info "现在可以使用生产环境配置进行部署测试了"
    else
        log_info "DRY RUN完成 - 使用 --help 查看更多选项"
    fi
}

# 检查Docker是否运行
if ! docker info >/dev/null 2>&1; then
    log_error "Docker未运行或无法访问"
    exit 1
fi

# 执行主函数
main "$@"
