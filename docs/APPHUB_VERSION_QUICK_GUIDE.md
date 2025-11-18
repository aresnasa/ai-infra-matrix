# AppHub 动态版本管理 - 快速使用指南

## 新增命令

### 1. 查看应用版本

```bash
./build.sh apphub-versions
```

输出所有 AppHub 应用的最新版本。

### 2. 更新版本号

```bash
./build.sh apphub-update-versions
```

自动更新 Dockerfile 中的版本号到最新版本。

### 3. 构建时自动更新

```bash
./build.sh build apphub --force
```

构建 AppHub 时会自动检查并更新版本（如果有新版本）。

## 添加新应用

### 步骤 1: 编辑配置文件

编辑 `src/apphub/app-repos.conf`，添加新应用：

```
# 格式: APP_NAME|GIT_REPO_URL|VERSION_PREFIX
categraf|https://github.com/flashcatcloud/categraf.git|v
nightingale|https://github.com/ccfos/nightingale.git|v
```

### 步骤 2: 在 Dockerfile 中添加 ARG

在 `src/apphub/Dockerfile` 中添加对应的 ARG 变量：

```dockerfile
ARG NIGHTINGALE_VERSION=v6.0.0
ARG NIGHTINGALE_REPO=https://github.com/ccfos/nightingale.git
```

### 步骤 3: 使用版本变量

在构建脚本中使用这些变量：

```dockerfile
RUN git clone --depth 1 --branch "${NIGHTINGALE_VERSION}" "${NIGHTINGALE_REPO}"
```

### 步骤 4: 验证

```bash
# 获取最新版本
./build.sh apphub-versions

# 更新 Dockerfile
./build.sh apphub-update-versions

# 构建测试
./build.sh build apphub --force
```

## 配置文件格式

`src/apphub/app-repos.conf`:

```
# 注释行
APP_NAME|GIT_REPO_URL|VERSION_PREFIX

# 示例
categraf|https://github.com/flashcatcloud/categraf.git|v
```

字段说明：
- **APP_NAME**: 应用名称（小写，对应 scripts/<app-name> 目录）
- **GIT_REPO_URL**: Git 仓库地址
- **VERSION_PREFIX**: 版本标签前缀（如 `v` 或留空）

## 常见问题

### Q: 如何锁定特定版本？

A: 在配置文件中注释掉该应用，手动在 Dockerfile 中设置版本：

```dockerfile
# Categraf 版本锁定在 v0.4.22
ARG CATEGRAF_VERSION=v0.4.22
```

### Q: 支持哪些版本格式？

A: 支持语义化版本格式：
- ✅ `v0.4.22`
- ✅ `0.4.22`
- ✅ `v1.0.0-beta.1`
- ❌ `latest`
- ❌ `main`

### Q: 如何查看版本更新历史？

A: 查看 Git 提交历史：

```bash
git log --oneline src/apphub/Dockerfile | grep VERSION
```

## 详细文档

完整文档请参考：
- [AppHub 版本管理详细文档](./APPHUB_VERSION_MANAGEMENT.md)
- [AppHub 动态版本实现报告](./APPHUB_DYNAMIC_VERSION_DONE.md)
