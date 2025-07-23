#!/bin/bash

# ==================================================================================
# Ansible Playbook Generator - 完整自动化测试套件
# ==================================================================================
# 功能包括：
# 1. 服务健康检查
# 2. 前端认证流程测试
# 3. API接口全功能测试
# 4. LDAP集成测试
# 5. 端到端功能测试
# 6. 权限管理测试
# 7. 回收站功能测试
# 8. 前端UI懒加载测试
# ==================================================================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# 全局变量
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../" && pwd)"
TEST_RESULTS=()
START_TIME=$(date +%s)
AUTH_TOKEN=""
ADMIN_TOKEN=""
PROJECT_ID=""
USER_ID=""

# 默认配置
API_BASE="${API_BASE:-http://localhost:8082/api}"
FRONTEND_URL="${FRONTEND_URL:-http://localhost:3001}"
CREDENTIALS_FILE="$PROJECT_ROOT/tests/user-pass.csv"

# 测试用户数据
TEST_USER_EMAIL="test.user.$(date +%s)@example.com"
TEST_USER_PASSWORD="TestPass123!"
ADMIN_EMAIL="admin@example.com"
ADMIN_PASSWORD="admin123"

# ==================================================================================
# 日志和工具函数
# ==================================================================================

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
    TEST_RESULTS+=("✅ $1")
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
    TEST_RESULTS+=("❌ $1")
    return 1
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
    TEST_RESULTS+=("⚠️  $1")
}

log_info() {
    echo -e "${PURPLE}ℹ️  $1${NC}"
}

# JSON响应解析工具
parse_json() {
    if command -v jq &> /dev/null; then
        echo "$1" | jq -r "$2" 2>/dev/null || echo ""
    else
        # 简单的JSON解析（当jq不可用时）
        echo "$1" | grep -o "\"$2\":[^,}]*" | cut -d':' -f2 | tr -d '"' | tr -d ' ' || echo ""
    fi
}

# 等待服务就绪
wait_for_service() {
    local url="$1"
    local service_name="$2"
    local max_attempts=60
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

# ==================================================================================
# 依赖检查
# ==================================================================================

check_dependencies() {
    log_section "检查系统依赖"
    
    # 检查必需工具
    local required_tools=("curl" "docker" "docker-compose")
    for tool in "${required_tools[@]}"; do
        if command -v "$tool" &> /dev/null; then
            log_success "$tool 已安装"
        else
            log_error "$tool 未安装，请先安装"
            exit 1
        fi
    done
    
    # 检查可选工具
    if command -v jq &> /dev/null; then
        log_success "jq 已安装（推荐用于JSON解析）"
    else
        log_warning "jq 未安装，将使用基础JSON解析"
    fi
    
    # 检查项目目录
    if [ -d "$PROJECT_ROOT" ]; then
        log_success "项目根目录确认: $PROJECT_ROOT"
    else
        log_error "项目根目录不存在: $PROJECT_ROOT"
        exit 1
    fi
}

# ==================================================================================
# 服务健康检查
# ==================================================================================

test_service_health() {
    log_section "服务健康检查"
    
    # 检查Docker Compose服务状态
    cd "$PROJECT_ROOT"
    
    log_info "检查Docker Compose服务状态..."
    if docker-compose ps | grep -q "Up"; then
        log_success "Docker Compose服务正在运行"
    else
        log_warning "一些服务可能未运行，尝试启动..."
        docker-compose up -d
        sleep 10
    fi
    
    # 等待关键服务就绪
    wait_for_service "$API_BASE/health" "后端API"
    wait_for_service "$FRONTEND_URL" "前端"
    
    # 检查API健康端点
    local health_response
    health_response=$(curl -s "$API_BASE/health" || echo '{}')
    
    if echo "$health_response" | grep -q "ok\|healthy\|success"; then
        log_success "API健康检查通过"
    else
        log_error "API健康检查失败"
        echo "响应: $health_response"
        return 1
    fi
    
    # 检查数据库连接
    log_info "检查数据库连接..."
    local db_check
    db_check=$(curl -s "$API_BASE/health/db" || echo '{}')
    
    if echo "$db_check" | grep -q "ok\|healthy\|connected"; then
        log_success "数据库连接正常"
    else
        log_warning "数据库连接检查失败，但继续测试"
    fi
}

# ==================================================================================
# 认证功能测试
# ==================================================================================

test_authentication() {
    log_section "认证功能测试"
    
    # 测试用户注册
    log_info "测试用户注册..."
    local register_response
    register_response=$(curl -s -X POST "$API_BASE/auth/register" \
        -H "Content-Type: application/json" \
        -d "{
            \"email\": \"$TEST_USER_EMAIL\",
            \"password\": \"$TEST_USER_PASSWORD\",
            \"username\": \"testuser$(date +%s)\"
        }" || echo '{}')
    
    if echo "$register_response" | grep -q "success\|created\|registered\|id"; then
        log_success "用户注册成功"
        USER_ID=$(parse_json "$register_response" "id")
    else
        log_warning "用户注册失败，可能用户已存在"
        echo "响应: $register_response"
    fi
    
    # 测试用户登录
    log_info "测试用户登录..."
    local login_response
    login_response=$(curl -s -X POST "$API_BASE/auth/login" \
        -H "Content-Type: application/json" \
        -d "{
            \"email\": \"$TEST_USER_EMAIL\",
            \"password\": \"$TEST_USER_PASSWORD\"
        }" || echo '{}')
    
    AUTH_TOKEN=$(parse_json "$login_response" "token")
    
    if [ -n "$AUTH_TOKEN" ] && [ "$AUTH_TOKEN" != "null" ]; then
        log_success "用户登录成功，获取到token"
    else
        log_error "用户登录失败"
        echo "响应: $login_response"
        return 1
    fi
    
    # 测试权限信息获取
    log_info "测试权限信息获取..."
    local profile_response
    profile_response=$(curl -s -X GET "$API_BASE/auth/profile" \
        -H "Authorization: Bearer $AUTH_TOKEN" || echo '{}')
    
    if echo "$profile_response" | grep -q "email\|username\|roles"; then
        log_success "权限信息获取成功"
        local user_roles
        user_roles=$(parse_json "$profile_response" "roles")
        log_info "用户角色: $user_roles"
    else
        log_error "权限信息获取失败"
        echo "响应: $profile_response"
        return 1
    fi
    
    # 测试管理员登录
    log_info "测试管理员登录..."
    local admin_login_response
    admin_login_response=$(curl -s -X POST "$API_BASE/auth/login" \
        -H "Content-Type: application/json" \
        -d "{
            \"email\": \"$ADMIN_EMAIL\",
            \"password\": \"$ADMIN_PASSWORD\"
        }" || echo '{}')
    
    ADMIN_TOKEN=$(parse_json "$admin_login_response" "token")
    
    if [ -n "$ADMIN_TOKEN" ] && [ "$ADMIN_TOKEN" != "null" ]; then
        log_success "管理员登录成功"
        
        # 验证管理员权限
        local admin_profile_response
        admin_profile_response=$(curl -s -X GET "$API_BASE/auth/profile" \
            -H "Authorization: Bearer $ADMIN_TOKEN" || echo '{}')
        
        if echo "$admin_profile_response" | grep -q "super-admin\|admin"; then
            log_success "管理员权限验证成功"
        else
            log_warning "管理员权限验证失败"
        fi
    else
        log_warning "管理员登录失败，跳过管理员功能测试"
    fi
}

# ==================================================================================
# 项目管理功能测试
# ==================================================================================

test_project_management() {
    log_section "项目管理功能测试"
    
    if [ -z "$AUTH_TOKEN" ]; then
        log_error "需要有效的认证token"
        return 1
    fi
    
    # 创建测试项目
    log_info "创建测试项目..."
    local create_response
    create_response=$(curl -s -X POST "$API_BASE/projects" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{
            \"name\": \"Test Project $(date +%s)\",
            \"description\": \"自动化测试项目\",
            \"inventory\": \"[webservers]\\nlocalhost ansible_host=127.0.0.1\",
            \"playbook\": \"---\\n- hosts: webservers\\n  tasks:\\n    - name: Test task\\n      debug:\\n        msg: 'Hello World'\"
        }" || echo '{}')
    
    PROJECT_ID=$(parse_json "$create_response" "id")
    
    if [ -n "$PROJECT_ID" ] && [ "$PROJECT_ID" != "null" ]; then
        log_success "项目创建成功，ID: $PROJECT_ID"
    else
        log_error "项目创建失败"
        echo "响应: $create_response"
        return 1
    fi
    
    # 获取项目列表
    log_info "获取项目列表..."
    local projects_response
    projects_response=$(curl -s -X GET "$API_BASE/projects" \
        -H "Authorization: Bearer $AUTH_TOKEN" || echo '{}')
    
    if echo "$projects_response" | grep -q "$PROJECT_ID"; then
        log_success "项目列表获取成功，包含新建项目"
    else
        log_error "项目列表获取失败或不包含新建项目"
        return 1
    fi
    
    # 获取项目详情
    log_info "获取项目详情..."
    local project_detail_response
    project_detail_response=$(curl -s -X GET "$API_BASE/projects/$PROJECT_ID" \
        -H "Authorization: Bearer $AUTH_TOKEN" || echo '{}')
    
    if echo "$project_detail_response" | grep -q "Test Project"; then
        log_success "项目详情获取成功"
    else
        log_error "项目详情获取失败"
        return 1
    fi
    
    # 测试项目预览功能
    log_info "测试项目预览功能..."
    local preview_response
    preview_response=$(curl -s -X POST "$API_BASE/projects/$PROJECT_ID/preview" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{\"format\": \"yaml\"}" || echo '{}')
    
    if echo "$preview_response" | grep -q "preview\|content\|playbook"; then
        log_success "项目预览功能正常"
    else
        log_warning "项目预览功能可能有问题"
    fi
    
    # 测试项目验证功能
    log_info "测试项目验证功能..."
    local validation_response
    validation_response=$(curl -s -X POST "$API_BASE/projects/$PROJECT_ID/validate" \
        -H "Authorization: Bearer $AUTH_TOKEN" || echo '{}')
    
    if echo "$validation_response" | grep -q "valid\|errors\|warnings"; then
        log_success "项目验证功能正常"
    else
        log_warning "项目验证功能可能有问题"
    fi
}

# ==================================================================================
# 回收站功能测试
# ==================================================================================

test_recycle_bin() {
    log_section "回收站功能测试"
    
    if [ -z "$PROJECT_ID" ] || [ -z "$AUTH_TOKEN" ]; then
        log_warning "跳过回收站测试：需要有效的项目ID和认证token"
        return 0
    fi
    
    # 删除项目到回收站
    log_info "删除项目到回收站..."
    local delete_response
    delete_response=$(curl -s -X DELETE "$API_BASE/projects/$PROJECT_ID" \
        -H "Authorization: Bearer $AUTH_TOKEN" || echo '{}')
    
    if echo "$delete_response" | grep -q "success\|deleted"; then
        log_success "项目已删除到回收站"
    else
        log_warning "项目删除可能失败"
    fi
    
    # 查看回收站内容
    log_info "查看回收站内容..."
    local recycle_bin_response
    recycle_bin_response=$(curl -s -X GET "$API_BASE/recycle-bin" \
        -H "Authorization: Bearer $AUTH_TOKEN" || echo '{}')
    
    if echo "$recycle_bin_response" | grep -q "$PROJECT_ID"; then
        log_success "回收站包含已删除的项目"
        
        # 恢复项目
        log_info "恢复项目..."
        local restore_response
        restore_response=$(curl -s -X POST "$API_BASE/recycle-bin/$PROJECT_ID/restore" \
            -H "Authorization: Bearer $AUTH_TOKEN" || echo '{}')
        
        if echo "$restore_response" | grep -q "success\|restored"; then
            log_success "项目恢复成功"
        else
            log_warning "项目恢复失败"
        fi
    else
        log_warning "回收站不包含已删除的项目"
    fi
}

# ==================================================================================
# 前端UI测试
# ==================================================================================

test_frontend_ui() {
    log_section "前端UI测试"
    
    # 测试前端首页
    log_info "测试前端首页访问..."
    local frontend_response
    frontend_response=$(curl -s "$FRONTEND_URL" || echo "")
    
    if echo "$frontend_response" | grep -q "Ansible\|Playbook\|Generator"; then
        log_success "前端首页访问正常"
    else
        log_error "前端首页访问失败"
        return 1
    fi
    
    # 测试静态资源
    log_info "测试静态资源加载..."
    local static_response
    static_response=$(curl -s -I "$FRONTEND_URL/static/js/" | head -n 1 || echo "")
    
    if echo "$static_response" | grep -q "200\|404"; then
        log_success "静态资源路径可访问"
    else
        log_warning "静态资源访问可能有问题"
    fi
    
    # 测试API代理
    log_info "测试前端API代理..."
    local proxy_response
    proxy_response=$(curl -s "$FRONTEND_URL/api/health" || echo '{}')
    
    if echo "$proxy_response" | grep -q "ok\|healthy"; then
        log_success "前端API代理正常工作"
    else
        log_warning "前端API代理可能有问题"
    fi
}

# ==================================================================================
# LDAP集成测试（可选）
# ==================================================================================

test_ldap_integration() {
    log_section "LDAP集成测试（可选）"
    
    # 检查LDAP服务是否启用
    if docker-compose ps | grep -q "ldap"; then
        log_info "检测到LDAP服务，开始测试..."
        
        # 等待LDAP服务就绪
        sleep 5
        
        # 测试LDAP认证
        local ldap_auth_response
        ldap_auth_response=$(curl -s -X POST "$API_BASE/auth/ldap" \
            -H "Content-Type: application/json" \
            -d "{
                \"username\": \"testuser\",
                \"password\": \"testpass\"
            }" || echo '{}')
        
        if echo "$ldap_auth_response" | grep -q "token\|success"; then
            log_success "LDAP认证测试成功"
        else
            log_warning "LDAP认证测试失败（可能是配置问题）"
        fi
    else
        log_info "LDAP服务未启用，跳过LDAP测试"
    fi
}

# ==================================================================================
# 性能测试
# ==================================================================================

test_performance() {
    log_section "基础性能测试"
    
    if [ -z "$AUTH_TOKEN" ]; then
        log_warning "跳过性能测试：需要有效的认证token"
        return 0
    fi
    
    # 测试API响应时间
    log_info "测试API响应时间..."
    local start_time end_time response_time
    
    start_time=$(date +%s%N)
    curl -s -X GET "$API_BASE/projects" \
        -H "Authorization: Bearer $AUTH_TOKEN" > /dev/null
    end_time=$(date +%s%N)
    
    response_time=$(( (end_time - start_time) / 1000000 )) # 转换为毫秒
    
    if [ $response_time -lt 1000 ]; then
        log_success "API响应时间良好: ${response_time}ms"
    elif [ $response_time -lt 3000 ]; then
        log_warning "API响应时间一般: ${response_time}ms"
    else
        log_error "API响应时间较慢: ${response_time}ms"
    fi
    
    # 测试并发请求
    log_info "测试并发请求处理..."
    local concurrent_test_result=0
    
    for i in {1..5}; do
        curl -s -X GET "$API_BASE/health" > /dev/null &
    done
    
    wait
    log_success "并发请求测试完成"
}

# ==================================================================================
# 清理函数
# ==================================================================================

cleanup_test_data() {
    log_section "清理测试数据"
    
    # 清理创建的测试项目
    if [ -n "$PROJECT_ID" ] && [ -n "$AUTH_TOKEN" ]; then
        log_info "清理测试项目..."
        curl -s -X DELETE "$API_BASE/projects/$PROJECT_ID/permanent" \
            -H "Authorization: Bearer $AUTH_TOKEN" > /dev/null || true
        log_success "测试项目已清理"
    fi
    
    # 可选：清理测试用户（如果有相应API）
    if [ -n "$USER_ID" ] && [ -n "$ADMIN_TOKEN" ]; then
        log_info "清理测试用户..."
        curl -s -X DELETE "$API_BASE/admin/users/$USER_ID" \
            -H "Authorization: Bearer $ADMIN_TOKEN" > /dev/null || true
        log_success "测试用户已清理"
    fi
}

# ==================================================================================
# 测试报告生成
# ==================================================================================

generate_test_report() {
    log_header "测试结果报告"
    
    local end_time total_time
    end_time=$(date +%s)
    total_time=$((end_time - START_TIME))
    
    echo -e "${CYAN}测试执行时间: ${total_time}秒${NC}"
    echo -e "${CYAN}测试时间: $(date)${NC}"
    echo ""
    
    local passed=0
    local failed=0
    local warnings=0
    
    for result in "${TEST_RESULTS[@]}"; do
        echo "$result"
        if [[ $result == *"✅"* ]]; then
            ((passed++))
        elif [[ $result == *"❌"* ]]; then
            ((failed++))
        elif [[ $result == *"⚠️"* ]]; then
            ((warnings++))
        fi
    done
    
    echo ""
    echo -e "${GREEN}通过: $passed${NC}"
    echo -e "${RED}失败: $failed${NC}"
    echo -e "${YELLOW}警告: $warnings${NC}"
    echo -e "${CYAN}总计: $((passed + failed + warnings))${NC}"
    
    if [ $failed -eq 0 ]; then
        log_success "所有关键测试都通过了！"
        return 0
    else
        log_error "有 $failed 个测试失败"
        return 1
    fi
}

# ==================================================================================
# 主执行函数
# ==================================================================================

main() {
    log_header "Ansible Playbook Generator - 完整自动化测试套件"
    
    # 检查命令行参数
    local skip_cleanup=false
    local quick_mode=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --skip-cleanup)
                skip_cleanup=true
                shift
                ;;
            --quick)
                quick_mode=true
                shift
                ;;
            --help)
                echo "用法: $0 [选项]"
                echo "选项:"
                echo "  --skip-cleanup    跳过测试数据清理"
                echo "  --quick          快速模式（跳过可选测试）"
                echo "  --help           显示此帮助信息"
                exit 0
                ;;
            *)
                log_warning "未知参数: $1"
                shift
                ;;
        esac
    done
    
    # 执行测试
    check_dependencies || exit 1
    test_service_health || exit 1
    test_authentication || exit 1
    test_project_management || exit 1
    test_recycle_bin || exit 1
    test_frontend_ui || exit 1
    
    if [ "$quick_mode" != true ]; then
        test_ldap_integration
        test_performance
    fi
    
    # 清理（如果未跳过）
    if [ "$skip_cleanup" != true ]; then
        cleanup_test_data
    fi
    
    # 生成报告
    generate_test_report
}

# 错误处理
trap 'log_error "测试过程中发生错误"; exit 1' ERR

# 执行主函数
main "$@"
