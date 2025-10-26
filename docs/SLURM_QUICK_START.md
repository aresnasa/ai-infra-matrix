# Backend SLURM 客户端 - 快速指南

## 问题解决方案

原始问题：
```bash
docker exec ai-infra-backend "source /etc/profile ;sinfo"
# 错误：OCI runtime exec failed
```

新的正确方式：
```bash
docker exec ai-infra-backend sh -c 'sinfo'
```

## 三步安装 SLURM 客户端

### 步骤 1: 构建 SLURM APK 包

```bash
cd src/apphub
./build-slurm-apk.sh
```

### 步骤 2: 重新构建 Backend

```bash
docker-compose build backend
docker-compose up -d backend
```

### 步骤 3: 验证安装

```bash
docker exec ai-infra-backend sh -c 'sinfo --version'
```

## 架构变更

**之前**: 下载 tar.gz → 解压 → 运行 install.sh → 复杂且易出错

**现在**: AppHub APK 仓库 → apk add slurm-client → 简单可靠

```
AppHub (http://apphub/apks/alpine/)
    ↓ APK 仓库
Backend Dockerfile
    ↓ apk add slurm-client
SLURM 客户端安装完成
```

## 常用命令

```bash
# 查看集群状态
docker exec ai-infra-backend sh -c 'sinfo'

# 查看作业队列
docker exec ai-infra-backend sh -c 'squeue'

# 提交作业
docker exec ai-infra-backend sh -c 'sbatch job.sh'

# 进入容器
docker exec -it ai-infra-backend bash
```

## 重要提示

### ✅ 正确的命令格式

```bash
docker exec ai-infra-backend sh -c 'sinfo'
docker exec ai-infra-backend bash -c 'source /etc/profile && sinfo'
```

### ❌ 错误的命令格式

```bash
# 不要这样用！
docker exec ai-infra-backend "source /etc/profile ;sinfo"
docker exec ai-infra-backend sinfo && squeue
```

## 故障排查

### SLURM 客户端未安装

```bash
# 检查
docker exec ai-infra-backend sh -c 'command -v sinfo'

# 修复
cd src/apphub && ./build-slurm-apk.sh
docker-compose build backend
docker-compose up -d backend
```

### 无法连接 SLURM 控制节点

```bash
# 检查 slurm-master 运行
docker ps | grep slurm-master

# 检查网络
docker exec ai-infra-backend sh -c 'nc -zv slurm-master 6817'

# 查看日志
docker logs ai-infra-slurm-master
```

## 测试脚本

```bash
# 自动测试和修复
./scripts/fix-backend-slurm.sh

# 完整测试
./scripts/test-slurm-client.sh
```

## 更多信息

- 详细指南: [BACKEND_SLURM_APK_GUIDE.md](./BACKEND_SLURM_APK_GUIDE.md)
- 故障排除: [BACKEND_SLURM_FIX.md](./BACKEND_SLURM_FIX.md)
- SLURM 文档: <https://slurm.schedmd.com/>
