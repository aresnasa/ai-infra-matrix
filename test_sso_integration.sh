#!/bin/bash

echo "🔧 AI-Infra-Matrix SSO 测试脚本"
echo "=================================="

# 测试前端是否正常运行
echo "1. 测试前端服务..."
FRONTEND_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/)
if [ "$FRONTEND_STATUS" = "200" ]; then
    echo "✅ 前端服务正常运行 (HTTP $FRONTEND_STATUS)"
else
    echo "❌ 前端服务异常 (HTTP $FRONTEND_STATUS)"
fi

# 测试后端API
echo -e "\n2. 测试后端API..."
BACKEND_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/health)
if [ "$BACKEND_STATUS" = "200" ]; then
    echo "✅ 后端API正常运行 (HTTP $BACKEND_STATUS)"
else
    echo "❌ 后端API异常 (HTTP $BACKEND_STATUS)"
fi

# 测试JupyterHub服务
echo -e "\n3. 测试JupyterHub服务..."
JUPYTER_STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/jupyter/)
if [ "$JUPYTER_STATUS" = "200" ] || [ "$JUPYTER_STATUS" = "302" ]; then
    echo "✅ JupyterHub服务正常运行 (HTTP $JUPYTER_STATUS)"
else
    echo "❌ JupyterHub服务异常 (HTTP $JUPYTER_STATUS)"
fi

# 测试登录API
echo -e "\n4. 测试登录API..."
LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8080/api/auth/login \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin123"}')

if echo "$LOGIN_RESPONSE" | grep -q "token"; then
    echo "✅ 登录API正常工作"
    
    # 提取token
    TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    
    # 测试JupyterHub登录token生成
    echo -e "\n5. 测试JupyterHub登录token生成..."
    JUPYTER_TOKEN_RESPONSE=$(curl -s -X POST http://localhost:8080/api/auth/jupyterhub-login \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d '{"username":"admin"}')
    
    if echo "$JUPYTER_TOKEN_RESPONSE" | grep -q "success.*true"; then
        echo "✅ JupyterHub登录token生成成功"
        echo "SSO功能已准备就绪！"
    else
        echo "❌ JupyterHub登录token生成失败"
        echo "响应: $JUPYTER_TOKEN_RESPONSE"
    fi
else
    echo "❌ 登录API异常"
    echo "响应: $LOGIN_RESPONSE"
fi

echo -e "\n6. 服务状态总览:"
docker-compose ps --format "table {{.Name}}\t{{.Status}}"

echo -e "\n🎉 测试完成！"
echo "✨ 访问地址:"
echo "   - 主页面: http://localhost:8080"
echo "   - JupyterHub: http://localhost:8080/jupyter"
echo "   - 管理面板: http://localhost:8080 (登录后)"
