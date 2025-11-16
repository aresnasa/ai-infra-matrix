# SLURM Docker Environment Setup Guide

本文档说明如何在 Docker 容器环境中正确配置和运行 SLURM 计算节点。

## 问题背景

在 Docker 容器中运行 SLURM 节点时,默认的 cgroup v2 配置会导致 slurmd 启动失败:

```
error: cannot create cgroup context for cgroup/v2
error: Unable to initialize cgroup plugin
error: slurmd initialization failed
```

这是因为:
1. Docker 容器内的 cgroup 访问受限
2. SLURM 默认尝试使用 cgroup 进行资源管理和进程跟踪
3. 某些 cgroup 约束在容器环境中无法正常工作

## 解决方案

### 1. 配置文件调整

#### slurm.conf

在 Docker 环境中,需要禁用 cgroup 相关的插件:

```ini
# 进程跟踪 - 使用 pgid 而不是 cgroup
ProctrackType=proctrack/pgid

# 任务插件 - 使用 task/none 避免 cgroup 依赖
TaskPlugin=task/none

# MPI - 在测试环境可以设置为 none
MpiDefault=none

# 不使用 JobContainerType (需要 cgroup 支持)
# JobContainerType=job_container/tmpfs
```

#### cgroup.conf

创建一个空的或最小化的 cgroup.conf:

```ini
###
# Slurm cgroup configuration (Docker container mode - disabled)
# In Docker containers, resource management is handled by Docker itself
###
```

### 2. 必要的目录结构

确保以下目录存在且权限正确:

```bash
mkdir -p /var/run/slurm /var/log/slurm /var/spool/slurmd
chmod 755 /var/run/slurm /var/log/slurm /var/spool/slurmd
```

### 3. 系统差异

#### Ubuntu/Debian
- 配置目录: `/etc/slurm`
- 插件目录: `/usr/lib/aarch64-linux-gnu/slurm` (ARM) 或 `/usr/lib/x86_64-linux-gnu/slurm` (x86)
- 日志位置: `/var/log/slurm/slurmd.log`

#### Rocky Linux/RHEL
- 配置目录: `/etc/slurm` → `/usr/etc/slurm` (软链接)
- 插件目录: `/usr/lib64/slurm`
- 注意: Rocky Linux 的 SLURM 包可能硬编码了一些 cgroup 依赖,在 Docker 环境中可能更难配置

### 4. 验证步骤

#### 检查配置文件

```bash
# 查看配置
cat /etc/slurm/slurm.conf | grep -E "TaskPlugin|ProctrackType"

# 应该输出:
# ProctrackType=proctrack/pgid
# TaskPlugin=task/none
```

#### 测试 slurmd 配置

```bash
# 测试配置文件语法
/usr/sbin/slurmd -c

# 前台运行查看详细输出
/usr/sbin/slurmd -D -vvv
```

#### 启动服务

```bash
# 启动 munge (认证)
systemctl start munge

# 启动 slurmd
systemctl start slurmd

# 检查状态
systemctl status slurmd
```

### 5. 常见错误和解决方法

#### 错误: "cannot create cgroup context"

**原因**: 配置文件中仍然启用了 cgroup 插件

**解决**:
1. 检查 `/etc/slurm/slurm.conf` 中的 `TaskPlugin` 和 `ProctrackType`
2. 确保 cgroup.conf 是空的或不存在
3. 重启 slurmd

#### 错误: "Unable to open pidfile"

**原因**: `/var/run/slurm` 目录不存在

**解决**:
```bash
mkdir -p /var/run/slurm
chmod 755 /var/run/slurm
systemctl restart slurmd
```

#### 错误: "Address already in use"

**原因**: 端口 6818 已被占用

**解决**:
```bash
# 停止旧的进程
pkill -9 slurmd
# 重启服务
systemctl start slurmd
```

## 自动化配置

项目已包含自动检测 Docker 环境的脚本:

### install-slurm-node.sh

安装脚本会自动:
1. 检测是否在 Docker 容器中运行
2. 创建适合环境的 cgroup.conf
3. 创建必要的目录
4. 配置 systemd 服务

### slurm.conf.base

配置模板已更新,注释说明了 Docker 和物理机的不同配置:

```ini
# Docker/container configuration (without cgroup):
TaskPlugin=task/affinity
ProctrackType=proctrack/linuxproc
```

## 测试验证

### 查看节点状态

在 slurm-master 上:

```bash
# 查看所有节点
sinfo -Nel

# 应该看到节点状态为 idle
```

### 提交测试作业

```bash
# 创建测试脚本
cat > /tmp/test.sh << 'EOF'
#!/bin/bash
hostname
date
echo "Hello from SLURM!"
EOF

chmod +x /tmp/test.sh

# 提交作业
sbatch --output=/tmp/slurm-%j.out /tmp/test.sh

# 查看作业状态
squeue

# 等待作业完成后查看输出
cat /tmp/slurm-*.out
```

### 预期输出

作业应该成功运行,输出类似:

```
test-ssh01
Sun Nov 16 14:20:29 CST 2025
Hello from SLURM!
```

## 物理机环境

在物理服务器或虚拟机上,可以启用完整的 cgroup 支持:

```ini
# slurm.conf
TaskPlugin=task/affinity,task/cgroup
ProctrackType=proctrack/cgroup
JobContainerType=job_container/tmpfs
PrologFlags=Contain
```

```ini
# cgroup.conf
CgroupPlugin=cgroup/v2
CgroupMountpoint=/sys/fs/cgroup
ConstrainCores=yes
ConstrainRAMSpace=yes
ConstrainSwapSpace=yes
ConstrainDevices=yes
AllowedRAMSpace=100
AllowedSwapSpace=0
```

## 参考资料

- [SLURM Configuration Guide](https://slurm.schedmd.com/slurm.conf.html)
- [SLURM Cgroup Guide](https://slurm.schedmd.com/cgroup.conf.html)
- [Docker Cgroup v2](https://docs.docker.com/config/containers/resource_constraints/)

## 已知限制

1. **Rocky Linux 节点**: 在 Docker 环境中,Rocky Linux 打包的 SLURM 可能有额外的 cgroup 依赖,导致启动失败。建议在物理机环境测试 Rocky 节点。

2. **资源限制**: 没有 cgroup 支持时,SLURM 无法强制限制作业的 CPU/内存使用。在 Docker 环境中,这由 Docker 本身管理。

3. **MPI 支持**: PMIx 库可能不可用,建议在生产环境安装完整的 MPI 支持。
