# AppHub SLURM cgroup 构建配置修复

## 问题描述

之前 SLURM 包的构建可能默认启用了 cgroup 支持，导致：
- Docker 容器环境中 slurmd 无法启动（cgroup v2 初始化失败）
- 缺乏灵活性，无法根据部署环境选择不同的进程跟踪方式

## 解决方案

### 1. RPM 包构建（Rocky Linux）

**修改位置**: `src/apphub/Dockerfile` Line ~473

**变更内容**:
```dockerfile
# 在 ~/.rpmmacros 中添加
echo '%without_cgroup --without-cgroup' >> ~/.rpmmacros;
```

**效果**:
- `rpmbuild` 构建时不启用 cgroup 支持
- 生成的 RPM 包不会硬编码要求 cgroup
- 运行时通过 `slurm.conf` 配置决定是否使用 cgroup

### 2. DEB 包构建（Ubuntu）

**修改位置**: `src/apphub/Dockerfile` Line ~168

**变更内容**:
```dockerfile
# 添加构建选项和说明
export DEB_BUILD_OPTIONS="nocheck parallel=$(nproc)";
```

**效果**:
- 优化构建速度（并行构建，跳过测试）
- 通过注释明确 cgroup 不在编译时硬编码
- 保持 debian/rules 的默认行为（通常不强制启用 cgroup）

### 3. 文档更新

**修改位置**: `src/apphub/Dockerfile` 顶部注释

**添加说明**:
```
# SLURM 构建策略说明：
# - cgroup 支持不在编译时硬编码启用，而是通过运行时配置管理
# - 这允许同一个 SLURM 包在 Docker 容器和物理机环境中灵活部署
# - Docker 环境：使用 proctrack/pgid 或 proctrack/linuxproc，无 cgroup
# - 物理机环境：可通过 slurm.conf 启用 proctrack/cgroup 和 task/cgroup
```

## 部署策略

### Docker 环境

使用以下配置模板：
- `slurm-docker-minimal.conf.template` - 极简配置
- `slurm-docker-full.conf.template` - 完整功能配置

关键配置：
```conf
ProctrackType=proctrack/pgid      # 或 proctrack/linuxproc
TaskPlugin=task/none              # 或 task/affinity
# 不使用 proctrack/cgroup 或 task/cgroup
```

### 物理机/虚拟机环境

使用标准配置模板：
- `slurm.conf.template`
- `cgroup.conf.template`

可选启用 cgroup：
```conf
ProctrackType=proctrack/cgroup
TaskPlugin=task/affinity,task/cgroup
JobContainerType=job_container/tmpfs
```

## 技术原理

### 为什么不在编译时启用 cgroup？

1. **Docker 容器限制**:
   - 容器已被 Docker 的 cgroup 管理
   - 容器内无法再创建子 cgroup 层级
   - 强制启用 cgroup 会导致 slurmd 启动失败

2. **灵活性需求**:
   - 开发测试环境使用 Docker
   - 生产环境可能使用物理机或 VM
   - 同一个包应该支持两种环境

3. **SLURM 设计理念**:
   - SLURM 支持多种进程跟踪机制
   - cgroup 只是其中一个选项
   - 通过配置文件选择更灵活

### 编译时 vs 运行时配置

| 配置项 | 编译时 | 运行时 | 推荐 |
|--------|--------|--------|------|
| cgroup 支持 | `--with-cgroup` | `ProctrackType=proctrack/cgroup` | 运行时 |
| munge 认证 | `--with-munge` | 默认启用 | 编译时 |
| MySQL 支持 | `--with-mysql` | 数据库连接配置 | 编译时 |

## 验证方法

### 1. 检查 RPM 包依赖

```bash
rpm -qpR /path/to/slurm-*.rpm | grep cgroup
# 应该没有输出或仅显示可选依赖
```

### 2. 检查 DEB 包依赖

```bash
dpkg-deb -I /path/to/slurm_*.deb | grep -i cgroup
# 应该没有硬性依赖（Depends），可能有建议（Recommends）
```

### 3. 测试 Docker 环境

```bash
# 使用 slurm-docker-minimal.conf
docker exec slurm-master scontrol show config | grep -i proctrack
# 应显示: ProctrackType = proctrack/pgid

# 检查节点状态
docker exec slurm-master sinfo
# 节点应该是 idle 状态，不是 down
```

### 4. 测试物理机环境

```bash
# 使用启用 cgroup 的配置
scontrol show config | grep -i proctrack
# 应显示: ProctrackType = proctrack/cgroup

# 检查 cgroup 挂载
mount | grep cgroup
# 应显示 cgroup2 挂载点
```

## 相关文件

- `src/apphub/Dockerfile` - 包构建配置
- `src/slurm-master/config/slurm-docker-*.conf.template` - Docker 配置模板
- `src/slurm-master/config/slurm.conf.template` - 标准配置模板
- `src/slurm-master/config/README.md` - 配置详细文档
- `src/backend/scripts/install-slurm-node.sh` - 节点安装脚本

## 注意事项

1. **重新构建镜像**: 修改后需要重新构建 AppHub 镜像
   ```bash
   ./build.sh --build-slurm
   ```

2. **清理旧包**: 删除之前构建的可能启用 cgroup 的包
   ```bash
   docker exec apphub rm -rf /usr/share/nginx/html/pkgs/slurm-*
   ```

3. **配置一致性**: 确保 slurm.conf 与实际部署环境匹配
   - Docker 环境：不使用 cgroup
   - 物理机环境：可选使用 cgroup

4. **测试流程**:
   - 先在 Docker 环境验证基本功能
   - 再在物理机环境测试 cgroup 功能
   - 确保两种环境都能正常工作

## 参考资料

- [SLURM 管理员快速指南 - RPM 构建](https://slurm.schedmd.com/quickstart_admin.html#rpmbuild)
- [SLURM cgroup 配置](https://slurm.schedmd.com/cgroup.html)
- [Docker 环境 SLURM 部署指南](../src/slurm-master/config/README.md)
- [SLURM 进程跟踪插件](https://slurm.schedmd.com/proctrack_plugins.html)

## 更新历史

- 2025-01-16: 初始版本，禁用编译时 cgroup 启用
- 说明文档创建，包含验证方法和部署策略
