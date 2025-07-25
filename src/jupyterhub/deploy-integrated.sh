#!/bin/bash

echo "=== AI-Infra-Matrix JupyterHub 集成修复部署脚本 (Docker Compose版本) ==="

# 切换到src目录
cd "$(dirname "$0")/.."

# 设置环境变量启用JupyterHub profile
export COMPOSE_PROFILES=jupyterhub

# 停止现有容器
echo "停止现有JupyterHub容器..."
docker-compose down jupyterhub 2>/dev/null || true
docker stop ai-infra-jupyterhub-test ai-infra-jupyterhub ai-infra-jupyterhub-integrated 2>/dev/null || true
docker rm ai-infra-jupyterhub-test ai-infra-jupyterhub ai-infra-jupyterhub-integrated 2>/dev/null || true

# 使用docker-compose构建和启动JupyterHub服务
echo "使用Docker Compose构建JupyterHub服务..."
docker-compose build --no-cache jupyterhub

if [ $? -eq 0 ]; then
    echo "✅ JupyterHub镜像构建成功"
    
    # 启动依赖服务
    echo "启动依赖服务（postgres, redis, backend）..."
    docker-compose up -d postgres redis backend
    
    # 等待依赖服务就绪
    echo "等待后端服务就绪..."
    sleep 15
    
    # 启动JupyterHub服务
    echo "启动JupyterHub集成版本..."
    docker-compose up -d jupyterhub
    
    if [ $? -eq 0 ]; then
        echo "✅ JupyterHub集成版本启动成功"
        echo "🔗 访问地址: http://localhost:8088"
        echo "📝 配置修复项目:"
        echo "   - 端口配置: 前端配置已修正为8088"
        echo "   - 登录态共享: JWT token通过URL参数传递"
        echo "   - Dockerfile: 已改为python:3.13-alpine优化构建"
        echo "   - 重定向循环: 自定义认证器已修复"
        echo "   - 目录结构: JupyterHub已移动到src目录"
        echo "   - 幂等构建: 使用Docker Compose确保一致性"
        echo "   - Profile启用: 使用jupyterhub profile启动"
        
        echo "等待JupyterHub启动..."
        sleep 15
        
        echo "检查容器状态..."
        docker-compose ps jupyterhub
        
        echo "检查日志..."
        docker-compose logs jupyterhub --tail 15
        
        echo "测试JupyterHub访问..."
        curl -I http://localhost:8088/hub/health 2>/dev/null && echo "✅ JupyterHub健康检查通过" || echo "⚠️  JupyterHub健康检查失败，检查是否还在启动中"
        
        echo "测试基本访问..."
        curl -I http://localhost:8088/ 2>/dev/null && echo "✅ JupyterHub基本访问正常" || echo "⚠️  JupyterHub基本访问可能有问题"
    else
        echo "❌ JupyterHub集成版本启动失败"
        docker-compose logs jupyterhub
    fi
else
    echo "❌ JupyterHub镜像构建失败"
fi

echo "=== 部署完成 ==="
echo "💡 提示："
echo "   - 使用 'COMPOSE_PROFILES=jupyterhub docker-compose up -d' 启动完整环境"
echo "   - 使用 'docker-compose logs jupyterhub -f' 查看实时日志"
echo "   - 使用 'docker-compose down' 停止所有服务"
