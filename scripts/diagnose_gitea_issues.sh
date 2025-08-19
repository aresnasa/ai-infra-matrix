#!/usr/bin/env bash
set -euo pipefail

# Quick diagnosis for Gitea assets and SSO via nginx
ROOT=${ROOT:-http://localhost:8080}
ASSET_JS="$ROOT/gitea/assets/js/index.js?v=1.24.5"
ASSET_CSS="$ROOT/gitea/assets/css/index.css?v=1.24.5"
GITEA_ROOT="$ROOT/gitea/"
HEALTH_URL="$ROOT/health"

red(){ printf "\033[31m%s\033[0m\n" "${*}"; }
green(){ printf "\033[32m%s\033[0m\n" "${*}"; }
yellow(){ printf "\033[33m%s\033[0m\n" "${*}"; }

check_url(){
  local url="$1"
  local expect="${2:-200}"
  local code
  code=$(curl -s -o /dev/null -w "%{http_code}" "$url" || true)
  if [[ "$code" == "$expect" ]]; then
    green "[OK] $url -> $code"
  else
    red   "[FAIL] $url -> $code (expect $expect)"
  fi
}

main(){
  echo "== Basic checks =="
  check_url "$HEALTH_URL" 200

  echo "\n== Gitea asset checks =="
  check_url "$ASSET_JS" 200
  check_url "$ASSET_CSS" 200

  echo "\n== Gitea root (should be 200/302) =="
  curl -sS -o /dev/null -w "Status: %{http_code}\nLocation: %{redirect_url}\n" "$GITEA_ROOT" || true

  echo "\n== Nginx logs hint =="
  echo "Inspect inside container ai-infra-nginx:"
  echo "  /var/log/nginx/gitea_access.log (assets/auth)"
  echo "  /var/log/nginx/auth_request.log (SSO)"
  echo "If Docker is available, run: docker logs --tail=200 ai-infra-nginx"
}

main "$@"

