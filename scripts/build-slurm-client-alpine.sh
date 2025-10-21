#!/bin/bash
#===============================================================================
# SLURM Alpine 客户端构建脚本
# 功能：编译 SLURM 客户端工具并打包为 tar.gz，上传到 AppHub
# 用途：为 Alpine Linux 容器提供预编译的 SLURM 客户端
#===============================================================================

set -e

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 配置
SLURM_VERSION="${SLURM_VERSION:-23.11.10}"
BUILD_DIR="/tmp/slurm-alpine-build"
OUTPUT_DIR="./pkgs/slurm-apk"
PACKAGE_NAME="slurm-client-${SLURM_VERSION}-alpine.tar.gz"
APPHUB_CONTAINER="ai-infra-apphub"
APPHUB_PATH="/usr/share/nginx/html/pkgs/slurm-apk"

print_info "开始构建 SLURM Alpine 客户端 v${SLURM_VERSION}..."

# 创建输出目录
mkdir -p "$OUTPUT_DIR"

# 创建构建容器
print_info "创建 Alpine 构建容器..."
docker run --rm -v "$(pwd)/$OUTPUT_DIR:/output" alpine:latest /bin/sh -c "
set -e

# 配置 Alpine 镜像源（多镜像回退）
for MIR in mirrors.tuna.tsinghua.edu.cn mirrors.aliyun.com mirrors.ustc.edu.cn dl-cdn.alpinelinux.org; do
    sed -i \"s#://[^/]\\+/alpine#://\$MIR/alpine#g\" /etc/apk/repositories || true
    apk update && break || true
done

# 安装构建依赖（某些包可能不可用，使用 || true 忽略）
echo '>>> 安装构建依赖...'
apk add --no-cache \
    build-base \
    linux-headers \
    openssl-dev \
    readline-dev \
    curl \
    wget \
    perl \
    python3 \
    mariadb-dev \
    ncurses-dev \
    json-c-dev \
    yaml-dev \
    libevent-dev \
    lz4-dev \
    zlib-dev \
    bzip2-dev

# 尝试安装可选依赖（可能不可用）
apk add --no-cache munge-dev || echo '  ⚠ munge-dev not available (optional)'
apk add --no-cache pam-dev || echo '  ⚠ pam-dev not available (optional)'
apk add --no-cache http-parser-dev || echo '  ⚠ http-parser-dev not available (optional)'
apk add --no-cache numactl-dev || echo '  ⚠ numactl-dev not available (optional)'
apk add --no-cache hwloc-dev || echo '  ⚠ hwloc-dev not available (optional)'

# 下载 SLURM 源码
cd /tmp
echo '>>> 下载 SLURM 源码...'
SLURM_URL=\"https://download.schedmd.com/slurm/slurm-${SLURM_VERSION}.tar.bz2\"
wget -q \"\$SLURM_URL\" || {
    echo 'SLURM 官方源下载失败，尝试 GitHub 镜像...'
    SLURM_URL=\"https://github.com/SchedMD/slurm/archive/refs/tags/slurm-${SLURM_VERSION//./-}.tar.gz\"
    wget -q \"\$SLURM_URL\" -O slurm-${SLURM_VERSION}.tar.bz2
}

tar xjf slurm-${SLURM_VERSION}.tar.bz2
cd slurm-${SLURM_VERSION}

# 配置编译选项（仅客户端工具，禁用不可用的特性）
echo '>>> 配置编译选项...'
./configure \\
    --prefix=/usr/local/slurm \\
    --sysconfdir=/etc/slurm \\
    --without-munge \\
    --without-pam \\
    --without-rpath \\
    --disable-debug \\
    --without-gtk2 \\
    --without-hdf5 \\
    --without-numa \\
    --without-hwloc || {
        echo '配置失败，查看 config.log...'
        tail -100 config.log
        exit 1
    }

# 仅编译客户端工具
echo '>>> 编译 SLURM 客户端工具...'
make -j\$(nproc) || make

# 创建安装目录
mkdir -p /tmp/slurm-install/usr/local/slurm/bin
mkdir -p /tmp/slurm-install/usr/local/slurm/lib
mkdir -p /tmp/slurm-install/etc/slurm

# 安装客户端工具
echo '>>> 安装客户端工具...'
cd src
make install DESTDIR=/tmp/slurm-install || true

# 手动复制客户端工具（确保存在）
cd /tmp/slurm-${SLURM_VERSION}
for cmd in sinfo squeue scontrol scancel sbatch srun salloc sacct; do
    if [ -f \"src/\${cmd}/\${cmd}\" ]; then
        cp -f \"src/\${cmd}/\${cmd}\" /tmp/slurm-install/usr/local/slurm/bin/
        echo \"  ✓ Installed: \${cmd}\"
    fi
done

# 复制必要的库
echo '>>> 复制依赖库...'
cp -f src/common/.libs/libslurm.so* /tmp/slurm-install/usr/local/slurm/lib/ 2>/dev/null || true

# 创建版本信息
echo '${SLURM_VERSION}' > /tmp/slurm-install/usr/local/slurm/VERSION

# 创建安装脚本
cat > /tmp/slurm-install/install.sh << 'INSTALL_EOF'
#!/bin/sh
set -e

echo \"Installing SLURM client tools...\"

# 复制文件
cp -r usr/local/slurm /usr/local/
cp -r etc/slurm /etc/ 2>/dev/null || mkdir -p /etc/slurm

# 设置权限
chmod +x /usr/local/slurm/bin/*

# 创建符号链接到 /usr/bin
for cmd in /usr/local/slurm/bin/*; do
    ln -sf \"\$cmd\" /usr/bin/\$(basename \"\$cmd\")
done

# 配置库路径
if [ ! -f /etc/ld.so.conf.d/slurm.conf ]; then
    mkdir -p /etc/ld.so.conf.d
    echo \"/usr/local/slurm/lib\" > /etc/ld.so.conf.d/slurm.conf
    ldconfig 2>/dev/null || true
fi

# 设置环境变量
if ! grep -q 'SLURM_HOME' /etc/profile 2>/dev/null; then
    cat >> /etc/profile << 'PROFILE_EOF'

# SLURM Client Environment
export SLURM_HOME=/usr/local/slurm
export PATH=\$SLURM_HOME/bin:\$PATH
export LD_LIBRARY_PATH=\$SLURM_HOME/lib:\$LD_LIBRARY_PATH
PROFILE_EOF
fi

echo \"SLURM client tools installed successfully!\"
echo \"Version: \$(cat /usr/local/slurm/VERSION 2>/dev/null || echo 'unknown')\"
echo \"\"
echo \"Available commands:\"
ls -1 /usr/local/slurm/bin/
INSTALL_EOF

chmod +x /tmp/slurm-install/install.sh

# 创建卸载脚本
cat > /tmp/slurm-install/uninstall.sh << 'UNINSTALL_EOF'
#!/bin/sh
echo \"Uninstalling SLURM client tools...\"
rm -rf /usr/local/slurm
rm -f /usr/bin/sinfo /usr/bin/squeue /usr/bin/scontrol /usr/bin/scancel /usr/bin/sbatch /usr/bin/srun /usr/bin/salloc /usr/bin/sacct
rm -f /etc/ld.so.conf.d/slurm.conf
rm -rf /etc/slurm
sed -i '/SLURM_HOME/,+2d' /etc/profile 2>/dev/null || true
echo \"SLURM client tools uninstalled.\"
UNINSTALL_EOF

chmod +x /tmp/slurm-install/uninstall.sh

# 创建 README
cat > /tmp/slurm-install/README.md << 'README_EOF'
# SLURM Alpine Client Tools

## Version
\$(cat /tmp/slurm-install/usr/local/slurm/VERSION)

## Installation

\`\`\`bash
# Extract package
tar xzf slurm-client-*-alpine.tar.gz
cd slurm-client-*/

# Run installation script
./install.sh
\`\`\`

## Verification

\`\`\`bash
sinfo --version
which sinfo squeue scontrol
\`\`\`

## Client Tools Included

- sinfo - View cluster/node information
- squeue - View job queue
- scontrol - Administrative tool
- scancel - Cancel jobs
- sbatch - Submit batch job
- srun - Run parallel job
- salloc - Allocate resources
- sacct - Job accounting

## Uninstallation

\`\`\`bash
cd slurm-client-*/
./uninstall.sh
\`\`\`

## Environment Variables

After installation, these are set in /etc/profile:
- SLURM_HOME=/usr/local/slurm
- PATH includes \$SLURM_HOME/bin
- LD_LIBRARY_PATH includes \$SLURM_HOME/lib

## Requirements

Alpine Linux with:
- openssl
- readline
- ncurses
- json-c

Install runtime dependencies:
\`\`\`bash
apk add --no-cache openssl readline ncurses json-c yaml libevent
\`\`\`
README_EOF

# 打包
echo '>>> 打包客户端工具...'
cd /tmp/slurm-install
tar czf /output/${PACKAGE_NAME} .

# 显示包内容
echo '>>> 包内容:'
tar tzf /output/${PACKAGE_NAME} | head -20
echo '...'

# 显示包大小
ls -lh /output/${PACKAGE_NAME}

echo '>>> 构建完成!'
"

if [ ! -f "$OUTPUT_DIR/$PACKAGE_NAME" ]; then
    print_error "构建失败：未找到输出文件 $OUTPUT_DIR/$PACKAGE_NAME"
    exit 1
fi

print_success "SLURM Alpine 客户端包构建完成: $OUTPUT_DIR/$PACKAGE_NAME"
print_info "包大小: $(du -h "$OUTPUT_DIR/$PACKAGE_NAME" | cut -f1)"

# 上传到 AppHub
print_info "上传到 AppHub..."

# 检查 AppHub 容器是否运行
if ! docker ps --format '{{.Names}}' | grep -q "^${APPHUB_CONTAINER}$"; then
    print_warning "AppHub 容器未运行，跳过上传"
    print_info "手动上传命令："
    echo "  docker cp $OUTPUT_DIR/$PACKAGE_NAME ${APPHUB_CONTAINER}:${APPHUB_PATH}/"
    exit 0
fi

# 创建目录（如果不存在）
docker exec "$APPHUB_CONTAINER" mkdir -p "$APPHUB_PATH"

# 上传包
docker cp "$OUTPUT_DIR/$PACKAGE_NAME" "${APPHUB_CONTAINER}:${APPHUB_PATH}/"

# 验证上传
if docker exec "$APPHUB_CONTAINER" ls "$APPHUB_PATH/$PACKAGE_NAME" > /dev/null 2>&1; then
    print_success "已上传到 AppHub: ${APPHUB_PATH}/${PACKAGE_NAME}"
    
    # 获取 AppHub URL（从 docker-compose.yml 或环境变量）
    APPHUB_PORT=$(docker port "$APPHUB_CONTAINER" 80 2>/dev/null | cut -d: -f2 || echo "8081")
    APPHUB_URL="http://localhost:${APPHUB_PORT}/pkgs/slurm-apk/${PACKAGE_NAME}"
    
    print_info "下载 URL: $APPHUB_URL"
    print_info "内网 URL: http://apphub/pkgs/slurm-apk/${PACKAGE_NAME}"
else
    print_error "上传到 AppHub 失败"
    exit 1
fi

# 创建符号链接（latest 版本）
print_info "创建 latest 符号链接..."
docker exec "$APPHUB_CONTAINER" sh -c "cd $APPHUB_PATH && ln -sf $PACKAGE_NAME slurm-client-latest-alpine.tar.gz"

print_success "✅ 全部完成！"
echo ""
echo "📦 使用方法（在 Dockerfile 中）："
echo ""
cat << 'USAGE_EOF'
# 下载并安装 SLURM 客户端
RUN set -eux; \
    wget -q http://apphub/pkgs/slurm-apk/slurm-client-latest-alpine.tar.gz -O /tmp/slurm.tar.gz; \
    cd /tmp; \
    tar xzf slurm.tar.gz; \
    ./install.sh; \
    rm -rf /tmp/slurm.tar.gz /tmp/install.sh

# 验证安装
RUN sinfo --version
USAGE_EOF
