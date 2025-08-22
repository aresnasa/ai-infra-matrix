# AI Infrastructure Matrix Build.sh 使用指南

## 🚀 三环境统一部署方案

AI Infrastructure Matrix 现在支持三种环境的统一管理：

1. **开发环境 (Development)** - 本地开发和测试
2. **CI/CD环境 (CI/CD Server)** - 镜像构建和转发
3. **生产环境 (Production)** - 内网隔离部署

## 📋 前置要求

### 所有环境
- Docker 和 Docker Compose
- Bash 4.0+
- Git

### 开发环境额外要求
- 本地开发工具
- 至少 8GB 内存

### CI/CD环境额外要求
- 网络访问外网和内网仓库
- 足够的磁盘空间存储镜像

### 生产环境额外要求
- 仅能访问内网镜像仓库
- Kubernetes 集群（可选，用于 Helm 部署）

## 🔧 环境配置

### 方法1: 环境变量设置

```bash
# 开发环境
export AI_INFRA_ENV_TYPE=development

# CI/CD环境
export AI_INFRA_ENV_TYPE=cicd

# 生产环境
export AI_INFRA_ENV_TYPE=production
```

### 方法2: 系统配置文件

```bash
# 在服务器上创建环境标识文件
echo "production" | sudo tee /etc/ai-infra-env
```

### 方法3: 自动检测

脚本会自动检测环境：
- 检测到 Kubernetes → `production`
- 检测到 CI/CD 环境变量 → `cicd`
- 默认 → `development`

## 📖 使用方法

### 1. 开发环境使用流程

```bash
# 设置环境类型
export AI_INFRA_ENV_TYPE=development

# 查看当前配置
./build.sh env

# 构建所有镜像
./build.sh build v0.3.5

# 构建并启动开发环境
./build.sh dev-start v0.3.5

# 查看服务状态
docker-compose ps

# 停止开发环境
./build.sh dev-stop
```

#### 开发环境特点
- ✅ 本地Docker构建
- ✅ 调试模式启用
- ✅ 简单密码配置
- ✅ 单副本部署
- ✅ 热重载支持

### 2. CI/CD环境使用流程

```bash
# 设置环境类型
export AI_INFRA_ENV_TYPE=cicd

# 可选：启用本地镜像清理
export CLEANUP_LOCAL_IMAGES=true

# 查看当前配置
./build.sh env

# 转发镜像到内网仓库
./build.sh transfer registry.internal.com/ai-infra v0.3.5

# 查看转发状态
echo "镜像转发完成，请检查内网仓库"
```

#### CI/CD环境特点
- ✅ 从外网拉取镜像
- ✅ 推送到内网仓库
- ✅ 自动镜像清理（可选）
- ✅ 详细日志输出
- ✅ 失败重试机制

### 3. 生产环境使用流程

#### Docker Compose 部署

```bash
# 设置环境类型
export AI_INFRA_ENV_TYPE=production

# 查看当前配置
./build.sh env

# 从内网仓库部署（暂未完全实现）
./build.sh deploy-compose registry.internal.com/ai-infra v0.3.5

# 临时使用标准启动方式
./build.sh start
```

#### Kubernetes 部署

```bash
# 设置环境类型
export AI_INFRA_ENV_TYPE=production

# 检查Kubernetes连接
kubectl cluster-info

# 使用Helm部署（暂未完全实现）
./build.sh deploy-helm registry.internal.com/ai-infra v0.3.5
```

#### 生产环境特点
- ✅ 使用内网仓库镜像
- ✅ 生产级安全配置
- ✅ 多副本高可用
- ✅ 完整监控日志
- ✅ 自动健康检查

## 🛡️ 安全配置

### 生产环境密码修改

在生产环境部署前，必须修改 `.env.prod` 中的密码：

```bash
# 编辑生产环境配置
vim .env.prod

# 需要修改的关键配置
POSTGRES_PASSWORD=CHANGE_IN_PRODUCTION_PostgreSQL_2024!
REDIS_PASSWORD=CHANGE_IN_PRODUCTION_Redis_2024!
LDAP_ADMIN_PASSWORD=CHANGE_IN_PRODUCTION_LDAP_2024!
JWT_SECRET=CHANGE_IN_PRODUCTION_JWT_SECRET_2024_RANDOM_STRING_HERE
```

### 生成安全密码

```bash
# 生成随机密码
openssl rand -base64 32

# 生成JWT密钥
openssl rand -hex 64

# 生成Crypt Key
openssl rand -hex 32
```

## 🚦 命令选项

### 通用选项

```bash
--env <type>        # 强制指定环境类型 (development/cicd/production)
--force             # 强制执行，跳过环境检查
--verbose           # 详细输出
--dry-run           # 预览模式（计划中）
--cleanup           # 清理本地镜像（CI/CD环境）
```

### 使用示例

```bash
# 强制在开发环境执行镜像转发
./build.sh --env development --force transfer registry.internal.com/test v0.3.5

# 详细模式构建镜像
./build.sh --verbose build v0.3.5

# CI/CD环境转发并清理
./build.sh --cleanup transfer registry.internal.com/ai-infra v0.3.5
```

## 📊 实际使用场景

### 场景1: 开发者本地开发

```bash
# 开发者 A 在本地开发新功能
cd ai-infra-matrix
export AI_INFRA_ENV_TYPE=development

# 构建并启动开发环境
./build.sh dev-start

# 开发完成后停止
./build.sh dev-stop

# 提交代码到Git
git add .
git commit -m "Add new feature"
git push origin feature-branch
```

### 场景2: CI/CD 服务器自动化

```bash
#!/bin/bash
# CI/CD 构建脚本

# 设置环境
export AI_INFRA_ENV_TYPE=cicd
export CLEANUP_LOCAL_IMAGES=true

# 获取版本号
VERSION=$(git describe --tags --always)

# 转发镜像到内网
./build.sh transfer registry.internal.com/ai-infra $VERSION

# 构建部署包（计划中）
./build.sh package registry.internal.com/ai-infra $VERSION

echo "构建完成，版本: $VERSION"
```

### 场景3: 生产环境部署

```bash
# 生产环境管理员部署
export AI_INFRA_ENV_TYPE=production

# 检查环境配置
./build.sh env

# 使用Docker Compose部署
./build.sh start

# 或使用Kubernetes部署（计划中）
# ./build.sh deploy-helm registry.internal.com/ai-infra v0.3.5
```

## 🔍 故障排除

### 常见问题

1. **环境检测错误**
   ```bash
   # 手动设置环境类型
   export AI_INFRA_ENV_TYPE=development
   ./build.sh env
   ```

2. **Docker服务未运行**
   ```bash
   # 启动Docker服务
   sudo systemctl start docker
   # 或 macOS
   open -a Docker
   ```

3. **镜像转发失败**
   ```bash
   # 检查Docker登录状态
   docker info
   
   # 登录私有仓库
   docker login registry.internal.com
   ```

4. **权限问题**
   ```bash
   # 确保脚本可执行
   chmod +x build.sh
   
   # 检查Docker权限
   sudo usermod -aG docker $USER
   ```

### 调试方法

```bash
# 详细输出模式
./build.sh --verbose build v0.3.5

# 检查脚本语法
bash -n build.sh

# 逐步执行
bash -x build.sh env
```

## 📈 当前实现状态

### ✅ 已实现功能
- [x] 三环境自动检测和配置
- [x] 开发环境镜像构建
- [x] 开发环境启动/停止
- [x] CI/CD环境镜像转发
- [x] 环境安全检查
- [x] 详细日志输出
- [x] 参数验证

### 🚧 计划中功能
- [ ] 生产环境Docker Compose部署
- [ ] 生产环境Kubernetes Helm部署
- [ ] 配置文件打包功能
- [ ] 预览模式 (--dry-run)
- [ ] 自动密码生成
- [ ] 健康检查集成

### 📝 使用建议

1. **开发阶段**: 使用 `dev-start` 快速启动开发环境
2. **测试阶段**: 使用 `transfer` 命令准备镜像
3. **部署阶段**: 使用环境特定的部署命令
4. **维护阶段**: 定期检查环境配置和安全设置

## 📞 技术支持

如遇到问题：
1. 首先查看 `./build.sh help`
2. 检查环境配置 `./build.sh env`
3. 查看项目文档 `docs/` 目录
4. 提交 Issue 到项目仓库

---

**AI Infrastructure Matrix** - 让多环境部署变得简单！
