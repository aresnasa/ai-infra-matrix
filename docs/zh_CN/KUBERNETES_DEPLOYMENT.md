# Kubernetes 部署指南

**中文** | **[English](en/KUBERNETES_DEPLOYMENT.md)**

## 概述

本指南介绍如何将 AI Infrastructure Matrix 部署到 Kubernetes 集群。

## 前置要求

- Kubernetes 1.24+
- kubectl 已配置
- Helm 3.0+
- StorageClass 配置（用于持久化存储）
- LoadBalancer 或 Ingress Controller

## 部署架构

```
┌─────────────────────────────────────┐
│         Ingress/LoadBalancer        │
│       (nginx-ingress/traefik)       │
└──────────────┬──────────────────────┘
               │
    ┌──────────┴──────────┐
    │                     │
┌───▼────┐         ┌──────▼──────┐
│ Nginx  │         │  Services   │
│ (Pod)  │         │  (ClusterIP)│
└───┬────┘         └──────┬──────┘
    │                     │
┌───▼─────────────────────▼───────┐
│    Application Pods              │
│  - Frontend                      │
│  - Backend                       │
│  - JupyterHub                    │
│  - Gitea                         │
│  - Slurm Master                  │
│  - SaltStack                     │
│  - AppHub                        │
│  - Nightingale                   │
└──────────┬───────────────────────┘
           │
┌──────────▼───────────────────────┐
│    Stateful Services             │
│  - PostgreSQL (StatefulSet)      │
│  - MySQL (StatefulSet)           │
│  - Redis (StatefulSet)           │
│  - Kafka (StatefulSet)           │
│  - MinIO (StatefulSet)           │
│  - OceanBase (StatefulSet)       │
└──────────┬───────────────────────┘
           │
┌──────────▼───────────────────────┐
│    Persistent Volumes            │
│  (PV/PVC with StorageClass)      │
└──────────────────────────────────┘
```

## 使用 Helm 部署

### 1. 添加 Helm 仓库

```bash
# 克隆项目
git clone https://github.com/aresnasa/ai-infra-matrix.git
cd ai-infra-matrix
```

### 2. 配置 values.yaml

```bash
# 复制示例配置
cp helm/ai-infra-matrix/values.yaml helm/ai-infra-matrix/values.custom.yaml

# 编辑配置
vi helm/ai-infra-matrix/values.custom.yaml
```

关键配置项：

```yaml
# 镜像仓库
global:
  imageRegistry: "your-registry.com/ai-infra-matrix"
  imageTag: "v0.3.8"
  
# 存储类
persistence:
  storageClass: "standard"  # 根据集群配置修改
  
# Ingress 配置
ingress:
  enabled: true
  className: "nginx"
  host: "ai-infra.example.com"
  tls:
    enabled: true
    secretName: "ai-infra-tls"

# 数据库配置
postgresql:
  enabled: true
  persistence:
    size: 50Gi
    
mysql:
  enabled: true
  persistence:
    size: 50Gi

# 资源配置
resources:
  backend:
    requests:
      cpu: 500m
      memory: 1Gi
    limits:
      cpu: 2000m
      memory: 4Gi
```

### 3. 安装

```bash
# 创建命名空间
kubectl create namespace ai-infra

# 部署
helm install ai-infra-matrix ./helm/ai-infra-matrix \
  --namespace ai-infra \
  --values helm/ai-infra-matrix/values.custom.yaml
```

### 4. 验证部署

```bash
# 查看 Pod 状态
kubectl get pods -n ai-infra

# 查看服务
kubectl get svc -n ai-infra

# 查看 Ingress
kubectl get ingress -n ai-infra
```

## 手动部署（使用 kubectl）

### 1. 创建命名空间

```bash
kubectl create namespace ai-infra
```

### 2. 创建 ConfigMap

```bash
kubectl create configmap ai-infra-config \
  --from-file=.env \
  --namespace ai-infra
```

### 3. 创建 Secret

```bash
# 数据库密码
kubectl create secret generic db-credentials \
  --from-literal=postgres-password=yourpassword \
  --from-literal=mysql-password=yourpassword \
  --namespace ai-infra

# MinIO 凭据
kubectl create secret generic minio-credentials \
  --from-literal=root-user=minioadmin \
  --from-literal=root-password=minioadmin \
  --namespace ai-infra
```

### 4. 创建 PersistentVolumeClaim

```yaml
# postgres-pvc.yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: postgres-data
  namespace: ai-infra
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 50Gi
  storageClassName: standard
```

```bash
kubectl apply -f postgres-pvc.yaml
```

### 5. 部署数据库

```yaml
# postgres-deployment.yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: postgres
  namespace: ai-infra
spec:
  serviceName: postgres
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:15-alpine
        env:
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db-credentials
              key: postgres-password
        - name: POSTGRES_DB
          value: ai-infra-matrix
        ports:
        - containerPort: 5432
        volumeMounts:
        - name: data
          mountPath: /var/lib/postgresql/data
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: ["ReadWriteOnce"]
      storageClassName: standard
      resources:
        requests:
          storage: 50Gi
---
apiVersion: v1
kind: Service
metadata:
  name: postgres
  namespace: ai-infra
spec:
  selector:
    app: postgres
  ports:
  - port: 5432
  clusterIP: None
```

### 6. 部署应用服务

```yaml
# backend-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backend
  namespace: ai-infra
spec:
  replicas: 2
  selector:
    matchLabels:
      app: backend
  template:
    metadata:
      labels:
        app: backend
    spec:
      containers:
      - name: backend
        image: your-registry.com/ai-infra-matrix/backend:v0.3.8
        ports:
        - containerPort: 8000
        env:
        - name: POSTGRES_HOST
          value: postgres
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db-credentials
              key: postgres-password
        resources:
          requests:
            cpu: 500m
            memory: 1Gi
          limits:
            cpu: 2000m
            memory: 4Gi
---
apiVersion: v1
kind: Service
metadata:
  name: backend
  namespace: ai-infra
spec:
  selector:
    app: backend
  ports:
  - port: 8000
    targetPort: 8000
```

### 7. 配置 Ingress

```yaml
# ingress.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ai-infra-ingress
  namespace: ai-infra
  annotations:
    nginx.ingress.kubernetes.io/proxy-body-size: "100m"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "600"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - ai-infra.example.com
    secretName: ai-infra-tls
  rules:
  - host: ai-infra.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: nginx
            port:
              number: 80
      - path: /api
        pathType: Prefix
        backend:
          service:
            name: backend
            port:
              number: 8000
      - path: /jupyter
        pathType: Prefix
        backend:
          service:
            name: jupyterhub
            port:
              number: 8000
```

## 高可用配置

### 数据库高可用

使用 Operator 部署高可用数据库：

```bash
# PostgreSQL Operator
helm install postgres-operator \
  oci://registry-1.docker.io/bitnamicharts/postgresql-ha \
  --namespace ai-infra

# MySQL Operator
helm install mysql-operator \
  oci://registry-1.docker.io/bitnamicharts/mysql \
  --namespace ai-infra
```

### 应用服务高可用

- 设置多副本（replicas >= 2）
- 配置 Pod 反亲和性
- 使用 HorizontalPodAutoscaler

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: backend-hpa
  namespace: ai-infra
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: backend
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

## 监控和日志

### Prometheus 监控

```bash
# 安装 Prometheus
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace

# 配置 ServiceMonitor
kubectl apply -f monitoring/servicemonitor.yaml
```

### 日志收集

```bash
# 安装 EFK Stack
kubectl apply -f https://download.elastic.co/downloads/eck/2.10.0/crds.yaml
kubectl apply -f https://download.elastic.co/downloads/eck/2.10.0/operator.yaml
```

## 备份和恢复

### 使用 Velero 备份

```bash
# 安装 Velero
velero install \
  --provider aws \
  --plugins velero/velero-plugin-for-aws:v1.8.0 \
  --bucket velero-backups \
  --backup-location-config region=us-west-2

# 备份命名空间
velero backup create ai-infra-backup --include-namespaces ai-infra

# 恢复
velero restore create --from-backup ai-infra-backup
```

## 升级

```bash
# 使用 Helm 升级
helm upgrade ai-infra-matrix ./helm/ai-infra-matrix \
  --namespace ai-infra \
  --values helm/ai-infra-matrix/values.custom.yaml \
  --set global.imageTag=v0.3.9

# 查看升级历史
helm history ai-infra-matrix -n ai-infra

# 回滚
helm rollback ai-infra-matrix <revision> -n ai-infra
```

## 故障排查

### 查看 Pod 日志

```bash
kubectl logs -f <pod-name> -n ai-infra
kubectl logs -f <pod-name> -c <container-name> -n ai-infra
```

### 进入 Pod 调试

```bash
kubectl exec -it <pod-name> -n ai-infra -- /bin/bash
```

### 查看事件

```bash
kubectl get events -n ai-infra --sort-by='.lastTimestamp'
```

### 查看资源使用

```bash
kubectl top pods -n ai-infra
kubectl top nodes
```

## 安全配置

### NetworkPolicy

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: backend-network-policy
  namespace: ai-infra
spec:
  podSelector:
    matchLabels:
      app: backend
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: nginx
    ports:
    - protocol: TCP
      port: 8000
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: postgres
    ports:
    - protocol: TCP
      port: 5432
```

### Pod Security Standards

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: ai-infra
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
```

## 性能优化

- 使用 NodeSelector 和 Affinity 优化 Pod 调度
- 配置资源 requests 和 limits
- 使用本地存储提升 I/O 性能
- 启用 CDN 加速静态资源
- 配置数据库连接池

## 参考资源

- [Kubernetes 官方文档](https://kubernetes.io/docs/)
- [Helm 官方文档](https://helm.sh/docs/)
- [项目 Helm Chart](../helm/ai-infra-matrix/)
