#!/bin/bash
# =============================================================================
# 日常巡检脚本 - GPU 集群和物理机
# 用法: ./ops-daily-inspection.sh [--json] [--output FILE]
# =============================================================================

# 参数解析
OUTPUT_JSON=false
OUTPUT_FILE=""
while [[ $# -gt 0 ]]; do
    case $1 in
        --json) OUTPUT_JSON=true; shift ;;
        --output) OUTPUT_FILE="$2"; shift 2 ;;
        *) shift ;;
    esac
done

# 如果需要 JSON 输出，重定向
if [ "$OUTPUT_JSON" = true ]; then
    exec 3>&1  # 保存原始 stdout
fi

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║               日 常 巡 检 报 告                                ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo "主机名: $(hostname)"
echo "IP 地址: $(hostname -I 2>/dev/null | awk '{print $1}')"
echo "巡检时间: $(date '+%Y-%m-%d %H:%M:%S')"
echo "运行时长: $(uptime -p 2>/dev/null || uptime)"
echo ""

# ========== 1. 系统基础信息 ==========
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "【1. 系统基础信息】"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "操作系统: $(cat /etc/os-release 2>/dev/null | grep PRETTY_NAME | cut -d'"' -f2)"
echo "内核版本: $(uname -r)"
echo "CPU 核心: $(nproc) 核"
echo "负载均值: $(cat /proc/loadavg | awk '{print $1, $2, $3}')"
echo ""

# ========== 2. 内存状态 ==========
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "【2. 内存状态】"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
free -h
MEM_USED_PCT=$(free | awk '/Mem/{printf "%.1f", $3/$2*100}')
if (( $(echo "$MEM_USED_PCT > 90" | bc -l) )); then
    echo "⚠️ 警告: 内存使用率 $MEM_USED_PCT% 超过 90%"
else
    echo "✅ 内存使用率: $MEM_USED_PCT%"
fi
echo ""

# ========== 3. 磁盘状态 ==========
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "【3. 磁盘状态】"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
df -h | grep -v "tmpfs\|loop\|udev\|overlay"
echo ""
echo "--- 磁盘使用率告警检查 ---"
DISK_WARN=0
while read -r line; do
    usage=$(echo "$line" | awk '{print $5}' | tr -d '%')
    mount=$(echo "$line" | awk '{print $6}')
    if [ -n "$usage" ] && [ "$usage" -gt 85 ] 2>/dev/null; then
        echo "⚠️ 磁盘 $mount 使用率 $usage% (警告阈值: 85%)"
        DISK_WARN=1
    fi
done < <(df -h | grep -v "tmpfs\|loop\|udev\|overlay\|Filesystem")
[ "$DISK_WARN" -eq 0 ] && echo "✅ 所有磁盘使用率正常"
echo ""

# ========== 4. GPU 状态 (NVIDIA) ==========
if command -v nvidia-smi &> /dev/null; then
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "【4. GPU 状态 (NVIDIA)】"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    GPU_COUNT=$(nvidia-smi -L 2>/dev/null | wc -l)
    echo "检测到 GPU 数量: $GPU_COUNT"
    echo ""
    nvidia-smi --query-gpu=index,name,driver_version,temperature.gpu,utilization.gpu,memory.used,memory.total --format=csv
    echo ""
    # 检查 GPU 温度
    echo "--- GPU 温度检查 ---"
    nvidia-smi --query-gpu=index,temperature.gpu --format=csv,noheader | while read line; do
        idx=$(echo "$line" | cut -d',' -f1)
        temp=$(echo "$line" | cut -d',' -f2 | tr -d ' ')
        if [ "$temp" -gt 85 ]; then
            echo "⚠️ GPU $idx 温度 $temp°C 过高！"
        elif [ "$temp" -gt 75 ]; then
            echo "⚠️ GPU $idx 温度 $temp°C 偏高"
        else
            echo "✅ GPU $idx 温度 $temp°C 正常"
        fi
    done
    echo ""
    
    # 检查 XID 错误
    echo "--- GPU XID 错误检查 ---"
    XID_COUNT=$(dmesg 2>/dev/null | grep -c "NVRM: Xid" || echo "0")
    if [ "$XID_COUNT" -gt 0 ]; then
        echo "⚠️ 发现 $XID_COUNT 条 XID 错误日志"
        dmesg 2>/dev/null | grep "NVRM: Xid" | tail -5
    else
        echo "✅ 无 XID 错误"
    fi
    echo ""
fi

# ========== 5. NPU 状态 (华为昇腾) ==========
if command -v npu-smi &> /dev/null; then
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "【5. NPU 状态 (华为昇腾)】"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    npu-smi info 2>/dev/null | head -30
    echo ""
fi

# ========== 6. 网络状态 ==========
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "【6. 网络状态】"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
# 显示物理网卡状态
ip -br link show | grep -v "lo\|docker\|veth\|br-"
echo ""

# ========== 7. InfiniBand 状态 ==========
if command -v ibstat &> /dev/null; then
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "【7. InfiniBand 状态】"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    IB_DOWN=$(ibstat 2>/dev/null | grep -c "State: Down")
    IB_ACTIVE=$(ibstat 2>/dev/null | grep -c "State: Active")
    echo "IB 端口状态: Active=$IB_ACTIVE, Down=$IB_DOWN"
    if [ "$IB_DOWN" -gt 0 ]; then
        echo "⚠️ 发现 $IB_DOWN 个 IB 端口处于 Down 状态"
        ibstat 2>/dev/null | grep -B5 "State: Down"
    else
        echo "✅ 所有 IB 端口正常"
    fi
    echo ""
fi

# ========== 8. 关键服务状态 ==========
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "【8. 关键服务状态】"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
for svc in docker containerd kubelet slurmd slurmctld salt-minion; do
    if systemctl is-active "$svc" &>/dev/null; then
        echo "✅ $svc: active"
    elif systemctl list-unit-files 2>/dev/null | grep -q "^$svc"; then
        echo "❌ $svc: inactive"
    fi
done
echo ""

# ========== 9. 最近错误日志 ==========
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "【9. 最近错误日志 (最近1小时)】"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
journalctl --since "1 hour ago" -p err --no-pager 2>/dev/null | tail -10 || echo "无法读取 journalctl"
echo ""

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║                    巡 检 完 成                                 ║"
echo "╚════════════════════════════════════════════════════════════════╝"

# 如果指定了输出文件
if [ -n "$OUTPUT_FILE" ]; then
    echo "报告已保存到: $OUTPUT_FILE"
fi
