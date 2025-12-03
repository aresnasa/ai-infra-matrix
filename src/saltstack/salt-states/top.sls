# SaltStack 基础状态文件
base:
  '*':
    - common
    - ai-infra-monitoring
    - node-metrics
  
  'ai-infra-*':
    - ai-infra-agent
    - docker-setup
  
  'web-*':
    - nginx-config
    - ssl-certs
  
  'db-*':
    - database-setup
    - backup-scripts
