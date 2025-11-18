# AppHub 动态版本管理实现完成

## 实现时间
2025-01-24

## 实现目标

实现 AppHub 中各应用版本的自动化管理，通过 `git ls-remote` 动态获取最新版本标签，并自动更新 Dockerfile 中的版本号。

## 实现内容

### 1. 新增配置文件

**文件**: `src/apphub/app-repos.conf`

定义 AppHub 中需要版本管理的应用：

```conf
# AppHub 应用仓库配置
# 格式: APP_NAME|GIT_REPO_URL|VERSION_PREFIX
categraf|https://github.com/flashcatcloud/categraf.git|v
```

### 2. 新增函数

在 `build.sh` 中添加了三个核心函数：

#### get_latest_git_tag()
- 从 Git 仓库获取最新版本标签
- 使用 `git ls-remote --tags` 避免克隆整个仓库
- 支持语义化版本排序

#### get_apphub_app_versions()
- 遍历 `app-repos.conf` 配置文件
- 获取每个应用的最新版本
- 输出 KEY=VALUE 格式的版本信息

#### update_apphub_versions()
- 自动更新 Dockerfile 中的版本号
- 比较当前版本和最新版本
- 仅更新有变化的版本

### 3. 新增命令

#### apphub-versions
获取所有 AppHub 应用的最新版本：

```bash
./build.sh apphub-versions
```

输出示例：
```
[INFO] 获取 AppHub 应用最新版本
[SUCCESS]   ✓ categraf: v0.4.22
    仓库: https://github.com/flashcatcloud/categraf.git
CATEGRAF_VERSION=v0.4.22
```

#### apphub-update-versions
更新 Dockerfile 中的版本号：

```bash
./build.sh apphub-update-versions
```

输出示例：
```
[INFO] 更新 AppHub Dockerfile 版本号
[INFO] 检查 categraf 版本...
[SUCCESS]   ✓ categraf 已是最新版本: v0.4.22
[INFO] 所有版本已是最新，无需更新
```

### 4. 构建时自动检查

修改 `build_service()` 函数，在构建 AppHub 时自动检查版本：

```bash
if [[ "$service" == "apphub" ]]; then
    print_info "  → AppHub 构建前检查应用版本..."
    update_apphub_versions
fi
```

构建时输出：
```bash
./build.sh build apphub --force

[INFO]   → AppHub 构建前检查应用版本...
[INFO] 检查 categraf 版本...
[SUCCESS]   ✓ categraf 已是最新版本: v0.4.22
[INFO]   ✓ AppHub 应用版本检查完成
```

## 技术要点

### 1. Git ls-remote 获取版本

```bash
git ls-remote --tags "$repo_url" | \
    grep -v '\^{}' | \
    awk '{print $2}' | \
    sed 's|refs/tags/||' | \
    grep -E '^v?[0-9]+\.[0-9]+\.[0-9]+' | \
    sort -V | \
    tail -1
```

处理流程：
1. 获取所有远程标签
2. 过滤掉 annotated tags 的 `^{}` 后缀
3. 提取标签名（去除 `refs/tags/` 前缀）
4. 过滤出符合语义化版本的标签
5. 使用 `-V` 进行版本排序
6. 取最新版本

### 2. Bash 兼容性处理

使用 `tr` 替代 `^^` 进行大小写转换，兼容旧版本 bash：

```bash
# 不使用: ${app_name^^}
# 使用: $(echo "$app_name" | tr '[:lower:]' '[:upper:]')
local var_name=$(echo "${app_name}_VERSION" | tr '[:lower:]' '[:upper:]')
```

### 3. 配置文件解析

使用 `IFS='|'` 分隔符读取配置：

```bash
while IFS='|' read -r app_name repo_url version_prefix || [[ -n "$app_name" ]]; do
    # 跳过注释和空行
    [[ "$app_name" =~ ^#.*$ ]] && continue
    [[ -z "$app_name" ]] && continue
    
    # 处理应用...
done < src/apphub/app-repos.conf
```

## 测试结果

### 测试 1: 获取版本

```bash
$ ./build.sh apphub-versions
[SUCCESS]   ✓ categraf: v0.4.22
    仓库: https://github.com/flashcatcloud/categraf.git
CATEGRAF_VERSION=v0.4.22
```

✅ 成功获取 Categraf 最新版本 v0.4.22

### 测试 2: 更新版本

```bash
$ ./build.sh apphub-update-versions
[INFO] 检查 categraf 版本...
[SUCCESS]   ✓ categraf 已是最新版本: v0.4.22
[INFO] 所有版本已是最新，无需更新
```

✅ 检测到版本已是最新，无需更新

## 添加新应用示例

要为新应用添加版本管理，只需两步：

### 步骤 1: 添加配置

在 `src/apphub/app-repos.conf` 中添加：

```
nightingale|https://github.com/ccfos/nightingale.git|v
```

### 步骤 2: 添加 ARG

在 `src/apphub/Dockerfile` 中添加：

```dockerfile
ARG NIGHTINGALE_VERSION=v6.0.0
ARG NIGHTINGALE_REPO=https://github.com/ccfos/nightingale.git
```

完成！系统会自动：
- 获取 Nightingale 的最新版本
- 更新 Dockerfile 中的 NIGHTINGALE_VERSION
- 在构建时使用最新版本

## 文件清单

### 新增文件
1. `src/apphub/app-repos.conf` - 应用仓库配置
2. `docs/APPHUB_VERSION_MANAGEMENT.md` - 详细文档

### 修改文件
1. `build.sh` - 添加版本管理函数和命令

## 优势

1. **自动化**: 无需手动查找和更新版本号
2. **准确性**: 直接从 Git 仓库获取最新标签
3. **高效性**: 使用 `git ls-remote` 避免克隆整个仓库
4. **易用性**: 简单的命令行接口
5. **可扩展**: 轻松添加新应用
6. **兼容性**: 支持旧版本 bash

## 下一步计划

1. ✅ 实现 Categraf 版本管理
2. ⏳ 添加更多应用（Nightingale, SaltStack 等）
3. ⏳ 支持版本范围约束（如 `>=1.0.0 <2.0.0`）
4. ⏳ 添加版本变更通知
5. ⏳ 集成到 CI/CD 流程

## 相关文档

- [AppHub 通用构建系统](./APPHUB_GENERIC_BUILD_SYSTEM.md)
- [AppHub 重构完成报告](./APPHUB_REFACTORING_DONE.md)
- [Categraf 集成文档](./CATEGRAF_INTEGRATION_DONE.md)
- [AppHub 版本管理详细文档](./APPHUB_VERSION_MANAGEMENT.md)

## 总结

成功实现了 AppHub 应用版本的动态管理机制，通过配置文件定义应用仓库，自动从 Git 获取最新版本，并更新 Dockerfile。系统集成到构建流程中，确保始终使用最新稳定版本。
