# 阿里云容器镜像服务 (ACR) 支持指南

## 概述

`build.sh` 脚本现在完全支持阿里云容器镜像服务 (Alibaba Cloud Container Registry, ACR)。系统会自动检测阿里云注册表并应用正确的命名约定。

## 阿里云ACR命名规范

### 基本格式
```
registry.cn-region.aliyuncs.com/namespace/repository:tag
```

### 支持的镜像格式映射

对于ai-infra组件，系统会将所有组件映射到统一的repository中，使用tag进行区分：

| 源镜像名称 | ACR最终格式 |
|-----------|-------------|
| ai-infra-backend | `xxx.aliyuncs.com/ai-infra-matrix/ai-infra-matrix:backend-v0.0.3.3` |
| ai-infra-frontend | `xxx.aliyuncs.com/ai-infra-matrix/ai-infra-matrix:frontend-v0.0.3.3` |
| ai-infra-nginx | `xxx.aliyuncs.com/ai-infra-matrix/ai-infra-matrix:nginx-v0.0.3.3` |
| ai-infra-jupyterhub | `xxx.aliyuncs.com/ai-infra-matrix/ai-infra-matrix:jupyterhub-v0.0.3.3` |

## 使用方法

### 1. 登录阿里云ACR
```bash
# 替换为您的实际注册表地址
docker login xxx.aliyuncs.com
```

### 2. 构建并推送所有组件
```bash
# 方法1: 指定完整注册表路径（包含命名空间）
./scripts/build.sh prod \
  --registry xxx.aliyuncs.com/ai-infra-matrix \
  --push \
  --version v0.0.3.3

# 方法2: 仅指定注册表域名（使用默认命名空间 ai-infra-matrix）
./scripts/build.sh prod \
  --registry xxx.aliyuncs.com \
  --push \
  --version v0.0.3.3
```

### 3. 构建并推送单个组件
```bash
# 构建并推送backend
./scripts/build.sh prod \
  --registry xxx.aliyuncs.com/ai-infra-matrix \
  --push \
  --version v0.0.3.3 \
  --component backend

# 构建并推送frontend
./scripts/build.sh prod \
  --registry xxx.aliyuncs.com/ai-infra-matrix \
  --push \
  --version v0.0.3.3 \
  --component frontend
```

### 4. 推送依赖镜像到ACR
```bash
# 推送所有依赖到阿里云ACR
./scripts/build.sh prod \
  --push-deps \
  --deps-namespace xxx.aliyuncs.com/ai-infra-matrix \
  --version v0.0.3.3
```

## 配置文件更新

构建完成后，需要更新部署配置文件中的镜像地址：

### docker-compose.yml
```yaml
services:
  backend:
    image: xxx.aliyuncs.com/ai-infra-matrix/ai-infra-matrix:backend-v0.0.3.3
  
  frontend:
    image: xxx.aliyuncs.com/ai-infra-matrix/ai-infra-matrix:frontend-v0.0.3.3
  
  nginx:
    image: xxx.aliyuncs.com/ai-infra-matrix/ai-infra-matrix:nginx-v0.0.3.3
```

### Kubernetes部署文件
```yaml
spec:
  containers:
  - name: backend
    image: xxx.aliyuncs.com/ai-infra-matrix/ai-infra-matrix:backend-v0.0.3.3
```

## 自动检测特性

系统会自动检测注册表类型：

- **阿里云ACR检测**: 如果注册表地址包含 `.aliyuncs.com`，系统会自动应用ACR命名规范
- **其他注册表**: Docker Hub、Harbor、本地注册表等使用标准命名格式

## 命名空间处理

### 带命名空间的注册表地址
```bash
--registry xxx.aliyuncs.com/my-namespace
# 结果: xxx.aliyuncs.com/my-namespace/ai-infra-matrix:component-version
```

### 仅域名的注册表地址
```bash
--registry xxx.aliyuncs.com
# 结果: xxx.aliyuncs.com/ai-infra-matrix/ai-infra-matrix:component-version
# 自动使用默认命名空间 ai-infra-matrix
```

## 测试验证

运行测试脚本验证ACR命名逻辑：

```bash
./scripts/test-acr-naming.sh
```

该脚本会测试各种场景下的镜像命名转换，确保符合阿里云ACR规范。

## 常见问题

### Q: 为什么所有ai-infra组件使用同一个repository？
A: 这是为了符合阿里云ACR的最佳实践，通过tag区分不同组件，便于管理和权限控制。

### Q: 如何推送到不同的命名空间？
A: 在注册表地址中指定命名空间：`--registry xxx.aliyuncs.com/your-namespace`

### Q: 是否支持私有注册表实例？
A: 是的，支持任何 `.aliyuncs.com` 域名的注册表实例。

## 示例命令汇总

```bash
# 完整构建和推送流程
./scripts/build.sh prod \
  --registry registry.cn-hangzhou.aliyuncs.com/ai-infra-matrix \
  --push \
  --push-deps \
  --version v0.0.3.3 \
  --skip-existing-deps

# 仅推送依赖
./scripts/build.sh prod \
  --push-deps \
  --deps-namespace registry.cn-hangzhou.aliyuncs.com/ai-infra-matrix

# 测试构建（不推送）
./scripts/build.sh prod \
  --registry registry.cn-hangzhou.aliyuncs.com/ai-infra-matrix \
  --version v0.0.3.3
```
