```markdown
# System Monitoring Guide

## Overview

AI Infrastructure Matrix integrates the Nightingale monitoring system, providing full-stack monitoring and alerting capabilities.

## Accessing the Monitoring Dashboard

- **URL**: <http://localhost:8080/n9e>
- **Default Account**: admin / admin123

## Monitoring Architecture

```
┌──────────────────────────────────────┐
│      Nightingale Frontend            │
│   (Monitoring Dashboard and          │
│    Alert Management Interface)       │
└───────────────┬──────────────────────┘
                │
┌───────────────▼──────────────────────┐
│      Nightingale Server              │
│   (Data Aggregation, Alert           │
│    Rules Engine)                     │
└───────┬──────────────┬───────────────┘
        │              │
┌───────▼──────┐  ┌───▼───────────────┐
│  Prometheus  │  │  Categraf Agents  │
│ (Time Series │  │  (Metrics         │
│     Data)    │  │   Collection)     │
└──────────────┘  └───┬───────────────┘
                      │
        ┌─────────────┼─────────────┐
        │             │             │
    ┌───▼───┐    ┌───▼───┐    ┌───▼───┐
    │ Node1 │    │ Node2 │    │ Node3 │
    └───────┘    └───────┘    └───────┘
```

## Core Features

### 1. Dashboard

#### System Overview Dashboard

Displays overall system status:
- CPU usage
- Memory usage
- Disk usage
- Network traffic
- Service running status

#### Slurm Cluster Dashboard

Displays Slurm cluster status:
- Node status statistics
- Queue job counts
- Resource utilization
- Job success/failure rate

#### Database Dashboard

Monitors database performance:
- Connection count
- Query QPS
- Slow query statistics
- Cache hit rate

### 2. Monitoring Metrics

#### Host Metrics

```promql
# CPU usage
100 - (avg by(instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)

# Memory usage
100 * (1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes))

# Disk usage
100 - ((node_filesystem_avail_bytes{mountpoint="/"} * 100) / node_filesystem_size_bytes{mountpoint="/"})

# Network traffic
rate(node_network_receive_bytes_total[5m])
rate(node_network_transmit_bytes_total[5m])
```

#### Container Metrics

```promql
# Container CPU usage
rate(container_cpu_usage_seconds_total{container!=""}[5m])

# Container memory usage
container_memory_usage_bytes{container!=""}

# Container network traffic
rate(container_network_receive_bytes_total[5m])
rate(container_network_transmit_bytes_total[5m])
```

#### Application Metrics

```promql
# HTTP request rate
rate(http_requests_total[5m])

# HTTP request latency
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# Error rate
rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m])
```

### 3. Alert Rules

#### CPU Alert

```yaml
name: "High CPU Usage"
promql: "100 - (avg by(instance) (rate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100) > 80"
duration: 5m
severity: warning
notify_channels:
  - webhook
  - email
```

#### Memory Alert

```yaml
name: "High Memory Usage"
promql: "100 * (1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) > 85"
duration: 5m
severity: warning
```

#### Disk Alert

```yaml
name: "Low Disk Space"
promql: "100 - ((node_filesystem_avail_bytes{mountpoint=\"/\"} * 100) / node_filesystem_size_bytes{mountpoint=\"/\"}) > 90"
duration: 10m
severity: critical
```

#### Service Alert

```yaml
name: "Service Unavailable"
promql: "up{job=\"backend\"} == 0"
duration: 1m
severity: critical
notify_channels:
  - webhook
  - sms
  - phone
```

### 4. Notification Channels

#### Webhook

```json
{
  "url": "https://your-webhook.example.com/alerts",
  "method": "POST",
  "headers": {
    "Authorization": "Bearer your-token"
  }
}
```

#### Email Notification

Configure SMTP server:

```yaml
smtp:
  host: smtp.example.com
  port: 587
  username: alerts@example.com
  password: your-password
  from: AI Infra Alerts <alerts@example.com>
```

#### DingTalk Robot

```json
{
  "webhook": "https://oapi.dingtalk.com/robot/send?access_token=xxx",
  "secret": "your-secret"
}
```

## Configuring Categraf Collector

Categraf is a lightweight metrics collector, already integrated in AppHub.

### Basic Configuration

```toml
# /opt/categraf/conf/config.toml
[global]
hostname = ""
interval = 15
providers = ["local"]

[writer_opt]
batch = 1000
chan_size = 10000

[[writers]]
url = "http://nightingale:19000/prometheus/v1/write"
timeout = 5000
```

### Collection Plugin Configuration

#### System Metrics

```toml
# /opt/categraf/conf/input.cpu/cpu.toml
[[instances]]
collect_per_cpu = false
report_active = true

# /opt/categraf/conf/input.mem/mem.toml
[[instances]]
collect_platform_fields = true

# /opt/categraf/conf/input.disk/disk.toml
[[instances]]
ignore_fs = ["tmpfs", "devtmpfs"]
```

#### Docker Containers

```toml
# /opt/categraf/conf/input.docker/docker.toml
[[instances]]
endpoint = "unix:///var/run/docker.sock"
gather_services = false
```

#### MySQL Monitoring

```toml
# /opt/categraf/conf/input.mysql/mysql.toml
[[instances]]
address = "mysql:3306"
username = "root"
password = "your-password"
```

#### PostgreSQL Monitoring

```toml
# /opt/categraf/conf/input.postgresql/postgresql.toml
[[instances]]
address = "postgres://postgres:password@postgres:5432/ai-infra-matrix?sslmode=disable"
```

## Creating Custom Dashboards

### 1. Log in to Nightingale

Visit <http://localhost:8080/n9e>

### 2. Create Dashboard

1. Click "Dashboard" -> "New Dashboard"
2. Enter dashboard name and description
3. Select tag category

### 3. Add Charts

#### Time Series Chart

```json
{
  "type": "timeseries",
  "title": "CPU Usage",
  "query": "100 - (avg by(instance) (rate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100)",
  "legend": "{{instance}}",
  "unit": "percent",
  "min": 0,
  "max": 100
}
```

#### Single Value Chart

```json
{
  "type": "stat",
  "title": "Online Node Count",
  "query": "count(up{job=\"node-exporter\"} == 1)",
  "unit": "short",
  "colorMode": "value"
}
```

#### Table Chart

```json
{
  "type": "table",
  "title": "Node List",
  "queries": [
    "up{job=\"node-exporter\"}",
    "100 - (avg by(instance) (rate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100)"
  ],
  "columns": ["instance", "status", "cpu_usage"]
}
```

## Log Monitoring

### Viewing Service Logs

```bash
# Docker Compose environment
docker compose logs -f backend
docker compose logs -f jupyterhub
docker compose logs -f slurm-master

# Kubernetes environment
kubectl logs -f deployment/backend -n ai-infra
kubectl logs -f deployment/jupyterhub -n ai-infra
```

### Configuring Log Collection

Use Promtail + Loki to collect logs:

```yaml
# promtail-config.yaml
clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  - job_name: containers
    docker_sd_configs:
      - host: unix:///var/run/docker.sock
    relabel_configs:
      - source_labels: ['__meta_docker_container_name']
        target_label: 'container'
```

## Performance Analysis

### Deep Analysis with Grafana

```bash
# Install Grafana
docker run -d \
  -p 3000:3000 \
  -e GF_SECURITY_ADMIN_PASSWORD=admin \
  --name grafana \
  grafana/grafana

# Add Prometheus data source
# URL: http://nightingale:9090
```

### Import Dashboards

Import commonly used dashboards from Grafana official:
- Node Exporter Full (ID: 1860)
- Docker Dashboard (ID: 893)
- MySQL Overview (ID: 7362)

## Alert Testing

### Manually Trigger Alerts

```bash
# Simulate high CPU load
stress-ng --cpu 4 --timeout 300s

# Simulate memory pressure
stress-ng --vm 2 --vm-bytes 2G --timeout 300s

# Simulate disk writes
dd if=/dev/zero of=/tmp/test.img bs=1M count=10000
```

### Verify Alerts

1. Check Nightingale alert list
2. Confirm notification channels received messages
3. View alert history

## Best Practices

### 1. Monitoring Metrics Design

- Use the RED method (Rate, Errors, Duration)
- Set reasonable collection intervals (15-60 seconds)
- Avoid high cardinality labels (such as user_id)

### 2. Alert Rule Design

- Set reasonable thresholds
- Avoid alert storms (use suppression rules)
- Distinguish different severity levels
- Configure alert silence periods

### 3. Dashboard Design

- Organize by business modules
- Use template variables to improve reusability
- Add necessary documentation
- Regularly review and optimize

### 4. Data Retention Policy

```yaml
# Prometheus data retention
retention.time: 15d
retention.size: 50GB

# Downsampling configuration
- source: 15s
  retention: 7d
- source: 1m
  retention: 30d
- source: 5m
  retention: 180d
```

## Troubleshooting

### Missing Metrics

1. Check Categraf running status
2. Verify network connectivity
3. View collector logs

### Alerts Not Triggering

1. Verify PromQL syntax
2. Check alert rule configuration
3. Confirm notification channels are working

### Performance Issues

1. Optimize query statements
2. Increase collection intervals
3. Use recording rules for pre-computation

## Reference Resources

- [Nightingale Official Documentation](https://n9e.github.io/)
- [Prometheus Query Syntax](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [Categraf Configuration Guide](https://github.com/flashcatcloud/categraf)

```
