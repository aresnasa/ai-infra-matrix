#!/bin/bash
# 测试认证传递功能 - 模拟用户登录后访问JupyterHub
echo "🧪 测试JupyterHub认证传递功能"
echo "=========================================="

# 清理之前的测试文件
rm -f test_auth_cookies.txt

echo "1. 测试访问认证桥接页面..."
AUTH_BRIDGE_RESPONSE=$(curl -s -w "%{http_code}" http://localhost:8080/jupyterhub -o /tmp/auth_bridge.html)
if [ "$AUTH_BRIDGE_RESPONSE" = "200" ]; then
    echo "✅ 认证桥接页面访问成功"
    if grep -q "JupyterHub认证中转" /tmp/auth_bridge.html; then
        echo "✅ 页面内容正确"
    else
        echo "⚠️  页面内容异常"
    fi
else
    echo "❌ 认证桥接页面访问失败 (HTTP $AUTH_BRIDGE_RESPONSE)"
fi

echo
echo "2. 测试后端登录API..."
# 模拟用户登录获取token
LOGIN_RESPONSE=$(curl -s -X POST \
    -H "Content-Type: application/json" \
    -d '{"username":"admin","password":"admin123"}' \
    -c test_auth_cookies.txt \
    http://localhost:8080/api/auth/login 2>/dev/null)

if [ $? -eq 0 ] && echo "$LOGIN_RESPONSE" | grep -q "token"; then
    echo "✅ 后端登录成功"
    TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
    if [ ! -z "$TOKEN" ]; then
        echo "✅ 获取到token: ${TOKEN:0:20}..."
        # 模拟保存到localStorage (实际由浏览器JS处理)
        echo "ℹ️  Token将由浏览器JS保存到localStorage"
    else
        echo "❌ 未能获取token"
    fi
else
    echo "❌ 后端登录失败"
    echo "响应: $LOGIN_RESPONSE"
fi

echo
echo "3. 测试token验证API..."
if [ ! -z "$TOKEN" ]; then
    VERIFY_RESPONSE=$(curl -s -w "%{http_code}" \
        -H "Authorization: Bearer $TOKEN" \
        http://localhost:8080/api/auth/verify \
        -o /tmp/verify_response.json)
    
    if [ "$VERIFY_RESPONSE" = "200" ]; then
        echo "✅ Token验证成功"
        USER_INFO=$(cat /tmp/verify_response.json)
        echo "✅ 用户信息: $USER_INFO"
    else
        echo "❌ Token验证失败 (HTTP $VERIFY_RESPONSE)"
    fi
else
    echo "⏭️  跳过token验证 (无token)"
fi

echo
echo "4. 测试SSO登录页面..."
SSO_RESPONSE=$(curl -s -w "%{http_code}" http://localhost:8080/sso/ -o /tmp/sso_page.html)
if [ "$SSO_RESPONSE" = "200" ]; then
    echo "✅ SSO登录页面访问成功"
    if grep -q "单点登录" /tmp/sso_page.html; then
        echo "✅ SSO页面内容正确"
    else
        echo "⚠️  SSO页面内容异常"
    fi
else
    echo "❌ SSO登录页面访问失败 (HTTP $SSO_RESPONSE)"
fi

echo
echo "5. 测试JupyterHub后端集成认证..."
# 检查JupyterHub是否使用了后端集成配置
JUPYTERHUB_STATUS=$(curl -s -w "%{http_code}" http://localhost:8080/jupyter/hub/api -o /dev/null)
if [ "$JUPYTERHUB_STATUS" = "200" ] || [ "$JUPYTERHUB_STATUS" = "403" ]; then
    echo "✅ JupyterHub API响应正常 (HTTP $JUPYTERHUB_STATUS)"
else
    echo "❌ JupyterHub API异常 (HTTP $JUPYTERHUB_STATUS)"
fi

echo
echo "6. 检查JupyterHub自动登录端点..."
AUTO_LOGIN_STATUS=$(curl -s -I http://localhost:8080/jupyter/auto-login | head -1)
echo "ℹ️  自动登录端点状态: $AUTO_LOGIN_STATUS"

# 清理临时文件
rm -f test_auth_cookies.txt /tmp/auth_bridge.html /tmp/verify_response.json /tmp/sso_page.html

echo
echo "=========================================="
echo "🎯 认证传递流程说明："
echo "1. 用户访问 /jupyterhub"
echo "2. 认证桥接页面检查localStorage中的token"
echo "3. 如果token有效，自动提交到 /jupyter/auto-login"
echo "4. 如果token无效或不存在，重定向到 /sso/ 登录"
echo "5. 登录成功后，返回JupyterHub并自动认证"
echo
echo "🔧 下一步："
echo "- 打开浏览器访问 http://localhost:8080/sso/ 先登录"
echo "- 然后访问 http://localhost:8080/jupyterhub 测试自动认证"
