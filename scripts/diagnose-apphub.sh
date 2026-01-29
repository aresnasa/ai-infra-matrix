#!/usr/bin/env bash
# AppHub 诊断脚本 - 帮助诊断 "invalid host apphub:invalid IP" 问题

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "========================================="
echo "  AppHub 诊断工具"
echo "========================================="
echo ""

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_debug() { echo -e "${BLUE}[DEBUG]${NC} $1"; }

# 1. 检查 Docker 是否运行
echo "1️⃣  检查 Docker 服务..."
if ! docker ps >/dev/null 2>&1; then
    log_error "Docker 服务未运行或无权限访问"
    exit 1
fi
log_info "✓ Docker 服务正常"
echo ""

# 2. 检查 ai-infra-network 是否存在
echo "2️⃣  检查 ai-infra-network 网络..."
if docker network ls | grep -q "ai-infra-network"; then
    log_info "✓ 网络 ai-infra-network 存在"
else
    log_warn "⚠️  网络 ai-infra-network 不存在"
    echo "   尝试创建网络..."
    docker network create ai-infra-network 2>/dev/null || {
        log_error "创建网络失败"
        exit 1
    }
    log_info "✓ 网络已创建"
fi
echo ""

# 3. 检查 apphub 容器状态
echo "3️⃣  检查 AppHub 容器状态..."
APPHUB_RUNNING=$(docker ps --filter "name=^ai-infra-apphub$" --filter "status=running" --format "{{.ID}}" 2>/dev/null || echo "")

if [[ -z "$APPHUB_RUNNING" ]]; then
    log_warn "⚠️  AppHub 容器未运行"
    echo ""
    echo "   现有容器状态："
    docker ps -a --filter "name=apphub" 2>/dev/null || echo "   (无相关容器)"
    echo ""
    echo "   💡 建议：启动 AppHub 容器"
    echo "      docker-compose up -d apphub"
else
    log_info "✓ AppHub 容器正在运行 (ID: ${APPHUB_RUNNING:0:12})"
fi
echo ""

# 4. 检查 apphub IP 地址
echo "4️⃣  检查 AppHub IP 地址..."
if [[ -n "$APPHUB_RUNNING" ]]; then
    # 尝试从 ai-infra-network 获取 IP
    APPHUB_IP=$(docker inspect -f '{{index .NetworkSettings.Networks "ai-infra-network" .IPAddress}}' ai-infra-apphub 2>/dev/null || echo "")
    
    if [[ -n "$APPHUB_IP" ]]; then
        log_info "✓ AppHub IP: $APPHUB_IP (来自 ai-infra-network)"
        
        # 验证 IP 格式
        if [[ $APPHUB_IP =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]]; then
            log_info "✓ IP 地址格式有效"
        else
            log_error "❌ IP 地址格式无效: $APPHUB_IP"
        fi
    else
        log_warn "⚠️  无法获取 ai-infra-network 上的 IP"
        
        # 尝试获取所有网络中的 IP
        echo "   尝试获取其他网络上的 IP..."
        ALL_IPS=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}} {{end}}' ai-infra-apphub 2>/dev/null || echo "")
        if [[ -n "$ALL_IPS" ]]; then
            log_info "   其他网络上的 IP: $ALL_IPS"
        else
            log_error "   无法获取任何网络上的 IP"
        fi
    fi
else
    log_warn "⚠️  AppHub 容器未运行，跳过 IP 检查"
fi
echo ""

# 5. 检查 apphub 网络连接性
echo "5️⃣  检查 AppHub 网络连接..."
if [[ -n "$APPHUB_RUNNING" ]]; then
    # 检查端口是否开放
    APPHUB_PORT="${APPHUB_PORT:-28080}"
    if netstat -tuln 2>/dev/null | grep -q ":$APPHUB_PORT "; then
        log_info "✓ AppHub 端口 $APPHUB_PORT 已开放"
    else
        log_warn "⚠️  AppHub 端口 $APPHUB_PORT 可能未开放"
    fi
    
    # 检查健康状态
    if docker inspect ai-infra-apphub --format='{{.State.Health.Status}}' 2>/dev/null | grep -q "healthy"; then
        log_info "✓ AppHub 健康检查通过"
    else
        HEALTH_STATUS=$(docker inspect ai-infra-apphub --format='{{.State.Health.Status}}' 2>/dev/null || echo "unknown")
        log_warn "⚠️  AppHub 健康检查状态: $HEALTH_STATUS"
    fi
else
    log_warn "⚠️  AppHub 容器未运行，跳过连接检查"
fi
echo ""

# 6. 检查构建环境
echo "6️⃣  检查构建环境..."
if docker buildx ls >/dev/null 2>&1; then
    log_info "✓ Docker Buildx 可用"
    
    # 检查构建器
    BUILDERS=$(docker buildx ls 2>/dev/null | grep -v "^NAME" | awk '{print $1}' || echo "")
    if [[ -n "$BUILDERS" ]]; then
        log_debug "  可用的构建器:"
        echo "$BUILDERS" | sed 's/^/    - /'
    fi
else
    log_warn "⚠️  Docker Buildx 不可用"
fi
echo ""

# 7. 建议
echo "7️⃣  诊断建议"
echo ""

if [[ -z "$APPHUB_RUNNING" ]]; then
    echo "  问题：AppHub 容器未运行"
    echo "  解决方案："
    echo "    1. 启动 AppHub:"
    echo "       docker-compose up -d apphub"
    echo ""
    echo "    2. 等待容器就绪（检查健康检查）:"
    echo "       docker-compose ps apphub"
    echo ""
    echo "    3. 然后重新开始构建:"
    echo "       ./build.sh build gitea --force"
elif [[ -z "$APPHUB_IP" ]]; then
    echo "  问题：无法获取 AppHub 容器的 IP 地址"
    echo "  解决方案："
    echo "    1. 检查容器是否正确连接到 ai-infra-network:"
    echo "       docker inspect ai-infra-apphub | jq '.NetworkSettings.Networks'"
    echo ""
    echo "    2. 重启 AppHub 容器以重新连接到网络:"
    echo "       docker-compose down apphub"
    echo "       docker-compose up -d apphub"
    echo ""
    echo "    3. 检查网络是否损坏："
    echo "       docker network inspect ai-infra-network"
else
    echo "  AppHub 配置看起来正常！"
    echo "  如果仍然遇到问题，请尝试："
    echo "    1. 清理并重启所有服务:"
    echo "       docker-compose down"
    echo "       docker-compose up -d"
    echo ""
    echo "    2. 检查构建器驱动程序:"
    echo "       docker buildx ls"
    echo ""
    echo "    3. 强制重新构建:"
    echo "       ./build.sh build gitea --force"
fi
echo ""

echo "========================================="
echo "  诊断完成"
echo "========================================="
