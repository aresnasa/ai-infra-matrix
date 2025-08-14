# ai-infra single-user notebook image pinned to JupyterHub 5.3.x
# Base on jupyter/docker-stacks base-notebook for a full Lab experience
FROM jupyter/base-notebook:latest

# Version metadata
ARG VERSION="dev"
ENV APP_VERSION=${VERSION}

USER root

# Align jupyterhub-singleuser with Hub 5.3.x to avoid auth/redirect quirks
RUN pip install --no-cache-dir \
	"jupyterhub==5.3.*" \
	ipykernel \
	jupyterlab \
	jupyterlab-execute-time \
	jupyterlab-code-formatter \
	jupyterlab-lsp \
	python-lsp-server[all]

# Optional: pre-enable Lab (the base image already does, keep explicit)
ENV JUPYTER_ENABLE_LAB=yes

# Pre-enable execute time display
ENV JUPYTERLAB_SETTINGS_DIR=/home/jovyan/.jupyter/lab/user-settings
USER ${NB_UID}
RUN mkdir -p ${JUPYTERLAB_SETTINGS_DIR}/jupyterlab-execute-time && \
	echo '{"enabled": true}' > ${JUPYTERLAB_SETTINGS_DIR}/jupyterlab-execute-time/plugin.jupyterlab-settings || true

# Ensure ipykernel is available
RUN python -m ipykernel install --user --name python3 --display-name "Python 3 (ipykernel)" || true

LABEL maintainer="AI Infrastructure Team" \
	org.opencontainers.image.title="ai-infra-singleuser" \
	org.opencontainers.image.version="${APP_VERSION}" \
	org.opencontainers.image.description="AI Infra Matrix - Singleuser Notebook"
