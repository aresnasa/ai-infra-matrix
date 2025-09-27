# SaltStack Remote Installation Scripts

This document describes the scripts available for remotely installing SaltStack Minion on target hosts via SSH.

## Overview

The AI Infrastructure Matrix project includes two scripts for remotely installing SaltStack Minion:

1. `install-saltstack-remote.sh` - Installs SaltStack on a single host
2. `install-saltstack-parallel.sh` - Installs SaltStack on multiple hosts in parallel

These scripts support both password and key-based SSH authentication and automatically detect the target OS to install the appropriate SaltStack packages.

## Single Host Installation

### Usage

```bash
./scripts/install-saltstack-remote.sh [OPTIONS] <host>
```

### Options

- `-h, --help` - Show help message
- `-u, --user USER` - SSH user (default: root)
- `-p, --port PORT` - SSH port (default: 22)
- `-i, --identity KEY_PATH` - SSH private key path
- `-P, --password PASSWORD` - SSH password (not recommended for production)
- `--master MASTER` - SaltStack Master address (default: salt-master)
- `--minion-id MINION_ID` - SaltStack Minion ID (default: hostname)

### Examples

```bash
# Install on a single host with key authentication
./scripts/install-saltstack-remote.sh -u ubuntu -i ~/.ssh/id_rsa --master 192.168.1.10 192.168.1.100

# Install with password authentication
./scripts/install-saltstack-remote.sh -u centos -P password123 --minion-id worker01 192.168.1.101
```

## Parallel Installation

### Usage

```bash
./scripts/install-saltstack-parallel.sh [OPTIONS] <hosts_file>
```

### Options

- `-h, --help` - Show help message
- `-u, --user USER` - SSH user (default: root)
- `-p, --port PORT` - SSH port (default: 22)
- `-i, --identity KEY_PATH` - SSH private key path
- `-P, --password PASSWORD` - SSH password (not recommended for production)
- `-c, --concurrent COUNT` - Number of concurrent installations (default: 5)
- `--master MASTER` - SaltStack Master address (default: salt-master)

### Hosts File Format

The hosts file should contain one host per line with the following format:

```
hostname_or_ip[:port] [minion_id]
```

Lines starting with `#` are treated as comments and ignored.

Example hosts file:

```
# Simple host entry with default port and minion ID
192.168.1.100

# Host with custom minion ID
192.168.1.101 worker01

# Host with custom port
192.168.1.102:2222

# Host with both custom port and minion ID
192.168.1.103:2201 worker03
```

### Examples

```bash
# Install on hosts listed in hosts.txt
./scripts/install-saltstack-parallel.sh config/example-hosts.txt

# Install with custom parameters
./scripts/install-saltstack-parallel.sh -u ubuntu -i ~/.ssh/id_rsa --master 192.168.1.10 -c 10 config/example-hosts.txt
```

## Supported Operating Systems

The scripts currently support the following operating systems:

1. Ubuntu/Debian
2. CentOS/RHEL 8+

## Prerequisites

1. SSH access to target hosts
2. For password authentication: `sshpass` utility installed
3. For key-based authentication: SSH private key with appropriate permissions
4. Target hosts must have internet access to download packages

## Process Overview

The installation process performs the following steps:

1. Establish SSH connection to target host
2. Detect OS distribution and version
3. Install required dependencies
4. Add SaltStack repository
5. Install SaltStack Minion package
6. Configure Minion to connect to the specified Master
7. Start the Minion service

## Security Considerations

1. Password authentication should only be used for testing purposes
2. In production, always use key-based authentication
3. Protect SSH private keys with appropriate file permissions (600)
4. Consider using a configuration management tool like SaltStack itself for ongoing management

## Troubleshooting

1. **SSH connection failures**:
   - Check SSH credentials
   - Verify target host is reachable
   - Ensure SSH service is running on target

2. **Package installation failures**:
   - Check internet connectivity on target host
   - Verify OS is supported
   - Check available disk space

3. **Service start failures**:
   - Check configuration file syntax
   - Verify Master address is reachable
   - Check system logs on target host

## Integration with AI Infrastructure Matrix

These scripts complement the existing SaltStack management functionality in the AI Infrastructure Matrix by providing a way to bootstrap new minions. Once installed, minions can be managed through the platform's existing SaltStack API and web interface.