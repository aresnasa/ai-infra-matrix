#!/bin/bash
#
# AI Infrastructure Matrix - 一键启用 HTTPS
# 
# 功能：
# - 生成自签名 SSL 证书
# - 配置环境变量启用 HTTPS
# - 重建并重启 nginx 容器
#
# 使用方法：
#   ./scripts/enable-https.sh                    # 使用默认域名 ai-infra.local
#   ./scripts/enable-https.sh your-domain.com   # 使用指定域名
#   ./scripts/enable-https.sh --disable         # 禁用 HTTPS，恢复 HTTP
#   ./scripts/enable-https.sh --status          # 查看当前状态
#

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 脚本目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
ENV_FILE="${PROJECT_ROOT}/.env"

# 默认配置
DEFAULT_DOMAIN="ai-infra.local"
SSL_CERT_DIR="${PROJECT_ROOT}/ssl-certs/nginx"
HTTPS_PORT="${HTTPS_PORT:-443}"

# 打印信息函数
info() { echo -e "${BLUE}[INFO]${NC} $1"; }
success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

# 显示帮助信息
show_help() {
    cat << EOF
AI Infrastructure Matrix - HTTPS 配置工具

用法:
  $0 [选项] [域名]

选项:
  --enable, -e [域名]    启用 HTTPS (默认域名: ${DEFAULT_DOMAIN})
  --disable, -d          禁用 HTTPS，恢复 HTTP 模式
  --status, -s           显示当前 HTTPS 状态
  --regenerate, -r       重新生成 SSL 证书
  --help, -h             显示此帮助信息

示例:
  $0                           # 使用默认域名启用 HTTPS
  $0 my-domain.com             # 使用指定域名启用 HTTPS
  $0 --disable                 # 禁用 HTTPS
  $0 --status                  # 查看状态
  $0 --regenerate my-domain.com # 重新生成指定域名的证书

环境变量:
  HTTPS_PORT      HTTPS 端口 (默认: 443)
  SSL_CERT_DIR    证书目录 (默认: ./ssl-certs/nginx)

EOF
}

# 显示当前状态
show_status() {
    info "当前 HTTPS 配置状态:"
    echo ""
    
    # 检查 .env 文件
    if [ -f "$ENV_FILE" ]; then
        ENABLE_TLS=$(grep "^ENABLE_TLS=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2 || echo "false")
        EXTERNAL_SCHEME=$(grep "^EXTERNAL_SCHEME=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2 || echo "http")
        CURRENT_HTTPS_PORT=$(grep "^HTTPS_PORT=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2 || echo "8443")
        
        if [ "$ENABLE_TLS" = "true" ]; then
            success "HTTPS 状态: 已启用"
        else
            warn "HTTPS 状态: 未启用"
        fi
        echo "   协议: ${EXTERNAL_SCHEME}"
        echo "   HTTPS 端口: ${CURRENT_HTTPS_PORT}"
    else
        warn "未找到 .env 文件"
    fi
    
    echo ""
    
    # 检查证书文件
    if [ -f "${SSL_CERT_DIR}/server.crt" ] && [ -f "${SSL_CERT_DIR}/server.key" ]; then
        success "SSL 证书: 已存在"
        CERT_SUBJECT=$(openssl x509 -in "${SSL_CERT_DIR}/server.crt" -noout -subject 2>/dev/null | sed 's/subject=//' || echo "未知")
        CERT_EXPIRE=$(openssl x509 -in "${SSL_CERT_DIR}/server.crt" -noout -enddate 2>/dev/null | sed 's/notAfter=//' || echo "未知")
        echo "   主题: ${CERT_SUBJECT}"
        echo "   过期时间: ${CERT_EXPIRE}"
    else
        warn "SSL 证书: 不存在"
        echo "   证书目录: ${SSL_CERT_DIR}"
    fi
    
    echo ""
    
    # 检查 nginx 容器状态
    if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "ai-infra-nginx"; then
        NGINX_STATUS=$(docker inspect --format='{{.State.Status}}' ai-infra-nginx 2>/dev/null || echo "unknown")
        if [ "$NGINX_STATUS" = "running" ]; then
            success "Nginx 容器: 运行中"
        else
            warn "Nginx 容器: ${NGINX_STATUS}"
        fi
    else
        warn "Nginx 容器: 未运行"
    fi
}

# 更新或添加 .env 变量
update_env() {
    local key="$1"
    local value="$2"
    
    if [ ! -f "$ENV_FILE" ]; then
        echo "${key}=${value}" > "$ENV_FILE"
    elif grep -q "^${key}=" "$ENV_FILE"; then
        # macOS 兼容的 sed 语法
        if [[ "$OSTYPE" == "darwin"* ]]; then
            sed -i '' "s|^${key}=.*|${key}=${value}|" "$ENV_FILE"
        else
            sed -i "s|^${key}=.*|${key}=${value}|" "$ENV_FILE"
        fi
    else
        echo "${key}=${value}" >> "$ENV_FILE"
    fi
}

# 启用 HTTPS
enable_https() {
    local domain="${1:-$DEFAULT_DOMAIN}"
    
    info "启用 HTTPS 模式..."
    info "域名: ${domain}"
    echo ""
    
    # 步骤 1: 生成 SSL 证书
    if [ ! -f "${SSL_CERT_DIR}/server.crt" ] || [ ! -f "${SSL_CERT_DIR}/server.key" ]; then
        info "步骤 1/4: 生成 SSL 证书..."
        
        if [ -f "${SCRIPT_DIR}/generate-ssl.sh" ]; then
            bash "${SCRIPT_DIR}/generate-ssl.sh" quick "$domain"
        else
            error "找不到 generate-ssl.sh 脚本"
        fi
    else
        info "步骤 1/4: SSL 证书已存在，跳过生成"
        CERT_CN=$(openssl x509 -in "${SSL_CERT_DIR}/server.crt" -noout -subject 2>/dev/null | grep -o "CN = [^,]*" | sed 's/CN = //' || echo "")
        if [ -n "$CERT_CN" ] && [ "$CERT_CN" != "$domain" ]; then
            warn "现有证书域名 (${CERT_CN}) 与指定域名 (${domain}) 不匹配"
            read -p "是否重新生成证书? [y/N]: " regenerate
            if [ "$regenerate" = "y" ] || [ "$regenerate" = "Y" ]; then
                bash "${SCRIPT_DIR}/generate-ssl.sh" quick "$domain"
            fi
        fi
    fi
    echo ""
    
    # 步骤 2: 更新 .env 文件
    info "步骤 2/4: 更新环境变量..."
    update_env "ENABLE_TLS" "true"
    update_env "EXTERNAL_SCHEME" "https"
    update_env "HTTPS_PORT" "${HTTPS_PORT}"
    update_env "SSL_CERT_DIR" "./ssl-certs/nginx"
    success "环境变量已更新"
    echo ""
    
    # 步骤 3: 重建 nginx 镜像
    info "步骤 3/4: 重建 nginx 镜像..."
    cd "$PROJECT_ROOT"
    docker compose build nginx || docker-compose build nginx
    success "nginx 镜像重建完成"
    echo ""
    
    # 步骤 4: 重启 nginx 容器
    info "步骤 4/4: 重启 nginx 容器..."
    docker compose up -d nginx || docker-compose up -d nginx
    success "nginx 容器已重启"
    echo ""
    
    # 完成
    echo "=============================================="
    success "HTTPS 已成功启用!"
    echo "=============================================="
    echo ""
    info "访问地址: https://${domain}:${HTTPS_PORT}"
    echo ""
    warn "注意: 使用自签名证书时，浏览器会显示安全警告"
    info "信任证书方法:"
    echo "   macOS: ${SCRIPT_DIR}/generate-ssl.sh trust"
    echo "   手动:  将 ${SSL_CERT_DIR}/ca.crt 添加到系统信任的根证书"
}

# 禁用 HTTPS
disable_https() {
    info "禁用 HTTPS 模式..."
    echo ""
    
    # 步骤 1: 更新 .env 文件
    info "步骤 1/3: 更新环境变量..."
    update_env "ENABLE_TLS" "false"
    update_env "EXTERNAL_SCHEME" "http"
    success "环境变量已更新"
    echo ""
    
    # 步骤 2: 重建 nginx 镜像
    info "步骤 2/3: 重建 nginx 镜像..."
    cd "$PROJECT_ROOT"
    docker compose build nginx || docker-compose build nginx
    success "nginx 镜像重建完成"
    echo ""
    
    # 步骤 3: 重启 nginx 容器
    info "步骤 3/3: 重启 nginx 容器..."
    docker compose up -d nginx || docker-compose up -d nginx
    success "nginx 容器已重启"
    echo ""
    
    # 完成
    echo "=============================================="
    success "已恢复 HTTP 模式"
    echo "=============================================="
    echo ""
    
    EXTERNAL_HOST=$(grep "^EXTERNAL_HOST=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2 || echo "localhost")
    EXTERNAL_PORT=$(grep "^EXTERNAL_PORT=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2 || echo "80")
    info "访问地址: http://${EXTERNAL_HOST}:${EXTERNAL_PORT}"
}

# 重新生成证书
regenerate_cert() {
    local domain="${1:-$DEFAULT_DOMAIN}"
    
    info "重新生成 SSL 证书..."
    info "域名: ${domain}"
    echo ""
    
    # 删除旧证书
    rm -rf "${PROJECT_ROOT}/ssl-certs"
    
    # 生成新证书
    bash "${SCRIPT_DIR}/generate-ssl.sh" quick "$domain"
    
    # 如果 HTTPS 已启用，重启 nginx
    ENABLE_TLS=$(grep "^ENABLE_TLS=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2 || echo "false")
    if [ "$ENABLE_TLS" = "true" ]; then
        info "重启 nginx 容器以应用新证书..."
        cd "$PROJECT_ROOT"
        docker compose restart nginx || docker-compose restart nginx
        success "nginx 已重启"
    fi
    
    success "证书重新生成完成"
}

# 主逻辑
main() {
    case "${1:-}" in
        --help|-h)
            show_help
            ;;
        --status|-s)
            show_status
            ;;
        --disable|-d)
            disable_https
            ;;
        --regenerate|-r)
            regenerate_cert "${2:-$DEFAULT_DOMAIN}"
            ;;
        --enable|-e)
            enable_https "${2:-$DEFAULT_DOMAIN}"
            ;;
        "")
            enable_https "$DEFAULT_DOMAIN"
            ;;
        -*)
            error "未知选项: $1\n使用 --help 查看帮助"
            ;;
        *)
            # 直接传入域名
            enable_https "$1"
            ;;
    esac
}

main "$@"
