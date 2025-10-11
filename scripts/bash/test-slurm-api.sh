#!/bin/bash

# SLURM作业提交和查询测试脚本
# 用于验证SSH节点注册后的作业管理功能

set -euo pipefail

# 配置
BACKEND_URL="${BACKEND_URL:-http://192.168.0.200:8082}"
TEST_USER="admin"
TEST_PASSWORD="admin123"

# 颜色定义
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

print_header() {
    echo -e "\n${BLUE}========================================${NC}"
    echo -e "${BLUE} $1${NC}"
    echo -e "${BLUE}========================================${NC}\n"
}

# 获取认证令牌
get_auth_token() {
    print_info "获取认证令牌..."
    
    local login_response=$(curl -s -X POST "$BACKEND_URL/api/auth/login" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"$TEST_USER\",\"password\":\"$TEST_PASSWORD\"}")
    
    if [[ $? -eq 0 ]] && echo "$login_response" | jq -e '.token' >/dev/null 2>&1; then
        local token=$(echo "$login_response" | jq -r '.token')
        print_success "✓ 认证成功"
        echo "$token"
    else
        print_error "✗ 认证失败: $login_response"
        return 1
    fi
}

# 测试SLURM状态
test_slurm_status() {
    local token="$1"
    print_header "测试SLURM状态"
    
    local response=$(curl -s -H "Authorization: Bearer $token" "$BACKEND_URL/api/slurm/summary")
    
    if [[ $? -eq 0 ]] && echo "$response" | jq . >/dev/null 2>&1; then
        print_success "✓ SLURM状态API响应正常"
        echo "$response" | jq .
    else
        print_error "✗ SLURM状态API失败: $response"
        return 1
    fi
}

# 测试节点列表
test_nodes_list() {
    local token="$1"
    print_header "测试节点列表"
    
    local response=$(curl -s -H "Authorization: Bearer $token" "$BACKEND_URL/api/slurm/nodes")
    
    if [[ $? -eq 0 ]] && echo "$response" | jq . >/dev/null 2>&1; then
        print_success "✓ 节点列表API响应正常"
        local node_count=$(echo "$response" | jq '.nodes | length' 2>/dev/null || echo "0")
        print_info "节点数量: $node_count"
        echo "$response" | jq .
    else
        print_error "✗ 节点列表API失败: $response"
        return 1
    fi
}

# 创建测试集群
create_test_cluster() {
    local token="$1"
    print_header "创建测试集群"
    
    local cluster_data='{
        "id": "test-cluster-001",
        "name": "测试SLURM集群",
        "description": "用于测试作业提交和查询的集群",
        "host": "slurm-controller",
        "port": 22,
        "username": "root",
        "password": "",
        "status": "active"
    }'
    
    local response=$(curl -s -X POST "$BACKEND_URL/api/clusters" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        -d "$cluster_data")
    
    if [[ $? -eq 0 ]] && (echo "$response" | jq -e '.id' >/dev/null 2>&1 || echo "$response" | grep -q "already exists"); then
        print_success "✓ 测试集群创建/存在"
        echo "test-cluster-001"
    else
        print_error "✗ 创建测试集群失败: $response"
        return 1
    fi
}

# 测试作业提交
test_job_submission() {
    local token="$1"
    local cluster_id="$2"
    print_header "测试作业提交"
    
    local job_data="{
        \"name\": \"test-job-$(date +%s)\",
        \"command\": \"echo 'Hello SLURM' && sleep 10 && echo 'Job completed'\",
        \"cluster_id\": \"$cluster_id\",
        \"partition\": \"compute\",
        \"nodes\": 1,
        \"cpus\": 1,
        \"memory\": \"100M\",
        \"time_limit\": \"00:05:00\",
        \"std_out\": \"/tmp/test-job-%j.out\",
        \"std_err\": \"/tmp/test-job-%j.err\"
    }"
    
    local response=$(curl -s -X POST "$BACKEND_URL/api/jobs/submit" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        -d "$job_data")
    
    if [[ $? -eq 0 ]] && echo "$response" | jq -e '.id' >/dev/null 2>&1; then
        local job_id=$(echo "$response" | jq -r '.id')
        print_success "✓ 作业提交成功，作业ID: $job_id"
        echo "$job_id"
    else
        print_error "✗ 作业提交失败: $response"
        return 1
    fi
}

# 测试作业状态查询
test_job_status() {
    local token="$1"
    local job_id="$2"
    print_header "测试作业状态查询"
    
    print_info "等待2秒后查询作业状态..."
    sleep 2
    
    local response=$(curl -s -H "Authorization: Bearer $token" "$BACKEND_URL/api/jobs/$job_id/status")
    
    if [[ $? -eq 0 ]] && echo "$response" | jq . >/dev/null 2>&1; then
        local status=$(echo "$response" | jq -r '.state // .status // "UNKNOWN"')
        print_success "✓ 作业状态查询成功: $status"
        echo "$response" | jq .
        return 0
    else
        print_error "✗ 作业状态查询失败: $response"
        return 1
    fi
}

# 测试作业列表
test_jobs_list() {
    local token="$1"
    local cluster_id="$2"
    print_header "测试作业列表"
    
    local response=$(curl -s -H "Authorization: Bearer $token" "$BACKEND_URL/api/jobs?cluster_id=$cluster_id")
    
    if [[ $? -eq 0 ]] && echo "$response" | jq . >/dev/null 2>&1; then
        local job_count=$(echo "$response" | jq '.jobs | length' 2>/dev/null || echo "0")
        print_success "✓ 作业列表查询成功，作业数量: $job_count"
        echo "$response" | jq .
    else
        print_error "✗ 作业列表查询失败: $response"
        return 1
    fi
}

# 清理测试作业
cleanup_test_job() {
    local token="$1"
    local job_id="$2"
    print_header "清理测试作业"
    
    local response=$(curl -s -X DELETE "$BACKEND_URL/api/jobs/$job_id" \
        -H "Authorization: Bearer $token")
    
    if [[ $? -eq 0 ]]; then
        print_success "✓ 测试作业清理完成"
    else
        print_warning "⚠ 测试作业清理失败，请手动清理: $response"
    fi
}

# 测试SSH节点注册
test_ssh_node_registration() {
    local token="$1"
    print_header "测试SSH节点注册"
    
    # 这里可以添加测试SSH节点的注册逻辑
    # 由于需要实际的SSH节点，这里仅测试API端点的可用性
    
    local test_data='{
        "nodes": [
            {
                "ssh": {
                    "host": "test-node-1",
                    "port": 22,
                    "user": "root",
                    "password": "testpass"
                },
                "role": "compute"
            }
        ],
        "repoURL": "http://localhost:8090/pkgs/slurm-deb"
    }'
    
    print_info "测试节点注册API端点可用性..."
    local response=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BACKEND_URL/api/slurm/init-nodes" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        -d "$test_data")
    
    if [[ "$response" =~ ^[2-4][0-9][0-9]$ ]]; then
        print_success "✓ 节点注册API端点可用 (HTTP $response)"
    else
        print_warning "⚠ 节点注册API端点不可用 (HTTP $response)"
    fi
}

# 完整测试流程
run_full_test() {
    print_header "SLURM作业管理完整测试"
    
    # 获取认证令牌
    local token
    if ! token=$(get_auth_token); then
        print_error "无法获取认证令牌，测试终止"
        return 1
    fi
    
    # 测试基础API
    test_slurm_status "$token" || true
    test_nodes_list "$token" || true
    test_ssh_node_registration "$token" || true
    
    # 创建测试集群
    local cluster_id
    if cluster_id=$(create_test_cluster "$token"); then
        print_success "测试集群准备完成: $cluster_id"
        
        # 测试作业管理
        local job_id
        if job_id=$(test_job_submission "$token" "$cluster_id"); then
            print_success "作业提交测试完成: $job_id"
            
            # 测试状态查询
            test_job_status "$token" "$job_id" || true
            
            # 测试作业列表
            test_jobs_list "$token" "$cluster_id" || true
            
            # 清理测试作业
            cleanup_test_job "$token" "$job_id" || true
        else
            print_error "作业提交测试失败"
        fi
    else
        print_error "测试集群创建失败"
    fi
}

# 快速状态检查
quick_check() {
    print_header "快速状态检查"
    
    # 检查后端服务
    print_info "检查后端服务..."
    if curl -s "$BACKEND_URL/api/health" >/dev/null; then
        print_success "✓ 后端服务正常"
    else
        print_error "✗ 后端服务不可用"
        return 1
    fi
    
    # 检查SLURM容器
    print_info "检查SLURM容器..."
    if docker ps --format "{{.Names}}" | grep -q "ai-infra-slurm-master"; then
        print_success "✓ SLURM控制器容器运行中"
    else
        print_error "✗ SLURM控制器容器未运行"
        return 1
    fi
    
    print_success "快速检查完成"
}

# 主函数
main() {
    local action="${1:-full}"
    
    case "$action" in
        "quick")
            quick_check
            ;;
        "auth")
            get_auth_token >/dev/null && print_success "认证测试成功"
            ;;
        "status")
            token=$(get_auth_token) && test_slurm_status "$token"
            ;;
        "nodes")
            token=$(get_auth_token) && test_nodes_list "$token"
            ;;
        "submit")
            token=$(get_auth_token)
            cluster_id=$(create_test_cluster "$token")
            job_id=$(test_job_submission "$token" "$cluster_id")
            echo "提交的作业ID: $job_id"
            ;;
        "full"|*)
            run_full_test
            ;;
    esac
}

# 显示帮助
if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
    echo "用法: $0 [action]"
    echo ""
    echo "Actions:"
    echo "  quick  - 快速状态检查"
    echo "  auth   - 测试认证"
    echo "  status - 测试SLURM状态"
    echo "  nodes  - 测试节点列表"
    echo "  submit - 测试作业提交"
    echo "  full   - 完整测试流程（默认）"
    echo ""
    echo "环境变量:"
    echo "  BACKEND_URL - 后端服务地址（默认: http://localhost:8080）"
    echo ""
    echo "示例:"
    echo "  $0           # 运行完整测试"
    echo "  $0 quick     # 快速检查"
    echo "  $0 submit    # 仅测试作业提交"
    exit 0
fi

# 执行主函数
main "$@"