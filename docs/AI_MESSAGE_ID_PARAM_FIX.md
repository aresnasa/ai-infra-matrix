# AI 助手消息ID参数修复

## 日期
2025-10-20

## 问题描述

用户在使用 AI 助手功能时遇到错误：
```json
{
  "error": "无效的消息ID"
}
```

## 根本原因

**路由参数不匹配**：路由定义使用了 `:id` 参数，但控制器代码中尝试获取 `messageId` 参数。

### 问题详情

1. **路由定义** (`src/backend/cmd/main.go`)：
   ```go
   ai.GET("/messages/:id/status", aiAssistantController.GetMessageStatus)
   // 缺少此路由：ai.PATCH("/messages/:id/stop", aiAssistantController.StopMessage)
   ```

2. **控制器代码错误** (`src/backend/internal/controllers/ai_assistant_controller.go`)：
   ```go
   // 错误：获取 "messageId" 参数
   messageID, err := strconv.ParseUint(c.Param("messageId"), 10, 32)
   
   // 正确：应该获取 "id" 参数
   messageID, err := strconv.ParseUint(c.Param("id"), 10, 32)
   ```

3. **缺少路由**：
   - 前端调用 `PATCH /ai/messages/${messageId}/stop`
   - 但路由中没有定义这个端点

## 修复内容

### 1. 修复 GetMessageStatus 函数（第739行）

**修改前**：
```go
messageID, err := strconv.ParseUint(c.Param("messageId"), 10, 32)
```

**修改后**：
```go
messageID, err := strconv.ParseUint(c.Param("id"), 10, 32)
```

### 2. 修复 DeleteMessage 函数（第855行）

**修改前**：
```go
messageID, err := strconv.ParseUint(c.Param("messageId"), 10, 32)
```

**修改后**：
```go
messageID, err := strconv.ParseUint(c.Param("id"), 10, 32)
```

### 3. 修复 StopMessage 函数（第1001行）

**修改前**：
```go
messageID := c.Param("messageId")
```

**修改后**：
```go
messageID := c.Param("id")
```

### 4. 添加缺失的路由

在 `src/backend/cmd/main.go` 第 1074 行添加：

**修改前**：
```go
// 消息管理
ai.POST("/conversations/:id/messages", aiAssistantController.SendMessage)
ai.GET("/conversations/:id/messages", aiAssistantController.GetMessages)
ai.GET("/messages/:id/status", aiAssistantController.GetMessageStatus)
```

**修改后**：
```go
// 消息管理
ai.POST("/conversations/:id/messages", aiAssistantController.SendMessage)
ai.GET("/conversations/:id/messages", aiAssistantController.GetMessages)
ai.GET("/messages/:id/status", aiAssistantController.GetMessageStatus)
ai.PATCH("/messages/:id/stop", aiAssistantController.StopMessage)
```

## 影响的功能

### 1. 获取消息状态
- **端点**: `GET /ai/messages/:id/status`
- **用途**: 查询消息处理状态
- **影响**: 修复前无法正确获取消息ID，导致查询失败

### 2. 删除消息
- **端点**: `DELETE /ai/messages/:id`（需要确认）
- **用途**: 删除指定消息
- **影响**: 修复前无法正确解析消息ID

### 3. 停止消息处理
- **端点**: `PATCH /ai/messages/:id/stop`
- **用途**: 停止正在处理的消息
- **影响**: 修复前路由不存在，功能完全不可用

## 测试验证

### 1. 测试获取消息状态

```bash
# 假设消息ID为123
curl -X GET "http://localhost:8080/api/v1/ai/messages/123/status" \
  -H "Authorization: Bearer YOUR_TOKEN"

# 预期响应
{
  "data": {
    "message_id": 123,
    "status": "completed",
    "result": {...}
  }
}
```

### 2. 测试停止消息

```bash
curl -X PATCH "http://localhost:8080/api/v1/ai/messages/123/stop" \
  -H "Authorization: Bearer YOUR_TOKEN"

# 预期响应
{
  "success": true,
  "message": "消息处理已停止"
}
```

### 3. 前端测试

在 AI 助手界面：
1. 发送一个消息
2. 立即点击"停止"按钮
3. 验证消息处理是否停止
4. 查看消息状态

## 相关文件

### 修改的文件

1. **src/backend/internal/controllers/ai_assistant_controller.go**
   - 第 739 行：GetMessageStatus 函数
   - 第 855 行：DeleteMessage 函数
   - 第 1001 行：StopMessage 函数

2. **src/backend/cmd/main.go**
   - 第 1074 行：添加 StopMessage 路由

### 相关前端代码

**src/frontend/src/services/api.js**:
```javascript
// 获取消息状态
getMessageStatus: (messageId) => api.get(`/ai/messages/${messageId}/status`),

// 停止消息处理
stopMessage: (messageId) => api.patch(`/ai/messages/${messageId}/stop`),
```

## 部署步骤

### 1. 重新构建 Backend

```bash
./build.sh build backend --force
```

### 2. 重启 Backend 服务

```bash
docker-compose up -d --force-recreate backend
```

### 3. 验证服务

```bash
# 检查服务状态
docker-compose ps backend

# 查看日志
docker-compose logs -f backend
```

## 预防措施

### 建议

1. **统一路由参数命名**：
   - 建议所有ID参数统一使用 `:id` 而不是 `:messageId`, `:conversationId` 等
   - 或者在控制器中明确使用对应的参数名称

2. **添加单元测试**：
   ```go
   func TestGetMessageStatus(t *testing.T) {
       // 测试有效的消息ID
       // 测试无效的消息ID
       // 测试不存在的消息ID
   }
   ```

3. **API文档**：
   - 更新 Swagger/OpenAPI 文档
   - 明确标注所有参数名称

4. **错误信息改进**：
   ```go
   if err != nil {
       c.JSON(http.StatusBadRequest, gin.H{
           "error": "无效的消息ID",
           "detail": fmt.Sprintf("无法解析消息ID参数: %v", err),
           "param": "id",  // 明确指出问题参数
       })
       return
   }
   ```

## 参考

- [Gin路由参数文档](https://gin-gonic.com/docs/examples/param-in-path/)
- 相关Issue: #91 (Backend SLURM 客户端安装)
- 相关文档: `docs/AI_ASSISTANT_404_FIX.md`

## 总结

这是一个典型的**路由参数不匹配**问题：

1. ✅ **路由定义**: `/messages/:id/status` (使用 `:id`)
2. ❌ **控制器代码**: `c.Param("messageId")` (获取 `messageId`)
3. 💥 **结果**: 参数获取失败，返回"无效的消息ID"

**修复方法**：统一使用 `:id` 参数，并添加缺失的路由。

**建议**：在开发过程中使用一致的命名约定，并添加自动化测试来捕获此类错误。
