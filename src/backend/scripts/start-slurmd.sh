#!/bin/bash
# 启动 slurmd 服务的脚本
# 用法: start-slurmd.sh

set -e

echo "[INFO] 正在启动 slurmd 服务..."

# 1. 启动 munge 服务（认证）
echo "[INFO] 启动 munge 服务..."
if command -v systemctl >/dev/null 2>&1; then
    systemctl enable munge 2>/dev/null || true
    systemctl start munge 2>/dev/null || true
fi

# 确保 munge 运行
if ! pgrep -x munged >/dev/null; then
    /usr/sbin/munged -f 2>/dev/null || /usr/sbin/munged || true
fi

sleep 2

# 验证 munge
if pgrep -x munged >/dev/null; then
    echo "[SUCCESS] munge 服务已启动"
else
    echo "[WARN] munge 服务可能未启动"
fi

# 2. 创建必要的目录
mkdir -p /var/log/slurm /var/run/slurm /var/spool/slurm/slurmd
chmod 755 /var/spool/slurm/slurmd

# 3. 杀死可能存在的旧进程
echo "[INFO] 清理旧的 slurmd 进程..."
pkill -9 slurmd 2>/dev/null || true
sleep 1

# 4. 检测配置文件路径
if [ -f /etc/slurm-llnl/slurm.conf ]; then
    SLURM_CONF="/etc/slurm-llnl/slurm.conf"
elif [ -f /etc/slurm/slurm.conf ]; then
    SLURM_CONF="/etc/slurm/slurm.conf"
else
    echo "[ERROR] slurm.conf 不存在"
    exit 1
fi
echo "[INFO] 使用配置文件: $SLURM_CONF"

# 5. 获取节点名称
NODE_NAME=${1:-$(hostname -s)}
echo "[INFO] 节点名称: $NODE_NAME"

# 6. 启动 slurmd
echo "[INFO] 启动 slurmd 守护进程..."

# 尝试使用 systemctl
if command -v systemctl >/dev/null 2>&1; then
    systemctl enable slurmd 2>/dev/null || true
    systemctl start slurmd 2>/dev/null
    
    if systemctl is-active slurmd >/dev/null 2>&1; then
        echo "[SUCCESS] slurmd 通过 systemctl 启动成功"
        exit 0
    fi
fi

# 如果 systemctl 失败，使用 nohup 后台启动（明确指定配置文件和节点名称）
echo "[INFO] 使用 nohup 启动 slurmd (配置: $SLURM_CONF, 节点: $NODE_NAME)..."
nohup /usr/sbin/slurmd -D -f "$SLURM_CONF" -N "$NODE_NAME" > /var/log/slurm/slurmd.log 2>&1 </dev/null &
sleep 2

# 7. 验证启动
if pgrep -x slurmd >/dev/null; then
    echo "[SUCCESS] slurmd 启动成功"
    ps aux | grep '[s]lurmd'
    exit 0
else
    echo "[ERROR] slurmd 启动失败"
    echo "[ERROR] 查看日志:"
    tail -20 /var/log/slurm/slurmd.log 2>/dev/null || echo "无日志文件"
    exit 1
fi
