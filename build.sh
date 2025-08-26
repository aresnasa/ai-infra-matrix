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
VERSION="   1.0.0"
CONFIG_FILE="$SCRIPT_DIR/config.toml"
OS_TYPE=$(detect_os)
FORCE_REBUILD=false  # 强制重新构建标志

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

# 获取所有依赖镜像（包含测试工具）
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

# 获取生产环境依赖镜像（移除测试工具）
get_production_dependencies() {
    if [[ ! -f "$CONFIG_FILE" ]]; then
        echo "postgres:15-alpine redis:7-alpine tecnativa/tcp-proxy nginx:1.27-alpine quay.io/minio/minio:latest"
        return
    fi
    
    awk -F' *= *' '
        /^\[dependencies\]/ { in_dependencies = 1; next }
        /^\[/ { in_dependencies = 0; next }
        in_dependencies && NF > 1 {
            gsub(/^"/, "", $2)
            gsub(/"$/, "", $2)
            # 排除测试工具和LDAP服务
            if ($2 !~ /phpldapadmin/ && $2 !~ /redisinsight/ && $2 !~ /openldap/) {
                print $2
            }
        }
    ' "$CONFIG_FILE" | tr '\n' ' '
}

# 初始化配置
DEFAULT_IMAGE_TAG=$(read_config "project" "version")
[[ -z "$DEFAULT_IMAGE_TAG" ]] && DEFAULT_IMAGE_TAG="v0.3.5"

# 动态加载服务和依赖配置
SRC_SERVICES=$(get_all_services | tr '\n' ' ')
DEPENDENCY_IMAGES=$(get_all_dependencies | tr '\n' ' ')

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
            "singleuser") echo "src/singleuser" ;;
            "gitea") echo "src/gitea" ;;
            "backend-init") echo "src/backend" ;;  # backend-init 使用 backend 的 Dockerfile
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
# 环境变量管理函数
# ==========================================

# 检测并确定唯一的环境文件
detect_env_file() {
    local env_file=""
    
    # 优先级检查：.env.prod > .env > .env.example
    if [[ -f ".env.prod" ]]; then
        env_file=".env.prod"
        echo "使用生产环境配置: $env_file" >&2
    elif [[ -f ".env" ]]; then
        env_file=".env"
        echo "使用开发环境配置: $env_file" >&2
    elif [[ -f ".env.example" ]]; then
        echo "未找到环境配置文件，从模板创建..." >&2
        cp ".env.example" ".env"
        env_file=".env"
        echo "✓ 从.env.example创建了.env文件" >&2
    else
        echo "错误: 未找到任何环境配置文件（.env.prod, .env, .env.example）" >&2
        return 1
    fi
    
    echo "$env_file"
    return 0
}

# 验证环境文件有效性
validate_env_file() {
    local env_file="$1"
    
    if [[ ! -f "$env_file" ]]; then
        echo "错误: 环境文件不存在: $env_file" >&2
        return 1
    fi
    
    # 检查关键变量是否存在
    local required_vars=("IMAGE_TAG" "COMPOSE_PROJECT_NAME")
    local missing_vars=()
    
    for var in "${required_vars[@]}"; do
        if ! grep -q "^${var}=" "$env_file" 2>/dev/null; then
            missing_vars+=("$var")
        fi
    done
    
    if [[ ${#missing_vars[@]} -gt 0 ]]; then
        echo "警告: 环境文件 $env_file 缺少必要变量: ${missing_vars[*]}" >&2
        echo "建议检查并补充这些变量" >&2
    fi
    
    return 0
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

# 根据镜像映射配置获取私有镜像名称和版本
# 支持latest标签到git版本的映射
get_mapped_private_image() {
    local original_image="$1"
    local registry="$2"
    local target_tag="${3:-v0.3.5}"  # 默认目标git版本
    local mapping_file="$SCRIPT_DIR/config/image-mapping.conf"
    
    if [[ -z "$registry" ]]; then
        echo "$original_image"
        return 0
    fi
    
    # 如果映射文件不存在，使用原有逻辑
    if [[ ! -f "$mapping_file" ]]; then
        get_private_image_name "$original_image" "$registry"
        return 0
    fi
    
    # 标准化镜像名称（移除tag用于匹配）
    local image_base=""
    local original_tag=""
    
    if [[ "$original_image" == *":"* ]]; then
        image_base="${original_image%%:*}"
        original_tag="${original_image##*:}"
    else
        image_base="$original_image"
        original_tag="latest"
    fi
    
    # 读取映射配置
    local mapped_project=""
    local mapped_version=""
    local found_mapping=false
    
    while IFS='|' read -r pattern project version special; do
        # 跳过注释和空行
        [[ "$pattern" =~ ^[[:space:]]*# ]] && continue
        [[ -z "$pattern" ]] && continue
        
        # 检查是否匹配（支持精确匹配和基础名匹配）
        if [[ "$original_image" == "$pattern" ]] || 
           [[ "$image_base" == "$pattern" ]] ||
           [[ "$image_base:$original_tag" == "$pattern" ]]; then
            mapped_project="$project"
            mapped_version="$version"
            found_mapping=true
            break
        fi
    done < "$mapping_file"
    
    if [[ "$found_mapping" == "true" ]]; then
        # 处理特殊变量替换
        if [[ "$mapped_version" == *"\${TARGET_TAG}"* ]]; then
            # 项目镜像，使用传入的target_tag
            mapped_version="${mapped_version//\${TARGET_TAG}/$target_tag}"
        elif [[ "$mapped_version" == *"\${IMAGE_TAG}"* ]]; then
            # 兼容旧格式
            mapped_version="${mapped_version//\${IMAGE_TAG}/$target_tag}"
        fi
        
        # 提取原始镜像的简短名称（不含namespace）
        local simple_name=""
        if [[ "$image_base" == *"/"* ]]; then
            # 处理带namespace的镜像，如 tecnativa/tcp-proxy -> tcp-proxy
            simple_name="${image_base##*/}"
        else
            # 直接使用镜像名，如 postgres -> postgres
            simple_name="$image_base"
        fi
        
        # 构建统一的 aiharbor.msxf.local/aihpc/servicename:version 格式
        local final_image="${registry}/${simple_name}:${mapped_version}"
        
        echo "$final_image"
    else
        # 未找到映射，使用原有逻辑
        get_private_image_name "$original_image" "$registry"
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
    
    # 检查镜像是否已存在
    if [[ "$FORCE_REBUILD" == "false" ]] && docker image inspect "$target_image" >/dev/null 2>&1; then
        print_success "  ✓ 镜像已存在，跳过构建: $target_image"
        
        # 如果指定了registry，确保本地别名也存在
        if [[ -n "$registry" ]] && [[ "$target_image" != "$base_image" ]]; then
            if ! docker image inspect "$base_image" >/dev/null 2>&1; then
                if docker tag "$target_image" "$base_image"; then
                    print_info "  ✓ 创建本地别名: $base_image"
                fi
            fi
        fi
        
        return 0
    fi
    
    # 构建镜像
    print_info "  → 正在构建镜像..."
    local build_context="$SCRIPT_DIR/$service_path"
    local dockerfile_name="Dockerfile"
    
    # 特殊处理：backend 和 backend-init 需要项目根目录作为构建上下文
    if [[ "$service" == "backend" ]] || [[ "$service" == "backend-init" ]]; then
        # 对于 backend-init，需要指定特殊的 target
        local target_arg=""
        if [[ "$service" == "backend-init" ]]; then
            target_arg="--target backend-init"
        fi
        
        if docker build -f "$dockerfile_path" $target_arg -t "$target_image" "$SCRIPT_DIR"; then
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
    # 检查是否在服务目录中构建
    elif [[ -f "$build_context/$dockerfile_name" ]]; then
        # 切换到服务目录进行构建
        cd "$build_context"
        if docker build -f "$dockerfile_name" -t "$target_image" .; then
            cd "$SCRIPT_DIR"  # 返回原目录
            print_success "✓ 构建成功: $target_image"
            
            # 如果指定了registry，同时创建本地别名
            if [[ -n "$registry" ]] && [[ "$target_image" != "$base_image" ]]; then
                if docker tag "$target_image" "$base_image"; then
                    print_info "  ✓ 本地别名: $base_image"
                fi
            fi
            
            return 0
        else
            cd "$SCRIPT_DIR"  # 返回原目录
            print_error "✗ 构建失败: $target_image"
            return 1
        fi
    else
        # 后备方案：使用项目根目录作为构建上下文
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
    
    # 获取所有服务（包括原扩展组件）
    local all_services="$SRC_SERVICES"
    
    # 计算服务总数
    for service in $all_services; do
        total_count=$((total_count + 1))
    done
    
    # 构建所有服务
    for service in $all_services; do
        print_info "构建服务: $service"
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
    print_info "源镜像标签: $tag (如果为latest则会映射到v0.3.5)"
    
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
        
        # 使用新的映射机制生成目标镜像名
        local target_image
        target_image=$(get_mapped_private_image "$dep_image" "$registry" "$tag")
        
        # 检查目标镜像是否已存在
        if [[ "$FORCE_REBUILD" == "false" ]] && docker image inspect "$target_image" >/dev/null 2>&1; then
            print_success "  ✓ 镜像已存在，跳过: $target_image"
            success_count=$((success_count + 1))
            continue
        fi
        
        # 检查原始镜像是否已存在本地
        if docker image inspect "$dep_image" >/dev/null 2>&1; then
            print_success "  ✓ 本地镜像已存在: $dep_image"
        else
            # 拉取原始镜像
            print_info "  → 正在拉取镜像: $dep_image"
            if ! docker pull "$dep_image"; then
                print_error "  ✗ 拉取失败: $dep_image"
                failed_deps+=("$dep_image")
                continue
            fi
            print_success "  ✓ 拉取成功: $dep_image"
        fi
        
        # 标记镜像
        if docker tag "$dep_image" "$target_image"; then
            print_success "  ✓ 标记成功: $target_image"
            success_count=$((success_count + 1))
        else
            print_error "  ✗ 标记失败: $target_image"
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
    print_info "源镜像标签: $tag (如果为latest则会映射到v0.3.5)"
    
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
        
        # 使用新的映射机制生成目标镜像名
        local target_image
        target_image=$(get_mapped_private_image "$dep_image" "$registry" "$tag")
        
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
# 生产环境依赖镜像处理功能
# ==========================================

# 拉取并标记生产环境依赖镜像（排除测试工具）
pull_and_tag_production_dependencies() {
    local registry="$1"
    local tag="${2:-latest}"
    
    if [[ -z "$registry" ]]; then
        print_error "需要指定 registry"
        return 1
    fi
    
    print_info "=========================================="
    print_info "拉取并标记生产环境依赖镜像到 $registry"
    print_info "=========================================="
    print_info "源镜像标签: $tag (如果为latest则会映射到v0.3.5)"
    
    # 使用生产环境依赖镜像列表
    local dependency_images
    dependency_images=$(get_production_dependencies | tr '\n' ' ')
    print_info "收集到生产环境依赖镜像: $dependency_images"
    echo
    
    local success_count=0
    local total_count=0
    local failed_deps=()
    
    for dep_image in $dependency_images; do
        if [[ -z "$dep_image" ]]; then
            continue
        fi
        
        ((total_count++))
        
        # 获取目标镜像名称
        local target_image
        target_image=$(get_mapped_private_image "$dep_image" "$registry" "$tag")
        
        # 检查镜像是否已存在
        if docker image inspect "$target_image" >/dev/null 2>&1; then
            print_success "  ✓ 镜像已存在，跳过: $target_image"
            ((success_count++))
            continue
        fi
        
        print_info "处理生产环境依赖镜像: $dep_image"
        
        # 拉取原始镜像
        if ! docker pull "$dep_image"; then
            print_error "  ✗ 拉取失败: $dep_image"
            failed_deps+=("$dep_image")
            continue
        fi
        
        # 标记为目标镜像
        if ! docker tag "$dep_image" "$target_image"; then
            print_error "  ✗ 标记失败: $dep_image -> $target_image"
            failed_deps+=("$dep_image")
            continue
        fi
        
        print_success "  ✓ 处理成功: $dep_image -> $target_image"
        ((success_count++))
    done
    you y
    print_info "=========================================="
    print_success "生产环境依赖镜像处理完成: $success_count/$total_count 成功"
    
    if [[ ${#failed_deps[@]} -gt 0 ]]; then
        print_warning "失败的依赖镜像: ${failed_deps[*]}"
        return 1
    else
        print_success "🎉 所有生产环境依赖镜像处理成功！"
        return 0
    fi
}

# 推送生产环境依赖镜像
push_production_dependencies() {
    local registry="$1"
    local tag="${2:-latest}"
    
    if [[ -z "$registry" ]]; then
        print_error "需要指定 registry"
        return 1
    fi
    
    print_info "=========================================="
    print_info "推送生产环境依赖镜像到 $registry"
    print_info "=========================================="
    print_info "源镜像标签: $tag (如果为latest则会映射到v0.3.5)"
    
    # 使用生产环境依赖镜像列表
    local dependency_images
    dependency_images=$(get_production_dependencies | tr '\n' ' ')
    print_info "收集到生产环境依赖镜像: $dependency_images"
    echo
    
    local success_count=0
    local total_count=0
    local failed_deps=()
    
    for dep_image in $dependency_images; do
        if [[ -z "$dep_image" ]]; then
            continue
        fi
        
        ((total_count++))
        
        # 获取目标镜像名称
        local target_image
        target_image=$(get_mapped_private_image "$dep_image" "$registry" "$tag")
        
        print_info "推送生产环境依赖镜像: $target_image"
        
        if docker push "$target_image"; then
            print_success "  ✓ 推送成功: $target_image"
            ((success_count++))
        else
            print_error "  ✗ 推送失败: $target_image"
            failed_deps+=("$target_image")
        fi
    done
    
    print_info "=========================================="
    print_success "生产环境依赖镜像推送完成: $success_count/$total_count 成功"
    
    if [[ ${#failed_deps[@]} -gt 0 ]]; then
        print_warning "失败的依赖镜像: ${failed_deps[*]}"
        return 1
    else
        print_success "🎉 所有生产环境依赖镜像推送成功！"
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
    
    # 检测并确保环境变量文件存在
    local env_file
    env_file=$(detect_env_file)
    if [[ $? -ne 0 ]]; then
        return 1
    fi
    
    # 验证环境文件
    if ! validate_env_file "$env_file"; then
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
    
    # 1. 使用映射配置更新基础镜像和第三方镜像，项目镜像保持原有逻辑
    print_info "使用映射配置更新基础镜像和第三方镜像... (OS: $OS_TYPE)"
    
    # 首先处理项目镜像（保持原有逻辑）
    print_info "处理项目镜像..."
    if [[ "$OS_TYPE" == "macOS" ]]; then
        sed -i.bak "s|ghcr.io/aresnasa/ai-infra-matrix|${registry}|g" "$output_file"
        sed -i.bak "s|image: ai-infra-|image: ${registry}/ai-infra-|g" "$output_file"
    else
        sed -i "s|ghcr.io/aresnasa/ai-infra-matrix|${registry}|g" "$output_file"
        sed -i "s|image: ai-infra-|image: ${registry}/ai-infra-|g" "$output_file"
    fi
    
    # 然后处理基础镜像和第三方镜像（使用映射配置）
    print_info "处理基础镜像和第三方镜像..."
    declare -a base_images_to_replace=(
        "postgres:15-alpine"
        "redis:7-alpine" 
        "nginx:1.27-alpine"
        "tecnativa/tcp-proxy:latest"
        "tecnativa/tcp-proxy"
        "quay.io/minio/minio:latest"
        "minio/minio:latest"
        "minio/minio"
    )
    
    # 使用映射配置替换基础镜像
    for original_image in "${base_images_to_replace[@]}"; do
        # 获取映射后的镜像（使用传入的tag参数）
        local mapped_image
        mapped_image=$(get_mapped_private_image "$original_image" "$registry" "$tag")
        
        if [[ "$mapped_image" != "$original_image" ]]; then
            print_info "  映射: $original_image -> $mapped_image"
            
            # 执行替换
            if [[ "$OS_TYPE" == "macOS" ]]; then
                sed -i.bak "s|image: ${original_image}|image: ${mapped_image}|g" "$output_file"
            else
                sed -i "s|image: ${original_image}|image: ${mapped_image}|g" "$output_file"
            fi
        fi
    done
    
    # 2. 更新项目镜像的环境变量标签
    print_info "更新项目镜像环境变量标签..."
    if [[ "$OS_TYPE" == "macOS" ]]; then
        # 只更新项目镜像的环境变量标签
        sed -i.bak "s|\${IMAGE_TAG}|${tag}|g" "$output_file"
        sed -i.bak "s|\${IMAGE_TAG:-v[^}]*}|${tag}|g" "$output_file"
    else
        # 只更新项目镜像的环境变量标签
        sed -i "s|\${IMAGE_TAG}|${tag}|g" "$output_file"
        sed -i "s|\${IMAGE_TAG:-v[^}]*}|${tag}|g" "$output_file"
    fi
    
    # 3. 移除生产环境非必须服务（使用改进的处理逻辑）
    print_info "移除openldap、phpldapadmin和redisinsight服务..."
    
    # 检查是否有Python和PyYAML
    if command -v python3 >/dev/null 2>&1 && python3 -c "import yaml" 2>/dev/null; then
        # 使用Python脚本精确移除非必须服务
        if python3 fix_ldap_removal.py "$output_file" "$output_file.tmp" 2>/dev/null; then
            mv "$output_file.tmp" "$output_file"
            print_success "✓ 使用Python脚本成功移除生产环境非必须服务"
        else
            print_warning "Python脚本移除失败，使用改进的备用方案"
            # 改进的备用方案：更完整的sed和awk处理
            if [[ "$OS_TYPE" == "macOS" ]]; then
                # macOS版本 - 移除整个服务块
                sed -i.bak '/^  openldap:/,/^  [a-zA-Z]/{ /^  [a-zA-Z]/!d; }' "$output_file"
                sed -i.bak '/^  phpldapadmin:/,/^  [a-zA-Z]/{ /^  [a-zA-Z]/!d; }' "$output_file"
                sed -i.bak '/^  redisinsight:/,/^  [a-zA-Z]/{ /^  [a-zA-Z]/!d; }' "$output_file"
                sed -i.bak '/^  redis-insight:/,/^  [a-zA-Z]/{ /^  [a-zA-Z]/!d; }' "$output_file"
                sed -i.bak '/^  openldap:/d' "$output_file"
                sed -i.bak '/^  phpldapadmin:/d' "$output_file"
                sed -i.bak '/^  redisinsight:/d' "$output_file"
                sed -i.bak '/^  redis-insight:/d' "$output_file"
                
                # 移除depends_on中的openldap依赖（包括复杂格式）
                sed -i.bak '/^[[:space:]]*- openldap$/d' "$output_file"
                sed -i.bak '/LDAP_SERVER=/d' "$output_file"
                sed -i.bak '/PHPLDAPADMIN_/d' "$output_file"
            else
                # Linux版本 - 移除整个服务块
                sed -i '/^  openldap:/,/^  [a-zA-Z]/{ /^  [a-zA-Z]/!d; }' "$output_file"
                sed -i '/^  phpldapadmin:/,/^  [a-zA-Z]/{ /^  [a-zA-Z]/!d; }' "$output_file"
                sed -i '/^  redisinsight:/,/^  [a-zA-Z]/{ /^  [a-zA-Z]/!d; }' "$output_file"
                sed -i '/^  redis-insight:/,/^  [a-zA-Z]/{ /^  [a-zA-Z]/!d; }' "$output_file"
                sed -i '/^  openldap:/d' "$output_file"
                sed -i '/^  phpldapadmin:/d' "$output_file"
                sed -i '/^  redisinsight:/d' "$output_file"
                sed -i '/^  redis-insight:/d' "$output_file"
                
                # 移除depends_on中的openldap依赖（包括复杂格式）
                sed -i '/^[[:space:]]*- openldap$/d' "$output_file"
                sed -i '/LDAP_SERVER=/d' "$output_file"
                sed -i '/PHPLDAPADMIN_/d' "$output_file"
            fi
            
            # 使用awk清理复杂的多行openldap依赖块（适用于所有系统）
            awk '
            BEGIN { 
                in_openldap_dep = 0
                print_line = 1
            }
            {
                # 检测openldap依赖块的开始
                if ($0 ~ /^[[:space:]]*openldap:[[:space:]]*$/) {
                    in_openldap_dep = 1
                    print_line = 0
                }
                # 检测openldap依赖块的结束
                else if (in_openldap_dep && $0 ~ /^[[:space:]]*condition: service_healthy[[:space:]]*$/) {
                    in_openldap_dep = 0
                    print_line = 0
                }
                # 检测下一个服务或配置块（重置状态）
                else if (in_openldap_dep && $0 ~ /^[[:space:]]*[a-zA-Z][a-zA-Z0-9_-]*:[[:space:]]*/) {
                    in_openldap_dep = 0
                    print_line = 1
                }
                # 普通情况
                else {
                    print_line = 1
                }
                
                # 只打印非openldap依赖的行
                if (print_line && !in_openldap_dep) {
                    print $0
                }
            }
            ' "$output_file" > "$output_file.tmp" && mv "$output_file.tmp" "$output_file"
        fi
    else
        print_warning "未安装PyYAML，使用改进的sed方案移除生产环境非必须服务"
        # 改进的纯sed和awk方案
        if [[ "$OS_TYPE" == "macOS" ]]; then
            # macOS版本
            sed -i.bak '/^  openldap:/,/^  [a-zA-Z]/{ /^  [a-zA-Z]/!d; }' "$output_file"
            sed -i.bak '/^  phpldapadmin:/,/^  [a-zA-Z]/{ /^  [a-zA-Z]/!d; }' "$output_file"
            sed -i.bak '/^  redisinsight:/,/^  [a-zA-Z]/{ /^  [a-zA-Z]/!d; }' "$output_file"
            sed -i.bak '/^  redis-insight:/,/^  [a-zA-Z]/{ /^  [a-zA-Z]/!d; }' "$output_file"
            sed -i.bak '/^  openldap:/d' "$output_file"
            sed -i.bak '/^  phpldapadmin:/d' "$output_file"
            sed -i.bak '/^  redisinsight:/d' "$output_file"
            sed -i.bak '/^  redis-insight:/d' "$output_file"
            
            # 移除简单的依赖和环境变量
            sed -i.bak '/^[[:space:]]*- openldap$/d' "$output_file"
            sed -i.bak '/LDAP_SERVER=/d' "$output_file"
            sed -i.bak '/PHPLDAPADMIN_/d' "$output_file"
        else
            # Linux版本
            sed -i '/^  openldap:/,/^  [a-zA-Z]/{ /^  [a-zA-Z]/!d; }' "$output_file"
            sed -i '/^  phpldapadmin:/,/^  [a-zA-Z]/{ /^  [a-zA-Z]/!d; }' "$output_file"
            sed -i '/^  redisinsight:/,/^  [a-zA-Z]/{ /^  [a-zA-Z]/!d; }' "$output_file"
            sed -i '/^  redis-insight:/,/^  [a-zA-Z]/{ /^  [a-zA-Z]/!d; }' "$output_file"
            sed -i '/^  openldap:/d' "$output_file"
            sed -i '/^  phpldapadmin:/d' "$output_file"
            sed -i '/^  redisinsight:/d' "$output_file"
            sed -i '/^  redis-insight:/d' "$output_file"
            
            # 移除简单的依赖和环境变量
            sed -i '/^[[:space:]]*- openldap$/d' "$output_file"
            sed -i '/LDAP_SERVER=/d' "$output_file"
            sed -i '/PHPLDAPADMIN_/d' "$output_file"
        fi
        
        # 使用awk清理复杂的多行openldap依赖块
        awk '
        BEGIN { 
            in_openldap_dep = 0
            print_line = 1
        }
        {
            # 检测openldap依赖块的开始
            if ($0 ~ /^[[:space:]]*openldap:[[:space:]]*$/) {
                in_openldap_dep = 1
                print_line = 0
            }
            # 检测openldap依赖块的结束
            else if (in_openldap_dep && $0 ~ /^[[:space:]]*condition: service_healthy[[:space:]]*$/) {
                in_openldap_dep = 0
                print_line = 0
            }
            # 检测下一个服务或配置块（重置状态）
            else if (in_openldap_dep && $0 ~ /^[[:space:]]*[a-zA-Z][a-zA-Z0-9_-]*:[[:space:]]*/) {
                in_openldap_dep = 0
                print_line = 1
            }
            # 普通情况
            else {
                print_line = 1
            }
            
            # 只打印非openldap依赖的行
            if (print_line && !in_openldap_dep) {
                print $0
            }
        }
        ' "$output_file" > "$output_file.tmp" && mv "$output_file.tmp" "$output_file"
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
    print_info "  1. 请确保所有依赖镜像已推送到内部registry (使用 deps-prod 命令)"
    print_info "  2. 请确保所有源码服务镜像已推送到内部registry (使用 build-push 命令)"
    print_info "  3. 生产环境已移除LDAP、phpldapadmin和redisinsight服务，服务可独立启动"
    print_info "  4. 请检查生成的配置文件并根据需要调整环境变量"
    print_info "  5. 当前使用环境文件: $env_file"
    echo
    
    return 0
}


# 启动生产环境
start_production() {
    local registry="$1"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    local force_local="${3:-false}"  # 新增参数：是否强制使用本地镜像
    local compose_file="docker-compose.prod.yml"
    
    if [[ -z "$registry" ]]; then
        print_error "请指定目标 registry"
        return 1
    fi
    
    # 检测环境文件
    local env_file
    env_file=$(detect_env_file)
    if [[ $? -ne 0 ]]; then
        return 1
    fi
    
    # 验证环境文件
    if ! validate_env_file "$env_file"; then
        return 1
    fi
    
    # 总是重新生成生产配置文件以确保使用正确的registry和tag
    print_info "生成生产配置文件 (使用 registry: $registry, tag: $tag)..."
    if ! generate_production_config "$registry" "$tag"; then
        return 1
    fi
    
    print_info "=========================================="
    print_info "启动生产环境"
    print_info "=========================================="
    print_info "配置文件: $compose_file"
    print_info "环境文件: $env_file"
    print_info "Registry: $registry"
    print_info "标签: $tag"
    if [[ "$force_local" == "true" ]]; then
        print_info "模式: 强制使用本地镜像 (跳过拉取)"
    fi
    echo
    
    # 根据 force_local 参数决定是否拉取镜像
    if [[ "$force_local" == "true" ]]; then
        print_info "跳过镜像拉取，使用本地已有镜像..."
        
        # 检查并构建缺失的镜像（包括有build配置的服务）
        print_info "检查并构建需要的镜像..."
        if ! check_and_build_missing_images "$compose_file" "$env_file" "$registry" "$tag"; then
            print_warning "部分镜像构建失败，继续尝试启动..."
        fi
    else
        print_info "拉取所有镜像..."
        if ! ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" pull; then
            print_error "镜像拉取失败"
            return 1
        fi
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

# 检查并构建缺失的镜像
check_and_build_missing_images() {
    local compose_file="$1"
    local env_file="$2"
    local registry="$3"
    local tag="$4"
    
    if [[ ! -f "$compose_file" ]]; then
        print_error "compose文件不存在: $compose_file"
        return 1
    fi
    
    print_info "分析compose文件中需要的镜像..."
    
    # 直接构建已知的关键服务（简化方案）
    local critical_services=("backend-init" "gitea" "singleuser-builder")
    local missing_count=0
    
    for service in "${critical_services[@]}"; do
        # 构造预期的镜像名
        local expected_image="${registry}/ai-infra-${service}:${tag}"
        
        # 检查镜像是否存在
        if ! docker image inspect "$expected_image" >/dev/null 2>&1; then
            print_info "缺失镜像: $expected_image"
            if build_service_if_missing "$service" "$compose_file" "$env_file"; then
                # 构建成功后标记镜像
                local local_image="ai-infra-${service}:${tag}"
                if docker image inspect "$local_image" >/dev/null 2>&1; then
                    docker tag "$local_image" "$expected_image"
                    print_success "✓ 已标记: $local_image -> $expected_image"
                fi
            else
                missing_count=$((missing_count + 1))
            fi
        else
            print_success "✓ 镜像已存在: $expected_image"
        fi
    done
    
    if [[ $missing_count -eq 0 ]]; then
        print_success "所有关键镜像都已准备就绪"
        return 0
    else
        print_warning "有 $missing_count 个关键服务构建失败"
        return 1
    fi
}

# 构建单个服务（如果缺失）
build_service_if_missing() {
    local service="$1"
    local compose_file="$2"
    local env_file="$3"
    
    print_info "尝试构建服务: $service"
    
    # 使用docker-compose构建特定服务
    if ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" build "$service" 2>/dev/null; then
        print_success "✓ 构建成功: $service"
        return 0
    else
        print_warning "✗ 构建失败: $service (可能不存在build配置)"
        return 1
    fi
}

# 停止生产环境
stop_production() {
    local compose_file="docker-compose.prod.yml"
    
    if [[ ! -f "$compose_file" ]]; then
        print_error "生产配置文件不存在: $compose_file"
        return 1
    fi
    
    # 检测环境文件
    local env_file
    env_file=$(detect_env_file)
    if [[ $? -ne 0 ]]; then
        return 1
    fi
    
    print_info "=========================================="
    print_info "停止生产环境"
    print_info "=========================================="
    print_info "使用环境文件: $env_file"
    
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
    
    if [[ ! -f "$compose_file" ]]; then
        print_error "生产配置文件不存在: $compose_file"
        return 1
    fi
    
    # 检测环境文件
    local env_file
    env_file=$(detect_env_file)
    if [[ $? -ne 0 ]]; then
        return 1
    fi
    
    print_info "=========================================="
    print_info "生产环境状态"
    print_info "=========================================="
    print_info "使用环境文件: $env_file"
    
    ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" ps
}

# 查看生产环境日志
production_logs() {
    local compose_file="docker-compose.prod.yml"
    local service="$1"
    local follow="${2:-false}"
    
    if [[ ! -f "$compose_file" ]]; then
        print_error "生产配置文件不存在: $compose_file"
        return 1
    fi
    
    # 检测环境文件
    local env_file
    env_file=$(detect_env_file)
    if [[ $? -ne 0 ]]; then
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

# ==========================================
# 镜像验证功能
# ==========================================

# 验证单个镜像是否可用
verify_image() {
    local image="$1"
    local timeout="${2:-10}"
    
    # 先尝试检查本地镜像
    if docker image inspect "$image" >/dev/null 2>&1; then
        return 0
    fi
    
    # 尝试拉取验证（用于远程镜像）
    if timeout "$timeout" docker pull "$image" >/dev/null 2>&1; then
        return 0
    fi
    
    return 1
}

# 验证私有仓库中的所有AI-Infra镜像
verify_private_images() {
    local registry="$1"
    local tag="${2:-v0.3.5}"
    
    if [[ -z "$registry" ]]; then
        print_error "使用方法: verify <registry_base> [tag]"
        print_info "示例: verify aiharbor.msxf.local/aihpc v0.3.5"
        return 1
    fi
    
    print_info "=== AI Infrastructure Matrix 镜像验证 ==="
    print_info "目标仓库: $registry"
    print_info "镜像标签: $tag"
    print_info "开始时间: $(date)"
    echo
    
    print_info "📋 Harbor项目检查："
    print_info "验证前请确保以下项目已在Harbor中创建："
    print_info "  • aihpc (主项目)"
    print_info "  • library (基础镜像)"
    print_info "  • tecnativa (第三方镜像)"
    print_info "  • redislabs (第三方镜像)"
    print_info "  • minio (第三方镜像)"
    echo
    print_info "如未创建，请参考: docs/HARBOR_PROJECT_SETUP.md"
    echo
    
    # 源码镜像列表
    local source_images=(
        "ai-infra-backend-init"
        "ai-infra-backend"
        "ai-infra-frontend"
        "ai-infra-jupyterhub"
        "ai-infra-singleuser"
        "ai-infra-saltstack"
        "ai-infra-nginx"
        "ai-infra-gitea"
    )
    
    # 基础镜像列表（从配置文件获取）
    local base_image_patterns=(
        "postgres:15-alpine"
        "redis:7-alpine"
        "nginx:1.27-alpine"
        "tecnativa/tcp-proxy:latest"
        "redislabs/redisinsight:latest"
        "quay.io/minio/minio:latest"
    )
    
    local total_images=$((${#source_images[@]} + ${#base_image_patterns[@]}))
    local success_count=0
    local failed_images=()
    
    print_info "计划验证 $total_images 个镜像"
    print_info "============================================"
    
    # 验证源码镜像
    print_info "验证源码镜像 (${#source_images[@]} 个):"
    for image_base in "${source_images[@]}"; do
        local target_image="${registry}/${image_base}:${tag}"
        
        printf "  检查: %-45s" "$target_image"
        if verify_image "$target_image" 5; then
            echo "    ✓ 可用"
            ((success_count++))
        else
            echo "    ✗ 不可用"
            failed_images+=("$target_image")
        fi
    done
    
    echo
    # 验证基础镜像
    print_info "验证基础镜像 (${#base_image_patterns[@]} 个):"
    for base_pattern in "${base_image_patterns[@]}"; do
        # 使用映射配置获取目标镜像名
        local target_image
        target_image=$(get_mapped_private_image "$base_pattern" "$registry" "$tag")
        
        printf "  检查: %-45s" "$target_image"
        if verify_image "$target_image" 5; then
            echo "    ✓ 可用"
            ((success_count++))
        else
            echo "    ✗ 不可用"
            failed_images+=("$target_image")
        fi
    done
    
    echo
    print_info "============================================"
    print_info "验证结果汇总:"
    print_info "总计镜像: $total_images"
    print_success "验证通过: $success_count"
    print_error "验证失败: $((total_images - success_count))"
    
    if [[ ${#failed_images[@]} -gt 0 ]]; then
        echo
        print_error "失败镜像列表:"
        for failed_image in "${failed_images[@]}"; do
            echo "  ✗ $failed_image"
        done
        
        echo
        print_info "建议操作:"
        print_info "1. 检查网络连接和仓库权限"
        print_info "2. 重新运行基础镜像迁移脚本:"
        print_info "   ./scripts/migrate-base-images.sh $registry"
        print_info "3. 重新构建和推送源码镜像:"
        print_info "   ./build.sh build-push $registry $tag"
        
        return 1
    else
        echo
        print_success "🎉 所有镜像验证通过！"
        return 0
    fi
}

# 快速验证关键镜像
verify_key_images() {
    local registry="$1"
    local tag="${2:-v0.3.5}"
    
    if [[ -z "$registry" ]]; then
        print_error "使用方法: verify-key <registry_base> [tag]"
        return 1
    fi
    
    print_info "=== 快速验证关键镜像 ==="
    print_info "目标仓库: $registry"
    print_info "镜像标签: $tag"
    echo
    
    # 关键服务镜像
    local key_images=(
        "ai-infra-backend"
        "ai-infra-frontend" 
        "ai-infra-jupyterhub"
        "ai-infra-nginx"
    )
    
    # 关键基础镜像
    local key_base_images=(
        "postgres:15-alpine"
        "redis:7-alpine"
    )
    
    local success_count=0
    local total_count=$((${#key_images[@]} + ${#key_base_images[@]}))
    
    print_info "验证关键服务镜像:"
    for image_base in "${key_images[@]}"; do
        local target_image="${registry}/${image_base}:${tag}"
        printf "  %-40s" "$target_image"
        
        if verify_image "$target_image" 3; then
            echo " ✓"
            ((success_count++))
        else
            echo " ✗"
        fi
    done
    
    print_info "验证关键基础镜像:"
    for base_pattern in "${key_base_images[@]}"; do
        local target_image
        target_image=$(get_mapped_private_image "$base_pattern" "$registry" "$tag")
        printf "  %-40s" "$target_image"
        
        if verify_image "$target_image" 3; then
            echo " ✓"
            ((success_count++))
        else
            echo " ✗"
        fi
    done
    
    echo
    if [[ $success_count -eq $total_count ]]; then
        print_success "🎉 所有关键镜像验证通过 ($success_count/$total_count)"
        return 0
    else
        print_warning "⚠ 部分关键镜像验证失败 ($success_count/$total_count)"
        return 1
    fi
}

# ==========================================
# 清理功能
# ==========================================

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

# 清理所有镜像（包括依赖镜像）
clean_all_images() {
    local tag="${1:-$DEFAULT_IMAGE_TAG}"
    local force="${2:-false}"
    
    print_info "=========================================="
    print_info "清理所有 AI-Infra 相关镜像"
    print_info "=========================================="
    print_info "目标标签: $tag"
    echo
    
    local images_to_clean=()
    
    # 收集AI-Infra源码服务镜像
    print_info "收集源码服务镜像..."
    for service in $SRC_SERVICES; do
        local image="ai-infra-${service}:${tag}"
        if docker image inspect "$image" >/dev/null 2>&1; then
            images_to_clean+=("$image")
            echo "  • $image"
        fi
    done
    
    # 收集依赖镜像
    print_info "收集依赖镜像..."
    local dependency_images
    dependency_images=$(collect_dependency_images)
    
    for dep_image in $dependency_images; do
        # 检查原始镜像
        if docker image inspect "$dep_image" >/dev/null 2>&1; then
            images_to_clean+=("$dep_image")
            echo "  • $dep_image"
        fi
        
        # 检查带标签的依赖镜像（如果不是latest）
        if [[ "$tag" != "latest" && "$dep_image" == *":latest" ]]; then
            local tagged_image="${dep_image%:latest}:${tag}"
            if docker image inspect "$tagged_image" >/dev/null 2>&1; then
                images_to_clean+=("$tagged_image")
                echo "  • $tagged_image"
            fi
        fi
    done
    
    # 收集重新标记的依赖镜像（用于推送的镜像）
    print_info "收集重新标记的依赖镜像..."
    local retagged_images
    retagged_images=$(docker images --format "{{.Repository}}:{{.Tag}}" | grep -E "(ai-infra-dep-|/ai-infra-)" | sort -u)
    
    if [[ -n "$retagged_images" ]]; then
        while IFS= read -r image; do
            if [[ -n "$image" ]]; then
                images_to_clean+=("$image")
                echo "  • $image"
            fi
        done <<< "$retagged_images"
    fi
    
    if [[ ${#images_to_clean[@]} -eq 0 ]]; then
        print_info "没有找到需要清理的镜像"
        return 0
    fi
    
    echo
    print_info "找到 ${#images_to_clean[@]} 个镜像需要清理"
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
    local failed_count=0
    
    for image in "${images_to_clean[@]}"; do
        if docker rmi "$image" 2>/dev/null; then
            print_success "✓ 已删除: $image"
            success_count=$((success_count + 1))
        else
            print_error "✗ 删除失败: $image"
            failed_count=$((failed_count + 1))
        fi
    done
    
    echo
    print_success "清理完成: $success_count 成功, $failed_count 失败"
    
    if [[ $failed_count -gt 0 ]]; then
        print_warning "某些镜像可能正在被容器使用，请先停止相关容器后重试"
        print_info "可以使用以下命令停止所有容器："
        echo "  docker-compose down"
        echo "  docker stop \$(docker ps -aq)"
    fi
}

# 清理悬空镜像和未使用的镜像
clean_dangling_images() {
    local force="${1:-false}"
    
    print_info "=========================================="
    print_info "清理悬空镜像和未使用的镜像"
    print_info "=========================================="
    
    # 清理悬空镜像
    local dangling_images
    dangling_images=$(docker images -f "dangling=true" -q)
    
    if [[ -n "$dangling_images" ]]; then
        print_info "找到悬空镜像:"
        docker images -f "dangling=true" --format "table {{.Repository}}\t{{.Tag}}\t{{.ID}}\t{{.Size}}"
        echo
        
        if [[ "$force" == "true" ]]; then
            print_info "正在删除悬空镜像..."
            docker rmi $dangling_images 2>/dev/null || true
            print_success "悬空镜像清理完成"
        else
            read -p "是否删除这些悬空镜像? (y/N): " confirm
            if [[ "$confirm" == "y" || "$confirm" == "Y" ]]; then
                docker rmi $dangling_images 2>/dev/null || true
                print_success "悬空镜像清理完成"
            fi
        fi
    else
        print_info "没有找到悬空镜像"
    fi
    
    # 清理未使用的镜像
    echo
    print_info "检查未使用的镜像..."
    
    if [[ "$force" == "true" ]]; then
        print_info "正在清理未使用的镜像..."
        docker image prune -f
        print_success "未使用镜像清理完成"
    else
        read -p "是否清理所有未使用的镜像? (y/N): " confirm
        if [[ "$confirm" == "y" || "$confirm" == "Y" ]]; then
            docker image prune -f
            print_success "未使用镜像清理完成"
        fi
    fi
}

# 深度清理：清理所有Docker资源
deep_clean() {
    local force="${1:-false}"
    
    print_warning "=========================================="
    print_warning "深度清理 - 清理所有Docker资源"
    print_warning "=========================================="
    print_warning "这将删除："
    print_warning "  • 所有停止的容器"
    print_warning "  • 所有未使用的网络"
    print_warning "  • 所有悬空镜像"
    print_warning "  • 所有未使用的镜像"
    print_warning "  • 所有构建缓存"
    echo
    
    if [[ "$force" != "true" ]]; then
        read -p "确认执行深度清理? 这可能会影响其他Docker项目 (y/N): " confirm
        if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
            print_info "已取消深度清理操作"
            return 0
        fi
    fi
    
    print_info "正在执行深度清理..."
    
    # 清理停止的容器
    print_info "清理停止的容器..."
    docker container prune -f || true
    
    # 清理未使用的网络
    print_info "清理未使用的网络..."
    docker network prune -f || true
    
    # 清理未使用的卷
    print_info "清理未使用的卷..."
    docker volume prune -f || true
    
    # 清理镜像
    print_info "清理未使用的镜像..."
    docker image prune -a -f || true
    
    # 清理构建缓存
    print_info "清理构建缓存..."
    docker builder prune -a -f || true
    
    print_success "深度清理完成"
    
    # 显示清理后的磁盘使用情况
    echo
    print_info "清理后的Docker磁盘使用情况:"
    docker system df
}

# 显示帮助信息
show_help() {
    echo "AI Infrastructure Matrix - 精简构建脚本 v$VERSION"
    echo
    echo "专注于 src/ 目录下的 Dockerfile 构建，支持依赖镜像管理和 Mock 测试"
    echo
    echo "用法:"
    echo "  $0 [--force] <命令> [参数...]"
    echo
    echo "全局选项:"
    echo "  --force                         - 对构建命令：强制重新构建，忽略本地存在的镜像"
    echo "                                    对prod-up命令：跳过镜像拉取，使用本地已有镜像"
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
    echo "  deps-prod <registry> [tag]      - 拉取、标记并推送生产环境依赖镜像（排除测试工具）"
    echo
    echo "生产环境命令:"
    echo "  prod-generate <registry> [tag]  - 生成生产环境配置文件（使用内部镜像）"
    echo "  prod-up <registry> [tag]        - 启动生产环境"
    echo "  prod-up --force <registry> [tag] - 启动生产环境（跳过镜像拉取，使用本地镜像）"
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
    echo "镜像验证命令:"
    echo "  verify <registry> [tag]        - 验证所有镜像是否可用"
    echo "  verify-key <registry> [tag]    - 快速验证关键镜像"
    echo
    echo "工具命令:"
    echo "  clean [type] [tag] [--force]   - 清理镜像"
    echo "    • clean ai-infra [tag]       - 清理AI-Infra镜像 (默认)"
    echo "    • clean all [tag]            - 清理所有镜像 (AI-Infra + 依赖)"
    echo "    • clean dangling             - 清理悬空镜像"
    echo "    • clean deep                 - 深度清理所有Docker资源"
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
    echo "  $0 --force build backend       # 强制重新构建 backend 服务"
    echo "  $0 build-all v0.3.5            # 构建所有服务，标签 v0.3.5"
    echo "  $0 --force build-all v0.3.5    # 强制重新构建所有服务"
    echo "  $0 build-push registry.local/ai-infra v0.3.5"
    echo "                                  # 构建并推送到私有仓库"
    echo
    echo "  # 依赖镜像操作"
    echo "  $0 deps-pull registry.local/ai-infra latest"
    echo "                                  # 拉取并标记依赖镜像"
    echo "  $0 --force deps-pull registry.local/ai-infra latest"
    echo "                                  # 强制重新拉取依赖镜像"
    echo "  $0 deps-push registry.local/ai-infra latest"
    echo "                                  # 推送依赖镜像"
    echo "  $0 deps-all registry.local/ai-infra v0.3.5"
    echo "  $0 deps-prod registry.local/ai-infra v0.3.5"
    echo "                                  # 完整依赖镜像操作"
    echo
    echo "  # 生产环境操作"
    echo "  $0 prod-generate registry.local/ai-infra v0.3.5"
    echo "                                  # 生成生产环境配置"
    echo "  $0 prod-up registry.local/ai-infra v0.3.5"
    echo "                                  # 启动生产环境"
    echo "  $0 --force prod-up registry.local/ai-infra v0.3.5"
    echo "                                  # 启动生产环境（跳过镜像拉取）"
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
    echo "  注意:"
    echo "    • 默认镜像标签: $DEFAULT_IMAGE_TAG"
    echo "    • 支持 Harbor 和传统 registry 格式"
    echo "    • 构建上下文固定为项目根目录"
    echo
    echo "  # 首次部署"
    echo "  ./scripts/generate-prod-passwords.sh"
    echo "  ./build.sh prod-generate harbor.company.com/ai-infra v1.0.0"
    echo "  ./build.sh prod-up harbor.company.com/ai-infra v1.0.0"
    echo
    echo "  # 版本更新"
    echo "  ./build.sh prod-down"
    echo "  ./build.sh prod-generate harbor.company.com/ai-infra v1.1.0"
    echo "  ./build.sh prod-up harbor.company.com/ai-infra v1.1.0"
    echo
    echo "  # 监控运维"
    echo "  ./build.sh prod-status"
    echo "  ./build.sh prod-logs --follow"
}

# 主函数
main() {
    # 预处理命令行参数，检查 --force 标志
    local args=()
    for arg in "$@"; do
        if [[ "$arg" == "--force" ]]; then
            FORCE_REBUILD=true
            print_info "启用强制重新构建模式"
        else
            args+=("$arg")
        fi
    done
    
    # 重新设置位置参数
    set -- "${args[@]}"
    
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
                print_info "用法: $0 deps-pull <registry> [tag]"
                exit 1
            fi
            pull_and_tag_dependencies "$2" "${3:-latest}"
            ;;
            
        "deps-push")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                print_info "用法: $0 deps-push <registry> [tag]"
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
            
        "deps-prod")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                exit 1
            fi
            local deps_tag="${3:-latest}"
            print_info "执行生产环境依赖镜像操作（排除测试工具）..."
            if pull_and_tag_production_dependencies "$2" "$deps_tag"; then
                push_production_dependencies "$2" "$deps_tag"
            else
                print_error "生产环境依赖镜像拉取失败，停止推送操作"
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
            # 检查是否有 --force 参数
            local force_local="false"
            if [[ "$FORCE_REBUILD" == "true" ]]; then
                force_local="true"
            fi
            start_production "$2" "${3:-$DEFAULT_IMAGE_TAG}" "$force_local"
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
            
        # 镜像验证命令
        "verify")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                print_info "用法: $0 verify <registry> [tag]"
                exit 1
            fi
            verify_private_images "$2" "${3:-v0.3.5}"
            ;;
            
        "verify-key")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                print_info "用法: $0 verify-key <registry> [tag]"
                exit 1
            fi
            verify_key_images "$2" "${3:-v0.3.5}"
            ;;
            
        "clean")
            local clean_type="${2:-ai-infra}"
            local tag_or_force="$3"
            local force_flag="$4"
            local force="false"
            local tag="$DEFAULT_IMAGE_TAG"
            
            # 解析参数
            case "$clean_type" in
                "all")
                    if [[ "$tag_or_force" == "--force" ]]; then
                        force="true"
                    elif [[ -n "$tag_or_force" && "$tag_or_force" != "--force" ]]; then
                        tag="$tag_or_force"
                        if [[ "$force_flag" == "--force" ]]; then
                            force="true"
                        fi
                    fi
                    clean_all_images "$tag" "$force"
                    ;;
                "dangling")
                    if [[ "$tag_or_force" == "--force" ]]; then
                        force="true"
                    fi
                    clean_dangling_images "$force"
                    ;;
                "deep")
                    if [[ "$tag_or_force" == "--force" ]]; then
                        force="true"
                    fi
                    deep_clean "$force"
                    ;;
                "ai-infra"|*)
                    # 默认清理AI-Infra镜像（保持原有行为）
                    if [[ "$clean_type" != "ai-infra" && "$clean_type" != "--force" ]]; then
                        tag="$clean_type"
                    fi
                    if [[ "$tag_or_force" == "--force" ]]; then
                        force="true"
                    elif [[ -n "$tag_or_force" && "$tag_or_force" != "--force" && "$clean_type" == "ai-infra" ]]; then
                        tag="$tag_or_force"
                        if [[ "$force_flag" == "--force" ]]; then
                            force="true"
                        fi
                    fi
                    clean_images "$tag" "$force"
                    ;;
            esac
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
