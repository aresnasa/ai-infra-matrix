#!/bin/bash

# AI Infrastructure Matrix - 快速构建和测试脚本
# 用途：一键构建所有服务并启动测试环境

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=========================================="
echo "AI Infrastructure Matrix"
echo "快速构建和测试部署"
echo "=========================================="
echo

# 检查 Docker
if ! command -v docker &> /dev/null; then
    echo "❌ 错误: 未找到 Docker"
    echo "请先安装 Docker: https://docs.docker.com/get-docker/"
    exit 1
fi

# 检查 Docker Compose
if ! docker compose version &> /dev/null; then
    echo "❌ 错误: 未找到 Docker Compose"
    echo "请确保使用 Docker Compose v2"
    exit 1
fi

# 步骤 1: 构建所有服务
echo "=========================================="
echo "步骤 1: 构建所有服务镜像"
echo "=========================================="
if ./build.sh build-all --force; then
    echo "✅ 所有服务构建完成"
else
    echo "❌ 构建失败"
    exit 1
fi
echo

# 步骤 2: 确保网络存在
echo "=========================================="
echo "步骤 2: 创建 Docker 网络"
echo "=========================================="
if docker network create ai-infra-network 2>/dev/null; then
    echo "✅ 网络创建成功: ai-infra-network"
else
    echo "ℹ️  网络已存在: ai-infra-network"
fi
echo

# 步骤 3: 启动测试容器
echo "=========================================="
echo "步骤 3: 启动测试容器"
echo "=========================================="
if docker compose -f docker-compose.test.yml up -d; then
    echo "✅ 测试容器启动成功"
else
    echo "❌ 测试容器启动失败"
    exit 1
fi
echo

# 步骤 4: 等待容器就绪
echo "=========================================="
echo "步骤 4: 等待容器就绪"
echo "=========================================="
echo "等待 10 秒..."
sleep 10

# 步骤 5: 验证部署
echo "=========================================="
echo "步骤 5: 验证部署"
echo "=========================================="

echo
echo "检查容器状态..."
docker ps | grep test-ssh | awk '{print "  ✓ " $NF " - 运行正常"}'

echo
echo "检查 systemd 状态..."
for container in test-ssh01 test-ssh02 test-ssh03; do
    state=$(docker exec $container systemctl is-system-running 2>/dev/null || echo "未知")
    if [[ "$state" == "running" ]] || [[ "$state" == "degraded" ]]; then
        echo "  ✓ $container: systemd 状态 = $state"
    else
        echo "  ⚠ $container: systemd 状态 = $state"
    fi
done

echo
echo "检查 SSH 服务..."
for container in test-ssh01 test-ssh02 test-ssh03; do
    if docker exec $container systemctl is-active ssh >/dev/null 2>&1; then
        echo "  ✓ $container: SSH 服务正常"
    else
        echo "  ⚠ $container: SSH 服务异常"
    fi
done

echo
echo "=========================================="
echo "✅ 部署完成！"
echo "=========================================="
echo
echo "测试容器访问信息:"
echo "  test-ssh01: ssh -p 2201 testuser@localhost"
echo "  test-ssh02: ssh -p 2202 testuser@localhost"
echo "  test-ssh03: ssh -p 2203 testuser@localhost"
echo "  密码: testpass123"
echo
echo "查看容器日志:"
echo "  docker logs test-ssh01"
echo "  docker logs test-ssh02"
echo "  docker logs test-ssh03"
echo
echo "进入容器:"
echo "  docker exec -it test-ssh01 bash"
echo "  docker exec -it test-ssh02 bash"
echo "  docker exec -it test-ssh03 bash"
echo
echo "停止测试环境:"
echo "  docker compose -f docker-compose.test.yml down"
echo
