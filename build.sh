#!/bin/bash
set -e

# ==============================================================================
# AI Infrastructure Matrix - Refactored Build Script
# ==============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENV_FILE="$SCRIPT_DIR/.env"
ENV_EXAMPLE="$SCRIPT_DIR/.env.example"
SRC_DIR="$SCRIPT_DIR/src"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# ==============================================================================
# 1. Configuration & Environment
# ==============================================================================

# 检测外部主机地址 (智能检测真实网络出口IP)
# 支持 Linux (ip addr) 和 macOS (ifconfig)
# 自动过滤 loopback、docker、k8s、虚拟网桥等非物理接口
detect_external_host() {
    local detected_ip=""
    
    # 方法1：使用默认路由检测出口IP (最准确的方法)
    # Linux: ip route get 1.1.1.1
    # 这个方法直接获取访问外网时使用的源IP
    if command -v ip &> /dev/null; then
        detected_ip=$(ip route get 1.1.1.1 2>/dev/null | grep -oP 'src \K[0-9.]+' | head -n1)
        # 备用方法：ip route get 1
        if [[ -z "$detected_ip" ]]; then
            detected_ip=$(ip route get 1 2>/dev/null | awk '{for(i=1;i<=NF;i++) if($i=="src") print $(i+1)}' | head -n1)
        fi
    fi
    
    # 方法2：使用 ip addr 枚举接口 (Linux 备用)
    if [[ -z "$detected_ip" ]] && command -v ip &> /dev/null; then
        detected_ip=$(ip -4 addr show scope global 2>/dev/null | \
            grep -v -E "(docker|veth|br-|cni|flannel|calico|weave|kube|virbr|vboxnet|vmnet|tun|tap|lo:)" | \
            grep "inet " | \
            awk '{print $2}' | cut -d'/' -f1 | \
            grep -v -E "^(127\.|10\.96\.|10\.244\.|172\.17\.|172\.18\.|172\.19\.|192\.168\.49\.)" | \
            head -n1)
    fi
    
    # 方法3：使用 ifconfig (macOS/BSD)
    if [[ -z "$detected_ip" ]] && command -v ifconfig &> /dev/null; then
        # macOS: 优先检测 en0 (通常是主网卡)
        if [[ "$OSTYPE" == "darwin"* ]]; then
            detected_ip=$(ifconfig en0 2>/dev/null | grep "inet " | awk '{print $2}')
            # 如果 en0 没有IP，尝试其他接口
            if [[ -z "$detected_ip" ]]; then
                detected_ip=$(ifconfig | awk '
                    /^[a-z0-9]+:/ { iface=$1; sub(/:/, "", iface) }
                    /inet / && !/127\.0\.0\.1/ {
                        # 排除虚拟接口
                        if (iface !~ /^(lo|docker|veth|br|vmnet|vboxnet|tun|tap|virbr|utun|bridge|awdl|llw)/)
                            print $2
                    }' | head -n1)
            fi
        else
            # 其他 BSD 系统
            detected_ip=$(ifconfig | awk '
                /^[a-z0-9]+:/ { iface=$1; sub(/:/, "", iface) }
                /inet / && !/127\.0\.0\.1/ {
                    if (iface !~ /^(lo|docker|veth|br|vmnet|vboxnet|tun|tap|virbr)/)
                        print $2
                }' | \
                grep -v -E "^(10\.96\.|10\.244\.|172\.17\.|172\.18\.|172\.19\.)" | \
                head -n1)
        fi
    fi
    
    # 方法4：从现有 .env 读取（如果已配置且不是自引用）
    if [[ -z "$detected_ip" ]] && [[ -f "$ENV_FILE" ]]; then
        local env_ip=$(grep "^EXTERNAL_HOST=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2)
        # 忽略自引用、空值和 localhost
        if [[ -n "$env_ip" ]] && [[ ! "$env_ip" =~ \$\{ ]] && [[ "$env_ip" != "localhost" ]]; then
            detected_ip="$env_ip"
        fi
    fi
    
    # 方法5：hostname -I (Linux 最后备用)
    if [[ -z "$detected_ip" ]] && command -v hostname &> /dev/null; then
        detected_ip=$(hostname -I 2>/dev/null | awk '{print $1}')
    fi
    
    # 返回检测到的IP或默认值
    echo "${detected_ip:-localhost}"
}

# 更新 .env 文件中的变量
# 用法: update_env_variable "VAR_NAME" "var_value"
update_env_variable() {
    local var_name="$1"
    local var_value="$2"
    local env_file="$ENV_FILE"
    
    if [[ ! -f "$env_file" ]]; then
        return 1
    fi
    
    # 检查变量是否已存在
    if grep -q "^${var_name}=" "$env_file"; then
        # 更新现有变量 (macOS 兼容)
        if [[ "$OSTYPE" == "darwin"* ]]; then
            sed -i '' "s|^${var_name}=.*|${var_name}=${var_value}|" "$env_file"
        else
            sed -i "s|^${var_name}=.*|${var_name}=${var_value}|" "$env_file"
        fi
    else
        # 添加新变量
        echo "${var_name}=${var_value}" >> "$env_file"
    fi
}

# 同步 .env 与 .env.example 中的缺失变量
# 用法: sync_env_with_example
sync_env_with_example() {
    local env_file="$ENV_FILE"
    local example_file="$ENV_EXAMPLE"
    
    if [[ ! -f "$example_file" ]]; then
        log_error ".env.example not found: $example_file"
        return 1
    fi
    
    if [[ ! -f "$env_file" ]]; then
        log_info "Creating .env from .env.example..."
        cp "$example_file" "$env_file"
        return 0
    fi
    
    local missing_vars=()
    local updated_vars=()
    
    # 读取 .env.example 中的所有变量，同步缺失的变量到 .env
    while IFS= read -r line || [[ -n "$line" ]]; do
        # 跳过注释和空行
        if [[ "$line" =~ ^[[:space:]]*# ]] || [[ -z "$line" ]] || [[ "$line" =~ ^[[:space:]]*$ ]]; then
            continue
        fi
        
        # 提取变量名和值
        if [[ "$line" =~ ^([A-Za-z_][A-Za-z0-9_]*)=(.*)$ ]]; then
            local var_name="${BASH_REMATCH[1]}"
            local example_value="${BASH_REMATCH[2]}"
            
            # 检查 .env 中是否存在该变量
            if ! grep -q "^${var_name}=" "$env_file" 2>/dev/null; then
                # 变量不存在，添加到文件末尾
                echo "${var_name}=${example_value}" >> "$env_file"
                missing_vars+=("$var_name")
            else
                # 变量存在，检查是否为空值
                local current_value
                current_value=$(grep "^${var_name}=" "$env_file" | head -1 | cut -d'=' -f2-)
                
                # 如果当前值为空且 example 有默认值，则更新
                if [[ -z "$current_value" ]] && [[ -n "$example_value" ]]; then
                    update_env_variable "$var_name" "$example_value"
                    updated_vars+=("$var_name")
                fi
            fi
        fi
    done < "$example_file"
    
    # 显示同步结果
    if [[ ${#missing_vars[@]} -gt 0 ]]; then
        log_info "Added ${#missing_vars[@]} new variables from .env.example:"
        for var in "${missing_vars[@]}"; do
            log_info "  + $var"
        done
    fi
    
    if [[ ${#updated_vars[@]} -gt 0 ]]; then
        log_info "Updated ${#updated_vars[@]} empty variables with defaults:"
        for var in "${updated_vars[@]}"; do
            log_info "  ↻ $var"
        done
    fi
    
    if [[ ${#missing_vars[@]} -eq 0 ]] && [[ ${#updated_vars[@]} -eq 0 ]]; then
        log_info "✓ .env is in sync with .env.example"
    else
        log_info "✓ Synced ${#missing_vars[@]} new + ${#updated_vars[@]} updated variables"
    fi
    
    return 0
}

# 初始化或同步 .env 文件
# 自动检测 EXTERNAL_HOST 等关键变量
init_env_file() {
    local force="${1:-false}"
    
    # 检测外部地址
    local detected_host=$(detect_external_host)
    local detected_port="${EXTERNAL_PORT:-8080}"
    local detected_scheme="${EXTERNAL_SCHEME:-http}"
    
    if [[ ! -f "$ENV_FILE" ]]; then
        log_info "Creating .env from .env.example..."
        if [[ -f "$ENV_EXAMPLE" ]]; then
            cp "$ENV_EXAMPLE" "$ENV_FILE"
        else
            log_error ".env.example not found!"
            return 1
        fi
        force="true"
    fi
    
    # 同步 .env.example 中的新变量到 .env
    sync_env_with_example
    
    # 检查是否需要更新关键变量
    local current_host=$(grep "^EXTERNAL_HOST=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2)
    
    # 如果是自引用或空值，需要更新
    if [[ "$force" == "true" ]] || [[ "$current_host" =~ \$\{ ]] || [[ -z "$current_host" ]]; then
        log_info "Initializing environment variables..."
        log_info "  EXTERNAL_HOST=$detected_host"
        log_info "  EXTERNAL_PORT=$detected_port"
        log_info "  EXTERNAL_SCHEME=$detected_scheme"
        
        update_env_variable "EXTERNAL_HOST" "$detected_host"
        update_env_variable "DOMAIN" "$detected_host"
        update_env_variable "EXTERNAL_PORT" "$detected_port"
        update_env_variable "EXTERNAL_SCHEME" "$detected_scheme"
        
        log_info "✓ Environment variables initialized"
    fi
}

# 初始化环境
init_env_file

# Load .env variables
set -a
source "$ENV_FILE"
set +a

# Initialize COMMON_IMAGES array after loading .env
# This ensures version variables are available
COMMON_IMAGES=(
    "postgres:${POSTGRES_VERSION:-15-alpine}"
    "mysql:${MYSQL_VERSION:-8.0}"
    "redis:${REDIS_VERSION:-7-alpine}"
    "confluentinc/cp-kafka:${KAFKA_VERSION:-7.5.0}"
    "provectuslabs/kafka-ui:${KAFKA_UI_VERSION:-latest}"
    "osixia/openldap:${OPENLDAP_VERSION:-stable}"
    "osixia/phpldapadmin:${PHPLDAPADMIN_VERSION:-stable}"
    "redislabs/redisinsight:${REDISINSIGHT_VERSION:-latest}"
    "minio/minio:${MINIO_VERSION:-latest}"
    "oceanbase/oceanbase-ce:${OCEANBASE_VERSION:-4.3.5-lts}"
    "victoriametrics/victoria-metrics:${VICTORIAMETRICS_VERSION:-v1.115.0}"
)

# Ensure SSH Keys
SSH_KEY_DIR="$SCRIPT_DIR/ssh-key"
if [ ! -f "$SSH_KEY_DIR/id_rsa" ]; then
    log_info "Generating SSH keys..."
    mkdir -p "$SSH_KEY_DIR"
    ssh-keygen -t rsa -b 4096 -f "$SSH_KEY_DIR/id_rsa" -N "" -C "ai-infra-system@shared"
fi

# Ensure Third Party Directory
mkdir -p "$SCRIPT_DIR/third_party"

# ==============================================================================
# 2. Helper Functions
# ==============================================================================

detect_compose_command() {
    if command -v docker-compose >/dev/null 2>&1; then
        echo "docker-compose"
    elif docker compose version >/dev/null 2>&1; then
        echo "docker compose"
    else
        return 1
    fi
}

wait_for_apphub_ready() {
    local timeout="${1:-300}"
    local container_name="ai-infra-apphub"
    local check_interval=5
    local elapsed=0
    
    local apphub_port="${APPHUB_PORT:-28080}"
    local external_host="${EXTERNAL_HOST:-$(detect_external_host)}"
    local apphub_url="http://${external_host}:${apphub_port}"
    
    log_info "Waiting for AppHub at $apphub_url (Timeout: ${timeout}s)..."
    
    while [[ $elapsed -lt $timeout ]]; do
        # Check if container is running
        if ! docker ps --filter "name=$container_name" --filter "status=running" | grep -q "$container_name"; then
            log_warn "[${elapsed}s] Container not running..."
            sleep $check_interval
            elapsed=$((elapsed + check_interval))
            continue
        fi
        
        # Check if packages are accessible
        if curl -sf --connect-timeout 2 --max-time 5 "${apphub_url}/pkgs/slurm-deb/Packages" >/dev/null 2>&1; then
            log_info "✅ AppHub is ready!"
            return 0
        fi
        
        log_warn "[${elapsed}s] AppHub not ready yet..."
        sleep $check_interval
        elapsed=$((elapsed + check_interval))
    done
    
    log_error "❌ AppHub failed to become ready."
    return 1
}

# ==============================================================================
# Template Rendering Functions - 模板渲染功能
# ==============================================================================

# Define variables that need to be rendered in templates
# These are read from .env and used to replace {{VARIABLE}} placeholders
# 
# IMPORTANT: Variables are divided into two categories:
# 1. BUILD-TIME variables (Dockerfile.tpl) - Used during docker build
# 2. RUNTIME variables (docker-compose.yml.tpl, config templates) - Used at container startup
#
# Build-time variables are baked into the image and cannot be changed at runtime
# Runtime variables can be overridden via environment when starting containers
TEMPLATE_VARIABLES=(
    # ===========================================
    # Mirror configurations (Build-time)
    # Used in Dockerfile.tpl for package downloads during build
    # ===========================================
    "GITHUB_MIRROR"      # GitHub download accelerator (e.g., https://ghfast.top/)
    "APT_MIRROR"         # APT mirror for Debian/Ubuntu (e.g., mirrors.aliyun.com)
    "YUM_MIRROR"         # YUM mirror for Rocky/CentOS (e.g., mirrors.aliyun.com)
    "ALPINE_MIRROR"      # Alpine mirror (e.g., mirrors.aliyun.com)
    "GO_PROXY"           # Go module proxy (e.g., https://goproxy.cn,direct)
    "PYPI_INDEX_URL"     # PyPI mirror (e.g., https://mirrors.aliyun.com/pypi/simple/)
    "NPM_REGISTRY"       # npm registry mirror (e.g., https://registry.npmmirror.com)
    "INTERNAL_FILE_SERVER"  # Internal file server for intranet (e.g., http://192.168.1.100:8080/packages)
    
    # ===========================================
    # Base image versions (Build-time)
    # Used in Dockerfile.tpl FROM statements
    # ===========================================
    "UBUNTU_VERSION"              # Ubuntu base image (e.g., 22.04)
    "ROCKYLINUX_VERSION"          # Rocky Linux version (e.g., 9)
    "ALPINE_VERSION"              # Alpine version (e.g., 3.22)
    "NGINX_VERSION"               # Nginx version (e.g., stable-alpine-perl)
    "NGINX_ALPINE_VERSION"        # Nginx Alpine version (e.g., 1.27-alpine)
    "PYTHON_VERSION"              # Python version (e.g., 3.14)
    "PYTHON_ALPINE_VERSION"       # Python Alpine version (e.g., 3.14-alpine)
    "NODE_VERSION"                # Node.js major version (e.g., 22)
    "NODE_ALPINE_VERSION"         # Node.js Alpine version (e.g., 22-alpine)
    "NODE_BOOKWORM_VERSION"       # Node.js Bookworm version (e.g., 22-bookworm)
    "NODE_JS_VERSION"             # Node.js full version for prebuilt binaries (e.g., 22.11.0)
    "NODE_IMAGE_VERSION"          # Node.js image version for build (e.g., 22-bookworm)
    "GOLANG_VERSION"              # Go version (e.g., 1.25)
    "GOLANG_IMAGE_VERSION"        # Go image version (e.g., 1.25-bookworm)
    "JUPYTER_BASE_NOTEBOOK_VERSION"  # Jupyter base notebook version (e.g., latest)
    
    # ===========================================
    # Full base image names (for private registry support)
    # 完整基础镜像名称 (支持内网私有仓库)
    # Internet: golang:1.25-bookworm
    # Intranet: harbor.example.com/library/golang:1.25-bookworm
    # ===========================================
    "GOLANG_IMAGE"                # Full golang image name (e.g., golang:1.25-bookworm)
    "UBUNTU_IMAGE"                # Full ubuntu image name (e.g., ubuntu:22.04)
    "ROCKYLINUX_IMAGE"            # Full rockylinux image name (e.g., rockylinux:9)
    "NODE_IMAGE"                  # Full node image name for build (e.g., node:22-bookworm)
    "NODE_ALPINE_IMAGE"           # Full node alpine image name (e.g., node:22-alpine)
    "NODE_BOOKWORM_IMAGE"         # Full node bookworm image name (e.g., node:22-bookworm)
    "JUPYTER_BASE_IMAGE"          # Full jupyter base image name (e.g., jupyter/base-notebook:latest)
    "GITEA_IMAGE"                 # Full gitea image name (e.g., gitea/gitea:1.25.1)
    
    # ===========================================
    # Component/Application versions (Build-time)
    # Used in Dockerfile.tpl for building specific components
    # ===========================================
    "SLURM_VERSION"       # SLURM version (e.g., 24.11.5)
    "SALTSTACK_VERSION"   # SaltStack version (e.g., 3007.8)
    "CATEGRAF_VERSION"    # Categraf version (e.g., 0.4.6)
    "NODE_EXPORTER_VERSION" # Node Exporter version (e.g., v1.8.2)
    "SINGULARITY_VERSION" # Singularity version
    "GITEA_VERSION"       # Gitea version (e.g., 1.25.1)
    "JUPYTERHUB_VERSION"  # JupyterHub version (e.g., 5.3.*)
    "PIP_VERSION"         # pip version (e.g., 24.2)
    "N9E_FE_VERSION"      # Nightingale frontend version (e.g., v7.7.2, empty for auto-detect)
    
    # ===========================================
    # Project settings (Build-time & Runtime)
    # ===========================================
    "IMAGE_TAG"           # Docker image tag (e.g., v0.3.8)
    "TZ"                  # Timezone (e.g., Asia/Shanghai)
    
    # ===========================================
    # Nginx configuration variables (Runtime)
    # Used in src/nginx/templates/*.conf.tpl
    # ===========================================
    "EXTERNAL_HOST"       # External host IP/domain
    "EXTERNAL_SCHEME"     # http or https
    "FRONTEND_HOST"       # Frontend service host (default: frontend)
    "FRONTEND_PORT"       # Frontend service port (default: 3000)
    "BACKEND_HOST"        # Backend service host (default: backend)
    "BACKEND_PORT"        # Backend service port (default: 8082)
    "JUPYTERHUB_HOST"     # JupyterHub service host (default: jupyterhub)
    "JUPYTERHUB_PORT"     # JupyterHub service port (default: 8000)
    "NIGHTINGALE_HOST"    # Nightingale service host (default: nightingale)
    "NIGHTINGALE_PORT"    # Nightingale service port (default: 17000)
    
    # ===========================================
    # Third-party image versions (for docker-compose.yml.tpl)
    # ===========================================
    "POSTGRES_VERSION"    # PostgreSQL version (e.g., 15-alpine)
    "MYSQL_VERSION"       # MySQL version (e.g., 8.0)
    "REDIS_VERSION"       # Redis version (e.g., 7-alpine)
    "KAFKA_VERSION"       # Kafka version (e.g., 7.5.0)
    "KAFKA_UI_VERSION"    # Kafka UI version (e.g., latest)
    "OPENLDAP_VERSION"    # OpenLDAP version (e.g., stable)
    "PHPLDAPADMIN_VERSION" # phpLDAPadmin version (e.g., stable)
    "MINIO_VERSION"       # MinIO version (e.g., latest)
    "OCEANBASE_VERSION"   # OceanBase version (e.g., 4.3.5-lts)
    "PROMETHEUS_VERSION"  # Prometheus version (e.g., latest)
    "VICTORIAMETRICS_VERSION" # VictoriaMetrics version (e.g., v1.115.0)
    "GRAFANA_VERSION"     # Grafana version (e.g., latest)
    "ALERTMANAGER_VERSION" # AlertManager version (e.g., latest)
    "REDISINSIGHT_VERSION" # RedisInsight version (e.g., latest)
    
    # ===========================================
    # SaltStack configuration (Runtime)
    # Used for external node minion installation
    # ===========================================
    "SALT_MASTER_HOST"    # Salt Master host for container internal (e.g., saltstack)
    "SALT_MASTER_PORT"    # Salt Master publish port (e.g., 4505)
    "SALT_RETURN_PORT"    # Salt Master return port (e.g., 4506)
    "SALT_API_PORT"       # Salt API port (e.g., 8002)
    
    # ===========================================
    # AppHub configuration (Runtime)
    # ===========================================
    "APPHUB_PORT"         # AppHub port for package download (e.g., 28080)
)

# Render a single template file
# Args: $1 = template file (.tpl), $2 = output file (optional, defaults to removing .tpl extension)
render_template() {
    local template_file="$1"
    local output_file="${2:-${template_file%.tpl}}"
    
    if [[ ! -f "$template_file" ]]; then
        log_error "Template file not found: $template_file"
        return 1
    fi
    
    log_info "Rendering: $template_file -> $output_file"
    
    # Start with the template content
    local content
    content=$(cat "$template_file")
    
    # Replace each {{VARIABLE}} with its value from environment
    for var in "${TEMPLATE_VARIABLES[@]}"; do
        local value="${!var}"
        if [[ -n "$value" ]]; then
            # Escape special characters for sed
            local escaped_value
            escaped_value=$(printf '%s\n' "$value" | sed -e 's/[&/\]/\\&/g')
            content=$(echo "$content" | sed "s|{{${var}}}|${escaped_value}|g")
        else
            # If variable is empty, replace with empty string
            content=$(echo "$content" | sed "s|{{${var}}}||g")
        fi
    done
    
    # Write to output file
    echo "$content" > "$output_file"
    
    # Check for any remaining unreplaced placeholders
    local remaining
    remaining=$(grep -o '{{[A-Z_]*}}' "$output_file" 2>/dev/null | sort -u | head -5)
    if [[ -n "$remaining" ]]; then
        log_warn "  ⚠️  Unreplaced placeholders found: $remaining"
    fi
    
    log_info "  ✓ Rendered successfully"
    return 0
}

# Render all Dockerfile.tpl files in src/*/ and docker-compose.yml.tpl
render_all_templates() {
    local force="${1:-false}"
    
    log_info "=========================================="
    log_info "🔧 Rendering templates"
    log_info "=========================================="
    
    # Step 1: Sync .env with .env.example (add missing variables)
    log_info "Step 1: Syncing .env with .env.example..."
    sync_env_with_example
    
    # Reload .env after sync
    set -a
    source "$ENV_FILE"
    set +a
    
    log_info ""
    log_info "Step 2: Rendering template files..."
    log_info "Source: .env / .env.example"
    log_info "Pattern: src/*/Dockerfile.tpl, docker-compose.yml.tpl"
    echo
    
    # Show key variables being used
    log_info "Template variables:"
    log_info "  GITHUB_MIRROR=${GITHUB_MIRROR:-<empty>}"
    log_info "  APT_MIRROR=${APT_MIRROR:-<empty>}"
    log_info "  YUM_MIRROR=${YUM_MIRROR:-<empty>}"
    log_info "  ALPINE_MIRROR=${ALPINE_MIRROR:-<empty>}"
    log_info "  UBUNTU_VERSION=${UBUNTU_VERSION:-<empty>}"
    log_info "  SLURM_VERSION=${SLURM_VERSION:-<empty>}"
    log_info "  SALTSTACK_VERSION=${SALTSTACK_VERSION:-<empty>}"
    log_info "  CATEGRAF_VERSION=${CATEGRAF_VERSION:-<empty>}"
    log_info "  IMAGE_TAG=${IMAGE_TAG:-<empty>}"
    echo
    
    local success_count=0
    local fail_count=0
    local skip_count=0
    
    # Render docker-compose.yml.tpl if exists
    local compose_tpl="${SCRIPT_DIR}/docker-compose.yml.tpl"
    if [[ -f "$compose_tpl" ]]; then
        local compose_output="${SCRIPT_DIR}/docker-compose.yml"
        
        # Check if output file exists and is newer than template
        if [[ "$force" != "true" ]] && [[ -f "$compose_output" ]]; then
            if [[ "$compose_output" -nt "$compose_tpl" ]] && [[ "$compose_output" -nt "$ENV_FILE" ]]; then
                log_info "Skipping docker-compose.yml (up to date)"
                skip_count=$((skip_count + 1))
            else
                if render_template "$compose_tpl" "$compose_output"; then
                    success_count=$((success_count + 1))
                else
                    fail_count=$((fail_count + 1))
                fi
            fi
        else
            if render_template "$compose_tpl" "$compose_output"; then
                success_count=$((success_count + 1))
            else
                fail_count=$((fail_count + 1))
            fi
        fi
    fi
    
    # Find all Dockerfile.tpl files
    while IFS= read -r -d '' template_file; do
        local output_file="${template_file%.tpl}"
        local component_name=$(basename "$(dirname "$template_file")")
        
        # Check if output file exists and is newer than template
        if [[ "$force" != "true" ]] && [[ -f "$output_file" ]]; then
            if [[ "$output_file" -nt "$template_file" ]] && [[ "$output_file" -nt "$ENV_FILE" ]]; then
                log_info "Skipping $component_name (up to date)"
                skip_count=$((skip_count + 1))
                continue
            fi
        fi
        
        if render_template "$template_file" "$output_file"; then
            success_count=$((success_count + 1))
        else
            fail_count=$((fail_count + 1))
        fi
    done < <(find "$SRC_DIR" -name "Dockerfile.tpl" -print0 2>/dev/null)
    
    # ===========================================
    # Render Nginx configuration templates
    # ===========================================
    local nginx_template_dir="${SCRIPT_DIR}/src/nginx/templates"
    local nginx_output_dir="${SCRIPT_DIR}/src/nginx"
    
    if [[ -d "$nginx_template_dir" ]]; then
        log_info "Rendering Nginx configuration templates..."
        
        # Render main server config
        local main_conf_tpl="$nginx_template_dir/conf.d/server-main.conf.tpl"
        if [[ -f "$main_conf_tpl" ]]; then
            local main_conf_out="$nginx_output_dir/conf.d/server-main.conf"
            mkdir -p "$(dirname "$main_conf_out")"
            if render_template "$main_conf_tpl" "$main_conf_out"; then
                success_count=$((success_count + 1))
            else
                fail_count=$((fail_count + 1))
            fi
        fi
        
        # Render includes configs
        local includes_dir="$nginx_template_dir/conf.d/includes"
        if [[ -d "$includes_dir" ]]; then
            mkdir -p "$nginx_output_dir/conf.d/includes"
            while IFS= read -r -d '' tpl_file; do
                local out_file="$nginx_output_dir/conf.d/includes/$(basename "${tpl_file%.tpl}")"
                if render_template "$tpl_file" "$out_file"; then
                    success_count=$((success_count + 1))
                else
                    fail_count=$((fail_count + 1))
                fi
            done < <(find "$includes_dir" -name "*.tpl" -print0 2>/dev/null)
        fi
    fi
    
    # ===========================================
    # Render scripts/templates (e.g., install-salt-minion.sh.tpl)
    # These scripts contain EXTERNAL_HOST and other runtime variables
    # ===========================================
    local scripts_template_dir="${SCRIPT_DIR}/scripts/templates"
    if [[ -d "$scripts_template_dir" ]]; then
        log_info "Rendering script templates (scripts/templates/)..."
        
        while IFS= read -r -d '' tpl_file; do
            local tpl_basename=$(basename "$tpl_file")
            local out_file="${SCRIPT_DIR}/scripts/${tpl_basename%.tpl}"
            
            # Check if output file exists and is newer than template
            if [[ "$force" != "true" ]] && [[ -f "$out_file" ]]; then
                if [[ "$out_file" -nt "$tpl_file" ]] && [[ "$out_file" -nt "$ENV_FILE" ]]; then
                    log_info "Skipping $(basename "$out_file") (up to date)"
                    skip_count=$((skip_count + 1))
                    continue
                fi
            fi
            
            if render_template "$tpl_file" "$out_file"; then
                chmod +x "$out_file" 2>/dev/null || true
                success_count=$((success_count + 1))
            else
                fail_count=$((fail_count + 1))
            fi
        done < <(find "$scripts_template_dir" -name "*.tpl" -print0 2>/dev/null)
    fi
    
    echo
    log_info "=========================================="
    log_info "Template rendering complete:"
    log_info "  ✓ Success: $success_count"
    [[ $skip_count -gt 0 ]] && log_info "  ⏭️  Skipped: $skip_count"
    [[ $fail_count -gt 0 ]] && log_warn "  ✗ Failed: $fail_count"
    log_info "=========================================="
    
    if [[ $fail_count -gt 0 ]]; then
        return 1
    fi
    return 0
}

# Sync templates - alias for render_all_templates with force
sync_templates() {
    render_all_templates "true"
}

# ==============================================================================
# Pull Functions - 镜像拉取功能
# ==============================================================================

# Default retry settings
DEFAULT_MAX_RETRIES=3
DEFAULT_RETRY_DELAY=5

# Log file for tracking failures
FAILURE_LOG="${SCRIPT_DIR}/.build-failures.log"

# Log failure to file
log_failure() {
    local operation="$1"
    local target="$2"
    local error_msg="$3"
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    echo "[$timestamp] $operation FAILED: $target - $error_msg" >> "$FAILURE_LOG"
    log_error "[$timestamp] $operation FAILED: $target - $error_msg"
}

# Pull single image with retry mechanism
# Args: $1 = image name, $2 = max retries (default 3), $3 = retry delay (default 5)
pull_image_with_retry() {
    local image="$1"
    local max_retries="${2:-$DEFAULT_MAX_RETRIES}"
    local retry_delay="${3:-$DEFAULT_RETRY_DELAY}"
    local retry_count=0
    local last_error=""
    
    # Check if image already exists locally
    if docker image inspect "$image" >/dev/null 2>&1; then
        log_info "  ✓ Image exists: $image"
        return 0
    fi
    
    while [[ $retry_count -lt $max_retries ]]; do
        retry_count=$((retry_count + 1))
        
        if [[ $retry_count -gt 1 ]]; then
            log_warn "  🔄 Retry $retry_count/$max_retries: $image (waiting ${retry_delay}s...)"
            sleep $retry_delay
        else
            log_info "  ⬇ Pulling: $image"
        fi
        
        # Capture both stdout and stderr
        local output
        if output=$(docker pull "$image" 2>&1); then
            log_info "  ✓ Pulled: $image"
            return 0
        else
            last_error="$output"
            log_warn "  ⚠ Attempt $retry_count failed: $(echo "$last_error" | head -1)"
        fi
    done
    
    # All retries exhausted - log failure
    log_failure "PULL" "$image" "Failed after $max_retries attempts. Last error: $(echo "$last_error" | head -1)"
    return 1
}

# Extract base images from Dockerfile
# Args: $1 = Dockerfile path
extract_base_images() {
    local dockerfile="$1"
    
    if [[ ! -f "$dockerfile" ]]; then
        return 1
    fi
    
    # Extract FROM statements
    # Pattern: FROM image:tag [AS alias]
    # Skip: ARG variables (${...}), local build stages, empty images
    grep -E "^FROM\s+" "$dockerfile" 2>/dev/null | \
        awk '{
            img=$2
            # Skip ARG variables (contains ${...})
            if (img ~ /\$\{/) next
            # Skip platform flags
            if (img ~ /^--/) next
            # Skip if no colon and no slash (likely a build stage alias like "builder")
            if (img !~ /[:\/]/) next
            print img
        }' | \
        sort -u
}

# Prefetch base images from Dockerfiles
# Args: $1 = service name (optional, if empty prefetch all)
prefetch_base_images() {
    local service_name="$1"
    local max_retries="${2:-3}"
    
    log_info "📦 Prefetching base images..."
    
    local dockerfiles=()
    
    if [[ -n "$service_name" ]]; then
        local dockerfile="$SRC_DIR/$service_name/Dockerfile"
        if [[ -f "$dockerfile" ]]; then
            dockerfiles+=("$dockerfile")
        fi
    else
        # Find all Dockerfiles
        while IFS= read -r df; do
            dockerfiles+=("$df")
        done < <(find "$SRC_DIR" -name "Dockerfile" -type f 2>/dev/null)
    fi
    
    local all_images=()
    local pull_count=0
    local skip_count=0
    local fail_count=0
    
    # Extract all base images
    for dockerfile in "${dockerfiles[@]}"; do
        local images
        images=$(extract_base_images "$dockerfile")
        while IFS= read -r img; do
            [[ -z "$img" ]] && continue
            [[ "$img" =~ ^[a-z_-]+$ ]] && continue  # Skip internal build stages
            all_images+=("$img")
        done <<< "$images"
    done
    
    # Remove duplicates
    local unique_images=($(printf '%s\n' "${all_images[@]}" | sort -u))
    
    log_info "Found ${#unique_images[@]} unique base images to check"
    
    for image in "${unique_images[@]}"; do
        if docker image inspect "$image" >/dev/null 2>&1; then
            log_info "  ✓ Exists: $image"
            skip_count=$((skip_count + 1))
        else
            log_info "  ⬇ Pulling: $image"
            if pull_image_with_retry "$image" "$max_retries"; then
                pull_count=$((pull_count + 1))
            else
                fail_count=$((fail_count + 1))
            fi
        fi
    done
    
    log_info "📊 Prefetch summary: pulled=$pull_count, skipped=$skip_count, failed=$fail_count"
    return 0
}

# Pull all project images from registry
# Smart mode:
#   - No registry: Pull only common images from Docker Hub (internet mode)
#   - With registry: Pull all images from private registry (intranet mode)
# Args: $1 = registry (optional), $2 = tag
# 
# For Harbor/private registries, registry path should include project name:
#   ✓ harbor.example.com/ai-infra    (correct - includes project)
#   ✗ harbor.example.com             (wrong - missing project)
pull_all_services() {
    local registry="$1"
    local tag="${2:-${IMAGE_TAG:-latest}}"
    local max_retries="${3:-$DEFAULT_MAX_RETRIES}"
    
    discover_services
    
    local success_count=0
    local total_count=0
    local failed_services=()
    
    # Validate registry path for private registries (Harbor requires project in path)
    if [[ -n "$registry" ]]; then
        # Check if registry contains project path (should have at least one /)
        if [[ ! "$registry" =~ / ]]; then
            log_warn "=========================================="
            log_warn "⚠️  Registry path may be incomplete!"
            log_warn "=========================================="
            log_warn "Provided: $registry"
            log_warn ""
            log_warn "Harbor registries require a project name in the path:"
            log_warn "  ✓ $registry/ai-infra    (recommended)"
            log_warn "  ✓ $registry/<project>   (your project name)"
            log_warn ""
            log_warn "Example usage:"
            log_warn "  $0 pull-all $registry/ai-infra $tag"
            log_warn ""
            read -p "Continue anyway? [y/N] " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                log_info "Cancelled. Please use correct registry path."
                return 1
            fi
            log_warn "Continuing with incomplete registry path..."
        fi
    fi
    
    if [[ -z "$registry" ]]; then
        # ==========================================
        # Mode 1: Pull from Docker Hub (internet mode)
        # Only pulls public common/third-party images
        # ==========================================
        log_info "=========================================="
        log_info "Pulling images from Docker Hub (Internet Mode)"
        log_info "=========================================="
        log_info "Mode: Public (Docker Hub)"
        log_info "Max retries: $max_retries"
        echo
        
        # Phase 1: Pull common/third-party images from Docker Hub
        log_info "=== Phase 1: Common/third-party images ==="
        for image in "${COMMON_IMAGES[@]}"; do
            total_count=$((total_count + 1))
            log_info "[$total_count/${#COMMON_IMAGES[@]}] $image"
            
            if docker image inspect "$image" &>/dev/null; then
                log_info "  ✓ Already exists"
                success_count=$((success_count + 1))
                continue
            fi
            
            if pull_image_with_retry "$image" "$max_retries"; then
                log_info "  ✓ Pulled"
                success_count=$((success_count + 1))
            else
                log_warn "  ✗ Failed"
                failed_services+=("common:$image")
            fi
        done
        echo
        
        log_info "=== Phase 2: Project services (skipped - need registry) ==="
        log_info "ℹ️  Project images require registry to pull"
        log_info "💡 Usage: $0 pull-all <registry> [tag]"
        echo
        
    else
        # ==========================================
        # Mode 2: Pull from private registry (intranet mode)
        # Pulls all images: common + dependency + project
        # ==========================================
        registry="${registry%/}"  # Remove trailing slash
        
        log_info "=========================================="
        log_info "Pulling images from Private Registry (Intranet Mode)"
        log_info "=========================================="
        log_info "Mode: Private Registry"
        log_info "Registry: $registry"
        log_info "Tag: $tag"
        log_info "Max retries: $max_retries"
        echo
        
        # Phase 1: Pull common images from private registry
        log_info "=== Phase 1: Common/third-party images ==="
        for image in "${COMMON_IMAGES[@]}"; do
            total_count=$((total_count + 1))
            
            # Extract short name (e.g., confluentinc/cp-kafka:7.5.0 -> cp-kafka)
            local image_name="${image%%:*}"
            local image_tag="${image##*:}"
            local short_name="${image_name##*/}"
            local remote_image="${registry}/${short_name}:${image_tag}"
            
            log_info "[$total_count] $remote_image"
            
            # Check if original image already exists
            if docker image inspect "$image" &>/dev/null; then
                log_info "  ✓ Already exists locally"
                success_count=$((success_count + 1))
                continue
            fi
            
            # Try to pull from private registry
            if pull_image_with_retry "$remote_image" "$max_retries"; then
                # Tag as original image name for docker-compose compatibility
                if docker tag "$remote_image" "$image"; then
                    log_info "  ✓ Pulled and tagged as $image"
                    success_count=$((success_count + 1))
                else
                    log_warn "  ⚠ Pulled but failed to tag"
                    success_count=$((success_count + 1))
                fi
            else
                log_warn "  ✗ Failed to pull from registry"
                failed_services+=("common:$short_name")
            fi
        done
        echo
        
        # Phase 2: Pull dependency images with project tag
        log_info "=== Phase 2: Dependency images (tag: $tag) ==="
        local dependencies=($(get_dependency_mappings))
        for mapping in "${dependencies[@]}"; do
            total_count=$((total_count + 1))
            
            local source_image="${mapping%%|*}"
            local short_name="${mapping##*|}"
            local remote_image="${registry}/${short_name}:${tag}"
            
            log_info "[$total_count] $remote_image -> $source_image"
            
            # Check if source image already exists
            if docker image inspect "$source_image" &>/dev/null; then
                log_info "  ✓ Already exists locally"
                success_count=$((success_count + 1))
                continue
            fi
            
            if pull_image_with_retry "$remote_image" "$max_retries"; then
                # Tag as original image name for docker-compose compatibility
                if docker tag "$remote_image" "$source_image"; then
                    log_info "  ✓ Pulled and tagged"
                    success_count=$((success_count + 1))
                else
                    log_warn "  ⚠ Pulled but failed to tag"
                    success_count=$((success_count + 1))
                fi
            else
                log_warn "  ✗ Failed"
                failed_services+=("dep:$short_name")
            fi
        done
        echo
        
        # Phase 3: Pull project services
        log_info "=== Phase 3: Project services (tag: $tag) ==="
        for service in "${FOUNDATION_SERVICES[@]}" "${DEPENDENT_SERVICES[@]}"; do
            total_count=$((total_count + 1))
            local image_name="ai-infra-${service}:${tag}"
            local remote_image="${registry}/${image_name}"
            
            log_info "[$total_count] $remote_image"
            
            if pull_image_with_retry "$remote_image" "$max_retries"; then
                if docker tag "$remote_image" "$image_name"; then
                    log_info "  ✓ Pulled and tagged as $image_name"
                    success_count=$((success_count + 1))
                else
                    log_warn "  ⚠ Pulled but failed to tag"
                    success_count=$((success_count + 1))
                fi
            else
                log_warn "  ✗ Failed"
                failed_services+=("$service")
            fi
        done
        echo
        
        # Phase 4: Pull special images (multi-stage build targets, etc.)
        # These are images that don't have their own src/ directory
        log_info "=== Phase 4: Special images (tag: $tag) ==="
        local special_images=(
            "backend-init"    # Multi-stage build target from backend
        )
        for special in "${special_images[@]}"; do
            total_count=$((total_count + 1))
            local image_name="ai-infra-${special}:${tag}"
            local remote_image="${registry}/${image_name}"
            
            log_info "[$total_count] $remote_image"
            
            if pull_image_with_retry "$remote_image" "$max_retries"; then
                if docker tag "$remote_image" "$image_name"; then
                    log_info "  ✓ Pulled and tagged as $image_name"
                    success_count=$((success_count + 1))
                else
                    log_warn "  ⚠ Pulled but failed to tag"
                    success_count=$((success_count + 1))
                fi
            else
                log_warn "  ✗ Failed"
                failed_services+=("special:$special")
            fi
        done
        echo
    fi
    
    log_info "=========================================="
    log_info "Pull completed: $success_count/$total_count successful"
    
    if [[ ${#failed_services[@]} -gt 0 ]]; then
        log_warn "Failed: ${failed_services[*]}"
        log_info "Check failure log: $FAILURE_LOG"
        return 1
    fi
    
    log_info "🎉 All images pulled successfully!"
    return 0
}

# Pull only common/third-party images (no registry required)
# Useful for preparing environment before starting services
pull_common_images() {
    local max_retries="${1:-$DEFAULT_MAX_RETRIES}"
    
    log_info "=========================================="
    log_info "Pulling common/third-party images"
    log_info "=========================================="
    log_info "Images to pull: ${#COMMON_IMAGES[@]}"
    log_info "Max retries: $max_retries"
    echo
    
    local success_count=0
    local total_count=0
    local failed_images=()
    
    for image in "${COMMON_IMAGES[@]}"; do
        total_count=$((total_count + 1))
        log_info "[$total_count/${#COMMON_IMAGES[@]}] Pulling: $image"
        
        # Check if image already exists locally
        if docker image inspect "$image" &>/dev/null; then
            log_info "  ✓ Already exists: $image"
            success_count=$((success_count + 1))
            continue
        fi
        
        if pull_image_with_retry "$image" "$max_retries"; then
            log_info "  ✓ Pulled: $image"
            success_count=$((success_count + 1))
        else
            log_warn "  ✗ Failed: $image"
            failed_images+=("$image")
        fi
    done
    
    echo
    log_info "=========================================="
    log_info "Pull completed: $success_count/$total_count successful"
    
    if [[ ${#failed_images[@]} -gt 0 ]]; then
        log_warn "Failed images: ${failed_images[*]}"
        return 1
    fi
    
    log_info "🎉 All common images pulled successfully!"
    return 0
}

# ==============================================================================
# Push Functions - 镜像推送功能
# ==============================================================================

# Push single image with retry mechanism
# Args: $1 = image, $2 = max retries (default 3), $3 = retry delay (default 5)
push_image_with_retry() {
    local image="$1"
    local max_retries="${2:-$DEFAULT_MAX_RETRIES}"
    local retry_delay="${3:-$DEFAULT_RETRY_DELAY}"
    local retry_count=0
    local last_error=""
    
    while [[ $retry_count -lt $max_retries ]]; do
        retry_count=$((retry_count + 1))
        
        if [[ $retry_count -gt 1 ]]; then
            log_warn "  🔄 Retry $retry_count/$max_retries: $image (waiting ${retry_delay}s...)"
            sleep $retry_delay
        fi
        
        # Capture both stdout and stderr
        local output
        if output=$(docker push "$image" 2>&1); then
            log_info "  ✓ Pushed: $image"
            return 0
        else
            last_error="$output"
            log_warn "  ⚠ Attempt $retry_count failed: $(echo "$last_error" | head -1)"
        fi
    done
    
    # All retries exhausted - log failure
    log_failure "PUSH" "$image" "Failed after $max_retries attempts. Last error: $(echo "$last_error" | head -1)"
    return 1
}

# Push single service image
# Args: $1 = service, $2 = tag, $3 = registry
push_service() {
    local service="$1"
    local tag="${2:-${IMAGE_TAG:-latest}}"
    local registry="$3"
    local max_retries="${4:-$DEFAULT_MAX_RETRIES}"
    
    if [[ -z "$registry" ]]; then
        log_error "Registry is required for push"
        return 1
    fi
    
    local base_image="ai-infra-${service}:${tag}"
    local target_image="$registry/ai-infra-${service}:${tag}"
    
    log_info "Pushing service: $service"
    log_info "  Source: $base_image"
    log_info "  Target: $target_image"
    
    # Check if source image exists
    if ! docker image inspect "$base_image" >/dev/null 2>&1; then
        log_warn "Local image not found: $base_image"
        log_info "Building image first..."
        if ! build_component "$service"; then
            log_failure "BUILD" "$base_image" "Build failed before push"
            return 1
        fi
    fi
    
    # Tag for registry with retry
    if [[ "$base_image" != "$target_image" ]]; then
        log_info "  Tagging: $base_image -> $target_image"
        if ! docker tag "$base_image" "$target_image"; then
            log_failure "TAG" "$target_image" "Failed to tag image"
            return 1
        fi
    fi
    
    # Push to registry with retry
    log_info "  Pushing: $target_image"
    if push_image_with_retry "$target_image" "$max_retries"; then
        return 0
    else
        return 1
    fi
}

# Push all service images (including common/dependency images)
# This function pushes images in 3 phases for complete offline deployment:
#   Phase 1: Common images (original tags) - for general use
#   Phase 2: Dependency images (project tag) - for version-controlled deployment
#   Phase 3: Project services (project tag) - the main application images
# Args: $1 = registry, $2 = tag
#
# For Harbor/private registries, registry path should include project name:
#   ✓ harbor.example.com/ai-infra    (correct - includes project)
#   ✗ harbor.example.com             (wrong - missing project)
push_all_services() {
    local registry="$1"
    local tag="${2:-${IMAGE_TAG:-latest}}"
    local max_retries="${3:-$DEFAULT_MAX_RETRIES}"
    
    if [[ -z "$registry" ]]; then
        log_error "Registry is required for push-all"
        log_info "Usage: $0 push-all <registry/project> [tag]"
        log_info "Example: $0 push-all harbor.example.com/ai-infra v0.3.8"
        return 1
    fi
    
    # Validate registry path (Harbor requires project in path)
    if [[ ! "$registry" =~ / ]]; then
        log_warn "=========================================="
        log_warn "⚠️  Registry path may be incomplete!"
        log_warn "=========================================="
        log_warn "Provided: $registry"
        log_warn ""
        log_warn "Harbor registries require a project name in the path:"
        log_warn "  ✓ $registry/ai-infra    (recommended)"
        log_warn "  ✓ $registry/<project>   (your project name)"
        log_warn ""
        log_warn "Example usage:"
        log_warn "  $0 push-all $registry/ai-infra $tag"
        log_warn ""
        read -p "Continue anyway? [y/N] " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Cancelled. Please use correct registry path."
            return 1
        fi
        log_warn "Continuing with incomplete registry path..."
    fi
    
    # Ensure registry ends without trailing slash
    registry="${registry%/}"
    
    log_info "=========================================="
    log_info "Pushing ALL images to registry"
    log_info "=========================================="
    log_info "Registry: $registry"
    log_info "Tag: $tag"
    log_info "Max retries: $max_retries"
    echo
    
    discover_services
    
    local success_count=0
    local total_count=0
    local failed_services=()
    
    # Phase 1: Push common/third-party images with original tags
    log_info "=== Phase 1: Common/third-party images (original tags) ==="
    log_info "These images keep their original tags for general compatibility"
    echo
    for image in "${COMMON_IMAGES[@]}"; do
        total_count=$((total_count + 1))
        
        # Extract image name without tag for target naming
        local image_name="${image%%:*}"
        local image_tag="${image##*:}"
        # Remove registry prefix if any (e.g., confluentinc/cp-kafka -> cp-kafka)
        local short_name="${image_name##*/}"
        local target_image="${registry}/${short_name}:${image_tag}"
        
        log_info "[$total_count] $image -> $target_image"
        
        # Check if source image exists locally
        if ! docker image inspect "$image" >/dev/null 2>&1; then
            log_info "  Pulling source image..."
            if ! pull_image_with_retry "$image" "$max_retries"; then
                log_warn "  ✗ Failed to pull: $image"
                failed_services+=("common:$image")
                continue
            fi
        fi
        
        # Tag for registry
        if ! docker tag "$image" "$target_image"; then
            log_warn "  ✗ Failed to tag: $target_image"
            failed_services+=("common:$image")
            continue
        fi
        
        # Push to registry
        if push_image_with_retry "$target_image" "$max_retries"; then
            log_info "  ✓ Pushed"
            success_count=$((success_count + 1))
        else
            failed_services+=("common:$image")
        fi
    done
    echo
    
    # Phase 2: Push dependency images with project tag
    log_info "=== Phase 2: Dependency images (tag: $tag) ==="
    log_info "These images are tagged with project version for version-controlled deployment"
    echo
    local dependencies=($(get_dependency_mappings))
    for mapping in "${dependencies[@]}"; do
        total_count=$((total_count + 1))
        
        local source_image="${mapping%%|*}"
        local short_name="${mapping##*|}"
        local target_image="${registry}/${short_name}:${tag}"
        
        log_info "[$total_count] $source_image -> $target_image"
        
        # Check if source image exists locally
        if ! docker image inspect "$source_image" >/dev/null 2>&1; then
            log_info "  Pulling source image..."
            if ! pull_image_with_retry "$source_image" "$max_retries"; then
                log_warn "  ✗ Failed to pull: $source_image"
                failed_services+=("dep:$short_name")
                continue
            fi
        fi
        
        # Tag for registry with project tag
        if ! docker tag "$source_image" "$target_image"; then
            log_warn "  ✗ Failed to tag: $target_image"
            failed_services+=("dep:$short_name")
            continue
        fi
        
        # Push to registry
        if push_image_with_retry "$target_image" "$max_retries"; then
            log_info "  ✓ Pushed"
            success_count=$((success_count + 1))
        else
            failed_services+=("dep:$short_name")
        fi
    done
    echo
    
    # Phase 3: Push project services
    log_info "=== Phase 3: Project services (tag: $tag) ==="
    log_info "Main application images built from src/*"
    echo
    for service in "${FOUNDATION_SERVICES[@]}" "${DEPENDENT_SERVICES[@]}"; do
        total_count=$((total_count + 1))
        
        if push_service "$service" "$tag" "$registry" "$max_retries"; then
            log_info "  ✓ $service pushed"
            success_count=$((success_count + 1))
        else
            failed_services+=("$service")
        fi
    done
    echo
    
    # Phase 4: Push special images (multi-stage build targets, etc.)
    log_info "=== Phase 4: Special images (tag: $tag) ==="
    log_info "Images from multi-stage builds that don't have their own src/ directory"
    echo
    local special_images=(
        "backend-init"    # Multi-stage build target from backend
    )
    for special in "${special_images[@]}"; do
        total_count=$((total_count + 1))
        local image_name="ai-infra-${special}:${tag}"
        local target_image="${registry}/${image_name}"
        
        log_info "[$total_count] $image_name -> $target_image"
        
        # Check if source image exists locally
        if ! docker image inspect "$image_name" >/dev/null 2>&1; then
            log_warn "  ✗ Source image not found: $image_name"
            log_info "    Hint: Build with 'docker compose build backend-init'"
            failed_services+=("special:$special")
            continue
        fi
        
        # Tag for registry
        if ! docker tag "$image_name" "$target_image"; then
            log_warn "  ✗ Failed to tag: $target_image"
            failed_services+=("special:$special")
            continue
        fi
        
        # Push to registry
        if push_image_with_retry "$target_image" "$max_retries"; then
            log_info "  ✓ Pushed"
            success_count=$((success_count + 1))
        else
            failed_services+=("special:$special")
        fi
    done
    echo
    
    log_info "=========================================="
    log_info "Push completed: $success_count/$total_count successful"
    
    if [[ ${#failed_services[@]} -gt 0 ]]; then
        log_warn "Failed: ${failed_services[*]}"
        log_info "Check failure log: $FAILURE_LOG"
        return 1
    fi
    
    log_info "🚀 All images pushed successfully!"
    return 0
}

# Get dependency image mappings
get_dependency_mappings() {
    local mappings=(
        "confluentinc/cp-kafka:${KAFKA_VERSION:-7.5.0}|cp-kafka"
        "provectuslabs/kafka-ui:${KAFKAUI_VERSION:-latest}|kafka-ui"
        "postgres:${POSTGRES_VERSION:-15-alpine}|postgres"
        "redis:${REDIS_VERSION:-7-alpine}|redis"
        "minio/minio:${MINIO_VERSION:-latest}|minio"
        "osixia/openldap:${OPENLDAP_VERSION:-stable}|openldap"
        "osixia/phpldapadmin:${PHPLDAPADMIN_VERSION:-stable}|phpldapadmin"
        "mysql:${MYSQL_VERSION:-8.0}|mysql"
        "victoriametrics/victoria-metrics:${VICTORIAMETRICS_VERSION:-v1.115.0}|victoria-metrics"
    )
    echo "${mappings[@]}"
}

# Push all dependency images
# Args: $1 = registry, $2 = tag
push_all_dependencies() {
    local registry="$1"
    local tag="${2:-${IMAGE_TAG:-latest}}"
    local max_retries="${3:-$DEFAULT_MAX_RETRIES}"
    
    if [[ -z "$registry" ]]; then
        log_error "Registry is required for push-dep"
        log_info "Usage: $0 push-dep <registry> [tag]"
        return 1
    fi
    
    # Ensure registry ends without trailing slash for consistent handling
    registry="${registry%/}"
    
    log_info "=========================================="
    log_info "Pushing all dependency images"
    log_info "=========================================="
    log_info "Registry: $registry"
    log_info "Tag: $tag"
    log_info "Max retries: $max_retries"
    echo
    
    local dependencies=($(get_dependency_mappings))
    local success_count=0
    local total_count=${#dependencies[@]}
    local failed_images=()
    
    for mapping in "${dependencies[@]}"; do
        local source_image="${mapping%%|*}"
        local short_name="${mapping##*|}"
        local target_image="${registry}/${short_name}:${tag}"
        
        log_info "Processing: $source_image"
        log_info "  → Target: $target_image"
        
        # 1. Pull or check source image (with retry)
        log_info "  [1/3] Checking source image..."
        if docker image inspect "$source_image" >/dev/null 2>&1; then
            log_info "  ✓ Image exists locally"
        else
            if ! pull_image_with_retry "$source_image" "$max_retries"; then
                failed_images+=("$source_image")
                echo
                continue
            fi
        fi
        
        # 2. Tag for registry
        log_info "  [2/3] Tagging image..."
        if ! docker tag "$source_image" "$target_image"; then
            log_failure "TAG" "$target_image" "Failed to tag from $source_image"
            failed_images+=("$source_image")
            echo
            continue
        fi
        log_info "  ✓ Tagged"
        
        # 3. Push to registry (with retry)
        log_info "  [3/3] Pushing image..."
        if push_image_with_retry "$target_image" "$max_retries"; then
            success_count=$((success_count + 1))
        else
            failed_images+=("$source_image")
        fi
        echo
    done
    
    log_info "=========================================="
    log_info "Dependency push completed: $success_count/$total_count successful"
    
    if [[ ${#failed_images[@]} -gt 0 ]]; then
        log_warn "Failed images: ${failed_images[*]}"
        log_info "Check failure log: $FAILURE_LOG"
        return 1
    fi
    
    log_info "🚀 All dependency images pushed successfully!"
    return 0
}

# Pull and tag dependencies from registry
# Args: $1 = registry, $2 = tag
pull_and_tag_dependencies() {
    local registry="$1"
    local tag="${2:-${IMAGE_TAG:-latest}}"
    local max_retries="${3:-$DEFAULT_MAX_RETRIES}"
    
    if [[ -z "$registry" ]]; then
        log_error "Registry is required"
        log_info "Usage: $0 deps-pull <registry> [tag]"
        return 1
    fi
    
    registry="${registry%/}"
    
    log_info "=========================================="
    log_info "Pulling dependencies from: $registry"
    log_info "=========================================="
    log_info "Tag: $tag"
    log_info "Max retries: $max_retries"
    echo
    
    local dependencies=($(get_dependency_mappings))
    local success_count=0
    local total_count=${#dependencies[@]}
    local failed_deps=()
    
    for mapping in "${dependencies[@]}"; do
        local source_image="${mapping%%|*}"
        local short_name="${mapping##*|}"
        local remote_image="${registry}/${short_name}:${tag}"
        
        log_info "Pulling: $remote_image"
        
        if pull_image_with_retry "$remote_image" "$max_retries"; then
            # Tag as original image name
            if docker tag "$remote_image" "$source_image"; then
                log_info "  ✓ Tagged: $source_image"
                success_count=$((success_count + 1))
            else
                log_failure "TAG" "$source_image" "Failed to tag from $remote_image"
                failed_deps+=("$short_name")
            fi
        else
            failed_deps+=("$short_name")
        fi
    done
    
    echo
    log_info "=========================================="
    log_info "Dependencies pull completed: $success_count/$total_count"
    
    if [[ ${#failed_deps[@]} -gt 0 ]]; then
        log_warn "Failed: ${failed_deps[*]}"
        log_info "Check failure log: $FAILURE_LOG"
        return 1
    fi
    
    log_info "🎉 All dependencies pulled successfully!"
    return 0
}

# ==============================================================================
# 3. Build Logic
# ==============================================================================

# Global flag for force rebuild (--no-cache)
FORCE_BUILD=false

# Prepare base build args
BASE_BUILD_ARGS=()
if [ -f "$ENV_EXAMPLE" ]; then
    while IFS='=' read -r key value; do
        [[ "$key" =~ ^#.*$ ]] && continue
        [[ -z "$key" ]] && continue
        curr_val="${!key}"
        if [ -n "$curr_val" ]; then
            BASE_BUILD_ARGS+=("--build-arg" "$key=$curr_val")
        fi
    done < <(grep -v '^#' "$ENV_EXAMPLE")
fi
BASE_BUILD_ARGS+=("--build-arg" "BUILD_ENV=${BUILD_ENV:-production}")

build_component() {
    local component="$1"
    local extra_args=("${@:2}") # Capture all remaining arguments
    local component_dir="$SRC_DIR/$component"
    
    if [ ! -d "$component_dir" ]; then
        log_error "Component directory not found: $component_dir"
        return 1
    fi

    # Check if template exists and render it
    local template_file="$component_dir/Dockerfile.tpl"
    if [ -f "$template_file" ]; then
        log_info "Rendering template for $component..."
        if ! render_template "$template_file"; then
            log_error "Failed to render template for $component"
            return 1
        fi
    fi

    # Check for dependency configuration (External Image)
    local dep_conf="$component_dir/dependency.conf"
    if [ -f "$dep_conf" ]; then
        local upstream_image=$(grep -v '^#' "$dep_conf" | head -n 1 | tr -d '[:space:]')
        if [ -z "$upstream_image" ]; then
            log_error "Empty dependency config for $component"
            return 1
        fi
        
        local target_image="ai-infra-$component:${IMAGE_TAG:-latest}"
        if [ -n "$PRIVATE_REGISTRY" ]; then
            target_image="$PRIVATE_REGISTRY/$target_image"
        fi
        
        log_info "Processing dependency $component: $upstream_image -> $target_image"
        
        if pull_image_with_retry "$upstream_image" 3 5; then
            if docker tag "$upstream_image" "$target_image"; then
                log_info "✓ Dependency ready: $target_image"
                return 0
            else
                log_error "✗ Failed to tag $upstream_image"
                return 1
            fi
        else
            log_error "✗ Failed to pull $upstream_image after retries"
            return 1
        fi
    fi
    
    if [ ! -f "$component_dir/Dockerfile" ]; then
        log_warn "No Dockerfile or dependency.conf in $component, skipping..."
        return 0
    fi

    # Check for build-targets.conf
    local targets_file="$component_dir/build-targets.conf"
    local targets=()
    local images=()
    
    if [ -f "$targets_file" ]; then
        while read -r target image_suffix || [ -n "$target" ]; do
            [[ "$target" =~ ^#.*$ ]] && continue
            [[ -z "$target" ]] && continue
            targets+=("$target")
            images+=("$image_suffix")
        done < "$targets_file"
    else
        targets+=("default")
        images+=("ai-infra-$component")
    fi

    for i in "${!targets[@]}"; do
        local target="${targets[$i]}"
        local image_name="${images[$i]}"
        local full_image_name="${image_name}:${IMAGE_TAG:-latest}"
        
        if [ -n "$PRIVATE_REGISTRY" ]; then
            full_image_name="$PRIVATE_REGISTRY/$full_image_name"
        fi
        
        log_info "Building $component [$target] -> $full_image_name"
        
        local cmd=("docker" "build")
        
        # Add --no-cache if force build is enabled
        if [[ "$FORCE_BUILD" == "true" ]]; then
            cmd+=("--no-cache")
        fi
        
        cmd+=("${BASE_BUILD_ARGS[@]}" "${extra_args[@]}" "-t" "$full_image_name" "-f" "$component_dir/Dockerfile")
        
        if [ "$target" != "default" ]; then
            cmd+=("--target" "$target")
        fi
        
        # Add build context (project root)
        cmd+=("$SCRIPT_DIR")
        
        if "${cmd[@]}"; then
            log_info "✓ Build success: $full_image_name"
        else
            log_error "✗ Build failed: $full_image_name"
            return 1
        fi
    done
}

discover_services() {
    log_info "Discovering components in $SRC_DIR..."
    DEPENDENCY_SERVICES=()
    FOUNDATION_SERVICES=()
    DEPENDENT_SERVICES=()

    # Use find to avoid issues if directory is empty and sort for deterministic order
    while IFS= read -r dir; do
        local component=$(basename "$dir")
        
        # 1. Check for dependency.conf (External Image)
        if [ -f "$dir/dependency.conf" ]; then
            DEPENDENCY_SERVICES+=("$component")
            continue
        fi
        
        # 2. Check for Dockerfile (Buildable Component)
        if [ -f "$dir/Dockerfile" ]; then
            local phase="dependent" # Default phase
            
            # Check for build.conf override
            if [ -f "$dir/build.conf" ]; then
                local conf_phase=$(grep "^BUILD_PHASE=" "$dir/build.conf" | cut -d= -f2 | tr -d '[:space:]')
                if [ -n "$conf_phase" ]; then
                    phase="$conf_phase"
                fi
            fi
            
            if [ "$phase" == "foundation" ]; then
                FOUNDATION_SERVICES+=("$component")
            else
                DEPENDENT_SERVICES+=("$component")
            fi
        fi
    done < <(find "$SRC_DIR" -mindepth 1 -maxdepth 1 -type d | sort)
    
    log_info "Found ${#DEPENDENCY_SERVICES[@]} dependency services: ${DEPENDENCY_SERVICES[*]}"
    log_info "Found ${#FOUNDATION_SERVICES[@]} foundation services: ${FOUNDATION_SERVICES[*]}"
    log_info "Found ${#DEPENDENT_SERVICES[@]} dependent services: ${DEPENDENT_SERVICES[*]}"
}

build_all() {
    local force="${1:-false}"
    
    if [[ "$force" == "true" ]]; then
        log_info "Starting coordinated build process (FORCE MODE - no cache)..."
        FORCE_BUILD=true
        
        # In force mode, auto-detect and update EXTERNAL_HOST if needed
        log_info "=== Phase -1: Verifying Network Configuration ==="
        local current_host=$(grep "^EXTERNAL_HOST=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2)
        local detected_host=$(detect_external_host)
        
        if [[ "$current_host" != "$detected_host" ]]; then
            log_warn "EXTERNAL_HOST changed: $current_host -> $detected_host"
            log_info "Updating .env with new IP address..."
            update_env_variable "EXTERNAL_HOST" "$detected_host"
            update_env_variable "DOMAIN" "$detected_host"
            log_info "✓ EXTERNAL_HOST updated to $detected_host"
        else
            log_info "✓ EXTERNAL_HOST is correct: $current_host"
        fi
        echo
    else
        log_info "Starting coordinated build process..."
    fi
    
    # 0. Render all templates first
    log_info "=== Phase 0: Rendering Dockerfile Templates ==="
    if ! render_all_templates "$force"; then
        log_error "Template rendering failed. Aborting build."
        exit 1
    fi
    echo
    
    # 0.5. Prefetch base images with retry
    log_info "=== Phase 0.5: Prefetching Base Images (with retry) ==="
    prefetch_base_images "" 3  # 3 retries
    echo
    
    # Discover services dynamically
    discover_services
    
    # 1. Pull & Tag Dependency Services
    log_info "=== Phase 1: Processing Dependency Services ==="
    for service in "${DEPENDENCY_SERVICES[@]}"; do
        build_component "$service"
    done
    
    # 2. Build Foundation Services
    log_info "=== Phase 2: Building Foundation Services ==="
    for service in "${FOUNDATION_SERVICES[@]}"; do
        build_component "$service"
    done
    
    # 3. Start AppHub Service
    log_info "=== Phase 3: Starting AppHub Service ==="
    local compose_cmd=$(detect_compose_command)
    if [ -z "$compose_cmd" ]; then
        log_error "docker-compose not found! Cannot start AppHub."
        exit 1
    fi
    
    log_info "Starting AppHub container..."
    $compose_cmd up -d apphub
    
    if ! wait_for_apphub_ready 300; then
        log_error "AppHub failed to start. Aborting build."
        exit 1
    fi
    
    # 4. Build Dependent Services
    log_info "=== Phase 4: Building Dependent Services ==="
    
    # Determine AppHub URL for build args
    local apphub_port="${APPHUB_PORT:-28080}"
    local external_host="${EXTERNAL_HOST:-$(detect_external_host)}"
    local apphub_url="http://${external_host}:${apphub_port}"
    
    log_info "Using AppHub URL for builds: $apphub_url"
    
    for service in "${DEPENDENT_SERVICES[@]}"; do
        # Pass APPHUB_URL to dependent services
        build_component "$service" "--build-arg" "APPHUB_URL=$apphub_url"
    done
    
    log_info "=== Build Process Completed Successfully ==="
}

# Tag private registry images as local images
# This allows docker-compose to find images that were pulled from a private registry
# and use them with local names (without the registry prefix)
#
# Two modes:
# 1. If PRIVATE_REGISTRY is set: tag images from that specific registry
# 2. Auto-detect mode: scan local images for any registry-prefixed ai-infra images
tag_private_images_as_local() {
    local private_registry="${PRIVATE_REGISTRY:-}"
    local image_tag="${IMAGE_TAG:-v0.3.8}"
    
    # List of ai-infra images that may need tagging
    local images=(
        "ai-infra-frontend"
        "ai-infra-backend"
        "ai-infra-backend-init"
        "ai-infra-nginx"
        "ai-infra-apphub"
        "ai-infra-saltstack"
        "ai-infra-slurm-master"
        "ai-infra-jupyterhub"
        "ai-infra-singleuser"
        "ai-infra-gitea"
        "ai-infra-nightingale"
        "ai-infra-test-containers"
    )
    
    local tagged=0
    local skipped=0
    
    log_info "Checking for images that need local tagging..."
    
    # Mode 1: If PRIVATE_REGISTRY is configured, use it directly
    if [[ -n "$private_registry" ]]; then
        log_info "Using configured private registry: ${private_registry}"
        
        for img in "${images[@]}"; do
            local private_image="${private_registry}${img}:${image_tag}"
            local local_image="${img}:${image_tag}"
            
            # Check if private image exists locally
            if docker image inspect "$private_image" &>/dev/null; then
                # Check if local image already exists
                if docker image inspect "$local_image" &>/dev/null; then
                    skipped=$((skipped + 1))
                else
                    # Tag private image as local
                    if docker tag "$private_image" "$local_image"; then
                        log_info "  ✓ Tagged: ${private_image} -> ${local_image}"
                        tagged=$((tagged + 1))
                    else
                        log_warn "  ✗ Failed to tag: ${private_image}"
                    fi
                fi
            fi
        done
    else
        # Mode 2: Auto-detect registry-prefixed images
        log_info "Auto-detecting registry-prefixed images..."
        
        for img in "${images[@]}"; do
            local local_image="${img}:${image_tag}"
            
            # Skip if local image already exists
            if docker image inspect "$local_image" &>/dev/null; then
                skipped=$((skipped + 1))
                continue
            fi
            
            # Search for any registry-prefixed version of this image
            # Pattern: */ai-infra-xxx:tag or */*/*/ai-infra-xxx:tag
            local found_image=""
            found_image=$(docker images --format '{{.Repository}}:{{.Tag}}' | grep "/${img}:${image_tag}$" | head -1)
            
            if [[ -n "$found_image" ]]; then
                if docker tag "$found_image" "$local_image"; then
                    log_info "  ✓ Tagged: ${found_image} -> ${local_image}"
                    tagged=$((tagged + 1))
                else
                    log_warn "  ✗ Failed to tag: ${found_image}"
                fi
            fi
        done
    fi
    
    if [[ $tagged -gt 0 ]]; then
        log_info "Image tagging complete: $tagged tagged, $skipped already exist"
    elif [[ $skipped -gt 0 ]]; then
        log_info "All $skipped images already exist locally"
    else
        log_info "No registry-prefixed images found to tag"
    fi
}

# 更新运行时环境变量（启动阶段使用）
# 与构建阶段不同，运行时需要检测当前机器的真实 IP
update_runtime_env() {
    log_info "=========================================="
    log_info "🔄 更新运行时环境变量"
    log_info "=========================================="
    
    # 检测当前机器的外部地址
    local detected_host=$(detect_external_host)
    local current_host=$(grep "^EXTERNAL_HOST=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2)
    
    log_info "当前配置的 EXTERNAL_HOST: ${current_host:-<未设置>}"
    log_info "检测到的本机地址: $detected_host"
    
    # 如果 IP 不同，说明是在不同机器上运行
    if [[ "$current_host" != "$detected_host" ]]; then
        log_info "⚠️  检测到环境变化（可能是从其他机器构建的镜像）"
        log_info "   正在更新 EXTERNAL_HOST: $current_host -> $detected_host"
        
        # 更新 .env 文件中的 EXTERNAL_HOST
        update_env_variable "EXTERNAL_HOST" "$detected_host"
        update_env_variable "DOMAIN" "$detected_host"
        
        # 重新加载环境变量
        set -a
        source "$ENV_FILE"
        set +a
        
        # 重新渲染配置模板
        log_info "🔧 重新渲染配置模板..."
        render_all_templates "true"
        
        log_info "✓ 运行时环境变量已更新"
    else
        log_info "✓ EXTERNAL_HOST 配置正确，无需更新"
    fi
}

start_all() {
    log_info "Starting all services (with HA profile for SaltStack multi-master)..."
    local compose_cmd=$(detect_compose_command)
    if [ -z "$compose_cmd" ]; then
        log_error "docker-compose not found!"
        exit 1
    fi
    
    # 【关键】在启动前更新运行时环境变量
    # 这解决了构建阶段与运行阶段在不同机器上 IP 不一致的问题
    update_runtime_env
    
    # Tag private registry images as local if needed
    tag_private_images_as_local
    
    # Use --no-build to prevent rebuilding when images already exist
    # Use --pull never to prevent checking remote registry (important for offline/intranet environments)
    # Use --profile ha to enable SaltStack multi-master high availability
    $compose_cmd --profile ha up -d --no-build --pull never
    log_info "All services started (SaltStack HA enabled)."
}

# ==============================================================================
# Clean Functions - 清理功能
# ==============================================================================

# Clean project images
# Args: $1 = tag (optional), $2 = force (optional)
clean_images() {
    local tag="${1:-}"
    local force="${2:-false}"
    
    log_info "=========================================="
    log_info "Cleaning AI-Infra Docker images"
    log_info "=========================================="
    
    local images_to_remove=()
    
    # Find all ai-infra images
    if [[ -n "$tag" ]]; then
        log_info "Finding images with tag: $tag"
        while IFS= read -r img; do
            [[ -n "$img" ]] && images_to_remove+=("$img")
        done < <(docker images --format '{{.Repository}}:{{.Tag}}' | grep "ai-infra" | grep ":${tag}$")
    else
        log_info "Finding all ai-infra images"
        while IFS= read -r img; do
            [[ -n "$img" ]] && images_to_remove+=("$img")
        done < <(docker images --format '{{.Repository}}:{{.Tag}}' | grep "ai-infra")
    fi
    
    if [[ ${#images_to_remove[@]} -eq 0 ]]; then
        log_info "No ai-infra images found to clean"
        return 0
    fi
    
    log_info "Found ${#images_to_remove[@]} images to remove:"
    for img in "${images_to_remove[@]}"; do
        echo "  • $img"
    done
    
    if [[ "$force" != "true" ]]; then
        echo
        read -p "Are you sure you want to remove these images? [y/N] " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Cancelled"
            return 0
        fi
    fi
    
    local removed=0
    local failed=0
    
    for img in "${images_to_remove[@]}"; do
        if docker rmi "$img" 2>/dev/null; then
            log_info "  ✓ Removed: $img"
            removed=$((removed + 1))
        else
            log_warn "  ✗ Failed to remove: $img (may be in use)"
            failed=$((failed + 1))
        fi
    done
    
    log_info "=========================================="
    log_info "Removed: $removed, Failed: $failed"
    return 0
}

# Clean project volumes
# Args: $1 = force (optional)
clean_volumes() {
    local force="${1:-false}"
    
    log_info "=========================================="
    log_info "Cleaning AI-Infra Docker volumes"
    log_info "=========================================="
    
    local volumes_to_remove=()
    
    # Find all ai-infra related volumes
    while IFS= read -r vol; do
        [[ -n "$vol" ]] && volumes_to_remove+=("$vol")
    done < <(docker volume ls --format '{{.Name}}' | grep -E "ai-infra|ai_infra")
    
    # Also check for compose project volumes
    local compose_project="ai-infra-matrix"
    while IFS= read -r vol; do
        [[ -n "$vol" ]] && volumes_to_remove+=("$vol")
    done < <(docker volume ls --format '{{.Name}}' | grep -E "^${compose_project}_")
    
    # Remove duplicates
    volumes_to_remove=($(printf '%s\n' "${volumes_to_remove[@]}" | sort -u))
    
    if [[ ${#volumes_to_remove[@]} -eq 0 ]]; then
        log_info "No ai-infra volumes found to clean"
        return 0
    fi
    
    log_info "Found ${#volumes_to_remove[@]} volumes to remove:"
    for vol in "${volumes_to_remove[@]}"; do
        echo "  • $vol"
    done
    
    if [[ "$force" != "true" ]]; then
        echo
        read -p "Are you sure you want to remove these volumes? This will DELETE ALL DATA! [y/N] " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Cancelled"
            return 0
        fi
    fi
    
    local removed=0
    local failed=0
    
    for vol in "${volumes_to_remove[@]}"; do
        if docker volume rm "$vol" 2>/dev/null; then
            log_info "  ✓ Removed: $vol"
            removed=$((removed + 1))
        else
            log_warn "  ✗ Failed to remove: $vol (may be in use)"
            failed=$((failed + 1))
        fi
    done
    
    log_info "=========================================="
    log_info "Removed: $removed, Failed: $failed"
    return 0
}

# Stop all project containers
stop_all() {
    log_info "Stopping all AI-Infra services..."
    local compose_cmd=$(detect_compose_command)
    if [ -z "$compose_cmd" ]; then
        log_error "docker-compose not found!"
        return 1
    fi
    
    $compose_cmd down
    log_info "All services stopped."
}

# Clean all: stop containers, remove images and volumes
# Args: $1 = force (optional, "--force" or "true")
clean_all() {
    local force="false"
    
    if [[ "$1" == "--force" || "$1" == "-f" || "$1" == "true" ]]; then
        force="true"
    fi
    
    if [[ "$1" == "--help" || "$1" == "-h" ]]; then
        echo "clean-all - Clean all project Docker resources"
        echo ""
        echo "Usage: $0 clean-all [--force]"
        echo ""
        echo "Options:"
        echo "  --force, -f    Skip confirmation prompts"
        echo ""
        echo "This command will:"
        echo "  1. Stop all running containers"
        echo "  2. Remove all ai-infra Docker images"
        echo "  3. Remove all ai-infra Docker volumes"
        echo "  4. Clean dangling images and build cache"
        echo ""
        echo "⚠️  WARNING: This will DELETE ALL DATA in volumes!"
        return 0
    fi
    
    log_info "=========================================="
    log_info "🧹 Complete cleanup of AI-Infra resources"
    log_info "=========================================="
    
    if [[ "$force" != "true" ]]; then
        echo
        log_warn "⚠️  This will stop all containers, remove all images and DELETE ALL DATA!"
        read -p "Are you sure you want to continue? [y/N] " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log_info "Cancelled"
            return 0
        fi
        # Set force=true for subsequent operations to avoid repeated prompts
        force="true"
    fi
    
    echo
    log_info "Step 1/4: Stopping all containers..."
    stop_all 2>/dev/null || log_warn "No containers to stop or compose not available"
    
    echo
    log_info "Step 2/4: Removing project images..."
    clean_images "" "$force"
    
    echo
    log_info "Step 3/4: Removing project volumes..."
    clean_volumes "$force"
    
    echo
    log_info "Step 4/4: Cleaning dangling resources..."
    # Remove dangling images
    local dangling_count=$(docker images -f "dangling=true" -q | wc -l | tr -d ' ')
    if [[ "$dangling_count" -gt 0 ]]; then
        log_info "Removing $dangling_count dangling images..."
        docker image prune -f 2>/dev/null || true
    fi
    
    # Clean build cache (optional, only if --force)
    if [[ "$force" == "true" ]]; then
        log_info "Cleaning build cache..."
        docker builder prune -f 2>/dev/null || true
    fi
    
    echo
    log_info "=========================================="
    log_info "🎉 Cleanup completed!"
    log_info "=========================================="
    
    # Show remaining resources - use tr to remove newlines and ensure clean numeric output
    local remaining_images
    local remaining_volumes
    remaining_images=$(docker images --format '{{.Repository}}:{{.Tag}}' 2>/dev/null | grep "ai-infra" 2>/dev/null | wc -l | tr -d ' \n' || echo "0")
    remaining_volumes=$(docker volume ls --format '{{.Name}}' 2>/dev/null | grep -E "ai-infra|ai_infra" 2>/dev/null | wc -l | tr -d ' \n' || echo "0")
    
    # Ensure numeric values (default to 0 if empty)
    [[ -z "$remaining_images" ]] && remaining_images=0
    [[ -z "$remaining_volumes" ]] && remaining_volumes=0
    
    if [[ "$remaining_images" != "0" ]] || [[ "$remaining_volumes" != "0" ]]; then
        log_warn "Some resources could not be removed (may be in use):"
        [[ "$remaining_images" != "0" ]] && log_warn "  Images: $remaining_images"
        [[ "$remaining_volumes" != "0" ]] && log_warn "  Volumes: $remaining_volumes"
    fi
    
    return 0
}

# ==============================================================================
# Export Offline Images
# ==============================================================================

# Export all images to tar files for offline deployment
# Args: $1 = output_dir (default: ./offline-images), $2 = tag, $3 = include common images (default: true)
export_offline_images() {
    local output_dir="${1:-./offline-images}"
    local tag="${2:-${IMAGE_TAG:-latest}}"
    local include_common="${3:-true}"
    
    # Show help
    if [[ "$1" == "--help" ]] || [[ "$1" == "-h" ]]; then
        echo "Usage: $0 export-offline [output_dir] [tag] [include_common]"
        echo ""
        echo "Arguments:"
        echo "  output_dir      Output directory (default: ./offline-images)"
        echo "  tag             Image tag (default: $IMAGE_TAG)"
        echo "  include_common  Include common images like mysql, redis, kafka (default: true)"
        echo ""
        echo "Description:"
        echo "  Export all AI-Infra service images and dependency images to tar files"
        echo "  Automatically generates image manifest and import script"
        echo ""
        echo "Examples:"
        echo "  $0 export-offline ./my-images v0.3.8 true"
        echo "  $0 export-offline ./images v0.3.8 false"
        return 0
    fi
    
    log_info "=========================================="
    log_info "📦 Exporting Offline Images"
    log_info "=========================================="
    log_info "Output directory: $output_dir"
    log_info "Image tag: $tag"
    log_info "Include common images: $include_common"
    echo
    
    # Create output directory
    mkdir -p "$output_dir"
    
    discover_services
    
    local exported_count=0
    local failed_count=0
    local failed_images=()
    
    # Phase 1: Export AI-Infra project images
    log_info "=== Phase 1: Exporting AI-Infra service images ==="
    
    local all_services=("${FOUNDATION_SERVICES[@]}" "${DEPENDENT_SERVICES[@]}")
    
    for service in "${all_services[@]}"; do
        local image_name="ai-infra-${service}:${tag}"
        local safe_name=$(echo "$image_name" | sed 's|:|_|g')
        local output_file="${output_dir}/${safe_name}.tar"
        
        log_info "→ Exporting: $image_name"
        if docker image inspect "$image_name" >/dev/null 2>&1; then
            if docker save "$image_name" -o "$output_file"; then
                local file_size=$(du -h "$output_file" | cut -f1)
                log_info "  ✓ Exported: $(basename "$output_file") ($file_size)"
                exported_count=$((exported_count + 1))
            else
                log_warn "  ✗ Failed to export: $image_name"
                failed_images+=("$image_name")
                failed_count=$((failed_count + 1))
            fi
        else
            log_warn "  ! Image not found, skipping: $image_name"
            failed_images+=("$image_name")
            failed_count=$((failed_count + 1))
        fi
    done
    echo
    
    # Phase 2: Export dependency images (from deps.yaml mapping)
    log_info "=== Phase 2: Exporting dependency images ==="
    local dependencies=($(get_dependency_mappings))
    
    for mapping in "${dependencies[@]}"; do
        local source_image="${mapping%%|*}"
        local short_name="${mapping##*|}"
        local safe_name=$(echo "$source_image" | sed 's|/|-|g' | sed 's|:|_|g')
        local output_file="${output_dir}/${safe_name}.tar"
        
        log_info "→ Exporting: $source_image"
        if docker image inspect "$source_image" >/dev/null 2>&1; then
            if docker save "$source_image" -o "$output_file"; then
                local file_size=$(du -h "$output_file" | cut -f1)
                log_info "  ✓ Exported: $(basename "$output_file") ($file_size)"
                exported_count=$((exported_count + 1))
            else
                log_warn "  ✗ Failed to export: $source_image"
                failed_images+=("$source_image")
                failed_count=$((failed_count + 1))
            fi
        else
            log_warn "  ! Image not found, skipping: $source_image"
            failed_images+=("$source_image")
            failed_count=$((failed_count + 1))
        fi
    done
    echo
    
    # Phase 3: Export common/third-party images
    if [[ "$include_common" == "true" ]]; then
        log_info "=== Phase 3: Exporting common/third-party images ==="
        
        for image in "${COMMON_IMAGES[@]}"; do
            local safe_name=$(echo "$image" | sed 's|/|-|g' | sed 's|:|_|g')
            local output_file="${output_dir}/${safe_name}.tar"
            
            log_info "→ Exporting: $image"
            if docker image inspect "$image" >/dev/null 2>&1; then
                if docker save "$image" -o "$output_file"; then
                    local file_size=$(du -h "$output_file" | cut -f1)
                    log_info "  ✓ Exported: $(basename "$output_file") ($file_size)"
                    exported_count=$((exported_count + 1))
                else
                    log_warn "  ✗ Failed to export: $image"
                    failed_images+=("$image")
                    failed_count=$((failed_count + 1))
                fi
            else
                log_warn "  ! Image not found, skipping: $image"
                failed_images+=("$image")
                failed_count=$((failed_count + 1))
            fi
        done
        echo
    fi
    
    # Generate image manifest file
    log_info "📋 Generating image manifest..."
    local manifest_file="${output_dir}/images-manifest.txt"
    cat > "$manifest_file" << EOF
# AI Infrastructure Matrix - Offline Images Manifest
# Generated: $(date)
# Image Tag: $tag
# Include Common Images: $include_common

# AI-Infra Service Images
EOF
    
    for service in "${all_services[@]}"; do
        local image_name="ai-infra-${service}:${tag}"
        local safe_name=$(echo "$image_name" | sed 's|:|_|g')
        local tar_file="${safe_name}.tar"
        if [[ -f "${output_dir}/${tar_file}" ]]; then
            echo "$image_name|$tar_file" >> "$manifest_file"
        fi
    done
    
    echo "" >> "$manifest_file"
    echo "# Dependency Images" >> "$manifest_file"
    
    for mapping in "${dependencies[@]}"; do
        local source_image="${mapping%%|*}"
        local safe_name=$(echo "$source_image" | sed 's|/|-|g' | sed 's|:|_|g')
        local tar_file="${safe_name}.tar"
        if [[ -f "${output_dir}/${tar_file}" ]]; then
            echo "$source_image|$tar_file" >> "$manifest_file"
        fi
    done
    
    if [[ "$include_common" == "true" ]]; then
        echo "" >> "$manifest_file"
        echo "# Common/Third-party Images" >> "$manifest_file"
        
        for image in "${COMMON_IMAGES[@]}"; do
            local safe_name=$(echo "$image" | sed 's|/|-|g' | sed 's|:|_|g')
            local tar_file="${safe_name}.tar"
            if [[ -f "${output_dir}/${tar_file}" ]]; then
                echo "$image|$tar_file" >> "$manifest_file"
            fi
        done
    fi
    
    # Generate import script
    log_info "📜 Generating import script..."
    local import_script="${output_dir}/import-images.sh"
    cat > "$import_script" << 'IMPORT_SCRIPT_EOF'
#!/bin/bash

# AI Infrastructure Matrix - Offline Images Import Script
# Usage: ./import-images.sh [images_directory]

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
IMAGES_DIR="${1:-$SCRIPT_DIR}"
MANIFEST_FILE="${IMAGES_DIR}/images-manifest.txt"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

if [[ ! -f "$MANIFEST_FILE" ]]; then
    log_error "Manifest file not found: $MANIFEST_FILE"
    exit 1
fi

log_info "=========================================="
log_info "Importing Offline Images"
log_info "=========================================="
log_info "Images directory: $IMAGES_DIR"
log_info "Manifest file: $MANIFEST_FILE"
echo

imported_count=0
failed_count=0

while IFS='|' read -r image_name tar_file; do
    # Skip comments and empty lines
    [[ "$image_name" =~ ^[[:space:]]*# ]] && continue
    [[ -z "$image_name" ]] && continue
    
    tar_path="${IMAGES_DIR}/${tar_file}"
    
    if [[ -f "$tar_path" ]]; then
        log_info "→ Importing: $image_name"
        if docker load -i "$tar_path"; then
            log_info "  ✓ Imported successfully: $image_name"
            imported_count=$((imported_count + 1))
        else
            log_error "  ✗ Failed to import: $image_name"
            failed_count=$((failed_count + 1))
        fi
    else
        log_error "  ✗ Tar file not found: $tar_path"
        failed_count=$((failed_count + 1))
    fi
done < "$MANIFEST_FILE"

echo
log_info "=========================================="
log_info "Import completed: $imported_count success, $failed_count failed"

if [[ $failed_count -eq 0 ]]; then
    log_info "🎉 All images imported successfully!"
    echo
    log_info "Next steps:"
    log_info "  1. Check images: docker images | grep -E 'ai-infra|postgres|redis'"
    log_info "  2. Start services: docker compose --profile ha up -d"
else
    log_error "Some images failed to import. Please check the errors above."
fi
IMPORT_SCRIPT_EOF
    
    chmod +x "$import_script"
    
    # Calculate total size
    local total_size=$(du -sh "$output_dir" | cut -f1)
    
    # Print summary
    log_info "=========================================="
    log_info "🎉 Offline Export Complete!"
    log_info "=========================================="
    echo
    log_info "📊 Export Statistics:"
    log_info "  • Exported: $exported_count images"
    log_info "  • Failed: $failed_count images"
    log_info "  • Total size: $total_size"
    echo
    
    if [[ ${#failed_images[@]} -gt 0 ]]; then
        log_warn "⚠️  Failed images:"
        for img in "${failed_images[@]}"; do
            log_warn "    - $img"
        done
        echo
    fi
    
    log_info "📁 Output files:"
    log_info "  • Images directory: $output_dir"
    log_info "  • Manifest file: $manifest_file"
    log_info "  • Import script: $import_script"
    echo
    log_info "📋 Usage instructions:"
    log_info "  1. Copy the entire '$output_dir' directory to the offline environment"
    log_info "  2. Run: cd $output_dir && ./import-images.sh"
    log_info "  3. Start services: docker compose --profile ha up -d"
    
    return 0
}

print_help() {
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Global Options (can be used with any command):"
    echo "  --force, -f, --no-cache    Force rebuild without Docker cache"
    echo ""
    echo "Environment Commands:"
    echo "  init-env [host]     Initialize/sync .env file (auto-detect EXTERNAL_HOST)"
    echo "  init-env --force    Force re-initialize all environment variables"
    echo ""
    echo "Build Commands:"
    echo "  build-all, all           Build all components in the correct order"
    echo "  build-all --force        Force rebuild all (no cache, re-render templates)"
    echo "  [component]              Build a specific component (e.g., backend, frontend)"
    echo "  [component] --force      Force rebuild a component without cache"
    echo ""
    echo "Template Commands:"
    echo "  render, sync        Render all Dockerfile.tpl templates from .env config"
    echo "  render --force      Force re-render all templates (ignore cache)"
    echo ""
    echo "Service Commands:"
    echo "  start-all           Start all services (with SaltStack HA multi-master)"
    echo "  stop-all            Stop all services"
    echo "  tag-images          Tag private registry images as local (for intranet)"
    echo ""
    echo "Pull Commands (Smart Mode):"
    echo "  prefetch            Prefetch all base images from Dockerfiles"
    echo "  pull-common         Pull common/third-party images (mysql, kafka, redis, etc.)"
    echo "  pull-all                              Internet mode: pull from Docker Hub"
    echo "  pull-all <registry/project> [tag]    Intranet mode: pull from private registry"
    echo "  deps-pull <registry/project> [tag]   Pull dependency images from registry"
    echo ""
    echo "Push Commands:"
    echo "  push <service> <registry/project> [tag]  Push single service to registry"
    echo "  push-all <registry/project> [tag]        Push all images (4 phases)"
    echo "  push-dep <registry/project> [tag]        Push dependency images to registry"
    echo ""
    echo "  ⚠️  Harbor registries require project name in path:"
    echo "     ✓ harbor.example.com/ai-infra     (correct)"
    echo "     ✗ harbor.example.com              (wrong - missing project)"
    echo ""
    echo "Clean Commands:"
    echo "  clean-images [tag]  Remove ai-infra Docker images (optional: specific tag)"
    echo "  clean-volumes       Remove ai-infra Docker volumes"
    echo "  clean-all [--force] Remove all images, volumes and stop containers"
    echo ""
    echo "Download Commands:"
    echo "  download-deps       Download third-party dependencies to third_party/"
    echo "                      (Prometheus, Node Exporter, Alertmanager, Categraf, etc.)"
    echo ""
    echo "Offline Export Commands:"
    echo "  export-offline [dir] [tag] [include_common]  Export images to tar files"
    echo "                                               default: ./offline-images, latest, true"
    echo ""
    echo "Template Variables (from .env):"
    echo "  === Mirror Configuration (Build-time) ==="
    echo "  GITHUB_MIRROR       GitHub mirror URL prefix (e.g., https://ghfast.top/)"
    echo "  APT_MIRROR          APT mirror for Ubuntu/Debian (e.g., mirrors.aliyun.com)"
    echo "  YUM_MIRROR          YUM mirror for Rocky/CentOS"
    echo "  ALPINE_MIRROR       Alpine mirror"
    echo "  GO_PROXY            Go module proxy"
    echo "  PYPI_INDEX_URL      PyPI mirror"
    echo "  NPM_REGISTRY        npm registry mirror"
    echo ""
    echo "  === Base Image Versions ==="
    echo "  UBUNTU_VERSION      Ubuntu base image version"
    echo "  ROCKYLINUX_VERSION  Rocky Linux version"
    echo "  GOLANG_VERSION      Go version"
    echo ""
    echo "  === Component Versions ==="
    echo "  SLURM_VERSION       SLURM version to build"
    echo "  SALTSTACK_VERSION   SaltStack version"
    echo "  CATEGRAF_VERSION    Categraf version"
    echo "  GITEA_VERSION       Gitea version"
    echo ""
    echo "Examples:"
    echo "  # Environment setup"
    echo "  $0 init-env                        # Auto-detect and initialize .env"
    echo "  $0 init-env 192.168.0.100          # Set specific EXTERNAL_HOST"
    echo "  $0 init-env --force                # Force re-initialize"
    echo ""
    echo "  # Template rendering"
    echo "  $0 render                          # Render templates from .env"
    echo "  $0 render --force                  # Force re-render all templates"
    echo ""
    echo "  # Building"
    echo "  $0 build-all                       # Build all services"
    echo "  $0 backend                         # Build single service"
    echo ""
    echo "  # Internet mode (Docker Hub)"
    echo "  $0 prefetch                        # Prefetch base images"
    echo "  $0 pull-all                        # Pull common images from Docker Hub"
    echo ""
    echo "  # Intranet mode (Private Registry)"
    echo "  $0 push-all harbor.example.com/ai-infra v0.3.8    # Push to registry"
    echo "  $0 pull-all harbor.example.com/ai-infra v0.3.8    # Pull from registry"
    echo ""
    echo "  # Offline export"
    echo "  $0 export-offline ./offline-images v0.3.8         # Export all images to tar"
    echo "  $0 export-offline ./images v0.3.8 false           # Export without common images"
    echo ""
    echo "  # Cleanup"
    echo "  $0 clean-all --force"
    echo ""
    echo "  # Download third-party dependencies (for faster AppHub builds)"
    echo "  $0 download-deps                       # Download to third_party/"
}

# ==============================================================================
# 4. Main Execution
# ==============================================================================

if [ $# -eq 0 ]; then
    print_help
    exit 0
fi

# Parse global options first (--force, --no-cache, -f)
# These can appear anywhere in the command line
FORCE_BUILD=false
FORCE_RENDER=false
REMAINING_ARGS=()

for arg in "$@"; do
    case "$arg" in
        --force|-f|--no-cache)
            FORCE_BUILD=true
            FORCE_RENDER=true
            ;;
        *)
            REMAINING_ARGS+=("$arg")
            ;;
    esac
done

# Show force mode message after parsing
if [[ "$FORCE_BUILD" == "true" ]]; then
    log_info "🔧 Force mode enabled (--no-cache for Docker builds)"
fi

COMMAND="${REMAINING_ARGS[0]:-}"
ARG2="${REMAINING_ARGS[1]:-}"
ARG3="${REMAINING_ARGS[2]:-}"
ARG4="${REMAINING_ARGS[3]:-}"

case "$COMMAND" in
    init-env)
        # 初始化或同步 .env 文件
        if [[ "$FORCE_BUILD" == "true" ]]; then
            log_info "Force re-initializing .env..."
            init_env_file "true"
        elif [[ -n "$ARG2" ]]; then
            # 使用指定的 EXTERNAL_HOST
            log_info "Setting EXTERNAL_HOST=$ARG2..."
            update_env_variable "EXTERNAL_HOST" "$ARG2"
            update_env_variable "DOMAIN" "$ARG2"
            log_info "✓ EXTERNAL_HOST updated to $ARG2"
        else
            init_env_file "true"
        fi
        # 显示当前配置
        echo
        log_info "Current environment configuration:"
        grep -E "^(EXTERNAL_HOST|DOMAIN|EXTERNAL_PORT|EXTERNAL_SCHEME)=" "$ENV_FILE"
        ;;
    build-all|all)
        if [[ "$FORCE_BUILD" == "true" ]]; then
            build_all "true"
        else
            build_all
        fi
        ;;
    render|sync|sync-templates)
        if [[ "$FORCE_BUILD" == "true" ]] || [[ "$FORCE_RENDER" == "true" ]]; then
            render_all_templates "true"
        else
            render_all_templates
        fi
        ;;
    start-all)
        start_all
        ;;
    tag-images)
        tag_private_images_as_local
        ;;
    stop-all)
        stop_all
        ;;
    clean-images)
        clean_images "$ARG2" "${ARG3:-false}"
        ;;
    clean-volumes)
        clean_volumes "${ARG2:-false}"
        ;;
    clean-all)
        clean_all "$ARG2"
        ;;
    prefetch)
        prefetch_base_images "$ARG2"
        ;;
    pull-common)
        pull_common_images
        ;;
    pull-all)
        # Smart mode: without registry -> Docker Hub, with registry -> private registry
        pull_all_services "$ARG2" "${ARG3:-${IMAGE_TAG:-latest}}"
        ;;
    deps-pull)
        if [[ -z "$ARG2" ]]; then
            log_error "Registry is required"
            log_info "Usage: $0 deps-pull <registry> [tag]"
            exit 1
        fi
        pull_and_tag_dependencies "$ARG2" "${ARG3:-${IMAGE_TAG:-latest}}"
        ;;
    push)
        if [[ -z "$ARG2" ]]; then
            log_error "Service name is required"
            log_info "Usage: $0 push <service> <registry> [tag]"
            exit 1
        fi
        if [[ -z "$ARG3" ]]; then
            log_error "Registry is required"
            log_info "Usage: $0 push <service> <registry> [tag]"
            exit 1
        fi
        push_service "$ARG2" "${ARG4:-${IMAGE_TAG:-latest}}" "$ARG3"
        ;;
    push-all)
        if [[ -z "$ARG2" ]]; then
            log_error "Registry is required"
            log_info "Usage: $0 push-all <registry> [tag]"
            exit 1
        fi
        push_all_services "$ARG2" "${ARG3:-${IMAGE_TAG:-latest}}"
        ;;
    push-dep|push-dependencies)
        if [[ -z "$ARG2" ]]; then
            log_error "Registry is required"
            log_info "Usage: $0 push-dep <registry> [tag]"
            exit 1
        fi
        push_all_dependencies "$ARG2" "${ARG3:-${IMAGE_TAG:-latest}}"
        ;;
    export-offline)
        # Export all images to tar files for offline deployment
        export_offline_images "$ARG2" "${ARG3:-${IMAGE_TAG:-latest}}" "${ARG4:-true}"
        ;;
    download-deps)
        # Download third-party dependencies to third_party/
        log_info "📦 Downloading third-party dependencies..."
        if [[ -x "$SCRIPT_DIR/scripts/download_third_party.sh" ]]; then
            "$SCRIPT_DIR/scripts/download_third_party.sh"
            log_info "✅ Third-party dependencies downloaded to third_party/"
            log_info "💡 These files will be used during AppHub build for faster builds"
        else
            log_error "download_third_party.sh not found or not executable"
            exit 1
        fi
        ;;
    help|--help|-h)
        print_help
        ;;
    "")
        print_help
        ;;
    *)
        # Single component build - collect all non-option arguments as components
        components=()
        for arg in "${REMAINING_ARGS[@]}"; do
            # Skip if it's an option (starts with -)
            [[ "$arg" == -* ]] && continue
            components+=("$arg")
        done
        
        if [[ ${#components[@]} -eq 0 ]]; then
            log_error "No component specified"
            print_help
            exit 1
        fi
        
        log_info "Building components: ${components[*]}"
        [[ "$FORCE_BUILD" == "true" ]] && log_info "  with --no-cache (force rebuild)"
        
        for component in "${components[@]}"; do
            build_component "$component"
        done
        ;;
esac
