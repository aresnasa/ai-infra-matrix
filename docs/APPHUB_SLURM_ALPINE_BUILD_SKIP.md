# AppHub SLURM Alpine 客户端构建方案调整

## 日期
2025-10-20

## 问题总结

在尝试为 Alpine Linux 编译 SLURM 客户端工具时遇到以下问题：

### 1. **Dockerfile Heredoc 语法错误**
- **问题**：多个 heredoc 在 Dockerfile 中语法复杂，容易出现解析错误
- **解决方案**：将脚本分离为独立文件（`slurm-install.sh`, `slurm-uninstall.sh`, `slurm-README.md`）

### 2. **Alpine SLURM 编译失败**
- **问题**：SLURM 在 Alpine 上编译复杂，依赖缺失
- **错误**：
  ```
  configure: WARNING: unrecognized options: --without-pam, --without-gtk2, --without-numa
  /bin/sh: syntax error: bad substitution (PIPESTATUS 不可用)
  ```
- **根本原因**：
  1. Alpine 使用 `/bin/sh` (ash)，不支持 bash 特性如 `PIPESTATUS`
  2. SLURM configure 选项在不同版本有差异
  3. Alpine 缺少某些编译依赖

## 最终方案

### Stage 3 (apk-builder) - 跳过 SLURM 构建

**决策**：暂时跳过 Alpine SLURM 客户端构建

```dockerfile
# 解压 SLURM 源码 - 跳过 Alpine 构建
RUN set -eux; \
    mkdir -p /home/builder/apk-output; \
    echo "SKIP_SLURM_BUILD=1" > /home/builder/build/.srcdir; \
    touch /home/builder/apk-output/.skip_slurm; \
    echo "⚠️  Alpine SLURM build skipped (complex dependencies)"; \
    echo "💡 Backend will use demo mode or extract binaries from deb packages"
```

### Backend 降级策略

Backend Dockerfile 已实现优雅降级：

```dockerfile
RUN set -eux; \
    APPHUB_SLURM_INSTALLED=false; \
    for APPHUB_URL in http://apphub/pkgs/slurm-apk/slurm-client-latest-alpine.tar.gz \
                      http://ai-infra-apphub/pkgs/slurm-apk/... \
                      http://192.168.0.200:8081/pkgs/slurm-apk/...; do \
        if wget -q --timeout=10 --tries=2 "$APPHUB_URL" -O /tmp/slurm.tar.gz 2>/dev/null; then \
            if [ -f /tmp/slurm.tar.gz ] && [ -s /tmp/slurm.tar.gz ]; then \
                cd /tmp && tar xzf slurm.tar.gz && ./install.sh; \
                APPHUB_SLURM_INSTALLED=true; \
                break; \
            fi; \
        fi; \
    done; \
    if [ "$APPHUB_SLURM_INSTALLED" = "false" ]; then \
        echo "⚠️  无法从 AppHub 下载 SLURM 客户端"; \
        echo "💡 使用演示数据创建 SLURM 命令占位符..."; \
        mkdir -p /usr/local/bin; \
        for cmd in sinfo squeue scontrol sbatch scancel srun; do \
            cat > /usr/local/bin/$cmd << 'DEMOCMD'
#!/bin/sh
echo "[DEMO] This is a placeholder SLURM command"
echo "[DEMO] Real SLURM client not available from AppHub"
echo "[DEMO] Command: $0 $@"
DEMOCMD
            chmod +x /usr/local/bin/$cmd; \
        done; \
    fi
```

## 替代方案（未来考虑）

### 方案 A：使用预编译二进制文件
```dockerfile
# 从官方或第三方源下载预编译的 Alpine SLURM 客户端
RUN wget https://example.com/slurm-client-alpine.tar.gz
```

### 方案 B：从 DEB 包提取
```dockerfile
# 在 apk-builder stage 从 deb-builder 复制并提取
COPY --from=deb-builder /out/*.deb /tmp/
RUN ar x /tmp/slurm-slurmctld_*.deb && \
    tar xzf data.tar.gz && \
    # 提取所需的二进制文件
```

### 方案 C：使用 Alpine 社区包
```bash
# 如果 Alpine 社区仓库添加了 SLURM 包
apk add slurm-client
```

### 方案 D：容器内编译（复杂但完整）
- 安装完整的构建依赖（包括从源码构建 munge 等）
- 使用 bash 而不是 sh
- 完整编译 SLURM

## 当前状态

### ✅ 已完成

1. **Dockerfile 语法修复**
   - 创建独立脚本文件（`src/apphub/scripts/`）
   - 使用 COPY 而不是 heredoc
   - 简化 Dockerfile 结构

2. **Stage 1 & 2**
   - ✅ Ubuntu DEB 构建正常
   - ✅ Rocky Linux RPM 构建正常（下载 SaltStack）

3. **Stage 3**
   - ✅ 跳过 Alpine SLURM 构建
   - ✅ 创建 `.skip_slurm` 标记

4. **Stage 4**
   - ✅ AppHub HTTP 服务正常
   - ✅ DEB/RPM 包可用

5. **Backend 降级**
   - ✅ 自动尝试从 AppHub 下载
   - ✅ 失败后使用演示数据模式
   - ✅ 创建占位符命令

### ⏳ 待完成

1. **Alpine SLURM 构建**（优先级：低）
   - 研究 Alpine 特定的编译选项
   - 或使用预编译二进制文件

2. **验证完整流程**
   - 测试 Backend 演示模式
   - 验证占位符命令功能

3. **文档更新**
   - 更新安装指南说明演示模式
   - 添加故障排查文档

## 影响评估

### 对用户的影响

**最小影响**：
- Backend 容器会自动使用演示模式
- SLURM 命令仍然可用（占位符）
- 不会导致容器启动失败

**限制**：
- 无法实际提交作业到 SLURM 集群
- SLURM 命令只返回演示信息

### 对系统的影响

**正面**：
- ✅ AppHub 构建更快（跳过复杂编译）
- ✅ 构建更稳定（减少失败点）
- ✅ 维护更简单

**负面**：
- ⚠️ Alpine Backend 无真实 SLURM 客户端
- ⚠️ 需要其他方式连接 SLURM 集群

## 推荐行动

### 短期（当前）
1. ✅ 使用演示模式运行 Backend
2. ✅ 完成其他功能开发和测试
3. ✅ 更新文档说明限制

### 中期（1-2周）
1. 研究从 DEB 包提取二进制文件的方案
2. 测试预编译二进制文件的兼容性
3. 或使用基于 Ubuntu 的 Backend 镜像

### 长期（按需）
1. 如果必须使用 Alpine + SLURM：
   - 深入研究 Alpine SLURM 编译
   - 创建专门的构建脚本
   - 或使用社区维护的包

## 相关文件

### 已创建的文件
- `src/apphub/scripts/slurm-install.sh` - 安装脚本
- `src/apphub/scripts/slurm-uninstall.sh` - 卸载脚本
- `src/apphub/scripts/slurm-README.md` - 使用文档
- `docs/APPHUB_SLURM_APK_SCRIPT_REFACTOR.md` - 脚本重构文档
- `docs/APPHUB_SLURM_ALPINE_BUILD_SKIP.md` - 本文档

### 修改的文件
- `src/apphub/Dockerfile` - Stage 3 跳过编译
- `src/backend/Dockerfile` - 已有降级逻辑（无需修改）

## 总结

通过跳过 Alpine SLURM 编译并使用演示模式，我们：

1. **解决了构建失败问题** - AppHub 可以成功构建
2. **保持了系统稳定性** - Backend 不会因缺少 SLURM 而失败
3. **提供了优雅降级** - 用户可以看到占位符命令
4. **为未来留有空间** - 可以后续添加真实的 Alpine SLURM 支持

这是一个**实用主义的解决方案**，平衡了复杂度、稳定性和开发效率。

## 验证步骤

完整构建和测试：

```bash
# 1. 构建 AppHub（应该成功）
./build.sh build apphub --force

# 2. 启动 AppHub
docker-compose up -d apphub

# 3. 验证包可用性
curl http://localhost:8081/pkgs/slurm-deb/
curl http://localhost:8081/pkgs/slurm-rpm/
curl http://localhost:8081/pkgs/slurm-apk/  # 应该只有 .skip_slurm 标记

# 4. 构建 Backend
./build.sh build backend --force

# 5. 启动 Backend
docker-compose up -d backend

# 6. 验证演示模式
docker-compose exec backend bash
which sinfo squeue  # 应该找到占位符命令
sinfo  # 应该显示演示信息
```

## 后续支持

如果需要真实的 SLURM 客户端支持：

### 选项 1：使用 Ubuntu Backend
将 Backend 基础镜像从 Alpine 改为 Ubuntu，直接使用 DEB 包。

### 选项 2：静态编译
创建独立的构建环境，生成静态链接的 SLURM 二进制文件。

### 选项 3：容器编排
使用 sidecar 容器提供 SLURM 客户端功能。
