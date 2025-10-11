#!/bin/bash

# 生产环境部署修复脚本
# 修复 PostgreSQL 密码认证失败和环境文件挂载问题

set -e

echo "==========================================="
echo "AI-Infra-Matrix 生产环境部署修复"
echo "==========================================="

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

# 检查必要文件
check_required_files() {
    print_info "检查必要文件..."
    
    local required_files=(".env.prod" "docker-compose.prod.yml" "build.sh")
    local missing_files=()
    
    for file in "${required_files[@]}"; do
        if [[ ! -f "$file" ]]; then
            missing_files+=("$file")
        fi
    done
    
    if [[ ${#missing_files[@]} -gt 0 ]]; then
        print_error "缺少必要文件: ${missing_files[*]}"
        print_info "请运行以下命令生成："
        print_info "  ./build.sh prod-generate <registry> <tag>"
        return 1
    fi
    
    print_success "✓ 所有必要文件存在"
    return 0
}

# 检查环境文件配置
check_env_config() {
    print_info "检查环境文件配置..."
    
    if [[ ! -f ".env.prod" ]]; then
        print_error "未找到 .env.prod 文件"
        return 1
    fi
    
    # 检查关键配置
    local postgres_password=$(grep -E '^POSTGRES_PASSWORD=' .env.prod | cut -d'=' -f2)
    local postgres_user=$(grep -E '^POSTGRES_USER=' .env.prod | cut -d'=' -f2)
    
    if [[ -z "$postgres_password" ]] || [[ "$postgres_password" == "postgres" ]]; then
        print_warning "PostgreSQL 密码未设置或使用默认值"
        print_info "建议修改 .env.prod 中的 POSTGRES_PASSWORD"
    else
        print_success "✓ PostgreSQL 密码已配置"
    fi
    
    if [[ -z "$postgres_user" ]]; then
        print_warning "PostgreSQL 用户未设置"
    else
        print_success "✓ PostgreSQL 用户: $postgres_user"
    fi
    
    return 0
}

# 检查 backend-init 配置
check_backend_init_config() {
    print_info "检查 backend-init 服务配置..."
    
    if [[ ! -f "docker-compose.prod.yml" ]]; then
        print_error "未找到 docker-compose.prod.yml 文件"
        return 1
    fi
    
    # 检查是否有 volume 挂载
    if grep -q "\.env\.prod:/app/\.env:ro" docker-compose.prod.yml; then
        print_success "✓ 环境文件挂载配置正确"
    else
        print_error "✗ 缺少环境文件挂载配置"
        print_info "需要重新生成配置文件或手动添加："
        print_info "    volumes:"
        print_info "    - ./.env.prod:/app/.env:ro"
        return 1
    fi
    
    # 检查重启策略
    if grep -A 30 "backend-init:" docker-compose.prod.yml | grep -q 'restart: .no.'; then
        print_success "✓ 重启策略配置正确"
    else
        print_warning "⚠ 重启策略可能不正确"
    fi
    
    return 0
}

# 验证 Docker Compose 配置
validate_compose() {
    print_info "验证 Docker Compose 配置..."
    
    if command -v docker >/dev/null 2>&1; then
        if docker compose -f docker-compose.prod.yml config >/dev/null 2>&1; then
            print_success "✓ Docker Compose 配置验证通过"
        else
            print_error "✗ Docker Compose 配置验证失败"
            print_info "详细错误："
            docker compose -f docker-compose.prod.yml config
            return 1
        fi
    else
        print_warning "未安装 Docker，跳过配置验证"
    fi
    
    return 0
}

# 提供修复建议
provide_fix_suggestions() {
    print_info "==========================================="
    print_info "修复建议"
    print_info "==========================================="
    
    echo
    print_info "1. 确保使用最新的构建脚本生成配置："
    echo "   ./build.sh prod-generate <registry> <tag>"
    
    echo
    print_info "2. 检查 .env.prod 文件中的数据库密码："
    echo "   grep POSTGRES_PASSWORD .env.prod"
    
    echo
    print_info "3. 启动生产环境："
    echo "   ./build.sh --force prod-up <registry> <tag>"
    
    echo
    print_info "4. 查看 backend-init 日志："
    echo "   docker logs ai-infra-backend-init"
    
    echo
    print_info "5. 如果仍有问题，检查数据库连接："
    echo "   docker logs ai-infra-postgres"
    
    echo
    print_success "修复完成后，backend-init 应该能够："
    print_success "  - 读取 .env 文件（挂载到 /app/.env）"
    print_success "  - 连接到 PostgreSQL 数据库"
    print_success "  - 完成初始化后正常退出"
    print_success "  - 允许 backend 服务启动"
}

# 主函数
main() {
    echo
    
    # 检查必要文件
    if ! check_required_files; then
        exit 1
    fi
    
    echo
    
    # 检查环境配置
    check_env_config
    
    echo
    
    # 检查 backend-init 配置
    if ! check_backend_init_config; then
        echo
        provide_fix_suggestions
        exit 1
    fi
    
    echo
    
    # 验证 compose 配置
    if ! validate_compose; then
        echo
        provide_fix_suggestions
        exit 1
    fi
    
    echo
    print_success "🎉 所有配置检查通过！"
    print_info "您现在可以安全地启动生产环境"
    
    echo
    provide_fix_suggestions
}

# 运行主函数
main "$@"
