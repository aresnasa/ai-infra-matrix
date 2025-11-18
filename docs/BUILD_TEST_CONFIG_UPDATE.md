# 构建和测试配置更新总结

## 日期
2025年10月9日

## 更新内容

### 1. 支持测试容器的统一构建

#### 修改的文件：

**build.sh**
- 在 `get_all_services()` 函数中添加了 `test-containers` 服务
- 在 `get_service_path()` 函数中添加了 test-containers 的路径映射
- 现在 `./build.sh build-all` 会包含测试容器的构建

**src/test-containers/Dockerfile**
- 创建了标准的 Dockerfile（从 Dockerfile.ssh 复制并添加 VERSION 参数）
- 添加了 ARG VERSION 和 APP_VERSION 环境变量
- 添加了标准的镜像标签（maintainer, version, description）
- 保持与 Dockerfile.ssh 相同的功能（SSH + systemd）

**docker-compose.test.yml**
- 添加了 `image` 字段，使用预构建的镜像 `ai-infra-test-containers:${IMAGE_TAG}`
- 保留 `build` 字段作为后备方案（如果镜像不存在会自动构建）
- 添加了 `args.VERSION` 参数传递
- 所有三个测试容器（test-ssh01, test-ssh02, test-ssh03）都使用相同的镜像

### 2. 新增文档

**docs/BUILD_AND_TEST_GUIDE.md**
- 完整的构建和测试指南
- 包含快速开始、详细说明、故障排查等章节
- 提供了完整的命令示例和预期输出

**quick-start-test.sh**
- 一键构建和测试脚本
- 自动执行所有必要步骤：
  1. 构建所有服务镜像
  2. 创建 Docker 网络
  3. 启动测试容器
  4. 验证部署状态
- 提供友好的输出和错误处理

### 3. 工作流程更新

#### 旧流程：
```bash
# 手动构建每个服务
docker build -t ai-infra-backend:v0.3.6-dev src/backend
docker build -t ai-infra-frontend:v0.3.6-dev src/frontend
# ... 更多服务

# 手动启动测试容器（每次都重新构建）
docker compose -f docker-compose.test.yml up -d
```

#### 新流程：
```bash
# 方式 1: 使用 build.sh（推荐）
./build.sh build-all --force && docker compose -f docker-compose.test.yml up -d

# 方式 2: 使用快速启动脚本
./quick-start-test.sh
```

## 技术改进

### 1. 镜像复用
- 测试容器现在作为标准服务构建，生成可复用的镜像
- 避免每次启动测试环境时重新构建
- 提高测试启动速度

### 2. 版本管理
- 测试容器镜像支持版本标签（通过 IMAGE_TAG 环境变量）
- 与其他服务镜像保持一致的版本管理
- 支持多版本并存

### 3. 统一构建流程
- 所有服务（包括测试容器）通过同一个 build.sh 脚本管理
- 统一的命名规范：`ai-infra-{service}:{tag}`
- 统一的构建参数和配置

### 4. ARM64 支持
- test-containers/Dockerfile 已配置阿里云镜像源
- 支持 ARM64 架构（Apple Silicon）
- 与 saltstack 和 slurm-master 使用相同的镜像源策略

## 验证测试

### 构建测试
```bash
$ ./build.sh build test-containers v0.3.6-dev
[INFO] 构建服务: test-containers
[INFO]   Dockerfile: src/test-containers/Dockerfile
[INFO]   目标镜像: ai-infra-test-containers:v0.3.6-dev
[SUCCESS] ✓ 构建成功: ai-infra-test-containers:v0.3.6-dev

$ docker images | grep test-containers
ai-infra-test-containers   v0.3.6-dev   9b37d29404f2   About a minute ago   729MB
```

### 启动测试
```bash
$ docker compose -f docker-compose.test.yml up -d
✔ Container test-ssh01  Started
✔ Container test-ssh02  Started
✔ Container test-ssh03  Started

$ docker ps | grep test-ssh
test-ssh01   ai-infra-test-containers:v0.3.6-dev   Up 28 seconds   0.0.0.0:2201->22/tcp
test-ssh02   ai-infra-test-containers:v0.3.6-dev   Up 28 seconds   0.0.0.0:2202->22/tcp
test-ssh03   ai-infra-test-containers:v0.3.6-dev   Up 28 seconds   0.0.0.0:2203->22/tcp
```

### systemd 验证
```bash
$ docker exec test-ssh01 systemctl status --no-pager | head -10
● test-ssh01
    State: running
     Jobs: 0 queued
   Failed: 0 units
    Since: Thu 2025-10-09 13:37:43 UTC; 1min 4s ago
```

## 使用示例

### 完整构建和测试流程

```bash
# 步骤 1: 克隆仓库
git clone <repository-url>
cd ai-infra-matrix

# 步骤 2: 一键构建和启动
./quick-start-test.sh

# 或者手动执行
./build.sh build-all --force
docker compose -f docker-compose.test.yml up -d

# 步骤 3: 测试 SSH 连接
ssh -p 2201 testuser@localhost
# 密码: testpass123

# 步骤 4: 清理环境
docker compose -f docker-compose.test.yml down
```

### 单独构建测试容器

```bash
# 构建特定版本
./build.sh build test-containers v1.0.0

# 查看构建的镜像
docker images | grep test-containers

# 使用特定版本启动
IMAGE_TAG=v1.0.0 docker compose -f docker-compose.test.yml up -d
```

## 优势

1. **效率提升**
   - 测试容器镜像只需构建一次，可多次使用
   - 减少了重复构建的时间（从每次 ~3 分钟降低到 ~5 秒启动时间）

2. **一致性**
   - 所有服务使用统一的构建流程
   - 统一的镜像命名和版本管理
   - 统一的配置模板渲染

3. **可维护性**
   - 集中管理所有服务（包括测试工具）
   - 统一的构建脚本和文档
   - 清晰的依赖关系

4. **可扩展性**
   - 易于添加新的测试容器
   - 支持多种测试场景（SSH、K8s、SLURM 等）
   - 灵活的版本和配置管理

## 后续计划

1. **CI/CD 集成**
   - 在 CI pipeline 中使用 `./build.sh build-all`
   - 自动化测试容器的构建和验证
   - 镜像推送到私有仓库

2. **测试场景扩展**
   - 添加更多类型的测试容器（K8s nodes, SLURM compute nodes）
   - 集成自动化测试脚本
   - 添加健康检查和监控

3. **性能优化**
   - 并行构建多个服务
   - 使用 BuildKit 缓存优化
   - 多阶段构建减小镜像大小

## 相关文档

- [BUILD_AND_TEST_GUIDE.md](./BUILD_AND_TEST_GUIDE.md) - 详细的构建和测试指南
- [DOCKER_COMPOSE_V2.39.2_COMPATIBILITY.md](./DOCKER_COMPOSE_V2.39.2_COMPATIBILITY.md) - Docker Compose 兼容性
- [SYSTEMD_CONTAINER_GUIDE.md](./SYSTEMD_CONTAINER_GUIDE.md) - Systemd 容器配置（如果存在）

## 问题反馈

如遇到问题，请：
1. 查看 [BUILD_AND_TEST_GUIDE.md](./BUILD_AND_TEST_GUIDE.md) 的故障排查章节
2. 检查容器日志：`docker logs test-ssh01`
3. 验证 systemd 状态：`docker exec test-ssh01 systemctl status`
4. 提交 issue 到项目仓库
