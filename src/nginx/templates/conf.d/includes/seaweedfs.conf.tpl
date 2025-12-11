# SeaweedFS Object Storage Configuration
# This file handles SeaweedFS Filer, S3 API and Master routing

# SeaweedFS Filer 静态资源路由 (CSS/JS 等)
location ^~ /seaweedfsstatic/ {
    proxy_hide_header X-Frame-Options;
    proxy_hide_header Content-Security-Policy;
    
    proxy_pass http://seaweedfs_filer/seaweedfsstatic/;
    proxy_set_header Host $http_host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    
    # 静态资源缓存
    proxy_cache_valid 200 1d;
    expires 1d;
    add_header Cache-Control "public, immutable";
}

# SeaweedFS S3 API 路由 (用于 S3 兼容访问)
location ^~ /seaweedfs-s3/ {
    # S3 API 直通代理
    proxy_hide_header X-Frame-Options;
    proxy_hide_header Content-Security-Policy;
    
    proxy_pass http://seaweedfs_s3/;
    proxy_set_header Host $http_host;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $external_scheme;
    proxy_set_header X-Forwarded-Host $external_host;
    proxy_set_header Accept-Encoding "";
    
    # S3 API 特定配置
    proxy_set_header X-Forwarded-Server $host;
    proxy_connect_timeout 300s;
    proxy_send_timeout 300s;
    proxy_read_timeout 300s;
    proxy_buffering off;
    client_max_body_size 0;
    
    # CORS 支持
    add_header Access-Control-Allow-Origin * always;
    add_header Access-Control-Allow-Methods "GET, POST, PUT, DELETE, HEAD, OPTIONS" always;
    add_header Access-Control-Allow-Headers "Authorization, Origin, X-Requested-With, Content-Type, Accept, X-Amz-Date, X-Amz-Content-Sha256, X-Amz-User-Agent" always;
    add_header Access-Control-Expose-Headers "ETag, x-amz-request-id" always;
    
    if ($request_method = OPTIONS) {
        add_header Access-Control-Allow-Origin * always;
        add_header Access-Control-Allow-Methods "GET, POST, PUT, DELETE, HEAD, OPTIONS" always;
        add_header Access-Control-Allow-Headers "Authorization, Origin, X-Requested-With, Content-Type, Accept, X-Amz-Date, X-Amz-Content-Sha256, X-Amz-User-Agent" always;
        add_header Access-Control-Max-Age 3600 always;
        return 204;
    }
}

# SeaweedFS Filer Web界面路由 (用于文件浏览和管理)
location ^~ /seaweedfs-filer/ {
    # 允许在iframe中嵌入
    proxy_hide_header X-Frame-Options;
    proxy_hide_header Content-Security-Policy;
    proxy_hide_header X-Content-Security-Policy;

    # 将上游返回的根路径重写为带前缀
    proxy_redirect ~^/(.*)$ /seaweedfs-filer/$1;
    proxy_redirect http://seaweedfs-filer:8888/ /seaweedfs-filer/;
    proxy_redirect https://seaweedfs-filer:8888/ /seaweedfs-filer/;

    # 补全尾随斜杠
    rewrite ^/seaweedfs-filer$ /seaweedfs-filer/ permanent;

    # 传递前缀信息
    proxy_set_header X-Forwarded-Prefix /seaweedfs-filer;

    # 允许同源页面内嵌
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header Content-Security-Policy "frame-ancestors 'self' $external_scheme://$external_host" always;

    proxy_pass http://seaweedfs_filer/;
    proxy_set_header Host $http_host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $external_scheme;
    proxy_set_header X-Forwarded-Host $external_host;
    
    # WebSocket 升级支持
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection $connection_upgrade;
    
    # 缓冲与超时配置
    proxy_buffering off;
    proxy_request_buffering off;
    proxy_connect_timeout 300s;
    proxy_send_timeout 300s;
    proxy_read_timeout 300s;
    
    # 大文件上传支持
    client_max_body_size 0;
    
    # 安全头配置
    add_header X-Content-Type-Options nosniff always;
    add_header X-XSS-Protection "1; mode=block" always;
    
    # CORS 配置
    add_header Access-Control-Allow-Origin "$external_scheme://$external_host" always;
    add_header Access-Control-Allow-Methods "GET, POST, PUT, DELETE, OPTIONS" always;
    add_header Access-Control-Allow-Headers "Content-Type, Authorization, X-Requested-With" always;
    add_header Access-Control-Allow-Credentials "true" always;
    
    if ($request_method = OPTIONS) {
        add_header Access-Control-Allow-Origin "$external_scheme://$external_host" always;
        add_header Access-Control-Allow-Methods "GET, POST, PUT, DELETE, OPTIONS" always;
        add_header Access-Control-Allow-Headers "Content-Type, Authorization, X-Requested-With" always;
        add_header Access-Control-Allow-Credentials "true" always;
        add_header Access-Control-Max-Age 3600 always;
        return 204;
    }
}

# SeaweedFS Master 管理界面路由 (用于集群状态监控)
location ^~ /seaweedfs-master/ {
    proxy_hide_header X-Frame-Options;
    proxy_hide_header Content-Security-Policy;
    proxy_hide_header X-Content-Security-Policy;

    proxy_redirect ~^/(.*)$ /seaweedfs-master/$1;
    proxy_redirect http://seaweedfs-master:9333/ /seaweedfs-master/;
    proxy_redirect https://seaweedfs-master:9333/ /seaweedfs-master/;

    rewrite ^/seaweedfs-master$ /seaweedfs-master/ permanent;

    proxy_set_header X-Forwarded-Prefix /seaweedfs-master;

    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header Content-Security-Policy "frame-ancestors 'self' $external_scheme://$external_host" always;

    proxy_pass http://seaweedfs_master/;
    proxy_set_header Host $http_host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $external_scheme;
    proxy_set_header X-Forwarded-Host $external_host;
    
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection $connection_upgrade;
    
    proxy_buffering off;
    proxy_connect_timeout 60s;
    proxy_send_timeout 60s;
    proxy_read_timeout 60s;
    
    add_header X-Content-Type-Options nosniff always;
    add_header X-XSS-Protection "1; mode=block" always;
}

# SeaweedFS Volume 状态接口路由 (用于监控)
location ^~ /seaweedfs-volume/ {
    proxy_hide_header X-Frame-Options;
    proxy_hide_header Content-Security-Policy;
    
    proxy_pass http://seaweedfs_volume/;
    proxy_set_header Host $http_host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $external_scheme;
    proxy_set_header X-Forwarded-Host $external_host;
    
    proxy_buffering off;
    proxy_connect_timeout 60s;
    proxy_send_timeout 300s;
    proxy_read_timeout 300s;
    
    # 大文件支持
    client_max_body_size 0;
}
