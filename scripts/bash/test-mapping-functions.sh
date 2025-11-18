#!/bin/bash

# AI Infrastructure Matrix - 映射功能测试脚本
# 全面测试 build.sh 中的所有映射相关功能

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BUILD_SCRIPT="$SCRIPT_DIR/../build.sh"
TEST_REGISTRY="aiharbor.msxf.local/aihpc"
TEST_TAG="v0.3.5"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印函数
print_header() {
    echo -e "${BLUE}========================================"
    echo -e "$1"
    echo -e "========================================${NC}"
}

print_test() {
    echo -e "${YELLOW}[测试] $1${NC}"
}

print_success() {
    echo -e "${GREEN}[成功] $1${NC}"
}

print_error() {
    echo -e "${RED}[错误] $1${NC}"
}

# 测试计数器
total_tests=0
passed_tests=0
failed_tests=0

run_test() {
    local test_name="$1"
    local test_command="$2"
    local expected_pattern="$3"
    
    total_tests=$((total_tests + 1))
    print_test "$test_name"
    
    # 运行测试命令并捕获输出
    local output
    local exit_code
    
    if output=$(eval "$test_command" 2>&1); then
        exit_code=0
    else
        exit_code=$?
    fi
    
    # 检查期望的模式
    if [[ -n "$expected_pattern" ]]; then
        if echo "$output" | grep -q "$expected_pattern"; then
            print_success "✓ 测试通过: $test_name"
            passed_tests=$((passed_tests + 1))
            return 0
        else
            print_error "✗ 测试失败: $test_name"
            print_error "期望包含: $expected_pattern"
            echo "实际输出:"
            echo "$output" | head -10
            failed_tests=$((failed_tests + 1))
            return 1
        fi
    else
        if [[ $exit_code -eq 0 ]]; then
            print_success "✓ 测试通过: $test_name"
            passed_tests=$((passed_tests + 1))
            return 0
        else
            print_error "✗ 测试失败: $test_name (退出码: $exit_code)"
            echo "错误输出:"
            echo "$output" | head -10
            failed_tests=$((failed_tests + 1))
            return 1
        fi
    fi
}

# 主测试函数
main() {
    print_header "AI Infrastructure Matrix 映射功能测试"
    echo "测试Registry: $TEST_REGISTRY"
    echo "测试标签: $TEST_TAG"
    echo
    
    # 检查build.sh是否存在
    if [[ ! -f "$BUILD_SCRIPT" ]]; then
        print_error "build.sh 不存在: $BUILD_SCRIPT"
        exit 1
    fi
    
    print_header "1. 生产环境配置生成测试"
    
    # 测试生产环境配置生成
    run_test "生成生产环境配置" \
        "$BUILD_SCRIPT prod-generate $TEST_REGISTRY $TEST_TAG" \
        "生产环境配置文件生成成功"
    
    # 测试生成的配置文件中的镜像映射
    if [[ -f "docker-compose.prod.yml" ]]; then
        run_test "检查PostgreSQL镜像映射" \
            "grep 'image.*postgres' docker-compose.prod.yml" \
            "aiharbor.msxf.local/library/postgres:v0.3.5"
        
        run_test "检查Redis镜像映射" \
            "grep 'image.*redis' docker-compose.prod.yml" \
            "aiharbor.msxf.local/library/redis:v0.3.5"
        
        run_test "检查MinIO镜像映射" \
            "grep 'image.*minio' docker-compose.prod.yml" \
            "aiharbor.msxf.local/minio/minio:v0.3.5"
        
        run_test "检查项目镜像映射" \
            "grep 'image.*ai-infra-backend' docker-compose.prod.yml" \
            "aiharbor.msxf.local/aihpc/ai-infra-matrix/ai-infra-backend:v0.3.5"
    else
        print_error "docker-compose.prod.yml 未生成"
        failed_tests=$((failed_tests + 1))
    fi
    
    print_header "2. 映射配置文件测试"
    
    # 测试映射配置文件存在
    run_test "检查映射配置文件" \
        "test -f config/image-mapping.conf" \
        ""
    
    # 测试映射配置内容
    if [[ -f "config/image-mapping.conf" ]]; then
        run_test "检查PostgreSQL映射配置" \
            "grep 'postgres:15-alpine' config/image-mapping.conf" \
            "library|v0.3.5"
        
        run_test "检查Redis映射配置" \
            "grep 'redis:7-alpine' config/image-mapping.conf" \
            "library|v0.3.5"
        
        run_test "检查MinIO映射配置" \
            "grep 'minio/minio:latest' config/image-mapping.conf" \
            "minio|v0.3.5"
    fi
    
    print_header "3. 映射函数测试"
    
    # 测试映射函数（通过临时脚本）
    cat > /tmp/test_mapping.sh << 'EOF'
#!/bin/bash
source ./build.sh
# 测试get_mapped_private_image函数
echo "PostgreSQL: $(get_mapped_private_image 'postgres:15-alpine' 'aiharbor.msxf.local/aihpc' 'v0.3.5')"
echo "Redis: $(get_mapped_private_image 'redis:7-alpine' 'aiharbor.msxf.local/aihpc' 'v0.3.5')"
echo "MinIO: $(get_mapped_private_image 'minio/minio:latest' 'aiharbor.msxf.local/aihpc' 'v0.3.5')"
EOF
    chmod +x /tmp/test_mapping.sh
    
    run_test "测试PostgreSQL映射函数" \
        "/tmp/test_mapping.sh | grep 'PostgreSQL:'" \
        "aiharbor.msxf.local/library/postgres:v0.3.5"
    
    run_test "测试Redis映射函数" \
        "/tmp/test_mapping.sh | grep 'Redis:'" \
        "aiharbor.msxf.local/library/redis:v0.3.5"
    
    run_test "测试MinIO映射函数" \
        "/tmp/test_mapping.sh | grep 'MinIO:'" \
        "aiharbor.msxf.local/minio/minio:v0.3.5"
    
    # 清理
    rm -f /tmp/test_mapping.sh
    
    print_header "4. 依赖镜像命令测试"
    
    # 测试依赖镜像相关命令的帮助信息
    run_test "deps-pull命令帮助" \
        "$BUILD_SCRIPT deps-pull 2>&1 || true" \
        "用法.*deps-pull"
    
    run_test "deps-push命令帮助" \
        "$BUILD_SCRIPT deps-push 2>&1 || true" \
        "用法.*deps-push"
    
    print_header "5. 版本管理测试"
    
    # 测试各种latest标签是否正确映射到v0.3.5
    if [[ -f "docker-compose.prod.yml" ]]; then
        run_test "确保没有latest标签残留" \
            "! grep -E 'image:.*:latest[^/]' docker-compose.prod.yml" \
            ""
        
        run_test "确保基础镜像使用v0.3.5" \
            "grep -E 'postgres|redis|nginx.*v0.3.5' docker-compose.prod.yml" \
            "v0.3.5"
    fi
    
    print_header "测试结果总结"
    
    echo "总测试数: $total_tests"
    echo "通过测试: $passed_tests"
    echo "失败测试: $failed_tests"
    echo
    
    if [[ $failed_tests -eq 0 ]]; then
        print_success "🎉 所有测试通过！映射功能工作正常。"
        exit 0
    else
        print_error "❌ 有 $failed_tests 个测试失败，请检查相关功能。"
        exit 1
    fi
}

# 运行测试
main "$@"
