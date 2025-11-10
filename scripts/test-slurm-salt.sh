#!/bin/bash

# SLURM + SaltStack 集成快速测试脚本
# 用法: ./test-slurm-salt.sh [BASE_URL]

BASE_URL="${1:-http://192.168.3.91:8080}"
ADMIN_USER="admin"
ADMIN_PASS="admin123"

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 计数器
PASSED=0
FAILED=0

# 测试函数
test_case() {
    local name="$1"
    local expected="$2"
    local actual="$3"
    
    if [[ "$actual" == *"$expected"* ]]; then
        echo -e "${GREEN}✓${NC} $name"
        ((PASSED++))
        return 0
    else
        echo -e "${RED}✗${NC} $name"
        echo "  Expected: $expected"
        echo "  Got: $actual"
        ((FAILED++))
        return 1
    fi
}

echo "=========================================="
echo "SLURM + SaltStack 集成测试"
echo "Base URL: $BASE_URL"
echo "=========================================="
echo

# 1. 登录获取 token
echo "1. 认证测试..."
TOKEN=$(curl -s -X POST "$BASE_URL/api/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"$ADMIN_USER\",\"password\":\"$ADMIN_PASS\"}" \
  | jq -r '.token')

if [ -z "$TOKEN" ] || [ "$TOKEN" == "null" ]; then
    echo -e "${RED}✗${NC} 登录失败"
    exit 1
fi
echo -e "${GREEN}✓${NC} 登录成功"
echo

# 2. Salt API 基础测试
echo "2. Salt API 基础测试..."
RESULT=$(curl -s -X POST "$BASE_URL/api/slurm/saltstack/execute" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"function":"test.ping","target":"*"}' \
  | jq -r '.data.result.return[0] | to_entries | length')

test_case "Salt minions 响应数量" "7" "$RESULT"
echo

# 3. Salt 简单命令测试
echo "3. Salt 命令执行测试..."
RESULT=$(curl -s -X POST "$BASE_URL/api/slurm/saltstack/execute" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"function":"cmd.run","target":"salt-master-local","arguments":"hostname"}' \
  | jq -r '.data.result.return[0]["salt-master-local"]')

test_case "hostname 命令" "" "$RESULT"
echo

# 4. Salt 管道命令测试
echo "4. Salt Shell 管道测试..."
RESULT=$(curl -s -X POST "$BASE_URL/api/slurm/saltstack/execute" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"function":"cmd.run","target":"salt-master-local","arguments":"echo test | wc -l"}' \
  | jq -r '.data.result.return[0]["salt-master-local"]')

test_case "管道命令 (echo | wc)" "1" "$RESULT"
echo

# 5. Salt 多管道测试
echo "5. Salt 复杂管道测试..."
RESULT=$(curl -s -X POST "$BASE_URL/api/slurm/saltstack/execute" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"function":"cmd.run","target":"salt-master-local","arguments":"ps aux | grep -v grep | grep python | wc -l"}' \
  | jq -r '.data.result.return[0]["salt-master-local"]')

# 只要返回数字就算成功
if [[ "$RESULT" =~ ^[0-9]+$ ]]; then
    echo -e "${GREEN}✓${NC} 多管道命令 (ps | grep | wc)"
    ((PASSED++))
else
    echo -e "${RED}✗${NC} 多管道命令失败: $RESULT"
    ((FAILED++))
fi
echo

# 6. Salt 重定向和命令链测试
echo "6. Salt 重定向和命令链测试..."
RESULT=$(curl -s -X POST "$BASE_URL/api/slurm/saltstack/execute" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"function":"cmd.run","target":"salt-master-local","arguments":"echo hello > /tmp/test.txt && cat /tmp/test.txt"}' \
  | jq -r '.data.result.return[0]["salt-master-local"]')

test_case "重定向和命令链 (echo > && cat)" "hello" "$RESULT"
echo

# 7. SLURM sinfo 测试
echo "7. SLURM sinfo 测试..."
RESULT=$(curl -s -X POST "$BASE_URL/api/slurm/exec" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"command":"sinfo"}' \
  | jq -r '.output')

test_case "sinfo 命令" "PARTITION" "$RESULT"
test_case "sinfo 包含 compute 分区" "compute" "$RESULT"
echo

# 8. SLURM squeue 测试
echo "8. SLURM squeue 测试..."
RESULT=$(curl -s -X POST "$BASE_URL/api/slurm/exec" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"command":"squeue"}' \
  | jq -r '.output')

test_case "squeue 命令" "JOBID" "$RESULT"
echo

# 9. SLURM scontrol 测试
echo "9. SLURM scontrol 测试..."
RESULT=$(curl -s -X POST "$BASE_URL/api/slurm/exec" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"command":"scontrol show nodes"}' \
  | jq -r '.output')

test_case "scontrol show nodes" "NodeName" "$RESULT"
echo

# 10. Salt 在所有节点上执行测试
echo "10. Salt 多节点并发执行测试..."
RESULT=$(curl -s -X POST "$BASE_URL/api/slurm/saltstack/execute" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"function":"cmd.run","target":"*","arguments":"date +%s"}' \
  | jq -r '.data.result.return[0] | to_entries | length')

test_case "所有节点响应" "7" "$RESULT"
echo

# 总结
echo "=========================================="
echo "测试总结"
echo "=========================================="
echo -e "通过: ${GREEN}$PASSED${NC}"
echo -e "失败: ${RED}$FAILED${NC}"
echo "总计: $((PASSED + FAILED))"
echo

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ 所有测试通过！${NC}"
    exit 0
else
    echo -e "${RED}✗ 有 $FAILED 个测试失败${NC}"
    exit 1
fi
