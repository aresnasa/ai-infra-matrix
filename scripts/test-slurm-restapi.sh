#!/bin/bash
#
# SLURM REST API 测试脚本
# 用于验证 SLURM REST API 是否正确安装并可用
#

set -e

CONTAINER_NAME="${1:-ai-infra-slurm-master}"

echo "=========================================="
echo "  SLURM REST API 安装测试"
echo "  容器: ${CONTAINER_NAME}"
echo "=========================================="
echo ""

# 检查容器是否运行
echo "[测试 1/7] 检查容器状态..."
if docker ps --format '{{.Names}}' | grep -q "^${CONTAINER_NAME}$"; then
    echo "  ✅ 容器正在运行"
else
    echo "  ❌ 容器未运行"
    exit 1
fi

# 检查 SLURM 包安装
echo ""
echo "[测试 2/7] 检查 SLURM 包安装..."
docker exec ${CONTAINER_NAME} bash -c '
if dpkg -l | grep -q slurm-smd; then
    echo "  ✅ SLURM SMD 包已安装"
    dpkg -l | grep slurm-smd | awk "{print \"    \", \$2, \$3}"
elif dpkg -l | grep -q slurm-wlm; then
    echo "  ✅ SLURM WLM 包已安装"
    dpkg -l | grep slurm-wlm | awk "{print \"    \", \$2, \$3}"
else
    echo "  ⚠️  未检测到 SLURM 包"
fi
'

# 检查 slurmrestd 二进制
echo ""
echo "[测试 3/7] 检查 slurmrestd 二进制..."
if docker exec ${CONTAINER_NAME} which slurmrestd &>/dev/null; then
    SLURMRESTD_PATH=$(docker exec ${CONTAINER_NAME} which slurmrestd)
    echo "  ✅ slurmrestd 已安装: ${SLURMRESTD_PATH}"
    docker exec ${CONTAINER_NAME} ${SLURMRESTD_PATH} -V 2>&1 | head -1 | sed 's/^/    /'
else
    echo "  ❌ slurmrestd 未找到"
fi

# 检查客户端工具
echo ""
echo "[测试 4/7] 检查 SLURM 客户端工具..."
for tool in sinfo squeue scontrol srun sbatch; do
    if docker exec ${CONTAINER_NAME} which ${tool} &>/dev/null; then
        echo "  ✅ ${tool} 可用"
    else
        echo "  ⚠️  ${tool} 未找到"
    fi
done

# 检查 JWT 配置
echo ""
echo "[测试 5/7] 检查 JWT 认证配置..."
if docker exec ${CONTAINER_NAME} test -f /var/spool/slurm/statesave/jwt_hs256.key; then
    echo "  ✅ JWT 密钥文件存在"
    docker exec ${CONTAINER_NAME} ls -lh /var/spool/slurm/statesave/jwt_hs256.key | sed 's/^/    /'
else
    echo "  ⚠️  JWT 密钥文件不存在"
fi

if docker exec ${CONTAINER_NAME} grep -q "AuthAltTypes=auth/jwt" /etc/slurm/slurm.conf 2>/dev/null; then
    echo "  ✅ slurm.conf 中 JWT 配置已启用"
else
    echo "  ⚠️  slurm.conf 中未找到 JWT 配置"
fi

# 检查 slurmctld 状态
echo ""
echo "[测试 6/7] 检查 slurmctld 服务状态..."
if docker exec ${CONTAINER_NAME} pgrep -f slurmctld &>/dev/null; then
    echo "  ✅ slurmctld 进程正在运行"
    docker exec ${CONTAINER_NAME} pgrep -af slurmctld | sed 's/^/    /'
else
    echo "  ⚠️  slurmctld 进程未运行"
fi

# 测试 REST API
echo ""
echo "[测试 7/7] 测试 SLURM REST API..."

# 首先尝试启动 slurmrestd（如果未运行）
if ! docker exec ${CONTAINER_NAME} pgrep -f slurmrestd &>/dev/null; then
    echo "  ℹ️  启动 slurmrestd..."
    docker exec -d ${CONTAINER_NAME} bash -c 'export SLURM_JWT=daemon && slurmrestd :6820' &>/dev/null || true
    sleep 3
fi

if docker exec ${CONTAINER_NAME} pgrep -f slurmrestd &>/dev/null; then
    echo "  ✅ slurmrestd 进程正在运行"
    docker exec ${CONTAINER_NAME} pgrep -af slurmrestd | sed 's/^/    /'
    
    # 尝试获取 JWT token 并调用 API
    echo ""
    echo "  🔑 获取 JWT token..."
    TOKEN_OUTPUT=$(docker exec ${CONTAINER_NAME} bash -c 'unset SLURM_JWT && scontrol token 2>/dev/null' || echo "")
    
    if [ -n "$TOKEN_OUTPUT" ]; then
        echo "  ✅ Token 获取成功"
        echo "$TOKEN_OUTPUT" | sed 's/^/    /'
        
        echo ""
        echo "  🌐 调用 REST API /slurm/v0.0.40/diag..."
        API_RESULT=$(docker exec ${CONTAINER_NAME} bash -c "
            export \$(scontrol token 2>/dev/null)
            if [ -n \"\$SLURM_JWT\" ]; then
                curl -s -H \"X-SLURM-USER-TOKEN:\$SLURM_JWT\" \
                    http://localhost:6820/slurm/v0.0.40/diag 2>/dev/null
            else
                echo '{\"error\": \"No token available\"}'
            fi
        " 2>/dev/null)
        
        if echo "$API_RESULT" | jq . &>/dev/null; then
            echo "  ✅ REST API 响应成功"
            echo "$API_RESULT" | jq -C '.' | head -20 | sed 's/^/    /'
        else
            echo "  ⚠️  REST API 响应异常"
            echo "$API_RESULT" | head -10 | sed 's/^/    /'
        fi
    else
        echo "  ⚠️  无法获取 JWT token"
    fi
else
    echo "  ❌ slurmrestd 未运行"
fi

echo ""
echo "=========================================="
echo "  测试完成"
echo "=========================================="
echo ""
echo "📝 手动测试命令："
echo "  # 进入容器"
echo "  docker exec -it ${CONTAINER_NAME} bash"
echo ""
echo "  # 获取 token"
echo "  export \$(scontrol token)"
echo ""
echo "  # 调用 API"
echo "  curl -H \"X-SLURM-USER-TOKEN:\$SLURM_JWT\" \\"
echo "    http://localhost:6820/slurm/v0.0.40/diag | jq ."
echo ""
