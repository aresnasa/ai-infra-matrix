#!/bin/sh

echo "Uninstalling SLURM client tools..."

# 删除 SLURM 文件
rm -rf /usr/local/slurm

# 删除符号链接
rm -f /usr/bin/sinfo /usr/bin/squeue /usr/bin/scontrol /usr/bin/scancel
rm -f /usr/bin/sbatch /usr/bin/srun /usr/bin/salloc /usr/bin/sacct

# 删除动态库配置
rm -f /etc/ld.so.conf.d/slurm.conf

# 删除配置目录
rm -rf /etc/slurm

# 从 /etc/profile 中删除环境变量配置
sed -i '/SLURM_HOME/,+2d' /etc/profile 2>/dev/null || true

echo "SLURM client tools uninstalled."
