# SaltStack 集成环境变量配置说明

## 概述

本次修复将 SaltStack API 连接配置从硬编码改为环境变量方式，提高了系统的灵活性和可维护性。

## 问题背景

之前 `/api/slurm/saltstack/integration` 接口返回演示数据（demo=true），无法获取真实的 SaltStack 集群状态。原因是：

1. Salt API 端口配置错误（8002 → 8000）
2. 认证方式配置不正确（pam → file）
3. 所有配置都硬编码在代码中，无法灵活调整

## 修复内容

### 1. 代码修改

#### `slurm_controller.go`

修改了 `getRealSaltStackStatus` 方法，从环境变量读取配置：

```go
// 从环境变量读取 Salt API 配置
saltAPIURL := c.getSaltAPIURL()
username := c.getEnv("SALT_API_USERNAME", "saltapi")
password := c.getEnv("SALT_API_PASSWORD", "saltapi123")
eauth := c.getEnv("SALT_API_EAUTH", "file")
timeout := c.getEnvDuration("SALT_API_TIMEOUT", 8*time.Second)
```

新增了辅助方法：
- `getSaltAPIURL()`: 获取 Salt API URL
- `getEnv()`: 获取字符串环境变量
- `getEnvDuration()`: 获取 Duration 类型环境变量

### 2. 环境变量配置

#### 完整 URL 方式（推荐）

```bash
SALTSTACK_MASTER_URL=http://salt-master:8000
```

#### 分别配置方式

如果 `SALTSTACK_MASTER_URL` 未设置，将使用以下变量组合：

```bash
SALT_API_SCHEME=http
SALT_MASTER_HOST=salt-master
SALT_API_PORT=8000
```

#### 认证配置

```bash
SALT_API_USERNAME=saltapi
SALT_API_PASSWORD=saltapi123
SALT_API_EAUTH=file
```

#### 超时配置（可选）

```bash
SALT_API_TIMEOUT=8s
```

### 3. 配置文件更新

更新了以下配置文件：

1. **`src/backend/.env.example`**
   - 添加完整的 Salt API 配置说明

2. **`.env.example`**
   - 修正了 `SALTSTACK_MASTER_URL` 的默认端口（8002 → 8000）
   - 修正了 `SALT_MASTER_HOST`（saltstack → salt-master）
   - 添加了 `SALT_API_TIMEOUT` 配置项
   - 更新了认证方式说明

## 使用方法

### 1. 开发环境

复制示例配置文件并根据实际情况修改：

```bash
# 后端环境变量
cp src/backend/.env.example src/backend/.env

# 主环境变量
cp .env.example .env
```

### 2. 生产环境

在 `.env` 文件中配置：

```bash
# 完整 URL 方式
SALTSTACK_MASTER_URL=http://your-salt-master:8000
SALT_API_USERNAME=your-username
SALT_API_PASSWORD=your-password
SALT_API_EAUTH=file
```

或者使用组合方式：

```bash
SALT_API_SCHEME=https
SALT_MASTER_HOST=your-salt-master.example.com
SALT_API_PORT=8443
SALT_API_USERNAME=your-username
SALT_API_PASSWORD=your-password
SALT_API_EAUTH=pam
```

### 3. Docker Compose 部署

环境变量会自动从 `.env` 文件加载到容器中：

```bash
docker-compose build backend
docker-compose up -d backend
```

### 4. Kubernetes 部署

在 ConfigMap 或 Secret 中配置：

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: saltstack-config
data:
  SALTSTACK_MASTER_URL: "http://salt-master:8000"
  SALT_API_USERNAME: "saltapi"
  SALT_API_EAUTH: "file"
---
apiVersion: v1
kind: Secret
metadata:
  name: saltstack-secret
type: Opaque
stringData:
  SALT_API_PASSWORD: "your-password"
```

## 配置优先级

环境变量读取优先级：

1. **SALTSTACK_MASTER_URL** - 如果设置，直接使用
2. **SALT_API_SCHEME + SALT_MASTER_HOST + SALT_API_PORT** - 如果 URL 未设置，组合这些变量
3. **默认值** - 如果都未设置，使用代码中的默认值

## 验证配置

### 测试 Salt API 连接

```bash
# 获取 token
TOKEN=$(curl -s -X POST http://192.168.0.200:8082/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | jq -r '.token')

# 测试集成接口
curl -s -H "Authorization: Bearer $TOKEN" \
  http://192.168.0.200:8082/api/slurm/saltstack/integration | jq '.'
```

### 期望输出

```json
{
  "data": {
    "demo": false,
    "enabled": true,
    "master_status": "connected",
    "api_status": "connected",
    "minions": {
      "total": 4,
      "online": 4,
      "offline": 0
    },
    "minion_list": [
      {"id": "salt-master-local", "name": "salt-master-local", "status": "online"},
      {"id": "test-ssh01", "name": "test-ssh01", "status": "online"},
      {"id": "test-ssh02", "name": "test-ssh02", "status": "online"},
      {"id": "test-ssh03", "name": "test-ssh03", "status": "online"}
    ],
    "services": {
      "salt-master": "running",
      "salt-api": "running"
    }
  }
}
```

## 常见问题

### Q1: 为什么端口从 8002 改为 8000？

**A:** Salt API 的默认端口是 8000，之前配置的 8002 是错误的。

### Q2: 为什么认证方式从 pam 改为 file？

**A:** 在 Docker 容器环境中，`file` 方式更简单可靠，不依赖系统 PAM 配置。

### Q3: 如何自定义超时时间？

**A:** 设置 `SALT_API_TIMEOUT` 环境变量，例如：
```bash
SALT_API_TIMEOUT=15s  # 15秒
SALT_API_TIMEOUT=2m   # 2分钟
```

### Q4: 连接失败怎么办？

**A:** 检查以下几点：

1. Salt API 服务是否运行：
   ```bash
   docker-compose ps salt-master
   ```

2. 端口是否正确：
   ```bash
   curl http://salt-master:8000
   ```

3. 用户名密码是否正确：
   ```bash
   curl -X POST http://salt-master:8000/login \
     -H "Content-Type: application/json" \
     -d '{"username":"saltapi","password":"saltapi123","eauth":"file"}'
   ```

4. 查看后端日志：
   ```bash
   docker-compose logs -f backend
   ```

## 相关文件

- `src/backend/internal/controllers/slurm_controller.go` - 控制器实现
- `src/backend/.env.example` - 后端环境变量示例
- `.env.example` - 主环境变量示例
- `docs/SALTSTACK_INTEGRATION_ENV_CONFIG.md` - 本文档

## 更新日志

- **2025-10-21**: 初始版本，实现环境变量配置
  - 修正 Salt API 默认端口
  - 修正认证方式
  - 添加超时配置
  - 更新配置文件

## 测试验证

请参考 `test/e2e/specs/saltstack-integration-fix-test.spec.js` 进行自动化测试。
