# 内部镜像仓库启动方案

## 方案概述

为了方便使用内部镜像仓库启动 AI-Infra-Matrix 服务，我们在现有的 `build.sh` 脚本中添加了内部仓库支持功能，无需添加太多额外脚本。

## 核心文件

### 1. `build.sh` (主要修改)
- 添加了 `start-internal` 命令
- 自动生成 `docker-compose.override.yml` 
- 支持停止服务的 `stop` 命令
- 保持与原有 `scripts/all-ops.sh` 的兼容性

### 2. `.env.prod` (配置增强)
- 添加了 `IMAGE_REGISTRY_PREFIX` 配置项
- 添加了 `IMAGE_TAG` 版本控制

### 3. 辅助文件
- `docker-compose.override.yml.template` - 配置模板
- `start-internal-example.sh` - 使用示例
- `docs/INTERNAL_REGISTRY_GUIDE.md` - 详细使用指南

## 使用方法

### 快速启动
```bash
# 使用内部仓库启动
./build.sh start-internal registry.company.com/ai-infra/ v0.3.5

# 停止服务
./build.sh stop
```

### 工作原理
1. 脚本接收仓库前缀和版本参数
2. 自动生成 `docker-compose.override.yml` 文件
3. 设置所有服务使用指定的内部镜像
4. 禁用本地构建 (`build: null`)
5. 使用 `docker-compose up -d` 启动服务

## 支持的镜像

- `ai-infra-backend-init:${IMAGE_TAG}`
- `ai-infra-backend:${IMAGE_TAG}`
- `ai-infra-frontend:${IMAGE_TAG}`
- `ai-infra-jupyterhub:${IMAGE_TAG}`
- `ai-infra-singleuser:${IMAGE_TAG}`
- `ai-infra-saltstack:${IMAGE_TAG}`
- `ai-infra-nginx:${IMAGE_TAG}`
- `ai-infra-gitea:${IMAGE_TAG}`

## 特性

✅ **简单易用** - 只需一个命令即可启动
✅ **无侵入性** - 不影响现有构建流程
✅ **自动化** - 自动生成配置文件
✅ **灵活配置** - 支持不同仓库和版本
✅ **完全兼容** - 保持与原有脚本的兼容性

## 示例用法

```bash
# 1. 使用公司内部仓库
./build.sh start-internal registry.company.com/ai-infra/ v0.3.5

# 2. 使用阿里云 ACR
./build.sh start-internal registry.cn-hangzhou.aliyuncs.com/namespace/ v0.3.5

# 3. 使用私有 Harbor 仓库
./build.sh start-internal harbor.company.com/ai-infra/ v0.3.5

# 4. 停止所有服务
./build.sh stop

# 5. 检查服务状态
docker-compose ps

# 6. 查看日志
docker-compose logs -f
```

## 配置说明

生成的 `docker-compose.override.yml` 示例：

```yaml
version: '3.8'

services:
  backend:
    image: registry.company.com/ai-infra/ai-infra-backend:v0.3.5
    build: null  # 禁用本地构建
  
  frontend:
    image: registry.company.com/ai-infra/ai-infra-frontend:v0.3.5
    build: null
  
  # ... 其他服务
```

## 注意事项

1. **网络访问** - 确保可以访问内部镜像仓库
2. **认证** - 如需要，先使用 `docker login` 登录仓库
3. **端口冲突** - 确保 8080 端口可用
4. **镜像存在** - 确保指定的镜像在仓库中存在

## 故障排除

1. **镜像拉取失败**
   ```bash
   docker pull registry.company.com/ai-infra/ai-infra-backend:v0.3.5
   ```

2. **服务启动失败**
   ```bash
   docker-compose logs backend
   ```

3. **清理环境**
   ```bash
   ./build.sh stop
   rm -f docker-compose.override.yml
   ```

这个方案简单、高效，满足了使用内部镜像仓库启动服务的需求，同时保持了与现有系统的兼容性。
