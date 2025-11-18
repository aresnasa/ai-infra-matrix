# SLURM 节点管理功能实现计划

## 问题诊断总结

### 当前状态
1. ✅ 前端已实现 SaltStack 状态列和集成卡片显示
2. ✅ 后端已实现 `enrichNodesWithSaltStackStatus` 函数（节点状态增强逻辑）
3. ✅ Playwright E2E 测试全部通过（4 passed）
4. ⚠️ SLURM 集群中没有任何节点配置（`sinfo -N` 返回空）
5. ⚠️ 未启用 Slurm REST API 服务（`slurmrestd`）
6. ⚠️ API 返回 `{"data": null, "demo": false}`，导致 `salt_status` 字段不存在

### 问题根源
```
SLURM Master (健康) → 无节点配置 → sinfo -N 返回空
    ↓
后端 GetNodes API → 返回 null
    ↓
enrichNodesWithSaltStackStatus → 无数据可处理
    ↓
前端显示 → 节点列表为空，salt_status 字段不存在
```

## 实现方案

### 方案 1: 集成 Slurm REST API (推荐)

**优点：**
- 官方标准方案
- 支持完整的节点管理（增删改查）
- 支持节点状态操作（resume/drain/down）
- 符合用户需求文档

**实现步骤：**

#### 1. 配置 slurmrestd 服务
- 在 SLURM Master 容器中启动 `slurmrestd`
- 配置 JWT 认证
- 监听端口 6820

#### 2. 后端实现 Slurm REST API 客户端
**文件：** `src/backend/internal/clients/slurm_rest_client.go`

```go
type SlurmRestClient struct {
    baseURL string
    token   string
    client  *http.Client
}

// 节点管理方法
func (c *SlurmRestClient) CreateNode(node NodeCreateRequest) error
func (c *SlurmRestClient) GetNode(name string) (*Node, error)
func (c *SlurmRestClient) GetNodes() ([]Node, error)
func (c *SlurmRestClient) UpdateNode(name string, update NodeUpdateRequest) error
func (c *SlurmRestClient) DeleteNode(name string) error

// 节点操作方法
func (c *SlurmRestClient) ResumeNode(name string) error
func (c *SlurmRestClient) DrainNode(name string, reason string) error
func (c *SlurmRestClient) SetNodeState(name string, state string) error
```

#### 3. 后端实现节点管理 API
**文件：** `src/backend/internal/controllers/slurm_node_controller.go`

```go
// POST /api/slurm/nodes - 批量创建节点
func (c *SlurmController) CreateNodes(ctx *gin.Context)

// POST /api/slurm/nodes/:name - 创建单个节点
func (c *SlurmController) CreateNode(ctx *gin.Context)

// PUT /api/slurm/nodes/:name - 更新节点
func (c *SlurmController) UpdateNode(ctx *gin.Context)

// DELETE /api/slurm/nodes/:name - 删除节点
func (c *SlurmController) DeleteNode(ctx *gin.Context)

// POST /api/slurm/nodes/:name/actions - 节点操作
func (c *SlurmController) NodeAction(ctx *gin.Context)
```

#### 4. 前端实现节点管理 UI
**文件：** `src/frontend/src/pages/SlurmDashboard.js`

**功能模块：**
- 添加节点按钮和表单
- 节点列表操作按钮（编辑/删除/resume/drain）
- 节点状态管理弹窗
- 批量操作支持

**UI 组件：**
```javascript
// 添加节点按钮
<Button type="primary" icon={<PlusOutlined />} onClick={showAddNodeModal}>
  添加节点
</Button>

// 节点操作下拉菜单
<Dropdown menu={nodeActionsMenu}>
  <Button>操作 <DownOutlined /></Button>
</Dropdown>

// 添加节点表单
<Modal title="添加 SLURM 节点" visible={addNodeVisible}>
  <Form layout="vertical">
    <Form.Item label="节点名称" name="name">
      <Input placeholder="node01" />
    </Form.Item>
    <Form.Item label="CPU 核心数" name="cpus">
      <InputNumber min={1} />
    </Form.Item>
    <Form.Item label="内存 (MB)" name="memory">
      <InputNumber min={1} />
    </Form.Item>
    <Form.Item label="分区" name="partition">
      <Select>
        <Option value="normal">normal</Option>
        <Option value="debug">debug</Option>
      </Select>
    </Form.Item>
  </Form>
</Modal>
```

### 方案 2: 简化方案（快速验证）

**优点：**
- 实现简单
- 无需配置 slurmrestd
- 快速验证前端显示效果

**实现步骤：**

#### 1. 创建测试节点配置
在 `slurm.conf` 中添加节点定义：

```bash
# SLURM 节点配置
NodeName=test-node[01-03] CPUs=4 RealMemory=8192 State=UNKNOWN
PartitionName=normal Nodes=test-node[01-03] Default=YES MaxTime=INFINITE State=UP
```

#### 2. 重启 slurmctld 服务
```bash
docker exec ai-infra-slurm-master scontrol reconfigure
```

#### 3. 验证节点显示
- 运行 `sinfo -N` 验证节点存在
- 访问前端页面查看节点列表
- 验证 `salt_status` 字段显示

## 推荐实现顺序

### Phase 1: 快速验证（本周）
1. **配置测试节点** - 使用方案 2 创建测试节点
2. **验证前端显示** - 确认 salt_status 字段正确显示
3. **更新文档** - 记录配置步骤

### Phase 2: REST API 集成（下周）
1. **配置 slurmrestd** - 启用 Slurm REST API 服务
2. **实现后端客户端** - 创建 `slurm_rest_client.go`
3. **实现后端 API** - 创建节点管理接口
4. **单元测试** - 编写测试用例

### Phase 3: 前端功能（下下周）
1. **节点管理 UI** - 实现添加/编辑/删除节点表单
2. **节点操作** - 实现 resume/drain 等操作
3. **批量操作** - 支持批量管理节点
4. **E2E 测试** - 编写自动化测试

## 技术参考

### Slurm REST API 文档
- **Quick Start:** https://slurm.schedmd.com/rest_quickstart.html
- **API Methods:** https://slurm.schedmd.com/rest_api.html
- **版本：** v0.0.44

### 关键 API 端点
```
POST   /slurm/v0.0.44/new/node/          - 创建节点
GET    /slurm/v0.0.44/nodes/              - 获取所有节点
GET    /slurm/v0.0.44/node/{node_name}    - 获取单个节点
POST   /slurm/v0.0.44/node/{node_name}    - 更新节点
DELETE /slurm/v0.0.44/node/{node_name}    - 删除节点
```

### 认证方式
```bash
# 获取 JWT Token
unset SLURM_JWT; export $(scontrol token lifespan=3600)

# 使用 Token 调用 API
curl -H "X-SLURM-USER-TOKEN:$SLURM_JWT" \
  http://localhost:6820/slurm/v0.0.44/nodes/
```

## 风险和注意事项

### 技术风险
1. **slurmrestd 配置复杂度** - 需要配置 JWT 认证
2. **版本兼容性** - REST API 版本需要与 SLURM 版本匹配
3. **性能影响** - 大量节点时 API 调用可能较慢

### 操作风险
1. **节点误删** - 需要添加二次确认机制
2. **状态不一致** - SLURM 状态与 SaltStack 状态可能不同步
3. **权限控制** - 需要限制普通用户的节点管理权限

## 成功标准

### 功能指标
- ✅ 可以通过 Web 界面添加 SLURM 节点
- ✅ 可以查看节点列表和详细信息
- ✅ 可以更新节点配置（CPU、内存等）
- ✅ 可以删除节点
- ✅ 可以执行节点操作（resume、drain、down）
- ✅ `salt_status` 字段正确显示 SaltStack minion 状态

### 性能指标
- ⚡ 节点列表加载时间 < 3s（100 个节点）
- ⚡ 节点操作响应时间 < 2s
- ⚡ 节点状态刷新间隔 < 30s

### 测试覆盖
- ✅ 单元测试覆盖率 > 80%
- ✅ E2E 测试覆盖核心流程
- ✅ API 集成测试全部通过

## 下一步行动

### 阶段 0: 修复 AppHub（必须先完成）

**问题诊断：**
- ❌ AppHub 构建的 SLURM 包中不包含 `slurmrestd`
- ❌ AppHub Dockerfile 在提取 SLURM 二进制时只包含了客户端命令
- ❌ SLURM Master 容器中没有 `slurmrestd` 可执行文件

**修复步骤：**
1. **修改 AppHub Dockerfile** (`src/apphub/Dockerfile` 第 778 行)
   - 当前仅提取：`sinfo squeue scontrol scancel sbatch srun salloc sacct sacctmgr`
   - 需要添加：`slurmrestd slurmd slurmctld slurmdbd`
   
2. **重新构建 AppHub**
   ```bash
   docker-compose build --no-cache apphub
   docker-compose up -d apphub
   ```

3. **重新构建 SLURM Master**
   ```bash
   docker-compose build --no-cache slurm-master
   docker-compose up -d slurm-master
   ```

4. **验证 slurmrestd 安装**
   ```bash
   docker exec ai-infra-slurm-master which slurmrestd
   docker exec ai-infra-slurm-master slurmrestd --version
   ```

### 阶段 1: 配置 slurmrestd 服务（今天）

1. **配置 JWT 认证**
   - [ ] 在 SLURM Master 中配置 JWT 密钥
   - [ ] 配置 `slurmrestd` systemd 服务
   - [ ] 测试 JWT token 生成

2. **启动 slurmrestd**
   - [ ] 创建 `slurmrestd.service` systemd 配置
   - [ ] 配置监听端口 6820
   - [ ] 验证服务健康状态

3. **测试 REST API**
   - [ ] 获取 JWT token
   - [ ] 测试 GET /slurm/v0.0.44/ping
   - [ ] 测试 GET /slurm/v0.0.44/diag
   - [ ] 测试 GET /slurm/v0.0.44/nodes

### 阶段 2: 实现后端集成（本周）

1. **后端 REST API 客户端**
   - [ ] 创建 `slurm_rest_client.go`
   - [ ] 实现 JWT token 管理
   - [ ] 实现节点查询方法

2. **后端节点管理 API**
   - [ ] POST /api/slurm/nodes - 创建节点
   - [ ] GET /api/slurm/nodes - 获取节点列表
   - [ ] PUT /api/slurm/nodes/:name - 更新节点
   - [ ] DELETE /api/slurm/nodes/:name - 删除节点

3. **测试和文档**
   - [ ] 编写单元测试
   - [ ] 编写 API 文档
   - [ ] 更新 `dev-md.md`

### 阶段 3: 前端实现（下周）

1. **节点管理 UI**
   - [ ] 添加节点按钮和表单
   - [ ] 节点列表操作菜单
   - [ ] 节点状态管理

2. **E2E 测试**
   - [ ] 编写添加节点测试
   - [ ] 编写删除节点测试
   - [ ] 验证 salt_status 显示

3. **发布**
   - [ ] 更新版本号
   - [ ] 发布 Release Notes
