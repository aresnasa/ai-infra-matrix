#!/bin/bash
# get_tpu_info.sh - 获取 TPU (Google TPU 或其他 AI 加速器) 信息
# 用法: 
#   get_tpu_info.sh google   - 获取 Google TPU 信息
#   get_tpu_info.sh habana   - 获取 Habana Gaudi 信息
#   get_tpu_info.sh count    - 获取总数量
#   get_tpu_info.sh all      - 自动检测并获取所有 TPU 信息

get_google_tpu() {
    # Google TPU 通常在 GCP 环境中
    # 检查 libtpu 或 tpu_driver
    if [ -d "/dev/accel" ] || [ -f "/usr/lib/libtpu.so" ]; then
        # 尝试获取 TPU 信息
        count=$(ls -1 /dev/accel* 2>/dev/null | wc -l || echo "0")
        version="google-tpu"
        model="Cloud TPU"
        
        echo "google|${version}|${model}|${count:-0}|0|0|0"
    else
        echo "google|not_installed|N/A|0|0|0|0"
    fi
}

get_habana_gaudi() {
    # Habana Gaudi 使用 hl-smi 命令
    if command -v hl-smi &>/dev/null; then
        version=$(hl-smi 2>/dev/null | grep -i "driver version" | head -1 | awk -F': ' '{print $2}')
        count=$(hl-smi -L 2>/dev/null | wc -l 2>/dev/null || echo "0")
        model=$(hl-smi 2>/dev/null | grep -i "Product Name" | head -1 | awk -F': ' '{print $2}')
        
        echo "habana|${version:-N/A}|${model:-Gaudi}|${count:-0}|0|0|0"
    else
        echo "habana|not_installed|N/A|0|0|0|0"
    fi
}

get_graphcore_ipu() {
    # Graphcore IPU 使用 gc-info 命令
    if command -v gc-info &>/dev/null; then
        version=$(gc-info --ipu-count 2>/dev/null | head -1)
        count=$(gc-info --ipu-count 2>/dev/null | grep -oE '[0-9]+' | head -1 || echo "0")
        model="Graphcore IPU"
        
        echo "graphcore|${version:-N/A}|${model}|${count:-0}|0|0|0"
    else
        echo "graphcore|not_installed|N/A|0|0|0|0"
    fi
}

get_total_count() {
    total=0
    
    # Google TPU
    if [ -d "/dev/accel" ]; then
        google_count=$(ls -1 /dev/accel* 2>/dev/null | wc -l || echo "0")
        total=$((total + google_count))
    fi
    
    # Habana Gaudi
    if command -v hl-smi &>/dev/null; then
        habana_count=$(hl-smi -L 2>/dev/null | wc -l 2>/dev/null || echo "0")
        total=$((total + habana_count))
    fi
    
    # Graphcore IPU
    if command -v gc-info &>/dev/null; then
        graphcore_count=$(gc-info --ipu-count 2>/dev/null | grep -oE '[0-9]+' | head -1 || echo "0")
        total=$((total + graphcore_count))
    fi
    
    echo "$total"
}

case "${1:-all}" in
    google|tpu)
        get_google_tpu
        ;;
    habana|gaudi)
        get_habana_gaudi
        ;;
    graphcore|ipu)
        get_graphcore_ipu
        ;;
    count)
        get_total_count
        ;;
    all)
        # 自动检测并输出所有 TPU/其他加速器 信息
        result=""
        
        # 检测 Google TPU
        if [ -d "/dev/accel" ] || [ -f "/usr/lib/libtpu.so" ]; then
            google_result=$(get_google_tpu)
            if [[ ! "$google_result" =~ "not_installed" ]]; then
                result="${result}${google_result}\n"
            fi
        fi
        
        # 检测 Habana Gaudi
        if command -v hl-smi &>/dev/null; then
            habana_result=$(get_habana_gaudi)
            if [[ ! "$habana_result" =~ "not_installed" ]]; then
                result="${result}${habana_result}\n"
            fi
        fi
        
        # 检测 Graphcore IPU
        if command -v gc-info &>/dev/null; then
            graphcore_result=$(get_graphcore_ipu)
            if [[ ! "$graphcore_result" =~ "not_installed" ]]; then
                result="${result}${graphcore_result}\n"
            fi
        fi
        
        if [ -z "$result" ]; then
            echo "none|not_installed|N/A|0|0|0|0"
        else
            echo -e "$result" | grep -v '^$'
        fi
        ;;
esac
