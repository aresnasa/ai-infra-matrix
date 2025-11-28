# AI Infrastructure Matrix User Guide

## Overview

This guide provides detailed instructions for using the AI Infrastructure Matrix platform, helping users get started with various features quickly.

## Table of Contents

- [Login and Authentication](#login-and-authentication)
- [JupyterHub Usage](#jupyterhub-usage)
- [Gitea Code Repository](#gitea-code-repository)
- [Slurm Job Management](#slurm-job-management)
- [Object Storage](#object-storage)
- [Monitoring Dashboard](#monitoring-dashboard)

## Login and Authentication

### First Login

1. Open your browser and visit `http://localhost:8080`
2. Log in with the default administrator account:
   - Username: `admin`
   - Password: `admin123`
3. It is recommended to change your password immediately after first login

### User Management

Administrators can create and manage user accounts in the backend management interface.

## JupyterHub Usage

### Accessing JupyterHub

Visit `http://localhost:8080/jupyter` to access the JupyterHub environment.

### Creating a Notebook

1. After logging in, click the "New" button
2. Select Python 3 kernel
3. Start writing code

### Using GPU Resources

If GPU resources are configured, you can use them in your Notebook:

```python
import torch
print(torch.cuda.is_available())
```

For more details, refer to: [JupyterHub Usage Guide](JUPYTERHUB_UNIFIED_AUTH_GUIDE.md)

## Gitea Code Repository

### Accessing Gitea

Visit `http://localhost:8080/gitea/` to access the Gitea code repository.

### Creating a Repository

1. After logging in, click the "+" button in the upper right corner
2. Select "New Repository"
3. Fill in the repository name and description
4. Choose public or private
5. Click "Create Repository"

### Cloning a Repository

```bash
git clone http://localhost:8080/gitea/username/repository.git
```

### LFS Large File Storage

Gitea is configured with MinIO as the LFS backend, supporting large file storage:

```bash
# Install Git LFS
git lfs install

# Track large files
git lfs track "*.psd"
git add .gitattributes
git commit -m "Track PSD files"
```

## Slurm Job Management

### Accessing Slurm Management Interface

Navigate to "Slurm Cluster Management" in the main interface.

### Submitting Jobs

1. Click "Job Management" -> "New Job"
2. Fill in job parameters:
   - Job name
   - Queue (partition)
   - Number of nodes
   - CPU/Memory requirements
3. Upload or write job script
4. Click "Submit"

### Viewing Job Status

In the job list, you can view:
- Job ID
- Status (Queued/Running/Completed/Failed)
- Runtime
- Resource usage

### Node Management

Administrators can:
- Add compute nodes
- View node status
- Configure partitions (queues)
- Set resource limits

## Object Storage

### Accessing MinIO Console

Visit `http://localhost:8080/minio-console/`

### Creating a Bucket

1. Log in to the console
2. Click "Buckets" -> "Create Bucket"
3. Enter bucket name
4. Configure access policy
5. Click "Create"

### Uploading Files

1. Select the bucket
2. Click the "Upload" button
3. Select files or drag and drop to upload

### S3 API Access

Access using AWS CLI or boto3:

```python
import boto3

s3 = boto3.client('s3',
    endpoint_url='http://localhost:8080/minio',
    aws_access_key_id='minioadmin',
    aws_secret_access_key='minioadmin'
)

# List all buckets
buckets = s3.list_buckets()
```

## Monitoring Dashboard

### Accessing Nightingale

Visit `http://localhost:8080/n9e` to access the monitoring system.

### Viewing Dashboards

- System Overview
- Node Monitoring
- Service Status
- Resource Usage Trends

### Configuring Alerts

1. Go to "Alert Rules"
2. Click "New Rule"
3. Configure monitoring metrics and thresholds
4. Set notification methods
5. Save the rule

## FAQ

### Forgot Password

Contact the administrator to reset your password.

### Job Stuck in Queue

Check:
1. Whether cluster nodes are online
2. Whether resource quota is sufficient
3. Whether queue configuration is correct

### Service Access Failed

1. Check service status: `docker compose ps`
2. View service logs: `docker compose logs [service-name]`
3. Confirm network connection is normal

## Getting Help

- 📧 Technical Support: support@example.com
- 📖 Documentation: [docs/](.)
- 🐛 Issue Reporting: [GitHub Issues](https://github.com/aresnasa/ai-infra-matrix/issues)
