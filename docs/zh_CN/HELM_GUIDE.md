# Helm Chart 部署指南

**中文** | **[English](en/HELM_GUIDE.md)**

## 概述

本指南介绍如何使用 Helm Chart 快速部署 AI Infrastructure Matrix。

## Helm Chart 结构

```
helm/ai-infra-matrix/
├── Chart.yaml              # Chart 元数据
├── values.yaml            # 默认配置
├── templates/             # Kubernetes 资源模板
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── ingress.yaml
│   ├── configmap.yaml
│   ├── secret.yaml
│   ├── statefulset.yaml
│   ├── pvc.yaml
│   └── _helpers.tpl       # 模板辅助函数
└── charts/                # 依赖 Chart（子 Chart）
    ├── postgresql/
    ├── mysql/
    ├── redis/
    └── seaweedfs/
```

## 快速开始

### 1. 安装 Helm

```bash
# macOS
brew install helm

# Linux
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Windows
choco install kubernetes-helm
```

### 2. 添加项目仓库

```bash
# 克隆项目
git clone https://github.com/aresnasa/ai-infra-matrix.git
cd ai-infra-matrix/helm
```

### 3. 查看 Chart 信息

```bash
# 查看 Chart 详情
helm show chart ai-infra-matrix

# 查看默认配置
helm show values ai-infra-matrix

# 查看所有信息
helm show all ai-infra-matrix
```

### 4. 安装 Chart

```bash
# 使用默认配置安装
helm install my-ai-infra ai-infra-matrix

# 指定命名空间
helm install my-ai-infra ai-infra-matrix \
  --namespace ai-infra \
  --create-namespace

# 使用自定义配置
helm install my-ai-infra ai-infra-matrix \
  --namespace ai-infra \
  --create-namespace \
  --values custom-values.yaml
```

## 配置说明

### 全局配置

```yaml
# values.yaml
global:
  # 镜像仓库配置
  imageRegistry: "docker.io"
  imagePullPolicy: IfNotPresent
  imageTag: "v0.3.8"
  
  # 镜像拉取凭据
  imagePullSecrets:
    - name: registry-secret
  
  # 存储类
  storageClass: "standard"
  
  # 时区
  timezone: "Asia/Shanghai"
```

### 应用服务配置

```yaml
# Backend 配置
backend:
  enabled: true
  replicaCount: 2
  image:
    repository: ai-infra-matrix/backend
    tag: v0.3.8
  
  resources:
    requests:
      cpu: 500m
      memory: 1Gi
    limits:
      cpu: 2000m
      memory: 4Gi
  
  autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 10
    targetCPUUtilizationPercentage: 70
  
  env:
    - name: LOG_LEVEL
      value: "info"
    - name: POSTGRES_HOST
      value: "postgres"

# Frontend 配置
frontend:
  enabled: true
  replicaCount: 2
  image:
    repository: ai-infra-matrix/frontend
    tag: v0.3.8
  
  resources:
    requests:
      cpu: 100m
      memory: 256Mi
    limits:
      cpu: 500m
      memory: 512Mi

# JupyterHub 配置
jupyterhub:
  enabled: true
  replicaCount: 1
  image:
    repository: ai-infra-matrix/jupyterhub
    tag: v0.3.8
  
  config:
    spawner:
      image: ai-infra-matrix/singleuser:v0.3.8
      cpu_limit: 2
      mem_limit: 4G

# Gitea 配置
gitea:
  enabled: true
  image:
    repository: ai-infra-matrix/gitea
    tag: v0.3.8
  
  persistence:
    enabled: true
    size: 50Gi
  
  config:
    lfs:
      enabled: true
      storage: seaweedfs

# Slurm Master 配置
slurmMaster:
  enabled: true
  image:
    repository: ai-infra-matrix/slurm-master
    tag: v0.3.8
  
  persistence:
    enabled: true
    size: 20Gi

# AppHub 配置
apphub:
  enabled: true
  image:
    repository: ai-infra-matrix/apphub
    tag: v0.3.8
  
  persistence:
    enabled: true
    size: 100Gi

# Nightingale 配置
nightingale:
  enabled: true
  image:
    repository: ai-infra-matrix/nightingale
    tag: v0.3.8
  
  persistence:
    enabled: true
    size: 50Gi
```

### 数据库配置

```yaml
# PostgreSQL
postgresql:
  enabled: true
  image:
    repository: postgres
    tag: 15-alpine
  
  auth:
    username: postgres
    password: "changeme"
    database: ai-infra-matrix
  
  persistence:
    enabled: true
    size: 50Gi
    storageClass: standard
  
  resources:
    requests:
      cpu: 500m
      memory: 1Gi
    limits:
      cpu: 2000m
      memory: 4Gi

# MySQL
mysql:
  enabled: true
  image:
    repository: mysql
    tag: "8.0"
  
  auth:
    rootPassword: "changeme"
    database: slurm_acct_db
    username: slurm
    password: "changeme"
  
  persistence:
    enabled: true
    size: 50Gi

# Redis
redis:
  enabled: true
  image:
    repository: redis
    tag: 7-alpine
  
  persistence:
    enabled: true
    size: 10Gi

# Kafka
kafka:
  enabled: true
  replicaCount: 1
  
  persistence:
    enabled: true
    size: 50Gi

# SeaweedFS
seaweedfs:
  enabled: true
  
  accessKey: seaweedfs_admin
  secretKey: seaweedfs_secret_key_change_me
  
  persistence:
    enabled: true
    size: 100Gi
  
  resources:
    requests:
      cpu: 500m
      memory: 1Gi

# OceanBase
oceanbase:
  enabled: true
  image:
    repository: oceanbase/oceanbase-ce
    tag: 4.3.5-lts
  
  persistence:
    enabled: true
    size: 100Gi
```

### Ingress 配置

```yaml
ingress:
  enabled: true
  className: "nginx"
  
  annotations:
    nginx.ingress.kubernetes.io/proxy-body-size: "100m"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "600"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
  
  host: "ai-infra.example.com"
  
  tls:
    enabled: true
    secretName: ai-infra-tls
  
  paths:
    - path: /
      service: nginx
      port: 80
    - path: /api
      service: backend
      port: 8000
    - path: /jupyter
      service: jupyterhub
      port: 8000
    - path: /gitea
      service: gitea
      port: 3000
    - path: /n9e
      service: nightingale
      port: 18000
```

### 持久化存储配置

```yaml
persistence:
  # 存储类
  storageClass: "standard"
  
  # 各组件存储配置
  volumes:
    postgres:
      enabled: true
      size: 50Gi
      accessMode: ReadWriteOnce
    
    mysql:
      enabled: true
      size: 50Gi
      accessMode: ReadWriteOnce
    
    gitea:
      enabled: true
      size: 50Gi
      accessMode: ReadWriteOnce
    
    seaweedfs:
      enabled: true
      size: 100Gi
      accessMode: ReadWriteOnce
    
    apphub:
      enabled: true
      size: 100Gi
      accessMode: ReadWriteOnce
```

## 常用操作

### 安装 Chart

```bash
# 基础安装
helm install my-release ai-infra-matrix

# 指定版本
helm install my-release ai-infra-matrix --version 0.3.8

# 使用自定义配置文件
helm install my-release ai-infra-matrix -f custom-values.yaml

# 命令行覆盖配置
helm install my-release ai-infra-matrix \
  --set backend.replicaCount=3 \
  --set postgresql.auth.password=mypassword
```

### 查看状态

```bash
# 查看 Release 列表
helm list -n ai-infra

# 查看 Release 详情
helm status my-release -n ai-infra

# 查看 Release 历史
helm history my-release -n ai-infra

# 查看实际应用的配置
helm get values my-release -n ai-infra
```

### 升级 Chart

```bash
# 基础升级
helm upgrade my-release ai-infra-matrix

# 升级并修改配置
helm upgrade my-release ai-infra-matrix \
  -f custom-values.yaml \
  --set global.imageTag=v0.3.9

# 强制升级（重新创建 Pod）
helm upgrade my-release ai-infra-matrix --force

# 升级前模拟（不实际执行）
helm upgrade my-release ai-infra-matrix --dry-run --debug
```

### 回滚

```bash
# 回滚到上一个版本
helm rollback my-release

# 回滚到指定版本
helm rollback my-release 2

# 查看回滚历史
helm history my-release -n ai-infra
```

### 卸载 Chart

```bash
# 卸载 Release
helm uninstall my-release -n ai-infra

# 卸载并保留历史记录
helm uninstall my-release -n ai-infra --keep-history

# 删除命名空间
kubectl delete namespace ai-infra
```

## 高级配置

### 使用外部数据库

```yaml
# 禁用内置数据库，使用外部数据库
postgresql:
  enabled: false

# 配置外部数据库连接
externalDatabase:
  host: external-postgres.example.com
  port: 5432
  database: ai-infra-matrix
  username: postgres
  password: "secretpassword"
  
  # 使用 Secret 存储密码（推荐）
  existingSecret: "postgres-credentials"
  existingSecretPasswordKey: "password"
```

### 多环境配置

```bash
# 开发环境
helm install dev-release ai-infra-matrix \
  -f values-dev.yaml \
  -n ai-infra-dev

# 测试环境
helm install test-release ai-infra-matrix \
  -f values-test.yaml \
  -n ai-infra-test

# 生产环境
helm install prod-release ai-infra-matrix \
  -f values-prod.yaml \
  -n ai-infra-prod
```

### 依赖管理

```bash
# 更新 Chart 依赖
helm dependency update ai-infra-matrix

# 构建 Chart 包
helm package ai-infra-matrix

# 推送到 Chart 仓库
helm push ai-infra-matrix-0.3.8.tgz oci://registry.example.com/charts
```

## 模板调试

```bash
# 渲染模板但不安装
helm template my-release ai-infra-matrix

# 指定配置文件渲染
helm template my-release ai-infra-matrix -f custom-values.yaml

# 渲染特定模板
helm template my-release ai-infra-matrix -s templates/deployment.yaml

# 完整调试输出
helm install my-release ai-infra-matrix --dry-run --debug
```

## 测试 Chart

```bash
# 运行 Chart 测试
helm test my-release -n ai-infra

# 查看测试日志
kubectl logs -n ai-infra -l "app.kubernetes.io/component=test"
```

## 最佳实践

### 1. 版本管理

- 使用语义化版本号
- 在 Chart.yaml 中明确版本
- 标记重要版本的 Git tag

### 2. 配置管理

- 使用 values.yaml 提供默认配置
- 为不同环境创建独立的 values 文件
- 敏感信息使用 Secret 或外部密钥管理

### 3. 资源限制

- 为所有容器设置 requests 和 limits
- 根据实际负载调整资源配额
- 启用 HPA 实现自动扩缩容

### 4. 持久化存储

- 生产环境必须启用持久化
- 选择合适的 StorageClass
- 定期备份重要数据

### 5. 安全配置

- 修改默认密码
- 使用 TLS 加密通信
- 配置 NetworkPolicy 限制流量
- 定期更新镜像版本

## 故障排查

### Chart 安装失败

```bash
# 查看详细错误信息
helm install my-release ai-infra-matrix --debug

# 检查 Kubernetes 事件
kubectl get events -n ai-infra --sort-by='.lastTimestamp'

# 检查 Pod 状态
kubectl get pods -n ai-infra
kubectl describe pod <pod-name> -n ai-infra
```

### 升级失败

```bash
# 查看升级历史
helm history my-release -n ai-infra

# 回滚到稳定版本
helm rollback my-release <revision> -n ai-infra

# 强制删除失败的 Release
helm uninstall my-release -n ai-infra --no-hooks
```

## 参考资源

- [Helm 官方文档](https://helm.sh/docs/)
- [Chart 最佳实践](https://helm.sh/docs/chart_best_practices/)
- [项目 Helm Chart 源码](../helm/ai-infra-matrix/)
- [Kubernetes 部署指南](KUBERNETES_DEPLOYMENT.md)
