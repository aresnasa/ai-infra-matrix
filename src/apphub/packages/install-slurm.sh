#!/bin/sh
# SLURM Client Binary Installation Script
# Downloads and installs SLURM binaries from AppHub based on architecture

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Get AppHub URL from environment or use default
APPHUB_URL="${APPHUB_URL:-http://apphub}"

# Detect architecture
ARCH=$(uname -m)
print_info "Detected architecture: ${ARCH}"

# Map to directory name
case "$ARCH" in
    x86_64|amd64)
        ARCH_DIR="x86_64"
        ;;
    aarch64|arm64)
        ARCH_DIR="aarch64"
        ;;
    *)
        print_error "Unsupported architecture: ${ARCH}"
        exit 1
        ;;
esac

print_info "Using architecture directory: ${ARCH_DIR}"

# Create temporary directory
TEMP_DIR="/tmp/slurm-install-$$"
mkdir -p "$TEMP_DIR"
cd "$TEMP_DIR"

print_info "Downloading SLURM binaries from AppHub..."

# Download binaries
BIN_URL="${APPHUB_URL}/pkgs/slurm-binaries/${ARCH_DIR}/bin"
LIB_URL="${APPHUB_URL}/pkgs/slurm-binaries/${ARCH_DIR}/lib"

# Create installation directories
mkdir -p /usr/local/slurm/bin
mkdir -p /usr/local/slurm/lib
mkdir -p /etc/slurm

# Download all binaries
for cmd in sinfo squeue scontrol scancel sbatch srun salloc sacct sacctmgr; do
    print_info "Downloading ${cmd}..."
    if wget --timeout=30 --tries=3 -q -O "${cmd}" "${BIN_URL}/${cmd}" 2>/dev/null; then
        chmod +x "${cmd}"
        mv "${cmd}" /usr/local/slurm/bin/
        print_success "Installed: ${cmd}"
    else
        print_error "Failed to download: ${cmd}"
    fi
done

# Download libraries
print_info "Downloading SLURM libraries..."
# Get list of library files
LIB_LIST=$(wget -q -O - "${APPHUB_URL}/pkgs/slurm-binaries/${ARCH_DIR}/lib/" 2>/dev/null | grep -oP 'href="\K[^"]+\.so[^"]*' || true)

if [ -n "$LIB_LIST" ]; then
    for lib in $LIB_LIST; do
        if wget --timeout=30 --tries=3 -q -O "/usr/local/slurm/lib/${lib}" "${LIB_URL}/${lib}" 2>/dev/null; then
            print_success "Downloaded library: ${lib}"
        fi
    done
fi

# Download version file
wget -q -O /usr/local/slurm/VERSION "${APPHUB_URL}/pkgs/slurm-binaries/${ARCH_DIR}/VERSION" 2>/dev/null || echo "unknown" > /usr/local/slurm/VERSION

# Create symlinks to /usr/local/bin
print_info "Creating symlinks..."
for cmd in /usr/local/slurm/bin/*; do
    if [ -f "$cmd" ]; then
        ln -sf "$cmd" /usr/local/bin/$(basename "$cmd")
    fi
done

# Configure library path
echo "/usr/local/slurm/lib" > /etc/ld-musl-$(uname -m).path 2>/dev/null || true

# Configure environment
cat >> /etc/profile << 'EOF'

# SLURM Client Environment
export SLURM_HOME=/usr/local/slurm
export PATH=$SLURM_HOME/bin:$PATH
export LD_LIBRARY_PATH=$SLURM_HOME/lib:$LD_LIBRARY_PATH
EOF

# Cleanup
cd /
rm -rf "$TEMP_DIR"

# Verify installation
print_info "Verifying SLURM installation..."
if command -v sinfo >/dev/null 2>&1; then
    VERSION=$(cat /usr/local/slurm/VERSION 2>/dev/null || echo "unknown")
    print_success "SLURM client installed successfully"
    print_success "Version: ${VERSION}"
    print_info "Available commands:"
    for cmd in sinfo squeue scontrol scancel sbatch srun salloc sacct sacctmgr; do
        if command -v $cmd >/dev/null 2>&1; then
            echo "  - $cmd"
        fi
    done
else
    print_error "SLURM installation verification failed"
    exit 1
fi

print_success "SLURM client installation completed"
