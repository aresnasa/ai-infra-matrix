#!/bin/bash

# 启动SSH测试容器脚本
# 用于SaltStack客户端安装测试

set -e

echo "🚀 启动SSH测试容器..."

# 确保网络存在
echo "📡 检查Docker网络..."
if ! docker network ls | grep -q "ai-infra-network"; then
    echo "创建ai-infra-network网络..."
    docker network create ai-infra-network
else
    echo "ai-infra-network网络已存在"
fi

# 构建并启动测试容器
echo "🏗️ 构建并启动SSH测试容器..."
docker-compose -f docker-compose.test.yml up -d --build

echo "⏰ 等待容器启动..."
sleep 10

# 检查容器状态
echo "✅ 检查容器状态..."
docker-compose -f docker-compose.test.yml ps

# 测试SSH连接
echo "🔍 测试SSH连接..."
for port in 2201 2202 2203; do
    echo "测试端口 $port..."
    if timeout 5 bash -c "</dev/tcp/localhost/$port" &>/dev/null; then
        echo "✅ 端口 $port 可访问"
        # 测试SSH认证
        if sshpass -p testpass123 ssh -o StrictHostKeyChecking=no -o ConnectTimeout=5 testuser@localhost -p $port 'echo "SSH连接成功"' 2>/dev/null; then
            echo "✅ SSH认证成功 (端口 $port)"
        else
            echo "❌ SSH认证失败 (端口 $port)"
        fi
    else
        echo "❌ 端口 $port 不可访问"
    fi
done

echo ""
echo "🎉 SSH测试容器启动完成!"
echo ""
echo "📋 容器信息:"
echo "  test-ssh01: localhost:2201 (testuser/testpass123)"
echo "  test-ssh02: localhost:2202 (testuser/testpass123)"
echo "  test-ssh03: localhost:2203 (testuser/testpass123)"
echo ""
echo "🔧 可以通过以下命令测试SSH连接:"
echo "  ssh testuser@localhost -p 2201"
echo "  ssh testuser@localhost -p 2202"
echo "  ssh testuser@localhost -p 2203"
echo ""
echo "🛠️ SaltStack客户端安装API端点:"
echo "  POST http://localhost:8080/api/saltstack/install"
echo "  GET  http://localhost:8080/api/saltstack/install"
echo "  GET  http://localhost:8080/api/saltstack/test-hosts"
echo ""
