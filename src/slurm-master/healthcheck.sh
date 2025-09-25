#!/bin/bash

# AI Infrastructure Matrix SLURM Master健康检查
# 此脚本用于Docker Compose健康检查

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 输出函数
log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 健康检查函数
check_munge() {
    log_info "检查Munge认证服务..."
    
    # 检查Munge进程
    if ! pgrep -f munged >/dev/null; then
        log_error "Munge进程未运行"
        return 1
    fi
    
    # 测试Munge功能
    if ! timeout 5 munge -n | unmunge >/dev/null 2>&1; then
        log_error "Munge认证测试失败"
        return 1
    fi
    
    log_info "✅ Munge服务正常"
    return 0
}

check_slurmdbd() {
    log_info "检查slurmdbd服务..."
    
    # 检查slurmdbd进程
    if ! pgrep -f slurmdbd >/dev/null; then
        log_error "slurmdbd进程未运行"
        return 1
    fi
    
    # 检查slurmdbd端口
    local port=${SLURM_SLURMDBD_PORT:-6818}
    if ! timeout 3 nc -z localhost $port >/dev/null 2>&1; then
        log_error "slurmdbd端口 $port 不可达"
        return 1
    fi
    
    log_info "✅ slurmdbd服务正常"
    return 0
}

check_slurmctld() {
    log_info "检查slurmctld服务..."
    
    # 检查slurmctld进程
    if ! pgrep -f slurmctld >/dev/null; then
        log_error "slurmctld进程未运行"
        return 1
    fi
    
    # 检查slurmctld端口
    local port=${SLURM_CONTROLLER_PORT:-6817}
    if ! timeout 3 nc -z localhost $port >/dev/null 2>&1; then
        log_error "slurmctld端口 $port 不可达"
        return 1
    fi
    
    log_info "✅ slurmctld服务正常"
    return 0
}

check_slurm_cluster() {
    log_info "检查SLURM集群状态..."
    
    # 检查集群信息
    if ! timeout 10 scontrol ping >/dev/null 2>&1; then
        log_error "SLURM集群控制器无响应"
        return 1
    fi
    
    # 检查分区状态
    if ! timeout 10 sinfo -h >/dev/null 2>&1; then
        log_warn "无法获取分区信息（可能节点未连接）"
    else
        log_info "✅ 集群分区信息可访问"
    fi
    
    # 检查作业队列
    if ! timeout 10 squeue -h >/dev/null 2>&1; then
        log_warn "无法获取作业队列信息"
    else
        log_info "✅ 作业队列可访问"
    fi
    
    log_info "✅ SLURM集群基础功能正常"
    return 0
}

check_database_connection() {
    log_info "检查数据库连接..."
    
    local db_host=${SLURM_DB_HOST:-postgres}
    local db_port=${SLURM_DB_PORT:-5432}
    local db_name=${SLURM_DB_NAME:-slurm_acct_db}
    local db_user=${SLURM_DB_USER:-slurm}
    
    # 检查数据库端口连通性
    if ! timeout 5 nc -z "$db_host" "$db_port" >/dev/null 2>&1; then
        log_error "数据库端口 $db_host:$db_port 不可达"
        return 1
    fi
    
    log_info "✅ 数据库连接正常"
    return 0
}

check_config_files() {
    log_info "检查配置文件..."
    
    local configs=(
        "/etc/slurm/slurm.conf"
        "/etc/slurm/slurmdbd.conf"
        "/etc/slurm/cgroup.conf"
    )
    
    for config in "${configs[@]}"; do
        if [ ! -f "$config" ]; then
            log_error "配置文件不存在: $config"
            return 1
        fi
        
        if [ ! -r "$config" ]; then
            log_error "配置文件不可读: $config"
            return 1
        fi
    done
    
    log_info "✅ 配置文件检查通过"
    return 0
}

# 主健康检查函数
main_health_check() {
    log_info "开始SLURM Master健康检查..."
    
    local failed_checks=0
    
    # 基础检查
    check_config_files || ((failed_checks++))
    check_database_connection || ((failed_checks++))
    check_munge || ((failed_checks++))
    
    # 服务检查
    check_slurmdbd || ((failed_checks++))
    check_slurmctld || ((failed_checks++))
    
    # 功能检查
    check_slurm_cluster || ((failed_checks++))
    
    # 总结
    if [ $failed_checks -eq 0 ]; then
        log_info "🎉 所有健康检查通过！SLURM Master服务正常运行"
        return 0
    else
        log_error "❌ $failed_checks 项检查失败，服务可能异常"
        return 1
    fi
}

# 快速检查函数（用于Docker健康检查）
quick_health_check() {
    # 只检查关键服务是否运行
    pgrep -f munged >/dev/null 2>&1 || exit 1
    pgrep -f slurmdbd >/dev/null 2>&1 || exit 1
    pgrep -f slurmctld >/dev/null 2>&1 || exit 1
    
    # 检查端口连通性
    nc -z localhost ${SLURM_CONTROLLER_PORT:-6817} >/dev/null 2>&1 || exit 1
    nc -z localhost ${SLURM_SLURMDBD_PORT:-6818} >/dev/null 2>&1 || exit 1
    
    exit 0
}

# 详细状态报告
status_report() {
    echo "====== SLURM Master状态报告 ======"
    echo "时间: $(date)"
    echo ""
    
    echo "=== 环境配置 ==="
    echo "集群名称: ${SLURM_CLUSTER_NAME:-ai-infra-cluster}"
    echo "控制器地址: ${SLURM_CONTROLLER_HOST:-slurm-master}:${SLURM_CONTROLLER_PORT:-6817}"
    echo "数据库地址: ${SLURM_DB_HOST:-postgres}:${SLURM_DB_PORT:-5432}/${SLURM_DB_NAME:-slurm_acct_db}"
    echo ""
    
    echo "=== 进程状态 ==="
    echo "Munge进程: $(pgrep -f munged >/dev/null && echo '✅ 运行中' || echo '❌ 未运行')"
    echo "slurmdbd进程: $(pgrep -f slurmdbd >/dev/null && echo '✅ 运行中' || echo '❌ 未运行')"
    echo "slurmctld进程: $(pgrep -f slurmctld >/dev/null && echo '✅ 运行中' || echo '❌ 未运行')"
    echo ""
    
    echo "=== 端口状态 ==="
    echo "控制器端口 ${SLURM_CONTROLLER_PORT:-6817}: $(nc -z localhost ${SLURM_CONTROLLER_PORT:-6817} >/dev/null 2>&1 && echo '✅ 开放' || echo '❌ 关闭')"
    echo "数据库端口 ${SLURM_SLURMDBD_PORT:-6818}: $(nc -z localhost ${SLURM_SLURMDBD_PORT:-6818} >/dev/null 2>&1 && echo '✅ 开放' || echo '❌ 关闭')"
    echo ""
    
    echo "=== SLURM集群信息 ==="
    if timeout 5 scontrol ping >/dev/null 2>&1; then
        echo "集群状态: ✅ 在线"
        echo ""
        echo "分区信息:"
        timeout 5 sinfo 2>/dev/null || echo "  无法获取分区信息"
        echo ""
        echo "节点信息:"
        timeout 5 sinfo -N 2>/dev/null || echo "  无法获取节点信息"
    else
        echo "集群状态: ❌ 离线或无响应"
    fi
    
    echo "================================="
}

# 命令行参数处理
case "${1:-health-check}" in
    health-check|check)
        main_health_check
        ;;
    quick)
        quick_health_check
        ;;
    status)
        status_report
        ;;
    *)
        echo "用法: $0 {health-check|quick|status}"
        echo ""
        echo "  health-check  - 完整健康检查"
        echo "  quick        - 快速检查（Docker健康检查用）"
        echo "  status       - 详细状态报告"
        exit 1
        ;;
esac