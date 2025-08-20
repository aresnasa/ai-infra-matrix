#!/bin/bash

# SaltStack 集成测试脚本
# 验证 SaltStack 服务是否正常工作

set -euo pipefail

# 颜色定义
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

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# 检查服务是否运行
check_service() {
    local service_name="$1"
    if docker compose ps --format "table {{.Name}}\t{{.Status}}" | grep -q "$service_name.*Up"; then
        print_success "$service_name 服务正在运行"
        return 0
    else
        print_error "$service_name 服务未运行"
        return 1
    fi
}

# 检查API端点
check_api() {
    local endpoint="$1"
    local description="$2"
    
    print_info "检查 $description API: $endpoint"
    
    if curl -s -f -H "Authorization: Bearer ${JWT_TOKEN:-}" "http://localhost:8080$endpoint" > /dev/null; then
        print_success "$description API 响应正常"
        return 0
    else
        print_error "$description API 无响应"
        return 1
    fi
}

# 主测试函数
main() {
    print_info "开始 SaltStack 集成测试"
    echo "================================"
    
    # 检查 docker-compose 是否可用
    if ! command -v docker compose &> /dev/null; then
        print_error "docker compose 命令不可用"
        exit 1
    fi
    
    # 检查核心服务状态
    print_info "检查服务状态..."
    check_service "ai-infra-backend"
    check_service "ai-infra-frontend" 
    check_service "ai-infra-nginx"
    
    # 检查 SaltStack 服务
    if check_service "ai-infra-saltstack"; then
        print_info "检查 SaltStack 服务详细状态..."
        
        # 检查容器内的服务
        docker exec ai-infra-saltstack supervisorctl status | while read line; do
            if echo "$line" | grep -q "RUNNING"; then
                service=$(echo "$line" | awk '{print $1}')
                print_success "SaltStack 进程 $service 正在运行"
            else
                service=$(echo "$line" | awk '{print $1}')
                status=$(echo "$line" | awk '{print $2}')
                print_warning "SaltStack 进程 $service 状态: $status"
            fi
        done
        
        # 检查端口
        print_info "检查 SaltStack 端口..."
        for port in 4505 4506 8000; do
            if docker exec ai-infra-saltstack netstat -ln | grep -q ":$port "; then
                print_success "端口 $port 正在监听"
            else
                print_warning "端口 $port 未监听"
            fi
        done
    fi
    
    # 等待服务启动
    print_info "等待服务完全启动..."
    sleep 10
    
    # 检查 API 端点（需要先登录获取token）
    print_info "检查后端API连通性..."
    
    # 检查健康检查端点
    if curl -s -f "http://localhost:8080/api/health" > /dev/null; then
        print_success "后端健康检查API正常"
        
        # 尝试检查 SaltStack API（可能需要认证）
        print_info "检查 SaltStack API 端点..."
        
        # 检查状态端点（演示模式应该可以访问）
        response=$(curl -s "http://localhost:8080/api/saltstack/status" || echo "")
        if echo "$response" | grep -q "demo\|status"; then
            print_success "SaltStack 状态 API 返回数据"
            echo "响应示例: $(echo "$response" | head -c 100)..."
        else
            print_warning "SaltStack 状态 API 需要认证或返回错误"
        fi
        
        # 检查 Minions 端点
        response=$(curl -s "http://localhost:8080/api/saltstack/minions" || echo "")
        if echo "$response" | grep -q "demo\|data"; then
            print_success "SaltStack Minions API 返回数据"
        else
            print_warning "SaltStack Minions API 需要认证或返回错误"
        fi
        
        # 检查 Jobs 端点  
        response=$(curl -s "http://localhost:8080/api/saltstack/jobs" || echo "")
        if echo "$response" | grep -q "demo\|data"; then
            print_success "SaltStack Jobs API 返回数据"
        else
            print_warning "SaltStack Jobs API 需要认证或返回错误"
        fi
        
    else
        print_error "后端健康检查API无响应，跳过其他API测试"
    fi
    
    # 检查前端
    print_info "检查前端访问..."
    if curl -s -f "http://localhost:8080/" > /dev/null; then
        print_success "前端页面可访问"
    else
        print_error "前端页面无法访问"
    fi
    
    # 检查 SaltStack 配置文件
    print_info "检查 SaltStack 配置..."
    if docker exec ai-infra-saltstack test -f /etc/salt/master; then
        print_success "Salt Master 配置文件存在"
    else
        print_error "Salt Master 配置文件缺失"
    fi
    
    if docker exec ai-infra-saltstack test -f /etc/salt/minion; then
        print_success "Salt Minion 配置文件存在"
    else
        print_error "Salt Minion 配置文件缺失"
    fi
    
    # 显示日志摘要
    print_info "显示 SaltStack 服务日志摘要..."
    echo "最近的日志:"
    docker logs ai-infra-saltstack --tail 10 2>/dev/null || print_warning "无法获取 SaltStack 日志"
    
    echo ""
    print_info "SaltStack 集成测试完成！"
    print_info "访问地址:"
    echo "  🌐 前端: http://localhost:8080"
    echo "  📊 Slurm 面板(带 SaltStack): http://localhost:8080/slurm"
    echo "  🔧 SaltStack API: http://localhost:8080/api/saltstack/status"
}

main "$@"
