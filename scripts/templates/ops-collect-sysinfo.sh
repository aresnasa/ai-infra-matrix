#!/bin/bash
# =============================================================================
# 系统信息采集脚本 - 用于资产管理 (JSON 格式输出)
# 用法: ./ops-collect-sysinfo.sh [--pretty]
# =============================================================================

PRETTY=false
[ "$1" = "--pretty" ] && PRETTY=true

# JSON 输出函数
json_escape() {
    echo -n "$1" | sed 's/\\/\\\\/g; s/"/\\"/g; s/\t/\\t/g' | tr -d '\n'
}

echo "{"
echo "  \"hostname\": \"$(hostname)\","
echo "  \"collected_at\": \"$(date -Iseconds)\","

# 操作系统信息
OS_ID=$(cat /etc/os-release 2>/dev/null | grep ^ID= | cut -d= -f2 | tr -d '"')
OS_VERSION=$(cat /etc/os-release 2>/dev/null | grep VERSION_ID | cut -d= -f2 | tr -d '"')
echo "  \"os\": {"
echo "    \"name\": \"$OS_ID\","
echo "    \"version\": \"$OS_VERSION\","
echo "    \"kernel\": \"$(uname -r)\","
echo "    \"arch\": \"$(uname -m)\""
echo "  },"

# CPU 信息
CPU_MODEL=$(cat /proc/cpuinfo | grep "model name" | head -1 | cut -d: -f2 | xargs)
CPU_CORES=$(nproc)
CPU_SOCKETS=$(cat /proc/cpuinfo | grep "physical id" | sort -u | wc -l)
[ "$CPU_SOCKETS" -eq 0 ] && CPU_SOCKETS=1
echo "  \"cpu\": {"
echo "    \"model\": \"$(json_escape "$CPU_MODEL")\","
echo "    \"cores\": $CPU_CORES,"
echo "    \"sockets\": $CPU_SOCKETS"
echo "  },"

# 内存信息
MEM_TOTAL=$(free -b | awk '/Mem:/{print $2}')
MEM_TOTAL_GB=$(echo "scale=2; $MEM_TOTAL/1024/1024/1024" | bc 2>/dev/null || echo "0")
echo "  \"memory\": {"
echo "    \"total_bytes\": $MEM_TOTAL,"
echo "    \"total_gb\": $MEM_TOTAL_GB"
echo "  },"

# GPU 信息
if command -v nvidia-smi &> /dev/null; then
    GPU_COUNT=$(nvidia-smi -L 2>/dev/null | wc -l)
    GPU_MODEL=$(nvidia-smi --query-gpu=name --format=csv,noheader 2>/dev/null | head -1)
    GPU_DRIVER=$(nvidia-smi --query-gpu=driver_version --format=csv,noheader 2>/dev/null | head -1)
    GPU_MEM=$(nvidia-smi --query-gpu=memory.total --format=csv,noheader 2>/dev/null | head -1)
    echo "  \"gpu\": {"
    echo "    \"vendor\": \"nvidia\","
    echo "    \"count\": $GPU_COUNT,"
    echo "    \"model\": \"$(json_escape "$GPU_MODEL")\","
    echo "    \"driver_version\": \"$GPU_DRIVER\","
    echo "    \"memory\": \"$GPU_MEM\""
    echo "  },"
else
    echo "  \"gpu\": null,"
fi

# NPU 信息
if command -v npu-smi &> /dev/null; then
    NPU_COUNT=$(npu-smi info -l 2>/dev/null | grep -c "NPU ID" || echo "0")
    echo "  \"npu\": {"
    echo "    \"vendor\": \"huawei\","
    echo "    \"count\": $NPU_COUNT"
    echo "  },"
else
    echo "  \"npu\": null,"
fi

# 磁盘信息
echo "  \"disks\": ["
FIRST_DISK=1
for disk in $(lsblk -d -o NAME,TYPE 2>/dev/null | awk '$2=="disk"{print $1}'); do
    SIZE=$(lsblk -d -b -o SIZE /dev/$disk 2>/dev/null | tail -1)
    SIZE_GB=$(echo "scale=2; ${SIZE:-0}/1024/1024/1024" | bc 2>/dev/null || echo "0")
    MODEL=$(cat /sys/block/$disk/device/model 2>/dev/null | xargs || echo "unknown")
    [ $FIRST_DISK -eq 0 ] && echo ","
    echo "    {\"name\": \"/dev/$disk\", \"size_gb\": $SIZE_GB, \"model\": \"$(json_escape "$MODEL")\"}"
    FIRST_DISK=0
done
echo ""
echo "  ],"

# 网卡信息
echo "  \"network_interfaces\": ["
FIRST_NIC=1
for nic in $(ip -br link show 2>/dev/null | awk '$1!="lo"{print $1}' | grep -v "docker\|veth\|br-"); do
    MAC=$(ip link show $nic 2>/dev/null | awk '/link\/ether/{print $2}')
    IP=$(ip -4 addr show $nic 2>/dev/null | awk '/inet /{print $2}' | cut -d/ -f1)
    SPEED=$(ethtool $nic 2>/dev/null | grep Speed | awk '{print $2}')
    [ $FIRST_NIC -eq 0 ] && echo ","
    echo "    {\"name\": \"$nic\", \"mac\": \"$MAC\", \"ip\": \"$IP\", \"speed\": \"$SPEED\"}"
    FIRST_NIC=0
done
echo ""
echo "  ],"

# InfiniBand 信息
if command -v ibstat &> /dev/null; then
    IB_COUNT=$(ibstat -l 2>/dev/null | wc -l)
    echo "  \"infiniband\": {"
    echo "    \"count\": $IB_COUNT"
    echo "  },"
else
    echo "  \"infiniband\": null,"
fi

# 服务状态
echo "  \"services\": {"
FIRST_SVC=1
for svc in docker kubelet slurmd salt-minion containerd; do
    STATUS=$(systemctl is-active $svc 2>/dev/null || echo "not-found")
    [ $FIRST_SVC -eq 0 ] && echo ","
    echo "    \"$svc\": \"$STATUS\""
    FIRST_SVC=0
done
echo ""
echo "  }"

echo "}"
