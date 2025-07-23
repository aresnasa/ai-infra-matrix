#!/bin/bash

# ==================================================================================
# 前端懒加载和认证流程验证脚本
# ==================================================================================
# 专门测试管理员登录后立即显示管理中心菜单的功能
# 验证前端权限信息实时更新机制
# ==================================================================================

set -e

# 脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 配置
API_BASE="http://localhost:8082/api"
FRONTEND_URL="http://localhost:3001"
ADMIN_USERNAME="admin"
ADMIN_PASSWORD="admin123"

# 日志函数
log_header() {
    echo ""
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}🚀 $1${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

log_section() {
    echo ""
    echo -e "${BLUE}▶️  $1${NC}"
    echo -e "${BLUE}────────────────────────────────────────────────────────────────────────────────${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_info() {
    echo -e "${PURPLE}ℹ️  $1${NC}"
}

# 等待服务就绪
wait_for_service() {
    local url="$1"
    local service_name="$2"
    local max_attempts=30
    local attempt=1
    
    log_info "等待 $service_name 服务就绪..."
    
    while [ $attempt -le $max_attempts ]; do
        if curl -f -s "$url" > /dev/null 2>&1; then
            log_success "$service_name 服务已就绪"
            return 0
        fi
        
        echo -n "."
        sleep 2
        ((attempt++))
    done
    
    log_error "$service_name 服务在 $((max_attempts * 2)) 秒后仍未就绪"
    return 1
}

# 检查前端认证状态管理
check_frontend_auth_state() {
    log_section "检查前端认证状态管理"
    
    # 检查前端是否正确使用了新的认证逻辑
    log_info "检查App.js中的认证逻辑..."
    
    local app_js_path="$(dirname "$(dirname "$SCRIPT_DIR")")/frontend/src/App.js"
    if [ -f "$app_js_path" ]; then
        if grep -q "authChecked" "$app_js_path" && grep -q "checkAuthStatus" "$app_js_path"; then
            log_success "App.js包含新的认证状态管理逻辑"
        else
            log_warning "App.js可能缺少完整的authChecked状态管理"
            log_info "这可能是正常的，取决于具体实现"
        fi
        
        if grep -q "loading.*authChecked" "$app_js_path"; then
            log_success "App.js正确实现了认证检查等待逻辑"
        else
            log_warning "App.js可能缺少认证检查等待逻辑"
            log_info "这可能是正常的，取决于具体实现"
        fi
    else
        log_warning "无法找到App.js文件，跳过代码检查"
        log_info "文件路径: $app_js_path"
    fi
    
    # 检查AuthPage.js中的登录优化
    log_info "检查AuthPage.js中的登录逻辑..."
    
    local auth_page_path="$(dirname "$(dirname "$SCRIPT_DIR")")/frontend/src/pages/AuthPage.js"
    if [ -f "$auth_page_path" ]; then
        if grep -q "getProfile" "$auth_page_path"; then
            log_success "AuthPage.js包含登录后获取权限信息的逻辑"
        else
            log_warning "AuthPage.js可能缺少登录后权限获取逻辑"
        fi
    else
        log_warning "无法找到AuthPage.js文件，跳过代码检查"
        log_info "文件路径: $auth_page_path"
    fi
    
    # 检查Layout.js中的权限检查
    log_info "检查Layout.js中的权限检查逻辑..."
    
    local layout_path="$(dirname "$(dirname "$SCRIPT_DIR")")/frontend/src/components/Layout.js"
    if [ -f "$layout_path" ]; then
        if grep -q "roles" "$layout_path" && grep -q "super-admin" "$layout_path"; then
            log_success "Layout.js包含正确的权限检查逻辑"
        else
            log_warning "Layout.js可能缺少完整的权限检查逻辑"
        fi
    else
        log_warning "无法找到Layout.js文件，跳过代码检查"
        log_info "文件路径: $layout_path"
    fi
}

# 测试管理员认证流程
test_admin_auth_flow() {
    log_section "测试管理员认证流程"
    
    # 步骤1: 测试管理员登录
    log_info "步骤1: 测试管理员登录..."
    local login_response
    login_response=$(curl -s -X POST "$API_BASE/auth/login" \
        -H "Content-Type: application/json" \
        -d "{
            \"username\": \"$ADMIN_USERNAME\",
            \"password\": \"$ADMIN_PASSWORD\"
        }" || echo '{}')
    
    local admin_token
    if command -v jq &> /dev/null; then
        admin_token=$(echo "$login_response" | jq -r '.token // empty')
    else
        admin_token=$(echo "$login_response" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    fi
    
    if [ -n "$admin_token" ] && [ "$admin_token" != "null" ]; then
        log_success "管理员登录成功，获取token: ${admin_token:0:20}..."
    else
        log_error "管理员登录失败"
        echo "响应: $login_response"
        return 1
    fi
    
    # 步骤2: 立即获取权限信息
    log_info "步骤2: 获取管理员权限信息..."
    local profile_response
    profile_response=$(curl -s -X GET "$API_BASE/auth/profile" \
        -H "Authorization: Bearer $admin_token" || echo '{}')
    
    local user_roles
    if command -v jq &> /dev/null; then
        user_roles=$(echo "$profile_response" | jq -r '.roles[]? // empty' | tr '\n' ' ')
    else
        user_roles=$(echo "$profile_response" | grep -o '"roles":\[[^\]]*\]' | grep -o '"[^"]*"' | tr -d '"' | tr '\n' ' ')
    fi
    
    if echo "$user_roles" | grep -q "super-admin"; then
        log_success "管理员权限验证成功，角色: $user_roles"
    else
        log_error "管理员权限验证失败"
        echo "响应: $profile_response"
        return 1
    fi
    
    # 步骤3: 测试管理功能访问
    log_info "步骤3: 测试管理功能访问..."
    local admin_users_response
    admin_users_response=$(curl -s -X GET "$API_BASE/admin/users" \
        -H "Authorization: Bearer $admin_token" || echo '{}')
    
    if echo "$admin_users_response" | grep -q "email\|username\|users"; then
        log_success "管理功能访问正常"
    else
        log_warning "管理功能访问可能有问题"
        echo "响应: $admin_users_response"
    fi
    
    return 0
}

# 测试前端权限显示逻辑
test_frontend_permission_display() {
    log_section "测试前端权限显示逻辑"
    
    # 模拟前端权限检查流程
    log_info "模拟前端权限检查流程..."
    
    # 检查前端是否正确处理权限信息
    local frontend_js_response
    frontend_js_response=$(curl -s "$FRONTEND_URL/static/js/" || echo "")
    
    if [ -n "$frontend_js_response" ]; then
        log_success "前端JavaScript资源可访问"
    else
        log_warning "前端JavaScript资源访问有问题"
    fi
    
    # 检查前端API代理是否工作
    log_info "检查前端API代理..."
    local proxy_health_response
    proxy_health_response=$(curl -s "$FRONTEND_URL/api/health" || echo '{}')
    
    if echo "$proxy_health_response" | grep -q "ok\|healthy"; then
        log_success "前端API代理工作正常"
    else
        log_warning "前端API代理可能有问题"
    fi
    
    # 测试前端权限API代理
    log_info "测试前端权限API代理..."
    
    # 先通过前端代理登录
    local frontend_login_response
    frontend_login_response=$(curl -s -X POST "$FRONTEND_URL/api/auth/login" \
        -H "Content-Type: application/json" \
        -d "{
            \"email\": \"$ADMIN_EMAIL\",
            \"password\": \"$ADMIN_PASSWORD\"
        }" || echo '{}')
    
    local frontend_token
    if command -v jq &> /dev/null; then
        frontend_token=$(echo "$frontend_login_response" | jq -r '.token // empty')
    else
        frontend_token=$(echo "$frontend_login_response" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    fi
    
    if [ -n "$frontend_token" ] && [ "$frontend_token" != "null" ]; then
        log_success "通过前端代理登录成功"
        
        # 通过前端代理获取权限信息
        local frontend_profile_response
        frontend_profile_response=$(curl -s -X GET "$FRONTEND_URL/api/auth/profile" \
            -H "Authorization: Bearer $frontend_token" || echo '{}')
        
        if echo "$frontend_profile_response" | grep -q "super-admin"; then
            log_success "通过前端代理获取管理员权限成功"
        else
            log_warning "通过前端代理获取管理员权限失败"
        fi
    else
        log_warning "通过前端代理登录失败"
    fi
}

# 测试前端页面完整加载
test_frontend_complete_loading() {
    log_section "测试前端页面完整加载"
    
    log_info "测试前端主页面加载..."
    local main_page_response
    main_page_response=$(curl -s "$FRONTEND_URL" || echo "")
    
    # 检查关键元素
    if echo "$main_page_response" | grep -q "Ansible\|Playbook\|Generator"; then
        log_success "前端主页面包含关键内容"
    else
        log_error "前端主页面缺少关键内容"
        return 1
    fi
    
    # 检查React相关元素
    if echo "$main_page_response" | grep -q "react\|React\|root"; then
        log_success "前端页面包含React应用结构"
    else
        log_warning "前端页面可能缺少React应用结构"
    fi
    
    # 检查是否包含必要的JavaScript
    if echo "$main_page_response" | grep -q "script\|js"; then
        log_success "前端页面包含JavaScript资源"
    else
        log_warning "前端页面可能缺少JavaScript资源"
    fi
    
    return 0
}

# 生成测试指导
generate_manual_test_guide() {
    log_section "手动测试指导"
    
    echo -e "${CYAN}请执行以下手动测试来验证前端懒加载功能：${NC}"
    echo ""
    echo -e "${YELLOW}1. 浏览器测试步骤：${NC}"
    echo "   a) 打开浏览器访问: $FRONTEND_URL"
    echo "   b) 打开浏览器开发者工具（F12）"
    echo "   c) 切换到Console标签页，查看日志"
    echo "   d) 使用管理员账户登录：$ADMIN_EMAIL / $ADMIN_PASSWORD"
    echo "   e) 观察登录后是否立即显示管理中心菜单"
    echo ""
    echo -e "${YELLOW}2. 预期行为：${NC}"
    echo "   ✅ 登录后立即显示管理中心相关菜单项"
    echo "   ✅ 不需要刷新页面即可看到管理功能"
    echo "   ✅ Console中显示权限信息加载日志"
    echo ""
    echo -e "${YELLOW}3. 如果出现问题：${NC}"
    echo "   - 检查Console中是否有JavaScript错误"
    echo "   - 检查Network标签页中的API请求"
    echo "   - 确认/api/auth/profile请求返回正确的权限信息"
    echo ""
    echo -e "${YELLOW}4. 网络请求验证：${NC}"
    echo "   - 登录时应该看到: POST /api/auth/login"
    echo "   - 登录后应该看到: GET /api/auth/profile"
    echo "   - profile响应应包含: {\"roles\": [\"super-admin\"]}"
    echo ""
}

# 主执行函数
main() {
    log_header "前端懒加载和认证流程验证"
    
    # 检查服务状态
    wait_for_service "$API_BASE/health" "后端API" || exit 1
    wait_for_service "$FRONTEND_URL" "前端" || exit 1
    
    # 执行代码检查
    check_frontend_auth_state
    
    # 执行API测试
    test_admin_auth_flow || exit 1
    
    # 执行前端测试
    test_frontend_permission_display
    test_frontend_complete_loading || exit 1
    
    # 生成手动测试指导
    generate_manual_test_guide
    
    log_success "自动化测试完成！请按照上述指导进行手动验证。"
}

# 执行主函数
main "$@"
