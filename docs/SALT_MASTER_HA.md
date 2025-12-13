# SaltStack 双 Master 高可用配置指南

## 概述

AI Infra Matrix 支持 SaltStack 双 Master 高可用架构，确保 Salt Master 服务的可靠性和故障转移能力。

## 架构说明

```
                    ┌─────────────────────────────────────┐
                    │           Nginx (负载均衡)           │
                    │  - Salt API HTTP LB (port 8002)     │
                    └─────────────────────────────────────┘
                               │
              ┌────────────────┼────────────────┐
              ▼                                 ▼
    ┌─────────────────────┐         ┌─────────────────────┐
    │   salt-master-1     │         │   salt-master-2     │
    │   (Primary Master)  │         │   (Backup Master)   │
    │                     │         │                     │
    │  对外端口:           │         │  对外端口:           │
    │  - 4505 (Publisher) │         │  - 4507 (Publisher) │
    │  - 4506 (ReqServer) │         │  - 4508 (ReqServer) │
    │  - 8002 (API)       │         │  - 8002 (API)       │
    └─────────────────────┘         └─────────────────────┘
              │                                 │
              └────────────┬────────────────────┘
                           │
                    共享存储卷
                    - salt_keys (PKI 密钥)
                    - salt_states (状态文件)
                    - salt_pillar (Pillar 数据)
```

## 配置方式

### 1. Minion 端多 Master 配置 (推荐)

在 Minion 配置文件 `/etc/salt/minion` 中配置多个 Master：

```yaml
# /etc/salt/minion
master:
  - 192.168.1.81:4505    # salt-master-1
  - 192.168.1.81:4507    # salt-master-2 (映射 4507->4505)

# 主 Master 故障时自动切换
master_type: failover

# Master 心跳检测间隔（秒）
master_alive_interval: 30

# 多 Master 随机选择（负载均衡）
random_master: true
```

### 2. 安装 Minion 时指定多 Master

使用 Salt Bootstrap 脚本安装时：

```bash
curl -L https://bootstrap.saltproject.io -o install_salt.sh
sudo sh install_salt.sh -M -A "192.168.1.81,192.168.1.81:4507"
```

或通过我们的自动化脚本：

```bash
# 环境变量方式
export SALT_MASTER_HOSTS="192.168.1.81,192.168.1.81:4507"
./install-minion.sh
```

### 3. HTTP API 负载均衡

对于 Salt API 调用，Nginx 已配置负载均衡：

```
# 访问地址
http://192.168.1.81:8080/salt-api/

# 等效于直接访问
http://salt-master-1:8002/ 或 http://salt-master-2:8002/
```

## 后端 SaltMasterPool 配置

后端服务通过 `SALT_MASTER_URLS` 环境变量配置多 Master 池：

```env
SALT_MASTER_URLS=http://salt-master-1:8002,http://salt-master-2:8002
```

后端会自动：
- 检测 Master 健康状态
- 在 Master 故障时自动切换
- 支持轮询负载均衡

## 故障转移测试

### 测试 Master 1 故障

```bash
# 停止 Master 1
docker stop ai-infra-salt-master-1

# 检查 Minion 是否自动切换到 Master 2
salt-call test.ping

# 恢复 Master 1
docker start ai-infra-salt-master-1
```

### 测试 API 负载均衡

```bash
# 通过 Nginx 访问 Salt API
curl -k http://192.168.1.81:8080/salt-api/

# 检查后端日志看到请求分发到不同 Master
docker logs ai-infra-backend | grep "salt_api"
```

## 密钥同步说明

两个 Master 共享同一套 PKI 密钥：

- `/etc/salt/pki/master/master.pem` - Master 私钥
- `/etc/salt/pki/master/master.pub` - Master 公钥
- `/etc/salt/pki/master/minions/` - 已接受的 Minion 密钥

**重要**: 
- `salt_keys` 卷确保密钥一致性
- `salt-master-1` 作为主节点负责生成密钥
- `salt-master-2` 启动时等待密钥同步完成

## 监控

### 检查 Master 状态

```bash
# 检查 Master 1 状态
docker exec ai-infra-salt-master-1 salt-run manage.status

# 检查 Master 2 状态
docker exec ai-infra-salt-master-2 salt-run manage.status
```

### 通过 Web UI 监控

访问 SLURM/SaltStack 集成页面：
- http://192.168.1.81:8080/slurm

页面显示：
- Master 状态 (connected/disconnected)
- API 状态 (available/unavailable)
- 在线/离线 Minions 数量
