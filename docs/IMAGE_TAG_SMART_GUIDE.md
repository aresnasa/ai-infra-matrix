# 智能镜像 Tag 管理指南

## 概述

`tag-localhost` 命令提供智能镜像 tag 管理功能，根据网络环境自动选择最佳策略：
- **公网环境**：优先使用原始镜像名称，创建兼容性别名
- **内网环境**：使用 Harbor 仓库镜像，创建标准别名

## 功能特性

### 1. 自动网络环境检测
- 自动检测公网/内网环境
- 根据环境选择最佳镜像策略
- 支持手动指定网络环境

### 2. 智能镜像别名管理
- 公网环境：`原始镜像` → `localhost/镜像`
- 内网环境：`Harbor镜像` → `原始镜像` + `localhost/镜像`

### 3. 自动提取 Dockerfile 依赖
- 扫描所有服务的 Dockerfile
- 自动提取 FROM 指令中的基础镜像
- 去重并批量处理

## 使用方法

### 基本用法

```bash
# 自动处理所有 Dockerfile 中的基础镜像（推荐）
./build.sh tag-localhost

# 处理单个镜像
./build.sh tag-localhost redis:7-alpine

# 处理多个镜像
./build.sh tag-localhost redis:7-alpine nginx:stable postgres:15-alpine
```

### 高级用法

```bash
# 强制公网模式
./build.sh tag-localhost --network external redis:7-alpine

# 强制内网模式
./build.sh tag-localhost --network internal

# 指定 Harbor 仓库地址
./build.sh tag-localhost --harbor my-harbor.com/repo redis:7-alpine

# 内网模式 + 自定义 Harbor
./build.sh tag-localhost --network internal --harbor custom-harbor.com/project
```

## 网络环境策略

### 公网环境 (external)

**检测条件**：
- 可以 ping 通 `8.8.8.8` 或 `mirrors.aliyun.com`
- 可以访问 `https://mirrors.aliyun.com/pypi/simple/`

**镜像策略**：
```
原始镜像: redis:7-alpine
↓ 创建
localhost/redis:7-alpine
```

**适用场景**：
- 公司办公网络
- 家庭网络
- VPN 连接后的网络
- 可以直接访问 Docker Hub 的环境

### 内网环境 (internal)

**检测条件**：
- 无法访问外网
- 环境变量 `AI_INFRA_NETWORK_ENV=internal`
- 环境变量 `NETWORK_ENV=internal`

**镜像策略**：
```
Harbor镜像: aiharbor.msxf.local/aihpc/redis:7-alpine
↓ 创建
redis:7-alpine
↓ 同时创建
localhost/redis:7-alpine
```

**适用场景**：
- 企业内网环境
- 无外网访问的服务器
- 使用内部 Harbor 仓库的部署环境
- 需要离线部署的场景

## 实际应用案例

### 案例 1：公网环境构建

**场景**：开发机器可以访问公网，直接使用 Docker Hub 镜像

```bash
# 1. 检测网络环境
./build.sh detect-network
# 输出: 当前网络环境: external

# 2. 处理所有依赖镜像
./build.sh tag-localhost

# 3. 结果
docker images | grep redis
# redis:7-alpine                  61.4MB  (原始镜像)
# localhost/redis:7-alpine        61.4MB  (兼容性别名)
```

### 案例 2：内网环境部署

**场景**：生产服务器无法访问外网，使用内部 Harbor 仓库

```bash
# 1. 在公网环境下，推送镜像到 Harbor
docker pull redis:7-alpine
docker tag redis:7-alpine aiharbor.msxf.local/aihpc/redis:7-alpine
docker push aiharbor.msxf.local/aihpc/redis:7-alpine

# 2. 在内网环境下，从 Harbor 拉取
docker pull aiharbor.msxf.local/aihpc/redis:7-alpine

# 3. 创建标准别名
./build.sh tag-localhost --network internal redis:7-alpine

# 4. 结果
docker images | grep redis
# aiharbor.msxf.local/aihpc/redis:7-alpine   61.4MB  (Harbor 镜像)
# redis:7-alpine                              61.4MB  (标准别名)
# localhost/redis:7-alpine                    61.4MB  (兼容性别名)
```

### 案例 3：混合环境（自动检测）

**场景**：笔记本在公司内网和家庭网络之间切换

```bash
# 无需手动指定网络环境，自动检测并应用最佳策略
./build.sh tag-localhost

# 内网环境：使用 Harbor 镜像
# 公网环境：使用 Docker Hub 镜像
```

## Docker Compose 集成

### 问题场景

`docker-compose.yml` 引用的镜像名称可能与本地镜像不匹配：

```yaml
services:
  redis:
    image: redis:7-alpine  # 标准名称
```

但本地只有：
- `localhost/redis:7-alpine` (内网拉取)
- `aiharbor.msxf.local/aihpc/redis:7-alpine` (Harbor 镜像)

### 解决方案

运行 `tag-localhost` 创建标准别名：

```bash
./build.sh tag-localhost redis:7-alpine
```

现在 docker-compose 可以正常启动：
```bash
docker-compose up -d
```

## 环境变量配置

### INTERNAL_REGISTRY

指定内网 Harbor 仓库地址

```bash
# .env 文件
INTERNAL_REGISTRY=aiharbor.msxf.local/aihpc

# 或者临时设置
export INTERNAL_REGISTRY=my-harbor.com/project
./build.sh tag-localhost --network internal
```

### AI_INFRA_NETWORK_ENV

强制指定网络环境

```bash
# 强制内网模式
export AI_INFRA_NETWORK_ENV=internal
./build.sh tag-localhost

# 强制公网模式
export AI_INFRA_NETWORK_ENV=external
./build.sh tag-localhost
```

## 最佳实践

### 1. 公网环境开发

```bash
# 直接使用 Docker Hub 镜像
docker pull redis:7-alpine

# 创建兼容性别名
./build.sh tag-localhost redis:7-alpine
```

### 2. 内网环境部署

```bash
# 方案 A：直接从 Harbor 拉取并创建别名
docker pull aiharbor.msxf.local/aihpc/redis:7-alpine
./build.sh tag-localhost --network internal redis:7-alpine

# 方案 B：批量处理所有依赖镜像
./build.sh tag-localhost --network internal
```

### 3. CI/CD 流程

```bash
#!/bin/bash
# deploy.sh

# 1. 检测网络环境
NETWORK_ENV=$(./build.sh detect-network | grep "当前网络环境" | awk '{print $3}')

# 2. 根据环境自动处理镜像
if [ "$NETWORK_ENV" = "internal" ]; then
    # 内网：从 Harbor 拉取
    docker pull aiharbor.msxf.local/aihpc/redis:7-alpine
    ./build.sh tag-localhost --network internal
else
    # 公网：直接拉取
    docker pull redis:7-alpine
    ./build.sh tag-localhost --network external
fi

# 3. 启动服务
docker-compose up -d
```

## 故障排查

### 问题 1：Harbor 镜像不存在

**错误信息**：
```
✗ Harbor 镜像不存在: aiharbor.msxf.local/aihpc/redis:7-alpine
💡 提示：请先从 Harbor 拉取镜像
   docker pull aiharbor.msxf.local/aihpc/redis:7-alpine
```

**解决方法**：
```bash
# 先从 Harbor 拉取镜像
docker pull aiharbor.msxf.local/aihpc/redis:7-alpine

# 再执行 tag 操作
./build.sh tag-localhost --network internal redis:7-alpine
```

### 问题 2：网络环境检测错误

**现象**：实际是公网环境，但检测为内网

**解决方法**：
```bash
# 手动指定网络环境
./build.sh tag-localhost --network external
```

### 问题 3：镜像名称不匹配

**现象**：docker-compose 提示 `image not found`

**解决方法**：
```bash
# 查看本地镜像
docker images | grep <image-name>

# 创建标准别名
./build.sh tag-localhost <image-name>
```

## 技术实现

### 核心函数

1. **tag_image_smart()** - 智能镜像 tag 函数
   - 自动检测网络环境
   - 根据环境选择策略
   - 创建必要的别名

2. **batch_tag_images_smart()** - 批量处理函数
   - 支持批量处理镜像列表
   - 统计成功/失败数量
   - 显示详细处理信息

3. **extract_base_images()** - 提取基础镜像函数
   - 从 Dockerfile 提取 FROM 指令
   - 过滤内部构建阶段
   - 去重并排序

### 镜像名称处理逻辑

```bash
# 移除 localhost/ 前缀
base_image="${image#localhost/}"

# 移除 Harbor 仓库前缀
base_image=$(echo "$base_image" | sed -E 's|^[^/]+\.[^/]+/[^/]+/||')
```

## 相关命令

- `./build.sh detect-network` - 检测网络环境
- `./build.sh build-all` - 构建所有服务（自动处理镜像）
- `./build.sh harbor-pull-deps` - 从 Harbor 拉取依赖镜像

## 更新日志

### v0.3.7 (2025-10-11)
- ✅ 新增智能镜像 tag 管理功能
- ✅ 支持公网/内网环境自动检测
- ✅ 集成 Harbor 仓库支持
- ✅ 自动从 Dockerfile 提取基础镜像
- ✅ 移除硬编码的镜像列表

### v0.3.6
- 🔧 使用硬编码镜像列表
- 🔧 仅支持 localhost/ 前缀双向 tag

## 参考文档

- [Docker 镜像管理最佳实践](https://docs.docker.com/develop/dev-best-practices/)
- [Harbor 用户指南](https://goharbor.io/docs/)
- [网络环境检测实现](./NETWORK_DETECTION.md)
