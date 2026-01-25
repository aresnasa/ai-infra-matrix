# Kafka UI Proxy Configuration
# Kafka UI is served via iframe embedded in the main application
# Template variables: KAFKA_UI_HOST (default: kafka-ui), KAFKA_UI_PORT (default: 8080)

# Kafka UI proxy location
location ^~ /kafka-ui-backend {
    # Note: Authentication is handled by the parent page (already logged in)
    # The iframe inherits the session cookies from the parent page
    
    # Proxy to Kafka UI (Kafka UI is configured with SERVER_SERVLET_CONTEXT_PATH=/kafka-ui-backend)
    proxy_pass http://{{KAFKA_UI_HOST}}:{{KAFKA_UI_PORT}};
    
    # Standard proxy headers
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
    proxy_set_header X-Forwarded-Host $host;
    proxy_set_header X-Forwarded-Port $server_port;
    
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
    
    # Add headers to allow iframe embedding
    add_header X-Frame-Options "SAMEORIGIN" always;
}
