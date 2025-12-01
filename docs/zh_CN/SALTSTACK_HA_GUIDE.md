# SaltStack 多 Master 高可用指南

## 概述

AI-Infra-Matrix 支持 SaltStack 多 Master 高可用架构，通过部署多个 Salt Master 节点提供故障转移能力。

## 架构设计

```text
                    ┌─────────────────────────────────────┐
                    │          AI-Infra Backend           │
                    │      (SaltMasterPool 管理)          │
                    └─────────────┬───────────────────────┘
                                  │
                    ┌─────────────▼───────────────┐
                    │    Salt API Load Balancer    │
                    │  (后端 Pool 自动故障转移)    │
                    └─────────────┬───────────────┘
                                  │
            ┌─────────────────────┼─────────────────────┐
            │                     │                     │
    ┌───────▼───────┐     ┌───────▼───────┐     ┌───────▼───────┐
    │ salt-master-1 │     │ salt-master-2 │     │ salt-master-N │
    │   (Primary)   │     │  (Secondary)  │     │  (Secondary)  │
    │  Port 4505/6  │     │  Port 4507/8  │     │     ...       │
    └───────┬───────┘     └───────┬───────┘     └───────────────┘
            │                     │
            └─────────┬───────────┘
                      │
            ┌─────────▼─────────┐
            │   共享 PKI 密钥   │
            │  (salt_keys Vol)  │
            └─────────┬─────────┘
                      │
            ┌─────────▼─────────┐
            │   Salt Minions    │
            │ (外部节点连接)    │
            └───────────────────┘
```

## 关键特性

1. **共享 PKI 密钥**: 所有 Master 使用相同的 PKI 密钥对
2. **自动故障转移**: 后端 SaltMasterPool 自动选择健康的 Master
3. **主从角色**: Primary 负责生成密钥，Secondary 等待密钥就绪
4. **独立缓存**: 每个 Master 有独立的缓存目录，避免冲突

## 部署模式

### 单 Master 模式 (默认)

```bash
# 使用默认配置启动
docker compose up -d salt-master-1
```

### 高可用模式 (多 Master)

```bash
# 使用 --profile ha 启动多 Master
docker compose --profile ha up -d
```

## 配置说明

### 环境变量

| 变量名 | 默认值 | 说明 |
|--------|--------|------|
| `SALT_MASTER_URLS` | `http://salt-master-1:8002,http://salt-master-2:8002` | 多 Master URL 列表 (逗号分隔) |
| `SALT_MASTER_HOST` | `salt-master-1` | 主 Master 容器名 |
| `SALT_START_LOCAL_MINION` | `false` | 是否启动本地 Minion |

### docker-compose.yml 关键配置

```yaml
# salt-master-1 (主节点)
salt-master-1:
  environment:
    - SALT_MASTER_ID=salt-master-1
    - SALT_MASTER_ROLE=primary    # 主节点负责生成 PKI
  volumes:
    - salt_keys:/etc/salt/pki     # 共享 PKI
    - salt_states:/srv/salt       # 共享 States
    - salt_pillar:/srv/pillar     # 共享 Pillar

# salt-master-2 (备用节点)
salt-master-2:
  environment:
    - SALT_MASTER_ID=salt-master-2
    - SALT_MASTER_ROLE=secondary  # 备用节点等待 PKI
  depends_on:
    salt-master-1:
      condition: service_healthy
  profiles:
    - ha  # 仅在 HA 模式下启动
```

## 后端 Pool 配置

后端使用 `SaltMasterPool` 管理多个 Master 连接：

```go
// 从环境变量加载
// 方式1: JSON 配置
SALT_MASTERS_CONFIG='[{"url":"http://salt-master-1:8002","priority":0},{"url":"http://salt-master-2:8002","priority":1}]'

// 方式2: URL 列表
SALT_MASTER_URLS="http://salt-master-1:8002,http://salt-master-2:8002"

// 方式3: 单 Master (兼容模式)
SALT_MASTER_HOST="salt-master-1"
SALT_API_PORT="8002"
```

## Minion 配置

对于多 Master 架构，Minion 需要配置多个 Master 地址：

```yaml
# /etc/salt/minion.d/masters.conf
master:
  - <PRIMARY_MASTER_IP>:4505
  - <SECONDARY_MASTER_IP>:4507

# 或者使用域名 (推荐)
master:
  - salt-master.your-domain.com

master_type: failover
master_alive_interval: 30
```

## 故障恢复

### 场景1: 主节点故障

1. 后端 `SaltMasterPool` 自动检测到 salt-master-1 不健康
2. 自动切换到 salt-master-2 处理 API 请求
3. Minion 连接会尝试备用 Master

### 场景2: 主节点恢复

1. salt-master-1 重新启动
2. 使用共享 volume 中的 PKI 密钥
3. 后端健康检查恢复后优先使用 salt-master-1

### 手动故障转移

```bash
# 停止主节点
docker compose stop salt-master-1

# 确认备用节点健康
docker compose logs salt-master-2

# 恢复主节点
docker compose start salt-master-1
```

## 验证部署

### 检查 Master 状态

```bash
# 检查两个 Master 的健康状态
curl http://localhost:8082/api/saltstack/status

# 响应应包含多 Master 信息
{
  "status": "online",
  "master_count": 2,
  "healthy_count": 2,
  "masters": [
    {"url": "http://salt-master-1:8002", "healthy": true},
    {"url": "http://salt-master-2:8002", "healthy": true}
  ]
}
```

### 检查 PKI 一致性

```bash
# 两个 Master 的公钥指纹应该相同
docker exec ai-infra-salt-master-1 md5sum /etc/salt/pki/master/master.pub
docker exec ai-infra-salt-master-2 md5sum /etc/salt/pki/master/master.pub
```

## 注意事项

1. **PKI 一致性**: 确保所有 Master 使用相同的 PKI 密钥
2. **States 同步**: salt_states 和 salt_pillar volume 被所有 Master 共享
3. **缓存隔离**: 每个 Master 有独立的缓存目录
4. **端口规划**:
   - salt-master-1: 4505, 4506 (外部 Minion 主入口)
   - salt-master-2: 4507, 4508 (备用入口)

## 扩展阅读

- [SaltStack 官方多 Master 文档](https://docs.saltproject.io/en/latest/topics/tutorials/multimaster.html)
- [AI-Infra-Matrix 架构文档](./ARCHITECTURE.md)
