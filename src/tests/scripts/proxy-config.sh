#!/bin/bash
# 代理配置脚本
# 用于设置和管理代理环境变量

PROXY_HTTP="http://127.0.0.1:7890"
PROXY_HTTPS="http://127.0.0.1:7890"
PROXY_SOCKS="socks5://127.0.0.1:7890"
NO_PROXY_LIST="localhost,127.0.0.1,::1,.local"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 设置代理
set_proxy() {
    export HTTP_PROXY="$PROXY_HTTP"
    export HTTPS_PROXY="$PROXY_HTTPS"
    export http_proxy="$PROXY_HTTP"
    export https_proxy="$PROXY_HTTPS"
    export ALL_PROXY="$PROXY_SOCKS"
    export NO_PROXY="$NO_PROXY_LIST"
    
    echo -e "${GREEN}✅ Proxy settings applied:${NC}"
    echo "  HTTP_PROXY=$HTTP_PROXY"
    echo "  HTTPS_PROXY=$HTTPS_PROXY"
    echo "  ALL_PROXY=$ALL_PROXY"
    echo "  NO_PROXY=$NO_PROXY"
}

# 清除代理
unset_proxy() {
    unset HTTP_PROXY
    unset HTTPS_PROXY
    unset http_proxy
    unset https_proxy
    unset ALL_PROXY
    unset NO_PROXY
    
    echo -e "${GREEN}✅ Proxy settings cleared${NC}"
}

# 显示当前代理设置
show_proxy() {
    echo -e "${BLUE}🔍 Current proxy configuration:${NC}"
    echo "  HTTP_PROXY=${HTTP_PROXY:-Not set}"
    echo "  HTTPS_PROXY=${HTTPS_PROXY:-Not set}"
    echo "  http_proxy=${http_proxy:-Not set}"
    echo "  https_proxy=${https_proxy:-Not set}"
    echo "  ALL_PROXY=${ALL_PROXY:-Not set}"
    echo "  NO_PROXY=${NO_PROXY:-Not set}"
}

# 测试代理连接
test_proxy() {
    echo -e "${BLUE}🔍 Testing proxy connectivity...${NC}"
    
    # 测试HTTP代理
    echo "Testing HTTP proxy..."
    if HTTP_PROXY="$PROXY_HTTP" curl -I --connect-timeout 5 http://www.google.com >/dev/null 2>&1; then
        echo -e "${GREEN}✅ HTTP proxy working${NC}"
    else
        echo -e "${RED}❌ HTTP proxy failed${NC}"
    fi
    
    # 测试HTTPS代理
    echo "Testing HTTPS proxy..."
    if HTTPS_PROXY="$PROXY_HTTPS" curl -I --connect-timeout 5 https://www.google.com >/dev/null 2>&1; then
        echo -e "${GREEN}✅ HTTPS proxy working${NC}"
    else
        echo -e "${RED}❌ HTTPS proxy failed${NC}"
    fi
    
    # 测试本地代理服务
    echo "Testing local proxy service..."
    if curl -I --connect-timeout 3 http://127.0.0.1:7890 >/dev/null 2>&1; then
        echo -e "${GREEN}✅ Local proxy service (7890) is running${NC}"
    else
        echo -e "${YELLOW}⚠️  Local proxy service (7890) not accessible${NC}"
        echo "   Make sure your proxy client (Clash/v2ray/etc.) is running"
    fi
}

# 显示帮助信息
show_help() {
    echo "Proxy Configuration Script"
    echo "=========================="
    echo ""
    echo "Usage: $0 [COMMAND]"
    echo ""
    echo "Commands:"
    echo "  set     - Set proxy environment variables"
    echo "  unset   - Clear proxy environment variables"
    echo "  show    - Show current proxy settings"
    echo "  test    - Test proxy connectivity"
    echo "  help    - Show this help message"
    echo ""
    echo "Proxy Configuration:"
    echo "  HTTP/HTTPS Proxy: $PROXY_HTTP"
    echo "  SOCKS Proxy: $PROXY_SOCKS"
    echo "  No Proxy: $NO_PROXY_LIST"
    echo ""
    echo "Examples:"
    echo "  $0 set          # Set proxy for current session"
    echo "  $0 test         # Test proxy connectivity"
    echo "  source $0 set   # Set proxy and export to current shell"
}

# 导出函数以便在其他脚本中使用
export_proxy_functions() {
    export -f set_proxy
    export -f unset_proxy
    export -f show_proxy
    export -f test_proxy
}

# 主函数
main() {
    case "${1:-help}" in
        "set")
            set_proxy
            ;;
        "unset")
            unset_proxy
            ;;
        "show")
            show_proxy
            ;;
        "test")
            test_proxy
            ;;
        "export")
            export_proxy_functions
            echo -e "${GREEN}✅ Proxy functions exported${NC}"
            ;;
        "help"|*)
            show_help
            ;;
    esac
}

# 如果脚本被直接执行，运行主函数
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi
