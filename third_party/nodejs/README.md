# Node.js 预编译二进制包

用于 JupyterHub 容器构建，避免在 Docker 构建时从网络下载。

## 文件说明

- `node-v22.11.0-linux-x64.tar.xz` - x86_64 架构
- `node-v22.11.0-linux-arm64.tar.xz` - ARM64 架构

## 更新方法

```bash
# 使用国内镜像下载
./download.sh

# 使用官方源下载
NODE_MIRROR=https://nodejs.org/dist ./download.sh
```

## 版本信息

- Node.js: 22.11.0 LTS
- npm: 包含在 Node.js 中
