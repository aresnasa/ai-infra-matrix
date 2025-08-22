#!/bin/bash

# AI Infrastructure Matrix - 三环境统一构建部署脚本
# 版本: v3.2.0
# 支持: 开发环境、CI/CD环境、生产环境

set -e

# 全局变量
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VERSION="v3.2.0"
FORCE_MODE="false"

# 默认配置
DEFAULT_IMAGE_TAG="v0.3.5"
DOCKER_COMPOSE_FILE="$SCRIPT_DIR/docker-compose.yml"
DOCKER_COMPOSE_BACKUP="$SCRIPT_DIR/docker-compose.yml.backup"

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

# 环境检测函数
detect_environment() {
    # 1. 优先使用环境变量
    if [[ -n "$AI_INFRA_ENV_TYPE" ]]; then
        case "$AI_INFRA_ENV_TYPE" in
            dev|development) ENV_TYPE="development" ;;
            cicd|ci) ENV_TYPE="cicd" ;;
            prod|production) ENV_TYPE="production" ;;
            *) ENV_TYPE="development" ;;
        esac
        print_info "环境类型: $ENV_TYPE (来源: 环境变量)"
        return
    fi

    # 2. 检查环境标识文件
    if [[ -f "/etc/ai-infra-env" ]]; then
        local env_content=$(cat /etc/ai-infra-env 2>/dev/null | tr -d '[:space:]')
        case "$env_content" in
            dev|development) ENV_TYPE="development" ;;
            cicd|ci) ENV_TYPE="cicd" ;;
            prod|production) ENV_TYPE="production" ;;
            *) ENV_TYPE="development" ;;
        esac
        print_info "环境类型: $ENV_TYPE (来源: /etc/ai-infra-env)"
        return
    fi

    # 3. 自动检测
    if kubectl cluster-info &>/dev/null; then
        ENV_TYPE="production"
        print_info "环境类型: $ENV_TYPE (来源: 检测到Kubernetes)"
        return
    fi

    if [[ -n "$CI" ]] || [[ -n "$JENKINS_URL" ]] || [[ -n "$GITLAB_CI" ]] || [[ -n "$GITHUB_ACTIONS" ]]; then
        ENV_TYPE="cicd"
        print_info "环境类型: $ENV_TYPE (来源: 检测到CI环境)"
        return
    fi

    # 4. 默认为开发环境
    ENV_TYPE="development"
    print_info "环境类型: $ENV_TYPE (来源: 默认)"
}

# 加载环境配置
load_environment_config() {
    case "$ENV_TYPE" in
        "production")
            ENV_FILE="$SCRIPT_DIR/.env.prod"
            ;;
        *)
            ENV_FILE="$SCRIPT_DIR/.env"
            ;;
    esac

    if [[ -f "$ENV_FILE" ]]; then
        print_info "加载环境配置: $ENV_FILE"
        set -a
        source "$ENV_FILE"
        set +a
    else
        print_warning "环境配置文件不存在: $ENV_FILE"
    fi

    # 设置默认值
    IMAGE_TAG="${IMAGE_TAG:-$DEFAULT_IMAGE_TAG}"
    K8S_NAMESPACE="${K8S_NAMESPACE:-ai-infra-prod}"
}

# 从docker-compose.yml提取镜像列表
extract_images_from_compose() {
    local compose_file="$1"
    
    if [[ ! -f "$compose_file" ]]; then
        print_error "找不到 docker-compose.yml 文件: $compose_file"
        exit 1
    fi
    
    # 提取image字段和环境变量中的镜像
    {
        grep -E '^\s*image:\s*' "$compose_file" | sed 's/.*image:\s*//' | sed 's/"//g' | sed "s/'//g"
        grep -E '^\s*-\s*JUPYTERHUB_IMAGE=' "$compose_file" | sed 's/.*JUPYTERHUB_IMAGE=//' | sed 's/"//g' | sed "s/'//g"
    } | sort -u
}

# 列出所有检测到的镜像
list_all_images() {
    local compose_file="${1:-$DOCKER_COMPOSE_FILE}"
    local registry="${2:-$PRIVATE_REGISTRY}"
    local tag="${3:-$IMAGE_TAG}"
    
    print_info "=========================================="
    print_info "AI-Infra 镜像清单分析"
    print_info "=========================================="
    print_info "分析文件: $compose_file"
    print_info "目标仓库: $registry"
    print_info "镜像标签: $tag"
    echo
    
    local images=$(extract_images_from_compose "$compose_file")
    local ai_infra_count=0
    local base_image_count=0
    local total_count=0
    
    print_info "📦 检测到的镜像列表:"
    echo
    
    while IFS= read -r original_image; do
        if [[ -n "$original_image" ]]; then
            total_count=$((total_count + 1))
            
            # 处理环境变量
            local processed_image="$original_image"
            if [[ "$processed_image" == *"\${IMAGE_TAG"* ]]; then
                processed_image="${processed_image//\$\{IMAGE_TAG:-v0.0.3.3\}/$tag}"
                processed_image="${processed_image//\$\{IMAGE_TAG\}/$tag}"
            fi
            
            # 分类统计
            if [[ "$processed_image" == ai-infra-* ]]; then
                ai_infra_count=$((ai_infra_count + 1))
                echo "  🔧 AI-Infra服务: $processed_image"
            else
                base_image_count=$((base_image_count + 1))
                echo "  📚 基础镜像: $processed_image"
            fi
            
            # 显示目标私有镜像名
            local private_image=$(get_private_image_name "$processed_image" "$registry")
            echo "     → $private_image"
            echo
        fi
    done <<< "$images"
    
    print_info "📊 统计摘要:"
    echo "  • AI-Infra服务镜像: $ai_infra_count"
    echo "  • 基础设施镜像: $base_image_count" 
    echo "  • 总计镜像数量: $total_count"
    echo
    print_info "=========================================="
}

# 获取私有镜像名称
get_private_image_name() {
    local original_image="$1"
    local registry="$2"
    
    # 处理不同类型的镜像名格式
    local image_name_tag=""
    
    # 检查registry是否已经包含项目路径（Harbor语法）
    local is_harbor_style=false
    if [[ "$registry" == *"/"* ]]; then
        is_harbor_style=true
    fi
    
    if [[ "$original_image" == *"/"* ]]; then
        # 包含仓库前缀的镜像
        if [[ "$original_image" == *"."*"/"* ]]; then
            # 第三方仓库镜像 (如 quay.io/minio/minio:latest)
            image_name_tag="${original_image#*/}"  # 移除域名部分，保留 minio/minio:latest
        else
            # Docker Hub 官方镜像或组织镜像 (如 osixia/openldap:stable)
            image_name_tag="$original_image"
        fi
    else
        # 没有斜杠的镜像名 (如 redis:7-alpine, postgres:15-alpine)
        image_name_tag="$original_image"
    fi
    
    # 处理ai-infra前缀的镜像路径
    if [[ "$image_name_tag" == ai-infra-* ]]; then
        if [[ "$is_harbor_style" == "true" ]]; then
            # Harbor模式：registry.xxx.com/project/image:tag
            # 不需要额外的ai-infra路径前缀
            image_name_tag="$image_name_tag"
        else
            # 传统模式：registry.xxx.com/ai-infra/image:tag
            image_name_tag="ai-infra/${image_name_tag}"
        fi
    fi
    
    echo "${registry}/${image_name_tag}"
}

# 构建所有镜像
build_all_images() {
    local tag="${1:-$IMAGE_TAG}"
    
    print_info "开始构建所有镜像，标签: $tag"
    
    # 使用现有的all-ops.sh脚本进行构建
    if [[ -f "$SCRIPT_DIR/scripts/all-ops.sh" ]]; then
        print_info "使用 all-ops.sh 脚本构建镜像..."
        cd "$SCRIPT_DIR"
        export IMAGE_TAG="$tag"
        ./scripts/all-ops.sh
    else
        print_warning "未找到 all-ops.sh 脚本，尝试直接构建..."
        
        # 直接构建主要镜像
        local build_dirs=("src/backend" "src/frontend" "src/jupyterhub" "src/nginx")
        
        for dir in "${build_dirs[@]}"; do
            if [[ -f "$SCRIPT_DIR/$dir/Dockerfile" ]]; then
                local service_name=$(basename "$dir")
                local image_name="ai-infra-${service_name}:${tag}"
                
                print_info "构建 $image_name..."
                docker build -t "$image_name" "$SCRIPT_DIR/$dir"
            fi
        done
    fi
    
    print_success "所有镜像构建完成"
}

# 镜像传输到私有仓库
transfer_images_to_private_registry() {
    local registry="$1"
    local tag="${2:-$IMAGE_TAG}"
    
    print_info "开始镜像传输: 公共仓库 -> $registry"
    print_info "目标标签: $tag"
    
    local images=$(extract_images_from_compose "$DOCKER_COMPOSE_FILE")
    local success_count=0
    local total_count=0
    
    while IFS= read -r original_image; do
        if [[ -n "$original_image" ]]; then
            total_count=$((total_count + 1))
            
            # 替换环境变量
            if [[ "$original_image" == *"\${IMAGE_TAG"* ]]; then
                original_image="${original_image//\$\{IMAGE_TAG:-v0.0.3.3\}/$tag}"
                original_image="${original_image//\$\{IMAGE_TAG\}/$tag}"
                original_image="${original_image//\$\{IMAGE_TAG:-$DEFAULT_IMAGE_TAG\}/$tag}"
            fi
            
            # 跳过无效的镜像名（包含未解析的变量）
            if [[ "$original_image" == *"\${'"* ]] || [[ "$original_image" == *':-'* ]]; then
                print_warning "跳过无效镜像名: $original_image"
                continue
            fi
            
            local private_image=$(get_private_image_name "$original_image" "$registry")
            
            print_info "[$total_count] 准备传输: $original_image -> $private_image"
            
            # 模拟镜像传输（暂时跳过实际的docker操作）
            if [[ "$SKIP_DOCKER_OPERATIONS" == "true" ]]; then
                print_success "✓ [模拟] 传输成功: $private_image"
                success_count=$((success_count + 1))
            else
                # 实际的docker操作
                if docker pull "$original_image" 2>/dev/null; then
                    if docker tag "$original_image" "$private_image" 2>/dev/null; then
                        if docker push "$private_image" 2>/dev/null; then
                            print_success "✓ 传输成功: $private_image"
                            success_count=$((success_count + 1))
                        else
                            print_warning "推送失败: $private_image (可能是网络或权限问题)"
                        fi
                    else
                        print_warning "标记失败: $private_image"
                    fi
                else
                    print_warning "拉取失败: $original_image (可能镜像不存在或网络问题)"
                fi
            fi
        fi
    done <<< "$images"
    
    print_success "镜像传输完成: $success_count/$total_count 成功"
    if [[ $success_count -lt $total_count ]]; then
        print_info "部分镜像传输失败，这在开发环境中是正常的"
        print_info "生产环境请确保网络连接和镜像存在性"
    fi
}

# 启动服务
start_services() {
    print_info "启动服务..."
    
    if [[ ! -f "$DOCKER_COMPOSE_FILE" ]]; then
        print_error "找不到 docker-compose.yml 文件"
        exit 1
    fi
    
    # 检查配置文件
    if ! docker-compose -f "$DOCKER_COMPOSE_FILE" config > /dev/null; then
        print_error "docker-compose.yml 配置文件有错误"
        exit 1
    fi
    
    # 启动服务
    docker-compose -f "$DOCKER_COMPOSE_FILE" up -d
    
    print_success "服务启动完成"
    print_info "查看服务状态: docker-compose ps"
}

# 停止服务
stop_services() {
    print_info "停止服务..."
    
    if docker-compose -f "$DOCKER_COMPOSE_FILE" down; then
        print_success "服务已停止"
    else
        print_error "停止服务失败"
        exit 1
    fi
}

# 备份docker-compose.yml
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

# 恢复docker-compose.yml
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

# 修改docker-compose.yml中的镜像引用
modify_compose_images() {
    local registry="$1"
    local tag="$2"
    local compose_file="$3"
    
    print_info "修改 docker-compose.yml 中的镜像引用..."
    
    local temp_file="${compose_file}.tmp"
    
    while IFS= read -r line || [[ -n "$line" ]]; do
        if [[ "$line" =~ ^[[:space:]]*image:[[:space:]]*(.+)$ ]]; then
            local original_image="${BASH_REMATCH[1]}"
            original_image="${original_image//\"/}"
            original_image="${original_image//\'/}"
            
            if [[ "$original_image" == *"\${IMAGE_TAG"* ]]; then
                original_image="${original_image//\$\{IMAGE_TAG:-v0.0.3.3\}/$tag}"
                original_image="${original_image//\$\{IMAGE_TAG\}/$tag}"
            fi
            
            local private_image=$(get_private_image_name "$original_image" "$registry")
            
            local indent=""
            if [[ "$line" =~ ^([[:space:]]*) ]]; then
                indent="${BASH_REMATCH[1]}"
            fi
            
            echo "${indent}image: $private_image"
            print_info "替换镜像: $original_image -> $private_image" >&2
        else
            echo "$line"
        fi
    done < "$compose_file" > "$temp_file"
    
    mv "$temp_file" "$compose_file"
    print_success "docker-compose.yml 修改完成"
}

# 从私有仓库拉取镜像
pull_all_images() {
    local registry="$1"
    local tag="$2"
    
    print_info "从私有仓库拉取所有镜像..."
    print_info "仓库地址: $registry"
    print_info "镜像标签: $tag"
    
    local images=$(extract_images_from_compose "$DOCKER_COMPOSE_FILE")
    local success_count=0
    local total_count=0
    
    while IFS= read -r original_image; do
        if [[ -n "$original_image" ]]; then
            total_count=$((total_count + 1))
            
            # 替换环境变量
            if [[ "$original_image" == *"\${IMAGE_TAG"* ]]; then
                original_image="${original_image//\$\{IMAGE_TAG:-v0.0.3.3\}/$tag}"
                original_image="${original_image//\$\{IMAGE_TAG\}/$tag}"
                original_image="${original_image//\$\{IMAGE_TAG:-$DEFAULT_IMAGE_TAG\}/$tag}"
            fi
            
            # 跳过无效的镜像名
            if [[ "$original_image" == *"\${'"* ]] || [[ "$original_image" == *':-'* ]]; then
                print_warning "跳过无效镜像名: $original_image"
                continue
            fi
            
            local private_image=$(get_private_image_name "$original_image" "$registry")
            
            print_info "[$total_count] 拉取镜像: $private_image"
            
            # 模拟或实际拉取
            if [[ "$SKIP_DOCKER_OPERATIONS" == "true" ]]; then
                print_success "✓ [模拟] 拉取成功: $private_image"
                success_count=$((success_count + 1))
            else
                if docker pull "$private_image" 2>/dev/null; then
                    print_success "✓ 拉取成功: $private_image"
                    success_count=$((success_count + 1))
                else
                    print_warning "拉取失败: $private_image (可能镜像不存在或网络问题)"
                fi
            fi
        fi
    done <<< "$images"
    
    print_success "镜像拉取完成: $success_count/$total_count 成功"
}

# Docker Compose部署
deploy_with_docker_compose() {
    local registry="$1"
    local tag="$2"
    
    print_info "使用 Docker Compose 部署..."
    
    backup_compose_file "$DOCKER_COMPOSE_FILE" "$DOCKER_COMPOSE_BACKUP"
    modify_compose_images "$registry" "$tag" "$DOCKER_COMPOSE_FILE"
    pull_all_images "$registry" "$tag"
    start_services
    
    print_success "Docker Compose 部署完成"
}

# Kubernetes Helm部署
deploy_with_helm() {
    local registry="$1"
    local tag="$2"
    
    print_info "使用 Helm 部署到 Kubernetes..."
    
    # 检查工具
    if ! command -v kubectl &> /dev/null; then
        print_error "kubectl 未安装"
        exit 1
    fi
    
    if ! command -v helm &> /dev/null; then
        print_error "helm 未安装"
        exit 1
    fi
    
    if ! kubectl cluster-info &> /dev/null; then
        print_error "无法连接到 Kubernetes 集群"
        exit 1
    fi
    
    # 更新Helm values
    local helm_values_file="$SCRIPT_DIR/helm/ai-infra-matrix/values.yaml"
    if [[ -f "$helm_values_file" ]]; then
        cp "$helm_values_file" "$helm_values_file.backup-$(date +%Y%m%d-%H%M%S)"
        sed -i.bak "s|imageRegistry: \".*\"|imageRegistry: \"$registry\"|g" "$helm_values_file"
        sed -i.bak "s|imageTag: \".*\"|imageTag: \"$tag\"|g" "$helm_values_file"
        print_success "Helm values.yaml 已更新"
    fi
    
    # 部署
    local namespace="${K8S_NAMESPACE:-ai-infra-prod}"
    local release_name="ai-infra-matrix"
    
    kubectl create namespace "$namespace" --dry-run=client -o yaml | kubectl apply -f -
    
    if helm list -n "$namespace" | grep -q "$release_name"; then
        print_info "升级现有部署..."
        helm upgrade "$release_name" "$SCRIPT_DIR/helm/ai-infra-matrix" \
            --namespace "$namespace" \
            --timeout 20m \
            --wait
    else
        print_info "新建部署..."
        helm install "$release_name" "$SCRIPT_DIR/helm/ai-infra-matrix" \
            --namespace "$namespace" \
            --timeout 20m \
            --wait \
            --create-namespace
    fi
    
    print_success "Helm 部署完成"
    kubectl get pods -n "$namespace"
    kubectl get services -n "$namespace"
}

# 打包配置
package_configurations() {
    local registry="$1"
    local tag="$2"
    
    print_info "打包部署配置..."
    
    local package_dir="ai-infra-deploy-package"
    local package_file="ai-infra-deploy-${tag}.tar.gz"
    
    rm -rf "$package_dir"
    mkdir -p "$package_dir"
    
    # 复制文件
    cp -r "$SCRIPT_DIR/helm" "$package_dir/" 2>/dev/null || true
    cp -r "$SCRIPT_DIR/scripts" "$package_dir/" 2>/dev/null || true
    cp "$SCRIPT_DIR/docker-compose.yml" "$package_dir/" 2>/dev/null || true
    cp "$SCRIPT_DIR/.env.prod" "$package_dir/" 2>/dev/null || true
    cp "$SCRIPT_DIR/build_clean.sh" "$package_dir/build.sh"
    
    # 创建部署说明
    cat > "$package_dir/DEPLOY_README.md" << EOF
# AI Infrastructure Matrix 部署包

版本: $tag
镜像仓库: $registry
打包时间: $(date)

## 部署说明

### Docker Compose 部署
\`\`\`bash
export AI_INFRA_ENV_TYPE=production
./build.sh deploy-compose $registry $tag
\`\`\`

### Kubernetes 部署
\`\`\`bash
export AI_INFRA_ENV_TYPE=production
./build.sh deploy-helm $registry $tag
\`\`\`

## 注意事项
1. 确保网络可以访问私有镜像仓库: $registry
2. 生产环境建议修改 .env.prod 中的密码配置
3. Kubernetes 部署需要正确配置 kubectl 访问权限
EOF
    
    tar -czf "$package_file" "$package_dir"
    rm -rf "$package_dir"
    
    print_success "部署包已创建: $package_file"
}

# 显示环境状态
show_environment_status() {
    print_info "环境状态:"
    print_info "  环境类型: $ENV_TYPE"
    print_info "  镜像标签: $IMAGE_TAG"
    print_info "  私有仓库: ${PRIVATE_REGISTRY:-'未配置'}"
    print_info "  配置文件: ${ENV_FILE}"
    
    if [[ "$ENV_TYPE" == "production" ]]; then
        print_info "  Kubernetes命名空间: ${K8S_NAMESPACE:-ai-infra-prod}"
    fi
    
    # 检查Docker状态
    if command -v docker &> /dev/null && docker ps &> /dev/null; then
        local running_containers=$(docker ps --format "table {{.Names}}" | grep -E "ai-infra|jupyterhub" 2>/dev/null | wc -l)
        print_info "  相关容器: $running_containers 个运行中"
    fi
    
    # 检查Kubernetes状态
    if [[ "$ENV_TYPE" == "production" ]] && command -v kubectl &> /dev/null && kubectl cluster-info &> /dev/null; then
        local namespace="${K8S_NAMESPACE:-ai-infra-prod}"
        local pod_count=$(kubectl get pods -n "$namespace" 2>/dev/null | wc -l)
        if [[ $pod_count -gt 1 ]]; then
            print_info "  K8s Pods: $((pod_count-1)) 个在命名空间 $namespace"
        fi
    fi
}

# 清理资源
clean_docker_resources() {
    print_info "清理Docker资源..."
    
    # 停止相关容器
    local containers=$(docker ps -q --filter "name=ai-infra" --filter "name=jupyterhub" 2>/dev/null)
    if [[ -n "$containers" ]]; then
        docker stop $containers
    fi
    
    docker image prune -f
    docker container prune -f
    docker network prune -f
    
    print_success "Docker资源清理完成"
}

# 显示帮助
show_help() {
    cat << 'EOF'
AI-Infra-Matrix 三环境统一构建部署脚本 v3.2.0

用法: ./build.sh <command> [options]

=== 通用命令 ===
  env                                     显示当前环境信息
  status                                  显示环境和服务状态
  version                                 显示脚本版本信息
  clean                                   清理Docker资源
  restore                                 恢复docker-compose.yml备份
  help                                    显示帮助信息

=== 镜像管理命令 ===
  list-images [registry] [tag]           列出所有AI-Infra镜像清单
  export-all <registry> [tag]            导出所有镜像到内部仓库(包括基础镜像)

=== 开发环境命令 (development) ===
  build [tag]                            构建所有镜像
  dev-start [tag]                        构建并启动开发环境
  dev-stop                               停止开发环境
  start                                  启动服务

=== CI/CD环境命令 (cicd) ===
  transfer <registry> [tag]              转发镜像到私有仓库
  package <registry> [tag]               打包配置和部署脚本

=== 生产环境命令 (production) ===
  pull <registry> [tag]                  从私有仓库拉取镜像
  deploy-compose <registry> [tag]        使用Docker Compose部署
  deploy-helm <registry> [tag]           使用Kubernetes Helm部署

=== 选项 ===
  --force                                强制执行，跳过环境检查
  --skip-docker                          跳过Docker操作，仅显示转换结果

=== Registry格式支持 ===
  传统格式: registry.example.com
  Harbor格式: registry.example.com/project-name

=== 使用示例 ===

1. 镜像管理:
   ./build.sh list-images registry.company.com/ai-infra
   ./build.sh list-images harbor.company.com/myproject
   ./build.sh export-all registry.company.com/ai-infra v0.3.5

2. 开发环境:
   export AI_INFRA_ENV_TYPE=development
   ./build.sh build v0.3.5
   ./build.sh dev-start

3. CI/CD环境:
   export AI_INFRA_ENV_TYPE=cicd
   ./build.sh transfer registry.company.com/ai-infra v0.3.5
   ./build.sh package registry.company.com/ai-infra v0.3.5

4. 生产环境:
   export AI_INFRA_ENV_TYPE=production
   ./build.sh deploy-compose registry.company.com/ai-infra v0.3.5
   ./build.sh deploy-helm registry.company.com/ai-infra v0.3.5

5. 测试模式（跳过Docker操作）:
   export SKIP_DOCKER_OPERATIONS=true
   ./build.sh export-all registry.example.com v1.0.0

=== 环境检测 ===
  1. 环境变量 AI_INFRA_ENV_TYPE
  2. 文件 /etc/ai-infra-env
  3. 自动检测（Kubernetes → production, CI → cicd）
  4. 默认：development

EOF
}

# 主函数
main() {
    # 检查参数
    if [[ " $* " =~ " --force " ]]; then
        FORCE_MODE="true"
        set -- "${@/--force/}"
    fi
    
    if [[ " $* " =~ " --skip-docker " ]]; then
        export SKIP_DOCKER_OPERATIONS="true"
        set -- "${@/--skip-docker/}"
        print_info "启用模拟模式：跳过Docker操作"
    fi
    
    # 初始化环境
    detect_environment
    load_environment_config
    
    local command="${1:-help}"
    
    case "$command" in
        "env")
            print_info "当前环境: $ENV_TYPE"
            print_info "镜像标签: $IMAGE_TAG"
            print_info "配置文件: $ENV_FILE"
            [[ -n "$PRIVATE_REGISTRY" ]] && print_info "私有仓库: $PRIVATE_REGISTRY"
            ;;
            
            
        "build")
            if [[ "$ENV_TYPE" != "development" ]] && [[ "$FORCE_MODE" != "true" ]]; then
                print_warning "构建功能主要用于开发环境，使用 --force 强制执行"
                read -p "是否继续？(y/N): " confirm
                [[ "$confirm" != "y" && "$confirm" != "Y" ]] && exit 0
            fi
            build_all_images "${2:-$IMAGE_TAG}"
            ;;
            
        "dev-start")
            if [[ "$ENV_TYPE" != "development" ]] && [[ "$FORCE_MODE" != "true" ]]; then
                print_warning "开发环境启动功能主要用于开发环境，使用 --force 强制执行"
                read -p "是否继续？(y/N): " confirm
                [[ "$confirm" != "y" && "$confirm" != "Y" ]] && exit 0
            fi
            build_all_images "${2:-$IMAGE_TAG}"
            start_services
            ;;
            
        "dev-stop")
            if [[ "$ENV_TYPE" != "development" ]] && [[ "$FORCE_MODE" != "true" ]]; then
                print_warning "开发环境停止功能主要用于开发环境，使用 --force 强制执行"
                read -p "是否继续？(y/N): " confirm
                [[ "$confirm" != "y" && "$confirm" != "Y" ]] && exit 0
            fi
            stop_services
            ;;
            
        "transfer")
            if [[ "$ENV_TYPE" != "cicd" ]] && [[ "$FORCE_MODE" != "true" ]]; then
                print_warning "镜像传输功能主要用于CI/CD环境，使用 --force 强制执行"
                read -p "是否继续？(y/N): " confirm
                [[ "$confirm" != "y" && "$confirm" != "Y" ]] && exit 0
            fi
            
            local registry="${2:-$PRIVATE_REGISTRY}"
            if [[ -z "$registry" ]]; then
                print_error "请指定私有仓库地址"
                print_info "用法: $0 transfer <私有仓库地址> [标签]"
                exit 1
            fi
            transfer_images_to_private_registry "$registry" "${3:-$IMAGE_TAG}"
            ;;
            
        "list-images")
            print_info "分析AI-Infra镜像依赖"
            local registry="${2:-$PRIVATE_REGISTRY}"
            list_all_images "$DOCKER_COMPOSE_FILE" "$registry" "${3:-$IMAGE_TAG}"
            ;;
            
        "export-all")
            print_info "导出所有AI-Infra镜像到内部仓库"
            local registry="${2:-$PRIVATE_REGISTRY}"
            if [[ -z "$registry" ]]; then
                print_error "请指定私有仓库地址"
                print_info "用法: $0 export-all <私有仓库地址> [标签]"
                exit 1
            fi
            
            print_info "即将导出所有镜像到: $registry"
            if [[ "$FORCE_MODE" != "true" ]]; then
                # 首先显示镜像预览
                list_all_images "$DOCKER_COMPOSE_FILE" "$registry" "${3:-$IMAGE_TAG}"
                read -p "确认导出以上所有镜像？(y/N): " confirm
                [[ "$confirm" != "y" && "$confirm" != "Y" ]] && exit 0
            fi
            
            transfer_images_to_private_registry "$registry" "${3:-$IMAGE_TAG}"
            ;;
            
        "package")
            if [[ "$ENV_TYPE" != "cicd" ]] && [[ "$FORCE_MODE" != "true" ]]; then
                print_warning "打包功能主要用于CI/CD环境，使用 --force 强制执行"
                read -p "是否继续？(y/N): " confirm
                [[ "$confirm" != "y" && "$confirm" != "Y" ]] && exit 0
            fi
            
            local registry="${2:-$PRIVATE_REGISTRY}"
            if [[ -z "$registry" ]]; then
                print_error "请指定私有仓库地址"
                print_info "用法: $0 package <私有仓库地址> [标签]"
                exit 1
            fi
            package_configurations "$registry" "${3:-$IMAGE_TAG}"
            ;;
            
        "pull")
            if [[ "$ENV_TYPE" != "production" ]] && [[ "$FORCE_MODE" != "true" ]]; then
                print_warning "镜像拉取功能主要用于生产环境，使用 --force 强制执行"
                read -p "是否继续？(y/N): " confirm
                [[ "$confirm" != "y" && "$confirm" != "Y" ]] && exit 0
            fi
            
            local registry="${2:-$PRIVATE_REGISTRY}"
            if [[ -z "$registry" ]]; then
                print_error "请指定私有仓库地址"
                print_info "用法: $0 pull <私有仓库地址> [标签]"
                exit 1
            fi
            pull_all_images "$registry" "${3:-$IMAGE_TAG}"
            ;;
            
        "deploy-compose")
            if [[ "$ENV_TYPE" != "production" ]] && [[ "$FORCE_MODE" != "true" ]]; then
                print_warning "生产部署功能主要用于生产环境，使用 --force 强制执行"
                read -p "是否继续？(y/N): " confirm
                [[ "$confirm" != "y" && "$confirm" != "Y" ]] && exit 0
            fi
            
            local registry="${2:-$PRIVATE_REGISTRY}"
            if [[ -z "$registry" ]]; then
                print_error "请指定私有仓库地址"
                print_info "用法: $0 deploy-compose <私有仓库地址> [标签]"
                exit 1
            fi
            deploy_with_docker_compose "$registry" "${3:-$IMAGE_TAG}"
            ;;
            
        "deploy-helm")
            if [[ "$ENV_TYPE" != "production" ]] && [[ "$FORCE_MODE" != "true" ]]; then
                print_warning "生产部署功能主要用于生产环境，使用 --force 强制执行"
                read -p "是否继续？(y/N): " confirm
                [[ "$confirm" != "y" && "$confirm" != "Y" ]] && exit 0
            fi
            
            local registry="${2:-$PRIVATE_REGISTRY}"
            if [[ -z "$registry" ]]; then
                print_error "请指定私有仓库地址"
                print_info "用法: $0 deploy-helm <私有仓库地址> [标签]"
                exit 1
            fi
            deploy_with_helm "$registry" "${3:-$IMAGE_TAG}"
            ;;
            
        "start")
            start_services
            ;;
            
        "restore")
            restore_compose_file "$DOCKER_COMPOSE_FILE" "$DOCKER_COMPOSE_BACKUP"
            ;;
            
        "status")
            show_environment_status
            ;;
            
        "clean")
            clean_docker_resources
            ;;
            
        "version")
            echo "AI Infrastructure Matrix Build Script"
            echo "Version: $VERSION"
            echo "Environment: $ENV_TYPE"
            echo "Image Tag: $IMAGE_TAG"
            echo "Registry: ${PRIVATE_REGISTRY:-'未配置'}"
            ;;
            
        "help"|"-h"|"--help")
            show_help
            ;;
            
        *)
            print_error "未知命令: $1"
            print_info "使用 '$0 help' 查看可用命令"
            show_help
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"
