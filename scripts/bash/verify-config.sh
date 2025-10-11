#!/bin/bash

# 配置文件验证脚本
# 检查所有必需的环境变量是否已设置

set -euo pipefail

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

ENV_FILE="${1:-.env.prod}"

if [[ ! -f "$ENV_FILE" ]]; then
    print_error "环境文件不存在: $ENV_FILE"
    exit 1
fi

print_info "检查环境文件: $ENV_FILE"

# 必需的环境变量列表
REQUIRED_VARS=(
    "COMPOSE_PROJECT_NAME"
    "IMAGE_TAG"
    "POSTGRES_DB"
    "POSTGRES_USER"
    "POSTGRES_PASSWORD"
    "REDIS_PASSWORD"
    "JWT_SECRET"
    "CONFIGPROXY_AUTH_TOKEN"
    "JUPYTERHUB_ADMIN_USERS"
    "JUPYTERHUB_CRYPT_KEY"
    "JUPYTERHUB_CORS_ORIGIN"
    "JUPYTERHUB_MEM_LIMIT"
    "JUPYTERHUB_CPU_LIMIT"
    "JUPYTERHUB_IDLE_TIMEOUT"
    "JUPYTERHUB_CULL_TIMEOUT"
    "JUPYTERHUB_DEBUG"
    "JUPYTERHUB_LOG_LEVEL"
    "JUPYTERHUB_ACCESS_LOG"
    "JUPYTERHUB_IDLE_CULLER_ENABLED"
    "MINIO_ACCESS_KEY"
    "MINIO_SECRET_KEY"
    "GITEA_ADMIN_USER"
    "GITEA_ADMIN_PASSWORD"
    "GITEA_ADMIN_EMAIL"
    "GITEA_ADMIN_FULL_NAME"
    "GITEA_ENABLED"
    "GITEA_DB_USER"
    "GITEA_DB_PASSWD"
    "GITEA_ALIAS_ADMIN_TO"
    "GITEA_BASE_URL"
    "GITEA_AUTO_CREATE"
    "LDAP_ADMIN_PASSWORD"
    "LDAP_CONFIG_PASSWORD"
    "NGINX_PORT"
    "LOG_LEVEL"
    "ENV_FILE"
    "DEBUG_MODE"
    "BUILD_ENV"
)

# 可以为空的变量列表
OPTIONAL_VARS=(
    "GITEA_ADMIN_TOKEN"
    "IMAGE_REGISTRY_PREFIX"
)

# 加载环境文件
source "$ENV_FILE"

MISSING_VARS=()
WEAK_PASSWORDS=()

# 检查每个必需变量
for var in "${REQUIRED_VARS[@]}"; do
    if [[ -z "${!var:-}" ]]; then
        MISSING_VARS+=("$var")
    else
        # 检查是否是默认的弱密码
        case "$var" in
            *PASSWORD*|*SECRET*|*KEY*)
                if [[ "${!var}" == *"change_me"* ]] || [[ "${!var}" == "admin123" ]] || [[ "${!var}" == "postgres" ]]; then
                    WEAK_PASSWORDS+=("$var")
                fi
                ;;
        esac
    fi
done

# 检查可选变量（确保它们在环境文件中被定义，即使是空值）
for var in "${OPTIONAL_VARS[@]}"; do
    if ! grep -q "^${var}=" "$ENV_FILE"; then
        MISSING_VARS+=("$var")
    else
        # 检查是否是默认的弱密码（即使是可选变量）
        case "$var" in
            *PASSWORD*|*SECRET*|*KEY*)
                if [[ "${!var:-}" == *"change_me"* ]] || [[ "${!var:-}" == "admin123" ]]; then
                    WEAK_PASSWORDS+=("$var")
                fi
                ;;
        esac
    fi
done

# 报告结果
if [[ ${#MISSING_VARS[@]} -eq 0 ]]; then
    print_success "所有必需的环境变量都已设置"
else
    print_error "缺少以下环境变量："
    for var in "${MISSING_VARS[@]}"; do
        echo "  - $var"
    done
fi

if [[ ${#WEAK_PASSWORDS[@]} -eq 0 ]]; then
    print_success "没有发现弱密码"
else
    print_warning "发现以下变量使用了默认或弱密码："
    for var in "${WEAK_PASSWORDS[@]}"; do
        echo "  - $var = ${!var}"
    done
    print_warning "建议在生产环境中修改这些密码"
fi

# 验证docker-compose配置
print_info "验证 Docker Compose 配置..."
if ENV_FILE="$ENV_FILE" docker-compose config --quiet; then
    print_success "Docker Compose 配置语法正确"
else
    print_error "Docker Compose 配置验证失败"
    exit 1
fi

if [[ ${#MISSING_VARS[@]} -eq 0 ]]; then
    print_success "配置验证通过！可以启动服务"
    exit 0
else
    print_error "配置验证失败！请修复上述问题后重试"
    exit 1
fi
