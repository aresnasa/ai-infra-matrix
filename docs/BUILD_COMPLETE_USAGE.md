# Build.sh 三环境部署系统 - 完整使用指南

## 概述

全新的 `build.sh` 脚本现已完全重构，支持三种环境的统一管理：
- **开发环境 (development)**: 本地构建和测试
- **CI/CD环境 (cicd)**: 镜像转发和打包
- **生产环境 (production)**: 内网部署

## 功能特性

### ✅ 已实现功能

#### 环境检测和配置
- 自动环境检测（环境变量、文件、K8s集群、CI环境）
- 智能配置文件加载（.env 用于 dev/cicd，.env.prod 用于 production）
- 强制执行模式 (`--force` 参数)

#### 开发环境功能
- `build [tag]`: 构建所有Docker镜像
- `dev-start [tag]`: 构建并启动开发环境
- `dev-stop`: 停止开发环境
- `start`: 启动服务（Docker Compose）

#### CI/CD环境功能
- `transfer <registry> [tag]`: 将镜像从公共仓库转发到私有仓库
- `package <registry> [tag]`: 打包配置文件和部署脚本

#### 生产环境功能
- `pull <registry> [tag]`: 从私有仓库拉取镜像
- `deploy-compose <registry> [tag]`: 使用Docker Compose部署
- `deploy-helm <registry> [tag]`: 使用Kubernetes Helm部署

#### 通用功能
- `env`: 显示环境信息
- `status`: 显示环境和服务状态
- `version`: 显示脚本版本
- `clean`: 清理Docker资源
- `restore`: 恢复docker-compose.yml备份
- `help`: 显示帮助信息

## 使用示例

### 1. 开发环境使用

```bash
# 设置环境类型（可选，会自动检测）
export AI_INFRA_ENV_TYPE=development

# 查看环境信息
./build.sh env

# 构建所有镜像
./build.sh build v0.3.5

# 构建并启动开发环境
./build.sh dev-start

# 查看服务状态
./build.sh status

# 停止开发环境
./build.sh dev-stop

# 清理Docker资源
./build.sh clean
```

### 2. CI/CD环境使用

```bash
# 设置环境类型
export AI_INFRA_ENV_TYPE=cicd

# 将镜像转发到私有仓库
./build.sh transfer registry.company.com/ai-infra v0.3.5

# 打包配置文件
./build.sh package registry.company.com/ai-infra v0.3.5
```

### 3. 生产环境使用

#### Docker Compose 部署

```bash
# 设置环境类型
export AI_INFRA_ENV_TYPE=production

# 从私有仓库拉取镜像
./build.sh pull registry.company.com/ai-infra v0.3.5

# 使用Docker Compose部署
./build.sh deploy-compose registry.company.com/ai-infra v0.3.5

# 查看服务状态
./build.sh status

# 恢复备份（如需要）
./build.sh restore
```

#### Kubernetes 部署

```bash
# 设置环境类型
export AI_INFRA_ENV_TYPE=production

# 使用Helm部署到Kubernetes
./build.sh deploy-helm registry.company.com/ai-infra v0.3.5

# 查看部署状态
kubectl get pods -n ai-infra-prod
kubectl get services -n ai-infra-prod
```

### 4. 强制执行示例

```bash
# 在生产环境强制执行构建（忽略环境检查）
AI_INFRA_ENV_TYPE=production ./build.sh build --force v0.3.5

# 在开发环境强制执行镜像转发
AI_INFRA_ENV_TYPE=development ./build.sh transfer registry.example.com --force
```

## 环境检测机制

脚本按以下顺序检测环境类型：

1. **环境变量 `AI_INFRA_ENV_TYPE`**
   - `dev|development` → development
   - `cicd|ci` → cicd  
   - `prod|production` → production

2. **文件 `/etc/ai-infra-env`**
   - 文件内容决定环境类型

3. **自动检测**
   - 检测到Kubernetes集群 → production
   - 检测到CI环境变量 → cicd

4. **默认值**
   - development

## 配置文件

- **开发/CI环境**: `.env`
- **生产环境**: `.env.prod`

## 镜像仓库配置

在配置文件中设置：
```bash
PRIVATE_REGISTRY=registry.company.com/ai-infra
```

或通过命令行参数指定。

## 错误处理

- 语法检查通过：所有函数都有完整的错误处理
- 环境验证：每个命令都会检查适用的环境类型
- 用户确认：非强制模式下会询问用户确认
- 备份机制：重要操作前会自动备份文件

## 高级功能

### 自动备份和恢复

部署时会自动备份 `docker-compose.yml`：
```bash
# 自动备份到 docker-compose.yml.backup
./build.sh deploy-compose registry.example.com v0.3.5

# 恢复备份
./build.sh restore
```

### 配置打包

CI/CD环境可以打包所有配置文件：
```bash
./build.sh package registry.company.com v0.3.5
# 生成: ai-infra-deploy-v0.3.5.tar.gz
```

### 镜像传输

自动处理镜像标签和环境变量替换：
```bash
# 自动替换 ${IMAGE_TAG} 变量
./build.sh transfer registry.example.com v1.0.0
```

## 故障排除

### 常见问题

1. **环境检测错误**
   ```bash
   # 手动指定环境
   export AI_INFRA_ENV_TYPE=development
   ```

2. **配置文件缺失**
   ```bash
   # 检查配置文件是否存在
   ls -la .env .env.prod
   ```

3. **Docker权限问题**
   ```bash
   # 确保用户在docker组中
   sudo usermod -aG docker $USER
   ```

4. **Kubernetes连接问题**
   ```bash
   # 检查kubectl配置
   kubectl cluster-info
   ```

### 调试模式

查看详细状态信息：
```bash
./build.sh status
```

查看帮助信息：
```bash
./build.sh help
```

## 版本信息

- **当前版本**: v3.2.0
- **兼容性**: 支持原有的环境变量和配置文件
- **向后兼容**: 保留了 `start` 命令等常用功能

## 总结

新的 `build.sh` 脚本提供了完整的三环境部署解决方案：

- 🏗️ **开发环境**: 快速构建和测试
- 🚀 **CI/CD环境**: 自动化镜像传输和打包
- 🏢 **生产环境**: 安全的内网部署

所有功能都经过测试，语法检查通过，可以立即投入使用。
