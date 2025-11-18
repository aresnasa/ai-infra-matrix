# SLURM 作业队列管理 - 快速开始指南

## 功能说明

SLURM 作业队列管理功能支持对运行中和等待中的作业进行批量操作，包括：

- **取消作业 (CANCEL)** - 终止作业执行
- **暂停调度 (HOLD)** - 阻止作业被调度（仅对 PENDING 作业有效）
- **释放作业 (RELEASE)** - 允许暂停的作业重新被调度
- **挂起作业 (SUSPEND)** - 暂停正在运行的作业
- **恢复作业 (RESUME)** - 继续被挂起的作业
- **重新入队 (REQUEUE)** - 将作业重新放入队列

## 前端使用

### 1. 访问作业队列

1. 打开 Web 界面：http://your-server:8080/slurm
2. 点击"作业队列"标签页
3. 查看当前运行的作业列表

### 2. 选择作业

- 点击表格左侧的复选框选择单个或多个作业
- 使用表头的复选框可以全选/反选/清除选择
- 选中的作业数量会显示在表格右上角

### 3. 执行操作

1. 选择至少一个作业
2. 点击右上角的"作业操作"按钮
3. 从下拉菜单中选择要执行的操作
4. 在确认对话框中点击"确认"
5. 等待操作完成并查看结果消息

## API 使用

### 请求格式

```http
POST /api/slurm/jobs/manage
Content-Type: application/json

{
  "job_ids": ["123", "456", "789"],
  "action": "hold",
  "signal": ""
}
```

### 参数说明

- `job_ids` (必填) - 作业 ID 数组
- `action` (必填) - 操作类型：`cancel`/`hold`/`release`/`suspend`/`resume`/`requeue`
- `signal` (可选) - 取消作业时的信号，如 `SIGTERM`, `SIGKILL`

### 响应格式

**成功响应：**
```json
{
  "success": true,
  "message": "成功对 3 个作业执行 hold 操作",
  "jobs": ["123", "456", "789"],
  "action": "hold"
}
```

**失败响应（SSH 模式）：**
```json
{
  "success": false,
  "error": "执行命令失败: connection timeout",
  "output": "..."
}
```

**部分失败响应（REST API 模式）：**
```json
{
  "success": false,
  "message": "部分作业操作失败: 2 成功, 1 失败",
  "failed_jobs": ["789"],
  "action": "hold"
}
```

## 操作示例

### 示例 1: 取消单个作业

```bash
curl -X POST http://localhost:8080/api/slurm/jobs/manage \
  -H "Content-Type: application/json" \
  -d '{
    "job_ids": ["123"],
    "action": "cancel"
  }'
```

### 示例 2: 暂停多个作业

```bash
curl -X POST http://localhost:8080/api/slurm/jobs/manage \
  -H "Content-Type: application/json" \
  -d '{
    "job_ids": ["123", "456", "789"],
    "action": "hold"
  }'
```

### 示例 3: 使用特定信号取消作业

```bash
curl -X POST http://localhost:8080/api/slurm/jobs/manage \
  -H "Content-Type: application/json" \
  -d '{
    "job_ids": ["123"],
    "action": "cancel",
    "signal": "SIGKILL"
  }'
```

## 配置双模式

### SSH 模式（默认）

在 `docker-compose.yml` 中配置：

```yaml
backend:
  environment:
    USE_SLURMRESTD: "false"  # 或不设置此变量
    SLURM_MASTER_HOST: "192.168.0.200"
    SLURM_MASTER_USER: "root"
    SLURM_SSH_KEY_PATH: "/root/.ssh/id_rsa"
```

### REST API 模式

在 `docker-compose.yml` 中配置：

```yaml
backend:
  environment:
    USE_SLURMRESTD: "true"
    SLURM_REST_API_URL: "http://192.168.0.200:6820"
    SLURM_JWT_TOKEN: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

## 常见问题

### Q1: 操作显示成功但作业状态未改变

**A:** 检查以下几点：
1. 确认作业仍在运行（未已完成或失败）
2. 某些操作只对特定状态的作业有效（如 HOLD 只对 PENDING 作业有效）
3. 等待 30 秒让页面自动刷新，或手动点击"刷新"按钮

### Q2: REST API 模式下所有操作都失败

**A:** 检查 slurmrestd 服务：
```bash
# 在 SLURM Master 上
systemctl status slurmrestd
curl http://localhost:6820/slurm/v0.0.40/diag
```

### Q3: 作业操作按钮不显示

**A:** 确保：
1. 至少选择了一个作业
2. Frontend 版本是 v0.3.8 或更高
3. 清除浏览器缓存并刷新页面

### Q4: 如何批量取消某个用户的所有作业？

**A:** 使用 SSH 登录 SLURM Master：
```bash
# 查看用户的所有作业
squeue -u username

# 批量取消
scancel -u username
```

或使用 API：
```bash
# 先获取作业列表
curl http://localhost:8080/api/slurm/jobs | jq '.[] | select(.user=="username") | .id'

# 然后批量取消
curl -X POST http://localhost:8080/api/slurm/jobs/manage \
  -H "Content-Type: application/json" \
  -d '{
    "job_ids": ["123", "456", "789"],
    "action": "cancel"
  }'
```

## 权限要求

### SSH 模式
- 需要 SSH 访问 SLURM Master 的权限
- SSH 用户需要有执行 `scancel` 和 `scontrol` 命令的权限
- 通常需要 root 或 slurm 用户权限

### REST API 模式
- 需要有效的 JWT Token
- Token 对应的用户需要有作业管理权限
- 可以通过 SLURM 的 ACL 配置细粒度权限

## 性能说明

### SSH 模式
- 每个操作需要建立 SSH 连接
- 批量操作 10 个作业约需 2-3 秒
- 适合小规模作业管理

### REST API 模式
- 每个操作通过 HTTP 请求
- 批量操作 10 个作业约需 0.5-1 秒
- 支持并发请求，适合大规模作业管理

## 日志查看

### Backend 日志
```bash
docker-compose logs -f backend | grep "ManageJobs\|JobViaAPI"
```

### Frontend 日志
在浏览器控制台（F12）查看：
```javascript
// 操作成功
"成功恢复 3 个作业"

// 操作失败
"恢复作业失败: connection timeout"
```

## 相关文档

- [SLURM 双模式配置指南](./SLURM_DUAL_MODE_CONFIG.md)
- [节点管理指南](./NODE_MANAGEMENT.md)
- [API 文档](./API_REFERENCE.md)

## 更新日志

### v0.3.8 (2024-01)
- ✅ 实现作业队列管理 UI
- ✅ 添加 6 种作业操作支持
- ✅ 支持 SSH 和 REST API 双模式
- ✅ 批量操作支持

### v0.3.6 (2023-12)
- 基础作业列表查看功能
