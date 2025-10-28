# SaltStack Minion 管理功能实现

## 功能概述

新增 SaltStack Minion 的完整生命周期管理功能：
1. **删除 Minion** - 从 Master 删除 Minion key 并卸载客户端
2. **重新安装 Minion** - 使用 AppHub 中的包重新安装
3. **升级 Minion** - 升级到 AppHub 中的最新版本

## API 端点

### 1. 删除 Minion
```
DELETE /api/saltstack/minions/:minionId
```

**功能：**
- 从 Salt Master 删除 Minion key（accepted/pending/rejected）
- 可选：通过 SSH 卸载远程节点上的 SaltStack 客户端
- 清理本地缓存

**请求参数：**
```json
{
  "uninstall": true,  // 是否卸载远程客户端
  "ssh_host": "192.168.1.100",
  "ssh_port": 22,
  "ssh_user": "root",
  "ssh_password": "password",
  "ssh_key": ""  // 或使用 SSH key
}
```

**响应：**
```json
{
  "success": true,
  "message": "Minion deleted successfully",
  "details": {
    "key_deleted": true,
    "client_uninstalled": true
  }
}
```

### 2. 重新安装 Minion
```
POST /api/saltstack/minions/:minionId/reinstall
```

**功能：**
- 卸载旧版本（如果存在）
- 从 AppHub 下载并安装新版本
- 自动配置 Master 地址和 Minion ID
- 启动服务并等待 key 认证

**请求参数：**
```json
{
  "ssh_host": "192.168.1.100",
  "ssh_port": 22,
  "ssh_user": "root",
  "ssh_password": "password",
  "ssh_key": "",
  "master_host": "saltstack",
  "minion_id": "custom-minion-id",  // 可选，默认使用hostname
  "auto_accept_key": true,  // 是否自动接受key
  "apphub_url": "http://apphub:80"  // AppHub地址
}
```

**响应：**
```json
{
  "opId": "salt-reinstall-xxx-timestamp",
  "message": "Reinstall started"
}
```

### 3. 升级 Minion
```
POST /api/saltstack/minions/:minionId/upgrade
```

**功能：**
- 检查当前版本与 AppHub 版本
- 备份配置文件
- 下载并安装新版本
- 恢复配置并重启服务
- 保留 Minion key

**请求参数：**
```json
{
  "ssh_host": "192.168.1.100",
  "ssh_port": 22,
  "ssh_user": "root",
  "ssh_password": "password",
  "ssh_key": "",
  "target_version": "3007.8",  // 可选，默认最新
  "backup_config": true,  // 是否备份配置
  "apphub_url": "http://apphub:80"
}
```

**响应：**
```json
{
  "opId": "salt-upgrade-xxx-timestamp",
  "message": "Upgrade started"
}
```

### 4. 批量操作
```
POST /api/saltstack/minions/batch-operation
```

**功能：**
- 支持批量删除/重新安装/升级
- 并发控制
- 进度跟踪

**请求参数：**
```json
{
  "operation": "upgrade",  // delete, reinstall, upgrade
  "minions": [
    {
      "id": "minion-1",
      "ssh_host": "192.168.1.100",
      "ssh_port": 22,
      "ssh_user": "root",
      "ssh_password": "password"
    }
  ],
  "parallel": 3,  // 并发数
  "auto_accept_key": true
}
```

## 实现细节

### 删除流程
1. 检查 Minion 是否存在
2. 从 Salt Master 删除 key（使用 wheel.key.delete）
3. 如果 uninstall=true：
   - 连接 SSH
   - 检测 OS 类型（DEB/RPM）
   - 执行卸载命令：
     - DEB: `apt-get remove --purge salt-minion -y`
     - RPM: `yum remove salt-minion -y`
   - 清理配置文件和缓存
4. 清理本地缓存

### 重新安装流程
1. 删除旧 Minion（如果存在）
2. 连接 SSH 到目标节点
3. 检测 OS 类型和架构
4. 从 AppHub 下载安装脚本：
   - DEB: `/root/scripts/install-salt-minion-deb.sh`
   - RPM: `/root/scripts/install-salt-minion-rpm.sh`
5. 执行安装脚本（传递 Master 和 Minion ID）
6. 如果 auto_accept_key=true，自动接受 key
7. 验证安装和连接状态

### 升级流程
1. 检查当前版本（salt-minion --version）
2. 比较 AppHub 版本
3. 如果需要升级：
   - 备份配置：`/etc/salt/minion`和`/etc/salt/minion.d/`
   - 停止服务：`systemctl stop salt-minion`
   - 升级包：
     - DEB: `apt-get install --only-upgrade salt-minion`
     - RPM: `yum update salt-minion`
   - 恢复配置
   - 启动服务：`systemctl start salt-minion`
4. 验证升级结果

## 前端页面

### Minion 列表页面增强
在现有 Minion 列表页面添加操作按钮：
- **删除按钮** - 打开删除确认对话框
- **重新安装按钮** - 打开重新安装配置对话框
- **升级按钮** - 打开升级确认对话框
- **批量操作** - 选择多个 Minion 进行批量操作

### 删除确认对话框
```tsx
<Dialog>
  <h3>删除 Minion: {minionId}</h3>
  <Checkbox>
    <label>同时卸载远程客户端</label>
  </Checkbox>
  {uninstall && (
    <SSHCredentialsForm />
  )}
  <Button onClick={handleDelete}>确认删除</Button>
</Dialog>
```

### 重新安装配置对话框
```tsx
<Dialog>
  <h3>重新安装 Minion: {minionId}</h3>
  <SSHCredentialsForm />
  <Input label="Master Host" value={masterHost} />
  <Input label="Minion ID (可选)" value={minionId} />
  <Checkbox label="自动接受 Key" checked={autoAccept} />
  <Button onClick={handleReinstall}>开始安装</Button>
</Dialog>
```

### 升级确认对话框
```tsx
<Dialog>
  <h3>升级 Minion: {minionId}</h3>
  <p>当前版本: {currentVersion}</p>
  <p>目标版本: {targetVersion} (AppHub)</p>
  <SSHCredentialsForm />
  <Checkbox label="备份配置" checked={backupConfig} />
  <Button onClick={handleUpgrade}>开始升级</Button>
</Dialog>
```

### 进度跟踪对话框
```tsx
<Dialog>
  <h3>操作进度: {operation}</h3>
  <ProgressBar value={progress} />
  <List>
    {steps.map(step => (
      <StepItem key={step.name} status={step.status}>
        {step.message}
      </StepItem>
    ))}
  </List>
  <Button onClick={handleClose}>关闭</Button>
</Dialog>
```

## 安全考虑

1. **SSH 凭证安全**
   - 不存储明文密码
   - 使用完即销毁
   - 支持 SSH key 认证

2. **操作权限**
   - 需要管理员权限
   - 记录操作日志
   - 审计追踪

3. **错误处理**
   - 详细的错误信息
   - 回滚机制
   - 超时保护

## 测试场景

1. **删除测试**
   - 删除在线 Minion
   - 删除离线 Minion
   - 删除不存在的 Minion
   - 删除并卸载客户端

2. **重新安装测试**
   - DEB 系统安装
   - RPM 系统安装
   - 网络故障处理
   - SSH 连接失败处理

3. **升级测试**
   - 同版本升级（跳过）
   - 跨版本升级
   - 降级（不支持，提示错误）
   - 配置备份恢复

4. **批量操作测试**
   - 批量删除
   - 批量重新安装
   - 批量升级
   - 部分失败处理
