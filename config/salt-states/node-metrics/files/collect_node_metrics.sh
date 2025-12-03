#!/bin/bash
# 节点指标采集脚本
# 采集 GPU 驱动版本、IB 状态等信息并回调到 Salt Master
# 每 3 分钟执行一次

set -e

SCRIPT_DIR="$(dirname "$(readlink -f "$0")")"
CONFIG_FILE="${SCRIPT_DIR}/callback.conf"
LOG_PREFIX="[$(date '+%Y-%m-%d %H:%M:%S')] [node-metrics]"

# 读取配置
if [ -f "$CONFIG_FILE" ]; then
    source "$CONFIG_FILE"
fi

# 默认回调地址
CALLBACK_URL="${CALLBACK_URL:-http://ai-infra-matrix:8080/api/saltstack/metrics/callback}"

# 获取 minion_id
get_minion_id() {
    if [ -f /etc/salt/minion_id ]; then
        cat /etc/salt/minion_id
    else
        hostname -f 2>/dev/null || hostname
    fi
}

# 获取 GPU 信息
collect_gpu_info() {
    local gpu_info="{}"
    
    # 检查 nvidia-smi 是否存在
    if command -v nvidia-smi &> /dev/null; then
        # 获取驱动版本
        local driver_version=$(nvidia-smi --query-gpu=driver_version --format=csv,noheader,nounits 2>/dev/null | head -1 | tr -d ' ')
        
        # 获取 CUDA 版本
        local cuda_version=$(nvidia-smi 2>/dev/null | grep -oP 'CUDA Version: \K[0-9.]+' | head -1)
        
        # 获取 GPU 数量
        local gpu_count=$(nvidia-smi -L 2>/dev/null | wc -l)
        
        # 获取 GPU 型号
        local gpu_model=$(nvidia-smi --query-gpu=name --format=csv,noheader 2>/dev/null | head -1 | tr -d '\n')
        
        # 获取 GPU 显存总量
        local mem_total=$(nvidia-smi --query-gpu=memory.total --format=csv,noheader 2>/dev/null | head -1 | tr -d ' ')
        
        # 构建 JSON
        gpu_info=$(cat <<EOF
{
    "driver_version": "${driver_version:-}",
    "cuda_version": "${cuda_version:-}",
    "count": ${gpu_count:-0},
    "model": "${gpu_model:-}",
    "memory_total": "${mem_total:-}"
}
EOF
)
    else
        gpu_info='null'
    fi
    
    echo "$gpu_info"
}

# 获取 IB 信息
collect_ib_info() {
    local ib_info='null'
    
    # 检查 ibstat 是否存在
    if command -v ibstat &> /dev/null; then
        local ports=()
        local active_count=0
        local current_ca=""
        local current_port=""
        local current_state=""
        local current_rate=""
        local current_firmware=""
        local current_guid=""
        
        # 解析 ibstat 输出
        while IFS= read -r line; do
            # 检测 CA 名称，例如: CA 'mlx5_0'
            if [[ "$line" =~ ^CA\ \'([^\']+)\' ]]; then
                current_ca="${BASH_REMATCH[1]}"
            fi
            
            # 检测端口号，例如: Port 1:
            if [[ "$line" =~ ^[[:space:]]*Port\ ([0-9]+): ]]; then
                current_port="${BASH_REMATCH[1]}"
            fi
            
            # 检测状态，例如: State: Active
            if [[ "$line" =~ ^[[:space:]]*State:\ (.+) ]]; then
                current_state="${BASH_REMATCH[1]}"
            fi
            
            # 检测速率，例如: Rate: 400
            if [[ "$line" =~ ^[[:space:]]*Rate:\ (.+) ]]; then
                current_rate="${BASH_REMATCH[1]}"
            fi
            
            # 检测固件版本
            if [[ "$line" =~ ^[[:space:]]*Firmware\ version:\ (.+) ]]; then
                current_firmware="${BASH_REMATCH[1]}"
            fi
            
            # 检测 Port GUID
            if [[ "$line" =~ ^[[:space:]]*Port\ GUID:\ (.+) ]]; then
                current_guid="${BASH_REMATCH[1]}"
                
                # 一个完整的端口信息收集完毕，只保存 Active 状态的端口
                if [ -n "$current_ca" ] && [ "$current_state" == "Active" ]; then
                    local port_json=$(cat <<EOF
{
    "name": "${current_ca}",
    "state": "${current_state}",
    "rate": "${current_rate}",
    "firmware": "${current_firmware}",
    "guid": "${current_guid}"
}
EOF
)
                    ports+=("$port_json")
                    ((active_count++)) || true
                fi
            fi
        done < <(ibstat 2>/dev/null)
        
        # 构建端口数组 JSON
        local ports_json="["
        local first=true
        for port in "${ports[@]}"; do
            if [ "$first" = true ]; then
                first=false
            else
                ports_json+=","
            fi
            ports_json+="$port"
        done
        ports_json+="]"
        
        if [ ${#ports[@]} -gt 0 ]; then
            ib_info=$(cat <<EOF
{
    "active_count": ${active_count},
    "ports": ${ports_json}
}
EOF
)
        fi
    fi
    
    echo "$ib_info"
}

# 获取系统基本信息
collect_system_info() {
    local hostname=$(hostname -f 2>/dev/null || hostname)
    local kernel=$(uname -r)
    local os_info=""
    
    if [ -f /etc/os-release ]; then
        os_info=$(. /etc/os-release && echo "$NAME $VERSION_ID")
    fi
    
    # CPU 使用率
    local cpu_usage=$(top -bn1 | grep "Cpu(s)" | awk '{print $2}' | cut -d'%' -f1 2>/dev/null || echo "0")
    
    # 内存使用率
    local mem_info=$(free -m | awk 'NR==2{printf "%.1f", $3*100/$2}' 2>/dev/null || echo "0")
    
    cat <<EOF
{
    "hostname": "${hostname}",
    "kernel": "${kernel}",
    "os": "${os_info}",
    "cpu_usage": ${cpu_usage:-0},
    "memory_usage": ${mem_info:-0}
}
EOF
}

# 主函数
main() {
    local minion_id=$(get_minion_id)
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    
    echo "$LOG_PREFIX Starting metrics collection for ${minion_id}"
    
    # 采集各项指标
    local gpu_info=$(collect_gpu_info)
    local ib_info=$(collect_ib_info)
    local system_info=$(collect_system_info)
    
    # 构建完整的 JSON payload
    local payload=$(cat <<EOF
{
    "minion_id": "${minion_id}",
    "timestamp": "${timestamp}",
    "gpu": ${gpu_info},
    "ib": ${ib_info},
    "system": ${system_info}
}
EOF
)
    
    echo "$LOG_PREFIX Payload: $payload"
    
    # 回调到服务器
    if [ -n "$CALLBACK_URL" ]; then
        echo "$LOG_PREFIX Sending metrics to ${CALLBACK_URL}"
        
        # 构建 curl 命令，支持可选的 API Token
        local curl_cmd="curl -s -w '\n%{http_code}' -X POST -H 'Content-Type: application/json'"
        if [ -n "$API_TOKEN" ]; then
            curl_cmd="$curl_cmd -H 'X-API-Token: $API_TOKEN'"
        fi
        
        local response=$(eval "$curl_cmd -d '$payload' '$CALLBACK_URL'" 2>/dev/null || echo "error")
        
        local http_code=$(echo "$response" | tail -n1)
        local body=$(echo "$response" | sed '$d')
        
        if [ "$http_code" == "200" ]; then
            echo "$LOG_PREFIX Metrics sent successfully"
        else
            echo "$LOG_PREFIX Failed to send metrics: HTTP $http_code - $body"
        fi
    else
        echo "$LOG_PREFIX No callback URL configured, skipping send"
    fi
    
    echo "$LOG_PREFIX Collection completed"
}

# 执行主函数
main "$@"
