# AppHub 组件选择性构建指南

## 概述

AppHub Dockerfile 现在支持通过构建参数（Build Args）选择性构建特定组件，这样可以：
- 加快构建速度
- 减小镜像体积
- 按需构建所需组件

## 构建参数

### 可用的构建开关

| 参数名 | 默认值 | 说明 |
|--------|--------|------|
| `BUILD_SLURM` | `true` | 是否构建 SLURM 包（DEB/RPM） |
| `BUILD_SALTSTACK` | `true` | 是否从 GitHub 下载 SaltStack 包 |
| `BUILD_CATEGRAF` | `true` | 是否构建 Categraf 监控组件 |
| `BUILD_SINGULARITY` | `false` | 是否构建 Singularity 容器运行时（暂未实现） |

## 使用方法

### 方法1：使用 build.sh 脚本

```bash
# 完整构建（所有组件）
./build.sh build apphub --force

# 只构建 SLURM
./build.sh build apphub --force \
    --build-arg BUILD_SLURM=true \
    --build-arg BUILD_SALTSTACK=false \
    --build-arg BUILD_CATEGRAF=false

# 只构建 SaltStack 和 Categraf
./build.sh build apphub --force \
    --build-arg BUILD_SLURM=false \
    --build-arg BUILD_SALTSTACK=true \
    --build-arg BUILD_CATEGRAF=true

# 最小构建（不构建任何应用，只有基础镜像）
./build.sh build apphub --force \
    --build-arg BUILD_SLURM=false \
    --build-arg BUILD_SALTSTACK=false \
    --build-arg BUILD_CATEGRAF=false
```

### 方法2：直接使用 Docker 命令

```bash
# 构建 AppHub，只包含 SaltStack
docker build \
    --build-arg BUILD_SLURM=false \
    --build-arg BUILD_SALTSTACK=true \
    --build-arg BUILD_CATEGRAF=false \
    -t ai-infra-apphub:saltstack-only \
    -f src/apphub/Dockerfile \
    src/apphub

# 构建 AppHub，只包含 Categraf
docker build \
    --build-arg BUILD_SLURM=false \
    --build-arg BUILD_SALTSTACK=false \
    --build-arg BUILD_CATEGRAF=true \
    -t ai-infra-apphub:categraf-only \
    -f src/apphub/Dockerfile \
    src/apphub
```

### 方法3：使用 docker-compose

在 `docker-compose.yml` 中添加 build args：

```yaml
services:
  apphub:
    build:
      context: ./src/apphub
      dockerfile: Dockerfile
      args:
        BUILD_SLURM: "true"
        BUILD_SALTSTACK: "true"
        BUILD_CATEGRAF: "true"
        BUILD_SINGULARITY: "false"
```

然后构建：

```bash
docker-compose build apphub
```

## 构建时间对比

| 构建配置 | 预计构建时间 | 镜像大小 |
|---------|-------------|----------|
| 全部组件 | ~10-15分钟 | ~800MB |
| 仅 SLURM | ~8-12分钟 | ~500MB |
| 仅 SaltStack | ~3-5分钟 | ~350MB |
| 仅 Categraf | ~2-3分钟 | ~150MB |
| 无应用 | ~1-2分钟 | ~100MB |

## 验证构建结果

### 检查包目录

```bash
# 启动容器
docker run --rm -it ai-infra-apphub:latest /bin/sh

# 检查 SLURM DEB 包
ls -lh /usr/share/nginx/html/pkgs/slurm-deb/

# 检查 SaltStack 包
ls -lh /usr/share/nginx/html/pkgs/saltstack-deb/
ls -lh /usr/share/nginx/html/pkgs/saltstack-rpm/

# 检查 Categraf 包
ls -lh /usr/share/nginx/html/pkgs/categraf/
```

### 查看包统计

容器启动时会输出包统计信息：

```
📊 Package Summary:
  - SLURM deb packages: 17
  - SLURM rpm packages: 6
  - SLURM apk packages: 0
  - SaltStack deb packages: 7
  - SaltStack rpm packages: 7
  - Categraf packages: 2
```

## 版本管理

所有组件版本都在 Dockerfile 顶部定义为 ARG：

```dockerfile
ARG SLURM_VERSION=25.05.4
ARG SALTSTACK_VERSION=v3007.8
ARG CATEGRAF_VERSION=v0.4.22
ARG SINGULARITY_VERSION=v4.3.4
```

可以在构建时覆盖：

```bash
docker build \
    --build-arg SALTSTACK_VERSION=v3007.9 \
    --build-arg BUILD_SALTSTACK=true \
    -t ai-infra-apphub:custom \
    -f src/apphub/Dockerfile \
    src/apphub
```

## 故障排除

### 构建失败

1. 检查构建日志中的错误信息
2. 验证版本号是否正确
3. 确认网络连接正常（GitHub releases 下载）

### 包缺失

如果某个组件的包数量为 0：

1. 检查对应的 `BUILD_*` 参数是否设置为 `true`
2. 查看构建日志中的下载错误
3. 验证版本号在 GitHub releases 中存在

### 镜像过大

如果只需要特定组件，记得关闭其他组件：

```bash
# 只要 Categraf
docker build \
    --build-arg BUILD_SLURM=false \
    --build-arg BUILD_SALTSTACK=false \
    --build-arg BUILD_CATEGRAF=true \
    -t ai-infra-apphub:categraf \
    -f src/apphub/Dockerfile \
    src/apphub
```

## 最佳实践

1. **CI/CD 环境**：根据需要构建不同的变体镜像
2. **开发环境**：使用最小构建加快迭代速度
3. **生产环境**：构建包含所有组件的完整镜像
4. **测试环境**：只构建需要测试的组件

## 示例：多阶段部署

```bash
# 步骤1：快速构建最小镜像进行测试
./build.sh build apphub \
    --build-arg BUILD_SLURM=false \
    --build-arg BUILD_SALTSTACK=false \
    --build-arg BUILD_CATEGRAF=true

# 步骤2：验证 Categraf 功能
docker run -p 8080:80 ai-infra-apphub:latest

# 步骤3：添加其他组件重新构建
./build.sh build apphub --force
```

## 自动化脚本

创建一个便捷脚本 `build-apphub-variants.sh`：

```bash
#!/bin/bash

# 构建所有变体
./build.sh build apphub --tag apphub:full --force

./build.sh build apphub --tag apphub:slurm-only \
    --build-arg BUILD_SLURM=true \
    --build-arg BUILD_SALTSTACK=false \
    --build-arg BUILD_CATEGRAF=false

./build.sh build apphub --tag apphub:saltstack-only \
    --build-arg BUILD_SLURM=false \
    --build-arg BUILD_SALTSTACK=true \
    --build-arg BUILD_CATEGRAF=false

./build.sh build apphub --tag apphub:categraf-only \
    --build-arg BUILD_SLURM=false \
    --build-arg BUILD_SALTSTACK=false \
    --build-arg BUILD_CATEGRAF=true
```

## 相关文档

- [AppHub 使用指南](./APPHUB_USAGE_GUIDE.md)
- [构建脚本使用说明](./BUILD_USAGE_GUIDE.md)
- [版本管理文档](./APPHUB_VERSION_MANAGEMENT.md)
