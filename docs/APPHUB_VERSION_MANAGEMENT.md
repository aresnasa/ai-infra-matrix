# AppHub 应用版本动态管理

## 概述

为了解决 AppHub 中各应用版本手动更新的问题，我们实现了一套自动化的版本管理机制。系统会从 Git 仓库动态获取最新版本标签，并自动更新 Dockerfile 中的版本号。

## 功能特性

### 1. 动态版本获取
- 从 Git 仓库获取最新的版本标签
- 支持带前缀的版本号（如 `v0.4.22`）
- 使用 `git ls-remote` 命令避免克隆整个仓库
- 自动排序并选择最新版本

### 2. 自动版本更新
- 比较当前版本和最新版本
- 仅更新有变化的版本号
- 保留原始 Dockerfile 格式
- 自动备份和清理

### 3. 构建时自动检查
- 构建 AppHub 镜像时自动检查版本
- 确保始终使用最新稳定版本
- 避免手动维护版本号的错误

## 配置文件

### app-repos.conf

位置: `src/apphub/app-repos.conf`

格式:
```
# APP_NAME|GIT_REPO_URL|VERSION_PREFIX
categraf|https://github.com/flashcatcloud/categraf.git|v
```

字段说明:
- `APP_NAME`: 应用名称（对应 `scripts/<app-name>` 目录）
- `GIT_REPO_URL`: Git 仓库地址
- `VERSION_PREFIX`: 版本标签前缀（如 `v` 或留空）

## 使用方法

### 手动获取版本

查看所有 AppHub 应用的最新版本：

```bash
./build.sh apphub-versions
```

输出示例:
```
[INFO] ===========================================
[INFO] 获取 AppHub 应用最新版本
[INFO] ===========================================

[INFO] 正在获取 categraf 的最新版本...
[SUCCESS]   ✓ categraf: v0.4.22
    仓库: https://github.com/flashcatcloud/categraf.git
CATEGRAF_VERSION=v0.4.22
```

### 手动更新版本

更新 Dockerfile 中的版本号：

```bash
./build.sh apphub-update-versions
```

输出示例:
```
[INFO] ===========================================
[INFO] 更新 AppHub Dockerfile 版本号
[INFO] ===========================================

[INFO] 检查 categraf 版本...
[INFO]   更新 categraf: v0.3.90 → v0.4.22
[SUCCESS] ✓ Dockerfile 版本已更新: src/apphub/Dockerfile
```

或者如果已是最新:
```
[INFO] 检查 categraf 版本...
[SUCCESS]   ✓ categraf 已是最新版本: v0.4.22
[INFO] 所有版本已是最新，无需更新
```

### 自动更新（构建时）

构建 AppHub 镜像时会自动检查和更新版本：

```bash
./build.sh build apphub --force
```

构建过程中会看到:
```
[INFO]   → AppHub 构建前检查应用版本...
[INFO] 检查 categraf 版本...
[SUCCESS]   ✓ categraf 已是最新版本: v0.4.22
[INFO]   ✓ AppHub 应用版本检查完成
```

## 实现原理

### 1. 版本获取函数 (`get_latest_git_tag`)

```bash
get_latest_git_tag() {
    local repo_url="$1"
    
    # 使用 git ls-remote 获取所有标签
    local latest_tag
    latest_tag=$(git ls-remote --tags "$repo_url" 2>/dev/null | \
                 grep -v '\^{}' | \
                 awk '{print $2}' | \
                 sed 's|refs/tags/||' | \
                 grep -E '^v?[0-9]+\.[0-9]+\.[0-9]+' | \
                 sort -V | \
                 tail -1)
    
    echo "$latest_tag"
}
```

处理流程:
1. 使用 `git ls-remote --tags` 获取远程标签
2. 过滤掉 annotated tags 的 `^{}` 后缀
3. 提取标签名称（去除 `refs/tags/` 前缀）
4. 过滤出符合语义化版本的标签
5. 使用 `-V` 选项进行版本号排序
6. 取最新的一个版本

### 2. 应用版本遍历 (`get_apphub_app_versions`)

```bash
while IFS='|' read -r app_name repo_url version_prefix; do
    # 跳过注释和空行
    [[ "$app_name" =~ ^#.*$ ]] && continue
    [[ -z "$app_name" ]] && continue
    
    # 获取最新版本
    latest_version=$(get_latest_git_tag "$repo_url")
    
    # 输出版本信息
    echo "${app_name}_VERSION=${latest_version}"
done < src/apphub/app-repos.conf
```

### 3. Dockerfile 版本更新 (`update_apphub_versions`)

```bash
# 查找当前版本
old_version=$(grep "^ARG ${var_name}=" Dockerfile | cut -d'=' -f2)

# 比较并更新
if [[ "$old_version" != "$latest_version" ]]; then
    sed -i "s|^ARG ${var_name}=.*|ARG ${var_name}=${latest_version}|g" Dockerfile
fi
```

### 4. 构建时自动检查

在 `build_service` 函数中添加特殊处理：

```bash
if [[ "$service" == "apphub" ]]; then
    print_info "  → AppHub 构建前检查应用版本..."
    update_apphub_versions
fi
```

## 添加新应用

要为新应用添加版本管理，只需在 `app-repos.conf` 中添加一行：

```
# 添加新应用
nightingale|https://github.com/ccfos/nightingale.git|v
n9e-agent|https://github.com/flashcatcloud/categraf.git|v
```

然后在 Dockerfile 中添加对应的 ARG：

```dockerfile
# Stage N: Build new-app
FROM golang:1.23-alpine AS new-app-builder

ARG NEWAPP_VERSION=v1.0.0
ARG NEWAPP_REPO=https://github.com/xxx/newapp.git
```

注意:
- 配置文件中的应用名使用小写（如 `newapp`）
- Dockerfile 中的 ARG 变量名使用大写（如 `NEWAPP_VERSION`）
- 系统会自动进行大小写转换

## 版本标签规范

系统会过滤出符合以下正则表达式的标签：

```regex
^v?[0-9]+\.[0-9]+\.[0-9]+
```

支持的格式:
- ✅ `v0.4.22`
- ✅ `0.4.22`
- ✅ `v1.0.0-beta.1`
- ✅ `2.3.4-rc1`
- ❌ `latest`
- ❌ `main`
- ❌ `dev-branch`

## 命令参考

### apphub-versions

获取所有 AppHub 应用的最新版本。

```bash
./build.sh apphub-versions
```

参数: 无

输出:
- 每个应用的最新版本号
- Git 仓库地址
- KEY=VALUE 格式的版本信息

### apphub-update-versions

更新 AppHub Dockerfile 中的版本号。

```bash
./build.sh apphub-update-versions
```

参数: 无

功能:
- 自动获取最新版本
- 比较当前版本和最新版本
- 仅更新有变化的版本
- 保持 Dockerfile 格式不变

### build apphub

构建 AppHub 镜像（自动检查版本）。

```bash
./build.sh build apphub [--force]
```

参数:
- `--force`: 强制重新构建（可选）

特性:
- 构建前自动检查应用版本
- 如有新版本则自动更新
- 确保使用最新稳定版本

## 最佳实践

### 1. 定期检查版本

建议定期运行版本检查：

```bash
# 每周检查一次
./build.sh apphub-versions
```

### 2. 更新前验证

更新版本前建议查看更新日志：

```bash
# 1. 获取最新版本
./build.sh apphub-versions

# 2. 查看对应仓库的 CHANGELOG
# 3. 确认无重大变更后更新
./build.sh apphub-update-versions
```

### 3. 测试新版本

更新版本后建议进行测试：

```bash
# 1. 更新版本
./build.sh apphub-update-versions

# 2. 构建测试镜像
./build.sh build apphub --force

# 3. 运行测试
docker run --rm ai-infra-apphub:test ls /usr/share/nginx/html/pkgs/
```

### 4. 版本锁定

如果需要锁定特定版本，可以：

1. 在 `app-repos.conf` 中注释掉该应用
2. 手动在 Dockerfile 中设置固定版本
3. 添加注释说明锁定原因

```dockerfile
# Categraf 版本锁定在 v0.4.22（因为 v0.4.23 有已知问题）
ARG CATEGRAF_VERSION=v0.4.22
```

## 故障排查

### 问题: 无法获取版本

症状:
```
[ERROR] 无法获取仓库 https://... 的最新标签
```

可能原因:
1. 网络连接问题
2. Git 仓库地址错误
3. 仓库没有符合规范的版本标签

解决方法:
```bash
# 手动测试 Git 仓库连接
git ls-remote --tags https://github.com/flashcatcloud/categraf.git | grep -v '\^{}' | tail -20

# 检查配置文件
cat src/apphub/app-repos.conf
```

### 问题: 版本更新失败

症状:
```
[WARNING] 未在 Dockerfile 中找到 ARG CATEGRAF_VERSION
```

可能原因:
1. Dockerfile 中缺少对应的 ARG 声明
2. 变量名大小写不匹配

解决方法:
```bash
# 检查 Dockerfile 中的 ARG 声明
grep "ARG.*VERSION" src/apphub/Dockerfile

# 添加缺少的 ARG
echo "ARG CATEGRAF_VERSION=v0.4.22" >> src/apphub/Dockerfile
```

### 问题: bash 版本兼容性

症状:
```
bad substitution
```

可能原因:
- 使用了旧版本 bash，不支持某些语法

解决方法:
- 已在代码中使用 `tr '[:lower:]' '[:upper:]'` 替代 `^^` 语法
- 确保使用 bash 4.0 或更高版本

## 相关文件

- `build.sh` - 主构建脚本（包含版本管理函数）
- `src/apphub/app-repos.conf` - 应用仓库配置
- `src/apphub/Dockerfile` - AppHub 镜像定义
- `src/apphub/scripts/` - 应用构建脚本目录

## 更新日志

### v1.0.0 (2025-01-24)

**新增功能:**
- ✅ 实现 `get_latest_git_tag` 函数获取最新版本
- ✅ 实现 `get_apphub_app_versions` 遍历所有应用
- ✅ 实现 `update_apphub_versions` 自动更新版本
- ✅ 添加 `apphub-versions` 命令
- ✅ 添加 `apphub-update-versions` 命令
- ✅ 构建时自动检查版本
- ✅ 创建 `app-repos.conf` 配置文件
- ✅ 集成 Categraf 版本管理

**技术改进:**
- 使用 `git ls-remote` 避免克隆整个仓库
- 使用 `-V` 选项进行语义化版本排序
- 兼容旧版本 bash（使用 `tr` 替代 `^^`）
- 自动清理 sed 备份文件

## 参考资料

- [语义化版本规范](https://semver.org/lang/zh-CN/)
- [Git ls-remote 文档](https://git-scm.com/docs/git-ls-remote)
- [Categraf 项目](https://github.com/flashcatcloud/categraf)
- [AppHub 通用构建系统](./APPHUB_GENERIC_BUILD_SYSTEM.md)
