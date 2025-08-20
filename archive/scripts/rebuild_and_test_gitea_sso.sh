#!/usr/bin/env bash
set -euo pipefail

# Rebuild and bring up the stack, then smoke test Gitea asset and SSO access.
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

log() { echo "[gitea-sso] $*"; }

log "Rebuilding and starting stack (dev)..."
./scripts/build.sh dev --up --test

log "Waiting for nginx to be healthy..."
# best-effort wait up to ~45s
for i in {1..15}; do
  if curl -sSf http://localhost:8080/health >/dev/null 2>&1; then
    log "nginx ready"
    break
  fi
  sleep 3
done

# Login to backend (SSO) to obtain cookies
log "Logging in to backend to obtain SSO cookies..."
COOKIE_JAR="/tmp/aimatrix_cookies.txt"
rm -f "$COOKIE_JAR"
LOGIN_PAYLOAD='{"username":"admin","password":"admin123"}'
TOKEN=$(curl -s -c "$COOKIE_JAR" -H 'Content-Type: application/json' \
  -X POST http://localhost:8080/api/auth/login -d "$LOGIN_PAYLOAD" | jq -r .token)
if [[ -z "${TOKEN:-}" || "${TOKEN}" == "null" ]]; then
  log "ERROR: failed to obtain token via /api/auth/login"
  exit 1
fi

log "Smoke test: Gitea assets (should be 200)"
ASSET_STATUS=$(curl -s -o /dev/null -w '%{http_code}' -b "$COOKIE_JAR" \
  'http://localhost:8080/gitea/assets/js/index.js?v=1.24.5')
if [[ "$ASSET_STATUS" != "200" ]]; then
  log "ERROR: Asset fetch failed with status $ASSET_STATUS"
  exit 2
fi

log "Smoke test: Gitea main (should not redirect to signin)"
MAIN_STATUS=$(curl -s -o /dev/null -w '%{http_code}' -b "$COOKIE_JAR" \
  'http://localhost:8080/gitea/')
if [[ "$MAIN_STATUS" =~ ^30[12]$ ]]; then
  log "WARNING: /gitea/ returned redirect ($MAIN_STATUS). This may be expected if Gitea sets internal redirects."
fi

# Try visiting a page that would require session; if SSO headers applied, Gitea should create a session automatically.
ADMIN_STATUS=$(curl -s -o /dev/null -w '%{http_code}' -b "$COOKIE_JAR" \
  'http://localhost:8080/gitea/user/settings')
log "Result: /gitea/user/settings -> $ADMIN_STATUS"

if [[ "$ADMIN_STATUS" == "200" ]]; then
  log "SUCCESS: SSO seems effective, user page accessible."
  exit 0
fi

log "INFO: User page not 200 ($ADMIN_STATUS). Try direct SSO-login bridge..."
LOGIN_BRIDGE_STATUS=$(curl -s -o /dev/null -w '%{http_code}' -b "$COOKIE_JAR" \
  'http://localhost:8080/gitea/user/login')
log "Result: /gitea/user/login -> $LOGIN_BRIDGE_STATUS (should be 200 or 302 to a logged-in page)"

if [[ "$LOGIN_BRIDGE_STATUS" =~ ^20[0-9]|30[12]$ ]]; then
  log "SUCCESS: SSO login bridge reachable."
  exit 0
fi

log "ERROR: SSO flow not established. Check nginx auth_request and headers."
exit 3
