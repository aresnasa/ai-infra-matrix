    # JupyterHub 认证后的入口与桥接页面
    location = /jupyterhub-authenticated {
        return 302 /jupyterhub-auth-bridge?target_url=/jupyter/hub/;
    }

    location = /jupyterhub-auth-bridge {
        root /usr/share/nginx/html/jupyterhub;
        try_files /jupyterhub_auth_bridge.html =404;
        add_header Content-Type "text/html; charset=utf-8";
        add_header Cache-Control "no-cache, no-store, must-revalidate";
        # 允许在 iframe 中嵌入（支持 HTTPS 访问）
        add_header Content-Security-Policy "frame-ancestors 'self' $external_scheme://$external_host http://$http_host https://$http_host;" always;
        add_header X-Frame-Options SAMEORIGIN always;
    }

    # JupyterHub 前端入口交给 React 路由处理
    # 注释掉 location = /jupyter 让前端路由处理 /jupyter 页面（iframe展示）

    location ^~ /jupyter/ {
        proxy_pass http://jupyterhub;
        proxy_set_header Host $http_host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        # 使用 $external_scheme 确保正确传递协议 (http/https)
        proxy_set_header X-Forwarded-Proto $external_scheme;
        proxy_set_header X-Forwarded-Host $http_host;
        proxy_set_header X-Forwarded-Port $server_port;
        proxy_set_header Cookie $http_cookie;
        proxy_http_version 1.1;
        # WebSocket 支持 - 使用全局定义的 $connection_upgrade 变量
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_buffering off;
        proxy_request_buffering off;
        proxy_read_timeout 86400s;
        proxy_send_timeout 86400s;
        # 避免中断 WebSocket/SSE 连接
        proxy_connect_timeout 60s;
        proxy_hide_header Content-Security-Policy;
        proxy_hide_header X-Frame-Options;
        # CSP frame-ancestors 使用动态变量，支持任意访问来源
        # $external_scheme 和 $external_host 在 server 块中定义
        add_header Content-Security-Policy "frame-ancestors 'self' $external_scheme://$external_host http://$http_host https://$http_host;" always;
        add_header X-Frame-Options SAMEORIGIN always;
    }

    # Favicon 透传
    location = /favicon.ico {
        proxy_pass http://jupyterhub/favicon.ico;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $external_scheme;
    }
