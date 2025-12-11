# 构建自动版本管理指南

## 概述

从 v0.3.8 开始，`build.sh` 脚本新增了自动版本管理功能，支持基于 Git 分支的自动标签检测和依赖版本统一管理，大幅简化内网部署流程。

## 核心功能

### 1. Git 分支自动标签检测

**功能**: 自动将当前 Git 分支名作为镜像标签

**支持命令**:
- `build-all`
- `push-all`
- `push-dep`

**工作原理**:
```bash
# 示例：当前分支为 v0.3.8
git branch --show-current  # 输出: v0.3.8

# 执行构建时未指定标签
./build.sh build-all

# 自动检测并使用 v0.3.8 作为镜像标签
# 等同于: ./build.sh build-all v0.3.8
```

### 2. deps.yaml 依赖版本管理

**功能**: 集中定义所有上游依赖镜像版本

**文件位置**: `deps.yaml`

**文件格式**:
```yaml
# 数据库
postgres: "15-alpine"
mysql: "8.0"
redis: "7-alpine"

# 编程语言运行时
golang: "1.25-alpine"
node: "22-alpine"
python: "3.14-alpine"

# 应用程序
gitea: "1.25.1"
nginx: "stable-alpine-perl"
seaweedfs: "latest"
```

**同步机制**:
- `build-all` 自动调用 `sync_deps_from_yaml()` 函数
- 将 deps.yaml 中的版本同步到 .env 文件
- 例如: `postgres: "15-alpine"` → `.env` 中 `POSTGRES_VERSION=15-alpine`

### 3. 环境变量自动更新

**功能**: 自动更新 .env 文件中的版本相关变量

**更新变量**:
- `IMAGE_TAG` - 组件镜像标签
- `DEFAULT_IMAGE_TAG` - 默认镜像标签
- `*_VERSION` - 各依赖镜像版本（从 deps.yaml 同步）

## 使用场景

### 场景 1: 开发环境快速构建

```bash
# 1. 切换到目标分支
git checkout v0.3.8

# 2. 直接构建，无需指定标签
./build.sh build-all

# 输出:
# [INFO] 步骤 0: 自动版本检测与同步
# [INFO] 未手动指定标签，自动从 Git 分支检测...
# [INFO] 检测到 Git 分支: v0.3.8
# [INFO] 已设置组件标签: v0.3.8
# [INFO] 已自动设置标签为: v0.3.8
# [INFO] 从 deps.yaml 同步依赖版本到 /path/to/.env
# [INFO]   ✓ POSTGRES_VERSION=15-alpine
# [INFO]   ✓ MYSQL_VERSION=8.0
# ...
```

### 场景 2: 内网部署镜像推送

```bash
# 1. 确认当前分支
git branch --show-current  # v0.3.8

# 2. 推送所有镜像到内网 Harbor
./build.sh push-all harbor.example.com.example.local/aihpc

# 输出:
# [INFO] 未手动指定标签，从 Git 分支自动检测...
# [INFO] 已自动设置标签为: v0.3.8
# [INFO] 正在推送 ai-infra-backend:v0.3.8...
# [INFO] 正在推送 ai-infra-frontend:v0.3.8...
# ...

# 3. 只推送依赖镜像
./build.sh push-dep harbor.example.com.example.local/aihpc

# 输出:
# [INFO] 未手动指定标签，从 Git 分支自动检测...
# [INFO] 已自动设置标签为: v0.3.8
# [INFO] 正在推送 postgres:15-alpine...
# [INFO] 正在推送 redis:7-alpine...
# ...
```

### 场景 3: 多版本并行开发

```bash
# 开发 v0.3.8 功能分支
git checkout v0.3.8
./build.sh build-all
# → 构建 v0.3.8 标签镜像

# 切换到 v0.4.0 开发分支
git checkout v0.4.0
./build.sh build-all
# → 构建 v0.4.0 标签镜像

# 两个版本的镜像互不干扰
docker images | grep ai-infra-backend
# ai-infra-backend    v0.3.8    ...
# ai-infra-backend    v0.4.0    ...
```

### 场景 4: 手动指定标签（覆盖自动检测）

```bash
# 当前分支是 v0.3.8，但想构建测试版本
./build.sh build-all v0.3.8-test

# 输出:
# [INFO] 使用手动指定的标签: v0.3.8-test
# [INFO] 从 deps.yaml 同步依赖版本到 /path/to/.env
# ...
# → 构建 v0.3.8-test 标签镜像
```

## 依赖版本升级流程

### 1. 更新 deps.yaml

```bash
# 编辑 deps.yaml
vim deps.yaml

# 例如升级 PostgreSQL
postgres: "15-alpine"  # 旧版本
↓
postgres: "16-alpine"  # 新版本
```

### 2. 同步到 .env

```bash
# 方式 1: 执行 build-all 自动同步
./build.sh build-all

# 方式 2: 手动调用同步（在脚本内部）
# sync_deps_from_yaml "$SCRIPT_DIR/.env"
```

### 3. 验证同步结果

```bash
# 检查 .env 文件
cat .env | grep POSTGRES_VERSION
# 输出: POSTGRES_VERSION=16-alpine
```

### 4. 重新构建依赖的服务

```bash
# 强制重建使用 PostgreSQL 的服务
./build.sh build backend --force
```

## 实现细节

### 新增函数

#### 1. `get_current_git_branch()`

**功能**: 获取当前 Git 分支名

**实现**:
```bash
get_current_git_branch() {
    local branch=""
    
    # 检查是否在 Git 仓库中
    if git rev-parse --git-dir > /dev/null 2>&1; then
        # 尝试获取当前分支名
        branch=$(git branch --show-current 2>/dev/null || git rev-parse --abbrev-ref HEAD 2>/dev/null)
    fi
    
    # 返回分支名，如果获取失败则返回默认标签
    if [[ -n "$branch" && "$branch" != "HEAD" ]]; then
        echo "$branch"
    else
        echo "${DEFAULT_IMAGE_TAG:-latest}"
    fi
}
```

**后备机制**:
- 如果不在 Git 仓库中，返回 `DEFAULT_IMAGE_TAG`
- 如果 HEAD 处于分离状态，返回 `DEFAULT_IMAGE_TAG`

#### 2. `sync_deps_from_yaml()`

**功能**: 从 deps.yaml 同步版本到 .env

**实现**:
```bash
sync_deps_from_yaml() {
    local deps_file="$SCRIPT_DIR/deps.yaml"
    local env_file="${1:-$ENV_FILE}"
    
    if [[ ! -f "$deps_file" ]]; then
        print_warning "依赖文件不存在: $deps_file，跳过同步"
        return 0
    fi
    
    print_info "从 deps.yaml 同步依赖版本到 $env_file"
    
    # 读取 deps.yaml 并更新 .env
    while IFS=: read -r key value; do
        # 跳过注释和空行
        [[ "$key" =~ ^[[:space:]]*# ]] && continue
        [[ -z "$key" ]] && continue
        
        # 清理并转换为环境变量格式
        key=$(echo "$key" | xargs)
        value=$(echo "$value" | sed -e 's/^[[:space:]]*"//' -e 's/"[[:space:]]*$//' | xargs)
        
        local env_var_name=$(echo "${key}_VERSION" | tr '[:lower:]' '[:upper:]' | tr '-' '_')
        
        # 更新到 .env 文件
        set_or_update_env_var "$env_var_name" "$value" "$env_file"
        
        print_info "  ✓ $env_var_name=$value"
    done < <(grep -E '^\s*[a-zA-Z0-9_-]+:' "$deps_file")
    
    print_info "依赖版本同步完成"
    return 0
}
```

**命名规则**:
- YAML 键名转大写
- 添加 `_VERSION` 后缀
- 例如: `golang` → `GOLANG_VERSION`

#### 3. `update_component_tags_from_branch()`

**功能**: 根据分支更新组件标签

**实现**:
```bash
update_component_tags_from_branch() {
    local branch=$(get_current_git_branch)
    local env_file="${1:-$ENV_FILE}"
    
    print_info "检测到 Git 分支: $branch"
    
    # 导出到当前环境
    export IMAGE_TAG="$branch"
    export DEFAULT_IMAGE_TAG="$branch"
    
    # 更新到 .env 文件
    set_or_update_env_var "IMAGE_TAG" "$branch" "$env_file"
    set_or_update_env_var "DEFAULT_IMAGE_TAG" "$branch" "$env_file"
    
    print_info "已设置组件标签: $branch"
    return 0
}
```

### 函数调用流程

#### build-all 命令

```
build-all
  ↓
build_all_pipeline()
  ↓
步骤 0: 自动版本检测与同步
  ├─ update_component_tags_from_branch()  # 更新 IMAGE_TAG
  │    ├─ get_current_git_branch()        # 获取分支名
  │    └─ set_or_update_env_var()         # 写入 .env
  └─ sync_deps_from_yaml()                # 同步依赖版本
       └─ set_or_update_env_var()         # 写入 .env
  ↓
步骤 1-6: 原有构建流程
```

#### push-all 命令

```
push-all <registry> [tag]
  ↓
case "push-all":
  ├─ 检查是否手动指定标签
  │    ├─ 未指定 → get_current_git_branch()
  │    └─ 已指定 → 使用指定标签
  ├─ push_all_services()
  └─ push_all_dependencies()
```

## 最佳实践

### 1. 分支命名规范

**推荐格式**: `v<major>.<minor>.<patch>`

**示例**:
- `v0.3.8` - 稳定版本
- `v0.4.0` - 下一个大版本
- `v0.3.9-dev` - 开发分支
- `v0.3.8-hotfix` - 热修复分支

**优势**:
- 自动生成符合语义化版本规范的镜像标签
- 便于内网部署时识别版本
- 支持 Docker 镜像标签规范

### 2. deps.yaml 维护

**更新时机**:
- 上游依赖发布新版本
- 发现安全漏洞需要升级
- 新增依赖组件

**版本选择原则**:
- 优先使用 alpine 变体（减小镜像体积）
- 使用明确的版本号，避免 `latest`
- 测试后再应用到生产环境

**示例 deps.yaml**:
```yaml
# 数据库 - 使用明确的次版本号
postgres: "15.5-alpine"    # 推荐
# postgres: "latest"       # 不推荐

# 应用 - 指定完整版本
gitea: "1.25.1"            # 推荐
# gitea: "1.25"            # 可接受
# gitea: "latest"          # 不推荐
```

### 3. 内网部署工作流

**完整流程**:
```bash
# === 外网构建环境 ===
# 1. 切换到发布分支
git checkout v0.3.8

# 2. 构建所有镜像
./build.sh build-all

# 3. 推送到内网 Harbor
./build.sh push-all harbor.example.com.example.local/aihpc

# 4. 推送依赖镜像
./build.sh push-dep harbor.example.com.example.local/aihpc

# === 内网部署环境 ===
git checkout v0.3.8
./build.sh build-all
./build.sh push-all harbor.example.com/ai-infra
./build.sh push-dep harbor.example.com/ai-infra

# === 内网部署环境 ===
# 5. 拉取镜像
./build.sh harbor-pull-all harbor.example.com/ai-infra v0.3.8

# 6. 启动服务
docker-compose up -d
```

**自动化脚本**:
```bash
#!/bin/bash
# deploy-to-intranet.sh

BRANCH="v0.3.8"
REGISTRY="harbor.example.com/ai-infra"

echo "=== 步骤 1: 切换分支 ==="
git checkout "$BRANCH"

echo "=== 步骤 2: 构建镜像 ==="
./build.sh build-all

echo "=== 步骤 3: 推送服务镜像 ==="
./build.sh push-all "$REGISTRY"

echo "=== 步骤 4: 推送依赖镜像 ==="
./build.sh push-dep "$REGISTRY"

echo "=== 完成 ==="
echo "现在可以在内网环境执行："
echo "  ./build.sh harbor-pull-all $REGISTRY $BRANCH"
```

### 4. 多环境配置管理

**场景**: 开发、测试、生产环境使用不同配置

**方案 1: 分支隔离**
```bash
# 开发环境
git checkout dev
./build.sh build-all  # → dev 标签

# 测试环境
git checkout test
./build.sh build-all  # → test 标签

# 生产环境
git checkout v0.3.8
./build.sh build-all  # → v0.3.8 标签
```

**方案 2: 手动覆盖**
```bash
# 生产环境强制使用 prod 标签
./build.sh build-all prod
./build.sh push-all harbor.example.com/ai-infra prod
```

## 故障排查

### 问题 1: 无法检测 Git 分支

**症状**:
```
[INFO] 检测到 Git 分支: latest
```

**原因**:
- 不在 Git 仓库中
- HEAD 处于分离状态

**解决**:
```bash
# 确认是否在 Git 仓库中
git status

# 切换到具体分支
git checkout v0.3.8

# 或手动指定标签
./build.sh build-all v0.3.8
```

### 问题 2: deps.yaml 同步失败

**症状**:
```
[WARNING] 依赖文件不存在: /path/to/deps.yaml，跳过同步
```

**原因**:
- deps.yaml 文件不存在
- 文件路径错误

**解决**:
```bash
# 确认文件存在
ls -l deps.yaml

# 检查文件格式
cat deps.yaml

# 手动创建（如果缺失）
cat > deps.yaml <<'EOF'
postgres: "15-alpine"
redis: "7-alpine"
EOF
```

### 问题 3: 环境变量未更新

**症状**:
```bash
echo $IMAGE_TAG
# 输出: latest (预期: v0.3.8)
```

**原因**:
- 环境变量未重新加载

**解决**:
```bash
# 重新加载 .env
source .env

# 或在新的 shell 会话中执行
./build.sh build-all
```

## 性能优化

### 1. 跳过依赖同步（高级用户）

如果确认 deps.yaml 未变化，可以注释掉同步调用：

```bash
# 编辑 build.sh 中的 build_all_pipeline() 函数
# 注释掉这一行:
# sync_deps_from_yaml "$SCRIPT_DIR/.env"
```

**警告**: 仅在确认依赖未变化时使用，否则可能导致版本不一致。

### 2. 并行构建多个服务

```bash
# 使用 xargs 并行构建
echo "backend frontend apphub" | xargs -P 3 -n 1 -I {} ./build.sh build {}
```

## 升级说明

### 从旧版本迁移

**v0.3.7 及更早版本 → v0.3.8**

1. **创建 deps.yaml**:
```bash
cat > deps.yaml <<'EOF'
postgres: "15-alpine"
mysql: "8.0"
redis: "7-alpine"
golang: "1.25-alpine"
node: "22-alpine"
python: "3.14-alpine"
EOF
```

2. **调整构建脚本调用**:
```bash
# 旧方式（仍然有效）
./build.sh build-all v0.3.8

# 新方式（自动检测）
./build.sh build-all
```

3. **验证功能**:
```bash
# 测试自动标签检测
git checkout v0.3.8
./build.sh build-all --help
```

## 相关文档

- [构建脚本完整指南](BUILD_COMPLETE_USAGE.md)
- [内网部署指南](BUILD_ENV_MANAGEMENT.md)
- [依赖镜像管理](BUILD_IMAGE_PREFETCH.md)
- [版本管理最佳实践](APPHUB_VERSION_MANAGEMENT.md)

## 总结

自动版本管理功能通过以下方式简化了构建和部署流程：

1. **自动标签检测**: Git 分支名自动成为镜像标签
2. **集中版本管理**: deps.yaml 统一管理所有依赖版本
3. **一键同步**: 自动将配置同步到 .env 文件
4. **内网友好**: 专为内网部署场景优化

**核心优势**:
- ✅ 减少手动输入错误
- ✅ 确保版本一致性
- ✅ 简化内网部署流程
- ✅ 支持多版本并行开发
- ✅ 向后兼容（可手动覆盖）
