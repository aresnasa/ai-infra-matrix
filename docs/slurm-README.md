# SLURM Alpine Client Tools

这是为 Alpine Linux 编译的 SLURM 客户端工具包。

## 安装

```bash
tar xzf slurm-client-*.tar.gz
cd slurm-client-*/
./install.sh
```

安装后需要加载环境变量：

```bash
source /etc/profile
```

## 验证安装

```bash
sinfo --version
which sinfo squeue scontrol
```

## 客户端工具说明

- `sinfo` - 查看集群/节点信息
- `squeue` - 查看作业队列
- `scontrol` - 管理工具
- `scancel` - 取消作业
- `sbatch` - 提交批处理作业
- `srun` - 运行并行作业
- `salloc` - 分配资源
- `sacct` - 作业统计

## 配置

安装后，您需要配置 `/etc/slurm/slurm.conf` 文件以连接到您的 SLURM 集群。

最小配置示例：

```conf
ClusterName=mycluster
ControlMachine=slurm-master
SlurmUser=slurm
SlurmctldPort=6817
SlurmdPort=6818
AuthType=auth/none
StateSaveLocation=/var/spool/slurm/ctld
SlurmdSpoolDir=/var/spool/slurm/d
SwitchType=switch/none
MpiDefault=none
ProctrackType=proctrack/pgid
ReturnToService=2
SlurmctldPidFile=/var/run/slurmctld.pid
SlurmdPidFile=/var/run/slurmd.pid

# Nodes and Partitions
NodeName=node[1-2] CPUs=4 State=UNKNOWN
PartitionName=debug Nodes=ALL Default=YES MaxTime=INFINITE State=UP
```

## 卸载

```bash
./uninstall.sh
```

## 编译选项

本包使用以下编译选项：

- `--prefix=/usr/local/slurm` - 安装路径
- `--sysconfdir=/etc/slurm` - 配置文件路径
- `--without-munge` - 不使用 Munge 认证（Alpine 不可用）
- `--without-pam` - 不使用 PAM（Alpine 不可用）
- `--without-numa` - 不使用 NUMA 支持（可选）
- `--without-hwloc` - 不使用 hwloc（可选）

## 版本信息

SLURM Version: 25.05.4
Build Platform: Alpine Linux (latest)
Build Date: 2025-10-20

## 故障排查

### 命令找不到

确保已经加载环境变量：

```bash
source /etc/profile
# 或
export PATH=/usr/local/slurm/bin:$PATH
```

### 库文件找不到

```bash
export LD_LIBRARY_PATH=/usr/local/slurm/lib:$LD_LIBRARY_PATH
ldconfig 2>/dev/null || true
```

### 无法连接到集群

检查 `/etc/slurm/slurm.conf` 配置文件，确保 `ControlMachine` 设置正确。

## 更多信息

官方文档: https://slurm.schedmd.com/documentation.html
