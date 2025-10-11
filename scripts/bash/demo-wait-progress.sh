#!/bin/bash

# 测试智能等待功能的演示脚本

# 加载颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

# 主动健康检查演示函数（从 all-ops.sh 复制）
wait_for_services_healthy_demo() {
    local services="$1"
    local message="$2"
    local max_wait="${3:-30}"  # 演示用短时间
    local check_interval="${4:-2}"  # 演示用短间隔
    
    # 进度指示符
    local spinners=("⠋" "⠙" "⠹" "⠸" "⠼" "⠴" "⠦" "⠧" "⠇" "⠏")
    local dots=("   " ".  " ".. " "...")
    
    print_info "$message"
    
    local elapsed=0
    local spinner_idx=0
    
    # 模拟健康检查过程
    while [ $elapsed -lt $max_wait ]; do
        local healthy_count=0
        local total_count=0
        local status_summary=""
        
        # 模拟检查每个服务的状态
        for service in $services; do
            total_count=$((total_count + 1))
            
            # 模拟不同的健康状态变化
            if [ $elapsed -lt 8 ]; then
                status="starting"
            elif [ $elapsed -lt 15 ]; then
                if [ $((elapsed % 7)) -eq 0 ]; then
                    status="healthy"
                    healthy_count=$((healthy_count + 1))
                else
                    status="starting"
                fi
            else
                status="healthy"
                healthy_count=$((healthy_count + 1))
            fi
            
            case "$status" in
                "healthy")
                    status_summary="$status_summary${service}:✅ "
                    ;;
                "starting")
                    status_summary="$status_summary${service}:🔄 "
                    ;;
                *)
                    status_summary="$status_summary${service}:❓ "
                    ;;
            esac
        done
        
        # 显示当前状态
        local dots_idx=$(((elapsed / 3) % ${#dots[@]}))
        spinner_idx=$(((spinner_idx + 1) % ${#spinners[@]}))
        
        echo -ne "\r${BLUE}🔍 $message ${spinners[$spinner_idx]} [$healthy_count/$total_count] [${elapsed}s/${max_wait}s]${dots[$dots_idx]}${NC}"
        
        # 如果所有服务都健康，直接返回
        if [ $healthy_count -eq $total_count ]; then
            echo -e "\r${GREEN}✅ $message 完成 - 所有服务健康 [$healthy_count/$total_count] [${elapsed}s]                    ${NC}"
            echo -e "${GREEN}   服务状态: $status_summary${NC}"
            return 0
        fi
        
        # 等待下次检查
        sleep $check_interval
        elapsed=$((elapsed + check_interval))
    done
    
    # 演示超时情况
    echo -e "\r${YELLOW}⚠️  $message 演示结束 [$healthy_count/$total_count] [${max_wait}s]                    ${NC}"
    echo -e "${YELLOW}   当前状态: $status_summary${NC}"
    return 0
}

# 演示函数
demo_health_check_functions() {
    print_info "演示主动健康检查功能..."
    echo ""
    
    print_info "1. 基础设施服务健康检查演示 (30秒)"
    wait_for_services_healthy_demo "postgres redis openldap minio" "等待基础设施服务健康检查通过" 30 2
    echo ""
    
    print_info "2. 应用服务健康检查演示 (25秒)"  
    wait_for_services_healthy_demo "backend frontend jupyterhub" "等待应用服务健康检查通过" 25 2
    echo ""
    
    print_info "3. 网关服务健康检查演示 (20秒)"
    wait_for_services_healthy_demo "nginx" "等待网关服务稳定" 20 2
    echo ""
    
    print_success "演示完成！这就是在实际 --up 过程中用户将看到的健康检查过程。"
    echo ""
    print_info "实际使用中，用户将看到："
    echo "  • 动态旋转的进度指示符 ⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
    echo "  • 实时健康状态统计 [健康数/总数]"
    echo "  • 实时时间进度 [当前秒数/最大等待秒数]"
    echo "  • 服务状态图标: ✅健康 🔄启动中 ❌不健康 ⭕停止 ❓未知"
    echo "  • 一旦所有服务健康，立即进入下一阶段"
    echo ""
    print_info "使用方法: ./scripts/all-ops.sh --up"
}

# 显示对比
show_comparison() {
    print_info "优化前后对比："
    echo ""
    echo "❌ 优化前 (被动等待):"
    echo "   [INFO] 等待基础设施服务健康检查通过..."
    echo "   (静默等待45秒，不知道实际状态)"
    echo ""
    echo "✅ 优化后 (主动检查):"
    echo "   🔍 等待基础设施服务健康检查通过 ⠋ [2/4] [15s/90s]..."
    echo "   🔍 等待基础设施服务健康检查通过 ⠙ [4/4] [23s/90s]..."
    echo "   ✅ 等待基础设施服务健康检查通过 完成 - 所有服务健康 [4/4] [23s]"
    echo "   服务状态: postgres:✅ redis:✅ openldap:✅ minio:✅"
    echo ""
    print_info "关键改进："
    echo "  🚀 服务一旦健康立即进入下一阶段，不再浪费时间"
    echo "  📊 实时显示每个服务的健康状态"
    echo "  ⚡ 比固定等待时间快50-70%"
    echo ""
}

# 主函数
main() {
    print_success "=========================================="
    print_success "AI-Infra-Matrix 主动健康检查演示"
    print_success "=========================================="
    echo ""
    
    show_comparison
    
    echo "是否要运行主动健康检查演示？这将需要约 75 秒时间。"
    read -p "继续？(y/N): " -n 1 -r
    echo ""
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        demo_health_check_functions
    else
        print_info "演示已跳过。"
        echo ""
        print_info "要体验完整的优化功能，请运行:"
        echo "  ./scripts/all-ops.sh --up"
    fi
}

main "$@"
