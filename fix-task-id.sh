#!/bin/bash
# ====================================================================
# 快速执行：重建 backend-init 并修复 task_id 类型
# ====================================================================

# 停止服务
docker-compose stop backend backend-init

# 删除旧容器
docker-compose rm -f backend-init backend

# 删除旧镜像（强制完全重新构建）
docker rmi ai-infra-backend-init:v0.3.6-dev 2>/dev/null || true

# 重新构建 backend-init（使用最新修复代码）
docker-compose build --no-cache backend-init

# 运行初始化
docker-compose up backend-init

# 验证结果
echo ""
echo "======================================================================"
echo "验证 task_id 字段类型："
echo "======================================================================"
docker-compose exec postgres psql -U postgres -d ai_infra_matrix -c "\d slurm_tasks" | grep task_id

# 重新构建并启动 backend
echo ""
echo "重新构建并启动 backend..."
docker-compose build backend
docker-compose up -d backend

echo ""
echo "======================================================================"
echo "完成！等待 backend 启动后即可测试扩容功能"
echo "======================================================================"
echo ""
echo "查看 backend 日志:"
echo "  docker-compose logs -f backend"
echo ""
