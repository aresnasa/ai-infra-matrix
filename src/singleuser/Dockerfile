# ai-infra single-user notebook image pinned to JupyterHub 5.3.x
FROM jupyter/base-notebook:latest

ARG VERSION="dev"
ENV APP_VERSION=${VERSION}

USER root

RUN pip config set global.index-url https://mirrors.aliyun.com/pypi/simple/ &&     pip config set global.trusted-host mirrors.aliyun.com &&     pip install --no-cache-dir     "jupyterhub==5.3.*"     ipykernel     jupyterlab     jupyterlab-execute-time     jupyterlab-code-formatter     jupyterlab-lsp     python-lsp-server[all]

ENV JUPYTER_ENABLE_LAB=yes
ENV JUPYTERLAB_SETTINGS_DIR=/home/jovyan/.jupyter/lab/user-settings
USER ${NB_UID}
RUN mkdir -p ${JUPYTERLAB_SETTINGS_DIR}/jupyterlab-execute-time &&     echo '{"enabled": true}' > ${JUPYTERLAB_SETTINGS_DIR}/jupyterlab-execute-time/plugin.jupyterlab-settings || true

RUN python -m ipykernel install --user --name python3 --display-name "Python 3 (ipykernel)" || true

LABEL maintainer="AI Infrastructure Team"     org.opencontainers.image.title="ai-infra-singleuser"     org.opencontainers.image.version="${APP_VERSION}"     org.opencontainers.image.description="AI Infra Matrix - Singleuser Notebook"
