# Prometheus VERSION_PLACEHOLDER - Linux ARCH_PLACEHOLDER

Prometheus 是一个开源的监控和告警系统，具有强大的查询语言和多维数据模型。

## 安装

```bash
sudo ./install.sh
```

## 配置

编辑配置文件：
```bash
vim /etc/prometheus/prometheus.yml
```

主要配置项：
- `global.scrape_interval`: 采集间隔
- `scrape_configs`: 采集目标配置
- `rule_files`: 告警规则文件

## 启动

```bash
# 启动服务
systemctl start prometheus

# 查看状态
systemctl status prometheus

# 查看日志
journalctl -u prometheus -f
```

## 访问

- Web UI: http://localhost:9090
- Metrics: http://localhost:9090/metrics

## 常用 PromQL 查询

```promql
# CPU 使用率
100 - (avg by(instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])) * 100)

# 内存使用率
(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100

# 磁盘使用率
(1 - (node_filesystem_avail_bytes / node_filesystem_size_bytes)) * 100
```

## 卸载

```bash
sudo ./uninstall.sh
```

## 更多信息

- 官网: https://prometheus.io/
- 文档: https://prometheus.io/docs/
- GitHub: https://github.com/prometheus/prometheus
