# SLURM 节点加入问题修复

## 问题描述

计算节点上的 `slurmd` 服务无法启动，错误信息：

```
error: PluginDir: /usr/lib/slurm: No such file or directory
error: Bad value "/usr/lib/slurm" for PluginDir
fatal: Unable to process configuration file
```

## 根本原因

1. **缺少必要的目录**：节点加入脚本 (`install-slurm-node.sh`) 没有创建 `/usr/lib/slurm` 目录
2. **插件目录未初始化**：即使创建了目录，也可能为空，导致 slurmd 无法找到必需的插件
3. **权限问题**：某些目录的权限设置不正确

## 修复方案

### 1. 修复节点安装脚本

**文件**: `src/backend/scripts/install-slurm-node.sh`

#### 修改 1: 确保创建 PluginDir

在 `create_directories()` 函数中添加 `/usr/lib/slurm` 目录创建：

```bash
# 创建标准目录（包括 PluginDir）
mkdir -p /etc/slurm \
         /usr/lib/slurm \
         /var/spool/slurm/d \
         /var/spool/slurm/ctld \
         /var/log/slurm \
         /run/slurm

# 设置权限
chmod 755 /usr/lib/slurm
chmod 755 /run/slurm
```

#### 修改 2: 改进 ensure_plugin_dir() 函数

确保即使找不到现有插件目录也能创建目标目录：

```bash
ensure_plugin_dir() {
    log_info "Ensuring canonical SLURM plugin directory..."
    local canonical="/usr/lib/slurm"
    
    # 确保 canonical 目录存在
    mkdir -p "$canonical"
    
    # 尝试从常见位置复制插件
    local arch=$(uname -m)
    local candidates=(
        "/usr/lib/slurm-wlm"
        "/usr/lib/${arch}/slurm-wlm"
        "/usr/lib/${arch}/slurm"
        "/usr/lib64/slurm-wlm"
        "/usr/lib64/slurm"
    )
    
    local resolved=""
    for dir in "${candidates[@]}"; do
        if [ -d "$dir" ] && [ -n "$(ls -A "$dir" 2>/dev/null)" ]; then
            resolved="$dir"
            break
        fi
    done
    
    # 如果找到插件源，复制或链接
    if [ -n "$resolved" ] && [ "$resolved" != "$canonical" ]; then
        if [ -z "$(ls -A "$canonical" 2>/dev/null)" ]; then
            if cp -a "$resolved/." "$canonical/"; then
                log_info "📁 Copied plugins to $canonical from $resolved"
            else
                # 如果无法复制，创建符号链接
                rm -rf "$canonical"
                ln -sf "$resolved" "$canonical"
                log_info "Created symlink: $canonical -> $resolved"
            fi
        fi
    fi
}
```

### 2. 手动修复现有节点

对于已经部署的节点，使用修复脚本：

**脚本**: `scripts/fix-slurm-plugin-dir.sh`

```bash
#!/bin/bash
# 在每个计算节点上执行

# 创建必要目录
mkdir -p /usr/lib/slurm \
         /var/spool/slurm/d \
         /var/log/slurm \
         /run/slurm

# 设置权限
chmod 755 /usr/lib/slurm /run/slurm
chown -R slurm:slurm /var/spool/slurm /var/log/slurm /run/slurm

# 复制插件（如果需要）
PLUGIN_SRC=$(find /usr/lib* -type d -name "slurm*" -o -name "*slurm*" | grep -E "(slurm-wlm|x86_64.*slurm)" | head -1)
if [ -n "$PLUGIN_SRC" ] && [ -d "$PLUGIN_SRC" ]; then
    cp -a "$PLUGIN_SRC/." /usr/lib/slurm/
fi

# 重启服务
systemctl restart slurmd
```

### 3. 通过 Backend API 批量修复

使用 Backend 服务的远程执行功能批量修复所有节点：

```bash
# 上传修复脚本到所有节点
curl -X POST http://backend:5000/api/ansible/ssh/upload-script \
  -H "Content-Type: application/json" \
  -d '{
    "nodes": ["test-ssh01", "test-ssh02", "test-ssh03"],
    "script_path": "/tmp/fix-slurm-plugin-dir.sh",
    "script_content": "...(脚本内容)..."
  }'

# 执行修复脚本
curl -X POST http://backend:5000/api/ansible/ssh/execute \
  -H "Content-Type: application/json" \
  -d '{
    "nodes": ["test-ssh01", "test-ssh02", "test-ssh03"],
    "command": "bash /tmp/fix-slurm-plugin-dir.sh"
  }'
```

## 验证修复

### 1. 检查目录结构

```bash
# 在计算节点上执行
ls -la /usr/lib/slurm
ls -la /var/spool/slurm
ls -la /run/slurm
```

预期输出：
```
drwxr-xr-x 2 root  root  4096 Nov 15 23:00 /usr/lib/slurm
drwxr-xr-x 4 slurm slurm 4096 Nov 15 23:00 /var/spool/slurm
drwxr-xr-x 2 slurm slurm 4096 Nov 15 23:00 /run/slurm
```

### 2. 检查 slurmd 服务状态

```bash
systemctl status slurmd
```

预期输出：
```
● slurmd.service - Slurm node daemon
     Loaded: loaded (/lib/systemd/system/slurmd.service; enabled)
     Active: active (running) since Sat 2025-11-15 23:10:00 CST
```

### 3. 检查 SLURM 集群状态

在 slurm-master 上：

```bash
docker exec ai-infra-slurm-master sinfo
```

预期输出（节点状态应该从 `down*` 变为 `idle` 或 `alloc`）：
```
PARTITION AVAIL  TIMELIMIT  NODES  STATE NODELIST
compute*     up   infinite      6   idle test-rocky[01-03],test-ssh[01-03]
```

### 4. 测试作业提交

```bash
docker exec ai-infra-slurm-master srun -w test-ssh03 hostname
```

预期输出：
```
test-ssh03
```

## 相关文件

- **节点安装脚本**: `src/backend/scripts/install-slurm-node.sh`
- **修复脚本**: `scripts/fix-slurm-plugin-dir.sh`
- **slurmd 配置**: `/etc/slurm/slurm.conf`（PluginDir配置）
- **systemd 服务**: `/lib/systemd/system/slurmd.service`

## 常见问题

### Q1: 为什么需要 `/usr/lib/slurm` 目录？

A: `slurm.conf` 中的 `PluginDir` 配置指向这个目录，slurmd 启动时会从这里加载各种插件（如 MPI、认证、任务调度等）。如果目录不存在或为空，slurmd 无法启动。

### Q2: 插件目录为空怎么办？

A: 从 SLURM 包安装的实际插件目录（如 `/usr/lib/x86_64-linux-gnu/slurm-wlm/`）复制文件到 `/usr/lib/slurm/`，或者创建符号链接。

### Q3: 修复后节点仍然是 down 状态？

A: 可能原因：
1. **Munge 密钥不匹配**：确保所有节点使用相同的 munge.key
2. **网络连接问题**：检查节点能否访问 slurm-master:6817
3. **配置文件错误**：检查 `/etc/slurm/slurm.conf` 是否正确

### Q4: 如何更新现有部署的节点？

A: 
1. 通过 Backend API 批量执行修复脚本
2. 或者重新运行节点加入流程（使用修复后的脚本）
3. 或者通过 Ansible/SaltStack 推送修复脚本

## 预防措施

1. **完善安装脚本**：确保所有必需目录在安装时创建
2. **添加健康检查**：在节点加入后自动验证关键目录和文件
3. **文档化目录结构**：明确说明 SLURM 节点需要的目录结构和权限
4. **自动化测试**：添加 E2E 测试验证节点加入流程的完整性

## 时间线

- **2025-11-15 23:09**: 发现 slurmd 启动失败，PluginDir 错误
- **2025-11-15 23:15**: 定位问题：缺少 `/usr/lib/slurm` 目录
- **2025-11-15 23:20**: 修复 `install-slurm-node.sh` 脚本
- **2025-11-15 23:25**: 创建手动修复脚本
- **2025-11-15 23:30**: ✅ 修复完成并验证
