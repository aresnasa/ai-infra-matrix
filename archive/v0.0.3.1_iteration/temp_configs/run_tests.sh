#!/bin/bash
# 测试运行脚本
# 用于快速运行各种测试套件

set -e

echo "🧪 AI Infrastructure Matrix 测试套件"
echo "====================================="

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_section() {
    echo -e "\n${BLUE}📂 $1${NC}"
    echo "----------------------------------------"
}

run_test() {
    local test_file=$1
    local description=$2
    echo -e "🔍 运行: ${description}"
    if python "$test_file" > /dev/null 2>&1; then
        echo -e "  ${GREEN}✅ 通过${NC}"
        return 0
    else
        echo -e "  ${RED}❌ 失败${NC}"
        return 1
    fi
}

# 解析命令行参数
case "${1:-all}" in
    "iframe")
        print_section "iframe 功能测试"
        run_test "tests/iframe/quick_iframe_test.py" "快速iframe测试"
        run_test "tests/iframe/test_iframe_fix_verification.py" "iframe修复验证"
        ;;
    
    "jupyterhub")
        print_section "JupyterHub 服务测试"
        run_test "tests/jupyterhub/test_jupyterhub_wrapper_optimized.py" "wrapper优化测试"
        run_test "tests/jupyterhub/test_jupyterhub_consistency.py" "一致性测试"
        ;;
    
    "login")
        print_section "登录认证测试"
        run_test "tests/login/test_simple_auto_login.py" "简单自动登录"
        run_test "tests/login/test_quick_login.py" "快速登录测试"
        ;;
    
    "integration")
        print_section "集成测试"
        run_test "tests/integration/test_complete_flow.py" "完整流程测试"
        run_test "tests/integration/simple_wrapper_test.py" "wrapper集成测试"
        ;;
    
    "api")
        print_section "API 和重定向测试"
        run_test "tests/api/test_api_endpoints.py" "API端点测试"
        ;;
    
    "quick")
        print_section "快速测试套件"
        echo "🚀 运行关键测试..."
        run_test "tests/iframe/quick_iframe_test.py" "iframe功能"
        run_test "tests/integration/simple_wrapper_test.py" "wrapper集成"
        run_test "tests/api/test_api_endpoints.py" "API端点"
        ;;
    
    "all")
        print_section "完整测试套件"
        echo "🚀 运行所有测试..."
        
        # iframe测试
        echo -e "\n${YELLOW}iframe 测试:${NC}"
        run_test "tests/iframe/quick_iframe_test.py" "快速iframe测试"
        
        # JupyterHub测试
        echo -e "\n${YELLOW}JupyterHub 测试:${NC}"
        run_test "tests/jupyterhub/test_jupyterhub_consistency.py" "一致性测试"
        
        # API测试
        echo -e "\n${YELLOW}API 测试:${NC}"
        run_test "tests/api/test_api_endpoints.py" "API端点测试"
        
        # 集成测试
        echo -e "\n${YELLOW}集成 测试:${NC}"
        run_test "tests/integration/simple_wrapper_test.py" "wrapper集成测试"
        ;;
    
    "help"|"-h"|"--help")
        echo "用法: $0 [测试类型]"
        echo ""
        echo "测试类型:"
        echo "  iframe      - iframe功能测试"
        echo "  jupyterhub  - JupyterHub服务测试"
        echo "  login       - 登录认证测试"
        echo "  integration - 集成测试"
        echo "  api         - API和重定向测试"
        echo "  quick       - 快速测试套件"
        echo "  all         - 完整测试套件 (默认)"
        echo "  help        - 显示此帮助信息"
        echo ""
        echo "示例:"
        echo "  $0 quick     # 运行快速测试"
        echo "  $0 iframe    # 只运行iframe测试"
        echo "  $0 all       # 运行所有测试"
        exit 0
        ;;
    
    *)
        echo -e "${RED}❌ 未知的测试类型: $1${NC}"
        echo "使用 '$0 help' 查看可用选项"
        exit 1
        ;;
esac

echo -e "\n${GREEN}🎉 测试完成！${NC}"
