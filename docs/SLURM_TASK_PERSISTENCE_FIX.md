# SLURM 任务统计修复报告

## 问题描述

用户反馈 http://192.168.18.154:8080/slurm-tasks 中的统计信息不准确，任务信息没有正确持久化到数据库。

### 问题根因

1. **ScaleUpAsync 方法缺少数据库持久化**
   - 只使用内存中的 ProgressManager 跟踪任务
   - 没有调用 `taskSvc.CreateTask()` 创建数据库记录
   - 任务完成后没有更新数据库状态

2. **统计查询来源不匹配**
   - `GetTaskStatistics` 从数据库查询统计
   - 但实际任务只存在于内存 ProgressManager 中
   - 导致统计信息不完整

## 解决方案

### 1. 修改 ScaleUpAsync 方法

**文件:** `src/backend/internal/controllers/slurm_controller.go`

#### 添加的功能：

1. **创建数据库任务记录**
```go
// 获取当前用户信息
userID, exists := middleware.GetCurrentUserID(ctx)
if !exists {
    userID = 0 // 默认系统用户
}

// 构建目标节点列表
targetNodes := make([]string, len(req.Nodes))
for i, node := range req.Nodes {
    targetNodes[i] = node.Host
}

// 创建数据库任务记录
taskReq := services.CreateTaskRequest{
    Name:        "SLURM集群扩容",
    Type:        "scale_up",
    UserID:      userID,
    TargetNodes: targetNodes,
    Parameters: map[string]interface{}{
        "nodes":       req.Nodes,
        "node_count":  len(req.Nodes),
        "salt_master": getSaltStackMasterHost(),
        "apphub_url":  getAppHubBaseURL(),
    },
    Tags:     []string{"scale-up", "slurm"},
    Priority: 5, // 普通优先级
}

dbTask, err := c.taskSvc.CreateTask(ctx.Request.Context(), taskReq)
if err != nil {
    ctx.JSON(http.StatusInternalServerError, gin.H{"error": "创建任务记录失败: " + err.Error()})
    return
}

taskID := dbTask.TaskID
```

2. **启动任务**
```go
// 启动任务
if err := c.taskSvc.StartTask(bgCtx, dbTaskID); err != nil {
    fmt.Printf("启动任务失败: %v\n", err)
}
```

3. **更新任务进度**
```go
// 在各个步骤中更新进度
c.taskSvc.UpdateTaskProgress(bgCtx, dbTaskID, progress, currentStep)

// 示例：
c.taskSvc.UpdateTaskProgress(bgCtx, dbTaskID, 60, "添加节点到集群数据库")
c.taskSvc.UpdateTaskProgress(bgCtx, dbTaskID, 80, "执行SLURM扩容")
c.taskSvc.UpdateTaskProgress(bgCtx, dbTaskID, 100, "扩容完成")
```

4. **记录任务事件**
```go
// 记录成功事件
c.taskSvc.AddTaskEvent(bgCtx, dbTaskID, "success", "deploy-minion", 
    "Minion部署成功", host, progress, nil)

// 记录错误事件
c.taskSvc.AddTaskEvent(bgCtx, dbTaskID, "error", "deploy-minion", 
    "Minion部署失败: "+result.Error, host, progress, nil)
```

5. **完成任务**
```go
defer func() {
    // 完成内存任务
    pm.Complete(opID, failed, "扩容完成")
    
    // 更新数据库任务状态
    status := "completed"
    if failed {
        status = "failed"
    }
    if err := c.taskSvc.UpdateTaskStatus(bgCtx, dbTaskID, status, finalError); err != nil {
        fmt.Printf("更新任务状态失败: %v\n", err)
    }
}()
```

6. **返回任务ID**
```go
ctx.JSON(http.StatusAccepted, gin.H{
    "opId":   op.ID,      // ProgressManager ID
    "taskId": taskID,      // 数据库任务ID
})
```

### 2. 添加必要的导入

```go
import (
    // ... 其他导入 ...
    "github.com/aresnasa/ai-infra-matrix/src/backend/internal/middleware"
)
```

## 数据流程

### 修复前：
```
ScaleUpAsync
  ↓
ProgressManager (内存)
  ↓
GetTaskStatistics → 数据库 (空) → 统计不准确 ❌
```

### 修复后：
```
ScaleUpAsync
  ↓
  ├─→ ProgressManager (内存) → 实时进度
  └─→ taskSvc.CreateTask() → 数据库
        ↓
        ├─→ taskSvc.StartTask()
        ├─→ taskSvc.UpdateTaskProgress()
        ├─→ taskSvc.AddTaskEvent()
        └─→ taskSvc.UpdateTaskStatus()
  ↓
GetTaskStatistics → 数据库 → 统计准确 ✅
```

## 持久化的信息

### 任务基本信息
- **name**: "SLURM集群扩容"
- **type**: "scale_up"
- **user_id**: 当前用户ID
- **target_nodes**: 目标节点列表 ["192.168.0.100", ...]
- **status**: pending → running → completed/failed
- **progress**: 0 → 100
- **created_at**: 任务创建时间
- **started_at**: 任务开始时间
- **completed_at**: 任务完成时间

### 任务参数 (parameters)
```json
{
  "nodes": [
    {
      "host": "192.168.0.100",
      "port": 22,
      "user": "root",
      "password": "***"
    }
  ],
  "node_count": 1,
  "salt_master": "192.168.0.50",
  "apphub_url": "http://apphub:8081"
}
```

### 任务事件 (events)
- 每个关键步骤都记录事件
- 包含：事件类型、步骤名、消息、主机、进度、数据
- 支持详细的任务追踪和问题排查

### 任务统计 (statistics)
- 自动计算成功率
- 执行时长
- 事件类型统计
- 步骤统计

## 测试验证

### 测试脚本
```bash
./scripts/test-slurm-task-persistence.sh
```

### 测试步骤
1. ✅ 检查后端服务状态
2. ✅ 获取当前任务统计
3. ✅ 创建测试扩容任务
4. ✅ 验证任务已记录到数据库
5. ✅ 检查任务参数完整性
6. ✅ 检查任务事件记录
7. ✅ 验证统计信息更新
8. ✅ 检查任务列表

### 预期结果
- 任务立即持久化到数据库
- 统计信息实时准确
- 任务详情包含完整参数
- 事件记录完整可追溯

## API 响应变化

### ScaleUpAsync 响应

**修复前：**
```json
{
  "opId": "random-uuid"
}
```

**修复后：**
```json
{
  "opId": "random-uuid",
  "taskId": "database-task-uuid"
}
```

### 统计信息查询

**GET /api/slurm/tasks/statistics**

现在能正确返回：
- `total_tasks`: 包含所有数据库任务
- `running_tasks`: 正在运行的任务数
- `completed_tasks`: 已完成的任务数
- `failed_tasks`: 失败的任务数
- `type_stats`: 按类型统计 {"scale_up": 5, "scale_down": 2}
- `success_rate`: 成功率百分比

## 影响范围

### 受益功能
1. ✅ 任务统计页面 (`/slurm-tasks` 的统计标签)
2. ✅ 任务列表显示
3. ✅ 任务详情查询
4. ✅ 任务历史记录
5. ✅ 用户操作审计

### 不受影响
- ProgressManager 实时进度跟踪
- 现有的任务查询 API
- 前端任务展示逻辑

## 后续优化建议

1. **其他异步操作的持久化**
   - ScaleDown 操作
   - 节点初始化操作
   - 集群配置操作

2. **任务清理策略**
   - 定期清理旧任务记录
   - 保留重要任务的归档

3. **性能优化**
   - 批量事件写入
   - 异步统计计算
   - 缓存热点数据

4. **监控告警**
   - 任务失败率告警
   - 长时间运行任务提醒
   - 数据库持久化异常监控

## 相关文件

- `src/backend/internal/controllers/slurm_controller.go` - ScaleUpAsync 方法
- `src/backend/internal/services/slurm_task_service.go` - 任务服务
- `src/backend/internal/models/slurm_task.go` - 任务模型
- `scripts/test-slurm-task-persistence.sh` - 测试脚本
- `docs/SLURM_TASK_PERSISTENCE_FIX.md` - 本文档

## 总结

通过在 `ScaleUpAsync` 方法中添加完整的数据库持久化逻辑，现在所有扩容任务都会：

1. ✅ 立即创建数据库记录
2. ✅ 实时更新任务状态和进度
3. ✅ 记录详细的执行事件
4. ✅ 提供准确的统计信息
5. ✅ 支持完整的任务追溯

这确保了 `/slurm-tasks` 页面能够显示准确的统计信息，并且所有任务数据都能持久化存储，支持历史查询和审计需求。
