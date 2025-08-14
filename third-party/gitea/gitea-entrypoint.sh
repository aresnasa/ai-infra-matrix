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
: "${MINIO_ENDPOINT:=minio:9000}"
: "${MINIO_BUCKET:=gitea}"
: "${MINIO_USE_SSL:=false}"
: "${MINIO_LOCATION:=us-east-1}"

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
    # Ensure SUBURL line exists and is updated
    if grep -q '^SUBURL *=.*' "$APP_INI"; then
      sed -i "s#^SUBURL *=.*#SUBURL = ${SUBURL}#" "$APP_INI"
    else
      sed -i "/^\[server\]/,/^\[/ s#^HTTP_PORT *=.*#&\nSUBURL = ${SUBURL}#" "$APP_INI"
    fi
  fi

  # Ensure data directories
  mkdir -p "$APP_DIR" "$DATA_PATH" /data/git/repositories
  chown -R 1000:1000 /data || true

  # Delegate to upstream entrypoint; pass through arguments
  exec /usr/bin/entrypoint "$@"
}

main "$@"
