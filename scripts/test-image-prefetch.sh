#!/bin/bash

# 镜像预拉取功能测试脚本

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

echo "=========================================="
echo "测试 build.sh 镜像预拉取功能"
echo "=========================================="
echo

# 测试 1: 提取基础镜像功能
echo "测试 1: 提取 Dockerfile 中的基础镜像"
echo "----------------------------------------"

test_dockerfile="$PROJECT_DIR/src/backend/Dockerfile"
if [[ -f "$test_dockerfile" ]]; then
    echo "测试文件: $test_dockerfile"
    echo
    echo "提取的基础镜像:"
    grep -E '^\s*FROM\s+' "$test_dockerfile" | \
        sed -E 's/^\s*FROM\s+(--platform=[^\s]+\s+)?([^\s]+)(\s+AS\s+.*)?$/\2/' | \
        grep -v '^$' | \
        sort -u
    echo
else
    echo "⚠ 测试文件不存在: $test_dockerfile"
fi

# 测试 2: 扫描所有服务
echo "测试 2: 扫描所有服务的 Dockerfile"
echo "----------------------------------------"

src_dirs=(
    "backend"
    "frontend"
    "jupyterhub"
    "nginx"
    "saltstack"
    "slurm-master"
    "test-containers"
)

total_images=()

for service in "${src_dirs[@]}"; do
    dockerfile="$PROJECT_DIR/src/$service/Dockerfile"
    if [[ -f "$dockerfile" ]]; then
        echo "✓ $service: 找到 Dockerfile"
        
        # 提取镜像
        images=$(grep -E '^\s*FROM\s+' "$dockerfile" 2>/dev/null | \
            sed -E 's/^\s*FROM\s+(--platform=[^\s]+\s+)?([^\s]+)(\s+AS\s+.*)?$/\2/' | \
            grep -v '^$' | \
            grep -v '^[a-z_-]\+$' || true)
        
        if [[ -n "$images" ]]; then
            while IFS= read -r image; do
                total_images+=("$image")
            done <<< "$images"
        fi
    else
        echo "✗ $service: Dockerfile 不存在"
    fi
done

echo
echo "----------------------------------------"
echo "📊 统计信息"
echo "----------------------------------------"
echo "扫描的服务数: ${#src_dirs[@]}"
echo "发现的镜像数（含重复）: ${#total_images[@]}"

# 去重
unique_images=($(printf '%s\n' "${total_images[@]}" | sort -u))
echo "唯一的基础镜像数: ${#unique_images[@]}"
echo

# 测试 3: 检查镜像是否存在
echo "测试 3: 检查常见基础镜像"
echo "----------------------------------------"

common_images=(
    "alpine:3.20"
    "golang:1.23-alpine"
    "node:20-alpine"
    "nginx:alpine"
    "ubuntu:22.04"
)

exist_count=0
missing_count=0

for image in "${common_images[@]}"; do
    if docker image inspect "$image" >/dev/null 2>&1; then
        echo "✓ 已存在: $image"
        ((exist_count++))
    else
        echo "✗ 不存在: $image"
        ((missing_count++))
    fi
done

echo
echo "已存在: $exist_count"
echo "不存在: $missing_count"
echo

# 测试 4: 模拟预拉取（只检查，不实际拉取）
echo "测试 4: 模拟预拉取流程"
echo "----------------------------------------"

echo "📦 将要拉取的镜像（前 5 个）:"
count=0
for image in "${unique_images[@]}"; do
    if [[ $count -ge 5 ]]; then
        break
    fi
    
    if docker image inspect "$image" >/dev/null 2>&1; then
        echo "  ⊙ $image (已存在)"
    else
        echo "  ⬇ $image (需要拉取)"
    fi
    
    ((count++))
done

if [[ ${#unique_images[@]} -gt 5 ]]; then
    echo "  ... 还有 $((${#unique_images[@]} - 5)) 个镜像"
fi

echo
echo "=========================================="
echo "✅ 测试完成"
echo "=========================================="
echo
echo "建议操作:"
echo "1. 运行 './build.sh build-service backend' 测试单个服务预拉取"
echo "2. 运行 './build.sh build-all' 测试批量预拉取"
echo "3. 检查日志中的 '📦 预拉取依赖镜像' 部分"
echo

exit 0
