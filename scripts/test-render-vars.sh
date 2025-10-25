#!/bin/bash

# 测试脚本：验证模板渲染变量是否正确导出

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# 从 build.sh 加载必要的函数
source "$SCRIPT_DIR/build.sh"

# 加载环境变量
echo "正在加载环境变量..."
load_environment_variables

# 检查关键变量是否已导出
echo ""
echo "=========================================="
echo "关键变量检查"
echo "=========================================="
echo "NIGHTINGALE_HOST=${NIGHTINGALE_HOST:-未设置}"
echo "NIGHTINGALE_PORT=${NIGHTINGALE_PORT:-未设置}"
echo "GITEA_ALIAS_ADMIN_TO=${GITEA_ALIAS_ADMIN_TO:-未设置}"
echo "GITEA_ADMIN_EMAIL=${GITEA_ADMIN_EMAIL:-未设置}"
echo "BACKEND_HOST=${BACKEND_HOST:-未设置}"
echo "BACKEND_PORT=${BACKEND_PORT:-未设置}"
echo "FRONTEND_HOST=${FRONTEND_HOST:-未设置}"
echo "FRONTEND_PORT=${FRONTEND_PORT:-未设置}"
echo "JUPYTERHUB_HOST=${JUPYTERHUB_HOST:-未设置}"
echo "JUPYTERHUB_PORT=${JUPYTERHUB_PORT:-未设置}"
echo ""

# 检查 ENV_ 前缀的变量（向后兼容）
echo "=========================================="
echo "ENV_ 前缀变量检查（向后兼容）"
echo "=========================================="
echo "ENV_NIGHTINGALE_HOST=${ENV_NIGHTINGALE_HOST:-未设置}"
echo "ENV_NIGHTINGALE_PORT=${ENV_NIGHTINGALE_PORT:-未设置}"
echo "ENV_GITEA_ALIAS_ADMIN_TO=${ENV_GITEA_ALIAS_ADMIN_TO:-未设置}"
echo ""

# 测试模板渲染
echo "=========================================="
echo "测试模板渲染"
echo "=========================================="
if [[ -f "$SCRIPT_DIR/src/nginx/templates/conf.d/includes/nightingale.conf.tpl" ]]; then
    echo "发现 Nightingale 模板文件"
    
    # 创建临时测试文件
    temp_test_file=$(mktemp)
    echo "proxy_pass http://{{NIGHTINGALE_HOST}}:{{NIGHTINGALE_PORT}}/;" > "$temp_test_file.tpl"
    echo "if (\$user_header = \"admin\") { set \$user_header \"\${GITEA_ALIAS_ADMIN_TO}\"; }" >> "$temp_test_file.tpl"
    
    # 渲染测试
    if render_template "$temp_test_file.tpl" "$temp_test_file.out"; then
        echo ""
        echo "渲染结果："
        cat "$temp_test_file.out"
        echo ""
        
        # 验证渲染是否成功
        if grep -q "{{NIGHTINGALE_HOST}}" "$temp_test_file.out"; then
            echo "❌ 错误：NIGHTINGALE_HOST 未被渲染"
        else
            echo "✅ 成功：NIGHTINGALE_HOST 已渲染"
        fi
        
        if grep -q "{{NIGHTINGALE_PORT}}" "$temp_test_file.out"; then
            echo "❌ 错误：NIGHTINGALE_PORT 未被渲染"
        else
            echo "✅ 成功：NIGHTINGALE_PORT 已渲染"
        fi
        
        if grep -q "\${GITEA_ALIAS_ADMIN_TO}" "$temp_test_file.out"; then
            echo "❌ 错误：GITEA_ALIAS_ADMIN_TO 未被渲染"
        else
            echo "✅ 成功：GITEA_ALIAS_ADMIN_TO 已渲染"
        fi
    else
        echo "渲染失败"
    fi
    
    # 清理
    rm -f "$temp_test_file" "$temp_test_file.tpl" "$temp_test_file.out"
else
    echo "未找到 Nightingale 模板文件"
fi

echo ""
echo "测试完成"
