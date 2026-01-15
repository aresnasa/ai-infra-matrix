# build.sh 多架构构建问题分析

## 问题总结

根据对 `build.sh` 代码的分析，发现了多架构构建中存在的关键问题。

## 核心发现

### 1. **当前代码结构（已有功能）**

#### ✅ 已实现的多架构支持：
- `build_all_multiplatform()` 函数（第5623-5900行）
  - 接收平台参数：`amd64,arm64`
  - 规范化平台格式：`linux/amd64`, `linux/arm64`
  - 循环为每个平台构建所有服务
  - 已支持 AppHub 跨平台启动

- `build_component_for_platform()` 函数（第5920-6130行）
  - 单平台单组件构建
  - 使用 Docker Buildx 多架构构建
  - 应用架构后缀标签：`-amd64`, `-arm64`

#### ✅ 命令行参数处理（第7650-7730行）
```bash
--platform=*)
    BUILD_PLATFORMS="${arg#*=}"
    ;;
```
参数正确被提取并保存到 `BUILD_PLATFORMS` 变量

#### ✅ main 命令分发（第7895-7910行）
```bash
case "$COMMAND" in
    build-all|all)
        if [[ -n "$BUILD_PLATFORMS" ]]; then
            # Multi-platform build mode
            build_all_multiplatform "$BUILD_PLATFORMS" "${force_flag}"
        else
            # Standard single-platform build
            build_all "${force_flag}"
        fi
        ;;
```

**调用逻辑正确**：`build.sh all --platform=amd64,arm64` 会正确调用 `build_all_multiplatform`

---

### 2. **实际问题（从导出日志）**

导出日志显示的问题：
1. **12个组件中只有3个构建了amd64版本**
   - ✅ apphub (3.9G)
   - ✅ slurm-master (2.7G)  
   - ✅ test-containers (191M)
   - ❌ 缺失：gitea, nginx, saltstack, backend, frontend, jupyterhub, nightingale, prometheus, singleuser

2. **arm64 版本全部缺失**（0/12）

3. **导出日志中的 "Image not found" 警告**说明镜像根本没被构建

---

### 3. **根本原因分析**

通过代码分析，找到了可能导致大部分组件未构建的原因：

#### 问题A：`build_component()` 仅支持本地架构

**问题位置**：第4742行

```bash
build_component() {
    local component="$1"
    # ... 
    # 调用 build_component_for_platform 传递本地平台
    build_component_for_platform "$component" "$native_platform"
}
```

**问题**：`build_component()` 是被 `build_all()` 使用的（单架构版本），**但不支持多平台参数**

**影响**：
- `build_all()` 函数在第6180-6190行循环调用 `build_component`
- 这些调用**不会尊重 `BUILD_PLATFORMS` 变量**
- 只构建本地架构

---

#### 问题B：`build_all_multiplatform()` 正确实现，但可能在其他地方有问题

**已验证的正确部分**：
- ✅ Phase 0: 模板渲染 ✓
- ✅ Phase 0.5: 预拉base镜像 ✓  
- ✅ Phase 1: 依赖服务处理 ✓
- ✅ Phase 2: Foundation 服务循环构建 ✓
- ✅ Phase 3: AppHub 启动 ✓
- ✅ Phase 4: Dependent 服务循环构建 ✓

---

#### 问题C：未找到 Docker Manifest 创建逻辑

**关键发现**：grep搜索中没有找到任何 `docker manifest create` 或 `docker manifest push` 的调用

```bash
# 搜索结果显示"manifest"仅出现在离线导出的元数据文件生成中
# 没有实际的 docker manifest 操作
```

这意味着：
- 即使两个架构的镜像都构建了，也没有创建统一的 manifest list
- 推送镜像到仓库时，也不支持多架构

---

### 4. **为什么导出日志显示无法找到镜像**

导出脚本期望的标签格式：
- 对于本地镜像：`ai-infra-backend:v0.3.8`
- 对于跨架构镜像：`ai-infra-backend:v0.3.8-amd64`, `ai-infra-backend:v0.3.8-arm64`

但如果：
1. `build.sh all --platform=amd64,arm64` 实际调用了单平台 `build_all()`（如果有 bug）
2. 或者这些组件的 build_component_for_platform() 执行失败但没有正确报错
3. 那么镜像就不会被标记，导出时找不到

---

## 完整的多架构构建流程（应该工作的方式）

```
build.sh all --platform=amd64,arm64
        ↓
命令行参数解析 (line 7650-7730)
        ↓
BUILD_PLATFORMS="amd64,arm64"
        ↓
case $COMMAND (line 7895)
        ↓
build_all_multiplatform "amd64,arm64"
        ↓
循环处理两个平台:
  for platform in "linux/amd64" "linux/arm64"
    → Phase 0: 渲染模板
    → Phase 1: 处理依赖服务
    → Phase 2: 构建 Foundation 服务
       for service in foundation_services
         → build_component_for_platform $service $platform
    → Phase 3: 启动 AppHub
    → Phase 4: 构建 Dependent 服务
       for service in dependent_services
         → build_component_for_platform $service $platform
        ↓
所有 12 个组件 × 2 个架构 = 24 个镜像
        ↓
[缺失] 创建 Docker Manifest
        ↓
镜像推送到仓库
```

---

## 可能的失败点

### 失败点1：服务发现（discover_services）
- 可能某些服务没有被正确识别为 FOUNDATION_SERVICES 或 DEPENDENT_SERVICES
- 导致它们完全跳过构建

### 失败点2：build_component_for_platform() 中的错误处理
- 如果构建失败，错误可能被吞掉
- 需要检查错误日志

### 失败点3：Docker Buildx 构建器问题
- 可能 multiarch-builder 创建失败
- 可能权限问题导致 arm64 构建失败

### 失败点4：QEMU 支持缺失
- 如果在 amd64 机器上构建 arm64，需要 QEMU
- 可能没有被正确安装

---

## 修复方案

### 立即修复（必须）

#### 1. **添加 Docker Manifest 支持**

在导出前添加 manifest 创建逻辑：

```bash
# 在 build_all_multiplatform() 完成后添加
create_multiarch_manifest() {
    local components=("${@}")
    local tag="${IMAGE_TAG:-latest}"
    
    for component in "${components[@]}"; do
        local image_base="ai-infra-${component}"
        
        # 检查amd64和arm64镜像是否都存在
        if docker image inspect "${image_base}:${tag}-amd64" >/dev/null 2>&1 && \
           docker image inspect "${image_base}:${tag}-arm64" >/dev/null 2>&1; then
            
            # 创建 manifest list
            docker manifest create "${image_base}:${tag}" \
                "${image_base}:${tag}-amd64" \
                "${image_base}:${tag}-arm64"
            
            # 设置架构信息
            docker manifest annotate "${image_base}:${tag}" \
                "${image_base}:${tag}-amd64" --os linux --arch amd64
            docker manifest annotate "${image_base}:${tag}" \
                "${image_base}:${tag}-arm64" --os linux --arch arm64
            
            log_info "✓ Created manifest for $image_base:$tag"
        else
            log_warn "Missing architectures for $image_base:$tag"
            if ! docker image inspect "${image_base}:${tag}-amd64" >/dev/null 2>&1; then
                log_warn "  - Missing amd64 image"
            fi
            if ! docker image inspect "${image_base}:${tag}-arm64" >/dev/null 2>&1; then
                log_warn "  - Missing arm64 image"
            fi
        fi
    done
}
```

#### 2. **改进错误报告**

在 `build_component_for_platform()` 中添加更详细的日志：

```bash
if "${cmd[@]}"; then
    log_info "✓ Built: $full_image_name"
    save_service_build_info "$component" "$tag" "$build_id" "$service_hash"
else
    log_error "✗ Failed to build: $full_image_name"
    log_error "  Platform: $platform"
    log_error "  Component: $component"
    log_error "  Command: ${cmd[*]}"
    # 返回错误，不是默默继续
    return 1
fi
```

#### 3. **验证 QEMU 支持**

在多架构构建前确保 QEMU 已安装：

```bash
_ensure_qemu_for_multiarch() {
    if [[ "$BUILD_PLATFORMS" == *"arm64"* ]] && [[ "$(uname -m)" == "x86_64" ]]; then
        log_info "Enabling QEMU for arm64 cross-compilation..."
        docker run --rm --privileged tonistiigi/binfmt --install arm64
    fi
}
```

#### 4. **添加构建验证**

在导出前验证所有镜像都存在：

```bash
verify_multiarch_images() {
    local components=("$@")
    local tag="${IMAGE_TAG:-latest}"
    local missing=0
    
    for component in "${components[@]}"; do
        local amd64_image="ai-infra-${component}:${tag}-amd64"
        local arm64_image="ai-infra-${component}:${tag}-arm64"
        
        if ! docker image inspect "$amd64_image" >/dev/null 2>&1; then
            log_error "Missing: $amd64_image"
            missing=$((missing + 1))
        fi
        
        if ! docker image inspect "$arm64_image" >/dev/null 2>&1; then
            log_error "Missing: $arm64_image"
            missing=$((missing + 1))
        fi
    done
    
    if [[ $missing -gt 0 ]]; then
        log_error "⚠️  $missing architecture images missing!"
        return 1
    fi
    
    log_info "✓ All multiarch images present"
    return 0
}
```

---

### 测试步骤（诊断问题）

```bash
# Step 1: 启用 QEMU (如果在 amd64 机器上)
docker run --rm --privileged tonistiigi/binfmt --install arm64

# Step 2: 查看 buildx 设置
docker buildx ls

# Step 3: 运行多架构构建并记录详细日志
./build.sh all --platform=amd64,arm64 2>&1 | tee build.log

# Step 4: 检查构建的镜像
docker images | grep ai-infra

# Step 5: 检查是否有错误
grep -i "error\|fail\|not found" build.log

# Step 6: 手动验证特定镜像
docker image inspect ai-infra-backend:v0.3.8-amd64
docker image inspect ai-infra-backend:v0.3.8-arm64
```

---

## 推荐方案

### 短期修复（1-2小时）
1. 添加 Docker Manifest 创建支持
2. 改进错误报告机制
3. 添加镜像存在性验证
4. 编写测试脚本验证多架构

### 长期方案（使用 Docker Buildx 原生支持）
```bash
# 使用 buildx 的 --push 和多架构构建
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --push \
  -t registry.example.com/ai-infra/backend:v0.3.8 \
  -f src/backend/Dockerfile \
  .
```

这样可以：
- ✅ 同时为两个架构构建
- ✅ 自动创建 manifest list
- ✅ 一次性推送所有架构

---

## 代码位置索引

| 问题 | 文件 | 行号 | 现象 |
|-----|------|------|------|
| 参数解析 | build.sh | 7670 | `BUILD_PLATFORMS="${arg#*=}"` |
| 命令分发 | build.sh | 7895 | `case "$COMMAND" in ... build-all\|all)` |
| 多架构实现 | build.sh | 5623 | `build_all_multiplatform()` |
| 单架构实现 | build.sh | 4742 | `build_component()` |
| 单平台单组件 | build.sh | 5920 | `build_component_for_platform()` |
| 缺失manifest | build.sh | 全文 | 无 `docker manifest create` |
| 导出函数 | build.sh | 7015 | `_export_image_for_platform()` |

