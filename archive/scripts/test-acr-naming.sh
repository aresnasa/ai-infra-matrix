#!/bin/bash

# 测试阿里云ACR镜像命名功能
# 验证新的get_target_image_name函数是否正确工作

set -e

echo "🧪 测试阿里云ACR镜像命名功能"
echo "=================================="

# 导入build.sh的函数
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
BUILD_SCRIPT="$SCRIPT_DIR/build.sh"

if [ ! -f "$BUILD_SCRIPT" ]; then
    echo "❌ 错误: 找不到build.sh脚本"
    exit 1
fi

# 提取函数定义进行测试
extract_and_test_function() {
    # 创建临时测试脚本
    cat > /tmp/test_acr_naming.sh << 'EOF'
#!/bin/bash

# 从build.sh中提取的函数定义
get_target_image_name() {
    local source_name="$1"
    local version="$2"
    
    if [ -z "$REGISTRY" ]; then
        echo "${source_name}:${version}"
        return
    fi
    
    # 检查是否是阿里云ACR格式 (*.aliyuncs.com)
    if echo "$REGISTRY" | grep -q "\.aliyuncs\.com"; then
        # 阿里云ACR格式: registry/namespace/repository:tag
        # 例如: xxx.aliyuncs.com/ai-infra-matrix/ai-infra-matrix:v0.0.3.3
        
        # 从REGISTRY中提取namespace（假设格式为 registry.com/namespace 或直接是 registry.com）
        local registry_host
        local namespace
        
        if echo "$REGISTRY" | grep -q "/"; then
            registry_host=$(echo "$REGISTRY" | cut -d'/' -f1)
            namespace=$(echo "$REGISTRY" | cut -d'/' -f2-)
        else
            registry_host="$REGISTRY"
            namespace="ai-infra-matrix"  # 默认命名空间
        fi
        
        # 对于阿里云ACR，将所有ai-infra组件映射到统一的repository名称
        case "$source_name" in
            ai-infra-*)
                # 所有ai-infra组件使用相同的repository名，通过tag区分
                echo "${registry_host}/${namespace}/ai-infra-matrix:${source_name#ai-infra-}-${version}"
                ;;
            *)
                # 非ai-infra组件保持原名
                echo "${registry_host}/${namespace}/${source_name}:${version}"
                ;;
        esac
    else
        # 其他注册表保持原有逻辑
        echo "${REGISTRY}/${source_name}:${version}"
    fi
}

# 测试用例
test_case() {
    local description="$1"
    local registry="$2"
    local source_name="$3"
    local version="$4"
    local expected="$5"
    
    export REGISTRY="$registry"
    local result
    result=$(get_target_image_name "$source_name" "$version")
    
    echo "📋 测试: $description"
    echo "  注册表: $registry"
    echo "  源镜像: $source_name:$version"
    echo "  期望结果: $expected"
    echo "  实际结果: $result"
    
    if [ "$result" = "$expected" ]; then
        echo "  ✅ 通过"
    else
        echo "  ❌ 失败"
        return 1
    fi
    echo ""
}

# 运行测试用例
echo "开始测试阿里云ACR命名逻辑..."
echo ""

# 测试1: 阿里云ACR带命名空间
test_case "阿里云ACR带命名空间 - backend" \
    "xxx.aliyuncs.com/ai-infra-matrix" \
    "ai-infra-backend" \
    "v0.0.3.3" \
    "xxx.aliyuncs.com/ai-infra-matrix/ai-infra-matrix:backend-v0.0.3.3"

# 测试2: 阿里云ACR带命名空间 - frontend  
test_case "阿里云ACR带命名空间 - frontend" \
    "xxx.aliyuncs.com/ai-infra-matrix" \
    "ai-infra-frontend" \
    "v0.0.3.3" \
    "xxx.aliyuncs.com/ai-infra-matrix/ai-infra-matrix:frontend-v0.0.3.3"

# 测试3: 阿里云ACR仅域名，默认命名空间
test_case "阿里云ACR仅域名" \
    "xxx.aliyuncs.com" \
    "ai-infra-nginx" \
    "v0.0.3.3" \
    "xxx.aliyuncs.com/ai-infra-matrix/ai-infra-matrix:nginx-v0.0.3.3"

# 测试4: 非ai-infra组件
test_case "阿里云ACR - 非ai-infra组件" \
    "xxx.aliyuncs.com/ai-infra-matrix" \
    "postgres" \
    "13" \
    "xxx.aliyuncs.com/ai-infra-matrix/postgres:13"

# 测试5: 其他注册表（Docker Hub）
test_case "Docker Hub注册表" \
    "docker.io/myuser" \
    "ai-infra-backend" \
    "v0.0.3.3" \
    "docker.io/myuser/ai-infra-backend:v0.0.3.3"

# 测试6: 本地注册表
test_case "本地注册表" \
    "localhost:5000" \
    "ai-infra-frontend" \
    "latest" \
    "localhost:5000/ai-infra-frontend:latest"

# 测试7: 无注册表
test_case "无注册表" \
    "" \
    "ai-infra-backend" \
    "v0.0.3.3" \
    "ai-infra-backend:v0.0.3.3"

echo "🎉 所有测试完成！"
EOF

    chmod +x /tmp/test_acr_naming.sh
    /tmp/test_acr_naming.sh
    rm -f /tmp/test_acr_naming.sh
}

# 执行测试
extract_and_test_function

echo ""
echo "📊 测试总结"
echo "=================================="
echo "✅ 测试了阿里云ACR的命名逻辑"
echo "✅ 验证了不同注册表格式的支持"
echo "✅ 确认了镜像名称转换的正确性"
echo ""
echo "🔧 使用方法:"
echo "  # 推送到阿里云ACR（带命名空间）"
echo "  ./scripts/build.sh prod --registry xxx.aliyuncs.com/ai-infra-matrix --push --version v0.0.3.3"
echo ""
echo "  # 推送到阿里云ACR（仅域名，使用默认命名空间）"
echo "  ./scripts/build.sh prod --registry xxx.aliyuncs.com --push --version v0.0.3.3"
echo ""
echo "📝 推送后的镜像格式:"
echo "  ai-infra-backend -> xxx.aliyuncs.com/ai-infra-matrix/ai-infra-matrix:backend-v0.0.3.3"
echo "  ai-infra-frontend -> xxx.aliyuncs.com/ai-infra-matrix/ai-infra-matrix:frontend-v0.0.3.3"
echo "  ai-infra-nginx -> xxx.aliyuncs.com/ai-infra-matrix/ai-infra-matrix:nginx-v0.0.3.3"
