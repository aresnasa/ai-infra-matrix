# AI Infrastructure Matrix - Nginx Server Configuration Template (TLS/HTTPS)
# Generated from docker-compose.yml configuration
# 此配置用于启用 HTTPS 模式

# Frontend Upstream
upstream frontend {
    server {{FRONTEND_HOST}}:{{FRONTEND_PORT}};
}

# Backend Upstream
upstream backend {
    server {{BACKEND_HOST}}:{{BACKEND_PORT}};
}

# JupyterHub Upstream
upstream jupyterhub {
    server {{JUPYTERHUB_HOST}}:{{JUPYTERHUB_PORT}};
}

# SaltStack API Upstream - 负载均衡双 Master
upstream salt_api {
    least_conn;
    server salt-master-1:8002 max_fails=2 fail_timeout=10s;
    server salt-master-2:8002 max_fails=2 fail_timeout=10s backup;
}

# SeaweedFS Upstream Definitions
upstream seaweedfs_master {
    server seaweedfs-master:9333;
}

upstream seaweedfs_volume {
    server seaweedfs-volume:8080;
}

upstream seaweedfs_filer {
    server seaweedfs-filer:8888;
}

upstream seaweedfs_s3 {
    server seaweedfs-filer:8333;
}

upstream nightingale_console {
    server nightingale:17000;
}

# HTTP 服务器 - 重定向到 HTTPS
server {
    listen 80;
    server_name _;
    
    # 设置重定向目标端口 (从环境变量获取，默认 8443)
    set $https_port "{{HTTPS_PORT}}";
    
    # 健康检查保持 HTTP 可访问
    location /health {
        access_log off;
        return 200 "healthy\n";
        add_header Content-Type text/plain;
    }
    
    # 其他所有请求重定向到 HTTPS
    # 使用 $host 提取主机名（不含端口），然后添加 HTTPS 端口
    location / {
        # 如果 HTTPS 端口是 443，则不需要在 URL 中显示端口
        if ($https_port = "443") {
            return 301 https://$host$request_uri;
        }
        return 301 https://$host:$https_port$request_uri;
    }
}

# HTTPS 服务器 - 主服务
server {
    listen 443 ssl http2;
    server_name _;
    
    # SSL 证书配置 (使用通配符匹配证书文件)
    ssl_certificate /etc/nginx/ssl/server.crt;
    ssl_certificate_key /etc/nginx/ssl/server.key;
    
    # SSL 协议和加密套件 (现代安全配置)
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:DHE-RSA-AES128-GCM-SHA256:DHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;
    
    # SSL 会话配置
    ssl_session_timeout 1d;
    ssl_session_cache shared:SSL:50m;
    ssl_session_tickets off;
    
    # 禁用绝对重定向，使用相对路径（保留端口号）
    absolute_redirect off;
    port_in_redirect on;
    
    client_max_body_size 100M;
    
    # 安全头
    add_header X-Content-Type-Options nosniff always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;
    add_header Strict-Transport-Security "max-age=63072000; includeSubDomains" always;
    
    # 日志格式
    access_log /var/log/nginx/access.log main;
    error_log /var/log/nginx/error.log warn;

    # 定义基础变量
    set $external_scheme "https";
    set $external_host "{{EXTERNAL_HOST}}";

    # 认证验证端点（内部认证检查时调用后端）
    location = /__auth/verify {
        internal;
        proxy_pass http://backend/api/auth/verify;
        proxy_pass_request_body off;
        proxy_set_header Content-Length "";
        proxy_set_header X-Original-URI $request_uri;
        proxy_set_header X-Original-Remote-Addr $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $external_scheme;
        proxy_set_header X-Forwarded-Host $external_host;
        proxy_set_header X-External-Host $external_host;
        proxy_set_header Authorization $http_authorization;
        proxy_set_header Cookie $http_cookie;
        proxy_hide_header Access-Control-Allow-Origin;
        proxy_hide_header Access-Control-Allow-Methods;
        proxy_hide_header Access-Control-Allow-Headers;
        proxy_hide_header Access-Control-Allow-Credentials;
    }

    # /jupyter 路径由前端React路由处理（iframe页面），启用SPA fallback
    location = /jupyter {
        add_header Cache-Control "no-store, no-cache, must-revalidate" always;
        expires -1;
        try_files $uri $uri/ /index.html;
    }

    # /monitoring 路径由前端React路由处理（iframe页面），启用SPA fallback
    location = /monitoring {
        add_header Cache-Control "no-store, no-cache, must-revalidate" always;
        expires -1;
        try_files $uri $uri/ /index.html;
    }

    # 按模块拆分：各服务路由在独立文件中，便于单独调试
    include /etc/nginx/conf.d/includes/gitea.conf;
    include /etc/nginx/conf.d/includes/jupyterhub.conf;
    include /etc/nginx/conf.d/includes/nightingale.conf;
    include /etc/nginx/conf.d/includes/seaweedfs.conf;

    # Nightingale API 代理 - 使用 ^~ 确保优先于 /api/ 匹配
    location ^~ /api/n9e/ {
        # SSO Integration: Extract username from JWT token via auth_request
        auth_request /__auth/verify;
        auth_request_set $auth_username $upstream_http_x_user;
        
        # 不需要 rewrite，直接代理到 Nightingale，保持完整路径
        proxy_pass http://nightingale_console;
        
        # Pass the authenticated username to Nightingale for ProxyAuth
        proxy_set_header X-User-Name $auth_username;
        
        proxy_set_header Host $http_host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
        
        # WebSocket support
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
    }

    # SaltStack API 代理 - 负载均衡双 Master
    location /salt-api/ {
        # 认证验证
        auth_request /__auth/verify;
        auth_request_set $auth_username $upstream_http_x_user;
        
        # 重写路径并代理到 Salt API upstream
        rewrite ^/salt-api/(.*)$ /$1 break;
        proxy_pass http://salt_api;
        
        proxy_set_header Host $http_host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
        proxy_set_header X-Auth-Token $http_x_auth_token;
        
        # Salt API 需要更长的超时时间
        proxy_connect_timeout 60s;
        proxy_send_timeout 300s;
        proxy_read_timeout 300s;
    }

    # 后端 API 代理 + CORS
    location /api/ {
        proxy_pass http://backend/api/;
        proxy_set_header Host $http_host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
        proxy_set_header X-Forwarded-Host $external_host;
        proxy_set_header X-External-Host $external_host;
        proxy_set_header Authorization $http_authorization;
        proxy_set_header Cookie $http_cookie;
        
        # 增大代理缓冲区以处理大的响应头（如JWT token）
        proxy_buffer_size 128k;
        proxy_buffers 4 256k;
        proxy_busy_buffers_size 256k;
        
        set $cors_origin "*";
        if ($http_origin ~ ^https?://(.*\.)?(localhost|[\d\.]+)(:\d+)?$) { set $cors_origin $http_origin; }
        add_header Access-Control-Allow-Origin $cors_origin always;
        add_header Access-Control-Allow-Methods "GET, POST, PUT, DELETE, OPTIONS" always;
        add_header Access-Control-Allow-Headers "Content-Type, Authorization, X-Requested-With, X-External-Host" always;
        add_header Access-Control-Allow-Credentials "true" always;
        if ($request_method = OPTIONS) {
            add_header Access-Control-Allow-Origin $cors_origin always;
            add_header Access-Control-Allow-Methods "GET, POST, PUT, DELETE, OPTIONS" always;
            add_header Access-Control-Allow-Headers "Content-Type, Authorization, X-Requested-With, X-External-Host" always;
            add_header Access-Control-Allow-Credentials "true" always;
            return 204;
        }
    }

    # 静态调试与测试页
    location = /test_auth.html {
        root /usr/share/nginx/html;
        add_header Content-Type "text/html; charset=utf-8";
        add_header Cache-Control "no-cache, no-store, must-revalidate";
    }

    location = /debug_auth.html {
        root /usr/share/nginx/html;
        try_files /debug_auth.html /debug_auth.html =404;
        add_header Content-Type "text/html; charset=utf-8";
        add_header Cache-Control "no-cache, no-store, must-revalidate";
    }

    location = /token_setup.html {
        root /usr/share/nginx/html;
        add_header Content-Type text/html;
        add_header Cache-Control "no-cache, no-store, must-revalidate";
    }

    # Nightingale static assets - must come before general static file location
    # These paths are used by Nightingale monitoring system
    location ~ ^/(font|js|image)/ {
        proxy_pass http://nightingale_console;
        proxy_set_header Host $http_host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
        expires 1d;
        add_header Cache-Control "public";
    }

    # 前端静态资源与入口
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
        proxy_pass http://frontend;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }

    location / {
        proxy_pass http://frontend/;
        proxy_set_header Host $http_host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
    }

    # 健康检查
    location /health {
        access_log off;
        return 200 "healthy\n";
        add_header Content-Type text/plain;
    }

    # 轻量级响应头调试端点
    location = /__headers_check/seaweedfs-console {
        default_type text/plain;
        add_header X-Frame-Options "SAMEORIGIN" always;
        add_header Content-Security-Policy "frame-ancestors 'self' $external_scheme://$external_host" always;
        return 200 "ok\n";
    }
}
