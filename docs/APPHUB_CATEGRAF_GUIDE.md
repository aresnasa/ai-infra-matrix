# Categraf 集成指南

本文档说明如何通过 AppHub 部署和使用 Categraf 监控采集器。

## 概述

Categraf 是 Nightingale 监控系统的默认数据采集器，支持：

- **Metrics 采集**: CPU、内存、磁盘、网络等系统指标
- **Log 采集**: 日志文件收集和解析
- **Trace 采集**: 分布式追踪数据
- **Event 采集**: 自定义事件上报

AppHub 为 Categraf 提供了 **x86_64** 和 **ARM64** 两种架构的预编译包。

## 架构支持

| 架构 | 下载链接 | 适用系统 |
|------|----------|----------|
| AMD64 (x86_64) | `http://<server>/pkgs/categraf/categraf-latest-linux-amd64.tar.gz` | Ubuntu、Debian、RHEL、Rocky、Alpine (x86) |
| ARM64 (aarch64) | `http://<server>/pkgs/categraf/categraf-latest-linux-arm64.tar.gz` | Ubuntu ARM、Debian ARM、Rocky ARM、Alpine ARM |

## 快速开始

### 1. 下载和安装

```bash
# 检测系统架构
ARCH=$(uname -m)
if [ "$ARCH" = "x86_64" ]; then
    PKG_URL="http://192.168.0.200:8080/pkgs/categraf/categraf-latest-linux-amd64.tar.gz"
elif [ "$ARCH" = "aarch64" ]; then
    PKG_URL="http://192.168.0.200:8080/pkgs/categraf/categraf-latest-linux-arm64.tar.gz"
else
    echo "Unsupported architecture: $ARCH"
    exit 1
fi

# 下载
wget $PKG_URL -O categraf.tar.gz

# 解压
tar xzf categraf.tar.gz
cd categraf-*

# 查看包内容
ls -la
# 目录结构:
#   bin/        - categraf 可执行文件
#   conf/       - 配置文件
#   logs/       - 日志目录（空）
#   install.sh  - 安装脚本
#   uninstall.sh - 卸载脚本
#   categraf.service - systemd 服务文件
#   README.md   - 使用说明

# 安装到系统
sudo ./install.sh
```

### 2. 配置 Categraf

安装后需要配置 Nightingale 服务器地址：

```bash
sudo vim /usr/local/categraf/conf/config.toml
```

关键配置项：

```toml
# 全局配置
[global]
# 采集间隔（秒）
interval = 15

# 主机名（默认自动获取）
hostname = ""

# 是否启用本地模式（true=写入本地文件，false=上报到服务器）
enable_local = false

# Writer 配置 - 数据上报目标
[[writers]]
# Nightingale 服务器地址
url = "http://192.168.0.200:8080/prometheus/v1/write"

# 基础认证（如果启用）
# basic_auth_user = ""
# basic_auth_pass = ""

# 请求超时
timeout = "5s"

# 自定义标签（可选）
[global.labels]
region = "cn-hangzhou"
env = "production"
```

### 3. 启动服务

```bash
# 启用自启动
sudo systemctl enable categraf

# 启动服务
sudo systemctl start categraf

# 查看状态
sudo systemctl status categraf

# 查看日志
sudo journalctl -u categraf -f
```

### 4. 验证数据上报

等待 15-30 秒后，在 Nightingale 监控页面检查是否收到数据：

```bash
# 方法1: 通过 Web 界面
# 访问 http://192.168.0.200:8080/monitoring
# 选择 "即时查询" -> 输入 "up" 或 "node_cpu_seconds_total"

# 方法2: 通过 API 查询
curl -X POST http://192.168.0.200:8080/api/n9e/prometheus/api/v1/query \
  -H 'Content-Type: application/json' \
  -d '{
    "query": "up",
    "time": "'$(date +%s)'"
  }'
```

## 配置采集插件

Categraf 采用插件化架构，默认启用以下插件：

### 系统指标采集

```bash
# 编辑配置
sudo vim /usr/local/categraf/conf/input.cpu/cpu.toml
sudo vim /usr/local/categraf/conf/input.mem/mem.toml
sudo vim /usr/local/categraf/conf/input.disk/disk.toml
sudo vim /usr/local/categraf/conf/input.net/net.toml
```

示例 - CPU 采集器：

```toml
[[instances]]
# 是否采集每个 CPU 核心的指标
collect_cpu_time = true

# 采集间隔（覆盖全局配置）
# interval = 15
```

### 应用监控

#### MySQL 监控

```bash
sudo vim /usr/local/categraf/conf/input.mysql/mysql.toml
```

```toml
[[instances]]
address = "root:password@tcp(127.0.0.1:3306)/"
timeout_seconds = 3

# 额外标签
labels = { cluster="main" }
```

#### Redis 监控

```bash
sudo vim /usr/local/categraf/conf/input.redis/redis.toml
```

```toml
[[instances]]
address = "127.0.0.1:6379"
password = ""

# 采集慢查询
# slowlog_max_len = 128
```

#### HTTP 健康检查

```bash
sudo vim /usr/local/categraf/conf/input.http_response/http_response.toml
```

```toml
[[instances]]
targets = [
  "http://localhost:8080/health",
  "https://example.com",
]

# 超时时间
timeout = 3

# 期望的状态码
expect_status = 200
```

### 日志采集

```bash
sudo vim /usr/local/categraf/conf/input.log/log.toml
```

```toml
[[instances]]
# 日志文件路径（支持通配符）
log_path = "/var/log/nginx/access.log"

# 日志格式（支持正则表达式）
# pattern = '...'

# 附加标签
labels = { service="nginx", type="access" }
```

## 高级配置

### 自定义指标上报

创建自定义脚本采集器：

```bash
sudo mkdir -p /usr/local/categraf/conf/input.exec
sudo vim /usr/local/categraf/conf/input.exec/custom.toml
```

```toml
[[instances]]
# 执行的命令
commands = [
  "/usr/local/bin/my_metrics_script.sh",
]

# 超时时间
timeout = 10

# 数据格式（influx, prometheus）
data_format = "influx"

# 采集间隔
interval = 60
```

示例脚本 `/usr/local/bin/my_metrics_script.sh`：

```bash
#!/bin/bash
# 输出 Influx 行协议格式
echo "custom_metric,host=$(hostname),type=test value=123 $(date +%s)000000000"
```

### 多实例配置

对于需要监控多个相同类型服务的场景：

```toml
# 实例1
[[instances]]
address = "mysql-master:3306"
labels = { role="master" }

# 实例2
[[instances]]
address = "mysql-slave-1:3306"
labels = { role="slave" }

# 实例3
[[instances]]
address = "mysql-slave-2:3306"
labels = { role="slave" }
```

### 数据过滤

```toml
# 在 writer 中配置过滤规则
[[writers]]
url = "http://192.168.0.200:8080/prometheus/v1/write"

# 只上报特定指标
[writers.filters]
# 白名单（只上报匹配的）
# namepass = ["cpu_*", "mem_*"]

# 黑名单（排除匹配的）
# namedrop = ["temp_*"]

# 标签过滤
# tagpass = { env = ["prod"] }
# tagdrop = { debug = ["true"] }
```

## 故障排查

### 1. 服务无法启动

```bash
# 查看详细日志
sudo journalctl -u categraf -n 50 --no-pager

# 检查配置文件语法
/usr/local/categraf/bin/categraf --test --config /usr/local/categraf/conf/config.toml

# 检查文件权限
ls -la /usr/local/categraf/bin/categraf
ls -la /usr/local/categraf/conf/
```

### 2. 数据未上报

```bash
# 检查网络连通性
curl -v http://192.168.0.200:8080/prometheus/v1/write

# 查看 Categraf 日志中的错误
sudo journalctl -u categraf -f | grep -i error

# 手动测试采集
/usr/local/categraf/bin/categraf --inputs cpu,mem --test
```

### 3. 性能问题

```bash
# 查看 Categraf 进程资源占用
top -p $(pgrep categraf)

# 调整采集间隔
sudo vim /usr/local/categraf/conf/config.toml
# 增加 interval 值，如从 15 改为 30

# 禁用不需要的插件
# 重命名或删除对应配置文件
sudo mv /usr/local/categraf/conf/input.disk/disk.toml /usr/local/categraf/conf/input.disk/disk.toml.disabled
```

### 4. 配置重载

```bash
# 修改配置后重启服务
sudo systemctl restart categraf

# 或发送 SIGHUP 信号热重载
sudo kill -HUP $(pgrep categraf)
```

## 卸载

```bash
# 进入原解压目录
cd categraf-*-linux-*

# 运行卸载脚本
sudo ./uninstall.sh

# 或手动卸载
sudo systemctl stop categraf
sudo systemctl disable categraf
sudo rm -f /etc/systemd/system/categraf.service
sudo systemctl daemon-reload
sudo rm -rf /usr/local/categraf
```

## 参考资源

- **Categraf GitHub**: https://github.com/flashcatcloud/categraf
- **Nightingale 文档**: https://flashcat.cloud/docs/
- **插件列表**: https://github.com/flashcatcloud/categraf/tree/main/inputs
- **配置示例**: https://github.com/flashcatcloud/categraf/tree/main/conf

## 与 AppHub 集成

### 构建说明

AppHub 在构建过程中会自动编译 Categraf：

1. **Stage 4** (categraf-builder): 使用 Go 交叉编译构建两种架构
2. **构建参数**: `CGO_ENABLED=0` 生成静态链接二进制文件
3. **版本控制**: 通过 `CATEGRAF_VERSION` ARG 指定版本

### 自定义构建

如需构建特定版本：

```bash
# 编辑 Dockerfile
vim src/apphub/Dockerfile

# 修改 ARG CATEGRAF_VERSION
ARG CATEGRAF_VERSION=v0.3.90  # 改为所需版本

# 重新构建
docker build -t ai-infra-apphub:custom -f src/apphub/Dockerfile src/apphub
```

### 访问包文件

```bash
# 列出所有可用版本
curl http://192.168.0.200:8080/pkgs/categraf/

# 下载特定版本（AMD64）
wget http://192.168.0.200:8080/pkgs/categraf/categraf-v0.3.90-linux-amd64.tar.gz

# 下载最新版本（ARM64）
wget http://192.168.0.200:8080/pkgs/categraf/categraf-latest-linux-arm64.tar.gz
```

## 监控最佳实践

### 1. 标签规范

统一使用标签方便后续查询和告警：

```toml
[global.labels]
# 环境标签
env = "production"  # production, staging, development

# 区域标签
region = "cn-east"  # cn-east, cn-west, us-west

# 集群标签
cluster = "k8s-prod-01"

# 业务标签
business = "order-service"
```

### 2. 采集间隔建议

| 指标类型 | 建议间隔 | 说明 |
|---------|---------|------|
| 系统指标 | 10-15s | CPU、内存、磁盘、网络 |
| 应用指标 | 30-60s | MySQL、Redis、Nginx |
| 业务指标 | 60-300s | 订单量、支付成功率等 |
| 日志采集 | 实时 | 错误日志、审计日志 |

### 3. 告警配置

在 Nightingale 中基于 Categraf 采集的指标配置告警：

```promql
# CPU 使用率超过 80%
100 - (avg by (instance) (irate(node_cpu_seconds_total{mode="idle"}[5m])) * 100) > 80

# 内存使用率超过 90%
(1 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)) * 100 > 90

# 磁盘使用率超过 85%
(1 - (node_filesystem_avail_bytes / node_filesystem_size_bytes)) * 100 > 85

# MySQL 主从延迟超过 10 秒
mysql_slave_status_seconds_behind_master > 10
```

## 更新日志

| 日期 | 版本 | 变更说明 |
|------|------|----------|
| 2025-01-XX | v0.3.90 | 初始集成到 AppHub，支持 x86_64 和 ARM64 |

---

**维护**: AI-Infra-Matrix Team  
**更新**: 2025-01-XX  
