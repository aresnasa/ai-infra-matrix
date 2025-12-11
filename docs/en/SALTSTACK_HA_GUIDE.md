# SaltStack Multi-Master High Availability Guide

**[中文文档](../zh_CN/SALTSTACK_HA_GUIDE.md)** | **English**

## Overview

AI-Infra-Matrix supports SaltStack multi-master high availability architecture, providing failover capability by deploying multiple Salt Master nodes.

## Architecture Design

```text
                    ┌─────────────────────────────────────┐
                    │          AI-Infra Backend           │
                    │      (SaltMasterPool Management)    │
                    └─────────────┬───────────────────────┘
                                  │
                    ┌─────────────▼───────────────┐
                    │    Salt API Load Balancer    │
                    │  (Backend Pool Auto Failover)│
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
            │   Shared PKI Keys │
            │  (salt_keys Vol)  │
            └─────────┬─────────┘
                      │
            ┌─────────▼─────────┐
            │   Salt Minions    │
            │ (External Nodes)  │
            └───────────────────┘
```

## Key Features

1. **Shared PKI Keys**: All Masters use the same PKI key pair
2. **Auto Failover**: Backend SaltMasterPool automatically selects healthy Master
3. **Primary/Secondary Roles**: Primary generates keys, Secondary waits for keys to be ready
4. **Independent Cache**: Each Master has independent cache directory to avoid conflicts

## Deployment Modes

### Single Master Mode (Default)

```bash
# Start with default configuration
docker compose up -d salt-master-1
```

### High Availability Mode (Multi-Master)

```bash
# Start multi-Master with --profile ha
docker compose --profile ha up -d
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SALT_MASTER_URLS` | `http://salt-master-1:8002,http://salt-master-2:8002` | Multi-Master URL list (comma separated) |
| `SALT_MASTER_HOST` | `salt-master-1` | Primary Master container name |
| `SALT_START_LOCAL_MINION` | `false` | Whether to start local Minion |

### docker-compose.yml Key Configuration

```yaml
# salt-master-1 (Primary)
salt-master-1:
  environment:
    - SALT_MASTER_ID=salt-master-1
    - SALT_MASTER_ROLE=primary    # Primary generates PKI
  volumes:
    - salt_keys:/etc/salt/pki     # Shared PKI
    - salt_states:/srv/salt       # Shared States
    - salt_pillar:/srv/pillar     # Shared Pillar

# salt-master-2 (Secondary)
salt-master-2:
  environment:
    - SALT_MASTER_ID=salt-master-2
    - SALT_MASTER_ROLE=secondary  # Secondary waits for PKI
  depends_on:
    salt-master-1:
      condition: service_healthy
  profiles:
    - ha  # Only start in HA mode
```

## Backend Pool Configuration

Backend uses `SaltMasterPool` to manage multiple Master connections:

```go
// Load from environment variable
// Method 1: JSON configuration
SALT_MASTERS_CONFIG='[{"url":"http://salt-master-1:8002","priority":0},{"url":"http://salt-master-2:8002","priority":1}]'

// Method 2: URL list
SALT_MASTER_URLS="http://salt-master-1:8002,http://salt-master-2:8002"

// Method 3: Single Master (compatible mode)
SALT_MASTER_HOST="salt-master-1"
SALT_API_PORT="8002"
```

## Minion Configuration

For multi-Master architecture, Minion needs to configure multiple Master addresses:

```yaml
# /etc/salt/minion.d/masters.conf
master:
  - <PRIMARY_MASTER_IP>:4505
  - <SECONDARY_MASTER_IP>:4507

# Or use domain name (recommended)
master:
  - salt-master.your-domain.com

master_type: failover
master_alive_interval: 30
```

## Failover Recovery

### Scenario 1: Primary Node Failure

1. Backend `SaltMasterPool` detects salt-master-1 is unhealthy
2. Automatically switches to salt-master-2 for API requests
3. Minion connections will try secondary Master

### Scenario 2: Primary Node Recovery

1. salt-master-1 restarts
2. Uses PKI keys from shared volume
3. Backend health check recovers, prioritizes salt-master-1

### Manual Failover

```bash
# Stop primary
docker compose stop salt-master-1

# Verify secondary is healthy
docker compose logs salt-master-2

# Recover primary
docker compose start salt-master-1
```

## Verify Deployment

### Check Master Status

```bash
# Check health status of both Masters
curl http://localhost:8082/api/saltstack/status

# Response should include multi-Master info
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

### Check PKI Consistency

```bash
# Both Masters' public key fingerprints should be identical
docker exec ai-infra-salt-master-1 md5sum /etc/salt/pki/master/master.pub
docker exec ai-infra-salt-master-2 md5sum /etc/salt/pki/master/master.pub
```

## Notes

1. **PKI Consistency**: Ensure all Masters use the same PKI keys
2. **States Sync**: salt_states and salt_pillar volumes are shared by all Masters
3. **Cache Isolation**: Each Master has independent cache directory
4. **Port Planning**:
   - salt-master-1: 4505, 4506 (external Minion main entry)
   - salt-master-2: 4507, 4508 (backup entry)

## Further Reading

- [SaltStack Official Multi-Master Documentation](https://docs.saltproject.io/en/latest/topics/tutorials/multimaster.html)
- [AI-Infra-Matrix Architecture Documentation](./ARCHITECTURE.md)
