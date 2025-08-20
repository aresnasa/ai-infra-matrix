#!/bin/bash

# AI Infrastructure Matrix - 改进的启动脚本
# 解决PostgreSQL等基础服务启动顺序问题

set -euo pipefail

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# 检查Docker Compose是否可用
check_docker_compose() {
    if command -v "docker compose" &> /dev/null; then
        COMPOSE_CMD="docker compose"
    elif command -v "docker-compose" &> /dev/null; then
        COMPOSE_CMD="docker-compose"
    else
        print_error "Docker Compose 未安装或不可用"
        exit 1
    fi
    print_info "使用 Docker Compose: $COMPOSE_CMD"
}

# 等待服务健康检查通过
wait_for_service() {
    local service_name="$1"
    local max_attempts="${2:-30}"
    local attempt=0
    
    print_info "等待服务 $service_name 启动..."
    
    while [ $attempt -lt $max_attempts ]; do
        # 使用更简单的检查方式，避免jq解析问题
        if $COMPOSE_CMD ps | grep "$service_name" | grep -q "healthy\|Up"; then
            print_success "服务 $service_name 已就绪"
            return 0
        fi
        
        attempt=$((attempt + 1))
        echo -n "."
        sleep 2
    done
    
    print_error "服务 $service_name 启动超时"
    return 1
}

# 检查服务是否正在运行
check_service_running() {
    local service_name="$1"
    if $COMPOSE_CMD ps | grep "$service_name" | grep -q "Up\|running"; then
        return 0
    else
        return 1
    fi
}

# 分阶段启动服务
staged_startup() {
    print_info "开始分阶段启动服务..."
    
    # 第一阶段：基础设施服务
    print_info "第一阶段：启动基础设施服务（PostgreSQL, Redis, OpenLDAP）"
    $COMPOSE_CMD up -d postgres redis openldap
    
    # 等待基础服务就绪
    wait_for_service "postgres" 60
    wait_for_service "redis" 30
    wait_for_service "openldap" 60
    
    # 第二阶段：MinIO 和 phpLDAPadmin
    print_info "第二阶段：启动存储和管理服务"
    $COMPOSE_CMD up -d minio phpldapadmin
    wait_for_service "minio" 30
    
    # 第三阶段：应用初始化
    print_info "第三阶段：运行后端初始化"
    $COMPOSE_CMD up -d backend-init
    
    # 等待初始化完成
    print_info "等待后端初始化完成..."
    local attempts=0
    while [ $attempts -lt 30 ]; do
        if ! check_service_running "backend-init"; then
            print_success "后端初始化完成"
            break
        fi
        attempts=$((attempts + 1))
        sleep 2
    done
    
    # 第四阶段：核心应用服务
    print_info "第四阶段：启动核心应用服务"
    $COMPOSE_CMD up -d backend frontend jupyterhub gitea
    
    # 等待核心服务就绪
    wait_for_service "backend" 60
    wait_for_service "frontend" 30
    wait_for_service "jupyterhub" 90
    wait_for_service "gitea" 60
    
    # 第五阶段：反向代理和调试服务
    print_info "第五阶段：启动网关和调试服务"
    $COMPOSE_CMD up -d nginx
    
    # 可选服务
    print_info "启动可选服务..."
    $COMPOSE_CMD up -d singleuser-builder k8s-proxy redis-insight gitea-debug-proxy || true
    
    # 等待nginx就绪
    wait_for_service "nginx" 30
    
    print_success "所有服务启动完成！"
}

# 显示服务状态
show_status() {
    print_info "服务状态："
    $COMPOSE_CMD ps
    echo ""
    
    print_info "健康检查状态："
    # 使用简单的表格格式，避免模板解析问题
    $COMPOSE_CMD ps | head -1
    $COMPOSE_CMD ps | tail -n +2 | while read line; do
        echo "$line"
    done
    echo ""
    
    print_info "服务访问地址："
    echo "  主页:          http://localhost:8080/"
    echo "  JupyterHub:    http://localhost:8080/jupyterhub/"
    echo "  API:           http://localhost:8080/api/"
    echo "  Gitea:         http://localhost:8080/gitea/"
    echo "  MinIO:         http://localhost:8080/minio/"
    echo "  phpLDAPadmin:  http://localhost:8080/ldap/"
    echo ""
}

# 运行健康检查
run_health_check() {
    local script_dir=$(cd "$(dirname "$0")" && pwd)
    local health_script="$script_dir/test-health.sh"
    
    if [ -x "$health_script" ]; then
        print_info "运行健康检查..."
        if "$health_script"; then
            print_success "健康检查通过"
            return 0
        else
            print_error "健康检查失败"
            return 1
        fi
    else
        print_warning "健康检查脚本不存在或不可执行: $health_script"
        return 0
    fi
}

# 主函数
main() {
    local do_test=""
    local quick_start=""
    
    # 解析参数
    while [[ $# -gt 0 ]]; do
        case $1 in
            --test)
                do_test="yes"
                shift
                ;;
            --quick)
                quick_start="yes"
                shift
                ;;
            --help|-h)
                echo "用法: $0 [选项]"
                echo ""
                echo "选项:"
                echo "  --test    启动后运行健康检查"
                echo "  --quick   快速启动（跳过分阶段启动）"
                echo "  --help    显示此帮助信息"
                exit 0
                ;;
            *)
                print_error "未知参数: $1"
                exit 1
                ;;
        esac
    done
    
    # 检查依赖
    check_docker_compose
    
    # 不再依赖jq，使用标准工具
    print_info "使用内置状态检查（不依赖jq）"
    
    # 确保在项目根目录
    if [ ! -f "docker-compose.yml" ]; then
        print_error "请在项目根目录运行此脚本"
        exit 1
    fi
    
    # 加载环境变量
    if [ -f ".env" ]; then
        print_info "加载环境变量文件 .env"
        export $(grep -v '^#' .env | xargs)
    fi
    
    # 启动服务
    if [ -n "$quick_start" ]; then
        print_info "快速启动模式"
        $COMPOSE_CMD up -d
        sleep 30  # 等待服务启动
    else
        staged_startup
    fi
    
    # 显示状态
    show_status
    
    # 运行测试
    if [ -n "$do_test" ]; then
        print_info "等待10秒让服务完全稳定..."
        sleep 10
        run_health_check
    fi
    
    print_success "启动脚本执行完成！"
}

# 执行主函数
main "$@"
