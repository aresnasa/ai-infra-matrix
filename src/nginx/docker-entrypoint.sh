#!/bin/bash
set -e

# 输出启动信息
echo "🚀 AI基础设施矩阵 - Nginx代理服务启动中..."
echo "📅 启动时间: $(date)"
echo "🏗️ 构建环境: ${BUILD_ENV:-production}"
echo "🔧 调试模式: ${DEBUG_MODE:-false}"
echo "🔒 TLS 模式: ${ENABLE_TLS:-false}"

# 处理nginx配置文件
echo "⚙️ 配置nginx..."

# 移除官方默认站点，确保我们的 server-main.conf 生效
rm -f /etc/nginx/conf.d/default.conf || true

# 处理环境变量替换 (必须在nginx -t之前)
echo "🔧 处理配置文件中的环境变量..."

# 设置默认值
export GITEA_ALIAS_ADMIN_TO="${GITEA_ALIAS_ADMIN_TO:-admin}"
export GITEA_ADMIN_EMAIL="${GITEA_ADMIN_EMAIL:-admin@example.com}"
export FRONTEND_HOST="${FRONTEND_HOST:-frontend}"
export FRONTEND_PORT="${FRONTEND_PORT:-80}"
export BACKEND_HOST="${BACKEND_HOST:-backend}"
export BACKEND_PORT="${BACKEND_PORT:-8082}"
export JUPYTERHUB_HOST="${JUPYTERHUB_HOST:-jupyterhub}"
export JUPYTERHUB_PORT="${JUPYTERHUB_PORT:-8000}"
export NIGHTINGALE_HOST="${NIGHTINGALE_HOST:-nightingale}"
export NIGHTINGALE_PORT="${NIGHTINGALE_PORT:-17000}"
export EXTERNAL_SCHEME="${EXTERNAL_SCHEME:-http}"
export EXTERNAL_HOST_ONLY="${EXTERNAL_HOST:-localhost}"
export EXTERNAL_PORT="${EXTERNAL_PORT:-80}"
export ENABLE_TLS="${ENABLE_TLS:-false}"

# 组合 EXTERNAL_HOST 包含端口 (仅当端口不是默认的80或443时)
if [ "$EXTERNAL_PORT" = "80" ] && [ "$EXTERNAL_SCHEME" = "http" ]; then
    export EXTERNAL_HOST="${EXTERNAL_HOST_ONLY}"
elif [ "$EXTERNAL_PORT" = "443" ] && [ "$EXTERNAL_SCHEME" = "https" ]; then
    export EXTERNAL_HOST="${EXTERNAL_HOST_ONLY}"
else
    export EXTERNAL_HOST="${EXTERNAL_HOST_ONLY}:${EXTERNAL_PORT}"
fi

echo "   GITEA_ALIAS_ADMIN_TO: ${GITEA_ALIAS_ADMIN_TO}"
echo "   GITEA_ADMIN_EMAIL: ${GITEA_ADMIN_EMAIL}"
echo "   FRONTEND: ${FRONTEND_HOST}:${FRONTEND_PORT}"
echo "   BACKEND: ${BACKEND_HOST}:${BACKEND_PORT}"
echo "   JUPYTERHUB: ${JUPYTERHUB_HOST}:${JUPYTERHUB_PORT}"
echo "   NIGHTINGALE: ${NIGHTINGALE_HOST}:${NIGHTINGALE_PORT}"
echo "   EXTERNAL: ${EXTERNAL_SCHEME}://${EXTERNAL_HOST}"

# TLS/HTTPS 配置切换
if [ "$ENABLE_TLS" = "true" ]; then
    echo "🔒 启用 TLS/HTTPS 模式..."
    
    # 检查 SSL 证书是否存在 (支持多种命名方式)
    SSL_CERT=""
    SSL_KEY=""
    
    # 优先查找通用名称 server.crt/server.key
    if [ -f /etc/nginx/ssl/server.crt ] && [ -f /etc/nginx/ssl/server.key ]; then
        SSL_CERT="/etc/nginx/ssl/server.crt"
        SSL_KEY="/etc/nginx/ssl/server.key"
    # 其次查找域名命名的证书 (如 192.168.18.131.crt)
    elif [ -n "$EXTERNAL_HOST_ONLY" ]; then
        CERT_PATTERN="/etc/nginx/ssl/${EXTERNAL_HOST_ONLY}.crt"
        KEY_PATTERN="/etc/nginx/ssl/${EXTERNAL_HOST_ONLY}.key"
        if [ -f "$CERT_PATTERN" ] && [ -f "$KEY_PATTERN" ]; then
            SSL_CERT="$CERT_PATTERN"
            SSL_KEY="$KEY_PATTERN"
            # 创建符号链接以统一使用 server.crt/server.key
            ln -sf "$SSL_CERT" /etc/nginx/ssl/server.crt
            ln -sf "$SSL_KEY" /etc/nginx/ssl/server.key
            echo "   已创建符号链接: server.crt -> ${EXTERNAL_HOST_ONLY}.crt"
        fi
    fi
    
    # 如果仍未找到，尝试查找任意 .crt/.key 文件
    if [ -z "$SSL_CERT" ]; then
        for cert_file in /etc/nginx/ssl/*.crt; do
            if [ -f "$cert_file" ]; then
                key_file="${cert_file%.crt}.key"
                if [ -f "$key_file" ]; then
                    SSL_CERT="$cert_file"
                    SSL_KEY="$key_file"
                    ln -sf "$SSL_CERT" /etc/nginx/ssl/server.crt
                    ln -sf "$SSL_KEY" /etc/nginx/ssl/server.key
                    echo "   已创建符号链接: server.crt -> $(basename $cert_file)"
                    break
                fi
            fi
        done
    fi
    
    if [ -z "$SSL_CERT" ] || [ -z "$SSL_KEY" ]; then
        echo "❌ 错误: SSL 证书文件不存在!"
        echo "   请确保以下文件已挂载到容器:"
        echo "   - /etc/nginx/ssl/server.crt (证书文件)"
        echo "   - /etc/nginx/ssl/server.key (私钥文件)"
        echo "   或使用域名命名的证书文件:"
        echo "   - /etc/nginx/ssl/${EXTERNAL_HOST_ONLY}.crt"
        echo "   - /etc/nginx/ssl/${EXTERNAL_HOST_ONLY}.key"
        echo ""
        echo "   提示: 使用以下命令生成自签名证书:"
        echo "   ./build.sh ssl-setup"
        echo "   或"
        echo "   ./scripts/generate-ssl.sh quick -d ${EXTERNAL_HOST_ONLY}"
        exit 1
    fi
    
    # 验证证书文件
    echo "🔍 验证 SSL 证书..."
    echo "   证书文件: $SSL_CERT"
    echo "   私钥文件: $SSL_KEY"
    if openssl x509 -in "$SSL_CERT" -noout -text > /dev/null 2>&1; then
        CERT_SUBJECT=$(openssl x509 -in "$SSL_CERT" -noout -subject 2>/dev/null | sed 's/subject=//')
        CERT_EXPIRE=$(openssl x509 -in "$SSL_CERT" -noout -enddate 2>/dev/null | sed 's/notAfter=//')
        echo "   ✅ 证书主题: ${CERT_SUBJECT}"
        echo "   📅 过期时间: ${CERT_EXPIRE}"
    else
        echo "   ⚠️ 证书验证失败，但继续启动..."
    fi
    
    # 启用 TLS 配置，禁用 HTTP 配置
    if [ -f /etc/nginx/conf.d/server-main-tls.conf ]; then
        echo "   启用 server-main-tls.conf..."
        # 禁用 HTTP 配置
        if [ -f /etc/nginx/conf.d/server-main.conf ]; then
            mv /etc/nginx/conf.d/server-main.conf /etc/nginx/conf.d/server-main.conf.disabled
        fi
    else
        echo "❌ 错误: server-main-tls.conf 不存在!"
        exit 1
    fi
    
    # 确保 EXTERNAL_SCHEME 是 https
    export EXTERNAL_SCHEME="https"
else
    echo "🌐 使用 HTTP 模式..."
    
    # 禁用 TLS 配置，启用 HTTP 配置
    if [ -f /etc/nginx/conf.d/server-main-tls.conf ]; then
        mv /etc/nginx/conf.d/server-main-tls.conf /etc/nginx/conf.d/server-main-tls.conf.disabled
    fi
    
    # 恢复 HTTP 配置（如果被禁用）
    if [ -f /etc/nginx/conf.d/server-main.conf.disabled ]; then
        mv /etc/nginx/conf.d/server-main.conf.disabled /etc/nginx/conf.d/server-main.conf
    fi
fi

# 替换配置文件中的环境变量
# 注意：只处理 {{VAR}} 格式，因为 ${VAR} 会被 nginx 解析为 nginx 变量导致错误
# 模板文件应使用 {{VAR}} 格式，build.sh 在构建时会替换它们
# 这里作为后备，确保在开发环境或直接启动时也能正常工作
find /etc/nginx/conf.d/ -name "*.conf" -type f -exec sed -i "s/{{GITEA_ALIAS_ADMIN_TO}}/${GITEA_ALIAS_ADMIN_TO}/g" {} \;
find /etc/nginx/conf.d/ -name "*.conf" -type f -exec sed -i "s/{{GITEA_ADMIN_EMAIL}}/${GITEA_ADMIN_EMAIL}/g" {} \;
find /etc/nginx/conf.d/ -name "*.conf" -type f -exec sed -i "s/{{FRONTEND_HOST}}/${FRONTEND_HOST}/g" {} \;
find /etc/nginx/conf.d/ -name "*.conf" -type f -exec sed -i "s/{{FRONTEND_PORT}}/${FRONTEND_PORT}/g" {} \;
find /etc/nginx/conf.d/ -name "*.conf" -type f -exec sed -i "s/{{BACKEND_HOST}}/${BACKEND_HOST}/g" {} \;
find /etc/nginx/conf.d/ -name "*.conf" -type f -exec sed -i "s/{{BACKEND_PORT}}/${BACKEND_PORT}/g" {} \;
find /etc/nginx/conf.d/ -name "*.conf" -type f -exec sed -i "s/{{JUPYTERHUB_HOST}}/${JUPYTERHUB_HOST}/g" {} \;
find /etc/nginx/conf.d/ -name "*.conf" -type f -exec sed -i "s/{{JUPYTERHUB_PORT}}/${JUPYTERHUB_PORT}/g" {} \;
find /etc/nginx/conf.d/ -name "*.conf" -type f -exec sed -i "s/{{NIGHTINGALE_HOST}}/${NIGHTINGALE_HOST}/g" {} \;
find /etc/nginx/conf.d/ -name "*.conf" -type f -exec sed -i "s/{{NIGHTINGALE_PORT}}/${NIGHTINGALE_PORT}/g" {} \;
find /etc/nginx/conf.d/ -name "*.conf" -type f -exec sed -i "s/{{EXTERNAL_SCHEME}}/${EXTERNAL_SCHEME}/g" {} \;
find /etc/nginx/conf.d/ -name "*.conf" -type f -exec sed -i "s/{{EXTERNAL_HOST}}/${EXTERNAL_HOST}/g" {} \;

echo "✅ 环境变量替换完成"

if [ "${DEBUG_MODE}" = "true" ]; then
    echo "🔧 启用调试模式 - 8001 调试服务可用"
    if [ -d "/usr/share/nginx/html/debug" ] && [ "$(ls -A /usr/share/nginx/html/debug)" ]; then
        echo "   ✅ 调试文件已加载"
    else
        echo "   ⚠️ 调试文件目录为空"
    fi
else
    echo "🚀 生产模式 - 禁用 8001 调试服务"
    # 通过移动/重命名调试server片段来禁用
    if [ -f /etc/nginx/conf.d/server-debug-jupyterhub.conf ]; then
        mv /etc/nginx/conf.d/server-debug-jupyterhub.conf /etc/nginx/conf.d/server-debug-jupyterhub.conf.disabled || true
    fi
    # 简易禁用提示页
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
