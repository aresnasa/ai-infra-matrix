#!/bin/bash

# 测试完整的SSO认证流程

echo "=== 测试完整SSO认证流程 ==="

# 1. 清理mock token
echo "1. 清理浏览器中的mock token..."
curl -s -H "Content-Type: text/html" http://localhost:8080/fix_sso_mock_token.html > /dev/null

# 2. 测试后端登录API
echo "2. 测试后端登录API..."
LOGIN_RESPONSE=$(curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "admin123"}')

echo "登录响应: $LOGIN_RESPONSE"

# 提取token
TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*"' | cut -d'"' -f4)

if [ -n "$TOKEN" ]; then
    echo "✅ 获得JWT token: ${TOKEN:0:50}..."
    
    # 3. 验证token
    echo "3. 验证JWT token..."
    VERIFY_RESPONSE=$(curl -s -X GET http://localhost:8080/api/auth/verify \
      -H "Authorization: Bearer $TOKEN")
    
    echo "验证响应: $VERIFY_RESPONSE"
    
    # 4. 测试JupyterHub访问API
    echo "4. 测试JupyterHub访问API..."
    JUPYTER_ACCESS_RESPONSE=$(curl -s -X POST http://localhost:8080/api/jupyter/access \
      -H "Authorization: Bearer $TOKEN" \
      -H "Content-Type: application/json")
    
    echo "JupyterHub访问API响应: $JUPYTER_ACCESS_RESPONSE"
    
    # 4.1. 测试JupyterHub状态API
    JUPYTER_STATUS_RESPONSE=$(curl -s -X GET http://localhost:8080/api/jupyter/status \
      -H "Authorization: Bearer $TOKEN")
    
    echo "JupyterHub状态API响应: $JUPYTER_STATUS_RESPONSE"
    
    # 5. 测试SSO认证流程
    echo "5. 测试SSO页面加载..."
    SSO_RESPONSE=$(curl -s -L -w "HTTP Status: %{http_code}" \
      http://localhost:8080/sso/jwt_sso_bridge.html)
    
    echo "SSO页面响应状态: $SSO_RESPONSE"
    
    echo "✅ 认证流程测试完成"
    echo "现在可以在浏览器中测试: http://localhost:8080/jupyterhub"
    
else
    echo "❌ 登录失败，无法获得token"
    exit 1
fi
