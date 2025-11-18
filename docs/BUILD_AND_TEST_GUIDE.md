# AI Infrastructure Matrix - 构建和测试指南

## 概述

本文档描述如何使用统一的构建和测试流程来构建所有服务并启动测试环境。

## 快速开始

### 一键构建所有服务

使用 `build.sh` 脚本的 `build-all` 命令来构建所有服务镜像（包括测试容器）：

```bash
./build.sh build-all --force
```

这个命令会：
1. 生成/刷新 `.env` 配置文件（从 `.env.example`）
2. 同步配置文件到各个服务
3. 渲染配置模板（Nginx、Docker Compose 等）
4. 构建所有服务镜像：
   - backend
   - frontend
   - jupyterhub
   - nginx
   - saltstack
   - singleuser
   - gitea
   - backend-init
   - apphub
   - slurm-build
   - slurm-master
   - **test-containers** (新增)

### 启动测试环境

构建完成后，使用 `docker-compose.test.yml` 启动测试环境：

```bash
docker compose -f docker-compose.test.yml up -d
```

这会启动 3 个 SSH 测试容器：
- test-ssh01 (端口 2201)
- test-ssh02 (端口 2202)
- test-ssh03 (端口 2203)

## 详细说明

### 构建参数

`build-all` 命令支持以下参数：

```bash
./build.sh build-all [tag] [registry] [--force]
```

- `tag`: 镜像标签（默认：v0.3.6-dev）
- `registry`: 目标镜像仓库（可选，默认本地构建）
- `--force`: 强制覆盖生成 .env 等配置文件

**示例：**

```bash
# 使用默认标签构建
./build.sh build-all

# 指定版本标签
./build.sh build-all v1.0.0

# 构建并推送到私有仓库
./build.sh build-all v1.0.0 harbor.company.com/ai-infra --force
```

### 单独构建测试容器

如果只需要构建测试容器：

```bash
./build.sh build test-containers v0.3.6-dev
```

### 测试容器配置

测试容器基于 Ubuntu 22.04 + systemd，包含：

**系统组件：**
- systemd（PID 1）
- OpenSSH Server
- Python 3
- 基础网络工具（curl, wget, dnsutils, iputils-ping, net-tools）

**用户配置：**
- root 用户密码: `rootpass123`
- 测试用户: `testuser` / `testpass123`
- testuser 具有 sudo 权限（免密码）

**systemd 要求：**
- privileged: true
- security_opt: seccomp=unconfined
- cgroup: host
- stop_signal: SIGRTMIN+3
- tmpfs: /run, /run/lock
- volumes: /sys/fs/cgroup:/sys/fs/cgroup:rw

### Docker Compose 配置

`docker-compose.test.yml` 配置要点：

1. **镜像使用：** 优先使用已构建的镜像 `ai-infra-test-containers:${IMAGE_TAG}`
2. **回退构建：** 如果镜像不存在，会自动从 Dockerfile 构建
3. **网络：** 使用外部网络 `ai-infra-network`（需要提前创建）
4. **systemd 支持：** 所有容器都配置为支持 systemd 作为 PID 1

### 创建测试网络

如果 `ai-infra-network` 不存在，需要先创建：

```bash
docker network create ai-infra-network
```

### 验证测试环境

检查容器状态：

```bash
docker ps | grep test-ssh
```

预期输出应显示 3 个容器都在运行且健康。

测试 SSH 连接：

```bash
# 连接到 test-ssh01
ssh -p 2201 testuser@localhost
# 密码: testpass123

# 连接到 test-ssh02
ssh -p 2202 testuser@localhost

# 连接到 test-ssh03
ssh -p 2203 testuser@localhost
```

验证 systemd：

```bash
docker exec test-ssh01 systemctl status
```

应该显示 systemd 正在运行，State: running。

## 完整的构建和测试流程

### 步骤 1: 构建所有服务

```bash
./build.sh build-all --force
```

**预期输出：**
- ✓ 配置文件同步完成
- ✓ 所有模板渲染完成
- ✓ 构建完成: 12/12 成功

### 步骤 2: 创建网络（如果不存在）

```bash
docker network create ai-infra-network 2>/dev/null || echo "网络已存在"
```

### 步骤 3: 启动测试容器

```bash
docker compose -f docker-compose.test.yml up -d
```

**预期输出：**
```
✔ Container test-ssh01  Started
✔ Container test-ssh02  Started
✔ Container test-ssh03  Started
```

### 步骤 4: 验证部署

```bash
# 检查容器状态
docker ps | grep test-ssh

# 检查 systemd 状态
docker exec test-ssh01 systemctl status --no-pager | head -10
docker exec test-ssh02 systemctl status --no-pager | head -10
docker exec test-ssh03 systemctl status --no-pager | head -10

# 测试 SSH 连接
ssh -p 2201 testuser@localhost 'echo "SSH OK"'
```

## 故障排查

### 问题 1: 构建失败 - 镜像拉取超时

**原因：** 网络问题或 Docker Hub 限流

**解决：**
- 使用镜像加速器
- 重试构建命令
- 检查 `/etc/apt/sources.list` 中的镜像源配置

### 问题 2: 容器启动后立即退出

**原因：** systemd 无法在 Docker 中运行

**解决：**
- 确保 Docker 版本 >= 20.10
- 检查 `docker-compose.test.yml` 中的 systemd 配置
- 验证 cgroup v2 支持

### 问题 3: SSH 连接被拒绝

**原因：** SSH 服务未启动或端口冲突

**解决：**
```bash
# 检查容器日志
docker logs test-ssh01

# 检查 SSH 服务状态
docker exec test-ssh01 systemctl status ssh

# 检查端口占用
lsof -i :2201
```

### 问题 4: 网络不通

**原因：** Docker 网络未创建或配置错误

**解决：**
```bash
# 检查网络
docker network ls | grep ai-infra

# 重新创建网络
docker network rm ai-infra-network
docker network create ai-infra-network

# 重启容器
docker compose -f docker-compose.test.yml down
docker compose -f docker-compose.test.yml up -d
```

## 清理环境

### 停止测试容器

```bash
docker compose -f docker-compose.test.yml down
```

### 删除测试镜像

```bash
docker rmi ai-infra-test-containers:v0.3.6-dev
```

### 清理所有未使用的镜像和容器

```bash
docker system prune -a --volumes
```

## 最佳实践

1. **版本管理：** 始终为生产环境使用明确的版本标签
2. **增量构建：** 只在代码变更时使用 `--force` 重建
3. **网络隔离：** 为不同的环境使用不同的 Docker 网络
4. **日志监控：** 定期检查容器日志以发现问题
5. **资源清理：** 定期清理未使用的镜像和容器以节省空间

## 相关文档

- [Docker Compose v2 兼容性指南](./DOCKER_COMPOSE_V2.39.2_COMPATIBILITY.md)
- [Systemd 容器部署指南](./SYSTEMD_CONTAINER_GUIDE.md)
- [开发环境设置](./DEVELOPMENT_SETUP.md)

## 更新历史

- 2025-10-09: 添加 test-containers 支持，统一构建流程
- 2025-10-09: 修复 ARM64 镜像源问题
- 2025-10-09: 添加 systemd 容器配置要求
