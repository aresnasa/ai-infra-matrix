#!/bin/bash
# 测试优化后的JupyterHub认证传递
echo "🚀 测试优化后的JupyterHub认证传递"
echo "=========================================="

# 清理之前的测试文件
rm -f test_cookies.txt

echo "1. 测试未登录状态 - 访问 /jupyterhub 应该直接重定向到 /sso/"
REDIRECT_TEST=$(curl -s -L -w "%{url_effective}" http://localhost:8080/jupyterhub -o /dev/null)
echo "最终URL: $REDIRECT_TEST"
if echo "$REDIRECT_TEST" | grep -q "sso"; then
    echo "✅ 未登录状态正确重定向到SSO页面"
else
    echo "❌ 重定向行为异常"
fi

echo
echo "2. 模拟用户登录获取token..."
LOGIN_RESPONSE=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin123"}' \
    -c test_cookies.txt \
    http://localhost:8080/api/auth/login 2>/dev/null)

if [ $? -eq 0 ] && echo "$LOGIN_RESPONSE" | grep -q "token"; then
    echo "✅ 后端登录成功"
    TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    if [ ! -z "$TOKEN" ]; then
        echo "✅ 获取到token: ${TOKEN:0:20}..."
    fi
else
    echo "❌ 后端登录失败"
    exit 1
fi

echo
echo "3. 测试 /jupyterhub-direct 路径 (带token参数)..."
if [ ! -z "$TOKEN" ]; then
    DIRECT_URL="http://localhost:8080/jupyterhub-direct/?auth_token=$TOKEN&username=admin"
    DIRECT_RESPONSE=$(curl -s -w "%{http_code}" "$DIRECT_URL" -o /tmp/jupyterhub_direct.html)
    
    echo "直接访问响应码: $DIRECT_RESPONSE"
    if [ "$DIRECT_RESPONSE" = "200" ] || [ "$DIRECT_RESPONSE" = "302" ]; then
        echo "✅ JupyterHub直接访问路径工作正常"
    else
        echo "❌ JupyterHub直接访问路径异常"
    fi
else
    echo "⏭️  跳过直接访问测试 (无token)"
fi

echo
echo "4. 测试认证桥接页面内容..."
AUTH_BRIDGE_CONTENT=$(curl -s http://localhost:8080/jupyterhub)
if echo "$AUTH_BRIDGE_CONTENT" | grep -q "预检查"; then
    echo "✅ 认证桥接页面包含预检查逻辑"
else
    echo "⚠️  认证桥接页面可能有问题"
fi

echo
echo "5. 检查各个组件状态..."
echo "Backend API: $(curl -s -w "%{http_code}" http://localhost:8080/api/auth/verify -H "Authorization: Bearer $TOKEN" -o /dev/null)"
echo "JupyterHub API: $(curl -s -w "%{http_code}" http://localhost:8080/jupyter/hub/api -o /dev/null)"
echo "SSO页面: $(curl -s -w "%{http_code}" http://localhost:8080/sso/ -o /dev/null)"

# 清理
rm -f test_cookies.txt /tmp/jupyterhub_direct.html

echo
echo "=========================================="
echo "🎯 优化后的认证流程："
echo "1. 用户访问 /jupyterhub"
echo "2. 认证桥接页面预检查localStorage中的token"
echo "3. 无token: 立即跳转到 /sso/ (避免白屏)"
echo "4. 有token: 验证后跳转到 /jupyterhub-direct/?auth_token=..."
echo "5. JupyterHub通过URL参数自动完成认证"
echo
echo "🔧 测试说明："
echo "- 第一次访问会很快跳转到登录页面"
echo "- 登录后再访问会直接进入JupyterHub"
echo "- 不再需要手动刷新或点击按钮"
