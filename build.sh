#!/bin/bash

# AI Infrastructure Matrix - 精简构建脚本
# 版本: v1.0.0
# 专注于 src/ 目录下的 Dockerfile 构建

set -e

# 操作系统检测
detect_os() {
    if [[ "$OSTYPE" == "darwin"* ]]; then
        echo "macOS"
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        echo "Linux"
    else
        echo "Other"
    fi
}

# 全局变量
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VERSION="   1.0.0"
CONFIG_FILE="$SCRIPT_DIR/config.toml"
OS_TYPE=$(detect_os)
FORCE_REBUILD=false  # 强制重新构建标志

# 基本输出函数（早期定义，供其他函数使用）
print_error() {
    echo -e "\033[31m[ERROR]\033[0m $1"
}

print_info() {
    echo -e "\033[32m[INFO]\033[0m $1"
}

# ==========================================
# 配置文件解析功能
# ==========================================

# 读取TOML配置文件中的值
read_config() {
    local section="$1"
    local key="$2"
    local subsection="$3"
    
    if [[ ! -f "$CONFIG_FILE" ]]; then
        # 配置文件不存在时返回空值，由调用者处理默认值
        return 1
    fi
    
    if [[ -n "$subsection" ]]; then
        # 读取嵌套配置 [section.subsection]
        awk -F' *= *' -v section="$section" -v subsection="$subsection" -v key="$key" '
            /^\[[[:space:]]*[^.]+\.[^]]+\]/ {
                # 匹配 [section.subsection] 格式
                gsub(/^\[|\]$/, "")
                split($0, parts, "\\.")
                if (parts[1] == section && parts[2] == subsection) {
                    in_target = 1
                } else {
                    in_target = 0
                }
                next
            }
            /^\[/ { in_target = 0; next }
            in_target && $1 == key {
                gsub(/^"/, "", $2)
                gsub(/"$/, "", $2)
                print $2
                exit
            }
        ' "$CONFIG_FILE"
    else
        # 读取简单配置 [section]
        awk -F' *= *' -v section="$section" -v key="$key" '
            /^\[[[:space:]]*[^.]+\]/ {
                gsub(/^\[|\]$/, "")
                if ($0 == section) {
                    in_target = 1
                } else {
                    in_target = 0
                }
                next
            }
            /^\[/ { in_target = 0; next }
            in_target && $1 == key {
                gsub(/^"/, "", $2)
                gsub(/"$/, "", $2)
                print $2
                exit
            }
        ' "$CONFIG_FILE"
    fi
}

# 获取所有服务名称
get_all_services() {
    if [[ ! -f "$CONFIG_FILE" ]]; then
        echo "backend frontend jupyterhub nginx saltstack singleuser gitea backend-init"
        return
    fi
    
    awk '
        /^\[services\.[^]]+\]/ {
            gsub(/^\[services\.|\]$/, "")
            print $0
        }
    ' "$CONFIG_FILE" | sort
}

# 获取所有依赖镜像（包含测试工具和构建依赖）
get_all_dependencies() {
    if [[ ! -f "$CONFIG_FILE" ]]; then
        echo "postgres:15-alpine redis:7-alpine osixia/openldap:stable osixia/phpldapadmin:stable tecnativa/tcp-proxy redislabs/redisinsight:latest nginx:1.27-alpine minio/minio:latest node:22-alpine nginx:stable-alpine-perl golang:1.25-alpine python:3.13-alpine gitea/gitea:1.24.5 jupyter/base-notebook:latest"
        return
    fi
    
    awk -F' *= *' '
        /^\[dependencies\]/ { in_dependencies = 1; next }
        /^\[/ { in_dependencies = 0; next }
        in_dependencies && NF > 1 {
            gsub(/^"/, "", $2)
            gsub(/"$/, "", $2)
            print $2
        }
    ' "$CONFIG_FILE" | tr '\n' ' '
}

# 获取生产环境依赖镜像（移除测试工具和构建依赖）
get_production_dependencies() {
    if [[ ! -f "$CONFIG_FILE" ]]; then
        echo "postgres:15-alpine redis:7-alpine tecnativa/tcp-proxy nginx:1.27-alpine minio/minio:latest"
        return
    fi
    
    awk -F' *= *' '
        /^\[dependencies\]/ { in_dependencies = 1; next }
        /^\[/ { in_dependencies = 0; next }
        in_dependencies && NF > 1 {
            gsub(/^"/, "", $2)
            gsub(/"$/, "", $2)
            # 排除测试工具和LDAP服务
            if ($2 !~ /phpldapadmin/ && $2 !~ /redisinsight/ && $2 !~ /openldap/) {
                print $2
            }
        }
    ' "$CONFIG_FILE" | tr '\n' ' '
}

# 初始化配置
DEFAULT_IMAGE_TAG=$(read_config "project" "version" 2>/dev/null || echo "")
[[ -z "$DEFAULT_IMAGE_TAG" ]] && DEFAULT_IMAGE_TAG="v0.3.5"

# 动态加载服务和依赖配置
SRC_SERVICES=$(get_all_services | tr '\n' ' ')
DEPENDENCY_IMAGES=$(get_all_dependencies | tr '\n' ' ')

# 动态收集依赖镜像函数
collect_dependency_images() {
    # 优先使用配置文件中的依赖镜像列表
    if [[ -f "$CONFIG_FILE" ]]; then
        echo "$DEPENDENCY_IMAGES"
        return
    fi
    
    # 使用统一的静态依赖列表，确保与get_all_dependencies一致
    echo "postgres:15-alpine redis:7-alpine osixia/openldap:stable osixia/phpldapadmin:stable tecnativa/tcp-proxy redislabs/redisinsight:latest nginx:1.27-alpine minio/minio:latest node:22-alpine nginx:stable-alpine-perl golang:1.25-alpine python:3.13-alpine gitea/gitea:1.24.5 jupyter/base-notebook:latest"
}

# Mock 数据测试相关配置
MOCK_DATA_ENABLED="${MOCK_DATA_ENABLED:-false}"
MOCK_POSTGRES_IMAGE="postgres:15-alpine"
MOCK_REDIS_IMAGE="redis:7-alpine"

# 获取服务对应的路径
get_service_path() {
    local service="$1"
    
    # 从配置文件读取路径
    local path=$(read_config "services" "path" "$service" 2>/dev/null || echo "")
    
    # 如果配置文件中没有，使用后备方案
    if [[ -z "$path" ]]; then
        case "$service" in
            "backend") echo "src/backend" ;;
            "frontend") echo "src/frontend" ;;
            "jupyterhub") echo "src/jupyterhub" ;;
            "nginx") echo "src/nginx" ;;
            "saltstack") echo "src/saltstack" ;;
            "singleuser") echo "src/singleuser" ;;
            "gitea") echo "src/gitea" ;;
            "backend-init") echo "src/backend" ;;  # backend-init 使用 backend 的 Dockerfile
            *) echo "" ;;
        esac
    else
        echo "$path"
    fi
}

# 颜色输出函数（扩展）
print_success() {
    echo -e "\033[32m[SUCCESS]\033[0m $1"
}

print_warning() {
    echo -e "\033[33m[WARNING]\033[0m $1"
}

# ==========================================
# 模板渲染功能
# ==========================================

# 从 docker-compose.yml 加载环境变量
load_environment_variables() {
    local env_file="$SCRIPT_DIR/.env.prod"
    
    # 检测外部主机地址
    local detected_host="localhost"
    local detected_port="8080"
    
    if [[ -f "$SCRIPT_DIR/scripts/detect-external-host.sh" ]]; then
        detected_host=$(cd "$SCRIPT_DIR" && bash scripts/detect-external-host.sh | grep "检测到的主机地址:" | cut -d: -f2 | xargs)
        if [[ -n "$detected_host" && "$detected_host" != "localhost" ]]; then
            print_info "自动检测到外部主机: $detected_host"
        else
            detected_host="localhost"
        fi
    fi
    
    # 从 .env.prod 文件加载变量并进行动态替换
    if [[ -f "$env_file" ]]; then
        while IFS='=' read -r key value; do
            # 跳过注释和空行
            [[ "$key" =~ ^[[:space:]]*# ]] && continue
            [[ -z "$key" ]] && continue
            
            # 移除引号
            value=${value#\"}
            value=${value%\"}
            value=${value#\'}
            value=${value%\'}
            
            # 动态替换变量中的占位符
            value=${value//\$\{EXTERNAL_HOST\}/$detected_host}
            value=${value//\$\{EXTERNAL_PORT\}/8080}
            value=${value//\$\{EXTERNAL_SCHEME\}/http}
            
            eval "ENV_${key}=\"$value\""
        done < "$env_file"
    fi
    
    # 设置动态变量
    EXTERNAL_HOST="${ENV_EXTERNAL_HOST:-$detected_host}"
    EXTERNAL_PORT="${ENV_EXTERNAL_PORT:-8080}"
    EXTERNAL_SCHEME="${ENV_EXTERNAL_SCHEME:-http}"
    
    # 从 docker-compose.yml 提取默认值
    if [[ -f "$SCRIPT_DIR/docker-compose.yml" ]]; then
        # 提取环境变量默认值
        BACKEND_HOST="${ENV_BACKEND_HOST:-backend}"
        BACKEND_PORT="${ENV_BACKEND_PORT:-8082}"
        FRONTEND_HOST="${ENV_FRONTEND_HOST:-frontend}"
        FRONTEND_PORT="${ENV_FRONTEND_PORT:-80}"
        JUPYTERHUB_HOST="${ENV_JUPYTERHUB_HOST:-jupyterhub}"
        JUPYTERHUB_PORT="${ENV_JUPYTERHUB_PORT:-8000}"
        EXTERNAL_SCHEME="${ENV_EXTERNAL_SCHEME:-http}"
        EXTERNAL_HOST="${ENV_EXTERNAL_HOST:-$detected_host}"
        EXTERNAL_PORT="${ENV_EXTERNAL_PORT:-8080}"
        GITEA_ALIAS_ADMIN_TO="${ENV_GITEA_ALIAS_ADMIN_TO:-admin}"
        GITEA_ADMIN_EMAIL="${ENV_GITEA_ADMIN_EMAIL:-admin@example.com}"
    fi
}

# 渲染模板文件
render_template() {
    local template_file="$1"
    local output_file="$2"
    
    if [[ ! -f "$template_file" ]]; then
        print_error "模板文件不存在: $template_file"
        return 1
    fi
    
    print_info "渲染模板: $template_file -> $output_file"
    
    # 创建输出目录
    local output_dir
    output_dir=$(dirname "$output_file")
    mkdir -p "$output_dir"
    
    # 使用Python渲染器来处理模板变量替换，避免shell sed的特殊字符问题
    if [[ -f "$SCRIPT_DIR/scripts/template_renderer.py" ]]; then
        python3 "$SCRIPT_DIR/scripts/template_renderer.py" "$template_file" "$output_file"
        if [[ $? -eq 0 ]]; then
            print_success "✓ 模板渲染完成: $output_file"
            return 0
        else
            print_warning "Python模板渲染失败，尝试shell方法..."
        fi
    fi
    
    # 备用方案：使用shell进行模板变量替换
    local template_content
    template_content=$(<"$template_file")
    
    # 设置所有模板变量的默认值
    local BACKEND_HOST="${BACKEND_HOST:-backend}"
    local BACKEND_PORT="${BACKEND_PORT:-8082}"
    local FRONTEND_HOST="${FRONTEND_HOST:-frontend}"
    local FRONTEND_PORT="${FRONTEND_PORT:-3000}"
    local JUPYTERHUB_HOST="${JUPYTERHUB_HOST:-jupyterhub}"
    local JUPYTERHUB_PORT="${JUPYTERHUB_PORT:-8000}"
    local EXTERNAL_SCHEME="${EXTERNAL_SCHEME:-http}"
    local EXTERNAL_HOST="${EXTERNAL_HOST:-localhost}"
    local GITEA_ALIAS_ADMIN_TO="${GITEA_ALIAS_ADMIN_TO:-admin@example.com}"
    local GITEA_ADMIN_EMAIL="${GITEA_ADMIN_EMAIL:-admin@example.com}"
    
    # JupyterHub特定变量
    local ENVIRONMENT="${ENVIRONMENT:-development}"
    local AUTH_TYPE="${AUTH_TYPE:-local}"
    local GENERATION_TIME="$(date)"
    local JUPYTERHUB_HUB_PORT="${JUPYTERHUB_HUB_PORT:-8081}"
    local JUPYTERHUB_BASE_URL="${JUPYTERHUB_BASE_URL:-/jupyter/}"
    local JUPYTERHUB_HUB_CONNECT_HOST="${JUPYTERHUB_HUB_CONNECT_HOST:-jupyterhub}"
    local JUPYTERHUB_PUBLIC_URL="${JUPYTERHUB_PUBLIC_URL:-http://localhost:8080/jupyter/}"
    local CONFIGPROXY_AUTH_TOKEN="${CONFIGPROXY_AUTH_TOKEN:-ai-infra-proxy-token-dev}"
    local JUPYTERHUB_DB_URL="${JUPYTERHUB_DB_URL:-sqlite:///jupyterhub.sqlite}"
    local JUPYTERHUB_LOG_LEVEL="${JUPYTERHUB_LOG_LEVEL:-INFO}"
    local SESSION_TIMEOUT_DAYS="${SESSION_TIMEOUT_DAYS:-7}"
    local SINGLEUSER_IMAGE="${SINGLEUSER_IMAGE:-ai-infra-singleuser:latest}"
    local DOCKER_NETWORK="${DOCKER_NETWORK:-ai-infra-matrix_default}"
    local JUPYTERHUB_MEM_LIMIT="${JUPYTERHUB_MEM_LIMIT:-2G}"
    local JUPYTERHUB_CPU_LIMIT="${JUPYTERHUB_CPU_LIMIT:-1.0}"
    local JUPYTERHUB_MEM_GUARANTEE="${JUPYTERHUB_MEM_GUARANTEE:-1G}"
    local JUPYTERHUB_CPU_GUARANTEE="${JUPYTERHUB_CPU_GUARANTEE:-0.5}"
    local USER_STORAGE_CAPACITY="${USER_STORAGE_CAPACITY:-10Gi}"
    local JUPYTERHUB_STORAGE_CLASS="${JUPYTERHUB_STORAGE_CLASS:-default}"
    local SHARED_STORAGE_PATH="${SHARED_STORAGE_PATH:-/srv/shared-notebooks}"
    local AI_INFRA_BACKEND_URL="${AI_INFRA_BACKEND_URL:-http://backend:8082}"
    local KUBERNETES_NAMESPACE="${KUBERNETES_NAMESPACE:-ai-infra-users}"
    local KUBERNETES_SERVICE_ACCOUNT="${KUBERNETES_SERVICE_ACCOUNT:-ai-infra-matrix-jupyterhub}"
    local JUPYTERHUB_START_TIMEOUT="${JUPYTERHUB_START_TIMEOUT:-300}"
    local JUPYTERHUB_HTTP_TIMEOUT="${JUPYTERHUB_HTTP_TIMEOUT:-30}"
    local JWT_SECRET="${JWT_SECRET:-}"
    local JUPYTERHUB_AUTO_LOGIN="${JUPYTERHUB_AUTO_LOGIN:-False}"
    local AUTH_REFRESH_AGE="${AUTH_REFRESH_AGE:-3600}"
    local ADMIN_USERS="${ADMIN_USERS:-'admin'}"
    
    
    # 备用shell方法：只处理基础变量替换，使用envsubst
    if command -v envsubst >/dev/null 2>&1; then
        # 导出所有变量供envsubst使用
        export BACKEND_HOST BACKEND_PORT FRONTEND_HOST FRONTEND_PORT
        export JUPYTERHUB_HOST JUPYTERHUB_PORT EXTERNAL_SCHEME EXTERNAL_HOST
        export GITEA_ALIAS_ADMIN_TO GITEA_ADMIN_EMAIL
        export ENVIRONMENT AUTH_TYPE GENERATION_TIME
        export JUPYTERHUB_HUB_PORT JUPYTERHUB_BASE_URL JUPYTERHUB_HUB_CONNECT_HOST
        export JUPYTERHUB_PUBLIC_URL CONFIGPROXY_AUTH_TOKEN JUPYTERHUB_DB_URL
        export JUPYTERHUB_LOG_LEVEL SESSION_TIMEOUT_DAYS SINGLEUSER_IMAGE
        export DOCKER_NETWORK JUPYTERHUB_MEM_LIMIT JUPYTERHUB_CPU_LIMIT
        export JUPYTERHUB_MEM_GUARANTEE JUPYTERHUB_CPU_GUARANTEE
        export USER_STORAGE_CAPACITY JUPYTERHUB_STORAGE_CLASS SHARED_STORAGE_PATH
        export AI_INFRA_BACKEND_URL KUBERNETES_NAMESPACE KUBERNETES_SERVICE_ACCOUNT
        export JUPYTERHUB_START_TIMEOUT JUPYTERHUB_HTTP_TIMEOUT JWT_SECRET
        export JUPYTERHUB_AUTO_LOGIN AUTH_REFRESH_AGE ADMIN_USERS
        export AUTH_CONFIG SPAWNER_CONFIG SHARED_STORAGE_CONFIG ADDITIONAL_CONFIG
        
        echo "$template_content" | envsubst > "$output_file"
    else
        print_warning "envsubst不可用，使用基础字符串替换"
        echo "$template_content" > "$output_file"
    fi
    
    print_success "✓ 模板渲染完成: $output_file"
}

# 渲染所有nginx模板
render_nginx_templates() {
    print_info "===========================================" 
    print_info "渲染 Nginx 配置模板"
    print_info "==========================================="
    
    # 加载环境变量
    load_environment_variables
    
    local template_dir="$SCRIPT_DIR/src/nginx/templates"
    local output_dir="$SCRIPT_DIR/src/nginx"
    
    if [[ ! -d "$template_dir" ]]; then
        print_error "模板目录不存在: $template_dir"
        return 1
    fi
    
    # 渲染主配置文件
    render_template "$template_dir/conf.d/server-main.conf.tpl" "$output_dir/conf.d/server-main.conf"
    
    # 渲染includes配置文件  
    render_template "$template_dir/conf.d/includes/gitea.conf.tpl" "$output_dir/conf.d/includes/gitea.conf"
    render_template "$template_dir/conf.d/includes/jupyterhub.conf.tpl" "$output_dir/conf.d/includes/jupyterhub.conf"
    
    print_success "✓ Nginx 模板渲染完成"
    echo
}

# 渲染JupyterHub配置模板
render_jupyterhub_templates() {
    print_info "===========================================" 
    print_info "渲染 JupyterHub 配置模板"
    print_info "==========================================="
    
    # 加载环境变量
    load_environment_variables
    
    local template_dir="$SCRIPT_DIR/src/jupyterhub/templates"
    local output_dir="$SCRIPT_DIR/src/jupyterhub"
    
    if [[ ! -d "$template_dir" ]]; then
        print_error "JupyterHub模板目录不存在: $template_dir"
        return 1
    fi
    
    # 设置JupyterHub特定的环境变量
    setup_jupyterhub_variables
    
    # 读取和渲染子模板内容
    local auth_config=""
    local spawner_config=""
    local shared_storage_config=""
    local additional_config=""
    
    # 根据环境和配置选择认证方式
    if [[ "${USE_CUSTOM_AUTH:-false}" == "true" ]]; then
        if [[ -f "$template_dir/auth_backend.py.tpl" ]]; then
            # 先渲染认证模板到临时文件，再读取内容
            local temp_auth_file="$output_dir/.temp_auth_config.py"
            python3 "$SCRIPT_DIR/scripts/template_renderer.py" "$template_dir/auth_backend.py.tpl" "$temp_auth_file"
            if [[ -f "$temp_auth_file" ]]; then
                auth_config=$(<"$temp_auth_file")
                rm -f "$temp_auth_file"
            fi
        fi
    else
        if [[ -f "$template_dir/auth_local.py.tpl" ]]; then
            # 先渲染认证模板到临时文件，再读取内容
            local temp_auth_file="$output_dir/.temp_auth_config.py"
            python3 "$SCRIPT_DIR/scripts/template_renderer.py" "$template_dir/auth_local.py.tpl" "$temp_auth_file"
            if [[ -f "$temp_auth_file" ]]; then
                auth_config=$(<"$temp_auth_file")
                rm -f "$temp_auth_file"
            fi
        fi
    fi
    
    # 根据环境选择Spawner配置
    if [[ "${ENVIRONMENT:-development}" == "production" || "${JUPYTERHUB_SPAWNER:-docker}" == "kubernetes" ]]; then
        if [[ -f "$template_dir/spawner_kubernetes.py.tpl" ]]; then
            # 先渲染Spawner模板到临时文件，再读取内容
            local temp_spawner_file="$output_dir/.temp_spawner_config.py"
            python3 "$SCRIPT_DIR/scripts/template_renderer.py" "$template_dir/spawner_kubernetes.py.tpl" "$temp_spawner_file"
            if [[ -f "$temp_spawner_file" ]]; then
                spawner_config=$(<"$temp_spawner_file")
                rm -f "$temp_spawner_file"
            fi
            
            # 处理共享存储配置
            if [[ -f "$template_dir/shared_storage_k8s.py.tpl" ]]; then
                local temp_storage_file="$output_dir/.temp_storage_config.py"
                python3 "$SCRIPT_DIR/scripts/template_renderer.py" "$template_dir/shared_storage_k8s.py.tpl" "$temp_storage_file"
                if [[ -f "$temp_storage_file" ]]; then
                    shared_storage_config=$(<"$temp_storage_file")
                    rm -f "$temp_storage_file"
                fi
            fi
        fi
    else
        if [[ -f "$template_dir/spawner_docker.py.tpl" ]]; then
            # 先渲染Spawner模板到临时文件，再读取内容
            local temp_spawner_file="$output_dir/.temp_spawner_config.py"
            python3 "$SCRIPT_DIR/scripts/template_renderer.py" "$template_dir/spawner_docker.py.tpl" "$temp_spawner_file"
            if [[ -f "$temp_spawner_file" ]]; then
                spawner_config=$(<"$temp_spawner_file")
                rm -f "$temp_spawner_file"
            fi
        fi
    fi
    
    
    # 设置模板变量环境变量供Python渲染器使用
    export AUTH_CONFIG="$auth_config"
    export SPAWNER_CONFIG="$spawner_config"
    export SHARED_STORAGE_CONFIG="$shared_storage_config"
    export ADDITIONAL_CONFIG="$additional_config"
    
    # 渲染主配置文件
    if [[ -f "$template_dir/jupyterhub_config.py.tpl" ]]; then
        python3 "$SCRIPT_DIR/scripts/template_renderer.py" "$template_dir/jupyterhub_config.py.tpl" "$output_dir/jupyterhub_config_generated.py"
    fi
    
    # 生成不同环境的配置文件
    ENVIRONMENT="development" AUTH_TYPE="local" python3 "$SCRIPT_DIR/scripts/template_renderer.py" "$template_dir/jupyterhub_config.py.tpl" "$output_dir/jupyterhub_config_development_generated.py"
    ENVIRONMENT="production" AUTH_TYPE="backend" USE_CUSTOM_AUTH="true" JUPYTERHUB_SPAWNER="kubernetes" python3 "$SCRIPT_DIR/scripts/template_renderer.py" "$template_dir/jupyterhub_config.py.tpl" "$output_dir/jupyterhub_config_production_generated.py"
    
    print_success "✓ JupyterHub 模板渲染完成"
    echo
}

# 设置JupyterHub特定变量
setup_jupyterhub_variables() {
    # 从环境变量或.env文件中读取JupyterHub配置
    ENVIRONMENT="${ENVIRONMENT:-${ENV_ENVIRONMENT:-development}}"
    AUTH_TYPE="${AUTH_TYPE:-${ENV_AUTH_TYPE:-local}}"
    JUPYTERHUB_HUB_PORT="${JUPYTERHUB_HUB_PORT:-${ENV_JUPYTERHUB_HUB_PORT:-8081}}"
    JUPYTERHUB_BASE_URL="${JUPYTERHUB_BASE_URL:-${ENV_JUPYTERHUB_BASE_URL:-/jupyter/}}"
    JUPYTERHUB_HUB_CONNECT_HOST="${JUPYTERHUB_HUB_CONNECT_HOST:-${ENV_JUPYTERHUB_HUB_CONNECT_HOST:-jupyterhub}}"
    JUPYTERHUB_PUBLIC_URL="${JUPYTERHUB_PUBLIC_URL:-${ENV_JUPYTERHUB_PUBLIC_URL:-http://localhost:8080/jupyter/}}"
    CONFIGPROXY_AUTH_TOKEN="${CONFIGPROXY_AUTH_TOKEN:-${ENV_CONFIGPROXY_AUTH_TOKEN:-}}"
    JUPYTERHUB_DB_URL="${JUPYTERHUB_DB_URL:-${ENV_JUPYTERHUB_DB_URL:-sqlite:///jupyterhub.sqlite}}"
    JUPYTERHUB_LOG_LEVEL="${JUPYTERHUB_LOG_LEVEL:-${ENV_JUPYTERHUB_LOG_LEVEL:-INFO}}"
    SESSION_TIMEOUT_DAYS="${SESSION_TIMEOUT_DAYS:-${ENV_SESSION_TIMEOUT_DAYS:-7}}"
    SINGLEUSER_IMAGE="${SINGLEUSER_IMAGE:-${ENV_SINGLEUSER_IMAGE:-ai-infra-singleuser:latest}}"
    DOCKER_NETWORK="${DOCKER_NETWORK:-${ENV_DOCKER_NETWORK:-ai-infra-matrix_default}}"
    JUPYTERHUB_MEM_LIMIT="${JUPYTERHUB_MEM_LIMIT:-${ENV_JUPYTERHUB_MEM_LIMIT:-2G}}"
    JUPYTERHUB_CPU_LIMIT="${JUPYTERHUB_CPU_LIMIT:-${ENV_JUPYTERHUB_CPU_LIMIT:-1.0}}"
    JUPYTERHUB_MEM_GUARANTEE="${JUPYTERHUB_MEM_GUARANTEE:-${ENV_JUPYTERHUB_MEM_GUARANTEE:-1G}}"
    JUPYTERHUB_CPU_GUARANTEE="${JUPYTERHUB_CPU_GUARANTEE:-${ENV_JUPYTERHUB_CPU_GUARANTEE:-0.5}}"
    USER_STORAGE_CAPACITY="${USER_STORAGE_CAPACITY:-${ENV_USER_STORAGE_CAPACITY:-10Gi}}"
    JUPYTERHUB_STORAGE_CLASS="${JUPYTERHUB_STORAGE_CLASS:-${ENV_JUPYTERHUB_STORAGE_CLASS:-default}}"
    SHARED_STORAGE_PATH="${SHARED_STORAGE_PATH:-${ENV_SHARED_STORAGE_PATH:-/srv/shared-notebooks}}"
    AI_INFRA_BACKEND_URL="${AI_INFRA_BACKEND_URL:-${ENV_AI_INFRA_BACKEND_URL:-http://backend:8082}}"
    KUBERNETES_NAMESPACE="${KUBERNETES_NAMESPACE:-${ENV_KUBERNETES_NAMESPACE:-ai-infra-users}}"
    KUBERNETES_SERVICE_ACCOUNT="${KUBERNETES_SERVICE_ACCOUNT:-${ENV_KUBERNETES_SERVICE_ACCOUNT:-ai-infra-matrix-jupyterhub}}"
    JUPYTERHUB_START_TIMEOUT="${JUPYTERHUB_START_TIMEOUT:-${ENV_JUPYTERHUB_START_TIMEOUT:-300}}"
    JUPYTERHUB_HTTP_TIMEOUT="${JUPYTERHUB_HTTP_TIMEOUT:-${ENV_JUPYTERHUB_HTTP_TIMEOUT:-30}}"
    JWT_SECRET="${JWT_SECRET:-${ENV_JWT_SECRET:-}}"
    JUPYTERHUB_AUTO_LOGIN="${JUPYTERHUB_AUTO_LOGIN:-${ENV_JUPYTERHUB_AUTO_LOGIN:-False}}"
    AUTH_REFRESH_AGE="${AUTH_REFRESH_AGE:-${ENV_AUTH_REFRESH_AGE:-3600}}"
    ADMIN_USERS="${ADMIN_USERS:-${ENV_ADMIN_USERS:-'admin'}}"
    USE_CUSTOM_AUTH="${USE_CUSTOM_AUTH:-${ENV_USE_CUSTOM_AUTH:-false}}"
    JUPYTERHUB_SPAWNER="${JUPYTERHUB_SPAWNER:-${ENV_JUPYTERHUB_SPAWNER:-docker}}"
}

# ==========================================
# ==========================================
# 随机密码生成函数
# ==========================================

# 生成安全的随机密码
generate_random_password() {
    local length="${1:-24}"  # 默认长度24
    local password_type="${2:-standard}"  # standard, hex, alphanumeric
    
    case "$password_type" in
        "hex")
            # 64位十六进制密钥 (用于JupyterHub等需要特定长度的密钥)
            if [[ "$length" == "64" ]]; then
                openssl rand -hex 32
            else
                openssl rand -hex "$((length/2))"
            fi
            ;;
        "alphanumeric")
            # 字母数字组合，避免特殊字符
            LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom | head -c "$length"
            ;;
        "standard"|*)
            # 标准密码：字母、数字、部分安全特殊字符
            LC_ALL=C tr -dc 'A-Za-z0-9._-' < /dev/urandom | head -c "$length"
            ;;
    esac
}

# 生产环境强密码生成器 (集成自 scripts/generate-prod-passwords.sh)
generate_production_passwords() {
    local env_file="${1:-.env.prod}"
    local force="${2:-false}"
    
    print_info "======================================================================"
    print_info "🔧 AI Infrastructure Matrix 生产环境密码生成器"
    print_info "======================================================================"
    print_warning "⚠️  此脚本将生成新的系统服务密码"
    print_warning "⚠️  默认管理员账户 (admin/admin123) 不会被此脚本修改"
    print_warning "⚠️  请在系统部署后通过Web界面修改管理员密码"
    print_info "======================================================================"
    
    # 如果目标环境文件不存在，从 .env.example 复制
    if [[ ! -f "$env_file" ]]; then
        if [[ -f ".env.example" ]]; then
            print_info "环境文件不存在，从 .env.example 创建: $env_file"
            cp ".env.example" "$env_file"
            print_success "✓ 已从 .env.example 创建环境文件: $env_file"
        else
            print_error "环境文件不存在: $env_file"
            print_error "且模板文件 .env.example 也不存在"
            return 1
        fi
    fi
    
    # 创建备份
    local backup_file="${env_file}.backup.$(date +%Y%m%d_%H%M%S)"
    print_info "创建备份: $backup_file"
    cp "$env_file" "$backup_file"
    
    print_info "生成新的强密码..."
    
    # 生成新密码 (使用openssl更安全，确保没有换行符)
    local postgres_password=$(openssl rand -base64 32 | tr -d "=+/\n" | cut -c1-24)
    local redis_password=$(openssl rand -base64 32 | tr -d "=+/\n" | cut -c1-24)
    local jwt_secret=$(openssl rand -base64 64 | tr -d "=+/\n" | cut -c1-48)
    local configproxy_token=$(openssl rand -base64 64 | tr -d "=+/\n" | cut -c1-48)
    local jupyterhub_crypt_key=$(openssl rand -hex 32)
    local minio_access_key=$(openssl rand -base64 32 | tr -d "=+/\n" | cut -c1-20)
    local minio_secret_key=$(openssl rand -base64 64 | tr -d "=+/\n" | cut -c1-40)
    local gitea_admin_password=$(openssl rand -base64 32 | tr -d "=+/\n" | cut -c1-24)
    local gitea_db_passwd=$(openssl rand -base64 32 | tr -d "=+/\n" | cut -c1-24)
    local ldap_admin_password=$(openssl rand -base64 32 | tr -d "=+/\n" | cut -c1-24)
    local ldap_config_password=$(openssl rand -base64 32 | tr -d "=+/\n" | cut -c1-24)
    
    # 使用awk进行安全的替换（避免sed特殊字符问题）
    # 创建临时文件
    local temp_file="${env_file}.updating"
    
    # 使用awk替换，更安全地处理特殊字符
    awk -v pg_pass="$postgres_password" \
        -v redis_pass="$redis_password" \
        -v jwt_sec="$jwt_secret" \
        -v config_token="$configproxy_token" \
        -v hub_key="$jupyterhub_crypt_key" \
        -v minio_access="$minio_access_key" \
        -v minio_secret="$minio_secret_key" \
        -v gitea_admin="$gitea_admin_password" \
        -v gitea_db="$gitea_db_passwd" \
        -v ldap_admin="$ldap_admin_password" \
        -v ldap_config="$ldap_config_password" \
        '
        /^POSTGRES_PASSWORD=/ { print "POSTGRES_PASSWORD=" pg_pass; next }
        /^REDIS_PASSWORD=/ { print "REDIS_PASSWORD=" redis_pass; next }
        /^JWT_SECRET=/ { print "JWT_SECRET=" jwt_sec; next }
        /^CONFIGPROXY_AUTH_TOKEN=/ { print "CONFIGPROXY_AUTH_TOKEN=" config_token; next }
        /^JUPYTERHUB_CRYPT_KEY=/ { print "JUPYTERHUB_CRYPT_KEY=" hub_key; next }
        /^MINIO_ACCESS_KEY=/ { print "MINIO_ACCESS_KEY=" minio_access; next }
        /^MINIO_SECRET_KEY=/ { print "MINIO_SECRET_KEY=" minio_secret; next }
        /^GITEA_ADMIN_PASSWORD=/ { print "GITEA_ADMIN_PASSWORD=" gitea_admin; next }
        /^GITEA_DB_PASSWD=/ { print "GITEA_DB_PASSWD=" gitea_db; next }
        /^LDAP_ADMIN_PASSWORD=/ { print "LDAP_ADMIN_PASSWORD=" ldap_admin; next }
        /^LDAP_CONFIG_PASSWORD=/ { print "LDAP_CONFIG_PASSWORD=" ldap_config; next }
        { print }
        ' "$env_file" > "$temp_file"
    
    # 替换原文件
    mv "$temp_file" "$env_file"
    
    print_success "已生成并应用新的强密码"
    
    print_info "======================================================================"
    print_warning "🔑 重要！默认管理员账户信息："
    echo
    print_success "  用户名: admin"
    print_error "  初始密码: admin123"
    echo
    print_warning "⚠️  请在首次登录后立即更改管理员密码！"
    print_warning "⚠️  管理员密码未通过此脚本更改，需要在系统内修改！"
    print_info "======================================================================"
    
    print_info "系统服务密码信息:"
    echo "POSTGRES_PASSWORD: $postgres_password"
    echo "REDIS_PASSWORD: $redis_password"
    echo "JWT_SECRET: $jwt_secret"
    echo "CONFIGPROXY_AUTH_TOKEN: $configproxy_token"
    echo "JUPYTERHUB_CRYPT_KEY: $jupyterhub_crypt_key"
    echo "MINIO_ACCESS_KEY: $minio_access_key"
    echo "MINIO_SECRET_KEY: $minio_secret_key"
    echo "GITEA_ADMIN_PASSWORD: $gitea_admin_password"
    echo "GITEA_DB_PASSWD: $gitea_db_passwd"
    echo "LDAP_ADMIN_PASSWORD: $ldap_admin_password"
    echo "LDAP_CONFIG_PASSWORD: $ldap_config_password"
    
    print_warning "请妥善保存这些密码信息！"
    print_info "原配置文件已备份至: $backup_file"
    
    return 0
}

# 替换环境文件中的模板密码
replace_template_passwords() {
    local template_file="$1"
    local target_file="$2"
    local force="${3:-false}"
    
    if [[ ! -f "$template_file" ]]; then
        print_error "模板文件不存在: $template_file"
        return 1
    fi
    
    if [[ -f "$target_file" ]] && [[ "$force" != "true" ]]; then
        print_warning "目标文件已存在: $target_file"
        print_info "如需强制覆盖，请使用 --force 参数"
        return 1
    fi
    
    print_info "正在从模板生成环境文件: $target_file"
    
    # 复制模板文件
    cp "$template_file" "$target_file"
    
    # 生成所有需要的密码
    local postgres_password=$(generate_random_password 24 "alphanumeric")
    local redis_password=$(generate_random_password 24 "alphanumeric")
    local jwt_secret=$(generate_random_password 48 "standard")
    local configproxy_token=$(generate_random_password 48 "standard")
    local jupyterhub_crypt_key=$(generate_random_password 64 "hex")
    local minio_access_key=$(generate_random_password 20 "alphanumeric")
    local minio_secret_key=$(generate_random_password 40 "standard")
    local gitea_admin_password=$(generate_random_password 24 "alphanumeric")
    local gitea_db_password=$(generate_random_password 24 "alphanumeric")
    local ldap_admin_password=$(generate_random_password 24 "alphanumeric")
    local ldap_config_password=$(generate_random_password 24 "alphanumeric")
    
    # 替换模板中的密码占位符
    sed -i.bak \
        -e "s/TEMPLATE_POSTGRES_PASSWORD/$postgres_password/g" \
        -e "s/TEMPLATE_REDIS_PASSWORD/$redis_password/g" \
        -e "s/TEMPLATE_JWT_SECRET/$jwt_secret/g" \
        -e "s/TEMPLATE_CONFIGPROXY_AUTH_TOKEN/$configproxy_token/g" \
        -e "s/TEMPLATE_JUPYTERHUB_CRYPT_KEY/$jupyterhub_crypt_key/g" \
        -e "s/TEMPLATE_MINIO_ACCESS_KEY/$minio_access_key/g" \
        -e "s/TEMPLATE_MINIO_SECRET_KEY/$minio_secret_key/g" \
        -e "s/TEMPLATE_GITEA_ADMIN_PASSWORD/$gitea_admin_password/g" \
        -e "s/TEMPLATE_GITEA_DB_PASSWD/$gitea_db_password/g" \
        -e "s/TEMPLATE_LDAP_ADMIN_PASSWORD/$ldap_admin_password/g" \
        -e "s/TEMPLATE_LDAP_CONFIG_PASSWORD/$ldap_config_password/g" \
        "$target_file"
    
    # 处理环境变量展开的URL (替换 ${VARIABLE} 形式)
    # 读取当前文件内容并替换变量引用
    local temp_content=$(cat "$target_file")
    
    # 处理DATABASE_URL
    temp_content=$(echo "$temp_content" | sed "s|\\\${POSTGRES_USER}|postgres|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${POSTGRES_PASSWORD}|$postgres_password|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${POSTGRES_HOST}|postgres|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${POSTGRES_PORT}|5432|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${POSTGRES_DB}|aiinfra|g")
    
    # 处理REDIS_URL
    temp_content=$(echo "$temp_content" | sed "s|\\\${REDIS_PASSWORD}|$redis_password|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${REDIS_HOST}|redis|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${REDIS_PORT}|6379|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${REDIS_DB}|0|g")
    
    # 处理其他服务URL
    temp_content=$(echo "$temp_content" | sed "s|\\\${BACKEND_HOST}|backend|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${BACKEND_PORT}|8082|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${FRONTEND_HOST}|frontend|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${FRONTEND_PORT}|80|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${JUPYTERHUB_HOST}|jupyterhub|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${JUPYTERHUB_PORT}|8000|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${GITEA_HOST}|gitea|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${GITEA_PORT}|3000|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${GITEA_INTERNAL_URL}|http://gitea:3000|g")
    
    # 处理外部访问变量 (动态检测)
    load_environment_variables
    temp_content=$(echo "$temp_content" | sed "s|\\\${EXTERNAL_HOST}|$EXTERNAL_HOST|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${EXTERNAL_PORT}|$EXTERNAL_PORT|g")
    temp_content=$(echo "$temp_content" | sed "s|\\\${EXTERNAL_SCHEME}|$EXTERNAL_SCHEME|g")
    
    # 写回文件
    echo "$temp_content" > "$target_file"
    
    # 删除备份文件
    rm -f "${target_file}.bak"
    
    print_success "✓ 生成环境文件完成: $target_file"
    print_info "所有密码已自动生成，请妥善保管！"
    
    return 0
}

# ==========================================
# 环境变量管理函数
# ==========================================

# 生成环境文件从模板
create_env_from_template() {
    local env_type="${1:-dev}"  # dev 或 prod
    local force="${2:-false}"
    
    print_info "正在创建环境配置文件..."
    
    case "$env_type" in
        "prod"|"production")
            local template_file=".env.prod.example"
            local target_file=".env.prod"
            ;;
        "dev"|"development"|*)
            local template_file=".env.example"
            local target_file=".env"
            ;;
    esac
    
    # 对于生产环境，使用密码替换功能
    if [[ "$env_type" == "prod" ]] || [[ "$env_type" == "production" ]]; then
        if replace_template_passwords "$template_file" "$target_file" "$force"; then
            # 检查并创建backend目录的环境文件
            if [[ ! -f "src/backend/.env" ]] && [[ -f "src/backend/.env.example" ]]; then
                cp "src/backend/.env.example" "src/backend/.env"
                print_success "✓ 创建后端环境文件: src/backend/.env"
            fi
            
            # 应用生产环境特殊配置
            print_info "应用生产环境配置..."
            sed -i.bak 's/DEBUG_MODE=true/DEBUG_MODE=false/g' "$target_file" 2>/dev/null || true
            sed -i.bak 's/LOG_LEVEL=debug/LOG_LEVEL=info/g' "$target_file" 2>/dev/null || true
            sed -i.bak 's/BUILD_ENV=development/BUILD_ENV=production/g' "$target_file" 2>/dev/null || true
            rm -f "${target_file}.bak"
            
            return 0
        else
            return 1
        fi
    fi
    
    # 检查模板文件是否存在
    if [[ ! -f "$template_file" ]]; then
        print_error "模板文件不存在: $template_file"
        return 1
    fi
    
    # 检查目标文件是否已存在
    if [[ -f "$target_file" ]] && [[ "$force" != "true" ]]; then
        print_warning "环境文件已存在: $target_file"
        print_info "如需强制覆盖，请使用 --force 参数"
        return 0
    fi
    
    # 复制模板文件并进行变量替换 (开发环境)
    if cp "$template_file" "$target_file"; then
        print_success "✓ 创建环境文件: $target_file (从 $template_file)"
        
        # 加载环境变量
        load_environment_variables
        
        # 进行基本的外部变量替换
        local temp_content=$(cat "$target_file")
        temp_content=$(echo "$temp_content" | sed "s|\\\${EXTERNAL_HOST}|$EXTERNAL_HOST|g")
        temp_content=$(echo "$temp_content" | sed "s|\\\${EXTERNAL_PORT}|$EXTERNAL_PORT|g")
        temp_content=$(echo "$temp_content" | sed "s|\\\${EXTERNAL_SCHEME}|$EXTERNAL_SCHEME|g")
        echo "$temp_content" > "$target_file"
        
        print_success "✓ 应用外部变量替换: EXTERNAL_HOST=$EXTERNAL_HOST, EXTERNAL_PORT=$EXTERNAL_PORT"
        
        # 检查并创建backend目录的环境文件
        if [[ ! -f "src/backend/.env" ]] && [[ -f "src/backend/.env.example" ]]; then
            cp "src/backend/.env.example" "src/backend/.env"
            print_success "✓ 创建后端环境文件: src/backend/.env"
        fi
        
        return 0
    else
        print_error "创建环境文件失败"
        return 1
    fi
}

# 自动生成环境文件（用于自动修复）
auto_generate_env_files() {
    local force="${1:-false}"
    
    print_info "=========================================="
    print_info "自动生成环境配置文件"
    print_info "=========================================="
    
    local generated_count=0
    local failed_count=0
    
    # 生成主环境文件
    if [[ ! -f ".env" ]] || [[ "$force" == "true" ]]; then
        print_info "生成主环境文件 .env..."
        if create_env_from_template "dev" "$force"; then
            ((generated_count++))
        else
            ((failed_count++))
        fi
    else
        print_info "主环境文件 .env 已存在，跳过"
    fi
    
    # 生成生产环境文件
    if [[ ! -f ".env.prod" ]] || [[ "$force" == "true" ]]; then
        print_info "生成生产环境文件 .env.prod..."
        if create_env_from_template "prod" "$force"; then
            ((generated_count++))
        else
            ((failed_count++))
        fi
    else
        print_info "生产环境文件 .env.prod 已存在，跳过"
    fi
    
    # 检查并修复PostgreSQL密码一致性
    print_info "检查PostgreSQL密码配置一致性..."
    local env_postgres_password=$(grep -E '^POSTGRES_PASSWORD=' .env 2>/dev/null | cut -d'=' -f2 | tr -d '"' | tr -d "'")
    local env_postgres_user=$(grep -E '^POSTGRES_USER=' .env 2>/dev/null | cut -d'=' -f2 | tr -d '"' | tr -d "'")
    
    if [[ -n "$env_postgres_password" ]] && [[ -n "$env_postgres_user" ]]; then
        print_success "✓ PostgreSQL配置: 用户=$env_postgres_user, 密码=<已设置>"
    else
        print_warning "PostgreSQL密码配置可能有问题，请检查.env文件"
    fi
    
    # 检查Redis密码配置
    local redis_password=$(grep -E '^REDIS_PASSWORD=' .env 2>/dev/null | cut -d'=' -f2 | tr -d '"' | tr -d "'")
    if [[ -n "$redis_password" ]]; then
        print_success "✓ Redis密码配置正常"
    else
        print_warning "Redis密码配置可能有问题，请检查.env文件"
    fi
    
    print_info "=========================================="
    if [[ $failed_count -eq 0 ]]; then
        print_success "环境文件生成完成: $generated_count 个文件"
        print_info "建议重启所有服务以应用新配置"
        return 0
    else
        print_error "环境文件生成失败: $failed_count 个文件"
        return 1
    fi
}

# 检测并确定唯一的环境文件
detect_env_file() {
    local env_file=""
    
    # 优先级检查：.env.prod > .env > .env.example
    if [[ -f ".env.prod" ]]; then
        env_file=".env.prod"
        echo "使用生产环境配置: $env_file" >&2
    elif [[ -f ".env" ]]; then
        env_file=".env"
        echo "使用开发环境配置: $env_file" >&2
    elif [[ -f ".env.example" ]]; then
        echo "未找到环境配置文件，从模板创建..." >&2
        if create_env_from_template "dev"; then
            env_file=".env"
            echo "✓ 从.env.example创建了.env文件" >&2
        else
            echo "错误: 创建环境文件失败" >&2
            return 1
        fi
    else
        echo "错误: 未找到任何环境配置文件（.env.prod, .env, .env.example）" >&2
        return 1
    fi
    
    echo "$env_file"
    return 0
}

# 验证环境文件有效性
validate_env_file() {
    local env_file="$1"
    
    if [[ ! -f "$env_file" ]]; then
        echo "错误: 环境文件不存在: $env_file" >&2
        return 1
    fi
    
    # 检查关键变量是否存在
    local required_vars=("IMAGE_TAG" "COMPOSE_PROJECT_NAME")
    local missing_vars=()
    
    for var in "${required_vars[@]}"; do
        if ! grep -q "^${var}=" "$env_file" 2>/dev/null; then
            missing_vars+=("$var")
        fi
    done
    
    if [[ ${#missing_vars[@]} -gt 0 ]]; then
        echo "警告: 环境文件 $env_file 缺少必要变量: ${missing_vars[*]}" >&2
        echo "建议检查并补充这些变量" >&2
    fi
    
    return 0
}

# 更新外部主机配置
update_external_host_config() {
    local host_ip="${1:-auto}"
    
    print_info "=========================================="
    print_info "🌐 更新外部主机配置"
    print_info "=========================================="
    
    # 自动检测外部主机IP
    if [[ "$host_ip" == "auto" ]]; then
        print_info "自动检测外部主机IP..."
        
        # 尝试检测外部可访问的IP地址
        local detected_ip=""
        
        # 方法1: 通过默认路由检测
        if command -v ip >/dev/null 2>&1; then
            detected_ip=$(ip route get 8.8.8.8 2>/dev/null | sed -n 's/.*src \([0-9.]*\).*/\1/p' | head -1)
        fi
        
        # 方法2: 通过ifconfig检测（macOS兼容）
        if [[ -z "$detected_ip" ]] && command -v ifconfig >/dev/null 2>&1; then
            detected_ip=$(ifconfig | grep -E 'inet\s+([0-9]{1,3}\.){3}[0-9]{1,3}' | grep -v '127.0.0.1' | awk '{print $2}' | head -1)
        fi
        
        # 方法3: 通过route命令（macOS兼容）
        if [[ -z "$detected_ip" ]] && command -v route >/dev/null 2>&1; then
            detected_ip=$(route get default 2>/dev/null | grep interface | awk '{print $2}' | xargs ifconfig 2>/dev/null | grep -E 'inet\s+([0-9]{1,3}\.){3}[0-9]{1,3}' | grep -v '127.0.0.1' | awk '{print $2}' | head -1)
        fi
        
        # 备用方案: 使用localhost
        if [[ -z "$detected_ip" ]]; then
            detected_ip="localhost"
            print_warning "无法自动检测外部IP，使用默认值: localhost"
        else
            print_success "检测到外部IP: $detected_ip"
        fi
        
        host_ip="$detected_ip"
    fi
    
    print_info "目标主机IP: $host_ip"
    
    # 确定要更新的环境文件
    local env_files=()
    [[ -f ".env" ]] && env_files+=(".env")
    [[ -f ".env.prod" ]] && env_files+=(".env.prod")
    [[ -f ".env.example" ]] && env_files+=(".env.example")
    
    if [[ ${#env_files[@]} -eq 0 ]]; then
        print_error "未找到任何环境配置文件"
        return 1
    fi
    
    print_info "将更新以下环境文件: ${env_files[*]}"
    
    local success_count=0
    local total_count=${#env_files[@]}
    
    for env_file in "${env_files[@]}"; do
        print_info "→ 更新文件: $env_file"
        
        # 备份原文件
        local backup_file="${env_file}.backup.$(date +%Y%m%d_%H%M%S)"
        if cp "$env_file" "$backup_file"; then
            print_info "  ✓ 创建备份: $backup_file"
        else
            print_warning "  ⚠ 无法创建备份文件"
        fi
        
        # 更新EXTERNAL_HOST
        if grep -q "^EXTERNAL_HOST=" "$env_file"; then
            # 更新现有的EXTERNAL_HOST
            if sed -i.tmp "s/^EXTERNAL_HOST=.*/EXTERNAL_HOST=$host_ip/" "$env_file" && rm -f "${env_file}.tmp"; then
                print_success "  ✓ 更新EXTERNAL_HOST=$host_ip"
            else
                print_error "  ✗ 更新EXTERNAL_HOST失败"
                continue
            fi
        else
            # 添加新的EXTERNAL_HOST
            echo "EXTERNAL_HOST=$host_ip" >> "$env_file"
            print_success "  ✓ 添加EXTERNAL_HOST=$host_ip"
        fi
        
        # 确保其他动态配置变量存在
        local dynamic_vars=(
            "EXTERNAL_PORT=8080"
            "EXTERNAL_SCHEME=http"
        )
        
        for var_line in "${dynamic_vars[@]}"; do
            local var_name=$(echo "$var_line" | cut -d'=' -f1)
            if ! grep -q "^${var_name}=" "$env_file"; then
                echo "$var_line" >> "$env_file"
                print_success "  ✓ 添加默认配置: $var_line"
            fi
        done
        
        ((success_count++))
    done
    
    print_info "=========================================="
    if [[ $success_count -eq $total_count ]]; then
        print_success "✅ 外部主机配置更新完成: $success_count/$total_count 文件"
        print_info "新的外部主机: $host_ip"
        print_info "建议重新生成nginx配置并重启服务："
        print_info "  $0 build nginx"
        print_info "  docker compose restart nginx"
    else
        print_error "❌ 外部主机配置更新失败: $success_count/$total_count 文件"
        return 1
    fi
    
    return 0
}

# 更新外部端口配置
update_external_port_config() {
    local port="${1:-8080}"
    
    print_info "=========================================="
    print_info "🔌 更新外部端口配置"
    print_info "=========================================="
    print_info "目标端口: $port"
    
    # 验证端口号格式
    if ! [[ "$port" =~ ^[0-9]+$ ]] || [[ "$port" -lt 1 ]] || [[ "$port" -gt 65535 ]]; then
        print_error "无效的端口号: $port (必须是1-65535之间的数字)"
        return 1
    fi
    
    # 确定要更新的环境文件
    local env_files=()
    [[ -f ".env" ]] && env_files+=(".env")
    [[ -f ".env.prod" ]] && env_files+=(".env.prod")
    [[ -f ".env.example" ]] && env_files+=(".env.example")
    
    if [[ ${#env_files[@]} -eq 0 ]]; then
        print_error "未找到任何环境配置文件"
        return 1
    fi
    
    print_info "将更新以下环境文件: ${env_files[*]}"
    
    local success_count=0
    local total_count=${#env_files[@]}
    
    for env_file in "${env_files[@]}"; do
        print_info "→ 更新文件: $env_file"
        
        # 备份原文件
        local backup_file="${env_file}.backup.$(date +%Y%m%d_%H%M%S)"
        if cp "$env_file" "$backup_file"; then
            print_info "  ✓ 创建备份: $backup_file"
        else
            print_warning "  ⚠ 无法创建备份文件"
        fi
        
        # 更新EXTERNAL_PORT
        if grep -q "^EXTERNAL_PORT=" "$env_file"; then
            # 更新现有的EXTERNAL_PORT
            if sed -i.tmp "s/^EXTERNAL_PORT=.*/EXTERNAL_PORT=$port/" "$env_file" && rm -f "${env_file}.tmp"; then
                print_success "  ✓ 更新EXTERNAL_PORT=$port"
            else
                print_error "  ✗ 更新EXTERNAL_PORT失败"
                continue
            fi
        else
            # 添加新的EXTERNAL_PORT
            echo "EXTERNAL_PORT=$port" >> "$env_file"
            print_success "  ✓ 添加EXTERNAL_PORT=$port"
        fi
        
        # 计算并显示相关端口
        local jupyter_port=$((port + 8))
        local gitea_port=$((port - 5070))
        local debug_port=$((port - 79))
        
        print_info "  → 计算的端口配置:"
        print_info "    主入口端口: $port"
        print_info "    JupyterHub端口: $jupyter_port"
        print_info "    Gitea端口: $gitea_port"
        print_info "    调试端口: $debug_port"
        
        ((success_count++))
    done
    
    print_info "=========================================="
    if [[ $success_count -eq $total_count ]]; then
        print_success "✅ 外部端口配置更新完成: $success_count/$total_count 文件"
        print_info "新的外部端口: $port"
        print_info "端口映射:"
        print_info "  • 主入口: $port"
        print_info "  • JupyterHub: $((port + 8))"
        print_info "  • Gitea: $((port - 5070))"
        print_info "  • 调试端口: $((port - 79))"
        print_info ""
        print_info "建议重新生成配置并重启服务："
        print_info "  $0 build nginx --force"
        print_info "  docker compose down && docker compose up -d"
    else
        print_error "❌ 外部端口配置更新失败: $success_count/$total_count 文件"
        return 1
    fi
    
    return 0
}

# 一键更新端口并重新部署
quick_deploy_with_port() {
    local port="${1:-8080}"
    local host="${2:-auto}"
    
    print_info "=========================================="
    print_info "🚀 一键更新端口并重新部署"
    print_info "=========================================="
    print_info "目标端口: $port"
    print_info "目标主机: $host"
    
    # 步骤1: 更新外部主机配置
    print_info "步骤1: 更新外部主机配置..."
    if ! update_external_host_config "$host"; then
        print_error "外部主机配置更新失败"
        return 1
    fi
    
    # 步骤2: 更新端口配置
    print_info "步骤2: 更新端口配置..."
    if ! update_external_port_config "$port"; then
        print_error "端口配置更新失败"
        return 1
    fi
    
    # 步骤3: 重新构建nginx
    print_info "步骤3: 重新构建nginx配置..."
    FORCE_REBUILD=true
    if ! build_service "nginx" "$DEFAULT_IMAGE_TAG"; then
        print_error "nginx构建失败"
        return 1
    fi
    
    # 步骤4: 重启nginx服务
    print_info "步骤4: 重启nginx服务..."
    if docker compose restart nginx; then
        print_success "✓ nginx服务重启成功"
    else
        print_error "nginx服务重启失败"
        return 1
    fi
    
    # 显示服务信息
    print_info "=========================================="
    print_success "🎉 一键部署完成！"
    print_info "服务访问地址:"
    local current_host=$(grep "^EXTERNAL_HOST=" .env.example | cut -d'=' -f2)
    local current_scheme=$(grep "^EXTERNAL_SCHEME=" .env.example | cut -d'=' -f2)
    print_info "  • 主入口: ${current_scheme}://${current_host}:${port}"
    print_info "  • JupyterHub: ${current_scheme}://${current_host}:$((port + 8))/jupyter/"
    print_info "  • Gitea: ${current_scheme}://${current_host}:$((port - 5070))/gitea/"
    print_info "  • 调试接口: ${current_scheme}://${current_host}:$((port - 79))/debug/"
    print_info "=========================================="
    
    return 0
}

# 对比两个环境文件的差异
compare_env_files() {
    local env1="$1"
    local env2="$2"
    
    if [[ ! -f "$env1" ]] || [[ ! -f "$env2" ]]; then
        print_error "环境文件不存在: $env1 或 $env2"
        return 1
    fi
    
    print_info "对比环境文件: $env1 vs $env2"
    
    # 提取所有变量名（排除注释和空行）
    local vars1=$(grep -E "^[A-Z_][A-Z0-9_]*=" "$env1" | cut -d'=' -f1 | sort)
    local vars2=$(grep -E "^[A-Z_][A-Z0-9_]*=" "$env2" | cut -d'=' -f1 | sort)
    
    # 找出差异变量
    local only_in_1=$(comm -23 <(echo "$vars1") <(echo "$vars2"))
    local only_in_2=$(comm -13 <(echo "$vars1") <(echo "$vars2"))
    local common_vars=$(comm -12 <(echo "$vars1") <(echo "$vars2"))
    
    if [[ -n "$only_in_1" ]]; then
        print_warning "仅在 $env1 中存在的变量:"
        echo "$only_in_1" | while read var; do
            echo "  - $var"
        done
    fi
    
    if [[ -n "$only_in_2" ]]; then
        print_warning "仅在 $env2 中存在的变量:"
        echo "$only_in_2" | while read var; do
            echo "  - $var"
        done
    fi
    
    # 检查共同变量的值差异
    local diff_count=0
    echo "$common_vars" | while read var; do
        if [[ -n "$var" ]]; then
            local val1=$(grep "^${var}=" "$env1" | cut -d'=' -f2- | tr -d '"'"'"'"')
            local val2=$(grep "^${var}=" "$env2" | cut -d'=' -f2- | tr -d '"'"'"'"')
            if [[ "$val1" != "$val2" ]]; then
                if [[ $diff_count -eq 0 ]]; then
                    print_info "值不同的变量:"
                fi
                echo "  $var:"
                echo "    $env1: $val1"
                echo "    $env2: $val2"
                ((diff_count++))
            fi
        fi
    done
    
    if [[ -z "$only_in_1" ]] && [[ -z "$only_in_2" ]] && [[ $diff_count -eq 0 ]]; then
        print_success "✓ 环境文件配置一致"
    fi
    
    return 0
}

# 校验环境文件的完整性和一致性
validate_env_consistency() {
    local dev_env=".env"
    local prod_env=".env.prod"
    local example_env=".env.example"
    
    print_info "=========================================="
    print_info "环境文件一致性校验"
    print_info "=========================================="
    
    # 检查文件存在性
    local files_exist=()
    local files_missing=()
    
    for env_file in "$dev_env" "$prod_env" "$example_env"; do
        if [[ -f "$env_file" ]]; then
            files_exist+=("$env_file")
        else
            files_missing+=("$env_file")
        fi
    done
    
    print_info "存在的环境文件: ${files_exist[*]}"
    if [[ ${#files_missing[@]} -gt 0 ]]; then
        print_warning "缺失的环境文件: ${files_missing[*]}"
    fi
    
    # 如果开发环境和生产环境文件都存在，进行对比
    if [[ -f "$dev_env" ]] && [[ -f "$prod_env" ]]; then
        echo
        compare_env_files "$dev_env" "$prod_env"
    fi
    
    # 校验必要的变量
    echo
    for env_file in "${files_exist[@]}"; do
        print_info "校验 $env_file..."
        validate_env_file "$env_file"
    done
    
    return 0
}

# ==========================================
# Docker Compose 版本检测和适配
# ==========================================

# 检测Docker Compose版本并返回最佳命令
detect_compose_command() {
    local compose_cmd=""
    local compose_version=""
    
    # 优先使用docker compose (v2)
    if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
        compose_cmd="docker compose"
        compose_version=$(docker compose version --short 2>/dev/null || docker compose version | grep -o 'v[0-9.]*' | head -1)
        echo "$compose_cmd"
        return 0
    fi
    
    # 回退到docker-compose (v1)
    if command -v docker-compose >/dev/null 2>&1; then
        compose_cmd="docker-compose"
        compose_version=$(docker-compose version --short 2>/dev/null || docker-compose version | grep -o '[0-9.]*' | head -1)
        echo "$compose_cmd"
        return 0
    fi
    
    return 1
}

# 检查Docker Compose版本兼容性
check_compose_compatibility() {
    local compose_cmd
    compose_cmd=$(detect_compose_command)
    local exit_code=$?
    
    if [ $exit_code -ne 0 ]; then
        print_error "未找到Docker Compose命令"
        print_info "请安装Docker Compose v2.0+:"
        print_info "  https://docs.docker.com/compose/install/"
        return 1
    fi
    
    local version=""
    if [[ "$compose_cmd" == "docker compose" ]]; then
        version=$(docker compose version --short 2>/dev/null || docker compose version | grep -o 'v[0-9.]*' | head -1 | sed 's/v//')
        print_info "检测到Docker Compose v2: $version"
        
        # 清理版本号，移除v前缀和额外信息
        local clean_version=$(echo "$version" | sed 's/^v//' | sed 's/-.*$//')
        
        # 检查是否为v2.39.2或更高版本
        if command -v python3 >/dev/null 2>&1; then
            local is_compatible=$(python3 -c "
import sys
from packaging import version
try:
    current = version.parse('$clean_version')
    required = version.parse('2.39.2')
    print('true' if current >= required else 'false')
except Exception as e:
    print('true')  # 默认兼容
" 2>/dev/null || echo "true")
            
            if [[ "$is_compatible" == "true" ]]; then
                print_success "✓ Docker Compose版本兼容 (v$clean_version >= v2.39.2)"
            else
                print_warning "⚠ Docker Compose版本较旧 (v$clean_version < v2.39.2)，建议升级"
                print_info "当前版本应该仍可工作，但建议升级以获得最佳体验"
            fi
        else
            print_info "✓ 使用Docker Compose v2: $clean_version"
        fi
    else
        version=$(docker-compose version --short 2>/dev/null || docker-compose version | grep -o '[0-9.]*' | head -1)
        print_warning "检测到Docker Compose v1: $version"
        print_info "建议升级到Docker Compose v2以获得更好的性能和功能"
    fi
    
    echo "$compose_cmd"
    return 0
}

# 验证compose文件格式
validate_compose_file() {
    local file="$1"
    local compose_cmd="$2"
    
    if [[ ! -f "$file" ]]; then
        print_error "Compose文件不存在: $file"
        return 1
    fi
    
    print_info "验证compose文件: $file"
    
    if ! $compose_cmd -f "$file" config >/dev/null 2>&1; then
        print_error "Compose文件验证失败: $file"
        print_info "详细错误信息："
        $compose_cmd -f "$file" config 2>&1 | head -10
        return 1
    fi
    
    print_success "✓ Compose文件验证通过: $file"
    return 0
}

# 获取私有镜像名称（支持Harbor格式：registry/project）
get_private_image_name() {
    local original_image="$1"
    local registry="$2"
    
    if [[ -z "$registry" ]]; then
        echo "$original_image"
        return 0
    fi
    
    # 检查original_image是否已经包含了registry信息
    if [[ "$original_image" == "$registry"/* ]]; then
        echo "$original_image"
        return 0
    fi
    
    # 处理不同类型的registry格式
    local registry_base=""
    local project_path=""
    local is_harbor_style=false
    
    if [[ "$registry" == *"/"* ]]; then
        # Harbor格式：registry.xxx.com/project
        is_harbor_style=true
        registry_base="${registry%%/*}"  # 获取 registry.xxx.com
        project_path="${registry#*/}"    # 获取 project
    else
        # 传统格式：registry.xxx.com
        registry_base="$registry"
    fi
    
    # 处理镜像名称
    local image_name_tag=""
    
    if [[ "$original_image" == *"/"* ]]; then
        # 包含组织/用户名的镜像
        if [[ "$original_image" == *"."*"/"* ]]; then
            # 第三方仓库镜像 (如 quay.io/minio/minio:latest)
            image_name_tag="${original_image#*/}"  # 移除域名部分
        else
            # Docker Hub 组织镜像 (如 osixia/openldap:stable)
            image_name_tag="$original_image"
        fi
    else
        # 简单镜像名 (如 redis:7-alpine, postgres:15-alpine)
        image_name_tag="$original_image"
    fi
    
    # 构建最终镜像路径
    if [[ "$is_harbor_style" == "true" ]]; then
        # Harbor模式：registry.xxx.com/project/image:tag
        echo "${registry}/${image_name_tag}"
    else
        # 传统模式：registry.xxx.com/image:tag
        echo "${registry}/${image_name_tag}"
    fi
}

# 根据镜像映射配置获取私有镜像名称和版本
# 支持latest标签到git版本的映射
get_mapped_private_image() {
    local original_image="$1"
    local registry="$2"
    local target_tag="${3:-v0.3.5}"  # 默认目标git版本
    local mapping_file="$SCRIPT_DIR/config/image-mapping.conf"
    
    if [[ -z "$registry" ]]; then
        echo "$original_image"
        return 0
    fi
    
    # 标准化镜像名称（移除tag用于匹配）
    local image_base=""
    local original_tag=""
    
    if [[ "$original_image" == *":"* ]]; then
        image_base="${original_image%%:*}"
        original_tag="${original_image##*:}"
    else
        image_base="$original_image"
        original_tag="latest"
    fi
    
    # 提取原始镜像的简短名称（不含namespace）
    local simple_name=""
    if [[ "$image_base" == *"/"* ]]; then
        # 处理带namespace的镜像，如 tecnativa/tcp-proxy -> tcp-proxy
        simple_name="${image_base##*/}"
    else
        # 直接使用镜像名，如 postgres -> postgres
        simple_name="$image_base"
    fi
    
    # 如果映射文件存在，尝试读取映射配置
    local mapped_project=""
    local mapped_version=""
    local found_mapping=false
    
    if [[ -f "$mapping_file" ]]; then
        while IFS='|' read -r pattern project version special; do
            # 跳过注释和空行
            [[ "$pattern" =~ ^[[:space:]]*# ]] && continue
            [[ -z "$pattern" ]] && continue
            
            # 检查是否匹配（支持精确匹配和基础名匹配）
            if [[ "$original_image" == "$pattern" ]] || 
               [[ "$image_base" == "$pattern" ]] ||
               [[ "$image_base:$original_tag" == "$pattern" ]]; then
                mapped_project="$project"
                mapped_version="$version"
                found_mapping=true
                break
            fi
        done < "$mapping_file"
    fi
    
    local final_version=""
    if [[ "$found_mapping" == "true" ]]; then
        # 处理特殊变量替换
        if [[ "$mapped_version" == *'${TARGET_TAG}'* ]]; then
            # 项目镜像，使用传入的target_tag
            final_version="${mapped_version//\$\{TARGET_TAG\}/$target_tag}"
        elif [[ "$mapped_version" == *'${IMAGE_TAG}'* ]]; then
            # 兼容旧格式
            final_version="${mapped_version//\$\{IMAGE_TAG\}/$target_tag}"
        else
            # 使用配置文件中的版本
            final_version="$mapped_version"
        fi
    else
        # 未找到映射，强制使用目标标签
        final_version="$target_tag"
    fi
    
    # 构建最终镜像名：registry/simple_name:final_version
    local final_image="${registry}/${simple_name}:${final_version}"
    
    echo "$final_image"
}

# 检查 Dockerfile 是否存在
check_dockerfile() {
    local service="$1"
    local service_path=$(get_service_path "$service")
    
    if [[ -z "$service_path" ]]; then
        print_error "未知服务: $service"
        return 1
    fi
    
    local dockerfile_path="$SCRIPT_DIR/$service_path/Dockerfile"
    
    if [[ ! -f "$dockerfile_path" ]]; then
        print_error "Dockerfile 不存在: $dockerfile_path"
        return 1
    fi
    return 0
}

# 构建单个服务镜像
build_service() {
    local service="$1"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    local registry="${3:-}"
    
    local service_path=$(get_service_path "$service")
    if [[ -z "$service_path" ]]; then
        print_error "未知服务: $service"
        print_info "可用服务: $SRC_SERVICES"
        return 1
    fi
    
    if ! check_dockerfile "$service"; then
        return 1
    fi
    
    local dockerfile_path="$SCRIPT_DIR/$service_path/Dockerfile"
    local base_image="ai-infra-${service}:${tag}"
    
    # 确定目标镜像名
    local target_image="$base_image"
    if [[ -n "$registry" ]]; then
        target_image=$(get_private_image_name "$base_image" "$registry")
    fi
    
    print_info "构建服务: $service"
    print_info "  Dockerfile: $service_path/Dockerfile"
    print_info "  目标镜像: $target_image"
    
    # 检查镜像是否已存在
    if [[ "$FORCE_REBUILD" == "false" ]] && docker image inspect "$target_image" >/dev/null 2>&1; then
        print_success "  ✓ 镜像已存在，跳过构建: $target_image"
        
        # 如果指定了registry，确保本地别名也存在
        if [[ -n "$registry" ]] && [[ "$target_image" != "$base_image" ]]; then
            if ! docker image inspect "$base_image" >/dev/null 2>&1; then
                if docker tag "$target_image" "$base_image"; then
                    print_info "  ✓ 创建本地别名: $base_image"
                fi
            fi
        fi
        
        return 0
    fi
    
    # 构建镜像
    print_info "  → 正在构建镜像..."
    
    # 特殊处理nginx和jupyterhub的构建上下文
    local build_context
    if [[ "$service" == "nginx" ]]; then
        # nginx构建前先渲染模板
        print_info "  → nginx构建前渲染配置模板..."
        render_nginx_templates
        build_context="$SCRIPT_DIR"  # 使用项目根目录作为构建上下文
    elif [[ "$service" == "jupyterhub" ]]; then
        # jupyterhub构建前先渲染配置模板
        print_info "  → jupyterhub构建前渲染配置模板..."
        render_jupyterhub_templates
        build_context="$SCRIPT_DIR/$service_path"
    else
        build_context="$SCRIPT_DIR/$service_path"
    fi
    
    local dockerfile_name="Dockerfile"
    
    # 统一处理：所有服务都使用各自的src子目录作为构建上下文
    local target_arg=""
    if [[ "$service" == "backend-init" ]]; then
        target_arg="--target backend-init"
    elif [[ "$service" == "backend" ]]; then
        target_arg="--target backend"
    fi
    
    # 使用各自的src子目录作为构建上下文
    if docker build -f "$dockerfile_path" $target_arg -t "$target_image" "$build_context"; then
        print_success "✓ 构建成功: $target_image"
        
        # 如果指定了registry，同时创建本地别名
        if [[ -n "$registry" ]] && [[ "$target_image" != "$base_image" ]]; then
            if docker tag "$target_image" "$base_image"; then
                print_info "  ✓ 本地别名: $base_image"
            fi
        fi
        
        return 0
    else
        print_error "✗ 构建失败: $target_image"
        return 1
    fi
}

# 构建所有服务镜像
build_all_services() {
    local tag="${1:-$DEFAULT_IMAGE_TAG}"
    local registry="${2:-}"
    
    print_info "=========================================="
    print_info "构建所有 AI-Infra 服务镜像"
    print_info "=========================================="
    print_info "镜像标签: $tag"
    if [[ -n "$registry" ]]; then
        print_info "目标仓库: $registry"
    else
        print_info "目标仓库: 本地构建"
    fi
    echo
    
    local success_count=0
    local total_count=0
    local failed_services=()
    
    # 获取所有服务（包括原扩展组件）
    local all_services="$SRC_SERVICES"
    
    # 计算服务总数
    for service in $all_services; do
        total_count=$((total_count + 1))
    done
    
    # 构建所有服务
    for service in $all_services; do
        print_info "构建服务: $service"
        if build_service "$service" "$tag" "$registry"; then
            success_count=$((success_count + 1))
        else
            failed_services+=("$service")
        fi
        echo
    done
    
    print_info "=========================================="
    print_success "构建完成: $success_count/$total_count 成功"
    
    if [[ ${#failed_services[@]} -gt 0 ]]; then
        print_warning "失败的服务: ${failed_services[*]}"
        return 1
    else
        print_success "🎉 所有服务构建成功！"
        return 0
    fi
}

# 推送单个服务镜像
push_service() {
    local service="$1"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    local registry="$3"
    
    if [[ -z "$registry" ]]; then
        print_error "推送操作需要指定 registry"
        return 1
    fi
    
    local base_image="ai-infra-${service}:${tag}"
    local target_image=$(get_private_image_name "$base_image" "$registry")
    
    print_info "推送服务: $service"
    print_info "  原始镜像: $base_image"
    print_info "  目标镜像: $target_image"
    print_info "  Registry: $registry"
    
    # 检查镜像是否存在
    if ! docker image inspect "$base_image" >/dev/null 2>&1; then
        print_warning "本地镜像不存在: $base_image"
        print_info "尝试构建镜像..."
        if ! build_service "$service" "$tag" "$registry"; then
            print_error "构建失败，无法推送"
            return 1
        fi
    else
        print_success "✓ 本地镜像存在: $base_image"
    fi
    
    # 如果需要标记为目标镜像
    if [[ "$base_image" != "$target_image" ]]; then
        print_info "标记镜像: $base_image -> $target_image"
        if ! docker tag "$base_image" "$target_image"; then
            print_error "镜像标记失败"
            return 1
        fi
    fi
    
    # 推送镜像
    print_info "推送镜像: $target_image"
    if docker push "$target_image"; then
        print_success "✓ 推送成功: $target_image"
        return 0
    else
        print_error "✗ 推送失败: $target_image"
        return 1
    fi
}

# 推送所有服务镜像
push_all_services() {
    local tag="${1:-$DEFAULT_IMAGE_TAG}"
    local registry="$2"
    
    if [[ -z "$registry" ]]; then
        print_error "推送操作需要指定 registry"
        print_info "用法: $0 push-all <registry> [tag]"
        return 1
    fi
    
    print_info "=========================================="
    print_info "推送所有 AI-Infra 服务镜像"
    print_info "=========================================="
    print_info "目标仓库: $registry"
    print_info "镜像标签: $tag"
    echo
    
    local success_count=0
    local total_count=0
    local failed_services=()
    
    # 计算服务总数
    for service in $SRC_SERVICES; do
        total_count=$((total_count + 1))
    done
    
    for service in $SRC_SERVICES; do
        if push_service "$service" "$tag" "$registry"; then
            success_count=$((success_count + 1))
        else
            failed_services+=("$service")
        fi
        echo
    done
    
    print_info "=========================================="
    print_success "推送完成: $success_count/$total_count 成功"
    
    if [[ ${#failed_services[@]} -gt 0 ]]; then
        print_warning "失败的服务: ${failed_services[*]}"
        return 1
    else
        print_success "🚀 所有服务推送成功！"
        return 0
    fi
}

# 一键构建并推送
build_and_push_all() {
    local tag="${1:-$DEFAULT_IMAGE_TAG}"
    local registry="$2"
    
    if [[ -z "$registry" ]]; then
        print_error "一键构建推送需要指定 registry"
        print_info "用法: $0 build-push <registry> [tag]"
        return 1
    fi
    
    print_info "=========================================="
    print_info "一键构建并推送所有服务"
    print_info "=========================================="
    print_info "目标仓库: $registry"
    print_info "镜像标签: $tag"
    echo
    
    # 第一阶段：构建所有镜像
    print_info "🔨 第一阶段：构建所有镜像..."
    if ! build_all_services "$tag" "$registry"; then
        print_error "构建阶段失败，停止执行"
        return 1
    fi
    
    echo
    print_info "🚀 第二阶段：推送所有镜像..."
    if ! push_all_services "$tag" "$registry"; then
        print_error "推送阶段失败"
        return 1
    fi
    
    print_success "🎉 一键构建推送完成！"
}

# 拉取并标记依赖镜像
pull_and_tag_dependencies() {
    local registry="$1"
    local tag="${2:-latest}"
    
    if [[ -z "$registry" ]]; then
        print_error "需要指定 registry"
        print_info "用法: $0 deps-pull <registry> [tag]"
        return 1
    fi
    
    print_info "=========================================="
    print_info "拉取并标记依赖镜像到 $registry"
    print_info "=========================================="
    print_info "目标镜像标签: $tag (所有依赖镜像将统一使用此版本标签)"
    
    # 动态收集依赖镜像
    local dependency_images
    dependency_images=$(collect_dependency_images)
    print_info "收集到依赖镜像: $dependency_images"
    echo
    
    local success_count=0
    local total_count=0
    local failed_deps=()
    
    for dep_image in $dependency_images; do
        total_count=$((total_count + 1))
        print_info "处理依赖镜像: $dep_image"
        
        # 使用新的映射机制生成目标镜像名
        local target_image
        target_image=$(get_mapped_private_image "$dep_image" "$registry" "$tag")
        
        # 检查目标镜像是否已存在
        if [[ "$FORCE_REBUILD" == "false" ]] && docker image inspect "$target_image" >/dev/null 2>&1; then
            print_success "  ✓ 镜像已存在，跳过: $target_image"
            success_count=$((success_count + 1))
            continue
        fi
        
        # 检查原始镜像是否已存在本地
        if docker image inspect "$dep_image" >/dev/null 2>&1; then
            print_success "  ✓ 本地镜像已存在: $dep_image"
        else
            # 拉取原始镜像
            print_info "  → 正在拉取镜像: $dep_image"
            if ! docker pull "$dep_image"; then
                print_error "  ✗ 拉取失败: $dep_image"
                failed_deps+=("$dep_image")
                continue
            fi
            print_success "  ✓ 拉取成功: $dep_image"
        fi
        
        # 标记镜像
        if docker tag "$dep_image" "$target_image"; then
            print_success "  ✓ 标记成功: $target_image"
            success_count=$((success_count + 1))
        else
            print_error "  ✗ 标记失败: $target_image"
            failed_deps+=("$dep_image")
        fi
        echo
    done
    
    print_info "=========================================="
    print_success "依赖镜像处理完成: $success_count/$total_count 成功"
    
    if [[ ${#failed_deps[@]} -gt 0 ]]; then
        print_warning "失败的依赖镜像: ${failed_deps[*]}"
        return 1
    else
        print_success "🎉 所有依赖镜像处理成功！"
        return 0
    fi
}

# 推送依赖镜像
push_dependencies() {
    local registry="$1"
    local tag="${2:-latest}"
    
    if [[ -z "$registry" ]]; then
        print_error "需要指定 registry"
        print_info "用法: $0 deps-push <registry> [tag]"
        return 1
    fi
    
    print_info "=========================================="
    print_info "推送依赖镜像到 $registry"
    print_info "=========================================="
    print_info "目标镜像标签: $tag (所有依赖镜像将统一使用此版本标签)"
    
    # 动态收集依赖镜像
    local dependency_images
    dependency_images=$(collect_dependency_images)
    print_info "收集到依赖镜像: $dependency_images"
    echo
    
    local success_count=0
    local total_count=0
    local failed_deps=()
    
    for dep_image in $dependency_images; do
        total_count=$((total_count + 1))
        
        # 使用新的映射机制生成目标镜像名
        local target_image
        target_image=$(get_mapped_private_image "$dep_image" "$registry" "$tag")
        
        print_info "推送依赖镜像: $target_image"
        
        if docker push "$target_image"; then
            print_success "  ✓ 推送成功: $target_image"
            success_count=$((success_count + 1))
        else
            print_error "  ✗ 推送失败: $target_image"
            failed_deps+=("$target_image")
        fi
        echo
    done
    
    print_info "=========================================="
    print_success "依赖镜像推送完成: $success_count/$total_count 成功"
    
    if [[ ${#failed_deps[@]} -gt 0 ]]; then
        print_warning "失败的依赖镜像: ${failed_deps[*]}"
        return 1
    else
        print_success "🎉 所有依赖镜像推送成功！"
        return 0
    fi
}

# ==========================================
# 生产环境依赖镜像处理功能
# ==========================================

# 拉取并标记生产环境依赖镜像（排除测试工具）
pull_and_tag_production_dependencies() {
    local registry="$1"
    local tag="${2:-latest}"
    
    if [[ -z "$registry" ]]; then
        print_error "需要指定 registry"
        return 1
    fi
    
    print_info "=========================================="
    print_info "拉取并标记生产环境依赖镜像到 $registry"
    print_info "=========================================="
    print_info "目标镜像标签: $tag (所有生产环境依赖镜像将统一使用此版本标签)"
    
    # 使用生产环境依赖镜像列表
    local dependency_images
    dependency_images=$(get_production_dependencies | tr '\n' ' ')
    print_info "收集到生产环境依赖镜像: $dependency_images"
    echo
    
    local success_count=0
    local total_count=0
    local failed_deps=()
    
    for dep_image in $dependency_images; do
        if [[ -z "$dep_image" ]]; then
            continue
        fi
        
        ((total_count++))
        
        # 获取目标镜像名称
        local target_image
        target_image=$(get_mapped_private_image "$dep_image" "$registry" "$tag")
        
        # 检查镜像是否已存在
        if docker image inspect "$target_image" >/dev/null 2>&1; then
            print_success "  ✓ 镜像已存在，跳过: $target_image"
            ((success_count++))
            continue
        fi
        
        print_info "处理生产环境依赖镜像: $dep_image"
        
        # 拉取原始镜像
        if ! docker pull "$dep_image"; then
            print_error "  ✗ 拉取失败: $dep_image"
            failed_deps+=("$dep_image")
            continue
        fi
        
        # 标记为目标镜像
        if ! docker tag "$dep_image" "$target_image"; then
            print_error "  ✗ 标记失败: $dep_image -> $target_image"
            failed_deps+=("$dep_image")
            continue
        fi
        
        print_success "  ✓ 处理成功: $dep_image -> $target_image"
        ((success_count++))
    done
    you y
    print_info "=========================================="
    print_success "生产环境依赖镜像处理完成: $success_count/$total_count 成功"
    
    if [[ ${#failed_deps[@]} -gt 0 ]]; then
        print_warning "失败的依赖镜像: ${failed_deps[*]}"
        return 1
    else
        print_success "🎉 所有生产环境依赖镜像处理成功！"
        return 0
    fi
}

# 推送生产环境依赖镜像
push_production_dependencies() {
    local registry="$1"
    local tag="${2:-latest}"
    
    if [[ -z "$registry" ]]; then
        print_error "需要指定 registry"
        return 1
    fi
    
    print_info "=========================================="
    print_info "推送生产环境依赖镜像到 $registry"
    print_info "=========================================="
    print_info "源镜像标签: $tag (如果为latest则会映射到v0.3.5)"
    
    # 使用生产环境依赖镜像列表
    local dependency_images
    dependency_images=$(get_production_dependencies | tr '\n' ' ')
    print_info "收集到生产环境依赖镜像: $dependency_images"
    echo
    
    local success_count=0
    local total_count=0
    local failed_deps=()
    
    for dep_image in $dependency_images; do
        if [[ -z "$dep_image" ]]; then
            continue
        fi
        
        ((total_count++))
        
        # 获取目标镜像名称
        local target_image
        target_image=$(get_mapped_private_image "$dep_image" "$registry" "$tag")
        
        print_info "推送生产环境依赖镜像: $target_image"
        
        if docker push "$target_image"; then
            print_success "  ✓ 推送成功: $target_image"
            ((success_count++))
        else
            print_error "  ✗ 推送失败: $target_image"
            failed_deps+=("$target_image")
        fi
    done
    
    print_info "=========================================="
    print_success "生产环境依赖镜像推送完成: $success_count/$total_count 成功"
    
    if [[ ${#failed_deps[@]} -gt 0 ]]; then
        print_warning "失败的依赖镜像: ${failed_deps[*]}"
        return 1
    else
        print_success "🎉 所有生产环境依赖镜像推送成功！"
        return 0
    fi
}

# 推送构建依赖镜像（仅包含构建时需要的镜像）
push_build_dependencies() {
    local registry="$1"
    local tag="${2:-latest}"
    
    if [[ -z "$registry" ]]; then
        print_error "需要指定 registry"
        print_info "用法: $0 build-deps-push <registry> [tag]"
        return 1
    fi
    
    print_info "=========================================="
    print_info "推送构建依赖镜像到 $registry"
    print_info "=========================================="
    print_info "目标镜像标签: $tag"
    
    # 定义构建依赖镜像
    local build_dependencies=(
        "node:22-alpine"
        "nginx:stable-alpine-perl"
        "golang:1.25-alpine"
        "python:3.13-alpine"
        "gitea/gitea:1.24.5"
        "jupyter/base-notebook:latest"
    )
    
    local success_count=0
    local total_count=${#build_dependencies[@]}
    local failed_deps=()
    
    for dep_image in "${build_dependencies[@]}"; do
        # 使用新的映射机制生成目标镜像名
        local target_image
        target_image=$(get_mapped_private_image "$dep_image" "$registry" "$tag")
        
        print_info "推送构建依赖镜像: $target_image"
        
        if docker push "$target_image"; then
            print_success "  ✓ 推送成功: $target_image"
            success_count=$((success_count + 1))
        else
            print_error "  ✗ 推送失败: $target_image"
            failed_deps+=("$target_image")
        fi
        echo
    done
    
    print_info "=========================================="
    print_success "构建依赖镜像推送完成: $success_count/$total_count 成功"
    
    if [[ ${#failed_deps[@]} -gt 0 ]]; then
        print_warning "失败的构建依赖镜像: ${failed_deps[*]}"
        return 1
    else
        print_success "🎉 所有构建依赖镜像推送成功！"
        return 0
    fi
}

# ==========================================
# AI Harbor 镜像拉取管理
# ==========================================

# 从 AI Harbor 拉取所有服务镜像
pull_aiharbor_services() {
    local registry="${1:-aiharbor.msxf.local/aihpc}"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    
    print_info "=========================================="
    print_info "🚢 从 AI Harbor 拉取服务镜像"
    print_info "=========================================="
    print_info "Harbor地址: $registry"
    print_info "镜像标签: $tag"
    echo
    
    local services=("backend" "frontend" "jupyterhub" "nginx" "saltstack" "singleuser" "gitea" "backend-init")
    local success_count=0
    local total_count=${#services[@]}
    local failed_services=()
    
    for service in "${services[@]}"; do
        local harbor_image="${registry}/ai-infra-${service}:${tag}"
        local local_image="ai-infra-${service}:${tag}"
        
        print_info "→ 拉取服务: $service"
        print_info "  Harbor镜像: $harbor_image"
        print_info "  本地标签: $local_image"
        
        # 尝试拉取镜像
        if docker pull "$harbor_image"; then
            print_success "  ✓ 拉取成功: $harbor_image"
            
            # 标记为本地镜像名
            if docker tag "$harbor_image" "$local_image"; then
                print_success "  ✓ 标记为本地镜像: $local_image"
                success_count=$((success_count + 1))
            else
                print_error "  ✗ 标记失败: $local_image"
                failed_services+=("$service")
            fi
        else
            print_error "  ✗ 拉取失败: $harbor_image"
            failed_services+=("$service")
        fi
        echo
    done
    
    print_info "=========================================="
    print_success "拉取完成: $success_count/$total_count 成功"
    
    if [[ ${#failed_services[@]} -gt 0 ]]; then
        print_warning "失败的服务: ${failed_services[*]}"
        print_info "可以尝试以下操作:"
        print_info "1. 检查 Harbor 仓库访问权限"
        print_info "2. 验证镜像标签是否存在: $tag"
        print_info "3. 确认网络连接正常"
        return 1
    else
        print_success "🚀 所有AI-Infra服务镜像拉取成功！"
        print_info "现在可以使用本地镜像启动服务："
        print_info "  docker compose -f docker-compose.yml.example up -d"
        return 0
    fi
}

# 从 AI Harbor 拉取依赖镜像  
pull_aiharbor_dependencies() {
    local registry="${1:-aiharbor.msxf.local/aihpc}"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    
    print_info "=========================================="
    print_info "🚢 从 AI Harbor 拉取依赖镜像"
    print_info "=========================================="
    print_info "Harbor地址: $registry"
    print_info "镜像标签: $tag"
    echo
    
    # 从配置文件或预定义列表收集依赖镜像
    local dependency_images=$(get_all_dependencies | tr '\n' ' ')
    if [[ -z "$dependency_images" ]]; then
        dependency_images="postgres:15-alpine redis:7-alpine nginx:1.27-alpine tecnativa/tcp-proxy minio/minio:latest osixia/openldap:stable osixia/phpldapadmin:stable redislabs/redisinsight:latest node:22-alpine nginx:stable-alpine-perl golang:1.25-alpine python:3.13-alpine gitea/gitea:1.24.5 jupyter/base-notebook:latest"
    fi
    
    print_info "依赖镜像列表: $dependency_images"
    echo
    
    local success_count=0
    local total_count=0
    local failed_deps=()
    
    for dep_image in $dependency_images; do
        if [[ -z "$dep_image" ]]; then
            continue
        fi
        
        ((total_count++))
        
        # 获取映射后的Harbor镜像名称
        local harbor_image
        harbor_image=$(get_mapped_private_image "$dep_image" "$registry" "$tag")
        
        print_info "→ 拉取依赖: $(basename "$dep_image")"
        print_info "  Harbor镜像: $harbor_image"
        print_info "  原始镜像: $dep_image"
        
        # 尝试拉取Harbor镜像
        if docker pull "$harbor_image"; then
            print_success "  ✓ 拉取成功: $harbor_image"
            
            # 标记为原始镜像名
            if docker tag "$harbor_image" "$dep_image"; then
                print_success "  ✓ 标记为原始镜像: $dep_image"
                success_count=$((success_count + 1))
            else
                print_error "  ✗ 标记失败: $dep_image"
                failed_deps+=("$dep_image")
            fi
        else
            print_warning "  ! Harbor拉取失败，尝试官方源: $dep_image"
            # 回退到官方镜像拉取
            if docker pull "$dep_image"; then
                print_success "  ✓ 从官方源拉取成功: $dep_image"
                success_count=$((success_count + 1))
            else
                print_error "  ✗ 所有源都拉取失败: $dep_image"
                failed_deps+=("$dep_image")
            fi
        fi
        echo
    done
    
    print_info "=========================================="
    print_success "依赖镜像拉取完成: $success_count/$total_count 成功"
    
    if [[ ${#failed_deps[@]} -gt 0 ]]; then
        print_warning "失败的依赖镜像: ${failed_deps[*]}"
        return 1
    else
        print_success "🚀 所有依赖镜像拉取成功！"
        return 0
    fi
}

# 从 AI Harbor 拉取所有镜像（服务+依赖）
pull_aiharbor_all() {
    local registry="${1:-aiharbor.msxf.local/aihpc}"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    
    print_info "=========================================="
    print_info "🚢 从 AI Harbor 拉取所有镜像"
    print_info "=========================================="
    print_info "Harbor地址: $registry"
    print_info "镜像标签: $tag"
    echo
    
    local overall_success=true
    
    # 先拉取依赖镜像
    print_info "步骤 1/2: 拉取依赖镜像..."
    if ! pull_aiharbor_dependencies "$registry" "$tag"; then
        print_warning "部分依赖镜像拉取失败，但继续拉取服务镜像..."
        overall_success=false
    fi
    
    echo
    print_info "步骤 2/2: 拉取服务镜像..."
    if ! pull_aiharbor_services "$registry" "$tag"; then
        print_error "服务镜像拉取失败"
        overall_success=false
    fi
    
    echo
    print_info "=========================================="
    if [[ "$overall_success" == "true" ]]; then
        print_success "🎉 所有镜像拉取完成！"
        print_info ""
        print_info "接下来可以："
        print_info "1. 启动服务: docker compose -f docker-compose.yml.example up -d"
        print_info "2. 查看状态: ./build.sh prod-status"
        return 0
    else
        print_warning "⚠️  部分镜像拉取失败，请检查错误信息"
        print_info "建议操作："
        print_info "1. 检查Harbor访问权限和网络连接"
        print_info "2. 验证镜像标签 $tag 是否存在"
        print_info "3. 重新运行失败的拉取命令"
        return 1
    fi
}

# ==========================================
# 双环境部署支持功能
# ==========================================

# 创建生产环境配置文件 (.env.prod)
create_production_env() {
    local mode="${1:-production}"  # production 或 intranet
    local registry="${2:-aiharbor.msxf.local/aihpc}"
    local tag="${3:-$DEFAULT_IMAGE_TAG}"
    
    local env_file=".env.prod"
    local template_file=".env.example"
    
    print_info "创建生产环境配置文件: $env_file"
    print_info "模式: $mode"
    print_info "镜像仓库: $registry"
    print_info "镜像标签: $tag"
    
    # 检查模板文件
    if [[ ! -f "$template_file" ]]; then
        print_error "模板文件不存在: $template_file"
        return 1
    fi
    
    # 复制模板文件
    cp "$template_file" "$env_file"
    
    # 根据模式配置不同的参数
    case "$mode" in
        "build"|"builder")
            # 构建环境配置
            sed -i.bak \
                -e "s|^IMAGE_TAG=.*|IMAGE_TAG=$tag|" \
                -e "s|^PRIVATE_REGISTRY=.*|PRIVATE_REGISTRY=$registry|" \
                -e "s|^BUILD_ENV=.*|BUILD_ENV=production|" \
                -e "s|^DEBUG_MODE=.*|DEBUG_MODE=false|" \
                -e "s|^LOG_LEVEL=.*|LOG_LEVEL=info|" \
                -e "s|^ENV_FILE=.*|ENV_FILE=.env.prod|" \
                -e "s|^DOMAIN=.*|DOMAIN=ai-infra.local|" \
                "$env_file"
            ;;
        "intranet"|"runtime")
            # 内网运行环境配置
            sed -i.bak \
                -e "s|^IMAGE_TAG=.*|IMAGE_TAG=$tag|" \
                -e "s|^PRIVATE_REGISTRY=.*|PRIVATE_REGISTRY=$registry|" \
                -e "s|^BUILD_ENV=.*|BUILD_ENV=production|" \
                -e "s|^DEBUG_MODE=.*|DEBUG_MODE=false|" \
                -e "s|^LOG_LEVEL=.*|LOG_LEVEL=info|" \
                -e "s|^ENV_FILE=.*|ENV_FILE=.env.prod|" \
                -e "s|^DOMAIN=.*|DOMAIN=ai-infra.local|" \
                "$env_file"
            ;;
        *)
            print_error "不支持的模式: $mode"
            print_info "支持的模式: build, intranet"
            return 1
            ;;
    esac
    
    # 删除备份文件
    rm -f "${env_file}.bak"
    
    print_success "✓ 已创建生产环境配置: $env_file"
    print_info "请根据实际环境调整配置文件中的参数"
    
    return 0
}

# 构建环境模式 - 构建并推送所有镜像
build_environment_deploy() {
    local registry="${1:-aiharbor.msxf.local/aihpc}"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    
    print_info "=========================================="
    print_info "构建环境部署模式"
    print_info "=========================================="
    print_info "镜像仓库: $registry"
    print_info "镜像标签: $tag"
    print_info "目标: 构建所有镜像并推送到仓库"
    echo
    
    # 1. 创建生产环境配置
    if ! create_production_env "build" "$registry" "$tag"; then
        return 1
    fi
    
    # 2. 构建所有服务镜像
    print_info "构建所有服务镜像..."
    if ! build_all_services "$tag" "$registry"; then
        print_error "服务镜像构建失败"
        return 1
    fi
    
    # 3. 推送所有镜像到仓库
    print_info "推送所有镜像到仓库..."
    if ! push_all_services "$tag" "$registry"; then
        print_error "镜像推送失败"
        return 1
    fi
    
    # 4. 推送依赖镜像
    print_info "推送依赖镜像..."
    if ! push_all_dependencies "$tag" "$registry"; then
        print_error "依赖镜像推送失败"
        return 1
    fi
    
    # 5. 生成生产环境docker-compose配置
    print_info "复制生产环境配置文件..."
    if [[ -f "docker-compose.yml.example" ]]; then
        cp docker-compose.yml.example docker-compose.yml
        print_success "✓ 已复制 docker-compose.yml.example 到 docker-compose.yml"
    else
        print_error "docker-compose.yml.example 文件不存在"
        return 1
    fi
    
    print_success "✅ 构建环境部署完成！"
    print_info "生成的文件:"
    print_info "  - .env.prod (生产环境配置)"
    print_info "  - docker-compose.prod.yml (生产环境编排文件)"
    print_info ""
    print_info "已推送到仓库的镜像:"
    print_info "  - 所有服务镜像 (标签: $tag)"
    print_info "  - 所有依赖镜像"
    print_info ""
    print_info "下一步: 将以下文件复制到内网环境："
    print_info "  - .env.prod"
    print_info "  - docker-compose.prod.yml"
    print_info "  - build.sh (用于内网部署)"
    
    return 0
}

# 内网环境模式 - 拉取镜像并启动服务
intranet_environment_deploy() {
    local registry="${1:-aiharbor.msxf.local/aihpc}"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    
    print_info "=========================================="
    print_info "内网环境部署模式"
    print_info "=========================================="
    print_info "镜像仓库: $registry"
    print_info "镜像标签: $tag"
    print_info "目标: 拉取镜像并启动所有服务"
    echo
    
    # 1. 检查或创建生产环境配置
    if [[ ! -f ".env.prod" ]]; then
        print_info "创建生产环境配置..."
        if ! create_production_env "intranet" "$registry" "$tag"; then
            return 1
        fi
    else
        print_info "使用现有的生产环境配置: .env.prod"
    fi
    
    # 2. 检查或生成docker-compose.prod.yml
    if [[ ! -f "docker-compose.prod.yml" ]]; then
        print_info "复制生产环境编排文件..."
        if [[ -f "docker-compose.yml.example" ]]; then
            cp docker-compose.yml.example docker-compose.prod.yml
            print_success "✓ 已复制 docker-compose.yml.example 到 docker-compose.prod.yml"
        else
            print_error "docker-compose.yml.example 文件不存在"
            return 1
        fi
    else
        print_info "使用现有的编排文件: docker-compose.prod.yml"
    fi
    
    # 3. 启动生产环境服务
    print_info "启动生产环境服务..."
    if ! start_production "$registry" "$tag" "false"; then
        print_error "服务启动失败"
        return 1
    fi
    
    print_success "✅ 内网环境部署完成！"
    print_info "服务状态:"
    production_status
    
    return 0
}

# ==========================================
# 生产环境部署相关功能
# ==========================================

# 部署到指定HOST（动态配置域名）
deploy_to_host() {
    local host="$1"
    local registry="$2"
    local tag="${3:-$DEFAULT_IMAGE_TAG}"
    
    if [[ -z "$host" ]]; then
        print_error "必须指定HOST地址"
        return 1
    fi
    
    print_info "===========================================" 
    print_info "部署AI-Infra到指定HOST: $host"
    print_info "==========================================="
    print_info "Host: $host"
    print_info "Registry: ${registry:-'(本地镜像)'}"
    print_info "Tag: $tag"
    echo
    
    # 备份原始.env.prod文件
    if [[ -f ".env.prod" ]]; then
        cp ".env.prod" ".env.prod.backup.$(date +%Y%m%d%H%M%S)"
        print_info "已备份原始.env.prod文件"
    fi
    
    # 检测HOST格式并设置PORT
    local nginx_port="8080"
    local public_host="$host:$nginx_port"
    local public_protocol="http"
    
    if [[ "$host" =~ ^https?:// ]]; then
        print_error "HOST不应包含协议前缀，请使用纯域名或IP，如: example.com 或 192.168.1.100"
        return 1
    fi
    
    if [[ "$host" =~ :[0-9]+$ ]]; then
        public_host="$host"
        print_info "检测到HOST包含端口: $public_host"
    else
        public_host="$host:$nginx_port"
        print_info "使用默认端口: $public_host"
    fi
    
    # 临时设置环境变量（用于生成配置）
    export AI_INFRA_HOST="$host"
    
    # 更新.env.prod文件中的HOST相关配置
    print_info "更新.env.prod中的HOST配置..."
    
    # 使用sed命令更新配置
    if [[ "$OS_TYPE" == "macOS" ]]; then
        sed -i.bak "s|^DOMAIN=.*|DOMAIN=$host|g" .env.prod
        sed -i.bak "s|^PUBLIC_HOST=.*|PUBLIC_HOST=$public_host|g" .env.prod  
        sed -i.bak "s|^JUPYTERHUB_PUBLIC_HOST=.*|JUPYTERHUB_PUBLIC_HOST=$public_host|g" .env.prod
        sed -i.bak "s|^JUPYTERHUB_CORS_ORIGIN=.*|JUPYTERHUB_CORS_ORIGIN=$public_protocol://$public_host|g" .env.prod
        sed -i.bak "s|^ROOT_URL=.*|ROOT_URL=$public_protocol://$public_host/gitea/|g" .env.prod
        rm -f .env.prod.bak
    else
        sed -i "s|^DOMAIN=.*|DOMAIN=$host|g" .env.prod
        sed -i "s|^PUBLIC_HOST=.*|PUBLIC_HOST=$public_host|g" .env.prod
        sed -i "s|^JUPYTERHUB_PUBLIC_HOST=.*|JUPYTERHUB_PUBLIC_HOST=$public_host|g" .env.prod
        sed -i "s|^JUPYTERHUB_CORS_ORIGIN=.*|JUPYTERHUB_CORS_ORIGIN=$public_protocol://$public_host|g" .env.prod
        sed -i "s|^ROOT_URL=.*|ROOT_URL=$public_protocol://$public_host/gitea/|g" .env.prod
    fi
    
    print_success "✓ HOST配置更新完成"
    
    # 复制生产环境配置文件
    print_info "复制生产环境配置文件..."
    if [[ -f "docker-compose.yml.example" ]]; then
        cp docker-compose.yml.example docker-compose.yml
        print_success "✓ 已复制 docker-compose.yml.example 到 docker-compose.yml"
    else
        print_error "docker-compose.yml.example 文件不存在"
        print_error "生产环境配置生成失败"
        return 1
    fi
    
    # 启动服务（使用本地镜像模式）
    print_info "启动生产环境服务..."
    if ! start_production "$registry" "$tag" "true"; then
        print_error "生产环境启动失败"
        return 1
    fi
    
    print_success "=========================================="
    print_success "🎉 部署完成！"
    print_success "=========================================="
    print_info "访问地址:"
    print_info "  主页: $public_protocol://$public_host/"
    print_info "  JupyterHub: $public_protocol://$public_host/jupyterhub/"
    print_info "  Gitea: $public_protocol://$public_host/gitea/"
    print_info ""
    print_info "管理命令:"
    print_info "  查看状态: $0 prod-status"
    print_info "  查看日志: $0 prod-logs [service]"
    print_info "  停止服务: $0 prod-down"
    echo
    
    return 0
}


# 启动生产环境
start_production() {
    local registry="$1"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    local force_local="${3:-false}"  # 新增参数：是否强制使用本地镜像
    local compose_file="docker-compose.yml"
    
    # 自动检测外部主机地址
    if [[ -f "scripts/detect-external-host.sh" ]]; then
        print_info "自动检测外部主机地址..."
        source scripts/detect-external-host.sh
        print_info "使用检测到的主机地址: $EXTERNAL_HOST"
    fi
    
    # registry 可以为空（使用本地镜像）
    if [[ -z "$registry" ]]; then
        print_info "使用本地镜像（无 registry 前缀）"
        registry=""
    fi
    
    # 检测环境文件 - 生产环境优先使用 .env.prod
    local env_file
    if [[ -f ".env.prod" ]]; then
        env_file=".env.prod"
        print_info "使用生产环境文件: $env_file"
    else
        env_file=$(detect_env_file)
        if [[ $? -ne 0 ]]; then
            return 1
        fi
        print_warning "未找到 .env.prod，使用: $env_file"
    fi
    
    # 验证环境文件
    if ! validate_env_file "$env_file"; then
        return 1
    fi
    
    # 总是重新复制配置文件
    print_info "复制配置文件 (registry: $registry, tag: $tag)..."
    if [[ -f "docker-compose.yml.example" ]]; then
        cp docker-compose.yml.example docker-compose.yml
        print_success "✓ 已复制 docker-compose.yml.example 到 docker-compose.yml"
    else
        print_error "docker-compose.yml.example 文件不存在"
        return 1
    fi
    
    print_info "=========================================="
    print_info "启动生产环境"
    print_info "=========================================="
    print_info "配置文件: $compose_file"
    print_info "环境文件: $env_file"
    print_info "Registry: $registry"
    print_info "标签: $tag"
    if [[ "$force_local" == "true" ]]; then
        print_info "模式: 强制使用本地镜像 (跳过拉取)"
    fi
    echo
    
    # 根据 force_local 参数决定是否拉取镜像
    if [[ "$force_local" == "true" ]]; then
        print_info "跳过镜像拉取，使用本地已有镜像..."
        
        # 如果指定了registry，标记本地镜像为新的registry标签
        if [[ -n "$registry" ]]; then
            tag_local_images_for_registry "$registry" "$tag"
        fi
        
        # 检查并构建缺失的镜像（包括有build配置的服务）
        print_info "检查并构建需要的镜像..."
        if ! check_and_build_missing_images "$compose_file" "$env_file" "$registry" "$tag"; then
            print_warning "部分镜像构建失败，继续尝试启动..."
        fi
    else
        print_info "拉取所有镜像..."
        if ! ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" pull; then
            print_error "镜像拉取失败"
            return 1
        fi
    fi
    
    print_info "启动生产环境..."
    if ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" up -d; then
        print_success "✓ 生产环境启动成功"
        echo
        print_info "查看服务状态:"
        ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" ps
        return 0
    else
        print_error "✗ 生产环境启动失败"
        return 1
    fi
}

# 检查并构建缺失的镜像
# 标记本地镜像为新的registry标签
tag_local_images_for_registry() {
    local registry="$1"
    local tag="$2"
    
    print_info "标记本地镜像为新的registry标签..."
    
    # 智能查找本地镜像的函数
    find_local_image() {
        local image_name="$1"
        local target_tag="$2"
        
        # 先尝试精确匹配
        if docker image inspect "${image_name}:${target_tag}" >/dev/null 2>&1; then
            echo "${image_name}:${target_tag}"
            return 0
        fi
        
        # 如果精确匹配失败，尝试查找包含目标标签的镜像
        local found_image=$(docker images --format "table {{.Repository}}:{{.Tag}}" | grep "^${image_name}:" | grep -E "(test-)?${target_tag}$" | head -n1)
        if [[ -n "$found_image" ]]; then
            echo "$found_image"
            return 0
        fi
        
        # 如果还是找不到，查找最新的镜像
        local latest_image=$(docker images --format "table {{.Repository}}:{{.Tag}}" | grep "^${image_name}:" | grep -v "<none>" | head -n1)
        if [[ -n "$latest_image" ]]; then
            echo "$latest_image"
            return 0
        fi
        
        return 1
    }
    
    # 定义需要标记的镜像基础名称
    local ai_infra_images=(
        "ai-infra-backend"
        "ai-infra-backend-init"
        "ai-infra-frontend"
        "ai-infra-jupyterhub"
        "ai-infra-gitea"
        "ai-infra-nginx"
        "ai-infra-saltstack"
        "ai-infra-singleuser"
    )
    
    # 定义依赖镜像
    local dependency_images=(
        "postgres:15-alpine"
        "redis:7-alpine"
        "nginx:1.27-alpine"
        "tecnativa/tcp-proxy:latest"
        "minio/minio:latest"
        "osixia/openldap:stable"
        "osixia/phpldapadmin:stable"
        "redislabs/redisinsight:latest"
        "node:22-alpine"
        "nginx:stable-alpine-perl"
        "golang:1.25-alpine"
        "python:3.13-alpine"
        "gitea/gitea:1.24.5"
        "jupyter/base-notebook:latest"
    )
    
    local tagged_count=0
    local missing_count=0
    
    # 处理AI-Infra自研镜像
    for image_name in "${ai_infra_images[@]}"; do
        local target_image="${registry}/${image_name}:${tag}"
        
        # 检查目标镜像是否已存在
        if docker image inspect "$target_image" >/dev/null 2>&1; then
            print_info "  ✓ 已存在: $target_image"
            continue
        fi
        
        # 智能查找本地镜像
        local source_image=$(find_local_image "$image_name" "$tag")
        if [[ -n "$source_image" ]]; then
            # 标记镜像
            if docker tag "$source_image" "$target_image" 2>/dev/null; then
                print_success "  ✓ 已标记: $source_image -> $target_image"
                tagged_count=$((tagged_count + 1))
            else
                print_warning "  ✗ 标记失败: $source_image -> $target_image"
            fi
        else
            print_warning "  ✗ 本地未找到镜像: $image_name"
            missing_count=$((missing_count + 1))
        fi
    done
    
    # 处理依赖镜像
    for source_image in "${dependency_images[@]}"; do
        # 计算目标镜像名（移除域名前缀）
        local clean_name=$(echo "$source_image" | sed 's|^[^/]*/||' | sed 's|^[^/]*/||')
        local target_image="${registry}/${clean_name}"
        
        # 检查目标镜像是否已存在
        if docker image inspect "$target_image" >/dev/null 2>&1; then
            print_info "  ✓ 已存在: $target_image"
            continue
        fi
        
        # 检查源镜像是否存在
        if docker image inspect "$source_image" >/dev/null 2>&1; then
            # 标记镜像
            if docker tag "$source_image" "$target_image" 2>/dev/null; then
                print_success "  ✓ 已标记: $source_image -> $target_image"
                tagged_count=$((tagged_count + 1))
            else
                print_warning "  ✗ 标记失败: $source_image -> $target_image"
            fi
        else
            print_warning "  ✗ 源镜像不存在: $source_image"
            missing_count=$((missing_count + 1))
        fi
    done
    
    print_info "镜像标记完成: 成功 $tagged_count 个，缺失 $missing_count 个"
    
    return 0
}

check_and_build_missing_images() {
    local compose_file="$1"
    local env_file="$2"
    local registry="$3"
    local tag="$4"
    
    if [[ ! -f "$compose_file" ]]; then
        print_error "compose文件不存在: $compose_file"
        return 1
    fi
    
    print_info "分析compose文件中需要的镜像..."
    
    # 直接构建已知的关键服务（简化方案）
    local critical_services=("backend-init" "gitea" "singleuser-builder")
    local missing_count=0
    
    for service in "${critical_services[@]}"; do
        # 构造预期的镜像名
        local expected_image="${registry}/ai-infra-${service}:${tag}"
        
        # 检查镜像是否存在
        if ! docker image inspect "$expected_image" >/dev/null 2>&1; then
            print_info "缺失镜像: $expected_image"
            if build_service_if_missing "$service" "$compose_file" "$env_file"; then
                # 构建成功后标记镜像
                local local_image="ai-infra-${service}:${tag}"
                if docker image inspect "$local_image" >/dev/null 2>&1; then
                    docker tag "$local_image" "$expected_image"
                    print_success "✓ 已标记: $local_image -> $expected_image"
                fi
            else
                missing_count=$((missing_count + 1))
            fi
        else
            print_success "✓ 镜像已存在: $expected_image"
        fi
    done
    
    if [[ $missing_count -eq 0 ]]; then
        print_success "所有关键镜像都已准备就绪"
        return 0
    else
        print_warning "有 $missing_count 个关键服务构建失败"
        return 1
    fi
}

# 构建单个服务（如果缺失）
build_service_if_missing() {
    local service="$1"
    local compose_file="$2"
    local env_file="$3"
    
    print_info "尝试构建服务: $service"
    
    # 使用docker-compose构建特定服务
    if ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" build "$service" 2>/dev/null; then
        print_success "✓ 构建成功: $service"
        return 0
    else
        print_warning "✗ 构建失败: $service (可能不存在build配置)"
        return 1
    fi
}

# 停止生产环境
stop_production() {
    local compose_file="docker-compose.yml"
    
    if [[ ! -f "$compose_file" ]]; then
        print_error "生产配置文件不存在: $compose_file"
        return 1
    fi
    
    # 检测环境文件 - 生产环境优先使用 .env.prod
    local env_file
    if [[ -f ".env.prod" ]]; then
        env_file=".env.prod"
        print_info "使用生产环境文件: $env_file"
    else
        env_file=$(detect_env_file)
        if [[ $? -ne 0 ]]; then
            return 1
        fi
    fi
    
    print_info "=========================================="
    print_info "停止生产环境"
    print_info "=========================================="
    print_info "使用环境文件: $env_file"
    
    if ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" down; then
        print_success "✓ 生产环境已停止"
        return 0
    else
        print_error "✗ 生产环境停止失败"
        return 1
    fi
}

# 重启生产环境
restart_production() {
    local registry="$1"
    local tag="${2:-$DEFAULT_IMAGE_TAG}"
    
    print_info "=========================================="
    print_info "重启生产环境"
    print_info "=========================================="
    
    # 先停止
    stop_production
    
    # 等待一段时间
    sleep 2
    
    # 再启动
    start_production "$registry" "$tag"
}

# 查看生产环境状态
production_status() {
    local compose_file="docker-compose.yml"
    
    if [[ ! -f "$compose_file" ]]; then
        print_error "生产配置文件不存在: $compose_file"
        return 1
    fi
    
    # 检测环境文件 - 生产环境优先使用 .env.prod
    local env_file
    if [[ -f ".env.prod" ]]; then
        env_file=".env.prod"
        print_info "使用生产环境配置: $env_file"
    else
        env_file=$(detect_env_file)
        if [[ $? -ne 0 ]]; then
            return 1
        fi
    fi
    
    print_info "=========================================="
    print_info "生产环境状态"
    print_info "=========================================="
    print_info "使用环境文件: $env_file"
    
    ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" ps
}

# 查看生产环境日志
production_logs() {
    local compose_file="docker-compose.yml"
    local service="$1"
    local follow="${2:-false}"
    
    if [[ ! -f "$compose_file" ]]; then
        print_error "生产配置文件不存在: $compose_file"
        return 1
    fi
    
    # 检测环境文件
    local env_file
    env_file=$(detect_env_file)
    if [[ $? -ne 0 ]]; then
        return 1
    fi
    
    if [[ -z "$service" ]]; then
        # 显示所有服务的日志
        if [[ "$follow" == "true" ]]; then
            ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" logs -f
        else
            ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" logs --tail=100
        fi
    else
        # 显示指定服务的日志
        if [[ "$follow" == "true" ]]; then
            ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" logs -f "$service"
        else
            ENV_FILE="$env_file" docker-compose -f "$compose_file" --env-file "$env_file" logs --tail=100 "$service"
        fi
    fi
}

# ==========================================
# 服务列表功能
# ==========================================

# 列出所有服务和镜像
list_services() {
    local tag="${1:-$DEFAULT_IMAGE_TAG}"
    local registry="${2:-}"
    
    print_info "=========================================="
    print_info "AI-Infra 服务清单"
    print_info "=========================================="
    print_info "镜像标签: $tag"
    if [[ -n "$registry" ]]; then
        print_info "目标仓库: $registry"
    else
        print_info "目标仓库: 本地构建"
    fi
    echo
    
    local service_count=0
    for service in $SRC_SERVICES; do
        service_count=$((service_count + 1))
    done
    
    print_info "📦 源码服务 ($service_count 个):"
    for service in $SRC_SERVICES; do
        local service_path=$(get_service_path "$service")
        local dockerfile_path="$service_path/Dockerfile"
        local base_image="ai-infra-${service}:${tag}"
        local target_image="$base_image"
        
        if [[ -n "$registry" ]]; then
            target_image=$(get_private_image_name "$base_image" "$registry")
        fi
        
        # 检查 Dockerfile 是否存在
        local status="✅"
        if [[ ! -f "$SCRIPT_DIR/$dockerfile_path" ]]; then
            status="❌"
        fi
        
        echo "  $status $service"
        echo "       Dockerfile: $dockerfile_path"
        echo "       镜像名称: $target_image"
        echo
    done
    
    print_info "=========================================="
}

# ==========================================
# 镜像验证功能
# ==========================================

# 验证单个镜像是否可用
verify_image() {
    local image="$1"
    local timeout="${2:-10}"
    
    # 先尝试检查本地镜像
    if docker image inspect "$image" >/dev/null 2>&1; then
        return 0
    fi
    
    # 尝试拉取验证（用于远程镜像）
    if timeout "$timeout" docker pull "$image" >/dev/null 2>&1; then
        return 0
    fi
    
    return 1
}

# 验证私有仓库中的所有AI-Infra镜像
verify_private_images() {
    local registry="$1"
    local tag="${2:-v0.3.5}"
    
    if [[ -z "$registry" ]]; then
        print_error "使用方法: verify <registry_base> [tag]"
        print_info "示例: verify aiharbor.msxf.local/aihpc v0.3.5"
        return 1
    fi
    
    print_info "=== AI Infrastructure Matrix 镜像验证 ==="
    print_info "目标仓库: $registry"
    print_info "镜像标签: $tag"
    print_info "开始时间: $(date)"
    echo
    
    print_info "📋 Harbor项目检查："
    print_info "验证前请确保以下项目已在Harbor中创建："
    print_info "  • aihpc (主项目)"
    print_info "  • library (基础镜像)"
    print_info "  • tecnativa (第三方镜像)"
    print_info "  • redislabs (第三方镜像)"
    print_info "  • minio (第三方镜像)"
    echo
    print_info "如未创建，请参考: docs/HARBOR_PROJECT_SETUP.md"
    echo
    
    # 源码镜像列表
    local source_images=(
        "ai-infra-backend-init"
        "ai-infra-backend"
        "ai-infra-frontend"
        "ai-infra-jupyterhub"
        "ai-infra-singleuser"
        "ai-infra-saltstack"
        "ai-infra-nginx"
        "ai-infra-gitea"
    )
    
    # 基础镜像列表（从配置文件获取）
    local base_image_patterns=(
        "postgres:15-alpine"
        "redis:7-alpine"
        "nginx:1.27-alpine"
        "tecnativa/tcp-proxy:latest"
        "redislabs/redisinsight:latest"
        "quay.io/minio/minio:latest"
    )
    
    local total_images=$((${#source_images[@]} + ${#base_image_patterns[@]}))
    local success_count=0
    local failed_images=()
    
    print_info "计划验证 $total_images 个镜像"
    print_info "============================================"
    
    # 验证源码镜像
    print_info "验证源码镜像 (${#source_images[@]} 个):"
    for image_base in "${source_images[@]}"; do
        local target_image="${registry}/${image_base}:${tag}"
        
        printf "  检查: %-45s" "$target_image"
        if verify_image "$target_image" 5; then
            echo "    ✓ 可用"
            ((success_count++))
        else
            echo "    ✗ 不可用"
            failed_images+=("$target_image")
        fi
    done
    
    echo
    # 验证基础镜像
    print_info "验证基础镜像 (${#base_image_patterns[@]} 个):"
    for base_pattern in "${base_image_patterns[@]}"; do
        # 使用映射配置获取目标镜像名
        local target_image
        target_image=$(get_mapped_private_image "$base_pattern" "$registry" "$tag")
        
        printf "  检查: %-45s" "$target_image"
        if verify_image "$target_image" 5; then
            echo "    ✓ 可用"
            ((success_count++))
        else
            echo "    ✗ 不可用"
            failed_images+=("$target_image")
        fi
    done
    
    echo
    print_info "============================================"
    print_info "验证结果汇总:"
    print_info "总计镜像: $total_images"
    print_success "验证通过: $success_count"
    print_error "验证失败: $((total_images - success_count))"
    
    if [[ ${#failed_images[@]} -gt 0 ]]; then
        echo
        print_error "失败镜像列表:"
        for failed_image in "${failed_images[@]}"; do
            echo "  ✗ $failed_image"
        done
        
        echo
        print_info "建议操作:"
        print_info "1. 检查网络连接和仓库权限"
        print_info "2. 重新运行基础镜像迁移脚本:"
        print_info "   ./scripts/migrate-base-images.sh $registry"
        print_info "3. 重新构建和推送源码镜像:"
        print_info "   ./build.sh build-push $registry $tag"
        
        return 1
    else
        echo
        print_success "🎉 所有镜像验证通过！"
        return 0
    fi
}

# 快速验证关键镜像
verify_key_images() {
    local registry="$1"
    local tag="${2:-v0.3.5}"
    
    if [[ -z "$registry" ]]; then
        print_error "使用方法: verify-key <registry_base> [tag]"
        return 1
    fi
    
    print_info "=== 快速验证关键镜像 ==="
    print_info "目标仓库: $registry"
    print_info "镜像标签: $tag"
    echo
    
    # 关键服务镜像
    local key_images=(
        "ai-infra-backend"
        "ai-infra-frontend" 
        "ai-infra-jupyterhub"
        "ai-infra-nginx"
    )
    
    # 关键基础镜像
    local key_base_images=(
        "postgres:15-alpine"
        "redis:7-alpine"
    )
    
    local success_count=0
    local total_count=$((${#key_images[@]} + ${#key_base_images[@]}))
    
    print_info "验证关键服务镜像:"
    for image_base in "${key_images[@]}"; do
        local target_image="${registry}/${image_base}:${tag}"
        printf "  %-40s" "$target_image"
        
        if verify_image "$target_image" 3; then
            echo " ✓"
            ((success_count++))
        else
            echo " ✗"
        fi
    done
    
    print_info "验证关键基础镜像:"
    for base_pattern in "${key_base_images[@]}"; do
        local target_image
        target_image=$(get_mapped_private_image "$base_pattern" "$registry" "$tag")
        printf "  %-40s" "$target_image"
        
        if verify_image "$target_image" 3; then
            echo " ✓"
            ((success_count++))
        else
            echo " ✗"
        fi
    done
    
    echo
    if [[ $success_count -eq $total_count ]]; then
        print_success "🎉 所有关键镜像验证通过 ($success_count/$total_count)"
        return 0
    else
        print_warning "⚠ 部分关键镜像验证失败 ($success_count/$total_count)"
        return 1
    fi
}

# ==========================================
# 清理功能
# ==========================================

# 清理本地镜像
clean_images() {
    local tag="${1:-$DEFAULT_IMAGE_TAG}"
    local force="${2:-false}"
    
    print_info "=========================================="
    print_info "清理本地 AI-Infra 镜像"
    print_info "=========================================="
    print_info "目标标签: $tag"
    echo
    
    local images_to_clean=()
    
    # 收集需要清理的镜像
    for service in $SRC_SERVICES; do
        local image="ai-infra-${service}:${tag}"
        if docker image inspect "$image" >/dev/null 2>&1; then
            images_to_clean+=("$image")
        fi
    done
    
    if [[ ${#images_to_clean[@]} -eq 0 ]]; then
        print_info "没有找到需要清理的镜像"
        return 0
    fi
    
    print_info "找到 ${#images_to_clean[@]} 个镜像:"
    for image in "${images_to_clean[@]}"; do
        echo "  • $image"
    done
    echo
    
    if [[ "$force" != "true" ]]; then
        read -p "确认删除这些镜像? (y/N): " confirm
        if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
            print_info "已取消清理操作"
            return 0
        fi
    fi
    
    # 删除镜像
    local success_count=0
    for image in "${images_to_clean[@]}"; do
        if docker rmi "$image" 2>/dev/null; then
            print_success "✓ 已删除: $image"
            success_count=$((success_count + 1))
        else
            print_error "✗ 删除失败: $image"
        fi
    done
    
    print_success "清理完成: $success_count/${#images_to_clean[@]} 成功"
}

# 清理所有AI-Infra相关资源（镜像、容器、数据卷、配置文件）
clean_all() {
    local force="${1:-false}"
    
    # 处理帮助参数
    if [[ "$1" == "--help" || "$1" == "-h" ]]; then
        echo "clean-all - 完整清理AI-Infra系统"
        echo
        echo "用法: $0 clean-all [--force]"
        echo
        echo "选项:"
        echo "  --force    跳过确认提示，强制执行清理"
        echo
        echo "说明:"
        echo "  此命令将删除所有AI-Infra相关的资源："
        echo "  • 容器和服务"
        echo "  • 镜像"
        echo "  • 数据卷（数据库、文件等）"
        echo "  • 配置文件（.env.prod, docker-compose.yml）"
        echo
        echo "警告: 此操作不可逆转，请谨慎使用！"
        return 0
    fi
    
    print_info "=========================================="
    print_info "完整清理 AI-Infra 系统"
    print_info "=========================================="
    print_warning "⚠️  这将删除所有AI-Infra相关的:"
    print_warning "   • 容器和服务"
    print_warning "   • 镜像"
    print_warning "   • 数据卷（数据库、文件等）"
    print_warning "   • 配置文件（.env.prod, docker-compose.yml）"
    echo
    
    if [[ "$force" != "true" ]]; then
        read -p "确认执行完整清理? 这将删除所有数据! (yes/NO): " confirm
        if [[ "$confirm" != "yes" ]]; then
            print_info "已取消清理操作"
            return 0
        fi
    fi
    
    # 1. 停止并删除容器
    print_info "1. 停止并删除容器..."
    if [[ -f "docker-compose.yml" && -f ".env.prod" ]]; then
        ENV_FILE=.env.prod docker-compose -f docker-compose.yml --env-file .env.prod down --remove-orphans 2>/dev/null || true
    fi
    
    # 删除所有ai-infra相关容器
    local containers=$(docker ps -aq --filter "name=ai-infra" 2>/dev/null || true)
    if [[ -n "$containers" ]]; then
        docker rm -f $containers 2>/dev/null || true
        print_success "✓ 容器清理完成"
    else
        print_info "  没有找到相关容器"
    fi
    
    # 2. 删除镜像
    print_info "2. 删除镜像..."
    local images=$(docker images --format "{{.Repository}}:{{.Tag}}" | grep "ai-infra" 2>/dev/null || true)
    if [[ -n "$images" ]]; then
        echo "$images" | xargs docker rmi -f 2>/dev/null || true
        print_success "✓ 镜像清理完成"
    else
        print_info "  没有找到相关镜像"
    fi
    
    # 3. 删除数据卷
    print_info "3. 删除数据卷..."
    local volumes=$(docker volume ls --format "{{.Name}}" | grep "ai-infra" 2>/dev/null || true)
    if [[ -n "$volumes" ]]; then
        echo "$volumes" | xargs docker volume rm -f 2>/dev/null || true
        print_success "✓ 数据卷清理完成"
    else
        print_info "  没有找到相关数据卷"
    fi
    
    # 4. 删除网络
    print_info "4. 删除网络..."
    local networks=$(docker network ls --format "{{.Name}}" | grep "ai-infra" 2>/dev/null || true)
    if [[ -n "$networks" ]]; then
        echo "$networks" | xargs docker network rm 2>/dev/null || true
        print_success "✓ 网络清理完成"
    else
        print_info "  没有找到相关网络"
    fi
    
    # 5. 删除配置文件
    print_info "5. 删除配置文件..."
    local files_removed=0
    if [[ -f ".env.prod" ]]; then
        rm -f .env.prod
        files_removed=$((files_removed + 1))
        print_info "  ✓ 删除 .env.prod"
    fi
    if [[ -f "docker-compose.yml" ]]; then
        rm -f docker-compose.yml
        files_removed=$((files_removed + 1))
        print_info "  ✓ 删除 docker-compose.yml"
    fi
    if [[ $files_removed -gt 0 ]]; then
        print_success "✓ 配置文件清理完成 ($files_removed 个文件)"
    else
        print_info "  没有找到配置文件"
    fi
    
    # 6. 清理备份文件
    print_info "6. 清理备份文件..."
    local backup_files=$(find . -maxdepth 1 -name "*.env.prod.backup.*" -o -name "docker-compose.yml.bak" 2>/dev/null || true)
    if [[ -n "$backup_files" ]]; then
        echo "$backup_files" | xargs rm -f 2>/dev/null || true
        print_success "✓ 备份文件清理完成"
    else
        print_info "  没有找到备份文件"
    fi
    
    echo
    print_success "🎉 完整清理完成！"
    print_info "提示: 使用以下命令重新部署系统:"
    print_info "  1. ./build.sh create-env-prod intranet \"\" v0.3.6"
    print_info "  2. ./build.sh build-all \"\" v0.3.6"
    print_info "  3. docker compose -f docker-compose.yml.example up -d"
}

# 重置数据库（仅删除数据库相关数据卷）
reset_database() {
    local force="${1:-false}"
    
    # 处理帮助参数
    if [[ "$1" == "--help" || "$1" == "-h" ]]; then
        echo "reset-db - 重置数据库"
        echo
        echo "用法: $0 reset-db [--force]"
        echo
        echo "选项:"
        echo "  --force    跳过确认提示，强制执行重置"
        echo
        echo "说明:"
        echo "  此命令将删除所有数据库相关的数据卷："
        echo "  • PostgreSQL 数据"
        echo "  • Redis 数据"
        echo "  • JupyterHub 数据"
        echo "  • Gitea 数据"
        echo
        echo "注意:"
        echo "  • 镜像和容器不会被删除"
        echo "  • 配置文件不会被删除"
        echo "  • 此操作不可逆转，请谨慎使用！"
        return 0
    fi
    
    print_info "=========================================="
    print_info "重置数据库"
    print_info "=========================================="
    print_warning "⚠️  这将删除所有数据库数据:"
    print_warning "   • PostgreSQL 数据"
    print_warning "   • Redis 数据"
    print_warning "   • JupyterHub 数据"
    print_warning "   • Gitea 数据"
    echo
    
    if [[ "$force" != "true" ]]; then
        read -p "确认重置数据库? 这将删除所有数据! (yes/NO): " confirm
        if [[ "$confirm" != "yes" ]]; then
            print_info "已取消重置操作"
            return 0
        fi
    fi
    
    # 停止相关服务
    print_info "停止数据库相关服务..."
    if [[ -f "docker-compose.yml" && -f ".env.prod" ]]; then
        ENV_FILE=.env.prod docker-compose -f docker-compose.yml --env-file .env.prod stop postgres redis jupyterhub gitea backend backend-init 2>/dev/null || true
    fi
    
    # 删除数据库相关数据卷
    print_info "删除数据库数据卷..."
    local db_volumes=(
        "ai-infra-postgres-data"
        "ai-infra-redis-data"
        "ai-infra-jupyterhub-data"
        "ai-infra-jupyterhub-notebooks"
        "ai-infra-gitea-data"
    )
    
    local removed_count=0
    for volume in "${db_volumes[@]}"; do
        if docker volume inspect "$volume" >/dev/null 2>&1; then
            if docker volume rm "$volume" 2>/dev/null; then
                print_info "  ✓ 删除 $volume"
                removed_count=$((removed_count + 1))
            else
                print_error "  ✗ 删除失败 $volume (可能被容器使用)"
            fi
        fi
    done
    
    if [[ $removed_count -gt 0 ]]; then
        print_success "✓ 数据库重置完成 ($removed_count 个数据卷)"
        print_info "提示: 使用 ./build.sh prod-up 重新启动服务"
    else
        print_info "没有找到需要重置的数据库数据卷"
    fi
}

# 显示帮助信息
show_help() {
    echo "AI Infrastructure Matrix - 构建脚本 v$VERSION"
    echo
    echo "用法: $0 [--force|--skip-pull] <命令> [参数...]"
    echo
    echo "全局选项:"
    echo "  --force      - 强制重新构建/跳过镜像拉取"
    echo "  --skip-pull  - 跳过镜像拉取，使用本地镜像"
    echo
    echo "主要命令:"
    echo "  list [tag] [registry]           - 列出所有服务和镜像"
    echo "  build <service> [tag] [registry] - 构建单个服务"
    echo "  build-all [tag] [registry]      - 构建所有服务"
    echo "  build-push <registry> [tag]     - 构建并推送所有服务"
    echo "  push-all <registry> [tag]       - 推送所有服务"
    echo
    echo "依赖镜像:"
    echo "  deps-pull <registry> [tag]      - 拉取依赖镜像"
    echo "  deps-push <registry> [tag]      - 推送依赖镜像"
    echo "  deps-all <registry> [tag]       - 拉取、标记并推送依赖镜像"
    echo
    echo "AI Harbor镜像拉取:"
    echo "  harbor-pull-services [registry] [tag] - 从AI Harbor拉取AI-Infra服务镜像"
    echo "  harbor-pull-deps [registry] [tag]     - 从AI Harbor拉取依赖镜像"
    echo "  harbor-pull-all [registry] [tag]      - 从AI Harbor拉取所有镜像"
    echo
    echo "生产环境:"
    echo "  prod-deploy <host> [registry] [tag] - 部署到指定HOST（自动配置域名）"
    echo "  prod-up [registry] [tag]        - 启动生产环境"
    echo "  prod-down                       - 停止生产环境"
    echo "  prod-status                     - 查看状态"
    echo "  prod-logs [service] [--follow]  - 查看日志"
    echo "  generate-passwords [file] [--force] - 生成生产环境强密码"
    echo
    echo "工具命令:"
    echo "  clean [tag] [--force]           - 清理镜像"
    echo "  clean-all [--force]             - 完整清理（镜像、容器、数据卷、配置文件）"
    echo "  reset-db [--force]              - 重置数据库（仅删除数据库数据卷）"
    echo "  verify <registry> [tag]         - 验证镜像"
    echo "  create-env [dev|prod] [--force] - 创建环境配置"
    echo "  validate-env                    - 校验环境配置"
    echo "  render-templates [nginx|jupyterhub|all] - 渲染配置模板"
    echo "  version                         - 显示版本"
    echo "  help                            - 显示帮助"
    echo
    echo "动态配置管理:"
    echo "  update-host [host|auto]         - 更新外部主机配置（auto=自动检测）"
    echo "  update-port <port>              - 更新外部端口配置（自动计算相关端口）"
    echo "  quick-deploy [port] [host]      - 一键更新配置并重新部署（默认8080 auto）"
    echo
    echo "===================================================================================="
    echo "📦 CI/CD服务器运行实例 (构建和推送镜像):"
    echo "===================================================================================="
    echo "  # 构建所有服务并推送到私有仓库"
    echo "  $0 build-push harbor.example.com/ai-infra v1.2.0"
    echo
    echo "  # 推送依赖镜像到私有仓库"
    echo "  $0 deps-all harbor.example.com/ai-infra v1.2.0"
    echo
    echo "  # 分步骤操作（推荐用于CI/CD Pipeline）"
    echo "  $0 build-all v1.2.0                                    # 步骤1: 构建所有服务"
    echo "  $0 push-all harbor.example.com/ai-infra v1.2.0         # 步骤2: 推送项目镜像"
    echo "  $0 deps-push harbor.example.com/ai-infra v1.2.0        # 步骤3: 推送依赖镜像"
    echo
    echo "===================================================================================="
    echo "🚀 生产节点运行实例 (启动服务):"
    echo "===================================================================================="
    echo "  # 从AI Harbor拉取镜像完整部署流程"
    echo "  $0 harbor-pull-all aiharbor.msxf.local/aihpc v1.2.0    # 步骤1: 拉取所有镜像"
    echo "  $0 generate-passwords .env.prod --force                # 步骤2: 生成强密码"
    echo "  docker compose -f docker-compose.yml.example up -d     # 步骤3: 启动所有服务"
    echo
    echo "  # 标准私有仓库部署流程"
    echo "  $0 generate-passwords .env.prod --force                # 步骤1: 生成强密码"
    echo "  docker compose -f docker-compose.yml.example up -d     # 步骤2: 启动所有服务"
    echo
    echo "  # 快速启动 (生产配置已存在)"
    echo "  $0 prod-up harbor.example.com/ai-infra v1.2.0"
    echo
    echo "  # 本地镜像部署 (无需registry)"
    echo "  $0 generate-passwords .env.prod                        # 生成密码"
    echo "  docker compose -f docker-compose.yml.example up -d     # 启动服务"
    echo
    echo "  # 服务管理"
    echo "  $0 prod-status                                         # 查看服务状态"
    echo "  $0 prod-logs jupyterhub --follow                       # 查看实时日志"
    echo "  $0 prod-down                                           # 停止所有服务"
    echo
    echo "===================================================================================="
    echo "💡 常用开发实例:"
    echo "===================================================================================="
    echo "  # 从AI Harbor快速获取镜像进行本地开发"
    echo "  $0 harbor-pull-services aiharbor.msxf.local/aihpc v1.2.0  # 拉取AI-Infra服务"
    echo "  $0 harbor-pull-deps aiharbor.msxf.local/aihpc             # 拉取依赖镜像"
    echo "  docker compose -f docker-compose.yml.example up -d        # 启动服务"
    echo
    echo "  # 本地开发测试"
    echo "  $0 build-all test-v0.3.5                              # 构建测试版本"
    echo "  docker compose -f docker-compose.yml.example up -d backend frontend  # 启动核心服务"
    echo
    echo "  # 单服务调试"
    echo "  $0 build backend test-debug                           # 构建调试版本"
    echo "  docker compose up -d postgres redis                  # 启动依赖"
    echo "  docker run --rm -it ai-infra-backend:test-debug bash  # 交互式调试"
    echo
    echo "===================================================================================="
    echo "🔧 动态配置管理实例:"
    echo "===================================================================================="
    echo "  # 自动检测外部IP并更新配置"
    echo "  $0 update-host auto                                   # 自动检测外部主机IP"
    echo "  $0 build nginx --force && docker compose restart nginx  # 应用新配置"
    echo
    echo "  # 手动指定外部主机"
    echo "  $0 update-host 192.168.1.100                         # 设置外部主机为指定IP"
    echo
    echo "  # 修改外部端口（便捷部署不同环境）"
    echo "  $0 update-port 9090                                  # 更新外部端口为9090"
    echo "                                                        # 自动计算：主入口9090，JupyterHub9098，Gitea4020"
    echo "  $0 build nginx --force                               # 重新构建nginx配置"
    echo "  docker compose down && docker compose up -d          # 重启所有服务"
    echo
    echo "  # 快速切换部署端口"
    echo "  $0 update-port 8080 && $0 build nginx --force        # 切换到8080端口并更新配置"
    echo "  $0 update-port 9000 && $0 build nginx --force        # 切换到9000端口并更新配置"
    echo
    echo "===================================================================================="
    echo "⚠️  重要提醒:"
    echo "  • 首次部署必须运行 generate-passwords 生成强密码"
    echo "  • 默认管理员账户: admin / admin123 (部署后请立即修改)"
    echo "  • 生产环境配置文件 docker-compose.yml 会被自动生成，请勿手动编辑"
    echo "  • 服务访问端口: Web界面:8080, JupyterHub:8088, Gitea:3010"
    echo "===================================================================================="
}

# 主函数
main() {
    # 预处理命令行参数，检查 --force 和 --skip-pull 标志
    local args=()
    for arg in "$@"; do
        if [[ "$arg" == "--force" ]]; then
            FORCE_REBUILD=true
            print_info "启用强制重新构建模式"
        elif [[ "$arg" == "--skip-pull" ]]; then
            SKIP_PULL=true
            print_info "启用跳过拉取模式"
        else
            args+=("$arg")
        fi
    done
    
    # 重新设置位置参数
    set -- "${args[@]}"
    
    # 早期Docker Compose兼容性检查
    if [[ "${1:-}" != "version" && "${1:-}" != "help" && "${1:-}" != "-h" && "${1:-}" != "--help" ]]; then
        if ! check_compose_compatibility; then
            exit 1
        fi
    fi
    
    case "${1:-help}" in
        "list")
            list_services "${2:-$DEFAULT_IMAGE_TAG}" "$3"
            ;;
            
        "build")
            if [[ -z "$2" ]]; then
                print_error "请指定要构建的服务"
                print_info "可用服务: $SRC_SERVICES"
                exit 1
            fi
            build_service "$2" "${3:-$DEFAULT_IMAGE_TAG}" "$4"
            ;;
            
        "build-all")
            build_all_services "${2:-$DEFAULT_IMAGE_TAG}" "$3"
            ;;
            
        "test-push")
            if [[ -z "$2" ]]; then
                print_error "请指定要测试推送的服务"
                print_info "可用服务: $SRC_SERVICES"
                exit 1
            fi
            if [[ -z "$3" ]]; then
                print_error "请指定目标 registry"
                exit 1
            fi
            
            local service="$2"
            local registry="$3"
            local tag="${4:-$DEFAULT_IMAGE_TAG}"
            local base_image="ai-infra-${service}:${tag}"
            local target_image=$(get_private_image_name "$base_image" "$registry")
            
            print_info "=========================================="
            print_info "测试推送配置（不实际推送）"
            print_info "=========================================="
            print_info "服务名称: $service"
            print_info "Registry: $registry"
            print_info "标签: $tag"
            print_info "原始镜像: $base_image"
            print_info "目标镜像: $target_image"
            
            # 检查镜像是否存在
            if docker image inspect "$base_image" >/dev/null 2>&1; then
                print_success "✓ 本地镜像存在: $base_image"
            else
                print_warning "✗ 本地镜像不存在: $base_image"
                print_info "需要先构建镜像：./build.sh build $service $tag"
            fi
            
            print_info "推送命令预览："
            print_info "  docker tag $base_image $target_image"
            print_info "  docker push $target_image"
            ;;
            
        "push")
            if [[ -z "$2" ]]; then
                print_error "请指定要推送的服务"
                print_info "可用服务: $SRC_SERVICES"
                exit 1
            fi
            if [[ -z "$3" ]]; then
                print_error "请指定目标 registry"
                exit 1
            fi
            push_service "$2" "${4:-$DEFAULT_IMAGE_TAG}" "$3"
            ;;
            
        "push-all")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                exit 1
            fi
            push_all_services "${3:-$DEFAULT_IMAGE_TAG}" "$2"
            ;;
            
        "build-push")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                exit 1
            fi
            build_and_push_all "${3:-$DEFAULT_IMAGE_TAG}" "$2"
            ;;
            
        # 双环境部署命令
        "build-env")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                print_info "示例: $0 build-env aiharbor.msxf.local/aihpc v0.3.5"
                exit 1
            fi
            build_environment_deploy "$2" "${3:-$DEFAULT_IMAGE_TAG}"
            ;;
            
        "intranet-env")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                print_info "示例: $0 intranet-env aiharbor.msxf.local/aihpc v0.3.5"
                exit 1
            fi
            intranet_environment_deploy "$2" "${3:-$DEFAULT_IMAGE_TAG}"
            ;;
            
        "create-env-prod")
            local mode="${2:-production}"
            local registry="${3:-aiharbor.msxf.local/aihpc}"
            local tag="${4:-$DEFAULT_IMAGE_TAG}"
            create_production_env "$mode" "$registry" "$tag"
            ;;
            
        # 环境配置管理命令
        "create-env")
            local env_type="${2:-dev}"
            local force="false"
            
            # 使用全局 FORCE_REBUILD 标志
            if [[ "$FORCE_REBUILD" == "true" ]]; then
                force="true"
            fi
            
            create_env_from_template "$env_type" "$force"
            ;;
            
        # 更新外部主机配置命令
        "update-host")
            local host_ip="${2:-auto}"
            update_external_host_config "$host_ip"
            ;;
            
        # 更新外部端口配置命令
        "update-port")
            local port="${2:-8080}"
            update_external_port_config "$port"
            ;;
            
        # 一键更新端口并重新部署
        "quick-deploy")
            local port="${2:-8080}"
            local host="${3:-auto}"
            quick_deploy_with_port "$port" "$host"
            ;;
            
        "auto-env")
            local force="false"
            
            # 使用全局 FORCE_REBUILD 标志
            if [[ "$FORCE_REBUILD" == "true" ]]; then
                force="true"
            fi
            
            auto_generate_env_files "$force"
            ;;
            
        # 生成生产环境密码命令
        "generate-passwords")
            local env_file="${2:-.env.prod}"
            local force="false"
            if [[ "$FORCE_REBUILD" == "true" || "$3" == "--force" ]]; then
                force="true"
            fi
            
            if generate_production_passwords "$env_file" "$force"; then
                print_success "✓ 生产环境密码生成完成"
            else
                print_error "密码生成失败"
                exit 1
            fi
            ;;
            
        # 依赖镜像管理命令
        "deps-pull")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                print_info "用法: $0 deps-pull <registry> [tag]"
                exit 1
            fi
            pull_and_tag_dependencies "$2" "${3:-v0.3.5}"
            ;;
            
        "deps-push")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                print_info "用法: $0 deps-push <registry> [tag]"
                exit 1
            fi
            push_dependencies "$2" "${3:-v0.3.5}"
            ;;
            
        "deps-all")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                exit 1
            fi
            local deps_tag="${3:-v0.3.5}"
            print_info "执行完整的依赖镜像操作..."
            if pull_and_tag_dependencies "$2" "$deps_tag"; then
                push_dependencies "$2" "$deps_tag"
            else
                print_error "依赖镜像拉取失败，停止推送操作"
                exit 1
            fi
            ;;
            
        # AI Harbor 镜像拉取命令
        "harbor-pull-services")
            local harbor_registry="${2:-aiharbor.msxf.local/aihpc}"
            local harbor_tag="${3:-$DEFAULT_IMAGE_TAG}"
            pull_aiharbor_services "$harbor_registry" "$harbor_tag"
            ;;
            
        "harbor-pull-deps")
            local harbor_registry="${2:-aiharbor.msxf.local/aihpc}"
            local harbor_tag="${3:-$DEFAULT_IMAGE_TAG}"
            pull_aiharbor_dependencies "$harbor_registry" "$harbor_tag"
            ;;
            
        "harbor-pull-all")
            local harbor_registry="${2:-aiharbor.msxf.local/aihpc}"
            local harbor_tag="${3:-$DEFAULT_IMAGE_TAG}"
            pull_aiharbor_all "$harbor_registry" "$harbor_tag"
            ;;
            
        "deps-prod")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                exit 1
            fi
            local deps_tag="${3:-v0.3.5}"
            print_info "执行生产环境依赖镜像操作（排除测试工具）..."
            if pull_and_tag_production_dependencies "$2" "$deps_tag"; then
                push_production_dependencies "$2" "$deps_tag"
            else
                print_error "生产环境依赖镜像拉取失败，停止推送操作"
                exit 1
            fi
            ;;
            
        "prod-deploy")
            if [[ -z "$2" ]]; then
                print_error "请指定部署的HOST地址"
                print_info "用法: $0 prod-deploy <host> [registry] [tag]"
                print_info "示例: $0 prod-deploy 192.168.1.100 harbor.company.com/ai-infra v1.0.0"
                print_info "示例: $0 prod-deploy example.com \"\" v1.0.0  # 使用本地镜像"
                exit 1
            fi
            deploy_to_host "$2" "${3:-}" "${4:-$DEFAULT_IMAGE_TAG}"
            ;;
            
        "prod-up")
            # registry 参数可以为空（使用本地镜像）
            # 检查是否有 --force 或 --skip-pull 参数
            local force_local="false"
            if [[ "$FORCE_REBUILD" == "true" || "$SKIP_PULL" == "true" ]]; then
                force_local="true"
            fi
            start_production "${2:-}" "${3:-$DEFAULT_IMAGE_TAG}" "$force_local"
            ;;
            
        "prod-down")
            stop_production
            ;;
            
        "prod-restart")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                exit 1
            fi
            restart_production "$2" "${3:-$DEFAULT_IMAGE_TAG}"
            ;;
            
        "prod-status")
            production_status
            ;;
            
        "prod-logs")
            local follow="false"
            if [[ "$3" == "--follow" || "$3" == "-f" ]]; then
                follow="true"
            fi
            production_logs "$2" "$follow"
            ;;
            
        # Mock 测试环境命令
        "mock-setup")
            setup_mock_environment "${2:-$DEFAULT_IMAGE_TAG}"
            ;;
            
        "mock-up"|"mock-start")
            run_mock_tests "${2:-$DEFAULT_IMAGE_TAG}" "up"
            ;;
            
        "mock-down"|"mock-stop")
            run_mock_tests "${2:-$DEFAULT_IMAGE_TAG}" "down"
            ;;
            
        "mock-restart")
            run_mock_tests "${2:-$DEFAULT_IMAGE_TAG}" "restart"
            ;;
            
        "mock-test")
            run_mock_tests "${2:-$DEFAULT_IMAGE_TAG}" "test"
            ;;
            
        # 镜像验证命令
        "verify")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                print_info "用法: $0 verify <registry> [tag]"
                exit 1
            fi
            verify_private_images "$2" "${3:-v0.3.5}"
            ;;
            
        "verify-key")
            if [[ -z "$2" ]]; then
                print_error "请指定目标 registry"
                print_info "用法: $0 verify-key <registry> [tag]"
                exit 1
            fi
            verify_key_images "$2" "${3:-v0.3.5}"
            ;;
            
        "clean")
            local clean_type="${2:-ai-infra}"
            local tag_or_force="$3"
            local force_flag="$4"
            local force="false"
            local tag="$DEFAULT_IMAGE_TAG"
            
            # 解析参数
            case "$clean_type" in
                "ai-infra"|*)
                    # 默认清理AI-Infra镜像（保持原有行为）
                    if [[ "$clean_type" != "ai-infra" && "$clean_type" != "--force" ]]; then
                        tag="$clean_type"
                    fi
                    if [[ "$tag_or_force" == "--force" ]]; then
                        force="true"
                    elif [[ -n "$tag_or_force" && "$tag_or_force" != "--force" && "$clean_type" == "ai-infra" ]]; then
                        tag="$tag_or_force"
                        if [[ "$force_flag" == "--force" ]]; then
                            force="true"
                        fi
                    fi
                    clean_images "$tag" "$force"
                    ;;
            esac
            ;;
            
        "clean-all")
            # 检查是否需要帮助
            if [[ "$2" == "--help" || "$2" == "-h" ]]; then
                clean_all "--help"
                exit 0
            fi
            
            # 使用全局FORCE_REBUILD变量
            local force="false"
            if [[ "$FORCE_REBUILD" == "true" ]]; then
                force="true"
            fi
            clean_all "$force"
            ;;
            
        "reset-db")
            # 检查是否需要帮助
            if [[ "$2" == "--help" || "$2" == "-h" ]]; then
                reset_database "--help"
                exit 0
            fi
            
            # 使用全局FORCE_REBUILD变量
            local force="false"
            if [[ "$FORCE_REBUILD" == "true" ]]; then
                force="true"
            fi
            reset_database "$force"
            ;;
            
        "render-templates")
            case "${2:-all}" in
                "nginx")
                    render_nginx_templates
                    ;;
                "jupyterhub")
                    render_jupyterhub_templates
                    ;;
                "all")
                    render_nginx_templates
                    render_jupyterhub_templates
                    ;;
                *)
                    print_error "未知的模板类型: $2"
                    print_info "可用模板类型: nginx, jupyterhub, all"
                    exit 1
                    ;;
            esac
            ;;
            
        "version")
            echo "AI Infrastructure Matrix Build Script"
            echo "Version: $VERSION"
            echo "Default Tag: $DEFAULT_IMAGE_TAG"
            echo "Services: $SRC_SERVICES"
            echo
            echo "Dependency Images:"
            for dep in $DEPENDENCY_IMAGES; do
                echo "  • $dep"
            done
            ;;
            
        "validate-env")
            validate_env_consistency
            ;;
            
        "help"|"-h"|"--help")
            show_help
            ;;
            
        *)
            print_error "未知命令: $1"
            print_info "使用 '$0 help' 查看可用命令"
            exit 1
            ;;
    esac
}

# 执行主函数
main "$@"
