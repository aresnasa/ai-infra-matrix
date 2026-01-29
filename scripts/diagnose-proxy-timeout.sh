#!/bin/bash

# ARM64 构建代理和超时问题 - 深度诊断和修复脚本
# 解决 OAuth 超时和网络隔离问题

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}  ARM64 构建代理和超时问题诊断${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo

# ============================================================================
# 第一步：检查代理配置
# ============================================================================
echo -e "${YELLOW}【第一步】检查代理和镜像配置${NC}"
echo

DAEMON_JSON="$HOME/.docker/daemon.json"

if [[ ! -f "$DAEMON_JSON" ]]; then
    echo -e "${YELLOW}⚠  daemon.json 不存在${NC}"
else
    echo "✓ daemon.json 配置:"
    
    # 检查代理配置
    if grep -q "httpProxy\|httpsProxy" "$DAEMON_JSON"; then
        echo -e "${CYAN}  代理配置检测到:${NC}"
        grep -o '"http[s]*Proxy": "[^"]*"' "$DAEMON_JSON" | while read line; do
            echo "    $line"
        done
    else
        echo "  代理配置: 未设置"
    fi
    
    # 检查镜像加速
    if grep -q "registry-mirrors" "$DAEMON_JSON"; then
        echo -e "${CYAN}  镜像加速检测到:${NC}"
        grep -A10 '"registry-mirrors"' "$DAEMON_JSON" | grep -oE 'https?://[^"]+' | while read mirror; do
            echo "    $mirror"
        done
    else
        echo "  镜像加速: 未设置"
    fi
    
    # 检查不安全仓库
    if grep -q "insecure-registries" "$DAEMON_JSON"; then
        echo -e "${CYAN}  不安全仓库:${NC}"
        grep -A5 '"insecure-registries"' "$DAEMON_JSON" | grep -oE '"[^"]+"' | grep -v insecure-registries | while read reg; do
            echo "    $reg"
        done
    fi
fi
echo

# ============================================================================
# 第二步：检查 multiarch-builder 网络配置
# ============================================================================
echo -e "${YELLOW}【第二步】检查 multiarch-builder 网络隔离问题${NC}"
echo

BUILDER_NAME="multiarch-builder"

if ! docker buildx ls | grep -q "^$BUILDER_NAME"; then
    echo -e "${RED}✗ Builder '$BUILDER_NAME' 不存在${NC}"
else
    echo "✓ Builder 存在"
    
    # 检查网络模式
    net_mode=$(docker buildx inspect "$BUILDER_NAME" 2>&1 | grep "Driver Options" | grep -o 'network="[^"]*"' || echo 'network="bridge"')
    echo "  网络模式: $net_mode"
    
    if [[ "$net_mode" == *"host"* ]]; then
        echo -e "${GREEN}  ✓ 已配置 host 网络${NC}"
    else
        echo -e "${YELLOW}  ⚠  网络隔离可能导致超时问题${NC}"
        echo "  建议重新创建 builder: docker buildx rm $BUILDER_NAME"
    fi
    
    # 检查 BuildKit flags
    if docker buildx inspect "$BUILDER_NAME" 2>&1 | grep -q "network.host"; then
        echo -e "${GREEN}  ✓ network.host entitlement 已启用${NC}"
    else
        echo -e "${YELLOW}  ⚠  network.host entitlement 未启用${NC}"
    fi
fi
echo

# ============================================================================
# 第三步：诊断 localhost 访问（代理最常见的问题）
# ============================================================================
echo -e "${YELLOW}【第三步】诊断 localhost 和代理访问${NC}"
echo

# 检查本地代理服务
if netstat -tuln 2>/dev/null | grep -q ":7890"; then
    echo -e "${GREEN}✓ 本地代理监听在 127.0.0.1:7890${NC}"
    
    # 尝试通过代理访问
    echo "  测试代理连接..."
    if timeout 5 curl -x http://127.0.0.1:7890 -I https://docker.io >/dev/null 2>&1; then
        echo -e "${GREEN}  ✓ 通过代理可以访问 docker.io${NC}"
    else
        echo -e "${YELLOW}  ⚠  通过代理访问失败，可能原因：${NC}"
        echo "     1. 代理需要认证"
        echo "     2. 代理配置错误"
        echo "     3. 代理服务未运行或超时"
    fi
else
    echo -e "${YELLOW}⚠  未检测到本地代理监听在 127.0.0.1:7890${NC}"
    echo "  检查其他可能的代理端口..."
    netstat -tuln 2>/dev/null | grep LISTEN | grep "127.0.0.1" || echo "  未找到本地监听服务"
fi
echo

# ============================================================================
# 第四步：检查 buildkit 容器能否访问代理
# ============================================================================
echo -e "${YELLOW}【第四步】检查 buildkit 容器网络隔离问题${NC}"
echo

# 获取 buildkit 容器 ID
BUILDKIT_CONTAINER=$(docker ps -q -f "name=${BUILDER_NAME}" 2>/dev/null | head -1)

if [[ -z "$BUILDKIT_CONTAINER" ]]; then
    echo -e "${YELLOW}⚠  multiarch-builder 容器未运行${NC}"
    echo "  将在下次构建时启动"
else
    echo "✓ buildkit 容器运行中: $BUILDKIT_CONTAINER"
    
    # 检查容器网络模式
    container_net=$(docker inspect "$BUILDKIT_CONTAINER" -f '{{.HostConfig.NetworkMode}}' 2>/dev/null)
    echo "  容器网络模式: $container_net"
    
    if [[ "$container_net" == "host" ]]; then
        echo -e "${GREEN}  ✓ 容器使用 host 网络，可以直接访问本地代理${NC}"
    else
        echo -e "${YELLOW}  ⚠  容器网络隔离，无法访问 localhost:7890${NC}"
    fi
fi
echo

# ============================================================================
# 第五步：建议修复方案
# ============================================================================
echo -e "${YELLOW}【第五步】修复方案${NC}"
echo

cat << 'EOF'
根据诊断结果，如果遇到 OAuth 超时问题，尝试以下方案：

【方案 1】重新创建 multiarch-builder（推荐）
  docker buildx rm multiarch-builder
  # 脚本会自动重新创建配置正确的 builder
  ./build.sh build-platform arm64 --force

  这会应用以下改进：
  - 启用 host 网络（直接访问代理）
  - 自动检测并应用 daemon.json 中的代理和镜像配置
  - 添加更好的重试机制

【方案 2】在 daemon.json 中明确设置代理（如果尚未设置）
  {
    "httpProxy": "http://127.0.0.1:7890",
    "httpsProxy": "http://127.0.0.1:7890",
    "noProxy": "localhost,127.0.0.1"
  }
  
  然后重启 Docker：
  killall Docker || true
  # 重启 Docker Desktop

【方案 3】如果使用了 daemon.json 中的镜像加速
  脚本已更新，会自动在 BuildKit 配置中使用这些镜像
  确保镜像地址可访问（通常需要在代理白名单中）

【方案 4】预先拉取容易超时的镜像
  脚本的 prefetch_base_images_for_platform() 函数会：
  - 自动发现所有需要的基础镜像
  - 在构建前预先拉取（避免构建时超时）
  - 智能重试处理超时和网络错误

【方案 5】如果仍然超时，检查以下几点
  1. 代理服务是否正常运行：
     telnet 127.0.0.1 7890
     
  2. 代理是否支持 HTTPS CONNECT：
     curl -x http://127.0.0.1:7890 -I https://auth.docker.io
     
  3. 检查代理日志是否有错误
  
  4. 尝试增加超时时间：
     curl --connect-timeout 10 -x http://127.0.0.1:7890 -I https://docker.io

【验证修复】
  1. 重新创建 builder：
     docker buildx rm multiarch-builder
  
  2. 运行诊断脚本：
     ./test-arm64-network.sh
  
  3. 尝试构建：
     ./build.sh build-platform arm64 --force
  
  4. 观察日志：
     tail -100 .build-failures.log | grep -i "error\|timeout\|oauth"
EOF

echo
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${YELLOW}快速命令参考：${NC}"
echo
echo "# 检查代理配置"
echo "  cat ~/.docker/daemon.json | grep -A2 'httpProxy\|registry-mirrors'"
echo
echo "# 重新创建 builder（应用所有修复）"
echo "  docker buildx rm multiarch-builder"
echo "  ./build.sh build-platform arm64 --force"
echo
echo "# 测试代理连接"
echo "  curl -x http://127.0.0.1:7890 -I https://docker.io"
echo
echo "# 查看 BuildKit 日志"
echo "  docker buildx du"
echo
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
