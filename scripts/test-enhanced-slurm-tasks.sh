#!/bin/bash

# SLURM 任务管理系统测试脚本
# 测试增强后的任务管理功能

echo "🚀 开始测试 SLURM 任务管理系统增强功能..."

BASE_URL="http://localhost:8082/api"
TOKEN=""

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查后端服务
check_backend() {
    log_info "检查后端服务状态..."
    
    response=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/health" || echo "000")
    
    if [ "$response" = "200" ]; then
        log_info "✅ 后端服务运行正常"
        return 0
    else
        log_error "❌ 后端服务不可用 (HTTP: $response)"
        return 1
    fi
}

# 测试任务列表 API
test_get_tasks() {
    log_info "测试获取任务列表..."
    
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -H "Authorization: Bearer $TOKEN" \
        "$BASE_URL/slurm/tasks?page=1&limit=10")
    
    http_code=$(echo $response | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    body=$(echo $response | sed -E 's/HTTPSTATUS\:[0-9]{3}$//')
    
    if [ "$http_code" = "200" ]; then
        log_info "✅ 任务列表获取成功"
        echo "响应: $body" | jq . 2>/dev/null || echo "$body"
    else
        log_error "❌ 任务列表获取失败 (HTTP: $http_code)"
        echo "响应: $body"
    fi
}

# 测试任务统计 API
test_get_statistics() {
    log_info "测试获取任务统计..."
    
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -H "Authorization: Bearer $TOKEN" \
        "$BASE_URL/slurm/tasks/statistics")
    
    http_code=$(echo $response | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    body=$(echo $response | sed -E 's/HTTPSTATUS\:[0-9]{3}$//')
    
    if [ "$http_code" = "200" ]; then
        log_info "✅ 任务统计获取成功"
        echo "响应: $body" | jq . 2>/dev/null || echo "$body"
    else
        log_error "❌ 任务统计获取失败 (HTTP: $http_code)"
        echo "响应: $body"
    fi
}

# 测试创建任务 (通过扩容操作)
test_create_task() {
    log_info "测试创建任务（通过扩容操作）..."
    
    task_data='{
        "nodes": [
            {
                "name": "test-node-1",
                "cpu": 4,
                "memory": "8Gi",
                "ssh": {
                    "host": "192.168.1.100",
                    "port": 22,
                    "user": "root",
                    "password": "test123"
                }
            }
        ]
    }'
    
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X POST \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d "$task_data" \
        "$BASE_URL/slurm/scaling/scale-up/async")
    
    http_code=$(echo $response | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    body=$(echo $response | sed -E 's/HTTPSTATUS\:[0-9]{3}$//')
    
    if [ "$http_code" = "200" ] || [ "$http_code" = "201" ]; then
        log_info "✅ 任务创建成功"
        echo "响应: $body" | jq . 2>/dev/null || echo "$body"
        
        # 提取任务ID
        task_id=$(echo "$body" | jq -r '.data.task_id // .data.id // empty' 2>/dev/null)
        if [ ! -z "$task_id" ] && [ "$task_id" != "null" ]; then
            echo "TASK_ID=$task_id"
            return 0
        fi
    else
        log_warn "⚠️  任务创建可能失败 (HTTP: $http_code)"
        echo "响应: $body"
    fi
    
    return 1
}

# 测试任务详情 API
test_get_task_detail() {
    local task_id=$1
    
    if [ -z "$task_id" ]; then
        log_warn "⚠️  跳过任务详情测试 - 没有有效的任务ID"
        return
    fi
    
    log_info "测试获取任务详情 (ID: $task_id)..."
    
    response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -H "Authorization: Bearer $TOKEN" \
        "$BASE_URL/slurm/tasks/$task_id/detail")
    
    http_code=$(echo $response | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    body=$(echo $response | sed -E 's/HTTPSTATUS\:[0-9]{3}$//')
    
    if [ "$http_code" = "200" ]; then
        log_info "✅ 任务详情获取成功"
        echo "响应: $body" | jq . 2>/dev/null || echo "$body"
    else
        log_error "❌ 任务详情获取失败 (HTTP: $http_code)"
        echo "响应: $body"
    fi
}

# 测试前端可访问性
test_frontend() {
    log_info "测试前端页面可访问性..."
    
    # 测试主页
    response=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:3000" || echo "000")
    
    if [ "$response" = "200" ]; then
        log_info "✅ 前端服务运行正常"
    else
        log_warn "⚠️  前端服务不可用 (HTTP: $response)"
        log_info "请确保前端开发服务器正在运行: npm start"
    fi
}

# 主测试流程
main() {
    echo "=========================================="
    echo "  SLURM 任务管理系统功能测试"
    echo "=========================================="
    echo
    
    # 检查必要工具
    command -v curl >/dev/null 2>&1 || { log_error "需要安装 curl"; exit 1; }
    command -v jq >/dev/null 2>&1 || log_warn "建议安装 jq 以获得更好的 JSON 显示"
    
    # 检查后端服务
    if ! check_backend; then
        log_error "后端服务不可用，请启动后端服务后重试"
        exit 1
    fi
    
    echo
    log_info "开始 API 功能测试..."
    echo
    
    # 运行测试
    test_get_tasks
    echo
    
    test_get_statistics  
    echo
    
    # 尝试创建任务并获取详情
    if task_result=$(test_create_task); then
        task_id=$(echo "$task_result" | grep "TASK_ID=" | cut -d'=' -f2)
        echo
        test_get_task_detail "$task_id"
    fi
    echo
    
    # 测试前端
    test_frontend
    echo
    
    echo "=========================================="
    log_info "测试完成！"
    echo
    log_info "📋 前端访问地址："
    echo "   • 任务管理页面: http://localhost:3000/slurm-tasks"
    echo "   • 主仪表板: http://localhost:3000/dashboard"
    echo
    log_info "🔧 API 端点："
    echo "   • GET  $BASE_URL/slurm/tasks - 任务列表"
    echo "   • GET  $BASE_URL/slurm/tasks/statistics - 统计信息"
    echo "   • GET  $BASE_URL/slurm/tasks/{id}/detail - 任务详情"
    echo "   • POST $BASE_URL/slurm/tasks/{id}/cancel - 取消任务"
    echo "   • POST $BASE_URL/slurm/tasks/{id}/retry - 重试任务"
    echo "=========================================="
}

# 执行主程序
main "$@"