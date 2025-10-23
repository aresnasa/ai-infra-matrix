# Nightingale Monitoring System Proxy Configuration
# Simple reverse proxy with sub_filter for path rewriting
# Template variables: NIGHTINGALE_HOST, NIGHTINGALE_PORT

# Catch dynamically loaded resources and rewrite to /nightingale/ prefix
# These resources are loaded by JavaScript and bypass sub_filter
location ~ ^/(font|js|image|api/n9e)/ {
    rewrite ^/(.*)$ /nightingale/$1 last;
}

# Main Nightingale location
# Use ^~ to stop regex matching (prevents static file location from intercepting)
location ^~ /nightingale/ {
    # Proxy to Nightingale backend (with trailing slash to strip /nightingale prefix)
    proxy_pass http://{{NIGHTINGALE_HOST}}:{{NIGHTINGALE_PORT}}/;
    
    # ProxyAuth disabled to enable normal login/logout functionality
    # If you need SSO integration, uncomment and configure properly:
    # proxy_set_header X-User-Name $http_x_user_name;
    
    # Standard proxy headers
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    
    # Disable compression for sub_filter to work
    proxy_set_header Accept-Encoding "";
    
    # Rewrite all absolute paths to include /nightingale prefix
    # This is needed because Nightingale uses absolute paths like /assets/
    sub_filter_types text/html application/javascript text/css application/json;
    sub_filter_once off;
    sub_filter '="/' '="/nightingale/';
    sub_filter "='/" "='/nightingale/";
    
    # Hide iframe blocking headers
    proxy_hide_header X-Frame-Options;
    proxy_hide_header Content-Security-Policy;
    
    # WebSocket support
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection $connection_upgrade;
    
    # Enable buffering for sub_filter
    proxy_buffering on;
    proxy_buffer_size 128k;
    proxy_buffers 4 256k;
    
    # Timeouts
    proxy_connect_timeout 60s;
    proxy_send_timeout 60s;
    proxy_read_timeout 60s;
}

