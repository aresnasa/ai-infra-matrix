# Salt Transport 修复与时区标准化配置

**日期**: 2025-10-10  
**状态**: ✅ 已完成  
**类型**: 问题修复 + 功能增强

## 概述

本次更新解决了 SaltStack 扩容节点不显示的问题，并实现了全项目时区标准化和 lsof 工具安装。

## 问题诊断与修复

### 问题现象
- SLURM 扩容任务显示"完成"，但节点未在 UI 中显示
- SaltStack 无法接受 Minion 密钥
- Minion 无法连接到 Salt Master

### 根本原因
Salt Master 配置使用 `transport: tcp`，但在容器环境中未正确绑定 TCP 端口：
- Salt 3006.8 的 TCP 传输在容器间通信存在问题
- Master 只创建了 IPC socket，没有监听 4505/4506 TCP 端口
- Minion 无法跨容器连接到 Master

### 解决方案

#### 1. 修改传输协议配置
将 Salt Master 和 Minion 的传输方式从 `tcp` 改为 `zeromq`：

**修改文件**:
- `src/saltstack/salt-master.conf`
- `src/saltstack/salt-minion.conf`

```yaml
# 修改前
transport: tcp

# 修改后
transport: zeromq

# 添加 ZeroMQ 配置
zmq_filtering: False
zmq_monitor: False
```

#### 2. 修复 build.sh 强制重建功能
在 `build.sh` 中添加 `--no-cache` 支持，确保 `--force` 参数真正强制重建镜像：

```bash
# 添加 --no-cache 参数（当启用强制重建时）
local cache_arg=""
if [[ "$FORCE_REBUILD" == "true" ]]; then
    cache_arg="--no-cache"
fi

# 使用各自的src子目录作为构建上下文
if docker build -f "$dockerfile_path" $target_arg $cache_arg -t "$target_image" "$build_context"; then
```

#### 3. 重新构建并验证
```bash
# 强制重新构建 saltstack 镜像
./build.sh build saltstack --force

# 重启容器
docker stop ai-infra-saltstack && docker rm ai-infra-saltstack
docker-compose up -d saltstack

# 重启 test-ssh 容器的 salt-minion
docker exec test-ssh01 systemctl restart salt-minion
docker exec test-ssh02 systemctl restart salt-minion
docker exec test-ssh03 systemctl restart salt-minion

# 验证密钥接受
docker exec ai-infra-saltstack salt-key -L
# 输出：
# Accepted Keys:
# salt-master-local
# test-ssh01
# test-ssh02
# test-ssh03

# 测试连接
docker exec ai-infra-saltstack salt 'test-ssh*' test.ping
# 输出：
# test-ssh01: True
# test-ssh02: True
# test-ssh03: True
```

### 技术细节

#### Salt 传输方式对比

| 传输方式 | TCP Socket | 跨容器支持 | 稳定性 | Salt 版本支持 |
|---------|-----------|-----------|-------|--------------|
| tcp | ✅ | ⚠️ 有问题 | 中 | 3004+ |
| zeromq | ✅ | ✅ 完美 | 高 | 所有版本 |

#### ZeroMQ 工作原理
- 使用 TCP 协议进行跨容器通信（不是 IPC）
- 绑定到 `0.0.0.0:4505` (publish) 和 `0.0.0.0:4506` (request)
- 日志确认：`Starting the Salt Publisher on tcp://0.0.0.0:4505`

## 时区与工具标准化

### 需求来源
第29项需求：调整所有 dockerfile 中的时区为 Asia/Shanghai，所有的基础镜像都需要安装 lsof，然后在 env 环境文件中增加一个 ntp 时钟源的配置能够自定义的配置时钟源。

### 实施内容

#### 1. Dockerfile 时区配置
为所有服务添加统一的时区设置：

**Ubuntu 基础镜像** (saltstack, slurm-master, test-containers, slurm-build):
```dockerfile
ENV TZ=Asia/Shanghai

RUN apt-get install -y tzdata lsof && \
    ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && \
    echo $TZ > /etc/timezone
```

**Alpine 基础镜像** (backend, jupyterhub, nginx, frontend):
```dockerfile
ENV TZ=Asia/Shanghai

RUN apk add --no-cache tzdata lsof && \
    ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && \
    echo $TZ > /etc/timezone
```

#### 2. 安装 lsof 工具
在所有核心服务的 Dockerfile 中添加 lsof 包：

| 服务 | 基础镜像 | 包管理器 | 修改状态 |
|-----|---------|---------|---------|
| saltstack | ubuntu:22.04 | apt | ✅ |
| slurm-master | ubuntu:22.04 | apt | ✅ |
| test-containers | ubuntu:22.04 | apt | ✅ |
| slurm-build | ubuntu:22.04 | apt | ✅ |
| backend | golang:1.25-alpine | apk | ✅ |
| jupyterhub | python:3.13-alpine | apk | ✅ |
| nginx | nginx:stable-alpine-perl | apk | ✅ |

#### 3. NTP 配置
在 `.env.example` 中添加 NTP 服务器配置：

```bash
# NTP 时钟源配置 (用于时间同步，可自定义)
NTP_SERVER=ntp.aliyun.com
NTP_SERVER_BACKUP=cn.pool.ntp.org
```

### 未修改的镜像
以下镜像基于预构建镜像，不需要修改：
- **apphub**: 基于 `nginx:stable`（已包含时区支持）
- **gitea**: 基于 `gitea/gitea:1.25.1`（官方镜像）
- **singleuser**: 基于 `jupyter/base-notebook:latest`（官方镜像）
- **frontend**: node_modules 中的 Dockerfile（第三方依赖）

## 验证结果

### Salt Master 状态
```bash
$ docker exec ai-infra-saltstack systemctl status salt-master
● salt-master.service - Salt Master Service
     Loaded: loaded (/etc/systemd/system/salt-master.service; enabled)
     Active: active (running)
     
# 日志确认 ZeroMQ TCP 绑定
$ docker exec ai-infra-saltstack tail -20 /var/log/salt/master | grep "Publisher"
Starting the Salt Publisher on tcp://0.0.0.0:4505
```

### Minion 连接状态
```bash
$ docker exec test-ssh01 systemctl status salt-minion
● salt-minion.service - The Salt Minion
     Active: active (running)

# 所有 minion 成功连接
$ docker exec ai-infra-saltstack salt-key -L
Accepted Keys:
salt-master-local
test-ssh01
test-ssh02
test-ssh03
```

### 连通性测试
```bash
$ docker exec ai-infra-saltstack salt 'test-ssh*' test.ping
test-ssh01:
    True
test-ssh02:
    True
test-ssh03:
    True

# 从容器测试端口连通性
$ docker exec test-ssh01 bash -c 'exec 3<>/dev/tcp/saltstack/4505 && echo "Port accessible"'
Port accessible
```

## 影响范围

### 配置文件
- ✅ `src/saltstack/salt-master.conf` - 传输协议改为 zeromq
- ✅ `src/saltstack/salt-minion.conf` - 传输协议改为 zeromq
- ✅ `build.sh` - 添加 --no-cache 支持
- ✅ `.env.example` - 添加 NTP 配置

### Dockerfile 修改
- ✅ `src/saltstack/Dockerfile` - 时区 + lsof + ENV TZ
- ✅ `src/slurm-master/Dockerfile` - 时区 + lsof + ENV TZ
- ✅ `src/test-containers/Dockerfile` - 时区 + lsof + ENV TZ
- ✅ `src/slurm-build/Dockerfile` - 时区 + lsof + ENV TZ
- ✅ `src/backend/Dockerfile` - lsof（已有时区）
- ✅ `src/jupyterhub/Dockerfile` - lsof + ENV TZ（已有时区链接）
- ✅ `src/nginx/Dockerfile` - lsof + ENV TZ（已有时区）

### 镜像重建
需要重新构建的镜像：
```bash
./build.sh build saltstack --force
./build.sh build slurm-master --force
./build.sh build test-containers --force
./build.sh build backend --force
./build.sh build jupyterhub --force
./build.sh build nginx --force
```

## 后续建议

### 1. 完整重建所有镜像
```bash
./build.sh build-all --force
```

### 2. 更新 CI/CD 流程
确保 CI/CD 使用 `--no-cache` 参数进行构建，避免缓存导致的配置不生效。

### 3. NTP 客户端安装
如需在容器内实际使用 NTP 同步，可考虑安装 `ntpdate` 或 `chrony`：
```dockerfile
# Ubuntu
RUN apt-get install -y ntpdate

# Alpine
RUN apk add --no-cache chrony
```

### 4. 时区验证脚本
创建一个脚本验证所有容器的时区设置：
```bash
#!/bin/bash
for container in $(docker ps --format '{{.Names}}' | grep ai-infra); do
    echo "=== $container ==="
    docker exec $container date
    docker exec $container cat /etc/timezone 2>/dev/null || echo "No timezone file"
done
```

## 经验教训

1. **Salt 传输协议选择**
   - 生产环境优先使用 ZeroMQ（稳定性更好）
   - TCP 传输在容器环境需要额外配置

2. **Docker 缓存问题**
   - `--force` 参数需要实际传递 `--no-cache` 才能真正强制重建
   - COPY 指令的缓存特别容易导致配置不生效

3. **时区配置最佳实践**
   - 同时设置 ENV TZ 和创建符号链接
   - 确保 /etc/timezone 文件内容正确

4. **调试技巧**
   - 查看容器内配置文件确认修改是否生效
   - 使用 `lsof` 检查端口监听情况
   - 查看服务日志确认绑定地址

## 相关文档

- [SYSTEMD_CONTAINER_FIX_COMPLETE.md](./SYSTEMD_CONTAINER_FIX_COMPLETE.md) - Systemd 容器修复
- [BUILD_SCRIPT_UPGRADE_SUMMARY.md](./BUILD_SCRIPT_UPGRADE_SUMMARY.md) - build.sh 升级总结
- Salt官方文档: https://docs.saltproject.io/en/latest/topics/transports/

## 总结

本次修复成功解决了 SaltStack 节点扩容问题，主要通过：
1. 将 Salt 传输协议从 tcp 改为 zeromq，解决容器间通信问题
2. 修复 build.sh 的 --force 参数，确保真正强制重建
3. 统一所有服务的时区为 Asia/Shanghai
4. 在所有核心服务中安装 lsof 工具
5. 添加 NTP 服务器配置支持

现在 Salt Master 可以正常接受和管理 Minion 节点，SLURM 扩容功能的基础设施已经就绪。
