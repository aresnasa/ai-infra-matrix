# Third-Party Dependency Download Guide

**[中文文档](../zh_CN/DOWNLOAD_THIRD_PARTY_GUIDE.md)** | **English**

## Overview

The project provides a unified third-party component download script `scripts/download_third_party.sh` to download all required third-party binary files to the `third_party/` directory.

**Recommended Workflow:**
```bash
# 1. Pre-download all dependencies (speeds up subsequent builds)
./build.sh download-deps

# 2. Commit downloaded files to git (team shared cache)
git add third_party/
git commit -m "feat: add third-party dependencies"

# 3. Build AppHub (automatically uses pre-downloaded files)
./build.sh apphub
```

## Supported Components

| Component | Source | Architecture Support | Purpose |
|-----------|--------|---------------------|---------|
| Prometheus | prometheus/prometheus | amd64, arm64 | Monitoring time-series database |
| Node Exporter | prometheus/node_exporter | amd64, arm64 | Host metrics collection |
| Alertmanager | prometheus/alertmanager | amd64, arm64 | Alert management |
| Categraf | flashcatcloud/categraf | amd64, arm64 | Nightingale monitoring agent |
| Munge | dun/munge | Source | Slurm authentication |
| Singularity | sylabs/singularity | **amd64 only** (DEB/RPM), Source | Container runtime |
| SaltStack | saltstack/salt | amd64, arm64 (DEB/RPM) | Configuration management |

> ⚠️ **Note**: Singularity CE 4.3.x only provides x86_64/amd64 pre-built packages. ARM64 users need to compile from source.

## Quick Start

### 1. Download All Components

```bash
# Run unified download script
./scripts/download_third_party.sh

# Use GitHub mirror for acceleration
GITHUB_MIRROR=https://gh-proxy.com/ ./scripts/download_third_party.sh

# Disable mirror, download directly from GitHub
GITHUB_MIRROR="" ./scripts/download_third_party.sh
```

### 2. Directory Structure After Download

```
third_party/
├── prometheus/
│   ├── prometheus-3.4.1.linux-amd64.tar.gz
│   ├── prometheus-3.4.1.linux-arm64.tar.gz
│   └── version.json
├── node_exporter/
│   ├── node_exporter-1.8.2.linux-amd64.tar.gz
│   ├── node_exporter-1.8.2.linux-arm64.tar.gz
│   └── version.json
├── alertmanager/
│   ├── alertmanager-0.28.1.linux-amd64.tar.gz
│   ├── alertmanager-0.28.1.linux-arm64.tar.gz
│   └── version.json
├── categraf/
│   ├── categraf-v0.4.25-linux-amd64.tar.gz
│   ├── categraf-v0.4.25-linux-arm64.tar.gz
│   └── version.json
├── munge/
│   ├── munge-0.5.16.tar.xz
│   └── version.json
├── singularity/
│   ├── singularity-ce_4.3.6-jammy_amd64.deb
│   ├── singularity-ce_4.3.6-noble_amd64.deb
│   ├── singularity-ce-4.3.6-1.el8.x86_64.rpm
│   ├── singularity-ce-4.3.6-1.el9.x86_64.rpm
│   ├── singularity-ce-4.3.6-1.el10.x86_64.rpm
│   ├── singularity-ce-4.3.6.tar.gz  # Source (ARM64 compile from this)
│   └── version.json
└── saltstack/
    ├── salt-common_3007.1_amd64.deb
    ├── salt-minion_3007.1_amd64.deb
    ├── salt-3007.1-0.x86_64.rpm
    └── version.json
```

## Version Configuration

### Version Source Priority

1. **Environment Variable**: Direct setting like `PROMETHEUS_VERSION=v3.5.0`
2. **.env File**: Read from `.env` file in project root
3. **Dockerfile**: Read from ARG definitions in `src/apphub/Dockerfile`
4. **Default Value**: Built-in default version in script

### Modifying Versions

Method 1: Edit `.env` file
```bash
# .env
PROMETHEUS_VERSION=v3.4.1
NODE_EXPORTER_VERSION=v1.8.2
ALERTMANAGER_VERSION=v0.28.1
```

Method 2: Override via environment variable
```bash
PROMETHEUS_VERSION=v3.5.0 ./scripts/download_third_party.sh
```

Method 3: Modify ARG in `src/apphub/Dockerfile`
```dockerfile
ARG CATEGRAF_VERSION=v0.4.26
ARG SINGULARITY_VERSION=v4.2.2
ARG SALTSTACK_VERSION=v3007.1
```

## GitHub Mirror Acceleration

In regions with restricted network access (like mainland China), the script uses `gh-proxy.com` mirror by default.

### Supported Mirrors

```bash
# Default mirror
GITHUB_MIRROR=https://gh-proxy.com/

# Other available mirrors
GITHUB_MIRROR=https://ghproxy.net/
GITHUB_MIRROR=https://mirror.ghproxy.com/

# Disable mirror
GITHUB_MIRROR=""
```

### Fallback Mechanism

The script automatically performs fallback:
1. First try download via mirror (30 second timeout)
2. If mirror fails, try direct download from GitHub (60 second timeout)
3. Maximum 3 retries

## AppHub Independent Download Scripts

Besides the unified script, `src/apphub/scripts/` directory contains independent download scripts for each component, used inside AppHub container:

```
src/apphub/scripts/
├── prometheus/
│   └── download-prometheus.sh    # Prometheus download
├── node_exporter/
│   └── download-node-exporter.sh # Node Exporter download
├── categraf/
│   └── download-categraf.sh      # Categraf download
│   └── install-categraf.sh       # Categraf installation
└── ...
```

These scripts are suitable for:
- AppHub container runtime dynamic download
- Download to `/usr/share/nginx/html/pkgs/` for HTTP serving
- Generate latest symlinks

## Docker Build Integration

### Pre-download Before Build

```bash
# 1. Pre-download all dependencies
./scripts/download_third_party.sh

# 2. Build AppHub (will use files from third_party/)
./build.sh apphub
```

### Usage in Dockerfile

```dockerfile
# Copy pre-downloaded files
COPY third_party/categraf/ /tmp/categraf/
COPY third_party/prometheus/ /tmp/prometheus/

# Install
RUN tar xzf /tmp/categraf/categraf-*.tar.gz -C /opt/
```

## version.json Format

Each component directory generates `version.json` recording download information:

```json
{
    "component": "prometheus",
    "version": "v3.4.1",
    "downloaded_at": "2025-01-15T10:30:00Z"
}
```

## Troubleshooting

### Download Failed

1. Check network connection
2. Try different GitHub mirror
3. Verify version number is correct (release exists)

### File Exists But Need Re-download

```bash
# Delete downloaded files
rm -rf third_party/prometheus/

# Re-download
./scripts/download_third_party.sh
```

### Version Number Format Issues

- Prometheus/Node Exporter/Alertmanager: Filename without `v` prefix (e.g., `prometheus-3.4.1.linux-amd64.tar.gz`)
- Categraf: Filename with `v` prefix (e.g., `categraf-v0.4.25-linux-amd64.tar.gz`)
- Script automatically handles these differences

## Related Documentation

- [AppHub Build Guide](./APPHUB_BUILD_COMPONENTS.md)
- [Monitoring System Architecture](./MONITORING.md)
- [Categraf Integration](./APPHUB_CATEGRAF_GUIDE.md)
