#!/bin/bash

# AI Infrastructure Matrix - 精简构建脚本
# 版本: v1.0.0
# 专注于 src/ 目录下的 Dockerfile 构建

set -e

# 全局变量
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VERSION="1.0.0"
DEFAULT_IMAGE_TAG="v0.3.5"

# 源码服务定义 - 使用数组而不是关联数组（兼容macOS bash 3.2）
SRC_SERVICES="backend frontend jupyterhub nginx saltstack"
SRC_PATHS="src/backend src/frontend src/jupyterhub src/nginx src/saltstack"

# 依赖镜像定义（第三方中间件）
DEPENDENCY_IMAGES="postgres:15-alpine redis:7-alpine osixia/openldap:stable osixia/phpldapadmin:stable tecnativa/tcp-proxy redislabs/redisinsight:latest nginx:1.27-alpine quay.io/minio/minio:latest"

# Mock 数据测试相关配置
MOCK_DATA_ENABLED="${MOCK_DATA_ENABLED:-false}"
MOCK_POSTGRES_IMAGE="postgres:15-alpine"
MOCK_REDIS_IMAGE="redis:7-alpine"

# 获取服务对应的路径
get_service_path() {
    local service="$1"
    case "$service" in
        "backend") echo "src/backend" ;;
        "frontend") echo "src/frontend" ;;
        "jupyterhub") echo "src/jupyterhub" ;;
        "nginx") echo "src/nginx" ;;
        "saltstack") echo "src/saltstack" ;;
        *) echo "" ;;
    esac
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
    echo
    
    local success_count=0
    local total_count=0
    local failed_deps=()
    
    for dep_image in $DEPENDENCY_IMAGES; do
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
    echo
    
    local success_count=0
    local total_count=0
    local failed_deps=()
    
    for dep_image in $DEPENDENCY_IMAGES; do
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
    for dep_image in $DEPENDENCY_IMAGES; do
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
