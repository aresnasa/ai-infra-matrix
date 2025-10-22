# Nightingale Monitoring System Proxy Configuration
# ProxyAuth SSO integration with X-User-Name header
# Template variables: NIGHTINGALE_HOST, NIGHTINGALE_PORT

# Main Nightingale location - accessible from frontend at /nightingale/
location /nightingale/ {
    # Authentication required
    if ($has_authz = "0") {
        return 401 "Authentication required";
    }

    # Get user info from backend via auth_request
    auth_request /internal/nightingale-auth;
    auth_request_set $auth_username $upstream_http_x_user_name;
    auth_request_set $auth_email $upstream_http_x_user_email;
    
    # Remove /nightingale prefix and proxy to Nightingale
    rewrite ^/nightingale/(.*) /$1 break;
    
    proxy_pass http://{{NIGHTINGALE_HOST}}:{{NIGHTINGALE_PORT}};
    
    # ProxyAuth headers - inject username from authenticated session
    proxy_set_header X-User-Name $auth_username;
    
    # Standard proxy headers
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    
    # WebSocket support
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection $connection_upgrade;
    
    # Timeouts
    proxy_connect_timeout 60s;
    proxy_send_timeout 60s;
    proxy_read_timeout 60s;
    
    # Buffer settings
    proxy_buffering off;
    proxy_request_buffering off;
}

# Auth subrequest endpoint - get user info from backend and return as headers
location = /internal/nightingale-auth {
    internal;
    proxy_pass http://backend:8080/api/auth/me;
    proxy_pass_request_body off;
    proxy_set_header Content-Length "";
    proxy_set_header Authorization $authz_final;
    proxy_set_header X-Original-URI $request_uri;
    
    # Backend /api/auth/me endpoint returns user info with these headers:
    # X-User-Name, X-User-Email, X-User-ID
}
