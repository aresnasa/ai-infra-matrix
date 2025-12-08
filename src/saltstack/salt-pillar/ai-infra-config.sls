# AI Infrastructure 通用配置
ai_infra:
  version: "v0.3.8"
  environment: "production"
  
  # 服务配置
  services:
    backend:
      port: 8082
      db_connection: "postgresql://user:pass@postgres:5432/ai_infra"
    
    frontend:
      port: 3000
      api_url: "/api"
    
    nginx:
      port: 80
      ssl_port: 443
    
    jupyterhub:
      port: 8000
    
    saltstack:
      master_port: 4505
      ret_port: 4506
      api_port: 8000

  # 网络配置
  network:
    name: "ai-infra-network"
    subnet: "172.20.0.0/16"

  # 存储配置
  storage:
    postgres_data: "/var/lib/postgresql/data"
    redis_data: "/data"
    uploads: "/app/uploads"

# Node Metrics 采集配置
# 用于配置节点指标采集脚本的回调地址和采集间隔
node_metrics:
  # 回调 URL - Minion 会将采集到的指标发送到这个地址
  # 如果 Minion 与 Master 在同一网络，使用 Docker 服务名
  # 如果 Minion 在外部网络，需要配置为可访问的外部 URL
  callback_url: "http://ai-infra-backend:8082/api/saltstack/node-metrics/callback"
  # 采集间隔（分钟）
  collect_interval: 3
  # API Token（可选，用于认证）
  api_token: ""
