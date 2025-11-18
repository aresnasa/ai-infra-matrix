# build.sh AppHub 新脚本结构适配报告

## 📋 当前状态

### AppHub Dockerfile 已完成重构
- ✅ 新脚本结构已实现：`scripts/categraf/` 和 `scripts/slurm/`
- ✅ 通用构建脚本：`scripts/build-app.sh`
- ✅ Dockerfile Stage 4 已使用新的 COPY 模式
- ✅ 构建命令已简化为：`/scripts/build-app.sh categraf`

### build.sh 兼容性分析

#### 1. 常规构建流程（✅ 无需修改）
```bash
./build.sh build apphub v0.3.8
```

**原因**：
- `build.sh` 的 `build_service()` 函数使用通用逻辑
- 直接调用 `docker build -f src/apphub/Dockerfile src/apphub`
- AppHub Dockerfile 内部已处理所有新脚本逻辑
- 构建上下文正确包含 `scripts/` 目录

#### 2. copy_slurm_packages_to_apphub() 函数（⚠️ 已废弃）
**位置**：`build.sh:2506`
**状态**：仅定义，从未被调用
**建议**：保留作为后备方案，但不影响当前构建流程

#### 3. build-all 流程（✅ 自动兼容）
```bash
./build.sh build-all v0.3.8
```

**构建顺序**：
1. 预拉取依赖镜像（包括 `golang:1.23-alpine`）
2. 按依赖顺序构建各服务
3. AppHub 将在适当时机构建（包含所有多阶段构建）

## 🔧 关键适配点

### 1. Dockerfile 构建上下文
**当前配置**：
```dockerfile
# Stage 4: Categraf Builder
COPY scripts/build-app.sh /scripts/build-app.sh
COPY scripts/categraf/ /scripts/categraf/

RUN /scripts/build-app.sh categraf
```

**build.sh 构建命令**：
```bash
docker build -f src/apphub/Dockerfile -t ai-infra-apphub:v0.3.8 src/apphub
```

**构建上下文路径**：`src/apphub/`
**关键文件包含检查**：
- ✅ `src/apphub/scripts/build-app.sh`
- ✅ `src/apphub/scripts/categraf/categraf-build.sh`
- ✅ `src/apphub/scripts/categraf/*.sh`（install/uninstall）
- ✅ `src/apphub/scripts/slurm/*.sh`

### 2. 环境变量传递
**Dockerfile ARG 定义**：
```dockerfile
ARG CATEGRAF_VERSION=v0.3.90
ARG CATEGRAF_REPO=https://github.com/flashcatcloud/categraf.git
```

**构建时覆盖**（可选）：
```bash
docker build \
  --build-arg CATEGRAF_VERSION=v0.3.91 \
  --build-arg CATEGRAF_REPO=https://gitee.com/flashcat/categraf.git \
  -f src/apphub/Dockerfile \
  -t ai-infra-apphub:custom \
  src/apphub
```

### 3. 多架构支持
**当前支持**：
- AMD64: `categraf-v0.3.90-linux-amd64.tar.gz`
- ARM64: `categraf-v0.3.90-linux-arm64.tar.gz`

**构建平台检测**：自动（在 `categraf-build.sh` 中通过 `uname -m` 检测）

## ✅ 验证清单

### 构建前检查
```bash
# 1. 验证脚本目录结构
ls -R src/apphub/scripts/
# 预期输出：
# scripts/:
# build-app.sh  categraf/  slurm/
# 
# scripts/categraf/:
# categraf-build.sh  install.sh  uninstall.sh  systemd.service  readme.md
#
# scripts/slurm/:
# install.sh  uninstall.sh

# 2. 检查脚本权限
find src/apphub/scripts -name "*.sh" -exec ls -lh {} \;
# 预期：所有 .sh 文件为 -rwxr-xr-x

# 3. 验证 Dockerfile 语法
docker build --dry-run -f src/apphub/Dockerfile src/apphub 2>&1 | head -20
```

### 构建测试
```bash
# 1. 单独构建 AppHub
./build.sh build apphub v0.3.8

# 2. 验证 Categraf 包生成
docker run --rm ai-infra-apphub:v0.3.8 ls -lh /usr/share/nginx/html/pkgs/categraf/

# 预期输出：
# categraf-latest-linux-amd64.tar.gz -> categraf-v0.3.90-linux-amd64.tar.gz
# categraf-latest-linux-arm64.tar.gz -> categraf-v0.3.90-linux-arm64.tar.gz
# categraf-v0.3.90-linux-amd64.tar.gz
# categraf-v0.3.90-linux-arm64.tar.gz
# install.sh
# readme.md
# uninstall.sh

# 3. 测试包下载
docker run -d --name test-apphub -p 8888:80 ai-infra-apphub:v0.3.8
curl -I http://localhost:8888/pkgs/categraf/categraf-latest-linux-amd64.tar.gz
docker rm -f test-apphub
```

## 📊 构建性能优化

### 1. Docker 层缓存利用
**优化点**：
- ✅ 基础镜像层（Ubuntu, Rocky, Alpine, Golang）会被缓存
- ✅ 依赖安装层（apt/dnf/apk install）会被缓存
- ⚠️ Categraf 克隆和编译层每次重建（因为可能有新版本）

**改进建议**（可选）：
```dockerfile
# 在 categraf-builder 阶段添加版本标签缓存
LABEL categraf.version="${CATEGRAF_VERSION}"
```

### 2. 并行构建
**当前行为**：
- Docker BuildKit 自动并行执行独立的构建阶段
- Stage 1-4 可以并行构建（无依赖关系）
- 最终镜像（Stage 5）依赖所有前置阶段

**build.sh 优化**（已支持）：
```bash
# 启用 BuildKit（默认）
export DOCKER_BUILDKIT=1

# 强制重建（清除缓存）
./build.sh build apphub v0.3.8 --force
```

## 🛠️ 故障排查

### 问题 1: "COPY scripts/categraf/ failed"
**原因**：构建上下文不包含 scripts 目录
**解决方案**：
```bash
# 检查构建上下文
ls src/apphub/scripts/categraf/

# 如果目录不存在，检查当前目录
pwd  # 应该在项目根目录

# 确保从正确位置运行 build.sh
cd /path/to/ai-infra-matrix
./build.sh build apphub v0.3.8
```

### 问题 2: "categraf-build.sh: not found"
**原因**：脚本权限或路径问题
**解决方案**：
```bash
# 修复权限
chmod +x src/apphub/scripts/build-app.sh
chmod +x src/apphub/scripts/categraf/*.sh

# 验证脚本存在
ls -la src/apphub/scripts/categraf/categraf-build.sh
```

### 问题 3: "git clone failed" (Categraf 下载)
**原因**：网络问题或 GitHub 访问限制
**解决方案 1**：使用镜像仓库
```bash
docker build \
  --build-arg CATEGRAF_REPO=https://gitee.com/flashcat/categraf.git \
  -f src/apphub/Dockerfile \
  -t ai-infra-apphub:v0.3.8 \
  src/apphub
```

**解决方案 2**：预下载 Categraf 源码
```bash
# 在 src/apphub/ 目录下创建 .categraf-cache/
mkdir -p src/apphub/.categraf-cache
cd src/apphub/.categraf-cache
git clone --depth=1 --branch=v0.3.90 https://github.com/flashcatcloud/categraf.git

# 修改 categraf-build.sh 使用本地源码（需要更新脚本逻辑）
```

## 📝 build.sh 集成建议

### 当前状态：无需修改 ✅
**原因**：
1. `build_service()` 函数已通用化，无需为 AppHub 添加特殊逻辑
2. AppHub 的所有复杂性都封装在 Dockerfile 内部
3. 构建上下文路径配置正确（`src/apphub/`）

### 可选增强（未来考虑）

#### 1. 添加 AppHub 包验证步骤
```bash
# 在 build.sh 的 build_service() 函数后添加
verify_apphub_packages() {
    local tag="${1:-$DEFAULT_IMAGE_TAG}"
    local image="ai-infra-apphub:${tag}"
    
    print_info "验证 AppHub 包完整性..."
    
    # 验证 Categraf 包
    local categraf_count=$(docker run --rm "$image" ls /usr/share/nginx/html/pkgs/categraf/*.tar.gz 2>/dev/null | wc -l)
    if [[ $categraf_count -ge 2 ]]; then
        print_success "✓ Categraf 包完整: $categraf_count 个架构"
    else
        print_warning "⚠️ Categraf 包不完整: 只找到 $categraf_count 个包"
    fi
    
    # 验证 SLURM 包
    local slurm_deb_count=$(docker run --rm "$image" ls /usr/share/nginx/html/pkgs/slurm-deb/*.deb 2>/dev/null | wc -l || echo 0)
    local slurm_rpm_count=$(docker run --rm "$image" ls /usr/share/nginx/html/pkgs/slurm-rpm/*.rpm 2>/dev/null | wc -l || echo 0)
    print_info "✓ SLURM deb 包: $slurm_deb_count 个"
    print_info "✓ SLURM rpm 包: $slurm_rpm_count 个"
}

# 在构建成功后调用
if build_service "apphub" "$tag" "$registry"; then
    verify_apphub_packages "$tag"
fi
```

#### 2. 添加快速重建选项（仅重建 Categraf 阶段）
```bash
rebuild_categraf() {
    local tag="${1:-$DEFAULT_IMAGE_TAG}"
    
    print_info "快速重建 Categraf（复用其他阶段缓存）..."
    
    docker build \
        --target categraf-builder \
        --build-arg CATEGRAF_VERSION=v0.3.90 \
        -t categraf-builder-temp:latest \
        -f src/apphub/Dockerfile \
        src/apphub
    
    # 然后重新构建最终镜像
    docker build \
        -t "ai-infra-apphub:${tag}" \
        -f src/apphub/Dockerfile \
        src/apphub
}
```

## 🎯 总结

### 当前适配状态：✅ 完全兼容

**无需修改 build.sh 的原因**：
1. ✅ Dockerfile 已完成新脚本结构重构
2. ✅ 构建上下文配置正确
3. ✅ build.sh 的通用构建逻辑已覆盖 AppHub
4. ✅ 所有特殊逻辑封装在 Dockerfile 内部

### 推荐使用方式

**标准构建**：
```bash
# 构建单个 AppHub 服务
./build.sh build apphub v0.3.8

# 构建所有服务（包括 AppHub）
./build.sh build-all v0.3.8
```

**自定义 Categraf 版本**：
```bash
# 方法1: 修改 Dockerfile 中的 ARG CATEGRAF_VERSION
sed -i 's/ARG CATEGRAF_VERSION=v0.3.90/ARG CATEGRAF_VERSION=v0.3.91/' src/apphub/Dockerfile

# 方法2: 使用 --build-arg（需要直接调用 docker build）
docker build \
  --build-arg CATEGRAF_VERSION=v0.3.91 \
  -t ai-infra-apphub:custom \
  -f src/apphub/Dockerfile \
  src/apphub
```

**验证构建结果**：
```bash
# 1. 检查镜像大小
docker images | grep ai-infra-apphub

# 2. 查看包列表
docker run --rm ai-infra-apphub:v0.3.8 find /usr/share/nginx/html/pkgs -type f -name "*.tar.gz"

# 3. 启动测试服务器
docker run -d -p 8080:80 --name apphub-test ai-infra-apphub:v0.3.8
curl http://localhost:8080/pkgs/categraf/
docker rm -f apphub-test
```

---

**文档版本**：1.0  
**最后更新**：2025-10-24  
**状态**：✅ build.sh 无需修改，直接可用
