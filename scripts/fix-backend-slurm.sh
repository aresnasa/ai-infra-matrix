#!/bin/bash

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Backend SLURM 客户端自动修复工具${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# 1. 检查容器状态
echo -e "${YELLOW}步骤 1: 检查容器状态...${NC}"
if docker ps --format '{{.Names}}' | grep -q "^ai-infra-backend$"; then
    echo -e "${GREEN}✓ ai-infra-backend 容器正在运行${NC}"
else
    echo -e "${RED}✗ ai-infra-backend 容器未运行${NC}"
    echo "请先启动容器: docker-compose up -d backend"
    exit 1
fi

if docker ps --format '{{.Names}}' | grep -q "^ai-infra-apphub$"; then
    echo -e "${GREEN}✓ ai-infra-apphub 容器正在运行${NC}"
else
    echo -e "${RED}✗ ai-infra-apphub 容器未运行${NC}"
    echo "请先启动容器: docker-compose up -d apphub"
    exit 1
fi

# 2. 检查 SLURM 客户端
echo ""
echo -e "${YELLOW}步骤 2: 检查 SLURM 客户端安装...${NC}"
if docker exec ai-infra-backend sh -c 'command -v sinfo' >/dev/null 2>&1; then
    echo -e "${GREEN}✓ SLURM 客户端已安装${NC}"
    VERSION=$(docker exec ai-infra-backend sh -c 'sinfo --version 2>&1' | head -1)
    echo "  版本: $VERSION"
    
    echo ""
    echo -e "${YELLOW}测试 SLURM 连接...${NC}"
    if docker exec ai-infra-backend sh -c 'sinfo 2>&1' | grep -q "Unable to contact"; then
        echo -e "${YELLOW}⚠ SLURM 客户端已安装但无法连接到 slurm-master${NC}"
        echo "  请检查:"
        echo "  1. ai-infra-slurm-master 容器是否运行"
        echo "  2. /etc/slurm/slurm.conf 配置是否正确"
        echo "  3. 网络连接是否正常"
    else
        echo -e "${GREEN}✓ SLURM 客户端工作正常${NC}"
        docker exec ai-infra-backend sh -c 'sinfo'
    fi
    exit 0
else
    echo -e "${RED}✗ SLURM 客户端未安装${NC}"
fi

# 3. 提供安装选项
echo ""
echo -e "${YELLOW}步骤 3: 提供解决方案...${NC}"
echo ""
echo "有两种方式安装 SLURM 客户端:"
echo ""
echo -e "${BLUE}方案 1: 从 AppHub APK 仓库安装（推荐）${NC}"
echo "  1. 构建 SLURM APK 包:"
echo "     cd src/apphub && ./build-slurm-apk.sh"
echo ""
echo "  2. 重新构建并启动 backend 容器:"
echo "     docker-compose build backend"
echo "     docker-compose up -d backend"
echo ""
echo -e "${BLUE}方案 2: 检查现有 AppHub APK 仓库${NC}"
echo "  docker exec ai-infra-apphub ls -lh /usr/share/nginx/html/apks/alpine/"
echo ""

# 4. 询问是否自动执行
echo -n "是否自动执行方案 1？(y/N): "
read -r response
if [[ "$response" =~ ^[Yy]$ ]]; then
    echo ""
    echo -e "${YELLOW}开始自动修复...${NC}"
    
    # 检查构建脚本
    if [ ! -f "./src/apphub/build-slurm-apk.sh" ]; then
        echo -e "${RED}✗ 构建脚本不存在: src/apphub/build-slurm-apk.sh${NC}"
        exit 1
    fi
    
    # 执行构建
    echo -e "${YELLOW}步骤 4: 构建 SLURM APK 包...${NC}"
    cd src/apphub
    bash ./build-slurm-apk.sh
    BUILD_EXIT_CODE=$?
    cd ../..
    
    if [ $BUILD_EXIT_CODE -ne 0 ]; then
        echo -e "${RED}✗ APK 包构建失败，退出码: $BUILD_EXIT_CODE${NC}"
        exit $BUILD_EXIT_CODE
    fi
    
    # 重新构建容器
    echo ""
    echo -e "${YELLOW}步骤 5: 重新构建 backend 容器...${NC}"
    docker-compose build backend
    
    # 重启容器
    echo ""
    echo -e "${YELLOW}步骤 6: 重启 backend 容器...${NC}"
    docker-compose up -d backend
    
    # 等待容器就绪
    echo ""
    echo -e "${YELLOW}等待容器就绪...${NC}"
    sleep 5
    
    # 验证安装
    echo ""
    echo -e "${YELLOW}步骤 7: 验证 SLURM 客户端安装...${NC}"
    if docker exec ai-infra-backend sh -c 'command -v sinfo' >/dev/null 2>&1; then
        echo -e "${GREEN}✓ SLURM 客户端安装成功！${NC}"
        VERSION=$(docker exec ai-infra-backend sh -c 'sinfo --version 2>&1' | head -1)
        echo "  版本: $VERSION"
        
        # 测试连接
        echo ""
        echo -e "${YELLOW}步骤 8: 测试 SLURM 连接...${NC}"
        docker exec ai-infra-backend sh -c 'sinfo'
        
        echo ""
        echo -e "${GREEN}========================================${NC}"
        echo -e "${GREEN}修复完成！${NC}"
        echo -e "${GREEN}========================================${NC}"
    else
        echo -e "${RED}✗ SLURM 客户端安装失败${NC}"
        echo "请检查容器日志: docker logs ai-infra-backend"
        exit 1
    fi
else
    echo ""
    echo -e "${YELLOW}请手动执行上述命令进行修复${NC}"
fi
