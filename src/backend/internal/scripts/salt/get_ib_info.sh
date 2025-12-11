#!/bin/bash
# get_ib_info.sh - 获取 InfiniBand 信息
# 用法:
#   get_ib_info.sh check  - 检查 ibstat 命令是否存在
#   get_ib_info.sh status - 获取 IB 状态信息
#   get_ib_info.sh all    - 获取所有 IB 信息

case "${1:-all}" in
    check)
        which ibstat 2>/dev/null || command -v ibstat 2>/dev/null
        ;;
    status)
        ibstat 2>/dev/null
        ;;
    all)
        if which ibstat >/dev/null 2>&1 || command -v ibstat >/dev/null 2>&1; then
            ibstat 2>/dev/null
        else
            echo "ibstat not found"
        fi
        ;;
esac
