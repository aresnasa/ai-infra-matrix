#!/bin/bash
#
# 手动完整安装SLURM节点（模拟InstallSlurmNode函数的完整流程）
# 用于测试和验证安装逻辑
#

set -e

echo "=========================================="
echo "手动SLURM节点完整安装"
echo "=========================================="
echo

# 配置
APPHUB_URL="http://ai-infra-apphub"
SLURM_MASTER="ai-infra-slurm-master"
NODE_NAME="${1:-test-rocky02}"

echo "节点名称: $NODE_NAME"
echo "SLURM Master: $SLURM_MASTER"
echo "AppHub URL: $APPHUB_URL"
echo

# 步骤1: 获取配置文件
echo "=========================================="
echo "[1/5] 获取配置文件"
echo "=========================================="

echo "从 $SLURM_MASTER 获取 slurm.conf..."
docker exec "$SLURM_MASTER" cat /etc/slurm/slurm.conf > /tmp/slurm.conf.$$
echo "✓ slurm.conf 已保存到 /tmp/slurm.conf.$$"

echo "从 $SLURM_MASTER 获取 munge.key..."
docker exec "$SLURM_MASTER" cat /etc/munge/munge.key > /tmp/munge.key.$$
echo "✓ munge.key 已保存到 /tmp/munge.key.$$"
echo

# 步骤2: 安装SLURM包
echo "=========================================="
echo "[2/5] 安装SLURM包"
echo "=========================================="

# 检测操作系统类型
OS_TYPE=$(docker exec "$NODE_NAME" bash -c "
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        if [[ \$ID == 'ubuntu' || \$ID == 'debian' ]]; then
            echo 'ubuntu'
        else
            echo 'rocky'
        fi
    else
        echo 'unknown'
    fi
")

echo "检测到操作系统类型: $OS_TYPE"

# 上传并执行安装脚本
echo "上传安装脚本到节点..."
docker cp src/backend/scripts/install-slurm-node.sh "$NODE_NAME:/tmp/install-slurm-test.sh"
docker exec "$NODE_NAME" chmod +x /tmp/install-slurm-test.sh

echo "执行安装脚本..."
docker exec "$NODE_NAME" /tmp/install-slurm-test.sh "$APPHUB_URL" compute
echo "✓ SLURM包安装完成"
echo

# 步骤3: 部署配置文件
echo "=========================================="
echo "[3/5] 部署配置文件"
echo "=========================================="

if [ "$OS_TYPE" = "ubuntu" ]; then
    SLURM_CONF_PATH="/etc/slurm-llnl/slurm.conf"
else
    SLURM_CONF_PATH="/etc/slurm/slurm.conf"
fi

echo "部署 slurm.conf 到 $SLURM_CONF_PATH..."
docker cp /tmp/slurm.conf.$$ "$NODE_NAME:$SLURM_CONF_PATH"
docker exec "$NODE_NAME" chmod 644 "$SLURM_CONF_PATH"
echo "✓ slurm.conf 部署完成"

echo "部署 munge.key 到 /etc/munge/munge.key..."
docker cp /tmp/munge.key.$$ "$NODE_NAME:/etc/munge/munge.key"
docker exec "$NODE_NAME" bash -c "
    chown munge:munge /etc/munge/munge.key
    chmod 400 /etc/munge/munge.key
"
echo "✓ munge.key 部署完成"
echo

# 步骤4: 启动服务
echo "=========================================="
echo "[4/5] 启动服务"
echo "=========================================="

echo "执行启动脚本..."
docker cp src/backend/scripts/start-slurmd.sh "$NODE_NAME:/tmp/start-slurmd-test.sh"
docker exec "$NODE_NAME" chmod +x /tmp/start-slurmd-test.sh
docker exec "$NODE_NAME" /tmp/start-slurmd-test.sh
echo "✓ 服务启动脚本执行完成"
echo

# 步骤5: 验证安装
echo "=========================================="
echo "[5/5] 验证安装"
echo "=========================================="

echo "检查安装的包..."
docker exec "$NODE_NAME" bash -c "
    if command -v rpm >/dev/null 2>&1; then
        rpm -qa | grep -E 'slurm|munge'
    elif command -v dpkg >/dev/null 2>&1; then
        dpkg -l | grep -E 'slurm|munge' | awk '{print \$2, \$3}'
    fi
"
echo

echo "检查服务进程..."
docker exec "$NODE_NAME" bash -c "ps aux | egrep 'slurmd|munged' | grep -v grep || echo '警告: 服务未运行'"
echo

echo "检查munge认证..."
docker exec "$NODE_NAME" bash -c "
    if command -v munge >/dev/null 2>&1 && pgrep munged >/dev/null; then
        if munge -n | unmunge >/dev/null 2>&1; then
            echo '✓ Munge认证测试通过'
        else
            echo '✗ Munge认证测试失败'
        fi
    else
        echo '✗ Munge服务未运行'
    fi
"
echo

echo "检查slurmd进程详情..."
docker exec "$NODE_NAME" bash -c "pgrep -a slurmd || echo '无slurmd进程'"
echo

# 清理临时文件
rm -f /tmp/slurm.conf.$$ /tmp/munge.key.$$

echo "=========================================="
echo "安装完成！"
echo "=========================================="
echo
echo "下一步："
echo "1. 在SLURM Master上执行: scontrol update nodename=$NODE_NAME state=resume"
echo "2. 检查节点状态: sinfo -Nel | grep $NODE_NAME"
