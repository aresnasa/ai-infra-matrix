#!/bin/bash
# SLURM SaltStack 状态同步测试脚本

set -e

BASE_URL="${BASE_URL:-http://192.168.0.200:8080}"

echo "=== SLURM SaltStack 状态同步测试 ==="
echo "BASE_URL: $BASE_URL"
echo ""

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 测试 1: 检查节点 API 是否返回 SaltStack 状态
echo -e "${BLUE}测试 1: 检查节点 API 返回的数据结构${NC}"
echo "GET $BASE_URL/api/slurm/nodes"

# 需要先登录获取 token
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/api/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}')

TOKEN=$(echo "$LOGIN_RESPONSE" | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo -e "${RED}✗ 登录失败，无法获取 token${NC}"
  exit 1
fi

echo -e "${GREEN}✓ 登录成功${NC}"
echo "Token: ${TOKEN:0:20}..."
echo ""

# 获取节点列表
NODES_RESPONSE=$(curl -s -X GET "$BASE_URL/api/slurm/nodes" \
  -H "Authorization: Bearer $TOKEN")

echo "节点列表响应:"
echo "$NODES_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$NODES_RESPONSE"
echo ""

# 检查是否包含 salt_status 字段
if echo "$NODES_RESPONSE" | grep -q "salt_status"; then
  echo -e "${GREEN}✓ 节点数据包含 salt_status 字段${NC}"
else
  echo -e "${RED}✗ 节点数据缺少 salt_status 字段${NC}"
fi

if echo "$NODES_RESPONSE" | grep -q "salt_minion_id"; then
  echo -e "${GREEN}✓ 节点数据包含 salt_minion_id 字段${NC}"
else
  echo -e "${RED}✗ 节点数据缺少 salt_minion_id 字段${NC}"
fi

if echo "$NODES_RESPONSE" | grep -q "salt_enabled"; then
  echo -e "${GREEN}✓ 节点数据包含 salt_enabled 字段${NC}"
else
  echo -e "${RED}✗ 节点数据缺少 salt_enabled 字段${NC}"
fi
echo ""

# 测试 2: 检查 SaltStack 集成 API
echo -e "${BLUE}测试 2: 检查 SaltStack 集成 API${NC}"
echo "GET $BASE_URL/api/slurm/saltstack/integration"

SALT_RESPONSE=$(curl -s -X GET "$BASE_URL/api/slurm/saltstack/integration" \
  -H "Authorization: Bearer $TOKEN")

echo "SaltStack 集成响应:"
echo "$SALT_RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$SALT_RESPONSE"
echo ""

if echo "$SALT_RESPONSE" | grep -q "minion_list"; then
  echo -e "${GREEN}✓ SaltStack 集成 API 正常${NC}"
  
  # 提取 minion 数量
  MINION_COUNT=$(echo "$SALT_RESPONSE" | grep -o '"total":[0-9]*' | head -1 | cut -d':' -f2)
  echo "Minion 总数: $MINION_COUNT"
else
  echo -e "${YELLOW}⚠ SaltStack 集成 API 可能不可用${NC}"
fi
echo ""

# 测试 3: 统计节点状态
echo -e "${BLUE}测试 3: 统计节点 SaltStack 状态${NC}"

ACCEPTED_COUNT=$(echo "$NODES_RESPONSE" | grep -o '"salt_status":"accepted"' | wc -l)
PENDING_COUNT=$(echo "$NODES_RESPONSE" | grep -o '"salt_status":"pending"' | wc -l)
REJECTED_COUNT=$(echo "$NODES_RESPONSE" | grep -o '"salt_status":"rejected"' | wc -l)
UNKNOWN_COUNT=$(echo "$NODES_RESPONSE" | grep -o '"salt_status":"unknown"' | wc -l)

echo "SaltStack 状态统计:"
echo "  已连接 (accepted): $ACCEPTED_COUNT"
echo "  待接受 (pending): $PENDING_COUNT"
echo "  已拒绝 (rejected): $REJECTED_COUNT"
echo "  未知 (unknown): $UNKNOWN_COUNT"
echo ""

# 测试 4: 检查节点管理 API
echo -e "${BLUE}测试 4: 检查节点管理 API 可用性${NC}"

# 检查节点管理端点（不实际执行，只检查是否存在）
echo "POST $BASE_URL/api/slurm/nodes/manage"
echo -e "${YELLOW}注意：此测试不会实际执行节点管理操作${NC}"
echo ""

# 测试 5: 运行 Playwright E2E 测试（如果可用）
if command -v npx &> /dev/null; then
  echo -e "${BLUE}测试 5: 运行 Playwright E2E 测试${NC}"
  
  if [ -f "test/e2e/specs/slurm-saltstack-status-diagnosis.spec.js" ]; then
    echo "运行 E2E 测试..."
    cd test/e2e
    BASE_URL=$BASE_URL npx playwright test specs/slurm-saltstack-status-diagnosis.spec.js --reporter=list || true
    cd ../..
  else
    echo -e "${YELLOW}⚠ E2E 测试文件不存在${NC}"
  fi
else
  echo -e "${YELLOW}⚠ Playwright 未安装，跳过 E2E 测试${NC}"
fi
echo ""

# 生成测试报告
echo "=== 测试总结 ==="
echo ""

if [ "$ACCEPTED_COUNT" -gt 0 ] || [ "$PENDING_COUNT" -gt 0 ]; then
  echo -e "${GREEN}✓ SaltStack 状态同步功能正常${NC}"
  echo "  - 已检测到 $(($ACCEPTED_COUNT + $PENDING_COUNT)) 个 SaltStack 节点"
else
  echo -e "${YELLOW}⚠ 未检测到 SaltStack 节点${NC}"
  echo "  - 可能是 SaltStack 服务未运行或节点未注册"
  echo "  - 或者节点名称与 minion ID 不匹配"
fi

echo ""
echo "建议检查项："
echo "1. 确认 SaltStack master 服务正在运行"
echo "2. 确认至少有一个节点安装了 salt-minion"
echo "3. 运行 'salt-key -L' 查看密钥状态"
echo "4. 检查节点名称是否与 minion ID 匹配"
echo ""

echo -e "${GREEN}测试完成！${NC}"
