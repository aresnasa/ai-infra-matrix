#!/bin/bash
# AI Infra Matrix 项目根目录测试入口
# 自动切换到test目录并运行测试

set -e

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$SCRIPT_DIR/test"

if [ ! -d "$TEST_DIR" ]; then
    echo "❌ 测试目录不存在: $TEST_DIR"
    exit 1
fi

echo "🎯 AI Infra Matrix 测试工具"
echo "=========================="
echo "📁 测试目录: $TEST_DIR"
echo ""

# 切换到test目录并运行测试
cd "$TEST_DIR"
exec ./test.sh "$@"
