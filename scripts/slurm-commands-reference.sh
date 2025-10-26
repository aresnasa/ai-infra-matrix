#!/bin/bash
# SLURM 客户端快速命令参考
# 用于 ai-infra-backend 容器

# =============================================================================
# 正确的 docker exec 命令格式
# =============================================================================

# ❌ 错误: 直接使用分号分隔的复合命令
# docker exec ai-infra-backend "source /etc/profile ;sinfo"

# ✅ 正确: 使用 sh -c 执行复合命令
docker exec ai-infra-backend sh -c 'source /etc/profile && sinfo'

# ✅ 正确: 使用 bash -c 执行复合命令
docker exec ai-infra-backend bash -c 'source /etc/profile && sinfo'

# ✅ 正确: 直接执行单个命令（如果不需要 source profile）
docker exec ai-infra-backend sinfo

# =============================================================================
# 常用 SLURM 命令
# =============================================================================

# 1. 查看集群节点状态
docker exec ai-infra-backend sh -c 'sinfo'

# 2. 查看详细节点信息
docker exec ai-infra-backend sh -c 'sinfo -N -l'

# 3. 查看分区摘要
docker exec ai-infra-backend sh -c 'sinfo -s'

# 4. 查看作业队列
docker exec ai-infra-backend sh -c 'squeue'

# 5. 查看特定用户的作业
docker exec ai-infra-backend sh -c 'squeue -u root'

# 6. 提交简单作业
docker exec ai-infra-backend sh -c 'srun -N1 hostname'

# 7. 查看节点详细配置
docker exec ai-infra-backend sh -c 'scontrol show node'

# 8. 查看分区配置
docker exec ai-infra-backend sh -c 'scontrol show partition'

# 9. 查看作业详情
docker exec ai-infra-backend sh -c 'scontrol show job <job_id>'

# 10. 取消作业
docker exec ai-infra-backend sh -c 'scancel <job_id>'

# =============================================================================
# 验证与调试命令
# =============================================================================

# 检查 SLURM 客户端是否安装
docker exec ai-infra-backend sh -c 'command -v sinfo && sinfo --version'

# 检查配置文件
docker exec ai-infra-backend sh -c 'cat /etc/slurm/slurm.conf'

# 检查网络连接到 slurm-master
docker exec ai-infra-backend sh -c 'nc -zv slurm-master 6817'

# 查看环境变量
docker exec ai-infra-backend sh -c 'env | grep SLURM'

# 进入容器交互式 shell
docker exec -it ai-infra-backend bash

# =============================================================================
# 一键测试脚本
# =============================================================================

# 运行完整测试
./scripts/test-slurm-client.sh

# =============================================================================
# 故障排查
# =============================================================================

# 查看 backend 容器日志
docker logs ai-infra-backend

# 查看 slurm-master 容器日志
docker logs ai-infra-slurm-master

# 重启 backend 容器
docker-compose restart backend

# 重新构建 backend 镜像（如果需要安装 SLURM 客户端）
docker-compose build backend
docker-compose up -d backend
