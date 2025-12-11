#!/bin/bash
# get_cpu_memory.sh - 获取 CPU 和内存使用率信息
# 输出格式: cpu_percent|mem_total_kb|mem_available_kb
# CPU 使用率采用两次采样计算差值的方式，确保获取的是瞬时使用率

# 第一次读取 CPU 统计
read cpu_user1 cpu_nice1 cpu_system1 cpu_idle1 cpu_iowait1 cpu_irq1 cpu_softirq1 cpu_steal1 <<< $(awk '/^cpu / {print $2,$3,$4,$5,$6,$7,$8,$9}' /proc/stat)
cpu_total1=$((cpu_user1 + cpu_nice1 + cpu_system1 + cpu_idle1 + cpu_iowait1 + cpu_irq1 + cpu_softirq1 + cpu_steal1))
cpu_idle_total1=$((cpu_idle1 + cpu_iowait1))

# 等待 0.5 秒
sleep 0.5

# 第二次读取 CPU 统计
read cpu_user2 cpu_nice2 cpu_system2 cpu_idle2 cpu_iowait2 cpu_irq2 cpu_softirq2 cpu_steal2 <<< $(awk '/^cpu / {print $2,$3,$4,$5,$6,$7,$8,$9}' /proc/stat)
cpu_total2=$((cpu_user2 + cpu_nice2 + cpu_system2 + cpu_idle2 + cpu_iowait2 + cpu_irq2 + cpu_softirq2 + cpu_steal2))
cpu_idle_total2=$((cpu_idle2 + cpu_iowait2))

# 计算差值
cpu_total_diff=$((cpu_total2 - cpu_total1))
cpu_idle_diff=$((cpu_idle_total2 - cpu_idle_total1))

# 计算 CPU 使用率
if [ "$cpu_total_diff" -gt 0 ]; then
    cpu=$(awk "BEGIN {printf \"%.2f\", 100 * (1 - $cpu_idle_diff / $cpu_total_diff)}")
else
    cpu="0.00"
fi

# 获取内存信息
mem_total=$(awk '/MemTotal/ {print $2}' /proc/meminfo 2>/dev/null || echo "0")
mem_avail=$(awk '/MemAvailable/ {print $2}' /proc/meminfo 2>/dev/null || echo "0")

echo "${cpu}|${mem_total}|${mem_avail}"
