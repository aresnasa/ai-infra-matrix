#!/bin/bash
# get_gpu_info.sh - 获取 GPU (NVIDIA) 信息
# 用法: 
#   get_gpu_info.sh driver   - 获取驱动版本
#   get_gpu_info.sh cuda     - 获取 CUDA 版本
#   get_gpu_info.sh model    - 获取 GPU 型号
#   get_gpu_info.sh count    - 获取 GPU 数量
#   get_gpu_info.sh all      - 获取所有信息 (格式: driver|cuda|model|count)

case "${1:-all}" in
    driver)
        nvidia-smi --query-gpu=driver_version --format=csv,noheader,nounits 2>/dev/null | head -1
        ;;
    cuda)
        nvidia-smi 2>/dev/null | grep -oP 'CUDA Version: \K[0-9.]+' | head -1
        ;;
    model)
        nvidia-smi --query-gpu=name,count --format=csv,noheader 2>/dev/null | head -1
        ;;
    count)
        nvidia-smi -L 2>/dev/null | wc -l
        ;;
    all)
        driver=$(nvidia-smi --query-gpu=driver_version --format=csv,noheader,nounits 2>/dev/null | head -1)
        cuda=$(nvidia-smi 2>/dev/null | grep -oP 'CUDA Version: \K[0-9.]+' | head -1)
        model=$(nvidia-smi --query-gpu=name --format=csv,noheader 2>/dev/null | head -1)
        count=$(nvidia-smi -L 2>/dev/null | wc -l)
        echo "${driver:-N/A}|${cuda:-N/A}|${model:-N/A}|${count:-0}"
        ;;
esac
