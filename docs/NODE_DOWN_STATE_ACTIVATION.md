# SLURM 节点默认 DOWN 状态说明

## 修改概述

从本版本开始，通过 Web 接口添加的 SLURM 节点将默认设置为 **DOWN 状态**，需要管理员手动激活后才能投入使用。

## 修改原因

1. **安全性**：避免未完全就绪的节点自动接受作业调度
2. **可控性**：给管理员更多的控制权，可以在激活前进行必要的检查和配置
3. **稳定性**：防止 slurmd 未运行的节点显示为 `unknown` 状态
4. **最佳实践**：符合生产环境中的节点管理规范

## 节点添加流程

### 1. 添加节点到集群

通过 Web 界面或 API 添加节点后，节点将：
- ✅ 被添加到 slurm.conf 配置文件
- ✅ 状态设置为 **DOWN**
- ✅ Reason 标注为："新添加节点，请确认slurmd正常运行后手动激活"

### 2. 确认节点就绪

在激活节点前，请确认：

```bash
# 检查节点是否可达
ping <节点IP>

# 检查 slurmd 服务状态
ssh <节点IP> systemctl status slurmd

# 或者在节点上直接检查
systemctl status slurmd

# 确认 slurmd 可以连接到 controller
ssh <节点IP> slurmd -C
```

### 3. 激活节点

确认节点就绪后，使用以下命令激活：

```bash
# 方式1：在 SLURM master 容器中执行
docker exec ai-infra-slurm-master scontrol update NodeName=<节点名> State=RESUME

# 方式2：如果已配置 scontrol 命令
scontrol update NodeName=<节点名> State=RESUME
```

### 4. 验证节点状态

```bash
# 查看所有节点状态
docker exec ai-infra-slurm-master sinfo

# 查看特定节点详情
docker exec ai-infra-slurm-master scontrol show node <节点名>
```

**期望状态：**
- `State=IDLE` - 节点空闲，可以接受作业
- `State=ALLOCATED` - 节点正在运行作业
- `State=MIXED` - 节点部分资源被使用

## API 响应变化

### ScaleUp 操作响应

添加节点后，API 将返回：

```json
{
  "operation_id": "xxx",
  "success": true,
  "results": [
    {
      "node_id": "192.168.1.10",
      "success": true,
      "message": "节点已添加到SLURM集群(状态: DOWN)，请确认slurmd运行后执行: scontrol update NodeName=192.168.1.10 State=RESUME"
    }
  ]
}
```

### 日志输出

后端日志中会看到：

```
[INFO] 节点 node01 已设置为 DOWN 状态，等待手动激活
[INFO] 所有新节点已添加到配置并设置为 DOWN 状态
[INFO] 请确认节点上 slurmd 服务正常运行后，使用以下命令激活节点：
[INFO] scontrol update NodeName=<节点名> State=RESUME
```

## 常见场景处理

### 场景 1：批量添加节点

```bash
# 批量激活多个节点
for node in node01 node02 node03; do
  docker exec ai-infra-slurm-master scontrol update NodeName=$node State=RESUME
done

# 验证
docker exec ai-infra-slurm-master sinfo
```

### 场景 2：节点未响应

如果激活后节点仍然 DOWN：

```bash
# 1. 检查节点状态和原因
docker exec ai-infra-slurm-master scontrol show node <节点名>

# 2. 常见原因
# - slurmd 服务未启动
# - 网络不可达
# - slurm.conf 配置不一致

# 3. 检查日志
ssh <节点IP> journalctl -u slurmd -n 50

# 4. 重启 slurmd
ssh <节点IP> systemctl restart slurmd

# 5. 再次激活
docker exec ai-infra-slurm-master scontrol update NodeName=<节点名> State=RESUME
```

### 场景 3：节点维护

如果需要对节点进行维护：

```bash
# 1. 设置为 DRAIN（排空）
scontrol update NodeName=<节点名> State=DRAIN Reason="系统维护"

# 2. 等待当前作业完成
squeue -w <节点名>

# 3. 进行维护...

# 4. 恢复节点
scontrol update NodeName=<节点名> State=RESUME
```

## 代码修改位置

### 后端修改

**文件：**`src/backend/internal/services/slurm_service.go`

**修改 1：ScaleUp 函数** (Line 590-605)
```go
// 将所有新节点设置为 DOWN 状态，等待用户手动激活
for _, node := range nodes {
    if node.NodeType == "compute" || node.NodeType == "node" {
        downCmd := fmt.Sprintf("scontrol update NodeName=%s State=DOWN Reason=\"新添加节点，请确认slurmd正常运行后手动激活\"", node.NodeName)
        s.ExecuteSlurmCommand(ctx, downCmd)
    }
}
```

**修改 2：ScaleUpViaAPI 函数** (Line 1701-1708)
```go
nodeSpec := SlurmNodeSpec{
    NodeName: node.Host,
    CPUs:     cpus,
    Memory:   memory,
    Features: features,
    State:    "DOWN", // 新节点初始状态设置为 DOWN
}
```

## 与旧版本的区别

### 旧版本行为
- ✅ 节点添加后自动尝试激活（State=RESUME）
- ⚠️ 如果 slurmd 未运行，显示为 `unknown` 状态
- ⚠️ 需要手动设置为 DOWN

### 新版本行为
- ✅ 节点添加后默认为 DOWN 状态
- ✅ 明确标注需要手动激活
- ✅ 避免 unknown 状态
- ✅ 更符合生产环境管理规范

## FAQ

### Q1: 为什么不自动激活节点？

**A:** 自动激活可能导致问题：
1. 节点上 slurmd 未安装或未运行
2. 节点配置不完整（如缺少必要的软件包）
3. 节点网络配置有问题
4. 节点需要先进行安全审计或性能测试

手动激活让管理员有机会在节点投入使用前进行必要的检查。

### Q2: 如何查看哪些节点需要激活？

**A:** 使用以下命令：

```bash
# 查看 DOWN 状态的节点
docker exec ai-infra-slurm-master sinfo -t down

# 查看节点的 Reason（原因）
docker exec ai-infra-slurm-master scontrol show node | grep -A 5 "State=DOWN"
```

### Q3: 可以批量激活所有 DOWN 节点吗？

**A:** 可以，但不推荐盲目批量激活。建议逐个检查后激活：

```bash
# 获取所有 DOWN 节点列表
nodes=$(docker exec ai-infra-slurm-master sinfo -t down -h -o "%N")

# 逐个检查并激活
for node in $nodes; do
  echo "检查节点: $node"
  # 这里可以添加检查逻辑
  # 确认后激活
  docker exec ai-infra-slurm-master scontrol update NodeName=$node State=RESUME
done
```

### Q4: 旧集群升级后，现有节点会受影响吗？

**A:** 不会。此修改只影响新添加的节点，现有节点的状态保持不变。

### Q5: 如何在前端显示激活提示？

**A:** 前端会显示返回消息中的激活命令。管理员可以直接复制命令到终端执行。

## 最佳实践建议

1. **标准化节点准备流程**
   - 创建节点初始化脚本
   - 包含 slurmd 安装、配置、启动
   - 在脚本中验证服务状态

2. **使用 Salt/Ansible 自动化**
   - 通过配置管理工具确保节点一致性
   - 自动化 slurmd 安装和配置

3. **建立监控告警**
   - 监控 DOWN 状态节点数量
   - 节点长时间 DOWN 时发送告警
   - 监控 slurmd 服务状态

4. **文档化流程**
   - 记录标准的节点激活检查清单
   - 培训管理员使用 scontrol 命令
   - 建立节点问题排查指南

## 相关命令速查

```bash
# 查看节点状态
sinfo
sinfo -N -l                    # 详细信息
sinfo -t down                  # 只看 DOWN 节点

# 节点操作
scontrol show node <节点名>    # 查看节点详情
scontrol update NodeName=<节点名> State=RESUME    # 激活节点
scontrol update NodeName=<节点名> State=DOWN Reason="维护"  # 设为 DOWN
scontrol update NodeName=<节点名> State=DRAIN Reason="排空" # 排空节点

# 批量操作
scontrol update NodeName=node[01-10] State=RESUME  # 激活 node01-node10
```

## 更新日期

2025年11月8日

## 相关文档

- [NODE_UNKNOWN_STATE_FIX.md](./NODE_UNKNOWN_STATE_FIX.md) - 节点 unknown 状态问题修复
- [SLURM 官方文档 - scontrol](https://slurm.schedmd.com/scontrol.html)
