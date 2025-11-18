#!/bin/sh
set -e

echo "Installing SLURM client tools..."

# 复制 SLURM 文件到系统目录
cp -r usr/local/slurm /usr/local/
cp -r etc/slurm /etc/ 2>/dev/null || mkdir -p /etc/slurm

# 设置执行权限
chmod +x /usr/local/slurm/bin/*

# 创建符号链接到 /usr/bin
for cmd in /usr/local/slurm/bin/*; do
    ln -sf "$cmd" /usr/bin/$(basename "$cmd")
done

# 配置动态库路径
if [ ! -f /etc/ld.so.conf.d/slurm.conf ]; then
    mkdir -p /etc/ld.so.conf.d
    echo "/usr/local/slurm/lib" > /etc/ld.so.conf.d/slurm.conf
    ldconfig 2>/dev/null || true
fi

# 配置环境变量
if ! grep -q 'SLURM_HOME' /etc/profile 2>/dev/null; then
    cat >> /etc/profile << 'EOPROFILE'

# SLURM Client Environment
export SLURM_HOME=/usr/local/slurm
export PATH=$SLURM_HOME/bin:$PATH
export LD_LIBRARY_PATH=$SLURM_HOME/lib:$LD_LIBRARY_PATH
EOPROFILE
fi

echo "SLURM client tools installed successfully!"
echo "Version: $(cat /usr/local/slurm/VERSION 2>/dev/null || echo 'unknown')"
echo ""
echo "Available commands:"
ls -1 /usr/local/slurm/bin/

echo ""
echo "To use SLURM commands in current shell, run:"
echo "  source /etc/profile"
