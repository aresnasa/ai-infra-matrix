# SLURM UID/GID 统一配置 - 完整部署指南

## 修改摘要

本次修改统一了 SLURM 集群中所有节点（master 和 compute）上的 slurm 和 munge 用户 UID/GID，解决了因 UID/GID 不一致导致的安全违规和作业卡死问题。

### 统一的 UID/GID 标准

| 用户  | UID | GID | 说明 |
|-------|-----|-----|------|
| munge | 998 | 998 | SLURM 认证服务用户 |
| slurm | 999 | 999 | SLURM 主用户 |

## 修改的文件清单

### 1. slurm-master 镜像
**文件**: `src/slurm-master/Dockerfile`

**修改内容**:
```dockerfile
# 在安装任何包之前先创建用户（固定 UID/GID）
RUN groupadd -g 998 munge && useradd -u 998 -g munge -d /var/lib/munge -s /sbin/nologin munge && \
    groupadd -g 999 slurm && useradd -u 999 -g slurm -d /var/lib/slurm -s /bin/bash slurm
```

**说明**: 
- 在安装 munge 和 slurm 包之前显式创建用户
- 避免包管理器自动分配不同的 UID/GID
- 确保 master 节点使用固定的 UID/GID

### 2. compute 节点安装脚本
**文件**: `src/backend/scripts/install-slurm-node.sh`

**修改位置 1 - install_munge() 函数**:
```bash
install_munge() {
    # ... 前面代码 ...
    
    # 统一使用固定的 munge UID/GID
    MUNGE_UID=998
    MUNGE_GID=998
    
    # 确保 munge 用户和组存在，使用固定的 UID/GID
    if ! getent group munge &>/dev/null; then
        groupadd -g $MUNGE_GID munge
    else
        # 检查并修正现有 GID
        EXISTING_MUNGE_GID=$(getent group munge | cut -d: -f3)
        if [ "$EXISTING_MUNGE_GID" != "$MUNGE_GID" ]; then
            groupmod -g $MUNGE_GID munge
            # 更新文件所有权
        fi
    fi
    
    if ! getent passwd munge &>/dev/null; then
        useradd -u $MUNGE_UID -g munge -d /var/lib/munge -s /sbin/nologin munge
    else
        # 检查并修正现有 UID
        EXISTING_MUNGE_UID=$(id -u munge)
        if [ "$EXISTING_MUNGE_UID" != "$MUNGE_UID" ]; then
            usermod -u $MUNGE_UID munge
            # 更新文件所有权
        fi
    fi
}
```

**修改位置 2 - create_slurm_user() 函数**:
```bash
create_slurm_user() {
    # 统一使用 UID/GID 999（与 slurm-master 保持一致）
    SLURM_UID=999
    SLURM_GID=999
    
    # 类似的创建和验证逻辑
}
```

**说明**:
- 将 SLURM_UID/GID 从 1999 改为 999
- 添加 MUNGE_UID/GID 固定值 998
- 自动检测并修正现有的 UID/GID
- 更新文件所有权以匹配新的 UID/GID

### 3. test-containers 镜像
**文件**: 
- `src/test-containers/Dockerfile` (Ubuntu)
- `src/test-containers/Dockerfile.rocky` (Rocky Linux)

**修改内容**:
```dockerfile
# 预先创建 SLURM 相关用户（使用固定 UID/GID 确保一致性）
RUN groupadd -g 998 munge && useradd -u 998 -g munge -d /var/lib/munge -s /sbin/nologin munge && \
    groupadd -g 999 slurm && useradd -u 999 -g slurm -d /var/spool/slurm -s /sbin/nologin slurm
```

**说明**:
- 在测试容器中预先创建用户
- 确保测试环境与生产环境一致
- 避免后续安装 SLURM 时产生 UID/GID 冲突

## 部署流程

### 阶段 1: 停止现有服务并取消卡住的作业

```bash
# 1. 取消卡住的作业
docker exec ai-infra-slurm-master scancel 3

# 2. 停止 slurm-master
docker-compose stop slurm-master
docker-compose rm -f slurm-master
```

### 阶段 2: 重新构建镜像

```bash
# 1. 重新构建 slurm-master
docker-compose build slurm-master

# 或使用专用脚本
./scripts/build-slurm-master.sh

# 2. 重新构建 test-containers（如果需要）
docker-compose build test-ssh
docker-compose build test-rocky
```

### 阶段 3: 修复已部署的 compute 节点

#### 选项 A: 使用自动化脚本（推荐）

```bash
# 修复所有默认节点
./scripts/fix-slurm-uid-gid.sh

# 或指定特定节点
./scripts/fix-slurm-uid-gid.sh test-ssh01 test-ssh02
```

#### 选项 B: 手动修复单个节点

```bash
docker exec test-ssh01 bash << 'EOF'
# 停止服务
systemctl stop slurmd munge

# 修正 munge
MUNGE_UID=$(id -u munge)
if [ "$MUNGE_UID" != "998" ]; then
    usermod -u 998 munge
    groupmod -g 998 munge
    find /etc/munge /var/lib/munge /var/log/munge /run/munge -user $MUNGE_UID -exec chown munge:munge {} \;
fi

# 修正 slurm
SLURM_UID=$(id -u slurm)
if [ "$SLURM_UID" != "999" ]; then
    usermod -u 999 slurm
    groupmod -g 999 slurm
    find /var/spool/slurm /var/log/slurm /run/slurm /etc/slurm -user $SLURM_UID -exec chown slurm:slurm {} \;
fi

# 重启服务
systemctl start munge slurmd

# 验证
id munge
id slurm
EOF
```

### 阶段 4: 重新启动环境

```bash
# 启动 slurm-master
docker-compose up -d slurm-master

# 等待服务就绪
sleep 30

# 检查集群状态
docker exec ai-infra-slurm-master sinfo
```

## 验证步骤

### 1. 验证 UID/GID 一致性

```bash
# 检查 master
echo "=== Master ==="
docker exec ai-infra-slurm-master sh -c "id munge; id slurm"

# 检查所有 compute 节点
for node in test-ssh01 test-ssh02 test-ssh03; do
    echo "=== $node ==="
    docker exec $node sh -c "id munge; id slurm"
done
```

**期望输出**（所有节点一致）:
```
uid=998(munge) gid=998(munge) groups=998(munge)
uid=999(slurm) gid=999(slurm) groups=999(slurm)
```

### 2. 测试 SLURM 作业

```bash
# 提交简单测试作业
docker exec ai-infra-slurm-master srun -N1 hostname

# 提交到特定节点
docker exec ai-infra-slurm-master srun -N1 -w test-ssh01 hostname

# 检查作业队列
docker exec ai-infra-slurm-master squeue

# 查看作业详情
docker exec ai-infra-slurm-master scontrol show job <job_id>
```

**成功标志**:
- 作业快速完成（几秒内）
- 作业状态为 `COMPLETED`
- `Reason=None`（不是 `Reason=Prolog`）
- 没有安全违规错误

### 3. 检查日志

```bash
# Master 日志 - 不应有安全违规错误
docker exec ai-infra-slurm-master tail -50 /var/log/slurm/slurmctld.log | grep -i "security\|error"

# Compute 节点日志
docker exec test-ssh01 tail -50 /var/log/slurm/slurmd.log | grep -i "security\|error"
```

**正常情况**: 不应出现以下错误
- ❌ `Security violation: REQUEST_LAUNCH_PROLOG request from uid 999`
- ❌ `Do you have SlurmUser configured as uid 999?`
- ❌ `Zero Bytes were transmitted or received`

## 新节点部署

对于新部署的节点，使用更新后的脚本会自动应用正确的 UID/GID：

```bash
# 通过 backend API 部署（推荐）
curl -X POST http://localhost:8080/api/slurm/nodes/install \
  -H "Content-Type: application/json" \
  -d '{
    "hostname": "new-node-01",
    "ip": "192.168.1.10",
    "ssh_port": 22,
    "ssh_user": "root",
    "ssh_password": "password"
  }'

# 或手动执行安装脚本
ssh root@new-node-01 'bash -s' < src/backend/scripts/install-slurm-node.sh http://apphub:8080 compute
```

脚本会自动：
1. 创建 munge 用户（UID=998, GID=998）
2. 创建 slurm 用户（UID=999, GID=999）
3. 安装 SLURM 组件
4. 配置正确的文件权限

## 注意事项

### 关键点

1. **顺序很重要**: 必须先创建用户再安装包，否则包管理器会自动创建用户并分配随机 UID/GID

2. **文件权限**: 修改 UID/GID 后必须更新相关文件的所有权，否则服务无法启动

3. **服务重启**: 修改用户后必须重启 munge 和 slurmd 服务

4. **测试验证**: 每次修改后都应该测试作业提交确保功能正常

### 常见问题

**Q: 如果 UID 998/999 已被占用怎么办？**

A: 可以选择其他 UID/GID，但必须：
- 在所有相关配置文件中统一修改
- 确保所有节点使用相同的值
- 选择系统用户范围内的值（通常 100-999）

**Q: 现有的作业数据会丢失吗？**

A: 不会。只要正确更新文件所有权，所有历史数据都会保留。

**Q: 需要清理 SLURM 数据库吗？**

A: 不需要。UID/GID 的修改不影响数据库中的记录。

## 回滚计划

如果修复后出现问题，可以回滚：

```bash
# 1. 停止新的 master
docker-compose stop slurm-master

# 2. 恢复原始镜像
docker-compose up -d slurm-master-backup

# 3. 在 compute 节点上恢复原 UID/GID
# （需要记录原始的 UID/GID 值）
```

## 技术支持

如有问题，请查看：
- 详细文档: `docs/SLURM_UID_GID_FIX.md`
- 日志位置: `/var/log/slurm/`
- GitHub Issues: 提交问题到项目仓库

## 更新日志

- **2025-11-16**: 初始版本，统一 UID/GID 配置
  - slurm: 1999 → 999
  - munge: 自动分配 → 998
  - 更新所有相关脚本和 Dockerfile
