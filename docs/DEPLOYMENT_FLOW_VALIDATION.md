# 部署流程与状态验证机制

## 📋 概述

本文档说明SaltStack Minion和SLURM客户端安装的完整流程，以及如何确保只有在安装成功或失败后才更新任务数据库。

## 🏗️ 架构设计原则

### 关注点分离

- **Bash脚本**: 负责所有安装逻辑、验证和错误处理
- **Go代码**: 只负责调度执行和收集结果，不包含安装逻辑

### 错误传递链

```
Bash脚本退出码 → SSH Session → executeCommand() → deploySingleMinion() → DeploymentResult → 数据库更新
```

## 📂 部署脚本结构

### 脚本目录: `src/backend/scripts/salt-minion/`

```
salt-minion/
├── README.md                        # 脚本使用文档
├── 01-install-salt-minion.sh       # 安装salt-minion包
├── 02-configure-minion.sh          # 配置Master地址
├── 03-start-service.sh             # 启动salt-minion服务
└── 04-verify-status.sh             # 验证安装和服务状态
```

### 脚本编写规范

所有脚本遵循以下规范：

1. **使用 `set -e`**: 遇到错误立即退出
2. **明确的退出码**:
   - `exit 0`: 成功
   - `exit 1`: 失败
3. **验证关键步骤**: 每个脚本都验证其核心功能
4. **清晰的输出**: 使用标准化格式 `[Salt] ✓/✗/⚠ 消息`

### 04-verify-status.sh 关键验证

```bash
# 1. 验证命令存在
if ! command -v salt-minion >/dev/null 2>&1; then
    echo "[Salt] ✗ 致命错误: salt-minion 命令未找到"
    exit 1
fi

# 2. 验证配置文件
if [ ! -f /etc/salt/minion.d/99-master-address.conf ]; then
    echo "[Salt] ✗ 致命错误: Master配置文件不存在"
    exit 1
fi

# 3. 验证服务运行状态
if systemctl is-active --quiet salt-minion; then
    service_running=true
elif pgrep -x salt-minion >/dev/null; then
    service_running=true
else
    echo "[Salt] ✗ 致命错误: salt-minion服务未运行"
    exit 1
fi
```

## 🔧 Go代码流程

### 1. executeDeploymentSteps() - 脚本调度

```go
func (s *SSHService) executeDeploymentSteps(client *ssh.Client, config SaltStackDeploymentConfig) (string, error) {
    // 1. 加载脚本
    scripts, err := s.loadDeploymentScripts("scripts/salt-minion")
    
    // 2. 设置环境变量
    envVars := map[string]string{
        "APPHUB_URL":       config.AppHubURL,
        "SALT_MASTER_HOST": config.MasterHost,
    }
    
    // 3. 按顺序执行脚本
    for _, script := range scripts {
        fullCommand := envExports.String() + script.Content
        
        // 执行并检查退出码
        stepOutput, err := s.executeCommand(client, fullCommand)
        output.WriteString(stepOutput)
        
        // 脚本失败则立即返回错误
        if err != nil {
            return output.String(), fmt.Errorf("脚本 '%s' 执行失败: %v", script.Name, err)
        }
    }
    
    return output.String(), nil  // 所有脚本成功
}
```

**关键点**:
- ✅ 脚本失败会通过SSH session.Wait()返回错误
- ✅ 遇到第一个失败脚本立即停止
- ✅ 不添加额外验证逻辑，完全信任脚本的退出码

### 2. executeCommand() - SSH执行

```go
func (s *SSHService) executeCommand(client *ssh.Client, command string) (string, error) {
    session, err := client.NewSession()
    // ...
    
    // 启动命令
    session.Start(command)
    
    // 等待完成
    err := session.Wait()  // 这里会返回脚本的退出码
    
    return output.String(), err  // err != nil 表示非零退出码
}
```

**关键点**:
- ✅ `session.Wait()` 会捕获Shell脚本的退出码
- ✅ 非零退出码会转换为Go error
- ✅ 正确传递错误到上层

### 3. deploySingleMinion() - 单节点部署

```go
func (s *SSHService) deploySingleMinion(ctx context.Context, conn SSHConnection, config SaltStackDeploymentConfig) DeploymentResult {
    result := DeploymentResult{
        Host:    conn.Host,
        Success: false,  // 默认失败
    }
    
    // 执行部署
    output, err := s.executeDeploymentSteps(client, config)
    
    if err != nil {
        result.Error = fmt.Sprintf("部署失败: %v", err)
        result.Output = output
        return result  // 返回失败结果
    }
    
    result.Success = true  // 只有无错误才设置成功
    result.Output = output
    return result
}
```

**关键点**:
- ✅ 默认状态为失败
- ✅ 只有executeDeploymentSteps无错误才设置Success=true
- ✅ 错误信息和输出都被保存

### 4. ScaleUpAsync() - 任务控制器

```go
go func() {
    failed := false
    var finalError string
    
    defer func() {
        // 根据failed标志更新数据库
        status := "completed"
        if failed {
            status = "failed"
        }
        c.taskSvc.UpdateTaskStatus(bgCtx, dbTaskID, status, finalError)
    }()
    
    // 部署Minion
    results, err := c.sshSvc.DeploySaltMinion(ctx, connections, saltConfig)
    if err != nil {
        failed = true
        finalError = err.Error()
        return  // defer会更新数据库
    }
    
    // 检查每个节点的部署结果
    for i, result := range results {
        if result.Success {
            successCount++
            c.taskSvc.AddTaskEvent(bgCtx, dbTaskID, "success", ...)
        } else {
            failed = true  // 任何节点失败都标记任务失败
            failedCount++
            finalError = result.Error
            c.taskSvc.AddTaskEvent(bgCtx, dbTaskID, "error", ...)
        }
    }
    
    // 继续其他步骤...
}()
```

**关键点**:
- ✅ `failed`标志控制最终任务状态
- ✅ 任何节点失败都会设置`failed=true`
- ✅ defer确保无论如何都会更新数据库
- ✅ 每个节点的结果都记录为独立事件

## 🔄 完整执行流程

### 成功场景

```
1. Bash脚本执行成功
   ↓ exit 0
2. session.Wait() 返回 nil
   ↓ err == nil
3. executeCommand() 返回 (output, nil)
   ↓ err == nil
4. executeDeploymentSteps() 返回 (output, nil)
   ↓ err == nil
5. deploySingleMinion() 设置 Success=true
   ↓ result.Success == true
6. ScaleUpAsync() 不设置 failed=true
   ↓ failed == false
7. 数据库更新: status = "completed"
```

### 失败场景

```
1. Bash脚本验证失败
   ↓ exit 1
2. session.Wait() 返回 error
   ↓ err != nil
3. executeCommand() 返回 (output, error)
   ↓ err != nil
4. executeDeploymentSteps() 返回 (output, error)
   ↓ err != nil
5. deploySingleMinion() 设置 Success=false, Error=...
   ↓ result.Success == false
6. ScaleUpAsync() 设置 failed=true, finalError=...
   ↓ failed == true
7. 数据库更新: status = "failed", error_message = finalError
```

## ✅ 验证检查清单

### Bash脚本验证

- [x] 01-install: 验证salt-minion安装成功
- [x] 02-configure: 验证配置文件创建
- [x] 03-start: 验证服务启动
- [x] 04-verify: 综合验证所有关键点
- [x] 所有脚本使用`set -e`
- [x] 所有脚本有明确的exit码

### Go代码验证

- [x] executeCommand正确传递退出码
- [x] executeDeploymentSteps检查每个脚本错误
- [x] deploySingleMinion正确设置Success标志
- [x] ScaleUpAsync根据结果更新数据库
- [x] 错误信息被正确记录
- [x] 任务事件被正确记录

### 数据库验证

- [x] 任务状态只在成功/失败后更新
- [x] 失败时记录错误信息
- [x] 每个节点的结果独立记录
- [x] 进度正确更新

## 🎯 最佳实践

### 1. 脚本设计

```bash
#!/bin/bash
set -e  # 必须！确保错误自动传播

# 明确的错误处理
if ! some_command; then
    echo "[Salt] ✗ 错误: 命令失败"
    exit 1
fi

# 成功退出
echo "[Salt] ✓ 操作成功"
exit 0
```

### 2. Go代码设计

```go
// 不要在Go代码中添加验证逻辑
// ❌ 错误示例
if !strings.Contains(output, "SUCCESS") {
    return error
}

// ✅ 正确示例
output, err := executeCommand(...)
if err != nil {
    return err  // 直接传递脚本错误
}
```

### 3. 错误消息

```go
// ✅ 包含上下文的错误消息
fmt.Errorf("脚本 '%s' 执行失败: %v", script.Name, err)

// ✅ 保留原始输出
result.Output = output  // 即使失败也保存输出用于调试
```

## 📊 监控和调试

### 查看部署日志

```bash
# 查看任务详情
curl http://backend:8082/api/slurm/tasks/{taskId}

# 查看任务事件
curl http://backend:8082/api/slurm/tasks/{taskId}/events
```

### 手动测试脚本

```bash
# 设置环境变量
export APPHUB_URL="http://apphub:80"
export SALT_MASTER_HOST="saltstack"

# 按顺序执行脚本
cd /path/to/scripts/salt-minion
bash 01-install-salt-minion.sh
bash 02-configure-minion.sh
bash 03-start-service.sh
bash 04-verify-status.sh

# 检查退出码
echo $?  # 0=成功, 非0=失败
```

## 🔒 安全考虑

1. **脚本权限**: 脚本文件应只有受信任用户可修改
2. **环境变量**: 敏感信息不应通过环境变量传递
3. **输出过滤**: 日志中不应包含密码等敏感信息
4. **超时设置**: 避免脚本无限期运行

## 📝 总结

当前实现确保了：

1. ✅ **单一职责**: Bash负责安装，Go负责调度
2. ✅ **错误传递**: 脚本错误正确传递到数据库
3. ✅ **状态一致**: 数据库状态准确反映实际部署结果
4. ✅ **可维护性**: 安装逻辑独立可测试
5. ✅ **可扩展性**: 添加新步骤只需增加脚本文件

所有修改遵循了"Go代码只做公共步骤处理，安装逻辑封装在Bash脚本"的原则。
