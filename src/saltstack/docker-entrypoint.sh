#!/bin/bash

set -e

log() {
    echo "$1"
}

log "🧂 AI基础设施矩阵 - SaltStack服务启动中..."
log "📅 启动时间: $(date)"
log "🏷️ 版本: ${APP_VERSION:-dev}"

ensure_directories() {
    mkdir -p /var/log/salt
    mkdir -p /var/cache/salt/master
    mkdir -p /var/cache/salt/minion
    mkdir -p /var/run/salt
    mkdir -p /etc/salt/pki/master
    mkdir -p /etc/salt/pki/minion
}

generate_master_keys() {
    if [ ! -f /etc/salt/pki/master/master.pem ]; then
        log "🔐 生成Salt Master密钥..."
        salt-key --gen-keys=master --gen-keys-dir=/etc/salt/pki/master/
    fi
}

verify_configs() {
    log "🔧 验证Salt配置文件..."
    if [ ! -r /etc/salt/master.d/master.conf ]; then
        log "❌ Salt Master配置文件不可读"
        exit 1
    fi

    if [ ! -r /etc/salt/minion.d/minion.conf ]; then
        log "❌ Salt Minion配置文件不可读"
        exit 1
    fi

    log "✅ Salt配置文件验证通过"
}

configure_eauth() {
    SALT_API_USER=${SALT_API_USERNAME:-saltapi}
    SALT_API_PASS=${SALT_API_PASSWORD:-}
    EAUTH_FILE=/etc/salt/master.d/eauth.txt

    if [ -n "$SALT_API_PASS" ]; then
        log "🔐 生成Salt API文件认证凭据 ($EAUTH_FILE)"
        mkdir -p "$(dirname "$EAUTH_FILE")"
        echo "${SALT_API_USER}:${SALT_API_PASS}" > "$EAUTH_FILE"
        chmod 600 "$EAUTH_FILE" || true
    else
        log "⚠️ 未设置 SALT_API_PASSWORD，Salt API 将允许匿名访问公开端点，但需要认证的runner/wheel可能失败"
    fi
}

create_systemd_units() {
    log "🛠️ 写入systemd单元文件..."

    cat <<'EOF' >/etc/systemd/system/salt-master.service
[Unit]
Description=Salt Master Service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/salt-master --log-level=info
Restart=on-failure
RestartSec=5s
LimitNOFILE=65535
KillMode=process

[Install]
WantedBy=multi-user.target
EOF

    cat <<'EOF' >/etc/systemd/system/salt-api.service
[Unit]
Description=Salt API Service
After=salt-master.service
Requires=salt-master.service

[Service]
Type=simple
ExecStart=/usr/local/bin/salt-api --log-level=info
Restart=on-failure
RestartSec=5s
KillMode=process

[Install]
WantedBy=multi-user.target
EOF

    cat <<'EOF' >/etc/systemd/system/salt-minion-local.service
[Unit]
Description=Local Salt Minion for development
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/salt-minion --log-level=info
Restart=on-failure
RestartSec=5s
KillMode=process

[Install]
WantedBy=multi-user.target
EOF
}

enable_systemd_services() {
    mkdir -p /etc/systemd/system/multi-user.target.wants

    ln -sf /etc/systemd/system/salt-master.service /etc/systemd/system/multi-user.target.wants/salt-master.service
    ln -sf /etc/systemd/system/salt-api.service /etc/systemd/system/multi-user.target.wants/salt-api.service

    if [ "${START_LOCAL_MINION:-true}" = "true" ]; then
        ln -sf /etc/systemd/system/salt-minion-local.service /etc/systemd/system/multi-user.target.wants/salt-minion-local.service
    else
        rm -f /etc/systemd/system/multi-user.target.wants/salt-minion-local.service
        log "ℹ️ 已禁用本地Salt Minion (START_LOCAL_MINION=false)"
    fi
}

start_systemd() {
    log "✅ SaltStack服务准备完成"
    log "🚀 支持功能:"
    log "   ✅ Salt Master (端口: 4505, 4506)"
    log "   ✅ Salt API (端口: 8002)"
    log "   ✅ AI-Infra-Matrix SSO集成"
    log "   ✅ 自动密钥管理"

    exec /sbin/init
}

case "${1:-start-services}" in
    start-services)
        ensure_directories
        generate_master_keys
        verify_configs
        configure_eauth
        create_systemd_units
        enable_systemd_services
        start_systemd
        ;;
    *)
        exec "$@"
        ;;
esac
