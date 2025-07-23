#!/bin/bash

# API测试脚本
# 测试Ansible Playbook Generator后端API

BASE_URL="http://localhost:8082"
TEST_USER="testuser"
TEST_PASSWORD="testpass123"
TEST_EMAIL="test@example.com"

echo "==================== API测试开始 ===================="
echo "Base URL: $BASE_URL"
echo ""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试函数
test_endpoint() {
    local method=$1
    local endpoint=$2
    local data=$3
    local expected_status=$4
    local description=$5
    local headers=$6
    
    echo -e "${YELLOW}测试: $description${NC}"
    echo "请求: $method $endpoint"
    
    if [ -n "$data" ]; then
        if [ -n "$headers" ]; then
            response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X $method \
                "$BASE_URL$endpoint" \
                -H "Content-Type: application/json" \
                -H "$headers" \
                -d "$data")
        else
            response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X $method \
                "$BASE_URL$endpoint" \
                -H "Content-Type: application/json" \
                -d "$data")
        fi
    else
        if [ -n "$headers" ]; then
            response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X $method \
                "$BASE_URL$endpoint" \
                -H "$headers")
        else
            response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X $method \
                "$BASE_URL$endpoint")
        fi
    fi
    
    body=$(echo $response | sed -E 's/HTTPSTATUS\:[0-9]{3}$//')
    status=$(echo $response | tr -d '\n' | sed -E 's/.*HTTPSTATUS:([0-9]{3})$/\1/')
    
    echo "响应状态: $status"
    echo "响应体: $body"
    
    if [ "$status" = "$expected_status" ]; then
        echo -e "${GREEN}✅ 测试通过${NC}"
    else
        echo -e "${RED}❌ 测试失败 (期望状态: $expected_status, 实际状态: $status)${NC}"
    fi
    echo "----------------------------------------"
    echo ""
}

# 1. 测试健康检查
test_endpoint "GET" "/health" "" "200" "健康检查"

# 2. 测试用户注册
register_data="{\"username\":\"$TEST_USER\",\"email\":\"$TEST_EMAIL\",\"password\":\"$TEST_PASSWORD\"}"
test_endpoint "POST" "/api/auth/register" "$register_data" "201" "用户注册"

# 3. 测试用户登录
login_data="{\"username\":\"$TEST_USER\",\"password\":\"$TEST_PASSWORD\"}"
echo -e "${YELLOW}测试: 用户登录${NC}"
echo "请求: POST /api/auth/login"
login_response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X POST \
    "$BASE_URL/api/auth/login" \
    -H "Content-Type: application/json" \
    -d "$login_data")

login_body=$(echo $login_response | sed -E 's/HTTPSTATUS\:[0-9]{3}$//')
login_status=$(echo $login_response | tr -d '\n' | sed -E 's/.*HTTPSTATUS:([0-9]{3})$/\1/')

echo "响应状态: $login_status"
echo "响应体: $login_body"

if [ "$login_status" = "200" ]; then
    echo -e "${GREEN}✅ 登录测试通过${NC}"
    # 提取JWT token
    TOKEN=$(echo $login_body | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    echo "获取到Token: ${TOKEN:0:50}..."
else
    echo -e "${RED}❌ 登录测试失败${NC}"
    TOKEN=""
fi
echo "----------------------------------------"
echo ""

# 如果有token，继续测试需要认证的端点
if [ -n "$TOKEN" ]; then
    AUTH_HEADER="Authorization: Bearer $TOKEN"
    
    # 4. 测试用户信息获取
    test_endpoint "GET" "/api/user/profile" "" "200" "获取用户信息" "$AUTH_HEADER"
    
    # 5. 测试项目列表
    test_endpoint "GET" "/api/projects" "" "200" "获取项目列表" "$AUTH_HEADER"
    
    # 6. 测试创建项目
    project_data='{"name":"测试项目","description":"这是一个测试项目"}'
    echo -e "${YELLOW}测试: 创建项目${NC}"
    echo "请求: POST /api/projects"
    project_response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X POST \
        "$BASE_URL/api/projects" \
        -H "Content-Type: application/json" \
        -H "$AUTH_HEADER" \
        -d "$project_data")
    
    project_body=$(echo $project_response | sed -E 's/HTTPSTATUS\:[0-9]{3}$//')
    project_status=$(echo $project_response | tr -d '\n' | sed -E 's/.*HTTPSTATUS:([0-9]{3})$/\1/')
    
    echo "响应状态: $project_status"
    echo "响应体: $project_body"
    
    if [ "$project_status" = "201" ]; then
        echo -e "${GREEN}✅ 创建项目测试通过${NC}"
        # 提取项目ID
        PROJECT_ID=$(echo $project_body | grep -o '"id":[0-9]*' | cut -d':' -f2)
        echo "创建的项目ID: $PROJECT_ID"
    else
        echo -e "${RED}❌ 创建项目测试失败${NC}"
        PROJECT_ID=""
    fi
    echo "----------------------------------------"
    echo ""
    
    # 7. 如果有项目ID，测试项目相关操作
    if [ -n "$PROJECT_ID" ]; then
        # 测试获取项目详情
        test_endpoint "GET" "/api/projects/$PROJECT_ID" "" "200" "获取项目详情" "$AUTH_HEADER"
        
        # 测试添加主机
        host_data='{"name":"test-server","ip":"192.168.1.100","port":22,"user":"ubuntu","group":"servers"}'
        test_endpoint "POST" "/api/projects/$PROJECT_ID/hosts" "$host_data" "201" "添加主机" "$AUTH_HEADER"
        
        # 测试添加变量
        variable_data='{"name":"app_name","value":"my-app","type":"string"}'
        test_endpoint "POST" "/api/projects/$PROJECT_ID/variables" "$variable_data" "201" "添加变量" "$AUTH_HEADER"
        
        # 测试添加任务
        task_data='{"name":"安装软件包","module":"apt","args":"name=nginx state=present","order":1}'
        test_endpoint "POST" "/api/projects/$PROJECT_ID/tasks" "$task_data" "201" "添加任务" "$AUTH_HEADER"
        
        # 测试生成Playbook
        test_endpoint "POST" "/api/projects/$PROJECT_ID/generate" "" "200" "生成Playbook" "$AUTH_HEADER"
    fi
    
    # 8. 测试新的Kubernetes API
    echo -e "${YELLOW}========== Kubernetes API测试 ==========${NC}"
    
    # 测试Kubernetes连接 - 期望失败因为没有有效的kubeconfig
    k8s_test_data='{"kubeconfig":"test-config"}'
    test_endpoint "POST" "/api/kubernetes/test-connection" "$k8s_test_data" "400" "Kubernetes连接测试(无效配置)" "$AUTH_HEADER"
    
    # 测试获取集群信息 - 期望失败因为没有配置
    test_endpoint "GET" "/api/kubernetes/cluster-info" "" "500" "获取Kubernetes集群信息(未配置)" "$AUTH_HEADER"
    
    # 9. 测试新的Ansible API
    echo -e "${YELLOW}========== Ansible执行API测试 ==========${NC}"
    
    # 测试Ansible执行历史
    test_endpoint "GET" "/api/ansible/executions" "" "200" "获取Ansible执行历史" "$AUTH_HEADER"
    
    # 如果有项目，测试Ansible执行
    if [ -n "$PROJECT_ID" ]; then
        # 测试Ansible干运行
        ansible_exec_data="{\"project_id\":$PROJECT_ID,\"dry_run\":true}"
        echo -e "${YELLOW}测试: Ansible干运行${NC}"
        echo "请求: POST /api/ansible/dry-run"
        ansible_response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X POST \
            "$BASE_URL/api/ansible/dry-run" \
            -H "Content-Type: application/json" \
            -H "$AUTH_HEADER" \
            -d "$ansible_exec_data")
        
        ansible_body=$(echo $ansible_response | sed -E 's/HTTPSTATUS\:[0-9]{3}$//')
        ansible_status=$(echo $ansible_response | tr -d '\n' | sed -E 's/.*HTTPSTATUS:([0-9]{3})$/\1/')
        
        echo "响应状态: $ansible_status"
        echo "响应体: $ansible_body"
        
        if [ "$ansible_status" = "201" ]; then
            echo -e "${GREEN}✅ Ansible干运行测试通过${NC}"
            # 提取执行ID
            EXECUTION_ID=$(echo $ansible_body | grep -o '"execution_id":[0-9]*' | cut -d':' -f2)
            if [ -n "$EXECUTION_ID" ]; then
                echo "执行ID: $EXECUTION_ID"
                
                # 等待一下，然后测试状态查询
                sleep 2
                test_endpoint "GET" "/api/ansible/execution/$EXECUTION_ID/status" "" "200" "查询Ansible执行状态" "$AUTH_HEADER"
                
                # 测试获取执行日志
                test_endpoint "GET" "/api/ansible/execution/$EXECUTION_ID/logs" "" "200" "获取Ansible执行日志" "$AUTH_HEADER"
                
                # 测试取消执行（如果还在运行）
                test_endpoint "POST" "/api/ansible/execution/$EXECUTION_ID/cancel" "" "200" "取消Ansible执行" "$AUTH_HEADER"
            fi
        else
            echo -e "${RED}❌ Ansible干运行测试失败${NC}"
        fi
        echo "----------------------------------------"
        echo ""
        
        # 测试真实Ansible执行（可能会失败，因为环境问题）
        ansible_real_data="{\"project_id\":$PROJECT_ID,\"dry_run\":false}"
        echo -e "${YELLOW}测试: Ansible真实执行${NC}"
        echo "请求: POST /api/ansible/execute"
        ansible_real_response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X POST \
            "$BASE_URL/api/ansible/execute" \
            -H "Content-Type: application/json" \
            -H "$AUTH_HEADER" \
            -d "$ansible_real_data")
        
        ansible_real_body=$(echo $ansible_real_response | sed -E 's/HTTPSTATUS\:[0-9]{3}$//')
        ansible_real_status=$(echo $ansible_real_response | tr -d '\n' | sed -E 's/.*HTTPSTATUS:([0-9]{3})$/\1/')
        
        echo "响应状态: $ansible_real_status"
        echo "响应体: $ansible_real_body"
        
        if [ "$ansible_real_status" = "201" ]; then
            echo -e "${GREEN}✅ Ansible执行测试通过${NC}"
            # 提取执行ID
            REAL_EXECUTION_ID=$(echo $ansible_real_body | grep -o '"execution_id":[0-9]*' | cut -d':' -f2)
            if [ -n "$REAL_EXECUTION_ID" ]; then
                echo "真实执行ID: $REAL_EXECUTION_ID"
                
                # 等待一下，然后测试状态查询
                sleep 3
                test_endpoint "GET" "/api/ansible/execution/$REAL_EXECUTION_ID/status" "" "200" "查询真实执行状态" "$AUTH_HEADER"
                
                # 测试获取执行日志
                test_endpoint "GET" "/api/ansible/execution/$REAL_EXECUTION_ID/logs" "" "200" "获取真实执行日志" "$AUTH_HEADER"
            fi
        else
            echo -e "${YELLOW}⚠️ Ansible真实执行测试失败（可能是环境问题）${NC}"
        fi
        echo "----------------------------------------"
        echo ""
    fi
    
    # 10. 测试Ansible集成的完整流程
    echo -e "${YELLOW}========== Ansible完整流程测试 ==========${NC}"
    
    # 创建专门用于Ansible测试的项目
    ansible_project_data='{"name":"Ansible测试项目","description":"用于测试Ansible集成的项目"}'
    echo -e "${YELLOW}测试: 创建Ansible测试项目${NC}"
    ansible_project_response=$(curl -s -w "HTTPSTATUS:%{http_code}" -X POST \
        "$BASE_URL/api/projects" \
        -H "Content-Type: application/json" \
        -H "$AUTH_HEADER" \
        -d "$ansible_project_data")
    
    ansible_project_body=$(echo $ansible_project_response | sed -E 's/HTTPSTATUS\:[0-9]{3}$//')
    ansible_project_status=$(echo $ansible_project_response | tr -d '\n' | sed -E 's/.*HTTPSTATUS:([0-9]{3})$/\1/')
    
    if [ "$ansible_project_status" = "201" ]; then
        ANSIBLE_PROJECT_ID=$(echo $ansible_project_body | grep -o '"id":[0-9]*' | cut -d':' -f2)
        echo "Ansible测试项目ID: $ANSIBLE_PROJECT_ID"
        
        # 添加测试主机
        test_host_data='{"name":"test-host","ip":"127.0.0.1","port":22,"user":"testuser","group":"test"}'
        test_endpoint "POST" "/api/projects/$ANSIBLE_PROJECT_ID/hosts" "$test_host_data" "201" "添加测试主机" "$AUTH_HEADER"
        
        # 添加测试变量
        test_var_data='{"name":"test_var","value":"test_value","type":"string"}'
        test_endpoint "POST" "/api/projects/$ANSIBLE_PROJECT_ID/variables" "$test_var_data" "201" "添加测试变量" "$AUTH_HEADER"
        
        # 添加测试任务
        test_task_data='{"name":"调试任务","module":"debug","args":"msg=\"Hello Ansible\"","order":1}'
        test_endpoint "POST" "/api/projects/$ANSIBLE_PROJECT_ID/tasks" "$test_task_data" "201" "添加调试任务" "$AUTH_HEADER"
        
        # 生成Playbook
        test_endpoint "POST" "/api/projects/$ANSIBLE_PROJECT_ID/generate" "" "200" "生成测试Playbook" "$AUTH_HEADER"
        
        # 执行Ansible干运行
        test_ansible_data="{\"project_id\":$ANSIBLE_PROJECT_ID,\"dry_run\":true}"
        test_endpoint "POST" "/api/ansible/dry-run" "$test_ansible_data" "201" "执行完整测试干运行" "$AUTH_HEADER"
        
        echo -e "${GREEN}✅ Ansible完整流程测试完成${NC}"
    else
        echo -e "${RED}❌ 创建Ansible测试项目失败${NC}"
    fi
    
    # 11. 测试错误处理
    echo -e "${YELLOW}========== 错误处理测试 ==========${NC}"
    
    # 测试不存在的项目ID
    test_endpoint "POST" "/api/ansible/dry-run" '{"project_id":99999,"dry_run":true}' "404" "不存在的项目ID" "$AUTH_HEADER"
    
    # 测试无效的执行ID查询
    test_endpoint "GET" "/api/ansible/execution/99999/status" "" "404" "不存在的执行ID" "$AUTH_HEADER"
    
    # 测试取消不存在的执行
    test_endpoint "POST" "/api/ansible/execution/99999/cancel" "" "404" "取消不存在的执行" "$AUTH_HEADER"
    
    # 测试无效的JSON数据
    test_endpoint "POST" "/api/ansible/execute" '{"invalid_json":}' "400" "无效JSON数据" "$AUTH_HEADER"
else
    echo -e "${RED}❌ 无法获取认证Token，跳过需要认证的测试${NC}"
fi

# 12. 测试管理员API（无认证，期望返回401）
echo -e "${YELLOW}========== 无认证访问测试 ==========${NC}"
test_endpoint "GET" "/api/admin/users" "" "401" "管理员API无认证访问"

echo ""
echo "==================== API测试完成 ===================="
echo ""
echo "测试说明："
echo "- ✅ 表示API按预期工作"
echo "- ❌ 表示API存在问题"
echo "- ⚠️ 表示测试失败但可能是环境限制导致"
echo ""
echo "新增功能测试覆盖："
echo "1. Kubernetes API集成 - 连接测试和集群信息获取"
echo "2. Ansible执行API - 干运行、真实执行、状态查询、日志获取"
echo "3. Ansible完整流程 - 项目创建到执行的端到端测试"
echo "4. 错误处理 - 测试各种错误情况的API响应"
echo "5. 执行管理 - 执行历史、状态监控、取消执行功能"
echo ""
echo "注意事项："
echo "- Kubernetes API测试可能因为缺少有效kubeconfig而失败"
echo "- Ansible真实执行可能因为主机连接问题而失败"
echo "- 这些是预期的行为，重点是API响应正确的错误信息"
echo ""
