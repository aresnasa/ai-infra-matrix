#!/bin/bash
# Quick test to verify SLURM spec file location and rpmbuild workflow

set -eux

SLURM_VERSION="25.05.4"
BUILD_DIR="/tmp/slurm-test-build"
mkdir -p "$BUILD_DIR"
cd "$BUILD_DIR"

echo "=== SLURM RPM Build Diagnostic Test ==="
echo ""

# Download SLURM tarball (if not already present)
if [ ! -f "slurm-${SLURM_VERSION}.tar.bz2" ]; then
    echo ">>> Downloading SLURM ${SLURM_VERSION}..."
    curl -sL "https://github.com/SchedMD/slurm/archive/refs/tags/slurm-${SLURM_VERSION}.tar.gz" \
        -o "slurm-${SLURM_VERSION}.tar.gz" || \
    curl -sL "https://www.schedmd.com/downloads/latest/slurm-${SLURM_VERSION}.tar.bz2" \
        -o "slurm-${SLURM_VERSION}.tar.bz2"
fi

tarball=$(ls slurm-*.tar.* | head -1)
echo "✓ Using tarball: $tarball"
echo ""

# Extract and inspect
echo ">>> Extracting tarball..."
tar -xf "$tarball" 2>&1 | head -20 || true
extracted_dir=$(tar -tf "$tarball" | head -1 | cut -d'/' -f1)
echo "✓ Extracted to: $extracted_dir"
echo ""

# Find spec files
echo ">>> Looking for .spec files:"
find "$extracted_dir" -name "*.spec" -type f || echo "No .spec files found"
echo ""

# Try standard location
echo ">>> Checking standard contribs/slurm.spec:"
if [ -f "$extracted_dir/contribs/slurm.spec" ]; then
    echo "✓ Found! First 30 lines:"
    head -30 "$extracted_dir/contribs/slurm.spec"
else
    echo "✗ Not found at contribs/slurm.spec"
    echo ">>> Contents of extracted_dir/contribs/:"
    ls -la "$extracted_dir/contribs/" 2>/dev/null | head -20 || echo "contribs/ not found"
fi
echo ""

echo "=== End of diagnostic test ==="
echo ""
echo "Next steps:"
echo "1. If spec file found: RPM build should work"
echo "2. If spec file not found: Update the build path in Dockerfile.tpl"
