# SLURM 集群初始化指南

本文档说明如何使用后端API初始化SLURM集群、准备简单的APT仓库并在多个节点上通过SSH自动安装。

## 概览

流程：

1. 在准备作为仓库的主机上安装 Nginx + dpkg-dev，创建 `deb` 仓库目录
2. 将构建好的 slurm `.deb` 包放入该目录，并生成 `Packages` 索引
3. 通过API让节点配置APT源并安装 `slurmctld`/`slurmd`

> 注意：当前实现面向Deb系发行版（Ubuntu/Debian）。RPM系仅安装部分命令，仓库配置需自行调整。

## API 端点

### 1) 准备仓库

`POST /api/slurm/repo/setup`

请求 JSON：

```json
{
  "repoHost": {"host": "repo-host", "port": 22, "user": "root", "password": "***"},
  "basePath": "/var/www/html/deb/slurm",
  "baseURL": "http://repo-host/deb/slurm",
  "enableIndex": true
}
```

说明：

- 在目标主机安装 `nginx` 和 `dpkg-dev`（Deb系）并创建 `basePath` 目录
- 可选生成 `Packages.gz` 索引
- 成功后你可将 `.deb` 包复制到 `basePath`

将包放入仓库目录后，可手动再次生成索引：

```bash
ssh root@repo-host "cd /var/www/html/deb/slurm && apt-ftparchive packages . > Packages && gzip -f Packages"
```

### 2) 初始化节点

`POST /api/slurm/init-nodes`

请求 JSON：

```json
{
  "repoURL": "http://repo-host/deb/slurm",
  "nodes": [
    {"ssh": {"host": "test-ssh-01", "port": 22, "user": "root", "password": "***"}, "role": "controller"},
    {"ssh": {"host": "test-ssh-02", "port": 22, "user": "root", "password": "***"}, "role": "node"},
    {"ssh": {"host": "test-ssh-03", "port": 22, "user": "root", "password": "***"}, "role": "node"}
  ]
}
```

行为：

- 为每个节点写入APT源 `deb [trusted=yes] <repoURL> ./` 并 `apt-get update`
- 安装 `slurmctld/slurmd/slurm-client` 并根据 `role` 启/重启对应服务

返回：

```json
{
  "success": true,
  "results": [
    {"host": "test-ssh-01", "success": true, "tookMs": 5200},
    {"host": "test-ssh-02", "success": true, "tookMs": 4100}
  ]
}
```

## 前端接入建议

- 在“Slurm 管理页面”增加“仓库准备”与“节点初始化”两个向导卡片
- 支持保存常用 repo 主机/URL 与节点清单
- 初始化操作提供每主机的实时日志与结果状态

## 后续增强

- 自动将构建出的 `.deb` 产物上传至仓库并生成索引
- RPM 系仓库与安装适配（createrepo/dnf/yum）
- 通过Salt/Ansible 批量执行与回滚
