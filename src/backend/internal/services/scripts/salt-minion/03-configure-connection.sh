#!/bin/bash
# ====================================================================
# Salt Minion 连接配置脚本
# ====================================================================
# 描述: 配置 salt-minion 连接到 SaltStack Master
# 参数 (通过环境变量传入):
#   SALT_MASTER_HOST - Master 主机地址 (必需)
#   SALT_MINION_ID   - Minion ID (必需)
# ====================================================================

set -e

SALT_MASTER_HOST="${SALT_MASTER_HOST:-}"
SALT_MINION_ID="${SALT_MINION_ID:-}"

if [ -z "$SALT_MASTER_HOST" ]; then
    echo "[Salt] ✗ 错误: SALT_MASTER_HOST 未设置"
    exit 1
fi

if [ -z "$SALT_MINION_ID" ]; then
    echo "[Salt] ✗ 错误: SALT_MINION_ID 未设置"
    exit 1
fi

echo "=== 配置 Salt Minion 连接 ==="
echo "Master Host: $SALT_MASTER_HOST"
echo "Minion ID: $SALT_MINION_ID"

# 确保配置目录存在
mkdir -p /etc/salt

# 清理旧的 minion.d 配置文件（避免与主配置冲突）
echo "清理旧配置文件..."
rm -f /etc/salt/minion.d/99-master-address.conf 2>/dev/null || true
rm -f /etc/salt/minion.d/00-minion-id.conf 2>/dev/null || true
rm -f /etc/salt/minion.d/master.conf 2>/dev/null || true

# 清理旧的 PKI 密钥（确保使用新的 master 密钥）
echo "清理旧 PKI 密钥..."
rm -rf /etc/salt/pki/minion/* 2>/dev/null || true

# 生成主配置文件 /etc/salt/minion
echo "写入配置文件 /etc/salt/minion..."
cat > /etc/salt/minion <<EOF
# Salt Minion 配置
# 由 AI-Infra-Matrix 自动生成

# Master 连接配置
master: $SALT_MASTER_HOST
id: $SALT_MINION_ID

# 连接配置
master_alive_interval: 30
master_tries: -1
acceptance_wait_time: 10
random_reauth_delay: 3

# Mine 配置
mine_enabled: true
mine_return_job: true
mine_interval: 60
EOF

chmod 640 /etc/salt/minion

echo "=== Salt Minion 配置完成 ==="
echo "配置文件内容:"
cat /etc/salt/minion

exit 0
