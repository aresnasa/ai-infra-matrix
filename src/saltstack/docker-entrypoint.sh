#!/bin/bash

set -e

log() {
    echo "$1"
}

log "🧂 AI基础设施矩阵 - SaltStack服务启动中..."
log "📅 启动时间: $(date)"
log "🏷️ 版本: ${APP_VERSION:-dev}"
log "🎯 Master ID: ${SALT_MASTER_ID:-default}"
log "🔀 Master 角色: ${SALT_MASTER_ROLE:-standalone}"

ensure_directories() {
    mkdir -p /var/log/salt
    mkdir -p /var/cache/salt/master
    mkdir -p /var/cache/salt/minion
    mkdir -p /var/run/salt
    mkdir -p /etc/salt/pki/master
    mkdir -p /etc/salt/pki/minion
    mkdir -p /srv/salt
    mkdir -p /srv/pillar
}

# 生成或等待 Master 密钥（多 Master 高可用核心逻辑）
setup_master_keys() {
    local role="${SALT_MASTER_ROLE:-standalone}"
    local master_id="${SALT_MASTER_ID:-master}"
    local pki_dir="/etc/salt/pki/master"
    local max_wait=60
    local wait_interval=2
    
    log "🔐 设置 Master 密钥 (角色: $role)..."
    
    if [ "$role" = "primary" ] || [ "$role" = "standalone" ]; then
        # 主节点：如果没有密钥则生成
        if [ ! -f "$pki_dir/master.pem" ]; then
            log "🔑 [Primary] 生成新的 Master 密钥对..."
            salt-key --gen-keys=master --gen-keys-dir="$pki_dir/"
            chmod 400 "$pki_dir/master.pem"
            chmod 644 "$pki_dir/master.pub"
            log "✅ Master 密钥生成完成"
        else
            log "♻️ [Primary] 使用现有的 Master 密钥"
        fi
    elif [ "$role" = "secondary" ]; then
        # 备用节点：等待主节点生成密钥
        log "⏳ [Secondary] 等待主节点生成 PKI 密钥..."
        local waited=0
        while [ ! -f "$pki_dir/master.pem" ] || [ ! -f "$pki_dir/master.pub" ]; do
            if [ $waited -ge $max_wait ]; then
                log "❌ [Secondary] 等待主节点密钥超时 (${max_wait}s)"
                exit 1
            fi
            sleep $wait_interval
            waited=$((waited + wait_interval))
            log "   等待中... ($waited/${max_wait}s)"
        done
        log "✅ [Secondary] 检测到 PKI 密钥，继续启动"
        
        # 确保文件权限正确
        chmod 400 "$pki_dir/master.pem" 2>/dev/null || true
        chmod 644 "$pki_dir/master.pub" 2>/dev/null || true
    fi
    
    # 显示密钥指纹（用于验证多 Master 密钥一致性）
    if [ -f "$pki_dir/master.pub" ]; then
        local fingerprint=$(salt-key --finger-all 2>/dev/null | head -n5 || echo "无法获取指纹")
        log "🔏 Master 公钥指纹: $(md5sum $pki_dir/master.pub | cut -d' ' -f1)"
    fi
}

# 配置多 Master 相关设置
configure_multi_master() {
    local master_id="${SALT_MASTER_ID:-master}"
    local master_conf="/etc/salt/master.d/master.conf"
    
    # 添加 Master ID 标识
    if ! grep -q "^id:" "$master_conf" 2>/dev/null; then
        echo "" >> "$master_conf"
        echo "# Multi-Master Configuration" >> "$master_conf"
        echo "id: $master_id" >> "$master_conf"
        log "📝 已配置 Master ID: $master_id"
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
        setup_master_keys
        configure_multi_master
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
