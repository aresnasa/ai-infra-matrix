#!/bin/bash
# 构建 SLURM Alpine APK 包并上传到 AppHub
# 用途: 为 Backend 容器提供 SLURM 客户端 APK 安装包

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 配置
SLURM_VERSION="${SLURM_VERSION:-24.05.5}"
APK_OUTPUT_DIR="./apks/alpine"
APPHUB_CONTAINER="ai-infra-apphub"
APPHUB_APK_PATH="/usr/share/nginx/html/apks/alpine"

print_info "开始构建 SLURM Alpine APK v${SLURM_VERSION}..."

# 创建输出目录
mkdir -p "$APK_OUTPUT_DIR"

# 检查 AppHub 容器是否运行
if ! docker ps --format '{{.Names}}' | grep -q "^${APPHUB_CONTAINER}$"; then
    print_warning "AppHub 容器未运行，尝试启动..."
    docker-compose up -d apphub || {
        print_error "无法启动 AppHub 容器"
        exit 1
    }
    sleep 5
fi

print_success "AppHub 容器运行正常"

# 使用 Alpine 容器构建 APK 包
print_info "创建 Alpine 构建容器..."
docker run --rm -v "$(pwd)/$APK_OUTPUT_DIR:/output" alpine:latest /bin/sh -c "
set -e

# 配置 Alpine 镜像源
for MIR in mirrors.tuna.tsinghua.edu.cn mirrors.aliyun.com mirrors.ustc.edu.cn dl-cdn.alpinelinux.org; do
    sed -i \"s#://[^/]\\+/alpine#://\$MIR/alpine#g\" /etc/apk/repositories || true
    apk update && break || true
done

echo '>>> 安装构建工具和依赖...'
apk add --no-cache \\
    alpine-sdk \\
    sudo \\
    abuild \\
    wget \\
    tar \\
    gzip

echo '>>> 设置构建用户...'
adduser -D builder
addgroup builder abuild
echo 'builder ALL=(ALL) NOPASSWD: ALL' >> /etc/sudoers

# 切换到 builder 用户
su - builder -c \"
set -e
cd ~

# 生成 APK 签名密钥
abuild-keygen -a -i -n

# 创建 SLURM APK 构建目录
mkdir -p ~/slurm-client
cd ~/slurm-client

# 创建 APKBUILD 文件
cat > APKBUILD <<'EOF'
# Maintainer: AI Infra Matrix Team
pkgname=slurm-client
pkgver=${SLURM_VERSION}
pkgrel=0
pkgdesc='SLURM client tools (sinfo, squeue, sbatch, etc.)'
url='https://slurm.schedmd.com/'
arch='all'
license='GPL-2.0-or-later'
depends='munge json-c libyaml hwloc'
makedepends='linux-headers openssl-dev munge-dev json-c-dev libyaml-dev hwloc-dev'
source=\"https://download.schedmd.com/slurm/slurm-\${pkgver}.tar.bz2\"

build() {
    cd \"\$srcdir/slurm-\$pkgver\"
    ./configure \\
        --prefix=/usr \\
        --sysconfdir=/etc/slurm \\
        --localstatedir=/var \\
        --without-pam \\
        --without-rpath \\
        --enable-slurmrestd=no \\
        --with-munge=/usr
    make -j\$(nproc)
}

package() {
    cd \"\$srcdir/slurm-\$pkgver\"
    
    # 只安装客户端工具
    mkdir -p \"\$pkgdir/usr/bin\"
    install -m 755 src/sinfo/sinfo \"\$pkgdir/usr/bin/\"
    install -m 755 src/squeue/squeue \"\$pkgdir/usr/bin/\"
    install -m 755 src/sbatch/sbatch \"\$pkgdir/usr/bin/\"
    install -m 755 src/salloc/salloc \"\$pkgdir/usr/bin/\"
    install -m 755 src/srun/srun \"\$pkgdir/usr/bin/\"
    install -m 755 src/scancel/scancel \"\$pkgdir/usr/bin/\"
    install -m 755 src/scontrol/scontrol \"\$pkgdir/usr/bin/\"
    install -m 755 src/sacct/sacct \"\$pkgdir/usr/bin/\"
    install -m 755 src/sacctmgr/sacctmgr \"\$pkgdir/usr/bin/\"
    
    # 安装共享库
    mkdir -p \"\$pkgdir/usr/lib\"
    find src -name '*.so*' -exec install -m 755 {} \"\$pkgdir/usr/lib/\" \\;
    
    # 创建配置目录
    mkdir -p \"\$pkgdir/etc/slurm\"
}
EOF

# 构建 APK 包
echo '>>> 构建 APK 包...'
abuild checksum
abuild -r

# 复制生成的 APK 到输出目录
echo '>>> 复制 APK 包到输出目录...'
find ~/packages -name 'slurm-client-*.apk' -exec cp {} /output/ \\;

# 生成 APK 索引
cd /output
apk index -o APKINDEX.tar.gz *.apk || echo 'Warning: apk index generation may have issues'
abuild-sign -k ~/.abuild/*.rsa APKINDEX.tar.gz || echo 'Warning: signing may have issues'

echo '✓ APK 构建完成'
\"
" || {
    print_error "APK 构建失败"
    exit 1
}

print_success "SLURM APK 包构建成功"

# 列出生成的文件
print_info "生成的 APK 文件:"
ls -lh "$APK_OUTPUT_DIR"/*.apk 2>/dev/null || print_warning "未找到 APK 文件"

# 上传到 AppHub
print_info "上传到 AppHub 容器..."
docker exec "$APPHUB_CONTAINER" mkdir -p "$APPHUB_APK_PATH" || true
docker cp "$APK_OUTPUT_DIR/." "$APPHUB_CONTAINER:$APPHUB_APK_PATH/" || {
    print_error "上传到 AppHub 失败"
    exit 1
}

print_success "已上传到 AppHub: $APPHUB_APK_PATH"

# 验证
print_info "验证 AppHub 中的 APK 仓库..."
docker exec "$APPHUB_CONTAINER" ls -lh "$APPHUB_APK_PATH/" || true

print_success "SLURM APK 包已准备就绪！"
echo ""
print_info "下一步："
echo "  1. 重新构建 backend 镜像: docker-compose build backend"
echo "  2. 重启 backend 容器: docker-compose up -d backend"
echo "  3. 验证安装: docker exec ai-infra-backend sh -c 'sinfo --version'"
