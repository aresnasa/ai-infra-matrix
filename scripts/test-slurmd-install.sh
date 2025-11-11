#!/bin/bash

# 测试 slurmd 安装和启动

echo "=========================================="
echo "测试 slurmd 安装和启动"
echo "=========================================="
echo

# 测试节点列表
NODES=("test-rocky01" "test-rocky02" "test-rocky03" "test-ssh01" "test-ssh02" "test-ssh03")

echo "步骤 1: 检查当前 slurmd 进程状态"
echo "=========================================="
for node in "${NODES[@]}"; do
    echo "=== $node ==="
    docker exec $node bash -c "ps aux | grep '[s]lurmd' || echo 'No slurmd process'"
    echo
done

echo
echo "步骤 2: 手动启动 slurmd（使用修复后的方法）"
echo "=========================================="

for node in "${NODES[@]}"; do
    echo "=== 启动 $node 上的 slurmd ==="
    
    # 检测操作系统
    OS_TYPE=$(docker exec $node bash -c "cat /etc/os-release | grep -i 'ID=' | head -1 | cut -d'=' -f2 | tr -d '\"'")
    echo "操作系统: $OS_TYPE"
    
    # 启动命令
    docker exec $node bash -c '
        # 杀死旧进程
        pkill -9 slurmd 2>/dev/null || true
        sleep 1
        
        # 创建日志目录
        mkdir -p /var/log/slurm
        
        # 尝试使用 systemctl
        if command -v systemctl >/dev/null 2>&1; then
            systemctl enable slurmd 2>/dev/null || true
            systemctl start slurmd 2>/dev/null
            if systemctl is-active slurmd >/dev/null 2>&1; then
                echo "✓ slurmd started via systemctl"
                exit 0
            fi
        fi
        
        # 使用 nohup 后台启动
        echo "Starting slurmd with nohup..."
        nohup /usr/sbin/slurmd -D > /var/log/slurm/slurmd.log 2>&1 </dev/null &
        sleep 2
        
        # 验证
        if pgrep -x slurmd >/dev/null; then
            echo "✓ slurmd started successfully"
        else
            echo "✗ slurmd failed to start"
            tail -20 /var/log/slurm/slurmd.log 2>/dev/null || echo "No log"
        fi
    '
    echo
done

echo
echo "步骤 3: 验证 slurmd 进程"
echo "=========================================="
sleep 3

for node in "${NODES[@]}"; do
    echo "=== $node ==="
    docker exec $node bash -c "ps aux | grep '[s]lurmd' || echo '✗ No slurmd process'"
    echo
done

echo
echo "步骤 4: 检查 SLURM 节点状态"
echo "=========================================="
docker exec ai-infra-slurm-master sinfo
echo

echo
echo "步骤 5: 详细节点信息"
echo "=========================================="
docker exec ai-infra-slurm-master scontrol show nodes | grep -E "(NodeName|State|Reason)" | head -30
echo

echo "=========================================="
echo "测试完成！"
echo "=========================================="
echo
echo "如果节点状态是 DOWN，可以尝试恢复："
echo "docker exec ai-infra-slurm-master scontrol update nodename=test-rocky01,test-rocky02,test-rocky03,test-ssh01,test-ssh02,test-ssh03 state=resume"
