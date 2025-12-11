#!/bin/bash
# get_cpu_memory_loadavg.sh - 获取 CPU、内存使用率和负载信息
# 输出格式: cpu_percent|mem_total_kb|mem_available_kb|load_avg

cpu=$(cat /proc/stat | head -1 | awk '{idle=$5; total=$2+$3+$4+$5+$6+$7+$8; print 100*(1-idle/total)}' 2>/dev/null || echo "0")
mem_total=$(awk '/MemTotal/ {print $2}' /proc/meminfo 2>/dev/null || echo "0")
mem_avail=$(awk '/MemAvailable/ {print $2}' /proc/meminfo 2>/dev/null || echo "0")
load_avg=$(cat /proc/loadavg 2>/dev/null | awk '{print $1", "$2", "$3}' || echo "0, 0, 0")
echo "${cpu}|${mem_total}|${mem_avail}|${load_avg}"
