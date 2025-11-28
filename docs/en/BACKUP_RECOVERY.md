```markdown
# Backup and Recovery Guide

## Overview

This guide covers the data backup and recovery strategies for AI Infrastructure Matrix.

## Backup Strategy

### Backup Content

Critical data that needs to be backed up:

1. **Databases**
   - PostgreSQL (application data)
   - MySQL (Slurm job data)
   - OceanBase (optional)

2. **Object Storage**
   - MinIO data (Gitea LFS, user files)

3. **Configuration Files**
   - Environment configuration (.env)
   - Service configuration (docker-compose.yml)

4. **Application Data**
   - Gitea repository data
   - JupyterHub user data
   - Slurm configuration and logs

### Backup Frequency

| Data Type | Backup Frequency | Retention Period |
|-----------|------------------|------------------|
| Databases | Daily | 30 days |
| Object Storage | Weekly | 90 days |
| Configuration Files | On every change | Permanent |
| Full Backup | Weekly | 4 weeks |

## Database Backup

### PostgreSQL Backup

#### Manual Backup

```bash
# Export all databases
docker exec ai-infra-postgres pg_dumpall -U postgres > backup/postgres-all-$(date +%Y%m%d).sql

# Export single database
docker exec ai-infra-postgres pg_dump -U postgres ai-infra-matrix > backup/postgres-aiinfra-$(date +%Y%m%d).sql

# Compressed backup
docker exec ai-infra-postgres pg_dump -U postgres -F c ai-infra-matrix > backup/postgres-$(date +%Y%m%d).dump
```

#### Automated Backup Script

```bash
#!/bin/bash
# backup-postgres.sh

BACKUP_DIR="/data/backups/postgres"
RETENTION_DAYS=30
DATE=$(date +%Y%m%d_%H%M%S)

mkdir -p $BACKUP_DIR

# Backup
docker exec ai-infra-postgres pg_dump -U postgres \
  -F c ai-infra-matrix > $BACKUP_DIR/postgres-$DATE.dump

# Compress
gzip $BACKUP_DIR/postgres-$DATE.dump

# Clean up old backups
find $BACKUP_DIR -name "postgres-*.dump.gz" -mtime +$RETENTION_DAYS -delete

echo "Backup completed: postgres-$DATE.dump.gz"
```

#### Scheduled Task

```bash
# Add to crontab
crontab -e

# Backup at 2 AM daily
0 2 * * * /path/to/backup-postgres.sh
```

### MySQL Backup

```bash
# Backup Slurm database
docker exec ai-infra-mysql mysqldump -u root -p${MYSQL_ROOT_PASSWORD} \
  slurm_acct_db > backup/mysql-slurm-$(date +%Y%m%d).sql

# Backup all databases
docker exec ai-infra-mysql mysqldump -u root -p${MYSQL_ROOT_PASSWORD} \
  --all-databases > backup/mysql-all-$(date +%Y%m%d).sql
```

### Redis Backup

```bash
# Trigger RDB snapshot
docker exec ai-infra-redis redis-cli BGSAVE

# Copy RDB file
docker cp ai-infra-redis:/data/dump.rdb backup/redis-$(date +%Y%m%d).rdb
```

## Object Storage Backup

### MinIO Backup

#### Using mc Client

```bash
# Install mc client
docker run -it --rm --entrypoint=/bin/sh minio/mc

# Configure alias
mc alias set local http://minio:9000 minioadmin minioadmin

# Mirror backup
mc mirror local/gitea /backup/minio/gitea

# Incremental backup
mc mirror --watch local/gitea /backup/minio/gitea
```

#### Data Volume Backup

```bash
# Stop service
docker compose stop minio

# Backup data volume
docker run --rm \
  -v ai-infra-matrix_minio_data:/data \
  -v $(pwd)/backup:/backup \
  alpine tar czf /backup/minio-data-$(date +%Y%m%d).tar.gz -C /data .

# Start service
docker compose start minio
```

## Application Data Backup

### Gitea Backup

```bash
# Create Gitea backup
docker exec -u git ai-infra-gitea /app/gitea/gitea dump \
  -c /data/gitea/conf/app.ini \
  -f /data/gitea-backup-$(date +%Y%m%d).zip

# Copy to host
docker cp ai-infra-gitea:/data/gitea-backup-*.zip backup/
```

### JupyterHub Backup

```bash
# Backup user data
docker run --rm \
  -v ai-infra-matrix_jupyterhub_data:/data \
  -v $(pwd)/backup:/backup \
  alpine tar czf /backup/jupyterhub-$(date +%Y%m%d).tar.gz -C /data .

# Backup configuration
docker cp ai-infra-jupyterhub:/srv/jupyterhub/jupyterhub_config.py \
  backup/jupyterhub_config-$(date +%Y%m%d).py
```

### Slurm Configuration Backup

```bash
# Backup Slurm configuration
docker cp ai-infra-slurm-master:/etc/slurm/ backup/slurm-config-$(date +%Y%m%d)/

# Backup job history
docker exec ai-infra-slurm-master sacct -a --format=ALL > backup/slurm-jobs-$(date +%Y%m%d).txt
```

## Full System Backup

### Backup Script

```bash
#!/bin/bash
# backup-all.sh

BACKUP_ROOT="/data/backups"
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="$BACKUP_ROOT/full-$DATE"

mkdir -p $BACKUP_DIR

echo "Starting full backup at $(date)"

# 1. Stop services (optional, ensures data consistency)
# docker compose stop

# 2. Backup databases
echo "Backing up databases..."
docker exec ai-infra-postgres pg_dump -U postgres -F c ai-infra-matrix > $BACKUP_DIR/postgres.dump
docker exec ai-infra-mysql mysqldump -u root -p${MYSQL_ROOT_PASSWORD} --all-databases > $BACKUP_DIR/mysql.sql

# 3. Backup configuration files
echo "Backing up configurations..."
cp .env $BACKUP_DIR/
cp docker-compose.yml $BACKUP_DIR/
cp -r config/ $BACKUP_DIR/config/

# 4. Backup data volumes
echo "Backing up volumes..."
docker run --rm \
  -v ai-infra-matrix_postgres_data:/data \
  -v $BACKUP_DIR:/backup \
  alpine tar czf /backup/postgres-volume.tar.gz -C /data .

docker run --rm \
  -v ai-infra-matrix_minio_data:/data \
  -v $BACKUP_DIR:/backup \
  alpine tar czf /backup/minio-volume.tar.gz -C /data .

# 5. Backup application data
echo "Backing up application data..."
docker exec -u git ai-infra-gitea /app/gitea/gitea dump \
  -c /data/gitea/conf/app.ini -f /data/gitea-backup.zip
docker cp ai-infra-gitea:/data/gitea-backup.zip $BACKUP_DIR/

# 6. Restart services
# docker compose start

# 7. Compress backup
echo "Compressing backup..."
tar czf $BACKUP_ROOT/full-backup-$DATE.tar.gz -C $BACKUP_ROOT full-$DATE
rm -rf $BACKUP_DIR

echo "Backup completed: full-backup-$DATE.tar.gz"
```

## Data Recovery

### PostgreSQL Recovery

```bash
# Restore custom format backup
docker exec -i ai-infra-postgres pg_restore -U postgres \
  -d ai-infra-matrix -c < backup/postgres.dump

# Restore SQL file
docker exec -i ai-infra-postgres psql -U postgres \
  ai-infra-matrix < backup/postgres.sql

# Restore all databases
docker exec -i ai-infra-postgres psql -U postgres < backup/postgres-all.sql
```

### MySQL Recovery

```bash
# Restore single database
docker exec -i ai-infra-mysql mysql -u root -p${MYSQL_ROOT_PASSWORD} \
  slurm_acct_db < backup/mysql-slurm.sql

# Restore all databases
docker exec -i ai-infra-mysql mysql -u root -p${MYSQL_ROOT_PASSWORD} \
  < backup/mysql-all.sql
```

### Data Volume Recovery

```bash
# 1. Stop services
docker compose down

# 2. Remove old data volume (use caution!)
docker volume rm ai-infra-matrix_postgres_data

# 3. Create new data volume
docker volume create ai-infra-matrix_postgres_data

# 4. Restore data
docker run --rm \
  -v ai-infra-matrix_postgres_data:/data \
  -v $(pwd)/backup:/backup \
  alpine sh -c "cd /data && tar xzf /backup/postgres-volume.tar.gz"

# 5. Start services
docker compose up -d
```

### Gitea Recovery

```bash
# 1. Copy backup file to container
docker cp backup/gitea-backup.zip ai-infra-gitea:/tmp/

# 2. Restore
docker exec -u git ai-infra-gitea /app/gitea/gitea restore \
  --config /data/gitea/conf/app.ini \
  --from /tmp/gitea-backup.zip

# 3. Restart service
docker compose restart gitea
```

## Disaster Recovery

### Full System Recovery

```bash
# 1. Extract backup
tar xzf backup/full-backup-20251118.tar.gz -C /tmp/

# 2. Restore configuration files
cp /tmp/full-20251118/.env .
cp /tmp/full-20251118/docker-compose.yml .

# 3. Create data volumes
docker volume create ai-infra-matrix_postgres_data
docker volume create ai-infra-matrix_mysql_data
docker volume create ai-infra-matrix_minio_data

# 4. Restore data volumes
docker run --rm \
  -v ai-infra-matrix_postgres_data:/data \
  -v /tmp/full-20251118:/backup \
  alpine tar xzf /backup/postgres-volume.tar.gz -C /data

# Repeat for other data volumes...

# 5. Start services
docker compose up -d

# 6. Restore database
docker exec -i ai-infra-postgres pg_restore -U postgres \
  -d ai-infra-matrix -c < /tmp/full-20251118/postgres.dump
```

## Backup Verification

### Regularly Test Recovery

```bash
# 1. Create test environment
docker compose -f docker-compose.test.yml up -d

# 2. Restore backup to test environment
# ... execute recovery steps ...

# 3. Verify data integrity
docker exec test-postgres psql -U postgres -d ai-infra-matrix -c "SELECT COUNT(*) FROM users;"

# 4. Clean up test environment
docker compose -f docker-compose.test.yml down -v
```

## Automated Backup

### Using Cron

```bash
# Edit crontab
crontab -e

# Add tasks
# 2:00 AM database backup daily
0 2 * * * /path/to/backup-postgres.sh

# 3:00 AM MySQL backup daily
0 3 * * * /path/to/backup-mysql.sh

# 1:00 AM full backup every Sunday
0 1 * * 0 /path/to/backup-all.sh
```

### Using Kubernetes CronJob

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

## Offsite Backup

### Sync to Cloud Storage

```bash
# AWS S3
aws s3 sync /data/backups/ s3://my-backup-bucket/ai-infra-matrix/

# Alibaba Cloud OSS
ossutil cp -r /data/backups/ oss://my-backup-bucket/ai-infra-matrix/

# Using rclone
rclone sync /data/backups/ remote:backup-bucket/ai-infra-matrix/
```

### Remote Server Backup

```bash
# Using rsync
rsync -avz --delete /data/backups/ backup-server:/backups/ai-infra-matrix/

# Using scp
scp -r /data/backups/full-backup-*.tar.gz backup-server:/backups/
```

## Best Practices

1. **3-2-1 Backup Strategy**
   - 3 copies of data
   - 2 different storage media
   - 1 offsite backup

2. **Regularly Test Recovery**
   - Test full recovery once a month
   - Verify backup integrity
   - Document recovery procedures

3. **Monitor Backup Status**
   - Set up backup failure alerts
   - Check backup file sizes
   - Verify backup timestamps

4. **Security Protection**
   - Encrypt backup files
   - Restrict backup access permissions
   - Regularly audit backup logs

## Related Documentation

- [Deployment Guide](QUICK_START.md)
- [Troubleshooting](TROUBLESHOOTING.md)
- [Monitoring Guide](MONITORING.md)

```
