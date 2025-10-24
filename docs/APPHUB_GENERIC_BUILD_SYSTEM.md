# AppHub 泛化构建系统

## 目录结构

```
src/apphub/
├── Dockerfile                          # 主 Dockerfile
├── scripts/
│   ├── build-app.sh                   # 通用构建脚本
│   ├── categraf/                      # Categraf 应用
│   │   ├── categraf-build.sh         # Categraf 构建脚本
│   │   ├── install.sh                # 安装脚本模板
│   │   ├── uninstall.sh              # 卸载脚本模板
│   │   ├── systemd.service           # systemd 服务模板
│   │   └── readme.md                 # README 模板
│   ├── slurm/                         # SLURM 应用
│   │   ├── install.sh
│   │   └── uninstall.sh
│   └── <future-app>/                  # 未来的应用
│       ├── <app>-build.sh
│       └── ...
└── ...
```

## 设计理念

### 1. 应用隔离
每个应用的所有文件都放在 `scripts/<app>/` 目录下，互不干扰。

### 2. 泛化构建
Dockerfile 中的构建步骤高度泛化，添加新应用只需：
1. 创建 `scripts/<app>/` 目录
2. 添加 `<app>-build.sh` 构建脚本
3. 在 Dockerfile 中复制该目录并调用 `build-app.sh <app>`

### 3. 标准化接口
所有应用构建脚本遵循统一的接口：
- **环境变量输入**:
  - `BUILD_DIR`: 构建目录（默认 `/build`）
  - `OUTPUT_DIR`: 输出目录（默认 `/out`）
  - `SCRIPT_DIR`: 脚本目录（`/scripts/<app>`）
  - 其他应用特定变量（如 `CATEGRAF_VERSION`）

- **输出要求**:
  - 所有构建产物放到 `${OUTPUT_DIR}/`
  - 支持 `tar.gz` 或其他包格式

## Dockerfile 构建阶段模板

### 添加新应用的步骤

假设要添加名为 `myapp` 的新应用：

#### 1. 创建应用目录和脚本

```bash
mkdir -p src/apphub/scripts/myapp
```

#### 2. 创建构建脚本 `myapp-build.sh`

```bash
#!/bin/bash
set -e

# 使用环境变量
MYAPP_VERSION=${MYAPP_VERSION:-"v1.0.0"}
BUILD_DIR=${BUILD_DIR:-"/build"}
OUTPUT_DIR=${OUTPUT_DIR:-"/out"}
SCRIPT_DIR=${SCRIPT_DIR:-"/scripts/myapp"}

echo "Building MyApp ${MYAPP_VERSION}..."

# 你的构建逻辑
# ...

# 输出到 OUTPUT_DIR
tar czf "${OUTPUT_DIR}/myapp-${MYAPP_VERSION}.tar.gz" ...

echo "✓ MyApp build completed"
```

#### 3. 添加其他模板文件（可选）

```bash
# 安装/卸载脚本
scripts/myapp/install.sh
scripts/myapp/uninstall.sh

# 配置模板
scripts/myapp/config.toml
scripts/myapp/readme.md
```

#### 4. 在 Dockerfile 中添加构建阶段

```dockerfile
# =============================================================================
# Stage X: Build MyApp
# =============================================================================
FROM <base-image> AS myapp-builder

# 应用版本配置
ARG MYAPP_VERSION=v1.0.0

# 配置镜像源（如需要）
RUN set -eux; \
    # ... 镜像源配置 ...

# 安装构建依赖
RUN apk add --no-cache git make bash tar gzip ...

# 复制构建脚本（只需这一行！）
COPY scripts/build-app.sh /scripts/build-app.sh
COPY scripts/myapp/ /scripts/myapp/
RUN chmod +x /scripts/build-app.sh /scripts/myapp/*.sh

# 创建输出目录
RUN mkdir -p /out

# 执行构建（使用通用脚本）
RUN MYAPP_VERSION=${MYAPP_VERSION} \
    BUILD_DIR=/build \
    OUTPUT_DIR=/out \
    /scripts/build-app.sh myapp
```

#### 5. 在最终阶段复制包文件

```dockerfile
# Stage 5: final
FROM nginx:alpine

# ...

# 复制 MyApp 包
COPY --from=myapp-builder /out/ /usr/share/nginx/html/pkgs/myapp/

# 创建符号链接
RUN cd /usr/share/nginx/html/pkgs/myapp && \
    ln -sf myapp-v1.0.0.tar.gz myapp-latest.tar.gz
```

## 泛化的优势

### 1. 简洁的 Dockerfile
Dockerfile 中的构建阶段非常简洁，只需 3 个核心步骤：
```dockerfile
COPY scripts/build-app.sh /scripts/build-app.sh
COPY scripts/<app>/ /scripts/<app>/
RUN /scripts/build-app.sh <app>
```

### 2. 易于维护
- 所有应用逻辑都在各自的脚本中
- Dockerfile 不包含复杂的构建逻辑
- 修改应用构建只需编辑对应的 `<app>-build.sh`

### 3. 易于扩展
添加新应用只需：
1. 创建新目录 `scripts/<newapp>/`
2. 添加 `<newapp>-build.sh`
3. 在 Dockerfile 复制并调用

### 4. 可独立测试
每个应用的构建脚本可以独立测试：
```bash
docker run --rm -it \
  -v $(pwd)/scripts:/scripts \
  golang:alpine \
  /scripts/build-app.sh categraf
```

### 5. 标准化流程
所有应用遵循相同的构建模式：
- 环境变量配置
- 构建逻辑
- 打包输出

## 当前已集成应用

### Categraf
- **路径**: `scripts/categraf/`
- **构建脚本**: `categraf-build.sh`
- **架构**: AMD64, ARM64
- **包格式**: tar.gz
- **下载地址**: `/pkgs/categraf/`

### SLURM
- **路径**: `scripts/slurm/`
- **安装脚本**: `install.sh`, `uninstall.sh`
- **包格式**: deb, rpm, tar.gz
- **下载地址**: `/pkgs/slurm-{deb,rpm,apk}/`

## 最佳实践

### 1. 命名约定
- 构建脚本: `<app>-build.sh`
- 安装脚本: `install.sh`
- 卸载脚本: `uninstall.sh`
- 配置模板: 使用占位符（如 `VERSION_PLACEHOLDER`）

### 2. 环境变量
- 使用大写命名
- 提供默认值: `${VAR:-"default"}`
- 传递给构建脚本: `export VAR`

### 3. 错误处理
```bash
set -e  # 遇错即退
set -u  # 未定义变量报错
set -o pipefail  # 管道错误传递
```

### 4. 输出规范
- 使用 emoji 增强可读性: 📥 📦 🔨 ✓
- 打印关键信息: 版本、架构、包大小
- 输出清单: `ls -lh ${OUTPUT_DIR}/*.tar.gz`

### 5. 清理临时文件
```bash
# 构建完成后
rm -rf ${BUILD_DIR}/temp-*
```

## 故障排查

### 构建失败

```bash
# 查看详细日志
docker build --progress=plain --no-cache \
  -f src/apphub/Dockerfile \
  src/apphub

# 进入构建阶段调试
docker build --target categraf-builder \
  -t debug-categraf \
  -f src/apphub/Dockerfile \
  src/apphub

docker run --rm -it debug-categraf sh
```

### 脚本权限问题

```bash
# 确保脚本可执行
chmod +x src/apphub/scripts/**/*.sh
```

### 环境变量未传递

```bash
# 在 Dockerfile 中检查
RUN env | grep -i myapp
```

## 未来扩展示例

### 示例：添加 Prometheus

```bash
# 1. 创建目录
mkdir -p scripts/prometheus

# 2. 创建构建脚本
cat > scripts/prometheus/prometheus-build.sh <<'EOF'
#!/bin/bash
set -e

PROMETHEUS_VERSION=${PROMETHEUS_VERSION:-"v2.45.0"}
ARCH=$(uname -m)

wget https://github.com/prometheus/prometheus/releases/download/${PROMETHEUS_VERSION}/prometheus-${PROMETHEUS_VERSION}.linux-${ARCH}.tar.gz
tar xzf prometheus-${PROMETHEUS_VERSION}.linux-${ARCH}.tar.gz
mv prometheus-${PROMETHEUS_VERSION}.linux-${ARCH} ${OUTPUT_DIR}/
EOF

# 3. 在 Dockerfile 添加阶段
# FROM alpine AS prometheus-builder
# COPY scripts/build-app.sh /scripts/build-app.sh
# COPY scripts/prometheus/ /scripts/prometheus/
# RUN /scripts/build-app.sh prometheus
```

## 总结

通过泛化的构建系统：
- ✅ Dockerfile 保持简洁
- ✅ 应用逻辑模块化
- ✅ 易于添加新应用
- ✅ 统一的构建流程
- ✅ 便于测试和维护

---

**维护**: AI-Infra-Matrix Team  
**更新**: 2025-01-24
