# SLURM 节点健康检测与自动修复功能

## 功能概述

新增智能节点健康检测功能，可自动检测并尝试修复异常的 SLURM 节点状态。当节点出现 `UNKNOWN`、`NOT_RESPONDING` 或 `DOWN` 状态时，系统会自动尝试修复，如果多次尝试失败则提供详细的诊断信息和修复建议。

## 功能特性

### 1. 自动健康检测
- ✅ 节点添加后自动执行健康检测
- ✅ 智能识别节点状态（IDLE、DOWN、NOT_RESPONDING、UNKNOWN）
- ✅ 自动尝试修复异常状态（最多3次）
- ✅ 渐进式重试机制（2秒、4秒、6秒间隔）

### 2. 多状态智能处理
| 节点状态 | 系统行为 | 修复操作 |
|---------|---------|---------|
| IDLE/ALLOCATED/MIXED | ✅ 正常 | 无需操作 |
| DOWN（可响应） | ⚠️ 尝试修复 | 执行 State=RESUME |
| NOT_RESPONDING | ⚠️ 尝试修复 | 设为 DOWN + 继续重试 |
| UNKNOWN | ⚠️ 尝试修复 | 设为 DOWN + 继续重试 |
| 修复失败 | ❌ 报告异常 | 提供诊断建议 |

### 3. 前端健康检测按钮
- ✅ 每个节点都有"健康检测"按钮
- ✅ 检测中显示加载动画
- ✅ 检测结果实时反馈（Toast 通知）
- ✅ 失败时显示诊断建议

## 后端实现

### 核心函数：DetectAndFixNodeState

**文件：**`src/backend/internal/services/slurm_service.go` (Line 617-688)

**功能：**
1. 检测节点状态
2. 根据状态执行相应的修复操作
3. 支持多次重试（可配置）
4. 返回详细的错误信息和修复建议

**检测逻辑：**
```go
func (s *SlurmService) DetectAndFixNodeState(ctx context.Context, nodeName string, maxRetries int) error {
    for attempt := 1; attempt <= maxRetries; attempt++ {
        // 1. 等待状态稳定
        time.Sleep(...)
        
        // 2. 检查节点状态
        output := scontrol show node
        
        // 3. 分析状态并执行相应操作
        if IDLE/ALLOCATED/MIXED {
            return nil  // 正常
        } else if DOWN (可响应) {
            scontrol update State=RESUME  // 激活节点
        } else if NOT_RESPONDING/UNKNOWN {
            scontrol update State=DOWN Reason="..."  // 标记为 DOWN
        }
        
        // 4. 继续下一次检测
    }
    
    // 5. 所有重试失败，返回错误
    return error with suggestions
}
```

**修改位置：**

1. **ScaleUp 函数** (Line 590-613)
   - 添加自动健康检测调用
   - 每个新节点都会自动检测

```go
// 将所有新节点设置为 DOWN 状态，并检测节点健康状态
for _, node := range nodes {
    if node.NodeType == "compute" || node.NodeType == "node" {
        // 设置为 DOWN
        downCmd := fmt.Sprintf("scontrol update NodeName=%s State=DOWN Reason=\"新添加节点，正在检测状态\"", node.NodeName)
        s.ExecuteSlurmCommand(ctx, downCmd)
        
        // 自动健康检测（最多3次）
        if err := s.DetectAndFixNodeState(ctx, node.NodeName, 3); err != nil {
            log.Printf("[ERROR] 节点 %s 健康检测失败: %v", node.NodeName, err)
        }
    }
}
```

### API 端点：健康检测接口

**文件：**`src/backend/internal/controllers/slurm_controller.go` (Line 773-830)

**端点：**`POST /api/slurm/nodes/health-check`

**请求参数：**
```json
{
  "node_name": "node01",
  "max_retries": 3
}
```

**成功响应（200）：**
```json
{
  "success": true,
  "node_name": "node01",
  "status": "healthy",
  "message": "节点健康检测通过，状态正常"
}
```

**失败响应（200）：**
```json
{
  "success": false,
  "node_name": "node01",
  "status": "unhealthy",
  "message": "节点健康检测失败",
  "details": "节点未响应或状态未知，可能原因：slurmd未运行、网络不可达或配置错误。请执行：1) 检查节点连接性 2) 确认slurmd服务状态 3) 手动执行: scontrol update NodeName=node01 State=RESUME",
  "suggestions": [
    "1. 检查节点网络连接: ping node01",
    "2. 确认slurmd服务状态: ssh node01 systemctl status slurmd",
    "3. 检查slurm配置: ssh node01 slurmd -C",
    "4. 手动激活节点: scontrol update NodeName=node01 State=RESUME"
  ]
}
```

**路由注册：**
```go
// src/backend/cmd/main.go (Line 945)
slurm.POST("/nodes/health-check", slurmController.HealthCheckNode)
```

## 前端实现

### 修改文件
`src/frontend/src/components/slurm/ClusterDetailsDialog.jsx`

### 1. 新增状态管理
```javascript
const [checkingNodes, setCheckingNodes] = useState(new Set());
```

### 2. 健康检测函数
```javascript
const handleHealthCheck = async (nodeName) => {
  setCheckingNodes(prev => new Set(prev).add(nodeName));
  
  try {
    const response = await api.post('/api/slurm/nodes/health-check', {
      node_name: nodeName,
      max_retries: 3,
    });

    if (response.data.success) {
      toast({
        title: '健康检测成功',
        description: `节点 ${nodeName} 状态正常`,
      });
      fetchClusterDetails(); // 刷新节点列表
    } else {
      toast({
        title: '健康检测失败',
        description: response.data.details,
        variant: 'destructive',
        action: /* 显示建议 */,
      });
    }
  } catch (error) {
    toast({
      title: '检测错误',
      description: error.response?.data?.error || '无法连接到服务器',
      variant: 'destructive',
    });
  } finally {
    setCheckingNodes(prev => {
      const newSet = new Set(prev);
      newSet.delete(nodeName);
      return newSet;
    });
  }
};
```

### 3. UI 组件
```jsx
<TableCell>
  <Button
    size="sm"
    variant="outline"
    onClick={() => handleHealthCheck(node.node_name)}
    disabled={checkingNodes.has(node.node_name)}
    className="flex items-center gap-2"
  >
    {checkingNodes.has(node.node_name) ? (
      <>
        <Loader2 className="h-3 w-3 animate-spin" />
        检测中...
      </>
    ) : (
      <>
        <Stethoscope className="h-3 w-3" />
        健康检测
      </>
    )}
  </Button>
</TableCell>
```

### 4. 新增图标导入
```javascript
import {
  Stethoscope,    // 健康检测图标
  AlertCircle,    // 警告图标
  Loader2,        // 加载动画
  // ... 其他图标
} from 'lucide-react';
```

## 使用场景

### 场景 1：自动检测（节点添加时）

```bash
# 1. 通过 Web 界面添加节点
# 2. 后端自动执行以下操作：

[DEBUG] 节点 node01 已设置为 DOWN 状态，开始健康检测...
[DEBUG] 开始检测节点 node01 的健康状态（最多重试 3 次）
[DEBUG] 第 1 次检测前等待 2s...
[INFO] 第 1/3 次：节点 node01 处于 DOWN 状态，尝试激活...
[INFO] 第 1/3 次：节点 node01 激活命令已执行
[DEBUG] 第 2 次检测前等待 4s...
[SUCCESS] 节点 node01 状态正常: IDLE/ALLOCATED/MIXED
[INFO] 所有新节点已完成初始化和健康检测
```

### 场景 2：手动检测（前端操作）

```bash
# 1. 用户在节点列表中点击"健康检测"按钮
# 2. 前端发送请求到后端
# 3. 后端执行检测和修复
# 4. 前端显示结果通知

✅ 成功：节点 node01 状态正常
或
❌ 失败：节点 node01 状态异常
   建议操作：
   1. 检查节点网络连接: ping node01
   2. 确认slurmd服务状态: ssh node01 systemctl status slurmd
```

### 场景 3：检测失败处理

**后端日志：**
```
[WARNING] 第 1/3 次：节点 node01 未响应或状态未知
[WARNING] 第 1/3 次：设置节点 node01 为 DOWN 失败: ...
[WARNING] 第 2/3 次：节点 node01 未响应或状态未知
[WARNING] 第 3/3 次：节点 node01 未响应或状态未知
[ERROR] 节点 node01 健康检测失败: 节点未响应或状态未知，可能原因：slurmd未运行、网络不可达或配置错误
```

**前端显示：**
```
❌ 健康检测失败
节点 node01 状态异常

详情：节点未响应或状态未知，可能原因：slurmd未运行...

建议操作：
1. 检查节点网络连接: ping node01
2. 确认slurmd服务状态: ssh node01 systemctl status slurmd
3. 检查slurm配置: ssh node01 slurmd -C
4. 手动激活节点: scontrol update NodeName=node01 State=RESUME
```

## 检测流程图

```
┌─────────────────────┐
│  添加节点/手动检测  │
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ 设置节点为 DOWN     │
│ Reason: 正在检测状态│
└──────────┬──────────┘
           │
           ▼
┌─────────────────────┐
│ 开始检测循环        │
│ (最多 N 次)         │
└──────────┬──────────┘
           │
           ▼
    ┌─────────────┐
    │ 等待稳定    │
    │ (2/4/6秒)   │
    └──────┬──────┘
           │
           ▼
    ┌─────────────────────┐
    │ 执行: scontrol show │
    └──────┬──────────────┘
           │
           ▼
    ┌──────────────────────┐
    │  分析节点状态        │
    └──────┬───────────────┘
           │
     ┌─────┴─────┐
     │           │
     ▼           ▼
┌─────────┐ ┌─────────────┐
│正常状态│ │  异常状态   │
│IDLE/    │ │DOWN/NOT_    │
│ALLOCATED│ │RESPONDING   │
└────┬────┘ └──────┬──────┘
     │             │
     │             ▼
     │      ┌─────────────┐
     │      │ 执行修复    │
     │      │State=RESUME │
     │      └──────┬──────┘
     │             │
     │             ▼
     │      ┌─────────────┐
     │      │ 继续下一次  │
     │      │   检测      │
     │      └──────┬──────┘
     │             │
     ▼             ▼
┌──────────────────────┐
│  返回检测结果        │
│  - 成功/失败        │
│  - 错误详情         │
│  - 修复建议         │
└──────────────────────┘
```

## 配置参数

### 后端配置

```go
// ScaleUp 中自动检测的重试次数
maxRetries := 3

// 检测间隔时间（渐进式）
waitTime := attempt * 2 * time.Second
// 第1次：2秒
// 第2次：4秒
// 第3次：6秒
```

### 前端配置

```javascript
// API 请求参数
{
  "node_name": nodeName,
  "max_retries": 3  // 可调整
}
```

## 错误诊断建议

当节点健康检测失败时，系统会提供以下诊断建议：

1. **检查网络连接**
   ```bash
   ping <节点名或IP>
   ```

2. **确认 slurmd 服务状态**
   ```bash
   ssh <节点> systemctl status slurmd
   # 如果未运行：
   ssh <节点> systemctl start slurmd
   ```

3. **检查 SLURM 配置**
   ```bash
   ssh <节点> slurmd -C
   # 查看节点硬件配置是否正确
   ```

4. **手动激活节点**
   ```bash
   scontrol update NodeName=<节点名> State=RESUME
   ```

5. **检查日志**
   ```bash
   # 节点上的 slurmd 日志
   ssh <节点> journalctl -u slurmd -n 50
   
   # 控制器上的 slurmctld 日志
   docker exec ai-infra-slurm-master journalctl -u slurmctld -n 50
   ```

## 最佳实践

### 1. 节点准备检查清单
- [ ] 确认节点网络可达
- [ ] 安装 slurmd 服务
- [ ] 配置 slurm.conf（与控制器一致）
- [ ] 启动 slurmd 服务
- [ ] 验证服务状态

### 2. 健康检测建议
- ✅ 添加节点后等待自动检测完成
- ✅ 定期对 DOWN 状态节点执行手动检测
- ✅ 关注检测失败的节点，根据建议排查问题
- ✅ 修复后再次执行检测验证

### 3. 监控告警
- 设置告警：检测失败超过 N 次
- 监控 DOWN 状态节点数量
- 定期审查健康检测日志

## 测试验证

### 1. 正常节点测试
```bash
# 添加一个正常的节点（slurmd已运行）
# 期望：自动检测成功，节点状态变为 IDLE
```

### 2. 未安装 slurmd 测试
```bash
# 添加一个未安装slurmd的节点
# 期望：检测失败，显示诊断建议
```

### 3. 手动检测测试
```bash
# 在前端点击"健康检测"按钮
# 期望：显示检测进度和结果
```

### 4. 修复测试
```bash
# 1. 添加节点（slurmd未启动）
# 2. 检测失败
# 3. 启动slurmd
# 4. 再次手动检测
# 期望：检测成功，节点状态正常
```

## 更新日期

2025年11月8日

## 相关文档

- [NODE_DOWN_STATE_ACTIVATION.md](./NODE_DOWN_STATE_ACTIVATION.md) - 节点 DOWN 状态激活说明
- [NODE_UNKNOWN_STATE_FIX.md](./NODE_UNKNOWN_STATE_FIX.md) - 节点 UNKNOWN 状态修复
