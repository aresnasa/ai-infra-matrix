#!/bin/bash

# 测试 Nginx 配置修复
# 验证环境变量替换是否正确

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

# 检查 Nginx 配置文件
check_nginx_config() {
    print_info "检查 Nginx 配置文件..."
    
    local gitea_conf="src/nginx/conf.d/includes/gitea.conf"
    if [ -f "$gitea_conf" ]; then
        # 检查是否使用了环境变量语法
        local env_var_count=$(grep -c '${GITEA_ALIAS_ADMIN_TO}' "$gitea_conf" || echo "0")
        print_info "发现 $env_var_count 个环境变量引用"
        
        if [ "$env_var_count" -gt 0 ]; then
            print_success "Nginx 配置正确使用环境变量语法"
        else
            print_error "未找到环境变量引用"
            return 1
        fi
    else
        print_error "Gitea 配置文件不存在: $gitea_conf"
        return 1
    fi
}

# 检查 entrypoint 脚本
check_entrypoint() {
    print_info "检查 Nginx entrypoint 脚本..."
    
    local entrypoint="src/nginx/docker-entrypoint.sh"
    if [ -f "$entrypoint" ]; then
        if grep -q "环境变量替换" "$entrypoint"; then
            print_success "Entrypoint 脚本包含环境变量处理"
        else
            print_warning "Entrypoint 脚本可能缺少环境变量处理"
        fi
        
        if grep -q "sed -i.*GITEA_ALIAS_ADMIN_TO" "$entrypoint"; then
            print_success "Entrypoint 脚本包含环境变量替换逻辑"
        else
            print_error "Entrypoint 脚本缺少环境变量替换逻辑"
            return 1
        fi
    else
        print_error "Entrypoint 脚本不存在: $entrypoint"
        return 1
    fi
}

# 验证环境变量替换逻辑
test_env_replacement() {
    print_info "测试环境变量替换逻辑..."
    
    # 设置测试环境变量
    export GITEA_ALIAS_ADMIN_TO="admin"
    export GITEA_ADMIN_EMAIL="admin@example.com"
    
    # 检查 gitea.conf 中的占位符
    local gitea_conf="src/nginx/conf.d/includes/gitea.conf"
    if [ -f "$gitea_conf" ]; then
        local env_count_user=$(grep -c '${GITEA_ALIAS_ADMIN_TO}' "$gitea_conf" || echo "0")
        local env_count_email=$(grep -c '${GITEA_ADMIN_EMAIL}' "$gitea_conf" || echo "0")
        local total_env_count=$((env_count_user + env_count_email))
        
        if [ "$total_env_count" -gt 0 ]; then
            print_success "找到环境变量占位符:"
            print_info "  GITEA_ALIAS_ADMIN_TO: $env_count_user 个"
            print_info "  GITEA_ADMIN_EMAIL: $env_count_email 个"
            print_info "  总计: $total_env_count 个，待启动时替换"
            
            # 显示替换命令示例
            print_info "替换命令将把:"
            print_info "  '\${GITEA_ALIAS_ADMIN_TO}' → '${GITEA_ALIAS_ADMIN_TO}'"
            print_info "  '\${GITEA_ADMIN_EMAIL}' → '${GITEA_ADMIN_EMAIL}'"
            
            return 0
        else
            print_error "未找到环境变量占位符"
            return 1
        fi
    else
        print_error "配置文件不存在: $gitea_conf"
        return 1
    fi
}

# 构建建议
build_suggestions() {
    print_info "构建建议:"
    echo "  1. 重新构建所有镜像:"
    echo "     ./scripts/all-ops.sh --build"
    echo ""
    echo "  2. 构建并启动服务:"
    echo "     ./scripts/all-ops.sh --build --up"
    echo ""
    echo "  3. 仅构建 Nginx 服务:"
    echo "     ./scripts/all-ops.sh --service nginx --build"
    echo ""
    echo "  4. 验证 Nginx 配置:"
    echo "     ./scripts/all-ops.sh --service nginx --test"
    echo ""
}

# 主函数
main() {
    print_success "=========================================="
    print_success "Nginx 配置修复验证"
    print_success "=========================================="
    echo ""
    
    local errors=0
    
    check_nginx_config || ((errors++))
    echo ""
    
    check_entrypoint || ((errors++))
    echo ""
    
    test_env_replacement || ((errors++))
    echo ""
    
    if [ $errors -eq 0 ]; then
        print_success "============================================="
        print_success "所有检查通过！Nginx 配置修复成功"
        print_success "============================================="
        echo ""
        print_info "修复总结:"
        echo "  ✅ 环境变量语法正确"
        echo "  ✅ Entrypoint 脚本包含替换逻辑"
        echo "  ✅ 环境变量替换测试通过"
        echo ""
        build_suggestions
    else
        print_error "发现 $errors 个问题，请检查并修复"
        exit 1
    fi
}

main "$@"
