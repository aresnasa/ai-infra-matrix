#!/bin/bash
#
# fix-slurm-plugin-dir.sh - 修复SLURM节点的PluginDir问题
#
# 此脚本在SLURM计算节点上创建必要的目录并重启slurmd服务
#

set -euo pipefail

echo "=========================================="
echo "修复 SLURM PluginDir 问题"
echo "=========================================="
echo ""

# 创建必要的目录
echo "📁 创建 SLURM 目录..."
mkdir -p /usr/lib/slurm \
         /var/spool/slurm/d \
         /var/spool/slurm/ctld \
         /var/log/slurm \
         /run/slurm

echo "✓ 目录已创建"

# 设置权限
echo "🔐 设置目录权限..."
chmod 755 /usr/lib/slurm
chmod 755 /run/slurm
chmod 755 /var/spool/slurm
chmod 755 /var/log/slurm

if id slurm &>/dev/null; then
    chown -R slurm:slurm /var/spool/slurm /var/log/slurm /run/slurm
    echo "✓ 权限已设置"
else
    echo "⚠️  slurm 用户不存在，跳过权限设置"
fi

# 检查插件目录是否有内容
echo ""
echo "🔍 检查插件目录..."
if [ -d "/usr/lib/slurm" ]; then
    PLUGIN_COUNT=$(ls /usr/lib/slurm 2>/dev/null | wc -l)
    echo "   /usr/lib/slurm: $PLUGIN_COUNT 个文件"
    
    if [ "$PLUGIN_COUNT" -eq 0 ]; then
        echo "⚠️  插件目录为空，尝试从其他位置复制..."
        
        # 尝试从常见位置复制插件
        ARCH=$(uname -m)
        CANDIDATES=(
            "/usr/lib/slurm-wlm"
            "/usr/lib/${ARCH}/slurm-wlm"
            "/usr/lib/${ARCH}/slurm"
            "/usr/lib64/slurm-wlm"
            "/usr/lib64/slurm"
        )
        
        FOUND=0
        for DIR in "${CANDIDATES[@]}"; do
            if [ -d "$DIR" ] && [ -n "$(ls -A "$DIR" 2>/dev/null)" ]; then
                echo "   找到插件目录: $DIR"
                if cp -a "$DIR/." /usr/lib/slurm/; then
                    echo "✓ 插件已复制到 /usr/lib/slurm"
                    FOUND=1
                    break
                fi
            fi
        done
        
        if [ $FOUND -eq 0 ]; then
            echo "⚠️  未找到插件文件，slurmd 可能仍然无法启动"
            echo "   请确保 SLURM 包已正确安装"
        fi
    else
        echo "✓ 插件目录有内容"
    fi
fi

# 重启 slurmd 服务
echo ""
echo "🔄 重启 slurmd 服务..."
if systemctl is-active --quiet slurmd; then
    systemctl restart slurmd
    echo "✓ slurmd 服务已重启"
else
    systemctl start slurmd
    echo "✓ slurmd 服务已启动"
fi

# 等待服务启动
sleep 2

# 检查服务状态
echo ""
echo "📊 检查服务状态..."
if systemctl is-active --quiet slurmd; then
    echo "✅ slurmd 服务运行正常"
    systemctl status slurmd --no-pager | head -10
else
    echo "❌ slurmd 服务启动失败"
    echo ""
    echo "查看日志:"
    journalctl -u slurmd -n 20 --no-pager
    exit 1
fi

echo ""
echo "=========================================="
echo "✓ 修复完成"
echo "=========================================="
