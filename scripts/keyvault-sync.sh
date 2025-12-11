#!/bin/bash
# =============================================================================
# KeyVault 密钥同步客户端脚本
# 使用一次性令牌安全地从 AI-Infra-Matrix Backend 同步密钥
# =============================================================================
# 使用方式:
#   1. 首先通过 API 获取一次性同步令牌
#   2. 将令牌传递给此脚本进行密钥同步
#
# 环境变量:
#   KEYVAULT_URL     - KeyVault API URL (例如: http://192.168.0.200:8082/api/keyvault)
#   SYNC_TOKEN       - 一次性同步令牌
#   KEY_NAME         - 要同步的密钥名称
#   OUTPUT_FILE      - 输出文件路径 (可选，默认输出到标准输出)
#   KEY_PERMISSION   - 输出文件权限 (可选，默认 0600)
# =============================================================================

set -euo pipefail

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# 显示帮助
show_help() {
    cat << EOF
KeyVault 密钥同步客户端

使用方式:
  $0 [选项]

选项:
  -u, --url URL           KeyVault API URL
  -t, --token TOKEN       一次性同步令牌
  -k, --key-name NAME     要同步的密钥名称
  -o, --output FILE       输出文件路径 (可选)
  -p, --permission PERM   输出文件权限 (默认: 0600)
  -b, --batch NAMES       批量同步，逗号分隔的密钥名称
  -d, --output-dir DIR    批量同步时的输出目录
  -h, --help              显示此帮助信息

示例:
  # 同步单个密钥到标准输出
  $0 -u http://backend:8082/api/keyvault -t TOKEN123 -k salt_master_public

  # 同步单个密钥到文件
  $0 -u http://backend:8082/api/keyvault -t TOKEN123 -k salt_master_public -o /etc/salt/pki/master/master.pub

  # 批量同步多个密钥
  $0 -u http://backend:8082/api/keyvault -t TOKEN123 -b "salt_master_public,salt_master_private" -d /etc/salt/pki/master/
EOF
}

# 解析命令行参数
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -u|--url)
                KEYVAULT_URL="$2"
                shift 2
                ;;
            -t|--token)
                SYNC_TOKEN="$2"
                shift 2
                ;;
            -k|--key-name)
                KEY_NAME="$2"
                shift 2
                ;;
            -o|--output)
                OUTPUT_FILE="$2"
                shift 2
                ;;
            -p|--permission)
                KEY_PERMISSION="$2"
                shift 2
                ;;
            -b|--batch)
                BATCH_KEYS="$2"
                shift 2
                ;;
            -d|--output-dir)
                OUTPUT_DIR="$2"
                shift 2
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            *)
                log_error "未知选项: $1"
                show_help
                exit 1
                ;;
        esac
    done
}

# 验证必要参数
validate_params() {
    if [[ -z "${KEYVAULT_URL:-}" ]]; then
        log_error "缺少 KeyVault URL，使用 -u 或设置 KEYVAULT_URL 环境变量"
        exit 1
    fi

    if [[ -z "${SYNC_TOKEN:-}" ]]; then
        log_error "缺少同步令牌，使用 -t 或设置 SYNC_TOKEN 环境变量"
        exit 1
    fi

    if [[ -z "${KEY_NAME:-}" ]] && [[ -z "${BATCH_KEYS:-}" ]]; then
        log_error "缺少密钥名称，使用 -k 指定单个密钥或 -b 指定批量密钥"
        exit 1
    fi
}

# 同步单个密钥
sync_single_key() {
    local key_name="$1"
    local output_file="${2:-}"
    local permission="${3:-0600}"

    log_info "正在同步密钥: $key_name"

    # 构建请求
    local response
    response=$(curl -sS -X POST "${KEYVAULT_URL}/sync" \
        -H "Content-Type: application/json" \
        -d "{\"sync_token\": \"${SYNC_TOKEN}\", \"key_name\": \"${key_name}\"}" \
        2>&1) || {
        log_error "API 请求失败"
        return 1
    }

    # 检查响应是否包含错误
    if echo "$response" | grep -q '"error"'; then
        local error_msg
        error_msg=$(echo "$response" | grep -o '"error":"[^"]*"' | cut -d'"' -f4)
        log_error "同步失败: $error_msg"
        return 1
    fi

    # 提取并解码密钥数据
    local key_data_base64
    key_data_base64=$(echo "$response" | grep -o '"key_data":"[^"]*"' | cut -d'"' -f4)
    
    if [[ -z "$key_data_base64" ]]; then
        log_error "响应中未找到密钥数据"
        return 1
    fi

    # Base64 解码
    local key_data
    key_data=$(echo "$key_data_base64" | base64 -d 2>/dev/null) || {
        log_error "Base64 解码失败"
        return 1
    }

    # 输出到文件或标准输出
    if [[ -n "$output_file" ]]; then
        # 确保目录存在
        mkdir -p "$(dirname "$output_file")"
        
        # 写入文件
        echo "$key_data" > "$output_file"
        chmod "$permission" "$output_file"
        
        log_success "密钥已保存到: $output_file (权限: $permission)"
    else
        echo "$key_data"
    fi

    return 0
}

# 批量同步密钥
sync_batch_keys() {
    local keys="$1"
    local output_dir="${2:-}"
    local permission="${3:-0600}"

    # 将逗号分隔的密钥名称转换为数组
    IFS=',' read -ra KEY_ARRAY <<< "$keys"

    log_info "正在批量同步 ${#KEY_ARRAY[@]} 个密钥"

    # 构建 JSON 数组
    local key_names_json="["
    for i in "${!KEY_ARRAY[@]}"; do
        if [[ $i -gt 0 ]]; then
            key_names_json+=","
        fi
        key_names_json+="\"${KEY_ARRAY[$i]}\""
    done
    key_names_json+="]"

    # 发送批量请求
    local response
    response=$(curl -sS -X POST "${KEYVAULT_URL}/sync/batch" \
        -H "Content-Type: application/json" \
        -d "{\"sync_token\": \"${SYNC_TOKEN}\", \"key_names\": ${key_names_json}}" \
        2>&1) || {
        log_error "批量 API 请求失败"
        return 1
    }

    # 检查响应是否包含错误
    if echo "$response" | grep -q '"error"'; then
        local error_msg
        error_msg=$(echo "$response" | grep -o '"error":"[^"]*"' | cut -d'"' -f4)
        log_error "批量同步失败: $error_msg"
        return 1
    fi

    # 解析响应并保存各个密钥
    # 注意：这里需要 jq 来解析 JSON
    if ! command -v jq &> /dev/null; then
        log_error "需要安装 jq 来处理批量同步响应"
        log_info "尝试安装 jq: apt-get install jq 或 yum install jq"
        return 1
    fi

    local total
    total=$(echo "$response" | jq -r '.total')
    log_info "成功获取 $total 个密钥"

    # 遍历并保存每个密钥
    echo "$response" | jq -r '.keys | to_entries[] | "\(.key)|\(.value)"' | while IFS='|' read -r key_name key_data_base64; do
        if [[ -n "$output_dir" ]]; then
            local output_file="${output_dir}/${key_name}"
            mkdir -p "$output_dir"
            
            echo "$key_data_base64" | base64 -d > "$output_file"
            chmod "$permission" "$output_file"
            
            log_success "密钥 $key_name 已保存到: $output_file"
        else
            echo "=== $key_name ==="
            echo "$key_data_base64" | base64 -d
            echo ""
        fi
    done

    return 0
}

# 主函数
main() {
    # 解析参数
    parse_args "$@"

    # 验证参数
    validate_params

    # 设置默认权限
    KEY_PERMISSION="${KEY_PERMISSION:-0600}"

    # 执行同步
    if [[ -n "${BATCH_KEYS:-}" ]]; then
        sync_batch_keys "$BATCH_KEYS" "${OUTPUT_DIR:-}" "$KEY_PERMISSION"
    else
        sync_single_key "$KEY_NAME" "${OUTPUT_FILE:-}" "$KEY_PERMISSION"
    fi
}

# 执行主函数
main "$@"
