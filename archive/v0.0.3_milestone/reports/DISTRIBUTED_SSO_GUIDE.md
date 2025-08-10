# 分布式部署配置指南
# AI基础设施矩阵 - 分布式SSO单点登录支持

## 🌐 分布式部署架构

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  外部负载均衡器   │    │   Nginx代理节点   │    │   用户浏览器     │
│  (可选)         │────│  (反向代理)      │────│                │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                              ├─── Backend节点群 (API/认证)
                              ├─── JupyterHub节点群 (Notebook)
                              └─── Frontend节点群 (Web UI)
```

## 🔧 分布式部署配置

### 1. 环境变量配置

创建 `.env.distributed` 文件：

```bash
# ========================================
# 分布式部署环境变量配置
# ========================================

# 外部访问配置
EXTERNAL_HOST=your-domain.com
EXTERNAL_PORT=443
EXTERNAL_SCHEME=https

# 后端节点配置
BACKEND_HOST=backend-node-1.internal
BACKEND_PORT=8082
BACKEND_NODES=backend-node-1.internal:8082,backend-node-2.internal:8082

# JupyterHub节点配置  
JUPYTERHUB_HOST=jupyterhub-node-1.internal
JUPYTERHUB_PORT=8000
JUPYTERHUB_NODES=jupyterhub-node-1.internal:8000,jupyterhub-node-2.internal:8000

# 前端节点配置
FRONTEND_HOST=frontend-node-1.internal
FRONTEND_PORT=80
FRONTEND_NODES=frontend-node-1.internal:80,frontend-node-2.internal:80

# SSL证书配置 (分布式HTTPS)
SSL_CERT_PATH=/etc/ssl/certs/domain.crt
SSL_KEY_PATH=/etc/ssl/private/domain.key

# JWT配置 (分布式共享密钥)
JWT_SECRET=your-shared-secret-across-all-nodes
JUPYTERHUB_CRYPT_KEY=your-shared-crypt-key-32-chars

# 数据库配置 (分布式共享)
DB_HOST=postgres-cluster.internal
DB_PORT=5432
REDIS_HOST=redis-cluster.internal
REDIS_PORT=6379

# LDAP配置 (分布式共享)
LDAP_SERVER=ldap-cluster.internal
LDAP_PORT=389
```

### 2. Nginx分布式配置模板

`nginx.distributed.conf`:

```nginx
# 分布式upstream配置
upstream backend_cluster {
    # 后端节点群
    server backend-node-1.internal:8082 weight=3 max_fails=3 fail_timeout=30s;
    server backend-node-2.internal:8082 weight=3 max_fails=3 fail_timeout=30s;
    server backend-node-3.internal:8082 weight=2 backup;
    
    # 负载均衡策略
    least_conn;
    keepalive 32;
}

upstream jupyterhub_cluster {
    # JupyterHub节点群
    server jupyterhub-node-1.internal:8000 weight=3 max_fails=2 fail_timeout=30s;
    server jupyterhub-node-2.internal:8000 weight=3 max_fails=2 fail_timeout=30s;
    
    # 会话保持 (基于IP hash)
    ip_hash;
    keepalive 16;
}

upstream frontend_cluster {
    # 前端节点群
    server frontend-node-1.internal:80 weight=3 max_fails=3 fail_timeout=30s;
    server frontend-node-2.internal:80 weight=3 max_fails=3 fail_timeout=30s;
    
    least_conn;
    keepalive 32;
}

# 分布式主机映射
map $http_host $external_host {
    default $http_host;
    # 内部访问映射到外部主机
    "~^nginx-node-.*\.internal" "your-domain.com";
    "~^10\..*" "your-domain.com";
    "~^192\.168\..*" "your-domain.com";
}

map $http_x_forwarded_proto $external_scheme {
    default $scheme;
    https https;
    http http;
}

server {
    listen 80;
    listen 443 ssl http2;
    server_name your-domain.com *.your-domain.com;
    
    # SSL证书配置
    ssl_certificate /etc/ssl/certs/domain.crt;
    ssl_certificate_key /etc/ssl/private/domain.key;
    
    # 分布式部署优化
    absolute_redirect off;
    
    # 安全头设置
    add_header X-Frame-Options SAMEORIGIN always;
    add_header X-Content-Type-Options nosniff always;
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    
    # API代理到后端集群
    location /api/ {
        proxy_pass http://backend_cluster/api/;
        
        # 分布式代理头
        proxy_set_header Host $http_host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $external_scheme;
        proxy_set_header X-Forwarded-Host $external_host;
        proxy_set_header X-External-Host $external_host;
        
        # SSO认证支持
        proxy_set_header Authorization $http_authorization;
        proxy_set_header Cookie $http_cookie;
        
        # 分布式CORS
        add_header Access-Control-Allow-Origin "https://$external_host" always;
        add_header Access-Control-Allow-Credentials "true" always;
    }
    
    # JupyterHub代理到JupyterHub集群
    location /jupyter/ {
        proxy_pass http://jupyterhub_cluster/jupyter/;
        
        # 分布式代理头
        proxy_set_header Host $http_host;
        proxy_set_header X-Forwarded-Proto $external_scheme;
        proxy_set_header X-Forwarded-Host $external_host;
        proxy_set_header X-External-Host $external_host;
        
        # SSO认证支持
        proxy_set_header Authorization $http_authorization;
        proxy_set_header Cookie $http_cookie;
        
        # WebSocket支持
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        
        # 会话保持
        proxy_set_header X-Forwarded-Prefix /jupyter;
    }
    
    # 前端代理到前端集群
    location / {
        proxy_pass http://frontend_cluster/;
        
        proxy_set_header Host $http_host;
        proxy_set_header X-Forwarded-Proto $external_scheme;
        proxy_set_header X-Forwarded-Host $external_host;
    }
}
```

### 3. Docker Compose分布式配置

`docker-compose.distributed.yml`:

```yaml
version: '3.8'

services:
  # Nginx代理节点
  nginx-proxy:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.distributed.conf:/etc/nginx/nginx.conf
      - ./ssl:/etc/ssl
    environment:
      - BACKEND_NODES=${BACKEND_NODES}
      - JUPYTERHUB_NODES=${JUPYTERHUB_NODES}
      - FRONTEND_NODES=${FRONTEND_NODES}
    networks:
      - distributed-network
    deploy:
      replicas: 2
      placement:
        constraints: [node.role == manager]

networks:
  distributed-network:
    driver: overlay
    attachable: true
```

### 4. Kubernetes分布式配置

`k8s/nginx-proxy.yaml`:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-proxy
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx-proxy
  template:
    metadata:
      labels:
        app: nginx-proxy
    spec:
      containers:
      - name: nginx
        image: nginx:alpine
        ports:
        - containerPort: 80
        - containerPort: 443
        volumeMounts:
        - name: nginx-config
          mountPath: /etc/nginx/nginx.conf
          subPath: nginx.conf
        env:
        - name: EXTERNAL_HOST
          valueFrom:
            configMapKeyRef:
              name: distributed-config
              key: external-host
      volumes:
      - name: nginx-config
        configMap:
          name: nginx-distributed-config

---
apiVersion: v1
kind: Service
metadata:
  name: nginx-proxy-service
spec:
  type: LoadBalancer
  ports:
  - port: 80
    targetPort: 80
  - port: 443
    targetPort: 443
  selector:
    app: nginx-proxy
```

## 🔐 分布式SSO配置要点

### 1. 统一认证密钥
```bash
# 所有节点必须使用相同的JWT密钥
JWT_SECRET=same-secret-across-all-nodes

# JupyterHub集群必须使用相同的Crypt Key
JUPYTERHUB_CRYPT_KEY=same-32-char-key-across-jupyterhub-nodes
```

### 2. 共享会话存储
```bash
# 使用Redis集群共享会话
REDIS_CLUSTER=redis-node-1:6379,redis-node-2:6379,redis-node-3:6379

# 使用数据库集群共享用户数据
DB_CLUSTER=postgres-primary:5432,postgres-replica:5432
```

### 3. 外部主机配置
```bash
# 确保所有服务知道外部访问地址
EXTERNAL_BASE_URL=https://your-domain.com
JUPYTERHUB_PUBLIC_HOST=your-domain.com
FRONTEND_PUBLIC_URL=https://your-domain.com
```

## 🚀 分布式部署步骤

### 1. 准备基础设施
```bash
# 创建分布式网络
docker network create --driver overlay distributed-ai-infra

# 部署共享服务 (数据库、Redis、LDAP)
docker stack deploy -c shared-services.yml shared
```

### 2. 部署应用节点
```bash
# 部署后端节点群
docker stack deploy -c backend-cluster.yml backend

# 部署JupyterHub节点群  
docker stack deploy -c jupyterhub-cluster.yml jupyterhub

# 部署前端节点群
docker stack deploy -c frontend-cluster.yml frontend
```

### 3. 部署Nginx代理
```bash
# 部署Nginx代理节点
docker stack deploy -c nginx-proxy.yml proxy
```

### 4. 验证分布式SSO
```bash
# 测试外部访问
curl -H "Host: your-domain.com" https://your-domain.com/api/health

# 测试SSO流程
python test_distributed_sso.py
```

## 🔍 分布式故障排除

### 1. 网络连通性检查
```bash
# 检查节点间连通性
docker exec nginx-container ping backend-node-1.internal
docker exec nginx-container ping jupyterhub-node-1.internal
```

### 2. 负载均衡状态检查
```bash
# 检查upstream状态
curl http://nginx-node/nginx_status
```

### 3. SSO跨节点验证
```bash
# 验证JWT在所有节点有效
curl -H "Authorization: Bearer $TOKEN" http://backend-node-1/api/auth/verify
curl -H "Authorization: Bearer $TOKEN" http://backend-node-2/api/auth/verify
```

## 📊 分布式监控建议

### 1. 健康检查
- Nginx: `/health`
- Backend: `/api/health`  
- JupyterHub: `/jupyter/hub/api`

### 2. 指标监控
- 请求延迟
- 节点负载
- SSO成功率
- 跨节点会话一致性

### 3. 日志聚合
- 集中化日志收集
- SSO事件跟踪
- 错误关联分析
