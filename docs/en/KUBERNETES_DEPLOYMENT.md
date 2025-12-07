# Kubernetes Deployment Guide

**[中文文档](../KUBERNETES_DEPLOYMENT.md)** | **English**

## Overview

This guide describes how to deploy AI Infrastructure Matrix to a Kubernetes cluster.

## Prerequisites

- Kubernetes 1.24+
- kubectl configured
- Helm 3.0+
- StorageClass configured (for persistent storage)
- LoadBalancer or Ingress Controller

## Deployment Architecture

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
│  - SeaweedFS (StatefulSet)       │
│  - OceanBase (StatefulSet)       │
└──────────┬───────────────────────┘
           │
┌──────────▼───────────────────────┐
│    Persistent Volumes            │
│  (PV/PVC with StorageClass)      │
└──────────────────────────────────┘
```

## Deploy with Helm

### 1. Add Helm Repository

```bash
# Clone project
git clone https://github.com/aresnasa/ai-infra-matrix.git
cd ai-infra-matrix
```

### 2. Configure values.yaml

```bash
# Copy example configuration
cp helm/ai-infra-matrix/values.yaml helm/ai-infra-matrix/values.custom.yaml

# Edit configuration
vi helm/ai-infra-matrix/values.custom.yaml
```

Key configuration items:

```yaml
# Image registry
global:
  imageRegistry: "docker.io"
  imagePullSecrets: []
  storageClass: "standard"

# Ingress configuration
ingress:
  enabled: true
  className: "nginx"
  annotations:
    nginx.ingress.kubernetes.io/proxy-body-size: "100m"
  hosts:
    - host: ai-infra.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: ai-infra-tls
      hosts:
        - ai-infra.example.com

# PostgreSQL configuration
postgresql:
  enabled: true
  auth:
    postgresPassword: "your-secure-password"
    database: "ai-infra-matrix"
  primary:
    persistence:
      enabled: true
      size: 20Gi

# Redis configuration
redis:
  enabled: true
  auth:
    password: "your-secure-password"
  master:
    persistence:
      enabled: true
      size: 8Gi
```

### 3. Install

```bash
# Create namespace
kubectl create namespace ai-infra

# Install Chart
helm install ai-infra ./helm/ai-infra-matrix \
  --namespace ai-infra \
  --values helm/ai-infra-matrix/values.custom.yaml
```

### 4. Verify Installation

```bash
# Check pod status
kubectl get pods -n ai-infra

# Check services
kubectl get svc -n ai-infra

# Check ingress
kubectl get ingress -n ai-infra
```

## Manual Deployment

### 1. Create Namespace

```bash
kubectl create namespace ai-infra
```

### 2. Create ConfigMaps and Secrets

```bash
# Create secrets
kubectl create secret generic ai-infra-secrets \
  --namespace ai-infra \
  --from-literal=postgres-password=your-password \
  --from-literal=mysql-root-password=your-password \
  --from-literal=redis-password=your-password \
  --from-literal=seaweedfs-access-key=seaweedfs_admin \
  --from-literal=seaweedfs-secret-key=seaweedfs_secret_key_change_me

# Create configmap
kubectl create configmap ai-infra-config \
  --namespace ai-infra \
  --from-file=./config/
```

### 3. Deploy Database Services

```bash
# PostgreSQL
kubectl apply -f kubernetes/postgresql/

# MySQL
kubectl apply -f kubernetes/mysql/

# Redis
kubectl apply -f kubernetes/redis/
```

### 4. Deploy Application Services

```bash
# Backend
kubectl apply -f kubernetes/backend/

# Frontend
kubectl apply -f kubernetes/frontend/

# JupyterHub
kubectl apply -f kubernetes/jupyterhub/

# Other services
kubectl apply -f kubernetes/
```

## High Availability Configuration

### PostgreSQL HA

```yaml
postgresql:
  architecture: replication
  replication:
    enabled: true
    readReplicas: 2
  primary:
    persistence:
      enabled: true
      size: 50Gi
```

### Redis Cluster

```yaml
redis:
  architecture: replication
  replica:
    replicaCount: 2
  sentinel:
    enabled: true
```

## Monitoring and Logging

### Deploy Prometheus Stack

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace
```

### Configure ServiceMonitor

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: ai-infra-backend
  namespace: ai-infra
spec:
  selector:
    matchLabels:
      app: backend
  endpoints:
    - port: metrics
      interval: 30s
```

## Backup Strategy

### Database Backup

```bash
# PostgreSQL backup CronJob
apiVersion: batch/v1
kind: CronJob
metadata:
  name: postgres-backup
  namespace: ai-infra
spec:
  schedule: "0 2 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: postgres:13
            command:
            - /bin/sh
            - -c
            - pg_dump -h postgres -U postgres ai-infra-matrix > /backup/backup-$(date +%Y%m%d).sql
            volumeMounts:
            - name: backup
              mountPath: /backup
          volumes:
          - name: backup
            persistentVolumeClaim:
              claimName: backup-pvc
          restartPolicy: OnFailure
```

## Troubleshooting

### Common Issues

#### Pods Not Starting

```bash
# Check pod events
kubectl describe pod <pod-name> -n ai-infra

# Check logs
kubectl logs <pod-name> -n ai-infra
```

#### PVC Pending

```bash
# Check StorageClass
kubectl get storageclass

# Check PVC status
kubectl describe pvc <pvc-name> -n ai-infra
```

#### Service Unreachable

```bash
# Check endpoints
kubectl get endpoints -n ai-infra

# Test connectivity
kubectl run test --rm -it --image=busybox -- wget -qO- http://backend:8000/health
```

## Scaling

### Horizontal Pod Autoscaler

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
        averageUtilization: 80
```

## Related Documentation

- [Helm Chart Guide](HELM_GUIDE.md)
- [Docker Hub Push Guide](DOCKER-HUB-PUSH.md)
- [Monitoring Guide](MONITORING.md)
- [Backup & Recovery](BACKUP_RECOVERY.md)
