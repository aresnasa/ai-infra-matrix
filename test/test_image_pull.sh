#!/bin/bash

# AI-Infra-Matrix 镜像拉取功能测试脚本
# 用于验证 build.sh 的镜像拉取功能

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_info "开始测试 AI-Infra-Matrix 镜像拉取功能"
echo "======================================"

# 测试1: 帮助信息中是否包含 --pull 参数
print_info "测试1: 检查帮助信息是否包含 --pull 参数"
if scripts/build.sh --help | grep -q "\-\-pull"; then
    print_success "帮助信息包含 --pull 参数"
else
    print_error "帮助信息缺少 --pull 参数"
    exit 1
fi

# 测试2: 无注册表的拉取命令应该报错
print_info "测试2: 检查无注册表时是否正确报错"
if scripts/build.sh prod --pull --version v0.3.8 2>&1 | grep -q "拉取镜像需要指定 --registry 参数"; then
    print_success "无注册表时正确报错"
else
    print_error "无注册表时错误处理不正确"
    exit 1
fi

# 测试3: 拉取模式检测
print_info "测试3: 检查拉取模式是否正确识别"
output=$(scripts/build.sh prod --registry test.com --pull --version v0.3.8 2>&1 || true)
if echo "$output" | grep -q "镜像拉取模式"; then
    print_success "拉取模式正确识别"
else
    print_error "拉取模式识别失败"
    echo "输出: $output"
    exit 1
fi

# 测试4: 版本参数传递
print_info "测试4: 检查版本参数是否正确传递"
output=$(scripts/build.sh prod --registry test.com --pull --version v1.2.3 2>&1 || true)
if echo "$output" | grep -q "镜像版本: v1.2.3"; then
    print_success "版本参数正确传递"
else
    print_error "版本参数传递失败"
    exit 1
fi

# 测试5: 语法检查
print_info "测试5: 进行脚本语法检查"
if bash -n scripts/build.sh; then
    print_success "脚本语法检查通过"
else
    print_error "脚本语法错误"
    exit 1
fi

echo ""
print_success "🎉 所有测试通过!"
echo "======================================"
print_info "现在您可以安全地使用镜像拉取功能:"
echo "  scripts/build.sh prod --registry YOUR_REGISTRY --pull --version VERSION"
echo ""
print_info "示例命令:"
echo "  scripts/build.sh prod --registry crpi-jl2i63tqhvx30nje.cn-chengdu.personal.cr.aliyuncs.com/ai-infra-matrix --pull --version v0.3.8"
