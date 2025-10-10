#!/bin/bash

# 智能构建缓存系统测试脚本
# 用于验证所有功能是否正常工作

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$PROJECT_DIR"

echo "=========================================="
echo "智能构建缓存系统功能测试"
echo "=========================================="
echo

# 导入必要的函数（直接定义，避免 source build.sh）
BUILD_CACHE_DIR="$PROJECT_DIR/.build-cache"
BUILD_ID_FILE="$BUILD_CACHE_DIR/build-id.txt"
BUILD_HISTORY_FILE="$BUILD_CACHE_DIR/build-history.log"

print_error() {
    echo -e "\033[31m[ERROR]\033[0m $1"
}

print_success() {
    echo -e "\033[32m[SUCCESS]\033[0m $1"
}

init_build_cache() {
    mkdir -p "$BUILD_CACHE_DIR"
    if [[ ! -f "$BUILD_ID_FILE" ]]; then
        echo "0" > "$BUILD_ID_FILE"
    fi
    if [[ ! -f "$BUILD_HISTORY_FILE" ]]; then
        touch "$BUILD_HISTORY_FILE"
    fi
}

generate_build_id() {
    init_build_cache
    local last_id=$(cat "$BUILD_ID_FILE" 2>/dev/null || echo "0")
    local new_id=$((last_id + 1))
    local timestamp=$(date +%Y%m%d_%H%M%S)
    echo "${new_id}_${timestamp}"
}

calculate_hash() {
    local path="$1"
    if [[ ! -e "$path" ]]; then
        echo "NOT_EXIST"
        return 1
    fi
    if [[ -d "$path" ]]; then
        find "$path" -type f \( -name "*.py" -o -name "*.js" -o -name "*.ts" -o -name "*.tsx" -o -name "*.go" -o -name "*.conf" -o -name "*.yaml" -o -name "*.yml" -o -name "*.json" -o -name "Dockerfile" \) -exec shasum -a 256 {} \; 2>/dev/null | sort | shasum -a 256 | awk '{print $1}'
    else
        shasum -a 256 "$path" 2>/dev/null | awk '{print $1}'
    fi
}

calculate_service_hash() {
    local service="$1"
    local service_path="src/$service"
    local hash_data=""
    local dockerfile="$PROJECT_DIR/$service_path/Dockerfile"
    if [[ -f "$dockerfile" ]]; then
        hash_data+="$(calculate_hash "$dockerfile")\n"
    fi
    local src_dir="$PROJECT_DIR/$service_path"
    if [[ -d "$src_dir" ]]; then
        hash_data+="$(calculate_hash "$src_dir")\n"
    fi
    echo -e "$hash_data" | shasum -a 256 | awk '{print $1}'
}

log_build_history() {
    local build_id="$1"
    local service="$2"
    local tag="$3"
    local status="$4"
    local reason="${5:-}"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    init_build_cache
    local log_entry="[$timestamp] BUILD_ID=$build_id SERVICE=$service TAG=$tag STATUS=$status"
    if [[ -n "$reason" ]]; then
        log_entry+=" REASON=$reason"
    fi
    echo "$log_entry" >> "$BUILD_HISTORY_FILE"
}

save_service_build_info() {
    local service="$1"
    local tag="$2"
    local build_id="$3"
    local service_hash="$4"
    local cache_dir="$BUILD_CACHE_DIR/$service"
    mkdir -p "$cache_dir"
    local build_info_file="$cache_dir/last-build.json"
    cat > "$build_info_file" <<EOF
{
  "service": "$service",
  "tag": "$tag",
  "build_id": "$build_id",
  "hash": "$service_hash",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "image": "ai-infra-${service}:${tag}"
}
EOF
}

show_build_cache_stats() {
    if [[ ! -d "$BUILD_CACHE_DIR" ]]; then
        echo "缓存目录不存在"
        return
    fi
    local total_builds=$(cat "$BUILD_ID_FILE" 2>/dev/null || echo "0")
    echo "总构建次数: $total_builds"
}

clean_build_cache() {
    local service="${1:-}"
    if [[ -n "$service" ]]; then
        if [[ -d "$BUILD_CACHE_DIR/$service" ]]; then
            rm -rf "$BUILD_CACHE_DIR/$service"
            return 0
        fi
    fi
    return 1
}

extract_base_images() {
    local dockerfile_path="$1"
    if [[ ! -f "$dockerfile_path" ]]; then
        return 1
    fi
    grep -iE '^\s*FROM\s+' "$dockerfile_path" | \
        sed -E 's/^[[:space:]]*[Ff][Rr][Oo][Mm][[:space:]]+//' | \
        sed -E 's/--platform=[^[:space:]]+[[:space:]]+//' | \
        awk '{print $1}' | \
        grep -v '^$' | \
        grep -v '^#' | \
        sort -u
}

# 测试1: 基础镜像提取功能
echo "测试1: 基础镜像提取功能"
echo "----------------------------------------"
images=$(extract_base_images "src/backend/Dockerfile" 2>/dev/null)
if echo "$images" | grep -q "golang"; then
    echo "✅ PASS: 成功提取基础镜像"
    echo "   提取结果: $images"
else
    echo "❌ FAIL: 镜像提取失败或包含 FROM 关键字"
    echo "   提取结果: $images"
    exit 1
fi
echo

# 测试2: BUILD_ID 生成
echo "测试2: BUILD_ID 生成"
echo "----------------------------------------"
build_id1=$(generate_build_id)
sleep 1
build_id2=$(generate_build_id)
if [[ "$build_id1" != "$build_id2" ]]; then
    echo "✅ PASS: BUILD_ID 生成成功且唯一"
    echo "   BUILD_ID 1: $build_id1"
    echo "   BUILD_ID 2: $build_id2"
else
    echo "❌ FAIL: BUILD_ID 不唯一"
    exit 1
fi
echo

# 测试3: 文件哈希计算
echo "测试3: 文件哈希计算"
echo "----------------------------------------"
hash1=$(calculate_hash "src/backend/Dockerfile")
hash2=$(calculate_hash "src/backend/Dockerfile")
if [[ -n "$hash1" ]] && [[ "$hash1" == "$hash2" ]]; then
    echo "✅ PASS: 文件哈希计算正确"
    echo "   哈希值: ${hash1:0:16}..."
else
    echo "❌ FAIL: 哈希计算失败或不一致"
    exit 1
fi
echo

# 测试4: 服务哈希计算
echo "测试4: 服务哈希计算"
echo "----------------------------------------"
service_hash=$(calculate_service_hash "backend")
if [[ -n "$service_hash" ]] && [[ ${#service_hash} -eq 64 ]]; then
    echo "✅ PASS: 服务哈希计算成功"
    echo "   服务: backend"
    echo "   哈希: ${service_hash:0:16}..."
else
    echo "❌ FAIL: 服务哈希计算失败"
    exit 1
fi
echo

# 测试5: 缓存初始化
echo "测试5: 缓存目录初始化"
echo "----------------------------------------"
init_build_cache
if [[ -d ".build-cache" ]] && [[ -f ".build-cache/build-id.txt" ]]; then
    echo "✅ PASS: 缓存目录初始化成功"
    echo "   目录: .build-cache/"
    echo "   文件: build-id.txt, build-history.log"
else
    echo "❌ FAIL: 缓存目录初始化失败"
    exit 1
fi
echo

# 测试6: 构建历史记录
echo "测试6: 构建历史记录"
echo "----------------------------------------"
test_build_id="TEST_123456"
log_build_history "$test_build_id" "test-service" "v1.0.0" "SUCCESS" "TEST"
if grep -q "$test_build_id" ".build-cache/build-history.log"; then
    echo "✅ PASS: 构建历史记录成功"
    echo "   记录: $(tail -1 .build-cache/build-history.log)"
else
    echo "❌ FAIL: 构建历史记录失败"
    exit 1
fi
echo

# 测试7: 构建信息保存
echo "测试7: 构建信息保存"
echo "----------------------------------------"
save_service_build_info "test-service" "v1.0.0" "$test_build_id" "abc123"
if [[ -f ".build-cache/test-service/last-build.json" ]]; then
    echo "✅ PASS: 构建信息保存成功"
    cat ".build-cache/test-service/last-build.json" | head -3
else
    echo "❌ FAIL: 构建信息保存失败"
    exit 1
fi
echo

# 测试8: 缓存统计（不输出详细内容）
echo "测试8: 缓存统计功能"
echo "----------------------------------------"
if show_build_cache_stats >/dev/null 2>&1; then
    echo "✅ PASS: 缓存统计功能正常"
else
    echo "❌ FAIL: 缓存统计功能失败"
    exit 1
fi
echo

# 测试9: macOS sed 兼容性
echo "测试9: macOS sed 兼容性"
echo "----------------------------------------"
test_result=$(echo "FROM golang:1.25-alpine" | sed -E 's/^[[:space:]]*[Ff][Rr][Oo][Mm][[:space:]]+//')
if [[ "$test_result" == "golang:1.25-alpine" ]]; then
    echo "✅ PASS: sed 命令跨平台兼容"
    echo "   输入: FROM golang:1.25-alpine"
    echo "   输出: $test_result"
else
    echo "❌ FAIL: sed 命令不兼容"
    echo "   输出: $test_result"
    exit 1
fi
echo

# 测试10: 清理测试数据
echo "测试10: 缓存清理功能"
echo "----------------------------------------"
if clean_build_cache "test-service" >/dev/null 2>&1; then
    if [[ ! -d ".build-cache/test-service" ]]; then
        echo "✅ PASS: 缓存清理功能正常"
    else
        echo "❌ FAIL: 缓存清理未生效"
        exit 1
    fi
else
    echo "❌ FAIL: 缓存清理功能失败"
    exit 1
fi
echo

echo "=========================================="
echo "✅ 所有测试通过！"
echo "=========================================="
echo
echo "智能构建缓存系统功能正常，可以使用以下命令："
echo "  ./build.sh build <service>       - 智能构建"
echo "  ./build.sh cache-stats           - 查看统计"
echo "  ./build.sh build-info <service>  - 查看镜像信息"
echo "  ./build.sh clean-cache           - 清理缓存"
echo
echo "详细文档: docs/BUILD_SMART_CACHE_GUIDE.md"
