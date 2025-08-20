#!/bin/bash
# 增强版健康检查脚本，用于验证所有服务的完整功能

set -e

# 设置代理环境变量
export HTTP_PROXY=http://127.0.0.1:7890
export HTTPS_PROXY=http://127.0.0.1:7890
export http_proxy=http://127.0.0.1:7890
export https_proxy=http://127.0.0.1:7890
export ALL_PROXY=socks5://127.0.0.1:7890
export NO_PROXY="localhost,127.0.0.1,::1,.local"

MAX_WAIT=${MAX_WAIT:-300}  # 最大等待时间（秒）
WAIT_INTERVAL=${WAIT_INTERVAL:-10}  # 检查间隔（秒）

echo "🏥 Enhanced Health Check for Ansible Playbook Generator"
echo "====================================================="

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 辅助函数
print_status() {
    local service=$1
    local status=$2
    if [ "$status" = "healthy" ]; then
        echo -e "${GREEN}✅ $service: Healthy${NC}"
    elif [ "$status" = "waiting" ]; then
        echo -e "${YELLOW}⏳ $service: Waiting...${NC}"
    else
        echo -e "${RED}❌ $service: Unhealthy${NC}"
    fi
}

# 检查服务是否运行
check_service_running() {
    local service=$1
    docker-compose -f docker-compose.test.yml ps $service | grep -q "Up"
}

# 检查数据库连接
check_postgres() {
    if check_service_running "postgres-test"; then
        if docker exec postgres-test pg_isready -U test_user -d ansible_generator_test > /dev/null 2>&1; then
            echo "healthy"
        else
            echo "unhealthy"
        fi
    else
        echo "not_running"
    fi
}

# 检查Redis连接
check_redis() {
    if check_service_running "redis-test"; then
        if docker exec redis-test redis-cli ping > /dev/null 2>&1; then
            echo "healthy"
        else
            echo "unhealthy"
        fi
    else
        echo "not_running"
    fi
}

# 检查后端API
check_backend() {
    if check_service_running "backend-test"; then
        if curl -f -s http://localhost:8083/health > /dev/null 2>&1; then
            # 进一步检查API响应
            response=$(curl -s http://localhost:8083/health)
            if echo "$response" | grep -q "ok"; then
                echo "healthy"
            else
                echo "unhealthy"
            fi
        else
            echo "unhealthy"
        fi
    else
        echo "not_running"
    fi
}

# 检查前端
check_frontend() {
    if check_service_running "frontend-test"; then
        if curl -f -s http://localhost:3001 > /dev/null 2>&1; then
            echo "healthy"
        else
            echo "unhealthy"
        fi
    else
        echo "not_running"
    fi
}

# 主健康检查循环
waited=0
echo -e "${BLUE}🔍 Starting health check monitoring...${NC}"
echo ""

while [ $waited -lt $MAX_WAIT ]; do
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "⏱️  Health Check - Elapsed Time: ${waited}s / ${MAX_WAIT}s"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # 检查所有服务
    postgres_status=$(check_postgres)
    redis_status=$(check_redis)
    backend_status=$(check_backend)
    frontend_status=$(check_frontend)
    
    # 显示状态
    print_status "PostgreSQL Test DB" $postgres_status
    print_status "Redis Test Instance" $redis_status
    print_status "Backend API" $backend_status
    print_status "Frontend App" $frontend_status
    
    # 检查是否所有服务都健康
    if [ "$postgres_status" = "healthy" ] && \
       [ "$redis_status" = "healthy" ] && \
       [ "$backend_status" = "healthy" ] && \
       [ "$frontend_status" = "healthy" ]; then
        echo ""
        echo -e "${GREEN}🎉 All services are healthy and ready!${NC}"
        echo ""
        echo "📋 Service URLs:"
        echo "  🌐 Frontend: http://localhost:3001"
        echo "  🔧 Backend API: http://localhost:8083"
        echo "  📚 API Documentation: http://localhost:8083/swagger/index.html"
        echo "  🗄️  PostgreSQL: localhost:5433"
        echo "  🔄 Redis: localhost:6380"
        echo ""
        echo -e "${GREEN}✅ Ready for testing!${NC}"
        exit 0
    fi
    
    # 如果有服务未运行，显示错误
    if [ "$postgres_status" = "not_running" ] || \
       [ "$redis_status" = "not_running" ] || \
       [ "$backend_status" = "not_running" ] || \
       [ "$frontend_status" = "not_running" ]; then
        echo ""
        echo -e "${RED}⚠️  Some services are not running. Check docker-compose status:${NC}"
        docker-compose -f docker-compose.test.yml ps
        echo ""
    fi
    
    echo ""
    echo -e "${YELLOW}⏳ Waiting ${WAIT_INTERVAL} seconds before next check...${NC}"
    sleep $WAIT_INTERVAL
    waited=$((waited + WAIT_INTERVAL))
done

# 超时处理
echo ""
echo -e "${RED}⚠️  Health check timeout after ${MAX_WAIT} seconds${NC}"
echo ""
echo "📊 Final Service Status:"
docker-compose -f docker-compose.test.yml ps
echo ""
echo "🔍 Troubleshooting Commands:"
echo "  docker-compose -f docker-compose.test.yml logs"
echo "  docker-compose -f docker-compose.test.yml logs [service_name]"
echo "  docker-compose -f docker-compose.test.yml restart [service_name]"
echo ""
exit 1
