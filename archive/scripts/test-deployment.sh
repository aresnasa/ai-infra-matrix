#!/bin/bash

# AI Infrastructure Matrix - 完整测试脚本
echo "🧪 AI Infrastructure Matrix 完整测试..."

# 1. 完全清理环境
echo "🧹 完全清理环境..."
docker-compose down --remove-orphans -v
docker system prune -f

# 2. 一键启动
echo "🚀 执行一键启动..."
./start-services.sh

# 3. 等待所有服务就绪
echo "⏳ 等待服务完全就绪..."
sleep 30

# 4. 检查服务状态
echo "📊 检查所有服务状态..."
docker-compose ps

echo ""
echo "🔍 详细服务验证:"

# 5. 测试各个服务
echo "  ✅ PostgreSQL数据库..."
docker-compose exec -T postgres psql -U postgres -c "SELECT version();" >/dev/null 2>&1 && echo "    ✓ PostgreSQL运行正常" || echo "    ✗ PostgreSQL故障"

echo "  ✅ Redis缓存..."
docker-compose exec -T redis redis-cli ping >/dev/null 2>&1 && echo "    ✓ Redis运行正常" || echo "    ✗ Redis故障"

echo "  ✅ 后端API..."
curl -f http://localhost:8080/api/health >/dev/null 2>&1 && echo "    ✓ 后端API正常" || echo "    ✗ 后端API故障"

echo "  ✅ 前端应用..."
curl -f http://localhost:8080/ >/dev/null 2>&1 && echo "    ✓ 前端应用正常" || echo "    ✗ 前端应用故障"

echo "  ✅ JupyterHub..."
curl -f http://localhost:8080/jupyter/hub/api >/dev/null 2>&1 && echo "    ✓ JupyterHub正常" || echo "    ✗ JupyterHub故障"

echo "  ✅ Nginx代理..."
curl -f http://localhost:8080/health >/dev/null 2>&1 && echo "    ✓ Nginx代理正常" || echo "    ✗ Nginx代理故障"

echo ""
echo "🌐 访问地址测试:"
echo "  主应用: http://localhost:8080"
curl -I http://localhost:8080 2>/dev/null | head -1 | grep -q "200" && echo "    ✓ 主应用可访问" || echo "    ✗ 主应用不可访问"

echo "  JupyterHub: http://localhost:8080/jupyter"
curl -I http://localhost:8080/jupyter 2>/dev/null | head -1 | grep -q "302\|200" && echo "    ✓ JupyterHub可访问" || echo "    ✗ JupyterHub不可访问"

echo "  后端API: http://localhost:8080/api"
curl -I http://localhost:8080/api/health 2>/dev/null | head -1 | grep -q "200" && echo "    ✓ 后端API可访问" || echo "    ✗ 后端API不可访问"

echo ""
echo "📊 最终状态报告:"
healthy_count=$(docker-compose ps --format "table {{.Status}}" | grep -c "healthy")
running_count=$(docker-compose ps --format "table {{.Status}}" | grep -c "Up")
total_count=$(docker-compose ps | wc -l | xargs)
total_count=$((total_count - 1))  # 减去标题行

echo "  总服务数: $total_count"
echo "  运行中: $running_count"
echo "  健康: $healthy_count"

if [ "$running_count" -eq "$total_count" ]; then
    echo ""
    echo "🎉 恭喜！AI Infrastructure Matrix 一键启动测试完全成功！"
    echo "🌟 所有服务已启动并运行正常"
else
    echo ""
    echo "⚠️  部分服务可能存在问题，请检查日志"
    echo "📝 使用以下命令查看详细日志:"
    echo "    docker-compose logs [service_name]"
fi
