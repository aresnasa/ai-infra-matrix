#!/bin/bash
# 多架构构建修复脚本
# 此脚本包含需要添加到 build.sh 的新函数和改进

# ==================== 新增函数 1: 多架构镜像验证 ====================
verify_multiarch_images() {
    local components=("$@")
    local tag="${IMAGE_TAG:-latest}"
    local missing_count=0
    local present_count=0
    
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "🔍 Verifying Multi-Architecture Images"
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    for component in "${components[@]}"; do
        local amd64_image="ai-infra-${component}:${tag}-amd64"
        local arm64_image="ai-infra-${component}:${tag}-arm64"
        local unified_image="ai-infra-${component}:${tag}"
        
        echo -n "  $component: "
        
        local amd64_exists=false
        local arm64_exists=false
        local manifest_exists=false
        
        if docker image inspect "$amd64_image" >/dev/null 2>&1; then
            amd64_exists=true
            present_count=$((present_count + 1))
        else
            missing_count=$((missing_count + 1))
        fi
        
        if docker image inspect "$arm64_image" >/dev/null 2>&1; then
            arm64_exists=true
            present_count=$((present_count + 1))
        else
            missing_count=$((missing_count + 1))
        fi
        
        # 检查 manifest
        if docker manifest inspect "$unified_image" >/dev/null 2>&1; then
            manifest_exists=true
        fi
        
        # 输出状态
        local status="["
        [[ "$amd64_exists" == "true" ]] && status+="✓amd64" || status+="✗amd64"
        status+=" "
        [[ "$arm64_exists" == "true" ]] && status+="✓arm64" || status+="✗arm64"
        [[ "$manifest_exists" == "true" ]] && status+=" ✓manifest" || status+=" ✗manifest"
        status+="]"
        
        echo "$status"
    done
    
    echo ""
    log_info "Summary: $present_count present, $missing_count missing"
    
    if [[ $missing_count -gt 0 ]]; then
        log_error "⚠️  Some images missing. Build may have failed."
        return 1
    else
        log_info "✓ All images present"
        return 0
    fi
}

# ==================== 新增函数 2: 创建多架构 Manifest ====================
create_multiarch_manifests() {
    local components=("$@")
    local tag="${IMAGE_TAG:-latest}"
    
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "📦 Creating Docker Manifests for Multi-Architecture Support"
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    local created=0
    local failed=0
    
    for component in "${components[@]}"; do
        local base_image="ai-infra-${component}"
        local amd64_image="${base_image}:${tag}-amd64"
        local arm64_image="${base_image}:${tag}-arm64"
        local manifest_image="${base_image}:${tag}"
        
        # 检查两个架构的镜像是否都存在
        if ! docker image inspect "$amd64_image" >/dev/null 2>&1; then
            log_warn "  ⚠️  Missing amd64: $amd64_image (skipping manifest creation)"
            failed=$((failed + 1))
            continue
        fi
        
        if ! docker image inspect "$arm64_image" >/dev/null 2>&1; then
            log_warn "  ⚠️  Missing arm64: $arm64_image (skipping manifest creation)"
            failed=$((failed + 1))
            continue
        fi
        
        # 删除旧的 manifest（如果存在）
        docker manifest rm "$manifest_image" 2>/dev/null || true
        
        # 创建新的 manifest list
        log_info "  Creating: $manifest_image"
        
        if docker manifest create "$manifest_image" "$amd64_image" "$arm64_image"; then
            # 添加架构注解
            docker manifest annotate "$manifest_image" "$amd64_image" \
                --os linux --arch amd64 2>/dev/null || true
            docker manifest annotate "$manifest_image" "$arm64_image" \
                --os linux --arch arm64 2>/dev/null || true
            
            log_info "    ✓ Manifest created"
            created=$((created + 1))
        else
            log_error "    ✗ Failed to create manifest"
            failed=$((failed + 1))
        fi
    done
    
    echo ""
    log_info "Manifest creation summary: $created created, $failed failed"
    
    if [[ $failed -eq 0 ]]; then
        log_info "✓ All manifests created successfully"
        return 0
    else
        log_warn "⚠️  Some manifests failed"
        return 1
    fi
}

# ==================== 新增函数 3: 推送多架构镜像和 Manifest ====================
push_multiarch_images() {
    local registry="$1"  # e.g., harbor.example.com/ai-infra
    local tag="${2:-${IMAGE_TAG:-latest}}"
    local components=("${@:3}")
    
    if [[ -z "$registry" ]]; then
        log_error "Registry required: push-multiarch <registry/project> [tag] [components...]"
        return 1
    fi
    
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "🚀 Pushing Multi-Architecture Images to Registry"
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "Registry: $registry"
    log_info "Tag: $tag"
    log_info "Components: ${#components[@]}"
    echo
    
    if [[ ${#components[@]} -eq 0 ]]; then
        # 如果没有指定组件，使用所有已知的组件
        components=(
            "apphub" "backend" "backend-init" "frontend" "nginx"
            "gitea" "saltstack" "slurm-master" "jupyterhub" "singleuser"
            "nightingale" "test-containers" "prometheus"
        )
    fi
    
    local pushed=0
    local failed=0
    
    for component in "${components[@]}"; do
        local local_amd64="ai-infra-${component}:${tag}-amd64"
        local local_arm64="ai-infra-${component}:${tag}-arm64"
        local local_manifest="ai-infra-${component}:${tag}"
        
        local remote_amd64="${registry}/ai-infra-${component}:${tag}-amd64"
        local remote_arm64="${registry}/ai-infra-${component}:${tag}-arm64"
        local remote_manifest="${registry}/ai-infra-${component}:${tag}"
        
        log_info "📦 $component"
        
        # 推送 amd64
        if docker image inspect "$local_amd64" >/dev/null 2>&1; then
            log_info "  Pushing amd64..."
            if docker tag "$local_amd64" "$remote_amd64" && \
               docker push "$remote_amd64"; then
                log_info "    ✓ amd64 pushed"
            else
                log_error "    ✗ amd64 push failed"
                failed=$((failed + 1))
                continue
            fi
        else
            log_warn "    ⚠️  amd64 image not found"
            failed=$((failed + 1))
            continue
        fi
        
        # 推送 arm64
        if docker image inspect "$local_arm64" >/dev/null 2>&1; then
            log_info "  Pushing arm64..."
            if docker tag "$local_arm64" "$remote_arm64" && \
               docker push "$remote_arm64"; then
                log_info "    ✓ arm64 pushed"
            else
                log_error "    ✗ arm64 push failed"
                failed=$((failed + 1))
                continue
            fi
        else
            log_warn "    ⚠️  arm64 image not found"
            failed=$((failed + 1))
            continue
        fi
        
        # 创建并推送 manifest
        if docker manifest create "$remote_manifest" "$remote_amd64" "$remote_arm64"; then
            docker manifest annotate "$remote_manifest" "$remote_amd64" \
                --os linux --arch amd64 2>/dev/null || true
            docker manifest annotate "$remote_manifest" "$remote_arm64" \
                --os linux --arch arm64 2>/dev/null || true
            
            if docker manifest push "$remote_manifest"; then
                log_info "    ✓ manifest pushed"
                pushed=$((pushed + 1))
            else
                log_error "    ✗ manifest push failed"
                failed=$((failed + 1))
            fi
            
            docker manifest rm "$remote_manifest" 2>/dev/null || true
        else
            log_error "    ✗ failed to create manifest"
            failed=$((failed + 1))
        fi
    done
    
    echo ""
    log_info "Push summary: $pushed succeeded, $failed failed"
    
    if [[ $failed -eq 0 ]]; then
        log_info "✓ All images pushed successfully"
        return 0
    else
        log_error "⚠️  Some images failed to push"
        return 1
    fi
}

# ==================== 新增函数 4: 确保 QEMU 支持 ====================
ensure_qemu_for_multiarch() {
    local target_arch="${1:-arm64}"
    local host_arch=$(uname -m)
    
    # 如果主机是 x86_64 并要构建 arm64，需要 QEMU
    if [[ "$host_arch" == "x86_64" ]] && [[ "$target_arch" == "arm64" ]]; then
        log_info "🔧 Setting up QEMU for arm64 cross-compilation..."
        
        if docker run --rm --privileged tonistiigi/binfmt --install arm64 >/dev/null 2>&1; then
            log_info "✓ QEMU arm64 support enabled"
            return 0
        else
            log_error "✗ Failed to setup QEMU"
            log_error "  Try: docker run --rm --privileged tonistiigi/binfmt --install all"
            return 1
        fi
    fi
    
    return 0
}

# ==================== 改进的 build_component_for_platform 包装 ====================
# 添加更好的错误处理和报告
build_component_for_platform_v2() {
    local component="$1"
    local platform="$2"
    local extra_args=("${@:3}")
    
    local arch_name="${platform##*/}"
    
    log_info "🔨 Building $component for $arch_name..."
    
    # 调用原始函数
    if build_component_for_platform "$component" "$platform" "${extra_args[@]}"; then
        local tag="${IMAGE_TAG:-latest}"
        local native_platform=$(_detect_docker_platform)
        local native_arch="${native_platform##*/}"
        
        # 验证输出镜像
        local arch_suffix=""
        if [[ "$arch_name" != "$native_arch" ]]; then
            arch_suffix="-${arch_name}"
        fi
        local expected_image="ai-infra-${component}:${tag}${arch_suffix}"
        
        if docker image inspect "$expected_image" >/dev/null 2>&1; then
            log_info "✓ Verified: $expected_image"
            return 0
        else
            log_error "✗ Build completed but image not found: $expected_image"
            return 1
        fi
    else
        log_error "✗ Build failed for $component on $arch_name"
        return 1
    fi
}

# ==================== 使用示例 ====================
# 
# 在 build_all_multiplatform() 的末尾添加：
#
#   # 验证所有镜像
#   if ! verify_multiarch_images "${FOUNDATION_SERVICES[@]}" "${DEPENDENT_SERVICES[@]}"; then
#       log_error "Build verification failed"
#       return 1
#   fi
#   
#   # 创建 manifest
#   if ! create_multiarch_manifests "${FOUNDATION_SERVICES[@]}" "${DEPENDENT_SERVICES[@]}"; then
#       log_warn "Some manifests failed, but build may still be usable"
#   fi
#

# ==================== 命令行集成示例 ====================
#
# 在 case "$COMMAND" 部分添加新命令：
#
#   verify-multiarch)
#       # 验证多架构镜像
#       discover_services
#       verify_multiarch_images "${FOUNDATION_SERVICES[@]}" "${DEPENDENT_SERVICES[@]}"
#       ;;
#   
#   create-manifest)
#       # 创建 manifest
#       discover_services
#       create_multiarch_manifests "${FOUNDATION_SERVICES[@]}" "${DEPENDENT_SERVICES[@]}"
#       ;;
#   
#   push-multiarch)
#       # 推送多架构镜像
#       discover_services
#       push_multiarch_images "$ARG2" "$ARG3" "${FOUNDATION_SERVICES[@]}" "${DEPENDENT_SERVICES[@]}"
#       ;;
#

echo "✓ Multiarch build improvements loaded"
