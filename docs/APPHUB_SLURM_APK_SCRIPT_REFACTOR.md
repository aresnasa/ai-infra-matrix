# AppHub SLURM APK 构建脚本重构

## 修改日期
2025-10-20

## 修改原因

在 Dockerfile 中使用 `cat` 和 heredoc 语法创建脚本文件会导致 Docker 解析错误，特别是当存在多个 heredoc 时。为了简化 Dockerfile 并提高可维护性，将安装/卸载脚本分离为独立的脚本文件。

## 问题背景

### 原始实现问题

原始 Dockerfile 使用 `cat` 和 heredoc 在构建时创建脚本：

```dockerfile
RUN set -eux; \
    cat > /tmp/slurm-install/install.sh <<'EOINSTALL'
#!/bin/sh
...脚本内容...
EOINSTALL
    chmod +x /tmp/slurm-install/install.sh; \
    cat > /tmp/slurm-install/uninstall.sh <<'EOUNINSTALL'
#!/bin/sh
...脚本内容...
EOUNINSTALL
```

**问题**：
1. Dockerfile 语法复杂，heredoc 结束标记必须顶格且无缩进
2. 多个 heredoc 之间的分隔容易出错
3. Docker 解析器可能将 shell 命令误认为 Dockerfile 指令
4. 脚本内容嵌入在 Dockerfile 中，难以维护和测试

### 错误信息示例

```
ERROR: failed to solve: dockerfile parse error on line 424: 
unknown instruction: chmod (did you mean cmd?)
```

## 解决方案

### 新的文件结构

```
src/apphub/
├── Dockerfile
├── scripts/
│   ├── slurm-install.sh      # 安装脚本
│   ├── slurm-uninstall.sh    # 卸载脚本
│   └── slurm-README.md       # 使用文档
├── nginx.conf
└── entrypoint.sh
```

### Dockerfile 修改

**修改前**（使用 cat heredoc）：
```dockerfile
RUN set -eux; \
    if [ ! -f /home/builder/apk-output/.skip_slurm ]; then \
        cat > /tmp/slurm-install/install.sh <<'EOINSTALL'
#!/bin/sh
...大量脚本内容...
EOINSTALL
        chmod +x /tmp/slurm-install/install.sh; \
        cat > /tmp/slurm-install/uninstall.sh <<'EOUNINSTALL'
#!/bin/sh
...大量脚本内容...
EOUNINSTALL
        chmod +x /tmp/slurm-install/uninstall.sh; \
        cat > /tmp/slurm-install/README.md <<'EOREADME'
...大量文档内容...
EOREADME
        cd /tmp/slurm-install; \
        tar czf /home/builder/apk-output/slurm-client-25.05.4-alpine.tar.gz .; \
    fi
```

**修改后**（使用 COPY）：
```dockerfile
# 复制安装脚本到容器
COPY --chown=root:root scripts/slurm-install.sh /tmp/slurm-install/install.sh
COPY --chown=root:root scripts/slurm-uninstall.sh /tmp/slurm-install/uninstall.sh
COPY --chown=root:root scripts/slurm-README.md /tmp/slurm-install/README.md

# 设置脚本权限并打包
RUN set -eux; \
    if [ ! -f /home/builder/apk-output/.skip_slurm ]; then \
        chmod +x /tmp/slurm-install/install.sh /tmp/slurm-install/uninstall.sh; \
        cd /tmp/slurm-install; \
        mkdir -p /home/builder/apk-output; \
        tar czf /home/builder/apk-output/slurm-client-25.05.4-alpine.tar.gz .; \
        echo "✓ SLURM Alpine package created"; \
        ls -lh /home/builder/apk-output/; \
    fi
```

## 脚本文件内容

### 1. slurm-install.sh

安装脚本负责：
- 复制 SLURM 文件到系统目录（`/usr/local/slurm`, `/etc/slurm`）
- 创建符号链接到 `/usr/bin`
- 配置动态库路径（`/etc/ld.so.conf.d/slurm.conf`）
- 设置环境变量（`/etc/profile`）

```bash
#!/bin/sh
set -e

echo "Installing SLURM client tools..."

# 复制文件
cp -r usr/local/slurm /usr/local/
cp -r etc/slurm /etc/ 2>/dev/null || mkdir -p /etc/slurm

# 设置权限
chmod +x /usr/local/slurm/bin/*

# 创建符号链接
for cmd in /usr/local/slurm/bin/*; do
    ln -sf "$cmd" /usr/bin/$(basename "$cmd")
done

# 配置库路径
if [ ! -f /etc/ld.so.conf.d/slurm.conf ]; then
    mkdir -p /etc/ld.so.conf.d
    echo "/usr/local/slurm/lib" > /etc/ld.so.conf.d/slurm.conf
    ldconfig 2>/dev/null || true
fi

# 配置环境变量
if ! grep -q 'SLURM_HOME' /etc/profile 2>/dev/null; then
    cat >> /etc/profile << 'EOPROFILE'

# SLURM Client Environment
export SLURM_HOME=/usr/local/slurm
export PATH=$SLURM_HOME/bin:$PATH
export LD_LIBRARY_PATH=$SLURM_HOME/lib:$LD_LIBRARY_PATH
EOPROFILE
fi

echo "SLURM client tools installed successfully!"
echo "Version: $(cat /usr/local/slurm/VERSION 2>/dev/null || echo 'unknown')"
echo ""
echo "Available commands:"
ls -1 /usr/local/slurm/bin/
```

### 2. slurm-uninstall.sh

卸载脚本负责：
- 删除 SLURM 文件和目录
- 删除符号链接
- 删除库配置
- 清理环境变量

```bash
#!/bin/sh

echo "Uninstalling SLURM client tools..."

rm -rf /usr/local/slurm
rm -f /usr/bin/sinfo /usr/bin/squeue /usr/bin/scontrol /usr/bin/scancel
rm -f /usr/bin/sbatch /usr/bin/srun /usr/bin/salloc /usr/bin/sacct
rm -f /etc/ld.so.conf.d/slurm.conf
rm -rf /etc/slurm
sed -i '/SLURM_HOME/,+2d' /etc/profile 2>/dev/null || true

echo "SLURM client tools uninstalled."
```

### 3. slurm-README.md

完整的使用文档，包含：
- 安装说明
- 验证步骤
- 配置示例
- 故障排查
- 版本信息

## 优势

### 1. **简化 Dockerfile**
- 移除复杂的 heredoc 语法
- 减少 Dockerfile 行数（从约 80 行减少到 10 行）
- 提高可读性

### 2. **易于维护**
- 脚本可以独立编辑和测试
- 语法高亮支持（.sh 文件）
- 版本控制更清晰

### 3. **避免解析错误**
- 不再需要担心 heredoc 结束标记的缩进
- 不会出现 Docker 指令混淆
- 构建更稳定

### 4. **可测试性**
- 可以独立测试脚本（不需要构建整个镜像）
- 可以在容器外验证脚本逻辑
- 更容易调试

### 5. **可重用性**
- 脚本可以在其他项目中重用
- 可以单独分发（如果需要）
- 更容易文档化

## 构建验证

### 构建命令

```bash
./build.sh build apphub --force
```

### 验证步骤

1. **检查 Dockerfile 语法**：
   ```bash
   docker build -t test-apphub -f src/apphub/Dockerfile src/apphub --no-cache
   ```

2. **验证脚本文件存在**：
   ```bash
   ls -la src/apphub/scripts/
   # 应该看到:
   # slurm-install.sh
   # slurm-uninstall.sh
   # slurm-README.md
   ```

3. **检查包内容**：
   ```bash
   docker run --rm ai-infra-apphub:v0.3.6-dev tar tzf /usr/share/nginx/html/pkgs/slurm-apk/slurm-client-25.05.4-alpine.tar.gz | head -20
   # 应该包含:
   # install.sh
   # uninstall.sh
   # README.md
   # usr/local/slurm/...
   ```

4. **测试安装脚本**：
   ```bash
   # 在 backend 容器中
   docker-compose exec backend bash
   cd /tmp
   wget http://apphub/pkgs/slurm-apk/slurm-client-latest-alpine.tar.gz
   tar xzf slurm-client-latest-alpine.tar.gz
   ./install.sh
   source /etc/profile
   sinfo --version
   ```

## 后续工作

### 已完成
- ✅ 创建独立的安装脚本（`slurm-install.sh`）
- ✅ 创建独立的卸载脚本（`slurm-uninstall.sh`）
- ✅ 创建完整的 README（`slurm-README.md`）
- ✅ 修改 Dockerfile 使用 COPY 而非 heredoc
- ✅ 验证 Dockerfile 语法

### 待完成
- ⏳ 完整构建测试
- ⏳ Backend 集成测试
- ⏳ 端到端验证（Backend 自动下载和安装）

## 相关文档

- [AppHub SLURM APK Build](./APPHUB_SLURM_APK_BUILD.md) - APK 构建详解
- [AppHub SLURM APK Quickstart](./APPHUB_SLURM_APK_QUICKSTART.md) - 快速开始指南
- [Backend SLURM Client Setup](./BACKEND_SLURM_CLIENT_SETUP.md) - Backend 安装配置
- [SLURM AppHub Installation](./SLURM_APPHUB_INSTALLATION.md) - 完整安装指南

## 总结

通过将脚本从 Dockerfile 中分离到独立文件，我们：
1. **消除了 heredoc 语法错误**
2. **简化了 Dockerfile 结构**
3. **提高了可维护性和可测试性**
4. **使构建过程更加稳定可靠**

这是一个最佳实践的实现，推荐在其他类似场景中使用。
