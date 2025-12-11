#!/bin/bash
# get_cpu_memory.sh - 获取 CPU 和内存使用率信息
# 输出格式: cpu_percent|mem_total_kb|mem_available_kb

cpu=$(cat /proc/stat | head -1 | awk '{idle=$5; total=$2+$3+$4+$5+$6+$7+$8; print 100*(1-idle/total)}' 2>/dev/null || echo "0")
mem_total=$(awk '/MemTotal/ {print $2}' /proc/meminfo 2>/dev/null || echo "0")
mem_avail=$(awk '/MemAvailable/ {print $2}' /proc/meminfo 2>/dev/null || echo "0")
echo "${cpu}|${mem_total}|${mem_avail}"
