# SLURM 节点部署快速指南

## 问题现象

Rocky Linux 节点安装 SLURM 后状态为 `down*` 或 `unk*`，无法接受任务。

## 快速修复

### 1. 确保 cgroup v2 插件存在

```bash
# 检查插件
ls /usr/lib64/slurm/cgroup_v2.so

# 如果不存在，从 AppHub 下载
wget http://ai-infra-apphub:8080/pkgs/slurm-plugins/cgroup_v2.so -O /usr/lib64/slurm/cgroup_v2.so
chmod 755 /usr/lib64/slurm/cgroup_v2.so

# 安装依赖
dnf install -y dbus-libs
```

### 2. 统一 slurm UID（所有节点）

```bash
# 统一使用 UID 1999
usermod -u 1999 slurm
groupmod -g 1999 slurm
chown -R slurm:slurm /var/spool/slurm /var/log/slurm /run/slurm
```

### 3. 配置 Munge 权限

```bash
# 创建目录
mkdir -p /etc/munge /var/lib/munge /var/log/munge /run/munge

# 设置权限
chown -R root:root /var/log/munge /var/lib/munge
chown -R munge:munge /etc/munge /run/munge
chmod 700 /etc/munge /var/lib/munge /var/log/munge
chmod 755 /run/munge

# munge key 权限
chown munge:munge /etc/munge/munge.key
chmod 400 /etc/munge/munge.key
```

### 4. 同步配置文件

```bash
# Rocky: 使用 /usr/etc/slurm/slurm.conf
# Ubuntu: 使用 /etc/slurm/slurm.conf

# 从 master 复制配置
scp master:/etc/slurm/slurm.conf /usr/etc/slurm/slurm.conf  # Rocky
scp master:/etc/slurm/slurm.conf /etc/slurm/slurm.conf      # Ubuntu

# Rocky: 创建符号链接
ln -sf /usr/etc/slurm /etc/slurm
```

### 5. 重启服务

```bash
# 重启 munge
systemctl restart munge

# 重启 slurmd
pkill -9 slurmd
/usr/sbin/slurmd
```

### 6. 激活节点（在 master 上执行）

```bash
# 恢复 DOWN 状态的节点
scontrol update NodeName=<node_name> State=RESUME

# 或使用脚本批量恢复
./backend/scripts/resume-slurm-nodes.sh
```

## 验证

```bash
# 检查集群状态
sinfo

# 应该显示：
# PARTITION AVAIL  TIMELIMIT  NODES  STATE NODELIST
# compute*     up   infinite      6   idle test-rocky[01-03],test-ssh[01-03]

# 测试任务提交
srun -w test-rocky01 hostname
```

## 自动化部署

### 使用更新后的脚本

```bash
# 1. 安装 SLURM（包含所有修复）
./backend/scripts/install-slurm-node.sh http://ai-infra-apphub:8080 compute

# 2. 配置节点（部署 munge key 和 slurm.conf）
./backend/scripts/configure-slurm-node.sh \
    ai-infra-slurm-master \
    "$(base64 < /etc/munge/munge.key)" \
    "$(base64 < /etc/slurm/slurm.conf)"

# 3. 在 master 上恢复节点
./backend/scripts/resume-slurm-nodes.sh
```

## Web 界面部署

访问 http://192.168.3.91:8080/slurm 部署节点，系统会：

1. ✅ 自动安装 SLURM 包（包含 cgroup v2 修复）
2. ✅ 自动统一 slurm UID 为 1999
3. ✅ 自动配置 munge 权限
4. ✅ 自动同步配置文件到正确路径
5. ✅ 自动启动服务
6. ⚠️ **需要手动执行**：在 master 上恢复节点状态

### 激活新部署的节点

部署完成后，在 master 容器中执行：

```bash
# 方法 1：手动激活
docker exec ai-infra-slurm-master scontrol update NodeName=<node_name> State=RESUME

# 方法 2：批量激活（推荐）
docker exec ai-infra-slurm-master bash -c "
    for node in \$(sinfo -h -o '%N %T' | grep down | awk '{print \$1}'); do
        scontrol update NodeName=\$node State=RESUME
    done
"

# 方法 3：使用脚本
docker cp src/backend/scripts/resume-slurm-nodes.sh ai-infra-slurm-master:/tmp/
docker exec ai-infra-slurm-master bash /tmp/resume-slurm-nodes.sh
```

## 常见问题

### Q1: 节点仍然是 DOWN 状态

**检查：**
```bash
# 1. slurmd 是否运行
pgrep -a slurmd

# 2. munge 是否正常
systemctl status munge
munge -n | unmunge

# 3. 查看日志
tail -f /var/log/slurm/slurmd.log
```

**解决：**
```bash
# 重启 slurmd
pkill -9 slurmd
/usr/sbin/slurmd

# 在 master 上重新激活
scontrol update NodeName=<node> State=RESUME
```

### Q2: munge 认证失败

**检查：**
```bash
# munge key MD5 是否一致
md5sum /etc/munge/munge.key

# slurm UID 是否一致
id slurm
```

**解决：**
```bash
# 重新部署 munge key
scp master:/etc/munge/munge.key /etc/munge/
chown munge:munge /etc/munge/munge.key
chmod 400 /etc/munge/munge.key
systemctl restart munge
```

### Q3: Rocky 节点缺少 cgroup_v2.so

**解决：**
```bash
# 从 AppHub 下载
wget http://ai-infra-apphub:8080/pkgs/slurm-plugins/cgroup_v2.so \
    -O /usr/lib64/slurm/cgroup_v2.so

# 或从 Ubuntu 节点复制
scp ubuntu-node:/usr/lib/aarch64-linux-gnu/slurm-wlm/cgroup_v2.so \
    /usr/lib64/slurm/

# 设置权限
chmod 755 /usr/lib64/slurm/cgroup_v2.so

# 安装依赖
dnf install -y dbus-libs

# 重启 slurmd
pkill -9 slurmd && /usr/sbin/slurmd
```

## 关键配置检查清单

- [ ] cgroup_v2.so 插件存在（Rocky 节点）
- [ ] slurm UID = 1999（所有节点一致）
- [ ] munge key MD5 一致（所有节点）
- [ ] munge 目录权限正确
- [ ] slurm.conf 同步（MD5 一致）
- [ ] Rocky: /usr/etc/slurm/slurm.conf 存在
- [ ] Ubuntu: /etc/slurm/slurm.conf 存在
- [ ] munge 服务运行中
- [ ] slurmd 进程运行中
- [ ] 节点状态已 RESUME

## 详细文档

参见：`docs/SLURM_ROCKY_RPM_FIX.md`
