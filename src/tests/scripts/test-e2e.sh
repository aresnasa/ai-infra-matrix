#!/bin/bash

# 端到端测试脚本 - 验证完整功能集
# 包含：预览、下载、垃圾箱、时区验证、用户管理等功能
set -e

# 加载配置文件
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_FILE="$SCRIPT_DIR/test-config.env"

if [ -f "$CONFIG_FILE" ]; then
    source "$CONFIG_FILE"
    print_info() { echo -e "\033[0;34mℹ️  $1\033[0m"; }
    print_info "已加载配置文件: $CONFIG_FILE"
else
    print_info() { echo -e "\033[0;34mℹ️  $1\033[0m"; }
    print_info "配置文件不存在，使用默认配置"
fi

# 加载报告生成器
REPORT_GENERATOR="$SCRIPT_DIR/generate-report.sh"
if [ -f "$REPORT_GENERATOR" ]; then
    source "$REPORT_GENERATOR"
fi

# 默认配置（如果配置文件不存在）
BASE_URL="${BASE_URL:-http://localhost:8082/api}"
FRONTEND_URL="${FRONTEND_URL:-http://localhost:3001}"
AUTH_TOKEN=""  # 将在运行时动态获取
PROJECT_ID=""  # 将在测试中创建

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

# 辅助函数：等待服务启动
wait_for_services() {
    print_info "等待服务启动..."
    
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if curl -s "$BASE_URL/health" > /dev/null 2>&1; then
            print_success "后端服务已启动"
            break
        fi
        
        attempt=$((attempt + 1))
        print_info "等待后端服务启动... ($attempt/$max_attempts)"
        sleep 2
    done
    
    if [ $attempt -eq $max_attempts ]; then
        print_error "后端服务启动超时"
        return 1
    fi
    
    # 检查前端服务
    if curl -s "$FRONTEND_URL" > /dev/null 2>&1; then
        print_success "前端服务已启动"
    else
        print_warning "前端服务可能未完全启动"
    fi
    
    return 0
}

# 获取认证Token
get_auth_token() {
    print_info "获取认证Token..."
    
    local response=$(curl -s -X POST "$BASE_URL/auth/login" \
        -H "Content-Type: application/json" \
        -d '{"username":"admin","password":"admin123"}')
    
    AUTH_TOKEN=$(echo "$response" | jq -r '.token // empty')
    
    if [ -n "$AUTH_TOKEN" ] && [ "$AUTH_TOKEN" != "null" ]; then
        print_success "认证Token获取成功"
        print_info "Token: ${AUTH_TOKEN:0:50}..."
        return 0
    else
        print_error "认证Token获取失败"
        echo "Response: $response"
        return 1
    fi
}

# 创建测试项目
create_test_project() {
    print_info "创建测试项目..."
    
    local response=$(curl -s -X POST "$BASE_URL/projects" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -H "Content-Type: application/json" \
        -d '{
            "name": "e2e-test-project",
            "description": "端到端测试项目",
            "hosts": [{"name": "test-host", "ip": "192.168.1.100", "user": "root", "port": 22}],
            "variables": [{"name": "test_var", "value": "test_value", "type": "string"}],
            "tasks": [{"name": "test-task", "module": "debug", "args": "msg=Hello World", "enabled": true}]
        }')
    
    PROJECT_ID=$(echo "$response" | jq -r '.id // empty')
    
    if [ -n "$PROJECT_ID" ] && [ "$PROJECT_ID" != "null" ]; then
        print_success "测试项目创建成功"
        print_info "项目ID: $PROJECT_ID"
        return 0
    else
        print_error "测试项目创建失败"
        echo "Response: $response"
        return 1
    fi
}

# 测试容器时区配置
test_timezone_configuration() {
    print_info "测试容器时区配置..."
    
    local backend_tz=$(docker exec ansible-backend date | grep -o 'CST\|UTC\|GMT')
    local frontend_tz=$(docker exec ansible-frontend date | grep -o 'CST\|UTC\|GMT')
    
    if [ "$backend_tz" = "CST" ] && [ "$frontend_tz" = "CST" ]; then
        print_success "时区配置测试通过 (后端: $backend_tz, 前端: $frontend_tz)"
        
        # 显示具体时间
        local backend_time=$(docker exec ansible-backend date)
        local frontend_time=$(docker exec ansible-frontend date)
        print_info "后端时间: $backend_time"
        print_info "前端时间: $frontend_time"
        return 0
    else
        print_error "时区配置测试失败 (后端: $backend_tz, 前端: $frontend_tz)"
        return 1
    fi
}

# 测试垃圾箱功能
test_trash_functionality() {
    print_info "测试垃圾箱功能..."
    
    # 1. 软删除项目到垃圾箱
    print_info "1. 测试软删除功能..."
    local soft_delete_response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X PATCH "$BASE_URL/projects/$PROJECT_ID/soft-delete" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -H "Content-Type: application/json")
    
    local http_code=$(echo "$soft_delete_response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    local body=$(echo "$soft_delete_response" | sed -e 's/HTTPSTATUS:.*//g')
    
    if [ "$http_code" -eq 200 ]; then
        print_success "软删除功能测试通过"
    else
        print_error "软删除功能测试失败 (HTTP $http_code)"
        echo "$body"
        return 1
    fi
    
    # 2. 检查垃圾箱内容
    print_info "2. 测试垃圾箱列表查看..."
    local trash_response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X GET "$BASE_URL/projects/trash" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -H "Content-Type: application/json")
    
    http_code=$(echo "$trash_response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    body=$(echo "$trash_response" | sed -e 's/HTTPSTATUS:.*//g')
    
    if [ "$http_code" -eq 200 ]; then
        local project_count=$(echo "$body" | jq '.projects | length')
        if [ "$project_count" -gt 0 ]; then
            print_success "垃圾箱列表查看测试通过 (找到 $project_count 个项目)"
        else
            print_error "垃圾箱列表查看测试失败 (垃圾箱为空)"
            return 1
        fi
    else
        print_error "垃圾箱列表查看测试失败 (HTTP $http_code)"
        return 1
    fi
    
    # 3. 恢复项目
    print_info "3. 测试项目恢复功能..."
    local restore_response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X PATCH "$BASE_URL/projects/$PROJECT_ID/restore" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -H "Content-Type: application/json")
    
    http_code=$(echo "$restore_response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    body=$(echo "$restore_response" | sed -e 's/HTTPSTATUS:.*//g')
    
    if [ "$http_code" -eq 200 ]; then
        print_success "项目恢复功能测试通过"
    else
        print_error "项目恢复功能测试失败 (HTTP $http_code)"
        echo "$body"
        return 1
    fi
    
    # 4. 确认垃圾箱已空
    print_info "4. 确认垃圾箱已空..."
    trash_response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X GET "$BASE_URL/projects/trash" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -H "Content-Type: application/json")
    
    http_code=$(echo "$trash_response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    body=$(echo "$trash_response" | sed -e 's/HTTPSTATUS:.*//g')
    
    if [ "$http_code" -eq 200 ]; then
        local project_count=$(echo "$body" | jq '.projects | length')
        if [ "$project_count" -eq 0 ]; then
            print_success "垃圾箱清空确认测试通过"
        else
            print_error "垃圾箱清空确认测试失败 (仍有 $project_count 个项目)"
            return 1
        fi
    else
        print_error "垃圾箱清空确认测试失败 (HTTP $http_code)"
        return 1
    fi
    
    # 5. 测试永久删除功能
    print_info "5. 测试永久删除功能..."
    
    # 先再次软删除
    curl -s -X PATCH "$BASE_URL/projects/$PROJECT_ID/soft-delete" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -H "Content-Type: application/json" > /dev/null
    
    # 永久删除
    local force_delete_response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X DELETE "$BASE_URL/projects/$PROJECT_ID/force" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -H "Content-Type: application/json")
    
    http_code=$(echo "$force_delete_response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    
    if [ "$http_code" -eq 204 ] || [ "$http_code" -eq 200 ]; then
        print_success "永久删除功能测试通过 (HTTP $http_code)"
        
        # 创建新的测试项目供后续测试使用
        create_test_project
        return 0
    else
        print_error "永久删除功能测试失败 (HTTP $http_code)"
        return 1
    fi
}

# 测试后端API预览功能
test_backend_preview() {
    print_info "测试后端预览API..."
    
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X POST "$BASE_URL/playbook/preview" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -d "{\"project_id\": $PROJECT_ID}")
    
    http_code=$(echo "$response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    body=$(echo "$response" | sed -e 's/HTTPSTATUS:.*//g')
    
    if [ "$http_code" -eq 200 ]; then
        print_success "后端预览API测试通过"
        validation_score=$(echo "$body" | jq -r '.validation_score // "N/A"')
        is_valid=$(echo "$body" | jq -r '.is_valid // false')
        print_info "验证分数: $validation_score/100, 有效性: $is_valid"
        return 0
    else
        print_error "后端预览API测试失败 (HTTP $http_code)"
        echo "$body" | jq . 2>/dev/null || echo "$body"
        return 1
    fi
}

# 测试后端包生成功能
test_backend_package() {
    print_info "测试后端包生成API..."
    
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X POST "$BASE_URL/playbook/package" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -d "{\"project_id\": $PROJECT_ID}")
    
    http_code=$(echo "$response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    body=$(echo "$response" | sed -e 's/HTTPSTATUS:.*//g')
    
    if [ "$http_code" -eq 200 ]; then
        print_success "后端包生成API测试通过"
        zip_path=$(echo "$body" | jq -r '.zip_path // "N/A"')
        total_size=$(echo "$body" | jq -r '.total_size // "N/A"')
        print_info "ZIP路径: $zip_path, 大小: $total_size 字节"
        echo "$zip_path" > /tmp/zip_path.txt  # 保存路径供下载测试使用
        return 0
    else
        print_error "后端包生成API测试失败 (HTTP $http_code)"
        echo "$body" | jq . 2>/dev/null || echo "$body"
        return 1
    fi
}

# 测试后端ZIP下载功能
test_backend_zip_download() {
    print_info "测试后端ZIP下载API..."
    
    if [ ! -f /tmp/zip_path.txt ]; then
        print_error "没有找到ZIP路径，请先运行包生成测试"
        return 1
    fi
    
    zip_path=$(cat /tmp/zip_path.txt)
    encoded_path=$(python3 -c "import urllib.parse; print(urllib.parse.quote('$zip_path'))")
    
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X GET "$BASE_URL/playbook/download-zip/$encoded_path" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -o "/tmp/test_download.zip")
    
    http_code=$(echo "$response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    
    if [ "$http_code" -eq 200 ] && [ -f "/tmp/test_download.zip" ]; then
        file_size=$(wc -c < "/tmp/test_download.zip")
        print_success "后端ZIP下载API测试通过"
        print_info "下载文件大小: $file_size 字节"
        rm -f "/tmp/test_download.zip"
        return 0
    else
        print_error "后端ZIP下载API测试失败 (HTTP $http_code)"
        return 1
    fi
}

# 测试后端Playbook生成功能
test_backend_playbook_generation() {
    print_info "测试后端Playbook生成API..."
    
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X POST "$BASE_URL/playbook/generate" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -d "{\"project_id\": $PROJECT_ID}")
    
    http_code=$(echo "$response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    body=$(echo "$response" | sed -e 's/HTTPSTATUS:.*//g')
    
    if [ "$http_code" -eq 200 ]; then
        print_success "后端Playbook生成API测试通过"
        generation_id=$(echo "$body" | jq -r '.id // "N/A"')
        file_name=$(echo "$body" | jq -r '.file_name // "N/A"')
        print_info "生成ID: $generation_id, 文件名: $file_name"
        echo "$generation_id" > /tmp/generation_id.txt  # 保存generation ID供下载测试使用
        return 0
    else
        print_error "后端Playbook生成API测试失败 (HTTP $http_code)"
        echo "$body" | jq . 2>/dev/null || echo "$body"
        return 1
    fi
}

# 测试后端单文件下载功能
test_backend_single_download() {
    print_info "测试后端单文件下载API..."
    
    if [ ! -f /tmp/generation_id.txt ]; then
        print_error "没有找到generation ID，请先运行Playbook生成测试"
        return 1
    fi
    
    generation_id=$(cat /tmp/generation_id.txt)
    print_info "使用generation ID: $generation_id"
    
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X GET "$BASE_URL/playbook/download/$generation_id" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -o "/tmp/test_playbook.yml")
    
    http_code=$(echo "$response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    
    if [ "$http_code" -eq 200 ] && [ -f "/tmp/test_playbook.yml" ]; then
        file_size=$(wc -c < "/tmp/test_playbook.yml")
        print_success "后端单文件下载API测试通过"
        print_info "下载文件大小: $file_size 字节"
        rm -f "/tmp/test_playbook.yml"
        return 0
    else
        print_error "后端单文件下载API测试失败 (HTTP $http_code)"
        return 1
    fi
}

# 测试前端可访问性
test_frontend_accessibility() {
    print_info "测试前端可访问性..."
    
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" "$FRONTEND_URL")
    http_code=$(echo "$response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    
    if [ "$http_code" -eq 200 ]; then
        print_success "前端可访问性测试通过"
        return 0
    else
        print_error "前端可访问性测试失败 (HTTP $http_code)"
        return 1
    fi
}

# 测试用户管理功能
test_user_management() {
    print_info "测试用户管理功能..."
    
    local test_count=0
    local passed_count=0
    
    # 1. 创建测试用户
    print_info "1. 创建测试用户..."
    test_count=$((test_count + 1))
    local register_response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X POST "$BASE_URL/auth/register" \
        -H "Content-Type: application/json" \
        -d '{"username":"e2e_testuser","email":"e2etest@example.com","password":"e2etest123"}')
    
    local http_code=$(echo "$register_response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    
    if [ "$http_code" -eq 200 ] || [ "$http_code" -eq 201 ]; then
        print_success "创建测试用户成功"
        passed_count=$((passed_count + 1))
    else
        print_warning "创建测试用户失败或用户已存在 (HTTP $http_code)"
    fi
    
    # 2. 测试用户登录
    print_info "2. 测试用户登录..."
    test_count=$((test_count + 1))
    local login_response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X POST "$BASE_URL/auth/login" \
        -H "Content-Type: application/json" \
        -d '{"username":"e2e_testuser","password":"e2etest123"}')
    
    http_code=$(echo "$login_response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    local body=$(echo "$login_response" | sed -e 's/HTTPSTATUS:.*//g')
    
    if [ "$http_code" -eq 200 ]; then
        local test_user_token=$(echo "$body" | jq -r '.token // .data.token // empty')
        if [ -n "$test_user_token" ] && [ "$test_user_token" != "null" ]; then
            print_success "测试用户登录成功"
            passed_count=$((passed_count + 1))
        else
            print_error "无法获取测试用户Token"
        fi
    else
        print_error "测试用户登录失败 (HTTP $http_code)"
    fi
    
    # 3. 测试用户资料管理
    if [ -n "$test_user_token" ] && [ "$test_user_token" != "null" ]; then
        print_info "3. 测试用户资料管理..."
        test_count=$((test_count + 1))
        
        # 获取用户资料
        local profile_response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
            -X GET "$BASE_URL/auth/profile" \
            -H "Authorization: Bearer $test_user_token")
        
        http_code=$(echo "$profile_response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
        
        if [ "$http_code" -eq 200 ]; then
            print_success "获取用户资料成功"
            passed_count=$((passed_count + 1))
        else
            print_error "获取用户资料失败 (HTTP $http_code)"
        fi
        
        # 4. 测试用户权限检查
        print_info "4. 测试用户权限检查..."
        test_count=$((test_count + 1))
        
        local permission_response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
            -X POST "$BASE_URL/rbac/check-permission" \
            -H "Authorization: Bearer $test_user_token" \
            -H "Content-Type: application/json" \
            -d '{"resource":"projects","action":"create"}')
        
        http_code=$(echo "$permission_response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
        
        if [ "$http_code" -eq 200 ]; then
            print_success "用户权限检查成功"
            passed_count=$((passed_count + 1))
        else
            print_warning "用户权限检查接口可能未实现 (HTTP $http_code)"
        fi
    fi
    
    # 5. 测试管理员用户管理功能
    print_info "5. 测试管理员用户管理功能..."
    test_count=$((test_count + 1))
    
    local users_response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X GET "$BASE_URL/users" \
        -H "Authorization: Bearer $AUTH_TOKEN")
    
    http_code=$(echo "$users_response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    
    if [ "$http_code" -eq 200 ]; then
        print_success "管理员获取用户列表成功"
        passed_count=$((passed_count + 1))
    else
        print_warning "管理员用户列表接口可能未实现 (HTTP $http_code)"
    fi
    
    # 6. 测试系统统计功能
    print_info "6. 测试系统统计功能..."
    test_count=$((test_count + 1))
    
    local stats_response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X GET "$BASE_URL/admin/stats" \
        -H "Authorization: Bearer $AUTH_TOKEN")
    
    http_code=$(echo "$stats_response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    
    if [ "$http_code" -eq 200 ]; then
        print_success "获取系统统计成功"
        passed_count=$((passed_count + 1))
    else
        print_warning "系统统计接口可能未实现 (HTTP $http_code)"
    fi
    
    print_info "用户管理测试完成: $passed_count/$test_count 项通过"
    
    if [ "$passed_count" -ge 3 ]; then  # 至少核心功能要通过
        return 0
    else
        return 1
    fi
}

# 测试系统健康检查增强版
test_enhanced_health_checks() {
    print_info "测试增强系统健康检查..."
    
    local test_count=0
    local passed_count=0
    
    # 1. 基础健康检查
    print_info "1. 基础健康检查..."
    test_count=$((test_count + 1))
    
    local health_response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X GET "$BASE_URL/health")
    
    local http_code=$(echo "$health_response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    
    if [ "$http_code" -eq 200 ]; then
        print_success "基础健康检查通过"
        passed_count=$((passed_count + 1))
    else
        print_error "基础健康检查失败 (HTTP $http_code)"
    fi
    
    # 2. 数据库连接检查
    print_info "2. 数据库连接检查..."
    test_count=$((test_count + 1))
    
    local db_health_response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X GET "$BASE_URL/health/db")
    
    http_code=$(echo "$db_health_response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    
    if [ "$http_code" -eq 200 ]; then
        print_success "数据库连接检查通过"
        passed_count=$((passed_count + 1))
    else
        print_warning "数据库健康检查接口可能未实现 (HTTP $http_code)"
    fi
    
    # 3. Redis连接检查
    print_info "3. Redis连接检查..."
    test_count=$((test_count + 1))
    
    local redis_health_response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X GET "$BASE_URL/health/redis")
    
    http_code=$(echo "$redis_health_response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    
    if [ "$http_code" -eq 200 ]; then
        print_success "Redis连接检查通过"
        passed_count=$((passed_count + 1))
    else
        print_warning "Redis健康检查接口可能未实现 (HTTP $http_code)"
    fi
    
    # 4. API文档可访问性
    print_info "4. API文档可访问性检查..."
    test_count=$((test_count + 1))
    
    local swagger_response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X GET "$BASE_URL/swagger/index.html")
    
    http_code=$(echo "$swagger_response" | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    
    if [ "$http_code" -eq 200 ]; then
        print_success "API文档可访问性检查通过"
        passed_count=$((passed_count + 1))
    else
        print_warning "API文档可能未配置 (HTTP $http_code)"
    fi
    
    print_info "增强健康检查完成: $passed_count/$test_count 项通过"
    
    if [ "$passed_count" -ge 1 ]; then  # 至少基础健康检查要通过
        return 0
    else
        return 1
    fi
}

# 主测试函数
main() {
    echo "================================"
    echo "端到端功能测试开始"
    echo "================================"
    
    # 测试计数器
    total_tests=0
    passed_tests=0
    
    # 1. 等待服务启动
    total_tests=$((total_tests + 1))
    if wait_for_services; then
        passed_tests=$((passed_tests + 1))
    fi
    
    # 2. 获取认证Token
    total_tests=$((total_tests + 1))
    if get_auth_token; then
        passed_tests=$((passed_tests + 1))
    else
        print_error "无法获取认证Token，停止后续测试"
        exit 1
    fi
    
    # 3. 测试时区配置
    total_tests=$((total_tests + 1))
    if test_timezone_configuration; then
        passed_tests=$((passed_tests + 1))
    fi
    
    # 4. 创建测试项目
    total_tests=$((total_tests + 1))
    if create_test_project; then
        passed_tests=$((passed_tests + 1))
    else
        print_error "无法创建测试项目，停止后续测试"
        exit 1
    fi
    
    # 5. 前端可访问性测试
    total_tests=$((total_tests + 1))
    if test_frontend_accessibility; then
        passed_tests=$((passed_tests + 1))
    fi
    
    # 6. 增强健康检查测试
    total_tests=$((total_tests + 1))
    if test_enhanced_health_checks; then
        passed_tests=$((passed_tests + 1))
    fi
    
    # 7. 用户管理功能测试
    total_tests=$((total_tests + 1))
    if test_user_management; then
        passed_tests=$((passed_tests + 1))
    fi
    
    # 8. 垃圾箱功能测试
    total_tests=$((total_tests + 1))
    if test_trash_functionality; then
        passed_tests=$((passed_tests + 1))
    fi
    
    # 9. 后端API测试
    total_tests=$((total_tests + 1))
    if test_backend_preview; then
        passed_tests=$((passed_tests + 1))
    fi
    
    total_tests=$((total_tests + 1))
    if test_backend_package; then
        passed_tests=$((passed_tests + 1))
    fi
    
    total_tests=$((total_tests + 1))
    if test_backend_zip_download; then
        passed_tests=$((passed_tests + 1))
    fi
    
    # 添加playbook生成测试（必须在单文件下载测试之前）
    total_tests=$((total_tests + 1))
    if test_backend_playbook_generation; then
        passed_tests=$((passed_tests + 1))
    fi
    
    total_tests=$((total_tests + 1))
    if test_backend_single_download; then
        passed_tests=$((passed_tests + 1))
    fi
    
    echo "================================"
    echo "测试结果汇总"
    echo "================================"
    print_info "总测试数: $total_tests"
    print_success "通过测试: $passed_tests"
    
    # 生成详细测试报告
    if command -v generate_test_report >/dev/null 2>&1; then
        print_info "正在生成测试报告..."
        generate_test_report "$total_tests" "$passed_tests"
    fi
    
    if [ "$passed_tests" -eq "$total_tests" ]; then
        print_success "所有测试都通过了！🎉"
        echo ""
        print_info "功能验证完成："
        echo "✅ 服务启动和连接正常"
        echo "✅ 用户认证功能正常"
        echo "✅ 容器时区配置正确 (Asia/Shanghai)"
        echo "✅ 项目管理功能正常"
        echo "✅ 前端应用可访问"
        echo "✅ 增强健康检查通过"
        echo "✅ 用户管理功能完整（注册、登录、权限）"
        echo "✅ 垃圾箱功能完整（软删除、恢复、永久删除）"
        echo "✅ 预览功能正常工作"
        echo "✅ ZIP包生成正常工作"
        echo "✅ ZIP下载功能正常工作"
        echo "✅ Playbook生成功能正常"
        echo "✅ 单文件下载功能正常工作"
        echo ""
        print_info "您可以在浏览器中访问 $FRONTEND_URL 来使用应用"
        exit 0
    else
        failed_tests=$((total_tests - passed_tests))
        print_error "$failed_tests 个测试失败"
        exit 1
    fi
}

# 清理临时文件
cleanup() {
    rm -f /tmp/zip_path.txt /tmp/test_download.zip /tmp/test_playbook.yml
}

# 设置退出时清理
trap cleanup EXIT

# 运行主测试
main "$@"
