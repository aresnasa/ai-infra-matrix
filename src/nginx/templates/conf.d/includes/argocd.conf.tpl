# ArgoCD GitOps 反向代理配置
# 用于 GitOps 持续部署
# Template variables: ARGOCD_HOST (default: argocd-server), ARGOCD_PORT (default: 8080)
# 注意：使用变量方式进行 DNS 解析，避免 ArgoCD 服务未启动时 nginx 无法启动

# ArgoCD 服务 (/argocd)
location /argocd {
    # 认证验证
    auth_request /__auth/verify;
    auth_request_set $auth_username $upstream_http_x_user;
    
    # 使用变量延迟 DNS 解析 - 允许服务不存在时 nginx 仍能启动
    set $argocd_upstream "{{ARGOCD_HOST}}:{{ARGOCD_PORT}}";
    proxy_pass http://$argocd_upstream/argocd;
    
    proxy_set_header Host $http_host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Forwarded-Host $http_host;
    
    # 传递认证用户信息
    proxy_set_header X-Remote-User $auth_username;
    
    # 支持 WebSocket (用于 ArgoCD 实时更新)
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection $connection_upgrade;
    
    # 增大缓冲区
    proxy_buffer_size 128k;
    proxy_buffers 4 256k;
    proxy_busy_buffers_size 256k;
    
    # 超时配置 (ArgoCD 同步操作可能较慢)
    proxy_connect_timeout 60s;
    proxy_send_timeout 300s;
    proxy_read_timeout 300s;
    
    # 允许 iframe 嵌入
    proxy_hide_header X-Frame-Options;
    proxy_hide_header Content-Security-Policy;
    add_header X-Frame-Options "SAMEORIGIN" always;
}

# ArgoCD API 代理
location /argocd/api/ {
    # 认证验证
    auth_request /__auth/verify;
    auth_request_set $auth_username $upstream_http_x_user;
    
    # 使用变量延迟 DNS 解析
    set $argocd_upstream "{{ARGOCD_HOST}}:{{ARGOCD_PORT}}";
    proxy_pass http://$argocd_upstream/argocd/api/;
    
    proxy_set_header Host $http_host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Remote-User $auth_username;
    
    # WebSocket 支持 (用于 ArgoCD API 流)
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection $connection_upgrade;
    
    # 超时配置
    proxy_connect_timeout 60s;
    proxy_send_timeout 300s;
    proxy_read_timeout 300s;
    
    # CORS 配置
    set $cors_origin "*";
    if ($http_origin ~ ^https?://(.*\.)?(localhost|[\d\.]+)(:\d+)?$) {
        set $cors_origin $http_origin;
    }
    
    add_header Access-Control-Allow-Origin $cors_origin always;
    add_header Access-Control-Allow-Methods "GET, POST, PUT, DELETE, OPTIONS" always;
    add_header Access-Control-Allow-Headers "Content-Type, Authorization, X-Requested-With, Grpc-Metadata-Token" always;
    add_header Access-Control-Allow-Credentials "true" always;
    
    if ($request_method = OPTIONS) {
        add_header Access-Control-Allow-Origin $cors_origin always;
        add_header Access-Control-Allow-Methods "GET, POST, PUT, DELETE, OPTIONS" always;
        add_header Access-Control-Allow-Headers "Content-Type, Authorization, X-Requested-With, Grpc-Metadata-Token" always;
        add_header Access-Control-Allow-Credentials "true" always;
        return 204;
    }
}

# ArgoCD Dex 回调 (用于 SSO)
location /argocd/api/dex/callback {
    # 使用变量延迟 DNS 解析
    set $argocd_upstream "{{ARGOCD_HOST}}:{{ARGOCD_PORT}}";
    proxy_pass http://$argocd_upstream/argocd/api/dex/callback;
    
    proxy_set_header Host $http_host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}

# ArgoCD 健康检查
location /argocd/healthz {
    # 使用变量延迟 DNS 解析
    set $argocd_upstream "{{ARGOCD_HOST}}:{{ARGOCD_PORT}}";
    proxy_pass http://$argocd_upstream/argocd/healthz;
    proxy_set_header Host $http_host;
    access_log off;
}

# ArgoCD 静态资源
location ~ ^/argocd/(assets|dist)/ {
    # 使用变量延迟 DNS 解析
    set $argocd_upstream "{{ARGOCD_HOST}}:{{ARGOCD_PORT}}";
    proxy_pass http://$argocd_upstream;
    
    proxy_set_header Host $http_host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    
    # 缓存静态资源
    expires 1y;
    add_header Cache-Control "public, immutable";
}
