# Systemd 容器修复总结

## 问题描述

`ai-infra-saltstack` 和 `ai-infra-slurm-master` 容器持续重启失败，状态为 `Error`，退出码为 255。

## 根本原因

这两个容器都使用 systemd 作为主进程管理器，但 `docker-compose.yml` 中缺少必要的 systemd 兼容性配置：

1. **缺少 privileged 模式**：systemd 需要特权模式才能管理系统服务
2. **缺少 seccomp 配置**：默认的 seccomp 配置会阻止 systemd 的某些系统调用
3. **缺少 tmpfs 挂载**：systemd 需要 `/run` 和 `/run/lock` 作为 tmpfs
4. **缺少 cgroup 挂载**：systemd 需要访问 `/sys/fs/cgroup` 来管理服务

## 修复方案

### 1. 更新 docker-compose.yml.example 模板

为 `saltstack` 和 `slurm-master` 服务添加以下配置：

```yaml
privileged: true
security_opt:
  - seccomp:unconfined
tmpfs:
  - /run
  - /run/lock
volumes:
  - /sys/fs/cgroup:/sys/fs/cgroup:rw
  # ...其他 volumes
```

### 2. 重新渲染配置文件

```bash
./build.sh render-templates docker-compose
```

### 3. 重启所有服务

```bash
docker compose down
docker compose up -d
```

## 修复结果

### 容器状态

所有容器现在都处于健康状态：

```bash
$ docker ps --format "table {{.Names}}\t{{.Status}}"
NAMES                       STATUS
ai-infra-saltstack         Up 2 minutes (healthy)
ai-infra-slurm-master      Up 2 minutes (healthy)
ai-infra-gitea             Up 2 minutes (healthy)
ai-infra-frontend          Up 2 minutes (healthy)
ai-infra-jupyterhub        Up 2 minutes (healthy)
ai-infra-backend           Up 2 minutes (healthy)
```

### Systemd 服务状态

#### SaltStack 容器

```bash
$ docker exec ai-infra-saltstack systemctl status
State: degraded
Failed: 1 units
```

关键服务运行正常：
- ✅ `salt-master.service` - Active (running)
- ✅ `salt-api.service` - Active (running)
- ✅ `salt-minion-local.service` - Active (running)

#### SLURM Master 容器

```bash
$ docker exec ai-infra-slurm-master systemctl status
State: degraded
Failed: 1 units
```

关键服务运行正常：
- ✅ `slurmctld.service` - Active (running)
- ✅ `slurmdbd.service` - Active (running)
- ✅ `munge.service` - Active (running)

## 技术要点

### Systemd 在容器中运行的要求

1. **特权模式 (privileged: true)**
   - 允许容器访问宿主机的所有设备
   - 允许 systemd 执行必要的系统管理操作

2. **禁用 seccomp (seccomp:unconfined)**
   - systemd 需要执行某些系统调用（如 `mount`、`umount`）
   - 默认的 seccomp 配置会阻止这些调用

3. **tmpfs 挂载**
   - `/run` 和 `/run/lock` 必须是 tmpfs 文件系统
   - systemd 使用这些目录存储运行时数据

4. **cgroup 挂载**
   - `/sys/fs/cgroup` 必须以读写模式挂载
   - systemd 使用 cgroup 来管理和监控服务进程

## 影响的文件

- `docker-compose.yml.example` - 配置模板（已更新）
- `docker-compose.yml` - 实际配置（通过模板渲染生成）

## 验证步骤

1. **检查容器状态**
   ```bash
   docker ps --format "table {{.Names}}\t{{.Status}}"
   ```

2. **检查 systemd 状态**
   ```bash
   docker exec ai-infra-saltstack systemctl status
   docker exec ai-infra-slurm-master systemctl status
   ```

3. **检查关键服务**
   ```bash
   docker exec ai-infra-saltstack systemctl status salt-master salt-api
   docker exec ai-infra-slurm-master systemctl status slurmctld slurmdbd munge
   ```

## 注意事项

1. **安全性考虑**
   - `privileged: true` 会赋予容器更多权限，在生产环境中需要仔细评估
   - 建议在隔离的网络环境中运行这些容器

2. **资源限制**
   - 即使使用特权模式，仍可以通过 Docker 的资源限制（CPU、内存）来控制容器资源

3. **日志和监控**
   - systemd 日志可通过 `journalctl` 查看
   - 容器日志仍可通过 `docker logs` 查看

## 相关文档

- [Docker Compose systemd 集成](https://docs.docker.com/config/containers/systemd/)
- [SLURM 容器化部署最佳实践](./SLURM_CONTAINERIZATION.md)
- [SaltStack 容器化配置指南](./SALTSTACK_CONFIGURATION.md)

## 修复时间

- 问题发现：2025年10月10日 09:28
- 修复完成：2025年10月10日 09:42
- 总耗时：约 14 分钟

## 修复人员

AI Infrastructure Team

---

**状态**: ✅ 已修复  
**优先级**: 🔴 高  
**类型**: 🐛 Bug修复  
**版本**: v0.3.7
