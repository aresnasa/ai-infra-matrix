#!/bin/bash
# =============================================================================
# Git 用户切换脚本
# 根据 git remote origin 自动切换不同的用户身份
# =============================================================================

set -e

# =============================================================================
# 配置区域 - 在这里配置你的不同仓库和对应的用户信息
# =============================================================================

# 配置格式: "origin_pattern|user_name|user_email"
# origin_pattern: 匹配 remote origin URL 的正则表达式
declare -a GIT_USER_CONFIGS=(
    # 示例配置 - 请根据实际情况修改
    "github.com/aresnasa|aresnasa|aresnasa@126.com"
    "gitlab.zs.shaipower.online|xuchao|your.name@company.com"
    "gitee.com|Wolverinexu|aresnasa@126.com"
    # 添加更多配置...
)

# 默认用户（当没有匹配到任何配置时使用）
DEFAULT_USER_NAME="Default User"
DEFAULT_USER_EMAIL="default@example.com"

# =============================================================================
# 函数定义
# =============================================================================

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 显示当前 git 用户配置
show_current_config() {
    local current_name=$(git config user.name 2>/dev/null || echo "未设置")
    local current_email=$(git config user.email 2>/dev/null || echo "未设置")
    local origin_url=$(git remote get-url origin 2>/dev/null || echo "未设置")
    
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}📋 当前 Git 配置${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo "  Remote Origin: $origin_url"
    echo "  User Name:     $current_name"
    echo "  User Email:    $current_email"
    echo ""
}

# 列出所有配置的用户
list_configs() {
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}📝 已配置的用户列表${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    local idx=1
    for config in "${GIT_USER_CONFIGS[@]}"; do
        IFS='|' read -r pattern name email <<< "$config"
        echo "  [$idx] Pattern: $pattern"
        echo "       Name:    $name"
        echo "       Email:   $email"
        echo ""
        idx=$((idx + 1))
    done
    
    echo "  [默认] Name:  $DEFAULT_USER_NAME"
    echo "         Email: $DEFAULT_USER_EMAIL"
    echo ""
}

# 根据 origin 自动切换用户
auto_switch() {
    local origin_url=$(git remote get-url origin 2>/dev/null)
    
    if [[ -z "$origin_url" ]]; then
        log_error "当前目录不是 git 仓库或没有设置 origin"
        return 1
    fi
    
    log_info "检测到 Origin: $origin_url"
    
    # 遍历配置，查找匹配项
    for config in "${GIT_USER_CONFIGS[@]}"; do
        IFS='|' read -r pattern name email <<< "$config"
        
        if echo "$origin_url" | grep -qE "$pattern"; then
            log_info "匹配配置: $pattern"
            git config user.name "$name"
            git config user.email "$email"
            log_info "✓ 已切换到: $name <$email>"
            return 0
        fi
    done
    
    # 没有匹配，使用默认配置
    log_warn "未找到匹配的配置，使用默认用户"
    git config user.name "$DEFAULT_USER_NAME"
    git config user.email "$DEFAULT_USER_EMAIL"
    log_info "✓ 已切换到默认用户: $DEFAULT_USER_NAME <$DEFAULT_USER_EMAIL>"
}

# 手动设置用户
manual_set() {
    local name="$1"
    local email="$2"
    
    if [[ -z "$name" ]] || [[ -z "$email" ]]; then
        log_error "用法: $0 set <name> <email>"
        return 1
    fi
    
    git config user.name "$name"
    git config user.email "$email"
    log_info "✓ 已设置用户: $name <$email>"
}

# 交互式选择用户
interactive_select() {
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BLUE}🔄 选择要切换的用户${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    local idx=1
    local options=()
    
    for config in "${GIT_USER_CONFIGS[@]}"; do
        IFS='|' read -r pattern name email <<< "$config"
        echo "  [$idx] $name <$email>"
        options+=("$config")
        idx=$((idx + 1))
    done
    
    echo "  [$idx] 默认: $DEFAULT_USER_NAME <$DEFAULT_USER_EMAIL>"
    echo "  [0] 取消"
    echo ""
    
    read -p "请选择 (0-$idx): " choice
    
    if [[ "$choice" == "0" ]]; then
        log_info "已取消"
        return 0
    fi
    
    if [[ "$choice" == "$idx" ]]; then
        # 选择默认用户
        git config user.name "$DEFAULT_USER_NAME"
        git config user.email "$DEFAULT_USER_EMAIL"
        log_info "✓ 已切换到默认用户"
        return 0
    fi
    
    if [[ "$choice" -ge 1 ]] && [[ "$choice" -lt "$idx" ]]; then
        local selected="${options[$((choice - 1))]}"
        IFS='|' read -r pattern name email <<< "$selected"
        git config user.name "$name"
        git config user.email "$email"
        log_info "✓ 已切换到: $name <$email>"
        return 0
    fi
    
    log_error "无效的选择"
    return 1
}

# 为所有仓库设置 git hooks（自动切换）
setup_hook() {
    local hook_dir=".git/hooks"
    local hook_file="$hook_dir/post-checkout"
    
    if [[ ! -d ".git" ]]; then
        log_error "当前目录不是 git 仓库"
        return 1
    fi
    
    mkdir -p "$hook_dir"
    
    cat > "$hook_file" << 'HOOK'
#!/bin/bash
# 自动切换 git 用户的 hook
# 由 git-switch-user.sh 创建

SCRIPT_PATH="$(dirname "$(readlink -f "$0")")/../../scripts/git-switch-user.sh"

if [[ -f "$SCRIPT_PATH" ]]; then
    "$SCRIPT_PATH" auto
fi
HOOK

    chmod +x "$hook_file"
    log_info "✓ 已安装 post-checkout hook"
    log_info "  每次 checkout 后会自动切换用户"
}

# 显示帮助
show_help() {
    echo "用法: $0 [命令] [参数]"
    echo ""
    echo "命令:"
    echo "  auto          根据 origin 自动切换用户（默认）"
    echo "  show          显示当前配置"
    echo "  list          列出所有配置的用户"
    echo "  select        交互式选择用户"
    echo "  set <n> <e>   手动设置用户 (name, email)"
    echo "  hook          安装 git hook 实现自动切换"
    echo "  help          显示此帮助"
    echo ""
    echo "示例:"
    echo "  $0 auto           # 自动切换"
    echo "  $0 select         # 交互式选择"
    echo "  $0 set 'John' 'john@example.com'"
    echo ""
    echo "配置说明:"
    echo "  编辑此脚本顶部的 GIT_USER_CONFIGS 数组来添加配置"
    echo "  格式: \"origin_pattern|user_name|user_email\""
}

# =============================================================================
# 主入口
# =============================================================================

main() {
    local cmd="${1:-auto}"
    
    case "$cmd" in
        auto)
            auto_switch
            ;;
        show)
            show_current_config
            ;;
        list)
            list_configs
            ;;
        select)
            interactive_select
            ;;
        set)
            manual_set "$2" "$3"
            ;;
        hook)
            setup_hook
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            log_error "未知命令: $cmd"
            show_help
            return 1
            ;;
    esac
}

main "$@"
