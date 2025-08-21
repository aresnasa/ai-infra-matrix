#!/bin/bash

# AI-Infra-Matrix 全栈镜像管理脚本
# 支持私有仓库镜像管理、自动标签和推送

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

# 全局变量
PRIVATE_REGISTRY=""
IMAGE_TAG="v0.3.5"
DOCKER_COMPOSE_FILE="docker-compose.yml"
DOCKER_COMPOSE_BACKUP="docker-compose.yml.backup"

# 获取镜像映射名称的函数
get_image_mapping() {
    local original_image="$1"
    
    case "$original_image" in
        "postgres:15-alpine")
            echo "postgres:15-alpine"
            ;;
        "redis:7-alpine")
            echo "redis:7-alpine"
            ;;
        "osixia/openldap:stable")
            echo "openldap:stable"
            ;;
        "osixia/phpldapadmin:stable")
            echo "phpldapadmin:stable"
            ;;
        "nginx:1.27-alpine")
            echo "nginx:1.27-alpine"
            ;;
        "quay.io/minio/minio:latest")
            echo "minio:latest"
            ;;
        "redislabs/redisinsight:latest")
            echo "redisinsight:latest"
            ;;
        "tecnativa/tcp-proxy")
            echo "tcp-proxy:latest"
            ;;
        "ai-infra-backend-init")
            echo "ai-infra-backend-init"
            ;;
        "ai-infra-backend")
            echo "ai-infra-backend"
            ;;
        "ai-infra-frontend")
            echo "ai-infra-frontend"
            ;;
        "ai-infra-jupyterhub")
            echo "ai-infra-jupyterhub"
            ;;
        "ai-infra-singleuser")
            echo "ai-infra-singleuser"
            ;;
        "ai-infra-saltstack")
            echo "ai-infra-saltstack"
            ;;
        "ai-infra-nginx")
            echo "ai-infra-nginx"
            ;;
        "ai-infra-gitea")
            echo "ai-infra-gitea"
            ;;
        *)
            # 默认处理：移除前缀域名
            echo "${original_image##*/}"
            ;;
    esac
}

# 解析镜像名称和标签
parse_image() {
    local image="$1"
    local name tag
    
    if [[ "$image" == *":"* ]]; then
        name="${image%:*}"
        tag="${image#*:}"
    else
        name="$image"
        tag="latest"
    fi
    
    echo "$name:$tag"
}

# 生成私有仓库镜像名称
get_private_image_name() {
    local original_image="$1"
    local registry="$2"
    
    # 移除 registry 末尾的斜杠（如果有）
    registry="${registry%/}"
    
    # 解析原始镜像
    local name tag
    if [[ "$original_image" == *":"* ]]; then
        name="${original_image%:*}"
        tag="${original_image#*:}"
    else
        name="$original_image"
        tag="latest"
    fi
    
    # 处理变量替换
    if [[ "$tag" == *"\${IMAGE_TAG"* ]]; then
        tag="${IMAGE_TAG}"
    fi
    
    # 获取映射名称
    local mapped_name=$(get_image_mapping "$name")
    if [[ -z "$mapped_name" ]]; then
        mapped_name=$(get_image_mapping "$original_image")
    fi
    
    if [[ -z "$mapped_name" ]]; then
        # 默认处理：移除前缀域名
        mapped_name="${name##*/}"
    fi
    
    # 如果映射名称包含标签，使用映射的标签
    if [[ "$mapped_name" == *":"* ]]; then
        echo "${registry}/${mapped_name}"
    else
        echo "${registry}/${mapped_name}:${tag}"
    fi
}

# 从 docker-compose.yml 提取所有镜像
extract_images_from_compose() {
    local compose_file="$1"
    
    # 使用更精确的方式提取镜像
    grep -E "^\s*image:\s*" "$compose_file" | \
    sed -E 's/^\s*image:\s*//g' | \
    sed -E 's/["'\''"]//g' | \
    sort -u
}

# 备份 docker-compose.yml 文件
backup_compose_file() {
    local compose_file="$1"
    local backup_file="$2"
    
    if [[ -f "$compose_file" ]]; then
        print_info "备份 docker-compose.yml -> ${backup_file}"
        cp "$compose_file" "$backup_file"
        print_success "备份完成: $backup_file"
    else
        print_error "找不到 docker-compose.yml 文件"
        exit 1
    fi
}

# 恢复 docker-compose.yml 文件
restore_compose_file() {
    local compose_file="$1"
    local backup_file="$2"
    
    if [[ -f "$backup_file" ]]; then
        print_info "恢复 docker-compose.yml <- ${backup_file}"
        cp "$backup_file" "$compose_file"
        print_success "恢复完成: $compose_file"
    else
        print_warning "找不到备份文件: $backup_file"
    fi
}

# 直接修改 docker-compose.yml 文件中的镜像
modify_compose_images() {
    local registry="$1"
    local tag="$2"
    local compose_file="$3"
    
    print_info "直接修改 docker-compose.yml 中的镜像..."
    print_info "私有仓库: $registry"
    print_info "镜像标签: $tag"
    
    # 创建临时文件
    local temp_file="${compose_file}.tmp"
    
    # 逐行处理文件
    while IFS= read -r line || [[ -n "$line" ]]; do
        if [[ "$line" =~ ^[[:space:]]*image:[[:space:]]*(.+)$ ]]; then
            local original_image="${BASH_REMATCH[1]}"
            # 移除引号
            original_image="${original_image//\"/}"
            original_image="${original_image//\'/}"
            
            # 替换环境变量
            if [[ "$original_image" == *"\${IMAGE_TAG"* ]]; then
                original_image="${original_image//\$\{IMAGE_TAG:-v0.0.3.3\}/$tag}"
                original_image="${original_image//\$\{IMAGE_TAG\}/$tag}"
            fi
            
            local private_image=$(get_private_image_name "$original_image" "$registry")
            
            # 保持原有的缩进
            local indent=""
            if [[ "$line" =~ ^([[:space:]]*) ]]; then
                indent="${BASH_REMATCH[1]}"
            fi
            
            echo "${indent}image: $private_image"
            print_info "替换镜像: $original_image -> $private_image" >&2
        elif [[ "$line" =~ ^[[:space:]]*-[[:space:]]*JUPYTERHUB_IMAGE=(.+)$ ]]; then
            # 处理 JUPYTERHUB_IMAGE 环境变量
            local original_image="${BASH_REMATCH[1]}"
            if [[ "$original_image" == *"\${IMAGE_TAG"* ]]; then
                original_image="${original_image//\$\{IMAGE_TAG:-v0.0.3.3\}/$tag}"
                original_image="${original_image//\$\{IMAGE_TAG\}/$tag}"
            fi
            
            local private_image=$(get_private_image_name "$original_image" "$registry")
            
            # 保持原有的缩进
            local indent=""
            if [[ "$line" =~ ^([[:space:]]*) ]]; then
                indent="${BASH_REMATCH[1]}"
            fi
            
            echo "${indent}- JUPYTERHUB_IMAGE=$private_image"
            print_info "替换环境变量: JUPYTERHUB_IMAGE=$original_image -> $private_image" >&2
        else
            echo "$line"
        fi
    done < "$compose_file" > "$temp_file"
    
    # 替换原文件
    mv "$temp_file" "$compose_file"
    
    print_success "docker-compose.yml 修改完成"
}

# 拉取所有镜像到本地
pull_all_images() {
    local registry="$1"
    local tag="$2"
    
    print_info "从私有仓库拉取所有镜像..."
    
    # 提取所有镜像
    local images=$(extract_images_from_compose "$DOCKER_COMPOSE_FILE")
    
    while IFS= read -r original_image; do
        if [[ -n "$original_image" ]]; then
            # 替换环境变量
            if [[ "$original_image" == *"\${IMAGE_TAG"* ]]; then
                original_image="${original_image//\$\{IMAGE_TAG:-v0.0.3.3\}/$tag}"
                original_image="${original_image//\$\{IMAGE_TAG\}/$tag}"
            fi
            
            local private_image=$(get_private_image_name "$original_image" "$registry")
            
            print_info "拉取镜像: $private_image"
            if docker pull "$private_image"; then
                print_success "拉取成功: $private_image"
            else
                print_warning "拉取失败: $private_image"
            fi
        fi
    done <<< "$images"
}

# 标签并推送所有镜像
tag_and_push_all_images() {
    local registry="$1"
    local tag="$2"
    
    print_info "标签并推送所有镜像到私有仓库..."
    
    # 提取所有镜像
    local images=$(extract_images_from_compose "$DOCKER_COMPOSE_FILE")
    
    while IFS= read -r original_image; do
        if [[ -n "$original_image" ]]; then
            # 替换环境变量
            local processed_image="$original_image"
            if [[ "$original_image" == *"\${IMAGE_TAG"* ]]; then
                processed_image="${original_image//\$\{IMAGE_TAG:-v0.0.3.3\}/$tag}"
                processed_image="${processed_image//\$\{IMAGE_TAG\}/$tag}"
            fi
            
            local private_image=$(get_private_image_name "$processed_image" "$registry")
            
            print_info "处理镜像: $processed_image -> $private_image"
            
            # 检查本地是否存在原始镜像
            if docker image inspect "$processed_image" >/dev/null 2>&1; then
                print_info "标签镜像: $processed_image -> $private_image"
                docker tag "$processed_image" "$private_image"
                
                print_info "推送镜像: $private_image"
                if docker push "$private_image"; then
                    print_success "推送成功: $private_image"
                else
                    print_error "推送失败: $private_image"
                fi
            else
                print_warning "本地镜像不存在: $processed_image"
                
                # 尝试从 Docker Hub 拉取并转推
                print_info "尝试从公共仓库拉取: $processed_image"
                if docker pull "$processed_image"; then
                    print_info "标签镜像: $processed_image -> $private_image"
                    docker tag "$processed_image" "$private_image"
                    
                    print_info "推送镜像: $private_image"
                    if docker push "$private_image"; then
                        print_success "推送成功: $private_image"
                    else
                        print_error "推送失败: $private_image"
                    fi
                else
                    print_error "无法拉取镜像: $processed_image"
                fi
            fi
        fi
    done <<< "$images"
}

# 启动服务
start_services() {
    print_info "启动服务..."
    
    local compose_cmd
    if command -v docker-compose &> /dev/null; then
        compose_cmd="docker-compose"
    elif docker compose version &> /dev/null; then
        compose_cmd="docker compose"
    else
        print_error "Docker Compose 未找到"
        exit 1
    fi
    
    # 确定使用的环境文件
    local env_file=".env"
    if [[ -n "$PRIVATE_REGISTRY" ]]; then
        env_file=".env.prod"
    fi
    
    # 设置环境变量
    export IMAGE_TAG="$IMAGE_TAG"
    export ENV_FILE="$env_file"
    
    print_info "使用环境文件: $env_file"
    print_info "镜像标签: $IMAGE_TAG"
    
    $compose_cmd --env-file "$env_file" up -d
    
    print_success "服务已启动"
    
    # 显示服务状态
    print_info "检查服务状态..."
    sleep 5
    $compose_cmd ps
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

# 显示镜像映射
show_image_mappings() {
    local registry="${1:-<private-registry>}"
    
    print_info "镜像映射表 (目标仓库: $registry)"
    echo ""
    
    printf "%-40s -> %-60s\n" "原始镜像" "私有仓库镜像"
    printf "%-40s -> %-60s\n" "----------------------------------------" "------------------------------------------------------------"
    
    # 从 docker-compose.yml 提取实际使用的镜像
    local images=$(extract_images_from_compose "$DOCKER_COMPOSE_FILE")
    
    while IFS= read -r original_image; do
        if [[ -n "$original_image" ]]; then
            # 替换环境变量显示
            local display_image="$original_image"
            if [[ "$original_image" == *"\${IMAGE_TAG"* ]]; then
                display_image="${original_image//\$\{IMAGE_TAG:-v0.0.3.3\}/\${IMAGE_TAG\}}"
                display_image="${display_image//\$\{IMAGE_TAG\}/\${IMAGE_TAG\}}"
            fi
            
            local private_image=$(get_private_image_name "$original_image" "$registry")
            # 替换实际的镜像标签为变量显示
            private_image="${private_image//:$IMAGE_TAG/:\${IMAGE_TAG\}}"
            
            printf "%-40s -> %-60s\n" "$display_image" "$private_image"
        fi
    done <<< "$images"
    
    echo ""
}

# 显示帮助信息
show_help() {
    cat << EOF
AI-Infra-Matrix 全栈镜像管理脚本

用法: $0 <command> [options]

命令:
  registry <registry_url> [tag]           设置私有仓库并修改 docker-compose.yml
  pull <registry_url> [tag]              从私有仓库拉取所有镜像
  push <registry_url> [tag]              标签并推送所有镜像到私有仓库
  start [registry_url] [tag]             启动服务（可选使用私有仓库）
  stop                                   停止所有服务
  images [registry_url]                  显示镜像映射表
  restore                                恢复原始 docker-compose.yml
  help                                   显示帮助信息
  
  其他命令会传递给 scripts/all-ops.sh 处理

示例:
  # 设置私有仓库并修改 docker-compose.yml
  $0 registry registry.company.com/ai-infra v0.3.5
  
  # 推送所有镜像到私有仓库
  $0 push registry.company.com/ai-infra v0.3.5
  
  # 从私有仓库启动服务
  $0 start registry.company.com/ai-infra v0.3.5
  
  # 显示镜像映射
  $0 images registry.company.com/ai-infra
  
  # 停止服务
  $0 stop
  
  # 恢复原始配置
  $0 restore

支持的私有仓库格式:
  - registry.company.com/project
  - harbor.company.com/ai-infra
  - xxx.dockerhub.com/xxx-project

环境变量:
  IMAGE_TAG              镜像标签 (默认: v0.3.5)
  ENV_FILE               环境文件 (默认: .env.prod)

注意:
  - 脚本会自动备份原始 docker-compose.yml 为 docker-compose.yml.backup
  - 使用 'restore' 命令可以恢复原始配置
EOF
}

# 清理配置文件
clean_configs() {
    restore_compose_file "$DOCKER_COMPOSE_FILE" "$DOCKER_COMPOSE_BACKUP"
}

# 主逻辑
main() {
    local command="${1:-help}"
    
    case "$command" in
        "registry")
            if [[ $# -lt 2 ]]; then
                print_error "用法: $0 registry <registry_url> [tag]"
                print_info "示例: $0 registry registry.company.com/ai-infra v0.3.5"
                exit 1
            fi
            PRIVATE_REGISTRY="$2"
            IMAGE_TAG="${3:-$IMAGE_TAG}"
            backup_compose_file "$DOCKER_COMPOSE_FILE" "$DOCKER_COMPOSE_BACKUP"
            modify_compose_images "$PRIVATE_REGISTRY" "$IMAGE_TAG" "$DOCKER_COMPOSE_FILE"
            ;;
        "pull")
            if [[ $# -lt 2 ]]; then
                print_error "用法: $0 pull <registry_url> [tag]"
                exit 1
            fi
            PRIVATE_REGISTRY="$2"
            IMAGE_TAG="${3:-$IMAGE_TAG}"
            pull_all_images "$PRIVATE_REGISTRY" "$IMAGE_TAG"
            ;;
        "push")
            if [[ $# -lt 2 ]]; then
                print_error "用法: $0 push <registry_url> [tag]"
                exit 1
            fi
            PRIVATE_REGISTRY="$2"
            IMAGE_TAG="${3:-$IMAGE_TAG}"
            tag_and_push_all_images "$PRIVATE_REGISTRY" "$IMAGE_TAG"
            ;;
        "start")
            if [[ $# -ge 2 ]]; then
                PRIVATE_REGISTRY="$2"
                IMAGE_TAG="${3:-$IMAGE_TAG}"
                backup_compose_file "$DOCKER_COMPOSE_FILE" "$DOCKER_COMPOSE_BACKUP"
                modify_compose_images "$PRIVATE_REGISTRY" "$IMAGE_TAG" "$DOCKER_COMPOSE_FILE"
            fi
            start_services
            ;;
        "stop")
            stop_services
            ;;
        "images")
            show_image_mappings "${2:-<private-registry>}"
            ;;
        "restore")
            restore_compose_file "$DOCKER_COMPOSE_FILE" "$DOCKER_COMPOSE_BACKUP"
            ;;
        "clean")
            clean_configs
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
