#!/bin/bash

set -e

echo "🧂 AI基础设施矩阵 - SaltStack服务启动中..."
echo "📅 启动时间: $(date)"
echo "🏷️ 版本: ${APP_VERSION:-dev}"

# 创建必要的目录
mkdir -p /var/log/salt
mkdir -p /var/cache/salt/master
mkdir -p /var/cache/salt/minion
mkdir -p /var/run/salt
mkdir -p /etc/salt/pki/master
mkdir -p /etc/salt/pki/minion

# 生成Salt Master密钥
if [ ! -f /etc/salt/pki/master/master.pem ]; then
    echo "🔐 生成Salt Master密钥..."
    salt-key --gen-keys=master --gen-keys-dir=/etc/salt/pki/master/
fi

# 验证Salt配置文件
echo "🔧 验证Salt配置文件..."
if [ ! -r /etc/salt/master.d/master.conf ]; then
    echo "❌ Salt Master配置文件不可读"
    exit 1
fi

if [ ! -r /etc/salt/minion.d/minion.conf ]; then
    echo "❌ Salt Minion配置文件不可读"
    exit 1
fi

echo "✅ Salt配置文件验证通过"

echo "✅ SaltStack服务准备完成"
echo "🚀 支持功能:"
echo "   ✅ Salt Master (端口: 4505, 4506)"
echo "   ✅ Salt API (端口: 8002)"
echo "   ✅ AI-Infra-Matrix SSO集成"
echo "   ✅ 自动密钥管理"

# 配置 Salt API 文件认证凭据（file eauth）
SALT_API_USER=${SALT_API_USERNAME:-saltapi}
SALT_API_PASS=${SALT_API_PASSWORD:-}
EAUTH_FILE=/etc/salt/master.d/eauth.txt
if [ -n "$SALT_API_PASS" ]; then
    echo "🔐 生成Salt API文件认证凭据 ($EAUTH_FILE)"
    mkdir -p "$(dirname "$EAUTH_FILE")"
    echo "${SALT_API_USER}:${SALT_API_PASS}" > "$EAUTH_FILE"
    chmod 600 "$EAUTH_FILE" || true
else
    echo "⚠️ 未设置 SALT_API_PASSWORD，Salt API 将允许匿名访问公开端点，但需要认证的runner/wheel可能失败"
fi

# 后台启动Salt API（如果需要）
if [ "$1" = "salt-master" ]; then
    echo "🌐 启动Salt API服务..."
    # 启动Salt API（端口由配置文件控制，默认8002）
    salt-api --daemon || echo "⚠️ Salt API启动失败，继续启动Master"
    
        if [ "${START_LOCAL_MINION:-true}" = "true" ]; then
            echo "🔧 启动Salt Minion (本地测试)..."
            # 确保日志目录和文件可写
            mkdir -p /var/log/salt
            touch /var/log/salt/minion
            chmod 644 /var/log/salt/minion || true
            salt-minion --daemon || echo "⚠️ Salt Minion启动失败，继续启动Master"
        else
            echo "ℹ️ 已禁用本地测试Minion启动 (START_LOCAL_MINION=false)"
        fi
    
    # 等待服务就绪
    sleep 3
    
    echo "🎯 启动Salt Master..."
    exec salt-master --log-level=info
else
    # 如果不是启动master，直接执行命令
    exec "$@"
fi
