# SLURM 节点删除功能测试指南

## 功能概述

新增了SLURM节点删除功能，支持通过前端界面或API删除集群中的节点。删除操作会：
1. 停止节点上的SLURMD和Munge服务（如果可访问）
2. 从数据库中删除节点记录
3. 可选择强制删除（跳过停止服务步骤）

## 后端API

### 1. 删除节点
```http
DELETE /api/slurm/nodes/:nodeId?force=false
```

**参数**：
- `nodeId` (路径参数): 节点ID
- `force` (查询参数): 是否强制删除，默认false

**响应示例**：
```json
{
  "success": true,
  "message": "节点删除成功"
}
```

### 2. 获取节点详情
```http
GET /api/slurm/nodes/:nodeId
```

**响应示例**：
```json
{
  "success": true,
  "data": {
    "id": 123,
    "cluster_id": 1,
    "node_name": "test-rocky02",
    "host": "192.168.3.100",
    "port": 22,
    "status": "idle"
  }
}
```

### 3. 列出集群节点
```http
GET /api/slurm/nodes/cluster/:clusterId
```

**响应示例**：
```json
{
  "success": true,
  "data": [
    {
      "id": 123,
      "node_name": "test-rocky02",
      "status": "idle"
    }
  ],
  "count": 1
}
```

## 前端功能

### 界面位置
访问 `http://192.168.3.91:8080/slurm` → "节点管理" 标签页

### 删除单个节点

1. 在节点列表中找到要删除的节点
2. 点击该节点行右侧的"删除"按钮（红色减号图标）
3. 在确认对话框中查看删除说明
4. 点击"确定"执行删除

**确认对话框内容**：
```
确定要删除节点 test-rocky02 吗？
这将停止节点上的服务并从数据库中移除记录
```

### 批量删除节点（待实现）

未来可以支持：
1. 勾选多个节点
2. 点击"节点操作"下拉菜单
3. 选择"删除选中节点"

## 测试场景

### 场景1: 删除正常运行的节点

**前置条件**：
- 节点正在运行且可SSH访问
- munged和slurmd服务正常运行

**测试步骤**：
```bash
# 1. 检查节点状态
docker exec ai-infra-slurm-master sinfo -N | grep test-rocky02

# 2. 通过API删除节点
curl -X DELETE "http://localhost:8082/api/slurm/nodes/123" \
  -H "Authorization: Bearer YOUR_TOKEN"

# 3. 验证服务已停止
docker exec test-rocky02 ps aux | egrep 'slurmd|munged'

# 4. 验证节点已从数据库删除
curl "http://localhost:8082/api/slurm/nodes/123" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

**预期结果**：
- ✅ 节点服务被停止
- ✅ 数据库记录被删除
- ✅ 前端列表中不再显示该节点
- ✅ sinfo命令不再显示该节点

### 场景2: 强制删除不可访问的节点

**前置条件**：
- 节点已关机或网络不可达
- 无法SSH连接到节点

**测试步骤**：
```bash
# 1. 停止节点容器模拟不可达
docker stop test-rocky02

# 2. 尝试普通删除（会警告但仍删除）
curl -X DELETE "http://localhost:8082/api/slurm/nodes/123" \
  -H "Authorization: Bearer YOUR_TOKEN"

# 或使用强制删除
curl -X DELETE "http://localhost:8082/api/slurm/nodes/123?force=true" \
  -H "Authorization: Bearer YOUR_TOKEN"

# 3. 验证数据库记录已删除
curl "http://localhost:8082/api/slurm/nodes/123" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

**预期结果**：
- ✅ 跳过停止服务步骤（因为不可访问）
- ✅ 数据库记录仍被删除
- ⚠️ 日志中记录警告信息

### 场景3: 通过前端删除节点

**测试步骤**：
1. 打开浏览器访问 `http://192.168.3.91:8080/slurm`
2. 点击"节点管理"标签
3. 找到 `test-rocky03` 节点
4. 点击该行右侧的红色"删除"按钮
5. 阅读确认对话框
6. 点击"确定"

**预期结果**：
- ✅ 显示确认对话框，说明删除操作的影响
- ✅ 删除成功后显示提示消息
- ✅ 节点列表自动刷新，不再显示已删除节点
- ✅ 如果删除失败，显示错误消息

### 场景4: 删除不存在的节点

**测试步骤**：
```bash
curl -X DELETE "http://localhost:8082/api/slurm/nodes/99999" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

**预期响应**：
```json
{
  "success": false,
  "error": "删除节点失败: 节点不存在: record not found"
}
```

### 场景5: 完整工作流测试

**测试步骤**：
```bash
# 1. 添加节点
curl -X POST "http://localhost:8082/api/slurm/nodes/scale" \
  -H "Content-Type: application/json" \
  -d '{
    "cluster_id": 1,
    "node_names": ["test-new-node"]
  }'

# 2. 等待节点初始化完成
sleep 10

# 3. 列出节点
curl "http://localhost:8082/api/slurm/nodes/cluster/1"

# 4. 获取节点详情
NODE_ID=$(curl "http://localhost:8082/api/slurm/nodes/cluster/1" | jq '.data[] | select(.node_name=="test-new-node") | .id')

# 5. 删除节点
curl -X DELETE "http://localhost:8082/api/slurm/nodes/$NODE_ID"

# 6. 验证删除
curl "http://localhost:8082/api/slurm/nodes/$NODE_ID"  # 应该返回404
```

## 错误处理

### 常见错误

**错误1: 无效的节点ID**
```json
{
  "success": false,
  "error": "无效的节点ID"
}
```
**原因**: 提供的节点ID不是有效的数字
**解决**: 检查URL中的nodeId参数

**错误2: 节点不存在**
```json
{
  "success": false,
  "error": "删除节点失败: 节点不存在: record not found"
}
```
**原因**: 数据库中没有该节点记录
**解决**: 使用正确的节点ID或先查询节点列表

**错误3: SSH连接失败**
```
日志中显示: "停止节点服务失败（继续删除）: 创建SSH连接失败: ..."
```
**原因**: 无法连接到节点
**影响**: 节点上的服务未停止，但数据库记录已删除
**解决**: 
- 检查节点网络连通性
- 验证SSH凭据
- 或使用 `force=true` 参数跳过服务停止

## 日志查看

### 后端日志
```bash
# 查看删除节点的日志
docker logs -f ai-infra-backend 2>&1 | grep -i "删除节点\|DeleteNode"

# 示例输出
[INFO] 开始删除节点: test-rocky02 (ID: 123)
[INFO] 停止节点 test-rocky02 的服务
[INFO] 节点 test-rocky02 服务已停止
[INFO] 节点删除成功: test-rocky02
```

### 前端控制台
```javascript
// 打开浏览器开发者工具 - Console
// 查看API调用
DELETE /api/slurm/nodes/123
Response: { success: true, message: "节点删除成功" }
```

## 性能指标

| 操作 | 预期时间 | 说明 |
|------|----------|------|
| 删除单个节点（可访问） | 3-5秒 | 包括SSH连接和服务停止 |
| 删除单个节点（不可访问） | 1-2秒 | 跳过服务停止，直接删除记录 |
| 强制删除 | < 1秒 | 仅删除数据库记录 |

## 安全性

### 权限控制
- ✅ 需要用户登录认证
- ✅ 使用 `AuthMiddlewareWithSession()` 中间件
- ✅ 验证用户身份

### 审计日志
- ✅ 记录删除操作的用户ID
- ✅ 记录节点名称和ID
- ✅ 记录操作时间

## 回滚操作

如果误删除了节点，可以通过以下步骤恢复：

1. **重新添加节点**
   ```bash
   # 通过API重新添加
   curl -X POST "http://localhost:8082/api/slurm/clusters/1/init-node" \
     -d '{"node_id": NEW_ID, "install_packages": true}'
   ```

2. **从备份恢复**
   ```bash
   # 如果有数据库备份
   docker exec ai-infra-postgres pg_restore -d slurm backup.sql
   ```

3. **手动重新初始化**
   ```bash
   ./scripts/manage-slurm-nodes.sh init test-rocky02
   ```

## 待优化功能

### 短期优化
- [ ] 添加批量删除支持
- [ ] 删除前检查节点是否有运行中的作业
- [ ] 添加删除操作的二次确认

### 长期优化
- [ ] 软删除（标记为已删除而非物理删除）
- [ ] 删除操作的撤销功能
- [ ] 集成到审计日志系统
- [ ] 删除节点时自动更新slurm.conf

## 总结

删除节点功能已完全实现，包括：
✅ 后端API（3个新端点）
✅ 服务层逻辑（停止服务 + 删除记录）
✅ 前端UI（删除按钮 + 确认对话框）
✅ 错误处理和日志记录

可以通过前端界面或API安全地删除SLURM节点。
