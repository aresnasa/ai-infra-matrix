#!/bin/bash

# 测试单节点 SLURM 安装（带详细日志）

echo "=========================================="
echo "测试 SLURM 节点安装 API"
echo "=========================================="

# 使用 curl 调用 API
echo "正在调用 InstallSlurmNode API..."
echo

response=$(curl -s -X POST http://localhost:8080/api/slurm/nodes/install \
  -H "Content-Type: application/json" \
  -d '{
    "node_name": "test-rocky01",
    "os_type": "rocky"
  }')

echo "API 响应:"
echo "=========================================="
echo "$response" | jq '.' || echo "$response"
echo "=========================================="
echo

# 提取日志
logs=$(echo "$response" | jq -r '.logs // empty')
if [ -n "$logs" ]; then
    echo "安装日志:"
    echo "=========================================="
    echo "$logs"
    echo "=========================================="
    echo
fi

# 再次检查节点状态
echo "验证安装结果:"
echo "=========================================="
docker exec test-rocky01 bash -c "
    echo '1. 检查 SLURM 包:'
    command -v slurmd && slurmd -V || echo '  ✗ slurmd 未安装'
    echo
    
    echo '2. 检查 munge:'
    command -v munged && munged --version || echo '  ✗ munged 未安装'
    echo
    
    echo '3. 检查配置文件:'
    ls -la /etc/slurm*/slurm.conf 2>/dev/null || echo '  ✗ slurm.conf 不存在'
    ls -la /etc/munge/munge.key 2>/dev/null || echo '  ✗ munge.key 不存在'
    echo
    
    echo '4. 检查进程:'
    ps aux | grep '[m]unged' || echo '  ✗ 无 munged 进程'
    ps aux | grep '[s]lurmd' || echo '  ✗ 无 slurmd 进程'
    echo
    
    echo '5. 检查日志:'
    if [ -f /var/log/slurm/slurmd.log ]; then
        echo '  slurmd.log (最后20行):'
        tail -20 /var/log/slurm/slurmd.log
    else
        echo '  ✗ /var/log/slurm/slurmd.log 不存在'
    fi
"

echo "=========================================="
echo "测试完成"
echo "=========================================="
