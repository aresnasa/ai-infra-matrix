# AppHub Categraf 集成总结

## 变更说明

已成功将 Categraf 监控采集器集成到 AppHub，支持 x86_64 和 ARM64 两种架构。

## 修改的文件

### 1. Dockerfile (`src/apphub/Dockerfile`)

**变更内容**:
- 添加了新的 Stage 4: `categraf-builder`
- 使用 `golang:1.23-alpine` 作为基础镜像
- 移除所有 `cat ... <<'EOF'` heredoc 语法，改为调用独立脚本
- 复制构建脚本和模板文件到容器
- 调用 `categraf-build.sh` 执行构建
- 在 Stage 5 中添加 Categraf 包目录和拷贝命令

**关键改进**:
- ✅ 避免使用 heredoc，解决 Dockerfile linter 问题
- ✅ 构建逻辑模块化，便于维护和测试
- ✅ 支持通过 `ARG CATEGRAF_VERSION` 自定义版本

### 2. 构建脚本 (`src/apphub/scripts/categraf-build.sh`)

**功能**:
- 克隆 Categraf 仓库
- 交叉编译 AMD64 和 ARM64 二进制文件
- 创建完整的安装包（包含配置、脚本、文档）
- 生成 tar.gz 压缩包

**环境变量**:
```bash
CATEGRAF_VERSION - Categraf 版本标签（默认: v0.3.90）
CATEGRAF_REPO - Git 仓库 URL
BUILD_DIR - 构建目录（默认: /build）
OUTPUT_DIR - 输出目录（默认: /out）
```

### 3. 模板文件

创建了以下独立模板文件，避免在 Dockerfile 中使用 heredoc：

- **`scripts/categraf-systemd.service`**: systemd 服务配置
- **`scripts/categraf-install.sh`**: 安装脚本
- **`scripts/categraf-uninstall.sh`**: 卸载脚本
- **`scripts/categraf-readme.md`**: README 模板（使用占位符）

### 4. 文档

- **`docs/APPHUB_CATEGRAF_GUIDE.md`**: 详细的使用指南
- **`docs/APPHUB_CATEGRAF_BUILD_TEST.md`**: 构建和测试指南
- **`src/apphub/README.md`**: 更新了 Categraf 下载说明

### 5. 测试脚本

- **`src/apphub/test-categraf.sh`**: 快速验证脚本

## 构建流程

### Dockerfile 构建流程

```
Stage 4: categraf-builder (golang:1.23-alpine)
├─ 配置 Alpine 镜像源
├─ 安装构建依赖 (git, make, bash, tar, gzip, sed)
├─ 复制构建脚本和模板文件
│  ├─ categraf-build.sh
│  ├─ categraf-systemd.service
│  ├─ categraf-install.sh
│  ├─ categraf-uninstall.sh
│  └─ categraf-readme.md
└─ 执行 categraf-build.sh
   ├─ 克隆 Categraf 仓库
   ├─ 构建 AMD64 二进制 (CGO_ENABLED=0 GOOS=linux GOARCH=amd64)
   ├─ 构建 ARM64 二进制 (CGO_ENABLED=0 GOOS=linux GOARCH=arm64)
   ├─ 打包 AMD64 版本
   │  ├─ categraf-${VERSION}-linux-amd64/
   │  │  ├─ bin/categraf
   │  │  ├─ conf/
   │  │  ├─ install.sh
   │  │  ├─ uninstall.sh
   │  │  ├─ categraf.service
   │  │  └─ README.md
   │  └─ categraf-${VERSION}-linux-amd64.tar.gz
   └─ 打包 ARM64 版本
      ├─ categraf-${VERSION}-linux-arm64/
      └─ categraf-${VERSION}-linux-arm64.tar.gz

Stage 5: final (nginx:alpine)
├─ 创建目录 /usr/share/nginx/html/pkgs/categraf/
├─ 从 categraf-builder 复制包文件
├─ 创建符号链接
│  ├─ categraf-latest-linux-amd64.tar.gz -> categraf-v0.3.90-linux-amd64.tar.gz
│  └─ categraf-latest-linux-arm64.tar.gz -> categraf-v0.3.90-linux-arm64.tar.gz
└─ 更新包统计信息
```

## 使用方法

### 1. 构建 AppHub 镜像

```bash
# 使用默认版本 (v0.3.90)
docker build -t ai-infra-apphub:latest -f src/apphub/Dockerfile src/apphub

# 指定 Categraf 版本
docker build \
  --build-arg CATEGRAF_VERSION=v0.3.85 \
  -t ai-infra-apphub:latest \
  -f src/apphub/Dockerfile \
  src/apphub
```

### 2. 启动 AppHub

```bash
docker run -d --name apphub -p 8080:80 ai-infra-apphub:latest
```

### 3. 下载 Categraf

```bash
# AMD64 版本
wget http://192.168.0.200:8080/pkgs/categraf/categraf-latest-linux-amd64.tar.gz

# ARM64 版本
wget http://192.168.0.200:8080/pkgs/categraf/categraf-latest-linux-arm64.tar.gz
```

### 4. 安装 Categraf

```bash
# 解压
tar xzf categraf-latest-linux-amd64.tar.gz
cd categraf-*-linux-amd64

# 安装
sudo ./install.sh

# 配置
sudo vim /usr/local/categraf/conf/config.toml

# 启动
sudo systemctl enable categraf
sudo systemctl start categraf
```

## 技术亮点

### 1. 无 Heredoc 设计

**问题**: Dockerfile linter 不识别 heredoc 中的 shell 脚本语法

**解决方案**:
- 将所有文件内容提取到独立的模板文件
- 在构建脚本中使用 `cp` 和 `sed` 替换占位符
- Dockerfile 只负责调用脚本，不包含复杂逻辑

### 2. 静态编译

使用 `CGO_ENABLED=0` 生成静态链接二进制文件：
- ✅ 无需依赖系统库
- ✅ 跨发行版兼容（Ubuntu, Debian, RHEL, Alpine 等）
- ✅ 包体积更小

### 3. 模板化 README

使用占位符机制生成不同架构的 README：
```bash
sed "s/VERSION_PLACEHOLDER/${VERSION}/g; s/ARCH_PLACEHOLDER/amd64/g" \
  /scripts/categraf-readme.md > README.md
```

### 4. 符号链接

为最新版本创建 `categraf-latest-*` 符号链接：
- 用户无需关心具体版本号
- 方便自动化部署脚本

## 验证测试

### 快速测试

```bash
cd src/apphub
./test-categraf.sh
```

测试内容：
1. ✅ 检查 AppHub 镜像
2. ✅ 启动容器
3. ✅ 验证 HTTP 访问
4. ✅ 下载 AMD64 包
5. ✅ 下载 ARM64 包
6. ✅ 验证包完整性
7. ✅ 检查脚本语法
8. ✅ 验证二进制架构

### 完整测试

参考 `docs/APPHUB_CATEGRAF_BUILD_TEST.md` 进行：
- 多架构测试
- 性能测试
- 集成测试
- 生产部署检查清单

## 后续优化建议

### 1. 版本自动更新

可以添加定时任务自动检测 Categraf 新版本：
```bash
# 获取最新版本
LATEST=$(git ls-remote --tags https://github.com/flashcatcloud/categraf.git | \
  grep -oP 'v\d+\.\d+\.\d+$' | sort -V | tail -1)

# 触发构建
docker build --build-arg CATEGRAF_VERSION=${LATEST} ...
```

### 2. 多版本共存

支持同时提供多个 Categraf 版本：
```
/pkgs/categraf/
├── v0.3.90/
│   ├── categraf-v0.3.90-linux-amd64.tar.gz
│   └── categraf-v0.3.90-linux-arm64.tar.gz
├── v0.3.85/
│   ├── categraf-v0.3.85-linux-amd64.tar.gz
│   └── categraf-v0.3.85-linux-arm64.tar.gz
└── latest/ -> v0.3.90/
```

### 3. 添加校验和

为每个包生成 SHA256 校验和：
```bash
sha256sum categraf-*.tar.gz > SHA256SUMS
```

### 4. 支持更多架构

添加 ARM32、RISC-V 等架构支持：
```bash
for arch in amd64 arm64 arm ppc64le; do
    CGO_ENABLED=0 GOOS=linux GOARCH=${arch} go build ...
done
```

## 故障排查

### 构建失败

1. **网络问题** - Git 克隆失败
   ```bash
   # 使用代理
   docker build --build-arg https_proxy=... ...
   ```

2. **Go 模块下载失败**
   ```bash
   # 使用国内代理
   docker build --build-arg GOPROXY=https://goproxy.cn,direct ...
   ```

### 运行时问题

1. **包不存在**
   ```bash
   # 检查容器内文件
   docker exec apphub ls -la /usr/share/nginx/html/pkgs/categraf/
   ```

2. **下载失败**
   ```bash
   # 检查 Nginx 日志
   docker logs apphub | grep categraf
   ```

## 总结

成功将 Categraf 集成到 AppHub：
- ✅ 支持 x86_64 和 ARM64 两种架构
- ✅ 提供完整的安装包（二进制、配置、脚本、文档）
- ✅ 避免 Dockerfile heredoc 问题
- ✅ 构建逻辑模块化、可维护
- ✅ 提供详细的文档和测试脚本
- ✅ 与现有 SLURM/SaltStack 包无冲突

---

**维护**: AI-Infra-Matrix Team  
**日期**: 2025-01-XX  
**相关文档**:
- `docs/APPHUB_CATEGRAF_GUIDE.md` - 使用指南
- `docs/APPHUB_CATEGRAF_BUILD_TEST.md` - 构建测试指南
- `src/apphub/README.md` - AppHub 说明
