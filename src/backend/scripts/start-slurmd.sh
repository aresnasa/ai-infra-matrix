#!/bin/bash
# 启动 slurmd 服务的脚本
# 用法: start-slurmd.sh

set -e

echo "[INFO] 正在启动 slurmd 服务..."

prepare_directories() {
    echo "[INFO] 创建必要的目录..."
    mkdir -p /etc/munge /var/lib/munge /var/log/munge /run/munge
    mkdir -p /var/log/slurm /var/run/slurm /var/spool/slurm/slurmd

    # munge 权限配置：munge 用户需要对所有目录有写权限
    # /etc/munge - munge.key 所在目录
    # /var/lib/munge - munge socket 文件
    # /var/log/munge - munge 日志文件
    # /run/munge - munge 运行时文件
    chown -R munge:munge /etc/munge /var/lib/munge /var/log/munge /run/munge
    chmod 700 /etc/munge /var/lib/munge
    chmod 755 /var/log/munge /run/munge
    
    # munge.key 需要严格权限
    if [ -f /etc/munge/munge.key ]; then
        chown munge:munge /etc/munge/munge.key
        chmod 400 /etc/munge/munge.key
    fi
    
    # slurm 目录权限
    chown -R slurm:slurm /var/run/slurm /var/spool/slurm /var/log/slurm
    chmod 755 /var/run/slurm /var/spool/slurm /var/spool/slurm/slurmd /var/log/slurm
}

prepare_directories

# 1. 启动 munge 服务（认证）
echo "[INFO] 启动 munge 服务..."

# 确保 munge.key 存在
if [ ! -f /etc/munge/munge.key ]; then
    echo "[ERROR] munge.key 不存在，无法启动 munge"
    exit 1
fi

# 先停止已有的 munge 进程
pkill -9 munged 2>/dev/null || true
sleep 1

# 尝试使用 systemctl 启动
if command -v systemctl >/dev/null 2>&1; then
    systemctl enable munge 2>/dev/null || true
    systemctl start munge 2>/dev/null
    sleep 2
    
    if systemctl is-active munge >/dev/null 2>&1; then
        echo "[SUCCESS] munge 通过 systemctl 启动成功"
    else
        echo "[WARN] systemctl 启动 munge 失败，尝试直接启动..."
        # 直接启动 munged
        /usr/sbin/munged || true
        sleep 2
    fi
else
    # 没有 systemctl，直接启动
    /usr/sbin/munged || true
    sleep 2
fi

# 验证 munge
if pgrep -x munged >/dev/null; then
    echo "[SUCCESS] munge 服务已启动"
    # 测试 munge 认证
    if command -v munge >/dev/null 2>&1; then
        if echo "test" | munge | unmunge >/dev/null 2>&1; then
            echo "[SUCCESS] munge 认证测试成功"
        else
            echo "[WARN] munge 认证测试失败"
        fi
    fi
else
    echo "[ERROR] munge 服务启动失败"
    echo "[ERROR] 查看日志:"
    tail -20 /var/log/munge/munged.log 2>/dev/null || echo "无日志文件"
    exit 1
fi

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
