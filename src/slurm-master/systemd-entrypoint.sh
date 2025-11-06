#!/bin/bash
set -e

# 设置root密码（如果提供了SLURM_MASTER_PASSWORD环境变量）
if [ -n "${SLURM_MASTER_PASSWORD:-}" ]; then
    echo "正在设置root密码..."
    echo "root:${SLURM_MASTER_PASSWORD}" | chpasswd
fi

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
SLURM_TEST_NODES=${SLURM_TEST_NODES:-test-ssh01,test-ssh02,test-ssh03}
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

exec /sbin/init "$@"
