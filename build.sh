#!/bin/bash

# AI Infrastructure Matrix - 精简构建脚本
# 版本: v1.0.0
# 专注于 src/ 目录下的 Dockerfile 构建

set -e

# 操作系统检测
detect_os() {
    if [[ "$OSTYPE" == "darwin"* ]]; then
        echo "macOS"
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        echo "Linux"
    else
        echo "Other"
    fi
}

# 全局变量
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VERSION="1.0.0"
CONFIG_FILE="$SCRIPT_DIR/config.toml"
OS_TYPE=$(detect_os)

# ==========================================
# 配置文件解析功能
# ==========================================

# 读取TOML配置文件中的值
read_config() {
    local section="$1"
    local key="$2"
    local subsection="$3"
    
    if [[ ! -f "$CONFIG_FILE" ]]; then
        print_error "配置文件不存在: $CONFIG_FILE"
        return 1
    fi
    
    if [[ -n "$subsection" ]]; then
        # 读取嵌套配置 [section.subsection]
        awk -F' *= *' -v section="$section" -v subsection="$subsection" -v key="$key" '
            /^\[[[:space:]]*[^.]+\.[^]]+\]/ {
                # 匹配 [section.subsection] 格式
                gsub(/^\[|\]$/, "")
                split($0, parts, "\\.")
                if (parts[1] == section && parts[2] == subsection) {
                    in_target = 1
                } else {
                    in_target = 0
                }
                next
            }
            /^\[/ { in_target = 0; next }
            in_target && $1 == key {
                gsub(/^"/, "", $2)
                gsub(/"$/, "", $2)
                print $2
                exit
            }
        ' "$CONFIG_FILE"
    else
        # 读取简单配置 [section]
        awk -F' *= *' -v section="$section" -v key="$key" '
            /^\[[[:space:]]*[^.]+\]/ {
                gsub(/^\[|\]$/, "")
                if ($0 == section) {
                    in_target = 1
                } else {
                    in_target = 0
                }
                next
            }
            /^\[/ { in_target = 0; next }
            in_target && $1 == key {
                gsub(/^"/, "", $2)
                gsub(/"$/, "", $2)
                print $2
                exit
            }
        ' "$CONFIG_FILE"
    fi
}

# 获取所有服务名称
get_all_services() {
    if [[ ! -f "$CONFIG_FILE" ]]; then
        echo "backend frontend jupyterhub nginx saltstack"
        return
    fi
    
    awk '
        /^\[services\.[^]]+\]/ {
            gsub(/^\[services\.|\]$/, "")
            print $0
        }
    ' "$CONFIG_FILE" | sort
}

# 获取所有依赖镜像
get_all_dependencies() {
    if [[ ! -f "$CONFIG_FILE" ]]; then
        echo "postgres:15-alpine redis:7-alpine osixia/openldap:stable osixia/phpldapadmin:stable tecnativa/tcp-proxy redislabs/redisinsight:latest nginx:1.27-alpine quay.io/minio/minio:latest"
        return
    fi
    
    awk -F' *= *' '
        /^\[dependencies\]/ { in_dependencies = 1; next }
        /^\[/ { in_dependencies = 0; next }
        in_dependencies && NF > 1 {
            gsub(/^"/, "", $2)
            gsub(/"$/, "", $2)
            print $2
        }
    ' "$CONFIG_FILE" | tr '\n' ' '
}

# 获取所有扩展组件镜像
get_all_extensions() {
    if [[ ! -f "$CONFIG_FILE" ]]; then
        echo "ai-infra-singleuser ai-infra-backend-init ai-infra-gitea"
        return
    fi
    
    awk -F' *= *' '
        /^\[extensions\]/ { in_extensions = 1; next }
        /^\[/ { in_extensions = 0; next }
        in_extensions && NF > 1 {
            gsub(/^"/, "", $2)
            gsub(/"$/, "", $2)
            print $2
        }
    ' "$CONFIG_FILE" | tr '\n' ' '
}

# 初始化配置
DEFAULT_IMAGE_TAG=$(read_config "project" "version")
[[ -z "$DEFAULT_IMAGE_TAG" ]] && DEFAULT_IMAGE_TAG="v0.3.5"

# 动态加载服务和依赖配置
SRC_SERVICES=$(get_all_services | tr '\n' ' ')
DEPENDENCY_IMAGES=$(get_all_dependencies | tr '\n' ' ')
EXTENSION_IMAGES=$(get_all_extensions | tr '\n' ' ')

# 动态收集依赖镜像函数
collect_dependency_images() {
    # 优先使用配置文件中的依赖镜像列表
    if [[ -f "$CONFIG_FILE" ]]; then
        echo "$DEPENDENCY_IMAGES"
        return
    fi
    
    # 后备方案：从docker-compose文件中提取
    local compose_files=()
    local script_dir="$(cd "$(dirname "$0")" && pwd)"
    
    # 收集所有compose文件
    [ -f "docker-compose.yml" ] && compose_files+=("docker-compose.yml")
    [ -f "src/docker/production/docker-compose.yml" ] && compose_files+=("src/docker/production/docker-compose.yml")
    
    if [ ${#compose_files[@]} -eq 0 ]; then
        print_warning "未找到docker-compose.yml文件，使用静态依赖列表"
        echo "postgres:15-alpine redis:7-alpine osixia/openldap:stable osixia/phpldapadmin:stable tecnativa/tcp-proxy redislabs/redisinsight:latest nginx:1.27-alpine quay.io/minio/minio:latest"
        return
    fi
    
    # 提取所有镜像，排除ai-infra-*镜像
    local images_list
    images_list=$(
        for f in "${compose_files[@]}"; do
            grep -E '^[[:space:]]*image:[[:space:]]' "$f" 2>/dev/null | \
                sed -E 's/^[[:space:]]*image:[[:space:]]*//' | \
                sed -E 's/[[:space:]]+#.*$//' | \
                tr -d '"' | tr -d "'" | \
                sed 's/\${[^}]*}//' | \
                sed 's/:$//' || true
        done | \
        grep -vE '^(ai-infra-|$)' | \
        awk 'NF{print $1}' | sort -u
    )
    
    # 返回收集到的镜像列表
    if [ -n "$images_list" ]; then
        echo "$images_list" | tr '\n' ' '
    else
        echo "postgres:15-alpine redis:7-alpine osixia/openldap:stable osixia/phpldapadmin:stable tecnativa/tcp-proxy redislabs/redisinsight:latest nginx:1.27-alpine quay.io/minio/minio:latest"
    fi
}

# Mock 数据测试相关配置
MOCK_DATA_ENABLED="${MOCK_DATA_ENABLED:-false}"
MOCK_POSTGRES_IMAGE="postgres:15-alpine"
MOCK_REDIS_IMAGE="redis:7-alpine"

# 获取服务对应的路径
get_service_path() {
    local service="$1"
    
    # 从配置文件读取路径
    local path=$(read_config "services" "path" "$service")
    
    # 如果配置文件中没有，使用后备方案
    if [[ -z "$path" ]]; then
        case "$service" in
            "backend") echo "src/backend" ;;
            "frontend") echo "src/frontend" ;;
            "jupyterhub") echo "src/jupyterhub" ;;
            "nginx") echo "src/nginx" ;;
            "saltstack") echo "src/saltstack" ;;
            *) echo "" ;;
        esac
    else
        echo "$path"
    fi
}

# 颜色输出函数
print_info() {
    echo -e "\033[34m[INFO]\033[0m $1"
}

print_success() {
    echo -e "\033[32m[SUCCESS]\033[0m $1"
}

print_warning() {
    echo -e "\033[33m[WARNING]\033[0m $1"
}

print_error() {
    echo -e "\033[31m[ERROR]\033[0m $1"
}

# ==========================================
# Docker Compose 版本检测和适配
# ==========================================

# 检测Docker Compose版本并返回最佳命令
detect_compose_command() {
    local compose_cmd=""
    local compose_version=""
    
    # 优先使用docker compose (v2)
    if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
        compose_cmd="docker compose"
        compose_version=$(docker compose version --short 2>/dev/null || docker compose version | grep -o 'v[0-9.]*' | head -1)
        echo "$compose_cmd"
        return 0
    fi
    
    # 回退到docker-compose (v1)
    if command -v docker-compose >/dev/null 2>&1; then
        compose_cmd="docker-compose"
        compose_version=$(docker-compose version --short 2>/dev/null || docker-compose version | grep -o '[0-9.]*' | head -1)
        echo "$compose_cmd"
        return 0
    fi
    
    return 1
}

# 检查Docker Compose版本兼容性
check_compose_compatibility() {
    local compose_cmd
    compose_cmd=$(detect_compose_command)
    local exit_code=$?
    
    if [ $exit_code -ne 0 ]; then
        print_error "未找到Docker Compose命令"
        print_info "请安装Docker Compose v2.0+:"
        print_info "  https://docs.docker.com/compose/install/"
        return 1
    fi
    
    local version=""
    if [[ "$compose_cmd" == "docker compose" ]]; then
        version=$(docker compose version --short 2>/dev/null || docker compose version | grep -o 'v[0-9.]*' | head -1 | sed 's/v//')
        print_info "检测到Docker Compose v2: $version"
        
        # 清理版本号，移除v前缀和额外信息
        local clean_version=$(echo "$version" | sed 's/^v//' | sed 's/-.*$//')
        
        # 检查是否为v2.39.2或更高版本
        if command -v python3 >/dev/null 2>&1; then
            local is_compatible=$(python3 -c "
import sys
from packaging import version
try:
    current = version.parse('$clean_version')
    required = version.parse('2.39.2')
    print('true' if current >= required else 'false')
except Exception as e:
    print('true')  # 默认兼容
" 2>/dev/null || echo "true")
            
            if [[ "$is_compatible" == "true" ]]; then
                print_success "✓ Docker Compose版本兼容 (v$clean_version >= v2.39.2)"
            else
                print_warning "⚠ Docker Compose版本较旧 (v$clean_version < v2.39.2)，建议升级"
                print_info "当前版本应该仍可工作，但建议升级以获得最佳体验"
            fi
        else
            print_info "✓ 使用Docker Compose v2: $clean_version"
        fi
    else
        version=$(docker-compose version --short 2>/dev/null || docker-compose version | grep -o '[0-9.]*' | head -1)
        print_warning "检测到Docker Compose v1: $version"
        print_info "建议升级到Docker Compose v2以获得更好的性能和功能"
    fi
    
    echo "$compose_cmd"
    return 0
}

# 验证compose文件格式
validate_compose_file() {
    local file="$1"
    local compose_cmd="$2"
    
    if [[ ! -f "$file" ]]; then
        print_error "Compose文件不存在: $file"
        return 1
    fi
    
    print_info "验证compose文件: $file"
    
    if ! $compose_cmd -f "$file" config >/dev/null 2>&1; then
        print_error "Compose文件验证失败: $file"
        print_info "详细错误信息："
        $compose_cmd -f "$file" config 2>&1 | head -10
        return 1
    fi
    
    print_success "✓ Compose文件验证通过: $file"
    return 0
}

# 获取私有镜像名称（支持Harbor格式：registry/project）
get_private_image_name() {
    local original_image="$1"
    local registry="$2"
    
    if [[ -z "$registry" ]]; then
        echo "$original_image"
        return 0
    fi
    
    # 检查original_image是否已经包含了registry信息
    if [[ "$original_image" == "$registry"/* ]]; then
        echo "$original_image"
        return 0
    fi
    
    # 处理不同类型的registry格式
    local registry_base=""
    local project_path=""
    local is_harbor_style=false
    
    if [[ "$registry" == *"/"* ]]; then
        # Harbor格式：registry.xxx.com/project
        is_harbor_style=true
        registry_base="${registry%%/*}"  # 获取 registry.xxx.com
        project_path="${registry#*/}"    # 获取 project
    else
        # 传统格式：registry.xxx.com
        registry_base="$registry"
    fi
    
    # 处理镜像名称
    local image_name_tag=""
    
    if [[ "$original_image" == *"/"* ]]; then
        # 包含组织/用户名的镜像
        if [[ "$original_image" == *"."*"/"* ]]; then
            # 第三方仓库镜像 (如 quay.io/minio/minio:latest)
            image_name_tag="${original_image#*/}"  # 移除域名部分
        else
            # Docker Hub 组织镜像 (如 osixia/openldap:stable)
            image_name_tag="$original_image"
        fi
    else
        # 简单镜像名 (如 redis:7-alpine, postgres:15-alpine)
        image_name_tag="$original_image"
    fi
    
    # 构建最终镜像路径
    if [[ "$is_harbor_style" == "true" ]]; then
        # Harbor模式：registry.xxx.com/project/image:tag
        echo "${registry}/${image_name_tag}"
    else
        # 传统模式：registry.xxx.com/image:tag
        echo "${registry}/${image_name_tag}"
    fi
}

# 检查 Dockerfile 是否存在
check_dockerfile() {
    local service="$1"
    local service_path=$(get_service_path "$service")
    
    if [[ -z "$service_path" ]]; then
        print_error "未知服务: $service"
        return 1
    fi
    
    local dockerfile_path="$SCRIPT_DIR/$service_path/Dockerfile"
    
    if [[ ! -f "$dockerfile_path" ]]; then
        print_error "Dockerfile 不存在: $dockerfile_path"
        return 1
    fi
    return 0
}

# 构建单个服务镜像
build_service() {
    local service="$1"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    local registry="${3:-}"
    
    local service_path=$(get_service_path "$service")
    if [[ -z "$service_path" ]]; then
        print_error "未知服务: $service"
        print_info "可用服务: $SRC_SERVICES"
        return 1
    fi
    
    if ! check_dockerfile "$service"; then
        return 1
    fi
    
    local dockerfile_path="$SCRIPT_DIR/$service_path/Dockerfile"
    local base_image="ai-infra-${service}:${tag}"
    
    # 确定目标镜像名
    local target_image="$base_image"
    if [[ -n "$registry" ]]; then
        target_image=$(get_private_image_name "$base_image" "$registry")
    fi
    
    print_info "构建服务: $service"
    print_info "  Dockerfile: $service_path/Dockerfile"
    print_info "  目标镜像: $target_image"
    
    # 构建镜像
    if docker build -f "$dockerfile_path" -t "$target_image" "$SCRIPT_DIR"; then
        print_success "✓ 构建成功: $target_image"
        
        # 如果指定了registry，同时创建本地别名
        if [[ -n "$registry" ]] && [[ "$target_image" != "$base_image" ]]; then
            if docker tag "$target_image" "$base_image"; then
                print_info "  ✓ 本地别名: $base_image"
            fi
        fi
        
        return 0
    else
        print_error "✗ 构建失败: $target_image"
        return 1
    fi
}

# 构建所有服务镜像
build_all_services() {
    local tag="${1:-$DEFAULT_IMAGE_TAG}"
    local registry="${2:-}"
    
    print_info "=========================================="
    print_info "构建所有 AI-Infra 服务镜像"
    print_info "=========================================="
    print_info "镜像标签: $tag"
    if [[ -n "$registry" ]]; then
        print_info "目标仓库: $registry"
    else
        print_info "目标仓库: 本地构建"
    fi
    echo
    
    local success_count=0
    local total_count=0
    local failed_services=()
    
    # 计算服务总数
    for service in $SRC_SERVICES; do
        total_count=$((total_count + 1))
    done
    
    for service in $SRC_SERVICES; do
        if build_service "$service" "$tag" "$registry"; then
            success_count=$((success_count + 1))
        else
            failed_services+=("$service")
        fi
        echo
    done
    
    print_info "=========================================="
    print_success "构建完成: $success_count/$total_count 成功"
    
    if [[ ${#failed_services[@]} -gt 0 ]]; then
        print_warning "失败的服务: ${failed_services[*]}"
        return 1
    else
        print_success "🎉 所有服务构建成功！"
        return 0
    fi
}

# 推送单个服务镜像
push_service() {
    local service="$1"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    local registry="$3"
    
    if [[ -z "$registry" ]]; then
        print_error "推送操作需要指定 registry"
        return 1
    fi
    
    local base_image="ai-infra-${service}:${tag}"
    local target_image=$(get_private_image_name "$base_image" "$registry")
    
    print_info "推送服务: $service"
    print_info "  目标镜像: $target_image"
    
    # 检查镜像是否存在
    if ! docker image inspect "$target_image" >/dev/null 2>&1; then
        print_warning "镜像不存在，尝试构建..."
        if ! build_service "$service" "$tag" "$registry"; then
            return 1
        fi
    fi
    
    # 推送镜像
    if docker push "$target_image"; then
        print_success "✓ 推送成功: $target_image"
        return 0
    else
        print_error "✗ 推送失败: $target_image"
        return 1
    fi
}

# 推送所有服务镜像
push_all_services() {
    local tag="${1:-$DEFAULT_IMAGE_TAG}"
    local registry="$2"
    
    if [[ -z "$registry" ]]; then
        print_error "推送操作需要指定 registry"
        print_info "用法: $0 push-all <registry> [tag]"
        return 1
    fi
    
    print_info "=========================================="
    print_info "推送所有 AI-Infra 服务镜像"
    print_info "=========================================="
    print_info "目标仓库: $registry"
    print_info "镜像标签: $tag"
    echo
    
    local success_count=0
    local total_count=0
    local failed_services=()
    
    # 计算服务总数
    for service in $SRC_SERVICES; do
        total_count=$((total_count + 1))
    done
    
    for service in $SRC_SERVICES; do
        if push_service "$service" "$tag" "$registry"; then
            success_count=$((success_count + 1))
        else
            failed_services+=("$service")
        fi
        echo
    done
    
    print_info "=========================================="
    print_success "推送完成: $success_count/$total_count 成功"
    
    if [[ ${#failed_services[@]} -gt 0 ]]; then
        print_warning "失败的服务: ${failed_services[*]}"
        return 1
    else
        print_success "🚀 所有服务推送成功！"
        return 0
    fi
}

# 一键构建并推送
build_and_push_all() {
    local tag="${1:-$DEFAULT_IMAGE_TAG}"
    local registry="$2"
    
    if [[ -z "$registry" ]]; then
        print_error "一键构建推送需要指定 registry"
        print_info "用法: $0 build-push <registry> [tag]"
        return 1
    fi
    
    print_info "=========================================="
    print_info "一键构建并推送所有服务"
    print_info "=========================================="
    print_info "目标仓库: $registry"
    print_info "镜像标签: $tag"
    echo
    
    # 第一阶段：构建所有镜像
    print_info "🔨 第一阶段：构建所有镜像..."
    if ! build_all_services "$tag" "$registry"; then
        print_error "构建阶段失败，停止执行"
        return 1
    fi
    
    echo
    print_info "🚀 第二阶段：推送所有镜像..."
    if ! push_all_services "$tag" "$registry"; then
        print_error "推送阶段失败"
        return 1
    fi
    
    print_success "🎉 一键构建推送完成！"
}

# 拉取并标记依赖镜像
pull_and_tag_dependencies() {
    local registry="$1"
    local tag="${2:-latest}"
    
    if [[ -z "$registry" ]]; then
        print_error "需要指定 registry"
        print_info "用法: $0 deps-pull <registry> [tag]"
        return 1
    fi
    
    print_info "=========================================="
    print_info "拉取并标记依赖镜像到 $registry"
    print_info "=========================================="
    print_info "目标标签: $tag"
    
    # 动态收集依赖镜像
    local dependency_images
    dependency_images=$(collect_dependency_images)
    print_info "收集到依赖镜像: $dependency_images"
    echo
    
    local success_count=0
    local total_count=0
    local failed_deps=()
    
    for dep_image in $dependency_images; do
        total_count=$((total_count + 1))
        print_info "处理依赖镜像: $dep_image"
        
        # 拉取原始镜像
        if docker pull "$dep_image"; then
            print_success "  ✓ 拉取成功: $dep_image"
            
            # 生成目标镜像名（使用统一的命名规则）
            local base_name
            if [[ "$dep_image" == *"/"* ]]; then
                # 包含组织名的镜像，提取最后的镜像名
                base_name=$(echo "$dep_image" | sed 's|.*/||' | sed 's|:.*||')
            else
                # 简单镜像名
                base_name=$(echo "$dep_image" | sed 's|:.*||')
            fi
            
            # 使用get_private_image_name函数生成目标镜像名
            local target_image
            target_image=$(get_private_image_name "ai-infra-deps-$base_name:$tag" "$registry")
            
            # 标记镜像
            if docker tag "$dep_image" "$target_image"; then
                print_success "  ✓ 标记成功: $target_image"
                success_count=$((success_count + 1))
            else
                print_error "  ✗ 标记失败: $target_image"
                failed_deps+=("$dep_image")
            fi
        else
            print_error "  ✗ 拉取失败: $dep_image"
            failed_deps+=("$dep_image")
        fi
        echo
    done
    
    print_info "=========================================="
    print_success "依赖镜像处理完成: $success_count/$total_count 成功"
    
    if [[ ${#failed_deps[@]} -gt 0 ]]; then
        print_warning "失败的依赖镜像: ${failed_deps[*]}"
        return 1
    else
        print_success "🎉 所有依赖镜像处理成功！"
        return 0
    fi
}

# 推送依赖镜像
push_dependencies() {
    local registry="$1"
    local tag="${2:-latest}"
    
    if [[ -z "$registry" ]]; then
        print_error "需要指定 registry"
        print_info "用法: $0 deps-push <registry> [tag]"
        return 1
    fi
    
    print_info "=========================================="
    print_info "推送依赖镜像到 $registry"
    print_info "=========================================="
    print_info "目标标签: $tag"
    
    # 动态收集依赖镜像
    local dependency_images
    dependency_images=$(collect_dependency_images)
    print_info "收集到依赖镜像: $dependency_images"
    echo
    
    local success_count=0
    local total_count=0
    local failed_deps=()
    
    for dep_image in $dependency_images; do
        total_count=$((total_count + 1))
        
        # 生成目标镜像名（与拉取时保持一致）
        local base_name
        if [[ "$dep_image" == *"/"* ]]; then
            # 包含组织名的镜像，提取最后的镜像名
            base_name=$(echo "$dep_image" | sed 's|.*/||' | sed 's|:.*||')
        else
            # 简单镜像名
            base_name=$(echo "$dep_image" | sed 's|:.*||')
        fi
        
        # 使用get_private_image_name函数生成目标镜像名
        local target_image
        target_image=$(get_private_image_name "ai-infra-deps-$base_name:$tag" "$registry")
        
        print_info "推送依赖镜像: $target_image"
        
        if docker push "$target_image"; then
            print_success "  ✓ 推送成功: $target_image"
            success_count=$((success_count + 1))
        else
            print_error "  ✗ 推送失败: $target_image"
            failed_deps+=("$target_image")
        fi
        echo
    done
    
    print_info "=========================================="
    print_success "依赖镜像推送完成: $success_count/$total_count 成功"
    
    if [[ ${#failed_deps[@]} -gt 0 ]]; then
        print_warning "失败的依赖镜像: ${failed_deps[*]}"
        return 1
    else
        print_success "🎉 所有依赖镜像推送成功！"
        return 0
    fi
}

# ==========================================
# 生产环境部署相关功能
# ==========================================

# 生成生产环境配置文件
generate_production_config() {
    local registry="$1"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    local base_file="docker-compose.yml"
    local output_file="docker-compose.prod.yml"
    
    if [[ -z "$registry" ]]; then
        print_error "请指定目标 registry"
        return 1
    fi
    
    if [[ ! -f "$base_file" ]]; then
        print_error "基础配置文件不存在: $base_file"
        return 1
    fi
    
    # 验证原始配置文件
    print_info "验证原始配置文件..."
    local compose_cmd=$(detect_compose_command)
    local compose_version=$(echo "$compose_cmd" | cut -d'|' -f2)
    compose_cmd=$(echo "$compose_cmd" | cut -d'|' -f1)
    
    if [[ "$compose_cmd" != "none" ]]; then
        # 验证compose文件
        if ! validate_compose_file "$base_file" "$compose_cmd"; then
            print_error "原始配置文件验证失败: $base_file"
            return 1
        fi
        print_success "配置文件验证通过 (使用 $compose_cmd $compose_version)"
    else
        print_warning "未找到可用的Docker Compose命令，跳过原始配置验证"
    fi
    
    print_info "生成生产环境配置文件..."
    print_info "  Registry: $registry"
    print_info "  Tag: $tag"
    print_info "  输出文件: $output_file"
    echo
    
    # 复制基础配置文件
    cp "$base_file" "$output_file"
    
    # 1. 更新镜像registry路径
    print_info "更新镜像registry路径... (OS: $OS_TYPE)"
    # 兼容macOS和Linux的sed命令
    if [[ "$OS_TYPE" == "macOS" ]]; then
        sed -i.bak "s|ghcr.io/aresnasa/ai-infra-matrix|${registry}/ai-infra-matrix|g" "$output_file"
        sed -i.bak "s|image: ai-infra-|image: ${registry}/ai-infra-|g" "$output_file"
    else
        sed -i "s|ghcr.io/aresnasa/ai-infra-matrix|${registry}/ai-infra-matrix|g" "$output_file"
        sed -i "s|image: ai-infra-|image: ${registry}/ai-infra-|g" "$output_file"
    fi
    
    # 2. 更新镜像标签
    print_info "更新镜像标签..."
    if [[ "$OS_TYPE" == "macOS" ]]; then
        sed -i.bak "s|:latest|:${tag}|g" "$output_file"
        sed -i.bak "s|\${IMAGE_TAG}|${tag}|g" "$output_file"
        sed -i.bak "s|\${IMAGE_TAG:-v[^}]*}|${tag}|g" "$output_file"
    else
        sed -i "s|:latest|:${tag}|g" "$output_file"
        sed -i "s|\${IMAGE_TAG}|${tag}|g" "$output_file"
        sed -i "s|\${IMAGE_TAG:-v[^}]*}|${tag}|g" "$output_file"
    fi
    
    # 3. 移除LDAP相关服务（使用Python脚本精确处理）
    print_info "移除openldap和phpldapadmin服务..."
    
    # 检查是否有Python和PyYAML
    if command -v python3 >/dev/null 2>&1 && python3 -c "import yaml" 2>/dev/null; then
        # 使用Python脚本精确移除LDAP服务
        if python3 fix_ldap_removal.py "$output_file" "$output_file.tmp" 2>/dev/null; then
            mv "$output_file.tmp" "$output_file"
            print_success "✓ 使用Python脚本成功移除LDAP服务"
        else
            print_warning "Python脚本移除失败，使用备用方案"
            # 备用方案：使用sed简单移除（兼容macOS和Linux）
            if [[ "$OS_TYPE" == "macOS" ]]; then
                sed -i.bak '/^  openldap:/,/^  [a-zA-Z]/{ /^  [a-zA-Z]/!d; }' "$output_file"
                sed -i.bak '/^  phpldapadmin:/,/^  [a-zA-Z]/{ /^  [a-zA-Z]/!d; }' "$output_file"
                sed -i.bak '/^  openldap:/d' "$output_file"
                sed -i.bak '/^  phpldapadmin:/d' "$output_file"
            else
                sed -i '/^  openldap:/,/^  [a-zA-Z]/{ /^  [a-zA-Z]/!d; }' "$output_file"
                sed -i '/^  phpldapadmin:/,/^  [a-zA-Z]/{ /^  [a-zA-Z]/!d; }' "$output_file"
                sed -i '/^  openldap:/d' "$output_file"
                sed -i '/^  phpldapadmin:/d' "$output_file"
            fi
        fi
    else
        print_warning "未安装PyYAML，使用简化方案移除LDAP服务"
        # 简化方案：使用sed移除服务块（兼容macOS和Linux）
        if [[ "$OS_TYPE" == "macOS" ]]; then
            sed -i.bak '/^  openldap:/,/^  [a-zA-Z]/{ /^  [a-zA-Z]/!d; }' "$output_file"
            sed -i.bak '/^  phpldapadmin:/,/^  [a-zA-Z]/{ /^  [a-zA-Z]/!d; }' "$output_file"
            sed -i.bak '/^  openldap:/d' "$output_file"
            sed -i.bak '/^  phpldapadmin:/d' "$output_file"
            
            # 手动移除一些可能的残留
            sed -i.bak '/LDAP_SERVER=/d' "$output_file"
            sed -i.bak '/PHPLDAPADMIN_/d' "$output_file"
            sed -i.bak '/openldap:/,/condition: service_healthy/d' "$output_file"
        else
            sed -i '/^  openldap:/,/^  [a-zA-Z]/{ /^  [a-zA-Z]/!d; }' "$output_file"
            sed -i '/^  phpldapadmin:/,/^  [a-zA-Z]/{ /^  [a-zA-Z]/!d; }' "$output_file"
            sed -i '/^  openldap:/d' "$output_file"
            sed -i '/^  phpldapadmin:/d' "$output_file"
            
            # 手动移除一些可能的残留
            sed -i '/LDAP_SERVER=/d' "$output_file"
            sed -i '/PHPLDAPADMIN_/d' "$output_file"
            sed -i '/openldap:/,/condition: service_healthy/d' "$output_file"
        fi
    fi
    
    # 4. 清理重复的networks配置（如果有的话）
    print_info "清理重复的networks配置..."
    awk '
    BEGIN { prev_line = "" }
    {
        # 如果当前行和上一行都是"    networks:"，则跳过当前行
        if ($0 ~ /^[[:space:]]*networks:[[:space:]]*$/ && prev_line ~ /^[[:space:]]*networks:[[:space:]]*$/) {
            next
        }
        print prev_line
        prev_line = $0
    }
    END { if (prev_line != "") print prev_line }
    ' "$output_file" > "$output_file.tmp" && mv "$output_file.tmp" "$output_file"
    
    # 5. 清理备份文件（仅在macOS上存在）
    if [[ "$OS_TYPE" == "macOS" ]]; then
        rm -f "$output_file.bak"
    fi
    
    # 6. 验证配置文件
    print_info "验证配置文件..."
    
    # 验证YAML语法
    local yaml_valid=false
    if command -v python3 >/dev/null 2>&1; then
        if python3 -c "
import yaml
import sys
try:
    with open('$output_file', 'r') as f:
        yaml.safe_load(f)
    print('✓ YAML语法正确')
    sys.exit(0)
except yaml.YAMLError as e:
    print(f'✗ YAML语法错误: {e}')
    sys.exit(1)
except Exception as e:
    print(f'✗ 文件读取错误: {e}')
    sys.exit(1)
"; then
            yaml_valid=true
        else
            print_error "YAML语法验证失败"
            return 1
        fi
    else
        print_warning "未安装Python3，跳过YAML语法验证"
        yaml_valid=true
    fi
    
    # 验证docker-compose配置
    if [[ "$yaml_valid" == "true" ]]; then
        print_info "验证docker-compose配置..."
        if command -v docker-compose >/dev/null 2>&1; then
            # 使用docker-compose config命令验证配置文件
            if docker-compose -f "$output_file" config >/dev/null 2>&1; then
                print_success "✓ docker-compose配置验证通过"
            else
                print_error "✗ docker-compose配置验证失败"
                print_info "详细错误信息："
                docker-compose -f "$output_file" config 2>&1 | head -10
                return 1
            fi
        elif command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
            # 使用docker compose命令验证配置文件
            if docker compose -f "$output_file" config >/dev/null 2>&1; then
                print_success "✓ docker compose配置验证通过"
            else
                print_error "✗ docker compose配置验证失败"
                print_info "详细错误信息："
                docker compose -f "$output_file" config 2>&1 | head -10
                return 1
            fi
        else
            print_warning "未安装docker-compose或docker compose，跳过配置验证"
        fi
    fi
    
    print_success "✓ 生产环境配置文件生成成功: $output_file"
    echo
    print_info "注意事项："
    print_info "  1. 请确保所有依赖镜像已推送到内部registry (使用 deps-all 命令)"
    print_info "  2. 请确保所有源码服务镜像已推送到内部registry (使用 build-push 命令)"
    print_info "  3. 生产环境已移除LDAP服务依赖，服务可独立启动"
    print_info "  4. 请检查生成的配置文件并根据需要调整环境变量"
    echo
    
    return 0
}


# 启动生产环境
start_production() {
    local registry="$1"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    local compose_file="docker-compose.prod.yml"
    local env_file=".env.prod"
    
    if [[ -z "$registry" ]]; then
        print_error "请指定目标 registry"
        return 1
    fi
    
    # 检查生产环境配置文件
    if [[ ! -f "$env_file" ]]; then
        print_error "生产环境配置文件不存在: $env_file"
        print_info "请使用以下命令生成生产环境密码:"
        print_info "  ./scripts/generate-prod-passwords.sh"
        return 1
    fi
    
    # 如果生产配置文件不存在，先生成
    if [[ ! -f "$compose_file" ]]; then
        print_info "生产配置文件不存在，正在生成..."
        if ! generate_production_config "$registry" "$tag"; then
            return 1
        fi
    fi
    
    print_info "=========================================="
    print_info "启动生产环境"
    print_info "=========================================="
    print_info "配置文件: $compose_file"
    print_info "环境文件: $env_file"
    print_info "Registry: $registry"
    print_info "标签: $tag"
    echo
    
    print_info "拉取所有镜像..."
    if ! ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" pull; then
        print_error "镜像拉取失败"
        return 1
    fi
    
    print_info "启动生产环境..."
    if ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" up -d; then
        print_success "✓ 生产环境启动成功"
        echo
        print_info "查看服务状态:"
        ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" ps
        return 0
    else
        print_error "✗ 生产环境启动失败"
        return 1
    fi
}

# 停止生产环境
stop_production() {
    local compose_file="docker-compose.prod.yml"
    local env_file=".env.prod"
    
    if [[ ! -f "$compose_file" ]]; then
        print_error "生产配置文件不存在: $compose_file"
        return 1
    fi
    
    print_info "=========================================="
    print_info "停止生产环境"
    print_info "=========================================="
    
    if ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" down; then
        print_success "✓ 生产环境已停止"
        return 0
    else
        print_error "✗ 生产环境停止失败"
        return 1
    fi
}

# 重启生产环境
restart_production() {
    local registry="$1"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    
    print_info "=========================================="
    print_info "重启生产环境"
    print_info "=========================================="
    
    # 先停止
    stop_production
    
    # 等待一段时间
    sleep 2
    
    # 再启动
    start_production "$registry" "$tag"
}

# 查看生产环境状态
production_status() {
    local compose_file="docker-compose.prod.yml"
    local env_file=".env.prod"
    
    if [[ ! -f "$compose_file" ]]; then
        print_error "生产配置文件不存在: $compose_file"
        return 1
    fi
    
    print_info "=========================================="
    print_info "生产环境状态"
    print_info "=========================================="
    
    ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" ps
}

# 查看生产环境日志
production_logs() {
    local compose_file="docker-compose.prod.yml"
    local env_file=".env.prod"
    local service="$1"
    local follow="${2:-false}"
    
    if [[ ! -f "$compose_file" ]]; then
        print_error "生产配置文件不存在: $compose_file"
        return 1
    fi
    
    if [[ -z "$service" ]]; then
        # 显示所有服务的日志
        if [[ "$follow" == "true" ]]; then
            ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" logs -f
        else
            ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" logs --tail=100
        fi
    else
        # 显示指定服务的日志
        if [[ "$follow" == "true" ]]; then
            ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" logs -f "$service"
        else
            ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" logs --tail=100 "$service"
        fi
    fi
}

# 创建简化的 Mock 测试环境（仅用于脚本功能验证）
setup_mock_environment() {
    local tag="${1:-$DEFAULT_IMAGE_TAG}"
    
    print_info "=========================================="
    print_info "设置 Mock 测试环境（脚本功能验证）"
    print_info "=========================================="
    print_info "镜像标签: $tag"
    echo
    
    # 创建 mock 数据目录
    local mock_dir="$SCRIPT_DIR/test/mock-data"
    mkdir -p "$mock_dir"
    
    # 创建简化的 Mock 测试 docker-compose 文件
    cat > "$mock_dir/docker-compose-mock.yml" << EOF
services:
  mock-postgres:
    image: postgres:15-alpine
    container_name: ai-infra-mock-postgres
    environment:
      POSTGRES_DB: test_db
      POSTGRES_USER: test_user
      POSTGRES_PASSWORD: test_pass
      TZ: Asia/Shanghai
    ports:
      - "15432:5432"
    volumes:
      - mock_postgres_data:/var/lib/postgresql/data
    networks:
      - mock-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U test_user -d test_db"]
      interval: 5s
      timeout: 3s
      retries: 3

  mock-redis:
    image: redis:7-alpine
    container_name: ai-infra-mock-redis
    command: redis-server --requirepass test_redis_pass
    ports:
      - "16379:6379"
    volumes:
      - mock_redis_data:/data
    networks:
      - mock-network
    healthcheck:
      test: ["CMD-SHELL", "redis-cli -a test_redis_pass ping || exit 1"]
      interval: 5s
      timeout: 3s
      retries: 3

  # 如果backend镜像存在，则启动测试实例
  mock-backend:
    image: ai-infra-backend:$tag
    container_name: ai-infra-mock-backend
    environment:
      POSTGRES_HOST: mock-postgres
      POSTGRES_PORT: 5432
      POSTGRES_DB: test_db
      POSTGRES_USER: test_user
      POSTGRES_PASSWORD: test_pass
      REDIS_HOST: mock-redis
      REDIS_PORT: 6379
      REDIS_PASSWORD: test_redis_pass
      MOCK_MODE: "true"
    ports:
      - "18080:8080"
    depends_on:
      mock-postgres:
        condition: service_healthy
      mock-redis:
        condition: service_healthy
    networks:
      - mock-network
    profiles:
      - backend-test

volumes:
  mock_postgres_data:
  mock_redis_data:

networks:
  mock-network:
    driver: bridge
EOF

    # 创建简单的测试脚本
    cat > "$mock_dir/test-connectivity.sh" << 'EOF'
#!/bin/bash
# Mock 环境连接测试脚本

echo "=== Mock 环境连接测试 ==="

# 测试 PostgreSQL
echo "测试 PostgreSQL 连接..."
if docker exec ai-infra-mock-postgres psql -U test_user -d test_db -c "SELECT version();" >/dev/null 2>&1; then
    echo "✓ PostgreSQL 连接正常"
else
    echo "✗ PostgreSQL 连接失败"
fi

# 测试 Redis
echo "测试 Redis 连接..."
if docker exec ai-infra-mock-redis redis-cli -a test_redis_pass ping >/dev/null 2>&1; then
    echo "✓ Redis 连接正常"
else
    echo "✗ Redis 连接失败"
fi

echo "=== 测试完成 ==="
EOF

    chmod +x "$mock_dir/test-connectivity.sh"
    
    print_success "✓ Mock 测试环境配置已创建"
    print_info "  配置文件: $mock_dir/docker-compose-mock.yml"
    print_info "  测试脚本: $mock_dir/test-connectivity.sh"
    echo
    print_info "启动基础 Mock 环境:"
    print_info "  cd $mock_dir && docker-compose -f docker-compose-mock.yml up -d"
    print_info "启动包含 backend 的完整环境:"
    print_info "  cd $mock_dir && docker-compose -f docker-compose-mock.yml --profile backend-test up -d"
    print_info "停止 Mock 环境:"
    print_info "  cd $mock_dir && docker-compose -f docker-compose-mock.yml down"
}

# 运行简化的 Mock 测试（仅验证脚本功能）
run_mock_tests() {
    local tag="${1:-$DEFAULT_IMAGE_TAG}"
    local action="${2:-up}"
    
    local mock_dir="$SCRIPT_DIR/test/mock-data"
    
    if [[ ! -f "$mock_dir/docker-compose-mock.yml" ]]; then
        print_warning "Mock 环境配置不存在，正在创建..."
        setup_mock_environment "$tag"
    fi
    
    print_info "=========================================="
    case "$action" in
        "up"|"start")
            print_info "启动 Mock 测试环境（脚本功能验证）"
            ;;
        "down"|"stop")
            print_info "停止 Mock 测试环境"
            ;;
        "restart")
            print_info "重启 Mock 测试环境"
            ;;
        "test")
            print_info "运行 Mock 环境连接测试"
            ;;
        *)
            print_error "无效的操作: $action"
            print_info "支持的操作: up/start, down/stop, restart, test"
            return 1
            ;;
    esac
    print_info "=========================================="
    echo
    
    cd "$mock_dir"
    
    case "$action" in
        "up"|"start")
            # 检查 backend 镜像是否存在
            local has_backend=false
            if docker image inspect "ai-infra-backend:$tag" >/dev/null 2>&1; then
                has_backend=true
                print_info "检测到 backend 镜像，将启动完整测试环境"
                if docker-compose -f docker-compose-mock.yml --profile backend-test up -d; then
                    print_success "✓ Mock 环境（包含 backend）启动成功"
                    print_info "服务访问地址:"
                    print_info "  Backend API: http://localhost:18080"
                    print_info "  PostgreSQL: localhost:15432 (test_user/test_pass)"
                    print_info "  Redis: localhost:16379 (test_redis_pass)"
                else
                    print_error "✗ Mock 环境启动失败"
                    return 1
                fi
            else
                print_info "未检测到 backend 镜像，启动基础环境"
                if docker-compose -f docker-compose-mock.yml up -d mock-postgres mock-redis; then
                    print_success "✓ Mock 基础环境启动成功"
                    print_info "服务访问地址:"
                    print_info "  PostgreSQL: localhost:15432 (test_user/test_pass)"
                    print_info "  Redis: localhost:16379 (test_redis_pass)"
                else
                    print_error "✗ Mock 环境启动失败"
                    return 1
                fi
            fi
            
            # 等待服务启动
            print_info "等待服务启动..."
            sleep 5
            
            # 运行连接测试
            if [[ -x "./test-connectivity.sh" ]]; then
                print_info "运行连接测试..."
                ./test-connectivity.sh
            fi
            ;;
            
        "down"|"stop")
            if docker-compose -f docker-compose-mock.yml down; then
                print_success "✓ Mock 环境停止成功"
            else
                print_error "✗ Mock 环境停止失败"
                return 1
            fi
            ;;
            
        "restart")
            print_info "停止现有环境..."
            docker-compose -f docker-compose-mock.yml down
            sleep 2
            print_info "启动环境..."
            run_mock_tests "$tag" "up"
            ;;
            
        "test")
            if [[ -x "./test-connectivity.sh" ]]; then
                ./test-connectivity.sh
            else
                print_error "测试脚本不存在或不可执行"
                return 1
            fi
            ;;
    esac
    
    cd "$SCRIPT_DIR"
}

# 列出所有服务和镜像
list_services() {
    local tag="${1:-$DEFAULT_IMAGE_TAG}"
    local registry="${2:-}"
    
    print_info "=========================================="
    print_info "AI-Infra 服务清单"
    print_info "=========================================="
    print_info "镜像标签: $tag"
    if [[ -n "$registry" ]]; then
        print_info "目标仓库: $registry"
    else
        print_info "目标仓库: 本地构建"
    fi
    echo
    
    local service_count=0
    for service in $SRC_SERVICES; do
        service_count=$((service_count + 1))
    done
    
    print_info "📦 源码服务 ($service_count 个):"
    for service in $SRC_SERVICES; do
        local service_path=$(get_service_path "$service")
        local dockerfile_path="$service_path/Dockerfile"
        local base_image="ai-infra-${service}:${tag}"
        local target_image="$base_image"
        
        if [[ -n "$registry" ]]; then
            target_image=$(get_private_image_name "$base_image" "$registry")
        fi
        
        # 检查 Dockerfile 是否存在
        local status="✅"
        if [[ ! -f "$SCRIPT_DIR/$dockerfile_path" ]]; then
            status="❌"
        fi
        
        echo "  $status $service"
        echo "       Dockerfile: $dockerfile_path"
        echo "       镜像名称: $target_image"
        echo
    done
    
    print_info "=========================================="
}

# 清理本地镜像
clean_images() {
    local tag="${1:-$DEFAULT_IMAGE_TAG}"
    local force="${2:-false}"
    
    print_info "=========================================="
    print_info "清理本地 AI-Infra 镜像"
    print_info "=========================================="
    print_info "目标标签: $tag"
    echo
    
    local images_to_clean=()
    
    # 收集需要清理的镜像
    for service in $SRC_SERVICES; do
        local image="ai-infra-${service}:${tag}"
        if docker image inspect "$image" >/dev/null 2>&1; then
            images_to_clean+=("$image")
        fi
    done
    
    if [[ ${#images_to_clean[@]} -eq 0 ]]; then
        print_info "没有找到需要清理的镜像"
        return 0
    fi
    
    print_info "找到 ${#images_to_clean[@]} 个镜像:"
    for image in "${images_to_clean[@]}"; do
        echo "  • $image"
    done
    echo
    
    if [[ "$force" != "true" ]]; then
        read -p "确认删除这些镜像? (y/N): " confirm
        if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
            print_info "已取消清理操作"
            return 0
        fi
    fi
    
    # 删除镜像
    local success_count=0
    for image in "${images_to_clean[@]}"; do
        if docker rmi "$image" 2>/dev/null; then
            print_success "✓ 已删除: $image"
            success_count=$((success_count + 1))
        else
            print_error "✗ 删除失败: $image"
        fi
    done
    
    print_success "清理完成: $success_count/${#images_to_clean[@]} 成功"
}

# 显示帮助信息
show_help() {
    echo "AI Infrastructure Matrix - 精简构建脚本 v$VERSION"
    echo
    echo "专注于 src/ 目录下的 Dockerfile 构建，支持依赖镜像管理和 Mock 测试"
    echo
    echo "用法:"
    echo "  $0 <命令> [参数...]"
    echo
    echo "源码服务命令:"
    echo "  list [tag] [registry]           - 列出所有服务和镜像"
    echo "  build <service> [tag] [registry] - 构建单个服务"
    echo "  build-all [tag] [registry]      - 构建所有服务"
    echo "  push <service> <registry> [tag] - 推送单个服务"
    echo "  push-all <registry> [tag]       - 推送所有服务"
    echo "  build-push <registry> [tag]     - 一键构建并推送所有服务"
    echo
    echo "依赖镜像命令:"
    echo "  deps-pull <registry> [tag]      - 拉取并标记依赖镜像"
    echo "  deps-push <registry> [tag]      - 推送依赖镜像"
    echo "  deps-all <registry> [tag]       - 拉取、标记并推送所有依赖镜像"
    echo
    echo "生产环境命令:"
    echo "  prod-generate <registry> [tag]  - 生成生产环境配置文件（使用内部镜像）"
    echo "  prod-up <registry> [tag]        - 启动生产环境"
    echo "  prod-down                       - 停止生产环境"
    echo "  prod-restart <registry> [tag]   - 重启生产环境"
    echo "  prod-status                     - 查看生产环境状态"
    echo "  prod-logs [service] [--follow]  - 查看生产环境日志"
    echo "  注意: 首次使用生产环境前请运行: ./scripts/generate-prod-passwords.sh"
    echo
    echo "Mock 测试命令:"
    echo "  mock-setup [tag]               - 创建 Mock 数据测试环境配置"
    echo "  mock-up [tag]                  - 启动 Mock 测试环境"
    echo "  mock-down                      - 停止 Mock 测试环境"
    echo "  mock-restart [tag]             - 重启 Mock 测试环境"
    echo
    echo "工具命令:"
    echo "  clean [tag] [--force]          - 清理本地镜像"
    echo "  version                        - 显示版本信息"
    echo "  help                           - 显示此帮助信息"
    echo
    echo "服务列表 (源码):"
    for service in $SRC_SERVICES; do
        local service_path=$(get_service_path "$service")
        echo "  • $service ($service_path)"
    done
    echo
    echo "依赖镜像列表:"
    local dependency_images
    dependency_images=$(collect_dependency_images)
    for dep_image in $dependency_images; do
        echo "  • $dep_image"
    done
    echo
    echo "示例:"
    echo "  # 源码服务操作"
    echo "  $0 list                         # 列出所有服务"
    echo "  $0 build backend               # 构建 backend 服务"
    echo "  $0 build-all v0.3.5            # 构建所有服务，标签 v0.3.5"
    echo "  $0 build-push registry.local/ai-infra v0.3.5"
    echo "                                  # 构建并推送到私有仓库"
    echo
    echo "  # 依赖镜像操作"
    echo "  $0 deps-pull registry.local/ai-infra latest"
    echo "                                  # 拉取并标记依赖镜像"
    echo "  $0 deps-push registry.local/ai-infra latest"
    echo "                                  # 推送依赖镜像"
    echo "  $0 deps-all registry.local/ai-infra v0.3.5"
    echo "                                  # 完整依赖镜像操作"
    echo
    echo "  # 生产环境操作"
    echo "  $0 prod-generate registry.local/ai-infra v0.3.5"
    echo "                                  # 生成生产环境配置"
    echo "  $0 prod-up registry.local/ai-infra v0.3.5"
    echo "                                  # 启动生产环境"
    echo "  $0 prod-down                   # 停止生产环境"
    echo "  $0 prod-status                 # 查看生产环境状态"
    echo "  $0 prod-logs backend --follow   # 实时查看backend服务日志"
    echo
    echo "  # Mock 测试操作"
    echo "  $0 mock-setup v0.3.5           # 创建 Mock 环境配置"
    echo "  $0 mock-up v0.3.5              # 启动 Mock 测试环境"
    echo "  $0 mock-down                   # 停止 Mock 测试环境"
    echo "  $0 clean v0.3.5 --force        # 强制清理标签为 v0.3.5 的镜像"
    echo
    echo "注意:"
    echo "  • 默认镜像标签: $DEFAULT_IMAGE_TAG"
    echo "  • 支持 Harbor 和传统 registry 格式"
    echo "  • 构建上下文固定为项目根目录"
}

# 主函数
main() {
    # 早期Docker Compose兼容性检查
    if [[ "${1:-}" != "version" && "${1:-}" != "help" && "${1:-}" != "-h" && "${1:-}" != "--help" ]]; then
        if ! check_compose_compatibility; then
            exit 1
        fi
    fi
    
    case "${1:-help}" in
        "list")
            list_services "${2:-$DEFAULT_IMAGE_TAG}" "$3"
            ;;
            
        "build")
            if [[ -z "$2" ]]; then
                print_error "请指定要构建的服务"
                print_info "可用服务: $SRC_SERVICES"
                exit 1
            fi
            build_service "$2" "${3:-$DEFAULT_IMAGE_TAG}" "$4"
            ;;
            
        "build-all")
            build_all_services "${2:-$DEFAULT_IMAGE_TAG}" "$3"
            ;;
            
        "push")
            if [[ -z "$2" ]]; then
                print_error "请指定要推送的服务"
                print_info "可用服务: $SRC_SERVICES"
                exit 1
            fi
            if [[ -z "$3" ]]; then
                print_error "请指定目标 registry"
                exit 1
            fi
            push_service "$2" "${4:-$DEFAULT_IMAGE_TAG}" "$3"
            ;;
            
        "push-all")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                exit 1
            fi
            push_all_services "${3:-$DEFAULT_IMAGE_TAG}" "$2"
            ;;
            
        "build-push")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                exit 1
            fi
            build_and_push_all "${3:-$DEFAULT_IMAGE_TAG}" "$2"
            ;;
            
        # 依赖镜像管理命令
        "deps-pull")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                exit 1
            fi
            pull_and_tag_dependencies "$2" "${3:-latest}"
            ;;
            
        "deps-push")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                exit 1
            fi
            push_dependencies "$2" "${3:-latest}"
            ;;
            
        "deps-all")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                exit 1
            fi
            local deps_tag="${3:-latest}"
            print_info "执行完整的依赖镜像操作..."
            if pull_and_tag_dependencies "$2" "$deps_tag"; then
                push_dependencies "$2" "$deps_tag"
            else
                print_error "依赖镜像拉取失败，停止推送操作"
                exit 1
            fi
            ;;
            
        # 生产环境部署命令
        "prod-generate")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                exit 1
            fi
            generate_production_config "$2" "${3:-$DEFAULT_IMAGE_TAG}"
            ;;
            
        "prod-up")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                exit 1
            fi
            start_production "$2" "${3:-$DEFAULT_IMAGE_TAG}"
            ;;
            
        "prod-down")
            stop_production
            ;;
            
        "prod-restart")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                exit 1
            fi
            restart_production "$2" "${3:-$DEFAULT_IMAGE_TAG}"
            ;;
            
        "prod-status")
            production_status
            ;;
            
        "prod-logs")
            local follow="false"
            if [[ "$3" == "--follow" || "$3" == "-f" ]]; then
                follow="true"
            fi
            production_logs "$2" "$follow"
            ;;
            
        # Mock 测试环境命令
        "mock-setup")
            setup_mock_environment "${2:-$DEFAULT_IMAGE_TAG}"
            ;;
            
        "mock-up"|"mock-start")
            run_mock_tests "${2:-$DEFAULT_IMAGE_TAG}" "up"
            ;;
            
        "mock-down"|"mock-stop")
            run_mock_tests "${2:-$DEFAULT_IMAGE_TAG}" "down"
            ;;
            
        "mock-restart")
            run_mock_tests "${2:-$DEFAULT_IMAGE_TAG}" "restart"
            ;;
            
        "mock-test")
            run_mock_tests "${2:-$DEFAULT_IMAGE_TAG}" "test"
            ;;
            
        "clean")
            local force="false"
            if [[ "$3" == "--force" ]]; then
                force="true"
            fi
            clean_images "${2:-$DEFAULT_IMAGE_TAG}" "$force"
            ;;
            
        "version")
            echo "AI Infrastructure Matrix Build Script"
            echo "Version: $VERSION"
            echo "Default Tag: $DEFAULT_IMAGE_TAG"
            echo "Services: $SRC_SERVICES"
            echo
            echo "Dependency Images:"
            for dep in $DEPENDENCY_IMAGES; do
                echo "  • $dep"
            done
            ;;
            
        "help"|"-h"|"--help")
            show_help
            ;;
            
        *)
            print_error "未知命令: $1"
            print_info "使用 '$0 help' 查看可用命令"
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"
