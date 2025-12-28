    # JupyterHub 认证后的入口与桥接页面
    location = /jupyterhub-authenticated {
        return 302 /jupyterhub-auth-bridge?target_url=/jupyter/hub/;
    }

    location = /jupyterhub-auth-bridge {
        root /usr/share/nginx/html/jupyterhub;
        try_files /jupyterhub_auth_bridge.html =404;
        add_header Content-Type "text/html; charset=utf-8";
        add_header Cache-Control "no-cache, no-store, must-revalidate";
    }

    # JupyterHub 前端入口交给 React 路由处理
    # 注释掉 location = /jupyter 让前端路由处理 /jupyter 页面（iframe展示）

    location ^~ /jupyter/ {
        proxy_pass http://jupyterhub;
        proxy_set_header Host $http_host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Cookie $http_cookie;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_buffering off;
        proxy_request_buffering off;
        proxy_read_timeout 86400s;
        proxy_send_timeout 86400s;
        proxy_hide_header Content-Security-Policy;
        proxy_hide_header X-Frame-Options;
        # CSP frame-ancestors 支持 HTTP 和 HTTPS
        # 注意: EXTERNAL_HOST 是纯主机名/IP，端口通过 EXTERNAL_PORT/HTTPS_PORT 指定
        add_header Content-Security-Policy "frame-ancestors 'self' http://localhost:{{EXTERNAL_PORT}} http://0.0.0.0:{{EXTERNAL_PORT}} http://{{EXTERNAL_HOST}}:{{EXTERNAL_PORT}} https://localhost:{{HTTPS_PORT}} https://0.0.0.0:{{HTTPS_PORT}} https://{{EXTERNAL_HOST}}:{{HTTPS_PORT}};" always;
        add_header X-Frame-Options SAMEORIGIN always;
    }

    # Favicon 透传
    location = /favicon.ico {
        proxy_pass http://jupyterhub/favicon.ico;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
