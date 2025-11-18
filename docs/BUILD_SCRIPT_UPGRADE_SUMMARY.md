# Build.sh 脚本升级总结

## 升级概述

AI Infrastructure Matrix 的 `build.sh` 脚本已成功升级，新增了以下主要功能：

### 🚀 新增功能

#### 1. Harbor 格式支持
- **完全支持 Harbor 仓库格式**: `registry.example.com/project`
- **自动识别仓库类型**: 传统格式 vs Harbor 格式
- **统一的镜像命名**: 使用 `get_private_image_name()` 函数处理

#### 2. 依赖镜像管理
- **拉取并标记依赖镜像**: `deps-pull <registry> [tag]`
- **推送依赖镜像**: `deps-push <registry> [tag]` 
- **一键依赖镜像操作**: `deps-all <registry> [tag]`
- **支持的依赖镜像**:
  - postgres:15-alpine
  - redis:7-alpine
  - osixia/openldap:stable
  - osixia/phpldapadmin:stable
  - tecnativa/tcp-proxy
  - redislabs/redisinsight:latest
  - nginx:1.27-alpine
  - quay.io/minio/minio:latest

#### 3. Mock 测试环境
- **简化的 Mock 环境**: 仅用于脚本功能验证
- **智能服务检测**: 自动检测是否存在 backend 镜像
- **健康检查**: PostgreSQL 和 Redis 服务健康检查
- **连接测试**: `mock-test` 命令验证服务连通性
- **灵活的启动模式**:
  - 基础模式: 仅启动 PostgreSQL 和 Redis
  - 完整模式: 包含 backend 服务（当镜像存在时）

### 🔧 核心改进

#### 1. Harbor 格式镜像名生成
```bash
# Harbor 格式示例
registry.example.com/ai-infra/ai-infra-backend:v0.3.5
registry.example.com/ai-infra/ai-infra-deps-postgres:v0.3.5

# 传统格式示例  
registry.example.com/ai-infra-backend:v0.3.5
registry.example.com/ai-infra-deps-postgres:v0.3.5
```

#### 2. 统一的镜像处理逻辑
- 所有镜像操作都使用 `get_private_image_name()` 函数
- 支持自动检测和处理不同格式的镜像名
- 确保依赖镜像和源码镜像使用一致的命名规则

#### 3. 简化的 Mock 环境
- 移除复杂的数据初始化
- 专注于脚本功能验证
- 提供连接性测试工具

### 📋 可用命令

#### 源码服务命令
```bash
./build.sh list [tag] [registry]              # 列出所有服务和镜像
./build.sh build <service> [tag] [registry]   # 构建单个服务
./build.sh build-all [tag] [registry]         # 构建所有服务
./build.sh push <service> <registry> [tag]    # 推送单个服务
./build.sh push-all <registry> [tag]          # 推送所有服务
./build.sh build-push <registry> [tag]        # 一键构建并推送所有服务
```

#### 依赖镜像命令
```bash
./build.sh deps-pull <registry> [tag]         # 拉取并标记依赖镜像
./build.sh deps-push <registry> [tag]         # 推送依赖镜像
./build.sh deps-all <registry> [tag]          # 拉取、标记并推送所有依赖镜像
```

#### Mock 测试命令
```bash
./build.sh mock-setup [tag]                   # 创建 Mock 环境配置
./build.sh mock-up [tag]                      # 启动 Mock 测试环境
./build.sh mock-down                          # 停止 Mock 测试环境
./build.sh mock-restart [tag]                 # 重启 Mock 测试环境
./build.sh mock-test                          # 运行连接测试
```

#### 工具命令
```bash
./build.sh clean [tag] [--force]              # 清理本地镜像
./build.sh version                            # 显示版本信息
./build.sh help                               # 显示帮助信息
```

### 🎯 使用示例

#### Harbor 仓库操作
```bash
# 构建并推送到 Harbor 仓库
./build.sh build-push harbor.company.com/ai-infra v0.3.5

# 处理依赖镜像
./build.sh deps-all harbor.company.com/ai-infra v0.3.5

# 单个服务操作
./build.sh build backend v0.3.5 harbor.company.com/ai-infra
./build.sh push backend harbor.company.com/ai-infra v0.3.5
```

#### Mock 测试环境
```bash
# 设置并启动 Mock 环境
./build.sh mock-setup v0.3.5
./build.sh mock-up v0.3.5

# 测试连接
./build.sh mock-test

# 停止环境
./build.sh mock-down
```

### ✅ 验证结果

#### 1. 构建功能
- ✅ 所有 5 个服务构建成功
- ✅ Harbor 格式镜像名正确生成
- ✅ 本地别名自动创建

#### 2. 依赖镜像功能
- ✅ 镜像标记格式正确
- ✅ 支持 Harbor 项目路径

#### 3. Mock 环境功能
- ✅ 环境配置生成成功
- ✅ 服务启动正常
- ✅ 连接测试工具可用

### 🔄 兼容性

- **向后兼容**: 所有原有命令继续工作
- **macOS 支持**: 兼容 bash 3.2，无需升级
- **Docker 版本**: 兼容标准 Docker 和 Docker Desktop

### 📝 注意事项

1. **Harbor 格式**: 使用 `registry.domain.com/project` 格式时，脚本会自动识别为 Harbor 格式
2. **依赖镜像命名**: 依赖镜像使用 `ai-infra-deps-` 前缀以区分源码镜像
3. **Mock 环境**: 仅用于脚本功能验证，不包含完整的业务数据
4. **镜像标签**: 默认使用 `v0.3.5`，可通过参数覆盖

### 🎉 总结

升级后的 `build.sh` 脚本提供了完整的 CI/CD 支持，包括：
- 源码服务的构建和推送
- 依赖镜像的管理和分发  
- Mock 环境的快速验证
- Harbor 仓库的原生支持

脚本已通过完整测试，可以投入生产使用。
