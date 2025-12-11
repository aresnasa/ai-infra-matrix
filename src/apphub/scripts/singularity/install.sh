#!/bin/bash
# Singularity Installation Script
# Downloads and installs Singularity from AppHub
# Supports: Ubuntu/Debian (.deb) and RHEL/Rocky/CentOS (.rpm)
# Architectures: amd64/x86_64, arm64/aarch64

set -e

# Configuration
APPHUB_URL="${APPHUB_URL:-http://apphub:8081}"
# Remove 'v' prefix if present for version number usage
VERSION_TAG="${SINGULARITY_VERSION:-v4.3.4}"
VERSION_NUM="${VERSION_TAG#v}"

echo "=========================================="
echo "Installing Singularity ${VERSION_TAG}"
echo "=========================================="
echo "AppHub URL: ${APPHUB_URL}"

# Detect Architecture
ARCH=$(uname -m)
case ${ARCH} in
    x86_64)
        DEB_ARCH="amd64"
        RPM_ARCH="x86_64"
        ;;
    aarch64|arm64)
        DEB_ARCH="arm64"
        RPM_ARCH="aarch64"
        ;;
    *)
        echo "ERROR: Unsupported architecture: ${ARCH}"
        exit 1
        ;;
esac
echo "Detected Architecture: ${ARCH}"

# Detect OS and Install
if [ -f /etc/debian_version ]; then
    # Debian/Ubuntu
    echo "Detected OS: Debian/Ubuntu family"
    
    # We use the Ubuntu 22.04 build as the generic .deb package
    PACKAGE_NAME="singularity-ce_${VERSION_NUM}-1~ubuntu22.04_${DEB_ARCH}.deb"
    PACKAGE_URL="${APPHUB_URL}/pkgs/singularity/${PACKAGE_NAME}"
    
    echo "Downloading ${PACKAGE_NAME}..."
    if ! curl -fsSL "${PACKAGE_URL}" -o "/tmp/${PACKAGE_NAME}"; then
        echo "ERROR: Failed to download ${PACKAGE_URL}"
        exit 1
    fi
    
    echo "Installing..."
    # Install dependencies if needed (using apt-get install -f)
    dpkg -i "/tmp/${PACKAGE_NAME}" || apt-get install -f -y
    rm -f "/tmp/${PACKAGE_NAME}"

elif [ -f /etc/redhat-release ]; then
    # RHEL/Rocky/CentOS
    echo "Detected OS: RHEL/Rocky/CentOS family"
    
    # We use the EL9 build as the generic .rpm package
    PACKAGE_NAME="singularity-ce-${VERSION_NUM}-1.el9.${RPM_ARCH}.rpm"
    PACKAGE_URL="${APPHUB_URL}/pkgs/singularity/${PACKAGE_NAME}"
    
    echo "Downloading ${PACKAGE_NAME}..."
    if ! curl -fsSL "${PACKAGE_URL}" -o "/tmp/${PACKAGE_NAME}"; then
        echo "ERROR: Failed to download ${PACKAGE_URL}"
        exit 1
    fi
    
    echo "Installing..."
    # Use yum/dnf to handle dependencies
    if command -v dnf >/dev/null; then
        dnf install -y "/tmp/${PACKAGE_NAME}"
    else
        yum install -y "/tmp/${PACKAGE_NAME}"
    fi
    rm -f "/tmp/${PACKAGE_NAME}"

else
    echo "ERROR: Unsupported Operating System"
    echo "This script supports Debian/Ubuntu and RHEL/Rocky/CentOS based systems."
    exit 1
fi

# Verify Installation
if command -v singularity &> /dev/null; then
    INSTALLED_VERSION=$(singularity --version)
    echo ""
    echo "=========================================="
    echo "✅ Singularity Installed Successfully"
    echo "=========================================="
    echo "Version: ${INSTALLED_VERSION}"
    echo "Path: $(which singularity)"
    echo ""
else
    echo "ERROR: Installation failed, 'singularity' command not found."
    exit 1
fi
