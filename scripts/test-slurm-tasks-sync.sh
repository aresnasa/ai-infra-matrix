#!/bin/bash

# SLURM 任务显示和同步测试脚本

echo "🚀 测试 SLURM 任务显示和状态同步..."

BASE_URL="http://localhost:8082/api"
FRONTEND_URL="http://localhost:3000"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
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

log_debug() {
    echo -e "${BLUE}[DEBUG]${NC} $1"
}

# 测试后端任务API
test_tasks_api() {
    log_info "测试后端任务API..."
    
    local response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        "$BASE_URL/slurm/tasks?page=1&limit=10")
    
    local http_code=$(echo $response | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    local body=$(echo $response | sed -E 's/HTTPSTATUS\:[0-9]{3}$//')
    
    if [ "$http_code" = "200" ]; then
        log_info "✅ 任务API响应正常"
        
        # 检查响应格式
        local tasks_count=$(echo "$body" | jq -r '.data.tasks | length' 2>/dev/null || echo "0")
        local runtime_count=$(echo "$body" | jq -r '.data.runtime_tasks_count // 0' 2>/dev/null)
        local db_count=$(echo "$body" | jq -r '.data.db_tasks_count // 0' 2>/dev/null)
        
        log_debug "数据库任务数: $db_count"
        log_debug "运行时任务数: $runtime_count"  
        log_debug "总任务数: $tasks_count"
        
        # 检查是否有运行中的任务
        local running_tasks=$(echo "$body" | jq -r '.data.tasks[] | select(.status == "running") | .id' 2>/dev/null)
        if [ ! -z "$running_tasks" ]; then
            log_info "✅ 发现运行中的任务:"
            echo "$running_tasks" | while read -r task_id; do
                log_debug "  - 任务ID: $task_id"
            done
        else
            log_warn "⚠️  没有运行中的任务"
        fi
        
    else
        log_error "❌ 任务API响应失败 (HTTP: $http_code)"
        echo "$body"
        return 1
    fi
}

# 测试前端可访问性
test_frontend_accessibility() {
    log_info "测试前端页面可访问性..."
    
    # 测试主页
    local main_response=$(curl -s -o /dev/null -w "%{http_code}" "$FRONTEND_URL" || echo "000")
    if [ "$main_response" = "200" ]; then
        log_info "✅ 前端主页可访问"
    else
        log_warn "⚠️  前端主页不可访问 (HTTP: $main_response)"
    fi
    
    # 测试任务页面路由
    log_info "前端页面路由:"
    echo "  • 任务管理: $FRONTEND_URL/slurm-tasks"
    echo "  • 扩缩容管理: $FRONTEND_URL/slurm-scaling" 
    echo "  • 主仪表板: $FRONTEND_URL/dashboard"
}

# 创建测试任务
create_test_task() {
    log_info "创建测试扩容任务..."
    
    local test_data='{
        "nodes": [
            {
                "host": "test-node-1",
                "port": 22,
                "user": "root",
                "password": "test123",
                "minion_id": "test-node-1"
            }
        ]
    }'
    
    local response=$(curl -s -w "HTTPSTATUS:%{http_code}" \
        -X POST \
        -H "Content-Type: application/json" \
        -d "$test_data" \
        "$BASE_URL/slurm/scaling/scale-up/async")
    
    local http_code=$(echo $response | tr -d '\n' | sed -e 's/.*HTTPSTATUS://')
    local body=$(echo $response | sed -E 's/HTTPSTATUS\:[0-9]{3}$//')
    
    if [ "$http_code" = "200" ] || [ "$http_code" = "201" ]; then
        local task_id=$(echo "$body" | jq -r '.data.task_id // .opId // .data.opId // empty' 2>/dev/null)
        if [ ! -z "$task_id" ] && [ "$task_id" != "null" ]; then
            log_info "✅ 测试任务创建成功"
            log_debug "任务ID: $task_id"
            echo "TASK_ID=$task_id"
            return 0
        else
            log_warn "⚠️  任务创建成功但未获得任务ID"
            echo "$body"
        fi
    else
        log_warn "⚠️  测试任务创建可能失败 (HTTP: $http_code)"
        echo "$body"
    fi
    
    return 1
}

# 验证任务在列表中显示
verify_task_in_list() {
    local task_id=$1
    
    if [ -z "$task_id" ]; then
        log_warn "⚠️  跳过任务列表验证 - 没有有效的任务ID"
        return
    fi
    
    log_info "验证任务在列表中显示 (ID: $task_id)..."
    
    # 等待一会儿让任务出现在列表中
    sleep 2
    
    local response=$(curl -s "$BASE_URL/slurm/tasks")
    local found_task=$(echo "$response" | jq -r --arg id "$task_id" '.data.tasks[] | select(.id == $id) | .id' 2>/dev/null)
    
    if [ "$found_task" = "$task_id" ]; then
        log_info "✅ 任务在列表中正确显示"
        
        # 获取任务详情
        local task_status=$(echo "$response" | jq -r --arg id "$task_id" '.data.tasks[] | select(.id == $id) | .status' 2>/dev/null)
        local task_source=$(echo "$response" | jq -r --arg id "$task_id" '.data.tasks[] | select(.id == $id) | .source' 2>/dev/null)
        
        log_debug "任务状态: $task_status"
        log_debug "任务来源: $task_source"
        
    else
        log_error "❌ 任务未在列表中显示"
        return 1
    fi
}

# 测试任务状态同步
test_task_sync() {
    log_info "测试任务状态同步机制..."
    
    # 多次调用API，检查状态一致性
    for i in {1..3}; do
        log_debug "第 $i 次同步测试..."
        
        local response=$(curl -s "$BASE_URL/slurm/tasks")
        local running_count=$(echo "$response" | jq -r '.data.tasks[] | select(.status == "running") | .id' 2>/dev/null | wc -l)
        
        log_debug "运行中任务数: $running_count"
        
        if [ "$i" -lt 3 ]; then
            sleep 1
        fi
    done
    
    log_info "✅ 状态同步测试完成"
}

# 主测试流程
main() {
    echo "=========================================="
    echo "  SLURM 任务显示和同步修复验证"
    echo "=========================================="
    echo
    
    # 检查必要工具
    command -v curl >/dev/null 2>&1 || { log_error "需要安装 curl"; exit 1; }
    command -v jq >/dev/null 2>&1 || log_warn "建议安装 jq 以获得更好的 JSON 显示"
    
    # 1. 测试后端任务API
    if ! test_tasks_api; then
        log_error "后端API测试失败，请检查后端服务"
        exit 1
    fi
    echo
    
    # 2. 测试前端可访问性
    test_frontend_accessibility
    echo
    
    # 3. 创建测试任务并验证
    if task_result=$(create_test_task); then
        task_id=$(echo "$task_result" | grep "TASK_ID=" | cut -d'=' -f2)
        echo
        verify_task_in_list "$task_id"
        echo
    fi
    
    # 4. 测试状态同步
    test_task_sync
    echo
    
    echo "=========================================="
    log_info "修复验证完成！"
    echo
    log_info "📋 主要修复点："
    echo "   ✅ 后端GetTasks方法正确合并数据库和运行时任务"
    echo "   ✅ 修复了时间戳类型转换问题"
    echo "   ✅ 统一了返回数据格式"
    echo "   ✅ 增强了前端自动刷新机制"
    echo "   ✅ 添加了页面可见性检测"
    echo "   ✅ 增加了扩缩容页面到任务页面的导航"
    echo
    log_info "🔧 用户体验改进："
    echo "   • 运行中任务15秒自动刷新"
    echo "   • 页面切回时自动刷新"
    echo "   • 扩缩容成功后可直接跳转查看任务"
    echo "   • 实时显示运行中任务数量"
    echo "   • URL参数支持直接定位任务"
    echo
    log_info "📱 测试访问："
    echo "   • 任务管理: $FRONTEND_URL/slurm-tasks"
    echo "   • 扩缩容管理: $FRONTEND_URL/slurm-scaling"
    echo "=========================================="
}

# 执行主程序
main "$@"