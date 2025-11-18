#!/bin/bash
set -euo pipefail

# SLURM 二进制编译脚本
# 用于在 Ubuntu 22.04 环境中编译 SLURM 客户端工具

echo "=== SLURM Binary Build Script ==="

# 检测架构
ARCH=$(uname -m)
echo "Architecture: ${ARCH}"

# 查找 SLURM 源码包
TARBALL=$(ls slurm-*.tar.bz2 2>/dev/null | head -1)
if [ -z "$TARBALL" ]; then
    echo "ERROR: No SLURM tarball found"
    exit 1
fi

echo "Found SLURM tarball: ${TARBALL}"

# 解压源码
tar -xaf "${TARBALL}"
SRCDIR=$(basename "${TARBALL}" .tar.bz2)
cd "${SRCDIR}"

echo ">>> Configuring SLURM..."
./configure \
    --prefix=/usr/local/slurm \
    --sysconfdir=/etc/slurm \
    --disable-debug \
    --without-rpath

echo ">>> Building SLURM (full build)..."
make -j$(nproc)

echo ">>> Collecting SLURM binaries..."
mkdir -p /out/packages/${ARCH}/bin
mkdir -p /out/packages/${ARCH}/lib

# 收集客户端工具
for cmd in sinfo squeue scontrol scancel sbatch srun salloc sacct sacctmgr; do
    if [ -f "src/${cmd}/${cmd}" ]; then
        cp -f "src/${cmd}/${cmd}" /out/packages/${ARCH}/bin/
        chmod +x /out/packages/${ARCH}/bin/${cmd}
        echo "  ✓ Collected: ${cmd}"
    elif [ -f "src/${cmd}/.libs/${cmd}" ]; then
        cp -f "src/${cmd}/.libs/${cmd}" /out/packages/${ARCH}/bin/
        chmod +x /out/packages/${ARCH}/bin/${cmd}
        echo "  ✓ Collected: ${cmd} (from .libs)"
    else
        echo "  ✗ Not found: ${cmd}"
    fi
done

echo ">>> Collecting SLURM libraries..."
if [ -d "src/api/.libs" ]; then
    find src/api/.libs -name "libslurm*.so*" -type f -exec cp {} /out/packages/${ARCH}/lib/ \; || true
fi
if [ -d "src/common/.libs" ]; then
    find src/common/.libs -name "libslurm*.so*" -type f -exec cp {} /out/packages/${ARCH}/lib/ \; || true
fi

# 提取版本号
VERSION=$(echo "${SRCDIR}" | grep -oP '\d+\.\d+\.\d+' || echo 'unknown')
echo "${VERSION}" > /out/packages/${ARCH}/VERSION

echo ""
echo "📦 SLURM binaries for ${ARCH}:"
ls -lh /out/packages/${ARCH}/bin/ || true
echo ""
echo "📚 SLURM libraries for ${ARCH}:"
ls -lh /out/packages/${ARCH}/lib/ || true
echo ""
echo "✓ Build completed successfully!"
