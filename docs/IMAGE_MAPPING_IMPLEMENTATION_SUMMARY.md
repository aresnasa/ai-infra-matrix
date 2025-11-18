# 镜像版本映射机制实现总结

## 概述

实现了一个智能的镜像版本映射机制，解决了以下需求：
- **开发/CI/CD环境**: 可以使用 `latest` 标签进行开发
- **内部registry部署**: 自动将 `latest` 映射到 git 版本 `v0.3.5` 进行统一版本管理
- **项目镜像**: 保持使用传入的标签（如 `v0.3.5`）
- **基础镜像**: 自动映射到内部registry的对应版本

## 关键文件

### 1. 映射配置文件: `config/image-mapping.conf`
```bash
# 基础镜像映射 (latest -> v0.3.5)
postgres:15-alpine|library|v0.3.5
redis:7-alpine|library|v0.3.5
nginx:1.27-alpine|library|v0.3.5

# 第三方镜像映射 (latest -> v0.3.5)
tecnativa/tcp-proxy:latest|tecnativa|v0.3.5
redislabs/redisinsight:latest|redislabs|v0.3.5
quay.io/minio/minio:latest|minio|v0.3.5
```

### 2. 核心映射函数: `build.sh`中的 `get_mapped_private_image()`
- 读取映射配置文件
- 支持模式匹配（精确匹配、基础名匹配）
- 处理Harbor项目结构
- 将latest标签映射到v0.3.5

### 3. 生产环境配置生成: `generate_production_config()`
- 使用映射机制替换基础镜像和第三方镜像
- 项目镜像保持原有逻辑，使用传入的标签
- 支持macOS和Linux环境

## 映射效果示例

### 原始镜像 → 内部registry映射
```bash
# 基础镜像
postgres:15-alpine → aiharbor.msxf.local/library/postgres:v0.3.5
redis:7-alpine → aiharbor.msxf.local/library/redis:v0.3.5
nginx:1.27-alpine → aiharbor.msxf.local/library/nginx:v0.3.5

# 第三方镜像
tecnativa/tcp-proxy:latest → aiharbor.msxf.local/tecnativa/tcp-proxy:v0.3.5
redislabs/redisinsight:latest → aiharbor.msxf.local/redislabs/redisinsight:v0.3.5
quay.io/minio/minio:latest → aiharbor.msxf.local/minio/minio:v0.3.5

# 项目镜像
ghcr.io/aresnasa/ai-infra-matrix/ai-infra-backend:v0.3.5 → aiharbor.msxf.local/aihpc/ai-infra-matrix/ai-infra-backend:v0.3.5
```

## 已更新的功能

### 1. 依赖镜像管理
- ✅ `pull_and_tag_dependencies()`: 使用新映射机制拉取和标记依赖镜像
- ✅ `push_dependencies()`: 使用新映射机制推送依赖镜像
- ✅ 支持Harbor项目结构命名空间

### 2. 生产环境部署
- ✅ `generate_production_config()`: 使用映射配置生成生产环境配置
- ✅ `start_production()`: 总是重新生成配置以确保正确的registry和版本
- ✅ 移除LDAP服务，支持独立启动

### 3. 项目镜像管理
- ✅ `push_service()`: 正确处理项目镜像的registry映射
- ✅ `build_all_services()`: 构建时使用正确的镜像命名
- ✅ `push_all_services()`: 推送时使用正确的镜像命名

### 4. 命令行接口
- ✅ 所有相关命令的错误信息包含用法说明
- ✅ 支持所有现有的命令格式
- ✅ 向后兼容性保持

## 测试验证

### 自动化测试脚本: `scripts/test-mapping-functions.sh`
```bash
./scripts/test-mapping-functions.sh
```

测试覆盖：
- ✅ 生产环境配置生成 (5项测试)
- ✅ 映射配置文件 (4项测试)
- ✅ 映射函数 (3项测试)
- ✅ 依赖镜像命令 (2项测试)
- ✅ 版本管理 (2项测试)

**总计: 16项测试全部通过**

## 使用示例

### 1. 生成生产环境配置
```bash
./build.sh prod-generate aiharbor.msxf.local/aihpc v0.3.5
```

### 2. 启动生产环境
```bash
./build.sh prod-up aiharbor.msxf.local/aihpc v0.3.5
```

### 3. 处理依赖镜像
```bash
# 拉取并标记依赖镜像
./build.sh deps-pull aiharbor.msxf.local/aihpc latest

# 推送依赖镜像
./build.sh deps-push aiharbor.msxf.local/aihpc latest

# 一键处理所有依赖镜像
./build.sh deps-all aiharbor.msxf.local/aihpc latest
```

### 4. 构建和推送项目镜像
```bash
./build.sh build-push aiharbor.msxf.local/aihpc v0.3.5
```

## 关键优势

1. **版本管理统一**: 所有镜像在内部registry中使用统一的git版本标签
2. **开发友好**: 开发和CI/CD过程中仍可使用latest标签
3. **配置驱动**: 通过配置文件轻松管理映射关系
4. **Harbor兼容**: 支持Harbor的项目结构和命名空间
5. **向后兼容**: 不影响现有的构建和部署流程
6. **全面测试**: 包含自动化测试验证所有功能

## 文档链接

- [Harbor项目设置指南](./HARBOR_PROJECT_SETUP.md)
- [私有仓库迁移指南](./PRIVATE_REGISTRY_MIGRATION_GUIDE.md)
- [构建脚本使用指南](./BUILD_USAGE_GUIDE.md)
