#!/bin/bash
# 测试nginx直接代理到JupyterHub的功能
# 不再使用wrapper页面

echo "🧪 测试JupyterHub直接代理功能"
echo "============================================"

# 清理之前的cookies
rm -f test_cookies.txt

# 1. 测试/jupyterhub重定向
echo "1. 测试 /jupyterhub 重定向..."
REDIRECT_RESPONSE=$(curl -s -I http://localhost:8080/jupyterhub)
if echo "$REDIRECT_RESPONSE" | grep -q "301 Moved Permanently"; then
    echo "✅ /jupyterhub 正确返回301重定向"
    LOCATION=$(echo "$REDIRECT_RESPONSE" | grep -i "location:" | cut -d' ' -f2 | tr -d '\r')
    echo "   重定向到: $LOCATION"
else
    echo "❌ /jupyterhub 重定向失败"
    echo "$REDIRECT_RESPONSE"
    exit 1
fi

# 2. 测试最终JupyterHub页面访问
echo
echo "2. 测试跟随重定向到JupyterHub登录页..."
LOGIN_PAGE=$(curl -s -L -c test_cookies.txt http://localhost:8080/jupyterhub)
if echo "$LOGIN_PAGE" | grep -q "JupyterHub"; then
    echo "✅ 成功访问JupyterHub登录页"
    if echo "$LOGIN_PAGE" | grep -q "login"; then
        echo "✅ 页面包含登录表单"
    else
        echo "⚠️  页面不包含登录表单"
    fi
else
    echo "❌ 无法访问JupyterHub登录页"
    exit 1
fi

# 3. 测试直接访问/jupyter/hub/
echo
echo "3. 测试直接访问 /jupyter/hub/..."
DIRECT_ACCESS=$(curl -s -w "%{http_code}" http://localhost:8080/jupyter/hub/ -o /dev/null)
if [ "$DIRECT_ACCESS" = "200" ]; then
    echo "✅ /jupyter/hub/ 直接访问成功"
else
    echo "❌ /jupyter/hub/ 直接访问失败 (HTTP $DIRECT_ACCESS)"
fi

# 4. 测试登录功能
echo
echo "4. 测试登录功能..."
# 先获取XSRF token
XSRF_TOKEN=$(curl -s -c test_cookies.txt -b test_cookies.txt http://localhost:8080/jupyter/hub/login | grep '_xsrf' | grep 'value=' | sed 's/.*value="\([^"]*\)".*/\1/')

if [ ! -z "$XSRF_TOKEN" ]; then
    echo "✅ 获取XSRF token成功: ${XSRF_TOKEN:0:20}..."
    
    # 尝试登录
    LOGIN_RESULT=$(curl -s -X POST \
        -c test_cookies.txt \
        -b test_cookies.txt \
        -d "username=admin" \
        -d "password=admin123" \
        -d "_xsrf=$XSRF_TOKEN" \
        -w "%{http_code}" \
        http://localhost:8080/jupyter/hub/login \
        -o /dev/null)
    
    if [ "$LOGIN_RESULT" = "302" ] || [ "$LOGIN_RESULT" = "200" ]; then
        echo "✅ 登录请求成功 (HTTP $LOGIN_RESULT)"
        
        # 检查是否重定向到spawn页面
        SPAWN_CHECK=$(curl -s -L -c test_cookies.txt -b test_cookies.txt http://localhost:8080/jupyter/hub/spawn)
        if echo "$SPAWN_CHECK" | grep -q "spawn\|server\|ready"; then
            echo "✅ 登录后成功访问spawn页面"
        else
            echo "⚠️  登录后页面检查异常"
        fi
    else
        echo "❌ 登录失败 (HTTP $LOGIN_RESULT)"
    fi
else
    echo "❌ 无法获取XSRF token"
fi

# 5. 清理
echo
echo "5. 清理测试文件..."
rm -f test_cookies.txt

echo
echo "============================================"
echo "🎉 测试完成！JupyterHub现在通过nginx直接代理访问"
echo "用户可以直接访问: http://localhost:8080/jupyterhub"
echo "这会自动重定向到: http://localhost:8080/jupyter/hub/"
echo "不再需要wrapper页面，所有认证都通过JupyterHub处理"
