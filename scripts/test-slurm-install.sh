#!/bin/bash
#
# 测试SLURM节点安装脚本
# 用于手动验证安装逻辑是否正确
#

set -e

echo "=========================================="
echo "SLURM节点安装测试脚本"
echo "=========================================="
echo

# 配置
APPHUB_URL="http://ai-infra-apphub"
TEST_NODES="test-rocky02 test-rocky03 test-ssh02 test-ssh03"

# 读取安装脚本
SCRIPT_PATH="src/backend/scripts/install-slurm-node.sh"
if [ ! -f "$SCRIPT_PATH" ]; then
    echo "错误: 找不到安装脚本 $SCRIPT_PATH"
    exit 1
fi

echo "使用安装脚本: $SCRIPT_PATH"
echo "AppHub URL: $APPHUB_URL"
echo

for node in $TEST_NODES; do
    echo "=========================================="
    echo "测试节点: $node"
    echo "=========================================="
    
    # 1. 检查节点是否在线
    if ! docker ps | grep -q "$node"; then
        echo "警告: 节点 $node 不在线，跳过"
        continue
    fi
    
    # 2. 上传安装脚本到节点
    echo "[1/4] 上传安装脚本到节点..."
    docker cp "$SCRIPT_PATH" "$node:/tmp/install-slurm-test.sh"
    docker exec "$node" chmod +x /tmp/install-slurm-test.sh
    
    # 3. 执行安装脚本
    echo "[2/4] 执行安装脚本..."
    docker exec "$node" /tmp/install-slurm-test.sh "$APPHUB_URL" compute 2>&1 | tail -30
    
    # 4. 验证安装
    echo "[3/4] 验证SLURM包安装..."
    docker exec "$node" bash -c "
        if command -v rpm >/dev/null 2>&1; then
            rpm -qa | grep -E 'slurm|munge' || echo '未找到SLURM/Munge包 (RPM)'
        elif command -v dpkg >/dev/null 2>&1; then
            dpkg -l | grep -E 'slurm|munge' | awk '{print \$2, \$3}' || echo '未找到SLURM/Munge包 (DEB)'
        fi
    "
    
    # 5. 检查服务状态
    echo "[4/4] 检查服务状态..."
    docker exec "$node" bash -c "ps aux | grep -E 'slurmd|munged' | grep -v grep || echo '服务未运行'"
    
    echo "节点 $node 测试完成"
    echo
done

echo "=========================================="
echo "测试完成"
echo "=========================================="
