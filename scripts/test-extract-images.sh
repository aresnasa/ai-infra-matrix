#!/bin/bash
# 测试 extract_base_images 函数修复

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# 测试 Dockerfile
TEST_DOCKERFILE="$PROJECT_ROOT/src/test-containers/Dockerfile.ssh-key"

echo "==================================="
echo "测试镜像提取功能"
echo "==================================="
echo

echo "测试文件: $TEST_DOCKERFILE"
echo

echo "--- Dockerfile 内容 ---"
head -5 "$TEST_DOCKERFILE"
echo

echo "--- 提取的镜像列表 ---"
# 使用修复后的提取逻辑
grep -iE '^\s*FROM\s+' "$TEST_DOCKERFILE" | \
    sed -E 's/^\s*FROM\s+//I' | \
    sed -E 's/--platform=[^\s]+\s+//' | \
    awk '{print $1}' | \
    grep -v '^$' | \
    grep -v '^#' | \
    sort -u

echo
echo "==================================="
echo "✅ 测试完成"
echo "==================================="
