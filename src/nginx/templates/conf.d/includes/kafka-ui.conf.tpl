# Kafka UI Proxy Configuration
# Kafka UI is served via iframe embedded in the main application
# Template variables: KAFKA_UI_HOST (default: kafka-ui), KAFKA_UI_PORT (default: 8080)

# Kafka UI proxy location
location ^~ /kafka-ui-backend {
    # Authentication check - require valid JWT token
    auth_request /__auth/verify;
    auth_request_set $auth_username $upstream_http_x_user;
    
    # Proxy to Kafka UI (no rewrite needed since Kafka UI is configured with context path)
    proxy_pass http://{{KAFKA_UI_HOST}}:{{KAFKA_UI_PORT}};
    
    # Standard proxy headers
    proxy_set_header Host $http_host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto https;
    
    # WebSocket support (Kafka UI may use WebSockets)
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection $connection_upgrade;
    
    # Timeout settings
    proxy_connect_timeout 60s;
    proxy_send_timeout 120s;
    proxy_read_timeout 120s;
    
    # Disable buffering for real-time updates
    proxy_buffering off;
    
    # Allow embedding in iframe (remove X-Frame-Options from Kafka UI response)
    proxy_hide_header X-Frame-Options;
    proxy_hide_header Content-Security-Policy;
}
