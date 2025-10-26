#!/bin/bash
# ====================================================================
# 快速修复 backend-init 脚本
# ====================================================================

set -e

echo "======================================================================"
echo "快速修复 backend-init"
echo "======================================================================"

# 步骤1: 停止服务
echo "停止 backend-init 和 backend..."
docker-compose stop backend-init backend

# 步骤2: 删除旧容器
echo "删除旧的 backend-init 容器..."
docker-compose rm -f backend-init

# 步骤3: 重新构建 backend-init
echo "重新构建 backend-init 镜像..."
docker-compose build backend-init

# 步骤4: 运行初始化
echo "运行 backend-init 初始化..."
docker-compose up backend-init

# 步骤5: 重启 backend
echo "重启 backend 服务..."
docker-compose up -d backend

echo ""
echo "======================================================================"
echo "✓ 修复完成！"
echo "======================================================================"
echo ""
echo "验证修复:"
echo "  docker-compose exec postgres psql -U postgres -d ai_infra_matrix -c \"\\d slurm_tasks\" | grep task_id"
echo ""
echo "查看日志:"
echo "  docker-compose logs backend-init"
echo "  docker-compose logs backend | tail -20"
echo ""
