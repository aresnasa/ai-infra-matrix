#!/bin/bash

set -e

echo "🧂 AI基础设施矩阵 - SaltStack服务启动中..."
echo "📅 启动时间: $(date)"
echo "🏷️ 版本: ${APP_VERSION:-dev}"

# 创建必要的目录
mkdir -p /var/log/supervisor
mkdir -p /var/log/salt

# 设置权限
chown -R salt:salt /var/log/salt
chown -R salt:salt /var/cache/salt
chown -R salt:salt /var/run

# 生成Salt Master密钥
if [ ! -f /etc/salt/pki/master/master.pem ]; then
    echo "🔐 生成Salt Master密钥..."
    salt-key --gen-keys=master --gen-keys-dir=/etc/salt/pki/master/
    cp /etc/salt/pki/master/master.pem /etc/salt/pki/master/master.pub /etc/salt/pki/master/
    chown -R salt:salt /etc/salt/pki
fi

# 等待网络就绪
echo "🌐 等待网络服务就绪..."
sleep 5

echo "✅ SaltStack服务准备完成"
echo "🚀 支持功能:"
echo "   ✅ Salt Master (端口: 4505, 4506)"
echo "   ✅ Salt API (端口: 8000)"
echo "   ✅ AI-Infra-Matrix SSO集成"
echo "   ✅ 自动密钥管理"

# 启动supervisor
exec "$@"
