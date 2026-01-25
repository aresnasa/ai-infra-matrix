# ArgoCD GitOps 持续部署服务
# 版本: {{ARGOCD_VERSION}}
# 用途: 为 AI Infrastructure Matrix 提供 GitOps 持续部署能力
# 支持架构: linux/amd64, linux/arm64

ARG ARGOCD_VERSION={{ARGOCD_VERSION}}

FROM quay.io/argoproj/argocd:${ARGOCD_VERSION}

# 设置时区
ENV TZ=Asia/Shanghai

# 复制自定义配置
COPY argocd-cm.yaml /home/argocd/
COPY argocd-rbac-cm.yaml /home/argocd/

# 切换到 argocd 用户
USER argocd

# 工作目录
WORKDIR /home/argocd

# 暴露端口
# 8080 - ArgoCD Server HTTP
# 8083 - ArgoCD Server Metrics
EXPOSE 8080 8083

# 入口点由基础镜像提供
