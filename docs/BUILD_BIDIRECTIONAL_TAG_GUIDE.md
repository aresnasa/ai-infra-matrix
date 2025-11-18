# 双向镜像Tag系统说明

## 概述

本项目实现了智能的双向镜像tag系统，可以自动适配公网和内网环境，确保镜像在不同环境下都能正确引用。

## 问题背景

Docker Compose 在某些环境下（特别是使用 Podman 或特定配置的 Docker）会自动为拉取的镜像添加 `localhost/` 前缀，导致：

```bash
# docker-compose.yml 中定义
image: redis:7-alpine

# 但实际拉取后变成
localhost/redis:7-alpine
```

这会导致后续使用标准名称（如 `redis:7-alpine`）无法找到镜像。

## 解决方案

### 双向Tag机制

智能tag系统会根据网络环境创建合适的镜像别名：

#### 公网环境 (external)
```bash
# 对于 redis:7-alpine
✓ redis:7-alpine           # 标准名称
✓ localhost/redis:7-alpine  # localhost 别名（兼容性）

# 对于 osixia/openldap:stable
✓ osixia/openldap:stable        # 完整命名空间
✓ openldap:stable               # 短名称
✓ localhost/openldap:stable     # localhost + 短名称
```

#### 内网环境 (internal)
```bash
# 对于 redis:7-alpine
✓ aiharbor.msxf.local/aihpc/redis:7-alpine  # Harbor 源
✓ redis:7-alpine                             # 标准别名
✓ localhost/redis:7-alpine                   # localhost 别名

# 对于 osixia/openldap:stable
✓ aiharbor.msxf.local/aihpc/osixia/openldap:stable  # Harbor 源
✓ osixia/openldap:stable                             # 完整命名空间
✓ openldap:stable                                    # 短名称
✓ localhost/openldap:stable                          # localhost + 短名称
```

## 使用方法

### 1. 自动tag所有依赖镜像

```bash
# 自动检测网络环境并处理所有 Dockerfile 中的基础镜像
./build.sh tag-localhost

# 显示详细帮助
./build.sh tag-localhost --help
```

### 2. 手动指定单个镜像

```bash
# 处理单个镜像
./build.sh tag-localhost redis:7-alpine

# 处理多个镜像
./build.sh tag-localhost redis:7-alpine postgres:16-alpine
```

### 3. 强制指定网络环境

```bash
# 强制公网模式
./build.sh tag-localhost --network external redis:7-alpine

# 强制内网模式
./build.sh tag-localhost --network internal

# 指定自定义 Harbor 仓库
./build.sh tag-localhost --network internal --harbor my-harbor.com/repo
```

### 4. 集成到build-all

build-all 命令会自动调用 tag-localhost，确保所有依赖镜像都有正确的tag：

```bash
./build.sh build-all
```

## 工作流程

### 1. 网络环境自动检测

```bash
detect_network_environment() {
    # 检测内网 Harbor 仓库是否可达
    if ping -c 1 -W 1 aiharbor.msxf.local >/dev/null 2>&1; then
        echo "internal"
    else
        echo "external"
    fi
}
```

### 2. 智能Tag策略

**公网环境**:
- 优先使用原始镜像名称（如 `redis:7-alpine`）
- 创建 `localhost/` 前缀别名（兼容性）
- 对于命名空间镜像，创建短名称别名

**内网环境**:
- 优先从 Harbor 仓库拉取
- 创建原始名称别名
- 创建 `localhost/` 前缀别名
- 对于命名空间镜像，创建短名称别名

### 3. 降级处理

如果优先源不可用，系统会自动降级：

```
Harbor镜像 → 完整命名空间 → 短名称 → localhost/ 版本
```

## 相关函数

### 主要函数

| 函数名 | 说明 |
|--------|------|
| `tag_image_smart` | 智能tag单个镜像 |
| `batch_tag_images_smart` | 批量智能tag镜像列表 |
| `detect_network_environment` | 自动检测网络环境 |
| `extract_base_images` | 从 Dockerfile 提取基础镜像 |

### 兼容性函数

| 函数名 | 说明 |
|--------|------|
| `tag_image_bidirectional` | 旧版双向tag接口 |
| `batch_tag_images_bidirectional` | 旧版批量tag接口 |

## 故障排查

### 问题1: 镜像只有 localhost/ 前缀

```bash
# 症状
$ docker images | grep redis
localhost/redis    7-alpine    xxx

# 解决
$ ./build.sh tag-localhost redis:7-alpine

# 结果
redis              7-alpine    xxx
localhost/redis    7-alpine    xxx
```

### 问题2: Docker Compose 找不到镜像

```bash
# 错误信息
Error: image redis:7-alpine not found

# 检查
$ docker images | grep redis

# 如果只有 localhost/redis，运行
$ ./build.sh tag-localhost redis:7-alpine
```

### 问题3: 内网环境无法拉取

```bash
# 症状
Failed to pull aiharbor.msxf.local/aihpc/redis:7-alpine

# 检查网络
$ ping aiharbor.msxf.local

# 如果不通，使用公网模式
$ ./build.sh tag-localhost --network external redis:7-alpine
```

## 环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `INTERNAL_REGISTRY` | `aiharbor.msxf.local/aihpc` | 内网 Harbor 仓库地址 |

可在 `.env` 文件中自定义：

```bash
# .env
INTERNAL_REGISTRY=my-harbor.com/my-project
```

## 最佳实践

### 1. 开发环境初始化

```bash
# 克隆项目后首次运行
git clone ...
cd ai-infra-matrix
./build.sh tag-localhost  # 创建所有必要的tag
./build.sh build-all      # 构建所有服务
```

### 2. 切换网络环境

```bash
# 从公网切换到内网
./build.sh tag-localhost --network internal

# 从内网切换到公网
./build.sh tag-localhost --network external
```

### 3. 更新依赖镜像

```bash
# 拉取最新镜像并重新tag
docker pull redis:7-alpine
./build.sh tag-localhost redis:7-alpine
```

## 实现细节

### 镜像名称解析

系统会智能识别不同类型的镜像前缀：

```bash
# 标准 Docker Hub 镜像
redis:7-alpine
→ base_image: redis:7-alpine
→ short_name: redis:7-alpine

# 带命名空间的镜像
osixia/openldap:stable
→ base_image: osixia/openldap:stable
→ short_name: openldap:stable

# Harbor 私有仓库镜像
aiharbor.msxf.local/aihpc/redis:7-alpine
→ base_image: redis:7-alpine
→ short_name: redis:7-alpine

# localhost 前缀镜像
localhost/redis:7-alpine
→ base_image: redis:7-alpine
→ short_name: redis:7-alpine
```

### Tag 创建逻辑

```bash
# 公网环境
if 镜像存在（任意格式）; then
    source_image = 找到的第一个镜像
    创建 base_image 别名
    创建 short_name 别名（如果不同）
    创建 localhost/short_name 别名
fi

# 内网环境
if Harbor镜像存在; then
    source_image = Harbor镜像
    创建 base_image 别名
    创建 short_name 别名
    创建 localhost/short_name 别名
else if 本地镜像存在（任意格式）; then
    source_image = 本地镜像
    创建所有别名
fi
```

## 相关文档

- [构建系统文档](BUILD_USAGE_GUIDE.md)
- [网络配置文档](CROSS_PLATFORM_COMPATIBILITY.md)
- [Docker Compose 模板](DOCKER_COMPOSE_TEMPLATE_VERIFICATION.md)

## 版本历史

- v0.3.7: 引入智能双向tag系统
- v0.3.6: 初始tag功能
