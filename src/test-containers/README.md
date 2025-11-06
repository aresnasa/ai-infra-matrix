# Test Containers for AI Infrastructure Matrix

This folder provides test SSH containers for testing SaltStack and SLURM client installation on different Linux distributions.

## 📦 Available Images

### Ubuntu 22.04 Test Container
- **Dockerfile**: `Dockerfile`
- **Image Name**: `test-ubuntu:${IMAGE_TAG}`
- **Base**: Ubuntu 22.04 LTS
- **Purpose**: Test SaltStack/SLURM installation on Debian/Ubuntu systems

### Rocky Linux 9 Test Container
- **Dockerfile**: `Dockerfile.rocky`
- **Image Name**: `test-rocky:${IMAGE_TAG}`
- **Base**: Rocky Linux 9
- **Purpose**: Test SaltStack/SLURM installation on RHEL-based systems

### SSH Key-based Auth Container (Optional)
- **Dockerfile**: `Dockerfile.ssh-key`
- **Image Name**: `test-ubuntu-ssh-key:${IMAGE_TAG}`
- **Purpose**: Testing SSH key-based authentication scenarios

## 🔧 Common Configuration

All test containers include:

- **SSH Server**: OpenSSH with password and root login enabled
- **Systemd**: Full systemd support for service management
- **Default Credentials**:
  - Root: `root:rootpass123`
  - Test User: `testuser:testpass123`
- **Sudo**: Test user has passwordless sudo access
- **Python 3**: Pre-installed with pip configured for Aliyun mirrors
- **Network Tools**: curl, wget, ping, netstat, etc.
- **Work Directories**: `/opt/saltstack`, `/opt/slurm`

## 🚀 Quick Start

### Using docker-compose (Recommended)

```bash
# Build all test containers
docker-compose -f docker-compose.test.yml build

# Start all test containers
docker-compose -f docker-compose.test.yml up -d

# Check status
docker-compose -f docker-compose.test.yml ps

# View logs
docker-compose -f docker-compose.test.yml logs -f test-ssh01

# Stop all containers
docker-compose -f docker-compose.test.yml down
```

### Manual Build

```bash
# Build Ubuntu test container
docker build -f src/test-containers/Dockerfile \
  -t test-ubuntu:v0.3.6-dev \
  --build-arg VERSION=v0.3.6-dev \
  src/test-containers

# Build Rocky Linux test container
docker build -f src/test-containers/Dockerfile.rocky \
  -t test-rocky:v0.3.6-dev \
  --build-arg VERSION=v0.3.6-dev \
  src/test-containers
```

## 🔌 Container Mapping

When started via `docker-compose.test.yml`:

| Container | OS | Hostname | SSH Port | Container Name |
|-----------|-------|----------|----------|----------------|
| test-ssh01 | Ubuntu 22.04 | test-ssh01 | 2201 | test-ssh01 |
| test-ssh02 | Ubuntu 22.04 | test-ssh02 | 2202 | test-ssh02 |
| test-ssh03 | Ubuntu 22.04 | test-ssh03 | 2203 | test-ssh03 |
| test-rocky01 | Rocky Linux 9 | test-rocky01 | 2211 | test-rocky01 |
| test-rocky02 | Rocky Linux 9 | test-rocky02 | 2212 | test-rocky02 |
| test-rocky03 | Rocky Linux 9 | test-rocky03 | 2213 | test-rocky03 |

## 📡 Connect to Containers

```bash
# SSH to Ubuntu container
ssh root@localhost -p 2201
# Password: rootpass123

# SSH to Rocky Linux container
ssh root@localhost -p 2211
# Password: rootpass123

# Using test user
ssh testuser@localhost -p 2201
# Password: testpass123
```

## 🧪 Testing Workflow

### 1. Install SaltStack Minion

```bash
# SSH into container
ssh root@localhost -p 2201

# Install from AppHub (served by ai-infra-apphub)
curl -fsSL http://apphub:8081/packages/install-salt-minion.sh | bash
```

### 2. Install SLURM Client

```bash
# SSH into container
ssh root@localhost -p 2201

# Install from AppHub
curl -fsSL http://apphub:8081/packages/install-slurm.sh | bash
```

### 3. Verify Installation

```bash
# Check SaltStack service
systemctl status salt-minion

# Check SLURM client
sinfo
squeue
```

## 🔍 Environment Variables

Customize via `.env` or docker-compose environment:

```bash
# Image version tag
IMAGE_TAG=v0.3.6-dev

# SSH credentials (can override defaults)
TEST_SSH_PASSWORD=testpass123
TEST_ROOT_PASSWORD=rootpass123
```

## 📝 Notes

- **Mirrors**: All images use Aliyun mirrors for faster package downloads in China
- **Systemd**: Containers run with systemd as PID 1 for full service management
- **Privileged Mode**: Required for systemd and cgroup support
- **Network**: All containers connect to `ai-infra-network` for inter-service communication
- **Security**: These are TEST containers only - do not use in production!

## 🔄 Cleanup

```bash
# Stop and remove all test containers
docker-compose -f docker-compose.test.yml down

# Remove images
docker rmi test-ubuntu:v0.3.6-dev test-rocky:v0.3.6-dev
```

## 🛠️ Troubleshooting

### Container fails to start

Check if systemd is working:
```bash
docker logs test-ssh01
```

### SSH connection refused

Verify container is running and SSH service is active:
```bash
docker exec test-ssh01 systemctl status ssh
# or for Rocky Linux:
docker exec test-rocky01 systemctl status sshd
```

### Cannot resolve hostnames

Ensure you're in the correct Docker network:
```bash
docker network inspect ai-infra-network
```
