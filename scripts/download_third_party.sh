#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
THIRD_PARTY_DIR="$PROJECT_ROOT/third_party"
DOCKERFILE="$PROJECT_ROOT/src/apphub/Dockerfile"

mkdir -p "$THIRD_PARTY_DIR"

# Helper to extract ARG value
get_arg() {
    local name=$1
    grep "ARG $name=" "$DOCKERFILE" | head -1 | cut -d'=' -f2 | tr -d '"' | tr -d ' '
}

# Helper to extract variable from RUN command
get_run_var() {
    local name=$1
    # Use sed to extract value between quotes or after =, handling quotes, spaces, semicolons, and backslashes
    grep "$name=" "$DOCKERFILE" | head -1 | sed -E "s/.*$name=\"?([^ \";\\\\]+)\"?.*/\1/"
}

SALTSTACK_VERSION=$(get_arg SALTSTACK_VERSION)
CATEGRAF_VERSION=$(get_arg CATEGRAF_VERSION)
SINGULARITY_VERSION=$(get_arg SINGULARITY_VERSION)
MUNGE_VERSION=$(get_run_var MUNGE_VERSION)

if [ -z "$MUNGE_VERSION" ]; then
    MUNGE_VERSION="0.5.16"
fi

echo "Versions detected:"
echo "SaltStack: $SALTSTACK_VERSION"
echo "Categraf: $CATEGRAF_VERSION"
echo "Singularity: $SINGULARITY_VERSION"
echo "Munge: $MUNGE_VERSION"

# 1. Categraf (Tarball)
echo "----------------------------------------------------------------"
echo "Processing Categraf..."
CATEGRAF_DIR="$THIRD_PARTY_DIR/categraf"
mkdir -p "$CATEGRAF_DIR"

# Ensure version starts with v
if [[ ! "$CATEGRAF_VERSION" =~ ^v ]]; then
    CATEGRAF_VERSION="v${CATEGRAF_VERSION}"
fi

for ARCH in amd64 arm64; do
    TAR_FILE="categraf-${CATEGRAF_VERSION}-linux-${ARCH}.tar.gz"
    if [ ! -f "$CATEGRAF_DIR/$TAR_FILE" ]; then
        echo "Downloading Categraf ($ARCH)..."
        wget -nv "https://github.com/flashcatcloud/categraf/releases/download/${CATEGRAF_VERSION}/$TAR_FILE" -O "$CATEGRAF_DIR/$TAR_FILE" || echo "Failed to download $TAR_FILE"
    else
        echo "Categraf ($ARCH) already downloaded."
    fi
done

# 2. Munge (Tarball)
echo "----------------------------------------------------------------"
echo "Processing Munge..."
MUNGE_DIR="$THIRD_PARTY_DIR/munge"
mkdir -p "$MUNGE_DIR"
MUNGE_FILE="munge-${MUNGE_VERSION}.tar.xz"
if [ ! -f "$MUNGE_DIR/$MUNGE_FILE" ]; then
    echo "Downloading Munge..."
    wget -nv "https://github.com/dun/munge/releases/download/munge-${MUNGE_VERSION}/$MUNGE_FILE" -O "$MUNGE_DIR/$MUNGE_FILE"
else
    echo "Munge already downloaded."
fi

# 3. Singularity (DEB)
echo "----------------------------------------------------------------"
echo "Processing Singularity..."
SINGULARITY_DIR="$THIRD_PARTY_DIR/singularity"
mkdir -p "$SINGULARITY_DIR"
SINGULARITY_VER_NUM="${SINGULARITY_VERSION#v}"
for ARCH in amd64 arm64; do
    DEB_FILE="singularity-ce_${SINGULARITY_VER_NUM}-1~ubuntu22.04_${ARCH}.deb"
    if [ ! -f "$SINGULARITY_DIR/$DEB_FILE" ]; then
        echo "Downloading Singularity ($ARCH)..."
        wget -nv "https://github.com/sylabs/singularity/releases/download/${SINGULARITY_VERSION}/$DEB_FILE" -O "$SINGULARITY_DIR/$DEB_FILE" || echo "Failed to download $DEB_FILE"
    else
        echo "Singularity ($ARCH) already downloaded."
    fi
done

# 4. SaltStack (DEB & RPM)
echo "----------------------------------------------------------------"
echo "Processing SaltStack..."
SALT_DIR="$THIRD_PARTY_DIR/saltstack"
mkdir -p "$SALT_DIR"

SALT_VER_NUM="${SALTSTACK_VERSION#v}"
RELEASE_TAG="${SALTSTACK_VERSION}"
if [[ ! "$RELEASE_TAG" =~ ^v ]]; then
    RELEASE_TAG="v${RELEASE_TAG}"
fi
BASE_URL="https://github.com/saltstack/salt/releases/download/${RELEASE_TAG}"

# DEB
for ARCH in amd64 arm64; do
    for pkg in salt-common salt-master salt-minion salt-api salt-ssh salt-syndic salt-cloud; do
        PKG_FILE="${pkg}_${SALT_VER_NUM}_${ARCH}.deb"
        if [ ! -f "$SALT_DIR/$PKG_FILE" ]; then
            echo "Downloading $PKG_FILE..."
            wget -nv "${BASE_URL}/${PKG_FILE}" -O "$SALT_DIR/$PKG_FILE" || echo "Failed to download $PKG_FILE"
        else
            echo "$PKG_FILE already exists."
        fi
    done
done

# RPM
for ARCH in x86_64 aarch64; do
    for pkg in salt salt-master salt-minion salt-api salt-ssh salt-syndic salt-cloud; do
        PKG_FILE="${pkg}-${SALT_VER_NUM}-0.${ARCH}.rpm"
        if [ ! -f "$SALT_DIR/$PKG_FILE" ]; then
            echo "Downloading $PKG_FILE..."
            wget -nv "${BASE_URL}/${PKG_FILE}" -O "$SALT_DIR/$PKG_FILE" || echo "Failed to download $PKG_FILE"
        else
            echo "$PKG_FILE already exists."
        fi
    done
done

echo "----------------------------------------------------------------"
echo "All third-party dependencies processed."
