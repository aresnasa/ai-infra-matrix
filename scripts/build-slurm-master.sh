#!/bin/bash
#
# AI Infrastructure Matrix - SLURM Master 构建脚本
# 确保从AppHub安装SLURM以保持版本一致性
#

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

cd "$PROJECT_ROOT"

echo "========================================="
echo "🚀 构建 SLURM Master 镜像"
echo "========================================="
echo ""

# 检查docker-compose是否可用
if ! command -v docker-compose &> /dev/null; then
    echo "❌ 错误: docker-compose 未安装"
    exit 1
fi

# 检查AppHub是否运行
echo "📋 步骤 1/3: 检查AppHub服务状态..."
if docker ps | grep -q "ai-infra-apphub"; then
    echo "✅ AppHub服务正在运行"
else
    echo "⚠️  AppHub服务未运行，正在启动..."
    docker-compose up -d apphub
    echo "⏳ 等待AppHub服务就绪 (10秒)..."
    sleep 10
    
    # 再次检查
    if docker ps | grep -q "ai-infra-apphub"; then
        echo "✅ AppHub服务已启动"
    else
        echo "❌ 错误: AppHub服务启动失败"
        echo "💡 请检查: docker-compose logs apphub"
        exit 1
    fi
fi

# 测试AppHub连接
echo ""
echo "📋 步骤 2/3: 测试AppHub连接..."

# 从.env文件读取配置
if [ -f "$PROJECT_ROOT/.env" ]; then
    source "$PROJECT_ROOT/.env"
fi

EXTERNAL_HOST="${EXTERNAL_HOST:-localhost}"
APPHUB_PORT="${APPHUB_PORT:-8088}"
APPHUB_URL="http://${EXTERNAL_HOST}:${APPHUB_PORT}"

echo "📌 AppHub URL: $APPHUB_URL"

if curl -s --max-time 5 "${APPHUB_URL}/pkgs/slurm-deb/Packages" > /dev/null 2>&1; then
    echo "✅ AppHub可访问"
    
    # 显示可用的SLURM包
    PACKAGE_COUNT=$(curl -s "${APPHUB_URL}/pkgs/slurm-deb/Packages" | grep -c "^Package: slurm" || echo "0")
    if [ "$PACKAGE_COUNT" -gt 0 ]; then
        echo "✅ 发现 $PACKAGE_COUNT 个SLURM包"
    else
        echo "⚠️  警告: AppHub中未找到SLURM包"
        echo "💡 请确保AppHub已构建SLURM包"
    fi
else
    echo "❌ 错误: 无法连接到AppHub ($APPHUB_URL)"
    echo "💡 请检查AppHub服务是否正常运行"
    exit 1
fi

# 构建slurm-master镜像
echo ""
echo "📋 步骤 3/3: 构建slurm-master镜像..."
echo "⏳ 这可能需要几分钟时间..."
echo "📌 构建参数:"
echo "   - APPHUB_URL: $APPHUB_URL"
echo "   - 网络模式: host (允许访问宿主机上的AppHub服务)"
echo ""

if docker-compose build slurm-master; then
    echo ""
    echo "========================================="
    echo "✅ SLURM Master 镜像构建成功"
    echo "========================================="
    echo ""
    echo "📦 验证安装..."
    
    # 创建临时容器验证SLURM版本
    TEMP_CONTAINER="slurm-master-verify-$$"
    docker run --rm --name "$TEMP_CONTAINER" \
        $(docker-compose config | grep "image:" | grep slurm-master | awk '{print $2}') \
        slurmctld -V 2>/dev/null || echo "⚠️  无法验证SLURM版本"
    
    echo ""
    echo "💡 下一步:"
    echo "   docker-compose up -d slurm-master"
    echo ""
else
    echo ""
    echo "========================================="
    echo "❌ SLURM Master 镜像构建失败"
    echo "========================================="
    echo ""
    echo "💡 故障排查:"
    echo "   1. 检查AppHub服务日志: docker-compose logs apphub"
    echo "   2. 检查构建日志以获取详细错误信息"
    echo "   3. 确保AppHub中有SLURM 25.05.4包"
    echo ""
    exit 1
fi
