# AppHub IP 解析错误修复

## 问题描述

构建过程中出现以下错误：
```
ERROR: failed to build: invalid host apphub:invalid IP
[INFO] [arm64] Using apphub at invalid IP
```

## 根本原因

在 `build.sh` 中，构建代码获取 AppHub 容器的 IP 地址时存在以下问题：

1. **容器状态检查不足**：代码直接尝试获取容器 IP，但没有先检查容器是否正在运行
2. **网络指定问题**：`docker inspect` 命令使用了错误的格式，可能返回空值或格式不对
3. **缺乏 IP 验证**：获取的 IP 地址没有验证其有效性（是否为正确的 IPv4 格式）
4. **没有回退机制**：当获取 IP 失败时没有回退到本地地址

## 解决方案

修改 [build.sh](build.sh#L7206-L7211) 中的 AppHub IP 获取逻辑：

### 改进项：

1. **容器状态预检**：先检查 AppHub 容器是否在运行
   ```bash
   apphub_container_status=$(docker ps --filter "name=^ai-infra-apphub$" --filter "status=running" ...)
   ```

2. **明确指定网络**：使用 `ai-infra-network` 明确获取指定网络上的 IP
   ```bash
   docker inspect -f '{{index .NetworkSettings.Networks "ai-infra-network" .IPAddress}}'
   ```

3. **IP 地址验证**：使用正则表达式验证获取的 IP 是否为有效的 IPv4 地址
   ```bash
   if [[ $apphub_ip =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]]; then
   ```

4. **优雅回退**：当获取 IP 失败时，回退到 `127.0.0.1`
   ```bash
   cmd+=("--add-host" "apphub:127.0.0.1")
   ```

## 文件变更

- **修改文件**：[build.sh](build.sh#L7206-L7228)
- **行数**：7206-7228（原为 7206-7211）

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

修改后，请重新执行构建命令：

```bash
./build.sh build gitea --force
```

预期结果：
- 如果 AppHub 容器运行正常，会看到：`Using apphub at <valid_ip>`
- 如果 AppHub 未运行，会自动回退到：`Using apphub at 127.0.0.1 (fallback)`
- 不会再出现 `invalid host apphub:invalid IP` 错误

## 相关文件

- [build.sh](build.sh) - 构建脚本（已修改）
- [src/gitea/Dockerfile](src/gitea/Dockerfile) - Gitea Dockerfile
- [src/slurm-master/Dockerfile](src/slurm-master/Dockerfile) - SLURM Dockerfile（使用 APPHUB_URL）
