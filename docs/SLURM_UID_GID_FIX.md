# SLURM UID/GID 统一修复

## 问题描述

SLURM 作业卡在 PROLOG 阶段，计算节点日志显示安全违规错误：
```
error: Security violation: REQUEST_LAUNCH_PROLOG request from uid 999
error: Do you have SlurmUser configured as uid 999?
```

**根本原因**：slurm 和 munge 用户在 master 和 compute 节点上的 UID/GID 不一致
- **slurm-master**: slurm UID=999, munge UID=103 (自动分配)
- **compute 节点**: slurm UID=1999, munge UID 不确定

这导致 compute 节点拒绝来自 master (UID 999) 的 RPC 请求。

## 修复方案

### 统一的 UID/GID 配置

所有节点（master 和 compute）现在使用固定的 UID/GID：

| 用户  | UID | GID | 说明 |
|-------|-----|-----|------|
| munge | 998 | 998 | SLURM 认证服务用户 |
| slurm | 999 | 999 | SLURM 主用户 |

### 修改的文件

1. **`src/slurm-master/Dockerfile`**
   - 在安装包之前显式创建 munge 和 slurm 用户
   - 指定固定的 UID=998 (munge), UID=999 (slurm)

2. **`src/backend/scripts/install-slurm-node.sh`**
   - 修改 `create_slurm_user()`: 将 SLURM_UID/GID 从 1999 改为 999
   - 修改 `install_munge()`: 添加固定的 MUNGE_UID=998, MUNGE_GID=998
   - 添加 UID/GID 验证和自动修正逻辑

### 为什么选择这些 UID/GID？

- **999**: 常见的系统用户 UID 范围上限，不会与普通用户冲突
- **998**: 紧邻 999，便于管理，也在系统用户范围内
- 这些值在大多数 Linux 发行版中都可以安全使用

## 部署步骤

### 1. 取消当前卡住的作业

```bash
docker exec ai-infra-slurm-master scancel 3
```

### 2. 重新构建 slurm-master 镜像

```bash
# 停止现有的 slurm-master
docker-compose stop slurm-master
docker-compose rm -f slurm-master

# 重新构建
./scripts/build-slurm-master.sh

# 或使用 docker-compose
docker-compose build slurm-master
```

### 3. 修复已部署的 compute 节点

对于已经部署的计算节点，需要修正 slurm 和 munge 用户的 UID/GID：

```bash
# 在每个 compute 节点上执行（例如 test-ssh01）
docker exec test-ssh01 bash << 'EOF'
# 停止服务
systemctl stop slurmd || pkill -9 slurmd
systemctl stop munge || pkill -9 munged

# 修正 munge 用户
EXISTING_MUNGE_UID=$(id -u munge 2>/dev/null)
if [ "$EXISTING_MUNGE_UID" != "998" ]; then
    echo "Fixing munge UID: $EXISTING_MUNGE_UID -> 998"
    usermod -u 998 munge
    groupmod -g 998 munge
    find /etc/munge /var/lib/munge /var/log/munge /run/munge -user $EXISTING_MUNGE_UID -exec chown munge:munge {} \; 2>/dev/null || true
fi

# 修正 slurm 用户
EXISTING_SLURM_UID=$(id -u slurm 2>/dev/null)
if [ "$EXISTING_SLURM_UID" != "999" ]; then
    echo "Fixing slurm UID: $EXISTING_SLURM_UID -> 999"
    usermod -u 999 slurm
    groupmod -g 999 slurm
    find /var/spool/slurm /var/log/slurm /run/slurm /etc/slurm -user $EXISTING_SLURM_UID -exec chown slurm:slurm {} \; 2>/dev/null || true
fi

# 重启服务
systemctl start munge
systemctl start slurmd

# 验证
echo "munge user: $(id munge)"
echo "slurm user: $(id slurm)"
EOF
```

### 4. 批量修复脚本（适用于多个节点）

```bash
#!/bin/bash
# 文件: scripts/fix-slurm-uid-gid.sh

NODES="test-ssh01 test-ssh02 test-ssh03 test-rocky01 test-rocky02 test-rocky03"

for node in $NODES; do
    echo "=========================================="
    echo "Fixing $node..."
    echo "=========================================="
    
    docker exec $node bash -c '
        systemctl stop slurmd 2>/dev/null || pkill -9 slurmd
        systemctl stop munge 2>/dev/null || pkill -9 munged
        
        # Fix munge
        MUNGE_UID=$(id -u munge 2>/dev/null)
        if [ -n "$MUNGE_UID" ] && [ "$MUNGE_UID" != "998" ]; then
            usermod -u 998 munge
            groupmod -g 998 munge
            find /etc/munge /var/lib/munge /var/log/munge /run/munge -user $MUNGE_UID -exec chown munge:munge {} \; 2>/dev/null || true
            echo "✓ Fixed munge: $MUNGE_UID -> 998"
        fi
        
        # Fix slurm
        SLURM_UID=$(id -u slurm 2>/dev/null)
        if [ -n "$SLURM_UID" ] && [ "$SLURM_UID" != "999" ]; then
            usermod -u 999 slurm
            groupmod -g 999 slurm
            find /var/spool/slurm /var/log/slurm /run/slurm /etc/slurm -user $SLURM_UID -exec chown slurm:slurm {} \; 2>/dev/null || true
            echo "✓ Fixed slurm: $SLURM_UID -> 999"
        fi
        
        systemctl start munge
        systemctl start slurmd
        
        echo "Current UIDs:"
        id munge
        id slurm
    ' 2>&1
    
    echo ""
done
```

### 5. 重新启动整个环境（推荐）

```bash
# 停止所有服务
docker-compose down

# 重新构建并启动
docker-compose up -d

# 等待服务就绪
sleep 30

# 检查节点状态
docker exec ai-infra-slurm-master sinfo
```

## 验证修复

### 1. 检查 UID/GID 一致性

```bash
# 在 master 上检查
docker exec ai-infra-slurm-master sh -c "echo 'Master:'; id slurm; id munge"

# 在 compute 节点上检查
docker exec test-ssh01 sh -c "echo 'test-ssh01:'; id slurm; id munge"
docker exec test-ssh02 sh -c "echo 'test-ssh02:'; id slurm; id munge"
```

**期望输出**（所有节点应该一致）：
```
Master:
uid=999(slurm) gid=999(slurm) groups=999(slurm),998(munge)
uid=998(munge) gid=998(munge) groups=998(munge)
test-ssh01:
uid=999(slurm) gid=999(slurm) groups=999(slurm)
uid=998(munge) gid=998(munge) groups=998(munge)
```

### 2. 测试 SLURM 作业

```bash
# 提交测试作业
docker exec ai-infra-slurm-master srun -N1 hostname

# 检查作业状态
docker exec ai-infra-slurm-master squeue

# 查看作业详情
docker exec ai-infra-slurm-master scontrol show job <job_id>
```

**成功标志**：
- 作业状态为 `RUNNING` 且 `Reason=None`（不再是 `Reason=Prolog`）
- 作业在几秒钟内完成
- 没有 "Security violation" 错误

### 3. 检查日志

```bash
# Master 日志
docker exec ai-infra-slurm-master tail -50 /var/log/slurm/slurmctld.log

# Compute 节点日志
docker exec test-ssh01 tail -50 /var/log/slurm/slurmd.log
```

**不应出现**：
- `Security violation` 错误
- `slurm.conf hash mismatch` 警告
- `Zero Bytes were transmitted or received` 错误

## 预防措施

### 新节点部署

使用更新后的 `install-slurm-node.sh` 脚本部署新节点时，会自动使用正确的 UID/GID。

### 镜像构建

在构建任何涉及 slurm 或 munge 的镜像时，始终在安装包之前显式创建用户：

```dockerfile
# 先创建用户（固定 UID/GID）
RUN groupadd -g 998 munge && useradd -u 998 -g munge -d /var/lib/munge -s /sbin/nologin munge && \
    groupadd -g 999 slurm && useradd -u 999 -g slurm -d /var/lib/slurm -s /bin/bash slurm

# 再安装包
RUN apt-get install -y munge slurm-smd ...
```

### 文档更新

在所有 SLURM 部署文档中明确说明：
- slurm 用户必须使用 UID=999, GID=999
- munge 用户必须使用 UID=998, GID=998

## 常见问题

### Q: 为什么不使用 `-r` 参数创建系统用户？

A: 使用 `-r` 参数时，系统会从可用的系统 UID 范围中自动选择，可能在不同节点上分配不同的值。显式指定 UID/GID 确保一致性。

### Q: 如果 UID 998/999 已被其他用户占用怎么办？

A: 可以选择其他可用的 UID/GID，但必须确保：
1. 所有节点使用相同的值
2. 值在系统用户范围内（通常 100-999）
3. 更新所有相关配置文件

### Q: 现有作业会受影响吗？

A: 修改 UID/GID 后，正在运行的作业可能会失败。建议：
1. 先取消所有运行中的作业
2. 修复 UID/GID
3. 重新提交作业

## 参考资料

- [SLURM Security Guide](https://slurm.schedmd.com/security.html)
- [Munge Authentication](https://github.com/dun/munge)
- [Linux User ID Management](https://www.cyberciti.biz/faq/understanding-etcpasswd-file-format/)

## 更新日志

- **2025-11-16**: 初始版本，修复 UID/GID 不一致问题
