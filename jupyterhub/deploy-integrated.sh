#!/bin/bash

echo "=== AI-Infra-Matrix JupyterHub 集成修复部署脚本 ==="

# 停止现有容器
echo "停止现有JupyterHub容器..."
docker stop ai-infra-jupyterhub-test ai-infra-jupyterhub 2>/dev/null || true
docker rm ai-infra-jupyterhub-test ai-infra-jupyterhub 2>/dev/null || true

# 构建新镜像
echo "构建集成修复的JupyterHub镜像..."
docker build -t ai-infra-jupyterhub-integrated:latest .

if [ $? -eq 0 ]; then
    echo "✅ JupyterHub镜像构建成功"
    
    # 启动新容器
    echo "启动JupyterHub集成版本..."
    docker run -d \
        --name ai-infra-jupyterhub-integrated \
        --network src_ansible-network \
        -p 8088:8000 \
        -e AI_INFRA_BACKEND_URL=http://backend:8082 \
        -e JUPYTERHUB_AUTO_LOGIN=true \
        -v $(pwd):/srv/jupyterhub/config:ro \
        ai-infra-jupyterhub-integrated:latest \
        jupyterhub -f /srv/jupyterhub/config/ai_infra_jupyterhub_config.py
    
    if [ $? -eq 0 ]; then
        echo "✅ JupyterHub集成版本启动成功"
        echo "🔗 访问地址: http://localhost:8088"
        echo "📝 配置修复项目:"
        echo "   - 端口配置: 前端配置已修正为8088"
        echo "   - 登录态共享: JWT token通过URL参数传递"
        echo "   - Dockerfile: 已改为python:3.13-alpine优化构建"
        echo "   - 重定向循环: 自定义认证器已修复"
        
        echo "等待JupyterHub启动..."
        sleep 5
        
        echo "检查容器状态..."
        docker ps | grep ai-infra-jupyterhub-integrated
        
        echo "检查日志..."
        docker logs ai-infra-jupyterhub-integrated --tail 10
    else
        echo "❌ JupyterHub集成版本启动失败"
        docker logs ai-infra-jupyterhub-integrated
    fi
else
    echo "❌ JupyterHub镜像构建失败"
fi

echo "=== 部署完成 ==="
