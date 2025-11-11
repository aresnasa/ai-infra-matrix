# SLURM + SaltStack 安装指南

## 概述

本文档说明如何使用 SaltStack 从 AppHub 安装和配置 SLURM 集群节点。

## 架构

```
┌─────────────────┐
│  AppHub         │  提供 SLURM 25.05.4 包
│  (RPM + DEB)    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐        ┌─────────────────┐
│  Salt Master    │───────▶│  Target Node    │
│ (ai-infra-      │  RPC   │ (test-rocky01)  │
│  saltstack)     │        └─────────────────┘
└─────────────────┘

```

## 安装脚本

### 1. install-slurm-node.sh

**用途**：在新节点上安装 SLURM 包

**位置**：`src/backend/scripts/install-slurm-node.sh`

**功能**：
- 检测操作系统类型（Rocky/Ubuntu）
- 配置 AppHub 仓库（RPM 或 DEB）
- 安装 munge 认证服务
- 安装 SLURM 包（slurm, slurmd, libpmi）
- 创建目录结构
- 配置 systemd 服务

**使用方法**：
```bash
./install-slurm-node.sh <apphub_url> <node_type>

# 示例
./install-slurm-node.sh http://ai-infra-apphub compute
```

**参数**：
- `apphub_url`: AppHub 服务器 URL（默认：http://ai-infra-apphub）
- `node_type`: 节点类型（compute 或 login，默认：compute）

### 2. configure-slurm-node.sh

**用途**：配置已安装 SLURM 的节点

**位置**：`src/backend/scripts/configure-slurm-node.sh`

**功能**：
- 部署 munge key（从 master）
- 部署 slurm.conf（从 master）
- 启动 munge 服务
- 启动 slurmd 守护进程

**使用方法**：
```bash
./configure-slurm-node.sh <master_host> <munge_key_b64> <slurm_conf_b64>

# 示例
MUNGE_KEY=$(cat /etc/munge/munge.key | base64)
SLURM_CONF=$(cat /etc/slurm/slurm.conf | base64)
./configure-slurm-node.sh ai-infra-slurm-master "$MUNGE_KEY" "$SLURM_CONF"
```

**参数**：
- `master_host`: SLURM master 主机名
- `munge_key_b64`: Base64 编码的 munge key
- `slurm_conf_b64`: Base64 编码的 slurm.conf（可选，使用"-"跳过）

## 通过 SaltStack 安装

### 前提条件

1. Salt Master 运行中（ai-infra-saltstack）
2. 目标节点已接受 Salt Key
3. AppHub 服务运行中

### 安装步骤

#### 步骤 1：安装 SLURM 包

```bash
# 1. 复制安装脚本到 Salt Master
docker cp src/backend/scripts/install-slurm-node.sh ai-infra-saltstack:/tmp/

# 2. 传输脚本到目标节点
docker exec ai-infra-saltstack salt-cp 'test-rocky01' \
  /tmp/install-slurm-node.sh /tmp/install-slurm.sh

# 3. 执行安装
docker exec ai-infra-saltstack salt 'test-rocky01' cmd.run \
  '/tmp/install-slurm.sh http://ai-infra-apphub compute' \
  timeout=600
```

#### 步骤 2：配置和启动节点

```bash
# 1. 导出 master 配置（base64 编码）
docker exec ai-infra-slurm-master cat /etc/munge/munge.key | base64 > /tmp/munge.key.b64
docker exec ai-infra-slurm-master cat /etc/slurm/slurm.conf | base64 > /tmp/slurm.conf.b64

MUNGE_KEY_B64=$(cat /tmp/munge.key.b64 | tr -d '\n')
SLURM_CONF_B64=$(cat /tmp/slurm.conf.b64 | tr -d '\n')

# 2. 复制配置脚本到 Salt Master
docker cp src/backend/scripts/configure-slurm-node.sh ai-infra-saltstack:/tmp/

# 3. 传输脚本到目标节点
docker exec ai-infra-saltstack salt-cp 'test-rocky01' \
  /tmp/configure-slurm-node.sh /tmp/configure-slurm.sh

# 4. 执行配置
docker exec ai-infra-saltstack salt 'test-rocky01' cmd.run \
  "/tmp/configure-slurm.sh ai-infra-slurm-master '$MUNGE_KEY_B64' '$SLURM_CONF_B64'" \
  timeout=60
```

### 批量安装多个节点

```bash
# 安装阶段
for node in test-rocky01 test-rocky02 test-rocky03; do
  docker exec ai-infra-saltstack bash -c "
    salt-cp '$node' /tmp/install-slurm-node.sh /tmp/install-slurm.sh &&
    salt '$node' cmd.run '/tmp/install-slurm.sh http://ai-infra-apphub compute' timeout=600
  "
done

# 配置阶段
MUNGE_KEY_B64=$(docker exec ai-infra-slurm-master cat /etc/munge/munge.key | base64 | tr -d '\n')
SLURM_CONF_B64=$(docker exec ai-infra-slurm-master cat /etc/slurm/slurm.conf | base64 | tr -d '\n')

for node in test-rocky01 test-rocky02 test-rocky03; do
  docker exec ai-infra-saltstack bash -c "
    salt-cp '$node' /tmp/configure-slurm-node.sh /tmp/configure-slurm.sh &&
    salt '$node' cmd.run \"/tmp/configure-slurm.sh ai-infra-slurm-master '$MUNGE_KEY_B64' '$SLURM_CONF_B64'\" timeout=60
  "
done
```

## 验证安装

### 检查节点状态

```bash
# 查看集群状态
docker exec ai-infra-slurm-master sinfo

# 查看节点详情
docker exec ai-infra-slurm-master scontrol show nodes

# 检查节点进程
docker exec test-rocky01 pgrep -a slurmd
docker exec test-rocky01 pgrep -a munged
```

### 测试任务提交

```bash
# 简单测试
docker exec ai-infra-slurm-master srun -w test-rocky01 hostname

# 批量测试
docker exec ai-infra-slurm-master srun -N 3 hostname
```

## 支持的操作系统

| OS | 版本 | SLURM 包类型 | 状态 |
|----|------|-------------|------|
| Ubuntu | 22.04 | DEB (slurm-smd) | ✅ 已验证工作 |
| Rocky Linux | 9.3 | RPM (slurm) | ⚠️ 需要进一步测试 |

## 已知问题

### Rocky Linux 节点状态异常

**症状**：节点显示 `idle*` 或 `unk*`，无法接受任务

**可能原因**：
1. RPM 包使用 `--nodeps` 构建，可能缺少关键依赖
2. 与 master 的通信协议不兼容

**临时解决方案**：
- 优先使用 Ubuntu 节点
- 手动恢复节点：`scontrol update NodeName=test-rocky01 State=RESUME`

**长期解决方案**：
- 重新构建 RPM 包，确保完整依赖
- 或使用 DEB 包 + alien 工具转换

## Go 代码集成

### installSlurmPackages() 函数

位置：`src/backend/internal/services/slurm_service.go`

功能：
1. 读取安装脚本
2. 通过 Salt 传输到目标节点
3. 执行安装（600 秒超时）
4. 清理临时文件

### 新增：configureSlurmNodeViaSalt() 函数（待实现）

建议添加：
```go
func (s *SlurmService) configureSlurmNodeViaSalt(
    ctx context.Context, 
    nodeName string,
    logWriter io.Writer,
) error {
    // 1. 读取 master 的 munge.key 和 slurm.conf
    // 2. Base64 编码
    // 3. 通过 Salt 传输配置脚本
    // 4. 执行配置脚本
    // 5. 验证节点状态
}
```

## 故障排查

### 节点无法连接 master

```bash
# 检查网络连接
docker exec test-rocky01 ping ai-infra-slurm-master

# 检查 munge key 一致性
docker exec ai-infra-slurm-master md5sum /etc/munge/munge.key
docker exec test-rocky01 md5sum /etc/munge/munge.key
```

### slurmd 进程未运行

```bash
# 查看日志
docker exec test-rocky01 journalctl -xeu slurmd -n 50

# 手动启动
docker exec test-rocky01 /usr/sbin/slurmd -D -vvv
```

### Salt 执行超时

```bash
# 检查 Salt 连接
docker exec ai-infra-saltstack salt 'test-rocky01' test.ping

# 增加超时时间
docker exec ai-infra-saltstack salt 'test-rocky01' cmd.run \
  '<command>' timeout=1200
```

## 最佳实践

1. **分阶段部署**：先安装包，再配置服务
2. **批量操作**：使用 Salt 的 glob 模式批量执行
3. **验证每一步**：安装后检查包版本，配置后检查进程
4. **统一版本**：确保所有节点使用相同的 SLURM 版本（25.05.4）
5. **日志记录**：保存每次部署的完整日志

## 参考文档

- SLURM 官方文档：https://slurm.schedmd.com/
- SaltStack 文档：https://docs.saltproject.io/
- AppHub 构建说明：`docs/APPHUB_SLURM_BUILD.md`
