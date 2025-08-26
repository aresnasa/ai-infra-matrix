#!/bin/bash

# =============================================================================
# AI Infrastructure Matrix - 生产环境密码生成脚本
# =============================================================================
# 功能：自动生成强密码并更新.env.prod文件
# 注意：此脚本只更改系统服务密码，不会更改默认admin用户密码
# 默认管理员账户: admin / admin123 (请在首次登录后修改)
# =============================================================================

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

# 生成随机密码
generate_password() {
    local length="${1:-32}"
    openssl rand -base64 "$length" | tr -d "=+/" | cut -c1-"$length"
}

# 生成64字符的hex key
generate_hex_key() {
    openssl rand -hex 32
}

ENV_FILE=".env.prod"
BACKUP_FILE=".env.prod.backup.$(date +%Y%m%d_%H%M%S)"

if [[ ! -f "$ENV_FILE" ]]; then
    print_error "环境文件不存在: $ENV_FILE"
    exit 1
fi

echo
print_info "======================================================================"
print_info "🔧 AI Infrastructure Matrix 生产环境密码生成器"
print_info "======================================================================"
print_warning "⚠️  此脚本将生成新的系统服务密码"
print_warning "⚠️  默认管理员账户 (admin/admin123) 不会被此脚本修改"
print_warning "⚠️  请在系统部署后通过Web界面修改管理员密码"
print_info "======================================================================"
echo

print_info "创建备份: $BACKUP_FILE"
cp "$ENV_FILE" "$BACKUP_FILE"

print_info "生成新的强密码..."

# 生成新密码
POSTGRES_PASSWORD=$(generate_password 24)
REDIS_PASSWORD=$(generate_password 24)
JWT_SECRET=$(generate_password 48)
CONFIGPROXY_AUTH_TOKEN=$(generate_password 48)
JUPYTERHUB_CRYPT_KEY=$(generate_hex_key)
MINIO_ACCESS_KEY=$(generate_password 20)
MINIO_SECRET_KEY=$(generate_password 40)
GITEA_ADMIN_PASSWORD=$(generate_password 24)
GITEA_DB_PASSWD=$(generate_password 24)
LDAP_ADMIN_PASSWORD=$(generate_password 24)
LDAP_CONFIG_PASSWORD=$(generate_password 24)

# 更新配置文件
sed -i.tmp "s/POSTGRES_PASSWORD=.*/POSTGRES_PASSWORD=$POSTGRES_PASSWORD/" "$ENV_FILE" && rm -f "$ENV_FILE.tmp"
sed -i.tmp "s/REDIS_PASSWORD=.*/REDIS_PASSWORD=$REDIS_PASSWORD/" "$ENV_FILE" && rm -f "$ENV_FILE.tmp"
sed -i.tmp "s/JWT_SECRET=.*/JWT_SECRET=$JWT_SECRET/" "$ENV_FILE" && rm -f "$ENV_FILE.tmp"
sed -i.tmp "s/CONFIGPROXY_AUTH_TOKEN=.*/CONFIGPROXY_AUTH_TOKEN=$CONFIGPROXY_AUTH_TOKEN/" "$ENV_FILE" && rm -f "$ENV_FILE.tmp"
sed -i.tmp "s/JUPYTERHUB_CRYPT_KEY=.*/JUPYTERHUB_CRYPT_KEY=$JUPYTERHUB_CRYPT_KEY/" "$ENV_FILE" && rm -f "$ENV_FILE.tmp"
sed -i.tmp "s/MINIO_ACCESS_KEY=.*/MINIO_ACCESS_KEY=$MINIO_ACCESS_KEY/" "$ENV_FILE" && rm -f "$ENV_FILE.tmp"
sed -i.tmp "s/MINIO_SECRET_KEY=.*/MINIO_SECRET_KEY=$MINIO_SECRET_KEY/" "$ENV_FILE" && rm -f "$ENV_FILE.tmp"
sed -i.tmp "s/GITEA_ADMIN_PASSWORD=.*/GITEA_ADMIN_PASSWORD=$GITEA_ADMIN_PASSWORD/" "$ENV_FILE" && rm -f "$ENV_FILE.tmp"
sed -i.tmp "s/GITEA_DB_PASSWD=.*/GITEA_DB_PASSWD=$GITEA_DB_PASSWD/" "$ENV_FILE" && rm -f "$ENV_FILE.tmp"
sed -i.tmp "s/LDAP_ADMIN_PASSWORD=.*/LDAP_ADMIN_PASSWORD=$LDAP_ADMIN_PASSWORD/" "$ENV_FILE" && rm -f "$ENV_FILE.tmp"
sed -i.tmp "s/LDAP_CONFIG_PASSWORD=.*/LDAP_CONFIG_PASSWORD=$LDAP_CONFIG_PASSWORD/" "$ENV_FILE" && rm -f "$ENV_FILE.tmp"

print_success "已生成并应用新的强密码"

echo
print_info "======================================================================"
print_warning "🔑 重要！默认管理员账户信息："
echo
echo -e "${GREEN}  用户名: ${YELLOW}admin${NC}"
echo -e "${GREEN}  初始密码: ${RED}admin123${NC}"
echo
print_warning "⚠️  请在首次登录后立即更改管理员密码！"
print_warning "⚠️  管理员密码未通过此脚本更改，需要在系统内修改！"
print_info "======================================================================"
echo

print_info "系统服务密码信息:"
echo "POSTGRES_PASSWORD: $POSTGRES_PASSWORD"
echo "REDIS_PASSWORD: $REDIS_PASSWORD"
echo "JWT_SECRET: $JWT_SECRET"
echo "CONFIGPROXY_AUTH_TOKEN: $CONFIGPROXY_AUTH_TOKEN"
echo "JUPYTERHUB_CRYPT_KEY: $JUPYTERHUB_CRYPT_KEY"
echo "MINIO_ACCESS_KEY: $MINIO_ACCESS_KEY"
echo "MINIO_SECRET_KEY: $MINIO_SECRET_KEY"
echo "GITEA_ADMIN_PASSWORD: $GITEA_ADMIN_PASSWORD"
echo "GITEA_DB_PASSWD: $GITEA_DB_PASSWD"
echo "LDAP_ADMIN_PASSWORD: $LDAP_ADMIN_PASSWORD"
echo "LDAP_CONFIG_PASSWORD: $LDAP_CONFIG_PASSWORD"

echo
print_warning "请妥善保存这些密码信息！"
print_info "原配置文件已备份至: $BACKUP_FILE"
