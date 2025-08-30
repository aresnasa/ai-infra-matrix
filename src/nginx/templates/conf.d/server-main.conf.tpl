# AI Infrastructure Matrix - Nginx Server Configuration Template
# Generated from docker-compose.yml configuration

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

server {
    listen 80;
    server_name _;
    
    client_max_body_size 100M;
    
    # 安全头（除X-Frame-Options外，其他全局应用）
    add_header X-Content-Type-Options nosniff always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;
    
    # 日志格式
    access_log /var/log/nginx/access.log main;
    error_log /var/log/nginx/error.log warn;

    # 定义基础变量
    set $external_scheme "{{EXTERNAL_SCHEME}}";
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

    # 按模块拆分：Gitea 与 JupyterHub 路由在独立文件中，便于单独调试
    include /etc/nginx/conf.d/includes/gitea.conf;
    include /etc/nginx/conf.d/includes/jupyterhub.conf;

    # 后端 API 代理 + CORS
    location /api/ {
        proxy_pass http://backend/api/;
        proxy_set_header Host $http_host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $external_scheme;
        proxy_set_header X-Forwarded-Host $external_host;
        proxy_set_header X-External-Host $external_host;
        proxy_set_header Authorization $http_authorization;
        proxy_set_header Cookie $http_cookie;
        set $cors_origin "*";
        if ($http_origin ~ ^https?://(.*\\.)?(localhost|[\\d\\.]+)(:\\d+)?$) { set $cors_origin $http_origin; }
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

    # 前端静态资源与入口
    location ~* \\.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
        proxy_pass http://frontend;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }

    location / {
        add_header X-Frame-Options DENY always;
        proxy_pass http://frontend/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # 健康检查
    location /health {
        access_log off;
        return 200 "healthy\\n";
        add_header Content-Type text/plain;
    }
}
