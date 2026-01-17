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
    # Try multiple download methods
    curl -sL "https://www.schedmd.com/downloads/latest/slurm-${SLURM_VERSION}.tar.bz2" \
        -o "slurm-${SLURM_VERSION}.tar.bz2" 2>/dev/null || \
    curl -sL "https://github.com/SchedMD/slurm/archive/refs/tags/slurm-${SLURM_VERSION}.tar.gz" \
        -o "slurm-${SLURM_VERSION}.tar.gz" 2>/dev/null || \
    echo "⚠️  Could not download SLURM tarball"
fi

tarball=$(ls slurm-*.tar.* 2>/dev/null | head -1)
if [ -z "$tarball" ]; then
    echo "❌ ERROR: No SLURM tarball found!"
    echo "Please download from: https://www.schedmd.com/downloads/latest/"
    exit 1
fi
echo "✓ Using tarball: $tarball"
echo ""

# Extract and inspect
echo ">>> Extracting tarball..."
if [[ "$tarball" == *.tar.gz ]]; then
    tar -xzf "$tarball" 2>&1 | head -5 || true
elif [[ "$tarball" == *.tar.bz2 ]]; then
    tar -xjf "$tarball" 2>&1 | head -5 || true
else
    tar -xf "$tarball" 2>&1 | head -5 || true
fi

extracted_dir=$(tar -tf "$tarball" | head -1 | cut -d'/' -f1)
echo "✓ Extracted to: $extracted_dir"
echo ""

# Find spec files
echo ">>> Looking for .spec files:"
spec_count=$(find "$extracted_dir" -name "*.spec" -type f 2>/dev/null | wc -l)
echo "Found $spec_count .spec file(s):"
find "$extracted_dir" -name "*.spec" -type f 2>/dev/null || echo "No .spec files found"
echo ""

# Try standard location
echo ">>> Checking standard locations:"
for location in "contribs/slurm.spec" "docs/slurm.spec" "slurm.spec"; do
    if [ -f "$extracted_dir/$location" ]; then
        echo "✓ Found at: $location"
        echo "  Contents (first 30 lines):"
        head -30 "$extracted_dir/$location" | sed 's/^/    /'
        break
    else
        echo "✗ Not found at: $location"
    fi
done
echo ""

echo "=== End of diagnostic test ==="
echo ""
echo "Next steps:"
echo "1. If spec file found: RPM build should work"
echo "2. If not found in standard location: Check actual location above"
echo "3. Update Dockerfile.tpl spec_file path if needed"
