# ArgoCD GitOps 持续部署服务
# 版本: v2.13
# 用途: 为 AI Infrastructure Matrix 提供 GitOps 持续部署能力
# 支持架构: linux/amd64, linux/arm64

{{if .BASE_IMAGE_REGISTRY}}
ARG BASE_IMAGE_REGISTRY={{.BASE_IMAGE_REGISTRY}}
{{else}}
ARG BASE_IMAGE_REGISTRY=
{{end}}

FROM ${BASE_IMAGE_REGISTRY}quay.io/argoproj/argocd:{{.ARGOCD_VERSION | default "v2.13.3"}}

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
