#!/bin/bash

# 修复所有使用 web-v2/backend 路径的文件
files=(
    "internal/services/session_service.go"
    "internal/services/host_service.go"
    "internal/services/variable_task_service.go"
    "internal/services/project_service.go"
    "internal/handlers/user_handler.go"
)

# 对每个文件执行替换
for file in "${files[@]}"; do
    if [ -f "$file" ]; then
        echo "正在修复 $file"
        sed -i '' 's|web-v2/backend|ansible-playbook-generator-backend|g' "$file"
        echo "已完成 $file"
    else
        echo "文件 $file 不存在"
    fi
done

echo "所有导入路径已修复！"
