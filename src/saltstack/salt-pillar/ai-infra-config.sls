# =============================================================================
# AI Infrastructure 通用配置 (模板文件)
# 此文件由 build.sh render 自动生成 ai-infra-config.sls
# 
# 变量说明:
#   192.168.216.66    - 外部访问主机地址
#   8080    - Nginx 主端口 (默认 8080)
#   https  - 协议 (http 或 https)
#   backend     - 后端服务主机 (Docker 内部: backend)
#   8082     - 后端服务端口 (默认 8082)
#   v0.3.8        - 镜像版本标签
# =============================================================================

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
  # 
  # 自动配置说明：
  # - 如果 Minion 与 AI-Infra 在同一网络（能访问 EXTERNAL_HOST），使用外部 URL
  # - 外部 URL 格式: https://192.168.216.66:8080/api/saltstack/node-metrics/callback
  # 
  # 注意：Minion 在外部网络时无法解析 Docker 服务名（如 backend:8082）
  callback_url: "https://192.168.216.66:8080/api/saltstack/node-metrics/callback"
  
  # 采集间隔（分钟）
  collect_interval: 3
  
  # API Token（可选，用于认证）
  api_token: ""
