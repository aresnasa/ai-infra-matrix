# Harbor 镜像 Tag 支持增强

## 概述

`tag_image_smart` 函数现已支持自动创建 Harbor 仓库 tag，方便后续推送镜像到私有仓库。

## 功能说明

### 自动创建的 Tag 类型

当指定 Harbor 仓库地址时，函数会自动创建以下 4 种 tag：

#### 示例 1: 标准镜像（无命名空间）
输入镜像：`golang:1.25-alpine`  
Harbor 地址：`aiharbor.msxf.local/aihpc`

创建的 tag：
```bash
golang:1.25-alpine                                   # 1. 标准名称
localhost/golang:1.25-alpine                         # 2. localhost 别名
aiharbor.msxf.local/aihpc/golang:1.25-alpine        # 3. Harbor 完整路径 ✨新增
```

#### 示例 2: 带命名空间的镜像
输入镜像：`osixia/openldap:stable`  
Harbor 地址：`aiharbor.msxf.local/aihpc`

创建的 tag：
```bash
osixia/openldap:stable                               # 1. 完整名称（带命名空间）
openldap:stable                                      # 2. 短名称（无命名空间）
localhost/openldap:stable                            # 3. localhost 别名
aiharbor.msxf.local/aihpc/osixia/openldap:stable    # 4. Harbor 完整路径 ✨新增
```

## 使用场景

### 场景 1: 自动拉取并创建 Harbor Tag

在内网环境中，构建服务时会自动：
1. 检查本地是否已有镜像
2. 如果不存在，优先从 Harbor 拉取
3. 创建所有必要的 tag（包括 Harbor tag）

```bash
./build.sh build backend --force
```

输出示例：
```
[INFO] 📦 预拉取依赖镜像: backend
[INFO]   ✓ 镜像已存在: golang:1.25-alpine
[INFO]   ✓ 本地已有镜像: golang:1.25-alpine
[INFO]   🏢 内网环境：创建 tag 别名
[SUCCESS]     ✓ 已创建别名: golang:1.25-alpine → localhost/golang:1.25-alpine
[SUCCESS]     ✓ 已创建 Harbor 别名: golang:1.25-alpine → aiharbor.msxf.local/aihpc/golang:1.25-alpine
```

### 场景 2: 批量为镜像创建 Harbor Tag

在 `build-all` 流程中，所有基础镜像都会自动创建 Harbor tag：

```bash
./build.sh build-all --force
```

步骤 1 输出示例：
```
[INFO] 步骤 1/5: 智能镜像管理（拉取 + Tag）
[INFO] 🌐 检测到网络环境: internal
[INFO] 📦 内网 Harbor 仓库: aiharbor.msxf.local/aihpc

[INFO] 处理镜像: golang:1.25-alpine
[INFO]   ✓ 本地已有镜像: golang:1.25-alpine
[INFO]   🏢 内网环境：创建 tag 别名
[SUCCESS]     ✓ 已创建别名: golang:1.25-alpine → localhost/golang:1.25-alpine
[SUCCESS]     ✓ 已创建 Harbor 别名: golang:1.25-alpine → aiharbor.msxf.local/aihpc/golang:1.25-alpine

[INFO] 处理镜像: osixia/openldap:stable
[INFO]   ✓ 本地已有镜像: osixia/openldap:stable
[INFO]   🏢 内网环境：创建 tag 别名
[SUCCESS]     ✓ 已创建别名: osixia/openldap:stable → openldap:stable
[SUCCESS]     ✓ 已创建别名: osixia/openldap:stable → localhost/openldap:stable
[SUCCESS]     ✓ 已创建 Harbor 别名: osixia/openldap:stable → aiharbor.msxf.local/aihpc/osixia/openldap:stable
```

### 场景 3: 手动指定 Harbor 地址

可以通过 `--registry` 参数指定自定义的 Harbor 地址：

```bash
./build.sh build backend --registry harbor.company.com/ai-infra
```

这样会创建：
```bash
golang:1.25-alpine
localhost/golang:1.25-alpine
harbor.company.com/ai-infra/golang:1.25-alpine    # 使用自定义 Harbor 地址
```

## 推送到 Harbor

创建 Harbor tag 后，可以直接推送到私有仓库：

```bash
# 推送单个镜像
docker push aiharbor.msxf.local/aihpc/golang:1.25-alpine

# 批量推送所有基础镜像
./build.sh deps-push aiharbor.msxf.local/aihpc v0.3.6-dev
```

## 技术细节

### tag_image_smart 函数参数

```bash
tag_image_smart <image> [network_env] [harbor_registry] [auto_pull]
```

- `image`: 镜像名称（必需）
- `network_env`: 网络环境（`auto`/`external`/`internal`，默认 `auto`）
- `harbor_registry`: Harbor 仓库地址（默认 `aiharbor.msxf.local/aihpc`）
- `auto_pull`: 是否自动拉取不存在的镜像（默认 `true`）

### 智能检测逻辑

1. **检查本地是否已有镜像**（任意一种 tag 存在即可）：
   - `image:tag`
   - `short_name:tag`
   - `localhost/short_name:tag`

2. **如果不存在且 `auto_pull=true`**：
   - 内网环境：优先从 Harbor 拉取，失败则尝试公共源
   - 公网环境：直接从公共源拉取

3. **创建所有必要的 tag**：
   - 标准名称（完整命名空间）
   - 短名称（无命名空间）
   - localhost 别名
   - **Harbor 完整路径**（内网环境或明确指定时）

### Harbor Tag 创建条件

Harbor tag 会在以下情况创建：

**内网环境 (internal)**:
- 总是创建 `harbor_registry/base_image:tag`
- 前提：`harbor_registry` 参数有效且不是源镜像本身

**公网环境 (external)**:
- 仅当用户**明确指定**非默认 Harbor 地址时创建
- 判断条件：`harbor_registry != "aiharbor.msxf.local/aihpc"`

## 版本历史

### v1.1.0 (2025-10-12)
- ✅ 新增：自动创建 Harbor 完整路径 tag
- ✅ 新增：`auto_pull` 参数控制自动拉取行为
- ✅ 优化：统一的镜像检测逻辑（步骤 1）
- ✅ 优化：智能拉取策略（步骤 2）
- ✅ 优化：双向 tag 创建（步骤 3）

### v1.0.0 (2025-10-11)
- ✅ 基础功能：双向 tag（标准名称 ↔ localhost）
- ✅ 基础功能：网络环境自动检测

## 相关文档

- [构建系统双向 Tag 指南](BUILD_BIDIRECTIONAL_TAG_GUIDE.md)
- [镜像 Tag 问题修复报告](IMAGE_TAG_FIX_REPORT.md)
- [构建脚本使用指南](BUILD_USAGE_GUIDE.md)

## 常见问题

### Q: 为什么需要 Harbor tag？
A: 方便后续推送镜像到私有仓库，无需手动 tag。

### Q: Harbor tag 什么时候会被创建？
A: 内网环境下总是创建；公网环境下仅当明确指定非默认 Harbor 地址时创建。

### Q: 如果本地已有镜像，还会拉取吗？
A: 不会。检测到本地已有任意一种 tag（标准名称/短名称/localhost），就直接使用本地镜像创建其他 tag。

### Q: 如何禁用自动拉取？
A: 设置 `auto_pull=false`，或使用全局参数 `./build.sh build backend --skip-pull`。

### Q: 推送到 Harbor 后，其他机器如何使用？
A: 其他机器可以直接从 Harbor 拉取：
```bash
docker pull aiharbor.msxf.local/aihpc/golang:1.25-alpine
```
然后构建脚本会自动创建所有必要的 tag。
