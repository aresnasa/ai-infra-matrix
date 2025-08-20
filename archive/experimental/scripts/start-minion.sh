#!/bin/bash
set -e

echo "Starting Salt Minion with ID: ${MINION_ID}"

# Wait for master to be available
echo "Waiting for master to be available..."
while ! nc -z salt-master 4506; do
    echo "Master not ready, waiting..."
    sleep 5
done

echo "Master is available, starting minion..."

# Create necessary directories
mkdir -p /var/cache/salt/minion /var/log/salt /var/run/salt

# Set ownership
chown -R salt:salt /var/cache/salt /var/log/salt /var/run/salt /etc/salt

# Update minion ID in config if environment variable is set
if [ -n "$MINION_ID" ]; then
    echo "id: $MINION_ID" > /etc/salt/minion.d/00-id.conf
fi

# Start salt-minion in foreground
exec salt-minion --log-level=${SALT_LOG_LEVEL:-info} --log-file=/var/log/salt/minion
