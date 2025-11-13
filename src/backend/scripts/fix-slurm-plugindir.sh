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

# 如果实际目录不是Ubuntu路径，创建软链接
UBUNTU_ARM64_DIR="/usr/lib/aarch64-linux-gnu/slurm"
UBUNTU_X86_64_DIR="/usr/lib/x86_64-linux-gnu/slurm"

# 根据架构创建对应的软链接
ARCH=$(uname -m)
if [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
    TARGET_DIR="$UBUNTU_ARM64_DIR"
    PARENT_DIR="/usr/lib/aarch64-linux-gnu"
elif [ "$ARCH" = "x86_64" ]; then
    TARGET_DIR="$UBUNTU_X86_64_DIR"
    PARENT_DIR="/usr/lib/x86_64-linux-gnu"
else
    echo "[WARN] 未知架构: $ARCH，跳过修复"
    exit 0
fi

# 如果实际目录已经是目标目录，无需修复
if [ "$REAL_DIR" = "$TARGET_DIR" ]; then
    echo "[INFO] PluginDir路径已正确，无需修复"
    exit 0
fi

# 创建软链接
echo "[INFO] 创建PluginDir软链接..."
mkdir -p "$PARENT_DIR"

# 如果目标已存在，先删除
if [ -L "$TARGET_DIR" ]; then
    echo "[INFO] 删除旧的软链接: $TARGET_DIR"
    rm -f "$TARGET_DIR"
elif [ -d "$TARGET_DIR" ]; then
    echo "[WARN] $TARGET_DIR 已存在且是目录，跳过"
    exit 0
fi

# 创建软链接
ln -sf "$REAL_DIR" "$TARGET_DIR"
echo "[INFO] ✓ 已创建软链接: $TARGET_DIR -> $REAL_DIR"

# 验证软链接
if [ -L "$TARGET_DIR" ]; then
    echo "[INFO] ✓ 软链接验证成功"
    ls -la "$TARGET_DIR"
else
    echo "[ERROR] 软链接创建失败"
    exit 1
fi

echo "[INFO] ✓ PluginDir路径修复完成"
