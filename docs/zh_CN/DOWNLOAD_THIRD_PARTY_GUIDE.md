# 第三方依赖下载指南

## 概述

项目提供统一的第三方组件下载脚本 `scripts/download_third_party.sh`，用于下载所有需要的第三方二进制文件到 `third_party/` 目录。

**推荐工作流：**
```bash
# 1. 预下载所有依赖（加速后续构建）
./build.sh download-deps

# 2. 将下载的文件提交到 git（团队共享缓存）
git add third_party/
git commit -m "feat: add third-party dependencies"

# 3. 构建 AppHub（自动使用预下载文件）
./build.sh apphub
```

## 支持的组件

| 组件 | 来源 | 架构支持 | 用途 |
|------|------|----------|------|
| Prometheus | prometheus/prometheus | amd64, arm64 | 监控时序数据库 |
| Node Exporter | prometheus/node_exporter | amd64, arm64 | 主机指标采集 |
| Alertmanager | prometheus/alertmanager | amd64, arm64 | 告警管理 |
| Categraf | flashcatcloud/categraf | amd64, arm64 | 夜莺监控采集器 |
| Munge | dun/munge | 源码 | Slurm 认证 |
| Singularity | sylabs/singularity | amd64, arm64 (DEB) | 容器运行时 |
| SaltStack | saltstack/salt | amd64, arm64 (DEB/RPM) | 配置管理 |

## 快速使用

### 1. 下载所有组件

```bash
# 运行统一下载脚本
./scripts/download_third_party.sh

# 使用 GitHub 镜像加速
GITHUB_MIRROR=https://gh-proxy.com/ ./scripts/download_third_party.sh

# 禁用镜像，直接从 GitHub 下载
GITHUB_MIRROR="" ./scripts/download_third_party.sh
```

### 2. 下载后的目录结构

```
third_party/
├── prometheus/
│   ├── prometheus-3.4.1.linux-amd64.tar.gz
│   ├── prometheus-3.4.1.linux-arm64.tar.gz
│   └── version.json
├── node_exporter/
│   ├── node_exporter-1.8.2.linux-amd64.tar.gz
│   ├── node_exporter-1.8.2.linux-arm64.tar.gz
│   └── version.json
├── alertmanager/
│   ├── alertmanager-0.28.1.linux-amd64.tar.gz
│   ├── alertmanager-0.28.1.linux-arm64.tar.gz
│   └── version.json
├── categraf/
│   ├── categraf-v0.4.25-linux-amd64.tar.gz
│   ├── categraf-v0.4.25-linux-arm64.tar.gz
│   └── version.json
├── munge/
│   ├── munge-0.5.16.tar.xz
│   └── version.json
├── singularity/
│   ├── singularity-ce_4.2.2-1~ubuntu22.04_amd64.deb
│   ├── singularity-ce_4.2.2-1~ubuntu22.04_arm64.deb
│   └── version.json
└── saltstack/
    ├── salt-common_3007.1_amd64.deb
    ├── salt-minion_3007.1_amd64.deb
    ├── salt-3007.1-0.x86_64.rpm
    └── version.json
```

## 版本配置

### 版本来源优先级

1. **环境变量**: 直接设置如 `PROMETHEUS_VERSION=v3.5.0`
2. **.env 文件**: 从项目根目录的 `.env` 文件读取
3. **Dockerfile**: 从 `src/apphub/Dockerfile` 读取 ARG 定义
4. **默认值**: 脚本内置的默认版本

### 修改版本

方法 1: 修改 `.env` 文件
```bash
# .env
PROMETHEUS_VERSION=v3.4.1
NODE_EXPORTER_VERSION=v1.8.2
ALERTMANAGER_VERSION=v0.28.1
```

方法 2: 通过环境变量覆盖
```bash
PROMETHEUS_VERSION=v3.5.0 ./scripts/download_third_party.sh
```

方法 3: 修改 `src/apphub/Dockerfile` 中的 ARG
```dockerfile
ARG CATEGRAF_VERSION=v0.4.26
ARG SINGULARITY_VERSION=v4.2.2
ARG SALTSTACK_VERSION=v3007.1
```

## GitHub 镜像加速

在中国大陆等网络受限地区，脚本默认使用 `gh-proxy.com` 镜像加速。

### 支持的镜像

```bash
# 默认镜像
GITHUB_MIRROR=https://gh-proxy.com/

# 其他可用镜像
GITHUB_MIRROR=https://ghproxy.net/
GITHUB_MIRROR=https://mirror.ghproxy.com/

# 禁用镜像
GITHUB_MIRROR=""
```

### 回退机制

脚本会自动进行回退：
1. 首先尝试通过镜像下载 (超时 30 秒)
2. 镜像失败后尝试直接从 GitHub 下载 (超时 60 秒)
3. 最多重试 3 次

## AppHub 独立下载脚本

除了统一脚本外，`src/apphub/scripts/` 目录下保留了各组件的独立下载脚本，用于 AppHub 容器内部使用：

```
src/apphub/scripts/
├── prometheus/
│   └── download-prometheus.sh    # Prometheus 下载
├── node_exporter/
│   └── download-node-exporter.sh # Node Exporter 下载
├── categraf/
│   └── download-categraf.sh      # Categraf 下载
│   └── install-categraf.sh       # Categraf 安装
└── ...
```

这些脚本适用于：
- AppHub 容器运行时动态下载
- 下载到 `/usr/share/nginx/html/pkgs/` 提供 HTTP 服务
- 生成 latest 符号链接

## 与 Docker 构建集成

### 构建前预下载

```bash
# 1. 预下载所有依赖
./scripts/download_third_party.sh

# 2. 构建 AppHub (会使用 third_party/ 中的文件)
./build.sh apphub
```

### Dockerfile 中使用

```dockerfile
# 复制预下载的文件
COPY third_party/categraf/ /tmp/categraf/
COPY third_party/prometheus/ /tmp/prometheus/

# 安装
RUN tar xzf /tmp/categraf/categraf-*.tar.gz -C /opt/
```

## version.json 格式

每个组件目录下会生成 `version.json` 记录下载信息：

```json
{
    "component": "prometheus",
    "version": "v3.4.1",
    "downloaded_at": "2025-01-15T10:30:00Z"
}
```

## 故障排除

### 下载失败

1. 检查网络连接
2. 尝试更换 GitHub 镜像
3. 检查版本号是否正确 (是否存在该 release)

### 文件已存在但需要重新下载

```bash
# 删除已下载的文件
rm -rf third_party/prometheus/

# 重新下载
./scripts/download_third_party.sh
```

### 版本号格式问题

- Prometheus/Node Exporter/Alertmanager: 文件名不带 `v` 前缀 (如 `prometheus-3.4.1.linux-amd64.tar.gz`)
- Categraf: 文件名带 `v` 前缀 (如 `categraf-v0.4.25-linux-amd64.tar.gz`)
- 脚本会自动处理这些差异

## 相关文档

- [AppHub 构建指南](./APPHUB_BUILD_COMPONENTS.md)
- [监控系统架构](../docs/MONITORING.md)
- [Categraf 集成](./APPHUB_CATEGRAF_GUIDE.md)
