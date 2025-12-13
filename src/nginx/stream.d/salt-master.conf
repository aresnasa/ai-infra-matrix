# SaltStack Master TCP Load Balancing Configuration
# =================================================
# 实现 Salt Master 双主高可用负载均衡
# - 4505 端口: Salt Publisher (ZeroMQ PUB)
# - 4506 端口: Salt Request Server (ZeroMQ REP)
#
# 使用轮询负载均衡，带健康检查和故障转移

# Salt Publisher 负载均衡 (端口 4505)
upstream salt_publisher {
    # 主节点 - 默认优先
    server salt-master-1:4505 weight=5 max_fails=2 fail_timeout=30s;
    # 备用节点 - 故障转移
    server salt-master-2:4505 weight=3 max_fails=2 fail_timeout=30s backup;
}

# Salt Request Server 负载均衡 (端口 4506)
upstream salt_reqserver {
    # 主节点 - 默认优先
    server salt-master-1:4506 weight=5 max_fails=2 fail_timeout=30s;
    # 备用节点 - 故障转移
    server salt-master-2:4506 weight=3 max_fails=2 fail_timeout=30s backup;
}

# Salt API HTTP 负载均衡 (端口 8002)
upstream salt_api_stream {
    # 轮询负载均衡
    least_conn;
    server salt-master-1:8002 max_fails=2 fail_timeout=10s;
    server salt-master-2:8002 max_fails=2 fail_timeout=10s;
}

# Salt Publisher 代理服务器 (4505)
server {
    listen 4505;
    proxy_pass salt_publisher;
    proxy_timeout 300s;
    proxy_connect_timeout 5s;
}

# Salt Request Server 代理服务器 (4506)
server {
    listen 4506;
    proxy_pass salt_reqserver;
    proxy_timeout 300s;
    proxy_connect_timeout 5s;
}

# Salt API HTTP 代理服务器 (8002)
server {
    listen 8002;
    proxy_pass salt_api_stream;
    proxy_timeout 60s;
    proxy_connect_timeout 5s;
}
