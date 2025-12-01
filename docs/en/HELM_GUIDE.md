# Helm Chart Deployment Guide

**[中文文档](../HELM_GUIDE.md)** | **English**

## Overview

This guide describes how to quickly deploy AI Infrastructure Matrix using Helm Chart.

## Helm Chart Structure

```
helm/ai-infra-matrix/
├── Chart.yaml              # Chart metadata
├── values.yaml             # Default configuration
├── templates/              # Kubernetes resource templates
│   ├── deployment.yaml
│   ├── service.yaml
│   ├── ingress.yaml
│   ├── configmap.yaml
│   ├── secret.yaml
│   ├── statefulset.yaml
│   ├── pvc.yaml
│   └── _helpers.tpl        # Template helper functions
└── charts/                 # Dependency Charts (sub-charts)
    ├── postgresql/
    ├── mysql/
    ├── redis/
    └── minio/
```

## Quick Start

### 1. Install Helm

```bash
# macOS
brew install helm

# Linux
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Windows
choco install kubernetes-helm
```

### 2. Add Project Repository

```bash
# Clone project
git clone https://github.com/aresnasa/ai-infra-matrix.git
cd ai-infra-matrix/helm
```

### 3. View Chart Information

```bash
# View Chart details
helm show chart ai-infra-matrix

# View default configuration
helm show values ai-infra-matrix

# View all information
helm show all ai-infra-matrix
```

### 4. Install Chart

```bash
# Install with default configuration
helm install my-ai-infra ai-infra-matrix

# Specify namespace
helm install my-ai-infra ai-infra-matrix \
  --namespace ai-infra \
  --create-namespace

# Use custom configuration
helm install my-ai-infra ai-infra-matrix \
  --namespace ai-infra \
  --create-namespace \
  --values custom-values.yaml
```

## Configuration Options

### Global Configuration

```yaml
global:
  # Image registry
  imageRegistry: "docker.io"
  # Image pull secrets
  imagePullSecrets:
    - name: registry-secret
  # StorageClass
  storageClass: "standard"
  # Default resource limits
  resources:
    limits:
      cpu: "2"
      memory: "4Gi"
    requests:
      cpu: "500m"
      memory: "1Gi"
```

### Backend Configuration

```yaml
backend:
  enabled: true
  replicaCount: 2
  image:
    repository: aresnasa/ai-infra-backend
    tag: "v0.3.8"
    pullPolicy: IfNotPresent
  
  service:
    type: ClusterIP
    port: 8000
  
  env:
    - name: GIN_MODE
      value: "release"
    - name: DATABASE_URL
      valueFrom:
        secretKeyRef:
          name: ai-infra-secrets
          key: database-url
  
  resources:
    limits:
      cpu: "2"
      memory: "4Gi"
    requests:
      cpu: "500m"
      memory: "1Gi"
```

### Frontend Configuration

```yaml
frontend:
  enabled: true
  replicaCount: 2
  image:
    repository: aresnasa/ai-infra-frontend
    tag: "v0.3.8"
    pullPolicy: IfNotPresent
  
  service:
    type: ClusterIP
    port: 80
  
  resources:
    limits:
      cpu: "500m"
      memory: "512Mi"
    requests:
      cpu: "100m"
      memory: "128Mi"
```

### Database Configuration

```yaml
postgresql:
  enabled: true
  auth:
    postgresPassword: "your-password"
    database: "ai-infra-matrix"
  primary:
    persistence:
      enabled: true
      size: 20Gi
      storageClass: "standard"
    resources:
      limits:
        cpu: "2"
        memory: "4Gi"
      requests:
        cpu: "500m"
        memory: "1Gi"

mysql:
  enabled: true
  auth:
    rootPassword: "your-password"
    database: "slurm_acct_db"
  primary:
    persistence:
      enabled: true
      size: 10Gi

redis:
  enabled: true
  auth:
    password: "your-password"
  master:
    persistence:
      enabled: true
      size: 8Gi
```

### Ingress Configuration

```yaml
ingress:
  enabled: true
  className: "nginx"
  annotations:
    nginx.ingress.kubernetes.io/proxy-body-size: "100m"
    nginx.ingress.kubernetes.io/proxy-read-timeout: "3600"
    nginx.ingress.kubernetes.io/proxy-send-timeout: "3600"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
  
  hosts:
    - host: ai-infra.example.com
      paths:
        - path: /
          pathType: Prefix
          service:
            name: frontend
            port: 80
        - path: /api
          pathType: Prefix
          service:
            name: backend
            port: 8000
  
  tls:
    - secretName: ai-infra-tls
      hosts:
        - ai-infra.example.com
```

## Common Operations

### Update Configuration

```bash
# Update release configuration
helm upgrade my-ai-infra ai-infra-matrix \
  --namespace ai-infra \
  --values custom-values.yaml

# View update history
helm history my-ai-infra -n ai-infra
```

### Rollback

```bash
# Rollback to previous version
helm rollback my-ai-infra 1 -n ai-infra

# View available versions
helm history my-ai-infra -n ai-infra
```

### Uninstall

```bash
# Uninstall release
helm uninstall my-ai-infra -n ai-infra

# Also delete namespace
kubectl delete namespace ai-infra
```

### View Resources

```bash
# View release information
helm status my-ai-infra -n ai-infra

# View generated manifests
helm get manifest my-ai-infra -n ai-infra

# View current configuration
helm get values my-ai-infra -n ai-infra
```

## Advanced Configuration

### Enable HPA

```yaml
autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilizationPercentage: 80
  targetMemoryUtilizationPercentage: 80
```

### Configure PodDisruptionBudget

```yaml
podDisruptionBudget:
  enabled: true
  minAvailable: 1
  # Or use maxUnavailable
  # maxUnavailable: 1
```

### Configure Network Policies

```yaml
networkPolicy:
  enabled: true
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              name: ingress-nginx
  egress:
    - to:
        - namespaceSelector:
            matchLabels:
              name: kube-system
```

### Configure ServiceAccount

```yaml
serviceAccount:
  create: true
  name: "ai-infra-sa"
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::123456789:role/my-role
```

## Production Recommendations

1. **Resource Limits**: Always set resource requests and limits
2. **Replicas**: Use at least 2 replicas for critical services
3. **Persistent Storage**: Enable persistent storage for databases
4. **Secrets Management**: Use external secrets management (Vault, AWS Secrets Manager)
5. **TLS**: Always enable TLS in production
6. **Monitoring**: Deploy Prometheus and Grafana to monitor application status
7. **Logging**: Configure centralized logging (ELK, Loki)
8. **Backups**: Configure regular database backups

## Troubleshooting

### Chart Installation Failed

```bash
# View detailed error messages
helm install my-ai-infra ai-infra-matrix --debug --dry-run

# Check pod status
kubectl get pods -n ai-infra
kubectl describe pod <pod-name> -n ai-infra
```

### Configuration Not Taking Effect

```bash
# Verify applied configuration
helm get values my-ai-infra -n ai-infra

# Force update
helm upgrade my-ai-infra ai-infra-matrix \
  --namespace ai-infra \
  --values custom-values.yaml \
  --force
```

## Related Documentation

- [Kubernetes Deployment Guide](KUBERNETES_DEPLOYMENT.md)
- [Docker Hub Push Guide](DOCKER-HUB-PUSH.md)
- [Monitoring Guide](MONITORING.md)
