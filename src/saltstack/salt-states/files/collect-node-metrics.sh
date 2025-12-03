#!/bin/bash
#
# Node Metrics Collection Script
# 采集 CPU、内存、网络带宽、GPU 利用率/显存、IB 网卡状态、RoCE 等信息并回调到 Backend
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

# 采集 CPU 信息
collect_cpu_info() {
    log "Collecting CPU information..."
    
    local cpu_cores
    cpu_cores=$(nproc 2>/dev/null || grep -c ^processor /proc/cpuinfo 2>/dev/null || echo "0")
    
    local cpu_model
    cpu_model=$(grep -m1 'model name' /proc/cpuinfo 2>/dev/null | cut -d: -f2 | xargs || echo "unknown")
    
    # CPU 使用率 (取3次平均)
    local cpu_usage="0"
    if command_exists mpstat; then
        cpu_usage=$(mpstat 1 1 2>/dev/null | awk '/Average/ && $NF ~ /[0-9.]+/ {print 100 - $NF}' || echo "0")
    elif [ -f /proc/stat ]; then
        # 使用 /proc/stat 计算 CPU 使用率
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
            # 旧版内核可能没有 MemAvailable
            local mem_free=$(grep MemFree /proc/meminfo | awk '{print $2}')
            local buffers=$(grep Buffers /proc/meminfo | awk '{print $2}')
            local cached=$(grep "^Cached:" /proc/meminfo | awk '{print $2}')
            mem_available=$((mem_free + buffers + cached))
        fi
        mem_used=$((mem_total - mem_available))
        mem_percent=$(awk "BEGIN {printf \"%.1f\", $mem_used * 100 / $mem_total}")
        
        # 转换为 GB
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
    
    # 获取所有网络接口的流量统计
    for iface in $(ls /sys/class/net/ 2>/dev/null | grep -v "^lo$"); do
        if [ -d "/sys/class/net/$iface" ]; then
            local rx_bytes1=$(cat /sys/class/net/$iface/statistics/rx_bytes 2>/dev/null || echo 0)
            local tx_bytes1=$(cat /sys/class/net/$iface/statistics/tx_bytes 2>/dev/null || echo 0)
            
            sleep 0.5
            
            local rx_bytes2=$(cat /sys/class/net/$iface/statistics/rx_bytes 2>/dev/null || echo 0)
            local tx_bytes2=$(cat /sys/class/net/$iface/statistics/tx_bytes 2>/dev/null || echo 0)
            
            # 计算速率 (bytes/s * 2 因为我们只等了0.5秒)
            local rx_rate=$(( (rx_bytes2 - rx_bytes1) * 2 ))
            local tx_rate=$(( (tx_bytes2 - tx_bytes1) * 2 ))
            
            # 获取接口状态
            local state=$(cat /sys/class/net/$iface/operstate 2>/dev/null || echo "unknown")
            
            # 获取 IP 地址
            local ip_addr=""
            if command_exists ip; then
                ip_addr=$(ip addr show $iface 2>/dev/null | grep 'inet ' | head -1 | awk '{print $2}' | cut -d/ -f1 || echo "")
            fi
            
            # 获取链路速度 (Mbps)
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
    
    # 获取活跃连接数
    local active_connections=0
    if command_exists ss; then
        active_connections=$(ss -tun 2>/dev/null | grep -c ESTAB || echo 0)
    elif command_exists netstat; then
        active_connections=$(netstat -tun 2>/dev/null | grep -c ESTABLISHED || echo 0)
    fi
    
    cat <<EOF
{
    "interfaces": $interfaces_json,
    "active_connections": $active_connections
}
EOF
}

# 采集 GPU 信息 (增强版，包含利用率和显存)
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
        
        # 获取总体利用率（所有 GPU 平均）
        local avg_utilization=0
        local avg_memory_used=0
        local total_memory=0
        
        # 获取每个 GPU 的详细信息
        local gpus_detail="["
        local first=true
        while IFS=',' read -r idx uuid name mem_total mem_used mem_free temp power_draw power_limit util; do
            if [ -n "$idx" ]; then
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
                
                # 计算平均值
                avg_utilization=$((avg_utilization + ${util:-0}))
                
                # 解析显存数值 (去掉 MiB 单位)
                local mem_used_val=$(echo "$mem_used" | sed 's/[^0-9]//g')
                local mem_total_val=$(echo "$mem_total" | sed 's/[^0-9]//g')
                avg_memory_used=$((avg_memory_used + ${mem_used_val:-0}))
                total_memory=$((total_memory + ${mem_total_val:-0}))
                
                gpus_detail+="{\"index\":$idx,\"uuid\":\"$uuid\",\"name\":\"$name\",\"memory_total\":\"$mem_total\",\"memory_used\":\"$mem_used\",\"memory_free\":\"$mem_free\",\"temperature\":${temp:-0},\"power_draw\":\"$power_draw\",\"power_limit\":\"$power_limit\",\"utilization\":${util:-0}}"
            fi
        done < <(nvidia-smi --query-gpu=index,uuid,name,memory.total,memory.used,memory.free,temperature.gpu,power.draw,power.limit,utilization.gpu --format=csv,noheader 2>/dev/null || echo "")
        gpus_detail+="]"
        
        # 计算平均利用率和显存使用率
        if [ "$gpu_count" -gt 0 ]; then
            avg_utilization=$((avg_utilization / gpu_count))
            if [ "$total_memory" -gt 0 ]; then
                local memory_usage_percent=$(awk "BEGIN {printf \"%.1f\", $avg_memory_used * 100 / $total_memory}")
            else
                local memory_usage_percent="0"
            fi
        else
            memory_usage_percent="0"
        fi
        
        gpu_json=$(cat <<EOF
{
    "driver_version": "$driver_version",
    "cuda_version": "$cuda_version",
    "count": $gpu_count,
    "model": "$gpu_model",
    "memory_total": "$memory_total",
    "avg_utilization": $avg_utilization,
    "memory_used_mb": $avg_memory_used,
    "memory_total_mb": $total_memory,
    "memory_usage_percent": $memory_usage_percent,
    "gpus": $gpus_detail
}
EOF
)
        log "GPU: driver=$driver_version, cuda=$cuda_version, count=$gpu_count, avg_util=${avg_utilization}%, mem_usage=${memory_usage_percent}%"
    else
        log "nvidia-smi not found, skipping GPU collection"
        gpu_json="null"
    fi
    
    echo "$gpu_json"
}

# 采集 InfiniBand 信息 (增强版)
collect_ib_info() {
    local ib_json="{}"
    
    if command_exists ibstat; then
        log "Collecting InfiniBand information..."
        
        local active_count=0
        local down_count=0
        local ports_json="["
        local first=true
        
        # 解析 ibstat 输出
        local current_ca=""
        local current_ca_type=""
        local current_firmware=""
        local current_port_num=""
        local current_state=""
        local current_phys_state=""
        local current_rate=""
        local current_guid=""
        local in_port_section=false
        
        while IFS= read -r line; do
            # 检测 CA 名称行 (例如: CA 'mlx5_0')
            if [[ "$line" =~ ^CA\ \'([^\']+)\' ]]; then
                current_ca="${BASH_REMATCH[1]}"
                current_ca_type=""
                current_firmware=""
                in_port_section=false
            fi
            
            # CA 类型
            if [[ "$line" =~ CA\ type:\ *(.+) ]]; then
                current_ca_type=$(echo "${BASH_REMATCH[1]}" | xargs)
            fi
            
            # 固件版本
            if [[ "$line" =~ Firmware\ version:\ *([^ ]+) ]]; then
                current_firmware="${BASH_REMATCH[1]}"
            fi
            
            # 端口号
            if [[ "$line" =~ Port\ ([0-9]+): ]]; then
                current_port_num="${BASH_REMATCH[1]}"
                in_port_section=true
                current_state=""
                current_phys_state=""
                current_rate=""
                current_guid=""
            fi
            
            # 端口状态
            if $in_port_section && [[ "$line" =~ ^[[:space:]]+State:\ *([A-Za-z]+) ]]; then
                current_state="${BASH_REMATCH[1]}"
            fi
            
            # 物理状态
            if $in_port_section && [[ "$line" =~ Physical\ state:\ *([A-Za-z]+) ]]; then
                current_phys_state="${BASH_REMATCH[1]}"
            fi
            
            # 速率
            if $in_port_section && [[ "$line" =~ Rate:\ *([0-9]+) ]]; then
                current_rate="${BASH_REMATCH[1]}"
            fi
            
            # 端口 GUID
            if $in_port_section && [[ "$line" =~ Port\ GUID:\ *([^ ]+) ]]; then
                current_guid="${BASH_REMATCH[1]}"
                
                # 收集完整的端口信息，保存
                if [ -n "$current_ca" ] && [ -n "$current_port_num" ]; then
                    local port_name="${current_ca}/port${current_port_num}"
                    
                    if [ "$first" = true ]; then
                        first=false
                    else
                        ports_json+=","
                    fi
                    
                    ports_json+="{\"name\":\"$current_ca\",\"port\":$current_port_num,\"ca_type\":\"$current_ca_type\",\"firmware\":\"$current_firmware\",\"state\":\"$current_state\",\"physical_state\":\"$current_phys_state\",\"rate\":\"${current_rate:-0}\",\"guid\":\"$current_guid\"}"
                    
                    if [ "$current_state" = "Active" ]; then
                        ((active_count++)) || true
                    else
                        ((down_count++)) || true
                    fi
                fi
                in_port_section=false
            fi
            
        done < <(ibstat 2>/dev/null || echo "")
        
        ports_json+="]"
        
        ib_json=$(cat <<EOF
{
    "active_count": $active_count,
    "down_count": $down_count,
    "total_count": $((active_count + down_count)),
    "ports": $ports_json
}
EOF
)
        log "InfiniBand: active=$active_count, down=$down_count"
    else
        log "ibstat not found, skipping IB collection"
        ib_json="null"
    fi
    
    echo "$ib_json"
}

# 采集 RoCE 网络信息
collect_roce_info() {
    local roce_json="null"
    
    # 检查是否有 RDMA 设备
    if [ -d /sys/class/infiniband ] && ls /sys/class/infiniband/ 2>/dev/null | grep -q .; then
        log "Collecting RoCE information..."
        
        local devices_json="["
        local first=true
        
        for device in /sys/class/infiniband/*; do
            if [ -d "$device" ]; then
                local dev_name=$(basename "$device")
                local node_type=""
                local node_guid=""
                
                if [ -f "$device/node_type" ]; then
                    node_type=$(cat "$device/node_type" 2>/dev/null | xargs || echo "")
                fi
                
                if [ -f "$device/node_guid" ]; then
                    node_guid=$(cat "$device/node_guid" 2>/dev/null | xargs || echo "")
                fi
                
                # 检查是否是 RoCE 设备 (node_type 通常是 1=CA 表示 InfiniBand, 4=RNIC 表示 RoCE)
                local is_roce="false"
                if [[ "$node_type" == *"RNIC"* ]] || [[ "$dev_name" == *"roce"* ]]; then
                    is_roce="true"
                fi
                
                # 检查端口状态
                local port_state=""
                if [ -f "$device/ports/1/state" ]; then
                    port_state=$(cat "$device/ports/1/state" 2>/dev/null | awk '{print $2}' || echo "")
                fi
                
                # 检查 GID 表（RoCE 的 IPv4/IPv6 地址）
                local gid=""
                if [ -f "$device/ports/1/gids/0" ]; then
                    gid=$(cat "$device/ports/1/gids/0" 2>/dev/null || echo "")
                fi
                
                if [ "$first" = true ]; then
                    first=false
                else
                    devices_json+=","
                fi
                
                devices_json+="{\"name\":\"$dev_name\",\"node_type\":\"$node_type\",\"node_guid\":\"$node_guid\",\"is_roce\":$is_roce,\"state\":\"$port_state\",\"gid\":\"$gid\"}"
            fi
        done
        
        devices_json+="]"
        
        roce_json=$(cat <<EOF
{
    "devices": $devices_json
}
EOF
)
        log "RoCE: collected device info"
    else
        log "No RDMA devices found, skipping RoCE collection"
    fi
    
    echo "$roce_json"
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
    
    local uptime_seconds
    uptime_seconds=$(cat /proc/uptime 2>/dev/null | awk '{print int($1)}' || echo "0")
    
    local uptime_str
    uptime_str=$(uptime -p 2>/dev/null || uptime 2>/dev/null | awk '{print $3,$4,$5}' || echo "unknown")
    
    cat <<EOF
{
    "kernel_version": "$kernel_version",
    "os_version": "$os_version",
    "hostname": "$hostname",
    "uptime": "$uptime_str",
    "uptime_seconds": $uptime_seconds
}
EOF
}

# 发送数据到 Backend
send_metrics() {
    local cpu_info="$1"
    local memory_info="$2"
    local network_info="$3"
    local gpu_info="$4"
    local ib_info="$5"
    local roce_info="$6"
    local system_info="$7"
    
    local timestamp
    timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    
    # 构建 JSON payload
    local payload
    payload=$(cat <<EOF
{
    "minion_id": "$MINION_ID",
    "timestamp": "$timestamp",
    "cpu": $cpu_info,
    "memory": $memory_info,
    "network": $network_info,
    "gpu": $gpu_info,
    "ib": $ib_info,
    "roce": $roce_info,
    "system": $system_info
}
EOF
)
    
    log "Sending metrics to $CALLBACK_URL"
    
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
    local cpu_info
    cpu_info=$(collect_cpu_info)
    
    local memory_info
    memory_info=$(collect_memory_info)
    
    local network_info
    network_info=$(collect_network_info)
    
    local gpu_info
    gpu_info=$(collect_gpu_info)
    
    local ib_info
    ib_info=$(collect_ib_info)
    
    local roce_info
    roce_info=$(collect_roce_info)
    
    local system_info
    system_info=$(collect_system_info)
    
    # 发送到 Backend
    if send_metrics "$cpu_info" "$memory_info" "$network_info" "$gpu_info" "$ib_info" "$roce_info" "$system_info"; then
        log "Collection completed successfully"
    else
        log "Collection completed with errors"
        exit 1
    fi
}

# 执行主函数
main "$@"
