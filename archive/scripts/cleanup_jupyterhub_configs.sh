#!/bin/bash

# 清理JupyterHub冗余配置文件
echo "🧹 开始清理JupyterHub冗余配置文件..."

cd /Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix/src/jupyterhub

# 保留的文件
KEEP_FILES=(
    "jupyterhub_config.py"           # 主配置文件
    "backend_integrated_config.py"   # 统一后端集成配置
    "Dockerfile"                     # Docker构建文件
)

# 需要删除的冗余配置文件
DELETE_FILES=(
    "jwt_config.py"
    "clean_optimized_jwt_config.py"
    "unified_config.py"
    "ultimate_config.py"
    "postgres_authenticator.py"
    "test_config.py"
    "redis_unified_config.py"
    "ai_infra_jupyterhub_config.py"
    "unified_config_simple.py"
    "no_redirect_config.py"
    "basic_config.py"
    "simple_config.py"
    "anti_redirect_config.py"
    "optimized_jwt_config.py"
    "unified_backend_config.py"
    "minimal_config.py"
    "absolute_no_redirect_config.py"
    "universal_config.py"
    "simple_test_config.py"
    "working_jwt_config.py"
    "fixed_config.py"
    "ai_infra_auth.py"
)

echo "📋 将要删除的文件:"
for file in "${DELETE_FILES[@]}"; do
    if [ -f "$file" ]; then
        echo "  - $file"
    fi
done

echo ""
read -p "确认删除这些文件? (y/N): " confirm

if [[ $confirm == [yY] || $confirm == [yY][eE][sS] ]]; then
    echo "🗑️  删除冗余配置文件..."
    
    for file in "${DELETE_FILES[@]}"; do
        if [ -f "$file" ]; then
            rm "$file"
            echo "  ✅ 已删除: $file"
        fi
    done
    
    echo ""
    echo "📁 保留的文件:"
    for file in "${KEEP_FILES[@]}"; do
        if [ -f "$file" ]; then
            echo "  ✅ $file"
        fi
    done
    
    echo ""
    echo "✅ 清理完成！"
    echo "📋 当前目录文件:"
    ls -la *.py 2>/dev/null || echo "  (无Python文件)"
    
else
    echo "❌ 取消删除操作"
fi
