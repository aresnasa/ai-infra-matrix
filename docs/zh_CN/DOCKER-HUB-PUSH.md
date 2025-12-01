# 依赖镜像推送功能 - Docker Hub 集成

**中文** | **[English](en/DOCKER-HUB-PUSH.md)**

## 概述

AI-Infra-Matrix 现在支持自动推送所有依赖镜像到 Docker Hub，解决国内网络环境下拉取镜像困难的问题。

## 功能特性

### 🚀 自动依赖发现
- 自动扫描 `docker-compose.yml` 文件
- 识别所有第三方依赖镜像（排除 ai-infra-* 自建镜像）
- 支持多个 compose 文件（根目录 + 生产环境目录）

### 📦 智能镜像推送
- 自动重新标记镜像到指定命名空间
- 支持自定义 Docker Hub 命名空间
- 自动生成 latest 标签
- 跳过已存在镜像选项

### 🔧 灵活配置
- 支持自定义命名空间
- 可跳过已存在的镜像
- 详细的推送进度和结果报告

## 使用方法

### 基础用法

```bash
# 推送所有依赖镜像到默认命名空间 (aresnasa)
./scripts/build.sh prod --push-deps
```

### 自定义命名空间

```bash
# 推送到自定义 Docker Hub 命名空间
./scripts/build.sh prod --push-deps --deps-namespace myusername
```

### 跳过已存在镜像

```bash
# 跳过已推送的镜像，只推送新的或更新的
./scripts/build.sh prod --push-deps --skip-existing-deps
```

### 组合使用

```bash
# 完整的构建和推送流程
./scripts/build.sh prod --version v1.0.0 --push-deps --deps-namespace mycompany --skip-existing-deps
```

## 前置要求

### Docker Hub 登录
```bash
# 确保已登录 Docker Hub
docker login
```

### 权限要求
- Docker Hub 账号
- 对目标命名空间的推送权限
- 足够的存储配额

## 推送的镜像命名规则

原始镜像会被重新标记为：
```
docker.io/[命名空间]/ai-infra-dep-[镜像名]:[标签]
```

### 示例

| 原始镜像 | 推送后镜像 |
|---------|-----------|
| `postgres:13` | `docker.io/aresnasa/ai-infra-dep-postgres:13` |
| `redis:7-alpine` | `docker.io/aresnasa/ai-infra-dep-redis:7-alpine` |
| `nginx:latest` | `docker.io/aresnasa/ai-infra-dep-nginx:latest` |

## 使用推送的镜像

### 修改 docker-compose.yml

```yaml
# 原始配置
services:
  postgres:
    image: postgres:13
  redis:
    image: redis:7-alpine

# 使用推送的镜像
services:
  postgres:
    image: docker.io/aresnasa/ai-infra-dep-postgres:13
  redis:
    image: docker.io/aresnasa/ai-infra-dep-redis:7-alpine
```

### 使用环境变量切换

```bash
# 设置环境变量使用推送的镜像
export REGISTRY_PREFIX="docker.io/aresnasa/ai-infra-dep-"

# 在 compose 文件中使用
services:
  postgres:
    image: ${REGISTRY_PREFIX:-}postgres:13
```

## 推送状态报告

推送完成后会显示详细报告：

```
🎉 依赖镜像推送完成！
================================
✅ 成功推送: 8 个镜像
⚠️  跳过镜像: 2 个镜像  
❌ 推送失败: 0 个镜像

推送的镜像可通过以下方式访问:
  docker pull docker.io/aresnasa/ai-infra-dep-<镜像名>:latest

示例镜像列表:
  docker pull docker.io/aresnasa/ai-infra-dep-postgres:latest
  docker pull docker.io/aresnasa/ai-infra-dep-redis:latest
  docker pull docker.io/aresnasa/ai-infra-dep-nginx:latest
  ... 还有 5 个镜像
```

## 故障排除

### 常见问题

1. **未登录 Docker Hub**
   ```bash
   docker login
   ```

2. **权限不足**
   - 确保对目标命名空间有推送权限
   - 检查 Docker Hub 配额

3. **网络超时**
   ```bash
   # 重试推送，跳过已成功的镜像
   ./scripts/build.sh prod --push-deps --skip-existing-deps
   ```

4. **镜像不存在**
   ```bash
   # 先拉取基础镜像
   ./scripts/build.sh prod --update-images --push-deps
   ```

### 调试信息

推送过程中会显示详细的调试信息：
- 镜像发现过程
- 重新标记步骤  
- 推送进度
- 错误详情

## 高级用法

### 批量推送到多个注册表

```bash
# 推送到多个命名空间
./scripts/build.sh prod --push-deps --deps-namespace company1
./scripts/build.sh prod --push-deps --deps-namespace company2 --skip-existing-deps
```

### 集成到 CI/CD

```yaml
# GitHub Actions 示例
- name: Push dependency images
  run: |
    echo "${{ secrets.DOCKER_PASSWORD }}" | docker login -u "${{ secrets.DOCKER_USERNAME }}" --password-stdin
    ./scripts/build.sh prod --push-deps --deps-namespace ${{ secrets.DOCKER_NAMESPACE }}
```

## 注意事项

### 许可证合规
- 确保推送的镜像符合原始许可证要求
- 仅用于内部或授权用途

### 存储成本
- Docker Hub 免费账户有存储限制
- 考虑使用私有注册表用于大量镜像

### 安全考虑
- 不要推送包含敏感信息的镜像
- 定期清理旧版本镜像

## 相关命令

```bash
# 查看帮助
./scripts/build.sh --help

# 测试功能
./scripts/test-push-deps.sh

# 查看依赖镜像列表（不推送）
grep -E '^[[:space:]]*image:' docker-compose.yml
```
