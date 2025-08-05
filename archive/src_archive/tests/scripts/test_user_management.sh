#!/bin/bash

# 用户管理功能测试脚本
# 测试新实现的用户管理API端点

BASE_URL="http://localhost:8082/api"
ADMIN_TOKEN=""
USER_TOKEN=""

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# 测试API响应
test_api() {
    local method=$1
    local endpoint=$2
    local data=$3
    local token=$4
    local description=$5
    
    echo "Testing: $description"
    
    if [ -n "$token" ]; then
        headers="-H 'Authorization: Bearer $token'"
    else
        headers=""
    fi
    
    if [ -n "$data" ]; then
        response=$(eval curl -s -X $method $headers -H "Content-Type: application/json" -d '$data' "$BASE_URL$endpoint")
    else
        response=$(eval curl -s -X $method $headers "$BASE_URL$endpoint")
    fi
    
    http_code=$(eval curl -s -o /dev/null -w "%{http_code}" -X $method $headers -H "Content-Type: application/json" ${data:+-d '$data'} "$BASE_URL$endpoint")
    
    if [ $http_code -eq 200 ] || [ $http_code -eq 201 ]; then
        log_info "✓ $description - HTTP $http_code"
        echo "Response: $response" | jq . 2>/dev/null || echo "Response: $response"
    else
        log_error "✗ $description - HTTP $http_code"
        echo "Response: $response"
    fi
    
    echo "---"
}

# 1. 登录获取token
echo "=== 1. 用户认证测试 ==="

# 管理员登录
log_info "管理员登录..."
admin_login_data='{"username":"admin","password":"admin123"}'
admin_response=$(curl -s -X POST -H "Content-Type: application/json" -d "$admin_login_data" "$BASE_URL/auth/login")
echo "Admin login response: $admin_response"

if echo "$admin_response" | jq . >/dev/null 2>&1; then
    ADMIN_TOKEN=$(echo "$admin_response" | jq -r '.token // .data.token // empty')
    if [ -n "$ADMIN_TOKEN" ] && [ "$ADMIN_TOKEN" != "null" ]; then
        log_info "管理员登录成功，Token: ${ADMIN_TOKEN:0:20}..."
    else
        log_error "无法获取管理员token"
        exit 1
    fi
else
    log_error "管理员登录失败"
    exit 1
fi

# 创建测试用户
log_info "创建测试用户..."
test_user_data='{"username":"testuser","email":"test@example.com","password":"test123"}'
test_api "POST" "/auth/register" "$test_user_data" "" "创建测试用户"

# 测试用户登录
log_info "测试用户登录..."
user_login_data='{"username":"testuser","password":"test123"}'
user_response=$(curl -s -X POST -H "Content-Type: application/json" -d "$user_login_data" "$BASE_URL/auth/login")
echo "User login response: $user_response"

if echo "$user_response" | jq . >/dev/null 2>&1; then
    USER_TOKEN=$(echo "$user_response" | jq -r '.token // .data.token // empty')
    if [ -n "$USER_TOKEN" ] && [ "$USER_TOKEN" != "null" ]; then
        log_info "测试用户登录成功，Token: ${USER_TOKEN:0:20}..."
    else
        log_error "无法获取用户token"
    fi
else
    log_error "测试用户登录失败"
fi

echo ""

# 2. 用户资料管理测试
echo "=== 2. 用户资料管理测试 ==="

# 获取用户资料
test_api "GET" "/auth/profile" "" "$USER_TOKEN" "获取用户资料"

# 更新用户资料
update_profile_data='{"email":"newemail@example.com","full_name":"Test User"}'
test_api "PUT" "/auth/profile" "$update_profile_data" "$USER_TOKEN" "更新用户资料"

# 修改密码
change_password_data='{"old_password":"test123","new_password":"newtest123"}'
test_api "PUT" "/auth/change-password" "$change_password_data" "$USER_TOKEN" "修改密码"

echo ""

# 3. 管理员用户管理测试
echo "=== 3. 管理员用户管理测试 ==="

# 获取所有用户
test_api "GET" "/users" "" "$ADMIN_TOKEN" "管理员获取用户列表"

# 重置用户密码
reset_password_data='{"new_password":"resetpassword123"}'
test_api "PUT" "/users/2/reset-password" "$reset_password_data" "$ADMIN_TOKEN" "管理员重置用户密码"

# 更新用户组
update_groups_data='{"group_ids":[1,2]}'
test_api "PUT" "/users/2/groups" "$update_groups_data" "$ADMIN_TOKEN" "管理员更新用户组"

echo ""

# 4. RBAC和项目访问控制测试
echo "=== 4. RBAC和项目访问控制测试 ==="

# 获取项目列表（基于用户组权限）
test_api "GET" "/projects" "" "$USER_TOKEN" "普通用户获取项目列表"
test_api "GET" "/projects" "" "$ADMIN_TOKEN" "管理员获取项目列表"

# 检查权限
permission_check_data='{"resource":"projects","action":"create"}'
test_api "POST" "/rbac/check-permission" "$permission_check_data" "$USER_TOKEN" "普通用户权限检查"
test_api "POST" "/rbac/check-permission" "$permission_check_data" "$ADMIN_TOKEN" "管理员权限检查"

echo ""

# 5. 健康检查
echo "=== 5. 系统健康检查 ==="
test_api "GET" "/health" "" "" "系统健康检查"

echo ""

# 6. 管理员专用功能测试
echo "=== 6. 管理员专用功能测试 ==="

# 获取系统统计
test_api "GET" "/admin/stats" "" "$ADMIN_TOKEN" "获取系统统计"

# 获取所有用户详情
test_api "GET" "/admin/users" "" "$ADMIN_TOKEN" "管理员获取用户详情"

# 获取所有项目详情
test_api "GET" "/admin/projects" "" "$ADMIN_TOKEN" "管理员获取项目详情"

echo ""

# 7. 错误场景测试
echo "=== 7. 错误场景测试 ==="

# 无权限访问管理员功能
test_api "GET" "/admin/users" "" "$USER_TOKEN" "普通用户尝试访问管理员功能"

# 无效token访问
test_api "GET" "/auth/profile" "" "invalid_token" "无效token访问"

# 修改不存在的用户密码
reset_nonexist_data='{"new_password":"test123"}'
test_api "PUT" "/users/99999/reset-password" "$reset_nonexist_data" "$ADMIN_TOKEN" "重置不存在用户的密码"

echo ""

log_info "用户管理功能测试完成！"
log_warning "请检查以上输出，确保所有核心功能正常工作"
log_info "预期结果:"
echo "  - 管理员可以查看所有用户信息（除密码外）"
echo "  - 管理员可以重置用户密码"
echo "  - 管理员可以修改用户组"
echo "  - 用户可以修改自己的资料和密码"
echo "  - 基于RBAC的项目访问控制正常工作"
echo "  - 健康检查通过"
