# 代理配置指南

## 概述

本项目的测试系统支持通过代理服务器进行网络访问，这在企业网络环境或需要翻墙的场景下非常有用。

## 代理配置

### 默认代理设置
- **HTTP Proxy**: http://127.0.0.1:7890
- **HTTPS Proxy**: http://127.0.0.1:7890  
- **SOCKS Proxy**: socks5://127.0.0.1:7890
- **No Proxy**: localhost,127.0.0.1,::1,.local

### 支持的代理客户端
- Clash for Windows/Mac
- V2Ray/V2RayN
- Shadowsocks
- 其他支持HTTP/SOCKS5代理的工具

## 使用方法

### 1. 通过 Makefile 使用代理

```bash
# 显示当前代理设置
make show-proxy

# 设置代理环境变量（显示设置命令）
make set-proxy

# 清除代理设置（显示清除命令）
make unset-proxy

# 测试代理连接
make test-proxy

# 使用代理构建镜像
make build-all

# 使用代理启动测试环境
make start-test-env

# 运行完整自动化测试（包含代理支持）
make auto-test

# 运行快速测试（包含代理支持）
make quick-test
```

### 2. 通过独立脚本使用代理

```bash
# 显示帮助
./scripts/proxy-config.sh help

# 设置代理
./scripts/proxy-config.sh set

# 测试代理连接
./scripts/proxy-config.sh test

# 显示当前代理设置
./scripts/proxy-config.sh show

# 在当前shell中设置代理
source ./scripts/proxy-config.sh set
```

### 3. 手动设置代理环境变量

```bash
# 设置代理
export HTTP_PROXY=http://127.0.0.1:7890
export HTTPS_PROXY=http://127.0.0.1:7890
export http_proxy=http://127.0.0.1:7890
export https_proxy=http://127.0.0.1:7890
export ALL_PROXY=socks5://127.0.0.1:7890
export NO_PROXY="localhost,127.0.0.1,::1,.local"

# 清除代理
unset HTTP_PROXY HTTPS_PROXY http_proxy https_proxy ALL_PROXY NO_PROXY
```

## Docker 构建代理支持

项目的 Docker 构建过程支持代理参数：

```bash
# 手动构建时传递代理参数
docker build \
  --build-arg HTTP_PROXY=http://127.0.0.1:7890 \
  --build-arg HTTPS_PROXY=http://127.0.0.1:7890 \
  -t image-name .

# 通过 docker-compose 构建（自动传递代理参数）
docker-compose build --build-arg HTTP_PROXY=http://127.0.0.1:7890
```

## 代理配置文件

### Go 模块代理
在 Dockerfile 中配置了中国镜像源：
```dockerfile
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn
```

### NPM 镜像源
在前端 Dockerfile 中配置了npm镜像源：
```dockerfile
RUN npm config set registry https://registry.npmmirror.com/
```

## 故障排除

### 1. 代理连接问题

```bash
# 检查代理服务是否运行
curl -I http://127.0.0.1:7890

# 测试HTTP代理
HTTP_PROXY=http://127.0.0.1:7890 curl -I http://www.google.com

# 测试HTTPS代理
HTTPS_PROXY=http://127.0.0.1:7890 curl -I https://www.google.com
```

### 2. Docker 构建问题

```bash
# 查看构建日志
docker-compose build --no-cache

# 检查代理参数是否正确传递
docker build --build-arg HTTP_PROXY=$HTTP_PROXY -t test .
```

### 3. 网络连接问题

```bash
# 检查代理客户端状态
# Clash: 查看 http://127.0.0.1:9090/ui
# V2Ray: 检查本地配置

# 验证代理端口
netstat -tulpn | grep 7890
```

## 配置自定义代理

如果需要使用不同的代理地址，可以修改以下文件：

1. **Makefile**: 更新 `PROXY_HTTP`、`PROXY_HTTPS`、`PROXY_SOCKS` 变量
2. **scripts/proxy-config.sh**: 更新代理地址常量
3. **所有测试脚本**: 更新脚本开头的代理环境变量

例如，使用不同端口的代理：

```bash
# 在 Makefile 中
PROXY_HTTP := http://127.0.0.1:1080
PROXY_HTTPS := http://127.0.0.1:1080
PROXY_SOCKS := socks5://127.0.0.1:1080
```

## 安全注意事项

1. **本地环境**: 默认配置假设代理运行在本地（127.0.0.1）
2. **生产环境**: 请勿在生产环境中使用代理，或确保代理安全性
3. **凭据管理**: 如果代理需要认证，请安全地管理凭据
4. **网络策略**: 确保代理配置符合组织的网络安全策略

## 性能优化

1. **镜像源**: 使用国内镜像源加速下载
2. **缓存**: Docker 构建时启用缓存
3. **并行下载**: 合理配置代理客户端的并发连接数

## 集成示例

### CI/CD 环境
```yaml
# GitHub Actions 示例
env:
  HTTP_PROXY: http://proxy.company.com:8080
  HTTPS_PROXY: http://proxy.company.com:8080

steps:
  - name: Build with proxy
    run: make build-all
```

### 企业环境
```bash
# 企业代理配置
export HTTP_PROXY=http://proxy.company.com:8080
export HTTPS_PROXY=http://proxy.company.com:8080
export NO_PROXY="localhost,127.0.0.1,*.company.com"

make auto-test
```
