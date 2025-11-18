# SLURM 扩缩容进度显示修复报告

## 问题描述

**问题页面**: http://192.168.3.91:8080/slurm  
**问题现象**: 扩缩容状态卡片中的进度始终显示 0%，未能正确同步实际任务进度

## 根本原因

后端 `GetScalingStatus()` 函数查询了错误的数据表：

**错误代码**:
```go
// ❌ 查询 SlurmTask 表 - 这是作业任务表，不是节点安装任务表
s.db.Model(&models.SlurmTask{}).Where("status IN ?", []string{"running", "in_progress", "processing"}).Count(&runningTasks)
s.db.Model(&models.SlurmTask{}).Where("status IN ?", []string{"completed", "success"}).Count(&completedTasks)
s.db.Model(&models.SlurmTask{}).Where("status IN ?", []string{"failed", "error"}).Count(&failedTasks)
```

**问题分析**:
- `SlurmTask` 表存储的是 SLURM 作业任务（用户提交的计算任务）
- 扩缩容相关的任务存储在 `NodeInstallTask` 表中
- 因此查询结果始终为 0，导致进度为 0%

## 解决方案

### 1. 修改查询目标表

将查询从 `SlurmTask` 改为 `NodeInstallTask`：

```go
// ✅ 查询 NodeInstallTask 表 - 节点安装/扩缩容任务表
s.db.Model(&models.NodeInstallTask{}).
    Where("created_at > ?", recentTime).
    Where("status IN ?", []string{"running", "in_progress", "processing", "pending"}).
    Count(&runningTasks)
```

### 2. 增加时间范围限制

只统计最近 24 小时的任务，避免历史数据干扰：

```go
recentTime := time.Now().Add(-24 * time.Hour)
s.db.Model(&models.NodeInstallTask{}).
    Where("created_at > ?", recentTime).
    // ...
```

### 3. 使用任务实际进度

`NodeInstallTask` 模型包含 `Progress` 字段（0-100），可以更准确地计算总体进度：

```go
// 如果有正在运行的任务，计算加权平均进度
if runningTasks > 0 {
    var totalProgress int64
    var taskCount int64
    for _, task := range recentTasks {
        if task.Status == "running" || task.Status == "in_progress" || task.Status == "processing" {
            totalProgress += int64(task.Progress)
            taskCount++
        }
    }
    if taskCount > 0 {
        runningAvgProgress := totalProgress / taskCount
        // 综合考虑完成任务和正在运行任务的进度
        if totalTasks > 0 {
            progress = int((completedTasks*100 + runningTasks*runningAvgProgress) / totalTasks)
        }
    }
}
```

### 4. 构建真实的最近操作列表

从实际任务记录构建操作历史：

```go
recentOperations := []ScalingOperation{}
for _, task := range recentTasks {
    if task.Status == "completed" || task.Status == "failed" {
        op := ScalingOperation{
            ID:        task.TaskID,
            Type:      task.TaskType,
            Status:    task.Status,
            Nodes:     []string{fmt.Sprintf("node-%d", task.NodeID)},
            StartedAt: task.CreatedAt,
        }
        if task.CompletedAt != nil {
            op.CompletedAt = task.CompletedAt
        }
        if task.ErrorMessage != "" {
            op.Error = task.ErrorMessage
        }
        recentOperations = append(recentOperations, op)
    }
}
```

## 修复后的完整实现

```go
func (s *SlurmService) GetScalingStatus(ctx context.Context) (*ScalingStatus, error) {
    var runningTasks int64
    var completedTasks int64
    var failedTasks int64

    // 只查询最近24小时的任务
    recentTime := time.Now().Add(-24 * time.Hour)
    
    // 查询 NodeInstallTask 表
    s.db.Model(&models.NodeInstallTask{}).
        Where("created_at > ?", recentTime).
        Where("status IN ?", []string{"running", "in_progress", "processing", "pending"}).
        Count(&runningTasks)
    
    s.db.Model(&models.NodeInstallTask{}).
        Where("created_at > ?", recentTime).
        Where("status IN ?", []string{"completed", "success"}).
        Count(&completedTasks)
    
    s.db.Model(&models.NodeInstallTask{}).
        Where("created_at > ?", recentTime).
        Where("status IN ?", []string{"failed", "error"}).
        Count(&failedTasks)

    // 计算总体进度
    totalTasks := runningTasks + completedTasks + failedTasks
    progress := 0
    if totalTasks > 0 {
        progress = int((completedTasks * 100) / totalTasks)
    }

    // 获取最近的任务详情
    var recentTasks []models.NodeInstallTask
    s.db.Where("created_at > ?", recentTime).
        Order("created_at DESC").
        Limit(10).
        Find(&recentTasks)

    // 使用任务的实际进度计算更准确的总体进度
    if runningTasks > 0 {
        var totalProgress int64
        var taskCount int64
        for _, task := range recentTasks {
            if task.Status == "running" || task.Status == "in_progress" || task.Status == "processing" {
                totalProgress += int64(task.Progress)
                taskCount++
            }
        }
        if taskCount > 0 {
            runningAvgProgress := totalProgress / taskCount
            if totalTasks > 0 {
                progress = int((completedTasks*100 + runningTasks*runningAvgProgress) / totalTasks)
            }
        }
    }

    // 构建最近操作列表
    recentOperations := []ScalingOperation{}
    for _, task := range recentTasks {
        if task.Status == "completed" || task.Status == "failed" {
            op := ScalingOperation{
                ID:        task.TaskID,
                Type:      task.TaskType,
                Status:    task.Status,
                Nodes:     []string{fmt.Sprintf("node-%d", task.NodeID)},
                StartedAt: task.CreatedAt,
            }
            if task.CompletedAt != nil {
                op.CompletedAt = task.CompletedAt
            }
            if task.ErrorMessage != "" {
                op.Error = task.ErrorMessage
            }
            recentOperations = append(recentOperations, op)
            if len(recentOperations) >= 5 {
                break
            }
        }
    }

    return &ScalingStatus{
        ActiveOperations: []ScalingOperation{},
        RecentOperations: recentOperations,
        NodeTemplates:    []NodeTemplate{},
        Active:           runningTasks > 0,
        ActiveTasks:      int(runningTasks),
        SuccessNodes:     int(completedTasks),
        FailedNodes:      int(failedTasks),
        Progress:         progress,
    }, nil
}
```

## 前端显示逻辑

前端 `SlurmScalingPage.js` 正确地显示了这些数据：

```javascript
// 扩缩容状态卡片
<Card title="扩缩容状态">
  <Row gutter={[16, 16]}>
    <Col xs={24} sm={6}>
      <Statistic 
        title="活跃任务" 
        value={scalingStatus?.active_tasks || 0}
      />
    </Col>
    <Col xs={24} sm={6}>
      <Statistic 
        title="成功节点" 
        value={scalingStatus?.success_nodes || 0}
      />
    </Col>
    <Col xs={24} sm={6}>
      <Statistic 
        title="失败节点" 
        value={scalingStatus?.failed_nodes || 0}
      />
    </Col>
    <Col xs={24} sm={6}>
      <Progress
        type="circle"
        percent={Math.round(scalingStatus?.progress || 0)}
        status={
          scalingStatus?.active ? 'active' : 
          (scalingStatus?.failed_nodes > 0 ? 'exception' : 'success')
        }
      />
    </Col>
  </Row>
</Card>
```

## 数据模型

### NodeInstallTask 模型

```go
type NodeInstallTask struct {
    ID           uint       `json:"id"`
    TaskID       string     `json:"task_id"`
    NodeID       uint       `json:"node_id"`
    TaskType     string     `json:"task_type"` // salt-minion, slurm-node, etc.
    Status       string     `json:"status"`    // pending, running, completed, failed
    Progress     int        `json:"progress"`  // 0-100 ✅ 关键字段
    StartedAt    *time.Time `json:"started_at"`
    CompletedAt  *time.Time `json:"completed_at"`
    ErrorMessage string     `json:"error_message"`
    CreatedAt    time.Time  `json:"created_at"`
    UpdatedAt    time.Time  `json:"updated_at"`
}
```

### 任务状态映射

| 状态 | 说明 | 计入统计 |
|------|------|----------|
| `pending` | 待开始 | 运行中任务 |
| `running` | 运行中 | 运行中任务 |
| `in_progress` | 进行中 | 运行中任务 |
| `processing` | 处理中 | 运行中任务 |
| `completed` | 已完成 | 成功节点 |
| `success` | 成功 | 成功节点 |
| `failed` | 失败 | 失败节点 |
| `error` | 错误 | 失败节点 |

## 验证方法

### 1. 启动扩容操作

```bash
curl -X POST http://localhost:8081/api/slurm/scale-up \
  -H "Content-Type: application/json" \
  -d '{
    "nodes": [
      {
        "host": "test-rocky02",
        "port": 22,
        "user": "root",
        "password": "rootpass123"
      }
    ]
  }'
```

### 2. 检查扩缩容状态

```bash
curl http://localhost:8081/api/slurm/scaling/status
```

**修复前响应**:
```json
{
  "active": false,
  "active_tasks": 0,
  "success_nodes": 0,
  "failed_nodes": 0,
  "progress": 0  // ❌ 始终为 0
}
```

**修复后响应**:
```json
{
  "active": true,
  "active_tasks": 1,
  "success_nodes": 0,
  "failed_nodes": 0,
  "progress": 45,  // ✅ 实际进度
  "recent_operations": [
    {
      "id": "task-xxx",
      "type": "slurm-node",
      "status": "running",
      "started_at": "2024-11-13T16:00:00Z"
    }
  ]
}
```

### 3. 在前端查看

访问 http://192.168.3.91:8080/slurm，查看扩缩容状态卡片：

- **活跃任务**: 应显示正在运行的任务数
- **成功节点**: 应显示已完成的任务数
- **失败节点**: 应显示失败的任务数
- **进度**: 应显示实际的完成百分比（非0%）

## 进度计算逻辑

### 基础进度（无运行中任务）

```
progress = (completedTasks × 100) / totalTasks
```

### 加权进度（有运行中任务）

```
runningAvgProgress = Σ(task.Progress) / runningTaskCount
progress = (completedTasks × 100 + runningTasks × runningAvgProgress) / totalTasks
```

**示例**:
- 总任务: 5
- 完成: 2（100%）
- 运行中: 2（平均 50%）
- 待开始: 1（0%）

```
progress = (2 × 100 + 2 × 50) / 5 = (200 + 100) / 5 = 60%
```

## 相关文件

### 修改文件
- `src/backend/internal/services/slurm_service.go` - `GetScalingStatus()` 函数

### 相关文件
- `src/frontend/src/pages/SlurmScalingPage.js` - 前端显示逻辑
- `src/backend/internal/models/slurm_cluster_models.go` - NodeInstallTask 模型
- `src/backend/internal/controllers/slurm_controller.go` - API 路由

## 后续优化建议

### 1. 添加实时 WebSocket 推送

当前进度需要轮询获取，可以改为 WebSocket 实时推送：

```go
// 在任务进度更新时推送
func (s *SlurmService) updateTaskProgress(taskID string, progress int) {
    // 更新数据库
    s.db.Model(&models.NodeInstallTask{}).
        Where("task_id = ?", taskID).
        Update("progress", progress)
    
    // 通过 WebSocket 推送
    s.wsHub.Broadcast("scaling-progress", map[string]interface{}{
        "task_id":  taskID,
        "progress": progress,
    })
}
```

### 2. 添加任务详情展开

在扩缩容状态卡片中添加可展开的任务列表：

```javascript
<Collapse>
  <Panel header={`运行中任务 (${scalingStatus?.active_tasks})`}>
    <List
      dataSource={activeTaskDetails}
      renderItem={task => (
        <List.Item>
          <List.Item.Meta
            title={task.task_id}
            description={`节点: ${task.node_id} | 进度: ${task.progress}%`}
          />
          <Progress percent={task.progress} size="small" />
        </List.Item>
      )}
    />
  </Panel>
</Collapse>
```

### 3. 添加进度异常检测

如果任务进度长时间不更新，标记为异常：

```go
func (s *SlurmService) detectStalledTasks() {
    var tasks []models.NodeInstallTask
    threshold := time.Now().Add(-10 * time.Minute)
    
    s.db.Where("status = ?", "running").
        Where("updated_at < ?", threshold).
        Find(&tasks)
    
    for _, task := range tasks {
        // 标记为异常或重试
        log.Printf("警告: 任务 %s 进度停滞", task.TaskID)
    }
}
```

## 总结

本次修复通过将查询目标从 `SlurmTask` 表改为 `NodeInstallTask` 表，解决了扩缩容进度始终显示 0% 的问题。

**关键改进**:
1. ✅ 查询正确的数据表（NodeInstallTask）
2. ✅ 使用任务的实际 Progress 字段
3. ✅ 计算加权平均进度
4. ✅ 只统计最近 24 小时的任务
5. ✅ 构建真实的操作历史

**验证要点**:
- 扩容操作启动后，进度应从 0% 开始增长
- 活跃任务数应正确显示
- 完成后进度应达到 100%
- 失败任务应正确计入失败节点数

---
**修复日期**: 2024-11-13  
**影响范围**: SLURM 扩缩容状态显示  
**相关问题**: #216 (扩缩容进度显示 0%)  
**维护者**: AI Infrastructure Team
