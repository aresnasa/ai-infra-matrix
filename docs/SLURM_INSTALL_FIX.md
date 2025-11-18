# SLURM节点安装问题修复

## 问题描述

通过Web界面添加SLURM节点后，节点上只有salt-minion运行，但没有安装slurm和munge服务。

## 根本原因

1. **Salt命令超时**：原始的`installSlurmPackages`函数使用Salt Master执行安装脚本，但Salt命令可能超时或卡住
2. **脚本未正确执行**：即使脚本被上传到节点，也可能因为权限或路径问题未能执行
3. **仓库未配置**：安装脚本应该先配置AppHub仓库，但这一步可能失败

## 已修复的代码

### 文件: `src/backend/internal/services/slurm_service.go`

**修复前（使用Salt）:**
```go
// 使用 salt 命令将脚本写入节点
saltWriteCmd := exec.CommandContext(ctx, "docker", "exec", saltMaster,
    "salt", nodeName, "cmd.run", writeScriptCmd)
```

**修复后（直接使用docker exec）:**
```go
// 步骤1: 将脚本内容写入节点的临时文件
tmpScriptPath := fmt.Sprintf("/tmp/install-slurm-%s.sh", nodeName)
writeScriptCmd := fmt.Sprintf("cat > %s << 'SCRIPT_EOF'\n%s\nSCRIPT_EOF\nchmod +x %s", 
    tmpScriptPath, string(scriptContent), tmpScriptPath)

cmd := exec.CommandContext(ctx, "docker", "exec", nodeName, "bash", "-c", writeScriptCmd)
cmd.Stdout = logWriter
cmd.Stderr = logWriter
if err := cmd.Run(); err != nil {
    return fmt.Errorf("上传脚本到节点失败: %v", err)
}

// 步骤2: 执行安装脚本
executeScriptCmd := fmt.Sprintf("%s %s compute", tmpScriptPath, apphubURL)
cmd = exec.CommandContext(ctx, "docker", "exec", nodeName, "bash", "-c", executeScriptCmd)
```

## 修复步骤

### 1. 重新构建后端镜像

```bash
cd src/backend
docker build -t ai-infra-backend:v0.3.8 .
```

### 2. 更新docker-compose.yml

确保backend服务使用新版本镜像：

```yaml
backend:
  image: ai-infra-backend:v0.3.8
```

### 3. 重启后端服务

```bash
docker-compose down backend
docker-compose up -d backend
```

### 4. 测试安装（可选）

运行测试脚本验证安装逻辑：

```bash
./test-slurm-install.sh
```

### 5. 通过Web界面添加节点

1. 访问 http://192.168.3.91:8080/slurm-scaling
2. 点击"扩容节点"按钮
3. 输入节点信息：
   - 节点：test-rocky02, test-rocky03, test-ssh02, test-ssh03
   - SSH用户：root
   - SSH认证：密码或密钥
4. 提交扩容任务

### 6. 验证安装结果

检查节点上的服务：

```bash
for node in test-rocky02 test-rocky03 test-ssh02 test-ssh03; do
  echo "=== $node ==="
  docker exec $node bash -c "ps aux | egrep 'slurmd|munged' | grep -v grep"
  docker exec $node bash -c "rpm -qa | grep -E 'slurm|munge' || dpkg -l | grep -E 'slurm|munge'"
done
```

## 手动修复已添加的节点

如果节点已经添加但没有安装SLURM，可以手动执行安装：

```bash
# 对于Rocky Linux节点
docker exec test-rocky02 bash -c "
  # 配置仓库
  cat > /etc/yum.repos.d/slurm-apphub.repo <<EOF
[slurm-apphub]
name=SLURM from AI-Infra AppHub
baseurl=http://ai-infra-apphub/pkgs/slurm-rpm/
enabled=1
gpgcheck=0
priority=1
EOF

  # 安装包
  dnf clean all
  dnf install -y slurm slurmd munge munge-libs
  
  # 创建目录和用户
  mkdir -p /etc/munge /var/lib/munge /var/log/munge /run/munge
  mkdir -p /etc/slurm /var/log/slurm /var/spool/slurmd /var/run/slurm
  
  useradd -r -s /bin/false munge 2>/dev/null || true
  useradd -r -s /bin/false slurm 2>/dev/null || true
  
  # 设置权限
  chown -R root:root /var/log/munge /var/lib/munge
  chown -R munge:munge /etc/munge /run/munge
  chmod 700 /etc/munge /var/lib/munge /var/log/munge
  chmod 755 /run/munge
  
  chown -R slurm:slurm /var/log/slurm /var/spool/slurmd /var/run/slurm
  chmod 755 /var/spool/slurmd /var/log/slurm
"

# 对于Ubuntu节点
docker exec test-ssh02 bash -c "
  # 配置仓库
  echo 'deb [trusted=yes] http://ai-infra-apphub/pkgs/slurm-deb /' > /etc/apt/sources.list.d/slurm-apphub.list
  
  # 安装包
  apt-get update -qq
  apt-get install -y slurm-client slurmd munge
  
  # 创建目录和用户
  mkdir -p /etc/munge /var/lib/munge /var/log/munge /run/munge
  mkdir -p /etc/slurm-llnl /var/log/slurm /var/spool/slurmd /var/run/slurm
  
  # 设置权限
  chown -R root:root /var/log/munge /var/lib/munge
  chown -R munge:munge /etc/munge /run/munge
  chmod 700 /etc/munge /var/lib/munge /var/log/munge
  chmod 755 /run/munge
  
  chown -R slurm:slurm /var/log/slurm /var/spool/slurmd /var/run/slurm
  chmod 755 /var/spool/slurmd /var/log/slurm
"
```

然后复制配置文件并启动服务：

```bash
# 获取配置文件
docker exec ai-infra-slurm-master cat /etc/slurm/slurm.conf > /tmp/slurm.conf
docker exec ai-infra-slurm-master cat /etc/munge/munge.key > /tmp/munge.key

# 对于Rocky Linux节点
for node in test-rocky02 test-rocky03; do
  docker cp /tmp/slurm.conf $node:/etc/slurm/slurm.conf
  docker cp /tmp/munge.key $node:/etc/munge/munge.key
  docker exec $node chown munge:munge /etc/munge/munge.key
  docker exec $node chmod 400 /etc/munge/munge.key
  
  # 启动服务
  docker exec $node bash -c "
    systemctl enable munge slurmd
    systemctl start munge
    sleep 2
    systemctl start slurmd
  "
done

# 对于Ubuntu节点
for node in test-ssh02 test-ssh03; do
  docker cp /tmp/slurm.conf $node:/etc/slurm-llnl/slurm.conf
  docker cp /tmp/munge.key $node:/etc/munge/munge.key
  docker exec $node chown munge:munge /etc/munge/munge.key
  docker exec $node chmod 400 /etc/munge/munge.key
  
  # 启动服务
  docker exec $node bash -c "
    systemctl enable munge slurmd
    systemctl start munge
    sleep 2
    systemctl start slurmd
  "
done

# 清理
rm -f /tmp/slurm.conf /tmp/munge.key
```

## 验证修复

1. **检查服务进程**：
   ```bash
   for node in test-rocky02 test-rocky03 test-ssh02 test-ssh03; do
     echo "=== $node ==="
     docker exec $node ps aux | egrep 'slurmd|munged' | grep -v grep
   done
   ```

2. **检查SLURM集群状态**：
   ```bash
   docker exec ai-infra-slurm-master sinfo -Nel
   ```

3. **测试提交作业**：
   ```bash
   docker exec ai-infra-slurm-master srun -N1 hostname
   ```

## 注意事项

1. **确保AppHub可访问**：节点需要能够访问`http://ai-infra-apphub`
2. **网络连通性**：节点需要与slurm-master网络互通
3. **Munge密钥一致性**：所有节点必须使用相同的munge.key
4. **slurm.conf配置**：确保NodeName配置包含所有节点

## 相关文件

- 安装脚本：`src/backend/scripts/install-slurm-node.sh`
- 安装函数：`src/backend/internal/services/slurm_service.go:installSlurmPackages()`
- 测试脚本：`test-slurm-install.sh`
