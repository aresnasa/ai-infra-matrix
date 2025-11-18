#!/bin/bash
# ====================================================================
# Salt Minion 配置脚本
# ====================================================================
# 描述: 配置salt-minion连接到SaltStack Master
# 参数:
#   SALT_MASTER_HOST - Master主机地址 (必需)
#   SALT_MINION_ID   - Minion ID (可选，默认使用主机名)
# ====================================================================

set -eo pipefail

SALT_MASTER_HOST="${SALT_MASTER_HOST:-}"
SALT_MINION_ID="${SALT_MINION_ID:-}"

if [ -z "$SALT_MASTER_HOST" ]; then
	echo "[Salt] ✗ 错误: SALT_MASTER_HOST 未设置"
	exit 1
fi

echo "=== 配置 Salt Minion ==="
echo "Master Host: $SALT_MASTER_HOST"
echo "Minion ID: ${SALT_MINION_ID:-自动检测}"

# 创建配置目录
mkdir -p /etc/salt/minion.d

# 生成配置文件
cat > /etc/salt/minion.d/99-master-address.conf <<EOF
# Salt Master 配置
master: $SALT_MASTER_HOST

# 自动接受密钥 (生产环境建议关闭)
master_alive_interval: 30
master_tries: -1
acceptance_wait_time: 10
random_reauth_delay: 3
EOF

# 如果指定了 Minion ID，设置它
if [ -n "$SALT_MINION_ID" ]; then
	echo "id: $SALT_MINION_ID" > /etc/salt/minion.d/00-minion-id.conf
	echo "[Salt] Minion ID 设置为: $SALT_MINION_ID"
fi

echo "=== Salt Minion 配置完成 ==="
cat /etc/salt/minion.d/99-master-address.conf

exit 0
