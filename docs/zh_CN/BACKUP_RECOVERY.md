# 备份与恢复指南

**中文** | **[English](en/BACKUP_RECOVERY.md)**

## 概述

本指南介绍 AI Infrastructure Matrix 的数据备份和恢复策略。

## 备份策略

### 备份内容

需要备份的关键数据：

1. **数据库**
   - PostgreSQL（应用数据）
   - MySQL（Slurm 作业数据）
   - OceanBase（可选）

2. **对象存储**
   - MinIO 数据（Gitea LFS、用户文件）

3. **配置文件**
   - 环境配置（.env）
   - 服务配置（docker-compose.yml）

4. **应用数据**
   - Gitea 仓库数据
   - JupyterHub 用户数据
   - Slurm 配置和日志

### 备份频率

| 数据类型 | 备份频率 | 保留时间 |
|---------|---------|---------|
| 数据库 | 每天 | 30天 |
| 对象存储 | 每周 | 90天 |
| 配置文件 | 每次变更 | 永久 |
| 完整备份 | 每周 | 4周 |

## 数据库备份

### PostgreSQL 备份

#### 手动备份

```bash
# 导出所有数据库
docker exec ai-infra-postgres pg_dumpall -U postgres > backup/postgres-all-$(date +%Y%m%d).sql

# 导出单个数据库
docker exec ai-infra-postgres pg_dump -U postgres ai-infra-matrix > backup/postgres-aiinfra-$(date +%Y%m%d).sql

# 压缩备份
docker exec ai-infra-postgres pg_dump -U postgres -F c ai-infra-matrix > backup/postgres-$(date +%Y%m%d).dump
```

#### 自动备份脚本

```bash
#!/bin/bash
# backup-postgres.sh

BACKUP_DIR="/data/backups/postgres"
RETENTION_DAYS=30
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p $BACKUP_DIR

# 备份
docker exec ai-infra-postgres pg_dump -U postgres \
  -F c ai-infra-matrix > $BACKUP_DIR/postgres-$DATE.dump

# 压缩
gzip $BACKUP_DIR/postgres-$DATE.dump

# 清理旧备份
find $BACKUP_DIR -name "postgres-*.dump.gz" -mtime +$RETENTION_DAYS -delete

echo "Backup completed: postgres-$DATE.dump.gz"
```

#### 定时任务

```bash
# 添加到 crontab
crontab -e

# 每天凌晨 2 点备份
0 2 * * * /path/to/backup-postgres.sh
```

### MySQL 备份

```bash
# 备份 Slurm 数据库
docker exec ai-infra-mysql mysqldump -u root -p${MYSQL_ROOT_PASSWORD} \
  slurm_acct_db > backup/mysql-slurm-$(date +%Y%m%d).sql

# 备份所有数据库
docker exec ai-infra-mysql mysqldump -u root -p${MYSQL_ROOT_PASSWORD} \
  --all-databases > backup/mysql-all-$(date +%Y%m%d).sql
```

### Redis 备份

```bash
# 触发 RDB 快照
docker exec ai-infra-redis redis-cli BGSAVE

# 复制 RDB 文件
docker cp ai-infra-redis:/data/dump.rdb backup/redis-$(date +%Y%m%d).rdb
```

## 对象存储备份

### MinIO 备份

#### 使用 mc 客户端

```bash
# 安装 mc 客户端
docker run -it --rm --entrypoint=/bin/sh minio/mc

# 配置别名
mc alias set local http://minio:9000 minioadmin minioadmin

# 镜像备份
mc mirror local/gitea /backup/minio/gitea

# 增量备份
mc mirror --watch local/gitea /backup/minio/gitea
```

#### 数据卷备份

```bash
# 停止服务
docker compose stop minio

# 备份数据卷
docker run --rm \
  -v ai-infra-matrix_minio_data:/data \
  -v $(pwd)/backup:/backup \
  alpine tar czf /backup/minio-data-$(date +%Y%m%d).tar.gz -C /data .

# 启动服务
docker compose start minio
```

## 应用数据备份

### Gitea 备份

```bash
# 创建 Gitea 备份
docker exec -u git ai-infra-gitea /app/gitea/gitea dump \
  -c /data/gitea/conf/app.ini \
  -f /data/gitea-backup-$(date +%Y%m%d).zip

# 复制到主机
docker cp ai-infra-gitea:/data/gitea-backup-*.zip backup/
```

### JupyterHub 备份

```bash
# 备份用户数据
docker run --rm \
  -v ai-infra-matrix_jupyterhub_data:/data \
  -v $(pwd)/backup:/backup \
  alpine tar czf /backup/jupyterhub-$(date +%Y%m%d).tar.gz -C /data .

# 备份配置
docker cp ai-infra-jupyterhub:/srv/jupyterhub/jupyterhub_config.py \
  backup/jupyterhub_config-$(date +%Y%m%d).py
```

### Slurm 配置备份

```bash
# 备份 Slurm 配置
docker cp ai-infra-slurm-master:/etc/slurm/ backup/slurm-config-$(date +%Y%m%d)/

# 备份作业历史
docker exec ai-infra-slurm-master sacct -a --format=ALL > backup/slurm-jobs-$(date +%Y%m%d).txt
```

## 完整系统备份

### 备份脚本

```bash
#!/bin/bash
# backup-all.sh

BACKUP_ROOT="/data/backups"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="$BACKUP_ROOT/full-$DATE"

mkdir -p $BACKUP_DIR

echo "Starting full backup at $(date)"

# 1. 停止服务（可选，确保数据一致性）
# docker compose stop

# 2. 备份数据库
echo "Backing up databases..."
docker exec ai-infra-postgres pg_dump -U postgres -F c ai-infra-matrix > $BACKUP_DIR/postgres.dump
docker exec ai-infra-mysql mysqldump -u root -p${MYSQL_ROOT_PASSWORD} --all-databases > $BACKUP_DIR/mysql.sql

# 3. 备份配置文件
echo "Backing up configurations..."
cp .env $BACKUP_DIR/
cp docker-compose.yml $BACKUP_DIR/
cp -r config/ $BACKUP_DIR/config/

# 4. 备份数据卷
echo "Backing up volumes..."
docker run --rm \
  -v ai-infra-matrix_postgres_data:/data \
  -v $BACKUP_DIR:/backup \
  alpine tar czf /backup/postgres-volume.tar.gz -C /data .

docker run --rm \
  -v ai-infra-matrix_minio_data:/data \
  -v $BACKUP_DIR:/backup \
  alpine tar czf /backup/minio-volume.tar.gz -C /data .

# 5. 备份应用数据
echo "Backing up application data..."
docker exec -u git ai-infra-gitea /app/gitea/gitea dump \
  -c /data/gitea/conf/app.ini -f /data/gitea-backup.zip
docker cp ai-infra-gitea:/data/gitea-backup.zip $BACKUP_DIR/

# 6. 重启服务
# docker compose start

# 7. 压缩备份
echo "Compressing backup..."
tar czf $BACKUP_ROOT/full-backup-$DATE.tar.gz -C $BACKUP_ROOT full-$DATE
rm -rf $BACKUP_DIR

echo "Backup completed: full-backup-$DATE.tar.gz"
```

## 数据恢复

### PostgreSQL 恢复

```bash
# 恢复自定义格式备份
docker exec -i ai-infra-postgres pg_restore -U postgres \
  -d ai-infra-matrix -c < backup/postgres.dump

# 恢复 SQL 文件
docker exec -i ai-infra-postgres psql -U postgres \
  ai-infra-matrix < backup/postgres.sql

# 恢复所有数据库
docker exec -i ai-infra-postgres psql -U postgres < backup/postgres-all.sql
```

### MySQL 恢复

```bash
# 恢复单个数据库
docker exec -i ai-infra-mysql mysql -u root -p${MYSQL_ROOT_PASSWORD} \
  slurm_acct_db < backup/mysql-slurm.sql

# 恢复所有数据库
docker exec -i ai-infra-mysql mysql -u root -p${MYSQL_ROOT_PASSWORD} \
  < backup/mysql-all.sql
```

### 数据卷恢复

```bash
# 1. 停止服务
docker compose down

# 2. 删除旧数据卷（谨慎操作！）
docker volume rm ai-infra-matrix_postgres_data

# 3. 创建新数据卷
docker volume create ai-infra-matrix_postgres_data

# 4. 恢复数据
docker run --rm \
  -v ai-infra-matrix_postgres_data:/data \
  -v $(pwd)/backup:/backup \
  alpine sh -c "cd /data && tar xzf /backup/postgres-volume.tar.gz"

# 5. 启动服务
docker compose up -d
```

### Gitea 恢复

```bash
# 1. 复制备份文件到容器
docker cp backup/gitea-backup.zip ai-infra-gitea:/tmp/

# 2. 恢复
docker exec -u git ai-infra-gitea /app/gitea/gitea restore \
  --config /data/gitea/conf/app.ini \
  --from /tmp/gitea-backup.zip

# 3. 重启服务
docker compose restart gitea
```

## 灾难恢复

### 完整系统恢复

```bash
# 1. 解压备份
tar xzf backup/full-backup-20251118.tar.gz -C /tmp/

# 2. 恢复配置文件
cp /tmp/full-20251118/.env .
cp /tmp/full-20251118/docker-compose.yml .

# 3. 创建数据卷
docker volume create ai-infra-matrix_postgres_data
docker volume create ai-infra-matrix_mysql_data
docker volume create ai-infra-matrix_minio_data

# 4. 恢复数据卷
docker run --rm \
  -v ai-infra-matrix_postgres_data:/data \
  -v /tmp/full-20251118:/backup \
  alpine tar xzf /backup/postgres-volume.tar.gz -C /data

# 重复其他数据卷...

# 5. 启动服务
docker compose up -d

# 6. 恢复数据库
docker exec -i ai-infra-postgres pg_restore -U postgres \
  -d ai-infra-matrix -c < /tmp/full-20251118/postgres.dump
```

## 备份验证

### 定期测试恢复

```bash
# 1. 创建测试环境
docker compose -f docker-compose.test.yml up -d

# 2. 恢复备份到测试环境
# ... 执行恢复步骤 ...

# 3. 验证数据完整性
docker exec test-postgres psql -U postgres -d ai-infra-matrix -c "SELECT COUNT(*) FROM users;"

# 4. 清理测试环境
docker compose -f docker-compose.test.yml down -v
```

## 自动化备份

### 使用 Cron

```bash
# 编辑 crontab
crontab -e

# 添加任务
# 每天 2:00 数据库备份
0 2 * * * /path/to/backup-postgres.sh

# 每天 3:00 MySQL 备份
0 3 * * * /path/to/backup-mysql.sh

# 每周日 1:00 完整备份
0 1 * * 0 /path/to/backup-all.sh
```

### 使用 Kubernetes CronJob

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: postgres-backup
spec:
  schedule: "0 2 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: postgres:15-alpine
            command:
            - /bin/sh
            - -c
            - pg_dump -h postgres -U postgres ai-infra-matrix > /backup/postgres-$(date +%Y%m%d).sql
            volumeMounts:
            - name: backup
              mountPath: /backup
          volumes:
          - name: backup
            persistentVolumeClaim:
              claimName: backup-pvc
          restartPolicy: OnFailure
```

## 异地备份

### 同步到云存储

```bash
# AWS S3
aws s3 sync /data/backups/ s3://my-backup-bucket/ai-infra-matrix/

# 阿里云 OSS
ossutil cp -r /data/backups/ oss://my-backup-bucket/ai-infra-matrix/

# 使用 rclone
rclone sync /data/backups/ remote:backup-bucket/ai-infra-matrix/
```

### 远程服务器备份

```bash
# 使用 rsync
rsync -avz --delete /data/backups/ backup-server:/backups/ai-infra-matrix/

# 使用 scp
scp -r /data/backups/full-backup-*.tar.gz backup-server:/backups/
```

## 最佳实践

1. **3-2-1 备份策略**
   - 3 份数据副本
   - 2 种不同存储介质
   - 1 份异地备份

2. **定期测试恢复**
   - 每月测试一次完整恢复
   - 验证备份完整性
   - 文档化恢复流程

3. **监控备份状态**
   - 设置备份失败告警
   - 检查备份文件大小
   - 验证备份时间戳

4. **安全保护**
   - 加密备份文件
   - 限制备份访问权限
   - 定期审计备份日志

## 相关文档

- [部署指南](QUICK_START.md)
- [故障排除](TROUBLESHOOTING.md)
- [监控指南](MONITORING.md)
