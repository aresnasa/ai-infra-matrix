#!/bin/bash

# 调试脚本：查询特定任务的详细信息
TASK_ID="5bd679be-4c5c-48b1-99a1-9c809c39a051"
API_BASE="http://localhost:8080/api"

echo "=========================================="
echo "调试任务ID: $TASK_ID"
echo "=========================================="

echo "1. 查询任务进度详情:"
curl -s "${API_BASE}/slurm/progress/${TASK_ID}" | jq '.' || echo "API调用失败"

echo ""
echo "2. 查询所有任务列表:"
curl -s "${API_BASE}/slurm/tasks" | jq '.data[] | select(.id == "'${TASK_ID}'")' || echo "未找到该任务"

echo ""
echo "3. 如果任务不存在，显示最近的任务:"
curl -s "${API_BASE}/slurm/tasks" | jq '.data | sort_by(.started_at) | reverse | .[0:5]' || echo "获取任务列表失败"

echo ""
echo "=========================================="
echo "调试完成"
echo "=========================================="