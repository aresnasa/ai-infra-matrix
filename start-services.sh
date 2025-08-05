#!/bin/bash

# AI Infrastructure Matrix - 一键启动脚本
echo "🚀 启动 AI Infrastructure Matrix..."

# 清理旧的容器和网络（可选）
echo "🧹 清理旧的容器..."
docker-compose down --remove-orphans 2>/dev/null || true

# 构建所有服务
echo "🔨 构建服务镜像..."
docker-compose build

# 启动所有服务
echo "🌟 启动所有服务..."
docker-compose up -d

# 等待初始化完成
echo "🔧 等待后端初始化完成（创建admin用户）..."
docker-compose logs -f backend-init &
LOGS_PID=$!

# 等待初始化服务完成
while [ "$(docker-compose ps -q backend-init)" ]; do
    if [ "$(docker inspect --format='{{.State.Status}}' ai-infra-backend-init 2>/dev/null)" = "exited" ]; then
        break
    fi
    sleep 2
done

# 停止日志跟踪
kill $LOGS_PID 2>/dev/null || true

# 检查初始化是否成功
if [ "$(docker inspect --format='{{.State.ExitCode}}' ai-infra-backend-init 2>/dev/null)" = "0" ]; then
    echo "✅ 后端初始化完成，admin用户已创建"
else
    echo "❌ 后端初始化失败，请检查日志"
fi

# 等待其他服务启动
echo "⏳ 等待其他服务启动完成..."
sleep 10

# 检查服务状态
echo "📊 检查服务状态..."
docker-compose ps

echo ""
echo "✅ AI Infrastructure Matrix 启动完成！"
echo ""
echo "🌐 访问地址:"
echo "  主应用: http://localhost:8080"
echo "  JupyterHub: http://localhost:8080/jupyter"
echo "  后端API: http://localhost:8080/api"
echo ""
echo "👤 默认admin用户信息:"
echo "  用户名: admin"
echo "  密码: admin123"
echo "  邮箱: admin@example.com"
echo ""
echo "📝 管理命令:"
echo "  查看日志: docker-compose logs -f [service_name]"
echo "  停止服务: docker-compose down"
echo "  重启服务: docker-compose restart [service_name]"
echo ""
echo "🔍 验证服务健康状态:"
echo "  docker-compose ps"
