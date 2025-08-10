#!/bin/bash
set -e

# 输出启动信息
echo "🚀 AI基础设施矩阵 - Nginx代理服务启动中..."
echo "📅 启动时间: $(date)"
echo "�️ 构建环境: ${BUILD_ENV:-production}"
echo "🔧 调试模式: ${DEBUG_MODE:-false}"

# 处理nginx配置文件
echo "⚙️ 配置nginx..."

# 复制基础配置
cp /tmp/nginx.conf.template /etc/nginx/nginx.conf

# 根据DEBUG_MODE处理调试路由
if [ "${DEBUG_MODE}" = "true" ]; then
    echo "🔧 启用调试模式 - 保留调试路由"
    echo "   🔍 调试工具入口: /debug/"
    echo "   🧪 认证调试器: /debug/debug_jupyterhub_auth.html"
    echo "   🎫 Token管理: /debug/token_setup.html"
    echo "   🧪 认证测试: /debug/test_jupyterhub_auth.html"
    
    # 确保调试文件存在
    if [ -d "/usr/share/nginx/html/debug" ] && [ "$(ls -A /usr/share/nginx/html/debug)" ]; then
        echo "   ✅ 调试文件已加载"
    else
        echo "   ⚠️ 调试文件目录为空"
    fi
else
    echo "🚀 生产模式 - 移除调试路由"
    # 移除调试路由块
    sed -i '/# DEBUG_MODE_BLOCK_START/,/# DEBUG_MODE_BLOCK_END/d' /etc/nginx/nginx.conf
    
    # 同时移除其他调试相关的路由
    sed -i '/# 调试页面/,/}/d' /etc/nginx/nginx.conf
    sed -i '/# Token设置页面/,/}/d' /etc/nginx/nginx.conf
    
    # 确保调试目录被禁用
    echo "<html><body><h1>Debug tools are disabled in production mode</h1></body></html>" > /usr/share/nginx/html/debug/index.html
fi

echo "�🌐 支持功能:"
echo "   ✅ 分布式部署代理"
echo "   ✅ SSO单点登录支持"
echo "   ✅ JupyterHub upstream访问"
echo "   ✅ 动态CORS配置"
echo "   ✅ 认证头转发"

if [ "${DEBUG_MODE}" = "true" ]; then
    echo "   🔧 开发调试工具"
fi

# 检查配置文件
echo "🔧 检查Nginx配置..."
nginx -t

# 显示监听端口
echo "📡 监听端口: 80 (HTTP), 443 (HTTPS预留)"

# 显示静态文件
echo "📁 静态文件目录:"
echo "   SSO桥接: /usr/share/nginx/html/sso/"
echo "   JupyterHub: /usr/share/nginx/html/jupyterhub/"
if [ "${DEBUG_MODE}" = "true" ]; then
    echo "   调试工具: /usr/share/nginx/html/debug/"
else
    echo "   调试工具: 已禁用 (生产模式)"
fi

# 环境变量支持
if [ ! -z "$BACKEND_HOST" ]; then
    echo "🔄 检测到分布式环境变量:"
    echo "   Backend: ${BACKEND_HOST}:${BACKEND_PORT:-8082}"
    echo "   JupyterHub: ${JUPYTERHUB_HOST}:${JUPYTERHUB_PORT:-8000}"
    echo "   Frontend: ${FRONTEND_HOST}:${FRONTEND_PORT:-80}"
fi

echo "✅ Nginx配置验证完成，启动服务..."

# 启动Nginx
exec "$@"
