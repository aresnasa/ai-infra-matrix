#!/bin/bash
# Test Container Entrypoint Script
# 动态配置SSH密码并启动systemd

set -e

# 从环境变量设置root密码（默认为rootpass123）
if [ -n "$ROOT_PASSWORD" ]; then
    echo "root:$ROOT_PASSWORD" | chpasswd
    echo "✓ Root密码已从环境变量设置"
else
    echo "root:rootpass123" | chpasswd
    echo "✓ Root密码已设置为默认值: rootpass123"
fi

# 从环境变量设置testuser密码（默认为testpass123）
if [ -n "$SSH_PASSWORD" ]; then
    echo "testuser:$SSH_PASSWORD" | chpasswd
    echo "✓ testuser密码已从环境变量设置"
else
    echo "testuser:testpass123" | chpasswd
    echo "✓ testuser密码已设置为默认值: testpass123"
fi

# 确保SSH目录存在
mkdir -p /var/run/sshd /run/sshd /root/.ssh

# 设置 SSH 密钥（用于访问 slurm-master）
if [ -f /srv/shared/ssh-key/id_rsa ]; then
    cp /srv/shared/ssh-key/id_rsa /root/.ssh/id_rsa
    chmod 600 /root/.ssh/id_rsa
    echo "✓ SSH私钥已配置"
fi

if [ -f /srv/shared/ssh-key/id_rsa.pub ]; then
    cp /srv/shared/ssh-key/id_rsa.pub /root/.ssh/id_rsa.pub
    chmod 644 /root/.ssh/id_rsa.pub
    
    # 添加到 authorized_keys 以允许从 slurm-master 访问
    cat /srv/shared/ssh-key/id_rsa.pub >> /root/.ssh/authorized_keys
    chmod 600 /root/.ssh/authorized_keys
    echo "✓ SSH公钥已配置"
fi

# 配置SSH客户端（禁用严格主机密钥检查，便于测试）
cat > /root/.ssh/config <<EOF
Host *
    StrictHostKeyChecking no
    UserKnownHostsFile=/dev/null
EOF
chmod 600 /root/.ssh/config

# 显示容器信息
echo "=========================================="
echo "测试容器启动完成"
echo "主机名: $(hostname)"
echo "时间: $(date)"
echo "SSH端口: 22"
echo "Root用户: root (密码: ${ROOT_PASSWORD:-rootpass123})"
echo "测试用户: testuser (密码: ${SSH_PASSWORD:-testpass123})"
echo "=========================================="

# 启动systemd (作为PID 1)
exec /sbin/init "$@"
