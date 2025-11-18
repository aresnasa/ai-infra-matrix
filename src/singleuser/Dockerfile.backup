# ai-infra single-user notebook image pinned to JupyterHub 5.3.x
# Base on jupyter/docker-stacks base-notebook for a full Lab experience
FROM jupyter/base-notebook:latest

# Version metadata
ARG VERSION="dev"
ENV APP_VERSION=${VERSION}

USER root

# ========================================
# 构建阶段：预安装所有必要的Python包
# ========================================
# 配置pip镜像源（构建时使用，运行时不依赖网络）
RUN pip config set global.index-url https://mirrors.aliyun.com/pypi/simple/ && \
    pip config set global.trusted-host mirrors.aliyun.com

# 安装所有必要的Python包（构建阶段完成，运行时无需网络）
RUN pip install --no-cache-dir \
    "jupyterhub==5.3.*" \
    ipykernel \
    jupyterlab \
    jupyterlab-execute-time \
    jupyterlab-code-formatter \
    jupyterlab-lsp \
    python-lsp-server[all]

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

# ========================================
# 验证阶段：确保所有组件正常工作
# ========================================
# 验证关键组件是否正确安装
RUN python -c "import jupyterhub, jupyterlab, ipykernel; print('✓ 核心组件验证成功')" && \
    jupyter --version && \
    jupyter lab --version

LABEL maintainer="AI Infrastructure Team" \
    org.opencontainers.image.title="ai-infra-singleuser" \
    org.opencontainers.image.version="${APP_VERSION}" \
    org.opencontainers.image.description="AI Infra Matrix - Singleuser Notebook (Offline Build Mode)"

