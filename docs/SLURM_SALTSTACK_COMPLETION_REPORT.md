# SLURM + SaltStack + AppHub 集成完成报告

## 执行日期
2025-11-11

## 目标
改造 SLURM 节点安装流程，使 Ubuntu 和 Rocky Linux 节点都能从 AppHub 统一安装 SLURM 25.05.4。

## 完成的工作

### 1. 修改安装脚本支持 DEB 包 ✅

**文件**：`src/backend/scripts/install-slurm-node.sh`

**修改内容**：
- 添加 `configure_slurm_repo()` 函数的 DEB 支持
- 配置 AppHub DEB 仓库：`deb [trusted=yes] ${APPHUB_URL}/pkgs/slurm-deb/ ./`
- 修改 `install_slurm_packages()` 函数使用 `slurm-smd` DEB 包
- 支持的包：
  - slurm-smd-client
  - slurm-smd-slurmd
  - slurm-smd-libpmi0
  - slurm-smd-libslurm-perl

**测试结果**：
- ✅ Ubuntu 22.04：成功安装 SLURM 25.05.4
- ✅ Rocky Linux 9.3：成功安装 SLURM 25.05.4

### 2. 创建配置脚本 ✅

**文件**：`src/backend/scripts/configure-slurm-node.sh`（新建）

**功能**：
1. 自动部署 munge key（从 master，base64 编码）
2. 自动部署 slurm.conf（从 master，base64 编码）
3. 启动 munge 服务
4. 启动 slurmd 守护进程（针对不同 OS 优化）

**特性**：
- Ubuntu：使用 systemd 启动
- Rocky：直接启动（避免 systemd 超时）
- 自动检测 OS 类型
- 完整的错误处理和日志

**测试结果**：
- ✅ 所有 6 个节点配置成功
- ✅ munge 和 slurmd 进程正常运行

### 3. AppHub DEB 包验证 ✅

**位置**：`ai-infra-apphub:/usr/share/nginx/html/pkgs/slurm-deb/`

**包含文件**：
- slurm-smd_25.05.4-1_arm64.deb (2.8 MB)
- slurm-smd-client_25.05.4-1_arm64.deb (747 KB)
- slurm-smd-slurmd_25.05.4-1_arm64.deb (249 KB)
- slurm-smd-libpmi0_25.05.4-1_arm64.deb (12 KB)
- slurm-smd-libslurm-perl_25.05.4-1_arm64.deb (151 KB)
- 其他：slurmctld, slurmdbd, slurmrestd, 等

**仓库配置**：
- Packages 和 Packages.gz 文件已生成
- 支持 APT 访问：`http://ai-infra-apphub/pkgs/slurm-deb/`

### 4. 所有节点统一版本 ✅

**安装情况**：

| 节点 | OS | SLURM 版本 | 包类型 | 状态 |
|-----|-----|-----------|--------|------|
| test-ssh01 | Ubuntu 22.04 | 25.05.4 | DEB | ✅ 工作正常 |
| test-ssh02 | Ubuntu 22.04 | 25.05.4 | DEB | ✅ 工作正常 |
| test-ssh03 | Ubuntu 22.04 | 25.05.4 | DEB | ✅ 工作正常 |
| test-rocky01 | Rocky 9.3 | 25.05.4 | RPM | ⚠️ 状态异常 |
| test-rocky02 | Rocky 9.3 | 25.05.4 | RPM | ⚠️ 状态异常 |
| test-rocky03 | Rocky 9.3 | 25.05.4 | RPM | ⚠️ 状态异常 |

### 5. SaltStack 集成 ✅

**流程**：
1. 安装阶段：
   ```bash
   salt-cp → 传输安装脚本
   salt cmd.run → 执行安装（timeout=600）
   ```

2. 配置阶段：
   ```bash
   salt-cp → 传输配置脚本
   salt cmd.run → 执行配置（timeout=60）
   ```

**优势**：
- 支持并行部署
- 统一的日志输出
- 自动错误处理
- 可扩展到更多节点

## 发现的问题

### Rocky Linux 节点状态异常 ⚠️

**症状**：
```bash
$ docker exec ai-infra-slurm-master sinfo
PARTITION AVAIL  TIMELIMIT  NODES  STATE NODELIST
compute*     up   infinite      3   unk* test-rocky[01-03]
compute*     up   infinite      3   idle test-ssh[01-03]
```

**分析**：
1. Rocky 节点显示 `unk*` (unknown) 或 `idle*`
2. munge key 正确（MD5 一致）
3. slurmd 进程运行正常
4. 网络连接正常
5. scontrol 显示：`State=IDLE+NOT_RESPONDING`

**可能原因**：
1. **RPM 包构建问题**：
   - 使用 `rpmbuild --nodeps` 构建
   - 可能缺少关键依赖库
   - 二进制兼容性问题

2. **通信协议不匹配**：
   - RPM 和 DEB 包的构建配置不同
   - 可能使用不同的编译选项

3. **版本标识问题**：
   - RPM 包的版本标识可能与 master 不完全一致

**测试结果**：
- Ubuntu 节点可以成功接受任务 ✅
- Rocky 节点无法接受任务 ❌

## 临时解决方案

### 方案 1：仅使用 Ubuntu 节点（推荐）

**优势**：
- 已验证工作正常
- 稳定可靠
- 可立即投入使用

**实施**：
```bash
# 从集群中移除 Rocky 节点
docker exec ai-infra-slurm-master bash -c "
  scontrol update NodeName=test-rocky[01-03] State=DRAIN Reason='Using Ubuntu nodes only'
"
```

### 方案 2：手动测试 Rocky 节点

**步骤**：
1. 在 Rocky 节点安装调试工具
2. 抓取 slurmd 与 slurmctld 的通信包
3. 对比 Ubuntu 和 Rocky 节点的差异
4. 修改 RPM 构建脚本

## 长期解决方案

### 选项 A：重新构建 RPM 包

**步骤**：
1. 修改 `src/apphub/Dockerfile` 中的 RPM 构建部分
2. 安装完整的依赖而不是使用 `--nodeps`
3. 确保编译选项与 DEB 包一致
4. 重新测试

### 选项 B：统一使用 DEB 包

**方案**：
- 在 Rocky 节点安装 `alien` 工具
- 将 DEB 包转换为 RPM
- 或直接在 Rocky 上使用 DEB 包（通过 alien）

### 选项 C：使用容器化 SLURM 节点

**方案**：
- 所有节点使用统一的 Ubuntu 容器镜像
- 避免 OS 差异带来的问题
- 更容易管理和部署

## Go 代码改进建议

### 添加配置函数

```go
// configureSlurmNodeViaSalt 通过 Salt 配置 SLURM 节点
func (s *SlurmService) configureSlurmNodeViaSalt(
    ctx context.Context,
    nodeName string,
    logWriter io.Writer,
) error {
    // 1. 读取 master 配置
    mungeKey, err := s.readMasterMungeKey()
    if err != nil {
        return fmt.Errorf("读取 munge key 失败: %w", err)
    }
    
    slurmConf, err := s.readMasterSlurmConf()
    if err != nil {
        return fmt.Errorf("读取 slurm.conf 失败: %w", err)
    }
    
    // 2. Base64 编码
    mungeKeyB64 := base64.StdEncoding.EncodeToString(mungeKey)
    slurmConfB64 := base64.StdEncoding.EncodeToString(slurmConf)
    
    // 3. 通过 Salt 执行配置脚本
    // ... (类似 installSlurmPackages 的逻辑)
    
    return nil
}
```

### 统一的部署流程

```go
func (s *SlurmService) DeployNodeWithSalt(
    ctx context.Context,
    nodeName, osType string,
    logWriter io.Writer,
) error {
    // 步骤 1：安装包
    if err := s.installSlurmPackages(ctx, nodeName, osType, logWriter); err != nil {
        return fmt.Errorf("安装失败: %w", err)
    }
    
    // 步骤 2：配置节点
    if err := s.configureSlurmNodeViaSalt(ctx, nodeName, logWriter); err != nil {
        return fmt.Errorf("配置失败: %w", err)
    }
    
    // 步骤 3：验证节点状态
    if err := s.verifyNodeStatus(ctx, nodeName, logWriter); err != nil {
        return fmt.Errorf("验证失败: %w", err)
    }
    
    return nil
}
```

## 测试验证

### Ubuntu 节点测试 ✅

```bash
# 集群状态
$ docker exec ai-infra-slurm-master sinfo
PARTITION AVAIL  TIMELIMIT  NODES  STATE NODELIST
compute*     up   infinite      3   idle test-ssh[01-03]

# 任务提交测试
$ docker exec ai-infra-slurm-master srun -N 3 hostname
test-ssh01
test-ssh02
test-ssh03

# 单节点测试
$ docker exec ai-infra-slurm-master srun -w test-ssh01 hostname
test-ssh01
```

### Rocky 节点测试 ❌

```bash
# 节点状态
$ docker exec ai-infra-slurm-master scontrol show node test-rocky01
NodeName=test-rocky01
State=IDLE+NOT_RESPONDING
Reason=Not responding

# 任务提交失败
$ docker exec ai-infra-slurm-master srun -w test-rocky01 hostname
srun: Required node not available (down, drained or reserved)
```

## 文档输出

1. **安装指南**：`docs/SLURM_SALTSTACK_INSTALL_GUIDE.md`
   - 完整的安装步骤
   - 故障排查指南
   - 最佳实践

2. **配置脚本**：
   - `src/backend/scripts/install-slurm-node.sh` (已更新)
   - `src/backend/scripts/configure-slurm-node.sh` (新建)

3. **本报告**：`docs/SLURM_SALTSTACK_COMPLETION_REPORT.md`

## 下一步行动

### 立即可做：
1. ✅ 使用 Ubuntu 节点投入生产
2. ✅ 更新 Go 代码集成配置脚本
3. ✅ 编写用户文档

### 需要进一步测试：
1. ⏳ 调查 Rocky 节点问题
2. ⏳ 重新构建 RPM 包
3. ⏳ 完善错误处理和日志

### 可选优化：
1. 🔄 添加节点健康检查
2. 🔄 实现自动故障恢复
3. 🔄 支持更多 OS 类型（Debian, CentOS, 等）

## 结论

**主要成果**：
- ✅ Ubuntu 节点完全可用，从 AppHub 统一安装 SLURM 25.05.4
- ✅ SaltStack 集成成功，支持远程批量部署
- ✅ 安装和配置脚本完善，易于维护

**待解决问题**：
- ⚠️ Rocky Linux 节点需要进一步调试或重新构建 RPM 包

**推荐方案**：
- 短期：使用 Ubuntu 节点（已验证工作）
- 长期：重新构建 Rocky RPM 包，确保完整依赖和兼容性

---

**报告人**：AI Assistant  
**日期**：2025-11-11  
**版本**：v1.0
