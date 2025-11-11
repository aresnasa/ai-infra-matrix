#!/bin/bash

# SLURM 安装诊断脚本

echo "=========================================="
echo "SLURM 安装诊断"
echo "=========================================="
echo

NODES=("test-rocky01" "test-rocky02" "test-rocky03" "test-ssh01" "test-ssh02" "test-ssh03")

# 选择一个节点进行详细诊断
TEST_NODE="test-rocky01"

echo "诊断节点: $TEST_NODE"
echo "=========================================="
echo

echo "步骤 1: 检查 SLURM 是否已安装"
echo "----------------------------------------"
docker exec $TEST_NODE bash -c "command -v slurmd && slurmd -V || echo '未安装 slurmd'"
echo

echo "步骤 2: 检查配置文件"
echo "----------------------------------------"
docker exec $TEST_NODE bash -c "ls -la /etc/slurm*/slurm.conf 2>/dev/null || echo '配置文件不存在'"
docker exec $TEST_NODE bash -c "ls -la /etc/munge/munge.key 2>/dev/null || echo 'munge.key 不存在'"
echo

echo "步骤 3: 尝试手动启动 munge"
echo "----------------------------------------"
docker exec $TEST_NODE bash -c "
    # 确保 munge 用户存在
    id munge 2>/dev/null || echo 'munge 用户不存在'
    
    # 设置权限
    if [ -f /etc/munge/munge.key ]; then
        chown munge:munge /etc/munge/munge.key 2>/dev/null || echo '无法设置 munge.key 权限'
        chmod 400 /etc/munge/munge.key
    fi
    
    # 启动 munge
    /usr/sbin/munged -f 2>&1 || /usr/sbin/munged 2>&1
    sleep 2
    
    # 验证
    if pgrep -x munged >/dev/null; then
        echo '✓ munge 启动成功'
        ps aux | grep '[m]unged'
    else
        echo '✗ munge 启动失败'
    fi
"
echo

echo "步骤 4: 检查 slurm.conf 内容"
echo "----------------------------------------"
docker exec $TEST_NODE bash -c "
    if [ -f /etc/slurm/slurm.conf ]; then
        echo '=== /etc/slurm/slurm.conf ==='
        head -20 /etc/slurm/slurm.conf
    elif [ -f /etc/slurm-llnl/slurm.conf ]; then
        echo '=== /etc/slurm-llnl/slurm.conf ==='
        head -20 /etc/slurm-llnl/slurm.conf
    else
        echo 'slurm.conf 不存在'
    fi
"
echo

echo "步骤 5: 尝试手动启动 slurmd（前台模式）"
echo "----------------------------------------"
docker exec $TEST_NODE bash -c "
    # 创建必要目录
    mkdir -p /var/log/slurm /var/run/slurm /var/spool/slurm/slurmd
    chmod 755 /var/spool/slurm/slurmd
    
    echo '尝试前台启动 slurmd（5秒后自动退出）...'
    timeout 5 /usr/sbin/slurmd -D -vvv 2>&1 || echo '前台启动测试完成'
"
echo

echo "步骤 6: 尝试后台启动 slurmd"
echo "----------------------------------------"
docker exec $TEST_NODE bash -c "
    # 杀死旧进程
    pkill -9 slurmd 2>/dev/null || true
    sleep 1
    
    # 后台启动
    nohup /usr/sbin/slurmd -D > /var/log/slurm/slurmd.log 2>&1 </dev/null &
    SLURMD_PID=\$!
    echo \"slurmd PID: \$SLURMD_PID\"
    sleep 3
    
    # 验证进程
    if ps -p \$SLURMD_PID >/dev/null 2>&1; then
        echo '✓ slurmd 进程存在'
        ps aux | grep '[s]lurmd'
    else
        echo '✗ slurmd 进程已退出'
    fi
    
    # 查看日志
    if [ -f /var/log/slurm/slurmd.log ]; then
        echo '=== 日志内容 ==='
        tail -30 /var/log/slurm/slurmd.log
    fi
"
echo

echo "步骤 7: 检查所有节点的 slurmd 状态"
echo "=========================================="
for node in "${NODES[@]}"; do
    echo "=== $node ==="
    docker exec $node bash -c "ps aux | grep '[s]lurmd' || echo '无 slurmd 进程'"
    echo
done

echo "=========================================="
echo "诊断完成"
echo "=========================================="
