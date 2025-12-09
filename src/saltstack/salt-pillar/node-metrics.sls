# Node Metrics 采集配置
# 用于配置节点指标采集脚本的回调地址和采集间隔

node_metrics:
  # 回调 URL - Minion 会将采集到的指标发送到这个地址
  # 注意：如果 Minion 在外部网络（不在 Docker 网络内），需要配置为可访问的外部 URL
  # 默认使用环境变量，如果未设置则使用 localhost（需要在部署时覆盖）
  # 生产环境示例: http://192.168.3.101:8080/api/saltstack/node-metrics/callback
  callback_url: "http://{{ salt['grains.get']('master', 'localhost') }}:8080/api/saltstack/node-metrics/callback"
  
  # 采集间隔（分钟）
  collect_interval: 3
  
  # API Token（可选，用于认证）
  # 如果后端配置了 NODE_METRICS_API_TOKEN 环境变量，需要在此处设置相同的值
  api_token: ""
