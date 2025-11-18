#!/bin/bash

set -e

echo "=========================================="
echo "测试 SLURM RPM Builder 阶段"
echo "=========================================="
echo

cd src/apphub

echo "1. 检查 SLURM tarball..."
ls -lh slurm-*.tar.bz2 || { echo "❌ tarball 不存在"; exit 1; }
echo "✓ tarball 存在"
echo

echo "2. 构建 rpm-builder 阶段..."
docker build --target rpm-builder \
  --build-arg BUILD_SLURM=true \
  --build-arg SLURM_VERSION=25.05.4 \
  --build-arg SLURM_TARBALL_PATH=slurm-25.05.4.tar.bz2 \
  -t test-rpm-builder . \
  --progress=plain 2>&1 | tee /tmp/rpm-builder-test.log

echo
echo "3. 检查构建输出..."
echo "=== /out/ 目录内容 ==="
docker run --rm test-rpm-builder ls -la /out/

echo
echo "=== 查找所有 RPM 文件 ==="
docker run --rm test-rpm-builder find /home/builder -name "*.rpm" -type f 2>/dev/null || echo "未找到 RPM 文件"

echo
echo "=== /out/slurm-rpm/ 目录 ==="
docker run --rm test-rpm-builder ls -la /out/slurm-rpm/ 2>/dev/null || echo "目录不存在"

echo
echo "=== 检查 .skip_slurm 标记 ==="
if docker run --rm test-rpm-builder test -f /out/.skip_slurm 2>/dev/null; then
    echo "⚠️  发现 .skip_slurm 标记文件 - SLURM 构建被跳过"
else
    echo "✓ 没有 .skip_slurm 标记"
fi

echo
echo "=========================================="
echo "测试完成"
echo "=========================================="
echo "详细日志：/tmp/rpm-builder-test.log"
echo "查看最后 100 行：tail -100 /tmp/rpm-builder-test.log"
