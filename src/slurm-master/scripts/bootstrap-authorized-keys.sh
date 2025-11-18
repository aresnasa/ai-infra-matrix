#!/bin/bash
set -euo pipefail

ROOT_SSH_DIR="/root/.ssh"
DEFAULT_KEY="${ROOT_SSH_DIR}/authorized_keys.bootstrap"
TARGET_KEY="${ROOT_SSH_DIR}/authorized_keys"
SHARED_DIR="/srv/shared/ssh"

mkdir -p "${ROOT_SSH_DIR}" "${SHARED_DIR}"

if [ -s "${SHARED_DIR}/authorized_keys" ]; then
    cp "${SHARED_DIR}/authorized_keys" "${TARGET_KEY}"
elif [ -s "${SHARED_DIR}/id_rsa.pub" ]; then
    cp "${SHARED_DIR}/id_rsa.pub" "${TARGET_KEY}"
elif [ -s "${DEFAULT_KEY}" ]; then
    cp "${DEFAULT_KEY}" "${TARGET_KEY}"
fi

chmod 700 "${ROOT_SSH_DIR}"
chmod 600 "${TARGET_KEY}"
