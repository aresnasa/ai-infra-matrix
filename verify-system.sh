#!/bin/bash

echo "🔧 AI-Infra-Matrix v0.0.3.1 功能验证"
echo "======================================"

# 测试主页
echo -n "📄 测试主页: "
if curl -s -o /dev/null -w "%{http_code}" http://localhost:8080 | grep -q "200"; then
    echo "✅ 正常"
else
    echo "❌ 失败"
fi

# 测试API健康检查
echo -n "🔌 测试API健康检查: "
HEALTH_RESPONSE=$(curl -s http://localhost:8080/api/health)
if echo "$HEALTH_RESPONSE" | grep -q "healthy"; then
    echo "✅ 正常"
else
    echo "❌ 失败"
fi

# 测试静态资源
echo -n "🎨 测试CSS资源: "
if curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/static/css/main.6c67a7d4.css | grep -q "200"; then
    echo "✅ 正常"
else
    echo "❌ 失败"
fi

echo -n "⚙️ 测试JS资源: "
if curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/static/js/main.1e36a283.js | grep -q "200"; then
    echo "✅ 正常"
else
    echo "❌ 失败"
fi

# 测试Favicon
echo -n "🌟 测试Favicon: "
if curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/favicon.svg | grep -q "200"; then
    echo "✅ 正常"
else
    echo "❌ 失败"
fi

# 测试CORS头
echo -n "🌐 测试CORS头: "
CORS_HEADER=$(curl -s -I http://localhost:8080/api/health | grep -i "access-control-allow-origin")
if [ ! -z "$CORS_HEADER" ]; then
    echo "✅ 正常"
else
    echo "❌ 失败"
fi

# 测试容器状态
echo ""
echo "📊 容器状态:"
docker-compose ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}" | grep -E "(frontend|backend|nginx)"

echo ""
echo "🎯 核心功能验证完成！"
echo ""
echo "💡 访问指南:"
echo "   - 主页: http://localhost:8080"
echo "   - 登录: http://localhost:8080/login"
echo "   - API健康: http://localhost:8080/api/health"
echo "   - JupyterHub: http://localhost:8000"
echo ""
echo "🔑 默认登录信息:"
echo "   - 用户名: admin"
echo "   - 密码: admin123"
