#!/bin/bash
# Mock 环境连接测试脚本

echo "=== Mock 环境连接测试 ==="

# 测试 PostgreSQL
echo "测试 PostgreSQL 连接..."
if docker exec ai-infra-mock-postgres psql -U test_user -d test_db -c "SELECT version();" >/dev/null 2>&1; then
    echo "✓ PostgreSQL 连接正常"
else
    echo "✗ PostgreSQL 连接失败"
fi

# 测试 Redis
echo "测试 Redis 连接..."
if docker exec ai-infra-mock-redis redis-cli -a test_redis_pass ping >/dev/null 2>&1; then
    echo "✓ Redis 连接正常"
else
    echo "✗ Redis 连接失败"
fi

echo "=== 测试完成 ==="
