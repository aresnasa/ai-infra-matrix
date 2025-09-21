# Test Containers: Ubuntu 22.04 with OpenSSH Server

This folder provides simple Ubuntu 22.04 images with OpenSSH server enabled for testing file transfer and remote command features.

Variants provided:

- `Dockerfile.ssh`: Baseline Ubuntu 22.04 + openssh-server, user `tester` with password auth.
- `Dockerfile.ssh-key`: Same as above but sets up a default SSH key for `tester` (use for key-auth demos).

## Common defaults

- Base image: `ubuntu:22.04`
- SSH port: `22` inside container
- Default user: `tester` / password: `tester123`
- Root login: disabled
- Authorized keys: when using the key variant, baked-in demo key (replace for real usage!)

## Build

From the repo root:

```bash
# Build baseline SSH image
docker build -f src/test-containers/Dockerfile.ssh -t test-ubuntu-ssh:22.04 .

# Build key-auth image
docker build -f src/test-containers/Dockerfile.ssh-key -t test-ubuntu-ssh-key:22.04 .
```

## Run

```bash
# Run baseline SSH container on port 2222
docker run -d --name test-ssh -p 2222:22 test-ubuntu-ssh:22.04

# Run key-auth container on port 2223
docker run -d --name test-ssh-key -p 2223:22 test-ubuntu-ssh-key:22.04
```

## Connect

```bash
# Password auth
ssh tester@localhost -p 2222
# password: tester123

# Key auth (key variant ships a demo key: id_rsa)
ssh -i ./src/test-containers/demo_keys/id_rsa tester@localhost -p 2223
```

## Notes

- Images switch apt sources to Aliyun mirrors for speed inside China mainland. Adjust if necessary.
- For security, replace the demo key in `demo_keys/` and change the default password.
- These images are for local testing only.
