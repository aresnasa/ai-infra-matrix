#!/bin/bash
# 等待测试服务就绪的脚本

set -e

# 设置代理环境变量
export HTTP_PROXY=http://127.0.0.1:7890
export HTTPS_PROXY=http://127.0.0.1:7890
export http_proxy=http://127.0.0.1:7890
export https_proxy=http://127.0.0.1:7890
export ALL_PROXY=socks5://127.0.0.1:7890
export NO_PROXY="localhost,127.0.0.1,::1,.local"

echo "等待 PostgreSQL 测试数据库就绪..."
until docker exec postgres-test pg_isready -U test_user -d ansible_generator_test > /dev/null 2>&1; do
  echo "PostgreSQL 还未就绪，等待 2 秒..."
  sleep 2
done
echo "✅ PostgreSQL 测试数据库已就绪"

echo "等待 Redis 测试实例就绪..."
until docker exec redis-test redis-cli ping > /dev/null 2>&1; do
  echo "Redis 还未就绪，等待 2 秒..."
  sleep 2
done
echo "✅ Redis 测试实例已就绪"

echo "等待后端测试服务就绪..."
until curl -f http://localhost:8083/health > /dev/null 2>&1; do
  echo "后端服务还未就绪，等待 3 秒..."
  sleep 3
done
echo "✅ 后端测试服务已就绪"

echo "🎉 所有测试服务已就绪，可以开始测试！"
