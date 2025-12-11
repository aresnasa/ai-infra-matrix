#!/bin/bash
# get_npu_info.sh - 获取 NPU (华为昇腾/寒武纪等) 信息
# 用法: 
#   get_npu_info.sh huawei   - 获取华为昇腾 NPU 信息
#   get_npu_info.sh cambricon - 获取寒武纪 MLU 信息
#   get_npu_info.sh all      - 自动检测并获取所有 NPU 信息

get_huawei_npu() {
    # 华为昇腾 NPU 使用 npu-smi 命令
    if command -v npu-smi &>/dev/null; then
        driver=$(npu-smi info -d 0 2>/dev/null | grep -i "driver" | head -1 | awk '{print $NF}')
        count=$(npu-smi info -l 2>/dev/null | grep -c "NPU ID" || echo "0")
        model=$(npu-smi info -d 0 2>/dev/null | grep -i "Name" | head -1 | awk -F': ' '{print $2}')
        
        # 获取使用率（如果支持）
        utilization=$(npu-smi info -t board -d 0 2>/dev/null | grep -i "ai_core" | awk '{print $NF}' | tr -d '%')
        memory_used=$(npu-smi info -t board -d 0 2>/dev/null | grep -i "memory_used" | awk '{print $NF}')
        memory_total=$(npu-smi info -t board -d 0 2>/dev/null | grep -i "memory_total" | awk '{print $NF}')
        
        echo "huawei|${driver:-N/A}|${model:-N/A}|${count:-0}|${utilization:-0}|${memory_used:-0}|${memory_total:-0}"
    else
        echo "huawei|not_installed|N/A|0|0|0|0"
    fi
}

get_cambricon_mlu() {
    # 寒武纪 MLU 使用 cnmon 命令
    if command -v cnmon &>/dev/null; then
        driver=$(cnmon info 2>/dev/null | grep -i "driver version" | head -1 | awk -F': ' '{print $2}')
        count=$(cnmon info 2>/dev/null | grep -c "MLU" || echo "0")
        model=$(cnmon info 2>/dev/null | grep -i "Product Name" | head -1 | awk -F': ' '{print $2}')
        
        # 获取使用率
        utilization=$(cnmon info 2>/dev/null | grep -i "Board Util" | head -1 | awk '{print $NF}' | tr -d '%')
        memory_used=$(cnmon info 2>/dev/null | grep -i "Memory Used" | head -1 | awk '{print $NF}')
        memory_total=$(cnmon info 2>/dev/null | grep -i "Memory Total" | head -1 | awk '{print $NF}')
        
        echo "cambricon|${driver:-N/A}|${model:-N/A}|${count:-0}|${utilization:-0}|${memory_used:-0}|${memory_total:-0}"
    else
        echo "cambricon|not_installed|N/A|0|0|0|0"
    fi
}

get_iluvatar_gpu() {
    # 天数智芯 GPU 使用 ixsmi 命令
    if command -v ixsmi &>/dev/null; then
        driver=$(ixsmi -q 2>/dev/null | grep -i "driver version" | head -1 | awk -F': ' '{print $2}')
        count=$(ixsmi -L 2>/dev/null | wc -l || echo "0")
        model=$(ixsmi -q 2>/dev/null | grep -i "Product Name" | head -1 | awk -F': ' '{print $2}')
        
        echo "iluvatar|${driver:-N/A}|${model:-N/A}|${count:-0}|0|0|0"
    else
        echo "iluvatar|not_installed|N/A|0|0|0|0"
    fi
}

case "${1:-all}" in
    huawei|ascend)
        get_huawei_npu
        ;;
    cambricon|mlu)
        get_cambricon_mlu
        ;;
    iluvatar)
        get_iluvatar_gpu
        ;;
    all)
        # 自动检测并输出所有 NPU 信息
        result=""
        
        # 检测华为昇腾
        if command -v npu-smi &>/dev/null; then
            result="${result}$(get_huawei_npu)\n"
        fi
        
        # 检测寒武纪
        if command -v cnmon &>/dev/null; then
            result="${result}$(get_cambricon_mlu)\n"
        fi
        
        # 检测天数智芯
        if command -v ixsmi &>/dev/null; then
            result="${result}$(get_iluvatar_gpu)\n"
        fi
        
        if [ -z "$result" ]; then
            echo "none|not_installed|N/A|0|0|0|0"
        else
            echo -e "$result" | head -n -1  # 移除最后的空行
        fi
        ;;
esac
