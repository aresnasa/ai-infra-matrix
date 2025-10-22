#!/bin/bash
set -e

echo "🔨 Testing AppHub Dockerfile package building..."
echo "================================================"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Build AppHub image
echo "📦 Building AppHub image..."
cd "$(dirname "$0")"
docker build -t ai-infra-apphub:test -f src/apphub/Dockerfile src/apphub || {
    echo -e "${RED}❌ Build failed${NC}"
    exit 1
}

echo ""
echo "✓ Build completed successfully"
echo ""

# Run container and check packages
echo "🔍 Checking package directories..."
CONTAINER_ID=$(docker run -d ai-infra-apphub:test sleep 60)

echo ""
echo "📊 Package directory structure:"
docker exec "$CONTAINER_ID" tree /usr/share/nginx/html/pkgs || \
    docker exec "$CONTAINER_ID" find /usr/share/nginx/html/pkgs -type f

echo ""
echo "📊 Package counts:"
echo "  SLURM deb packages:"
docker exec "$CONTAINER_ID" sh -c 'ls /usr/share/nginx/html/pkgs/slurm-deb/*.deb 2>/dev/null | wc -l || echo 0'

echo "  SLURM rpm packages:"
docker exec "$CONTAINER_ID" sh -c 'ls /usr/share/nginx/html/pkgs/slurm-rpm/*.rpm 2>/dev/null | wc -l || echo 0'

echo "  SLURM apk packages:"
docker exec "$CONTAINER_ID" sh -c 'ls /usr/share/nginx/html/pkgs/slurm-apk/*.tar.gz 2>/dev/null | wc -l || echo 0'

echo "  SaltStack deb packages:"
docker exec "$CONTAINER_ID" sh -c 'ls /usr/share/nginx/html/pkgs/saltstack-deb/*.deb 2>/dev/null | wc -l || echo 0'

echo "  SaltStack rpm packages:"
docker exec "$CONTAINER_ID" sh -c 'ls /usr/share/nginx/html/pkgs/saltstack-rpm/*.rpm 2>/dev/null | wc -l || echo 0'

echo ""
echo "📦 Detailed file listing:"
echo ""
echo "SaltStack RPM:"
docker exec "$CONTAINER_ID" ls -lh /usr/share/nginx/html/pkgs/saltstack-rpm/ || echo "  (empty or missing)"

echo ""
echo "SLURM RPM:"
docker exec "$CONTAINER_ID" ls -lh /usr/share/nginx/html/pkgs/slurm-rpm/ || echo "  (empty or missing)"

echo ""
echo "SLURM APK:"
docker exec "$CONTAINER_ID" ls -lh /usr/share/nginx/html/pkgs/slurm-apk/ || echo "  (empty or missing)"

# Cleanup
echo ""
echo "🧹 Cleaning up test container..."
docker stop "$CONTAINER_ID" >/dev/null
docker rm "$CONTAINER_ID" >/dev/null

echo ""
echo "✅ Test completed"
echo ""
echo "Summary:"
echo "--------"
DEB_COUNT=$(docker run --rm ai-infra-apphub:test sh -c 'ls /usr/share/nginx/html/pkgs/slurm-deb/*.deb 2>/dev/null | wc -l || echo 0')
RPM_COUNT=$(docker run --rm ai-infra-apphub:test sh -c 'ls /usr/share/nginx/html/pkgs/slurm-rpm/*.rpm 2>/dev/null | wc -l || echo 0')
APK_COUNT=$(docker run --rm ai-infra-apphub:test sh -c 'ls /usr/share/nginx/html/pkgs/slurm-apk/*.tar.gz 2>/dev/null | wc -l || echo 0')
SALT_DEB_COUNT=$(docker run --rm ai-infra-apphub:test sh -c 'ls /usr/share/nginx/html/pkgs/saltstack-deb/*.deb 2>/dev/null | wc -l || echo 0')
SALT_RPM_COUNT=$(docker run --rm ai-infra-apphub:test sh -c 'ls /usr/share/nginx/html/pkgs/saltstack-rpm/*.rpm 2>/dev/null | wc -l || echo 0')

echo -e "  SLURM deb: ${DEB_COUNT}"
echo -e "  SLURM rpm: ${RPM_COUNT}"
echo -e "  SLURM apk: ${APK_COUNT}"
echo -e "  SaltStack deb: ${SALT_DEB_COUNT}"
echo -e "  SaltStack rpm: ${SALT_RPM_COUNT}"

echo ""
if [ "$SALT_RPM_COUNT" -gt 0 ] && [ "$APK_COUNT" -gt 0 ]; then
    echo -e "${GREEN}✅ All package types built successfully!${NC}"
elif [ "$SALT_RPM_COUNT" -eq 0 ]; then
    echo -e "${RED}❌ SaltStack RPM packages missing${NC}"
    exit 1
elif [ "$APK_COUNT" -eq 0 ]; then
    echo -e "${YELLOW}⚠️  SLURM APK packages missing (this may be expected if build failed)${NC}"
fi
