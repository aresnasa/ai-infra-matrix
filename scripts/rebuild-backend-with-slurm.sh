#!/bin/bash
# ====================================================================
# 重新构建 backend 并验证 SLURM 客户端安装
# ====================================================================

set -e

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}======================================================================"
echo "重新构建 backend 并安装 SLURM 客户端"
echo -e "======================================================================${NC}"

# 步骤1: 停止 backend
echo -e "\n${YELLOW}步骤 1/5: 停止 backend 服务...${NC}"
docker-compose stop backend
echo -e "${GREEN}✓ backend 已停止${NC}"

# 步骤2: 删除旧容器
echo -e "\n${YELLOW}步骤 2/5: 删除旧容器...${NC}"
docker-compose rm -f backend
echo -e "${GREEN}✓ 旧容器已删除${NC}"

# 步骤3: 重新构建 backend（不使用缓存）
echo -e "\n${YELLOW}步骤 3/5: 重新构建 backend 镜像...${NC}"
echo "   这可能需要几分钟..."
docker-compose build --no-cache backend

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ backend 镜像构建成功${NC}"
else
    echo -e "${RED}✗ 镜像构建失败${NC}"
    exit 1
fi

# 步骤4: 启动 backend
echo -e "\n${YELLOW}步骤 4/5: 启动 backend 服务...${NC}"
docker-compose up -d backend

# 等待服务启动
echo "   等待服务启动（10秒）..."
sleep 10

# 步骤5: 验证 SLURM 客户端安装
echo -e "\n${YELLOW}步骤 5/5: 验证 SLURM 客户端安装...${NC}"

# 检查 sinfo 命令
if docker exec ai-infra-backend which sinfo >/dev/null 2>&1; then
    echo -e "${GREEN}✓ sinfo 命令已安装${NC}"
    echo "   路径:"
    docker exec ai-infra-backend which sinfo
    
    echo ""
    echo "   版本:"
    docker exec ai-infra-backend sinfo --version 2>&1 | head -1 || echo "   (版本信息不可用)"
    
    echo ""
    echo "   测试执行:"
    if docker exec ai-infra-backend sinfo 2>&1 | head -5; then
        echo -e "${GREEN}   ✓ sinfo 命令可以正常执行${NC}"
    else
        echo -e "${YELLOW}   ⚠ sinfo 执行但可能需要配置 SLURM 连接${NC}"
    fi
else
    echo -e "${RED}✗ sinfo 命令未安装${NC}"
    echo ""
    echo "已安装的 SLURM 相关包:"
    docker exec ai-infra-backend apk info | grep -i slurm || echo "  (无)"
    
    echo ""
    echo "可能的原因:"
    echo "  1. AppHub 仓库不可用"
    echo "  2. Alpine edge 仓库连接失败"
    echo "  3. SLURM 包名称不匹配"
    
    echo ""
    echo "解决方案:"
    echo "  1. 检查构建日志: docker-compose logs backend | grep -i slurm"
    echo "  2. 手动安装: docker exec ai-infra-backend apk add slurm --repository=http://dl-cdn.alpinelinux.org/alpine/edge/community"
    echo "  3. 构建自定义 APK: cd src/apphub && ./build-slurm-apk.sh"
    exit 1
fi

echo ""
echo -e "${GREEN}======================================================================"
echo "✓✓✓ backend 重新构建完成，SLURM 客户端已安装"
echo -e "======================================================================${NC}"

echo ""
echo "测试命令:"
echo "  docker exec ai-infra-backend sinfo"
echo "  docker exec ai-infra-backend squeue"
echo "  docker exec ai-infra-backend scontrol show config"
echo ""
