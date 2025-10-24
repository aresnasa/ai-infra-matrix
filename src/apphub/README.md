# AppHub - AI Infra Matrix Package Repository

AppHub 是基于 Nginx 的软件包下载中心，提供多种分发格式的软件包。

## 可用软件包

### SLURM 工作负载管理器

- **DEB 包** (Ubuntu/Debian): `http://<server>/pkgs/slurm-deb/`
- **RPM 包** (RHEL/Rocky): `http://<server>/pkgs/slurm-rpm/`
- **Alpine 客户端工具**: `http://<server>/pkgs/slurm-apk/`

### SaltStack 配置管理

- **DEB 包** (Ubuntu/Debian): `http://<server>/pkgs/saltstack-deb/`
- **RPM 包** (RHEL/Rocky): `http://<server>/pkgs/saltstack-rpm/`

### Categraf 监控采集器

Categraf 是 Nightingale 监控系统的默认数据采集器，支持多种监控数据采集。

- **Linux AMD64 (x86_64)**: `http://<server>/pkgs/categraf/categraf-latest-linux-amd64.tar.gz`
- **Linux ARM64 (aarch64)**: `http://<server>/pkgs/categraf/categraf-latest-linux-arm64.tar.gz`

#### 快速安装 Categraf

```bash
# 下载适合你架构的包（根据系统选择 amd64 或 arm64）
wget http://<server>/pkgs/categraf/categraf-latest-linux-amd64.tar.gz

# 解压
tar xzf categraf-latest-linux-amd64.tar.gz
cd categraf-*-linux-amd64

# 安装
sudo ./install.sh

# 配置（编辑 Nightingale 服务器地址）
sudo vim /usr/local/categraf/conf/config.toml

# 启动服务
sudo systemctl enable categraf
sudo systemctl start categraf
```

更多信息：https://github.com/flashcatcloud/categraf

## APT 仓库配置

对于 DEB 包，AppHub 提供了 APT 仓库索引：
