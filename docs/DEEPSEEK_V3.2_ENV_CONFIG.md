# DeepSeek V3.2 环境变量配置说明

## 需求背景

**需求编号**: 87  
**日期**: 2025年10月20日  
**目标**: 确保 Go 后端从环境变量读取 DeepSeek 配置，支持 DeepSeek-V3.2-Exp 模型

## 配置概述

### DeepSeek-V3.2-Exp 模型说明

DeepSeek 已将 `deepseek-chat` 和 `deepseek-reasoner` 升级为 **DeepSeek-V3.2-Exp**：

- **deepseek-chat**: DeepSeek-V3.2-Exp 的**非思考模式**
  - 适用场景：快速对话、一般任务、日常问答
  - 响应速度：快
  - 推理深度：标准

- **deepseek-reasoner**: DeepSeek-V3.2-Exp 的**思考模式**
  - 适用场景：复杂推理、数学问题、深度分析
  - 响应速度：较慢（包含推理过程）
  - 推理深度：深入

### API 端点说明

DeepSeek 使用 **OpenAI 兼容 API**：

```
https://api.deepseek.com/v1
```

**重要说明**：
- 此处的 `v1` 与模型版本**无关**，是 API 端点版本
- 保持 OpenAI API 兼容性，可以使用 OpenAI SDK

## 环境变量配置

### .env 文件配置

```bash
# DeepSeek 配置
# API Key从官网获取: https://platform.deepseek.com/
DEEPSEEK_API_KEY=sk-your-api-key-here

# Base URL: 使用 v1 端点以保持 OpenAI 兼容性（v1 与模型版本无关）
DEEPSEEK_BASE_URL=https://api.deepseek.com/v1

# DeepSeek-V3.2-Exp 模型配置
# 说明: deepseek-chat 和 deepseek-reasoner 都已升级为 DeepSeek-V3.2-Exp
# - deepseek-chat: DeepSeek-V3.2-Exp 非思考模式（快速对话）
# - deepseek-reasoner: DeepSeek-V3.2-Exp 思考模式（深度推理）
DEEPSEEK_DEFAULT_MODEL=deepseek-chat
DEEPSEEK_CHAT_MODEL=deepseek-chat
DEEPSEEK_REASONER_MODEL=deepseek-reasoner
```

### 环境变量说明

| 变量名 | 必填 | 默认值 | 说明 |
|--------|------|--------|------|
| `DEEPSEEK_API_KEY` | ✅ | - | DeepSeek API 密钥 |
| `DEEPSEEK_BASE_URL` | ❌ | `https://api.deepseek.com/v1` | API 基础 URL |
| `DEEPSEEK_DEFAULT_MODEL` | ❌ | `deepseek-chat` | 默认使用的模型 |
| `DEEPSEEK_CHAT_MODEL` | ❌ | `deepseek-chat` | 快速对话模型 |
| `DEEPSEEK_REASONER_MODEL` | ❌ | `deepseek-reasoner` | 深度推理模型 |

## Go 代码实现

### 1. 模型初始化（ai_service.go）

```go
// createOtherProviderConfigs 创建其他AI提供商的配置
func (s *aiServiceImpl) createOtherProviderConfigs(createdConfigs *int) {
    // 创建DeepSeek配置
    if deepseekAPIKey := os.Getenv("DEEPSEEK_API_KEY"); deepseekAPIKey != "" {
        baseURL := getEnvOrDefault("DEEPSEEK_BASE_URL", "https://api.deepseek.com")
        
        // 创建 DeepSeek Chat 配置（非思考模式）
        chatModel := getEnvOrDefault("DEEPSEEK_CHAT_MODEL", "deepseek-chat")
        deepseekChatConfig := &models.AIAssistantConfig{
            Name:         "DeepSeek-V3.2-Exp (Chat)",
            Provider:     models.ProviderDeepSeek,
            ModelType:    models.ModelTypeChat,
            APIKey:       deepseekAPIKey,
            APIEndpoint:  baseURL,
            Model:        chatModel,
            MaxTokens:    8192,
            Temperature:  0.7,
            TopP:         1.0,
            SystemPrompt: "你是DeepSeek助手，基于DeepSeek-V3.2-Exp模型。请提供准确、有用的回答。",
            IsEnabled:    true,
            IsDefault:    (*createdConfigs == 0),
            Description:  "DeepSeek-V3.2-Exp 非思考模式，适合快速对话和一般任务",
            Category:     "通用对话",
        }
        
        // ... 保存配置
    }
}
```

### 2. 提供商工厂（factory.go）

```go
// createDeepSeekProvider 创建DeepSeek提供商（使用OpenAI兼容接口）
func (f *DefaultProviderFactory) createDeepSeekProvider(config *models.AIAssistantConfig) (AIProvider, error) {
    // DeepSeek使用OpenAI兼容的API
    deepSeekConfig := *config
    if deepSeekConfig.APIEndpoint == "" {
        deepSeekConfig.APIEndpoint = "https://api.deepseek.com/v1/chat/completions"
    }
    deepSeekConfig.APIEndpoint = normalizeDeepSeekEndpoint(deepSeekConfig.APIEndpoint)
    
    // 从环境变量读取默认模型，支持不同模式
    if deepSeekConfig.Model == "" {
        // 优先使用 DEEPSEEK_DEFAULT_MODEL，如果未设置则使用 deepseek-chat
        defaultModel := os.Getenv("DEEPSEEK_DEFAULT_MODEL")
        if defaultModel == "" {
            defaultModel = os.Getenv("DEEPSEEK_CHAT_MODEL")
        }
        if defaultModel == "" {
            defaultModel = "deepseek-chat" // 最终回退值
        }
        deepSeekConfig.Model = defaultModel
    }
    
    // 使用 OpenAI Provider（兼容 API）
    provider := NewOpenAIProvider(&deepSeekConfig)
    return provider, nil
}
```

### 3. 端点规范化

```go
// normalizeDeepSeekEndpoint 确保 DeepSeek 端点包含正确的路径
func normalizeDeepSeekEndpoint(endpoint string) string {
    if endpoint == "" {
        return "https://api.deepseek.com/v1/chat/completions"
    }

    // 移除末尾的斜杠
    endpoint = strings.TrimSuffix(endpoint, "/")

    parsed, err := url.Parse(endpoint)
    if err != nil || parsed.Scheme == "" || parsed.Host == "" {
        return "https://api.deepseek.com/v1/chat/completions"
    }

    path := strings.TrimSuffix(parsed.Path, "/")
    
    // 智能补全路径
    switch {
    case path == "" || path == "/":
        parsed.Path = "/v1/chat/completions"
    case path == "/v1":
        parsed.Path = "/v1/chat/completions"
    case strings.HasSuffix(path, "/chat/completions"):
        // 已经正确
    default:
        parsed.Path = "/v1/chat/completions"
    }

    return parsed.String()
}
```

## 使用方法

### 1. 配置环境变量

编辑 `.env` 文件：

```bash
DEEPSEEK_API_KEY=sk-xxxxxxxxxxxxxxxx
DEEPSEEK_BASE_URL=https://api.deepseek.com/v1
DEEPSEEK_DEFAULT_MODEL=deepseek-chat
```

### 2. 重启后端服务

```bash
# 重新构建并启动
./build.sh build backend --force

# 或者只重启服务
docker-compose restart backend
```

### 3. 验证配置

检查日志：

```bash
docker-compose logs backend | grep -i deepseek
```

期望输出：

```
backend | INFO[0002] 已创建DeepSeek Chat (V3.2-Exp) 配置
backend | INFO[0002] 已创建DeepSeek Reasoner (V3.2-Exp) 配置
```

### 4. 通过 API 使用

#### Chat 模式（快速对话）

```bash
curl -X POST http://localhost:8080/api/ai/chat \
  -H "Content-Type: application/json" \
  -d '{
    "config_id": 1,
    "message": "你好，请介绍一下 DeepSeek-V3.2"
  }'
```

#### Reasoner 模式（深度推理）

```bash
curl -X POST http://localhost:8080/api/ai/chat \
  -H "Content-Type: application/json" \
  -d '{
    "config_id": 2,
    "message": "请详细解释量子计算的基本原理"
  }'
```

## 配置优先级

系统按以下优先级读取配置：

1. **数据库中的配置**（最高优先级）
   - 用户在前端界面创建的配置
   
2. **环境变量**
   - `DEEPSEEK_DEFAULT_MODEL`
   - `DEEPSEEK_CHAT_MODEL`
   - `DEEPSEEK_REASONER_MODEL`

3. **硬编码的默认值**（最低优先级）
   - Chat: `deepseek-chat`
   - Reasoner: `deepseek-reasoner`

## 配置示例

### 开发环境

```bash
# .env
DEEPSEEK_API_KEY=sk-dev-key-here
DEEPSEEK_BASE_URL=https://api.deepseek.com/v1
DEEPSEEK_DEFAULT_MODEL=deepseek-chat
```

### 生产环境

```bash
# Kubernetes Secret 或 .env
DEEPSEEK_API_KEY=sk-prod-key-here
DEEPSEEK_BASE_URL=https://api.deepseek.com/v1
DEEPSEEK_DEFAULT_MODEL=deepseek-reasoner  # 生产环境使用推理模式
```

### 测试环境

```bash
# .env.test
DEEPSEEK_API_KEY=sk-test-key-here
DEEPSEEK_BASE_URL=https://api.deepseek.com/v1
DEEPSEEK_CHAT_MODEL=deepseek-chat
DEEPSEEK_REASONER_MODEL=deepseek-reasoner
```

## 故障排查

### 问题 1: 未创建 DeepSeek 配置

**症状**:
```
WARN 未提供DEEPSEEK_API_KEY环境变量，跳过DeepSeek配置创建
```

**解决方案**:
1. 检查 `.env` 文件中是否设置了 `DEEPSEEK_API_KEY`
2. 确认环境变量已正确加载
3. 重启服务

### 问题 2: API 端点错误

**症状**:
```
ERROR invalid DeepSeek config: invalid endpoint
```

**解决方案**:
1. 检查 `DEEPSEEK_BASE_URL` 是否正确
2. 确保使用 `https://api.deepseek.com/v1`
3. 不要在末尾添加 `/chat/completions`（系统会自动补全）

### 问题 3: 模型不存在

**症状**:
```
ERROR model not found: deepseek-chat-v2
```

**解决方案**:
1. 使用标准模型名称：
   - `deepseek-chat` (非思考模式)
   - `deepseek-reasoner` (思考模式)
2. 不要使用过时的模型名称
3. 检查 DeepSeek 官方文档获取最新模型列表

### 问题 4: 配置未生效

**症状**:
仍然使用旧的配置或模型

**解决方案**:
1. 删除数据库中的旧配置：
   ```sql
   DELETE FROM ai_assistant_configs WHERE provider = 'deepseek';
   ```
2. 重启 backend 服务触发初始化
3. 检查日志确认配置已创建

## 相关文档

- [DeepSeek 官方文档](https://platform.deepseek.com/docs)
- [OpenAI 兼容 API](https://platform.deepseek.com/api-docs/quick_start)
- [AI 服务配置指南](./AI_ASSISTANT_CONFIGURATION.md)
- [环境变量完整列表](../.env.example)

## 模型对比

| 特性 | deepseek-chat | deepseek-reasoner |
|------|---------------|-------------------|
| **响应速度** | 快 | 较慢 |
| **推理深度** | 标准 | 深入 |
| **适用场景** | 日常对话、快速问答 | 复杂推理、数学证明、代码分析 |
| **Token 消耗** | 标准 | 较高（包含推理过程） |
| **成本** | 较低 | 较高 |
| **最大 Token** | 8192 | 8192 |
| **Context Window** | 32K | 32K |

## 性能建议

### Chat 模式优化

```go
config := &models.AIAssistantConfig{
    Model:       "deepseek-chat",
    Temperature: 0.7,      // 平衡创造性和准确性
    MaxTokens:   2048,     // 快速响应
    TopP:        0.9,      // 适当的多样性
}
```

### Reasoner 模式优化

```go
config := &models.AIAssistantConfig{
    Model:       "deepseek-reasoner",
    Temperature: 0.5,      // 更注重准确性
    MaxTokens:   8192,     // 允许完整推理
    TopP:        0.95,     // 更高的确定性
}
```

## 更新历史

| 日期 | 版本 | 变更内容 |
|------|------|---------|
| 2025-10-20 | 1.0 | 初始版本，支持 DeepSeek-V3.2-Exp |
| 2025-10-20 | 1.1 | 添加环境变量配置说明 |
| 2025-10-20 | 1.2 | 完善故障排查和性能优化 |

## 总结

✅ **已实现功能**：
- 从环境变量读取 DeepSeek 配置
- 支持 DeepSeek-V3.2-Exp 的 Chat 和 Reasoner 模式
- OpenAI 兼容 API 实现
- 智能端点规范化
- 配置优先级系统

⚠️ **注意事项**：
- API Key 需要从 DeepSeek 官网获取
- Base URL 的 `v1` 与模型版本无关
- Chat 和 Reasoner 都已升级为 V3.2-Exp
- 推理模式响应较慢但结果更详细

🎯 **最佳实践**：
- 开发环境使用 Chat 模式（快速迭代）
- 生产环境根据需求选择合适模式
- 定期更新 API Key 和配置
- 监控 Token 使用和成本
