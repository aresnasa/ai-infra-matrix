# SLURM 命令执行 UI 测试指南

## 概述

本测试套件验证 SLURM Dashboard 中新增的 SaltStack 命令执行和集群状态监控功能。

## 新增组件

### 1. SaltCommandExecutor 组件
- **路径**: `src/frontend/src/components/SaltCommandExecutor.js`
- **功能**: 
  - 命令执行表单
  - 命令历史记录
  - 最近作业列表
  - 命令详情展示

### 2. SlurmClusterStatus 组件
- **路径**: `src/frontend/src/components/SlurmClusterStatus.js`
- **功能**:
  - 集群健康度监控
  - 节点状态统计
  - 资源使用情况
  - 分区信息展示
  - SaltStack 集成状态

### 3. SlurmDashboard 增强
- **路径**: `src/frontend/src/pages/SlurmDashboard.js`
- **改进**:
  - Tab 架构（集群概览、集群状态监控、SaltStack 命令执行）
  - 集成新组件

## 测试文件

- **测试文件**: `test/e2e/specs/slurm-command-execution-ui.spec.js`
- **测试数量**: 11 个测试用例
- **测试套件**: 2 个（UI 功能测试、命令执行流程测试）

## 运行测试

### 前提条件

1. **启动服务**:
   ```bash
   # 确保所有服务运行
   docker-compose up -d
   
   # 检查后端服务
   curl http://192.168.18.154:8080/api/health
   ```

2. **配置环境变量**:
   ```bash
   export BASE_URL=http://192.168.18.154:8080
   ```

### 运行完整测试套件

```bash
cd test/e2e
BASE_URL=http://192.168.18.154:8080 npx playwright test specs/slurm-command-execution-ui.spec.js --reporter=list
```

### 运行特定测试

```bash
# 只测试 Tab 结构
BASE_URL=http://192.168.18.154:8080 npx playwright test specs/slurm-command-execution-ui.spec.js -g "验证 SLURM Dashboard Tab 结构"

# 只测试集群状态监控
BASE_URL=http://192.168.18.154:8080 npx playwright test specs/slurm-command-execution-ui.spec.js -g "验证集群状态监控 Tab 功能"

# 只测试命令执行
BASE_URL=http://192.168.18.154:8080 npx playwright test specs/slurm-command-execution-ui.spec.js -g "验证 SaltStack 命令执行 Tab 功能"
```

### 调试模式

```bash
# 使用 UI 模式（可视化调试）
BASE_URL=http://192.168.18.154:8080 npx playwright test specs/slurm-command-execution-ui.spec.js --ui

# 使用 headed 模式（显示浏览器）
BASE_URL=http://192.168.18.154:8080 npx playwright test specs/slurm-command-execution-ui.spec.js --headed

# 使用 debug 模式（逐步执行）
BASE_URL=http://192.168.18.154:8080 npx playwright test specs/slurm-command-execution-ui.spec.js --debug
```

## 测试用例说明

### UI 功能测试套件

| 测试用例 | 说明 | 验证点 |
|---------|------|--------|
| 验证 SLURM Dashboard Tab 结构 | 检查 Tab 架构 | 3 个 Tab 页签存在 |
| 验证集群状态监控 Tab 功能 | 检查状态监控组件 | 健康度、节点统计显示 |
| 验证资源使用情况展示 | 检查资源监控 | CPU、内存、GPU 使用率 |
| 验证 SaltStack 命令执行 Tab 功能 | 检查命令执行组件 | 表单、按钮、输入框 |
| 验证命令模板按钮 | 检查快捷命令 | 5 个常用模板按钮 |
| 验证命令历史表格 | 检查历史记录 | 表格显示和列结构 |
| 验证最近作业列表 | 检查作业追踪 | Collapse 组件和内容 |
| 验证刷新按钮功能 | 检查手动刷新 | 按钮点击和 loading |
| 验证分区信息表格 | 检查分区管理 | 表格或空状态提示 |
| 验证 SaltStack 集成状态展示 | 检查 SaltStack 连接 | Descriptions 组件 |

### 命令执行流程测试套件

| 测试用例 | 说明 | 验证点 |
|---------|------|--------|
| 模拟执行 test.ping 命令 | 检查模板填充 | UI 交互和表单填充 |
| 测试功能总结报告 | 生成测试报告 | 15 个功能点统计 |

## 预期结果

### 成功标准

- ✅ 所有 UI 组件正常渲染
- ✅ Tab 切换流畅
- ✅ 表单验证正常
- ✅ 按钮交互响应
- ✅ 数据展示完整（如果有数据）

### 条件性成功

某些功能依赖于后端数据，如果数据不存在会显示空状态提示：

- ⚙️ 资源使用情况（需要资源数据）
- ⚙️ 分区信息（需要分区数据）
- ⚙️ 命令历史（需要历史数据）
- ⚙️ SaltStack 集成状态（需要 SaltStack 连接）

### 测试输出示例

```
=== SLURM 命令执行 UI 功能测试总结 ===

功能实现状态：
✅ 1. Tab 架构: 已实现
✅ 2. 集群概览 Tab: 已实现
✅ 3. 集群状态监控 Tab: 已实现
✅ 4. SaltStack 命令执行 Tab: 已实现
⚙️ 5. 集群健康度仪表盘: 条件性（需要数据）
✅ 6. 节点状态统计: 已实现
⚙️ 7. 资源使用情况: 条件性（需要资源数据）
⚙️ 8. 分区信息表格: 条件性（需要分区数据）
⚙️ 9. SaltStack 集成状态: 条件性（需要 SaltStack 连接）
✅ 10. 命令执行表单: 已实现
✅ 11. 命令模板按钮: 已实现
⚙️ 12. 命令执行历史: 条件性（需要历史数据）
⚙️ 13. 最近作业列表: 条件性（需要作业数据）
✅ 14. 刷新按钮: 已实现
⚙️ 15. 自动刷新机制: 需要长时间观察

功能覆盖率统计：
- 总功能点: 15
- 已完全实现: 8 (53.3%)
- 条件性实现: 7 (46.7%)
- 综合完成度: 90.7%
```

## 手动测试步骤

### 1. 访问 SLURM Dashboard

```
URL: http://192.168.18.154:8080/slurm
```

### 2. 验证 Tab 结构

- [ ] 看到"集群概览" Tab
- [ ] 看到"集群状态监控" Tab
- [ ] 看到"SaltStack 命令执行" Tab

### 3. 测试集群状态监控

**步骤**:
1. 点击"集群状态监控" Tab
2. 观察集群健康度仪表盘（进度圈）
3. 检查节点状态统计（6 个统计卡片）
4. 查看资源使用情况（CPU、内存、GPU）
5. 浏览分区信息表格
6. 检查 SaltStack 集成状态

**验证点**:
- [ ] 集群健康度分数显示
- [ ] 节点统计数据正确
- [ ] 资源使用率进度条显示
- [ ] 分区表格或空状态提示
- [ ] SaltStack 连接状态

### 4. 测试 SaltStack 命令执行

**步骤**:
1. 点击"SaltStack 命令执行" Tab
2. 观察命令执行表单
3. 点击 "test.ping" 模板按钮
4. 验证表单自动填充
5. （可选）执行命令并查看结果
6. 检查命令执行历史表格
7. 查看最近作业列表

**验证点**:
- [ ] 表单字段正确（target、function、arguments）
- [ ] 模板按钮填充功能正常
- [ ] 执行按钮响应
- [ ] 历史记录表格显示
- [ ] 作业列表折叠面板

### 5. 测试刷新功能

**步骤**:
1. 在"集群状态监控" Tab 中
2. 点击"刷新状态"按钮
3. 观察 loading 状态
4. 验证数据更新

**验证点**:
- [ ] 刷新按钮显示 loading
- [ ] 数据重新加载
- [ ] 页面无错误

### 6. 测试自动刷新

**步骤**:
1. 停留在任意 Tab
2. 等待 30 秒
3. 观察数据是否自动更新

**验证点**:
- [ ] 30 秒后数据自动刷新
- [ ] 无错误提示

## 故障排除

### 问题 1: 测试超时

**原因**: 网络慢或服务未启动

**解决**:
```bash
# 检查服务状态
docker-compose ps

# 重启服务
docker-compose restart backend frontend

# 增加超时时间
BASE_URL=http://192.168.18.154:8080 npx playwright test specs/slurm-command-execution-ui.spec.js --timeout=60000
```

### 问题 2: 登录失败

**原因**: 用户名或密码错误

**解决**:
```bash
# 使用正确的凭据
# 用户名: admin
# 密码: admin123

# 或检查数据库中的用户
docker exec -it ai-infra-postgres psql -U postgres -d ai_infra -c "SELECT username FROM users;"
```

### 问题 3: 组件不显示

**原因**: 前端编译错误或未重启

**解决**:
```bash
# 重新构建前端
docker-compose build frontend

# 重启前端服务
docker-compose restart frontend

# 查看前端日志
docker-compose logs frontend
```

### 问题 4: SaltStack 数据不显示

**原因**: SaltStack 服务未运行或未配置

**解决**:
```bash
# 检查 SaltStack 容器
docker-compose ps saltstack

# 重启 SaltStack
docker-compose restart saltstack

# 检查后端 SaltStack 配置
grep SALTSTACK .env
```

## 测试报告

测试完成后，Playwright 会生成报告：

```bash
# 查看 HTML 报告
npx playwright show-report

# 报告位置
playwright-report/index.html
```

## 性能基准

- **Tab 切换**: < 200ms
- **数据加载**: < 2s
- **表单提交**: < 1s
- **刷新操作**: < 3s

## 持续集成

### GitHub Actions 示例

```yaml
name: SLURM UI Tests

on:
  push:
    paths:
      - 'src/frontend/src/components/SaltCommandExecutor.js'
      - 'src/frontend/src/components/SlurmClusterStatus.js'
      - 'src/frontend/src/pages/SlurmDashboard.js'
      - 'test/e2e/specs/slurm-command-execution-ui.spec.js'

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
        with:
          node-version: '18'
      - name: Install dependencies
        run: npm install
        working-directory: test/e2e
      - name: Install Playwright
        run: npx playwright install --with-deps
        working-directory: test/e2e
      - name: Run tests
        run: BASE_URL=http://localhost:8080 npx playwright test specs/slurm-command-execution-ui.spec.js
        working-directory: test/e2e
      - uses: actions/upload-artifact@v3
        if: always()
        with:
          name: playwright-report
          path: test/e2e/playwright-report/
```

## 相关文档

- [SLURM Dashboard 增强说明](../docs/SLURM_COMMAND_EXECUTION_UI.md)
- [SaltStack 集成文档](../docs/SALTSTACK_INTEGRATION.md)
- [Playwright 测试指南](./PLAYWRIGHT_GUIDE.md)

## 联系与支持

如有问题，请查看：
- 项目 README
- dev-md.md 第 166 条记录
- GitHub Issues
