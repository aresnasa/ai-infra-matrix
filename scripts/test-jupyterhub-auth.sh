#!/bin/bash

# JupyterHub统一认证测试脚本
# 测试JupyterHub与AI基础设施矩阵后端的统一认证集成

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 配置
BACKEND_URL="http://localhost:8080"
JUPYTERHUB_URL="http://localhost:8090"
TEST_USER="testuser"
TEST_PASSWORD="testpass123"
TEST_EMAIL="test@example.com"

print_info "开始JupyterHub统一认证集成测试..."

# 测试1: 检查后端API健康状态
test_backend_health() {
    print_info "测试1: 检查后端API健康状态"
    
    if curl -s "$BACKEND_URL/api/health" > /dev/null; then
        print_success "后端API响应正常"
        return 0
    else
        print_error "后端API无法访问"
        return 1
    fi
}

# 测试2: 创建测试用户
test_create_user() {
    print_info "测试2: 创建测试用户"
    
    response=$(curl -s -X POST "$BACKEND_URL/api/auth/register" \
        -H "Content-Type: application/json" \
        -d "{
            \"username\": \"$TEST_USER\",
            \"email\": \"$TEST_EMAIL\",
            \"password\": \"$TEST_PASSWORD\"
        }")
    
    if echo "$response" | grep -q "id"; then
        print_success "测试用户创建成功"
        return 0
    else
        print_warning "用户可能已存在或创建失败: $response"
        return 0  # 继续测试
    fi
}

# 测试3: 后端登录获取JWT
test_backend_login() {
    print_info "测试3: 后端登录获取JWT token"
    
    response=$(curl -s -X POST "$BACKEND_URL/api/auth/login" \
        -H "Content-Type: application/json" \
        -d "{
            \"username\": \"$TEST_USER\",
            \"password\": \"$TEST_PASSWORD\"
        }")
    
    JWT_TOKEN=$(echo "$response" | jq -r '.token // empty')
    
    if [ -n "$JWT_TOKEN" ] && [ "$JWT_TOKEN" != "null" ]; then
        print_success "JWT token获取成功: ${JWT_TOKEN:0:50}..."
        return 0
    else
        print_error "JWT token获取失败: $response"
        return 1
    fi
}

# 测试4: 验证JWT token
test_verify_token() {
    print_info "测试4: 验证JWT token"
    
    if [ -z "$JWT_TOKEN" ]; then
        print_error "没有可用的JWT token"
        return 1
    fi
    
    response=$(curl -s -X POST "$BACKEND_URL/api/auth/verify-token" \
        -H "Content-Type: application/json" \
        -d "{\"token\": \"$JWT_TOKEN\"}")
    
    if echo "$response" | grep -q '"valid":true'; then
        print_success "JWT token验证成功"
        USER_INFO=$(echo "$response" | jq -r '.user.username')
        print_info "验证用户: $USER_INFO"
        return 0
    else
        print_error "JWT token验证失败: $response"
        return 1
    fi
}

# 测试5: JupyterHub专用登录
test_jupyterhub_login() {
    print_info "测试5: JupyterHub专用登录接口"
    
    response=$(curl -s -X POST "$BACKEND_URL/api/auth/jupyterhub-login" \
        -H "Content-Type: application/json" \
        -d "{
            \"username\": \"$TEST_USER\",
            \"password\": \"$TEST_PASSWORD\"
        }")
    
    if echo "$response" | grep -q '"success":true'; then
        print_success "JupyterHub登录接口测试成功"
        JUPYTER_TOKEN=$(echo "$response" | jq -r '.token')
        print_info "JupyterHub token: ${JUPYTER_TOKEN:0:50}..."
        return 0
    else
        print_error "JupyterHub登录接口失败: $response"
        return 1
    fi
}

# 测试6: Token刷新
test_refresh_token() {
    print_info "测试6: Token刷新功能"
    
    if [ -z "$JWT_TOKEN" ]; then
        print_error "没有可用的JWT token"
        return 1
    fi
    
    response=$(curl -s -X POST "$BACKEND_URL/api/auth/refresh-token" \
        -H "Content-Type: application/json" \
        -d "{\"token\": \"$JWT_TOKEN\"}")
    
    if echo "$response" | grep -q '"success":true'; then
        print_success "Token刷新成功"
        NEW_TOKEN=$(echo "$response" | jq -r '.token')
        print_info "新token: ${NEW_TOKEN:0:50}..."
        return 0
    else
        print_error "Token刷新失败: $response"
        return 1
    fi
}

# 测试7: 检查JupyterHub服务状态
test_jupyterhub_status() {
    print_info "测试7: 检查JupyterHub服务状态"
    
    if curl -s "$JUPYTERHUB_URL" > /dev/null; then
        print_success "JupyterHub服务响应正常"
        return 0
    else
        print_warning "JupyterHub服务未启动或无法访问"
        return 1
    fi
}

# 测试8: 检查统一认证配置
test_auth_config() {
    print_info "测试8: 检查JupyterHub统一认证配置"
    
    CONFIG_FILE="../src/jupyterhub/ai_infra_jupyterhub_config.py"
    
    if [ -f "$CONFIG_FILE" ]; then
        if grep -q "AIInfraMatrixAuthenticator" "$CONFIG_FILE"; then
            print_success "统一认证器配置检查通过"
            return 0
        else
            print_error "统一认证器未正确配置"
            return 1
        fi
    else
        print_error "JupyterHub配置文件不存在"
        return 1
    fi
}

# 测试9: 检查认证器文件
test_auth_file() {
    print_info "测试9: 检查认证器文件"
    
    AUTH_FILE="../src/jupyterhub/ai_infra_auth.py"
    
    if [ -f "$AUTH_FILE" ]; then
        if grep -q "class AIInfraMatrixAuthenticator" "$AUTH_FILE"; then
            print_success "认证器文件检查通过"
            return 0
        else
            print_error "认证器类未找到"
            return 1
        fi
    else
        print_error "认证器文件不存在"
        return 1
    fi
}

# 运行所有测试
run_all_tests() {
    local failed_tests=0
    
    echo "======================================"
    echo "JupyterHub 统一认证集成测试"
    echo "======================================"
    
    test_backend_health || ((failed_tests++))
    test_create_user || ((failed_tests++))
    test_backend_login || ((failed_tests++))
    test_verify_token || ((failed_tests++))
    test_jupyterhub_login || ((failed_tests++))
    test_refresh_token || ((failed_tests++))
    test_jupyterhub_status || ((failed_tests++))
    test_auth_config || ((failed_tests++))
    test_auth_file || ((failed_tests++))
    
    echo "======================================"
    if [ $failed_tests -eq 0 ]; then
        print_success "所有测试通过! 统一认证系统配置正确"
    else
        print_error "$failed_tests 个测试失败"
        exit 1
    fi
    echo "======================================"
}

# 清理测试数据
cleanup() {
    print_info "清理测试数据..."
    
    if [ -n "$JWT_TOKEN" ]; then
        # 这里可以添加删除测试用户的API调用
        print_info "测试用户将保留以供进一步测试"
    fi
}

# 主函数
main() {
    # 检查依赖
    if ! command -v curl &> /dev/null; then
        print_error "curl 命令未找到，请安装 curl"
        exit 1
    fi
    
    if ! command -v jq &> /dev/null; then
        print_error "jq 命令未找到，请安装 jq"
        exit 1
    fi
    
    # 运行测试
    run_all_tests
    
    # 清理
    cleanup
}

# 如果直接运行此脚本
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
