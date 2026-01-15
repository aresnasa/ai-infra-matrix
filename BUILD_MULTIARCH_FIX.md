# 多架构构建和 Manifest 支持修复方案

## 问题分析

### 当前问题
1. `build.sh all --platform=amd64,arm64` 被调用时，只在本地构建
2. 缺少 Docker manifest 创建，无法跨架构访问镜像
3. arm64 架构的镜像未被正确标记和推送

### 根本原因
- `build.sh` 中的 `build_component()` 和 `build_component_for_platform()` 没有利用 Docker buildx 的多架构能力
- 缺少 manifest list 的创建逻辑
- 本地构建时，非原生架构的镜像没有被正确保存

## 修复步骤

### 1. 启用 buildx 构建器（前置条件）

```bash
# 检查 buildx 是否可用
docker buildx ls

# 如果不可用，创建新的 buildx builder
docker buildx create --name ai-infra-builder --use

# 启用 qemu 支持（用于模拟其他架构）
docker run --rm --privileged tonistiigi/binfmt --install all
```

### 2. 执行多架构构建

```bash
# 方式 1：使用 build-multiarch 命令（推荐）
./build.sh build-multiarch "linux/amd64,linux/arm64"

# 方式 2：使用 build-platform 为特定架构构建
./build.sh build-platform linux/amd64
./build.sh build-platform linux/arm64

# 方式 3：推送到仓库并创建 manifest
./build.sh build-multiarch "linux/amd64,linux/arm64"
./build.sh push-all <registry/project> v0.3.8
```

### 3. 使用 Docker Buildx 直接构建

```bash
# 直接用 buildx 构建并推送（创建 manifest）
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --push \
  -t <registry>/ai-infra-backend:v0.3.8 \
  -f src/backend/Dockerfile \
  .

# 验证 manifest
docker buildx inspect --bootstrap
docker manifest inspect <registry>/ai-infra-backend:v0.3.8
```

## 关键修复点

### 修复 1：`build_component_for_platform()` 函数

**当前问题**：单个平台构建没有利用 buildx 的多架构能力

**解决方案**：
- 对于单个架构：使用 `--load` 直接加载到本地
- 对于多个架构：使用 `--push` 推送到仓库并自动创建 manifest

```bash
# 修改逻辑：
if [[ "$platform_count" == "1" ]]; then
    cmd+=("--load")  # 单架构：加载到本地
else
    cmd+=("--push")  # 多架构：推送到仓库
    cmd+=("-t" "$registry_image")  # 需要指定完整的仓库地址
fi
```

### 修复 2：添加 Manifest 创建功能

```bash
# 创建多架构 manifest
create_multiarch_manifest() {
    local image_base="$1"
    local tag="$2"
    local arches="${3:-amd64,arm64}"
    
    # 推送各架构镜像
    for arch in $(echo $arches | tr ',' ' '); do
        docker buildx build --platform linux/$arch -t "${image_base}-${arch}:${tag}" --push .
    done
    
    # 创建 manifest
    docker manifest create "${image_base}:${tag}" \
        "${image_base}-amd64:${tag}" \
        "${image_base}-arm64:${tag}"
    
    # 推送 manifest
    docker manifest push "${image_base}:${tag}"
}
```

### 修复 3：支持 `all --platform=amd64,arm64` 命令

当前 `build.sh all --platform=amd64,arm64` 被解析为普通 `all` 命令，需要：

1. **添加参数解析**：
```bash
# 在 main command parser 中添加
if [[ "$cmd" == "all" ]]; then
    # 提取 --platform 参数
    local platforms=""
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --platform=*) platforms="${1#--platform=}" ;;
            --force) FORCE_BUILD=true ;;
        esac
        shift
    done
    
    if [[ -n "$platforms" ]]; then
        build_all_multiplatform "$platforms"
    else
        build_all
    fi
fi
```

## 实施建议

### 短期方案（推荐）
使用现有的 `build-multiarch` 命令：
```bash
# 1. 构建多架构镜像（保存为 OCI tarball）
./build.sh build-multiarch "amd64,arm64"

# 2. 推送到仓库
./build.sh push-all <registry/project> v0.3.8

# 3. 验证 manifest
docker manifest inspect <registry>/ai-infra-backend:v0.3.8
```

### 长期方案
修改 `build.sh` 支持 `all --platform=amd64,arm64` 语法，完全兼容原始意图。

## 测试验证

```bash
# 1. 验证本地 buildx builder
docker buildx ls

# 2. 构建单个组件
./build.sh build-platform linux/amd64

# 3. 检查构建结果
docker images | grep ai-infra

# 4. 推送到仓库
./build.sh push-all harbor.example.com/ai-infra v0.3.8

# 5. 验证 manifest
docker pull harbor.example.com/ai-infra/ai-infra-backend:v0.3.8
```

## 常见问题

### Q1: 为什么 arm64 镜像没有构建？
**A**: `build_component()` 只支持本地架构，需要使用 `build_component_for_platform()` 或 buildx

### Q2: 如何支持离线环境？
**A**: 
1. 在有网络的机器上构建：`./build.sh build-multiarch amd64,arm64`
2. 导出 OCI tar 包：存储在 `./multiarch-images/`
3. 在目标机器上导入：`docker load < image.tar`

### Q3: 本地没有 arm64 环境，如何构建？
**A**: 使用 QEMU 仿真（自动安装）：
```bash
docker run --rm --privileged tonistiigi/binfmt --install all
./build.sh build-multiarch "amd64,arm64"  # 会使用 QEMU 仿真 arm64
```

## 参考文档

- Docker BuildX: https://docs.docker.com/build/architecture/
- Multi-arch Images: https://docs.docker.com/build/building/multi-platform/
- Manifest Lists: https://docs.docker.com/docker-hub/multi-arch/
