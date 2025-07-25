#!/bin/bash

# AI助手异步架构测试脚本
# 用于验证新的消息队列和缓存功能

set -e

# 配置
BASE_URL="http://localhost:8082/api"
TEST_USER="admin"
TEST_PASSWORD="admin123"
TOKEN=""

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查服务状态
check_service() {
    log_info "检查服务健康状态..."
    
    # 检查基础健康状态
    response=$(curl -s -w "%{http_code}" -o /tmp/health.json "$BASE_URL/health")
    if [ "$response" = "200" ]; then
        log_success "基础服务健康检查通过"
    else
        log_error "基础服务健康检查失败 (HTTP $response)"
        return 1
    fi
    
    # 检查AI系统健康状态
    if [ -n "$TOKEN" ]; then
        response=$(curl -s -w "%{http_code}" -o /tmp/ai_health.json \
            -H "Authorization: Bearer $TOKEN" \
            "$BASE_URL/ai/health")
        if [ "$response" = "200" ]; then
            log_success "AI系统健康检查通过"
            cat /tmp/ai_health.json | jq '.' 2>/dev/null || cat /tmp/ai_health.json
        else
            log_warning "AI系统健康检查失败 (HTTP $response)"
        fi
    fi
}

# 用户登录获取Token
login() {
    log_info "正在登录用户: $TEST_USER"
    
    response=$(curl -s -w "%{http_code}" -o /tmp/login.json \
        -X POST \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$TEST_USER\",\"password\":\"$TEST_PASSWORD\"}" \
        "$BASE_URL/auth/login")
    
    if [ "$response" = "200" ]; then
        TOKEN=$(cat /tmp/login.json | jq -r '.token // .data.token // empty' 2>/dev/null)
        if [ -n "$TOKEN" ] && [ "$TOKEN" != "null" ]; then
            log_success "登录成功，获取到Token: ${TOKEN:0:20}..."
            return 0
        else
            log_error "登录响应中未找到Token"
            cat /tmp/login.json
            return 1
        fi
    else
        log_error "登录失败 (HTTP $response)"
        cat /tmp/login.json
        return 1
    fi
}

# 测试AI配置
test_ai_configs() {
    log_info "测试AI配置管理..."
    
    response=$(curl -s -w "%{http_code}" -o /tmp/configs.json \
        -H "Authorization: Bearer $TOKEN" \
        "$BASE_URL/ai/configs")
    
    if [ "$response" = "200" ]; then
        config_count=$(cat /tmp/configs.json | jq '.data | length' 2>/dev/null || echo "0")
        log_success "成功获取AI配置列表，共 $config_count 个配置"
        
        if [ "$config_count" = "0" ]; then
            log_warning "未找到AI配置，请确保已初始化默认配置"
        fi
    else
        log_error "获取AI配置失败 (HTTP $response)"
        return 1
    fi
}

# 测试对话管理
test_conversations() {
    log_info "测试对话管理..."
    
    # 获取对话列表
    response=$(curl -s -w "%{http_code}" -o /tmp/conversations.json \
        -H "Authorization: Bearer $TOKEN" \
        "$BASE_URL/ai/conversations")
    
    if [ "$response" = "200" ]; then
        conv_count=$(cat /tmp/conversations.json | jq '.data | length' 2>/dev/null || echo "0")
        log_success "成功获取对话列表，共 $conv_count 个对话"
    else
        log_error "获取对话列表失败 (HTTP $response)"
        return 1
    fi
}

# 测试异步消息发送
test_async_messaging() {
    log_info "测试异步消息处理..."
    
    # 发送快速聊天请求
    response=$(curl -s -w "%{http_code}" -o /tmp/quick_chat.json \
        -X POST \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d "{\"message\":\"你好，这是一个测试消息\",\"context\":\"/test\"}" \
        "$BASE_URL/ai/quick-chat")
    
    if [ "$response" = "202" ]; then
        message_id=$(cat /tmp/quick_chat.json | jq -r '.message_id // empty' 2>/dev/null)
        if [ -n "$message_id" ]; then
            log_success "异步消息发送成功，消息ID: $message_id"
            
            # 测试状态查询
            test_message_status "$message_id"
        else
            log_warning "消息发送成功但未获取到消息ID"
        fi
    else
        log_error "异步消息发送失败 (HTTP $response)"
        cat /tmp/quick_chat.json
        return 1
    fi
}

# 测试消息状态查询
test_message_status() {
    local message_id="$1"
    log_info "测试消息状态查询: $message_id"
    
    local max_attempts=10
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        response=$(curl -s -w "%{http_code}" -o /tmp/status.json \
            -H "Authorization: Bearer $TOKEN" \
            "$BASE_URL/ai/messages/$message_id/status")
        
        if [ "$response" = "200" ]; then
            status=$(cat /tmp/status.json | jq -r '.data.status // empty' 2>/dev/null)
            log_info "消息状态 (尝试 $attempt/$max_attempts): $status"
            
            case "$status" in
                "completed")
                    result=$(cat /tmp/status.json | jq -r '.data.result // empty' 2>/dev/null)
                    log_success "消息处理完成！结果: ${result:0:50}..."
                    return 0
                    ;;
                "failed")
                    error=$(cat /tmp/status.json | jq -r '.data.error // empty' 2>/dev/null)
                    log_error "消息处理失败: $error"
                    return 1
                    ;;
                "processing"|"pending")
                    log_info "消息正在处理中，等待2秒后重试..."
                    sleep 2
                    ;;
                *)
                    log_warning "未知状态: $status"
                    ;;
            esac
        else
            log_warning "状态查询失败 (HTTP $response), 尝试 $attempt/$max_attempts"
        fi
        
        attempt=$((attempt + 1))
        sleep 2
    done
    
    log_warning "消息状态查询超时"
    return 1
}

# 测试集群操作提交
test_cluster_operations() {
    log_info "测试集群操作功能..."
    
    response=$(curl -s -w "%{http_code}" -o /tmp/cluster_op.json \
        -X POST \
        -H "Authorization: Bearer $TOKEN" \
        -H "Content-Type: application/json" \
        -d "{
            \"operation\":\"在Kubernetes上部署Nginx应用\",
            \"parameters\":{
                \"namespace\":\"default\",
                \"replicas\":3,
                \"image\":\"nginx:latest\"
            },
            \"description\":\"测试部署操作\"
        }" \
        "$BASE_URL/ai/cluster-operations")
    
    if [ "$response" = "202" ]; then
        operation_id=$(cat /tmp/cluster_op.json | jq -r '.operation_id // empty' 2>/dev/null)
        if [ -n "$operation_id" ]; then
            log_success "集群操作提交成功，操作ID: $operation_id"
            
            # 测试操作状态查询
            test_operation_status "$operation_id"
        else
            log_warning "集群操作提交成功但未获取到操作ID"
        fi
    else
        log_error "集群操作提交失败 (HTTP $response)"
        cat /tmp/cluster_op.json
        return 1
    fi
}

# 测试操作状态查询
test_operation_status() {
    local operation_id="$1"
    log_info "测试操作状态查询: $operation_id"
    
    local max_attempts=5
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        response=$(curl -s -w "%{http_code}" -o /tmp/op_status.json \
            -H "Authorization: Bearer $TOKEN" \
            "$BASE_URL/ai/operations/$operation_id/status")
        
        if [ "$response" = "200" ]; then
            status=$(cat /tmp/op_status.json | jq -r '.data.status // empty' 2>/dev/null)
            log_info "操作状态 (尝试 $attempt/$max_attempts): $status"
            
            case "$status" in
                "completed")
                    result=$(cat /tmp/op_status.json | jq -r '.data.result // empty' 2>/dev/null)
                    log_success "操作完成！结果: ${result:0:100}..."
                    return 0
                    ;;
                "failed")
                    error=$(cat /tmp/op_status.json | jq -r '.data.error // empty' 2>/dev/null)
                    log_error "操作失败: $error"
                    return 1
                    ;;
                "processing"|"pending")
                    log_info "操作正在处理中，等待3秒后重试..."
                    sleep 3
                    ;;
                *)
                    log_warning "未知操作状态: $status"
                    ;;
            esac
        else
            log_warning "操作状态查询失败 (HTTP $response), 尝试 $attempt/$max_attempts"
        fi
        
        attempt=$((attempt + 1))
        sleep 3
    done
    
    log_warning "操作状态查询超时"
    return 1
}

# 清理临时文件
cleanup() {
    rm -f /tmp/health.json /tmp/ai_health.json /tmp/login.json /tmp/configs.json 
    rm -f /tmp/conversations.json /tmp/quick_chat.json /tmp/status.json
    rm -f /tmp/cluster_op.json /tmp/op_status.json
}

# 主测试流程
main() {
    log_info "开始AI助手异步架构测试..."
    echo
    
    # 清理
    cleanup
    
    # 检查服务状态
    if ! check_service; then
        log_error "服务健康检查失败，请确保服务正常运行"
        exit 1
    fi
    echo
    
    # 登录
    if ! login; then
        log_error "用户登录失败，请检查用户名和密码"
        exit 1
    fi
    echo
    
    # 再次检查AI系统健康状态（需要认证）
    check_service
    echo
    
    # 测试AI配置
    if ! test_ai_configs; then
        log_warning "AI配置测试失败，部分功能可能不可用"
    fi
    echo
    
    # 测试对话管理
    if ! test_conversations; then
        log_warning "对话管理测试失败"
    fi
    echo
    
    # 测试异步消息
    if ! test_async_messaging; then
        log_warning "异步消息测试失败"
    fi
    echo
    
    # 测试集群操作
    if ! test_cluster_operations; then
        log_warning "集群操作测试失败"
    fi
    echo
    
    log_success "AI助手异步架构测试完成！"
    log_info "请查看上述输出结果，确保核心功能正常工作"
    
    # 清理
    cleanup
}

# 显示使用帮助
show_help() {
    echo "AI助手异步架构测试脚本"
    echo ""
    echo "用法: $0 [选项]"
    echo ""
    echo "选项:"
    echo "  -h, --help     显示帮助信息"
    echo "  -u, --user     指定测试用户名 (默认: admin)"
    echo "  -p, --password 指定测试密码 (默认: admin123)"
    echo "  -b, --base-url 指定API基础URL (默认: http://localhost:8082/api)"
    echo ""
    echo "示例:"
    echo "  $0                           # 使用默认设置运行测试"
    echo "  $0 -u testuser -p testpass   # 使用指定用户运行测试"
    echo ""
}

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -u|--user)
            TEST_USER="$2"
            shift 2
            ;;
        -p|--password)
            TEST_PASSWORD="$2"
            shift 2
            ;;
        -b|--base-url)
            BASE_URL="$2"
            shift 2
            ;;
        *)
            log_error "未知参数: $1"
            show_help
            exit 1
            ;;
    esac
done

# 运行主程序
main
