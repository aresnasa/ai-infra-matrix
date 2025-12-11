# 监控组件部署 SOP (Standard Operating Procedure)

本文档详细描述 AI Infra Matrix 监控栈的部署流程，包括 Prometheus、Node Exporter、Categraf 以及与 Nightingale/VictoriaMetrics 的集成。

## 目录

- [概述](#概述)
- [架构说明](#架构说明)
- [前置条件](#前置条件)
- [组件版本](#组件版本)
- [部署流程](#部署流程)
  - [1. 验证 AppHub 服务](#1-验证-apphub-服务)
  - [2. 部署 Node Exporter](#2-部署-node-exporter)
  - [3. 部署 Categraf](#3-部署-categraf)
  - [4. 部署 Prometheus (可选)](#4-部署-prometheus-可选)
- [验证监控数据](#验证监控数据)
- [故障排查](#故障排查)
- [附录](#附录)

---

## 概述

AI Infra Matrix 采用以下监控架构：

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   目标主机       │     │   目标主机       │     │   目标主机       │
│  ┌───────────┐  │     │  ┌───────────┐  │     │  ┌───────────┐  │
│  │ Categraf  │──┼─────┼──│ Categraf  │──┼─────┼──│ Categraf  │  │
│  └───────────┘  │     │  └───────────┘  │     │  └───────────┘  │
│  ┌───────────┐  │     │  ┌───────────┐  │     │  ┌───────────┐  │
│  │Node Export│  │     │  │Node Export│  │     │  │Node Export│  │
│  └───────────┘  │     │  └───────────┘  │     │  └───────────┘  │
└────────┬────────┘     └────────┬────────┘     └────────┬────────┘
         │                       │                       │
         │    ┌─────────────────────────────────────┐   │
         └────┤         Nightingale (n9e)           ├───┘
              │   - 指标聚合、告警规则、可视化       │
              └───────────────┬─────────────────────┘
                              │
              ┌───────────────▼───────────────┐
              │      VictoriaMetrics          │
              │   - 时序数据存储 (长期存储)    │
              └───────────────────────────────┘
```

### 监控组件职责

| 组件 | 职责 | 端口 |
|------|------|------|
| **Categraf** | 多功能监控采集代理，支持 push 模式上报到 Nightingale | 9100 (metrics) |
| **Node Exporter** | Prometheus 风格的节点指标导出器，支持 pull 模式 | 9100 (metrics) |
| **Nightingale** | 告警平台、指标聚合、可视化仪表板 | 17000 |
| **VictoriaMetrics** | 高性能时序数据库，兼容 Prometheus | 8428 |
| **Prometheus** | (可选) 传统的监控抓取和存储 | 9090 |

---

## 架构说明

### 数据流向

1. **Push 模式 (推荐)**:
   ```
   Categraf → Nightingale → VictoriaMetrics
   ```

2. **Pull 模式**:
   ```
   Prometheus/Nightingale ← Node Exporter
   ```

### 端口规划

| 服务 | 端口 | 协议 | 说明 |
|------|------|------|------|
| AppHub | 28080 | HTTP | 安装包和脚本下载 |
| Nightingale | 17000 | HTTP | 监控平台 Web UI 和 API |
| VictoriaMetrics | 8428 | HTTP | 时序数据库 HTTP API |
| Categraf | 9100 | HTTP | Metrics endpoint |
| Node Exporter | 9100 | HTTP | Metrics endpoint |
| Prometheus | 9090 | HTTP | Prometheus Web UI (可选) |

---

## 前置条件

### 1. 平台服务运行状态

确保以下核心服务已运行：

```bash
# 检查 Docker 服务状态
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | grep -E "apphub|nightingale|victoriametrics"
```

预期输出：
```
apphub              Up X hours   0.0.0.0:28080->80/tcp
nightingale         Up X hours   0.0.0.0:17000->17000/tcp
victoriametrics     Up X hours   0.0.0.0:8428->8428/tcp
```

### 2. AppHub 包可用性验证

```bash
# 设置 AppHub 地址
export APPHUB_HOST="192.168.0.200"
export APPHUB_PORT="28080"

# 验证 Categraf 包
curl -sI "http://${APPHUB_HOST}:${APPHUB_PORT}/pkgs/categraf/categraf-latest-linux-amd64.tar.gz" | head -5

# 验证 Node Exporter 包
curl -sI "http://${APPHUB_HOST}:${APPHUB_PORT}/pkgs/node_exporter/VERSION" | head -5

# 验证 Prometheus 包
curl -sI "http://${APPHUB_HOST}:${APPHUB_PORT}/pkgs/prometheus/VERSION" | head -5
```

### 3. 网络连通性

```bash
# 从目标主机测试到 AppHub 的连通性
curl -s "http://${APPHUB_HOST}:${APPHUB_PORT}/" | head -5

# 从目标主机测试到 Nightingale 的连通性
curl -s "http://${APPHUB_HOST}:17000/api/n9e/version"
```

---

## 组件版本

当前支持的版本（存储在 AppHub 中）：

| 组件 | 版本 | amd64 | arm64 |
|------|------|-------|-------|
| Categraf | v0.4.25 | ✅ | ✅ |
| Node Exporter | 1.8.2 | ✅ | ✅ |
| Prometheus | 3.4.1 | ✅ | ✅ |

查询最新版本：
```bash
curl -s "http://${APPHUB_HOST}:${APPHUB_PORT}/pkgs/categraf/version.json" | jq .
curl -s "http://${APPHUB_HOST}:${APPHUB_PORT}/pkgs/node_exporter/VERSION"
curl -s "http://${APPHUB_HOST}:${APPHUB_PORT}/pkgs/prometheus/VERSION"
```

---

## 部署流程

### 1. 验证 AppHub 服务

在进行任何安装之前，首先验证 AppHub 服务状态：

```bash
#!/bin/bash
# 验证 AppHub 服务
APPHUB_URL="http://192.168.0.200:28080"

echo "检查 AppHub 服务..."
if curl -sI "${APPHUB_URL}/pkgs/" | grep -q "200 OK"; then
    echo "✓ AppHub 服务正常"
else
    echo "✗ AppHub 服务不可用"
    exit 1
fi

echo ""
echo "可用的安装包："
curl -s "${APPHUB_URL}/pkgs/" | grep -oP 'href="\K[^"]+' | grep -v "^\.\." | head -20
```

### 2. 部署 Node Exporter

Node Exporter 提供标准的 Prometheus 风格指标导出。

#### 方式一：使用渲染后的安装脚本

```bash
# 从 AppHub 下载并执行安装脚本
curl -fsSL http://192.168.0.200:28080/scripts/install-node-exporter.sh | sudo bash
```

#### 方式二：手动安装

```bash
# 设置变量
NODE_EXPORTER_VERSION="1.8.2"
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
APPHUB_URL="http://192.168.0.200:28080"

# 下载
cd /tmp
curl -fsSL -o node_exporter.tar.gz \
    "${APPHUB_URL}/pkgs/node_exporter/node_exporter-${NODE_EXPORTER_VERSION}.linux-${ARCH}.tar.gz"

# 解压并安装
tar xzf node_exporter.tar.gz
sudo cp node_exporter-*/node_exporter /usr/local/bin/
sudo chmod +x /usr/local/bin/node_exporter

# 创建用户
sudo useradd --no-create-home --shell /bin/false node_exporter 2>/dev/null || true

# 创建 systemd 服务
sudo tee /etc/systemd/system/node_exporter.service > /dev/null << 'EOF'
[Unit]
Description=Node Exporter
Wants=network-online.target
After=network-online.target

[Service]
User=node_exporter
Group=node_exporter
Type=simple
ExecStart=/usr/local/bin/node_exporter

[Install]
WantedBy=multi-user.target
EOF

# 启动服务
sudo systemctl daemon-reload
sudo systemctl enable --now node_exporter

# 验证
curl -s localhost:9100/metrics | head -10
```

#### 验证 Node Exporter

```bash
# 检查服务状态
sudo systemctl status node_exporter

# 检查 metrics 端点
curl -s localhost:9100/metrics | grep -E "^node_cpu|^node_memory" | head -10

# 预期输出示例：
# node_cpu_seconds_total{cpu="0",mode="idle"} 123456.78
# node_memory_MemTotal_bytes 1.6777216e+10
```

### 3. 部署 Categraf

Categraf 是推荐的监控代理，支持主动 push 数据到 Nightingale。

#### 方式一：使用渲染后的安装脚本 (推荐)

```bash
# 从 AppHub 下载并执行安装脚本
curl -fsSL http://192.168.0.200:28080/scripts/install-categraf.sh | sudo bash
```

#### 方式二：手动安装

```bash
# 设置变量
CATEGRAF_VERSION="v0.4.25"
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
APPHUB_URL="http://192.168.0.200:28080"
NIGHTINGALE_HOST="192.168.0.200"
NIGHTINGALE_PORT="17000"

# 下载
cd /tmp
curl -fsSL -o categraf.tar.gz \
    "${APPHUB_URL}/pkgs/categraf/categraf-${CATEGRAF_VERSION}-linux-${ARCH}.tar.gz"

# 解压并安装
tar xzf categraf.tar.gz
sudo mkdir -p /opt/categraf
sudo cp -r categraf/* /opt/categraf/ 2>/dev/null || sudo cp -r categraf-*/* /opt/categraf/
sudo chmod +x /opt/categraf/categraf

# 创建用户
sudo useradd --system --no-create-home --shell /bin/false categraf 2>/dev/null || true

# 配置上报地址
sudo tee /opt/categraf/conf/config.toml > /dev/null << EOF
[global]
hostname = ""
omit_hostname = false
interval = 15

[[writers]]
url = "http://${NIGHTINGALE_HOST}:${NIGHTINGALE_PORT}/prometheus/v1/write"
timeout = "5s"
dial_timeout = "2500ms"
max_idle_conns_per_host = 100

[heartbeat]
enable = true
url = "http://${NIGHTINGALE_HOST}:${NIGHTINGALE_PORT}/v1/n9e/heartbeat"
interval = "10s"

[http]
enable = true
address = ":9100"
print_access = false

[log]
file_name = "/var/log/categraf/categraf.log"
max_size = 100
max_age = 1
max_backups = 3
local_time = true
compress = false
level = "info"
EOF

# 创建日志目录
sudo mkdir -p /var/log/categraf
sudo chown -R categraf:categraf /var/log/categraf
sudo chown -R categraf:categraf /opt/categraf

# 创建 systemd 服务
sudo tee /etc/systemd/system/categraf.service > /dev/null << 'EOF'
[Unit]
Description=Categraf Monitoring Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=categraf
Group=categraf
ExecStart=/opt/categraf/categraf
WorkingDirectory=/opt/categraf
Restart=on-failure
RestartSec=5
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

# 启动服务
sudo systemctl daemon-reload
sudo systemctl enable --now categraf

# 验证
curl -s localhost:9100/metrics | head -10
```

#### 验证 Categraf

```bash
# 检查服务状态
sudo systemctl status categraf

# 检查本地 metrics
curl -s localhost:9100/metrics | head -20

# 检查日志
sudo tail -f /var/log/categraf/categraf.log

# 验证数据已写入 VictoriaMetrics
curl -s "http://192.168.0.200:8428/api/v1/status/tsdb" | jq '.data.totalSeries'
```

### 4. 部署 Prometheus (可选)

如果需要独立的 Prometheus 实例：

```bash
# 设置变量
PROMETHEUS_VERSION="3.4.1"
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
APPHUB_URL="http://192.168.0.200:28080"

# 下载
cd /tmp
curl -fsSL -o prometheus.tar.gz \
    "${APPHUB_URL}/pkgs/prometheus/prometheus-${PROMETHEUS_VERSION}.linux-${ARCH}.tar.gz"

# 解压并安装
tar xzf prometheus.tar.gz
sudo mkdir -p /usr/local/prometheus /var/lib/prometheus /etc/prometheus
sudo cp prometheus-*/prometheus prometheus-*/promtool /usr/local/prometheus/
sudo cp -r prometheus-*/consoles prometheus-*/console_libraries /etc/prometheus/

# 创建用户
sudo useradd --system --no-create-home --shell /bin/false prometheus 2>/dev/null || true

# 创建基本配置
sudo tee /etc/prometheus/prometheus.yml > /dev/null << 'EOF'
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']

  - job_name: 'node_exporter'
    static_configs:
      - targets: ['localhost:9100']
EOF

# 设置权限
sudo chown -R prometheus:prometheus /usr/local/prometheus /var/lib/prometheus /etc/prometheus

# 创建 systemd 服务
sudo tee /etc/systemd/system/prometheus.service > /dev/null << 'EOF'
[Unit]
Description=Prometheus
Wants=network-online.target
After=network-online.target

[Service]
User=prometheus
Group=prometheus
Type=simple
ExecStart=/usr/local/prometheus/prometheus \
    --config.file=/etc/prometheus/prometheus.yml \
    --storage.tsdb.path=/var/lib/prometheus \
    --storage.tsdb.retention.time=30d \
    --web.listen-address=0.0.0.0:9090

[Install]
WantedBy=multi-user.target
EOF

# 启动服务
sudo systemctl daemon-reload
sudo systemctl enable --now prometheus

# 验证
curl -s localhost:9090/api/v1/status/config | jq .
```

---

## 验证监控数据

### 1. 检查 VictoriaMetrics 数据

```bash
# 检查时序数据总数
curl -s "http://192.168.0.200:8428/api/v1/status/tsdb" | jq '.data.totalSeries'

# 查询 CPU 指标
curl -s "http://192.168.0.200:8428/api/v1/query?query=cpu_usage_idle" | jq '.data.result | length'

# 查询内存指标
curl -s "http://192.168.0.200:8428/api/v1/query?query=mem_available_percent" | jq '.data.result[0].value'
```

### 2. 检查 Nightingale 主机列表

登录 Nightingale Web UI: http://192.168.0.200:17000

- 默认用户名: `root`
- 默认密码: `root.2020`

验证步骤：
1. 进入 "基础设施" -> "机器列表"
2. 确认已注册的主机出现在列表中
3. 查看主机的 CPU、内存、磁盘指标

### 3. 指标查询示例

在 Nightingale 的 "即时查询" 中测试：

```promql
# CPU 使用率
100 - cpu_usage_idle

# 内存使用率
100 - mem_available_percent

# 磁盘使用率
100 - disk_free_percent{path="/"}

# 网络接收速率
rate(net_bytes_recv[5m])

# 系统负载
system_load1
```

---

## 故障排查

### 问题 1: Categraf 无法连接 Nightingale

**症状**: Categraf 日志显示连接错误

```bash
# 检查日志
sudo journalctl -u categraf -n 50 --no-pager
```

**解决方案**:
```bash
# 1. 检查网络连通性
curl -v "http://192.168.0.200:17000/api/n9e/version"

# 2. 检查配置文件中的 URL
grep -A 5 "\[\[writers\]\]" /opt/categraf/conf/config.toml

# 3. 确保 Nightingale 服务运行
docker ps | grep nightingale
```

### 问题 2: Node Exporter 服务启动失败

**症状**: systemctl status 显示 failed

```bash
# 检查详细错误
sudo journalctl -u node_exporter -n 50

# 常见问题：端口冲突
sudo ss -tlnp | grep 9100
```

**解决方案**:
```bash
# 如果端口被占用，修改端口
sudo sed -i 's/:9100/:9101/' /etc/systemd/system/node_exporter.service
sudo systemctl daemon-reload
sudo systemctl restart node_exporter
```

### 问题 3: VictoriaMetrics 无数据

**症状**: totalSeries 为 0

```bash
# 检查 VictoriaMetrics 状态
curl -s "http://192.168.0.200:8428/api/v1/status/tsdb" | jq .
```

**解决方案**:
```bash
# 1. 检查 Nightingale 到 VictoriaMetrics 的连接
docker exec -it nightingale curl -s http://victoriametrics:8428/api/v1/status/tsdb

# 2. 检查 Nightingale 配置
docker exec -it nightingale cat /app/etc/config.toml | grep -A 5 "Writers"

# 3. 确保 Categraf 正在发送数据
curl -s localhost:9100/metrics | wc -l
```

### 问题 4: AppHub 包下载失败

**症状**: curl 返回 404 或连接超时

```bash
# 检查 AppHub 服务
docker ps | grep apphub
docker logs apphub --tail 20
```

**解决方案**:
```bash
# 1. 重建 AppHub 镜像
./build.sh build apphub

# 2. 或手动从 GitHub 下载
export GITHUB_MIRROR="https://ghfast.top/"
curl -fsSL "${GITHUB_MIRROR}https://github.com/flashcatcloud/categraf/releases/download/v0.4.25/categraf-v0.4.25-linux-amd64.tar.gz" -o categraf.tar.gz
```

---

## 附录

### A. 模板变量说明

以下变量用于安装脚本模板渲染（在 `.env` 文件中配置）：

| 变量 | 说明 | 默认值 |
|------|------|--------|
| `EXTERNAL_HOST` | AppHub/Nightingale 外部访问地址 | 192.168.0.200 |
| `APPHUB_PORT` | AppHub 端口 | 28080 |
| `NIGHTINGALE_PORT` | Nightingale 端口 | 17000 |
| `CATEGRAF_VERSION` | Categraf 版本 | v0.4.25 |
| `NODE_EXPORTER_VERSION` | Node Exporter 版本 | 1.8.2 |
| `PROMETHEUS_VERSION` | Prometheus 版本 | 3.4.1 |

### B. 安装脚本模板位置

```
scripts/templates/
├── install-categraf.sh.tpl        # Categraf 安装模板
├── install-node-exporter.sh.tpl   # Node Exporter 安装模板
└── install-prometheus.sh.tpl      # Prometheus 安装模板 (待创建)

# 渲染后的脚本位置
scripts/
├── install-categraf.sh
├── install-node-exporter.sh
└── install-prometheus.sh
```

### C. 渲染安装脚本

```bash
# 渲染所有模板
./build.sh render

# 或手动渲染单个模板
sed -e 's/{{EXTERNAL_HOST}}/192.168.0.200/g' \
    -e 's/{{APPHUB_PORT}}/28080/g' \
    -e 's/{{NIGHTINGALE_PORT}}/17000/g' \
    -e 's/{{CATEGRAF_VERSION}}/v0.4.25/g' \
    scripts/templates/install-categraf.sh.tpl > scripts/install-categraf.sh
```

### D. 快速部署命令

```bash
# 一键部署 Categraf (推荐)
curl -fsSL http://192.168.0.200:28080/scripts/install-categraf.sh | sudo bash

# 一键部署 Node Exporter
curl -fsSL http://192.168.0.200:28080/scripts/install-node-exporter.sh | sudo bash

# 验证监控数据
curl -s "http://192.168.0.200:8428/api/v1/status/tsdb" | jq '.data.totalSeries'
```

---

## 更新历史

| 日期 | 版本 | 说明 |
|------|------|------|
| 2024-11-28 | 1.0 | 初始版本，包含 Categraf、Node Exporter、Prometheus 部署流程 |

---

**维护者**: AI Infra Matrix Team  
**最后更新**: 2024-11-28
