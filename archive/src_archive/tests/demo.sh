#!/bin/bash
# 演示脚本：展示Ansible Playbook Generator的自动化测试功能

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${CYAN}🎯 Ansible Playbook Generator - 自动化测试演示${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo -e "${BLUE}本演示将展示如何使用Docker Compose进行完全自动化的测试${NC}"
echo ""
echo -e "${GREEN}🚀 可用的测试命令：${NC}"
echo ""
echo -e "${YELLOW}1. make auto-test${NC}     - 运行完整的自动化测试套件（包含性能和安全测试）"
echo -e "${YELLOW}2. make quick-test${NC}    - 运行快速测试（仅核心功能）"
echo -e "${YELLOW}3. make test-all${NC}      - 运行所有手动测试"
echo -e "${YELLOW}4. make health-check${NC}  - 检查所有服务的健康状态"
echo -e "${YELLOW}5. make start-prod${NC}    - 启动生产环境"
echo ""
echo -e "${PURPLE}✨ 特性：${NC}"
echo "  🐳 完全基于 Docker Compose"
echo "  🔄 自动化环境管理"
echo "  🏥 健康检查监控"
echo "  📊 详细的测试报告"
echo "  🌐 端到端测试"
echo "  🛡️  安全性检查"
echo "  ⚡ 性能测试"
echo ""

# 检查当前目录
if [ ! -f "Makefile" ]; then
    echo -e "${RED}❌ 请在 tests 目录中运行此演示脚本${NC}"
    exit 1
fi

echo -e "${BLUE}📁 当前工作目录：$(pwd)${NC}"
echo -e "${BLUE}🐳 Docker 版本：$(docker --version)${NC}"
echo -e "${BLUE}🔧 Docker Compose 版本：$(docker-compose --version)${NC}"
echo ""

echo -e "${GREEN}❓ 选择要运行的测试类型：${NC}"
echo ""
echo "1) 快速演示测试 (推荐)"
echo "2) 完整自动化测试"
echo "3) 仅健康检查"
echo "4) 启动生产环境"
echo "5) 显示帮助信息"
echo ""
read -p "请输入选择 (1-5): " choice

case $choice in
    1)
        echo ""
        echo -e "${GREEN}🚀 运行快速演示测试...${NC}"
        echo ""
        make quick-test
        ;;
    2)
        echo ""
        echo -e "${GREEN}🚀 运行完整自动化测试...${NC}"
        echo ""
        make auto-test
        ;;
    3)
        echo ""
        echo -e "${GREEN}🏥 运行健康检查...${NC}"
        echo ""
        make start-test-env
        make health-check
        echo ""
        read -p "是否停止测试环境？ (y/N): " stop_env
        if [[ $stop_env =~ ^[Yy]$ ]]; then
            make stop-test-env
        fi
        ;;
    4)
        echo ""
        echo -e "${GREEN}🌐 启动生产环境...${NC}"
        echo ""
        make start-prod
        echo ""
        echo -e "${CYAN}🌐 生产环境已启动！${NC}"
        echo -e "${CYAN}访问地址：${NC}"
        echo "  Frontend: http://localhost:3001"
        echo "  Backend API: http://localhost:8082"
        echo "  Swagger文档: http://localhost:8082/swagger/index.html"
        echo ""
        read -p "按回车键停止生产环境..." 
        make stop-prod
        ;;
    5)
        echo ""
        make help
        ;;
    *)
        echo -e "${RED}❌ 无效选择${NC}"
        exit 1
        ;;
esac

echo ""
echo -e "${GREEN}🎉 演示完成！${NC}"
echo ""
echo -e "${BLUE}💡 提示：${NC}"
echo "  • 查看更多命令：make help"
echo "  • 查看服务状态：make status"
echo "  • 查看日志：make logs"
echo "  • 清理环境：make clean"
echo ""
echo -e "${CYAN}谢谢使用 Ansible Playbook Generator！${NC}"
