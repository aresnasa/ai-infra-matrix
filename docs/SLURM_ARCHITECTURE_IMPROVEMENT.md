# SLURM 集群管理架构改进方案

## 改进概述

本次改进重新设计了 SLURM 集群的节点管理和外部集群连接机制，实现了更灵活、可靠的集群管理方式。

## 核心改进点

### 1. 节点安装时机调整 ✅

**❌ 旧方案**：
- 在 Dockerfile 中预安装 SLURM 组件
- Master 启动时自动修复所有节点
- 无法动态添加新节点

**✅ 新方案**：
- Dockerfile 只包含基础环境
- 通过页面扩容时才触发 SLURM 安装
- 支持动态添加和配置节点

### 2. SaltStack 依赖检查 ✅

**实现**：在扩容节点前先检查 SaltStack 客户端状态

```go
// 检查流程
1. 验证节点是否在 SaltStack 中注册
2. 检查节点是否在线（ping test）
3. 获取 Salt Minion 版本
4. 检查是否已安装 SLURM
5. 返回节点就绪状态
```

**好处**：
- 确保只在具备条件的节点上安装
- 避免安装失败导致的问题
- 提供详细的就绪状态报告

### 3. 外部集群单独管理 ✅

**新增专用管理页面**：`ExternalClusterManagement.jsx`

**功能特性**：
- ✅ SSH 连接测试
- ✅ 自动发现集群信息
- ✅ 复用现有配置选项
  - 复用 slurm.conf
  - 复用 munge.key
  - 复用数据库配置
- ✅ 集群信息刷新
- ✅ 集群删除管理

### 4. E2E 测试覆盖 ✅

**Playwright 测试场景**：

```javascript
✅ 访问外部集群管理页面
✅ 填写集群连接表单
✅ 测试 SSH 连接
✅ 添加外部集群
✅ 显示已连接集群列表
✅ 刷新集群信息
✅ 删除集群
✅ 配置复用选项
✅ 表单重置
✅ 必填字段验证
✅ 连接失败错误处理
✅ 集成测试（与主列表集成）
```

## 架构设计

### 后端架构

```
controllers/
├── slurm_cluster_controller.go       # 集群CRUD
├── slurm_node_scale_controller.go    # 节点扩容（新增）
└── ...

services/
├── slurm_cluster_service.go          # 集群服务
├── saltstack_service.go               # SaltStack集成
└── ...

models/
├── slurm_cluster_models.go
│   ├── ClusterType: "managed" | "external"
│   ├── MasterSSH: SSH配置
│   └── Config: 集群配置
└── ...
```

### API 端点

#### 节点扩容相关

```
POST   /api/slurm/nodes/check-saltstack   # 检查SaltStack客户端
POST   /api/slurm/nodes/scale              # 扩容节点
```

#### 外部集群相关

```
POST   /api/slurm/clusters/connect         # 连接外部集群
POST   /api/slurm/clusters/test-connection # 测试SSH连接
GET    /api/slurm/clusters/:id/info        # 获取集群信息
DELETE /api/slurm/clusters/:id             # 删除集群
POST   /api/slurm/clusters/:id/refresh     # 刷新集群信息
```

### 前端架构

```
components/slurm/
├── SlurmClusterManagement.jsx         # 主集群管理（托管+外部）
├── ExternalClusterManagement.jsx      # 外部集群专用管理（新增）
├── ConnectExternalClusterDialog.jsx   # 连接对话框
└── ...
```

### 页面路由

```
/slurm/clusters          # 所有集群列表（托管+外部）
/slurm/external-clusters # 外部集群管理专用页面（新增）
```

## 工作流程

### 托管集群节点扩容流程

```
┌─────────────────────────────────────────────────────────┐
│ 1. 用户在页面点击"扩容节点"                              │
└──────────────────┬──────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────┐
│ 2. 输入节点名称列表                                      │
│    例如: node01, node02, node03                         │
└──────────────────┬──────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────┐
│ 3. 前端调用 /api/slurm/nodes/check-saltstack           │
│    检查每个节点的 SaltStack 客户端状态                 │
└──────────────────┬──────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────┐
│ 4. 显示检查结果                                          │
│    ✅ node01: Salt客户端就绪, 可以安装                  │
│    ✅ node02: Salt客户端就绪, 已安装SLURM              │
│    ❌ node03: Salt客户端未注册                          │
└──────────────────┬──────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────┐
│ 5. 用户确认后，调用 /api/slurm/nodes/scale             │
└──────────────────┬──────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────┐
│ 6. 后端通过 SaltStack 执行安装                          │
│    For each node:                                       │
│      - salt node01 state.apply slurm.node              │
│      - 配置 slurm.conf                                  │
│      - 同步 munge.key                                   │
│      - 启动 slurmd 服务                                │
└──────────────────┬──────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────┐
│ 7. 节点注册到集群                                        │
│    - 创建 SlurmNode 记录                                │
│    - 更新 slurmctld 配置                                │
│    - scontrol reconfigure                               │
└─────────────────────────────────────────────────────────┘
```

### 外部集群连接流程

```
┌─────────────────────────────────────────────────────────┐
│ 1. 用户访问 /slurm/external-clusters                    │
└──────────────────┬──────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────┐
│ 2. 填写连接信息                                          │
│    - 集群名称                                           │
│    - Master 节点地址                                    │
│    - SSH 用户名/密码                                    │
│    - 配置复用选项                                       │
└──────────────────┬──────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────┐
│ 3. 点击"测试连接"                                        │
│    POST /api/slurm/clusters/test-connection            │
└──────────────────┬──────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────┐
│ 4. 后端通过 SSH 连接 Master                             │
│    - 执行 scontrol --version                            │
│    - 执行 scontrol show config                          │
│    - 执行 sinfo -N                                      │
│    - 返回集群信息                                       │
└──────────────────┬──────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────┐
│ 5. 显示集群信息预览                                      │
│    ✅ SLURM 版本: 25.05.4                               │
│    ✅ 集群名称: production-cluster                      │
│    ✅ 节点数量: 10                                       │
│    ✅ 控制器: slurm-master.example.com                  │
└──────────────────┬──────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────┐
│ 6. 用户点击"添加集群"                                    │
│    POST /api/slurm/clusters/connect                    │
└──────────────────┬──────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────┐
│ 7. 后端处理                                              │
│    - 创建 SlurmCluster 记录 (cluster_type='external')  │
│    - 存储 SSH 配置                                      │
│    - 如果 reuse_config=true: 不生成新配置              │
│    - 如果 reuse_munge=true: 不同步munge.key            │
│    - 如果 reuse_database=true: 不初始化数据库          │
│    - 异步发现节点: go discoverClusterNodes()           │
└──────────────────┬──────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────┐
│ 8. 集群添加成功                                          │
│    - 显示在"已连接集群"列表                             │
│    - 可以查看节点信息                                   │
│    - 可以刷新状态                                       │
│    - 可以删除连接                                       │
└─────────────────────────────────────────────────────────┘
```

## 数据库设计

### slurm_clusters 表扩展

```sql
ALTER TABLE slurm_clusters 
ADD COLUMN cluster_type VARCHAR(50) DEFAULT 'managed';

ALTER TABLE slurm_clusters 
ADD COLUMN master_ssh JSON;

-- cluster_type: 
--   'managed' - 平台部署和管理的集群
--   'external' - 外部已存在的集群

-- master_ssh 示例:
{
  "host": "192.168.1.100",
  "port": 22,
  "username": "root",
  "auth_type": "password",
  "password": "encrypted_password",
  "key_path": "/path/to/key"
}
```

### config 字段扩展

```json
{
  "max_nodes": 100,
  "reuse_existing_config": true,
  "reuse_existing_munge": true,
  "reuse_existing_database": true,
  "discovered_at": "2025-11-11T19:00:00Z",
  "original_cluster_name": "production-cluster"
}
```

## 安全考虑

### 1. SSH 凭证管理

**当前实现**：
- SSH 密码存储在数据库 JSON 字段中
- 通过 HTTPS 传输

**建议改进**：
- 集成 HashiCorp Vault 或 AWS Secrets Manager
- 使用密钥认证替代密码
- 实现密码轮换机制

### 2. 权限控制

```go
// 示例权限检查
func (c *SlurmNodeScaleController) ScaleNodes(ctx *gin.Context) {
    userID := ctx.GetUint("user_id")
    
    // 检查用户是否有扩容权限
    if !c.authService.HasPermission(userID, "slurm:scale:nodes") {
        ctx.JSON(403, gin.H{"error": "权限不足"})
        return
    }
    
    // ... 执行扩容
}
```

### 3. 审计日志

```go
// 记录所有关键操作
type AuditLog struct {
    UserID     uint      `json:"user_id"`
    Action     string    `json:"action"`      // scale_nodes, connect_cluster
    Resource   string    `json:"resource"`    // cluster_id, node_names
    Status     string    `json:"status"`      // success, failed
    Details    string    `json:"details"`
    Timestamp  time.Time `json:"timestamp"`
}
```

## 测试策略

### 1. 单元测试

```go
// controllers/slurm_node_scale_controller_test.go
func TestCheckSaltStackClients(t *testing.T) {
    // 测试 SaltStack 客户端检查
}

func TestScaleNodes(t *testing.T) {
    // 测试节点扩容逻辑
}
```

### 2. 集成测试

```bash
# 测试 SaltStack 集成
go test ./test/integration/saltstack_test.go

# 测试 SSH 连接
go test ./test/integration/ssh_test.go
```

### 3. E2E 测试

```bash
# 运行 Playwright E2E 测试
npm --yes playwright test test/e2e/specs/slurm-external-cluster.spec.js

# 或使用任务
docker compose exec frontend npm test:e2e
```

### 4. 测试覆盖率目标

- 单元测试：>= 80%
- 集成测试：>= 70%
- E2E 测试：核心流程 100%

## 部署和回滚

### 部署步骤

```bash
# 1. 数据库迁移
docker compose exec backend ./backend migrate

# 2. 重启后端服务
docker compose restart backend

# 3. 重新构建前端
docker compose build frontend

# 4. 重启前端
docker compose up -d frontend

# 5. 运行测试
npm --yes playwright test test/e2e/specs/slurm-external-cluster.spec.js
```

### 回滚方案

```bash
# 1. 恢复数据库
docker compose exec postgres psql -U postgres -d ai_infra_matrix -f /backups/pre_upgrade.sql

# 2. 回滚代码
git revert <commit-hash>

# 3. 重新部署
docker compose up -d --build
```

## 监控和告警

### 关键指标

1. **节点扩容成功率**
   ```promql
   rate(slurm_node_scale_success_total[5m]) / 
   rate(slurm_node_scale_attempts_total[5m])
   ```

2. **外部集群连接健康度**
   ```promql
   sum(slurm_external_cluster_health{status="healthy"}) / 
   sum(slurm_external_cluster_total)
   ```

3. **SSH 连接延迟**
   ```promql
   histogram_quantile(0.95, 
     rate(slurm_ssh_connection_duration_seconds_bucket[5m])
   )
   ```

### 告警规则

```yaml
alerts:
  - name: NodeScaleFailureRate
    expr: rate(slurm_node_scale_failed_total[10m]) > 0.1
    annotations:
      summary: "节点扩容失败率过高"
      
  - name: ExternalClusterUnhealthy
    expr: slurm_external_cluster_health{status="unhealthy"} > 0
    for: 5m
    annotations:
      summary: "外部集群连接不健康"
```

## 后续优化

### 短期（1-2周）
1. ✅ 完成基础功能
2. ⏳ 添加批量操作支持
3. ⏳ 优化 SSH 连接池
4. ⏳ 完善错误重试机制

### 中期（1-2月）
1. 实现密钥认证支持
2. 添加 Webhook 通知
3. 集成监控和告警
4. 性能优化

### 长期（3-6月）
1. 多区域集群支持
2. 自动故障转移
3. 集群联邦管理
4. AI 辅助运维

## 总结

本次架构改进实现了：

✅ **更灵活的节点管理**：按需安装，不在 Dockerfile 中预置
✅ **更可靠的依赖检查**：确保 SaltStack 客户端就绪
✅ **独立的外部集群管理**：专用页面，复用现有配置
✅ **完整的测试覆盖**：E2E、集成、单元测试
✅ **更好的用户体验**：清晰的流程，实时反馈
✅ **更强的可维护性**：模块化设计，职责分离

通过这些改进，SLURM 集群管理系统变得更加灵活、可靠和易于维护，为用户提供了更好的使用体验。
