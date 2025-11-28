#!/bin/bash
# 停止 slurmd 服务的脚本
# 用法: stop-slurmd.sh

echo "[INFO] 正在停止 slurmd 服务..."

# 尝试使用 systemctl
if command -v systemctl >/dev/null 2>&1; then
    if systemctl is-active slurmd >/dev/null 2>&1; then
        systemctl stop slurmd
        echo "[INFO] slurmd 已通过 systemctl 停止"
    fi
fi

# 强制杀死进程
pkill -9 slurmd 2>/dev/null || true

sleep 1

# 验证
if pgrep -x slurmd >/dev/null; then
    echo "[WARN] slurmd 进程仍在运行"
    ps aux | grep '[s]lurmd'
    exit 1
else
    echo "[SUCCESS] slurmd 已停止"
    exit 0
fi
