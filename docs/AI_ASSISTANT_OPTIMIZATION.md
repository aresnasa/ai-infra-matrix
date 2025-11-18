# AI Assistant 优化功能

## 概述

AI Assistant 已经过全面优化，集成了消息持久化、Kafka缓存和高级检索功能，大幅提升了性能和可靠性。

## 新增功能

### 1. 消息持久化服务 (MessagePersistenceService)

- **自动保存**: 所有消息自动保存到数据库，确保数据不丢失
- **批量处理**: 支持批量保存消息，提高性能
- **事务安全**: 使用数据库事务确保数据一致性
- **定期清理**: 自动清理孤立消息和归档旧数据

### 2. Kafka消息缓存 (KafkaMessageService)

- **实时流处理**: 使用Kafka进行消息流处理
- **事件驱动**: 支持消息事件发布和订阅
- **高可用**: 支持Kafka集群部署
- **异步处理**: 非阻塞的消息处理机制

### 3. 优化检索服务 (MessageRetrievalService)

- **智能缓存**: 多层缓存策略，优先从缓存获取数据
- **高级搜索**: 支持关键词搜索和时间范围查询
- **流式获取**: 支持实时消息流
- **权限控制**: 严格的用户权限验证

### 4. 服务管理器 (AIServiceManager)

- **统一管理**: 集中管理所有AI相关服务
- **健康检查**: 实时监控服务状态
- **优雅启动**: 按依赖顺序启动服务
- **自动重试**: 失败任务自动重试机制

## 配置说明

### 环境变量

```bash
# Kafka配置
export KAFKA_BROKERS="localhost:9092,localhost:9093,localhost:9094"

# Redis配置
export REDIS_HOST="localhost"
export REDIS_PORT="6379"
export REDIS_PASSWORD=""

# 数据库配置
export DB_HOST="localhost"
export DB_PORT="3306"
export DB_USER="ai_user"
export DB_PASSWORD="your_password"
export DB_NAME="ai_infra_matrix"
```

### 配置文件

复制并修改配置文件：

```bash
cp config/ai_assistant_config.yaml config/ai_assistant.yaml
```

## API端点

### 新增端点

#### 搜索消息
```
GET /api/v1/conversations/{id}/messages/search?keyword={keyword}&limit={limit}
```

#### 获取消息统计
```
GET /api/v1/messages/stats?start_date=2024-01-01&end_date=2024-01-31
```

#### 删除消息
```
DELETE /api/v1/conversations/{id}/messages/{messageId}
```

#### 流式获取消息
```
GET /api/v1/conversations/{id}/messages/stream?last_message_id={id}
```

#### 预加载消息
```
POST /api/v1/conversations/{id}/messages/preload
```

### 优化端点

#### 获取消息 (增强版)
```
GET /api/v1/conversations/{id}/messages?limit={limit}&offset={offset}&keyword={keyword}
```

## 使用示例

### 1. 基本消息发送

```go
// 发送消息（自动保存到数据库和Kafka）
messageID, err := messageQueueService.SendChatRequest(userID, &conversationID, "Hello AI", nil)
```

### 2. 高级消息检索

```go
// 搜索消息
messages, err := messageRetrievalService.SearchMessagesInConversation(conversationID, userID, "keyword", 20)

// 获取时间范围内的消息
messages, err := messageRetrievalService.GetMessagesByTimeRange(conversationID, userID, startTime, endTime)
```

### 3. 流式消息处理

```go
// 启动消息流
messageChan, err := messageRetrievalService.StreamMessages(conversationID, userID, lastMessageID)

for message := range messageChan {
    // 处理实时消息
    fmt.Printf("New message: %s\n", message.Content)
}
```

### 4. 批量消息保存

```go
// 批量保存消息
err := messagePersistenceService.BatchSaveMessages(messageList)
```

## 性能优化

### 缓存策略

1. **多层缓存**: Redis + 内存缓存
2. **智能过期**: 根据访问频率调整缓存时间
3. **预加载**: 主动预加载热点数据

### 数据库优化

1. **索引优化**: 为常用查询字段建立索引
2. **批量操作**: 减少数据库连接次数
3. **连接池**: 使用连接池提高并发性能

### Kafka优化

1. **分区策略**: 按用户ID分区，提高并发处理能力
2. **批量发送**: 批量发送消息，减少网络开销
3. **消费者组**: 使用消费者组实现负载均衡

## 监控和维护

### 健康检查

```go
// 获取服务健康状态
health := serviceManager.HealthCheck()
```

### 统计信息

```go
// 获取消息统计
stats, err := messagePersistenceService.GetMessageStats(userID, startDate, endDate)
```

### 日志监控

系统会自动记录详细的操作日志，包括：
- 消息处理时间
- 缓存命中率
- 错误统计
- 性能指标

## 部署指南

### 1. 安装依赖

```bash
# 安装Kafka
docker run -d --name kafka -p 9092:9092 confluentinc/cp-kafka:latest

# 安装Redis
docker run -d --name redis -p 6379:6379 redis:latest
```

### 2. 配置环境

```bash
export KAFKA_BROKERS="localhost:9092"
export REDIS_HOST="localhost"
export DB_HOST="localhost"
```

### 3. 启动服务

```go
manager := services.NewAIServiceManager()
err := manager.Start()
```

## 故障排除

### 常见问题

1. **Kafka连接失败**
   - 检查Kafka服务是否运行
   - 验证broker地址配置

2. **Redis连接失败**
   - 检查Redis服务状态
   - 验证连接参数

3. **数据库连接失败**
   - 检查数据库服务
   - 验证连接字符串

4. **消息处理延迟**
   - 检查消费者组状态
   - 监控队列长度

### 性能调优

1. **增加Kafka分区数**: 提高并发处理能力
2. **调整Redis连接池**: 优化缓存性能
3. **数据库索引优化**: 提升查询性能
4. **批量处理大小**: 根据负载调整批量大小

## 扩展开发

### 添加新的消息处理器

```go
type CustomMessageHandler struct{}

func (h *CustomMessageHandler) HandleMessage(message *KafkaMessage) error {
    // 自定义消息处理逻辑
    return nil
}
```

### 自定义缓存策略

```go
type CustomCacheStrategy struct{}

func (s *CustomCacheStrategy) Get(key string) interface{} {
    // 自定义缓存获取逻辑
    return nil
}
```

## 版本兼容性

- **Go版本**: 1.19+
- **Kafka版本**: 2.8+
- **Redis版本**: 6.0+
- **数据库**: MySQL 8.0+ / PostgreSQL 12+

## 贡献指南

1. Fork项目
2. 创建特性分支
3. 提交变更
4. 发起Pull Request

## 许可证

本项目采用MIT许可证。
