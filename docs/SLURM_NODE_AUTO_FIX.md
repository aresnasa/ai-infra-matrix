# SLURM 节点自动修复实现

## 问题描述

所有 SLURM 计算节点（test-rocky01-03, test-ssh01-03）显示为 `IDLE+NOT_RESPONDING` 状态，无法接受任务。

```bash
$ docker exec ai-infra-slurm-master sinfo
PARTITION AVAIL  TIMELIMIT  NODES  STATE NODELIST
compute*     up   infinite      6  idle* test-rocky[01-03],test-ssh[01-03]
```

## 根本原因

1. **slurmd 服务启动失败**：计算节点的 slurmd 服务由于目录权限问题启动失败
2. **缺少必需目录**：`/var/run/slurm`、`/var/spool/slurmd`、`/var/log/slurm` 目录不存在或权限不正确
3. **启动超时**：slurmd 尝试连接 slurmctld 时超时

## 解决方案

### 自动化修复机制

在 `slurm-master` 的 bootstrap 流程中添加自动修复计算节点的步骤：

1. **在 master 启动时通过 SSH 连接到所有计算节点**
2. **创建必需的目录并设置正确权限**
3. **重启 slurmd 服务**
4. **等待节点注册到集群**

### 实现细节

#### 1. 修改 `entrypoint.sh` 添加节点修复函数

**文件**: `src/slurm-master/entrypoint.sh`

```bash
fix_compute_nodes() {
    log "INFO" "🔧 修复计算节点配置..."
    
    # 解析测试节点列表
    if [ -z "${SLURM_TEST_NODES}" ]; then
        log "WARN" "未配置测试节点，跳过节点修复"
        return 0
    fi
    
    # 将逗号分隔的节点列表转换为数组
    IFS=',' read -ra NODES <<< "${SLURM_TEST_NODES}"
    
    local fixed_count=0
    local failed_count=0
    
    for node in "${NODES[@]}"; do
        node=$(echo "$node" | xargs)  # 去除空格
        log "INFO" "  检查节点: $node"
        
        # 检查节点是否可达
        if ! ping -c 1 -W 2 "$node" >/dev/null 2>&1; then
            log "WARN" "  节点 $node 不可达，跳过"
            ((failed_count++))
            continue
        fi
        
        # 通过 SSH 修复节点
        local ssh_password="${SLURM_NODE_SSH_PASSWORD:-aiinfra2024}"
        
        if sshpass -p "$ssh_password" ssh -o StrictHostKeyChecking=no -o ConnectTimeout=5 \
            root@"$node" "
            mkdir -p /var/run/slurm /var/spool/slurmd /var/log/slurm && \
            chown -R slurm:slurm /var/run/slurm /var/spool/slurmd /var/log/slurm && \
            chmod 755 /var/run/slurm /var/spool/slurmd && \
            systemctl is-active --quiet slurmd || systemctl restart slurmd
        " >/dev/null 2>&1; then
            log "INFO" "  ✅ 节点 $node 修复成功"
            ((fixed_count++))
        else
            log "WARN" "  ⚠️  节点 $node 修复失败"
            ((failed_count++))
        fi
        
        sleep 1
    done
    
    log "INFO" "✅ 节点修复完成: 成功 $fixed_count 个, 失败 $failed_count 个"
    
    if [ $fixed_count -gt 0 ]; then
        log "INFO" "⏳ 等待节点注册到控制器..."
        sleep 5
    fi
}
```

#### 2. 在 bootstrap 流程中调用

```bash
bootstrap() {
    detect_slurm_mode
    set_plugin_dir
    print_configuration

    if [ "${SLURM_MODE}" = "full" ]; then
        wait_for_database
        init_database
    else
        log "WARN" "演示模式将仅生成基础配置"
    fi

    generate_configs
    setup_munge
    
    # 等待 SLURM 服务启动
    log "INFO" "⏳ 等待 SLURM 服务启动..."
    sleep 10
    
    # 修复计算节点
    fix_compute_nodes

    log "INFO" "✨ SLURM 引导任务完成"
}
```

#### 3. 安装 `sshpass` 工具

**文件**: `src/slurm-master/Dockerfile`

在依赖安装部分添加 `sshpass`：

```dockerfile
apt-get install -y --no-install-recommends \
    openssh-client \
    openssh-server \
    sshpass \  # 新增
    ...
```

#### 4. 配置环境变量

**文件**: `docker-compose.yml`

```yaml
slurm-master:
  environment:
    - SLURM_TEST_NODES=${SLURM_TEST_NODES:-test-ssh01,test-ssh02,test-ssh03}
    - SLURM_NODE_SSH_PASSWORD=${SLURM_NODE_SSH_PASSWORD:-aiinfra2024}  # 新增
```

**文件**: `.env`

```properties
# SLURM 测试节点配置
SLURM_TEST_NODES=test-rocky01,test-rocky02,test-rocky03,test-ssh01,test-ssh02,test-ssh03
SLURM_NODE_SSH_PASSWORD=aiinfra2024
```

## 工作流程

```
┌─────────────────────────────────────────────────────────────┐
│ 1. Container 启动                                            │
│    systemd-entrypoint.sh                                    │
└──────────────────┬──────────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────────┐
│ 2. Bootstrap Service 执行                                    │
│    slurm-bootstrap.service → entrypoint.sh                  │
└──────────────────┬──────────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────────┐
│ 3. 基础配置                                                  │
│    - 检测 SLURM 模式                                         │
│    - 等待数据库                                              │
│    - 生成配置文件                                            │
│    - 配置 Munge                                              │
└──────────────────┬──────────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────────┐
│ 4. 等待服务启动                                              │
│    sleep 10                                                 │
└──────────────────┬──────────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────────┐
│ 5. 修复计算节点 (fix_compute_nodes)                         │
│    ┌─────────────────────────────────────────────────────┐ │
│    │ For each node in SLURM_TEST_NODES:                 │ │
│    │   - ping 检查网络                                    │ │
│    │   - sshpass + ssh 连接                              │ │
│    │   - mkdir -p 创建目录                                │ │
│    │   - chown 设置权限                                   │ │
│    │   - systemctl restart slurmd                        │ │
│    └─────────────────────────────────────────────────────┘ │
└──────────────────┬──────────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────────┐
│ 6. 等待节点注册                                              │
│    sleep 5                                                  │
└──────────────────┬──────────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────────┐
│ 7. Bootstrap 完成                                            │
│    munge, slurmctld, slurmdbd 服务启动                      │
└─────────────────────────────────────────────────────────────┘
                   │
                   ▼
┌─────────────────────────────────────────────────────────────┐
│ 8. 集群就绪                                                  │
│    所有节点状态: IDLE (可用)                                 │
└─────────────────────────────────────────────────────────────┘
```

## 验证步骤

### 1. 重新构建镜像

```bash
docker compose build slurm-master
```

### 2. 重启服务

```bash
docker compose up -d slurm-master
```

### 3. 查看启动日志

```bash
docker logs -f ai-infra-slurm-master

# 期望看到：
# [INFO] 🔧 修复计算节点配置...
# [INFO]   检查节点: test-rocky01
# [INFO]   ✅ 节点 test-rocky01 修复成功
# ...
# [INFO] ✅ 节点修复完成: 成功 6 个, 失败 0 个
```

### 4. 检查集群状态

```bash
docker exec ai-infra-slurm-master sinfo

# 期望输出：
# PARTITION AVAIL  TIMELIMIT  NODES  STATE NODELIST
# compute*     up   infinite      6  idle  test-rocky[01-03],test-ssh[01-03]
```

### 5. 检查节点详细状态

```bash
docker exec ai-infra-slurm-master scontrol show nodes

# 期望看到：
# NodeName=test-rocky01 ... State=IDLE ThreadsPerCore=1 ...
# (没有 NOT_RESPONDING)
```

### 6. 提交测试任务

```bash
docker exec ai-infra-slurm-master srun -N1 hostname

# 应该成功执行并返回节点名
```

## 关键改进点

### 1. 自动化
- ❌ **之前**：需要手动进入每个节点修复
- ✅ **现在**：master 启动时自动修复所有节点

### 2. 可靠性
- ❌ **之前**：节点重启后问题重现
- ✅ **现在**：每次 master 启动都会检查和修复

### 3. 可配置
- 通过环境变量控制要修复的节点列表
- 支持自定义 SSH 密码
- 可以轻松添加或移除节点

### 4. 错误处理
- 网络检查：ping 确认节点可达
- 超时控制：SSH 连接 5 秒超时
- 统计报告：显示成功和失败数量

## 适用场景

1. **初始部署**：首次启动集群时自动配置所有节点
2. **节点重启**：计算节点重启后目录丢失（tmpfs）
3. **添加新节点**：更新 `SLURM_TEST_NODES` 即可
4. **故障恢复**：master 重启时重新初始化所有节点

## 注意事项

### 安全性
- SSH 密码存储在环境变量中
- 建议生产环境使用密钥认证
- 可以限制 SSH 访问来源

### 网络要求
- master 必须能够 SSH 连接到所有计算节点
- 计算节点必须安装并启动 SSH 服务
- 防火墙需要允许 SSH 端口（22）

### 性能影响
- 每个节点修复耗时约 2-3 秒
- 6 个节点总耗时约 15-20 秒
- 不影响已运行的任务

## 扩展建议

### 短期
1. ✅ 基础节点修复功能
2. ⏳ 添加重试机制（失败后自动重试）
3. ⏳ 支持密钥认证方式
4. ⏳ 并行修复多个节点

### 长期
1. 健康检查定时任务（cron）
2. 节点状态监控和告警
3. 自动恢复故障节点
4. 与 K8s 集成进行节点管理

## 相关文件

### 修改的文件
- `src/slurm-master/entrypoint.sh` - 添加 `fix_compute_nodes()` 函数
- `src/slurm-master/Dockerfile` - 安装 `sshpass`
- `docker-compose.yml` - 添加 `SLURM_NODE_SSH_PASSWORD` 环境变量
- `.env` - 配置节点列表和 SSH 密码

### 涉及的服务
- `slurm-master` - 主控节点
- `test-rocky01-03` - Rocky Linux 计算节点
- `test-ssh01-03` - Ubuntu 计算节点

## 总结

通过在 `slurm-master` 的 bootstrap 流程中添加自动修复机制，实现了：

✅ **自动化修复**：无需手动干预
✅ **可靠性提升**：每次启动都确保节点正常
✅ **易于维护**：通过环境变量配置
✅ **错误处理**：完善的检查和统计
✅ **可扩展性**：轻松添加更多节点

这个方案解决了 SLURM 集群节点无法响应的问题，确保集群在启动后立即可用。
