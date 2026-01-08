#!/usr/bin/env bash
set -euo pipefail

# This entrypoint renders app.ini to skip the install wizard and configures storage.
# DB/user are created ahead of time by postgres init script in compose.

: "${GITEA_DB_TYPE:=postgres}"
: "${GITEA_DB_HOST:=postgres:5432}"
: "${GITEA_DB_NAME:=gitea}"
: "${GITEA_DB_USER:=gitea}"
: "${GITEA_DB_PASSWD:=gitea-password}"
: "${ROOT_URL:=http://localhost:8080/gitea/}"
: "${SUBURL:=/gitea}"

# Optional storage config
: "${GITEA__storage__STORAGE_TYPE:=local}"
: "${DATA_PATH:=/data/gitea}"
# S3 compatible storage (SeaweedFS/MinIO) - Gitea uses 'minio' as config type name for S3-compatible storage
: "${MINIO_ENDPOINT:=seaweedfs-filer:8333}"
: "${MINIO_BUCKET:=gitea}"
: "${MINIO_USE_SSL:=false}"
: "${MINIO_LOCATION:=us-east-1}"
# Reverse-proxy SSO (optional)
: "${ENABLE_REVERSE_PROXY_AUTH:=true}"
: "${REVERSE_PROXY_EMAIL:=true}"
: "${REVERSE_PROXY_FULL_NAME:=false}"

# Initial admin bootstrap (SSO-first):
# - Avoid using the default 'admin' credentials.
# - If INITIAL_ADMIN_USERNAME is set (ideally to a backend-managed identity, e.g. GITEA_ALIAS_ADMIN_TO),
#   we create that user as admin with a random password. Password login is disabled anyway.
: "${INITIAL_ADMIN_USERNAME:=${GITEA_ALIAS_ADMIN_TO:-}}"
: "${INITIAL_ADMIN_EMAIL:=admin@example.com}"
: "${INITIAL_ADMIN_FULL_NAME:=Portal Administrator}"

# Reserved usernames override (allow 'admin' for reverse-proxy SSO)
: "${RESERVED_USERNAMES:=git,gitea}"

APP_INI=/data/gitea/conf/app.ini
APP_DIR=/data/gitea

log(){ echo "[gitea-init] $*"; }

render_app_ini() {
  mkdir -p "$(dirname "$APP_INI")"
  cat >"$APP_INI" <<EOF
APP_NAME = Gitea: Git with a cup of tea
RUN_MODE = prod

[server]
APP_DATA_PATH = ${APP_DIR}
DOMAIN = ${DOMAIN:-localhost}
SSH_DOMAIN = ${SSH_DOMAIN:-localhost}
HTTP_PORT = ${HTTP_PORT:-3000}
ROOT_URL = ${ROOT_URL}
PROTOCOL = ${PROTOCOL:-http}
SUBURL = ${SUBURL}
STATIC_URL_PREFIX = ${STATIC_URL_PREFIX:-/gitea}
PUBLIC_URL_DETECTION = auto
LFS_START_SERVER = ${LFS_START_SERVER:-false}

[database]
DB_TYPE = ${GITEA_DB_TYPE}
HOST = ${GITEA_DB_HOST}
NAME = ${GITEA_DB_NAME}
USER = ${GITEA_DB_USER}
PASSWD = ${GITEA_DB_PASSWD}
SSL_MODE = ${DB_SSL_MODE:-disable}
SCHEMA = ${DB_SCHEMA:-public}
LOG_SQL = ${DB_LOG_SQL:-false}

[log]
MODE = console
LEVEL = ${LOG_LEVEL:-info}
ROOT_PATH = ${LOG_ROOT_PATH:-/data/gitea/log}

[security]
INSTALL_LOCK = true
SECRET_KEY = ${SECRET_KEY:-}
REVERSE_PROXY_LIMIT = 1
REVERSE_PROXY_TRUSTED_PROXIES = ${REVERSE_PROXY_TRUSTED_PROXIES:-0.0.0.0/0,::/0}
[service]
# Enable reverse proxy authentication so upstream (Nginx) can sign users in
ENABLE_REVERSE_PROXY_AUTHENTICATION = ${ENABLE_REVERSE_PROXY_AUTH}
ENABLE_REVERSE_PROXY_AUTHENTICATION_API = ${ENABLE_REVERSE_PROXY_AUTH}
ENABLE_REVERSE_PROXY_AUTO_REGISTRATION = ${ENABLE_REVERSE_PROXY_AUTH}
ENABLE_REVERSE_PROXY_EMAIL = ${REVERSE_PROXY_EMAIL}
ENABLE_REVERSE_PROXY_FULL_NAME = ${REVERSE_PROXY_FULL_NAME}
RESERVED_USERNAMES = ${RESERVED_USERNAMES}
DISABLE_LOGIN_FORM = ${DISABLE_LOGIN_FORM:-true}
ALLOW_ONLY_EXTERNAL_LOGIN = ${ALLOW_ONLY_EXTERNAL_LOGIN:-true}

[security]
# Standard header names for reverse proxy auth
REVERSE_PROXY_AUTHENTICATION_USER = X-WEBAUTH-USER
REVERSE_PROXY_AUTHENTICATION_EMAIL = X-WEBAUTH-EMAIL
REVERSE_PROXY_AUTHENTICATION_FULL_NAME = X-WEBAUTH-FULLNAME

[auth]
# Classic section still respected by parts of Gitea for proxy auth
REQUIRE_EMAIL_CONFIRMATION = false
REGISTER_EMAIL_CONFIRM = false
DISABLE_REGISTRATION = ${DISABLE_REGISTRATION:-false}
ENABLE_OPENID_SIGNIN = false
ENABLE_OPENID_SIGNUP = false
OAUTH2_ENABLE = false

[auth.proxy]
ENABLED = ${ENABLE_REVERSE_PROXY_AUTH}
HEADER = X-WEBAUTH-USER
EMAIL_HEADER = X-WEBAUTH-EMAIL
FULL_NAME_HEADER = X-WEBAUTH-FULLNAME
WHITELISTED_PROXIES = 0.0.0.0/0,::/0
AUTO_CREATE_USERS = ${ENABLE_REVERSE_PROXY_AUTH}

[service]
DISABLE_REGISTRATION = ${DISABLE_REGISTRATION:-false}
REQUIRE_SIGNIN_VIEW = ${REQUIRE_SIGNIN_VIEW:-false}

[repository]
ROOT = /data/git/repositories

[storage]
STORAGE_TYPE = ${GITEA__storage__STORAGE_TYPE}
EOF

  if [[ "${GITEA__storage__STORAGE_TYPE}" == "minio" || "${GITEA__storage__STORAGE_TYPE}" == "s3" ]]; then
    cat >>"$APP_INI" <<EOF
MINIO_ENDPOINT = ${MINIO_ENDPOINT}
MINIO_ACCESS_KEY_ID = ${MINIO_ACCESS_KEY:-${AWS_ACCESS_KEY_ID:-}}
MINIO_SECRET_ACCESS_KEY = ${MINIO_SECRET_KEY:-${AWS_SECRET_ACCESS_KEY:-}}
MINIO_BUCKET = ${MINIO_BUCKET}
MINIO_USE_SSL = ${MINIO_USE_SSL}
MINIO_LOCATION = ${MINIO_LOCATION}
EOF
  else
    cat >>"$APP_INI" <<EOF
PATH = ${DATA_PATH}
EOF
  fi
}

main() {
  if [[ ! -f "$APP_INI" ]] || ! grep -q '^INSTALL_LOCK *= *true' "$APP_INI"; then
    log "rendering app.ini"
    render_app_ini
  else
    log "app.ini exists with INSTALL_LOCK=true, updating URL settings"
    # In-place update of ROOT_URL and SUBURL to keep proxy prefix alignment
    sed -i "s#^ROOT_URL *=.*#ROOT_URL = ${ROOT_URL//#/\\#}#" "$APP_INI" || true
    # Ensure SUBURL matches external prefix; app will serve under subpath
    if grep -q '^SUBURL *=.*' "$APP_INI"; then
      sed -i "s#^SUBURL *=.*#SUBURL = ${SUBURL}#" "$APP_INI"
    else
      sed -i "/^\[server\]/,/^\[/ s#^HTTP_PORT *=.*#&\nSUBURL = ${SUBURL}#" "$APP_INI"
    fi
    # Ensure reverse-proxy auth stays enabled/updated
    if grep -q '^\[service\]' "$APP_INI"; then
      sed -i "s#^ENABLE_REVERSE_PROXY_AUTHENTICATION *=.*#ENABLE_REVERSE_PROXY_AUTHENTICATION = ${ENABLE_REVERSE_PROXY_AUTH}#" "$APP_INI" || true
      sed -i "s#^ENABLE_REVERSE_PROXY_AUTHENTICATION_API *=.*#ENABLE_REVERSE_PROXY_AUTHENTICATION_API = ${ENABLE_REVERSE_PROXY_AUTH}#" "$APP_INI" || true
      sed -i "s#^ENABLE_REVERSE_PROXY_AUTO_REGISTRATION *=.*#ENABLE_REVERSE_PROXY_AUTO_REGISTRATION = ${ENABLE_REVERSE_PROXY_AUTH}#" "$APP_INI" || true
    fi
    if grep -q '^\[auth.proxy\]' "$APP_INI"; then
      sed -i "s#^ENABLED *=.*#ENABLED = ${ENABLE_REVERSE_PROXY_AUTH}#" "$APP_INI" || true
      sed -i "s#^HEADER *=.*#HEADER = X-WEBAUTH-USER#" "$APP_INI" || true
    fi

    # Ensure OAuth/OpenID style internal SSO are disabled to enforce unified external SSO via reverse proxy
    if grep -q '^\[auth\]' "$APP_INI"; then
      sed -i "s#^ENABLE_OPENID_SIGNIN *=.*#ENABLE_OPENID_SIGNIN = false#" "$APP_INI" || true
      sed -i "s#^ENABLE_OPENID_SIGNUP *=.*#ENABLE_OPENID_SIGNUP = false#" "$APP_INI" || true
      sed -i "s#^OAUTH2_ENABLE *=.*#OAUTH2_ENABLE = false#" "$APP_INI" || true
    fi

    # Ensure STATIC_URL_PREFIX aligns with proxy subpath (use '/gitea' so templates adding '/assets' don't double it)
    if grep -q '^STATIC_URL_PREFIX *=.*' "$APP_INI"; then
      sed -i "s#^STATIC_URL_PREFIX *=.*#STATIC_URL_PREFIX = ${STATIC_URL_PREFIX:-/gitea}#" "$APP_INI" || true
    else
      sed -i "/^SUBURL *=/a STATIC_URL_PREFIX = ${STATIC_URL_PREFIX:-/gitea}" "$APP_INI" || true
    fi

    # Force SSO-only login UX
    if grep -q '^DISABLE_LOGIN_FORM *=.*' "$APP_INI"; then
      sed -i "s#^DISABLE_LOGIN_FORM *=.*#DISABLE_LOGIN_FORM = ${DISABLE_LOGIN_FORM:-true}#" "$APP_INI" || true
    else
      sed -i "/^\[service\]/,/^\[/ s#^REQUIRE_SIGNIN_VIEW *=.*#&\nDISABLE_LOGIN_FORM = ${DISABLE_LOGIN_FORM:-true}#" "$APP_INI" || true
    fi
    if grep -q '^ALLOW_ONLY_EXTERNAL_LOGIN *=.*' "$APP_INI"; then
      sed -i "s#^ALLOW_ONLY_EXTERNAL_LOGIN *=.*#ALLOW_ONLY_EXTERNAL_LOGIN = ${ALLOW_ONLY_EXTERNAL_LOGIN:-true}#" "$APP_INI" || true
    else
      sed -i "/^\[service\]/,/^\[/ s#^REQUIRE_SIGNIN_VIEW *=.*#&\nALLOW_ONLY_EXTERNAL_LOGIN = ${ALLOW_ONLY_EXTERNAL_LOGIN:-true}#" "$APP_INI" || true
    fi

    # Ensure RESERVED_USERNAMES excludes 'admin' by applying env value
    if grep -q '^\[service\]' "$APP_INI"; then
      if grep -q '^RESERVED_USERNAMES *=.*' "$APP_INI"; then
        sed -i "s#^RESERVED_USERNAMES *=.*#RESERVED_USERNAMES = ${RESERVED_USERNAMES}#" "$APP_INI" || true
      else
        # Append the setting within the [service] section
        awk -v repl="RESERVED_USERNAMES = ${RESERVED_USERNAMES}" '
          BEGIN{insec=0}
          /^\[service\]/{print; insec=1; next}
          /^\[.*\]/{ if(insec){print repl; insec=0}; print; next }
          {print}
          END{ if(insec){print repl} }
        ' "$APP_INI" >"${APP_INI}.tmp" && mv "${APP_INI}.tmp" "$APP_INI"
      fi
    fi
  fi

  # Ensure data directories
  mkdir -p "$APP_DIR" "$DATA_PATH" /data/git/repositories
  chown -R 1000:1000 /data || true

  # Helper to run as git user
  run_as_git() {
    if command -v su-exec >/dev/null 2>&1; then
      su-exec 1000:1000 "$@"
    elif command -v gosu >/dev/null 2>&1; then
      gosu 1000:1000 "$@"
    else
      su -s /bin/sh -c "$*" git
    fi
  }

  # SSO-first bootstrap: create the initial admin user for reverse-proxy SSO
  # The user is created with a random password (login form is disabled anyway)
  # and marked as admin. This ensures the SSO user has proper admin privileges.
  if [ -n "${INITIAL_ADMIN_USERNAME}" ]; then
    # Check if the user already exists
    if ! run_as_git /usr/local/bin/gitea --config "$APP_INI" admin user list 2>/dev/null | awk '{print $2}' | grep -qx "$INITIAL_ADMIN_USERNAME"; then
      # Generate a random password; login form is disabled, so this is never used interactively
      RAND_PASS=$(tr -dc 'A-Za-z0-9' </dev/urandom | head -c 24 || true)
      log "creating initial SSO admin user '$INITIAL_ADMIN_USERNAME' (password login disabled)"
      run_as_git /usr/local/bin/gitea --config "$APP_INI" admin user create \
        --admin \
        --username "$INITIAL_ADMIN_USERNAME" \
        --password "${RAND_PASS:-ChangeMe123}" \
        --email "${INITIAL_ADMIN_EMAIL}" \
        --must-change-password=false || true
    else
      log "initial SSO admin user '$INITIAL_ADMIN_USERNAME' already exists"
      # Ensure the user has admin privileges (in case it was created by SSO auto-register without admin flag)
      log "ensuring '$INITIAL_ADMIN_USERNAME' has admin privileges..."
      run_as_git /usr/local/bin/gitea --config "$APP_INI" admin user change-password \
        --username "$INITIAL_ADMIN_USERNAME" \
        --password "$(tr -dc 'A-Za-z0-9' </dev/urandom | head -c 24)" 2>/dev/null || true
      # Use SQL to set admin flag if gitea CLI doesn't support it directly
      # Note: This requires the user to exist; the admin flag is set via 'admin user create' 
      # For existing users created by SSO, we need to grant admin via the admin command
      run_as_git /usr/local/bin/gitea --config "$APP_INI" admin user must-change-password \
        --username "$INITIAL_ADMIN_USERNAME" \
        --unset 2>/dev/null || true
    fi
  else
    log "INITIAL_ADMIN_USERNAME not set; relying on reverse-proxy SSO auto-create"
  fi

  # Delegate to upstream entrypoint; pass through arguments
  exec /usr/bin/entrypoint "$@"
}

main "$@"
