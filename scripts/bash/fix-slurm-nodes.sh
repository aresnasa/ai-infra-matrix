#!/bin/bash

# SLURM节点注册修复脚本
# 解决SSH节点注册到SLURM任务提交后无法查询到相关任务的问题

set -euo pipefail

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印函数
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

# 检查依赖
check_dependencies() {
    print_header "检查依赖"
    
    local deps=("docker" "docker-compose" "curl" "jq")
    for dep in "${deps[@]}"; do
        if command -v "$dep" >/dev/null 2>&1; then
            print_success "✓ $dep 已安装"
        else
            print_error "✗ $dep 未安装，请先安装"
            exit 1
        fi
    done
}

# 检查SLURM服务状态
check_slurm_status() {
    print_header "检查SLURM服务状态"
    
    # 检查SLURM控制器容器
    if docker ps --format "{{.Names}}" | grep -q "ai-infra-slurm-master"; then
        print_success "✓ SLURM控制器容器正在运行"
        
        # 检查SLURM服务
        if docker exec ai-infra-slurm-master sinfo >/dev/null 2>&1; then
            print_success "✓ SLURM服务正常"
        else
            print_warning "⚠ SLURM服务可能存在问题"
        fi
    else
        print_error "✗ SLURM控制器容器未运行"
        print_info "尝试启动SLURM服务..."
        docker-compose up -d slurm-master
        sleep 10
    fi
}

# 显示当前SLURM节点配置
show_current_nodes() {
    print_header "当前SLURM节点配置"
    
    echo "=== SLURM节点信息 ==="
    if docker exec ai-infra-slurm-master sinfo 2>/dev/null; then
        print_success "✓ 成功获取SLURM节点信息"
    else
        print_warning "⚠ 无法获取SLURM节点信息"
    fi
    
    echo -e "\n=== 数据库中的节点 ==="
    if docker exec ai-infra-postgres psql -U postgres -d ansible_playbook_generator -c "SELECT node_name, host, port, status, node_type FROM slurm_nodes WHERE status='active';" 2>/dev/null; then
        print_success "✓ 成功获取数据库节点信息"
    else
        print_warning "⚠ 无法获取数据库节点信息"
    fi
}

# 重新生成SLURM配置
regenerate_slurm_config() {
    print_header "重新生成SLURM配置"
    
    print_info "正在重新生成slurm.conf..."
    
    # 从数据库获取活跃节点
    local nodes_query="SELECT node_name FROM slurm_nodes WHERE status='active' AND node_type IN ('compute', 'node');"
    local nodes=$(docker exec ai-infra-postgres psql -U postgres -d ansible_playbook_generator -t -c "$nodes_query" 2>/dev/null | xargs)
    
    if [[ -z "$nodes" ]]; then
        print_warning "⚠ 数据库中没有找到活跃的计算节点"
        return 1
    fi
    
    print_info "找到节点: $nodes"
    
    # 生成新的slurm.conf
    local slurm_conf="/tmp/slurm.conf.new"
    cat > "$slurm_conf" << EOF
# SLURM配置文件 - AI Infrastructure Matrix
ClusterName=ai-infra-cluster
ControlMachine=slurm-controller
ControlAddr=slurm-controller

# 认证和安全
AuthType=auth/munge
CryptoType=crypto/munge

# 调度器配置
SchedulerType=sched/backfill
SelectType=select/cons_res
SelectTypeParameters=CR_Core

# 日志配置
SlurmdLogFile=/var/log/slurm/slurmd.log
SlurmctldLogFile=/var/log/slurm/slurmctld.log
SlurmdSpoolDir=/var/spool/slurm

# 节点配置
EOF

    # 添加节点定义
    local node_names=()
    for node in $nodes; do
        echo "NodeName=$node CPUs=2 Sockets=1 CoresPerSocket=2 ThreadsPerCore=1 RealMemory=1000 State=UNKNOWN" >> "$slurm_conf"
        node_names+=("$node")
    done
    
    # 添加分区配置
    local nodes_list=$(IFS=','; echo "${node_names[*]}")
    echo "PartitionName=compute Nodes=$nodes_list Default=YES MaxTime=INFINITE State=UP" >> "$slurm_conf"
    
    print_success "✓ 新的slurm.conf已生成"
    
    # 上传到SLURM控制器
    print_info "上传配置到SLURM控制器..."
    docker cp "$slurm_conf" ai-infra-slurm-master:/etc/slurm/slurm.conf
    
    # 重新加载配置
    print_info "重新加载SLURM配置..."
    if docker exec ai-infra-slurm-master scontrol reconfigure 2>/dev/null; then
        print_success "✓ SLURM配置已重新加载"
    else
        print_warning "⚠ SLURM配置重新加载失败，尝试重启SLURM服务"
        docker exec ai-infra-slurm-master supervisorctl restart slurmctld
        sleep 5
    fi
    
    # 清理临时文件
    rm -f "$slurm_conf"
}

# 测试作业提交和查询
test_job_submission() {
    print_header "测试作业提交和查询"
    
    # 创建测试脚本
    local test_script="/tmp/test_job.sh"
    cat > "$test_script" << 'EOF'
#!/bin/bash
#SBATCH --job-name=test-job
#SBATCH --output=/tmp/test-job-%j.out
#SBATCH --error=/tmp/test-job-%j.err
#SBATCH --partition=compute
#SBATCH --nodes=1
#SBATCH --time=00:01:00

echo "测试作业开始执行"
echo "当前时间: $(date)"
echo "主机名: $(hostname)"
echo "用户: $(whoami)"
sleep 30
echo "测试作业执行完成"
EOF

    # 上传测试脚本
    print_info "上传测试脚本到SLURM控制器..."
    docker cp "$test_script" ai-infra-slurm-master:/tmp/test_job.sh
    docker exec ai-infra-slurm-master chmod +x /tmp/test_job.sh
    
    # 提交作业
    print_info "提交测试作业..."
    local job_output=$(docker exec ai-infra-slurm-master sbatch /tmp/test_job.sh 2>&1)
    
    if [[ $job_output == *"Submitted batch job"* ]]; then
        local job_id=$(echo "$job_output" | grep -o '[0-9]\+')
        print_success "✓ 作业提交成功，作业ID: $job_id"
        
        # 查询作业状态
        print_info "查询作业状态..."
        sleep 2
        
        local job_status=$(docker exec ai-infra-slurm-master squeue -h -j "$job_id" -o '%T' 2>/dev/null || echo "NOT_FOUND")
        
        if [[ "$job_status" != "NOT_FOUND" && -n "$job_status" ]]; then
            print_success "✓ 作业状态查询成功: $job_status"
        else
            # 尝试使用sacct查询历史作业
            local job_status_hist=$(docker exec ai-infra-slurm-master sacct -j "$job_id" --format=State -n 2>/dev/null || echo "NOT_FOUND")
            if [[ "$job_status_hist" != "NOT_FOUND" && -n "$job_status_hist" ]]; then
                print_success "✓ 历史作业状态查询成功: $job_status_hist"
            else
                print_error "✗ 无法查询作业状态"
                return 1
            fi
        fi
        
        # 取消测试作业（如果还在运行）
        print_info "清理测试作业..."
        docker exec ai-infra-slurm-master scancel "$job_id" 2>/dev/null || true
        
    else
        print_error "✗ 作业提交失败: $job_output"
        return 1
    fi
    
    # 清理
    rm -f "$test_script"
    docker exec ai-infra-slurm-master rm -f /tmp/test_job.sh
}

# 检查API响应
test_api_endpoints() {
    print_header "测试API端点"
    
    local backend_url="http://localhost:8080"
    
    # 测试SLURM状态端点
    print_info "测试SLURM状态API..."
    if curl -s "$backend_url/api/slurm/status" >/dev/null; then
        print_success "✓ SLURM状态API响应正常"
    else
        print_warning "⚠ SLURM状态API无响应"
    fi
    
    # 测试节点列表端点
    print_info "测试节点列表API..."
    if curl -s "$backend_url/api/slurm/nodes" >/dev/null; then
        print_success "✓ 节点列表API响应正常"
    else
        print_warning "⚠ 节点列表API无响应"
    fi
}

# 修复权限问题
fix_permissions() {
    print_header "修复权限问题"
    
    print_info "修复SLURM配置文件权限..."
    docker exec ai-infra-slurm-master chown slurm:slurm /etc/slurm/slurm.conf
    docker exec ai-infra-slurm-master chmod 644 /etc/slurm/slurm.conf
    
    print_info "修复日志目录权限..."
    docker exec ai-infra-slurm-master mkdir -p /var/log/slurm /var/spool/slurm
    docker exec ai-infra-slurm-master chown -R slurm:slurm /var/log/slurm /var/spool/slurm
    
    print_success "✓ 权限修复完成"
}

# 显示修复建议
show_recommendations() {
    print_header "修复建议"
    
    echo "为了确保SLURM节点注册和作业管理正常工作，建议："
    echo ""
    echo "1. 📝 确保新注册的SSH节点配置正确的认证信息"
    echo "   - 在添加节点时提供正确的用户名和密码"
    echo "   - 确保SSH连接可用"
    echo ""
    echo "2. 🔄 每次添加新节点后自动重新生成SLURM配置"
    echo "   - 后端服务会自动调用UpdateSlurmConfig"
    echo "   - 如果自动更新失败，手动运行此脚本"
    echo ""
    echo "3. 🔍 定期检查SLURM集群状态"
    echo "   - 运行 'docker exec ai-infra-slurm-master sinfo' 检查节点状态"
    echo "   - 运行 'docker exec ai-infra-slurm-master squeue' 检查作业队列"
    echo ""
    echo "4. 🛠 如果作业提交后无法查询，检查："
    echo "   - SLURM控制器服务是否正常运行"
    echo "   - 节点是否正确注册到SLURM集群"
    echo "   - SSH认证信息是否正确"
    echo ""
    echo "5. 📊 使用API测试端点验证功能"
    echo "   - GET /api/slurm/status - 检查SLURM状态"
    echo "   - GET /api/slurm/nodes - 查看节点列表"
    echo "   - POST /api/jobs/submit - 测试作业提交"
}

# 主函数
main() {
    print_header "SLURM节点注册修复工具"
    
    echo "此脚本用于修复SSH节点注册到SLURM任务提交后无法查询到相关任务的问题"
    echo ""
    
    local action="${1:-all}"
    
    case "$action" in
        "check")
            check_dependencies
            check_slurm_status
            show_current_nodes
            ;;
        "fix")
            check_dependencies
            check_slurm_status
            fix_permissions
            regenerate_slurm_config
            ;;
        "test")
            check_dependencies
            test_job_submission
            test_api_endpoints
            ;;
        "all"|*)
            check_dependencies
            check_slurm_status
            show_current_nodes
            fix_permissions
            regenerate_slurm_config
            test_job_submission
            test_api_endpoints
            show_recommendations
            ;;
    esac
    
    print_success "修复流程完成！"
}

# 显示帮助
if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
    echo "用法: $0 [action]"
    echo ""
    echo "Actions:"
    echo "  check  - 仅检查当前状态"
    echo "  fix    - 修复配置问题"
    echo "  test   - 测试作业提交和查询"
    echo "  all    - 执行所有操作（默认）"
    echo ""
    echo "示例:"
    echo "  $0        # 执行完整修复流程"
    echo "  $0 check  # 仅检查状态"
    echo "  $0 fix    # 仅修复配置"
    echo "  $0 test   # 仅测试功能"
    exit 0
fi

# 执行主函数
main "$@"