#!/bin/bash
# 重新构建测试容器
# 用于更新 Dockerfile 后快速重建和重启容器

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

echo "🔧 重新构建测试容器..."
echo ""

# 询问要重建哪些容器
if [ -z "$1" ]; then
    echo "用法: $0 [ubuntu|rocky|all]"
    echo ""
    echo "示例:"
    echo "  $0 rocky    # 只重建 Rocky Linux 容器"
    echo "  $0 ubuntu   # 只重建 Ubuntu 容器"
    echo "  $0 all      # 重建所有测试容器"
    echo ""
    read -p "请选择要重建的容器 [ubuntu/rocky/all]: " REBUILD_TYPE
else
    REBUILD_TYPE="$1"
fi

case "$REBUILD_TYPE" in
    rocky)
        echo "📦 重建 Rocky Linux 测试容器..."
        docker-compose build test-rocky
        echo ""
        echo "🔄 重启 Rocky Linux 容器..."
        docker-compose up -d test-rocky01 test-rocky02 test-rocky03
        echo ""
        echo "✅ Rocky Linux 容器已重建并重启"
        echo ""
        echo "验证命令:"
        echo "  docker exec test-rocky01 ps aux"
        echo "  docker exec test-rocky01 ss -tuln"
        ;;
    ubuntu)
        echo "📦 重建 Ubuntu 测试容器..."
        docker-compose build test-ssh
        echo ""
        echo "🔄 重启 Ubuntu 容器..."
        docker-compose up -d test-ssh01 test-ssh02 test-ssh03
        echo ""
        echo "✅ Ubuntu 容器已重建并重启"
        ;;
    all)
        echo "📦 重建所有测试容器..."
        docker-compose build test-ssh test-rocky
        echo ""
        echo "🔄 重启所有测试容器..."
        docker-compose up -d test-ssh01 test-ssh02 test-ssh03 test-rocky01 test-rocky02 test-rocky03
        echo ""
        echo "✅ 所有测试容器已重建并重启"
        ;;
    *)
        echo "❌ 无效的选项: $REBUILD_TYPE"
        echo "请选择: ubuntu, rocky, 或 all"
        exit 1
        ;;
esac

echo ""
echo "📊 容器状态:"
docker-compose ps | grep -E "test-ssh|test-rocky" || echo "没有运行的测试容器"

echo ""
echo "✨ 完成！"
