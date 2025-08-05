#!/bin/bash

# 登录获取token
TOKEN=$(curl -s -X POST http://localhost:8082/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123"}' | \
  jq -r '.token')

echo "Token: $TOKEN"

# 读取kubeconfig内容
KUBECONFIG_CONTENT=$(cat complete-k8s-config.yaml | base64 -w 0)

# 创建k8s集群配置
curl -X POST http://localhost:8082/api/kubernetes/clusters \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d "{
    \"name\": \"Docker Desktop\",
    \"description\": \"Local Docker Desktop Kubernetes cluster\",
    \"api_server\": \"https://kubernetes.docker.internal:6443\",
    \"kube_config\": \"$(cat complete-k8s-config.yaml | tr '\n' ' ' | sed 's/"/\\"/g')\",
    \"namespace\": \"default\"
  }"
