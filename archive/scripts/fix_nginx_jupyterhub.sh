#!/bin/bash

# AI Infrastructure Matrix - Nginx和JupyterHub启动修复脚本
# 版本: v1.0.0
# 解决问题: JupyterHub数据库模式不匹配 + Nginx找不到JupyterHub容器

set -e

echo "🔧 AI Infrastructure Matrix - Nginx和JupyterHub修复脚本"
echo "============================================================"

# 1. 停止所有相关容器
echo "📋 步骤1: 停止相关容器"
echo "停止nginx容器..."
docker stop ai-infra-nginx || true

echo "停止jupyterhub容器..."
docker stop ai-infra-jupyterhub || true

echo "删除已退出的jupyterhub容器..."
docker rm ai-infra-jupyterhub || true

# 2. 清理JupyterHub数据库模式
echo ""
echo "📋 步骤2: 清理JupyterHub数据库模式"
echo "连接到PostgreSQL容器并清理JupyterHub相关表..."

# 连接到PostgreSQL并删除JupyterHub表（如果存在）
docker exec ai-infra-postgres psql -U postgres -d ansible_playbook_generator -c "
DROP TABLE IF EXISTS jupyterhub_users CASCADE;
DROP TABLE IF EXISTS jupyterhub_spawners CASCADE;
DROP TABLE IF EXISTS jupyterhub_services CASCADE;
DROP TABLE IF EXISTS jupyterhub_tokens CASCADE;
DROP TABLE IF EXISTS jupyterhub_oauth_codes CASCADE;
DROP TABLE IF EXISTS jupyterhub_groups CASCADE;
DROP TABLE IF EXISTS jupyterhub_user_group_map CASCADE;
DROP TABLE IF EXISTS alembic_version CASCADE;
SELECT 'JupyterHub表已清理' as status;
"

echo "JupyterHub数据库表已清理完成"

# 3. 重建JupyterHub镜像（确保最新配置）
echo ""
echo "📋 步骤3: 重建JupyterHub镜像"
echo "重建JupyterHub Docker镜像..."
docker compose build jupyterhub

# 4. 启动JupyterHub（不依赖nginx）
echo ""
echo "📋 步骤4: 启动JupyterHub服务"
echo "启动JupyterHub容器..."
docker compose up -d jupyterhub

# 5. 等待JupyterHub启动并初始化数据库
echo ""
echo "📋 步骤5: 等待JupyterHub初始化"
echo "等待JupyterHub容器启动并初始化数据库..."

# 等待JupyterHub容器变为healthy状态
attempt=0
max_attempts=30
while [ $attempt -lt $max_attempts ]; do
    if docker ps --filter "name=ai-infra-jupyterhub" --filter "status=running" --format "{{.Status}}" | grep -q "Up"; then
        echo "✅ JupyterHub容器已启动"
        break
    fi
    
    attempt=$((attempt + 1))
    echo "⏳ 等待JupyterHub启动... ($attempt/$max_attempts)"
    sleep 2
done

if [ $attempt -eq $max_attempts ]; then
    echo "❌ JupyterHub启动超时，检查日志："
    docker logs ai-infra-jupyterhub --tail 20
    exit 1
fi

# 6. 检查JupyterHub健康状态
echo ""
echo "📋 步骤6: 检查JupyterHub状态"
sleep 5  # 给JupyterHub一些时间完全启动

echo "检查JupyterHub日志中的错误..."
docker logs ai-infra-jupyterhub --tail 10

# 7. 启动Nginx
echo ""
echo "📋 步骤7: 启动Nginx反向代理"
echo "重启Nginx容器..."
docker compose up -d nginx

# 8. 等待Nginx启动
echo ""
echo "📋 步骤8: 验证服务状态"
sleep 3

# 检查所有容器状态
echo "检查所有容器状态："
docker ps --filter "name=ai-infra" --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}"

# 9. 测试连接
echo ""
echo "📋 步骤9: 测试服务连接"

# 测试nginx到jupyterhub的连接
echo "测试nginx到jupyterhub的内部连接..."
if docker exec ai-infra-nginx wget -q --spider http://ai-infra-jupyterhub:8000/hub/health; then
    echo "✅ Nginx -> JupyterHub 连接正常"
else
    echo "❌ Nginx -> JupyterHub 连接失败"
fi

# 10. 显示最终状态
echo ""
echo "🎉 修复完成！"
echo "============================================================"
echo ""
echo "📊 最终状态检查:"
echo ""

# 显示容器状态
echo "容器状态:"
docker ps --filter "name=ai-infra-nginx\|ai-infra-jupyterhub" --format "table {{.Names}}\t{{.Status}}"
echo ""

# 显示任何错误日志
echo "最近错误日志 (如有):"
echo "Nginx:"
docker logs ai-infra-nginx --tail 3 2>/dev/null | grep -i error || echo "  无错误"
echo "JupyterHub:"
docker logs ai-infra-jupyterhub --tail 3 2>/dev/null | grep -i error || echo "  无错误"

echo ""
echo "✅ 修复脚本执行完成"
echo "🌐 现在可以通过 http://localhost 访问应用"
echo "📚 JupyterHub访问路径: http://localhost/jupyter/"
echo "🔧 如果仍有问题，请检查容器日志："
echo "   docker logs ai-infra-nginx"
echo "   docker logs ai-infra-jupyterhub"
