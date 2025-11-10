# SLURM 节点自动安装功能实现总结

## 问题描述

原始问题：执行 `install-slurm-nodes.sh` 脚本时，Ubuntu 节点出现配置文件路径错误：
```
Error response from daemon: Could not find the file /etc/slurm-llnl in container test-ssh01
```

## 解决方案

### 1. 修复配置文件路径问题

**文件:** `install-slurm-nodes.sh`

**修改:**
- 在复制配置文件前，先创建必要的目录：`/etc/slurm-llnl` 和 `/etc/munge`
- 确保 Ubuntu 和 Rocky Linux 使用正确的配置路径
  - Rocky/CentOS: `/etc/slurm/slurm.conf`
  - Ubuntu/Debian: `/etc/slurm-llnl/slurm.conf`

```bash
# 确保目录存在
docker exec $node mkdir -p /etc/slurm-llnl /etc/munge
```

### 2. 后端服务集成

**文件:** `src/backend/internal/services/slurm_service.go`

**新增功能:**
- `InstallSlurmNode()` - 单节点安装
- `BatchInstallSlurmNodes()` - 批量节点安装
- `getSlurmMasterConfig()` - 获取 slurm.conf
- `getMungeKey()` - 获取 munge.key
- `installSlurmPackages()` - 安装 SLURM 包
- `configureSlurmNode()` - 配置节点
- `startSlurmServices()` - 启动服务

**核心逻辑:**
```go
1. 从 slurm-master 获取 slurm.conf 和 munge.key
2. 根据 OS 类型安装 SLURM 包
   - Rocky/CentOS: dnf install slurm slurm-slurmd munge
   - Ubuntu/Debian: apt-get install slurm-client slurmd munge
3. 复制配置文件到正确路径
4. 启动 munge 和 slurmd 服务
5. 返回详细的安装日志
```

### 3. API 端点

**文件:** `src/backend/internal/controllers/slurm_controller.go` 和 `src/backend/cmd/main.go`

**新增路由:**
- `POST /api/slurm/nodes/install` - 单节点安装
- `POST /api/slurm/nodes/batch-install` - 批量节点安装

**请求格式:**
```json
// 单节点
{
  "node_name": "test-rocky01",
  "os_type": "rocky"
}

// 批量
{
  "nodes": [
    {"node_name": "test-rocky01", "os_type": "rocky"},
    {"node_name": "test-ssh01", "os_type": "ubuntu"}
  ]
}
```

### 4. 测试工具

**创建的测试脚本:**
1. `test-install-node-api.sh` - 单节点安装测试
2. `test-batch-install-nodes.sh` - 批量安装测试

**使用方法:**
```bash
# 单节点安装
chmod +x test-install-node-api.sh
./test-install-node-api.sh test-rocky01 rocky

# 批量安装
chmod +x test-batch-install-nodes.sh
./test-batch-install-nodes.sh
```

### 5. 文档

创建完整文档：`docs/SLURM_NODE_AUTO_INSTALL.md`

包含内容：
- 功能概述和支持的操作系统
- API 端点详细说明
- 使用流程（脚本、API、前端集成）
- 安装过程详解
- 故障排除指南
- 下一步计划

## 技术亮点

1. **跨平台支持:** 自动识别并处理 Rocky Linux 和 Ubuntu 的差异
2. **配置同步:** 自动从 slurm-master 获取最新配置
3. **错误处理:** 完整的错误处理和日志记录
4. **批量操作:** 支持一次安装多个节点
5. **RESTful API:** 标准的 REST API 设计，易于集成
6. **可观测性:** 返回详细的安装日志，便于调试

## 文件清单

```
修改的文件:
- install-slurm-nodes.sh (修复路径问题)
- src/backend/internal/services/slurm_service.go (新增安装功能)
- src/backend/internal/controllers/slurm_controller.go (新增 API 端点)
- src/backend/cmd/main.go (注册路由)

新创建的文件:
- test-install-node-api.sh (单节点测试脚本)
- test-batch-install-nodes.sh (批量测试脚本)
- docs/SLURM_NODE_AUTO_INSTALL.md (完整文档)
- docs/SLURM_NODE_AUTO_INSTALL_SUMMARY.md (本总结文档)
```

## 使用示例

### 通过 API 安装节点

```bash
# 1. 获取 token
TOKEN=$(curl -s -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.data.token')

# 2. 安装单个节点
curl -X POST http://localhost:8080/api/slurm/nodes/install \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "node_name": "test-rocky01",
    "os_type": "rocky"
  }' | jq '.'

# 3. 批量安装
curl -X POST http://localhost:8080/api/slurm/nodes/batch-install \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "nodes": [
      {"node_name": "test-rocky01", "os_type": "rocky"},
      {"node_name": "test-rocky02", "os_type": "rocky"},
      {"node_name": "test-ssh01", "os_type": "ubuntu"}
    ]
  }' | jq '.'
```

### 在前端页面集成

```javascript
// 安装节点按钮点击处理
async function handleInstallNode(nodeName, osType) {
  try {
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
      showSuccessMessage(`节点 ${nodeName} 安装成功`);
      console.log('安装日志:', result.logs);
      // 刷新节点列表
      refreshNodeList();
    } else {
      showErrorMessage(`安装失败: ${result.error}`);
    }
  } catch (error) {
    showErrorMessage(`请求失败: ${error.message}`);
  }
}
```

## 验证步骤

1. **检查节点状态:**
   ```bash
   docker exec ai-infra-slurm-master sinfo
   ```

2. **查看详细信息:**
   ```bash
   docker exec ai-infra-slurm-master scontrol show nodes
   ```

3. **恢复 DOWN 节点:**
   ```bash
   docker exec ai-infra-slurm-master scontrol update \
     nodename=test-rocky01,test-rocky02,test-rocky03,test-ssh01,test-ssh02,test-ssh03 \
     state=idle
   ```

4. **提交测试作业:**
   ```bash
   docker exec ai-infra-slurm-master sbatch --wrap="hostname && uptime"
   docker exec ai-infra-slurm-master squeue
   ```

## 下一步建议

1. **前端 UI 集成:**
   - 在节点管理页面添加"安装 SLURM"按钮
   - 显示安装进度和日志
   - 支持批量选择和安装

2. **自动化增强:**
   - 自动检测节点操作系统类型
   - 安装后自动恢复节点状态
   - 集成健康检查

3. **监控和告警:**
   - 实时监控安装进度
   - 失败时发送告警通知
   - 记录安装历史

4. **配置模板:**
   - 支持自定义安装配置模板
   - 不同节点类型使用不同配置
   - 版本管理

## 结论

成功实现了 SLURM 节点的自动化安装功能，从手动脚本升级为完整的 REST API 服务。该功能：

- ✅ 修复了原始脚本的路径问题
- ✅ 支持跨平台（Rocky Linux 和 Ubuntu）
- ✅ 提供完整的 API 接口
- ✅ 包含详细的测试工具和文档
- ✅ 易于集成到前端页面
- ✅ 支持批量操作

现在可以通过 API 或前端页面轻松为 SLURM 集群添加新的计算节点，大大简化了运维工作。
