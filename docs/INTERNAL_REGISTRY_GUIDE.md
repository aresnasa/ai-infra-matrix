# 内部镜像仓库启动指南

本指南说明如何使用内部镜像仓库启动 AI-Infra-Matrix 服务。

## 快速开始

### 1. 使用内部仓库启动服务

```bash
# 基本用法
./build.sh start-internal <镜像仓库前缀> [镜像标签]

# 示例：使用公司内部仓库
./build.sh start-internal registry.company.com/ai-infra/ v0.3.5

# 使用默认标签 (v0.3.5)
./build.sh start-internal registry.company.com/ai-infra/

# 使用阿里云 ACR
./build.sh start-internal registry.cn-hangzhou.aliyuncs.com/namespace/ v0.3.5
```

### 2. 停止服务

```bash
./build.sh stop
```

### 3. 查看帮助

```bash
./build.sh help
```

## 配置说明

### 环境变量配置

脚本会自动在 `.env.prod` 文件中查找以下配置：

```bash
# 镜像仓库前缀
IMAGE_REGISTRY_PREFIX=registry.company.com/ai-infra/

# 镜像版本标签
IMAGE_TAG=v0.3.5
```

### 所需镜像列表

脚本会从内部仓库拉取以下镜像：

- `ai-infra-backend-init:${IMAGE_TAG}`
- `ai-infra-backend:${IMAGE_TAG}`
- `ai-infra-frontend:${IMAGE_TAG}`
- `ai-infra-jupyterhub:${IMAGE_TAG}`
- `ai-infra-singleuser:${IMAGE_TAG}`
- `ai-infra-saltstack:${IMAGE_TAG}`
- `ai-infra-nginx:${IMAGE_TAG}`
- `ai-infra-gitea:${IMAGE_TAG}`

## 工作原理

1. **生成 Override 文件**: 脚本会自动生成 `docker-compose.override.yml` 文件，覆盖默认的镜像配置
2. **禁用构建**: 通过设置 `build: null` 禁用本地构建，直接使用镜像仓库的镜像
3. **启动服务**: 使用 `docker-compose up -d` 启动所有服务
4. **状态检查**: 启动后自动检查服务状态

## 注意事项

### 镜像仓库访问

确保您的环境可以访问内部镜像仓库：

```bash
# 测试仓库连接
docker pull registry.company.com/ai-infra/ai-infra-backend:v0.3.5

# 如果需要登录
docker login registry.company.com
```

### 网络要求

- 确保 Docker 守护进程正在运行
- 确保可以访问内部镜像仓库
- 确保端口 8080 可用

### 故障排除

1. **镜像拉取失败**
   ```bash
   # 检查仓库连接
   docker pull <registry_prefix>ai-infra-backend:v0.3.5
   
   # 检查登录状态
   docker login <registry_host>
   ```

2. **服务启动失败**
   ```bash
   # 查看日志
   docker-compose logs
   
   # 查看特定服务日志
   docker-compose logs backend
   ```

3. **端口冲突**
   ```bash
   # 检查端口占用
   lsof -i :8080
   
   # 修改端口映射（在 docker-compose.override.yml 中）
   ```

## 清理

如果需要完全重置环境：

```bash
# 停止并删除所有容器
./build.sh stop
docker-compose down -v

# 删除 override 文件
rm -f docker-compose.override.yml

# 清理未使用的镜像
docker image prune -f
```

## 高级用法

### 自定义配置

如果需要自定义配置，可以直接编辑生成的 `docker-compose.override.yml` 文件：

```yaml
services:
  nginx:
    ports:
      - "9080:80"  # 使用不同端口
    environment:
      - CUSTOM_VAR=value
```

### 部分服务启动

```bash
# 只启动特定服务
docker-compose up -d backend frontend nginx
```

### 检查服务状态

```bash
# 查看所有服务状态
docker-compose ps

# 查看详细日志
docker-compose logs -f

# 查看特定服务日志
docker-compose logs -f backend
```
