# SLURM 双模式配置指南

## 概述

SLURM 集群管理现在支持两种操作模式：
1. **SSH 模式**（默认）：通过 SSH 连接到 SLURM Master 执行 `scontrol`/`scancel` 命令
2. **REST API 模式**：通过 `slurmrestd` REST API 执行操作

## 功能支持

### 节点管理
- ✅ 恢复节点 (RESUME)
- ✅ 排空节点 (DRAIN)
- ✅ 下线节点 (DOWN)
- ✅ 设置空闲 (IDLE)
- ✅ 节点电源管理 (POWER_DOWN/POWER_UP)

### 作业管理
- ✅ 取消作业 (CANCEL)
- ✅ 暂停调度 (HOLD)
- ✅ 释放作业 (RELEASE)
- ✅ 挂起作业 (SUSPEND)
- ✅ 恢复作业 (RESUME)
- ✅ 重新入队 (REQUEUE)

## 环境变量配置

### USE_SLURMRESTD
控制是否启用 `slurmrestd` REST API 模式。

**默认值：** `false` (使用 SSH 模式)

**可选值：**
- `true` - 启用 REST API 模式
- `false` - 使用 SSH 模式

### SLURM_REST_API_URL
`slurmrestd` 服务的 URL。

**默认值：** `http://slurm-master:6820`

**示例：**
```bash
SLURM_REST_API_URL=http://192.168.0.200:6820
```

### SLURM_JWT_TOKEN
可选的预设 JWT Token。如果不提供，系统会通过 SSH 自动获取。

**示例：**
```bash
SLURM_JWT_TOKEN=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### SSH 相关配置（SSH 模式需要）
- `SLURM_MASTER_HOST` - SLURM Master 主机地址（默认：`slurm-master`）
- `SLURM_MASTER_PORT` - SSH 端口（默认：`22`）
- `SLURM_MASTER_USER` - SSH 用户名（默认：`root`）
- `SLURM_SSH_KEY_PATH` - SSH 私钥路径（默认：`/root/.ssh/id_rsa`）

## Docker Compose 配置示例

### 场景 1: 使用 SSH 模式（默认）

```yaml
services:
  backend:
    image: ai-infra-backend:latest
    environment:
      # SSH 模式配置（默认）
      USE_SLURMRESTD: "false"
      SLURM_MASTER_HOST: "192.168.0.200"
      SLURM_MASTER_PORT: "22"
      SLURM_MASTER_USER: "root"
      SLURM_SSH_KEY_PATH: "/root/.ssh/id_rsa"
    volumes:
      - ./ssh-key:/root/.ssh:ro
```

### 场景 2: 使用 REST API 模式

```yaml
services:
  backend:
    image: ai-infra-backend:latest
    environment:
      # REST API 模式配置
      USE_SLURMRESTD: "true"
      SLURM_REST_API_URL: "http://192.168.0.200:6820"
      # 可选：提供预设的 JWT Token
      SLURM_JWT_TOKEN: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
      # 如果不提供 Token，仍需要 SSH 配置来获取 Token
      SLURM_MASTER_HOST: "192.168.0.200"
      SLURM_MASTER_USER: "root"
      SLURM_SSH_KEY_PATH: "/root/.ssh/id_rsa"
```

### 场景 3: REST API 模式 + 预设 Token（推荐）

```yaml
services:
  backend:
    image: ai-infra-backend:latest
    environment:
      # REST API 模式配置
      USE_SLURMRESTD: "true"
      SLURM_REST_API_URL: "http://192.168.0.200:6820"
      SLURM_JWT_TOKEN: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
      # 不需要 SSH 配置
```

## 启用 slurmrestd 服务

### 1. 在 SLURM Master 上安装 slurmrestd

```bash
# CentOS/RHEL
yum install slurm-slurmrestd

# Ubuntu/Debian
apt-get install slurmrestd
```

### 2. 配置 slurmrestd

编辑 `/etc/slurm/slurm.conf`，添加：

```conf
# REST API 配置
AuthType=auth/jwt
AuthAltTypes=auth/jwt
AuthAltParameters=jwt_key=/etc/slurm/jwt_hs256.key
```

### 3. 生成 JWT 密钥

```bash
dd if=/dev/urandom bs=32 count=1 > /etc/slurm/jwt_hs256.key
chmod 600 /etc/slurm/jwt_hs256.key
chown slurm:slurm /etc/slurm/jwt_hs256.key
```

### 4. 启动 slurmrestd

```bash
# 使用 systemd
systemctl enable slurmrestd
systemctl start slurmrestd

# 或手动启动
slurmrestd -vvv 0.0.0.0:6820
```

### 5. 获取 JWT Token

```bash
# 方法 1: 使用 scontrol
scontrol token

# 方法 2: 使用 OpenAPI
curl -H "X-SLURM-USER-NAME:root" -H "X-SLURM-USER-TOKEN:$(dd if=/dev/urandom bs=32 count=1 | base64)" \
  http://localhost:6820/slurm/v0.0.40/diag
```

## 测试验证

### 1. 测试 SSH 模式

```bash
# 设置环境变量
export USE_SLURMRESTD=false

# 重启 Backend
docker-compose restart backend

# 测试节点操作
curl -X POST http://localhost:8080/api/slurm/nodes/manage \
  -H "Content-Type: application/json" \
  -d '{
    "node_names": ["node01"],
    "action": "resume"
  }'
```

### 2. 测试 REST API 模式

```bash
# 设置环境变量
export USE_SLURMRESTD=true
export SLURM_REST_API_URL=http://192.168.0.200:6820

# 重启 Backend
docker-compose restart backend

# 测试作业操作
curl -X POST http://localhost:8080/api/slurm/jobs/manage \
  -H "Content-Type: application/json" \
  -d '{
    "job_ids": ["123"],
    "action": "hold"
  }'
```

### 3. 前端测试

1. 访问 http://localhost:8080/slurm
2. 在"节点管理"标签页选择节点，点击"节点操作"
3. 在"作业队列"标签页选择作业，点击"作业操作"
4. 检查操作是否成功执行

## 工作原理

### 代码流程

```
┌─────────────────────────────────────────────────────────────┐
│                        Frontend                              │
│  (SlurmScalingPage.js)                                      │
│  - 节点操作按钮                                              │
│  - 作业操作按钮                                              │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      │ HTTP POST
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                   Backend Controller                         │
│  (slurm_controller.go)                                      │
│  - ManageNodes(ctx)                                         │
│  - ManageJobs(ctx)                                          │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      │ 检查 useSlurmrestd 配置
                      ▼
         ┌────────────┴────────────┐
         │                         │
    REST API 模式              SSH 模式
         │                         │
         ▼                         ▼
┌────────────────────┐    ┌────────────────────┐
│ UpdateNodeViaAPI   │    │ ExecuteSlurmCommand│
│ CancelJobViaAPI    │    │ (scontrol/scancel) │
│ UpdateJobViaAPI    │    │                    │
└─────────┬──────────┘    └─────────┬──────────┘
          │                         │
          │                         │
          ▼                         ▼
┌─────────────────────┐    ┌────────────────────┐
│  slurmrestd API     │    │  SSH + Commands    │
│  (http://...:6820)  │    │  (node:22)         │
└─────────────────────┘    └────────────────────┘
```

### 模式选择逻辑

```go
if c.slurmSvc.GetUseSlurmRestd() {
    // 使用 REST API 方式
    for _, nodeName := range req.NodeNames {
        update := services.SlurmNodeUpdate{
            State:  slurmState,
            Reason: req.Reason,
        }
        err := c.slurmSvc.UpdateNodeViaAPI(ctx, nodeName, update)
        // 处理错误
    }
} else {
    // 使用 SSH 方式
    command := fmt.Sprintf("scontrol update NodeName=%s State=%s", nodeList, slurmState)
    output, err := c.slurmSvc.ExecuteSlurmCommand(ctx, command)
    // 处理错误
}
```

## 故障排查

### 问题 1: REST API 模式下操作失败

**症状：** 节点/作业操作返回错误

**排查步骤：**
1. 检查 `slurmrestd` 服务是否运行
   ```bash
   systemctl status slurmrestd
   ```

2. 检查防火墙规则
   ```bash
   firewall-cmd --list-ports
   # 应该包含 6820/tcp
   ```

3. 检查 JWT Token 是否有效
   ```bash
   scontrol token
   ```

4. 查看 Backend 日志
   ```bash
   docker-compose logs backend | grep "UpdateNodeViaAPI\|CancelJobViaAPI"
   ```

### 问题 2: SSH 模式下操作失败

**症状：** 节点/作业操作返回"执行命令失败"

**排查步骤：**
1. 检查 SSH 连接
   ```bash
   ssh -i /path/to/key root@slurm-master
   ```

2. 检查 SSH 密钥权限
   ```bash
   chmod 600 /root/.ssh/id_rsa
   ```

3. 手动测试 SLURM 命令
   ```bash
   ssh root@slurm-master "scontrol show nodes"
   ```

### 问题 3: 模式切换不生效

**症状：** 修改环境变量后仍使用旧模式

**解决方案：**
1. 确认环境变量已在 docker-compose.yml 中设置
2. 重启 Backend 容器
   ```bash
   docker-compose restart backend
   ```

3. 检查日志确认配置已加载
   ```bash
   docker-compose logs backend | grep "useSlurmrestd"
   ```

## 最佳实践

### 生产环境建议

1. **使用 REST API 模式**
   - 性能更好（减少 SSH 连接开销）
   - 更安全（使用 JWT Token 而非 SSH 密钥）
   - 支持更细粒度的权限控制

2. **提供预设 JWT Token**
   - 避免每次都通过 SSH 获取 Token
   - 定期轮换 Token（建议每月一次）

3. **监控 slurmrestd 服务**
   - 添加健康检查
   - 监控 API 响应时间
   - 设置日志级别为 WARNING

### 开发/测试环境建议

1. **使用 SSH 模式**
   - 配置简单
   - 调试方便
   - 不需要配置 slurmrestd

2. **启用详细日志**
   ```yaml
   environment:
     LOG_LEVEL: "DEBUG"
   ```

## 版本历史

- **v0.3.8** (2024-01)
  - ✅ 添加 REST API 模式支持
  - ✅ 实现节点管理双模式
  - ✅ 实现作业管理双模式
  - ✅ 前端 UI 完全支持

- **v0.3.6** (2023-12)
  - 仅支持 SSH 模式

## 相关文档

- [SLURM REST API 官方文档](https://slurm.schedmd.com/rest_api.html)
- [JWT Token 配置指南](https://slurm.schedmd.com/jwt.html)
- [节点管理操作说明](./NODE_MANAGEMENT.md)
- [作业队列管理说明](./JOB_MANAGEMENT.md)

## 支持与反馈

如有问题或建议，请联系：
- 技术支持：support@example.com
- 问题反馈：https://github.com/example/issues
