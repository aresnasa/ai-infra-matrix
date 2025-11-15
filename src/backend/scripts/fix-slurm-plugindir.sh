#!/bin/bash
# fix-slurm-plugindir.sh
# 修复SLURM PluginDir路径兼容性问题
# slurm-master(Ubuntu)使用: /usr/lib/aarch64-linux-gnu/slurm
# Rocky Linux使用: /usr/lib64/slurm
# 通过创建软链接来确保兼容性

set -e

echo "[INFO] 检查SLURM PluginDir路径..."

# 检测实际的插件目录
REAL_DIR=""
if [ -d /usr/lib64/slurm ]; then
    REAL_DIR="/usr/lib64/slurm"
    echo "[INFO] 发现Rocky Linux插件目录: $REAL_DIR"
elif [ -d /usr/lib/slurm ]; then
    REAL_DIR="/usr/lib/slurm"
    echo "[INFO] 发现标准插件目录: $REAL_DIR"
elif [ -d /usr/lib/aarch64-linux-gnu/slurm ]; then
    REAL_DIR="/usr/lib/aarch64-linux-gnu/slurm"
    echo "[INFO] 发现Ubuntu插件目录: $REAL_DIR (无需修复)"
    exit 0
elif [ -d /usr/lib/x86_64-linux-gnu/slurm ]; then
    REAL_DIR="/usr/lib/x86_64-linux-gnu/slurm"
    echo "[INFO] 发现Ubuntu x86_64插件目录: $REAL_DIR (无需修复)"
    exit 0
else
    echo "[WARN] 找不到SLURM插件目录，跳过修复"
    exit 0
fi

# 为多架构创建兼容软链接，避免slurm.conf中的PluginDir指向不存在的路径
create_compat_symlink() {
    local target_dir="$1"
    local parent_dir

    if [ -z "$target_dir" ]; then
        return
    fi

    if [ -L "$target_dir" ]; then
        echo "[INFO] 软链接已存在: $target_dir -> $(readlink -f "$target_dir")"
        return
    fi

    if [ -d "$target_dir" ]; then
        echo "[INFO] 目录 $target_dir 已存在，跳过创建"
        return
    fi

    parent_dir=$(dirname "$target_dir")
    mkdir -p "$parent_dir"
    ln -sf "$REAL_DIR" "$target_dir"
    echo "[INFO] ✓ 创建兼容软链接: $target_dir -> $REAL_DIR"
}

UBUNTU_ARM64_DIR="/usr/lib/aarch64-linux-gnu/slurm"
UBUNTU_X86_64_DIR="/usr/lib/x86_64-linux-gnu/slurm"

# 如果实际目录已经是Ubuntu路径，无需额外处理
if [ "$REAL_DIR" = "$UBUNTU_ARM64_DIR" ] || [ "$REAL_DIR" = "$UBUNTU_X86_64_DIR" ]; then
    echo "[INFO] PluginDir路径已是Ubuntu格式: $REAL_DIR"
    exit 0
fi

echo "[INFO] 创建跨架构PluginDir兼容软链接..."
create_compat_symlink "$UBUNTU_ARM64_DIR"
create_compat_symlink "$UBUNTU_X86_64_DIR"

echo "[INFO] ✓ PluginDir路径兼容性修复完成"
