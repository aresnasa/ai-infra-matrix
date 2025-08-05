#!/bin/bash

# 整理 src 目录脚本
echo "🔧 整理 src 目录..."

cd src

# 创建 src 专用归档目录
mkdir -p ../archive/src_archive

# 移动 src 中的开发文件
echo "📦 归档 src 中的开发文件..."

# 移动测试和开发文件
mv test_* ../archive/src_archive/ 2>/dev/null || true
mv *test* ../archive/src_archive/ 2>/dev/null || true
mv quick-* ../archive/src_archive/ 2>/dev/null || true
mv run-* ../archive/src_archive/ 2>/dev/null || true
mv *.sh ../archive/src_archive/ 2>/dev/null || true
mv *.js ../archive/src_archive/ 2>/dev/null || true
mv *.html ../archive/src_archive/ 2>/dev/null || true
mv *.txt ../archive/src_archive/ 2>/dev/null || true
mv *.md ../archive/src_archive/ 2>/dev/null || true
mv *.json ../archive/src_archive/ 2>/dev/null || true
mv *.crt ../archive/src_archive/ 2>/dev/null || true
mv *.yaml ../archive/src_archive/ 2>/dev/null || true
mv Dockerfile.* ../archive/src_archive/ 2>/dev/null || true
mv cookies.txt ../archive/src_archive/ 2>/dev/null || true
mv *.ipynb ../archive/src_archive/ 2>/dev/null || true

# 移动开发目录
mv archive ../archive/src_archive/ 2>/dev/null || true
mv docs ../archive/src_archive/ 2>/dev/null || true
mv dev_doc ../archive/src_archive/ 2>/dev/null || true
mv tests ../archive/src_archive/ 2>/dev/null || true
mv tools ../archive/src_archive/ 2>/dev/null || true
mv shared ../archive/src_archive/ 2>/dev/null || true
mv python ../archive/src_archive/ 2>/dev/null || true
mv node_modules ../archive/src_archive/ 2>/dev/null || true

cd ..

echo "✨ src 目录整理完成！"
echo ""
echo "📁 保留的 src 结构："
echo "src/"
echo "├── backend/          # 后端 API 代码"
echo "├── frontend/         # 前端 React 代码"
echo "├── jupyterhub/       # JupyterHub 配置"
echo "├── nginx/            # nginx 配置"
echo "└── docker/           # Docker 配置"
echo ""
echo "🗃️ 归档位置: archive/src_archive/"
