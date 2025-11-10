# SLURM 脚本执行架构文档

## 概述

本文档描述了 SLURM 节点管理脚本的架构和使用方法。系统支持通过 SSH（密码或密钥认证）远程执行脚本，实现 slurmd 服务的自动化管理。

## 脚本位置

所有 SLURM 相关的管理脚本位于：
```
src/backend/scripts/
├── start-slurmd.sh   # 启动 slurmd 服务
├── stop-slurmd.sh    # 停止 slurmd 服务
└── check-slurmd.sh   # 检查 slurmd 状态
```

## 架构设计

### 1. 脚本与代码分离

**设计理念：**
- 不在 Go 代码中硬编码 shell 命令
- 将复杂的启动逻辑封装到独立的 shell 脚本
- Go 代码负责脚本的传输和执行

**优点：**
- 脚本易于维护和测试
- 可以独立修改启动逻辑而无需重新编译
- 支持版本控制和回滚
- 便于调试和排查问题

### 2. SSH 执行机制

支持两种 SSH 认证方式：

#### 密码认证
```go
output, err := s.executeSSHCommandWithKey(
    host, port, user, 
    password, // 密码
    "",       // 空密钥
    command
)
```

#### 密钥认证
```go
output, err := s.executeSSHCommandWithKey(
    host, port, user, 
    "",         // 空密码
    privateKey, // 私钥路径或内容
    command
)
```

#### 混合认证（推荐）
```go
output, err := s.executeSSHCommandWithKey(
    host, port, user, 
    password,   // 密码作为备用
    privateKey, // 优先使用密钥
    command
)
```

## 核心方法

### 1. executeSSHCommandWithKey

```go
func (s *SlurmService) executeSSHCommandWithKey(
    host string,      // 目标主机
    port int,         // SSH 端口（默认 22）
    user string,      // 用户名
    password string,  // 密码（可选）
    privateKey string,// 私钥路径或内容（可选）
    command string    // 要执行的命令
) (string, error)
```

**功能：**
- 支持密码和密钥双重认证
- 自动检测私钥格式（文件路径或 PEM 内容）
- 返回命令输出和错误信息

**私钥格式支持：**
1. **文件路径：** `/path/to/id_rsa`
2. **PEM 内容：** 以 `-----BEGIN` 开头的字符串

### 2. executeScriptViaSSH

```go
func (s *SlurmService) executeScriptViaSSH(
    ctx context.Context,
    host string,
    port int,
    user string,
    password string,
    privateKey string,
    scriptPath string
) (string, error)
```

**功能：**
- 读取脚本文件内容
- 通过 SSH 远程执行脚本
- 支持相对路径（相对于项目根目录）

### 3. 服务管理方法

#### startSlurmServicesViaSSH
```go
func (s *SlurmService) startSlurmServicesViaSSH(
    ctx context.Context,
    host string, port int,
    user, password, privateKey string,
    logWriter io.Writer
) error
```
启动 slurmd 服务，执行 `start-slurmd.sh` 脚本。

#### checkSlurmServicesViaSSH
```go
func (s *SlurmService) checkSlurmServicesViaSSH(
    ctx context.Context,
    host string, port int,
    user, password, privateKey string
) (string, error)
```
检查 slurmd 服务状态，执行 `check-slurmd.sh` 脚本。

#### stopSlurmServicesViaSSH
```go
func (s *SlurmService) stopSlurmServicesViaSSH(
    ctx context.Context,
    host string, port int,
    user, password, privateKey string
) (string, error)
```
停止 slurmd 服务，执行 `stop-slurmd.sh` 脚本。

## 使用示例

### 示例 1: 使用密码认证启动服务

```go
ctx := context.Background()
logWriter := os.Stdout

err := slurmSvc.startSlurmServicesViaSSH(
    ctx,
    "192.168.1.100",  // 节点IP
    22,               // SSH端口
    "root",           // 用户名
    "password123",    // 密码
    "",               // 不使用私钥
    logWriter,
)

if err != nil {
    log.Printf("启动失败: %v", err)
}
```

### 示例 2: 使用私钥认证启动服务

```go
// 方式1: 使用私钥文件路径
privateKeyPath := "/home/user/.ssh/id_rsa"

err := slurmSvc.startSlurmServicesViaSSH(
    ctx,
    "192.168.1.100",
    22,
    "root",
    "",              // 不使用密码
    privateKeyPath,  // 私钥文件路径
    logWriter,
)

// 方式2: 使用私钥内容
privateKeyContent := `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA...
-----END RSA PRIVATE KEY-----`

err := slurmSvc.startSlurmServicesViaSSH(
    ctx,
    "192.168.1.100",
    22,
    "root",
    "",
    privateKeyContent,  // 私钥内容
    logWriter,
)
```

### 示例 3: 检查服务状态

```go
output, err := slurmSvc.checkSlurmServicesViaSSH(
    ctx,
    "192.168.1.100",
    22,
    "root",
    "password123",
    "",
)

if err != nil {
    log.Printf("检查失败: %v", err)
} else {
    fmt.Println("服务状态:")
    fmt.Println(output)
}
```

### 示例 4: 在扩容流程中使用

```go
// 在 ScaleUp 方法中
for _, node := range nodes {
    // 安装 SLURM 包...
    // 配置文件...
    
    // 启动服务（支持密钥认证）
    var logBuffer bytes.Buffer
    err := s.startSlurmServicesViaSSH(
        ctx,
        node.Host,
        node.Port,
        node.User,
        node.Password,
        node.PrivateKey,  // 从 NodeConfig 获取
        &logBuffer,
    )
    
    if err != nil {
        log.Printf("节点 %s 启动失败: %v", node.Host, err)
        log.Printf("日志: %s", logBuffer.String())
    }
}
```

## NodeConfig 结构体

```go
type NodeConfig struct {
    Host       string `json:"host"`        // 节点主机名或IP
    Port       int    `json:"port"`        // SSH端口
    User       string `json:"user"`        // SSH用户名
    KeyPath    string `json:"key_path"`    // 私钥文件路径
    PrivateKey string `json:"private_key"` // 私钥内容（内联）
    Password   string `json:"password"`    // SSH密码
    
    // 硬件配置
    CPUs    int `json:"cpus"`
    Memory  int `json:"memory"`
    // ...
}
```

## API 接口

### 扩容时自动启动服务

```bash
curl -X POST http://localhost:8080/api/slurm/scaling/scale-up \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "nodes": [
      {
        "host": "192.168.1.100",
        "port": 22,
        "user": "root",
        "password": "password123",
        "cpus": 8,
        "memory": 16384
      },
      {
        "host": "192.168.1.101",
        "port": 22,
        "user": "root",
        "private_key": "-----BEGIN RSA PRIVATE KEY-----\n...",
        "cpus": 16,
        "memory": 32768
      }
    ]
  }'
```

## 脚本详解

### start-slurmd.sh

**功能：**
1. 启动 munge 服务（认证）
2. 创建必要的目录
3. 清理旧进程
4. 启动 slurmd 守护进程
5. 验证启动状态

**特点：**
- 自动选择 systemctl 或 nohup 方式
- 完整的日志输出
- 启动失败时显示详细错误

### stop-slurmd.sh

**功能：**
1. 尝试通过 systemctl 停止
2. 强制终止进程
3. 验证停止状态

### check-slurmd.sh

**功能：**
1. 检查 munge 和 slurmd 进程
2. 验证配置文件
3. 显示最近的日志
4. 检查网络端口

## 故障排查

### 问题 1: SSH 连接失败

**错误：** `SSH连接失败: dial tcp ...`

**解决：**
1. 检查目标主机网络是否可达：`ping <host>`
2. 验证 SSH 服务是否运行：`ssh user@host`
3. 检查防火墙规则

### 问题 2: 认证失败

**错误：** `ssh: unable to authenticate`

**解决：**
1. 验证密码正确性
2. 检查私钥格式：`ssh-keygen -y -f <keyfile>`
3. 确认用户权限

### 问题 3: 脚本执行失败

**错误：** `slurmd failed to start`

**解决：**
1. 查看完整日志输出
2. 手动执行脚本测试：
   ```bash
   ssh user@host "bash -s" < src/backend/scripts/start-slurmd.sh
   ```
3. 检查 slurm.conf 和 munge.key 配置

### 问题 4: slurmd 进程不持久

**现象：** 脚本执行成功但进程立即退出

**原因：**
- 使用 `slurmd -D &` 导致 SSH 会话结束时进程被终止

**解决：**
- 脚本已使用 `nohup` 确保进程持久化
- 验证：`ps aux | grep slurmd`

## 最佳实践

### 1. 安全性

- 优先使用密钥认证
- 密钥文件权限：`chmod 600 ~/.ssh/id_rsa`
- 避免在日志中输出敏感信息
- 使用专用的部署账户

### 2. 可靠性

- 实现重试机制
- 记录详细日志
- 验证每个步骤的执行结果
- 提供回滚能力

### 3. 性能

- 使用并发执行（已实现）
- 限制并发数（默认 5）
- 合理设置超时时间

### 4. 可维护性

- 脚本添加详细注释
- 版本化管理脚本
- 提供独立测试工具
- 编写完整文档

## 测试

### 单元测试

```bash
# 测试脚本语法
bash -n src/backend/scripts/start-slurmd.sh

# 本地测试脚本
bash src/backend/scripts/start-slurmd.sh
```

### 集成测试

```bash
# 测试 SSH 连接
ssh -i ~/.ssh/id_rsa user@host "echo 'SSH OK'"

# 测试脚本远程执行
ssh user@host "bash -s" < src/backend/scripts/check-slurmd.sh

# 测试完整流程
./test-slurmd-install.sh
```

## 未来改进

1. **脚本参数化：** 支持传递参数到脚本
2. **脚本版本管理：** 支持多版本脚本并存
3. **执行结果缓存：** 避免重复执行
4. **异步执行：** 支持长时间运行的脚本
5. **执行历史：** 记录所有脚本执行历史
6. **脚本热更新：** 支持动态更新脚本内容

## 总结

新的脚本执行架构具有以下优势：

- ✅ **灵活性：** 支持密码和密钥双重认证
- ✅ **可维护性：** 脚本与代码分离，易于修改
- ✅ **可扩展性：** 易于添加新的管理脚本
- ✅ **安全性：** 支持密钥认证，保护敏感信息
- ✅ **可靠性：** 完整的错误处理和日志记录
- ✅ **性能：** 并发执行多个节点

通过这套架构，实现了 SLURM 节点的自动化、标准化管理。
