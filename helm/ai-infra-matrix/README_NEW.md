# AI Infrastructure Matrix - Helm Chart

这是AI Infrastructure Matrix平台的Kubernetes Helm Chart，提供了完整的AI基础设施栈，包括JupyterHub、后端服务、统一认证等功能。

## 🏗️ 架构组件

### 核心服务
- **Backend API** - Go/Gin REST API服务
- **Frontend** - React Web应用
- **JupyterHub** - 交互式计算环境
- **Nginx** - 反向代理和负载均衡

### 存储和数据库
- **PostgreSQL** - 主数据库
- **Redis** - 缓存和会话存储
- **SeaweedFS** - 对象存储服务 (S3兼容)

### 认证和权限
- **OpenLDAP** - 目录服务
- **Gitea** - Git仓库管理
- **phpLDAPadmin** - LDAP管理界面

## 🚀 快速开始

### 前置要求

1. **Kubernetes集群** (v1.24+)
2. **Helm** (v3.8+)
3. **kubectl** 已配置并连接到集群

### 安装步骤

1. **克隆仓库**
```bash
git clone https://github.com/aresnasa/ai-infra-matrix.git
cd ai-infra-matrix
```

2. **更新Helm依赖**
```bash
helm dependency update helm/ai-infra-matrix
```

3. **部署应用**
```bash
# 使用测试脚本部署
./test-helm-deploy.sh deploy

# 或手动部署
helm install ai-infra-matrix helm/ai-infra-matrix \
  --namespace ai-infra-matrix \
  --create-namespace \
  --wait
```

4. **访问应用**
```bash
# 设置端口转发
kubectl port-forward service/ai-infra-matrix-nginx -n ai-infra-matrix 8080:80

# 在浏览器中访问
open http://localhost:8080
```

## 🔧 配置说明

### 核心配置项

编辑 `helm/ai-infra-matrix/values.yaml` 来自定义部署配置：

```yaml
# 全局配置
global:
  imageRegistry: ""
  imagePullSecrets: []
  storageClass: ""

# PostgreSQL配置
postgresql:
  enabled: true
  auth:
    postgresPassword: "postgres123"
    database: "ai_infra_matrix"

# Redis配置
redis:
  enabled: true
  auth:
    password: "redis123"

# JupyterHub配置
jupyterhub:
  enabled: true
  config:
    singleuser:
      image:
        name: "ai-infra-singleuser"
        tag: "v0.3.8"
```

### 服务端点

部署完成后，以下端点将可用：

| 服务 | 端点 | 描述 |
|------|------|------|
| 主页 | `http://localhost:8080/` | 应用主入口 |
| JupyterHub | `http://localhost:8080/jupyterhub/` | 交互式计算环境 |
| API | `http://localhost:8080/api/` | 后端API |
| Gitea | `http://localhost:8080/gitea/` | Git仓库管理 |
| SeaweedFS | `http://localhost:8080/seaweedfs/` | 对象存储管理 |
| phpLDAPadmin | `http://localhost:8080/ldap/` | LDAP管理界面 |

## 🛠️ 开发和调试

### 测试脚本

项目提供了便捷的测试脚本：

```bash
# 验证Chart语法
./test-helm-deploy.sh validate

# 完整部署
./test-helm-deploy.sh deploy

# 清理环境
./test-helm-deploy.sh clean

# 重新部署
./test-helm-deploy.sh redeploy

# 验证现有部署
./test-helm-deploy.sh verify

# 设置端口转发
./test-helm-deploy.sh port-forward
```

### 日志查看

```bash
# 查看所有Pod状态
kubectl get pods -n ai-infra-matrix

# 查看特定服务日志
kubectl logs -f deployment/ai-infra-matrix-backend -n ai-infra-matrix
kubectl logs -f deployment/ai-infra-matrix-jupyterhub -n ai-infra-matrix

# 查看初始化作业日志
kubectl logs job/ai-infra-matrix-backend-init -n ai-infra-matrix
```

### 调试常见问题

1. **Pod启动失败**
```bash
# 查看Pod详细信息
kubectl describe pod <pod-name> -n ai-infra-matrix

# 查看事件
kubectl get events -n ai-infra-matrix --sort-by='.lastTimestamp'
```

2. **服务连接问题**
```bash
# 测试服务连通性
kubectl exec -it deployment/ai-infra-matrix-backend -n ai-infra-matrix -- curl http://ai-infra-matrix-postgresql:5432

# 检查DNS解析
kubectl exec -it deployment/ai-infra-matrix-backend -n ai-infra-matrix -- nslookup ai-infra-matrix-postgresql
```

3. **配置问题**
```bash
# 查看ConfigMap
kubectl get configmap ai-infra-matrix-config -n ai-infra-matrix -o yaml

# 查看Secrets
kubectl get secret ai-infra-matrix-secrets -n ai-infra-matrix -o yaml
```

## 🔐 安全配置

### 默认密码

> ⚠️ **生产环境请务必修改这些默认密码！**

- PostgreSQL: `postgres123`
- Redis: `redis123`
- LDAP Admin: `admin123`
- JWT Secret: `your-jwt-secret-change-in-production`

### 生产环境配置

1. **修改密码**
```yaml
# values.yaml
postgresql:
  auth:
    postgresPassword: "your-secure-postgres-password"

redis:
  auth:
    password: "your-secure-redis-password"
```

2. **使用Kubernetes Secrets**
```bash
# 创建自定义Secret
kubectl create secret generic ai-infra-secrets \
  --from-literal=postgres-password=your-password \
  --from-literal=redis-password=your-password \
  --from-literal=jwt-secret=your-jwt-secret \
  -n ai-infra-matrix
```

3. **配置TLS**
```yaml
# values.yaml
nginx:
  tls:
    enabled: true
    secretName: ai-infra-tls
```

## 📊 监控和维护

### 资源使用

```bash
# 查看资源使用情况
kubectl top pods -n ai-infra-matrix
kubectl top nodes
```

### 备份

```bash
# 备份PostgreSQL数据
kubectl exec -it deployment/ai-infra-matrix-postgresql -n ai_infra_matrix -- pg_dump -U postgres ai_infra_matrix > backup.sql

# 备份PVC数据
kubectl get pvc -n ai-infra-matrix
```

### 升级

```bash
# 更新Chart
helm upgrade ai-infra-matrix helm/ai-infra-matrix \
  --namespace ai-infra-matrix \
  --reuse-values

# 回滚
helm rollback ai-infra-matrix 1 -n ai-infra-matrix
```

## 🤝 贡献

1. Fork 项目
2. 创建功能分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 打开 Pull Request

## 📄 许可证

此项目使用 Apache License 2.0 许可证 - 查看 [LICENSE](../../LICENSE) 文件了解详情。

## 🆘 支持

如果遇到问题，请：

1. 查看 [故障排除指南](#调试常见问题)
2. 搜索 [已知问题](https://github.com/aresnasa/ai-infra-matrix/issues)
3. 创建新的 [Issue](https://github.com/aresnasa/ai-infra-matrix/issues/new)

---

**Version**: v0.0.4  
**Last Updated**: 2024-12-19
