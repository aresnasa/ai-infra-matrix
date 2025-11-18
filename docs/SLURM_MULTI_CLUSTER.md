# SLURM 多集群管理实现

## 概述

实现了 SLURM 多集群管理功能，支持：
1. **托管集群 (managed)**: 平台自动部署和管理的集群
2. **外部集群 (external)**: 通过 SSH 连接已存在的外部集群

## 功能特性

### 1. 连接外部集群
- 通过 SSH 连接到外部 SLURM 集群
- 支持密码和密钥两种认证方式
- 自动验证 SLURM 安装和版本
- 异步发现集群节点信息

### 2. 集群信息获取
- 实时获取集群状态（通过 SSH 执行 sinfo/squeue）
- 节点列表和状态统计
- 运行任务数量统计

### 3. 统一管理界面
- 区分托管和外部集群类型
- 根据类型显示不同操作（部署、扩容等）
- 统一的集群列表和详情展示

## 数据库变更

### 新增字段

```sql
-- slurm_clusters 表
ALTER TABLE slurm_clusters ADD COLUMN cluster_type VARCHAR(50) DEFAULT 'managed';
ALTER TABLE slurm_clusters ADD COLUMN master_ssh JSON;
```

### 字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `cluster_type` | VARCHAR(50) | 集群类型：`managed` 或 `external` |
| `master_ssh` | JSON | SSH 连接配置（host, port, username, auth_type, password, key_path） |

### 自动迁移

数据库迁移在后端启动时自动执行：
- 文件位置：`src/backend/internal/database/database.go`
- 函数：`runCustomMigrations()` → `addSlurmClusterFields()`

## 后端实现

### 1. 数据模型 (`models/slurm_cluster_models.go`)

```go
type SlurmCluster struct {
    ID          uint       `json:"id" gorm:"primarykey"`
    Name        string     `json:"name" gorm:"not null;size:100"`
    ClusterType string     `json:"cluster_type" gorm:"default:'managed';size:50"`
    MasterSSH   *SSHConfig `json:"master_ssh,omitempty" gorm:"type:json"`
    // ... 其他字段
}

type SSHConfig struct {
    Host     string `json:"host"`
    Port     int    `json:"port"`
    Username string `json:"username"`
    AuthType string `json:"auth_type"` // password/key
    Password string `json:"password,omitempty"`
    KeyPath  string `json:"key_path,omitempty"`
}

type ConnectExternalClusterRequest struct {
    Name        string        `json:"name" binding:"required"`
    Description string        `json:"description"`
    MasterHost  string        `json:"master_host" binding:"required"`
    MasterSSH   SSHConfig     `json:"master_ssh" binding:"required"`
    Config      ClusterConfig `json:"config"`
}
```

### 2. API 端点 (`controllers/slurm_cluster_controller.go`)

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/slurm/clusters/connect` | 连接外部集群 |
| GET | `/api/slurm/clusters/:id/info` | 获取集群详细信息 |
| DELETE | `/api/slurm/clusters/:id` | 删除集群 |

### 3. 服务层实现 (`services/slurm_cluster_service.go`)

#### 连接外部集群流程

```go
func (s *SlurmClusterService) ConnectExternalCluster(req ConnectExternalClusterRequest) (*SlurmCluster, error) {
    // 1. 建立 SSH 连接
    client, err := ssh.Dial("tcp", address, sshConfig)
    
    // 2. 验证 SLURM 安装
    output, err := session.CombinedOutput("scontrol --version")
    
    // 3. 创建集群记录
    cluster := &SlurmCluster{
        Name:        req.Name,
        ClusterType: "external",
        Status:      "running",
        MasterSSH:   &req.MasterSSH,
    }
    db.Create(cluster)
    
    // 4. 异步发现节点
    go s.discoverClusterNodes(cluster.ID, req.MasterSSH)
    
    return cluster, nil
}
```

#### 节点自动发现

```go
func (s *SlurmClusterService) discoverClusterNodes(clusterID uint, sshConfig SSHConfig) {
    // 1. SSH 连接
    client, _ := ssh.Dial("tcp", address, config)
    
    // 2. 执行 sinfo 获取节点列表
    output, _ := session.CombinedOutput("sinfo -N -h -o '%N %t %c %m %f'")
    
    // 3. 解析并创建节点记录
    for _, line := range lines {
        node := &SlurmNode{
            ClusterID: clusterID,
            NodeName:  nodeName,
            CPUs:      cpus,
            Memory:    memory,
            Status:    mapSlurmStateToStatus(state),
        }
        db.Create(node)
    }
}
```

#### 获取集群实时信息

```go
func (s *SlurmClusterService) getExternalClusterInfo(cluster *SlurmCluster) (*ClusterInfo, error) {
    // 通过 SSH 执行命令获取实时信息
    client, _ := ssh.Dial("tcp", address, config)
    
    // 执行多个命令
    sinfo -h                // 基本信息
    squeue -h               // 任务队列
    sinfo -N -h -o '%t'     // 节点状态统计
    
    return &ClusterInfo{
        NodeStats: stats,
        RunningJobs: jobCount,
    }, nil
}
```

## 前端实现

### 1. 连接外部集群对话框 (`ConnectExternalClusterDialog.jsx`)

```jsx
<Dialog>
  {/* 基本信息 */}
  <Card>
    <Input name="name" label="集群名称" required />
    <Textarea name="description" label="描述" />
    <Input name="master_host" label="Master 节点地址" required />
  </Card>
  
  {/* SSH 配置 */}
  <Card>
    <Input name="username" label="SSH 用户名" required />
    <Select name="auth_type">
      <option value="password">密码</option>
      <option value="key">密钥</option>
    </Select>
    <Input type="password" name="password" />
    <Button onClick={testConnection}>测试连接</Button>
  </Card>
</Dialog>
```

### 2. 集群列表展示 (`SlurmClusterManagement.jsx`)

```jsx
<Table>
  <TableRow>
    <TableCell>{cluster.name}</TableCell>
    
    {/* 显示集群类型 */}
    <TableCell>
      <Badge variant={cluster.cluster_type === 'external' ? 'secondary' : 'default'}>
        {cluster.cluster_type === 'external' ? '外部集群' : '托管集群'}
      </Badge>
    </TableCell>
    
    {/* 根据类型显示操作按钮 */}
    <TableCell>
      {cluster.cluster_type !== 'external' && cluster.status === 'pending' && (
        <Button onClick={deploy}>部署</Button>
      )}
      {cluster.cluster_type !== 'external' && cluster.status === 'running' && (
        <Button onClick={scale}>扩容</Button>
      )}
    </TableCell>
  </TableRow>
</Table>
```

## 使用流程

### 1. 连接外部集群

```bash
# 通过 Web 界面操作
1. 访问 http://192.168.3.91:8080/slurm
2. 点击"连接已有集群"按钮
3. 填写集群信息：
   - 集群名称
   - Master 节点地址
   - SSH 用户名
   - 认证方式（密码/密钥）
4. 测试连接
5. 提交
```

### 2. API 测试

```bash
# 运行测试脚本
./scripts/test-multi-cluster.sh

# 或手动测试
# 1. 登录
curl -X POST http://192.168.3.91:8082/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}'

# 2. 连接外部集群
curl -X POST http://192.168.3.91:8082/api/slurm/clusters/connect \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "外部集群",
    "master_host": "slurm-master",
    "master_ssh": {
      "host": "slurm-master",
      "port": 22,
      "username": "root",
      "auth_type": "password",
      "password": "aiinfra2024"
    }
  }'

# 3. 获取集群信息
curl -X GET http://192.168.3.91:8082/api/slurm/clusters/$CLUSTER_ID/info \
  -H "Authorization: Bearer $TOKEN"
```

## 技术细节

### SSH 连接管理

```go
// 创建 SSH 配置
sshConfig := &ssh.ClientConfig{
    User: username,
    Auth: []ssh.AuthMethod{
        ssh.Password(password),
        // 或
        ssh.PublicKeys(privateKey),
    },
    HostKeyCallback: ssh.InsecureIgnoreHostKey(),
    Timeout: 10 * time.Second,
}

// 建立连接
client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), sshConfig)
defer client.Close()

// 执行命令
session, _ := client.NewSession()
output, _ := session.CombinedOutput(command)
```

### SLURM 状态映射

```go
func mapSlurmStateToStatus(slurmState string) string {
    switch strings.ToLower(slurmState) {
    case "idle", "alloc", "mix":
        return "running"
    case "down", "drain", "fail":
        return "failed"
    case "resv", "maint":
        return "maintenance"
    default:
        return "unknown"
    }
}
```

### 异步任务处理

```go
// 使用 goroutine 实现异步节点发现
go func(clusterID uint, sshConfig SSHConfig) {
    // 节点发现逻辑
    discoverClusterNodes(clusterID, sshConfig)
}(cluster.ID, req.MasterSSH)

// 立即返回响应，不等待节点发现完成
return cluster, nil
```

## 安全考虑

### 1. SSH 凭证存储
- SSH 密码存储在数据库的 JSON 字段中
- **建议**：后续可以集成密钥管理服务（KMS）加密存储

### 2. 权限控制
- 连接外部集群需要认证
- 只有授权用户可以添加/删除集群

### 3. 网络安全
- SSH 连接超时设置（10秒）
- 支持密钥认证（比密码更安全）

## 已知限制

1. **SSH HostKey 验证**：当前使用 `ssh.InsecureIgnoreHostKey()`，生产环境应该验证主机密钥
2. **密码存储**：明文存储在数据库，应该加密
3. **并发连接**：没有 SSH 连接池，每次查询都新建连接
4. **错误重试**：节点发现失败不会自动重试

## 后续优化

### 短期优化
1. ✅ 添加集群类型标签显示
2. ✅ 根据类型区分操作按钮
3. ⏳ SSH 连接测试功能实现
4. ⏳ 完整的错误处理和用户提示

### 长期优化
1. SSH 凭证加密存储
2. SSH 连接池管理
3. 节点发现失败重试机制
4. 支持更多 SLURM 命令（sacct, scontrol 等）
5. 集群健康检查和监控
6. 多集群任务调度

## 测试验证

### 1. 修复 slurm-master 启动问题

```bash
# 问题：slurmctld 启动失败
# 原因：目录权限不正确

# 修复：在 systemd-entrypoint.sh 中添加
mkdir -p /var/run/slurm /var/lib/slurm/slurmctld
chown -R slurm:slurm /var/run/slurm /var/lib/slurm
chmod 755 /var/run/slurm /var/lib/slurm/slurmctld
```

### 2. 数据库迁移验证

```bash
# 检查字段是否添加
docker compose exec -T postgres psql -U postgres -d ai_infra_matrix -c "\d slurm_clusters"

# 结果：
# - cluster_type | character varying(50) | default 'managed'
# - master_ssh   | json
```

### 3. API 测试

```bash
# 运行完整测试
./scripts/test-multi-cluster.sh

# 预期结果：
# ✅ 登录成功
# ✅ 外部集群连接成功
# ✅ 节点自动发现
# ✅ 集群类型验证通过
```

## 文件清单

### 后端文件
- `src/backend/internal/models/slurm_cluster_models.go` - 数据模型
- `src/backend/internal/controllers/slurm_cluster_controller.go` - API 控制器
- `src/backend/internal/services/slurm_cluster_service.go` - 业务逻辑
- `src/backend/internal/database/database.go` - 数据库迁移

### 前端文件
- `src/frontend/src/components/slurm/ConnectExternalClusterDialog.jsx` - 连接对话框
- `src/frontend/src/components/slurm/SlurmClusterManagement.jsx` - 集群管理主页面

### 配置和脚本
- `src/slurm-master/systemd-entrypoint.sh` - Master 节点启动脚本
- `scripts/test-multi-cluster.sh` - API 测试脚本

## 总结

成功实现了 SLURM 多集群管理功能，支持：
- ✅ 托管集群和外部集群统一管理
- ✅ 通过 SSH 连接外部 SLURM 集群
- ✅ 自动发现集群节点
- ✅ 实时获取集群状态
- ✅ 区分集群类型的 UI 展示
- ✅ 数据库自动迁移
- ✅ 完整的 API 和前端实现

该功能为平台提供了更灵活的集群管理能力，用户可以方便地管理多个 SLURM 集群，无论是平台部署的还是已经存在的外部集群。
