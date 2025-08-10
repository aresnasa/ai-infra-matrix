#!/bin/bash

echo "测试JupyterHub登录功能..."

# 步骤1: 获取登录页面和CSRF token
echo "1. 获取登录页面..."
COOKIE_JAR=$(mktemp)
LOGIN_PAGE=$(curl -s -c "$COOKIE_JAR" http://localhost:8080/jupyter/hub/login)
XSRF_TOKEN=$(echo "$LOGIN_PAGE" | grep -o 'name="_xsrf" value="[^"]*"' | cut -d'"' -f4)

echo "XSRF Token: ${XSRF_TOKEN:0:50}..."

# 步骤2: 提交登录表单
echo "2. 提交登录表单..."
LOGIN_RESULT=$(curl -s -b "$COOKIE_JAR" -c "$COOKIE_JAR" \
  -X POST \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "username=admin&password=admin123&_xsrf=$XSRF_TOKEN" \
  -w "%{http_code}" \
  http://localhost:8080/jupyter/hub/login)

HTTP_CODE=$(echo "$LOGIN_RESULT" | tail -c 4)
echo "HTTP 状态码: $HTTP_CODE"

# 步骤3: 检查登录结果
if [[ "$HTTP_CODE" == "302" ]]; then
    echo "✅ 登录成功！收到重定向响应"
    
    # 尝试访问用户控制台
    echo "3. 访问用户控制台..."
    CONSOLE_RESULT=$(curl -s -b "$COOKIE_JAR" -w "%{http_code}" http://localhost:8080/jupyter/hub/)
    CONSOLE_CODE=$(echo "$CONSOLE_RESULT" | tail -c 4)
    echo "控制台访问状态码: $CONSOLE_CODE"
    
    if [[ "$CONSOLE_CODE" == "200" ]]; then
        echo "✅ 成功访问用户控制台"
    else
        echo "❌ 无法访问用户控制台"
    fi
else
    echo "❌ 登录失败"
    echo "响应内容:"
    echo "$LOGIN_RESULT" | head -10
fi

# 清理临时文件
rm -f "$COOKIE_JAR"
