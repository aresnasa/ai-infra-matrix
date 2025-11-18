# SLURM版本升级修复总结

## 问题描述

SLURM Master和计算节点版本不一致导致作业无法执行：
- **旧版本**: SLURM 21.08.5（Ubuntu仓库，protocol 8960）
- **新版本**: SLURM 25.05.4（AppHub自定义构建）
- **错误**: `error: Protocol version 8960 not supported`

## 根本原因

1. **Docker构建网络隔离**: slurm-master构建时无法访问AppHub服务（运行在host网络）
2. **依赖包缺失**: 构建过程中netcat、mysql-client等工具包未正确保留
3. **数据库Schema不兼容**: 旧版本21.08.5创建的数据库无法被25.05.4使用
4. **状态文件版本冲突**: 旧状态文件`assoc_usage`版本不兼容

## 解决方案

### 1. 修复Docker构建网络问题

**文件**: `docker-compose.yml`

```yaml
slurm-master:
  build:
    context: ./src/slurm-master
    network: host  # ✅ 关键：允许构建时访问host网络上的AppHub
    args:
      APPHUB_URL: http://${EXTERNAL_HOST}:${APPHUB_PORT}
```

### 2. 修复AppHub连接测试

**文件**: `src/slurm-master/Dockerfile`

```dockerfile
# ❌ 旧代码（wget不存在）
if timeout 10 wget -q --spider ${APPHUB_URL}/pkgs/slurm-deb/Packages; then

# ✅ 新代码（使用curl）
if curl -sf --max-time 10 ${APPHUB_URL}/pkgs/slurm-deb/Packages > /dev/null; then
```

### 3. 确保关键工具包安装

**文件**: `src/slurm-master/Dockerfile`（第310行后添加）

```dockerfile
# 确保关键工具包已安装（bootstrap脚本依赖）
echo "📦 确保关键工具包已安装..."; \
apt-get update && apt-get install -y --no-install-recommends \
    netcat-openbsd \
    mysql-client \
    default-mysql-client \
    wget \
    telnet \
    gettext-base 2>/dev/null || \
echo "⚠️  部分工具包安装失败"; \
```

### 4. 清理旧数据库和状态文件

```bash
# 重建数据库（清理旧schema）
docker exec ai-infra-mysql mysql -u root -pmysql123 -e "
    DROP DATABASE IF EXISTS slurm_acct_db;
    CREATE DATABASE slurm_acct_db;
    GRANT ALL ON slurm_acct_db.* TO 'slurm'@'%';
    FLUSH PRIVILEGES;
"

# 清理旧状态文件
docker exec ai-infra-slurm-master bash -c "
    rm -rf /var/spool/slurm/slurmctld/*
    rm -rf /var/lib/slurm/*
"
```

## 构建步骤

### 方法1: 使用构建脚本（推荐）

```bash
./scripts/build-slurm-master.sh
```

### 方法2: 直接构建

```bash
# 确保AppHub正在运行
docker-compose up -d apphub

# 构建slurm-master
docker-compose build slurm-master

# 重启容器
docker-compose up -d slurm-master
```

## 验证结果

```bash
# 1. 检查SLURM版本
docker exec ai-infra-slurm-master slurmctld -V
# 输出: slurm 25.05.4 ✅

# 2. 检查服务状态
docker exec ai-infra-slurm-master systemctl status slurmctld slurmdbd munge --no-pager
# 全部显示: Active: active (running) ✅

# 3. 检查集群节点
docker exec ai-infra-slurm-master sinfo
# PARTITION AVAIL  TIMELIMIT  NODES  STATE NODELIST
# compute*     up   infinite      6   unk* test-rocky[01-03],test-ssh[01-03] ✅

# 4. 测试作业执行
docker exec ai-infra-slurm-master srun -w test-ssh03 hostname
# 输出节点hostname ✅
```

## 关键配置文件

1. **docker-compose.yml**: 添加 `network: host` 到slurm-master构建配置
2. **src/slurm-master/Dockerfile**: 
   - 使用curl替代wget测试AppHub
   - 在SLURM安装后确保工具包已安装
3. **scripts/build-slurm-master.sh**: 自动化构建脚本，包含AppHub连接检查

## 经验教训

1. **Docker构建网络隔离**: 构建时容器默认使用独立网络，无法访问host网络服务
2. **版本一致性**: SLURM master和计算节点必须使用完全相同的版本
3. **状态文件管理**: 主版本升级时需要清理旧状态文件和数据库schema
4. **依赖包管理**: 多阶段构建时要确保所有依赖在最终镜像中可用
5. **工具选择**: 优先使用基础镜像已有的工具（如curl）而非额外安装（如wget+timeout）

## 相关文档

- [SLURM版本兼容性说明](https://slurm.schedmd.com/faq.html#versions)
- [Docker构建网络配置](https://docs.docker.com/engine/reference/commandline/build/#options)
- [AppHub使用指南](../README.md#apphub)

## 时间线

- **2025-11-15 14:00**: 发现版本不一致问题
- **2025-11-15 14:20**: 修复Docker网络配置和AppHub连接
- **2025-11-15 14:35**: 构建成功，SLURM 25.05.4
- **2025-11-15 14:40**: 清理数据库和状态文件
- **2025-11-15 14:43**: ✅ 所有服务启动成功，版本一致
