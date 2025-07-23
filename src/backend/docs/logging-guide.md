# 日志级别管理功能文档

## 概述

系统现在支持动态日志级别管理，允许管理员在运行时调整日志级别，以便于开发调试和生产环境监控。

## 支持的日志级别

| 级别 | 描述 | 使用场景 |
|------|------|----------|
| `trace` | 最详细的日志级别 | 深度调试，包含所有执行路径 |
| `debug` | 调试级别 | 开发环境，问题排查 |
| `info` | 信息级别 | 生产环境默认级别 |
| `warn` | 警告级别 | 只记录警告和错误 |
| `error` | 错误级别 | 只记录错误信息 |
| `fatal` | 致命错误级别 | 记录致命错误后程序退出 |
| `panic` | 恐慌级别 | 最高级别，记录后程序panic |

## 配置方式

### 1. 环境变量配置

在 `.env` 文件中设置：
```bash
LOG_LEVEL=debug
```

### 2. Docker环境配置

通过docker-compose启动时设置：
```bash
LOG_LEVEL=debug docker-compose up -d
```

或在docker-compose.yml中修改环境变量。

### 3. 运行时动态调整

通过API接口动态调整（需要管理员权限）：

#### 获取当前日志级别
```bash
curl -X GET "http://localhost:8082/api/admin/logging/level" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

#### 设置日志级别
```bash
curl -X POST "http://localhost:8082/api/admin/logging/level" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{"level": "debug"}'
```

#### 测试日志级别
```bash
curl -X POST "http://localhost:8082/api/admin/logging/test" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

#### 获取日志配置信息
```bash
curl -X GET "http://localhost:8082/api/admin/logging/info" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## 日志输出示例

### trace级别
```json
{
  "level": "trace",
  "msg": "Cache key retrieved successfully",
  "key": "user:session:123",
  "time": "2024-01-01T10:00:00Z"
}
```

### debug级别
```json
{
  "level": "debug",
  "msg": "GORM logging set to Info level (shows all SQL)",
  "time": "2024-01-01T10:00:00Z"
}
```

### info级别
```json
{
  "level": "info",
  "msg": "Database connected successfully",
  "time": "2024-01-01T10:00:00Z"
}
```

### warn级别
```json
{
  "level": "warning",
  "msg": "Slow response",
  "method": "GET",
  "path": "/api/projects",
  "latency": "2.5s",
  "time": "2024-01-01T10:00:00Z"
}
```

### error级别
```json
{
  "level": "error",
  "msg": "Failed to connect to database",
  "error": "connection refused",
  "time": "2024-01-01T10:00:00Z"
}
```

## HTTP请求日志

系统会根据HTTP响应状态码和响应时间自动设置不同的日志级别：

- **500+状态码**: error级别
- **400-499状态码**: warn级别  
- **响应时间>1秒**: warn级别
- **300-399状态码**: info级别
- **200-299状态码**: debug级别

每个请求都会分配唯一的request_id便于追踪：

```json
{
  "level": "debug",
  "msg": "Successful response",
  "method": "GET",
  "path": "/api/health",
  "status": 200,
  "latency": "15ms",
  "client_ip": "127.0.0.1",
  "request_id": 1704110400123456789,
  "time": "2024-01-01T10:00:00Z"
}
```

## 数据库日志

GORM的日志级别会根据应用日志级别自动调整：

- **trace/debug**: 显示所有SQL语句
- **info**: 只显示慢查询和错误
- **warn/error及以上**: 只显示错误

## 缓存日志

Redis操作在不同级别下的日志输出：

- **trace**: 记录所有成功的缓存操作
- **debug**: 记录缓存键的设置和获取
- **info**: 连接状态信息
- **error**: 缓存操作失败

## 最佳实践

### 开发环境
```bash
LOG_LEVEL=debug
```
便于调试和问题排查。

### 测试环境
```bash
LOG_LEVEL=info
```
记录重要操作，不过度冗余。

### 生产环境
```bash
LOG_LEVEL=warn
```
只记录警告和错误，减少日志量。

### 问题排查
临时通过API调整为debug或trace级别，排查完成后恢复到原来级别。

## 注意事项

1. trace和debug级别会产生大量日志，谨慎在生产环境使用
2. 日志级别调整是全局的，会影响所有组件的日志输出
3. 数据库密码等敏感信息不会出现在日志中
4. 运行时调整的日志级别不会持久化，重启后恢复为配置文件中的级别
5. 管理员权限才能调整日志级别

## 故障排查

如果日志功能异常：

1. 检查环境变量LOG_LEVEL是否设置正确
2. 验证日志级别拼写是否正确
3. 确认管理员权限是否正常
4. 查看容器启动日志是否有错误信息
