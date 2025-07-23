#!/bin/bash
set -e

echo "Starting Salt Master..."

# Create necessary directories if they don't exist
mkdir -p /var/cache/salt/master /var/log/salt /var/run/salt

# Set ownership
chown -R salt:salt /var/cache/salt /var/log/salt /var/run/salt /etc/salt

# Start salt-master in foreground
exec salt-master --log-level=${SALT_LOG_LEVEL:-info} --log-file=/var/log/salt/master
