#!/bin/bash

# Script to initialize Nightingale admin user
# This syncs the admin user from the main system to Nightingale

set -e

echo "=== Initializing Nightingale Admin User ==="

# Get admin password from main system (hash it for Nightingale)
# Nightingale root password "root.2020" has hash: 042c05fffc2f49ca29a76223f3a41e83
# We'll use admin/admin123 (consistent with main system)
# MD5 hash of "admin123" = 0192023a7bbd73250516f069df18b500

ADMIN_PASSWORD_HASH="0192023a7bbd73250516f069df18b500"
CURRENT_TIMESTAMP=$(date +%s)

# Check if admin user exists in Nightingale
EXISTS=$(docker exec ai-infra-postgres psql -U postgres -d nightingale -t -c "SELECT COUNT(*) FROM users WHERE username='admin';")

if [ "$EXISTS" -gt 0 ]; then
    echo "✓ Admin user already exists in Nightingale, updating..."
    docker exec ai-infra-postgres psql -U postgres -d nightingale -c "
        UPDATE users 
        SET 
            password = '${ADMIN_PASSWORD_HASH}',
            nickname = 'Administrator',
            email = 'admin@example.com',
            roles = 'Admin',
            maintainer = 1,
            update_at = ${CURRENT_TIMESTAMP},
            update_by = 'system'
        WHERE username = 'admin';
    "
    echo "✓ Admin user updated successfully"
else
    echo "Creating new admin user in Nightingale..."
    docker exec ai-infra-postgres psql -U postgres -d nightingale -c "
        INSERT INTO users (
            username, nickname, password, email, roles, 
            maintainer, create_at, update_at, create_by, update_by
        ) VALUES (
            'admin', 
            'Administrator', 
            '${ADMIN_PASSWORD_HASH}', 
            'admin@example.com', 
            'Admin',
            1,
            ${CURRENT_TIMESTAMP},
            ${CURRENT_TIMESTAMP},
            'system',
            'system'
        );
    "
    echo "✓ Admin user created successfully"
fi

# Ensure admin user is in admin-group
echo "Checking admin user group membership..."

# Check if admin-group exists
GROUP_EXISTS=$(docker exec ai-infra-postgres psql -U postgres -d nightingale -t -c "SELECT COUNT(*) FROM user_group WHERE name='admin-group';")

if [ "$GROUP_EXISTS" -eq 0 ]; then
    echo "Creating admin-group..."
    # Get next available ID
    NEXT_ID=$(docker exec ai-infra-postgres psql -U postgres -d nightingale -t -c "SELECT COALESCE(MAX(id), 0) + 1 FROM user_group;" | xargs)
    docker exec ai-infra-postgres psql -U postgres -d nightingale -c "
        INSERT INTO user_group (
            id, name, note, create_at, update_at, create_by, update_by
        ) VALUES (
            ${NEXT_ID},
            'admin-group',
            'Administrators Group',
            ${CURRENT_TIMESTAMP},
            ${CURRENT_TIMESTAMP},
            'system',
            'system'
        );
    "
    echo "✓ Admin group created with ID ${NEXT_ID}"
fi

# Get admin user ID and group ID
ADMIN_USER_ID=$(docker exec ai-infra-postgres psql -U postgres -d nightingale -t -c "SELECT id FROM users WHERE username='admin';" | xargs)
ADMIN_GROUP_ID=$(docker exec ai-infra-postgres psql -U postgres -d nightingale -t -c "SELECT id FROM user_group WHERE name='admin-group';" | xargs)

echo "Admin User ID: ${ADMIN_USER_ID}, Admin Group ID: ${ADMIN_GROUP_ID}"

# Check if membership exists
MEMBER_EXISTS=$(docker exec ai-infra-postgres psql -U postgres -d nightingale -t -c "SELECT COUNT(*) FROM user_group_member WHERE user_id=${ADMIN_USER_ID} AND group_id=${ADMIN_GROUP_ID};")

if [ "$MEMBER_EXISTS" -eq 0 ]; then
    echo "Adding admin to admin-group..."
    docker exec ai-infra-postgres psql -U postgres -d nightingale -c "
        INSERT INTO user_group_member (group_id, user_id)
        VALUES (${ADMIN_GROUP_ID}, ${ADMIN_USER_ID});
    "
    echo "✓ Admin added to admin-group"
else
    echo "✓ Admin already in admin-group"
fi

# Create default business group if not exists
BUSI_GROUP_EXISTS=$(docker exec ai-infra-postgres psql -U postgres -d nightingale -t -c "SELECT COUNT(*) FROM busi_group WHERE name='Default';")

if [ "$BUSI_GROUP_EXISTS" -eq 0 ]; then
    echo "Creating default business group..."
    # Get next available ID
    NEXT_BUSI_ID=$(docker exec ai-infra-postgres psql -U postgres -d nightingale -t -c "SELECT COALESCE(MAX(id), 0) + 1 FROM busi_group;" | xargs)
    docker exec ai-infra-postgres psql -U postgres -d nightingale -c "
        INSERT INTO busi_group (
            id, name, label_enable, label_value, create_at, update_at, create_by, update_by
        ) VALUES (
            ${NEXT_BUSI_ID},
            'Default',
            0,
            '',
            ${CURRENT_TIMESTAMP},
            ${CURRENT_TIMESTAMP},
            'system',
            'system'
        );
    "
    echo "✓ Default business group created with ID ${NEXT_BUSI_ID}"
fi

# Link admin-group to business group with full permissions
BUSI_GROUP_ID=$(docker exec ai-infra-postgres psql -U postgres -d nightingale -t -c "SELECT id FROM busi_group WHERE name='Default';" | xargs)

BUSI_MEMBER_EXISTS=$(docker exec ai-infra-postgres psql -U postgres -d nightingale -t -c "SELECT COUNT(*) FROM busi_group_member WHERE busi_group_id=${BUSI_GROUP_ID} AND user_group_id=${ADMIN_GROUP_ID};")

if [ "$BUSI_MEMBER_EXISTS" -eq 0 ]; then
    echo "Linking admin-group to Default business group..."
    docker exec ai-infra-postgres psql -U postgres -d nightingale -c "
        INSERT INTO busi_group_member (busi_group_id, user_group_id, perm_flag)
        VALUES (${BUSI_GROUP_ID}, ${ADMIN_GROUP_ID}, 'rw');
    "
    echo "✓ Admin-group linked to business group with rw permissions"
else
    echo "✓ Admin-group already linked to business group"
fi

echo ""
echo "=== Nightingale Admin Initialization Complete ==="
echo "Username: admin"
echo "Password: admin123"
echo "Role: Admin"
echo "Access: http://192.168.18.114:8080/monitoring"
echo ""
