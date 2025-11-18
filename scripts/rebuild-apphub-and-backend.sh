#!/bin/bash
# 重建 AppHub 和 Backend 确保 SLURM 从 AppHub 安装
# 用途: 修复 SLURM 客户端安装问题，确保只从 AppHub 安装

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

# 检查 SLURM 源码包
print_header "步骤 1: 检查 SLURM 源码包"
if [ -f "src/apphub/slurm-25.05.4.tar.bz2" ]; then
    print_success "SLURM 源码包存在"
    ls -lh src/apphub/slurm-25.05.4.tar.bz2
else
    print_error "SLURM 源码包不存在: src/apphub/slurm-25.05.4.tar.bz2"
    print_info "请从 https://download.schedmd.com/slurm/ 下载"
    exit 1
fi

# 停止现有容器
print_header "步骤 2: 停止现有容器"
print_info "停止 backend 和 backend-init..."
docker-compose stop backend backend-init || true
print_info "停止 apphub..."
docker-compose stop apphub || true
sleep 2

# 删除旧容器
print_header "步骤 3: 删除旧容器"
print_info "删除 backend 和 backend-init 容器..."
docker-compose rm -f backend backend-init || true
print_info "删除 apphub 容器..."
docker-compose rm -f apphub || true

# 重建 AppHub
print_header "步骤 4: 重建 AppHub（包含 SLURM 包）"
print_info "使用 --no-cache 强制重新构建..."
print_warning "这可能需要 10-30 分钟，具体取决于网络和机器性能"
echo ""

if docker-compose build --no-cache apphub; then
    print_success "AppHub 构建成功"
else
    print_error "AppHub 构建失败"
    print_info "请检查构建日志："
    echo "  docker-compose logs --tail=100 apphub"
    exit 1
fi

# 启动 AppHub
print_header "步骤 5: 启动 AppHub"
if docker-compose up -d apphub; then
    print_success "AppHub 已启动"
else
    print_error "AppHub 启动失败"
    exit 1
fi

# 等待 AppHub 就绪
print_info "等待 AppHub 服务就绪..."
sleep 5

# 验证 SLURM APK 包
print_header "步骤 6: 验证 SLURM APK 包"
print_info "检查 AppHub 中的 SLURM 包..."
if docker exec ai-infra-apphub ls -lh /usr/share/nginx/html/pkgs/slurm-apk/ 2>/dev/null; then
    echo ""
    APK_COUNT=$(docker exec ai-infra-apphub sh -c "ls /usr/share/nginx/html/pkgs/slurm-apk/*.tar.gz 2>/dev/null | wc -l" || echo 0)
    if [ "$APK_COUNT" -gt 0 ]; then
        print_success "找到 ${APK_COUNT} 个 SLURM APK 包"
        docker exec ai-infra-apphub ls -lh /usr/share/nginx/html/pkgs/slurm-apk/*.tar.gz
    else
        print_error "AppHub 中没有 SLURM APK 包"
        print_info "检查构建日志:"
        docker-compose logs apphub | grep -A 10 "SLURM" | tail -20
        exit 1
    fi
else
    print_error "无法访问 AppHub 容器"
    exit 1
fi

# 测试 HTTP 访问
print_info "测试 HTTP 访问..."
if curl -s -o /dev/null -w "%{http_code}" http://localhost:8081/pkgs/slurm-apk/ | grep -q "200"; then
    print_success "AppHub HTTP 服务正常"
else
    print_warning "AppHub HTTP 服务可能未就绪"
fi

# 重建 Backend
print_header "步骤 7: 重建 Backend（从 AppHub 安装 SLURM）"
print_info "使用 --no-cache 强制重新构建..."
print_warning "这可能需要 5-15 分钟"
echo ""

if docker-compose build --no-cache backend; then
    print_success "Backend 构建成功"
else
    print_error "Backend 构建失败"
    print_info "可能原因："
    echo "  1. AppHub 中没有 SLURM 包"
    echo "  2. 网络连接问题"
    echo "  3. 包格式错误"
    print_info "检查构建日志："
    echo "  docker-compose logs --tail=100 backend"
    exit 1
fi

# 启动 Backend
print_header "步骤 8: 启动 Backend"
if docker-compose up -d backend backend-init; then
    print_success "Backend 已启动"
else
    print_error "Backend 启动失败"
    exit 1
fi

# 等待 Backend 就绪
print_info "等待 Backend 服务就绪..."
sleep 10

# 验证 SLURM 安装
print_header "步骤 9: 验证 SLURM 客户端安装"
print_info "检查 sinfo 命令..."
if docker exec ai-infra-backend which sinfo >/dev/null 2>&1; then
    print_success "sinfo 命令已安装"
    echo ""
    print_info "SLURM 版本信息:"
    docker exec ai-infra-backend sinfo --version 2>&1 | head -3 || true
    echo ""
    print_info "可用的 SLURM 命令:"
    for cmd in sinfo squeue scontrol scancel sbatch srun salloc sacct sacctmgr; do
        if docker exec ai-infra-backend which $cmd >/dev/null 2>&1; then
            echo "  ✓ $cmd"
        else
            echo "  ✗ $cmd (未找到)"
        fi
    done
else
    print_error "sinfo 命令未找到"
    print_info "检查 Backend 构建日志:"
    docker-compose logs backend | grep -i "slurm" | tail -30
    exit 1
fi

# 最终验证
print_header "步骤 10: 最终验证"
print_info "测试 SLURM 命令执行..."
if docker exec ai-infra-backend sinfo --help >/dev/null 2>&1; then
    print_success "SLURM 客户端工作正常"
else
    print_warning "SLURM 命令执行可能有问题"
fi

# 完成
print_header "✅ 重建完成"
echo ""
print_success "AppHub 和 Backend 已成功重建"
print_success "SLURM 客户端已从 AppHub 安装"
echo ""
print_info "下一步:"
echo "  1. 测试 SLURM 功能:"
echo "     docker exec ai-infra-backend sinfo --version"
echo ""
echo "  2. 访问 Web 界面:"
echo "     http://localhost:8080/slurm-tasks"
echo ""
echo "  3. 测试任务创建:"
echo "     创建一个扩容任务并验证功能"
echo ""
print_info "查看日志:"
echo "  docker-compose logs -f backend"
echo ""
