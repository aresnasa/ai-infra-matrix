# JupyterHub多节点部署指南

本指南说明如何将JupyterHub从单节点localhost部署改造为支持多节点分布式部署的Kubernetes集群。

## 问题描述

原有架构存在以下限制：
- 使用DockerSpawner，仅支持单节点部署
- 配置为localhost访问，无法跨节点访问
- 无法实现分布式微服务架构

## 解决方案

### 1. 架构改造

从**DockerSpawner**改为**KubeSpawner**，实现：
- Kubernetes原生Pod-per-user spawning
- 分布式服务发现和负载均衡
- 多节点资源调度
- 统一存储和网络管理

### 2. 配置更新

#### 主要变更：
- **Spawner类型**: `DockerSpawner` → `KubeSpawner`
- **网络访问**: `localhost:8080` → `192.168.0.199:8080`
- **存储方式**: 本地存储 → Kubernetes PVC + 共享存储
- **用户隔离**: 容器网络 → Kubernetes namespace

#### 关键配置项：
```yaml
# JupyterHub Spawner配置
JUPYTERHUB_SPAWNER: "kubernetes"
KUBERNETES_NAMESPACE: "ai-infra-users"
KUBERNETES_SERVICE_ACCOUNT: "ai-infra-matrix-jupyterhub"
JUPYTERHUB_STORAGE_CLASS: "local-path"
SHARED_STORAGE_CLASS: "nfs-client"
```

### 3. 部署方式

#### 快速部署：
```bash
# 1. 使用新的Kubernetes部署脚本
./scripts/deploy-k8s.sh deploy

# 2. 或者使用Helm直接部署
helm install ai-infra-matrix ./helm/ai-infra-matrix \
  --namespace ai-infra-matrix \
  --values ./helm/ai-infra-matrix/values-k8s-prod.yaml \
  --create-namespace

# 3. 查看部署状态
./scripts/deploy-k8s.sh status
```

#### 配置文件对比：

| 配置项 | 原值(单节点) | 新值(多节点) |
|--------|-------------|-------------|
| `DOMAIN` | `localhost` | `192.168.0.199` |
| `JUPYTERHUB_PUBLIC_HOST` | `localhost:8080` | `192.168.0.199:8080` |
| `JUPYTERHUB_SPAWNER` | `docker` | `kubernetes` |
| `JUPYTERHUB_NETWORK` | `ai-infra-network` | `pod` |
| `存储方式` | `disable` | `dynamic PVC` |

## 核心特性

### 1. 动态Spawner切换
支持通过环境变量控制spawner类型：
```python
# 环境变量: JUPYTERHUB_SPAWNER=kubernetes|docker
if SPAWNER_TYPE == 'kubernetes' and KUBESPAWNER_AVAILABLE:
    configure_kubespawner(c)
else:
    # 回退到DockerSpawner
    c.JupyterHub.spawner_class = ContainerSpawner
```

### 2. 资源管理
- **CPU限制**: 1.0 cores (保证 0.5 cores)
- **内存限制**: 2G (保证 1G)
- **存储**: 10Gi per user + 50Gi 共享存储
- **启动超时**: 300秒 (适应镜像拉取)

### 3. 安全配置
- 非root用户运行 (UID: 1000, GID: 100)
- 禁用特权升级
- 自动挂载服务账户token禁用
- Pod安全上下文限制

### 4. 网络配置
- 支持多节点间通信
- 正确的CORS配置
- Service discovery集成

## 部署验证

### 1. 检查Pod状态
```bash
# 查看主要组件
kubectl get pods -n ai-infra-matrix

# 查看用户Pod
kubectl get pods -n ai-infra-users
```

### 2. 验证网络访问
```bash
# 测试主页面访问
curl -I http://192.168.0.199:8080

# 测试JupyterHub访问  
curl -I http://192.168.0.199:8080/jupyter
```

### 3. 测试用户登录
1. 访问: `http://192.168.0.199:8080/jupyter`
2. 登录用户: `admin` / `demo-password`
3. 验证Pod创建: `kubectl get pods -n ai-infra-users`

## 故障排除

### 1. Spawner相关
```bash
# 查看JupyterHub日志
kubectl logs -n ai-infra-matrix deployment/ai-infra-matrix-jupyterhub

# 查看用户Pod启动问题
kubectl describe pod -n ai-infra-users <user-pod-name>
```

### 2. 网络问题
```bash
# 检查Service状态
kubectl get svc -n ai-infra-matrix

# 检查NodePort访问
kubectl get svc ai-infra-matrix-nginx -o yaml
```

### 3. 存储问题
```bash
# 检查PVC状态
kubectl get pvc -n ai-infra-matrix
kubectl get pvc -n ai-infra-users

# 检查StorageClass
kubectl get storageclass
```

## 配置文件位置

- **Kubernetes Values**: `helm/ai-infra-matrix/values-k8s-prod.yaml`
- **环境变量**: `.env.k8s.prod`
- **JupyterHub配置**: `src/jupyterhub/backend_integrated_config.py`
- **KubeSpawner配置**: `src/jupyterhub/kubernetes_spawner_config.py`
- **部署脚本**: `scripts/deploy-k8s.sh`

## 升级路径

从单节点切换到多节点：

1. **数据备份**
   ```bash
   # 备份数据库
   kubectl exec -it deployment/ai-infra-matrix-postgresql -- pg_dump ai_infra_db > backup.sql
   ```

2. **配置切换**
   ```bash
   # 切换到新配置
   cp .env.k8s.prod .env.prod
   ```

3. **重新部署**
   ```bash
   ./scripts/deploy-k8s.sh deploy
   ```

4. **数据恢复**
   ```bash
   # 恢复数据库（如需要）
   kubectl exec -i deployment/ai-infra-matrix-postgresql -- psql ai_infra_db < backup.sql
   ```

## 性能建议

1. **存储优化**
   - 使用SSD存储类
   - 配置共享存储提高数据共享效率

2. **资源调度**
   - 配置节点亲和性
   - 设置资源限制防止资源争抢

3. **网络优化**
   - 使用CNI网络插件
   - 配置网络策略提高安全性

## 总结

通过本次改造，JupyterHub已从单节点localhost部署成功转换为支持多节点的分布式Kubernetes部署：

✅ **已完成**：
- KubeSpawner集成
- 多节点网络配置
- 分布式存储支持
- Kubernetes RBAC配置
- 部署自动化脚本

✅ **测试验证**：
- 访问地址：`http://192.168.0.199:8080/jupyter`
- 用户Pod自动调度到集群节点
- 持久化存储和共享存储工作正常

现在JupyterHub支持真正的分布式微服务架构，可以在多个Kubernetes节点上运行和扩展！
