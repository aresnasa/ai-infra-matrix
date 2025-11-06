#!/bin/bash
#
# SLURM 完整安装脚本（包含 REST API）
# 参考: https://slurm.schedmd.com/rest_quickstart.html
# 版本: 25.05.4
#

set -e

# 配置参数
APPHUB_URL=${APPHUB_URL:-"http://apphub"}
SLURM_VERSION="25.05.4-1"
SLURM_VERSION_SHORT="25.05.4"

# 检测系统架构
ARCH=$(uname -m)
case "${ARCH}" in
    x86_64)
        ARCH_DEB="amd64"
        ARCH_BIN="x86_64"
        ;;
    aarch64|arm64)
        ARCH_DEB="arm64"
        ARCH_BIN="arm64"
        ;;
    *)
        echo "❌ 不支持的架构: ${ARCH}"
        exit 1
        ;;
esac

echo "=========================================="
echo "  SLURM 完整安装脚本"
echo "  版本: ${SLURM_VERSION_SHORT}"
echo "  架构: ${ARCH} (${ARCH_DEB})"
echo "=========================================="
echo ""

# ==================== 第一部分：安装依赖包 ====================
echo "[1/6] 安装系统依赖包..."
apt-get update -qq
apt-get install -y --no-install-recommends \
    libdbus-1-3 \
    liblua5.3-0 \
    libmariadb3 \
    librdkafka1 \
    libhttp-parser2.9 \
    libjson-c5 \
    libyaml-0-2 \
    libjwt0 \
    curl \
    wget

echo "✅ 系统依赖包安装完成"

# ==================== 第二部分：下载并安装 DEB 包 ====================
echo ""
echo "[2/6] 下载 SLURM DEB 包..."
cd /tmp

DEB_BASE_URL="${APPHUB_URL}/pkgs/slurm-deb"
PACKAGES=(
    "slurm-smd_${SLURM_VERSION}_${ARCH_DEB}.deb"
    "slurm-smd-client_${SLURM_VERSION}_${ARCH_DEB}.deb"
    "slurm-smd-slurmctld_${SLURM_VERSION}_${ARCH_DEB}.deb"
    "slurm-smd-slurmd_${SLURM_VERSION}_${ARCH_DEB}.deb"
    "slurm-smd-slurmdbd_${SLURM_VERSION}_${ARCH_DEB}.deb"
    "slurm-smd-slurmrestd_${SLURM_VERSION}_${ARCH_DEB}.deb"
    "slurm-smd-libpmi0_${SLURM_VERSION}_${ARCH_DEB}.deb"
    "slurm-smd-libpmi2-0_${SLURM_VERSION}_${ARCH_DEB}.deb"
    "slurm-smd-libslurm-perl_${SLURM_VERSION}_${ARCH_DEB}.deb"
    "slurm-smd-libpam-slurm-adopt_${SLURM_VERSION}_${ARCH_DEB}.deb"
    "slurm-smd-libnss-slurm_${SLURM_VERSION}_${ARCH_DEB}.deb"
)

for pkg in "${PACKAGES[@]}"; do
    if [ ! -f "$pkg" ]; then
        echo "下载: $pkg"
        wget -q "${DEB_BASE_URL}/${pkg}" || {
            echo "  ⚠️  下载失败: $pkg (继续...)"
            continue
        }
    else
        echo "跳过（已存在）: $pkg"
    fi
done

echo "✅ DEB 包下载完成"

# ==================== 第三部分：按依赖顺序安装 DEB 包 ====================
echo ""
echo "[3/6] 安装 SLURM DEB 包..."

# 定义安装顺序（按依赖关系）
INSTALL_ORDER=(
    "slurm-smd_${SLURM_VERSION}_${ARCH_DEB}.deb"
    "slurm-smd-libpmi0_${SLURM_VERSION}_${ARCH_DEB}.deb"
    "slurm-smd-libpmi2-0_${SLURM_VERSION}_${ARCH_DEB}.deb"
    "slurm-smd-client_${SLURM_VERSION}_${ARCH_DEB}.deb"
    "slurm-smd-slurmctld_${SLURM_VERSION}_${ARCH_DEB}.deb"
    "slurm-smd-slurmd_${SLURM_VERSION}_${ARCH_DEB}.deb"
    "slurm-smd-slurmdbd_${SLURM_VERSION}_${ARCH_DEB}.deb"
    "slurm-smd-slurmrestd_${SLURM_VERSION}_${ARCH_DEB}.deb"
    "slurm-smd-libslurm-perl_${SLURM_VERSION}_${ARCH_DEB}.deb"
    "slurm-smd-libpam-slurm-adopt_${SLURM_VERSION}_${ARCH_DEB}.deb"
    "slurm-smd-libnss-slurm_${SLURM_VERSION}_${ARCH_DEB}.deb"
)

for pkg in "${INSTALL_ORDER[@]}"; do
    if [ -f "$pkg" ]; then
        echo "  安装: $pkg"
        dpkg -i "$pkg" 2>/dev/null || true
        apt-get install -f -y -qq
    else
        echo "  ⚠️  跳过（文件不存在）: $pkg"
    fi
done

echo "✅ DEB 包安装完成"

# ==================== 第四部分：下载并安装客户端工具 ====================
echo ""
echo "[4/6] 下载 SLURM 客户端工具..."

# 安装目录
INSTALL_DIR="/usr/local/slurm"
BIN_DIR="${INSTALL_DIR}/bin"
LIB_DIR="${INSTALL_DIR}/lib"

mkdir -p "${BIN_DIR}" "${LIB_DIR}"

# 客户端工具列表
BINARIES="sinfo squeue scontrol scancel sbatch srun salloc sacct sacctmgr"
BIN_BASE_URL="${APPHUB_URL}/pkgs/slurm-binaries/${ARCH_BIN}"

for bin in ${BINARIES}; do
    if [ ! -f "${BIN_DIR}/${bin}" ]; then
        echo "  下载: ${bin}"
        wget -q "${BIN_BASE_URL}/bin/${bin}" -O "${BIN_DIR}/${bin}" || {
            echo "  ⚠️  下载失败: ${bin} (继续...)"
            continue
        }
        chmod +x "${BIN_DIR}/${bin}"
        # 创建符号链接
        ln -sf "${BIN_DIR}/${bin}" "/usr/local/bin/${bin}" 2>/dev/null || true
    else
        echo "  跳过（已存在）: ${bin}"
    fi
done

# 下载库文件（如果有）
if wget -q --spider "${BIN_BASE_URL}/lib/" 2>/dev/null; then
    echo "  下载库文件..."
    wget -q -r -np -nH --cut-dirs=3 -P "${LIB_DIR}" \
        "${BIN_BASE_URL}/lib/" 2>/dev/null || true
fi

# 配置 PATH 和 LD_LIBRARY_PATH
if ! grep -q "${BIN_DIR}" /etc/profile 2>/dev/null; then
    echo "  配置环境变量..."
    cat >> /etc/profile << EOF

# SLURM 客户端工具
export PATH=\${PATH}:${BIN_DIR}
export LD_LIBRARY_PATH=\${LD_LIBRARY_PATH}:${LIB_DIR}
EOF
fi

echo "✅ 客户端工具安装完成"

# ==================== 第五部分：配置 JWT 认证 ====================
echo ""
echo "[5/6] 配置 JWT 认证..."

# 创建 JWT 密钥（如果不存在）
if [ ! -f /var/spool/slurm/statesave/jwt_hs256.key ]; then
    echo "  生成 JWT HS256 密钥..."
    dd if=/dev/random of=/var/spool/slurm/statesave/jwt_hs256.key bs=32 count=1 2>/dev/null
    chown slurm:slurm /var/spool/slurm/statesave/jwt_hs256.key
    chmod 0600 /var/spool/slurm/statesave/jwt_hs256.key
    echo "  ✅ JWT 密钥已生成"
else
    echo "  ✅ JWT 密钥已存在"
fi

# 更新 slurm.conf 启用 JWT
if ! grep -q "AuthAltTypes=auth/jwt" /etc/slurm/slurm.conf; then
    echo "  添加 JWT 认证配置到 slurm.conf..."
    sed -i '/^AuthType=/a AuthAltTypes=auth/jwt' /etc/slurm/slurm.conf
    echo "  ✅ JWT 认证配置已添加"
else
    echo "  ✅ JWT 认证配置已存在"
fi

echo "✅ JWT 认证配置完成"

# ==================== 第六部分：配置 slurmrestd ====================
echo ""
echo "[6/6] 配置 slurmrestd..."

# 创建 slurmrestd 用户（如果不存在）
if ! id slurmrestd &>/dev/null; then
    useradd -M -r -s /usr/sbin/nologin -U slurmrestd 2>/dev/null || true
    echo "  ✅ slurmrestd 用户已创建"
else
    echo "  ✅ slurmrestd 用户已存在"
fi

# 创建环境配置文件
mkdir -p /etc/default
cat > /etc/default/slurmrestd << 'EOF'
# SLURM REST API 配置
SLURM_JWT=daemon
SLURMRESTD_DEBUG=debug
SLURMRESTD_LISTEN=:6820
SLURMRESTD_OPTIONS="-vvvv"
EOF

echo "  ✅ slurmrestd 配置文件已创建"

# 添加 supervisor 配置（如果使用 supervisor）
if [ -d /etc/supervisor/conf.d ]; then
    cat > /etc/supervisor/conf.d/slurmrestd.conf << 'EOF'
[program:slurmrestd]
command=/usr/sbin/slurmrestd :6820
directory=/var/spool/slurm
user=slurmrestd
autostart=true
autorestart=true
redirect_stderr=true
stdout_logfile=/var/log/slurm/slurmrestd.log
environment=SLURM_JWT=daemon
EOF
    echo "  ✅ Supervisor 配置已创建"
fi

echo "✅ slurmrestd 配置完成"

echo ""
echo "=========================================="
echo "  🎉 安装完成！"
echo "=========================================="
echo ""
echo "� 已安装组件："
echo "  - SLURM ${SLURM_VERSION_SHORT} DEB 包"
echo "  - SLURM 客户端工具 (sinfo, squeue, scontrol...)"
echo "  - slurmrestd REST API 服务"
echo "  - JWT 认证支持"
echo ""
echo "�📝 下一步操作："
echo "  1. 重启 slurmctld："
echo "     supervisorctl restart slurmctld"
echo ""
echo "  2. 启动 slurmrestd (选择其一)："
echo "     # 方式1: 使用 supervisor"
echo "     supervisorctl update"
echo "     supervisorctl start slurmrestd"
echo ""
echo "     # 方式2: 直接启动"
echo "     slurmrestd :6820 &"
echo ""
echo "  3. 测试 REST API："
echo "     export \$(scontrol token)"
echo "     curl -H \"X-SLURM-USER-TOKEN:\$SLURM_JWT\" \\"
echo "       http://localhost:6820/slurm/v0.0.40/diag | jq ."
echo ""
echo "  4. 验证集群状态："
echo "     sinfo"
echo "     squeue"
echo ""
echo "🔗 REST API 端点: http://slurm-master:6820"
echo "📖 API 文档: https://slurm.schedmd.com/rest_api.html"
echo "📖 快速参考: https://slurm.schedmd.com/rest_quickstart.html"
echo ""
