#!/bin/bash
# 多架构构建诊断脚本
# 用于快速定位和诊断 build.sh all --platform=amd64,arm64 的问题

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "════════════════════════════════════════════════════════════════"
echo "🔍 AI-Infra-Matrix Multi-Architecture Build Diagnosis"
echo "════════════════════════════════════════════════════════════════"
echo ""

# ==================== 诊断 1: 环境检查 ====================
echo "1️⃣  Environment Check"
echo "─────────────────────────────────────────────────────────────────"

echo "Host Architecture:"
ARCH=$(uname -m)
echo "  $(uname -s) $(uname -m)"

echo ""
echo "Docker Information:"
docker --version
docker buildx version 2>/dev/null || echo "  ⚠️  docker buildx not available"

echo ""
echo "BuildX Builders:"
docker buildx ls 2>/dev/null | head -10 || echo "  ⚠️  No buildx builders available"

echo ""
echo "QEMU Support:"
if docker run --rm --privileged tonistiigi/binfmt --version 2>/dev/null | grep -q "binfmt"; then
    echo "  ✓ binfmt-misc available"
    docker run --rm --privileged tonistiigi/binfmt --version 2>/dev/null || true
else
    echo "  ⚠️  binfmt-misc not available"
fi

echo ""

# ==================== 诊断 2: .env 配置检查 ====================
echo "2️⃣  Configuration Check (.env)"
echo "─────────────────────────────────────────────────────────────────"

if [[ -f "$SCRIPT_DIR/.env" ]]; then
    echo "✓ .env file found"
    grep -E "^(IMAGE_TAG|EXTERNAL_HOST|DOMAIN|ENABLE_TLS)=" "$SCRIPT_DIR/.env" 2>/dev/null || echo "  (no relevant vars)"
else
    echo "⚠️  .env not found - run: ./build.sh init-env"
fi

echo ""

# ==================== 诊断 3: 本地 Docker 镜像检查 ====================
echo "3️⃣  Local Docker Images"
echo "─────────────────────────────────────────────────────────────────"

# 获取 IMAGE_TAG
IMAGE_TAG=$(grep "^IMAGE_TAG=" "$SCRIPT_DIR/.env" 2>/dev/null | cut -d'=' -f2 || echo "latest")

echo "Looking for images with tag: $IMAGE_TAG"
echo ""

# 定义所有已知的组件
COMPONENTS=(
    "apphub" "backend" "backend-init" "frontend" "nginx"
    "gitea" "saltstack" "slurm-master" "jupyterhub" "singleuser"
    "nightingale" "test-containers" "prometheus"
)

echo "Component Status:"
echo ""

TOTAL_IMAGES=0
AMD64_IMAGES=0
ARM64_IMAGES=0
MANIFEST_COUNT=0

for component in "${COMPONENTS[@]}"; do
    printf "  %-20s " "ai-infra-$component"
    
    local_amd64="ai-infra-${component}:${IMAGE_TAG}-amd64"
    local_arm64="ai-infra-${component}:${IMAGE_TAG}-arm64"
    manifest="ai-infra-${component}:${IMAGE_TAG}"
    
    STATUS=""
    
    if docker image inspect "$local_amd64" >/dev/null 2>&1; then
        STATUS+="✓amd64 "
        AMD64_IMAGES=$((AMD64_IMAGES + 1))
    else
        STATUS+="✗amd64 "
    fi
    TOTAL_IMAGES=$((TOTAL_IMAGES + 1))
    
    if docker image inspect "$local_arm64" >/dev/null 2>&1; then
        STATUS+="✓arm64 "
        ARM64_IMAGES=$((ARM64_IMAGES + 1))
    else
        STATUS+="✗arm64 "
    fi
    TOTAL_IMAGES=$((TOTAL_IMAGES + 1))
    
    if docker manifest inspect "$manifest" >/dev/null 2>&1; then
        STATUS+="✓mani"
        MANIFEST_COUNT=$((MANIFEST_COUNT + 1))
    else
        STATUS+="✗mani"
    fi
    
    echo "$STATUS"
done

echo ""
echo "Summary:"
echo "  Total images needed: $((${#COMPONENTS[@]} * 2)) (both archs × components)"
echo "  AMD64 images found: $AMD64_IMAGES / ${#COMPONENTS[@]}"
echo "  ARM64 images found: $ARM64_IMAGES / ${#COMPONENTS[@]}"
echo "  Manifests created: $MANIFEST_COUNT / ${#COMPONENTS[@]}"
echo ""

# ==================== 诊断 4: build.sh 功能检查 ====================
echo "4️⃣  build.sh Script Analysis"
echo "─────────────────────────────────────────────────────────────────"

if [[ ! -f "$SCRIPT_DIR/build.sh" ]]; then
    echo "✗ build.sh not found!"
    exit 1
fi

echo "Script size: $(wc -l < "$SCRIPT_DIR/build.sh") lines"
echo ""

# 检查关键函数是否存在
echo "Key Functions:"

if grep -q "^build_all_multiplatform()" "$SCRIPT_DIR/build.sh"; then
    echo "  ✓ build_all_multiplatform()"
else
    echo "  ✗ build_all_multiplatform() - MISSING!"
fi

if grep -q "^build_component_for_platform()" "$SCRIPT_DIR/build.sh"; then
    echo "  ✓ build_component_for_platform()"
else
    echo "  ✗ build_component_for_platform() - MISSING!"
fi

if grep -q "docker manifest create" "$SCRIPT_DIR/build.sh"; then
    echo "  ✓ docker manifest support"
else
    echo "  ⚠️  docker manifest not found in script"
fi

echo ""

# ==================== 诊断 5: 命令行参数处理检查 ====================
echo "5️⃣  Command-Line Parameter Handling"
echo "─────────────────────────────────────────────────────────────────"

if grep -q 'BUILD_PLATFORMS="${arg#\*=}"' "$SCRIPT_DIR/build.sh"; then
    echo "  ✓ --platform parameter parsing present"
else
    echo "  ⚠️  --platform parameter parsing not found"
fi

if grep -q 'build_all_multiplatform.*BUILD_PLATFORMS' "$SCRIPT_DIR/build.sh"; then
    echo "  ✓ Parameter passed to build_all_multiplatform()"
else
    echo "  ⚠️  Parameter not passed to build_all_multiplatform()"
fi

echo ""

# ==================== 诊断 6: 模拟构建命令 ====================
echo "6️⃣  Test Commands"
echo "─────────────────────────────────────────────────────────────────"

echo "To reproduce the issue:"
echo ""
echo "  Step 1: Setup QEMU (if building arm64 on amd64)"
echo "    docker run --rm --privileged tonistiigi/binfmt --install all"
echo ""
echo "  Step 2: Initialize environment"
echo "    cd $SCRIPT_DIR"
echo "    ./build.sh init-env"
echo ""
echo "  Step 3: Run multi-architecture build with verbose logging"
echo "    ./build.sh all --platform=amd64,arm64 2>&1 | tee build.log"
echo ""
echo "  Step 4: Check for errors"
echo "    grep -i 'error\\|fail\\|not found' build.log"
echo ""
echo "  Step 5: Verify images after build"
echo "    ./diagnose-multiarch.sh"  # This script itself
echo ""

# ==================== 诊断 7: 快速问题诊断 ====================
echo "7️⃣  Problem Diagnosis"
echo "─────────────────────────────────────────────────────────────────"

if [[ $AMD64_IMAGES -eq 0 ]] && [[ $ARM64_IMAGES -eq 0 ]]; then
    echo "❌ CRITICAL: No images found for any architecture"
    echo ""
    echo "Possible causes:"
    echo "  1. Build.sh never ran (missing .env or initialization)"
    echo "  2. Build command failed silently"
    echo "  3. Image naming scheme changed"
    echo ""
    echo "Next steps:"
    echo "  1. Check if .env exists: ls -la .env"
    echo "  2. Run: ./build.sh init-env"
    echo "  3. Run: ./build.sh render (to generate Dockerfiles)"
    echo "  4. Run with verbose: ./build.sh all --platform=amd64,arm64 -v"
    echo ""
    
elif [[ $AMD64_IMAGES -gt 0 ]] && [[ $ARM64_IMAGES -eq 0 ]]; then
    echo "⚠️  PARTIAL: AMD64 images found, but ARM64 images missing"
    echo ""
    echo "Possible causes:"
    echo "  1. QEMU support not enabled on amd64 machine"
    echo "  2. ARM64 build failed but error was silently ignored"
    echo "  3. Docker buildx not configured for ARM64"
    echo ""
    echo "Next steps:"
    echo "  1. Setup QEMU: docker run --rm --privileged tonistiigi/binfmt --install all"
    echo "  2. Verify buildx: docker buildx ls"
    echo "  3. Try single-platform rebuild: ./build.sh build-platform arm64 --force"
    echo ""
    
elif [[ $MANIFEST_COUNT -eq 0 ]] && [[ $((AMD64_IMAGES + ARM64_IMAGES)) -gt 0 ]]; then
    echo "✓ PARTIAL: Images built for both/one architecture, but manifests missing"
    echo ""
    echo "This is expected if build just completed."
    echo ""
    echo "Next steps:"
    echo "  1. Create manifests: ./build.sh create-manifest"
    echo "  2. Or push to registry: ./build.sh push-all registry.example.com/ai-infra latest"
    echo ""
    
elif [[ $MANIFEST_COUNT -gt 0 ]]; then
    echo "✓ GOOD: Multi-architecture images and manifests present"
    echo ""
    echo "Status: Build successful and manifests created"
    echo ""
    echo "Next steps:"
    echo "  1. Export for offline use: ./build.sh export-offline ./offline latest true amd64,arm64"
    echo "  2. Or push to registry: ./build.sh push-all registry.example.com/ai-infra latest"
    echo ""
fi

echo ""
echo "════════════════════════════════════════════════════════════════"
echo "📋 Diagnosis Complete"
echo "════════════════════════════════════════════════════════════════"
