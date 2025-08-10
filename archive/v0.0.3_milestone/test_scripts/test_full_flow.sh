#!/bin/bash

echo "完整JupyterHub访问流程测试..."

COOKIE_JAR=$(mktemp)

# 步骤1: 获取登录页面
echo "1. 获取登录页面..."
LOGIN_PAGE=$(curl -s -c "$COOKIE_JAR" http://localhost:8080/jupyter/hub/login)
XSRF_TOKEN=$(echo "$LOGIN_PAGE" | grep -o 'name="_xsrf" value="[^"]*"' | cut -d'"' -f4)

# 步骤2: 登录
echo "2. 登录..."
curl -s -b "$COOKIE_JAR" -c "$COOKIE_JAR" \
  -X POST \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "username=admin&password=admin123&_xsrf=$XSRF_TOKEN" \
  http://localhost:8080/jupyter/hub/login > /dev/null

# 步骤3: 跟踪重定向到spawn页面
echo "3. 访问spawn页面..."
SPAWN_RESPONSE=$(curl -s -b "$COOKIE_JAR" -L http://localhost:8080/jupyter/hub/spawn)

if echo "$SPAWN_RESPONSE" | grep -q "spawn"; then
    echo "✅ 成功访问spawn页面"
    
    # 检查是否有启动表单
    if echo "$SPAWN_RESPONSE" | grep -q "form"; then
        echo "✅ 找到启动表单"
    else
        echo "📝 页面内容预览:"
        echo "$SPAWN_RESPONSE" | head -20
    fi
elif echo "$SPAWN_RESPONSE" | grep -q "JupyterHub"; then
    echo "📋 收到JupyterHub页面，内容预览:"
    echo "$SPAWN_RESPONSE" | head -10
else
    echo "❌ 收到意外响应"
    echo "$SPAWN_RESPONSE" | head -10
fi

# 步骤4: 检查用户状态
echo "4. 检查用户状态..."
USER_INFO=$(curl -s -b "$COOKIE_JAR" http://localhost:8080/jupyter/hub/api/user)
echo "用户信息: $USER_INFO"

rm -f "$COOKIE_JAR"
