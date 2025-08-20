# Pillar Top文件
base:
  '*':
    - common
    - ai-infra-config
  
  'ai-infra-*':
    - ai-infra-secrets
  
  'production':
    - production-config
