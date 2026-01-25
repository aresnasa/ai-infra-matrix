# ArgoCD GitOps 部署指南

## 概述

ArgoCD 是 AI Infrastructure Matrix 的 GitOps 持续部署工具，提供：
- **声明式部署**: 使用 Git 仓库作为唯一真相来源
- **自动同步**: 自动检测并同步配置变更
- **可视化管理**: 直观的应用部署状态可视化
- **回滚能力**: 快速回滚到任意历史版本

## 架构图

```
┌─────────────────────────────────────────────────────────────────┐
│                        开发者                                    │
│                    (Git Push)                                    │
└─────────────────────────┬───────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Gitea (Git 仓库)                            │
│                    manifests/k8s/*.yaml                          │
└─────────────────────────┬───────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│                        ArgoCD                                    │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────────────────┐ │
│  │ ArgoCD Server│ │ Repo Server  │ │ Application Controller   │ │
│  │   (API/UI)   │ │  (Git 同步)   │ │    (状态协调)             │ │
│  └──────────────┘ └──────────────┘ └──────────────────────────┘ │
└─────────────────────────┬───────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Kubernetes 集群                               │
│              (Deployments, Services, ConfigMaps)                 │
└─────────────────────────────────────────────────────────────────┘
```

## 配置步骤

### 1. 启用 ArgoCD 服务

在 `.env` 文件中设置：

```bash
# 启用 ArgoCD
ARGOCD_ENABLED=true
ARGOCD_VERSION=v2.13.3
ARGOCD_HTTP_PORT=8282

# Gitea 仓库配置
ARGOCD_REPO_URL=http://gitea:3000/org/manifests.git
ARGOCD_REPO_USERNAME=argocd
ARGOCD_REPO_PASSWORD=argocd-git-token

# Keycloak SSO 集成
KEYCLOAK_ARGOCD_CLIENT_SECRET=argocd-secret-change-me
```

### 2. 启动服务

```bash
# 使用 argocd profile 启动
./build.sh up --profile argocd

# 或者使用 full profile
./build.sh up --profile full
```

### 3. 初始密码

首次启动时，ArgoCD 会生成管理员密码。获取密码：

```bash
# 查看初始密码
docker-compose exec argocd-server argocd admin initial-password
```

### 4. 配置仓库

在 ArgoCD 中添加 Gitea 仓库：

```bash
# 使用 CLI
argocd repo add http://gitea:3000/org/manifests.git \
  --username argocd \
  --password your-git-token
```

或通过 Web UI:
1. 访问 ArgoCD UI: `http://your-host/argocd`
2. 导航到 `Settings` > `Repositories`
3. 点击 `Connect Repo`
4. 填写仓库信息

## 创建应用

### 通过 Web UI

1. 点击 `New App`
2. 填写应用信息：
   - **Application Name**: `my-app`
   - **Project**: `default`
   - **Sync Policy**: `Automatic` (自动同步)
   - **Repository URL**: `http://gitea:3000/org/manifests.git`
   - **Path**: `./k8s/my-app`
   - **Cluster URL**: `https://kubernetes.default.svc`
   - **Namespace**: `my-app`

### 通过 YAML

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: my-app
  namespace: argocd
spec:
  project: default
  source:
    repoURL: http://gitea:3000/org/manifests.git
    targetRevision: HEAD
    path: ./k8s/my-app
  destination:
    server: https://kubernetes.default.svc
    namespace: my-app
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
```

### 通过 API (Backend 集成)

前端可以通过 Backend API 管理 ArgoCD 应用：

```javascript
// 创建应用
await api.post('/argocd/applications', {
  metadata: {
    name: 'my-app',
    namespace: 'argocd'
  },
  spec: {
    project: 'default',
    source: {
      repoURL: 'http://gitea:3000/org/manifests.git',
      path: './k8s/my-app',
      targetRevision: 'HEAD'
    },
    destination: {
      server: 'https://kubernetes.default.svc',
      namespace: 'my-app'
    }
  }
});

// 同步应用
await api.post('/argocd/applications/my-app/sync');

// 获取应用状态
const app = await api.get('/argocd/applications/my-app');
```

## RBAC 配置

ArgoCD RBAC 与 Keycloak 组映射：

| Keycloak 组 | ArgoCD 权限 |
|------------|------------|
| administrators | admin (所有权限) |
| sre-team | applications:* (应用管理) |
| engineering | applications:get,sync (查看和同步) |
| viewers | applications:get (只读) |

RBAC 策略配置 (`argocd-rbac-cm.yaml`):

```yaml
policy.csv: |
  # 管理员 - 所有权限
  g, administrators, role:admin
  
  # SRE 团队 - 应用和集群管理
  p, role:sre, applications, *, */*, allow
  p, role:sre, clusters, get, *, allow
  p, role:sre, repositories, get, *, allow
  g, sre-team, role:sre
  
  # 开发团队 - 查看和同步
  p, role:developer, applications, get, */*, allow
  p, role:developer, applications, sync, */*, allow
  g, engineering, role:developer
  
  # 只读用户
  p, role:readonly, applications, get, */*, allow
  g, viewers, role:readonly
```

## 同步策略

### 自动同步

启用自动同步后，ArgoCD 会：
1. 每 3 分钟检查 Git 仓库变更
2. 自动应用新的配置
3. 自动修复漂移 (Self-Heal)
4. 自动清理已删除的资源 (Prune)

### 手动同步

对于关键应用，建议使用手动同步：

```bash
# CLI 同步
argocd app sync my-app

# 通过 API
curl -X POST http://argocd-server:8080/api/v1/applications/my-app/sync
```

## 健康状态

ArgoCD 会监控应用健康状态：

| 状态 | 含义 |
|-----|------|
| Healthy | 所有资源运行正常 |
| Progressing | 正在部署或更新中 |
| Degraded | 部分资源不健康 |
| Suspended | 应用已暂停 |
| Missing | 资源缺失 |
| Unknown | 无法确定状态 |

## 与 Keycloak SSO 集成

ArgoCD 通过 Dex 集成 Keycloak 实现 SSO：

1. 用户访问 ArgoCD UI
2. 重定向到 Keycloak 登录
3. Keycloak 验证身份
4. 返回 JWT Token 到 ArgoCD
5. ArgoCD 根据 Token 中的组信息分配权限

## 故障排除

### 问题：应用同步失败

1. 检查 Git 仓库访问权限
2. 验证 YAML 语法
3. 查看同步日志: `argocd app logs my-app`

### 问题：健康检查失败

1. 检查 Pod 状态: `kubectl get pods -n my-app`
2. 查看 Pod 日志: `kubectl logs -n my-app <pod-name>`
3. 检查资源配额

### 问题：SSO 登录失败

1. 验证 Keycloak 配置
2. 检查 Dex 日志
3. 确认客户端密钥正确

## 最佳实践

1. **Git 仓库结构**:
   ```
   manifests/
   ├── base/           # 基础配置
   ├── overlays/       # 环境特定配置
   │   ├── dev/
   │   ├── staging/
   │   └── prod/
   └── apps/           # ArgoCD Application 定义
   ```

2. **分支策略**:
   - `main`: 生产环境
   - `staging`: 预发布环境
   - `develop`: 开发环境

3. **应用分组**:
   - 使用 ArgoCD Projects 组织相关应用
   - 为每个项目配置适当的权限

4. **监控告警**:
   - 配置 Nightingale 监控 ArgoCD 指标
   - 设置同步失败告警

## API 参考

| 端点 | 方法 | 描述 |
|-----|------|------|
| /argocd/applications | GET | 列出所有应用 |
| /argocd/applications | POST | 创建应用 |
| /argocd/applications/:name | GET | 获取应用详情 |
| /argocd/applications/:name | DELETE | 删除应用 |
| /argocd/applications/:name/sync | POST | 同步应用 |
| /argocd/applications/:name/refresh | POST | 刷新应用状态 |
| /argocd/repositories | GET | 列出仓库 |
| /argocd/repositories | POST | 添加仓库 |
| /argocd/clusters | GET | 列出集群 |
| /argocd/projects | GET | 列出项目 |

## 参考资料

- [ArgoCD 官方文档](https://argo-cd.readthedocs.io/)
- [GitOps 原则](https://www.gitops.tech/)
- [Kubernetes 声明式配置](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/declarative-config/)
