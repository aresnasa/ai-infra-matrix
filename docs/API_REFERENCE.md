# API 参考文档

## 概述

AI Infrastructure Matrix 提供 RESTful API 用于程序化访问和集成。

## 基础信息

- **Base URL**: `http://localhost:8080/api`
- **认证方式**: JWT Token
- **Content-Type**: `application/json`

## 认证

### 获取 Token

```http
POST /api/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "admin123"
}
```

**响应：**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "username": "admin",
    "role": "admin"
  }
}
```

### 使用 Token

在后续请求中添加 Authorization header：

```http
Authorization: Bearer <token>
```

## Slurm 集群管理 API

### 获取集群列表

```http
GET /api/slurm/clusters
Authorization: Bearer <token>
```

**响应：**

```json
{
  "clusters": [
    {
      "id": 1,
      "name": "default",
      "status": "active",
      "nodes": 3,
      "total_cpus": 24,
      "total_memory": "128GB"
    }
  ]
}
```

### 提交作业

```http
POST /api/slurm/jobs
Authorization: Bearer <token>
Content-Type: application/json

{
  "job_name": "test_job",
  "partition": "compute",
  "nodes": 1,
  "ntasks": 4,
  "script": "#!/bin/bash\nsleep 60"
}
```

**响应：**

```json
{
  "job_id": 12345,
  "status": "pending",
  "submit_time": "2025-11-18T13:00:00Z"
}
```

### 查询作业状态

```http
GET /api/slurm/jobs/{job_id}
Authorization: Bearer <token>
```

**响应：**

```json
{
  "job_id": 12345,
  "status": "running",
  "partition": "compute",
  "nodes": ["node01"],
  "start_time": "2025-11-18T13:01:00Z",
  "elapsed_time": "00:05:30"
}
```

### 取消作业

```http
DELETE /api/slurm/jobs/{job_id}
Authorization: Bearer <token>
```

### 获取节点列表

```http
GET /api/slurm/nodes
Authorization: Bearer <token>
```

**响应：**

```json
{
  "nodes": [
    {
      "name": "node01",
      "state": "allocated",
      "cpus": 8,
      "memory": "32GB",
      "jobs": [12345]
    },
    {
      "name": "node02",
      "state": "idle",
      "cpus": 8,
      "memory": "32GB",
      "jobs": []
    }
  ]
}
```

## SaltStack 管理 API

### 获取 Minion 列表

```http
GET /api/saltstack/minions
Authorization: Bearer <token>
```

### 执行命令

```http
POST /api/saltstack/execute
Authorization: Bearer <token>
Content-Type: application/json

{
  "target": "node01",
  "function": "test.ping"
}
```

### 部署配置

```http
POST /api/saltstack/deploy
Authorization: Bearer <token>
Content-Type: application/json

{
  "target": "node*",
  "state": "slurm.compute"
}
```

## JupyterHub API

### 获取用户列表

```http
GET /api/jupyterhub/users
Authorization: Bearer <token>
```

### 创建用户

```http
POST /api/jupyterhub/users
Authorization: Bearer <token>
Content-Type: application/json

{
  "username": "newuser",
  "admin": false
}
```

### 启动服务器

```http
POST /api/jupyterhub/users/{username}/server
Authorization: Bearer <token>
```

## 对象存储 API

MinIO 兼容 AWS S3 API，参考：

- [AWS S3 API 文档](https://docs.aws.amazon.com/s3/)
- [MinIO SDK](https://min.io/docs/minio/linux/developers/minio-drivers.html)

### Python 示例

```python
import boto3

s3 = boto3.client('s3',
    endpoint_url='http://localhost:8080/minio',
    aws_access_key_id='minioadmin',
    aws_secret_access_key='minioadmin'
)

# 上传文件
s3.upload_file('local.txt', 'bucket-name', 'remote.txt')

# 下载文件
s3.download_file('bucket-name', 'remote.txt', 'local.txt')
```

## 监控 API

### 获取系统指标

```http
GET /api/monitoring/metrics
Authorization: Bearer <token>
```

**响应：**

```json
{
  "cpu_usage": 45.5,
  "memory_usage": 68.2,
  "disk_usage": 52.1,
  "network_in": 1024000,
  "network_out": 512000
}
```

### 查询历史数据

```http
GET /api/monitoring/history?metric=cpu_usage&start=2025-11-18T00:00:00Z&end=2025-11-18T23:59:59Z
Authorization: Bearer <token>
```

## 错误码

| 状态码 | 说明 |
|--------|------|
| 200 | 成功 |
| 201 | 创建成功 |
| 400 | 请求参数错误 |
| 401 | 未授权（Token 无效或过期）|
| 403 | 权限不足 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

## SDK 和工具

### Go SDK

```go
import "github.com/aresnasa/ai-infra-matrix/sdk/go"

client := aiinfra.NewClient("http://localhost:8080", "your-token")
jobs, err := client.Slurm.ListJobs()
```

### Python SDK

```python
from aiinfra import Client

client = Client("http://localhost:8080", token="your-token")
jobs = client.slurm.list_jobs()
```

### CLI 工具

```bash
# 安装
go install github.com/aresnasa/ai-infra-matrix/cmd/aiinfra@latest

# 配置
aiinfra config set-url http://localhost:8080
aiinfra login admin

# 使用
aiinfra slurm jobs list
aiinfra slurm jobs submit --script job.sh
```

## Webhook

支持配置 Webhook 接收事件通知：

```json
{
  "url": "https://your-webhook.example.com",
  "events": ["job.completed", "job.failed", "node.down"],
  "secret": "your-webhook-secret"
}
```

## 限流

API 限流策略：
- 认证请求：10次/分钟
- 一般请求：100次/分钟
- 管理操作：50次/分钟

## 更多信息

- OpenAPI 规范：`/api/openapi.json`
- Swagger UI：`http://localhost:8080/api/docs`
- GraphQL：`http://localhost:8080/api/graphql`（实验性）
