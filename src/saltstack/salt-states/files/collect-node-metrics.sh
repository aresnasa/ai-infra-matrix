#!/bin/bash
#
# Node Metrics Collection Script
# 采集 GPU 驱动版本、CUDA 版本、IB 网卡状态等信息并回调到 Backend
#
# 由 Salt State 自动部署，通过 cron 定期执行
#

set -e

# 加载配置
CONFIG_FILE="/opt/ai-infra/scripts/node-metrics.conf"
if [ -f "$CONFIG_FILE" ]; then
    source "$CONFIG_FILE"
fi

# 默认值
CALLBACK_URL="${CALLBACK_URL:-http://localhost:8080/api/saltstack/node-metrics/callback}"
MINION_ID="${MINION_ID:-$(hostname)}"
API_TOKEN="${API_TOKEN:-}"

# 日志函数
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"
}

# 检查命令是否存在
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# 采集 GPU 信息
collect_gpu_info() {
    local gpu_json="{}"
    
    if command_exists nvidia-smi; then
        log "Collecting GPU information..."
        
        # 获取驱动版本
        local driver_version
        driver_version=$(nvidia-smi --query-gpu=driver_version --format=csv,noheader,nounits 2>/dev/null | head -1 | tr -d ' ' || echo "")
        
        # 获取 CUDA 版本
        local cuda_version
        cuda_version=$(nvidia-smi 2>/dev/null | grep -oP 'CUDA Version: \K[0-9.]+' | head -1 || echo "")
        
        # 获取 GPU 数量
        local gpu_count
        gpu_count=$(nvidia-smi -L 2>/dev/null | wc -l || echo "0")
        
        # 获取 GPU 型号
        local gpu_model
        gpu_model=$(nvidia-smi --query-gpu=name --format=csv,noheader 2>/dev/null | head -1 | sed 's/,//g' || echo "")
        
        # 获取 GPU 总显存
        local memory_total
        memory_total=$(nvidia-smi --query-gpu=memory.total --format=csv,noheader 2>/dev/null | head -1 || echo "")
        
        # 获取每个 GPU 的详细信息
        local gpus_detail="[]"
        if [ "$gpu_count" -gt 0 ]; then
            gpus_detail="["
            local first=true
            while IFS=',' read -r idx uuid name mem_total mem_used mem_free temp power_draw power_limit util; do
                if [ "$first" = true ]; then
                    first=false
                else
                    gpus_detail+=","
                fi
                # 清理空格
                idx=$(echo "$idx" | tr -d ' ')
                uuid=$(echo "$uuid" | tr -d ' ')
                name=$(echo "$name" | xargs)
                mem_total=$(echo "$mem_total" | tr -d ' ')
                mem_used=$(echo "$mem_used" | tr -d ' ')
                mem_free=$(echo "$mem_free" | tr -d ' ')
                temp=$(echo "$temp" | tr -d ' ')
                power_draw=$(echo "$power_draw" | tr -d ' ')
                power_limit=$(echo "$power_limit" | tr -d ' ')
                util=$(echo "$util" | tr -d ' %')
                
                gpus_detail+="{\"index\":$idx,\"uuid\":\"$uuid\",\"name\":\"$name\",\"memory_total\":\"$mem_total\",\"memory_used\":\"$mem_used\",\"memory_free\":\"$mem_free\",\"temperature\":${temp:-0},\"power_draw\":\"$power_draw\",\"power_limit\":\"$power_limit\",\"utilization\":${util:-0}}"
            done < <(nvidia-smi --query-gpu=index,uuid,name,memory.total,memory.used,memory.free,temperature.gpu,power.draw,power.limit,utilization.gpu --format=csv,noheader 2>/dev/null || echo "")
            gpus_detail+="]"
        fi
        
        gpu_json=$(cat <<EOF
{
    "driver_version": "$driver_version",
    "cuda_version": "$cuda_version",
    "count": $gpu_count,
    "model": "$gpu_model",
    "memory_total": "$memory_total",
    "gpus": $gpus_detail
}
EOF
)
        log "GPU: driver=$driver_version, cuda=$cuda_version, count=$gpu_count, model=$gpu_model"
    else
        log "nvidia-smi not found, skipping GPU collection"
        gpu_json="null"
    fi
    
    echo "$gpu_json"
}

# 采集 InfiniBand 信息
collect_ib_info() {
    local ib_json="{}"
    
    if command_exists ibstat; then
        log "Collecting InfiniBand information..."
        
        local active_count=0
        local ports_json="["
        local first=true
        
        # 解析 ibstat 输出
        local current_port=""
        local current_state=""
        local current_rate=""
        local current_guid=""
        
        while IFS= read -r line; do
            # 检测端口名称行 (例如: CA 'mlx5_0')
            if [[ "$line" =~ ^CA\ \'([^\']+)\' ]]; then
                # 保存上一个端口的信息
                if [ -n "$current_port" ]; then
                    if [ "$first" = true ]; then
                        first=false
                    else
                        ports_json+=","
                    fi
                    ports_json+="{\"name\":\"$current_port\",\"state\":\"$current_state\",\"rate\":\"$current_rate\",\"guid\":\"$current_guid\"}"
                    
                    if [ "$current_state" = "Active" ]; then
                        ((active_count++)) || true
                    fi
                fi
                current_port="${BASH_REMATCH[1]}"
                current_state=""
                current_rate=""
                current_guid=""
            fi
            
            # 解析状态
            if [[ "$line" =~ State:\ *([A-Za-z]+) ]]; then
                current_state="${BASH_REMATCH[1]}"
            fi
            
            # 解析速率
            if [[ "$line" =~ Rate:\ *([0-9]+) ]]; then
                current_rate="${BASH_REMATCH[1]} Gb/sec"
            fi
            
            # 解析 GUID
            if [[ "$line" =~ Port\ GUID:\ *([^ ]+) ]]; then
                current_guid="${BASH_REMATCH[1]}"
            fi
            
        done < <(ibstat 2>/dev/null || echo "")
        
        # 保存最后一个端口
        if [ -n "$current_port" ]; then
            if [ "$first" = true ]; then
                first=false
            else
                ports_json+=","
            fi
            ports_json+="{\"name\":\"$current_port\",\"state\":\"$current_state\",\"rate\":\"$current_rate\",\"guid\":\"$current_guid\"}"
            
            if [ "$current_state" = "Active" ]; then
                ((active_count++)) || true
            fi
        fi
        
        ports_json+="]"
        
        ib_json=$(cat <<EOF
{
    "active_count": $active_count,
    "ports": $ports_json
}
EOF
)
        log "InfiniBand: active_count=$active_count"
    else
        log "ibstat not found, skipping IB collection"
        ib_json="null"
    fi
    
    echo "$ib_json"
}

# 采集系统信息
collect_system_info() {
    log "Collecting system information..."
    
    local kernel_version
    kernel_version=$(uname -r 2>/dev/null || echo "unknown")
    
    local os_version=""
    if [ -f /etc/os-release ]; then
        os_version=$(grep -oP 'PRETTY_NAME="\K[^"]+' /etc/os-release 2>/dev/null || echo "")
    fi
    
    local hostname
    hostname=$(hostname 2>/dev/null || echo "unknown")
    
    local uptime_str
    uptime_str=$(uptime -p 2>/dev/null || uptime 2>/dev/null | awk '{print $3,$4,$5}' || echo "unknown")
    
    cat <<EOF
{
    "kernel_version": "$kernel_version",
    "os_version": "$os_version",
    "hostname": "$hostname",
    "uptime": "$uptime_str"
}
EOF
}

# 发送数据到 Backend
send_metrics() {
    local gpu_info="$1"
    local ib_info="$2"
    local system_info="$3"
    
    local timestamp
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    
    # 构建 JSON payload
    local payload
    payload=$(cat <<EOF
{
    "minion_id": "$MINION_ID",
    "timestamp": "$timestamp",
    "gpu": $gpu_info,
    "ib": $ib_info,
    "system": $system_info
}
EOF
)
    
    log "Sending metrics to $CALLBACK_URL"
    
    # 构建 curl 命令
    local curl_opts="-s -X POST -H 'Content-Type: application/json'"
    if [ -n "$API_TOKEN" ]; then
        curl_opts="$curl_opts -H 'X-API-Token: $API_TOKEN'"
    fi
    
    # 发送请求
    local response
    local http_code
    
    if [ -n "$API_TOKEN" ]; then
        response=$(curl -s -w "\n%{http_code}" -X POST \
            -H "Content-Type: application/json" \
            -H "X-API-Token: $API_TOKEN" \
            -d "$payload" \
            --connect-timeout 10 \
            --max-time 30 \
            "$CALLBACK_URL" 2>/dev/null || echo -e "\n000")
    else
        response=$(curl -s -w "\n%{http_code}" -X POST \
            -H "Content-Type: application/json" \
            -d "$payload" \
            --connect-timeout 10 \
            --max-time 30 \
            "$CALLBACK_URL" 2>/dev/null || echo -e "\n000")
    fi
    
    http_code=$(echo "$response" | tail -1)
    response=$(echo "$response" | head -n -1)
    
    if [ "$http_code" = "200" ]; then
        log "Metrics sent successfully: $response"
        return 0
    else
        log "Failed to send metrics: HTTP $http_code - $response"
        return 1
    fi
}

# 主函数
main() {
    log "=========================================="
    log "Starting node metrics collection..."
    log "Minion ID: $MINION_ID"
    log "Callback URL: $CALLBACK_URL"
    log "=========================================="
    
    # 采集各类信息
    local gpu_info
    gpu_info=$(collect_gpu_info)
    
    local ib_info
    ib_info=$(collect_ib_info)
    
    local system_info
    system_info=$(collect_system_info)
    
    # 发送到 Backend
    if send_metrics "$gpu_info" "$ib_info" "$system_info"; then
        log "Collection completed successfully"
    else
        log "Collection completed with errors"
        exit 1
    fi
}

# 执行主函数
main "$@"
