```markdown
# Troubleshooting Guide

## Common Issues

### Services Cannot Start

#### Symptoms

```bash
$ docker compose up -d
Error response from daemon: container not found
```

#### Solution

1. Check Docker service status

```bash
docker info
```

2. Clean up old containers

```bash
docker compose down
docker system prune -a
```

3. Rebuild images

```bash
./build.sh build-all v0.3.8
docker compose up -d
```

### Port Conflict

#### Symptoms

```
Error starting userland proxy: listen tcp 0.0.0.0:8080: bind: address already in use
```

#### Solution

1. Find the process occupying the port

```bash
lsof -i :8080
# or
netstat -tulpn | grep 8080
```

2. Terminate the occupying process or modify the port

```bash
# Modify .env file
EXTERNAL_PORT=8081
```

### Database Connection Failure

#### Symptoms

```
Error: connection refused to postgres:5432
```

#### Solution

1. Check database container status

```bash
docker compose ps postgres
docker compose logs postgres
```

2. Verify connection configuration

```bash
# Test database connection
docker exec -it ai-infra-postgres psql -U postgres -d ai-infra-matrix
```

3. Restart database service

```bash
docker compose restart postgres
```

### Slurm Node DOWN Status

#### Symptoms

Nodes showing as DOWN or UNKNOWN status

#### Solution

1. Check node connection

```bash
# In slurm-master container
docker exec ai-infra-slurm-master sinfo
docker exec ai-infra-slurm-master scontrol show node node01
```

2. Restart slurmd service

```bash
# On compute node
systemctl restart slurmd
```

3. Manually recover node

```bash
# In slurm-master container
scontrol update NodeName=node01 State=RESUME
```

Reference: [Slurm Node Recovery Guide](SLURM_NODE_RECOVERY_GUIDE.md)

### JupyterHub Cannot Start

#### Symptoms

JupyterHub user server fails to start

#### Solution

1. Check JupyterHub logs

```bash
docker compose logs jupyterhub
```

2. Verify image availability

```bash
docker images | grep singleuser
```

3. Clean up old user containers

```bash
docker ps -a | grep jupyter
docker rm -f $(docker ps -a | grep jupyter | awk '{print $1}')
```

### Gitea LFS Upload Failure

#### Symptoms

```
Error: LFS upload failed
```

#### Solution

1. Check MinIO status

```bash
docker compose ps minio
docker compose logs minio
```

2. Verify S3 configuration

```bash
# Check Gitea configuration
docker exec ai-infra-gitea cat /data/gitea/conf/app.ini | grep -A 5 "\[lfs\]"
```

3. Test MinIO connection

```bash
docker exec ai-infra-gitea wget -O- http://minio:9000/minio/health/live
```

### Frontend White Screen

#### Symptoms

Browser displays white screen or 404

#### Solution

1. Clear browser cache

```bash
# Chrome: Ctrl+Shift+Delete
# Firefox: Ctrl+Shift+Del
```

2. Check Nginx configuration

```bash
docker exec ai-infra-nginx nginx -t
docker compose restart nginx
```

3. Rebuild frontend image

```bash
./build.sh build src/frontend v0.3.8
docker compose up -d frontend
```

### Missing Monitoring Data

#### Symptoms

Nightingale monitoring dashboard shows no data

#### Solution

1. Check Categraf status

```bash
docker compose logs categraf
```

2. Verify Prometheus connection

```bash
curl http://localhost:9090/-/healthy
```

3. Check metric collection

```bash
curl http://localhost:9090/api/v1/query?query=up
```

## Log Viewing

### View All Service Logs

```bash
docker compose logs -f
```

### View Specific Service Logs

```bash
docker compose logs -f backend
docker compose logs -f frontend
docker compose logs -f jupyterhub
docker compose logs -f slurm-master
```

### View Last N Lines of Logs

```bash
docker compose logs --tail=100 backend
```

### Export Logs

```bash
docker compose logs > logs/full-logs.txt
docker compose logs backend > logs/backend.log
```

## Performance Issues

### High CPU Usage

1. View resource usage

```bash
docker stats
```

2. Limit container resources

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

### Insufficient Memory

1. Check memory usage

```bash
free -h
docker stats --no-stream
```

2. Clean up unused resources

```bash
docker system prune -a
docker volume prune
```

### Insufficient Disk Space

1. Check disk usage

```bash
df -h
du -sh /var/lib/docker/*
```

2. Clean up old images and containers

```bash
docker system prune -a --volumes
```

## Network Issues

### Containers Cannot Communicate

1. Check network configuration

```bash
docker network ls
docker network inspect ai-infra-network
```

2. Rebuild network

```bash
docker compose down
docker network prune
docker compose up -d
```

### DNS Resolution Failure

1. Check DNS configuration

```bash
docker exec backend nslookup postgres
```

2. Use IP address instead of hostname

```bash
docker inspect postgres | grep IPAddress
```

## Data Recovery

### Database Recovery

```bash
# PostgreSQL
docker exec -i ai-infra-postgres psql -U postgres ai-infra-matrix < backup.sql

# MySQL
docker exec -i ai-infra-mysql mysql -u root -p slurm_acct_db < slurm_backup.sql
```

### File Recovery

```bash
# Restore Gitea data
docker cp backup/gitea/ ai-infra-gitea:/data/

# Restore JupyterHub configuration
docker cp backup/jupyterhub/ ai-infra-jupyterhub:/srv/jupyterhub/
```

## Getting Help

### Collect Diagnostic Information

```bash
# System information
uname -a
docker version
docker compose version

# Service status
docker compose ps
docker compose logs > diagnostic-logs.txt

# Resource usage
docker stats --no-stream > resource-usage.txt
df -h > disk-usage.txt
free -h > memory-usage.txt
```

### Contact Support

- 📧 Email: <support@example.com>
- 🐛 GitHub Issues: <https://github.com/aresnasa/ai-infra-matrix/issues>
- 📚 Documentation: [docs/](.)

### Community Resources

- [GitHub Discussions](https://github.com/aresnasa/ai-infra-matrix/discussions)
- [Stack Overflow](https://stackoverflow.com/questions/tagged/ai-infra-matrix)

## Related Documentation

- [System Architecture](PROJECT_STRUCTURE.md)
- [Deployment Guide](QUICK_START.md)
- [Monitoring Guide](MONITORING.md)
- [Backup & Recovery](BACKUP_RECOVERY.md)

```
