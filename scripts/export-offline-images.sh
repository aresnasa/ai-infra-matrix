#!/bin/bash

# AI Infrastructure Matrix - 离线环境镜像导出脚本
# 版本: v1.0.0
# 功能: 导出所有必需的Docker镜像到tar文件，用于离线环境部署

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 打印函数
print_info() { echo -e "${BLUE}ℹ️  $1${NC}"; }
print_success() { echo -e "${GREEN}✅ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠️  $1${NC}"; }
print_error() { echo -e "${RED}❌ $1${NC}"; }

# 配置参数
VERSION="${IMAGE_TAG:-v0.3.6-dev}"
EXPORT_DIR="${1:-./offline-images}"
COMPRESS="${2:-yes}"

# 所有需要的镜像列表
THIRD_PARTY_IMAGES=(
    "postgres:15-alpine"
    "redis:7-alpine"
    "confluentinc/cp-kafka:7.5.0"
    "provectuslabs/kafka-ui:latest"
    "osixia/openldap:stable"
    "osixia/phpldapadmin:stable"
    "tecnativa/tcp-proxy"
    "redislabs/redisinsight:latest"
    "minio/minio:latest"
)

AI_INFRA_IMAGES=(
    "ai-infra-backend-init:${VERSION}"
    "ai-infra-backend:${VERSION}"
    "ai-infra-frontend:${VERSION}"
    "ai-infra-jupyterhub:${VERSION}"
    "ai-infra-singleuser:${VERSION}"
    "ai-infra-saltstack:${VERSION}"
    "ai-infra-nginx:${VERSION}"
    "ai-infra-gitea:${VERSION}"
)

# 创建导出目录
prepare_export_dir() {
    if [ -d "$EXPORT_DIR" ]; then
        print_warning "导出目录已存在: $EXPORT_DIR"
        read -p "是否清空并继续? (y/N): " -r
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            rm -rf "$EXPORT_DIR"
        else
            print_info "取消操作"
            exit 0
        fi
    fi
    
    mkdir -p "$EXPORT_DIR"
    print_success "创建导出目录: $EXPORT_DIR"
}

# 检查镜像是否存在
check_image_exists() {
    local image="$1"
    if docker image inspect "$image" >/dev/null 2>&1; then
        return 0
    else
        return 1
    fi
}

# 拉取缺失的镜像
pull_missing_images() {
    print_info "检查并拉取缺失的镜像..."
    
    local missing_images=()
    local all_images=("${THIRD_PARTY_IMAGES[@]}" "${AI_INFRA_IMAGES[@]}")
    
    for image in "${all_images[@]}"; do
        if ! check_image_exists "$image"; then
            missing_images+=("$image")
        fi
    done
    
    if [ ${#missing_images[@]} -eq 0 ]; then
        print_success "所有镜像都已存在"
        return 0
    fi
    
    print_warning "发现 ${#missing_images[@]} 个缺失镜像:"
    for image in "${missing_images[@]}"; do
        echo "  - $image"
    done
    
    # 分别处理第三方镜像和AI-Infra镜像
    local third_party_missing=()
    local ai_infra_missing=()
    
    for image in "${missing_images[@]}"; do
        if [[ "$image" == ai-infra-* ]]; then
            ai_infra_missing+=("$image")
        else
            third_party_missing+=("$image")
        fi
    done
    
    # 拉取第三方镜像
    if [ ${#third_party_missing[@]} -gt 0 ]; then
        print_info "拉取第三方镜像..."
        for image in "${third_party_missing[@]}"; do
            print_info "拉取: $image"
            if docker pull "$image"; then
                print_success "拉取成功: $image"
            else
                print_error "拉取失败: $image"
                exit 1
            fi
        done
    fi
    
    # AI-Infra镜像需要先构建
    if [ ${#ai_infra_missing[@]} -gt 0 ]; then
        print_warning "发现缺失的AI-Infra镜像，需要先构建:"
        for image in "${ai_infra_missing[@]}"; do
            echo "  - $image"
        done
        
        print_info "正在构建AI-Infra镜像..."
        if [ -f "./build.sh" ]; then
            ./build.sh prod --version "$VERSION"
        else
            print_error "找不到build.sh脚本，请先构建AI-Infra镜像"
            exit 1
        fi
    fi
}

# 导出镜像到tar文件
export_images() {
    print_info "开始导出镜像..."
    
    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local all_images=("${THIRD_PARTY_IMAGES[@]}" "${AI_INFRA_IMAGES[@]}")
    
    # 分类导出
    export_image_set "third-party" "${THIRD_PARTY_IMAGES[@]}"
    export_image_set "ai-infra" "${AI_INFRA_IMAGES[@]}"
    
    # 创建完整的导出包
    local full_export="${EXPORT_DIR}/ai-infra-matrix-complete-${VERSION}-${timestamp}.tar"
    print_info "创建完整镜像包: $(basename "$full_export")"
    
    if docker save "${all_images[@]}" -o "$full_export"; then
        local file_size=$(du -h "$full_export" | cut -f1)
        print_success "完整镜像包导出成功: $file_size"
        
        # 压缩镜像包
        if [ "$COMPRESS" = "yes" ]; then
            print_info "压缩镜像包..."
            if gzip "$full_export"; then
                local compressed_size=$(du -h "${full_export}.gz" | cut -f1)
                print_success "压缩完成: ${compressed_size}"
            else
                print_warning "压缩失败，保留未压缩版本"
            fi
        fi
    else
        print_error "完整镜像包导出失败"
        exit 1
    fi
}

# 导出指定镜像集合
export_image_set() {
    local set_name="$1"
    shift
    local images=("$@")
    
    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local export_file="${EXPORT_DIR}/ai-infra-${set_name}-${VERSION}-${timestamp}.tar"
    
    print_info "导出${set_name}镜像集 (${#images[@]}个镜像)..."
    
    if docker save "${images[@]}" -o "$export_file"; then
        local file_size=$(du -h "$export_file" | cut -f1)
        print_success "${set_name}镜像集导出成功: $file_size"
        
        # 压缩
        if [ "$COMPRESS" = "yes" ]; then
            if gzip "$export_file"; then
                local compressed_size=$(du -h "${export_file}.gz" | cut -f1)
                print_info "${set_name}镜像集压缩完成: ${compressed_size}"
            fi
        fi
    else
        print_error "${set_name}镜像集导出失败"
        exit 1
    fi
}

# 生成镜像清单
generate_manifest() {
    local manifest_file="${EXPORT_DIR}/image-manifest.txt"
    print_info "生成镜像清单: $(basename "$manifest_file")"
    
    cat > "$manifest_file" << EOF
# AI Infrastructure Matrix - 镜像清单
# 生成时间: $(date)
# 版本: $VERSION
# 导出目录: $EXPORT_DIR

## 第三方镜像 (${#THIRD_PARTY_IMAGES[@]}个)
EOF
    
    for image in "${THIRD_PARTY_IMAGES[@]}"; do
        echo "$image" >> "$manifest_file"
    done
    
    cat >> "$manifest_file" << EOF

## AI-Infra镜像 (${#AI_INFRA_IMAGES[@]}个)
EOF
    
    for image in "${AI_INFRA_IMAGES[@]}"; do
        echo "$image" >> "$manifest_file"
    done
    
    # 添加镜像详细信息
    cat >> "$manifest_file" << EOF

## 镜像详细信息
EOF
    
    local all_images=("${THIRD_PARTY_IMAGES[@]}" "${AI_INFRA_IMAGES[@]}")
    for image in "${all_images[@]}"; do
        if check_image_exists "$image"; then
            echo "### $image" >> "$manifest_file"
            docker image inspect "$image" --format "SIZE: {{.Size}} bytes ($(docker images "$image" --format "{{.Size}}"))" >> "$manifest_file"
            docker image inspect "$image" --format "CREATED: {{.Created}}" >> "$manifest_file"
            echo "" >> "$manifest_file"
        fi
    done
    
    print_success "镜像清单生成完成"
}

# 生成导入脚本
generate_import_script() {
    local import_script="${EXPORT_DIR}/import-images.sh"
    print_info "生成镜像导入脚本: $(basename "$import_script")"
    
    cat > "$import_script" << 'EOF'
#!/bin/bash

# AI Infrastructure Matrix - 离线镜像导入脚本
# 自动生成，用于在离线环境中导入镜像

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() { echo -e "${BLUE}ℹ️  $1${NC}"; }
print_success() { echo -e "${GREEN}✅ $1${NC}"; }
print_error() { echo -e "${RED}❌ $1${NC}"; }

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

print_info "AI Infrastructure Matrix - 镜像导入"
print_info "导入目录: $SCRIPT_DIR"

# 查找镜像文件
image_files=($(find "$SCRIPT_DIR" -name "*.tar" -o -name "*.tar.gz" | sort))

if [ ${#image_files[@]} -eq 0 ]; then
    print_error "未找到镜像文件 (.tar 或 .tar.gz)"
    exit 1
fi

print_info "找到 ${#image_files[@]} 个镜像文件:"
for file in "${image_files[@]}"; do
    echo "  - $(basename "$file")"
done

# 导入镜像
for file in "${image_files[@]}"; do
    print_info "导入: $(basename "$file")"
    
    if [[ "$file" == *.gz ]]; then
        # 解压并导入
        if gunzip -c "$file" | docker load; then
            print_success "导入成功: $(basename "$file")"
        else
            print_error "导入失败: $(basename "$file")"
            exit 1
        fi
    else
        # 直接导入
        if docker load -i "$file"; then
            print_success "导入成功: $(basename "$file")"
        else
            print_error "导入失败: $(basename "$file")"
            exit 1
        fi
    fi
done

print_success "所有镜像导入完成!"
print_info "查看已导入的镜像:"
docker images | grep -E "(ai-infra-|postgres|redis|kafka|ldap|minio)" || true
EOF
    
    chmod +x "$import_script"
    print_success "镜像导入脚本生成完成"
}

# 显示使用说明
show_usage() {
    cat << EOF
AI Infrastructure Matrix - 离线环境镜像导出脚本

用法: $0 [导出目录] [是否压缩]

参数:
  导出目录      导出镜像的目标目录 (默认: ./offline-images)
  是否压缩      是否压缩导出的镜像 (yes/no, 默认: yes)

示例:
  $0                              # 使用默认设置
  $0 /tmp/images                  # 指定导出目录
  $0 /tmp/images no               # 不压缩镜像

功能:
  ✅ 自动检查并拉取缺失镜像
  ✅ 分类导出第三方镜像和AI-Infra镜像
  ✅ 生成完整镜像包
  ✅ 自动压缩镜像文件
  ✅ 生成镜像清单和导入脚本

导出后的文件:
  📦 ai-infra-third-party-*.tar.gz    # 第三方镜像
  📦 ai-infra-ai-infra-*.tar.gz       # AI-Infra镜像
  📦 ai-infra-matrix-complete-*.tar.gz # 完整镜像包
  📋 image-manifest.txt               # 镜像清单
  🔧 import-images.sh                 # 导入脚本
EOF
}

# 主函数
main() {
    echo "🐋 AI Infrastructure Matrix - 离线环境镜像导出"
    echo "================================================="
    print_info "版本: $VERSION"
    print_info "导出目录: $EXPORT_DIR"
    print_info "压缩选项: $COMPRESS"
    echo ""
    
    # 检查Docker
    if ! command -v docker &> /dev/null; then
        print_error "Docker 未安装或不可用"
        exit 1
    fi
    
    # 显示帮助
    if [ "$1" = "--help" ] || [ "$1" = "-h" ]; then
        show_usage
        exit 0
    fi
    
    # 执行导出流程
    prepare_export_dir
    pull_missing_images
    export_images
    generate_manifest
    generate_import_script
    
    # 显示结果
    echo ""
    echo "🎉 镜像导出完成!"
    echo "================================================="
    print_success "导出目录: $EXPORT_DIR"
    print_info "导出文件:"
    ls -lh "$EXPORT_DIR"
    echo ""
    print_info "💡 使用方法:"
    echo "1. 将整个 $EXPORT_DIR 目录复制到目标服务器"
    echo "2. 运行: cd $EXPORT_DIR && ./import-images.sh"
    echo "3. 运行: ./offline-start.sh (如果已生成)"
}

# 执行主函数
main "$@"