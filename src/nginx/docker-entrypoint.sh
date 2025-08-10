#!/bin/bash
set -e

# 输出启动信息
echo "🚀 AI基础设施矩阵 - Nginx代理服务启动中..."
echo "📅 启动时间: $(date)"
echo "🌐 支持功能:"
echo "   ✅ 分布式部署代理"
echo "   ✅ SSO单点登录支持"
echo "   ✅ JupyterHub upstream访问"
echo "   ✅ 动态CORS配置"
echo "   ✅ 认证头转发"

# 检查配置文件
echo "🔧 检查Nginx配置..."
nginx -t

# 显示监听端口
echo "📡 监听端口: 80 (HTTP), 443 (HTTPS预留)"

# 显示静态文件
echo "📁 静态文件目录:"
echo "   SSO桥接: /usr/share/nginx/html/sso/"
echo "   JupyterHub: /usr/share/nginx/html/jupyterhub/"
echo "   调试工具: /usr/share/nginx/html/debug.html"

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
