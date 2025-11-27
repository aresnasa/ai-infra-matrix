#!/bin/bash
set -e

# 设置root密码（如果提供了SLURM_MASTER_PASSWORD环境变量）
if [ -n "${SLURM_MASTER_PASSWORD:-}" ]; then
    echo "正在设置root密码..."
    echo "root:${SLURM_MASTER_PASSWORD}" | chpasswd
fi

# 创建 SLURM 必需的目录并设置权限
echo "创建 SLURM 运行目录..."
mkdir -p /var/run/slurm
mkdir -p /var/lib/slurm/slurmctld
mkdir -p /var/lib/slurm/slurmdbd
mkdir -p /var/log/slurm
chown -R slurm:slurm /var/run/slurm
chown -R slurm:slurm /var/lib/slurm
chown -R slurm:slurm /var/log/slurm
chmod 755 /var/run/slurm
chmod 755 /var/lib/slurm/slurmctld
chmod 755 /var/lib/slurm/slurmdbd

ENV_FILE="/etc/sysconfig/slurm-env"
mkdir -p /etc/sysconfig /etc/systemd/system/multi-user.target.wants

cat >"${ENV_FILE}" <<EOF
SLURM_CLUSTER_NAME=${SLURM_CLUSTER_NAME:-ai-infra-cluster}
SLURM_CONTROLLER_HOST=${SLURM_CONTROLLER_HOST:-slurm-master}
SLURM_CONTROLLER_PORT=${SLURM_CONTROLLER_PORT:-6817}
SLURM_SLURMDBD_HOST=${SLURM_SLURMDBD_HOST:-slurm-master}
SLURM_SLURMDBD_PORT=${SLURM_SLURMDBD_PORT:-6818}
SLURM_DB_HOST=${SLURM_DB_HOST:-mysql}
SLURM_DB_PORT=${SLURM_DB_PORT:-3306}
SLURM_DB_NAME=${SLURM_DB_NAME:-slurm_acct_db}
SLURM_DB_USER=${SLURM_DB_USER:-slurm}
SLURM_DB_PASSWORD=${SLURM_DB_PASSWORD:-slurm123}
MYSQL_ROOT_USER=${MYSQL_ROOT_USER:-root}
MYSQL_ROOT_PASSWORD=${MYSQL_ROOT_PASSWORD:-}
SLURM_AUTH_TYPE=${SLURM_AUTH_TYPE:-auth/munge}
SLURM_MUNGE_KEY=${SLURM_MUNGE_KEY:-ai-infra-slurm-munge-key-dev}
SLURM_PARTITION_NAME=${SLURM_PARTITION_NAME:-compute}
SLURM_DEFAULT_PARTITION=${SLURM_DEFAULT_PARTITION:-compute}
SLURM_NODE_PREFIX=${SLURM_NODE_PREFIX:-compute}
SLURM_NODE_COUNT=${SLURM_NODE_COUNT:-3}
# 默认不添加测试节点，通过 Web 界面或环境变量添加节点
SLURM_TEST_NODES=${SLURM_TEST_NODES:-}
SLURM_TEST_NODE_CPUS=${SLURM_TEST_NODE_CPUS:-4}
SLURM_TEST_NODE_MEMORY=${SLURM_TEST_NODE_MEMORY:-8192}
SLURM_MAX_JOB_COUNT=${SLURM_MAX_JOB_COUNT:-10000}
SLURM_MAX_ARRAY_SIZE=${SLURM_MAX_ARRAY_SIZE:-1000}
SLURM_DEFAULT_TIME_LIMIT=${SLURM_DEFAULT_TIME_LIMIT:-01:00:00}
SLURM_MAX_TIME_LIMIT=${SLURM_MAX_TIME_LIMIT:-24:00:00}
AI_INFRA_BACKEND_URL=${AI_INFRA_BACKEND_URL:-http://backend:8082}
DEBUG_MODE=${DEBUG_MODE:-false}
EOF

if [ -f /opt/slurmctld-path ]; then
	echo "SLURMCTLD_BIN=$(cat /opt/slurmctld-path)" >>"${ENV_FILE}"
fi

if [ -f /opt/slurmdbd-path ]; then
	echo "SLURMDBD_BIN=$(cat /opt/slurmdbd-path)" >>"${ENV_FILE}"
fi

# Create munge drop-in to ensure it runs after bootstrap and fix service configuration
mkdir -p /etc/systemd/system/munge.service.d
cat >/etc/systemd/system/munge.service.d/override.conf <<'MUNGE_EOF'
[Unit]
After=slurm-bootstrap.service
Requires=slurm-bootstrap.service

[Service]
# Change to simple type to avoid PID file ownership issues
Type=simple
PIDFile=
ExecStartPre=
ExecStartPre=/bin/mkdir -p /run/munge
ExecStartPre=/bin/chown munge:munge /run/munge
ExecStartPre=/bin/chmod 755 /run/munge
ExecStart=
ExecStart=/usr/sbin/munged --foreground
MUNGE_EOF

# Enable systemd units shipped with the image
ln -sf /etc/systemd/system/slurm-bootstrap.service /etc/systemd/system/multi-user.target.wants/slurm-bootstrap.service
ln -sf /etc/systemd/system/slurmctld.service /etc/systemd/system/multi-user.target.wants/slurmctld.service
ln -sf /etc/systemd/system/slurmdbd.service /etc/systemd/system/multi-user.target.wants/slurmdbd.service
ln -sf /lib/systemd/system/munge.service /etc/systemd/system/multi-user.target.wants/munge.service

# 动态查找 systemd 可执行文件
# 在正确构建的镜像中，systemd 应该已经安装
# 如果找不到，说明镜像构建有问题，给出明确提示
ensure_systemd() {
	# 检查常见的 systemd 路径
	if [ -x /sbin/init ]; then
		# 验证 /sbin/init 是否真的是 systemd
		if /sbin/init --version 2>&1 | grep -q systemd; then
			SYSTEMD_BIN="/sbin/init"
			echo "✅ 找到 systemd: $SYSTEMD_BIN"
			return 0
		fi
	fi

	if [ -x /lib/systemd/systemd ]; then
		SYSTEMD_BIN="/lib/systemd/systemd"
		echo "✅ 找到 systemd: $SYSTEMD_BIN"
		return 0
	fi

	if command -v systemd >/dev/null 2>&1; then
		SYSTEMD_BIN="$(command -v systemd)"
		echo "✅ 找到 systemd: $SYSTEMD_BIN"
		return 0
	fi

	# systemd 未找到，尝试运行时安装（仅作为后备方案）
	echo "⚠️  systemd 未找到，这可能表示镜像构建不完整"
	echo "📦 尝试在启动阶段安装 systemd..."
	export DEBIAN_FRONTEND=noninteractive
	
	# 使用多种方式尝试安装
	if apt-get update 2>/dev/null && apt-get install -y --no-install-recommends systemd systemd-sysv 2>/dev/null; then
		echo "✅ systemd 安装成功"
		if [ -x /lib/systemd/systemd ]; then
			SYSTEMD_BIN="/lib/systemd/systemd"
			return 0
		elif [ -x /sbin/init ]; then
			SYSTEMD_BIN="/sbin/init"
			return 0
		fi
	fi

	# 所有尝试都失败
	echo "❌ 无法找到或安装 systemd" >&2
	echo "" >&2
	echo "可能的原因:" >&2
	echo "  1. 镜像构建时未能成功安装 systemd" >&2
	echo "  2. 使用的是旧版本镜像，需要重新构建" >&2
	echo "  3. 容器内网络无法访问 APT 源" >&2
	echo "" >&2
	echo "建议:" >&2
	echo "  - 重新构建镜像: ./build.sh slurm-master" >&2
	echo "  - 或从私有仓库拉取最新镜像" >&2
	return 1
}

ensure_systemd || exit 1

# 检查 cgroup 挂载情况
echo "🔍 检查 cgroup 挂载..."
if [ ! -d /sys/fs/cgroup ]; then
    echo "❌ /sys/fs/cgroup 不存在！"
    echo "   请确保 docker-compose.yml 中包含以下挂载:"
    echo "   volumes:"
    echo "     - /sys/fs/cgroup:/sys/fs/cgroup:rw"
    exit 1
fi

# 检测 cgroup 版本
if [ -f /sys/fs/cgroup/cgroup.controllers ]; then
    echo "✅ 检测到 cgroup v2"
    CGROUP_VERSION="v2"
else
    echo "ℹ️  使用 cgroup v1 或混合模式"
    CGROUP_VERSION="v1"
fi

# 列出 cgroup 内容用于调试
echo "📋 /sys/fs/cgroup 内容:"
ls -la /sys/fs/cgroup/ 2>/dev/null | head -10

# 确保 systemd 需要的目录存在
mkdir -p /run/systemd/system

# 对于 cgroup v2，可能需要确保某些权限
if [ "$CGROUP_VERSION" = "v2" ]; then
    # 检查是否可写
    if [ -w /sys/fs/cgroup ]; then
        echo "✅ cgroup v2 可写"
    else
        echo "⚠️  cgroup v2 不可写，可能影响 systemd 启动"
    fi
fi

# 如果仍然使用默认的 /sbin/init，则替换为实际存在的 systemd
if [ "$#" -eq 0 ] || [ "$1" = "/sbin/init" ]; then
    echo "🚀 启动 systemd: $SYSTEMD_BIN"
    # 直接运行 systemd，不加额外参数（由 systemd 自己判断）
    set -- "$SYSTEMD_BIN"
fi

exec "$@"
