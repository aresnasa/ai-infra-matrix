# API Reference

## Overview

AI Infrastructure Matrix provides RESTful APIs for programmatic access and integration.

## Basic Information

- **Base URL**: `http://localhost:8080/api`
- **Authentication**: JWT Token
- **Content-Type**: `application/json`

## Authentication

### Get Token

```http
POST /api/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "admin123"
}
```

**Response:**

```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "username": "admin",
    "role": "admin"
  }
}
```

### Using Token

Add Authorization header to subsequent requests:

```http
Authorization: Bearer <token>
```

## Slurm Cluster Management API

### Get Cluster List

```http
GET /api/slurm/clusters
Authorization: Bearer <token>
```

**Response:**

```json
{
  "clusters": [
    {
      "id": 1,
      "name": "default",
      "status": "active",
      "nodes": 3,
      "total_cpus": 24,
      "total_memory": "128GB"
    }
  ]
}
```

### Submit Job

```http
POST /api/slurm/jobs
Authorization: Bearer <token>
Content-Type: application/json

{
  "job_name": "test_job",
  "partition": "compute",
  "nodes": 1,
  "ntasks": 4,
  "script": "#!/bin/bash\nsleep 60"
}
```

**Response:**

```json
{
  "job_id": 12345,
  "status": "pending",
  "submit_time": "2025-11-18T13:00:00Z"
}
```

### Query Job Status

```http
GET /api/slurm/jobs/{job_id}
Authorization: Bearer <token>
```

**Response:**

```json
{
  "job_id": 12345,
  "status": "running",
  "partition": "compute",
  "nodes": ["node01"],
  "start_time": "2025-11-18T13:01:00Z",
  "elapsed_time": "00:05:30"
}
```

### Cancel Job

```http
DELETE /api/slurm/jobs/{job_id}
Authorization: Bearer <token>
```

### Get Node List

```http
GET /api/slurm/nodes
Authorization: Bearer <token>
```

**Response:**

```json
{
  "nodes": [
    {
      "name": "node01",
      "state": "allocated",
      "cpus": 8,
      "memory": "32GB",
      "jobs": [12345]
    },
    {
      "name": "node02",
      "state": "idle",
      "cpus": 8,
      "memory": "32GB",
      "jobs": []
    }
  ]
}
```

## SaltStack Management API

### Get Minion List

```http
GET /api/saltstack/minions
Authorization: Bearer <token>
```

### Execute Command

```http
POST /api/saltstack/execute
Authorization: Bearer <token>
Content-Type: application/json

{
  "target": "node01",
  "function": "test.ping"
}
```

### Deploy Configuration

```http
POST /api/saltstack/deploy
Authorization: Bearer <token>
Content-Type: application/json

{
  "target": "node*",
  "state": "slurm.compute"
}
```

## JupyterHub API

### Get User List

```http
GET /api/jupyterhub/users
Authorization: Bearer <token>
```

### Create User

```http
POST /api/jupyterhub/users
Authorization: Bearer <token>
Content-Type: application/json

{
  "username": "newuser",
  "admin": false
}
```

### Start Server

```http
POST /api/jupyterhub/users/{username}/server
Authorization: Bearer <token>
```

## Object Storage API

MinIO is compatible with AWS S3 API. Reference:

- [AWS S3 API Documentation](https://docs.aws.amazon.com/s3/)
- [MinIO SDK](https://min.io/docs/minio/linux/developers/minio-drivers.html)

### Python Example

```python
import boto3

s3 = boto3.client('s3',
    endpoint_url='http://localhost:8080/minio',
    aws_access_key_id='minioadmin',
    aws_secret_access_key='minioadmin'
)

# Upload file
s3.upload_file('local.txt', 'bucket-name', 'remote.txt')

# Download file
s3.download_file('bucket-name', 'remote.txt', 'local.txt')
```

## Monitoring API

### Get System Metrics

```http
GET /api/monitoring/metrics
Authorization: Bearer <token>
```

**Response:**

```json
{
  "cpu_usage": 45.5,
  "memory_usage": 68.2,
  "disk_usage": 52.1,
  "network_in": 1024000,
  "network_out": 512000
}
```

### Query Historical Data

```http
GET /api/monitoring/history?metric=cpu_usage&start=2025-11-18T00:00:00Z&end=2025-11-18T23:59:59Z
Authorization: Bearer <token>
```

## Error Codes

| Status Code | Description |
|-------------|-------------|
| 200 | Success |
| 201 | Created Successfully |
| 400 | Bad Request |
| 401 | Unauthorized (Invalid or expired token) |
| 403 | Forbidden |
| 404 | Not Found |
| 500 | Internal Server Error |

## SDK and Tools

### Go SDK

```go
import "github.com/aresnasa/ai-infra-matrix/sdk/go"

client := aiinfra.NewClient("http://localhost:8080", "your-token")
jobs, err := client.Slurm.ListJobs()
```

### Python SDK

```python
from aiinfra import Client

client = Client("http://localhost:8080", token="your-token")
jobs = client.slurm.list_jobs()
```

### CLI Tool

```bash
# Install
go install github.com/aresnasa/ai-infra-matrix/cmd/aiinfra@latest

# Configure
aiinfra config set-url http://localhost:8080
aiinfra login admin

# Usage
aiinfra slurm jobs list
aiinfra slurm jobs submit --script job.sh
```

## Webhook

Support webhook configuration for event notifications:

```json
{
  "url": "https://your-webhook.example.com",
  "events": ["job.completed", "job.failed", "node.down"],
  "secret": "your-webhook-secret"
}
```

## Rate Limiting

API rate limiting policies:
- Authentication requests: 10/minute
- General requests: 100/minute
- Administrative operations: 50/minute

## More Information

- OpenAPI Specification: `/api/openapi.json`
- Swagger UI: `http://localhost:8080/api/docs`
- GraphQL: `http://localhost:8080/api/graphql` (Experimental)
