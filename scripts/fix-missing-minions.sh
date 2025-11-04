#!/bin/bash

# 修复缺失的 Salt Minion 安装
set -e

APPHUB_URL="${1:-http://192.168.0.200:53434}"
SALT_MASTER="saltstack"

echo "=== 修复缺失的 Salt Minion ==="
echo "AppHub URL: $APPHUB_URL"
echo "Salt Master: $SALT_MASTER"
echo

# 定义需要修复的节点
UBUNTU_NODES=("test-ssh02")
ROCKY_NODES=("test-rocky01" "test-rocky02" "test-rocky03")

# 修复 Ubuntu 节点
for node in "${UBUNTU_NODES[@]}"; do
    echo "=== 修复 $node (Ubuntu) ==="
    
    docker exec $node bash -c "
        set -e
        export DEBIAN_FRONTEND=noninteractive
        
        # 下载并执行安装脚本
        curl -fsSL ${APPHUB_URL}/scripts/install-salt-minion-deb.sh -o /tmp/install-salt-minion.sh
        chmod +x /tmp/install-salt-minion.sh
        
        # 执行安装
        bash /tmp/install-salt-minion.sh '${APPHUB_URL}' 'v3007.8'
        
        # 配置 Minion
        mkdir -p /etc/salt
        cat > /etc/salt/minion <<EOF
master: ${SALT_MASTER}
id: ${node}
log_level: info
master_port: 4506
EOF
        
        # 启动服务
        systemctl daemon-reload
        systemctl enable salt-minion
        systemctl start salt-minion
        systemctl status salt-minion --no-pager
    "
    
    echo "  ✅ $node 修复完成"
    echo
done

# 修复 Rocky 节点
for node in "${ROCKY_NODES[@]}"; do
    echo "=== 修复 $node (Rocky) ==="
    
    docker exec $node bash -c "
        set -e
        
        # 下载并执行安装脚本
        curl -fsSL ${APPHUB_URL}/scripts/install-salt-minion-rpm.sh -o /tmp/install-salt-minion.sh
        chmod +x /tmp/install-salt-minion.sh
        
        # 执行安装
        bash /tmp/install-salt-minion.sh '${APPHUB_URL}' 'v3007.8'
        
        # 配置已经存在，启动服务
        systemctl daemon-reload
        systemctl enable salt-minion
        systemctl start salt-minion
        systemctl status salt-minion --no-pager
    "
    
    echo "  ✅ $node 修复完成"
    echo
done

echo "=== 等待 Minion 连接到 Master ==="
sleep 5

echo "=== 在 Salt Master 上接受所有 keys ==="
docker exec ai-infra-saltstack bash -c "salt-key -A -y"

echo
echo "=== 最终状态 ==="
docker exec ai-infra-saltstack bash -c "salt-key -L"

echo
echo "=== 测试连接 ==="
docker exec ai-infra-saltstack bash -c "salt '*' test.ping"

echo
echo "✅ 修复完成！所有节点现在应该都在 Salt 集群中了。"
