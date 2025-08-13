#!/usr/bin/env bash
set -euo pipefail

# Simple health probe script for docker-compose stack
# Probes:
#  - Frontend via Nginx:        http://localhost:8080/
#  - Backend via Nginx:         http://localhost:8080/api/health
#  - Backend internal (exec):   curl inside backend container to :8082
#  - JupyterHub via Nginx:      http://localhost:8080/jupyter/hub/health
#  - JupyterHub direct:         http://localhost:8088/jupyter/hub/health

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

failures=0

probe() {
  local name="$1" url="$2"
  local code
  code=$(curl -s -o /dev/null -w '%{http_code}' "$url" || true)
  if [[ "$code" =~ ^2[0-9][0-9]$ ]]; then
    printf "${GREEN}PASS${NC} %-36s => %s\n" "$name" "$code"
  else
    printf "${RED}FAIL${NC} %-36s => %s\n" "$name" "$code"
    failures=$((failures+1))
  fi
}

echo "Running health checks..."

probe "frontend /" "http://localhost:8080/"
probe "backend via nginx /api/health" "http://localhost:8080/api/health"

# Backend internal check (container scope)
if docker compose ps backend >/dev/null 2>&1 || docker-compose ps backend >/dev/null 2>&1; then
  if docker compose exec -T backend curl -s -o /dev/null -w '%{http_code}' http://localhost:8082/api/health >/tmp/backend_code 2>/dev/null || \
     docker-compose exec -T backend curl -s -o /dev/null -w '%{http_code}' http://localhost:8082/api/health >/tmp/backend_code 2>/dev/null; then
    code=$(cat /tmp/backend_code || echo "000")
    if [[ "$code" =~ ^2[0-9][0-9]$ ]]; then
      printf "${GREEN}PASS${NC} %-36s => %s\n" "backend internal 8082 /api/health" "$code"
    else
      printf "${RED}FAIL${NC} %-36s => %s\n" "backend internal 8082 /api/health" "$code"
      failures=$((failures+1))
    fi
    rm -f /tmp/backend_code || true
  else
    printf "${YELLOW}SKIP${NC} %-36s (exec failed)\n" "backend internal 8082 /api/health"
  fi
else
  printf "${YELLOW}SKIP${NC} %-36s (service not found)\n" "backend internal 8082 /api/health"
fi

probe "jupyterhub via nginx /jupyter/hub/health" "http://localhost:8080/jupyter/hub/health"
probe "jupyterhub direct 8088 /jupyter/hub/health" "http://localhost:8088/jupyter/hub/health"

# Optional static asset check to ensure path prefixing works
probe "jupyter static via nginx /jupyter/hub/static/favicon.ico" "http://localhost:8080/jupyter/hub/static/favicon.ico"

if [[ $failures -eq 0 ]]; then
  echo -e "${GREEN}All health checks passed.${NC}"
  exit 0
else
  echo -e "${RED}$failures check(s) failed.${NC}"
  exit 1
fi
