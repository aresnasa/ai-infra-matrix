# 版本管理系统使用指南

## 概述

本项目实现了统一的版本管理系统，通过 `.env` 文件集中管理所有组件版本，`build.sh` 脚本自动读取并动态渲染到各个 Dockerfile 中。

## 优势

✅ **统一管理**：所有版本配置集中在 `.env` 文件中  
✅ **易于升级**：只需修改 `.env` 文件即可升级组件  
✅ **版本追踪**：版本变更可通过 Git 历史追踪  
✅ **CI/CD 友好**：支持通过环境变量覆盖版本  
✅ **向后兼容**：Dockerfile 中提供默认值，无 `.env` 时仍可构建  

## 版本配置位置

### 主配置文件
- `.env` - 实际使用的环境变量文件
- `.env.example` - 环境变量模板（包含所有可配置项和说明）

### 配置类别

#### 1. 基础镜像版本
```bash
# Golang 基础镜像
GOLANG_VERSION=1.25
GOLANG_ALPINE_VERSION=1.25-alpine

# Node.js 基础镜像
NODE_VERSION=22
NODE_ALPINE_VERSION=22-alpine

# Python 基础镜像
PYTHON_VERSION=3.14
PYTHON_ALPINE_VERSION=3.14-alpine

# Ubuntu 基础镜像
UBUNTU_VERSION=22.04

# Nginx 基础镜像
NGINX_VERSION=stable-alpine-perl
```

#### 2. 应用组件版本
```bash
# Gitea 版本
GITEA_VERSION=1.25.1

# SaltStack 版本
SALTSTACK_VERSION=3007.8

# SLURM 版本
SLURM_VERSION=25.05.4

# AppHub 组件版本
CATEGRAF_VERSION=v0.4.22
SINGULARITY_VERSION=v4.3.4
```

#### 3. 依赖工具版本
```bash
# Python pip 版本
PIP_VERSION=24.2

# Go 模块代理
GO_PROXY=https://goproxy.cn,https://proxy.golang.org,direct

# PyPI 镜像源
PYPI_INDEX_URL=https://mirrors.aliyun.com/pypi/simple/

# npm 镜像源
NPM_REGISTRY=https://registry.npmmirror.com
```

## 使用方法

### 1. 初始化配置

如果 `.env` 文件不存在，从模板创建：
```bash
cp .env.example .env
```

### 2. 修改版本

编辑 `.env` 文件，修改需要的版本号：
```bash
# 示例：升级 Golang 版本
GOLANG_VERSION=1.26
GOLANG_ALPINE_VERSION=1.26-alpine

# 示例：升级 SLURM 版本
SLURM_VERSION=25.05.5
```

### 3. 构建镜像

版本参数会自动应用到构建过程：
```bash
# 构建单个服务
./build.sh build backend

# 构建所有服务
./build.sh build all

# 强制重建（忽略缓存）
./build.sh build backend --force
```

### 4. 验证版本

构建完成后，版本信息会作为标签保存在镜像中：
```bash
# 查看镜像标签
docker inspect ai-infra-backend:v0.3.6-dev | jq '.[0].Config.Labels'

# 查看版本信息
docker inspect ai-infra-backend:v0.3.6-dev | jq '.[0].Config.Labels | 
  with_entries(select(.key | startswith("build.") or . == "slurm.version"))'
```

## 实现原理

### 1. build.sh 工作流程

```
加载 .env 文件
    ↓
读取版本变量
    ↓
生成 --build-arg 参数
    ↓
传递给 docker build
    ↓
Dockerfile 接收 ARG
    ↓
应用到 FROM 和 RUN 指令
```

### 2. 关键函数

#### `load_env_file()`
- 加载 `.env` 文件到当前 shell 环境
- 跳过注释和空行
- 处理引号包裹的值
- 不覆盖已有环境变量

#### `get_version_build_args(service)`
- 读取所有版本相关环境变量
- 根据服务类型生成对应的 `--build-arg` 参数
- 返回完整的构建参数字符串

### 3. Dockerfile 适配

所有 Dockerfile 已适配版本参数：

```dockerfile
# 1. 在 FROM 之前声明 ARG（全局作用域）
ARG GOLANG_ALPINE_VERSION=1.25-alpine

# 2. 在 FROM 中使用变量
FROM golang:${GOLANG_ALPINE_VERSION} AS builder

# 3. 在新的构建阶段重新声明（如果需要跨阶段使用）
ARG GOLANG_ALPINE_VERSION=1.25-alpine
FROM golang:${GOLANG_ALPINE_VERSION} AS runtime

# 4. 设置默认值（确保向后兼容）
ARG PIP_VERSION=24.2
```

## 版本映射表

| 服务 | 基础镜像变量 | 应用版本变量 | 依赖工具变量 |
|------|-------------|--------------|--------------|
| backend | GOLANG_ALPINE_VERSION | - | GO_PROXY |
| frontend | NODE_ALPINE_VERSION, NGINX_VERSION | - | NPM_REGISTRY |
| gitea | - | GITEA_VERSION | - |
| jupyterhub | PYTHON_ALPINE_VERSION | - | PIP_VERSION, PYPI_INDEX_URL, NPM_REGISTRY |
| saltstack | UBUNTU_VERSION | SALTSTACK_VERSION | PIP_VERSION, PYPI_INDEX_URL |
| slurm-master | UBUNTU_VERSION | SLURM_VERSION | - |
| apphub | - | SLURM_VERSION, CATEGRAF_VERSION, SINGULARITY_VERSION | - |

## 升级示例

### 示例 1：升级 Golang 版本

```bash
# 1. 编辑 .env
vim .env

# 修改：
GOLANG_VERSION=1.26
GOLANG_ALPINE_VERSION=1.26-alpine

# 2. 重新构建 backend
./build.sh build backend --force

# 3. 验证
docker run --rm ai-infra-backend:v0.3.6-dev go version
```

### 示例 2：升级 SaltStack 版本

```bash
# 1. 编辑 .env
vim .env

# 修改：
SALTSTACK_VERSION=3008.1

# 2. 重新构建 saltstack
./build.sh build saltstack --force

# 3. 验证
docker run --rm ai-infra-saltstack:v0.3.6-dev salt --version
```

### 示例 3：修改镜像源

```bash
# 1. 编辑 .env
vim .env

# 修改为清华源：
PYPI_INDEX_URL=https://mirrors.tuna.tsinghua.edu.cn/pypi/web/simple/
NPM_REGISTRY=https://mirrors.tuna.tsinghua.edu.cn/npm/

# 2. 重新构建相关服务
./build.sh build jupyterhub frontend --force
```

## 环境变量优先级

1. **命令行环境变量**（最高优先级）
   ```bash
   GOLANG_VERSION=1.27 ./build.sh build backend
   ```

2. **shell 已导出的环境变量**
   ```bash
   export GOLANG_VERSION=1.27
   ./build.sh build backend
   ```

3. **.env 文件中的变量**
   ```bash
   # .env 文件
   GOLANG_VERSION=1.26
   ```

4. **Dockerfile 中的 ARG 默认值**（最低优先级）
   ```dockerfile
   ARG GOLANG_VERSION=1.25
   ```

## 调试技巧

### 1. 查看实际使用的版本参数

在 build.sh 中临时添加调试输出：
```bash
# 在 build_service 函数中添加
echo "DEBUG: version_args = $version_args"
```

### 2. 检查环境变量加载

```bash
# 手动加载并查看
source .env
env | grep -E "(VERSION|PROXY|REGISTRY)"
```

### 3. 验证构建参数传递

```bash
# 查看 Docker 构建日志
./build.sh build backend 2>&1 | grep "build-arg"
```

### 4. 检查镜像中的版本

```bash
# 查看运行时版本
docker run --rm ai-infra-backend:v0.3.6-dev go version
docker run --rm ai-infra-frontend:v0.3.6-dev node --version
docker run --rm ai-infra-jupyterhub:v0.3.6-dev python --version
```

## 最佳实践

1. **版本固定**：生产环境使用精确版本号，避免 `latest` 标签
2. **版本测试**：升级前在测试环境验证新版本兼容性
3. **版本记录**：在 Git commit 中记录版本变更原因
4. **回退准备**：保留旧版本镜像，以便快速回退
5. **文档同步**：更新版本时同步更新相关文档

## 故障排查

### 问题 1：构建时未使用 .env 中的版本

**可能原因**：
- `.env` 文件不存在
- 环境变量名称拼写错误
- build.sh 未正确加载环境文件

**解决方法**：
```bash
# 检查 .env 文件是否存在
ls -la .env

# 验证环境变量内容
cat .env | grep VERSION

# 手动测试加载
source .env && echo $GOLANG_VERSION
```

### 问题 2：ARG 变量在 FROM 中不可用

**可能原因**：
- ARG 声明在 FROM 之后
- 跨构建阶段未重新声明

**解决方法**：
```dockerfile
# 正确做法：在 FROM 之前声明
ARG GOLANG_ALPINE_VERSION=1.25-alpine
FROM golang:${GOLANG_ALPINE_VERSION}

# 跨阶段使用需要重新声明
ARG GOLANG_ALPINE_VERSION=1.25-alpine
FROM golang:${GOLANG_ALPINE_VERSION} AS stage2
```

### 问题 3：版本参数未传递到 Docker build

**可能原因**：
- build.sh 中 `get_version_build_args` 未被调用
- 构建命令缺少 `$version_args` 变量

**解决方法**：
```bash
# 检查 build_service 函数
grep "version_args" build.sh

# 确保构建命令包含版本参数
docker build ... $version_args ...
```

## 扩展新组件

添加新组件版本管理的步骤：

### 1. 在 .env.example 中添加版本变量

```bash
# ==================== 新组件版本配置 ====================
# 新组件名称
NEW_COMPONENT_VERSION=1.0.0
```

### 2. 在 get_version_build_args 中添加参数

```bash
# 在 build.sh 的 get_version_build_args 函数中添加
[[ -n "${NEW_COMPONENT_VERSION:-}" ]] && \
    build_args+=" --build-arg NEW_COMPONENT_VERSION=${NEW_COMPONENT_VERSION}"
```

### 3. 在 Dockerfile 中使用 ARG

```dockerfile
ARG NEW_COMPONENT_VERSION=1.0.0
RUN install-component --version=${NEW_COMPONENT_VERSION}
```

### 4. 测试构建

```bash
# 设置版本
echo "NEW_COMPONENT_VERSION=1.0.1" >> .env

# 构建测试
./build.sh build new-component
```

## 参考资料

- [Docker ARG and ENV](https://docs.docker.com/engine/reference/builder/#arg)
- [Build.sh 源码](../build.sh)
- [环境变量模板](.env.example)
- [各服务 Dockerfile](../src/)

## 更新历史

- **2025-01-06**: 初始版本，实现统一版本管理系统
- 支持基础镜像、应用组件、依赖工具的版本管理
- 所有主要服务 Dockerfile 已适配

---

**维护者**: AI Infrastructure Team  
**最后更新**: 2025年1月6日
