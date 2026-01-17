#!/bin/bash
set -eux

# 设置中国镜像源，加快 Go 模块下载
export GOPROXY=https://goproxy.cn,https://goproxy.io,direct

echo "=========================================="
echo "更新 backend 项目依赖"
echo "=========================================="
cd /Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix/src/backend

echo "1. 运行 go mod tidy..."
go mod tidy

echo "2. 运行 go mod verify..."
go mod verify

echo "3. 检查直接依赖是否有更新（补丁版本）..."
go get -u=patch ./...

echo "✓ backend 项目更新完成"

echo ""
echo "=========================================="
echo "更新 nightingale 项目依赖"
echo "=========================================="
cd /Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix/third_party/nightingale

echo "1. 运行 go mod tidy..."
go mod tidy

echo "2. 运行 go mod verify..."
go mod verify

echo "3. 检查直接依赖是否有更新（补丁版本）..."
go get -u=patch ./...

echo "✓ nightingale 项目更新完成"

echo ""
echo "=========================================="
echo "所有项目依赖更新完成！"
echo "=========================================="
echo "摘要："
echo "- 清理了未使用的依赖"
echo "- 添加了缺失的依赖"
echo "- 更新了所有可用的补丁版本"
echo "- 验证了模块完整性"
