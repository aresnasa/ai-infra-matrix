# 🚨 Harbor部署错误修复指南

## 问题诊断

您遇到的错误：
```
Error response from daemon: unknown: artifact library/nginx:v0.3.5 not found
```

**根本原因**: Harbor仓库中缺少映射的依赖镜像。

## 🛠️ 快速修复方案

### 方案1: 使用自动修复脚本（推荐）

```bash
# 1. 先登录Harbor
docker login aiharbor.msxf.local

# 2. 运行自动修复脚本
./scripts/fix-harbor-deployment.sh

# 3. 如果需要自定义参数
./scripts/fix-harbor-deployment.sh --registry aiharbor.msxf.local/aihpc --tag v0.3.5
```

### 方案2: 手动分步修复

```bash
# 1. 登录Harbor
docker login aiharbor.msxf.local

# 2. 推送所有依赖镜像
./build.sh deps-all aiharbor.msxf.local/aihpc v0.3.5

# 3. 构建并推送项目镜像
./build.sh build-push aiharbor.msxf.local/aihpc v0.3.5

# 4. 启动生产环境
./build.sh prod-up aiharbor.msxf.local/aihpc v0.3.5
```

## 📋 检查清单

在修复前，请确认：

- [ ] Harbor登录状态正常
- [ ] 对 `aiharbor.msxf.local/aihpc` 项目有推送权限
- [ ] 网络可以访问Harbor仓库
- [ ] 本地Docker环境正常

## 🔍 故障排除

### 1. 检查Harbor登录
```bash
docker login aiharbor.msxf.local
```

### 2. 测试Harbor连接
```bash
# 尝试拉取一个测试镜像
docker pull aiharbor.msxf.local/library/hello-world:latest || echo "Harbor连接失败"
```

### 3. 检查推送权限
```bash
# 尝试推送一个测试镜像
docker tag hello-world:latest aiharbor.msxf.local/aihpc/test:latest
docker push aiharbor.msxf.local/aihpc/test:latest
docker rmi aiharbor.msxf.local/aihpc/test:latest
```

### 4. 查看详细错误
```bash
# 只检查状态，不执行修复
./scripts/fix-harbor-deployment.sh --check-only
```

## 📊 依赖镜像列表

需要推送到Harbor的依赖镜像：

| 原始镜像 | Harbor映射 |
|---------|-----------|
| `postgres:15-alpine` | `aiharbor.msxf.local/library/postgres:v0.3.5` |
| `redis:7-alpine` | `aiharbor.msxf.local/library/redis:v0.3.5` |
| `nginx:1.27-alpine` | `aiharbor.msxf.local/library/nginx:v0.3.5` |
| `minio/minio:latest` | `aiharbor.msxf.local/minio/minio:v0.3.5` |
| `tecnativa/tcp-proxy:latest` | `aiharbor.msxf.local/tecnativa/tcp-proxy:v0.3.5` |
| `redislabs/redisinsight:latest` | `aiharbor.msxf.local/redislabs/redisinsight:v0.3.5` |

## ⚡ 快速命令

```bash
# 完整修复（推荐）
./scripts/fix-harbor-deployment.sh

# 只推送依赖镜像
./build.sh deps-all aiharbor.msxf.local/aihpc v0.3.5

# 只构建项目镜像
./build.sh build-push aiharbor.msxf.local/aihpc v0.3.5

# 检查Harbor中的镜像状态
./scripts/fix-harbor-deployment.sh --check-only

# 查看当前Docker Compose状态
docker-compose -f docker-compose.prod.yml ps
```

## 🎯 预期结果

修复成功后，您应该看到：

```
✅ 依赖镜像推送完成
✅ 项目镜像构建推送完成
✅ 生产环境启动成功

查看服务状态:
NAME                    IMAGE                                           STATUS
ai-infra-backend        aiharbor.msxf.local/aihpc/ai-infra-backend:v0.3.5   Up
ai-infra-frontend       aiharbor.msxf.local/aihpc/ai-infra-frontend:v0.3.5  Up
ai-infra-nginx          aiharbor.msxf.local/aihpc/ai-infra-nginx:v0.3.5     Up
...
```

## 📞 需要帮助？

如果修复脚本仍然失败，请运行：

```bash
./scripts/fix-harbor-deployment.sh --check-only
```

并提供输出信息以进行进一步诊断。
