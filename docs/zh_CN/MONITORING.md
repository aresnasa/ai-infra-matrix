# 系统监控指南

**中文** | **[English](en/MONITORING.md)**

## 概述

AI Infrastructure Matrix 集成了 Nightingale 监控系统，提供全栈监控和告警功能。

## 访问监控面板

- **URL**: <http://localhost:8080/n9e>
- **默认账号**: admin / admin123

## 监控架构

```
┌──────────────────────────────────────┐
│      Nightingale Frontend            │
│   (监控仪表盘和告警管理界面)           │
└───────────────┬──────────────────────┘
                │
┌───────────────▼──────────────────────┐
│      Nightingale Server              │
│   (数据聚合、告警规则引擎)             │
└───────┬──────────────┬───────────────┘
        │              │
┌───────▼──────┐  ┌───▼───────────────┐
│  Prometheus  │  │  Categraf Agents  │
│   (时序数据)  │  │  (指标采集)        │
└──────────────┘  └───┬───────────────┘
                      │
        ┌─────────────┼─────────────┐
        │             │             │
    ┌───▼───┐    ┌───▼───┐    ┌───▼───┐
    │ Node1 │    │ Node2 │    │ Node3 │
    └───────┘    └───────┘    └───────┘
```

## 核心功能

### 1. 仪表盘 (Dashboard)

#### 系统概览仪表盘

显示整体系统状态：
- CPU 使用率
- 内存使用率
- 磁盘使用率
- 网络流量
- 服务运行状态

#### Slurm 集群仪表盘

显示 Slurm 集群状态：
- 节点状态统计
- 队列作业数量
- 资源利用率
- 作业成功/失败率

#### 数据库仪表盘

监控数据库性能：
- 连接数
- 查询 QPS
- 慢查询统计
- 缓存命中率

### 2. 监控指标

#### 主机指标

```promql
# CPU 使用率
100 - (avg by(instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)

# 内存使用率
100 * (1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes))

# 磁盘使用率
100 - ((node_filesystem_avail_bytes{mountpoint="/"} * 100) / node_filesystem_size_bytes{mountpoint="/"})

# 网络流量
rate(node_network_receive_bytes_total[5m])
rate(node_network_transmit_bytes_total[5m])
```

#### 容器指标

```promql
# 容器 CPU 使用率
rate(container_cpu_usage_seconds_total{container!=""}[5m])

# 容器内存使用
container_memory_usage_bytes{container!=""}

# 容器网络流量
rate(container_network_receive_bytes_total[5m])
rate(container_network_transmit_bytes_total[5m])
```

#### 应用指标

```promql
# HTTP 请求速率
rate(http_requests_total[5m])

# HTTP 请求延迟
histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))

# 错误率
rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m])
```

### 3. 告警规则

#### CPU 告警

```yaml
name: "CPU 使用率过高"
promql: "100 - (avg by(instance) (rate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100) > 80"
duration: 5m
severity: warning
notify_channels:
  - webhook
  - email
```

#### 内存告警

```yaml
name: "内存使用率过高"
promql: "100 * (1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) > 85"
duration: 5m
severity: warning
```

#### 磁盘告警

```yaml
name: "磁盘空间不足"
promql: "100 - ((node_filesystem_avail_bytes{mountpoint=\"/\"} * 100) / node_filesystem_size_bytes{mountpoint=\"/\"}) > 90"
duration: 10m
severity: critical
```

#### 服务告警

```yaml
name: "服务不可用"
promql: "up{job=\"backend\"} == 0"
duration: 1m
severity: critical
notify_channels:
  - webhook
  - sms
  - phone
```

### 4. 通知渠道

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

#### 邮件通知

配置 SMTP 服务器：

```yaml
smtp:
  host: smtp.example.com
  port: 587
  username: alerts@example.com
  password: your-password
  from: AI Infra Alerts <alerts@example.com>
```

#### 钉钉机器人

```json
{
  "webhook": "https://oapi.dingtalk.com/robot/send?access_token=xxx",
  "secret": "your-secret"
}
```

## 配置 Categraf 采集器

Categraf 是轻量级的指标采集器，已集成在 AppHub 中。

### 基础配置

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

### 采集插件配置

#### 系统指标

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

#### Docker 容器

```toml
# /opt/categraf/conf/input.docker/docker.toml
[[instances]]
endpoint = "unix:///var/run/docker.sock"
gather_services = false
```

#### MySQL 监控

```toml
# /opt/categraf/conf/input.mysql/mysql.toml
[[instances]]
address = "mysql:3306"
username = "root"
password = "your-password"
```

#### PostgreSQL 监控

```toml
# /opt/categraf/conf/input.postgresql/postgresql.toml
[[instances]]
address = "postgres://postgres:password@postgres:5432/ai-infra-matrix?sslmode=disable"
```

## 创建自定义仪表盘

### 1. 登录 Nightingale

访问 <http://localhost:8080/n9e>

### 2. 创建仪表盘

1. 点击 "仪表盘" -> "新建仪表盘"
2. 输入仪表盘名称和描述
3. 选择标签分类

### 3. 添加图表

#### 时序图表

```json
{
  "type": "timeseries",
  "title": "CPU 使用率",
  "query": "100 - (avg by(instance) (rate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100)",
  "legend": "{{instance}}",
  "unit": "percent",
  "min": 0,
  "max": 100
}
```

#### 单值图表

```json
{
  "type": "stat",
  "title": "在线节点数",
  "query": "count(up{job=\"node-exporter\"} == 1)",
  "unit": "short",
  "colorMode": "value"
}
```

#### 表格图表

```json
{
  "type": "table",
  "title": "节点列表",
  "queries": [
    "up{job=\"node-exporter\"}",
    "100 - (avg by(instance) (rate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100)"
  ],
  "columns": ["instance", "status", "cpu_usage"]
}
```

## 日志监控

### 查看服务日志

```bash
# Docker Compose 环境
docker compose logs -f backend
docker compose logs -f jupyterhub
docker compose logs -f slurm-master

# Kubernetes 环境
kubectl logs -f deployment/backend -n ai-infra
kubectl logs -f deployment/jupyterhub -n ai-infra
```

### 配置日志采集

使用 Promtail + Loki 收集日志：

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

## 性能分析

### 使用 Grafana 深度分析

```bash
# 安装 Grafana
docker run -d \
  -p 3000:3000 \
  -e GF_SECURITY_ADMIN_PASSWORD=admin \
  --name grafana \
  grafana/grafana

# 添加 Prometheus 数据源
# URL: http://nightingale:9090
```

### 导入 Dashboard

从 Grafana 官方导入常用 Dashboard：
- Node Exporter Full (ID: 1860)
- Docker Dashboard (ID: 893)
- MySQL Overview (ID: 7362)

## 告警测试

### 手动触发告警

```bash
# 模拟 CPU 高负载
stress-ng --cpu 4 --timeout 300s

# 模拟内存压力
stress-ng --vm 2 --vm-bytes 2G --timeout 300s

# 模拟磁盘写入
dd if=/dev/zero of=/tmp/test.img bs=1M count=10000
```

### 验证告警

1. 检查 Nightingale 告警列表
2. 确认通知渠道收到消息
3. 查看告警历史记录

## 最佳实践

### 1. 监控指标设计

- 使用 RED 方法（Rate, Errors, Duration）
- 设置合理的采集间隔（15-60秒）
- 避免高基数标签（如 user_id）

### 2. 告警规则设计

- 设置合理的阈值
- 避免告警风暴（使用抑制规则）
- 区分不同严重级别
- 配置告警静默时间

### 3. 仪表盘设计

- 按业务模块组织
- 使用模板变量提高复用性
- 添加必要的说明文档
- 定期审查和优化

### 4. 数据保留策略

```yaml
# Prometheus 数据保留
retention.time: 15d
retention.size: 50GB

# 降采样配置
- source: 15s
  retention: 7d
- source: 1m
  retention: 30d
- source: 5m
  retention: 180d
```

## 故障排查

### 指标缺失

1. 检查 Categraf 运行状态
2. 验证网络连接
3. 查看采集器日志

### 告警不触发

1. 验证 PromQL 语法
2. 检查告警规则配置
3. 确认通知渠道正常

### 性能问题

1. 优化查询语句
2. 增加采集间隔
3. 使用录制规则预计算

## 参考资源

- [Nightingale 官方文档](https://n9e.github.io/)
- [Prometheus 查询语法](https://prometheus.io/docs/prometheus/latest/querying/basics/)
- [Categraf 配置指南](https://github.com/flashcatcloud/categraf)
