#!/usr/bin/env bash
set -euo pipefail

# Derive IMAGE_TAG from git describe or branch, fallback to v0.0.3.2
TAG=${1:-}
if [ -z "${TAG}" ]; then
  if git describe --tags --abbrev=0 >/dev/null 2>&1; then
    TAG=$(git describe --tags --always | sed 's/^v\?\(.*\)$/v\1/')
  else
    TAG="v0.0.3.2"
  fi
fi
export IMAGE_TAG="$TAG"
echo "IMAGE_TAG=${IMAGE_TAG}"

echo "You can now run: IMAGE_TAG=${IMAGE_TAG} docker-compose build"
