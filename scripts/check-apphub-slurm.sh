#!/bin/bash
# 检查 AppHub SLURM 包状态
# 快速诊断工具

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

print_header() { echo -e "${CYAN}========================================${NC}"; echo -e "${CYAN}$1${NC}"; echo -e "${CYAN}========================================${NC}"; }
print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 检查 1: SLURM 源码包
print_header "检查 1: SLURM 源码包"
if [ -f "src/apphub/slurm-25.05.4.tar.bz2" ]; then
    print_success "SLURM 源码包存在"
    ls -lh src/apphub/slurm-25.05.4.tar.bz2
else
    print_error "SLURM 源码包不存在: src/apphub/slurm-25.05.4.tar.bz2"
    NEEDS_REBUILD=true
fi
echo ""

# 检查 2: AppHub 容器状态
print_header "检查 2: AppHub 容器状态"
if docker ps --format '{{.Names}}' | grep -q "^ai-infra-apphub$"; then
    print_success "AppHub 容器正在运行"
    docker ps --filter "name=ai-infra-apphub" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
else
    print_warning "AppHub 容器未运行"
    print_info "启动 AppHub: docker-compose up -d apphub"
    NEEDS_REBUILD=true
fi
echo ""

# 检查 3: AppHub 中的 SLURM 包
print_header "检查 3: AppHub SLURM APK 包"
if docker exec ai-infra-apphub test -d /usr/share/nginx/html/pkgs/slurm-apk 2>/dev/null; then
    print_info "检查 SLURM APK 目录..."
    APK_COUNT=$(docker exec ai-infra-apphub sh -c "ls /usr/share/nginx/html/pkgs/slurm-apk/*.tar.gz 2>/dev/null | wc -l" || echo 0)
    if [ "$APK_COUNT" -gt 0 ]; then
        print_success "找到 ${APK_COUNT} 个 SLURM APK 包"
        docker exec ai-infra-apphub ls -lh /usr/share/nginx/html/pkgs/slurm-apk/*.tar.gz
        echo ""
        print_info "包内容预览:"
        PKG_NAME=$(docker exec ai-infra-apphub sh -c "ls /usr/share/nginx/html/pkgs/slurm-apk/*.tar.gz 2>/dev/null | head -1" || echo "")
        if [ -n "$PKG_NAME" ]; then
            docker exec ai-infra-apphub tar tzf "$PKG_NAME" | head -20
        fi
    else
        print_error "AppHub 中没有 SLURM APK 包"
        print_info "目录内容:"
        docker exec ai-infra-apphub ls -la /usr/share/nginx/html/pkgs/slurm-apk/ || true
        NEEDS_REBUILD=true
    fi
else
    print_error "无法访问 AppHub 容器"
    NEEDS_REBUILD=true
fi
echo ""

# 检查 4: AppHub HTTP 服务
print_header "检查 4: AppHub HTTP 服务"
if curl -s -o /dev/null -w "%{http_code}" http://localhost:8081/pkgs/slurm-apk/ 2>/dev/null | grep -q "200"; then
    print_success "AppHub HTTP 服务正常 (http://localhost:8081)"
    print_info "测试下载 SLURM 包..."
    if curl -I http://localhost:8081/pkgs/slurm-apk/slurm-client-latest-alpine.tar.gz 2>/dev/null | grep -q "200 OK"; then
        print_success "SLURM 包可下载"
    else
        print_warning "SLURM latest 符号链接可能未创建"
    fi
else
    print_warning "AppHub HTTP 服务访问失败"
    print_info "检查端口映射: docker-compose ps apphub"
fi
echo ""

# 检查 5: Backend 容器状态
print_header "检查 5: Backend 容器状态"
if docker ps --format '{{.Names}}' | grep -q "^ai-infra-backend$"; then
    print_success "Backend 容器正在运行"
    docker ps --filter "name=ai-infra-backend" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"
    echo ""
    print_info "检查 SLURM 客户端安装..."
    if docker exec ai-infra-backend which sinfo >/dev/null 2>&1; then
        print_success "SLURM 客户端已安装"
        docker exec ai-infra-backend sinfo --version 2>&1 | head -3 || true
    else
        print_error "SLURM 客户端未安装"
        print_info "Backend 需要重建"
        NEEDS_REBUILD=true
    fi
else
    print_warning "Backend 容器未运行"
fi
echo ""

# 检查 6: Docker 网络连通性
print_header "检查 6: Docker 网络连通性"
if docker ps --format '{{.Names}}' | grep -q "^ai-infra-backend$"; then
    print_info "测试 Backend → AppHub 网络连通性..."
    if docker exec ai-infra-backend wget -q --spider --timeout=5 http://apphub/pkgs/slurm-apk/ 2>/dev/null; then
        print_success "Backend 可以访问 AppHub (http://apphub)"
    else
        print_warning "Backend 无法访问 AppHub"
        print_info "检查 Docker 网络配置"
    fi
else
    print_info "Backend 未运行，跳过网络测试"
fi
echo ""

# 检查 7: 构建配置
print_header "检查 7: Dockerfile 配置"
print_info "检查 AppHub Dockerfile SLURM 配置..."
if grep -q "SLURM_VERSION=25.05.4" src/apphub/Dockerfile; then
    print_success "AppHub Dockerfile 配置正确"
else
    print_warning "AppHub Dockerfile SLURM 版本配置可能不匹配"
fi

print_info "检查 Backend Dockerfile SLURM 安装逻辑..."
if grep -q "slurm-client-latest-alpine.tar.gz" src/backend/Dockerfile; then
    print_success "Backend Dockerfile 配置正确"
else
    print_warning "Backend Dockerfile SLURM 安装配置可能不匹配"
fi
echo ""

# 总结
print_header "诊断总结"
if [ "${NEEDS_REBUILD:-false}" = "true" ]; then
    print_warning "需要重新构建"
    echo ""
    print_info "推荐操作:"
    echo "  1. 确保 SLURM 源码包存在:"
    echo "     ls -lh src/apphub/slurm-25.05.4.tar.bz2"
    echo ""
    echo "  2. 重新构建 AppHub 和 Backend:"
    echo "     chmod +x scripts/rebuild-apphub-and-backend.sh"
    echo "     ./scripts/rebuild-apphub-and-backend.sh"
    echo ""
    echo "  3. 或者手动构建:"
    echo "     docker-compose build --no-cache apphub"
    echo "     docker-compose up -d apphub"
    echo "     docker-compose build --no-cache backend"
    echo "     docker-compose up -d backend"
else
    print_success "所有检查通过，系统状态正常"
    echo ""
    print_info "可以开始测试 SLURM 功能:"
    echo "  docker exec ai-infra-backend sinfo --version"
    echo "  访问 http://localhost:8080/slurm-tasks"
fi
echo ""
