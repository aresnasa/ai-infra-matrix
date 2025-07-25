#!/bin/bash

echo "=== Docker Compose 配置验证脚本 ==="

cd "$(dirname "$0")"

# 设置环境变量启用JupyterHub profile
export COMPOSE_PROFILES=jupyterhub

echo "1. 验证Docker Compose文件语法..."
docker-compose config > /dev/null
if [ $? -eq 0 ]; then
    echo "✅ Docker Compose语法正确"
else
    echo "❌ Docker Compose语法错误"
    exit 1
fi

echo "2. 检查JupyterHub服务配置..."
docker-compose config --services | grep jupyterhub > /dev/null
if [ $? -eq 0 ]; then
    echo "✅ JupyterHub服务已定义"
else
    echo "❌ 未找到JupyterHub服务"
    exit 1
fi

echo "3. 验证构建上下文..."
if [ -d "./jupyterhub" ]; then
    echo "✅ JupyterHub构建上下文存在"
else
    echo "❌ JupyterHub构建上下文不存在"
    exit 1
fi

echo "4. 检查Dockerfile..."
if [ -f "./jupyterhub/Dockerfile" ]; then
    echo "✅ JupyterHub Dockerfile存在"
else
    echo "❌ JupyterHub Dockerfile不存在"
    exit 1
fi

echo "5. 验证配置文件..."
if [ -f "./jupyterhub/ai_infra_jupyterhub_config.py" ]; then
    echo "✅ JupyterHub配置文件存在"
else
    echo "❌ JupyterHub配置文件不存在"
    exit 1
fi

echo "6. 检查端口配置..."
docker-compose config | grep -A 5 -B 5 "8088:8000" > /dev/null
if [ $? -eq 0 ]; then
    echo "✅ 端口映射配置正确 (8088:8000)"
else
    echo "⚠️  端口映射可能需要检查"
fi

echo "7. 验证网络配置..."
docker-compose config | grep "ansible-network" > /dev/null
if [ $? -eq 0 ]; then
    echo "✅ 网络配置正确"
else
    echo "❌ 网络配置有问题"
fi

echo "8. 检查依赖关系..."
docker-compose config | grep -A 3 "depends_on:" | grep "backend" > /dev/null
if [ $? -eq 0 ]; then
    echo "✅ 后端依赖配置正确"
else
    echo "⚠️  后端依赖可能需要检查"
fi

echo "9. 验证Profile配置..."
docker-compose config | grep -A 2 "profiles:" | grep "jupyterhub" > /dev/null
if [ $? -eq 0 ]; then
    echo "✅ Profile配置正确"
else
    echo "⚠️  Profile配置可能需要检查"
fi

echo ""
echo "=== 配置验证完成 ==="
echo "如果所有检查都通过，可以运行以下命令启动服务："
echo "export COMPOSE_PROFILES=jupyterhub"
echo "docker-compose up -d jupyterhub"
echo ""
echo "或者使用集成部署脚本："
echo "./jupyterhub/deploy-integrated.sh"
echo ""
echo "查看JupyterHub服务配置："
echo "COMPOSE_PROFILES=jupyterhub docker-compose config jupyterhub"
