#!/bin/bash

# 删除冗余文档和配置文件
echo "🧹 清理冗余文档和配置文件..."

cd /Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix

# 删除根目录下的重复报告文件
echo "📁 清理根目录重复报告..."
rm -f JUPYTERHUB_*.md NGINX_*.md INFINITE_*.md INTEGRATION_*.md PROJECT_*.md UNIFIED_*.md BACKEND_*.md

# 删除jupyterhub目录下的冗余文件
echo "📁 清理JupyterHub目录..."
cd src/jupyterhub
rm -f *.sh README.md QUICK_FIX.md TROUBLESHOOTING.md requirements-*.txt
rm -rf __pycache__ templates data notebooks

# 只保留核心文件
echo "✅ 保留核心文件:"
echo "  - Dockerfile"
echo "  - jupyterhub_config.py"
echo "  - backend_integrated_config.py" 
echo "  - requirements.txt"

# 回到根目录
cd /Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix

# 删除dev_doc目录下的大部分文件（保留少数核心文档）
echo "📁 清理开发文档..."
cd dev_doc
# 保留核心架构文档，删除其他
ls -1 | grep -v -E "(01-01-ai-middleware-architecture\.md|02-03-deployment-guide\.md)" | xargs rm -f
cd ..

# 删除docs目录下的冗余文档
echo "📁 清理docs目录..."
if [ -d "docs" ]; then
    cd docs
    ls -1 | grep -v "JUPYTERHUB_UNIFIED_AUTH_GUIDE.md" | xargs rm -f
    cd ..
fi

# 删除examples目录下的过时示例
echo "📁 清理examples目录..."
if [ -d "examples" ]; then
    rm -rf examples/*
fi

echo ""
echo "✅ 清理完成！现在项目结构更精简："
echo "📋 保留的核心文件:"
echo "  🏠 AI_INFRA_UNIFIED_GUIDE.md (统一指南)"
echo "  🐳 docker-compose.yml (服务编排)"
echo "  🔧 src/backend/ (后端核心)"
echo "  🔧 src/frontend/ (前端核心)"
echo "  🔧 src/jupyterhub/ (Jupyter集成)"
echo "  📁 k8s/ (Kubernetes配置)"
echo ""
echo "🎯 项目现在更加精炼和可维护！"
