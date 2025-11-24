#!/bin/bash
# Post-build hook for AppHub
# Extracts built packages from the image to the local cache

IMAGE_NAME="$1"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
CACHE_DIR="$SCRIPT_DIR/.build-cache/apphub-packages"

echo "📦 [AppHub Hook] Extracting packages from $IMAGE_NAME to $CACHE_DIR..."

mkdir -p "$CACHE_DIR"

# Create a temporary container
CONTAINER_ID=$(docker create "$IMAGE_NAME")

if [ -z "$CONTAINER_ID" ]; then
    echo "❌ Failed to create temporary container"
    exit 1
fi

# Helper function to extract
extract_dir() {
    local src="$1"
    local dest="$2"
    local name="$3"
    
    if docker cp "$CONTAINER_ID:$src" "$dest" 2>/dev/null; then
        local count=$(find "$dest" -type f | wc -l)
        echo "  ✓ Extracted $name: $count files"
    else
        echo "  ⚠️  Could not extract $name (path may not exist in image)"
    fi
}

extract_dir "/usr/share/nginx/html/pkgs/slurm-deb" "$CACHE_DIR/" "SLURM deb"
extract_dir "/usr/share/nginx/html/pkgs/slurm-rpm" "$CACHE_DIR/" "SLURM rpm"
extract_dir "/usr/share/nginx/html/pkgs/saltstack-deb" "$CACHE_DIR/" "SaltStack deb"
extract_dir "/usr/share/nginx/html/pkgs/saltstack-rpm" "$CACHE_DIR/" "SaltStack rpm"
extract_dir "/usr/share/nginx/html/pkgs/slurm-binaries" "$CACHE_DIR/" "SLURM binaries"
extract_dir "/usr/share/nginx/html/pkgs/categraf" "$CACHE_DIR/" "Categraf"

# Cleanup
docker rm "$CONTAINER_ID" >/dev/null

echo "✅ [AppHub Hook] Extraction complete"
