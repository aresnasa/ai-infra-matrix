#!/bin/sh
# Initialize ProxyAuth user for Nightingale
# This script ensures the admin user exists before Nightingale starts serving requests

set -e

echo "Waiting for PostgreSQL to be ready..."
until PGPASSWORD="${POSTGRES_PASSWORD:-your-postgres-password}" psql -h "${POSTGRES_HOST:-postgres}" -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nightingale}" -c '\q' 2>/dev/null; do
  echo "PostgreSQL is unavailable - sleeping"
  sleep 2
done

echo "PostgreSQL is up - checking/creating ProxyAuth user"

# Use INSERT ... ON CONFLICT to handle duplicate user gracefully
# 【重要】contacts 字段必须是有效的 JSON 字符串，不能是 NULL 或空字符串
# Nightingale 使用 GORM 扫描 JSON 列，空字符串会导致 "unexpected end of JSON input" 错误
PGPASSWORD="${POSTGRES_PASSWORD:-your-postgres-password}" psql -h "${POSTGRES_HOST:-postgres}" -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nightingale}" <<-EOSQL
  -- 修复现有用户的 contacts 字段（将 NULL 或空字符串转换为空 JSON 对象）
  UPDATE users SET contacts = '{}' WHERE contacts IS NULL OR contacts = '';
  
  -- Insert admin user with ON CONFLICT clause to handle duplicates
  -- contacts 使用 '{}' 而不是 NULL，避免 JSON 解析错误
  INSERT INTO users (username, nickname, password, phone, email, portrait, roles, contacts, maintainer, create_at, create_by, update_at, update_by, belong, last_active_time)
  VALUES ('admin', 'admin', '', '', '', '', 'Admin', '{}', 0, EXTRACT(EPOCH FROM NOW())::BIGINT, 'system', EXTRACT(EPOCH FROM NOW())::BIGINT, 'system', '', 0)
  ON CONFLICT (username) DO UPDATE SET contacts = COALESCE(NULLIF(users.contacts, ''), '{}');
EOSQL

echo "ProxyAuth user initialization complete"
