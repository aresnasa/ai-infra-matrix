# SLURM 节点自动安装功能

本文档描述如何使用自动化安装功能为 SLURM 集群添加新的计算节点。

## 功能概述

该功能通过 REST API 自动在指定节点上安装和配置 SLURM 客户端，包括：

1. 自动检测操作系统类型并安装相应的 SLURM 包
2. 从 SLURM master 获取并部署配置文件
3. 配置 Munge 认证
4. 启动 slurmd 服务
5. 返回详细的安装日志

支持的操作系统：
- Rocky Linux 9 / CentOS
- Ubuntu 22.04 / Debian

## API 端点

### 1. 单节点安装

**端点:** `POST /api/slurm/nodes/install`

**请求体:**
```json
{
  "node_name": "test-rocky01",
  "os_type": "rocky"
}
```

**参数说明:**
- `node_name` (必填): 节点名称（Docker 容器名或主机名）
- `os_type` (必填): 操作系统类型，支持值：`rocky`, `centos`, `ubuntu`, `debian`

**响应示例:**
```json
{
  "success": true,
  "message": "SLURM安装成功",
  "node": "test-rocky01",
  "logs": "[INFO] 在 test-rocky01 上安装SLURM包 (OS: rocky)\n..."
}
```

**curl 示例:**
```bash
curl -X POST http://localhost:8080/api/slurm/nodes/install \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "node_name": "test-rocky01",
    "os_type": "rocky"
  }'
```

### 2. 批量节点安装

**端点:** `POST /api/slurm/nodes/batch-install`

**请求体:**
```json
{
  "nodes": [
    {"node_name": "test-rocky01", "os_type": "rocky"},
    {"node_name": "test-rocky02", "os_type": "rocky"},
    {"node_name": "test-ssh01", "os_type": "ubuntu"}
  ]
}
```

**响应示例:**
```json
{
  "success": true,
  "total": 3,
  "success_count": 3,
  "failure_count": 0,
  "results": {
    "test-rocky01": {
      "success": true,
      "message": "SLURM安装成功",
      "logs": "..."
    },
    "test-rocky02": {
      "success": true,
      "message": "SLURM安装成功",
      "logs": "..."
    },
    "test-ssh01": {
      "success": true,
      "message": "SLURM安装成功",
      "logs": "..."
    }
  }
}
```

## 使用流程

### 方式一：使用测试脚本

我们提供了两个测试脚本方便使用：

#### 单节点安装测试

```bash
chmod +x test-install-node-api.sh
./test-install-node-api.sh test-rocky01 rocky
```

#### 批量节点安装测试

```bash
chmod +x test-batch-install-nodes.sh
./test-batch-install-nodes.sh
```

### 方式二：直接使用 API

1. **获取认证 Token**

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.data.token')
```

2. **安装单个节点**

```bash
curl -X POST http://localhost:8080/api/slurm/nodes/install \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "node_name": "test-rocky01",
    "os_type": "rocky"
  }' | jq '.'
```

3. **批量安装节点**

```bash
curl -X POST http://localhost:8080/api/slurm/nodes/batch-install \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "nodes": [
      {"node_name": "test-rocky01", "os_type": "rocky"},
      {"node_name": "test-rocky02", "os_type": "rocky"},
      {"node_name": "test-rocky03", "os_type": "rocky"},
      {"node_name": "test-ssh01", "os_type": "ubuntu"},
      {"node_name": "test-ssh02", "os_type": "ubuntu"},
      {"node_name": "test-ssh03", "os_type": "ubuntu"}
    ]
  }' | jq '.'
```

### 方式三：在前端页面中集成

在 SLURM 节点管理页面添加"安装 SLURM"按钮，调用安装 API：

```javascript
// 单节点安装
async function installSlurmNode(nodeName, osType) {
  const response = await fetch('/api/slurm/nodes/install', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`
    },
    body: JSON.stringify({
      node_name: nodeName,
      os_type: osType
    })
  });
  
  const result = await response.json();
  if (result.success) {
    console.log('安装成功:', result.message);
    console.log('日志:', result.logs);
  } else {
    console.error('安装失败:', result.error);
  }
}

// 批量安装
async function batchInstallNodes(nodes) {
  const response = await fetch('/api/slurm/nodes/batch-install', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`
    },
    body: JSON.stringify({ nodes })
  });
  
  const result = await response.json();
  console.log(`安装完成: 成功 ${result.success_count}, 失败 ${result.failure_count}`);
  return result.results;
}
```

## 安装后验证

### 1. 检查节点状态

```bash
docker exec ai-infra-slurm-master sinfo
```

输出应该类似：
```
PARTITION AVAIL  TIMELIMIT  NODES  STATE NODELIST
compute*     up   infinite      6   idle test-rocky[01-03],test-ssh[01-03]
```

### 2. 检查详细节点信息

```bash
docker exec ai-infra-slurm-master scontrol show nodes
```

### 3. 如果节点显示 DOWN

节点可能显示 `DOWN+NOT_RESPONDING` 状态，这是因为节点刚刚加入。手动恢复：

```bash
docker exec ai-infra-slurm-master scontrol update \
  nodename=test-rocky01,test-rocky02,test-rocky03,test-ssh01,test-ssh02,test-ssh03 \
  state=idle
```

### 4. 测试作业提交

提交一个简单的测试作业：

```bash
docker exec ai-infra-slurm-master sbatch --wrap="hostname && uptime"
```

检查作业状态：

```bash
docker exec ai-infra-slurm-master squeue
docker exec ai-infra-slurm-master sacct
```

## 安装过程详解

安装脚本执行以下步骤：

### 对于 Rocky Linux / CentOS 节点：

1. 安装 EPEL 仓库
2. 使用 `dnf` 安装 `slurm`, `slurm-slurmd`, `munge`
3. 创建必要的目录：
   - `/var/spool/slurm/slurmd`
   - `/var/log/slurm`
   - `/var/run/slurm`
   - `/etc/munge`
4. 从 slurm-master 复制 `/etc/slurm/slurm.conf`
5. 从 slurm-master 复制 `/etc/munge/munge.key`
6. 设置正确的文件权限
7. 启动 `munge` 服务
8. 启动 `slurmd` 服务

### 对于 Ubuntu / Debian 节点：

1. 使用 `apt-get` 安装 `slurm-client`, `slurmd`, `munge`
2. 创建必要的目录（同上，额外包括 `/etc/slurm-llnl`）
3. 从 slurm-master 复制配置到 `/etc/slurm-llnl/slurm.conf`
4. 复制和配置 munge.key
5. 启动服务

## 故障排除

### 问题 1: 节点保持 DOWN 状态

**原因:** slurmd 服务未正常启动或无法连接到 slurm-master

**解决方案:**
```bash
# 检查 slurmd 日志
docker exec test-rocky01 journalctl -u slurmd -n 50

# 手动启动 slurmd
docker exec test-rocky01 systemctl restart slurmd

# 检查 munge 状态
docker exec test-rocky01 systemctl status munge
```

### 问题 2: munge 认证失败

**原因:** munge.key 权限不正确或不一致

**解决方案:**
```bash
# 重新复制 munge.key
docker cp /tmp/munge.key test-rocky01:/etc/munge/munge.key
docker exec test-rocky01 chown munge:munge /etc/munge/munge.key
docker exec test-rocky01 chmod 400 /etc/munge/munge.key
docker exec test-rocky01 systemctl restart munge
docker exec test-rocky01 systemctl restart slurmd
```

### 问题 3: slurm.conf 配置不一致

**原因:** 节点名称或地址配置错误

**解决方案:**
检查 `/etc/slurm/slurm.conf` 中节点配置是否正确：
```bash
docker exec ai-infra-slurm-master cat /etc/slurm/slurm.conf | grep NodeName
```

确保每个节点都有对应的配置行。

### 问题 4: 包安装失败

**原因:** 网络问题或仓库不可用

**解决方案:**
```bash
# Rocky Linux - 检查 EPEL 仓库
docker exec test-rocky01 dnf repolist

# Ubuntu - 更新包列表
docker exec test-ssh01 apt-get update
```

## 参考文档

- [install-slurm-nodes.sh](./install-slurm-nodes.sh) - 原始安装脚本
- [test-install-node-api.sh](./test-install-node-api.sh) - 单节点安装测试
- [test-batch-install-nodes.sh](./test-batch-install-nodes.sh) - 批量安装测试
- [SLURM 官方文档](https://slurm.schedmd.com/)

## 下一步计划

1. **前端集成:** 在节点管理页面添加"安装 SLURM"按钮
2. **自动发现:** 自动检测新节点的操作系统类型
3. **状态监控:** 实时显示安装进度和状态
4. **回滚功能:** 支持卸载和回滚安装
5. **健康检查:** 安装后自动执行健康检查
6. **模板支持:** 支持自定义安装配置模板

## 总结

该自动化安装功能大大简化了 SLURM 集群节点的部署流程，从手动逐节点安装变为一键批量部署。通过 REST API 的方式，可以轻松集成到自动化运维流程和前端管理界面中。
