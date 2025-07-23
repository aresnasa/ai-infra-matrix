#!/bin/bash

# Docker测试脚本 - Ansible Playbook Generator
# 用于快速启动、停止和管理Docker环境

set -e

PROJECT_NAME="ansible-playbook-generator"
COMPOSE_FILE="docker-compose.yml"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印带颜色的消息
print_message() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo -e "${BLUE}================================${NC}"
    echo -e "${BLUE} $1${NC}"
    echo -e "${BLUE}================================${NC}"
}

# 检查Docker和docker-compose是否安装
check_requirements() {
    if ! command -v docker &> /dev/null; then
        print_error "Docker未安装，请先安装Docker"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        print_error "docker-compose未安装，请先安装docker-compose"
        exit 1
    fi
}

# 启动服务
start_services() {
    print_header "启动Ansible Playbook Generator服务"
    
    # 检查.env文件是否存在
    if [ ! -f .env ]; then
        print_warning ".env文件不存在，使用.env.example创建"
        cp .env.example .env
        print_message "已创建.env文件，请根据需要修改配置"
    fi
    
    # 构建并启动服务
    print_message "构建并启动Docker容器..."
    docker-compose up -d --build
    
    print_message "等待服务启动..."
    sleep 10
    
    # 检查服务状态
    check_services_health
    
    print_header "服务启动完成"
    print_message "前端访问地址: http://localhost:3001"
    print_message "后端API地址: http://localhost:8082"
    print_message "API文档地址: http://localhost:8082/swagger/index.html"
    print_message "健康检查地址: http://localhost:8082/api/health"
}

# 停止服务
stop_services() {
    print_header "停止Ansible Playbook Generator服务"
    docker-compose down
    print_message "服务已停止"
}

# 重启服务
restart_services() {
    print_header "重启Ansible Playbook Generator服务"
    docker-compose down
    docker-compose up -d --build
    print_message "等待服务启动..."
    sleep 10
    check_services_health
    print_message "服务重启完成"
}

# 查看日志
view_logs() {
    local service=$1
    if [ -z "$service" ]; then
        print_message "显示所有服务日志..."
        docker-compose logs -f
    else
        print_message "显示${service}服务日志..."
        docker-compose logs -f "$service"
    fi
}

# 检查服务健康状态
check_services_health() {
    print_message "检查服务健康状态..."
    
    # 检查PostgreSQL
    if docker-compose exec -T postgres pg_isready -U postgres -d ansible_playbook_generator > /dev/null 2>&1; then
        print_message "✓ PostgreSQL 正常运行"
    else
        print_warning "✗ PostgreSQL 状态异常"
    fi
    
    # 检查Redis
    if docker-compose exec -T redis redis-cli -a ansible-redis-password ping > /dev/null 2>&1; then
        print_message "✓ Redis 正常运行"
    else
        print_warning "✗ Redis 状态异常"
    fi
    
    # 检查后端API
    if curl -f http://localhost:8082/api/health > /dev/null 2>&1; then
        print_message "✓ 后端API 正常运行"
    else
        print_warning "✗ 后端API 状态异常"
    fi
    
    # 检查前端
    if curl -f http://localhost:3001 > /dev/null 2>&1; then
        print_message "✓ 前端服务 正常运行"
    else
        print_warning "✗ 前端服务 状态异常"
    fi
}

# 清理Docker资源
cleanup() {
    print_header "清理Docker资源"
    docker-compose down -v --remove-orphans
    docker system prune -f
    print_message "清理完成"
}

# 显示服务状态
show_status() {
    print_header "服务状态"
    docker-compose ps
    echo ""
    check_services_health
}

# 进入容器shell
enter_container() {
    local service=$1
    if [ -z "$service" ]; then
        print_error "请指定要进入的容器名称 (backend/frontend/postgres/redis)"
        exit 1
    fi
    
    print_message "进入${service}容器..."
    case $service in
        "backend")
            docker-compose exec backend sh
            ;;
        "frontend")
            docker-compose exec frontend sh
            ;;
        "postgres")
            docker-compose exec postgres psql -U postgres -d ansible_playbook_generator
            ;;
        "redis")
            docker-compose exec redis redis-cli -a ansible-redis-password
            ;;
        *)
            print_error "不支持的容器名称: $service"
            exit 1
            ;;
    esac
}

# 测试API端点
test_api() {
    print_header "测试API端点"
    
    # 测试健康检查
    print_message "测试健康检查端点..."
    if curl -s http://localhost:8082/api/health | jq . > /dev/null 2>&1; then
        print_message "✓ 健康检查API正常"
        curl -s http://localhost:8082/api/health | jq .
    else
        print_warning "✗ 健康检查API异常"
    fi
    
    echo ""
    print_message "更多API测试请访问: http://localhost:8082/swagger/index.html"
}

# 显示帮助信息
show_help() {
    cat << EOF
Ansible Playbook Generator Docker测试脚本

用法: $0 [命令] [参数]

命令:
    start           启动所有服务
    stop            停止所有服务
    restart         重启所有服务
    status          显示服务状态
    logs [service]  查看日志 (可选指定服务名)
    health          检查服务健康状态
    test            测试API端点
    shell [service] 进入容器 (backend/frontend/postgres/redis)
    cleanup         清理Docker资源
    help            显示此帮助信息

示例:
    $0 start                # 启动所有服务
    $0 logs backend         # 查看后端日志
    $0 shell postgres       # 进入PostgreSQL容器
    $0 test                 # 测试API端点

服务访问地址:
    前端: http://localhost:3001
    后端API: http://localhost:8082
    API文档: http://localhost:8082/swagger/index.html
    健康检查: http://localhost:8082/api/health

EOF
}

# 主函数
main() {
    check_requirements
    
    case ${1:-help} in
        "start")
            start_services
            ;;
        "stop")
            stop_services
            ;;
        "restart")
            restart_services
            ;;
        "logs")
            view_logs "$2"
            ;;
        "status")
            show_status
            ;;
        "health")
            check_services_health
            ;;
        "test")
            test_api
            ;;
        "shell")
            enter_container "$2"
            ;;
        "cleanup")
            cleanup
            ;;
        "help"|"--help"|"-h")
            show_help
            ;;
        *)
            print_error "未知命令: $1"
            show_help
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"
