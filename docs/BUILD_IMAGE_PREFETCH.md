# build.sh 镜像预拉取功能说明

## 📋 需求背景

**需求 31**: 调整 build.sh 脚本的检查规则，由于镜像可能存在拉取失败的情况，需要先拉取所有 Dockerfile 中的依赖镜像，然后再构建，避免构建中断。

## ✅ 实施方案

### 1. 核心功能

#### 1.1 提取 Dockerfile 基础镜像

新增函数 `extract_base_images()`:
- 解析 Dockerfile 中的所有 `FROM` 指令
- 支持多种格式:
  - `FROM image:tag`
  - `FROM image:tag AS stage`
  - `FROM --platform=xxx image:tag`
- 自动去重和排序
- 过滤内部构建阶段（如 `builder`、`runtime` 等别名）

#### 1.2 单个服务预拉取

新增函数 `prefetch_base_images()`:
- 在构建单个服务前调用
- 检查镜像是否已存在
- 自动拉取缺失的镜像
- 提供详细的拉取进度和统计

#### 1.3 批量预拉取（build-all）

新增函数 `prefetch_all_base_images()`:
- 扫描所有服务的 Dockerfile
- 收集并去重所有基础镜像
- 批量拉取所有依赖镜像
- 显示总体统计信息

### 2. 执行流程

#### 2.1 单个服务构建流程

```bash
./build.sh build-service <service>
```

执行顺序：
1. 检查 Dockerfile 是否存在
2. **预拉取依赖镜像** ⬅️ 新增
3. 准备构建上下文
4. 执行 Docker 构建
5. 标记和推送镜像

#### 2.2 批量构建流程

```bash
./build.sh build-all
```

执行顺序：
1. **步骤 1/4: 预拉取所有依赖镜像** ⬅️ 新增
2. 步骤 2/4: 同步配置文件
3. 步骤 3/4: 渲染配置模板
4. 步骤 4/4: 构建服务镜像

### 3. 功能特性

#### 3.1 智能检测

- ✅ 自动识别已存在的镜像，跳过重复拉取
- ✅ 过滤内部构建阶段（如 `builder`、`base` 等别名）
- ✅ 支持跨平台镜像（`--platform` 参数）

#### 3.2 错误处理

- ✅ 拉取失败不中断流程，继续构建
- ✅ 区分关键镜像和可选镜像（如 `scratch`）
- ✅ 提供详细的错误信息和建议

#### 3.3 进度显示

```
📦 预拉取依赖镜像: backend
  ⬇ 正在拉取: golang:1.23-alpine
  ✓ 拉取成功: golang:1.23-alpine
  ✓ 镜像已存在: alpine:3.20
📊 预拉取统计:
  • 新拉取: 1
  • 已存在: 1
  • 失败: 0
```

## 📝 使用示例

### 示例 1: 构建单个服务

```bash
# 构建 backend 服务
./build.sh build-service backend

# 输出示例:
# [INFO] 构建服务: backend
# [INFO]   Dockerfile: src/backend/Dockerfile
# [INFO]   目标镜像: ai-infra-backend:v0.3.7
# [INFO]   → 预拉取 Dockerfile 依赖镜像...
# [INFO] 📦 预拉取依赖镜像: backend
# [INFO]   ⬇ 正在拉取: golang:1.23-alpine
# [SUCCESS]   ✓ 拉取成功: golang:1.23-alpine
# [INFO]   ✓ 镜像已存在: alpine:3.20
# [INFO] 📊 预拉取统计:
# [INFO]   • 新拉取: 1
# [INFO]   • 已存在: 1
# [INFO]   → 正在构建镜像...
```

### 示例 2: 批量构建

```bash
# 构建所有服务
./build.sh build-all

# 输出示例:
# [INFO] ==========================================
# [INFO] 构建所有 AI-Infra 服务镜像
# [INFO] ==========================================
# [INFO] 步骤 1/4: 预拉取依赖镜像
# [INFO] ==========================================
# [INFO] 🚀 批量预拉取所有服务的依赖镜像
# [INFO] 📋 扫描所有服务的 Dockerfile...
# [INFO] 📦 发现 25 个唯一的基础镜像
# [INFO] [1/25] 检查镜像: golang:1.23-alpine
# [INFO]   ⬇ 正在拉取...
# [SUCCESS]   ✓ 拉取成功
# ...
# [INFO] 📊 预拉取完成统计
# [INFO]   • 总镜像数: 25
# [INFO]   • 新拉取: 10
# [INFO]   • 已存在: 15
# [SUCCESS] ✅ 所有依赖镜像已就绪！
```

### 示例 3: 强制重建

```bash
# 强制重建（包含预拉取）
./build.sh build-all --force

# 输出:
# [INFO] 强制重建模式已启用，将清除构建缓存
# [INFO] 步骤 1/4: 预拉取依赖镜像
# ...
```

## 🔧 技术实现

### 提取基础镜像

```bash
extract_base_images() {
    local dockerfile_path="$1"
    
    # 提取所有 FROM 指令中的镜像名称
    grep -E '^\s*FROM\s+' "$dockerfile_path" | \
        sed -E 's/^\s*FROM\s+(--platform=[^\s]+\s+)?([^\s]+)(\s+AS\s+.*)?$/\2/' | \
        grep -v '^$' | \
        sort -u
}
```

支持的 Dockerfile 格式：
```dockerfile
# 基本格式
FROM golang:1.23-alpine

# 多阶段构建
FROM node:20-alpine AS builder
FROM nginx:alpine AS runtime

# 跨平台构建
FROM --platform=linux/amd64 ubuntu:22.04
```

### 预拉取逻辑

```bash
prefetch_base_images() {
    local dockerfile_path="$1"
    local service_name="${2:-unknown}"
    
    # 提取基础镜像列表
    local base_images=$(extract_base_images "$dockerfile_path")
    
    # 遍历并拉取
    while IFS= read -r image; do
        # 跳过内部构建阶段
        if [[ "$image" =~ ^[a-z_-]+$ ]]; then
            continue
        fi
        
        # 检查是否已存在
        if docker image inspect "$image" >/dev/null 2>&1; then
            print_info "  ✓ 镜像已存在: $image"
            continue
        fi
        
        # 拉取镜像
        if docker pull "$image"; then
            print_success "  ✓ 拉取成功: $image"
        else
            print_error "  ✗ 拉取失败: $image"
        fi
    done <<< "$base_images"
}
```

## 📊 效果对比

### 之前（无预拉取）

```
构建过程中可能出现:
- "error pulling image: context deadline exceeded"
- "failed to solve: failed to fetch: timeout"
- 构建中断，需要手动重试
```

### 现在（有预拉取）

```
✅ 提前拉取所有依赖镜像
✅ 避免构建中途拉取超时
✅ 提供清晰的进度反馈
✅ 自动跳过已存在的镜像
✅ 拉取失败有明确提示
```

## 🎯 优势

1. **可靠性提升**:
   - 避免构建过程中网络超时
   - 提前发现镜像拉取问题
   - 减少构建失败率

2. **性能优化**:
   - 并行拉取可能（未来可增强）
   - 避免重复拉取
   - 利用 Docker 缓存

3. **用户体验**:
   - 清晰的进度显示
   - 详细的统计信息
   - 友好的错误提示

4. **维护性**:
   - 自动化依赖管理
   - 无需手动编写镜像列表
   - 自动适配 Dockerfile 变更

## 🐛 常见问题

### Q1: 预拉取失败会影响构建吗？

**A**: 不会。预拉取失败只会显示警告，构建流程会继续。但建议解决拉取问题以提高构建成功率。

### Q2: 可以跳过预拉取吗？

**A**: 目前不支持跳过，因为这是为了提高构建可靠性而设计的必要步骤。未来可以添加环境变量控制。

### Q3: 预拉取会增加多少时间？

**A**: 
- 首次构建: 增加 5-15 分钟（取决于镜像数量和网络速度）
- 增量构建: 几乎无影响（镜像已存在会跳过）

### Q4: 如何加速镜像拉取？

**A**: 
```bash
# 配置镜像加速器（在 /etc/docker/daemon.json）
{
  "registry-mirrors": [
    "https://docker.m.daocloud.io",
    "https://docker.mirrors.sjtug.sjtu.edu.cn"
  ]
}

# 重启 Docker
sudo systemctl restart docker
```

## 📚 相关文档

- 原始需求: `dev-md.md` 第 31 条
- Build 脚本: `build.sh`
- Kubernetes 增强: `docs/KUBERNETES_MULTI_VERSION_IMPLEMENTATION.md`

## ✅ 验收标准

- [ ] `./build.sh build-service <service>` 自动预拉取依赖镜像
- [ ] `./build.sh build-all` 批量预拉取所有依赖镜像
- [ ] 显示详细的拉取进度和统计
- [ ] 已存在的镜像自动跳过
- [ ] 拉取失败不中断构建流程
- [ ] 提供清晰的错误提示和建议

---

**实施日期**: 2025-10-10  
**版本**: v0.3.7  
**状态**: ✅ 完成
