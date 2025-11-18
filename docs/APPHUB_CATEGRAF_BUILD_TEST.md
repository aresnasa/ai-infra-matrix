# AppHub Categraf 构建测试指南

本文档说明如何构建和测试集成了 Categraf 的 AppHub 镜像。

## 构建 AppHub 镜像

### 方法1: 使用现有 build.sh 脚本

```bash
# 进入项目根目录
cd /Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix

# 只构建 AppHub
./build.sh apphub

# 查看构建日志中的 Categraf 构建信息
# 应该看到:
#   🔨 Building Categraf for linux/amd64...
#   🔨 Building Categraf for linux/arm64...
#   📦 Packaging Categraf for amd64...
#   📦 Packaging Categraf for arm64...
```

### 方法2: 手动构建

```bash
# 直接使用 docker build
docker build \
  --build-arg CATEGRAF_VERSION=v0.3.90 \
  -t ai-infra-apphub:latest \
  -f src/apphub/Dockerfile \
  src/apphub
```

### 方法3: 构建特定版本

```bash
# 构建指定版本的 Categraf
docker build \
  --build-arg CATEGRAF_VERSION=v0.3.85 \
  --build-arg SLURM_VERSION=25.05.4 \
  -t ai-infra-apphub:categraf-v0.3.85 \
  -f src/apphub/Dockerfile \
  src/apphub
```

## 验证构建结果

### 1. 启动 AppHub 容器

```bash
# 启动容器
docker run -d \
  --name apphub-test \
  -p 8081:80 \
  ai-infra-apphub:latest

# 等待容器启动
sleep 5

# 检查容器状态
docker ps | grep apphub-test
```

### 2. 验证 Categraf 包存在

```bash
# 列出 Categraf 目录内容
curl http://localhost:8081/pkgs/categraf/

# 应该看到以下文件:
# - categraf-v0.3.90-linux-amd64.tar.gz
# - categraf-v0.3.90-linux-arm64.tar.gz
# - categraf-latest-linux-amd64.tar.gz  (软链接)
# - categraf-latest-linux-arm64.tar.gz  (软链接)

# 检查文件大小（应该在 10-30 MB）
curl -I http://localhost:8081/pkgs/categraf/categraf-latest-linux-amd64.tar.gz | grep Content-Length
```

### 3. 下载并测试 Categraf 包

```bash
# 下载 AMD64 版本
mkdir -p /tmp/categraf-test
cd /tmp/categraf-test
wget http://localhost:8081/pkgs/categraf/categraf-latest-linux-amd64.tar.gz

# 解压
tar xzf categraf-latest-linux-amd64.tar.gz
cd categraf-*-linux-amd64

# 验证目录结构
ls -la
# 应该看到:
#   bin/
#   conf/
#   logs/
#   install.sh
#   uninstall.sh
#   categraf.service
#   README.md

# 验证二进制文件
file bin/categraf
# 应该显示: ELF 64-bit LSB executable, x86-64, statically linked

# 测试运行（查看版本）
./bin/categraf --version
# 或
./bin/categraf --help
```

### 4. 测试安装脚本

```bash
# 查看安装脚本内容
cat install.sh

# 模拟安装（不实际执行，检查语法）
bash -n install.sh
echo $?  # 应该返回 0

# 如果需要实际测试安装（需要 root 权限）
# sudo ./install.sh
# sudo systemctl status categraf
```

## 多架构测试

### 测试 ARM64 包（在 ARM64 系统上）

如果有 ARM64 测试环境：

```bash
# 下载 ARM64 版本
wget http://localhost:8081/pkgs/categraf/categraf-latest-linux-arm64.tar.gz
tar xzf categraf-latest-linux-arm64.tar.gz
cd categraf-*-linux-arm64

# 验证架构
file bin/categraf
# 应该显示: ELF 64-bit LSB executable, ARM aarch64, statically linked

# 测试运行
./bin/categraf --version
```

### 使用 QEMU 模拟测试（在 x86 系统上测试 ARM64）

```bash
# 安装 QEMU
sudo apt-get install qemu-user-static

# 注册 binfmt
docker run --rm --privileged multiarch/qemu-user-static --reset -p yes

# 在 ARM64 容器中测试
docker run --rm -it \
  --platform linux/arm64 \
  -v $(pwd):/test \
  alpine:latest \
  /test/categraf-v0.3.90-linux-arm64/bin/categraf --version
```

## 集成测试

### 完整的端到端测试

```bash
#!/bin/bash
# test-categraf-integration.sh

set -e

echo "=== AppHub Categraf Integration Test ==="

# 1. 启动 AppHub
echo "Step 1: Starting AppHub..."
docker run -d --name apphub-test -p 8081:80 ai-infra-apphub:latest
sleep 5

# 2. 测试包下载
echo "Step 2: Testing package download..."
TMPDIR=$(mktemp -d)
cd $TMPDIR

# 下载 AMD64 版本
wget -q http://localhost:8081/pkgs/categraf/categraf-latest-linux-amd64.tar.gz
if [ $? -ne 0 ]; then
    echo "✗ Failed to download AMD64 package"
    exit 1
fi
echo "✓ AMD64 package downloaded"

# 下载 ARM64 版本
wget -q http://localhost:8081/pkgs/categraf/categraf-latest-linux-arm64.tar.gz
if [ $? -ne 0 ]; then
    echo "✗ Failed to download ARM64 package"
    exit 1
fi
echo "✓ ARM64 package downloaded"

# 3. 验证包内容
echo "Step 3: Validating package contents..."
tar xzf categraf-latest-linux-amd64.tar.gz
cd categraf-*-linux-amd64

# 检查必需文件
for file in bin/categraf install.sh uninstall.sh README.md categraf.service; do
    if [ ! -f "$file" ]; then
        echo "✗ Missing file: $file"
        exit 1
    fi
done
echo "✓ All required files present"

# 检查二进制文件可执行
if [ ! -x bin/categraf ]; then
    echo "✗ Binary is not executable"
    exit 1
fi
echo "✓ Binary is executable"

# 4. 测试脚本语法
echo "Step 4: Testing script syntax..."
bash -n install.sh
bash -n uninstall.sh
echo "✓ Scripts syntax valid"

# 5. 清理
echo "Step 5: Cleanup..."
cd /
rm -rf $TMPDIR
docker stop apphub-test
docker rm apphub-test

echo ""
echo "=== All tests passed! ==="
```

保存并运行：

```bash
chmod +x test-categraf-integration.sh
./test-categraf-integration.sh
```

## 性能测试

### 构建时间测试

```bash
# 记录构建开始时间
START_TIME=$(date +%s)

# 构建镜像
docker build \
  --no-cache \
  -t ai-infra-apphub:perf-test \
  -f src/apphub/Dockerfile \
  src/apphub 2>&1 | tee build.log

# 记录构建结束时间
END_TIME=$(date +%s)
BUILD_TIME=$((END_TIME - START_TIME))

echo "Total build time: ${BUILD_TIME} seconds"

# 分析 Categraf 构建阶段时间
grep "Stage 4:" build.log -A 50 | grep "✓"
```

### 包大小测试

```bash
# 启动容器
docker run -d --name apphub-size-test -p 8081:80 ai-infra-apphub:latest

# 检查各个包的大小
echo "Package sizes:"
curl -sI http://localhost:8081/pkgs/categraf/categraf-latest-linux-amd64.tar.gz | \
  grep -i content-length | \
  awk '{print "AMD64: " $2/1024/1024 " MB"}'

curl -sI http://localhost:8081/pkgs/categraf/categraf-latest-linux-arm64.tar.gz | \
  grep -i content-length | \
  awk '{print "ARM64: " $2/1024/1024 " MB"}'

# 检查镜像总大小
docker images ai-infra-apphub:latest --format "{{.Size}}"

# 清理
docker stop apphub-size-test
docker rm apphub-size-test
```

## 故障排查

### 构建失败

#### 问题1: Categraf 仓库克隆失败

```bash
# 错误信息: fatal: unable to access 'https://github.com/flashcatcloud/categraf.git'

# 解决方案1: 检查网络连接
curl -v https://github.com/flashcatcloud/categraf.git

# 解决方案2: 使用代理
docker build \
  --build-arg https_proxy=http://proxy.example.com:8080 \
  -t ai-infra-apphub:latest \
  -f src/apphub/Dockerfile \
  src/apphub

# 解决方案3: 使用镜像仓库（如果有）
# 修改 Dockerfile 中的 CATEGRAF_REPO ARG
```

#### 问题2: Go 模块下载失败

```bash
# 错误信息: go: downloading module failed

# 解决方案: 使用 Go 代理
docker build \
  --build-arg GOPROXY=https://goproxy.cn,direct \
  -t ai-infra-apphub:latest \
  -f src/apphub/Dockerfile \
  src/apphub
```

### 运行时问题

#### 问题1: 包文件不存在

```bash
# 检查容器内文件
docker exec apphub-test ls -la /usr/share/nginx/html/pkgs/categraf/

# 检查构建日志
docker logs apphub-test | grep -i categraf
```

#### 问题2: 下载的包损坏

```bash
# 验证包完整性
wget http://localhost:8081/pkgs/categraf/categraf-latest-linux-amd64.tar.gz
tar tzf categraf-latest-linux-amd64.tar.gz > /dev/null
if [ $? -eq 0 ]; then
    echo "✓ Package is valid"
else
    echo "✗ Package is corrupted"
fi
```

## 版本管理

### 构建多个版本

```bash
# 构建 v0.3.90
docker build \
  --build-arg CATEGRAF_VERSION=v0.3.90 \
  -t ai-infra-apphub:categraf-v0.3.90 \
  -f src/apphub/Dockerfile \
  src/apphub

# 构建 v0.3.85
docker build \
  --build-arg CATEGRAF_VERSION=v0.3.85 \
  -t ai-infra-apphub:categraf-v0.3.85 \
  -f src/apphub/Dockerfile \
  src/apphub

# 标记 latest
docker tag ai-infra-apphub:categraf-v0.3.90 ai-infra-apphub:latest
```

### 查看已构建版本

```bash
# 列出所有 AppHub 镜像
docker images | grep ai-infra-apphub

# 运行特定版本
docker run -d -p 8081:80 ai-infra-apphub:categraf-v0.3.85
```

## 生产部署检查清单

在将新构建的 AppHub 部署到生产环境前，请完成以下检查：

- [ ] 构建成功完成，无错误
- [ ] AMD64 和 ARM64 包均可正常下载
- [ ] 包可以成功解压，目录结构正确
- [ ] 二进制文件可执行且版本号正确
- [ ] install.sh 和 uninstall.sh 脚本语法正确
- [ ] 配置文件目录包含必要的 toml 文件
- [ ] README.md 包含正确的使用说明
- [ ] 最新版本软链接正确指向
- [ ] 镜像大小在合理范围内（< 2GB）
- [ ] 与现有 SLURM/SaltStack 包不冲突
- [ ] Nginx 目录索引正常显示

## CI/CD 集成建议

### GitHub Actions 示例

```yaml
name: Build AppHub with Categraf

on:
  push:
    branches: [ main ]
    paths:
      - 'src/apphub/**'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v2
      
      - name: Build AppHub
        run: |
          docker build \
            --build-arg CATEGRAF_VERSION=v0.3.90 \
            -t ai-infra-apphub:${{ github.sha }} \
            -f src/apphub/Dockerfile \
            src/apphub
      
      - name: Test Categraf packages
        run: |
          docker run -d --name test-apphub -p 8081:80 ai-infra-apphub:${{ github.sha }}
          sleep 5
          
          # Test AMD64 package
          wget http://localhost:8081/pkgs/categraf/categraf-latest-linux-amd64.tar.gz
          tar tzf categraf-latest-linux-amd64.tar.gz
          
          # Test ARM64 package
          wget http://localhost:8081/pkgs/categraf/categraf-latest-linux-arm64.tar.gz
          tar tzf categraf-latest-linux-arm64.tar.gz
          
          docker stop test-apphub
      
      - name: Push to registry
        if: success()
        run: |
          # Push to your registry
          docker tag ai-infra-apphub:${{ github.sha }} your-registry/ai-infra-apphub:latest
          docker push your-registry/ai-infra-apphub:latest
```

## 参考资源

- **Categraf 构建文档**: <https://github.com/flashcatcloud/categraf#build>
- **Go 交叉编译**: <https://go.dev/doc/install/source#environment>
- **Docker 多阶段构建**: <https://docs.docker.com/build/building/multi-stage/>

---

**维护**: AI-Infra-Matrix Team  
**更新**: 2025-01-XX
