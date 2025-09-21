# AppHub Package Repository

This container serves as a global package repository for deb and rpm packages, including Slurm and common tools.

## Usage

Build the image:

```bash
docker build -t apphub .
```

Run the container:

```bash
docker run -p 8080:80 -v /path/to/packages:/usr/share/nginx/html apphub
```

## Structure

- `/usr/share/nginx/html/deb/` - Debian packages
- `/usr/share/nginx/html/rpm/` - RPM packages

Packages are automatically indexed on startup.

## Integration

In docker-compose.yml, add:

```yaml
apphub:
  build: ./src/apphub
  ports:
    - "8080:80"
  volumes:
    - ./data/apphub:/usr/share/nginx/html
```
