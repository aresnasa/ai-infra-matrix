# 🔑 AI Infrastructure Matrix 密码管理指南

## 📋 密码体系概览

AI Infrastructure Matrix 系统包含两类密码：

### 1. 🔐 系统服务密码 (通过脚本管理)

- **PostgreSQL 数据库**: `POSTGRES_PASSWORD`
- **Redis 缓存**: `REDIS_PASSWORD`
- **JWT Token**: `JWT_SECRET`
- **JupyterHub**: `CONFIGPROXY_AUTH_TOKEN`, `JUPYTERHUB_CRYPT_KEY`
- **MinIO 对象存储**: `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`
- **Gitea**: `GITEA_ADMIN_PASSWORD`, `GITEA_DB_PASSWD`
- **LDAP**: `LDAP_ADMIN_PASSWORD`, `LDAP_CONFIG_PASSWORD`

### 2. 👤 用户账户密码 (通过Web界面管理)

- **默认管理员**: `admin` / `admin123`
- **其他用户**: 通过Web界面创建和管理

## 🛠️ 密码生成脚本使用

### 运行密码生成脚本

```bash
# 生成新的系统服务密码
./scripts/generate-prod-passwords.sh
```

### 脚本功能

1. **自动备份**: 创建 `.env.prod.backup.YYYYMMDD_HHMMSS` 备份文件
2. **强密码生成**: 使用 OpenSSL 生成强随机密码
3. **配置更新**: 自动更新 `.env.prod` 文件中的服务密码
4. **密码显示**: 显示生成的所有密码信息

### 示例输出

```text
====================================================================
🔧 AI Infrastructure Matrix 生产环境密码生成器
====================================================================
⚠️  此脚本将生成新的系统服务密码
⚠️  默认管理员账户 (admin/admin123) 不会被此脚本修改
⚠️  请在系统部署后通过Web界面修改管理员密码
====================================================================

ℹ️  创建备份: .env.prod.backup.20250826_140230
ℹ️  生成新的强密码...
✅ 已生成并应用新的强密码

====================================================================
🔑 重要！默认管理员账户信息：

  用户名: admin
  初始密码: admin123

⚠️  请在首次登录后立即更改管理员密码！
⚠️  管理员密码未通过此脚本更改，需要在系统内修改！
====================================================================

ℹ️  系统服务密码信息:
POSTGRES_PASSWORD: x8K2mL9qR5wN3pT6vY1cA7
REDIS_PASSWORD: F4bG8nM2zX5vC9pL1kR7wQ
...
```

## 🚨 重要安全提醒

### 首次部署后必须执行

1. **立即修改管理员密码**
   - 登录地址: `http://your-domain:8080`
   - 用户名: `admin`
   - 初始密码: `admin123`
   - 登录后前往: **用户设置 → 修改密码**

2. **安全地保存密码信息**
   - 将生成的密码信息保存到安全的密码管理器
   - 删除终端输出的密码历史记录
   - 确保备份文件的安全存储

### 生产环境安全实践

1. **定期更换密码**
   ```bash
   # 定期重新生成系统服务密码
   ./scripts/generate-prod-passwords.sh
   
   # 重启服务以应用新密码
   ./build.sh prod-down
   ./build.sh prod-up aiharbor.msxf.local/aihpc v0.3.5
   ```

2. **访问控制**
   - 限制对 `.env.prod` 文件的访问权限
   - 使用防火墙限制服务端口访问
   - 定期审查用户权限

3. **备份管理**
   - 定期清理过期的密码备份文件
   - 确保备份文件加密存储

## 🔧 密码重置流程

### 系统服务密码重置

```bash
# 1. 生成新密码
./scripts/generate-prod-passwords.sh

# 2. 重启所有服务
./build.sh prod-down
./build.sh prod-up aiharbor.msxf.local/aihpc v0.3.5
```

### 管理员密码重置

如果忘记管理员密码，可以通过数据库重置：

```bash
# 连接到PostgreSQL数据库
docker exec -it ai-infra-postgres psql -U postgres -d ai-infra-matrix

# 重置admin用户密码为admin123
UPDATE users SET password = '$2a$10$example_hash_for_admin123' WHERE username = 'admin';
```

或重新运行初始化容器：

```bash
# 重新运行backend-init容器
docker run --rm --network ai-infra-network \
  --env-file .env.prod \
  aiharbor.msxf.local/aihpc/ai-infra-backend-init:v0.3.5
```

## 📞 技术支持

如果遇到密码相关问题，请检查：

1. **服务日志**: `./build.sh prod-logs [service]`
2. **环境配置**: 确认 `.env.prod` 文件格式正确
3. **网络连接**: 确认各服务间网络通信正常

---

**记住**: 安全的密码管理是系统安全的基础！
