# 任务34和35实现计划

## 任务34：合并 apphub 和 slurm-build

### 目标
将 slurm-build 的构建功能整合到 apphub 中，使 apphub 成为统一的工具链镜像管理器。

### 实现方案

#### 方案A：多阶段构建（推荐）
```dockerfile
# Stage 1: 构建 Slurm deb 包
FROM ubuntu:22.04 AS slurm-builder
# ... slurm-build 的所有构建步骤 ...
RUN debuild -b -uc -us
RUN find /home/builder/build -name '*.deb' -exec mv {} /out/ \;

# Stage 2: Apphub 服务
FROM nginx:stable
# 复制构建好的 deb 包
COPY --from=slurm-builder /out/*.deb /usr/share/nginx/html/pkgs/slurm-deb/
# ... apphub 的其他配置 ...
```

**优点**：
- 一个镜像同时包含构建工具和服务
- 构建产物直接集成，无需复制
- 镜像体积可控（最终镜像不包含构建工具）

**缺点**：
- 构建时间较长
- 修改 Slurm 需要重建整个镜像

#### 方案B：保持分离，优化 copy 流程（当前）
保持 slurm-build 和 apphub 分离，修复 `copy_slurm_packages_to_apphub` 函数。

**当前问题**：
- `docker cp` 成功复制文件，但检查失败
- 需要修复容器启动和文件检查逻辑

### 推荐实施步骤

1. **短期修复**（任务34.1）：
   - 修复 `copy_slurm_packages_to_apphub` 函数
   - 确保 deb 包正确复制到 apphub

2. **长期优化**（任务34.2）：
   - 采用方案A：多阶段构建
   - 创建统一的 apphub 镜像

## 任务35：依赖镜像预拉取

### 当前问题分析

```
Error: failed to resolve reference "docker.io/minio/minio:latest": 
failed to authorize: failed to fetch oauth token: 
Post "https://auth.docker.io/token": EOF
```

**根本原因**：
1. Docker Hub 认证问题（网络/认证）
2. build-all 没有预先拉取依赖镜像
3. 没有处理内部 Harbor 镜像源

### 实现方案

#### 1. 提取所有依赖镜像
```bash
extract_all_dependencies() {
    local services="${1:-$(get_all_services)}"
    local images=()
    
    for service in $services; do
        local dockerfile=$(get_dockerfile_path "$service")
        if [ -f "$dockerfile" ]; then
            # 提取 FROM 指令中的镜像
            local base_images=$(grep -E '^FROM ' "$dockerfile" | awk '{print $2}' | grep -v ' AS ')
            images+=($base_images)
        fi
    done
    
    # 去重
    printf '%s\n' "${images[@]}" | sort -u
}
```

#### 2. 智能拉取策略
```bash
prefetch_dependencies() {
    local registry="$1"
    local images=$(extract_all_dependencies)
    
    for image in $images; do
        if docker image inspect "$image" >/dev/null 2>&1; then
            print_info "✓ 镜像已存在: $image"
            continue
        fi
        
        if [ -n "$registry" ]; then
            # 尝试从内部 Harbor 拉取
            local internal_image="${registry}/${image}"
            if docker pull "$internal_image" 2>/dev/null; then
                docker tag "$internal_image" "$image"
                continue
            fi
        fi
        
        # 从 Docker Hub 拉取
        docker pull "$image" || print_warning "拉取失败: $image"
    done
}
```

#### 3. 集成到 build-all
```bash
build_all_services() {
    local tag="${1:-$DEFAULT_IMAGE_TAG}"
    local registry="${2:-}"
    
    # 步骤0：预拉取依赖镜像
    print_info "步骤 0/5: 预拉取依赖镜像"
    prefetch_dependencies "$registry"
    
    # 步骤1：检查构建状态
    print_info "步骤 1/5: 检查构建状态"
    # ... 现有逻辑 ...
}
```

### Docker Hub 认证问题解决

#### 方案1：使用镜像加速器
```bash
# 配置 Docker daemon.json
{
  "registry-mirrors": [
    "https://docker.mirrors.ustc.edu.cn",
    "https://hub-mirror.c.163.com"
  ]
}
```

#### 方案2：使用内部 Harbor
```bash
# 所有依赖镜像预先推送到 Harbor
deps-push aiharbor.msxf.local/aihpc v0.3.6-dev
```

#### 方案3：离线镜像包
```bash
# 导出依赖镜像
docker save redis:7-alpine minio/minio:latest > deps.tar
# 导入
docker load < deps.tar
```

## 实施优先级

### 高优先级（立即实施）
1. ✅ **修复 slurm-build Dockerfile**（已完成）
2. 🔄 **修复 copy_slurm_packages_to_apphub**（进行中）
3. ⏭️ **实现依赖镜像预拉取**

### 中优先级（本周完成）
4. 多阶段构建整合 apphub
5. 优化镜像源配置

### 低优先级（按需实施）
6. 离线部署支持
7. 镜像加速器配置

## 测试计划

### 测试1：copy_slurm_packages 修复验证
```bash
./build.sh build slurm-build v0.3.6-dev
./build.sh build apphub v0.3.6-dev
# 手动测试复制函数
bash -c 'source build.sh && copy_slurm_packages_to_apphub v0.3.6-dev'
# 验证
docker run --rm ai-infra-apphub:v0.3.6-dev ls -la /usr/share/nginx/html/pkgs/slurm-deb/
```

### 测试2：依赖镜像预拉取
```bash
# 清理所有镜像
docker rmi $(docker images -q)
# 测试预拉取
./build.sh build-all v0.3.6-dev
# 应该自动拉取所有依赖并成功构建
```

### 测试3：内部 Harbor 支持
```bash
# 推送依赖到 Harbor
./build.sh deps-push aiharbor.msxf.local/aihpc v0.3.6-dev
# 从 Harbor 拉取并构建
./build.sh build-all v0.3.6-dev aiharbor.msxf.local/aihpc
```

## 交付物

1. 修复的 `copy_slurm_packages_to_apphub` 函数
2. 新增 `prefetch_dependencies` 函数
3. 更新的 `build_all_services` 函数
4. 测试报告
5. 实施文档

---

**创建时间**: 2025年10月10日  
**预计完成**: 2025年10月11日
