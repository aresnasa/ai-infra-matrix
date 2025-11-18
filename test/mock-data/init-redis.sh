#!/bin/bash
# Mock Redis 数据初始化

redis-cli SET "test:key1" "value1"
redis-cli SET "test:key2" "value2"
redis-cli HSET "test:user:1" "name" "admin" "email" "admin@test.com"
redis-cli HSET "test:user:2" "name" "testuser1" "email" "user1@test.com"
redis-cli SADD "test:active_users" "admin" "testuser1"
redis-cli ZADD "test:scores" 100 "admin" 95 "testuser1" 88 "testuser2"

echo "Mock Redis data initialized"
