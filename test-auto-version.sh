#!/bin/bash

# 测试自动版本管理功能
# 用法: ./test-auto-version.sh

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 颜色输出
print_info() {
    echo -e "\033[32m[INFO]\033[0m $1"
}

print_success() {
    echo -e "\033[32m[✓]\033[0m $1"
}

print_error() {
    echo -e "\033[31m[✗]\033[0m $1"
}

echo "=========================================="
echo "测试自动版本管理功能"
echo "=========================================="
echo

# 测试 1: get_current_git_branch
print_info "测试 1: 获取当前 Git 分支"
source "$SCRIPT_DIR/build.sh" > /dev/null 2>&1 || true
CURRENT_BRANCH=$(get_current_git_branch)
echo "  当前分支: $CURRENT_BRANCH"
if [[ -n "$CURRENT_BRANCH" ]]; then
    print_success "成功获取分支名"
else
    print_error "获取分支名失败"
    exit 1
fi
echo

# 测试 2: deps.yaml 同步
print_info "测试 2: 同步 deps.yaml 到 .env"
if [[ -f "$SCRIPT_DIR/deps.yaml" ]]; then
    # 创建临时 .env 文件
    TMP_ENV=$(mktemp)
    cp "$SCRIPT_DIR/.env" "$TMP_ENV" 2>/dev/null || touch "$TMP_ENV"
    
    # 同步
    sync_deps_from_yaml "$TMP_ENV"
    
    # 检查是否成功写入
    if grep -q "POSTGRES_VERSION" "$TMP_ENV"; then
        print_success "成功同步依赖版本"
        echo "  示例: $(grep POSTGRES_VERSION $TMP_ENV)"
    else
        print_error "同步依赖版本失败"
    fi
    
    rm -f "$TMP_ENV"
else
    print_error "deps.yaml 文件不存在"
fi
echo

# 测试 3: update_component_tags_from_branch
print_info "测试 3: 根据分支更新组件标签"
TMP_ENV=$(mktemp)
update_component_tags_from_branch "$TMP_ENV"
if grep -q "IMAGE_TAG=$CURRENT_BRANCH" "$TMP_ENV"; then
    print_success "成功更新组件标签"
    echo "  IMAGE_TAG: $(grep IMAGE_TAG $TMP_ENV | head -1)"
    echo "  DEFAULT_IMAGE_TAG: $(grep DEFAULT_IMAGE_TAG $TMP_ENV | head -1)"
else
    print_error "更新组件标签失败"
fi
rm -f "$TMP_ENV"
echo

# 测试 4: 验证 build.sh 语法
print_info "测试 4: 验证 build.sh 语法"
if bash -n "$SCRIPT_DIR/build.sh"; then
    print_success "build.sh 语法正确"
else
    print_error "build.sh 语法错误"
    exit 1
fi
echo

# 测试 5: 检查 deps.yaml 格式
print_info "测试 5: 检查 deps.yaml 格式"
if [[ -f "$SCRIPT_DIR/deps.yaml" ]]; then
    # 检查是否有有效的键值对
    VALID_LINES=$(grep -E '^\s*[a-zA-Z0-9_-]+:\s*"?[^"]+' "$SCRIPT_DIR/deps.yaml" | wc -l)
    echo "  有效配置项: $VALID_LINES"
    if [[ $VALID_LINES -gt 0 ]]; then
        print_success "deps.yaml 格式正确"
        echo "  示例配置:"
        grep -E '^\s*[a-zA-Z0-9_-]+:\s*"?[^"]+' "$SCRIPT_DIR/deps.yaml" | head -3
    else
        print_error "deps.yaml 格式错误"
    fi
else
    print_error "deps.yaml 文件不存在"
fi
echo

echo "=========================================="
print_success "所有测试完成"
echo "=========================================="
echo
echo "现在可以尝试以下命令："
echo "  ./build.sh build-all           # 自动使用分支 $CURRENT_BRANCH 作为标签"
echo "  ./build.sh push-all <registry> # 自动推送分支 $CURRENT_BRANCH 的镜像"
echo "  ./build.sh push-dep <registry> # 自动推送依赖镜像"
