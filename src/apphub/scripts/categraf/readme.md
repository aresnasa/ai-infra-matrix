# Categraf VERSION_PLACEHOLDER - Linux ARCH_PLACEHOLDER

Categraf 是 Nightingale 监控系统的默认采集器，支持 metric、log、trace、event 的采集。

## 安装

```bash
sudo ./install.sh
```

## 配置

编辑配置文件：
```bash
vim /usr/local/categraf/conf/config.toml
```

主要配置项：
- `interval`: 采集间隔
- `hostname`: 主机名
- `[writer]`: 数据推送目标（Nightingale 地址）

## 启动

```bash
# 启动服务
systemctl start categraf

# 查看状态
systemctl status categraf

# 查看日志
journalctl -u categraf -f
```

## 卸载

```bash
sudo ./uninstall.sh
```

## 更多信息

- GitHub: https://github.com/flashcatcloud/categraf
- 文档: https://flashcat.cloud/docs/categraf/
