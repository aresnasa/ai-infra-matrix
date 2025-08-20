#!/bin/bash

echo "🎯 AI-Infra-Matrix SSO 演示脚本"
echo "================================"

# 获取登录token
echo "1. 正在登录系统..."
LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8080/api/auth/login \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin123"}')

TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r .token)

if [ "$TOKEN" != "null" ] && [ -n "$TOKEN" ]; then
    echo "✅ 登录成功！Token: ${TOKEN:0:30}..."
    
    # 生成JupyterHub登录token
    echo -e "\n2. 生成JupyterHub登录token..."
    JUPYTER_RESPONSE=$(curl -s -X POST http://localhost:8080/api/auth/jupyterhub-login \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer $TOKEN" \
        -d '{"username":"admin"}')
    
    JUPYTER_TOKEN=$(echo "$JUPYTER_RESPONSE" | jq -r .token)
    SUCCESS=$(echo "$JUPYTER_RESPONSE" | jq -r .success)
    
    if [ "$SUCCESS" = "true" ] && [ "$JUPYTER_TOKEN" != "null" ]; then
        echo "✅ JupyterHub token生成成功！"
        echo "   JupyterHub Token: ${JUPYTER_TOKEN:0:30}..."
        
        # 构建登录URL
        JUPYTER_URL="http://localhost:8080/jupyter/hub/login?token=$JUPYTER_TOKEN&username=admin"
        echo -e "\n3. JupyterHub SSO登录URL已生成："
        echo "   $JUPYTER_URL"
        
        echo -e "\n🎉 SSO演示完成！"
        echo "📖 使用说明："
        echo "   1. 访问主页面: http://localhost:8080"
        echo "   2. 使用 admin/admin123 登录"
        echo "   3. 导航到JupyterHub页面"
        echo "   4. 点击'进入JupyterHub'按钮"
        echo "   5. 系统将自动在新窗口打开JupyterHub并完成登录"
        
        echo -e "\n✨ 关键优势："
        echo "   🔑 单点登录: 只需登录一次"
        echo "   🚀 无缝跳转: 自动进入JupyterHub"
        echo "   🛡️ 安全认证: JWT token安全传递"
        echo "   💫 用户友好: 清晰的状态提示"
        
    else
        echo "❌ JupyterHub token生成失败"
        echo "响应: $JUPYTER_RESPONSE"
    fi
else
    echo "❌ 登录失败"
    echo "响应: $LOGIN_RESPONSE"
fi

echo -e "\n🔧 当前服务状态："
docker-compose ps --format "table {{.Name}}\t{{.Status}}" | grep -E "(frontend|backend|jupyterhub|nginx)"
