#!/bin/bash

# 测试get_private_image_name函数修复
set -e

echo "🧪 测试镜像名处理修复"
echo "====================="

# 导入build.sh中的函数
source build.sh

# 测试用例
test_cases=(
    # registry original_image expected_result description
    "    "harbor.example.com/ai-infra|postgres:15-alpine|harbor.example.com/ai-infra/postgres:15-alpine|Harbor风格基础镜像""
    "    "harbor.example.com/ai-infra|redis:7-alpine|harbor.example.com/ai-infra/redis:7-alpine|Harbor风格Redis镜像""
    "    "harbor.example.com/ai-infra|osixia/openldap:stable|harbor.example.com/ai-infra/osixia/openldap:stable|Harbor风格组织镜像""
    "    "harbor.example.com/ai-infra|ai-infra-backend:v0.3.8|harbor.example.com/ai-infra/ai-infra-backend:v0.3.8|Harbor风格AI-Infra镜像""
    "registry.local:5000|postgres:15-alpine|registry.local:5000/postgres:15-alpine|传统风格基础镜像"
    "registry.local:5000|ai-infra-backend:v0.3.8|registry.local:5000/ai-infra/ai-infra-backend:v0.3.8|传统风格AI-Infra镜像"
    "    "harbor.example.com/ai-infra|harbor.example.com/ai-infra/postgres:15-alpine|harbor.example.com/ai-infra/postgres:15-alpine|已包含完整路径的镜像""
)

success_count=0
total_count=${#test_cases[@]}

for test_case in "${test_cases[@]}"; do
    IFS='|' read -r registry original_image expected description <<< "$test_case"
    
    echo ""
    echo "📋 测试: $description"
    echo "   Registry: $registry"
    echo "   Original: $original_image"
    echo "   Expected: $expected"
    
    result=$(get_private_image_name "$original_image" "$registry")
    echo "   Result:   $result"
    
    if [[ "$result" == "$expected" ]]; then
        echo "   ✅ 通过"
        success_count=$((success_count + 1))
    else
        echo "   ❌ 失败"
        echo "   💡 期望: $expected"
        echo "   💡 实际: $result"
    fi
done

echo ""
echo "🎯 测试结果总结"
echo "==============="
echo "总计: $total_count 个测试"
echo "通过: $success_count 个"
echo "失败: $((total_count - success_count)) 个"

if [[ $success_count -eq $total_count ]]; then
    echo "✅ 所有测试通过！镜像名处理修复成功。"
    exit 0
else
    echo "❌ 部分测试失败，需要进一步修复。"
    exit 1
fi
