# CI/CD一键构建和推送指南

## 概述

`build.sh cicd-build` 命令是专为CI/CD环境设计的一键构建和推送解决方案。它能够自动完成：
1. **拉取依赖基础镜像**
2. **构建AI-Infra服务镜像**  
3. **标记并推送基础镜像到内部registry**
4. **推送AI-Infra服务镜像到内部registry**

## 使用方法

### 基本用法

```bash
# 设置CI/CD环境
export AI_INFRA_ENV_TYPE=cicd

# 一键构建和推送到阿里云容器镜像服务(ACR)
./build.sh cicd-build xxx.aliyuncs.com/ai-infra-matrix v0.3.5

# 一键构建和推送到Harbor私有仓库
./build.sh cicd-build harbor.company.com/ai-infra v0.3.5

# 一键构建和推送到传统Docker Registry
./build.sh cicd-build registry.company.com v0.3.5
```

### 命令参数

- `<registry>`: 目标镜像仓库地址（必需）
- `[tag]`: 镜像标签，默认为 `v0.3.5`

### 环境要求

1. **Docker环境**: 需要Docker客户端和推送权限
2. **网络连接**: 能够访问公共镜像仓库和目标私有仓库
3. **认证配置**: 已配置目标registry的推送权限

## 支持的Registry格式

### 阿里云容器镜像服务(ACR)
```bash
./build.sh cicd-build xxx.aliyuncs.com/ai-infra-matrix v0.3.5
```
生成的镜像格式：
- AI-Infra服务: `xxx.aliyuncs.com/ai-infra-matrix/ai-infra-nginx:v0.3.5`
- 基础镜像: `xxx.aliyuncs.com/ai-infra-matrix/nginx:1.27-alpine`

### Harbor私有仓库
```bash
./build.sh cicd-build harbor.company.com/ai-infra v0.3.5
```
生成的镜像格式：
- AI-Infra服务: `harbor.company.com/ai-infra/ai-infra-nginx:v0.3.5`
- 基础镜像: `harbor.company.com/ai-infra/nginx:1.27-alpine`

### 传统Docker Registry
```bash
./build.sh cicd-build registry.company.com v0.3.5
```
生成的镜像格式：
- AI-Infra服务: `registry.company.com/ai-infra-nginx:v0.3.5`
- 基础镜像: `registry.company.com/nginx:1.27-alpine`

## 处理的镜像清单

### AI-Infra服务镜像 (5个)
1. `ai-infra-backend` - 后端API服务
2. `ai-infra-frontend` - 前端Web界面
3. `ai-infra-jupyterhub` - JupyterHub分布式计算环境
4. `ai-infra-nginx` - Nginx网关代理
5. `ai-infra-saltstack` - SaltStack配置管理

### 基础依赖镜像 (8个)
1. `nginx:1.27-alpine` - Web服务器
2. `osixia/openldap:stable` - LDAP认证服务
3. `osixia/phpldapadmin:stable` - LDAP管理界面
4. `postgres:15-alpine` - PostgreSQL数据库
5. `quay.io/minio/minio:latest` - 对象存储服务
6. `redis:7-alpine` - 内存数据库
7. `redislabs/redisinsight:latest` - Redis管理界面
8. `tecnativa/tcp-proxy` - TCP代理服务

## 执行流程

### 第一阶段：拉取基础镜像依赖
- 从公共镜像仓库拉取所有依赖的基础镜像
- 确保构建环境具备必要的镜像资源

### 第二阶段：构建AI-Infra服务镜像
- 使用项目根目录为构建上下文
- 为每个服务构建对应的镜像
- 直接标记为目标registry格式

### 第三阶段：标记并推送基础镜像
- 将基础镜像标记为目标registry格式
- 推送到私有仓库供部署使用

### 第四阶段：推送AI-Infra服务镜像
- 推送所有构建的AI-Infra服务镜像
- 完成整个部署包的准备

## 错误处理和调试

### 测试模式
```bash
# 模拟执行，不进行实际Docker操作
./build.sh cicd-build xxx.aliyuncs.com/ai-infra-matrix --skip-docker
```

### 强制执行
```bash
# 在非CI/CD环境中强制执行
./build.sh cicd-build xxx.aliyuncs.com/ai-infra-matrix --force
```

### 常见问题

1. **构建失败**: 检查Dockerfile语法和依赖文件是否存在
2. **推送失败**: 验证registry认证配置和网络连接
3. **权限问题**: 确保Docker daemon权限和registry推送权限

## 输出示例

成功执行后会看到类似输出：
```
[SUCCESS] 🎉 所有AI-Infra服务镜像构建和推送成功！
[SUCCESS] 🚀 项目已准备好在目标环境中部署
========================================
CI/CD一键构建和推送总结
========================================
  基础镜像拉取: 8/8 成功
  AI-Infra服务构建: 5/5 成功  
  基础镜像推送: 8/8 成功
  AI-Infra服务推送: 5/5 成功
```

## 后续部署

构建推送完成后，可以使用以下命令进行部署：

```bash
# Docker Compose部署
./build.sh deploy-compose xxx.aliyuncs.com/ai-infra-matrix v0.3.5

# Kubernetes Helm部署  
./build.sh deploy-helm xxx.aliyuncs.com/ai-infra-matrix v0.3.5
```

## 与现有命令的区别

- `build`: 仅构建镜像，不推送
- `build-for`: 构建并标记为目标registry格式，不推送
- `transfer`: 仅转发现有镜像，不构建
- `cicd-build`: **完整流程**，从拉取依赖到最终推送一步完成

这使得CI/CD管道能够通过单个命令完成所有镜像处理工作，简化了自动化部署流程。
