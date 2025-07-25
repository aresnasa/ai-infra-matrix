# AI异步架构Docker Compose测试指南

## 概述

本指南介绍如何使用Docker Compose测试AI助手的异步架构，包括消息队列、缓存服务、AI网关等核心组件。

## 架构组件

### 核心服务
- **Backend**: Go后端服务，包含AI异步处理逻辑
- **Redis**: 消息队列和缓存存储
- **PostgreSQL**: 主数据库
- **Frontend**: React前端应用

### 测试服务
- **ai-async-test**: 专门的AI异步功能测试容器
- **redis-insight**: Redis监控和管理界面

## 快速开始

### 1. 环境准备

确保系统已安装：
- Docker >= 20.0
- Docker Compose >= 2.0

```bash
# 检查Docker版本
docker --version
docker-compose --version
```

### 2. 快速功能测试

运行核心功能验证测试（推荐首次使用）：

```bash
# 进入项目目录
cd web-v2

# 运行快速测试
./quick-ai-test.sh
```

快速测试包含：
- ✅ 健康检查
- ✅ 快速聊天API
- ✅ 消息状态查询
- ✅ 集群操作提交
- ✅ Redis队列验证
- ✅ 简单性能测试

### 3. 完整测试套件

运行完整的AI异步架构测试：

```bash
# 运行完整测试套件
./run-ai-async-test.sh
```

完整测试包含：
- 🔧 消息队列功能测试
- 💾 缓存服务功能测试
- 🤖 AI网关功能测试
- 🔌 异步API功能测试
- ☸️ 集群操作功能测试
- 🚄 性能测试
- 💪 压力测试

## 手动测试

### 启动基础环境

```bash
# 启动基础服务
docker-compose up -d postgres redis openldap

# 启动后端服务
docker-compose up -d backend

# 启动前端服务  
docker-compose up -d frontend

# 启动监控服务
docker-compose --profile monitoring up -d redis-insight
```

### 运行特定测试

```bash
# 只运行AI异步测试
docker-compose --profile ai-test up --build ai-async-test

# 查看测试日志
docker-compose logs ai-async-test
```

### 服务访问

- **前端应用**: http://localhost:3001
- **后端API**: http://localhost:8082
- **Redis Insight**: http://localhost:8001
- **API文档**: http://localhost:8082/swagger/index.html

## 测试场景

### 1. 消息队列测试

验证AI消息的异步处理流程：

```bash
# 发送异步聊天请求
curl -X POST http://localhost:8082/api/ai/async/quick-chat \
  -H "Authorization: Bearer test-token-123" \
  -H "Content-Type: application/json" \
  -d '{"message": "测试异步处理", "context": "test"}'

# 查询消息处理状态
curl http://localhost:8082/api/ai/async/messages/{message_id}/status \
  -H "Authorization: Bearer test-token-123"
```

### 2. 集群操作测试

验证Kubernetes操作的异步处理：

```bash
# 提交集群操作
curl -X POST http://localhost:8082/api/ai/async/cluster-operations \
  -H "Authorization: Bearer test-token-123" \
  -H "Content-Type: application/json" \
  -d '{
    "operation": "get_pods",
    "parameters": {"namespace": "default"},
    "description": "获取Pod列表"
  }'

# 查询操作状态
curl http://localhost:8082/api/ai/async/operations/{operation_id}/status \
  -H "Authorization: Bearer test-token-123"
```

### 3. 缓存验证

检查Redis中的消息队列和缓存：

```bash
# 进入Redis容器
docker-compose exec redis redis-cli

# 查看消息队列
XLEN ai:chat:requests
XLEN ai:cluster:operations
XLEN ai:notifications

# 查看缓存键
KEYS ai:*
KEYS messages:*
KEYS user_*
```

### 4. 性能监控

通过Redis Insight监控系统性能：
1. 访问 http://localhost:8001
2. 连接到Redis (redis:6379)
3. 监控队列长度和处理速度

## 测试报告

### 报告位置

测试完成后，报告保存在：
- **快速测试**: `./quick-test-report-{timestamp}.txt`
- **完整测试**: `./test-reports/ai-async-{timestamp}/`

### 报告内容

- `comprehensive_report.md`: 完整测试报告
- `status_summary.txt`: 状态摘要
- `test_results.json`: 机器可读结果
- `performance_report.txt`: 性能基准
- `stress_test_report.txt`: 压力测试结果

## 故障排除

### 常见问题

1. **后端服务启动失败**
```bash
# 查看后端日志
docker-compose logs backend

# 检查数据库连接
docker-compose exec backend go version
```

2. **Redis连接失败**
```bash
# 测试Redis连接
docker-compose exec redis redis-cli ping

# 查看Redis日志
docker-compose logs redis
```

3. **测试超时**
```bash
# 增加等待时间
export TEST_TIMEOUT=300

# 重启服务
docker-compose restart backend
```

### 清理环境

```bash
# 停止所有服务
docker-compose --profile ai-test --profile monitoring down

# 清理数据卷
docker-compose down --volumes

# 清理Docker镜像
docker-compose down --rmi all
```

## 配置说明

### 环境变量

可以通过环境变量自定义测试配置：

```bash
# 设置日志级别
export LOG_LEVEL=debug

# 设置测试超时
export TEST_TIMEOUT=300

# 设置Redis配置
export REDIS_MAX_MEMORY=512m
```

### Docker Compose Profiles

- `default`: 基础服务 (postgres, redis, backend, frontend)
- `ai-test`: AI异步测试服务
- `monitoring`: 监控服务 (redis-insight)

```bash
# 启动特定profile
docker-compose --profile ai-test up
docker-compose --profile monitoring up
```

## 扩展测试

### 添加自定义测试

在 `tests/ai-async/` 目录下添加测试脚本：

```bash
# 创建自定义测试
vim tests/ai-async/test-custom.sh

# 在run-tests.sh中调用
echo "./tests/test-custom.sh" >> tests/ai-async/run-tests.sh
```

### 集成CI/CD

将测试集成到CI/CD流水线：

```yaml
# .github/workflows/ai-async-test.yml
name: AI Async Test
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - name: Run AI Async Tests
        run: |
          cd web-v2
          ./run-ai-async-test.sh
```

## 最佳实践

1. **开发环境**: 使用快速测试验证功能
2. **集成测试**: 使用完整测试套件验证所有组件
3. **性能测试**: 定期运行压力测试验证系统稳定性
4. **监控**: 使用Redis Insight监控生产环境性能

## 支持

如有问题，请检查：
1. Docker和Docker Compose版本
2. 系统资源是否充足 (至少4GB内存)
3. 端口是否被占用 (8082, 3001, 8001, 5432, 6379)
4. 网络连接是否正常
