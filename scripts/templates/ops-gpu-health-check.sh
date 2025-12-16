#!/bin/bash
# =============================================================================
# GPU 健康深度检查脚本
# 用法: ./ops-gpu-health-check.sh [--expected-gpus N]
# =============================================================================

EXPECTED_GPUS=${EXPECTED_GPUS:-8}

# 参数解析
while [[ $# -gt 0 ]]; do
    case $1 in
        --expected-gpus) EXPECTED_GPUS="$2"; shift 2 ;;
        *) shift ;;
    esac
done

echo "=== GPU 健康深度检查 ==="
echo "主机: $(hostname)"
echo "时间: $(date)"
echo ""

if ! command -v nvidia-smi &> /dev/null; then
    echo "❌ 未检测到 nvidia-smi，可能未安装 NVIDIA 驱动"
    exit 1
fi

# 1. 驱动和 CUDA 版本
echo "【1. 驱动信息】"
nvidia-smi --query-gpu=driver_version,cuda_version --format=csv
echo ""

# 2. GPU 列表
echo "【2. GPU 列表】"
nvidia-smi -L
GPU_COUNT=$(nvidia-smi -L | wc -l)
echo "总计: $GPU_COUNT 块 GPU"
if [ "$GPU_COUNT" -lt "$EXPECTED_GPUS" ]; then
    echo "⚠️ 警告: 预期 $EXPECTED_GPUS 块 GPU，实际检测到 $GPU_COUNT 块，可能存在掉卡！"
fi
echo ""

# 3. GPU 温度和功耗
echo "【3. GPU 温度/功耗/风扇】"
nvidia-smi --query-gpu=index,name,temperature.gpu,power.draw,fan.speed --format=csv
echo ""

# 4. GPU 利用率和显存
echo "【4. GPU 利用率/显存】"
nvidia-smi --query-gpu=index,utilization.gpu,utilization.memory,memory.used,memory.total --format=csv
echo ""

# 5. ECC 错误检查
echo "【5. ECC 错误检查】"
nvidia-smi --query-gpu=index,ecc.errors.corrected.volatile.total,ecc.errors.uncorrected.volatile.total --format=csv 2>/dev/null || echo "ECC 信息不可用"
echo ""

# 检查 ECC 错误
ECC_ERRORS=$(nvidia-smi --query-gpu=ecc.errors.uncorrected.volatile.total --format=csv,noheader 2>/dev/null | awk '{sum+=$1}END{print sum}')
if [ -n "$ECC_ERRORS" ] && [ "$ECC_ERRORS" -gt 0 ]; then
    echo "⚠️ 警告: 检测到 $ECC_ERRORS 个不可纠正的 ECC 错误！"
fi

# 6. PCIe 带宽
echo "【6. PCIe 信息】"
nvidia-smi --query-gpu=index,pcie.link.gen.current,pcie.link.gen.max,pcie.link.width.current --format=csv
echo ""

# 7. 持久模式和计算模式
echo "【7. GPU 模式设置】"
nvidia-smi --query-gpu=index,persistence_mode,compute_mode --format=csv
echo ""

# 8. 运行中的进程
echo "【8. GPU 上运行的进程】"
nvidia-smi --query-compute-apps=pid,name,gpu_bus_id,used_memory --format=csv 2>/dev/null || echo "无运行中的 GPU 进程"
echo ""

# 9. XID 错误检查
echo "【9. XID 错误检查 (最近24小时)】"
XID_COUNT=$(dmesg -T 2>/dev/null | grep -c "NVRM: Xid" || echo "0")
if [ "$XID_COUNT" -gt 0 ]; then
    echo "⚠️ 发现 $XID_COUNT 条 XID 错误日志:"
    dmesg -T 2>/dev/null | grep "NVRM: Xid" | tail -10
else
    echo "✅ 无 XID 错误"
fi
echo ""

# 10. 常见 XID 错误解释
if [ "$XID_COUNT" -gt 0 ]; then
    echo "【常见 XID 错误代码参考】"
    echo "XID 13: Graphics Engine Exception (显存问题或驱动bug)"
    echo "XID 31: GPU memory page fault (GPU 显存页错误)"
    echo "XID 43: GPU stopped processing (GPU 停止处理)"
    echo "XID 45: Preemptive cleanup, due to previous errors (前序错误导致的清理)"
    echo "XID 48: Double Bit ECC Error (ECC 双比特错误，需要重启)"
    echo "XID 63: ECC page retirement (ECC 页面退役)"
    echo "XID 64: ECC page retirement or row remapping failure (ECC 退役/重映射失败)"
    echo "XID 74: NVLink Error (NVLink 连接错误)"
    echo "XID 79: GPU has fallen off the bus (GPU 掉卡)"
    echo ""
fi

# 11. 健康评估
echo "【10. 健康评估总结】"
ISSUES=0
# 温度检查
HIGH_TEMP=$(nvidia-smi --query-gpu=temperature.gpu --format=csv,noheader | awk '$1>85{count++}END{print count+0}')
if [ "$HIGH_TEMP" -gt 0 ]; then
    echo "⚠️ $HIGH_TEMP 块 GPU 温度过高 (>85°C)"
    ISSUES=$((ISSUES+1))
fi
# ECC 检查
if [ -n "$ECC_ERRORS" ] && [ "$ECC_ERRORS" -gt 0 ]; then
    echo "⚠️ 存在不可纠正的 ECC 错误"
    ISSUES=$((ISSUES+1))
fi
# XID 检查
if [ "$XID_COUNT" -gt 0 ]; then
    echo "⚠️ 存在 XID 错误日志"
    ISSUES=$((ISSUES+1))
fi
# GPU 数量检查
if [ "$GPU_COUNT" -lt "$EXPECTED_GPUS" ]; then
    echo "⚠️ GPU 数量不足，预期 $EXPECTED_GPUS，实际 $GPU_COUNT"
    ISSUES=$((ISSUES+1))
fi

if [ "$ISSUES" -eq 0 ]; then
    echo "✅ GPU 健康状态良好"
else
    echo "❌ 发现 $ISSUES 类问题，请检查"
fi

echo ""
echo "=== GPU 健康检查完成 ==="
