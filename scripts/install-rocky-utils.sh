#!/bin/bash
# 为 Rocky Linux 测试容器安装基础诊断工具
# Usage: ./scripts/install-rocky-utils.sh

set -e

ROCKY_NODES=("test-rocky01" "test-rocky02" "test-rocky03")

echo "🔧 为 Rocky Linux 节点安装基础诊断工具..."
echo ""

for node in "${ROCKY_NODES[@]}"; do
    echo "=== 📦 处理节点: $node ==="
    
    # 检查容器是否运行
    if ! docker ps --format '{{.Names}}' | grep -q "^${node}$"; then
        echo "⚠️  容器 $node 未运行，跳过"
        echo ""
        continue
    fi
    
    # 安装基础工具
    echo "  → 安装 procps-ng (提供 ps, top, free 等命令)..."
    docker exec "$node" dnf install -y procps-ng 2>&1 | grep -E "(Installing|Installed|Already installed|Complete!)" || true
    
    echo "  → 安装 iproute (提供 ss, ip 命令)..."
    docker exec "$node" dnf install -y iproute 2>&1 | grep -E "(Installing|Installed|Already installed|Complete!)" || true
    
    echo "  → 安装 net-tools (提供 ifconfig, netstat 等命令)..."
    docker exec "$node" dnf install -y net-tools 2>&1 | grep -E "(Installing|Installed|Already installed|Complete!)" || true
    
    echo "  → 安装 bind-utils (提供 nslookup, dig 等 DNS 工具)..."
    docker exec "$node" dnf install -y bind-utils 2>&1 | grep -E "(Installing|Installed|Already installed|Complete!)" || true
    
    echo "  → 安装 vim (文本编辑器)..."
    docker exec "$node" dnf install -y vim-minimal 2>&1 | grep -E "(Installing|Installed|Already installed|Complete!)" || true
    
    echo "  → 安装 wget 和 curl (下载工具)..."
    docker exec "$node" dnf install -y wget curl 2>&1 | grep -E "(Installing|Installed|Already installed|Complete!)" || true
    
    echo "  ✓ $node 工具安装完成"
    echo ""
done

echo "🎉 所有 Rocky Linux 节点工具安装完成！"
echo ""
echo "验证安装："
echo "  for node in test-rocky01 test-rocky02 test-rocky03; do"
echo "    echo \"=== \$node ===\";"
echo "    docker exec \$node bash -c 'ps aux | head -3';"
echo "    docker exec \$node bash -c 'ip addr show | grep inet';"
echo "    echo;"
echo "  done"
