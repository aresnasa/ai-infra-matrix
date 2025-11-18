# DeepSeek API Key 环境变量配置和 Playwright 测试

## 📋 需求概述

**需求 ID**: 需求 92  
**日期**: 2025-10-21  
**状态**: ✅ 已完成

调整 DeepSeek 模型配置使用环境变量 `DEEPSEEK_API_KEY`，并创建 Playwright 自动化测试验证配置正确性。

## 🎯 实现目标

1. ✅ 修改 DeepSeek 初始化逻辑，仅在设置了 `DEEPSEEK_API_KEY` 时创建配置
2. ✅ 创建 Playwright 测试脚本验证 DeepSeek 模型配置
3. ✅ 确保两个 DeepSeek 模型的 `model_type` 字段正确设置为 "chat"
4. ✅ 测试通过，验证 API 响应数据完整性

## 🔧 技术实现

### 1. 修改 DeepSeek 初始化逻辑

**文件**: `src/backend/cmd/init/main.go`

**修改前**:
```go
// 检查是否配置了 DeepSeek 相关环境变量（API Key、Base URL 或 Model）
deepseekAPIKey := os.Getenv("DEEPSEEK_API_KEY")
deepseekBaseURL := os.Getenv("DEEPSEEK_BASE_URL")
deepseekChatModel := os.Getenv("DEEPSEEK_CHAT_MODEL")
deepseekReasonerModel := os.Getenv("DEEPSEEK_REASONER_MODEL")

// 只要配置了任意一个 DeepSeek 相关环境变量，就创建默认配置
if deepseekAPIKey != "" || deepseekBaseURL != "" || deepseekChatModel != "" || deepseekReasonerModel != "" {
    // 如果没有 API Key，使用占位符（用户可以后续在管理界面配置）
    if deepseekAPIKey == "" {
        deepseekAPIKey = "sk-placeholder-configure-in-admin-panel"
        log.Println("⚠️  DEEPSEEK_API_KEY 未配置，使用占位符创建默认模型，请在管理面板中配置")
    }
```

**修改后**:
```go
// 创建DeepSeek配置
// 检查是否配置了 DEEPSEEK_API_KEY 环境变量
deepseekAPIKey := os.Getenv("DEEPSEEK_API_KEY")

// 只有配置了 DEEPSEEK_API_KEY 才创建 DeepSeek 配置
if deepseekAPIKey != "" && deepseekAPIKey != "sk-test-demo-key-replace-with-real-api-key" {
```

**关键改进**:
- ✅ 简化判断逻辑，只检查 `DEEPSEEK_API_KEY`
- ✅ 排除测试占位符 Key
- ✅ 不再自动创建占位符配置
- ✅ 添加了 `*createdConfigs++` 统计 Reasoner 模型

### 2. 环境变量配置

**文件**: `.env`

```bash
# DeepSeek AI 配置
DEEPSEEK_API_KEY=sk-test-deepseek-api-key-for-testing
DEEPSEEK_BASE_URL=https://api.deepseek.com/v1
DEEPSEEK_CHAT_MODEL=deepseek-chat
DEEPSEEK_REASONER_MODEL=deepseek-reasoner
```

**配置说明**:
- `DEEPSEEK_API_KEY`: DeepSeek API 密钥（必需）
- `DEEPSEEK_BASE_URL`: API 端点（可选，默认 `https://api.deepseek.com`）
- `DEEPSEEK_CHAT_MODEL`: Chat 模型名称（可选，默认 `deepseek-chat`）
- `DEEPSEEK_REASONER_MODEL`: Reasoner 模型名称（可选，默认 `deepseek-reasoner`）

### 3. Playwright 自动化测试

**文件**: `test/e2e/specs/deepseek-model-config.spec.js`

**测试功能**:

#### Test 1: 验证 DeepSeek API 返回正确的模型配置
```javascript
test('验证 DeepSeek API 返回正确的模型配置', async ({ request }) => {
  // 1. 登录获取 token
  // 2. 调用 AI 配置 API
  // 3. 验证响应状态
  // 4. 过滤 DeepSeek 模型
  // 5. 验证至少有 2 个模型
  // 6. 验证每个模型的 model_type 不为空且为 "chat"
  // 7. 验证 Chat 和 Reasoner 模型都存在
});
```

#### Test 2: 验证 DeepSeek 模型的详细配置
```javascript
test('验证 DeepSeek 模型的详细配置', async ({ request }) => {
  // 1. 获取所有配置
  // 2. 验证字段类型
  // 3. 验证 API 端点包含 "deepseek"
  // 4. 验证模型名称匹配
  // 5. 验证启用状态
});
```

#### Test 3: 验证 DeepSeek 模型 ID 和 model_type 的映射
```javascript
test('验证 DeepSeek 模型 ID 和 model_type 的映射', async ({ request }) => {
  // 1. 创建 ID 到 model_type 的映射表
  // 2. 验证所有 model_type 都不为空
  // 3. 验证所有 model_type 都为 "chat"
});
```

**关键特性**:
- ✅ 使用 `beforeAll` hook 登录获取认证 token
- ✅ 所有 API 请求都带认证 header
- ✅ 详细的日志输出便于调试
- ✅ 完整的字段验证逻辑

## 📊 测试结果

### 运行测试
```bash
BASE_URL=http://192.168.0.200:8080 npx playwright test test/e2e/specs/deepseek-model-config.spec.js --reporter=line
```

### 测试输出
```
Running 3 tests using 1 worker

✓ DeepSeek 模型配置测试 › 验证 DeepSeek API 返回正确的模型配置
✓ DeepSeek 模型配置测试 › 验证 DeepSeek 模型的详细配置
✓ DeepSeek 模型配置测试 › 验证 DeepSeek 模型 ID 和 model_type 的映射

3 passed (1.4s)
```

### API 响应数据验证

**DeepSeek Chat 模型 (ID: 3)**:
```json
{
  "id": 3,
  "name": "DeepSeek-V3.2-Exp (Chat)",
  "provider": "deepseek",
  "model_type": "chat",  ✅ 正确
  "api_key": "***",
  "api_endpoint": "https://api.deepseek.com/v1",
  "model": "deepseek-chat",
  "max_tokens": 8192,
  "temperature": 0.7,
  "top_p": 1,
  "is_enabled": true,
  "is_default": false,
  "category": "通用对话"
}
```

**DeepSeek Reasoner 模型 (ID: 4)**:
```json
{
  "id": 4,
  "name": "DeepSeek-V3.2-Exp (Reasoner)",
  "provider": "deepseek",
  "model_type": "chat",  ✅ 正确
  "api_key": "***",
  "api_endpoint": "https://api.deepseek.com/v1",
  "model": "deepseek-reasoner",
  "max_tokens": 8192,
  "temperature": 0.7,
  "top_p": 1,
  "is_enabled": true,
  "is_default": false,
  "category": "深度推理"
}
```

## 🔄 部署步骤

### 1. 配置环境变量
```bash
# 编辑 .env 文件
vi .env

# 添加或修改 DEEPSEEK_API_KEY
DEEPSEEK_API_KEY=your-actual-api-key-here
```

### 2. 重新构建 backend-init
```bash
./build.sh build backend-init --force
```

### 3. 重新初始化数据库
```bash
docker-compose up -d --force-recreate backend-init
docker-compose logs -f backend-init
```

### 4. 验证配置
```bash
# 运行 Playwright 测试
BASE_URL=http://192.168.0.200:8080 npx playwright test test/e2e/specs/deepseek-model-config.spec.js
```

## 📝 初始化日志

```
2025/10/21 10:00:42 === Initializing Default AI Configurations ===
2025/10/21 10:00:42 ✓ Created OpenAI configuration with API key
2025/10/21 10:00:42 ✓ Created Claude configuration with API key
2025/10/21 10:00:42 ✓ Created DeepSeek Chat (V3.2-Exp) configuration
2025/10/21 10:00:42 ✓ Created DeepSeek Reasoner (V3.2-Exp) configuration
```

## 🎉 完成效果

### 修复前的问题
❌ ID 3 的 `model_type` 字段为空字符串  
❌ 即使没有 API Key 也会创建占位符配置  
❌ 没有自动化测试验证配置正确性

### 修复后的效果
✅ 所有 DeepSeek 模型的 `model_type` 都正确设置为 "chat"  
✅ 只有配置了有效的 `DEEPSEEK_API_KEY` 才会创建配置  
✅ Playwright 测试覆盖完整，确保 API 响应数据正确  
✅ 测试包含认证逻辑，更贴近实际使用场景

## 🔍 技术细节

### DeepSeek 模型类型说明

**Chat 模式** (ID: 3):
- 模型: `deepseek-chat`
- 用途: 快速对话和一般任务
- 特点: 响应速度快，适合日常交互

**Reasoner 模式** (ID: 4):
- 模型: `deepseek-reasoner`
- 用途: 复杂推理、数学问题和深度分析
- 特点: 包含详细推理过程，适合需要逻辑分析的任务

### API 端点配置

默认使用 DeepSeek 官方 API:
```
https://api.deepseek.com
```

可通过环境变量自定义:
```bash
DEEPSEEK_BASE_URL=https://your-custom-endpoint.com
```

### 字段验证清单

测试验证的字段包括:
- ✅ `id`: 数字类型
- ✅ `name`: 字符串类型，包含模型名称
- ✅ `provider`: 字符串类型，值为 "deepseek"
- ✅ `model_type`: 字符串类型，值为 "chat"，不为空
- ✅ `api_key`: 敏感信息，显示为 "***"
- ✅ `api_endpoint`: 包含 "deepseek" 关键字
- ✅ `model`: 匹配正则 `/deepseek/i`
- ✅ `is_enabled`: 布尔类型
- ✅ `is_default`: 布尔类型

## 🔐 安全建议

1. **API Key 保护**:
   - 不要在代码中硬编码 API Key
   - 使用环境变量或密钥管理服务
   - API 响应中自动脱敏显示为 "***"

2. **测试环境**:
   - 使用专门的测试 API Key
   - 定期轮换 API Key
   - 监控 API 调用量和异常

3. **生产环境**:
   ```bash
   # 使用 Docker Secrets 或 Kubernetes Secrets
   echo "your-real-api-key" | docker secret create deepseek_api_key -
   ```

## 📈 未来改进

1. **支持多个 API Key**:
   - 实现 API Key 轮询
   - 负载均衡和故障转移

2. **模型管理增强**:
   - 动态添加新模型
   - 模型版本管理
   - A/B 测试支持

3. **测试增强**:
   - 添加性能测试
   - 添加错误场景测试
   - 集成到 CI/CD 流程

## 📚 相关文档

- [AI 助手消息ID参数修复](./AI_MESSAGE_ID_PARAM_FIX.md)
- [Build.sh 智能构建指南](./BUILD_SMART_CACHE_GUIDE.md)
- [AppHub SLURM 客户端构建](./APPHUB_SLURM_BUILD_GUIDE.md)

## 👥 贡献者

- **开发**: GitHub Copilot + aresnasa
- **测试**: Playwright E2E 测试框架
- **日期**: 2025-10-21

---

**最后更新**: 2025-10-21 10:01:00  
**版本**: v0.3.6-dev  
**状态**: ✅ 已验证通过
