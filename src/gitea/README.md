# Gitea Integration (Auto-Init)

This image wraps `gitea/gitea:1.22` with an entrypoint that:

- Waits for PostgreSQL, creates DB/user (idempotent)
- Renders `/data/gitea/conf/app.ini` to skip the install wizard
- Supports storage on local/NFS or S3-compatible (SeaweedFS)

Environment highlights:

- DB: `GITEA_DB_TYPE=postgres`, `GITEA_DB_HOST=postgres:5432`, `GITEA_DB_NAME=gitea`, `GITEA_DB_USER=gitea`, `GITEA_DB_PASSWD=...`
- Server: `ROOT_URL=http://localhost:8080/gitea/`, `SUBURL=/gitea`
- Storage: `GITEA__storage__STORAGE_TYPE=local|minio` (Note: 'minio' is Gitea's config type for S3-compatible storage)
  - local: `DATA_PATH=/data/gitea` (backed by volume, can be NFS)
  - minio: `MINIO_ENDPOINT=seaweedfs-filer:8333`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET=gitea`, `MINIO_USE_SSL=false`

Compose uses SeaweedFS as the default S3-compatible object storage backend.

References:

- <https://docs.gitea.com/installation/install-with-docker>
- <https://docs.gitea.com/installation/install-on-kubernetes>

## Gitea integration for AI-Infra-Matrix

This module packages a Gitea service for embedding into the portal via an iframe at /gitea.

## How it works

- We build a thin image FROM gitea/gitea and run it as an internal service.
- Nginx proxies /gitea/ to the gitea container.
- The frontend exposes a menu item “Gitea” (between Projects and Kubernetes) that loads an iframe pointing to /gitea/.

## Configuration

- Default admin and data are provided by Gitea’s own entrypoint. Persist data using a volume.
- Override URL for the iframe via:
  - window.__GITEA_URL__ at runtime, or
  - REACT_APP_GITEA_URL at build time.

## Next steps

- Add Nginx location mapping and docker-compose service to wire this in runtime.
