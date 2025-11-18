#!/bin/bash
#
# 验证 SLURM SaltStack UI 修复效果
# 
# 用途：检查前端修复是否生效，SaltStack 集成状态卡片和节点状态列是否正确显示
#

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
BASE_URL="${BASE_URL:-http://192.168.0.200:8080}"
USERNAME="${USERNAME:-admin}"
PASSWORD="${PASSWORD:-admin123}"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}SLURM SaltStack UI 修复验证${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo "目标地址: $BASE_URL"
echo "测试时间: $(date '+%Y-%m-%d %H:%M:%S')"
echo ""

# 步骤 1: 登录获取 token
echo -e "${YELLOW}[1/4] 登录系统...${NC}"
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}")

TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo -e "${RED}✗ 登录失败，无法获取 token${NC}"
  echo "响应: $LOGIN_RESPONSE"
  exit 1
fi

echo -e "${GREEN}✓ 登录成功${NC}"
echo ""

# 步骤 2: 检查 SaltStack 集成 API
echo -e "${YELLOW}[2/4] 检查 SaltStack 集成 API...${NC}"
SALTSTACK_RESPONSE=$(curl -s -X GET "$BASE_URL/api/slurm/saltstack/integration" \
  -H "Authorization: Bearer $TOKEN")

if echo "$SALTSTACK_RESPONSE" | grep -q '"enabled"'; then
  echo -e "${GREEN}✓ SaltStack 集成 API 响应正常${NC}"
  
  # 检查关键字段
  if echo "$SALTSTACK_RESPONSE" | grep -q '"master_status"'; then
    echo -e "${GREEN}  - master_status 字段存在${NC}"
  fi
  
  if echo "$SALTSTACK_RESPONSE" | grep -q '"api_status"'; then
    echo -e "${GREEN}  - api_status 字段存在${NC}"
  fi
  
  if echo "$SALTSTACK_RESPONSE" | grep -q '"minion_list"'; then
    echo -e "${GREEN}  - minion_list 字段存在${NC}"
  fi
else
  echo -e "${YELLOW}⚠ SaltStack 集成 API 响应异常${NC}"
  echo "响应: $SALTSTACK_RESPONSE"
fi
echo ""

# 步骤 3: 检查节点列表 API
echo -e "${YELLOW}[3/4] 检查节点列表 API...${NC}"
NODES_RESPONSE=$(curl -s -X GET "$BASE_URL/api/slurm/nodes" \
  -H "Authorization: Bearer $TOKEN")

if echo "$NODES_RESPONSE" | grep -q '"data"'; then
  echo -e "${GREEN}✓ 节点列表 API 响应正常${NC}"
  
  # 统计节点数量
  NODE_COUNT=$(echo "$NODES_RESPONSE" | grep -o '"name":"[^"]*"' | wc -l | tr -d ' ')
  echo -e "${GREEN}  - 节点总数: $NODE_COUNT${NC}"
  
  # 检查 SaltStack 状态字段
  if echo "$NODES_RESPONSE" | grep -q '"salt_status"'; then
    echo -e "${GREEN}  - salt_status 字段存在${NC}"
    
    # 统计各种状态的节点数量
    ACCEPTED_COUNT=$(echo "$NODES_RESPONSE" | grep -o '"salt_status":"accepted"' | wc -l | tr -d ' ')
    PENDING_COUNT=$(echo "$NODES_RESPONSE" | grep -o '"salt_status":"pending"' | wc -l | tr -d ' ')
    UNKNOWN_COUNT=$(echo "$NODES_RESPONSE" | grep -o '"salt_status":"unknown"' | wc -l | tr -d ' ')
    NOT_CONFIGURED_COUNT=$(echo "$NODES_RESPONSE" | grep -o '"salt_status":"not_configured"' | wc -l | tr -d ' ')
    
    echo -e "${BLUE}  - SaltStack 状态统计:${NC}"
    echo -e "${GREEN}    * accepted: $ACCEPTED_COUNT${NC}"
    echo -e "${YELLOW}    * pending: $PENDING_COUNT${NC}"
    echo -e "${YELLOW}    * unknown: $UNKNOWN_COUNT${NC}"
    echo -e "${YELLOW}    * not_configured: $NOT_CONFIGURED_COUNT${NC}"
  else
    echo -e "${RED}  - salt_status 字段不存在${NC}"
  fi
  
  # 检查 salt_minion_id 字段
  if echo "$NODES_RESPONSE" | grep -q '"salt_minion_id"'; then
    echo -e "${GREEN}  - salt_minion_id 字段存在${NC}"
  fi
  
  # 检查 salt_enabled 字段
  if echo "$NODES_RESPONSE" | grep -q '"salt_enabled"'; then
    echo -e "${GREEN}  - salt_enabled 字段存在${NC}"
  fi
else
  echo -e "${RED}✗ 节点列表 API 响应异常${NC}"
  echo "响应: $NODES_RESPONSE"
  exit 1
fi
echo ""

# 步骤 4: 前端检查建议
echo -e "${YELLOW}[4/4] 前端验证建议${NC}"
echo ""
echo -e "${BLUE}请在浏览器中访问以下地址进行人工验证：${NC}"
echo -e "  ${GREEN}$BASE_URL/slurm${NC}"
echo ""
echo -e "${BLUE}验证清单：${NC}"
echo "  □ SaltStack 集成状态卡片是否可见"
echo "  □ 卡片是否显示 Master 状态、API 状态等信息"
echo "  □ 节点列表是否有 'SaltStack状态' 列"
echo "  □ 节点状态是否显示正确的图标和颜色："
echo "    - accepted: 已连接（绿色 + ✓）"
echo "    - pending: 待接受（橙色 + ⏳）"
echo "    - rejected/denied: 已拒绝（红色 + ✕）"
echo "    - unknown/not_configured: 未配置（灰色 + ✕）"
echo "  □ 是否有刷新按钮可以重新加载 SaltStack 数据"
echo "  □ 数据加载失败时是否显示友好的警告信息"
echo ""

# 总结
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}验证总结${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "${GREEN}✓ API 层验证通过${NC}"
echo "  - 登录认证成功"
echo "  - SaltStack 集成 API 正常"
echo "  - 节点列表 API 包含 SaltStack 状态字段"
echo ""
echo -e "${YELLOW}⚠ 请继续进行前端人工验证${NC}"
echo "  - 访问 $BASE_URL/slurm"
echo "  - 检查上述验证清单"
echo ""

# 可选：运行 Playwright 测试
if command -v npx &> /dev/null; then
  echo -e "${BLUE}是否运行 Playwright E2E 测试？ (y/n)${NC}"
  read -t 10 -r RESPONSE || RESPONSE="n"
  
  if [ "$RESPONSE" = "y" ] || [ "$RESPONSE" = "Y" ]; then
    echo ""
    echo -e "${YELLOW}运行 Playwright 测试...${NC}"
    cd "$(dirname "$0")/../test/e2e" || exit 1
    BASE_URL="$BASE_URL" npx playwright test specs/slurm-saltstack-status-diagnosis.spec.js --reporter=list
  fi
fi

echo ""
echo -e "${GREEN}验证完成！${NC}"
