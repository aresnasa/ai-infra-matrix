# SLURM Configuration Templates

此目录包含 SLURM 的配置文件模板,用于不同的部署环境。

## 配置文件说明

### 主配置模板

- **slurm.conf.template** - 标准 SLURM 配置模板
  - 适用于物理机和虚拟机环境
  - 支持完整的 cgroup 资源约束
  - 通过环境变量动态生成配置

- **slurm-docker-minimal.conf.template** - Docker 环境最小化配置
  - 专为 Docker 容器环境优化
  - 禁用所有 cgroup 相关功能
  - 使用 `ProctrackType=proctrack/pgid`
  - 使用 `TaskPlugin=task/none`
  - 推荐用于开发和测试环境

- **slurm-docker-full.conf.template** - Docker 环境完整配置
  - 包含更多功能特性
  - 禁用 cgroup 但保留其他高级功能
  - 使用 `ProctrackType=proctrack/linuxproc`
  - 使用 `TaskPlugin=task/affinity`

### Cgroup 配置

- **cgroup.conf.template** - 标准 cgroup 配置
  - 适用于物理机/VM 环境
  - 启用 cgroup v2 和资源约束

- **cgroup-docker.conf.template** - Docker 环境 cgroup 配置
  - 禁用 cgroup 功能
  - 避免容器内 cgroup 初始化失败

### 其他配置

- **mpi.conf.template** - MPI 配置模板
- **slurmdbd.conf.template** - SLURM 数据库守护进程配置

## 环境变量

配置模板支持以下环境变量:

### 基础配置
- `SLURM_CLUSTER_NAME` - 集群名称
- `SLURM_CONTROLLER_HOST` - 控制器主机名
- `SLURM_CONTROLLER_PORT` - 控制器端口
- `SLURM_SLURMDBD_HOST` - 数据库守护进程主机
- `SLURM_SLURMDBD_PORT` - 数据库守护进程端口

### 认证配置
- `SLURM_AUTH_TYPE` - 认证类型 (默认: auth/munge)
- `SLURM_AUTH_ALT_TYPES` - 备用认证类型
- `SLURM_AUTH_ALT_PARAMETERS` - 备用认证参数

### 插件配置
- `SLURM_PLUGIN_DIR` - 插件目录路径
- `SLURM_TASK_PLUGIN` - 任务插件 (Docker: task/none, 物理机: task/affinity,task/cgroup)
- `SLURM_PROCTRACK_TYPE` - 进程跟踪类型 (Docker: proctrack/pgid, 物理机: proctrack/cgroup)
- `SLURM_JOB_CONTAINER_TYPE` - 作业容器类型 (Docker 建议留空)
- `SLURM_PROLOG_FLAGS` - Prolog 标志

### 资源配置
- `SLURM_MAX_JOB_COUNT` - 最大作业数
- `SLURM_MAX_ARRAY_SIZE` - 最大数组大小

## Docker 环境部署指南

### 问题说明

在 Docker 容器中运行 SLURM 时,标准的 cgroup 配置会导致以下问题:

1. **cgroup v2 初始化失败**
   ```
   error: cannot create cgroup context for cgroup/v2
   error: Unable to initialize cgroup plugin
   ```

2. **原因分析**
   - Docker 容器本身已被宿主机的 cgroup 管理
   - 容器内的 SLURM 无法再次创建 cgroup 层级
   - cgroup 文件系统在容器内可能是只读或受限的

### 解决方案

#### 方案 1: 使用最小化配置 (推荐)

```bash
# 使用 slurm-docker-minimal.conf.template
cp slurm-docker-minimal.conf.template slurm.conf
# 使用空的 cgroup 配置
cp cgroup-docker.conf.template cgroup.conf
```

**特点:**
- 完全禁用 cgroup
- 最简配置,启动速度快
- 适合开发测试环境

#### 方案 2: 使用完整配置

```bash
# 使用 slurm-docker-full.conf.template
cp slurm-docker-full.conf.template slurm.conf
# 设置环境变量
export SLURM_TASK_PLUGIN="task/affinity"
export SLURM_PROCTRACK_TYPE="proctrack/linuxproc"
export SLURM_JOB_CONTAINER_TYPE=""
export SLURM_PROLOG_FLAGS=""
```

**特点:**
- 保留更多 SLURM 功能
- 仅禁用 cgroup 相关部分
- 适合需要更多特性的场景

### 必需的目录创建

在启动 slurmd 之前,需要创建以下目录:

```bash
mkdir -p /var/run/slurm
mkdir -p /var/log/slurm
mkdir -p /var/spool/slurmd
chmod 755 /var/run/slurm /var/log/slurm /var/spool/slurmd
```

### Ubuntu vs Rocky Linux

**Ubuntu 节点** (✅ 测试通过)
- SLURM 25.05.4
- 使用最小化或完整配置都能正常工作

**Rocky Linux 节点** (⚠️ 已知问题)
- SLURM 25.05.4
- 即使禁用配置,仍尝试初始化 cgroup (可能是编译时硬编码)
- 建议在物理机环境测试

## 物理机/VM 环境部署

对于物理机或虚拟机环境,使用标准配置:

```bash
# 使用标准模板
cp slurm.conf.template slurm.conf
cp cgroup.conf.template cgroup.conf

# 设置环境变量
export SLURM_TASK_PLUGIN="task/affinity,task/cgroup"
export SLURM_PROCTRACK_TYPE="proctrack/cgroup"
export SLURM_JOB_CONTAINER_TYPE="job_container/tmpfs"
export SLURM_PROLOG_FLAGS="Contain"
```

## 测试验证

### 检查配置

```bash
# 测试配置文件语法
slurmd -c

# 查看节点状态
sinfo -Nel

# 提交测试作业
sbatch test.sh
squeue
```

### 预期结果

- **配置测试**: 无错误输出
- **节点状态**: idle (空闲)或 alloc (已分配)
- **作业状态**: R (运行中)或 PD (等待)

## 故障排查

### slurmd 启动失败

1. 检查日志: `/var/log/slurm/slurmd.log`
2. 查看 systemd 状态: `systemctl status slurmd`
3. 直接运行查看详细输出: `/usr/sbin/slurmd -D -vvv`

### cgroup 错误

如果看到 "cannot create cgroup context",说明:
- 可能在 Docker 环境但使用了物理机配置
- 切换到 Docker 专用配置模板

### 节点 down 状态

```bash
# 恢复节点
scontrol update nodename=NODE_NAME state=resume reason="recovered"

# 查看节点详情
scontrol show node NODE_NAME
```

## 参考文档

- [SLURM Configuration Tool](https://slurm.schedmd.com/configurator.html)
- [SLURM Cgroup Guide](https://slurm.schedmd.com/cgroups.html)
- [SLURM Docker Guide](https://github.com/SchedMD/slurm-docker-cluster)
