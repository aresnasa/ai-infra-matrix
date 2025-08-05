#!/bin/bash
# 测试健康检查脚本
echo "🏥 测试 Docker Compose 健康检查..."
echo "========================================"

# 检查是否安装了 docker-compose
if ! command -v docker-compose &> /dev/null; then
    echo "❌ docker-compose 未安装，请先安装 docker-compose"
    exit 1
fi

# 启动服务
echo "🚀 启动所有服务..."
docker-compose down
docker-compose up --build -d

echo ""
echo "⏳ 等待服务启动..."
sleep 5

# 检查服务状态
echo ""
echo "📊 检查服务状态:"
echo "---------------"
docker-compose ps

echo ""
echo "🔍 检查服务健康状态:"
echo "------------------"

# 等待所有服务健康
max_wait=300  # 最多等待5分钟
waited=0

while [ $waited -lt $max_wait ]; do
    postgres_healthy=$(docker-compose ps postgres | grep -c "healthy")
    redis_healthy=$(docker-compose ps redis | grep -c "healthy")
    backend_healthy=$(docker-compose ps backend | grep -c "healthy")
    frontend_healthy=$(docker-compose ps frontend | grep -c "healthy")
    
    echo "PostgreSQL: $([ $postgres_healthy -eq 1 ] && echo "✅ 健康" || echo "⏳ 等待中...")"
    echo "Redis: $([ $redis_healthy -eq 1 ] && echo "✅ 健康" || echo "⏳ 等待中...")"
    echo "Backend: $([ $backend_healthy -eq 1 ] && echo "✅ 健康" || echo "⏳ 等待中...")"
    echo "Frontend: $([ $frontend_healthy -eq 1 ] && echo "✅ 健康" || echo "⏳ 等待中...")"
    
    if [ $postgres_healthy -eq 1 ] && [ $redis_healthy -eq 1 ] && [ $backend_healthy -eq 1 ] && [ $frontend_healthy -eq 1 ]; then
        echo ""
        echo "🎉 所有服务都已启动并健康!"
        break
    fi
    
    echo "---"
    sleep 10
    waited=$((waited + 10))
    
    if [ $waited -ge $max_wait ]; then
        echo ""
        echo "⚠️  等待超时，某些服务可能未正常启动"
        echo "请检查日志: docker-compose logs"
        break
    fi
done

echo ""
echo "🧪 测试健康检查端点:"
echo "-------------------"

# 测试后端健康检查
echo "测试后端健康检查..."
if curl -s -f http://localhost:8082/api/health > /dev/null; then
    response=$(curl -s http://localhost:8082/api/health)
    echo "✅ 后端健康检查响应: $response"
else
    echo "❌ 后端健康检查失败"
fi

# 测试前端
echo "测试前端..."
if curl -s -f http://localhost:3001 > /dev/null; then
    echo "✅ 前端可访问"
else
    echo "❌ 前端不可访问"
fi

echo ""
echo "📋 服务启动顺序验证:"
echo "------------------"
echo "根据 depends_on 配置，服务应该按以下顺序启动:"
echo "1. PostgreSQL (无依赖)"
echo "2. Redis (无依赖)" 
echo "3. Backend (依赖 PostgreSQL 和 Redis)"
echo "4. Frontend (依赖 Backend)"

echo ""
echo "🔧 有用的命令:"
echo "-------------"
echo "查看所有服务状态: docker-compose ps"
echo "查看服务日志: docker-compose logs [service_name]"
echo "停止所有服务: docker-compose down"
echo "重启服务: docker-compose restart [service_name]"

echo ""
echo "✨ 健康检查测试完成!"
