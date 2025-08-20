#!/bin/bash
# AI Infra Matrix 测试脚本
# 提供简化的测试入口

set -e

cd "$(dirname "$0")"

echo "🎯 AI Infra Matrix 测试工具"
echo "=========================="

# 检查Python依赖
if ! python3 -c "import requests" 2>/dev/null; then
    echo "📦 安装测试依赖..."
    pip3 install -r requirements.txt
fi

# 默认参数
TEST_TYPE="quick"
BASE_URL="http://localhost:8080"
VERBOSE=""

# 解析命令行参数
while [[ $# -gt 0 ]]; do
    case $1 in
        --test)
            TEST_TYPE="$2"
            shift 2
            ;;
        --url)
            BASE_URL="$2"
            shift 2
            ;;
        -v|--verbose)
            VERBOSE="-v"
            shift
            ;;
        -h|--help)
            echo "用法: $0 [选项]"
            echo ""
            echo "选项:"
            echo "  --test TYPE     测试类型 (quick|sso|health|all) [默认: quick]"
            echo "  --url URL       基础URL [默认: http://localhost:8080]"
            echo "  -v, --verbose   详细输出"
            echo "  -h, --help      显示帮助"
            echo ""
            echo "示例:"
            echo "  $0                    # 快速验证"
            echo "  $0 --test sso -v      # SSO详细测试"
            echo "  $0 --test health      # 健康检查"
            echo "  $0 --test all         # 完整测试"
            exit 0
            ;;
        *)
            echo "未知参数: $1"
            echo "使用 --help 查看帮助"
            exit 1
            ;;
    esac
done

echo "🔧 测试配置:"
echo "   类型: $TEST_TYPE"
echo "   URL:  $BASE_URL"
echo "   详细: ${VERBOSE:-否}"
echo ""

# 运行测试
python3 run_tests.py --test "$TEST_TYPE" --url "$BASE_URL" $VERBOSE
