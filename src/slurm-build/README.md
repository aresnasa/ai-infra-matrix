# Slurm DEB Builder (Ubuntu 22.04)

This container builds Slurm `.deb` packages from the provided source tarball (referencing the USTC guide).

- Base image: `ubuntu:22.04`
- Source tarball: `src/slurm-25.05.3.tar.bz2` (at repo root)
- Output: All generated artifacts will be placed in `/out` in the image

## Build

From the repository root, build the image:

```bash
# Build (uses default tarball path src/slurm-25.05.3.tar.bz2)
docker build -f src/slurm/Dockerfile -t slurm-deb:25.05.3 .

# Optionally override tarball path if you have a different version
# docker build --build-arg SLURM_TARBALL_PATH=src/slurm-<VERSION>.tar.bz2 -f src/slurm/Dockerfile -t slurm-deb:<VERSION> .
```

## Export artifacts

List artifacts by running the image:

```bash
docker run --rm slurm-deb:25.05.3
```

Copy the built packages out of the image:

```bash
# Create a throwaway container, then copy
CID=$(docker create slurm-deb:25.05.3)
docker cp "$CID":/out ./slurm-debs
docker rm "$CID"

ls -alh ./slurm-debs
```

Artifacts typically include packages like `slurm-smd`, `slurm-smd-client`, `slurm-smd-slurmctld`, etc., depending on the Slurm version.

## Notes

- The Dockerfile switches to Aliyun Ubuntu mirrors to speed up apt in China mainland; remove or adjust if undesired.
- `debuild` is executed as an unprivileged `builder` user to avoid building as root.
- During `mk-build-deps`, a temporary build-deps package is installed; the `--remove` flag cleans it afterward.
