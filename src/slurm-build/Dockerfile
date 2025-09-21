# syntax=docker/dockerfile:1.6
# Build Slurm deb packages using Ubuntu 22.04 and the local slurm source tarball
# Reference steps adapted from USTC guide: https://scc.ustc.edu.cn/hmli/doc/linux/slurm-install/slurm-install.html

FROM ubuntu:22.04 AS build
ENV DEBIAN_FRONTEND=noninteractive

# Accept optional tarball path relative to build context root
ARG SLURM_TARBALL_PATH=slurm-25.05.3.tar.bz2

# Install build prerequisites (use Aliyun Ubuntu mirrors for faster downloads in CN)
RUN sed -i 's|archive.ubuntu.com|mirrors.aliyun.com|g; s|security.ubuntu.com|mirrors.aliyun.com|g; s|ports.ubuntu.com|mirrors.aliyun.com|g' /etc/apt/sources.list \
    # Ensure universe/multiverse/restricted components are enabled for broader build-deps
    && sed -Ei 's/ jammy ([^\n]*)/ jammy \1 universe multiverse restricted/g; s/ jammy-updates ([^\n]*)/ jammy-updates \1 universe multiverse restricted/g; s/ jammy-security ([^\n]*)/ jammy-security \1 universe multiverse restricted/g' /etc/apt/sources.list \
    && apt-get update \
     && apt-get install -y --no-install-recommends \
       ca-certificates wget curl git gnupg2 \
         build-essential fakeroot devscripts equivs gdebi-core \
       libmunge-dev libmariadb-dev libpam0g-dev libcgroup-dev libhwloc-dev \
         pkg-config debhelper dh-autoreconf \
    && rm -rf /var/lib/apt/lists/*

# Add a non-root builder user to satisfy debuild (avoids building as root)
RUN useradd -m -u 1000 builder
USER builder
WORKDIR /home/builder/build

# Copy and extract the Slurm source tarball
# Note: The tarball is expected to exist at ${SLURM_TARBALL_PATH} in the build context
COPY --chown=builder:builder ${SLURM_TARBALL_PATH} /home/builder/build/

# Extract and enter source dir
RUN set -eux; \
    tar -xaf $(basename ${SLURM_TARBALL_PATH}); \
    srcdir=$(basename ${SLURM_TARBALL_PATH} .tar.bz2); \
    echo "SRC=${srcdir}" > /home/builder/build/.srcdir

# Install build-deps as root, then build .deb packages as unprivileged user
USER root
RUN set -eux; \
    srcdir=$(cut -d= -f2 /home/builder/build/.srcdir); \
    cd "/home/builder/build/${srcdir}"; \
    apt-get update; \
    mk-build-deps -i debian/control -t 'apt-get -y --no-install-recommends' --remove
USER builder
RUN set -eux; \
    srcdir=$(cut -d= -f2 /home/builder/build/.srcdir); \
    cd "/home/builder/build/${srcdir}"; \
    debuild -b -uc -us

# Collect artifacts into /out (root stage)
USER root
RUN mkdir -p /out \
    && chown -R root:root /home/builder

# Move all debuild outputs to /out (POSIX sh compatible)
RUN set -eux; \
    find /home/builder -maxdepth 1 -type f -name '*.deb' -exec mv {} /out/ \; || true; \
    find /home/builder -maxdepth 1 -type f -name '*.ddeb' -exec mv {} /out/ \; || true; \
    find /home/builder -maxdepth 1 -type f -name '*.build*' -exec mv {} /out/ \; || true; \
    find /home/builder -maxdepth 1 -type f -name '*.changes' -exec mv {} /out/ \; || true

# Default command lists produced artifacts
CMD ["bash", "-lc", "ls -alh /out && echo '\nTip: docker cp <container>:/out . to retrieve artifacts'" ]
