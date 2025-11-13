# SLURM 远程节点管理指南

## 概述

本指南介绍如何通过SSH远程管理SLURM计算节点，包括：
- 从SLURM Master远程初始化节点
- 同步SSH密钥实现免密登录
- 同步Munge密钥用于认证
- 远程启动和管理slurmd和munge服务

## 架构说明

```
┌─────────────────────┐
│  SLURM Master       │
│  (ai-infra-slurm-   │
│   master)           │
│                     │
│  - /etc/munge/      │
│    munge.key        │
│  - /etc/slurm/      │
│    slurm.conf       │
│  - /root/.ssh/      │
│    id_rsa.pub       │
└──────────┬──────────┘
           │ SSH
           ├───────────────────┐
           │                   │
           ▼                   ▼
┌─────────────────┐  ┌─────────────────┐
│ Compute Node 1  │  │ Compute Node 2  │
│ (test-rocky02)  │  │ (test-ssh02)    │
│                 │  │                 │
│ - munged        │  │ - munged        │
│ - slurmd        │  │ - slurmd        │
└─────────────────┘  └─────────────────┘
```

## 前置条件

1. **SLURM Master 运行正常**
   ```bash
   docker exec ai-infra-slurm-master sinfo
   ```

2. **计算节点容器已启动**
   ```bash
   docker ps | grep test-
   ```

3. **节点间网络互通**
   ```bash
   docker exec ai-infra-slurm-master ping -c 2 test-rocky02
   ```

## 快速开始

### 1. 检查节点状态

```bash
./scripts/manage-slurm-nodes.sh status test-rocky02 test-rocky03 test-ssh02 test-ssh03
```

预期输出：
```
=== test-rocky02 ===
root         561  0.0  0.0   3800  2620 ?        Ss   22:39   0:00 bash -c  ps aux|egrep 'slurm|munge'
root         568  0.0  0.0   3036  1420 ?        S    22:39   0:00 grep -E slurm|munge
```

如果没有看到 `munged` 和 `slurmd` 进程，说明服务未启动。

### 2. 完整初始化所有节点

```bash
./scripts/manage-slurm-nodes.sh init test-rocky02 test-rocky03 test-ssh02 test-ssh03
```

这个命令会自动执行：
1. ✅ 同步Munge密钥
2. ✅ 同步SLURM配置文件
3. ✅ 启动Munge服务
4. ✅ 启动SLURMD服务

### 3. 验证节点加入集群

```bash
docker exec ai-infra-slurm-master sinfo
```

预期输出：
```
PARTITION AVAIL  TIMELIMIT  NODES  STATE NODELIST
compute*     up   infinite      6  idle  test-rocky[01-03],test-ssh[01-03]
```

节点状态应该是 `idle`（空闲）而不是 `idle*`（未响应）。

## 使用场景

### 场景1: 新增计算节点

当添加新的计算节点到集群时：

```bash
# 1. 启动新节点容器
docker run -d --name test-new-node --network ai-infra-network \
  --hostname test-new-node ubuntu:20.04

# 2. 在Master的slurm.conf中添加节点配置
docker exec ai-infra-slurm-master bash -c "
cat >> /etc/slurm/slurm.conf <<EOF
NodeName=test-new-node NodeAddr=test-new-node CPUs=4 RealMemory=8192 State=UNKNOWN
EOF
scontrol reconfigure
"

# 3. 初始化新节点
./scripts/manage-slurm-nodes.sh init test-new-node

# 4. 验证
docker exec ai-infra-slurm-master sinfo -N -l | grep test-new-node
```

### 场景2: 重启后恢复节点

当节点重启后需要重新启动服务：

```bash
# 仅重启Munge和SLURMD服务
./scripts/manage-slurm-nodes.sh start-munge test-rocky02
./scripts/manage-slurm-nodes.sh start-slurmd test-rocky02
```

### 场景3: 更新SLURM配置

当Master的配置文件更新后，同步到所有节点：

```bash
# 1. 修改Master配置
docker exec ai-infra-slurm-master vi /etc/slurm/slurm.conf

# 2. 同步到所有节点
./scripts/manage-slurm-nodes.sh sync-conf test-rocky02 test-rocky03 test-ssh02 test-ssh03

# 3. 重启服务
./scripts/manage-slurm-nodes.sh start-slurmd test-rocky02 test-rocky03 test-ssh02 test-ssh03

# 4. Master重新加载配置
docker exec ai-infra-slurm-master scontrol reconfigure
```

### 场景4: 设置SSH免密登录

允许Master免密SSH到所有节点：

```bash
# 1. 同步SSH公钥到所有节点
./scripts/manage-slurm-nodes.sh sync-ssh test-rocky02 test-rocky03 test-ssh02 test-ssh03

# 2. 测试SSH连接
./scripts/manage-slurm-nodes.sh test-ssh test-rocky02
```

成功后，可以从Master直接SSH：
```bash
docker exec ai-infra-slurm-master ssh root@test-rocky02 hostname
```

## 后端API使用

### 1. 初始化单个节点

```bash
curl -X POST http://localhost:8082/api/slurm/clusters/1/init-node \
  -H "Content-Type: application/json" \
  -d '{
    "node_id": 123,
    "install_packages": true,
    "slurm_conf_path": "/etc/slurm/slurm.conf"
  }'
```

### 2. 同步SSH密钥

```bash
curl -X POST http://localhost:8082/api/slurm/clusters/1/sync-ssh-keys \
  -H "Content-Type: application/json" \
  -d '{
    "public_key_path": "/root/.ssh/id_rsa.pub"
  }'
```

### 3. 获取Munge密钥

```bash
curl http://localhost:8082/api/slurm/clusters/1/munge-key
```

响应：
```json
{
  "munge_key": "YWJjZGVmZ2hpams...",
  "encoding": "base64"
}
```

## 故障排查

### 问题1: Munge密钥权限错误

**症状**：
```
munged: Error: Failed to access "/etc/munge/munge.key"
```

**解决**：
```bash
docker exec test-rocky02 bash -c "
  chown munge:munge /etc/munge/munge.key
  chmod 400 /etc/munge/munge.key
"
```

### 问题2: SLURMD无法连接到Master

**症状**：
```
slurmd: error: Unable to connect to slurmctld
```

**检查**：
1. 确认Master可达：
   ```bash
   docker exec test-rocky02 ping ai-infra-slurm-master
   ```

2. 检查slurm.conf中的ControlMachine配置：
   ```bash
   docker exec test-rocky02 grep ControlMachine /etc/slurm/slurm.conf
   ```

3. 确认端口开放（6817）：
   ```bash
   docker exec ai-infra-slurm-master netstat -tlnp | grep 6817
   ```

### 问题3: 节点状态显示 idle*（带星号）

**症状**：
```
PARTITION AVAIL  TIMELIMIT  NODES  STATE NODELIST
compute*     up   infinite      1  idle* test-rocky02
```

**原因**：节点未响应slurmctld的健康检查

**解决**：
1. 检查slurmd是否运行：
   ```bash
   docker exec test-rocky02 ps aux | grep slurmd
   ```

2. 查看slurmd日志：
   ```bash
   docker exec test-rocky02 tail -f /var/log/slurm/slurmd.log
   ```

3. 重启slurmd：
   ```bash
   ./scripts/manage-slurm-nodes.sh start-slurmd test-rocky02
   ```

4. 在Master上更新节点状态：
   ```bash
   docker exec ai-infra-slurm-master scontrol update NodeName=test-rocky02 State=RESUME
   ```

### 问题4: Munge认证失败

**症状**：
```
slurmd: error: Munge encode failed
```

**检查**：
1. Munge密钥是否一致：
   ```bash
   # Master
   docker exec ai-infra-slurm-master md5sum /etc/munge/munge.key
   
   # Node
   docker exec test-rocky02 md5sum /etc/munge/munge.key
   ```

2. Munge服务是否运行：
   ```bash
   docker exec test-rocky02 ps aux | grep munged
   ```

3. 测试Munge：
   ```bash
   docker exec test-rocky02 munge -n | unmunge
   ```

**解决**：
```bash
# 重新同步密钥
./scripts/manage-slurm-nodes.sh sync-munge test-rocky02

# 重启Munge
./scripts/manage-slurm-nodes.sh start-munge test-rocky02
```

## 最佳实践

### 1. 定期备份密钥

```bash
# 备份Munge密钥
docker cp ai-infra-slurm-master:/etc/munge/munge.key ./backup/munge.key.$(date +%Y%m%d)

# 备份SSH密钥
docker cp ai-infra-slurm-master:/root/.ssh ./backup/ssh-keys.$(date +%Y%m%d)
```

### 2. 监控节点状态

创建监控脚本：
```bash
#!/bin/bash
# monitor-nodes.sh

while true; do
  echo "=== $(date) ==="
  docker exec ai-infra-slurm-master sinfo
  
  # 检查异常节点
  docker exec ai-infra-slurm-master sinfo -N -l | grep -E 'down|drain|idle\*'
  
  sleep 60
done
```

### 3. 自动化节点恢复

```bash
#!/bin/bash
# auto-recover-nodes.sh

# 获取所有idle*节点
down_nodes=$(docker exec ai-infra-slurm-master sinfo -h -N -o '%N %T' | grep 'idle\*' | awk '{print $1}')

for node in $down_nodes; do
  echo "恢复节点: $node"
  
  # 重启服务
  ./scripts/manage-slurm-nodes.sh start-munge "$node"
  ./scripts/manage-slurm-nodes.sh start-slurmd "$node"
  
  # 更新状态
  docker exec ai-infra-slurm-master scontrol update NodeName="$node" State=RESUME
done
```

## 高级配置

### 1. 使用SSH密钥认证（推荐）

在后端API中配置节点时使用密钥认证：

```json
{
  "node_name": "test-rocky02",
  "host": "192.168.3.100",
  "port": 22,
  "username": "root",
  "auth_type": "key",
  "key_path": "/root/.ssh/cluster_key"
}
```

### 2. 配置节点自动发现

修改Master配置启用动态节点：
```bash
docker exec ai-infra-slurm-master bash -c "
cat >> /etc/slurm/slurm.conf <<EOF

# 动态节点配置
NodeName=compute[001-100] NodeAddr=compute[001-100] CPUs=4 RealMemory=8192 State=CLOUD
EOF

scontrol reconfigure
"
```

### 3. 集成到CI/CD

在 `.github/workflows/deploy.yml` 中：
```yaml
- name: Initialize SLURM Nodes
  run: |
    ./scripts/manage-slurm-nodes.sh init \
      test-rocky02 test-rocky03 test-ssh02 test-ssh03
    
- name: Verify Cluster
  run: |
    docker exec ai-infra-slurm-master sinfo
    docker exec ai-infra-slurm-master squeue
```

## 相关文档

- [SLURM官方文档](https://slurm.schedmd.com/documentation.html)
- [Munge认证指南](https://github.com/dun/munge/wiki)
- [SSH免密登录配置](https://www.ssh.com/academy/ssh/copy-id)

## 总结

通过SSH远程管理SLURM节点的关键步骤：

1. ✅ **同步Munge密钥** - 确保所有节点使用相同的密钥
2. ✅ **同步SLURM配置** - 保持配置一致性
3. ✅ **启动服务** - munged → slurmd（顺序很重要）
4. ✅ **验证状态** - 使用sinfo检查节点状态

使用提供的脚本可以自动化这些步骤，简化集群管理工作。
