# 系统架构设计

## 概述

AI Infrastructure Matrix 是一个企业级 HPC 与 AI 基础设施平台，采用微服务架构和容器化部署。

## 整体架构

```text
┌─────────────────────────────────────────────────────────────┐
│                      External Access                        │
│              (Client Browsers / API Clients)                │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                    Reverse Proxy Layer                      │
│                      Nginx :8080                            │
│  ┌──────────┬──────────┬──────────┬──────────┬──────────┐  │
│  │   /      │  /api    │ /jupyter │  /gitea  │   /n9e   │  │
│  └──────────┴──────────┴──────────┴──────────┴──────────┘  │
└─────┬───────┬────────┬─────────┬────────┬────────┬─────────┘
      │       │        │         │        │        │
┌─────▼───────▼────────▼─────────▼────────▼────────▼─────────┐
│                   Application Services                       │
├──────────────┬────────────────┬──────────────┬──────────────┤
│   Frontend   │    Backend     │  JupyterHub  │    Gitea     │
│  (React SPA) │  (Go/FastAPI)  │  (Python)    │  (Go)        │
├──────────────┼────────────────┼──────────────┼──────────────┤
│ Nightingale  │ Slurm Master   │  SaltStack   │   AppHub     │
│ (Monitoring) │  (HPC Sched)   │  (Config)    │  (Packages)  │
└──────┬───────┴────────┬───────┴──────┬───────┴──────┬───────┘
       │                │              │              │
┌──────▼────────────────▼──────────────▼──────────────▼───────┐
│               Data & Storage Services                        │
├─────────────┬──────────────┬──────────────┬─────────────────┤
│ PostgreSQL  │    MySQL     │  OceanBase   │     Redis       │
│ (App Data)  │  (Slurm DB)  │  (Optional)  │  (Cache/MQ)     │
├─────────────┼──────────────┼──────────────┼─────────────────┤
│    Kafka    │    MinIO     │              │                 │
│ (Message Q) │  (Object S3) │              │                 │
└─────────────┴──────────────┴──────────────┴─────────────────┘
```

## 核心组件

### 前端层 (Frontend)

**技术栈**: React 18 + TypeScript + Ant Design

**职责**:

- 用户界面展示
- 路由管理
- 状态管理（Redux）
- API 调用封装

**关键模块**:

```typescript
src/frontend/
├── src/
│   ├── components/      // UI 组件
│   ├── pages/          // 页面
│   ├── services/       // API 服务
│   ├── store/          // Redux 状态
│   ├── utils/          // 工具函数
│   └── App.tsx         // 主应用
```

### 后端服务 (Backend)

**技术栈**: Go 1.21 + Gin/Fiber + GORM

**职责**:

- RESTful API 提供
- 业务逻辑处理
- 数据库交互
- 认证授权
- Slurm 集群管理
- SaltStack 集成
- Kubernetes 资源管理

**核心模块**:

```go
src/backend/
├── cmd/
│   └── main.go             // 入口
├── internal/
│   ├── api/                // API 路由
│   ├── services/           // 业务逻辑
│   │   ├── slurm_service.go
│   │   ├── saltstack_service.go
│   │   ├── k8s_service.go
│   │   └── user_service.go
│   ├── models/             // 数据模型
│   ├── middleware/         // 中间件
│   └── utils/              // 工具包
└── config/                 // 配置文件
```

### JupyterHub

**技术栈**: Python 3.11 + JupyterHub 4.0 + DockerSpawner

**职责**:

- 多用户 Jupyter 环境
- 用户认证集成
- 资源隔离
- GPU 支持

**配置**:

```python
# jupyterhub_config.py
c.JupyterHub.spawner_class = 'dockerspawner.DockerSpawner'
c.DockerSpawner.image = 'ai-infra-matrix/singleuser:v0.3.8'
c.DockerSpawner.network_name = 'ai-infra-network'
c.Spawner.cpu_limit = 2
c.Spawner.mem_limit = '4G'
```

### Gitea

**技术栈**: Go + SQLite/PostgreSQL

**职责**:

- Git 仓库托管
- Pull Request 工作流
- Webhook 集成
- LFS 大文件存储（MinIO 后端）

**集成**:

```ini
[server]
ROOT_URL = http://localhost:8080/gitea/

[lfs]
STORAGE_TYPE = minio
MINIO_ENDPOINT = minio:9000
MINIO_BUCKET = gitea
```

### Slurm Master

**技术栈**: Slurm 23.11 + Ubuntu 22.04

**职责**:

- 作业调度
- 资源分配
- 队列管理
- 节点管理

**架构**:

```text
┌─────────────────────────────────────┐
│         Slurm Master                │
├─────────────────────────────────────┤
│  slurmctld  (控制器守护进程)        │
│  slurmdbd   (数据库守护进程)        │
│  slurmrestd (REST API 服务)         │
└────────┬──────────────┬─────────────┘
         │              │
    ┌────▼────┐    ┌───▼────┐
    │  Node1  │    │ Node2  │
    │ slurmd  │    │ slurmd │
    └─────────┘    └────────┘
```

### SaltStack

**技术栈**: Salt 3006 + Python 3

**职责**:

- 配置管理
- 远程执行
- 状态管理
- Minion 部署

**架构**:

```text
┌──────────────────────┐
│    Salt Master       │
│   (配置中心)         │
└──────────┬───────────┘
           │
    ┌──────┼──────┐
    │      │      │
┌───▼──┐ ┌▼────┐ ┌▼────┐
│Minion│ │Minion│ │Minion│
│Node1 │ │Node2 │ │Node3 │
└──────┘ └─────┘ └─────┘
```

### KeyVault 安全服务

**技术栈**: Go + HMAC-SHA256

**职责**:

- Salt Master 公钥安全分发
- 一次性令牌生成与验证
- 密钥安全存储
- 请求签名验证

**安全分发流程**:

```text
┌─────────────────────────────────────────────────────────────┐
│                 安全密钥分发流程                             │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  1. 管理员/后端服务                                          │
│     │                                                       │
│     ├──► 调用 /api/keyvault/salt/generate-token             │
│     │                                                       │
│     ▼                                                       │
│  2. KeyVault 服务                                           │
│     │                                                       │
│     ├──► 生成一次性 Token (UUID)                            │
│     ├──► 生成 Nonce (随机 16 字节 hex)                       │
│     ├──► 计算 HMAC-SHA256 签名                              │
│     ├──► 存储 Token 到内存 (5 分钟过期)                      │
│     │                                                       │
│     ▼                                                       │
│  3. 返回给管理员                                             │
│     │   {token, signature, nonce, expires_at}               │
│     │                                                       │
│     ▼                                                       │
│  4. 批量安装脚本                                             │
│     │                                                       │
│     ├──► 携带 token + signature + nonce                     │
│     ├──► 调用 /api/keyvault/salt/master-pub                 │
│     │                                                       │
│     ▼                                                       │
│  5. KeyVault 验证                                           │
│     │                                                       │
│     ├──► 验证 Token 是否存在且未过期                         │
│     ├──► 验证 HMAC 签名                                     │
│     ├──► 验证 Nonce 未被使用过                              │
│     ├──► 标记 Token 为已使用（一次性）                       │
│     │                                                       │
│     ▼                                                       │
│  6. 返回 Master 公钥                                        │
│     │   {pub_key: "base64_encoded_key"}                     │
│     │                                                       │
│     ▼                                                       │
│  7. Minion 安装脚本                                         │
│     │                                                       │
│     └──► 将公钥写入 /etc/salt/pki/minion/minion_master.pub  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**安全特性**:

| 特性 | 描述 |
|------|------|
| 一次性令牌 | 令牌使用后立即销毁 |
| HMAC 签名 | 防止令牌篡改 |
| Nonce | 防止重放攻击 |
| 过期时间 | 默认 5 分钟有效期 |
| 请求超时 | 默认 10 秒超时 |

**API 端点**:

| 端点 | 方法 | 描述 |
|------|------|------|
| `/api/keyvault/salt/generate-token` | POST | 生成一次性令牌（需认证） |
| `/api/keyvault/salt/master-pub` | GET | 获取 Master 公钥（需令牌） |

### AppHub

**技术栈**: Ubuntu/RockyLinux + APK/RPM 构建工具

**职责**:

- Slurm 包构建（DEB/RPM/APK）
- Categraf 监控代理打包
- 多架构支持（x86_64/aarch64）
- 版本管理

**构建流程**:

```text
┌──────────────┐
│ Source Code  │
└──────┬───────┘
       │
┌──────▼───────┐
│Build Scripts │
│  (bash)      │
└──────┬───────┘
       │
┌──────▼───────┐     ┌──────────────┐
│ DEB Builder  ├────►│  APT Repo    │
└──────────────┘     └──────────────┘
┌──────────────┐     ┌──────────────┐
│ RPM Builder  ├────►│  YUM Repo    │
└──────────────┘     └──────────────┘
┌──────────────┐     ┌──────────────┐
│ APK Builder  ├────►│ Alpine Repo  │
└──────────────┘     └──────────────┘
```

### Nightingale

**技术栈**: Go + Vue.js + Prometheus

**职责**:

- 指标采集和存储
- 告警规则管理
- 仪表盘可视化
- 通知渠道集成

**组件**:

- n9e-server: 后端服务
- n9e-webapi: API 服务
- Categraf: 指标采集器
- Prometheus: 时序数据库

## 数据存储

### PostgreSQL

**用途**: 主数据库

**数据表**:

- users: 用户信息
- projects: 项目数据
- jupyterhub_*: JupyterHub 数据
- gitea_*: Gitea 数据

### MySQL

**用途**: Slurm 作业数据库

**数据表**:

- job_table: 作业记录
- assoc_table: 关联表
- cluster_table: 集群信息

### Redis

**用途**:

- 会话缓存
- 消息队列
- 分布式锁
- 临时数据存储

### MinIO

**用途**: 对象存储

**存储桶**:

- gitea: Gitea LFS 数据
- jupyter: JupyterHub 用户文件
- backups: 备份文件

## 网络架构

### Docker Compose 网络

```yaml
networks:
  ai-infra-network:
    driver: bridge
    ipam:
      config:
        - subnet: 172.20.0.0/16
```

### 服务间通信

- 服务名作为 DNS（如 `postgres`, `redis`, `backend`）
- 内部端口通信（不对外暴露）
- Nginx 作为统一入口

### 端口映射

| 服务 | 内部端口 | 外部端口 | 协议 |
|------|---------|---------|------|
| Nginx | 80 | 8080 | HTTP |
| Backend | 8000 | - | HTTP |
| JupyterHub | 8000 | - | HTTP |
| Gitea | 3000 | - | HTTP |
| PostgreSQL | 5432 | - | TCP |
| MySQL | 3306 | - | TCP |
| Redis | 6379 | - | TCP |
| MinIO | 9000 | - | HTTP |

## 安全架构

### 认证与授权

```text
┌──────────────┐
│   Client     │
└──────┬───────┘
       │ 1. Login
┌──────▼───────┐
│   Backend    │
│ (Auth API)   │
└──────┬───────┘
       │ 2. Verify
┌──────▼───────┐
│  PostgreSQL  │
│  (Users DB)  │
└──────┬───────┘
       │ 3. JWT Token
┌──────▼───────┐
│   Client     │
│ (Store Token)│
└──────────────┘
```

### RBAC 权限模型

```text
┌─────────────────────────────────────────────────────────────┐
│                    RBAC 权限模型                             │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  User (用户)                                                │
│    │                                                        │
│    ├──► role_template (角色模板)                            │
│    │     ├── admin         : 系统管理员                     │
│    │     ├── sre           : SRE运维工程师                  │
│    │     ├── data-developer: 数据开发人员                   │
│    │     ├── model-developer: 模型开发人员                  │
│    │     └── engineer      : 工程研发人员                   │
│    │                                                        │
│    └──► roles (角色关联)                                    │
│          │                                                  │
│          └──► permissions (权限)                            │
│               ├── resource : 资源类型 (projects, hosts...)  │
│               ├── verb     : 操作 (create, read, update...) │
│               └── scope    : 范围 (* 或 own)                │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**角色模板权限示例**:

| 角色模板 | 资源权限 |
|----------|----------|
| `admin` | `*:*:*` (所有权限) |
| `sre` | `saltstack:*:*`, `ansible:*:*`, `kubernetes:*:*`, `hosts:read:*` |
| `data-developer` | `projects:create:*`, `jupyterhub:*:*`, `hosts:read:*` |
| `model-developer` | `jupyterhub:*:own`, `projects:read:*` |
| `engineer` | `kubernetes:*:*`, `projects:*:own` |

### KeyVault 安全密钥分发

```text
┌─────────────────────────────────────────────────────────────┐
│              KeyVault 安全分发机制                           │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ┌────────────┐      ┌──────────────┐      ┌────────────┐  │
│  │   Admin    │ ───► │  KeyVault    │ ───► │  Minion    │  │
│  │  Request   │      │   Service    │      │   Node     │  │
│  └────────────┘      └──────────────┘      └────────────┘  │
│                                                             │
│  安全措施:                                                   │
│  ├── HMAC-SHA256 签名验证                                   │
│  ├── 一次性令牌 (使用后销毁)                                │
│  ├── Nonce 防重放                                           │
│  ├── 5 分钟有效期                                           │
│  └── 请求超时限制                                           │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 数据加密

- 传输层: HTTPS/TLS
- 存储层: 数据库加密
- 密码: bcrypt 哈希

### 网络隔离

- 服务运行在独立网络
- 仅 Nginx 对外暴露
- 防火墙规则限制

## 扩展性设计

### 水平扩展

```yaml
# 增加副本数
backend:
  deploy:
    replicas: 3
    
frontend:
  deploy:
    replicas: 2
```

### 负载均衡

```nginx
upstream backend {
    server backend-1:8000;
    server backend-2:8000;
    server backend-3:8000;
}
```

### 数据库扩展

- PostgreSQL: 主从复制 + 读写分离
- MySQL: InnoDB Cluster
- Redis: Cluster 模式
- MinIO: 分布式模式

## 监控与日志

### 监控体系

```text
┌──────────────────────────────────┐
│       Categraf Agents            │
│  (各服务和节点上的采集器)         │
└──────────┬───────────────────────┘
           │ Metrics
┌──────────▼───────────────────────┐
│      Prometheus TSDB             │
│     (时序数据存储)                │
└──────────┬───────────────────────┘
           │ Query
┌──────────▼───────────────────────┐
│     Nightingale Server           │
│   (告警规则引擎)                  │
└──────────┬───────────────────────┘
           │ Alert
┌──────────▼───────────────────────┐
│   Notification Channels          │
│  (Webhook/Email/SMS/DingTalk)    │
└──────────────────────────────────┘
```

### 日志体系

- 应用日志: 写入容器标准输出
- 系统日志: syslog
- 审计日志: PostgreSQL
- 日志聚合: ELK/Loki（可选）

## 高可用设计

### 服务高可用

- 多副本部署
- 健康检查
- 自动重启
- 滚动更新

### 数据高可用

- 数据库主从复制
- 定期备份
- 快照恢复
- 异地容灾

## 部署模式

### 单机部署

```bash
# Docker Compose
docker compose up -d
```

### 集群部署

```bash
# Kubernetes + Helm
helm install ai-infra ./helm/ai-infra-matrix
```

### 混合部署

- 控制平面: Kubernetes
- 计算节点: 物理机/虚拟机

## 技术选型理由

| 组件 | 选型 | 理由 |
|------|------|------|
| 前端框架 | React | 生态丰富、组件化、性能好 |
| 后端语言 | Go | 高性能、并发友好、部署简单 |
| 数据库 | PostgreSQL | 功能完善、性能优秀、开源 |
| 缓存 | Redis | 高性能、数据结构丰富 |
| 对象存储 | MinIO | S3 兼容、开源、易部署 |
| 监控 | Nightingale | 国产化、功能完善、易用 |
| 调度器 | Slurm | HPC 标准、功能强大 |
| 配置管理 | SaltStack | 灵活、强大、Python 生态 |

## 相关文档

- [项目结构](PROJECT_STRUCTURE.md)
- [部署指南](QUICK_START.md)
- [API 文档](API_REFERENCE.md)
- [开发指南](DEVELOPMENT_SETUP.md)
- [认证系统设计](AUTHENTICATION.md)
- [Salt Key 安全分发](../docs-all/SALT_KEY_SECURITY.md)
