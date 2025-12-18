# Helm High Availability Deployment Guide

本指南介绍如何使用 Helm 部署高可用版本的 AI Infra Matrix 到 Kubernetes 集群。

## 架构概述

高可用部署包含以下关键组件：

### 数据库层 - PostgreSQL HA (Patroni)
- **Patroni** 提供 PostgreSQL 的自动故障转移和高可用
- 使用 Kubernetes 作为分布式配置存储 (DCS)
- 默认部署 3 节点集群
- 自动主从切换，Leader 选举

### 缓存层 - Redis Cluster
- **Redis Cluster** 提供数据分片和高可用
- 默认 3 主节点 + 每主节点 1 副本 (共 6 节点)
- 自动数据分片，支持水平扩展
- 内置故障转移能力

### 消息队列 - Kafka with KRaft
- **Kafka KRaft 模式** 无需 Zookeeper 依赖
- 默认 3 broker 集群
- 支持高吞吐量消息处理
- 可选 Kafka UI 进行可视化管理

### 应用层弹性扩展
- **Backend**: 支持 HPA 自动扩缩容 (1-10 副本)
- **Frontend**: 支持 HPA 自动扩缩容 (1-5 副本)
- **Nginx**: 支持 HPA 自动扩缩容 (1-5 副本)

## 快速开始

### 前置条件

1. Kubernetes 集群 (推荐 1.24+)
2. Helm 3.x
3. kubectl 已配置
4. 存储类 (StorageClass) 支持动态卷供应
5. 足够的集群资源（建议至少 8 CPU, 16GB RAM）

### 开发环境部署

使用默认 values.yaml 进行单节点部署（非 HA）：

```bash
# 创建命名空间
kubectl create namespace ai-infra

# 更新依赖
cd helm/ai-infra-matrix
helm dependency update

# 安装 Chart
helm install ai-infra . -n ai-infra

# 查看部署状态
kubectl get pods -n ai-infra -w
```

### 生产环境部署 (高可用)

使用 values-prod.yaml 启用所有 HA 组件：

```bash
# 创建命名空间
kubectl create namespace ai-infra-prod

# 更新依赖
cd helm/ai-infra-matrix
helm dependency update

# 编辑 values-prod.yaml，修改必要的密码和配置
vim values-prod.yaml

# 安装 Chart（启用 HA）
helm install ai-infra . -f values-prod.yaml -n ai-infra-prod

# 查看部署状态
kubectl get pods -n ai-infra-prod -w
```

## 组件详细配置

### Patroni (PostgreSQL HA)

```yaml
patroni:
  enabled: true
  replicaCount: 3
  postgresql:
    database: "ai_infra_db"
    username: "postgres"
    password: "your-strong-password"
  persistence:
    enabled: true
    size: "20Gi"
```

**验证 Patroni 集群状态：**

```bash
# 查看 Patroni 集群状态
kubectl exec -it ai-infra-patroni-0 -n ai-infra-prod -- patronictl list

# 预期输出类似：
# + Cluster: ai-infra-patroni -------+---------+---------+----+-----------+
# | Member              | Host       | Role    | State   | TL | Lag in MB |
# +---------------------+------------+---------+---------+----+-----------+
# | ai-infra-patroni-0  | 10.x.x.x   | Leader  | running |  1 |           |
# | ai-infra-patroni-1  | 10.x.x.x   | Replica | running |  1 |         0 |
# | ai-infra-patroni-2  | 10.x.x.x   | Replica | running |  1 |         0 |
# +---------------------+------------+---------+---------+----+-----------+
```

### Redis Cluster

```yaml
redisCluster:
  enabled: true
  masterCount: 3
  replicasPerMaster: 1
  password: "your-redis-password"
  persistence:
    enabled: true
    size: "5Gi"
```

**验证 Redis Cluster 状态：**

```bash
# 查看集群信息
kubectl exec -it ai-infra-redis-cluster-0 -n ai-infra-prod -- redis-cli -a your-redis-password cluster info

# 查看集群节点
kubectl exec -it ai-infra-redis-cluster-0 -n ai-infra-prod -- redis-cli -a your-redis-password cluster nodes
```

### Kafka with KRaft

```yaml
kafka:
  enabled: true
  replicaCount: 3
  config:
    numPartitions: 3
    defaultReplicationFactor: 3
  ui:
    enabled: true
```

**验证 Kafka 状态：**

```bash
# 查看 broker 状态
kubectl exec -it ai-infra-kafka-0 -n ai-infra-prod -- kafka-broker-api-versions.sh --bootstrap-server localhost:9092

# 列出 topics
kubectl exec -it ai-infra-kafka-0 -n ai-infra-prod -- kafka-topics.sh --bootstrap-server localhost:9092 --list
```

### HPA (水平 Pod 自动扩缩容)

```yaml
backend:
  autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 10
    targetCPUUtilizationPercentage: 70
    targetMemoryUtilizationPercentage: 80
```

**查看 HPA 状态：**

```bash
kubectl get hpa -n ai-infra-prod
kubectl describe hpa ai-infra-backend-hpa -n ai-infra-prod
```

## 扩容操作

### 扩展 PostgreSQL (Patroni)

```bash
# 增加副本数
helm upgrade ai-infra . -f values-prod.yaml \
  --set patroni.replicaCount=5 \
  -n ai-infra-prod
```

### 扩展 Redis Cluster

```bash
# 增加 master 节点数
helm upgrade ai-infra . -f values-prod.yaml \
  --set redisCluster.masterCount=6 \
  -n ai-infra-prod

# 注意：扩展后需要重新平衡 slots
kubectl exec -it ai-infra-redis-cluster-0 -n ai-infra-prod -- redis-cli --cluster rebalance localhost:6379 -a your-password
```

### 扩展 Kafka

```bash
# 增加 broker 数量
helm upgrade ai-infra . -f values-prod.yaml \
  --set kafka.replicaCount=5 \
  -n ai-infra-prod
```

## 故障转移测试

### 测试 PostgreSQL 故障转移

```bash
# 删除当前 leader pod
kubectl delete pod ai-infra-patroni-0 -n ai-infra-prod

# 观察自动故障转移
kubectl exec -it ai-infra-patroni-1 -n ai-infra-prod -- patronictl list
```

### 测试 Redis 故障转移

```bash
# 删除一个 master 节点
kubectl delete pod ai-infra-redis-cluster-0 -n ai-infra-prod

# 检查集群状态
kubectl exec -it ai-infra-redis-cluster-1 -n ai-infra-prod -- redis-cli -a your-password cluster nodes
```

## 监控

### Prometheus Metrics

各组件暴露以下 metrics 端点：

- Patroni: `:8008/metrics`
- Redis: `:9121/metrics` (需要 Redis Exporter)
- Kafka: `:9308/metrics` (JMX Exporter)
- Backend: `:8082/metrics`

### Grafana Dashboards

推荐的 Grafana Dashboard IDs：
- PostgreSQL: `9628`
- Redis Cluster: `11835`
- Kafka: `7589`
- Kubernetes HPA: `17125`

## 备份与恢复

### PostgreSQL 备份

```bash
# 使用 pg_dump 备份
kubectl exec -it ai-infra-patroni-0 -n ai-infra-prod -- \
  pg_dump -U postgres ai_infra_db > backup.sql

# 或使用 pg_basebackup 进行物理备份
kubectl exec -it ai-infra-patroni-0 -n ai-infra-prod -- \
  pg_basebackup -D /tmp/backup -Fp -Xs -P
```

### Redis 备份

```bash
# 触发 RDB 快照
kubectl exec -it ai-infra-redis-cluster-0 -n ai-infra-prod -- \
  redis-cli -a your-password BGSAVE
```

## 常见问题

### Q: Patroni 无法选举 Leader？
A: 检查 RBAC 权限和 Kubernetes API 连通性：
```bash
kubectl auth can-i get endpoints --as=system:serviceaccount:ai-infra-prod:ai-infra-patroni
```

### Q: Redis Cluster 初始化失败？
A: 确保所有节点已启动，然后手动初始化：
```bash
kubectl exec -it ai-infra-redis-cluster-init-xxx -n ai-infra-prod -- cat /tmp/init.log
```

### Q: HPA 不工作？
A: 确保 Metrics Server 已安装：
```bash
kubectl top pods -n ai-infra-prod
```

## 升级

```bash
# 升级 Helm release
helm upgrade ai-infra . -f values-prod.yaml -n ai-infra-prod

# 查看升级历史
helm history ai-infra -n ai-infra-prod

# 回滚到上一版本
helm rollback ai-infra 1 -n ai-infra-prod
```

## 卸载

```bash
# 卸载 release (保留 PVC)
helm uninstall ai-infra -n ai-infra-prod

# 清理 PVC (警告：会删除所有数据)
kubectl delete pvc -l app.kubernetes.io/instance=ai-infra -n ai-infra-prod
```
