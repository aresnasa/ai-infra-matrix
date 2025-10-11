# 智能镜像 Tag 管理 - 快速开始

> 5分钟快速上手智能镜像 tag 管理功能

## 🚀 快速开始

### 方式 1：使用 build-all（推荐）

最简单的方式，一键完成所有操作：

```bash
# 自动检测环境，构建所有服务
./build.sh build-all
```

**自动执行**：
1. ✅ 检测网络环境（公网/内网）
2. ✅ 预拉取依赖镜像
3. ✅ 智能创建镜像别名
4. ✅ 同步配置文件
5. ✅ 构建所有服务
6. ✅ 验证构建结果

### 方式 2：独立使用 tag-localhost

只处理镜像别名，不构建：

```bash
# 自动处理所有 Dockerfile 中的基础镜像
./build.sh tag-localhost

# 处理单个镜像
./build.sh tag-localhost redis:7-alpine

# 处理多个镜像
./build.sh tag-localhost redis:7-alpine nginx:stable
```

## 🌍 常见场景

### 场景 1：开发环境（有外网）

```bash
# 直接运行，自动检测为公网环境
./build.sh build-all

# 效果：
# ✓ 从 Docker Hub 拉取镜像
# ✓ 创建 localhost/ 别名
# ✓ 构建所有服务
```

### 场景 2：生产环境（内网部署）

```bash
# 设置内网环境
export AI_INFRA_NETWORK_ENV=internal
export INTERNAL_REGISTRY=aiharbor.msxf.local/aihpc

# 运行构建
./build.sh build-all

# 效果：
# ✓ 从 Harbor 拉取镜像
# ✓ 创建原始名称别名
# ✓ 创建 localhost/ 别名
# ✓ 构建所有服务
```

### 场景 3：镜像不一致问题

**问题**：docker-compose 提示 `image not found`

```bash
# 检查本地镜像
docker images | grep redis
# 输出：localhost/redis:7-alpine

# 但 docker-compose.yml 需要：redis:7-alpine
```

**解决**：

```bash
# 快速创建别名
./build.sh tag-localhost redis:7-alpine

# 或者重新构建
./build.sh build-all
```

## 📝 环境变量

### INTERNAL_REGISTRY

指定内网 Harbor 仓库地址

```bash
# 默认值
INTERNAL_REGISTRY=aiharbor.msxf.local/aihpc

# 自定义
export INTERNAL_REGISTRY=my-harbor.com/project
```

### AI_INFRA_NETWORK_ENV

强制指定网络环境

```bash
# 强制内网模式
export AI_INFRA_NETWORK_ENV=internal

# 强制公网模式
export AI_INFRA_NETWORK_ENV=external
```

## 💡 实用技巧

### 技巧 1：检测网络环境

```bash
# 查看当前网络环境
./build.sh detect-network

# 输出示例：
# [INFO] 当前网络环境: external
# [SUCCESS] ✓ 检测到外网环境，可以正常访问外部服务
```

### 技巧 2：查看帮助

```bash
# 查看 build-all 帮助
./build.sh build-all --help

# 查看 tag-localhost 帮助
./build.sh tag-localhost --help
```

### 技巧 3：强制重建

```bash
# 强制重建所有服务
./build.sh build-all --force

# 强制重建并使用内网环境
AI_INFRA_NETWORK_ENV=internal ./build.sh build-all --force
```

## 🔍 验证结果

### 验证镜像别名

```bash
# 查看 redis 镜像
docker images | grep redis

# 期望输出（公网环境）：
# redis:7-alpine           61.4MB
# localhost/redis:7-alpine 61.4MB

# 期望输出（内网环境）：
# aiharbor.msxf.local/aihpc/redis:7-alpine  61.4MB
# redis:7-alpine                             61.4MB
# localhost/redis:7-alpine                   61.4MB
```

### 验证构建状态

```bash
# 查看所有服务的构建状态
./build.sh check-status

# 输出示例：
# [SUCCESS] ✓ 构建成功的服务 (11):
#   • backend
#   • frontend
#   • jupyterhub
#   ...
```

## ❓ 常见问题

### Q1: 网络环境检测不准确怎么办？

**A**: 手动指定环境

```bash
# 强制公网模式
export AI_INFRA_NETWORK_ENV=external
./build.sh build-all

# 强制内网模式
export AI_INFRA_NETWORK_ENV=internal
./build.sh build-all
```

### Q2: Harbor 镜像不存在怎么办？

**A**: 先手动拉取

```bash
# 1. 登录 Harbor
docker login aiharbor.msxf.local

# 2. 拉取镜像
docker pull aiharbor.msxf.local/aihpc/redis:7-alpine

# 3. 重新构建
./build.sh build-all
```

### Q3: 镜像别名创建失败怎么办？

**A**: 检查源镜像

```bash
# 1. 检查镜像是否存在
docker images | grep redis

# 2. 如果不存在，先拉取
docker pull redis:7-alpine

# 3. 重新创建别名
./build.sh tag-localhost redis:7-alpine
```

## 📚 详细文档

- [智能镜像 Tag 管理指南](./IMAGE_TAG_SMART_GUIDE.md) - 完整使用文档
- [Build-All 集成说明](./BUILD_ALL_SMART_TAG_INTEGRATION.md) - 集成详情
- [优化总结报告](./COMPLETE_OPTIMIZATION_REPORT.md) - 技术总览

## ⚡ 一分钟总结

```bash
# 开发环境（公网）
./build.sh build-all

# 生产环境（内网）
AI_INFRA_NETWORK_ENV=internal \
INTERNAL_REGISTRY=aiharbor.msxf.local/aihpc \
./build.sh build-all

# 只处理镜像别名
./build.sh tag-localhost

# 检测网络环境
./build.sh detect-network
```

---

**就这么简单！** 🎉

更多高级用法请查看 [完整文档](./IMAGE_TAG_SMART_GUIDE.md)
