# Monitoring Component Deployment SOP (Standard Operating Procedure)

**[中文文档](../zh_CN/MONITORING_DEPLOYMENT_SOP.md)** | **English**

This document details the deployment process for the AI Infra Matrix monitoring stack, including Prometheus, Node Exporter, Categraf, and integration with Nightingale/VictoriaMetrics.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Prerequisites](#prerequisites)
- [Component Versions](#component-versions)
- [Deployment Process](#deployment-process)
  - [1. Verify AppHub Service](#1-verify-apphub-service)
  - [2. Deploy Node Exporter](#2-deploy-node-exporter)
  - [3. Deploy Categraf](#3-deploy-categraf)
  - [4. Deploy Prometheus (Optional)](#4-deploy-prometheus-optional)
- [Verify Monitoring Data](#verify-monitoring-data)
- [Troubleshooting](#troubleshooting)
- [Appendix](#appendix)

---

## Overview

AI Infra Matrix uses the following monitoring architecture:

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   Target Host   │     │   Target Host   │     │   Target Host   │
│  ┌───────────┐  │     │  ┌───────────┐  │     │  ┌───────────┐  │
│  │ Categraf  │──┼─────┼──│ Categraf  │──┼─────┼──│ Categraf  │  │
│  └───────────┘  │     │  └───────────┘  │     │  └───────────┘  │
│  ┌───────────┐  │     │  ┌───────────┐  │     │  ┌───────────┐  │
│  │Node Export│  │     │  │Node Export│  │     │  │Node Export│  │
│  └───────────┘  │     │  └───────────┘  │     │  └───────────┘  │
└────────┬────────┘     └────────┬────────┘     └────────┬────────┘
         │                       │                       │
         │    ┌─────────────────────────────────────┐   │
         └────┤         Nightingale (n9e)           ├───┘
              │   - Metric aggregation, alerts, viz │
              └───────────────┬─────────────────────┘
                              │
              ┌───────────────▼───────────────┐
              │      VictoriaMetrics          │
              │   - Time series storage       │
              └───────────────────────────────┘
```

### Monitoring Component Responsibilities

| Component | Responsibility | Port |
|-----------|----------------|------|
| **Categraf** | Multi-function monitoring agent, push mode to Nightingale | 9100 (metrics) |
| **Node Exporter** | Prometheus-style node metrics exporter, pull mode | 9100 (metrics) |
| **Nightingale** | Alert platform, metric aggregation, visualization | 17000 |
| **VictoriaMetrics** | High-performance time series database, Prometheus compatible | 8428 |
| **Prometheus** | (Optional) Traditional monitoring scrape and storage | 9090 |

---

## Architecture

### Data Flow

1. **Push Mode (Recommended)**:
   ```
   Categraf → Nightingale → VictoriaMetrics
   ```

2. **Pull Mode**:
   ```
   Prometheus/Nightingale ← Node Exporter
   ```

### Port Planning

| Service | Port | Protocol | Description |
|---------|------|----------|-------------|
| AppHub | 28080 | HTTP | Package and script download |
| Nightingale | 17000 | HTTP | Monitoring platform Web UI and API |
| VictoriaMetrics | 8428 | HTTP | Time series database HTTP API |
| Categraf | 9100 | HTTP | Metrics endpoint |
| Node Exporter | 9100 | HTTP | Metrics endpoint |
| Prometheus | 9090 | HTTP | Prometheus Web UI (optional) |

---

## Prerequisites

### 1. Platform Service Status

Ensure the following core services are running:

```bash
# Check Docker service status
docker ps --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | grep -E "apphub|nightingale|victoriametrics"
```

Expected output:
```
apphub              Up X hours   0.0.0.0:28080->80/tcp
nightingale         Up X hours   0.0.0.0:17000->17000/tcp
victoriametrics     Up X hours   0.0.0.0:8428->8428/tcp
```

### 2. AppHub Package Availability Verification

```bash
# Set AppHub address
export APPHUB_HOST="192.168.0.200"
export APPHUB_PORT="28080"

# Verify Categraf package
curl -sI "http://${APPHUB_HOST}:${APPHUB_PORT}/pkgs/categraf/categraf-latest-linux-amd64.tar.gz" | head -5

# Verify Node Exporter package
curl -sI "http://${APPHUB_HOST}:${APPHUB_PORT}/pkgs/node_exporter/VERSION" | head -5

# Verify Prometheus package
curl -sI "http://${APPHUB_HOST}:${APPHUB_PORT}/pkgs/prometheus/VERSION" | head -5
```

### 3. Network Connectivity

```bash
# Test connectivity from target host to AppHub
curl -s "http://${APPHUB_HOST}:${APPHUB_PORT}/" | head -5

# Test connectivity from target host to Nightingale
curl -s "http://${APPHUB_HOST}:17000/api/n9e/version"
```

---

## Component Versions

Current supported versions (stored in AppHub):

| Component | Version | amd64 | arm64 |
|-----------|---------|-------|-------|
| Categraf | v0.4.25 | ✅ | ✅ |
| Node Exporter | 1.8.2 | ✅ | ✅ |
| Prometheus | 3.4.1 | ✅ | ✅ |

Query latest versions:
```bash
curl -s "http://${APPHUB_HOST}:${APPHUB_PORT}/pkgs/categraf/version.json" | jq .
curl -s "http://${APPHUB_HOST}:${APPHUB_PORT}/pkgs/node_exporter/VERSION"
curl -s "http://${APPHUB_HOST}:${APPHUB_PORT}/pkgs/prometheus/VERSION"
```

---

## Deployment Process

### 1. Verify AppHub Service

Before any installation, first verify AppHub service status:

```bash
#!/bin/bash
# Verify AppHub service
APPHUB_URL="http://192.168.0.200:28080"

echo "Checking AppHub service..."
if curl -sI "${APPHUB_URL}/pkgs/" | grep -q "200 OK"; then
    echo "✓ AppHub service is healthy"
else
    echo "✗ AppHub service unavailable"
    exit 1
fi

echo ""
echo "Available packages:"
curl -s "${APPHUB_URL}/pkgs/" | grep -oP 'href="\K[^"]+' | grep -v "^\.\." | head -20
```

### 2. Deploy Node Exporter

Node Exporter provides standard Prometheus-style metrics export.

#### Method 1: Use Rendered Install Script

```bash
# Download and execute install script from AppHub
curl -fsSL http://192.168.0.200:28080/scripts/install-node-exporter.sh | sudo bash
```

#### Method 2: Manual Installation

```bash
# Set variables
NODE_EXPORTER_VERSION="1.8.2"
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
APPHUB_URL="http://192.168.0.200:28080"

# Download
cd /tmp
curl -fsSL -o node_exporter.tar.gz \
    "${APPHUB_URL}/pkgs/node_exporter/node_exporter-${NODE_EXPORTER_VERSION}.linux-${ARCH}.tar.gz"

# Extract and install
tar xzf node_exporter.tar.gz
sudo cp node_exporter-*/node_exporter /usr/local/bin/
sudo chmod +x /usr/local/bin/node_exporter

# Create user
sudo useradd --no-create-home --shell /bin/false node_exporter 2>/dev/null || true

# Create systemd service
sudo tee /etc/systemd/system/node_exporter.service > /dev/null << 'EOF'
[Unit]
Description=Node Exporter
Wants=network-online.target
After=network-online.target

[Service]
User=node_exporter
Group=node_exporter
Type=simple
ExecStart=/usr/local/bin/node_exporter

[Install]
WantedBy=multi-user.target
EOF

# Start service
sudo systemctl daemon-reload
sudo systemctl enable --now node_exporter

# Verify
curl -s localhost:9100/metrics | head -10
```

#### Verify Node Exporter

```bash
# Check service status
sudo systemctl status node_exporter

# Check metrics endpoint
curl -s localhost:9100/metrics | grep -E "^node_cpu|^node_memory" | head -10

# Expected output example:
# node_cpu_seconds_total{cpu="0",mode="idle"} 123456.78
# node_memory_MemTotal_bytes 1.6777216e+10
```

### 3. Deploy Categraf

Categraf is the recommended monitoring agent, supporting active push to Nightingale.

#### Method 1: Use Rendered Install Script (Recommended)

```bash
# Download and execute install script from AppHub
curl -fsSL http://192.168.0.200:28080/scripts/install-categraf.sh | sudo bash
```

#### Verify Categraf

```bash
# Check service status
sudo systemctl status categraf

# Check local metrics
curl -s localhost:9100/metrics | head -20

# Check logs
sudo tail -f /var/log/categraf/categraf.log

# Verify data written to VictoriaMetrics
curl -s "http://192.168.0.200:8428/api/v1/status/tsdb" | jq '.data.totalSeries'
```

### 4. Deploy Prometheus (Optional)

If you need an independent Prometheus instance:

```bash
# Set variables
PROMETHEUS_VERSION="3.4.1"
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
APPHUB_URL="http://192.168.0.200:28080"

# Download and install (see full steps in Chinese documentation)
```

---

## Verify Monitoring Data

### 1. Check VictoriaMetrics Data

```bash
# Check total time series
curl -s "http://192.168.0.200:8428/api/v1/status/tsdb" | jq '.data.totalSeries'

# Query CPU metrics
curl -s "http://192.168.0.200:8428/api/v1/query?query=cpu_usage_idle" | jq '.data.result | length'

# Query memory metrics
curl -s "http://192.168.0.200:8428/api/v1/query?query=mem_available_percent" | jq '.data.result[0].value'
```

### 2. Check Nightingale Host List

Login to Nightingale Web UI: http://192.168.0.200:17000

- Default username: `root`
- Default password: `root.2020`

Verification steps:
1. Go to "Infrastructure" -> "Machine List"
2. Confirm registered hosts appear in the list
3. View host CPU, memory, disk metrics

### 3. Metric Query Examples

Test in Nightingale's "Instant Query":

```promql
# CPU usage
100 - cpu_usage_idle

# Memory usage
100 - mem_available_percent

# Disk usage
100 - disk_free_percent{path="/"}

# Network receive rate
rate(net_bytes_recv[5m])

# System load
system_load1
```

---

## Troubleshooting

### Issue 1: Categraf Cannot Connect to Nightingale

**Symptom**: Categraf logs show connection errors

```bash
# Check logs
sudo journalctl -u categraf -n 50 --no-pager
```

**Solution**:
```bash
# 1. Check network connectivity
curl -v "http://192.168.0.200:17000/api/n9e/version"

# 2. Check URL in config file
grep -A 5 "\[\[writers\]\]" /opt/categraf/conf/config.toml

# 3. Ensure Nightingale service is running
docker ps | grep nightingale
```

### Issue 2: Node Exporter Service Startup Failed

**Symptom**: systemctl status shows failed

```bash
# Check detailed error
sudo journalctl -u node_exporter -n 50

# Common issue: port conflict
sudo ss -tlnp | grep 9100
```

**Solution**:
```bash
# If port is occupied, modify port
sudo sed -i 's/:9100/:9101/' /etc/systemd/system/node_exporter.service
sudo systemctl daemon-reload
sudo systemctl restart node_exporter
```

### Issue 3: VictoriaMetrics Has No Data

**Symptom**: totalSeries is 0

```bash
# Check VictoriaMetrics status
curl -s "http://192.168.0.200:8428/api/v1/status/tsdb" | jq .
```

### Issue 4: AppHub Package Download Failed

**Symptom**: curl returns 404 or connection timeout

```bash
# Check AppHub service
docker ps | grep apphub
docker logs apphub --tail 20
```

---

## Appendix

### A. Template Variable Description

The following variables are used for install script template rendering (configured in `.env` file):

| Variable | Description | Default |
|----------|-------------|---------|
| `EXTERNAL_HOST` | AppHub/Nightingale external access address | 192.168.0.200 |
| `APPHUB_PORT` | AppHub port | 28080 |
| `NIGHTINGALE_PORT` | Nightingale port | 17000 |
| `CATEGRAF_VERSION` | Categraf version | v0.4.25 |
| `NODE_EXPORTER_VERSION` | Node Exporter version | 1.8.2 |
| `PROMETHEUS_VERSION` | Prometheus version | 3.4.1 |

### B. Quick Deployment Commands

```bash
# One-click deploy Categraf (recommended)
curl -fsSL http://192.168.0.200:28080/scripts/install-categraf.sh | sudo bash

# One-click deploy Node Exporter
curl -fsSL http://192.168.0.200:28080/scripts/install-node-exporter.sh | sudo bash

# Verify monitoring data
curl -s "http://192.168.0.200:8428/api/v1/status/tsdb" | jq '.data.totalSeries'
```

---

## Update History

| Date | Version | Description |
|------|---------|-------------|
| 2024-11-28 | 1.0 | Initial version with Categraf, Node Exporter, Prometheus deployment |

---

**Maintainer**: AI Infra Matrix Team  
**Last Updated**: 2024-11-28
