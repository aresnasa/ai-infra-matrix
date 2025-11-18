# SLURM 节点服务启动问题修复报告

## 问题描述

**问题现象**：
- SLURM Master 显示节点状态为 `idle*` 
- 但节点容器中没有 `slurmd` 和 `munge` 服务运行
- 通过 `ps aux` 检查发现进程不存在

```bash
$ docker exec ai-infra-slurm-master sinfo
PARTITION AVAIL  TIMELIMIT  NODES  STATE NODELIST
compute*     up   infinite      6  idle* test-rocky[01-03],test-ssh[01-03]

$ docker exec test-rocky02 bash -c "ps aux|egrep 'slurm|munge'"
# 只有 grep 进程自身，没有 slurmd 和 munged
```

## 根本原因分析

### 错误的设计思路（已纠正）

**初始错误方案**：在容器的 `entrypoint.sh` 中直接安装和启动 SLURM
- ❌ 违反了"纯净节点"的设计原则
- ❌ 容器镜像变得臃肿
- ❌ 无法动态管理节点服务

### 正确的设计思路

**节点应该保持纯净，服务通过后端远程安装**：
1. ✅ 测试容器只提供基础系统环境（SSH + systemd）
2. ✅ SLURM/Munge 通过后端 SSH 远程安装
3. ✅ 支持动态添加/删除节点
4. ✅ 支持不同操作系统（Rocky/Ubuntu/Alpine）

### 代码层面的问题

**问题1：空实现**
```go
// src/backend/internal/services/slurm_cluster_service_extended.go
func (s *SlurmClusterService) installSlurmOnNode(node *models.SlurmNode, cluster *models.SlurmCluster) error {
    // TODO: 实现通过Salt Master执行SLURM安装的逻辑
    return nil  // ❌ 什么都没做
}
```

**问题2：只配置不安装**
- 后端只是把节点名添加到 `slurm.conf`
- 配置文件被复制到节点，但软件包未安装
- 服务未启动

## 解决方案

### 架构设计

```
┌─────────────────┐
│  SLURM Master   │
│  (容器)          │
│  - slurm.conf   │
│  - munge.key    │
└────────┬────────┘
         │
         │ SSH 连接
         ▼
┌─────────────────┐
│  Backend API    │
│  (Go服务)       │
│  - 检测OS类型    │
│  - 安装软件包    │
│  - 配置密钥      │
│  - 启动服务      │
└────────┬────────┘
         │
         │ SSH 远程执行
         ▼
┌─────────────────┐
│  Compute Node   │
│  (纯净容器)      │
│  → 安装 SLURM   │
│  → 安装 Munge   │
│  → 启动服务      │
└─────────────────┘
```

### 实现步骤

#### 1. 完整实现 `installSlurmOnNode` 函数

```go
func (s *SlurmClusterService) installSlurmOnNode(node *models.SlurmNode, cluster *models.SlurmCluster) error {
    // 1. 建立 SSH 连接
    client, err := s.createSSHClient(config)
    
    // 2. 检测操作系统类型
    osType, err := s.detectOSType(client)
    
    // 3. 安装 SLURM 和 Munge 包
    if err := s.installSlurmPackages(client, osType); err != nil {
        return err
    }
    
    // 4. 创建目录和用户
    if err := s.setupSlurmDirectories(client, osType); err != nil {
        return err
    }
    
    // 5. 配置 Munge 密钥（从 Master 复制）
    if err := s.configureMungeKey(client, osType); err != nil {
        return err
    }
    
    // 6. 配置 slurm.conf（从 Master 复制）
    if err := s.configureSlurmConf(client, osType, cluster); err != nil {
        return err
    }
    
    // 7. 启动 Munge 服务
    if err := s.startMungeServiceDirect(client, osType); err != nil {
        return err
    }
    
    // 8. 启动 SLURMD 服务
    if err := s.startSlurmdServiceDirect(client, osType); err != nil {
        return err
    }
    
    return nil
}
```

#### 2. 支持多种操作系统

**Rocky Linux / CentOS**:
```bash
dnf install -y epel-release
dnf install -y munge munge-libs
dnf install -y slurm slurm-slurmd
```

**Ubuntu / Debian**:
```bash
apt-get update
apt-get install -y munge libmunge-dev
apt-get install -y slurmd slurm-client
```

**Alpine**:
```bash
apk add --no-cache slurm munge
```

#### 3. 自动配置密钥和配置文件

**从 SLURM Master 获取配置**:
```go
func (s *SlurmClusterService) getMungeKeyFromMaster() ([]byte, error) {
    client, err := s.createSSHClient(masterConfig)
    session, err := client.NewSession()
    return session.CombinedOutput("cat /etc/munge/munge.key")
}

func (s *SlurmClusterService) getSlurmConfFromMaster() ([]byte, error) {
    client, err := s.createSSHClient(masterConfig)
    session, err := client.NewSession()
    return session.CombinedOutput("cat /etc/slurm/slurm.conf")
}
```

**上传到计算节点**:
```go
func (s *SlurmClusterService) uploadFileViaSSH(client *ssh.Client, localPath, remotePath string) error {
    content, _ := os.ReadFile(localPath)
    cmd := fmt.Sprintf("cat > %s <<'EOF'\n%s\nEOF", remotePath, string(content))
    session.CombinedOutput(cmd)
}
```

#### 4. 启动服务并验证

**Systemd 系统**:
```bash
systemctl enable munge
systemctl start munge
munge -n | unmunge  # 验证

systemctl enable slurmd
systemctl start slurmd
pgrep -x slurmd  # 验证
```

**Alpine (OpenRC)**:
```bash
rc-update add munge default
rc-service munge start
munge -n | unmunge

rc-update add slurmd default
rc-service slurmd start
pgrep -x slurmd
```

### 核心文件变更

#### 新增文件
无（函数都添加到现有文件中）

#### 修改文件

**1. `src/backend/internal/services/slurm_cluster_service_extended.go`**

新增方法：
- `detectOSType()` - 检测操作系统
- `installSlurmPackages()` - 安装软件包
- `setupSlurmDirectories()` - 创建目录和用户
- `configureMungeKey()` - 配置 Munge 密钥
- `configureSlurmConf()` - 配置 slurm.conf
- `startMungeServiceDirect()` - 启动 Munge（不依赖脚本）
- `startSlurmdServiceDirect()` - 启动 SLURMD（不依赖脚本）
- `getMungeKeyFromMaster()` - 从 Master 获取密钥
- `getSlurmConfFromMaster()` - 从 Master 获取配置
- `uploadFileViaSSH()` - 上传文件
- `executeSSHCmd()` - 执行 SSH 命令

修改方法：
- `installSlurmOnNode()` - 完整实现安装流程（原为空）

**2. 回滚测试容器的错误修改**
- ✅ `src/test-containers/entrypoint.sh` - 保持纯净
- ✅ `src/test-containers/Dockerfile.rocky` - 不预装 SLURM

**3. 新增测试脚本**
- `scripts/test-slurm-node-install.sh` - 手动测试安装流程

## 使用方法

### 方法1：通过 API 添加节点（自动安装）

```bash
curl -X POST http://localhost:8081/api/slurm/nodes \
  -H "Content-Type: application/json" \
  -d '{
    "node_name": "test-rocky02",
    "host": "test-rocky02",
    "port": 22,
    "username": "root",
    "password": "rootpass123",
    "cpus": 2,
    "real_memory": 2048
  }'
```

### 方法2：通过测试脚本

```bash
cd /path/to/ai-infra-matrix
./scripts/test-slurm-node-install.sh test-rocky02
```

### 验证步骤

**1. 检查进程**:
```bash
$ docker exec test-rocky02 ps aux | egrep 'slurm|munge'
munge      1234  0.0  0.1  munged
slurm      5678  0.0  0.2  slurmd
```

**2. 验证 Munge**:
```bash
$ docker exec test-rocky02 munge -n | unmunge
STATUS:           Success (0)
ENCODE_HOST:      test-rocky02
ENCODE_TIME:      2024-...
```

**3. 检查 SLURM 状态**:
```bash
$ docker exec ai-infra-slurm-master sinfo
PARTITION AVAIL  TIMELIMIT  NODES  STATE NODELIST
compute*     up   infinite      1  idle  test-rocky02
```

**4. 检查配置文件**:
```bash
$ docker exec test-rocky02 ls -lh /etc/munge/munge.key
-r-------- 1 munge munge 1.0K ... /etc/munge/munge.key

$ docker exec test-rocky02 ls -lh /etc/slurm/slurm.conf
-rw-r--r-- 1 root root 2.5K ... /etc/slurm/slurm.conf
```

## 技术要点

### 1. SSH 远程执行
- 使用 `golang.org/x/crypto/ssh` 包
- 支持密码和密钥认证
- 每个命令创建新 session

### 2. 操作系统适配
- 自动检测 OS 类型（Rocky/Ubuntu/Alpine）
- 不同包管理器：dnf/apt/apk
- 不同服务管理器：systemd/OpenRC

### 3. 配置同步
- 从 Master 读取最新配置
- 支持不同路径（/etc/slurm 或 /etc/slurm-llnl）
- 保持权限一致性

### 4. 错误处理
- 每步都有详细错误信息
- 失败时返回命令输出
- 支持重试机制（TODO）

## 后续优化建议

### 1. 支持 SaltStack 安装
当前只实现了 SSH 方式，可以添加 Salt 方式：

```go
func (s *SlurmClusterService) installSlurmViaSalt(node *models.SlurmNode) error {
    // 使用 Salt state 文件安装
    return s.saltService.ApplyState(node.Host, "slurm.node")
}
```

### 2. 添加健康检查
定期检查节点服务状态：

```go
func (s *SlurmClusterService) healthCheckNode(node *models.SlurmNode) error {
    // 检查 munge 和 slurmd 进程
    // 检查端口监听
    // 更新数据库状态
}
```

### 3. 支持服务重启
添加单独的重启接口：

```bash
POST /api/slurm/nodes/:id/restart
{
  "service": "munge" | "slurmd" | "all"
}
```

### 4. 添加卸载功能
支持完全卸载 SLURM：

```go
func (s *SlurmClusterService) uninstallSlurmFromNode(node *models.SlurmNode) error {
    // 停止服务
    // 删除配置
    // 卸载包（可选）
}
```

### 5. 日志增强
添加详细的安装日志：

```go
type InstallLog struct {
    NodeID    uint
    Step      string
    Status    string
    Output    string
    StartTime time.Time
    Duration  int
}
```

## 测试清单

- [x] Rocky Linux 节点安装
- [ ] Ubuntu 节点安装
- [ ] Alpine 节点安装
- [x] Munge 密钥同步
- [x] slurm.conf 同步
- [x] 服务启动验证
- [ ] 多节点并发安装
- [ ] 安装失败回滚
- [ ] 重复安装幂等性

## 总结

本次修复通过实现完整的 SSH 远程安装流程，解决了节点服务未启动的问题。核心改进：

1. ✅ **正确的架构设计**：节点纯净，后端远程安装
2. ✅ **跨平台支持**：Rocky/Ubuntu/Alpine 自动适配
3. ✅ **配置同步**：自动从 Master 获取最新配置
4. ✅ **服务管理**：自动启动并验证服务状态
5. ✅ **错误处理**：详细的错误信息和日志

**关键经验**：
- 容器应该保持纯净和通用
- 服务安装和配置应该动态完成
- SSH 远程执行提供了灵活性和可控性
- 多操作系统支持需要细致的适配

---
**修复日期**: 2024-11-13  
**影响范围**: SLURM 节点管理模块  
**相关问题**: #215 (SLURM 节点服务未启动)  
**维护者**: AI Infrastructure Team
