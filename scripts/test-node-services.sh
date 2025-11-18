#!/bin/bash
#
# test-node-services.sh - 测试节点上的 SLURM 和 Munge 服务状态
#

set -e

NODES=${@:-test-ssh01 test-ssh02 test-ssh03 test-rocky01 test-rocky02 test-rocky03}

echo "=========================================="
echo "测试节点服务状态"
echo "=========================================="
echo ""

for node in $NODES; do
    echo "=== $node ==="
    
    # 检查服务状态
    echo "--- 服务状态 ---"
    docker exec $node bash -c "systemctl status munge slurmd --no-pager" || true
    
    # 检查进程
    echo ""
    echo "--- 进程状态 ---"
    docker exec $node bash -c "ps aux | grep -E 'munged|slurmd' | grep -v grep" || echo "无相关进程"
    
    # 检查配置文件
    echo ""
    echo "--- 配置文件 ---"
    docker exec $node bash -c "ls -lh /etc/munge/munge.key /etc/slurm/slurm.conf /etc/slurm/cgroup.conf 2>/dev/null" || echo "配置文件缺失"
    
    # 检查目录权限
    echo ""
    echo "--- 关键目录权限 ---"
    docker exec $node bash -c "ls -ld /etc/munge /var/lib/munge /var/log/munge /run/munge /var/spool/slurm/slurmd 2>/dev/null" || echo "目录缺失"
    
    echo ""
    echo "=========================================="
    echo ""
done
