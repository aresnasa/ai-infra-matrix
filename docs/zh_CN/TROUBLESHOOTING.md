# 故障排除指南

**中文** | **[English](en/TROUBLESHOOTING.md)**

## 常见问题

### 服务无法启动

#### 症状

```bash
$ docker compose up -d
Error response from daemon: container not found
```

#### 解决方案

1. 检查 Docker 服务状态

```bash
docker info
```

2. 清理旧容器

```bash
docker compose down
docker system prune -a
```

3. 重新构建镜像

```bash
./build.sh build-all v0.3.8
docker compose up -d
```

### 端口冲突

#### 症状

```
Error starting userland proxy: listen tcp 0.0.0.0:8080: bind: address already in use
```

#### 解决方案

1. 查找占用端口的进程

```bash
lsof -i :8080
# 或
netstat -tulpn | grep 8080
```

2. 终止占用进程或修改端口

```bash
# 修改 .env 文件
EXTERNAL_PORT=8081
```

### 数据库连接失败

#### 症状

```
Error: connection refused to postgres:5432
```

#### 解决方案

1. 检查数据库容器状态

```bash
docker compose ps postgres
docker compose logs postgres
```

2. 验证连接配置

```bash
# 测试数据库连接
docker exec -it ai-infra-postgres psql -U postgres -d ai-infra-matrix
```

3. 重启数据库服务

```bash
docker compose restart postgres
```

### Slurm 节点 DOWN 状态

#### 症状

节点显示为 DOWN 或 UNKNOWN 状态

#### 解决方案

1. 检查节点连接

```bash
# 在 slurm-master 容器中
docker exec ai-infra-slurm-master sinfo
docker exec ai-infra-slurm-master scontrol show node node01
```

2. 重启 slurmd 服务

```bash
# 在计算节点上
systemctl restart slurmd
```

3. 手动恢复节点

```bash
# 在 slurm-master 容器中
scontrol update NodeName=node01 State=RESUME
```

参考：[Slurm节点恢复指南](SLURM_NODE_RECOVERY_GUIDE.md)

### JupyterHub 无法启动

#### 症状

JupyterHub 用户服务器启动失败

#### 解决方案

1. 检查 JupyterHub 日志

```bash
docker compose logs jupyterhub
```

2. 验证镜像可用性

```bash
docker images | grep singleuser
```

3. 清理旧的用户容器

```bash
docker ps -a | grep jupyter
docker rm -f $(docker ps -a | grep jupyter | awk '{print $1}')
```

### Gitea LFS 上传失败

#### 症状

```
Error: LFS upload failed
```

#### 解决方案

1. 检查 MinIO 状态

```bash
docker compose ps minio
docker compose logs minio
```

2. 验证 S3 配置

```bash
# 检查 Gitea 配置
docker exec ai-infra-gitea cat /data/gitea/conf/app.ini | grep -A 5 "\[lfs\]"
```

3. 测试 MinIO 连接

```bash
docker exec ai-infra-gitea wget -O- http://minio:9000/minio/health/live
```

### 前端白屏

#### 症状

浏览器显示白屏或 404

#### 解决方案

1. 清除浏览器缓存

```bash
# Chrome: Ctrl+Shift+Delete
# Firefox: Ctrl+Shift+Del
```

2. 检查 Nginx 配置

```bash
docker exec ai-infra-nginx nginx -t
docker compose restart nginx
```

3. 重建前端镜像

```bash
./build.sh build src/frontend v0.3.8
docker compose up -d frontend
```

### 监控数据缺失

#### 症状

Nightingale 监控面板无数据

#### 解决方案

1. 检查 Categraf 状态

```bash
docker compose logs categraf
```

2. 验证 Prometheus 连接

```bash
curl http://localhost:9090/-/healthy
```

3. 检查指标采集

```bash
curl http://localhost:9090/api/v1/query?query=up
```

## 日志查看

### 查看所有服务日志

```bash
docker compose logs -f
```

### 查看特定服务日志

```bash
docker compose logs -f backend
docker compose logs -f frontend
docker compose logs -f jupyterhub
docker compose logs -f slurm-master
```

### 查看最近 N 行日志

```bash
docker compose logs --tail=100 backend
```

### 导出日志

```bash
docker compose logs > logs/full-logs.txt
docker compose logs backend > logs/backend.log
```

## 性能问题

### CPU 使用率过高

1. 查看资源使用

```bash
docker stats
```

2. 限制容器资源

```yaml
# docker-compose.yml
services:
  backend:
    deploy:
      resources:
        limits:
          cpus: '2'
          memory: 4G
```

### 内存不足

1. 检查内存使用

```bash
free -h
docker stats --no-stream
```

2. 清理未使用的资源

```bash
docker system prune -a
docker volume prune
```

### 磁盘空间不足

1. 检查磁盘使用

```bash
df -h
du -sh /var/lib/docker/*
```

2. 清理旧镜像和容器

```bash
docker system prune -a --volumes
```

## 网络问题

### 容器无法互相通信

1. 检查网络配置

```bash
docker network ls
docker network inspect ai-infra-network
```

2. 重建网络

```bash
docker compose down
docker network prune
docker compose up -d
```

### DNS 解析失败

1. 检查 DNS 配置

```bash
docker exec backend nslookup postgres
```

2. 使用 IP 地址代替主机名

```bash
docker inspect postgres | grep IPAddress
```

## 数据恢复

### 数据库恢复

```bash
# PostgreSQL
docker exec -i ai-infra-postgres psql -U postgres ai-infra-matrix < backup.sql

# MySQL
docker exec -i ai-infra-mysql mysql -u root -p slurm_acct_db < slurm_backup.sql
```

### 文件恢复

```bash
# 恢复 Gitea 数据
docker cp backup/gitea/ ai-infra-gitea:/data/

# 恢复 JupyterHub 配置
docker cp backup/jupyterhub/ ai-infra-jupyterhub:/srv/jupyterhub/
```

## 获取帮助

### 收集诊断信息

```bash
# 系统信息
uname -a
docker version
docker compose version

# 服务状态
docker compose ps
docker compose logs > diagnostic-logs.txt

# 资源使用
docker stats --no-stream > resource-usage.txt
df -h > disk-usage.txt
free -h > memory-usage.txt
```

### 联系支持

- 📧 Email: <support@example.com>
- 🐛 GitHub Issues: <https://github.com/aresnasa/ai-infra-matrix/issues>
- 📚 Documentation: [docs/](.)

### 社区资源

- [GitHub Discussions](https://github.com/aresnasa/ai-infra-matrix/discussions)
- [Stack Overflow](https://stackoverflow.com/questions/tagged/ai-infra-matrix)

## 相关文档

- [系统架构](PROJECT_STRUCTURE.md)
- [部署指南](QUICK_START.md)
- [监控指南](MONITORING.md)
- [备份恢复](BACKUP_RECOVERY.md)
