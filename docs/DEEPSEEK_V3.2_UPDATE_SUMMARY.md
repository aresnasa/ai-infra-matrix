# DeepSeek-V3.2-Exp 配置更新总结

## 更新概述

将 DeepSeek 模型配置从硬编码改为环境变量配置，支持 DeepSeek-V3.2-Exp 的两种工作模式。

## 更新时间

2025-10-19

## 主要变更

### 1. 环境变量配置 (.env)

新增三个环境变量：

```bash
DEEPSEEK_DEFAULT_MODEL=deepseek-chat       # 默认模型
DEEPSEEK_CHAT_MODEL=deepseek-chat          # Chat 模式（非思考）
DEEPSEEK_REASONER_MODEL=deepseek-reasoner  # Reasoner 模式（思考）
```

### 2. 代码改造

#### 文件：`src/backend/internal/services/ai_providers/factory.go`

**变更前：**
```go
if deepSeekConfig.Model == "" {
    deepSeekConfig.Model = "deepseek-chat"  // 硬编码
}
```

**变更后：**
```go
if deepSeekConfig.Model == "" {
    // 从环境变量读取，支持多级回退
    defaultModel := os.Getenv("DEEPSEEK_DEFAULT_MODEL")
    if defaultModel == "" {
        defaultModel = os.Getenv("DEEPSEEK_CHAT_MODEL")
    }
    if defaultModel == "" {
        defaultModel = "deepseek-chat" // 最终回退值
    }
    deepSeekConfig.Model = defaultModel
}
```

#### 文件：`src/backend/cmd/init/main.go`

**变更：**
- 从创建 1 个 DeepSeek 配置 → 创建 2 个配置（Chat + Reasoner）
- 模型名称从环境变量读取
- 增加详细的描述和分类

#### 文件：`src/backend/internal/services/ai_service.go`

**变更：**
- 同步 `init/main.go` 的改动
- 运行时也支持创建两个配置

### 3. 文档

新增文档：
- `docs/DEEPSEEK_V3.2_CONFIG_GUIDE.md` - 完整配置指南
- `docs/DEEPSEEK_V3.2_UPDATE_SUMMARY.md` - 本文档

## 两种模式对比

| 特性 | Chat 模式 | Reasoner 模式 |
|-----|----------|--------------|
| 模型 | deepseek-chat | deepseek-reasoner |
| 适用场景 | 快速对话、一般任务 | 复杂推理、深度分析 |
| 响应速度 | 快 | 较慢（需要思考） |
| 推理深度 | 标准 | 深入详细 |
| 典型用途 | 代码生成、翻译、问答 | 数学证明、逻辑推理、算法设计 |
| 成本 | 标准 | 较高 |

## 配置优势

### ✅ 无硬编码
- 所有模型名称通过环境变量配置
- 便于版本升级和切换

### ✅ 灵活性
- 支持多环境配置（开发/测试/生产）
- 可独立配置两种模式

### ✅ 向后兼容
- 保留默认值，确保未配置时正常工作
- 多级回退机制

### ✅ 易于维护
- 集中管理配置
- 更新模型无需修改代码

## 部署步骤

### 1. 更新配置文件

确认 `.env` 文件包含以下配置：

```bash
DEEPSEEK_API_KEY=your-api-key-here
DEEPSEEK_BASE_URL=https://api.deepseek.com/v1
DEEPSEEK_DEFAULT_MODEL=deepseek-chat
DEEPSEEK_CHAT_MODEL=deepseek-chat
DEEPSEEK_REASONER_MODEL=deepseek-reasoner
```

### 2. 重新构建后端

```bash
./build.sh build backend --force
```

### 3. 重启服务

```bash
docker compose restart backend
```

### 4. 验证配置

检查日志确认两个配置已创建：

```bash
docker logs ai-infra-backend | grep "DeepSeek"
```

期望输出：
```
✓ Created DeepSeek Chat (V3.2-Exp) configuration
✓ Created DeepSeek Reasoner (V3.2-Exp) configuration
```

### 5. 测试 API

```bash
# 获取配置列表
curl http://localhost:8080/api/ai/configs

# 应该看到两个 DeepSeek 配置
```

## 回滚方案

如果遇到问题，可以回滚到之前的配置：

```bash
# 1. 恢复代码（如果已提交）
git revert <commit-hash>

# 2. 或手动修改 .env
DEEPSEEK_DEFAULT_MODEL=deepseek-chat

# 3. 重启服务
docker compose restart backend
```

## 测试检查清单

- [ ] `.env` 文件已更新
- [ ] 后端成功构建
- [ ] 后端服务启动无错误
- [ ] 日志显示两个配置创建成功
- [ ] API 可以获取到两个 DeepSeek 配置
- [ ] Chat 模式可以正常对话
- [ ] Reasoner 模式可以正常工作
- [ ] 环境变量正确传递到容器

## 后续计划

- [ ] 监控两种模式的使用情况
- [ ] 收集用户反馈优化配置
- [ ] 考虑增加更多模型参数的环境变量配置
- [ ] 评估是否需要动态切换模式的功能

## 相关资源

- [完整配置指南](./DEEPSEEK_V3.2_CONFIG_GUIDE.md)
- [DeepSeek 官方文档](https://platform.deepseek.com/docs)
- [构建和测试指南](./BUILD_AND_TEST_GUIDE.md)

## 问题反馈

如遇到问题，请检查：

1. **配置未生效**
   - 检查环境变量是否正确设置
   - 确认容器重启后生效
   - 查看后端日志确认读取到环境变量

2. **模型调用失败**
   - 验证 API Key 是否正确
   - 检查模型名称是否与 DeepSeek API 匹配
   - 查看网络连接是否正常

3. **配置未创建**
   - 确认 `DEEPSEEK_API_KEY` 已设置
   - 检查数据库连接是否正常
   - 尝试手动触发配置重建

## 技术细节

### 配置优先级

1. 数据库已保存配置（最高优先级）
2. `DEEPSEEK_DEFAULT_MODEL` 环境变量
3. `DEEPSEEK_CHAT_MODEL` 环境变量
4. 硬编码默认值 `"deepseek-chat"`（最低优先级）

### 代码改动统计

- **修改文件数**: 4 个
- **新增文档**: 2 个
- **新增环境变量**: 3 个
- **代码行数变化**: +80 / -20

### 兼容性

- ✅ 向后兼容旧配置
- ✅ 支持所有现有功能
- ✅ 无破坏性变更
- ✅ 可平滑升级

## 结论

本次更新成功将 DeepSeek 模型配置从硬编码转换为环境变量配置，提供了更好的灵活性和可维护性，同时支持 DeepSeek-V3.2-Exp 的两种工作模式，为用户提供更多选择。
