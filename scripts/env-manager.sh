#!/bin/bash

# ====================================================================
# AI Infrastructure Matrix - 环境变量管理脚本
# ====================================================================
# 版本: v3.1.0
# 用途: 统一管理和切换环境变量配置
# 支持: Docker Compose 和 Helm 部署
# ====================================================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 网络接口配置
DEFAULT_NETWORK_INTERFACE="ens0"
FALLBACK_INTERFACES=("eth0" "enp0s3" "wlan0" "wlp2s0")

# 操作系统检测
detect_os() {
    if [[ "$OSTYPE" == "darwin"* ]]; then
        echo "macOS"
    elif [[ "$OSTYPE" == "linux-gnu"* ]] || [[ "$OSTYPE" == "linux"* ]]; then
        echo "Linux"
    elif [[ "$OSTYPE" == "msys"* ]] || [[ "$OSTYPE" == "cygwin"* ]]; then
        echo "Windows"
    else
        # 备用检测方法
        if [[ "$(uname -s)" == "Darwin" ]]; then
            echo "macOS"
        elif [[ "$(uname -s)" == "Linux" ]]; then
            echo "Linux"
        else
            echo "Other"
        fi
    fi
}

# 操作系统类型
OS_TYPE=$(detect_os)

# 项目根目录
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKUP_DIR="$PROJECT_ROOT/backup"
HELM_VALUES_FILE="$PROJECT_ROOT/helm/ai-infra-matrix/values.yaml"

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 显示帮助信息
show_help() {
    cat << EOF
AI Infrastructure Matrix - 环境变量管理脚本

用法:
    $0 [命令] [选项]

命令:
    switch <env>          切换环境 (dev|prod)
    validate             验证当前环境配置
    helm-sync            同步环境变量到 Helm values
    backup               备份当前环境配置
    restore <backup>     恢复指定备份
    compare              比较开发和生产环境差异
    detect-ip [iface]    检测IP地址 (默认检测ens0，支持--all)
    render-template      渲染环境变量模板
    help                 显示此帮助信息

选项:
    --force            强制执行，跳过确认
    --dry-run          预览模式，不执行实际操作

示例:
    $0 switch dev                       # 切换到开发环境
    $0 switch prod --force             # 强制切换到生产环境
    $0 validate                        # 验证当前配置
    $0 helm-sync                       # 同步到 Helm
    $0 compare                         # 比较环境差异
    $0 detect-ip                       # 检测ens0网卡IP地址
    $0 detect-ip eth0                  # 检测eth0网卡IP地址
    $0 detect-ip --all                 # 检测所有网卡IP地址
    $0 render-template                 # 使用自动检测的IP渲染.env文件
    $0 render-template .env.example .env 192.168.0.200 8080 http

EOF
}

# 备份函数
create_backup() {
    local backup_name="${1:-env-backup-$(date +%Y%m%d-%H%M%S)}"
    local backup_path="$BACKUP_DIR/$backup_name"
    
    log_info "创建备份: $backup_name"
    mkdir -p "$backup_path"
    
    # 备份所有环境文件
    if [[ -f "$PROJECT_ROOT/.env" ]]; then
        cp "$PROJECT_ROOT/.env" "$backup_path/.env"
    fi
    
    if [[ -f "$PROJECT_ROOT/.env.prod" ]]; then
        cp "$PROJECT_ROOT/.env.prod" "$backup_path/.env.prod"
    fi
    
    if [[ -f "$PROJECT_ROOT/.env.unified" ]]; then
        cp "$PROJECT_ROOT/.env.unified" "$backup_path/.env.unified"
    fi
    
    if [[ -f "$PROJECT_ROOT/.env.prod.unified" ]]; then
        cp "$PROJECT_ROOT/.env.prod.unified" "$backup_path/.env.prod.unified"
    fi
    
    if [[ -f "$HELM_VALUES_FILE" ]]; then
        cp "$HELM_VALUES_FILE" "$backup_path/values.yaml"
    fi
    
    # 记录备份信息
    cat > "$backup_path/backup-info.txt" << EOF
备份时间: $(date)
备份类型: 环境变量配置
项目版本: $(git describe --tags --always 2>/dev/null || echo "unknown")
分支: $(git branch --show-current 2>/dev/null || echo "unknown")
提交: $(git rev-parse HEAD 2>/dev/null || echo "unknown")
EOF
    
    log_success "备份已创建: $backup_path"
    echo "$backup_path"
}

# 切换环境函数
switch_environment() {
    local target_env="$1"
    local force_mode="$2"
    
    if [[ "$target_env" != "dev" && "$target_env" != "prod" ]]; then
        log_error "无效的环境类型: $target_env (只支持 dev 或 prod)"
        exit 1
    fi
    
    log_info "准备切换到 $target_env 环境"
    
    # 检查源文件是否存在
    local source_file
    if [[ "$target_env" == "dev" ]]; then
        source_file="$PROJECT_ROOT/.env.unified"
    else
        source_file="$PROJECT_ROOT/.env.prod.unified"
    fi
    
    if [[ ! -f "$source_file" ]]; then
        log_error "源文件不存在: $source_file"
        exit 1
    fi
    
    # 确认操作
    if [[ "$force_mode" != "--force" ]]; then
        echo
        log_warning "此操作将覆盖当前的 .env 和 .env.prod 文件"
        read -p "确认继续? (y/N): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "操作已取消"
            exit 0
        fi
    fi
    
    # 创建备份
    local backup_path
    backup_path=$(create_backup "pre-switch-$target_env-$(date +%Y%m%d-%H%M%S)")
    
    # 执行切换
    log_info "复制环境配置文件..."
    
    if [[ "$target_env" == "dev" ]]; then
        cp "$PROJECT_ROOT/.env.unified" "$PROJECT_ROOT/.env"
        # 保持生产环境文件不变，或者创建默认的
        if [[ ! -f "$PROJECT_ROOT/.env.prod" ]]; then
            cp "$PROJECT_ROOT/.env.prod.unified" "$PROJECT_ROOT/.env.prod"
        fi
    else
        cp "$PROJECT_ROOT/.env.prod.unified" "$PROJECT_ROOT/.env.prod"
        cp "$PROJECT_ROOT/.env.prod.unified" "$PROJECT_ROOT/.env"
    fi
    
    log_success "环境已切换到 $target_env"
    log_info "备份位置: $backup_path"
    
    # 验证切换结果
    validate_environment
}

# 验证环境配置
validate_environment() {
    log_info "验证环境配置..."
    
    local issues=0
    
    # 检查必需文件
    if [[ ! -f "$PROJECT_ROOT/.env" ]]; then
        log_error "缺少 .env 文件"
        ((issues++))
    fi
    
    if [[ ! -f "$PROJECT_ROOT/docker-compose.yml" ]]; then
        log_error "缺少 docker-compose.yml 文件"
        ((issues++))
    fi
    
    # 检查关键环境变量
    if [[ -f "$PROJECT_ROOT/.env" ]]; then
        local required_vars=(
            "IMAGE_TAG"
            "POSTGRES_PASSWORD"
            "REDIS_PASSWORD"
            "JWT_SECRET"
            "CONFIGPROXY_AUTH_TOKEN"
        )
        
        for var in "${required_vars[@]}"; do
            if ! grep -q "^$var=" "$PROJECT_ROOT/.env"; then
                log_warning "缺少环境变量: $var"
                ((issues++))
            fi
        done
        
        # 检查生产环境密码安全性
        if grep -q "CHANGE_IN_PRODUCTION" "$PROJECT_ROOT/.env"; then
            local build_env
            build_env=$(grep "^BUILD_ENV=" "$PROJECT_ROOT/.env" | cut -d'=' -f2)
            if [[ "$build_env" == "production" ]]; then
                log_error "生产环境仍使用默认密码，存在安全风险！"
                ((issues++))
            else
                log_warning "发现待修改的生产环境密码标记"
            fi
        fi
    fi
    
    # 检查 Docker Compose 配置
    if command -v docker-compose &> /dev/null; then
        if ! docker-compose -f "$PROJECT_ROOT/docker-compose.yml" config &> /dev/null; then
            log_error "Docker Compose 配置验证失败"
            ((issues++))
        fi
    fi
    
    # 总结验证结果
    if [[ $issues -eq 0 ]]; then
        log_success "环境配置验证通过"
        
        # 显示当前环境信息
        if [[ -f "$PROJECT_ROOT/.env" ]]; then
            local build_env
            local image_tag
            build_env=$(grep "^BUILD_ENV=" "$PROJECT_ROOT/.env" | cut -d'=' -f2 2>/dev/null || echo "unknown")
            image_tag=$(grep "^IMAGE_TAG=" "$PROJECT_ROOT/.env" | cut -d'=' -f2 2>/dev/null || echo "unknown")
            
            echo
            log_info "当前环境信息:"
            echo "  环境类型: $build_env"
            echo "  镜像版本: $image_tag"
            echo "  配置文件: .env"
        fi
    else
        log_error "发现 $issues 个配置问题"
        exit 1
    fi
}

# 同步到 Helm values
sync_to_helm() {
    log_info "同步环境变量到 Helm values.yaml..."
    
    if [[ ! -f "$PROJECT_ROOT/.env" ]]; then
        log_error ".env 文件不存在"
        exit 1
    fi
    
    # 创建备份
    if [[ -f "$HELM_VALUES_FILE" ]]; then
        cp "$HELM_VALUES_FILE" "$HELM_VALUES_FILE.backup-$(date +%Y%m%d-%H%M%S)"
    fi
    
    # 确保 helm 目录存在
    mkdir -p "$(dirname "$HELM_VALUES_FILE")"
    
    # 读取环境变量并生成 Helm values
    log_info "生成 Helm values.yaml..."
    
    cat > "$HELM_VALUES_FILE" << 'EOF'
# AI Infrastructure Matrix Helm Chart Values
# Generated from environment variables

global:
  imageRegistry: ""
  imageTag: ""
  
env:
  # This section will be populated from .env file
EOF
    
    # 从 .env 文件提取变量到 values.yaml
    while IFS='=' read -r key value; do
        # 跳过注释和空行
        [[ $key =~ ^[[:space:]]*# ]] && continue
        [[ -z $key ]] && continue
        
        # 清理值（移除引号）
        value=$(echo "$value" | sed 's/^"//;s/"$//')
        
        # 添加到 values.yaml
        echo "  $key: \"$value\"" >> "$HELM_VALUES_FILE"
        
    done < <(grep -E '^[A-Z_]+=.*' "$PROJECT_ROOT/.env")
    
    log_success "Helm values.yaml 已更新"
}

# 比较环境差异
compare_environments() {
    log_info "比较开发和生产环境差异..."
    
    if [[ ! -f "$PROJECT_ROOT/.env.unified" ]] || [[ ! -f "$PROJECT_ROOT/.env.prod.unified" ]]; then
        log_error "缺少统一配置文件"
        exit 1
    fi
    
    echo
    echo "=== 环境配置差异 ==="
    echo
    
    # 使用 diff 比较文件
    if command -v diff &> /dev/null; then
        diff -u "$PROJECT_ROOT/.env.unified" "$PROJECT_ROOT/.env.prod.unified" || true
    else
        log_warning "diff 命令不可用，跳过差异比较"
    fi
}

# 恢复备份
restore_backup() {
    local backup_name="$1"
    
    if [[ -z "$backup_name" ]]; then
        log_error "请指定备份名称"
        echo
        echo "可用备份:"
        ls -la "$BACKUP_DIR" 2>/dev/null || echo "无可用备份"
        exit 1
    fi
    
    local backup_path="$BACKUP_DIR/$backup_name"
    
    if [[ ! -d "$backup_path" ]]; then
        log_error "备份不存在: $backup_path"
        exit 1
    fi
    
    log_info "恢复备份: $backup_name"
    
    # 确认操作
    read -p "确认恢复备份? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "操作已取消"
        exit 0
    fi
    
    # 创建当前状态备份
    create_backup "pre-restore-$(date +%Y%m%d-%H%M%S)"
    
    # 恢复文件
    if [[ -f "$backup_path/.env" ]]; then
        cp "$backup_path/.env" "$PROJECT_ROOT/.env"
    fi
    
    if [[ -f "$backup_path/.env.prod" ]]; then
        cp "$backup_path/.env.prod" "$PROJECT_ROOT/.env.prod"
    fi
    
    if [[ -f "$backup_path/values.yaml" ]]; then
        cp "$backup_path/values.yaml" "$HELM_VALUES_FILE"
    fi
    
    log_success "备份已恢复"
    validate_environment
}

# ==========================================
# IP地址检测和模板渲染功能（从build.sh集成）
# ==========================================

# 跨平台sed命令包装器
sed_inplace() {
    if [[ "$OS_TYPE" == "macOS" ]]; then
        sed -i '.bak' "$@"
    else
        sed -i "$@"
    fi
}

# 清理备份文件
cleanup_backup_files() {
    local dir="${1:-.}"
    if [[ "$OS_TYPE" == "macOS" ]]; then
        find "$dir" -name "*.bak" -type f -delete 2>/dev/null || true
    fi
}

# 检测指定网卡的IP地址
detect_interface_ip() {
    local interface="${1:-$DEFAULT_NETWORK_INTERFACE}"
    local ip=""
    
    # 方法1: 使用ip命令（Linux优先）
    if command -v ip >/dev/null 2>&1; then
        ip=$(ip addr show "$interface" 2>/dev/null | grep -oP 'inet \K[0-9.]+' | head -1)
    fi
    
    # 方法2: 使用ifconfig命令（macOS和旧版Linux）
    if [[ -z "$ip" ]] && command -v ifconfig >/dev/null 2>&1; then
        case "$OS_TYPE" in
            "macOS")
                ip=$(ifconfig "$interface" 2>/dev/null | grep -E 'inet\s+[0-9.]+' | awk '{print $2}' | head -1)
                ;;
            *)
                ip=$(ifconfig "$interface" 2>/dev/null | grep -oP 'inet addr:\K[0-9.]+' | head -1)
                if [[ -z "$ip" ]]; then
                    # 新版本ifconfig格式
                    ip=$(ifconfig "$interface" 2>/dev/null | grep -oP 'inet \K[0-9.]+' | head -1)
                fi
                ;;
        esac
    fi
    
    echo "$ip"
}

# 自动检测外部主机IP
auto_detect_external_ip() {
    local detected_ip=""
    
    log_info "自动检测外部主机IP..."
    
    # 优先检测指定网卡
    detected_ip=$(detect_interface_ip "$DEFAULT_NETWORK_INTERFACE")
    
    # 如果指定网卡没有IP，尝试其他网卡
    if [[ -z "$detected_ip" ]]; then
        for interface in "${FALLBACK_INTERFACES[@]}"; do
            log_info "尝试检测网卡: $interface"
            detected_ip=$(detect_interface_ip "$interface")
            if [[ -n "$detected_ip" ]]; then
                log_success "在网卡 $interface 上检测到IP: $detected_ip"
                break
            fi
        done
    else
        log_success "在网卡 $DEFAULT_NETWORK_INTERFACE 上检测到IP: $detected_ip"
    fi
    
    # 方法3: 通过默认路由检测
    if [[ -z "$detected_ip" ]] && command -v ip >/dev/null 2>&1; then
        detected_ip=$(ip route get 8.8.8.8 2>/dev/null | sed -n 's/.*src \([0-9.]*\).*/\1/p' | head -1)
        [[ -n "$detected_ip" ]] && log_success "通过默认路由检测到IP: $detected_ip"
    fi
    
    # 方法4: 通过ifconfig检测任意可用IP（排除127.0.0.1）
    if [[ -z "$detected_ip" ]] && command -v ifconfig >/dev/null 2>&1; then
        case "$OS_TYPE" in
            "macOS")
                detected_ip=$(ifconfig | grep -E 'inet\s+[0-9.]+' | grep -v '127.0.0.1' | awk '{print $2}' | head -1)
                ;;
            *)
                detected_ip=$(ifconfig | grep -E 'inet\s+[0-9.]+' | grep -v '127.0.0.1' | awk '{print $2}' | head -1)
                ;;
        esac
        [[ -n "$detected_ip" ]] && log_success "通过ifconfig检测到IP: $detected_ip"
    fi
    
    # 备用方案: 使用localhost
    if [[ -z "$detected_ip" ]]; then
        detected_ip="localhost"
        log_warning "无法自动检测外部IP，使用默认值: localhost"
    fi
    
    echo "$detected_ip"
}

# 渲染环境变量模板
render_env_template() {
    local template_file="$1"
    local output_file="$2"
    local external_host="$3"
    local external_port="${4:-8080}"
    local external_scheme="${5:-http}"
    
    if [[ ! -f "$template_file" ]]; then
        log_error "模板文件不存在: $template_file"
        return 1
    fi
    
    log_info "渲染环境变量模板..."
    log_info "  模板文件: $template_file"
    log_info "  输出文件: $output_file"
    log_info "  外部主机: $external_host"
    log_info "  外部端口: $external_port"
    log_info "  外部协议: $external_scheme"
    
    # 读取模板内容
    local temp_content
    temp_content=$(cat "$template_file")
    
    # 替换模板变量
    temp_content="${temp_content//\$\{EXTERNAL_HOST\}/$external_host}"
    temp_content="${temp_content//\$\{EXTERNAL_PORT\}/$external_port}"
    temp_content="${temp_content//\$\{EXTERNAL_SCHEME\}/$external_scheme}"
    
    # 写入输出文件
    echo "$temp_content" > "$output_file"
    
    log_success "✓ 模板渲染完成: $output_file"
}

# IP地址检测命令
detect_ip() {
    local interface="${1:-$DEFAULT_NETWORK_INTERFACE}"
    local show_all="${2:-false}"
    
    if [[ "$show_all" == "--all" ]] || [[ "$show_all" == "-a" ]]; then
        log_info "检测所有网卡IP地址..."
        echo
        
        # 显示所有网卡信息
        local interfaces=("$DEFAULT_NETWORK_INTERFACE" "${FALLBACK_INTERFACES[@]}")
        for iface in "${interfaces[@]}"; do
            local ip
            ip=$(detect_interface_ip "$iface")
            if [[ -n "$ip" ]]; then
                echo "  $iface: $ip"
            else
                echo "  $iface: (未找到IP)"
            fi
        done
        
        echo
        log_info "自动检测结果:"
        auto_detect_external_ip
    else
        if [[ -n "$interface" ]] && [[ "$interface" != "$DEFAULT_NETWORK_INTERFACE" ]] && [[ "$interface" != "--all" ]] && [[ "$interface" != "-a" ]]; then
            # 检测指定网卡
            local ip
            ip=$(detect_interface_ip "$interface")
            if [[ -n "$ip" ]]; then
                echo "$ip"
            else
                log_warning "网卡 $interface 未找到IP地址"
                return 1
            fi
        else
            # 自动检测
            auto_detect_external_ip
        fi
    fi
}

# 渲染模板命令
render_template() {
    local template_file="${1:-$PROJECT_ROOT/.env.example}"
    local output_file="${2:-$PROJECT_ROOT/.env}"
    local external_host="$3"
    local external_port="${4:-8080}"
    local external_scheme="${5:-http}"
    
    # 如果没有指定external_host，自动检测
    if [[ -z "$external_host" ]]; then
        external_host=$(auto_detect_external_ip 2>/dev/null | tail -1)
    fi
    
    # 确保备份目录存在
    mkdir -p "$BACKUP_DIR"
    
    # 创建备份
    if [[ -f "$output_file" ]]; then
        local backup_name="$(basename "$output_file").backup-$(date +%Y%m%d-%H%M%S)"
        cp "$output_file" "$BACKUP_DIR/$backup_name"
        log_info "已备份原文件: $backup_name"
    fi
    
    # 渲染模板
    render_env_template "$template_file" "$output_file" "$external_host" "$external_port" "$external_scheme"
    
    # 清理备份文件
    cleanup_backup_files "$PROJECT_ROOT"
}

# 主函数
main() {
    cd "$PROJECT_ROOT"
    
    case "${1:-help}" in
        "switch")
            switch_environment "$2" "$3"
            ;;
        "validate")
            validate_environment
            ;;
        "helm-sync")
            sync_to_helm
            ;;
        "backup")
            create_backup
            ;;
        "restore")
            restore_backup "$2"
            ;;
        "compare")
            compare_environments
            ;;
        "detect-ip")
            detect_ip "$2" "$3"
            ;;
        "render-template")
            render_template "$2" "$3" "$4" "$5" "$6"
            ;;
        "help"|"--help"|"-h")
            show_help
            ;;
        *)
            log_error "未知命令: $1"
            echo
            show_help
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"
