# Playwright 测试指南

本目录包含 Playwright 自动化测试脚本，包括 MCP (Model Context Protocol) 测试和标准 E2E 测试。

## 📁 文件说明

### 测试脚本

1. **test-saltstack-exec-e2e.js** ⭐ **NEW** - SaltStack 命令执行 E2E 测试
   - 测试 SaltStack 页面的命令执行功能
   - 验证执行完成后状态正确更新（修复转圈问题）
   - 包含完整的测试流程和错误处理
   - 可直接运行：`node scripts/js/test-saltstack-exec-e2e.js`

2. **test-suite-mcp.js** - 完整的测试套件定义
   - 包含所有测试用例的详细定义
   - 定义了测试步骤和期望结果
   - 可以单独运行查看测试用例信息

3. **test-dashboard-mcp.js** - 仪表板测试脚本
   - 定义了仪表板相关的测试框架
   - 提供测试结果追踪功能

4. **test-slurm-saltstack.js** - SLURM 和 SaltStack 测试脚本
   - 使用传统 Playwright API 的测试脚本
   - 包含详细的测试逻辑和错误处理

### 测试报告

- **test-mcp-results.md** - 完整的测试执行报告
  - 测试结果汇总
  - 详细的测试步骤和结果
  - 问题分析和建议
  - 测试截图说明

## 使用 Playwright MCP 进行测试

### 什么是 Playwright MCP？

Playwright MCP 是基于 Model Context Protocol 的浏览器自动化工具，它允许 AI 助手直接控制浏览器进行测试和交互。

### 测试执行流程

1. **启动测试环境**
   ```bash
   # 确保 AI-Infra-Matrix 系统正在运行
   BASE_URL=http://192.168.0.200:8080
   ```

2. **使用 MCP 工具执行测试**
   - 通过 AI 助手使用 Playwright MCP 工具
   - 执行 `test-suite-mcp.js` 中定义的测试用例
   - 自动生成截图和测试报告

3. **查看测试结果**
   ```bash
   # 查看测试报告
   cat scripts/js/test-mcp-results.md
   
   # 查看截图
   ls -la .playwright-mcp/test-results/
   ```

### 查看测试用例定义

```bash
# 运行测试套件文件查看所有测试用例
node scripts/js/test-suite-mcp.js
```

这将显示：
- 测试用例名称和描述
- 测试步骤
- 期望结果
- 已知问题

## 测试覆盖范围

### 1. 用户认证 ✅
- 登录验证
- Token 认证
- 权限检查

### 2. SLURM 管理 ⚠️
- 页面加载
- 节点统计
- 作业队列
- **已知问题**: 后端服务可能返回 502 错误

### 3. SaltStack 管理 ✅
- 页面加载
- 服务状态检查
- Minions 管理
- 命令执行功能

### 4. 对象存储管理 ✅
- 页面加载
- 存储服务状态
- 连接验证

## 测试结果概览

| 测试项 | 状态 | 通过率 |
|--------|------|--------|
| 用户认证 | ✅ | 100% |
| SLURM 管理 | ⚠️ | 60% |
| SaltStack 管理 | ✅ | 100% |
| SaltStack 命令执行 | ✅ | 100% |
| 对象存储管理 | ✅ | 100% |
| **总计** | **✅** | **92%** |

## 测试截图

所有测试截图保存在 `.playwright-mcp/test-results/` 目录：

1. `slurm-dashboard-error.png` - SLURM 页面错误状态
2. `saltstack-dashboard.png` - SaltStack 仪表板
3. `saltstack-execute-success.png` - 命令执行成功结果
4. `object-storage-dashboard.png` - 对象存储仪表板

## 测试示例

### SaltStack 命令执行测试

**测试脚本**:
```bash
echo "Test from Playwright MCP"
hostname
date
```

**执行结果**:
- ✅ 4 个节点全部成功执行
- ✅ 执行时间: 162ms
- ✅ 所有节点返回正确的输出

**节点列表**:
- test-ssh03
- test-ssh02
- test-ssh01
- salt-master-local

## 环境变量

```bash
# 系统基础 URL
export BASE_URL=http://192.168.0.200:8080

# 管理员凭据
export ADMIN_USER=admin
export ADMIN_PASS=admin123

# 是否显示浏览器窗口
export HEADLESS=false
```

## 常见问题

### Q: 如何重新运行测试？

A: 使用 Playwright MCP 工具，按照 `test-suite-mcp.js` 中定义的测试步骤执行。

### Q: 测试失败怎么办？

A: 
1. 检查系统是否正常运行
2. 查看错误截图
3. 检查 `test-mcp-results.md` 中的问题分析
4. 验证环境变量设置

### Q: 如何添加新的测试用例？

A: 
1. 在 `test-suite-mcp.js` 的 `TestCases` 对象中添加新的测试定义
2. 定义测试步骤和期望结果
3. 使用 MCP 工具执行测试
4. 更新测试报告

## 最佳实践

1. **测试前准备**
   - 确保系统完全启动
   - 检查所有服务状态
   - 清理之前的测试数据

2. **测试执行**
   - 按顺序执行测试用例
   - 保存每个步骤的截图
   - 记录所有错误和警告

3. **结果分析**
   - 比较实际结果和期望结果
   - 分析失败原因
   - 更新测试报告

4. **持续改进**
   - 根据测试结果优化系统
   - 更新测试用例
   - 完善错误处理

## 参考资料

- [Playwright 官方文档](https://playwright.dev)
- [Model Context Protocol](https://modelcontextprotocol.io)
- [AI-Infra-Matrix 系统文档](../../docs/)

## 联系方式

如有问题或建议，请联系开发团队。

---

**最后更新**: 2025-10-11  
**测试工具版本**: Playwright MCP  
**系统版本**: AI-Infra-Matrix v0.3.7
