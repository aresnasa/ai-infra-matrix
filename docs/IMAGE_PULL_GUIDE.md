# AI-Infra-Matrix 镜像拉取指南

## 概述

本指南说明如何使用 `scripts/build.sh` 脚本从远程注册表拉取 AI-Infra-Matrix 的所有组件镜像。

## 功能特性

- 🔽 **一键拉取**: 自动拉取所有 AI-Infra-Matrix 组件镜像
- 🏷️ **智能标签管理**: 自动处理注册表特定的镜像命名规则（如阿里云ACR）
- 🔄 **本地重标记**: 拉取后自动重新标记为本地标准名称
- ✅ **状态报告**: 详细的拉取结果统计和错误报告
- 🎯 **版本控制**: 支持指定特定版本或使用latest标签

## 基本用法

### 从阿里云ACR拉取镜像

```bash
# 拉取指定版本的镜像
./scripts/build.sh prod --registry crpi-jl2i63tqhvx30nje.cn-chengdu.personal.cr.aliyuncs.com/ai-infra-matrix --pull --version v0.0.3.3

# 拉取最新版本的镜像（包含latest标签）
./scripts/build.sh prod --registry crpi-jl2i63tqhvx30nje.cn-chengdu.personal.cr.aliyuncs.com/ai-infra-matrix --pull --version v0.0.3.3 --tag-latest
```

### 从其他Docker注册表拉取镜像

```bash
# 从私有注册表拉取
./scripts/build.sh prod --registry registry.example.com:5000 --pull --version v0.0.3.3

# 从Docker Hub拉取
./scripts/build.sh prod --registry docker.io/username --pull --version v0.0.3.3
```

## 参数说明

| 参数 | 必需 | 说明 |
|------|------|------|
| `--pull` | ✅ | 启用拉取模式 |
| `--registry` | ✅ | 指定源注册表地址 |
| `--version` | ✅ | 指定要拉取的镜像版本 |
| `--tag-latest` | ❌ | 同时拉取latest标签 |

## 拉取的镜像列表

脚本会自动拉取以下所有组件镜像：

1. **ai-infra-backend** - 后端API服务
2. **ai-infra-backend-init** - 后端初始化服务
3. **ai-infra-frontend** - 前端Web应用
4. **ai-infra-singleuser** - JupyterHub单用户镜像
5. **ai-infra-jupyterhub** - JupyterHub核心服务
6. **ai-infra-nginx** - Nginx反向代理
7. **ai-infra-gitea** - Gitea代码仓库服务

## 阿里云ACR特殊处理

脚本自动识别阿里云ACR注册表（`*.aliyuncs.com`），并应用特殊的命名规则：

### 原始镜像名称规则

- 本地镜像: `ai-infra-backend:v0.0.3.3`
- ACR镜像: `xxx.aliyuncs.com/ai-infra-matrix/ai-infra-matrix:backend-v0.0.3.3`

### 自动处理流程

1. 从ACR拉取: `xxx.aliyuncs.com/ai-infra-matrix/ai-infra-matrix:backend-v0.0.3.3`
2. 重新标记: `ai-infra-backend:v0.0.3.3`
3. 可选latest: `ai-infra-backend:latest`

## 使用示例

### 完整部署流程

```bash
# 步骤1: 从阿里云ACR拉取所有镜像
./scripts/build.sh prod --registry crpi-jl2i63tqhvx30nje.cn-chengdu.personal.cr.aliyuncs.com/ai-infra-matrix --pull --version v0.0.3.3

# 步骤2: 启动所有服务
docker compose up -d

# 或者一键拉取并启动
./scripts/build.sh prod --registry crpi-jl2i63tqhvx30nje.cn-chengdu.personal.cr.aliyuncs.com/ai-infra-matrix --pull --version v0.0.3.3 --up
```

### 版本管理

```bash
# 拉取开发版本
./scripts/build.sh dev --registry xxx.aliyuncs.com/ai-infra-matrix --pull --version dev-latest

# 拉取稳定版本
./scripts/build.sh prod --registry xxx.aliyuncs.com/ai-infra-matrix --pull --version v0.0.3.3

# 拉取并标记为latest
./scripts/build.sh prod --registry xxx.aliyuncs.com/ai-infra-matrix --pull --version v0.0.3.3 --tag-latest
```

## 输出示例

```
🔽 AI-Infra-Matrix 镜像拉取模式
================================
ℹ️  拉取模式: 从注册表拉取镜像
ℹ️  注册表: crpi-jl2i63tqhvx30nje.cn-chengdu.personal.cr.aliyuncs.com/ai-infra-matrix
ℹ️  镜像版本: v0.0.3.3
ℹ️  拉取时间: Wed Aug 20 13:24:50 CST 2025

--------------------
ℹ️  从注册表拉取镜像: crpi-jl2i63tqhvx30nje.cn-chengdu.personal.cr.aliyuncs.com/ai-infra-matrix/ai-infra-matrix:backend-v0.0.3.3
✅ 拉取成功: crpi-jl2i63tqhvx30nje.cn-chengdu.personal.cr.aliyuncs.com/ai-infra-matrix/ai-infra-matrix:backend-v0.0.3.3
ℹ️  重新标记为本地镜像: ai-infra-backend:v0.0.3.3
--------------------
...

🎉 镜像拉取完成！
================================
✅ 成功拉取: 7 个镜像

ℹ️  本地现在可用的AI-Infra-Matrix镜像:
ai-infra-backend      v0.0.3.3   abc123def456   2 minutes ago   1.2GB
ai-infra-frontend     v0.0.3.3   def456ghi789   2 minutes ago   150MB
...

ℹ️  现在您可以使用以下命令启动服务:
  ./scripts/build.sh --up                        # 启动所有服务
  docker compose up -d           # 或直接使用compose启动
```

## 故障排除

### 常见错误及解决方案

#### 1. 注册表认证失败

```bash
# 错误: unauthorized: authentication required
# 解决: 先登录到注册表
docker login crpi-jl2i63tqhvx30nje.cn-chengdu.personal.cr.aliyuncs.com
```

#### 2. 镜像不存在

```bash
# 错误: manifest unknown: manifest unknown
# 解决: 检查版本号是否正确
./scripts/build.sh prod --registry xxx --pull --version v0.0.3.2  # 尝试其他版本
```

#### 3. 网络连接问题

```bash
# 错误: dial tcp: lookup xxx.aliyuncs.com: no such host
# 解决: 检查网络连接和DNS设置
ping crpi-jl2i63tqhvx30nje.cn-chengdu.personal.cr.aliyuncs.com
```

### 调试模式

```bash
# 启用详细输出
set -x
./scripts/build.sh prod --registry xxx --pull --version v0.0.3.3
set +x
```

## 与其他功能的集成

### 拉取后立即启动

```bash
./scripts/build.sh prod --registry xxx --pull --version v0.0.3.3 --up
```

### 拉取后运行健康检查

```bash
./scripts/build.sh prod --registry xxx --pull --version v0.0.3.3 --up --test
```

### 查看拉取的镜像

```bash
# 拉取完成后查看本地镜像
docker images | grep ai-infra-

# 查看镜像详细信息
docker inspect ai-infra-backend:v0.0.3.3
```

## 最佳实践

1. **版本固定**: 生产环境总是使用具体版本号，避免使用latest
2. **批量操作**: 一次性拉取所有镜像，而不是单独拉取
3. **登录检查**: 拉取前确保已正确登录到注册表
4. **网络优化**: 在网络较好的环境下进行大量镜像拉取
5. **存储清理**: 定期清理不需要的旧版本镜像

## 相关文档

- [QUICK_START.md](./QUICK_START.md) - 快速开始指南
- [DEVELOPMENT_SETUP.md](./DEVELOPMENT_SETUP.md) - 开发环境设置
- [ACR_IMPLEMENTATION_SUMMARY.md](./ACR_IMPLEMENTATION_SUMMARY.md) - 阿里云ACR集成说明
