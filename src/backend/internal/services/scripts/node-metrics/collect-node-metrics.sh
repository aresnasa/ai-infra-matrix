#!/bin/bash
#
# Node Metrics Collection Script
# 采集 CPU、内存、网络带宽、GPU 利用率/显存、IB 网卡状态、RoCE 等信息并回调到 Backend
#
# 由 Salt State 或批量安装自动部署，通过 cron 定期执行
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

# 采集 CPU 信息
collect_cpu_info() {
    log "Collecting CPU information..."
    
    local cpu_cores
    cpu_cores=$(nproc 2>/dev/null || grep -c ^processor /proc/cpuinfo 2>/dev/null || echo "0")
    
    local cpu_model
    cpu_model=$(grep -m1 'model name' /proc/cpuinfo 2>/dev/null | cut -d: -f2 | xargs || echo "unknown")
    
    # CPU 使用率
    local cpu_usage="0"
    if command_exists mpstat; then
        cpu_usage=$(mpstat 1 1 2>/dev/null | awk '/Average/ && $NF ~ /[0-9.]+/ {print 100 - $NF}' || echo "0")
    elif [ -f /proc/stat ]; then
        local cpu1 cpu2
        cpu1=$(grep '^cpu ' /proc/stat)
        sleep 0.5
        cpu2=$(grep '^cpu ' /proc/stat)
        cpu_usage=$(awk -v c1="$cpu1" -v c2="$cpu2" 'BEGIN {
            split(c1, a, " "); split(c2, b, " ");
            idle1=a[5]; idle2=b[5];
            total1=a[2]+a[3]+a[4]+a[5]+a[6]+a[7]+a[8];
            total2=b[2]+b[3]+b[4]+b[5]+b[6]+b[7]+b[8];
            printf "%.1f", 100 * (1 - (idle2-idle1)/(total2-total1));
        }' || echo "0")
    fi
    
    # CPU 负载
    local load_avg
    load_avg=$(cat /proc/loadavg 2>/dev/null | awk '{print $1","$2","$3}' || echo "0,0,0")
    
    cat <<EOF
{
    "cores": $cpu_cores,
    "model": "$cpu_model",
    "usage_percent": ${cpu_usage:-0},
    "load_avg": "$load_avg"
}
EOF
}

# 采集内存信息
collect_memory_info() {
    log "Collecting Memory information..."
    
    local mem_total mem_available mem_used mem_percent
    if [ -f /proc/meminfo ]; then
        mem_total=$(grep MemTotal /proc/meminfo | awk '{print $2}')
        mem_available=$(grep MemAvailable /proc/meminfo | awk '{print $2}')
        if [ -z "$mem_available" ]; then
            local mem_free=$(grep MemFree /proc/meminfo | awk '{print $2}')
            local buffers=$(grep Buffers /proc/meminfo | awk '{print $2}')
            local cached=$(grep "^Cached:" /proc/meminfo | awk '{print $2}')
            mem_available=$((mem_free + buffers + cached))
        fi
        mem_used=$((mem_total - mem_available))
        mem_percent=$(awk "BEGIN {printf \"%.1f\", $mem_used * 100 / $mem_total}")
        
        mem_total_gb=$(awk "BEGIN {printf \"%.2f\", $mem_total / 1024 / 1024}")
        mem_used_gb=$(awk "BEGIN {printf \"%.2f\", $mem_used / 1024 / 1024}")
        mem_available_gb=$(awk "BEGIN {printf \"%.2f\", $mem_available / 1024 / 1024}")
    else
        mem_total_gb="0"
        mem_used_gb="0"
        mem_available_gb="0"
        mem_percent="0"
    fi
    
    cat <<EOF
{
    "total_gb": $mem_total_gb,
    "used_gb": $mem_used_gb,
    "available_gb": $mem_available_gb,
    "usage_percent": $mem_percent
}
EOF
}

# 采集网络带宽信息
collect_network_info() {
    log "Collecting Network information..."
    
    local interfaces_json="["
    local first=true
    
    for iface in $(ls /sys/class/net/ 2>/dev/null | grep -v "^lo$"); do
        if [ -d "/sys/class/net/$iface" ]; then
            local rx_bytes1=$(cat /sys/class/net/$iface/statistics/rx_bytes 2>/dev/null || echo 0)
            local tx_bytes1=$(cat /sys/class/net/$iface/statistics/tx_bytes 2>/dev/null || echo 0)
            
            sleep 0.5
            
            local rx_bytes2=$(cat /sys/class/net/$iface/statistics/rx_bytes 2>/dev/null || echo 0)
            local tx_bytes2=$(cat /sys/class/net/$iface/statistics/tx_bytes 2>/dev/null || echo 0)
            
            local rx_rate=$(( (rx_bytes2 - rx_bytes1) * 2 ))
            local tx_rate=$(( (tx_bytes2 - tx_bytes1) * 2 ))
            
            local state=$(cat /sys/class/net/$iface/operstate 2>/dev/null || echo "unknown")
            
            local ip_addr=""
            if command_exists ip; then
                ip_addr=$(ip addr show $iface 2>/dev/null | grep 'inet ' | head -1 | awk '{print $2}' | cut -d/ -f1 || echo "")
            fi
            
            local speed="0"
            if [ -f "/sys/class/net/$iface/speed" ]; then
                speed=$(cat /sys/class/net/$iface/speed 2>/dev/null || echo "0")
                [ "$speed" = "-1" ] && speed="0"
            fi
            
            if [ "$first" = true ]; then
                first=false
            else
                interfaces_json+=","
            fi
            
            interfaces_json+="{\"name\":\"$iface\",\"state\":\"$state\",\"ip\":\"$ip_addr\",\"speed_mbps\":$speed,\"rx_bytes_per_sec\":$rx_rate,\"tx_bytes_per_sec\":$tx_rate}"
        fi
    done
    
    interfaces_json+="]"
    
    cat <<EOF
{
    "interfaces": $interfaces_json
}
EOF
}

# 采集 GPU 信息
collect_gpu_info() {
    log "Collecting GPU information..."
    
    if command_exists nvidia-smi; then
        local drv=$(nvidia-smi --query-gpu=driver_version --format=csv,noheader,nounits 2>/dev/null | head -1 | tr -d ' ')
        local cuda=$(nvidia-smi 2>/dev/null | grep -oP 'CUDA Version: \K[0-9.]+' | head -1)
        local cnt=$(nvidia-smi -L 2>/dev/null | wc -l)
        local model=$(nvidia-smi --query-gpu=name --format=csv,noheader 2>/dev/null | head -1 | sed 's/,//g')
        local mem=$(nvidia-smi --query-gpu=memory.total --format=csv,noheader 2>/dev/null | head -1)
        
        # 采集每个 GPU 的利用率和显存
        local gpus_json="["
        local first=true
        local i=0
        while read -r line; do
            local util=$(nvidia-smi -i $i --query-gpu=utilization.gpu --format=csv,noheader,nounits 2>/dev/null | tr -d ' ' || echo "0")
            local mem_used=$(nvidia-smi -i $i --query-gpu=memory.used --format=csv,noheader 2>/dev/null | tr -d ' ' || echo "0")
            local mem_total=$(nvidia-smi -i $i --query-gpu=memory.total --format=csv,noheader 2>/dev/null | tr -d ' ' || echo "0")
            local temp=$(nvidia-smi -i $i --query-gpu=temperature.gpu --format=csv,noheader,nounits 2>/dev/null | tr -d ' ' || echo "0")
            
            if [ "$first" = true ]; then
                first=false
            else
                gpus_json+=","
            fi
            
            gpus_json+="{\"index\":$i,\"utilization\":${util:-0},\"memory_used\":\"$mem_used\",\"memory_total\":\"$mem_total\",\"temperature\":${temp:-0}}"
            ((i++))
        done < <(nvidia-smi -L 2>/dev/null)
        gpus_json+="]"
        
        cat <<EOF
{
    "driver_version": "$drv",
    "cuda_version": "$cuda",
    "count": $cnt,
    "model": "$model",
    "memory_total": "$mem",
    "gpus": $gpus_json
}
EOF
    else
        echo "null"
    fi
}

# 采集 IB 信息
collect_ib_info() {
    log "Collecting IB information..."
    
    if command_exists ibstat; then
        local active=0
        local ports_json="["
        local first=true
        local port="" state="" rate="" port_num=""
        
        while IFS= read -r line; do
            # 匹配设备名：CA 'mlx5_0'
            if [[ "$line" =~ ^CA\ \'([^\']+)\' ]]; then
                # 保存上一个端口
                if [ -n "$port" ]; then
                    if [ "$first" = true ]; then
                        first=false
                    else
                        ports_json+=","
                    fi
                    ports_json+="{\"name\":\"$port\",\"port_num\":${port_num:-1},\"state\":\"$state\",\"rate\":\"$rate\"}"
                    [ "$state" = "Active" ] && ((active++))
                fi
                port="${BASH_REMATCH[1]}"
                state=""
                rate=""
                port_num=""
            fi
            # 匹配端口号
            [[ "$line" =~ Port\ ([0-9]+): ]] && port_num="${BASH_REMATCH[1]}"
            # 匹配状态
            [[ "$line" =~ State:\ *([A-Za-z]+) ]] && state="${BASH_REMATCH[1]}"
            # 匹配速率
            [[ "$line" =~ Rate:\ *([0-9]+) ]] && rate="${BASH_REMATCH[1]} Gb/sec"
        done < <(ibstat 2>/dev/null)
        
        # 保存最后一个端口
        if [ -n "$port" ]; then
            if [ "$first" = true ]; then
                first=false
            else
                ports_json+=","
            fi
            ports_json+="{\"name\":\"$port\",\"port_num\":${port_num:-1},\"state\":\"$state\",\"rate\":\"$rate\"}"
            [ "$state" = "Active" ] && ((active++))
        fi
        
        ports_json+="]"
        
        cat <<EOF
{
    "active_count": $active,
    "ports": $ports_json
}
EOF
    else
        echo "null"
    fi
}

# 采集 RoCE 信息
collect_roce_info() {
    log "Collecting RoCE information..."
    
    if command_exists ibdev2netdev; then
        local roce_json="["
        local first=true
        local count=0
        
        while read -r line; do
            # 格式: mlx5_0 port 1 ==> enp1s0f0np0 (Up)
            if [[ "$line" =~ ^([a-zA-Z0-9_]+)\ port\ ([0-9]+)\ ==\>\ ([a-zA-Z0-9_]+)\ \(([A-Za-z]+)\) ]]; then
                local rdma_dev="${BASH_REMATCH[1]}"
                local port="${BASH_REMATCH[2]}"
                local netdev="${BASH_REMATCH[3]}"
                local state="${BASH_REMATCH[4]}"
                
                if [ "$first" = true ]; then
                    first=false
                else
                    roce_json+=","
                fi
                
                roce_json+="{\"rdma_dev\":\"$rdma_dev\",\"port\":$port,\"netdev\":\"$netdev\",\"state\":\"$state\"}"
                ((count++))
            fi
        done < <(ibdev2netdev 2>/dev/null)
        
        roce_json+="]"
        
        cat <<EOF
{
    "count": $count,
    "interfaces": $roce_json
}
EOF
    else
        echo "null"
    fi
}

# 采集系统信息
collect_system_info() {
    log "Collecting System information..."
    
    local kern=$(uname -r)
    local os=""
    if [ -f /etc/os-release ]; then
        os=$(grep -oP 'PRETTY_NAME="\K[^"]+' /etc/os-release 2>/dev/null || echo "")
    fi
    local arch=$(uname -m)
    local uptime_sec=$(cat /proc/uptime 2>/dev/null | awk '{print int($1)}' || echo "0")
    
    cat <<EOF
{
    "kernel_version": "$kern",
    "os_version": "$os",
    "hostname": "$(hostname)",
    "arch": "$arch",
    "uptime_seconds": $uptime_sec
}
EOF
}

# 发送指标数据
send_metrics() {
    log "Sending metrics to $CALLBACK_URL..."
    
    local ts=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local cpu=$(collect_cpu_info)
    local memory=$(collect_memory_info)
    local network=$(collect_network_info)
    local gpu=$(collect_gpu_info)
    local ib=$(collect_ib_info)
    local roce=$(collect_roce_info)
    local sys=$(collect_system_info)
    
    # 构建 JSON payload
    local payload=$(cat <<EOF
{
    "minion_id": "$MINION_ID",
    "timestamp": "$ts",
    "cpu": $cpu,
    "memory": $memory,
    "network": $network,
    "gpu": $gpu,
    "ib": $ib,
    "roce": $roce,
    "system": $sys
}
EOF
)
    
    # 发送请求
    local curl_opts="-s -X POST -H 'Content-Type: application/json' --connect-timeout 10 --max-time 30"
    
    if [ -n "$API_TOKEN" ]; then
        curl -s -X POST \
            -H "Content-Type: application/json" \
            -H "X-API-Token: $API_TOKEN" \
            -d "$payload" \
            --connect-timeout 10 \
            --max-time 30 \
            "$CALLBACK_URL"
    else
        curl -s -X POST \
            -H "Content-Type: application/json" \
            -d "$payload" \
            --connect-timeout 10 \
            --max-time 30 \
            "$CALLBACK_URL"
    fi
}

# 主函数
main() {
    log "Starting node metrics collection for $MINION_ID..."
    
    if send_metrics; then
        log "Metrics sent successfully"
    else
        log "Failed to send metrics"
        exit 1
    fi
}

main "$@"
