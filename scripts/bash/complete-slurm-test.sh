#!/bin/bash

# 完整的SLURM作业提交测试
# 测试修复后的SSH节点注册和作业管理功能

set -euo pipefail

# 配置
BACKEND_URL="http://192.168.0.200:8082"
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

# 创建测试集群记录
create_test_cluster() {
    print_header "创建测试集群记录"
    
    # 直接在数据库中创建集群记录
    docker exec ai-infra-postgres psql -U postgres -d ansible_playbook_generator -c "
        INSERT INTO clusters (id, name, description, host, port, username, password, status, created_at, updated_at)
        VALUES ('test-cluster-001', '测试SLURM集群', '用于测试的集群', 'ai-infra-slurm-master', 22, 'root', '', 'active', NOW(), NOW())
        ON CONFLICT (id) DO UPDATE SET
            name = EXCLUDED.name,
            description = EXCLUDED.description,
            host = EXCLUDED.host,
            port = EXCLUDED.port,
            username = EXCLUDED.username,
            password = EXCLUDED.password,
            status = EXCLUDED.status,
            updated_at = NOW();
    "
    
    if [[ $? -eq 0 ]]; then
        print_success "✓ 测试集群记录创建成功"
    else
        print_error "✗ 创建测试集群记录失败"
        return 1
    fi
}

# 测试作业提交
test_job_submission() {
    local token="$1"
    print_header "测试作业提交"
    
    local job_data='{
        "name": "test-job-fix-verification",
        "command": "echo \"SLURM作业测试开始\" && echo \"当前时间: $(date)\" && echo \"主机名: $(hostname)\" && sleep 10 && echo \"测试完成\"",
        "cluster_id": "test-cluster-001",
        "partition": "compute",
        "nodes": 1,
        "cpus": 1,
        "memory": "100M",
        "time_limit": "00:05:00",
        "std_out": "/tmp/test-job-%j.out",
        "std_err": "/tmp/test-job-%j.err"
    }'
    
    print_info "提交作业数据: $job_data"
    
    local response=$(curl -s -X POST "$BACKEND_URL/api/jobs/submit" \
        -H "Authorization: Bearer $token" \
        -H "Content-Type: application/json" \
        -d "$job_data")
    
    print_info "作业提交响应: $response"
    
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
    
    print_info "等待5秒后查询作业状态..."
    sleep 5
    
    local response=$(curl -s -H "Authorization: Bearer $token" "$BACKEND_URL/api/jobs/$job_id/status")
    
    print_info "状态查询响应: $response"
    
    if [[ $? -eq 0 ]] && echo "$response" | jq . >/dev/null 2>&1; then
        local status=$(echo "$response" | jq -r '.state // .status // "UNKNOWN"')
        print_success "✓ 作业状态查询成功: $status"
        
        # 显示详细状态信息
        echo "$response" | jq .
        return 0
    else
        print_error "✗ 作业状态查询失败: $response"
        return 1
    fi
}

# 测试作业列表查询
test_jobs_list() {
    local token="$1"
    print_header "测试作业列表查询"
    
    local response=$(curl -s -H "Authorization: Bearer $token" "$BACKEND_URL/api/jobs?cluster_id=test-cluster-001")
    
    print_info "作业列表响应: $response"
    
    if [[ $? -eq 0 ]] && echo "$response" | jq . >/dev/null 2>&1; then
        local job_count=$(echo "$response" | jq '.jobs | length' 2>/dev/null || echo "0")
        print_success "✓ 作业列表查询成功，作业数量: $job_count"
        
        # 显示作业列表
        echo "$response" | jq .
        return 0
    else
        print_error "✗ 作业列表查询失败: $response"
        return 1
    fi
}

# 直接测试SLURM命令
test_slurm_direct() {
    print_header "测试SLURM直接命令"
    
    print_info "检查SLURM信息..."
    docker exec ai-infra-slurm-master sinfo
    
    print_info "检查作业队列..."
    docker exec ai-infra-slurm-master squeue
    
    print_info "检查SLURM配置..."
    docker exec ai-infra-slurm-master cat /etc/slurm/slurm.conf | head -20
}

# 检查数据库中的作业记录
check_job_database() {
    print_header "检查数据库中的作业记录"
    
    print_info "查询最近的作业记录..."
    docker exec ai-infra-postgres psql -U postgres -d ansible_playbook_generator -c "
        SELECT id, name, status, cluster_id, job_id, created_at, updated_at 
        FROM jobs 
        ORDER BY created_at DESC 
        LIMIT 5;
    "
    
    print_info "查询集群记录..."
    docker exec ai-infra-postgres psql -U postgres -d ansible_playbook_generator -c "
        SELECT id, name, host, port, username, status, created_at 
        FROM clusters 
        ORDER BY created_at DESC 
        LIMIT 3;
    "
}

# 清理测试数据
cleanup_test_data() {
    print_header "清理测试数据"
    
    print_info "清理测试作业..."
    docker exec ai-infra-postgres psql -U postgres -d ansible_playbook_generator -c "
        DELETE FROM jobs WHERE cluster_id = 'test-cluster-001';
    "
    
    print_info "清理测试集群..."
    docker exec ai-infra-postgres psql -U postgres -d ansible_playbook_generator -c "
        DELETE FROM clusters WHERE id = 'test-cluster-001';
    "
    
    print_success "✓ 测试数据清理完成"
}

# 主测试流程
main() {
    print_header "SLURM作业管理完整测试"
    
    # 获取认证令牌
    local token
    if ! token=$(get_auth_token); then
        print_error "无法获取认证令牌，测试终止"
        return 1
    fi
    
    # 创建测试集群
    if ! create_test_cluster; then
        print_error "创建测试集群失败"
        return 1
    fi
    
    # 测试SLURM直接命令
    test_slurm_direct
    
    # 测试作业提交
    local job_id
    if job_id=$(test_job_submission "$token"); then
        print_success "作业提交测试完成: $job_id"
        
        # 测试状态查询
        test_job_status "$token" "$job_id"
        
        # 测试作业列表
        test_jobs_list "$token"
        
        # 检查数据库记录
        check_job_database
        
    else
        print_error "作业提交测试失败"
    fi
    
    # 询问是否清理测试数据
    echo ""
    read -p "是否清理测试数据？(y/N): " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        cleanup_test_data
    fi
    
    print_success "测试完成！"
}

# 执行主函数
main "$@"