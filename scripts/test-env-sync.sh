#!/bin/bash
# 测试环境变量同步功能

echo "======================================"
echo "测试环境变量同步功能"
echo "======================================"
echo ""

echo "当前 .env 中的 Salt 配置："
grep -E "SALT_API_PORT=|SALTSTACK_MASTER_URL=" .env | while read line; do
    echo "  $line"
done

echo ""
echo ".env.example 中的推荐配置："
grep -E "SALT_API_PORT=|SALTSTACK_MASTER_URL=" .env.example | while read line; do
    echo "  $line"
done

echo ""
echo "======================================"
echo "差异："
echo "======================================"
echo "  .env:         SALT_API_PORT=8002"
echo "  .env.example: SALT_API_PORT=8000 ✓ (推荐)"
echo ""
echo "  .env:         SALTSTACK_MASTER_URL=http://saltstack:8002"
echo "  .env.example: SALTSTACK_MASTER_URL=http://saltstack:8000 ✓ (推荐)"
echo ""

echo "======================================"
echo "同步选项："
echo "======================================"
echo "运行构建时会看到以下选项："
echo ""
echo "  [y] - 保留当前 .env 值（8002）"
echo "  [u] - 更新为推荐值（8000）⭐ 推荐"
echo "  [d] - 查看详细差异"
echo "  [n] - 跳过同步"
echo ""
echo "推荐操作："
echo "  ./build.sh build-all"
echo "  # 当提示时选择 [u] 选项"
echo ""
