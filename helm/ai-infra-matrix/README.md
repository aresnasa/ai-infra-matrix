# AI Infrastructure Matrix Helm Chart

A comprehensive Helm chart for deploying the AI Infrastructure Matrix, including JupyterHub, backend services, frontend, and Nginx gateway on Kubernetes.

## Overview

This chart deploys a complete AI infrastructure platform consisting of:

- **JupyterHub**: Multi-user Jupyter notebook server with Kubernetes spawner
- **Backend API**: Node.js/Express backend with authentication and data services
- **Frontend**: React-based web application
- **Nginx Gateway**: Reverse proxy and SSL termination
- **PostgreSQL**: Primary database (via Bitnami chart)
- **Redis**: Caching and session storage (via Bitnami chart)

## Prerequisites

- Kubernetes 1.19+ cluster
- Helm 3.8+
- Storage class for persistent volumes
- Ingress controller (optional, for external access)

## Installation

### 1. Add Bitnami Repository

```bash
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update
```

### 2. Install Dependencies

```bash
helm dependency update
```

### 3. Create Namespace

```bash
kubectl create namespace ai-infra-matrix
kubectl create namespace ai-infra-users  # For single-user pods
```

### 4. Install Chart

```bash
# Basic installation
helm install ai-infra-matrix ./helm/ai-infra-matrix -n ai-infra-matrix

# With custom values
helm install ai-infra-matrix ./helm/ai-infra-matrix -n ai-infra-matrix -f values-production.yaml
```

## Configuration

### Key Configuration Options

| Parameter | Description | Default |
|-----------|-------------|---------|
| `global.imageRegistry` | Docker registry for images | `""` |
| `nginx.service.type` | Nginx service type | `LoadBalancer` |
| `ingress.enabled` | Enable ingress | `false` |
| `jupyterhub.enabled` | Enable JupyterHub | `true` |
| `postgresql.enabled` | Enable PostgreSQL | `true` |
| `redis.enabled` | Enable Redis | `true` |

### Storage Configuration

```yaml
jupyterhub:
  persistence:
    enabled: true
    size: 20Gi
    storageClass: "fast-ssd"

sharedStorage:
  enabled: true
  size: 100Gi
  storageClass: "shared-nfs"

jupyterhub:
  singleuser:
    storage:
      dynamic: true
      capacity: "10Gi"
      storageClass: "standard"
```

### Resource Configuration

```yaml
jupyterhub:
  resources:
    requests:
      memory: "2Gi"
      cpu: "1000m"
    limits:
      memory: "4Gi"
      cpu: "2000m"

backend:
  resources:
    requests:
      memory: "512Mi"
      cpu: "250m"
    limits:
      memory: "1Gi"
      cpu: "500m"
```

### Authentication Configuration

```yaml
jupyterhub:
  auth:
    autoLogin: true
    adminUsers:
      admin: true
      researcher: true
    sessionTimeout: 480  # minutes
```

## Access Methods

### LoadBalancer (Recommended for Cloud)

```yaml
nginx:
  service:
    type: LoadBalancer
    port: 80
```

Access via external IP:
```bash
kubectl get svc ai-infra-matrix-nginx -n ai-infra-matrix
```

### Ingress (Recommended for Production)

```yaml
ingress:
  enabled: true
  className: "nginx"
  hosts:
    - host: ai-platform.example.com
      paths:
        - path: /
          pathType: Prefix
          service: nginx
          port: 80
  tls:
    - secretName: ai-platform-tls
      hosts:
        - ai-platform.example.com
```

### NodePort (Development/Testing)

```yaml
nginx:
  service:
    type: NodePort
    nodePort: 30080
```

### Port Forward (Local Development)

```bash
kubectl port-forward svc/ai-infra-matrix-nginx 8080:80 -n ai-infra-matrix
```

## Service Endpoints

- **Frontend**: `/` - React web application
- **Backend API**: `/api/*` - REST API endpoints
- **JupyterHub**: `/jupyter/*` - Multi-user Jupyter environment
- **User Notebooks**: `/jupyter/user/{username}/` - Individual user workspaces

## Monitoring and Troubleshooting

### Check Deployment Status

```bash
# Overall status
kubectl get all -n ai-infra-matrix

# Pod status with more details
kubectl get pods -n ai-infra-matrix -o wide

# Single-user pods
kubectl get pods -n ai-infra-users
```

### View Logs

```bash
# JupyterHub logs
kubectl logs -f deployment/ai-infra-matrix-jupyterhub -n ai-infra-matrix

# Backend logs
kubectl logs -f deployment/ai-infra-matrix-backend -n ai-infra-matrix

# Nginx logs
kubectl logs -f deployment/ai-infra-matrix-nginx -n ai-infra-matrix

# Single-user pod logs
kubectl logs jupyter-{username} -n ai-infra-users
```

### Common Issues

#### 1. Single-user Pods Failing to Start

```bash
# Check events
kubectl describe pod jupyter-{username} -n ai-infra-users

# Check RBAC permissions
kubectl auth can-i create pods --as=system:serviceaccount:ai-infra-matrix:ai-infra-matrix-jupyterhub -n ai-infra-users
```

#### 2. Storage Issues

```bash
# Check PVC status
kubectl get pvc -n ai-infra-matrix
kubectl get pvc -n ai-infra-users

# Check storage class
kubectl get storageclass
```

#### 3. Network Connectivity

```bash
# Test internal service connectivity
kubectl exec -it deployment/ai-infra-matrix-nginx -n ai-infra-matrix -- curl ai-infra-matrix-jupyterhub:8000/jupyter/hub/health
```

## Scaling

### Horizontal Pod Autoscaling

```yaml
backend:
  autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 10
    targetCPUUtilizationPercentage: 70

frontend:
  autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 5
    targetCPUUtilizationPercentage: 80
```

### Vertical Scaling

```bash
# Scale manually
kubectl scale deployment ai-infra-matrix-backend --replicas=3 -n ai-infra-matrix

# Update resource limits
helm upgrade ai-infra-matrix ./helm/ai-infra-matrix -n ai-infra-matrix --set backend.resources.limits.memory=2Gi
```

## Security

### RBAC

The chart creates appropriate RBAC resources:
- Service accounts for each component
- Roles for JupyterHub to manage single-user pods
- Cluster roles for node access (if needed)

### Network Policies

```yaml
networkPolicies:
  enabled: true
  ingress:
    enabled: true
  egress:
    enabled: true
```

### Pod Security Standards

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 1000
  fsGroup: 1000
  seccompProfile:
    type: RuntimeDefault
```

## Backup and Recovery

### Database Backup

```bash
# PostgreSQL backup
kubectl exec -it ai-infra-matrix-postgresql-0 -n ai-infra-matrix -- pg_dump -U ai_infra_user ai_infra_db > backup.sql
```

### JupyterHub Data Backup

```bash
# Backup JupyterHub persistent data
kubectl create job backup-jupyterhub --from=cronjob/backup-cronjob -n ai-infra-matrix
```

## Migration from Docker Compose

To migrate from the existing Docker Compose setup:

1. **Export Data**: Backup databases and user data
2. **Update Images**: Ensure container images are available in a registry
3. **Configure Values**: Map Docker Compose environment variables to Helm values
4. **Deploy**: Install the Helm chart
5. **Import Data**: Restore databases and user data
6. **Test**: Verify all functionality

## Upgrading

```bash
# Update dependencies
helm dependency update

# Upgrade release
helm upgrade ai-infra-matrix ./helm/ai-infra-matrix -n ai-infra-matrix

# Rollback if needed
helm rollback ai-infra-matrix 1 -n ai-infra-matrix
```

## Uninstallation

```bash
# Remove the release
helm uninstall ai-infra-matrix -n ai-infra-matrix

# Clean up PVCs (if desired)
kubectl delete pvc --all -n ai-infra-matrix
kubectl delete pvc --all -n ai-infra-users

# Remove namespaces
kubectl delete namespace ai-infra-matrix
kubectl delete namespace ai-infra-users
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make changes to the chart
4. Test with `helm template` and `helm lint`
5. Submit a pull request

## Support

For issues and questions:
- GitHub Issues: [Link to repository issues]
- Documentation: [Link to docs]
- Community: [Link to community channels]
