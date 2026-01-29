# ArgoCD GitOps 反向代理配置
# 用于 GitOps 持续部署
# Template variables: ARGOCD_HOST (default: argocd-server), ARGOCD_PORT (default: 8080)
# 注意：使用变量方式进行 DNS 解析，避免 ArgoCD 服务未启动时 nginx 无法启动

# ArgoCD 服务不可用时的友好错误页面
location @argocd_unavailable {
    default_type text/html;
    return 503 '<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>ArgoCD - 服务未就绪</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif; background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); min-height: 100vh; display: flex; align-items: center; justify-content: center; }
        .container { background: white; border-radius: 16px; box-shadow: 0 20px 60px rgba(0,0,0,0.3); padding: 48px; max-width: 500px; text-align: center; }
        .icon { font-size: 64px; margin-bottom: 24px; }
        h1 { color: #333; font-size: 24px; margin-bottom: 16px; }
        p { color: #666; line-height: 1.6; margin-bottom: 24px; }
        .hint { background: #f8f9fa; border-radius: 8px; padding: 16px; font-family: monospace; font-size: 14px; color: #495057; text-align: left; margin-bottom: 24px; }
        .btn { display: inline-block; background: #667eea; color: white; padding: 12px 24px; border-radius: 8px; text-decoration: none; font-weight: 500; transition: background 0.3s; }
        .btn:hover { background: #5a6fd6; }
        .status { margin-top: 24px; padding-top: 24px; border-top: 1px solid #eee; }
        .status-item { display: flex; align-items: center; justify-content: space-between; padding: 8px 0; }
        .status-dot { width: 12px; height: 12px; border-radius: 50%; }
        .status-dot.offline { background: #dc3545; }
    </style>
</head>
<body>
    <div class="container">
        <div class="icon">🚀</div>
        <h1>ArgoCD 服务未启动</h1>
        <p>ArgoCD GitOps 服务当前未运行。请先启动 ArgoCD 服务后再访问此页面。</p>
        <div class="hint">
            <strong>启动命令：</strong><br>
            docker-compose --profile argocd up -d
        </div>
        <a href="/" class="btn">返回首页</a>
        <div class="status">
            <div class="status-item">
                <span>ArgoCD 服务状态</span>
                <span class="status-dot offline"></span>
            </div>
        </div>
    </div>
</body>
</html>';
}

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
    
    # 服务不可用时显示友好页面
    proxy_intercept_errors on;
    error_page 502 503 504 = @argocd_unavailable;
}

# ArgoCD API 不可用时的响应 (JSON 格式)
location @argocd_api_unavailable {
    default_type application/json;
    return 503 '{"error":"ArgoCD service unavailable","message":"ArgoCD 服务未启动","enabled":false,"ready":false,"action_hint":"请先启动 ArgoCD 服务：docker-compose --profile argocd up -d"}';
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
    
    # 服务不可用时返回 JSON 错误
    proxy_intercept_errors on;
    error_page 502 503 504 = @argocd_api_unavailable;
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
    
    # 服务不可用时重定向到首页
    proxy_intercept_errors on;
    error_page 502 503 504 = @argocd_unavailable;
}

# ArgoCD 健康检查
location /argocd/healthz {
    # 使用变量延迟 DNS 解析
    set $argocd_upstream "{{ARGOCD_HOST}}:{{ARGOCD_PORT}}";
    proxy_pass http://$argocd_upstream/argocd/healthz;
    proxy_set_header Host $http_host;
    access_log off;
    
    # 服务不可用时返回 JSON
    proxy_intercept_errors on;
    error_page 502 503 504 = @argocd_api_unavailable;
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
    
    # 服务不可用时显示友好页面
    proxy_intercept_errors on;
    error_page 502 503 504 = @argocd_unavailable;
}
