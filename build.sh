#!/bin/bash

# AI-Infra-Matrix 简化构建脚本
# 这是 scripts/all-ops.sh 的别名，便于用户使用

set -euo pipefail

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 检查 all-ops.sh 是否存在
ALL_OPS_SCRIPT="$SCRIPT_DIR/scripts/all-ops.sh"

if [ ! -f "$ALL_OPS_SCRIPT" ]; then
    echo "❌ 错误: 找不到 $ALL_OPS_SCRIPT"
    exit 1
fi

# 确保脚本可执行
chmod +x "$ALL_OPS_SCRIPT"

# 传递所有参数给 all-ops.sh
exec "$ALL_OPS_SCRIPT" "$@"
