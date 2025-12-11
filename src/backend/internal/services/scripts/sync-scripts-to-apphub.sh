#!/bin/bash
# sync-scripts-to-apphub.sh
# 将安装脚本同步到AppHub服务器
# 在backend容器启动时自动执行

set -e

APPHUB_HOST="${APPHUB_HOST:-ai-infra-apphub}"
APPHUB_SCRIPTS_DIR="${APPHUB_SCRIPTS_DIR:-/usr/share/nginx/html/scripts}"
LOCAL_SCRIPTS_DIR="/root/scripts"

echo "[INFO] 同步安装脚本到 AppHub ($APPHUB_HOST)..."

# 检查AppHub是否可达
if ! nc -z "$APPHUB_HOST" 80 2>/dev/null; then
    echo "[WARN] AppHub ($APPHUB_HOST:80) 不可达，跳过脚本同步"
    exit 0
fi

# 方式1: 使用docker cp（推荐，因为backend有docker socket访问权限）
if command -v docker &>/dev/null; then
    echo "[INFO] 使用 docker cp 同步脚本..."
    
    # 检查AppHub容器是否存在
    if docker ps --format '{{.Names}}' | grep -q "^${APPHUB_HOST}$"; then
        # 确保目标目录存在
        docker exec "$APPHUB_HOST" mkdir -p "$APPHUB_SCRIPTS_DIR" 2>/dev/null || true
        
        # 复制所有.sh脚本
        for script in "$LOCAL_SCRIPTS_DIR"/*.sh; do
            if [ -f "$script" ]; then
                script_name=$(basename "$script")
                echo "[INFO] 复制 $script_name 到 AppHub..."
                docker cp "$script" "${APPHUB_HOST}:${APPHUB_SCRIPTS_DIR}/"
                docker exec "$APPHUB_HOST" chmod 644 "${APPHUB_SCRIPTS_DIR}/${script_name}"
            fi
        done
        
        echo "[INFO] ✓ 脚本同步完成（使用 docker cp）"
        
        # 验证
        echo "[INFO] AppHub 上的脚本列表:"
        docker exec "$APPHUB_HOST" ls -lh "$APPHUB_SCRIPTS_DIR/" || true
        
        exit 0
    else
        echo "[WARN] 找不到 AppHub 容器: $APPHUB_HOST，尝试SSH方式..."
    fi
fi

# 方式2: 使用SCP/SSH（备选方案，需要SSH互信）
if command -v scp &>/dev/null && [ -f ~/.ssh/id_rsa ]; then
    echo "[INFO] 使用 SCP 同步脚本..."
    
    # 检查SSH连接
    if ssh -o ConnectTimeout=5 -o StrictHostKeyChecking=no root@"$APPHUB_HOST" "mkdir -p $APPHUB_SCRIPTS_DIR" 2>/dev/null; then
        # SSH连接成功，使用SCP复制
        for script in "$LOCAL_SCRIPTS_DIR"/*.sh; do
            if [ -f "$script" ]; then
                script_name=$(basename "$script")
                echo "[INFO] SCP复制 $script_name 到 AppHub..."
                scp -o StrictHostKeyChecking=no "$script" "root@${APPHUB_HOST}:${APPHUB_SCRIPTS_DIR}/"
                ssh -o StrictHostKeyChecking=no root@"$APPHUB_HOST" "chmod 644 ${APPHUB_SCRIPTS_DIR}/${script_name}"
            fi
        done
        
        echo "[INFO] ✓ 脚本同步完成（使用 SCP）"
        
        # 验证
        echo "[INFO] AppHub 上的脚本列表:"
        ssh -o StrictHostKeyChecking=no root@"$APPHUB_HOST" "ls -lh $APPHUB_SCRIPTS_DIR/" || true
        
        exit 0
    else
        echo "[WARN] SSH连接失败"
    fi
fi

echo "[WARN] 无法同步脚本到 AppHub（docker 和 SSH 都不可用）"
exit 0
