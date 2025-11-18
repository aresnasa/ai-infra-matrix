# Rocky Linux SLURM RPM 包修复报告

## 日期
2025-11-11

## 问题描述

Rocky Linux 节点在安装 SLURM 后无法正常工作，节点状态显示为 `unk*` 或 `down*`，无法接受任务。

## 根本原因分析

通过逐步诊断，发现以下几个关键问题：

### 1. cgroup v2 插件缺失 ⚠️ **核心问题**

**现象：**
```
[2025-11-11T16:40:02.627] error: cgroup/v2: init: Failed to initialize cgroup v2
[2025-11-11T16:40:02.627] error: Couldn't load specified plugin name for task/cgroup
```

**原因：**
- Rocky RPM 包只包含 `cgroup_v1.so`，缺少 `cgroup_v2.so`
- Rocky Linux 9.3 默认使用 cgroup v2
- AppHub 的 RPM 构建过程（`rpmbuild -ta --nodeps`）未包含 cgroup v2 支持

**验证：**
```bash
# Rocky 节点（有问题）
$ ls /usr/lib64/slurm/cgroup*.so
/usr/lib64/slurm/cgroup_v1.so

# Ubuntu 节点（正常）
$ ls /usr/lib/aarch64-linux-gnu/slurm-wlm/cgroup*.so
/usr/lib/aarch64-linux-gnu/slurm-wlm/cgroup_v1.so
/usr/lib/aarch64-linux-gnu/slurm-wlm/cgroup_v2.so  # ✓ 包含 v2
```

### 2. slurm UID 不一致

**现象：**
```
error: cred/munge: Unexpected uid (999) != Slurm uid (996)
```

**原因：**
- Master 容器：slurm UID=999
- Rocky 节点：slurm UID=996
- Ubuntu 节点：slurm UID=998
- Munge 认证要求所有节点的 slurm UID 必须一致

### 3. slurm.conf 路径差异

**问题：**
- Rocky RPM 包将配置文件安装到 `/usr/etc/slurm/slurm.conf`
- 标准路径是 `/etc/slurm/slurm.conf`
- 导致配置文件不一致警告

### 4. Munge 目录权限配置错误

**问题：**
```
munged: Error: Logfile is insecure: invalid ownership of "/var/log/munge"
```

**原因：**
- 修改 slurm UID 后，相关目录权限未正确更新
- `/var/log/munge` 和 `/var/lib/munge` 需要 root 所有
- `/etc/munge` 和 `/run/munge` 需要 munge 用户所有
- `munge.key` 必须由 munge 用户所有

## 解决方案

### 1. 修复 cgroup v2 插件缺失

**方法 A：从 Ubuntu DEB 包提取**（临时方案）
```bash
# 在 Ubuntu 节点上提取
docker cp test-ssh01:/usr/lib/aarch64-linux-gnu/slurm-wlm/cgroup_v2.so /tmp/

# 复制到 Rocky 节点
docker cp /tmp/cgroup_v2.so test-rocky01:/usr/lib64/slurm/

# 安装依赖
docker exec test-rocky01 dnf install -y dbus-libs
```

**方法 B：从 AppHub 提供**（推荐方案）
```bash
# 在部署脚本中添加
if [ ! -f /usr/lib64/slurm/cgroup_v2.so ]; then
    wget -q -O /tmp/cgroup_v2.so "${APPHUB_URL}/pkgs/slurm-plugins/cgroup_v2.so"
    cp /tmp/cgroup_v2.so /usr/lib64/slurm/
    chmod 755 /usr/lib64/slurm/cgroup_v2.so
fi
```

### 2. 统一 slurm UID

**在所有节点上执行：**
```bash
# 统一使用 UID/GID 1999
SLURM_UID=1999
SLURM_GID=1999

# 修改或创建 slurm 用户
groupadd -g $SLURM_GID slurm 2>/dev/null || groupmod -g $SLURM_GID slurm
useradd -u $SLURM_UID -g slurm -d /var/spool/slurm -s /sbin/nologin slurm 2>/dev/null || usermod -u $SLURM_UID slurm

# 更新文件所有权
chown -R slurm:slurm /var/spool/slurm /var/log/slurm /run/slurm
```

### 3. 处理配置文件路径差异

**Rocky 节点：**
```bash
# 创建符号链接
if [ -d /usr/etc/slurm ] && [ ! -L /etc/slurm ]; then
    ln -sf /usr/etc/slurm /etc/slurm
fi

# 或者直接使用 /usr/etc/slurm
SLURM_CONF_PATH="/usr/etc/slurm/slurm.conf"
```

### 4. 修复 Munge 权限

**正确的权限配置：**
```bash
# 创建目录
mkdir -p /etc/munge /var/lib/munge /var/log/munge /run/munge

# 设置所有权
chown -R root:root /var/log/munge /var/lib/munge
chown -R munge:munge /etc/munge /run/munge

# 设置权限
chmod 700 /etc/munge /var/lib/munge /var/log/munge
chmod 755 /run/munge
chmod 400 /etc/munge/munge.key

# 确保 munge key 由 munge 用户所有
chown munge:munge /etc/munge/munge.key
```

### 5. 同步配置文件

**从 master 同步到所有节点：**
```bash
# 获取 master 的配置
docker cp ai-infra-slurm-master:/etc/slurm/slurm.conf /tmp/slurm.conf

# 同步到 Rocky 节点
for node in test-rocky01 test-rocky02 test-rocky03; do
    docker cp /tmp/slurm.conf $node:/usr/etc/slurm/slurm.conf
done

# 同步到 Ubuntu 节点
for node in test-ssh01 test-ssh02 test-ssh03; do
    docker cp /tmp/slurm.conf $node:/etc/slurm/slurm.conf
done
```

### 6. 重启服务并激活节点

**顺序执行：**
```bash
# 1. 重启 master 的 munge 和 slurmctld
docker exec ai-infra-slurm-master bash -c "
    systemctl restart munge
    pkill -9 slurmctld
    /usr/sbin/slurmctld
"

# 2. 重启所有节点的 slurmd
for node in test-*; do
    docker exec $node bash -c "
        pkill -9 slurmd
        /usr/sbin/slurmd
    "
done

# 3. 等待节点注册
sleep 10

# 4. 检查状态
docker exec ai-infra-slurm-master sinfo
```

## 修复后的脚本更新

### 1. `install-slurm-node.sh` 更新内容

**添加 cgroup v2 支持：**
```bash
# 安装依赖（cgroup v2 需要 dbus-libs）
$PKG_MANAGER install -y dbus-libs

# 检查并下载 cgroup_v2.so
if [ ! -f /usr/lib64/slurm/cgroup_v2.so ]; then
    wget -q -O /tmp/cgroup_v2.so "${APPHUB_URL}/pkgs/slurm-plugins/cgroup_v2.so"
    cp /tmp/cgroup_v2.so /usr/lib64/slurm/
    chmod 755 /usr/lib64/slurm/cgroup_v2.so
fi
```

**统一 slurm UID：**
```bash
# 统一使用 UID/GID 1999
SLURM_UID=1999
SLURM_GID=1999

groupadd -g $SLURM_GID slurm 2>/dev/null || groupmod -g $SLURM_GID slurm
useradd -u $SLURM_UID -g slurm slurm 2>/dev/null || usermod -u $SLURM_UID slurm
```

**正确的 munge 权限：**
```bash
mkdir -p /etc/munge /var/lib/munge /var/log/munge /run/munge
chown -R root:root /var/log/munge /var/lib/munge
chown -R munge:munge /etc/munge /run/munge
chmod 700 /etc/munge /var/lib/munge /var/log/munge
chmod 755 /run/munge
```

### 2. `configure-slurm-node.sh` 更新内容

**处理配置文件路径：**
```bash
if [ "$OS_TYPE" = "rpm" ]; then
    SLURM_CONF_PATH="/usr/etc/slurm/slurm.conf"
    mkdir -p /usr/etc/slurm
    ln -sf /usr/etc/slurm /etc/slurm
else
    SLURM_CONF_PATH="/etc/slurm/slurm.conf"
fi
```

**验证 munge 和配置：**
```bash
# 显示 MD5 校验
KEY_MD5=$(md5sum /etc/munge/munge.key | cut -d' ' -f1)
log_info "Munge key deployed (MD5: $KEY_MD5)"

CONF_MD5=$(md5sum "$SLURM_CONF_PATH" | cut -d' ' -f1)
log_info "slurm.conf deployed (MD5: $CONF_MD5)"
```

### 3. 新增 `resume-slurm-nodes.sh`

自动检测并恢复 DOWN 状态的节点：
```bash
#!/bin/bash
# resume-slurm-nodes.sh - Resume DOWN nodes after installation

# 获取所有 DOWN 节点
DOWN_NODES=$(sinfo -h -o "%N %T" | grep -E "down|drain" | awk '{print $1}')

# 恢复每个节点
for node in $DOWN_NODES; do
    scontrol update NodeName=$node State=RESUME
    sleep 2
done

# 验证
sinfo
```

### 4. AppHub Dockerfile 更新

**提取并提供 cgroup_v2.so：**
```dockerfile
# 创建插件目录
RUN mkdir -p /usr/share/nginx/html/pkgs/slurm-plugins

# 从 DEB 包提取 cgroup_v2.so
RUN cd /usr/share/nginx/html/pkgs/slurm-deb; \
    DEB_FILE=$(ls -1 slurm-smd-slurmd_*.deb | head -1); \
    dpkg-deb -x "$DEB_FILE" /tmp/slurm-extract; \
    find /tmp/slurm-extract -name "cgroup_v2.so" \
        -exec cp {} /usr/share/nginx/html/pkgs/slurm-plugins/ \;
```

## 验证步骤

### 1. 验证插件安装
```bash
$ ls -l /usr/lib64/slurm/cgroup*.so
-rwxr-xr-x 1 root root 18480 Nov 11 16:42 cgroup_v1.so
-rwxr-xr-x 1 root root 22576 Nov 11 16:42 cgroup_v2.so  # ✓
```

### 2. 验证 UID 一致性
```bash
$ for node in ai-infra-slurm-master test-*; do
    echo "$node: $(docker exec $node id -u slurm)"
done
ai-infra-slurm-master: 1999
test-rocky01: 1999
test-rocky02: 1999
test-rocky03: 1999
test-ssh01: 1999
test-ssh02: 1999
test-ssh03: 1999
```

### 3. 验证 munge key 一致性
```bash
$ for node in ai-infra-slurm-master test-*; do
    echo "$node: $(docker exec $node md5sum /etc/munge/munge.key | cut -d' ' -f1)"
done
# 所有节点应该输出相同的 MD5
```

### 4. 验证集群状态
```bash
$ docker exec ai-infra-slurm-master sinfo
PARTITION AVAIL  TIMELIMIT  NODES  STATE NODELIST
compute*     up   infinite      6   idle test-rocky[01-03],test-ssh[01-03]
```

### 5. 测试任务提交
```bash
# 测试 Rocky 节点
$ docker exec ai-infra-slurm-master srun -w test-rocky01 hostname
test-rocky01

$ docker exec ai-infra-slurm-master srun -w test-rocky02 hostname
test-rocky02

# 并行测试
$ docker exec ai-infra-slurm-master srun -N 6 hostname | sort
test-rocky01
test-rocky02
test-rocky03
test-ssh01
test-ssh02
test-ssh03
```

## 最终结果

✅ **所有 6 个节点（3个 Rocky + 3个 Ubuntu）都正常工作**

```
PARTITION AVAIL  TIMELIMIT  NODES  STATE NODELIST
compute*     up   infinite      6   idle test-rocky[01-03],test-ssh[01-03]
```

**关键指标：**
- 节点状态：IDLE ✓
- cgroup v2 支持：已安装 ✓
- slurm UID：1999（统一）✓
- munge 认证：正常 ✓
- 配置文件：一致 ✓
- 任务提交：成功 ✓

## 长期优化建议

### 1. 修复 AppHub RPM 构建

**问题：**
当前使用 `rpmbuild -ta --nodeps` 构建，跳过了依赖检查，导致缺少 cgroup v2 支持。

**建议：**
```bash
# 在构建 RPM 前安装必要的构建依赖
dnf install -y dbus-devel systemd-devel

# 使用正确的构建选项
rpmbuild -ta slurm-*.tar.bz2
```

### 2. 标准化配置路径

**建议：**
在 RPM spec 文件中配置使用标准路径 `/etc/slurm`，而不是 `/usr/etc/slurm`。

### 3. 预配置 UID

**建议：**
在 Docker 镜像或初始化脚本中预先创建 slurm 用户（UID 1999），避免运行时修改。

### 4. 自动化测试

**建议：**
添加集成测试验证：
- cgroup v2 插件存在性
- UID 一致性
- munge 认证
- 节点状态
- 任务提交成功

## 相关文件

- `src/backend/scripts/install-slurm-node.sh` - 节点安装脚本（已更新）
- `src/backend/scripts/configure-slurm-node.sh` - 节点配置脚本（已更新）
- `src/backend/scripts/resume-slurm-nodes.sh` - 节点恢复脚本（新增）
- `src/apphub/Dockerfile` - AppHub 构建文件（已更新）

## 参考资料

- SLURM cgroup v2 文档：https://slurm.schedmd.com/cgroup_v2.html
- Munge 认证配置：https://github.com/dun/munge/wiki
- Rocky Linux cgroup v2：https://docs.rockylinux.org/

## 作者

AI Assistant（GitHub Copilot）

## 修订历史

- 2025-11-11：初始版本，记录完整修复过程
