# SSH密钥统一管理实现方案

## 概述

本文档描述AI基础设施系统中SSH密钥的统一管理方案，解决多容器间的安全通信问题。

## 设计原则

**统一密钥源**：整个系统使用一对统一的SSH密钥
- 位置：`ssh-key/id_rsa` (私钥) + `ssh-key/id_rsa.pub` (公钥)
- 生成：由build.sh自动检测并生成（如不存在）
- 密钥类型：RSA 4096位

**权限分离**：
- Backend（SSH客户端）：持有私钥，用于主动SSH连接
- AppHub/SLURM Master（SSH服务器）：仅持有公钥，接受Backend连接
- 测试容器：保持密码认证（便于测试）

## 技术实现

### 1. 构建时密钥同步（build.sh）

在Docker构建前，build.sh会自动将SSH密钥复制到各组件目录：

```bash
# 位置：build.sh第5750行附近
if [[ "$service" == "backend" ]] || [[ "$service" == "apphub" ]] || [[ "$service" == "slurm-master" ]]; then
    print_info "  → 同步统一SSH密钥到 $service 构建目录..."
    
    local ssh_key_src="$SCRIPT_DIR/ssh-key"
    local ssh_key_dest="$SCRIPT_DIR/$service_path/ssh-key"
    
    # 确保源密钥存在（如不存在则生成）
    if [[ ! -f "$ssh_key_src/id_rsa.pub" ]]; then
        mkdir -p "$ssh_key_src"
        ssh-keygen -t rsa -b 4096 -f "$ssh_key_src/id_rsa" -N "" -C "ai-infra-system@shared"
    fi
    
    # 创建目标目录并复制密钥
    mkdir -p "$ssh_key_dest"
    
    if [[ "$service" == "backend" ]]; then
        # Backend需要私钥和公钥
        cp "$ssh_key_src/id_rsa" "$ssh_key_dest/id_rsa"
        cp "$ssh_key_src/id_rsa.pub" "$ssh_key_dest/id_rsa.pub"
        chmod 600 "$ssh_key_dest/id_rsa"
        chmod 644 "$ssh_key_dest/id_rsa.pub"
    else
        # AppHub和SLURM Master只需要公钥
        cp "$ssh_key_src/id_rsa.pub" "$ssh_key_dest/id_rsa.pub"
        chmod 644 "$ssh_key_dest/id_rsa.pub"
    fi
fi
```

### 2. Docker构建集成

#### Backend Dockerfile
```dockerfile
# 复制统一SSH密钥（私钥）- Backend作为SSH客户端
COPY ssh-key/id_rsa ~/.ssh/id_rsa
COPY ssh-key/id_rsa.pub ~/.ssh/id_rsa.pub

# 设置正确的权限
RUN chmod 700 ~/.ssh && \
    chmod 600 ~/.ssh/id_rsa && \
    chmod 644 ~/.ssh/id_rsa.pub

# 配置SSH客户端（信任所有主机）
RUN echo 'Host *' > ~/.ssh/config && \
    echo '  StrictHostKeyChecking no' >> ~/.ssh/config && \
    echo '  UserKnownHostsFile=/dev/null' >> ~/.ssh/config && \
    chmod 600 ~/.ssh/config
```

#### AppHub Dockerfile
```dockerfile
# 安装SSH服务器
RUN apk add --no-cache openssh-server

# 配置SSH服务器（仅公钥认证）
RUN mkdir -p /root/.ssh /var/run/sshd && \
    chmod 700 /root/.ssh && \
    sed -i 's/#PermitRootLogin.*/PermitRootLogin prohibit-password/' /etc/ssh/sshd_config && \
    sed -i 's/#PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config && \
    sed -i 's/#PubkeyAuthentication.*/PubkeyAuthentication yes/' /etc/ssh/sshd_config && \
    ssh-keygen -A

# 复制统一SSH公钥
COPY ssh-key/id_rsa.pub /root/.ssh/authorized_keys
RUN chmod 600 /root/.ssh/authorized_keys
```

#### AppHub Entrypoint
```bash
# 启动SSH服务器
echo "Starting SSH server..."
/usr/sbin/sshd
echo "✓ SSH server started on port 22"
```

#### SLURM Master Dockerfile
```dockerfile
# 配置SSH服务（仅公钥认证）
RUN mkdir -p /var/run/sshd /root/.ssh && \
    chmod 700 /root/.ssh && \
    sed -i 's/#PermitRootLogin.*/PermitRootLogin prohibit-password/' /etc/ssh/sshd_config && \
    sed -i 's/#PasswordAuthentication.*/PasswordAuthentication no/' /etc/ssh/sshd_config && \
    sed -i 's/#PubkeyAuthentication.*/PubkeyAuthentication yes/' /etc/ssh/sshd_config && \
    systemctl enable ssh

# 复制统一的SSH公钥
COPY ssh-key/id_rsa.pub /root/.ssh/authorized_keys
RUN chmod 600 /root/.ssh/authorized_keys
```

### 3. Git忽略规则

为避免将副本密钥提交到Git：

```gitignore
# .gitignore

# SSH密钥安全: 忽略构建时同步到各组件的密钥副本
# 唯一密钥源存储在: ssh-key/
src/*/ssh-key/
src/backend/ssh-key/
src/apphub/ssh-key/
src/slurm-master/ssh-key/
```

## 密钥生命周期

### 生成
```bash
# 自动生成（由build.sh执行）
ssh-keygen -t rsa -b 4096 -f ssh-key/id_rsa -N "" -C "ai-infra-system@shared"
```

### 分发
构建时自动同步：
- `ssh-key/` → `src/backend/ssh-key/` (私钥+公钥)
- `ssh-key/` → `src/apphub/ssh-key/` (仅公钥)
- `ssh-key/` → `src/slurm-master/ssh-key/` (仅公钥)

### 销毁
重新生成密钥时：
```bash
# 删除旧密钥
rm -f ssh-key/id_rsa ssh-key/id_rsa.pub

# 清理构建副本
rm -rf src/backend/ssh-key/ src/apphub/ssh-key/ src/slurm-master/ssh-key/

# 重新构建
./build.sh backend apphub slurm-master
```

## 安全考虑

1. **最小权限原则**
   - 私钥仅存在于需要主动连接的容器（Backend）
   - 公钥仅分发到需要接受连接的容器（AppHub、SLURM Master）

2. **文件权限**
   - 私钥：600 (仅所有者可读写)
   - 公钥：644 (所有人可读，仅所有者可写)
   - SSH目录：700 (仅所有者可访问)

3. **传输安全**
   - 构建时复制（本地操作，无网络传输）
   - Docker镜像层级隔离

4. **认证配置**
   - SSH服务器禁用密码认证
   - 仅允许公钥认证（测试容器除外）
   - Backend配置为不检查主机密钥（内部网络）

## 使用示例

### Backend连接AppHub
```bash
# 容器内执行
ssh root@ai-infra-apphub "ls -lh /usr/share/nginx/html/scripts/"
```

### Backend连接SLURM Master
```bash
# 容器内执行
ssh root@ai-infra-slurm-master "scontrol show config"
```

### 脚本同步（Backend → AppHub）
```bash
# Backend容器启动时自动执行
/root/scripts/sync-scripts-to-apphub.sh

# 方式1: docker cp（优先，通过docker socket）
docker cp /root/scripts/install-slurm-node.sh ai-infra-apphub:/usr/share/nginx/html/scripts/

# 方式2: scp（备选，通过SSH密钥）
scp /root/scripts/*.sh root@ai-infra-apphub:/usr/share/nginx/html/scripts/
```

## 验证测试

### 1. 检查密钥是否正确复制
```bash
# 检查Backend私钥
docker exec ai-infra-backend ls -lh ~/.ssh/id_rsa

# 检查AppHub公钥
docker exec ai-infra-apphub cat /root/.ssh/authorized_keys

# 检查SLURM Master公钥
docker exec ai-infra-slurm-master cat /root/.ssh/authorized_keys
```

### 2. 测试SSH连接
```bash
# Backend → AppHub
docker exec ai-infra-backend ssh root@ai-infra-apphub "echo 'SSH to AppHub OK'"

# Backend → SLURM Master
docker exec ai-infra-backend ssh root@ai-infra-slurm-master "echo 'SSH to SLURM Master OK'"
```

### 3. 验证SSH服务器配置
```bash
# 检查AppHub的SSH配置
docker exec ai-infra-apphub grep -E "PermitRootLogin|PasswordAuthentication|PubkeyAuthentication" /etc/ssh/sshd_config

# 预期输出：
# PermitRootLogin prohibit-password
# PasswordAuthentication no
# PubkeyAuthentication yes
```

### 4. 测试脚本同步
```bash
# 检查Backend启动日志
docker-compose -f docker-compose.test.yml logs backend | grep "Syncing scripts"

# 检查AppHub的脚本目录
docker exec ai-infra-apphub ls -lh /usr/share/nginx/html/scripts/

# 预期文件：
# - install-slurm-node.sh
# - fix-slurm-plugindir.sh
# - 其他部署脚本
```

## 故障排查

### 问题1: SSH连接被拒绝
```bash
# 检查SSH服务是否运行
docker exec ai-infra-apphub pgrep sshd
docker exec ai-infra-slurm-master systemctl status ssh

# 检查公钥是否正确安装
docker exec ai-infra-apphub cat /root/.ssh/authorized_keys
```

### 问题2: 构建时找不到密钥
```bash
# 检查密钥源是否存在
ls -lh ssh-key/

# 手动生成（如果缺失）
mkdir -p ssh-key
ssh-keygen -t rsa -b 4096 -f ssh-key/id_rsa -N "" -C "ai-infra-system@shared"

# 重新构建
./build.sh backend apphub slurm-master
```

### 问题3: 权限错误
```bash
# 检查密钥权限
docker exec ai-infra-backend ls -l ~/.ssh/

# 修复权限（在容器内执行）
docker exec ai-infra-backend bash -c "chmod 700 ~/.ssh && chmod 600 ~/.ssh/id_rsa"
```

## 相关文件

- **密钥源**: `ssh-key/id_rsa` + `ssh-key/id_rsa.pub`
- **构建脚本**: `build.sh` (第5750行附近)
- **Backend Dockerfile**: `src/backend/Dockerfile`
- **AppHub Dockerfile**: `src/apphub/Dockerfile` + `entrypoint.sh`
- **SLURM Master Dockerfile**: `src/slurm-master/Dockerfile`
- **脚本同步工具**: `src/backend/scripts/sync-scripts-to-apphub.sh`
- **Git忽略规则**: `.gitignore`

## 相关文档

- [SLURM节点安装修复](./SLURM_NODE_INSTALL_FIX.md)
- [PluginDir路径修复](./SLURM_PLUGINDIR_FIX.md)
- [脚本同步机制](./SCRIPT_SYNC_TO_APPHUB.md)

## 更新历史

- 2024-11-13: 初始版本 - 实现统一SSH密钥管理
- 修复Docker构建上下文路径问题
- 添加build.sh自动密钥同步逻辑
- 配置Backend、AppHub、SLURM Master的SSH
