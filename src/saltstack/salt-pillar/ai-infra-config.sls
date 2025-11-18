# AI Infrastructure 通用配置
ai_infra:
  version: "v0.3.6-dev"
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
