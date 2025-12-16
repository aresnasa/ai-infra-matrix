# 安装脚本说明

本目录包含由模板渲染生成的安装脚本，用于在目标主机上部署各种组件。

## 目录结构

```
scripts/
├── README.md                     # 本文件
├── templates/                    # 模板源文件 (.tpl) 和运维脚本
│   ├── install-categraf.sh.tpl      # Categraf 安装模板
│   ├── install-node-exporter.sh.tpl # Node Exporter 安装模板
│   ├── install-prometheus.sh.tpl    # Prometheus 安装模板
│   ├── install-salt-minion.sh.tpl   # Salt Minion 安装模板
│   ├── ops-daily-inspection.sh      # 日常巡检脚本
│   ├── ops-gpu-health-check.sh      # GPU 健康检查脚本
│   └── ops-collect-sysinfo.sh       # 系统信息采集脚本
├── install-categraf.sh           # 渲染后的 Categraf 安装脚本
├── install-node-exporter.sh      # 渲染后的 Node Exporter 安装脚本
├── install-prometheus.sh         # 渲染后的 Prometheus 安装脚本
├── install-salt-minion.sh        # 渲染后的 Salt Minion 安装脚本
└── keyvault-sync.sh              # KeyVault 密钥同步脚本
```

## 运维脚本

### 日常巡检脚本

GPU 集群和物理机日常巡检脚本，包含以下检查项：

```bash
# 执行日常巡检
bash scripts/templates/ops-daily-inspection.sh

# 通过 SaltStack 批量执行
salt '*' cmd.run 'bash /path/to/ops-daily-inspection.sh'
```

检查项目：
- 系统基础信息（操作系统、内核、CPU）
- 内存状态和使用率
- 磁盘状态和使用率告警
- GPU 状态 (NVIDIA)、温度、XID 错误
- NPU 状态 (华为昇腾)
- 网络状态
- InfiniBand 状态
- 关键服务状态 (docker, kubelet, slurmd, salt-minion)
- 最近错误日志

### GPU 健康检查脚本

深度检查 GPU 健康状态：

```bash
# 检查 GPU 健康状态，预期 8 块 GPU
bash scripts/templates/ops-gpu-health-check.sh --expected-gpus 8

# 通过 SaltStack 批量执行
salt 'gpu*' cmd.run 'EXPECTED_GPUS=8 bash /path/to/ops-gpu-health-check.sh'
```

检查项目：
- 驱动版本和 CUDA 版本
- GPU 列表和数量（掉卡检测）
- GPU 温度、功耗、风扇
- GPU 利用率和显存
- ECC 错误
- PCIe 带宽
- 持久模式和计算模式
- GPU 上运行的进程
- XID 错误日志和解释

### 系统信息采集脚本

采集完整系统配置信息（JSON 格式），用于资产管理：

```bash
# 采集系统信息（JSON 格式）
bash scripts/templates/ops-collect-sysinfo.sh

# 保存到文件
bash scripts/templates/ops-collect-sysinfo.sh > /tmp/$(hostname)-sysinfo.json

# 通过 SaltStack 批量采集
salt '*' cmd.run 'bash /path/to/ops-collect-sysinfo.sh' --out=json
```

输出字段：
- hostname: 主机名
- os: 操作系统信息
- cpu: CPU 信息
- memory: 内存信息
- gpu: GPU 信息 (NVIDIA)
- npu: NPU 信息 (华为昇腾)
- disks: 磁盘列表
- network_interfaces: 网卡列表
- infiniband: InfiniBand 信息
- services: 服务状态

## 模板渲染

### 渲染流程

1. 模板文件位于 `scripts/templates/*.tpl`
2. 运行 `./build.sh render` 命令
3. 渲染后的脚本输出到 `scripts/` 目录

### 模板变量

模板使用 `{{VARIABLE}}` 格式的占位符，由 `build.sh` 从 `.env` 文件读取并替换：

| 变量 | 说明 | 示例值 |
|------|------|--------|
| `EXTERNAL_HOST` | 外部访问地址 | `192.168.0.200` |
| `APPHUB_PORT` | AppHub 端口 | `28080` |
| `NIGHTINGALE_PORT` | Nightingale 端口 | `17000` |
| `CATEGRAF_VERSION` | Categraf 版本 | `v0.4.25` |
| `NODE_EXPORTER_VERSION` | Node Exporter 版本 | `v1.8.2` |
| `PROMETHEUS_VERSION` | Prometheus 版本 | `v3.4.1` |
| `SALT_MASTER_HOST` | Salt Master 地址 | 同 `EXTERNAL_HOST` |
| `SALT_MASTER_PORT` | Salt Master 端口 | `4505` |

### 手动渲染

```bash
# 渲染所有模板
./build.sh render


# 强制重新渲染（忽略缓存）
./build.sh sync
```

## 安装脚本使用

### 方式一：从 AppHub 下载执行（推荐）

```bash
# 安装 Categraf
curl -fsSL http://192.168.0.200:28080/scripts/install-categraf.sh | sudo bash

# 安装 Node Exporter
curl -fsSL http://192.168.0.200:28080/scripts/install-node-exporter.sh | sudo bash

# 安装 Prometheus
curl -fsSL http://192.168.0.200:28080/scripts/install-prometheus.sh | sudo bash

# 安装 Salt Minion
curl -fsSL http://192.168.0.200:28080/scripts/install-salt-minion.sh | sudo bash
```

### 方式二：直接执行本地脚本

```bash
# 在项目目录中
sudo ./scripts/install-categraf.sh
sudo ./scripts/install-node-exporter.sh
sudo ./scripts/install-prometheus.sh
sudo ./scripts/install-salt-minion.sh
```

## 脚本功能详情

### install-categraf.sh

Categraf 是一个多功能的监控数据采集代理，支持 Push 模式上报到 Nightingale。

**主要功能**：
- 自动检测系统架构 (amd64/arm64)
- 从 AppHub 下载安装包（支持 fallback 到 GitHub）
- 配置上报地址到 Nightingale
- 创建 systemd 服务
- 配置默认采集器（CPU、内存、磁盘、网络等）

**使用参数**：
```bash
# 使用环境变量覆盖默认配置
APPHUB_HOST=10.0.0.100 \
NIGHTINGALE_HOST=10.0.0.100 \
CATEGRAF_VERSION=v0.4.20 \
./scripts/install-categraf.sh
```

### install-node-exporter.sh

Node Exporter 是 Prometheus 生态的标准节点指标导出器。

**主要功能**：
- 支持 amd64/arm64 架构
- 创建专用服务用户
- 配置 textfile collector 目录
- systemd/OpenRC 服务支持

**使用参数**：
```bash
./scripts/install-node-exporter.sh --help
./scripts/install-node-exporter.sh --port 9101        # 自定义端口
./scripts/install-node-exporter.sh --use-official     # 从 GitHub 下载
```

### install-prometheus.sh

Prometheus 时序数据库和监控系统。

**主要功能**：
- 完整的 Prometheus 服务器安装
- 自动创建配置文件和目录
- 支持文件发现配置
- 可配置数据保留时间

**使用参数**：
```bash
./scripts/install-prometheus.sh --help
./scripts/install-prometheus.sh --port 9091 --retention 15d
./scripts/install-prometheus.sh --config-only    # 仅更新配置
```

### install-salt-minion.sh

SaltStack Minion 安装脚本，用于节点配置管理。

**主要功能**：
- 从 AppHub 下载 Salt Minion 包
- 配置 Master 连接地址
- 自动注册到 Salt Master

## 注意事项

1. **权限要求**：所有安装脚本需要 root 权限执行
2. **网络要求**：目标主机需要能访问 AppHub 服务器
3. **端口冲突**：注意检查端口 9100（Categraf/Node Exporter）和 9090（Prometheus）是否被占用
4. **版本管理**：通过 `.env` 文件统一管理版本号

## 故障排查

### 脚本未更新

```bash
# 强制重新渲染
./build.sh sync

# 检查变量替换
grep -E "{{[A-Z_]+}}" scripts/*.sh
```

### 下载失败

```bash
# 检查 AppHub 可用性
curl -sI http://192.168.0.200:28080/pkgs/categraf/VERSION

# 使用 GitHub 镜像
export GITHUB_MIRROR="https://ghfast.top/"
```

### 服务启动失败

```bash
# 检查服务状态
sudo systemctl status categraf
sudo journalctl -u categraf -n 50

# 检查端口
sudo ss -tlnp | grep 9100
```

## 相关文档

- [监控部署 SOP](../docs/MONITORING_DEPLOYMENT_SOP.md)
- [构建系统指南](../docs/BUILD_AUTO_VERSION_GUIDE.md)
- [用户指南](../docs/USER_GUIDE.md)

---

**维护者**: AI Infra Matrix Team  
**最后更新**: 2024-11-28
