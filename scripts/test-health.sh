#!/usr/bin/env bash
set -euo pipefail

# Unified health checks for the docker-compose stack
# Exposes reusable test functions. When executed directly, runs run_all.
# Targets:
#  - Frontend via Nginx:        http://localhost:8080/
#  - Backend via Nginx:         http://localhost:8080/api/health
#  - Backend internal (exec):   curl inside backend to :8082
#  - JupyterHub via Nginx:      http://localhost:8080/jupyter/hub/health
#  - JupyterHub direct:         http://localhost:8088/jupyter/hub/health
#  - Jupyter static asset:      /jupyter/hub/static/favicon.ico
#  - Jupyter SPA entry:         /jupyter
#  - Frontend favicon:          /favicon.svg
#  - Gitea via Nginx:           /gitea/
#  - Optional SSO auth checks:  /debug/verify, /gitea/ with Authorization

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

failures=0
passes=0

# Generic probe (no auth)
probe() {
  local name="$1" url="$2" expected_regex="${3:-^2[0-9][0-9]$}"
  local code
  code=$(curl -s -o /dev/null -w '%{http_code}' "$url" || true)
  if [[ "$code" =~ $expected_regex ]]; then
    printf "${GREEN}PASS${NC} %-40s => %s\n" "$name" "$code"; passes=$((passes+1))
  else
    printf "${RED}FAIL${NC} %-40s => %s\n" "$name" "$code"; failures=$((failures+1))
  fi
}

# Probe with Authorization header if TOKEN provided
probe_auth() {
  local name="$1" url="$2" expected_regex="${3:-^2[0-9][0-9]$}"
  local token="${TOKEN:-${AI_INFRA_TOKEN:-${JWT_TOKEN:-${AUTH_TOKEN:-}}}}"
  if [[ -z "${token}" ]]; then
    printf "${YELLOW}SKIP${NC} %-40s (no TOKEN provided)\n" "$name"
    return 0
  fi
  local code
  code=$(curl -s -H "Authorization: Bearer ${token}" -o /dev/null -w '%{http_code}' "$url" || true)
  if [[ "$code" =~ $expected_regex ]]; then
    printf "${GREEN}PASS${NC} %-40s => %s\n" "$name" "$code"; passes=$((passes+1))
  else
    printf "${RED}FAIL${NC} %-40s => %s\n" "$name" "$code"; failures=$((failures+1))
  fi
}

# Individual test functions (can be sourced and called elsewhere)
test_frontend_root() { probe "frontend /" "http://localhost:8080/"; }
test_backend_via_nginx() { probe "backend via nginx /api/health" "http://localhost:8080/api/health"; }

test_backend_internal() {
  local label="backend internal 8082 /api/health"
  if docker compose ps backend >/dev/null 2>&1 || docker-compose ps backend >/dev/null 2>&1; then
    if docker compose exec -T backend curl -s -o /dev/null -w '%{http_code}' http://localhost:8082/api/health >/tmp/backend_code 2>/dev/null || \
       docker-compose exec -T backend curl -s -o /dev/null -w '%{http_code}' http://localhost:8082/api/health >/tmp/backend_code 2>/dev/null; then
      local code
      code=$(cat /tmp/backend_code || echo "000")
      rm -f /tmp/backend_code || true
      if [[ "$code" =~ ^2[0-9][0-9]$ ]]; then
        printf "${GREEN}PASS${NC} %-40s => %s\n" "$label" "$code"; passes=$((passes+1))
      else
        printf "${RED}FAIL${NC} %-40s => %s\n" "$label" "$code"; failures=$((failures+1))
      fi
    else
      printf "${YELLOW}SKIP${NC} %-40s (exec failed)\n" "$label"
    fi
  else
    printf "${YELLOW}SKIP${NC} %-40s (service not found)\n" "$label"
  fi
}

test_jhub_via_nginx() { probe "jupyterhub via nginx /jupyter/hub/health" "http://localhost:8080/jupyter/hub/health"; }
test_jhub_direct() { probe "jupyterhub direct 8088 /jupyter/hub/health" "http://localhost:8088/jupyter/hub/health"; }
test_jhub_static() { probe "jupyter static via nginx /jupyter/hub/static/favicon.ico" "http://localhost:8080/jupyter/hub/static/favicon.ico"; }
test_jupyter_spa_entry() { probe "embedded jupyter route" "http://localhost:8080/jupyter" "^2[0-9][0-9]$|^30[12]$"; }
test_frontend_favicon() { probe "frontend favicon.svg" "http://localhost:8080/favicon.svg"; }
test_gitea_basic() { probe "gitea via nginx /gitea/" "http://localhost:8080/gitea/" "^2[0-9][0-9]$|^30[12]$"; }

# Optional SSO checks (require TOKEN env var)
test_auth_debug_verify() { probe_auth "auth debug /debug/verify" "http://localhost:8080/debug/verify"; }
test_gitea_sso_auth() { probe_auth "gitea with Authorization header" "http://localhost:8080/gitea/" "^2[0-9][0-9]$|^30[12]$"; }

# Run all default checks
run_all() {
  echo "Running health checks..."
  test_frontend_root
  test_backend_via_nginx
  test_backend_internal
  test_jhub_via_nginx
  test_jhub_direct
  test_jhub_static
  test_jupyter_spa_entry
  test_frontend_favicon
  test_gitea_basic
  # Optional SSO-related checks (no-op if TOKEN unset)
  test_auth_debug_verify
  test_gitea_sso_auth

  if [[ $failures -eq 0 ]]; then
    echo -e "${GREEN}All health checks passed. (${passes} passed)${NC}"
    return 0
  else
    echo -e "${RED}$failures check(s) failed, ${passes} passed.${NC}"
    return 1
  fi
}

# If executed (not sourced), run all
if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  run_all
fi
