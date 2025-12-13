#!/bin/bash
# get_npu_info.sh - 获取 NPU (华为昇腾/寒武纪/天数智芯等) 信息
# 用法: 
#   get_npu_info.sh huawei   - 获取华为昇腾 NPU 信息
#   get_npu_info.sh cambricon - 获取寒武纪 MLU 信息
#   get_npu_info.sh iluvatar  - 获取天数智芯 GPU 信息
#   get_npu_info.sh count     - 获取所有 NPU/TPU 的总数量
#   get_npu_info.sh all       - 自动检测并获取所有 NPU 信息

get_huawei_npu() {
    # 华为昇腾 NPU 使用 npu-smi 命令
    if command -v npu-smi &>/dev/null; then
        # 获取版本信息
        version=$(npu-smi info 2>/dev/null | grep -i "Version:" | head -1 | awk -F': ' '{print $2}' | tr -d ' |')
        
        # 获取 NPU 数量 - 通过解析 npu-smi info 输出
        # 格式: | 0     910B3     OK ...
        count=$(npu-smi info 2>/dev/null | grep -E '^\|\s*[0-9]+\s+' | grep -v "NPU" | grep -v "Chip" | wc -l)
        
        # 获取型号
        model=$(npu-smi info 2>/dev/null | grep -E '^\|\s*[0-9]+\s+' | head -1 | awk '{print $2}')
        
        # 获取使用率和显存信息
        utilization=0
        memory_used=0
        memory_total=0
        
        # 尝试获取详细信息
        if npu-smi info -t common 2>/dev/null | grep -qi "aicore"; then
            utilization=$(npu-smi info -t common 2>/dev/null | grep -i "aicore" | head -1 | awk '{print $NF}' | tr -d '%')
        fi
        
        echo "huawei|${version:-N/A}|${model:-N/A}|${count:-0}|${utilization:-0}|${memory_used:-0}|${memory_total:-0}"
    else
        echo "huawei|not_installed|N/A|0|0|0|0"
    fi
}

get_cambricon_mlu() {
    # 寒武纪 MLU 使用 cnmon 命令
    if command -v cnmon &>/dev/null; then
        version=$(cnmon info 2>/dev/null | grep -i "driver version" | head -1 | awk -F': ' '{print $2}')
        count=$(cnmon info 2>/dev/null | grep -c "MLU" 2>/dev/null || echo "0")
        model=$(cnmon info 2>/dev/null | grep -i "Product Name" | head -1 | awk -F': ' '{print $2}')
        
        # 获取使用率
        utilization=$(cnmon info 2>/dev/null | grep -i "Board Util" | head -1 | awk '{print $NF}' | tr -d '%')
        memory_used=$(cnmon info 2>/dev/null | grep -i "Memory Used" | head -1 | awk '{print $NF}')
        memory_total=$(cnmon info 2>/dev/null | grep -i "Memory Total" | head -1 | awk '{print $NF}')
        
        echo "cambricon|${version:-N/A}|${model:-N/A}|${count:-0}|${utilization:-0}|${memory_used:-0}|${memory_total:-0}"
    else
        echo "cambricon|not_installed|N/A|0|0|0|0"
    fi
}

get_iluvatar_gpu() {
    # 天数智芯 GPU 使用 ixsmi 命令
    if command -v ixsmi &>/dev/null; then
        version=$(ixsmi -q 2>/dev/null | grep -i "driver version" | head -1 | awk -F': ' '{print $2}')
        count=$(ixsmi -L 2>/dev/null | wc -l 2>/dev/null || echo "0")
        model=$(ixsmi -q 2>/dev/null | grep -i "Product Name" | head -1 | awk -F': ' '{print $2}')
        
        echo "iluvatar|${version:-N/A}|${model:-N/A}|${count:-0}|0|0|0"
    else
        echo "iluvatar|not_installed|N/A|0|0|0|0"
    fi
}

get_total_count() {
    total=0
    
    # 华为昇腾
    if command -v npu-smi &>/dev/null; then
        huawei_count=$(npu-smi info 2>/dev/null | grep -E '^\|\s*[0-9]+\s+' | grep -v "NPU" | grep -v "Chip" | wc -l)
        total=$((total + huawei_count))
    fi
    
    # 寒武纪
    if command -v cnmon &>/dev/null; then
        cambricon_count=$(cnmon info 2>/dev/null | grep -c "MLU" 2>/dev/null || echo "0")
        total=$((total + cambricon_count))
    fi
    
    # 天数智芯
    if command -v ixsmi &>/dev/null; then
        iluvatar_count=$(ixsmi -L 2>/dev/null | wc -l 2>/dev/null || echo "0")
        total=$((total + iluvatar_count))
    fi
    
    echo "$total"
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
    count)
        get_total_count
        ;;
    all)
        # 自动检测并输出所有 NPU 信息
        result=""
        
        # 检测华为昇腾
        if command -v npu-smi &>/dev/null; then
            huawei_result=$(get_huawei_npu)
            if [[ ! "$huawei_result" =~ "not_installed" ]]; then
                result="${result}${huawei_result}\n"
            fi
        fi
        
        # 检测寒武纪
        if command -v cnmon &>/dev/null; then
            cambricon_result=$(get_cambricon_mlu)
            if [[ ! "$cambricon_result" =~ "not_installed" ]]; then
                result="${result}${cambricon_result}\n"
            fi
        fi
        
        # 检测天数智芯
        if command -v ixsmi &>/dev/null; then
            iluvatar_result=$(get_iluvatar_gpu)
            if [[ ! "$iluvatar_result" =~ "not_installed" ]]; then
                result="${result}${iluvatar_result}\n"
            fi
        fi
        
        if [ -z "$result" ]; then
            echo "none|not_installed|N/A|0|0|0|0"
        else
            echo -e "$result" | grep -v '^$'
        fi
        ;;
esac
