# AppHub IP 解析错误修复

## 问题描述

构建过程中出现以下错误：
```
ERROR: failed to build: invalid host apphub:invalid IP
[INFO] [arm64] Using apphub at invalid IP
```

## 问题分析

**诊断结果：AppHub 容器未运行**

错误的触发流程是：
1. 构建过程尝试获取 AppHub 容器的 IP 地址
2. 因为容器未运行，docker inspect 无法找到有效的 IP
3. 代码试图传递空的或无效的 IP 给 Docker buildx 的 --add-host 参数
4. Docker buildx 拒绝了无效的主机配置，报错：invalid host apphub:invalid IP

## 根本原因

在 build.sh 中，构建代码获取 AppHub 容器的 IP 地址时存在以下问题：

1. **缺乏容器状态检查**：代码直接尝试获取容器 IP，但没有先检查容器是否正在运行
2. **网络指定不明确**：docker inspect 命令使用了错误的格式，在容器连接多个网络时可能返回空值
3. **缺乏 IP 验证**：获取的 IP 地址没有验证其有效性（是否为正确的 IPv4 格式）
4. **没有回退机制**：当获取 IP 失败时没有优雅的回退方案

## 解决方案

### 1. 代码修复（已应用）

修改 build.sh 第 7206-7228 行的 AppHub IP 获取逻辑：

**改进项：**

1. **容器状态预检**：先检查 AppHub 容器是否在运行
   ```bash
   apphub_container_status=$(docker ps --filter "name=^ai-infra-apphub$" --filter "status=running" ...)
   ```

2. **明确指定网络**：使用 ai-infra-network 明确获取指定网络上的 IP
   ```bash
   docker inspect -f '{{index .NetworkSettings.Networks "ai-infra-network" .IPAddress}}'
   ```

3. **IP 地址验证**：使用正则表达式验证获取的 IP 是否为有效的 IPv4 地址
   ```bash
   if [[ $apphub_ip =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]]; then
   ```

4. **优雅回退**：当获取 IP 失败或容器未运行时，回退到 127.0.0.1
   ```bash
   cmd+=("--add-host" "apphub:127.0.0.1")
   ```

### 2. 启动 AppHub 容器（重要）

**首先，需要启动 AppHub 容器：**

```bash
cd /Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix

# 启动 AppHub 容器
docker-compose up -d apphub

# 等待容器就绪（观察健康检查）
docker-compose ps apphub

# 验证容器正在运行
docker ps | grep ai-infra-apphub
```

## 快速诊断

运行诊断脚本来检查 AppHub 配置：

```bash
bash diagnose-apphub.sh
```

此脚本会检查：
- Docker 服务状态
- ai-infra-network 网络
- AppHub 容器运行状态
- AppHub IP 地址有效性
- AppHub 网络连接性
- 构建环境配置

## 验证步骤

完成以上步骤后，重新执行构建命令：

```bash
./build.sh build gitea --force
```

预期结果：
- 如果 AppHub 容器运行正常，会看到：Using apphub at <valid_ip>
- 如果 AppHub 未运行（特殊情况下），会自动回退到：Using apphub at 127.0.0.1 (fallback)
- 不会再出现 invalid host apphub:invalid IP 错误

## 文件变更

- **修改文件**：build.sh (第 7206-7228 行)
- **新增文件**：diagnose-apphub.sh (诊断脚本)

## 常见问题

### Q: AppHub 容器启动失败怎么办？

```bash
# 查看容器日志
docker logs ai-infra-apphub

# 检查容器配置
docker inspect ai-infra-apphub | jq '.State'

# 重新创建容器
docker-compose down apphub
docker-compose up -d apphub
```

### Q: AppHub 无法访问怎么办？

```bash
# 检查端口绑定
docker port ai-infra-apphub

# 检查网络连接
docker network inspect ai-infra-network

# 手动测试连接
curl -f http://localhost:28080/health
```

### Q: 如何查看完整的构建日志？

```bash
# 启用调试模式
export DEBUG=1
./build.sh build gitea --force
```

## 相关文件

- build.sh - 构建脚本（已修改）
- diagnose-apphub.sh - 诊断脚本（新增）
- docker-compose.yml - Docker Compose 配置
- src/gitea/Dockerfile - Gitea Dockerfile
- src/slurm-master/Dockerfile - SLURM Dockerfile（使用 APPHUB_URL）
