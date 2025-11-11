#!/bin/bash
# 检查 slurmd 服务状态的脚本
# 用法: check-slurmd.sh

echo "=========================================="
echo "SLURM 服务状态检查"
echo "=========================================="

# 1. 检查 munge
echo -e "\n[1] Munge 服务状态:"
if pgrep -x munged >/dev/null; then
    echo "✓ munge 正在运行"
    ps aux | grep '[m]unged' | head -1
else
    echo "✗ munge 未运行"
fi

# 2. 检查 slurmd
echo -e "\n[2] Slurmd 服务状态:"
if pgrep -x slurmd >/dev/null; then
    echo "✓ slurmd 正在运行"
    ps aux | grep '[s]lurmd'
else
    echo "✗ slurmd 未运行"
fi

# 3. 检查配置文件
echo -e "\n[3] 配置文件检查:"
if [ -f /etc/slurm/slurm.conf ]; then
    echo "✓ /etc/slurm/slurm.conf 存在"
elif [ -f /etc/slurm-llnl/slurm.conf ]; then
    echo "✓ /etc/slurm-llnl/slurm.conf 存在"
else
    echo "✗ slurm.conf 不存在"
fi

if [ -f /etc/munge/munge.key ]; then
    echo "✓ /etc/munge/munge.key 存在"
    ls -l /etc/munge/munge.key
else
    echo "✗ munge.key 不存在"
fi

# 4. 检查日志
echo -e "\n[4] 最近的日志（最后10行）:"
if [ -f /var/log/slurm/slurmd.log ]; then
    tail -10 /var/log/slurm/slurmd.log
else
    echo "无日志文件"
fi

# 5. 检查网络连接
echo -e "\n[5] 网络连接检查:"
if command -v netstat >/dev/null 2>&1; then
    netstat -an | grep 6818 || echo "slurmd 端口 6818 未监听"
elif command -v ss >/dev/null 2>&1; then
    ss -an | grep 6818 || echo "slurmd 端口 6818 未监听"
fi

echo -e "\n=========================================="
