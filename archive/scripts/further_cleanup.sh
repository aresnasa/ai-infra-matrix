#!/bin/bash

# 进一步精简项目结构脚本
echo "🔧 进一步精简项目结构..."

# 移动不必要的目录到归档
echo "📦 归档额外的目录..."

# 移动 docker-saltstack 到归档（这是实验性的）
if [ -d "docker-saltstack" ]; then
    mv docker-saltstack archive/experimental/
    echo "   ✓ docker-saltstack -> archive/experimental/"
fi

# 移动 examples 到归档（示例代码）
if [ -d "examples" ]; then
    mv examples archive/
    echo "   ✓ examples -> archive/"
fi

# 移动 k8s 到归档（Kubernetes配置，当前项目不需要）
if [ -d "k8s" ]; then
    mv k8s archive/
    echo "   ✓ k8s -> archive/"
fi

# 移动 third-party 到归档
if [ -d "third-party" ]; then
    mv third-party archive/
    echo "   ✓ third-party -> archive/"
fi

# 精简 notebooks 目录，只保留生产相关的
if [ -d "notebooks" ]; then
    # 检查是否有生产相关的 notebook，如果没有则移动整个目录
    production_notebooks=$(find notebooks -name "*.ipynb" | grep -v -E "(test|debug|demo)" | wc -l)
    if [ "$production_notebooks" -eq 0 ]; then
        mv notebooks archive/
        echo "   ✓ notebooks (全部) -> archive/"
    else
        echo "   ℹ️  保留 notebooks 目录 (包含生产相关内容)"
    fi
fi

# 检查 scripts 目录是否为空或只包含开发脚本
if [ -d "scripts" ]; then
    production_scripts=$(find scripts -name "*.sh" | grep -v -E "(test|debug|dev)" | wc -l)
    if [ "$production_scripts" -eq 0 ]; then
        mv scripts archive/
        echo "   ✓ scripts (开发脚本) -> archive/"
    else
        echo "   ℹ️  保留 scripts 目录 (包含生产脚本)"
    fi
fi

# 清理空目录
echo "🧹 清理空目录..."
find . -type d -empty -not -path "./archive/*" -not -path "./.git/*" -delete 2>/dev/null || true

echo ""
echo "✨ 项目进一步精简完成！"
echo ""
echo "📁 最终精简项目结构："
echo "."
echo "├── 🏗️  核心部署配置"
echo "│   ├── docker-compose.yml"
echo "│   ├── deploy.sh"
echo "│   ├── .env.jupyterhub.example"
echo "│   └── README.md"
echo "├── 💻 源代码"
echo "│   ├── src/"
echo "│   │   ├── jupyterhub/          # JupyterHub 配置"
echo "│   │   └── nginx/               # nginx 反向代理配置"
echo "│   ├── docker/                  # Docker 镜像配置"
echo "│   └── jupyterhub/              # JupyterHub 运行时配置"
echo "├── 💾 数据和存储"
echo "│   ├── data/                    # 持久化数据"
echo "│   └── shared/                  # 共享存储"
echo "├── 📚 文档"
echo "│   ├── docs/                    # 用户文档"
echo "│   └── dev_doc/                 # 开发文档（精简版）"
echo "└── 🗃️  开发归档"
echo "    └── archive/                 # 所有开发过程文件"
echo ""
echo "🎯 项目现在非常简洁，只包含生产必需的文件！"
