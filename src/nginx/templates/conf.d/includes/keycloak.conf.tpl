# Keycloak IAM 反向代理配置
# 用于 SSO 单点登录和身份认证
# Template variables: KEYCLOAK_HOST (default: keycloak), KEYCLOAK_PORT (default: 8080)
# 注意：使用变量方式进行 DNS 解析，避免 Keycloak 服务未启动时 nginx 无法启动

# Keycloak 认证服务 (/auth)
location /auth {
    # 使用变量延迟 DNS 解析 - 允许服务不存在时 nginx 仍能启动
    set $keycloak_upstream "{{KEYCLOAK_HOST}}:{{KEYCLOAK_PORT}}";
    proxy_pass http://$keycloak_upstream/auth;
    
    proxy_set_header Host $http_host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Forwarded-Host $http_host;
    proxy_set_header X-Forwarded-Port $server_port;
    
    # 支持 WebSocket (用于 Keycloak Admin Console)
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection $connection_upgrade;
    
    # 增大缓冲区以处理大的响应（如 OIDC 配置）
    proxy_buffer_size 128k;
    proxy_buffers 4 256k;
    proxy_busy_buffers_size 256k;
    
    # 超时配置
    proxy_connect_timeout 60s;
    proxy_send_timeout 60s;
    proxy_read_timeout 60s;
    
    # CORS 配置 (用于前端 OIDC 流程)
    set $cors_origin "*";
    if ($http_origin ~ ^https?://(.*\.)?(localhost|[\d\.]+)(:\d+)?$) {
        set $cors_origin $http_origin;
    }
    
    add_header Access-Control-Allow-Origin $cors_origin always;
    add_header Access-Control-Allow-Methods "GET, POST, PUT, DELETE, OPTIONS" always;
    add_header Access-Control-Allow-Headers "Content-Type, Authorization, X-Requested-With" always;
    add_header Access-Control-Allow-Credentials "true" always;
    
    if ($request_method = OPTIONS) {
        add_header Access-Control-Allow-Origin $cors_origin always;
        add_header Access-Control-Allow-Methods "GET, POST, PUT, DELETE, OPTIONS" always;
        add_header Access-Control-Allow-Headers "Content-Type, Authorization, X-Requested-With" always;
        add_header Access-Control-Allow-Credentials "true" always;
        add_header Content-Length 0;
        add_header Content-Type text/plain;
        return 204;
    }
    
    # 允许 iframe 嵌入
    proxy_hide_header X-Frame-Options;
    proxy_hide_header Content-Security-Policy;
    add_header X-Frame-Options "SAMEORIGIN" always;
}

# Keycloak 健康检查端点
location /auth/health {
    # 使用变量延迟 DNS 解析
    set $keycloak_upstream "{{KEYCLOAK_HOST}}:{{KEYCLOAK_PORT}}";
    proxy_pass http://$keycloak_upstream/auth/health;
    proxy_set_header Host $http_host;
    access_log off;
}

# Keycloak OIDC 配置端点 (Well-Known)
location /auth/realms/ {
    # 使用变量延迟 DNS 解析
    set $keycloak_upstream "{{KEYCLOAK_HOST}}:{{KEYCLOAK_PORT}}";
    proxy_pass http://$keycloak_upstream/auth/realms/;
    
    proxy_set_header Host $http_host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    
    # 缓存 OIDC 配置 (减少请求)
    proxy_cache_valid 200 1h;
    add_header Cache-Control "public, max-age=3600";
}
