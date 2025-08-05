#!/bin/bash

# 项目文件整理脚本
# 将开发过程中的临时文件移动到 archive 文件夹

echo "🗂️  开始整理项目文件..."

# 创建 archive 目录结构
mkdir -p archive/{reports,tests,scripts,notebooks,logs,configs}

echo "📁 创建归档目录结构完成"

# 移动报告文件
echo "📋 归档报告文件..."
mv AI_INFRA_UNIFIED_GUIDE.md archive/reports/ 2>/dev/null || true
mv BACKEND_LOGIN_ISSUE_REPORT.md archive/reports/ 2>/dev/null || true
mv CLEANUP_COMPLETION_REPORT.md archive/reports/ 2>/dev/null || true
mv INFINITE_REDIRECT_RESOLUTION_SUCCESS.md archive/reports/ 2>/dev/null || true
mv JUPYTERHUB_INFINITE_REDIRECT_SOLUTION.md archive/reports/ 2>/dev/null || true
mv JUPYTERHUB_OPTIMIZATION_COMPLETE.md archive/reports/ 2>/dev/null || true
mv JUPYTERHUB_TOKEN_LOGIN_SOLUTION.md archive/reports/ 2>/dev/null || true
mv NGINX_JUPYTERHUB_FIX_SUCCESS_REPORT.md archive/reports/ 2>/dev/null || true
mv NGINX_UNIFIED_DEPLOYMENT_REPORT.md archive/reports/ 2>/dev/null || true
mv PROJECT_COMPLETION_REPORT.md archive/reports/ 2>/dev/null || true
mv UNIFIED_DEPLOYMENT_SUCCESS.md archive/reports/ 2>/dev/null || true

# 移动测试文件
echo "🧪 归档测试文件..."
mv test_*.py archive/tests/ 2>/dev/null || true
mv simple_jupyterhub_test.py archive/tests/ 2>/dev/null || true
mv clear_cookies_test.py archive/tests/ 2>/dev/null || true

# 移动开发脚本
echo "📜 归档开发脚本..."
mv cleanup_jupyterhub_configs.sh archive/scripts/ 2>/dev/null || true
mv cleanup_project.sh archive/scripts/ 2>/dev/null || true
mv docker-deploy-jupyterhub.sh archive/scripts/ 2>/dev/null || true
mv docker-deploy.sh archive/scripts/ 2>/dev/null || true
mv fix_nginx_jupyterhub.sh archive/scripts/ 2>/dev/null || true
mv jupyterhub-dev.sh archive/scripts/ 2>/dev/null || true
mv migrate_to_postgresql.sh archive/scripts/ 2>/dev/null || true

# 移动开发 notebook
echo "📓 归档开发 notebook..."
mv fix-auth-and-jupyter-issues.ipynb archive/notebooks/ 2>/dev/null || true
mv jupyterhub-auth-diagnosis.ipynb archive/notebooks/ 2>/dev/null || true
mv jupyterhub-login-debug.ipynb archive/notebooks/ 2>/dev/null || true
mv test_jupyterhub_login_complete.ipynb archive/notebooks/ 2>/dev/null || true

# 移动日志文件
echo "📊 归档日志文件..."
mv jupyterhub.log archive/logs/ 2>/dev/null || true
mv cookies.txt archive/logs/ 2>/dev/null || true
mv log/ archive/logs/jupyterhub/ 2>/dev/null || true

# 移动临时配置文件
echo "⚙️  归档临时配置文件..."
mv auth_functions_fix.js archive/configs/ 2>/dev/null || true
mv jupyterhub_auto_login.py archive/configs/ 2>/dev/null || true
mv requirements-jupyterhub.txt archive/configs/ 2>/dev/null || true

# 保留必要的目录但移动不必要的内容
echo "🏗️  整理目录结构..."

# 检查并移动 dev_doc 中的过时文档
if [ -d "dev_doc" ]; then
    mkdir -p archive/dev_docs
    cp -r dev_doc/* archive/dev_docs/ 2>/dev/null || true
    # 保留 dev_doc 但删除冗余内容，只保留最新的架构文档
    find dev_doc -name "*.md" -not -path "*/01-01-ai-middleware-architecture.md" -not -path "*/02-03-deployment-guide.md" -exec rm {} \; 2>/dev/null || true
fi

# 整理 notebooks 目录 - 只保留生产相关的
if [ -d "notebooks" ]; then
    mkdir -p archive/old_notebooks
    # 移动所有旧的调试 notebook 到归档
    find notebooks -name "*debug*" -exec mv {} archive/old_notebooks/ \; 2>/dev/null || true
    find notebooks -name "*test*" -exec mv {} archive/old_notebooks/ \; 2>/dev/null || true
fi

echo "✨ 项目文件整理完成！"
echo ""
echo "📁 当前项目结构："
echo "├── 核心配置文件："
echo "│   ├── docker-compose.yml (主要部署配置)"
echo "│   ├── deploy.sh (生产部署脚本)"
echo "│   └── README.md (项目说明)"
echo "├── 源代码目录："
echo "│   ├── src/ (JupyterHub配置、nginx配置)"
echo "│   ├── docker/ (Docker镜像配置)"
echo "│   └── jupyterhub/ (JupyterHub运行配置)"
echo "├── 数据目录："
echo "│   ├── data/ (持久化数据)"
echo "│   └── shared/ (共享存储)"
echo "└── 文档目录："
echo "    ├── docs/ (用户文档)"
echo "    └── dev_doc/ (开发文档 - 精简版)"
echo ""
echo "🗃️  已归档内容："
echo "├── archive/reports/ (开发报告)"
echo "├── archive/tests/ (测试文件)"
echo "├── archive/scripts/ (开发脚本)"
echo "├── archive/notebooks/ (调试notebook)"
echo "├── archive/logs/ (日志文件)"
echo "└── archive/configs/ (临时配置)"
echo ""
echo "🎯 项目现在更加简洁和专业！"
