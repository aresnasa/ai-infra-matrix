#!/bin/bash
#===============================================================================
# Cloudflare Origin Rule 配置脚本
# 用于设置 Origin 端口为 443，解决 525 SSL Handshake Failed 问题
#===============================================================================

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 配置文件路径
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
CF_SECRETS_FILE="${CF_SECRETS_FILE:-/root/.secrets/cloudflare.ini}"

# 从 cloudflare.ini 读取 API Token
load_cf_token() {
    if [ -f "$CF_SECRETS_FILE" ]; then
        CF_API_TOKEN=$(grep -E '^dns_cloudflare_api_token\s*=' "$CF_SECRETS_FILE" | cut -d'=' -f2 | tr -d ' ')
    fi
}

# 加载配置
load_cf_token

# 加载 .env 文件 (可选覆盖)
[ -f "$PROJECT_ROOT/.env" ] && source "$PROJECT_ROOT/.env"

# Cloudflare 配置
CF_API_TOKEN="${CF_API_TOKEN:-}"
CF_ZONE_ID="${CF_ZONE_ID:-28c1ce2ce46d158d8b948acccf9300ad}"
CF_DOMAIN="${CF_DOMAIN:-ai-infra-matrix.com}"

#===============================================================================
# 函数定义
#===============================================================================

print_header() {
    echo -e "${BLUE}===============================================================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}===============================================================================${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

# 检查必要的配置
check_config() {
    local missing=0
    
    if [ -z "$CF_API_TOKEN" ]; then
        print_error "CF_API_TOKEN 未设置"
        echo "  请在 .env 文件中添加: CF_API_TOKEN=your_api_token"
        echo "  或设置环境变量: export CF_API_TOKEN=your_api_token"
        echo ""
        echo "  获取 API Token: https://dash.cloudflare.com/profile/api-tokens"
        echo "  需要的权限: Zone.Zone Settings (Edit), Zone.DNS (Edit)"
        missing=1
    fi
    
    if [ -z "$CF_ZONE_ID" ]; then
        print_error "CF_ZONE_ID 未设置"
        echo "  请在 .env 文件中添加: CF_ZONE_ID=your_zone_id"
        echo "  或设置环境变量: export CF_ZONE_ID=your_zone_id"
        echo ""
        echo "  获取 Zone ID: Cloudflare Dashboard → 域名 → Overview → 右侧 API 区域"
        missing=1
    fi
    
    if [ $missing -eq 1 ]; then
        exit 1
    fi
    
    print_success "配置检查通过"
    print_info "Domain: $CF_DOMAIN"
    print_info "Zone ID: ${CF_ZONE_ID:0:8}..."
}

# 验证 API Token
verify_token() {
    print_info "验证 API Token..."
    
    local response=$(curl -s -X GET "https://api.cloudflare.com/client/v4/user/tokens/verify" \
        -H "Authorization: Bearer $CF_API_TOKEN" \
        -H "Content-Type: application/json")
    
    local success=$(echo "$response" | jq -r '.success')
    
    if [ "$success" = "true" ]; then
        print_success "API Token 有效"
    else
        print_error "API Token 无效"
        echo "$response" | jq .
        exit 1
    fi
}

# 获取现有的 Origin Rules Ruleset
get_origin_ruleset() {
    print_info "获取现有 Origin Rules..."
    
    local response=$(curl -s -X GET "https://api.cloudflare.com/client/v4/zones/$CF_ZONE_ID/rulesets?phase=http_request_origin" \
        -H "Authorization: Bearer $CF_API_TOKEN" \
        -H "Content-Type: application/json")
    
    echo "$response"
}

# 创建或更新 Origin Rule
create_origin_rule() {
    print_header "创建 Origin Rule - 重写目标端口为 443"
    
    # 先获取现有的 ruleset
    local existing=$(get_origin_ruleset)
    local ruleset_id=$(echo "$existing" | jq -r '.result[0].id // empty')
    
    # Origin Rule 的规则内容
    local rule_json=$(cat << EOF
{
    "action": "route",
    "action_parameters": {
        "origin": {
            "port": 443
        }
    },
    "expression": "(http.host eq \"$CF_DOMAIN\")",
    "description": "Route HTTPS traffic to origin port 443 for $CF_DOMAIN",
    "enabled": true
}
EOF
)
    
    if [ -n "$ruleset_id" ]; then
        print_info "发现现有 Ruleset: $ruleset_id"
        print_info "添加新规则到现有 Ruleset..."
        
        # 添加规则到现有 ruleset
        local response=$(curl -s -X POST "https://api.cloudflare.com/client/v4/zones/$CF_ZONE_ID/rulesets/$ruleset_id/rules" \
            -H "Authorization: Bearer $CF_API_TOKEN" \
            -H "Content-Type: application/json" \
            -d "$rule_json")
        
        local success=$(echo "$response" | jq -r '.success')
        
        if [ "$success" = "true" ]; then
            print_success "Origin Rule 创建成功！"
            echo "$response" | jq '.result.rules[-1]'
        else
            print_error "创建失败"
            echo "$response" | jq .
            return 1
        fi
    else
        print_info "创建新的 Origin Rules Ruleset..."
        
        # 创建新的 ruleset
        local ruleset_json=$(cat << EOF
{
    "name": "Origin Rules for $CF_DOMAIN",
    "kind": "zone",
    "phase": "http_request_origin",
    "rules": [
        {
            "action": "route",
            "action_parameters": {
                "origin": {
                    "port": 443
                }
            },
            "expression": "(http.host eq \"$CF_DOMAIN\")",
            "description": "Route HTTPS traffic to origin port 443",
            "enabled": true
        }
    ]
}
EOF
)
        
        local response=$(curl -s -X POST "https://api.cloudflare.com/client/v4/zones/$CF_ZONE_ID/rulesets" \
            -H "Authorization: Bearer $CF_API_TOKEN" \
            -H "Content-Type: application/json" \
            -d "$ruleset_json")
        
        local success=$(echo "$response" | jq -r '.success')
        
        if [ "$success" = "true" ]; then
            print_success "Origin Rules Ruleset 创建成功！"
            echo "$response" | jq '.result'
        else
            print_error "创建失败"
            echo "$response" | jq .
            return 1
        fi
    fi
}

# 列出所有 Origin Rules
list_origin_rules() {
    print_header "当前 Origin Rules"
    
    local response=$(get_origin_ruleset)
    local success=$(echo "$response" | jq -r '.success')
    
    if [ "$success" = "true" ]; then
        local count=$(echo "$response" | jq '.result | length')
        
        if [ "$count" = "0" ]; then
            print_warning "没有 Origin Rules"
        else
            echo "$response" | jq '.result[] | {id, name, rules: [.rules[]? | {id, description, expression, enabled, action_parameters}]}'
        fi
    else
        print_error "获取失败"
        echo "$response" | jq .
    fi
}

# 删除 Origin Rule
delete_origin_rule() {
    local rule_id="$1"
    
    if [ -z "$rule_id" ]; then
        print_error "请提供 Rule ID"
        echo "用法: $0 delete <rule_id>"
        exit 1
    fi
    
    print_header "删除 Origin Rule: $rule_id"
    
    # 获取 ruleset ID
    local existing=$(get_origin_ruleset)
    local ruleset_id=$(echo "$existing" | jq -r '.result[0].id // empty')
    
    if [ -z "$ruleset_id" ]; then
        print_error "找不到 Origin Rules Ruleset"
        exit 1
    fi
    
    local response=$(curl -s -X DELETE "https://api.cloudflare.com/client/v4/zones/$CF_ZONE_ID/rulesets/$ruleset_id/rules/$rule_id" \
        -H "Authorization: Bearer $CF_API_TOKEN" \
        -H "Content-Type: application/json")
    
    local success=$(echo "$response" | jq -r '.success')
    
    if [ "$success" = "true" ]; then
        print_success "规则删除成功"
    else
        print_error "删除失败"
        echo "$response" | jq .
    fi
}

# 设置 SSL 模式
set_ssl_mode() {
    local mode="${1:-full}"
    
    print_header "设置 SSL/TLS 模式为: $mode"
    
    local response=$(curl -s -X PATCH "https://api.cloudflare.com/client/v4/zones/$CF_ZONE_ID/settings/ssl" \
        -H "Authorization: Bearer $CF_API_TOKEN" \
        -H "Content-Type: application/json" \
        -d "{\"value\": \"$mode\"}")
    
    local success=$(echo "$response" | jq -r '.success')
    
    if [ "$success" = "true" ]; then
        print_success "SSL 模式设置为: $mode"
    else
        print_error "设置失败"
        echo "$response" | jq .
    fi
}

# 获取 SSL 模式
get_ssl_mode() {
    print_info "获取当前 SSL/TLS 模式..."
    
    local response=$(curl -s -X GET "https://api.cloudflare.com/client/v4/zones/$CF_ZONE_ID/settings/ssl" \
        -H "Authorization: Bearer $CF_API_TOKEN" \
        -H "Content-Type: application/json")
    
    local mode=$(echo "$response" | jq -r '.result.value')
    print_info "当前 SSL 模式: $mode"
}

# 清除缓存
purge_cache() {
    print_header "清除 Cloudflare 缓存"
    
    local response=$(curl -s -X POST "https://api.cloudflare.com/client/v4/zones/$CF_ZONE_ID/purge_cache" \
        -H "Authorization: Bearer $CF_API_TOKEN" \
        -H "Content-Type: application/json" \
        -d '{"purge_everything": true}')
    
    local success=$(echo "$response" | jq -r '.success')
    
    if [ "$success" = "true" ]; then
        print_success "缓存已清除"
    else
        print_error "清除失败"
        echo "$response" | jq .
    fi
}

# 完整配置流程
setup_all() {
    print_header "Cloudflare 完整配置 - 修复 525 错误"
    
    check_config
    verify_token
    
    echo ""
    get_ssl_mode
    
    echo ""
    print_info "设置 SSL 模式为 Full..."
    set_ssl_mode "full"
    
    echo ""
    create_origin_rule
    
    echo ""
    purge_cache
    
    echo ""
    print_header "配置完成！"
    echo ""
    echo "请等待 1-2 分钟让配置生效，然后测试:"
    echo "  curl -sk https://$CF_DOMAIN/health"
}

# 显示帮助
show_help() {
    cat << EOF
Cloudflare Origin Rule 配置脚本

用法: $0 <命令> [参数]

命令:
  setup           完整配置流程（推荐）
  create          创建 Origin Rule (端口 443)
  list            列出所有 Origin Rules
  delete <id>     删除指定的 Origin Rule
  ssl <mode>      设置 SSL 模式 (off/flexible/full/strict)
  ssl-status      获取当前 SSL 模式
  purge           清除 Cloudflare 缓存
  verify          验证 API Token
  help            显示此帮助

环境变量:
  CF_API_TOKEN    Cloudflare API Token (必需)
  CF_ZONE_ID      Cloudflare Zone ID (必需)
  CF_DOMAIN       域名 (默认: ai-infra-matrix.com)

示例:
  # 完整配置
  CF_API_TOKEN=xxx CF_ZONE_ID=yyy $0 setup

  # 或者在 .env 文件中配置后直接运行
  $0 setup

  # 仅创建 Origin Rule
  $0 create

  # 设置 SSL 模式为 Full (Strict)
  $0 ssl strict
EOF
}

#===============================================================================
# 主程序
#===============================================================================

case "${1:-help}" in
    setup)
        setup_all
        ;;
    create)
        check_config
        verify_token
        create_origin_rule
        ;;
    list)
        check_config
        list_origin_rules
        ;;
    delete)
        check_config
        delete_origin_rule "$2"
        ;;
    ssl)
        check_config
        set_ssl_mode "${2:-full}"
        ;;
    ssl-status)
        check_config
        get_ssl_mode
        ;;
    purge)
        check_config
        purge_cache
        ;;
    verify)
        check_config
        verify_token
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        print_error "未知命令: $1"
        echo ""
        show_help
        exit 1
        ;;
esac
