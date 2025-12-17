# SaltStack 页面修复总结

## 已完成的修复

### 1. ✅ 翻译问题修复

#### 问题描述
- `saltstack.executeFailed` 翻译键缺失 → 显示为 "undefined"
- `saltstack.settings` 翻译键缺失 → 设置标签页显示为 "undefined"

#### 修复方案
在两个翻译文件中添加缺失的翻译键：

**zh-CN.js 和 en-US.js 中的 saltstack 对象添加:**
```javascript
executeFailed: '执行失败' / 'Execution failed'
settings: '设置' / 'Settings'
jobRetentionSettings: '作业保留设置' / 'Job Retention Settings'
retentionDays: '保留天数' / 'Retention Days'
autoCleanupEnabled: '自动清理' / 'Auto Cleanup'
cleanupIntervalHours: '清理间隔（小时）' / 'Cleanup Interval (Hours)'
maxJobsCount: '最大作业数' / 'Max Jobs Count'
redisCacheDays: 'Redis缓存天数' / 'Redis Cache Days'
saveSuccess: '保存成功' / 'Saved successfully'
saveFailed: '保存失败' / 'Save failed'
manualCleanup: '手动清理' / 'Manual Cleanup'
triggerCleanup: '触发清理' / 'Trigger Cleanup'
cleanupSuccess: '清理成功' / 'Cleanup successful'
cleanupFailed: '清理失败' / 'Cleanup failed'
```

修改文件：
- [src/frontend/src/locales/zh-CN.js](src/frontend/src/locales/zh-CN.js#L260)
- [src/frontend/src/locales/en-US.js](src/frontend/src/locales/en-US.js#L260)

### 2. ✅ Master 版本和运行时间显示修复

#### 问题描述
- Master 版本显示为"未知"
- Master 运行时间显示为"未知"

#### 修复方案

在后端 `saltstack_handler.go` 中添加了以下改进：

**a) 获取 Salt 版本信息**
- 新增 `extractSaltVersion()` 函数：通过 `test.version` runner 命令获取真实的 Salt 版本
- 支持多种响应格式：
  - 直接版本字符串：`{"return": ["3006.9"]}`
  - 结构化响应：`{"return": [{"salt": "3006.9"}]}`

**b) 获取 Salt Master 运行时间**
- 新增 `getSaltMasterUptime()` 函数：
  - 优先从 Docker API 获取容器启动时间
  - 如果失败，通过 Salt 执行 `ps` 命令获取进程运行时间
  
- 新增 `getUptimeFromDocker()` 函数：
  - 连接到 Docker 守护进程
  - 查询 Salt Master 容器信息
  - 从容器 `StartedAt` 时间计算运行时长

**c) 格式化显示**
- 新增 `formatUptime()` 函数：
  - 将秒数格式化为人类可读的字符串
  - 示例：1000000秒 → "11天 13小时"

**d) 新增结构体字段**
- `SaltStackStatus` 中添加 `UptimeStr` 字段用于存储格式化的运行时间

**e) 前端更新**
- [SaltStackDashboard.js](src/frontend/src/pages/SaltStackDashboard.js#L3768) 
- 更新为显示 `uptime_str` 字段（格式化的运行时间）

修改文件：
- [src/backend/internal/handlers/saltstack_handler.go](src/backend/internal/handlers/saltstack_handler.go#L970-L2700)

### 3. 📝 任务状态更新逻辑（已存在，需要验证）

#### 现有逻辑
`ExecuteSaltCommand` 中的轮询逻辑：
1. 使用 `local_async` 模式异步执行命令
2. 获取 JID（作业ID）
3. 保存作业到数据库 (status="running")
4. 轮询 `jobs.lookup_jid` 等待结果（最多90秒）
5. 收到结果后调用 `UpdateJobResult` 更新数据库状态

#### 可能的问题
- 如果任务执行时间超过90秒，会被标记为 `timeout`
- 如果网络中断，轮询可能失败
- 前端可能没有及时刷新作业列表

#### 调试建议
1. 查看后端日志，搜索：
   - `[DEBUG] 作业状态已更新` - 表示 UpdateJobResult 被成功调用
   - `[WARNING] 等待作业` - 表示轮询超时

2. 在前端执行完任务后，点击"刷新"按钮重新加载作业列表

3. 检查数据库：
   ```sql
   SELECT jid, task_id, status, start_time, end_time, duration FROM salt_jobs 
   ORDER BY start_time DESC LIMIT 10;
   ```

## 测试步骤

### 快速测试翻译修复
1. 访问 http://192.168.48.123:8080/saltstack
2. 查看设置标签页，确认显示"设置"而不是"undefined"
3. 执行一个命令，确认错误消息显示"执行失败"而不是"undefined"

### 测试 Master 信息显示
1. 访问 SaltStack 页面
2. 查看"Master 信息"卡片中的：
   - 版本：应显示类似 "3006.9" 或具体版本号
   - 运行时间：应显示类似 "3天 5小时" 的格式

### 完整测试流程
```bash
# 1. 确保后端已重新编译
cd /Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix/src/backend
go build -o main ./cmd/main.go

# 2. 重启后端服务
docker-compose restart saltstack-backend  # 或相应的容器

# 3. 访问页面进行测试
# http://192.168.48.123:8080/saltstack

# 4. 查看后端日志
docker logs -f saltstack-backend 2>&1 | grep -E "SaltStackStatus|作业状态已更新"
```

## 相关文件修改

- ✅ [src/frontend/src/locales/zh-CN.js](src/frontend/src/locales/zh-CN.js) - 添加缺失翻译
- ✅ [src/frontend/src/locales/en-US.js](src/frontend/src/locales/en-US.js) - 添加缺失翻译
- ✅ [src/frontend/src/pages/SaltStackDashboard.js](src/frontend/src/pages/SaltStackDashboard.js#L3768) - 更新运行时间显示
- ✅ [src/backend/internal/handlers/saltstack_handler.go](src/backend/internal/handlers/saltstack_handler.go) - 添加版本和运行时间获取逻辑

## 状态

| 功能 | 状态 | 备注 |
|-----|------|------|
| executeFailed 翻译 | ✅ 完成 | 已添加到 saltstack.* |
| settings 翻译 | ✅ 完成 | 已添加到 saltstack.* |
| Master 版本显示 | ✅ 完成 | 通过 test.version 获取，添加了日志记录 |
| Master 运行时间 | ✅ 完成 | 通过 Docker API 或 ps 命令获取 |
| 作业状态更新 | ✅ 已实现 | 需要实时验证 |

