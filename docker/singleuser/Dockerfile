# ai-infra single-user notebook image pinned to JupyterHub 5.3.x
# Base on jupyter/docker-stacks base-notebook for a full Lab experience
FROM jupyter/base-notebook:latest

# Version metadata
ARG VERSION="dev"
ENV APP_VERSION=${VERSION}

USER root

# Align jupyterhub-singleuser with Hub 5.3.x to avoid auth/redirect quirks
RUN pip install --no-cache-dir "jupyterhub==5.3.*"

# Optional: pre-enable Lab (the base image already does, keep explicit)
ENV JUPYTER_ENABLE_LAB=yes

# Back to default user
USER ${NB_UID}

LABEL maintainer="AI Infrastructure Team" \
	org.opencontainers.image.title="ai-infra-singleuser" \
	org.opencontainers.image.version="${APP_VERSION}" \
	org.opencontainers.image.description="AI Infra Matrix - Singleuser Notebook"
