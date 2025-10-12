#!/usr/bin/env bash
# JupyterHub 配置验证脚本
# 用于验证 JupyterHub 配置渲染是否正确

# 不使用 set -e，以便看到所有验证结果
# set -e

echo "=================================="
echo "JupyterHub 配置验证脚本"
echo "=================================="
echo ""

# 定义颜色
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 验证结果
PASSED=0
FAILED=0

# 验证函数
check_config() {
    local file=$1
    local line_num=$2
    local expected=$3
    local description=$4
    
    local actual=$(sed -n "${line_num}p" "$file" | xargs)
    
    if [ "$actual" = "$expected" ]; then
        echo -e "${GREEN}✓${NC} $description"
        echo "  文件: $file:$line_num"
        echo "  期望: $expected"
        echo "  实际: $actual"
        ((PASSED++))
    else
        echo -e "${RED}✗${NC} $description"
        echo "  文件: $file:$line_num"
        echo "  期望: $expected"
        echo "  实际: $actual"
        ((FAILED++))
    fi
    echo ""
}

# 验证 base_url（应该是路径，不是完整URL）
check_base_url() {
    local file=$1
    local actual=$(grep "^c.JupyterHub.base_url" "$file" | head -1)
    local expected="c.JupyterHub.base_url = '/jupyter/'"
    
    if [ "$actual" = "$expected" ]; then
        echo -e "${GREEN}✓${NC} base_url 配置正确"
        echo "  文件: $file"
        echo "  值: $actual"
        ((PASSED++))
    else
        echo -e "${RED}✗${NC} base_url 配置错误"
        echo "  文件: $file"
        echo "  期望: $expected"
        echo "  实际: $actual"
        ((FAILED++))
    fi
    echo ""
}

# 验证 bind_url（不应该包含重复的URL）
check_bind_url() {
    local file=$1
    local actual=$(grep "^c.JupyterHub.bind_url" "$file" | head -1)
    local expected="c.JupyterHub.bind_url = 'http://0.0.0.0:8000/jupyter/'"
    
    # 检查是否包含重复的 http://
    if echo "$actual" | grep -q "http://.*http://"; then
        echo -e "${RED}✗${NC} bind_url 包含重复的 URL"
        echo "  文件: $file"
        echo "  值: $actual"
        ((FAILED++))
    elif [ "$actual" = "$expected" ]; then
        echo -e "${GREEN}✓${NC} bind_url 配置正确"
        echo "  文件: $file"
        echo "  值: $actual"
        ((PASSED++))
    else
        echo -e "${YELLOW}⚠${NC} bind_url 配置可能有问题"
        echo "  文件: $file"
        echo "  期望: $expected"
        echo "  实际: $actual"
        ((FAILED++))
    fi
    echo ""
}

# 验证 hub_connect_url（不应该包含 base_url 路径）
check_hub_connect_url() {
    local file=$1
    local actual=$(grep "^c.JupyterHub.hub_connect_url" "$file" | head -1)
    local expected="c.JupyterHub.hub_connect_url = 'http://jupyterhub:8081'"
    
    # 检查是否包含 /jupyter/ 路径
    if echo "$actual" | grep -q "/jupyter/"; then
        echo -e "${RED}✗${NC} hub_connect_url 不应包含 /jupyter/ 路径"
        echo "  文件: $file"
        echo "  值: $actual"
        ((FAILED++))
    elif [ "$actual" = "$expected" ]; then
        echo -e "${GREEN}✓${NC} hub_connect_url 配置正确"
        echo "  文件: $file"
        echo "  值: $actual"
        ((PASSED++))
    else
        echo -e "${YELLOW}⚠${NC} hub_connect_url 配置可能有问题"
        echo "  文件: $file"
        echo "  期望: $expected"
        echo "  实际: $actual"
        ((FAILED++))
    fi
    echo ""
}

# 配置文件列表
CONFIG_FILES=(
    "src/jupyterhub/jupyterhub_config_generated.py"
    "src/jupyterhub/jupyterhub_config_development_generated.py"
    "src/jupyterhub/jupyterhub_config_production_generated.py"
)

# 验证每个配置文件
for config_file in "${CONFIG_FILES[@]}"; do
    echo "=================================="
    echo "验证: $config_file"
    echo "=================================="
    echo ""
    
    if [ ! -f "$config_file" ]; then
        echo -e "${RED}✗${NC} 配置文件不存在: $config_file"
        ((FAILED++))
        echo ""
        continue
    fi
    
    check_base_url "$config_file"
    check_bind_url "$config_file"
    check_hub_connect_url "$config_file"
done

# 输出总结
echo "=================================="
echo "验证总结"
echo "=================================="
echo -e "通过: ${GREEN}$PASSED${NC}"
echo -e "失败: ${RED}$FAILED${NC}"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ 所有配置验证通过！${NC}"
    exit 0
else
    echo -e "${RED}✗ 配置验证失败，请检查上述错误${NC}"
    echo ""
    echo "修复建议："
    echo "1. 运行: bash build.sh render-templates jupyterhub"
    echo "2. 检查模板文件: src/jupyterhub/templates/jupyterhub_config.py.tpl"
    echo "3. 查看修复文档: docs/JUPYTERHUB_CONFIG_FIX.md"
    exit 1
fi
