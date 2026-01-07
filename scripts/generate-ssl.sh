#!/bin/bash
# =============================================================================
# AI-Infra-Matrix 自签名 SSL 证书生成工具
# 支持生成 CA 根证书、服务器证书，并自动配置 Nginx HTTPS
# =============================================================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# 日志函数
log_info()  { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step()  { echo -e "${CYAN}[STEP]${NC} $1"; }

# 默认配置
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
DEFAULT_SSL_DIR="$PROJECT_ROOT/ssl-certs"
DEFAULT_DAYS=3650  # 10年有效期
DEFAULT_KEY_SIZE=2048
DEFAULT_COUNTRY="CN"
DEFAULT_STATE="Beijing"
DEFAULT_CITY="Beijing"
DEFAULT_ORG="AI-Infra-Matrix"

# 显示帮助
usage() {
    cat << EOF
${CYAN}═══════════════════════════════════════════════════════════════════${NC}
${GREEN}AI-Infra-Matrix 自签名 SSL 证书生成工具${NC}
${CYAN}═══════════════════════════════════════════════════════════════════${NC}

用法: $0 [命令] [选项]

命令:
    ca          生成 CA 根证书
    server      生成服务器证书 (需要先生成 CA)
    quick       快速生成 (自动生成 CA + 服务器证书)
    nginx       配置 Nginx 使用生成的证书
    trust       将 CA 证书添加到系统信任 (需要 root)
    info        显示证书信息
    clean       清理所有生成的证书

选项:
    -d, --domain DOMAIN       域名 (必需，可多次指定)
    -o, --output DIR          输出目录 (默认: $DEFAULT_SSL_DIR)
    --days DAYS               证书有效期天数 (默认: $DEFAULT_DAYS)
    --key-size SIZE           密钥长度 (默认: $DEFAULT_KEY_SIZE)
    -C, --country CODE        国家代码 (默认: $DEFAULT_COUNTRY)
    -S, --state STATE         省/州 (默认: $DEFAULT_STATE)
    -L, --city CITY           城市 (默认: $DEFAULT_CITY)
    -O, --org ORG             组织名称 (默认: $DEFAULT_ORG)
    --ca-name NAME            CA 证书名称 (默认: AI-Infra-Matrix-CA)
    -h, --help                显示此帮助

示例:
    # 快速为域名生成证书
    $0 quick -d example.com -d api.example.com

    # 分步操作: 先生成 CA
    $0 ca

    # 分步操作: 再生成服务器证书
    $0 server -d example.com -d *.example.com

    # 配置 Nginx
    $0 nginx -d example.com

    # 显示证书信息
    $0 info -d example.com

EOF
}

# 解析参数
COMMAND=""
DOMAINS=()
OUTPUT_DIR="$DEFAULT_SSL_DIR"
VALID_DAYS=$DEFAULT_DAYS
KEY_SIZE=$DEFAULT_KEY_SIZE
COUNTRY=$DEFAULT_COUNTRY
STATE=$DEFAULT_STATE
CITY=$DEFAULT_CITY
ORG=$DEFAULT_ORG
CA_NAME="AI-Infra-Matrix-CA"

parse_args() {
    if [ $# -eq 0 ]; then
        usage
        exit 0
    fi

    # 处理第一个参数为 -h 或 --help 的情况
    if [[ "$1" == "-h" ]] || [[ "$1" == "--help" ]]; then
        usage
        exit 0
    fi

    COMMAND="$1"
    shift

    while [[ $# -gt 0 ]]; do
        case $1 in
            -d|--domain)
                DOMAINS+=("$2")
                shift 2
                ;;
            -o|--output)
                OUTPUT_DIR="$2"
                shift 2
                ;;
            --days)
                VALID_DAYS="$2"
                shift 2
                ;;
            --key-size)
                KEY_SIZE="$2"
                shift 2
                ;;
            -C|--country)
                COUNTRY="$2"
                shift 2
                ;;
            -S|--state)
                STATE="$2"
                shift 2
                ;;
            -L|--city)
                CITY="$2"
                shift 2
                ;;
            -O|--org)
                ORG="$2"
                shift 2
                ;;
            --ca-name)
                CA_NAME="$2"
                shift 2
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                log_error "未知选项: $1"
                usage
                exit 1
                ;;
        esac
    done
}

# 检查 openssl
check_openssl() {
    if ! command -v openssl &> /dev/null; then
        log_error "未找到 openssl，请先安装"
        exit 1
    fi
    log_info "OpenSSL 版本: $(openssl version)"
}

# 创建目录
setup_directories() {
    mkdir -p "$OUTPUT_DIR"/{ca,server,nginx}
    log_info "证书目录: $OUTPUT_DIR"
}

# 生成 CA 根证书
generate_ca() {
    log_step "生成 CA 根证书..."
    
    local ca_dir="$OUTPUT_DIR/ca"
    local ca_key="$ca_dir/ca.key"
    local ca_crt="$ca_dir/ca.crt"
    local ca_cnf="$ca_dir/ca.cnf"
    
    # 检查是否已存在
    if [ -f "$ca_crt" ]; then
        log_warn "CA 证书已存在: $ca_crt"
        read -p "是否重新生成? (y/N): " confirm
        if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
            log_info "使用现有 CA 证书"
            return 0
        fi
    fi
    
    # 创建 CA 配置文件
    cat > "$ca_cnf" << EOF
[req]
default_bits = $KEY_SIZE
prompt = no
default_md = sha256
distinguished_name = dn
x509_extensions = v3_ca

[dn]
C = $COUNTRY
ST = $STATE
L = $CITY
O = $ORG
CN = $CA_NAME

[v3_ca]
subjectKeyIdentifier = hash
authorityKeyIdentifier = keyid:always,issuer
basicConstraints = critical, CA:TRUE
keyUsage = critical, digitalSignature, cRLSign, keyCertSign
EOF

    # 生成 CA 私钥
    log_info "生成 CA 私钥..."
    openssl genrsa -out "$ca_key" $KEY_SIZE 2>/dev/null
    chmod 600 "$ca_key"
    
    # 生成 CA 证书
    log_info "生成 CA 证书..."
    openssl req -x509 -new -nodes \
        -key "$ca_key" \
        -sha256 \
        -days $VALID_DAYS \
        -out "$ca_crt" \
        -config "$ca_cnf"
    
    chmod 644 "$ca_crt"
    
    echo ""
    log_info "${GREEN}✓ CA 根证书生成成功${NC}"
    echo "  私钥: $ca_key"
    echo "  证书: $ca_crt"
    echo ""
    log_warn "请妥善保管 CA 私钥，它可以签发任何证书！"
}

# 生成服务器证书
generate_server_cert() {
    if [ ${#DOMAINS[@]} -eq 0 ]; then
        log_error "请指定至少一个域名 (-d)"
        exit 1
    fi

    local ca_dir="$OUTPUT_DIR/ca"
    local ca_key="$ca_dir/ca.key"
    local ca_crt="$ca_dir/ca.crt"
    
    # 检查 CA 是否存在
    if [ ! -f "$ca_key" ] || [ ! -f "$ca_crt" ]; then
        log_error "CA 证书不存在，请先运行: $0 ca"
        exit 1
    fi
    
    # 使用第一个域名作为主域名
    local primary_domain="${DOMAINS[0]}"
    local safe_name=$(echo "$primary_domain" | sed 's/\*/_wildcard_/g')
    
    log_step "为域名生成服务器证书: ${DOMAINS[*]}"
    
    local server_dir="$OUTPUT_DIR/server"
    local server_key="$server_dir/$safe_name.key"
    local server_csr="$server_dir/$safe_name.csr"
    local server_crt="$server_dir/$safe_name.crt"
    local server_cnf="$server_dir/$safe_name.cnf"
    local server_ext="$server_dir/$safe_name.ext"
    
    # 创建服务器证书配置
    cat > "$server_cnf" << EOF
[req]
default_bits = $KEY_SIZE
prompt = no
default_md = sha256
distinguished_name = dn
req_extensions = req_ext

[dn]
C = $COUNTRY
ST = $STATE
L = $CITY
O = $ORG
CN = $primary_domain

[req_ext]
subjectAltName = @alt_names

[alt_names]
EOF

    # 添加所有域名作为 SAN
    local i=1
    for domain in "${DOMAINS[@]}"; do
        echo "DNS.$i = $domain" >> "$server_cnf"
        ((i++))
    done
    
    # 添加 localhost 和 IP
    echo "DNS.$i = localhost" >> "$server_cnf"
    ((i++))
    echo "IP.1 = 127.0.0.1" >> "$server_cnf"
    echo "IP.2 = ::1" >> "$server_cnf"
    
    # 创建扩展配置
    cat > "$server_ext" << EOF
authorityKeyIdentifier = keyid,issuer
basicConstraints = CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth, clientAuth
subjectAltName = @alt_names

[alt_names]
EOF

    i=1
    for domain in "${DOMAINS[@]}"; do
        echo "DNS.$i = $domain" >> "$server_ext"
        ((i++))
    done
    echo "DNS.$i = localhost" >> "$server_ext"
    ((i++))
    echo "IP.1 = 127.0.0.1" >> "$server_ext"
    echo "IP.2 = ::1" >> "$server_ext"
    
    # 生成服务器私钥
    log_info "生成服务器私钥..."
    openssl genrsa -out "$server_key" $KEY_SIZE 2>/dev/null
    chmod 600 "$server_key"
    
    # 生成证书签名请求 (CSR)
    log_info "生成证书签名请求..."
    openssl req -new \
        -key "$server_key" \
        -out "$server_csr" \
        -config "$server_cnf"
    
    # 使用 CA 签发证书
    log_info "使用 CA 签发证书..."
    openssl x509 -req \
        -in "$server_csr" \
        -CA "$ca_crt" \
        -CAkey "$ca_key" \
        -CAcreateserial \
        -out "$server_crt" \
        -days $VALID_DAYS \
        -sha256 \
        -extfile "$server_ext"
    
    chmod 644 "$server_crt"
    
    # 创建证书链
    local chain_crt="$server_dir/$safe_name.chain.crt"
    cat "$server_crt" "$ca_crt" > "$chain_crt"
    
    # 复制到 nginx 目录
    cp "$server_key" "$OUTPUT_DIR/nginx/server.key"
    cp "$server_crt" "$OUTPUT_DIR/nginx/server.crt"
    cp "$chain_crt" "$OUTPUT_DIR/nginx/server.chain.crt"
    cp "$ca_crt" "$OUTPUT_DIR/nginx/ca.crt"
    
    echo ""
    log_info "${GREEN}✓ 服务器证书生成成功${NC}"
    echo "  私钥: $server_key"
    echo "  证书: $server_crt"
    echo "  证书链: $chain_crt"
    echo ""
    echo "  Nginx 配置文件目录: $OUTPUT_DIR/nginx/"
    echo ""
    
    # 显示证书信息
    show_cert_brief "$server_crt"
}

# 显示简要证书信息
show_cert_brief() {
    local cert="$1"
    echo "  证书主题: $(openssl x509 -in "$cert" -noout -subject | sed 's/subject=//')"
    echo "  有效期至: $(openssl x509 -in "$cert" -noout -enddate | sed 's/notAfter=//')"
    echo "  签发者: $(openssl x509 -in "$cert" -noout -issuer | sed 's/issuer=//')"
}

# 快速生成 (CA + 服务器证书)
quick_generate() {
    if [ ${#DOMAINS[@]} -eq 0 ]; then
        log_error "请指定至少一个域名 (-d)"
        exit 1
    fi
    
    log_step "快速生成模式: 自动创建 CA 和服务器证书"
    echo ""
    
    generate_ca
    echo ""
    generate_server_cert
    
    echo ""
    echo "═══════════════════════════════════════════════════════════════════"
    log_info "${GREEN}证书生成完成！${NC}"
    echo "═══════════════════════════════════════════════════════════════════"
    echo ""
    echo "下一步操作:"
    echo ""
    echo "  1. 配置 Nginx 使用证书:"
    echo "     $0 nginx -d ${DOMAINS[0]}"
    echo ""
    echo "  2. 将 CA 证书添加到系统信任 (可选):"
    echo "     sudo $0 trust"
    echo ""
    echo "  3. 或者手动导入 CA 证书到浏览器:"
    echo "     $OUTPUT_DIR/ca/ca.crt"
    echo ""
}

# 生成 Nginx 配置
generate_nginx_config() {
    if [ ${#DOMAINS[@]} -eq 0 ]; then
        log_error "请指定域名 (-d)"
        exit 1
    fi
    
    local primary_domain="${DOMAINS[0]}"
    local nginx_dir="$OUTPUT_DIR/nginx"
    local conf_file="$nginx_dir/${primary_domain}.conf"
    
    # 检查证书是否存在
    if [ ! -f "$nginx_dir/server.crt" ]; then
        log_error "服务器证书不存在，请先生成证书"
        exit 1
    fi
    
    log_step "生成 Nginx HTTPS 配置..."
    
    # 生成 server_name 列表
    local server_names="${DOMAINS[*]}"
    
    cat > "$conf_file" << EOF
# AI-Infra-Matrix HTTPS 配置
# 域名: $server_names
# 生成时间: $(date '+%Y-%m-%d %H:%M:%S')

# HTTP -> HTTPS 重定向
server {
    listen 80;
    listen [::]:80;
    server_name $server_names;

    # ACME challenge (用于 Let's Encrypt 续期)
    location /.well-known/acme-challenge/ {
        root /var/www/html;
        try_files \$uri =404;
    }

    # 其他请求重定向到 HTTPS
    location / {
        return 301 https://\$host\$request_uri;
    }
}

# HTTPS 配置
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name $server_names;

    # SSL 证书配置
    ssl_certificate     $nginx_dir/server.crt;
    ssl_certificate_key $nginx_dir/server.key;
    
    # SSL 优化配置
    ssl_session_timeout 1d;
    ssl_session_cache shared:SSL:50m;
    ssl_session_tickets off;

    # 现代 SSL 配置 (TLS 1.2+)
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;

    # HSTS (可选 - 仅在生产环境启用)
    # add_header Strict-Transport-Security "max-age=63072000" always;

    # OCSP Stapling (自签名证书无需此配置)
    # ssl_stapling on;
    # ssl_stapling_verify on;

    # 根目录 (根据实际项目调整)
    root /var/www/html;
    index index.html index.htm;

    # 日志
    access_log /var/log/nginx/${primary_domain}_access.log;
    error_log /var/log/nginx/${primary_domain}_error.log;

    # 反向代理示例 (AI-Infra-Matrix 后端)
    # location /api/ {
    #     proxy_pass http://127.0.0.1:8080;
    #     proxy_http_version 1.1;
    #     proxy_set_header Upgrade \$http_upgrade;
    #     proxy_set_header Connection "upgrade";
    #     proxy_set_header Host \$host;
    #     proxy_set_header X-Real-IP \$remote_addr;
    #     proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    #     proxy_set_header X-Forwarded-Proto \$scheme;
    # }

    location / {
        try_files \$uri \$uri/ /index.html;
    }
}
EOF

    echo ""
    log_info "${GREEN}✓ Nginx 配置已生成${NC}"
    echo "  配置文件: $conf_file"
    echo ""
    echo "部署步骤:"
    echo ""
    echo "  1. 复制证书到服务器:"
    echo "     scp -r $nginx_dir/* user@server:/etc/nginx/ssl/"
    echo ""
    echo "  2. 复制 Nginx 配置:"
    echo "     sudo cp $conf_file /etc/nginx/conf.d/"
    echo ""
    echo "  3. 测试并重载 Nginx:"
    echo "     sudo nginx -t && sudo nginx -s reload"
    echo ""
}

# 添加 CA 到系统信任
trust_ca() {
    local ca_crt="$OUTPUT_DIR/ca/ca.crt"
    
    if [ ! -f "$ca_crt" ]; then
        log_error "CA 证书不存在: $ca_crt"
        exit 1
    fi
    
    log_step "将 CA 证书添加到系统信任..."
    
    # 检测操作系统
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        log_info "检测到 macOS 系统"
        sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain "$ca_crt"
        log_info "CA 证书已添加到 macOS 系统钥匙串"
        
    elif [[ -f /etc/debian_version ]]; then
        # Debian/Ubuntu
        log_info "检测到 Debian/Ubuntu 系统"
        sudo cp "$ca_crt" /usr/local/share/ca-certificates/$CA_NAME.crt
        sudo update-ca-certificates
        log_info "CA 证书已添加到系统信任列表"
        
    elif [[ -f /etc/redhat-release ]]; then
        # CentOS/RHEL
        log_info "检测到 CentOS/RHEL 系统"
        sudo cp "$ca_crt" /etc/pki/ca-trust/source/anchors/$CA_NAME.crt
        sudo update-ca-trust extract
        log_info "CA 证书已添加到系统信任列表"
        
    else
        log_warn "未能识别操作系统，请手动添加 CA 证书到系统信任"
        echo "CA 证书位置: $ca_crt"
    fi
    
    echo ""
    log_info "如需在浏览器中信任证书，请手动导入:"
    echo "  Chrome: 设置 -> 隐私和安全 -> 安全 -> 管理证书 -> 授权机构"
    echo "  Firefox: 设置 -> 隐私与安全 -> 证书 -> 查看证书 -> 证书颁发机构"
    echo ""
}

# 显示证书详细信息
show_cert_info() {
    if [ ${#DOMAINS[@]} -eq 0 ]; then
        # 显示 CA 证书信息
        local ca_crt="$OUTPUT_DIR/ca/ca.crt"
        if [ -f "$ca_crt" ]; then
            echo ""
            log_info "CA 证书信息:"
            echo "═══════════════════════════════════════════════════════════════════"
            openssl x509 -in "$ca_crt" -noout -text | head -30
            echo ""
        else
            log_warn "CA 证书不存在"
        fi
    else
        # 显示服务器证书信息
        local primary_domain="${DOMAINS[0]}"
        local safe_name=$(echo "$primary_domain" | sed 's/\*/_wildcard_/g')
        local server_crt="$OUTPUT_DIR/server/$safe_name.crt"
        
        if [ -f "$server_crt" ]; then
            echo ""
            log_info "服务器证书信息 ($primary_domain):"
            echo "═══════════════════════════════════════════════════════════════════"
            openssl x509 -in "$server_crt" -noout -text
            echo ""
        else
            log_warn "服务器证书不存在: $server_crt"
        fi
    fi
}

# 清理证书
clean_certs() {
    log_warn "即将删除所有生成的证书!"
    read -p "确认删除 $OUTPUT_DIR ? (y/N): " confirm
    
    if [[ "$confirm" =~ ^[Yy]$ ]]; then
        rm -rf "$OUTPUT_DIR"
        log_info "证书目录已删除"
    else
        log_info "取消删除"
    fi
}

# 主函数
main() {
    parse_args "$@"
    
    echo ""
    echo "═══════════════════════════════════════════════════════════════════"
    echo -e "${GREEN}AI-Infra-Matrix 自签名 SSL 证书工具${NC}"
    echo "═══════════════════════════════════════════════════════════════════"
    echo ""
    
    check_openssl
    setup_directories
    
    case "$COMMAND" in
        ca)
            generate_ca
            ;;
        server)
            generate_server_cert
            ;;
        quick)
            quick_generate
            ;;
        nginx)
            generate_nginx_config
            ;;
        trust)
            trust_ca
            ;;
        info)
            show_cert_info
            ;;
        clean)
            clean_certs
            ;;
        *)
            log_error "未知命令: $COMMAND"
            usage
            exit 1
            ;;
    esac
}

main "$@"
