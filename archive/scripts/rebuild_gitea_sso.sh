#!/usr/bin/env bash
set -euo pipefail

# Rebuild and bring up stack, then run SSO+Gitea smoke tests
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

# 1) Rebuild and run all with tests, as requested
./scripts/build.sh dev --up --test

# 2) Basic smoke tests via nginx
sleep 3

red() { printf "\033[31m%s\033[0m\n" "$*"; }
green() { printf "\033[32m%s\033[0m\n" "$*"; }
yellow() { printf "\033[33m%s\033[0m\n" "$*"; }

check_url() {
  local url="$1"; shift
  local expect="$1"; shift
  local code
  code=$(curl -s -o /dev/null -w "%{http_code}" "$url" || true)
  if [[ "$code" == "$expect" ]]; then
    green "[OK] $url -> $code"
  else
    red   "[FAIL] $url -> $code (expect $expect)"
    return 1
  fi
}

# 2.1) Nginx health
check_url "http://localhost:8080/health" 200 || true

# 2.2) Backend health (via nginx)
check_url "http://localhost:8080/api/health" 200 || true

# 2.3) Gitea public assets must be accessible without auth
check_url "http://localhost:8080/gitea/assets/js/index.js?v=1.24.5" 200 || true
check_url "http://localhost:8080/gitea/assets/css/index.css?v=1.24.5" 200 || true

# 2.4) SSO bridge should be present
check_url "http://localhost:8080/sso/" 200 || true

yellow "Done. If assets still fail, inspect nginx logs inside ai-infra-nginx: /var/log/nginx/gitea_access.log and auth_request.log"
