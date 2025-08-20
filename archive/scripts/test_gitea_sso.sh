#!/bin/bash
# 自动重建并测试 Gitea SSO 登录态同步
# 用于修复 SSO 后访问 Gitea 仍需输入密码的问题

set -e

# 1. 重新构建所有服务（包括nginx/backend/gitea等）
./scripts/build.sh dev --up --test

# 2. 检查 SSO 登录态是否同步到 Gitea
# 访问 SSO 登录页，获取 token 并访问 Gitea
curl -s -c /tmp/cookie_jar -X POST http://localhost:8080/api/auth/login -H "Content-Type: application/json" -d '{"username":"admin","password":"admin123"}' | jq -r '.token' > /tmp/sso_token

# 3. 用 SSO token 访问 Gitea，检查是否跳转到登录页（如果跳转说明未同步）
GITEA_RESPONSE=$(curl -s -b /tmp/cookie_jar -L http://localhost:8080/gitea/)

if echo "$GITEA_RESPONSE" | grep -q "password"; then
  echo "[FAIL] SSO 登录态未同步到 Gitea，仍需输入密码。请检查nginx和backend配置。"
else
  echo "[OK] SSO 登录态已同步到 Gitea，无需输入密码。"
fi
