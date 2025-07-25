# JupyterHub Python CPU执行环境
FROM python:3.9-slim

# 设置环境变量
ENV PYTHONUNBUFFERED=1

# 安装系统依赖
RUN apt-get update && apt-get install -y \
    build-essential \
    git \
    curl \
    wget \
    vim \
    htop \
    && rm -rf /var/lib/apt/lists/*

# 升级pip
RUN pip install --upgrade pip setuptools wheel

# 安装常用的Python数据科学库
RUN pip install \
    numpy \
    pandas \
    matplotlib \
    seaborn \
    scikit-learn \
    scipy \
    jupyter \
    ipython \
    requests \
    pyyaml \
    pillow \
    flask \
    fastapi \
    uvicorn

# 创建工作目录
WORKDIR /workspace

# 创建共享目录
RUN mkdir -p /shared

# 设置默认命令
CMD ["/bin/bash"]
