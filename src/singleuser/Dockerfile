# ai-infra single-user notebook image pinned to JupyterHub 5.3.x
# Base on jupyter/docker-stacks base-notebook for a full Lab experience
# Includes VS Code Server support for VS Code client connections
FROM jupyter/base-notebook:latest

# Version metadata
ARG VERSION="dev"
ARG PYPI_INDEX_URL=https://mirrors.aliyun.com/pypi/simple/
ARG CODE_SERVER_VERSION=v4.107.0
ENV APP_VERSION=${VERSION}
ENV CODE_SERVER_VERSION=${CODE_SERVER_VERSION}

USER root

# ========================================
# 配置阿里云 APT 镜像源（加速构建）
# ========================================
ARG APT_MIRROR=mirrors.aliyun.com
RUN set -eux; \
    # 获取 Ubuntu 版本代号
    . /etc/os-release && CODENAME=${VERSION_CODENAME:-jammy}; \
    # 检测架构: x86_64 使用 ubuntu, arm64/aarch64 使用 ubuntu-ports
    ARCH=$(dpkg --print-architecture); \
    if [ "${ARCH}" = "amd64" ]; then \
        UBUNTU_PATH="ubuntu"; \
    else \
        UBUNTU_PATH="ubuntu-ports"; \
    fi; \
    # 配置阿里云镜像源
    echo "deb http://${APT_MIRROR}/${UBUNTU_PATH}/ ${CODENAME} main restricted universe multiverse" > /etc/apt/sources.list; \
    echo "deb http://${APT_MIRROR}/${UBUNTU_PATH}/ ${CODENAME}-updates main restricted universe multiverse" >> /etc/apt/sources.list; \
    echo "deb http://${APT_MIRROR}/${UBUNTU_PATH}/ ${CODENAME}-security main restricted universe multiverse" >> /etc/apt/sources.list; \
    echo "✓ APT mirror configured: ${APT_MIRROR}/${UBUNTU_PATH}"

# ========================================
# 系统依赖安装
# ========================================
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    wget \
    git \
    openssh-client \
    procps \
    htop \
    vim \
    nano \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# ========================================
# 安装 code-server (VS Code Server)
# ========================================
# 复制预下载的 code-server deb 包（支持 amd64 和 arm64）
COPY third_party/code_server/*.deb /tmp/code-server/

# 根据架构安装对应的 deb 包
RUN set -eux; \
    ARCH=$(dpkg --print-architecture); \
    DEB_FILE=$(ls /tmp/code-server/code-server_*_${ARCH}.deb 2>/dev/null | head -1); \
    if [ -z "$DEB_FILE" ]; then \
        echo "✗ No code-server deb found for ${ARCH}"; \
        ls -la /tmp/code-server/; \
        exit 1; \
    fi; \
    echo "→ Installing code-server from ${DEB_FILE}..."; \
    dpkg -i "${DEB_FILE}" || apt-get install -f -y; \
    rm -rf /tmp/code-server; \
    code-server --version; \
    echo "✓ code-server installed"

# 创建 code-server 配置目录
RUN mkdir -p /home/jovyan/.config/code-server \
    && mkdir -p /home/jovyan/.local/share/code-server/extensions

# 配置 code-server
COPY --chown=${NB_UID}:${NB_GID} src/singleuser/config/code-server-config.yaml /home/jovyan/.config/code-server/config.yaml

# ========================================
# 构建阶段：预安装所有必要的Python包
# ========================================
# 配置pip镜像源（构建时使用，运行时不依赖网络）
RUN pip config set global.index-url ${PYPI_INDEX_URL:-https://mirrors.aliyun.com/pypi/simple/} && \
    pip config set global.trusted-host mirrors.aliyun.com

# 安装所有必要的Python包（构建阶段完成，运行时无需网络）
RUN pip install --no-cache-dir \
    "jupyterhub==5.3.*" \
    ipykernel \
    jupyterlab \
    jupyterlab-execute-time \
    jupyterlab-code-formatter \
    jupyterlab-lsp \
    python-lsp-server[all] \
    jupyter-server-proxy \
    jupyter-vscode-proxy

# 安装额外的开发工具（可选，在构建时决定是否包含）
RUN pip install --no-cache-dir \
    numpy \
    pandas \
    matplotlib \
    seaborn \
    scikit-learn \
    requests

# ========================================
# 配置阶段：设置Jupyter环境
# ========================================
# 预启用JupyterLab
ENV JUPYTER_ENABLE_LAB=yes
# code-server 端口配置
ENV CODE_SERVER_PORT=8443
ENV CODE_SERVER_BIND_ADDR=0.0.0.0:8443

# 修复 /home/jovyan 目录权限（确保用户有写入权限）
RUN chown -R ${NB_UID}:${NB_GID} /home/jovyan

# 切换到普通用户进行配置
USER ${NB_UID}

# 预配置JupyterLab设置目录
ENV JUPYTERLAB_SETTINGS_DIR=/home/jovyan/.jupyter/lab/user-settings
RUN mkdir -p ${JUPYTERLAB_SETTINGS_DIR}/jupyterlab-execute-time

# 预启用执行时间显示扩展
RUN echo '{"enabled": true}' > ${JUPYTERLAB_SETTINGS_DIR}/jupyterlab-execute-time/plugin.jupyterlab-settings

# 预安装和配置Python内核（确保在构建时完成）
RUN python -m ipykernel install --user --name python3 --display-name "Python 3 (ipykernel)"

# 预构建JupyterLab扩展（避免运行时构建，使用更宽松的设置）
RUN jupyter lab build --dev-build=False --minimize=False || \
    jupyter lab build --dev-build=False --minimize=False --debug || \
    echo "Warning: JupyterLab build failed, will build at runtime"

# 预安装常用的 VS Code 扩展
USER root
RUN code-server --install-extension ms-python.python || true \
    && code-server --install-extension ms-toolsai.jupyter || true \
    && echo "✓ VS Code extensions installed"

# 复制 Jupyter Server 配置（启用 VS Code 代理）
COPY --chown=${NB_UID}:${NB_GID} src/singleuser/config/jupyter_server_config.py /home/jovyan/.jupyter/jupyter_server_config.py
COPY --chown=${NB_UID}:${NB_GID} src/singleuser/config/vscode_proxy.py /home/jovyan/.jupyter/vscode_proxy.py
COPY --chown=${NB_UID}:${NB_GID} src/singleuser/config/icons /home/jovyan/.jupyter/icons

USER ${NB_UID}

# ========================================
# 验证阶段：确保所有组件正常工作
# ========================================
# 验证关键组件是否正确安装
RUN python -c "import jupyterhub, jupyterlab, ipykernel; print('✓ 核心组件验证成功')" && \
    python -c "import jupyter_server_proxy; print('✓ jupyter-server-proxy 已安装')" && \
    jupyter --version && \
    jupyter lab --version && \
    code-server --version

# 复制启动脚本
USER root
COPY --chown=${NB_UID}:${NB_GID} src/singleuser/scripts/start-with-vscode.sh /usr/local/bin/start-with-vscode.sh
RUN chmod +x /usr/local/bin/start-with-vscode.sh

USER ${NB_UID}

LABEL maintainer="AI Infrastructure Team" \
    org.opencontainers.image.title="ai-infra-singleuser" \
    org.opencontainers.image.version="${APP_VERSION}" \
    org.opencontainers.image.description="AI Infra Matrix - Singleuser Notebook with VS Code Server Support"

