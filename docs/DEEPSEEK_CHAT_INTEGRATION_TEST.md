# DeepSeek 聊天集成测试文档

## 📋 测试概述

**测试文件**: `test/e2e/specs/deepseek-chat-integration.spec.js`  
**创建日期**: 2025-10-21  
**状态**: ⚠️ 部分完成（需要有效的 DEEPSEEK_API_KEY）

## 🎯 测试目标

1. ✅ 验证 DeepSeek 模型配置正确加载
2. ✅ 验证 API Key 来自环境变量（不在代码中硬编码）
3. ✅ 测试创建对话会话
4. ✅ 测试发送消息到 DeepSeek
5. ⏳ 等待并验证 DeepSeek 的响应（需要有效 API Key）

## 📝 测试用例

### Test 1: 使用 DeepSeek Chat 模型进行简单对话

**测试流程**:
1. 登录获取认证 token
2. 获取 DeepSeek Chat 配置 (ID: 3)
3. 创建新对话会话
4. 发送测试消息："你好，请用一句话介绍一下你自己。"
5. 等待 AI 响应（最多 60 秒）
6. 验证响应内容
7. 清理测试会话

**当前状态**: ⚠️ 消息发送成功，但响应超时

**原因分析**:
- 测试使用的 API Key 是占位符：`sk-test-deepseek-api-key-for-testing`
- DeepSeek API 拒绝无效的 API Key
- 需要设置真实的 DEEPSEEK_API_KEY

### Test 2: 使用 DeepSeek Reasoner 模型进行推理任务

**测试流程**:
1. 使用 DeepSeek Reasoner 配置 (ID: 4)
2. 发送数学问题："计算：15 + 27 = ?"
3. 验证响应包含正确答案 42

**当前状态**: ⏳ 未运行（等待 API Key）

### Test 3: 验证 DeepSeek API Key 来自环境变量 ✅

**验证内容**:
- API Key 在响应中被脱敏显示为 `***`
- API Endpoint 包含 `deepseek` 关键字
- 配置来自系统环境变量，不在代码中硬编码

**测试结果**: ✅ 通过
```
检查配置: DeepSeek-V3.2-Exp (Chat)
  ✓ API Key 已脱敏: ***
  ✓ API Endpoint: https://api.deepseek.com/v1
检查配置: DeepSeek-V3.2-Exp (Reasoner)
  ✓ API Key 已脱敏: ***
  ✓ API Endpoint: https://api.deepseek.com/v1
```

### Test 4: 测试网络错误处理 ✅

**验证内容**:
- 创建无效配置（无效的 endpoint）
- 发送消息并处理错误
- 清理测试数据

**测试结果**: ✅ 通过

## 🔧 使用方法

### 前置条件

1. **设置有效的 DEEPSEEK_API_KEY**:
   ```bash
   # 编辑 .env 文件
   vi .env
   
   # 设置真实的 API Key（从 DeepSeek 官网获取）
   DEEPSEEK_API_KEY=sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
   ```

2. **重新初始化数据库**:
   ```bash
   # 重新构建 backend-init
   ./build.sh build backend-init --force
   
   # 重新初始化（使用新的 API Key）
   docker-compose up -d --force-recreate backend-init
   
   # 查看初始化日志
   docker-compose logs backend-init | grep DeepSeek
   ```

3. **确保服务正常运行**:
   ```bash
   # 检查服务状态
   docker-compose ps
   
   # 确保 backend、kafka 都在运行
   docker ps --filter "name=backend\|kafka"
   ```

### 运行测试

```bash
# 设置测试 URL
export BASE_URL=http://192.168.0.200:8080

# 运行所有测试
npx playwright test test/e2e/specs/deepseek-chat-integration.spec.js --reporter=line --timeout=120000

# 只运行配置验证测试（不需要有效 API Key）
npx playwright test test/e2e/specs/deepseek-chat-integration.spec.js --grep "验证 DeepSeek API Key" --reporter=line

# 运行聊天测试（需要有效 API Key）
npx playwright test test/e2e/specs/deepseek-chat-integration.spec.js --grep "使用 DeepSeek Chat" --reporter=line --timeout=120000
```

## 📊 测试结果

### 当前测试运行结果

```bash
Running 4 tests using 1 worker

✅ DeepSeek 聊天集成测试 › 验证 DeepSeek API Key 来自环境变量
✅ DeepSeek 聊天集成测试 › 测试网络错误处理
⚠️  DeepSeek 聊天集成测试 › 使用 DeepSeek Chat 模型进行简单对话 (超时)
⚠️  DeepSeek 聊天集成测试 › 使用 DeepSeek Reasoner 模型进行推理任务 (未运行)

2 passed, 2 failed
```

### 消息发送成功示例

```json
{
  "message": "消息已提交处理",
  "message_id": "chat_1_1761012926063544009",
  "status": "pending"
}
```

### API 响应验证成功

```
检查配置: DeepSeek-V3.2-Exp (Chat)
  ✓ API Key 已脱敏: ***
  ✓ API Endpoint: https://api.deepseek.com/v1
检查配置: DeepSeek-V3.2-Exp (Reasoner)
  ✓ API Key 已脱敏: ***
  ✓ API Endpoint: https://api.deepseek.com/v1
```

## 🔍 问题诊断

### 问题 1: 消息超时未收到响应

**症状**:
```
等待 DeepSeek 响应...
  等待中... (5/60 秒)
  等待中... (10/60 秒)
  ...
  等待中... (60/60 秒)
Error: 超时：未收到 DeepSeek 的响应
```

**可能原因**:
1. ❌ **API Key 无效**（最可能）
   - 当前使用测试占位符：`sk-test-deepseek-api-key-for-testing`
   - DeepSeek API 拒绝无效请求
   
2. ⚠️ **消息队列处理器未启动**
   - Kafka 服务正常
   - 需要检查 backend 是否有消息处理器

3. ⚠️ **网络问题**
   - DeepSeek API 可能无法访问
   - 需要检查防火墙规则

**解决方案**:
```bash
# 1. 设置有效的 API Key
echo "DEEPSEEK_API_KEY=sk-your-real-api-key" >> .env

# 2. 重新初始化
./build.sh build backend-init --force
docker-compose up -d --force-recreate backend-init

# 3. 重启 backend 服务
docker-compose restart backend

# 4. 查看处理日志
docker logs -f ai-infra-backend | grep -i "deepseek\|chat\|message"
```

### 问题 2: API 请求格式错误

**已修复**: ✅

**之前的错误**:
```json
{
  "error": "Key: 'Message' Error:Field validation for 'Message' failed on the 'required' tag"
}
```

**修复方案**:
- 将请求字段从 `content` 改为 `message`
- 移除不需要的 `config_id` 字段

## 🔐 安全说明

### API Key 管理

**✅ 正确做法** (当前实现):
```javascript
// 测试代码中不包含 API Key
// API Key 从系统环境变量加载
const deepseekAPIKey = process.env.DEEPSEEK_API_KEY;  // ← 从系统读取
```

**❌ 错误做法** (避免):
```javascript
// 不要在测试代码中硬编码 API Key
const deepseekAPIKey = 'sk-xxxxxxxx';  // ← 危险！
```

### 环境变量配置

```bash
# .env 文件（本地开发）
DEEPSEEK_API_KEY=sk-your-real-api-key

# 生产环境（使用 Docker Secrets 或 Kubernetes Secrets）
docker secret create deepseek_api_key - <<< "sk-your-real-api-key"
```

### API Key 验证

测试会验证：
- ✅ API Key 在响应中被脱敏为 `***`
- ✅ 配置来自环境变量，不在代码中
- ✅ 前端无法直接获取明文 API Key

## 📈 下一步计划

### 立即任务

1. **获取真实的 DeepSeek API Key**:
   - 访问 [DeepSeek 官网](https://platform.deepseek.com)
   - 注册账号并申请 API Key
   - 在 `.env` 中配置

2. **重新运行测试**:
   ```bash
   # 使用真实 API Key 运行完整测试
   BASE_URL=http://192.168.0.200:8080 \
     npx playwright test test/e2e/specs/deepseek-chat-integration.spec.js \
     --reporter=line --timeout=120000
   ```

3. **验证完整流程**:
   - ✅ 消息发送
   - ✅ AI 响应接收
   - ✅ 响应内容验证
   - ✅ 会话清理

### 未来改进

1. **增加更多测试场景**:
   - 长文本对话
   - 流式响应测试
   - 并发请求测试
   - 错误恢复测试

2. **性能测试**:
   - 响应时间统计
   - 吞吐量测试
   - 负载测试

3. **集成到 CI/CD**:
   - 使用测试环境专用 API Key
   - 自动化测试流程
   - 测试结果报告

## 📚 相关文档

- [DeepSeek API Key 环境变量配置](./DEEPSEEK_API_KEY_ENV_CONFIG.md)
- [DeepSeek 模型配置测试](../test/e2e/specs/deepseek-model-config.spec.js)
- [AI 助手 API 文档](./AI_ASSISTANT_API.md)

## 🎯 成功标准

测试通过的条件：
- ✅ 所有 4 个测试用例通过
- ✅ DeepSeek 返回有效的响应数据
- ✅ 响应时间 < 30 秒
- ✅ 响应内容符合预期
- ✅ 无 API Key 泄漏

---

**最后更新**: 2025-10-21 10:20:00  
**版本**: v0.3.6-dev  
**状态**: ⚠️ 需要有效的 DEEPSEEK_API_KEY
