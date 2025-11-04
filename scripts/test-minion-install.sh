#!/bin/bash
# =============================================================================
# Test SaltStack Minion Installation on Test Containers
# =============================================================================

set -e

APPHUB_URL="${APPHUB_URL:-http://192.168.18.154:53434}"
SALT_VERSION="${SALT_VERSION:-3007.8}"
MASTER_HOST="${MASTER_HOST:-192.168.18.154}"

echo "=========================================="
echo "Testing SaltStack Minion Installation"
echo "=========================================="
echo "AppHub URL: ${APPHUB_URL}"
echo "Salt Version: ${SALT_VERSION}"
echo "Master Host: ${MASTER_HOST}"
echo ""

# Test Ubuntu container (test-ssh01)
echo "----------------------------------------"
echo "Testing Ubuntu container (test-ssh01)"
echo "----------------------------------------"
docker cp src/backend/scripts/install-salt-minion-deb.sh test-ssh01:/tmp/
docker exec test-ssh01 bash -c "chmod +x /tmp/install-salt-minion-deb.sh && /tmp/install-salt-minion-deb.sh ${APPHUB_URL} ${SALT_VERSION} ${MASTER_HOST} test-ssh01"

echo ""
echo "Verifying installation on test-ssh01..."
docker exec test-ssh01 salt-minion --version
docker exec test-ssh01 systemctl status salt-minion --no-pager || true

echo ""
echo "----------------------------------------"
echo "Testing Rocky Linux container (test-rocky01)"
echo "----------------------------------------"
docker cp src/backend/scripts/install-salt-minion-rpm.sh test-rocky01:/tmp/
docker exec test-rocky01 bash -c "chmod +x /tmp/install-salt-minion-rpm.sh && /tmp/install-salt-minion-rpm.sh ${APPHUB_URL} ${SALT_VERSION} ${MASTER_HOST} test-rocky01"

echo ""
echo "Verifying installation on test-rocky01..."
docker exec test-rocky01 salt-minion --version
docker exec test-rocky01 systemctl status salt-minion --no-pager || true

echo ""
echo "=========================================="
echo "Installation Tests Completed"
echo "=========================================="
echo ""
echo "Next steps:"
echo "1. On Salt Master, accept the minion keys:"
echo "   docker exec ai-infra-saltstack salt-key -L"
echo "   docker exec ai-infra-saltstack salt-key -a test-ssh01"
echo "   docker exec ai-infra-saltstack salt-key -a test-rocky01"
echo ""
echo "2. Test connectivity:"
echo "   docker exec ai-infra-saltstack salt 'test-*' test.ping"
echo ""
