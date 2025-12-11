#!/bin/bash
# Pre-build hook for SingleUser
# Generates Dockerfile based on network environment

COMPONENT="$1"
NETWORK_ENV="$2"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../.." && pwd)"
COMPONENT_DIR="$SCRIPT_DIR/src/$COMPONENT"
DOCKERFILE="$COMPONENT_DIR/Dockerfile"
DOCKERFILE_BACKUP="$COMPONENT_DIR/Dockerfile.backup"

echo "🔧 [SingleUser Hook] Preparing Dockerfile for $NETWORK_ENV environment..."

# Backup original Dockerfile if not exists
if [ ! -f "$DOCKERFILE_BACKUP" ] && [ -f "$DOCKERFILE" ]; then
    cp "$DOCKERFILE" "$DOCKERFILE_BACKUP"
fi

# Load config to get registry info
if [ -f "$SCRIPT_DIR/.env" ]; then
    source "$SCRIPT_DIR/.env"
fi

INTERNAL_REGISTRY="${INTERNAL_REGISTRY:-harbor.example.com}"
TARGET_TAG="${IMAGE_TAG:-v0.3.8}"

# Function to generate offline Dockerfile
generate_offline_dockerfile() {
    cat << EOF
# ai-infra single-user notebook image - Offline Version
# Uses pre-built image from internal registry
FROM ${INTERNAL_REGISTRY}/aihpc/ai-infra-singleuser:${TARGET_TAG}

ARG VERSION="${TARGET_TAG}"
ENV APP_VERSION=\${VERSION}

USER \${NB_UID}
ENV JUPYTER_ENABLE_LAB=yes
ENV JUPYTERLAB_SETTINGS_DIR=/home/jovyan/.jupyter/lab/user-settings

RUN echo "✓ Using internal pre-built image: ${INTERNAL_REGISTRY}/aihpc/ai-infra-singleuser:${TARGET_TAG}" && \
    python -c "import sys; print(f'✓ Python {sys.version}'); import jupyterhub, jupyterlab, ipykernel; print('✓ Core components ready')" && \
    jupyter --version

LABEL maintainer="AI Infrastructure Team" \
    org.opencontainers.image.title="ai-infra-singleuser-offline" \
    org.opencontainers.image.version="\${APP_VERSION}" \
    org.opencontainers.image.description="AI Infra Matrix - Singleuser Notebook (Offline Ready)" \
    org.opencontainers.image.source="${INTERNAL_REGISTRY}/aihpc/ai-infra-singleuser:${TARGET_TAG}"
EOF
}

# Function to generate offline build Dockerfile (fallback)
generate_offline_build_dockerfile() {
    cat << EOF
# ai-infra single-user notebook image pinned to JupyterHub 5.3.x
FROM jupyter/base-notebook:latest

ARG VERSION="dev"
ENV APP_VERSION=\${VERSION}

USER root

# Configure pip mirror for build time
ARG PIP_INDEX_URL="https://mirrors.aliyun.com/pypi/simple/"
ARG PIP_TRUSTED_HOST="mirrors.aliyun.com"

RUN pip config set global.index-url \${PIP_INDEX_URL} && \
    pip config set global.trusted-host \${PIP_TRUSTED_HOST}

RUN pip install --no-cache-dir \
    "jupyterhub==5.3.*" \
    ipykernel \
    jupyterlab \
    jupyterlab-execute-time \
    jupyterlab-code-formatter \
    jupyterlab-lsp \
    python-lsp-server[all]

# Optional tools
RUN pip install --no-cache-dir \
    numpy pandas matplotlib seaborn scikit-learn requests

ENV JUPYTER_ENABLE_LAB=yes
USER \${NB_UID}
ENV JUPYTERLAB_SETTINGS_DIR=/home/jovyan/.jupyter/lab/user-settings
RUN mkdir -p \${JUPYTERLAB_SETTINGS_DIR}/jupyterlab-execute-time
RUN echo '{"enabled": true}' > \${JUPYTERLAB_SETTINGS_DIR}/jupyterlab-execute-time/plugin.jupyterlab-settings

RUN python -m ipykernel install --user --name python3 --display-name "Python 3 (ipykernel)"

LABEL maintainer="AI Infrastructure Team" \
    org.opencontainers.image.title="ai-infra-singleuser" \
    org.opencontainers.image.version="\${APP_VERSION}" \
    org.opencontainers.image.description="AI Infra Matrix - Singleuser Notebook (Offline Build Mode)"
EOF
}

# Function to generate online Dockerfile
generate_online_dockerfile() {
    cat << EOF
# ai-infra single-user notebook image pinned to JupyterHub 5.3.x
FROM jupyter/base-notebook:latest

ARG VERSION="dev"
ENV APP_VERSION=\${VERSION}

USER root

RUN pip config set global.index-url https://mirrors.aliyun.com/pypi/simple/ && \
    pip config set global.trusted-host mirrors.aliyun.com && \
    pip install --no-cache-dir \
    "jupyterhub==5.3.*" \
    ipykernel \
    jupyterlab \
    jupyterlab-execute-time \
    jupyterlab-code-formatter \
    jupyterlab-lsp \
    python-lsp-server[all]

ENV JUPYTER_ENABLE_LAB=yes
ENV JUPYTERLAB_SETTINGS_DIR=/home/jovyan/.jupyter/lab/user-settings
USER \${NB_UID}
RUN mkdir -p \${JUPYTERLAB_SETTINGS_DIR}/jupyterlab-execute-time && \
    echo '{"enabled": true}' > \${JUPYTERLAB_SETTINGS_DIR}/jupyterlab-execute-time/plugin.jupyterlab-settings || true

RUN python -m ipykernel install --user --name python3 --display-name "Python 3 (ipykernel)" || true

LABEL maintainer="AI Infrastructure Team" \
    org.opencontainers.image.title="ai-infra-singleuser" \
    org.opencontainers.image.version="\${APP_VERSION}" \
    org.opencontainers.image.description="AI Infra Matrix - Singleuser Notebook"
EOF
}

# Logic to choose Dockerfile
if [ "$NETWORK_ENV" == "intranet" ]; then
    # Check if internal image exists
    HARBOR_IMAGE="${INTERNAL_REGISTRY}/aihpc/ai-infra-singleuser:${TARGET_TAG}"
    if docker manifest inspect "$HARBOR_IMAGE" &>/dev/null; then
        echo "✓ Internal image available: $HARBOR_IMAGE"
        generate_offline_dockerfile > "$DOCKERFILE"
    else
        echo "⚠ Internal image not found, falling back to offline build mode..."
        generate_offline_build_dockerfile > "$DOCKERFILE"
    fi
else
    generate_online_dockerfile > "$DOCKERFILE"
fi

echo "✅ [SingleUser Hook] Dockerfile generated."
