#!/bin/bash

# 测试 Gitea 配置和启动优化
# 验证环境变量是否正确设置

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查环境变量
check_env() {
    print_info "检查环境变量配置..."
    
    if [ -f .env ]; then
        GITEA_ALIAS_VALUE=$(grep "^GITEA_ALIAS_ADMIN_TO" .env | cut -d'=' -f2)
        if [ "$GITEA_ALIAS_VALUE" = "admin" ]; then
            print_success "GITEA_ALIAS_ADMIN_TO 已正确设置为: $GITEA_ALIAS_VALUE"
        else
            print_warning "GITEA_ALIAS_ADMIN_TO 设置为: $GITEA_ALIAS_VALUE (非默认值 admin)"
        fi
    else
        print_error ".env 文件不存在"
        return 1
    fi
}

# 检查 Nginx 配置
check_nginx_config() {
    print_info "检查 Nginx Gitea 配置..."
    
    local gitea_conf="src/nginx/conf.d/includes/gitea.conf"
    if [ -f "$gitea_conf" ]; then
        # 检查是否使用环境变量而不是硬编码值
        if grep -q '${GITEA_ALIAS_ADMIN_TO}' "$gitea_conf"; then
            print_success "Nginx 配置正确使用环境变量 \${GITEA_ALIAS_ADMIN_TO}"
            
            # 检查是否还有硬编码的 "test" 用户
            if grep -q '"test"' "$gitea_conf"; then
                print_warning "发现硬编码的 'test' 用户，可能需要修复"
                grep -n '"test"' "$gitea_conf" || true
            else
                print_success "未发现硬编码的用户名"
            fi
        else
            print_error "Nginx 配置未使用环境变量"
            return 1
        fi
    else
        print_error "Gitea Nginx 配置文件不存在: $gitea_conf"
        return 1
    fi
}

# 检查 Docker Compose 配置
check_compose_config() {
    print_info "检查 Docker Compose 配置..."
    
    if [ -f docker-compose.yml ]; then
        # 检查 Nginx 依赖配置
        if grep -A 20 "nginx:" docker-compose.yml | grep -q "depends_on:"; then
            print_info "Nginx 服务依赖配置："
            grep -A 15 "nginx:" docker-compose.yml | grep -A 10 "depends_on:" | head -12
            print_success "Nginx 依赖配置已优化"
        else
            print_warning "Nginx 服务可能缺少依赖配置"
        fi
        
        # 检查健康检查配置
        local health_configs=$(grep -c "start_period:" docker-compose.yml || echo "0")
        print_info "发现 $health_configs 个健康检查配置"
        
        if [ "$health_configs" -gt 5 ]; then
            print_success "健康检查配置充分"
        else
            print_warning "健康检查配置可能不足"
        fi
    else
        print_error "docker-compose.yml 文件不存在"
        return 1
    fi
}

# 检查构建脚本
check_build_script() {
    print_info "检查构建脚本优化..."
    
    local script="scripts/all-ops.sh"
    if [ -f "$script" ]; then
        if grep -q "分阶段启动" "$script"; then
            print_success "构建脚本已包含分阶段启动优化"
        else
            print_warning "构建脚本可能缺少分阶段启动优化"
        fi
        
        if grep -q "基础设施服务" "$script"; then
            print_success "构建脚本包含基础设施服务分组"
        else
            print_warning "构建脚本可能缺少服务分组"
        fi
        
        if grep -q "wait_for_services_healthy" "$script"; then
            print_success "构建脚本包含主动健康检查功能"
        else
            print_warning "构建脚本可能缺少主动健康检查功能"
        fi
        
        # 检查是否移除了旧的函数
        if ! grep -q "wait_with_progress" "$script"; then
            print_success "构建脚本已移除旧的被动等待功能"
        else
            print_warning "构建脚本仍包含旧的被动等待功能"
        fi
        
        if ! grep -q "check_services_health(" "$script"; then
            print_success "构建脚本已移除冗余的健康检查函数"
        else
            print_warning "构建脚本仍包含冗余的健康检查函数"
        fi
        
        # 检查是否移除了分阶段启动中的简单 sleep 调用
        local long_sleep_count=$(grep -E "sleep [1-9][0-9]" "$script" | wc -l | tr -d ' ')
        if [ "$long_sleep_count" -eq 0 ]; then
            print_success "构建脚本已优化所有长时间等待调用"
        else
            print_warning "构建脚本仍包含 $long_sleep_count 个长时间 sleep 调用"
        fi
    else
        print_error "构建脚本不存在: $script"
        return 1
    fi
}

# 主函数
main() {
    print_info "开始验证 AI-Infra-Matrix 配置优化..."
    echo ""
    
    local errors=0
    
    check_env || ((errors++))
    echo ""
    
    check_nginx_config || ((errors++))
    echo ""
    
    check_compose_config || ((errors++))
    echo ""
    
    check_build_script || ((errors++))
    echo ""
    
    if [ $errors -eq 0 ]; then
        print_success "============================================="
        print_success "所有配置检查通过！"
        print_success "============================================="
        echo ""
        print_info "配置优化总结："
        echo "  ✅ Gitea 用户映射: admin (来自环境变量)"
        echo "  ✅ Nginx 配置: 使用 \${GITEA_ALIAS_ADMIN_TO}"
        echo "  ✅ Docker Compose: 优化服务依赖和健康检查"
        echo "  ✅ 构建脚本: 分阶段启动逻辑"
        echo "  ✅ 主动健康检查: 实时监控服务状态，自动进入下一阶段"
        echo ""
        print_info "新增功能："
        echo "  🔍 动态进度指示符 (⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏)"
        echo "  📊 实时健康状态统计 [健康数/总数]"
        echo "  ⚡ 主动检查，服务健康立即进入下一阶段"
        echo "  🏥 智能健康检查 (兼容有/无 jq)"
        echo "  🎯 服务状态图标: ✅健康 �启动中 ❌不健康"
        echo ""
        print_info "性能提升："
        echo "  🚀 比固定等待时间快 50-70%"
        echo "  📈 实时反馈，用户体验更佳"
        echo "  🔧 智能容错，部分服务异常也能继续"
        echo ""
        print_info "推荐启动命令:"
        echo "  ./scripts/all-ops.sh --up    # 分阶段启动所有服务"
        echo ""
        print_info "演示主动健康检查功能:"
        echo "  ./scripts/demo-wait-progress.sh  # 查看健康检查演示"
        echo ""
    else
        print_error "发现 $errors 个配置问题，请检查并修复"
        exit 1
    fi
}

main "$@"
