#!/bin/bash
# ====================================================================
# 修复并测试 SLURM 任务取消功能
# ====================================================================

set -e

echo "======================================================================"
echo "修复 SLURM 任务查询逻辑"
echo "======================================================================"

# 重新构建 backend
echo "重新构建 backend..."
docker-compose build backend

# 重启 backend
echo "重启 backend 服务..."
docker-compose restart backend

# 等待服务启动
echo "等待服务启动（10秒）..."
sleep 10

# 测试任务列表
echo ""
echo "======================================================================"
echo "测试 1: 获取任务列表"
echo "======================================================================"
curl -s http://localhost:8080/api/slurm/tasks?page=1&limit=10 | jq '.'

echo ""
echo "======================================================================"
echo "测试完成！"
echo "======================================================================"
echo ""
echo "手动测试步骤:"
echo "  1. 访问 http://192.168.18.154:8080/slurm-tasks"
echo "  2. 查看任务列表是否正常加载"
echo "  3. 点击任意任务的「取消」按钮"
echo "  4. 验证取消操作是否成功"
echo ""
echo "如果需要创建测试任务:"
echo "  curl -X POST http://localhost:8080/api/v1/slurm/scale-up/async \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"partition\":\"compute\",\"node_count\":3,\"node_prefix\":\"test\"}'"
echo ""
