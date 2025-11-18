# SLURM 二进制构建方案

## 方案说明

**旧方案问题**：
- 尝试打包为 deb/rpm/apk，复杂且容易失败
- 多阶段构建但没有成功生成包文件
- Dockerfile 包含大量内联脚本，难以调试

**新方案优势**：
- **简化**：只编译二进制文件，不打包
- **通用**：二进制文件可在任何 Linux 发行版使用
- **清晰**：按架构分目录存放（x86_64/arm64）
- **易维护**：脚本独立，容易调试和测试

## 文件结构

```
src/apphub/
├── Dockerfile.slurm-binary          # 新的简化构建文件
├── packages/                         # 存放编译好的二进制文件
│   ├── x86_64/                      # x86_64 架构
│   │   └── slurm/
│   │       ├── bin/                 # 二进制文件
│   │       │   ├── sinfo
│   │       │   ├── squeue
│   │       │   ├── scontrol
│   │       │   ├── sbatch
│   │       │   ├── srun
│   │       │   ├── salloc
│   │       │   ├── scancel
│   │       │   ├── sacct
│   │       │   └── sacctmgr
│   │       ├── lib/                 # 共享库
│   │       │   └── libslurm*.so*
│   │       └── VERSION              # 版本信息
│   └── arm64/                       # arm64 架构
│       └── slurm/
│           └── ...
├── scripts/
│   └── slurm/
│       └── install-binary.sh        # 二进制安装脚本
└── slurm-25.05.4.tar.bz2           # SLURM 源码包
```

## 构建步骤

### 1. 构建 SLURM 二进制（AppHub）

```bash
# 进入项目目录
cd /Users/aresnasa/MyProjects/go/src/github.com/aresnasa/ai-infra-matrix

# 构建 x86_64 架构
docker build --platform linux/amd64 \
  -t ai-infra-apphub:slurm-binary \
  -f src/apphub/Dockerfile.slurm-binary \
  src/apphub

# 可选：构建 arm64 架构
docker build --platform linux/arm64 \
  -t ai-infra-apphub:slurm-binary-arm64 \
  -f src/apphub/Dockerfile.slurm-binary \
  src/apphub
```

### 2. 启动 AppHub

```bash
# 使用新镜像启动
docker run -d --name ai-infra-apphub \
  -p 8081:80 \
  ai-infra-apphub:slurm-binary

# 验证 SLURM 文件可访问
curl http://localhost:8081/packages/x86_64/slurm/VERSION
curl -I http://localhost:8081/packages/x86_64/slurm/bin/sinfo
```

### 3. 构建 Backend（自动从 AppHub 安装）

```bash
# Backend 构建时会自动：
# 1. 检测 AppHub 是否可访问
# 2. 下载安装脚本
# 3. 安装 SLURM 二进制文件到 /usr/local/bin

docker-compose build backend
docker-compose up -d backend

# 验证安装
docker exec ai-infra-backend sinfo --version
docker exec ai-infra-backend which sinfo
```

## 编译原理

### SLURM 源码编译流程

```bash
# 1. 解压源码
tar -xaf slurm-25.05.4.tar.bz2
cd slurm-25.05.4

# 2. 配置（简化版，无 PAM/Munge）
./configure \
    --prefix=/usr/local/slurm \
    --sysconfdir=/etc/slurm \
    --disable-debug \
    --without-pam \
    --without-munge \
    --without-rpath \
    --with-ssl=/usr

# 3. 编译
make -j$(nproc)

# 4. 收集二进制文件
# 编译后的客户端工具位于：
# - src/sinfo/sinfo
# - src/squeue/squeue
# - src/scontrol/scontrol
# - src/sbatch/sbatch
# - src/srun/srun
# - src/salloc/salloc
# - src/scancel/scancel
# - src/sacct/sacct
# - src/sacctmgr/sacctmgr

# 5. 共享库位于：
# - src/common/.libs/libslurm*.so*
```

### 架构检测

```bash
# 在安装脚本中自动检测
ARCH=$(uname -m)
case "${ARCH}" in
    x86_64|amd64) SLURM_ARCH="x86_64" ;;
    aarch64|arm64) SLURM_ARCH="arm64" ;;
esac
```

## 安装方式

### Backend 容器内安装

```dockerfile
# 下载安装脚本
wget http://apphub/packages/install-slurm.sh

# 执行安装（会自动检测架构）
APPHUB_URL=http://apphub ./install-slurm.sh

# 二进制文件被复制到 /usr/local/bin
# 可直接使用 sinfo、squeue 等命令
```

### 手动安装（任何 Linux 系统）

```bash
# 检测架构
ARCH=$(uname -m)
[[ "$ARCH" == "x86_64" ]] && SLURM_ARCH="x86_64"
[[ "$ARCH" == "aarch64" ]] && SLURM_ARCH="arm64"

# 下载二进制文件
wget http://apphub:8081/packages/${SLURM_ARCH}/slurm/bin/sinfo
wget http://apphub:8081/packages/${SLURM_ARCH}/slurm/bin/squeue
# ... 其他工具

# 安装到系统
chmod +x sinfo squeue ...
sudo mv sinfo squeue ... /usr/local/bin/
```

## 优势总结

1. **简单**：不依赖特定发行版的包管理器
2. **快速**：编译一次，所有 Linux 发行版可用
3. **可靠**：避免了打包过程的复杂性和失败率
4. **灵活**：可以选择性安装需要的工具
5. **轻量**：只安装必要的客户端工具，不包含服务端组件

## 故障排查

### AppHub 构建失败

```bash
# 查看构建日志
docker build -f src/apphub/Dockerfile.slurm-binary src/apphub 2>&1 | tee build.log

# 检查是否有 SLURM 源码包
ls -lh src/apphub/slurm-25.05.4.tar.bz2
```

### Backend 安装失败

```bash
# 检查 AppHub 是否运行
docker ps | grep apphub

# 检查 SLURM 文件是否存在
docker exec ai-infra-apphub ls -lh /usr/share/nginx/html/packages/x86_64/slurm/bin/

# 测试网络连通性
docker exec ai-infra-backend wget -O- http://apphub/packages/x86_64/slurm/VERSION
```

### 版本验证

```bash
# 检查 AppHub 中的版本
docker exec ai-infra-apphub cat /usr/share/nginx/html/packages/x86_64/slurm/VERSION

# 检查 Backend 中安装的版本
docker exec ai-infra-backend sinfo --version
```

## 下一步

1. 测试构建新的 AppHub 镜像
2. 验证二进制文件是否正确生成
3. 测试 Backend 安装流程
4. 多架构支持（x86_64 + arm64）
