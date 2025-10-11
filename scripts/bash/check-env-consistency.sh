#!/bin/bash
set -e

echo "🔍 AI-Infra-Matrix 环境变量配置统一检查和合并工具"
echo "=========================================================="

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() { echo -e "${BLUE}ℹ️  $1${NC}"; }
print_success() { echo -e "${GREEN}✅ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠️  $1${NC}"; }
print_error() { echo -e "${RED}❌ $1${NC}"; }

# 1. 检查现有环境变量文件
print_info "1. 检查现有的环境变量文件..."
env_files=$(find . -maxdepth 1 -name "*.env*" -type f | sort)
echo "发现的环境变量文件:"
for file in $env_files; do
    echo "  - $file"
done

# 2. 备份现有配置
backup_dir="backup/env-configs-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$backup_dir"
print_info "2. 备份现有环境变量文件到 $backup_dir..."
for file in $env_files; do
    if [ -f "$file" ]; then
        cp "$file" "$backup_dir/"
        print_success "已备份: $file"
    fi
done

# 3. 检查docker-compose.yml中的env_file引用
print_info "3. 检查docker-compose.yml中的环境变量引用..."
if [ -f "docker-compose.yml" ]; then
    echo "env_file 引用:"
    grep -n "env_file:" docker-compose.yml || echo "  未找到env_file引用"
    grep -n "ENV_FILE" docker-compose.yml || echo "  未找到ENV_FILE变量"
    echo ""
    echo "环境变量使用统计:"
    grep -o '\${[^}]*}' docker-compose.yml | sort | uniq -c | sort -nr | head -10
else
    print_error "未找到docker-compose.yml文件"
fi

# 4. 检查Helm values中的环境变量
print_info "4. 检查Helm Chart中的环境变量配置..."
helm_values_files=$(find helm -name "values*.yaml" 2>/dev/null || echo "")
if [ -n "$helm_values_files" ]; then
    for values_file in $helm_values_files; do
        echo "检查: $values_file"
        grep -n "environment:" "$values_file" || echo "  未找到environment配置"
    done
else
    print_warning "未找到Helm values文件"
fi

# 5. 检查Dockerfile中的环境变量
print_info "5. 检查Dockerfile中的环境变量定义..."
dockerfile_count=$(find . -name "Dockerfile*" -type f | wc -l)
echo "发现 $dockerfile_count 个Dockerfile文件"
echo "环境变量使用情况:"
find . -name "Dockerfile*" -type f -exec grep -l "ENV\|ARG" {} \; | head -5 | while read dockerfile; do
    echo "  - $dockerfile:"
    grep -E "^(ENV|ARG)" "$dockerfile" | head -3 | sed 's/^/    /'
done

echo ""
print_success "环境变量配置检查完成！"
print_info "检查结果已保存，原始文件已备份到 $backup_dir"
