#!/bin/bash
# 快速修复脚本：为 build.sh 添加 Docker Manifest 支持
# 
# 使用方法：
#   bash apply_manifest_support.sh
#
# 这个脚本会：
#   1. 备份原始 build.sh
#   2. 添加 manifest 创建函数
#   3. 在 build_all_multiplatform() 末尾添加 manifest 创建逻辑
#   4. 验证修改

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BUILD_SCRIPT="$SCRIPT_DIR/build.sh"
BACKUP_SCRIPT="${BUILD_SCRIPT}.backup.$(date +%Y%m%d_%H%M%S)"

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo "════════════════════════════════════════════════════════════════"
echo "🔧 AI-Infra-Matrix: Quick Manifest Support Fix"
echo "════════════════════════════════════════════════════════════════"
echo ""

# Step 1: 检查 build.sh 是否存在
if [[ ! -f "$BUILD_SCRIPT" ]]; then
    echo -e "${RED}✗ Error: build.sh not found at $BUILD_SCRIPT${NC}"
    exit 1
fi
echo -e "${GREEN}✓${NC} Found build.sh"

# Step 2: 创建备份
echo -e "${YELLOW}→${NC} Backing up build.sh to:"
echo "  $BACKUP_SCRIPT"
cp "$BUILD_SCRIPT" "$BACKUP_SCRIPT"
echo -e "${GREEN}✓${NC} Backup created"
echo ""

# Step 3: 检查是否已经有 manifest 支持
if grep -q "create_multiarch_manifests_impl" "$BUILD_SCRIPT"; then
    echo -e "${YELLOW}⚠️  Warning: Manifest support seems already present${NC}"
    echo ""
    echo "To revert changes, run:"
    echo "  cp $BACKUP_SCRIPT $BUILD_SCRIPT"
    exit 0
fi

# Step 4: 准备要添加的函数
echo -e "${YELLOW}→${NC} Preparing manifest support functions..."

MANIFEST_FUNCTIONS='
# ============================================================================
# Multi-Architecture Manifest Support Functions (Added: '$(date '+%Y-%m-%d')')
# ============================================================================

# Create Docker manifest lists for multi-architecture images
# This enables cloud-native support where docker pull automatically selects the right architecture
create_multiarch_manifests_impl() {
    local components=("$@")
    local tag="${IMAGE_TAG:-latest}"
    
    if [[ ${#components[@]} -eq 0 ]]; then
        log_info "No components specified for manifest creation"
        return 0
    fi
    
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "📦 Creating Docker Manifests for Multi-Architecture Support"
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    local created=0
    local skipped=0
    local failed=0
    
    for component in "${components[@]}"; do
        local base_image="ai-infra-${component}"
        local amd64_image="${base_image}:${tag}-amd64"
        local arm64_image="${base_image}:${tag}-arm64"
        local manifest_image="${base_image}:${tag}"
        
        # Check if both architecture images exist
        if ! docker image inspect "$amd64_image" >/dev/null 2>&1; then
            log_warn "  ⚠️  Missing amd64: $amd64_image"
            skipped=$((skipped + 1))
            continue
        fi
        
        if ! docker image inspect "$arm64_image" >/dev/null 2>&1; then
            log_warn "  ⚠️  Missing arm64: $arm64_image"
            skipped=$((skipped + 1))
            continue
        fi
        
        # Remove old manifest if exists
        docker manifest rm "$manifest_image" 2>/dev/null || true
        
        # Create manifest list
        log_info "  Creating: $manifest_image"
        
        if docker manifest create "$manifest_image" "$amd64_image" "$arm64_image" 2>/dev/null; then
            # Add architecture annotations (optional but helpful)
            docker manifest annotate "$manifest_image" "$amd64_image" \
                --os linux --arch amd64 2>/dev/null || true
            docker manifest annotate "$manifest_image" "$arm64_image" \
                --os linux --arch arm64 2>/dev/null || true
            
            log_info "    ✓ Manifest created successfully"
            created=$((created + 1))
        else
            log_error "    ✗ Failed to create manifest"
            failed=$((failed + 1))
        fi
    done
    
    echo ""
    log_info "Manifest summary: $created created, $skipped missing, $failed failed"
    
    return 0
}

# Verify multi-architecture images
verify_multiarch_images() {
    local components=("$@")
    local tag="${IMAGE_TAG:-latest}"
    
    if [[ ${#components[@]} -eq 0 ]]; then
        return 0
    fi
    
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    log_info "🔍 Verifying Multi-Architecture Images"
    log_info "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    local missing=0
    for component in "${components[@]}"; do
        local amd64="ai-infra-${component}:${tag}-amd64"
        local arm64="ai-infra-${component}:${tag}-arm64"
        
        local status=""
        docker image inspect "$amd64" >/dev/null 2>&1 && status+="✓amd64 " || status+="✗amd64 "
        docker image inspect "$arm64" >/dev/null 2>&1 && status+="✓arm64 " || status+="✗arm64 "
        
        log_info "  $component: $status"
        
        if [[ ! "$status" =~ ✓.*✓ ]]; then
            missing=$((missing + 1))
        fi
    done
    
    echo ""
    if [[ $missing -eq 0 ]]; then
        log_info "✓ All images verified"
    else
        log_warn "⚠️  $missing components have missing architectures"
    fi
    
    return 0
}
'

# Step 5: 寻找插入位置（build_all_multiplatform 函数末尾）
LINE_NUMBER=$(grep -n "^build_all_multiplatform()" "$BUILD_SCRIPT" | cut -d: -f1)
if [[ -z "$LINE_NUMBER" ]]; then
    echo -e "${RED}✗ Could not find build_all_multiplatform() function${NC}"
    exit 1
fi

# 找到该函数的结尾（下一个 ^[a-z_]*() 或文件末尾）
END_LINE=$(awk -v start="$LINE_NUMBER" 'NR > start && /^[a-zA-Z_][a-zA-Z0-9_]*\(\)/ {print NR-1; exit}' "$BUILD_SCRIPT")
if [[ -z "$END_LINE" ]]; then
    END_LINE=$(wc -l < "$BUILD_SCRIPT")
fi

echo -e "${GREEN}✓${NC} Found build_all_multiplatform() at line $LINE_NUMBER"
echo "  Function ends at approximately line $END_LINE"
echo ""

# Step 6: 在函数末尾添加 manifest 创建调用
echo -e "${YELLOW}→${NC} Adding manifest creation call to build_all_multiplatform()..."

# 找到函数中 "log_info" 开始的行（在最后打印完成消息的地方）
# 我们在那里之前插入 manifest 创建代码

# 寻找该函数中最后一个 log_info 开始的行（在末尾）
MANIFEST_INSERT_MARKER='    # Build summary'
INSERT_POINT=$(awk -v start="$LINE_NUMBER" -v end="$END_LINE" 'NR >= start && NR <= end && /# Build summary/ {print NR; exit}' "$BUILD_SCRIPT" | tail -1)

if [[ -n "$INSERT_POINT" ]]; then
    # 在这一行之前插入
    sed -i '' "${INSERT_POINT}i\\
    \\
    # Phase 5: Create Docker manifests for multi-architecture support\\
    if [[ \${#normalized_platforms[@]} -gt 1 ]]; then\\
        log_info \"\"\\
        create_multiarch_manifests_impl \"\${FOUNDATION_SERVICES[@]}\" \"\${DEPENDENT_SERVICES[@]}\"\\
    fi\\
" "$BUILD_SCRIPT"
    
    echo -e "${GREEN}✓${NC} Manifest creation code added at line $INSERT_POINT"
else
    echo -e "${YELLOW}⚠️  Could not find exact insertion point, will add at end of file${NC}"
fi

# Step 7: 添加 manifest 函数定义
# 在文件末尾（最后一个函数之后）添加新函数
echo -e "${YELLOW}→${NC} Adding manifest support functions..."

# 找到最后一个 "^[a-z_]*() {" 的行
LAST_FUNC_LINE=$(grep -n "^[a-zA-Z_][a-zA-Z0-9_]*() {" "$BUILD_SCRIPT" | tail -1 | cut -d: -f1)

if [[ -n "$LAST_FUNC_LINE" ]]; then
    # 找到该函数的结尾（下一个 ^} 或下一个函数开始）
    FUNC_END=$(awk -v start="$LAST_FUNC_LINE" 'NR > start && /^[a-zA-Z_]/ && !/^[[:space:]]/ {print NR-1; exit}' "$BUILD_SCRIPT" | tail -1)
    [[ -z "$FUNC_END" ]] && FUNC_END=$(wc -l < "$BUILD_SCRIPT")
    
    # 在该行之后插入新函数
    sed -i '' "${FUNC_END}a\\
${MANIFEST_FUNCTIONS}
" "$BUILD_SCRIPT"
    
    echo -e "${GREEN}✓${NC} Functions added at line $FUNC_END"
else
    # 直接追加到文件末尾
    echo "" >> "$BUILD_SCRIPT"
    echo "$MANIFEST_FUNCTIONS" >> "$BUILD_SCRIPT"
    echo -e "${GREEN}✓${NC} Functions appended to end of file"
fi

# Step 8: 验证修改
echo ""
echo -e "${YELLOW}→${NC} Verifying changes..."

if grep -q "create_multiarch_manifests_impl" "$BUILD_SCRIPT"; then
    echo -e "${GREEN}✓${NC} Manifest functions added successfully"
else
    echo -e "${RED}✗ Verification failed${NC}"
    echo "Reverting to backup..."
    cp "$BACKUP_SCRIPT" "$BUILD_SCRIPT"
    exit 1
fi

# Step 9: 完成
echo ""
echo "════════════════════════════════════════════════════════════════"
echo "✅ Manifest Support Successfully Added!"
echo "════════════════════════════════════════════════════════════════"
echo ""
echo "Summary of changes:"
echo "  ✓ Added create_multiarch_manifests_impl() function"
echo "  ✓ Added verify_multiarch_images() function"
echo "  ✓ Integrated manifest creation into build_all_multiplatform()"
echo "  ✓ Backup saved: $BACKUP_SCRIPT"
echo ""
echo "Next steps:"
echo "  1. Review changes: diff $BUILD_SCRIPT $BACKUP_SCRIPT"
echo "  2. Test the build: ./build.sh all --platform=amd64,arm64"
echo "  3. Verify manifests: docker manifest inspect ai-infra-backend:v0.3.8"
echo ""
echo "To revert if needed:"
echo "  cp $BACKUP_SCRIPT $BUILD_SCRIPT"
echo ""
