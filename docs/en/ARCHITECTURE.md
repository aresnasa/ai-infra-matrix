# System Architecture Design

## Overview

AI Infrastructure Matrix is an enterprise-grade HPC and AI infrastructure platform, utilizing microservices architecture and containerized deployment.

## Overall Architecture

```text
┌─────────────────────────────────────────────────────────────┐
│                      External Access                        │
│              (Client Browsers / API Clients)                │
└────────────────────────┬────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────┐
│                    Reverse Proxy Layer                      │
│                      Nginx :8080                            │
│  ┌──────────┬──────────┬──────────┬──────────┬──────────┐  │
│  │   /      │  /api    │ /jupyter │  /gitea  │   /n9e   │  │
│  └──────────┴──────────┴──────────┴──────────┴──────────┘  │
└─────┬───────┬────────┬─────────┬────────┬────────┬─────────┘
      │       │        │         │        │        │
┌─────▼───────▼────────▼─────────▼────────▼────────▼─────────┐
│                   Application Services                       │
├──────────────┬────────────────┬──────────────┬──────────────┤
│   Frontend   │    Backend     │  JupyterHub  │    Gitea     │
│  (React SPA) │  (Go/FastAPI)  │  (Python)    │  (Go)        │
├──────────────┼────────────────┼──────────────┼──────────────┤
│ Nightingale  │ Slurm Master   │  SaltStack   │   AppHub     │
│ (Monitoring) │  (HPC Sched)   │  (Config)    │  (Packages)  │
└──────┬───────┴────────┬───────┴──────┬───────┴──────┬───────┘
       │                │              │              │
┌──────▼────────────────▼──────────────▼──────────────▼───────┐
│               Data & Storage Services                        │
├─────────────┬──────────────┬──────────────┬─────────────────┤
│ PostgreSQL  │    MySQL     │  OceanBase   │     Redis       │
│ (App Data)  │  (Slurm DB)  │  (Optional)  │  (Cache/MQ)     │
├─────────────┼──────────────┼──────────────┼─────────────────┤
│    Kafka    │  SeaweedFS   │              │                 │
│ (Message Q) │  (Object S3) │              │                 │
└─────────────┴──────────────┴──────────────┴─────────────────┘
```

## Core Components

### Frontend Layer

**Tech Stack**: React 18 + TypeScript + Ant Design

**Responsibilities**:

- User interface rendering
- Route management
- State management (Redux)
- API call encapsulation

**Key Modules**:

```typescript
src/frontend/
├── src/
│   ├── components/      // UI Components
│   ├── pages/          // Pages
│   ├── services/       // API Services
│   ├── store/          // Redux Store
│   ├── utils/          // Utility Functions
│   └── App.tsx         // Main Application
```

### Backend Service

**Tech Stack**: Go 1.21 + Gin/Fiber + GORM

**Responsibilities**:

- RESTful API provision
- Business logic processing
- Database interaction
- Authentication and authorization
- Slurm cluster management
- SaltStack integration
- Kubernetes resource management

**Core Modules**:

```go
src/backend/
├── cmd/
│   └── main.go             // Entry point
├── internal/
│   ├── api/                // API routes
│   ├── services/           // Business logic
│   │   ├── slurm_service.go
│   │   ├── saltstack_service.go
│   │   ├── k8s_service.go
│   │   └── user_service.go
│   ├── models/             // Data models
│   ├── middleware/         // Middleware
│   └── utils/              // Utilities
└── config/                 // Configuration files
```

### JupyterHub

**Tech Stack**: Python 3.11 + JupyterHub 4.0 + DockerSpawner

**Responsibilities**:

- Multi-user Jupyter environment
- User authentication integration
- Resource isolation
- GPU support

**Configuration**:

```python
# jupyterhub_config.py
c.JupyterHub.spawner_class = 'dockerspawner.DockerSpawner'
c.DockerSpawner.image = 'ai-infra-matrix/singleuser:v0.3.8'
c.DockerSpawner.network_name = 'ai-infra-network'
c.Spawner.cpu_limit = 2
c.Spawner.mem_limit = '4G'
```

### Gitea

**Tech Stack**: Go + SQLite/PostgreSQL

**Responsibilities**:

- Git repository hosting
- Pull Request workflow
- Webhook integration
- LFS large file storage (SeaweedFS backend)

**Integration**:

```ini
[server]
ROOT_URL = http://localhost:8080/gitea/

[lfs]
# Note: 'minio' is Gitea's storage type name for S3-compatible storage
STORAGE_TYPE = minio
# Actual backend uses SeaweedFS S3 API
MINIO_ENDPOINT = seaweedfs-filer:8333
MINIO_BUCKET = gitea
```

### Slurm Master

**Tech Stack**: Slurm 23.11 + Ubuntu 22.04

**Responsibilities**:

- Job scheduling
- Resource allocation
- Queue management
- Node management

**Architecture**:

```text
┌─────────────────────────────────────┐
│         Slurm Master                │
├─────────────────────────────────────┤
│  slurmctld  (Controller Daemon)     │
│  slurmdbd   (Database Daemon)       │
│  slurmrestd (REST API Service)      │
└────────┬──────────────┬─────────────┘
         │              │
    ┌────▼────┐    ┌───▼────┐
    │  Node1  │    │ Node2  │
    │ slurmd  │    │ slurmd │
    └─────────┘    └────────┘
```

### SaltStack

**Tech Stack**: Salt 3006 + Python 3

**Responsibilities**:

- Configuration management
- Remote execution
- State management
- Minion deployment

**Architecture**:

```text
┌──────────────────────┐
│    Salt Master       │
│   (Config Center)    │
└──────────┬───────────┘
           │
    ┌──────┼──────┐
    │      │      │
┌───▼──┐ ┌▼────┐ ┌▼────┐
│Minion│ │Minion│ │Minion│
│Node1 │ │Node2 │ │Node3 │
└──────┘ └─────┘ └─────┘
```

### KeyVault Security Service

**Tech Stack**: Go + HMAC-SHA256

**Responsibilities**:

- Secure distribution of Salt Master public key
- One-time token generation and verification
- Secure key storage
- Request signature verification

**Secure Distribution Flow**:

```text
┌─────────────────────────────────────────────────────────────┐
│              Secure Key Distribution Flow                    │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  1. Admin/Backend Service                                   │
│     │                                                       │
│     ├──► Call /api/keyvault/salt/generate-token             │
│     │                                                       │
│     ▼                                                       │
│  2. KeyVault Service                                        │
│     │                                                       │
│     ├──► Generate one-time Token (UUID)                     │
│     ├──► Generate Nonce (random 16-byte hex)                │
│     ├──► Calculate HMAC-SHA256 signature                    │
│     ├──► Store Token in memory (5-min expiry)               │
│     │                                                       │
│     ▼                                                       │
│  3. Return to Admin                                         │
│     │   {token, signature, nonce, expires_at}               │
│     │                                                       │
│     ▼                                                       │
│  4. Batch Installation Script                               │
│     │                                                       │
│     ├──► Carry token + signature + nonce                    │
│     ├──► Call /api/keyvault/salt/master-pub                 │
│     │                                                       │
│     ▼                                                       │
│  5. KeyVault Verification                                   │
│     │                                                       │
│     ├──► Verify Token exists and not expired                │
│     ├──► Verify HMAC signature                              │
│     ├──► Verify Nonce has not been used                     │
│     ├──► Mark Token as used (one-time)                      │
│     │                                                       │
│     ▼                                                       │
│  6. Return Master Public Key                                │
│     │   {pub_key: "base64_encoded_key"}                     │
│     │                                                       │
│     ▼                                                       │
│  7. Minion Installation Script                              │
│     │                                                       │
│     └──► Write key to /etc/salt/pki/minion/minion_master.pub│
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Security Features**:

| Feature | Description |
|---------|-------------|
| One-time Token | Token destroyed immediately after use |
| HMAC Signature | Prevents token tampering |
| Nonce | Prevents replay attacks |
| Expiration | Default 5-minute validity |
| Request Timeout | Default 10-second timeout |

**API Endpoints**:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/keyvault/salt/generate-token` | POST | Generate one-time token (auth required) |
| `/api/keyvault/salt/master-pub` | GET | Get Master public key (token required) |

### AppHub

**Tech Stack**: Ubuntu/RockyLinux + APK/RPM Build Tools

**Responsibilities**:

- Slurm package building (DEB/RPM/APK)
- Categraf monitoring agent packaging
- Multi-architecture support (x86_64/aarch64)
- Version management

**Build Flow**:

```text
┌──────────────┐
│ Source Code  │
└──────┬───────┘
       │
┌──────▼───────┐
│Build Scripts │
│  (bash)      │
└──────┬───────┘
       │
┌──────▼───────┐     ┌──────────────┐
│ DEB Builder  ├────►│  APT Repo    │
└──────────────┘     └──────────────┘
┌──────────────┐     ┌──────────────┐
│ RPM Builder  ├────►│  YUM Repo    │
└──────────────┘     └──────────────┘
┌──────────────┐     ┌──────────────┐
│ APK Builder  ├────►│ Alpine Repo  │
└──────────────┘     └──────────────┘
```

### Nightingale

**Tech Stack**: Go + Vue.js + Prometheus

**Responsibilities**:

- Metrics collection and storage
- Alert rule management
- Dashboard visualization
- Notification channel integration

**Components**:

- n9e-server: Backend service
- n9e-webapi: API service
- Categraf: Metrics collector
- Prometheus: Time series database

## Data Storage

### PostgreSQL

**Usage**: Primary database

**Tables**:

- users: User information
- projects: Project data
- jupyterhub_*: JupyterHub data
- gitea_*: Gitea data

### MySQL

**Usage**: Slurm job database

**Tables**:

- job_table: Job records
- assoc_table: Association table
- cluster_table: Cluster information

### Redis

**Usage**:

- Session caching
- Message queue
- Distributed locks
- Temporary data storage

### SeaweedFS

**Usage**: Object storage

**Buckets**:

- gitea: Gitea LFS data
- jupyter: JupyterHub user files
- backups: Backup files

## Network Architecture

### Docker Compose Network

```yaml
networks:
  ai-infra-network:
    driver: bridge
    ipam:
      config:
        - subnet: 172.20.0.0/16
```

### Inter-service Communication

- Service names as DNS (e.g., `postgres`, `redis`, `backend`)
- Internal port communication (not exposed externally)
- Nginx as unified entry point

### Port Mapping

| Service | Internal Port | External Port | Protocol |
|---------|---------------|---------------|----------|
| Nginx | 80 | 8080 | HTTP |
| Backend | 8000 | - | HTTP |
| JupyterHub | 8000 | - | HTTP |
| Gitea | 3000 | - | HTTP |
| PostgreSQL | 5432 | - | TCP |
| MySQL | 3306 | - | TCP |
| Redis | 6379 | - | TCP |
| SeaweedFS | 8333 | - | HTTP |

## Security Architecture

### Authentication and Authorization

```text
┌──────────────┐
│   Client     │
└──────┬───────┘
       │ 1. Login
┌──────▼───────┐
│   Backend    │
│ (Auth API)   │
└──────┬───────┘
       │ 2. Verify
┌──────▼───────┐
│  PostgreSQL  │
│  (Users DB)  │
└──────┬───────┘
       │ 3. JWT Token
┌──────▼───────┐
│   Client     │
│ (Store Token)│
└──────────────┘
```

### RBAC Permission Model

```text
┌─────────────────────────────────────────────────────────────┐
│                    RBAC Permission Model                     │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  User                                                       │
│    │                                                        │
│    ├──► role_template (Role Templates)                      │
│    │     ├── admin         : System Administrator           │
│    │     ├── sre           : SRE Operations Engineer        │
│    │     ├── data-developer: Data Developer                 │
│    │     ├── model-developer: Model Developer               │
│    │     └── engineer      : Engineering Developer          │
│    │                                                        │
│    └──► roles (Role Associations)                           │
│          │                                                  │
│          └──► permissions (Permissions)                     │
│               ├── resource : Resource type (projects, hosts)│
│               ├── verb     : Action (create, read, update)  │
│               └── scope    : Scope (* or own)               │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

**Role Template Permission Examples**:

| Role Template | Resource Permissions |
|---------------|---------------------|
| `admin` | `*:*:*` (All permissions) |
| `sre` | `saltstack:*:*`, `ansible:*:*`, `kubernetes:*:*`, `hosts:read:*` |
| `data-developer` | `projects:create:*`, `jupyterhub:*:*`, `hosts:read:*` |
| `model-developer` | `jupyterhub:*:own`, `projects:read:*` |
| `engineer` | `kubernetes:*:*`, `projects:*:own` |

### Data Encryption

- Transport layer: HTTPS/TLS
- Storage layer: Database encryption
- Passwords: bcrypt hashing

### Network Isolation

- Services run in isolated network
- Only Nginx exposed externally
- Firewall rules restrictions

## Scalability Design

### Horizontal Scaling

```yaml
# Increase replica count
backend:
  deploy:
    replicas: 3
    
frontend:
  deploy:
    replicas: 2
```

### Load Balancing

```nginx
upstream backend {
    server backend-1:8000;
    server backend-2:8000;
    server backend-3:8000;
}
```

### Database Scaling

- PostgreSQL: Master-slave replication + Read-write separation
- MySQL: InnoDB Cluster
- Redis: Cluster mode
- SeaweedFS: Distributed mode

## Monitoring and Logging

### Monitoring System

```text
┌──────────────────────────────────┐
│       Categraf Agents            │
│  (Collectors on each service)    │
└──────────┬───────────────────────┘
           │ Metrics
┌──────────▼───────────────────────┐
│      Prometheus TSDB             │
│     (Time Series Storage)        │
└──────────┬───────────────────────┘
           │ Query
┌──────────▼───────────────────────┐
│     Nightingale Server           │
│   (Alert Rule Engine)            │
└──────────┬───────────────────────┘
           │ Alert
┌──────────▼───────────────────────┐
│   Notification Channels          │
│  (Webhook/Email/SMS/DingTalk)    │
└──────────────────────────────────┘
```

### Logging System

- Application logs: Written to container stdout
- System logs: syslog
- Audit logs: PostgreSQL
- Log aggregation: ELK/Loki (optional)

## High Availability Design

### Service High Availability

- Multi-replica deployment
- Health checks
- Auto restart
- Rolling updates

### Data High Availability

- Database master-slave replication
- Regular backups
- Snapshot recovery
- Disaster recovery

## Deployment Modes

### Standalone Deployment

```bash
# Docker Compose
docker compose up -d
```

### Cluster Deployment

```bash
# Kubernetes + Helm
helm install ai-infra ./helm/ai-infra-matrix
```

### Hybrid Deployment

- Control plane: Kubernetes
- Compute nodes: Physical/Virtual machines

## Technology Selection Rationale

| Component | Selection | Reason |
|-----------|-----------|--------|
| Frontend Framework | React | Rich ecosystem, component-based, good performance |
| Backend Language | Go | High performance, concurrency-friendly, easy deployment |
| Database | PostgreSQL | Feature-complete, excellent performance, open source |
| Cache | Redis | High performance, rich data structures |
| Object Storage | SeaweedFS | S3 compatible, open source, easy to deploy, high performance |
| Monitoring | Nightingale | Localized, feature-complete, user-friendly |
| Scheduler | Slurm | HPC standard, powerful features |
| Configuration Management | SaltStack | Flexible, powerful, Python ecosystem |

## Related Documentation

- [Project Structure](PROJECT_STRUCTURE.md)
- [Deployment Guide](QUICK_START.md)
- [API Documentation](API_REFERENCE.md)
- [Development Guide](DEVELOPMENT_SETUP.md)
- [Authentication System Design](AUTHENTICATION.md)
- [Salt Key Security Distribution](../docs-all/SALT_KEY_SECURITY.md)
