# SLURM扩容等待机制修复

## 问题描述

之前的SLURM扩容实现存在严重问题：

1. **异步并发导致的假成功**：`DeploySaltMinion` 函数只是并发执行SSH安装脚本，但不等待Minion真正连接到Master
2. **立即返回成功**：`ScaleUp` 立即返回成功，但实际上Minion可能还没有被Master接受
3. **无验证机制**：没有验证Minion是否真正加入了集群

这导致：
- 前端显示"扩容成功"，但实际节点并未加入集群
- 数据库中创建了节点记录，但 `salt_minion_id` 字段为空
- SaltStack Master看不到新的Minion

## 修复方案

### 1. 添加等待和验证机制

在 `ssh_service.go` 中添加了三个新方法：

#### `waitForMinionsAccepted`
```go
func (s *SSHService) waitForMinionsAccepted(ctx context.Context, hosts []string, masterHost string) map[string]error
```

- **功能**：并发等待多个Minion被Master接受
- **超时**：每个Minion最多等待3分钟
- **返回**：每个主机的错误信息（如果有）

#### `waitForSingleMinionAccepted`
```go
func (s *SSHService) waitForSingleMinionAccepted(ctx context.Context, host, masterHost string) error
```

- **功能**：等待单个Minion被接受
- **轮询间隔**：5秒
- **Minion ID识别**：支持多种格式（IP、主机名、短主机名）

#### `checkMinionAccepted`
```go
func (s *SSHService) checkMinionAccepted(masterHost string, possibleMinionIDs []string) (bool, error)
```

- **功能**：检查Minion是否已被Master接受
- **实现**：通过 `docker exec saltstack salt-key -L --out=json` 查询
- **自动接受**：如果Minion在pending列表中，自动执行 `salt-key -y -a <minion_id>`

### 2. 修改 `DeploySaltMinion` 方法

```go
func (s *SSHService) DeploySaltMinion(ctx context.Context, connections []SSHConnection, config SaltStackDeploymentConfig) ([]DeploymentResult, error)
```

**增强逻辑**：

1. 并发部署所有Minion（保持原有逻辑）
2. 收集成功部署的主机列表
3. **新增**：如果 `config.AutoAccept == true`，等待所有Minion被Master接受
4. **新增**：更新部署结果，标记未能加入集群的节点为失败

### 3. 工作流程

```
部署Minion
    ↓
SSH执行安装脚本
    ↓
启动salt-minion服务
    ↓
【新增】等待Minion连接到Master (轮询检查)
    ↓
【新增】检查salt-key状态
    ↓
如果在pending列表 → 【新增】自动接受密钥
    ↓
如果在accepted列表 → 成功
    ↓
超时3分钟 → 失败
```

## 代码修改

### 文件：`src/backend/internal/services/ssh_service.go`

#### 1. 添加 `encoding/json` 导入

```go
import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"  // 新增
	"fmt"
	// ...
)
```

#### 2. 修改 `DeploySaltMinion` 方法

```go
// DeploySaltMinion 并发部署SaltStack Minion到多个节点
func (s *SSHService) DeploySaltMinion(ctx context.Context, connections []SSHConnection, config SaltStackDeploymentConfig) ([]DeploymentResult, error) {
	results := make([]DeploymentResult, len(connections))
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, s.config.MaxConcurrency)

	// 原有并发部署逻辑
	for i, conn := range connections {
		wg.Add(1)
		go func(index int, connection SSHConnection) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			startTime := time.Now()
			result := s.deploySingleMinion(ctx, connection, config)
			result.Duration = time.Since(startTime)
			results[index] = result
		}(i, conn)
	}
	wg.Wait()
	
	// 【新增】等待所有成功部署的 Minion 被 Master 接受
	if config.AutoAccept {
		successfulHosts := []string{}
		for i, result := range results {
			if result.Success {
				successfulHosts = append(successfulHosts, connections[i].Host)
			}
		}
		
		if len(successfulHosts) > 0 {
			// 等待 Minion 密钥被接受（最多等待5分钟）
			waitCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
			defer cancel()
			
			acceptErrors := s.waitForMinionsAccepted(waitCtx, successfulHosts, config.MasterHost)
			
			// 更新结果中的错误信息
			for i, result := range results {
				if result.Success {
					host := connections[i].Host
					if err, exists := acceptErrors[host]; exists && err != nil {
						results[i].Success = false
						results[i].Error = fmt.Sprintf("Minion部署成功但未能加入集群: %v", err)
					}
				}
			}
		}
	}
	
	return results, nil
}
```

#### 3. 新增三个辅助方法

见上述"修复方案"部分的详细说明。

## 影响范围

### 直接影响

- **SLURM扩容**：`ScaleUp` 和 `ScaleUpAsync` 现在会真正等待节点加入集群
- **SaltStack部署**：所有通过 `DeploySaltMinion` 的部署都会验证成功

### 时间影响

- **增加部署时间**：每个Minion最多增加3分钟等待时间（实际通常10-30秒）
- **更可靠**：避免假成功，确保节点真正可用

### 兼容性

- **向后兼容**：如果 `config.AutoAccept = false`，行为与之前一致（不等待）
- **默认行为**：当前代码中 `AutoAccept` 默认为 `true`，启用等待机制

## 测试建议

### 1. 正常流程测试

```bash
# 1. 启动测试环境
docker-compose up -d

# 2. 通过API扩容节点
curl -X POST http://localhost:8080/api/slurm/scaling/scale-up-async \
  -H "Content-Type: application/json" \
  -d '{
    "nodes": [
      {
        "host": "test-ssh01",
        "port": 22,
        "user": "root",
        "keyPath": "/app/ssh-key/id_rsa"
      }
    ]
  }'

# 3. 检查任务进度
# 应该看到 "等待Minion加入集群" 的步骤

# 4. 验证结果
docker exec saltstack salt-key -L
# 应该在 Accepted Keys 中看到 test-ssh01
```

### 2. 超时测试

```bash
# 故意提供错误的Master地址，导致Minion无法连接
# 应该在3分钟后返回失败
```

### 3. 自动接受测试

```bash
# 1. 手动启动一个Minion（不自动接受）
# 2. 观察后端是否自动执行 salt-key -a

# 验证
docker exec saltstack salt-key -L
# 应该看到密钥从 Unaccepted 移动到 Accepted
```

## 后续优化建议

### 1. 可配置超时时间

```go
type SaltStackDeploymentConfig struct {
	MasterHost    string
	MasterPort    int
	AutoAccept    bool
	AcceptTimeout time.Duration  // 新增：允许自定义超时时间
}
```

### 2. 进度事件上报

在等待期间发送进度事件：

```go
pm.Emit(opID, services.ProgressEvent{
	Type:     "info",
	Step:     "wait-minion-accept",
	Message:  fmt.Sprintf("等待 %s 加入集群... (%d秒)", host, elapsed),
	Progress: currentProgress,
})
```

### 3. 健康检查

除了检查密钥接受，还可以：

```bash
# 验证Minion响应
salt 'test-ssh01' test.ping

# 检查Minion版本
salt 'test-ssh01' test.version
```

### 4. 数据库同步

自动更新数据库中的 `salt_minion_id`：

```go
// 在Minion被接受后
db.Exec("UPDATE slurm_nodes SET salt_minion_id = ? WHERE host = ?", minionID, host)
```

## 验证清单

- [ ] 编译通过，无语法错误
- [ ] 正常扩容流程：节点成功加入集群
- [ ] 超时处理：3分钟后返回失败
- [ ] 自动接受：pending密钥被自动接受
- [ ] 并发处理：多个节点同时部署
- [ ] 错误处理：部分成功/部分失败的情况
- [ ] 数据库一致性：`salt_minion_id` 字段正确更新
- [ ] 前端显示：进度和结果正确显示
- [ ] 性能影响：等待时间在可接受范围

## 相关文件

- `src/backend/internal/services/ssh_service.go` - SSH服务和Minion部署
- `src/backend/internal/controllers/slurm_controller.go` - SLURM扩容控制器
- `src/backend/internal/services/saltstack_service.go` - SaltStack API服务

## 参考文档

- [SaltStack Key Management](https://docs.saltproject.io/en/latest/topics/tutorials/walkthrough.html#key-management)
- [SaltStack Minion Configuration](https://docs.saltproject.io/salt/user-guide/en/latest/topics/overview.html#minions)
- [Go Context and Timeout](https://go.dev/blog/context)
