# Backend 架构说明

## 双 main.go 架构

本项目采用**双 main.go 架构**，分别对应两个不同职责的可执行程序：

### 1. `src/backend/cmd/init/main.go` → backend-init 容器

**职责：一次性数据库初始化**

```
职责范围：
├── 数据库管理
│   ├── 检查并备份现有数据库
│   ├── 删除并重建主数据库
│   └── 创建子系统数据库（JupyterHub/Gitea/SLURM/Nightingale）
├── 数据初始化
│   ├── 运行 GORM AutoMigrate
│   ├── 初始化 RBAC 系统
│   ├── 创建默认 admin 用户
│   ├── 初始化 AI 配置
│   ├── 初始化 LDAP 用户（如果启用）
│   └── 同步 Gitea 用户（如果启用）
└── 执行模式
    └── 运行一次后退出
```

**特点：**
- 🔴 **破坏性操作**：会删除并重建数据库
- ⏱️ **一次性运行**：完成后容器退出
- 🚀 **优先执行**：在 docker-compose 中作为 init container
- 🔧 **环境准备**：为主服务创建必要的数据结构

**使用场景：**
- 首次部署系统
- 重置开发环境
- 测试环境清理

### 2. `src/backend/cmd/main.go` → backend 容器

**职责：持续运行的 API 服务**

```
职责范围：
├── HTTP 服务
│   ├── 启动 Gin Web 框架
│   ├── 注册所有 API 路由
│   ├── 提供 REST API
│   └── WebSocket 连接管理
├── 服务初始化
│   ├── 连接数据库（只读/增量）
│   ├── 连接 Redis 缓存
│   ├── 初始化 AI 网关
│   ├── 初始化作业管理服务
│   └── 启动后台任务
└── 执行模式
    └── 持续运行直到手动停止
```

**特点：**
- ✅ **安全操作**：只做增量更新，不删除数据
- ∞ **持续运行**：提供 API 服务
- 📡 **对外服务**：处理客户端请求
- 🔄 **自动恢复**：支持容器重启

**使用场景：**
- 生产环境服务
- 开发环境 API 调试
- 集成测试

## 为什么需要两个 main.go？

### ❌ 不应该合并的理由

1. **Docker 最佳实践 - Init Container 模式**
   ```yaml
   services:
     backend-init:
       image: backend
       command: /app/init  # 运行初始化程序
       restart: "no"       # 只运行一次
       
     backend:
       image: backend
       command: /app/backend  # 运行主服务
       depends_on:
         backend-init:
           condition: service_completed_successfully
   ```

2. **职责分离原则**
   - Init: 环境准备（破坏性）
   - Backend: 服务提供（安全性）

3. **启动顺序控制**
   - Init 必须先完成
   - Backend 依赖 Init 的结果
   - 通过 `depends_on` 确保顺序

4. **错误隔离**
   - Init 失败 → 阻止 Backend 启动
   - Backend 失败 → 不影响数据库状态

5. **镜像复用**
   - 同一个镜像
   - 不同的 entrypoint
   - 减少构建时间

## 有重复代码怎么办？

### 当前重复的部分

两个 main.go 都包含：
- RBAC 初始化
- admin 用户创建
- AI 配置初始化

### 优化方案

#### ✅ 方案1：保持现状（推荐）

**理由：**
- 重复代码量较少（< 100行）
- 逻辑简单，维护成本低
- 清晰的代码结构，易于理解
- 避免过度抽象

**适用场景：**
- 当前项目规模
- 重复逻辑简单

#### 🔄 方案2：提取共享包

```go
// internal/initialization/common.go
package initialization

func InitializeRBAC() error { ... }
func CreateDefaultAdmin() error { ... }
func InitializeAIConfigs() error { ... }
```

**理由：**
- 消除重复代码
- 统一初始化逻辑
- 便于单元测试

**缺点：**
- 增加抽象层
- 需要额外维护

**适用场景：**
- 初始化逻辑复杂
- 需要频繁修改

#### 🚫 方案3：合并为一个 main.go（不推荐）

```go
func main() {
    mode := os.Getenv("RUN_MODE") // "init" or "serve"
    if mode == "init" {
        runInit()
    } else {
        runServer()
    }
}
```

**缺点：**
- 违反单一职责原则
- 增加条件判断复杂度
- 降低代码可读性
- 不符合 Docker 最佳实践

## 构建配置

### Dockerfile

```dockerfile
# 构建 backend-init
FROM golang:1.21-alpine AS builder-init
WORKDIR /build
COPY . .
RUN go build -o init ./cmd/init

# 构建 backend
FROM golang:1.21-alpine AS builder-backend
WORKDIR /build
COPY . .
RUN go build -o backend ./cmd/main.go

# 最终镜像
FROM alpine:latest
COPY --from=builder-init /build/init /app/init
COPY --from=builder-backend /build/backend /app/backend
```

### docker-compose.yml

```yaml
services:
  backend-init:
    image: ai-infra-backend:latest
    command: /app/init
    restart: "no"
    
  backend:
    image: ai-infra-backend:latest
    command: /app/backend
    depends_on:
      backend-init:
        condition: service_completed_successfully
```

## 总结

✅ **保持当前的双 main.go 架构**
- 清晰的职责分离
- 符合 Docker 最佳实践
- 便于维护和调试
- 启动顺序可控

❌ **不要合并**
- 会破坏架构清晰性
- 增加维护复杂度
- 违反单一职责原则

## 相关文档

- [Docker Init Container Pattern](https://docs.docker.com/compose/startup-order/)
- [12-Factor App](https://12factor.net/)
- [Go Project Layout](https://github.com/golang-standards/project-layout)
