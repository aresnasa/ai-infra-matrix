#!/bin/bash

# AI异步架构测试运行脚本
set -e

echo "🚀 启动AI异步架构测试..."

# 等待服务启动
echo "⏳ 等待服务启动..."
sleep 30

# 设置测试环境变量
export BACKEND_URL="http://backend:8080"
export REDIS_URL="redis://redis:6379"
export DB_URL="postgres://user:password@postgres:5432/ansible_db?sslmode=disable"

# 运行基础连接测试
echo "🔍 测试基础服务连接..."

# 测试后端健康检查
echo "检查后端服务..."
curl -f $BACKEND_URL/health || {
    echo "❌ 后端服务不可用"
    exit 1
}

# 测试Redis连接
echo "检查Redis连接..."
redis-cli -u $REDIS_URL ping || {
    echo "❌ Redis服务不可用"
    exit 1
}

# 测试数据库连接
echo "检查数据库连接..."
PGPASSWORD=password psql -h postgres -U user -d ansible_db -c "SELECT 1;" || {
    echo "❌ 数据库服务不可用"
    exit 1
}

echo "✅ 基础服务连接正常"

# 运行AI异步功能测试
echo "🤖 运行AI异步功能测试..."

# 测试消息队列
echo "测试消息队列功能..."
./tests/test-message-queue.sh

# 测试缓存服务
echo "测试缓存服务功能..."
./tests/test-cache-service.sh

# 测试AI网关
echo "测试AI网关功能..."
./tests/test-ai-gateway.sh

# 测试异步API
echo "测试异步API功能..."
./tests/test-async-api.sh

# 测试集群操作
echo "测试集群操作功能..."
./tests/test-cluster-operations.sh

# 运行性能测试
echo "🚄 运行性能测试..."
./tests/test-performance.sh

# 运行压力测试
echo "💪 运行压力测试..."
./tests/test-stress.sh

echo "🎉 所有测试完成！"

# 生成测试报告
echo "📊 生成测试报告..."
./tests/generate-report.sh

echo "✅ AI异步架构测试全部通过！"
