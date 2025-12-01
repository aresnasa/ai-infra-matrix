# Auto Version Management Guide

**[中文文档](../zh_CN/BUILD_AUTO_VERSION_GUIDE.md)** | **English**

## Overview

Starting from v0.3.8, the `build.sh` script introduces automatic version management features, supporting Git branch-based auto tag detection and unified dependency version management, significantly simplifying intranet deployment workflows.

## Core Features

### 1. Git Branch Auto Tag Detection

**Feature**: Automatically use the current Git branch name as the image tag

**Supported Commands**:
- `build-all`
- `push-all`
- `push-dep`

**How It Works**:
```bash
# Example: current branch is v0.3.8
git branch --show-current  # Output: v0.3.8

# Execute build without specifying tag
./build.sh build-all

# Auto-detect and use v0.3.8 as image tag
# Equivalent to: ./build.sh build-all v0.3.8
```

### 2. deps.yaml Dependency Version Management

**Feature**: Centralized definition of all upstream dependency image versions

**File Location**: `deps.yaml`

**File Format**:
```yaml
# Databases
postgres: "15-alpine"
mysql: "8.0"
redis: "7-alpine"

# Programming Language Runtimes
golang: "1.25-alpine"
node: "22-alpine"
python: "3.14-alpine"

# Applications
gitea: "1.25.1"
nginx: "stable-alpine-perl"
minio: "latest"
```

**Sync Mechanism**:
- `build-all` automatically calls `sync_deps_from_yaml()` function
- Syncs versions from deps.yaml to .env file
- Example: `postgres: "15-alpine"` → `.env`: `POSTGRES_VERSION=15-alpine`

### 3. Automatic Environment Variable Updates

**Feature**: Automatically update version-related variables in .env file

**Updated Variables**:
- `IMAGE_TAG` - Component image tag
- `DEFAULT_IMAGE_TAG` - Default image tag
- `*_VERSION` - Various dependency image versions (synced from deps.yaml)

## Usage Scenarios

### Scenario 1: Quick Local Development Build

```bash
# 1. Switch to target branch
git checkout v0.3.8

# 2. Build directly without specifying tag
./build.sh build-all

# Output:
# [INFO] Step 0: Auto version detection and sync
# [INFO] No tag specified, auto-detecting from Git branch...
# [INFO] Detected Git branch: v0.3.8
# [INFO] Component tag set to: v0.3.8
# [INFO] Auto-set tag to: v0.3.8
# [INFO] Syncing dependency versions from deps.yaml to /path/to/.env
# [INFO]   ✓ POSTGRES_VERSION=15-alpine
# [INFO]   ✓ MYSQL_VERSION=8.0
# ...
```

### Scenario 2: Intranet Deployment Image Push

```bash
# 1. Confirm current branch
git branch --show-current  # v0.3.8

# 2. Push all images to intranet Harbor
./build.sh push-all harbor.example.com/ai-infra

# Output:
# [INFO] No tag specified, auto-detecting from Git branch...
# [INFO] Auto-set tag to: v0.3.8
# [INFO] Pushing ai-infra-backend:v0.3.8...
# [INFO] Pushing ai-infra-frontend:v0.3.8...
# ...

# 3. Push only dependency images
./build.sh push-dep harbor.example.com/ai-infra

# Output:
# [INFO] No tag specified, auto-detecting from Git branch...
# [INFO] Auto-set tag to: v0.3.8
# [INFO] Pushing postgres:15-alpine...
# [INFO] Pushing redis:7-alpine...
# ...
```

### Scenario 3: Multi-Version Parallel Development

```bash
# Develop v0.3.8 feature branch
git checkout v0.3.8
./build.sh build-all
# → Build v0.3.8 tagged images

# Switch to v0.4.0 development branch
git checkout v0.4.0
./build.sh build-all
# → Build v0.4.0 tagged images

# Two versions coexist without interference
docker images | grep ai-infra-backend
# ai-infra-backend    v0.3.8    ...
# ai-infra-backend    v0.4.0    ...
```

### Scenario 4: Manual Tag Override

```bash
# Current branch is v0.3.8, but want to build test version
./build.sh build-all v0.3.8-test

# Output:
# [INFO] Using manually specified tag: v0.3.8-test
# [INFO] Syncing dependency versions from deps.yaml to /path/to/.env
# ...
# → Build v0.3.8-test tagged images
```

## Dependency Version Upgrade Workflow

### 1. Update deps.yaml

```bash
# Edit deps.yaml
vim deps.yaml

# Example: upgrade PostgreSQL
postgres: "15-alpine"  # Old version
↓
postgres: "16-alpine"  # New version
```

### 2. Sync to .env

```bash
# Method 1: Execute build-all for auto sync
./build.sh build-all

# Method 2: Manual sync (internal script function)
# sync_deps_from_yaml "$SCRIPT_DIR/.env"
```

### 3. Verify Sync Results

```bash
# Check .env file
cat .env | grep POSTGRES_VERSION
# Output: POSTGRES_VERSION=16-alpine
```

### 4. Rebuild Dependent Services

```bash
# Force rebuild services using PostgreSQL
./build.sh build backend --force
```

## Best Practices

### 1. Branch Naming Convention

**Recommended Format**: `v<major>.<minor>.<patch>`

**Examples**:
- `v0.3.8` - Stable version
- `v0.4.0` - Next major version
- `v0.3.9-dev` - Development branch
- `v0.3.8-hotfix` - Hotfix branch

**Benefits**:
- Auto-generate semantic version compliant image tags
- Easy to identify versions in intranet deployment
- Docker image tag specification compliant

### 2. deps.yaml Maintenance

**When to Update**:
- Upstream dependency releases new version
- Security vulnerability requires upgrade
- New dependency component added

**Version Selection Principles**:
- Prefer alpine variants (smaller image size)
- Use specific version numbers, avoid `latest`
- Test before applying to production

**Example deps.yaml**:
```yaml
# Databases - use specific minor version
postgres: "15.5-alpine"    # Recommended
# postgres: "latest"       # Not recommended

# Applications - specify full version
gitea: "1.25.1"            # Recommended
# gitea: "1.25"            # Acceptable
# gitea: "latest"          # Not recommended
```

### 3. Intranet Deployment Workflow

**Complete Process**:
```bash
# === External Build Environment ===
# 1. Switch to release branch
git checkout v0.3.8

# 2. Build all images
./build.sh build-all

# 3. Push to intranet Harbor
./build.sh push-all harbor.example.com/ai-infra

# 4. Push dependency images
./build.sh push-dep harbor.example.com/ai-infra

# === Intranet Deployment Environment ===
# 5. Pull images
./build.sh harbor-pull-all harbor.example.com/ai-infra v0.3.8

# 6. Start services
docker-compose up -d
```

## Troubleshooting

### Issue 1: Cannot Detect Git Branch

**Symptom**:
```
[INFO] Detected Git branch: latest
```

**Cause**:
- Not in a Git repository
- HEAD is in detached state

**Solution**:
```bash
# Confirm in Git repository
git status

# Switch to specific branch
git checkout v0.3.8

# Or manually specify tag
./build.sh build-all v0.3.8
```

### Issue 2: deps.yaml Sync Failed

**Symptom**:
```
[WARNING] Dependency file not found: /path/to/deps.yaml, skipping sync
```

**Cause**:
- deps.yaml file doesn't exist
- File path error

**Solution**:
```bash
# Confirm file exists
ls -l deps.yaml

# Check file format
cat deps.yaml

# Create manually if missing
cat > deps.yaml <<'EOF'
postgres: "15-alpine"
redis: "7-alpine"
EOF
```

### Issue 3: Environment Variable Not Updated

**Symptom**:
```bash
echo $IMAGE_TAG
# Output: latest (expected: v0.3.8)
```

**Cause**:
- Environment variable not reloaded

**Solution**:
```bash
# Reload .env
source .env

# Or execute in new shell session
./build.sh build-all
```

## Related Documentation

- [Build Script Complete Guide](BUILD_COMPLETE_USAGE.md)
- [Intranet Deployment Guide](BUILD_ENV_MANAGEMENT.md)
- [Dependency Image Management](BUILD_IMAGE_PREFETCH.md)
- [Version Management Best Practices](APPHUB_VERSION_MANAGEMENT.md)

## Summary

Auto version management simplifies build and deployment through:

1. **Auto Tag Detection**: Git branch name becomes image tag automatically
2. **Centralized Version Management**: deps.yaml manages all dependency versions
3. **One-Click Sync**: Auto sync configuration to .env file
4. **Intranet Friendly**: Optimized for intranet deployment scenarios

**Key Benefits**:
- ✅ Reduce manual input errors
- ✅ Ensure version consistency
- ✅ Simplify intranet deployment
- ✅ Support multi-version parallel development
- ✅ Backward compatible (manual override available)
