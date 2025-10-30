# SaltStack 容器名检测修复

## 问题描述

在测试 SLURM 扩容等待机制时，发现所有节点都报告 "Minion部署成功但未能加入集群: 等待超时：Minion未能在指定时间内加入集群"。

### 错误日志

```
2025-10-30 00:15:21 - 任务已创建
2025-10-30 00:15:21 - 任务开始执行
2025-10-30 00:18:23 - Minion部署失败: Minion部署成功但未能加入集群: 等待超时：Minion未能在指定时间内加入集群
2025-10-30 00:18:23 - Minion部署失败: Minion部署成功但未能加入集群: 等待超时：Minion未能在指定时间内加入集群
2025-10-30 00:18:23 - Minion部署失败: Minion部署成功但未能加入集群: 等待超时：Minion未能在指定时间内加入集群
2025-10-30 00:18:23 - 节点test-rocky01已添加到集群
2025-10-30 00:18:23 - 节点test-rocky02已添加到集群
2025-10-30 00:18:23 - 节点test-rocky03已添加到集群
2025-10-30 00:18:23 - SLURM扩容成功
```

## 根本原因

在 `ssh_service.go` 的 `checkMinionAccepted()` 函数中，**硬编码**了 SaltStack 容器名为 `saltstack`：

```go
// 错误的代码
cmd := exec.Command("docker", "exec", "saltstack", "salt-key", "-L", "--out=json")
```

但实际的容器名是 `ai-infra-saltstack`：

```bash
$ docker ps | grep salt
b025a754ab84   ai-infra-saltstack:v0.3.6-dev   ...   ai-infra-saltstack
```

这导致：
1. ❌ 无法执行 `docker exec` 命令查询密钥状态
2. ❌ 无法自动接受 pending 的 Minion 密钥
3. ❌ 等待 3 分钟超时后报告失败
4. ✅ 但 Minion 实际上**已经成功部署**并连接到 Master（只是代码无法验证）

## 修复方案

### 1. 添加容器名检测辅助函数

创建 `getSaltStackContainerName()` 函数，支持：
- **环境变量**：优先使用 `SALT_CONTAINER_NAME` 环境变量
- **自动检测**：尝试常见的容器名列表
- **错误提示**：如果都找不到，返回详细的错误信息

```go
// getSaltStackContainerName 获取 SaltStack 容器名称
func getSaltStackContainerName() (string, error) {
	// 优先使用环境变量
	if containerName := os.Getenv("SALT_CONTAINER_NAME"); containerName != "" {
		return containerName, nil
	}
	
	// 尝试常见的容器名
	possibleContainers := []string{"ai-infra-saltstack", "saltstack", "salt-master"}
	for _, name := range possibleContainers {
		testCmd := exec.Command("docker", "exec", name, "echo", "test")
		if err := testCmd.Run(); err == nil {
			return name, nil
		}
	}
	
	return "", fmt.Errorf("无法找到 SaltStack 容器，尝试了: %v", possibleContainers)
}
```

### 2. 修改 `checkMinionAccepted()` 函数

使用辅助函数获取容器名：

```go
// checkMinionAccepted 检查 Minion 是否已被 Master 接受
func (s *SSHService) checkMinionAccepted(masterHost string, possibleMinionIDs []string) (bool, error) {
	// 获取 SaltStack 容器名
	saltContainerName, err := getSaltStackContainerName()
	if err != nil {
		return false, err
	}
	
	// 使用 docker exec 在 SaltStack 容器中执行 salt-key 命令
	cmd := exec.Command("docker", "exec", saltContainerName, "salt-key", "-L", "--out=json")
	
	output, err := cmd.CombinedOutput()
	// ... 其余逻辑
}
```

### 3. 修改自动接受密钥逻辑

同样使用辅助函数：

```go
// 如果在 pending 列表中，尝试自动接受
for _, minionID := range possibleMinionIDs {
	for _, pending := range keyList.MinionsPending {
		if pending == minionID {
			// 获取容器名并自动接受密钥
			saltContainerName, err := getSaltStackContainerName()
			if err != nil {
				return false, fmt.Errorf("无法找到 SaltStack 容器进行密钥接受: %v", err)
			}
			
			acceptCmd := exec.Command("docker", "exec", saltContainerName, "salt-key", "-y", "-a", minionID)
			if output, err := acceptCmd.CombinedOutput(); err != nil {
				return false, fmt.Errorf("自动接受密钥失败: %v, output: %s", err, string(output))
			}
			// 接受后立即返回 true
			return true, nil
		}
	}
}
```

## 修复效果

现在代码会：

1. ✅ **优先使用环境变量** `SALT_CONTAINER_NAME`（如果设置）
2. ✅ **自动检测**常见的容器名：
   - `ai-infra-saltstack`（当前项目的实际名称）
   - `saltstack`（之前硬编码的名称）
   - `salt-master`（其他可能的名称）
3. ✅ **详细错误提示**：如果找不到容器，明确说明尝试了哪些名称
4. ✅ **提高可移植性**：适配不同的部署环境和容器命名规范

## 配置选项

### 方式一：使用环境变量（推荐）

在 `.env` 文件中添加：

```bash
SALT_CONTAINER_NAME=ai-infra-saltstack
```

或在 `docker-compose.yml` 中为后端服务添加环境变量：

```yaml
backend:
  environment:
    - SALT_CONTAINER_NAME=ai-infra-saltstack
```

### 方式二：依赖自动检测（默认）

不设置环境变量，代码会自动尝试常见的容器名。

## 测试验证

### 测试步骤

1. **重启后端服务**：
   ```bash
   docker-compose restart backend
   ```

2. **执行 SLURM 扩容**：
   - 通过前端界面添加 3 个节点
   - 观察任务日志

3. **验证 Minion 状态**：
   ```bash
   docker exec ai-infra-saltstack salt-key -L
   ```

### 预期结果

**之前**（错误）：
```
Minion部署失败: Minion部署成功但未能加入集群: 等待超时：Minion未能在指定时间内加入集群
```

**修复后**（成功）：
```
✓ Minion部署成功
✓ 密钥已自动接受
✓ 节点已加入集群
```

## 相关文件

- `src/backend/internal/services/ssh_service.go` - 修复的核心文件
  - `getSaltStackContainerName()` - 新增辅助函数（第 979 行）
  - `checkMinionAccepted()` - 修改容器名检测（第 914 行）
  - 自动接受密钥逻辑 - 使用辅助函数（第 948 行）

## 后续优化建议

### 1. 添加容器名缓存

避免每次检查都重新检测：

```go
var (
	cachedContainerName string
	containerNameMutex  sync.Mutex
)

func getSaltStackContainerName() (string, error) {
	containerNameMutex.Lock()
	defer containerNameMutex.Unlock()
	
	if cachedContainerName != "" {
		return cachedContainerName, nil
	}
	
	// ... 检测逻辑 ...
	
	cachedContainerName = name
	return name, nil
}
```

### 2. 添加日志记录

记录容器名检测过程：

```go
func getSaltStackContainerName() (string, error) {
	if containerName := os.Getenv("SALT_CONTAINER_NAME"); containerName != "" {
		logrus.Infof("使用环境变量指定的 SaltStack 容器: %s", containerName)
		return containerName, nil
	}
	
	logrus.Debug("未设置 SALT_CONTAINER_NAME，开始自动检测...")
	// ... 检测逻辑 ...
	logrus.Infof("自动检测到 SaltStack 容器: %s", name)
	return name, nil
}
```

### 3. 支持 Kubernetes 环境

如果在 Kubernetes 中运行，使用 `kubectl exec` 替代 `docker exec`：

```go
func getSaltStackPodName() (string, error) {
	// 尝试使用 kubectl
	cmd := exec.Command("kubectl", "get", "pods", "-l", "app=saltstack", "-o", "name")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
```

### 4. 健康检查

在应用启动时验证 SaltStack 容器可用：

```go
func init() {
	if _, err := getSaltStackContainerName(); err != nil {
		logrus.Warnf("SaltStack 容器检测失败: %v", err)
		logrus.Warn("SLURM 扩容功能可能无法正常工作")
	}
}
```

## 相关问题修复

这次修复同时解决了以下相关问题：

1. ✅ **DOCS/SLURM_SCALEUP_WAIT_FIX.md** - 等待机制现在可以正常工作
2. ✅ **容器名硬编码** - 支持环境变量和自动检测
3. ✅ **错误提示不明确** - 详细说明尝试了哪些容器名
4. ✅ **可移植性差** - 适配不同的容器命名规范

## 验证清单

测试前请确认：

- [ ] 后端服务已重启加载新代码
- [ ] SaltStack 容器正在运行
- [ ] 测试节点（test-rocky01/02/03）SSH 连接正常
- [ ] 网络连接正常（节点可以访问 SaltStack Master）

测试后验证：

- [ ] Minion 部署不再超时
- [ ] 密钥自动被接受
- [ ] `salt-key -L` 显示所有节点在 Accepted Keys 中
- [ ] 数据库 `slurm_nodes` 表的 `salt_minion_id` 字段已填充
- [ ] 任务日志显示成功信息

## 总结

通过添加智能的容器名检测机制，修复了硬编码容器名导致的扩容失败问题。现在代码更加灵活、可移植，能够适应不同的部署环境和命名规范。
